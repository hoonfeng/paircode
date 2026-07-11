<template>
  <div class="right-panel">
    <!-- 标题 -->
    <div class="rp-header">
      <span class="rp-header-title"><SvgIcon name="bot" :size="16" /> 对话</span>
      <div class="rp-header-actions">
        <button class="rp-btn" @click="newConversation" title="新对话"><SvgIcon name="plus" :size="14" /></button>
        <button class="rp-btn" @click="showDebugLog = !showDebugLog" title="Debug 日志"><SvgIcon name="bug" :size="14" /></button>
        <button class="rp-btn" @click="toggleRight" title="关闭"><SvgIcon name="close" :size="14" /></button>
      </div>
    </div>

    <div class="rp-body">
      <!-- 左侧：聊天消息 + 输入区 -->
      <div class="chat-area">
        <!-- 阶段指示器（自主模式多阶段切换） -->
        <div v-if="currentPhase" class="phase-bar">
          <span class="phase-icon"><SvgIcon :name="phaseIcon(currentPhase)" :size="14" /></span>
          <span class="phase-text">{{ currentPhase }}</span>
          <span class="phase-dots"><span class="pd1"></span><span class="pd2"></span><span class="pd3"></span></span>
        </div>
        <!-- 消息区（带虚拟滚动） -->
        <div class="chat-messages" ref="msgRef" @scroll="onScroll">
          <!-- 顶部加载更多提示 -->
          <div v-if="hasMoreTop" class="scroll-more-hint" ref="topSentinel">
            <span>加载更早消息...</span>
          </div>
          <!-- 渲染的消息列表 -->
          <div class="msg-list-wrap"
               :style="{ paddingTop: virtualOffset.top + 'px', paddingBottom: virtualOffset.bottom + 'px' }">
            <div v-for="msg in visibleMessages" :key="msg._key"
                 :class="['msg-item', msg.role === 'user' ? 'msg-user' : 'msg-assistant']"
                 :data-msg-idx="msg._idx">
              <!-- 头像 -->
              <div class="msg-avatar">
                <SvgIcon v-if="msg.role === 'user'" name="user" :size="16" />
                <SvgIcon v-else name="bot" :size="16" />
              </div>
              <!-- 气泡主体 -->
              <div class="msg-bubble" :class="msg.role === 'user' ? 'bubble-user' : (msg.segments && msg.segments.length > 0 ? 'bubble-agent' : 'bubble-assistant')">
                <!-- 用户消息：右对齐，自适应宽度，Markdown 渲染 -->
                <template v-if="msg.role === 'user'">
                  <div v-if="msg.content" class="user-msg-content">
                    <MarkdownRenderer :text="msg.content" :theme="state.theme" />
                  </div>
                  <div v-else class="user-msg-placeholder">（空消息）</div>
                  <div class="msg-time">{{ msg._time || '' }}</div>
                </template>
                <!-- Agent 分段渲染 -->
                <template v-if="msg.role === 'assistant' && msg.segments && msg.segments.length > 0">
                  <!-- 折叠摘要 -->
                  <div v-if="msg._folded" class="folded-summary" @click="msg._folded = !msg._folded">
                    <span class="folded-chevron">▸</span>
                    <SvgIcon name="list" :size="11" />
                    <span class="folded-title">完成摘要</span>
                    <span class="folded-desc">{{ msgSummary(msg) }}</span>
                  </div>
                  <template v-if="!msg._folded">
                    <template v-for="(seg, si) in msg.segments" :key="si">
                      <!-- Thinking：简约斜体，默认折叠，展开后在末尾提供折叠按钮 -->
                      <div v-if="seg.type === 'thinking'" class="tl-item">
                        <span class="tl-dot tl-dot-thinking"></span>
                        <div class="tl-body tl-think-body">
                          <div v-if="!seg._collapsed" class="tl-thinking-text">{{ seg.content }}</div>
                          <div v-else class="tl-thinking-collapsed" @click="seg._collapsed = !seg._collapsed"><SvgIcon name="message-square" :size="12" /> 思考…</div>
                          <div v-if="!seg._collapsed" class="tl-think-fold" @click.stop="seg._collapsed = !seg._collapsed" title="折叠思考">▲ 收起</div>
                        </div>
                      </div>
                      <!-- Tool Call：折叠行，无卡片包裹 -->
                      <div v-else-if="seg.type === 'tool_call'" class="tl-item">
                        <span class="tl-dot tl-dot-tool"></span>
                        <div class="tl-body tl-tool">
                          <div class="tl-tc-header" @click="seg._expanded = !seg._expanded">
                            <span class="tl-tc-chevron">{{ seg._expanded ? '▾' : '▸' }}</span>
                            <SvgIcon :name="toolMeta(seg).icon" :size="11" class="tl-tc-icon" />
                            <span class="tl-tc-name">{{ toolMeta(seg).title }}</span>
                            <span v-if="seg.result && !seg._expanded" class="tl-tc-summary">{{ toolResultSummary(seg) }}</span>
                          </div>
                          <div v-if="seg._expanded" class="tl-tc-detail">
                            <template v-if="isTerminalTool(seg)">
                              <div class="tl-tc-section"><div class="tl-tc-section-title">命令</div><div class="tl-tc-command">{{ formatTerminalCommand(seg) }}</div></div>
                              <div v-if="seg.result" class="tl-tc-section"><div class="tl-tc-section-title">输出</div><pre class="tl-tc-output">{{ seg.result }}</pre></div>
                            </template>
                            <template v-else>
                              <div v-if="seg.argsRaw" class="tl-tc-section"><div class="tl-tc-section-title">参数</div><pre><code>{{ seg.argsRaw }}</code></pre></div>
                              <div v-if="seg.result" class="tl-tc-section"><div class="tl-tc-section-title">结果</div><pre><code>{{ seg.result }}</code></pre></div>
                            </template>
                          </div>
                        </div>
                      </div>
                      <!-- Ask User：交互式 -->
                      <div v-else-if="seg.type === 'ask_user'" class="tl-item">
                        <span class="tl-dot tl-dot-ask"></span>
                        <div class="tl-body"><AskUserCard :question="seg.question" :call-id="seg.callId" :answered="seg._answered" @answer="onAskAnswer(seg, $event)" /></div>
                      </div>
                      <!-- Content：纯 Markdown，无边框无圆点包裹 -->
                      <div v-else-if="seg.type === 'content'" class="tl-item tl-content-item">
                        <span class="tl-dot tl-dot-content"></span>
                        <div class="tl-body"><MarkdownRenderer :text="seg.content" :theme="state.theme" /></div>
                      </div>
                    </template>
                  </template>
                </template>
                <div class="msg-time">{{ msg._time || '' }}</div>
              </div>
              <div v-if="msg._loading" class="msg-loading-dots">
                <span class="dot"></span><span class="dot"></span><span class="dot"></span>
              </div>
            </div>
          </div>
          <div v-if="state.chatLoading && state.messages.length > 0" class="msg-loading-banner">
            <span class="dot-pulse"></span><span>思考中...</span>
          </div>
          <div v-if="state.messages.length === 0 && !state.chatLoading" class="chat-empty">
            <div class="chat-empty-icon"><SvgIcon name="bot" :size="32" /></div>
            <div class="chat-empty-text">开始新的对话</div>
            <div class="chat-empty-hint">发送消息即可与 AI 助手对话</div>
          </div>
        </div>
        <!-- 输入区 -->
        <div class="chat-input-area">
          <!-- 运行时反馈条（Agent 执行中可补充纠正） -->
          <div v-if="state.chatLoading" class="feedback-bar">
            <input class="feedback-input" v-model="feedbackText" @keydown="onFeedbackKeydown" placeholder="输入补充/纠正信息，Agent 将在下一轮响应中处理..." />
            <button class="feedback-send-btn" @click="sendFeedback" :disabled="!feedbackText.trim()" title="发送反馈"><SvgIcon name="send" :size="14" /></button>
          </div>
          <div class="input-resizer" @mousedown.prevent="startInputResize" title="拖拽调整高度"></div>
          <div v-if="pendingAttachment" class="attachment-badge">
            <div class="att-icon"><SvgIcon :name="pendingAttachment.type === 'file' ? 'file' : 'file-code'" :size="14" /></div>
            <div class="att-info"><span class="att-filename">{{ pendingAttachment.path || pendingAttachment.filename }}</span>
              <span v-if="pendingAttachment.lineStart" class="att-lines">:{{ pendingAttachment.lineStart }}-{{ pendingAttachment.lineEnd }}</span>
              <span class="att-type">{{ pendingAttachment.type === 'file' ? '文件' : '选中代码' }}</span></div>
            <button class="att-close" @click="pendingAttachment = null" title="移除">×</button>
          </div>
          <textarea class="chat-input" ref="inputRef" v-model="inputText" @keydown="onKeydown" @dragover.prevent @drop="handleDrop" @paste="handlePaste" :style="{ height: inputHeight + 'px' }" placeholder="发送消息到 AI... (Enter 发送, Shift+Enter 换行)" :disabled="state.chatLoading"></textarea>
          <div class="input-overlay">
            <div class="overlay-btns">
              <span :class="['obtn', { active: autoReview }]" @click="toggleAuto('autoReview')" title="自动审核：开启=Agent自行审批，关闭=等待用户审批"><SvgIcon name="sparkles" :size="12" /> 审核</span>
              <span :class="['obtn', { active: autoCollapse }]" @click="autoCollapse = !autoCollapse" title="自动折叠：新消息发出时折叠旧输出，显示完成摘要"><SvgIcon name="list" :size="12" /> 折叠</span>
              <span class="obtn-sep"></span>
              <span :class="['obtn', 'obtn-agent', { active: autonomous }]" @click="toggleAuto('autonomous')" title="自主模式：开启=连续执行全部计划步骤，关闭=单次回复"><SvgIcon name="sparkles" :size="12" color="#d4a74e" /> 自主</span>
            </div>
            <button v-if="!state.chatLoading" class="send-btn" @click="sendMessage" :disabled="!inputText.trim()"><SvgIcon name="send" :size="16" /></button>
            <button v-else class="stop-btn" @click="stopChat"><SvgIcon name="close" :size="14" /></button>
          </div>
        </div>
      </div>
      <!-- 右侧：Debug日志面板 / 会话列表 -->
      <DebugLogPanel v-if="showDebugLog" @close="showDebugLog = false" />
      <ConvSidebar v-else :conversations="state.conversations" :current-conv-id="state.currentConvId" :ws-token-stats="wsTokenStats" :conv-ctx-stats="convCtxStats" :ctx-max-tokens-val="state.settings.contextMaxTokens || 1000000" :width="convListWidth" @new-conversation="newConversation" @switch-conversation="switchConv" @delete-conversation="deleteConv" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, inject, onMounted, onUnmounted, nextTick, watch, reactive } from 'vue'
