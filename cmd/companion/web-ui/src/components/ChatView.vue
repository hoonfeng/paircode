<template>
  <div class="chat-view">
    <!-- 头部 -->
    <div class="cv-header">
      <div class="cv-header-left">
        <span class="cv-title"><SvgIcon name="bot" :size="16" /> 对话</span>
        <span v-if="convName" class="cv-conv-name">{{ convName }}</span>
      </div>
      <div class="cv-header-actions">
        <button v-if="agentRunning" class="cv-btn cv-btn-stop" @click="stopAgent" title="停止">
          <SvgIcon name="square" :size="14" />
        </button>
        <button class="cv-btn" @click="newConversation" title="新对话">
          <SvgIcon name="plus" :size="14" />
        </button>
      </div>
    </div>

    <!-- 消息区 -->
    <div class="cv-messages" ref="msgRef" @scroll="onScroll">
      <div v-if="hasMoreTop" class="scroll-more-hint" ref="topSentinel">
        <span>加载更早消息...</span>
      </div>

      <div class="msg-list-wrap">
        <template v-for="(combo, ci) in messageCombos" :key="'c' + ci">
          <!-- 用户消息 -->
          <div v-if="combo.user" class="msg-item msg-user">
            <div class="msg-avatar"><SvgIcon name="user" :size="16" /></div>
            <div class="msg-bubble bubble-user">
              <div v-if="combo.user.content" class="user-msg-content">
                <MarkdownRenderer :text="cleanMsgContent(combo.user)" :theme="state.theme" />
              </div>
              <div v-else class="user-msg-placeholder">（空消息）</div>
              <div v-if="combo.user._time" class="msg-time">{{ combo.user._time }}</div>
            </div>
          </div>

          <!-- 助手消息 -->
          <div v-if="combo.assistant" class="msg-item msg-assistant">
            <div class="msg-avatar"><SvgIcon name="bot" :size="16" /></div>
            <div class="msg-bubble bubble-assistant">
              <!-- 用户反馈合并 -->
              <div v-if="combo.assistant._feedbacks && combo.assistant._feedbacks.length > 0" class="fb-merged-section">
                <div v-for="(fb, fi) in combo.assistant._feedbacks" :key="'fb'+fi" class="fb-merged-item">
                  <div class="fb-merge-label"><SvgIcon name="message-square" :size="11" /> 用户反馈</div>
                  <div class="fb-merge-content">{{ fb.content }}</div>
                </div>
              </div>

              <!-- 分段渲染 -->
              <template v-if="combo.assistant.segments && combo.assistant.segments.length > 0">
                <div v-if="combo.assistant._folded" class="folded-summary" @click="combo.assistant._folded = false">
                  <span class="folded-chevron">▸</span>
                  <SvgIcon name="list" :size="11" />
                  <span class="folded-title">完成摘要</span>
                  <span class="folded-desc">{{ msgSummary(combo.assistant) }}</span>
                </div>
                <template v-if="!combo.assistant._folded">
                  <template v-for="(seg, si) in combo.assistant.segments" :key="si">
                    <!-- 思考 -->
                    <div v-if="seg.type === 'thinking'" class="tl-item">
                      <span class="tl-dot tl-dot-thinking"></span>
                      <div class="tl-body tl-think-body">
                        <div v-if="!seg._collapsed" class="tl-thinking-text">{{ seg.content }}</div>
                        <div v-else class="tl-thinking-collapsed" @click="seg._collapsed = false">
                          <SvgIcon name="message-square" :size="12" /> 思考…
                        </div>
                        <div v-if="!seg._collapsed" class="tl-think-fold" @click.stop="seg._collapsed = true" title="折叠思考">▲ 收起</div>
                      </div>
                    </div>
                    <!-- 工具调用 -->
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
                    <!-- ask_user -->
                    <div v-else-if="seg.type === 'ask_user'" class="tl-item">
                      <span class="tl-dot tl-dot-ask"></span>
                      <div class="tl-body"><AskUserCard :question="seg.question" :ask-type="seg.askType" :options="seg.options" :call-id="seg.callId" :answered="seg._answered" @answer="(ans) => onAskAnswer(seg, ans)" /></div>
                    </div>
                    <!-- 正文 -->
                    <div v-else-if="seg.type === 'content'" class="tl-item tl-content-item">
                      <span class="tl-dot tl-dot-content"></span>
                      <div class="tl-body"><MarkdownRenderer :text="seg.content" :theme="state.theme" /></div>
                    </div>
                  </template>
                </template>
              </template>

              <!-- 历史消息 fallback -->
              <template v-if="!combo.assistant.segments || combo.assistant.segments.length === 0">
                <div v-if="combo.assistant.content" class="tl-item tl-content-item">
                  <span class="tl-dot tl-dot-content"></span>
                  <div class="tl-body"><MarkdownRenderer :text="combo.assistant.content" :theme="state.theme" /></div>
                </div>
              </template>

              <div v-if="!combo.assistant._folded && combo.assistant.segments && combo.assistant.segments.length > 0" class="msg-fold-btn" @click="combo.assistant._folded = true">
                <SvgIcon name="chevron-up" :size="12" /><span>折叠输出</span>
              </div>
              <div v-if="combo.assistant._time" class="msg-time">{{ combo.assistant._time }}</div>
            </div>
            <div v-if="combo.assistant._loading" class="msg-loading-dots">
              <span class="dot"></span><span class="dot"></span><span class="dot"></span>
            </div>
          </div>
        </template>
      </div>

      <!-- 空状态 -->
      <div v-if="(!msgs || msgs.length === 0) && !chatLoading" class="chat-empty">
        <div class="chat-empty-icon"><SvgIcon name="bot" :size="32" /></div>
        <div class="chat-empty-text">{{ convId ? '开始新的对话' : '选择一个对话开始交流' }}</div>
        <div class="chat-empty-hint">发送消息即可与 AI 助手对话</div>
      </div>

      <!-- 跳底按钮 -->
      <div v-if="showScrollDown" class="scroll-down-btn" :class="{ 'show-pulse': chatLoading }" @click.stop="scrollToBottom">
        <button><SvgIcon name="chevron-down" :size="14" /> 新消息</button>
      </div>
    </div>

    <!-- 输入区 -->
    <div class="cv-input-area">
      <textarea
        ref="inputRef"
        v-model="inputText"
        class="cv-input"
        placeholder="输入消息…（Enter 发送，Shift+Enter 换行）"
        rows="2"
        @keydown="onKeydown"
        :disabled="chatLoading"
      ></textarea>
      <button class="cv-send-btn" @click="sendMessage" :disabled="!inputText.trim() || chatLoading">
        <SvgIcon name="send" :size="16" />
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { state } from '../main.js'
import api from '../api.js'
import { useMessageCombos, cleanMsgContent, isFeedback, isDelegation, delegationAgent, msgSummary, toolMeta, toolResultSummary, isTerminalTool, formatTerminalCommand, mergeConsecutiveAssistant } from '../chat-utils.js'
import MarkdownRenderer from './MarkdownRenderer.vue'
import AskUserCard from './AskUserCard.vue'
import SvgIcon from './SvgIcon.vue'

