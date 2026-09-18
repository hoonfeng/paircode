'use strict';
// ═══════════════════════════════════════════════════════════════
// tool-voice — 人声创作工具面（说话 / 歌声）· 纯 DSP
//
// 运行时轨：**Node 桥**。package.json 声明 @audio/* 运行期依赖 →
//   nodePluginRuntime() 判定 "node" → 市场安装走
//   .pair/cordis/node/（源码落盘 + npm install + plugins.json）+ bridge.js 装载。
//
// 为什么不让 goja 轨也跑：@audio/* 是真实 npm ESM 包，goja 沙箱里 require 得到
// 的是 mock 空模块；且 DSP 计算（YIN/PSOLA/WSOLA）需要 V8 级性能。因此本插件在
// **磁盘扫描轨显式让位**（见 apply 首行的运行时自检）——不留半死的工具注册。
//
// 五件工具（全部纯 DSP：无换声模型、无生成式模型、无 TTS）：
//   voice_import   导入素材（WAV 解析 + 哈希 + 元信息 + 可选自动分析）
//   voice_analyze  分析（f0 / notes / vad / slices / loudness）
//   voice_edit     追加编辑命令（pitch.snap / time.warp / speech.removeSilence）
//   voice_render   渲染（源 + edits 的纯函数）+ sourceMap + recipe 落盘
//   voice_verify   六项「非换声 / 非生成式」自检（§7）
//
// 依赖（均 MIT）：@audio/{loudness,note,onset,pitch-yin,stretch-wsola,tune-snap}
// ═══════════════════════════════════════════════════════════════

const DEFAULT_PROJECT = 'voice.project.json';
const DEFAULT_OUTPUT = 'voice.render.wav';
const DEFAULT_TASKS = ['f0', 'notes', 'vad', 'slices', 'loudness'];

// ── 运行时自检（双轨兼容的关键）────────────────────────────
// goja 沙箱：无真实 process/require —— 让位（返回 true 之外什么都不做）。
function isNodeRuntime() {
  try {
    if (typeof process === 'undefined' || !process || !process.versions || !process.versions.node) return false;
    require('node:fs');
    return true;
  } catch (_) {
    return false;
  }
}

// wsRootOf 取工作区根：Node 桥把调用会话的 cwd 注入 exec.agent.session.header.cwd。
function wsRootOf(exec, ctx) {
  try {
    const c = exec && exec.agent && exec.agent.session && exec.agent.session.header && exec.agent.session.header.cwd;
    if (c) return c;
  } catch (_) { /* ignore */ }
  try { if (ctx && ctx.workspaceRoot) return ctx.workspaceRoot; } catch (_) { /* ignore */ }
  try { if (typeof process !== 'undefined' && process.env && process.env.CORDIS_WORKSPACE_ROOT) return process.env.CORDIS_WORKSPACE_ROOT; } catch (_) { /* ignore */ }
  try { return process.cwd(); } catch (_) { return '.'; }
}

function pickSource(proj, id) {
  if (!proj.sources || proj.sources.length === 0) throw new Error('工程里没有源素材（先 voice_import）');
  if (!id) return proj.sources[0];
  const s = proj.sources.find((x) => x.id === id) || proj.sources.find((x) => x.path === id);
  if (!s) throw new Error(`工程里没有源 ${id}（可用：${proj.sources.map((x) => x.id).join(', ')}）`);
  return s;
}

// 编辑命令批量预检（voice_edit / voice_add / voice_import 的 edits 共用）：
// 一条非法即抛出 —— 调用方据此保证「整批不写入」，错误信息点明第几条。
const VOICE_OPS = ['pitch.snap', 'time.warp', 'speech.removeSilence'];
function checkEdits(ops, what) {
  const label = what || 'ops';
  for (let i = 0; i < ops.length; i++) {
    const op = ops[i];
    if (!op || typeof op !== 'object' || Array.isArray(op)) throw new Error(`${label}[${i}] 必须是对象`);
    if (!op.op) throw new Error(`${label}[${i}] 缺 op 字段`);
    if (VOICE_OPS.indexOf(op.op) < 0) {
      throw new Error(`${label}[${i}] 未知 op：${op.op}（当前支持 ${VOICE_OPS.join(' / ')}）`);
    }
  }
}

