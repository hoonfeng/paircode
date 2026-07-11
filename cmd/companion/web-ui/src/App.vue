<template>
  <div class="app-root">
    <!-- 标题栏 + 菜单栏 -->
    <div class="titlebar" @click="closeAllMenus">
      <div class="app-logo">
        <SvgIcon name="code" :size="16" color="#0e639c" />
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

    <!-- 内容区域 -->
    <ActivityBar />
    <Sidebar v-if="state.sidebarVisible" />
    <div v-if="!state.focusMode" class="main-area">
      <EditorArea />
      <div class="bottom-panel" v-if="state.bottomPanelVisible"
           :style="{ height: bottomPanelHeight + 'px' }">
        <div class="panel-tabs">
          <button :class="{ active: state.bottomPanelTab === 'output' }"
                  @click="state.bottomPanelTab = 'output'">
            <SvgIcon name="output" size="14" /> 输出
          </button>
          <button :class="{ active: state.bottomPanelTab === 'tasks' }"
                  @click="state.bottomPanelTab = 'tasks'">
            <SvgIcon name="check" size="14" /> 任务
          </button>
          <button :class="{ active: state.bottomPanelTab === 'terminal' }"
                  @click="state.bottomPanelTab = 'terminal'">
            <SvgIcon name="terminal" size="14" /> 终端
          </button>
          <div class="panel-actions">
            <span class="panel-notify" v-if="state.notificationCount > 0" title="有待处理通知">
              <SvgIcon name="bell" :size="12" /> {{ state.notificationCount }}
            </span>
            <button class="panel-close" @click="state.bottomPanelVisible = false">
              <SvgIcon name="close" size="14" />
            </button>
          </div>
        </div>
        <div class="panel-content">
          <OutputPanel v-if="state.bottomPanelTab === 'output'" />
          <TasksPanel v-if="state.bottomPanelTab === 'tasks'" />
          <TerminalPanel v-if="state.bottomPanelTab === 'terminal'" />
        </div>
        <div class="panel-resizer" @mousedown.prevent="startBottomResize"></div>
      </div>
    </div>

    <!-- 右侧容器 -->
    <div v-if="state.rightPanelVisible" class="right-container"
         :class="{ 'focus-mode': state.focusMode }"
         :style="state.focusMode ? {} : { width: (rightPanelWidth + 4 + 1 + 250) + 'px' }">
      <div class="right-panel-resizer" @mousedown.prevent="startRightResize"></div>
      <RightPanel />
    </div>

    <!-- 状态栏 -->
    <StatusBar />

    <!-- 模态框 -->
    <SettingsModal v-if="showSettings" @close="showSettings = false" />
    <SystemModal v-if="showSystem" @close="showSystem = false" />
    <SourceModal v-if="showSource" @close="showSource = false" />
    <MarketplaceModal v-if="showMarketplace" @close="showMarketplace = false" />
    <GlobalDialogs />
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onUnmounted, provide, nextTick } from 'vue'
import { state, savePersistentState, loadPersistentState, applyTheme } from './main.js'
import api from './api.js'

import MenuBar from './components/MenuBar.vue'
import ActivityBar from './components/ActivityBar.vue'
import Sidebar from './components/Sidebar.vue'
import EditorArea from './components/EditorArea.vue'
import RightPanel from './components/RightPanel.vue'
import StatusBar from './components/StatusBar.vue'
import OutputPanel from './components/OutputPanel.vue'
import TasksPanel from './components/TasksPanel.vue'
import TerminalPanel from './components/TerminalPanel.vue'
import SettingsModal from './components/SettingsModal.vue'
import SystemModal from './components/SystemModal.vue'
import SourceModal from './components/SourceModal.vue'
import MarketplaceModal from './components/MarketplaceModal.vue'
import SvgIcon from './components/SvgIcon.vue'
import GlobalDialogs from './components/GlobalDialogs.vue'