const route = useRoute()
const router = useRouter()

const convId = computed(() => route.params?.convId || '')
const convName = ref('')
const inputText = ref('')
const chatLoading = computed(() => state.loadingByConv[convId.value] || false)
const agentRunning = computed(() => state.agentRunningByConv[convId.value] || false)
const msgRef = ref(null)
const inputRef = ref(null)
const showScrollDown = ref(false)
const loadingMoreTop = ref(false)

// 消息数据
const msgs = computed(() => {
  return convId.value ? (state.messagesByConv[convId.value] || []) : []
})

const messageCombos = useMessageCombos(msgs)

// 是否有更早消息
const hasMoreTop = computed(() => {
  if (!msgs.value || msgs.value.length === 0) return false
  if (msgs.value[0]._noMoreAbove) return false
  const oldestIdx = msgs.value[0]._idx
  return oldestIdx !== undefined && oldestIdx !== null && oldestIdx > 0
})

// 加载对话消息
async function loadMessages() {
  if (!convId.value) return
  try {
    const data = await api.getMessages(convId.value, { limit: 50 })
    const msgs = (data.messages || [])
      .filter(m => (m.message?.role || m.role) !== 'tool')
      .map((m, i) => ({
        role: m.message?.role || m.role || '',
        content: m.message?.content || m.content || '',
        segments: (m.segments || []).map(seg => {
          if (seg.type === 'ask_user') seg._answered = !!seg.answer
          return seg
        }),
        _key: 'msg_' + Date.now() + '_' + i,
        _idx: m.idx,
        _time: m.timestamp || '',
      }))
    const merged = mergeConsecutiveAssistant(msgs)
    state.messagesByConv[convId.value] = merged
    state.msgLoadedByConv[convId.value] = merged.length
    // 获取总数
    try {
      const cnt = await api.getMessagesCount(convId.value)
      state.msgTotalByConv[convId.value] = cnt.total || merged.length
    } catch {}
  } catch (e) {
    console.warn('ChatView: 加载消息失败', e)
  }
}

