// agent-events.js: agent 事件处理共享模块
// 从 RightPanel.vue 的 chatSubscribe 回调中提取，供 App.vue 的 WebSocket 回调调用。
// 支持：thinking/content/tool_call/tool_result/approval/error/usage/phase/notice/compacted/circling/evaluation/done
//
// 设计要点：
// - 状态写入全局 state（messagesByConv/loadingByConv/agentRunningByConv/approvalByConv/phaseByConv/wsTokenStats/convCtxStatsByConv/nudgeByConv）
// - UI 相关回调（scrollToBottom/loadWsTokenStats/autoNameConv/saveConvMsg/onPlanUpdate）通过 setGlobalCtx 注册
// - RightPanel 在 onMounted 调用 setGlobalCtx 注册回调；App.vue 的 WebSocket onmessage 调用 processAgentEvent

import { state } from './main.js'
import { reactive } from 'vue'

// ─── 按 convId 存储运行时状态（非响应式，普通对象）──
// { convId: { msgIdx, finalContent, lastUserText } }
const runtimes = {}

let msgKeyCounter = 0
function makeMsgKey() { return 'msg_' + Date.now() + '_' + (msgKeyCounter++) }

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
//   onPlanUpdate(plan, convId),    // update_plan 工具调用
//   onTaskCreate(task, convId),    // task_create 工具调用
//   onTaskUpdate(taskId, status, subject, convId), // task_update 工具调用
//   onTaskReplace(tasks, convId),  // update_tasks 全量替换子任务清单调用
//   onPhaseChange(convId),         // 阶段变化（RightPanel 启动定时器清除）
// }
let globalCtx = {}

export function setGlobalCtx(ctx) {
  globalCtx = ctx || {}
}

