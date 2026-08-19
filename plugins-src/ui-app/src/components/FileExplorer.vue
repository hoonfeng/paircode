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
      <div v-for="ws in wsItems" :key="ws.path"
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
      <div v-if="wsItems.length === 0" class="ws-empty">
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
          v-for="(folder, fi) in currentFolders"
          :key="folder"
          :item="{ name: folder.split('\\').pop(), isDir: true, path: folder }"
          :parentPath="folderParent(folder)"
          :depth="0"
          :defaultExpanded="true"
          :siblings="currentFolders"
          :siblingIndex="fi"
          @file-click="openFile"
        />
      </template>
      <div v-else class="proj-empty">
        <span>暂无项目，在工作区上右键添加</span>
      </div>
    </div>

    <!-- ── 工具集（卷帘：与文件树同区，工作区 .pair/toolsets/） ── -->
    <div class="ts-divider">
      <div class="ts-header" :class="{ open: tsOpen }" @click="toggleTs" title="工具集（工作区内，可折叠）">
        <SvgIcon name="package" :size="12" class="ts-header-icon" />
        <span class="divider-label ts-label">工具集</span>
        <span class="ts-spacer"></span>
        <SvgIcon name="chevron-right" :size="11" class="ts-chevron" :class="{ open: tsOpen }" />
      </div>
      <div v-if="tsOpen" class="ts-body">
        <!-- 工作区工具集（builtin 已加入内容：可移出） -->
        <div class="ts-build">
          <div class="ts-build-head">
            <span class="ts-build-title">工作区工具集</span>
            <button class="ts-btn mini" @click="openTransfer" title="穿梭框批量管理：未加入 ↔ 已加入">管理</button>
          </div>
          <input v-model="tsAddSearch" placeholder="搜索工具名…" class="ts-input" />
          <div class="ts-add-list">
            <div v-for="g in joinedGroups" :key="g.name" class="ts-add-group">
              <div class="ts-add-group-title">
                <span>{{ g.name }}</span>
              </div>
              <div v-for="t in filterTools(g.tools)" :key="t.name" class="ts-add-tool" :title="t.desc">
                <span class="ts-add-tool-name">{{ t.name }}</span>
                <button class="ts-btn mini danger" @click="toggleToolsetTool(t, g)" title="移出工作区工具集（该工具对 agent 不可见）">移出</button>
              </div>
            </div>
            <div v-if="manualToolNames.length" class="ts-add-group">
              <div class="ts-add-group-title">
                <span>_manual（手动）</span>
              </div>
              <div v-for="t in filterTools(manualToolObjs)" :key="t.name" class="ts-add-tool" :title="t.desc">
                <span class="ts-add-tool-name">{{ t.name }}</span>
                <button class="ts-btn mini danger" @click="toggleToolsetTool(t, g)" title="移出工作区工具集（该工具对 agent 不可见）">移出</button>
              </div>
            </div>
            <div v-if="!joinedGroups.length && !manualToolNames.length" class="ts-empty">未加入任何工具。点「管理」在穿梭框中加入。</div>
          </div>
          <div v-if="tsMsg" class="ts-msg" :class="{ err: tsMsgErr }">{{ tsMsg }}</div>
        </div>
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
    <!-- 工作区工具集穿梭框（未加入 ↔ 已加入 批量管理） -->
    <ToolsetTransfer
      v-if="tsTransferOpen"
      :groups="builtinInfo?.plugins || []"
      :joined="builtinInfo?.joined || []"
      :manual-tools="builtinInfo?.manualTools || []"
      :workspace-root="state.workspaceRoot"
      @close="tsTransferOpen = false"
      @changed="onTransferChanged"
    />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { state } from '../ui-state.js'
import api from '../api.js'
import FileTreeItem from './FileTreeItem.vue'
import ToolsetTransfer from './ToolsetTransfer.vue'

// ★ desktop(goja) workaround：渲染 effect 对整体赋值数组的 set 不收集依赖（v-for/v-if 不更新）。
//   App.vue setup 顶层已同步预取 wsList（mount 前），首次渲染即读到 5 项；此处用 computed
//   包装保持模板引用一致（后续若数据变化也走同一链路）。
const wsItems = computed(() => state.wsList)
import SvgIcon from './SvgIcon.vue'
import ContextMenu from './ContextMenu.vue'

