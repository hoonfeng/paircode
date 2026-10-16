// ═══════════════════════════════════════════════════════════════
// tool-flow 插件自测（mock ctx：真实文件系统 + 真实 bash 执行）—— 用法：node smoke-test.js
//
// 目的：在不起实例的前提下，用与宿主同语义的 ctx 桩驱动插件的全部工具/
//       接口/钩子/命令，验证「三层解析 → 校验 → 启动 → 门禁 → 推进 → 证据
//       → 导出导入 → 绑定矩阵 → 提示注入 → 拦截」整条链路。
// 纪律：mock ctx.fs.writeFile **不**自动建父目录（与真实一致）；
//       mock ctx.install.writeFile 自动建父目录（与真实一致）。
// ═══════════════════════════════════════════════════════════════
const fs = require('fs')
const path = require('path')
const cp = require('child_process')

const BASE = path.join(__dirname, '.smoke')
const WS = path.join(BASE, 'ws')
const INSTALL = path.join(BASE, 'install')
const PLUGIN_SRC = fs.existsSync(path.join(__dirname, 'index.js')) ? __dirname : path.join(__dirname, '..', '.pair', 'plugins', 'tool-flow')

let pass = 0, fail = 0
const fails = []
function check(name, cond, detail) {
  if (cond) { pass++; console.log('  PASS  ' + name) }
  else { fail++; fails.push(name + ' :: ' + String(detail || '').slice(0, 300)); console.log('  FAIL  ' + name + '  ' + String(detail || '').slice(0, 300)) }
}

// ── 目录准备 ─────────────────────────────────────────────────
function rmrf(p) { try { fs.rmSync(p, { recursive: true, force: true }) } catch (e) { } }
rmrf(BASE + '/ws'); rmrf(BASE + '/install')
fs.mkdirSync(WS, { recursive: true })
fs.mkdirSync(path.join(INSTALL, '.pair', 'plugins', 'tool-flow', 'assets', 'flows'), { recursive: true })
// 预设随包发布（从插件真源拷贝）
const presetSrc = path.join(PLUGIN_SRC, 'assets', 'flows')
for (const f of fs.readdirSync(presetSrc)) {
  fs.copyFileSync(path.join(presetSrc, f), path.join(INSTALL, '.pair', 'plugins', 'tool-flow', 'assets', 'flows', f))
}
// 工作区特征文件（项目类型识别：go）；git 仓库（gitHead 基线）
fs.writeFileSync(path.join(WS, 'go.mod'), 'module demo\n\ngo 1.22\n')
cp.execSync('git init -q && git add -A && git -c user.email=t@t -c user.name=t commit -qm init', { cwd: WS, shell: 'bash' })

// ── ctx 桩 ───────────────────────────────────────────────────
function resolveWs(p) {
  const abs = path.resolve(WS, String(p))
  if (abs !== WS && !abs.startsWith(WS + path.sep)) throw new Error('路径超出工作区范围（root: ' + WS + '）')
  return abs
}
function resolveInstall(p) {
  const abs = path.resolve(INSTALL, String(p))
  const dataRoot = path.join(INSTALL, '.pair')
  if (abs !== dataRoot && !abs.startsWith(dataRoot + path.sep)) throw new Error('路径超出安装数据区')
  if (abs === dataRoot) throw new Error('不允许直接操作安装数据区根目录')
  return abs
}
const fsSvc = {
  readFile: (p) => fs.readFileSync(resolveWs(p), 'utf8'),
  writeFile: (p, c) => { fs.writeFileSync(resolveWs(p), String(c)) },        // 不建父目录（同真实）
  appendFile: (p, c) => { fs.appendFileSync(resolveWs(p), String(c)) },
  exists: (p) => { try { return fs.existsSync(resolveWs(p)) } catch (e) { return false } },
  readdir: (p) => fs.readdirSync(resolveWs(p)),
  stat: (p) => { const s = fs.statSync(resolveWs(p)); return { name: path.basename(p), size: s.size, isDir: s.isDirectory(), mtime: s.mtime.toISOString() } },
  mkdir: (p, r) => { fs.mkdirSync(resolveWs(p), { recursive: r !== false }) },
  rm: (p, r) => { fs.rmSync(resolveWs(p), { recursive: !!r, force: true }) },
}
const installSvc = {
  root: () => path.join(INSTALL, '.pair'),
  exists: (p) => { try { return fs.existsSync(resolveInstall(p)) } catch (e) { return false } },
  readFile: (p) => fs.readFileSync(resolveInstall(p), 'utf8'),
  writeFile: (p, c) => { const a = resolveInstall(p); fs.mkdirSync(path.dirname(a), { recursive: true }); fs.writeFileSync(a, String(c)) },
  appendFile: (p, c) => { const a = resolveInstall(p); fs.mkdirSync(path.dirname(a), { recursive: true }); fs.appendFileSync(a, String(c)) },
  readdir: (p) => fs.readdirSync(resolveInstall(p)),
  list: (p) => {
    const dir = resolveInstall(p)
    return fs.readdirSync(dir).map((n) => {
      const abs = path.join(dir, n); const s = fs.statSync(abs)
      return { name: n, path: path.relative(path.join(INSTALL, '.pair'), abs).replace(/\\/g, '/'), isDir: s.isDirectory(), size: s.size, mtime: s.mtime.toISOString() }
    })
  },
  stat: (p) => { const s = fs.statSync(resolveInstall(p)); return { name: path.basename(p), size: s.size, isDir: s.isDirectory(), mtime: s.mtime.toISOString() } },
  mkdir: (p, r) => { fs.mkdirSync(resolveInstall(p), { recursive: r !== false }) },
  rm: (p, r) => { fs.rmSync(resolveInstall(p), { recursive: !!r, force: true }) },
  rename: (a, b) => { fs.renameSync(resolveInstall(a), resolveInstall(b)) },
}

