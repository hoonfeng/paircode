// 验证分段续跑配置化（★ 双闸门：stepBudget 步数 / toolCallBudget 工具调用 /
// maxToolBudgetSegments 段数上限）在运行实例上的配置注册与保存链路：
// GET /api/settings（schemas 渲染入口）→ PUT 保存（pluginSettings.agentloop）
// → 读回 + 落盘检查。
//
// ⚠️ 会写 config/settings.json（PUT 保存到 pluginSettings.agentloop）——运行前先备份：
//     cp config/settings.json _temp/settings.json.bak
//     验证后恢复：cp _temp/settings.json.bak config/settings.json
//
// 用法：WEB_PORT=9098 ./companion_test.exe &   # 独立二进制 + 非默认端口（勿动 9090）
//       node scripts/verify-budget-cfg.mjs 9098
import fs from 'node:fs'

const port = process.argv[2] || '9098'
const base = `http://127.0.0.1:${port}`
const results = []
const ok = (name, cond, detail = '') => {
  results.push({ name, pass: !!cond, detail })
  console.log(`${cond ? 'PASS' : 'FAIL'}  ${name}${detail ? '  ' + detail : ''}`)
}

const getJSON = async (p) => (await fetch(base + p)).json()

console.log(`node ${process.version}  base=${base}`)
const s0 = await getJSON('/api/settings')
const sch = (s0.schemas || []).find((s) => s.key === 'agentloop')
ok('agentloop 配置段已注册（schema 存在）', !!sch, sch ? `fields=${sch.fields.length}` : '')
const byName = Object.fromEntries(((sch && sch.fields) || []).map((f) => [f.name, f]))
const sb = byName.stepBudget
const tb = byName.toolCallBudget
const ms = byName.maxToolBudgetSegments
ok('schema 含 stepBudget 字段（★ 双闸门·步数）', !!sb, sb ? `label=${sb.label} type=${sb.type} default=${sb.default}` : '')
ok('schema 含 toolCallBudget 字段', !!tb, tb ? `label=${tb.label} type=${tb.type} default=${tb.default}` : '')
ok('schema 含 maxToolBudgetSegments 字段', !!ms, ms ? `label=${ms.label} type=${ms.type} default=${ms.default}` : '')
ok('默认值 = 120 步 / 120 次 / 20 段（设置面板显示）',
  sb && tb && ms && Number(sb.default) === 120 && Number(tb.default) === 120 && Number(ms.default) === 20,
  sb && tb && ms ? `${sb.default} / ${tb.default} / ${ms.default}` : '')
ok('字段非 binding（存插件命名空间，非 AppSettings 内置字段）',
  sb && !sb.binding && tb && !tb.binding && ms && !ms.binding,
  `binding=${JSON.stringify([sb && sb.binding, tb && tb.binding, ms && ms.binding])}`)
ok('hint 说明语义（0=默认、负数=不限 / 防失控）',
  !!(sb && /一步 = 一次模型调用/.test(sb.hint) && /0=默认 120/.test(sb.hint) && /不限/.test(sb.hint)) &&
  !!(tb && /0=默认 120/.test(tb.hint) && /不限/.test(tb.hint)) &&
  !!(ms && /0=默认 20/.test(ms.hint)), '')

// ── 保存：配置项可调（经插件命名空间落盘）──
const putRes = await fetch(base + '/api/settings', {
  method: 'PUT',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ pluginSettings: { agentloop: { stepBudget: 5, toolCallBudget: 3, maxToolBudgetSegments: 2 } } }),
})
ok('PUT /api/settings 保存成功', putRes.ok, `status=${putRes.status}`)

const s1 = await getJSON('/api/settings')
const saved = (s1.settings.pluginSettings || {}).agentloop || {}
ok('读回保存值（stepBudget=5 / toolCallBudget=3 / maxToolBudgetSegments=2）',
  Number(saved.stepBudget) === 5 && Number(saved.toolCallBudget) === 3 && Number(saved.maxToolBudgetSegments) === 2,
  JSON.stringify(saved))

const disk = JSON.parse(fs.readFileSync('config/settings.json', 'utf8'))
const diskAgent = (disk.pluginSettings || {}).agentloop || {}
ok('已落盘 config/settings.json',
  Number(diskAgent.stepBudget) === 5 && Number(diskAgent.toolCallBudget) === 3 && Number(diskAgent.maxToolBudgetSegments) === 2,
  JSON.stringify(diskAgent))
ok('未污染 AppSettings 顶层（无 stepBudget/toolCallBudget 内置键）',
  !('stepBudget' in disk) && !('toolCallBudget' in disk) && !('maxToolBudgetSegments' in disk),
  `顶层键数=${Object.keys(disk).length}`)

const failed = results.filter((r) => !r.pass)
console.log(`\n══ 结果：${results.length - failed.length}/${results.length} PASS ══`)
if (failed.length) {
  console.log(failed.map((f) => ' - ' + f.name + ' ' + f.detail).join('\n'))
  process.exit(1)
}
