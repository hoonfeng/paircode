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
        <!-- 消息区（滚动容器） -->
        <div class="chat-messages" ref="msgRef" @scroll="onScroll">
          <!-- 顶部加载更多提示 -->
          <div v-if="hasMoreTop" class="scroll-more-hint" ref="topSentinel">
            <span>加载更早消息...</span>
          </div>
          <!-- 渲染的消息列表（全量渲染，overflow-anchor 保底部稳定） -->
          <div class="msg-list-wrap">
            <div v-for="msg in state.messages" :key="msg._key"
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
                  <!-- agent 委派 / 反馈 徽章 -->
                  <div v-if="isDelegation(msg)" class="user-msg-header">
                    <span class="umh-badge badge-delegation"><SvgIcon name="git-branch" :size="10" /> 委派任务</span>
                    <span class="umh-agent">{{ delegationAgent(msg) }}</span>
                  </div>
                  <div v-else-if="isFeedback(msg)" class="user-msg-header">
                    <span class="umh-badge badge-feedback"><SvgIcon name="message-square" :size="10" /> 用户反馈</span>
                  </div>
                  <div v-if="msg.content" class="user-msg-content">
                    <MarkdownRenderer :text="cleanMsgContent(msg)" :theme="state.theme" />
                  </div>
                  <div v-else class="user-msg-placeholder">（空消息）</div>
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
                            <span v-if="toolMeta(seg).detail" class="tl-tc-param">{{ toolMeta(seg).detail }}</span>
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
                        <div class="tl-body"><AskUserCard :question="seg.question" :ask-type="seg.askType" :options="seg.options" :call-id="seg.callId" :answered="seg._answered" @answer="onAskAnswer(seg, $event)" /></div>
                      </div>
                      <!-- Content：纯 Markdown，无边框无圆点包裹 -->
                      <div v-else-if="seg.type === 'content'" class="tl-item tl-content-item">
                        <span class="tl-dot tl-dot-content"></span>
                        <div class="tl-body"><MarkdownRenderer :text="seg.content" :theme="state.theme" /></div>
                      </div>
                    </template>
                  </template>
                </template>
                <!-- 🔽 折叠整条输出（展开后可用） -->
                <div v-if="!msg._folded && msg.role === 'assistant' && msg.segments && msg.segments.length > 0" class="msg-fold-btn" @click="msg._folded = true">
                  <SvgIcon name="chevron-up" :size="12" />
                  <span>折叠输出</span>
                </div>
                <!-- 历史消息 fallback：assistant 有 content 但无 segments（从 API 加载的历史对话） -->
                <template v-if="msg.role === 'assistant' && (!msg.segments || msg.segments.length === 0)">
                  <div v-if="msg.content" class="tl-item tl-content-item">
                    <span class="tl-dot tl-dot-content"></span>
                    <div class="tl-body"><MarkdownRenderer :text="msg.content" :theme="state.theme" /></div>
                  </div>
                </template>
                <div v-if="msg._time" class="msg-time">{{ msg._time }}</div>
              </div>
              <div v-if="msg._loading" class="msg-loading-dots">
                <span class="dot"></span><span class="dot"></span><span class="dot"></span>
              </div>
            </div>
          </div>
          <div v-if="state.chatLoading && state.messages && state.messages.length > 0" class="msg-loading-banner">
            <span class="dot-pulse"></span><span>思考中...</span>
          </div>
          <div v-if="(!state.messages || state.messages.length === 0) && !state.chatLoading" class="chat-empty">
            <div class="chat-empty-icon"><SvgIcon name="bot" :size="32" /></div>
            <div class="chat-empty-text">开始新的对话</div>
            <div class="chat-empty-hint">发送消息即可与 AI 助手对话</div>
          </div>
          <!-- 新消息跳底按钮 -->
          <div v-if="showScrollDown" class="scroll-down-btn" :class="{ 'show-pulse': state.chatLoading }" @click.stop="scrollToBottom">
            <button><SvgIcon name="chevron-down" :size="14" /> 新消息</button>
          </div>
        </div>
        <!-- 执行步骤面板（update_plan 传入，由外层 AutonomousController 使用） -->
        <div class="plan-container" :class="{ 'plan-empty': currentPlan.length === 0 && currentTasks.length === 0 }">
          <PlanPanel v-if="currentPlan.length > 0" :plan="currentPlan" :expanded="planExpanded" @toggle="planExpanded = !planExpanded" />
          <!-- 子任务进度面板（task_create/task_update 创建的子任务，由内层 Loop 使用） -->
          <TaskPanel v-if="currentTasks.length > 0" :tasks="currentTasks" :expanded="tasksExpanded" @toggle="tasksExpanded = !tasksExpanded" />
        </div>
        <!-- 输入区 -->
        <div class="chat-input-area" ref="chatInputAreaRef">
          <ApprovalBar v-if="approvalState.waiting" :waiting="approvalState.waiting" :tool="approvalState.tool" :args="approvalState.args" :parsedArgs="approvalState.parsedArgs" @resolve="resolveApproval" />
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
              <span :class="['obtn', { active: autoCollapse }]" @click="toggleAuto('autoCollapse')" title="自动折叠：新消息发出时折叠旧输出，显示完成摘要"><SvgIcon name="list" :size="12" /> 折叠</span>
              <span :class="['obtn', { active: autoCommit }]" @click="toggleAuto('autoCommit')" title="自动 Git 提交：任务完成时自动 git add + commit"><SvgIcon name="git-commit" :size="12" /> 提交</span>
              <span class="obtn-sep"></span>
              <span :class="['obtn', 'obtn-agent', { active: autonomous }]" @click="toggleAuto('autonomous')" title="自主模式：开启=连续执行全部计划步骤，关闭=单次回复"><SvgIcon name="sparkles" :size="12" color="#d4a74e" /> 自主</span>
            </div>
            <button v-if="!state.chatLoading" class="send-btn" @click="sendMessage" :disabled="!inputText.trim()"><SvgIcon name="send-plane" :size="16" /></button>
            <button v-else class="stop-btn" @click="stopChat"><SvgIcon name="stop-dot" :size="20" /></button>
          </div>
        </div>
      </div>
      <!-- 右侧：Debug日志面板 / 会话列表 -->
      <DebugLogPanel v-if="showDebugLog" @close="showDebugLog = false" />
      <ConvSidebar v-else :conversations="state.conversations" :current-conv-id="state.currentConvId" :loading-by-conv="state.loadingByConv" :ws-token-stats="wsTokenStats" :conv-ctx-stats="convCtxStats" :ctx-max-tokens-val="state.settings.contextMaxTokens || 1000000" :width="convListWidth" @new-conversation="newConversation" @switch-conversation="switchConv" @delete-conversation="deleteConv" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, inject, onMounted, onUnmounted, nextTick, watch, reactive } from 'vue'
