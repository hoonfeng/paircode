'use strict';
// ═══════════════════════════════════════════════════════════════
// lib/dsp.js — 纯算法层（同步、零依赖）
//
// 只放不依赖 @audio/* 的计算：包络/静音、切片后处理、音符分段、谐波包络、
// 互相关、时间映射。需要 YIN/PSOLA/WSOLA 的部分在 lib/deps.js 注入调用。
//
// 三条来自 PoC 实测的硬约束（见 docs/voice-creation-plan.md §2.2 / §7.0）：
//   ① 谱通量 ODF 对含颤音的稳态音会过检（实测 3 段素材检出 18 个 onset）
//      → 切片必须后处理：最小间隔合并 ∩ 能量门限 ∩ F0 连续性；
//   ② 音色一致性必须分层（低次谐波严、高次谐波宽），全谱偏差会误报；
//   ③ YIN 有约 -5 ¢ 的系统性偏差 → 吸附 tolerance 默认必须 > 8 ¢（用 12 ¢）。
// ═══════════════════════════════════════════════════════════════

// ── 基础统计 ────────────────────────────────────────────────
function stats(data) {
  let peak = 0, sum = 0, dc = 0;
  for (let i = 0; i < data.length; i++) {
    const v = data[i];
    const a = v < 0 ? -v : v;
    if (a > peak) peak = a;
    sum += v * v;
    dc += v;
  }
  const n = Math.max(1, data.length);
  return { peak, rms: Math.sqrt(sum / n), dc: dc / n, frames: data.length };
}

function dbfs(x) {
  return 20 * Math.log10(Math.max(Math.abs(x), 1e-12));
}

function centsBetween(f1, f2) {
  if (!(f1 > 0) || !(f2 > 0)) return NaN;
  return 1200 * Math.log2(f2 / f1);
}

// ── 帧包络（RMS，dB）────────────────────────────────────────
// 返回 { hop, values: Float64Array(线性 RMS), db: Float64Array, times: Float64Array }
function envelope(data, opts) {
  const o = opts || {};
  const fs = o.fs || 48000;
  const frameMs = o.frameMs || 10;
  const win = Math.max(16, Math.round((fs * frameMs) / 1000));
  const hop = o.hop || win;
  const n = Math.max(1, Math.floor((data.length - win) / hop) + 1);
  const values = new Float64Array(n);
  const db = new Float64Array(n);
  const times = new Float64Array(n);
  for (let f = 0; f < n; f++) {
    const s = f * hop;
    let sum = 0;
    for (let i = s; i < s + win && i < data.length; i++) sum += data[i] * data[i];
    const rms = Math.sqrt(sum / win);
    values[f] = rms;
    db[f] = dbfs(rms);
    times[f] = (s + win / 2) / fs;
  }
  return { hop, win, fs, values, db, times, n };
}

// ── 静音检测（删除停顿用）──────────────────────────────────
// 判据：帧 dB < thresholdDb（相对峰值）持续 ≥ minGap；保留 keep 秒作为呼吸边距。
// 返回 { spans: [{a, b}], frames: envelope }
function detectSilence(data, opts) {
  const o = opts || {};
  const fs = o.fs || 48000;
  const env = o.env || envelope(data, { fs, frameMs: o.frameMs || 10 });
  const rel = o.thresholdDb != null ? o.thresholdDb : -40;
  const absolute = o.absolute != null ? o.absolute : false;
  const minGap = o.minGap != null ? o.minGap : 0.35;
  const keep = o.keep != null ? o.keep : 0.08;
  const ref = absolute ? 0 : dbfs(o.peak != null ? o.peak : stats(data).peak);
  const thr = ref + rel;
  const spans = [];
  let start = -1;
  for (let f = 0; f < env.n; f++) {
    const quiet = env.db[f] < thr;
    if (quiet && start < 0) start = f;
    if ((!quiet || f === env.n - 1) && start >= 0) {
      const end = quiet ? f : f - 1;
      const a = (start * env.hop) / fs + env.win / (2 * fs);
      const b = ((end + 1) * env.hop) / fs + env.win / (2 * fs);
      if (b - a >= minGap) spans.push({ a: Math.max(0, a + keep), b: Math.min(b - keep, data.length / fs) });
      else if (start === 0 && b - a > 0.02 && quiet) {
        // 开头长静音（<minGap 但位于起点）也保留处理机会
        spans.push({ a: 0, b: Math.max(0, b - keep) });
      }
      start = -1;
    }
  }
  // 首尾静音（若跨度不足 minGap 但确实存在）
  if (spans.length === 0) {
    const head = env.db.findIndex((v) => v >= thr);
    if (head > 0 && env.times[head] > 0.05) spans.push({ a: 0, b: Math.max(0, env.times[head] - keep) });
    let tail = -1;
    for (let f = env.n - 1; f >= 0; f--) if (env.db[f] >= thr) { tail = f; break; }
    if (tail >= 0 && tail < env.n - 1) {
      const t = env.times[env.n - 1];
      if (t - env.times[tail] > 0.05) spans.push({ a: env.times[tail] + keep, b: t });
    }
  }
  return { spans: spans.filter((s) => s.b > s.a).sort((x, y) => x.a - y.a), frames: env, threshold: thr };
}

