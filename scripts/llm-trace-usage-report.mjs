#!/usr/bin/env node
// scripts/llm-trace-usage-report.mjs — llm-trace 日志的「**API 真实 usage**」汇总报告。
//
// 用法：
//   node scripts/llm-trace-usage-report.mjs [日志路径] [--by-turn] [--raw-keys] [--json]
//   省略路径 → 自动取 .pair/logs/llm-trace/ 下最新的 llm-trace-*.jsonl
//
// 说明（2026-09-12 起）：
//   response 事件的 usage 为 **API 真实返回**（source:"api"），含
//     promptTokens / completionTokens / totalTokens / promptCacheHitTokens /
//     promptCacheMissTokens / cacheHitRate / **apiRaw**（provider 原始报文，未归一化）。
//   usageEstimated 为**本地估算**的 prompt 构成（source:"local-estimate"），
//     本脚本只用它做「估算 vs 真实」偏差提示，不计入成本口径。
//
// 成本口径：命中 token 按未命中价的 1/10 折算（可用 --hit-ratio 覆盖）。
//   注意：命中率是**比例**，账单是**绝对量**——把命中率做高可以通过灌水命中 token
//   实现（多发已缓存前缀），那会让账单上升；判断省不省要看 miss 绝对量。

import fs from 'node:fs';
import path from 'node:path';
import readline from 'node:readline';

const argv = process.argv.slice(2);
const flags = new Set(argv.filter((a) => a.startsWith('--')));
const positional = argv.filter((a) => !a.startsWith('--'));
let hitDiscount = 0.1;
const hrIdx = argv.findIndex((a) => a === '--hit-ratio');
if (hrIdx >= 0 && argv[hrIdx + 1]) hitDiscount = Number(argv[hrIdx + 1]);

function latestLog() {
  const dir = path.join(process.cwd(), '.pair', 'logs', 'llm-trace');
  if (!fs.existsSync(dir)) throw new Error(`目录不存在: ${dir}`);
  const files = fs
    .readdirSync(dir)
    .filter((f) => /^llm-trace-\d{8}(-\d+)?\.jsonl$/.test(f))
    .map((f) => ({ f, m: fs.statSync(path.join(dir, f)).mtimeMs }))
    .sort((a, b) => b.m - a.m);
  if (!files.length) throw new Error(`无日志文件: ${dir}`);
  return path.join(dir, files[0].f);
}

const file = positional[0] || latestLog();
if (!fs.existsSync(file)) throw new Error(`日志不存在: ${file}`);

const tot = {
  calls: 0, prompt: 0, completion: 0, hit: 0, miss: 0,
  noUsage: 0, legacy: 0, withRaw: 0, hitRateSum: 0, hitRateN: 0,
};
const byTurn = new Map();
const rawKeys = new Map();
const estVsReal = [];
let estSum = 0, estN = 0;

const rl = readline.createInterface({ input: fs.createReadStream(file, 'utf8'), crlfDelay: Infinity });
for await (const line of rl) {
  const s = line.trim();
  if (!s) continue;
  let ev;
  try { ev = JSON.parse(s); } catch { continue; }
  if (ev.phase !== 'response') continue;
  tot.calls++;
  const u = ev.usage;
  if (!u) { tot.noUsage++; continue; }
  if (u.source !== 'api') tot.legacy++; // 旧格式（本次改动前落盘）
  const p = u.promptTokens || 0, h = u.promptCacheHitTokens || 0, m = u.promptCacheMissTokens || 0;
  tot.prompt += p; tot.completion += u.completionTokens || 0;
  tot.hit += h; tot.miss += m;
  const denom = h + m;
  if (denom > 0) { tot.hitRateSum += h / denom; tot.hitRateN++; }
  if (u.apiRaw && typeof u.apiRaw === 'object') {
    tot.withRaw++;
    for (const k of Object.keys(u.apiRaw)) rawKeys.set(k, (rawKeys.get(k) || 0) + 1);
  }
  const t = ev.turn ?? '?';
  const g = byTurn.get(t) || { calls: 0, prompt: 0, hit: 0, miss: 0 };
  g.calls++; g.prompt += p; g.hit += h; g.miss += m;
  byTurn.set(t, g);
  // 估算 vs 真实（仅提示偏差，不参与成本）
  const est = ev.usageEstimated;
  if (est && est.source === 'local-estimate') {
    const es = (est.systemTokens || 0) + (est.skillsTokens || 0) + (est.mcpTokens || 0) +
      (est.toolTokens || 0) + (est.historyTokens || 0) + (est.otherTokens || 0);
    if (es > 0) { estSum += es; estN++; }
    if ((est.historyTokens || 0) === 0 && p > 0) estVsReal.push({ turn: t, step: ev.step, prompt: p, raw: u.apiRaw });
  }
}

