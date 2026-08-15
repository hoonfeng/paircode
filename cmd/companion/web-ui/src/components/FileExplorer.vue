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
      <div class="ts-header" @click="toggleTs" title="工具集（工作区内，可折叠）">
        <SvgIcon name="package" :size="12" class="ts-header-icon" />
        <span class="divider-label ts-label">工具集</span>
        <span v-if="toolsets.length" class="ts-count">{{ toolsets.length }}</span>
        <span class="ts-spacer"></span>
        <button class="ts-mini-btn" @click.stop="loadToolsets" title="刷新工具集"><SvgIcon name="refresh" :size="11" :class="{ spinning: tsRefreshing }" /></button>
        <button class="ts-mini-btn" @click.stop="tsBuildOpen = !tsBuildOpen" title="动态构建工具集"><SvgIcon name="plus" :size="12" /></button>
        <SvgIcon name="chevron-right" :size="11" class="ts-chevron" :class="{ open: tsOpen }" />
      </div>
      <div v-if="tsOpen" class="ts-body">
        <!-- 构建表单 -->
        <div v-if="tsBuildOpen" class="ts-build">
          <div class="ts-build-title">动态构建（分析项目 → 模板组合插件 → 固化 .pair/toolsets/）</div>
          <input v-model="tsForm.name" placeholder="工具集名（如 web-dev；默认 default）" class="ts-input" />
          <input v-model="tsForm.description" placeholder="用途描述（可选）" class="ts-input" />
          <input v-model="tsForm.requirement" placeholder="要求（可选）：如「Web 前端脚手架 + 接口调试」" class="ts-input" />
          <div class="ts-build-foot">
            <label class="ts-check"><input type="checkbox" v-model="tsForm.overwrite" /> 覆盖同名</label>
            <button class="ts-btn primary" :disabled="tsBuilding" @click="buildToolset">
              {{ tsBuilding ? '构建中…' : '构建并固化' }}
            </button>
          </div>
          <div v-if="tsMsg" class="ts-msg" :class="{ err: tsMsgErr }">{{ tsMsg }}</div>
        </div>
        <!-- 列表 -->
        <div v-if="toolsets.length" class="ts-list">
            <div v-for="ts in toolsets" :key="ts.name + '-' + ts.scope" class="ts-item">
              <div class="ts-item-head" @click="toggleTsDetail(ts)" :title="ts.scope === 'builtin' ? '内置工具包（点击展开分组与工具）' : '点击展开查看插件与工具'">
                <span class="ts-item-dot" :class="ts.scope === 'global' ? 'g' : ''"></span>
                <span class="ts-item-name" :title="ts.description">{{ ts.name }}</span>
                <span class="ts-item-scope" :class="ts.scope === 'builtin' ? 'b' : ''">{{ ts.scope === 'builtin' ? '内置' : (ts.scope === 'global' ? '全局' : '工作区') }}</span>
                <span class="ts-item-count">{{ ts.scope === 'builtin' ? ts.pluginCount + ' 组' : ts.pluginCount + ' 插件' }}</span>
                <SvgIcon name="chevron-right" :size="11" class="ts-chevron" :class="{ open: tsDetailOpen[tsKey(ts)] }" />
              </div>
              <div v-if="ts.description" class="ts-item-desc">{{ ts.description }}</div>
              <!-- 详情（点击展开）：内置工具包=分组+工具+开关；普通工具集=插件+工具 -->
              <div v-if="tsDetailOpen[tsKey(ts)]" class="ts-item-detail">
                <div v-if="tsDetailLoading[tsKey(ts)]" class="ts-detail-loading">加载…</div>
                <template v-else-if="tsDetail[tsKey(ts)]">
                  <!-- builtin：分组 + 工具清单 + 启用开关 -->
                  <div v-if="ts.scope === 'builtin'" class="ts-detail-groups">
                    <div v-for="g in tsDetail[tsKey(ts)].groups" :key="g.name" class="ts-detail-group">
                      <div class="ts-detail-grow">
                        <span class="ts-detail-gname" :class="{ off: !g.enabled && !g.partial }">{{ g.title }}</span>
                        <span class="ts-detail-gtools">{{ g.tools.length }} 工具<template v-if="g.partial">（部分）</template></span>
                      </div>
                      <label class="ts-switch" :title="g.enabled ? '组内工具全部对 agent 可见；点击移出（恢复默认过滤）' : '加入工作区：组内工具全部对 agent 可见'">
                        <input type="checkbox" :checked="g.enabled" @change="toggleTsGroup(ts, g)" />
                        <span class="ts-switch-track"></span>
                      </label>
                      <div v-if="g.tools.length" class="ts-detail-tools">
                        <span v-for="t in g.tools" :key="t.name" class="ts-tool-chip" :class="{ off: !t.enabled }">{{ t.name }}</span>
                      </div>
                    </div>
                  </div>
                  <!-- 普通工具集：插件清单 + 工具 -->
                  <div v-else class="ts-detail-plugins">
                    <div v-for="pl in tsDetail[tsKey(ts)].plugins" :key="pl.name" class="ts-detail-plugin">
                      <div class="ts-detail-prow">
                        <span class="ts-detail-pname">{{ pl.name }}</span>
                        <span v-if="pl.builtin" class="ts-detail-pbuiltin">内置组</span>
                      </div>
                      <div v-if="pl.purpose" class="ts-detail-ppurpose">{{ pl.purpose }}</div>
                      <div v-if="pl.tools && pl.tools.length" class="ts-detail-tools">
                        <span v-for="t in pl.tools" :key="t" class="ts-tool-chip" :class="{ off: (pl.disabledTools || []).includes(t) }">{{ t }}</span>
                      </div>
                      <div v-else class="ts-detail-muted">（插件运行时注册工具）</div>
                    </div>
                  </div>
                </template>
                <div v-else class="ts-detail-loading err">加载失败</div>
              </div>
              <div class="ts-item-actions">
                <button class="ts-btn" @click="exportToolset(ts)">导出</button>
                <button v-if="ts.scope !== 'builtin'" class="ts-btn danger" @click="removeToolset(ts)">删除</button>
              </div>
            </div>
        </div>
        <div v-else-if="!tsRefreshing" class="ts-empty">
          <span>暂无工具集。点 + 动态构建，或到市场安装插件工具集。</span>
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
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, reactive } from 'vue'
import { state } from '../main.js'
import api from '../api.js'
import FileTreeItem from './FileTreeItem.vue'

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
    const settings = await api.apiGet('/settings')
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

