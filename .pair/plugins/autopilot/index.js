// ═══════════════════════════════════════════════════════════════
// autopilot — 自主模式（监督者 / 「人」角色）
//
// ★ 一切皆插件：策略在插件（角色提示词 / 任务书 / 裁决语义 / 记录落盘 / 看板数据），
//   能力在宿主（Go）：
//     · ctx.loopFactory.registerAutopilot({id, decide}) —— 决策器注册位：工作 agent
//       每次自然结束（无 tool_call + 有正文）时，宿主在会话续轮处调用 decide(req)：
//       continue + task → 宿主把指令作为新任务唤醒工作 agent；done → 整轮收尾。
//       （宿主实现见 internal/agent/autopilot.go；旧「任务队列驱动下一阶段」已删除）
//     · ctx.subagent.run(spec) —— 「子 agent 回合」能力：独立系统提示/任务书/工具白名单/
//       独立历史/事件带来源标注（AgentName=supervisor，前端看板分区），返回轨迹与结论。
//       见 internal/agent/subagent.go。
//
// 工作流（一条消息的自然生命周期）：
//   工作 agent 执行 → 自然结束 → 本插件 decide()：
//     ① 构造任务书（用户目标 / 工作 agent 本轮汇报 / 运行统计 / 最近工作记录）
//     ② 派生「监督者」回合（可调全部工具，自行核查证据：read/grep/git/exec_command…）
//     ③ 监督者调 submit_result 提交裁决（action=done|continue + 评判 + 下一步指令 + 证据）
//     ④ 本插件把该回合记录落盘（.pair/autopilot/<convId>.jsonl），并经
//        ui:autopilot:round 事件 + /api/autopilot/rounds 接口供看板展示（可溯源）
//     ⑤ 把裁决交还宿主：continue → 唤醒工作 agent 继续；done → 收尾
//
// 可观测/可溯源（看板）：
//   · 实时：监督者回合的事件（thinking/content/tool_call/tool_result）由宿主经 WS 下发，
//     带 agentName=supervisor + turn=回合号 → 前端按来源分区渲染；
//   · 历史：本插件接口按会话返回全部监督回合（工作 agent 汇报 + 评判 + 指令 + 轨迹 +
//     用量/耗时/结束方式），刷新页面后仍可回看。
// ═══════════════════════════════════════════════════════════════

const SETTINGS_KEY = 'autopilot'
const REC_DIR = '.pair/autopilot'
const AGENT_NAME = 'supervisor'
const SUBMIT_TOOL = 'submit_result'
const MAX_STORED_ROUNDS = 200      // 单会话最多保留的监督回合记录（滚动截断）
const TRACE_SEG_MAX = 240          // 记录中最多保留的轨迹段数
const TRACE_TEXT_MAX = 4000        // 单段文本（thinking/content）截断
const TRACE_RESULT_MAX = 2000      // 单段工具结果截断
const TASK_HISTORY_MAX = 24        // 任务书里最近工作记录最多条数
const TASK_HISTORY_CHARS = 800     // 任务书里单条记录截断

// ── 语言锁定（策略）──────────────────────────────────────────
// ★ 为什么必须显式声明：子 agent 回合的系统提示**完全由插件给出**——宿主不拼内核
//   DefaultSystemPrompt（internal/agent/subagent.go：sys = spec.System），也不经插件装配器链
//   （同文件 SkipPluginAssembly）。内核提示与 agentloop 的系统提示都带「第一铁律：语言锁定
//   （中文）」，本插件原先只有角色职责 → 实测推理模型的思考过程（thinking）全英文，
//   而同回合的评判/指令因 schema 描述为中文而呈中文（.pair/autopilot/*.jsonl 实证）。
//   故此处显式前置，且独立于 promptOverride：用户覆盖角色提示也不丢语言约束。
const LANGUAGE_LOCK = [
  '## ⚠️ 第一铁律：语言锁定（中文）',
  '无论工具返回了什么代码、终端输出、英文文档或其他内容，你的思考过程（thinking）与一切输出',
  '（评判 assessment / 下一步指令 next_task / 证据 evidence / 正文）都必须使用中文——这是不可违背的铁律。',
  '工具输出中的英文是工作内容的一部分，不代表你的语言可以切换到英文；',
  '代码标识符、命令、文件路径、专有名词可保留原文。',
  '如果发现自己的思考变成了英文，立即停下并切换回中文。',
].join('\n')

