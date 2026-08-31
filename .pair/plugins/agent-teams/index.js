// ═══════════════════════════════════════════════════════════════
// agent-teams — 多智能体团队（移植自 dsh-agent-teams，适配项目宿主）
//
// 当前会话成为队长：创建可续聊成员会话、把目标拆成带依赖的任务、
// 通过直达消息协调成员工作；成员空闲后由共享调度器自动续领就绪任务。
//
// 核心机制：
//   · 状态层   team.json + inbox/*.jsonl（每成员一邮箱），位于队长工作区
//              <wsRoot>/<stateDir>/<teamId>/；磁盘即真相（Web 面板轮询读取）
//   · 成员     可续聊子 Agent（ctx.agents.start/followup），persona 注入，
//              队长专属工具 deny（agent_teams_create 等 7 个成员不可见）
//   · 调度器   subagent/idle 事件驱动 kickTeam：空闲成员自动领取就绪任务
//              （依赖全部 completed）；parked attempt 冷恢复；邮箱优先投递
//   · 质量门禁 task kind 契约（objective/acceptance/verify/inScope）：
//              review/requirements 仅 verdict=pass 可完成；needs_revision
//              自动生成 repair+复审（上限 codeMaxRounds/maxRepairAttempts）
//   · 两阶段   create 默认 approval=required → staged（人不介入不 spawn），
//              用户经 Web 面板批准/废弃；approval=automatic 立即运行
//
// 依赖宿主能力（inject 声明）：
//   fs / logger / timer / agents / llm / app / http
// ═══════════════════════════════════════════════════════════════

const PLUGIN = 'agent-teams'

// ─── 常量 ────────────────────────────────────────────────────
const CAPTAIN_KEY = 'captain'
const TERMINAL = ['completed', 'failed', 'cancelled']
const TASK_KINDS = ['requirements', 'implementation', 'verification', 'review', 'repair', 'integration', 'work']
const TRANSITIONS = {
  pending: ['claimed', 'cancelled'],
  claimed: ['in_progress', 'failed', 'cancelled'],
  in_progress: ['completed', 'failed', 'cancelled'],
  completed: [],
  failed: [],
  cancelled: [],
}
// 成员不可见的队长专属工具（denyTools）
const MEMBER_DENIED_TOOLS = [
  'agent_teams_create',
  'agent_teams_add_member',
  'agent_teams_remove_member',
  'agent_teams_reassign_task',
  'agent_teams_create_task',
  'agent_teams_resume',
  'agent_teams_delete',
]

// ─── 命名团队模板（R2-6：profiles 对齐参考 dsh-agent-teams lib/profiles.js）───
// create(profile=…) 一键建队：固定阵容（成员角色）+ seed 任务（队长可后续
// edit_plan 调整）。profiles 上限 MAX_TEAM_PROFILES=16（参考实现同值）。
// 内置模板：
//   · default         四人研发团队（analyst/engineer/verifier/reviewer）
//   · captain-planning 队长规划型（仅 roster，任务由队长现场设计——参考
//     profile 语义：profile 提供阵容与护栏，任务 DAG 由队长规划）
const MAX_TEAM_PROFILES = 16
const TEAM_PROFILES = [
  {
    name: 'default',
    description: '四人研发团队：分析→实施→独立验证→代码审查（角色齐全的通用质量流水线）',
    members: [
      { name: 'analyst', role: 'researcher', executionPrompt: '' },
      { name: 'engineer', role: 'engineer', executionPrompt: '' },
      { name: 'verifier', role: 'qa', executionPrompt: '' },
      { name: 'reviewer', role: 'reviewer', executionPrompt: '' },
    ],
    // ★ t4 F4（2026-09 t5 修复）：种子任务按序接线（ref/deps/reviewedTaskRef）——
    //   seed 时静态生成流水线 DAG：requirements → implementation → verification；
    //   review 依赖 implementation 且 reviewedTaskRef 指向其实施任务（质量门有源任务）。
    tasks: [
      { ref: 'req', subject: '需求分析：目标拆解与验收标准定义', assignee: 'analyst', kind: 'requirements', objective: '把团队目标拆解为可验收的需求（范围/验收/风险）', acceptance: ['需求覆盖团队目标', '验收标准可测', '含风险清单'], round: 1, deps: [] },
      { ref: 'impl', subject: '实施：按需求落地改动并自测', assignee: 'engineer', kind: 'implementation', objective: '按需求实施改动，带自测/验证', inScope: [], acceptance: ['需求项全部实现', '改动可回滚', '有测试或验证'], verify: [], deps: ['req'] },
      { ref: 'ver', subject: '独立验证：构建/冒烟/回归核对', assignee: 'verifier', kind: 'verification', objective: '独立验证实施结果（构建/装载/行为）', acceptance: ['验证命令全部通过', '异常项已说明'], verify: [], deps: ['impl'] },
      { ref: 'rev', subject: '代码审查：质量门与需求对齐', assignee: 'reviewer', kind: 'review', objective: '审查实施是否满足需求且无阻塞问题', acceptance: ['审查覆盖全部改动', '结论有依据'], reviewedTaskRef: 'impl', round: 1, deps: ['impl'] },
    ],
  },
  {
    name: 'captain-planning',
    description: '队长规划型模板：仅预置固定阵容（researcher/engineer/qa/reviewer），任务 DAG 由队长在 staged 阶段现场设计',
    members: [
      { name: 'researcher', role: 'researcher', executionPrompt: '' },
      { name: 'engineer', role: 'engineer', executionPrompt: '' },
      { name: 'verifier', role: 'qa', executionPrompt: '' },
      { name: 'reviewer', role: 'reviewer', executionPrompt: '' },
    ],
    tasks: [],
  },
]
function profileByName(name) {
  const n = String(name || '').trim().toLowerCase()
  if (!n) return null
  return TEAM_PROFILES.find((p) => p.name === n) || null
}
function profileNames() {
  return TEAM_PROFILES.map((p) => p.name).join('/')
}
const DEFAULT_REVIEW_POLICY = {
  requirementsMinRounds: 2,
  requirementsMaxRounds: 3,
  codeMaxRounds: 3,
  maxRepairAttempts: 2,
  requiredReviewers: [],
}
const DEFAULT_REVIEW_OBJECTIVE = 'Review whether the latest implementation satisfies the user goal'
const DEFAULT_REVIEW_ACCEPTANCE = [
  'All planned tasks are completed',
  'The implementation satisfies the user goal',
  'No unresolved high or blocker findings remain',
]
const DEP_OUTPUT_MAX = 2000
const DEP_OUTPUTS_TOTAL_MAX = 12000
const MAILBOX_DELIVERY_LEASE_MS = 60000

// ─── 小工具 ──────────────────────────────────────────────────
function now() { return Date.now() }
function uuid() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    const v = c === 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}
function clone(v) { return JSON.parse(JSON.stringify(v)) }
function trimOpt(v) {
  if (v === undefined || v === null) return undefined
  const s = String(v).trim()
  return s === '' ? undefined : s
}
function strList(v) {
  if (!Array.isArray(v)) return []
  return v.map((x) => String(x).trim()).filter((x) => x !== '')
}
function uniq(a) { return [...new Set(a)] }
// 名称 → 安全路径段（保留 CJK/字母/数字，其余折叠为 '-'；与参考实现同行为）
function sanitizeKey(name) {
  const cleaned = String(name).trim().toLowerCase().replace(/[^\u4e00-\u9fff\u3040-\u30ff\uac00-\ud7afA-Za-z0-9]+/g, '-').replace(/^-+|-+$/g, '')
  if (cleaned === '') {
    let h = 0
    const s = String(name)
    for (let i = 0; i < s.length; i++) h = ((h << 5) - h + s.charCodeAt(i)) | 0
    return 'k-' + Math.abs(h).toString(36)
  }
  return cleaned.length > 48 ? cleaned.slice(0, 48) + '-' + cleaned.length : cleaned
}

// ─── 状态层（ctx.fs 同步；路径相对会话根，fs.resolve 自动绑定）───
function stateRootPath(wsRoot, stateDir) {
  return wsRoot ? (wsRoot.replace(/[\\/]+$/, '') + '/' + stateDir) : stateDir
}

// 持久化层接口（插件 apply 时注入，隔离 fs 根与工作区）
let F = null          // 同步 fs 服务（readFile/writeFile/exists/readdir/stat/mkdir/rm/rename）
let SYS_WS_ROOT = ''  // 主工作区根（HTTP/UI 场景工具调用无会话根时兜底）

function atomicWriteText(file, content) {
  const tmp = file + '.tmp-' + Math.floor(Math.random() * 1e6)
  F.writeFile(tmp, content)
  try { F.rename(tmp, file) } catch (e) { F.writeFile(file, content); try { F.rm(tmp) } catch (e2) {} }
}

function readTeam(stateRoot, teamId) {
  try {
    const raw = F.readFile(stateRoot + '/' + teamId + '/team.json')
    return coerceTeam(JSON.parse(raw.replace(/^\uFEFF/, '')), teamId)
  } catch (e) {
    return undefined // 不存在/损坏 → undefined（损坏会静默；面板读时也允许失败）
  }
}
// 团队状态写入即广播（浏览器 client 半 ui.on('ui:agent-teams/change') 实时刷新面板）
let emitChangeSignal = null
function writeTeam(stateRoot, team) {
  atomicWriteText(stateRoot + '/' + team.id + '/team.json', JSON.stringify(team, null, 2))
  if (typeof emitChangeSignal === 'function') {
    try { emitChangeSignal() } catch (e) { /* 通知失败不阻塞写 */ }
  }
}
function listTeamIds(stateRoot) {
  let names = []
  try { names = F.readdir(stateRoot) } catch (e) { return [] }
  return names.filter((n) => {
    try { return F.stat(stateRoot + '/' + n).isDir && n !== 'archive' } catch (e) { return false }
  })
}
function findTeamByCaptain(stateRoot, captainId) {
  for (const id of listTeamIds(stateRoot)) {
    const team = readTeam(stateRoot, id)
    if (team && team.captainSessionId === captainId) return team
  }
  return undefined
}
function findTeamByParticipant(stateRoot, sessionId) {
  for (const id of listTeamIds(stateRoot)) {
    const team = readTeam(stateRoot, id)
    if (!team) continue
    if (team.captainSessionId === sessionId) return team
    if (team.members.some((m) => m.id === sessionId && m.status !== 'removed')) return team
  }
  return undefined
}
function memberByName(team, name) {
  return team.members.find((m) => m.name === name) || undefined
}
function taskById(team, id) {
  return team.tasks.find((t) => t.id === id) || undefined
}
function coerceTeam(value, teamId) {
  if (!value || typeof value !== 'object') return undefined
  const team = {
    name: String(value.name || teamId),
    id: String(value.id || teamId),
    description: typeof value.description === 'string' ? value.description : undefined,
    captainSessionId: String(value.captainSessionId || ''),
    createdAt: Number(value.createdAt || now()),
    members: Array.isArray(value.members) ? value.members.map((m) => ({
      id: String((m && m.id) || ''),
      name: String((m && m.name) || ''),
      role: trimOpt(m && m.role),
      provider: trimOpt(m && m.provider),
      model: trimOpt(m && m.model),
      executionPrompt: trimOpt(m && m.executionPrompt),
      joinedAt: Number((m && m.joinedAt) || now()),
      status: (m && m.status) === 'removed' ? 'removed' : ((m && m.status) || 'idle'),
    })) : [],
    tasks: Array.isArray(value.tasks) ? value.tasks.map(coerceTask) : [],
    taskSeq: Number(value.taskSeq || 0),
    phase: value.phase === 'staged' ? 'staged' : 'running',
    planReviewState: value.planReviewState === 'awaiting_feedback' ? 'awaiting_feedback' : 'awaiting_review',
    approvedAt: value.approvedAt ? Number(value.approvedAt) : undefined,
    halted: value.halted === true,
    haltedAt: value.haltedAt ? Number(value.haltedAt) : undefined,
    wsRoot: typeof value.wsRoot === 'string' ? value.wsRoot : '',
    reviewPolicy: value.reviewPolicy && typeof value.reviewPolicy === 'object' ? value.reviewPolicy : undefined,
    escalated: value.escalated === true,
  }
  if (team.captainSessionId === '') return undefined
  return team
}
function coerceTask(t) {
  return {
    id: String((t && t.id) || ''),
    subject: String((t && t.subject) || ''),
    description: trimOpt(t && t.description),
    status: ['pending', 'claimed', 'in_progress', 'completed', 'failed', 'cancelled'].includes(t && t.status) ? t.status : 'pending',
    assignee: trimOpt(t && t.assignee),
    dependencies: strList(t && t.dependencies),
    output: trimOpt(t && t.output),
    attempt: Number((t && t.attempt) || 0),
    attemptId: trimOpt(t && t.attemptId),
    handoffId: trimOpt(t && t.handoffId),
    reassigning: (t && t.reassigning) === true,
    kind: TASK_KINDS.includes(t && t.kind) ? t.kind : 'work',
    round: t && t.round ? Number(t.round) : undefined,
    verdict: ['pass', 'needs_revision', 'reject'].includes(t && t.verdict) ? t.verdict : undefined,
    findings: Array.isArray(t && t.findings) ? t.findings : undefined,
    objective: trimOpt(t && t.objective),
    inScope: Array.isArray(t && t.inScope) ? t.inScope : undefined,
    outOfScope: Array.isArray(t && t.outOfScope) ? t.outOfScope : undefined,
    acceptance: Array.isArray(t && t.acceptance) ? t.acceptance : undefined,
    verify: Array.isArray(t && t.verify) ? t.verify : undefined,
    reviewedTaskId: trimOpt(t && t.reviewedTaskId),
    sourceTaskId: trimOpt(t && t.sourceTaskId),
    sourceFindingIds: Array.isArray(t && t.sourceFindingIds) ? t.sourceFindingIds : undefined,
    coverageOf: Array.isArray(t && t.coverageOf) ? t.coverageOf : undefined,
    acceptanceResults: Array.isArray(t && t.acceptanceResults) ? t.acceptanceResults : undefined,
    commandsRun: Array.isArray(t && t.commandsRun) ? t.commandsRun : undefined,
    changedPaths: Array.isArray(t && t.changedPaths) ? t.changedPaths : undefined,
    createdAt: Number((t && t.createdAt) || now()),
    updatedAt: Number((t && t.updatedAt) || now()),
  }
}

