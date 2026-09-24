'use strict';
// ═══════════════════════════════════════════════════════════════
// lib/ops.js — 编辑操作（op）实现 + 帧级 sourceMap 累积
//
// 设计铁律（对齐 voice-creation-plan.md §2.3 / §4）：
//   ① 输出 = 源 + edits 的**纯函数**（同输入同 seed → 位级一致）；
//   ② 每个 op 必须给出「输出帧 → 输入帧时间」的 indexMap，最终合成一张
//      **输出帧 → 源帧时间** 的 sourceMap（7.2 帧级追溯与 7.6 无新增音源的证据）；
//   ③ 每个 op 记录 recipe（算法名 + 包版本 + 参数）—— 7.5 链可见；
//   ④ 参数越界即报错（不静默降级）：方案 §8 的质量上限由调用方决定，
//      但"沉默地把 ±12 半音硬做出来"是不允许的。
// ═══════════════════════════════════════════════════════════════

const dsp = require('./dsp');
const { loadDeps } = require('./deps');

// tune-snap 支持的音阶（见其 README 参数表）
const SCALES = [
  'chromatic', 'major', 'minor', 'harmonic-minor', 'melodic-minor',
  'pentatonic-major', 'pentatonic-minor', 'blues', 'dorian', 'mixolydian', 'whole',
];

// qualityLimits 质量上限（方案 §8）——供上层提示，不自动放宽。
const QUALITY_LIMITS = {
  transposeSemitonesWarn: 5,   // ±5 半音可接受
  transposeSemitonesMax: 12,   // >±12 不可用
  stretchFactorWarn: 1.5,      // 说话人声 >1.5× 出现颤动/金属感
  stretchFactorRange: [0.5, 2.0],
};

function numRange(v, def, lo, hi, label) {
  const x = v == null || v === '' ? def : Number(v);
  if (!Number.isFinite(x)) throw new Error(`${label} 不是有效数字: ${v}`);
  if (x < lo || x > hi) throw new Error(`${label} 超出允许范围 [${lo}, ${hi}]: ${x}`);
  return x;
}

function intRange(v, def, lo, hi, label) {
  const x = Math.round(numRange(v, def, lo, hi, label));
  return x;
}

// fullSegs 恒等映射的单段表示（输出 0..dur ↔ 输入 0..dur）。
function fullSegs(outFrames, inFrames, fs) {
  return [{ outA: 0, outB: outFrames / fs, srcA: 0, srcB: inFrames / fs }];
}

// composeSegs 分段线性映射合成：inner（新→旧）∘ outer（旧→源）= 新→源。
// 关键：按 outer 的段边界切分 inner 段 —— 分段线性函数的复合仍是分段线性，
// 只在边界对齐时精确；不对齐会出现"段内折线被拉直"的溯源误差。
// 好处（相对逐帧表）：10 分钟素材的映射也只需几十个段，且可 diff。
function composeSegs(inner, outer) {
  const out = [];
  const mapOutToSrc = (t) => {
    for (const s of outer) {
      if (t >= s.outA - 1e-12 && t <= s.outB + 1e-12) {
        const span = s.outB - s.outA;
        if (span <= 0) return s.srcA;
        return s.srcA + ((t - s.outA) / span) * (s.srcB - s.srcA);
      }
    }
    if (outer.length) {
      const first = outer[0], last = outer[outer.length - 1];
      if (t < first.outA) return first.srcA;
      if (t > last.outB) return last.srcB;
    }
    return NaN;
  };
  for (const s of inner) {
    const span = s.srcB - s.srcA;
    const bounds = [s.srcA, s.srcB];
    for (const o of outer) {
      if (o.outA > s.srcA && o.outA < s.srcB) bounds.push(o.outA);
      if (o.outB > s.srcA && o.outB < s.srcB) bounds.push(o.outB);
    }
    bounds.sort((a, b) => a - b);
    for (let i = 0; i + 1 < bounds.length; i++) {
      const a = bounds[i], b = bounds[i + 1];
      if (b - a <= 0) continue;
      const newA = span <= 0 ? s.outA : s.outA + ((a - s.srcA) / span) * (s.outB - s.outA);
      const newB = span <= 0 ? s.outB : s.outA + ((b - s.srcA) / span) * (s.outB - s.outA);
      const srcA = mapOutToSrc(a), srcB = mapOutToSrc(b);
      if (!Number.isFinite(srcA) || !Number.isFinite(srcB)) continue;
      out.push({ outA: newA, outB: newB, srcA, srcB });
    }
  }
  return out;
}

