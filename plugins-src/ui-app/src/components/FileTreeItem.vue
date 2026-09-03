<template>
  <div class="file-tree-item">
    <div class="item-row" :style="{ paddingLeft: depth * 16 + 'px' }"
         :class="{ 'drag-over': dragOver, 'selected': isSelected }"
         draggable="true"
         @click="handleClick($event)"
         @contextmenu.prevent="showContextMenu"
         @dragstart="onDragStart"
         @dragover.prevent="onDragOver"
         @dragleave="onDragLeave"
         @drop.prevent="onDrop">
      <span v-if="item.isDir" class="chevron-wrap">
        <SvgIcon name="chevron-right" :size="10" class="chevron" :class="{ expanded }" />
      </span>
      <span v-else class="chevron-placeholder"></span>
      <SvgIcon :name="fileIcon" :size="14" />
      <span class="item-name" :class="{ active: state.activeFile === fullPath }">{{ item.name }}</span>
    </div>
    <div v-if="expanded && item.isDir && children.length > 0">
<FileTreeItem v-for="(child, ci) in children" :key="fullPath + pathSep(fullPath) + child.name + '_' + ci"
              :item="child" :parentPath="fullPath" :depth="depth + 1"
              :siblings="children" :siblingIndex="ci"
              @file-click="(p) => emit('fileClick', p)" />
    </div>
    <!-- 重命名输入框 -->
    <div v-if="renaming" class="rename-input" :style="{ paddingLeft: (depth * 16 + 28) + 'px' }">
      <input ref="renameInputRef" v-model="renameValue"
             class="rename-field"
             @keyup.enter="confirmRename"
             @keyup.escape="cancelRename"
             @blur="confirmRename" />
    </div>
    <!-- 右键菜单 -->
    <ContextMenu ref="contextMenuRef" />
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'
import { state, layout } from '../ui-state.js'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'
import ContextMenu from './ContextMenu.vue'

const props = defineProps({
  item: { type: Object, required: true },
  parentPath: { type: String, default: '' },
  depth: { type: Number, default: 0 },
  defaultExpanded: { type: Boolean, default: false },
  siblings: { type: Array, default: () => [] },  // 兄弟节点列表（用于 Shift 范围选择）
  siblingIndex: { type: Number, default: 0 },     // 在 siblings 中的索引
})

const emit = defineEmits(['fileClick'])

// ★ 路径分隔符探测（2026-09-09 跨平台）：含正斜杠且无反斜杠 = Unix/mac（/），
//   否则默认 Windows（\）。目录拼接/上级目录推算等操作按此选择分隔符。
function pathSep(p) {
  if (typeof p !== 'string' || !p) return '\\'
  return p.includes('/') && !p.includes('\\') ? '/' : '\\'
}

const childFullPath = computed(() => {
  if (!props.parentPath) return props.item.path || props.item.name
  return props.parentPath + pathSep(props.parentPath) + props.item.name
})
const fullPath = childFullPath

const expanded = ref(
  state.expandedDirs[fullPath.value] ?? (props.defaultExpanded && props.item.isDir)
)
const children = ref(props.item.children || [])
const loaded = ref(props.item.loaded || false)
const dragOver = ref(false)

// ── 重命名状态 ──
const renaming = ref(false)
const renameValue = ref('')
const renameInputRef = ref(null)

// ── 右键菜单 ──
const contextMenuRef = ref(null)

// ── 自动展开（props.defaultExpanded=true 时自动加载子目录）──
if (props.defaultExpanded && props.item.isDir && !props.item.children) {
  api.apiGet('/fs/list', { path: fullPath.value }).then(entries => {
    children.value = entries || []
    loaded.value = true
  }).catch(() => {})
}

const parentDir = computed(() => {
  return props.parentPath || ''
})

