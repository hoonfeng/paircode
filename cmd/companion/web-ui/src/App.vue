<template>
  <div class="app-root" :class="{ 'panel-only': panelMode }">
    <!-- 标题栏 + 菜单栏（panel 模式下隐藏） -->
    <div v-if="!panelMode" class="titlebar" @click="closeAllMenus">
      <div class="app-logo">
        <img :src="logoUrl" class="logo-img" alt="PairCode" />
      </div>
      <MenuBar ref="menuBarRef" />
      <div class="title-center">{{ state.workspaceName }}</div>
      <div class="title-right">
        <button v-if="wsList.length > 1" class="ws-quick-btn"
                @click="showQuickSwitcher = !showQuickSwitcher" title="快速切换工作区">
          <SvgIcon name="folder" :size="14" />
        </button>
      </div>
    </div>

    <!-- 内容区域（panel 模式下隐藏） -->
    <ActivityBar v-if="!panelMode" />
    <Sidebar v-if="!panelMode && state.sidebarVisible && !state.focusMode" />
    <div v-if="!panelMode && !state.focusMode" class="main-area">
      <EditorArea />
      <div class="bottom-panel" v-if="state.bottomPanelVisible"
           :style="{ height: bottomPanelHeight + 'px' }">
        <div class="panel-content">
          <TerminalPanel @close-panel="state.bottomPanelVisible = false" />
        </div>
        <div class="panel-resizer" @mousedown.prevent="startBottomResize"></div>
      </div>
    </div>

    <!-- 右侧容器（panel 模式占满全屏） -->
    <div v-if="state.rightPanelVisible || panelMode" class="right-container"
         :class="{ 'focus-mode': state.focusMode, 'panel-only': panelMode }"
         :style="(state.focusMode || panelMode) ? {} : { width: (rightPanelWidth + 4 + 1 + 250) + 'px' }">
        <div v-if="!panelMode && !state.focusMode" class="right-panel-resizer" @mousedown.prevent="startRightResize"></div>
      <RightPanel :panel-mode="panelMode" />
    </div>

    <!-- 状态栏（panel 模式下隐藏） -->
    <StatusBar v-if="!panelMode" />

    <!-- 模态框 -->
    <SettingsModal v-if="showSettings" @close="showSettings = false" />
    <SystemModal v-if="showSystem" @close="showSystem = false" />
    <SourceModal v-if="showSource" @close="showSource = false" />
    <MarketplaceModal v-if="showMarketplace" @close="showMarketplace = false" />
    <HelpModal v-if="showHelp" @close="showHelp = false" @openAbout="onHelpOpenAbout" :initialDoc="helpDocTarget" />
    <AboutModal v-if="showAbout" @close="showAbout = false" @openHelp="onAboutOpenHelp" />
    <GlobalDialogs />
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onUnmounted, provide, nextTick } from 'vue'
import { state, savePersistentState, loadPersistentState, applyTheme } from './main.js'
import api from './api.js'
import { processAgentEvent, processAgentDone, processStatus, getConvCtxStats } from './agent-events.js'

// ★ 桌面端面板独立模式：desktopbridge 注入 window.__DESKTOP_PANEL_MODE__，
//   此时只渲染右侧面板（消息展示）占满全屏，隐藏 IDE 其他区域。
const panelMode = typeof window !== 'undefined' && window.__DESKTOP_PANEL_MODE__ === true

import MenuBar from './components/MenuBar.vue'
import ActivityBar from './components/ActivityBar.vue'
import Sidebar from './components/Sidebar.vue'
import EditorArea from './components/EditorArea.vue'
import RightPanel from './components/RightPanel.vue'
import StatusBar from './components/StatusBar.vue'
import TerminalPanel from './components/TerminalPanel.vue'
import SettingsModal from './components/SettingsModal.vue'
import SystemModal from './components/SystemModal.vue'
import SourceModal from './components/SourceModal.vue'
import MarketplaceModal from './components/MarketplaceModal.vue'
import HelpModal from './components/HelpModal.vue'
import AboutModal from './components/AboutModal.vue'
import SvgIcon from './components/SvgIcon.vue'
import GlobalDialogs from './components/GlobalDialogs.vue'
import logoUrl from './assets/logo.svg'

