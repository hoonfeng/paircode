<template>
  <div class="menubar">
    <div v-for="menu in menus" :key="menu.label" class="menu-group">
      <button class="menu-btn"
              :ref="el => { if (el) btnRefs[menu.label] = el }"
              @click="toggleMenu($event, menu.label)"
              @mouseenter="hoverMenu(menu.label)">
        {{ menu.label }}
      </button>
    </div>
    <div v-if="openMenu" class="menu-dropdown"
         :style="dropdownStyle"
         @mouseleave="scheduleClose()"
         @mouseenter="cancelClose()">
      <template v-for="(item, i) in currentItems" :key="i">
        <div v-if="item.divider" class="menu-divider"></div>
        <div v-else-if="item.label"
             :class="['menu-item', { disabled: item.disabled }]"
             @click="!item.disabled && execItem(item)">
          <span class="menu-item-label">{{ item.label }}</span>
          <span v-if="item.shortcut" class="menu-item-shortcut">{{ item.shortcut }}</span>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { state } from '../main.js'
import api from '../api.js'

const menus = [
  {
    label: '文件',
    items: [
      { label: '新建文件', shortcut: 'Ctrl+N', action: 'new-file' },
      { label: '打开文件…', shortcut: 'Ctrl+O', action: 'open-file' },
      { label: '打开文件夹…', action: 'open-folder' },
      { label: '添加文件夹到工作区…', action: 'add-folder' },
      { divider: true },
      { label: '保存', shortcut: 'Ctrl+S', action: 'save' },
      { label: '全部保存', action: 'save-all' },
      { divider: true },
      { label: '保存工作区', action: 'save-workspace' },
      { label: '管理工作区文件夹…', action: 'manage-workspace' },
      { divider: true },
      { label: '关闭项目', action: 'close-project' },
      { label: '关闭工作区', action: 'close-workspace' },
    ],
  },
  {
    label: '编辑',
    items: [
      { label: '撤销', shortcut: 'Ctrl+Z', action: 'undo' },
      { label: '重做', shortcut: 'Ctrl+Shift+Z', action: 'redo' },
      { divider: true },
      { label: '剪切', shortcut: 'Ctrl+X', action: 'cut' },
      { label: '复制', shortcut: 'Ctrl+C', action: 'copy' },
      { label: '粘贴', shortcut: 'Ctrl+V', action: 'paste' },
      { divider: true },
      { label: '查找对话', shortcut: 'Ctrl+F', action: 'find-chat' },
      { label: '跨文件搜索', shortcut: 'Ctrl+Shift+F', action: 'global-search' },
      { label: '命令面板', shortcut: 'Ctrl+P', action: 'find-file' },
    ],
  },
  {
    label: '视图',
    items: [
      { label: '文件资源管理器', shortcut: 'Ctrl+Shift+E', action: 'view-explorer' },
      { label: '搜索', action: 'view-search' },
      { label: '源代码管理', action: 'view-git' },
      { divider: true },
      { label: '专注模式', shortcut: 'Ctrl+K', action: 'focus-mode' },
      { label: '切换侧边栏', shortcut: 'Ctrl+B', action: 'toggle-sidebar' },
      { label: '切换终端', shortcut: 'Ctrl+J', action: 'toggle-terminal' },
      { label: '切换右侧面板', shortcut: 'Ctrl+Shift+C', action: 'toggle-right' },
    ],
  },
  {
    label: '终端',
    items: [
      { label: '新建终端', action: 'new-terminal' },
      { divider: true },
      { label: '清屏', action: 'clear-terminal' },
    ],
  },
  {
    label: 'Agent',
    items: [
      { label: '性能监控', icon: 'activity', action: 'perf-monitor' },
      { label: 'Agent 监控', icon: 'terminal', action: 'agent-monitor' },
      { label: '进化图', icon: 'git-branch', action: 'evolution-graph' },
      { label: '探索项目知识库', icon: 'search', action: 'explore-knowledge' },
    ],
  },
  {
    label: '帮助',
    items: [
      { label: '快捷键参考', action: 'help-shortcuts' },
      { label: '文档', action: 'help-docs' },
      { label: '检查更新', action: 'check-update' },
      { divider: true },
      { label: '报告问题', action: 'report-issue' },
      { divider: true },
      { label: '关于 PairCode IDE', action: 'about' },
    ],
  },
]

const openMenu = ref(null)
const btnRefs = ref({})
const dropdownPos = ref({ x: 0, y: 0 })
let closeTimer = null

const currentItems = computed(() => {
  const m = menus.find(m => m.label === openMenu.value)
  return m ? m.items : []
})

const dropdownStyle = computed(() => ({
  left: dropdownPos.value.x + 'px',
  top: dropdownPos.value.y + 'px',
}))

