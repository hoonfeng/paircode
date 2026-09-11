// agent-teams 改造冒烟（Node 仿真宿主 ctx；不走 Go 服务）
// 用法: node _t4_smoke.js
// 覆盖: create(staged) → create_task(DAG) → status → approve → claim → update → status → delete
'use strict'
const fs = require('fs')
const path = require('path')
const os = require('os')

const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'agteams-smoke-'))
const tools = new Map()
const registered = []
const routed = []
const clientMethods = {}
const logs = []

function loggerFor(name) {
  const f = (level) => (...args) => logs.push(`[${level}][${name}] ${args.join(' ')}`)
  return { info: f('info'), warn: f('warn'), error: f('error'), debug: f('debug') }
}

const ctx = {
  fs: {
    mkdir: (p, rec) => fs.mkdirSync(p, { recursive: !!rec }),
    readFile: (p) => fs.readFileSync(p, 'utf8'),
    appendFile: (p, s) => fs.appendFileSync(p, s),
    writeFile: (p, s) => fs.writeFileSync(p, s),
    readdir: (p) => fs.readdirSync(p),
    stat: (p) => { const st = fs.statSync(p); return { isDir: st.isDirectory(), size: st.size, mtime: st.mtimeMs } },
    exists: (p) => fs.existsSync(p),
    rename: (a, b) => fs.renameSync(a, b),
    rm: (p, rec) => fs.rmSync(p, { recursive: !!rec, force: true }),
    roots: () => [tmp],
  },
  logger: loggerFor,
  timer: { interval: () => () => {}, setTimeout: (fn, ms) => setTimeout(fn, ms) },
  app: { workspaceRoot: tmp, settings: {} },
  http: { register: (method, route, fn) => routed.push({ method, route, fn }) },
  commands: { register: (cmd) => routed.push({ command: cmd.name }) },
  tools: { register: (t) => { tools.set(t.name, t); registered.push(t.name) } },
  registerSettings: () => {},
  getSettings: () => ({}),
  registerClientMethod: (name, fn) => { clientMethods[name] = fn },
  systemPrompt: { section: () => {} },
  activation: { declare: () => ({ ok: true }) },
  on: () => {},
  emit: () => {},
  effect: () => {},
  provide: () => () => {},
  agents: { ready: () => false, followup: () => ({ ok: true }), list: () => [] },
  llm: { models: () => [], current: () => ({}) },
  hostTool: { exec: () => '{}' },
}

const src = fs.readFileSync(path.join(__dirname, 'index.js'), 'utf8')
const fn = new Function('ctx', 'process', 'console', src)
const plugin = fn(ctx, process, console)
console.log('插件加载:', plugin && plugin.name, '| inject:', JSON.stringify(plugin.inject))
plugin.apply(ctx)

console.log('\n注册工具(%d): %s', registered.length, registered.join(', '))
const expect = ['agent_teams_create', 'agent_teams_edit_plan', 'agent_teams_approve', 'agent_teams_create_task', 'agent_teams_delete_task', 'agent_teams_claim_task', 'agent_teams_update_task', 'agent_teams_status', 'agent_teams_resume', 'agent_teams_delete']
for (const name of expect) {
  if (!tools.has(name)) { console.error('缺少工具: ' + name); process.exit(1) }
}
for (const name of ['agent_teams_add_member', 'agent_teams_remove_member', 'agent_teams_reassign_task', 'agent_teams_send_message']) {
  if (tools.has(name)) { console.error('不应存在（已去成员）: ' + name); process.exit(1) }
}
console.log('工具面核对通过（10 个新工具；无成员工具）')

const CONV = 'conv_main_1'
const call = (name, args) => tools.get(name).execute(Object.assign({ _convID: CONV, _wsRoot: tmp }, args || {}))

// 1) create staged
let out = call('agent_teams_create', { name: '冒烟团队', description: '验证单 Agent 多步执行', approval: 'required' })
console.log('\n[create]', out)
if (!/staged plan/.test(out)) { console.error('create 未进入 staged'); process.exit(1) }

