<template>
  <div class="right-panel">
    <!-- ★ 2026-09-25 对齐设计稿 th130 主区子树：删除整行 .rp-header（680×36）——
         设计主列第一个子元素就是任务横幅行。原 5 个按钮迁到顶栏 UiTitlebar 右侧图标区
         （新对话 / 会话列表 / 专注 / Debug 日志 / 关闭），见 UiTitlebar.vue .tb-right。 -->

    <div class="rp-body">
      <!-- ★ chat 槽位（Slot 系统）：插件注册 chat 槽位并激活后，整个对话面板由插件渲染（UI 可更换） -->
      <div v-if="chatSlot.owner.value" :ref="chatSlot.hostRef" class="plugin-slot-host plugin-slot-chat"></div>
      <template v-else>
      <!-- 左侧：聊天消息 + 输入区 -->
      <div class="chat-area">
        <!-- ★ 2026-09-25 对齐设计稿 th65（row 816×48）＋ th64（内嵌 card 440×40,
             bg=surface-3, r8, border）＝ 当前任务标题横幅，置于主区最顶部。 -->
        <div v-if="currentTaskTitle" class="task-banner-row">
          <div class="task-banner-card" :title="currentTaskTitle">{{ currentTaskTitle }}</div>
        </div>
        <!-- ★ 2026-09-26 用户指令 + 实测判据：原消息区顶部 phase-bar（「上次运行 / 执行中…」
             ＋ 估算进度条）已**整条删除**。四条判据：
             ① 与右栏 StatsRail「运行统计」卡同源重复——同一份 state.runStatsByConv，
                右栏已给出「运行中/已完成 · 耗时 · 输出速度」+ 步数/工具调用/LLM 调用明细；
             ② 它的「阶段」分支（currentPhase ← state.phaseByConv）无生产方：Go 侧
                EventPhase 常量仅定义未发送、JS/插件侧无任何 phase 事件发送点 → 恒为空；
             ③ 进度条按「已耗时 ÷ 60min、封顶 95%」估算，非真实进度，易被误读为卡住/将完成；
             ④ 设计稿 shell-midnight th130 子树本无该节点。
             运行态可见性不受影响：消息流「思考中...」banner + 输入区停止按钮 +
             工具行 spinner + 右栏运行统计。 -->
        <div class="chat-messages" ref="msgRef" @scroll="onScroll">
          <!-- 顶部加载更多提示 -->
          <div v-if="hasMoreTop" class="scroll-more-hint" ref="topSentinel">
            <span>加载更早消息...</span>
          </div>
          <!-- 渲染的消息列表（全量渲染，overflow-anchor 保底部稳定） -->
          <div class="msg-list-wrap">
            <!-- ★ 遍历 messageCombos，每个 user / assistant 分别渲染为独立气泡 -->
            <template v-for="(combo, ci) in messageCombos" :key="'c' + ci">
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
                  <div v-if="combo.user._time" class="msg-time">{{ combo.user._time }}</div>
                </div>
              </div>
              <!-- ── Agent 回复独立气泡（左对齐），不分段，直接显示到一起 ── -->
              <div v-if="combo.assistant" class="msg-item msg-assistant" :data-idx="combo.assistant._idx">
                <div class="msg-bubble bubble-assistant">
                  <!-- ★ 2026-09-25 对齐设计稿 th70（row h24）：accent icon16 + 「PairCode」
                       (600, fg) + 模型胶囊 th69 124×24（r=full, bg=surface-3, 文字 accent）。
                       ★ 模型名取**真实会话模型**（composerModel → parseModelValue().model），
                         取不到则整颗胶囊不渲染 —— 不硬编码 deepseek-flash。
                       （原 .msg-avatar 头像已被本头行取代，图标改由 mh-icon 承担。） -->
                  <div class="msg-head">
                    <SvgIcon name="bot" :size="16" class="mh-icon" />
                    <span class="mh-brand">PairCode</span>
                    <span v-if="convModelName" class="mh-model-pill" :title="'当前模型：' + convModelName">{{ convModelName }}</span>
                  </div>
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
                            <div v-else class="tl-thinking-collapsed" @click="toggleThinking(seg, combo.assistant._idx, si)"><SvgIcon name="message-square" :size="12" /> 思考…</div>
                            <div v-if="!seg._collapsed" class="tl-think-fold" @click.stop="toggleThinking(seg, combo.assistant._idx, si)" title="折叠思考">▲ 收起</div>
                          </div>
                        </div>
                        <!-- ★ 2026-09-26 用户反馈修正：原实现把状态做成**贴最右的胶囊(tag)**
                             （.tr-pill 用 margin-left:auto 推出 + 名称固定 240 宽 → 中间大片
                             留白、整体像右侧标签），不符合预期。现改为**消息流内左对齐成组**：
                             accent icon14 + 工具名(xs, 自适应 max220, 单行省略)
                             + 运行中 spinner + 结果摘要(xs, 紧随其后、单行省略，用文字色区分
                             成功/错误/运行中，**无背景无圆角**) + chevron 紧随其后。
                             行本体沿用 th83/th90 行式（h32, bg=surface-2, r8, border）。
                             ★ 能力保留：整行可点，展开区（参数/结果/命令/输出）照旧，未删任何分支。 -->
                        <div v-else-if="seg.type === 'tool_call'" class="tool-row-wrap">
                          <div class="tool-row" :class="{ 'is-open': seg._expanded }"
                               @click="toggleTool(seg, combo.assistant._idx, si)">
                            <SvgIcon :name="toolMeta(seg).icon" :size="14" class="tr-icon" />
                            <span class="tr-name" :title="toolMeta(seg).title">{{ toolMeta(seg).title }}</span>
                            <span v-if="!seg.result" class="tr-spinner" title="运行中"></span>
                            <span class="tr-status"
                                  :class="isToolErr(seg) ? 'tr-status-err' : (seg.result ? 'tr-status-ok' : 'tr-status-run')"
                                  :title="seg.result ? toolResultSummary(seg) : '运行中'">{{ seg.result ? toolResultSummary(seg) : '运行中' }}</span>
                            <svg class="tr-chevron" viewBox="0 0 8 8" width="10" height="10" fill="currentColor" aria-hidden="true">
                              <path :d="seg._expanded ? 'M1.2 2.6 L4 6.8 L6.8 2.6 Z' : 'M2.6 1.2 L6.8 4 L2.6 6.8 Z'"/>
                            </svg>
                          </div>
                          <div v-if="seg._expanded" class="tl-tc-detail tool-row-detail">
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
                        <div v-else-if="seg.type === 'ask_user'" class="tl-item">
                          <span class="tl-dot tl-dot-ask"></span>
                          <!-- ★ 2026-09-25：answer/answers 一并传入——历史回看时回显当时的选择项/自定义输入（此前不传，卡片只显示原始问题） -->
                          <div class="tl-body"><AskUserCard :question="seg.question" :ask-type="seg.askType" :options="seg.options" :questions="seg.questions" :call-id="seg.callId" :answer="seg.answer" :answers="seg.answers" :answered="seg._answered" :stale="seg._stale" @answer="onAskAnswer(seg, $event)" /></div>
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
          <!-- ★ 2026-09-25（设计稿屏 1 ③）：「下一步可以试试」引导卡。
               首轮对话结束后出现在消息流末尾 —— 此时用户最常问「那你还能做什么」，
               用 4 个可点芯片把能力边界具体化；点击**填入输入框**（不直接发送，可先编辑）。
               可关闭，关闭后写独立 key，不再出现。 -->
          <div v-if="nextStepsVisible" class="next-steps">
            <div class="ns-head">
              <span class="ns-title">下一步可以试试</span>
              <span class="ns-hint">也可以直接描述你的目标</span>
              <!-- ★ 2026-09-25 对齐设计稿 th106：设计引导卡**无关闭「×」按钮**（原 close 入口下线）。 -->
            </div>
            <div class="ns-chips">
              <button v-for="(s, si) in NEXT_STEPS" :key="s.text" class="ns-chip"
                      :class="{ 'ns-chip-primary': si === 0 }" :title="s.hint" @click="useNextStep(s.text)">
                <span>{{ s.text }}</span>
              </button>
            </div>
            <!-- ★ 2026-09-25 对齐设计稿 th105（底部提示行 xs muted） -->
            <div class="ns-foot">不确定怎么开始？直接输入你的需求（Ctrl+K 可进入专注模式），或让 Agent 先读项目再给方案。</div>
          </div>
          <div v-if="(!state.messages || state.messages.length === 0) && !state.chatLoading" class="chat-empty">
            <!-- ★ P4-3（2026-09-25）首启引导卡：空对话 + 三步未走完时替代朴素空态，
                 给出「配置服务商 → 打开项目 → 开始对话」动线；两步都完成或点「不再显示」即消失。 -->
            <div v-if="onboardingVisible" class="ob-card">
              <div class="ob-head">
                <div class="ob-logo"><SvgIcon name="bot" :size="20" /></div>
                <div>
                  <div class="ob-title">欢迎使用 PairCode</div>
                  <div class="ob-sub">三步开始：接通模型 → 打开项目 → 开始对话</div>
                </div>
              </div>
              <ul class="ob-steps">
                <li class="ob-step" :class="{ 'is-done': obHasProvider }">
                  <span class="ob-no"><SvgIcon v-if="obHasProvider" name="check" :size="11" /><template v-else>1</template></span>
                  <span class="ob-body">
                    <span class="ob-t">配置模型服务商</span>
                    <span class="ob-d">{{ obHasProvider ? '已配置 ' + obProviderCount + ' 个服务商' : '填入 API Key 与模型，AI 才能工作' }}</span>
                  </span>
                  <button v-if="!obHasProvider" class="ob-act" @click="obOpenSettings">去配置</button>
                </li>
                <li class="ob-step" :class="{ 'is-done': obHasWorkspace }">
                  <span class="ob-no"><SvgIcon v-if="obHasWorkspace" name="check" :size="11" /><template v-else>2</template></span>
                  <span class="ob-body">
                    <span class="ob-t">打开项目文件夹</span>
                    <span class="ob-d">{{ obHasWorkspace ? obWorkspaceLabel : '让 AI 能读写你的代码（菜单 文件 → 打开文件夹 亦可）' }}</span>
                  </span>
                  <button v-if="!obHasWorkspace" class="ob-act" @click="obOpenFolder">打开</button>
                </li>
                <li class="ob-step" :class="{ 'is-done': obReady }">
                  <span class="ob-no"><SvgIcon v-if="obReady" name="check" :size="11" /><template v-else>3</template></span>
                  <span class="ob-body">
                    <span class="ob-t">开始对话</span>
                    <span class="ob-d">{{ obReady ? '准备就绪：在下方输入框描述任务，Enter 发送' : '完成前两步后即可开始' }}</span>
                  </span>
                </li>
              </ul>
              <div class="ob-foot">
                <button class="ob-dismiss" @click="dismissOnboarding">不再显示</button>
              </div>
            </div>
            <template v-else>
              <div class="chat-empty-icon"><SvgIcon name="bot" :size="32" /></div>
              <div class="chat-empty-text">开始新的对话</div>
              <div class="chat-empty-hint">发送消息即可与 AI 助手对话</div>
            </template>
          </div>
          <!-- 新消息跳底按钮 -->
          <div v-if="showScrollDown" class="scroll-down-btn" :class="{ 'show-pulse': state.chatLoading }" @click.stop="scrollToBottom">
            <button><SvgIcon name="chevron-down" :size="14" /> 新消息</button>
          </div>
        </div>
        <!-- 任务进度面板（task 体系：扁平任务列表） -->
        <!-- ★ 2026-09-25 对齐设计稿 th130：删除主区内的「任务进度」面板（与右栏
             StatsRail th164「任务进度」卡 264×152 重复）——任务进度唯一归宿是右栏。 -->
        <!-- 自主模式监督看板（autopilot 插件数据面：GET /api/autopilot/rounds + ui:autopilot:round 事件；
             无监督回合（插件未装载/未开启自主模式）时容器高度为 0，不占位） -->
        <div class="autopilot-container" :class="{ 'autopilot-empty': autopilotRounds.length === 0 }">
          <AutopilotPanel v-if="autopilotRounds.length > 0" :rounds="autopilotRounds" :expanded="autopilotExpanded" @toggle="autopilotExpanded = !autopilotExpanded" />
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
            <!-- ★ 2026-09-25 对齐设计稿 th129 行1（th114, h24）：模型胶囊(th107, bg=surface-3,
                 r=full, accent 文字, xs) + 工具集胶囊(th109, bg=surface-2, muted, xs)
                 + 弹性空隙。
                 ★ 2026-09-24 修复：行1 右端的「自主模式」文字状态已删除——它与行3 操作区
                 的 .obtn-agent 是同一开关的两处表达（即「输入框右上角还有状态显示」），
                 开关本体只保留操作区一处。 -->
            <div class="input-row1">
              <span class="ir-pill ir-pill-model" v-if="convModelName" :title="'当前模型：' + convModelName">{{ convModelName }}</span>
              <!-- ★ 2026-09-24 修复：原为 v-if="toolsetItems.length" —— 场景列表为空时
                   （<InstallDir>/.pair/toolsets/ 下无集合文件）整条降级为只读 span，用户
                   看到的就是「场景选择点了没反应／没有实现」。改为始终渲染可交互选择器：
                   空列表由 SheetPicker 的 empty-text 明示，功能入口恒在、不再静默消失。
                   ★ 行3 原有一处重复的工具集 SheetPicker 已删除（勿重复入口）。 -->
              <SheetPicker
                class="ir-pill-picker"
                v-model="convToolset"
                :items="toolsetSheetItems"
                :title="toolsetPickerTitle"
                :placeholder="toolsetPlaceholder"
                empty-text="暂无场景：请先在「工具集」面板创建或导入"
                @change="onConvToolsetChange"
              />
              <span class="ir-spacer"></span>
            </div>
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
            <!-- ★ 2026-09-25 对齐设计稿 th115：占位文案改为设计原文 -->
            <div class="chat-input" ref="inputRef" :contenteditable="state.chatLoading ? 'false' : 'true'" :style="{ height: inputHeight + 'px' }" data-placeholder="描述你要做的事，Agent 会自己选工具、跑验证…" @keydown="onKeydown" @input="onInput" @dragover.prevent @drop="handleDrop" @paste="handlePaste"></div>
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
                <!-- ★ 2026-09-25 监督纠正（缺陷 2）：此处原有的工具集 SheetPicker 已删除——
                     它是行1 胶囊的重复入口。现统一由行1 的 .ir-pill-picker 承载（单一入口）。 -->
                <!-- ★ 2026-09-25 监督纠正（缺陷 3）：按设计稿 ui-theme.notes.md:1262 重做输入卡操作区。
                     去掉 放行/折叠/自主 那套「自绘 checkbox + 勾选字形（☑/✓）+ 文字」表现，
                     统一为设计稿控件规格：3 个 32×32 圆角图标卡（radius=8，选中态走 --accent），
                     三者外观完全一致；模式名与其含义改由 title 悬浮提示承载（不再挤占宽度）。 -->
                <span :class="['obtn', reviewBtnClass]" @click="cycleReviewMode" :title="reviewBtnTitle"><SvgIcon :name="reviewIconName" :size="14" /></span>
                <span :class="['obtn', { active: autoCollapse }]" @click="toggleAuto('autoCollapse')" title="自动折叠：新消息发出时折叠旧输出，显示完成摘要"><SvgIcon name="list" :size="14" /></span>
                <span :class="['obtn', 'obtn-agent', { active: autonomous }]" @click="toggleAuto('autonomous')" :title="autonomous ? '自主模式：开（连续执行全部计划步骤）' : '自主模式：关（单次回复）'"><SvgIcon name="cycle" :size="14" color="var(--color-cat-amber)" /></span>
              </div>
              <!-- ★ 2026-09-25 对齐设计稿 th127：右侧「Enter 发送 · Shift+Enter 换行」(xs muted) -->
              <span class="ibb-enter-hint">Enter 发送 · Shift+Enter 换行</span>
              <!-- ★ 2026-09-25 对齐设计稿 th126：72×32 蓝底**文字**按钮「发送」（原为纯图标 = 错） -->
              <button v-if="!state.chatLoading" class="send-btn" @click="sendMessage" :disabled="!inputText.trim() && pendingAttachments.length === 0">发送</button>
              <button v-else class="stop-btn" @click="stopChat"><SvgIcon name="stop-dot" :size="20" /></button>
            </div>
          </div>
        </div>
      </div>
      <!-- 右侧：Debug日志面板 / 会话列表 -->
      <DebugLogPanel v-if="showDebugLog" @close="showDebugLog = false" />
      <!-- ★ 2026-09-25 对齐设计稿：会话列表已迁至【左栏】（Sidebar 的 chat 视图，
           th61），Token 统计/上下文构成已迁至【右栏】（StatsRail，th148/th173）。
           主区不再重复渲染会话列表与圆环统计（此前会与左右栏三处重复）。 -->
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
// ★ 2026-09-26：移除未使用的 rightPanelWidth 导入（构建期告警 "imported but never used"，
//   文件内零引用；右栏宽度由 layout.rightPanelWidth 直接读，不经此具名导入）。
import { state, layout, savePersistentState, showSettings } from '../ui-state.js'
import api from '../api.js'
import { visibleConversations } from '../conv-filters.js'
import { setGlobalCtx, startConvRuntime, resetConvRuntime, createAssistantPlaceholder, getConvRuntime, getConvCtxStats, resetConvCtxStats, normalizeAskType, markHistoryLoaded, fetchRunStats } from '../agent-events.js'
import { useSingleSlot, mountListSlot } from '../plugin-runtime.js'
import SvgIcon from './SvgIcon.vue'
import SheetPicker from './SheetPicker.vue'
import TaskPanel from './TaskPanel.vue'
import AutopilotPanel from './AutopilotPanel.vue'
import ApprovalBar from './ApprovalBar.vue'
import ConvSidebar from './ConvSidebar.vue'
import AskUserCard from './AskUserCard.vue'
// SubAgentBlock 不再使用，替换为内联时间线展示
import MarkdownRenderer from './MarkdownRenderer.vue'
import DebugLogPanel from './DebugLogPanel.vue'