// ── 角色提示（策略；可经设置项覆盖）────────────────────────────
const SUPERVISOR_SYSTEM = [
  '# 角色',
  '你是「人」——项目负责人 / 验收人，不是干活的那个 AI。你监督一个 AI 工作 agent 执行用户的任务：',
  '审核它的产出、评判质量、决定下一步该做什么。',
  '',
  '## 职责',
  '1. **审核**：读清工作 agent 的汇报与证据；必要时**亲自核查**——你有全部工具',
  '   （read / glob / grep / exec_command / git_* 等），关键结论必须落到证据上',
  '   （文件内容、命令输出、测试结果）。不要凭它的自述点头。',
  '2. **评判**：说清做成了什么、哪里不达标、有什么遗漏与风险。不要客套、不要复述它的汇报。',
  '3. **决策**（必须明确表态）：任务达成且你核查过关键证据 → 判定完成；',
  '   未达成 / 有遗留 / 证据不足 / 质量不合格 → 判定继续，并给出**具体、可执行**的下一步指令。',
  '',
  '## 工作方式',
  '- 你只做审核、评判、指挥与必要的核查；大规模实现工作交给工作 agent（避免与它重复劳动）。',
  '- 不要为了「显得在工作」而反复空转：核查为决策服务，够了就表决。',
  '- 判断要有依据：它说「测试通过」→ 你去看测试输出；它说「已修复」→ 你去看改动。',
  '- 结束时**必须**调用 ' + SUBMIT_TOOL + ' 提交裁决（只在正文里写结论不算完成）。',
  '',
  '## 纪律',
  '- 不放过「看起来完成了但没验证」的交付：宁可要求它补证据，也不放行未验证的结论。',
  '- 也不无限挑刺：目标已达成、遗留不影响交付时，判定完成并说明依据。',
].join('\n')

// submit_result 参数（裁决结构，宿主按此收集并结束本轮监督）
const SUBMIT_PARAMS = {
  type: 'object',
  properties: {
    action: {
      type: 'string',
      enum: ['done', 'continue'],
      description: 'done=任务已达成（收尾）；continue=需要工作 agent 继续（next_task 必填）',
    },
    assessment: {
      type: 'string',
      description: '评判：做成了什么 / 哪里不达标 / 遗漏与风险（给用户看的结论，不复述汇报）',
    },
    next_task: {
      type: 'string',
      description: 'action=continue 时必填：给工作 agent 的具体、可执行指令（做什么、验收标准、别再做什么）',
    },
    evidence: {
      type: 'string',
      description: '你核查过的证据摘要（读过哪些文件 / 跑过什么命令 / 输出关键行）',
    },
  },
  required: ['action', 'assessment'],
}