import { state } from '../main.js'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'
import PlanPanel from './PlanPanel.vue'
import ApprovalBar from './ApprovalBar.vue'
import ConvSidebar from './ConvSidebar.vue'
import AskUserCard from './AskUserCard.vue'
// SubAgentBlock 不再使用，替换为内联时间线展示
import MarkdownRenderer from './MarkdownRenderer.vue'
import DebugLogPanel from './DebugLogPanel.vue'

const showDebugLog = ref(false)

const rightPanelWidth = inject('rightPanelWidth')
const toggleRight = () => { state.rightPanelVisible = false }
const inputText = ref('')
const feedbackText = ref('')
const msgRef = ref(null)
const inputRef = ref(null)
const inputHeight = ref(150)
const convListWidth = ref(250)
const topSentinel = ref(null)
let currentAbortSSE = null
const autoReview = ref(true)
const autoIterate = ref(false)
const autoCollapse = ref(true)
const autonomous = ref(false)
const pendingAttachment = ref(null)

// nudge 提示条
let nudgeTimer = null
const currentNudge = ref('')
function showNudge(text) {
  currentNudge.value = text
  if (nudgeTimer) clearTimeout(nudgeTimer)
  nudgeTimer = setTimeout(() => { currentNudge.value = '' }, 4000)
}

let pendingAskCallId = ''
const currentPlan = ref([])
const planExpanded = ref(true)
const currentPhase = ref('')
let phaseTimer = null
let autoSaveTimer = null

// ── 虚拟滚动 ──
const SCROLL_BUFFER = 20
const ESTIMATED_HEIGHT = 100
const msgHeights = reactive({})
const scrollTopRef = ref(0)
const containerHeight = ref(600)
const isNearBottom = ref(true)