const registered = {}
const httpRoutes = {}
const commands = {}
const hooks = {}
const promptSections = []
const promptVars = {}
const settingsRegistry = { }
const rpcs = {}
const logs = []
let settingsStore = { gateStrength: 'enforce', autoRecommend: true, trackModifiedFiles: true }

const ctx = {
  logger: () => ({ log: () => { }, info: (m) => logs.push(m), warn: (m) => logs.push('WARN ' + m), debug: () => { }, error: (m) => logs.push('ERR ' + m) }),
  app: { workspaceRoot: WS, folders: [WS] },
  fs: fsSvc,
  install: installSvc,
  tools: { register: (t) => { registered[t.name] = t }, list: () => Object.keys(registered) },
  http: { register: (m, p, fn) => { httpRoutes[m + ' ' + p] = fn } },
  commands: { register: (c) => { commands[c.name] = c } },
  hooks: { register: (e, fn) => { hooks[e] = fn } },
  systemPrompt: { section: (s) => promptSections.push(s), variable: (v) => { promptVars[v.name] = v } },
  registerSettings: (s) => { settingsRegistry.key = s },
  getSettings: () => settingsStore,
  setSettings: (k, o) => { settingsStore = o || {} },
  registerClientMethod: (m, fn) => { rpcs[m] = fn },
  emit: () => { },
  bash: {
    exec: (cmd, cwd, timeout) => {
      try {
        const out = cp.execSync(String(cmd), { cwd: cwd ? resolveWs(cwd) : WS, encoding: 'utf8', timeout: 20000, shell: 'bash' })
        return { output: out }
      } catch (e) {
        return { output: String(e.stdout || '') + String(e.stderr || ''), error: 'exit ' + (e.status === undefined ? 1 : e.status) }
      }
    },
  },
}

// ── 装载插件 ─────────────────────────────────────────────────
const code = fs.readFileSync(path.join(PLUGIN_SRC, 'index.js'), 'utf8')
const def = (new Function(code))()
console.log('=== 装载 tool-flow ===')
console.log('  name=' + def.name + ' inject=[' + def.inject.join(',') + ']')
check('插件对象形态正确', def && def.name === 'tool-flow' && typeof def.apply === 'function')
def.apply(ctx, {})
check('工具注册数量 = 17', Object.keys(registered).length === 17, Object.keys(registered).join(','))
check('只读接口已注册', !!httpRoutes['GET /api/flow/state'], Object.keys(httpRoutes).join(','))
check('RPC flow_action 已注册', typeof rpcs['flow_action'] === 'function')
check('命令面已注册（flow/flow-board/flow-import）', !!commands.flow && !!commands['flow-board'] && !!commands['flow-import'])
check('钩子已注册（PreToolUse/PostToolUse）', !!hooks.PreToolUse && !!hooks.PostToolUse)
check('systemPrompt section 已注册', promptSections.length === 1 && promptSections[0].text === '{{paircode_flow_state}}')
check('systemPrompt variable 已注册', typeof (promptVars['paircode_flow_state'] || {}).provider === 'function')
check('settings 已注册', settingsRegistry.key && settingsRegistry.key.key === 'tool-flow')

const T = (n) => registered[n]
function txt(r) { return (r && r.text) || '' }

console.log('\n=== ① 流程库与识别（预设层经 ctx.install 读）===')
let r = T('flow_list').execute({})
check('flow_list 列出 8 个预设', /共 8 个/.test(txt(r)), txt(r).slice(0, 200))
check('flow_list 标注只读层', /preset\(只读\)/.test(txt(r)))
r = T('flow_recommend').execute({})
check('项目类型识别命中 go', /go/.test(txt(r)) && /go-backend-feature/.test(txt(r)), txt(r))
r = T('flow_show').execute({ id: 'go-backend-feature' })
check('flow_show 返回 5 阶段 + 校验通过', /5\./.test(txt(r)) && /\[通过\]/.test(txt(r)), txt(r).slice(0, 200))
r = T('flow_validate').execute({ id: 'go-backend-feature' })
check('flow_validate 通过', /\[通过\]/.test(txt(r)))
r = T('flow_validate').execute({ json: '{"id":"bad","stages":[]}' })
check('flow_validate 拒绝非法定义', /\[未通过\]/.test(txt(r)) && /stages/.test(txt(r)), txt(r))

