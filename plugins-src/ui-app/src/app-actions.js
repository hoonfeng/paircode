// ═══════════════════════════════════════════════════════════════
// app-actions.js — 应用级全局逻辑（2026-08-16 按槽位细粒度拆分）
//
// 背景：全 UI 按槽位拆成 7 个区域插件包（titlebar/activitybar/sidebar/
// editor/right-panel/statusbar/modals）后，原 App.vue 的「跨区域全局
// 逻辑」（工作区切换/文件树加载/快捷键/WS 事件桥/轮询）不能归任何一个
// 区域包——它们被多个区域共同消费。本模块作为共享核心之一：
//   壳（ShellApp）   → 调 init()/cleanup() 建立全局事件与 WS
//   MenuBar(titlebar) → loadWsList/saveWsList/switchWorkspace
//   ActivityBar       → switchActivity
//   Sidebar           → loadFileTree
//   各区域           → 直接读 ui-state 的共享状态（不经 provide/inject）
//
// ★ 构建：本模块被壳与全部区域 bundle external（挂 window.__PAIRCODE_CORE.actions），
//   保证所有区域操作同一份全局逻辑/事件注册（不重复注册、不重复建 WS）。
// ═══════════════════════════════════════════════════════════════
import { reactive, nextTick } from 'vue'
import api from './api.js'
import {
  state, loadPersistentState, savePersistentState,
  showSettings, showSystem,
} from './ui-state.js'
import {
  processAgentEvent, processAgentDone, processStatus, processAllDisconnected,
} from './agent-events.js'

// ─── 工作区列表 ──────────────────────────────────────────────
// （原 App.vue：wsList 从后端 /api/settings 拉取；desktop(goja) 预取逻辑
//   由壳 ShellApp 保留——本模块面向 web 端）

// ★ 路径归一化（2026-08-21）：历史异常数据里的双反斜杠（F:\\syproject\\gou-ide）
//   折叠为单反斜杠——清空配置重启后旧页面提交/读取时自愈，不再持续污染配置。
function normPath(p) {
  if (typeof p !== 'string' || !p) return p
  return p.replace(/\\\\+/g, '\\')
}

export async function loadWsList() {
  if (!state.wsList) state.wsList = reactive([])
  const wsList = state.wsList
  wsList.length = 0
  let loadedItems = []
  try {
    const resp = await api.apiGet('/settings')
    const settings = resp.settings || resp
    const projects = settings.recentProjects || []
    const folderLists = settings.workspaceFolderLists || {}
    const seen = new Set()
    for (const p0 of projects) {
      if (!p0 || seen.has(p0)) continue
      const p = normPath(p0) // 归一化历史双反斜杠路径（自愈）
      seen.add(p)
      const fl = folderLists[p0]?.length > 0 ? [...folderLists[p0]] : (folderLists[p]?.length > 0 ? [...folderLists[p]] : [])
      const folders = fl.length > 0 ? fl.map(normPath) : [p]
      loadedItems.push(reactive({
        path: p,
        name: p.split(/[\\/]/).filter(Boolean).pop() || p,
        folders: p === state.workspaceRoot && state.workspaceFolders?.length > 0
          ? [...state.workspaceFolders]
          : folders,
        notify: false,
      }))
    }
  } catch {}
  if (state.workspaceRoot && !loadedItems.find(w => w.path === state.workspaceRoot)) {
    loadedItems.push(reactive({
      path: state.workspaceRoot,
      name: state.workspaceRoot.split(/[\\/]/).filter(Boolean).pop() || state.workspaceRoot,
      folders: state.workspaceFolders?.length > 0 ? [...state.workspaceFolders] : [state.workspaceRoot],
      notify: false,
    }))
  }
  state.wsList = loadedItems
}

export async function saveWsList() {
  const wsList = state.wsList || []
  try {
    const resp = await api.apiGet('/settings')
    const settings = resp.settings || resp
    settings.recentProjects = wsList.slice(0, 20).map(w => normPath(w.path)).filter(Boolean)
    settings.workspaceFolderLists = settings.workspaceFolderLists || {}
    for (const ws of wsList) {
      if (ws.folders?.length > 0) {
        settings.workspaceFolderLists[normPath(ws.path)] = ws.folders.map(normPath)
      }
    }
    await api.apiPut('/settings', settings)
  } catch {}
}

export function checkNotifications() {
  const wsList = state.wsList || []
  for (const ws of wsList) {
    ws.notify = state.notificationCount > 0 && ws.path !== state.workspaceRoot
  }
}