const fileIcon = computed(() => {
  if (props.item.isDir) return expanded.value ? 'folder-open' : 'folder'
  const ext = (props.item.name || '').split('.').pop().toLowerCase()
  const iconMap = {
    js: 'file-js', jsx: 'file-js', ts: 'file-ts', tsx: 'file-ts',
    go: 'file-go', py: 'file-py', java: 'file-java',
    html: 'file-html', htm: 'file-html',
    css: 'file-css', scss: 'file-css', less: 'file-css',
    json: 'file-json', yaml: 'file-text', yml: 'file-text', toml: 'file-text',
    md: 'file-md', mdx: 'file-md',
    vue: 'file-vue', svelte: 'file-code',
    rs: 'file-code', rb: 'file-code', php: 'file-code', c: 'file-code',
    cpp: 'file-code', h: 'file-code', hpp: 'file-code',
    swift: 'file-code', kt: 'file-code', dart: 'file-code',
    xml: 'file-code', svg: 'file-code',
    gitignore: 'file-text', env: 'file-text', editorconfig: 'file-text',
    mod: 'file-text', sum: 'file-text',
    png: 'file', jpg: 'file', jpeg: 'file', gif: 'file', ico: 'file',
    woff: 'file', woff2: 'file', ttf: 'file', eot: 'file',
    zip: 'file', tar: 'file', gz: 'file', rar: 'file',
    pdf: 'file', doc: 'file', docx: 'file', xls: 'file', xlsx: 'file',
  }
  return iconMap[ext] || 'file'
})

// ── 点击展开/打开 ──
// ── 是否被选中（多选状态）──
const isSelected = computed(() => state.selectedFilePaths.includes(fullPath.value))

const handleClick = async (e) => {
  if (props.item.isDir) {
    // 目录：展开/折叠
    expanded.value = !expanded.value
    state.expandedDirs[fullPath.value] = expanded.value
    if (expanded.value && !loaded.value) {
      try {
        const entries = await api.apiGet('/fs/list', { path: fullPath.value })
        children.value = entries || []
        loaded.value = true
      } catch {}
    }
    return
  }

  // ── 文件：多选逻辑 ──
  if (e.ctrlKey || e.metaKey) {
    // Ctrl+点击：切换选择（追加/移除），更新锚点
    const idx = state.selectedFilePaths.indexOf(fullPath.value)
    if (idx >= 0) {
      state.selectedFilePaths.splice(idx, 1)
    } else {
      state.selectedFilePaths.push(fullPath.value)
    }
    state.lastClickedFilePath = fullPath.value
  } else if (e.shiftKey && state.lastClickedFilePath) {
    // Shift+点击：基于兄弟节点列表计算范围选择
    const sibs = props.siblings
    if (sibs && sibs.length > 0) {
      // 构建兄弟节点的路径列表
      const sibPaths = sibs.map(s => {
        if (typeof s === 'string') return s
        return s.path || (props.parentPath ? props.parentPath + pathSep(props.parentPath) + s.name : s.name)
      })
      const anchorIdx = sibPaths.indexOf(state.lastClickedFilePath)
      const curIdx = sibPaths.indexOf(fullPath.value)
      if (anchorIdx >= 0 && curIdx >= 0) {
        // ★ 清除旧选择，重新设定范围
        state.selectedFilePaths.length = 0
        const start = Math.min(anchorIdx, curIdx)
        const end = Math.max(anchorIdx, curIdx)
        for (let i = start; i <= end; i++) {
          const sp = sibPaths[i]
          if (sp && sp !== fullPath.value) {
            state.selectedFilePaths.push(sp)
          }
        }
        state.selectedFilePaths.push(fullPath.value)
        // 不更新 lastClickedFilePath（保持锚点不变用于连续 Shift）
      } else {
        // 锚点不在同组兄弟中，只选当前
        state.selectedFilePaths.length = 0
        state.selectedFilePaths.push(fullPath.value)
        state.lastClickedFilePath = fullPath.value
      }
    } else {
      // 无兄弟列表，只选当前
      state.selectedFilePaths.length = 0
      state.selectedFilePaths.push(fullPath.value)
      state.lastClickedFilePath = fullPath.value
    }
  } else {
    // 普通点击：清除多选，只选当前，打开文件
    state.selectedFilePaths.length = 0
    state.selectedFilePaths.push(fullPath.value)
    state.lastClickedFilePath = fullPath.value
    emit('fileClick', fullPath.value)
  }
}

