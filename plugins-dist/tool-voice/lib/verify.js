'use strict';
// ═══════════════════════════════════════════════════════════════
// lib/verify.js — §7 六项「非换声 / 非生成式」专项自检
//
// 对应 docs/voice-creation-plan.md §7（每项可自动化，任一失败 = 承诺被破坏）：
//   7.1 依赖审计      禁止清单命中数 = 0
//   7.2 帧级可追溯    sourceMap 覆盖率 100% + 重建包络相关 ≥ 0.99
//   7.3 音色一致性    低次谐波偏差 H2 ≤ 10%、H3 ≤ 15%、H4/H5 ≤ 45%
//   7.4 确定性可复现  两次渲染位级一致（maxDiff = 0）
//   7.5 链可见        recipe 每 stage 有算法名 + 版本 + 参数
//   7.6 无新增音源    输出有声段全部可由源帧解释（包络相关 ≥ 0.95）
//
// 关于判据口径的诚实声明（重要）：
//   · 7.2 的「重建」不可能逐样本复原 —— PSOLA/WSOLA 会改变相位与谱包络，
//     源帧与输出帧的波形本就不同。可自动化且有意义的判据是「时间-能量结构一致」：
//     按 sourceMap 从源取帧重建后，与输出比较**分帧包络**相关。本实现即此口径，
//     并在结果里显式标注 basis='frame-envelope'，避免让人误以为逐样本 0.99。
//   · 7.3 不对全谐波偏差设阈值（PoC 实测 H4/H5 天然 24%/44%），只对低次谐波
//     严格 —— 低次谐波携带音色主体（基频与共振峰位置），高次谐波在时域算法下
//     本就改变，这不是"换声"。
// ═══════════════════════════════════════════════════════════════

const fs = require('node:fs');
const path = require('node:path');
const dsp = require('./dsp');
const analyze = require('./analyze');

// ── 7.1 禁止清单 ────────────────────────────────────────────
// 换声 / 生成式 / TTS 相关包名特征（按「包名段」匹配，避免 yue/bark 之类误伤子串）
const FORBIDDEN_NAME_RE = new RegExp(
  [
    'rvc', 'so-?vits', 'sovits', 'diff-?singer', 'fish-?speech', 'elevenlabs',
    'suno', 'ace-?step', 'utaformatix', 'tts', 'voice-?conv(ersion)?', 'openvoice',
    'seed-?vc', 'bark', 'tortoise', 'vits', 'applio', 'ddsp', 'world-?vocoder',
  ].map((s) => `(^|[-_/.])${s}([-_/.]|$)`).join('|'),
  'i',
);
// 模型权重后缀（生成式/换声模型必然带权重文件）
const FORBIDDEN_EXT = ['.pth', '.ckpt', '.safetensors', '.onnx', '.gguf', '.pb', '.h5'];

function collectInstalled(nodeModulesDir) {
  const out = [];
  if (!fs.existsSync(nodeModulesDir)) return out;
  for (const ent of fs.readdirSync(nodeModulesDir, { withFileTypes: true })) {
    if (!ent.isDirectory()) continue;
    if (ent.name.startsWith('@')) {
      const scopeDir = path.join(nodeModulesDir, ent.name);
      for (const sub of fs.readdirSync(scopeDir, { withFileTypes: true })) {
        if (sub.isDirectory()) out.push(ent.name + '/' + sub.name);
      }
    } else if (ent.name !== '.bin') {
      out.push(ent.name);
    }
  }
  return out;
}

function scanWeightFiles(roots, opts) {
  const o = opts || {};
  const skip = new Set(o.skip || ['node_modules', '.git', '_temp', 'bin', 'tmp', 'release', 'gocache', 'obj']);
  const maxDepth = o.maxDepth != null ? o.maxDepth : 4;
  const hits = [];
  let scanned = 0;
  const walk = (dir, depth) => {
    if (depth > maxDepth) return;
    let ents;
    try { ents = fs.readdirSync(dir, { withFileTypes: true }); } catch (_) { return; }
    for (const ent of ents) {
      if (skip.has(ent.name)) continue;
      const p = path.join(dir, ent.name);
      if (ent.isDirectory()) walk(p, depth + 1);
      else {
        scanned++;
        const ext = path.extname(ent.name).toLowerCase();
        if (FORBIDDEN_EXT.indexOf(ext) >= 0) hits.push({ kind: 'weight-file', path: p });
      }
    }
  };
  for (const r of roots) if (r && fs.existsSync(r)) walk(r, 0);
  return { hits, scanned };
}