// ─── 邮箱（JSONL）───────────────────────────────────────────
function mailboxFile(stateRoot, teamId, agentKey) { return stateRoot + '/' + teamId + '/inbox/' + sanitizeKey(agentKey) + '.jsonl' }
function appendMailbox(stateRoot, teamId, agentKey, message) {
  F.mkdir(stateRoot + '/' + teamId + '/inbox', true)
  let existing = ''
  try { existing = F.readFile(mailboxFile(stateRoot, teamId, agentKey)) } catch (e) { existing = '' }
  const sep = existing !== '' && !existing.endsWith('\n') ? '\n' : ''
  F.appendFile(mailboxFile(stateRoot, teamId, agentKey), sep + JSON.stringify(message) + '\n')
}
function readMailbox(stateRoot, teamId, agentKey) {
  let raw = ''
  try { raw = F.readFile(mailboxFile(stateRoot, teamId, agentKey)) } catch (e) { return [] }
  const out = []
  for (const line of raw.split('\n')) {
    const l = line.replace(/^\uFEFF/, '').trim()
    if (l === '') continue
    try {
      const v = JSON.parse(l)
      if (v && typeof v.id === 'string' && typeof v.from === 'string' && typeof v.to === 'string' && typeof v.content === 'string') out.push(v)
    } catch (e) { /* 损坏行跳过 */ }
  }
  return out
}
function readUnreadMailbox(stateRoot, teamId, agentKey) {
  return readMailbox(stateRoot, teamId, agentKey).filter((m) => m.readAt === undefined && (m.deliveryClaimedAt === undefined || now() - m.deliveryClaimedAt >= MAILBOX_DELIVERY_LEASE_MS))
}
function mutateMailbox(stateRoot, teamId, agentKey, messageIds, mutate) {
  if (!messageIds || messageIds.length === 0) return
  const file = mailboxFile(stateRoot, teamId, agentKey)
  let raw = ''
  try { raw = F.readFile(file) } catch (e) { return }
  const sel = new Set(messageIds)
  const lines = raw.split('\n').map((rawLine) => {
    const l = rawLine.replace(/^\uFEFF/, '').trim()
    if (l === '') return rawLine
    try {
      const v = JSON.parse(l)
      if (!v || typeof v.id !== 'string' || !sel.has(v.id)) return rawLine
      return JSON.stringify(mutate(v))
    } catch (e) { return rawLine }
  })
  atomicWriteText(file, lines.join('\n'))
}
function claimMailboxDelivery(stateRoot, teamId, agentKey, ids) {
  mutateMailbox(stateRoot, teamId, agentKey, ids, (m) => Object.assign({}, m, { deliveryClaimedAt: now() }))
}
function releaseMailboxDelivery(stateRoot, teamId, agentKey, ids) {
  mutateMailbox(stateRoot, teamId, agentKey, ids, (m) => {
    const out = Object.assign({}, m)
    delete out.deliveryClaimedAt
    return out
  })
}
function acknowledgeMailbox(stateRoot, teamId, agentKey, ids) {
  mutateMailbox(stateRoot, teamId, agentKey, ids, (m) => {
    const out = Object.assign({}, m)
    delete out.deliveryClaimedAt
    out.deliveredAt = out.deliveredAt || now()
    out.readAt = out.readAt || now()
    return out
  })
}
function createMessage(from, to, content) {
  return { id: uuid(), from, to, content: String(content), ts: now() }
}
// retired-members：成员退役后不可唤醒（跨进程持久）
function retiredMembersFile(stateRoot) { return stateRoot + '/retired-members.json' }
function readRetiredMembers(stateRoot) {
  try {
    const arr = JSON.parse(F.readFile(retiredMembersFile(stateRoot)))
    return Array.isArray(arr) ? new Set(arr.filter((x) => typeof x === 'string' && x !== '')) : new Set()
  } catch (e) { return new Set() }
}
function recordRetiredMembers(stateRoot, ids) {
  const keep = (ids || []).filter((x) => x !== '')
  if (keep.length === 0) return
  const retired = readRetiredMembers(stateRoot)
  keep.forEach((x) => retired.add(x))
  F.mkdir(stateRoot, true)
  atomicWriteText(retiredMembersFile(stateRoot), JSON.stringify([...retired].sort(), null, 2) + '\n')
}
function archiveTeamDir(stateRoot, teamId) {
  const base = stateRoot + '/archive'
  F.mkdir(base, true)
  // ★ 2026-08-30：目标已存在（同 id 多次废弃/重建后再次存档）→ 追加时间戳唯一化。
  //   否则 Windows os.Rename 覆盖非空目录失败（【废弃后仍存在】根因），catch 静默吞掉。
  let target = base + '/' + teamId
  if (F.exists(target)) target = base + '/' + teamId + '-' + Date.now()
  try { F.rename(stateRoot + '/' + teamId, target) } catch (e) {
    // rename 失败（目标占用/权限）→ 兜底：去掉唯一后缀重试（清同 id 旧档）
    try { F.rm(base + '/' + teamId, true) } catch (e2) {}
    try { F.rename(stateRoot + '/' + teamId, base + '/' + teamId) } catch (e3) { /* 允许失败 */ }
  }
}

// ─── 任务规则 ────────────────────────────────────────────────
function unsatisfiedDependencies(tasks, deps) {
  const byId = {}
  tasks.forEach((t) => { byId[t.id] = t })
  return (deps || []).filter((id) => !byId[id] || byId[id].status !== 'completed')
}
function transitionError(current, next) {
  if (current === next) return undefined
  if (TRANSITIONS[current] && TRANSITIONS[current].includes(next)) return undefined
  return 'task status cannot move from "' + current + '" to "' + next + '"'
}
function activateTaskAttempt(task, assignee) {
  task.status = 'claimed'
  task.assignee = assignee
  task.attemptId = uuid()
  task.handoffId = undefined
  task.reassigning = false
  task.output = undefined
  task.updatedAt = now()
}
function beginTaskAttempt(task, assignee) {
  task.attempt = (task.attempt || 0) + 1
  activateTaskAttempt(task, assignee)
}
function invalidateTaskAttempt(task, nextAssignee, reassigning) {
  task.attemptId = undefined
  task.handoffId = uuid()
  task.status = 'pending'
  task.assignee = nextAssignee
  task.reassigning = reassigning === true
  task.output = undefined
  task.updatedAt = now()
}
function cancelUnfinishedTask(task, output) {
  if (TERMINAL.includes(task.status)) return
  task.status = 'cancelled'
  task.attemptId = undefined
  task.handoffId = undefined
  task.reassigning = false
  if (output !== undefined) task.output = output
  task.updatedAt = now()
}