console.log('\n=== ② 启动实例（任务写入宿主任务真相 + git 基线）===')
r = T('flow_start').execute({ flowId: 'go-backend-feature', convId: 'conv_test1' })
check('flow_start 成功', /流程已启动/.test(txt(r)), txt(r))
const runId = (r.run && r.run.runId) || ''
check('实例落盘 .pair/flow-runs/<runId>.json', fs.existsSync(path.join(WS, '.pair', 'flow-runs', runId + '.json')), runId)
check('git 基线已记录', !!(r.run && r.run.gitHead), JSON.stringify(r.run && r.run.gitHead))
const taskFiles = fs.readdirSync(path.join(WS, '.pair', 'tasks')).filter((f) => f.endsWith('.json'))
check('阶段任务已写入 .pair/tasks（3 条）', taskFiles.length === 3, 'tasks=' + taskFiles.length)
const t0 = JSON.parse(fs.readFileSync(path.join(WS, '.pair', 'tasks', taskFiles[0]), 'utf8'))
check('任务字段符合宿主契约（subject/status/convId）', !!t0.subject && t0.status === 'pending' && t0.convId === 'conv_test1', JSON.stringify(t0))
r = T('flow_start').execute({ flowId: 'go-backend-feature', convId: 'conv_test1' })
check('重复启动被拒（提示已有实例）', /已有进行中的流程实例/.test(txt(r)), txt(r))

console.log('\n=== ③ 门禁与推进 ===')
r = T('flow_advance').execute({ convId: 'conv_test1' })
check('未过门禁时拒绝推进', /不能推进/.test(txt(r)) && /g-recon-1/.test(txt(r)), txt(r).slice(0, 260))
r = T('flow_advance').execute({ convId: 'conv_test1', gateId: 'g-recon-1' })
check('check 门禁无证据 → pending 不通过', /未通过/.test(txt(r)) && /待提供证据/.test(txt(r)), txt(r).slice(0, 200))
check('未通过时回传 gate 记录', !!(r.result && r.result.gate && r.result.gate.id === 'g-recon-1' && r.result.gate.status === 'pending'), JSON.stringify((r.result && r.result.gate) || {}).slice(0, 200))
r = T('flow_advance').execute({ convId: 'conv_test1', gateId: 'g-recon-1', evidence: '已列出 internal/agent 受影响文件 3 个' })
check('阶段唯一必过门禁通过 → 自动推进到阶段 2', /进入阶段 2/.test(txt(r)), txt(r).slice(0, 220))
check('推进返回值带回门禁记录（PASS）', !!(r.result && r.result.gate && r.result.gate.status === 'pass'), JSON.stringify((r.result && r.result.gate) || {}).slice(0, 260))
check('可选门禁（g-recon-2 未标 required）不阻塞推进', r.result && r.result.advanced === true)
r = T('flow_advance').execute({ convId: 'conv_test1' })
check('新阶段有未过门禁 → 拒绝推进并列出', /不能推进/.test(txt(r)) && /g-impl-1/.test(txt(r)), txt(r).slice(0, 220))
const run1 = JSON.parse(fs.readFileSync(path.join(WS, '.pair', 'flow-runs', runId + '.json'), 'utf8'))
check('run 落盘：阶段1 completed + 阶段2 in_progress', run1.stages[0].status === 'completed' && run1.stages[1].status === 'in_progress', JSON.stringify(run1.stages.map((s) => s.status)))
check('阶段1 的 gateResults 留档（1 条必过门禁 + 证据文件）', (run1.stages[0].gateResults || []).length === 1 && !!run1.stages[0].gateResults[0].evidenceFile, JSON.stringify(run1.stages[0].gateResults || []).slice(0, 300))
check('阶段1 任务已标 completed', (() => { const ids = run1.stages[0].taskIds; return ids.every((id) => JSON.parse(fs.readFileSync(path.join(WS, '.pair', 'tasks', id + '.json'), 'utf8')).status === 'completed') })())
check('阶段2 任务已新建', (run1.stages[1].taskIds || []).length === 2, JSON.stringify(run1.stages[1].taskIds))