// ─── 运行时管理 ──
// msgIdx 是 assistant 占位消息在 messagesByConv[convId] 中的索引
// （由 createAssistantPlaceholder 返回，必须在 push 之后取 length-1）
export function startConvRuntime(convId, msgIdx, lastUserText = '') {
  runtimes[convId] = {
    msgIdx: msgIdx,
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

// 创建助手消息占位符（由 RightPanel sendMessage 调用，在 chatStart 之前）
export function createAssistantPlaceholder(convId) {
  const msgs = state.messagesByConv[convId]
  if (!msgs) return -1
  const msgIdx = msgs.length
  console.log('[AE] createAssistantPlaceholder conv=%s msgIdx=%d msgsLen=%d', convId, msgIdx, msgs.length)
  const assistantMsg = {
    role: 'assistant', content: '', segments: [], toolCalls: [],
    _key: makeMsgKey(), _idx: msgIdx,
    _time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
    _loading: true,
  }
  msgs.push(assistantMsg)
  return msgIdx
}

// ─── 事件处理 ──
export function processAgentEvent(convId, data) {
  const rt = runtimes[convId]
  if (!rt) {
    console.warn('[AE] processAgentEvent 丢弃: 无 runtime conv=%s type=%s', convId, data.type)
    return
  }
  const msgs = state.messagesByConv[convId]
  if (!msgs) {
    console.warn('[AE] processAgentEvent 丢弃: 无 messagesByConv conv=%s type=%s', convId, data.type)
    return
  }
  const msg = msgs[rt.msgIdx]
  if (!msg) {
    console.warn('[AE] processAgentEvent 丢弃: msgIdx=%d 越界 msgsLen=%d conv=%s type=%s', rt.msgIdx, msgs.length, convId, data.type)
    return
  }
  if (data.type !== 'content' && data.type !== 'thinking') {
    console.log('[AE] processAgentEvent conv=%s type=%s msgIdx=%d', convId, data.type, rt.msgIdx)
  }
  msg._loading = false

  const isCurrent = state.currentConvId === convId

  if (data.type === 'thinking') {
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
      try {
        const args = typeof data.args === 'string' ? JSON.parse(data.args) : data.args
        question = args.question || '（无问题内容）'
        askType = args.askType || 'text'
        if (Array.isArray(args.options)) {
          options = args.options
        }
      } catch {}
      msg.segments.push({
        type: 'ask_user', question, askType, options,
        callId: data.callId || data.callID || '',
        answer: '', _answered: false,
      })
    } else if (toolName === 'update_plan') {
      try {
        const args = data.args ? (typeof data.args === 'string' ? JSON.parse(data.args) : data.args) : {}
        if (Array.isArray(args.plan) && globalCtx.onPlanUpdate) globalCtx.onPlanUpdate(args.plan, convId)
      } catch {}
      // 也推送 segment，让用户在消息流中看到规划过程
      msg.segments.push({
        type: 'tool_call', name: toolName,
        callId: data.callId || data.callID || '',
        argsRaw: data.args ? (typeof data.args === 'string' ? data.args : JSON.stringify(data.args, null, 2)) : '',
        result: '', _mode: 'collapsed', _collapsed: false, _expanded: false,
      })
    } else if (toolName === 'task_create') {
      try {
        const args = data.args ? (typeof data.args === 'string' ? JSON.parse(data.args) : data.args) : {}
        if (globalCtx.onTaskCreate) globalCtx.onTaskCreate({ step: args.subject || '(新建任务)', status: 'pending', callId: data.callId || data.callID || '', _taskId: null }, convId)
      } catch {}
      msg.segments.push({
        type: 'tool_call', name: toolName,
        callId: data.callId || data.callID || '',
        argsRaw: data.args ? (typeof data.args === 'string' ? data.args : JSON.stringify(data.args, null, 2)) : '',
        result: '', _mode: 'expanded', _expanded: true,
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
        result: '', _mode: 'expanded', _expanded: true,
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
        result: '', _mode: 'expanded', _expanded: true,
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
      target._expanded = false
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
    const seg = pushSegment(msg.segments, 'content')
    seg.content += '**[错误]** ' + (data.content || '')
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

  if (isCurrent && globalCtx.scrollToBottom) globalCtx.scrollToBottom(convId)
}

export function processAgentDone(convId, data) {
  console.log('[AE] processAgentDone conv=%s hasRuntime=%s msgByConvLen=%d', convId, !!runtimes[convId], (state.messagesByConv[convId]||[]).length)
  const rt = runtimes[convId]
  const msgs = state.messagesByConv[convId]
  if (msgs && rt) {
    const msg = msgs[rt.msgIdx]
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
        console.log('[AE] processAgentDone 空占位 conv=%s msgIdx=%d 设为完成提示', convId, rt.msgIdx)
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
  // 无论 rt 是否存在，都要清除 loading 状态
  if (msgs) {
    for (const m of msgs) {
      if (m._loading) m._loading = false
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
    globalCtx.saveConvMsg(convId, rt.finalContent, rt.msgIdx)
  }
  const localConv = state.conversations.find(c => c.id === convId)
  if (localConv) localConv.msgCount = (localConv.msgCount || 0) + 1
  if (rt && rt.lastUserText && globalCtx.autoNameConv) {
    globalCtx.autoNameConv(convId, rt.lastUserText)
  }
  window.dispatchEvent(new Event('save-conversations'))
  if (isCurrent && globalCtx.scrollToBottom) globalCtx.scrollToBottom(convId)
  delete runtimes[convId]
}

// 处理 WebSocket 连接断开导致的会话错误（重连失败后标记）
export function processAgentDisconnect(convId, errMsg) {
  const rt = runtimes[convId]
  const msgs = state.messagesByConv[convId]
  if (msgs && rt) {
    const msg = msgs[rt.msgIdx]
    if (msg) {
      msg._loading = false
      pushSegment(msg.segments, 'content').content += '**[连接中断]** ' + errMsg
    }
  }
  // 注意：不重置 agentRunningByConv，因为后端 agent 可能仍在运行
  // 重连后会通过 status 消息同步真实状态
  const isCurrent = state.currentConvId === convId
  if (isCurrent) {
    state.chatLoading = false
    state.agentRunning = false
  }
}

// ─── processStatus 日志辅助 ──
let statusLogCounter = 0

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
  if (statusLogCounter++ < 100) {
    console.log('[AE] processStatus runningConvs=%o runtimeKeys=%o', runningConvs, Object.keys(runtimes))
  }
  // 标记仍在运行的
  for (const convId of runningSet) {
    state.agentRunningByConv[convId] = true
    // 后端认为它在运行，就标记 loading（即使已加载的历史消息没有 _loading 标记）
    state.loadingByConv[convId] = true
    // 页面刷新后 agent 仍在运行：前端没有 runtime，事件会被丢弃。
    // 为这些对话创建 messagesByConv 占位 + runtime，确保后续事件能被处理。
    if (!runtimes[convId]) {
      if (!state.messagesByConv[convId]) state.messagesByConv[convId] = []
      const msgs = state.messagesByConv[convId]
      // 若最后一条不是 loading 的 assistant 消息，则创建占位
      const last = msgs[msgs.length - 1]
      if (!last || last.role !== 'assistant' || !last._loading) {
        const msgIdx = msgs.length
        console.log('[AE] processStatus 创建runtime+loading conv=%s msgIdx=%d msgsLen=%d', convId, msgIdx, msgs.length)
        msgs.push({
          role: 'assistant', content: '', segments: [], toolCalls: [],
          _key: makeMsgKey(), _idx: msgIdx,
          _time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
          _loading: true,
        })
        runtimes[convId] = { msgIdx, finalContent: '', lastUserText: '' }
      } else {
        // 已有 loading 的 assistant 消息，直接复用其索引
        console.log('[AE] processStatus 复用runtime conv=%s msgIdx=%d', convId, msgs.length - 1)
        runtimes[convId] = { msgIdx: msgs.length - 1, finalContent: '', lastUserText: '' }
      }
      // 若是当前对话，同步 state.messages（仅当有更改时）
      if (state.currentConvId === convId) {
        state.messages = msgs
      }
    }
  }
  // 对于本地标记为 running 但后端已不在 running 列表的，修正状态
  // （可能是 WS 断连期间 agent 已结束，但 done 事件未送达）
  for (const convId of Object.keys(state.agentRunningByConv)) {
    if (state.agentRunningByConv[convId] && !runningSet.has(convId)) {
      state.agentRunningByConv[convId] = false
      state.loadingByConv[convId] = false
      if (state.currentConvId === convId) {
        state.chatLoading = false
        state.agentRunning = false
      }
      // 清理可能残留的 runtime
      delete runtimes[convId]
    }
  }
  // 也清理 loadingByConv 中遗漏的条目：某些场景下 agentRunningByConv 可能为空
  // 但 loadingByConv 仍残留 true（如页面刷新后 WS 重连前的 done 事件丢失）。
  // 只要 conv 不在 runningSet 中，loading 状态都应清除。
  for (const convId of Object.keys(state.loadingByConv)) {
    if (state.loadingByConv[convId] && !runningSet.has(convId)) {
      state.loadingByConv[convId] = false
    }
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