// ── ===== 右键菜单（匹配 GUI ctxmenu.go 实现）===== ──
async function showContextMenu(e) {
  await nextTick()
  const isDir = props.item.isDir
  const path = fullPath.value
  const name = props.item.name

  // ── 多选检测：当前右键项在选中列表中且选中文件 ≥ 2 ──
  const isMultiSelected = !isDir && state.selectedFilePaths.length > 1 && state.selectedFilePaths.includes(path)
  const selCount = isMultiSelected ? state.selectedFilePaths.length : 0

  let menuItems = []

  if (isDir) {
    // ── 目录右键菜单（匹配 GUI dirNodeMenu）──
    menuItems = [
      { label: '展开/折叠', action: 'toggle-expand' },
      { separator: true },
      { label: '新建文件', icon: 'file-plus', action: 'new-file' },
      { label: '新建文件夹', icon: 'folder-plus', action: 'new-folder' },
      { separator: true },
      { label: '添加到对话', icon: 'message-square', action: 'add-to-chat' },
      { separator: true },
      { label: '剪切', action: 'cut' },
      { label: '复制', action: 'copy' },
      { label: '粘贴', action: 'paste' },
      { separator: true },
      { label: '重命名', shortcut: 'F2', action: 'rename' },
      { label: '删除', action: 'delete' },
      { separator: true },
      { label: '复制路径', icon: 'copy', action: 'copy-path' },
      { label: '复制相对路径', action: 'copy-rel-path' },
      { separator: true },
      { label: '在终端中打开', icon: 'terminal', action: 'open-terminal' },
      { label: '在资源管理器中显示', action: 'show-in-explorer' },
      { label: '从工作区移除', action: 'remove-from-workspace' },
    ]
  } else {
    // ── 文件右键菜单（匹配 GUI fileNodeMenu）──
    menuItems = [
      { label: '打开', action: 'open' },
      { label: '打开到侧边', action: 'open-side' },
      { separator: true },
      { label: '添加到对话', icon: 'message-square', action: 'add-to-chat' },
      { separator: true },
      { label: '剪切', action: 'cut' },
      { label: '复制', action: 'copy' },
      { label: '粘贴', action: 'paste' },
      { separator: true },
      { label: '重命名', shortcut: 'F2', action: 'rename' },
      { label: isMultiSelected ? `删除选中的 ${selCount} 个文件` : '删除', action: 'delete' },
      { separator: true },
      { label: '复制路径', icon: 'copy', action: 'copy-path' },
      { label: '复制相对路径', action: 'copy-rel-path' },
      { label: '复制文件名', action: 'copy-filename' },
      { separator: true },
      { label: '在终端中打开', icon: 'terminal', action: 'open-terminal' },
      { label: '在资源管理器中显示', action: 'show-in-explorer' },
    ]
  }

  const result = await contextMenuRef.value.show({
    x: e.clientX, y: e.clientY,
    title: isMultiSelected ? `已选中 ${selCount} 个文件` : name,
    items: menuItems,
  })

  if (!result) return

  // ── 执行菜单操作 ──
  switch (result) {
    // 基本操作
    case 'open': openFile(path); break
    case 'open-side': openFile(path); break
    case 'toggle-expand': handleClick(); break

    // 新建
    case 'new-file': await createNewFile(); break
    case 'new-folder': await createNewFolder(); break

    // 剪贴板
    case 'cut': copyPath(path); break
    case 'copy': copyPath(path); break
    case 'paste': break

    // 文件操作
    case 'rename': startRename(); break
    case 'delete': { if (isMultiSelected) { await deleteSelected() } else { await deleteItem() } break }

    // 路径复制
    case 'copy-path': copyPath(path); break
    case 'copy-rel-path': copyRelPath(path); break
    case 'copy-filename': navigator.clipboard.writeText(name).catch(() => {}); break

    // 终端/系统
    case 'open-terminal': openInTerminal(path, isDir); break
    case 'show-in-explorer': showInExplorer(path); break

    // 添加到对话
    case 'add-to-chat': await addToChat(path, name, isDir); break

    // 添加到工作区
    case 'remove-from-workspace': await removeFromWorkspace(path); break

    // AI 操作（通过对话发送命令）

  }
}

// ── 辅助函数 ──

function openFile(path) {
  if (!state.openFiles.includes(path)) state.openFiles.push(path)
  state.activeFile = path
  layout.openEditor(path)
  loadFileContent(path)
}