const showSettings = ref(false)
const showSystem = ref(false)
const showSource = ref(false)
const showMarketplace = ref(false)
const showQuickSwitcher = ref(false)
const showHelp = ref(false)
const showAbout = ref(false)
const helpDocTarget = ref('features')

// showHelp 可被设为字符串（文档id）或 true（默认 features）
// 用 computed 包装以拦截 set（必须用 computed() 才能让 .value 赋值触发 setter）
const showHelpWrapper = computed({
  get() { return showHelp.value },
  set(v) {
    if (typeof v === 'string') {
      helpDocTarget.value = v
      showHelp.value = true
    } else {
      showHelp.value = !!v
      if (showHelp.value) helpDocTarget.value = 'getting-started'
    }
  }
})

function onAboutOpenHelp() {
  showAbout.value = false
  showHelp.value = true
  helpDocTarget.value = 'getting-started'
}

function onHelpOpenAbout() {
  showHelp.value = false
  showAbout.value = true
}

function loadPanelSize() {
  try {
    const d = JSON.parse(localStorage.getItem('paircode-panel-size') || '{}')
    // ★ 右侧面板宽度上限 520：desktop 1280 窗口下 600 宽（含 255 附加
    //    = 855px）会把 main-area（grid 1fr）挤压到 ~97px → 终端/xterm
    //    挤成 98px 窄条（终端只显示 w 的根因）。cap 后右侧最多
    //    520+255=775px，保证主区 ≥ ~350px。
    if (d.rpw) rightPanelWidth.value = Math.min(parseFloat(d.rpw) || 380, 520)
    if (d.bph) bottomPanelHeight.value = d.bph
  } catch {}
  // 恢复侧栏宽度
  try {
    const sw = localStorage.getItem('paircode-sidebar-width')
    if (sw) sidebarWidth.value = parseInt(sw, 10)
  } catch {}
}
function savePanelSize() {
  try {
    localStorage.setItem('paircode-panel-size', JSON.stringify({
      rpw: rightPanelWidth.value, bph: bottomPanelHeight.value
    }))
  } catch {}
}
loadPanelSize()

const bottomPanelHeight = ref(180)
// ★ 默认 380（原 600）：1280 窗口下 600 右面板挤压 main-area 至 97px
const rightPanelWidth = ref(380)
const sidebarWidth = ref(280)

provide('showSettings', showSettings)
provide('showSystem', showSystem)
provide('showSource', showSource)
provide('showMarketplace', showMarketplace)
provide('showHelp', showHelpWrapper)
provide('showAbout', showAbout)
provide('bottomPanelHeight', bottomPanelHeight)
provide('rightPanelWidth', rightPanelWidth)
provide('sidebarWidth', sidebarWidth)

if (!state.wsList) state.wsList = reactive([])
const wsList = state.wsList

// ★ desktop(goja) workaround：Vue 渲染 effect 对整体赋值数组（state.wsList = ...）
//   的 set 不触发（watch 触发但 v-for/v-if 不更新）。解决：在组件 mount 前
//   同步预取 wsList（desktop 环境 go.bridge_call 同步返回），mount 时渲染函数
//   首次执行就读到 5 项 → v-for 正常渲染。web 环境无 go.bridge_call，跳过，
//   走 onMounted 的 loadWsList（浏览器响应式正常）。
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

    // ★ desktop(goja) workaround（同上）：conversations 数组整体赋值
    //   （state.conversations = list）同样触发不了 v-for 渲染。在 mount
    //   前同步预取 health + conversations（go.bridge_call 同步返回），
    //   mount 时渲染函数首次执行就读到数据 → ConvSidebar 正常渲染。
    //   web 环境无 go.bridge_call，跳过（走 onMounted 异步加载）。
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

provide('wsList', wsList)
provide('saveWsList', saveWsList)
provide('switchWorkspace', switchWorkspace)

