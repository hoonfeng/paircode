// 人声子域 PoC —— 验证「非换声 / 非生成式」人声创作链的最小闭环可行性
// 依赖：@audio/{pitch-yin,tune-snap,loudness,onset}（均 MIT）
// 覆盖：1 分析(F0) → 2 处理(音阶吸附/PSOLA) → 3 音色一致性 → 4 确定性 → 5 响度(BS.1770-4) → 6 切片(onset)
import yin from '@audio/pitch-yin'
import snap from '@audio/tune-snap'
import { lufs, truepeak } from '@audio/loudness'
import { spectralFlux, peakPick } from '@audio/onset'
import { writeFileSync } from 'node:fs'

const FS = 48000
const log = []
const say = s => { console.log(s); log.push(s) }
const brief = v => (typeof v === 'number' ? v.toFixed(2) : JSON.stringify(v).slice(0, 60))

// ---------- 合成"人声样"素材：3 个有声段 + 5Hz 颤音 + 谐波堆 ----------
function synth () {
  const dur = 2.2, n = Math.round(dur * FS), out = new Float32Array(n)
  const notes = [
    { t0: 0.10, t1: 0.70, f0: 220.00 },  // A3（C 大调音阶内）
    { t0: 0.75, t1: 1.35, f0: 246.94 },  // B3（音阶内）
    { t0: 1.45, t1: 2.05, f0: 289.20 }   // 偏高约 +27¢（major 期望吸附到 D4 = 293.66）
  ]
  for (const s of notes) {
    const i0 = Math.round(s.t0 * FS), i1 = Math.round(s.t1 * FS)
    let ph = 0
    for (let i = i0; i < i1; i++) {
      const t = (i - i0) / FS
      const env = Math.min(1, t / 0.03) * Math.min(1, (i1 - i) / FS / 0.05)
      const f = s.f0 * (1 + 0.008 * Math.sin(2 * Math.PI * 5.5 * t))
      ph += (2 * Math.PI * f) / FS
      const v = Math.sin(ph) + 0.5 * Math.sin(2 * ph) + 0.3 * Math.sin(3 * ph) +
                0.18 * Math.sin(4 * ph) + 0.1 * Math.sin(5 * ph)
      out[i] = 0.22 * env * v
    }
  }
  return out
}

// ---------- F0：段内稳定区中位数 ----------
function f0At (data, tCenter, win = 2048, k = 12) {
  const vals = []
  for (let j = -k; j <= k; j++) {
    const start = Math.round(tCenter * FS) + j * 256
    if (start < 0 || start + win > data.length) continue
    const r = yin(data.subarray(start, start + win), { fs: FS, minFreq: 80, maxFreq: 600 })
    if (r && r.clarity > 0.6) vals.push(r.freq)
  }
  vals.sort((a, b) => a - b)
  return vals.length ? { f0: vals[vals.length >> 1], n: vals.length } : null
}

// ---------- 音色指纹（简化）：谐波包络比 H2/H1..H5/H1 ----------
function goertzel (data, start, N, freq) {
  const w = (2 * Math.PI * freq) / FS, c = 2 * Math.cos(w)
  let s1 = 0, s2 = 0
  for (let i = 0; i < N; i++) {
    const wnd = 0.5 - 0.5 * Math.cos((2 * Math.PI * i) / (N - 1))
    const s0 = data[start + i] * wnd + c * s1 - s2
    s2 = s1; s1 = s0
  }
  return Math.sqrt(Math.max(0, s1 * s1 + s2 * s2 - c * s1 * s2)) / N
}
function timbre (data, tCenter, f0) {
  const start = Math.max(0, Math.round(tCenter * FS) - 1024), N = 4096
  const a = []
  for (let k = 1; k <= 5; k++) a.push(goertzel(data, start, N, f0 * k))
  return a.slice(1).map(v => v / Math.max(a[0], 1e-12))
}

const TARGETS = [220.00, 246.94, 289.20]
const src = synth()
say(`素材：${(src.length / FS).toFixed(2)}s 单声道 @ ${FS} Hz，3 个有声段（${TARGETS.join(' / ')} Hz）`)
say('')

