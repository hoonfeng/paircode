// 验证「最大迭代数」配置项已彻底移除（2026-09-12）：
//  ① 设置 schema 不再暴露 maxIterations 字段（设置面板自动消失）；
//  ② 运行实例顶层 settings 与 pluginSettings.agentloop 均无该键；
//  ③ 磁盘 config/settings.json 无残留（顶层 AppSettings + 插件命名空间）；
//  ④ 可选（--chat）：真实对话跑通一轮 agent 循环，证明移除迭代表后循环仍能
//     正常启动（宿主按工具预算派生 iterationLimit）并自然终止。
//
// ⚠️ --chat 会向当前工作区写入一条测试会话（脚本会尝试 DELETE 清理）。
// ⚠️ 只读接口，不写 settings.json；但若先前跑过预设 apply 类操作，请从备份恢复。
//
// 用法：WEB_PORT=9098 ./companion_test.exe &
//       node scripts/verify-no-maxiter.mjs 9098            # 仅配置面
//       node scripts/verify-no-maxiter.mjs 9098 --chat     # 含真实对话
import fs from 'node:fs'

const port = process.argv[2] || '9098'
const wantChat = process.argv.includes('--chat')
const base = `http://127.0.0.1:${port}`
const wsRoot = process.cwd().replace(/\\/g, '\\')
const results = []
const ok = (name, cond, detail = '') => {
  results.push({ name, pass: !!cond, detail })
  console.log(`${cond ? 'PASS' : 'FAIL'}  ${name}${detail ? '  ' + detail : ''}`)
}
const skip = (name, reason) => {
  results.push({ name, pass: true, skipped: true, detail: reason })
  console.log(`SKIP  ${name}  ${reason}`)
}
const j = async (p, o) => {
  const r = await fetch(base + p, o)
  const t = await r.text()
  try {
    return { status: r.status, body: JSON.parse(t) }
  } catch {
    return { status: r.status, body: t }
  }
}
const H = { 'Content-Type': 'application/json' }
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

console.log(`node ${process.version}  base=${base}  chat=${wantChat}`)

// ── ① schema：字段已移除、预算字段仍在 ──
const s0 = await j('/api/settings')
const sch = (s0.body.schemas || []).find((s) => s.key === 'agentloop')
ok('agentloop 配置段已注册', !!sch, sch ? `fields=${sch.fields.length}` : '')
const names = ((sch && sch.fields) || []).map((f) => f.name)
ok('schema 不再含 maxIterations 字段', !!sch && !names.includes('maxIterations'), `fields=${names.join(',')}`)
ok('回归保护：toolCallBudget / maxToolBudgetSegments 仍在', names.includes('toolCallBudget') && names.includes('maxToolBudgetSegments'))

// ── ② 运行实例 settings ──
const s1 = s0.body.settings || {}
ok('顶层 settings 无 maxIterations 键', !('maxIterations' in s1), `顶层键数=${Object.keys(s1).length}`)
const plug = (s1.pluginSettings || {}).agentloop || {}
ok('pluginSettings.agentloop 无 maxIterations 键', !('maxIterations' in plug), JSON.stringify(plug))

// ── ③ 磁盘 settings.json ──
const disk = JSON.parse(fs.readFileSync('config/settings.json', 'utf8'))
ok('磁盘顶层无 maxIterations', !('maxIterations' in disk), `顶层键数=${Object.keys(disk).length}`)
const diskPlug = (disk.pluginSettings || {}).agentloop || {}
ok('磁盘 pluginSettings.agentloop 无 maxIterations', !('maxIterations' in diskPlug))

// ── ④ 真实对话（可选）：循环启动 + 自然终止 ──
if (wantChat) {
  const conv = 'conv_e2e_noiter_verify'
  const c = await j('/api/conversations', { method: 'POST', headers: H, body: JSON.stringify({ id: conv, title: 'noiter-verify', workspaceRoot: wsRoot }) })
  ok('创建测试会话', c.status === 200 && c.body && c.body.ok !== false, JSON.stringify(c.body).slice(0, 120))
  const s = await j('/api/chat/send', { method: 'POST', headers: H, body: JSON.stringify({ message: '请只回复 PONG 四个字母，不要调用任何工具。', convId: conv, workspaceRoot: wsRoot }) })
  ok('chat/send 已受理（循环启动）', s.status === 200 && s.body && s.body.ok, JSON.stringify(s.body).slice(0, 160))

  let msgCount = 0
  for (let i = 0; i < 20; i++) {
    await sleep(3000)
    const d = await j('/api/conversations/' + conv)
    msgCount = (d.body && d.body.msgCount) || 0
    if (msgCount >= 2) break
  }
  ok('对话完成（助手已回复，循环自然终止）', msgCount >= 2, `msgCount=${msgCount}`)

  const del = await j('/api/conversations/' + conv, { method: 'DELETE' })
  ok('测试会话已清理', del.status === 200, `status=${del.status}`)
} else {
  skip('真实对话端到端（--chat）', '未开启；如需验证循环启动/终止请加 --chat')
}

const failed = results.filter((r) => !r.pass)
const skipped = results.filter((r) => r.skipped).length
console.log(`\n══ 结果：${results.length - failed.length - skipped}/${results.length - skipped} PASS${skipped ? `（${skipped} SKIP）` : ''} ══`)
if (failed.length) {
  console.log(failed.map((f) => ' - ' + f.name + ' ' + f.detail).join('\n'))
  process.exit(1)
}