// ── 当前工作区的文件夹列表 ──
// 工作区运行中 agent 计数（从全局 state.runningByWorkspace 读取）
// 用于在工作区列表显示脉冲点 + 数字，提示该工作区有 agent 正在并行工作
function wsRunningCount(wsPath) {
  return state.runningByWorkspace[wsPath] || 0
}

const currentFolders = computed(() => {
  if (!state.workspaceRoot) return []
  // 从 wsList 中找到当前工作区
  const cur = state.wsList.find(w => w.path === state.workspaceRoot)
  if (cur && cur.folders && cur.folders.length > 0) {
    // 合并 wsList 的 folders 与 workspaceFolders（并集）——wsList 的
    // workspaceFolderLists 可能因历史原因漏掉某些文件夹（如 goskia），
    // workspaceFolders（/api/workspace 返回的完整列表）兜底补齐，保证
    // 与浏览器 fallback 路径行为一致。
    const merged = [...cur.folders, ...state.workspaceFolders].filter(Boolean)
    return [...new Set(merged)]
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

  // 对话消息已由后端持久化，前端不再缓存到 localStorage
  saveCurrentConversations()
  try {
    const folders = ws.folders || []
    const res = await api.apiPost('/workspace', {
      action: 'switch', root: ws.path,
      folders: folders.filter(f => f !== ws.path),
    })
    state.workspaceRoot = ws.path
    state.workspaceFolders = folders.length > 0 ? [...folders] : [ws.path]
    // 同步 settings 中的 workspaceFolders，防止设置对话框保存时覆盖
    state.settings.workspaceFolders = [...state.workspaceFolders]
    state.workspaceName = ws.name || ws.path.split('\\').filter(Boolean).pop() || ws.path
    document.title = 'PairCode IDE - ' + state.workspaceName

    // 清空编辑器
    state.openFiles = []
    state.activeFile = ''
    state.fileContents = {}
    state.fileSavedContent = {}
    state.fileDirty = {}

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
  // 对话消息由后端持久化到磁盘，前端不再缓存
}

async function loadConversationsForWorkspace(path) {
  // 从后端 API 拉取（后端已持久化到磁盘）
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
        await saveWsList()
      }
      break
    }
    case 'delete':
      (async () => {
        const result = await window.$confirmWithCheckbox(
          `确认删除工作区 "${ws.name}"？`,
          '删除工作区',
          '同时删除该工作区的对话历史、快照等文件 (.pair目录)'
        )
        if (!result || !result.confirmed) return
        try {
          await api.apiPost('/workspace', { action: 'delete', root: ws.path, deleteFiles: result.checked })
        } catch (e) { console.warn('删除工作区后端失败:', e) }
        state.wsList = state.wsList.filter(w => w.path !== ws.path)
        await saveWsList()
        if (state.workspaceRoot === ws.path) {
          state.workspaceRoot = state.wsList[0]?.path || ''
          state.workspaceName = state.wsList[0]?.name || ''
          if (state.workspaceRoot) await switchToWorkspace(state.wsList[0])
        }
      })()
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
      await saveWsList()
    }
  } catch (err) { wsError.value = err.message }
}

async function saveWsList() {
  // 同步工作区列表到后端 settings.recentProjects
  try {
    const resp = await api.apiGet('/settings')
    const settings = resp.settings || resp
    settings.recentProjects = (state.wsList || []).slice(0, 20).map(w => w.path).filter(Boolean)
    await api.apiPut('/settings', settings)
  } catch (e) {


}
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

function closeBrowse() {
  browseVisible.value = false
  browseError.value = ''
}

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
    // 同步 settings 中的 workspaceFolders，防止设置对话框保存时覆盖
    state.settings.workspaceFolders = [...state.workspaceFolders]
    await saveWsList()
  } catch (e) {
    console.warn('刷新工作区失败:', e)
  }
}

let _refreshingTree = false
let _savedTreeScrollTop = 0
async function refreshAll() {
  if (_refreshingTree) return // 防止 re-entry 循环
  _refreshingTree = true

  // ★ 保存滚动位置
  const container = document.querySelector('.project-section')
  if (container) _savedTreeScrollTop = container.scrollTop

  try {
    const health = await api.apiGet('/health')
    state.workspaceFolders = health.folders || []
    state.workspaceRoot = health.workspace || state.workspaceRoot
    for (const ws of state.wsList) {
      if (ws.path === state.workspaceRoot) {
        ws.folders = [...state.workspaceFolders]
      }
    }
    // 同步 settings 中的 workspaceFolders，防止设置对话框保存时覆盖
    state.settings.workspaceFolders = [...state.workspaceFolders]
    await saveWsList()
    window.dispatchEvent(new CustomEvent('refresh-tree'))
  } catch (e) {
    console.warn('刷新全部失败:', e)
  } finally {
    _refreshingTree = false
    // ★ 恢复滚动位置
    if (_savedTreeScrollTop > 0) {
      nextTick(() => {
        const c = document.querySelector('.project-section')
        if (c) c.scrollTop = _savedTreeScrollTop
      })
    }
  }
}