// 分析结果摘要（只留计数与关键指标，F0 轨迹在旁挂 JSON 里）
function analysisBrief(a) {
  if (!a) return null;
  return {
    f0: a.f0 || null,
    notes: (a.notes || []).length,
    vad: (a.vad || []).length,
    slices: (a.slices || []).length,
    sliceCandidates: a.sliceCandidates || 0,
    loudness: a.loudness || null,
  };
}

// 单件写法：除保留键外全部透传（pitch.snap 的 scale/root/strength/tolerance、time.warp 的 factor、…）
function composeEditFromArgs(args) {
  const o = {};
  for (const k of Object.keys(args)) {
    if (k === 'project' || k === 'edits' || k === 'ops' || k === 'mode' || k === 'path' || k === 'paths') continue;
    o[k] = args[k];
  }
  return o;
}

// ── 工具实现 ───────────────────────────────────────────────

// voice_import
async function voiceImport(args, exec, ctx, lib) {
  const ws = wsRootOf(exec, ctx);
  // ★ 批量：path（单个源）或 paths=[…]（一次导入多个源），可选 ids=[…] 逐个指定源标识。
  //   建工程 + 全部源 + 可选预置编辑链（edits=[…]）一次完成 —— 事务性：任一源缺失/非法即整批不做。
  const reqPaths = Array.isArray(args.paths) && args.paths.length
    ? args.paths.slice()
    : (args.path ? [args.path] : []);
  if (!reqPaths.length) throw new Error('voice_import 需要 path（单个源）或 paths（数组，一次导入多个源）');
  for (let i = 0; i < reqPaths.length; i++) {
    if (typeof reqPaths[i] !== 'string' || !reqPaths[i]) throw new Error(`paths[${i}] 必须是非空字符串`);
  }
  const presetEdits = Array.isArray(args.edits) ? args.edits : [];
  if (presetEdits.length) checkEdits(presetEdits, 'edits');
  const absList = reqPaths.map((p) => lib.project.resolveFrom(ws, p));
  for (let i = 0; i < absList.length; i++) {
    if (!lib.fs.existsSync(absList[i])) throw new Error(`源文件不存在：${absList[i]}（第 ${i + 1} 个；整批未导入）`);
  }
  const projPath = lib.project.resolveFrom(ws, args.project || DEFAULT_PROJECT);
  const proj = lib.project.loadProject(projPath) || lib.project.emptyProject();

  const warnings = [];
  const results = [];
  for (let i = 0; i < absList.length; i++) {
    const abs = absList[i];
    const src = lib.project.readSource(abs);
    const rel = nodeRel(ws, abs);
    const entry = lib.project.registerSource(proj, rel, src.stat);
    const idOpt = Array.isArray(args.ids) && args.ids.length ? args.ids[i] : (i === 0 ? args.id : undefined);
    if (idOpt) entry.id = String(idOpt);
    // 波形峰值缓存（UI 供数；与是否做分析无关）
    try {
      entry.wave = writeWaveCache(lib, src.mono, src.fs, projPath, entry.id);
    } catch (e) {
      /* 缓存失败不影响导入主流程 */
    }

    if (src.stat.peak >= 0.999) warnings.push(`[${entry.id}] 源素材已削波（峰值 ≥ 0.999）——建议先用 declip/降级修复再处理`);
    if (Math.abs(src.stat.dc) > 0.01) warnings.push(`[${entry.id}] 源素材含直流偏移（DC=${src.stat.dc}）——建议加 DC 阻断`);
    if (src.stat.channels > 1) warnings.push(`[${entry.id}] 源为 ${src.stat.channels} 声道，已按算术平均下混为单声道（人声轨）`);
    if (src.stat.seconds < 0.2) warnings.push(`[${entry.id}] 素材过短（<0.2s）——音高/切片分析可能不可靠`);

    let analysis = null;
    if (args.analyze !== false) {
      const tasks = Array.isArray(args.tasks) && args.tasks.length ? args.tasks : DEFAULT_TASKS;
      const a = await lib.analyze.analyzeAll(src.mono, { fs: src.fs, tasks });
      const track = a.track;
      delete a.track;
      if (track) {
        // F0 轨迹旁挂（保持工程 JSON 可 diff）
        const f0Rel = projPath.replace(/\.json$/i, '') + '.' + entry.id + '.f0.json';
        lib.fs.writeFileSync(f0Rel, JSON.stringify(Array.from(track.hz, (v) => (Number.isFinite(v) ? +v.toFixed(3) : null))));
        a.f0 = Object.assign({}, a.f0, { file: nodeBase(f0Rel), hop: track.hop, frameSize: track.frameSize, frames: track.frames });
      }
      a.analyzedAt = new Date().toISOString();
      proj.analysis[entry.id] = a;
      analysis = a;
    }
    results.push({ source: entry, analysis: analysisBrief(analysis) });
  }
  // ★ 建工程时预置编辑链（与 voice_add 同构）：省掉「导入后再逐条 voice_edit」
  if (presetEdits.length) proj.edits = (proj.edits || []).concat(presetEdits);
  lib.project.saveProject(projPath, proj);

  const base = {
    ok: true,
    project: projPath,
    count: results.length,
    edits: (proj.edits || []).length,
    warnings,
  };
  if (results.length === 1) {
    base.source = results[0].source;
    base.analysis = results[0].analysis;
    return base;
  }
  base.sources = results.map((r) => r.source);
  base.analyses = results.map((r) => r.analysis);
  return base;
}

