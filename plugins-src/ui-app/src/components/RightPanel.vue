<template>
  <div class="right-panel">
    <!-- 标题 -->
    <div class="rp-header">
      <span class="rp-header-title"><SvgIcon name="bot" :size="16" /> 对话</span>
      <div class="rp-header-actions">
        <button class="rp-btn" @click="newConversation" title="新对话"><SvgIcon name="plus" :size="14" /></button>
        <button class="rp-btn" @click="showDebugLog = !showDebugLog" title="Debug 日志"><SvgIcon name="bug" :size="14" /></button>
        <button v-if="!panelMode" class="rp-btn" @click="toggleFocus" :title="state.focusMode ? '退出专注（显示编辑器/终端）' : '专注对话（隐藏编辑器，保留文件面板）'">          <SvgIcon :name="state.focusMode ? 'eye-off' : 'eye'" :size="14" />
        </button>
        <button v-if="!panelMode" class="rp-btn" @click="toggleRight" title="关闭"><SvgIcon name="close" :size="14" /></button>
      </div>
    </div>

    <div class="rp-body">
      <!-- ★ chat 槽位（Slot 系统）：插件注册 chat 槽位并激活后，整个对话面板由插件渲染（UI 可更换） -->
      <div v-if="chatSlot.owner.value" :ref="chatSlot.hostRef" class="plugin-slot-host plugin-slot-chat"></div>
      <template v-else>
      <!-- 左侧：聊天消息 + 输入区 -->
      <div class="chat-area">
        <!-- 阶段指示器（自主模式多阶段切换） -->
        <div v-if="currentPhase || agentRunning" class="phase-bar">
          <span class="phase-icon"><SvgIcon :name="phaseIcon(currentPhase)" :size="14" /></span>
          <span class="phase-text">{{ currentPhase || '执行中…' }}</span>
          <span class="phase-stats">
            <span v-if="phaseToolCount > 0" class="phs-item"><SvgIcon name="zap" :size="10" /> {{ phaseToolCount }} 次调用</span>
            <span v-if="phaseElapsed" class="phs-item"><SvgIcon name="clock" :size="10" /> {{ phaseElapsed }}</span>
          </span>
          <span class="phase-bar-track"><span class="phase-bar-fill" :style="{ width: phaseProgress + '%' }"></span></span>
        </div>
        <div class="chat-messages" ref="msgRef" @scroll="onScroll">
          <!-- 顶部加载更多提示 -->
          <div v-if="hasMoreTop" class="scroll-more-hint" ref="topSentinel">
            <span>加载更早消息...</span>
          </div>
          <!-- 渲染的消息列表（全量渲染，overflow-anchor 保底部稳定） -->
          <div class="msg-list-wrap">
            <!-- ★ 遍历 messageCombos，每个 user / assistant 分别渲染为独立气泡 -->
            <template v-for="(combo, ci) in messageCombos" :key="'c' + ci">
              <!-- ── 背景上下文快照（循环同步的消息流背景信息，折叠系统信息条） ── -->
              <div v-if="combo._snapshots && combo._snapshots.length > 0" class="snapshot-strip">
                <div v-for="(snap, si) in combo._snapshots" :key="'snap'+si" class="snapshot-item" :class="{ open: snap._open }">
                  <div class="snapshot-head" @click="snap._open = !snap._open">
                    <svg class="folded-chevron" :class="{ rotated: snap._open }" viewBox="0 0 8 8" width="9" height="9" fill="currentColor" aria-hidden="true"><path d="M2.6 1.2 L6.8 4 L2.6 6.8 Z"/></svg>
                    <SvgIcon name="file-text" :size="11" />
                    <span>背景上下文</span>
                    <span class="snapshot-hint">（非当前任务）</span>
                  </div>
                  <div v-if="snap._open" class="snapshot-body">
                    <MarkdownRenderer :text="cleanMsgContent(snap)" :theme="state.theme" />
                  </div>
                </div>
              </div>
              <!-- ── 用户消息独立气泡（右对齐） ── -->
              <div v-if="combo.user" class="msg-item msg-user" :data-idx="combo.user._idx">
                <div class="msg-avatar"><SvgIcon name="user" :size="16" /></div>
                <div class="msg-bubble bubble-user">
                  <div v-if="isDelegation(combo.user)" class="user-msg-header">
                    <span class="umh-badge badge-delegation"><SvgIcon name="git-branch" :size="10" /> 委派任务</span>
                    <span class="umh-agent">{{ delegationAgent(combo.user) }}</span>
                  </div>
                  <div v-else-if="isFeedback(combo.user)" class="user-msg-header">
                    <span class="umh-badge badge-feedback"><SvgIcon name="message-square" :size="10" /> 用户反馈</span>
                  </div>
                  <div v-if="combo.user.content" class="user-msg-content">
                    <MarkdownRenderer :text="cleanMsgContent(combo.user)" :theme="state.theme" />
                  </div>
                  <div v-else class="user-msg-placeholder">（空消息）</div>
                  <div v-if="combo.user._attachments && combo.user._attachments.length > 0" class="user-attachments">
                    <div v-for="(att, ai) in combo.user._attachments" :key="'att'+ai" class="att-tag" :class="'att-tag-' + att.type">
                      <SvgIcon :name="att.type === 'image' ? 'image' : 'file'" :size="12" />
                      <span class="att-tag-label">{{ att.label }}</span>
                    </div>
                  </div>
                  <div class="rollback-area" v-if="!state.chatLoading">
                    <button class="rollback-btn" @click="rollbackTo(combo.user._idx)" title="回到此消息前的状态"><SvgIcon name="undo" :size="11" /> 回退</button>
                  </div>
                  <div v-if="combo.user._time" class="msg-time">{{ combo.user._time }}</div>
                </div>
              </div>
              <!-- ── Agent 回复独立气泡（左对齐），不分段，直接显示到一起 ── -->
              <div v-if="combo.assistant" class="msg-item msg-assistant" :data-idx="combo.assistant._idx">
                <div class="msg-avatar"><SvgIcon name="bot" :size="16" /></div>
                <div class="msg-bubble bubble-assistant">
                  <!-- ★ 2026-09-10 slash 命令结果卡片：命令命中执行后本地渲染（不唤醒模型） -->
                  <div v-if="combo.assistant._isSlashResult" class="slash-result-card">
                    <div class="slash-result-head">
                      <SvgIcon name="terminal" :size="12" />
                      <span>命令 /{{ combo.assistant._slashName }} 执行结果</span>
                    </div>
                  </div>
                  <!-- ★ 用户反馈标记（合并到 agent 输出中，不显示为独立用户气泡） -->
                  <div v-if="combo.assistant._feedbacks && combo.assistant._feedbacks.length > 0" class="fb-merged-section">
                    <div v-for="(fb, fi) in combo.assistant._feedbacks" :key="'fb'+fi" class="fb-merged-item">
                      <div class="fb-merge-label"><SvgIcon name="message-square" :size="11" /> 用户反馈</div>
                      <div class="fb-merge-content">{{ fb.content }}</div>
                    </div>
                  </div>
                  <!-- Agent 分段渲染（兼容 WS 流式更新通过 pushSegment 写入 segments） -->
                  <template v-if="combo.assistant.segments && combo.assistant.segments.length > 0">
                    <div v-if="combo.assistant._folded" class="folded-summary" @click="combo.assistant._folded = !combo.assistant._folded">
                      <svg class="folded-chevron" viewBox="0 0 8 8" width="9" height="9" fill="currentColor" aria-hidden="true"><path d="M2.6 1.2 L6.8 4 L2.6 6.8 Z"/></svg>
                      <SvgIcon name="list" :size="11" />
                      <span class="folded-title">完成摘要</span>
                      <span class="folded-desc">{{ msgSummary(combo.assistant) }}</span>
                    </div>
                    <template v-if="!combo.assistant._folded">
                      <template v-for="(seg, si) in combo.assistant.segments" :key="si">
                        <div v-if="seg.type === 'thinking'" class="tl-item">
                          <span class="tl-dot tl-dot-thinking"></span>
                          <div class="tl-body tl-think-body">
                            <div v-if="!seg._collapsed" class="tl-thinking-text">{{ seg.content }}</div>
                            <div v-else class="tl-thinking-collapsed" @click="seg._collapsed = !seg._collapsed"><SvgIcon name="message-square" :size="12" /> 思考…</div>
                            <div v-if="!seg._collapsed" class="tl-think-fold" @click.stop="seg._collapsed = !seg._collapsed" title="折叠思考">▲ 收起</div>
                          </div>
                        </div>
                        <div v-else-if="seg.type === 'tool_call'" class="tl-item">
                          <span class="tl-dot tl-dot-tool"></span>
                          <div class="tl-body tl-tool">
                            <div class="tl-tc-header" @click="seg._expanded = !seg._expanded">
                              <svg v-if="!seg._expanded" class="tl-tc-chevron" viewBox="0 0 8 8" width="8" height="8" fill="currentColor" aria-hidden="true"><path d="M2.6 1.2 L6.8 4 L2.6 6.8 Z"/></svg>
                              <svg v-else class="tl-tc-chevron" viewBox="0 0 8 8" width="8" height="8" fill="currentColor" aria-hidden="true"><path d="M1.2 2.6 L4 6.8 L6.8 2.6 Z"/></svg>
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
                        <div v-else-if="seg.type === 'ask_user'" class="tl-item">
                          <span class="tl-dot tl-dot-ask"></span>
                          <div class="tl-body"><AskUserCard :question="seg.question" :ask-type="seg.askType" :options="seg.options" :questions="seg.questions" :call-id="seg.callId" :answered="seg._answered" @answer="onAskAnswer(seg, $event)" /></div>
                        </div>
                        <div v-else-if="seg.type === 'content'" class="tl-item tl-content-item">
                          <span class="tl-dot tl-dot-content"></span>
                          <div class="tl-body"><MarkdownRenderer :text="seg.content" :theme="state.theme" /></div>
                        </div>
                      </template>
                    </template>
                  </template>
                  <div v-if="!combo.assistant._folded && combo.assistant.segments && combo.assistant.segments.length > 0 && !hasUnansweredAsk(combo.assistant)" class="msg-fold-btn" @click="combo.assistant._folded = true">
                    <SvgIcon name="chevron-up" :size="12" /><span>折叠输出</span>
                  </div>
                  <!-- 历史消息 fallback：有 content 但无 segments -->
                  <template v-if="(!combo.assistant.segments || combo.assistant.segments.length === 0)">
                    <div v-if="combo.assistant.content" class="tl-item tl-content-item">
                      <span class="tl-dot tl-dot-content"></span>
                      <div class="tl-body"><MarkdownRenderer :text="combo.assistant.content" :theme="state.theme" /></div>
                    </div>
                  </template>
                  <div v-if="combo.assistant._time" class="msg-time">{{ combo.assistant._time }}</div>
                </div>
                <div v-if="combo.assistant._loading" class="msg-loading-dots">
                  <span class="dot"></span><span class="dot"></span><span class="dot"></span>
                </div>
              </div>
            </template>
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
        <!-- 任务进度面板（task 体系：扁平任务列表） -->
        <div class="task-container" :class="{ 'task-empty': currentTasks.length === 0 }">
          <TaskPanel v-if="currentTasks.length > 0" :tasks="currentTasks" :expanded="tasksExpanded" @toggle="tasksExpanded = !tasksExpanded" />
        </div>
        <!-- 输入区 -->
        <div class="chat-input-area" ref="chatInputAreaRef">
          <!-- ★ chat-tools 槽位（list 型）：输入区上方工具条，插件可叠加快捷按钮（@文件/常用命令/图片等） -->
          <div ref="chatToolsEl" class="plugin-slot-host plugin-slot-chat-tools"></div>
          <!-- ★ 未完成任务提示条：上次运行异常中断/未完成时显示，一键继续 -->
          <div v-if="currentConvInterrupted" class="resume-banner">
            <span class="resume-icon">⚠️</span>
            <span class="resume-text">上次任务未完成，本对话上下文与进度已保留，可直接继续</span>
            <button class="resume-btn" @click="continueTask" title="沿用本对话上下文继续执行未完成任务">
              <SvgIcon name="refresh" :size="11" /> 继续任务
            </button>
          </div>
          <ApprovalBar v-if="approvalState.waiting" :waiting="approvalState.waiting" :tool="approvalState.tool" :args="approvalState.args" :parsedArgs="approvalState.parsedArgs" @resolve="resolveApproval" />
          <!-- 运行时反馈条（Agent 执行中可补充纠正） -->
          <div v-if="state.chatLoading" class="feedback-bar">
            <input class="feedback-input" v-model="feedbackText" @keydown="onFeedbackKeydown" placeholder="输入补充/纠正信息，Agent 将在下一轮响应中处理..." />
            <button class="feedback-send-btn" @click="sendFeedback" :disabled="!feedbackText.trim()" title="发送反馈"><SvgIcon name="send" :size="14" /></button>
          </div>
          <div class="input-resizer" @mousedown.prevent="startInputResize" title="拖拽调整高度"></div>
          <div class="input-wrapper">
            <!-- ★ Round3 ④.2 slash 命令菜单：输入以 "/" 开头时拉 /api/commands 提示，
                  ↑/↓ 移动选中，Enter 把选中命令写入输入框（可继续编辑/加参数），再次 Enter 执行
                  （结果由后端注入系统消息）；菜单外以 "/" 开头回车=直接执行；无匹配命令时原样发送（降级零破坏） -->
            <div v-if="slashOpen" class="slash-menu">
              <div v-for="(c, i) in slashMatches" :key="c.name"
                   :class="['slash-item', { active: i === slashIndex }]"
                   @mousedown.prevent="pickSlashCommand(c)">
                <span class="slash-name">/{{ c.name }}<em v-if="c.onDemand" class="slash-ondemand">按需</em></span>
                <span class="slash-desc">{{ c.description || '' }}</span>
              </div>
            </div>
            <!-- ★ 2026-08-22 输入框改造：contenteditable，附件以内联 tag 渲染在输入框内（光标处），
                 不再「上方 badge 区 + 文本内 token」。tag 点 × 或 Backspace/Delete 可删除。
                 文本内容由 @input 序列化到 inputText（tag → @@attN@@ token），发送时替换为语义化引用。 -->
            <div class="chat-input" ref="inputRef" :contenteditable="state.chatLoading ? 'false' : 'true'" :style="{ height: inputHeight + 'px' }" data-placeholder="发送消息到 AI... (Enter 发送, Shift+Enter 换行)" @keydown="onKeydown" @input="onInput" @dragover.prevent @drop="handleDrop" @paste="handlePaste"></div>
            <div class="input-bottom-bar">
              <div class="ibb-btns">
                <!-- ★ composer 模型选择器（2026-09-03 配置列表驱动；2026-09-05 移动端化：
                       bottom-sheet 弹层替代传统 <select>）：
                       下拉 = AI 设置面板「AI 配置」列表（ai-presets.json）：分组=配置名，
                       每组模型 = 该配置对应服务商（models.json）的可用模型列表；
                       选中配置只写当前会话（PUT /api/conversations/{id}），不改全局设置——历史对话保持各自模型不被牵连。 -->
                <SheetPicker
                  v-model="composerModel"
                  :items="modelSheetItems"
                  title="选择模型"
                  placeholder="选择模型…"
                  empty-text="暂无可用 AI 配置：请先在「设置 → AI → AI 配置」添加"
                  @change="onCmpModelChange"
                />
                <!-- ★ 2026-09-04 工具集（通用集合）模式选择器：会话级——选择当前对话
                     使用的工具集（default/full/dev/debug/test/docs 或自定义集合），
                     写入会话元数据（PUT /conversations/{id} toolset）只影响本会话；
                     agent 工具面按所选集合收敛（发送消息时后端应用）。
                     ★ 2026-09-05 移动端化：bottom-sheet 弹层替代传统 <select>。 -->
                <SheetPicker
                  v-if="toolsetItems.length"
                  v-model="convToolset"
                  :items="toolsetSheetItems"
                  title="切换工具集"
                  placeholder="选择工具集…"
                  @change="onConvToolsetChange"
                />
                <span class="obtn-sep"></span>
                <span :class="['obtn', reviewBtnClass]" @click="cycleReviewMode" :title="reviewBtnTitle"><SvgIcon :name="reviewIconName" :size="12" /> {{ reviewBtnLabel }}</span>
                <span :class="['obtn', { active: autoCollapse }]" @click="toggleAuto('autoCollapse')" title="自动折叠：新消息发出时折叠旧输出，显示完成摘要"><SvgIcon name="list" :size="12" /> 折叠</span>
                <span class="obtn-sep"></span>
                <span :class="['obtn', 'obtn-agent', { active: autonomous }]" @click="toggleAuto('autonomous')" title="自主模式：开启=连续执行全部计划步骤，关闭=单次回复"><SvgIcon name="cycle" :size="12" color="#d4a74e" /> 自主</span>
              </div>
              <button v-if="!state.chatLoading" class="send-btn" @click="sendMessage" :disabled="!inputText.trim() && pendingAttachments.length === 0"><SvgIcon name="send-plane" :size="16" /></button>
              <button v-else class="stop-btn" @click="stopChat"><SvgIcon name="stop-dot" :size="20" /></button>
            </div>
          </div>
        </div>
      </div>
      <!-- 右侧：Debug日志面板 / 会话列表 -->
      <DebugLogPanel v-if="showDebugLog" @close="showDebugLog = false" />
      <ConvSidebar v-else :conversations="convList" :current-conv-id="state.currentConvId" :loading-by-conv="state.loadingByConv" :ws-token-stats="wsTokenStats" :conv-ctx-stats="convCtxStats" :ctx-max-tokens-val="state.settings.contextMaxTokens || 1000000" :width="convListWidth" @new-conversation="newConversation" @switch-conversation="switchConv" @delete-conversation="deleteConv" />
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { state, rightPanelWidth } from '../ui-state.js'
import api from '../api.js'
import { setGlobalCtx, startConvRuntime, resetConvRuntime, createAssistantPlaceholder, getConvRuntime, getConvCtxStats, resetConvCtxStats, normalizeAskType, markHistoryLoaded } from '../agent-events.js'
import { useSingleSlot, mountListSlot } from '../plugin-runtime.js'
import SvgIcon from './SvgIcon.vue'
import SheetPicker from './SheetPicker.vue'
import TaskPanel from './TaskPanel.vue'
import ApprovalBar from './ApprovalBar.vue'
import ConvSidebar from './ConvSidebar.vue'
import AskUserCard from './AskUserCard.vue'
// SubAgentBlock 不再使用，替换为内联时间线展示
import MarkdownRenderer from './MarkdownRenderer.vue'
import DebugLogPanel from './DebugLogPanel.vue'

