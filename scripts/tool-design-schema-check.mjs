// tool-design-schema-check.mjs — tool-design 结构自省（design_schema）自检
//
// 用法：node scripts/tool-design-schema-check.mjs
//
// 背景（2026-10-15）：用户反馈「设计插件 agent 不知道他具体结构」——13 类节点的字段全表、
//   图标白名单、布局语义、判据、限额此前只散落在实现里，工具描述写不全，agent 只能猜
//   （猜错即 D2 悬空引用 / D7 图标不在白名单 FAIL）→ 设计稿退化成 frame+text+button 堆叠。
//   本脚本以与 goja 沙箱**相同的「函数体」语义**加载插件源码，注册工具并真实调用
//   design_schema，断言输出完整、范例 JSON 合法、描述错误（$spacing）已修。
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
// ★ 2026-10-16：设计插件转正为基线插件，真源从 plugins-dist/ 迁回 .pair/plugins/。
const file = path.join(root, '.pair', 'plugins', 'tool-design', 'index.js')
const src = fs.readFileSync(file, 'utf8')

// goja 把本文件当函数体执行（顶层 `return PLUGIN` 合法）——这里保持同一语义
const fn = new Function('module', 'exports', 'require', 'ctx', '__dirname', '__filename', src)
const mod = { exports: {} }
let ret
try {
  ret = fn(mod, mod.exports, () => { throw new Error('沙箱无 require') }, undefined, path.dirname(file), file)
} catch (e) {
  console.error('✗ 加载插件失败（语法/运行错误）：' + (e && e.message))
  process.exit(2)
}
const PLUGIN = (mod.exports && mod.exports.name) ? mod.exports : ret
if (!PLUGIN || !PLUGIN.name) { console.error('✗ 未取到 PLUGIN（module.exports / return 均为空）'); process.exit(2) }

const defs = []
const ctx = {
  tools: { register(t) { defs.push(t) } },
  fs: { exists: () => false, readFile: () => '', writeFile: () => {} },
  logger: () => ({ info: () => {}, warn: () => {}, error: () => {} }),
}
PLUGIN.apply(ctx)

const results = []
const check = (name, ok, detail) => results.push({ name, ok: !!ok, detail })

const byName = {}
for (const d of defs) byName[d.name] = d
const schema = byName['design_schema']

// A1 注册齐全（design_schema 在列且只读）
check('A1 design_schema 已注册且只读',
  !!schema && schema.readOnly === true && defs.length >= 8,
  `工具数=${defs.length}，含 design_schema=${!!schema}，readOnly=${schema && schema.readOnly}`)

const schemaText = t => schema.execute({ topic: t })

// A2 topic=all：七个小节都在
const all = schemaText('all')
const needAll = ['节点类型', '令牌引用', '图标', '布局要点', '交互（props.action）', '上限', '判据']
const missing = needAll.filter(k => all.indexOf(k) < 0)
check('A2 topic=all 含全部小节', missing.length === 0, missing.length ? '缺：' + missing.join('/') : `${all.length} 字符`)

// A3 topic=nodes：表格行数必须与 topic=all 自己声明的类数**自洽**（不硬编码，防真源漂移）
const nodesTxt = schemaText('nodes')
const rowCount = nodesTxt.split('\n').filter(l => /^\| [a-z]/.test(l)).length
const declaredM = all.match(/## 节点类型（(\d+) 类）/)
const declared = declaredM ? Number(declaredM[1]) : -1
check('A3 topic=nodes 字段全表与声明的类数自洽', declared > 0 && rowCount === declared,
  `声明 ${declared} 类，表格数据行 ${rowCount}`)

// A4 topic=tokens：8 个令牌组全在
const tokTxt = schemaText('tokens')
const groups = ['color', 'space', 'radius', 'fontSize', 'fontWeight', 'lineHeight', 'shadow', 'font']
const gmiss = groups.filter(g => tokTxt.indexOf('| ' + g + ' ') < 0)
check('A4 topic=tokens 覆盖全部 8 个令牌组', gmiss.length === 0, gmiss.length ? '缺：' + gmiss.join('/') : `含 ${groups.length} 组`)

// A5 topic=icons：白名单名字直接可用（抽查 3 个真实存在的名字）
const iconTxt = schemaText('icons')
const iconHit = ['home', 'chevron-right', 'trash'].filter(n => iconTxt.indexOf(n) >= 0)
check('A5 topic=icons 列出白名单名字', iconHit.length === 3, `抽查命中 ${iconHit.length}/3`)

// A6 topic=examples：每段范式都必须是**合法 JSON**（否则 agent 照抄必失败）
const exTxt = schemaText('examples')
const blocks = exTxt.split('```json').slice(1).map(s => s.split('```')[0].trim())
let badJson = []
blocks.forEach((b, i) => { try { JSON.parse(b) } catch (e) { badJson.push('#' + (i + 1) + ':' + e.message) } })
check('A6 topic=examples 的范式 JSON 全部合法', blocks.length >= 4 && badJson.length === 0,
  `片段 ${blocks.length} 段，非法 ${badJson.length}${badJson.length ? '（' + badJson.join(' ') + '）' : ''}`)

// A7 描述错误已修：不再出现 $spacing（真源是 space）
const badDesc = defs.filter(d => String(d.description || '').indexOf('$spacing') >= 0 || String(d.usageGuide || '').indexOf('$spacing') >= 0)
check('A7 描述里 $spacing 误用已修（真源组名 space）', badDesc.length === 0,
  badDesc.length ? '仍误用：' + badDesc.map(d => d.name).join('/') : '无残留')

// A8 未知 topic 给出可用清单（不静默）
const bad = schemaText('nope')
check('A8 未知 topic 返回可用清单', bad.indexOf('未知 topic') >= 0 && bad.indexOf('examples') >= 0, bad.slice(0, 60))

// A9 其它工具描述（description 或 usageGuide）指向 design_schema（可发现性）
const pointed = ['design_project', 'design_add'].filter(n => {
  const d = byName[n]
  if (!d) return false
  return (String(d.description || '') + String(d.usageGuide || '')).indexOf('design_schema') >= 0
})
check('A9 关键工具描述指向 design_schema', pointed.length === 2, `命中 ${pointed.join('/') || '无'}`)

console.log('\n=== tool-design 结构自省自检 ===')
let fail = 0
for (const r of results) { if (!r.ok) fail++; console.log((r.ok ? 'PASS  ' : 'FAIL  ') + r.name + '  —— ' + r.detail) }
console.log('\n结果：' + (results.length - fail) + '/' + results.length + ' 通过')
console.log('\n--- design_schema topic=all 摘要（前 12 行）---')
console.log(all.split('\n').slice(0, 12).join('\n'))
process.exit(fail ? 1 : 0)