// voice_analyze
async function voiceAnalyze(args, exec, ctx, lib) {
  const ws = wsRootOf(exec, ctx);
  const projPath = lib.project.resolveFrom(ws, args.project || DEFAULT_PROJECT);
  const proj = lib.project.loadProject(projPath);
  if (!proj) throw new Error(`工程不存在：${projPath}（先 voice_import）`);
  const entry = pickSource(proj, args.source);
  const abs = lib.project.resolveFrom(ws, entry.path);
  const src = lib.project.readSource(abs);
  const warnings = [];
  if (entry.sha256 && entry.sha256 !== src.stat.sha256) {
    warnings.push('源文件内容已变化（sha256 与工程登记不一致）——已按新内容分析并更新登记');
    Object.assign(entry, src.stat);
  }
  try {
    entry.wave = writeWaveCache(lib, src.mono, src.fs, projPath, entry.id);
  } catch (e) {
    /* 缓存失败不影响分析主流程 */
  }
  const tasks = Array.isArray(args.tasks) && args.tasks.length ? args.tasks : DEFAULT_TASKS;
  const a = await lib.analyze.analyzeAll(src.mono, { fs: src.fs, tasks, hop: args.hop, frameSize: args.frameSize, jumpCents: args.jumpCents, minDur: args.minDur });
  const track = a.track;
  delete a.track;
  if (track) {
    const f0Rel = projPath.replace(/\.json$/i, '') + '.' + entry.id + '.f0.json';
    lib.fs.writeFileSync(f0Rel, JSON.stringify(Array.from(track.hz, (v) => (Number.isFinite(v) ? +v.toFixed(3) : null))));
    a.f0 = Object.assign({}, a.f0, { file: nodeBase(f0Rel), hop: track.hop, frameSize: track.frameSize, frames: track.frames });
  }
  a.analyzedAt = new Date().toISOString();
  proj.analysis[entry.id] = a;
  lib.project.saveProject(projPath, proj);
  return {
    ok: true,
    project: projPath,
    source: entry.id,
    tasks,
    f0: a.f0 || null,
    notes: a.notes || [],
    vad: a.vad || [],
    slices: a.slices || [],
    loudness: a.loudness || null,
    warnings,
  };
}