// ── 设置 ──────────────────────────────────────────────────────
function settingFields() {
  return [
    { name: 'maxSuperviseRounds', label: '监督轮数上限', type: 'number', default: 20,
      hint: '单条消息内最多监督几轮（每轮 = 工作 agent 结束一次后的一次审核）；超过即自动收尾。0=内核默认 20' },
    { name: 'stepBudget', label: '监督者单回合步数上限', type: 'number', default: 40,
      hint: '一次监督回合内最多走多少步（一步 = 一次模型调用）' },
    { name: 'toolCallBudget', label: '监督者单回合工具调用上限', type: 'number', default: 40,
      hint: '一次监督回合内最多执行多少次工具调用（核查用）' },
    { name: 'timeoutMinutes', label: '单次监督超时（分钟）', type: 'number', default: 15,
      hint: '一次监督回合的最长运行时间；超时按「未裁决」收尾（避免卡死会话）' },
    { name: 'promptOverride', label: '监督者角色提示（覆盖）', type: 'textarea', default: '',
      hint: '留空使用插件内置角色提示（「人」/验收人）；填写则整体替换（语言锁定铁律仍然生效）' },
    { name: 'storeTrace', label: '记录监督者轨迹', type: 'checkbox', default: true,
      hint: '把监督者回合的思考/工具调用轨迹一并落盘（看板溯源用；关闭可减小记录体积）' },
    { name: 'commandHint', label: '任务书附加要求（可选）', type: 'textarea', default: '',
      hint: '追加到每次监督任务书末尾的自定义要求（如「必须跑 go test ./... 才算验证」）' },
  ]
}

// ── 小工具 ────────────────────────────────────────────────────
function num(v, def) {
  const n = Number(v)
  return Number.isFinite(n) && n > 0 ? n : def
}

function truncText(s, max) {
  s = String(s == null ? '' : s)
  return s.length <= max ? s : s.slice(0, max) + '…（已截断）'
}

function safeConvId(id) {
  return String(id == null ? '' : id).replace(/[^0-9a-zA-Z_-]/g, '') || 'unknown'
}

function parseQuery(raw) {
  const out = {}
  const q = String(raw == null ? '' : raw).replace(/^\?/, '')
  if (!q) return out
  for (const part of q.split('&')) {
    if (!part) continue
    const i = part.indexOf('=')
    const k = decodeURIComponent(i < 0 ? part : part.slice(0, i))
    const v = i < 0 ? '' : decodeURIComponent(part.slice(i + 1).replace(/\+/g, ' '))
    out[k] = v
  }
  return out
}

// roleLabel 历史消息角色 → 中文标签（任务书里可读）
function roleLabel(m) {
  const r = String((m && m.role) || '')
  if (r === 'user') return '用户/指令'
  if (r === 'assistant') return '工作 agent'
  if (r === 'tool') return '工具 ' + String((m && m.name) || '')
  if (r === 'system') return '系统'
  return r || '未知'
}

// buildHistoryText 最近工作记录 → 文本（供监督者核查上下文）
function buildHistoryText(history) {
  const arr = Array.isArray(history) ? history : []
  const use = arr.slice(-TASK_HISTORY_MAX)
  if (use.length === 0) return '（无）'
  const lines = []
  for (const m of use) {
    const body = truncText(String((m && m.content) || '').replace(/\s+\n/g, '\n').trim(), TASK_HISTORY_CHARS)
    if (!body) continue
    lines.push('[' + roleLabel(m) + '] ' + body)
  }
  return lines.length ? lines.join('\n\n') : '（无）'
}