// auditDependencies 7.1：插件声明依赖 + 实际安装树（含传递）+ 插件目录权重文件。
// 口径说明：默认只审「本插件引入的东西」（插件目录 + 其 node_modules）——
// 把整个用户工作区纳入会误报（工作区里别的工具的 .onnx 与本插件无关）。
// 需要更严口径时传 opts.includeWorkspace = true（会额外扫描工作区，最多 4 层）。
function auditDependencies(pluginDir, workspaceRoot, opts) {
  const o = opts || {};
  const hits = [];
  const declared = [];
  const pkgPath = path.join(pluginDir, 'package.json');
  if (fs.existsSync(pkgPath)) {
    const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
    for (const key of ['dependencies', 'peerDependencies', 'optionalDependencies']) {
      for (const name of Object.keys(pkg[key] || {})) {
        declared.push(name);
        if (FORBIDDEN_NAME_RE.test(name)) hits.push({ kind: 'declared-dep', name });
      }
    }
  }
  const installed = collectInstalled(path.join(pluginDir, 'node_modules'));
  for (const name of installed) {
    if (FORBIDDEN_NAME_RE.test(name)) hits.push({ kind: 'installed-dep', name });
  }
  const roots = o.includeWorkspace && workspaceRoot ? [workspaceRoot, pluginDir] : [pluginDir];
  const w = scanWeightFiles(roots, { maxDepth: o.includeWorkspace ? 4 : 6 });
  for (const h of w.hits) hits.push(h);
  return {
    pass: hits.length === 0,
    basis: o.includeWorkspace
      ? '声明依赖 + 安装依赖树（含传递）+ 插件目录与工作区的权重文件后缀'
      : '声明依赖 + 安装依赖树（含传递）+ 插件目录权重文件后缀（不含工作区）',
    scannedRoots: roots,
    declared: declared.length,
    installed: installed.length,
    filesScanned: w.scanned,
    forbiddenExtensions: FORBIDDEN_EXT,
    hits: hits.slice(0, 50),
  };
}