const showDebugLog = ref(false)

const props = defineProps({ panelMode: { type: Boolean, default: false } })
const panelMode = computed(() => !!props.panelMode)

// ─── 会话列表面板（Token 统计栏）显隐（2026-09-12 新增）───
//   头部按钮与 Ctrl+Shift+L 都走这里；状态读写见 ui-state.js（持久化 + layout.toggleConvList）。
//   ★ 专注模式联动已收归 ui-state.setFocusMode（唯一入口，一并处理左栏与会话列表）：
//     组件内不再 watch focusMode —— watch 默认 flush:'pre' 会在同块后续语句之后回写，
//     覆盖「显式唤出侧栏」的意图（详见 ui-state.js 注释）。
const toggleConvList = () => {
  // 经 layout 服务切换：专注态内手动调整会同步「退出专注」还原目标
  layout.toggleConvList()
  // ★ 立即落盘：宿主无全局 state watch（savePersistentState 原本只在 switchWorkspace 调用），
  //   不主动保存则刷新后无法记住本次选择。
  // ★ 专注态内为临时视图态，不落盘（与 Ctrl+B / Ctrl+Shift+L 同规则，防把专注收起误存为偏好）。
  if (state.focusMode === false) savePersistentState()
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
    // ★ 2026-09-08：带 workspaceRoot（后端激活后自动唤醒 agent 需按会话工作区路由）；
    //   执行成功后清空输入框——此前残留 "/name " 让用户误判「命令没生效」。
    const res = await api.runCommand(name, { args: m[2] || '' }, state.currentConvId, state.workspaceRoot)
    setInputText('')
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
// ★ 2026-09-25 对齐设计稿 th69：模型胶囊文案 = 当前会话真实模型名（不硬编码；取不到则隐藏胶囊）
const convModelName = computed(() => {
  try { return (parseModelValue(composerModel.value) || {}).model || '' } catch (e) { return '' }
})
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
  // ★ 2026-09-24 修复：会话当前模型可能不在候选列表里（服务商模型表更新、模型别名、
  //   自定义 model 名——例：会话存 "deepseek-flash" 而列表里是 "deepseek-v4-flash"）。
  //   此时原生 select 无匹配 option → selectedIndex=-1 → 回落显示 placeholder「选择模型…」，
  //   用户看不到本会话正在使用的模型。补一项「当前使用」置顶，保证下拉始终表达会话真值。
  const cur = composerModel.value
  if (cur && !out.some(it => it.value === cur)) {
    const parsed = parseModelValue(cur)
    if (parsed.model) {
      out.unshift({ value: cur, label: parsed.model, desc: (parsed.provider ? parsed.provider + ' · ' : '') + '当前使用' })
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
// 全局默认（激活配置）解析出的 服务商+模型：会话未设模型时下拉显示它
// ★ 2026-09-20：不再读 settings 顶层连接字段（provider/executeModel 已由 core 迁入
//   ai-presets.json 的一条配置并清空，见 core/settings_connection.go）——唯一来源是
//   激活配置（settings.preset → ai-presets.json 整套展开）。
function defaultProviderModel() {
  const s = state.settings || {}
  const md = modelData.value || {}
  const presets = md.presets || null // 激活配置（携带 Key），仅解析服务商/模型
  const pres = (presets && s.preset && presets[s.preset]) || null
  let prov = (pres && pres.provider) || ''
  let model = (pres && pres.executeModel) || ''
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

// ─── ★ P4-3（2026-09-25）首启引导卡 ─────────────────────────────────────────
//   目标：首次使用者在空对话里拿到一条「三步动线」，而不是一句「发送消息即可对话」。
//   显示条件：① 未点过「不再显示」② 前两步未同时完成（都完成了引导已无意义 → 自动消失）。
//   判定数据：服务商 ← modelData.providers（loadModelData 已拉 /api/models，直接复用，不额外请求）；
//            工作区 ← state.workspaceRoot / workspaceName（启动时由 /api/health 注入）。
//   「不再显示」写独立 key（不进 paircode-ide-state）：那是布局偏好，
//   这是「一次性引导已完成」语义，混入会导致清布局时引导意外复活。
const OB_DISMISS_KEY = 'paircode-onboarding-dismissed'
const onboardingDismissed = ref(false)
try { onboardingDismissed.value = localStorage.getItem(OB_DISMISS_KEY) === '1' } catch {}
const obProviderCount = computed(() => {
  const md = modelData.value
  return md && Array.isArray(md.providers) ? md.providers.length : 0
})
const obHasProvider = computed(() => obProviderCount.value > 0)
const obHasWorkspace = computed(() => !!(state.workspaceRoot && state.workspaceRoot !== ''))
const obWorkspaceLabel = computed(() => state.workspaceName || state.workspaceRoot || '')
const obReady = computed(() => obHasProvider.value && obHasWorkspace.value)
const onboardingVisible = computed(() => !onboardingDismissed.value && !obReady.value)
function obOpenSettings() { showSettings.value = true }
async function obOpenFolder() {
  // 与 MenuBar「文件 → 打开文件夹」同一条链路（后端 /workspace add-folder + 刷新文件树）
  const path = await window.$prompt('输入文件夹路径:', '', '打开文件夹')
  if (!path) return
  try {
    await api.apiPost('/workspace', { action: 'add-folder', path })
    window.dispatchEvent(new CustomEvent('refresh-tree'))
  } catch (err) { window.$toast('添加失败: ' + err.message, 'error') }
}
function dismissOnboarding() {
  onboardingDismissed.value = true
  try { localStorage.setItem(OB_DISMISS_KEY, '1') } catch {}
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
const toolsetItems = ref([])           // 全局工具集列表（GET /api/toolsets，不含 builtin 虚拟）
const convToolset = ref('')            // 选择器显示值（= 当前实际生效的集合名）
const convToolsetEffective = ref('')   // 实际生效集合名（未显式选择 → 后端默认集合；'' = 未收敛）
const convToolsetIsDefault = ref(true) // true = 会话未显式选择（生效值来自默认集合）
const convToolsetConverged = ref(false) // true = 按集合收敛工具面（false = 集合缺失，全量保留）
const convToolsetDefaultName = ref('')  // 默认集合名（presetNameDefault，如「基础」）
const pendingConvToolset = ref('')     // 新对话尚未创建时的暂存选择（建会话后写入）
// ★ 2026-09-25 会话列表底部「场景」胶囊接线：把本处解析出的权威值镜像到共享
//   state.convToolsetInfo，供 ConvSidebar（设计稿 th60「场景名 · N 插件」）显示。
//   ★ 本函数是 state.convToolsetInfo 的唯一写方（避免双源不一致）。
//   pluginCount 取「该集合装配的插件数」（toolsetItems 的 pluginCount），不是
//   state.pluginSchemas.length（后者是全部已装插件数，与场景无关）。
function publishConvToolsetInfo() {
  const name = convToolsetEffective.value || ''
  const hit = (toolsetItems.value || []).find(t => t.name === name)
  state.convToolsetInfo = {
    name: name,
    defaultName: convToolsetDefaultName.value || '',
    isDefault: convToolsetIsDefault.value,
    converged: convToolsetConverged.value,
    pluginCount: hit ? (hit.pluginCount || 0) : 0,
    resolved: true,
  }
}
// ★ 2026-09-05 移动端化：工具集选择器 bottom-sheet 选项
// ★ 2026-09 修复：生效项标注「默认生效 / 当前」——用户一眼看出当前对话用的是哪个集合。
const toolsetSheetItems = computed(() => {
  const eff = convToolsetEffective.value
  const out = toolsetItems.value.map(t => {
    const n = (t.pluginCount || 0) + ' 个插件'
    if (eff && t.name === eff) {
      return { value: t.name, label: t.name, desc: (convToolsetIsDefault.value ? '默认生效' : '当前') + ' · ' + n }
    }
    return { value: t.name, label: t.name, desc: n }
  })
  // ★ 2026-09-24 修复（与模型选择器同源问题）：生效集合不在列表里（集合文件缺失/被删/
  //   来自旧配置）时，原生 select 无匹配 option → 回落 placeholder，选择器读不出真实
  //   生效值。补一项「当前生效」保证控件恒能表达 /api/toolsets/active 返回的 effective。
  const cur = convToolset.value
  if (cur && !out.some(it => it.value === cur)) {
    out.unshift({ value: cur, label: cur, desc: '当前生效' })
  }
  return out
})
// 未收敛（集合缺失/无工具集配置）时不能留空白：明确告知「全部工具可用」
const toolsetPlaceholder = computed(() => (convToolsetEffective.value ? '选择工具集…' : '未收敛（全部工具可用）'))
const toolsetPickerTitle = computed(() => {
  const eff = convToolsetEffective.value
  if (!eff) return '当前未按工具集收敛：全部工具可用'
  return '当前生效工具集：' + eff + (convToolsetIsDefault.value ? '（默认）' : '')
})
async function loadToolsetItems() {
  try {
    const list = await api.getToolsets()
    toolsetItems.value = (list || []).filter(t => t.scope !== 'builtin' && t.name)
    // ★ 列表到达后重发一次胶囊：pluginCount 依赖本列表（首帧 sync 可能早于列表返回）
    publishConvToolsetInfo()
  } catch (e) {
    console.warn('[toolset] 工具集列表加载失败', e)
  }
}
// 同步选择器 = 当前会话「实际生效」的集合（★ 修复刷新后显示 placeholder 的问题：
// 会话未显式选择时显示后端实际生效的默认集合「基础」，而不是空白——
// 否则用户根本不知道当前对话正在使用哪个工具集）。
async function syncConvToolsetFromConv() {
  try {
    const info = await api.getActiveToolset(state.currentConvId || '', state.workspaceRoot || '')
    const selected = (info && info.selected) || ''
    const effective = (info && info.effective) || ''
    convToolsetIsDefault.value = !selected
    convToolsetEffective.value = effective
    convToolset.value = effective || selected
    convToolsetConverged.value = !!(info && info.converged)
    convToolsetDefaultName.value = (info && info.defaultName) || ''
    publishConvToolsetInfo()   // ★ 会话切换 → 场景胶囊同步为该会话的真实生效集合
  } catch (e) {
    console.warn('[toolset] 生效集合解析失败，回退会话元数据', e)
    try {
      const meta = state.currentConvId
        ? await api.getConversationMeta(state.currentConvId, state.workspaceRoot || '')
        : null
      const selected = (meta && meta.toolset) || ''
      convToolsetIsDefault.value = !selected
      convToolsetEffective.value = selected
      convToolset.value = selected
      convToolsetConverged.value = !!selected
      convToolsetDefaultName.value = ''
      publishConvToolsetInfo()
    } catch {
      convToolsetIsDefault.value = true; convToolsetEffective.value = ''; convToolset.value = ''
      convToolsetConverged.value = false; convToolsetDefaultName.value = ''
      publishConvToolsetInfo()
    }
  }
}
// 切换模式 = 只写当前会话（不动全局；后端发送消息时按会话集合收敛工具面）
async function onConvToolsetChange() {
  const convId = state.currentConvId
  const name = convToolset.value
  if (!name) return
  convToolsetEffective.value = name
  convToolsetIsDefault.value = false
  convToolsetConverged.value = true
  publishConvToolsetInfo()   // ★ 场景胶囊立即跟随（不等下一次 sync）
  if (!convId) {
    // 新对话尚未创建：暂存，首条消息建会话后写入（见 sendMessage）
    pendingConvToolset.value = name
    window.$toast && window.$toast('已选择工具集 ' + name + '（本对话首条消息生效）', 'info')
    return
  }
  try {
    await api.apiPut('/conversations/' + encodeURIComponent(convId), { toolset: name })
    window.$toast && window.$toast('本对话已切换工具集为 ' + name, 'success')
  } catch (e) {
    window.$toast && window.$toast('工具集切换失败: ' + (e.message || e), 'error')
    syncConvToolsetFromConv()   // 失败 → 回滚显示为真实生效值
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
  // ★ 2026-09-25 对齐设计稿 th129 行2（th116 h40）：钳制下限 80 → 40
  if (inputHeight.value < 40) inputHeight.value = 40
}
// ★ 2026-09-25 对齐设计稿 th129 行2：输入区默认高 150 → 40（拖拽能力保留）
const inputHeight = ref(40)
const topSentinel = ref(null)
const reviewMode = ref('auto')  // 'auto'=AI审核, 'manual'=人工审批, 'off'=全部放行
const reviewBtnTitle = computed(() => {
  const m = reviewMode.value
  return m === 'auto' ? 'AI审核：Agent自行审批写操作' : m === 'manual' ? '手动审批：每次操作需用户确认' : '关闭审核：全部放行，不经过任何审核'
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
    return '\n\n📎 目录: `' + att.path + '`（请用 glob 查看）'
  }
  return '\n\n📎 附件: `' + (att.path || att.filename) + '`（请用 read 读取）'
}

// nudge 自动清除定时器（state.nudgeByConv 的写入点见本文件右侧消息处理分支）。
// ★ 2026-09-25 扫描：本组件 nudge 展示 UI 已不存在（原 currentNudge 计算属性经证实
//   为死绑定，已摘除），state.nudgeByConv 当前无前端消费者——保留写入链路待产品决策。
let nudgeTimer = null
// ★ 2026-08-31：plan 体系已移除——currentPlan/planExpanded 下线，任务追踪只用 currentTasks。
const currentTasks = ref([])
const tasksExpanded = ref(false)

// ── 运行统计（★ 2026-09-26 收敛为「只拉取、不展示」）──
// 数据源：**后端**运行统计（state.runStatsByConv ← agent-events 的 fetchRunStats
//   经 GET /api/conversations/{id}/run-stats 拉取；后端 agent 循环埋点累加，
//   落盘 .pair/run-stats.json —— 前端不累加、不本地持久化）。
//   后端派生口径：token 速度 = 输出 token ÷ Σ LLM **生成阶段**耗时（首 chunk → 末 chunk），
//   不含工具执行 / 审批等待 / prefill 排队。
// ★ 展示位置唯一：右栏 StatsRail「运行统计」卡 —— 本组件不再重复展示任何状态/数字
//   （原消息区顶部 phase-bar 及只服务它的 computed / 函数 / CSS 已删除）。
// ★ 本组件仍负责在**会话切换 / 续跑**时调用 fetchRunStats 触发拉取
//   （见 loadConvMessages / reloadConvMessages）—— 该调用不可删：右栏卡依赖它写入的状态。

// ★ 2026-09-25（设计稿屏 1 ③）：「下一步可以试试」引导卡。
//   出现条件 = 有对话内容 + 不在生成中 + 未被关闭。
//   ★ 回填输入框用 DOM 事件而非直接改 Vue 状态：输入框的 v-model 监听原生 input 事件，
//     派发该事件即可同步 —— 无需为「填一句话」新增跨组件状态或改输入区绑定。
const NEXT_STEPS = [
  // ★ 2026-09-25 对齐设计稿 th96/th98/th100/th102：文案逐字一致
  //   （原「先带我读一遍这个项目 / 找出最近改动里的问题 / 跑一遍测试并总结结果 /
  //    给这个模块补一份文档」→ 改为设计稿的 4 条）。
  { icon: 'file-code', text: '解释这段代码', hint: '选中或指定文件，说明它的作用与关键实现' },
  { icon: 'check', text: '跑一次测试', hint: '执行测试套件，汇总失败项与原因' },
  { icon: 'bug', text: '修掉这个报错', hint: '定位报错根因并给出修复' },
  { icon: 'file-text', text: '补一份文档', hint: '按现有代码生成说明文档（含用法与示例）' },
]
const nextStepsDismissed = ref(false)
// ★ 2026-09-25 对齐设计稿 th64：任务横幅文案 = 当前任务标题（list 首项 subject）；
//   无任务时回退当前会话标题，二者皆空则不渲染该行。
const currentTaskTitle = computed(() => {
  const t = (currentTasks.value && currentTasks.value.length) ? currentTasks.value[0] : null
  if (t && (t.step || t.subject)) return t.step || t.subject
  const conv = (state.conversations || []).find(c => c.id === state.currentConvId)
  return conv ? (conv.title || '') : ''
})
const nextStepsVisible = computed(() =>
  !nextStepsDismissed.value
  && !state.chatLoading
  && Array.isArray(state.messages) && state.messages.length > 0)
function useNextStep(text) {
  try {
    const all = Array.prototype.slice.call(document.querySelectorAll('textarea'))
    const ta = all.filter(function (t) { return t.offsetParent !== null }).pop()
    if (!ta) return
    ta.value = text
    ta.dispatchEvent(new Event('input', { bubbles: true }))
    ta.focus()
  } catch (e) { }
}
try { nextStepsDismissed.value = localStorage.getItem('paircode-next-steps-dismissed') === '1' } catch (e) { }

// ★ 2026-09-26 删除：phase-bar 的文案 / 计时 / 估算进度条（phaseText、runBarTitle、
//   runTick 计时器、phaseProgress）与 phaseTimer —— 只服务已删除的顶部统计条。
//   右栏 StatsRail 自带同口径的耗时/速度计算（本组件不参与）。

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
  // 依据最早已加载消息的 _idx 判断是否还有更早消息（比 msgTotal/Loaded 更可靠）。
  // ★ Idx 负数 = 归档区（真实早期历史，后端 displayMessages 把归档原文编号为负）：
  //   归档区同样可继续向上翻，翻到归档最首时下一次请求返回空 → 由 _noMoreAbove 收敛。
  const oldestIdx = msgs[0]._idx
  return oldestIdx !== undefined && oldestIdx !== null
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
  // ★ 允许负 Idx（归档区）：此前 `oldestIdx <= 0` 会截断向上分页，
  //   前端永远看不到归档前的真实历史（只看到主文件首行的压缩摘要）。
  if (oldestIdx === undefined || oldestIdx === null) return
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
  if (/^apply_patch\b/.test(name)) return { icon: 'edit', title: '编辑文件', detail: ((args.patch || '').match(/\*\*\* (?:Add|Update|Delete) File: ([^\n]+)/) || [])[1] || '', summary: '已编辑', resultIcon: 'check' }
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
  if (/^screenshot(_(desktop|window|area))?$/.test(name)) {
    // 2026-09-12 三合一：screenshot(target=desktop/window/area)；旧名兼容
    const t = (args.target || (name === 'screenshot_window' ? 'window' : name === 'screenshot_area' ? 'area' : name === 'screenshot_desktop' ? 'desktop' : '')).toLowerCase()
    const tt = { desktop: '桌面截图', window: '窗口截图', area: '区域截图' }[t] || '截图'
    return { icon: 'image', title: tt, detail: (args.title || '').slice(0, 40), summary: '已截图', resultIcon: 'check' }
  }
  if (/^read_image\b/.test(name)) return { icon: 'image', title: '看图', detail: (args.file_path || args.path || '').slice(0, 50), summary: '已读取图片', resultIcon: 'check' }
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

// isToolErr 工具胶囊的错误色判定（原为模板内联正则，2026-09-25 提取以适配 result 裁断）。
// result 在 slim 响应中可能只带前 400 字符预览：此时后端已在裁断**之前**按全文算好
// `_err`（见 agent.Segment.Err），必须以后者为准 —— 否则关键词落在预览之外时会把
// 错误色误判成成功色（实测该会话 411 段含关键词的 result 中有 156 段属此情形）。
// `_err` 为 undefined 表示 result 未被裁断（含流式期间刚写入的段）→ 照旧跑完整内容；
// 后端侧同名正则见 cmd/companion/web_server.go 的 toolResultErrRe（逐词一致）。
const TOOL_ERR_RE = /错误|失败|error|Error|✗|Exception/
function isToolErr(seg) {
  if (!seg) return false
  if (seg._err !== undefined && seg._err !== null) return !!seg._err
  return TOOL_ERR_RE.test(seg.result || '')
}

// ── 折叠段的惰性加载（★ 2026-09-25 性能）─────────────────────────
// 首包中「折叠态不消费」的字段只带前 400 字符预览 + `_trunc`（原始字符数）标记：
//   · thinking.content  —— 折叠行只渲染「思考…」标签（content 有 v-if 守卫）
//   · tool_call.argsRaw —— 仅展开时渲染「参数」（同理由 v-if 守卫）
//   · tool_call.result  —— 折叠行只用前 120 字符（胶囊文案）/前 80 字符（未知工具
//     summary）/错误色判定，三者 400 字符预览均足够。
//     ★★ finish_task 的 result 例外（后端不裁）：它在 apiLoadAndBuildConv 里被
//     转成正文 content 段渲染，裁断会截断最终回答。
// 展开动作触发按需取全文（本地接口毫秒级返回）；请求期间保留预览、失败静默降级
// → 可见交互不变（消息摘要只用 content 段前 60 字符 + tool_call 计数，不受影响）。
// 唯一需要「跨裁断保持语义」的是错误色判定 → 见 isToolErr（后端 _err 预计算）。
// 索引口径与 /messages 完全一致（同一 limit、同一合并管道），否则会取错段。
const _segInflight = new Map()
const ensureSegFull = async (seg, msgIdx, segIdx) => {
  if (!seg || !seg._trunc) return
  const conv = state.currentConvId
  if (!conv) return
  const key = conv + '#' + msgIdx + '#' + segIdx
  const pending = _segInflight.get(key)
  if (pending) return pending
  const task = (async () => {
    try {
      const r = await api.apiGet('/conversations/' + conv + '/messages/segment', {
        idx: msgIdx, seg: segIdx, limit: 50, workspaceRoot: state.workspaceRoot,
      })
      if (r) {
        if (typeof r.content === 'string' && r.content) seg.content = r.content
        if (typeof r.argsRaw === 'string' && r.argsRaw) seg.argsRaw = r.argsRaw
        if (typeof r.result === 'string' && r.result) seg.result = r.result
      }
      seg._trunc = 0
    } catch (e) {
      // 静默降级：保留预览内容（不打断阅读、不报错）
      console.warn('[RP] 段全文加载失败 idx=%s seg=%s', msgIdx, segIdx, e)
    } finally {
      _segInflight.delete(key)
    }
  })()
  _segInflight.set(key, task)
  return task
}

// toggleThinking 思考段折叠/展开（展开时按需取全文）。
function toggleThinking(seg, msgIdx, segIdx) {
  seg._collapsed = !seg._collapsed
  if (!seg._collapsed) ensureSegFull(seg, msgIdx, segIdx)
}

// toggleTool 工具调用段折叠/展开（展开时按需取全文）。
function toggleTool(seg, msgIdx, segIdx) {
  seg._expanded = !seg._expanded
  if (seg._expanded) ensureSegFull(seg, msgIdx, segIdx)
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

// cleanMsgContent 去除消息中的标记前缀和附件尾注，只展示纯内容。
function cleanMsgContent(msg) {
  if (!msg.content) return ''
  return msg.content
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

  // ★ 新对话刚建好：补写「无会话时暂存的工具集选择」（发送前完成——后端按会话
  //   元数据收敛工具面，必须在 chatStart 之前落盘）。
  if (pendingConvToolset.value) {
    const want = pendingConvToolset.value
    pendingConvToolset.value = ''
    try {
      await api.apiPut('/conversations/' + encodeURIComponent(convId), { toolset: want })
      convToolset.value = want
      convToolsetEffective.value = want
      convToolsetIsDefault.value = false
      console.log('[RP] 新对话写入暂存工具集: %s', want)
    } catch (e) { console.warn('[toolset] 新对话写入工具集失败', e) }
  }

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
          // 读取失败 → 降级为文本附件（旧行为，提示模型用 read）
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

// ★ 2026-09-17 修复：「死会话」提交失败的恢复（用户报：已不存在会话中提交回答 400 后，
//   发送按钮仍禁用、无法继续工作）。
//   「会话不存在/未运行」= 服务端没有该运行时会话（agent 早已结束/等待回答超时/服务重启），
//   而本地可能仍残留运行中标记（如等待回答期间 chatLoading=true）——输入区会被永久锁定：
//   发送按钮被停止按钮替代、sendMessage 因 chatLoading 早退，用户无法发任何新消息。
//   分级恢复：死会话 → 复位本地运行态并标记卡片失效，让用户可直接发新消息继续；
//   其他错误（回答通道已满等）→ 保持运行态，仅提示重试。
const recoverDeadAskSession = (convId, e, seg) => {
  const msg = String((e && e.message) || e || '')
  const dead = msg.includes('会话不存在') || msg.includes('未运行')
  if (!dead) {
    window.$toast && window.$toast('回答失败：' + msg + '（请重试）', 'error')
    return
  }
  resetConvRuntime(convId)
  state.agentRunningByConv[convId] = false
  state.loadingByConv[convId] = false
  if (state.currentConvId === convId) {
    state.chatLoading = false
    state.agentRunning = false
  }
  // 历史消息里残留的 loading 占位一并清理（防止转圈动画永久残留）
  const msgs = state.messagesByConv[convId]
  if (msgs) for (const m of msgs) { if (m._loading) m._loading = false }
  // 卡片标记失效：会话已结束，反复提交必然失败
  if (seg) seg._stale = true
  window.$toast && window.$toast('该会话已结束（服务端不存在），回答无法提交。已恢复输入，可直接发送新消息继续。', 'error')
}

const submitAskAnswer = async (seg) => {
  // ★ 2026-09-17 修复：convId 为空时拦截提交（防后端 400「convId 必填」；同 sendFeedback 的判空约定）
  if (!state.currentConvId) {
    console.error('[RP] 回答提交被拦截：currentConvId 为空')
    window.$toast && window.$toast('回答失败：当前没有选中会话，请刷新页面后重试', 'error')
    return
  }
  const convId = state.currentConvId
  // ★ 陈旧卡片防护：会话已结束的卡片不再重复提交（提交按钮此时也已禁用）
  if (seg._stale) return
  // ★ Round3 ⑤：多问题 answers 数组优先，缺省回落单问题 answer（后端双兼容）
  if (seg.answers && seg.answers.length) {
    seg._answered = true
    try {
      const resp = await api.apiPost('/chat/answer', { convId, callId: seg.callId, answers: seg.answers })
      notifyLateAnswer(resp, convId)
    } catch (e) {
      console.error('[RP] 回答提交失败（多问题）:', e)
      seg._answered = false
      recoverDeadAskSession(convId, e, seg)
    }
    return
  }
  const answer = (seg.answer || '').trim()
  if (!answer) return; seg._answered = true
  try {
    const resp = await api.apiPost('/chat/answer', { convId, callId: seg.callId, answer })
    notifyLateAnswer(resp, convId)
  } catch (e) {
    console.error('[RP] 回答提交失败（单问题）:', e)
    seg._answered = false
    recoverDeadAskSession(convId, e, seg)
  }
}

// ★ 2026-09-25 提问已超时/停止后提交的回答：后端不再静默丢弃，而是落盘为一条
//   会话消息（响应带 recorded='message'）。此处提示用户并尝试刷新历史；
//   运行中会话的 reload 会被 reloadConvMessages 跳过（刷新页面即可见）。
const notifyLateAnswer = (resp, convId) => {
  if (!resp || resp.recorded !== 'message') return
  window.$toast && window.$toast('该提问已结束（超时/停止），你的回答已记录为会话消息（刷新页面可见）', 'info')
  Promise.resolve(reloadConvMessages(convId)).catch(() => {})
}

const resolveApproval = async (approved) => {
  // 兼容两种调用：直接传 boolean（旧）或 { approved, reply }（新）
  const isApproved = typeof approved === 'object' ? approved.approved : approved
  const reply = typeof approved === 'object' ? (approved.reply || '') : ''
  const convId = state.currentConvId
  const a = state.approvalByConv[convId]
  if (!a || !a.callId || !a.waiting) return
  a.waiting = false
  try {
    await api.apiPost('/chat/approve', { convId, approved: isApproved, reply })
  } catch (e) {
    console.error('[RP] 审批提交失败:', e)
    a.waiting = true
    window.$toast && window.$toast('审批失败：' + ((e && e.message) || e) + '（可能任务已结束，请刷新后重试）', 'error')
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
    state.conversations = visibleConversations(list)
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
    const mergedMsgs = mergeConsecutiveAssistant(loaded)
    // ══════════════════════════════════════════════════════════════
    // ★★ 2026-09-25 性能修复（首屏 13s → 亚秒级）：**在消息交给渲染之前**注入折叠标记 ★★
    //   原实现只在 switchConv 末尾调用 applyAutoCollapse()，而那里已被
    //   `await loadConvTasks()` / `await loadAutopilotRounds()`（实测 /autopilot/rounds
    //   对大对话返回 4.3MB、耗时 4.1s）推到很后面。于是首屏先按「全展开」渲染了一整遍：
    //   实测 1547 个 MarkdownRenderer / 46238 个 DOM 节点 / 391 万字符，Vue 同步渲染
    //   阻塞主线程约 13 秒（CPU profile：(program) 11.1s + GC 3.5s），随后
    //   applyAutoCollapse 才把 _folded 置 true → 整批 DOM 被丢弃重渲染（DOM 骤降至 1674）。
    //   现改为在**数据源头**同步注入 → 首次渲染即折叠态（实测 26 个 MarkdownRenderer），
    //   渲染量降约 98%，并与调用方（switchConv / reloadConvMessages）解耦。
    //   ★ 幂等安全：applyAutoCollapse 只对 _folded === undefined 的消息生效；
    //     autoCollapse 关闭时函数首行直接 return（用户选择「不自动折叠」的行为不变）。
    // ══════════════════════════════════════════════════════════════
    applyAutoCollapse(mergedMsgs)
    return { mergedMsgs, total: data.total || loaded.length }
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
  // ★ 运行统计（后端真源）：会话切换/续跑后拉取该会话最近一次运行结果
  fetchRunStats(convId)
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
    // ★ 桥专用会话（conv_wx_*）不进 PC 端列表：与加载路径同口径过滤
    const visible = visibleConversations(list)
    for (const m of visible) {
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
// ── 自主模式监督看板状态（数据源见 loadAutopilotRounds）──
const autopilotRounds = ref([])
const autopilotExpanded = ref(true)

// loadConvTasks 拉取指定会话的任务清单（TaskPanel 数据源）。
// ★ 2026-09-12 修复「刷新页面/切换对话后任务面板空白」：任务按会话持久化在
//   工作区 `.pair/tasks/*.json`（task.convId），前端此前只有运行时 WS 事件
//   更新它 —— 刷新或切会话后无事件可补，面板恒空白。切会话必须重拉。
// 竞态保护：返回时该会话已不是当前会话则丢弃结果（快速连点不串味）。
const loadConvTasks = async (convId) => {
  if (!convId) { currentTasks.value = []; return }
  try {
    const taskData = await api.apiGet('/tasks', { convId })
    if (state.currentConvId !== convId) return
    const list = (taskData && Array.isArray(taskData.tasks)) ? taskData.tasks : []
    currentTasks.value = list.map(t => ({
      step: t.step || t.subject || '',
      status: t.status,
      _taskId: t.taskId || t.id,
    }))
  } catch (e) {
    console.warn('[RP] loadConvTasks 失败 conv=%s', convId, e)
    if (state.currentConvId === convId) currentTasks.value = []
  }
}

// loadAutopilotRounds 拉取指定会话的监督回合列表（自主模式看板数据源）。
// ★ 数据面由 autopilot 插件提供（GET /api/autopilot/rounds?convId=…，记录落盘
//   .pair/autopilot/<convId>.jsonl）；插件未装载/从未监督过时接口 404 或 rounds 为空
//   → 静默置空（看板不显示）。竞态保护：返回时已切换会话则丢弃结果。
// ★ 2026-09-25 性能（并发请求合并）：同一会话的请求在途时复用同一 Promise，所有
//   等待方共享结果。实测首屏该接口被请求 5 次（单次 4.6MB → 合计约 23MB 传输）：
//   本组件有 4 处触发点（switchConv / WS 重连 / 会话结束补拉 / mounted nextTick），
//   其中「mounted 的 nextTick 补拉」与「switchConv 内的调用」在首屏几乎同帧触发
//   （见下方 onMounted 的 switchConv 与 L2419），叠加 autopilot 插件 client 半的
//   首拉 → 同一份数据被并发拉了多遍。合并后并发只发 1 次请求。
//   各调用点语义不变：拿到的一定是该会话的最新快照（合并窗口仅覆盖「请求在途」期间，
//   一旦返回即从表中移除，后续触发（如 WS 重连补拉）仍会真实发起新请求）。
const _roundsInflight = new Map()
const loadAutopilotRounds = async (convId) => {
  if (!convId) { autopilotRounds.value = []; return }
  const pending = _roundsInflight.get(convId)
  if (pending) return pending
  const task = (async () => {
    try {
      const res = await api.apiGet('/autopilot/rounds', { convId })
      if (state.currentConvId !== convId) return
      autopilotRounds.value = (res && Array.isArray(res.rounds)) ? res.rounds : []
    } catch (e) {
      // 静默：插件未装载或接口不可用时看板置空（不打扰用户）
      if (state.currentConvId === convId) autopilotRounds.value = []
    } finally {
      _roundsInflight.delete(convId)
    }
  })()
  _roundsInflight.set(convId, task)
  return task
}

// roundKey 监督回合唯一键（★ round 是「本次运行内的监督序号」，宿主每次 Run 从 1 重数，
// 同一会话多次运行会出现相同 round → 仅按 round 去重会覆盖历史，见 onPluginEvent）。
const roundKey = (r) => String((r && r.startedAt) || '') + '#' + String((r && r.round) || '')

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
  autopilotRounds.value = []

  // 加载 token 统计
  try {
    const ts = await api.apiGet('/conversations/' + id + '/token-stats' + (state.workspaceRoot ? '?workspaceRoot=' + encodeURIComponent(state.workspaceRoot) : ''))
    if (ts && ts.promptTokens !== undefined) Object.assign(getConvCtxStats(id), ts)
  } catch {}

  // 若该会话尚未完成过一次历史加载，从 API 加载。
  // ★ 2026-09-12 修复「慢启动时历史消息不出现」：原判定为 hasRealMsgs（messagesByConv
  //   里存在非 _loading 消息即视为「已有内容」）。但 WS 门控的兜底定时器
  //   （agent-events.js WS_PENDING_MAX_MS=3000）会在本函数开始之前就把快照/事件直接
  //   写入 messagesByConv（写入的消息 _loading=false），于是判定为「已有内容」→
  //   跳过本加载 → 历史永不出现（正是门控要防的「刷新后只有当前 ws 消息」）。
  //   首屏插件装载慢（本函数开始前常已超 3000ms）时必现。
  //   改为以「是否加载过历史」为准：msgLoadedByConv 有该键即加载过（刷新/换工作区
  //   会清空该表 → 刷新后必然重新加载）；切换会话时行为不变（已加载过则跳过）。
  const historyLoaded = Object.prototype.hasOwnProperty.call(state.msgLoadedByConv, id)
  if (!historyLoaded) {
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
      // ★ 运行统计（后端真源）：切换到该会话即拉取其最近一次运行结果
      //   （运行中的会话由 agent-events 的轮询持续刷新）
      fetchRunStats(id)
    } else {
      state.msgTotalByConv[id] = 0
      state.msgLoadedByConv[id] = 0
    }
  }

  // 加载任务状态（★ 任务按会话持久化在 .pair/tasks/*.json，见 loadConvTasks）
  await loadConvTasks(id)
  // 加载自主模式监督回合（看板；★ 按会话持久化在 .pair/autopilot/*.jsonl，见 loadAutopilotRounds）
  // ★ 2026-09-25 性能：改为**后台执行**（原为 await）—— 实测该接口对大对话返回 4.3MB /
  //   耗时 4.1s，await 会把紧随其后的首屏滚底（fillViewport / forceScrollToBottom）
  //   整体推迟 4 秒以上，表现为「切会话后要等好几秒才滚到底」。看板数据只喂右栏
  //   AutopilotPanel，与消息显示无关，无需阻塞主流程。
  loadAutopilotRounds(id)

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

// ★ 2026-09-26 删除 phaseIcon()：只服务 phase-bar 的阶段图标；phase 事件已无生产方
//   （state.phaseByConv 恒空），函数恒不命中。

// ★ 2026-09-25 删除 handleTaskTool()：全域零引用的死函数（模板/脚本段/跨文件均无调用，
//   函数体只是「命中任务工具名则 return true」且调用方从未使用返回值）。


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
  inputHeight.value = Math.max(40, Math.min(600, newH))
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
    // 外部文件无法获得工作区相对路径，agent 无法 read，提示用户
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
        // 图片保留 dataURL（图片无法用 read 读取）
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
      // ★ 2026-09-12 重连补偿：断线期间 agent 推送的 update_tasks 事件已丢失
      //   （任务面板会停在断开前状态）→ 重连即从服务端重拉该会话任务清单
      //   （任务持久化在 .pair/tasks，不依赖 WS 事件）。
      loadConvTasks(id)
      // 同上：断线期间的 ui:autopilot:round 事件已丢失 → 重连即重拉监督回合
      loadAutopilotRounds(id)
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

  // ── 监督回合实时追加（自主模式看板）：autopilot 插件每完成一次监督回合即
  //    ctx.emit('ui:autopilot:round', record) → 宿主 client 事件队列 → plugin-runtime
  //    dispatchHostEvent 在 window 上广播 pair-plugin-event → 此处按 round 去重追加。
  //    （主界面非插件实例，借广播消费 ui: 事件；完全不可用时由 loadAutopilotRounds 兜底）
  const onPluginEvent = (e) => {
    const ev = e && e.detail
    if (!ev || ev.name !== 'ui:autopilot:round') return
    const rec = ev.payload
    if (!rec || !rec.convId || rec.convId !== state.currentConvId) return
    // ★ 唯一键 = startedAt#round：记录里的 round 是「本次运行的第 N 次监督」
    //   （宿主每次 Run 从 1 重新计数）→ 同一会话多次运行会出现 round 相同的记录，
    //   仅按 round 去重会把新一轮覆盖掉上一轮（看板不累加、历史丢失）。
    const idx = autopilotRounds.value.findIndex(r => roundKey(r) === roundKey(rec))
    if (idx >= 0) {
      const next = [...autopilotRounds.value]
      next.splice(idx, 1, rec)
      autopilotRounds.value = next
    } else {
      autopilotRounds.value = [...autopilotRounds.value, rec]
    }
  }
  window.addEventListener('pair-plugin-event', onPluginEvent)

  // 自主模式收尾补拉：chatLoading 由 true→false 表示本会话本轮已结束（含监督者裁决 done）
  // → 从服务端补拉监督回合，避免事件丢失导致看板缺轮（continue 轮次由事件实时补）。
  watch(() => state.chatLoading, (now, prev) => {
    if (prev && !now && state.currentConvId) loadAutopilotRounds(state.currentConvId)
  })

watch(() => state.settings, (s) => { if (s) { autoIterate.value = !!s.autoIterateOnRejection; autonomous.value = !!s.autonomous; autoCollapse.value = s.autoCollapse !== undefined ? !!s.autoCollapse : true; } }, { immediate: true })

// ★ 2026-08-31 会话级模型：下拉不再跟随全局 settings（切模型只写会话）。
//   会话切换/新建时按会话元数据同步下拉；服务商配置（Key/模型列表）变化时刷新分组。
watch(() => state.currentConvId, () => { syncComposerModelFromConv() })
// ★ 2026-09-20：全局默认配置的判据改为「激活配置名」（settings.preset）——
//   连接字段（provider/executeModel）已退出 settings 顶层（迁入 ai-presets.json 并清空），
//   监听它们将永不触发（应用另一条配置后下拉不刷新）。
watch(() => state.settings && state.settings.preset, () => {
  // 全局默认配置变化：仅当当前会话未设置模型时下拉才需要刷新显示
  loadModelData().then(() => syncComposerModelFromConv())
})
// ★ 2026-09-04 工具集（通用集合）会话级：切换会话时同步实际生效集合；列表惰性加载
watch(() => state.currentConvId, () => {
  pendingConvToolset.value = ''   // 切换会话：丢弃上一个「尚未创建」对话的暂存选择
  syncConvToolsetFromConv()
})
watch(() => state.workspaceRoot, () => { loadToolsetItems().then(() => syncConvToolsetFromConv()) })


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

  // ⚡ 初始加载：若已有当前对话，从 API 加载任务状态（走统一入口，含竞态保护）
  // （页面刷新或从其他工作区切换回来时，currentTasks 为空，需要从 TaskManager 恢复）
  nextTick(() => { if (state.currentConvId) loadConvTasks(state.currentConvId) })
  // 同上：刷新页面后监督看板从服务端恢复（.pair/autopilot/*.jsonl）
  nextTick(() => { if (state.currentConvId) loadAutopilotRounds(state.currentConvId) })

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
  if (nudgeTimer) { clearTimeout(nudgeTimer); nudgeTimer = null }
  window.removeEventListener('pair-plugin-event', onPluginEvent)
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
/* 会话列表已收起：按钮半透明提示「当前为隐藏态」*/ 
.rp-btn-off { opacity: 0.45; }
.rp-body { flex: 1; display: flex; flex-direction: row; overflow: hidden; min-height: 0; }
/* ★ 2026-09-25 对齐设计稿 th130（col 840, pad=space.md）：主区内容列左右内边距 12px。
   宽度不硬编码 —— 由视口推导：1440 下 840−24=816；1280 下 680−24=656。 */
.chat-area { flex: 1; display: flex; flex-direction: column; min-width: 0; overflow: hidden; max-width: 100%; padding: 12px; box-sizing: border-box; }
/* 内边距已上提到 .chat-area：消息区自身不再叠加左右内边距（否则双重缩进）。 */
.chat-messages { flex: 1; overflow-y: auto; padding: 0; min-height: 0; position: relative; overflow-anchor: none; }
.msg-list-wrap { display: flex; flex-direction: column; gap: 14px; min-height: 100%; }
/* ★ 2026-09-25（设计稿屏 1 ③）：「下一步可以试试」引导卡（消息流末尾的建议芯片） */
.next-steps {
  margin: 4px 0 8px; padding: 10px 12px;
  border-radius: var(--radius-md);
  border: 1px dashed var(--color-border-strong);
  background: var(--color-surface);
}
/* ★ 2026-09-25 对齐设计稿 th95：标题行 — 标题(600,fg) + 右侧 muted xs 提示 + 关闭 */
.ns-head { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
/* ★ th92：标题 sm + 600 + fg */
.ns-title { font-size: var(--fs-sm); font-weight: 600; color: var(--text-primary); }
/* ★ th94：右侧 muted xs「也可以直接描述你的目标」 */
.ns-hint { margin-left: auto; font-size: var(--fs-xs); color: var(--text-muted); }
.ns-close {
  border: none; background: none; color: var(--text-muted); cursor: pointer;
  font-size: 15px; line-height: 1; padding: 0 3px;
}
.ns-close:hover { color: var(--text-primary); }
/* ★ 2026-09-25 对齐设计稿 th104（芯片行 792×32, gap=8）与 th97/99/101/103（芯片 h=32, r=full） */
.ns-chips { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.ns-chip { height: 32px; padding: 0 10px; border-radius: 999px; font-size: var(--fs-xs); }
/* ★ th97：首个芯片 accent 高亮（border=accent / accent 文字 / surface-3 底） */
/* ★ 提权：原 `.ns-chip-primary` 被既有 `.ns-chip { color: … }` 覆盖（实测 color=fg），
   改为双类选择器提高特异性，确保文字色真正等于当前主题 --accent。 */
.ns-chip.ns-chip-primary { border-color: var(--accent); color: var(--accent); background: var(--bg-active); }
/* ★ th65/th64：任务横幅行 816×48 + 内嵌 440×40 卡片 */
.task-banner-row { display: flex; align-items: center; height: 48px; flex: 0 0 auto; }
.task-banner-card {
  margin-left: auto; width: 440px; height: 40px; box-sizing: border-box;
  padding: 0 8px; display: flex; align-items: center;
  background: var(--bg-active); border: 1px solid var(--border-color); border-radius: 8px;
  font-size: var(--fs-sm); color: var(--text-primary);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
/* ★ th105：底部提示行 xs muted */
.ns-foot { margin-top: 8px; font-size: var(--fs-xs); color: var(--text-muted); }
/* ★ th106：卡片 816×140, bg=surface, r12, border, pad=12 */
.next-steps { border-radius: 12px; padding: 12px; background: var(--bg-secondary); border: 1px solid var(--border-color); }
.ns-chip {
  display: inline-flex; align-items: center; gap: 6px; padding: 5px 10px;
  border: 1px solid var(--border-color); border-radius: var(--radius-full);
  background: var(--color-surface-2); color: var(--text-primary);
  font-size: var(--fs-sm); cursor: pointer; transition: all .15s;
}
.ns-chip:hover { border-color: var(--accent); background: var(--color-accent-bg); }
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
.chat-empty-icon { margin-bottom: 16px; opacity: 0.35; color: var(--accent); filter: drop-shadow(0 0 12px var(--color-accent-ring)); }
.chat-empty-text { font-size: 16px; font-weight: 500; margin-bottom: 6px; color: var(--text-secondary); }
.chat-empty-hint { font-size: 13px; opacity: 0.7; }
/* ★ P4-3 首启引导卡：空对话位置的三步动线（.chat-empty 为居中容器，卡片自身左对齐排版） */
.ob-card { width: min(430px, 92%); text-align: left; background: var(--color-surface); border: 1px solid var(--border-color); border-radius: 10px; padding: 14px 16px; box-shadow: var(--shadow-lg); }
.ob-head { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }
.ob-logo { width: 34px; height: 34px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; border-radius: 9px; color: var(--accent); background: color-mix(in srgb, var(--accent) 12%, transparent); }
.ob-title { font-size: 14px; font-weight: 600; color: var(--text-primary); }
.ob-sub { font-size: 11px; color: var(--text-muted); margin-top: 2px; }
.ob-steps { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; }
.ob-step { display: flex; align-items: center; gap: 9px; padding: 7px 8px; border-radius: 7px; }
.ob-step:hover { background: color-mix(in srgb, var(--accent) 6%, transparent); }
.ob-no { width: 18px; height: 18px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; border-radius: 50%; font-size: 10px; font-weight: 600; color: var(--text-muted); border: 1px solid var(--border-color); }
.ob-step.is-done .ob-no { color: var(--accent); border-color: transparent; background: color-mix(in srgb, var(--accent) 16%, transparent); }
.ob-body { display: flex; flex-direction: column; min-width: 0; gap: 1px; }
.ob-t { font-size: 12px; font-weight: 500; color: var(--text-secondary); }
.ob-step.is-done .ob-t { color: var(--text-muted); }
.ob-d { font-size: 11px; color: var(--text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ob-act { margin-left: auto; flex-shrink: 0; padding: 3px 9px; font-size: 11px; border: 1px solid color-mix(in srgb, var(--accent) 45%, transparent); border-radius: 5px; background: color-mix(in srgb, var(--accent) 10%, transparent); color: var(--accent); cursor: pointer; }
.ob-act:hover { background: color-mix(in srgb, var(--accent) 20%, transparent); }
.ob-foot { display: flex; justify-content: flex-end; margin-top: 8px; padding-top: 8px; border-top: 1px solid var(--border-color); }
.ob-dismiss { border: none; background: transparent; color: var(--text-muted); font-size: 11px; cursor: pointer; padding: 2px 4px; border-radius: 4px; }
.ob-dismiss:hover { color: var(--text-secondary); background: color-mix(in srgb, var(--accent) 8%, transparent); }
.msg-user { flex-direction: row-reverse; justify-content: flex-start; gap: 10px; }




.msg-user .bubble-user {
  flex: 0 0 auto;
  max-width: 80%;
  min-width: 40px;
  /* ★ 2026-08-21 可视优化：纯色蓝底 → accent 渐变 + 柔和阴影，保持主题 accent 色调 */
  background: linear-gradient(135deg, var(--accent) 0%, var(--accent-light) 100%);
  color: var(--color-accent-fg);
  padding: 10px 16px;
  border-radius: 14px 14px 4px 14px;
  overflow-wrap: break-word;
  word-break: break-word;
  overflow-wrap: anywhere;
  margin: 2px 0;
  box-shadow: var(--shadow-sm);
  transition: box-shadow 0.15s ease, transform 0.1s ease;
  position: relative;
}
.msg-user .bubble-user:hover {
  box-shadow: var(--shadow-sm);
}
/* 选中文字在深色气泡上可见 */
.msg-user .bubble-user ::selection {
  background: rgba(255, 255, 255, 0.3);
  color: var(--color-accent-fg);
}
.user-msg-content { width: 100%; text-align: left; }
.user-msg-content :deep(p) { margin: 4px 0; white-space: pre-wrap; word-break: break-word; }
.user-msg-content :deep(p:first-child) { margin-top: 0; }
.user-msg-content :deep(p:last-child) { margin-bottom: 0; }
.user-msg-content :deep(pre) { white-space: pre-wrap; font-size: 12px; background: rgba(0,0,0,0.15); padding: 6px 8px; border-radius: 4px; max-width: 100%; overflow-x: auto; margin: 4px 0; }
.user-msg-content :deep(code) { font-size: 12px; }
.user-msg-header { margin-bottom: 6px; display: flex; align-items: center; gap: 6px; }
.umh-badge { display: inline-flex; align-items: center; gap: 3px; font-size: 10px; padding: 1px 6px; border-radius: 3px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; }
.badge-delegation { background: var(--color-cat-purple-bg); color: var(--color-cat-purple); border: 1px solid var(--color-cat-purple-bg); }
.badge-feedback { background: var(--color-cat-amber-bg); color: var(--color-cat-amber); border: 1px solid var(--color-cat-amber-bg); }
.umh-agent { font-size: 11px; color: var(--text-muted); background: var(--bg-tertiary); padding: 1px 6px; border-radius: 3px; }
.user-msg-placeholder { color: var(--color-subtle); font-style: italic; font-size: 12px; }
.msg-avatar { width: 28px; height: 28px; border-radius: 50%; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.msg-user .msg-avatar { background: linear-gradient(135deg, var(--accent) 0%, var(--accent-light) 100%); color: var(--color-accent-fg); box-shadow: 0 1px 4px var(--color-accent-ring); }
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
.bubble-user .msg-time { color: var(--color-muted); }
.msg-loading-dots { display: flex; align-items: center; gap: 3px; padding: 8px 12px; }
.msg-loading-dots .dot { width: 6px; height: 6px; border-radius: 50%; background: var(--text-muted); animation: dotPulse 1.4s infinite; }
.msg-loading-dots .dot:nth-child(2) { animation-delay: 0.2s; }
.msg-loading-dots .dot:nth-child(3) { animation-delay: 0.4s; }
@keyframes dotPulse { 0%, 60%, 100% { opacity: 0.3; transform: scale(0.8); } 30% { opacity: 1; transform: scale(1.2); } }
.msg-loading-banner { display: flex; align-items: center; justify-content: center; gap: 8px; padding: 8px; color: var(--text-muted); font-size: 12px; }
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
.tl-dot-thinking { border-color: var(--accent); background: var(--accent-bg); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px var(--color-cat-amber-bg); }
.tl-dot-tool { border-color: var(--color-cat-amber); background: var(--color-cat-amber-bg); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px var(--color-cat-amber-bg); }
.tl-dot-ask { border-color: var(--color-cat-purple); background: var(--color-cat-purple-bg); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px var(--color-cat-purple-bg); }
.tl-dot-content { border-color: var(--color-cat-green); background: var(--color-cat-green-bg); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px var(--color-cat-green-bg); }
.tl-dot-done { border-color: var(--accent); background: var(--accent); box-shadow: 0 0 0 2px var(--bg-primary), 0 0 6px var(--color-cat-blue-bg); }
.tl-body { flex: 1; min-width: 0; font-size: 13px; line-height: 1.6; }
/* ── 思考段：背景区分 + 左边框 + 改进滚动条 ── */
.tl-think-body { position: relative; }
.tl-thinking-text { color: var(--text-secondary); font-style: italic; white-space: pre-wrap; padding: 6px 10px; max-height: 300px; overflow-y: auto; background: var(--bg-tertiary); border-radius: 6px; border-left: 2px solid var(--accent); margin: 2px 0; }
.tl-think-fold { position: sticky; bottom: 0; display: inline-block; font-size: 11px; color: var(--accent); cursor: pointer; padding: 3px 10px; margin-top: 4px; user-select: none; background: var(--bg-tertiary); border: 1px solid var(--border-color); border-radius: 4px; transition: background 0.15s; }
.tl-think-fold:hover { background: var(--bg-hover); color: var(--accent-light); }
.tl-thinking-collapsed { color: var(--text-muted); font-style: italic; font-size: 12px; cursor: pointer; padding: 2px 0; }
/* ★ 2026-09-26 用户反馈修正：工具行不再用「贴最右的状态胶囊(tag)」——
   名称自适应宽度（原固定 240px 造成中间留白）、结果摘要**左对齐紧随其后**
   （文字色区分成功/错误/运行中，无背景无圆角；原 margin-left:auto 把胶囊推到行尾），
   chevron 紧随内容之后。行本体仍是 th83/th90 的行式（h32, bg=surface-2, r8, border）。
   旧的 .tl-tc-header 规则保留（思考段等仍在复用），不受影响。 */
.tool-row-wrap { display: flex; flex-direction: column; gap: 4px; margin: 2px 0; }
.tool-row {
  display: flex; align-items: center; height: 32px; padding: 0 6px; gap: 8px;
  background: var(--bg-secondary); border: 1px solid var(--border-color);
  border-radius: 8px; cursor: pointer; transition: border-color .15s;
}
.tool-row:hover, .tool-row.is-open { border-color: var(--accent); }
.tr-icon { flex: 0 0 auto; color: var(--accent); }
.tr-name {
  flex: 0 1 auto; max-width: 220px; font-size: var(--fs-xs); color: var(--text-primary);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.tr-spinner {
  flex: 0 0 auto; width: 10px; height: 10px; border: 2px solid var(--accent);
  border-top-color: transparent; border-radius: 50%; animation: tr-spin .8s linear infinite;
}
@keyframes tr-spin { to { transform: rotate(360deg); } }
/* 结果摘要：左对齐、紧随工具名成组（不再吃满剩余宽度把 chevron 顶到最右）。
   max-width 62% 保证与名称共处一行且可省略；无背景/无圆角 → 不是 tag。 */
.tr-status {
  flex: 0 1 auto; min-width: 0; max-width: 62%; font-size: var(--fs-xs);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap; text-align: left;
}
.tr-status-ok { color: var(--success, #4FD8A4); }
.tr-status-err { color: var(--danger, #FF8085); }
.tr-status-run { color: var(--text-muted); }
/* ★ 2026-09-25 对齐设计稿 th70/th69：助手消息头行（h24）与模型胶囊（124×24, r=full） */
.msg-head { display: flex; align-items: center; gap: 8px; height: 24px; margin-bottom: 4px; }
.mh-icon { flex: 0 0 auto; color: var(--accent); }
.mh-brand { flex: 0 0 auto; font-size: var(--fs-sm); font-weight: 600; color: var(--text-primary); }
.mh-model-pill {
  flex: 0 0 auto; display: inline-flex; align-items: center; justify-content: center;
  /* ★ th68：设计为 124×24 固定胶囊（原为内容自适应 99 宽） */
  min-width: 124px; height: 24px; padding: 0 10px; border-radius: 999px;
  background: var(--bg-active); color: var(--accent); font-size: var(--fs-xs);
}
/* ★ th71/th72 正文改 sm 字号（原继承值偏小） */
/* ★ 2026-09-25 对齐设计稿 th76（816 = 整内容列）：助手消息整块占满内容列 ——
   原通用 .msg-bubble{max-width:85%} 把助手头行(th70)/工具行(th83/90)压到 537（内容列 656）。
   ★ 就地改本规则（不新增同名规则）；用户气泡走 .msg-user，仍保持 85% 右对齐。
   ★ th75 命令卡 520×64 亦在此宽度内居左。 */
.msg-assistant .msg-bubble { font-size: var(--fs-sm); max-width: 100%; }
/* ★ 2026-09-25 对齐设计稿 th129（输入融合卡 816×128）
   notes §15.2「单一容器包裹三段」：输入区去掉自带描边与底色，描边/底色**上提到外层容器**。
   三段高度自洽：pad8 + 行1(24) + gap8 + 行2(40) + gap8 + 行3(32) + pad8 = 128。
   ★ 监督纠正：融合卡定义**合并进下方唯一的老规则 `.input-wrapper`**，
     本处不再保留第二条同名规则（此前同特异性双规则 → 老规则胜出，样式反复不生效）。 */
/* 行1（th114）：h24 — 模型胶囊 + 场景工具集选择器 + 弹性空隙 */
.input-row1 { display: flex; align-items: center; gap: 8px; height: 24px; flex: 0 0 auto; }
.ir-pill {
  display: inline-flex; align-items: center; justify-content: center;
  height: 24px; padding: 0 10px; border-radius: 999px;
  font-size: var(--fs-xs); overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
/* th107：模型胶囊 bg=surface-3 + accent 文字 */
.ir-pill-model { background: var(--bg-active); color: var(--accent); }
/* ★ 2026-09-25 监督纠正（缺陷 2）：行1 场景工具集选择器 = SheetPicker 胶囊化外观。
   对齐设计稿 th109（bg=surface-2/hover、muted 文字、radius=full、xs 字号、h24），
   使可交互的选择器与行1 其它胶囊（模型/自主模式）视觉一致。 */
.ir-pill-picker { flex: 0 0 auto; }
.ir-pill-picker :deep(.sp-wrap) { height: 24px; }
.ir-pill-picker :deep(.sp-select) {
  height: 24px;
  padding: 0 20px 0 10px;
  border: none;
  border-radius: 999px;
  background: var(--bg-hover);
  color: var(--text-muted);
  font-size: 11px;
  font-weight: 600;
  max-width: 160px;
  cursor: pointer;
}
.ir-pill-picker :deep(.sp-select):hover { color: var(--text-primary); }
.ir-pill-picker :deep(.sp-chevron) { right: 6px; pointer-events: none; }
.ir-spacer { flex: 1 1 auto; min-width: 0; }
/* 行2（th116）：输入区本身 —— 去掉自带 border/background，高度 40，字号 sm */
.input-wrapper .chat-input {
  border: 0 !important; background: transparent !important;
  height: 40px; min-height: 40px; font-size: var(--fs-sm);
  padding: 8px 0; box-sizing: border-box;
}
.input-wrapper .chat-input:empty::before { color: var(--text-muted); font-size: var(--fs-sm); }
/* 行3（th128）：h32 — 3 个 32×32 r8 图标卡 + 空隙 + Enter 提示 + 发送 72×32 */
.input-wrapper .input-bottom-bar { height: 32px; gap: 8px; }
.input-wrapper .ibb-btns { height: 32px; gap: 8px; align-items: center; }
.input-wrapper .ibb-btns > * { flex: 0 0 auto; }
.input-wrapper .obtn {
  display: inline-flex; align-items: center; justify-content: center; gap: 4px;
  width: 32px; height: 32px; border-radius: 8px; padding: 0;
}
.input-wrapper .obtn-sep { display: none; }
.input-wrapper .send-btn, .input-wrapper .stop-btn { width: 72px; height: 32px; border-radius: 8px; }
/* ★ th127：Enter 提示（xs muted），靠右置于发送按钮左侧 */
.ibb-enter-hint { margin-left: auto; font-size: var(--fs-xs); color: var(--text-muted); white-space: nowrap; flex: 0 0 auto; }
.input-wrapper .send-btn { font-size: var(--fs-sm); }
.tr-chevron { flex: 0 0 auto; color: var(--text-muted); }
.tool-row-detail { margin-top: 4px; }
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
.tl-tc-command { background: var(--color-canvas); color: var(--color-fg); padding: 6px 10px 6px 14px; border-radius: 4px; font-family: var(--font-code); font-size: 12px; white-space: pre-wrap; border: 1px solid var(--border-color); }
.tl-tc-output { background: var(--color-canvas); color: var(--color-cat-green); padding: 8px 10px; border-radius: 4px; font-family: var(--font-code); font-size: 11px; white-space: pre-wrap; max-height: 200px; overflow: auto; border: 1px solid var(--border-color); }
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
  background: var(--accent); color: var(--color-accent-fg);
  border: none; font-size: 12px; cursor: pointer;
  box-shadow: var(--shadow-sm);
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
  0%, 100% { box-shadow: var(--shadow-sm); }
  50% { box-shadow: var(--shadow-md); }
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
  background: var(--color-warning-bg);
  border: 1px solid var(--color-warning-bg);
  border-radius: 6px;
  font-size: 12px; color: var(--color-warning);
}
.resume-icon { flex-shrink: 0; font-size: 13px; }
.resume-text { flex: 1; line-height: 1.4; }
.resume-btn {
  flex-shrink: 0;
  display: inline-flex; align-items: center; gap: 4px;
  padding: 3px 10px; border-radius: 4px;
  background: var(--color-warning-bg);
  border: 1px solid var(--color-warning);
  color: var(--color-warning); font-size: 11px; font-weight: 500;
  cursor: pointer; white-space: nowrap;
}
.resume-btn:hover { background: var(--color-warning-bg); }
.input-resizer { position: absolute; top: -8px; left: 0; right: 0; height: 12px; cursor: ns-resize; z-index: 10; }
/* ★ 2026-09-25 对齐设计稿 th129：融合卡**唯一**规则 ——
   必须带 display:flex + flex-direction:column，否则 gap:8px 不生效（三段之间无 8px 衔接）。 */
.input-wrapper { position: relative; display: flex; flex-direction: column; background: var(--color-surface-2); border: 1px solid var(--color-input-border); border-radius: 12px; padding: 8px; gap: 8px; width: 100%; box-sizing: border-box; transition: border-color .15s ease, box-shadow .15s ease, background .15s ease; box-shadow: var(--shadow-sm); }
/* ★ Round3 ④.2 slash 命令菜单（输入 "/" 前缀时显示在输入框上方） */
.slash-menu { position: absolute; left: 0; right: 0; bottom: 100%; margin-bottom: 6px; background: var(--panel-bg); border: 1px solid var(--border-color); border-radius: 10px; box-shadow: var(--shadow-lg); max-height: 280px; overflow-y: auto; z-index: 60; padding: 4px; }
.slash-item { display: flex; align-items: baseline; gap: 10px; padding: 7px 10px; border-radius: 7px; cursor: pointer; }
.slash-item.active { background: var(--color-accent-bg); }
.slash-name { font-family: var(--mono-font); color: var(--color-accent); font-size: 13px; flex-shrink: 0; }
.slash-name .slash-ondemand { font-style: normal; font-size: 10px; color: var(--color-warning); border: 1px solid var(--color-warning-bg); border-radius: 4px; padding: 0 4px; margin-left: 6px; vertical-align: 1px; }
.slash-desc { color: var(--text-muted); font-size: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
/* ★ 2026-08-21 可视优化：输入框聚焦时 accent 描边（键盘可达性/审美） */
.input-wrapper:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--focus-ring), 0 4px 16px var(--color-scrim); }
/* ★ 2026-08-22 contenteditable 输入框改造：附件内联 tag 渲染在输入框内 */
.chat-input { display: block; width: 100%; background: transparent; border: none; color: var(--text-primary); padding: 16px 18px 8px 18px; border-radius: 0; font-size: 14px; resize: none; outline: none; min-height: 80px; font-family: inherit; line-height: 1.6; box-sizing: border-box; overflow-y: auto; white-space: pre-wrap; word-break: break-word; cursor: text; }
/* placeholder（contenteditable 无原生 placeholder，用 :empty 伪元素） */
.chat-input:empty::before { content: attr(data-placeholder); color: var(--text-muted); pointer-events: none; }
.chat-input[contenteditable="false"] { opacity: 0.55; cursor: not-allowed; }
/* 附件内联 tag：contenteditable=false 药丸，与文字混排 */
.att-inline { display: inline-flex; align-items: center; gap: 4px; padding: 1px 6px 1px 8px; margin: 0 2px; border-radius: 11px; font-size: 12px; line-height: 1.5; vertical-align: middle; user-select: none; cursor: default; max-width: 340px; background: var(--color-cat-blue-bg); color: var(--color-cat-blue); border: 1px solid var(--color-cat-blue-bg); }
.att-inline-image { background: var(--color-cat-green-bg); color: var(--color-cat-green); border-color: var(--color-cat-green-bg); }
.att-inline-dir { background: var(--color-cat-amber-bg); color: var(--color-cat-amber); border-color: var(--color-cat-amber-bg); }
.att-inline .att-inline-icon { display: inline-flex; flex-shrink: 0; }
.att-inline .att-inline-svg { width: 12px; height: 12px; }
.att-inline .att-inline-label { max-width: 220px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.att-inline .att-inline-x { display: inline-flex; align-items: center; justify-content: center; width: 14px; height: 14px; border-radius: 50%; font-size: 12px; line-height: 1; color: inherit; opacity: 0.7; cursor: pointer; flex-shrink: 0; }
.att-inline .att-inline-x:hover { opacity: 1; background: var(--color-surface-3); }
.input-bottom-bar { display: flex; align-items: center; justify-content: space-between; gap: 6px; padding: 0 10px 10px 10px; }
.ibb-btns { display: flex; align-items: center; gap: 4px; flex-wrap: wrap; position: relative; }
/* ★ 2026-09-05 移动端化：传统 <select> 已由 SheetPicker（bottom-sheet）替换，.cmp-sel 移除 */
/* ★ 2026-09-25 监督纠正（缺陷 3）：设计稿 ui-theme.notes.md:1262 规定的输入卡操作控件规格
   —— 32×32 圆角图标卡（radius=8、图标居中、选中态走 --accent），三卡尺寸/圆角/边框一致：
   放行（评审模式）/ 折叠（自动折叠）/ 自主（自主模式）。原 gap+padding+999px 胶囊 + 文字 +
   自绘勾选字形已全部移除（模式名改由 title 承载）。 */
.obtn { display: inline-flex; align-items: center; justify-content: center; flex: 0 0 auto; width: 32px; height: 32px; padding: 0; border-radius: 8px; cursor: pointer; color: var(--text-muted); background: var(--bg-tertiary); border: 1px solid var(--border-color); white-space: nowrap; user-select: none; transition: all 0.12s; }
.obtn:hover { color: var(--text-secondary); border-color: var(--text-muted); }
.obtn.active { color: var(--accent); background: var(--color-accent-bg); border-color: var(--color-accent-ring); }
.obtn-obtn-agent.active { color: var(--color-cat-amber); }
/* 三态审核按钮样式 */
.obtn-review-auto { color: var(--color-success); background: var(--color-success-bg); border-color: var(--color-success-bg); }
.obtn-review-manual { color: var(--color-warning); background: var(--color-warning-bg); border-color: var(--color-warning-bg); }
.obtn-review-off { color: var(--text-muted); background: var(--bg-tertiary); border-color: var(--border-color); opacity: 0.6; }



/* ★ 2026-09-05 移动端融合设计：发送/停止 = 圆形 FAB 按钮 */
.send-btn { background: linear-gradient(135deg, var(--accent) 0%, var(--accent-light) 100%); color: var(--color-accent-fg); width: 34px; height: 34px; padding: 0; border-radius: 50%; cursor: pointer; border: none; transition: opacity 0.15s, transform 0.1s, box-shadow 0.15s; display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; box-shadow: 0 3px 10px var(--color-accent-ring); }
.send-btn svg { margin-left: 2px; }
.send-btn:hover:not(:disabled) { opacity: 0.92; transform: translateY(-1px); box-shadow: 0 5px 14px var(--color-accent-ring); }
.send-btn:disabled { opacity: 0.35; cursor: not-allowed; box-shadow: none; }
.stop-btn { background: var(--color-danger); color: var(--color-accent-fg); width: 34px; height: 34px; padding: 0; border-radius: 50%; cursor: pointer; border: none; display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; box-shadow: 0 3px 10px var(--color-danger-bg); }
/* ── 用户消息附件标签 ── */
.user-attachments { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 6px; }
.att-tag { display: inline-flex; align-items: center; gap: 4px; padding: 1px 6px 1px 8px; border-radius: 11px; font-size: 12px; cursor: default; }
.att-tag-file, .att-tag-code, .att-tag-dir { background: var(--color-cat-blue-bg); color: var(--color-cat-blue); border: 1px solid var(--color-cat-blue-bg); }
.att-tag-image { background: var(--color-cat-green-bg); color: var(--color-cat-green); border: 1px solid var(--color-cat-green-bg); }
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
  color: var(--color-accent-fg);
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
  background: var(--accent); color: var(--color-accent-fg); border: none;
  padding: 4px 8px; border-radius: 4px; cursor: pointer;
  display: flex; align-items: center; flex-shrink: 0;
}
.feedback-send-btn:disabled { opacity: 0.4; cursor: default; }

/* ── 合并到 agent 气泡中的用户反馈标记 ── */
.fb-merged-section {
  background: var(--color-warning-bg); border: 1px dashed var(--color-warning);
  border-radius: 6px; padding: 6px 10px; margin-bottom: 8px;
}
.fb-merged-item { margin-bottom: 4px; }
.fb-merged-item:last-child { margin-bottom: 0; }

.fb-merge-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-cat-amber);
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
/* ── 自主模式监督看板容器（输入区上方，与任务进度容器同规格；无回合时高度 0 不占位）── */
.autopilot-container {
  flex-shrink: 0;
  transition: max-height 0.25s ease;
  padding: 0 8px;
  /* ★ 2026-09-25：容器自身做成 flex 列 + overflow:hidden，与 .ap-panel 的高度模型
     配套（面板 flex:0 1 auto + min-height:0、.ap-body flex:1 内滚）—— 面板高度恒
     ≤ 320px，不再溢出到下方被不透明的 .chat-input-area 盖住（用户报「展开被输入框
     盖住一部分」）。overflow:hidden 是兜底：即使未来面板内部再超界，也只在本容器
     内被裁，不会外溢遮挡输入区。 */
  display: flex; flex-direction: column;
  overflow: hidden;
}
.autopilot-container.autopilot-empty {
  max-height: 0;
  padding: 0 8px;
}
.autopilot-container:not(.autopilot-empty) {
  max-height: 320px;
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
  background: var(--color-cat-blue-bg); color: var(--color-cat-blue);
  border: 1px solid var(--color-cat-blue-bg);
  box-sizing: border-box;
}
.chat-input .att-inline-image {
  background: var(--color-cat-green-bg); color: var(--color-cat-green);
  border-color: var(--color-cat-green-bg);
}
.chat-input .att-inline-dir {
  background: var(--color-cat-amber-bg); color: var(--color-cat-amber);
  border-color: var(--color-cat-amber-bg);
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
.chat-input .att-inline .att-inline-x:hover { opacity: 1; background: var(--color-surface-3); }
</style>