// ── 完成报告不再独立展示，由 EventDone 追加为 content segment ──

const approvalState = ref({ callId: '', tool: '', args: '', waiting: false })
const virtualState = computed(() => {
  const msgs = state.messages
  const total = msgs.length
  if (total === 0) return { visible: [], offset: { top: 0, bottom: 0 }, totalHeight: 0 }
  const scrollTop = scrollTopRef.value
  const viewHeight = containerHeight.value || 600
  let acc = 0, startIdx = 0, endIdx = total, foundStart = false, foundEnd = false
  for (let i = 0; i < total; i++) {
    const h = msgHeights[msgs[i]._key] || ESTIMATED_HEIGHT
    if (!foundStart && acc + h > scrollTop - SCROLL_BUFFER * ESTIMATED_HEIGHT) {
      startIdx = Math.max(0, i - Math.floor(SCROLL_BUFFER / 2)); foundStart = true
    }
    if (foundStart && !foundEnd && acc + h > scrollTop + viewHeight + SCROLL_BUFFER * ESTIMATED_HEIGHT) {
      endIdx = Math.min(total, i + Math.floor(SCROLL_BUFFER / 2)); foundEnd = true
    }
    acc += h
  }
  if (!foundStart) startIdx = 0
  if (!foundEnd) endIdx = total
  let topOffset = 0, bottomOffset = 0
  for (let i = 0; i < startIdx; i++) topOffset += msgHeights[msgs[i]._key] || ESTIMATED_HEIGHT
  for (let i = endIdx; i < total; i++) bottomOffset += msgHeights[msgs[i]._key] || ESTIMATED_HEIGHT
  return { visible: msgs.slice(startIdx, endIdx), offset: { top: topOffset, bottom: bottomOffset }, total: total }
})
const visibleMessages = computed(() => virtualState.value.visible)
const virtualOffset = computed(() => virtualState.value.offset)
const hasMoreTop = computed(() => virtualState.value.visible.length < state.messages.length)

function onScroll() {
  if (msgRef.value) {
    scrollTopRef.value = msgRef.value.scrollTop
    containerHeight.value = msgRef.value.clientHeight
    const el = msgRef.value
    isNearBottom.value = el.scrollTop + el.clientHeight >= el.scrollHeight - 150
  }
}

// ── 段模式（兼容旧版）──
function segMode(seg) {
  if (seg._mode) return seg._mode
  if (seg.type === 'thinking') return seg._collapsed !== false ? 'collapsed' : 'expanded'
  if (seg.type === 'tool_call') return seg._expanded ? 'expanded' : 'collapsed'
  if (seg.type === 'ask_user') return 'expanded'
  return seg._collapsed === false ? 'expanded' : 'collapsed'
}

// ── 工具智能分类（简化版）──
function safeParse(json) {
  if (!json) return {}
  try { return JSON.parse(json) } catch { return {} }
}

function toolMeta(seg) {
  const name = seg.name || ''
  const args = safeParse(seg.argsRaw)
  if (/^read_file\b/.test(name)) return { icon: 'file-text', title: '读取文件', detail: args.path || '', summary: '已读取', resultIcon: 'check' }
  if (/^write_file\b/.test(name)) return { icon: 'file-plus', title: '写入文件', detail: args.path || '', summary: '已写入', resultIcon: 'check' }
  if (/^edit_file\b/.test(name)) return { icon: 'edit', title: '编辑文件', detail: args.path || '', summary: '已编辑', resultIcon: 'check' }
  if (/^run_command\b/.test(name)) return { icon: 'terminal', title: '执行命令', detail: (args.command || '').slice(0, 40), summary: '已完成', resultIcon: 'check' }
  if (/^search_content\b/.test(name)) return { icon: 'search', title: '搜索内容', detail: (args.pattern || '').slice(0, 40), summary: '已搜索', resultIcon: 'check' }
  if (/^search_files\b/.test(name)) return { icon: 'search', title: '搜索文件', detail: (args.pattern || '').slice(0, 40), summary: '已搜索', resultIcon: 'check' }
  if (/^web_search\b/.test(name)) return { icon: 'globe', title: '网络搜索', detail: (args.query || '').slice(0, 40), summary: '已搜索', resultIcon: 'globe' }
  if (/^git_status\b/.test(name)) return { icon: 'source-control', title: 'Git 状态', detail: '', summary: '已查看', resultIcon: 'check' }
  return { icon: 'wrench', title: seg.name || '工具调用', detail: '', summary: (seg.result || '').slice(0, 80), resultIcon: 'check' }
}

function toolResultSummary(seg) {
  const meta = toolMeta(seg)
  if (meta.summary) return meta.summary
  const r = seg.result || ''
  return r.length > 120 ? r.slice(0, 120) + '…' : r
}

function isTerminalTool(seg) {
  return /^(run_command|run_test|run_background|go_build|go_run|code_fix|code_format)$/.test(seg.name || '')
}

function formatTerminalCommand(seg) {
  const args = safeParse(seg.argsRaw)
  if ((seg.name || '') === 'run_command') return '$ ' + (args.command || '')
  return '$ ' + (seg.argsRaw || '')
}

// ── 工作区 Token 统计 ──
const wsTokenStats = reactive({ promptTokens: 0, completionTokens: 0, totalTokens: 0, cacheHitTokens: 0, cacheMissTokens: 0, systemTokens: 0, skillsTokens: 0, mcpTokens: 0, toolTokens: 0, historyTokens: 0, otherTokens: 0 })
const convCtxStats = reactive({ promptTokens: 0, completionTokens: 0, cacheHitTokens: 0, cacheMissTokens: 0, systemTokens: 0, skillsTokens: 0, mcpTokens: 0, toolTokens: 0, historyTokens: 0, otherTokens: 0 })

const loadWsTokenStats = async () => {
  try {
    const data = await api.apiGet('/tokens/stats')
    if (data) Object.assign(wsTokenStats, data)
    if (state.currentConvId) {
      const ts = await api.apiGet('/conversations/' + state.currentConvId + '/token-stats')
      if (ts && ts.promptTokens !== undefined) Object.assign(convCtxStats, ts)
    }
  } catch {}
}

// ── SSE 事件处理 ──
let msgKeyCounter = 0
function makeMsgKey() { return 'msg_' + Date.now() + '_' + (msgKeyCounter++) }