// ── F0 序列 → 音符段 ────────────────────────────────────────
// f0: Float64Array（未 voiced = NaN/0）；hop/sr 用于时间换算。
// 分段：相邻帧音高跳变 > jumpCents 即切分；短于 minDur 的段并入邻段。
function detectNotes(f0, opts) {
  const o = opts || {};
  const fs = o.fs || 48000;
  const hop = o.hop || 256;
  const clarity = o.clarity || null;
  const clarityMin = o.clarityMin != null ? o.clarityMin : 0.6;
  const jumpCents = o.jumpCents != null ? o.jumpCents : 80;
  const minDur = o.minDur != null ? o.minDur : 0.05;
  const t = (i) => (i * hop) / fs;
  const voiced = [];
  for (let i = 0; i < f0.length; i++) {
    const ok = f0[i] > 0 && (!clarity || clarity[i] >= clarityMin);
    voiced.push(ok);
  }
  const segs = [];
  let cur = null;
  for (let i = 0; i < f0.length; i++) {
    if (!voiced[i]) {
      if (cur) { segs.push(cur); cur = null; }
      continue;
    }
    if (!cur) { cur = { i0: i, i1: i, vals: [f0[i]], conf: clarity ? [clarity[i]] : [] }; continue; }
    const med = median(cur.vals);
    if (Math.abs(centsBetween(med, f0[i])) > jumpCents) {
      segs.push(cur);
      cur = { i0: i, i1: i, vals: [f0[i]], conf: clarity ? [clarity[i]] : [] };
    } else {
      cur.i1 = i;
      cur.vals.push(f0[i]);
      if (clarity) cur.conf.push(clarity[i]);
    }
  }
  if (cur) segs.push(cur);

  const out = [];
  for (const s of segs) {
    const dur = (s.i1 - s.i0 + 1) * (hop / fs);
    if (dur < minDur) {
      // 太短：并入前一段（尾音/毛刺）
      const prev = out[out.length - 1];
      if (prev) prev.t1 = t(s.i1 + 1);
      continue;
    }
    const med = median(s.vals);
    out.push({
      t0: t(s.i0),
      t1: t(s.i1 + 1),
      f0: med,
      frames: s.vals.length,
      conf: s.conf.length ? s.conf.reduce((a, b) => a + b, 0) / s.conf.length : null,
    });
  }
  return out;
}

function median(arr) {
  if (!arr || arr.length === 0) return NaN;
  const a = Array.from(arr).sort((x, y) => x - y);
  const m = a.length >> 1;
  return a.length % 2 ? a[m] : (a[m - 1] + a[m]) / 2;
}