console.log('\n=== ④ command 门禁（插件代跑 + 输出留证）===')
// 用自定义流程验证 command/artifact/approval 三种门禁
const customFlow = {
  schemaVersion: 1, id: 'gate-lab', title: '门禁实验室', version: 1,
  stages: [
    {
      id: 's1', title: '命令门禁组', goal: '验证 command/artifact/approval',
      tasks: ['跑命令门禁'],
      gates: [
        { id: 'g-cmd-ok', kind: 'command', text: 'echo 门禁通过', cmd: 'echo gate-ok', required: true, blocking: true },
        { id: 'g-cmd-bad', kind: 'command', text: '必失败命令', cmd: 'exit 3', required: false, onFail: 'rollback' },
        { id: 'g-art', kind: 'artifact', text: '产物存在', artifact: 'go.mod', required: true },
        { id: 'g-appr', kind: 'approval', text: '用户批准', required: true, blocking: true },
      ],
    },
  ],
  board: { kind: 'kanban' },
}
r = T('flow_define').execute({ json: JSON.stringify(customFlow), scope: 'project' })
check('flow_define 创建项目层流程', /已创建流程 gate-lab/.test(txt(r)) && fs.existsSync(path.join(WS, '.pair', 'flows', 'gate-lab.json')), txt(r))
r = T('flow_start').execute({ flowId: 'gate-lab', convId: 'conv_gate', force: true })
const runG = (r.run && r.run.runId) || ''
check('gate-lab 实例启动', !!runG, txt(r))
r = T('flow_advance').execute({ runId: runG, gateId: 'g-cmd-ok' })
check('command 门禁代跑通过', !!(r.result && r.result.gate && r.result.gate.status === 'pass'), JSON.stringify((r.result && r.result.gate) || {}).slice(0, 260))
check('证据 .log 落盘', fs.existsSync(path.join(WS, '.pair', 'flow-runs', 'evidence', runG, 'g-cmd-ok.log')))
check('.log 含命令输出', fs.readFileSync(path.join(WS, '.pair', 'flow-runs', 'evidence', runG, 'g-cmd-ok.log'), 'utf8').indexOf('gate-ok') >= 0)
r = T('flow_advance').execute({ runId: runG, gateId: 'g-cmd-bad' })
check('失败命令门禁 → fail', /未通过/.test(txt(r)) && /命令失败/.test(txt(r)), txt(r).slice(0, 200))
check('失败门禁（onFail=rollback）附结构化回滚建议', !!(r.result && r.result.rollback && r.result.rollback.failedGate === 'g-cmd-bad' && (r.result.rollback.rollbackOptions || []).length >= 2), JSON.stringify((r.result && r.result.rollback) || {}).slice(0, 320))
check('回滚建议含证据文件与失败原因', !!(r.result.rollback.evidenceFile && r.result.rollback.reason), JSON.stringify(r.result.rollback).slice(0, 240))
r = T('flow_advance').execute({ runId: runG, gateId: 'g-art' })
check('artifact 门禁校验产物存在 → pass', !!(r.result && r.result.gate && r.result.gate.status === 'pass'), JSON.stringify((r.result && r.result.gate) || {}).slice(0, 260))
r = T('flow_advance').execute({ runId: runG, gateId: 'g-appr' })
check('approval 门禁不得自动通过 → pending', !!(r.result && r.result.gate && r.result.gate.status === 'pending'), JSON.stringify((r.result && r.result.gate) || {}).slice(0, 260))
r = T('flow_advance').execute({ runId: runG, gateId: 'g-appr', approve: true, note: '对话确认' })
check('approval 显式批准 → approved 且自动完成单阶段流程', !!(r.result && r.result.gate && r.result.gate.status === 'approved') && /全部阶段已完成/.test(txt(r)), JSON.stringify((r.result && r.result.gate) || {}).slice(0, 260) + ' | ' + txt(r).slice(0, 120))
const runGj = JSON.parse(fs.readFileSync(path.join(WS, '.pair', 'flow-runs', runG + '.json'), 'utf8'))
const appr = (runGj.stages[0].gateResults || []).filter((x) => x.id === 'g-appr')[0]
check('审批留痕 approvedBy=user + note', appr && appr.status === 'approved' && appr.by === 'user' && appr.note === '对话确认', JSON.stringify(appr))
r = T('flow_advance').execute({ runId: runG })
check('已结束实例再推进 → 明确拒绝', /已结束/.test(txt(r)), txt(r).slice(0, 200))

console.log('\n=== ⑤ 流程管理面（clone/define 乐观锁/sync/delete/export/import）===')
r = T('flow_clone').execute({ id: 'go-backend-feature', newId: 'go-custom', scope: 'project' })
check('flow_clone 生成项目层副本', /已从 preset 层/.test(txt(r)) && fs.existsSync(path.join(WS, '.pair', 'flows', 'go-custom.json')), txt(r))
r = T('flow_define').execute({ json: JSON.stringify(Object.assign({}, customFlow, { version: 2 })), scope: 'project', expectedVersion: 1 })
check('乐观锁：版本一致则更新', /已更新流程 gate-lab/.test(txt(r)), txt(r))
r = T('flow_define').execute({ json: JSON.stringify(Object.assign({}, customFlow, { version: 3 })), scope: 'project', expectedVersion: 1 })
check('乐观锁：版本冲突则拒绝', /版本冲突/.test(txt(r)), txt(r))
r = T('flow_define').execute({ json: JSON.stringify(customFlow), scope: 'preset' })
check('preset 层只读拒写', /scope 必须是 project \| global/.test(txt(r)), txt(r))
r = T('flow_sync').execute({ id: 'go-backend-feature', from: 'preset', to: 'project' })
check('flow_sync 默认不回冲（项目层已有同名？否→新建）', /已同步/.test(txt(r)), txt(r))
r = T('flow_sync').execute({ id: 'go-backend-feature', from: 'preset', to: 'project' })
check('flow_sync 二次：目标已存在且无 force → 不覆盖', /不回冲|未写入/.test(txt(r)), txt(r).slice(0, 200))
r = T('flow_export').execute({ id: 'go-backend-feature' })
check('flow_export 生成目录 bundle', /已导出目录 bundle/.test(txt(r)), txt(r))
const bdir = path.join(WS, '.pair', 'flow-exports', 'flow-go-backend-feature')
check('bundle 含 flow.json / README.md / gates / templates',
  fs.existsSync(path.join(bdir, 'flow.json')) && fs.existsSync(path.join(bdir, 'README.md')) &&
  fs.existsSync(path.join(bdir, 'gates')) && fs.existsSync(path.join(bdir, 'templates')),
  fs.readdirSync(bdir).join(','))