const showDebugLog = ref(false)

const props = defineProps({ panelMode: { type: Boolean, default: false } })
const panelMode = computed(() => !!props.panelMode)
const toggleRight = () => { state.rightPanelVisible = false }
// 专注对话切换：专注模式只隐藏编辑器（main-area），文件资源侧边栏保留（sidebarVisible 独立控制）
const toggleFocus = () => {
  state.focusMode = !state.focusMode
}
const inputText = ref('')

// ─── slash 命令（Round3 ④.2：ctx.commands 面前端 "/" 菜单） ───
const slashCommands = ref([])   // 全量命令清单（/api/commands）
const slashMatches = ref([])    // 当前匹配项（前缀过滤）
const slashOpen = ref(false)    // 菜单是否展开
const slashIndex = ref(0)       // 当前高亮项
const _inRunSlash = ref(false)  // ★ 2026-09-10 命令无匹配降级原样发送时，防止 sendMessage 入口 "/" 拦截递归
// 输入以 "/" 开头时拉取命令清单（首次惰性 + 每次 send 后刷新）
// ★ 2026-08-31 修复：refreshSlashMenu 首次调用时清单为空（异步拉取），
//   同步 filter 恒空 → 菜单永不弹出；拉取完成后需再刷新一次。
async function ensureSlashCommands() {
  if (slashCommands.value.length) return
  try {
    const res = await api.listCommands()
    slashCommands.value = (res && res.commands) || []
  } catch (e) { slashCommands.value = [] }
  // 清单到位后重新评估当前输入的 slash 菜单（若输入仍以 "/" 开头）
  if (slashCommands.value.length && inputText.value.startsWith('/')) {
    refreshSlashMenu()
  }
}
function refreshSlashMenu() {
  const text = inputText.value
  // 仅「/name」前缀（不含空格后的参数）时展开
  if (!text.startsWith('/')) { slashOpen.value = false; return }
  const m = text.match(/^\/(\S*)$/)
  if (!m) { slashOpen.value = false; return }
  ensureSlashCommands()
  const q = m[1].toLowerCase()
  slashMatches.value = slashCommands.value.filter(c => (c.name || '').toLowerCase().startsWith(q)).slice(0, 8)
  slashOpen.value = slashMatches.value.length > 0
  slashIndex.value = 0
}
// 点击菜单项 / 菜单内 Enter：填入 "/name " 并聚焦输入框（不发送，可继续编辑）
function pickSlashCommand(c) {
  setInputText('/' + c.name + ' ')
  slashOpen.value = false
  if (inputRef.value) inputRef.value.focus()
}
// Enter 执行（菜单外）：匹配命令 → runCommand（结果由后端注入系统消息）→ 本地渲染命令结果卡片；
// ★ 2026-09-10 修复「命令被当作普通消息」：命令命中后不再把命令文本发给模型（模型会把
//   "/x" 当普通任务回一轮），改为命令结果直接在对话流展示；命令文本随系统消息落盘，
//   下一轮模型仍能读到命令结果（激活/协议提示同时生效）。
// 无匹配命令 → 原样发送（降级零破坏）。菜单展开时 Enter 已被 onKeydown 拦截为「填入选中项」，不会走到这里。
async function runSlashCommand() {
  const text = inputText.value.trim()
  const m = text.match(/^\/(\S+)\s*(.*)$/)
  slashOpen.value = false
  if (!m) { sendMessageSpecial(); return }
  const name = m[1]
  // ★ 2026-09-10 兜底：清单未加载（首次输入即回车）时先异步拉取再匹配，避免误判「无匹配」原样发送
  if (!slashCommands.value.length) await ensureSlashCommands()
  const cmd = slashCommands.value.find(c => c.name === name)
  if (!cmd) { sendMessageSpecial(); return } // 无匹配 → 原样发送
  try {
    const res = await api.runCommand(name, { args: m[2] || '' }, state.currentConvId)
    // ★ 2026-09-10 命令结果本地渲染（不唤醒模型；后端已同时注入系统消息并激活插件）
    pushSlashResult(name, (res && res.output) || '（命令执行完成，无输出）')
  } catch (e) {
    console.warn('[RP] slash 命令执行失败:', e)
    window.$toast?.('命令执行失败: ' + ((e && e.message) || e), 'error')
    // 失败不原样发送（避免模型把 "/x" 当普通任务），仅提示
  }
}
// 无匹配命令的原样发送（绕开 sendMessage 的 "/" 前缀拦截，防止递归）
async function sendMessageSpecial() {
  _inRunSlash.value = true
  try { await sendMessage() } finally { _inRunSlash.value = false }
}
// 命令结果卡片：追加一条 assistant 角色的命令结果消息（_isSlashResult 标记渲染专用样式）
function pushSlashResult(name, output) {
  const convId = state.currentConvId
  if (!convId) return
  const box = state.messagesByConv[convId]
  if (!box) return
  const msg = {
    role: 'assistant',
    content: output,
    segments: [{ type: 'content', content: output }],
    _isSlashResult: true,
    _slashName: name,
    _key: 'slash_' + Date.now(),
    _idx: box.length,
    _time: '',
    _folded: false,
  }
  box.push(msg)
  state.messages = [...box]
  scrollToBottom()
}

// ─── composer 模型选择器（★ 2026-09-03 配置列表驱动）───
//   ① 数据源 = AI 设置面板「AI 配置」列表（ai-presets.json，PresetManager 维护）：
//      分组=配置名（命名快照：服务商 + Key），每组模型 = 该配置对应服务商
//      （models.json）内的可用模型列表——模型按会话在对话面板中选，不在配置中指定；
//   ② 选择只写当前会话（PUT /api/conversations/{id} {provider, model}）——
//      全局 settings 不动，其他/历史对话的模型不被牵连；
//   ③ 配置对应服务商暂无模型（服务商面板未配）→ 该分组跳过；
//   ④ 新会话（无消息且未设模型）自动继承「上次选择」（localStorage），保持体验连续；
//   ⑤ 会话已有模型时逆映射回配置名分组显示（旧会话/自定义 provider::model 仍兼容）。
const LAST_MODEL_KEY = 'paircode.lastPickedModel'
const modelData = ref(null)          // /api/models + presets 快照
const composerProvider = ref('')     // 当前会话生效的服务商
const composerModel = ref('')        // 下拉值：'preset::<配置名>::<模型>'（配置分组）或旧编码 'provider::model'
// ★ 2026-09-03 配置列表驱动：直接遍历 /api/ai-presets 的配置（= AI 设置面板「AI 配置」列表），
//   分组=配置名，每组模型取该配置 provider 在 models.json 的模型列表。
const composerItems = computed(() => {
  const md = modelData.value || {}
  const presets = md.presets || {}
  const models = md.models || {}
  const names = Object.keys(presets)
  if (!names.length) return []
  // ★ 激活预设（settings.preset）置顶，其余按配置列表原序
  const s = state.settings || {}
  const order = (s.preset && names.includes(s.preset)) ? [s.preset, ...names.filter(n => n !== s.preset)] : names
  const out = []
  for (const n of order) {
    const p = presets[n] || {}
    const list = models[p.provider] || []
    if (!list.length) continue   // 配置对应服务商暂无模型（服务商面板未配）→ 跳过该分组
    out.push({ name: n, provider: p.provider || '', models: list, hasKey: !!p.apiKey })
  }
  return out
})
// ★ 2026-09-05 移动端化：模型/工具集选择器的 bottom-sheet 选项（SheetPicker）
const modelSheetItems = computed(() => {
  const out = []
  for (const g of composerItems.value) {
    for (const m of g.models) {
      // desc 只保留服务商（配置名已作为 optgroup 分组标题展示，避免重复）
      out.push({ value: 'preset::' + g.name + '::' + m, label: m, desc: g.provider, group: g.name })
    }
  }
  return out
})
// 下拉值解析：'preset::<配置名>::<模型>'（配置分组）或旧编码 'provider::model'（历史会话/自定义）
// ★ 2026-09-03 返回 preset（配置名）——切换时一并写入会话元数据，装配按配置整套展开。
function parseModelValue(v) {
  const s = String(v || '')
  if (s.startsWith('preset::')) {
    // ★ 2026-09-04 修复：'preset::' 是 8 字符（6+2 冒号），slice(7) 会残留 1 个冒号
    //   使配置名带污染前缀（':配置名'）→ presets 查不到 → provider 为空 →
    //   onCmpModelChange guard 拦截 → 下拉切换不发请求（实测症状）。slice(8) 修正。
    const rest = s.slice(8)
    const i = rest.indexOf('::')
    const name = i < 0 ? rest : rest.slice(0, i)
    const model = i < 0 ? '' : rest.slice(i + 2)
    const p = ((modelData.value || {}).presets || {})[name]
    if (p) return { preset: name, provider: p.provider || '', model }
    return { preset: name, provider: '', model }
  }
  const i = s.indexOf('::')
  if (i < 0) return { preset: '', provider: '', model: s }
  return { preset: '', provider: s.slice(0, i), model: s.slice(i + 2) }
}
// 会话 provider → 配置名（逆映射）：provider 匹配的配置分组；找不到返回 ''
function presetNameOf(provider, model) {
  const presets = ((modelData.value || {}).presets) || {}
  for (const n of Object.keys(presets)) {
    if ((presets[n] || {}).provider === provider) return n
  }
  return ''
}
function modelValueOf(provider, model) {
  if (!provider && !model) return ''
  const n = presetNameOf(provider, model)
  if (n) return 'preset::' + n + '::' + model
  return String(provider || '') + '::' + String(model || '')
}
// 全局默认（settings/preset）解析出的 服务商+模型：会话未设模型时下拉显示它
function defaultProviderModel() {
  const s = state.settings || {}
  const md = modelData.value || {}
  let prov = s.provider || ''
  let model = s.executeModel || ''
  const presets = md.presets || null // 激活预设（携带 Key），仅解析服务商/模型
  if (presets && s.preset && presets[s.preset]) {
    prov = presets[s.preset].provider || prov
    model = presets[s.preset].executeModel || model
  }
  // 回落：当前配置分组中取首项（模型取该服务商 models 列表首个）
  const items = composerItems.value
  if (!prov && items[0]) prov = items[0].provider
  if (!model) {
    const it = items.find(x => x.provider === prov) || items[0]
    if (it && it.models.length) model = it.models[0]
  }
  return { provider: prov, model }
}
async function loadModelData() {
  try {
    const md = await api.getModels()
    // ★ 2026-09-03 附带 ai-presets（配置列表数据源：下拉按配置展示 + 激活预设解析 provider/model）
    const pr = await api.getAiPresets().catch(() => null)
    md.presets = (pr && pr.presets) || null
    modelData.value = md
  } catch {}
}
// 依据当前会话元数据同步下拉（会话有自己的模型 → 显示它；否则显示全局默认）
// ★ 2026-09-03 会话记录了配置名（meta.preset）→ 直接按配置编码显示（不再反向猜）。
async function syncComposerModelFromConv() {
  // ★ 刷新保险：modelData（/api/models + presets）未就绪时直接跳过——
  //   presets 为空走旧编码 fallback 会把下拉设成无效值（无匹配 option），
  //   显示回退后不再修正。后续 initComposerModel（loadModelData 完成后）或
  //   watch(currentConvId) 会再次同步（彼时数据已就绪）。
  if (!modelData.value || !(modelData.value.presets)) return
  const convId = state.currentConvId
  let prov = '', model = '', preset = ''
  if (convId) {
    try {
      const meta = await api.getConversationMeta(convId, state.workspaceRoot || '')
      prov = (meta && meta.provider) || ''
      model = (meta && meta.model) || ''
      preset = (meta && meta.preset) || ''
    } catch {}
  }
  if (!prov && !model) {
    // 会话未设模型：新会话（无消息）继承上次选择并写入会话，老会话只显示默认不写
    const last = readLastPicked()
    const msgCount = (state.messages && state.messages.length) || 0
    if (convId && last.provider && last.model && msgCount === 0) {
      prov = last.provider; model = last.model; preset = last.preset || ''
      try { await api.setConvModel(convId, prov, model, preset, state.workspaceRoot || '') } catch {}
    } else {
      const d = defaultProviderModel()
      prov = d.provider; model = d.model
    }
  }
  composerProvider.value = prov
  // 会话记录了配置名且该配置仍存在 → 直接按配置编码
  if (preset && ((modelData.value || {}).presets || {})[preset]) {
    composerModel.value = 'preset::' + preset + '::' + model
  } else {
    composerModel.value = modelValueOf(prov, model)
  }
}
function readLastPicked() {
  try {
    const raw = window.localStorage && window.localStorage.getItem(LAST_MODEL_KEY)
    if (!raw) return { provider: '', model: '', preset: '' }
    const o = JSON.parse(raw)
    return { provider: o.provider || '', model: o.model || '', preset: o.preset || '' }
  } catch { return { provider: '', model: '', preset: '' } }
}
function writeLastPicked(provider, model, preset) {
  try { window.localStorage && window.localStorage.setItem(LAST_MODEL_KEY, JSON.stringify({ provider, model, preset: preset || '' })) } catch {}
}
function initComposerModel() {
  syncComposerModelFromConv()
}