export async function switchWorkspace(targetPath) {
  if (!targetPath || targetPath === state.workspaceRoot) return
  const wsList = state.wsList || []
  try {
    const targetWs = wsList.find(w => w.path === targetPath)
    const folders = targetWs?.folders || []
    await api.apiPost('/workspace', {
      action: 'switch', root: targetPath,
      folders: folders.filter(f => f !== targetPath),
    })
    state.workspaceRoot = targetPath
    state.workspaceFolders = folders.length > 0 ? [...folders] : [targetPath]
    state.settings.workspaceFolders = [...state.workspaceFolders]
    state.workspaceName = targetPath.split(/[\\/]/).filter(Boolean).pop() || targetPath
    document.title = 'PairCode IDE - ' + state.workspaceName
    state.openFiles = []
    state.activeFile = ''
    state.fileContents = {}
    await loadFileTree()
    // ★ 2026-08-23 工作区隔离：切换工作区必须清空当前对话选中状态与消息视图，
    //   否则切走后仍显示旧工作区的对话消息（与 FileExplorer.switchToWorkspace 对齐），
    //   同时按目标工作区加载对话列表（loadConversationsForWorkspace 含清空逻辑）。
    await loadConversationsForWorkspace(targetPath)
    window.dispatchEvent(new CustomEvent('workspace-switched'))
    const ws = wsList.find(w => w.path === targetPath)
    if (ws) ws.notify = false
    state.notificationCount = 0
    if (targetWs) { targetWs.folders = [...state.workspaceFolders] }
    if (!wsList.find(w => w.path === targetPath)) {
      wsList.push(reactive({ path: targetPath, name: state.workspaceName, folders: [...state.workspaceFolders], notify: false }))
    }
    await saveWsList()
    savePersistentState()
    api.reconnectWebSocket()
  } catch (err) {
    console.error('切换工作区失败:', err)
  }
}

export async function loadConversationsForWorkspace(path) {
  state.conversations = []
  state.currentConvId = ''
  state.messages = []
  if (typeof path !== 'string' || !path) return
  try {
    const list = await api.apiGet('/conversations', { workspace: path })
    state.conversations = list || []
    // ★ 2026-08-21 修复"刷新后不自动选对话"：加载列表后自动选中最近更新的对话
    //   （后端按 UpdatedAt 倒序 → 取第一个）；currentConvId 赋值触发 RightPanel 的
    //   watch → switchConv 加载消息。同时触发 save-conversations 持久化选中状态。
    if (state.conversations.length > 0) {
      state.currentConvId = state.conversations[0].id
    }
  } catch (e) {
    console.warn('从后端加载对话消息失败:', e)
  }
}

// switchActivity 活动栏切换（原 App.vue；settings/system/chat 分流）
export function switchActivity(id) {
  if (id === 'settings') { showSettings.value = true; return }
  if (id === 'system') { showSystem.value = true; return }
  if (id === 'chat') { state.rightPanelVisible = !state.rightPanelVisible; return }
  // ★ 市场已迁至主内容区 tab（2026-09）：活动栏/菜单「市场」→ 开/关市场 tab，
  //   不再切换侧边栏视图
  if (id === 'marketplace') { toggleMarketTab(); return }
  // ★ 工具集已迁至主内容区 tab：同市场，不再切换侧边栏视图
  if (id === 'toolsets') { toggleToolsetsTab(); return }
  if (state.activeActivity === id) {
    state.sidebarVisible = !state.sidebarVisible
  } else {
    state.activeActivity = id
    state.sidebarVisible = true
  }
}

// ── 市场 tab（主内容区 main-tabs 第三视图）：打开/激活/收起 ──
// 点击市场入口：未开→打开并激活；已开且激活→关闭；已开但未激活→激活。
export function toggleMarketTab() {
  if (state.marketTabOpen && state.panels.mainTab === 'market') {
    closeMarketTab()
  } else {
    state.marketTabOpen = true
    state.panels.mainTab = 'market'
    state.panels.editorOpen = false
  }
}
export function closeMarketTab() {
  state.marketTabOpen = false
  if (state.panels.mainTab === 'market') {
    state.panels.mainTab = 'conversation'
  }
}

// ── 工具集 tab（主内容区 main-tabs 视图）：打开/激活/收起（同市场范式）──
export function toggleToolsetsTab() {
  if (state.toolsetsTabOpen && state.panels.mainTab === 'toolsets') {
    closeToolsetsTab()
  } else {
    state.toolsetsTabOpen = true
    state.panels.mainTab = 'toolsets'
    state.panels.editorOpen = false
  }
}
export function closeToolsetsTab() {
  state.toolsetsTabOpen = false
  if (state.panels.mainTab === 'toolsets') {
    state.panels.mainTab = 'conversation'
  }
}