check('bundle README 含阶段与过关说明', /如何过关/.test(fs.readFileSync(path.join(bdir, 'README.md'), 'utf8')))
r = T('flow_import').execute({ dir: '.pair/flow-exports/flow-go-backend-feature', scope: 'project', dryRun: true })
check('flow_import dryRun 只报告不落盘', /dryRun/.test(txt(r)), txt(r).slice(0, 200))
r = T('flow_import').execute({ dir: '.pair/flow-exports/flow-go-backend-feature', scope: 'global' })
check('flow_import 导入全局层（ctx.install → flowlib）', /已导入流程 go-backend-feature → global/.test(txt(r)) && fs.existsSync(path.join(INSTALL, '.pair', 'flowlib', 'go-backend-feature.json')), txt(r))
r = T('flow_import').execute({ dir: '.pair/flow-exports/flow-go-backend-feature', scope: 'global' })
check('flow_import 同名冲突默认 skip（不静默覆盖）', /按 skip 跳过/.test(txt(r)), txt(r).slice(0, 200))
r = T('flow_import').execute({ dir: '.pair/flow-exports/flow-go-backend-feature', scope: 'global', onConflict: 'rename' })
check('flow_import onConflict=rename 生效', /已重命名/.test(txt(r)), txt(r).slice(0, 220))
r = T('flow_import').execute({ json: JSON.stringify({ schemaVersion: 9, id: 'x', stages: [{ id: 's', title: 's', tasks: [], gates: [] }] }), scope: 'project' })
check('schemaVersion 不符 → 拒绝（不猜测降级）', /schemaVersion 不支持/.test(txt(r)), txt(r))
r = T('flow_list').execute({})
check('flow_list 现含 global/project 层条目', /global/.test(txt(r)) && /project/.test(txt(r)), txt(r).slice(0, 300))
r = T('flow_delete').execute({ id: 'go-custom', scope: 'project' })
check('flow_delete 删除项目层流程', /已删除 project 层流程 go-custom/.test(txt(r)), txt(r))
r = T('flow_delete').execute({ id: 'go-backend-feature', scope: 'preset' })
check('flow_delete preset 层被拒', /preset 层只读/.test(txt(r)), txt(r))

console.log('\n=== ⑥ 项目绑定矩阵 ===')
r = T('flow_attach').execute({ flowId: 'go-backend-feature', note: '主项目' })
check('flow_attach 绑定成功', /已绑定/.test(txt(r)), txt(r))
r = T('flow_bindings').execute({})
check('flow_bindings 显示绑定 + 实例阶段', /go-backend-feature/.test(txt(r)) && /阶段/.test(txt(r)), txt(r))
check('绑定文件 _binding.json 落盘', fs.existsSync(path.join(WS, '.pair', 'flows', '_binding.json')))
r = T('flow_detach').execute({})
check('flow_detach 解绑', /已解除绑定/.test(txt(r)), txt(r))

