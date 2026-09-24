// ═══════════════════════════════════════════════════════════════
// ui-state.js — UI 全局状态/对话框/主题/持久化（原 main.js 状态部分）
//
// ★ 2026-08-16 按槽位细粒度拆分：本文件是全部区域插件包（titlebar/
// activitybar/sidebar/editor/right-panel/statusbar/modals）共享的核心
// 状态层——被各区域 bundle external（window.__PAIRCODE_CORE.uiState），
// 所有区域读写同一份 reactive 状态（替代原 App.vue 的 provide/inject）。
// 壳（ShellApp）与 app-actions 也引用本模块。
// ═══════════════════════════════════════════════════════════════
import { reactive, ref, computed, watch } from 'vue'
// ★ 视图打开状态真源在 plugin-runtime（localStorage viewOpen:<插件>:<视图id>）——
//   本模块只做「主区 tab 激活位」与「并排布局」的状态机，打开/关闭动作委托它。
//   （plugin-runtime 不反向 import 本模块，无循环依赖。）
import { setViewOpen } from './plugin-runtime.js'

// ─── 持久化键名 ──────────────────────────────────────────────
export const PERSIST_KEY = 'paircode-ide-state'

// ─── 全局状态 ────────────────────────────────────────────────
// ─── 全局对话框状态 ──────────────────────────────────────────
export const dialogState = reactive({
  show: false,
  type: '',       // 'confirm' | 'prompt' | 'alert'
  title: '',
  message: '',
  confirmText: '确定',
  cancelText: '取消',
  inputValue: '',
  inputPlaceholder: '',
  checkboxLabel: '',   // confirm 类型时可选 checkbox 文案
  checkboxValue: false,// confirm 类型时 checkbox 状态
  resolve: null,  // Promise resolve 函数
  toasts: [],     // { id, message, type }
})

window.$confirm = (message, title = '确认', confirmText = '确定', cancelText = '取消') => {
  return new Promise(resolve => {
    dialogState.type = 'confirm'
    dialogState.title = title
    dialogState.message = message
    dialogState.confirmText = confirmText
    dialogState.cancelText = cancelText
    dialogState.checkboxLabel = ''
    dialogState.checkboxValue = false
    dialogState.show = true
    dialogState.resolve = resolve
  })
}

// $confirmWithCheckbox 带 checkbox 的确认对话框，resolve({ confirmed: bool, checked: bool })
window.$confirmWithCheckbox = (message, title = '确认', checkboxLabel = '', confirmText = '确定', cancelText = '取消') => {
  return new Promise(resolve => {
    dialogState.type = 'confirm'
    dialogState.title = title
    dialogState.message = message
    dialogState.confirmText = confirmText
    dialogState.cancelText = cancelText
    dialogState.checkboxLabel = checkboxLabel
    dialogState.checkboxValue = false
    dialogState.show = true
    dialogState.resolve = resolve
  })
}

window.$prompt = (message, defaultValue = '', title = '输入', confirmText = '确定', cancelText = '取消') => {
  return new Promise(resolve => {
    dialogState.type = 'prompt'
    dialogState.title = title
    dialogState.message = message
    dialogState.inputValue = defaultValue
    dialogState.inputPlaceholder = ''
    dialogState.confirmText = confirmText
    dialogState.cancelText = cancelText
    dialogState.show = true
    dialogState.resolve = resolve
  })
}

window.$alert = (message, title = '提示') => {
  return new Promise(resolve => {
    dialogState.type = 'alert'
    dialogState.title = title
    dialogState.message = message
    dialogState.show = true
    dialogState.resolve = resolve
  })
}

window.$toast = (message, type = 'info', duration = 3000) => {
  const id = Date.now() + Math.random()
  dialogState.toasts.push({ id, message, type })
  setTimeout(() => {
    dialogState.toasts = dialogState.toasts.filter(t => t.id !== id)
  }, duration)
}

