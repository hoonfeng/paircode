<template>
  <div class="file-tree-item" :style="{ paddingLeft: depth * 16 + 'px' }">
    <div class="item-row"
         :class="{ 'drag-over': dragOver }"
         :draggable="!item.isDir"
         @click="handleClick"
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
      <FileTreeItem v-for="child in children" :key="child.name"
                    :item="child" :parentPath="fullPath" :depth="depth + 1"
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
import { ref, computed, nextTick } from 'vue'
import { state } from '../main.js'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'
import ContextMenu from './ContextMenu.vue'

const props = defineProps({
  item: { type: Object, required: true },
  parentPath: { type: String, default: '' },
  depth: { type: Number, default: 0 },
  defaultExpanded: { type: Boolean, default: false },
})

const emit = defineEmits(['fileClick'])
const expanded = ref(props.defaultExpanded && props.item.isDir)
const children = ref(props.item.children || [])
const loaded = ref(props.item.loaded || false)
const dragOver = ref(false)

// ── 重命名状态 ──
const renaming = ref(false)
const renameValue = ref('')
const renameInputRef = ref(null)

// ── 右键菜单 ──
const contextMenuRef = ref(null)

const fullPath = computed(() => {
  if (!props.parentPath) return props.item.path || props.item.name
  return props.parentPath + '\\' + props.item.name
})

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
const handleClick = async () => {
  if (props.item.isDir) {
    expanded.value = !expanded.value
    if (expanded.value && !loaded.value) {
      try {
        const entries = await api.apiGet('/fs/list', { path: fullPath.value })
        children.value = entries || []
        loaded.value = true
      } catch {}
    }
  } else {
    emit('fileClick', fullPath.value)
  }
}

// ── ===== 右键菜单（匹配 GUI ctxmenu.go 实现）===== ──
async function showContextMenu(e) {
  await nextTick()
  const isDir = props.item.isDir
  const path = fullPath.value
  const name = props.item.name

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
      { label: '添加到工作区', action: 'add-to-workspace' },
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
      { label: '删除', action: 'delete' },
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
    title: name,
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
    case 'delete': await deleteItem(); break

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
    case 'add-to-workspace': await addToWorkspace(path); break

    // AI 操作（通过对话发送命令）

  }
}

// ── 辅助函数 ──

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
    await api.apiPost('/fs/write', { path: fullPath.value + '\\' + name, content: '' })
    await reloadChildren()
  } catch (err) { window.$toast('创建文件失败: ' + err.message, 'error') }
}

// ── 新建文件夹 ──
async function createNewFolder() {
  const name = await window.$prompt('输入文件夹名:', '', '新建文件夹')
  if (!name) return
  try {
    await api.apiPost('/fs/mkdir', { path: fullPath.value + '\\' + name })
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
  const to = props.parentPath + '\\' + newName
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

// ── 删除 ──
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

// ── 复制路径 ──
function copyPath(path) { navigator.clipboard.writeText(path).catch(() => {}) }
function copyRelPath(path) {
  const root = state.workspaceRoot || ''
  if (root && path.startsWith(root)) {
    const rel = path.slice(root.length).replace(/^\\/, '')
    navigator.clipboard.writeText(rel).catch(() => {})
  } else {
    navigator.clipboard.writeText(path).catch(() => {})
  }
}

// ── 在终端中打开 ──
function openInTerminal(path, isDir) {
  const dir = isDir ? path : (props.parentPath || path.substring(0, path.lastIndexOf('\\')))
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

// ── 添加到对话（发送文件内容 + 路径引用）──
async function addToChat(path, name, isDir) {
  try {
    let content = ''
    if (isDir) {
      content = '📁 目录引用: `' + path + '`\n（请使用 list_files 查看目录内容）'
    } else {
      // 先尝试从缓存读取文件内容
      let fileContent = state.fileContents[path]
      if (fileContent === undefined || fileContent === null) {
        try {
          const data = await api.apiGet('/fs/read', { path })
          fileContent = data.content || ''
        } catch {
          fileContent = '（无法读取文件内容）'
        }
      }
      const MAX_CHARS = 40000
      if (fileContent.length > MAX_CHARS) {
        fileContent = fileContent.slice(0, MAX_CHARS) + '\n…（文件过长，已截断）'
      }
      content = '📎 **`' + path + '`**\n\n```\n' + fileContent + '\n```'
    }
    window.dispatchEvent(new CustomEvent('add-to-chat', {
      detail: { type: 'file', path, filename: name, content }
    }))
    state.rightPanelVisible = true
  } catch (err) { window.$toast('添加失败: ' + (err.message || err), 'error') }
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
const dragSourcePath = ref('')
const onDragStart = (e) => {
  if (props.item.isDir) { e.preventDefault(); return }
  dragSourcePath.value = fullPath.value
  e.dataTransfer.setData('text/plain', fullPath.value)
  e.dataTransfer.effectAllowed = 'move'
}
const onDragOver = (e) => {
  if (props.item.isDir) dragOver.value = true
}
const onDragLeave = () => { dragOver.value = false }
const onDrop = async (e) => {
  dragOver.value = false
  if (!props.item.isDir) return
  const srcPath = e.dataTransfer.getData('text/plain')
  if (!srcPath || srcPath === fullPath.value || srcPath.startsWith(fullPath.value + '\\')) return
  const srcName = srcPath.split('\\').pop()
  try {
    await api.apiPost('/fs/rename', { from: srcPath, to: fullPath.value + '\\' + srcName })
    window.dispatchEvent(new CustomEvent('refresh-tree'))
  } catch (err) { window.$toast('移动失败: ' + (err.message || err), 'error') }
}
</script>

<style scoped>
.file-tree-item { user-select: none; }
.item-row {
  display: flex; align-items: center; gap: 3px;
  padding: 2px 4px; cursor: pointer; font-size: 13px; white-space: nowrap;
}
.item-row:hover { background: var(--bg-hover); }
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