// expandSegs 把段表展开为逐帧源时间表（调试/可视化用；验证走段表重建更省内存）。
function expandSegs(segs, outFrames, fs) {
  const m = new Float64Array(outFrames);
  for (let i = 0; i < outFrames; i++) {
    const t = i / fs;
    let v = NaN;
    for (const s of segs) {
      if (t >= s.outA && t <= s.outB) {
        const span = s.outB - s.outA;
        v = span <= 0 ? s.srcA : s.srcA + ((t - s.outA) / span) * (s.srcB - s.srcA);
        break;
      }
    }
    if (!Number.isFinite(v) && segs.length) v = t < segs[0].outA ? segs[0].srcA : segs[segs.length - 1].srcB;
    m[i] = v;
  }
  return m;
}

// applyOp 应用单个 op。返回 { data, map, recipe, autos }
//   data   : Float32Array（输出样本）
//   map    : Float64Array（长度 = data.length，值 = 该输出帧对应的**输入**时间，秒）
//   recipe : { op, algo, version, params, autos? }
async function applyOp(data, fs, op) {
  const name = op && op.op;
  if (!name) throw new Error('op 缺少 op 字段');
  const d = await loadDeps();

  // ── pitch.snap：音阶吸附（Auto-Tune 类；PSOLA 改激励，不改音色）──
  if (name === 'pitch.snap') {
    const scale = op.scale || 'chromatic';
    if (SCALES.indexOf(scale) < 0) throw new Error(`pitch.snap 不支持 scale=${scale}（可选：${SCALES.join('/')}）`);
    const root = intRange(op.root, 0, 0, 11, 'pitch.snap.root');
    const strength = numRange(op.strength, 1, 0, 1, 'pitch.snap.strength');
    const // 默认容差 12¢：PoC 证明 YIN 存在系统性偏低（-5 ¢ 量级），
      // 容差低于该偏差会把「本来就准的音」当跑调去修（方案 §2.2 实测要点）。
      tolerance = numRange(op.tolerance, 12, 0, 100, 'pitch.snap.tolerance');
    const fade = numRange(op.fade, 0.01, 0, 0.2, 'pitch.snap.fade');
    const out = d.snap(data, { fs, scale, root, strength, tolerance, fade });
    const n = Math.min(out.length, data.length);
    const res = new Float32Array(n);
    for (let i = 0; i < n; i++) res[i] = out[i];
    return {
      data: res,
      segs: fullSegs(n, data.length, fs),
      recipe: {
        op: 'pitch.snap',
        algo: '@audio/tune-snap',
        version: d.versions['tune-snap'],
        waveformPreserving: false, // PSOLA 重调改变周期结构 → 波形级互相关不适用
        engine: 'YIN F0 → 段中位数 → 音阶吸附 → PSOLA 重调 + 干声交叉淡化',
        params: { scale, root, strength, tolerance, fade },
      },
    };
  }

  // ── time.warp：整体时值/语速伸缩（WSOLA，时域无 FFT）──
  if (name === 'time.warp') {
    const factor = numRange(op.factor, 1, QUALITY_LIMITS.stretchFactorRange[0], QUALITY_LIMITS.stretchFactorRange[1], 'time.warp.factor');
    const frameSize = intRange(op.frameSize, 2048, 256, 8192, 'time.warp.frameSize');
    const out = d.wsola(data, { factor, frameSize });
    const n = out.length;
    const res = new Float32Array(n);
    for (let i = 0; i < n; i++) res[i] = out[i];
    const autos = [];
    if (factor > QUALITY_LIMITS.stretchFactorWarn) {
      autos.push(`时值伸缩 ${factor}× > ${QUALITY_LIMITS.stretchFactorWarn}×：说话人声会出现颤动/金属感（方案 §8.2），已在 recipe 记录，未自动降级`);
    }
    return {
      data: res,
      segs: fullSegs(n, data.length, fs),
      autos,
      recipe: {
        op: 'time.warp',
        algo: '@audio/stretch-wsola',
        version: d.versions['stretch-wsola'],
        waveformPreserving: false, // WSOLA 重叠相加改变相位
        engine: 'WSOLA（波形相似度叠加，时域，无 FFT）',
        params: { factor, frameSize },
      },
    };
  }

  // ── speech.removeSilence：删除停顿（保留呼吸边距）──
  if (name === 'speech.removeSilence') {
    const minGap = numRange(op.minGap, 0.35, 0.05, 5, 'speech.removeSilence.minGap');
    const keep = numRange(op.keep, 0.08, 0, 1, 'speech.removeSilence.keep');
    const thresholdDb = numRange(op.thresholdDb, -40, -80, -10, 'speech.removeSilence.thresholdDb');
    const det = dsp.detectSilence(data, { fs, minGap, keep, thresholdDb });
    const total = data.length / fs;
    const cuts = [];
    let cur = 0;
    for (const s of det.spans) {
      const a = Math.max(cur, Math.min(s.a, total));
      const b = Math.max(a, Math.min(s.b, total));
      if (a > cur) cuts.push([cur, a]);
      cur = Math.max(cur, b);
    }
    if (cur < total) cuts.push([cur, total]);
    const frames = cuts.reduce((acc, seg) => acc + Math.max(0, Math.round((seg[1] - seg[0]) * fs)), 0);
    const out = new Float32Array(frames);
    const segs = [];
    let w = 0;
    for (const [a, b] of cuts) {
      const i0 = Math.max(0, Math.round(a * fs));
      const i1 = Math.min(data.length, Math.round(b * fs));
      if (i1 <= i0) continue;
      const w0 = w;
      for (let i = i0; i < i1; i++) { out[w] = data[i]; w++; }
      segs.push({ outA: w0 / fs, outB: w / fs, srcA: i0 / fs, srcB: i1 / fs });
    }
    return {
      data: out.subarray(0, w),
      segs,
      recipe: {
        op: 'speech.removeSilence',
        algo: 'builtin/energy-gate',
        version: '1.0.0',
        waveformPreserving: true, // 样本级搬运：输出每帧都是源的原始样本
        engine: '帧 RMS 包络（10 ms）→ 相对峰值门限 → 最小间隔判定 → 保留边距拼接',
        params: { minGap, keep, thresholdDb },
      },
      autos: det.spans.length === 0 ? [`未检出满足条件的停顿（minGap=${minGap}s、门限 ${thresholdDb} dB）——素材未改动`] : [],
      detail: { removed: det.spans.map((s) => ({ a: +s.a.toFixed(4), b: +s.b.toFixed(4), dur: +(s.b - s.a).toFixed(4) })) },
    };
  }

  throw new Error(`未知 op：${name}（当前支持：pitch.snap / time.warp / speech.removeSilence）`);
}

