<template>
  <div class="file-explorer">
    <!-- 顶部工具栏 -->
    <div class="explorer-toolbar">
      <span class="tb-title">工作区</span>
      <span class="tb-spacer"></span>
      <button class="tb-btn" @click="showWorkspaceDialog = true" title="新建工作区"><SvgIcon name="plus" :size="14" /></button>
      <button class="tb-btn" @click="refreshAll" title="刷新"><SvgIcon name="refresh" :size="14" /></button>
    </div>

    <!-- ── 上方：工作区列表 ── -->
    <div class="ws-section">
      <div v-for="ws in state.wsList" :key="ws.path"
           :class="['ws-item', { 'ws-active': ws.path === state.workspaceRoot }]"
           @click="switchToWorkspace(ws)"
           @contextmenu.prevent="showWsContextMenu($event, ws)">
        <div class="ws-left">
          <SvgIcon name="folder" :size="14" class="ws-icon" />
          <span class="ws-name">{{ ws.name }}</span>
        </div>
        <div class="ws-right">
          <span v-if="ws.notify" class="ws-notify" title="有待处理">●</span>
          <span v-if="ws.path === state.workspaceRoot" class="ws-badge">当前</span>
        </div>
      </div>
      <div v-if="state.wsList.length === 0" class="ws-empty">
        <span>暂无工作区</span>
        <button class="ws-create-btn" @click="showWorkspaceDialog = true">创建</button>
      </div>
    </div>

    <!-- ── 分隔线 ── -->
    <div class="ws-divider">
      <span class="divider-label">项目</span>
    </div>

    <!-- ── 下方：当前工作区的项目列表 ── -->
    <div class="project-section">
      <div v-if="!state.workspaceRoot" class="proj-empty">请先选择工作区</div>
      <template v-else-if="currentFolders.length > 0">
        <FileTreeItem
          v-for="folder in currentFolders"
          :key="folder"
          :item="{ name: folder.split('\\').pop(), isDir: true, path: folder }"
          :parentPath="folderParent(folder)"
          :depth="0"
          :defaultExpanded="true"
          @file-click="openFile"
        />
      </template>
      <div v-else class="proj-empty">
        <span>暂无项目，在工作区上右键添加</span>
      </div>
    </div>

    <!-- ===== 新建工作区对话框 ===== -->
    <div v-if="showWorkspaceDialog" class="dialog-overlay" @click.self="showWorkspaceDialog = false">
      <div class="dialog-box" style="max-width:420px">
        <div class="dialog-title">新建工作区</div>
        <div class="dialog-body">
          <label>工作区名称</label>
          <input v-model="newWsName" class="dlg-input" placeholder="例如: my-project" @keyup.enter="createWorkspace" />
          <label>保存路径（可选）</label>
          <div class="input-row">
            <input v-model="newWsPath" class="dlg-input flex-1" placeholder="留空则用默认目录" />
            <button class="dlg-btn-sm" @click="openBrowseDialog('ws-path')">浏览</button>
          </div>
        </div>
        <div class="dialog-footer">
          <span v-if="wsError" class="dlg-error">{{ wsError }}</span>
          <button class="dlg-btn" @click="showWorkspaceDialog = false">取消</button>
          <button class="dlg-btn primary" @click="createWorkspace" :disabled="!newWsName.trim()">创建</button>
        </div>
      </div>
    </div>

    <!-- ===== 目录浏览对话框 ===== -->
    <div v-if="browseVisible" class="dialog-overlay" @click.self="closeBrowse">
      <div class="dialog-box dir-browser-box">
        <div class="dialog-title">{{ browseTitle }}</div>
        <div class="dir-browser">
          <div class="dir-breadcrumb">
            <button class="bc-btn" @click="browseGoUp" :disabled="browsePath === ''">
              <SvgIcon name="chevron-right" :size="14" style="transform:rotate(180deg)" />
            </button>
            <span class="bc-path">{{ browsePath || '选择驱动器...' }}</span>
          </div>
          <div v-if="browsePath === ''" class="dir-list">
            <div v-for="drive in browseDrives" :key="drive"
                 class="dir-item dir-drive" @dblclick="browseEnter(drive)">
              <SvgIcon name="drive" :size="14" />
              <span class="dir-name">{{ drive }}</span>
            </div>
          </div>
          <div v-else class="dir-list">
            <div v-for="entry in browseEntries" :key="entry.name"
                 :class="['dir-item', { 'dir-selected': browseSelected === browsePath + '\\' + entry.name }]"
                 @click="browseSelect(entry)">
              <SvgIcon :name="entry.isDir ? 'folder' : 'file'" :size="14" />
              <span class="dir-name">{{ entry.name }}</span>
            </div>
            <div v-if="browseEntries.length === 0" class="dir-empty">空目录</div>
          </div>
        </div>
        <div class="dialog-footer">
          <span v-if="browseError" class="dlg-error">{{ browseError }}</span>
          <input v-if="browseMode === 'new'" v-model="newProjectName" class="dlg-input"
                 style="flex:1;margin-right:8px" placeholder="项目名称" @keyup.enter="browseConfirm" />
          <button class="dlg-btn" @click="closeBrowse">取消</button>
          <button class="dlg-btn primary" @click="browseConfirm" :disabled="browseConfirmDisabled">确认</button>
        </div>
      </div>
    </div>
    <!-- 右键菜单 -->
    <ContextMenu ref="wsContextMenuRef" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { state } from '../main.js'