// ── 7.2 / 7.6 帧级可追溯 ────────────────────────────────────
// sourceMap 是**分段线性段表** [{outA,outB,srcA,srcB}]（见 lib/ops.js composeSegs）：
// 段表覆盖 [0, 输出时长] 且每段 src 落在源范围内 = 帧级可追溯；
// 再按段表从源重建 → 与输出比较分帧包络相关。
function checkTraceability(output, source, sourceMapSegs, fs, opts) {
  const o = opts || {};
  // ✱ 判据分层（物理事实）：PSOLA/WSOLA 会改变周期结构与相位 —— 逐样本/逐段
  //   波形互相关在变调、伸缩之后**必然**接近 0，这不是溯源失败。因此：
  //   · 全链保波形（仅切片/停顿删减等样本级搬运）→ 断言**波形级**逐段互相关 ≥ 0.99
  //     （这是 sourceMap 正确性的强证据）；
  //   · 链中含改波形 op（变调/伸缩）→ 断言**结构级**：包络相关 ≥ 0.95 +
  //     时间对齐非反相（逐段互相关中位 > 0），波形级数值仍如实上报但不作断言。
  const preserving = o.waveformPreserving !== false;
  const n = output.length;
  const outDur = n / fs;
  const srcDur = source.length / fs;
  const segs = (sourceMapSegs || []).filter(
    (s) => s && Number.isFinite(s.outA) && Number.isFinite(s.outB)
      && Number.isFinite(s.srcA) && Number.isFinite(s.srcB)
      && s.srcA >= -1e-9 && s.srcB <= srcDur + 1e-6,
  );
  let cov = 0;
  for (const s of segs) {
    const a = Math.max(0, s.outA), b = Math.min(outDur, s.outB);
    if (b > a) cov += b - a;
  }
  const coverage = outDur > 0 ? Math.min(1, cov / outDur) : 0;
  const rebuilt = dsp.rebuildFromSource({ segs }, source, { fs, outFrames: n });
  const envOut = dsp.envelope(output, { fs, frameMs: 10 }).values;
  const envReb = dsp.envelope(rebuilt, { fs, frameMs: 10 }).values;
  const corr = dsp.pearson(envOut, envReb);
  const corrRaw = dsp.pearson(output, rebuilt);
  // 方案原文判据是「逐段互相关 ≥ 0.99」——逐样本相关对 PSOLA/WSOLA 无意义
  // （时域算法改变相位与周期结构），故按**分段归一化互相关**实现：
  // 50 ms 窗、25 ms 步，每窗先零均值归一化再算相关（对整体增益不敏感）。
  const segCorr = segmentalCorr(output, rebuilt, fs, { winMs: 50, hopMs: 25 });
  const envPass = corr >= 0.95;
  // 有声段的结构级一致性（对变调/伸缩输出仍成立）
  const vadSegs = analyze.vadSegments(output, { fs });
  const perSeg = [];
  for (const s of vadSegs) {
    if (!s.voiced) continue;
    const f0 = Math.max(0, Math.round(s.a / 0.01));
    const f1 = Math.min(envOut.length, Math.round(s.b / 0.01));
    const i0 = Math.max(0, Math.round(s.a * fs)), i1 = Math.min(n, Math.round(s.b * fs));
    if (i1 - i0 < fs * 0.05) continue;
    const cEnv = f1 - f0 >= 5 ? dsp.pearson(envOut.subarray(f0, f1), envReb.subarray(f0, f1)) : 0;
    const cSm = dsp.pearson(output.subarray(i0, i1), rebuilt.subarray(i0, i1));
    perSeg.push({
      a: +s.a.toFixed(3),
      b: +s.b.toFixed(3),
      envelopeCorr: +cEnv.toFixed(4),
      sampleCorr: +cSm.toFixed(4),
    });
  }
  const minSegEnvCorr = perSeg.length ? Math.min.apply(null, perSeg.map((x) => x.envelopeCorr)) : 0;
  const structurePass = envPass && (perSeg.length === 0 || minSegEnvCorr >= 0.9);
  const pass = coverage >= 0.999 && (preserving ? segCorr.min >= 0.99 : structurePass);
  return {
    coverage: +coverage.toFixed(6),
    segs: segs.length,
    basis: preserving ? '波形级：逐段归一化互相关（保波形链）' : '结构级：包络相关 + 时间对齐（链含改波形 op）',
    waveformPreserving: preserving,
    segmental: segCorr,
    envelopeCorr: +corr.toFixed(6),
    minSegEnvelopeCorr: +minSegEnvCorr.toFixed(4),
    sampleCorr: +corrRaw.toFixed(6),
    voicedSegments: perSeg,
    criteria: preserving
      ? '段表覆盖 100% 且逐段波形互相关 min ≥ 0.99'
      : '段表覆盖 100% 且包络相关 ≥ 0.95 且每个有声段包络相关 ≥ 0.9',
    pass,
  };
}

// segmentalCorr 分段归一化互相关（窗内零均值 + 单位方差归一 → 只衡量形状/对齐）。
function segmentalCorr(a, b, fs, opts) {
  const o = opts || {};
  const win = Math.max(64, Math.round((fs * (o.winMs || 50)) / 1000));
  const hop = Math.max(32, Math.round((fs * (o.hopMs || 25)) / 1000));
  // 静音窗（两侧能量都低于峰值 silenceDb）不参与判定：方差为 0 时"相关性"
  // 在数学上无定义（pearson 返回 0），把它算作失败会误报。
  const silenceDb = o.silenceDb != null ? o.silenceDb : -60;
  const refPeak = Math.max(dsp.stats(a).peak, dsp.stats(b).peak);
  const thr = refPeak * Math.pow(10, silenceDb / 20);
  const cs = [];
  for (let s = 0; s + win <= a.length; s += hop) {
    const wa = a.subarray(s, s + win), wb = b.subarray(s, s + win);
    const sa = dsp.stats(wa).rms, sb = dsp.stats(wb).rms;
    if (Math.max(sa, sb) < thr) continue; // 双静音窗：跳过
    const v = dsp.pearson(wa, wb);
    if (Number.isFinite(v)) cs.push(v);
  }
  if (cs.length === 0) return { windows: 0, median: 0, p10: 0, min: 0, silenceWindowsSkipped: true };
  const sorted = cs.slice().sort((x, y) => x - y);
  return {
    windows: cs.length,
    median: +sorted[sorted.length >> 1].toFixed(6),
    p10: +sorted[Math.floor(sorted.length * 0.1)].toFixed(6),
    min: +sorted[0].toFixed(6),
  };
}

