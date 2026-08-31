// agent-events.js: agent 事件处理共享模块
// 从 RightPanel.vue 的 chatSubscribe 回调中提取，供 App.vue 的 WebSocket 回调调用。
// 支持：thinking/content/tool_call/tool_result/approval/error/usage/phase/notice/compacted/circling/evaluation/done
//
// 设计要点：
// - 状态写入全局 state（messagesByConv/loadingByConv/agentRunningByConv/approvalByConv/phaseByConv/wsTokenStats/convCtxStatsByConv/nudgeByConv）
// - UI 相关回调（scrollToBottom/loadWsTokenStats/autoNameConv/saveConvMsg/onTaskReplace 等）通过 setGlobalCtx 注册
// - RightPanel 在 onMounted 调用 setGlobalCtx 注册回调；App.vue 的 WebSocket onmessage 调用 processAgentEvent

import { state } from './ui-state.js'
import { reactive } from 'vue'

// ─── 按 convId 存储运行时状态（非响应式，普通对象）──
// { convId: { msgIdx, finalContent, lastUserText } }
const runtimes = {}

let msgKeyCounter = 0
function makeMsgKey() { return 'msg_' + Date.now() + '_' + (msgKeyCounter++) }

// normalizeAskType 归一化 ask_user 的 askType 变体（与后端 parseAskArgs 对齐）：
// single_with_input / single-input / choice-with-input 等 → single-with-input；unknown → text。
export function normalizeAskType(raw) {
  const s = String(raw || '').trim().toLowerCase().replace(/_/g, '-')
  if (s === 'single' || s === 'choice' || s === 'radio' || s === 'single-choice') return 'single'
  if (s === 'multi' || s === 'multiple' || s === 'checkbox' || s === 'multi-choice' || s === 'multi-select') return 'multi'
  if (s === 'single-with-input' || s === 'single-with-custom' || s === 'choice-with-input' || s === 'single-input') return 'single-with-input'
  return 'text'
}

function pushSegment(segs, type, initial) {
  const last = segs[segs.length - 1]
  if (last && last.type === type) return last
  const seg = { type, content: '', ...initial }
  segs.push(seg)
  return seg
}

// ─── 全局 UI 回调集合（由 RightPanel onMounted 注册）──
// {
//   scrollToBottom(convId),        // 滚动到底部（仅当前对话）
//   loadWsTokenStats(),            // done 后重新加载 token 统计
//   autoNameConv(convId, text),    // 自动命名对话
//   saveConvMsg(convId, content),  // 保存助手消息到后端
//   onTaskUpdate(convId, taskId, status, subject),  // task 工具调用
//   onTaskCreate(task, convId),    // task_create 工具调用
//   onTaskUpdate(taskId, status, subject, convId), // task_update 工具调用
//   onTaskReplace(tasks, convId),  // update_tasks 全量替换子任务清单调用
//   onPhaseChange(convId),         // 阶段变化（RightPanel 启动定时器清除）
// }


let globalCtx = {}

export function setGlobalCtx(ctx) {
  globalCtx = ctx || {}
}

// ─── 刷新门控：历史加载完成前，WS 事件先入 pending ───
// 页面刷新后 WS（status/snapshot/流式事件）先于 HTTP 历史到达：若快照/流式消息
// 先进入 messagesByConv，switchConv 的 hasRealMsgs 判定「已有内容」→ 跳过 API 加载
// → 消息区只剩实时增量、历史永远不出现（用户报告「刷新后只有当前 ws 消息」）。
// 门控方案：历史未加载的会话，其 WS 事件全部入 pending（snapshot 覆盖存储），
// markHistoryLoaded（switchConv/reload 加载成功）后统一 flush：快照先（重建当前回合），
// 事件后（增量），与 HTTP 历史无缝拼接。
const wsPendingByConv = new Map()    // convId → { snapshot: null|data, events: [] }
const historyLoadedConvs = new Set() // 已完成历史加载的会话（刷新后防反复门控）

// markHistoryLoaded 标记会话历史已加载并 flush 门控期间的事件。
// 由 RightPanel 在 apiLoadAndBuildConv 成功后调用（switchConv / reloadConvMessages）。
export function markHistoryLoaded(convId) {
  if (!convId) return
  historyLoadedConvs.add(convId)
  const pend = wsPendingByConv.get(convId)
  if (!pend) return
  wsPendingByConv.delete(convId)
  const { snapshot, events } = pend
  if (snapshot) {
    try { processAgentEvent(convId, snapshot) } catch (e) { console.warn('[AE] flush snapshot 失败 conv=%s', convId, e) }
  }
  for (const ev of events) {
    try { processAgentEvent(convId, ev) } catch (e) { console.warn('[AE] flush event 失败 conv=%s', convId, e) }
  }
  console.log('[AE] markHistoryLoaded flush conv=%s snapshot=%s events=%d', convId, !!snapshot, events.length)
}

// ─── 运行时管理 ──
// msgKey 是 assistant 占位消息的唯一标识 _key（比 msgIdx 数组下标稳定，
// 不会被 loadMoreMessages 的 prepend 操作破坏）。
export function startConvRuntime(convId, msgKey, lastUserText = '') {
  runtimes[convId] = {
    msgKey: msgKey,
    finalContent: '',
    lastUserText,
  }
}