// voice_edit
async function voiceEdit(args, exec, ctx, lib) {
  const ws = wsRootOf(exec, ctx);
  const projPath = lib.project.resolveFrom(ws, args.project || DEFAULT_PROJECT);
  const proj = lib.project.loadProject(projPath);
  if (!proj) throw new Error(`工程不存在：${projPath}（先 voice_import）`);
  const ops = Array.isArray(args.ops) ? args.ops : args.op ? [args] : [];
  if (ops.length === 0) throw new Error('voice_edit 需要 ops 数组（或单个 op 对象）');
  // 合法性预检：真正应用由 voice_render 执行（这里只挡明显非法参数，避免工程被写脏）
  // ★ 整批预检 = 批量事务语义：任一条非法则整批不写入（错误信息点明第几条）
  checkEdits(ops, 'ops');
  const mode = args.mode === 'replace' ? 'replace' : 'append';
  proj.edits = mode === 'replace' ? ops.slice() : (proj.edits || []).concat(ops);
  lib.project.saveProject(projPath, proj);
  return { ok: true, action: 'edited', project: projPath, mode, applied: ops.length, edits: proj.edits, count: proj.edits.length, next: 'voice_render 渲染 / voice_verify 自检' };
}

// voice_add
// 批量追加编辑命令（一次调用 N 条，事务式：先整批预检、任一条非法即整批拒）——
// 与 tool-model 的 model_add（parts=[…]）同一形态：单件写法兼容 + 数组批量 + 整批不落盘。
async function voiceAdd(args, exec, ctx, lib) {
  const ws = wsRootOf(exec, ctx);
  const projPath = lib.project.resolveFrom(ws, args.project || DEFAULT_PROJECT);
  const proj = lib.project.loadProject(projPath);
  if (!proj) throw new Error(`工程不存在：${projPath}（先 voice_import）`);
  const ops = Array.isArray(args.edits) ? args.edits : (args.op ? [composeEditFromArgs(args)] : []);
  if (!ops.length) throw new Error('voice_add 需要 edits 数组（批量）；或给 op + 同层参数（单件）');
  checkEdits(ops, 'edits');
  const mode = args.mode === 'replace' ? 'replace' : 'append';
  proj.edits = mode === 'replace' ? ops.slice() : (proj.edits || []).concat(ops);
  lib.project.saveProject(projPath, proj);
  return {
    ok: true, action: 'added', project: projPath, mode,
    applied: ops.length, edits: proj.edits, count: proj.edits.length,
    next: 'voice_render 渲染（源 + edits 纯函数）/ voice_verify 自检',
  };
}

// voice_render
async function voiceRender(args, exec, ctx, lib) {
  const ws = wsRootOf(exec, ctx);
  const projPath = lib.project.resolveFrom(ws, args.project || DEFAULT_PROJECT);
  const proj = lib.project.loadProject(projPath);
  if (!proj) throw new Error(`工程不存在：${projPath}（先 voice_import）`);
  const entry = pickSource(proj, args.source);
  const srcAbs = lib.project.resolveFrom(ws, entry.path);
  const r = await lib.project.renderProject(proj, srcAbs);
  const outRel = args.out || DEFAULT_OUTPUT;
  const outAbs = lib.project.resolveFrom(ws, outRel);
  const bitDepth = Number(args.bitDepth) || 24;
  lib.fs.mkdirSync(nodeDir(outAbs), { recursive: true });
  const buf = lib.wav.writeWav([r.data], { fs: r.src.fs, bitDepth });
  lib.fs.writeFileSync(outAbs, buf);
  // 旁挂：sourceMap 段表 + recipe（可 diff 的 JSON）
  const mapAbs = outAbs.replace(/\.wav$/i, '') + '.sourcemap.json';
  lib.fs.writeFileSync(mapAbs, JSON.stringify({ fs: r.src.fs, frames: r.data.length, segs: r.segs }, null, 2));
  const recAbs = outAbs.replace(/\.wav$/i, '') + '.recipe.json';
  lib.fs.writeFileSync(recAbs, JSON.stringify({
    generatedAt: new Date().toISOString(),
    source: { id: entry.id, path: entry.path, sha256: entry.sha256 },
    edits: proj.edits,
    recipes: r.recipes,
    autos: r.autos,
    deterministic: true,
    engine: 'Node 桥（@audio/* 纯 DSP）· 无换声 / 无生成式模型',
  }, null, 2));
  const summary = lib.project.sourceMapSummary(r.segs, r.src.fs, r.data.length);
  proj.render = {
    output: nodeRel(ws, outAbs),
    bytes: buf.length,
    fs: r.src.fs,
    frames: r.data.length,
    seconds: +(r.data.length / r.src.fs).toFixed(4),
    bitDepth,
    sourceMap: nodeRel(ws, mapAbs),
    recipe: nodeRel(ws, recAbs),
    renderedAt: new Date().toISOString(),
  };
  proj.provenance = { deterministic: true, seed: 0, sourceMap: nodeRel(ws, mapAbs), recipe: nodeRel(ws, recAbs), sourceMapSummary: summary };
  lib.project.saveProject(projPath, proj);
  return {
    ok: true,
    project: projPath,
    output: nodeRel(ws, outAbs),
    bytes: buf.length,
    seconds: proj.render.seconds,
    bitDepth,
    sourceMap: proj.render.sourceMap,
    recipe: proj.render.recipe,
    sourceMapSummary: summary,
    autos: r.autos,
    stages: r.recipes.map((x) => ({ op: x.op, algo: x.algo, version: x.version })),
    next: 'voice_verify 六项自检',
  };
}