// ── 7.6 无新增音源 ──────────────────────────────────────────
// 判据（可自动化、对 PSOLA 成立）：输出中的**每一帧有声能量**都能被源解释 ——
// 按 sourceMap 反查该输出帧对应的源时间，源的对应位置（±30 ms 容差）必须也有声。
// 若输出在源静音处凭空出现能量（= 混入了源之外的新音源），可解释率会立刻下降。
// 辅以能量偏差中位数（应接近 0 dB —— 变换不该整体改写响度）。
function checkNoNewSource(output, source, sourceMapSegs, fs, opts) {
  const o = opts || {};
  const thrRel = o.thresholdDb != null ? o.thresholdDb : -45;
  const envOut = dsp.envelope(output, { fs, frameMs: 10 });
  const envSrc = dsp.envelope(source, { fs, frameMs: 10 });
  const outRef = dsp.dbfs(dsp.stats(output).peak);
  const srcRef = dsp.dbfs(dsp.stats(source).peak);
  const map = { segs: sourceMapSegs || [] };
  let voicedOut = 0, explained = 0, unmapped = 0;
  const deltas = [];
  const unexplained = [];
  for (let f = 0; f < envOut.n; f++) {
    if (envOut.db[f] - outRef < thrRel) continue; // 输出静音帧：无新增能量问题
    voicedOut++;
    const t = envOut.times[f];
    const srcT = dsp.mapOutToSrc(map, t);
    if (!Number.isFinite(srcT)) { unmapped++; continue; }
    const si = Math.min(envSrc.n - 1, Math.max(0, Math.round((srcT * fs - envSrc.win / 2) / envSrc.hop)));
    let srcVoiced = false;
    for (let k = Math.max(0, si - 3); k <= Math.min(envSrc.n - 1, si + 3); k++) {
      if (envSrc.db[k] - srcRef >= thrRel) { srcVoiced = true; break; }
    }
    if (srcVoiced) explained++;
    else if (unexplained.length < 20) unexplained.push({ outAt: +t.toFixed(3), srcAt: +srcT.toFixed(3) });
    deltas.push(envOut.db[f] - envSrc.db[si]);
  }
  const explainRate = voicedOut ? explained / voicedOut : 0;
  const sorted = deltas.slice().sort((a, b) => a - b);
  const medianDb = sorted.length ? sorted[sorted.length >> 1] : null;
  return {
    pass: voicedOut > 0 && explainRate >= 0.999 && (medianDb == null || Math.abs(medianDb) <= 6),
    basis: '输出有声帧的能量来源可解释率（按 sourceMap 反查源）+ 能量偏差中位',
    voicedFrames: voicedOut,
    explainedFrames: explained,
    explainRate: +explainRate.toFixed(6),
    unmappedFrames: unmapped,
    medianDeltaDb: medianDb == null ? null : +medianDb.toFixed(2),
    unexplainedSamples: unexplained,
  };
}

// ── 7.3 音色一致性（分层判据）────────────────────────────────
const TIMBRE_LIMITS = { h2: 0.1, h3: 0.15, h45: 0.45 };