function pushSegment(segs, type, initial) {
  const last = segs[segs.length - 1]
  if (last && last.type === type) return last
  const seg = { type, content: '', ...initial }
  segs.push(seg)
  return seg
}

function msgSummary(msg) {
  if (!msg.segments || msg.segments.length === 0) return '已完成'
  let toolCount = 0, hasContent = false, summaryText = ''
  for (const seg of msg.segments) {
    if (seg.type === 'tool_call') toolCount++
    if (seg.type === 'content' && seg.content) { hasContent = true; summaryText = seg.content.replace(/^#+\s*/, '').slice(0, 60) }
  }
  const parts = []
  if (toolCount > 0) parts.push(toolCount + ' 步工具调用')
  if (summaryText) parts.push('「' + summaryText + '…」')
  if (!hasContent && toolCount === 0) parts.push('已完成')
  return parts.join(' · ')
}

function collapsePreviousOutputs() {
  if (!autoCollapse.value) return
  for (const msg of state.messages) {
    if (msg.role !== 'assistant' || msg._loading) continue
    if (!msg.segments || msg.segments.length === 0) continue
    for (const seg of msg.segments) {
      if (seg.type === 'thinking') seg._collapsed = true
      if (seg.type === 'tool_call') seg._expanded = false
    }
    msg._folded = true
  }
}

const sendMessage = async () => {
  const text = inputText.value.trim()
  if (!text && !pendingAttachment.value) return
  if (state.chatLoading) return
  if (!state.currentConvId) {
    try {
      const conv = await api.apiPost('/conversations', { title: '新对话' })
      state.currentConvId = conv.id
      state.conversations.unshift({ id: conv.id, title: conv.title, createdAt: conv.createdAt, updatedAt: conv.updatedAt })
      convCtxStats.promptTokens = 0; convCtxStats.completionTokens = 0
    } catch {}
  }
  const userContent = text || ''
  let fullContent = userContent
  if (pendingAttachment.value) {
    fullContent += '\n\n---\n[附件] ' + (pendingAttachment.value.content || '').slice(0, 2000)
  }
  const lastUserText = text
  inputText.value = ''; pendingAttachment.value = null
  if (currentAbortSSE) { currentAbortSSE(); currentAbortSSE = null }
  collapsePreviousOutputs()
  const userMsg = { role: 'user', content: fullContent, segments: [], toolCalls: [], _key: makeMsgKey(), _idx: state.messages.length, _time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) }
  state.messages.push(userMsg)
  if (state.currentConvId) {
    await api.apiPost('/conversations/' + state.currentConvId + '/messages', { role: 'user', content: fullContent }).catch(() => {})
  }
  state.chatLoading = true; state.agentRunning = true
  if (!state.chatSessionId) state.chatSessionId = 'sess_' + Date.now()
  const msgIdx = state.messages.length
  const assistantMsg = { role: 'assistant', content: '', segments: [], toolCalls: [], _key: makeMsgKey(), _idx: msgIdx, _time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }), _loading: true }
  state.messages.push(assistantMsg)
  let finalContent = ''
  currentAbortSSE = api.chatSSE(fullContent, state.chatSessionId, autonomous.value, state.currentConvId, {
    onEvent: (data) => {
      const msg = state.messages[msgIdx]
      if (!msg) return; msg._loading = false
      if (data.type === 'thinking') { const seg = pushSegment(msg.segments, 'thinking', { _mode: 'collapsed', _collapsed: true }); seg.content += data.content || '' }
      else if (data.type === 'content') { finalContent += data.content || ''; const seg = pushSegment(msg.segments, 'content'); seg.content += data.content || '' }
      else if (data.type === 'tool_call') {
        const toolName = data.tool || data.name || ''
        if (toolName === 'finish_task') {
          // finish_task 不是普通工具调用——不创建 segment，结果由 EventDone 展示为完成报告
        } else if (toolName === 'ask_user') {
          let question = ''
          try { const args = typeof data.args === 'string' ? JSON.parse(data.args) : data.args; question = args.question || '（无问题内容）' } catch {}
          pendingAskCallId = data.callId || data.callID || ''
          msg.segments.push({ type: 'ask_user', question, callId: data.callId || data.callID || '', answer: '', _answered: false })
        } else {
          msg.segments.push({ type: 'tool_call', name: toolName, argsRaw: data.args ? (typeof data.args === 'string' ? data.args : JSON.stringify(data.args, null, 2)) : '', result: '', _mode: 'expanded', _expanded: true })
        }
      } else if (data.type === 'error') { const seg = pushSegment(msg.segments, 'content'); seg.content += '**[错误]** ' + (data.content || '') }
      else if (data.type === 'usage' && data.usage) {
        const u = data.usage
        wsTokenStats.promptTokens += u.prompt_tokens || 0; wsTokenStats.completionTokens += u.completion_tokens || 0
        wsTokenStats.totalTokens += (u.prompt_tokens || 0) + (u.completion_tokens || 0)
        convCtxStats.promptTokens = u.prompt_tokens || 0; convCtxStats.completionTokens = u.completion_tokens || 0
        // 填充上下文构成 breakdown（来自 PromptBreakdown 估算）
        if (u.prompt_breakdown) {
          const pb = u.prompt_breakdown
          convCtxStats.systemTokens = pb.system_tokens || 0
          convCtxStats.skillsTokens = pb.skills_tokens || 0
          convCtxStats.mcpTokens = pb.mcp_tokens || 0
          convCtxStats.toolTokens = pb.tool_tokens || 0
          convCtxStats.historyTokens = pb.history_tokens || 0
          convCtxStats.otherTokens = pb.other_tokens || 0
        }
      } else if (data.type === 'phase') {
        currentPhase.value = data.content || ''
        if (phaseTimer) clearTimeout(phaseTimer)
        phaseTimer = setTimeout(() => { currentPhase.value = '' }, 6000)
      } else if (data.type === 'notice') {
        const nudgeText = (data.content || '').replace(/\n/g, ' ').slice(0, 120)
        showNudge(nudgeText)
      } else if (data.type === 'done') {
        // 完成报告：直接追加为 content segment，不独立展示为组件
        // 清空 finalContent，避免 onDone 把旧内容写入 msg.content
        finalContent = ''
        if (data.content && msg) {
          msg.segments.push({ type: 'content', content: data.content })
        }
      }
      scrollToBottom()
      scrollToBottom()
    },
    onError: (err) => {
      const msg = state.messages[msgIdx]
      if (msg) { msg._loading = false; pushSegment(msg.segments, 'content').content += '**[连接错误]** ' + err }
      state.chatLoading = false; state.agentRunning = false; currentAbortSSE = null
    },
    onReconnect: (attempt, maxAttempts, delay) => {
      const msg = state.messages[msgIdx]
      if (msg) { const seg = pushSegment(msg.segments, 'content'); seg.content += '\n\n> [重连] ' + attempt + '/' + maxAttempts + '\n\n' }
      scrollToBottom()
    },
    onDone: () => {
      const msg = state.messages[msgIdx]
      if (msg) { msg._loading = false; msg.content = finalContent }
      state.chatLoading = false; state.agentRunning = false; currentAbortSSE = null
      if (phaseTimer) { clearTimeout(phaseTimer); phaseTimer = null }
      currentPhase.value = ''
      if (state.currentConvId && finalContent) {
        api.apiPost('/conversations/' + state.currentConvId + '/messages', { role: 'assistant', content: finalContent }).catch(() => {})
      }
      // 自动命名对话：用用户消息更新标题
      if (state.currentConvId && lastUserText) {
        autoNameConv(state.currentConvId, lastUserText)
      }
      window.dispatchEvent(new Event('save-conversations'))
      window.dispatchEvent(new Event('save-conversations'))
      scrollToBottom()
    },
  })
}