async function loadWsList() {
  // 从后端 /api/settings 拉取工作区列表（recentProjects），保证与后端一致
  // 后端返回结构：{settings: core.Settings, loaded} —— recentProjects 在 settings.settings 下。
  // ★ wb-ui(goja) 对 reactive 数组的整体赋值触发不了下游组件渲染（渲染 effect 依赖不收集），
  //   已由 setup 顶层同步预取解决（mount 前填充）。这里仍保持数据同步（web 端正常响应式）。
  wsList.length = 0
  let loadedItems = []
  try {
    const resp = await api.apiGet('/settings')
    const settings = resp.settings || resp
    const projects = settings.recentProjects || []
    const folderLists = settings.workspaceFolderLists || {}
    const seen = new Set()
    for (const p of projects) {
      if (!p || seen.has(p)) continue
      seen.add(p)
      // 从 workspaceFolderLists 恢复文件夹列表，没有则默认为 [p]
      const folders = folderLists[p]?.length > 0 ? [...folderLists[p]] : [p]
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
  // 整体赋值：替换 state.wsList 引用，保持数据与后端一致（desktop 下 DOM 已由 mount 预取渲染）。
  state.wsList = loadedItems
}

async function saveWsList() {
  // 同步工作区列表到后端 settings
  try {
    const resp = await api.apiGet('/settings')
    const settings = resp.settings || resp
    settings.recentProjects = wsList.slice(0, 20).map(w => w.path).filter(Boolean)
    // 持久化每工作区的文件夹列表
    settings.workspaceFolderLists = settings.workspaceFolderLists || {}
    for (const ws of wsList) {
      if (ws.folders?.length > 0) {
        settings.workspaceFolderLists[ws.path] = [...ws.folders]
      }
    }
    // 保持后端契约：PUT body = {settings: core.AppSettings}
    await api.apiPut('/settings', { settings })
  } catch {}
}

function checkNotifications() {
  for (const ws of wsList) {
    ws.notify = state.notificationCount > 0 && ws.path !== state.workspaceRoot
  }
}

async function switchWorkspace(targetPath) {
  if (!targetPath || targetPath === state.workspaceRoot) return
  try {
    const targetWs = wsList.find(w => w.path === targetPath)
    const folders = targetWs?.folders || []
    await api.apiPost('/workspace', {
      action: 'switch', root: targetPath,
      folders: folders.filter(f => f !== targetPath),
    })
    state.workspaceRoot = targetPath
    state.workspaceFolders = folders.length > 0 ? [...folders] : [targetPath]
    // 同步 settings 中的 workspaceFolders，防止设置对话框保存时覆盖
    state.settings.workspaceFolders = [...state.workspaceFolders]
    state.workspaceName = targetPath.split(/[\\/]/).filter(Boolean).pop() || targetPath
    document.title = 'PairCode IDE - ' + state.workspaceName
    state.openFiles = []
    state.activeFile = ''
    state.fileContents = {}
    await loadFileTree()
    try {
      const list = await api.apiGet('/conversations', { workspace: targetPath })
      state.conversations = list || []
    } catch {}
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
    // ★ 工作区切换后强制重建 WebSocket 连接（后端可能因工作区切换断开 WS）
    api.reconnectWebSocket()
  } catch (err) {
    console.error('切换工作区失败:', err)
  }
}

function saveCurrentConversations() {
  // 对话消息由后端持久化到磁盘，前端不再缓存到 localStorage
}

async function loadConversationsForWorkspace(path) {
  // 当前对话的消息全部从后端 API 拉取（后端已持久化到磁盘）
  state.conversations = []
  state.currentConvId = ''
  state.messages = []
  if (typeof path !== 'string' || !path) return
  try {
    const list = await api.apiGet('/conversations', { workspace: path })
    state.conversations = list || []
  } catch (e) {
    console.warn('从后端加载对话消息失败:', e)
  }
}

const switchActivity = (id) => {
  if (id === 'settings') { showSettings.value = true; return }
  if (id === 'system') { showSystem.value = true; return }
  if (id === 'chat') { state.rightPanelVisible = !state.rightPanelVisible; return }
  if (id === 'marketplace') { showMarketplace.value = true; return }
    // ★ 编辑/终端让位：切换到 IDE 侧栏（文件/搜索/Git/插件）时自动退出专注模式
    state.focusMode = false
    if (state.activeActivity === id) {
    state.sidebarVisible = !state.sidebarVisible
  } else {
    state.activeActivity = id
    state.sidebarVisible = true
  }
}
provide('switchActivity', switchActivity)

const menuBarRef = ref(null)
const closeAllMenus = () => { if (menuBarRef.value) menuBarRef.value.closeMenu?.() }

let dragging = false
let startY = 0, startH = 0
let startX = 0, startW = 0

const startBottomResize = (e) => {
  dragging = true; startY = e.clientY; startH = bottomPanelHeight.value
  document.addEventListener('mousemove', onBottomMove)
  document.addEventListener('mouseup', stopBottomResize)
}
const onBottomMove = (e) => {
  if (!dragging) return
  bottomPanelHeight.value = Math.max(60, Math.min(800, startH + (startY - e.clientY)))
}
const stopBottomResize = () => {
  dragging = false
  document.removeEventListener('mousemove', onBottomMove)
  document.removeEventListener('mouseup', stopBottomResize)
  savePanelSize()
}

const startRightResize = (e) => {
  dragging = true; startX = e.clientX; startW = rightPanelWidth.value
  document.addEventListener('mousemove', onRightMove)
  document.addEventListener('mouseup', stopRightResize)
}
const onRightMove = (e) => {
  if (!dragging) return
  rightPanelWidth.value = Math.max(260, Math.min(900, startW + (startX - e.clientX)))
}
const stopRightResize = () => {
  dragging = false
  document.removeEventListener('mousemove', onRightMove)
  document.removeEventListener('mouseup', stopRightResize)
  savePanelSize()
}

const handleKeydown = (e) => {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return
  if (e.ctrlKey && e.key === 'b') { e.preventDefault(); state.sidebarVisible = !state.sidebarVisible }
  if (e.ctrlKey && e.key === '`') { e.preventDefault(); state.bottomPanelVisible = !state.bottomPanelVisible }
  if (e.ctrlKey && e.shiftKey && e.key === 'E') { e.preventDefault(); state.activeActivity = 'explorer'; state.sidebarVisible = true }
  if (e.ctrlKey && e.shiftKey && e.key === 'F') { e.preventDefault(); state.activeActivity = 'search'; state.sidebarVisible = true }
  if (e.ctrlKey && e.shiftKey && e.key === 'T') { e.preventDefault(); state.rightPanelVisible = true }
  if (e.ctrlKey && e.shiftKey && e.key === 'C') { e.preventDefault(); state.rightPanelVisible = !state.rightPanelVisible }
  if (e.ctrlKey && e.key === 'k') { e.preventDefault(); state.focusMode = !state.focusMode }
}

const loadFileTree = async () => {
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
provide('loadFileTree', loadFileTree)

// ── 文件树自动刷新防抖：3 秒内不重复刷新 ──
let _lastRefreshTime = 0
const MIN_TREE_REFRESH_INTERVAL = 3000
let _savedTreeScrollTop = 0

onMounted(async () => {
  // ★ 先恢复 UI 布局偏好（视图/面板/主题），再做任何可能触发
  //    savePersistentState 的操作（applyTheme 等）。否则默认值
  //    explorer 会被先写入 localStorage，覆盖用户上次的 chat 视图。
  loadPersistentState()
  document.addEventListener('contextmenu', (e) => {
    if (!e.defaultPrevented) e.preventDefault()
  }, false)
  document.addEventListener('keydown', handleKeydown)

  try {
    const health = await api.apiGet('/health')
    state.workspaceRoot = health.workspace || ''
    state.workspaceFolders = health.folders || []
    state.workspaceName = state.workspaceRoot
      ? state.workspaceRoot.split('\\').filter(Boolean).pop() || state.workspaceRoot
      : '未设置工作区'
    document.title = 'PairCode IDE - ' + state.workspaceName
  } catch {}

  try {
    const resp = await api.apiGet('/settings')
    // 后端返回 {settings: core.Settings, loaded} 包装结构，解包后使用
    const settings = resp.settings || resp
    state.settings = settings
    state.settingsLoaded = true
    if (settings.theme && ['dark', 'light', 'warm', 'night'].includes(settings.theme)) {
      applyTheme(settings.theme)
    }
  } catch {}

  await loadWsList()
  if (state.workspaceRoot) {
    await loadConversationsForWorkspace(state.workspaceRoot)
    try {
      const list = await api.apiGet('/conversations', { workspace: state.workspaceRoot })
      if (list && list.length > 0) state.conversations = list
    } catch {}
    if (state.conversations.length > 0 && !state.rightPanelVisible) {
      state.rightPanelVisible = true
    }

    // ★ 从后端对话列表中自动选中最近更新的对话
    if (state.conversations.length > 0 && !state.currentConvId) {
      state.currentConvId = state.conversations[0].id
    }
  }

  // ★ 初始化 WebSocket（延迟 200ms，让 switchConv 先启动加载历史）
  //    processStatus 不再创建占位，所以 WS 早连也不会有副作用
  setTimeout(() => {
    api.initWebSocket({
      onStatus: (payload) => processStatus(payload),
      onEvent: (convId, data) => processAgentEvent(convId, data),
      onDone: (convId, data) => processAgentDone(convId, data),
      onDisconnected: () => processAllDisconnected(),
    })
  }, 200)

  if (state.openFiles.length > 0) {
    for (const fp of state.openFiles) {
      try {
        const content = await api.apiGet('/fs/read', { path: fp })
        state.fileContents[fp] = content
        state.fileDirty[fp] = false
      } catch {}
    }
  }
  if (state.activeFile && !state.openFiles.includes(state.activeFile)) {
    state.activeFile = state.openFiles.length > 0 ? state.openFiles[0] : ''
  }

  await loadFileTree()

  const _onRefreshTree = () => {
    const now = Date.now()
    if (now - _lastRefreshTime < MIN_TREE_REFRESH_INTERVAL) return
    _lastRefreshTime = now

    // ★ 保存滚动位置
    const scrollEl = document.querySelector('.project-section')
    if (scrollEl) _savedTreeScrollTop = scrollEl.scrollTop

    // 只清除未打开文件的缓存，已打开文件保留内容并重新读取（确保 AI 修改后同步）
    for (const path of Object.keys(state.fileContents)) {
      if (state.openFiles.includes(path)) {
        // 已打开文件：非 dirty 则重新读取最新内容
        if (!state.fileDirty[path]) {
          api.apiGet('/fs/read', { path }).then(data => {
            const normalized = (data.content || '').replace(/\r\n/g, '\n')
            state.fileContents[path] = normalized
            state.fileSavedContent[path] = normalized
            state.fileDirty[path] = false
          }).catch(() => {})
        }
        // dirty 文件保留用户编辑内容，不清除
      } else {
        // 未打开文件：清除缓存
        delete state.fileContents[path]
        delete state.fileSavedContent[path]
        delete state.fileDirty[path]
      }
    }
    loadFileTree().then(() => {
      // ★ 恢复滚动位置
      if (_savedTreeScrollTop > 0) {
        nextTick(() => {
          const c = document.querySelector('.project-section')
          if (c) c.scrollTop = _savedTreeScrollTop
        })
      }
    })
  }
  const _onSwitchActivity = (e) => { if (e.detail?.id) switchActivity(e.detail.id) }
  const _onOpenMarketplace = () => { showMarketplace.value = true }
  const _onOpenSettings = () => { showSettings.value = true }
  const _onStopAgent = () => { window.dispatchEvent(new CustomEvent('agent-stop')) }
  const _onSaveConversations = async () => { checkNotifications() }
  const _onOpenWorkspaceDialog = () => { state.activeActivity = 'explorer'; state.sidebarVisible = true }
  const _onSwitchWorkspace = async (e) => { if (e.detail?.path) await switchWorkspace(e.detail.path) }

  window.addEventListener('refresh-tree', _onRefreshTree)
  window.addEventListener('switch-activity', _onSwitchActivity)
  window.addEventListener('open-marketplace', _onOpenMarketplace)
  window.addEventListener('open-settings', _onOpenSettings)
  window.addEventListener('stop-agent', _onStopAgent)
  window.addEventListener('save-conversations', _onSaveConversations)
  window.addEventListener('open-workspace-dialog', _onOpenWorkspaceDialog)
  window.addEventListener('switch-workspace', _onSwitchWorkspace)

  // ★ 定时轮询文件树（检测 agent 新建/修改的文件），仅页面可见时执行
  let _refreshTimer = setInterval(() => {
    if (document.visibilityState === 'visible') {
      _lastRefreshTime = 0 // 重置防抖，让下一轮 refresh-tree 事件能触发
      window.dispatchEvent(new CustomEvent('refresh-tree'))
    }
  }, 5000)

  const _cleanupEvents = () => {
    if (_refreshTimer) { clearInterval(_refreshTimer); _refreshTimer = null }
    window.removeEventListener('refresh-tree', _onRefreshTree)
    window.removeEventListener('switch-activity', _onSwitchActivity)
    window.removeEventListener('open-marketplace', _onOpenMarketplace)
    window.removeEventListener('open-settings', _onOpenSettings)
    window.removeEventListener('stop-agent', _onStopAgent)
    window.removeEventListener('save-conversations', _onSaveConversations)
    window.removeEventListener('open-workspace-dialog', _onOpenWorkspaceDialog)
    window.removeEventListener('switch-workspace', _onSwitchWorkspace)
  }
  window._cleanupAppEvents = _cleanupEvents
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
  if (window._cleanupAppEvents) { window._cleanupAppEvents(); delete window._cleanupAppEvents }
  if (persistTimer) { clearTimeout(persistTimer); persistTimer = null }
  api.closeWebSocket()
})

state.notificationCount = 0
state.workspaceName = state.workspaceName || ''

watch(() => state.messages.length, async () => {
  saveCurrentConversations(); checkNotifications()
})

let persistTimer = null
function schedulePersist() {
  savePersistentState()
}

watch(() => state.sidebarVisible, schedulePersist)
watch(() => state.rightPanelVisible, schedulePersist)
watch(() => state.bottomPanelVisible, schedulePersist)

watch(() => state.activeActivity, schedulePersist)
watch(() => state.theme, (t) => { if (t) applyTheme(t); schedulePersist() })
watch(() => state.activeFile, schedulePersist)
watch(() => state.openFiles.length, schedulePersist)
</script>

<style scoped>
.app-root {
  display: grid;
  grid-template-columns: 48px auto 1fr auto;
  grid-template-rows: 30px 1fr 22px;
  width: 100%; height: 100%;
  background: var(--bg-primary);
  color: var(--text-primary);
  overflow: hidden;
  font-family: var(--font-ui);
}
/* ★ 桌面端面板独立模式：只渲染右侧面板，占满整个窗口 */
.app-root.panel-only {
  grid-template-columns: 1fr;
  grid-template-rows: 1fr;
}
.app-root.panel-only .right-container {
  grid-column: 1; grid-row: 1;
  width: 100% !important;
  height: 100%;
}
.app-root.panel-only .right-panel-resizer { display: none; }
.titlebar {
  grid-column: 1 / -1; grid-row: 1;
  display: flex; align-items: center; height: 30px;
  background: var(--bg-tertiary);
  border-bottom: 1px solid var(--border-color);
  z-index: 100; overflow: visible;
  -webkit-app-region: drag;
}
.app-logo {
  width: 48px; display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  -webkit-app-region: no-drag;
}
.logo-img {
  width: 18px; height: 18px;
}
.title-center {
  flex: 1; text-align: center; font-size: 12px; color: var(--text-muted);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap; padding: 0 8px;
}
.title-right {
  display: flex; align-items: center; padding-right: 8px; gap: 6px;
  -webkit-app-region: no-drag;
}
.ws-quick-btn {
  background: none; border: 1px solid var(--border-color); color: var(--text-secondary);
  padding: 2px 8px; border-radius: 3px; cursor: pointer; display: flex; align-items: center;
}
.ws-quick-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.activity-bar { grid-column: 1; grid-row: 2; z-index: 20; }
.sidebar { grid-column: 2; grid-row: 2; z-index: 10; overflow: hidden; }
.main-area {
  grid-column: 3; grid-row: 2;
  display: flex; flex-direction: column; min-width: 0; overflow: hidden;
}
.main-area > :first-child { flex: 1; }
.right-container {
  grid-column: 4; grid-row: 2;
  display: flex; flex-direction: row; overflow: hidden; position: relative;
}
.right-container.focus-mode { grid-column: 2 / -1; }
.right-panel-resizer {
  width: 4px; cursor: ew-resize; background: transparent; flex-shrink: 0; z-index: 10;
}
.right-panel-resizer:hover { background: var(--accent); }
.status-bar { grid-column: 1 / -1; grid-row: 3; z-index: 30; }
.bottom-panel {
  position: relative; background: var(--bg-secondary);
  border-top: 1px solid var(--border-color);
  display: flex; flex-direction: column; min-height: 60px;
}
.panel-content { flex: 1; overflow: hidden; padding: 0; }
.panel-resizer { position: absolute; top: -3px; left: 0; right: 0; height: 6px; cursor: ns-resize; z-index: 10; }
</style>