// buildTask 构造监督任务书（策略）
function buildTask(req, cfg) {
  const round = num(req.round, 1)
  const maxRounds = num(req.maxRounds, 0)
  const objective = String(req.objective || '').trim() || '（未能取到原始任务，请按最近工作记录判断）'
  const report = String(req.workerReport || '').trim() || '（工作 agent 本轮没有正文汇报）'
  const extra = String((cfg && cfg.commandHint) || '').trim()
  const parts = [
    '# 监督回合（第 ' + round + ' 次' + (maxRounds > 0 ? '，上限 ' + maxRounds + ' 次' : '') + '）',
    '',
    '## 用户目标',
    truncText(objective, 4000),
    '',
    '## 工作 agent 本轮汇报',
    truncText(report, 6000),
    '',
    '## 运行统计',
    '- 工作 agent 已完成轮数（Run 次数）：' + num(req.workerTurns, 0),
    '- 本次是第 ' + round + ' 次监督' + (maxRounds > 0 ? '（上限 ' + maxRounds + '）' : ''),
    '',
    '## 最近工作记录（时间正序，供核查参考）',
    buildHistoryText(req.recentHistory),
    '',
    '## 你的任务',
    '1. **审核**：上面的汇报是否可信？关键结论是否有证据？（你有全部工具，可自行核查：',
    '   读文件确认改动、跑测试/构建、git diff 看差异、grep 查引用……）',
    '2. **评判**：做成了什么、哪里不达标、遗漏与风险（不复述汇报）。',
    '3. **决定下一步**：',
    '   - 任务已达成且你核查过关键证据 → action="done"',
    '   - 未达成 / 有遗留 / 证据不足 / 质量不合格 → action="continue"，并在 next_task 里给出',
    '     给工作 agent 的具体指令（要做什么、验收标准是什么、别再做哪些无效动作）',
    '',
    '★ 结束时必须调用 ' + SUBMIT_TOOL + ' 提交裁决（action / assessment / next_task / evidence）。',
    '★ 思考过程（thinking）与评判、指令一律用中文（见系统提示「语言锁定」铁律）。',
    '★ 这是第 ' + round + ' 次监督：若反复要求继续而问题始终不收敛，请在评判中说明卡点并给出收窄范围的指令。',
  ]
  if (extra) {
    parts.push('', '## 附加要求（用户配置）', extra)
  }
  return parts.join('\n')
}

// extractVerdict 从监督回合结果中取裁决（提交工具优先，其次正文 JSON，最后兜底收尾）
function extractVerdict(res, errText) {
  const submitted = res && res.submitted && typeof res.submitted === 'object' ? res.submitted : null
  if (submitted) {
    return normalizeVerdict(submitted, 'submitted')
  }
  const content = String((res && res.content) || '')
  const json = lastJSONObject(content)
  if (json && (json.action || json.next_task || json.assessment)) {
    return normalizeVerdict(json, 'content')
  }
  return {
    action: 'done',
    assessment: errText
      ? '监督回合执行失败（' + errText + '），按完成收尾以免卡住会话。'
      : '监督者未提交明确裁决（未调用 ' + SUBMIT_TOOL + '），按完成收尾以免卡住会话。',
    nextTask: '',
    evidence: '',
    source: 'fallback',
  }
}

// lastJSONObject 取正文中最后一个可解析的 JSON 对象（监督者偶尔只在正文里写结论）
function lastJSONObject(text) {
  const s = String(text || '')
  const end = s.lastIndexOf('}')
  if (end < 0) return null
  const start = s.lastIndexOf('{', end)
  if (start < 0) return null
  try {
    const v = JSON.parse(s.slice(start, end + 1))
    return v && typeof v === 'object' ? v : null
  } catch (e) {
    return null
  }
}

function normalizeVerdict(v, source) {
  const action = String(v.action || '').trim().toLowerCase() === 'continue' ? 'continue' : 'done'
  const nextTask = String(v.next_task || v.nextTask || '').trim()
  return {
    action: action === 'continue' && !nextTask ? 'done' : action, // 说继续但没给指令 → 无法唤醒，收尾
    assessment: String(v.assessment || '').trim(),
    nextTask,
    evidence: String(v.evidence || '').trim(),
    source,
  }
}

// ── 记录落盘 / 读取（看板数据面）──────────────────────────────
function recordPath(convId) {
  return REC_DIR + '/' + safeConvId(convId) + '.jsonl'
}

function saveRound(ctx, convId, record) {
  try {
    if (!ctx.fs.exists(REC_DIR)) ctx.fs.mkdir(REC_DIR, true)
    ctx.fs.appendFile(recordPath(convId), JSON.stringify(record) + '\n')
    trimRounds(ctx, convId)
  } catch (e) {
    ctx.logger('autopilot').warn('监督回合记录落盘失败 conv=' + convId + ' err=' + (e && e.message ? e.message : e))
  }
}