// 切换对话时重新加载
watch(convId, (newId) => {
  if (newId) {
    state.currentConvId = newId
    if (!state.messagesByConv[newId] || state.messagesByConv[newId].length === 0) {
      loadMessages()
    }
    // 加载对话名称
    tryLoadConvName(newId)
    nextTick(() => scrollToBottom())
  }
})

async function tryLoadConvName(id) {
  try {
    const data = await api.apiGet('/conversations')
    const convs = data.conversations || data || []
    const found = convs.find(c => c.id === id)
    if (found) convName.value = found.title || ''
  } catch {}
}

// 发送消息
async function sendMessage() {
  const text = inputText.value.trim()
  if (!text || chatLoading.value) return
  if (!convId.value) {
    // 创建新对话
    newConversation()
    return
  }

  inputText.value = ''

  // 创建助手占位消息
  const msgKey = 'msg_' + Date.now() + '_' + Math.random()
  if (!state.messagesByConv[convId.value]) state.messagesByConv[convId.value] = []
  state.messagesByConv[convId.value].push({
    role: 'assistant',
    content: '',
    segments: [],
    _key: msgKey,
    _idx: (state.messagesByConv[convId.value].length || 0),
    _time: '',
    _loading: true,
  })

  state.loadingByConv[convId.value] = true

  try {
    await api.chatStart(convId.value, text, false, state.workspaceRoot)
  } catch (e) {
    console.error('chatStart 失败:', e)
    state.loadingByConv[convId.value] = false
    // 移除占位消息
    const arr = state.messagesByConv[convId.value]
    const idx = arr.findIndex(m => m._key === msgKey)
    if (idx >= 0) arr.splice(idx, 1)
  }
}

// 停止 agent
async function stopAgent() {
  if (!convId.value) return
  await api.chatStop(convId.value)
}

// 新对话
function newConversation() {
  const wsId = route.params?.wsId || state.workspaceRoot?.split('/').pop() || ''
  router.push('/workspace/' + encodeURIComponent(wsId || 'default'))
}

// ask_user 回答
async function onAskAnswer(seg, answer) {
  if (!convId.value) return
  seg._answered = true
  seg.answer = answer
  await api.answerChat(convId.value, answer)
}

// 键盘
function onKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

// 滚动
let scrollLocked = false

function scrollToBottom() {
  nextTick(() => {
    if (!msgRef.value) return
    msgRef.value.scrollTop = msgRef.value.scrollHeight
    showScrollDown.value = false
    scrollLocked = false
  })
}

// 向前分页加载更早消息：prepend 到数组并维护滚动位置
async function loadMoreMessages() {
  const id = convId.value
  if (!id) return
  const cur = msgs.value
  if (!cur || cur.length === 0) return
  if (cur[0]._noMoreAbove) return
  const oldestIdx = cur[0]._idx
  if (oldestIdx === undefined || oldestIdx === null || oldestIdx <= 0) return
  loadingMoreTop.value = true
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
      const mergedBefore = mergeConsecutiveAssistant([...older, ...cur])
      state.messagesByConv[id] = mergedBefore
      state.msgLoadedByConv[id] = (state.msgLoadedByConv[id] || 0) + older.length
      // 补偿滚动位置：保持当前视口内容不动（新增的 older 消息在顶部）
      nextTick(() => {
        if (msgRef.value) {
          msgRef.value.scrollTop = oldScrollTop + (msgRef.value.scrollHeight - oldScrollHeight)
        }
      })
    } else {
      // 无更早消息：标记防止重复请求
      cur[0]._noMoreAbove = true
    }
  } catch (e) {
    console.warn('ChatView: 加载更早消息失败', e)
  } finally {
    loadingMoreTop.value = false
  }
}

function onScroll() {
  if (!msgRef.value) return
  const el = msgRef.value
  const threshold = 100
  const nearBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - threshold
  if (!nearBottom) scrollLocked = true
  if (nearBottom) { scrollLocked = false; showScrollDown.value = false }
  else showScrollDown.value = true
  // 顶部懒加载：scrollTop < 100 且还有更早消息可加载
  if (el.scrollTop < threshold && !loadingMoreTop.value) {
    loadMoreMessages()
  }
}

