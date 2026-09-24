#!/usr/bin/env node
// ═══════════════════════════════════════════════════════════════
// scripts/audit-plugin-tool-desc.cjs — 审计「插件工具描述在 agent 侧的真实可见面」
//
// 背景（2026-09-18 排查结论）：
//   工具定义下发给 LLM 的唯一路径是 Registry.Definitions()（internal/agent/tools.go:297），
//   它只带两样东西：
//     ① Function.Description ← 经 trimToolDesc 截断到 ~120 rune（优先在「。」断，同文件 :328）
//     ② Function.Parameters  ← 原样完整下发，不截断
//   其余字段 agent 收不到：
//     · Tool.UsageGuide  → 只进 AllToolMeta()（前端 UI 面板可见），UsageGuideText() 恒返回空串
//     · package.json 的 purpose / PLUGIN.purpose → 宿主 Go 侧完全不读
//     · ctx.systemPrompt.section → 可用，但 PluginPromptSections 不随工具集裁剪联动（会误导）
//   故「给 agent 的指导」只有两个落点：description 前 120 字 + 参数 description。
//   改任何工具描述后，务必用本脚本看**截断后的真实结果**，避免关键信息被挤出可见区。
//
// 用法：
//   node scripts/audit-plugin-tool-desc.cjs                 # 扫六域创作插件
//   node scripts/audit-plugin-tool-desc.cjs tool-design     # 指定插件目录名（可多个）
//   node scripts/audit-plugin-tool-desc.cjs --brief         # 只打印被截断/缺基准的告警
//
// 说明：
//   · 用 new Function('ctx', src) 装载插件 —— 与 goja 把插件文件当**函数体**执行的语义一致，
//     文件末尾的顶层 `return PLUGIN` 因此合法；再调 apply(ctx) 捕获 ctx.tools.register 的实参。
//   · tool-voice 属 Node 桥轨（依赖真实 Node：Buffer/@audio/*），goja 轨显式让位、不注册工具
//     → 本脚本对其显示「0 工具」，其描述需人工核对（文本扫描）。
// ═══════════════════════════════════════════════════════════════

const fs = require('fs');
const path = require('path');

const REPO = path.resolve(__dirname, '..');
const DEFAULT_PKGS = ['tool-art', 'tool-design', 'tool-model', 'tool-music', 'tool-rig', 'tool-voice'];

const argv = process.argv.slice(2);
const brief = argv.includes('--brief');
const pkgs = argv.filter((a) => !a.startsWith('--'));
const targets = pkgs.length ? pkgs : DEFAULT_PKGS;

// ── 复刻宿主 trimToolDesc（internal/agent/tools.go:328）──────────
// 120 rune 处取 head，在 head 内找最后一个「。」；若该句号位置 < 60
// （说明句号太靠前或没有），回退 cut=100 再向前找空格/逗号，
// 找不到就停在 60 —— 这是宿主的已知缺陷（会把描述反向砍短，如 rig_model 只可见 60 字）。
function trimToolDesc(desc) {
  if (desc == null) return '';
  const runes = Array.from(desc);
  if (runes.length <= 120) return desc;
  const head = runes.slice(0, 120);
  let cut = -1;
  head.forEach((r, i) => { if (r === '。') cut = i; });
  if (cut < 60) {
    cut = 100;
    if (cut >= head.length) cut = head.length - 1;
    while (cut > 60 && head[cut] !== ' ' && head[cut] !== ',') cut--;
  }
  return head.slice(0, cut).join('').trim();
}

// 路径/产物类参数：这些必须说明解析基准（否则 agent 不知道文件落在哪）
const LOC_KEYS = ['path', 'out', 'targetDir', 'dir', 'root', 'report', 'file', 'tokens', 'html', 'project', 'output'];
const BASE_RE = /项目根|工作区根|绝对路径/;

// isLocArg 判定某参数是否属于「文件/目录落点」：
//   · 键名在 LOC_KEYS 内（path/out/report/targetDir/project/output…）
//   · 且类型是 string（JSON 对象/数组参数不是路径，如 design_tokens.tokens）
//   · 且描述不是「子参数说明」形态（如 rig_edit.output = '可选（physics.add）：{parameter,scale} 或直接参数 id'）
function isLocArg(k, p) {
  if (!LOC_KEYS.includes(k)) return false;
  if (!p || p.type !== 'string') return false;
  const d = p.description || '';
  if (/^\s*可选\s*（[a-z0-9_.]+[^）]*）\s*[:：]/.test(d)) return false;
  return true;
}