// ── ★ 2026-09-04 工具集（通用集合）模式：会话级选择 ──
const toolsetItems = ref([])    // 全局工具集列表（GET /api/toolsets，不含 builtin 虚拟）
const convToolset = ref('')     // 当前会话工具集名（'' = 未设置，后端用 default）
function toolsetLabel(t) {
  const scope = t.scope === 'builtin' ? '内置' : '全局'
  return t.name + '（' + scope + '·' + (t.pluginCount || 0) + ' 插件）'
}
// ★ 2026-09-05 移动端化：工具集选择器 bottom-sheet 选项
const toolsetSheetItems = computed(() =>
  toolsetItems.value.map(t => ({ value: t.name, label: t.name, desc: (t.pluginCount || 0) + ' 个插件' }))
)
async function loadToolsetItems() {
  try {
    const list = await api.getToolsets()
    toolsetItems.value = (list || []).filter(t => t.scope !== 'builtin' && t.name)
  } catch (e) {
    console.warn('[toolset] 工具集列表加载失败', e)
  }
}
// 依据当前会话元数据同步模式选择（会话已记录 toolset → 显示它；否则默认）
async function syncConvToolsetFromConv() {
  const convId = state.currentConvId
  if (!convId) { convToolset.value = ''; return }
  try {
    const meta = await api.getConversationMeta(convId, state.workspaceRoot || '')
    convToolset.value = (meta && meta.toolset) || ''
  } catch { convToolset.value = '' }
}
// 切换模式 = 只写当前会话（不动全局；后端发送消息时按会话集合收敛工具面）
async function onConvToolsetChange() {
  const convId = state.currentConvId
  const name = convToolset.value
  if (!convId) return
  try {
    await api.apiPut('/conversations/' + encodeURIComponent(convId), { toolset: name })
    window.$toast && window.$toast((name ? '本对话已切换工具集为 ' + name : '已清除工具集选择（回落 default）'), 'success')
  } catch (e) {
    window.$toast && window.$toast('工具集切换失败: ' + (e.message || e), 'error')
  }
}
// 切换模型 = 只写当前会话（不动全局 settings；★ 2026-09-03 连同配置名一起写入）
async function onCmpModelChange() {
  const { provider, model, preset } = parseModelValue(composerModel.value)
  if (!provider || !model) return
  const convId = state.currentConvId
  composerProvider.value = provider
  writeLastPicked(provider, model, preset)
  if (!convId) return   // 尚无会话：记住选择，新建会话时写入
  try {
    await api.setConvModel(convId, provider, model, preset, state.workspaceRoot || '')
    window.$toast && window.$toast('本对话已切换为 ' + provider + ' / ' + model, 'success')
  } catch (e) {
    window.$toast && window.$toast('模型切换失败: ' + (e.message || e), 'error')
  }
}

const feedbackText = ref('')
const msgRef = ref(null)
const inputRef = ref(null)
const chatInputAreaRef = ref(null)
const chatToolsEl = ref(null)
let chatToolsUnsub = null
// 按钮已移至 textarea 外部下方（.input-bottom-bar），无需动态 padding
function updateInputPadding() {
  if (inputHeight.value < 80) inputHeight.value = 80
}
const inputHeight = ref(150)
const convListWidth = ref(250)
// ★ wb-ui(goja) workaround：state.conversations 数组整体赋值（loadConvList
//   里 state.conversations = list）触发不了 prop 响应式更新——直接传
//   state.conversations 时 ConvSidebar 拿到的是旧引用/空数组。改用
//   computed 包装：渲染时实时求值 state.conversations（与 FileExplorer 的
//   currentFolders 同一模式），首次渲染即读到预取/加载的数据。
const convList = computed(() => state.conversations)
const topSentinel = ref(null)
const reviewMode = ref('auto')  // 'auto'=AI审核, 'manual'=人工审批, 'off'=全部放行
const reviewBtnTitle = computed(() => {
  const m = reviewMode.value
  return m === 'auto' ? 'AI审核：Agent自行审批写操作' : m === 'manual' ? '手动审批：每次操作需用户确认' : '关闭审核：全部放行，不经过任何审核'
})
const reviewBtnLabel = computed(() => {
  const m = reviewMode.value
  return m === 'auto' ? '审核' : m === 'manual' ? '审批' : '放行'
})
const reviewIconName = computed(() => {
  const m = reviewMode.value
  return m === 'off' ? 'shield-off' : 'shield'
})
const reviewBtnClass = computed(() => {
  const m = reviewMode.value
  if (m === 'auto') return 'obtn-review-auto'
  if (m === 'manual') return 'obtn-review-manual'
  return 'obtn-review-off'
})
function cycleReviewMode() {
  const m = reviewMode.value
  const next = m === 'auto' ? 'manual' : m === 'manual' ? 'off' : 'auto'
  reviewMode.value = next
  // ★ 2026-08-31 会话级：带 convId 写入会话（元数据持久化 + 当前 Loop 实时生效），
  //   不再污染工作区级默认（同工作区其他会话不受影响）。
  api.apiPut('/tools/review?convId=' + encodeURIComponent(state.currentConvId || ''), { reviewMode: next }).catch(() => {
    reviewMode.value = m
  })
}

const autoIterate = ref(false)
const autoCollapse = ref(localStorage.getItem('autoCollapse') !== 'false')
const autonomous = ref(false)
const pendingAttachments = ref([])     // ★ 2026-08-21 多附件 + 光标位置插入：数组化。每个附件带 _token 占位符
let attSeq = 0

// ── ★ 2026-08-21 多模态：文件树/拖拽添加的图片附件（仅路径无内容）──
// 通过 /api/fs/image 读取文件 → base64 dataURL，发送时才可走多模态 images 数组。
const IMG_EXTS = ['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp']
function isImagePath(p) {
  if (!p) return false
  const name = String(p).split(/[\\/]/).pop() || ''
  const dot = name.lastIndexOf('.')
  if (dot < 0) return false
  return IMG_EXTS.includes(name.slice(dot + 1).toLowerCase())
}
async function loadImageData(att) {
  try {
    if (!att.path) throw new Error('无路径')
    const res = await fetch('/api/fs/image?path=' + encodeURIComponent(att.path))
    if (!res.ok) throw new Error('HTTP ' + res.status)
    const blob = await res.blob()
    const dataUrl = await new Promise((resolve, reject) => {
      const fr = new FileReader()
      fr.onload = () => resolve(fr.result)
      fr.onerror = () => reject(new Error('FileReader 失败'))
      fr.readAsDataURL(blob)
    })
    att.content = dataUrl
    att.mimeType = blob.type || 'image/png'
  } catch (e) {
    console.warn('[RP] 图片附件读取失败 path=%s err=%s', att.path, e && e.message || e)
  }
  att._imageReady = true
}

// addAttachment 添加附件：分配唯一 token，渲染为内联 tag 插入输入框光标处（无光标则追加末尾）
function addAttachment(att) {
  // ★ 2026-08-21 多模态：文件是图片 → 统一按图片附件处理（异步读取 base64 内容）
  if (att.type === 'file' && isImagePath(att.path)) {
    att.type = 'image'
    att._imageReady = false
    att._imgPromise = loadImageData(att)
  }
  attSeq++
  att._token = '@@att' + attSeq + '@@'
  pendingAttachments.value.push(att)
  insertTagAtCursor(att)
  inputRef.value?.focus()
}

// ── ★ 2026-08-22 contenteditable 输入框：附件内联 tag（tag 随文字混排，在输入框内）──
// ATT_ICON_SVG：tag 内联图标（feather 风格，与 SvgIcon 同款，stroke currentColor）
const ATT_ICON_SVG = {
  file: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>',
  code: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="10" y1="12" x2="8" y2="14"/><line x1="10" y1="16" x2="8" y2="18"/><line x1="14" y1="12" x2="16" y2="14"/><line x1="14" y1="16" x2="16" y2="18"/>',
  dir: '<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>',
  image: '<rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/>',
}
function attIconKey(att) {
  if (att.type === 'image') return 'image'
  if (att.type === 'selection') return 'code'
  if (att.type === 'dir') return 'dir'
  return 'file'
}

// insertTagAtCursor 在光标处插入附件 tag（contenteditable=false 内联药丸）
function insertTagAtCursor(att) {
  const el = inputRef.value
  if (!el) return
  const kind = attIconKey(att)
  const span = document.createElement('span')
  span.className = 'att-inline att-inline-' + kind
  span.contentEditable = 'false'
  span.dataset.token = att._token || ''
  span.title = '附件（点 × 或 Backspace 移除）'
  const icon = document.createElement('span')
  icon.className = 'att-inline-icon'
  // ★ 2026-08-22 修复图标巨大：动态创建的 svg 无 width/height 且 scoped CSS 不生效
  //   → 默认 300x150 拉伸。内联 width/height 属性 + 全局样式双保险。
  icon.innerHTML = '<svg class="att-inline-svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">' + (ATT_ICON_SVG[kind] || ATT_ICON_SVG.file) + '</svg>'
  const label = document.createElement('span')
  label.className = 'att-inline-label'
  let name = att.path || att.filename || ''
  if (att.lineStart) name += ' L' + att.lineStart + '-' + (att.lineEnd || att.lineStart)
  label.textContent = name
  const x = document.createElement('span')
  x.className = 'att-inline-x'
  x.textContent = '×'
  x.title = '移除'
  x.addEventListener('mousedown', (ev) => ev.preventDefault()) // 防止抢走光标/清选区
  x.addEventListener('click', (ev) => { ev.stopPropagation(); removeAttTagByEl(span) })
  span.appendChild(icon); span.appendChild(label); span.appendChild(x)
  insertNodeAtCaret(span)
  syncInput()
}

// insertNodeAtCaret 把节点插入到当前光标处（输入框无焦点时追加末尾），光标移到节点后
function insertNodeAtCaret(node) {
  const el = inputRef.value
  if (!el) return
  const sel = window.getSelection()
  if (sel && sel.rangeCount > 0 && el.contains(sel.anchorNode)) {
    const range = sel.getRangeAt(0)
    range.deleteContents()
    range.insertNode(node)
    try {
      range.setStartAfter(node)
      range.collapse(true)
      sel.removeAllRanges()
      sel.addRange(range)
    } catch (_) {}
  } else {
    el.appendChild(node)
    // 无焦点追加：光标定位到节点后，避免 focus() 后光标跳到开头
    try {
      const sel2 = window.getSelection()
      const range2 = document.createRange()
      range2.setStartAfter(node)
      range2.collapse(true)
      sel2.removeAllRanges()
      sel2.addRange(range2)
    } catch (_) {}
  }
}

// serializeInput 序列化输入框 DOM → 纯文本（tag span → @@attN@@ token，换行 BR/DIV → \n）
function serializeInput() {
  const el = inputRef.value
  if (!el) return ''
  let out = ''
  for (const node of el.childNodes) {
    if (node.nodeType === Node.TEXT_NODE) {
      out += node.textContent
    } else if (node.nodeType === Node.ELEMENT_NODE) {
      if (node.nodeName === 'BR') {
        out += '\n'
      } else if (node.classList && node.classList.contains('att-inline')) {
        out += node.dataset.token || ''
      } else {
        const block = node.nodeName === 'DIV' || node.nodeName === 'P'
        if (block) out += '\n'
        out += node.textContent || ''
        if (block) out += '\n'
      }
    }
  }
  return out
}

// reconcileAttachments 对账：DOM 中已消失的 tag（选中删除/浏览器默认删除）→ 同步移除 pendingAttachments
function reconcileAttachments() {
  const el = inputRef.value
  if (!el) return
  const domTokens = new Set()
  el.querySelectorAll('.att-inline').forEach(n => { if (n.dataset.token) domTokens.add(n.dataset.token) })
  for (let i = pendingAttachments.value.length - 1; i >= 0; i--) {
    if (!domTokens.has(pendingAttachments.value[i]._token)) pendingAttachments.value.splice(i, 1)
  }
}

// syncInput DOM → inputText（输入后/发送前调用；顺带清理孤立 <br> 保证 :empty placeholder 生效）
function syncInput() {
  const el = inputRef.value
  if (!el) return
  if (el.childNodes.length === 1 && el.firstChild.nodeType === Node.ELEMENT_NODE && el.firstChild.nodeName === 'BR') {
    el.innerHTML = ''
  }
  inputText.value = serializeInput()
  reconcileAttachments()
}

// onInput 输入事件：同步 inputText + 附件对账 + slash 菜单刷新（Round3 ④.2）
function onInput() {
  syncInput()
  refreshSlashMenu()
}

// setInputText 程序化设置输入框内容（继续任务/切换对话/清空）：重建为纯文本节点
function setInputText(text) {
  const el = inputRef.value
  if (!el) { inputText.value = text; return }
  el.innerHTML = ''
  if (text) el.appendChild(document.createTextNode(text))
  inputText.value = text
  reconcileAttachments()
  refreshSlashMenu()
}

// insertTextAtCursor 光标处插入纯文本（多行 → <br> 分段；无焦点追加末尾）
function insertTextAtCursor(text) {
  if (!text) return
  const el = inputRef.value
  if (!el) { inputText.value += text; return }
  const parts = String(text).split('\n')
  const frag = document.createDocumentFragment()
  for (let i = 0; i < parts.length; i++) {
    if (i > 0) frag.appendChild(document.createElement('br'))
    if (parts[i]) frag.appendChild(document.createTextNode(parts[i]))
  }
  const sel = window.getSelection()
  if (sel && sel.rangeCount > 0 && el.contains(sel.anchorNode)) {
    const range = sel.getRangeAt(0)
    range.deleteContents()
    range.insertNode(frag)
    try {
      range.setStartAfter(frag)
      range.collapse(true)
      sel.removeAllRanges()
      sel.addRange(range)
    } catch (_) {}
  } else {
    el.appendChild(frag)
    try {
      const sel2 = window.getSelection()
      const range2 = document.createRange()
      range2.setStartAfter(frag)
      range2.collapse(true)
      sel2.removeAllRanges()
      sel2.addRange(range2)
    } catch (_) {}
  }
  syncInput()
}