// 2) 任务 DAG：t1 → t2（impl）→ t3（review 依赖 t2）
call('agent_teams_create_task', { subject: '需求分析', kind: 'requirements', objective: '拆解目标', acceptance: ['需求可测'] })
call('agent_teams_create_task', { subject: '实施改动', kind: 'implementation', objective: '落地改动', acceptance: ['实现完成'], inScope: ['src/'], verify: ['node --check index.js'] , dependencies: ['t1'] })
call('agent_teams_create_task', { subject: '审查实施', kind: 'review', objective: '审查改动', acceptance: ['结论有依据'], reviewedTaskId: 't2', dependencies: ['t2'] })

out = call('agent_teams_status', {})
console.log('\n[status before approve]\n' + out)
if (!/Ready to execute now \(1\)/.test(out) || !/t1: 需求分析/.test(out)) { console.error('status 就绪任务不平'); process.exit(1) }

// 3) 未批准不能 claim（团队仍 staged —— claim 不校验 phase，这里只验证依赖门）
// 4) approve
out = call('agent_teams_approve', { confirmation: '用户说：批准' })
console.log('\n[approve]', out)
if (!/approved: 3 tasks queued/.test(out) || !/Ready now: t1/.test(out)) { console.error('approve 输出异常'); process.exit(1) }

// 5) claim 未就绪任务应报错
try {
  call('agent_teams_claim_task', { task_id: 't2' })
  console.error('t2 依赖未满足却可领取'); process.exit(1)
} catch (e) {
  console.log('[claim t2 预期拒绝]', e.message)
}

// 6) claim t1 → update completed（requirements，需 verdict=pass）
out = call('agent_teams_claim_task', { task_id: 't1' })
console.log('\n[claim t1]', out)
const attempt = /attempt_id ([0-9a-f-]+)/.exec(out)[1]

// 缺 verdict 应被门禁拒绝
try {
  call('agent_teams_update_task', { task_id: 't1', status: 'completed', attempt_id: attempt })
  console.error('requirements 缺 verdict 却可完成'); process.exit(1)
} catch (e) {
  console.log('[update 缺 verdict 预期拒绝]', e.message)
}

out = call('agent_teams_update_task', { task_id: 't1', status: 'completed', attempt_id: attempt, verdict: 'pass', output: '需求已拆解' })
console.log('\n[update t1 → completed]', out)
if (!/Next ready: t2/.test(out)) { console.error('update 未提示下一就绪任务'); process.exit(1) }

// 8) t2：implementation 完成需 changedPaths + acceptanceResults + commandsRun
out = call('agent_teams_claim_task', { task_id: 't2' })
const attempt2 = /attempt_id ([0-9a-f-]+)/.exec(out)[1]
// stale attempt 拒绝（claim 后用失效 attempt_id）
try {
  call('agent_teams_update_task', { task_id: 't2', status: 'completed', attempt_id: 'stale-xxx' })
  console.error('stale attempt 未被拒绝'); process.exit(1)
} catch (e) {
  console.log('[stale attempt 预期拒绝]', e.message)
}
try {
  call('agent_teams_update_task', { task_id: 't2', status: 'completed', attempt_id: attempt2, changedPaths: ['src/a.js'], output: 'x' })
  console.error('implementation 缺证据却可完成'); process.exit(1)
} catch (e) {
  console.log('[update t2 缺证据预期拒绝]', e.message)
}
out = call('agent_teams_update_task', {
  task_id: 't2', status: 'completed', attempt_id: attempt2, output: '已实施',
  changedPaths: ['src/a.js'],
  acceptanceResults: [{ criterion: '实现完成', status: 'passed', evidence: 'diff' }],
  commandsRun: [{ command: 'node --check index.js', status: 'passed', exitCode: 0 }],
})
console.log('\n[update t2 → completed]', out)
if (!/Next ready: t3/.test(out)) { console.error('t2 完成后未提示 t3'); process.exit(1) }

// 9) t3：review needs_revision → 自动 repair+复审
out = call('agent_teams_claim_task', { task_id: 't3' })
const attempt3 = /attempt_id ([0-9a-f-]+)/.exec(out)[1]
out = call('agent_teams_update_task', {
  task_id: 't3', status: 'failed', attempt_id: attempt3,
  verdict: 'needs_revision',
  findings: [{ id: 'f1', severity: 'high', problem: '缺少边界处理', requiredFix: '补边界', file: 'src/a.js' }],
  output: '需修复',
})
console.log('\n[update t3 → failed(needs_revision)]', out)
if (!/repair-2/.test(out) || !/review-2/.test(out)) { console.error('质量循环未派生修复+复审'); process.exit(1) }

