import { createApp, reactive } from 'vue'
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
  fileDirty: {},
  cursorLine: 1,
  cursorCol: 1,
  conversations: [],
  currentConvId: '',
  messages: [],
  chatLoading: false,
  chatSessionId: '',
  agentRunning: false,
  settings: {},
  settingsLoaded: false,
  searchResults: [],
  tasks: [],
  notificationCount: 0,
  theme: 'dark',
  focusMode: false,
})

// ─── 字体加载映射 ────────────────────────────────────────────
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
    ui: ['Noto+Serif+SC:400,600,700'],
    code: ['Source+Code+Pro:400,500,600'],
    google: ['Noto Serif SC', 'Source Code Pro'],
  },
  cute: {
    ui: ['Nunito:400,600,700,800'],
    code: ['Fira+Code:400,500,600'],
    google: ['Nunito', 'Fira Code'],
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
  document.documentElement.classList.remove('theme-dark', 'theme-light', 'theme-warm', 'theme-cute')
  document.body.classList.remove('theme-dark', 'theme-light', 'theme-warm', 'theme-cute')

  // 添加对应 class
  const cls = 'theme-' + theme
  document.documentElement.classList.add(cls)
  document.body.classList.add(cls)

  // 加载字体
  loadThemeFonts(theme)

  // 持久化
  savePersistentState()
}

// ─── 持久化：保存全部状态到 localStorage ────────────────────
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
      workspaceRoot: state.workspaceRoot,
      workspaceFolders: [...state.workspaceFolders],
      workspaceName: state.workspaceName,
      openFiles: [...state.openFiles],
      activeFile: state.activeFile,
    }
    localStorage.setItem(PERSIST_KEY, JSON.stringify(data))
  } catch (e) {}
}

// ─── 持久化：从 localStorage 恢复状态 ──────────────────────
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
      if (['dark', 'light', 'warm', 'cute'].includes(data.theme)) {
        applyTheme(data.theme)
      }
    }
    if (typeof data.focusMode === 'boolean') state.focusMode = data.focusMode

    if (data.workspaceName) state.workspaceName = data.workspaceName
    if (data.openFiles && Array.isArray(data.openFiles)) {
      state.openFiles = data.openFiles.filter(f => f)
    }
    if (data.activeFile) state.activeFile = data.activeFile
  } catch (e) {}
}

// 初始化主题
applyTheme('dark')

const app = createApp(App)
app.mount('#app')