export const state = reactive({
  // ★ 2026-09-25 对齐设计稿 shell-midnight：左栏默认显示【会话列表】（th61「会话」），
  //   文件浏览器改为活动栏第二个图标进入（原默认 explorer）。
  activeActivity: 'chat',
  sidebarVisible: true,
  rightPanelVisible: true,
  // ★ 会话列表面板（.conv-sidebar，250px，含 Token 统计/上下文占用）整体显隐。
  //   默认显示；用户选择持久化；进入专注模式自动收起、退出还原（setFocusMode）。
  convListVisible: true,
  // ★ 右栏（2026-09-25 对齐设计稿 th178）：运行统计 / 任务进度 / 上下文构成 / 底提示，
  //   宽 288px，占网格第 4 列。默认显示。
  statsRailVisible: true,
  // ★ 右栏运行统计条折叠（P4-2，2026-09-25）——.phase-bar 里的步数/工具次数与
  //   耗时/墙钟耗时/输出速度/输出 token 六个数字整体收起，只留阶段图标+阶段文案+
  //   进度条（「正在做什么」始终可见，「做过多少」按需展开）。
  //   属纯展示偏好（非 focusMode 那类临时视图态）→ 持久化，与 convListVisible 同规则。
  runStatsCollapsed: false,
  bottomPanelVisible: true,
  bottomPanelTab: 'terminal',
  workspaceRoot: '',
  workspaceFolders: [],
  workspaceName: '',
  wsList: reactive([]),
  fileTree: [],
  expandedDirs: {},
  loadingDir: '',
  openFiles: [],
  activeFile: '',
  // ── 主内容区 tab（对话/编辑器/市场/工具集 多视图）──
  marketTabOpen: false,      // 「市场」tab 是否打开
  toolsetsTabOpen: false,    // 「工具集」tab 是否打开
  fileContents: {},
  fileSavedContent: {}, // 磁盘上原始内容，用于准确判断是否修改
  fileDirty: {},
  cursorLine: 1,
  cursorCol: 1,
  conversations: [],
  currentConvId: '',
  messages: [],
  chatLoading: false,
  chatSessionId: '',
  agentRunning: false,
  // ── 多会话并行：按 convId 存储各对话的独立状态 ──
  messagesByConv: {},        // { [convId]: [...] } 各对话消息数组
  loadingByConv: {},         // { [convId]: boolean } 各对话加载状态
  agentRunningByConv: {},    // { [convId]: boolean } 各对话 agent 运行状态
  approvalByConv: {},        // { [convId]: { callId, tool, args, waiting } } 各对话审批状态
  phaseByConv: {},           // { [convId]: string } 各对话当前阶段（自主模式）
  nudgeByConv: {},           // { [convId]: string } 各对话 nudge 提示文本
  convCtxStatsByConv: {},    // { [convId]: reactive({...}) } 各对话上下文 token 统计
  // ★ 各对话「本次运行」统计（耗时计时/步数/token 速度展示）——
  //   由 agent-events 的 usage/step 事件与 status 运行集合维护，RightPanel 渲染。
  //   运行中实时刷新（1s tick），结束后 endAt 定格保留（切会话显示各自的）。
  // { [convId]: { startAt, endAt, durationMs, running, steps, toolCalls, toolMs, llmCalls,
  //               llmMs, genMs, promptTokens, completionTokens, tokensPerSecond, fetchedAt } }
  // ★ 2026-09-12：字段与**后端**运行统计对齐（GET /api/conversations/{id}/run-stats）；
  //   前端只缓存展示，不累加、不本地持久化（真源在后端 .pair/run-stats.json）。
  runStatsByConv: {},
  msgTotalByConv: {},        // { [convId]: number } 各对话总消息数（懒加载判断是否还有更早消息）
  msgLoadedByConv: {},       // { [convId]: number } 各对话已加载消息数
  runningByWorkspace: {},    // { [wsRoot]: count } 各工作区运行中 agent 计数（供工作区列表显示脉冲点）
  wsTokenStatsByWs: {},      // { [wsRoot]: { promptTokens, ... } } 各工作区 token 统计（隔离）
  settings: {},
  settingsLoaded: false,
  pluginSchemas: [], // 插件注册的配置段（ctx.registerSettings → GET /api/settings.schemas）
  // ★ 2026-09-25 会话级「场景（=工具集）」生效信息镜像 —— 供会话侧栏底部胶囊
  //   （设计稿 th60「全栈开发 · 14 插件」= 场景名 + 该场景装配的插件数）读取。
  //   权威源 = GET /api/toolsets/active（agent.ResolveConvToolsetActive），由
  //   RightPanel.syncConvToolsetFromConv() 写入（唯一写方，避免双源不一致）。
  //   ★ 修复前史：侧栏读 settings.scenarioName/toolsetName/defaultToolset —— 这
  //   三个字段全项目从未被写入 → 胶囊恒显示硬编码兜底「默认场景」（假数据），
  //   且读全局 settings（非当前会话）→ 切会话也不变。
  //   { name, defaultName, isDefault, converged, pluginCount, resolved }
  convToolsetInfo: { name: '', defaultName: '', isDefault: true, converged: false, pluginCount: 0, resolved: false },
  searchResults: [],
  selectedFilePaths: [],  // 文件树多选路径列表
  lastClickedFilePath: '', // 文件树最近点击（Shift范围选择用）
  tasks: [],
  notificationCount: 0,
  theme: 'midnight',
  focusMode: false, // ★ 默认非专注：编辑器+对话区并排（右侧宽度可拖拽调整）；Ctrl+K 切换专注（隐藏编辑器）
  // ── ★ chat 优先薄壳布局：编辑器按需打开的装配状态（默认编辑器隐藏）──
  //   权威面只在 ctx.uiLayout / __PAIRCODE_CORE.layout（见下方 layout 服务），
  //   区域包通过服务读写，不直接改本字段（避免状态机分散 & 编辑器直接改私有开关）。
  //   editorOpen/editorWidth 为「临时视图」状态，不持久化到 localStorage（沿用
  //   focusMode 不持久化先例，避免「上次打开→下次启动就显示编辑器」的经典坑）。
  //   ★ 只放「编辑器可见性」这一真正新建的状态机字段；sidebarVisible/rightPanelVisible
  //   继续用顶层 state（已有、被大量组件直接读写），避免双源不一致。
  panels: {
    editorOpen: false,        // ★ 默认折叠：编辑器隐藏（不占主导视图）
    editorWidth: 360,         // 折叠后打开时的默认详情列宽（对齐 DETAILS_DEFAULT=360）
    editorLastWidth: 360,     // 上次打开宽（折叠还原用）
    // ★ 主区 tab 单一事实源（2026-09）：'conversation' | 'editor' | 'market'
    //   | 'toolsets' | 'view:<插件名>:<视图 id>'（插件注册的中间区域视图，
    //   见 plugin-runtime 的 registerView / clientViews）
    //   editorOpen 保留为兼容映射（view==='editor' ⇔ editorOpen=true）
    mainTab: 'conversation',
    // ★ 与对话并排（2026-09）：主内容区左右分栏 —— 一栏对话、一栏当前视图
    //   （编辑器/市场/工具集/插件视图）。splitView=false → 单栏（tab 互斥切换）。
    //   splitChatSide：对话在哪一栏（'left' | 'right'）。
    //   ★ 不持久化（与 focusMode/editorOpen 同规则）：临时视图态，避免「上次并排
    //     → 下次启动仍是并排」的意外；刷新回到用户默认单栏。
    splitView: false,
    splitChatSide: 'left',
  },
})