export function getConvRuntime(convId) {
  return runtimes[convId] || null
}

export function resetConvRuntime(convId) {
  delete runtimes[convId]
}

// 通过 _key 在消息数组中查找 assistant 消息（稳定，不受 prepend 影响）
function findMsgByKey(msgs, key) {
  if (!msgs || !key) return null
  for (const m of msgs) {
    if (m._key === key) return m
  }
  return null
}

// 创建助手消息占位符（由 RightPanel sendMessage 调用，在 chatStart 之前）
// 返回 msgKey（唯一标识，比数组下标稳定）。
export function createAssistantPlaceholder(convId, key) {
  const msgs = state.messagesByConv[convId]
  if (!msgs) return ''
  if (!key) key = makeMsgKey()
  // ★ 计算 _idx：取当前最大 _idx + 1，而非数组长度
  let nextIdx = msgs.length
  for (const m of msgs) { if ((m._idx ?? 0) >= nextIdx) nextIdx = (m._idx ?? 0) + 1 }
  const assistantMsg = {
    role: 'assistant', content: '', segments: [], toolCalls: [],
    _key: key, _idx: nextIdx,
    _time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
    _loading: true,
  }
  msgs.push(assistantMsg)
  return key
}

// ─── Live 快照（WS 断线补偿）──
// 服务端在客户端重连时推送 running 会话的流式生成进度快照，
// 重建占位消息（快照为完整最新状态 → 整体替换 segments/finalContent）。
// ★ 2026-08-22 时序修复：后端快照新增 events 有序序列（thinking/content/tool_call/
//   tool_result/error…按 emit 顺序），前端逐事件重放重建 segments——
//   保真「正文→工具→正文」的真实交错顺序。旧快照（仅 reasoning/tools/content
//   三个分离累积字段，无法还原交错）降级兼容：content 聚在工具调用后面，
//   表现为「上方总工具调用、下方总正文输出」，已由本修复根除。
function applyLiveSnapshot(convId, msg, rt, snap) {
  // ★ 2026-08-22 保留用户交互状态：快照重建 segments 前按类别收集旧的展开状态
  //   （thinking 展开 = _collapsed===false；tool_call 展开 = _expanded===true），
  //   重建时按出现顺序恢复——否则刷新后 WS 重连快照到达会把用户刚展开的
  //   思考/工具调用全部重置回折叠态（表现为「滚动/刷新后自动折叠」）。
  const oldSegs = msg.segments || []
  const oldThinkStates = []
  const oldToolStates = []
  for (const s of oldSegs) {
    if (s.type === 'thinking') oldThinkStates.push(s._collapsed === false ? 'expanded' : 'collapsed')
    else if (s.type === 'tool_call') oldToolStates.push(s._expanded === true ? 'expanded' : 'collapsed')
  }
  const events = Array.isArray(snap.events) && snap.events.length > 0 ? snap.events : null
  const segments = []
  // 恢复单个 tool_call 段的展开状态（按旧 segments 中 tool_call 的出现顺序匹配）
  const restoreToolState = (seg) => {
    if (oldToolStates.shift() === 'expanded') { seg._expanded = true; seg._mode = 'expanded' }
    return seg
  }
  if (events) {
    // ── 新路径：按有序事件序列逐事件重放（与实时流 processAgentEvent 同构）──
    for (const ev of events) {
      const type = ev.type || ''
      if (type === 'thinking') {
        const seg = pushSegment(segments, 'thinking', { _mode: 'collapsed', _collapsed: true })
        if (oldThinkStates.shift() === 'expanded') { seg._collapsed = false; seg._mode = 'expanded' }
        seg.content += ev.content || ''
      } else if (type === 'content') {
        const seg = pushSegment(segments, 'content')
        seg.content += ev.content || ''
      } else if (type === 'tool_call') {
        const toolName = ev.tool || ev.name || ''
        if (toolName === 'ask_user') {
          // 与 processAgentEvent 的 ask_user 分支对齐：重建交互式提问卡
          // ★ Round3 ⑤：questions 多问题数组（后端已解析进 seg.questions）
          let question = ''
          let askType = 'text'
          let options = []
          let questions = []
          try {
            const args = typeof ev.args === 'string' ? JSON.parse(ev.args) : ev.args
            question = args.question || '（无问题内容）'
            askType = normalizeAskType(args.askType || args.type || 'text')
            if (Array.isArray(args.options)) options = args.options
            if (Array.isArray(args.questions)) questions = args.questions
          } catch {}
          const seg = {
            type: 'ask_user', question, askType, options, questions,
            callId: ev.callId || '',
            answer: ev.content || '', _answered: !!ev.content,
          }
          if (questions.length > 0) seg.question = ''
          segments.push(seg)
        } else {
          segments.push(restoreToolState({
            type: 'tool_call', name: toolName,
            callId: ev.callId || '',
            argsRaw: ev.args ? (typeof ev.args === 'string' ? ev.args : JSON.stringify(ev.args, null, 2)) : '',
            result: ev.content || '', // 后端已将 tool_result 回填到 tool_call 事件的 Content
            _mode: 'collapsed', _collapsed: false, _expanded: false,
          }))
        }
      } else if (type === 'tool_result') {
        // 兜底：独立 tool_result 事件（异常路径）——回填最近的 tool_call 段
        const callId = ev.callId || ''
        for (let i = segments.length - 1; i >= 0; i--) {
          const s = segments[i]
          if (s.type === 'tool_call') {
            if (callId && s.callId === callId) { s.result = ev.content || ''; break }
            if (!s.result && !callId) { s.result = ev.content || ''; break }
          }
        }
      } else if (type === 'error') {
        const seg = pushSegment(segments, 'content')
        seg.content += '**[错误]** ' + (ev.content || '')
      } else if (type === 'notice' || type === 'compacted' || type === 'circling' || type === 'evaluation' || type === 'approval') {
        const seg = pushSegment(segments, 'content')
        seg.content += ev.content || ''
      }
    }
  } else {
    // ── 旧路径（兼容旧后端）：三字段无法还原交错，按推理→正文→工具 顺序归位──
    const reasoning = snap.reasoning || ''
    const content = snap.content || ''
    const tools = Array.isArray(snap.toolSegments) ? snap.toolSegments : []
    if (reasoning) {
      const seg = { type: 'thinking', content: reasoning, _mode: 'collapsed', _collapsed: true }
      if (oldThinkStates.shift() === 'expanded') { seg._collapsed = false; seg._mode = 'expanded' }
      segments.push(seg)
    }
    if (content) {
      segments.push({ type: 'content', content })
    }
    for (const t of tools) {
      if (!t || !t.name) continue
      segments.push(restoreToolState({
        type: 'tool_call', name: t.name, callId: t.callId || '',
        argsRaw: t.args ? (typeof t.args === 'string' ? t.args : JSON.stringify(t.args, null, 2)) : '',
        result: t.result || '', _mode: 'collapsed', _collapsed: false, _expanded: false,
      }))
    }
  }
  msg.segments = segments
  msg.content = snap.content || ''
  rt.finalContent = snap.content || ''
  console.log('[AE] liveSnapshot 已应用 conv=%s events=%d reasoning=%d content=%d tools=%d',
    convId, events ? events.length : 0, (snap.reasoning || '').length, (snap.content || '').length,
    (snap.toolSegments || []).length)
}

