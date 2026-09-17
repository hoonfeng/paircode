'use strict';
// ═══════════════════════════════════════════════════════════════
// lib/analyze.js — 分析段（f0 / notes / vad / slices / loudness）
//
// 全部按「源素材的客观测量」定义：不产生新音频，只产出可 diff 的 JSON。
// 三条 PoC 实测约束在此落地：
//   · YIN 系统性偏差 ≈ -5 ¢ → 返回 cents 时如实保留，供吸附环节加容差；
//   · 谱通量 ODF 过检 → slices 走 dsp.refineSlices（能量门限 + 最小间隔 + F0 连续性）；
//   · 音色判据在 verify 层分层（低次严、高次宽）。
// ═══════════════════════════════════════════════════════════════

const { loadDeps } = require('./deps');
const dsp = require('./dsp');

// f0Track 逐帧 F0（YIN）。返回 { hop, frameSize, frames, hz: Float64Array, clarity: Float64Array }
async function f0Track(mono, opts) {
  const o = opts || {};
  const fs = o.fs || 48000;
  const hop = Math.max(32, o.hop || 512);
  const frameSize = Math.max(512, o.frameSize || 2048);
  const clarityMin = o.clarityMin != null ? o.clarityMin : 0.6;
  const minFreq = o.minFreq || 70;
  const maxFreq = o.maxFreq || 1200;
  const d = await loadDeps();
  const n = Math.max(1, Math.floor(Math.max(0, mono.length - frameSize) / hop) + 1);
  const hz = new Float64Array(n);
  const clarity = new Float64Array(n);
  let voiced = 0;
  for (let i = 0; i < n; i++) {
    const s = i * hop;
    const win = mono.subarray(s, Math.min(mono.length, s + frameSize));
    let r = null;
    try {
      r = d.yin(win, { fs, minFreq, maxFreq, threshold: o.yinThreshold != null ? o.yinThreshold : 0.15 });
    } catch (_) { r = null; }
    if (r && r.clarity >= clarityMin && r.freq > 0) {
      hz[i] = r.freq;
      clarity[i] = r.clarity;
      voiced++;
    } else {
      hz[i] = NaN;
      clarity[i] = r && Number.isFinite(r.clarity) ? r.clarity : 0;
    }
  }
  return { hop, frameSize, frames: n, fs, hz, clarity, voiced, voicedRatio: n ? voiced / n : 0 };
}

// vad 有声/无声分段（能量门限）。
function vadSegments(mono, opts) {
  const o = opts || {};
  const fs = o.fs || 48000;
  const env = o.env || dsp.envelope(mono, { fs, frameMs: o.frameMs || 10 });
  const refDb = dsp.dbfs(dsp.stats(mono).peak);
  const thr = refDb + (o.thresholdDb != null ? o.thresholdDb : -45);
  const spans = [];
  let cur = null;
  for (let f = 0; f < env.n; f++) {
    const voiced = env.db[f] >= thr;
    if (!cur || cur.voiced !== voiced) {
      if (cur) { cur.b = env.times[f]; spans.push(cur); }
      cur = { a: env.times[f], b: env.times[f], voiced };
    }
  }
  if (cur) { cur.b = env.times[env.n - 1]; spans.push(cur); }
  return spans.map((s) => ({ a: +s.a.toFixed(4), b: +s.b.toFixed(4), voiced: s.voiced }));
}