// ─── 全局 UI 面板状态（跨区域共享，替代 App.vue provide/inject）───
// 对话框开关（titlebar 菜单/activitybar 打开 → modals 包消费）
export const showSettings = ref(false)
export const showSystem = ref(false)
export const showSource = ref(false)
export const showAbout = ref(false)
// 软件更新弹窗（状态栏「新版本可用」徽标 / 菜单打开 → UpdateModal 消费）
export const showUpdate = ref(false)
export const showQuickSwitcher = ref(false)
export const helpDocTarget = ref('features')
export const showHelp = ref(false)
// showHelp 可被设为字符串（文档id）或 true（默认 features）
// 用 computed 包装以拦截 set（必须用 computed() 才能让 .value 赋值触发 setter）
export const showHelpWrapper = computed({
  get() { return showHelp.value },
  set(v) {
    if (typeof v === 'string') {
      helpDocTarget.value = v
      showHelp.value = true
    } else {
      showHelp.value = !!v
      if (showHelp.value) helpDocTarget.value = 'getting-started'
    }
  },
})

// 面板尺寸（editor 包底部面板 / right-panel 包右侧面板 / sidebar 包侧栏）
export const bottomPanelHeight = ref(180)
export const rightPanelWidth = ref(320)
// ★ 2026-09-25 对齐设计稿 shell-midnight：侧栏（th61）标准宽 264px（原 280）。
export const sidebarWidth = ref(264)