// voice_verify
async function voiceVerify(args, exec, ctx, lib) {
  const ws = wsRootOf(exec, ctx);
  const projPath = lib.project.resolveFrom(ws, args.project || DEFAULT_PROJECT);
  const proj = lib.project.loadProject(projPath);
  if (!proj) throw new Error(`工程不存在：${projPath}（先 voice_import）`);
  const entry = pickSource(proj, args.source);
  const srcAbs = lib.project.resolveFrom(ws, entry.path);
  const src = lib.project.readSource(srcAbs);

  let output;
  let segs;
  let recipes;
  let quantBitDepth = null; // 审「已保存的 WAV」时的位深 —— 比较双方必须同量化
  if (args.output) {
    const outAbs = lib.project.resolveFrom(ws, args.output);
    if (!lib.fs.existsSync(outAbs)) throw new Error(`输出文件不存在：${outAbs}`);
    const read = lib.project.readSource(outAbs);
    output = read.mono;
    quantBitDepth = read.parsed.bitDepth;
    const mapAbs = outAbs.replace(/\.wav$/i, '') + '.sourcemap.json';
    if (lib.fs.existsSync(mapAbs)) segs = JSON.parse(lib.fs.readFileSync(mapAbs, 'utf8')).segs || [];
    else segs = [];
    const recAbs = outAbs.replace(/\.wav$/i, '') + '.recipe.json';
    recipes = lib.fs.existsSync(recAbs) ? JSON.parse(lib.fs.readFileSync(recAbs, 'utf8')).recipes || [] : [];
  } else {
    const r = await lib.project.renderProject(proj, srcAbs);
    output = r.data;
    segs = r.segs;
    recipes = r.recipes;
  }
  const res = await lib.verify.runVerify({
    pluginDir: nodeDir(nodeResolve(__filename)),
    workspaceRoot: ws,
    auditWorkspace: !!args.auditWorkspace,
    fs: src.fs,
    source: src.mono,
    output,
    sourceMapSegs: segs,
    recipes: recipes || [],
    edits: proj.edits || [],
    renderAgain: async () => {
      const again = await lib.project.renderProject(proj, srcAbs);
      // 审 WAV 文件时第二次渲染也按同位深量化后再比 —— 否则 24-bit 量化步长
      // （≈1.2e-7）会被误判成「不确定性」（7.4 判的是算法确定性，不是位深）。
      if (quantBitDepth) {
        const q = lib.wav.parseWav(lib.wav.writeWav([again.data], { fs: src.fs, bitDepth: quantBitDepth }));
        return { data: q.data[0], quantized: true };
      }
      return { data: again.data };
    },
  });
  proj.checks = {
    at: new Date().toISOString(),
    standard: res.standard,
    pass: res.pass,
    passed: res.passed,
    total: res.total,
    checks: res.checks.map((c) => ({ id: c.id, name: c.name, pass: c.pass, metric: c.metric })),
  };
  lib.project.saveProject(projPath, proj);
  return res;
}

// node:path 惰性包装（顶层不 require —— 保证 goja 让位路径零风险）
function nodePathLib() { return require('node:path'); }