function toggleMenu(event, label) {
  event.stopPropagation()
  if (openMenu.value === label) {
    openMenu.value = null
    return
  }
  const btn = btnRefs.value[label]
  if (btn) {
    const rect = btn.getBoundingClientRect()
    dropdownPos.value = { x: rect.left, y: rect.bottom }
  }
  openMenu.value = label
  cancelClose()
}

function hoverMenu(label) {
  if (openMenu.value && openMenu.value !== label) {
    const btn = btnRefs.value[label]
    if (btn) {
      const rect = btn.getBoundingClientRect()
      dropdownPos.value = { x: rect.left, y: rect.bottom }
    }
    openMenu.value = label
  }
}

const scheduleClose = () => {
  closeTimer = setTimeout(() => { openMenu.value = null }, 200)
}
const cancelClose = () => {
  if (closeTimer) { clearTimeout(closeTimer); closeTimer = null }
}

function closeMenu() { openMenu.value = null }
defineExpose({ closeMenu })

const execItem = async (item) => {
  openMenu.value = null
  const a = item.action

  // ── 视图操作 ──
  if (a === 'view-explorer') { state.activeActivity = 'explorer'; state.sidebarVisible = true; return }
  if (a === 'view-search') { state.activeActivity = 'search'; state.sidebarVisible = true; return }
  if (a === 'view-git') { state.activeActivity = 'source'; state.sidebarVisible = true; return }
  if (a === 'toggle-sidebar') { state.sidebarVisible = !state.sidebarVisible; return }
  if (a === 'toggle-terminal') { state.bottomPanelVisible = !state.bottomPanelVisible; state.bottomPanelTab = 'terminal'; return }
  if (a === 'toggle-right') { state.rightPanelVisible = !state.rightPanelVisible; return }
  if (a === 'focus-mode') {
    state.focusMode = !state.focusMode
    if (state.focusMode) {
      state.sidebarVisible = false
      state.bottomPanelVisible = false
    }
    return
  }

  // ── 文件操作 ──
  if (a === 'new-file') {
    const name = await window.$prompt('文件名:', '', '新建文件')
    if (!name) return
    const dir = state.activeFile ? state.activeFile.substring(0, state.activeFile.lastIndexOf('\\'))
      : (state.workspaceFolders[0] || state.workspaceRoot)
    if (!dir) return window.$toast('请先在文件浏览器中打开一个目录', 'warning')
    try {
      await api.apiPost('/fs/write', { path: dir + '\\' + name, content: '' })
      window.dispatchEvent(new CustomEvent('refresh-tree'))
    } catch (err) { window.$toast('创建失败: ' + err.message, 'error') }
    return
  }
  if (a === 'open-file') {
    const path = await window.$prompt('输入文件路径:', '', '打开文件')
    if (!path) return
    if (!state.openFiles.includes(path)) state.openFiles.push(path)
    state.activeFile = path
    return
  }
  if (a === 'open-folder') {
    const path = await window.$prompt('输入文件夹路径:', '', '打开文件夹')
    if (!path) return
    try {
      await api.apiPost('/workspace', { action: 'add-folder', path })
      window.dispatchEvent(new CustomEvent('refresh-tree'))
    } catch (err) { window.$toast('添加失败: ' + err.message, 'error') }
    return
  }
  if (a === 'add-folder') {
    const path = await window.$prompt('输入要添加的文件夹路径:', '', '添加文件夹')
    if (!path) return
    try {
      await api.apiPost('/workspace', { action: 'add-folder', path })
      window.dispatchEvent(new CustomEvent('refresh-tree'))
    } catch (err) { window.$toast('添加失败: ' + err.message, 'error') }
    return
  }
  if (a === 'save') {
    if (!state.activeFile || state.fileContents[state.activeFile] === undefined) return
    try {
      await api.apiPost('/fs/write', { path: state.activeFile, content: state.fileContents[state.activeFile] })
      state.fileDirty[state.activeFile] = false
    } catch (err) { window.$toast('保存失败: ' + err.message, 'error') }
    return
  }
  if (a === 'save-all') {
    for (const f of state.openFiles) {
      if (state.fileDirty[f] && state.fileContents[f] !== undefined) {
        try {
          await api.apiPost('/fs/write', { path: f, content: state.fileContents[f] })
          state.fileDirty[f] = false
        } catch {}
      }
    }
    return
  }
  if (a === 'save-workspace') {
    window.$toast('工作区已自动保存', 'success')
    return
  }
  if (a === 'manage-workspace') {
    state.activeActivity = 'explorer'
    state.sidebarVisible = true
    return
  }
  if (a === 'close-project') {
    state.openFiles = []
    state.activeFile = ''
    state.fileContents = {}
    return
  }
  if (a === 'close-workspace') {
    if (await window.$confirm('关闭工作区？所有未保存更改将丢失。')) {
      state.workspaceRoot = ''
      state.workspaceFolders = []
      state.fileTree = []
      state.openFiles = []
      state.activeFile = ''
      state.fileContents = {}
    }
    return
  }

  // ── 编辑操作 ──
  if (a === 'undo') {
    window.dispatchEvent(new CustomEvent('editor-undo'))
    return
  }
  if (a === 'redo') {
    window.dispatchEvent(new CustomEvent('editor-redo'))
    return
  }
  if (a === 'cut' || a === 'copy' || a === 'paste') {
    document.execCommand(a)
    return
  }

  // ── 搜索 ──
  if (a === 'find-chat') { state.rightPanelVisible = true; return }
  if (a === 'global-search') { state.activeActivity = 'search'; state.sidebarVisible = true; return }
  if (a === 'find-file') { state.activeActivity = 'search'; state.sidebarVisible = true; return }

  // ── 终端 ──
  if (a === 'new-terminal') { state.bottomPanelVisible = true; state.bottomPanelTab = 'terminal'; return }
  if (a === 'clear-terminal') { window.dispatchEvent(new CustomEvent('clear-terminal')); return }

  // ── Agent ──
  if (a === 'stop-agent') {
    window.dispatchEvent(new CustomEvent('stop-agent'))
    if (state.chatSessionId) {
      try { await api.stopChat(state.chatSessionId) } catch {}
    }
    return
  }
  if (a === 'perf-monitor') { window.$toast('性能监控面板开发中', 'info'); return }
  if (a === 'agent-monitor') {
    state.bottomPanelVisible = true
    state.bottomPanelTab = 'terminal'
    return
  }
  if (a === 'evolution-graph') { window.$toast('进化图功能开发中', 'info'); return }
  if (a === 'explore-knowledge') {
    // 打开搜索面板以探索知识库
    state.activeActivity = 'search'
    state.sidebarVisible = true
    return
  }

  // ── 帮助 ──
  if (a === 'help-shortcuts') {
    window.$alert('快捷键：\nCtrl+N 新建 | Ctrl+S 保存 | Ctrl+O 打开\nCtrl+F 查找对话 | Ctrl+P 命令面板\nCtrl+B 侧栏 | Ctrl+J 终端 | Ctrl+K 专注\nCtrl+Z 撤销 | Ctrl+Shift+Z 重做', '快捷键')
    return
  }
  if (a === 'help-docs') { window.open('https://github.com', '_blank'); return }
  if (a === 'check-update') { window.$alert('当前版本：v0.1.0（Web IDE）', '检查更新'); return }
  if (a === 'report-issue') { window.open('https://github.com', '_blank'); return }
  if (a === 'about') { window.$alert('PairCode IDE v0.1.0\n基于 GWui 的现代化 AI IDE', '关于'); return }
}