console.log('\n=== ⑦ 只读接口 + RPC + 提示注入 + 命令 ===')
r = T('flow_status').execute({ convId: 'conv_test1' })
check('flow_status 展示阶段与门禁标记', /进度：阶段/.test(txt(r)) && /PASS/.test(txt(r)), txt(r).slice(0, 260))
const snap = JSON.parse(httpRoutes['GET /api/flow/state']({ query: 'convId=conv_test1' }).body)
check('接口 /api/flow/state 返回快照', snap.ok === true && !!snap.run && Array.isArray(snap.stages), JSON.stringify(snap.run || {}).slice(0, 200))
check('快照含门禁状态与证据文件', snap.stages[0].gates.length === 2 && !!snap.stages[0].gates[0].evidenceFile)
check('快照含三层流程库', Array.isArray(snap.library) && snap.library.length >= 8)
check('快照含项目矩阵', Array.isArray(snap.matrix) && snap.matrix.length >= 1, JSON.stringify(snap.matrix).slice(0, 200))
check('快照含推荐（autoRecommend 开）', !!snap.recommend && snap.recommend.flowId === 'go-backend-feature')
const rpcStart = rpcs['flow_action']({ op: 'start', flowId: 'generic', convId: 'conv_rpc' })
check('RPC flow_action start（看板启动通道）', rpcStart.ok === true, JSON.stringify(rpcStart))
const rpcAppr = rpcs['flow_action']({ op: 'approve', gateId: 'g-x' })
check('RPC approve 未带 approve:true 被拒', rpcAppr.ok === false, JSON.stringify(rpcAppr))
const promptText = promptVars['paircode_flow_state'].provider()
check('提示注入含流程/阶段/未过门禁', /\[流程\]/.test(promptText) && /阶段/.test(promptText), promptText.slice(0, 240))
check('提示注入标明下一步动作', /flow_advance/.test(promptText))
const cmdOut = commands['flow'].handler('status')
check('/flow status 返回进度文本', /流程/.test(cmdOut), cmdOut.slice(0, 160))
const cmdList = commands['flow'].handler('list')
check('/flow list 返回流程库', /流程库/.test(cmdList), cmdList.slice(0, 160))
const cmdImp = commands['flow-import'].handler('')
check('/flow-import 无参给出用法', /用法/.test(cmdImp), cmdImp)
// ── P4：看板编辑器通道（show/validate/define/clone）──
const rpcShow = rpcs['flow_action']({ op: 'show', id: 'go-backend-feature' })
check('RPC show 返回流程定义（供看板编辑器）', rpcShow.ok === true && rpcShow.result.def && rpcShow.result.def.stages.length === 5, JSON.stringify({ scope: rpcShow.result && rpcShow.result.scope, ro: rpcShow.result && rpcShow.result.readOnly }))
// 注意：go-backend-feature 在本轮测试⑤ 已被 flow_sync 复制到项目层 → 优先级解析到可写的 project 层
check('RPC show：被项目层覆盖时解析到可写层', rpcShow.result.scope === 'project' && rpcShow.result.readOnly === false, JSON.stringify({ scope: rpcShow.result.scope, ro: rpcShow.result.readOnly }))
const rpcShowPreset = rpcs['flow_action']({ op: 'show', id: 'plugin-dev' })
check('RPC show：仅存于预设层时标 readOnly', rpcShowPreset.ok === true && rpcShowPreset.result.scope === 'preset' && rpcShowPreset.result.readOnly === true, JSON.stringify({ scope: rpcShowPreset.result && rpcShowPreset.result.scope, ro: rpcShowPreset.result && rpcShowPreset.result.readOnly }))
const rpcVal = rpcs['flow_action']({ op: 'validate', json: JSON.stringify(customFlow) })
check('RPC validate 返回校验结果', rpcVal.ok === true && rpcVal.result.ok === true, JSON.stringify(rpcVal.result))
const rpcValBad = rpcs['flow_action']({ op: 'validate', json: '{"id":"x","stages":[]}' })
check('RPC validate 捕获非法定义', rpcValBad.result.ok === false && rpcValBad.result.errors.length > 0, JSON.stringify(rpcValBad.result))
const rpcDefOk = rpcs['flow_action']({ op: 'define', json: JSON.stringify(Object.assign({}, customFlow, { version: 9 })), scope: 'project', expectedVersion: 2 })
check('RPC define 校验通过 + 乐观锁匹配则写入', rpcDefOk.ok === true && rpcDefOk.result.version === 9, JSON.stringify(rpcDefOk).slice(0, 200))
const rpcDefBad = rpcs['flow_action']({ op: 'define', json: JSON.stringify(Object.assign({}, customFlow, { version: 10 })), scope: 'project', expectedVersion: 1 })
check('RPC define 乐观锁冲突被拒', rpcDefBad.ok === false && /版本冲突/.test(rpcDefBad.error), JSON.stringify(rpcDefBad).slice(0, 200))
const rpcDefPreset = rpcs['flow_action']({ op: 'define', json: JSON.stringify(customFlow), scope: 'preset' })
check('RPC define preset 层被拒（只读）', rpcDefPreset.ok === false && /preset/.test(rpcDefPreset.error), JSON.stringify(rpcDefPreset).slice(0, 160))
const rpcClone = rpcs['flow_action']({ op: 'clone', id: 'go-backend-feature', newId: 'go-rpc-clone' })
check('RPC clone 生成项目级副本', rpcClone.ok === true && fs.existsSync(path.join(WS, '.pair', 'flows', 'go-rpc-clone.json')), JSON.stringify(rpcClone).slice(0, 200))

console.log('\n=== ⑦b 发布包（D9：flow_publish → flow_import 回导 → dryRun 冲突）===')
// 先造一个「无库元数据」的项目层流程，用于验证发布必须署名
r = T('flow_define').execute({ json: JSON.stringify({ schemaVersion: 1, id: 'pub-nolib', title: '无库元数据流程', stages: [{ id: 's1', title: 'S1', tasks: [], gates: [{ id: 'g1', kind: 'check', text: '确认', required: true }] }] }), scope: 'project' })
check('发布用例前置：写入无 library 的项目层流程', /已创建流程 pub-nolib|已保存流程 pub-nolib/.test(txt(r)), txt(r).slice(0, 160))
r = T('flow_publish').execute({ id: 'pub-nolib' })
check('flow_publish 缺 author → 拒绝（发布必须署名）', /缺少 author/.test(txt(r)), txt(r).slice(0, 160))
r = T('flow_publish').execute({ id: 'go-backend-feature', author: 'paircode-flow-lab', tags: 'go, backend', repo: 'github:paircode/flows', version: 2 })
check('flow_publish 生成发布包', /已生成发布包/.test(txt(r)) && /kind=paircode.flow/.test(txt(r)), txt(r).slice(0, 300))
const pubPath = '.pair/flow-exports/go-backend-feature.publish.json'
check('发布包落盘 .pair/flow-exports/<id>.publish.json', fs.existsSync(path.join(WS, pubPath)), pubPath)
const pub = JSON.parse(fs.readFileSync(path.join(WS, pubPath), 'utf8'))
check('发布包形态：kind=paircode.flow + author/tags/repo/version 元数据',
  pub.kind === 'paircode.flow' && pub.author === 'paircode-flow-lab' && pub.version === 2 &&
  pub.tags.join(',') === 'go,backend' && pub.repo === 'github:paircode/flows',
  JSON.stringify({ kind: pub.kind, author: pub.author, version: pub.version, tags: pub.tags, repo: pub.repo }))