async function loadFileContent(path) {
  // ★ 有未保存编辑时保留缓存，不覆盖用户正在编辑的内容
  if (state.fileDirty[path]) return
  // ★ 清除缓存，强制从后端重新读取最新内容
  delete state.fileContents[path]
  delete state.fileSavedContent[path]
  delete state.fileDirty[path]
  try {
    const data = await api.apiGet('/fs/read', { path })
    // ★ 标准化 CRLF→LF，与 CodeMirror 内部格式一致
    const normalized = (data.content || '').replace(/\r\n/g, '\n')
    state.fileSavedContent[path] = normalized
    state.fileContents[path] = normalized
    state.fileDirty[path] = false
  } catch (err) {
    state.fileContents[path] = '// 错误: ' + err.message
  }
}

function sendAICmd(cmd) {
  window.dispatchEvent(new CustomEvent('add-to-chat', {
    detail: { content: cmd, type: 'command' }
  }))
  state.rightPanelVisible = true
}

// ── 新建文件 ──
async function createNewFile() {
  const name = await window.$prompt('输入文件名:', '', '新建文件')
  if (!name) return
  try {
    await api.apiPost('/fs/write', { path: fullPath.value + pathSep(fullPath.value) + name, content: '' })
    await reloadChildren()
  } catch (err) { window.$toast('创建文件失败: ' + err.message, 'error') }
}

// ── 新建文件夹 ──
async function createNewFolder() {
  const name = await window.$prompt('输入文件夹名:', '', '新建文件夹')
  if (!name) return
  try {
    await api.apiPost('/fs/mkdir', { path: fullPath.value + pathSep(fullPath.value) + name })
    await reloadChildren()
  } catch (err) { window.$toast('创建文件夹失败: ' + err.message, 'error') }
}

// ── 重命名 ──
function startRename() {
  renameValue.value = props.item.name
  renaming.value = true
  nextTick(() => {
    if (renameInputRef.value) { renameInputRef.value.focus(); renameInputRef.value.select() }
  })
}

async function confirmRename() {
  if (!renaming.value) return
  const newName = renameValue.value.trim()
  renaming.value = false
  if (!newName || newName === props.item.name) return
  const from = fullPath.value
  const to = props.parentPath + pathSep(props.parentPath) + newName
  try {
    await api.apiPost('/fs/rename', { from, to })
    await reloadChildren()
    // 更新编辑器中的文件路径
    if (state.activeFile === from) {
      state.activeFile = to
      const idx = state.openFiles.indexOf(from)
      if (idx !== -1) { state.openFiles[idx] = to }
      if (state.fileContents[from]) {
        state.fileContents[to] = state.fileContents[from]
        delete state.fileContents[from]
      }
      if (state.fileDirty[from]) {
        state.fileDirty[to] = state.fileDirty[from]
        delete state.fileDirty[from]
      }
    }
  } catch (err) { window.$toast('重命名失败: ' + err.message, 'error') }
}
function cancelRename() { renaming.value = false }

// ── 删除（单个）──
async function deleteItem() {
  if (!(await window.$confirm('确认删除 ' + (props.item.isDir ? '文件夹' : '文件') + ' "' + props.item.name + '" ？'))) return
  try {
    await api.apiPost('/fs/delete', { path: fullPath.value })
    if (state.activeFile === fullPath.value) {
      state.openFiles = state.openFiles.filter(f => f !== fullPath.value)
      delete state.fileContents[fullPath.value]
      delete state.fileDirty[fullPath.value]
      state.activeFile = state.openFiles[state.openFiles.length - 1] || ''
    }
    await reloadChildren()
  } catch (err) { window.$toast('删除失败: ' + err.message, 'error') }
}