// 点击外部关闭菜单
const handleDocClick = (e) => {
  if (openMenu.value) {
    const path = e.composedPath ? e.composedPath() : []
    const inMenu = path.some(el => el.classList && el.classList.contains('menu-dropdown'))
    const inBtn = path.some(el => el.classList && el.classList.contains('menu-btn') || (el.closest && el.closest('.menubar')))
    if (!inMenu && !inBtn) openMenu.value = null
  }
}
onMounted(() => document.addEventListener('click', handleDocClick))
onUnmounted(() => document.removeEventListener('click', handleDocClick))
</script>

<style scoped>
.menubar { display: flex; flex-direction: row; align-items: center; height: 100%; gap: 0; }
.menu-group { position: relative; }
.menu-btn {
  display: flex; align-items: center; height: 30px; padding: 0 10px; font-size: 13px;
  color: var(--text-secondary); background: none; border: none; cursor: pointer; user-select: none; white-space: nowrap;
}
.menu-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.menu-dropdown {
  position: fixed;
  min-width: 220px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 4px 0;
  box-shadow: 0 4px 20px rgba(0,0,0,0.5);
  z-index: 9999;
}
.menu-item {
  display: flex; align-items: center; padding: 5px 14px; cursor: pointer;
  font-size: 13px; color: var(--text-primary); gap: 24px;
}
.menu-item:hover { background: var(--accent); color: #fff; }
.menu-item.disabled { color: var(--text-muted); cursor: default; }
.menu-item.disabled:hover { background: none; color: var(--text-muted); }
.menu-item-label { flex: 1; white-space: nowrap; }
.menu-item-shortcut { font-size: 11px; color: var(--text-muted); flex-shrink: 0; }
.menu-item:hover .menu-item-shortcut { color: rgba(255,255,255,0.7); }
.menu-divider { height: 1px; background: var(--border-color); margin: 4px 8px; opacity: 0.6; }
</style>
