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
  searchResults: [],
  selectedFilePaths: [],  // 文件树多选路径列表
  lastClickedFilePath: '', // 文件树最近点击（Shift范围选择用）
  tasks: [],
  notificationCount: 0,
  theme: 'dark',
  focusMode: false, // ★ 默认非专注：编辑器+对话区并排（右侧宽度可拖拽调整）；Ctrl+K 切换专注（隐藏编辑器）
})

// ─── 全局 UI 面板状态（跨区域共享，替代 App.vue provide/inject）───
// 对话框开关（titlebar 菜单/activitybar 打开 → modals 包消费）
export const showSettings = ref(false)
export const showSystem = ref(false)
export const showSource = ref(false)
export const showMarketplace = ref(false)
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
  const href = 'https://fonts.geekzu.org/css2?family=' + families + '&display=swap'

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