// loadFileTree 文件树加载（原 App.vue；Sidebar/FileExplorer 使用）
export const loadFileTree = async () => {
  const dirs = state.workspaceFolders.length > 0 ? [...state.workspaceFolders] : []
  if (dirs.length === 0 && state.workspaceRoot) dirs.push(state.workspaceRoot)
  const seen = new Set()
  const unique = dirs.filter(d => { if (seen.has(d) || !d) return false; seen.add(d); return true })
  state.fileTree = []
  for (const d of unique) {
    if (!d) continue
    try {
      const entries = await api.apiGet('/fs/list', { path: d })
      state.fileTree.push({ path: d, name: d.split('\\').filter(Boolean).pop() || d, children: entries || [], loaded: false })
    } catch {}
  }
}

// ─── 快捷键（原 App.vue handleKeydown；壳注册 document keydown）───
export function handleKeydown(e) {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return
  if (e.ctrlKey && e.key === 'b') { e.preventDefault(); state.sidebarVisible = !state.sidebarVisible }
  if (e.ctrlKey && e.key === '`') { e.preventDefault(); state.bottomPanelVisible = !state.bottomPanelVisible }
  if (e.ctrlKey && e.shiftKey && e.key === 'E') { e.preventDefault(); state.activeActivity = 'explorer'; state.sidebarVisible = true }
  if (e.ctrlKey && e.shiftKey && e.key === 'F') { e.preventDefault(); state.activeActivity = 'search'; state.sidebarVisible = true }
  if (e.ctrlKey && e.shiftKey && e.key === 'T') { e.preventDefault(); state.rightPanelVisible = true }
  if (e.ctrlKey && e.shiftKey && e.key === 'C') { e.preventDefault(); state.rightPanelVisible = !state.rightPanelVisible }
  if (e.ctrlKey && e.key === 'k') { e.preventDefault(); state.focusMode = !state.focusMode }
}

// ─── 文件树自动刷新防抖（原 App.vue）───
let _lastRefreshTime = 0
const MIN_TREE_REFRESH_INTERVAL = 3000
let _savedTreeScrollTop = 0

function refreshTree() {
  const now = Date.now()
  if (now - _lastRefreshTime < MIN_TREE_REFRESH_INTERVAL) return
  _lastRefreshTime = now

  const scrollEl = document.querySelector('.project-section')
  if (scrollEl) _savedTreeScrollTop = scrollEl.scrollTop

  for (const path of Object.keys(state.fileContents)) {
    if (state.openFiles.includes(path)) {
      if (!state.fileDirty[path]) {
        api.apiGet('/fs/read', { path }).then(data => {
          const normalized = (data.content || '').replace(/\r\n/g, '\n')
          state.fileContents[path] = normalized
          state.fileSavedContent[path] = normalized
          state.fileDirty[path] = false
        }).catch(() => {})
      }
    } else {
      delete state.fileContents[path]
      delete state.fileSavedContent[path]
      delete state.fileDirty[path]
    }
  }
  loadFileTree().then(() => {
    if (_savedTreeScrollTop > 0) {
      nextTick(() => {
        const c = document.querySelector('.project-section')
        if (c) c.scrollTop = _savedTreeScrollTop
      })
    }
  })
}

// ─── 全局初始化/清理（壳 ShellApp onMounted/onUnmounted 调用）───
let cleanupFns = []
let refreshTimer = null