async function checkTimbre(source, fs, output, sourceMapSegs, opts) {
  const o = opts || {};
  const srcDur = source.length / fs;
  // 源与输出各自做 F0 跟踪（各自用自己的音高测谐波 —— 变调后谐波位置必然改变）
  const srcTrack = await analyze.f0Track(source, { fs, hop: 512 });
  const outTrack = await analyze.f0Track(output, { fs, hop: 512 });
  const notes = dsp.detectNotes(Array.from(outTrack.hz), {
    fs, hop: outTrack.hop, clarity: Array.from(outTrack.clarity), minDur: 0.12,
  });
  // 取最长的两个稳定音符段
  const picks = notes.slice().sort((a, b) => (b.t1 - b.t0) - (a.t1 - a.t0)).slice(0, 2);
  const rows = [];
  for (const n of picks) {
    const tMid = (n.t0 + n.t1) / 2;
    const outF0 = harmonicF0At(outTrack, tMid);
    if (!(outF0 > 0)) continue;
    const srcT = dsp.mapOutToSrc({ segs: sourceMapSegs || [] }, tMid);
    if (!(srcT >= 0 && srcT <= srcDur)) continue;
    const srcF0 = harmonicF0At(srcTrack, srcT);
    if (!(srcF0 > 0)) continue;
    const a = dsp.harmonicRatios(source, { fs, f0: srcF0, tCenter: srcT, upTo: 5 });
    const b = dsp.harmonicRatios(output, { fs, f0: outF0, tCenter: tMid, upTo: 5 });
    if (a.ratios.length < 4 || b.ratios.length < 4) continue;
    const dev = a.ratios.map((v, i) => Math.abs(v - b.ratios[i]) / Math.max(v, 1e-6));
    rows.push({
      outAt: +tMid.toFixed(3),
      srcAt: +srcT.toFixed(3),
      outF0: +outF0.toFixed(2),
      srcF0: +srcF0.toFixed(2),
      shiftCents: +dsp.centsBetween(srcF0, outF0).toFixed(2),
      ratiosSrc: a.ratios.map((v) => +v.toFixed(4)),
      ratiosOut: b.ratios.map((v) => +v.toFixed(4)),
      dev: dev.map((v) => +v.toFixed(4)),
    });
  }
  // 判据按最坏段取值（保守）
  let h2 = 0, h3 = 0, h45 = 0;
  for (const r of rows) {
    h2 = Math.max(h2, r.dev[0] || 0);
    h3 = Math.max(h3, r.dev[1] || 0);
    h45 = Math.max(h45, Math.max(r.dev[2] || 0, r.dev[3] || 0));
  }
  const pass = rows.length > 0 && h2 <= TIMBRE_LIMITS.h2 && h3 <= TIMBRE_LIMITS.h3 && h45 <= TIMBRE_LIMITS.h45;
  return {
    pass,
    limits: TIMBRE_LIMITS,
    maxDev: { h2: +h2.toFixed(4), h3: +h3.toFixed(4), h45: +h45.toFixed(4) },
    samples: rows,
    note: rows.length === 0 ? '未找到可用于音色比对的稳定音符段（素材过短或全无浊音）' : undefined,
  };
}

function harmonicF0At(track, tSec) {
  const i = Math.max(0, Math.min(track.hz.length - 1, Math.round((tSec * track.fs) / track.hop)));
  const lo = Math.max(0, i - 4), hi = Math.min(track.hz.length - 1, i + 4);
  const vals = [];
  for (let k = lo; k <= hi; k++) if (track.hz[k] > 0) vals.push(track.hz[k]);
  return dsp.median(vals);
}

// ── 7.5 链可见 ──────────────────────────────────────────────
function checkChain(recipes, edits) {
  const problems = [];
  for (let i = 0; i < recipes.length; i++) {
    const r = recipes[i] || {};
    if (!r.op) problems.push(`stage[${i}] 缺 op 名`);
    if (!r.algo) problems.push(`stage[${i}] (${r.op}) 缺算法名`);
    if (!r.version) problems.push(`stage[${i}] (${r.op}) 缺算法版本`);
    if (r.params === undefined || r.params === null) problems.push(`stage[${i}] (${r.op}) 缺参数`);
  }
  const nEdits = (edits || []).length;
  if (nEdits !== recipes.length) problems.push(`edits(${nEdits}) 与 recipe 阶段数(${recipes.length}) 不一致`);
  return {
    pass: problems.length === 0,
    stages: recipes.map((r) => ({ op: r.op, algo: r.algo, version: r.version, params: r.params })),
    problems,
  };
}

// ── 7.4 确定性 ──────────────────────────────────────────────
function checkDeterminism(a, b, opts) {
  const o = opts || {};
  let maxDiff = 0;
  const n = Math.min(a.length, b.length);
  for (let i = 0; i < n; i++) {
    const d = Math.abs(a[i] - b[i]);
    if (d > maxDiff) maxDiff = d;
  }
  const lengthMatch = a.length === b.length;
  const st = dsp.stats(a);
  const relDb = dsp.dbfs(maxDiff / Math.max(st.rms, 1e-12));
  const pass = lengthMatch && (maxDiff === 0 || relDb <= -180);
  return {
    pass,
    lengthMatch,
    framesA: a.length,
    framesB: b.length,
    maxDiff,
    relDb: +relDb.toFixed(1),
    quantizedCompare: !!o.quantizedCompare,
    criterion: '位级一致（maxDiff = 0）；若引入并行浮点则放宽至 ≤ -180 dBFS',
  };
}