// 面板尺寸持久化（原 App.vue loadPanelSize/savePanelSize）
export function loadPanelSize() {
  try {
    const d = JSON.parse(localStorage.getItem('paircode-panel-size') || '{}')
    // ★ 不限位（2026-08-16 用户要求）：恢复保存值，只做防呆（负值/NaN → 0），
    //   上限 = 布局物理边界（编辑器保底 340，树折叠假设；树展开时溢出可拖拽修正）。
    //   旧残留 rpw 可能 520+（右侧 775px 挤压编辑器的历史坑）——此处不再硬限 400，
    //   布局 minmax(340px,1fr) 兜底。
    if (d.rpw) {
      const v = parseFloat(d.rpw)
      rightPanelWidth.value = Number.isFinite(v) ? Math.max(0, Math.min(v, window.innerWidth - 593)) : 320
    }
    if (d.bph) bottomPanelHeight.value = Math.max(120, Math.min(parseFloat(d.bph) || 180, 500))
  } catch {}
  try {
    const sw = localStorage.getItem('paircode-sidebar-width')
    if (sw) sidebarWidth.value = Math.min(Math.max(parseInt(sw, 10) || 280, 160), 480)
  } catch {}
}
export function savePanelSize() {
  try {
    localStorage.setItem('paircode-panel-size', JSON.stringify({
      rpw: rightPanelWidth.value, bph: bottomPanelHeight.value
    }))
  } catch {}
  try {
    localStorage.setItem('paircode-sidebar-width', String(sidebarWidth.value))
  } catch {}
}
loadPanelSize()