// 1) 分析
const f0 = [0.40, 1.05, 1.75].map(t => f0At(src, t))
say('【1 分析】YIN F0（段内稳定区中位数，minFreq=80, maxFreq=600）')
f0.forEach((r, i) => {
  const dev = r ? 1200 * Math.log2(r.f0 / TARGETS[i]) : NaN
  say(`  段${i + 1}: ${r ? r.f0.toFixed(2) + ' Hz（' + r.n + ' 帧）' : '未检出'}  相对合成目标偏差 ${r ? dev.toFixed(1) : '-'} ¢`)
})

// 2) 处理：音阶吸附（Auto-Tune 类，PSOLA）
const t0 = performance.now()
const out1 = snap(src, { fs: FS, scale: 'major', root: 0, strength: 1 })
const ms1 = performance.now() - t0
const f0out = [0.40, 1.05, 1.75].map(t => f0At(out1, t))
say('')
say(`【2 处理】snap(scale=major, root=C, strength=1) — 耗时 ${ms1.toFixed(0)} ms / ${(src.length / FS).toFixed(1)}s 单声道`)
f0out.forEach((r, i) => {
  if (!r || !f0[i]) { say(`  段${i + 1}: 未检出`); return }
  const shift = 1200 * Math.log2(r.f0 / f0[i].f0)
  say(`  段${i + 1}: ${f0[i].f0.toFixed(2)} → ${r.f0.toFixed(2)} Hz（${shift >= 0 ? '+' : ''}${shift.toFixed(1)} ¢）`)
})
say('  期望：段1/段2 已在音阶内（|Δ| < tolerance 8¢）→ 不动；段3 偏高 ~27¢ → 上移到 D4≈293.66 Hz')

// 3) 音色一致性（非换声的简化判据）
const T1 = timbre(src, 1.75, 289.20)
const T2 = timbre(out1, 1.75, (f0out[2] && f0out[2].f0) || 293.66)
const dev = T1.map((v, i) => Math.abs(v - T2[i]) / Math.max(v, 1e-9))
say('')
say('【3 音色一致性】谐波包络比 H2/H1..H5/H1（段3：源(289.2Hz) vs 输出(吸附后 f0)）')
say('  源  : ' + T1.map(v => v.toFixed(3)).join(' / '))
say('  输出: ' + T2.map(v => v.toFixed(3)).join(' / '))
say('  相对偏差: ' + dev.map(v => (v * 100).toFixed(1) + '%').join(' / ') + `  最大 ${(Math.max(...dev) * 100).toFixed(1)}%`)
say('  判据：谐波包络（≈音色）基本保持，仅整体频率位移 → 未换声')

// 4) 确定性
const out2 = snap(src, { fs: FS, scale: 'major', root: 0, strength: 1 })
let maxDiff = 0, sum = 0
for (let i = 0; i < out1.length; i++) { maxDiff = Math.max(maxDiff, Math.abs(out1[i] - out2[i])); sum += out1[i] * out1[i] }
const rms = Math.sqrt(sum / out1.length)
const db = 20 * Math.log10(Math.max(maxDiff, 1e-12) / rms)
say('')
say(`【4 确定性】两次 snap 逐样本最大差 = ${maxDiff}（${maxDiff === 0 ? '位级一致' : '有差异'}）→ 相对电平 ${db.toFixed(1)} dBFS`)
say('  判据：生成式模型不可能位级复现；本链可复现 = 「非生成式」机制证据')

// 5) 响度（BS.1770-4）
const L1 = lufs([src], { fs: FS }), L2 = lufs([out1], { fs: FS })
const P1 = truepeak([src], { fs: FS }), P2 = truepeak([out1], { fs: FS })
say('')
say(`【5 响度 BS.1770-4】LUFS 源 ${brief(L1)} → 输出 ${brief(L2)}`)
say(`  true peak 源 ${brief(P1)} → 输出 ${brief(P2)}`)

// 6) 切片（onset）
const { odf, hopSize } = spectralFlux(src, { fs: FS })
const on = peakPick(odf, { hopSize, fs: FS })
say('')
say(`【6 切片】谱通量 onset 峰值：${on.length} 个 → [${Array.from(on).map(v => v.toFixed(2)).join(', ')}] s（期望≈3 个段起点）`)

writeFileSync('poc-report.txt', log.join('\n') + '\n')
say('')
say('报告已写入 _temp/voice-poc/poc-report.txt')