// applyOps 顺序应用 ops，并把各步 indexMap 合成为「输出帧 → 源帧时间」表。
// 返回 { data, sourceMap: Float64Array, recipes: [...], autos: [...], details: [...] }
async function applyOps(mono, fs, ops) {
  let cur = mono;
  let acc = fullSegs(mono.length, mono.length, fs); // 当前时间 → 源时间
  const recipes = [];
  const autos = [];
  const details = [];
  for (let k = 0; k < (ops || []).length; k++) {
    const op = ops[k];
    let r;
    try {
      r = await applyOp(cur, fs, op);
    } catch (e) {
      const err = new Error(`op[${k}] ${op && op.op} 失败：${e && e.message ? e.message : e}`);
      err.opIndex = k;
      throw err;
    }
    // 合成映射：新时间 →（r.segs）→ cur 时间 →（acc）→ 源时间
    const next = composeSegs(r.segs, acc);
    cur = r.data;
    acc = next;
    recipes.push(r.recipe);
    if (r.autos && r.autos.length) autos.push(...r.autos);
    details.push({ index: k, op: r.recipe.op, frames: r.data.length, detail: r.detail || null });
  }
  return { data: cur, segs: acc, recipes, autos, details };
}

module.exports = { applyOp, applyOps, SCALES, QUALITY_LIMITS, fullSegs, composeSegs, expandSegs };