const stopChat = async () => {
  try { await api.apiGet('/chat/stop?sessionId=' + state.chatSessionId) } catch {}
  state.chatLoading = false; state.agentRunning = false
}

// ── 运行时反馈：Agent 执行中用户补充/纠正 ──
const sendFeedback = async () => {
  const text = feedbackText.value.trim()
  if (!text || !state.chatSessionId) return
  feedbackText.value = ''
  try {
    await api.sendFeedback(state.chatSessionId, text)
  } catch {}
}
const onFeedbackKeydown = (e) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendFeedback()
  }
}

const onAskAnswer = (seg, { callId, answer }) => {
  if (!answer) return; seg.answer = answer
  submitAskAnswer(seg)
}

const submitAskAnswer = async (seg) => {
  const answer = (seg.answer || '').trim()
  if (!answer) return; seg._answered = true
  try { await api.answerChat(state.chatSessionId, answer) } catch {}
}

const resolveApproval = async (approved) => {
  const a = approvalState.value
  if (!a.callId || !a.waiting) return; a.waiting = false
  try { await api.approveChat(state.chatSessionId, a.callId, approved) } catch { a.waiting = true }
}

const onKeydown = (e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMessage() } }

const scrollToBottom = () => {
  nextTick(() => { if (msgRef.value && isNearBottom.value) { msgRef.value.scrollTop = msgRef.value.scrollHeight; onScroll() } })
}

const forceScrollToBottom = () => {
  nextTick(() => { if (msgRef.value) { msgRef.value.scrollTop = msgRef.value.scrollHeight; isNearBottom.value = true; onScroll() } })
}

const loadConvList = async () => {
  try { const list = await api.apiGet('/conversations'); state.conversations = list || [] } catch {}
}

const newConversation = async () => {
  try {
    const conv = await api.apiPost('/conversations', { title: '新对话' })
    state.conversations.unshift({ id: conv.id, title: conv.title, createdAt: conv.createdAt, updatedAt: conv.updatedAt })
    state.currentConvId = conv.id; state.messages = []; currentPlan.value = []
    convCtxStats.promptTokens = 0; convCtxStats.completionTokens = 0; inputText.value = ''
  } catch {}
}

const deleteConv = async (id) => {
  try {
    await api.apiDelete('/conversations/' + id)
    state.conversations = state.conversations.filter(c => c.id !== id)
    if (state.currentConvId === id) { state.currentConvId = ''; state.messages = []; convCtxStats.promptTokens = 0; convCtxStats.completionTokens = 0 }
  } catch {}
}

const switchConv = async (id) => {
  state.currentConvId = id; state.messages = []; currentPlan.value = []
  convCtxStats.promptTokens = 0; convCtxStats.completionTokens = 0
  try {
    const ts = await api.apiGet('/conversations/' + id + '/token-stats')
    if (ts && ts.promptTokens !== undefined) Object.assign(convCtxStats, ts)
  } catch {}
  try {
    const msgs = await api.apiGet('/conversations/' + id + '/messages')
    state.messages = (msgs || []).map((m, idx) => ({
      role: m.role, content: m.content || '', toolCalls: m.toolCalls || [], segments: [], _key: 'msg_' + Date.now() + '_' + idx, _idx: idx, _time: m.createdAt ? m.createdAt.slice(11, 19) : '',
    }))
  } catch {}
  forceScrollToBottom()
}

const toggleAuto = async (field) => {
  const oldVal = !!state.settings[field]
  const newVal = !oldVal
  state.settings[field] = newVal
  // 同步 local ref（浅 watch 不触发，需要手动同步）
  if (field === 'autoReview') autoReview.value = newVal
  else if (field === 'autonomous') autonomous.value = newVal
  try { await api.apiPut('/settings', state.settings) } catch { state.settings[field] = oldVal; if (field === 'autoReview') autoReview.value = oldVal; else if (field === 'autonomous') autonomous.value = oldVal }
}