// ── 切片：onset 候选 → 后处理 ───────────────────────────────
// PoC 实测：谱通量 ODF 对含颤音的稳态音会过检（3 段素材检出 18 个候选）。
// 因此候选必须通过「起音事件」判据才算真切片 —— 三者之一：
//   ① F0 由无到有（清音/静音 → 浊音）：真实音符/音节起点；
//   ② F0 相对前窗跳变 ≥ jumpCents（默认 60¢）：换音起点；
//   ③ 能量相对前窗上升 ≥ riseDb（默认 6 dB）：同音高的新起音（如连续 "ka-ka"）。
// 稳态音的颤音调制（±14¢、<1 dB）三条全不满足 → 被滤除。
// 再叠加能量门限（排除静音区）与最小间隔合并（默认 80 ms）。
// 返回秒数数组（升序）。
function refineSlices(onsetTimes, opts) {
  const o = opts || {};
  const fs = o.fs || 48000;
  const minGap = o.minGap != null ? o.minGap : 0.08;
  const env = o.env || null;
  const envThrDb = o.envThrDb != null ? o.envThrDb : -45;
  const refDb = o.refDb != null ? o.refDb : 0;
  const f0 = o.f0 || null;
  const hop = o.hop || 512;
  const f0Min = o.f0Min != null ? o.f0Min : 1;
  const jumpCents = o.jumpCents != null ? o.jumpCents : 60;
  const riseDb = o.riseDb != null ? o.riseDb : 6;
  const lookback = o.lookbackMs != null ? o.lookbackMs / 1000 : 0.03;
  const eventCheck = o.eventCheck != null ? o.eventCheck : true;

  const envDbAt = (t) => {
    if (!env) return null;
    const idx = Math.min(env.n - 1, Math.max(0, Math.round((t * fs - env.win / 2) / env.hop)));
    return env.db[idx];
  };
  const f0Win = (t, from, to) => {
    if (!f0) return [];
    const i0 = Math.round(((t + from) * fs) / hop);
    const i1 = Math.round(((t + to) * fs) / hop);
    const out = [];
    for (let k = Math.max(0, i0); k <= Math.min(f0.length - 1, i1); k++) out.push(f0[k]);
    return out;
  };

  const cand = Array.from(onsetTimes).filter((x) => Number.isFinite(x)).sort((a, b) => a - b);
  const kept = [];
  for (const t of cand) {
    if (t < 0) continue;
    const curDb = envDbAt(t);
    // ① 能量门限：起音不能落在静音区
    if (env && curDb != null && curDb < refDb + envThrDb) continue;
    // ② 起音事件判据（需 F0 与包络同时可用；缺失时退化为只做门限+间隔）
    if (env && f0 && eventCheck) {
      const prevVoiced = f0Win(t, -lookback, -0.005).filter((v) => v > f0Min);
      const hereVoiced = f0Win(t, 0, lookback).filter((v) => v > f0Min);
      if (hereVoiced.length === 0) continue;
      let isEvent = prevVoiced.length === 0;
      if (!isEvent) {
        const jump = Math.abs(centsBetween(median(prevVoiced), median(hereVoiced)));
        if (jump >= jumpCents) isEvent = true;
      }
      if (!isEvent) {
        const prevDb = envDbAt(t - lookback);
        if (prevDb != null && curDb != null && curDb - prevDb >= riseDb) isEvent = true;
      }
      if (!isEvent) continue;
    }
    // ③ 最小间隔合并（保留更早者 —— 确定性、可复现）
    if (kept.length && t - kept[kept.length - 1] < minGap) continue;
    kept.push(t);
  }
  return kept;
}

// ── 多源候选聚类 ────────────────────────────────────────────
// 作用：切片候选来自四个源（谱通量 / 能量通量 / VAD 有声段起点 / F0 音符段起点），
// 它们在时间上会彼此邻近（同一真实起音被多个源各报一次）。聚类把「间隔 ≤ gapMs」
// 的候选归为一组，组内取**能量最大**者作为代表（起音瞬间能量最集中），
// 回退最早者 —— 结果确定性、与源顺序无关。
function clusterCandidates(times, opts) {
  const o = opts || {};
  const gapMs = o.gapMs != null ? o.gapMs : 60;
  const env = o.env || null;
  const fs = o.fs || 48000;
  const list = Array.from(times || []).filter((x) => Number.isFinite(x) && x >= 0).sort((a, b) => a - b);
  const groups = [];
  for (const t of list) {
    const g = groups[groups.length - 1];
    if (g && (t - g[g.length - 1]) * 1000 <= gapMs) g.push(t);
    else groups.push([t]);
  }
  const dbAt = (t) => {
    if (!env) return 0;
    const idx = Math.min(env.n - 1, Math.max(0, Math.round((t * fs - env.win / 2) / env.hop)));
    return env.db[idx];
  };
  return groups
    .map((g) => {
      let best = g[0], bestDb = -Infinity;
      for (const t of g) {
        const d = dbAt(t);
        if (d > bestDb) { bestDb = d; best = t; }
      }
      return best;
    })
    .sort((a, b) => a - b);
}