// 10) status 全景
out = call('agent_teams_status', {})
console.log('\n[status after loop]\n' + out)
if (!/repair-round-2/.test(out) || !/review-round-2/.test(out)) { console.error('status 缺修复/复审任务'); process.exit(1) }

// 11) 面板快照路由（GET）
const getRoute = routed.find((r) => r.method === 'GET' && r.route.indexOf('/api/agent-teams/teams') === 0)
const getResp = getRoute.fn({ path: '/api/agent-teams/teams' })
const snap = JSON.parse(getResp.body)
console.log('\n[panel snapshot] teams=%d tasks=%d ready=%s', snap.teams.length, snap.teams[0].tasks.length, JSON.stringify(snap.teams[0].readyTasks))
if (snap.teams[0].members) { console.error('快照仍含 members'); process.exit(1) }
if (!Array.isArray(snap.teams[0].readyTasks)) { console.error('快照缺 readyTasks'); process.exit(1) }
const TERMINAL = ['completed', 'failed', 'cancelled']
const archiveDir = path.join(tmp, '.agent-teams', 'archive')

// 12) 面板「删除团队」client method deleteTeam：取消未完成任务 + 归档（面板按钮链路）
if (typeof clientMethods.deleteTeam !== 'function') { console.error('缺少 client method: deleteTeam'); process.exit(1) }
const mainTeamId = snap.teams[0].teamId
const delRes = clientMethods.deleteTeam({ teamId: mainTeamId })
console.log('\n[panel deleteTeam]', JSON.stringify(delRes))
if (!delRes || delRes.archived !== true) { console.error('deleteTeam 返回异常'); process.exit(1) }
if (fs.existsSync(path.join(tmp, '.agent-teams', mainTeamId))) { console.error('deleteTeam 后活动目录仍在'); process.exit(1) }
const mainArchived = JSON.parse(fs.readFileSync(path.join(archiveDir, mainTeamId, 'team.json'), 'utf8'))
const stillOpen = mainArchived.tasks.filter((t) => TERMINAL.indexOf(t.status) < 0)
if (stillOpen.length > 0) { console.error('归档记录含未取消任务: ' + stillOpen.map((t) => t.id).join(',')); process.exit(1) }
console.log('deleteTeam OK：未完成任务已取消并归档（tasks=%d）', mainArchived.tasks.length)

// 13) HTTP delete 路由（与面板同语义；sessionId=队长的会话）
const postRoute = routed.find((r) => r.method === 'POST' && r.route.indexOf('/api/agent-teams/teams') === 0)
call('agent_teams_create', { name: '路由删除团队', approval: 'automatic' })
const routeTeamId = '路由删除团队'
const postResp = postRoute.fn({ path: '/api/agent-teams/teams/' + encodeURIComponent(routeTeamId) + '/delete', body: JSON.stringify({ sessionId: CONV, wsRoot: tmp }) })
console.log('\n[HTTP delete]', postResp.status, postResp.body)
if (postResp.status !== 200 || JSON.parse(postResp.body).archived !== true) { console.error('HTTP delete 路由失败'); process.exit(1) }
if (!fs.existsSync(path.join(archiveDir, routeTeamId, 'team.json'))) { console.error('HTTP delete 未归档'); process.exit(1) }

// 14) 工具 agent_teams_delete（队长）：取消未完成任务 + 归档（原链路不回归）
call('agent_teams_create', { name: '工具删除团队', approval: 'automatic' })
out = call('agent_teams_delete', {})
console.log('\n[delete]', out)
if (!fs.existsSync(archiveDir) || fs.readdirSync(archiveDir).length < 3) { console.error('归档目录不完整'); process.exit(1) }
console.log('归档:', fs.readdirSync(archiveDir).join(', '))

console.log('\n✅ agent-teams 冒烟全部通过（toolset=%d, clientMethods=%s）', registered.length, Object.keys(clientMethods).join(','))
fs.rmSync(tmp, { recursive: true, force: true })