// writeWaveCache 写波形峰值缓存（10 ms/帧的绝对值峰值）——供 UI 半（同包 client.js）
// 绘制波形，避免为"看一眼波形"新增一个工具或让 UI 去拉完整 PCM。
// 返回旁挂文件名（相对路径）。
function writeWaveCache(lib, mono, fs, basePath, sourceId) {
  const frameMs = 10;
  const win = Math.max(1, Math.round((fs * frameMs) / 1000));
  const n = Math.ceil(mono.length / win);
  const peaks = new Array(n);
  for (let i = 0; i < n; i++) {
    let mx = 0;
    const s = i * win;
    const e = Math.min(mono.length, s + win);
    for (let k = s; k < e; k++) {
      const a = mono[k] < 0 ? -mono[k] : mono[k];
      if (a > mx) mx = a;
    }
    peaks[i] = +mx.toFixed(4);
  }
  const rel = basePath.replace(/\.json$/i, '') + '.' + sourceId + '.wave.json';
  lib.fs.writeFileSync(rel, JSON.stringify({ fs, frameMs, frames: mono.length, peaks }));
  return nodeBase(rel);
}
function nodeResolve(p) { return nodePathLib().resolve(p); }
function nodeDir(p) { return nodePathLib().dirname(p); }
function nodeBase(p) { return nodePathLib().basename(p); }
function nodeRel(from, to) {
  try { return nodePathLib().relative(from, to).split(nodePathLib().sep).join('/') || nodeBase(to); } catch (_) { return to; }
}