import api from '../api.js'
import FileTreeItem from './FileTreeItem.vue'
import SvgIcon from './SvgIcon.vue'
import ContextMenu from './ContextMenu.vue'

// ── 当前工作区的文件夹列表 ──
const currentFolders = computed(() => {
  if (!state.workspaceRoot) return []
  // 从 wsList 中找到当前工作区
  const cur = state.wsList.find(w => w.path === state.workspaceRoot)
  if (cur && cur.folders && cur.folders.length > 0) {
    return [...new Set(cur.folders.filter(Boolean))]
  }
  if (state.workspaceFolders.length > 0) return [...new Set(state.workspaceFolders.filter(Boolean))]
  return [state.workspaceRoot]
})

// ── 计算文件夹的父路径（用于 FileTreeItem）──
function folderParent(folderPath) {
  const idx = folderPath.lastIndexOf('\\')
  return idx > 0 ? folderPath.substring(0, idx) : ''
}

// ── 切换工作区 ──
async function switchToWorkspace(ws) {
  if (ws.path === state.workspaceRoot) return

  // 保存当前对话
  saveCurrentConversations()
  try {
    const folders = ws.folders || []
    const res = await api.apiPost('/workspace', {
      action: 'switch', root: ws.path,
      folders: folders.filter(f => f !== ws.path),
    })
    state.workspaceRoot = ws.path
    state.workspaceFolders = folders.length > 0 ? [...folders] : [ws.path]
    state.workspaceName = ws.name || ws.path.split('\\').filter(Boolean).pop() || ws.path
    document.title = 'PairCode IDE - ' + state.workspaceName

    // 清空编辑器
    state.openFiles = []
    state.activeFile = ''
    state.fileContents = {}

    // 加载目标工作区对话
    await loadConversationsForWorkspace(ws.path)

    // 清除通知
    ws.notify = false
    state.notificationCount = 0
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
      state.messages = data.messages || []
    }
  } catch {}
}

// ── 新建工作区 ──
const showWorkspaceDialog = ref(false)
const newWsName = ref('')
const newWsPath = ref('')
const wsError = ref('')

// ── 工作区右键菜单 ──
const wsContextMenuRef = ref(null)

async function showWsContextMenu(e, ws) {
  await nextTick()
  const result = await wsContextMenuRef.value.show({
    x: e.clientX, y: e.clientY,
    title: ws.name,
    items: [
      { label: '添加项目', icon: 'plus', action: 'add-project' },
      { label: '新建项目', icon: 'folder-plus', action: 'new-project' },
      { separator: true },
      { label: '重命名工作区', action: 'rename' },
      { label: '删除工作区', action: 'delete' },
      { separator: true },
      { label: '在终端中打开', icon: 'terminal', action: 'open-terminal' },
      { label: '复制路径', icon: 'copy', action: 'copy-path' },
    ],
  })
  if (!result) return
  switch (result) {
    case 'add-project':
      openBrowseDialog('add')
      break
    case 'new-project':
      // 先检查是否有活动工作区
      state.workspaceRoot = ws.path
      state.workspaceName = ws.name
      openBrowseDialog('new')
      break
    case 'rename': {
      const name = await window.$prompt('新名称:', ws.name, '重命名工作区')
      if (name && name.trim()) {
        ws.name = name.trim()
        saveWsList()
      }
      break
    }
    case 'delete':
      if (!(await window.$confirm(`确认删除工作区 "${ws.name}" ？（不会删除文件）`))) return
      state.wsList = state.wsList.filter(w => w.path !== ws.path)
      saveWsList()
      if (state.workspaceRoot === ws.path) {
        state.workspaceRoot = state.wsList[0]?.path || ''
        state.workspaceName = state.wsList[0]?.name || ''
        if (state.workspaceRoot) await switchToWorkspace(state.wsList[0])
      }
      break
    case 'open-terminal': {
      state.bottomPanelVisible = true
      state.bottomPanelTab = 'terminal'
      window.dispatchEvent(new CustomEvent('terminal-cwd', { detail: { cwd: ws.path } }))
      break
    }
    case 'copy-path':
      navigator.clipboard.writeText(ws.path).catch(() => {})
      break
  }
}