// removeAttTagByEl 删除指定 tag 节点（× 点击 / Backspace/Delete 边缘），光标归位到删除位置
function removeAttTagByEl(span) {
  if (!span || !span.parentNode) return
  const el = inputRef.value
  const parent = span.parentNode
  const next = span.nextSibling
  parent.removeChild(span)
  if (el) {
    try {
      const sel = window.getSelection()
      const range = document.createRange()
      if (next) {
        range.setStart(next, 0)
      } else {
        const last = parent.lastChild
        if (last && last.nodeType === Node.TEXT_NODE) range.setStart(last, (last.textContent || '').length)
        else if (last) range.setStartAfter(last)
        else range.setStart(parent, 0)
      }
      range.collapse(true)
      sel.removeAllRanges()
      sel.addRange(range)
    } catch (_) {}
    el.focus()
  }
  const token = span.dataset.token
  if (token) {
    const i = pendingAttachments.value.findIndex(a => a._token === token)
    if (i >= 0) pendingAttachments.value.splice(i, 1)
  }
  syncInput()
}

// removeAttTagByToken 按 token 删除 tag + 附件（移除附件 API 入口）
function removeAttTagByToken(token) {
  if (!token) return
  const el = inputRef.value
  if (el) {
    const span = el.querySelector('.att-inline[data-token="' + token + '"]')
    if (span && span.parentNode) span.parentNode.removeChild(span)
  }
  const i = pendingAttachments.value.findIndex(a => a._token === token)
  if (i >= 0) pendingAttachments.value.splice(i, 1)
  syncInput()
}

// removeAttachment 移除附件（按索引，等价 token 删除）
function removeAttachment(idx) {
  const att = pendingAttachments.value[idx]
  if (att && att._token) removeAttTagByToken(att._token)
  else if (att) pendingAttachments.value.splice(idx, 1)
}

// buildAttachRefText 生成附件的语义化引用文本（发给 LLM 的路径引用）
function buildAttachRefText(att) {
  if (att.type === 'image') {
    return '![图片附件: ' + (att.filename || '图片') + ']'
  }
  if (att.type === 'selection') {
    let t = '\n\n📎 代码引用: `' + att.path + '` L' + (att.lineStart || 1) + '-' + (att.lineEnd || 1)
    if (att.content) t += '\n```\n' + att.content.slice(0, 3000) + '\n```'
    return t
  }
  if (att.type === 'dir') {
    return '\n\n📎 目录: `' + att.path + '`（请用 list_files 查看）'
  }
  return '\n\n📎 附件: `' + (att.path || att.filename) + '`（请用 read_file 读取）'
}

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
// ★ 2026-08-31：plan 体系已移除——currentPlan/planExpanded 下线，任务追踪只用 currentTasks。
const currentTasks = ref([])
const tasksExpanded = ref(false)
const currentPhase = computed(() => state.phaseByConv[state.currentConvId] || '')
const phaseToolCount = computed(() => {
  const msgs = state.messagesByConv[state.currentConvId]
  if (!msgs) return 0
  let count = 0
  for (const m of msgs) {
    if (m.segments) {
      for (const s of m.segments) {
        if (s.type === 'tool_call') count++
      }
    }
  }
  return count
})

// ★ 进度可视化：运行耗时
const agentStart = ref(null)
const phaseElapsed = ref('')
let elapsedTimer = null
watch(() => state.agentRunningByConv[state.currentConvId], (running) => {
  if (running) {
    agentStart.value = Date.now()
    elapsedTimer = setInterval(() => {
      const sec = Math.floor((Date.now() - agentStart.value) / 1000)
      if (sec < 60) phaseElapsed.value = sec + 's'
      else phaseElapsed.value = Math.floor(sec / 60) + 'm ' + (sec % 60) + 's'
    }, 2000)
  } else {
    phaseElapsed.value = ''
    if (elapsedTimer) { clearInterval(elapsedTimer); elapsedTimer = null }
  }
})

// ★ 进度条：基于总耗时估算（长时任务最多 60 分钟）
const phaseProgress = computed(() => {
  if (!agentStart.value || !state.agentRunningByConv[state.currentConvId]) return 0
  const maxSec = 60 * 60 // 60 分钟
  const elapsed = (Date.now() - agentStart.value) / 1000
  return Math.min(Math.round((elapsed / maxSec) * 100), 95) // 高顶 95%
})
let phaseTimer = null

// ── 滚动控制 ──
const scrollTopRef = ref(0)
const isNearBottom = ref(true)
const showScrollDown = ref(false)

// ── 审批状态从全局 state.approvalByConv 读取（仅当前对话）──
const approvalState = computed(() => state.approvalByConv[state.currentConvId] || { callId: '', tool: '', args: '', parsedArgs: {}, waiting: false })

// ★ currentConvInterrupted：当前对话是否"未完成可继续"。
// 两种来源：
//   1. 后端标记：ConversationMeta.interrupted（异常中断/LLM API 错误/用户停止后由 SessionManager 写盘）
//   2. 启发式兜底：进程崩溃、旧版后端等来不及写标记的场景，检查最后一条 assistant 消息
//      是否停在"中间态"——以工具调用或工具结果结尾（无论 result 是否已回填），
//      而不是以最终回答（content 段）收尾
const currentConvInterrupted = computed(() => {
  const convId = state.currentConvId
  if (!convId) return false
  // 运行中的对话不视为中断
  if (state.agentRunningByConv[convId] || state.loadingByConv[convId]) return false
  // 1. 后端标记
  const conv = state.conversations.find(c => c.id === convId)
  if (conv && conv.interrupted) return true
  // 2. 启发式兜底（tool 消息在加载时已过滤，只保留 assistant/user）
  const msgs = state.messagesByConv[convId]
  if (!msgs || msgs.length === 0) return false
  for (let i = msgs.length - 1; i >= 0; i--) {
    const m = msgs[i]
    if (m.role !== 'assistant') continue
    if (m._loading) return false
    const segs = m.segments || []
    if (segs.length === 0) return false
    const lastSeg = segs[segs.length - 1]
    // ① 未返回结果的工具调用 → 中断（进程崩溃、LLM 中途失败）
    if (lastSeg.type === 'tool_call' && !lastSeg.result) return true
    // ② 以工具调用/工具结果结尾（result 已回填，但后续没有最终回答 content 段）→ 中断
    //    （token 额度用完/LLM 报错：agent 在工具结果返回后、生成最终回答前被打断）
    if (lastSeg.type === 'tool_call' || lastSeg.type === 'tool_result') return true
    // ③ 错误结尾 → 中断
    if (lastSeg.type === 'content' && typeof lastSeg.content === 'string'
        && lastSeg.content.includes('**[错误]**')) return true
    // 正常完成：以最终回答（content 段）收尾
    return false
  }
  return false
})
// 一键继续：在输入框填充"继续"指令并直接发送（完全复用 sendMessage 链路）
const continueTask = () => {
  const convId = state.currentConvId
  if (!convId || state.chatLoading) return
  setInputText('请继续完成上次未完成的任务。请先回顾当前上下文中的进度与遗留问题（含执行日志与任务列表），然后继续推进直到任务完成。')
  nextTick(() => { sendMessage() })
}
const hasMoreTop = computed(() => {
  const id = state.currentConvId
  const msgs = state.messagesByConv[id]
  if (!msgs || msgs.length === 0) return false
  if (msgs[0]._noMoreAbove) return false
  // 依据最早已加载消息的 _idx 判断是否还有更早消息（比 msgTotal/Loaded 更可靠）
  const oldestIdx = msgs[0]._idx
  return oldestIdx !== undefined && oldestIdx !== null && oldestIdx > 0
})