// ── 工具声明 ───────────────────────────────────────────────
const TOOL_DEFS = [
  {
    name: 'voice_import',
    description: '导入人声素材（WAV）到人声工程：解析头信息、算 sha256/峰值/RMS/DC，登记到 voice.project.json，并可选自动分析（f0/notes/vad/slices/loudness）。支持一次导入多个源（paths=[…]）并在建工程时预置编辑链（edits=[…]）。纯 DSP 链路，不含任何换声/生成式模型。',
    usageGuide: '人声创作第一步。path / paths 相对主项目根解析（跨项目传绝对路径）。默认工程 voice.project.json、默认自动分析。一次导入多段素材用 paths=[…]（可选 ids=[…] 逐个指定源标识）；要顺带配好处理链就带 edits=[…]（与 voice_add 同构）。导入后即可 voice_add / voice_edit 追加编辑命令，再 voice_render 渲染、voice_verify 自检。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '源音频文件路径（WAV：PCM u8/i16/i24/i32 或 IEEE float32/64；相对主项目根解析，跨项目传绝对路径）' },
        paths: { type: 'array', description: '可选：一次导入多个源（字符串数组；与 path 二选一。任一源不存在则整批不导入）' },
        ids: { type: 'array', description: '可选：与 paths 对应的源标识数组（不传则自动 v1、v2…）' },
        project: { type: 'string', description: '可选：工程文件路径（默认 <主项目根>/voice.project.json；相对主项目根解析，跨项目传绝对路径）' },
        id: { type: 'string', description: '可选：源标识（默认 v1、v2…）' },
        analyze: { type: 'boolean', description: '可选：是否自动分析（默认 true）' },
        tasks: { type: 'array', description: '可选：分析项（默认 ["f0","notes","vad","slices","loudness"]）' },
        edits: { type: 'array', description: '可选：导入时预置的编辑链（每项与 voice_add 的 edits 项 / voice_edit 的 op 同构）；任一条非法则整批不导入' },
      },
    },
    impl: voiceImport,
  },
  {
    name: 'voice_analyze',
    description: '对工程内素材做分析并写回工程：f0（YIN 基频轨迹）、notes（音符段：起止/MIDI/置信度）、vad（有声/无声段）、slices（音节切片，多源候选 + 起音事件判据）、loudness（BS.1770-4 LUFS/true peak）。结果幂等、可重复调用。',
    usageGuide: '编辑前后都可跑：编辑前用于看源素材走向，编辑后用于核对吸附/伸缩效果。F0 轨迹另存旁挂 JSON（工程 JSON 保持可 diff）。分析的音高单位为 midiFloat（含小数半音）。',
    category: '创作',
    readOnly: false,
    parameters: {
      type: 'object',
      properties: {
        project: { type: 'string', description: '可选：工程路径（默认 <主项目根>/voice.project.json；相对主项目根解析）' },
        source: { type: 'string', description: '可选：源 id（默认第一个源）' },
        tasks: { type: 'array', description: '可选：分析项子集' },
        hop: { type: 'integer', description: '可选：F0 帧移（默认 512）' },
        frameSize: { type: 'integer', description: '可选：F0 分析窗（默认 2048）' },
        jumpCents: { type: 'number', description: '可选：音符段切分阈值（默认 80 音分）' },
        minDur: { type: 'number', description: '可选：最短短音符段时长（默认 0.05s，更短并入邻段）' },
      },
    },
    impl: voiceAnalyze,
  },
  {
    name: 'voice_add',
    description: '批量追加编辑命令到人声工程（一次调用任意多条，事务式）。edits=[{op:"pitch.snap",scale:"minor",root:0,strength:1},{op:"time.warp",factor:0.85},{op:"speech.removeSilence",minGap:0.35}] —— 每项与 voice_edit 的 ops 项同构（支持 pitch.snap / time.warp / speech.removeSilence）；也支持单件写法（op + 同层参数）。先整批预检，任一条非法即整批不写入（工程保持原样）。mode=replace 可整体替换编辑链。',
    usageGuide: '一次把整条链配好：voice_add edits=[{"op":"pitch.snap","scale":"minor","root":0},{"op":"speech.removeSilence","minGap":0.35,"keep":0.08},{"op":"time.warp","factor":0.9}]。渲染是「源 + edits」的纯函数，链上每条都会被 recipe 记录（voice_verify 的 7.5 链可见判据）。写完用 voice_render 渲染、voice_verify 自检；微调单条用 voice_edit。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        project: { type: 'string', description: '可选：工程路径（默认 <主项目根>/voice.project.json；相对主项目根解析）' },
        edits: { type: 'array', description: '编辑命令数组（每项 {op:"pitch.snap"|"time.warp"|"speech.removeSilence", …参数}；批量写法）' },
        op: { type: 'string', description: '可选（单件写法）：单条 op 名（与同层参数组成一条命令）' },
        mode: { type: 'string', description: '可选：append（默认，追加）| replace（替换整条编辑链）' },
      },
    },
    impl: voiceAdd,
  },
  {
    name: 'voice_edit',
    description: '向人声工程追加编辑命令（命令模型，与 UI 拖拽写的是同一种 op）。支持：pitch.snap（音阶吸附，Auto-Tune 类，PSOLA 只改音高不改音色）、time.warp（WSOLA 时值伸缩/语速）、speech.removeSilence（删停顿，保留呼吸边距）。批量追加整条链用 voice_add（edits 数组一次成型）更省。返回结构化结果（ok/applied/count）。',
    usageGuide: 'op 示例：{"op":"pitch.snap","scale":"minor","root":0,"strength":1,"tolerance":12} / {"op":"time.warp","factor":0.85} / {"op":"speech.removeSilence","minGap":0.35,"keep":0.08}。tolerance 默认 12 音分（大于 YIN 的系统性偏差，避免把准的音修坏）。mode=replace 可整体替换编辑链。写完后用 voice_render 渲染。 ★ 批量：一次调用可传任意多条 op（先整批预检、非法即整批拒，再统一写入编辑链）—— 多段处理一次传完，别一条一条调。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        project: { type: 'string', description: '可选：工程路径（默认 <主项目根>/voice.project.json；相对主项目根解析）' },
        ops: { type: 'array', description: '编辑命令数组（按顺序应用；也可直接传单个 op 对象的字段）' },
        op: { type: 'string', description: '可选：单个 op 名（与同层参数组成一条命令）' },
        mode: { type: 'string', description: '可选：append（默认，追加）| replace（替换整条编辑链）' },
      },
      required: ['ops'],
    },
    impl: voiceEdit,
  },
  {
    name: 'voice_render',
    description: '渲染人声工程：源素材 + edits 的纯函数 → WAV（默认 24-bit），同时落盘 sourceMap 段表（输出↔源的时间映射）与 recipe（每个处理阶段的算法名/版本/参数）。同一工程重复渲染位级一致。',
    usageGuide: '渲染产物三件套：<out>.wav、<out>.sourcemap.json、<out>.recipe.json（工程内登记）。sourceMap 是"非无源合成"的机制证据；recipe 是"链可见"的证据。渲染不改变工程（除登记产物路径）。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        project: { type: 'string', description: '可选：工程路径（默认 <主项目根>/voice.project.json；相对主项目根解析）' },
        source: { type: 'string', description: '可选：源 id（默认第一个源）' },
        out: { type: 'string', description: '可选：输出 WAV 路径（默认 <主项目根>/voice.render.wav；相对主项目根解析）' },
        bitDepth: { type: 'integer', description: '可选：位深 16/24/32（默认 24）' },
      },
    },
    impl: voiceRender,
  },
  {
    name: 'voice_verify',
    description: '执行六项「非换声 / 非生成式」专项自检：7.1 依赖审计（禁止清单与模型权重后缀零命中）、7.2 帧级可追溯（sourceMap 全覆盖 + 重建包络相关 ≥0.99）、7.3 音色一致性（低次谐波 H2≤10%、H3≤15%，高次放宽）、7.4 确定性（两次渲染位级一致）、7.5 链可见（recipe 覆盖全部阶段）、7.6 无新增音源（每个有声段都可由源帧解释）。',
    usageGuide: '可对任意渲染结果调用：传 output 则审该文件（需同目录 .sourcemap.json/.recipe.json），不传则现场渲染并审。任一 FAIL 意味着"基于用户自己声音"的承诺被破坏，应视为构建失败。auditWorkspace=true 时把工作区也纳入 7.1 扫描（更严，但可能命中与本插件无关的文件）。',
    category: '创作',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        project: { type: 'string', description: '可选：工程路径（默认 <主项目根>/voice.project.json；相对主项目根解析）' },
        source: { type: 'string', description: '可选：源 id（默认第一个源）' },
        output: { type: 'string', description: '可选：要审的渲染结果（相对主项目根解析；默认现场渲染）' },
        auditWorkspace: { type: 'boolean', description: '可选：7.1 是否连工作区一起扫（默认 false——只审本插件自身依赖）' },
      },
    },
    impl: voiceVerify,
  },
];