async function createWorkspace() {
  const name = newWsName.value.trim()
  if (!name) return
  wsError.value = ''
  try {
    const res = await api.apiPost('/workspace', { action: 'create', name, root: newWsPath.value.trim() || '' })
    if (res.ok || res.root) {
      const newPath = res.root || ''
      const ws = { path: newPath, name, folders: [newPath], notify: false }
      state.wsList.push(ws)
      showWorkspaceDialog.value = false
      newWsName.value = ''
      newWsPath.value = ''
      await switchToWorkspace(ws)
      saveWsList()
    }
  } catch (err) { wsError.value = err.message }
}

function saveWsList() {
  try {
    const data = state.wsList.map(w => ({
      path: w.path, name: w.name,
      folders: w.folders || [], notify: !!w.notify,
    }))
    localStorage.setItem('paircode-workspaces', JSON.stringify(data.slice(0, 20)))
  } catch {}
}

// ── 目录浏览 ──
const browseVisible = ref(false)
const browseMode = ref('add')
const browsePath = ref('')
const browseDrives = ref([])
const browseEntries = ref([])
const browseSelected = ref('')
const browseError = ref('')
const newProjectName = ref('')

const browseTitle = computed(() => ({
  add: '选择项目目录',
  new: '选择父目录创建新项目',
  'ws-path': '选择工作区保存路径',
}[browseMode.value] || '浏览目录'))

const browseConfirmDisabled = computed(() => {
  if (browseMode.value === 'new') return !newProjectName.value.trim()
  return !browseSelected.value && !browsePath.value
})

function openBrowseDialog(mode) {
  browseMode.value = mode
  browseVisible.value = true
  browseError.value = ''
  newProjectName.value = ''
  browsePath.value = ''
  browseSelected.value = ''
  browseEntries.value = []
  api.apiGet('/fs/drives').then(d => { browseDrives.value = d || [] }).catch(() => {})
}

function closeBrowse() { browseVisible.value = false; browseError.value = '' }

function browseSelect(entry) {
  if (!entry.isDir) return
  const full = browsePath.value + '\\' + entry.name
  browseSelected.value = full
  // 展开
  browsePath.value = full
  loadBrowseDir(full)
}

async function browseEnter(path) {
  browsePath.value = path
  browseSelected.value = ''
  loadBrowseDir(path)
}

async function browseGoUp() {
  if (!browsePath.value) return
  const parts = browsePath.value.replace(/\\$/, '').split('\\')
  if (parts.length <= 1) {
    browsePath.value = ''
    browseEntries.value = []
    browseSelected.value = ''
    return
  }
  parts.pop()
  browsePath.value = parts.join('\\') + '\\'
  loadBrowseDir(browsePath.value)
}

async function loadBrowseDir(path) {
  try {
    browseEntries.value = await api.apiGet('/fs/list', { path })
  } catch (err) { browseError.value = err.message }
}

async function browseConfirm() {
  browseError.value = ''
  if (browseMode.value === 'new') {
    const name = newProjectName.value.trim()
    if (!name) return
    const dir = browsePath.value
    if (!dir) { browseError.value = '请先选择父目录'; return }
    try {
      await api.apiPost('/workspace', { action: 'new-project', name, parentDir: dir })
      await refreshCurrentWs()
      closeBrowse()
    } catch (err) { browseError.value = err.message }
  } else if (browseMode.value === 'ws-path') {
    const p = browseSelected.value || browsePath.value
    if (!p) { browseError.value = '请先选择目录'; return }
    newWsPath.value = p
    closeBrowse()
  } else {
    const p = browseSelected.value || browsePath.value
    if (!p) { browseError.value = '请先选择目录'; return }
    try {
      await api.apiPost('/workspace', { action: 'add-folder', path: p })
      await refreshCurrentWs()
      closeBrowse()
    } catch (err) { browseError.value = err.message }
  }
}

