<template>
  <div class="editor-area">
    <!-- 标签栏 -->
    <div class="editor-tabs" v-if="state.openFiles.length > 0">
      <button v-for="file in state.openFiles" :key="file"
              :class="{ active: file === state.activeFile }"
              @click="switchTab(file)"
              @mouseup.middle="closeTab(file)"
              @contextmenu.prevent="showTabContextMenu($event, file)">
        <span class="tab-icon"><SvgIcon :name="getIcon(file)" :size="12" /></span>
        <span class="tab-name">{{ getName(file) }}</span>
        <span v-if="state.fileDirty[file]" class="dirty-mark">●</span>
        <span class="tab-close" @click.stop="closeTab(file)">×</span>
      </button>
      <span class="tab-spacer"></span>
      <button class="tb-action" @click="undoAction" title="撤消 (Ctrl+Z)" :disabled="!state.activeFile"><SvgIcon name="undo" :size="12" /></button>
      <button class="tb-action" @click="redoAction" title="重做 (Ctrl+Y)" :disabled="!state.activeFile"><SvgIcon name="redo" :size="12" /></button>
    </div>

    <!-- 欢迎页 / 编辑器 -->
    <div class="editor-body">
      <div v-if="state.openFiles.length === 0" class="welcome">
        <div class="welcome-logo">PairCode</div>
        <div class="welcome-text">Web IDE</div>
        <div class="welcome-sub">打开文件开始编辑 • Ctrl+B 切换侧栏 • Ctrl+` 切换面板</div>
        <div class="workspace-info" v-if="state.workspaceRoot">
          <div>工作区: {{ state.workspaceRoot }}</div>
        </div>
      </div>
      <div v-else-if="state.activeFile" class="editor-wrapper">
        <CodeEditor
          ref="editorRef"
          :key="state.activeFile"
          :modelValue="currentContent"
          :path="state.activeFile"
          @update:modelValue="onContentChange"
          @cursorPos="onCursorPos"
          @contextmenu="onEditorContextMenu"
          @save="saveFile"
        />
      </div>
    </div>

    <!-- 标签右键菜单 -->
    <ContextMenu ref="tabContextMenu" />
    <!-- 编辑器右键菜单（代码区右键） -->
    <ContextMenu ref="editorCtxMenu" />
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted, nextTick } from 'vue'
import { undo, redo } from '@codemirror/commands'
import { state } from '../main.js'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'
import CodeEditor from './CodeEditor.vue'
import ContextMenu from './ContextMenu.vue'

const editorRef = ref(null)
const tabContextMenu = ref(null)
const editorCtxMenu = ref(null)
let contextFile = ''

const currentContent = computed(() => state.fileContents[state.activeFile] ?? '')

const getName = (path) => path.split('\\').pop() || path.split('/').pop() || path

const getIcon = (path) => {
  const ext = (path || '').split('.').pop().toLowerCase()
  const icons = {
    js: 'file-js', ts: 'file-ts', go: 'file-go', py: 'file-py',
    vue: 'file-vue', html: 'file-html', css: 'file-css',
    json: 'file-json', md: 'file-md', xml: 'file-code',
    yaml: 'file-text', yml: 'file-text', toml: 'file-text',
    rs: 'file-code', java: 'file-java', c: 'file-code', cpp: 'file-code',
    gitignore: 'file-text', env: 'file-text', mod: 'file-text',
  }
  return icons[ext] || 'file'
}

const switchTab = (path) => { state.activeFile = path }

const closeTab = async (path) => {
  if (state.fileDirty[path]) {
    if (!(await window.$confirm(`文件 ${getName(path)} 有未保存的修改，是否关闭？`))) return
  }
  state.openFiles = state.openFiles.filter(f => f !== path)
  delete state.fileContents[path]
  delete state.fileDirty[path]
  if (state.activeFile === path) {
    state.activeFile = state.openFiles[state.openFiles.length - 1] || ''
  }
}

// ── ===== 标签右键菜单（匹配 GUI EditorTabMenu）===== ──
async function showTabContextMenu(e, file) {
  contextFile = file
  await nextTick()
  const fileCount = state.openFiles.length
  const isActive = file === state.activeFile

  const result = await tabContextMenu.value.show({
    x: e.clientX, y: e.clientY,
    title: getName(file),
    items: [
      { label: '关闭', icon: 'close', action: 'close' },
      { label: '关闭其他', action: 'close-others', disabled: fileCount <= 1 },
      { label: '关闭全部', action: 'close-all' },
      { separator: true },
      { label: '复制路径', icon: 'copy', action: 'copy-path' },
    ],
  })
  if (!result) return
  switch (result) {
    case 'close': closeTab(contextFile); break
    case 'close-others': await closeOtherTabs(contextFile); break
    case 'close-all': await closeAllTabs(); break
    case 'copy-path': navigator.clipboard.writeText(contextFile).catch(() => {}); break
  }
}

async function closeOtherTabs(file) {
  for (const f of state.openFiles) {
    if (f !== file && state.fileDirty[f]) {
      if (!(await window.$confirm(`文件 ${getName(f)} 有未保存的修改，是否关闭？`))) return
    }
  }
  state.openFiles = [file]
  state.activeFile = file
  for (const key of Object.keys(state.fileContents)) {
    if (key !== file) { delete state.fileContents[key]; delete state.fileDirty[key] }
  }
}

async function closeAllTabs() {
  for (const f of state.openFiles) {
    if (state.fileDirty[f]) {
      if (!(await window.$confirm(`文件 ${getName(f)} 有未保存的修改，是否关闭？`))) return
    }
  }
  state.openFiles = []; state.activeFile = ''
  for (const key of Object.keys(state.fileContents)) { delete state.fileContents[key]; delete state.fileDirty[key] }
}

// ── ===== 编辑器右键菜单（匹配 GUI EditorContentMenu）===== ──
async function onEditorContextMenu(ev) {
  const file = state.activeFile
  const fileName = file ? getName(file) : ''
  const path = file || ''

  if (ev.hasSelection && ev.text) {
    // ── 有选中文本：添加到对话 + 复制选中 ──
    await nextTick()
    const result = await editorCtxMenu.value.show({
      x: ev.x, y: ev.y,
      items: [
        { label: '撤销', shortcut: 'Ctrl+Z', action: 'undo' },
        { label: '重做', shortcut: 'Ctrl+Shift+Z', action: 'redo' },
        { separator: true },
        { label: '剪切', shortcut: 'Ctrl+X', action: 'cut' },
        { label: '复制', shortcut: 'Ctrl+C', action: 'copy' },
        { label: '粘贴', shortcut: 'Ctrl+V', action: 'paste' },
        { label: '全选', shortcut: 'Ctrl+A', action: 'select-all' },
        { separator: true },
        { label: 'AI: 添加到对话', icon: 'message-square', action: 'add-to-chat' },
      ],
    })
    if (!result) return
    switch (result) {
      case 'add-to-chat':
        window.dispatchEvent(new CustomEvent('add-to-chat', {
          detail: {
            type: 'selection', path, filename: fileName,
            lineStart: ev.lineStart, lineEnd: ev.lineEnd, content: ev.text,
          }
        }))
        state.rightPanelVisible = true
        break
      case 'undo': undoAction(); break
      case 'redo': redoAction(); break
      case 'cut': execCopy(ev.text); state.fileDirty[path] = true; break
      case 'copy': execCopy(ev.text); break
      case 'paste': break
      case 'select-all': break
    }
    return
  }

  // ── 无选中：完整编辑器右键菜单 ──
  await nextTick()
  const result = await editorCtxMenu.value.show({
    x: ev.x, y: ev.y,
    items: [
      { label: '撤销', shortcut: 'Ctrl+Z', action: 'undo' },
      { label: '重做', shortcut: 'Ctrl+Shift+Z', action: 'redo' },
      { separator: true },
      { label: '剪切', shortcut: 'Ctrl+X', action: 'cut' },
      { label: '复制', shortcut: 'Ctrl+C', action: 'copy' },
      { label: '粘贴', shortcut: 'Ctrl+V', action: 'paste' },
      { label: '全选', shortcut: 'Ctrl+A', action: 'select-all' },
      { separator: true },
      { label: 'AI: 添加到对话', icon: 'message-square', action: 'add-to-chat' },
      { separator: true },
      { label: '格式化文档', action: 'format' },
      { separator: true },
      { label: '复制行内容', action: 'copy-line' },
      { label: '复制文件名', action: 'copy-filename' },
      { label: '复制路径', icon: 'copy', action: 'copy-path' },
      { separator: true },
      { label: '命令面板', shortcut: 'Ctrl+P', action: 'command-palette' },
    ],
  })
  if (!result) return
  switch (result) {
    case 'save': saveFile(); break
    case 'undo': undoAction(); break
    case 'redo': redoAction(); break
    case 'cut': document.execCommand('cut'); break
    case 'copy': document.execCommand('copy'); break
    case 'paste': document.execCommand('paste'); break
    case 'select-all': document.execCommand('selectAll'); break
    case 'add-to-chat': addFileToChat(file); break
    case 'format': window.$toast('格式化功能开发中', 'info'); break
    case 'copy-line': copyCurrentLine(); break
    case 'copy-filename': navigator.clipboard.writeText(fileName).catch(() => {}); break
    case 'copy-path': navigator.clipboard.writeText(path).catch(() => {}); break
    case 'command-palette':
      state.activeActivity = 'search'
      state.sidebarVisible = true
      break
    case 'reveal':
      state.activeActivity = 'explorer'
      state.sidebarVisible = true
      break
  }
}

function execCopy(text) { navigator.clipboard.writeText(text).catch(() => {}) }

function addFileToChat(file) {
  const fileName = getName(file)
  const content = state.fileContents[file] || ''
  window.dispatchEvent(new CustomEvent('add-to-chat', {
    detail: { type: 'file', path: file, filename: fileName, content }
  }))
  state.rightPanelVisible = true
}

function copyCurrentLine() {
  const content = state.fileContents[state.activeFile] || ''
  const lines = content.split('\n')
  if (lines.length > 0) navigator.clipboard.writeText(lines[0]).catch(() => {})
}

// ── 编辑器内容变更 ──
const onContentChange = (val) => {
  state.fileContents[state.activeFile] = val
  state.fileDirty[state.activeFile] = true
}
const onCursorPos = (pos) => {
  state.cursorLine = pos.line
  state.cursorCol = pos.col
}

// ── 保存 ──
const saveFile = async () => {
  const path = state.activeFile
  if (!path) return
  try {
    await api.apiPost('/fs/write', { path, content: state.fileContents[path] })
    state.fileDirty[path] = false
  } catch (err) { window.$toast('保存失败: ' + err.message, 'error') }
}

const handleKeydown = (e) => {
  if ((e.ctrlKey || e.metaKey) && e.key === 's') {
    e.preventDefault()
    if (state.activeFile && state.fileDirty[state.activeFile]) saveFile()
  }
  if ((e.ctrlKey || e.metaKey) && e.key === 'z' && !e.shiftKey) {
    e.preventDefault(); undoAction()
  }
  if ((e.ctrlKey || e.metaKey) && e.key === 'z' && e.shiftKey) {
    e.preventDefault(); redoAction()
  }
  if ((e.ctrlKey || e.metaKey) && e.key === 'y') {
    e.preventDefault(); redoAction()
  }
}

const undoAction = () => {
  if (!editorRef.value) return
  const view = editorRef.value.getEditor()
  if (!view) return
  // 触发 CodeMirror 撤销
  view.dispatch({ effects: undo(view) })
}

const redoAction = () => {
  if (!editorRef.value) return
  const view = editorRef.value.getEditor()
  if (!view) return
  view.dispatch({ effects: redo(view) })
}

onMounted(() => { document.addEventListener('keydown', handleKeydown) })
onUnmounted(() => { document.removeEventListener('keydown', handleKeydown) })
</script>

<style scoped>
.editor-area { display: flex; flex-direction: column; overflow: hidden; flex: 1; }
.editor-tabs {
  display: flex; background: var(--bg-tertiary); border-bottom: 1px solid var(--border-color);
  min-height: 30px; overflow-x: auto; flex-shrink: 0;
}
.editor-tabs button {
  display: flex; align-items: center; gap: 4px; padding: 4px 10px;
  background: var(--tab-inactive-bg); border: none; border-right: 1px solid var(--border-color);
  color: var(--text-secondary); font-size: 12px; cursor: pointer; white-space: nowrap;
  min-width: 0; flex-shrink: 0;
}
.editor-tabs button.active {
  background: var(--tab-active-bg); color: var(--text-primary);
  border-bottom: 1px solid var(--bg-primary); margin-bottom: -1px;
}
.editor-tabs button:hover { color: var(--text-primary); }
.tab-spacer { flex: 1; }
.tb-action {
  background: none; border: none; color: var(--text-muted); padding: 2px 6px;
  cursor: pointer; display: flex; align-items: center; border-radius: 3px; margin: 0 1px;
}
.tb-action:hover:not(:disabled) { color: var(--text-primary); background: var(--bg-hover); }
.tb-action:disabled { opacity: 0.3; cursor: default; }
.tab-icon { font-size: 12px; }
.tab-name { overflow: hidden; text-overflow: ellipsis; max-width: 120px; }
.dirty-mark { color: #e2b714; font-size: 12px; } /* yellow — universal indicator */
.tab-close { font-size: 14px; margin-left: 4px; padding: 0 2px; opacity: 0.6; }
.tab-close:hover { opacity: 1; }
.editor-body { flex: 1; overflow: hidden; position: relative; }
.welcome {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  height: 100%; color: var(--text-muted); gap: 8px;
}
.welcome-logo { font-size: 48px; font-weight: bold; color: var(--accent); }
.welcome-text { font-size: 18px; color: var(--text-secondary); }
.welcome-sub { font-size: 13px; }
.workspace-info { margin-top: 20px; text-align: center; font-size: 12px; }
.editor-wrapper { height: 100%; overflow: hidden; }
</style>