check('发布包含 flow 主体 + stats + checksum', !!pub.flow && !!pub.checksum && pub.stats.stages === pub.flow.stages.length && pub.stats.gates >= 6,
  JSON.stringify({ stats: pub.stats, cs: pub.checksum }))
// dryRun：global 层已有同名流程（⑤ 段导入过）→ 必须报告冲突
r = T('flow_import').execute({ file: pubPath, scope: 'global', dryRun: true })
check('flow_import 消费发布包：dryRun 列出冲突', /dryRun（未落盘）：发现 1 处冲突/.test(txt(r)) && /现有 v/.test(txt(r)), txt(r).slice(0, 300))
check('dryRun 报告含动作/现有版本/导入版本/原因', !!(r.report && r.report[0] && r.report[0].action === 'skip' && r.report[0].exists === true && r.report[0].currentVersion === 1 && r.report[0].incomingVersion === 2), JSON.stringify(r.report))
check('dryRun 不落盘（global 层版本未变）', JSON.parse(fs.readFileSync(path.join(INSTALL, '.pair', 'flowlib', 'go-backend-feature.json'), 'utf8')).version === 1)
// 回导（overwrite）
r = T('flow_import').execute({ file: pubPath, scope: 'global', onConflict: 'overwrite' })
check('flow_import 回导发布包到 global 层', /已导入流程 go-backend-feature → global/.test(txt(r)), txt(r).slice(0, 200))
const glImported = JSON.parse(fs.readFileSync(path.join(INSTALL, '.pair', 'flowlib', 'go-backend-feature.json'), 'utf8'))
check('回导后版本与发布元数据落库', glImported.version === 2 && glImported.library.author === 'paircode-flow-lab' && glImported.library.tags.join(',') === 'go,backend', JSON.stringify({ v: glImported.version, lib: glImported.library }))
// 完整性 / 类型守卫
r = T('flow_import').execute({ json: JSON.stringify(Object.assign({}, pub, { checksum: 'deadbeef' })), scope: 'project' })
check('发布包 checksum 不符 → 拒绝导入', /校验和不符/.test(txt(r)), txt(r).slice(0, 160))
r = T('flow_import').execute({ json: JSON.stringify({ kind: 'paircode.toolset', flow: pub.flow }), scope: 'project' })
check('包 kind 非 paircode.flow → 拒绝（不猜测）', /不支持的包 kind/.test(txt(r)), txt(r).slice(0, 160))
r = T('flow_publish').execute({ id: 'not-exist-flow' })
check('flow_publish 未知流程 → 明确报错', /流程不存在/.test(txt(r)), txt(r).slice(0, 160))
// 同一能力经 RPC 通道（看板/外部调用一致）
const rpcPub = rpcs['flow_action']({ op: 'publish', id: 'generic', author: 'rpc-test' })
check('RPC publish 生成发布包', rpcPub.ok === true && /\.publish\.json$/.test((rpcPub.result || {}).path || ''), JSON.stringify(rpcPub).slice(0, 220))
const rpcImp = rpcs['flow_action']({ op: 'import', file: (rpcPub.result || {}).path, scope: 'project', dryRun: true })
check('RPC import dryRun 可达并回报动作', rpcImp.ok === true && !!((rpcImp.result || {}).report || [])[0], JSON.stringify(rpcImp).slice(0, 260))

console.log('\n=== ⑧ 门禁钩子（PreToolUse 拦截 + PostToolUse 记改动）===')
const blocked = hooks.PreToolUse({ toolName: 'bash', toolArgs: { command: 'rm -rf src' } })
check('破坏性操作在有阻塞门禁时被拦', !!blocked && blocked.block === true, JSON.stringify(blocked))
check('拦截反馈指明门禁与下一步', /流程门禁/.test(blocked.feedback) && /flow_advance/.test(blocked.feedback))
const notBlocked = hooks.PreToolUse({ toolName: 'read', toolArgs: { file_path: 'go.mod' } })
check('非破坏性操作不拦', notBlocked === null || notBlocked === undefined, JSON.stringify(notBlocked))
hooks.PostToolUse({ toolName: 'write', toolArgs: { file_path: 'internal/x.go' } })
hooks.PostToolUse({ toolName: 'write', toolArgs: { file_path: 'internal/x.go' } })
hooks.PostToolUse({ toolName: 'edit', toolArgs: { path: 'internal/y.go' } })
const runsAll = fs.readdirSync(path.join(WS, '.pair', 'flow-runs')).filter((f) => f.endsWith('.json'))
const withMod = runsAll.map((f) => JSON.parse(fs.readFileSync(path.join(WS, '.pair', 'flow-runs', f), 'utf8'))).filter((x) => (x.modifiedFiles || []).length)
check('PostToolUse 记录改动文件且去重', withMod.length > 0 && withMod[0].modifiedFiles.length === 2, JSON.stringify(withMod.map((x) => x.modifiedFiles)))
// 回滚建议已在 ④ 段以 g-cmd-bad 验证（含 rollbackOptions 结构）