import { state } from '../main.js'
import api from '../api.js'
import { setGlobalCtx, startConvRuntime, resetConvRuntime, createAssistantPlaceholder, getConvCtxStats, resetConvCtxStats } from '../agent-events.js'
import SvgIcon from './SvgIcon.vue'
import PlanPanel from './PlanPanel.vue'
import TaskPanel from './TaskPanel.vue'
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
const chatInputAreaRef = ref(null)
let inputOverlayObserver = null

// 根据 .input-overlay 的实际高度动态调整 .chat-input 的 padding-bottom
function updateInputPadding() {
  if (!chatInputAreaRef.value) return
  const overlay = chatInputAreaRef.value.querySelector('.input-overlay')
  const textarea = chatInputAreaRef.value.querySelector('.chat-input')
  if (!overlay || !textarea) return
  // overlay 高度 + 底部间距(16px) + 安全余量(12px)
  const overlayHeight = overlay.offsetHeight
  const paddingBottom = Math.max(68, overlayHeight + 28)
  textarea.style.paddingBottom = paddingBottom + 'px'
}
const inputHeight = ref(150)
const convListWidth = ref(250)
const topSentinel = ref(null)
const autoReview = ref(true)
const autoIterate = ref(false)
const autoCollapse = ref(true)
const autonomous = ref(false)
const autoCommit = ref(true)
const pendingAttachment = ref(null)

// nudge 提示条（从全局 state.nudgeByConv 读取，仅当前对话）
const currentNudge = computed(() => state.nudgeByConv[state.currentConvId] || '')
let nudgeTimer = null
function showNudge(text) {
  // nudge 写入全局 state，由 currentNudge computed 响应
  state.nudgeByConv[state.currentConvId] = text
  if (nudgeTimer) clearTimeout(nudgeTimer)
  nudgeTimer = setTimeout(() => { state.nudgeByConv[state.currentConvId] = '' }, 4000)
}

let pendingAskCallId = ''
const currentPlan = ref([])
const currentTasks = ref([])
const planExpanded = ref(true)
const tasksExpanded = ref(true)
// 阶段指示器从全局 state.phaseByConv 读取（仅当前对话）
const currentPhase = computed(() => state.phaseByConv[state.currentConvId] || '')
let phaseTimer = null
let autoSaveTimer = null

// ── 滚动控制 ──
const scrollTopRef = ref(0)
const isNearBottom = ref(true)
const showScrollDown = ref(false)

// ── 审批状态从全局 state.approvalByConv 读取（仅当前对话）──
const approvalState = computed(() => state.approvalByConv[state.currentConvId] || { callId: '', tool: '', args: '', parsedArgs: {}, waiting: false })
const hasMoreTop = computed(() => {
  const id = state.currentConvId
  const msgs = state.messagesByConv[id]
  if (!msgs || msgs.length === 0) return false
  if (msgs[0]._noMoreAbove) return false
  // 依据最早已加载消息的 _idx 判断是否还有更早消息（比 msgTotal/Loaded 更可靠）
  const oldestIdx = msgs[0]._idx
  return oldestIdx !== undefined && oldestIdx !== null && oldestIdx > 0
})

// loadingMoreTop 防止懒加载重复触发
const loadingMoreTop = ref(false)

function onScroll() {
  if (msgRef.value) {
    const el = msgRef.value
    scrollTopRef.value = el.scrollTop
    const threshold = 100
    const wasNearBottom = isNearBottom.value
    isNearBottom.value = el.scrollTop + el.clientHeight >= el.scrollHeight - threshold
    // 用户主动上翻→锁定自动滚底（直到手动滚回底部或点击跳底按钮才解锁）
    if (wasNearBottom && !isNearBottom.value) {
      window.__scrollLockTimer = true
    }
    // 用户手动滚回底部→立即解锁
    if (!wasNearBottom && isNearBottom.value) {
      window.__scrollLockTimer = false
    }
    // 显示跳到底部按钮
    // 用户离开底部（滚动向上 > threshold）时隐藏跳底按钮
    showScrollDown.value = !isNearBottom.value && state.messages && state.messages.length > 0
    // 顶部懒加载：scrollTop < 100 且还有更早消息可加载
    if (el.scrollTop < 100 && !loadingMoreTop.value) {
      loadMoreMessages()
    }
  }
}