// ─── ★ 跨区域布局服务（ctx.uiLayout / __PAIRCODE_CORE.layout）───
// 对齐 `ctx.layout` 的 LayoutController 语义（spec §3.4 / §6）：
//   面板转换（侧栏折叠 / 编辑器按需打开关闭）是区域包的唯一权威面。
// 区域包通过它读写布局开关，不直接改 state.panels 私有字段（spec E4）。
// 编辑器「折叠=隐藏（保持挂载，不 unmount）」，复用现有 CSS 宽度切换语义 ——
// 打开/关闭只改可见性（width:0 ↔ editorWidth），绝不触发 CM6/终端 WS 重挂。
export const layout = {
  // ★ 左栏（文件浏览器/搜索/Git 等侧栏区）显隐：专注态内手动切换时同步「退出专注」
  //   还原目标，避免退出专注时被旧值覆盖用户本次选择（与 toggleConvList 同规则）。
  toggleSidebar() {
    state.sidebarVisible = !state.sidebarVisible
    if (state.focusMode) sidebarBeforeFocus = state.sidebarVisible
  },
  // ★ 会话列表面板（Token 统计栏）显隐开关：与 toggleSidebar 同语义，只切可见性
  //   （v-show 保持挂载、不 unmount，避免会话列表重挂丢状态）；壳与区域包经本服务读写。
  toggleConvList() {
    state.convListVisible = !state.convListVisible
    if (state.focusMode) convListBeforeFocus = state.convListVisible
  },
  openEditor(filePath) {
    if (typeof filePath === 'string' && filePath) {
      state.activeFile = filePath
      if (!state.openFiles.includes(filePath)) state.openFiles.push(filePath)
      // ★ 打开文件即切到编辑器主 tab（主区 tab 互斥）
      state.panels.mainTab = 'editor'
    }
    // ★ 打开编辑器即退出专注（focusMode 是「纯对话」态：隐藏侧栏+编辑器；
    //   点文件树打开编辑时必须退出，否则编辑器仍被 focusMode 折叠不可见）。
    if (state.focusMode) setFocusMode(false)
    // ★ 从关闭态打开：记录上次打开宽（折叠还原用），再置可见。
    if (!state.panels.editorOpen && state.panels.editorWidth > 0) {
      state.panels.editorLastWidth = state.panels.editorWidth
    }
    state.panels.editorOpen = true
  },
  closeEditor() {
    // ★ 折叠不 unmount：只置 false，宽度由壳 css width:0 收缩；CM6/终端 WS 保留。
    state.panels.editorOpen = false
    // ★ 关闭编辑器 tab → 回到对话主视图
    if (state.panels.mainTab === 'editor') state.panels.mainTab = 'conversation'
  },
  toggleEditor() {
    if (state.panels.editorOpen) layout.closeEditor()
    else layout.openEditor()
  },
  isEditorOpen() {
    return !!state.panels.editorOpen
  },
  setEditorWidth(px) {
    const v = Number(px)
    if (Number.isFinite(v) && v > 0) {
      state.panels.editorWidth = v
      state.panels.editorLastWidth = v
    }
  },
  // ★ 主视图 tab 切换（对话 ⇄ 编辑器 ⇄ 市场）：mainTab 是单一事实源。
  //   各视图常驻挂载（壳 v-show 切换），互不影响（CM6/终端 WS 不重挂）。
  setMainView(view) {
    state.panels.mainTab = view
    state.panels.editorOpen = (view === 'editor')
  },
  // ─── ★ 插件中间区域视图（registerView 注册的视图 tab，2026-09）─────────
  //   mainTab 取值 'view:<插件名>:<视图 id>'；打开状态由 plugin-runtime 持久化。
  viewTabKey(pluginName, id) { return 'view:' + pluginName + ':' + id },
  // 打开视图 tab（activate=true 时同时激活；默认打开是「后台 tab」语义——
  // 插件注册时的 open:true 只是让 tab 出现，激活仍由用户点击触发）。
  openViewTab(pluginName, id, opts) {
    const activate = !opts || opts.activate !== false
    setViewOpen(pluginName, id, true)
    if (activate) this.activateViewTab(pluginName, id)
  },
  // 激活视图 tab（切主视图；并排开启时同时保持对话可见）。
  activateViewTab(pluginName, id) {
    state.panels.mainTab = this.viewTabKey(pluginName, id)
    state.panels.editorOpen = false
  },
  // 关闭视图 tab（× ）：持久化关闭状态；若正激活则回对话主视图。
  closeViewTab(pluginName, id) {
    setViewOpen(pluginName, id, false)
    if (state.panels.mainTab === this.viewTabKey(pluginName, id)) {
      state.panels.mainTab = 'conversation'
    }
  },
  isViewTabActive(pluginName, id) {
    return state.panels.mainTab === this.viewTabKey(pluginName, id)
  },
  // ─── ★ 与对话并排（可切换）─────────────────────────────────────────
  // 语义：主内容区左右分栏 —— 一栏固定显示对话，另一栏显示当前激活视图
  // （编辑器/市场/工具集/插件视图）。对话本身就是主视图时「并排」无意义（会变成
  // 两栏对话）→ 调用方需先切到某个视图；本方法对 mainTab==='conversation' 返回 false。
  toggleSplit() {
    if (state.panels.splitView) {
      state.panels.splitView = false
      return true
    }
    if (state.panels.mainTab === 'conversation') return false
    state.panels.splitView = true
    return true
  },
  setSplitChatSide(side) {
    state.panels.splitChatSide = side === 'right' ? 'right' : 'left'
  },
  // 并排是否真正生效（开关开启 + 当前不是纯对话视图）。
  isSplitActive() {
    return state.panels.splitView === true && state.panels.mainTab !== 'conversation'
  },
}