function openFile(path) {
  if (!state.openFiles.includes(path)) state.openFiles.push(path)
  state.activeFile = path
  loadFileContent(path)
}

async function loadFileContent(path) {
  // ★ 有未保存编辑时保留缓存，不覆盖用户正在编辑的内容
  if (state.fileDirty[path]) return
  // ★ 清除缓存，强制从后端重新读取最新内容
  delete state.fileContents[path]
  delete state.fileSavedContent[path]
  delete state.fileDirty[path]
  // 图片和已知二进制文件不加载文本内容（由 ImageViewer/HexViewer 自行处理）
  const ext = (path.split('.').pop() || '').toLowerCase()
  const imgExts = ['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'bmp', 'ico']
  const knownBinExts = ['exe', 'dll', 'so', 'dylib', 'zip', 'tar', 'gz', 'rar',
    '7z', 'pdf', 'ttf', 'otf', 'woff', 'woff2', 'eot', 'ico', 'icns']
  if (imgExts.includes(ext) || knownBinExts.includes(ext)) {
    state.fileContents[path] = '' // 空占位
    return
  }
  try {
    const data = await api.apiGet('/fs/read', { path })
    // ★ 标准化 CRLF→LF，与 CodeMirror 内部格式一致
    const normalized = (data.content || '').replace(/\r\n/g, '\n')
    state.fileSavedContent[path] = normalized
    state.fileContents[path] = normalized
    state.fileDirty[path] = false
  } catch (err) {
    state.fileContents[path] = `// 错误: ${err.message}`
  }
}

// ── 工具集（卷帘 section：与文件树同区；工作区工具集 = .pair/toolsets/*.json） ──
const tsAddSearch = ref('')
const tsMsg = ref('')
const tsMsgErr = ref(false)
const builtinInfo = ref(null)   // GET /api/plugins/builtin：{groups, joined, manualTools, toolTotal, enabledTotal}
const tsOpen = ref(true) // 卷帘默认展开
try {
  const saved = localStorage.getItem('paircode-ts-open')
  if (saved !== null) tsOpen.value = saved === '1'
} catch {}

// 已加入数（仅 source=builtin 且组名在 joined 中；_manual 手动工具）——
// ★ source 必须校验：剩余派生组可能与已加入组同名（如 system），
//   只看组名会把未加入组误判为已加入（历史 bug：未在数量不对）
const joinedToolCount = computed(() => {
  let n = 0
  const joined = new Set(builtinInfo.value?.joined || [])
  for (const g of builtinInfo.value?.groups || []) {
    if (g.source === 'builtin' && joined.has(g.name)) n += (g.tools || []).length
  }
  return n + (builtinInfo.value?.manualTools || []).length
})

// 已加入分组（工作区工具集内容展示）：source=builtin 的 joined 组 + _manual 手动工具
const joinedGroups = computed(() => {
  const joined = new Set(builtinInfo.value?.joined || [])
  const bg = (builtinInfo.value?.groups || []).filter(g => g.source === 'builtin' && joined.has(g.name))
  const pg = (builtinInfo.value?.plugins || [])
    .map(g => ({ ...g, tools: (g.tools || []).filter(t => t.enabled) }))
    .filter(g => (g.tools || []).length > 0)
  return [...bg, ...pg]
})
const manualToolNames = computed(() => builtinInfo.value?.manualTools || [])
const manualToolObjs = computed(() => manualToolNames.value.map(n => ({ name: n, desc: '手动加入的工具' })))

// 穿梭框（未加入 ↔ 已加入 批量管理）
const tsTransferOpen = ref(false)
function openTransfer() { tsTransferOpen.value = true }
function onTransferChanged() {
  loadBuiltin()
}
// ★ 工作区切换时重载工具集（管理弹窗按当前工作区隔离展示）
watch(() => state.workspaceRoot, () => { loadBuiltin() })

// 已捞入工作区工具集的工具名集合（joined 组工具 + 插件已启用工具 + _manual 手动条目工具）
const joinedTools = computed(() => {
  const set = {}
  const joined = new Set(builtinInfo.value?.joined || [])
  for (const g of builtinInfo.value?.groups || []) {
    if (g.source === 'builtin' && joined.has(g.name)) for (const t of g.tools) set[t.name] = true
  }
  for (const g of builtinInfo.value?.plugins || []) {
    for (const t of (g.tools || [])) if (t.enabled) set[t.name] = true
  }
  for (const tn of builtinInfo.value?.manualTools || []) set[tn] = true
  return set
})
// 添加工具面板的组列表（全部内置分组 + 插件分组）
const builtinGroups = computed(() => builtinInfo.value?.groups || [])
const pluginGroups = computed(() => builtinInfo.value?.plugins || [])

// 搜索过滤（工具名模糊匹配）
function filterTools(tools) {
  const q = tsAddSearch.value.trim().toLowerCase()
  if (!q) return tools
  return tools.filter(t => t.name.toLowerCase().includes(q))
}

// 加载工作区工具集（builtin）信息（池子）
async function loadBuiltin() {
  try {
    builtinInfo.value = await api.builtinPlugins(undefined, state.workspaceRoot)
  } catch (e) {
    console.warn('[toolset] 内置工具包加载失败', e)
  }
}

// 添加/移除工具（持久化：POST /api/plugins/builtin {tool, enabled} → 固化 builtin.json _manual 条目）
async function toggleToolsetTool(t, g) {
  try {
    if (g && g.source === 'plugin') {
      if (!joinedTools.value[t.name]) {
        const res = await api.toolsetEdit({ name: 'default', action: 'add_plugin', plugin_name: g.name, tools: t.name, workspaceRoot: state.workspaceRoot })
        tsMsg.value = res?.message || '已加入 ' + t.name
      } else {
        const res = await api.toolsetEdit({ name: 'default', action: 'rm_tool', plugin_name: g.name, tool: t.name, workspaceRoot: state.workspaceRoot })
        tsMsg.value = res?.message || '已移出 ' + t.name
      }
    } else {
      const enabled = !joinedTools.value[t.name]
      const res = await api.builtinPlugins({ tool: t.name, enabled }, state.workspaceRoot)
      tsMsg.value = res?.message || (enabled ? '已添加' : '已移除') + ' ' + t.name
    }
    tsMsgErr.value = false
    await loadBuiltin()
  } catch (err) {
    tsMsgErr.value = true
    tsMsg.value = '操作失败: ' + (err.message || err)
  }
}

// 展开/收起工具集详情（插件 + 工具数）
function toggleTs() {
  tsOpen.value = !tsOpen.value
  try { localStorage.setItem('paircode-ts-open', tsOpen.value ? '1' : '0') } catch {}
}

// ── 生命周期 ──
onMounted(() => {
  window.addEventListener('refresh-tree', refreshAll)
  window.addEventListener('refresh-workspace', refreshCurrentWs)
  loadBuiltin()
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
/* 工作区运行中 agent 指示器：脉冲点 + 计数 */
.ws-running-badge {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 1px 6px; border-radius: 8px;
  background: rgba(78, 204, 163, 0.15);
  font-size: 10px; color: #4ecca3;
}
.ws-running-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: #4ecca3;
  animation: ws-pulse 1.4s ease-in-out infinite;
}
.ws-running-num { font-weight: 600; font-variant-numeric: tabular-nums; }
@keyframes ws-pulse {
  0%, 100% { opacity: 1; transform: scale(1); box-shadow: 0 0 0 0 rgba(78,204,163,0.6); }
  50% { opacity: 0.6; transform: scale(1.2); box-shadow: 0 0 0 4px rgba(78,204,163,0); }
}
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

/* ── 工具集（卷帘 section：与文件树同区） ── */
.ts-divider { border-top: 1px solid var(--border-color); flex-shrink: 0; }
.ts-header {
  display: flex; align-items: center; gap: 5px;
  padding: 5px 8px; cursor: pointer; user-select: none;
  transition: background .12s;
}
.ts-header:hover { background: var(--bg-hover); }
.ts-header.open { background: var(--bg-hover); }
.ts-header-icon { color: var(--text-muted); flex-shrink: 0; transition: color .12s; }
.ts-header.open .ts-header-icon { color: var(--accent); }
.ts-label { padding: 0; text-transform: none; letter-spacing: 0; font-size: 11px; }
.ts-spacer { flex: 1; }
.ts-chevron { transition: transform .15s; color: var(--text-muted); flex-shrink: 0; }
.ts-chevron.open { transform: rotate(90deg); }
.ts-body { padding: 0 8px 8px; display: flex; flex-direction: column; gap: 6px; }
.ts-build {
  border: 1px solid var(--border-color); border-radius: 6px; padding: 6px;
  background: var(--bg-tertiary); display: flex; flex-direction: column; gap: 5px;
}
.ts-build-head { display: flex; align-items: center; justify-content: space-between; gap: 6px; }
.ts-build-title {
  font-size: 10px; font-weight: 700; color: var(--accent-light);
  letter-spacing: .4px; display: flex; align-items: center; gap: 5px;
}
.ts-build-title::before {
  content: ''; width: 3px; height: 10px; border-radius: 2px; background: var(--accent);
}
.ts-build-head .ts-btn.mini {
  border-color: var(--accent); color: var(--accent-light); background: var(--accent-bg);
}
.ts-build-head .ts-btn.mini:hover {
  background: color-mix(in srgb, var(--accent) 22%, transparent); color: var(--accent-light);
}
.ts-input {
  background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);
  border-radius: 4px; padding: 3px 8px; font-size: 11px; width: 100%; box-sizing: border-box;
  transition: border-color .12s, box-shadow .12s;
}
.ts-input:focus { border-color: var(--accent); outline: none; box-shadow: 0 0 0 2px var(--focus-ring); }
.ts-input::placeholder { color: var(--text-muted); }
.ts-build-foot { display: flex; align-items: center; gap: 6px; }
.ts-add-list {
  max-height: 240px; overflow-y: auto; display: flex; flex-direction: column; gap: 4px;
  border: 1px solid var(--border-color); border-radius: 6px; padding: 4px; background: var(--bg-secondary);
}
.ts-add-group { display: flex; flex-direction: column; gap: 1px; }
.ts-add-group-title {
  display: flex; align-items: center; gap: 5px;
  font-size: 10px; font-weight: 600; color: var(--accent-light);
  padding: 3px 4px;
  border-left: 2px solid var(--accent);
  border-bottom: 1px dashed var(--border-color);
  margin-bottom: 1px;
}
.ts-add-tool {
  display: flex; align-items: center; justify-content: space-between; gap: 6px;
  padding: 2px 4px 2px 10px; border-radius: 4px; transition: background .1s;
}
.ts-add-tool:hover { background: var(--bg-hover); }
.ts-add-tool-name {
  font-size: 11px; color: var(--text-primary); font-family: var(--font-code);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.ts-add-tool .ts-btn.mini { opacity: .55; }
.ts-add-tool:hover .ts-btn.mini { opacity: 1; }
.ts-btn.mini {
  font-size: 10px; padding: 1px 8px; border: 1px solid var(--border-color);
  border-radius: 10px; background: none; color: var(--text-secondary);
  cursor: pointer; flex-shrink: 0; transition: all .12s;
}
.ts-btn.mini:hover { background: var(--bg-hover); color: var(--accent); }
.ts-btn.mini.added { color: var(--accent); border-color: var(--accent); background: var(--accent-bg); }
.ts-check { display: flex; align-items: center; gap: 3px; font-size: 10px; color: var(--text-secondary); white-space: nowrap; }
.ts-btn {
  background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-secondary);
  border-radius: 4px; padding: 2px 8px; font-size: 10px; cursor: pointer; transition: all .12s;
}
.ts-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.ts-btn.primary { border-color: var(--accent); color: var(--accent-light); }
.ts-btn.danger { border-color: rgba(224,108,117,.5); color: #e06c75; }
.ts-btn.danger:hover { background: rgba(224,108,117,.12); color: #e06c75; }
.ts-remove-btn { flex-shrink: 0; display: flex; align-items: center; gap: 2px; padding: 1px 7px; font-size: 10px; }
.ts-btn:disabled { opacity: .5; cursor: not-allowed; }
.ts-msg { font-size: 10px; color: var(--accent-light); word-break: break-all; }
.ts-msg.err { color: #e06c75; }
.ts-list { display: flex; flex-direction: column; gap: 4px; }
.ts-empty { padding: 10px 4px; text-align: center; color: var(--text-muted); font-size: 11px; }


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