// ─── 质量门禁 ────────────────────────────────────────────────
function taskKindOf(task) { return task && TASK_KINDS.includes(task.kind) ? task.kind : 'work' }
function isQualityKind(kind) { return kind !== undefined && kind !== 'work' }
function resolveReviewPolicy(policy) {
  const p = policy && typeof policy === 'object' ? policy : {}
  return Object.assign({}, DEFAULT_REVIEW_POLICY, p)
}
function sanitizeReviewObjective(v, fallback) {
  const s = trimOpt(v)
  return s || fallback || DEFAULT_REVIEW_OBJECTIVE
}
function sanitizeReviewAcceptance(v) {
  const arr = strList(v)
  return arr.length > 0 ? arr : DEFAULT_REVIEW_ACCEPTANCE.slice()
}
function classifyChangedPath(path, inScope, outOfScope) {
  const norm = String(path).replace(/\\/g, '/').replace(/^\.\//, '')
  const matchAny = (patterns) => (patterns || []).some((raw) => {
    const p = String(raw).replace(/\\/g, '/').replace(/^\.\//, '').trim()
    if (p === '') return false
    if (p.endsWith('/')) return norm.startsWith(p)
    if (p.startsWith(':')) return norm === p.slice(1)
    return norm === p || norm.startsWith(p + '/')
  })
  if (matchAny(inScope)) return 'in_scope'
  if (matchAny(outOfScope)) return 'out_of_scope'
  return 'unscoped'
}
function openHighFindings(findings) {
  return (findings || []).filter((f) => f && f.resolved !== true && (f.severity === 'high' || f.severity === 'blocker'))
}
function acceptanceCovered(acceptance, results) {
  const list = strList(acceptance)
  if (list.length === 0) return true
  const rows = (results || []).filter((r) => r && r.status === 'passed')
  return list.every((c) => rows.some((r) => r.criterion === c))
}
function verifyCovered(verify, runs) {
  const list = strList(verify)
  if (list.length === 0) return true
  const rows = (runs || []).filter((r) => r && r.status === 'passed')
  return list.every((c) => rows.some((r) => r.command === c))
}
// 建任务契约校验（返回 {ok, kind?, error?}）
function validateCreateTask(team, input) {
  const subject = trimOpt(input.subject)
  if (!subject) return { ok: false, error: 'task subject must not be empty' }
  let kind = trimOpt(input.kind) || 'work'
  if (!TASK_KINDS.includes(kind)) kind = 'work'
  if (kind !== 'work') {
    const objective = sanitizeReviewObjective(input.objective)
    const acceptance = sanitizeReviewAcceptance(input.acceptance)
    if (kind === 'review' || kind === 'requirements') {
      if (kind === 'review' && !trimOpt(input.reviewedTaskId)) {
        return { ok: false, error: 'review task requires reviewedTaskId' }
      }
      return { ok: true, kind, objective, acceptance }
    }
    if (kind === 'repair' && !trimOpt(input.sourceTaskId)) {
      return { ok: false, error: 'repair task requires sourceTaskId' }
    }
    const inScope = strList(input.inScope)
    const verify = strList(input.verify)
    if (inScope.length === 0) return { ok: false, error: kind + ' task requires non-empty inScope' }
    if (verify.length === 0) return { ok: false, error: kind + ' task requires verify commands' }
    return { ok: true, kind, objective, acceptance, inScope, verify }
  }
  return { ok: true, kind }
}
// 完成时审计
function evaluateQualityCompletion(task, update) {
  const nextStatus = update.status
  if (nextStatus !== undefined && nextStatus !== task.status) {
    const err = transitionError(task.status, nextStatus)
    if (err) return { ok: false, error: err }
  }
  const kind = taskKindOf(task)
  if (kind === 'work') return { ok: true }
  const verdict = update.verdict !== undefined ? update.verdict : task.verdict
  const findings = update.findings !== undefined ? update.findings : task.findings
  if (kind === 'review' || kind === 'requirements') {
    if (nextStatus === 'completed') {
      if (verdict === undefined) return { ok: false, error: kind + ' cannot complete without verdict=pass' }
      if (verdict !== 'pass') return { ok: false, error: kind + ' with verdict=' + verdict + ' cannot complete' }
      if (openHighFindings(findings).length > 0) return { ok: false, error: kind + ' pass cannot leave unresolved high/blocker findings' }
    }
    if (nextStatus === 'failed' && (verdict === 'needs_revision' || verdict === 'reject')) {
      if (!findings || findings.length < 1) return { ok: false, error: kind + ' ' + verdict + ' requires at least one finding' }
    }
    return { ok: true }
  }
  if (['implementation', 'repair', 'verification', 'integration'].includes(kind)) {
    const commands = update.commandsRun !== undefined ? update.commandsRun : task.commandsRun
    if ((commands || []).some((r) => r && r.status === 'failed') && nextStatus === 'completed') {
      return { ok: false, error: 'verify failure must fail the task', requiredStatus: 'failed' }
    }
    if (nextStatus !== 'completed') return { ok: true }
    const accResults = update.acceptanceResults !== undefined ? update.acceptanceResults : task.acceptanceResults
    if (!acceptanceCovered(task.acceptance, accResults)) return { ok: false, error: kind + ' completion requires passed acceptanceResults for every acceptance item' }
    if (!verifyCovered(task.verify, commands)) return { ok: false, error: kind + ' completion requires a passed commandsRun entry for every verify command' }
    if (kind === 'implementation' || kind === 'repair') {
      const changed = update.changedPaths !== undefined ? update.changedPaths : task.changedPaths
      if (changed === undefined || changed.length === 0) return { ok: false, error: kind + ' completion requires changedPaths' }
      for (const path of changed) {
        const cls = classifyChangedPath(path, task.inScope || [], task.outOfScope || [])
        if (cls !== 'in_scope') return { ok: false, error: kind + ' cannot complete: ' + path + ' is ' + cls }
      }
    }
  }
  return { ok: true }
}
// 失败后自动跟进（review/requirements 失败 → repair+复审 / 下一轮需求）
function planQualityFollowUp(team, closed) {
  const empty = { created: [], tasks: [], escalated: false }
  const kind = taskKindOf(closed)
  if ((kind !== 'review' && kind !== 'requirements') || closed.status !== 'failed') return empty
  if (closed.verdict === 'reject') return Object.assign({}, empty, { escalated: true })
  if (closed.verdict !== 'needs_revision') return empty
  const policy = resolveReviewPolicy(team.reviewPolicy)
  const currentRound = closed.round || 1
  const nextRound = currentRound + 1
  const maxRounds = kind === 'requirements' ? policy.requirementsMaxRounds : policy.codeMaxRounds
  if (nextRound > maxRounds) return Object.assign({}, empty, { escalated: true })
  const unresolved = (closed.findings || []).filter((f) => f && f.resolved !== true)
  const findingIds = unresolved.map((f) => f.id)
  if (kind === 'requirements') {
    const nextTaskRow = {
      kind: 'requirements',
      subject: 'requirements-round-' + nextRound,
      assignee: closed.assignee,
      dependencies: [],
      round: nextRound,
      objective: sanitizeReviewObjective(closed.objective, 'Converge remaining open questions'),
      acceptance: sanitizeReviewAcceptance(unresolved.map((f) => f.requiredFix)),
    }
    return { created: [nextTaskRow], tasks: [nextTaskRow], escalated: false }
  }
  const sourceId = closed.reviewedTaskId || closed.sourceTaskId
  if (!sourceId) return empty
  const source = taskById(team, sourceId)
  const key = findingIds.slice().sort().join(',')
  const hasOpen = team.tasks.some((t) => taskKindOf(t) === 'repair' && t.sourceTaskId === sourceId && strList(t.sourceFindingIds).slice().sort().join(',') === key && ['pending', 'claimed', 'in_progress'].includes(t.status))
  if (hasOpen) return empty
  const repairCount = team.tasks.filter((t) => taskKindOf(t) === 'repair' && t.sourceTaskId === sourceId && strList(t.sourceFindingIds).slice().sort().join(',') === key).length
  if (repairCount >= policy.maxRepairAttempts) return Object.assign({}, empty, { escalated: true })
  const schedulable = (preferred, forbidden) => {
    if (preferred && preferred !== CAPTAIN_KEY && preferred !== forbidden) {
      const live = team.members.find((m) => m.name === preferred && m.status !== 'removed')
      if (live) return live.name
    }
    const alt = team.members.find((m) => m.status !== 'removed' && m.name !== CAPTAIN_KEY && m.name !== forbidden)
    return alt ? alt.name : undefined
  }
  const files = unresolved.map((f) => f.file).filter((f) => typeof f === 'string' && f !== '')
  const implementer = schedulable(source ? source.assignee : undefined, undefined)
  const repair = {
    kind: 'repair',
    subject: 'repair-round-' + nextRound,
    assignee: implementer,
    dependencies: [sourceId],
    round: nextRound,
    objective: (source && source.objective) || closed.objective || ('Fix findings from ' + sourceId),
    inScope: files.length > 0 ? files : (source ? source.inScope : undefined),
    outOfScope: source ? source.outOfScope : undefined,
    verify: source ? source.verify : undefined,
    acceptance: unresolved.map((f) => f.requiredFix),
    sourceTaskId: sourceId,
    sourceFindingIds: findingIds,
  }
  const reviewer = schedulable(closed.assignee !== implementer ? closed.assignee : undefined, implementer)
  const review = {
    kind: 'review',
    subject: 'review-round-' + nextRound,
    assignee: reviewer,
    dependencies: ['repair-round-' + nextRound],
    round: nextRound,
    objective: sanitizeReviewObjective(closed.objective, DEFAULT_REVIEW_OBJECTIVE),
    acceptance: sanitizeReviewAcceptance(closed.acceptance),
    reviewedTaskId: 'repair-round-' + nextRound,
  }
  return { created: [repair, review], tasks: [repair, review], escalated: false }
}
function resumeTeamState(team, reason) {
  if (team.halted !== true) return { status: 'already_running' }
  if (!reason || String(reason).trim() === '') return { status: 'rejected', error: 'resume requires a non-empty reason' }
  return { status: 'resumed' }
}
function buildCoverageMatrix(goalItems, tasks) {
  return strList(goalItems).map((goalItem) => {
    const covering = tasks.filter((t) => strList(t.coverageOf).includes(goalItem))
    const taskIds = covering.map((t) => t.id)
    if (covering.length === 0) return { goal_item: goalItem, task_ids: taskIds, status: 'missing' }
    if (covering.some((t) => t.status === 'failed' || t.status === 'cancelled')) return { goal_item: goalItem, task_ids: taskIds, status: 'blocked' }
    if (covering.some((t) => t.status !== 'completed')) return { goal_item: goalItem, task_ids: taskIds, status: 'partial' }
    return { goal_item: goalItem, task_ids: taskIds, status: 'covered' }
  })
}
function describeQualityLoop(team) {
  const closure = [
    'requirements-loop: ' + (team.tasks.some((t) => taskKindOf(t) === 'requirements') ? 'in graph' : 'none'),
    'review-loop: ' + (team.tasks.some((t) => taskKindOf(t) === 'review') ? 'in graph' : 'none'),
    'repair-loop: ' + (team.tasks.some((t) => taskKindOf(t) === 'repair') ? 'in graph' : 'none'),
  ]
  if (team.escalated === true) {
    closure.push('escalated: review/repair loop hit its configured ceiling; the captain must decide the next step')
  }
  return { summary: closure.join('; '), escalated: team.escalated === true }
}
function qualityPlanningPrompt() {
  return [
    'Quality gates: requirements/implementation/verification/review/repair/integration tasks need a contract (objective + acceptance; implementation/repair also inScope + verify).',
    'Review/requirements complete only with verdict=pass; needs_revision/reject must fail with findings. The system then opens repair + a next review depending on the successful source, never the failed review.',
    'Do not approve your own implementation. A failed immediate review or verification opens the repair/review loop automatically.',
  ].join(' ')
}

// ═══════════════════════════════════════════════════════════════
//  成员生命周期（适配 ctx.agents：start 首轮 / followup 续轮）
// ═══════════════════════════════════════════════════════════════
function memberPersona(team, member, stateDir, executionPrompt) {
  const goal = (team.description || '').trim() || '(not provided)'
  const injected = (member.executionPrompt || '').trim() || (executionPrompt || '').trim()
  return [
    'You are a team member of a multi-agent team. Work only the tasks you are assigned; you share the task list with teammates.',
    'Team goal: ' + goal,
    'Your name is "' + member.name + '"' + (member.role ? '; role: ' + member.role : '') + '.',
    injected ? 'Execution guidance:\n' + injected : '',
    'Team state lives under ' + stateDir + '/ — read-only diagnostics; mutate team state only through agent_teams_* tools.',
    'When you finish a task, call agent_teams_update_task immediately with the current attempt_id; then send_message to captain with the result summary. Do not explore beyond the assigned task in your turn.',
  ].filter((l) => l !== '').join('\n\n')
}
function memberWelcome(team, memberName) {
  const member = memberByName(team, memberName)
  return [
    'Welcome to team "' + team.name + '" as ' + memberName + '.',
    'Team goal: ' + ((team.description || '').trim() || '(not provided)'),
    'You are a durable continuable member: keep your persona and memory across turns. When idle you will be assigned ready tasks from the shared task list.',
    'Use agent_teams_status to see the shared state, agent_teams_claim_task to claim an assignment, and agent_teams_update_task to report progress/results.',
  ].join('\n')
}
function fallbackMailboxPrompt(messages) {
  return [
    'AgentTeams delivered messages that were persisted while live delivery was unavailable:',
  ].concat(messages.map((m) => 'From ' + m.from + ':\n' + m.content)).concat([
    'Handle these messages in this turn. Task assignments still require agent_teams_claim_task and the current attempt_id.',
  ]).join('\n')
}

// 派生依赖结果（completed 祖先按拓扑序）
function collectDependencyOutputs(tasks, taskId) {
  const byId = {}
  tasks.forEach((t) => { byId[t.id] = t })
  const visiting = new Set()
  const visited = new Set()
  const ordered = []
  const walk = (id) => {
    if (visiting.has(id)) return
    if (visited.has(id)) return
    visiting.add(id)
    const task = byId[id]
    if (task) {
      for (const dep of task.dependencies) walk(dep)
      if (id !== taskId) ordered.push(task)
    }
    visiting.delete(id)
    visited.add(id)
  }
  walk(taskId)
  return ordered.filter((t) => t.status === 'completed').map((t) => ({
    id: t.id, subject: t.subject, output: t.output,
  }))
}
function formatDependencyOutputs(items) {
  if (!items || items.length === 0) return '(none)'
  let total = 0
  const lines = items.map((item) => {
    const raw = item.output && item.output !== '' ? item.output : '(no output recorded)'
    const body = raw.length > DEP_OUTPUT_MAX ? raw.slice(0, DEP_OUTPUT_MAX) + ' [truncated]' : raw
    total += body.length
    return '- ' + item.id + ' ' + item.subject + ':\n  ' + body
  })
  let text = lines.join('\n')
  if (total > DEP_OUTPUTS_TOTAL_MAX) text = text.slice(0, DEP_OUTPUTS_TOTAL_MAX) + '\n[outputs truncated]'
  return text
}
function assignmentPrompt(ticket, stateDir, teamId) {
  const description = ticket.description ? '\n\n' + ticket.description : ''
  const goal = (ticket.teamDescription || '').trim() || '(not provided)'
  const kind = ticket.kind || 'work'
  const contract = [
    'Kind: ' + kind + (ticket.round ? ' (round ' + ticket.round + ')' : ''),
    ticket.objective ? 'Objective: ' + ticket.objective : '',
    ticket.inScope && ticket.inScope.length ? 'In scope: ' + ticket.inScope.join(', ') : '',
    ticket.outOfScope && ticket.outOfScope.length ? 'Out of scope: ' + ticket.outOfScope.join(', ') : '',
    ticket.acceptance && ticket.acceptance.length ? 'Acceptance: ' + ticket.acceptance.join('; ') : '',
    ticket.verify && ticket.verify.length ? 'Verify: ' + ticket.verify.join('; ') : '',
    ticket.reviewedTaskId ? 'Reviewed task: ' + ticket.reviewedTaskId : '',
  ].filter((l) => l !== '').join('\n')
  let structuredCompletion = ''
  if (['implementation', 'repair', 'verification', 'integration'].includes(kind)) {
    const acc = (ticket.acceptance || []).map((c) => ({ criterion: c, status: 'passed', evidence: '<what proved it>' }))
    const cmd = (ticket.verify || []).map((v) => ({ command: v, status: 'passed', exitCode: 0, evidence: '<observed result>' }))
    structuredCompletion = '\nStructured completion payload (keep arrays in contract order):\nacceptanceResults: ' + JSON.stringify(acc) + '\ncommandsRun: ' + JSON.stringify(cmd) + (kind === 'implementation' || kind === 'repair' ? '\nchangedPaths: list the actual workspace-relative POSIX paths you changed.' : '')
  }
  return 'AgentTeams automatic task assignment from the shared task list.\n\n' +
    'You are executing as configured member "' + ticket.memberName + '".\nDo not start a teammate\'s assigned task.\n\n' +
    'Team goal:\n' + goal + '\n\n' +
    (ticket.executionPrompt ? 'Execution guidance:\n' + ticket.executionPrompt + '\n\n' : '') +
    'Completed dependency results:\n' + formatDependencyOutputs(ticket.dependencyOutputs) + '\n\n' +
    'Task: ' + ticket.taskId + ' — ' + ticket.subject + description +
    (contract ? '\n\nContract:\n' + contract + '\n' : '') +
    structuredCompletion + '\n\n' +
    'Attempt: ' + ticket.attempt + '\nAttempt id: ' + ticket.attemptId + '\n\n' +
    'Call agent_teams_claim_task for ' + ticket.taskId + '; it will return this same attempt_id. Include attempt_id=' + ticket.attemptId + ' in every agent_teams_update_task call. If it is rejected as stale, stop work because the task was reassigned. claimed cannot jump to completed. Mark in_progress first, then completed or failed. Include attempt_id on every update. Then send_message to captain and become idle.\n' +
    'When finishing: use status=completed only when the task\'s success criteria are satisfied; use status=failed when blocking findings or validation failures mean downstream work must not proceed; include a concise output in either case. Quality kinds must submit structured fields. After the work and verification finish, call agent_teams_update_task immediately; do not wait for captain confirmation and do not continue exploring. Do not approve your own implementation. Mail is not a formal next review. Work only this task and only its in-scope paths in this turn.\n\n' +
    'State policy: ' + stateDir + '/' + teamId + '/ is read-only diagnostics; mutate team state only through agent_teams_* tools.'
}

// ═══════════════════════════════════════════════════════════════
//  调度器（subagent/idle 事件驱动）
// ═══════════════════════════════════════════════════════════════
function spawnedMemberId(team, memberName) {
  const m = memberByName(team, memberName)
  return m ? m.id : ''
}
function ownedOpenTask(tasks, memberName) {
  return tasks.find((t) => t.assignee === memberName && !TERMINAL.includes(t.status)) || undefined
}
function nextReadyTask(tasks, memberName) {
  for (const t of tasks) {
    if (t.status !== 'pending') continue
    if (unsatisfiedDependencies(tasks, t.dependencies).length > 0) continue
    if (t.assignee !== undefined && t.assignee !== memberName) continue
    return t
  }
  return undefined
}

return {
  name: 'agent-teams',
  // ctx.webServer 供 HTTP 路由；此处不需要（ctx.http 就够）——inject 声明按需
  inject: ['fs', 'logger', 'timer', 'agents', 'llm', 'app', 'http', 'commands'],
  purpose: '多智能体团队（dsh-agent-teams 移植）：队长派生可续聊成员、任务 DAG 自动调度、质量门禁修复循环、Web 活动面板',
  apply(ctx) {
    F = ctx.fs
    SYS_WS_ROOT = (ctx.app && ctx.app.workspaceRoot) || ''
    const log = ctx.logger('agent-teams')

    // ── ★ 2026-08-31 按需激活 ──
    // 本插件是重型协议插件：其系统提示段 + agent_teams_* 工具默认对 agent 隐藏
    // （避免 agent 每次对话都看到协议而「自我触发」）；用户在对话里执行
    // /agent-teams 命令后，本会话才激活（提示段/工具立即可见）。
    try {
      const r = ctx.activation.declare({ command: 'agent-teams' })
      if (r && r.ok) log.info('已声明按需激活：/agent-teams 触发本会话激活')
    } catch (e) {
      log.warn('ctx.activation.declare 失败（宿主不支持按需激活，插件按常驻注入）: ' + (e && e.message || e))
    }

    // ── 实时广播：团队状态变化 → 浏览器面板刷新（事件驱动，无轮询锁竞争）──
    emitChangeSignal = () => {
      try { ctx.emit('ui:agent-teams/change', { at: now() }) } catch (e) { /* 忽略 */ }
    }

    // ── 配置（默认 + getSettings 覆盖）──
    let settings = {}
    try { settings = ctx.getSettings('agent-teams') || {} } catch (e) { settings = {} }
    const CFG = {
      stateDir: trimOpt(settings.stateDir) || '.agent-teams',
      memberModel: trimOpt(settings.memberModel) || '',
      executionPrompt: trimOpt(settings.executionPrompt) || '',
      maxMembers: Number(settings.maxMembers || 8),
      codeMaxRounds: Number(settings.codeMaxRounds || DEFAULT_REVIEW_POLICY.codeMaxRounds),
      maxRepairAttempts: Number(settings.maxRepairAttempts || DEFAULT_REVIEW_POLICY.maxRepairAttempts),
      requirementsMaxRounds: Number(settings.requirementsMaxRounds || DEFAULT_REVIEW_POLICY.requirementsMaxRounds),
      // ★ Round3 ④.1：成员会话提供方式——'spawn'（新会话，默认，行为不变）|
      //   'fork'（以队长会话消息快照派生，DSH subagent_fork 对齐；persona 经 system 覆盖）。
      //   配置来源：插件设置 memberProvider → 环境变量 AGENT_TEAMS_MEMBER_PROVIDER → 默认 spawn。
      memberProvider: trimOpt(settings.memberProvider) ||
        (typeof process !== 'undefined' && process.env ? trimOpt(process.env.AGENT_TEAMS_MEMBER_PROVIDER) : '') || 'spawn',
    }
    if (CFG.maxMembers < 0) CFG.maxMembers = 8
    if (CFG.codeMaxRounds <= 0) CFG.codeMaxRounds = DEFAULT_REVIEW_POLICY.codeMaxRounds
    if (CFG.maxRepairAttempts <= 0) CFG.maxRepairAttempts = DEFAULT_REVIEW_POLICY.maxRepairAttempts
    if (CFG.memberProvider !== 'fork') CFG.memberProvider = 'spawn' // 非法值回落 spawn

    // 注册配置段（设置面板可见）
    try {
      ctx.registerSettings({
        key: 'agent-teams',
        title: '多智能体团队（agent-teams）',
        fields: [
          { name: 'stateDir', label: '团队状态目录（工作区相对）', type: 'text', default: '.agent-teams' },
          { name: 'memberModel', label: '成员默认模型（留空=跟随队长）', type: 'text' },
          { name: 'executionPrompt', label: '成员全局执行指导', type: 'text' },
          // Round3 ④.1：成员提供方式（spawn=新会话 / fork=队长快照派生）
          { name: 'memberProvider', label: '成员提供方式（spawn/fork）', type: 'text', default: 'spawn' },
          { name: 'maxMembers', label: '团队规模上限', type: 'number', default: 8 },
          { name: 'codeMaxRounds', label: '代码审查轮次上限', type: 'number', default: 3 },
          { name: 'maxRepairAttempts', label: '修复尝试上限', type: 'number', default: 2 },
        ],
      })
    } catch (e) { /* 设置面板注册失败不阻塞 */ }

    // ── 会话身份/工作区解析 ──
    function callerOf(args) {
      const convId = args && args._convID ? String(args._convID) : ''
      const wsRoot = args && args._wsRoot ? String(args._wsRoot) : SYS_WS_ROOT
      return { convId, wsRoot }
    }
    function stateRootOf(wsRoot) {
      const root = wsRoot || SYS_WS_ROOT
      return root ? (root.replace(/[\\/]+$/, '') + '/' + CFG.stateDir) : CFG.stateDir
    }
    // 身份解析：captain 或成员
    function requireIdentity(team, convId, allowCaptain) {
      if (!convId) throw new Error('agent_teams tools require a calling agent session')
      if (team.captainSessionId === convId && allowCaptain !== false) return { kind: 'captain', name: CAPTAIN_KEY, sessionId: convId }
      const member = team.members.find((m) => m.id === convId && m.status !== 'removed')
      if (member) return { kind: 'member', name: member.name, sessionId: convId }
      if (team.captainSessionId === convId) throw new Error('this tool is captain-only')
      throw new Error('you are not a participant of this team')
    }
    function requireCaptainTeam(wsRoot, captainId) {
      const team = findTeamByCaptain(stateRootOf(wsRoot), captainId)
      if (!team) throw new Error('you do not lead a team; call agent_teams_create first')
      return team
    }
    function requireParticipantTeam(wsRoot, convId) {
      const team = findTeamByParticipant(stateRootOf(wsRoot), convId)
      if (!team) throw new Error('you are not a member of any agent team')
      return team
    }
    function requireStaged(team) {
      if (team.phase !== 'staged') throw new Error('team "' + team.name + '" is not staged')
    }

    // ── 成员 spawn / 投递 ──
    // spawnMember：启动成员会话（显式 convId=member.id 持久可续）
    function spawnMember(ctxRef, team, member, task, captain) {
      const persona = memberPersona(team, member, CFG.stateDir, CFG.executionPrompt)
      const model = member.model || CFG.memberModel || ''
      const provider = member.provider || ''
      const spec = {
        convId: member.id,
        parentConvId: team.captainSessionId,
        label: 'agent-teams:' + member.name,
        team: team.id,
        member: member.name,
        task: task,
        system: persona,
        model: model,
        provider: provider,
        // R2-6：reasoning_effort 透传（成员档位覆盖，宿主 applyThinking 下发）
        reasoningEffort: member.reasoningEffort || '',
        wsRoot: team.wsRoot || '',
        denyTools: MEMBER_DENIED_TOOLS,
      }
      if (team.wsRoot) spec.wsRoot = team.wsRoot
      // ★ Round3 ④.1 memberProvider：fork 时以队长会话消息快照派生成员
      //   （fork 种子 = 队长历史；persona 经 system 覆盖；缺失 fork 能力回落 spawn）。
      if (CFG.memberProvider === 'fork' && captain) {
        try {
          const rec = ctxRef.agents.fork(Object.assign({}, spec, { forkFrom: captain }))
          log.info('member forked: ' + member.name + ' conv=' + rec.convId + ' forkFrom=' + captain)
          return rec
        } catch (e) {
          log.warn('member fork 失败，回落 spawn: ' + (e && e.message || e))
        }
      }
      const rec = ctxRef.agents.start(spec)
      log.info('member spawned: ' + member.name + ' conv=' + rec.convId)
      return rec
    }
    // 成员剩余模型（add_member 时队长未指定模型 → 快照队长当前路由）
    function currentModel() {
      try { return ctx.llm.current() || {} } catch (e) { return {} }
    }
    // deliverToMember：给成员投递一轮输入（未登记则重建会话 —— 冷恢复）
    function deliverToMember(wsRoot, teamId, member, text) {
      const stateRoot = stateRootOf(wsRoot)
      const retired = readRetiredMembers(stateRoot)
      if (retired.has(member.id)) return false
      try {
        const rec = ctx.agents.status(member.id)
        if (rec === null || rec === undefined) {
          // 冷启动：未登记 → 用完整规格重建（保 persona/model/denyTools）
          const team = readTeam(stateRoot, teamId)
          if (!team) return false
          const m = memberByName(team, member.name)
          if (!m) return false
          spawnMember(ctx, team, m, text, team.captainSessionId)
          return true
        }
        ctx.agents.followup(member.id, text)
        return true
      } catch (e) {
        log.warn('deliver failed for ' + member.name + ': ' + (e && e.message || e))
        return false
      }
    }

    // parked attempts：空闲成员仍持有开放 attempt → 不自动重派（等待指令/冷恢复例外）
    const parkedAttempts = new Map()

    // 队列串行（参考 withTeamLock 的近似：浏览器事件与工具调用都在 VM 锁内串行，
    // 事件回调顺序到达，此处仅防同一个成员的重入）
    const memberQueues = new Map()
    function serializeMember(key, op) {
      const prev = memberQueues.get(key) || Promise.resolve()
      const tail = prev.then(op)
      memberQueues.set(key, tail)
      const clear = () => { if (memberQueues.get(key) === tail) memberQueues.delete(key) }
      tail.then(clear, clear)
      return tail
    }

    function isMemberAvailable(member, team) {
      if (member.status === 'removed') return false
      if (!member.id) return false
      if (member.id === '') return false
      return true
    }

    async function kickMember(wsRoot, teamId, memberName) {
      const stateRoot = stateRootOf(wsRoot)
      const key = stateRoot + '\u0000' + teamId + '\u0000' + memberName
      await serializeMember(key, async () => {
        const team = readTeam(stateRoot, teamId)
        if (!team || team.halted === true || team.phase === 'staged' || team.captainSessionId === '') return
        const member = memberByName(team, memberName)
        if (!member || !isMemberAvailable(member, team)) return

        // ① 邮箱优先投递
        const unread = readUnreadMailbox(stateRoot, teamId, member.name)
        if (unread.length > 0) {
          claimMailboxDelivery(stateRoot, teamId, member.name, unread.map((m) => m.id))
          const accepted = deliverToMember(wsRoot, teamId, member, fallbackMailboxPrompt(unread))
          if (accepted) acknowledgeMailbox(stateRoot, teamId, member.name, unread.map((m) => m.id))
          else releaseMailboxDelivery(stateRoot, teamId, member.name, unread.map((m) => m.id))
          return
        }

        // ② 开放式 owned attempt：依赖满足才可恢复/认领；否则 park 或闲置等依赖
        const owned = ownedOpenTask(team.tasks, member.name)
        const ownedBlocked = owned !== undefined && unsatisfiedDependencies(team.tasks, owned.dependencies).length > 0
        const parked = parkedAttempts.get(member.id)
        const parkedId = parked ? parked.attemptId : undefined
        const recoverOwned = owned !== undefined && !ownedBlocked && (owned.attemptId === undefined || owned.attemptId !== parkedId)
        const task = owned !== undefined
          ? (ownedBlocked ? undefined : (recoverOwned ? owned : undefined))
          : nextReadyTask(team.tasks, member.name)
        if (!task) {
          if (member.status !== 'idle') { member.status = 'idle'; writeTeam(stateRoot, team) }
          return
        }
        // ★ 归属任务冷恢复缺依赖检查（v1 bug：assignee 指定但从未认领的任务，在依赖未
        //   完成时被成员按「owned 冷恢复」抢跑 → DAG 乱序执行，如 t4 在 t3 未完成时被
        //   verifier 提前认领）。修复：ownedBlocked 不派（等依赖完成）。
        // ★ 另：原 `if (!recoverOwned) return` 会拦截 nextReadyTask 新任务（自动调度
        //   死代码，成员 idle 后无人接力 → 团队看起来「直接结束」的根源之一）——已改为
        //   仅「owned 存在且非恢复路径」才拦截（停驻语义），无 owned 的新任务照常自动派发。
        if (owned !== undefined && (ownedBlocked || !recoverOwned)) return

        // 认领 + 唤醒
        const fresh = readTeam(stateRoot, teamId)
        if (!fresh || fresh.halted === true) return
        const freshMember = memberByName(fresh, memberName)
        if (!freshMember || !isMemberAvailable(freshMember, fresh)) return
        const freshTask = taskById(fresh, task.id)
        if (!freshTask || freshTask.status !== 'pending') return
        beginTaskAttempt(freshTask, memberName)
        if (freshTask.reassigning) { freshTask.reassigning = false }
        freshMember.status = 'working'
        writeTeam(stateRoot, fresh)

        const depOutputs = collectDependencyOutputs(fresh.tasks, freshTask.id)
        const ticket = {
          taskId: freshTask.id,
          memberName: memberName,
          memberId: freshMember.id,
          attempt: freshTask.attempt,
          attemptId: freshTask.attemptId,
          subject: freshTask.subject,
          description: freshTask.description,
          teamDescription: fresh.description,
          dependencyOutputs: depOutputs,
          executionPrompt: freshMember.executionPrompt || CFG.executionPrompt,
          kind: freshTask.kind,
          round: freshTask.round,
          objective: freshTask.objective,
          inScope: freshTask.inScope,
          outOfScope: freshTask.outOfScope,
          acceptance: freshTask.acceptance,
          verify: freshTask.verify,
          reviewedTaskId: freshTask.reviewedTaskId,
        }
        const prompt = assignmentPrompt(ticket, CFG.stateDir, teamId)
        const ok = deliverToMember(wsRoot, teamId, freshMember, prompt)
        if (!ok) {
          // 投递失败：任务回到 pending（attempt 撤销，等待下次调度）
          const rollback = readTeam(stateRoot, teamId)
          if (rollback) {
            const rt = taskById(rollback, task.id)
            if (rt && rt.attemptId === freshTask.attemptId) {
              invalidateTaskAttempt(rt)
              const rm = memberByName(rollback, memberName)
              if (rm) rm.status = 'idle'
              writeTeam(stateRoot, rollback)
            }
          }
        }
      })
    }
    async function kickTeam(wsRoot, teamId) {
      const stateRoot = stateRootOf(wsRoot)
      const team = readTeam(stateRoot, teamId)
      if (!team || team.halted === true || team.phase === 'staged') return
      for (const member of team.members) {
        if (member.status === 'removed') continue
        await kickMember(wsRoot, teamId, member.name)
      }
    }

    // subagent/idle 事件 → 同步成员状态 + 续领
    ctx.on('subagent/idle', (payload) => {
      const teamId = payload && payload.team ? String(payload.team) : ''
      const memberName = payload && payload.member ? String(payload.member) : ''
      const convId = payload && payload.convId ? String(payload.convId) : ''
      if (!teamId || !memberName) return
      // 找到团队（按 convId 定位更稳；事件携带 team/member）
      const wsRoot = SYS_WS_ROOT
      const stateRoot = stateRootOf(wsRoot)
      const team = readTeam(stateRoot, teamId)
      if (!team) return
      const member = memberByName(team, memberName)
      if (!member || member.status === 'removed') return
      const owned = ownedOpenTask(team.tasks, member.name)
      if (owned && owned.attemptId) parkedAttempts.set(member.id, { attemptId: owned.attemptId, at: now() })
      else parkedAttempts.delete(member.id)
      if (member.status !== 'idle') { member.status = 'idle'; writeTeam(stateRoot, team) }
      kickMember(wsRoot, teamId, memberName).catch((e) => log.warn('kickMember failed: ' + e))
    })

    // ── 调度兜底 watchdog（修复「成员轮次结束未更新任务 → 团队静止卡死」）──
    // 事件驱动（subagent/idle）可能漏：成员被 stop 无 idle 事件、成员一轮结束忘了
    // 调 agent_teams_update_task → 开放 attempt 被 park 且再无事件触发 → 团队看起来
    // 「直接结束」。兜底：周期扫描（10s 快速短路）：①park 超时（成员连续 4 分钟无
    // 动静）→ 回收 attempt 回共享池并通知队长；②存在依赖满足的 pending 任务但无
    // 任何成员 running → 再 kick 一轮（事件缺失自愈）。
    const STALE_ATTEMPT_MIN = 4
    const STALE_ATTEMPT_MS = STALE_ATTEMPT_MIN * 60 * 1000
    const watchdogCancel = ctx.timer.interval(() => {
      try {
        const roots = (() => { try { return ctx.fs.roots() || [] } catch (e) { return [] } })()
        const wsList = roots.length ? roots : [SYS_WS_ROOT]
        for (const ws of wsList) {
          const sr = stateRootOf(ws)
          for (const id of listTeamIds(sr)) {
            const team = readTeam(sr, id)
            if (!team || team.halted === true || team.phase !== 'running') continue
            let changed = false
            // ① 卡死 attempt 回收：成员 park 超时（轮次结束后未更新任务）
            for (const m of team.members) {
              if (m.status === 'removed' || !m.id) continue
              const parked = parkedAttempts.get(m.id)
              if (!parked || typeof parked !== 'object' || !parked.at) continue
              const owned = ownedOpenTask(team.tasks, m.name)
              if (!owned || !owned.attemptId) { parkedAttempts.delete(m.id); continue }
              if (now() - parked.at < STALE_ATTEMPT_MS) continue
              const tid = owned.id
              invalidateTaskAttempt(owned) // 回共享池（assignee 清空，其他成员可接手）
              parkedAttempts.delete(m.id)
              changed = true
              log.warn('watchdog: member ' + m.name + ' idle >' + STALE_ATTEMPT_MIN + 'min on ' + tid + ' → attempt returned to shared pool')
              try {
                ctx.agents.followup(team.captainSessionId, 'AgentTeams watchdog: task ' + tid + ' (' + owned.subject + ') was held by member ' + m.name + ' but made no progress for ' + STALE_ATTEMPT_MIN + ' minutes. Its attempt was returned to the shared pool for reassignment. Check agent_teams_status and reassign or instruct as needed.')
              } catch (e) { log.warn('watchdog notify failed: ' + ((e && e.message) || e)) }
            }
            if (changed) writeTeam(sr, team)
            // ② 有就绪任务但全员静止 → 兜底 kick 一轮
            const ready = team.tasks.some((t) =>
              t.status === 'pending' &&
              unsatisfiedDependencies(team.tasks, t.dependencies).length === 0 &&
              (t.assignee === undefined || team.members.some((m) => m.name === t.assignee && m.status !== 'removed')))
            if (!ready) continue
            let anyRunning = false
            try { anyRunning = (ctx.agents.list({ team: team.id }) || []).some((r) => r.state === 'running') } catch (e) { anyRunning = false }
            if (anyRunning) continue
kickTeam(ws, id).catch((e) => log.warn('watchdog kick failed: ' + ((e && e.message) || e)))
          }
        }
      } catch (e) {
        log.warn('watchdog tick failed: ' + ((e && e.message) || e))
      }
    }, 10000)
    if (watchdogCancel && typeof ctx.effect === 'function') ctx.effect(watchdogCancel)

    // ── 工具注册（13 个，语义对齐参考实现 dsh-agent-teams）──
    const tools = [
      // ═══ agent_teams_create（队长）═══
      {
        name: 'agent_teams_create',
        description: 'Create a team. Use approval=required for a two-phase plan: members and tasks remain unspawned/unclaimed until the user reviews the Web plan and explicitly approves it. approval=automatic preserves the legacy immediate-execution path. Optionally pass profile=<name> to seed the roster (and seed tasks) from a named team template: available profiles are ' + profileNames() + ' (default = 四人研发质量流水线; captain-planning = 仅预置阵容、任务由队长规划).',
        usageGuide: '创建 AgentTeams 团队（当前会话成为队长）。默认 approval=required：先建暂存计划，待用户批准后再启动成员。profile 可选：一键套用命名模板（固定阵容+seed 任务）。',
        parameters: {
          type: 'object',
          properties: {
            name: { type: 'string', description: 'Name for the new team (used as its stable id).' },
            description: { type: 'string', description: 'Team purpose / the goal the team will work on.' },
            profile: { type: 'string', description: 'Optional named team template to seed members/tasks: ' + profileNames() + '. Defaults to none (empty roster).' },
            approval: { type: 'string', enum: ['required', 'automatic'], description: 'required stages the plan for explicit user review; automatic starts immediately. Defaults to required.' },
          },
          required: ['name'],
        },
        execute(args) {
          const caller = callerOf(args)
          if (!caller.convId) throw new Error('agent_teams tools require a calling agent session')
          const stateRoot = stateRootOf(caller.wsRoot)
          const existing = findTeamByCaptain(stateRoot, caller.convId)
          if (existing) throw new Error('you already lead team "' + existing.name + '"; archive it before creating another')
          const name = trimOpt(args.name)
          if (!name) throw new Error('team name must not be empty')
          const teamId = sanitizeKey(name)
          if (readTeam(stateRoot, teamId)) throw new Error('team "' + name + '" already exists')
          const approval = args.approval === 'automatic' ? 'automatic' : 'required'
          const team = {
            name: name, id: teamId,
            description: trimOpt(args.description),
            captainSessionId: caller.convId,
            createdAt: now(),
            members: [], tasks: [], taskSeq: 0,
            phase: approval === 'automatic' ? 'running' : 'staged',
            planReviewState: approval === 'automatic' ? undefined : 'awaiting_review',
            wsRoot: caller.wsRoot,
            reviewPolicy: { codeMaxRounds: CFG.codeMaxRounds, maxRepairAttempts: CFG.maxRepairAttempts, requirementsMaxRounds: CFG.requirementsMaxRounds },
          }
          // ★ R2-6 profiles：套用命名模板（成员阵容 + seed 任务；队长可后续 edit_plan 调整）
          const prof = profileByName(args.profile)
          if (prof) {
            for (const pm of prof.members) {
              team.members.push({
                id: '', name: pm.name, role: pm.role || '',
                provider: '', model: '', reasoningEffort: '',
                executionPrompt: pm.executionPrompt || '',
                joinedAt: now(), status: 'idle',
              })
            }
            // ★ t4 F4（t5 修复）：两遍 seed——先建全部任务并登记 ref→id，再按
            //   deps/reviewedTaskRef 静态接线流水线 DAG（approval=automatic
            //   也不会乱序；review 质量门有源任务）。
            const refToId = {}
            const created = []
            for (const pt of prof.tasks || []) {
              team.taskSeq += 1
              const task = coerceTask({
                id: 't' + team.taskSeq, subject: pt.subject,
                description: pt.description || '', status: 'pending',
                assignee: pt.assignee || '', dependencies: [],
                kind: pt.kind || 'work', round: pt.round ? Number(pt.round) : undefined,
                objective: pt.objective, acceptance: pt.acceptance,
                createdAt: now(), updatedAt: now(),
              })
              if (pt.ref) refToId[pt.ref] = task.id
              created.push({ task, pt })
              team.tasks.push(task)
            }
            for (const { task, pt } of created) {
              task.dependencies = uniq((pt.deps || []).map((r) => refToId[r]).filter((x) => x))
              if (pt.reviewedTaskRef && refToId[pt.reviewedTaskRef]) {
                task.reviewedTaskId = refToId[pt.reviewedTaskRef]
              }
              if (task.dependencies.length > 0 || task.reviewedTaskId) {
                task.updatedAt = now()
              }
            }
          }
          F.mkdir(stateRoot + '/' + teamId + '/inbox', true)
          writeTeam(stateRoot, team)
          const seeded = prof ? ' from profile "' + prof.name + '" (' + team.members.length + ' members, ' + team.tasks.length + ' seed tasks)' : ''
          if (approval === 'automatic') {
            kickTeam(caller.wsRoot, teamId).catch(() => {})
            return 'Team "' + name + '" created (' + teamId + ') in running mode' + seeded + '; add members with agent_teams_add_member.'
          }
          return 'Team "' + name + '" created (' + teamId + ') as a staged plan' + seeded + '. Build the roster with agent_teams_add_member and the task DAG with agent_teams_create_task, then tell the user the Web plan is ready for review.'
        },
      },

      // ═══ agent_teams_edit_plan（队长，staged 专用）═══
      {
        name: 'agent_teams_edit_plan',
        description: 'Atomically revise the current staged AgentTeams plan without spawning members or scheduling tasks. Submit dependent edits in order (update downstream dependencies or assignees, then remove tasks, then remove unused members).',
        usageGuide: '原子批量修订暂存计划（staged 阶段）。mutations 按序执行：先改下游依赖/认领人，再删任务，最后删成员。',
        parameters: {
          type: 'object',
          properties: {
            mutations: {
              type: 'array',
              items: { type: 'object', properties: {} },
              description: 'Ordered batch of edits: {action:"update_member",memberName,role?,provider,model,reasoningEffort?,executionPrompt?} | {action:"update_task",taskId,subject,description?,assignee?,dependencies} | {action:"add_task",subject,description?,assignee?,dependencies} | {action:"remove_task",taskId} | {action:"remove_member",memberName}.',
            },
          },
          required: ['mutations'],
        },
        execute(args) {
          const caller = callerOf(args)
          const team = requireCaptainTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          requireStaged(team)
          const mutations = Array.isArray(args.mutations) ? args.mutations : []
          if (mutations.length === 0) throw new Error('at least one staged plan operation is required')
          for (const op of mutations) {
            const action = op && op.action
            if (action === 'update_member') {
              const m = memberByName(team, String(op.memberName || ''))
              if (!m) throw new Error('member "' + op.memberName + '" not found')
              if (m.id) throw new Error('staged member "' + m.name + '" was already spawned')
              m.role = trimOpt(op.role)
              if (op.provider !== undefined) m.provider = trimOpt(op.provider) || ''
              if (op.model !== undefined) m.model = trimOpt(op.model) || ''
              if (op.reasoningEffort !== undefined) m.reasoningEffort = trimOpt(op.reasoningEffort)
              if (op.executionPrompt !== undefined) m.executionPrompt = trimOpt(op.executionPrompt)
            } else if (action === 'update_task') {
              const t = taskById(team, String(op.taskId || ''))
              if (!t) throw new Error('task "' + op.taskId + '" not found')
              if (t.status !== 'pending' || (t.attempt || 0) !== 0) throw new Error('task "' + t.id + '" has already started and cannot be edited')
              const subject = trimOpt(op.subject)
              if (!subject) throw new Error('task subject must not be empty')
              t.subject = subject
              t.description = trimOpt(op.description)
              t.assignee = trimOpt(op.assignee)
              t.dependencies = uniq(strList(op.dependencies))
              t.updatedAt = now()
            } else if (action === 'add_task') {
              const subject = trimOpt(op.subject)
              if (!subject) throw new Error('task subject must not be empty')
              team.taskSeq += 1
              team.tasks.push(Object.assign(coerceTask({}), {
                id: 't' + team.taskSeq, subject: subject,
                description: trimOpt(op.description),
                status: 'pending', assignee: trimOpt(op.assignee),
                dependencies: uniq(strList(op.dependencies)),
                attempt: 0, kind: 'work',
                createdAt: now(), updatedAt: now(),
              }))
            } else if (action === 'remove_task') {
              const idx = team.tasks.findIndex((t) => t.id === String(op.taskId || ''))
              if (idx < 0) throw new Error('task "' + op.taskId + '" not found')
              team.tasks.splice(idx, 1)
            } else if (action === 'remove_member') {
              const idx = team.members.findIndex((m) => m.name === String(op.memberName || ''))
              if (idx < 0) throw new Error('member "' + op.memberName + '" not found')
              team.members.splice(idx, 1)
            } else {
              throw new Error('unknown plan action: ' + action)
            }
          }
          writeTeam(stateRoot, team)
          return 'Plan revised (' + team.members.length + ' members, ' + team.tasks.length + ' tasks). Reconfirm with the user before approve.'
        },
      },

      // ═══ agent_teams_approve（队长）═══
      {
        name: 'agent_teams_approve',
        description: 'Approve and start a staged team plan. Call this only in response to an explicit user approval in a new user turn; never call it during the turn that created or edited the plan.',
        usageGuide: '批准暂存计划并启动团队（spawn 成员 + 调度器开始派活）。必须在新用户轮次得到用户明确批准后调用。',
        parameters: {
          type: 'object',
          properties: {
            confirmation: { type: 'string', description: "The user's explicit approval statement." },
          },
          required: ['confirmation'],
        },
        execute(args) {
          const caller = callerOf(args)
          const team = requireCaptainTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          requireStaged(team)
          const count = approveStagedTeam(stateRoot, team)
          return 'Team "' + team.name + '" approved: ' + count.members + ' members spawned, ' + count.tasks + ' tasks queued. Scheduler is now running. Monitor with agent_teams_status.'
        },
      },

      // ═══ agent_teams_add_member（队长）═══
      {
        name: 'agent_teams_add_member',
        description: 'Add a member to the team roster. In a staged team this only adds an editable plan row and does not spawn a child; approval spawns the final configuration. In a running team it creates the durable continuable member immediately.',
        usageGuide: '向团队添加成员。staged 阶段仅添加计划行；running 阶段立即 spawn 可续聊成员会话。',
        parameters: {
          type: 'object',
          properties: {
            name: { type: 'string', description: 'Unique member name inside the team.' },
            role: { type: 'string', description: 'Role of the member (e.g. researcher, engineer, reviewer).' },
            provider: { type: 'string', description: 'Optional LLM provider route. Use only when the user explicitly requests a different provider; requires model.' },
            model: { type: 'string', description: 'Optional model override. Omit for the current model route.' },
            reasoningEffort: { type: 'string', description: 'Optional reasoning effort override.' },
            executionPrompt: { type: 'string', description: 'Optional member-specific execution prompt.' },
          },
          required: ['name'],
        },
        execute(args) {
          const caller = callerOf(args)
          const team = requireCaptainTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          const name = trimOpt(args.name)
          if (!name) throw new Error('member name must not be empty')
          if (memberByName(team, name)) throw new Error('member "' + name + '" already exists')
          const active = team.members.filter((m) => m.status !== 'removed').length
          if (active >= CFG.maxMembers) throw new Error('team member cap reached (' + CFG.maxMembers + ')')
          const cur = currentModel()
          const model = trimOpt(args.model) || CFG.memberModel || (cur && cur.model) || ''
          const memberRow = {
            id: '', name: name,
            role: trimOpt(args.role),
            provider: trimOpt(args.provider) || (cur && cur.provider) || '',
            model: model,
            reasoningEffort: trimOpt(args.reasoningEffort),
            executionPrompt: trimOpt(args.executionPrompt),
            joinedAt: now(), status: 'idle',
          }
          team.members.push(memberRow)
          writeTeam(stateRoot, team)
          if (team.phase === 'running') {
            memberRow.id = 'conv_sub_' + sanitizeKey(team.id) + '_' + sanitizeKey(name) + '_' + Math.floor(Math.random() * 100000)
            writeTeam(stateRoot, team)
            try { spawnMember(ctx, team, memberRow, memberWelcome(team, name), team.captainSessionId) } catch (e) { log.warn('add_member spawn failed: ' + e) }
            kickMember(caller.wsRoot, team.id, name).catch(() => {})
            return 'Member "' + name + '" added and spawned in running team "' + team.name + '".'
          }
          return 'Member "' + name + '" added to the staged plan of "' + team.name + '" (spawned on approval).'
        },
      },

      // ═══ agent_teams_remove_member（队长）═══
      {
        name: 'agent_teams_remove_member',
        description: 'Remove a member safely: revoke its current attempts, return all unfinished owned tasks to the shared pending pool, interrupt its live turn, and mark it removed.',
        usageGuide: '安全移除成员：撤销其 attempt、未完成任务回到共享池、中断其会话并标记 removed（不可再唤醒）。',
        parameters: {
          type: 'object',
          properties: { name: { type: 'string', description: 'Name of the member to remove.' } },
          required: ['name'],
        },
        execute(args) {
          const caller = callerOf(args)
          const team = requireCaptainTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          const m = memberByName(team, String(args.name || ''))
          if (!m) throw new Error('member "' + args.name + '" not found')
          if (team.phase === 'staged') {
            team.members = team.members.filter((x) => x.name !== m.name)
            writeTeam(stateRoot, team)
            return 'Member "' + m.name + '" removed from the staged plan.'
          }
          if (m.status === 'removed') throw new Error('member "' + m.name + '" is already removed')
          m.status = 'removed'
          for (const t of team.tasks) {
            if (t.assignee === m.name && !TERMINAL.includes(t.status)) invalidateTaskAttempt(t)
          }
          writeTeam(stateRoot, team)
          if (m.id) {
            recordRetiredMembers(stateRoot, [m.id])
            try { ctx.agents.stop(m.id) } catch (e) {}
          }
          kickTeam(caller.wsRoot, team.id).catch(() => {})
          return 'Member "' + m.name + '" removed; unfinished owned tasks returned to the shared pool.'
        },
      },

      // ═══ agent_teams_delete_task（队长）═══
      {
        name: 'agent_teams_delete_task',
        description: 'Delete a task that has reached a terminal state (completed/failed/cancelled). Any other tasks that list it in dependencies or as reviewedTaskId/sourceTaskId get those references cleared, so the DAG stays consistent. Use for obsolete finished tasks; keep finished tasks whose output is still needed.',
        usageGuide: '删除已结束（terminal）任务：completed/failed/cancelled 可删；运行中/待办不可删（暂存阶段调整用 edit_plan remove_task）。删除后自动清理其他任务的依赖引用。',
        parameters: {
          type: 'object',
          properties: {
            taskId: { type: 'string', description: 'Task id to delete (e.g. "t6").' },
          },
          required: ['taskId'],
        },
        execute(args) {
          const caller = callerOf(args)
          const team = requireCaptainTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          const target = taskById(team, args.taskId ? String(args.taskId) : '')
          if (!target) throw new Error('task "' + (args.taskId || '') + '" not found')
          if (!TERMINAL.includes(target.status)) {
            throw new Error('task "' + target.id + '" is not terminal ("' + target.status + '"); only completed/failed/cancelled tasks can be deleted. For staged plans use agent_teams_edit_plan remove_task.')
          }
          team.tasks = team.tasks.filter((x) => x.id !== target.id)
          for (const other of team.tasks) {
            let touched = false
            const deps = other.dependencies || []
            if (deps.includes(target.id)) { other.dependencies = deps.filter((d) => d !== target.id); touched = true }
            if (other.reviewedTaskId === target.id) { other.reviewedTaskId = undefined; touched = true }
            if (other.sourceTaskId === target.id) { other.sourceTaskId = undefined; touched = true }
            if (touched) other.updatedAt = now()
          }
          writeTeam(stateRoot, team)
          return 'Task "' + target.id + '" (' + target.subject + ', ' + target.status + ') deleted; dangling dependency references cleared.'
        },
      },

      // ═══ agent_teams_create_task（队长）═══
      {
        name: 'agent_teams_create_task',
        description: 'Create a task in your team\'s task list. Every call must include a non-empty subject, including verification and review tasks. Tasks can depend on other tasks (dependencies): a task is only claimable once every dependency is completed. Optionally assign it to a member, who still claims it before working.',
        usageGuide: '创建任务（队长）。subject 必填；dependencies 必填未完成前不可领取；审查/验证任务也要有 subject；质量种类需契约字段。',
        parameters: {
          type: 'object',
          properties: {
            subject: { type: 'string', description: 'Required non-empty title for this task.' },
            description: { type: 'string', description: 'What needs to be done, in detail.' },
            dependencies: { type: 'array', items: { type: 'string' }, description: 'Task ids this task depends on (must be completed before this task can be claimed).' },
            assignee: { type: 'string', description: 'Optional member name this task is intended for.' },
            kind: { type: 'string', enum: ['requirements', 'implementation', 'verification', 'review', 'repair', 'integration', 'work'], description: 'Task kind. Defaults to work (legacy, no quality gates).' },
            round: { type: 'integer', description: '1-based review / requirements / repair round.' },
            objective: { type: 'string', description: 'Required non-empty objective for quality kinds.' },
            inScope: { type: 'array', items: { type: 'string' }, description: 'Workspace-relative POSIX paths this task may change.' },
            outOfScope: { type: 'array', items: { type: 'string' }, description: 'Workspace-relative POSIX paths this task must not change.' },
            acceptance: { type: 'array', items: { type: 'string' }, description: 'Acceptance criteria. Required for quality kinds.' },
            verify: { type: 'array', items: { type: 'string' }, description: 'Verification commands. Required for implementation/repair.' },
            deliverables: { type: 'array', items: { type: 'string' }, description: 'Expected deliverable paths or names.' },
            nonGoals: { type: 'array', items: { type: 'string' }, description: 'Explicit non-goals.' },
            reviewedTaskId: { type: 'string', description: 'Task being reviewed. Required for kind=review.' },
            sourceTaskId: { type: 'string', description: 'Source implementation/artifact. Required for kind=repair.' },
            sourceFindingIds: { type: 'array', items: { type: 'string' }, description: 'Finding ids this repair must close.' },
            coverageOf: { type: 'array', items: { type: 'string' }, description: 'User-constraint / goal items this task covers.' },
            resume: { type: 'boolean', description: 'If true, clear halted in the same lock before creating the task.' },
            resumeReason: { type: 'string', description: 'Required non-empty reason when resume=true.' },
          },
          required: ['subject'],
        },
        execute(args) {
          const caller = callerOf(args)
          const team = requireCaptainTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          if (team.halted === true) {
            if (args.resume !== true) throw new Error('team is halted; call agent_teams_resume or pass resume=true with resumeReason')
            const reason = trimOpt(args.resumeReason)
            if (!reason) throw new Error('resumeReason is required when resume=true')
            team.halted = false
            team.haltedAt = undefined
          }
          const gate = validateCreateTask(team, {
            subject: args.subject, kind: args.kind, objective: args.objective,
            inScope: args.inScope, outOfScope: args.outOfScope,
            acceptance: args.acceptance, verify: args.verify,
            reviewedTaskId: args.reviewedTaskId, sourceTaskId: args.sourceTaskId,
            sourceFindingIds: args.sourceFindingIds,
          })
          if (!gate.ok) throw new Error(gate.error || 'create_task rejected by quality gates')
          const deps = uniq(strList(args.dependencies))
          for (const dep of deps) {
            if (!taskById(team, dep)) throw new Error('dependency "' + dep + '" does not exist in team "' + team.name + '"')
          }
          const assignee = trimOpt(args.assignee)
          if (assignee) {
            if (assignee !== CAPTAIN_KEY && !memberByName(team, assignee)) throw new Error('assignee "' + assignee + '" is not a member')
          }
          const kind = gate.kind || 'work'
          team.taskSeq += 1
          const task = coerceTask({
            id: 't' + team.taskSeq, subject: String(args.subject).trim(),
            description: trimOpt(args.description), status: 'pending',
            assignee: assignee, dependencies: deps,
            kind: kind, round: args.round ? Number(args.round) : undefined,
            objective: gate.objective, acceptance: gate.acceptance,
            inScope: gate.inScope, verify: gate.verify,
            outOfScope: strList(args.outOfScope), deliverables: strList(args.deliverables),
            nonGoals: strList(args.nonGoals), reviewedTaskId: trimOpt(args.reviewedTaskId),
            sourceTaskId: trimOpt(args.sourceTaskId), sourceFindingIds: strList(args.sourceFindingIds),
            coverageOf: strList(args.coverageOf),
            createdAt: now(), updatedAt: now(),
          })
          team.tasks.push(task)
          writeTeam(stateRoot, team)
          if (!team.halted) kickTeam(caller.wsRoot, team.id).catch(() => {})
          return 'Task "' + task.subject + '" created as ' + task.id + ' (status ' + task.status + (task.assignee ? ', assigned to ' + task.assignee : '') + ').'
        },
      },

      // ═══ agent_teams_reassign_task（队长）═══
      {
        name: 'agent_teams_reassign_task',
        description: 'Reassign a task to a member or to captain. Revokes the old attempt and waits for that member to quiesce; late results from the old owner cannot overwrite the new attempt.',
        usageGuide: '转派任务（队长）。assignee 可为成员名或 captain。旧 attempt 撤销，旧成员迟到结果不能覆盖新 attempt。',
        parameters: {
          type: 'object',
          properties: {
            task_id: { type: 'string', description: 'The task id to reassign.' },
            assignee: { type: 'string', description: 'New owner: a member name or "captain".' },
          },
          required: ['task_id', 'assignee'],
        },
        execute(args) {
          const caller = callerOf(args)
          const team = requireCaptainTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          const task = taskById(team, String(args.task_id || ''))
          if (!task) throw new Error('task "' + args.task_id + '" not found')
          if (TERMINAL.includes(task.status)) throw new Error('terminal task ' + task.id + ' is immutable')
          const assignee = trimOpt(args.assignee) || CAPTAIN_KEY
          if (assignee !== CAPTAIN_KEY && !memberByName(team, assignee)) throw new Error('assignee "' + assignee + '" is not a member')
          const hadAssignee = task.assignee
          invalidateTaskAttempt(task, assignee, assignee !== CAPTAIN_KEY)
          task.reassigning = false // 本项目无异步 quiesce 等待：立即生效（成员旧 attempt 已失效）
          writeTeam(stateRoot, team)
          if (assignee !== CAPTAIN_KEY) kickMember(caller.wsRoot, team.id, assignee).catch(() => {})
          return 'Task ' + task.id + ' reassigned from "' + (hadAssignee || 'nobody') + '" to "' + assignee + '".'
        },
      },

      // ═══ agent_teams_claim_task（成员/队长）═══
      {
        name: 'agent_teams_claim_task',
        description: 'Claim a task before working on it. The returned attempt_id is the current execution capability; every agent_teams_update_task call from that member must include it and becomes stale after retry/reassignment.',
        usageGuide: '领取任务（成员/队长）。返回 attempt_id 为当前执行凭据；后续 update_task 必须携带；转派后旧 attempt_id 失效。',
        parameters: {
          type: 'object',
          properties: {
            task_id: { type: 'string', description: 'The task id to claim.' },
            assignee: { type: 'string', description: 'Member to claim for (captain only; defaults to the task\'s assignee).' },
          },
          required: ['task_id'],
        },
        execute(args) {
          const caller = callerOf(args)
          const team = requireParticipantTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          const identity = requireIdentity(team, caller.convId)
          const task = taskById(team, String(args.task_id || ''))
          if (!task) throw new Error('task "' + args.task_id + '" not found')
          if (task.reassigning === true) throw new Error('task ' + task.id + ' is being reassigned; wait for the handoff to finish')
          let assignee = task.assignee
          if (identity.kind === 'captain') {
            if (args.assignee !== undefined) {
              const a = trimOpt(args.assignee)
              if (a && a !== CAPTAIN_KEY && !memberByName(team, a)) throw new Error('assignee "' + a + '" is not a member')
              assignee = a || CAPTAIN_KEY
            }
          } else {
            if (args.assignee !== undefined) throw new Error('members cannot set assignee when claiming a task')
            if (assignee !== undefined && assignee !== identity.name) throw new Error('task ' + task.id + ' is assigned to "' + assignee + '", not you')
            assignee = identity.name
          }
          if (task.status === 'claimed' || task.status === 'in_progress') {
            if (assignee === undefined || task.assignee !== assignee) throw new Error('task ' + task.id + ' is already claimed by "' + (task.assignee || 'nobody') + '"')
            return 'Task ' + task.id + ' already claimed by ' + assignee + ' (attempt ' + (task.attempt || 0) + ', status ' + task.status + ').'
          }
          if (task.status !== 'pending') throw new Error('task ' + task.id + ' cannot be claimed from "' + task.status + '"')
          const unsatisfied = unsatisfiedDependencies(team.tasks, task.dependencies || [])
          if (unsatisfied.length > 0) throw new Error('task ' + task.id + ' dependencies unsatisfied: ' + unsatisfied.join(', '))
          beginTaskAttempt(task, assignee || CAPTAIN_KEY)
          writeTeam(stateRoot, team)
          const m = assignee === CAPTAIN_KEY ? null : memberByName(team, assignee)
          if (m && m.id) { m.status = 'working'; writeTeam(stateRoot, team) }
          return 'Task ' + task.id + ' claimed by ' + (assignee || CAPTAIN_KEY) + ' (attempt ' + task.attempt + ', attempt_id ' + task.attemptId + ', status ' + task.status + ').'
        },
      },

      // ═══ agent_teams_update_task（成员/队长）═══
      {
        name: 'agent_teams_update_task',
        description: 'Update a task status/output. Members must supply the current attempt_id returned by claim_task; stale attempts are rejected after takeover/reassignment. Terminal results are immutable. A captain must use reassign_task(assignee="captain") before updating member-owned work.',
        usageGuide: '更新任务状态/输出（成员/队长）。成员必须携带当前 attempt_id；陈旧 attempt 报错；终态不可变。',
        parameters: {
          type: 'object',
          properties: {
            task_id: { type: 'string', description: 'The task id to update.' },
            status: { type: 'string', enum: ['in_progress', 'completed', 'failed', 'cancelled'], description: 'New status (in_progress, completed, failed, cancelled).' },
            output: { type: 'string', description: 'Result summary; set when completing or failing.' },
            attempt_id: { type: 'string', description: 'Current execution capability returned by claim_task (required for members when present on the task).' },
            verdict: { type: 'string', enum: ['pass', 'needs_revision', 'reject'], description: 'Required for completing requirements/review. needs_revision and reject must fail the task.' },
            findings: { type: 'array', items: { type: 'object', properties: { id: { type: 'string' }, severity: { type: 'string', enum: ['low', 'medium', 'high', 'blocker'] }, problem: { type: 'string' }, requiredFix: { type: 'string' }, file: { type: 'string' }, line: { type: 'integer' }, resolved: { type: 'boolean' } } }, description: 'Structured review findings. Required when verdict is needs_revision or reject.' },
            changedPaths: { type: 'array', items: { type: 'string' }, description: 'Workspace-relative POSIX paths changed by this implementation/repair.' },
            acceptanceResults: { type: 'array', items: { type: 'object', properties: { criterion: { type: 'string' }, status: { type: 'string', enum: ['passed', 'failed'] }, evidence: { type: 'string' } } }, description: 'Acceptance evidence in contract order.' },
            commandsRun: { type: 'array', items: { type: 'object', properties: { command: { type: 'string' }, status: { type: 'string', enum: ['passed', 'failed'] }, exitCode: { type: 'integer' }, evidence: { type: 'string' } } }, description: 'Verification evidence in contract order.' },
          },
          required: ['task_id', 'status'],
        },
        execute(args) {
          const caller = callerOf(args)
          const team = requireParticipantTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          const identity = requireIdentity(team, caller.convId)
          const task = taskById(team, String(args.task_id || ''))
          if (!task) throw new Error('task "' + args.task_id + '" not found')
          if (identity.kind === 'captain' && task.assignee !== undefined && task.assignee !== CAPTAIN_KEY) {
            throw new Error('task ' + task.id + ' is owned by member "' + task.assignee + '"; call agent_teams_reassign_task with assignee="captain" before takeover')
          }
          if (identity.kind === 'member') {
            if (task.assignee !== identity.name) throw new Error('task ' + task.id + ' is assigned to "' + (task.assignee || 'nobody') + '", not you')
            if (task.attemptId !== undefined && args.attempt_id !== task.attemptId) {
              throw new Error('stale attempt for task ' + task.id + ': expected the current attempt_id; stop work and request fresh assignment')
            }
          }
          const nextStatus = args.status
          if (TERMINAL.includes(task.status)) {
            const sameStatus = nextStatus === undefined || nextStatus === task.status
            const sameOutput = args.output === undefined || args.output === task.output
            if (!sameStatus || !sameOutput) throw new Error('terminal task ' + task.id + ' is immutable; use agent_teams_reassign_task to retry failed/cancelled work')
            return 'Task ' + task.id + ' is already ' + task.status + ' (immutable).'
          }
          // 质量门禁审计
          const gate = evaluateQualityCompletion(task, {
            status: nextStatus, verdict: args.verdict, findings: args.findings,
            acceptanceResults: args.acceptanceResults, commandsRun: args.commandsRun,
            changedPaths: args.changedPaths,
          })
          if (!gate.ok) throw new Error(gate.error)
          // 应用更新
          if (nextStatus !== undefined) task.status = nextStatus
          if (args.output !== undefined) task.output = String(args.output)
          if (args.verdict !== undefined) task.verdict = args.verdict
          if (args.findings !== undefined) task.findings = args.findings
          if (args.changedPaths !== undefined) task.changedPaths = strList(args.changedPaths)
          if (args.acceptanceResults !== undefined) task.acceptanceResults = args.acceptanceResults
          if (args.commandsRun !== undefined) task.commandsRun = args.commandsRun
          task.updatedAt = now()
          if (TERMINAL.includes(task.status)) {
            task.attemptId = undefined
            // 已认领该任务的成员回到 idle
            const owner = task.assignee && task.assignee !== CAPTAIN_KEY ? memberByName(team, task.assignee) : null
            if (owner) owner.status = 'idle'
          }
          writeTeam(stateRoot, team)
          // 失败的质量任务 → 自动跟进（repair+复审 / 下一轮需求）
          if (task.status === 'failed' && taskKindOf(task) !== 'work') {
            const follow = planQualityFollowUp(team, task)
            if (follow.tasks.length > 0) {
              for (const row of follow.tasks) {
                team.taskSeq += 1
                const t = coerceTask(Object.assign({}, row, {
                  id: 't' + team.taskSeq,
                  description: row.subject,
                  status: 'pending', attempt: 0,
                  createdAt: now(), updatedAt: now(),
                }))
                team.tasks.push(t)
              }
              if (follow.escalated) team.escalated = true
              writeTeam(stateRoot, team)
              kickTeam(caller.wsRoot, team.id).catch(() => {})
              return 'Task ' + task.id + ' → ' + task.status + '. Quality loop opened follow-up tasks (' + follow.created.map((x) => x.kind + '-' + (x.round || 1)).join(', ') + ').'
            }
            if (follow.escalated) {
              team.escalated = true
              writeTeam(stateRoot, team)
            }
          }
          kickTeam(caller.wsRoot, team.id).catch(() => {})
          return 'Task ' + task.id + ' attempt ' + (task.attempt || 0) + ' → ' + task.status + (task.output ? '\nOutput: ' + task.output : '')
        },
      },

      // ═══ agent_teams_send_message（成员/队长）═══
      {
        name: 'agent_teams_send_message',
        description: 'Send a message to the captain or to a teammate. Messages go straight into the recipient\'s mailbox; member recipients are woken as their next turn. No relay is involved: teammates talk to each other directly.',
        usageGuide: '发送消息（成员/队长）。to 为 "captain" 或成员名；成员收件人会被唤醒为下一轮。',
        parameters: {
          type: 'object',
          properties: {
            to: { type: 'string', description: 'Recipient: "captain" or a member name.' },
            content: { type: 'string', description: 'The message text.' },
            from: { type: 'string', description: 'Sender (defaults to the caller; may not be set to another identity).' },
          },
          required: ['to', 'content'],
        },
        execute(args) {
          const caller = callerOf(args)
          const team = requireParticipantTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          const identity = requireIdentity(team, caller.convId)
          const to = String(args.to || '').trim()
          const from = identity.name
          if (args.from !== undefined && args.from !== from) throw new Error('"from" must be your own identity ("' + from + '"), not "' + args.from + '"')
          const message = createMessage(from, to, args.content)
          appendMailbox(stateRoot, team.id, to, Object.assign({}, message, { deliveryClaimedAt: now() }))
          if (to === CAPTAIN_KEY) {
            // 队长在线时尝试 live 投递（队长会话接续一轮）
            let delivered = 'mailbox'
            if (identity.kind === 'member') {
              try {
                ctx.agents.followup(team.captainSessionId, 'AgentTeams message from member ' + from + ':\n\n' + args.content)
                delivered = 'live'
              } catch (e) { delivered = 'mailbox' }
            }
            if (delivered === 'live') acknowledgeMailbox(stateRoot, team.id, CAPTAIN_KEY, [message.id])
            else releaseMailboxDelivery(stateRoot, team.id, CAPTAIN_KEY, [message.id])
            return 'Message ' + message.id + ' ' + from + ' → captain delivered via ' + delivered + '.'
          }
          if (team.halted === true) throw new Error('team "' + team.name + '" is halted; call agent_teams_resume before waking a member')
          const recipient = memberByName(team, to)
          if (!recipient) throw new Error('recipient "' + to + '" is not a member')
          let delivered = 'mailbox'
          if (recipient.id && recipient.status !== 'removed') {
            const senderText = from === CAPTAIN_KEY ? args.content : 'Message from team member ' + from + ':\n\n' + args.content
            const text = 'AgentTeams state policy: inspect ' + CFG.stateDir + '/' + team.id + '/ read-only; never edit team.json or inbox files directly. Use agent_teams_* tools for team state.\n\n' + senderText
            const ok = deliverToMember(caller.wsRoot, team.id, recipient, text)
            delivered = ok ? 'wake' : 'mailbox'
          }
          if (delivered === 'wake') acknowledgeMailbox(stateRoot, team.id, recipient.name, [message.id])
          else releaseMailboxDelivery(stateRoot, team.id, recipient.name, [message.id])
          return 'Message ' + message.id + ' ' + from + ' → ' + recipient.name + ' delivered via ' + delivered + '.'
        },
      },

      // ═══ agent_teams_status（成员/队长）═══
      {
        name: 'agent_teams_status',
        description: 'Report the current team state: phase, halted, members with live activity, task DAG with statuses, dependency blocks, mailbox warnings, and matrix coverage of the goal.',
        usageGuide: '查看团队全景状态（成员/队长）。含成员活跃度、任务 DAG、未满足依赖、邮箱未读数、目标覆盖矩阵。',
        parameters: { type: 'object', properties: {} },
        readOnly: true,
        execute(args) {
          const caller = callerOf(args)
          const team = requireParticipantTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          const identity = requireIdentity(team, caller.convId)
          const unsatisfied = {}
          team.tasks.forEach((t) => {
            const s = unsatisfiedDependencies(team.tasks, t.dependencies || [])
            if (s.length > 0) unsatisfied[t.id] = s
          })
          const members = team.members.filter((m) => m.status !== 'removed').map((m) => {
            let activity = 'unknown'
            try {
              const rec = ctx.agents.status(m.id)
              if (rec && rec.state) activity = rec.state === 'running' ? 'working' : 'idle'
            } catch (e) {}
            return m.name + (m.role ? '(' + m.role + ')' : '') + ': ' + m.status + '/' + activity + (m.model ? ' [' + m.model + ']' : '')
          })
          const tasks = team.tasks.map((t) => {
            const depNote = unsatisfied[t.id] ? ' (blocked by ' + unsatisfied[t.id].join(',') + ')' : ''
            return t.id + ' ' + t.status + (t.assignee ? ' @' + t.assignee : '') + (t.kind && t.kind !== 'work' ? ' [' + t.kind + (t.round ? ' r' + t.round : '') + ']' : '') + depNote + ' — ' + t.subject
          })
          const unread = identity.kind === 'captain' ? readUnreadMailbox(stateRoot, team.id, CAPTAIN_KEY).length : readUnreadMailbox(stateRoot, team.id, identity.name).length
          const coverage = buildCoverageMatrix(strList(team.description), team.tasks)
          const loop = describeQualityLoop(team)
          const lines = [
            'Team "' + team.name + '" (' + team.id + ') phase=' + (team.phase || 'running') + (team.halted ? ' HALTED' : '') + (team.escalated ? ' ESCALATED' : ''),
            'Goal: ' + ((team.description || '').trim() || '(not provided)'),
            '',
            'Members (' + members.length + '):',
          ].concat(members.length > 0 ? members.map((x) => '  - ' + x) : ['  (none)']).concat(
            ['',
              'Tasks (' + team.tasks.length + '):',
            ].concat(tasks.length > 0 ? tasks.map((x) => '  - ' + x) : ['  (none)']),
            ['', 'Your unread: ' + unread, 'Quality loop: ' + loop.summary],
          )
          if (coverage.length > 0) lines.push('Coverage:', ...coverage.map((c) => '  - ' + c.goal_item + ': ' + c.status + (c.task_ids.length ? ' (' + c.task_ids.join(', ') + ')' : '')))
          // 确认邮箱（仅本人收件箱）
          const ackIds = (identity.kind === 'captain' ? readMailbox(stateRoot, team.id, CAPTAIN_KEY) : readMailbox(stateRoot, team.id, identity.name))
            .filter((m) => m.readAt === undefined).map((m) => m.id)
          if (ackIds.length > 0) acknowledgeMailbox(stateRoot, team.id, identity.kind === 'captain' ? CAPTAIN_KEY : identity.name, ackIds)
          return lines.join('\n')
        },
      },

      // ═══ agent_teams_resume（队长）═══
      {
        name: 'agent_teams_resume',
        description: 'Explicitly resume a halted team. Requires a non-empty reason. Does not recreate cancelled tasks; only still-pending work is scheduled.',
        usageGuide: '恢复暂停的团队（队长）。需非空 reason。只调度仍处于 pending 的任务。',
        parameters: {
          type: 'object',
          properties: { reason: { type: 'string', description: 'Why the team is being resumed.' } },
          required: ['reason'],
        },
        execute(args) {
          const caller = callerOf(args)
          const team = requireCaptainTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          const resumed = resumeTeamState(team, args.reason)
          if (resumed.status === 'rejected') throw new Error(resumed.error || 'resume rejected')
          if (resumed.status === 'resumed') {
            team.halted = false
            team.haltedAt = undefined
            writeTeam(stateRoot, team)
            kickTeam(caller.wsRoot, team.id).catch(() => {})
          }
          return 'Team ' + team.id + (resumed.status === 'already_running' ? ' is already running.' : ' resumed (' + args.reason + ').')
        },
      },

      // ═══ agent_teams_delete（队长）═══
      {
        name: 'agent_teams_delete',
        description: 'End the team: cancel unfinished work, retire member sessions, and archive the full record (tasks and mailboxes stay on disk for later review).',
        usageGuide: '结束团队（队长）：取消未完成任务、退役成员会话、归档完整记录（归档到 .agent-teams/archive/）。',
        parameters: { type: 'object', properties: {} },
        execute(args) {
          const caller = callerOf(args)
          const team = requireCaptainTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          const roster = team.members.slice()
          for (const m of team.members) {
            if (m.status === 'removed') continue
            m.status = 'removed'
            for (const t of team.tasks) {
              if (t.assignee === m.name && !TERMINAL.includes(t.status)) invalidateTaskAttempt(t)
            }
          }
          writeTeam(stateRoot, team)
          recordRetiredMembers(stateRoot, roster.map((m) => m.id).filter((x) => x !== ''))
          for (const m of roster) {
            if (m.id && m.id !== '') { try { ctx.agents.stop(m.id) } catch (e) {} }
          }
          archiveTeamDir(stateRoot, team.id)
          return 'Team "' + team.name + '" deleted (archived).'
        },
      },
    ]
    for (const t of tools) { ctx.tools.register(t) }

    // ── HTTP 快照（面板轮询用）──
    function assembleTeamSnapshot(stateRoot, team, workspace, historic) {
      const tasks = team.tasks
      const activityByConv = {}
      try {
        const recs = ctx.agents.list({ team: team.id }) || []
        recs.forEach((r) => { activityByConv[r.convId] = r.state === 'running' ? 'working' : 'idle' })
      } catch (e) { /* 无活动信息 */ }
      const roster = historic ? team.members : team.members.filter((m) => m.status !== 'removed')
      const members = roster.map((m) => {
        const owned = tasks.filter((t) => t.assignee === m.name)
        const done = owned.filter((t) => t.status === 'completed').length
        const current = tasks.find((t) => t.status === 'in_progress' && t.assignee === m.name)
        const unread = readUnreadMailbox(stateRoot, team.id, m.name).length
        return {
          id: m.id, name: m.name, role: m.role || '',
          provider: m.provider || '', model: m.model || '', reasoningEffort: m.reasoningEffort || '',
          executionPrompt: m.executionPrompt || '', status: m.status,
          activity: m.status === 'removed' ? 'unknown' : (activityByConv[m.id] || 'idle'),
          progress: owned.length === 0 ? 0 : Math.round((done / owned.length) * 100),
          done: done, total: owned.length,
          currentTask: current ? current.id : '',
          unread: unread,
        }
      })
      const byId = {}
      tasks.forEach((t) => { byId[t.id] = t })
      const depthOf = (id, visiting) => {
        const t = byId[id]
        if (!t) return 0
        if ((visiting || new Set()).has(id)) return 0
        const vs = new Set(visiting || []); vs.add(id)
        const deps = t.dependencies || []
        if (deps.length === 0) return 0
        return 1 + Math.max(...deps.map((d) => depthOf(d, vs)))
      }
      const taskRows = tasks.map((t) => ({
        id: t.id, subject: t.subject, description: t.description || '', status: t.status,
        state: t.status === 'completed' ? 'completed' : t.status === 'failed' ? 'failed' : t.status === 'cancelled' ? 'cancelled' : (t.status === 'in_progress' || t.status === 'claimed') ? 'running' : (unsatisfiedDependencies(tasks, t.dependencies).length > 0 ? 'blocked' : 'open'),
        assignee: t.assignee || '', model: '',
        dependencies: t.dependencies || [], depth: depthOf(t.id),
        kind: t.kind, round: t.round, verdict: t.verdict,
      }))
      const captainMail = readMailbox(stateRoot, team.id, CAPTAIN_KEY)
      return {
        workspace: workspace || '', teamId: team.id, name: team.name,
        description: team.description || '', captainSessionId: team.captainSessionId,
        phase: team.phase || 'running', planReviewState: team.planReviewState,
        halted: team.halted === true, escalated: team.escalated === true,
        members: members, tasks: taskRows,
        messageCount: captainMail.length,
        captainInbox: captainMail.slice(-5).map((m) => ({ from: m.from, content: m.content })),
      }
    }
    function collectTeamsActivity() {
      const roots = []
      try {
        const list = ctx.fs.roots() || []
        list.forEach((r) => { if (r) roots.push(r) })
      } catch (e) { roots.push(SYS_WS_ROOT) }
      if (roots.length === 0 && SYS_WS_ROOT) roots.push(SYS_WS_ROOT)
      const out = []
      const seen = new Set()
      for (const ws of roots) {
        const stateRoot = stateRootOf(ws)
        for (const id of listTeamIds(stateRoot)) {
          if (seen.has(id)) continue
          const team = readTeam(stateRoot, id)
          if (!team) continue
          seen.add(id)
          out.push(assembleTeamSnapshot(stateRoot, team, ws, false))
        }
      }
      return out
    }
    // 面板：活动+归档（ctx.http 形态：fn(req) → {status, body, headers}）
    const renderJSON = (body, status) => {
      return { status: status || 200, body: JSON.stringify(body), headers: { 'content-type': 'application/json; charset=utf-8', 'cache-control': 'no-store' } }
    }
    const routeApi = (req) => {
      const path = req.path || ''
      const m = path.match(/^\/api\/agent-teams\/teams\/([^/]+)\/(approve|discard|continue|halt)$/)
      if (!m) return renderJSON({ error: 'not found' }, 404)
      const teamId = decodeURIComponent(m[1])
      const action = m[2]
      let payload = {}
      try { payload = req.body ? JSON.parse(req.body) : {} } catch (e) { payload = {} }
      const sessionId = typeof payload.sessionId === 'string' ? payload.sessionId.trim() : ''
      if (!sessionId) return renderJSON({ error: 'sessionId required' }, 400)
      const wsRoot = payload.wsRoot || SYS_WS_ROOT
      const stateRoot = stateRootOf(wsRoot)
      const team = findTeamByCaptain(stateRoot, sessionId)
      if (!team || team.id !== teamId) return renderJSON({ error: 'team not found for this captain' }, 404)
      try {
        if (action === 'approve') {
          const approved = approveStagedTeam(stateRoot, team)
          return renderJSON({ ok: true, phase: 'running', teamId: team.id, members: approved.members, tasks: approved.tasks })
        } else if (action === 'discard') {
          requireStaged(team)
          archiveTeamDir(stateRoot, team.id)
          return renderJSON({ ok: true, phase: 'archived', teamId: team.id })
        } else if (action === 'continue') {
          requireStaged(team)
          team.planReviewState = 'awaiting_feedback'
          writeTeam(stateRoot, team)
          return renderJSON({ ok: true, phase: 'staged', review: 'awaiting_feedback', teamId: team.id })
        } else if (action === 'halt') {
          for (const mrow of team.members) {
            if (mrow.id && mrow.status !== 'removed') { try { ctx.agents.stop(mrow.id) } catch (e) {} }
          }
          team.halted = true
          team.haltedAt = now()
          for (const t of team.tasks) {
            if (!TERMINAL.includes(t.status)) cancelUnfinishedTask(t, 'halted by user')
          }
          writeTeam(stateRoot, team)
          return renderJSON({ ok: true, halted: true, teamId: team.id })
        }
      } catch (e) {
        return renderJSON({ error: (e && e.message) || 'operation failed' }, 409)
      }
    }
    // 路由：GET 快照（列表 + 单团队）+ POST 操作（approve/discard/continue/halt）
    try {
      ctx.http.register('GET', '/api/agent-teams/teams/*', (req) => {
        const path = req.path || ''
        const m = path.match(/\/api\/agent-teams\/teams\/([^/]+)/)
        const teams = collectTeamsActivity()
        const filtered = m ? teams.filter((t) => t.teamId === decodeURIComponent(m[1])) : teams
        return renderJSON({ teams: filtered })
      })
    } catch (e) { log.warn('route teams: ' + (e && e.message || e)) }
    try { ctx.http.register('POST', '/api/agent-teams/teams/*', routeApi) } catch (e) { log.warn('route api: ' + (e && e.message || e)) }

    // ── 批准/废弃/计划编辑 + 通知队长 ──
    function notifyCaptain(teamId, text) {
      const team = findTeamByCaptain(stateRootOf(SYS_WS_ROOT), teamId) || findTeamByParticipant(stateRootOf(SYS_WS_ROOT), teamId)
      if (!team) return
      try { ctx.agents.followup(team.captainSessionId, text); return true } catch (e) { return false }
    }
    function approveStagedTeam(stateRoot, team) {
      requireStaged(team)
      if (team.members.filter((m) => m.status !== 'removed').length === 0) {
        throw new Error('cannot approve a team with no members')
      }
      team.phase = 'running'
      team.planReviewState = undefined
      team.approvedAt = now()
      // 校验任务 DAG 完整性（依赖 ID 存在）
      for (const t of team.tasks) {
        for (const dep of t.dependencies || []) {
          if (!taskById(team, dep)) throw new Error('dependency "' + dep + '" does not exist')
        }
      }
      writeTeam(stateRoot, team)
      // 原子 spawn 全部成员（welcome 首轮）
      for (const m of team.members) {
        if (m.status === 'removed') continue
        if (m.id === '') { m.id = 'conv_sub_' + sanitizeKey(team.id) + '_' + sanitizeKey(m.name) + '_' + Math.floor(Math.random() * 100000) }
        try { spawnMember(ctx, team, m, memberWelcome(team, m.name), team.captainSessionId) } catch (e) { log.warn('approve spawn failed ' + m.name + ': ' + e) }
      }
      writeTeam(stateRoot, team)
      const count = { members: team.members.filter((m) => m.status !== 'removed').length, tasks: team.tasks.length }
      kickTeam(team.wsRoot || SYS_WS_ROOT, team.id).catch(() => {})
      return count
    }

    // ── slash 命令（Round3 ④.2）：/agent-teams 状态快照（宿主 ctx.commands 面）──
    //   无参/status → 团队面板快照；其余子命令按需扩展。宿主不支持 commands 面
    //   时静默降级（插件其余功能不受影响）。
    try {
      ctx.commands.register({
        name: 'agent-teams',
        description: '团队面板/状态快照（/agent-teams [status]）',
        handler: (args) => {
          try {
            const sub = String((args && args.args) || '').trim().toLowerCase()
            const teams = collectTeamsActivity() || []
            if (sub === '' || sub === 'status') {
              // ★ 2026-08-31 按需激活：本命令即激活入口（宿主已激活并注入协议说明）。
              //   此处返回团队状态快照或引导创建首个团队。
              if (!teams.length) return '（当前无团队。你是队长：请用 agent_teams_create 创建团队（approval required），按协议完成建队、任务 DAG 与验收。）'
              const rows = teams.map((t) => ({
                id: t.id,
                name: t.name,
                phase: t.phase,
                members: (t.members || []).length,
                tasks: (t.tasks || []).length,
                status: t.status || '',
              }))
              return JSON.stringify({ teams: rows }, null, 2)
            }
            return '可用子命令：status（默认，团队状态快照）'
          } catch (e) {
            return 'agent-teams 命令执行失败: ' + (e && e.message || e)
          }
        },
      })
      log.info('slash 命令 /agent-teams 已注册')
    } catch (e) { log.warn('ctx.commands 注册失败（宿主不支持 commands 面，命令不可用）: ' + e) }

    // 队长会话中来自 Web 面板的批准动作 → 通过 client 方法（同时保留 HTTP 路由）
    ctx.registerClientMethod('getTeams', () => collectTeamsActivity())
    ctx.registerClientMethod('approve', (args) => {
      const teamId = args && args.teamId ? String(args.teamId) : ''
      if (!teamId) throw new Error('teamId required')
      const wsRoot = SYS_WS_ROOT
      const stateRoot = stateRootOf(wsRoot)
      const team = readTeam(stateRoot, teamId)
      if (!team) throw new Error('team not found')
      const count = approveStagedTeam(stateRoot, team)
      notifyCaptain(teamId, 'The staged AgentTeams plan was approved from the Web panel. The team is now running: ' + count.members + ' members, ' + count.tasks + ' tasks. Lead by delegation: monitor with agent_teams_status and collect results.')
      return { ok: true, teamId: teamId, members: count.members, tasks: count.tasks }
    })
    ctx.registerClientMethod('discard', (args) => {
      const teamId = args && args.teamId ? String(args.teamId) : ''
      if (!teamId) throw new Error('teamId required')
      const stateRoot = stateRootOf(SYS_WS_ROOT)
      const team = readTeam(stateRoot, teamId)
      if (!team) throw new Error('team not found')
      requireStaged(team)
      archiveTeamDir(stateRoot, team.id)
      notifyCaptain(teamId, 'The staged AgentTeams plan "' + team.name + '" was discarded from the Web panel. Do not recreate it; wait for an explicit user request.')
      return { ok: true, teamId: teamId }
    })
    ctx.registerClientMethod('continuePlanning', (args) => {
      const teamId = args && args.teamId ? String(args.teamId) : ''
      if (!teamId) throw new Error('teamId required')
      const stateRoot = stateRootOf(SYS_WS_ROOT)
      const team = readTeam(stateRoot, teamId)
      if (!team) throw new Error('team not found')
      requireStaged(team)
      team.planReviewState = 'awaiting_feedback'
      writeTeam(stateRoot, team)
      notifyCaptain(teamId, 'The user selected "Return to chat and revise" for the staged plan "' + team.name + '". Ask one concise question about what they want changed; after the answer, revise the same draft with one atomic agent_teams_edit_plan call.')
      return { ok: true, teamId: teamId }
    })
    ctx.registerClientMethod('halt', (args) => {
      const teamId = args && args.teamId ? String(args.teamId) : ''
      if (!teamId) throw new Error('teamId required')
      const stateRoot = stateRootOf(SYS_WS_ROOT)
      const team = readTeam(stateRoot, teamId)
      if (!team) throw new Error('team not found')
      for (const mrow of team.members) {
        if (mrow.id && mrow.status !== 'removed') { try { ctx.agents.stop(mrow.id) } catch (e) {} }
      }
      team.halted = true
      team.haltedAt = now()
      for (const t of team.tasks) {
        if (!TERMINAL.includes(t.status)) cancelUnfinishedTask(t, 'halted by user')
      }
      writeTeam(stateRoot, team)
      return { ok: true, halted: true, teamId: teamId }
    })
    ctx.registerClientMethod('deleteTask', (args) => {
      const teamId = args && args.teamId ? String(args.teamId) : ''
      const taskId = args && args.taskId ? String(args.taskId) : ''
      if (!teamId || !taskId) throw new Error('teamId and taskId required')
      const stateRoot = stateRootOf(SYS_WS_ROOT)
      const team = readTeam(stateRoot, teamId)
      if (!team) throw new Error('team not found')
      const target = taskById(team, taskId)
      if (!target) throw new Error('task "' + taskId + '" not found')
      if (!TERMINAL.includes(target.status)) throw new Error('task "' + taskId + '" is not terminal; only finished/failed/cancelled tasks can be deleted')
      team.tasks = team.tasks.filter((x) => x.id !== target.id)
      for (const other of team.tasks) {
        const deps = other.dependencies || []
        if (deps.includes(target.id)) other.dependencies = deps.filter((d) => d !== target.id)
        if (other.reviewedTaskId === target.id) other.reviewedTaskId = undefined
        if (other.sourceTaskId === target.id) other.sourceTaskId = undefined
      }
      writeTeam(stateRoot, team)
      return { ok: true, deleted: target.id, subject: target.subject }
    })

    // ── 系统提示注入（用法协议）──
    // ★ 2026-08-31 方案 B：协议段常驻（alwaysVisible）——即使插件未激活
    //   （工具未并入会话）也注入本段；段首为「引导」：无团队工具 = 未激活，
    //   用户想要多智能体协作时引导其执行 /agent-teams 激活。激活后工具全量
    //   可用、协议主体生效。
    const toolNames = MEMBER_DENIED_TOOLS.concat(['agent_teams_claim_task', 'agent_teams_update_task', 'agent_teams_send_message', 'agent_teams_status']).join(', ')
    ctx.systemPrompt.section({
      name: 'agent-teams',
      order: 117,
      text: buildUsageSection(toolNames),
      alwaysVisible: true,
    })

    log.info('agent-teams 插件已装载（stateDir=' + CFG.stateDir + ', maxMembers=' + CFG.maxMembers + ', codeMaxRounds=' + CFG.codeMaxRounds + '）')
  },
}

// 用法协议（模型提示词）——段首为「激活引导」（方案 B 常驻段）：
// 未激活时工具不可见，模型须引导用户执行 /agent-teams；激活后协议主体生效。
function buildUsageSection(allToolsText) {
  return [
    'AgentTeams (multi-agent team) is available in this session, but it may not be activated yet.',
    'SELF-CHECK: if you do NOT have agent_teams_* tools in your toolset, AgentTeams is NOT activated. When the user wants a multi-agent team (or mentions AgentTeams / team mode), tell them to run the /agent-teams command (optionally with a goal, e.g. /agent-teams build feature X); the activation will make the tools available and this protocol applies from the next step.',
    'When AgentTeams IS activated and the user asks to run something with AgentTeams (e.g. "use AgentTeams to do X"), you are the captain of a multi-agent team. Follow this protocol:',
    '1. Call agent_teams_create with a team name, the goal as description, and approval="required". This creates a staged plan and must not spawn members or schedule work. Use approval="automatic" only when the user explicitly asks to skip review and run immediately.',
    '2. Call agent_teams_add_member once per role the goal needs (researcher, engineer, reviewer, ...). In staging these are editable roster entries, not running subagents. By default a member snapshots your current model route; use a different route only when the goal or user requires it.',
    '3. Analyze the goal and create the smallest useful task DAG while staged. Every agent_teams_create_task call must include a non-empty subject, including verification and review tasks. Independent work should be parallel; dependencies are only genuine prerequisites. Finish the complete roster and DAG, tell the user the Web plan is ready, then end this turn. Never call agent_teams_approve during the planning turn. The user may click Approve & Run, explicitly approve in a later user turn, return to chat to request changes, or discard the plan.',
    '4. After approval, the final member configuration is spawned and the scheduler starts ready work. Lead by delegation: monitor with agent_teams_status, send guidance with agent_teams_send_message, and let idle teammates execute ready work. Do not duplicate a teammate\'s work merely because its turn is slow.',
    '5. If the user explicitly asks to pause a running member, its open attempt remains parked after interruption; after answering the user, send that same member guidance with agent_teams_send_message so it continues the same attempt. If work must change owner, restart from scratch, or be taken over, call agent_teams_reassign_task first. Use assignee=captain only for one ready task you will personally drive to a terminal status in this same turn; never end your turn with captain-owned work open.',
    '6. Tasks carry attempt_id capabilities. Members must use the current attempt_id for updates; stale-attempt errors mean ownership changed. Check status after progress notifications until every required task is terminal.',
    '7. Add repair or retry tasks when review/test fails, but never make a new task depend on a failed task. Do not send_message to start the next stage; the scheduler assigns ready work after approval. Watch every required task until it is terminal before deleting the team. Never perform a real deployment without explicit user confirmation.',
    '8. Quality kinds need a contract: non-empty objective and acceptance; implementation/repair also need inScope and verify. Review/requirements can complete only with verdict=pass; needs_revision/reject must fail with findings. The system then opens repair + next review. Do not approve your own implementation. Call agent_teams_resume with a reason to resume a halted team.',
    '9. ' + qualityPlanningPrompt(),
    '10. Present the team\'s results to the user, then agent_teams_delete the team unless the user wants to keep working.',
    'Tools: ' + allToolsText,
  ].join('\n')
}