// ─── ★ 专注模式（focusMode）唯一权威入口：隐藏编辑器 + 临时收起左右侧栏 ───
//   语义：专注 = 纯对话视图。进入时隐藏编辑器，并收起左栏（文件浏览器/搜索/Git）
//   与右栏会话列表面板；退出时还原用户进入前的显隐选择（尊重既有偏好，避免
//   「用户本就隐藏 → 退出专注被强制显示」）。
//   ★ 为什么不用组件内 watch focusMode：watch 默认 flush:'pre'，回调在
//     「同一同步块内后续语句」之后执行 —— 菜单「视图 → 资源管理器」是
//     `setFocusMode(false)` 紧跟 `state.sidebarVisible = true`，若靠 watch 还原
//     会把用户显式要求的「显示侧栏」覆盖掉。集中到本函数内显式处理，调用方的
//     语句顺序天然生效（先还原、后显式覆盖）。
//   ★ 两栏原值只存内存：与 focusMode 同为临时视图态，不持久化 —— 刷新后回到
//     用户真实偏好，不会把「专注时收起」误存成偏好。
let sidebarBeforeFocus = state.sidebarVisible
let convListBeforeFocus = state.convListVisible

export function setFocusMode(on) {
  const next = !!on
  if (next === !!state.focusMode) return
  if (next) {
    // 进入专注：记住既有选择，再临时收起两栏
    sidebarBeforeFocus = state.sidebarVisible
    convListBeforeFocus = state.convListVisible
    state.sidebarVisible = false
    state.convListVisible = false
  } else {
    // 退出专注：还原进入前的显隐选择
    state.sidebarVisible = sidebarBeforeFocus
    state.convListVisible = convListBeforeFocus
  }
  state.focusMode = next
}

// ★ 调试探针入口：暴露全局 store，供 wb-ui probe 直接读取状态层
if (typeof window !== 'undefined') window.__state = state

// ─── 字体加载映射（所有主题统一用 Inter + JetBrains Mono）───
const FONT_CONFIG = {
  dark: {
    ui: ['Inter:400,500,600,700'],
    code: ['JetBrains Mono:400,500,600'],
    google: ['Inter', 'JetBrains Mono'],
  },
  light: {
    ui: ['Inter:400,500,600,700'],
    code: ['JetBrains Mono:400,500,600'],
    google: ['Inter', 'JetBrains Mono'],
  },
  warm: {
    ui: ['Inter:400,500,600,700'],
    code: ['JetBrains Mono:400,500,600'],
    google: ['Inter', 'JetBrains Mono'],
  },
  night: {
    ui: ['Inter:400,500,600,700'],
    code: ['JetBrains Mono:400,500,600'],
    google: ['Inter', 'JetBrains Mono'],
  },
}

// 加载 Google Fonts
let fontLinkEl = null
function loadThemeFonts(theme) {
  const cfg = FONT_CONFIG[theme] || FONT_CONFIG.dark
  const families = [cfg.ui[0], cfg.code[0]].filter(Boolean).join('&family=')
  const href = 'https://fonts.loli.net/css2?family=' + families + '&display=swap'

  // 移除旧 link
  if (fontLinkEl) { document.head.removeChild(fontLinkEl); fontLinkEl = null }

  // 创建新 link（try 防止网络问题导致报错）
  try {
    const link = document.createElement('link')
    link.rel = 'stylesheet'
    link.href = href
    link.onload = () => { fontLinkEl = link }
    link.onerror = () => { /* 离线时静默回退系统字体 */ }
    document.head.appendChild(link)
  } catch {}
}