const stats = { tools: 0, truncated: 0, shortCut: 0, locMissing: 0, locTotal: 0 };

for (const pkg of targets) {
  const file = path.join(REPO, 'plugins-dist', pkg, 'index.js');
  if (!fs.existsSync(file)) { console.log(`!! 找不到 ${file}`); continue; }
  const src = fs.readFileSync(file, 'utf8');

  const cap = { tools: [], sections: [], prompts: [] };
  const ctx = {
    tools: { register: (t) => cap.tools.push(t) },
    systemPrompt: { section: (s) => cap.sections.push(s), variable: () => {} },
    prompts: { provide: (o) => cap.prompts.push(o), remove: () => {} },
    logger: () => ({ info: () => {}, warn: () => {}, error: () => {} }),
    fs: {},
  };
  let loadErr = null;
  try {
    const ret = new Function('ctx', src)(ctx);
    if (ret && typeof ret.apply === 'function') ret.apply(ctx);
    else if (typeof ret === 'function') ret(ctx);
  } catch (e) { loadErr = e; }

  if (!brief) {
    console.log(`\n══════════ ${pkg}  工具 ${cap.tools.length} | 系统提示段 ${cap.sections.length} | 提示词资产 ${cap.prompts.length} ══════════`);
    if (loadErr) console.log(`  !! 装载失败: ${loadErr.message}`);
    if (!cap.tools.length) console.log('  (0 工具：Node 桥轨在 goja 下让位，或装载失败 —— 需人工核对描述文本)');
  }

  for (const t of cap.tools) {
    stats.tools++;
    const full = t.description || '';
    const vis = trimToolDesc(full);
    const cut = vis !== full;
    if (cut) stats.truncated++;
    const head = Array.from(full).slice(0, 120);
    const firstDot = head.findIndex((r) => r === '。');
    const suspicious = cut && firstDot >= 0 && firstDot < 60;
    if (suspicious) stats.shortCut++;

    const props = (t.parameters && t.parameters.properties) || {};
    const locHits = Object.keys(props).filter((k) => isLocArg(k, props[k]));
    const badLoc = locHits.filter((k) => !BASE_RE.test(props[k].description || ''));
    stats.locTotal += locHits.length;
    stats.locMissing += badLoc.length;

    if (brief) {
      if (cut || badLoc.length) {
        console.log(`[${pkg}] ${t.name}: 描述 ${Array.from(full).length}→${Array.from(vis).length} 字${suspicious ? ' (短句号砍短!)' : ''}${badLoc.length ? ` | 缺基准参数: ${badLoc.join(',')}` : ''}`);
      }
      continue;
    }

    console.log(`\n▸ ${t.name}  描述 ${Array.from(full).length} 字 → agent 可见 ${Array.from(vis).length} 字${cut ? '  ⚠ 被截断' : '  ✓ 完整'}`);
    console.log(`   可见: ${vis}`);
    if (suspicious) {
      console.log(`   ⚠ 宿主 trimToolDesc 缺陷：首个句号在第 ${firstDot} 字（<60），回退分支把描述砍到 ${Array.from(vis).length} 字 —— 考虑缩短首句或让首句号落在 60–120 区间`);
    }
    if (cut) {
      const rest = Array.from(full).slice(Array.from(vis).length).join('').trim();
      console.log(`   ✂ 被截掉: ${rest.slice(0, 80)}${rest.length > 80 ? '…' : ''}`);
    }
    for (const k of locHits) {
      const d = props[k].description || '';
      const ok = BASE_RE.test(d);
      console.log(`     [${ok ? '✓' : '✗'}] ${k}: ${d}`);
    }
  }
}

console.log('\n────── 汇总 ──────');
console.log(`工具 ${stats.tools} 个；描述被截断 ${stats.truncated} 个（其中疑似被宿主短句号缺陷砍短 ${stats.shortCut} 个）`);
console.log(`路径类参数 ${stats.locTotal} 个，未说明解析基准 ${stats.locMissing} 个`);
if (stats.locMissing || stats.shortCut) process.exitCode = 1;