// loadMoreMessages 向前分页加载更早消息，prepend 到数组并维护滚动位置
const loadMoreMessages = async () => {
  const id = state.currentConvId
  if (!id) return
  const msgs = state.messagesByConv[id]
  if (!msgs || msgs.length === 0) return
  if (msgs[0]._noMoreAbove) return
  const oldestIdx = msgs[0]._idx
  if (oldestIdx === undefined || oldestIdx === null || oldestIdx <= 0) return
  loadingMoreTop.value = true
  // 记录 prepend 前的 scrollHeight + scrollTop，用于补偿滚动位置
  const oldScrollHeight = msgRef.value ? msgRef.value.scrollHeight : 0
  const oldScrollTop = msgRef.value ? msgRef.value.scrollTop : 0
  try {
    const data = await api.getMessages(id, { before: oldestIdx, limit: 50 })
    const older = (data.messages || [])
      .filter(m => (m.message?.role || m.role) !== 'tool')
      .map((m, i) => ({
        role: m.message?.role || m.role || '',
        content: m.message?.content || m.content || '',
        segments: (m.segments || []).map(seg => {
          if (seg.type === 'ask_user') seg._answered = !!seg.answer
          return seg
        }),
        _key: 'msg_' + Date.now() + '_older_' + i,
        _idx: m.idx,
        _time: m.timestamp || '',
      }))
    if (older.length > 0) {
      // prepend 到数组头部（全量渲染下 scrollHeight 会自然增加）
      state.messagesByConv[id] = [...older, ...msgs]
      state.messages = state.messagesByConv[id]
      state.msgLoadedByConv[id] = (state.msgLoadedByConv[id] || 0) + older.length
      // 补偿滚动位置：保持当前视口内容不动（新增的 older 消息在顶部）
      nextTick(() => {
        if (msgRef.value) {
          msgRef.value.scrollTop = oldScrollTop + (msgRef.value.scrollHeight - oldScrollHeight)
        }
      })
    } else {
      // 无更早消息：标记防止重复请求
      msgs[0]._noMoreAbove = true
    }
  } catch (e) {
    console.warn('loadMoreMessages 失败:', e)
  } finally {
    loadingMoreTop.value = false
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
  if (/^multi_edit\b/.test(name)) return { icon: 'edit', title: '多处编辑', detail: args.path || '', summary: '已编辑', resultIcon: 'check' }
  if (/^run_command\b/.test(name)) return { icon: 'terminal', title: '执行命令', detail: '$ ' + (args.command || '').slice(0, 60), summary: '已完成', resultIcon: 'check' }
  if (/^run_test\b/.test(name)) return { icon: 'check', title: '运行测试', detail: args.package_path || '', summary: '已完成', resultIcon: 'check' }
  if (/^search_content\b/.test(name)) return { icon: 'search', title: '搜索内容', detail: '/' + (args.pattern || '') + '/', summary: '已搜索', resultIcon: 'check' }
  if (/^search_files\b/.test(name)) return { icon: 'search', title: '搜索文件', detail: (args.pattern || ''), summary: '已搜索', resultIcon: 'check' }
  if (/^web_search\b/.test(name)) return { icon: 'globe', title: '网络搜索', detail: (args.query || '').slice(0, 60), summary: '已搜索', resultIcon: 'globe' }
  if (/^web_fetch\b/.test(name)) return { icon: 'globe', title: '抓取网页', detail: (args.url || '').slice(0, 60), summary: '已抓取', resultIcon: 'globe' }
  if (/^git_status\b/.test(name)) return { icon: 'source-control', title: 'Git 状态', detail: '', summary: '已查看', resultIcon: 'check' }
  if (/^git_diff\b/.test(name)) return { icon: 'source-control', title: 'Git 差异', detail: args.file ? args.file.slice(0, 40) : '', summary: '已查看', resultIcon: 'check' }
  if (/^git_log\b/.test(name)) return { icon: 'source-control', title: 'Git 日志', detail: '', summary: '已查看', resultIcon: 'check' }
  if (/^find_symbol\b/.test(name)) return { icon: 'search', title: '查找符号', detail: args.symbol || args.symbol || '', summary: '已查找', resultIcon: 'check' }
  if (/^screenshot_desktop\b/.test(name)) return { icon: 'image', title: '桌面截图', detail: '', summary: '已截图', resultIcon: 'check' }
  if (/^screenshot_window\b/.test(name)) return { icon: 'image', title: '窗口截图', detail: (args.title || '').slice(0, 40), summary: '已截图', resultIcon: 'check' }
  if (/^web_debug\b/.test(name)) return { icon: 'globe', title: '网页调试', detail: args.url ? args.url.slice(0, 50) : '', summary: '已验证', resultIcon: 'check' }
  if (/^go_build\b/.test(name)) return { icon: 'terminal', title: 'Go 构建', detail: args.path || '.', summary: '已完成', resultIcon: 'check' }
  if (/^go_run\b/.test(name)) return { icon: 'terminal', title: 'Go 运行', detail: args.path || '.', summary: '已完成', resultIcon: 'check' }
  if (/^bug_detect\b/.test(name)) return { icon: 'bug', title: 'BUG 检测', detail: '', summary: '已完成', resultIcon: 'check' }
  if (/^bug_fix\b/.test(name)) return { icon: 'bug', title: 'BUG 修复', detail: '', summary: '已完成', resultIcon: 'check' }
  if (/^ask_user\b/.test(name)) return { icon: 'message-square', title: '询问用户', detail: '', summary: '', resultIcon: 'check' }
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

// ── 工作区 Token 统计（使用全局 state，与 agent-events.js 共享）──
const wsTokenStats = computed(() => state.wsTokenStatsByWs[state.workspaceRoot] || { totalTokens: 0, promptTokens: 0, completionTokens: 0, cacheHitTokens: 0, cacheMissTokens: 0, systemTokens: 0, skillsTokens: 0, mcpTokens: 0, toolTokens: 0, historyTokens: 0, otherTokens: 0 })
const convCtxStats = computed(() => getConvCtxStats(state.currentConvId))

const loadWsTokenStats = async () => {
  try {
    const data = await api.apiGet('/tokens/stats', { workspaceRoot: state.workspaceRoot })
    if (data) {
      if (!state.wsTokenStatsByWs[state.workspaceRoot]) {
        state.wsTokenStatsByWs[state.workspaceRoot] = { totalTokens: 0, promptTokens: 0, completionTokens: 0, cacheHitTokens: 0, cacheMissTokens: 0, systemTokens: 0, skillsTokens: 0, mcpTokens: 0, toolTokens: 0, historyTokens: 0, otherTokens: 0 }
      }
      Object.assign(state.wsTokenStatsByWs[state.workspaceRoot], data)
    }
    if (state.currentConvId) {
      const ts = await api.apiGet('/conversations/' + state.currentConvId + '/token-stats')
      if (ts && ts.promptTokens !== undefined) Object.assign(getConvCtxStats(state.currentConvId), ts)
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

// ── 消息展示辅助函数 ──

// isDelegation 判断用户消息是否来自外层 agent 委派任务。
function isDelegation(msg) {
  return msg.role === 'user' && typeof msg.content === 'string' && msg.content.startsWith('【任务委派 →')
}

// delegationAgent 从委派消息内容中提取目标 agent 名。
function delegationAgent(msg) {
  if (!msg.content) return ''
  const m = msg.content.match(/^【任务委派 → (\w+)】/)
  return m ? m[1] : ''
}

// isFeedback 判断用户消息是否为用户反馈。
function isFeedback(msg) {
  return msg.role === 'user' && typeof msg.content === 'string' && msg.content.startsWith('【用户反馈】')
}

// cleanMsgContent 去除消息中的标记前缀，只展示纯内容。
function cleanMsgContent(msg) {
  if (!msg.content) return ''
  return msg.content
    .replace(/^【任务委派 → \w+】\n*/, '')
    .replace(/^【用户反馈】/, '')
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
      state.conversations.unshift({ id: conv.id, title: conv.title, msgCount: 0, createdAt: conv.createdAt, updatedAt: conv.updatedAt })
      resetConvCtxStats(conv.id)
    } catch {}
  }
  const userContent = text || ''
  let fullContent = userContent
  if (pendingAttachment.value) {
    const att = pendingAttachment.value
    if (att.type === 'image') {
      // 图片保留 dataURL（无法用 read_file）
      fullContent += '\n\n---\n[图片附件] ' + (att.filename || '') + '\n' + (att.content || '').slice(0, 2000)
    } else if (att.type === 'file') {
      fullContent += '\n\n[参考文件] ' + att.path + '\n（如需查看文件内容，请使用 read_file 工具读取上述路径）'
    } else if (att.type === 'code') {
      fullContent += '\n\n[参考文件] ' + att.path + ':' + (att.lineStart || 1) + '-' + (att.lineEnd || 1) + '\n（如需查看代码，请使用 read_file 工具读取上述路径和行号）'
    }
  }
  const lastUserText = text
  inputText.value = ''; pendingAttachment.value = null
  // 多会话并行：不停止旧 agent 的订阅（旧对话后台继续运行）
  collapsePreviousOutputs()
  // 确保 messagesByConv[convId] 存在
  const convId = state.currentConvId
  if (!state.messagesByConv[convId]) state.messagesByConv[convId] = []
  const userMsg = { role: 'user', content: fullContent, segments: [], toolCalls: [], _key: makeMsgKey(), _idx: state.messagesByConv[convId].length, _time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) }
  state.messagesByConv[convId].push(userMsg)
  // 同步到 state.messages（当前对话快捷引用）
  state.messages = state.messagesByConv[convId]
  if (convId) {
    // 用户消息已由后端 handleChatSend → AppendPersistedUserMessage 持久化，前端不再重复 POST
    // 立即用用户消息更新对话标题（不等 onDone，避免 SSE 中断导致标题不更新）
    autoNameConv(convId, lastUserText || fullContent)
    // 本地递增消息计数
    const localConv = state.conversations.find(c => c.id === convId)
    if (localConv) localConv.msgCount = (localConv.msgCount || 0) + 1
  }
  // 标记 loading（按 convId 存储 + 当前对话快捷引用）
  state.loadingByConv[convId] = true
  state.agentRunningByConv[convId] = true
  state.chatLoading = true; state.agentRunning = true
  // 本地递增工作区运行计数（立即在工作区列表显示脉冲点）
  // 后端 done/error 时会通过 status 消息同步纠正
  if (state.workspaceRoot) {
    state.runningByWorkspace = {
      ...state.runningByWorkspace,
      [state.workspaceRoot]: (state.runningByWorkspace[state.workspaceRoot] || 0) + 1,
    }
  }
  if (!state.chatSessionId) state.chatSessionId = 'sess_' + Date.now()
  const msgIdx = createAssistantPlaceholder(convId)
  startConvRuntime(convId, msgIdx, lastUserText || fullContent)
  try {
    await api.chatStart(convId, fullContent, autonomous.value, state.workspaceRoot)
  } catch (err) {
    const msgs0 = state.messagesByConv[convId]
    if (msgs0) {
      const m = msgs0[msgIdx]
      if (m) { m._loading = false; pushSegment(m.segments, 'content').content += '**[启动失败]** ' + (err.message || err) }
    }
    state.loadingByConv[convId] = false
    state.agentRunningByConv[convId] = false
    state.chatLoading = false; state.agentRunning = false
    // 启动失败：递减工作区运行计数
    if (state.workspaceRoot && state.runningByWorkspace[state.workspaceRoot] > 0) {
      state.runningByWorkspace = {
        ...state.runningByWorkspace,
        [state.workspaceRoot]: Math.max(0, (state.runningByWorkspace[state.workspaceRoot] || 0) - 1),
      }
    }
    resetConvRuntime(convId)
    return
  }
  // 事件流由 App.vue 全局 WebSocket 接收 → agent-events.js processAgentEvent/Done 处理
  // 切换对话/工作区时 agent 后台继续运行，事件继续写入 messagesByConv[convId]
}

const stopChat = async () => {
  const convId = state.currentConvId
  if (!convId) return
  try { await api.chatStop(convId) } catch {}
  resetConvRuntime(convId)
  state.loadingByConv[convId] = false
  state.agentRunningByConv[convId] = false
  state.chatLoading = false; state.agentRunning = false
}

// ── 运行时反馈：Agent 执行中用户补充/纠正 ──
const sendFeedback = async () => {
  const text = feedbackText.value.trim()
  if (!text || !state.currentConvId) return
  feedbackText.value = ''
  try {
    await api.apiPost('/chat/feedback', { convId: state.currentConvId, feedback: text })
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
  try { await api.apiPost('/chat/answer', { convId: state.currentConvId, answer }) } catch {}
}

const resolveApproval = async (approved) => {
  const convId = state.currentConvId
  const a = state.approvalByConv[convId]
  if (!a || !a.callId || !a.waiting) return
  a.waiting = false
  try { await api.apiPost('/chat/approve', { convId, approved }) } catch { a.waiting = true }
}

const onKeydown = (e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMessage() } }

const scrollToBottom = () => {
  showScrollDown.value = false
  window.__scrollLockTimer = false
  nextTick(() => { if (msgRef.value) { msgRef.value.scrollTop = msgRef.value.scrollHeight; isNearBottom.value = true; } })
}

const forceScrollToBottom = () => {
  showScrollDown.value = false
  window.__scrollLockTimer = false
  nextTick(() => { if (msgRef.value) { msgRef.value.scrollTop = msgRef.value.scrollHeight; isNearBottom.value = true; } })
}

const loadConvList = async () => {
  try {
    const list = await api.apiGet('/conversations', { workspace: state.workspaceRoot })
    state.conversations = list || []
    // 无对话时自动创建一个"新对话"
    if (state.conversations.length === 0 && state.workspaceRoot) {
      await newConversation()
    }
  } catch {}
}

const newConversation = async () => {
  // 多会话并行：不停止旧 agent，立即在后端创建新对话并切换
  try {
    const conv = await api.apiPost('/conversations', { title: '新对话', workspaceRoot: state.workspaceRoot })
    state.currentConvId = conv.id
    state.conversations.unshift({ id: conv.id, title: conv.title || '新对话', msgCount: 0, createdAt: conv.createdAt, updatedAt: conv.updatedAt })
    if (!state.messagesByConv[conv.id]) state.messagesByConv[conv.id] = []
    state.messages = state.messagesByConv[conv.id]
    state.msgTotalByConv[conv.id] = 0
    state.msgLoadedByConv[conv.id] = 0
    resetConvCtxStats(conv.id)
  } catch {
    // 后端创建失败时兜底：用临时空对话
    state.currentConvId = ''
    state.messagesByConv[''] = state.messagesByConv[''] || []
    state.messages = state.messagesByConv['']
  }
  state.chatLoading = false
  state.agentRunning = false
  currentPlan.value = []
  currentTasks.value = []
  inputText.value = ''
  nextTick(() => inputRef.value?.focus())
}

const deleteConv = async (id) => {
  // 若该对话有运行中的 agent，停止它
  if (state.agentRunningByConv[id]) {
    try { await api.chatStop(id) } catch {}
    resetConvRuntime(id)
  }
  try {
    await api.apiDelete('/conversations/' + id)
    state.conversations = state.conversations.filter(c => c.id !== id)
    // 删除后立即同步到 localStorage，防止页面刷新后旧数据复现
    window.dispatchEvent(new Event('save-conversations'))
    delete state.messagesByConv[id]
    delete state.loadingByConv[id]
    delete state.agentRunningByConv[id]
    delete state.approvalByConv[id]
    delete state.phaseByConv[id]
    delete state.nudgeByConv[id]
    delete state.convCtxStatsByConv[id]
    delete state.msgTotalByConv[id]
    delete state.msgLoadedByConv[id]
    if (state.currentConvId === id) {
      state.currentConvId = ''
      state.messages = []
      state.chatLoading = false
      state.agentRunning = false
    }
  } catch {}
}

const switchConv = async (id) => {
  // 多会话并行：切换对话不停止旧 agent，事件继续写入 messagesByConv[oldConvId]
  state.currentConvId = id
  // 切换 state.messages 指向（不停止旧 agent）
  if (!state.messagesByConv[id]) state.messagesByConv[id] = []
  state.messages = state.messagesByConv[id]
  // 同步 loading 状态
  state.chatLoading = state.loadingByConv[id] || false
  state.agentRunning = state.agentRunningByConv[id] || false
  currentPlan.value = []
  currentTasks.value = []
  // approval/phase/nudge/convCtxStats 自动从 state.*ByConv[currentConvId] 读取，无需手动重置
  // 始终从后端刷新 token 统计（无论本地是否缓存了消息）
  try {
    const ts = await api.apiGet('/conversations/' + id + '/token-stats')
    if (ts && ts.promptTokens !== undefined) Object.assign(getConvCtxStats(id), ts)
  } catch {}
  // 懒加载：若本地尚无消息缓存，拉取最新 50 条
  if (state.messagesByConv[id].length === 0) {
    try {
      const data = await api.getMessages(id, { limit: 50 })
      const loaded = (data.messages || [])
        .filter(m => (m.message?.role || m.role) !== 'tool')
        .map((m, i) => {
          const role = m.message?.role || m.role || ''
          const segments = (m.segments || []).map(seg => {
            // 兼容旧数据：finish_task 工具调用转为 content 段渲染
            if (seg.type === 'tool_call' && seg.name === 'finish_task') {
              return { type: 'content', content: seg.result || '' }
            }
            // ask_user 持久化数据初始化 _answered 状态
            if (seg.type === 'ask_user') {
              seg._answered = !!seg.answer
            }
            return seg
          })
          return {
            role,
            content: m.message?.content || m.content || '',
            segments,
            _key: 'msg_' + Date.now() + '_' + i,
            _idx: m.idx,
            _time: m.timestamp || '',
          }
        })
        // 按 idx 升序排列（用户消息在前，agent 输出在后）
        loaded.sort((a, b) => (a._idx || 0) - (b._idx || 0))
      state.messagesByConv[id] = loaded
      state.messages = state.messagesByConv[id]
      state.msgTotalByConv[id] = data.total || loaded.length
      state.msgLoadedByConv[id] = loaded.length
    } catch {
      state.msgTotalByConv[id] = 0
      state.msgLoadedByConv[id] = 0
    }
  }
  // ── 从后端 API 加载真实任务状态（TaskManager 持久化到 .pair/tasks/*.json）──
  // 放在消息加载之外，确保每次切换/刷新都从真实数据加载而非从消息重建
  // 传入当前对话 ID 过滤，只显示本对话创建的任务
  try {
    const taskData = await api.apiGet('/tasks', { convId: id })
    if (taskData && taskData.tasks && taskData.tasks.length > 0) {
      currentTasks.value = taskData.tasks.map(t => ({
        step: t.step || t.subject || '',
        status: t.status,
        _taskId: t.taskId,
      }))
    } else {
      currentTasks.value = []
    }
  } catch {
    currentTasks.value = []
  }
  // ── 从消息重建 plan（update_plan 工具调用）──
  const msgs = state.messagesByConv[id] || []
  if (msgs.length > 0) {
    currentPlan.value = rebuildPlanFromMessages(msgs)
  } else {
    currentPlan.value = []
  }
  planExpanded.value = currentPlan.value.length > 0
  forceScrollToBottom()
}

const toggleAuto = async (field) => {
  const oldVal = !!state.settings[field]
  const newVal = !oldVal
  state.settings[field] = newVal
  // 同步 local ref（浅 watch 不触发，需要手动同步）
  if (field === 'autoReview') autoReview.value = newVal
  else if (field === 'autonomous') autonomous.value = newVal
  else if (field === 'autoCommit') autoCommit.value = newVal
  else if (field === 'autoCollapse') autoCollapse.value = newVal
  try { await api.apiPut('/settings?convId=' + encodeURIComponent(state.currentConvId), state.settings) } catch { state.settings[field] = oldVal; if (field === 'autoReview') autoReview.value = oldVal; else if (field === 'autonomous') autonomous.value = oldVal; else if (field === 'autoCommit') autoCommit.value = oldVal; else if (field === 'autoCollapse') autoCollapse.value = oldVal }
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

function rebuildPlanFromMessages(msgs) {
    // 从已加载消息的 segments 中扫描 update_plan 工具调用，重建执行步骤。
  let plan = []
  for (const msg of msgs) {
    if (!msg.segments) continue
    for (const seg of msg.segments) {
      if (seg.type !== 'tool_call') continue
      if (seg.name !== 'update_plan') continue
      let args
      try { args = seg.argsRaw ? JSON.parse(seg.argsRaw) : {} } catch { continue }
      if (Array.isArray(args.plan)) {
        plan = args.plan.map(s => ({ ...s }))
      }
    }
  }
  return plan
}

function handleTaskTool(data) {
  const toolName = data.tool || data.name || ''
  const taskTools = ['update_plan', 'task_create', 'task_update', 'task_list', 'task_delete', 'task_summary']
  if (!taskTools.includes(toolName)) return false
  try {
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
  // 优先检查工作区文件路径（文件树拖拽携带的路径）
  const wsPath = e.dataTransfer?.getData('application/x-file-path') || e.dataTransfer?.getData('text/x-file-path') || ''
  if (wsPath) {
    pendingAttachment.value = { type: 'file', path: wsPath, filename: wsPath.split(/[\\/]/).pop() }
    return
  }
  // 外部文件（浏览器文件系统）—— 不在工作区内，提示用户
  const files = e.dataTransfer?.files
  if (files && files.length > 0) {
    // 外部文件无法获得工作区相对路径，agent 无法 read_file，提示用户
    window.$toast && window.$toast('该文件不在工作区内，请先添加到工作区或从文件树拖入', 'warn')
    return
  }
  // 纯文本拖拽 —— 保留原逻辑
  const textData = e.dataTransfer?.getData('text/plain')
  if (textData) { inputText.value += textData; inputRef.value?.focus() }
}

const handlePaste = (e) => {
  const items = e.clipboardData?.items; if (!items) return
  for (const item of items) {
    if (item.kind === 'file') {
      e.preventDefault(); const file = item.getAsFile(); if (!file) continue
      if (file.type.startsWith('image/')) {
        // 图片保留 dataURL（图片无法用 read_file 读取）
        if (file.size > 1024 * 1024) {
          window.$toast && window.$toast('图片超过 1MB，请压缩后粘贴', 'warn')
          return
        }
        const reader = new FileReader()
        reader.onload = (ev) => { pendingAttachment.value = { type: 'image', path: file.name, filename: file.name, content: ev.target?.result || '' } }
        reader.readAsDataURL(file)
      } else {
        // 非图片文件 —— 不读取内容，提示从编辑器或文件树拖入
        window.$toast && window.$toast('粘贴文件不支持，请从文件树拖入或从编辑器选中代码后拖入', 'warn')
      }
      return
    }
  }
}

// ── 流式内容尺寸观察器：捕捉 segment 内容增长导致的容器尺寸变化
let contentResizeObserver = null
function startContentResizeObserver() {
  stopContentResizeObserver()
  if (!msgRef.value) return
  const wrap = msgRef.value.querySelector('.msg-list-wrap')
  if (!wrap) return
  contentResizeObserver = new ResizeObserver(() => {
    // 已移除自动滚动 — 由 agent-events.js 的 scrollToBottom 统一控制
  })
  contentResizeObserver.observe(wrap)
}
function stopContentResizeObserver() {
  if (contentResizeObserver) {
    contentResizeObserver.disconnect()
    contentResizeObserver = null
  }
}

// ── 新消息自动滚底（已移除 — 由 agent-events.js 的 scrollToBottom 统一控制）

// ── 对话切换时重启内容尺寸观察器（DOM 重建）
watch(() => state.currentConvId, () => {
  nextTick(() => startContentResizeObserver())
})

// ── 对话消息全量替换（如首次加载/切换）时也重启观察器
watch(() => state.messages, () => {
  nextTick(() => startContentResizeObserver())
}, { deep: false })

watch(() => state.settings, (s) => { if (s) { autoReview.value = s.autoReview !== undefined ? !!s.autoReview : true; autoIterate.value = !!s.autoIterateOnRejection; autonomous.value = !!s.autonomous; autoCollapse.value = s.autoCollapse !== undefined ? !!s.autoCollapse : true; autoCommit.value = s.autoCommit !== false; } }, { immediate: true })

// ── 工作区切换时加载 Token 统计（onMounted 时 workspaceRoot 可能还未设）
watch(() => state.workspaceRoot, (root) => {
  if (root && root !== '') {
    // 清理 onMounted 阶段可能存到空 key 的脏数据
    if (state.wsTokenStatsByWs['']) {
      delete state.wsTokenStatsByWs['']
    }
    loadWsTokenStats()
  }
})

const handleBeforeUnload = () => { if (state.currentConvId && state.messages.length > 0) { window.dispatchEvent(new Event('save-conversations')) } }

onMounted(() => {
  loadWsTokenStats(); loadConvList(); scrollToBottom()

  // 监听消息内容尺寸变化（流式输出时自动跟随滚底）
  nextTick(() => startContentResizeObserver())

  // 监听 .input-overlay 尺寸变化，动态调整 textarea padding-bottom
  nextTick(() => {
    updateInputPadding()
    if (chatInputAreaRef.value) {
      const overlay = chatInputAreaRef.value.querySelector('.input-overlay')
      if (overlay) {
        inputOverlayObserver = new ResizeObserver(() => updateInputPadding())
        inputOverlayObserver.observe(overlay)
      }
    }
  })

  // ⚡ 初始加载：若已有当前对话，从 API 加载任务状态
  // （页面刷新或从其他工作区切换回来时，currentTasks 为空，需要从 TaskManager 恢复）
  nextTick(async () => {
    if (state.currentConvId) {
      try {
        const taskData = await api.apiGet('/tasks', { convId: state.currentConvId })
        if (taskData && taskData.tasks && taskData.tasks.length > 0) {
          currentTasks.value = taskData.tasks.map(t => ({
            step: t.step || t.subject || '',
            status: t.status,
            _taskId: t.taskId,
          }))
        }
      } catch {}
    }
  })

  // 注册全局 UI 回调：App.vue 的 WebSocket onmessage → agent-events.js → 此处回调
  setGlobalCtx({
    scrollToBottom: () => {
      // ★ 用户上翻时锁定自动滚动，直到手动滚回底部或点击跳底按钮
      if (window.__scrollLockTimer) return
      scrollToBottom()
    },
    loadWsTokenStats: () => loadWsTokenStats(),
    autoNameConv: (convId, text) => autoNameConv(convId, text),
    saveConvMsg: (convId, content, msgIdx) => {
      // 后端 startEventPersistWorker 已通过 SegmentsFromMessage 自动持久化
      // loop.History 中的消息（含 ToolCalls→tool_call, Reasoning→thinking, Content→content）。
      // 前端不再重复 POST，避免消息重复追加。
    },
    onPlanUpdate: (plan, convId) => {
      if (state.currentConvId !== convId) return
      // update_plan 全量替换执行步骤清单（外层 AutonomousController 使用），只展示在 PlanPanel
      currentPlan.value = [...plan]
      planExpanded.value = true
    },
    onTaskCreate: (task, convId) => {
      if (state.currentConvId !== convId) return
      currentTasks.value = [...currentTasks.value, task]
    },
    onTaskSetId: (callId, taskId, convId) => {
      if (state.currentConvId !== convId) return
      const tasks = [...currentTasks.value]
      for (let i = 0; i < tasks.length; i++) {
        if (tasks[i].callId === callId) { tasks[i] = { ...tasks[i], _taskId: taskId }; break }
      }
      currentTasks.value = tasks
    },
    onTaskUpdate: (taskId, status, subject, convId) => {
      if (state.currentConvId !== convId) return
      const tasks = [...currentTasks.value]
      let changed = false
      for (let i = 0; i < tasks.length; i++) {
        if (tasks[i]._taskId && tasks[i]._taskId === taskId) { tasks[i] = { ...tasks[i], status }; changed = true; break }
        if (subject && tasks[i].step === subject) { tasks[i] = { ...tasks[i], status, _taskId: tasks[i]._taskId || taskId }; changed = true; break }
      }
      if (changed) currentTasks.value = tasks
    },
    onTaskReplace: (tasks, convId) => {
      // update_tasks 全量替换：直接覆盖 currentTasks
      if (state.currentConvId !== convId) return
      currentTasks.value = [...tasks]
      tasksExpanded.value = true
    },
    onPhaseChange: (convId) => {
      // 阶段指示器自动从 state.phaseByConv 读取，此处启动定时器自动清除
      if (phaseTimer) clearTimeout(phaseTimer)
      phaseTimer = setTimeout(() => { state.phaseByConv[convId] = '' }, 6000)
    },
    onPhaseEnd: (convId) => {
      if (phaseTimer) { clearTimeout(phaseTimer); phaseTimer = null }
      state.phaseByConv[convId] = ''
    },
    onNudge: (convId) => {
      // nudge 自动从 state.nudgeByConv 读取，此处启动定时器清除
      if (nudgeTimer) clearTimeout(nudgeTimer)
      nudgeTimer = setTimeout(() => { state.nudgeByConv[convId] = '' }, 4000)
    },
  })

  window.addEventListener('add-to-chat', (e) => {
    const detail = e.detail; if (!detail) return
    pendingAttachment.value = { type: detail.type || 'file', path: detail.path || '', filename: detail.filename || '', lineStart: detail.lineStart || null, lineEnd: detail.lineEnd || null, content: detail.content || '' }
  })
  window.addEventListener('workspace-switched', async () => {
    // 工作区切换：不清空 messagesByConv/loadingByConv/agentRunningByConv（agent 后台继续运行）
    // 仅重新加载当前工作区的对话列表；loadConvList 内部会在无对话时自动创建
    state.chatLoading = false
    state.agentRunning = false
    state.chatSessionId = ''
    inputText.value = ''
    currentPlan.value = []
    await loadConvList()
    // loadConvList 已处理 currentConvId 和 messages 的设置（自动创建或保持空）
    if (!state.currentConvId) {
      state.messages = []
    }
  })
  autoSaveTimer = setInterval(() => { if (state.currentConvId && state.messages.length > 0) { window.dispatchEvent(new Event('save-conversations')) } }, 15000)
  window.addEventListener('beforeunload', handleBeforeUnload)
})

onUnmounted(() => {
  if (autoSaveTimer) { clearInterval(autoSaveTimer); autoSaveTimer = null }
  if (phaseTimer) { clearTimeout(phaseTimer); phaseTimer = null }
  if (nudgeTimer) { clearTimeout(nudgeTimer); nudgeTimer = null }
  stopContentResizeObserver()
  if (inputOverlayObserver) { inputOverlayObserver.disconnect(); inputOverlayObserver = null }
  // 不关闭 WebSocket（由 App.vue 管理生命周期）；不清理 subscriptions（已移除 SSE 订阅模式）
  document.removeEventListener('mousemove', onInputResizeMove); document.removeEventListener('mouseup', stopInputResize)
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
.chat-messages { flex: 1; overflow-y: auto; padding: 8px 12px; min-height: 0; position: relative; overflow-anchor: none; }
.msg-list-wrap { display: flex; flex-direction: column; gap: 12px; min-height: 100%; }
.msg-item { display: flex; gap: 8px; align-items: flex-start; content-visibility: auto; contain-intrinsic-size: 60px; }
.msg-user { flex-direction: row-reverse; justify-content: flex-start; gap: 10px; }




.msg-user .bubble-user {
  flex: 0 0 auto;
  max-width: 80%;
  min-width: 40px;
  background: var(--accent);
  color: #fff;
  padding: 10px 16px;
  border-radius: 16px 16px 4px 16px;
  overflow-wrap: break-word;
  word-break: break-word;
  overflow-wrap: anywhere;
  margin: 2px 0;
  box-shadow: 0 1px 3px rgba(0,0,0,0.18);
  transition: box-shadow 0.15s ease, transform 0.1s ease;
}
.msg-user .bubble-user:hover {
  box-shadow: 0 2px 6px rgba(0,0,0,0.25);
}
/* 选中文字在深色气泡上可见 */
.msg-user .bubble-user ::selection {
  background: rgba(255, 255, 255, 0.3);
  color: #fff;
}
.user-msg-content { width: 100%; text-align: left; }
.user-msg-content :deep(p) { margin: 4px 0; white-space: pre-wrap; word-break: break-word; }
.user-msg-content :deep(p:first-child) { margin-top: 0; }
.user-msg-content :deep(p:last-child) { margin-bottom: 0; }
.user-msg-content :deep(pre) { white-space: pre-wrap; font-size: 12px; background: rgba(0,0,0,0.15); padding: 6px 8px; border-radius: 4px; max-width: 100%; overflow-x: auto; margin: 4px 0; }
.user-msg-content :deep(code) { font-size: 12px; }
.user-msg-header { margin-bottom: 6px; display: flex; align-items: center; gap: 6px; }
.umh-badge { display: inline-flex; align-items: center; gap: 3px; font-size: 10px; padding: 1px 6px; border-radius: 3px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; }
.badge-delegation { background: rgba(99, 102, 241, 0.2); color: #818cf8; border: 1px solid rgba(99, 102, 241, 0.3); }
.badge-feedback { background: rgba(251, 191, 36, 0.15); color: #fbbf24; border: 1px solid rgba(251, 191, 36, 0.3); }
.umh-agent { font-size: 11px; color: var(--text-muted); background: var(--bg-tertiary); padding: 1px 6px; border-radius: 3px; }
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
.folded-summary { display: flex; align-items: center; gap: 5px; padding: 5px 10px; background: var(--bg-primary); border: 1px solid var(--border-color); border-left: 3px solid var(--accent); border-radius: 6px; font-size: 12px; cursor: pointer; transition: background 0.15s, border-color 0.15s; }
.folded-summary:hover { background: var(--bg-hover); border-color: var(--accent); }

/* ── 折叠按钮（展开后使用） ── */
.msg-fold-btn {
  display: flex; align-items: center; justify-content: center; gap: 4px;
  padding: 4px 0; margin-top: 4px;
  font-size: 11px; color: var(--text-muted); cursor: pointer;
  user-select: none; border-top: 1px solid transparent;
  transition: all 0.12s; opacity: 0.4;
}
.msg-fold-btn:hover { opacity: 1; color: var(--text-secondary); background: var(--bg-hover); border-radius: 4px; }

/* ── 时间线展示（替代旧 SubAgentBlock 卡片 + content-flow）── */
.bubble-agent { position: relative; }
.bubble-agent::before { content: ''; position: absolute; left: 8px; top: 0; bottom: 0; width: 2px; background: linear-gradient(180deg, var(--accent) 0%, var(--border-color) 100%); opacity: 0.4; border-radius: 1px; }
.tl-item { display: flex; align-items: flex-start; gap: 0; padding: 2px 0; position: relative; }
.tl-dot { position: absolute; left: 8px; top: 7px; width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; border: 2px solid var(--border-color); background: var(--bg-primary); z-index: 1; box-shadow: 0 0 0 2px var(--bg-primary); }
.tl-dot-thinking { border-color: var(--accent); background: var(--accent-bg); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px rgba(212,167,78,0.3); }
.tl-dot-tool { border-color: #d4a74e; background: rgba(212,167,78,0.2); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px rgba(212,167,78,0.2); }
.tl-dot-ask { border-color: #c586c0; background: rgba(197,134,192,0.2); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px rgba(197,134,192,0.2); }
.tl-dot-content { border-color: #6a9955; background: rgba(106,153,85,0.2); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px rgba(106,153,85,0.2); }
.tl-dot-done { border-color: var(--accent); background: var(--accent); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px rgba(126,184,218,0.4); }
.tl-body { flex: 1; min-width: 0; font-size: 13px; line-height: 1.6; padding-left: 20px; }
/* ── 思考段：背景区分 + 左边框 + 改进滚动条 ── */
.tl-think-body { position: relative; }
.tl-thinking-text { color: var(--text-secondary); font-style: italic; white-space: pre-wrap; padding: 6px 10px; max-height: 300px; overflow-y: auto; background: var(--bg-tertiary); border-radius: 6px; border-left: 2px solid var(--accent); margin: 2px 0; }
.tl-think-fold { position: sticky; bottom: 0; display: inline-block; font-size: 11px; color: var(--accent); cursor: pointer; padding: 3px 10px; margin-top: 4px; user-select: none; background: var(--bg-tertiary); border: 1px solid var(--border-color); border-radius: 4px; transition: background 0.15s; }
.tl-think-fold:hover { background: var(--bg-hover); color: var(--accent-light); }
.tl-thinking-collapsed { color: var(--text-muted); font-style: italic; font-size: 12px; cursor: pointer; padding: 2px 0; }
.tl-tc-header { display: flex; align-items: center; gap: 4px; cursor: pointer; padding: 4px 8px; user-select: none; border-radius: 4px; transition: background 0.15s; }
.tl-tc-header:hover { background: var(--bg-hover); }
.tl-tc-chevron { font-size: 9px; color: var(--text-muted); width: 8px; text-align: center; flex-shrink: 0; transition: transform 0.15s; }
.tl-tc-icon { flex-shrink: 0; color: var(--text-secondary); }
.tl-tc-name { font-size: 12px; font-weight: 500; color: var(--text-primary); font-family: var(--font-code); flex-shrink: 0; }
.tl-tc-param { font-size: 11px; color: var(--accent-light); margin-left: 4px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; min-width: 0; font-family: var(--font-code); }
.tl-tc-summary { font-size: 11px; color: var(--text-muted); margin-left: 4px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; min-width: 0; }
.tl-tc-detail { padding: 6px 0 6px 12px; margin: 2px 0 2px 4px; border-left: 1px solid var(--border-color); }
.tl-tc-section { margin-bottom: 4px; }
.tl-tc-section-title { font-size: 10px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 2px; font-weight: 500; }
.tl-tc-section pre { background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 4px; padding: 6px 8px; font-size: 11px; color: var(--text-secondary); max-height: 150px; overflow: auto; white-space: pre-wrap; font-family: var(--font-code); margin: 0; }
.tl-tc-command { background: #1e1e1e; color: #d4d4d4; padding: 6px 10px 6px 14px; border-radius: 4px; font-family: var(--font-code); font-size: 12px; white-space: pre-wrap; border: 1px solid var(--border-color); }
.tl-tc-output { background: #1e1e1e; color: #6a9955; padding: 8px 10px; border-radius: 4px; font-family: var(--font-code); font-size: 11px; white-space: pre-wrap; max-height: 200px; overflow: auto; border: 1px solid var(--border-color); }
/* Content 段：纯 Markdown，由 MarkdownRenderer 统一管理样式，此处不覆盖 */

/* ── nudge 提示条 ── */
.chat-nudge-bar { position: sticky; bottom: 0; z-index: 20; margin: 4px 12px; padding: 4px 10px; border-radius: 4px; background: var(--bg-tertiary); border: 1px solid var(--border-color); font-size: 11px; color: var(--text-muted); text-align: center; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; animation: nudgeFadeIn 0.3s ease; }
@keyframes nudgeFadeIn { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: translateY(0); } }

/* ── 滚动到底部按钮 ── */
.scroll-down-btn {
  position: sticky; bottom: 0; z-index: 30;
  display: flex; align-items: center; justify-content: center; gap: 6px;
  width: 100%; padding: 4px 0 6px;
  pointer-events: none;
  transition: opacity 0.25s ease;
}
.scroll-down-btn > button {
  pointer-events: auto;
  display: flex; align-items: center; justify-content: center; gap: 5px;
  padding: 5px 14px; border-radius: 20px;
  background: var(--accent); color: #fff;
  border: none; font-size: 12px; cursor: pointer;
  box-shadow: 0 2px 8px rgba(0,0,0,0.25);
  transition: background 0.15s, transform 0.15s;
  white-space: nowrap;
}
.scroll-down-btn > button:hover {
  background: var(--accent-hover, var(--accent));
  transform: scale(1.05);
}
.scroll-down-btn > button:active {
  transform: scale(0.95);
}
@keyframes scrollDownPulse {
  0%, 100% { box-shadow: 0 2px 8px rgba(0,0,0,0.25); }
  50% { box-shadow: 0 2px 16px rgba(0,0,0,0.4); }
}
.scroll-down-btn.show-pulse > button {
  animation: scrollDownPulse 1.5s ease infinite;
}

/* ── 完成报告卡已移除，由 EventDone 追加为 content segment ── */

/* ── 输入区 ── */
.chat-input-area { position: relative; flex-shrink: 0; padding: 0 8px 10px 8px; background: var(--bg-secondary); }
.input-resizer { position: absolute; top: -8px; left: 0; right: 0; height: 12px; cursor: ns-resize; z-index: 10; }
.chat-input { display: block; width: 100%; background: var(--input-bg); border: 1px solid var(--border-color); color: var(--text-primary); padding: 14px 16px 68px 16px; border-radius: 8px; font-size: 14px; resize: none; outline: none; min-height: 80px; font-family: inherit; line-height: 1.6; box-sizing: border-box; }
.input-overlay { position: absolute; right: 12px; bottom: 16px; display: flex; align-items: center; gap: 6px; pointer-events: none; }
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

/* ── 执行步骤容器（输入区上方）── */
.plan-container {
  flex-shrink: 0;
  transition: max-height 0.25s ease;
  padding: 0 8px;
}
.plan-container.plan-empty {
  max-height: 0;
  padding: 0 8px;
}
.plan-container:not(.plan-empty) {
  max-height: 400px;
}
.plan-container .plan-panel {
  margin: 0 0 4px 0;
}
</style>