// ─── 应用主题 ────────────────────────────────────────────────
export function applyTheme(themeName) {
  const theme = themeName || state.theme || 'midnight'
  state.theme = theme

  // ★ 主题系统 v2：主题 class **按前缀通用清除**，新增主题无需再改此处。
  //   旧写法硬编码 4 个 remove 名 —— 扩到 8 套主题后会残留上一个主题的 class，
  //   由于后声明的主题块同特异性覆盖前块，会直接串色（切到 slate 后切回也仍是 slate）。
  const cls = 'theme-' + theme
  for (const el of [document.documentElement, document.body]) {
    for (const c of Array.from(el.classList)) {
      if (c.startsWith('theme-')) el.classList.remove(c)
    }
    el.classList.add(cls)
    // ★ P3：同时挂 data-theme 属性 —— class 面向样式表，data-* 面向插件/脚本
    //   读取「当前主题 id」（避免插件去解析 class 名）。
    //   取值是主题 id 本身（midnight/graphite/…/slate），不是 theme- 前缀形式。
    el.dataset.theme = theme
    // 同时标注明暗档，供背景/对比度守护等处直接用
    el.dataset.themeScheme = isDarkTheme(theme) ? 'dark' : 'light'
  }

  // 加载字体
  loadThemeFonts(theme)

  // 持久化
  savePersistentState()
}

// ─── 持久化：保存 UI 布局偏好到 localStorage（仅面板位置/主题等纯展示设置，不含工作区数据）───
export function savePersistentState() {
  try {
    const data = {
      version: 1,
      activeActivity: state.activeActivity,
      sidebarVisible: state.sidebarVisible,
      rightPanelVisible: state.rightPanelVisible,
      // 会话列表面板显隐：属面板偏好（非 focusMode 那类临时视图态）→ 持久化
      convListVisible: state.convListVisible,
      // 运行统计条折叠：同属面板偏好 → 持久化
      runStatsCollapsed: state.runStatsCollapsed,
      bottomPanelVisible: state.bottomPanelVisible,
      bottomPanelTab: state.bottomPanelTab,
      theme: state.theme,
      // focusMode 不持久化：专注模式是临时视图状态（Ctrl+K），跨会话记住
      // 会导致用户浏览器残留 true 时每次打开都隐藏编辑器（历史坑）。
    }
    localStorage.setItem(PERSIST_KEY, JSON.stringify(data))
  } catch (e) {
    console.warn('savePersistentState error:', e)
  }
}

// ─── 持久化：从 localStorage 恢复 UI 布局偏好 ────────────
// 只恢复面板位置/主题等纯展示设置，工作区和编辑器数据全部从 API 加载。
export function loadPersistentState() {
  try {
    const raw = localStorage.getItem(PERSIST_KEY)
    if (!raw) return
    const data = JSON.parse(raw)
    if (!data || !data.version) return

    // ★ 不恢复 activeActivity：活动栏切换成本极低，且旧 localStorage 残留
    //   'plugins'/'marketplace' 等会让侧边栏显示工具集/占位（用户预期
    //   每次打开是文件树）。默认每次 explorer。
    if (typeof data.sidebarVisible === 'boolean') state.sidebarVisible = data.sidebarVisible
    if (typeof data.rightPanelVisible === 'boolean') state.rightPanelVisible = data.rightPanelVisible
    // ★ 老 localStorage 无该字段 → 保持默认 true（向后兼容，升级无感）
    if (typeof data.convListVisible === 'boolean') state.convListVisible = data.convListVisible
    // ★ 老 localStorage 无该字段 → 保持默认 false（展开，向后兼容，升级无感）
    if (typeof data.runStatsCollapsed === 'boolean') state.runStatsCollapsed = data.runStatsCollapsed
    if (typeof data.bottomPanelVisible === 'boolean') state.bottomPanelVisible = data.bottomPanelVisible
    if (data.bottomPanelTab) state.bottomPanelTab = data.bottomPanelTab

    // 恢复主题
    if (data.theme) {
      // ★ 主题系统 v2：不再用硬编码白名单（旧写死 ['dark','light','warm','night']，
      //   扩到 8 套主题后新 id 会被静默拒绝 → 重载后主题丢失）。
      //   改为通用格式校验；未知 id 只会挂上不存在的 class，视觉回落默认主题，无害。
      if (/^[a-z][a-z0-9-]{0,23}$/.test(data.theme)) {
        applyTheme(data.theme)
      }
    }
    // ★ focusMode 不恢复（见 savePersistentState）：旧 localStorage 可能残留
    //   focusMode:true（默认专注时期保存），恢复会导致编辑器被永久隐藏。
    //   专注模式仅内存态（Ctrl+K 切换），每次加载默认 false。

    // ★ 不再从 localStorage 恢复 currentConvId，改为从后端 API 列表自动选中

    // ★ 以下字段不再从 localStorage 读取，全部从 API 获取：
    //   workspaceRoot / workspaceFolders / workspaceName ← 从 /api/health
    //   openFiles / activeFile / fileContents ← 从编辑器状态恢复
  } catch (e) {}
}