async function refreshCurrentWs() {
  try {
    const health = await api.apiGet('/health')
    state.workspaceFolders = health.folders || []
    const cur = state.wsList.find(w => w.path === state.workspaceRoot)
    if (cur) cur.folders = [...state.workspaceFolders]
    saveWsList()
  } catch {}
}

async function refreshAll() {
  try {
    const health = await api.apiGet('/health')
    state.workspaceFolders = health.folders || []
    state.workspaceRoot = health.workspace || state.workspaceRoot
    for (const ws of state.wsList) {
      if (ws.path === state.workspaceRoot) {
        ws.folders = [...state.workspaceFolders]
      }
    }
    saveWsList()
  } catch {}
  window.dispatchEvent(new CustomEvent('refresh-tree'))
}

function openFile(path) {
  if (!state.openFiles.includes(path)) state.openFiles.push(path)
  state.activeFile = path
  loadFileContent(path)
}

async function loadFileContent(path) {
  if (state.fileContents[path]) return
  try {
    const data = await api.apiGet('/fs/read', { path })
    state.fileContents[path] = data.content || ''
    state.fileDirty[path] = false
  } catch (err) {
    state.fileContents[path] = `// 错误: ${err.message}`
  }
}

// ── 生命周期 ──
onMounted(() => {
  window.addEventListener('refresh-tree', refreshAll)
  window.addEventListener('refresh-workspace', refreshCurrentWs)
})
onUnmounted(() => {
  window.removeEventListener('refresh-tree', refreshAll)
  window.removeEventListener('refresh-workspace', refreshCurrentWs)
})
</script>

<style scoped>
.file-explorer { font-size: 13px; display: flex; flex-direction: column; height: 100%; }

/* ── 工具栏 ── */
.explorer-toolbar {
  display: flex; align-items: center; gap: 2px; padding: 4px 8px;
  border-bottom: 1px solid var(--border-color); flex-shrink: 0;
}
.tb-title { font-size: 11px; font-weight: 600; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px; }
.tb-spacer { flex: 1; }
.tb-btn {
  background: none; border: 1px solid transparent; color: var(--text-secondary);
  padding: 2px 6px; cursor: pointer; border-radius: 3px; line-height: 1; display: flex; align-items: center;
}
.tb-btn:hover { background: var(--bg-hover); color: var(--text-primary); }