const autoNameConv = async (convId, content) => {
  if (!convId || !content) return
  try {
    let title = content.replace(/```[\s\S]*?```/g, '').replace(/[#*>`_~\[\]\(\)]/g, '').replace(/\s+/g, ' ').replace(/^[\s,;:，；：、。.！!？?]+/, '').trim()
    if (title.length > 30) title = title.slice(0, 28) + '…'
    if (title.length === 0) { title = content.replace(/```[\s\S]*?```/g, '').replace(/\s+/g, ' ').trim(); if (title.length > 30) title = title.slice(0, 28) + '…'; if (title.length === 0) return }
    await api.apiPut('/conversations/' + convId, { title })
    const conv = state.conversations.find(c => c.id === convId)
    if (conv) conv.title = title
  } catch {}
}

function phaseIcon(phase) {
  if (phase.includes('规划')) return 'list'
  if (phase.includes('探索')) return 'search'
  if (phase.includes('执行')) return 'terminal'
  if (phase.includes('验证')) return 'check'
  if (phase.includes('评测')) return 'layers'
  if (phase.includes('完成')) return 'check'
  if (phase.includes('继续')) return 'send'
  return 'cycle'
}

function handleTaskTool(data) {
  const toolName = data.tool || data.name || ''
  const taskTools = ['update_plan', 'task_create', 'task_update', 'task_list', 'task_delete', 'task_summary']
  if (!taskTools.includes(toolName)) return false
  try {
    const args = data.args ? (typeof data.args === 'string' ? JSON.parse(data.args) : data.args) : {}
    if (toolName === 'update_plan' && Array.isArray(args.plan)) { currentPlan.value = args.plan; planExpanded.value = true; return true }
    if (toolName === 'task_create') { currentPlan.value.push({ step: args.subject || '(新建任务)', status: 'pending', callId: data.callId || data.callID || '' }); planExpanded.value = true; return true }
    return true
  } catch { return false }
}

// ── 输入框拖拽调整 ──
let inputDragging = false
let inputStartY = 0
let inputStartH = 0
const startInputResize = (e) => {
  inputDragging = true; inputStartY = e.clientY; inputStartH = inputHeight.value
  document.addEventListener('mousemove', onInputResizeMove); document.addEventListener('mouseup', stopInputResize)
}
const onInputResizeMove = (e) => { if (!inputDragging) return; inputHeight.value = Math.max(80, Math.min(600, inputStartH + (inputStartY - e.clientY))) }
const stopInputResize = () => { inputDragging = false; document.removeEventListener('mousemove', onInputResizeMove); document.removeEventListener('mouseup', stopInputResize) }

// ── 文件拖拽/粘贴 ──
const handleDrop = (e) => {
  e.preventDefault()
  const files = e.dataTransfer?.files
  if (files && files.length > 0) {
    const file = files[0]; const reader = new FileReader()
    reader.onload = (ev) => { pendingAttachment.value = { type: 'file', path: file.name, filename: file.name, content: typeof ev.target?.result === 'string' ? ev.target.result.slice(0, 50000) : '' } }
    reader.readAsText(file, 'utf-8'); return
  }
  const textData = e.dataTransfer?.getData('text/plain')
  if (textData) { inputText.value += textData; inputRef.value?.focus() }
}

const handlePaste = (e) => {
  const items = e.clipboardData?.items; if (!items) return
  for (const item of items) {
    if (item.kind === 'file') {
      e.preventDefault(); const file = item.getAsFile(); if (!file) continue
      if (file.type.startsWith('image/')) {
        const reader = new FileReader()
        reader.onload = (ev) => { pendingAttachment.value = { type: 'image', path: file.name, filename: file.name, content: ev.target?.result || '' } }
        reader.readAsDataURL(file)
      } else {
        const reader = new FileReader()
        reader.onload = (ev) => { pendingAttachment.value = { type: 'file', path: file.name, filename: file.name, content: typeof ev.target?.result === 'string' ? ev.target.result.slice(0, 50000) : '' } }
        reader.readAsText(file, 'utf-8')
      }
      return
    }
  }
}

// ResizeObserver
let resizeObserver = null
const reObserveItems = () => {
  if (!msgRef.value || !resizeObserver) return
  msgRef.value.querySelectorAll('.msg-item').forEach(el => resizeObserver.observe(el))
}
const setupResizeObserver = () => {
  nextTick(() => {
    if (!msgRef.value) return
    if (!resizeObserver) {
      resizeObserver = new ResizeObserver((entries) => {
        let heightChanged = false
        for (const entry of entries) {
          const el = entry.target; const idx = el.dataset.msgIdx
          if (idx !== undefined) {
            const key = state.messages[Number(idx)]?._key
            if (key) {
              const oldH = msgHeights[key]; msgHeights[key] = entry.contentRect.height
              const keys = Object.keys(msgHeights)
              if (keys.length > 150) { const toDelete = keys.slice(0, keys.length - 150); for (const k of toDelete) delete msgHeights[k] }
              if (oldH && oldH !== entry.contentRect.height) heightChanged = true
            }
          }
        }
        if (heightChanged && isNearBottom.value && msgRef.value) { msgRef.value.scrollTop = msgRef.value.scrollHeight }
      })
    }
    reObserveItems()
  })
}

watch(() => state.messages.length, () => { nextTick(setupResizeObserver) })

watch(() => state.settings, (s) => { if (s) { autoReview.value = s.autoReview !== undefined ? !!s.autoReview : true; autoIterate.value = !!s.autoIterateOnRejection; autonomous.value = !!s.autonomous; autoCollapse.value = s.autoCollapse !== undefined ? !!s.autoCollapse : true; } }, { immediate: true })

const handleBeforeUnload = () => { if (state.currentConvId && state.messages.length > 0) { window.dispatchEvent(new Event('save-conversations')) } }

onMounted(() => {
  loadWsTokenStats(); loadConvList(); scrollToBottom(); setupResizeObserver()
  containerHeight.value = msgRef.value?.clientHeight || 600
  window.addEventListener('add-to-chat', (e) => {
    const detail = e.detail; if (!detail) return
    pendingAttachment.value = { type: detail.type || 'file', path: detail.path || '', filename: detail.filename || '', lineStart: detail.lineStart || null, lineEnd: detail.lineEnd || null, content: detail.content || '' }
  })
  window.addEventListener('workspace-switched', () => {
    loadConvList(); state.currentConvId = ''; state.messages = []; state.chatLoading = false; state.agentRunning = false; state.chatSessionId = ''; inputText.value = ''
    convCtxStats.promptTokens = 0; convCtxStats.completionTokens = 0; currentPlan.value = []
  })
  autoSaveTimer = setInterval(() => { if (state.currentConvId && state.messages.length > 0) { window.dispatchEvent(new Event('save-conversations')) } }, 15000)
  window.addEventListener('beforeunload', handleBeforeUnload)
})

onUnmounted(() => {
  if (autoSaveTimer) { clearInterval(autoSaveTimer); autoSaveTimer = null }
  if (phaseTimer) { clearTimeout(phaseTimer); phaseTimer = null }
  if (currentAbortSSE) { currentAbortSSE(); currentAbortSSE = null }
  document.removeEventListener('mousemove', onInputResizeMove); document.removeEventListener('mouseup', stopInputResize)
  if (resizeObserver) { resizeObserver.disconnect(); resizeObserver = null }
  window.removeEventListener('beforeunload', handleBeforeUnload)
})
</script>

<style scoped>
.right-panel { flex: 1; display: flex; flex-direction: column; overflow: hidden; background: var(--bg-secondary); min-width: 0; }
.rp-header { display: flex; align-items: center; justify-content: space-between; padding: 8px 12px; border-bottom: 1px solid var(--border-color); font-size: 13px; flex-shrink: 0; }
.rp-header-title { display: flex; align-items: center; gap: 6px; }
.rp-header-actions { display: flex; gap: 4px; }
.rp-btn { background: none; border: 1px solid transparent; color: var(--text-secondary); padding: 2px 6px; cursor: pointer; border-radius: 3px; display: flex; align-items: center; }
.rp-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.rp-body { flex: 1; display: flex; flex-direction: row; overflow: hidden; min-height: 0; }
.chat-area { flex: 1; display: flex; flex-direction: column; min-width: 0; overflow: hidden; max-width: 100%; }
.chat-messages { flex: 1; overflow-y: auto; padding: 8px 12px; min-height: 0; scroll-behavior: smooth; }
.msg-list-wrap { display: flex; flex-direction: column; gap: 8px; min-height: 100%; }
.msg-item { display: flex; gap: 8px; align-items: flex-start; content-visibility: auto; contain-intrinsic-size: 60px; }
.msg-user { flex-direction: row-reverse; justify-content: flex-start; }
.bubble-user {
  flex: 0 1 auto;
  width: fit-content;
  max-width: 85%;
  min-width: 60px;
  background: var(--accent);
  color: #fff;
  padding: 8px 14px;
  border-radius: 16px 16px 4px 16px;
  align-self: flex-end;
  margin-left: auto;
  overflow-wrap: break-word;
  word-break: break-word;
  overflow: hidden;
}
.user-msg-content { width: 100%; }
.user-msg-content :deep(p) { margin: 2px 0; white-space: pre-wrap; word-break: break-word; }
.user-msg-content :deep(pre) { white-space: pre-wrap; font-size: 12px; background: rgba(0,0,0,0.15); padding: 6px; border-radius: 4px; max-width: 100%; overflow-x: auto; }
.user-msg-content :deep(code) { font-size: 12px; }
.user-msg-placeholder { color: rgba(255,255,255,0.4); font-style: italic; font-size: 12px; }
.msg-avatar { width: 28px; height: 28px; border-radius: 50%; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.msg-user .msg-avatar { background: var(--accent); color: #fff; }
.msg-assistant .msg-avatar { background: var(--bg-tertiary); color: var(--text-secondary); border: 1px solid var(--border-color); }
.msg-bubble { flex: 1; min-width: 0; max-width: 85%; font-size: 13px; line-height: 1.6; word-break: break-word; overflow-wrap: break-word; position: relative; padding: 2px 0; }

.bubble-assistant { background: transparent; color: var(--text-primary); padding: 2px 0; }
.bubble-agent { background: transparent; border: none; padding: 0 0 0 18px; position: relative; }
.bubble-agent::before { content: ''; position: absolute; left: 8px; top: 0; bottom: 0; width: 2px; background: linear-gradient(180deg, var(--accent) 0%, var(--border-color) 100%); opacity: 0.4; border-radius: 1px; }
.msg-time { font-size: 10px; color: var(--text-muted); margin-top: 4px; opacity: 0.7; text-align: right; }
.bubble-user .msg-time { color: rgba(255,255,255,0.6); }
.msg-loading-dots { display: flex; align-items: center; gap: 3px; padding: 8px 12px; }
.msg-loading-dots .dot { width: 6px; height: 6px; border-radius: 50%; background: var(--text-muted); animation: dotPulse 1.4s infinite; }
.msg-loading-dots .dot:nth-child(2) { animation-delay: 0.2s; }
.msg-loading-dots .dot:nth-child(3) { animation-delay: 0.4s; }
@keyframes dotPulse { 0%, 60%, 100% { opacity: 0.3; transform: scale(0.8); } 30% { opacity: 1; transform: scale(1.2); } }
.msg-loading-banner { display: flex; align-items: center; justify-content: center; gap: 8px; padding: 8px; color: var(--text-muted); font-size: 12px; }
.phase-bar { display: flex; align-items: center; gap: 6px; padding: 4px 12px; background: linear-gradient(90deg, rgba(212, 167, 78, 0.08), rgba(212, 167, 78, 0.02)); border-bottom: 1px solid rgba(212, 167, 78, 0.2); font-size: 12px; color: #d4a74e; flex-shrink: 0; }
.chat-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%; min-height: 200px; color: var(--text-muted); }
.folded-summary { display: flex; align-items: center; gap: 5px; padding: 5px 10px; background: var(--bg-primary); border: 1px solid var(--border-color); border-left: 3px solid var(--accent); border-radius: 6px; font-size: 12px; cursor: pointer; }
/* ── 时间线展示（替代旧 SubAgentBlock 卡片 + content-flow）── */
.bubble-agent { position: relative; }
.bubble-agent::before { content: ''; position: absolute; left: 8px; top: 0; bottom: 0; width: 2px; background: linear-gradient(180deg, var(--accent) 0%, var(--border-color) 100%); opacity: 0.4; border-radius: 1px; }
.tl-item { display: flex; align-items: flex-start; gap: 0; padding: 2px 0; position: relative; }
.tl-dot { position: absolute; left: 8px; top: 7px; width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; border: 2px solid var(--border-color); background: var(--bg-primary); z-index: 1; }
.tl-dot-thinking { border-color: var(--accent); background: var(--accent-bg); }
.tl-dot-tool { border-color: #d4a74e; background: rgba(212,167,78,0.15); }
.tl-dot-ask { border-color: #c586c0; background: rgba(197,134,192,0.15); }
.tl-dot-content { border-color: #6a9955; background: rgba(106,153,85,0.15); }
.tl-body { flex: 1; min-width: 0; font-size: 13px; line-height: 1.6; padding-left: 20px; }
/* ── 思考段：默认折叠，展开后末尾有 sticky 收起按钮 ── */
.tl-think-body { position: relative; }
.tl-thinking-text { color: var(--text-secondary); font-style: italic; white-space: pre-wrap; padding: 2px 0; max-height: 300px; overflow-y: auto; }
.tl-think-fold { position: sticky; bottom: 0; display: inline-block; font-size: 11px; color: var(--accent); cursor: pointer; padding: 3px 10px; margin-top: 4px; user-select: none; background: var(--bg-tertiary); border: 1px solid var(--border-color); border-radius: 4px; transition: background 0.15s; }
.tl-think-fold:hover { background: var(--bg-hover); color: var(--accent-light); }
.tl-thinking-collapsed { color: var(--text-muted); font-style: italic; font-size: 12px; cursor: pointer; padding: 2px 0; }
.tl-tc-header { display: flex; align-items: center; gap: 4px; cursor: pointer; padding: 2px 0; user-select: none; border-radius: 3px; }
.tl-tc-header:hover { background: var(--bg-hover); }
.tl-tc-chevron { font-size: 9px; color: var(--text-muted); width: 8px; text-align: center; flex-shrink: 0; }
.tl-tc-icon { flex-shrink: 0; color: var(--text-secondary); }
.tl-tc-name { font-size: 12px; font-weight: 500; color: var(--text-primary); font-family: var(--font-code); }
.tl-tc-summary { font-size: 11px; color: var(--text-muted); margin-left: 4px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; min-width: 0; }
.tl-tc-detail { padding: 4px 0 4px 16px; }
.tl-tc-section { margin-bottom: 4px; }
.tl-tc-section-title { font-size: 10px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 2px; font-weight: 500; }
.tl-tc-section pre { background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 4px; padding: 6px 8px; font-size: 11px; color: var(--text-secondary); max-height: 150px; overflow: auto; white-space: pre-wrap; font-family: var(--font-code); margin: 0; }
.tl-tc-command { background: #1e1e1e; color: #d4d4d4; padding: 6px 10px 6px 14px; border-radius: 4px; font-family: var(--font-code); font-size: 12px; white-space: pre-wrap; border: 1px solid var(--border-color); }
.tl-tc-output { background: #1e1e1e; color: #6a9955; padding: 8px 10px; border-radius: 4px; font-family: var(--font-code); font-size: 11px; white-space: pre-wrap; max-height: 200px; overflow: auto; border: 1px solid var(--border-color); }
/* Content 段：纯 Markdown，无多余装饰 */
.tl-content-item .tl-body :deep(p) { margin: 4px 0; }
.tl-content-item .tl-body :deep(pre) { margin: 4px 0; white-space: pre-wrap; word-break: break-word; font-size: 12px; }
.tl-content-item .tl-body :deep(code) { font-size: 12px; }

/* ── nudge 提示条 ── */
.chat-nudge-bar { position: sticky; bottom: 0; z-index: 20; margin: 4px 12px; padding: 4px 10px; border-radius: 4px; background: var(--bg-tertiary); border: 1px solid var(--border-color); font-size: 11px; color: var(--text-muted); text-align: center; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; animation: nudgeFadeIn 0.3s ease; }
@keyframes nudgeFadeIn { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: translateY(0); } }

/* ── 完成报告卡已移除，由 EventDone 追加为 content segment ── */

/* ── 输入区 ── */
.chat-input-area { position: relative; flex-shrink: 0; padding: 0 8px 10px 8px; background: var(--bg-secondary); }
.input-resizer { position: absolute; top: -8px; left: 0; right: 0; height: 12px; cursor: ns-resize; z-index: 10; }
.chat-input { display: block; width: 100%; background: var(--input-bg); border: 1px solid var(--border-color); color: var(--text-primary); padding: 12px 16px 64px 16px; border-radius: 8px; font-size: 14px; resize: none; outline: none; min-height: 80px; font-family: inherit; line-height: 1.6; box-sizing: border-box; }
.input-overlay { position: absolute; right: 16px; bottom: 20px; display: flex; align-items: center; gap: 6px; pointer-events: none; }
.input-overlay > * { pointer-events: auto; }
.overlay-btns { display: flex; align-items: center; gap: 2px; }
.obtn { display: flex; align-items: center; gap: 3px; padding: 4px 8px; border-radius: 4px; cursor: pointer; font-size: 11px; color: var(--text-muted); background: var(--bg-tertiary); border: 1px solid var(--border-color); white-space: nowrap; user-select: none; }
.obtn.active { color: var(--accent); background: rgba(212, 167, 78, 0.1); border-color: rgba(212, 167, 78, 0.3); }
.obtn-obtn-agent.active { color: #d4a74e; }
.send-btn { background: var(--accent); color: #fff; padding: 6px 14px; border-radius: 4px; cursor: pointer; border: none; }
.stop-btn { background: #c03; color: #fff; padding: 6px 14px; border-radius: 4px; cursor: pointer; border: none; }
.attachment-badge { display: flex; align-items: center; gap: 6px; padding: 4px 8px; margin: 4px 0; background: var(--bg-tertiary); border: 1px solid var(--border-color); border-radius: 4px; font-size: 12px; }

/* ── SubAgentBlock 内部样式已移除，替换为时间线展示 ── */
.seg-content { line-height: 1.6; white-space: pre-wrap; word-break: break-word; }

/* ── 运行时反馈条 ── */
.feedback-bar {
  display: flex; align-items: center; gap: 4px;
  padding: 4px 8px; margin: 0 0 4px 0;
  background: var(--bg-tertiary); border: 1px solid var(--border-color);
  border-radius: 6px;
}
.feedback-input {
  flex: 1; background: transparent; border: none; outline: none;
  color: var(--text-primary); font-size: 12px; padding: 4px 0;
  font-family: inherit;
}
.feedback-input::placeholder { color: var(--text-muted); font-size: 11px; }
.feedback-send-btn {
  background: var(--accent); color: #fff; border: none;
  padding: 4px 8px; border-radius: 4px; cursor: pointer;
  display: flex; align-items: center; flex-shrink: 0;
}
.feedback-send-btn:disabled { opacity: 0.4; cursor: default; }

.scroll-more-hint { text-align: center; font-size: 11px; color: var(--text-muted); padding: 4px; }
.tool-calls { margin-top: 4px; }
.tool-call { background: var(--bg-primary); padding: 4px 8px; border-radius: 3px; margin-bottom: 2px; font-size: 12px; }
</style>