// trimRounds 记录数超过上限时滚动截断（保留最近 MAX_STORED_ROUNDS 条）
function trimRounds(ctx, convId) {
  try {
    const p = recordPath(convId)
    const text = ctx.fs.readFile(p)
    const lines = text.split('\n').filter((l) => l.trim())
    if (lines.length <= MAX_STORED_ROUNDS) return
    ctx.fs.writeFile(p, lines.slice(lines.length - MAX_STORED_ROUNDS).join('\n') + '\n')
  } catch (e) {
    // 截断失败不影响主流程（下次落盘再试）
  }
}

function loadRounds(ctx, convId) {
  try {
    const text = ctx.fs.readFile(recordPath(convId))
    const out = []
    for (const line of text.split('\n')) {
      const s = line.trim()
      if (!s) continue
      try {
        out.push(JSON.parse(s))
      } catch (e) {
        // 跳过损坏行
      }
    }
    return out
  } catch (e) {
    return []
  }
}

// ── 轨迹压缩（记录体积控制；看板溯源保留结构）──────────────────
function compactTrace(segments) {
  const arr = Array.isArray(segments) ? segments : []
  const use = arr.length > TRACE_SEG_MAX ? arr.slice(arr.length - TRACE_SEG_MAX) : arr
  return use.map((s) => {
    const t = String((s && s.type) || '')
    if (t === 'tool_call') {
      return {
        type: 'tool_call',
        name: String(s.name || ''),
        callId: String(s.callId || ''),
        args: truncText(s.argsRaw || '', TRACE_RESULT_MAX),
        result: truncText(s.result || '', TRACE_RESULT_MAX),
      }
    }
    return { type: t, content: truncText(s && s.content, TRACE_TEXT_MAX) }
  })
}

// ── 决策器（宿主在每次工作 agent 自然结束时调用）────────────────
function decide(ctx, req) {
  const log = ctx.logger('autopilot')
  const cfg = ctx.getSettings(SETTINGS_KEY) || {}
  const round = num(req.round, 1)
  const startedAt = new Date().toISOString()
  const t0 = Date.now()

  // 语言锁定前置（最强位置）：覆盖 promptOverride 场景，角色提示被整体替换也不丢中文约束
  const system = LANGUAGE_LOCK + '\n\n' + (String(cfg.promptOverride || '').trim() || SUPERVISOR_SYSTEM)
  const task = buildTask(req, cfg)

  let res = null
  let errText = ''
  try {
    res = ctx.subagent.run({
      name: AGENT_NAME,
      round: round,
      system: system,
      task: task,
      tools: null, // 全部工具（用户选择：监督者可直接下场核查）
      stepBudget: num(cfg.stepBudget, 40),
      toolCallBudget: num(cfg.toolCallBudget, 40),
      timeoutMs: num(cfg.timeoutMinutes, 15) * 60000,
      resultTool: {
        name: SUBMIT_TOOL,
        description: '提交本轮监督裁决（调用后本轮监督结束）',
        parameters: SUBMIT_PARAMS,
      },
    })
  } catch (e) {
    errText = String(e && e.message ? e.message : e)
    log.warn('监督回合执行失败 conv=' + req.convId + ' round=' + round + ' err=' + errText)
  }

  const verdict = extractVerdict(res, errText)
  const record = {
    round: round,
    convId: String(req.convId || ''),
    workspaceRoot: String(req.workspaceRoot || ''),
    startedAt: startedAt,
    endedAt: new Date().toISOString(),
    durationMs: Date.now() - t0,
    // 工作 agent 侧（看板左栏：实际工作输出）
    objective: String(req.objective || ''),
    workerReport: String(req.workerReport || ''),
    workerTurns: num(req.workerTurns, 0),
    // 监督者侧（看板右栏：自主 agent 的工作）
    action: verdict.action,
    assessment: verdict.assessment,
    nextTask: verdict.nextTask,
    evidence: verdict.evidence,
    verdictSource: verdict.source,
    steps: res ? num(res.steps, 0) : 0,
    toolCalls: res ? num(res.toolCalls, 0) : 0,
    promptTokens: res ? num(res.promptTokens, 0) : 0,
    outputTokens: res ? num(res.outputTokens, 0) : 0,
    ended: res ? String(res.ended || '') : 'error',
    error: errText || String((res && res.error) || ''),
    trace: cfg.storeTrace === false ? [] : compactTrace(res && res.segments),
  }

  saveRound(ctx, req.convId, record)
  // 实时下发给前端 client 半（看板可即时追加；无 client 半时无副作用）
  try {
    ctx.emit('ui:autopilot:round', record)
  } catch (e) {
    // 忽略：事件桥不可用时看板走 HTTP 拉取
  }
  log.info('监督回合 #' + round + ' 裁决=' + verdict.action +
    ' 耗时=' + record.durationMs + 'ms 步数=' + record.steps + ' 工具=' + record.toolCalls +
    (verdict.nextTask ? ' 下一步=' + truncText(verdict.nextTask, 60) : ''))

  if (!verdict.nextTask || verdict.action !== 'continue') {
    return { action: 'done', assessment: verdict.assessment }
  }
  // 指令加来源前缀：它作为工作 agent 的新任务落进对话历史（可溯源，见前端对话流）
  return {
    action: 'continue',
    assessment: verdict.assessment,
    task: '【监督者指令·第 ' + round + ' 次监督】\n\n' + verdict.nextTask +
      (verdict.assessment ? '\n\n> 监督者评判：' + verdict.assessment : ''),
  }
}