// ── 批量删除选中文件 ──
async function deleteSelected() {
  const paths = [...state.selectedFilePaths]
  if (paths.length === 0) return
  if (!(await window.$confirm(`确认删除选中的 ${paths.length} 个文件？`))) return
  let ok = 0; let fail = 0
  for (const fp of paths) {
    try {
      await api.apiPost('/fs/delete', { path: fp })
      ok++
      if (state.activeFile === fp) {
        state.openFiles = state.openFiles.filter(f => f !== fp)
        delete state.fileContents[fp]
        delete state.fileDirty[fp]
      }
    } catch (err) {
      fail++
      console.warn(`[批量删除] 删除 ${fp} 失败:`, err)
    }
  }
  state.activeFile = state.openFiles[state.openFiles.length - 1] || ''
  state.selectedFilePaths.length = 0
  if (fail === 0) {
    window.$toast(`已删除 ${ok} 个文件`, 'success')
  } else {
    window.$toast(`删除完成: ${ok} 成功, ${fail} 失败`, 'error')
  }
  // 刷新父目录
  await reloadChildren()
}

// ── 复制路径 ──
function copyPath(path) { navigator.clipboard.writeText(path).catch(() => {}) }
function copyRelPath(path) {
  const root = state.workspaceRoot || ''
  if (root && path.startsWith(root)) {
    const rel = path.slice(root.length).replace(/^[\\/]/, '')
    navigator.clipboard.writeText(rel).catch(() => {})
  } else {
    navigator.clipboard.writeText(path).catch(() => {})
  }
}

// ── 在终端中打开 ──
function openInTerminal(path, isDir) {
  const dir = isDir ? path : (props.parentPath || path.substring(0, path.lastIndexOf(pathSep(path))))
  state.bottomPanelVisible = true
  state.bottomPanelTab = 'terminal'
  window.dispatchEvent(new CustomEvent('terminal-cwd', { detail: { cwd: dir } }))
}

// ── 在资源管理器中显示 ──
function showInExplorer(path) {
  // Windows: 尝试打开 explorer 选择文件/文件夹
  const cmd = `explorer /select,"${path}"`
  try { api.apiPost('/system/exec', { command: cmd }) } catch {}
}

// ── 添加到工作区 ──
async function addToWorkspace(path) {
  try {
    await api.apiPost('/workspace', { action: 'add-folder', path })
    window.dispatchEvent(new CustomEvent('refresh-tree'))
  } catch (err) { window.$toast('添加失败: ' + err.message, 'error') }
}

// ── 从工作区移除 ──
async function removeFromWorkspace(path) {
  try {
    await api.apiPost('/workspace', { action: 'remove-folder', path })
    window.dispatchEvent(new CustomEvent('refresh-tree'))
  } catch (err) { window.$toast('移除失败: ' + err.message, 'error') }
}

// ── 添加到对话（仅传路径引用，不内联文件内容）──
async function addToChat(path, name, isDir) {
  if (isDir) {
    window.dispatchEvent(new CustomEvent('add-to-chat', {
      detail: { type: 'dir', path, filename: name }
    }))
  } else {
    window.dispatchEvent(new CustomEvent('add-to-chat', {
      detail: { type: 'file', path, filename: name }
    }))
  }
  state.rightPanelVisible = true
}

// ── 刷新子节点 ──
async function reloadChildren() {
  try {
    const entries = await api.apiGet('/fs/list', { path: props.parentPath })
    children.value = entries || []
  } catch {}
  window.dispatchEvent(new CustomEvent('refresh-tree'))
}