console.log('\n=== ⑧b 门禁强度三档（§10.1：enforce / warn / off）===')
// enforce（默认）：① 拦破坏性操作 ② 每轮注入未过门禁与下一步
settingsStore = { gateStrength: 'enforce', autoRecommend: true, trackModifiedFiles: true }
const enfBlock = hooks.PreToolUse({ toolName: 'bash', toolArgs: { command: 'rm -rf src' } })
check('[enforce] 破坏性操作被拦', !!enfBlock && enfBlock.block === true, JSON.stringify(enfBlock).slice(0, 160))
const enfPrompt = promptVars['paircode_flow_state'].provider()
check('[enforce] 每轮注入未过门禁 + 下一步动作', /未过门禁/.test(enfPrompt) && /flow_advance/.test(enfPrompt), enfPrompt.slice(0, 200))
// warn：只提示不拦
settingsStore = { gateStrength: 'warn', autoRecommend: true, trackModifiedFiles: true }
const warnBlock = hooks.PreToolUse({ toolName: 'bash', toolArgs: { command: 'rm -rf src' } })
check('[warn] 不拦破坏性操作', warnBlock === null || warnBlock === undefined, JSON.stringify(warnBlock))
const warnPrompt = promptVars['paircode_flow_state'].provider()
check('[warn] 仍每轮提示未过门禁（只提示）', /未过门禁/.test(warnPrompt), warnPrompt.slice(0, 200))
// off：只记录（不拦、不注入；证据与门禁记录照常落盘）
settingsStore = { gateStrength: 'off', autoRecommend: true, trackModifiedFiles: true }
const offBlock = hooks.PreToolUse({ toolName: 'bash', toolArgs: { command: 'rm -rf src' } })
check('[off] 不拦破坏性操作', offBlock === null || offBlock === undefined, JSON.stringify(offBlock))
check('[off] 不注入流程提示（只记录）', promptVars['paircode_flow_state'].provider() === '', JSON.stringify(promptVars['paircode_flow_state'].provider()).slice(0, 120))
const offRec = T('flow_advance').execute({ convId: 'conv_test1', gateId: 'g-impl-1', evidence: 'off 档下仍照常记录证据（只记录不拦）' })
check('[off] 门禁结果照常落盘（gateResults 记录）', !!(offRec.result && offRec.result.gate && offRec.result.gate.id === 'g-impl-1'), JSON.stringify((offRec.result && offRec.result.gate) || {}).slice(0, 240))
// 证据目录落点：<ws>/.pair/flow-runs/evidence/<runId>/<gateId>.*（见 index.js P_EVID）
const offEvid = path.join(WS, '.pair', 'flow-runs', 'evidence', runId)
const offEvidFiles = fs.existsSync(offEvid) ? fs.readdirSync(offEvid) : []
check('[off] 证据文件仍写入 evidence 目录', offEvidFiles.some((f) => /g-impl-1/.test(f)), offEvidFiles.join(',') || ('目录不存在 ' + offEvid))
settingsStore = { gateStrength: 'enforce', autoRecommend: true, trackModifiedFiles: true }

console.log('\n=== ⑨ 安全边界（mock 与真实同语义）===')
let boundaryOK = false
try { fsSvc.readFile('../../../etc/passwd') } catch (e) { boundaryOK = /超出工作区范围/.test(e.message) }
check('ctx.fs 越界读被拦（mock 语义同真实）', boundaryOK)
let installBoundary = false
try { installSvc.readFile('.pair/../../secret') } catch (e) { installBoundary = /超出安装数据区/.test(e.message) }
check('ctx.install 越界被拦', installBoundary)
check('插件未使用 ctx.bash 写安装目录（无绕行）', code.indexOf('flowlib') >= 0 && code.indexOf("bash.exec('mkdir") < 0 && code.indexOf('mkdir -p') < 0)

// ── ⑩ D4-④ 看板表单编辑器 UI 断言（真实 client.js + 最小 DOM 桩；见 ui-form-test.js）──
console.log('\n=== ⑩ D4-④ 看板表单编辑器（真实 client.js + DOM 桩）===')
;(async () => {
  try {
    await require('./ui-form-test.js')(check, {
      clientPath: path.join(PLUGIN_SRC, 'client.js'),
      snapshot: (convId) => JSON.parse(httpRoutes['GET /api/flow/state']({ query: convId ? ('convId=' + convId) : '' }).body),
      rpc: (args) => rpcs['flow_action'](args),
      ws: WS,
    })
  } catch (e) {
    check('[UI] 表单编辑器脚本执行无异常抛出', false, String((e && e.stack) || e).slice(0, 400))
  }
  console.log('\n────────────────────────────────────────────')
  console.log('结果：' + pass + ' 通过 / ' + fail + ' 失败')
  if (fails.length) { console.log('\n失败项：'); fails.forEach((f) => console.log('  · ' + f)) }
  console.log('（工作区模拟目录：' + WS + ' ；安装目录模拟：' + INSTALL + '）')
  process.exit(fail ? 1 : 0)
})()