// ── 工具集（卷帘 section：与文件树同区，工作区 .pair/toolsets/） ──
const toolsets = ref([])
const tsRefreshing = ref(false)
const tsBuilding = ref(false)
const tsBuildOpen = ref(false)
const tsMsg = ref('')
const tsMsgErr = ref(false)
const tsForm = reactive({ name: '', description: '', requirement: '', overwrite: false })
const tsOpen = ref(true) // 卷帘默认展开
try {
  const saved = localStorage.getItem('paircode-ts-open')
  if (saved !== null) tsOpen.value = saved === '1'
} catch {}

async function loadToolsets() {
  tsRefreshing.value = true
  try {
    const list = await api.apiGet('/toolsets')
      // 全部工具集（含虚拟内置工具包 builtin：scope=builtin，点击展开查看分组+工具+开关）
      toolsets.value = Array.isArray(list) ? list : []
  } catch (e) {
    console.warn('[toolset] 加载失败', e)
  } finally {
    tsRefreshing.value = false
  }
}

async function buildToolset() {
  tsBuilding.value = true
  tsMsg.value = ''
  tsMsgErr.value = false
  try {
    const res = await api.apiPost('/toolsets/build', {
      name: tsForm.name,
      description: tsForm.description,
      requirement: tsForm.requirement,
      overwrite: tsForm.overwrite,
    })
    tsMsg.value = `已构建并固化「${res.name}」（${res.pluginCount} 个插件）`
    tsForm.name = ''
    tsForm.description = ''
    tsForm.requirement = ''
    tsForm.overwrite = false
    tsBuildOpen.value = false
    await loadToolsets()
  } catch (err) {
    tsMsgErr.value = true
    tsMsg.value = '构建失败: ' + (err.message || err)
  } finally {
    tsBuilding.value = false
  }
}