// slices 切片（onset 候选 + 三条后处理）。
async function slices(mono, opts) {
  const o = opts || {};
  const fs = o.fs || 48000;
  const d = await loadDeps();
  const env = o.env || dsp.envelope(mono, { fs, frameMs: 10 });
  const refDb = dsp.dbfs(dsp.stats(mono).peak);
  const track = o.f0track || null;
  // 四源候选（PoC 结论：单一谱通量 ODF 会漏掉段起点、又会过报颤音处的伪峰，
  // 因此不把任何单一检测器当真相源）
  const src = { spectral: [], energy: [], vad: [], notes: [] };
  try {
    const flux = d.spectralFlux(mono, { fs, frameSize: o.frameSize || 2048, hopSize: o.hopSize || 512 });
    const picked = d.peakPick(flux.odf, { hopSize: flux.hopSize, fs: flux.fs, delta: o.delta != null ? o.delta : 1.4 });
    src.spectral = Array.from(picked);
  } catch (e) {
    src.spectral = [];
  }
  try {
    const ef = d.energyFlux(mono, { fs, frameSize: o.frameSize || 2048, hopSize: o.hopSize || 512 });
    src.energy = Array.from(d.peakPick(ef.odf, { hopSize: ef.hopSize, fs: ef.fs, delta: o.delta != null ? o.delta : 1.4 }));
  } catch (e) {
    src.energy = [];
  }
  // VAD 有声段起点（静音 → 有声 必为真实起音）
  const vad = o.vad || vadSegments(mono, { fs, env, thresholdDb: o.thresholdDb });
  src.vad = vad.filter((s) => s.voiced).map((s) => s.a);
  // F0 音符段起点（音高跳变处 = 换音起音）
  if (track) {
    src.notes = dsp
      .detectNotes(Array.from(track.hz), { fs, hop: track.hop, clarity: Array.from(track.clarity) })
      .map((n) => n.t0);
  }
  // 聚类合并（邻近候选归一组，取组内能量最大者）
  const candidates = [].concat(src.spectral, src.energy, src.vad, src.notes).filter((x) => Number.isFinite(x));
  const merged = dsp.clusterCandidates(candidates, { gapMs: o.clusterGapMs != null ? o.clusterGapMs : 60, env, fs });
  const kept = dsp.refineSlices(merged, {
    fs,
    minGap: o.minGap != null ? o.minGap : 0.06,
    env,
    refDb,
    envThrDb: o.envThrDb != null ? o.envThrDb : -45,
    f0: track ? Array.from(track.hz) : null,
    hop: track ? track.hop : 512,
    f0Min: o.f0Min != null ? o.f0Min : 1,
    jumpCents: o.sliceJumpCents != null ? o.sliceJumpCents : 60,
    riseDb: o.sliceRiseDb != null ? o.sliceRiseDb : 6,
  });
  // 尾端：按 VAD 有声段边界截断（去掉切片尾随静音），但不越过下一个切片起点
  const total = mono.length / fs;
  const tailEnd = (t) => {
    for (const s of vad) if (s.voiced && t >= s.a - 0.02 && t <= s.b + 0.02) return s.b;
    for (const s of vad) if (s.voiced && s.a > t) return s.b;
    return total;
  };
  return {
    candidates: candidates.length,
    sources: { spectral: src.spectral.length, energy: src.energy.length, vad: src.vad.length, notes: src.notes.length },
    merged: merged.length,
    slices: kept.map((t, i, arr) => {
      const next = i + 1 < arr.length ? arr[i + 1] : total;
      return { index: i, a: +t.toFixed(4), b: +Math.min(next, tailEnd(t), total).toFixed(4) };
    }),
  };
}

// loudness BS.1770-4 计量。
async function loudness(mono, opts) {
  const o = opts || {};
  const fs = o.fs || 48000;
  const d = await loadDeps();
  const out = { standard: 'BS.1770-4' };
  try { out.lufs = round3(d.lufs([mono], { fs })); } catch (e) { out.lufsError = String(e && e.message || e); }
  try { out.truePeak = round3(d.truepeak([mono], { fs })); } catch (e) { out.truePeakError = String(e && e.message || e); }
  try { out.lra = round3(d.lra([mono], { fs })); } catch (e) { out.lra = null; }
  return out;
}

function round3(v) {
  return typeof v === 'number' && Number.isFinite(v) ? +v.toFixed(3) : v;
}

// analyzeAll 组合分析（按 tasks 选择）。
async function analyzeAll(mono, opts) {
  const o = opts || {};
  const fs = o.fs || 48000;
  const tasks = o.tasks && o.tasks.length ? o.tasks : ['f0', 'notes', 'vad', 'slices', 'loudness'];
  const want = (t) => tasks.indexOf(t) >= 0;
  const env = dsp.envelope(mono, { fs, frameMs: 10 });
  const out = { fs, tasks };
  let track = null;
  if (want('f0') || want('notes') || want('slices')) {
    track = await f0Track(mono, { ...o, fs });
    out.track = track;
    if (want('f0')) {
      out.f0 = {
        hop: track.hop,
        frameSize: track.frameSize,
        frames: track.frames,
        voicedRatio: +track.voicedRatio.toFixed(4),
        medianHz: round3(dsp.median(Array.from(track.hz).filter((v) => v > 0))),
      };
    }
  }
  if (want('notes') && track) {
    const notes = dsp.detectNotes(Array.from(track.hz), {
      fs,
      hop: track.hop,
      clarity: Array.from(track.clarity),
      jumpCents: o.jumpCents != null ? o.jumpCents : 80,
      minDur: o.minDur != null ? o.minDur : 0.05,
    });
    out.notes = notes.map((n) => ({
      t0: +n.t0.toFixed(4),
      t1: +n.t1.toFixed(4),
      f0: round3(n.f0),
      midiFloat: round3(69 + 12 * Math.log2(n.f0 / 440)),
      frames: n.frames,
      conf: n.conf != null ? +n.conf.toFixed(3) : null,
    }));
  }
  if (want('vad')) out.vad = vadSegments(mono, { fs, env });
  if (want('slices')) {
    const s = await slices(mono, { ...o, fs, env, f0track: track });
    out.sliceCandidates = s.candidates;
    out.slices = s.slices;
  }
  if (want('loudness')) out.loudness = await loudness(mono, { fs });
  return out;
}

module.exports = { analyzeAll, f0Track, vadSegments, slices, loudness };
