import { createApp, reactive } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'

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
  tasks: [],
  notificationCount: 0,
  theme: 'dark',
  focusMode: false,
})

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
      focusMode: state.focusMode,
      currentConvId: state.currentConvId,
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

    if (data.activeActivity) state.activeActivity = data.activeActivity
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
    if (typeof data.focusMode === 'boolean') state.focusMode = data.focusMode

    // ★ 恢复当前对话 ID（页面刷新后自动选中上次对话）
    if (data.currentConvId) {
      state.currentConvId = data.currentConvId
    }

    // ★ 以下字段不再从 localStorage 读取，全部从 API 获取：
    //   workspaceRoot / workspaceFolders / workspaceName ← 从 /api/health
    //   openFiles / activeFile / fileContents ← 从编辑器状态恢复
  } catch (e) {}
}

// 初始化主题
applyTheme('dark')

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