function exportToolset(ts) {
  // 下载发布 JSON（可提交 GitHub 发布市场 / toolset_import 导入）
  const a = document.createElement('a')
  a.href = `/api/toolsets/export?name=${encodeURIComponent(ts.name)}`
  a.download = ts.name + '.toolset.json'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

async function removeToolset(ts) {
  if (!window.confirm(`删除工具集「${ts.name}」（${ts.scope}）？已装载插件将卸载。`)) return
  try {
    await api.apiPost('/toolsets/remove', { name: ts.name, scope: ts.scope === 'global' ? 'global' : 'project' })
    window.$toast?.('已删除工具集 ' + ts.name, 'success')
    await loadToolsets()
  } catch (err) {
    window.$toast?.('删除失败: ' + (err.message || err), 'error')
  }
}

// 工具集详情（点击展开：builtin=分组+工具+开关；普通=插件+工具）
const tsDetail = reactive({})
const tsDetailOpen = reactive({})
const tsDetailLoading = reactive({})

function tsKey(ts) { return ts.name + '-' + ts.scope }

async function toggleTsDetail(ts) {
  const k = tsKey(ts)
  tsDetailOpen[k] = !tsDetailOpen[k]
  if (tsDetailOpen[k] && tsDetail[k] === undefined) {
    tsDetailLoading[k] = true
    try {
      tsDetail[k] = await api.apiGet('/toolsets?name=' + encodeURIComponent(ts.name))
    } catch (e) {
      tsDetail[k] = null
    } finally {
      tsDetailLoading[k] = false
    }
  }
}

// 内置分组开关（文件浏览器工具集区操作；与插件面板同源：/api/plugins/builtin）
async function toggleTsGroup(ts, g) {
  const target = !g.enabled
  try {
    const res = await api.apiPost('/plugins/builtin', { group: g.name, enabled: target })
    window.$toast?.((res && res.message) || (target ? '已加入' : '已移出') + ' ' + g.name, 'info')
  } catch (e) {
    window.$toast?.('操作失败: ' + (e.message || e), 'error')
  }
  // 刷新详情 + 列表（保持展开）
  const k = tsKey(ts)
  try { tsDetail[k] = await api.apiGet('/toolsets?name=' + encodeURIComponent(ts.name)) } catch (e) {}
  loadToolsets()
}

function toggleTs() {
  tsOpen.value = !tsOpen.value
  try { localStorage.setItem('paircode-ts-open', tsOpen.value ? '1' : '0') } catch {}
}

// ── 生命周期 ──
onMounted(() => {
  window.addEventListener('refresh-tree', refreshAll)
  window.addEventListener('refresh-workspace', refreshCurrentWs)
  loadToolsets()
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
}

.ts-item-head { cursor: pointer; }
.ts-item-scope.b { border-color: var(--accent); color: var(--accent-light); }
.ts-item-detail {
  border-top: 1px dashed var(--border-color);
  margin-top: 4px; padding-top: 4px;
  display: flex; flex-direction: column; gap: 4px;
}
.ts-detail-loading { font-size: 10px; color: var(--text-muted); padding: 2px 0; }
.ts-detail-loading.err { color: #e06c75; }
.ts-detail-groups, .ts-detail-plugins { display: flex; flex-direction: column; gap: 4px; }
.ts-detail-group, .ts-detail-plugin {
  border: 1px solid var(--border-color); border-radius: 4px; padding: 4px 6px;
  background: var(--bg-tertiary); display: flex; flex-direction: column; gap: 3px;
}
.ts-detail-grow { display: flex; align-items: baseline; gap: 6px; }
.ts-detail-gname { font-size: 10px; font-weight: 600; }
.ts-detail-gname.off { color: var(--text-muted); opacity: .6; }
.ts-detail-gtools { font-size: 9px; color: var(--text-muted); }
.ts-detail-tools { display: flex; flex-wrap: wrap; gap: 3px; }
.ts-tool-chip {
  font-size: 9px; padding: 0 5px; border-radius: 3px;
  background: var(--bg-primary); border: 1px solid var(--border-color);
  color: var(--text-secondary); font-family: var(--font-code);
  white-space: nowrap;
}
.ts-tool-chip.off { text-decoration: line-through; opacity: .45; }
.ts-detail-prow { display: flex; align-items: center; gap: 6px; }
.ts-detail-pname { font-size: 10px; font-weight: 600; word-break: break-all; }
.ts-detail-pbuiltin { font-size: 8px; color: var(--accent-light); border: 1px solid var(--accent); border-radius: 3px; padding: 0 3px; flex-shrink: 0; }
.ts-detail-ppurpose { font-size: 9px; color: var(--text-muted); }
.ts-detail-muted { font-size: 9px; color: var(--text-muted); }
.ts-switch { position: relative; display: inline-flex; align-items: center; cursor: pointer; flex-shrink: 0; align-self: flex-start; }
.ts-switch input { position: absolute; opacity: 0; width: 0; height: 0; }
.ts-switch-track {
  width: 24px; height: 13px; background: var(--border-color); border-radius: 7px;
  transition: background .15s; position: relative;
}
.ts-switch-track::after {
  content: ''; position: absolute; top: 2px; left: 2px;
  width: 9px; height: 9px; background: #fff; border-radius: 50%;
  transition: transform .15s;
}
.ts-switch input:checked + .ts-switch-track { background: var(--accent); }
.ts-switch input:checked + .ts-switch-track::after { transform: translateX(11px); }

.ts-header:hover { background: var(--bg-hover); }
.ts-header-icon { color: var(--text-muted); flex-shrink: 0; }
.ts-label { padding: 0; text-transform: none; letter-spacing: 0; font-size: 11px; }
.ts-count {
  font-size: 9px; background: var(--bg-tertiary); color: var(--text-muted);
  border-radius: 8px; padding: 0 6px; line-height: 14px;
}
.ts-spacer { flex: 1; }
.ts-mini-btn {
  background: none; border: none; cursor: pointer; color: var(--text-muted);
  padding: 1px 3px; border-radius: 3px; display: flex; align-items: center;
}
.ts-mini-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.ts-chevron { transition: transform .15s; color: var(--text-muted); flex-shrink: 0; }
.ts-chevron.open { transform: rotate(90deg); }
.ts-body { padding: 0 8px 8px; display: flex; flex-direction: column; gap: 6px; }
.ts-build { border: 1px solid var(--border-color); border-radius: 4px; padding: 6px; background: var(--bg-tertiary); display: flex; flex-direction: column; gap: 4px; }
.ts-build-title { font-size: 10px; color: var(--text-secondary); }
.ts-input {
  background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-primary);
  border-radius: 3px; padding: 3px 6px; font-size: 11px; width: 100%; box-sizing: border-box;
}
.ts-input:focus { border-color: var(--accent); outline: none; }
.ts-build-foot { display: flex; align-items: center; gap: 6px; }
.ts-check { display: flex; align-items: center; gap: 3px; font-size: 10px; color: var(--text-secondary); white-space: nowrap; }
.ts-btn {
  background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-secondary);
  border-radius: 3px; padding: 2px 8px; font-size: 10px; cursor: pointer;
}
.ts-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.ts-btn.primary { border-color: var(--accent); color: var(--accent-light); }
.ts-btn.danger { border-color: #e06c75; color: #e06c75; }
.ts-btn:disabled { opacity: .5; cursor: not-allowed; }
.ts-msg { font-size: 10px; color: var(--accent-light); word-break: break-all; }
.ts-msg.err { color: #e06c75; }
.ts-list { display: flex; flex-direction: column; gap: 4px; }
.ts-item { border: 1px solid var(--border-color); border-radius: 4px; padding: 5px 6px; background: var(--bg-primary); }
.ts-item-head { display: flex; align-items: center; gap: 5px; }
.ts-item-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--text-muted); opacity: .4; flex-shrink: 0; }
.ts-item-dot.g { background: #4caf50; opacity: 1; }
.ts-item-name { flex: 1; font-weight: 500; font-size: 11px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ts-item-scope { font-size: 9px; color: var(--text-muted); border: 1px solid var(--border-color); border-radius: 3px; padding: 0 4px; flex-shrink: 0; }
.ts-item-count { font-size: 10px; color: var(--text-muted); flex-shrink: 0; }
.ts-item-desc { font-size: 10px; color: var(--text-muted); margin-top: 2px; line-height: 1.4; }
.ts-item-actions { display: flex; gap: 4px; margin-top: 4px; }
.ts-empty { padding: 10px 4px; text-align: center; color: var(--text-muted); font-size: 11px; }
.spinning { animation: ts-spin 1s linear infinite; }
@keyframes ts-spin { to { transform: rotate(360deg); } }

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