const showSettings = ref(false)
const showSystem = ref(false)
const showSource = ref(false)
const showMarketplace = ref(false)
const showQuickSwitcher = ref(false)

function loadPanelSize() {
  try {
    const d = JSON.parse(localStorage.getItem('paircode-panel-size') || '{}')
    if (d.rpw) rightPanelWidth.value = d.rpw
    if (d.bph) bottomPanelHeight.value = d.bph
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
const rightPanelWidth = ref(600)

provide('showSettings', showSettings)
provide('showSystem', showSystem)
provide('showSource', showSource)
provide('showMarketplace', showMarketplace)
provide('bottomPanelHeight', bottomPanelHeight)
provide('rightPanelWidth', rightPanelWidth)

if (!state.wsList) state.wsList = reactive([])
const wsList = state.wsList

provide('wsList', wsList)
provide('saveWsList', saveWsList)
provide('switchWorkspace', switchWorkspace)

function loadWsList() {
  try {
    const saved = JSON.parse(localStorage.getItem('paircode-workspaces') || '[]')
    wsList.length = 0
    for (const w of saved) { wsList.push(reactive(w)) }
  } catch { wsList.length = 0 }
  if (state.workspaceRoot && !wsList.find(w => w.path === state.workspaceRoot)) {
    wsList.push(reactive({
      path: state.workspaceRoot,
      name: state.workspaceRoot.split('\\').filter(Boolean).pop() || state.workspaceRoot,
      folders: state.workspaceFolders?.length > 0 ? [...state.workspaceFolders] : [state.workspaceRoot],
      notify: false,
    }))
  }
  checkNotifications()
}

function saveWsList() {
  try {
    const data = wsList.map(w => ({
      path: w.path, name: w.name, folders: w.folders || [], notify: !!w.notify,
    }))
    localStorage.setItem('paircode-workspaces', JSON.stringify(data.slice(0, 20)))
  } catch {}
}

function checkNotifications() {
  for (const ws of wsList) {
    ws.notify = state.notificationCount > 0 && ws.path !== state.workspaceRoot
  }
}

async function switchWorkspace(targetPath) {
  if (!targetPath || targetPath === state.workspaceRoot) return
  saveCurrentConversations()
  try {
    const targetWs = wsList.find(w => w.path === targetPath)
    const folders = targetWs?.folders || []
    await api.apiPost('/workspace', {
      action: 'switch', root: targetPath,
      folders: folders.filter(f => f !== targetPath),
    })
    state.workspaceRoot = targetPath
    state.workspaceFolders = folders.length > 0 ? [...folders] : [targetPath]
    state.workspaceName = targetPath.split('\\').filter(Boolean).pop() || targetPath
    document.title = 'PairCode IDE - ' + state.workspaceName
    state.openFiles = []
    state.activeFile = ''
    state.fileContents = {}
    await loadFileTree()
    try {
      const list = await api.apiGet('/conversations')
      if (list && list.length > 0) state.conversations = list
    } catch {}
    window.dispatchEvent(new CustomEvent('workspace-switched'))
    const ws = wsList.find(w => w.path === targetPath)
    if (ws) ws.notify = false
    state.notificationCount = 0
    if (targetWs) { targetWs.folders = [...state.workspaceFolders]; saveWsList() }
    if (!wsList.find(w => w.path === targetPath)) {
      wsList.push(reactive({ path: targetPath, name: state.workspaceName, folders: [...state.workspaceFolders], notify: false }))
      saveWsList()
    }
    savePersistentState()
  } catch (err) {
    console.error('切换工作区失败:', err)
  }
}

function saveCurrentConversations() {
  if (!state.workspaceRoot) return
  try {
    const key = 'conv_' + btoa(state.workspaceRoot).slice(0, 40)
    localStorage.setItem(key, JSON.stringify({
      conversations: state.conversations,
      currentConvId: state.currentConvId,
      messages: state.messages,
    }))
  } catch {}
}

async function loadConversationsForWorkspace(path) {
  state.conversations = []
  state.currentConvId = ''
  state.messages = []
  try {
    const key = 'conv_' + btoa(path).slice(0, 40)
    const saved = localStorage.getItem(key)
    if (saved) {
      const data = JSON.parse(saved)
      state.conversations = data.conversations || []
      state.currentConvId = data.currentConvId || ''
      state.messages = (data.messages || []).map(m => {
        if (m._loading) m._loading = false
        if (!m.content && m.segments && m.segments.length > 0) {
          m.content = m.segments.filter(s => s.type === 'content').map(s => s.content || '').join('')
        }
        return m
      })
    }
  } catch {}
}

const switchActivity = (id) => {
  if (id === 'settings') { showSettings.value = true; return }
  if (id === 'system') { showSystem.value = true; return }
  if (id === 'chat') { state.rightPanelVisible = !state.rightPanelVisible; return }
  if (id === 'marketplace') { showMarketplace.value = true; return }
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

onMounted(async () => {
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
    const settings = await api.apiGet('/settings')
    state.settings = settings
    state.settingsLoaded = true
  } catch {}

  loadWsList()
  if (state.workspaceRoot) {
    await loadConversationsForWorkspace(state.workspaceRoot)
    try {
      const list = await api.apiGet('/conversations')
      if (list && list.length > 0) state.conversations = list
    } catch {}
    if (state.conversations.length > 0 && !state.rightPanelVisible) {
      state.rightPanelVisible = true
    }
  }

  loadPersistentState()

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

  const _onRefreshTree = loadFileTree
  const _onSwitchActivity = (e) => { if (e.detail?.id) switchActivity(e.detail.id) }
  const _onOpenMarketplace = () => { showMarketplace.value = true }
  const _onOpenSettings = () => { showSettings.value = true }
  const _onStopAgent = () => { window.dispatchEvent(new CustomEvent('agent-stop')) }
  const _onSaveConversations = () => { saveCurrentConversations(); checkNotifications(); saveWsList() }
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

  const _cleanupEvents = () => {
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
})

state.notificationCount = 0
state.workspaceName = state.workspaceName || ''

watch(() => state.messages.length, () => {
  saveCurrentConversations(); checkNotifications(); saveWsList()
})

let persistTimer = null
function schedulePersist() {
  if (persistTimer) clearTimeout(persistTimer)
  persistTimer = setTimeout(() => { savePersistentState(); persistTimer = null }, 1000)
}

watch(() => state.sidebarVisible, schedulePersist)
watch(() => state.rightPanelVisible, schedulePersist)
watch(() => state.bottomPanelVisible, schedulePersist)
watch(() => state.bottomPanelTab, schedulePersist)
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
.right-container.focus-mode { grid-column: 3 / -1; }
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
.panel-tabs {
  display: flex; align-items: center; background: var(--bg-tertiary);
  border-bottom: 1px solid var(--border-color); padding: 0 8px; height: 28px; flex-shrink: 0; gap: 2px;
}
.panel-tabs button {
  background: none; border: none; color: var(--text-secondary); font-size: 12px;
  padding: 4px 12px; cursor: pointer; border-top: 2px solid transparent;
  display: flex; align-items: center; gap: 4px;
}
.panel-tabs button.active { color: var(--text-primary); border-top-color: var(--accent); background: var(--bg-secondary); }
.panel-tabs button:hover { color: var(--text-primary); }
.panel-actions { margin-left: auto; display: flex; align-items: center; gap: 6px; }
.panel-notify { font-size: 11px; color: #d4a74e; display: flex; align-items: center; gap: 2px; }
.panel-close { background: none; border: none; color: var(--text-secondary); font-size: 14px; cursor: pointer; padding: 2px 6px; }
.panel-close:hover { color: var(--text-primary); }
.panel-content { flex: 1; overflow: auto; padding: 4px; }
.panel-resizer { position: absolute; top: -3px; left: 0; right: 0; height: 6px; cursor: ns-resize; z-index: 10; }
</style>