// ── 插件导出（双轨兼容：CJS module.exports + 顶层 return）──────
const PLUGIN = {
  name: 'tool-voice',
  purpose: '人声创作工具面（纯 DSP）：导入/分析/编辑/渲染/六项自检 —— 音高吸附、时值伸缩、停顿删减、确定性溯源（无换声、无生成式模型）',
  inject: [],
  apply(ctx) {
    if (!isNodeRuntime()) {
      // goja 沙箱轨（.pair/plugins 磁盘扫描）——本插件依赖真实 Node（Buffer/fs/@audio/*），
      // 在此显式让位、不注册任何工具，避免"装上但半死"的工具面污染。
      try {
        if (ctx && ctx.logger && ctx.logger.info) {
          ctx.logger.info('[tool-voice] 当前为 goja 沙箱轨（无真实 Node）——本插件为 Node 桥轨，已让位（未注册工具）');
        }
      } catch (_) { /* ignore */ }
      return;
    }
    const lib = {
      fs: require('node:fs'),
      wav: require('./lib/wav'),
      dsp: require('./lib/dsp'),
      fixture: require('./lib/fixture'),
      deps: require('./lib/deps'),
      analyze: require('./lib/analyze'),
      ops: require('./lib/ops'),
      project: require('./lib/project'),
      verify: require('./lib/verify'),
    };
    for (const t of TOOL_DEFS) {
      ctx.tools.register({
        name: t.name,
        description: t.description,
        usageGuide: t.usageGuide,
        category: t.category,
        readOnly: !!t.readOnly,
        parameters: t.parameters,
        execute: (args, exec) => t.impl(args || {}, exec || {}, ctx, lib),
      });
    }
    try {
      if (ctx && ctx.logger && ctx.logger.info) {
        ctx.logger.info('[tool-voice] 已注册 ' + TOOL_DEFS.length + ' 个工具（Node 桥轨 · 纯 DSP 人声链）');
      }
    } catch (_) { /* ignore */ }
  },
};

if (typeof module !== 'undefined' && module.exports) module.exports = PLUGIN;
return PLUGIN;