// ─── ★ 2026-09-25 切换对话瞬间把「场景」胶囊置为"解析中" ────────────────────
// 背景：会话切换只改 state.currentConvId，真实场景（会话级工具集）要靠
//   RightPanel 的异步请求（GET /api/toolsets/active）解析出来。实测切回大对话
//   （217 条消息）时主线程被消息渲染占满，该请求/响应可延迟数秒；若不置位，
//   这几秒里胶囊继续显示**上一个对话**的场景名 —— 是错误信息而非加载态。
// 位置选择：注册在本核心模块（先于所有区域包组件求值）→ watch 回调执行顺序
//   在 RightPanel 的 watch 之前，切换那一刻即生效，无需在各切换入口分别插桩
//   （切换入口有多处：Sidebar.onSwitchConv、app-actions 选首位/清空、
//   open-conversation 事件）。
watch(() => state.currentConvId, (id, old) => {
  if (id === old) return
  state.convToolsetInfo = {
    name: '', defaultName: '', isDefault: true,
    converged: false, pluginCount: 0, resolved: false,
  }
})

// ─── 主题明暗判定（语法高亮 / mermaid 图表 / 终端配色按此双档）─────────────
// ★ 8 套 UI 主题按明暗归两档。别名（dark=midnight / night=obsidian /
//   light=daylight / warm=sand）一并识别，使旧值域调用点（历史组件传
//   'dark'/'light'）也能得到正确判定，而不会再落到「全不匹配」的默认分支。
const DARK_THEME_NAMES = new Set([
  'midnight', 'graphite', 'obsidian', 'aurora', // 8 套中暗色 4 套
  'dark', 'night',                               // 别名
])
export function isDarkTheme(name) {
  return DARK_THEME_NAMES.has(name || state.theme || 'midnight')
}

// ─── 启动时应用主题（P1-a）─────────────────────────────────────────────
// 问题：此前 applyTheme 只被「打开设置面板」这条路径调用（SettingsModal.loadSettings），
//   于是 ① 首次启动（无 localStorage）时主题 class 从未挂载；
//        ② 用户重载页面时后端 AppSettings.theme 不参与，主题回到默认。
// 处理：启动即恢复本地持久化偏好（同步执行，避免首屏闪主题）；
//   后端 AppSettings 预取就绪后再覆盖 —— 见 syncThemeFromSettings。
export function initTheme() {
  loadPersistentState()
}

// 后端 AppSettings.theme 为权威源（用户在设置面板的显式选择、跨设备一致）；
// 由 app-actions 的 settings 预取完成后调用。未设置时保留本地偏好，不写默认值。
export function syncThemeFromSettings() {
  const t = state.settings && state.settings.theme
  if (t && t !== state.theme) applyTheme(t)
}