const rate = (h, m) => (h + m > 0 ? (100 * h) / (h + m) : 0);
const cost = tot.miss + tot.hit * hitDiscount;              // 相对全价单位
const naiveCost = tot.prompt - tot.completion * 0;          // 全按未命中价（参照）

const out = {
  file,
  calls: tot.calls,
  callsWithoutUsage: tot.noUsage,
  callsLegacyFormat: tot.legacy,
  callsWithApiRaw: tot.withRaw,
  promptTokens: tot.prompt,
  completionTokens: tot.completion,
  cacheHitTokens: tot.hit,
  cacheMissTokens: tot.miss,
  cacheHitRatePct: Number(rate(tot.hit, tot.miss).toFixed(2)),
  costUnits: Math.round(cost),
  costIfNoCacheUnits: tot.prompt,
  hitDiscount,
  estimatedBreakdownSamples: estN,
  estimatedVsRealPromptAvg: estN ? Math.round(estSum / estN) : 0,
  realPromptAvg: tot.calls ? Math.round(tot.prompt / tot.calls) : 0,
};

if (flags.has('--json')) {
  console.log(JSON.stringify(out, null, 2));
} else {
  console.log(`日志: ${file}`);
  console.log(`调用数: ${out.calls}（无 usage ${out.callsWithoutUsage}，旧格式 ${out.callsLegacyFormat}，含 apiRaw ${out.callsWithApiRaw}）`);
  console.log(`prompt tokens: ${out.promptTokens.toLocaleString()}   completion: ${out.completionTokens.toLocaleString()}`);
  console.log(`缓存命中: ${out.cacheHitTokens.toLocaleString()}   未命中: ${out.cacheMissTokens.toLocaleString()}`);
  console.log(`★ 加权缓存命中率: ${out.cacheHitRatePct}%   （每次调用平均命中率 ${tot.hitRateN ? (100 * tot.hitRateSum / tot.hitRateN).toFixed(2) : '0'}%）`);
  console.log(`折算成本(相对全价): ${out.costUnits.toLocaleString()}   若不命中缓存: ${out.costIfNoCacheUnits.toLocaleString()}   节省 ${(100 * (1 - out.costUnits / (out.costIfNoCacheUnits || 1))).toFixed(1)}%`);
  if (estN) {
    console.log(`估算构成 vs 真实 prompt: 估算均值 ${out.estimatedVsRealPromptAvg.toLocaleString()} / 真实均值 ${out.realPromptAvg.toLocaleString()}（估算仅参考）`);
  }
  if (!tot.withRaw) {
    console.log('提示: 本日志无 apiRaw —— 说明是 2026-09-12 改动前落盘的旧格式，或宿主未重启加载新内核。');
  }
  if (flags.has('--raw-keys') && rawKeys.size) {
    console.log('\napiRaw 字段出现频次（provider 原始报文键）:');
    for (const [k, n] of [...rawKeys.entries()].sort((a, b) => b[1] - a[1])) {
      console.log(`  ${String(n).padStart(6)}  ${k}`);
    }
  }
  if (flags.has('--by-turn')) {
    console.log('\n按 turn 汇总:');
    console.log('  turn   calls   promptTokens  hitTokens  missTokens  hitRate');
    for (const [t, g] of [...byTurn.entries()].sort((a, b) => String(a[0]).localeCompare(String(b[0]), undefined, { numeric: true }))) {
      console.log(`  ${String(t).padEnd(6)} ${String(g.calls).padStart(5)}   ${String(g.prompt).padStart(12)} ${String(g.hit).padStart(10)} ${String(g.miss).padStart(11)}  ${rate(g.hit, g.miss).toFixed(1)}%`);
    }
  }
  if (estVsReal.length) {
    console.log(`\n注意: ${estVsReal.length} 次调用的估算构成 historyTokens=0（估算未覆盖历史段）——真实值以 usage/apiRaw 为准。`);
  }
}