// ── 拖拽事件 ──
const onDragStart = (e) => {
  // 如果有多选选中的文件，拖拽全部选中项；否则只拖当前
  let paths = []
  if (state.selectedFilePaths.length > 1 && state.selectedFilePaths.includes(fullPath.value)) {
    paths = state.selectedFilePaths
  } else {
    paths = [fullPath.value]
  }
  e.dataTransfer.setData('text/plain', JSON.stringify(paths))
  e.dataTransfer.effectAllowed = e.ctrlKey ? 'copy' : 'move'
  // 设置拖拽图标（可选）
  if (e.dataTransfer.setDragImage && paths.length === 1) {
    // 使用当前元素作为拖拽图标
    const el = e.currentTarget
    e.dataTransfer.setDragImage(el, 10, 10)
  }
}
const onDragOver = (e) => {
  if (props.item.isDir) {
    dragOver.value = true
    e.dataTransfer.dropEffect = e.ctrlKey ? 'copy' : 'move'
  }
}
const onDragLeave = () => { dragOver.value = false }
const onDrop = async (e) => {
  dragOver.value = false
  if (!props.item.isDir) return
  const raw = e.dataTransfer.getData('text/plain')
  if (!raw) return
  let paths = []
  try { paths = JSON.parse(raw) } catch { paths = [raw] }
  if (!Array.isArray(paths)) paths = [paths]
  if (paths.length === 0) return

  const targetDir = fullPath.value
  const isCopy = e.ctrlKey || e.shiftKey
  let successCount = 0
  let failCount = 0

  for (const srcPath of paths) {
    const sep = pathSep(targetDir)
    if (!srcPath || srcPath === targetDir || srcPath.startsWith(targetDir + sep)) continue
    const srcName = srcPath.split(/[\\/]/).pop()
    const destPath = targetDir + sep + srcName
    try {
      if (isCopy) {
        // Ctrl+拖拽 = 复制
        await api.apiPost('/fs/copy', { from: srcPath, to: destPath })
        // 复制编辑器缓存
        if (state.fileContents[srcPath]) {
          state.fileContents[destPath] = state.fileContents[srcPath]
          state.fileSavedContent[destPath] = state.fileSavedContent[srcPath]
        }
      } else {
        // 普通拖拽 = 移动
        await api.apiPost('/fs/rename', { from: srcPath, to: destPath })
        // 更新编辑器路径
        if (state.activeFile === srcPath) {
          state.activeFile = destPath
          const idx = state.openFiles.indexOf(srcPath)
          if (idx !== -1) state.openFiles[idx] = destPath
        }
        if (state.fileContents[srcPath]) {
          state.fileContents[destPath] = state.fileContents[srcPath]
          state.fileSavedContent[destPath] = state.fileSavedContent[srcPath]
          delete state.fileContents[srcPath]
          delete state.fileSavedContent[srcPath]
          delete state.fileDirty[srcPath]
        }
      }
      successCount++
    } catch (err) {
      console.warn('[拖拽] 操作失败:', srcPath, '→', destPath, err)
      failCount++
    }
  }

  if (successCount > 0) {
    window.dispatchEvent(new CustomEvent('refresh-tree'))
    window.$toast(isCopy
      ? `已复制 ${successCount} 个${failCount > 0 ? '（' + failCount + ' 个失败）' : ''}`
      : `已移动 ${successCount} 个${failCount > 0 ? '（' + failCount + ' 个失败）' : ''}`, 'success')
  } else if (failCount > 0) {
    window.$toast('拖拽操作失败: ' + failCount + ' 个错误', 'error')
  }
}

// ── 监听 refresh-tree：已展开的目录自动重新加载子节点 ──
function onRefreshTree() {
  if (expanded.value && props.item.isDir) {
    // 展开状态下保存当前展开标记
    state.expandedDirs[fullPath.value] = true
    loaded.value = false
    api.apiGet('/fs/list', { path: fullPath.value }).then(entries => {
      children.value = entries || []
      loaded.value = true
    }).catch(() => {})
  }
}
onMounted(() => {
  window.addEventListener('refresh-tree', onRefreshTree)
})
onUnmounted(() => {
  window.removeEventListener('refresh-tree', onRefreshTree)
})
</script>

<style scoped>
.file-tree-item { user-select: none; }
.item-row {
  display: flex; align-items: center; gap: 3px;
  padding: 2px 4px; cursor: pointer; font-size: 13px; white-space: nowrap;
}
.item-row:hover { background: var(--bg-hover); }
.item-row.selected { background: var(--accent-bg); outline: 1px solid var(--accent); outline-offset: -1px; }
.item-row.drag-over { background: rgba(126, 184, 218, 0.2); outline: 1px dashed var(--accent); }
.chevron-wrap { width: 12px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; }
.chevron { transition: transform .15s; color: var(--text-muted); }
.chevron.expanded { transform: rotate(90deg); }
.chevron-placeholder { width: 12px; flex-shrink: 0; }
.item-name { color: var(--text-primary); overflow: hidden; text-overflow: ellipsis; }
.item-name.active { color: var(--accent-light); }
.rename-input { padding: 2px 4px; }
.rename-field {
  width: 100%; box-sizing: border-box;
  background: var(--bg-primary); border: 1px solid var(--accent);
  color: var(--text-primary); font-family: var(--font-code); font-size: 13px; padding: 1px 4px;
  outline: none; border-radius: 2px;
}
</style>