// ★ messageCombos：将平铺的 user/assistant 消息按用户消息分组。
// 每组：{ user, assistant }，assistant 可能为 null。
// ★ 保持对原消息对象的引用，WS 流式写入自动反映到 combo 内。
// ★ 强制按 _idx 排序。连续 assistant 消息已在 switchConv/loadMoreMessages 中预处理合并。
// ★ 用户反馈（【用户反馈】前缀）合并到前一个 assistant 气泡，不创建独立用户气泡。
const messageCombos = computed(() => {
  const msgs = [...(state.messages || [])]
    .sort((a, b) => (a._idx ?? 0) - (b._idx ?? 0))
  const combos = []
  let current = null
  let pendingFeedback = null
  for (const msg of msgs) {
    if (msg.role === 'user') {
      // ★ 背景上下文快照：合并进当前组合的 _snapshots（折叠系统信息条），
      //   不创建独立用户气泡（快照是循环同步的消息流背景信息，语义同 dsh
      //   runtime context snapshot）。
      if (isContextSnapshot(msg)) {
        if (current) {
          if (!current._snapshots) current._snapshots = []
          current._snapshots.push(msg)
        } else {
          current = { user: null, assistant: null, _snapshots: [msg] }
          combos.push(current)
        }
        continue
      }
      // ★ 用户反馈消息：合并到前一个 assistant，不创建独立气泡
      if (isFeedback(msg)) {
        pendingFeedback = msg
        continue
      }
      pendingFeedback = null
      current = { user: msg, assistant: null }
      combos.push(current)
    } else if (msg.role === 'assistant') {
      // ★ 如果前一个消息是 feedback user，合并到上一个 combo 的 assistant 中
      if (pendingFeedback && current) {
        if (!current.assistant) {
          // 前一个 combo 没有 assistant → feedback 的回复成为 assistant
          current.assistant = msg
        }
        // 将 feedback 内容和 agent 回复附加到 assistant 的 _feedbacks 数组
        if (!current.assistant._feedbacks) current.assistant._feedbacks = []
        current.assistant._feedbacks.push({
          content: cleanMsgContent(pendingFeedback),
          replyMsg: msg
        })
        pendingFeedback = null
        continue
      }
      if (current && current.assistant === null) {
        // 正常配对：user → assistant
        current.assistant = msg
      } else {
        // 没有前置 user 或已有 assistant → 新起一个独立气泡
        current = { user: null, assistant: msg }
        combos.push(current)
      }
      pendingFeedback = null
    }
  }
  return combos
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
    // 顶部懒加载：scrollTop < 100 且还有更早消息可加载（★ 仅内容满一屏时触发；
    // 不满一屏由 fillViewport 按空间加载，避免每次 scroll 事件重复请求）
    if (el.scrollTop < 100 && el.scrollHeight > el.clientHeight + 10 && !loadingMoreTop.value) {
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
    const data = await api.getMessages(id, { before: oldestIdx, limit: 50, workspaceRoot: state.workspaceRoot })
    const older = (data.messages || [])
      .filter(m => (m.message?.role || m.role) !== 'tool')
      .map((m, i) => ({
        role: m.message?.role || m.role || '',
        content: m.message?.content || m.content || '',
        segments: (m.segments || []).map(seg => {
          if (seg.type === 'ask_user') {
            seg._answered = !!(seg.answer || (seg.answers && seg.answers.length))
            seg.askType = normalizeAskType(seg.askType || 'text')
          }
          return seg
        }),
        _key: 'msg_' + Date.now() + '_older_' + i,
        _idx: m.idx,
        _time: m.timestamp || '',
      }))
    if (older.length > 0) {
      // prepend 到数组头部（全量渲染下 scrollHeight 会自然增加）
      const mergedBefore = mergeConsecutiveAssistant([...older, ...msgs])
      state.messagesByConv[id] = mergedBefore
      state.messages = mergedBefore
      state.msgLoadedByConv[id] = (state.msgLoadedByConv[id] || 0) + older.length
      // ★ 对新增的 older 消息应用折叠默认值（thinking 折叠 / tool_call 折叠 / 完成摘要），
      //   与 switchConv 首次加载的 applyAutoCollapse 行为一致——否则滚动加载的历史消息
      //   因 seg._collapsed===undefined 被模板 `!seg._collapsed` 判为展开，全文/全部工具行
      //   铺开且撑破显示区域（浏览器短对话首次加载即覆盖全部消息无感知，长对话滚动必现）
      applyAutoCollapse()
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

// fillViewport 按空间加载：初始加载内容不足视口时自动加载更早消息填满可视区域
// （浏览器行为：视口高则多加载，而非固定 limit=50 原始行 → 折叠后只显示最后一轮）。
// 依赖引擎几何桥（clientHeight/scrollHeight 真实值）；内容不足时循环 loadMoreMessages
// 直到填满、无更早消息或达到轮次上限（防极端长对话无限循环）。
const fillViewport = async () => {
  if (!autoCollapse.value) return
  const el = msgRef.value
  if (!el) return
  for (let guard = 0; guard < 8; guard++) {
    if (el.scrollHeight > el.clientHeight + 10) break // 已填满视口
    const msgs = state.messagesByConv[state.currentConvId]
    if (!msgs || msgs.length === 0) break
    if (msgs[0]._noMoreAbove) break
    const before = msgs.length
    await loadMoreMessages()
    await nextTick()
    const after = state.messagesByConv[state.currentConvId]
    if (!after || after.length === before) break // 无新消息
  }
  // ★ 仅在用户未主动上翻时滚底（用户在浏览/展开历史时不应被强制拉到底部）
  if (!window.__scrollLockTimer) forceScrollToBottom()
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
      const ts = await api.apiGet('/conversations/' + state.currentConvId + '/token-stats' + (state.workspaceRoot ? '?workspaceRoot=' + encodeURIComponent(state.workspaceRoot) : ''))
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

// mergeConsecutiveAssistant 合并连续的 assistant 消息为一条。
// 历史加载时后端可能返回多条 assistant 消息（各 WS 事件阶段分别持久化），
// 合并后展示在同一个气泡中，避免分段显示。
// ★ 此函数在消息列表设置到 state 之前调用，不在 computed 中做，
//   避免修改响应式对象触发无限循环。
function mergeConsecutiveAssistant(msgs) {
  const result = []
  for (const msg of msgs) {
    if (msg.role === 'assistant' && result.length > 0) {
      const last = result[result.length - 1]
      if (last.role === 'assistant') {
        // 合并 segments 到已有的 assistant 消息
        if (msg.segments && msg.segments.length > 0) {
          if (!last.segments) last.segments = []
          for (const seg of msg.segments) {
            if (seg.type === 'content' && seg.content) {
              // content 段去重：避免重复追加
              const lastContent = [...last.segments].reverse().find(s => s.type === 'content')
              if (lastContent && lastContent.content.endsWith(seg.content)) continue
              last.segments.push(seg)
            } else {
              last.segments.push(seg)
            }
          }
        }
        // 合并 content 回填
        if (msg.content && !last.content) {
          last.content = msg.content
        }
        continue // 跳过此条，不加入 result
      }
    }
    result.push(msg)
  }
  return result
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

// isContextSnapshot 判断用户消息是否为「背景上下文快照」（语义同 dsh 的
// runtime context snapshot）：由循环同步进消息流，展示为折叠的系统信息条，
// 不创建独立用户气泡（合并进前一 assistant 组合的 _snapshots 数组）。
function isContextSnapshot(msg) {
  return msg.role === 'user' && typeof msg.content === 'string'
    && (msg.content.startsWith('【背景上下文·非当前任务】') || msg.content.startsWith('【历史归档】'))
}

// cleanMsgContent 去除消息中的标记前缀和附件尾注，只展示纯内容。
function cleanMsgContent(msg) {
  if (!msg.content) return ''
  return msg.content
    .replace(/^【背景上下文·非当前任务】\n*/, '')
    .replace(/^【任务委派 → \w+】\n*/, '')
    .replace(/^【用户反馈】/, '')
    .replace(/\n*📎 附件: .+/s, '')
    .replace(/\n*📎 代码引用: .+/s, '')
    .replace(/\n*📎 目录: .+/s, '')
    .replace(/\n*!\[图片\].+/s, '')
}

// hasUnansweredAsk 消息中是否存在未回答的 ask_user（这些消息不能被折叠，否则问题不可见）
function hasUnansweredAsk(msg) {
  return !!msg.segments && msg.segments.some(s => s.type === 'ask_user' && !s._answered)
}

function collapsePreviousOutputs() {
  if (!autoCollapse.value) return
  for (const msg of state.messages) {
    if (msg.role !== 'assistant' || msg._loading) continue
    if (!msg.segments || msg.segments.length === 0) continue
    // ★ 有未回答 ask_user 的消息保持展开（否则问题被折叠隐藏无法回答）
    if (hasUnansweredAsk(msg)) continue
    // ★ 保留用户交互状态：已手动展开的气泡（_folded===false）不再强制折叠，
    //   段级展开同样保留——否则「新消息发出时」用户刚展开的旧输出被折叠，
    //   表现为「展开折叠后 滚动/发送新消息 又被自动折叠」
    if (msg._folded === false) continue
    for (const seg of msg.segments) {
      if (seg.type === 'thinking' && seg._collapsed !== false) seg._collapsed = true
      if (seg.type === 'tool_call' && seg._expanded !== true) seg._expanded = false
    }
    msg._folded = true
  }
}

// applyAutoCollapse 页面刷新后对已加载的历史消息应用折叠状态。
// 在 switchConv 加载完消息后调用，确保折叠开关开启时历史消息正确折叠。
// ★ 2026-08-31 支持指定数组（断线重连 reload 非当前会话消息时同样注入折叠标记）。
function applyAutoCollapse(targetArr) {
  const arr = targetArr || state.messages
  if (!autoCollapse.value) return
  for (const msg of arr) {
    if (msg.role !== 'assistant' || msg._loading) continue
    if (!msg.segments || msg.segments.length === 0) continue
    // ★ 有未回答 ask_user 的消息保持展开
    if (hasUnansweredAsk(msg)) continue
    for (const seg of msg.segments) {
      if (seg.type === 'thinking' && seg._collapsed === undefined) seg._collapsed = true
      if (seg.type === 'tool_call' && seg._expanded === undefined) seg._expanded = false
    }
    if (msg._folded === undefined) msg._folded = true
  }
}

const sendMessage = async () => {
  // ★ 2026-08-22 contenteditable：发送前从 DOM 序列化（tag → token），保证 inputText 最新
  syncInput()
  const text = inputText.value.trim()
  if (!text && pendingAttachments.value.length === 0) return
  // ★ 2026-09-10 修复「/agent-teams 被当作普通消息」：以 "/" 开头的文本（点发送按钮、
  //   非 Enter 路径）一律先走 slash 命令识别——命中执行命令、未命中才原样发送。
  //   _inRunSlash 为「无匹配降级」（sendMessageSpecial）时的递归防护。
  if (!_inRunSlash.value && text.startsWith('/')) { runSlashCommand(); return }
  if (state.chatLoading) { console.log('[RP] sendMessage 跳过: chatLoading 已为 true'); return }

  // ★ 确保 WS 连接就绪（等待最多 3s，避免事件丢失）
  const wsReady = await api.waitForWebSocket(3000)
  if (!wsReady) {
    window.$toast?.('连接未就绪，请稍后重试', 'warning')
    console.warn('[RP] sendMessage WS 未就绪')
    return
  }

  // ★ 清理当前对话中所有旧的 loading 占位（防止 processStatus 创建的残留）
  if (state.messagesByConv[state.currentConvId]) {
    const msgs = state.messagesByConv[state.currentConvId]
    const cleaned = msgs.filter(m => !m._loading)
    if (cleaned.length < msgs.length) {
      console.log('[RP] sendMessage 清理旧 loading 占位: %d 个', msgs.length - cleaned.length)
      state.messagesByConv[state.currentConvId] = cleaned
      state.messages = [...cleaned]
    }
  }

  // ★ 立即锁定 loading 状态
  state.chatLoading = true; state.agentRunning = true

  // ★ 确保 convId 存在（在创建用户消息前完成，避免 await 间隙状态变化）
  if (!state.currentConvId) {
    try {
      const conv = await api.apiPost('/conversations', { title: '新对话' })
      state.currentConvId = conv.id
      state.conversations.unshift({ id: conv.id, title: conv.title, msgCount: 0, createdAt: conv.createdAt, updatedAt: conv.updatedAt })
      resetConvCtxStats(conv.id)
    } catch {}
  }
  const convId = state.currentConvId
  if (!convId) { state.chatLoading = false; state.agentRunning = false; return }

  // ── ★ 先创建 runtime（在 push 任何消息之前），防止 processStatus 竞态创建兜底占位 ──
  const msgKey = makeMsgKey()
  const lastUserText = text
  startConvRuntime(convId, msgKey, lastUserText)
  console.log('[RP] sendMessage ▸ 用户发送 conv=%s msgKey=%s textLen=%d', convId, msgKey, lastUserText.length)

  // ── ★ 创建用户消息（★ 2026-08-21 附件 token 原位替换为语义化引用 → 跟随文字位置） ──
  const rawText = inputText.value
  let fullContent = lastUserText
  const attachments = [] // 结构化的附件列表（用于前端标签渲染）
  const images = []      // ★ 2026-08-21 多模态：结构化图片数组（{data,mimeType,detail}），随 chatStart 发送
  const pendings = [...pendingAttachments.value]
  if (pendings.length > 0) {
    for (const att of pendings) {
      // token 在文本中 → 原位替换为语义化引用（跟随用户光标位置）；已删 token → 末尾兜底
      const refText = buildAttachRefText(att)
      const tok = att._token || ''
      if (tok && rawText.includes(tok)) {
        fullContent = fullContent.split(tok).join(refText)
      } else {
        fullContent += refText
      }
      // 结构化附件（气泡标签渲染）
      if (att.type === 'image') {
        // ★ 2026-08-21 多模态：图片不再内联 markdown，改为结构化 images 数组发送
        //   （后端转 Message.Images → Provider 以 OpenAI content 块数组发送）
        //   文件树添加的图片（仅路径）先等待 /api/fs/image 异步读取完成。
        if (att._imgPromise) await att._imgPromise.catch(() => {})
        if (att._imageReady && att.content) {
          const mime = att.mimeType || (att.content || '').match(/^data:([^;,]+)/)?.[1] || 'image/png'
          images.push({ data: att.content, mimeType: mime, detail: 'high' })
          attachments.push({ type: 'image', path: att.filename || '', label: att.filename || '图片', data: att.content })
        } else {
          // 读取失败 → 降级为文本附件（旧行为，提示模型用 read_file）
          attachments.push({ type: 'image', path: att.path || att.filename || '', filename: att.filename || '',
            label: att.filename || att.path.split(/[\\/]/).pop() || '图片' })
        }
      } else if (att.type === 'selection') {
        attachments.push({ type: 'code', path: att.path, lineStart: att.lineStart, lineEnd: att.lineEnd,
          label: (att.filename || att.path) + ':' + (att.lineStart || 1) + '-' + (att.lineEnd || 1) })
      } else if (att.type === 'dir') {
        attachments.push({ type: 'dir', path: att.path, label: att.filename || att.path.split(/[\\/]/).pop() || att.path })
      } else {
        attachments.push({ type: att.type, path: att.path, filename: att.filename,
          label: att.filename || att.path.split(/[\\/]/).pop() || att.path })
      }
    }
  }
  setInputText(''); pendingAttachments.value = []
  collapsePreviousOutputs()

  if (!state.messagesByConv[convId]) state.messagesByConv[convId] = []
  // ★ 计算新消息 _idx：取当前最大 _idx + 1，而非数组长度（历史消息 _idx 来自数据库，可能远大于数组长度）
  let nextIdx = state.messagesByConv[convId].length
  for (const m of state.messagesByConv[convId]) { if ((m._idx ?? 0) >= nextIdx) nextIdx = (m._idx ?? 0) + 1 }
  const userMsg = {
    role: 'user', content: fullContent, segments: [], toolCalls: [],
    _attachments: attachments.length > 0 ? attachments : undefined,
    _key: makeMsgKey(), _idx: nextIdx,
    _time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
  }
  state.messagesByConv[convId].push(userMsg)
  console.log('[RP] sendMessage ▸ 用户消息已推入 idx=%d', userMsg._idx)

  // ★ 创建 assistant 占位并用预生成的 msgKey（runtime 已绑定此 key）
  createAssistantPlaceholder(convId, msgKey)
  // ★ 同步响应式引用（仅一次；runtime 已存在，processStatus 不会创建兜底占位）
  state.messages = [...state.messagesByConv[convId]]

  // 更新对话标题和计数
  autoNameConv(convId, lastUserText || fullContent)
  const localConv = state.conversations.find(c => c.id === convId)
  if (localConv) localConv.msgCount = (localConv.msgCount || 0) + 1

  state.loadingByConv[convId] = true
  state.agentRunningByConv[convId] = true
  if (state.workspaceRoot) {
    state.runningByWorkspace = {
      ...state.runningByWorkspace,
      [state.workspaceRoot]: (state.runningByWorkspace[state.workspaceRoot] || 0) + 1,
    }
  }
  if (!state.chatSessionId) state.chatSessionId = 'sess_' + Date.now()
  console.log('[RP] sendMessage ▸ 调用 chatStart conv=%s msgsLen=%d', convId, state.messagesByConv[convId].length)

  // ── 调用后端 chatStart（HTTP 发送，WS 事件异步更新 assistant 消息） ──
  try {
    await api.chatStart(convId, fullContent, autonomous.value, state.workspaceRoot, images)
  } catch (err) {
    const msgs0 = state.messagesByConv[convId]
    if (msgs0) {
      const m = msgs0.find(x => x._key === msgKey)
      if (m) { m._loading = false; pushSegment(m.segments, 'content').content += '**[启动失败]** ' + (err.message || err) }
    }
    state.loadingByConv[convId] = false
    state.agentRunningByConv[convId] = false
    state.chatLoading = false; state.agentRunning = false
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
}
// 压缩按钮已移除；上下文压缩由 agent 自主管理

const stopChat = async () => {
  const convId = state.currentConvId
  console.log('[RP] stopChat conv=%s runtimeExists=%s', convId, !!getConvRuntime(convId))
  if (!convId) return
  try { await api.chatStop(convId) } catch {}
  // ★ 在清理 runtime 前保存 msgKey，用于清理 messagesByConv 中残留的 loading 占位
  const rt = getConvRuntime(convId)
  const oldMsgKey = rt ? rt.msgKey : ''
  resetConvRuntime(convId)
  // ★ 从 messagesByConv 中移除残留的 loading 占位（防止停止后 tool_result 写入导致残留）
  if (oldMsgKey && state.messagesByConv[convId]) {
    const msgs = state.messagesByConv[convId]
    const idx = msgs.findIndex(m => m._key === oldMsgKey && m._loading)
    if (idx >= 0) {
      msgs.splice(idx, 1)
      // 同步 state.messages 如果当前对话
      if (state.currentConvId === convId) {
        state.messages = msgs
      }
      console.log('[RP] stopChat 已移除旧 loading 占位 idx=%d', idx)
    }
  }
  state.loadingByConv[convId] = false
  state.agentRunningByConv[convId] = false
  state.chatLoading = false; state.agentRunning = false
  console.log('[RP] stopChat 完成 conv=%s messagesByConv=%d条', convId, (state.messagesByConv[convId]||[]).length)
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

const onAskAnswer = (seg, { callId, answer, answers }) => {
  if (answers && answers.length) {
    seg.answers = answers
  } else if (answer) {
    seg.answer = answer
  } else {
    return
  }
  submitAskAnswer(seg)
}

const submitAskAnswer = async (seg) => {
  // ★ Round3 ⑤：多问题 answers 数组优先，缺省回落单问题 answer（后端双兼容）
  if (seg.answers && seg.answers.length) {
    seg._answered = true
    try {
      await api.apiPost('/chat/answer', { convId: state.currentConvId, callId: seg.callId, answers: seg.answers })
    } catch {}
    return
  }
  const answer = (seg.answer || '').trim()
  if (!answer) return; seg._answered = true
  try { await api.apiPost('/chat/answer', { convId: state.currentConvId, answer }) } catch {}
}

const resolveApproval = async (approved) => {
  // 兼容两种调用：直接传 boolean（旧）或 { approved, reply }（新）
  const isApproved = typeof approved === 'object' ? approved.approved : approved
  const reply = typeof approved === 'object' ? (approved.reply || '') : ''
  const convId = state.currentConvId
  const a = state.approvalByConv[convId]
  if (!a || !a.callId || !a.waiting) return
  a.waiting = false
  try { await api.apiPost('/chat/approve', { convId, approved: isApproved, reply }) } catch { a.waiting = true }
}

// ── 回退按钮 ──
const rollbackTo = async (msgIdx) => {
  const convId = state.currentConvId
  if (!convId) return
  const ok = await window.$confirm?.(`确定回退到此消息？\n\n将恢复该消息之前的文件状态，并删除此消息之后的所有对话。此操作不可撤销。`, '回退确认', '确定回退', '取消')
  if (!ok) return
  try {
    await api.chatRollback(convId, msgIdx)
    window.$toast?.('已回退到消息 ' + (msgIdx + 1), 'success')
    // 强制重新加载对话
    state.messagesByConv[convId] = state.messagesByConv[convId].slice(0, msgIdx + 1)
    state.messages = state.messagesByConv[convId]
    // 更新对话的 msgCount
    const localConv = state.conversations.find(c => c.id === convId)
    if (localConv) localConv.msgCount = state.messages.length
    state.chatLoading = false
    state.agentRunning = false
    state.loadingByConv[convId] = false
    state.agentRunningByConv[convId] = false
    nextTick(() => scrollToBottom())
  } catch (err) {
    window.$toast?.('回退失败: ' + err.message, 'error')
  }
}

// ── ★ 2026-08-22 contenteditable 输入框：Enter 发送（IME 确认不拦截）+ Backspace/Delete 处理 tag 边缘删除 ──
// handleTagEdgeDelete Backspace/Delete 在 tag 相邻边缘时删除整个 tag（否则浏览器半删除/光标穿墙）
function handleTagEdgeDelete(e) {
  const el = inputRef.value
  if (!el) return
  const sel = window.getSelection()
  if (!sel || sel.rangeCount === 0) return
  const range = sel.getRangeAt(0)
  if (!range.collapsed) return // 非折叠选区：可能选中 tag，交给默认行为 + input 对账
  const back = e.key === 'Backspace'
  let target = null
  if (range.startContainer.nodeType === Node.TEXT_NODE) {
    const tn = range.startContainer
    const len = (tn.textContent || '').length
    if (back && range.startOffset > 0) return
    if (!back && range.startOffset < len) return
    target = back ? tn.previousSibling : tn.nextSibling
  } else {
    const children = range.startContainer.childNodes
    const idx = range.startOffset
    target = back ? children[idx - 1] : children[idx]
  }
  if (target && target.classList && target.classList.contains('att-inline')) {
    e.preventDefault()
    removeAttTagByEl(target)
  }
}
const onKeydown = (e) => {
  if (e.isComposing || e.keyCode === 229) return // IME 组合中不处理
  // slash 菜单导航（Round3 ④.2）
  if (slashOpen.value) {
    if (e.key === 'ArrowDown') { e.preventDefault(); slashIndex.value = (slashIndex.value + 1) % slashMatches.value.length; return }
    if (e.key === 'ArrowUp') { e.preventDefault(); slashIndex.value = (slashIndex.value - 1 + slashMatches.value.length) % slashMatches.value.length; return }
    if (e.key === 'Escape') { slashOpen.value = false; return }
  }
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    if (state.chatLoading) return
    // ★ 2026-09-10 常规键盘交互：菜单展开时 Enter = 把当前高亮命令「写入输入框」
    //   （同鼠标点击 pickSlashCommand：填入 "/name " + 关闭菜单 + 聚焦），不直接执行；
    //   再次 Enter（菜单已关，文本带尾随空格不再匹配前缀）才执行命令发送。
    if (slashOpen.value) {
      const c = slashMatches.value[slashIndex.value]
      if (c) { pickSlashCommand(c); return }
    }
    // ★ 2026-09-10 slash 命令兑底：文本以 "/" 开头时也走命令路径（含菜单未展开/带参数/
    //   清单未加载场景）；无匹配命令时 runSlashCommand 内部降级原样发送（零破坏）。
    if (inputText.value.trim().startsWith('/')) { runSlashCommand(); return }
    sendMessage()
    return
  }
  if (e.key === 'Backspace' || e.key === 'Delete') handleTagEdgeDelete(e)
}
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
    // ★ 2026-08-21 修复"刷新后不自动选对话"：列表加载后若无当前对话（或当前对话已被删），
    //   自动选中最近更新的对话（后端按 UpdatedAt 倒序 → 取第一个）。
    //   currentConvId 赋值即触发 watch → switchConv 加载消息。
    if (state.conversations.length > 0) {
      const cur = state.currentConvId
      if (!cur || !state.conversations.some(c => c.id === cur)) {
        state.currentConvId = state.conversations[0].id
      }
    }
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
  currentTasks.value = []
  setInputText('')
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

// ── 对话切换：纯历史消息加载，不管理运行时/不创建占位/不挂起 WS
//   WS 事件通过 processAgentEvent 直接写入 messagesByConv[convId]，
//   不受 switchConv 影响（多对话并行场景下各 conv 独立更新）。
//   页面刷新后的初始加载由 watch(currentConvId) 触发。
const _loadingConvs = new Set()

// apiLoadAndBuildConv 从服务端拉取最新消息并转换为渲染格式。
// switchConv 首次加载与断线重连 reload 共用同一转换管道（含 finish_task 归一、
// ask_user 标记、多模态附件映射、连续 assistant 合并）。
async function apiLoadAndBuildConv(convId) {
  try {
    const data = await api.getMessages(convId, { limit: 50, workspaceRoot: state.workspaceRoot })
    const loaded = (data.messages || [])
      .filter(m => (m.message?.role || m.role) !== 'tool')
      .map((m, i) => {
        const role = m.message?.role || m.role || ''
        const segments = (m.segments || []).map(seg => {
          if (seg.type === 'tool_call' && seg.name === 'finish_task') {
            return { type: 'content', content: seg.result || '' }
          }
          if (seg.type === 'ask_user') {
            seg._answered = !!(seg.answer || (seg.answers && seg.answers.length))
            seg.askType = normalizeAskType(seg.askType || 'text')
          }
          return seg
        })
        return {
          role, content: m.message?.content || m.content || '', segments,
          _key: 'msg_' + Date.now() + '_' + i, _idx: m.idx, _time: m.timestamp || '',
          // ★ 2026-08-21 多模态：历史消息带图片时映射为附件缩略图（dataURL 直显）
          _attachments: (m.message?.images || m.images || []).map(img => ({
            type: 'image', label: '图片', data: img.data || '', path: img.mimeType || '',
          })),
        }
      })
      .sort((a, b) => (a._idx || 0) - (b._idx || 0))
    // ★ 合并：API 返回的消息直接使用（processStatus 不再创建 loading 占位，无需保留逻辑）
    return { mergedMsgs: mergeConsecutiveAssistant(loaded), total: data.total || loaded.length }
  } catch (e) {
    console.warn('[RP] apiLoadAndBuildConv 失败 conv=%s err=%v', convId, e)
    return null
  }
}

// reloadConvMessages 断线重连同步：重载已完成会话的最新消息（agent-events processStatus 调用）。
// ★ 运行中的会话不重载（snapshot 占位的流式增量由 WS 实时补），只处理已结束的。
const reloadConvMessages = async (convId) => {
  if (!convId) return
  if (state.agentRunningByConv[convId] || state.loadingByConv[convId]) {
    console.log('[RP] reload 跳过运行中会话', convId)
    return
  }
  const res = await apiLoadAndBuildConv(convId)
  if (!res) return
  const { mergedMsgs, total } = res
  // ★ 折叠注入：非当前会话同样处理（否则切回时全部展开，与 loadMoreMessages 同类问题）
  applyAutoCollapse(mergedMsgs)
  state.messagesByConv[convId] = mergedMsgs
  state.msgTotalByConv[convId] = total
  state.msgLoadedByConv[convId] = mergedMsgs.length
  // ★ 刷新门控：reload 也算历史加载完成（flush 门控期间的 WS 事件）
  markHistoryLoaded(convId)
  if (state.currentConvId === convId) {
    state.messages = mergedMsgs
    state.chatLoading = false
    state.agentRunning = false
    forceScrollToBottom()
  }
  console.log('[RP] reloadConvMessages conv=%s loaded=%d total=%d', convId, mergedMsgs.length, total)
}

// refreshConvMeta 断线重连后刷新会话列表元数据（interrupted/updatedAt/msgCount 以服务端为准）。
// ★ 只按 id 合并原生字段，不重排列表、不切换当前会话、不新建对话。
const refreshConvMeta = async () => {
  try {
    const list = await api.apiGet('/conversations', { workspace: state.workspaceRoot })
    if (!Array.isArray(list)) return
    for (const m of list) {
      const local = state.conversations.find(c => c.id === m.id)
      if (local) {
        // 仅合并权威标量字段（保持本地引用与顺序稳定）
        local.interrupted = !!m.interrupted
        local.updatedAt = m.updatedAt || local.updatedAt
        if (m.msgCount !== undefined) local.msgCount = m.msgCount
        if (m.title) local.title = m.title
      } else {
        state.conversations.push(m)
      }
    }
  } catch (e) {
    console.warn('[RP] refreshConvMeta 失败:', e)
  }
}
const switchConv = async (id) => {
  if (!id || _loadingConvs.has(id)) return
  _loadingConvs.add(id)
  try {
  console.log('[RP] switchConv id=%s messagesByConvLen=%d', id, (state.messagesByConv[id]||[]).length)
  state.currentConvId = id
  if (!state.messagesByConv[id]) state.messagesByConv[id] = []
  state.messages = state.messagesByConv[id]
  state.chatLoading = state.loadingByConv[id] || false
  state.agentRunning = state.agentRunningByConv[id] || false
  currentTasks.value = []

  // 加载 token 统计
  try {
    const ts = await api.apiGet('/conversations/' + id + '/token-stats' + (state.workspaceRoot ? '?workspaceRoot=' + encodeURIComponent(state.workspaceRoot) : ''))
    if (ts && ts.promptTokens !== undefined) Object.assign(getConvCtxStats(id), ts)
  } catch {}

  // 若本地无缓存消息，从 API 加载
  const msgs = state.messagesByConv[id]
  const hasRealMsgs = msgs.length > 0 && msgs.some(m => !m._loading)
  if (!hasRealMsgs) {
    const res = await apiLoadAndBuildConv(id)
    if (res) {
      const { mergedMsgs, total } = res
      console.log('[RP] switchConv API返回 loaded=%d total=%d', mergedMsgs.length, total)
      state.messagesByConv[id] = mergedMsgs
      state.messages = mergedMsgs
      state.msgTotalByConv[id] = total
      state.msgLoadedByConv[id] = mergedMsgs.length

      // ★ 若该对话正在运行（agentRunningByConv 已由 processStatus 设置），
      //   创建 assistant 占位 + runtime，准备接收 WS 实时事件
      if (state.agentRunningByConv[id] && !getConvRuntime(id)) {
        const key = makeMsgKey()
        console.log('[RP] switchConv 对话运行中，创建占位 conv=%s key=%s', id, key)
        startConvRuntime(id, key, '')
        let siNextIdx = mergedMsgs.length
        for (const m of mergedMsgs) { if ((m._idx ?? 0) >= siNextIdx) siNextIdx = (m._idx ?? 0) + 1 }
        mergedMsgs.push({
          role: 'assistant', content: '', segments: [], toolCalls: [],
          _key: key, _idx: siNextIdx,
          _time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
          _loading: true,
        })
        state.messagesByConv[id] = mergedMsgs
        state.messages = mergedMsgs
      }
      // ★ 刷新门控：历史加载完成 → flush 门控期间的 WS 快照/事件（快照重建
      //   当前回合、事件续接增量），否则快照/流式先到会挤占历史加载。
      markHistoryLoaded(id)
    } else {
      state.msgTotalByConv[id] = 0
      state.msgLoadedByConv[id] = 0
    }
  }

  // 加载任务状态
  try {
    const taskData = await api.apiGet('/tasks', { convId: id })
    if (taskData && taskData.tasks && taskData.tasks.length > 0) {
      currentTasks.value = taskData.tasks.map(t => ({
        step: t.step || t.subject || '', status: t.status, _taskId: t.taskId,
      }))
    } else { currentTasks.value = [] }
  } catch { currentTasks.value = [] }

  // ★ 2026-08-31：plan 体系已移除，不再从消息重建计划（currentPlan 下线）。
  applyAutoCollapse()
  // ★ 按空间加载：初始内容不足视口时自动加载更早消息（浏览器行为），
  //   引擎几何桥修复后 clientHeight/scrollHeight 为真实值，fillViewport 可判断。
  //   ★ 2026-08-22 性能修复：不再 await——fillViewport 可能串行多轮大请求，
  //     阻塞任务状态/计划加载与首屏显示（表现为「消息加载慢」）；
  //     改为后台执行，先完成首屏滚底，fillViewport 完成后按锁定状态滚底。
  fillViewport()
  forceScrollToBottom()
  } finally {
    _loadingConvs.delete(id)
  }
}

const toggleAuto = async (field) => {
  const oldVal = !!state.settings[field]
  const newVal = !oldVal
  state.settings[field] = newVal
  // 同步 local ref（浅 watch 不触发，需要手动同步）
  if (field === 'autonomous') autonomous.value = newVal
  else if (field === 'autoCollapse') { autoCollapse.value = newVal; localStorage.setItem('autoCollapse', newVal) }
  try { await api.apiPut('/settings?convId=' + encodeURIComponent(state.currentConvId), state.settings) } catch { state.settings[field] = oldVal; if (field === 'autonomous') autonomous.value = oldVal; else if (field === 'autoCollapse') autoCollapse.value = oldVal }
}

const autoNameConv = async (convId, content) => {
  if (!convId || !content) return
  // ★ 只设置一次标题：若已有非默认标题，不再覆盖
  const existing = state.conversations.find(c => c.id === convId)
  if (existing && existing.title && existing.title !== '新对话' && existing.title !== '') {
    return
  }
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
  const taskTools = ['task_create', 'task_update', 'task_list', 'task_delete', 'task_summary']
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
  inputDragging = true; inputStartY = e.clientY
  // 用实际渲染高度（考虑 CSS min-height），避免跳变
  inputStartH = inputRef.value?.offsetHeight || inputHeight.value
  document.addEventListener('mousemove', onInputResizeMove); document.addEventListener('mouseup', stopInputResize)
}
const onInputResizeMove = (e) => {
  if (!inputDragging) return
  const newH = inputStartH + (inputStartY - e.clientY)
  // min-height + 拖拽方向保护
  inputHeight.value = Math.max(80, Math.min(600, newH))
}
const stopInputResize = () => { inputDragging = false; document.removeEventListener('mousemove', onInputResizeMove); document.removeEventListener('mouseup', stopInputResize); nextTick(() => updateInputPadding()) }

// ── 文件拖拽/粘贴 ──
const handleDrop = (e) => {
  e.preventDefault()
  // 优先检查工作区文件路径（文件树拖拽携带的路径）
  const wsPath = e.dataTransfer?.getData('application/x-file-path') || e.dataTransfer?.getData('text/x-file-path') || ''
  if (wsPath) {
    addAttachment({ type: 'file', path: wsPath, filename: wsPath.split(/[\\/]/).pop() })
    return
  }
  // 外部文件（浏览器文件系统）—— 不在工作区内，提示用户
  const files = e.dataTransfer?.files
  if (files && files.length > 0) {
    // 外部文件无法获得工作区相对路径，agent 无法 read_file，提示用户
    window.$toast && window.$toast('该文件不在工作区内，请先添加到工作区或从文件树拖入', 'warn')
    return
  }
  // 纯文本拖拽 —— 光标处插入文本
  const textData = e.dataTransfer?.getData('text/plain')
  if (textData) { insertTextAtCursor(textData); inputRef.value?.focus() }
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
        reader.onload = (ev) => { addAttachment({ type: 'image', path: file.name, filename: file.name, content: ev.target?.result || '' }) }
        reader.readAsDataURL(file)
      } else {
        // 非图片文件 —— 不读取内容，提示从编辑器或文件树拖入
        window.$toast && window.$toast('粘贴文件不支持，请从文件树拖入或从编辑器选中代码后拖入', 'warn')
      }
      return
    }
  }
  // ── 无文件粘贴：检测长文本 → 自动转为临时附件；短文本 → 光标处插入纯文本（防富文本） ──
  const plainText = e.clipboardData.getData('text/plain')
  if (plainText && plainText.length > 2000) {
    e.preventDefault()
    createTempAttachment(plainText)
  } else if (plainText) {
    e.preventDefault()
    insertTextAtCursor(plainText)
    inputRef.value?.focus()
  }
}

// ── 粘贴长文本：自动创建临时文件并设为附件 ──
async function createTempAttachment(text) {
  const ts = new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')
  const tempPath = (state.workspaceRoot || '') + '\\_temp'
  try {
    // 确保目录存在（用 mkdir，后端幂等处理）
    await api.apiPost('/fs/mkdir', { path: tempPath })
  } catch (_) { /* 目录可能已存在 */ }
  const filePath = tempPath + '\\paste_' + ts + '.txt'
  try {
    await api.apiPost('/fs/write', { path: filePath, content: text })
    addAttachment({ type: 'file', path: filePath, filename: 'paste_' + ts + '.txt' })
    window.$toast('长文本已保存为临时附件: ' + filePath, 'info')
  } catch (err) {
    window.$toast('创建临时文件失败: ' + err.message, 'error')
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

// ★ watch 兜底：App.vue async onMounted 设置 currentConvId 后自动加载消息
//   _loadingConvs 防重入：sidebar 直接调用时 watch 自动跳过
// ★ 2026-08-22 修复初始化竞态：ui-right-panel 插件挂载（watch 注册）晚于
//   initAppGlobals 异步设置 currentConvId → watch 永不触发 → 历史消息不加载。
//   加 immediate：挂载时若 currentConvId 已存在则立即 switchConv 加载。
watch(() => state.currentConvId, (id, oldId) => {
  if (id && id !== oldId) switchConv(id)
  nextTick(() => startContentResizeObserver())
}, { immediate: true })

  // ── 对话消息全量替换（如首次加载/切换）时也重启观察器
  watch(() => state.messages, () => {
    nextTick(() => startContentResizeObserver())
  }, { deep: false })

  // ── 检测 WS 连接恢复后，对当前对话重新拉取消息（修复断连期间消息丢失）
  window.addEventListener('ws-connection-change', (e) => {
    if (e.detail?.connected && state.currentConvId) {
      const id = state.currentConvId
      const msgs = state.messagesByConv[id]
      // 断连重连后，如果消息数量和 API 返回不匹配，触发 reload
      // 但只在用户没有正在发送消息时执行（avoid conflict with sendMessage）
      if (!state.chatLoading && msgs && msgs.length > 0) {
        // 检查最后一个消息是否有 loading 占位在等待
        const lastLoading = msgs.find(m => m._loading)
        if (lastLoading) {
          // 有 loading 占位但重连了——后端 agent 可能已丢失状态，清除 loading
          console.log('[RP] WS 重连后清除 dangling loading conv=%s key=%s', id, lastLoading._key)
          const idx = msgs.indexOf(lastLoading)
          if (idx >= 0) msgs.splice(idx, 1)
          state.messages = msgs
          state.loadingByConv[id] = false
          state.agentRunningByConv[id] = false
          state.chatLoading = false
          state.agentRunning = false
          resetConvRuntime(id)
        }
      }
    }
  })

watch(() => state.settings, (s) => { if (s) { autoIterate.value = !!s.autoIterateOnRejection; autonomous.value = !!s.autonomous; autoCollapse.value = s.autoCollapse !== undefined ? !!s.autoCollapse : true; } }, { immediate: true })

// ★ 2026-08-31 会话级模型：下拉不再跟随全局 settings（切模型只写会话）。
//   会话切换/新建时按会话元数据同步下拉；服务商配置（Key/模型列表）变化时刷新分组。
watch(() => state.currentConvId, () => { syncComposerModelFromConv() })
watch(() => [state.settings && state.settings.provider, state.settings && state.settings.executeModel], () => {
  // 全局默认配置变化：仅当当前会话未设置模型时下拉才需要刷新显示
  loadModelData().then(() => syncComposerModelFromConv())
})
// ★ 2026-09-04 工具集（通用集合）会话级：切换会话时同步模式选择；列表惰性加载
watch(() => state.currentConvId, () => { syncConvToolsetFromConv() })
watch(() => state.workspaceRoot, () => { loadToolsetItems() })


// ★ 从会话/工作区加载审核模式（黑白名单配置已由插件面板/工具集管理取代，不再加载）
// ★ 2026-08-31 会话级：带 convId 读取（会话元数据优先，未设置回落工作区）
async function loadWorkspaceReviewConfig() {
  try {
    const rc = await api.apiGet('/tools/review?convId=' + encodeURIComponent(state.currentConvId || ''))
    if (rc && rc.reviewMode) {
      reviewMode.value = rc.reviewMode
    }
  } catch (e) {
    // 失败时回退到全局 settings
    if (state.settings && state.settings.reviewMode) {
      reviewMode.value = state.settings.reviewMode
    }
  }
}
// 会话切换时同步该会话的审核模式（会话级选择持久化，切换即恢复）
watch(() => state.currentConvId, () => { loadWorkspaceReviewConfig() })

// ── 工作区切换时加载 Token 统计（onMounted 时 workspaceRoot 可能还未设）
watch(() => state.workspaceRoot, (root) => {
  if (root && root !== '') {
    // 清理 onMounted 阶段可能存到空 key 的脏数据
    if (state.wsTokenStatsByWs['']) {
      delete state.wsTokenStatsByWs['']
    }
    loadWsTokenStats()
    loadWorkspaceReviewConfig()
  }
})

// ── 工具开关弹窗已移除（能力由插件面板/工具集管理），无 toolConfigOpen watch

const handleBeforeUnload = () => { if (state.currentConvId && state.messages.length > 0) { window.dispatchEvent(new Event('save-conversations')) } }

// ─── chat 槽位（Slot 系统）：插件注册 chat 槽位并激活后，整个对话面板由插件渲染 ──
const chatSlot = useSingleSlot('chat')
chatSlot.init() // setup 同步初始化 owner（首帧直接走正确分支）

onMounted(() => {
  loadModelData().then(() => { initComposerModel() })
  loadToolsetItems().then(() => syncConvToolsetFromConv())
  loadWsTokenStats(); loadConvList(); scrollToBottom()
  if (state.workspaceRoot && state.workspaceRoot !== '') loadWorkspaceReviewConfig()

  // chat 槽位订阅（插件可替换对话面板）
  chatSlot.start()

  // chat-tools 槽位（list 型）：输入区上方工具条细粒度叠加
  chatToolsUnsub = mountListSlot(chatToolsEl, 'chat-tools')

  // 监听消息内容尺寸变化（流式输出时自动跟随滚底）
  nextTick(() => startContentResizeObserver())

  // 按钮在 textarea 外部下方，无需动态调整 padding
  nextTick(() => updateInputPadding())

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

  // ★ 直接检查是否需要恢复对话（替换 restore-conversation 事件机制：
  //   App.vue onMounted 中 dispatchEvent 时 RightPanel 尚未挂载，事件永远丢失。
  //   改为直接检测 currentConvId → 若已设但消息未加载，调用 switchConv 加载。）
  if (state.currentConvId && (!state.messagesByConv[state.currentConvId] || state.messagesByConv[state.currentConvId].length === 0)) {
    switchConv(state.currentConvId)
  }

  // 注册全局 UI 回调：App.vue 的 WebSocket onmessage → agent-events.js → 此处回调
  setGlobalCtx({
    scrollToBottom: () => {
      // ★ 用户上翻时锁定自动滚动，直到手动滚回底部或点击跳底按钮
      if (window.__scrollLockTimer) return
      scrollToBottom()
    },
    loadWsTokenStats: () => loadWsTokenStats(),
    autoNameConv: (convId, text) => autoNameConv(convId, text),
    // ★ 2026-08-31 断线重连同步（agent-events processStatus 调用）：
    //   reloadConvMessages 重载已完成会话最新消息；refreshConvMeta 刷新会话元数据
    reloadConvMessages: (convId) => reloadConvMessages(convId),
    refreshConvMeta: () => refreshConvMeta(),
    saveConvMsg: (convId, content, msgIdx) => {
      // 后端 startEventPersistWorker 已通过 SegmentsFromMessage 自动持久化
      // loop.History 中的消息（含 ToolCalls→tool_call, Reasoning→thinking, Content→content）。
      // 前端不再重复 POST，避免消息重复追加。
    },
    onPlanUpdate: (plan, convId) => {
      // ★ 2026-08-31：plan 体系已移除——onPlanUpdate 事件不再处理（保留回调壳防旧代码报错）
      if (state.currentConvId !== convId) return
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
    addAttachment({ type: detail.type || 'file', path: detail.path || '', filename: detail.filename || '', lineStart: detail.lineStart || null, lineEnd: detail.lineEnd || null, content: detail.content || '' })
  })
  window.addEventListener('workspace-switched', async () => {
    // 工作区切换：不清空 messagesByConv/loadingByConv/agentRunningByConv（agent 后台继续运行）
    // 仅重新加载当前工作区的对话列表；loadConvList 内部会在无对话时自动创建
    state.chatLoading = false
    state.agentRunning = false
    state.chatSessionId = ''
    setInputText('')
    // ★ 2026-08-31：plan 下线——currentPlan 已移除，只清任务列表。
    await loadConvList()
    // loadConvList 已处理 currentConvId 和 messages 的设置（自动创建或保持空）
    if (!state.currentConvId) {
      state.messages = []
    }
  })
  window.addEventListener('beforeunload', handleBeforeUnload)
})

onUnmounted(() => {
  if (phaseTimer) { clearTimeout(phaseTimer); phaseTimer = null }
  if (nudgeTimer) { clearTimeout(nudgeTimer); nudgeTimer = null }
  stopContentResizeObserver()
  chatSlot.stop()
  if (chatToolsUnsub) { chatToolsUnsub(); chatToolsUnsub = null }
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
.chat-messages { flex: 1; overflow-y: auto; padding: 10px 14px; min-height: 0; position: relative; overflow-anchor: none; }
.msg-list-wrap { display: flex; flex-direction: column; gap: 14px; min-height: 100%; }
.msg-item { display: flex; gap: 8px; align-items: flex-start; content-visibility: auto; contain-intrinsic-size: 60px; border-radius: 10px; padding: 2px 4px; transition: background 0.12s; }
.msg-item:hover { background: var(--bg-hover); }

/* 新对话空状态 */
.chat-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: var(--text-muted);
  position: absolute;
  inset: 0;
}
.chat-empty-icon { margin-bottom: 16px; opacity: 0.35; color: var(--accent); filter: drop-shadow(0 0 12px rgba(88, 166, 255, 0.35)); }
.chat-empty-text { font-size: 16px; font-weight: 500; margin-bottom: 6px; color: var(--text-secondary); }
.chat-empty-hint { font-size: 13px; opacity: 0.7; }
.msg-user { flex-direction: row-reverse; justify-content: flex-start; gap: 10px; }




.msg-user .bubble-user {
  flex: 0 0 auto;
  max-width: 80%;
  min-width: 40px;
  /* ★ 2026-08-21 可视优化：纯色蓝底 → accent 渐变 + 柔和阴影，保持主题 accent 色调 */
  background: linear-gradient(135deg, var(--accent) 0%, var(--accent-light) 100%);
  color: #fff;
  padding: 10px 16px;
  border-radius: 14px 14px 4px 14px;
  overflow-wrap: break-word;
  word-break: break-word;
  overflow-wrap: anywhere;
  margin: 2px 0;
  box-shadow: 0 2px 8px rgba(0,0,0,0.22);
  transition: box-shadow 0.15s ease, transform 0.1s ease;
  position: relative;
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
.rollback-area { opacity: 0; transition: opacity 0.15s; position: absolute; right: -2px; top: -6px; z-index: 2; }
.msg-item:hover .rollback-area { opacity: 1; }
.rollback-btn { display: flex; align-items: center; gap: 2px; padding: 1px 6px; border-radius: 8px; cursor: pointer; font-size: 10px; color: rgba(255,255,255,0.6); background: rgba(0,0,0,0.2); border: none; user-select: none; }
.rollback-btn:hover { color: #f48771; background: rgba(244, 135, 113, 0.25); }
.msg-avatar { width: 28px; height: 28px; border-radius: 50%; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.msg-user .msg-avatar { background: linear-gradient(135deg, var(--accent) 0%, var(--accent-light) 100%); color: #fff; box-shadow: 0 1px 4px rgba(88, 166, 255, 0.25); }
.msg-assistant .msg-avatar { background: linear-gradient(135deg, var(--bg-tertiary) 0%, var(--bg-active) 100%); color: var(--accent); border: 1px solid var(--border-color); }
.msg-bubble { flex: 1; min-width: 0; max-width: 85%; font-size: 13px; line-height: 1.6; word-break: break-word; overflow-wrap: break-word; position: relative; padding: 2px 0; }

.bubble-assistant { background: transparent; color: var(--text-primary); padding: 2px 0; }
/* ★ 2026-09-10 slash 命令结果卡片：命令命中执行后直接展示，不唤醒模型 */
.slash-result-card { margin: 4px 0 8px; }
.slash-result-head {
  display: inline-flex; align-items: center; gap: 6px;
  font-size: 11px; font-weight: 600; color: var(--accent);
  background: color-mix(in srgb, var(--accent) 8%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent) 25%, transparent);
  border-radius: 6px; padding: 3px 8px; margin-bottom: 6px;
}
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
.phase-bar { display: flex; align-items: center; gap: 6px; padding: 4px 12px; background: linear-gradient(90deg, rgba(212, 167, 78, 0.08), rgba(212, 167, 78, 0.02)); border-bottom: 1px solid rgba(212, 167, 78, 0.2); font-size: 12px; color: #d4a74e; flex-shrink: 0; flex-wrap: wrap; }
.phase-icon { flex-shrink: 0; }
.phase-text { font-weight: 600; }
.phase-stats { display: flex; gap: 12px; margin-left: auto; }
.phs-item { display: flex; align-items: center; gap: 3px; color: rgba(212, 167, 78, 0.6); font-size: 10px; }
.phase-bar-track { width: 100%; height: 2px; background: rgba(212, 167, 78, 0.1); border-radius: 1px; margin-top: 2px; }
.phase-bar-fill { height: 100%; background: #d4a74e; border-radius: 1px; transition: width 1s ease; }
.folded-summary { display: flex; align-items: center; gap: 5px; padding: 5px 10px; background: var(--bg-primary); border: 1px solid var(--border-color); border-left: 3px solid var(--accent); border-radius: 6px; font-size: 12px; cursor: pointer; transition: background 0.15s, border-color 0.15s; }
.folded-summary:hover { background: var(--bg-hover); border-color: var(--accent); }
.folded-chevron { flex-shrink: 0; color: var(--text-muted); display: block; }
.folded-title { color: var(--text-primary); font-weight: 500; }
.folded-desc { color: var(--text-muted); }

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
.tl-item { display: flex; align-items: flex-start; gap: 6px; padding: 2px 0; position: relative; }
.tl-dot { width: 8px; height: 8px; min-width: 8px; border-radius: 50%; flex-shrink: 0; margin-top: 5px; border: 2px solid var(--border-color); background: var(--bg-primary); z-index: 1; box-shadow: 0 0 0 2px var(--bg-primary); }
.tl-dot-thinking { border-color: var(--accent); background: var(--accent-bg); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px rgba(212,167,78,0.3); }
.tl-dot-tool { border-color: #d4a74e; background: rgba(212,167,78,0.2); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px rgba(212,167,78,0.2); }
.tl-dot-ask { border-color: #c586c0; background: rgba(197,134,192,0.2); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px rgba(197,134,192,0.2); }
.tl-dot-content { border-color: #6a9955; background: rgba(106,153,85,0.2); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px rgba(106,153,85,0.2); }
.tl-dot-done { border-color: var(--accent); background: var(--accent); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px rgba(126,184,218,0.4); }
.tl-body { flex: 1; min-width: 0; font-size: 13px; line-height: 1.6; }
/* ── 思考段：背景区分 + 左边框 + 改进滚动条 ── */
.tl-think-body { position: relative; }
.tl-thinking-text { color: var(--text-secondary); font-style: italic; white-space: pre-wrap; padding: 6px 10px; max-height: 300px; overflow-y: auto; background: var(--bg-tertiary); border-radius: 6px; border-left: 2px solid var(--accent); margin: 2px 0; }
.tl-think-fold { position: sticky; bottom: 0; display: inline-block; font-size: 11px; color: var(--accent); cursor: pointer; padding: 3px 10px; margin-top: 4px; user-select: none; background: var(--bg-tertiary); border: 1px solid var(--border-color); border-radius: 4px; transition: background 0.15s; }
.tl-think-fold:hover { background: var(--bg-hover); color: var(--accent-light); }
.tl-thinking-collapsed { color: var(--text-muted); font-style: italic; font-size: 12px; cursor: pointer; padding: 2px 0; }
.tl-tc-header { display: flex; align-items: center; gap: 4px; cursor: pointer; padding: 4px 8px; user-select: none; border-radius: 4px; transition: background 0.15s; }
.tl-tc-header:hover { background: var(--bg-hover); }
.tl-tc-chevron { color: var(--text-muted); width: 8px; flex-shrink: 0; display: block; }
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
.chat-input-area { display: flex; flex-direction: column; flex-shrink: 0; padding: 0 8px 8px 8px; background: var(--bg-secondary); }
/* chat-tools 槽位（list 型）：输入区上方工具条，细粒度叠加不撑满 */
.plugin-slot-chat-tools {
  display: flex;
  align-items: center;
  gap: 6px;
  height: auto;
  min-height: 0;
  padding: 2px 0;
}
.plugin-slot-chat-tools .plugin-slot-item {
  display: flex;
  align-items: center;
}
/* ★ 未完成任务提示条 */
.resume-banner {
  display: flex; align-items: center; gap: 8px;
  margin: 0 0 6px 0; padding: 6px 10px;
  background: rgba(232, 172, 82, 0.12);
  border: 1px solid rgba(232, 172, 82, 0.35);
  border-radius: 6px;
  font-size: 12px; color: #e8ac52;
}
.resume-icon { flex-shrink: 0; font-size: 13px; }
.resume-text { flex: 1; line-height: 1.4; }
.resume-btn {
  flex-shrink: 0;
  display: inline-flex; align-items: center; gap: 4px;
  padding: 3px 10px; border-radius: 4px;
  background: rgba(232, 172, 82, 0.25);
  border: 1px solid rgba(232, 172, 82, 0.5);
  color: #f0c674; font-size: 11px; font-weight: 500;
  cursor: pointer; white-space: nowrap;
}
.resume-btn:hover { background: rgba(232, 172, 82, 0.4); }
.input-resizer { position: absolute; top: -8px; left: 0; right: 0; height: 12px; cursor: ns-resize; z-index: 10; }
.input-wrapper { position: relative; background: var(--input-bg); border: 1px solid var(--border-color); border-radius: 16px; transition: border-color 0.15s ease, box-shadow 0.15s ease, background 0.15s ease; box-shadow: 0 2px 10px rgba(0, 0, 0, 0.12); }
/* ★ Round3 ④.2 slash 命令菜单（输入 "/" 前缀时显示在输入框上方） */
.slash-menu { position: absolute; left: 0; right: 0; bottom: 100%; margin-bottom: 6px; background: var(--panel-bg, #1f2430); border: 1px solid var(--border-color); border-radius: 10px; box-shadow: 0 8px 24px rgba(0, 0, 0, 0.35); max-height: 280px; overflow-y: auto; z-index: 60; padding: 4px; }
.slash-item { display: flex; align-items: baseline; gap: 10px; padding: 7px 10px; border-radius: 7px; cursor: pointer; }
.slash-item.active { background: rgba(0, 120, 212, 0.16); }
.slash-name { font-family: var(--mono-font, monospace); color: #4daafc; font-size: 13px; flex-shrink: 0; }
.slash-name .slash-ondemand { font-style: normal; font-size: 10px; color: #ffb454; border: 1px solid #ffb45455; border-radius: 4px; padding: 0 4px; margin-left: 6px; vertical-align: 1px; }
.slash-desc { color: var(--text-muted, #9aa4b2); font-size: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
/* ★ 2026-08-21 可视优化：输入框聚焦时 accent 描边（键盘可达性/审美） */
.input-wrapper:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--focus-ring), 0 4px 16px rgba(0, 0, 0, 0.16); }
/* ★ 2026-08-22 contenteditable 输入框改造：附件内联 tag 渲染在输入框内 */
.chat-input { display: block; width: 100%; background: transparent; border: none; color: var(--text-primary); padding: 16px 18px 8px 18px; border-radius: 0; font-size: 14px; resize: none; outline: none; min-height: 80px; font-family: inherit; line-height: 1.6; box-sizing: border-box; overflow-y: auto; white-space: pre-wrap; word-break: break-word; cursor: text; }
/* placeholder（contenteditable 无原生 placeholder，用 :empty 伪元素） */
.chat-input:empty::before { content: attr(data-placeholder); color: var(--text-muted); pointer-events: none; }
.chat-input[contenteditable="false"] { opacity: 0.55; cursor: not-allowed; }
/* 附件内联 tag：contenteditable=false 药丸，与文字混排 */
.att-inline { display: inline-flex; align-items: center; gap: 4px; padding: 1px 6px 1px 8px; margin: 0 2px; border-radius: 11px; font-size: 12px; line-height: 1.5; vertical-align: middle; user-select: none; cursor: default; max-width: 340px; background: rgba(0, 120, 212, 0.12); color: #4daafc; border: 1px solid rgba(0, 120, 212, 0.25); }
.att-inline-image { background: rgba(0, 180, 120, 0.12); color: #4cc9a0; border-color: rgba(0, 180, 120, 0.25); }
.att-inline-dir { background: rgba(180, 130, 30, 0.12); color: #e0b45e; border-color: rgba(180, 130, 30, 0.25); }
.att-inline .att-inline-icon { display: inline-flex; flex-shrink: 0; }
.att-inline .att-inline-svg { width: 12px; height: 12px; }
.att-inline .att-inline-label { max-width: 220px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.att-inline .att-inline-x { display: inline-flex; align-items: center; justify-content: center; width: 14px; height: 14px; border-radius: 50%; font-size: 12px; line-height: 1; color: inherit; opacity: 0.7; cursor: pointer; flex-shrink: 0; }
.att-inline .att-inline-x:hover { opacity: 1; background: rgba(0, 0, 0, 0.25); }
.input-bottom-bar { display: flex; align-items: center; justify-content: space-between; gap: 6px; padding: 0 10px 10px 10px; }
.ibb-btns { display: flex; align-items: center; gap: 4px; flex-wrap: wrap; position: relative; }
/* ★ 2026-09-05 移动端化：传统 <select> 已由 SheetPicker（bottom-sheet）替换，.cmp-sel 移除 */
.obtn { display: flex; align-items: center; gap: 3px; padding: 4px 10px; border-radius: 999px; cursor: pointer; font-size: 11px; color: var(--text-muted); background: var(--bg-tertiary); border: 1px solid var(--border-color); white-space: nowrap; user-select: none; transition: all 0.12s; }
.obtn:hover { color: var(--text-secondary); border-color: var(--text-muted); }
.obtn.active { color: var(--accent); background: rgba(212, 167, 78, 0.1); border-color: rgba(212, 167, 78, 0.3); }
.obtn-obtn-agent.active { color: #d4a74e; }
/* 三态审核按钮样式 */
.obtn-review-auto { color: #5bbc7a; background: rgba(91, 188, 122, 0.1); border-color: rgba(91, 188, 122, 0.3); }
.obtn-review-manual { color: #d4a74e; background: rgba(212, 167, 78, 0.1); border-color: rgba(212, 167, 78, 0.3); }
.obtn-review-off { color: var(--text-muted); background: var(--bg-tertiary); border-color: var(--border-color); opacity: 0.6; }



/* ★ 2026-09-05 移动端融合设计：发送/停止 = 圆形 FAB 按钮 */
.send-btn { background: linear-gradient(135deg, var(--accent) 0%, var(--accent-light) 100%); color: #fff; width: 34px; height: 34px; padding: 0; border-radius: 50%; cursor: pointer; border: none; transition: opacity 0.15s, transform 0.1s, box-shadow 0.15s; display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; box-shadow: 0 3px 10px rgba(79, 140, 255, 0.35); }
.send-btn svg { margin-left: 2px; }
.send-btn:hover:not(:disabled) { opacity: 0.92; transform: translateY(-1px); box-shadow: 0 5px 14px rgba(79, 140, 255, 0.45); }
.send-btn:disabled { opacity: 0.35; cursor: not-allowed; box-shadow: none; }
.stop-btn { background: #c03; color: #fff; width: 34px; height: 34px; padding: 0; border-radius: 50%; cursor: pointer; border: none; display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; box-shadow: 0 3px 10px rgba(204, 0, 51, 0.35); }
/* ── 用户消息附件标签 ── */
.user-attachments { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 6px; }
.att-tag { display: inline-flex; align-items: center; gap: 4px; padding: 1px 6px 1px 8px; border-radius: 11px; font-size: 12px; cursor: default; }
.att-tag-file, .att-tag-code, .att-tag-dir { background: rgba(0, 120, 212, 0.12); color: #4daafc; border: 1px solid rgba(0, 120, 212, 0.25); }
.att-tag-image { background: rgba(0, 180, 120, 0.12); color: #4cc9a0; border: 1px solid rgba(0, 180, 120, 0.25); }
.att-tag-label { max-width: 240px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
/* ★ 2026-08-22 附件 chip 只显示「左侧图标 + 文件名」文字大小尺寸：图片附件不再渲染 96px 大缩略图 */
/* ★ 2026-08-21 气泡内附件反色：用户气泡为 accent 渐变底色，原淡蓝标签与气泡同色系难辨认。
   改为白色半透明底 + 白字 + 白边，四主题（蓝/白/橙/紫）下均高对比 */
.bubble-user .att-tag,
.bubble-user .att-tag-file,
.bubble-user .att-tag-code,
.bubble-user .att-tag-dir,
.bubble-user .att-tag-image {
  background: rgba(255, 255, 255, 0.22);
  color: #fff;
  border: 1px solid rgba(255, 255, 255, 0.55);
}

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

/* ── 合并到 agent 气泡中的用户反馈标记 ── */
.fb-merged-section {
  background: rgba(255, 193, 7, 0.08); border: 1px dashed rgba(255, 193, 7, 0.35);
  border-radius: 6px; padding: 6px 10px; margin-bottom: 8px;
}
.fb-merged-item { margin-bottom: 4px; }
.fb-merged-item:last-child { margin-bottom: 0; }

/* ── 背景上下文快照（折叠系统信息条）── */
.snapshot-strip { margin: 6px 0 4px; }
.snapshot-item { border: 1px solid var(--border-color); border-left: 3px solid var(--accent); border-radius: 6px; background: var(--bg-secondary); font-size: 12px; margin-bottom: 4px; overflow: hidden; }
.snapshot-head { display: flex; align-items: center; gap: 5px; padding: 4px 10px; cursor: pointer; color: var(--fg-muted); user-select: none; }
.snapshot-head:hover { background: var(--bg-hover); }
.snapshot-head .folded-chevron { transition: transform 0.15s; }
.snapshot-head .folded-chevron.rotated { transform: rotate(90deg); }
.snapshot-hint { color: var(--fg-faint); font-size: 11px; }
.snapshot-body { padding: 6px 10px 8px; border-top: 1px solid var(--border-color); background: var(--bg-primary); max-height: 320px; overflow-y: auto; }
.fb-merge-label {
  font-size: 11px;
  font-weight: 600;
  color: #d4a74e;
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 4px;
}
.fb-merge-content {
  font-size: 12px;
  color: var(--text-primary);
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}

.scroll-more-hint { text-align: center; font-size: 11px; color: var(--text-muted); padding: 4px; }
.tool-calls { margin-top: 4px; }
.tool-call { background: var(--bg-primary); padding: 4px 8px; border-radius: 3px; margin-bottom: 2px; font-size: 12px; }

/* ── 任务进度容器（输入区上方；★ 2026-08-31 plan 体系已移除）── */
.task-container {
  flex-shrink: 0;
  transition: max-height 0.25s ease;
  padding: 0 8px;
}
.task-container.task-empty {
  max-height: 0;
  padding: 0 8px;
}
.task-container:not(.task-empty) {
  max-height: 400px;
}
.task-container .plan-panel {
  margin: 0 0 4px 0;
}
/* chat 槽位：插件渲染的对话面板占满 rp-body */
.plugin-slot-chat { flex: 1; min-height: 0; display: flex; overflow: hidden; }
</style>

<!-- ★ 2026-08-22 附件内联 tag 全局样式（非 scoped）：
     insertTagAtCursor 用 JS 动态创建 span/svg（无 data-v 属性），scoped 选择器
     [data-v-xxx] 匹配不到 → 样式全失效（svg 默认 300x150 巨大拉伸）。
     此处用作用域限定选择器（.chat-input 内）避免全局污染，与 scoped 版同规格。 -->
<style>
.chat-input .att-inline {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 1px 6px 1px 8px; margin: 0 2px;
  border-radius: 11px; font-size: 12px; line-height: 1.5;
  vertical-align: middle; user-select: none; cursor: default;
  max-width: 340px;
  background: rgba(0, 120, 212, 0.12); color: #4daafc;
  border: 1px solid rgba(0, 120, 212, 0.25);
  box-sizing: border-box;
}
.chat-input .att-inline-image {
  background: rgba(0, 180, 120, 0.12); color: #4cc9a0;
  border-color: rgba(0, 180, 120, 0.25);
}
.chat-input .att-inline-dir {
  background: rgba(180, 130, 30, 0.12); color: #e0b45e;
  border-color: rgba(180, 130, 30, 0.25);
}
.chat-input .att-inline .att-inline-icon { display: inline-flex; flex-shrink: 0; }
.chat-input .att-inline .att-inline-svg { width: 12px; height: 12px; display: block; }
.chat-input .att-inline .att-inline-label {
  max-width: 220px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.chat-input .att-inline .att-inline-x {
  display: inline-flex; align-items: center; justify-content: center;
  width: 14px; height: 14px; border-radius: 50%; font-size: 12px; line-height: 1;
  color: inherit; opacity: 0.7; cursor: pointer; flex-shrink: 0;
}
.chat-input .att-inline .att-inline-x:hover { opacity: 1; background: rgba(0, 0, 0, 0.25); }
</style>
