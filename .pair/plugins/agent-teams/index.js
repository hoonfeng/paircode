// ═══════════════════════════════════════════════════════════════
// agent-teams — 团队任务编排（单 Agent 多步执行；2026-09 去成员改造）
//
// ★ 架构（2026-09 用户决策）：**不使用子 Agent**。已去掉成员/角色/子会话
//   概念——当前会话（队长）就是唯一执行者，按任务 DAG 一步步执行：
//
//   建队(staged) → 用户批准 → 队长按依赖顺序 claim_task 领取就绪任务 →
//   用工具实际干活 → update_task 上报（质量门禁审计）→ 下一就绪任务 → …
//
//   长任务分段：宿主「工具调用轮次预算」（internal/agent/tool_budget.go，
//   默认 120 次/段）达到上限即结束当前执行段并自动开下一段（同会话续跑），
//   上下文由既有压缩机制控制——队长无需关心，多步执行自然持续。
//
// 保留的团队协作功能：
//   · 任务 DAG   带依赖的任务图（依赖未完成不可领取）；status 给出就绪任务
//   · 质量门禁   task kind 契约（objective/acceptance/verify/inScope）：
//               review/requirements 仅 verdict=pass 可完成；needs_revision
//               自动生成 repair+复审（上限 codeMaxRounds/maxRepairAttempts）
//   · 两阶段     create 默认 approval=required → staged（不执行任何任务），
//               用户经 Web 面板批准/废弃；approval=automatic 立即运行
//   · 面板/状态  Web 活动面板（任务 DAG/进度/批准按钮）+ status 工具
//   · 归档       delete/resume/halt（未完成任务取消、完整记录归档）
//
// 核心机制：
//   · 状态层   team.json（队长工作区 <wsRoot>/<stateDir>/<teamId>/）；
//              磁盘即真相（Web 面板轮询读取）
//   · 执行凭据 claim_task 返回 attempt_id；update_task 必须携带（防陈旧写入）
//   · 身份     仅队长（创建团队的会话）可操作全部 agent_teams_* 工具
//
// 依赖宿主能力（inject 声明）：
//   fs / logger / timer / agents / app / http / commands
// ═══════════════════════════════════════════════════════════════

const PLUGIN = 'agent-teams'

// ─── 常量 ────────────────────────────────────────────────────
const TERMINAL = ['completed', 'failed', 'cancelled']
const TASK_KINDS = ['requirements', 'implementation', 'verification', 'review', 'repair', 'integration', 'work']
const TRANSITIONS = {
  pending: ['claimed', 'cancelled'],
  // ★ 2026-09 单 Agent 改造：claimed 可直接 completed/failed（队长领取后一口气做完，
  //   不必先落一次 in_progress；in_progress 仍作为长任务的中间态保留）。
  claimed: ['in_progress', 'completed', 'failed', 'cancelled'],
  in_progress: ['completed', 'failed', 'cancelled'],
  completed: [],
  failed: [],
  cancelled: [],
}

