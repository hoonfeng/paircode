// ═══════════════════════════════════════════════════════════════
// ui-state.js — UI 全局状态/对话框/主题/持久化（原 main.js 状态部分）
//
// ★ 2026-08-16 按槽位细粒度拆分：本文件是全部区域插件包（titlebar/
// activitybar/sidebar/editor/right-panel/statusbar/modals）共享的核心
// 状态层——被各区域 bundle external（window.__PAIRCODE_CORE.uiState），
// 所有区域读写同一份 reactive 状态（替代原 App.vue 的 provide/inject）。
// 壳（ShellApp）与 app-actions 也引用本模块。
// ═══════════════════════════════════════════════════════════════
import { reactive, ref, computed } from 'vue'

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
  activeActivity: 'explorer',
  sidebarVisible: true,
  rightPanelVisible: true,
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
  // ── 市场 tab（主内容区 main-tabs 第三视图 tab）──
  marketTabOpen: false,      // 「市场」tab 是否打开
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
  msgTotalByConv: {},        // { [convId]: number } 各对话总消息数（懒加载判断是否还有更早消息）
  msgLoadedByConv: {},       // { [convId]: number } 各对话已加载消息数
  runningByWorkspace: {},    // { [wsRoot]: count } 各工作区运行中 agent 计数（供工作区列表显示脉冲点）
  wsTokenStatsByWs: {},      // { [wsRoot]: { promptTokens, ... } } 各工作区 token 统计（隔离）
  settings: {},
  settingsLoaded: false,
  pluginSchemas: [], // 插件注册的配置段（ctx.registerSettings → GET /api/settings.schemas）
  searchResults: [],
  selectedFilePaths: [],  // 文件树多选路径列表
  lastClickedFilePath: '', // 文件树最近点击（Shift范围选择用）
  tasks: [],
  notificationCount: 0,
  theme: 'dark',
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
    //   editorOpen 保留为兼容映射（view==='editor' ⇔ editorOpen=true）
    mainTab: 'conversation',
  },
})

// ─── 全局 UI 面板状态（跨区域共享，替代 App.vue provide/inject）───
// 对话框开关（titlebar 菜单/activitybar 打开 → modals 包消费）
export const showSettings = ref(false)
export const showSystem = ref(false)
export const showSource = ref(false)
export const showAbout = ref(false)
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
export const sidebarWidth = ref(280)

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
  toggleSidebar() {
    state.sidebarVisible = !state.sidebarVisible
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
    if (state.focusMode) state.focusMode = false
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
  const theme = themeName || state.theme || 'dark'
  state.theme = theme

  // 移除所有主题 class
  document.documentElement.classList.remove('theme-dark', 'theme-light', 'theme-warm', 'theme-night')
  document.body.classList.remove('theme-dark', 'theme-light', 'theme-warm', 'theme-night')

  // 添加对应 class
  const cls = 'theme-' + theme
  document.documentElement.classList.add(cls)
  document.body.classList.add(cls)

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
    if (typeof data.bottomPanelVisible === 'boolean') state.bottomPanelVisible = data.bottomPanelVisible
    if (data.bottomPanelTab) state.bottomPanelTab = data.bottomPanelTab

    // 恢复主题
    if (data.theme) {
      // 只在主题有效时恢复
      if (['dark', 'light', 'warm', 'night'].includes(data.theme)) {
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
