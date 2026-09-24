'use strict';
// ═══════════════════════════════════════════════════════════════
// lib/project.js — 工程文件（voice.project.json）读写与渲染纯函数
//
// 工程结构（对齐 voice-creation-plan.md §4，L1 为 V-P0 子集）：
//   { version, createdAt, sources[], analysis{}, edits[], render{}, provenance{}, checks{} }
//
// 关键不变量：
//   · analysis 与 edits 分离 —— 重分析不丢编辑；
//   · edits 即命令模型 —— UI 与 Agent 写同一种 op；
//   · 渲染 = 源 + edits 的**纯函数**（同源同 edits → 位级一致）；
//   · provenance 记录 sourceMap 段表与 recipe —— 「非换声 / 非生成式」的机制证据。
// ═══════════════════════════════════════════════════════════════

const nodeFs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const wav = require('./wav');
const dsp = require('./dsp');
const opsLib = require('./ops');

function sha256(buf) {
  return crypto.createHash('sha256').update(buf).digest('hex');
}

function isAbsPath(p) {
  const s = String(p == null ? '' : p);
  return /^[a-zA-Z]:[\\/]/.test(s) || s.startsWith('/') || s.startsWith('\\');
}

// resolveFrom 以 base 为基准解析（base 为工作区根；绝对路径原样）。
function resolveFrom(base, p) {
  const s = String(p == null ? '' : p);
  return isAbsPath(s) ? s : path.resolve(base || process.cwd(), s);
}

function emptyProject(name) {
  return {
    version: 1,
    name: name || 'voice',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    sources: [],
    analysis: {},
    edits: [],
    render: {},
    provenance: {},
    checks: {},
  };
}

function loadProject(absPath) {
  if (!nodeFs.existsSync(absPath)) return null;
  const raw = nodeFs.readFileSync(absPath, 'utf8');
  const proj = JSON.parse(raw);
  if (!Array.isArray(proj.sources)) proj.sources = [];
  if (!Array.isArray(proj.edits)) proj.edits = [];
  if (!proj.analysis) proj.analysis = {};
  return proj;
}

function saveProject(absPath, proj) {
  proj.updatedAt = new Date().toISOString();
  nodeFs.mkdirSync(path.dirname(absPath), { recursive: true });
  nodeFs.writeFileSync(absPath, JSON.stringify(proj, null, 2));
  return absPath;
}

// readSource 读 WAV（保持源采样率；多声道下混为单声道）。
function readSource(absPath) {
  const buf = nodeFs.readFileSync(absPath);
  const w = wav.parseWav(buf);
  const mono = wav.toMono(w.data);
  const st = dsp.stats(mono);
  return {
    buf,
    parsed: w,
    mono,
    fs: w.fs,
    stat: {
      sha256: sha256(buf),
      bytes: buf.length,
      frames: w.frames,
      channels: w.channels,
      bitDepth: w.bitDepth,
      format: w.format,
      fs: w.fs,
      seconds: +(w.frames / w.fs).toFixed(4),
      peak: +st.peak.toFixed(6),
      rms: +st.rms.toFixed(6),
      dc: +st.dc.toFixed(6),
    },
  };
}

// registerSource 把源登记进工程（同路径则更新，保留原 id）。
function registerSource(proj, relPath, stat) {
  let src = proj.sources.find((s) => s.path === relPath);
  if (!src) {
    const id = 'v' + (proj.sources.length + 1);
    src = { id, path: relPath };
    proj.sources.push(src);
  }
  Object.assign(src, stat);
  return src;
}

// renderProject 源 + edits → 输出（纯函数，不触碰磁盘）。
async function renderProject(proj, srcAbs) {
  const src = readSource(srcAbs);
  const r = await opsLib.applyOps(src.mono, src.fs, proj.edits || []);
  return { data: r.data, segs: r.segs, recipes: r.recipes, autos: r.autos, details: r.details, src };
}

// sourceMapSummary 段表的可 diff 摘要（进工程 JSON；完整段表随渲染结果旁挂）。
function sourceMapSummary(segs, fs, outFrames) {
  if (!segs || segs.length === 0) return { segs: 0 };
  const s = segs.map((x) => ({
    sourceSeconds: +(x.srcB - x.srcA).toFixed(4),
    outputSeconds: +(x.outB - x.outA).toFixed(4),
  }));
  const removed = +(outFrames / fs - s.reduce((a, x) => a + x.outputSeconds, 0)).toFixed(6);
  return {
    segs: segs.length,
    outputSeconds: +(outFrames / fs).toFixed(4),
    mappedOutputSeconds: +s.reduce((a, x) => a + x.outputSeconds, 0).toFixed(4),
    mappedSourceSeconds: +s.reduce((a, x) => a + x.sourceSeconds, 0).toFixed(4),
    unmappedSeconds: removed,
  };
}

module.exports = {
  sha256, isAbsPath, resolveFrom, emptyProject, loadProject, saveProject,
  readSource, registerSource, renderProject, sourceMapSummary,
};