/* ── 工作区列表 ── */
.ws-section {
  flex-shrink: 0;
  max-height: 40%;
  overflow-y: auto;
  padding: 4px 0;
  border-bottom: 1px solid var(--border-color);
}
.ws-item {
  display: flex; align-items: center; padding: 6px 10px; cursor: pointer;
  justify-content: space-between;
}
.ws-item:hover { background: var(--bg-hover); }
.ws-item.ws-active { background: var(--accent-bg); border-left: 2px solid var(--accent); padding-left: 8px; }
.ws-left { display: flex; align-items: center; gap: 6px; min-width: 0; flex: 1; }
.ws-icon { flex-shrink: 0; color: var(--accent); }
.ws-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-primary); font-size: 13px; }
.ws-right { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
.ws-notify { color: #d4a74e; font-size: 10px; }
.ws-badge {
  font-size: 9px; color: var(--accent); background: rgba(126,184,218,0.15);
  padding: 1px 6px; border-radius: 3px;
}
.ws-empty { padding: 16px 10px; text-align: center; color: var(--text-muted); font-size: 12px; display: flex; align-items: center; justify-content: center; gap: 6px; }
.ws-create-btn { background: var(--accent); color: #000; border: none; padding: 2px 10px; border-radius: 3px; cursor: pointer; font-size: 12px; }

/* ── 分隔线 ── */
.ws-divider {
  display: flex; align-items: center; padding: 4px 8px; flex-shrink: 0;
  user-select: none;
}
.ws-divider::before, .ws-divider::after {
  content: ''; flex: 1; height: 1px; background: var(--border-color);
}
.divider-label {
  font-size: 10px; color: var(--text-muted); padding: 0 8px;
  text-transform: uppercase; letter-spacing: 0.5px; font-weight: 600;
}

/* ── 项目列表 ── */
.project-section { flex: 1; overflow-y: auto; padding: 2px 0; }
.proj-empty { padding: 24px 12px; text-align: center; color: var(--text-muted); font-size: 12px; display: flex; flex-direction: column; align-items: center; gap: 8px; }
.proj-add-btn { background: var(--bg-tertiary); border: 1px solid var(--border-color); color: var(--text-primary); padding: 4px 12px; border-radius: 3px; cursor: pointer; font-size: 12px; }
.proj-add-btn:hover { background: var(--bg-hover); }
.proj-actions {
  display: flex; gap: 4px; padding: 6px 8px; flex-shrink: 0;
  border-top: 1px solid var(--border-color);
}
.pa-btn {
  display: flex; align-items: center; gap: 4px;
  background: var(--bg-tertiary); border: 1px solid var(--border-color);
  color: var(--text-secondary); padding: 3px 8px; border-radius: 3px;
  cursor: pointer; font-size: 11px; flex: 1; justify-content: center;
}
.pa-btn:hover { background: var(--bg-hover); color: var(--text-primary); }

/* ── 对话框样式（复用） ── */
.dialog-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); z-index: 1000; display: flex; align-items: center; justify-content: center; }
.dialog-box { background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: var(--border-radius-lg); padding: 20px; min-width: 320px; max-width: 600px; width: 90%; box-shadow: var(--shadow-md); }
.dialog-title { font-size: 16px; font-weight: 600; margin-bottom: 16px; color: var(--text-primary); }
.dialog-body { display: flex; flex-direction: column; gap: 8px; margin-bottom: 16px; }
.dialog-body label { font-size: 12px; color: var(--text-secondary); margin-top: 4px; }
.dlg-input { background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary); padding: 8px 10px; border-radius: 4px; font-size: 13px; outline: none; }
.dlg-input:focus { border-color: var(--accent); }
.flex-1 { flex: 1; }
.input-row { display: flex; gap: 6px; align-items: center; }
.dlg-btn-sm { background: var(--bg-tertiary); border: 1px solid var(--border-color); color: var(--text-primary); padding: 7px 12px; border-radius: 4px; cursor: pointer; font-size: 12px; white-space: nowrap; }
.dialog-footer { display: flex; align-items: center; gap: 8px; justify-content: flex-end; flex-wrap: wrap; }
.dlg-btn { background: var(--bg-tertiary); border: 1px solid var(--border-color); color: var(--text-primary); padding: 8px 20px; border-radius: 4px; cursor: pointer; font-size: 13px; }
.dlg-btn:hover { background: var(--bg-hover); }
.dlg-btn.primary { background: var(--accent); color: #000; border-color: var(--accent); }
.dlg-btn.primary:hover { filter: brightness(1.1); }
.dlg-btn:disabled { opacity: 0.5; cursor: default; }
.dlg-error { flex: 1; font-size: 12px; color: #e74c3c; }

/* 目录浏览器 */
.dir-browser-box { max-width: 560px; }
.dir-browser { border: 1px solid var(--border-color); border-radius: 4px; max-height: 280px; overflow: auto; background: var(--bg-primary); margin-bottom: 12px; }
.dir-breadcrumb { display: flex; align-items: center; gap: 6px; padding: 6px 8px; border-bottom: 1px solid var(--border-color); position: sticky; top: 0; background: var(--bg-primary); z-index: 1; }
.bc-btn { background: none; border: 1px solid var(--border-color); color: var(--text-primary); padding: 2px 8px; border-radius: 3px; cursor: pointer; display: flex; align-items: center; }
.bc-btn:hover { background: var(--bg-hover); }
.bc-btn:disabled { opacity: 0.4; cursor: default; }
.bc-path { font-size: 12px; color: var(--text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; }
.dir-list { padding: 2px 0; }
.dir-item { display: flex; align-items: center; gap: 6px; padding: 4px 8px; cursor: pointer; font-size: 13px; }
.dir-item:hover { background: var(--bg-hover); }
.dir-selected { background: var(--bg-active); }
.dir-drive { padding: 6px 8px; }
.dir-name { flex: 1; color: var(--text-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dir-empty { padding: 16px; text-align: center; color: var(--text-muted); font-size: 12px; }
</style>