onMounted(() => {
  const initId = route.params?.convId
  if (initId) {
    state.currentConvId = initId
    if (!state.messagesByConv[initId]) loadMessages()
    tryLoadConvName(initId)
    nextTick(() => scrollToBottom())
  }
})

// 暴露给父组件
defineExpose({ scrollToBottom })
</script>

<style scoped>
.chat-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-primary, #1e1e1e);
  color: var(--text-primary, #d4d4d4);
}

/* ── 头部 ── */
.cv-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border-color, #333);
  flex-shrink: 0;
}
.cv-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}
.cv-title {
  font-size: 14px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 6px;
}
.cv-conv-name {
  font-size: 12px;
  color: var(--text-muted, #888);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cv-header-actions {
  display: flex;
  gap: 4px;
}
.cv-btn {
  background: transparent;
  border: none;
  color: var(--text-muted, #888);
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 4px;
  display: flex;
  align-items: center;
}
.cv-btn:hover { color: var(--text-primary, #ccc); background: var(--bg-hover, #333); }
.cv-btn-stop { color: #e74c3c; }
.cv-btn-stop:hover { color: #ff5f5f; }

/* ── 消息区 ── */
.cv-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  position: relative;
}
.msg-list-wrap {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* ── 消息项 ── */
.msg-item {
  display: flex;
  gap: 10px;
  max-width: 85%;
}
.msg-user { align-self: flex-end; flex-direction: row-reverse; }
.msg-assistant { align-self: flex-start; }

.msg-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--bg-secondary, #2d2d2d);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.msg-bubble {
  padding: 10px 14px;
  border-radius: 10px;
  font-size: 13px;
  line-height: 1.6;
  word-break: break-word;
}
.bubble-user {
  background: var(--color-accent, #2b6cb0);
  color: #fff;
  border-bottom-right-radius: 3px;
}
.bubble-assistant {
  background: var(--bg-secondary, #2d2d2d);
  border: 1px solid var(--border-color, #333);
  border-bottom-left-radius: 3px;
}

.user-msg-content p { margin: 0; }
.user-msg-placeholder { color: rgba(255,255,255,0.5); font-style: italic; }
.msg-time {
  font-size: 10px;
  color: rgba(255,255,255,0.4);
  margin-top: 4px;
}

/* ── 折叠摘要 ── */
.folded-summary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: rgba(255,255,255,0.03);
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
  color: var(--text-muted, #888);
}
.folded-summary:hover { background: rgba(255,255,255,0.06); }
.folded-chevron { font-size: 10px; }
.folded-title { font-weight: 600; }
.folded-desc { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

/* ── 时间线 ── */
.tl-item {
  display: flex;
  gap: 8px;
  padding: 4px 0;
}
.tl-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-top: 6px;
  flex-shrink: 0;
}
.tl-dot-thinking { background: #a78bfa; }
.tl-dot-tool { background: #60a5fa; }
.tl-dot-content { background: #34d399; }
.tl-dot-ask { background: #fbbf24; }
.tl-body { flex: 1; min-width: 0; }

.tl-think-body {
  font-size: 12px;
  color: var(--text-muted, #888);
  border-left: 2px solid rgba(167,139,250,0.3);
  padding-left: 8px;
}
.tl-thinking-text { white-space: pre-wrap; }
.tl-thinking-collapsed { color: var(--text-muted, #666); cursor: pointer; font-size: 12px; }
.tl-think-fold { font-size: 11px; cursor: pointer; color: #888; margin-top: 4px; }

/* ── 工具调用 ── */
.tl-tc-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: rgba(96,165,250,0.08);
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
}
.tl-tc-header:hover { background: rgba(96,165,250,0.14); }
.tl-tc-chevron { font-size: 10px; color: #60a5fa; flex-shrink: 0; }
.tl-tc-icon { flex-shrink: 0; }
.tl-tc-name { font-weight: 600; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex-shrink: 1; }
.tl-tc-param { color: var(--text-muted, #888); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; min-width: 0; flex-shrink: 1; }
.tl-tc-summary { color: #34d399; margin-left: auto; font-size: 11px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; min-width: 0; flex-shrink: 1; }
.tl-tc-detail {
  background: var(--bg-primary, #1e1e1e);
  border: 1px solid var(--border-color, #333);
  border-radius: 0 0 6px 6px;
  padding: 10px;
}
.tl-tc-section { margin-bottom: 8px; }
.tl-tc-section:last-child { margin-bottom: 0; }
.tl-tc-section-title { font-size: 11px; color: var(--text-muted, #888); margin-bottom: 4px; font-weight: 600; }
.tl-tc-command { font-family: monospace; font-size: 12px; color: #60a5fa; background: rgba(0,0,0,0.3); padding: 6px 8px; border-radius: 4px; }
.tl-tc-output { font-size: 12px; white-space: pre-wrap; margin: 0; color: var(--text-muted, #aaa); max-height: 300px; overflow-y: auto; }
.tl-tc-detail pre { margin: 0; font-size: 12px; white-space: pre-wrap; }
.tl-tc-detail code { font-size: 12px; }

.msg-fold-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 6px;
  font-size: 11px;
  color: var(--text-muted, #666);
  cursor: pointer;
  padding: 2px 6px;
}
.msg-fold-btn:hover { color: var(--text-primary, #ccc); }

/* ── 反馈合并 ── */
.fb-merged-section { margin-bottom: 8px; }
.fb-merged-item {
  background: rgba(251,191,36,0.08);
  border-left: 3px solid #fbbf24;
  padding: 6px 10px;
  margin-bottom: 4px;
  border-radius: 0 4px 4px 0;
}
.fb-merge-label { font-size: 10px; color: #fbbf24; margin-bottom: 2px; display: flex; align-items: center; gap: 4px; }
.fb-merge-content { font-size: 12px; color: var(--text-muted, #aaa); }

/* ── 加载中 ── */
.msg-loading-dots {
  display: flex;
  gap: 4px;
  padding: 4px 0 0 38px;
}
.msg-loading-dots .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-muted, #666);
  animation: dotPulse 1.4s infinite;
}
.msg-loading-dots .dot:nth-child(2) { animation-delay: 0.2s; }
.msg-loading-dots .dot:nth-child(3) { animation-delay: 0.4s; }
@keyframes dotPulse {
  0%, 80%, 100% { opacity: 0.3; }
  40% { opacity: 1; }
}

/* ── 空状态 ── */
.chat-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: var(--text-muted, #666);
}
.chat-empty-icon { margin-bottom: 16px; opacity: 0.3; }
.chat-empty-text { font-size: 16px; margin-bottom: 8px; }
.chat-empty-hint { font-size: 13px; }

/* ── 跳底 ── */
.scroll-down-btn {
  position: absolute;
  bottom: 16px;
  right: 24px;
  z-index: 10;
}
.scroll-down-btn button {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border-radius: 20px;
  border: 1px solid var(--border-color, #333);
  background: var(--bg-secondary, #2d2d2d);
  color: var(--text-primary, #ccc);
  cursor: pointer;
  font-size: 12px;
}
.scroll-down-btn button:hover { background: var(--bg-hover, #3a3a3a); }
.scroll-down-btn.show-pulse button { animation: scrollPulse 1s ease-in-out infinite; }
@keyframes scrollPulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(96,165,250,0.4); }
  50% { box-shadow: 0 0 0 6px rgba(96,165,250,0); }
}

/* ── 滚动更多 ── */
.scroll-more-hint {
  text-align: center;
  padding: 8px;
  font-size: 12px;
  color: var(--text-muted, #666);
}

/* ── 输入区 ── */
.cv-input-area {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--border-color, #333);
  flex-shrink: 0;
}
.cv-input {
  flex: 1;
  resize: none;
  background: var(--bg-secondary, #2d2d2d);
  border: 1px solid var(--border-color, #444);
  border-radius: 8px;
  color: var(--text-primary, #d4d4d4);
  padding: 8px 12px;
  font-size: 13px;
  font-family: inherit;
  line-height: 1.5;
}
.cv-input:focus { outline: none; border-color: var(--color-accent, #2b6cb0); }
.cv-input:disabled { opacity: 0.5; }
.cv-send-btn {
  background: var(--color-accent, #2b6cb0);
  border: none;
  border-radius: 8px;
  color: #fff;
  cursor: pointer;
  padding: 8px 14px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.cv-send-btn:hover { background: #3b7dd0; }
.cv-send-btn:disabled { opacity: 0.4; cursor: not-allowed; }
</style>