// ─── 事件处理 ──
export function processAgentEvent(convId, data) {
  // ★ 刷新门控：历史未加载（页面刷新后 WS 事件先于 HTTP 历史到达）→ 事件先入
  //   pending，待 switchConv/reload 加载完成后 flush。避免快照/流式内容先占位
  //   导致 hasRealMsgs 误判、历史永不加载（「刷新后只有当前 ws 消息」）。
  if (!historyLoadedConvs.has(convId)) {
    let pend = wsPendingByConv.get(convId)
    if (!pend) { pend = { snapshot: null, events: [] }; wsPendingByConv.set(convId, pend) }
    if (data && data.type === 'snapshot') pend.snapshot = data
    else pend.events.push(data)
    return
  }
  // 确保 messagesByConv 存在
  if (!state.messagesByConv[convId]) state.messagesByConv[convId] = []
  const msgs = state.messagesByConv[convId]

  // ★ 自动恢复 runtime（防止因各种原因 runtime 丢失导致事件永久丢弃）
  let rt = runtimes[convId]
  if (!rt) {
    // 尝试找 messages 中最后一条 loading 的 assistant 消息
    const lastLoading = [...msgs].reverse().find(m => m.role === 'assistant' && m._loading)
    if (lastLoading && lastLoading._key) {
      console.log('[AE] processAgentEvent 自动恢复 runtime conv=%s key=%s type=%s', convId, lastLoading._key, data.type)
      rt = { msgKey: lastLoading._key, finalContent: '', lastUserText: '' }
      runtimes[convId] = rt
    } else if (data.type === 'content' || data.type === 'thinking' || data.type === 'done' || data.type === 'error' || data.type === 'snapshot') {
      // 内容类/错误类事件没有 runtime 也无 loading 消息时，创建新 assistant 占位。
      // ★ error 事件必须在此分支：否则异常中断（LLM API 错误/panic）时若 runtime 丢失
      //   （页面刷新后/事件乱序），错误被静默丢弃 → 用户看到 agent 无故停止且无任何提示。
      // ★ snapshot 事件必须在此分支：页面刷新后重连，需创建占位以承载断线补偿快照。
      console.log('[AE] processAgentEvent 创建临时占位 conv=%s type=%s', convId, data.type)
      const key = makeMsgKey()
      let phNextIdx = msgs.length
      for (const m of msgs) { if ((m._idx ?? 0) >= phNextIdx) phNextIdx = (m._idx ?? 0) + 1 }
      const placeholder = {
        role: 'assistant', content: '', segments: [], toolCalls: [],
        _key: key, _idx: phNextIdx,
        _time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
        _loading: false,
      }
      msgs.push(placeholder)
      rt = { msgKey: key, finalContent: '', lastUserText: '' }
      runtimes[convId] = rt
    } else {
      // 非内容类事件直接丢弃
      console.warn('[AE] processAgentEvent 丢弃(无 runtime 且无法恢复): conv=%s type=%s', convId, data.type)
      return
    }
  }
  if (!msgs) {
    console.warn('[AE] processAgentEvent 丢弃: 无 messagesByConv conv=%s type=%s', convId, data.type)
    return
  }
  const isCurrent = state.currentConvId === convId

  // 找目标 assistant 消息：优先用 rt.msgKey 定位，找不到则 fallback 到最后一个 assistant
  // （页面刷新后 runtime 的 msgKey 可能已失效，但历史消息中最后一条仍是正确的目标）
  let msg = findMsgByKey(msgs, rt.msgKey)
  if (!msg) {
    const lastAssistant = [...msgs].reverse().find(m => m.role === 'assistant')
    if (lastAssistant) {
      console.log('[AE] processAgentEvent fallback 到最后 assistant conv=%s type=%s', convId, data.type)
      msg = lastAssistant
      // 更新 runtime 的 msgKey 指向实际找到的消息
      if (lastAssistant._key) rt.msgKey = lastAssistant._key
    }
  }
  if (!msg) {
    console.warn('[AE] processAgentEvent 丢弃: 无目标 msg conv=%s type=%s', convId, data.type)
    return
  }
  msg._loading = false

  if (data.type === 'snapshot') {
    // ★ 2026-08-21 WS 断线补偿：用服务端快照重建占位消息。
    //   断线期间丢失的 content/thinking/tool 事件由快照补齐（快照是最新完整状态 → 整体替换）。
    applyLiveSnapshot(convId, msg, rt, data)
  } else if (data.type === 'thinking') {
    const seg = pushSegment(msg.segments, 'thinking', { _mode: 'collapsed', _collapsed: true })
    seg.content += data.content || ''
  } else if (data.type === 'content') {
    rt.finalContent += data.content || ''
    const seg = pushSegment(msg.segments, 'content')
    seg.content += data.content || ''
  } else if (data.type === 'tool_call') {
    const toolName = data.tool || data.name || ''
    if (toolName === 'ask_user') {
      let question = ''
      let askType = 'text'
      let options = []
      let questions = []
      try {
        const args = typeof data.args === 'string' ? JSON.parse(data.args) : data.args
        question = args.question || '（无问题内容）'
        askType = normalizeAskType(args.askType || args.type || 'text') // 容错：变体统一归一化
        if (Array.isArray(args.options)) {
          options = args.options
        }
        // ★ Round3 ⑤：多问题数组
        if (Array.isArray(args.questions)) {
          questions = args.questions
        }
      } catch {}
      const seg = {
        type: 'ask_user', question, askType, options, questions,
        callId: data.callId || data.callID || '',
        answer: '', _answered: false,
      }
      if (questions.length > 0) seg.question = ''
      msg.segments.push(seg)
    } else if (toolName === 'task_create') {
      try {
        const args = data.args ? (typeof data.args === 'string' ? JSON.parse(data.args) : data.args) : {}
        if (globalCtx.onTaskCreate) globalCtx.onTaskCreate({ step: args.subject || '(新建任务)', status: 'pending', callId: data.callId || data.callID || '', _taskId: null }, convId)
      } catch {}
      msg.segments.push({
        type: 'tool_call', name: toolName,
        callId: data.callId || data.callID || '',
        argsRaw: data.args ? (typeof data.args === 'string' ? data.args : JSON.stringify(data.args, null, 2)) : '',
        result: '', _mode: 'collapsed', _expanded: false,
      })
    } else if (toolName === 'task_update') {
      try {
        const args = data.args ? (typeof data.args === 'string' ? JSON.parse(data.args) : data.args) : {}
        if (globalCtx.onTaskUpdate) globalCtx.onTaskUpdate(args.id, args.status || '', args.subject || '', convId)
      } catch {}
      msg.segments.push({
        type: 'tool_call', name: toolName,
        callId: data.callId || data.callID || '',
        argsRaw: data.args ? (typeof data.args === 'string' ? data.args : JSON.stringify(data.args, null, 2)) : '',
        result: '', _mode: 'collapsed', _expanded: false,
      })
    } else if (toolName === 'update_tasks') {
      // update_tasks 全量替换子任务清单（用于内层 Loop 的 update_tasks 子任务跟踪）
      try {
        const args = data.args ? (typeof data.args === 'string' ? JSON.parse(data.args) : data.args) : {}
        if (Array.isArray(args.tasks) && globalCtx.onTaskReplace) {
          const tasks = args.tasks.map(t => ({
            step: t.subject || t.description || '(无标题)',
            status: t.status || 'pending',
            _taskId: t.id || null,
          }))
          globalCtx.onTaskReplace(tasks, convId)
        }
      } catch {}
      msg.segments.push({
        type: 'tool_call', name: toolName,
        callId: data.callId || data.callID || '',
        argsRaw: data.args ? (typeof data.args === 'string' ? data.args : JSON.stringify(data.args, null, 2)) : '',
        result: '', _mode: 'collapsed', _collapsed: false, _expanded: false,
      })
    } else {
      msg.segments.push({
        type: 'tool_call', name: toolName,
        callId: data.callId || data.callID || '',
        argsRaw: data.args ? (typeof data.args === 'string' ? data.args : JSON.stringify(data.args, null, 2)) : '',
        result: '', _mode: 'collapsed', _expanded: false,
      })
    }
  } else if (data.type === 'tool_result') {
    const callId = data.callId || data.callID || ''
    const toolName = data.tool || data.name || ''

    // ── 文件修改工具 → 触发文件树刷新 ──
    const fileTools = ['write_file', 'edit_file', 'multi_edit', 'delete_file', 'move_file']
    if (fileTools.includes(toolName)) {
      window.dispatchEvent(new CustomEvent('refresh-tree'))
    }

    // task_create 结果：提取任务 ID 更新计划（无对应 segment）
    if (toolName === 'task_create' && globalCtx.onTaskSetId) {
      const idMatch = (data.content || '').match(/ID:\s*`([^`]+)`/)
      if (idMatch) globalCtx.onTaskSetId(callId, idMatch[1], convId)
    }

    if (!msg || !msg.segments) return
    let target = null
    for (let i = msg.segments.length - 1; i >= 0; i--) {
      const seg = msg.segments[i]
      if (seg.type === 'tool_call') {
        if (callId && seg.callId === callId) { target = seg; break }
        if (!target) target = seg
      }
    }
    if (target) {
      target.result = data.content || ''
      // ★ 用户已手动展开的 tool_call 不被结果折叠：流式期间用户展开查看细节时，
      //   结果到达不强制收起——否则表现为「展开后又被自动折叠」
      if (target._expanded !== true) target._expanded = false
    }
  } else if (data.type === 'approval') {
    // 解析 args JSON，结构化展示
    let parsedArgs = {}
    try { parsedArgs = JSON.parse(data.args || '{}') } catch {}
    state.approvalByConv[convId] = {
      callId: data.callId || data.callID || '',
      tool: data.tool || '',
      args: data.args || '',
      parsedArgs, // 结构化后的参数
      waiting: true,
    }
  } else if (data.type === 'error') {
    const errText = (data.content || '').trim()
    const seg = pushSegment(msg.segments, 'content')
    seg.content += '**[错误]** ' + errText
    // 附带"可继续"引导：异常中断后可直接在本对话继续，不丢失进度
    seg.content += '\n\n> ⚠️ 本次任务未完成。可直接在下方输入继续（沿用本对话上下文），或点击对话列表中的该项恢复。'
    // ★ 后端异常/停止时只发 EventError 不发 EventDone，必须在此清理 loading 状态，
    //   否则对话永久显示"运行中"、输入框保持 disabled，用户无法继续对话。
    msg._loading = false
    state.loadingByConv[convId] = false
    state.agentRunningByConv[convId] = false
    if (isCurrent) {
      state.chatLoading = false
      state.agentRunning = false
      if (globalCtx.onPhaseEnd) globalCtx.onPhaseEnd(convId)
    }
    // 同步对话列表的运行标记（刷新侧边栏"运行中"指示）
    // ★ 本地更新该对话的 interrupted 标记，使侧边栏立即可见"⚠️ 未完成"（无需等刷新）
    const localConv = state.conversations.find(c => c.id === convId)
    if (localConv) localConv.interrupted = true
    window.dispatchEvent(new Event('save-conversations'))
    if (globalCtx.loadWsTokenStats) globalCtx.loadWsTokenStats()
    delete runtimes[convId]
    // ★ 不再执行后面的 scrollToBottom/state.messages 同步（直接返回）
    if (isCurrent) {
      state.messages = msgs
      if (globalCtx.scrollToBottom) globalCtx.scrollToBottom(convId)
    }
    return
  } else if (data.type === 'usage' && data.usage) {
    const u = data.usage
    // wsTokenStats 从 API /api/tokens/stats 加载（工作区级累积），不被 per-call 值覆盖
    // 仅当前对话才更新 convCtxStats（避免跨对话串扰）
    if (isCurrent) {
      const cs = getConvCtxStats(convId)
      cs.promptTokens = u.prompt_tokens || 0
      cs.completionTokens = u.completion_tokens || 0
      cs.cacheHitTokens = u.prompt_cache_hit_tokens || 0
      cs.cacheMissTokens = u.prompt_cache_miss_tokens || 0
      if (u.prompt_breakdown) {
        const pb = u.prompt_breakdown
        cs.systemTokens = pb.system_tokens || 0
        cs.skillsTokens = pb.skills_tokens || 0
        cs.mcpTokens = pb.mcp_tokens || 0
        cs.toolTokens = pb.tool_tokens || 0
        cs.historyTokens = pb.history_tokens || 0
        cs.otherTokens = pb.other_tokens || 0
      }

      // ★ 累加到工作区级 token 统计，让侧边栏实时更新，不等对话结束才显示
      const wsRoot = state.workspaceRoot
      if (wsRoot) {
        if (!state.wsTokenStatsByWs[wsRoot]) {
          state.wsTokenStatsByWs[wsRoot] = { totalTokens: 0, promptTokens: 0, completionTokens: 0, cacheHitTokens: 0, cacheMissTokens: 0, systemTokens: 0, skillsTokens: 0, mcpTokens: 0, toolTokens: 0, historyTokens: 0, otherTokens: 0 }
        }
        const wsStats = state.wsTokenStatsByWs[wsRoot]
        wsStats.promptTokens += u.prompt_tokens || 0
        wsStats.completionTokens += u.completion_tokens || 0
        wsStats.totalTokens += (u.prompt_tokens || 0) + (u.completion_tokens || 0)
        wsStats.cacheHitTokens += u.prompt_cache_hit_tokens || 0
        wsStats.cacheMissTokens += u.prompt_cache_miss_tokens || 0
        if (u.prompt_breakdown) {
          wsStats.systemTokens += u.prompt_breakdown.system_tokens || 0
          wsStats.skillsTokens += u.prompt_breakdown.skills_tokens || 0
          wsStats.mcpTokens += u.prompt_breakdown.mcp_tokens || 0
          wsStats.toolTokens += u.prompt_breakdown.tool_tokens || 0
          wsStats.historyTokens += u.prompt_breakdown.history_tokens || 0
          wsStats.otherTokens += u.prompt_breakdown.other_tokens || 0
        }
      }
    }
  } else if (data.type === 'phase') {
    state.phaseByConv[convId] = data.content || ''
    if (isCurrent && globalCtx.onPhaseChange) globalCtx.onPhaseChange(convId)
  } else if (data.type === 'notice') {
    if (isCurrent) {
      state.nudgeByConv[convId] = (data.content || '').replace(/\n/g, ' ').slice(0, 120)
      if (globalCtx.onNudge) globalCtx.onNudge(convId)
    }
  } else if (data.type === 'compacted') {
    msg.segments.push({ type: 'content', content: '> 📦 上下文已压缩（中段老消息已摘要）' })
  } else if (data.type === 'circling') {
    msg.segments.push({ type: 'content', content: '> ⚠️ 检测到重复操作，已提示 Agent 换思路' })
  } else if (data.type === 'evaluation') {
    msg.segments.push({ type: 'content', content: '> 📊 任务评测：\n' + (data.content || '') })
  }

  // ★ 同步 state.messages（当前对话时），确保 Vue 响应式更新
  //    processAgentEvent 直接修改 messagesByConv[convId]，需要同步到 state.messages
  //    让 messageCombos computed 能感知到消息对象内部属性的变化。
  if (isCurrent) {
    state.messages = msgs
  }

  if (isCurrent && globalCtx.scrollToBottom) globalCtx.scrollToBottom(convId)
}

export function processAgentDone(convId, data) {

  console.log('[AE] processAgentDone conv=%s hasRuntime=%s msgByConvLen=%d', convId, !!runtimes[convId], (state.messagesByConv[convId]||[]).length)
  const rt = runtimes[convId]
  const msgs = state.messagesByConv[convId]
  if (msgs && rt) {
    const msg = findMsgByKey(msgs, rt.msgKey)
    if (msg) {
      // 用流式累积的 finalContent 替换（已在 content 事件中逐字推送，不含重复）
      msg.content = rt.finalContent
      // ★ 不在此处追加 data.content 段 —— content 已被流式 content 事件或 tool_call/tool_result
      //   推送过（finish_task 的结果已在 tool_result 中显示）。若在此重复追加会造成「两次完
      //   成报告」的视觉重复。

      // ★ 处理空占位：若 finalContent 为空且 segments 中无有效内容（无 tool_call/ask_user、
      //   无非空 content/thinking 段），给个最低限度的提示，避免前端显示空白气泡。
      const hasEffectiveSeg = (msg.segments || []).some(seg => {
        if (seg.type === 'tool_call' || seg.type === 'ask_user') return true
        if (seg.type === 'content' && seg.content && seg.content.trim()) return true
        if (seg.type === 'thinking' && seg.content && seg.content.trim()) return true
        return false
      })
      const isEmptyPlaceholder = !rt.finalContent && !hasEffectiveSeg && (!data || data.doneReason !== 'stopped')
      if (isEmptyPlaceholder) {
        console.log('[AE] processAgentDone 空占位 conv=%s msgKey=%s 设为完成提示', convId, rt.msgKey)
        msg.content = '**[操作完成]**'
        pushSegment(msg.segments, 'content').content = '**[操作完成]**'
      }

      // ★ 用户主动停止时，若没有流式内容（finalContent 为空），显示停止提示
      if (data && data.doneReason === 'stopped') {
        if (!rt.finalContent) {
          msg.content = '**[任务已终止]** ' + (data.content || '用户终止了任务')
        } else {
          // 已有部分内容，追加一行说明
          pushSegment(msg.segments, 'content').content += '\n\n**[任务已终止]** ' + (data.content || '用户终止了任务')
        }
      }
    }
  }

  // ★ 兜底：即使 rt 或 msgKey 失效，也要修复 messages 中所有 loading 状态
  //   并保证最后一条 assistant 消息能够正确显示。
  if (msgs) {
    // 1. 清除所有 _loading 标记
    for (const m of msgs) {
      if (m._loading) m._loading = false
    }
    // 2. 如果最后一条 assistant 消息 content 仍为空且无 segments，
    //    尝试从 data.content 回填
    if (!rt) {
      const lastAssistant = [...msgs].reverse().find(m => m.role === 'assistant')
      if (lastAssistant && !lastAssistant.content && (!lastAssistant.segments || lastAssistant.segments.length === 0)) {
        if (data && data.content) {
          lastAssistant.content = data.content
        } else if (data && data.doneReason === 'stopped') {
          lastAssistant.content = '**[任务已终止]** ' + (data.content || '用户终止了任务')
        } else {
          lastAssistant.content = '**[操作完成]**'
        }
        console.log('[AE] processAgentDone 兜底回填 content conv=%s', convId)
      }
    }
  }
  state.loadingByConv[convId] = false
  state.agentRunningByConv[convId] = false
  const isCurrent = state.currentConvId === convId
  if (isCurrent) {
    state.chatLoading = false
    state.agentRunning = false
    if (globalCtx.onPhaseEnd) globalCtx.onPhaseEnd(convId)
  }
  if (globalCtx.loadWsTokenStats) globalCtx.loadWsTokenStats()
  if (rt && rt.finalContent && globalCtx.saveConvMsg) {
    const savedMsg = findMsgByKey(msgs, rt.msgKey)
    const savedIdx = savedMsg ? savedMsg._idx : -1
    globalCtx.saveConvMsg(convId, rt.finalContent, savedIdx)
  }
  const localConv = state.conversations.find(c => c.id === convId)
  if (localConv) localConv.msgCount = (localConv.msgCount || 0) + 1
  window.dispatchEvent(new Event('save-conversations'))
  // ★ 同步 state.messages（当前对话时），确保 Vue 响应式更新
  if (isCurrent) {
    state.messages = msgs
  }
  if (isCurrent && globalCtx.scrollToBottom) globalCtx.scrollToBottom(convId)
  delete runtimes[convId]
}

// 处理 WebSocket 连接断开导致的会话错误（重连失败后标记）
export function processAgentDisconnect(convId, errMsg) {
  const rt = runtimes[convId]
  const msgs = state.messagesByConv[convId]
  if (msgs && rt) {
    const msg = findMsgByKey(msgs, rt.msgKey)
    if (msg) {
      msg._loading = false
      pushSegment(msg.segments, 'content').content += '**[连接中断]** ' + errMsg
      pushSegment(msg.segments, 'content').content += '\n\n> ⚠️ 本次任务未完成。后端恢复后可继续本对话（进度已保存）。'
    }
    // ★ 连接中断（后端异常/网络断开）同样视为"未完成可继续"
    const localConv = state.conversations.find(c => c.id === convId)
    if (localConv) localConv.interrupted = true
    window.dispatchEvent(new Event('save-conversations'))
  }
  // 注意：不重置 agentRunningByConv，因为后端 agent 可能仍在运行
  // 重连后会通过 status 消息同步真实状态
  const isCurrent = state.currentConvId === convId
  if (isCurrent) {
    state.chatLoading = false
    state.agentRunning = false
  }
}

// 处理全部 WebSocket 重连失败（后端进程已关闭）。
// 清理所有标记为 running 的对话，重置状态。
export function processAllDisconnected() {
  for (const convId of Object.keys(state.agentRunningByConv)) {
    const rt = runtimes[convId]
    const msgs = state.messagesByConv[convId]
    if (msgs && rt) {
      const msg = findMsgByKey(msgs, rt.msgKey)
      if (msg) {
        msg._loading = false
        if (!msg.content) msg.content = ''
        pushSegment(msg.segments, 'content').content += '**[连接中断]** 后端进程已关闭，请重新发送消息。'
        pushSegment(msg.segments, 'content').content += '\n\n> ⚠️ 本次任务未完成。重启后端后在本对话继续即可（进度已保存）。'
      } else {
        // 清理所有 _loading 标记
        for (const m of msgs) {
          if (m._loading) m._loading = false
        }
      }
    } else if (msgs) {
      // 即使无 runtime，也要清理 messages 中的 loading 标记
      for (const m of msgs) {
        if (m._loading) m._loading = false
      }
    }
    // ★ 进程关闭 → 本对话视为"未完成可继续"（后端重启后同 convID 继续）
    const localConv = state.conversations.find(c => c.id === convId)
    if (localConv) localConv.interrupted = true
    state.agentRunningByConv[convId] = false
    state.loadingByConv[convId] = false
    delete runtimes[convId]
  }
  if (Object.keys(state.agentRunningByConv).length > 0) {
    window.dispatchEvent(new Event('save-conversations'))
  }
  state.chatLoading = false
  state.agentRunning = false
}

// 处理 WebSocket 收到的 status 消息（连接建立/重连后、以及会话 done 后推送）
// payload: { runningConvs: [...], runningByWorkspace: {wsRoot: count} }
export function processStatus(payload) {
  // 兼容旧调用：若传入数组则包装为对象
  const p = Array.isArray(payload) ? { runningConvs: payload, runningByWorkspace: {} } : (payload || {})
  const runningConvs = p.runningConvs || []
  const runningByWorkspace = p.runningByWorkspace || {}

  // 同步工作区运行计数（供 FileExplorer 工作区项显示脉冲点+计数）
  // 用全新对象替换以触发 Vue 响应式更新
  state.runningByWorkspace = { ...runningByWorkspace }

  // 同步运行中状态：后端报告的 runningConvs 标记为 running
  const runningSet = new Set(runningConvs)
  // 标记仍在运行的（只设置状态标记，不创建占位也不创建 runtime）
  // ★ 占位和 runtime 由 switchConv（加载完历史后）或 processAgentEvent（首次收到事件时）负责创建
  for (const convId of runningSet) {
    state.agentRunningByConv[convId] = true
    state.loadingByConv[convId] = true
    // ★ 兜底：若消息已加载（switchConv 已完成）但无 runtime，创建占位
    //   ★ 刷新门控：历史未加载时跳过（占位由 switchConv 加载后创建，避免先占位
    //     导致 hasRealMsgs 误判、历史不加载）
    const msgsArr = state.messagesByConv[convId]
    if (historyLoadedConvs.has(convId) && msgsArr && msgsArr.length > 0 && !runtimes[convId]) {
      const hasRealMsgs = msgsArr.some(m => !m._loading)
      const lastLoading = [...msgsArr].reverse().find(m => m._loading)
      if (hasRealMsgs && !lastLoading) {
        const key = makeMsgKey()
        let psNextIdx = msgsArr.length
        for (const m of msgsArr) { if ((m._idx ?? 0) >= psNextIdx) psNextIdx = (m._idx ?? 0) + 1 }
        console.log('[AE] processStatus 兜底创建占位 conv=%s key=%s', convId, key)
        msgsArr.push({
          role: 'assistant', content: '', segments: [], toolCalls: [],
          _key: key, _idx: psNextIdx,
          _time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
          _loading: true,
        })
        runtimes[convId] = { msgKey: key, finalContent: '', lastUserText: '' }
        // 若是当前对话，同步 state.messages
        if (state.currentConvId === convId) {
          state.messages = msgsArr
        }
      }
    }
  }
  // 对于本地标记为 running 但后端已不在 running 列表的，修正状态
  // （可能是 WS 断连期间 agent 已结束，但 done 事件未送达）
  // ★ 2026-08-31 断线重同步：这类会话断线期间已结束，仅清理状态不够——
  //   最终输出/done 内容都在断线期间发出并丢失，必须重载服务端最新消息，
  //   否则前端停留在「中断」画面（浏览器休眠恢复后任务早已完成也无感知）。
  const resyncCandidates = []
  for (const convId of Object.keys(state.agentRunningByConv)) {
    if (state.agentRunningByConv[convId] && !runningSet.has(convId)) {
      state.agentRunningByConv[convId] = false
      state.loadingByConv[convId] = false
      if (state.currentConvId === convId) {
        state.chatLoading = false
        state.agentRunning = false
      }
      // ★ 同时清除 messages 中残留的 _loading 标记（防止个别消息永久显示 loading 动画）
      const msgsArr = state.messagesByConv[convId]
      if (msgsArr) {
        for (const m of msgsArr) {
          if (m._loading) m._loading = false
        }
      }
      delete runtimes[convId]
      resyncCandidates.push(convId)
    }
  }
  // 也清理 loadingByConv 中遗漏的条目：某些场景下 agentRunningByConv 可能为空
  // 但 loadingByConv 仍残留 true（如页面刷新后 WS 重连前的 done 事件丢失）。
  // 只要 conv 不在 runningSet 中，loading 状态都应清除。
  for (const convId of Object.keys(state.loadingByConv)) {
    if (state.loadingByConv[convId] && !runningSet.has(convId)) {
      state.loadingByConv[convId] = false
      // ★ 同步清除 messages 中的 _loading 标记
      const msgsArr = state.messagesByConv[convId]
      if (msgsArr) {
        for (const m of msgsArr) {
          if (m._loading) m._loading = false
        }
      }
      if (!state.agentRunningByConv[convId]) resyncCandidates.push(convId)
    }
  }

  // ── 断线重同步：重载已完成会话的消息 + 会话元数据 ──
  // 触发条件：① 上方的「曾 running → 已结束」候选；② WS 重连成功标记（__wsReconnectedPending，
  // 由 api.js onopen 置位）——覆盖「断线期间完成但本地无 running 标记」的更一般场景。
  const reconnected = !!window.__wsReconnectedPending
  if (reconnected) {
    window.__wsReconnectedPending = false
    // 会话列表 meta 刷新（interrupted/updatedAt/msgCount 以服务端为准）
    try { globalCtx?.refreshConvMeta?.() } catch {}
    const cur = state.currentConvId
    if (cur && !runningSet.has(cur) && !resyncCandidates.includes(cur)) {
      resyncCandidates.push(cur)
    }
  }

  for (const convId of resyncCandidates) {
    try { globalCtx?.reloadConvMessages?.(convId) } catch (e) { console.warn('[AE] reloadConvMessages 失败', convId, e) }
  }
}

// ─── convCtxStats 辅助 ──
export function getConvCtxStats(convId) {
  if (!state.convCtxStatsByConv[convId]) {
    state.convCtxStatsByConv[convId] = reactive({
      promptTokens: 0, completionTokens: 0,
      cacheHitTokens: 0, cacheMissTokens: 0,
      systemTokens: 0, skillsTokens: 0, mcpTokens: 0,
      toolTokens: 0, historyTokens: 0, otherTokens: 0,
    })
  }
  return state.convCtxStatsByConv[convId]
}

export function resetConvCtxStats(convId) {
  if (state.convCtxStatsByConv[convId]) {
    Object.assign(state.convCtxStatsByConv[convId], {
      promptTokens: 0, completionTokens: 0,
      cacheHitTokens: 0, cacheMissTokens: 0,
      systemTokens: 0, skillsTokens: 0, mcpTokens: 0,
      toolTokens: 0, historyTokens: 0, otherTokens: 0,
    })
  }
}