export function initAppGlobals() {
  // 恢复 UI 布局偏好（面板/主题）
  loadPersistentState()

  // WS 事件桥（agent 事件 → agent-events 处理 → 各区域组件消费共享 state）
  api.initWebSocket({
    onStatus: (payload) => processStatus(payload),
    onEvent: (convId, data) => processAgentEvent(convId, data),
    onDone: (convId, data) => processAgentDone(convId, data),
    onDisconnected: () => processAllDisconnected(),
  })

  // 初始工作区（/api/health）：web 端 workspaceRoot 初始化（原 App.vue onMounted）
  ;(async () => {
    // ★ 设置与插件配置 schema 预取（修复：state.settings 此前从不从后端加载，
    //   设置面板打开即显示默认值；现全局拉取并缓存 pluginSchemas 供动态渲染）
    try {
      const sresp = await api.apiGet('/settings')
      if (sresp && sresp.settings) {
        state.settings = sresp.settings
        state.settingsLoaded = true
        state.pluginSchemas = sresp.schemas || []
      }
    } catch {}
    try {
      const health = await api.apiGet('/health')
      if (health && health.workspace) {
        state.workspaceRoot = health.workspace
        state.workspaceFolders = health.folders || []
        state.workspaceName = health.workspace.split('\\').filter(Boolean).pop() || health.workspace
        document.title = 'PairCode IDE - ' + state.workspaceName
        await loadConversationsForWorkspace(health.workspace)
      }
    } catch {}
    await loadFileTree()
  })()

  // 快捷键
  document.addEventListener('keydown', handleKeydown)

  // window 事件（原 App.vue 注册的 8 个）
  const handlers = {
    'refresh-tree': refreshTree,
    'switch-activity': (e) => { if (e.detail?.id) switchActivity(e.detail.id) },
    'open-marketplace': () => { switchActivity('marketplace') },
    'open-settings': () => { showSettings.value = true },
    'stop-agent': () => { window.dispatchEvent(new CustomEvent('agent-stop')) },
    'save-conversations': async () => { checkNotifications() },
    'open-workspace-dialog': () => { state.activeActivity = 'explorer'; state.sidebarVisible = true },
    'switch-workspace': async (e) => { if (e.detail?.path) await switchWorkspace(e.detail.path) },
    // ★ 2026-08-31 打开指定会话（agent-teams 团队成员子会话入口）：赋值
    //   currentConvId 触发 RightPanel watch → switchConv 加载消息。用于成员
    //   会话（conv_sub_*，不在顶层会话列表）从团队活动面板打开查看。
    'open-conversation': (e) => { if (e.detail?.id) { state.currentConvId = e.detail.id } },
  }
  for (const [ev, fn] of Object.entries(handlers)) window.addEventListener(ev, fn)

  // 定时轮询文件树（仅页面可见时）
  refreshTimer = setInterval(() => {
    if (document.visibilityState === 'visible') {
      _lastRefreshTime = 0
      window.dispatchEvent(new CustomEvent('refresh-tree'))
    }
  }, 5000)

  cleanupFns = [
    () => document.removeEventListener('keydown', handleKeydown),
    () => { for (const [ev, fn] of Object.entries(handlers)) window.removeEventListener(ev, fn) },
    () => { if (refreshTimer) { clearInterval(refreshTimer); refreshTimer = null } },
  ]
}

export function cleanupAppGlobals() {
  for (const fn of cleanupFns) { try { fn() } catch {} }
  cleanupFns = []
  api.closeWebSocket()
}

// ─── 桌面(goja) workaround：mount 前同步预取 wsList/health/conversations ───
// （原 App.vue setup 顶层逻辑；壳 ShellApp onMounted 最先调用）
export function desktopPrefetch() {
  try {
    if (typeof go !== 'undefined' && go.bridge_call && typeof window !== 'undefined' && window.__DESKTOP_MODE__) {
      const _r = go.bridge_call('GET', '/api/settings', '', '')
      const _parsed = JSON.parse(_r)
      const _settings = _parsed.body ? JSON.parse(_parsed.body).settings : {}
      const _projects = (_settings && _settings.recentProjects) || []
      const _folders = (_settings && _settings.workspaceFolderLists) || {}
      const _seen = new Set()
      const _items = []
      for (const _p of _projects) {
        if (!_p || _seen.has(_p)) continue
        _seen.add(_p)
        const _fl = (_folders[_p] && _folders[_p].length > 0) ? [..._folders[_p]] : [_p]
        _items.push(reactive({ path: _p, name: _p.split(/[\\/]/).filter(Boolean).pop() || _p, folders: _fl, notify: false }))
      }
      if (_items.length > 0) state.wsList = _items

      try {
        const _h = JSON.parse(go.bridge_call('GET', '/api/health', '', ''))
        const _health = JSON.parse(_h.body || '{}')
        if (_health.workspace) {
          state.workspaceRoot = _health.workspace
          state.workspaceFolders = _health.folders || []
          state.workspaceName = _health.workspace.split('\\').filter(Boolean).pop() || _health.workspace
          const _c = JSON.parse(go.bridge_call('GET', '/api/conversations?workspace=' + encodeURIComponent(_health.workspace), '', ''))
          const _list = JSON.parse(_c.body || '[]')
          if (Array.isArray(_list) && _list.length > 0) state.conversations = _list
        }
      } catch {}
    }
  } catch {}
}

export default {
  loadWsList, saveWsList, checkNotifications, switchWorkspace,
  loadConversationsForWorkspace, switchActivity, loadFileTree,
  handleKeydown, initAppGlobals, cleanupAppGlobals, desktopPrefetch,
}