// ── 插件装配 ──────────────────────────────────────────────────
return {
  name: 'autopilot',
  inject: ['fs', 'logger'],
  purpose: '自主模式（监督者/「人」角色）：工作 agent 自然结束时审核/评判/决定下一步，' +
    '可派生带全部工具的监督者回合自行核查；回合记录落盘 + 看板数据接口',
  apply(ctx) {
    const log = ctx.logger('autopilot')

    // ① 配置项（监督轮数/预算/超时/角色提示覆盖/轨迹记录）
    ctx.registerSettings({
      key: SETTINGS_KEY,
      title: '自主模式',
      fields: settingFields(),
    })

    // ② 监督轮数上限 → 宿主 LoopOpts（装配器覆盖；宿主会话续轮据此收敛）
    ctx.loopFactory.register((opts) => {
      const cfg = ctx.getSettings(SETTINGS_KEY) || {}
      const n = Number(cfg.maxSuperviseRounds)
      if (Number.isFinite(n) && n > 0) return { maxSuperviseRounds: n }
      return null
    })

    // ③ 决策器注册：工作 agent 每次自然结束，宿主调用本实现
    ctx.loopFactory.registerAutopilot({
      id: 'autopilot',
      decide: async (req) => decide(ctx, req),
    })

    // ④ 看板数据接口：按会话返回全部监督回合（刷新页面后仍可回看/溯源）
    //   注：ctx.http handler 的返回值会被宿主按 {status, body, headers} 解析——
    //   body 必须是字符串（返回裸对象会得到空响应体），故此处显式 JSON.stringify。
    ctx.http.register('GET', '/api/autopilot/rounds', (req) => {
      const q = parseQuery(req && req.query)
      const convId = q.convId || ''
      const respond = (obj) => ({
        status: 200,
        body: JSON.stringify(obj),
        headers: { 'Content-Type': 'application/json' },
      })
      if (!convId) return respond({ ok: false, error: '缺少 convId', rounds: [] })
      const rounds = loadRounds(ctx, convId)
      return respond({ ok: true, convId: convId, total: rounds.length, rounds: rounds })
    })

    log.info('自主模式已装配：监督者决策器已注册（工作 agent 每次自然结束时审核/评判/决定下一步）')
  },
}