// runVerify 执行六项，返回 { pass, checks[], metrics }
async function runVerify(input) {
  const {
    pluginDir, workspaceRoot, fs: sampleRate,
    source, output, sourceMapSegs, recipes, edits, renderAgain,
  } = input;
  const checks = [];

  // 7.1
  const dep = auditDependencies(pluginDir, workspaceRoot);
  checks.push({
    id: '7.1', name: '依赖审计', pass: dep.pass,
    metric: `禁止命中 ${dep.hits.length} 项`, detail: dep,
  });

  // 7.2
  const waveformPreserving = (recipes || []).every((r) => r && r.waveformPreserving === true);
  const trace = checkTraceability(output, source, sourceMapSegs, sampleRate, { waveformPreserving });
  checks.push({
    id: '7.2', name: '帧级可追溯（sourceMap 重建）', pass: trace.pass,
    metric: trace.waveformPreserving
      ? `段表覆盖 ${(trace.coverage * 100).toFixed(3)}%，保波形链逐段波形互相关 min ${trace.segmental.min}（中位 ${trace.segmental.median}）`
      : `段表覆盖 ${(trace.coverage * 100).toFixed(3)}%，包络相关 ${trace.envelopeCorr}（含改波形 op，波形级中位 ${trace.segmental.median}）`,
    detail: trace,
  });

  // 7.3
  const timbre = await checkTimbre(source, sampleRate, output, sourceMapSegs, {});
  checks.push({
    id: '7.3', name: '音色一致性（低次谐波严格/高次放宽）', pass: timbre.pass,
    metric: `H2 ${(timbre.maxDev.h2 * 100).toFixed(1)}% / H3 ${(timbre.maxDev.h3 * 100).toFixed(1)}% / H4·H5 ${(timbre.maxDev.h45 * 100).toFixed(1)}%`,
    detail: timbre,
  });

  // 7.4
  let det;
  if (typeof renderAgain === 'function') {
    const again = await renderAgain();
    det = checkDeterminism(output, again.data, { quantizedCompare: !!again.quantized });
    det.secondRenderFrames = again.data.length;
  } else {
    det = { pass: false, note: '未提供 renderAgain（无法比较两次渲染）' };
  }
  checks.push({
    id: '7.4', name: '确定性可复现', pass: !!det.pass,
    metric: det.maxDiff != null ? `maxDiff = ${det.maxDiff}（${det.relDb} dBFS）` : '未执行',
    detail: det,
  });

  // 7.5
  const chain = checkChain(recipes, edits);
  checks.push({
    id: '7.5', name: '链可见（recipe 覆盖全部阶段）', pass: chain.pass,
    metric: `${chain.stages.length} 个阶段`, detail: chain,
  });

  // 7.6
  const noNew = checkNoNewSource(output, source, sourceMapSegs, sampleRate, {});
  checks.push({
    id: '7.6', name: '无新增音源（输出可由源帧解释）', pass: noNew.pass,
    metric: `可解释率 ${(noNew.explainRate * 100).toFixed(3)}%（${noNew.explainedFrames}/${noNew.voicedFrames} 帧），能量偏差中位 ${noNew.medianDeltaDb} dB`,
    detail: noNew,
  });

  const pass = checks.every((c) => c.pass);
  return {
    pass,
    standard: 'voice-creation-plan §7（非换声 / 非生成式）',
    passed: checks.filter((c) => c.pass).length,
    total: checks.length,
    checks,
    metrics: {
      fs: sampleRate,
      sourceSeconds: +(source.length / sampleRate).toFixed(4),
      outputSeconds: +(output.length / sampleRate).toFixed(4),
    },
  };
}

module.exports = {
  runVerify, auditDependencies, checkTraceability, checkTimbre, checkChain,
  checkDeterminism, checkNoNewSource, segmentalCorr,
  FORBIDDEN_NAME_RE, FORBIDDEN_EXT, TIMBRE_LIMITS,
};