// ── 谐波包络（Goertzel，PoC 同法）────────────────────────────
// 返回 { ratios: [H2/H1..Hk/H1], amps: [a1..ak] }
function goertzel(data, start, N, fs, freq) {
  const w = (2 * Math.PI * freq) / fs, c = 2 * Math.cos(w);
  let s1 = 0, s2 = 0;
  for (let i = 0; i < N; i++) {
    const idx = start + i;
    if (idx < 0 || idx >= data.length) break;
    const wnd = 0.5 - 0.5 * Math.cos((2 * Math.PI * i) / (N - 1));
    const s0 = data[idx] * wnd + c * s1 - s2;
    s2 = s1; s1 = s0;
  }
  return Math.sqrt(Math.max(0, s1 * s1 + s2 * s2 - c * s1 * s2)) / N;
}

function harmonicRatios(data, opts) {
  const o = opts || {};
  const fs = o.fs || 48000;
  const f0 = o.f0;
  if (!(f0 > 0)) return { ratios: [], amps: [] };
  const N = o.N || 4096;
  const start = Math.max(0, Math.round((o.tCenter || 0) * fs - N / 2));
  const k = o.upTo || 5;
  const amps = [];
  for (let h = 1; h <= k; h++) amps.push(goertzel(data, start, N, fs, f0 * h));
  const a1 = Math.max(amps[0], 1e-12);
  return { ratios: amps.slice(1).map((v) => v / a1), amps };
}

// ── 相关性 ──────────────────────────────────────────────────
function pearson(a, b) {
  const n = Math.min(a.length, b.length);
  if (n === 0) return 0;
  let ma = 0, mb = 0;
  for (let i = 0; i < n; i++) { ma += a[i]; mb += b[i]; }
  ma /= n; mb /= n;
  let num = 0, da = 0, db = 0;
  for (let i = 0; i < n; i++) {
    const x = a[i] - ma, y = b[i] - mb;
    num += x * y; da += x * x; db += y * y;
  }
  if (da <= 0 || db <= 0) return 0;
  return num / Math.sqrt(da * db);
}

// ── 时间映射（sourceMap）────────────────────────────────────
// 段：{ outA, outB, srcA, srcB }（秒，线性对应）。查询/重建用。
function makeTimeMap() {
  return { segs: [] };
}
function pushTimeMap(map, seg) {
  map.segs.push(seg);
  return map;
}
function mapOutToSrc(map, tOut) {
  for (const s of map.segs) {
    if (tOut >= s.outA && tOut <= s.outB) {
      const span = s.outB - s.outA;
      if (span <= 0) return s.srcA;
      const k = (tOut - s.outA) / span;
      return s.srcA + k * (s.srcB - s.srcA);
    }
  }
  return NaN;
}
// 重建：按映射从源抽帧，输出到 out（长度为 outFrames）。
function rebuildFromSource(map, src, opts) {
  const o = opts || {};
  const fs = o.fs || 48000;
  const outFrames = o.outFrames || 0;
  const out = new Float32Array(outFrames);
  for (let i = 0; i < outFrames; i++) {
    const tOut = i / fs;
    const tSrc = mapOutToSrc(map, tOut);
    if (!Number.isFinite(tSrc)) continue;
    const p = tSrc * fs;
    const i0 = Math.floor(p);
    if (i0 < 0 || i0 + 1 >= src.length) {
      if (i0 >= 0 && i0 < src.length) out[i] = src[i0];
      continue;
    }
    const frac = p - i0;
    out[i] = src[i0] * (1 - frac) + src[i0 + 1] * frac;
  }
  return out;
}

module.exports = {
  stats, dbfs, centsBetween, envelope, detectSilence, detectNotes, median,
  refineSlices, clusterCandidates, goertzel, harmonicRatios, pearson,
  makeTimeMap, pushTimeMap, mapOutToSrc, rebuildFromSource,
};