// ─── 命名团队模板（R2-6：profiles；2026-09 去成员：只保留任务种子）───
// create(profile=…) 一键建队：按模板 seed 任务 DAG（队长可后续 edit_plan 调整）。
// profiles 上限 MAX_TEAM_PROFILES=16（参考实现同值）。
// 内置模板：
//   · default         研发质量流水线（需求分析→实施→验证→审查；纯任务 DAG）
//   · captain-planning 空任务图（任务 DAG 由队长在 staged 阶段现场设计）
const MAX_TEAM_PROFILES = 16
const TEAM_PROFILES = [
  {
    name: '默认任务模板',
    description: '研发质量流水线：需求分析→实施→独立验证→审查（任务 DAG，队长逐步执行）',
    tasks: [
      { ref: 'req', subject: '需求分析：目标拆解与验收标准定义', kind: 'requirements', objective: '把团队目标拆解为可验收的需求（范围/验收/风险）', acceptance: ['需求覆盖团队目标', '验收标准可测', '含风险清单'], round: 1, deps: [] },
      { ref: 'impl', subject: '实施：按需求落地改动并自测', kind: 'implementation', objective: '按需求实施改动，带自测/验证', inScope: [], acceptance: ['需求项全部实现', '改动可回滚', '有测试或验证'], verify: [], deps: ['req'] },
      { ref: 'ver', subject: '独立验证：构建/冒烟/回归核对', kind: 'verification', objective: '独立验证实施结果（构建/装载/行为）', acceptance: ['验证命令全部通过', '异常项已说明'], verify: [], deps: ['impl'] },
      { ref: 'rev', subject: '代码审查：质量门与需求对齐', kind: 'review', objective: '审查实施是否满足需求且无阻塞问题', acceptance: ['审查覆盖全部改动', '结论有依据'], reviewedTaskRef: 'impl', round: 1, deps: ['impl'] },
    ],
  },
  {
    name: 'captain-planning',
    description: '队长规划型模板：空任务图，DAG 由队长在 staged 阶段现场设计',
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
    dependencies: strList(t && t.dependencies),
    output: trimOpt(t && t.output),
    attempt: Number((t && t.attempt) || 0),
    attemptId: trimOpt(t && t.attemptId),
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
function activateTaskAttempt(task) {
  task.status = 'claimed'
  task.attemptId = uuid()
  task.output = undefined
  task.updatedAt = now()
}
function beginTaskAttempt(task) {
  task.attempt = (task.attempt || 0) + 1
  activateTaskAttempt(task)
}
function invalidateTaskAttempt(task) {
  task.attemptId = undefined
  task.status = 'pending'
  task.output = undefined
  task.updatedAt = now()
}
function cancelUnfinishedTask(task, output) {
  if (TERMINAL.includes(task.status)) return
  task.status = 'cancelled'
  task.attemptId = undefined
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
  const files = unresolved.map((f) => f.file).filter((f) => typeof f === 'string' && f !== '')
  const repair = {
    kind: 'repair',
    subject: 'repair-round-' + nextRound,
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
  const review = {
    kind: 'review',
    subject: 'review-round-' + nextRound,
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
    'A pass verdict requires evidence: review findings must be grounded in the actual diff/commands; a failed immediate review or verification opens the repair/review loop automatically.',
  ].join(' ')
}


return {
  name: 'agent-teams',
  // ctx.webServer 供 HTTP 路由；此处不需要（ctx.http 就够）——inject 声明按需
  // agents：仅用于「面板批准后唤醒队长会话」（ctx.agents.followup 投递；不是派生
  //         子 Agent——子 Agent 全局默认关闭，本插件不 spawn 任何成员会话）
  inject: ['fs', 'logger', 'timer', 'agents', 'app', 'http', 'commands'],
  purpose: '团队任务编排（单 Agent 多步执行，2026-09 去成员）：任务 DAG + 质量门禁 + 两阶段批准 + Web 面板；不使用子 Agent',
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
      codeMaxRounds: Number(settings.codeMaxRounds || DEFAULT_REVIEW_POLICY.codeMaxRounds),
      maxRepairAttempts: Number(settings.maxRepairAttempts || DEFAULT_REVIEW_POLICY.maxRepairAttempts),
      requirementsMaxRounds: Number(settings.requirementsMaxRounds || DEFAULT_REVIEW_POLICY.requirementsMaxRounds),
    }
    if (CFG.codeMaxRounds <= 0) CFG.codeMaxRounds = DEFAULT_REVIEW_POLICY.codeMaxRounds
    if (CFG.maxRepairAttempts <= 0) CFG.maxRepairAttempts = DEFAULT_REVIEW_POLICY.maxRepairAttempts

    // 注册配置段（设置面板可见）
    try {
      ctx.registerSettings({
        key: 'agent-teams',
        title: '团队任务编排（agent-teams）',
        fields: [
          { name: 'stateDir', label: '团队状态目录（工作区相对）', type: 'text', default: '.agent-teams' },
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
    function requireCaptainTeam(wsRoot, captainId) {
      const team = findTeamByCaptain(stateRootOf(wsRoot), captainId)
      if (!team) throw new Error('you do not lead a team; call agent_teams_create first')
      return team
    }
    function requireStaged(team) {
      if (team.phase !== 'staged') throw new Error('team "' + team.name + '" is not staged')
    }


    // ── 就绪任务 / 进度（单 Agent 多步执行的驱动视图）──
    // 就绪 = pending 且依赖全部 completed（可被 claim_task 领取）。
    function readyTaskList(team) {
      return (team.tasks || []).filter((t) =>
        t.status === 'pending' && unsatisfiedDependencies(team.tasks, t.dependencies || []).length === 0)
    }
    function blockedTaskList(team) {
      return (team.tasks || []).filter((t) =>
        t.status === 'pending' && unsatisfiedDependencies(team.tasks, t.dependencies || []).length > 0)
    }
    function teamProgress(team) {
      const count = (fn) => (team.tasks || []).filter(fn).length
      return {
        total: (team.tasks || []).length,
        completed: count((t) => t.status === 'completed'),
        failed: count((t) => t.status === 'failed'),
        cancelled: count((t) => t.status === 'cancelled'),
        inProgress: count((t) => t.status === 'claimed' || t.status === 'in_progress'),
        ready: readyTaskList(team).length,
        blocked: blockedTaskList(team).length,
      }
    }
    // nextReadyHint：update_task/claim_task 返回值里的「下一步」提示（多步执行驱动）。
    function nextReadyHint(team) {
      const ready = readyTaskList(team)
      if (ready.length > 0) {
        return ready.slice(0, 3).map((t) => t.id + ' (' + (t.subject || '') + ')').join('; ')
      }
      const blocked = blockedTaskList(team)
      if (blocked.length > 0) return '(none ready; ' + blocked.length + ' blocked by dependencies)'
      return '(none pending)'
    }

    // ── 工具注册（10 个：队长单 Agent 多步执行；无成员/子 Agent）──
    const tools = [
      // ═══ agent_teams_create（队长）═══
      {
        name: 'agent_teams_create',
        description: 'Create a team: a task DAG you (the current session) will execute yourself, step by step. Use approval=required for a two-phase plan: tasks stay unclaimed until the user reviews the Web plan and explicitly approves it. approval=automatic preserves the legacy immediate-execution path. Optionally pass profile=<name> to seed tasks from a named template: available profiles are ' + profileNames() + '.',
        usageGuide: '创建团队（当前会话成为队长=唯一执行者）。默认 approval=required：先建暂存计划，用户批准后按 DAG 开始执行。profile 可选：一键 seed 任务模板（契约字段需按需补全）。',
        parameters: {
          type: 'object',
          properties: {
            name: { type: 'string', description: 'Name for the new team (used as its stable id).' },
            description: { type: 'string', description: 'Team purpose / the goal the team will work on.' },
            profile: { type: 'string', description: 'Optional named team template to seed tasks: ' + profileNames() + '. Defaults to none (empty task graph).' },
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
            tasks: [], taskSeq: 0,
            phase: approval === 'automatic' ? 'running' : 'staged',
            planReviewState: approval === 'automatic' ? undefined : 'awaiting_review',
            wsRoot: caller.wsRoot,
            reviewPolicy: { codeMaxRounds: CFG.codeMaxRounds, maxRepairAttempts: CFG.maxRepairAttempts, requirementsMaxRounds: CFG.requirementsMaxRounds },
          }
          // ★ profiles：套用命名模板（seed 任务 DAG；队长可后续 edit_plan 调整契约字段）
          const prof = profileByName(args.profile)
          if (prof) {
            const refToId = {}
            const created = []
            for (const pt of prof.tasks || []) {
              team.taskSeq += 1
              const task = coerceTask({
                id: 't' + team.taskSeq, subject: pt.subject,
                description: pt.description || '', status: 'pending',
                dependencies: [],
                kind: pt.kind || 'work', round: pt.round ? Number(pt.round) : undefined,
                objective: pt.objective, acceptance: pt.acceptance,
                inScope: pt.inScope, outOfScope: pt.outOfScope, verify: pt.verify,
                createdAt: now(), updatedAt: now(),
              })
              if (pt.ref) refToId[pt.ref] = task.id
              created.push({ task, pt })
              team.tasks.push(task)
            }
            // 两遍 seed：先建全部任务并登记 ref→id，再按 deps/reviewedTaskRef 静态接线
            for (const { task, pt } of created) {
              task.dependencies = uniq((pt.deps || []).map((r) => refToId[r]).filter((x) => x))
              if (pt.reviewedTaskRef && refToId[pt.reviewedTaskRef]) {
                task.reviewedTaskId = refToId[pt.reviewedTaskRef]
              }
              // review 契约要求 reviewedTaskId：seed 模板可能未显式接线 → 落到首依赖
              if (taskKindOf(task) === 'review' && !task.reviewedTaskId && task.dependencies.length > 0) {
                task.reviewedTaskId = task.dependencies[0]
              }
              if (task.dependencies.length > 0 || task.reviewedTaskId) task.updatedAt = now()
            }
          }
          F.mkdir(stateRoot + '/' + teamId, true)
          writeTeam(stateRoot, team)
          emitChangeSignal()
          const seeded = prof ? ' from profile "' + prof.name + '" (' + team.tasks.length + ' seed tasks)' : ''
          if (approval === 'automatic') {
            return 'Team "' + name + '" created (' + teamId + ') in running mode' + seeded + '. Call agent_teams_status for ready tasks, then claim + execute them one by one.'
          }
          return 'Team "' + name + '" created (' + teamId + ') as a staged plan' + seeded + '. Design the task DAG with agent_teams_create_task (or edit_plan; seed tasks may need contract fields like inScope/verify), then tell the user the Web plan is ready for review.'
        },
      },

      // ═══ agent_teams_edit_plan（队长，staged 专用）═══
      {
        name: 'agent_teams_edit_plan',
        description: 'Atomically revise the current staged plan without executing anything. Submit dependent edits in order (update downstream dependencies first, then remove tasks, then add tasks). Actions: {action:"update_task",taskId,subject,description?,dependencies} | {action:"add_task",subject,description?,dependencies} | {action:"remove_task",taskId}.',
        usageGuide: '原子批量修订暂存计划（staged 阶段）。actions：update_task / add_task / remove_task（先改下游依赖，再删任务）。',
        parameters: {
          type: 'object',
          properties: {
            mutations: {
              type: 'array',
              items: { type: 'object', properties: {} },
              description: 'Ordered batch of edits: {action:"update_task",taskId,subject,description?,dependencies} | {action:"add_task",subject,description?,dependencies} | {action:"remove_task",taskId}.',
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
            if (action === 'update_task') {
              const t = taskById(team, String(op.taskId || ''))
              if (!t) throw new Error('task "' + op.taskId + '" not found')
              if (t.status !== 'pending' || (t.attempt || 0) !== 0) throw new Error('task "' + t.id + '" has already started and cannot be edited')
              const subject = trimOpt(op.subject)
              if (!subject) throw new Error('task subject must not be empty')
              t.subject = subject
              t.description = trimOpt(op.description)
              t.dependencies = uniq(strList(op.dependencies))
              t.updatedAt = now()
            } else if (action === 'add_task') {
              const subject = trimOpt(op.subject)
              if (!subject) throw new Error('task subject must not be empty')
              team.taskSeq += 1
              team.tasks.push(Object.assign(coerceTask({}), {
                id: 't' + team.taskSeq, subject: subject,
                description: trimOpt(op.description),
                status: 'pending',
                dependencies: uniq(strList(op.dependencies)),
                attempt: 0, kind: 'work',
                createdAt: now(), updatedAt: now(),
              }))
            } else if (action === 'remove_task') {
              const idx = team.tasks.findIndex((t) => t.id === String(op.taskId || ''))
              if (idx < 0) throw new Error('task "' + op.taskId + '" not found')
              team.tasks.splice(idx, 1)
            } else {
              throw new Error('unknown plan action: ' + action)
            }
          }
          writeTeam(stateRoot, team)
          emitChangeSignal()
          return 'Plan revised (' + team.tasks.length + ' tasks). Reconfirm with the user before approve.'
        },
      },

      // ═══ agent_teams_approve（队长）═══
      {
        name: 'agent_teams_approve',
        description: 'Approve and start a staged plan. Call this only in response to an explicit user approval in a new user turn; never call it during the turn that created or edited the plan. There is no member spawn: after approval YOU execute the task DAG yourself — agent_teams_status for ready tasks, claim_task, do the work, update_task, until every required task is terminal.',
        usageGuide: '批准暂存计划并开始执行（队长）。必须在新用户轮次得到用户明确批准后调用；批准后按 status 的就绪任务逐个 claim→执行→update_task。',
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
          const ready = readyTaskList(team)
          const hint = ready.length > 0
            ? ' Ready now: ' + ready.map((t) => t.id + ' (' + t.subject + ')').join('; ') + '. Claim the first with agent_teams_claim_task and start working — do not stop to ask again.'
            : ' No task is ready (check dependencies) — inspect with agent_teams_status.'
          return 'Team "' + team.name + '" approved: ' + count.tasks + ' tasks queued. You are the only executor; work the DAG step by step.' + hint
        },
      },

      // ═══ agent_teams_create_task（队长）═══
      {
        name: 'agent_teams_create_task',
        description: 'Create a task in your team\'s task graph. Every call must include a non-empty subject, including verification and review tasks. Tasks can depend on other tasks (dependencies): a task is only claimable once every dependency is completed. Quality kinds need a contract (see parameters); review requires reviewedTaskId, repair requires sourceTaskId.',
        usageGuide: '创建任务（队长）。subject 必填；dependencies 未完成前不可领取；审查/验证任务也要有 subject；质量种类需契约字段。',
        parameters: {
          type: 'object',
          properties: {
            subject: { type: 'string', description: 'Required non-empty title for this task.' },
            description: { type: 'string', description: 'What needs to be done, in detail.' },
            dependencies: { type: 'array', items: { type: 'string' }, description: 'Task ids this task depends on (must be completed before this task can be claimed).' },
            kind: { type: 'string', enum: ['requirements', 'implementation', 'verification', 'review', 'repair', 'integration', 'work'], description: 'Task kind. Defaults to work (legacy, no quality gates).' },
            round: { type: 'integer', description: '1-based review / requirements / repair round.' },
            objective: { type: 'string', description: 'Required non-empty objective for quality kinds.' },
            inScope: { type: 'array', items: { type: 'string' }, description: 'Workspace-relative POSIX paths this task may change.' },
            outOfScope: { type: 'array', items: { type: 'string' }, description: 'Workspace-relative POSIX paths this task must not change.' },
            acceptance: { type: 'array', items: { type: 'string' }, description: 'Acceptance criteria. Required for quality kinds.' },
            verify: { type: 'array', items: { type: 'string' }, description: 'Verification commands. Required for implementation/repair.' },
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
          const kind = gate.kind || 'work'
          team.taskSeq += 1
          const task = coerceTask({
            id: 't' + team.taskSeq, subject: String(args.subject).trim(),
            description: trimOpt(args.description), status: 'pending',
            dependencies: deps,
            kind: kind, round: args.round ? Number(args.round) : undefined,
            objective: gate.objective, acceptance: gate.acceptance,
            inScope: gate.inScope, verify: gate.verify,
            outOfScope: strList(args.outOfScope),
            reviewedTaskId: trimOpt(args.reviewedTaskId),
            sourceTaskId: trimOpt(args.sourceTaskId), sourceFindingIds: strList(args.sourceFindingIds),
            coverageOf: strList(args.coverageOf),
            createdAt: now(), updatedAt: now(),
          })
          team.tasks.push(task)
          writeTeam(stateRoot, team)
          emitChangeSignal()
          const ready = task.status === 'pending' && deps.length === 0
          return 'Task "' + task.subject + '" created as ' + task.id + ' (status ' + task.status + ', ' + kind + (ready ? ', ready to claim' : ', blocked until ' + deps.join(', ') + ' completed') + ').'
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
          emitChangeSignal()
          return 'Task "' + target.id + '" (' + target.subject + ', ' + target.status + ') deleted; dangling dependency references cleared.'
        },
      },

      // ═══ agent_teams_claim_task（队长）═══
      {
        name: 'agent_teams_claim_task',
        description: 'Claim a ready task before working on it (you are the only executor). The returned attempt_id is the current execution capability: pass it to every agent_teams_update_task call for this attempt. Dependencies must be completed first; a claimed task cannot be claimed again.',
        usageGuide: '领取就绪任务（队长）。依赖必须已完成；返回 attempt_id 为执行凭据，后续 update_task 必须携带。',
        parameters: {
          type: 'object',
          properties: {
            task_id: { type: 'string', description: 'The task id to claim.' },
          },
          required: ['task_id'],
        },
        execute(args) {
          const caller = callerOf(args)
          const team = requireCaptainTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          const task = taskById(team, String(args.task_id || ''))
          if (!task) throw new Error('task "' + args.task_id + '" not found')
          if (task.status === 'claimed' || task.status === 'in_progress') {
            return 'Task ' + task.id + ' is already claimed (attempt ' + (task.attempt || 0) + ', attempt_id ' + (task.attemptId || '') + ', status ' + task.status + '). Continue working and finish with agent_teams_update_task using that attempt_id.'
          }
          if (task.status !== 'pending') throw new Error('task ' + task.id + ' cannot be claimed from "' + task.status + '"')
          const unsatisfied = unsatisfiedDependencies(team.tasks, task.dependencies || [])
          if (unsatisfied.length > 0) throw new Error('task ' + task.id + ' dependencies unsatisfied: ' + unsatisfied.join(', '))
          beginTaskAttempt(task)
          writeTeam(stateRoot, team)
          emitChangeSignal()
          return 'Task ' + task.id + ' claimed (attempt ' + task.attempt + ', attempt_id ' + task.attemptId + ', status ' + task.status + '). Subject: ' + task.subject + '. Do the work now, then call agent_teams_update_task with the same attempt_id.'
        },
      },

      // ═══ agent_teams_update_task（队长）═══
      {
        name: 'agent_teams_update_task',
        description: 'Update a task status/output. Include the current attempt_id returned by claim_task (stale attempts are rejected); terminal results are immutable. Completed quality-kind tasks must submit their contract evidence. Failed review/requirements automatically open the repair + next-review loop.',
        usageGuide: '更新任务状态/输出（队长）。必须携带当前 attempt_id；终态不可变；质量种类完成需提交契约证据（verdict/findings/changedPaths/acceptanceResults/commandsRun）。',
        parameters: {
          type: 'object',
          properties: {
            task_id: { type: 'string', description: 'The task id to update.' },
            status: { type: 'string', enum: ['in_progress', 'completed', 'failed', 'cancelled'], description: 'New status (in_progress, completed, failed, cancelled).' },
            output: { type: 'string', description: 'Result summary; set when completing or failing.' },
            attempt_id: { type: 'string', description: 'Current execution capability returned by claim_task.' },
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
          const team = requireCaptainTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          const task = taskById(team, String(args.task_id || ''))
          if (!task) throw new Error('task "' + args.task_id + '" not found')
          if (task.attemptId !== undefined && args.attempt_id !== task.attemptId) {
            throw new Error('stale attempt for task ' + task.id + ': expected attempt_id ' + task.attemptId + ' (claim_task again if the attempt was invalidated)')
          }
          const nextStatus = args.status
          if (TERMINAL.includes(task.status)) {
            const sameStatus = nextStatus === undefined || nextStatus === task.status
            const sameOutput = args.output === undefined || args.output === task.output
            if (!sameStatus || !sameOutput) throw new Error('terminal task ' + task.id + ' is immutable')
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
          if (TERMINAL.includes(task.status)) task.attemptId = undefined
          writeTeam(stateRoot, team)
          emitChangeSignal()
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
              emitChangeSignal()
              return 'Task ' + task.id + ' → ' + task.status + '. Quality loop opened follow-up tasks (' + follow.created.map((x) => x.kind + '-' + (x.round || 1)).join(', ') + '). Next ready: ' + nextReadyHint(team)
            }
            if (follow.escalated) {
              team.escalated = true
              writeTeam(stateRoot, team)
              emitChangeSignal()
            }
          }
          const progresInfo = 'Next ready: ' + nextReadyHint(team)
          return 'Task ' + task.id + ' attempt ' + (task.attempt || 0) + ' → ' + task.status + (task.output ? '\nOutput: ' + task.output : '') + '\n' + progresInfo
        },
      },

      // ═══ agent_teams_status（队长）═══
      {
        name: 'agent_teams_status',
        description: 'Report the current team state: phase, halted/escalated flags, progress counts, ready-to-execute tasks (claimable right now), the task DAG with statuses/dependency blocks, and matrix coverage of the goal. Use it to pick the next task after every update.',
        usageGuide: '查看团队全景状态（队长）。含进度计数、就绪任务（现在可领取）、任务 DAG、依赖阻塞、质量循环、目标覆盖矩阵。',
        parameters: { type: 'object', properties: {} },
        readOnly: true,
        execute(args) {
          const caller = callerOf(args)
          const team = requireCaptainTeam(caller.wsRoot, caller.convId)
          const unsatisfied = {}
          team.tasks.forEach((t) => {
            const s = unsatisfiedDependencies(team.tasks, t.dependencies || [])
            if (s.length > 0) unsatisfied[t.id] = s
          })
          const ready = readyTaskList(team)
          const prog = teamProgress(team)
          const tasks = team.tasks.map((t) => {
            const depNote = unsatisfied[t.id] ? ' (blocked by ' + unsatisfied[t.id].join(',') + ')' : ''
            const contract = taskKindOf(t) !== 'work' ? ' [' + taskKindOf(t) + (t.round ? ' r' + t.round : '') + (t.verdict ? ' ' + t.verdict : '') + ']' : ''
            return t.id + ' ' + t.status + contract + depNote + ' — ' + t.subject
          })
          const coverage = buildCoverageMatrix(strList(team.description), team.tasks)
          const loop = describeQualityLoop(team)
          const lines = [
            'Team "' + team.name + '" (' + team.id + ') phase=' + (team.phase || 'running') + (team.halted ? ' HALTED' : '') + (team.escalated ? ' ESCALATED' : ''),
            'Goal: ' + ((team.description || '').trim() || '(not provided)'),
            'Executor: captain (this session) — single agent, step-by-step DAG execution',
            'Progress: ' + prog.completed + '/' + prog.total + ' completed, ' + prog.inProgress + ' in progress, ' + prog.ready + ' ready, ' + prog.blocked + ' blocked' + (prog.failed ? ', ' + prog.failed + ' failed' : '') + (prog.cancelled ? ', ' + prog.cancelled + ' cancelled' : ''),
            '',
            'Ready to execute now (' + ready.length + '):',
          ].concat(ready.length > 0 ? ready.map((t) => '  - ' + t.id + ': ' + t.subject) : ['  (none)']).concat(
            ['',
              'Tasks (' + team.tasks.length + '):',
            ].concat(tasks.length > 0 ? tasks.map((x) => '  - ' + x) : ['  (none)']),
            ['', 'Quality loop: ' + loop.summary],
          )
          if (coverage.length > 0) lines.push('Coverage:', ...coverage.map((c) => '  - ' + c.goal_item + ': ' + c.status + (c.task_ids.length ? ' (' + c.task_ids.join(', ') + ')' : '')))
          if (ready.length > 0) lines.push('', 'Next step: agent_teams_claim_task ' + ready[0].id + ' → do the work → agent_teams_update_task with the returned attempt_id.')
          return lines.join('\n')
        },
      },

      // ═══ agent_teams_resume（队长）═══
      {
        name: 'agent_teams_resume',
        description: 'Explicitly resume a halted team. Requires a non-empty reason. Does not recreate cancelled tasks; only still-pending work is claimable afterwards.',
        usageGuide: '恢复暂停的团队（队长）。需非空 reason。只恢复仍处于 pending 的任务（已取消任务不重建）。',
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
            emitChangeSignal()
          }
          return 'Team ' + team.id + (resumed.status === 'already_running' ? ' is already running.' : ' resumed (' + args.reason + '). Next ready: ' + nextReadyHint(team))
        },
      },

      // ═══ agent_teams_delete（队长）═══
      {
        name: 'agent_teams_delete',
        description: 'End the team: cancel unfinished work and archive the full record (team.json stays on disk under archive/ for later review).',
        usageGuide: '结束团队（队长）：取消未完成任务、归档完整记录（归档到 <stateDir>/archive/）。',
        parameters: { type: 'object', properties: {} },
        execute(args) {
          const caller = callerOf(args)
          const team = requireCaptainTeam(caller.wsRoot, caller.convId)
          const stateRoot = stateRootOf(caller.wsRoot)
          finishTeam(stateRoot, team)
          return 'Team "' + team.name + '" deleted (archived).'
        },
      },
    ]
    for (const t of tools) { ctx.tools.register(t) }

    // ── HTTP 快照（面板轮询用；单 Agent 执行：无成员/邮箱）──
    function assembleTeamSnapshot(stateRoot, team, workspace) {
      const tasks = team.tasks
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
        dependencies: t.dependencies || [], depth: depthOf(t.id),
        kind: t.kind, round: t.round, verdict: t.verdict,
      }))
      return {
        workspace: workspace || '', teamId: team.id, name: team.name,
        description: team.description || '', captainSessionId: team.captainSessionId,
        phase: team.phase || 'running', planReviewState: team.planReviewState,
        halted: team.halted === true, escalated: team.escalated === true,
        progress: teamProgress(team),
        readyTasks: readyTaskList(team).map((t) => ({ id: t.id, subject: t.subject })),
        tasks: taskRows,
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
          out.push(assembleTeamSnapshot(stateRoot, team, ws))
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
      const m = path.match(/^\/api\/agent-teams\/teams\/([^/]+)\/(approve|discard|continue|halt|delete)$/)
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
          return renderJSON({ ok: true, phase: 'running', teamId: team.id, tasks: approved.tasks, ready: readyTaskList(team).map((t) => t.id) })
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
          team.halted = true
          team.haltedAt = now()
          for (const t of team.tasks) {
            if (!TERMINAL.includes(t.status)) cancelUnfinishedTask(t, 'halted by user')
          }
          writeTeam(stateRoot, team)
          emitChangeSignal()
          return renderJSON({ ok: true, halted: true, teamId: team.id })
        } else if (action === 'delete') {
          finishTeam(stateRoot, team, 'The AgentTeams team "' + team.name + '" was deleted; its record is archived. Stop executing its tasks and do not recreate it unless the user explicitly asks.')
          return renderJSON({ ok: true, archived: true, teamId: team.id })
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

    // ── 批准/废弃 + 通知队长 ──
    // notifyCaptain：向队长会话投递一条消息（面板操作后唤醒队长继续执行）。
    //   ★ 这不是派生子 Agent：投递到已存在的队长会话（ctx.agents.followup）。
    //   失败不阻塞面板操作（用户仍可在对话里继续）。
    function notifyCaptain(teamId, text) {
      const team = findTeamByCaptain(stateRootOf(SYS_WS_ROOT), teamId)
      if (!team) return false
      try { ctx.agents.followup(team.captainSessionId, text); return true } catch (e) {
        log.warn('notifyCaptain 投递失败（不影响团队状态）: ' + ((e && e.message) || e))
        return false
      }
    }
    // finishTeam：结束团队（面板「删除团队」/ HTTP delete 路由 / agent_teams_delete 工具共用）。
    // 取消所有未完成任务 → 写回记录（广播）→（可选）向队长投递通知（必须在归档前——
    // 归档后 findTeamByCaptain 不再能找到该团队）→ 完整记录归档到 <stateDir>/archive/。
    function finishTeam(stateRoot, team, note) {
      for (const t of team.tasks) {
        if (!TERMINAL.includes(t.status)) cancelUnfinishedTask(t, 'team deleted')
      }
      writeTeam(stateRoot, team)
      if (note) notifyCaptain(team.id, note)
      archiveTeamDir(stateRoot, team.id)
    }
    // approveStagedTeam：staged → running（不 spawn 任何东西；队长自行执行 DAG）。
    function approveStagedTeam(stateRoot, team) {
      requireStaged(team)
      if (team.tasks.length === 0) {
        throw new Error('cannot approve a team with no tasks; build the task DAG first')
      }
      // 校验任务 DAG 完整性（依赖 ID 存在）
      for (const t of team.tasks) {
        for (const dep of t.dependencies || []) {
          if (!taskById(team, dep)) throw new Error('dependency "' + dep + '" does not exist')
        }
      }
      team.phase = 'running'
      team.planReviewState = undefined
      team.approvedAt = now()
      writeTeam(stateRoot, team)
      emitChangeSignal()
      return { tasks: team.tasks.length }
    }

    // ── slash 命令（Round3 ④.2）：/agent-teams 状态快照（宿主 ctx.commands 面）──
    //   无参/status → 团队面板快照；其余子命令按需扩展。宿主不支持 commands 面
    //   时静默降级（插件其余功能不受影响）。
    try {
      ctx.commands.register({
        name: 'agent-teams',
        description: '启动 AgentTeams 任务编排（/agent-teams <目标>；/agent-teams status 看状态）',
        handler: (args) => {
          try {
            // ★ 对齐 dsh（src/command.ts buildActivationDirective）：
            //   `/agent-teams <目标>` 返回**激活指令**（含目标）——宿主据此唤醒
            //   agent，队长当轮即按协议建队。子命令 status = 纯查询（不唤醒）。
            const raw = String((args && args.args) || '').trim()
            const sub = raw.toLowerCase()
            const teams = collectTeamsActivity() || []
            const snapshot = () => {
              if (!teams.length) return '（当前无团队。）'
              const rows = teams.map((t) => ({
                id: t.teamId,
                name: t.name,
                phase: t.phase,
                tasks: (t.tasks || []).length,
                progress: t.progress || {},
                ready: (t.readyTasks || []).map((x) => x.id),
              }))
              return JSON.stringify({ teams: rows }, null, 2)
            }
            if (sub === 'status') return snapshot()
            const lines = [
              'The user invoked an AgentTeams slash command. Activate the AgentTeams protocol from your instructions now: you are the captain and the only executor (no member subagents are spawned).',
              'Call agent_teams_create with approval="required". Build the complete staged task DAG, then stop and ask the user to review the Web plan. Do not approve or start it in this same turn.',
            ]
            if (raw === '') {
              lines.push('The goal was not given — ask the user what the team should accomplish.')
            } else {
              lines.push('Goal: ' + raw)
            }
            const snap = snapshot()
            if (snap && snap !== '（当前无团队。）') lines.push('Current teams: ' + snap)
            return lines.join('\n')
          } catch (e) {
            return 'agent-teams 命令执行失败: ' + (e && e.message || e)
          }
        },
      })
      log.info('slash 命令 /agent-teams 已注册')
    } catch (e) { log.warn('ctx.commands 注册失败（宿主不支持 commands 面，命令不可用）: ' + e) }

    // 面板动作 → 队长会话 client 方法（同时保留 HTTP 路由）
    ctx.registerClientMethod('getTeams', () => collectTeamsActivity())
    ctx.registerClientMethod('approve', (args) => {
      const teamId = args && args.teamId ? String(args.teamId) : ''
      if (!teamId) throw new Error('teamId required')
      const stateRoot = stateRootOf(SYS_WS_ROOT)
      const team = readTeam(stateRoot, teamId)
      if (!team) throw new Error('team not found')
      const count = approveStagedTeam(stateRoot, team)
      const ready = readyTaskList(team).map((t) => t.id + ' (' + t.subject + ')').join('; ')
      notifyCaptain(teamId, 'The staged AgentTeams plan was approved from the Web panel. The team is now running: ' + count.tasks + ' tasks queued. Ready now: ' + (ready || '(none)') + '. Execute the DAG yourself, step by step: agent_teams_status → agent_teams_claim_task → do the work → agent_teams_update_task; keep going until every required task is terminal.')
      return { ok: true, teamId: teamId, tasks: count.tasks }
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
      team.halted = true
      team.haltedAt = now()
      for (const t of team.tasks) {
        if (!TERMINAL.includes(t.status)) cancelUnfinishedTask(t, 'halted by user')
      }
      writeTeam(stateRoot, team)
      emitChangeSignal()
      return { ok: true, halted: true, teamId: teamId }
    })
    // 面板「删除团队」：结束并归档整个团队（等价 agent_teams_delete / HTTP delete 路由）。
    // 与 halt 的区别：halt 只暂停（团队仍在面板）；本方法取消未完成任务 + 归档，
    // 团队从活动面板消失（向队长投递「已删除」通知，归档前投递）。
    ctx.registerClientMethod('deleteTeam', (args) => {
      const teamId = args && args.teamId ? String(args.teamId) : ''
      if (!teamId) throw new Error('teamId required')
      const stateRoot = stateRootOf(SYS_WS_ROOT)
      const team = readTeam(stateRoot, teamId)
      if (!team) throw new Error('team not found')
      finishTeam(stateRoot, team, 'The AgentTeams team "' + team.name + '" was deleted from the Web panel; its record is archived. Stop executing its tasks and do not recreate it unless the user explicitly asks.')
      return { ok: true, teamId: teamId, archived: true, name: team.name }
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
      emitChangeSignal()
      return { ok: true, deleted: target.id, subject: target.subject }
    })

    // ── 系统提示注入（用法协议）──
    // ★ 2026-08-31 方案 B：协议段常驻（alwaysVisible）——即使插件未激活
    //   （工具未并入会话）也注入本段；段首为「引导」：无团队工具 = 未激活，
    //   用户想要团队协作时引导其执行 /agent-teams 激活。激活后工具全量
    //   可用、协议主体生效。
    // ★ 2026-09 去成员：队长=唯一执行者；长任务由宿主工具调用轮次预算自动
    //   分段续跑（agentloop/tool_budget.go），协议要求队长持续推进 DAG。
    const toolNames = [
      'agent_teams_create', 'agent_teams_edit_plan', 'agent_teams_approve',
      'agent_teams_create_task', 'agent_teams_delete_task',
      'agent_teams_claim_task', 'agent_teams_update_task',
      'agent_teams_status', 'agent_teams_resume', 'agent_teams_delete',
    ].join(', ')
    ctx.systemPrompt.section({
      name: 'agent-teams',
      order: 117,
      text: buildUsageSection(toolNames),
      alwaysVisible: true,
    })

    log.info('agent-teams 插件已装载（stateDir=' + CFG.stateDir + ', codeMaxRounds=' + CFG.codeMaxRounds + ', 单 Agent 多步执行/无成员）')
  },
}

// 用法协议（模型提示词）——段首为「激活引导」（方案 B 常驻段）：
// 未激活时工具不可见，模型须引导用户执行 /agent-teams；激活后协议主体生效。
function buildUsageSection(allToolsText) {
  return [
    'AgentTeams (task-DAG team orchestration; single-agent step-by-step execution) is available in this session, but it may not be activated yet.',
    'SELF-CHECK: if you do NOT have agent_teams_* tools in your toolset, AgentTeams is NOT activated. When the user wants a structured team/task workflow (or mentions AgentTeams / team mode), tell them to run the /agent-teams command (optionally with a goal, e.g. /agent-teams build feature X); the activation will make the tools available and this protocol applies from the next step.',
    'When AgentTeams IS activated and the user asks to run something with AgentTeams (e.g. "use AgentTeams to do X"), you are the captain and the ONLY executor — no member subagents are spawned. Follow this protocol:',
    '1. Call agent_teams_create with a team name, the goal as description, and approval="required". This creates a staged plan and must not start any work. Use approval="automatic" only when the user explicitly asks to skip review and run immediately.',
    '2. Analyze the goal and build the smallest useful task DAG while staged (agent_teams_create_task / agent_teams_edit_plan). Every task needs a non-empty subject, including verification and review tasks. Dependencies are only genuine prerequisites; independent work can be planned side by side (you still execute it yourself, in dependency order).',
    '3. Finish the complete DAG, tell the user the Web plan is ready, then end this turn. Never call agent_teams_approve during the planning turn. The user may click Approve & Run, explicitly approve in a later user turn, return to chat to request changes, or discard the plan.',
    '4. After approval, execute the DAG yourself, step by step: agent_teams_status → agent_teams_claim_task <ready id> → do the real work with your tools → agent_teams_update_task with the returned attempt_id → next ready task. Repeat until every required task is terminal. Never stop early to summarize while ready tasks remain.',
    '5. Long work is segmented automatically by the host: after the tool-call budget (default 120 calls) the current execution segment ends and the next segment starts immediately in the same session (context compression is automatic). Never wrap up just because the turn feels long — keep going until the DAG is done or you are truly blocked.',
    '6. Quality kinds need a contract: requirements/implementation/verification/review/repair/integration need non-empty objective + acceptance; implementation/repair also need inScope + verify. Review/requirements complete only with verdict=pass; needs_revision/reject must fail with findings — the system then opens repair + the next review automatically.',
    '7. Evidence discipline: completed implementation/repair needs changedPaths (every path in scope) and passed acceptanceResults/commandsRun for every contract item; a failed verification must fail the task. Do not mark work completed that you have not verified.',
    '8. Never make a new task depend on a failed task; open repair/retry tasks instead. Call agent_teams_resume with a reason to resume a halted team.',
    '9. ' + qualityPlanningPrompt(),
    '10. Present the results to the user, then agent_teams_delete the team unless the user wants to keep working. Never perform a real deployment without explicit user confirmation.',
    'Tools: ' + allToolsText,
  ].join('\n')
}
