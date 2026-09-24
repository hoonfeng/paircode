'use strict';
// ═══════════════════════════════════════════════════════════════
// lib/fixture.js — 确定性「人声样」测试素材合成
//
// 用途：为 voice_verify 与回归提供**位级可复现**的输入（生成式模型无法复现
//       ⇒ 用确定性素材才能证明"我们的链是确定性的"）。
//
// 与 PoC（docs/research/voice-poc/poc.mjs）同源但增强：
//   · 三个有声段（含段间静音）——覆盖切片/停顿删减/音阶吸附；
//   · 5.5 Hz 颤音（人声自然颤音区间 4–7 Hz）；
//   · 谐波堆 + **共振峰包络**（/a/ 型：F1≈700·F2≈1220·F3≈2600 Hz）——
//     使"音色一致性"判据（低次谐波包络比）具有真实意义；
//   · 第 3 段故意偏高 ≈ +27 ¢ —— 音阶吸附的对照段。
//
// 全程无随机数（确定性噪声用固定种子 LCG，默认关闭）。
// ═══════════════════════════════════════════════════════════════

const F1 = 700, F2 = 1220, F3 = 2600; // /a/ 型共振峰（Hz）
const F1BW = 130, F2BW = 180, F3BW = 260;

// formantGain 谐波 k 处（频率 f = k·f0）的共振峰包络增益（0..1 归一前）。
function formantGain(freq) {
  const g = (F, BW) => Math.exp(-((freq - F) * (freq - F)) / (2 * BW * BW));
  return 0.65 * g(F1, F1BW) + 0.45 * g(F2, F2BW) + 0.25 * g(F3, F3BW);
}

// 谐波幅度（基频权重 1 + 共振峰加权 + 1/k 衰减）。
function harmonicAmps(f0, k) {
  const amps = [];
  for (let h = 1; h <= k; h++) {
    const base = 1 / Math.pow(h, 1.15);
    amps.push(base * (0.45 + formantGain(h * f0)));
  }
  // 归一使基波为 1
  const a0 = amps[0] || 1;
  return amps.map((v) => v / a0);
}

// synthVoice 合成确定性人声样素材。
// opts: { fs=48000, segments?, vibratoHz=5.5, vibratoCents=14, harmonics=14, amp=0.22, noise=0 }
// 返回 { fs, data, segments, targetHz }
function synthVoice(opts) {
  const o = opts || {};
  const fs = Math.round(o.fs || 48000);
  const segments = o.segments || [
    { t0: 0.10, t1: 0.70, f0: 220.0 },   // A3（自然小调内）
    { t0: 0.75, t1: 1.35, f0: 246.94 },  // B3
    { t0: 1.45, t1: 2.05, f0: 289.2 },   // 偏高 ≈ +27¢（对照：吸附应上移到 D4≈293.66）
  ];
  const dur = o.dur || Math.max.apply(null, segments.map((s) => s.t1)) + 0.15;
  const n = Math.round(dur * fs);
  const data = new Float32Array(n);
  const vibHz = o.vibratoHz != null ? o.vibratoHz : 5.5;
  const vibCents = o.vibratoCents != null ? o.vibratoCents : 14;
  const harmonics = o.harmonics || 14;
  const amp = o.amp != null ? o.amp : 0.22;
  // 确定性噪声（固定种子 LCG；noise=0 时完全不使用）
  let seed = 12345;
  const rnd = () => { seed = (1103515245 * seed + 12345) & 0x7fffffff; return seed / 0x7fffffff - 0.5; };
  const noise = o.noise || 0;

  for (const s of segments) {
    const i0 = Math.round(s.t0 * fs), i1 = Math.min(n, Math.round(s.t1 * fs));
    const amps = harmonicAmps(s.f0, harmonics);
    const ph = new Float64Array(harmonics);
    for (let i = i0; i < i1; i++) {
      const t = (i - i0) / fs;
      // 30 ms 起振 + 50 ms 收尾（自然包络）
      const env = Math.min(1, t / 0.03) * Math.min(1, (i1 - i) / fs / 0.05);
      const f0 = s.f0 * Math.pow(2, (vibCents * Math.sin(2 * Math.PI * vibHz * t)) / 1200);
      let v = 0;
      for (let h = 0; h < harmonics; h++) {
        ph[h] += (2 * Math.PI * f0 * (h + 1)) / fs;
        // 高次谐波 1:1 相位（相位锁定）——更接近真实声源（脉冲串式激励）
        v += amps[h] * Math.sin(ph[h]);
      }
      data[i] = amp * env * v + (noise ? noise * rnd() * env : 0);
    }
  }
  return { fs, data, segments, targetHz: segments.map((s) => s.f0), dur };
}

module.exports = { synthVoice, formantGain, harmonicAmps };
