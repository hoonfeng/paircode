<template>
  <div class="menubar">
    <div v-for="menu in menus" :key="menu.label" class="menu-group">
      <!-- ★ 2026-09-25（用户指令：帮助菜单移到标题栏左侧 + 调整样式）：
           按钮 = 文字 + chevron-down 展开指示——IDE 菜单栏惯例，明示「此处可展开」；
           展开中箭头旋转 180° 且按钮保持 hover 态，指明当前打开的是哪一项。 -->
      <button class="menu-btn"
              :class="{ open: openMenu === menu.label }"
              :ref="el => { if (el) btnRefs[menu.label] = el }"
              @click="toggleMenu($event, menu.label)"
              @mouseenter="hoverMenu(menu.label)">
        <span class="menu-btn-label">{{ menu.label }}</span>
        <span class="menu-btn-caret"><SvgIcon name="chevron-down" :size="11" /></span>
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
import { state, layout, setFocusMode, showSettings as showSettingsModal, showHelpWrapper as showHelpModal, showAbout as showAboutModal } from '../ui-state.js'
import api from '../api.js'
import { switchActivity } from '../app-actions.js'
import SvgIcon from './SvgIcon.vue'

const menus = [
  {
    label: '帮助',
    items: [
      { label: '常见问题', action: 'help-faq' },
      { label: '快速开始', action: 'help-getting-started' },
      { label: '文档中心', action: 'help-docs' },
      { label: '功能介绍', action: 'help-features' },
      { label: 'API 文档', action: 'help-api' },
      { label: '工具文档', action: 'help-tools' },
      { label: '快捷键参考', action: 'help-shortcuts' },
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
  // ★ 显式切视图 = 退出专注并显示侧栏：setFocusMode(false) 先还原侧栏原值，
  //   紧随的 sidebarVisible = true 再显式覆盖 —— 语句顺序保证「用户要看侧栏」不被还原覆盖。
  if (a === 'view-explorer') { setFocusMode(false); state.activeActivity = 'explorer'; state.sidebarVisible = true; return }
  if (a === 'view-search') { setFocusMode(false); state.activeActivity = 'search'; state.sidebarVisible = true; return }
  if (a === 'view-git') { setFocusMode(false); state.activeActivity = 'source'; state.sidebarVisible = true; return }
  if (a === 'toggle-sidebar') { layout.toggleSidebar(); return }
  if (a === 'toggle-terminal') { state.bottomPanelVisible = !state.bottomPanelVisible; state.bottomPanelTab = 'terminal'; return }
  if (a === 'toggle-right') { state.rightPanelVisible = !state.rightPanelVisible; return }
  if (a === 'focus-mode') {
    const entering = !state.focusMode
    // ★ 专注模式唯一入口：隐藏编辑器 + 收起左栏与会话列表（见 ui-state.js setFocusMode）
    setFocusMode(entering)
    if (entering) {
      // 专注态连底部面板一并收起（纯对话视图）
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
    layout.openEditor(path)
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
    setFocusMode(false)
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
  if (a === 'global-search') { setFocusMode(false); state.activeActivity = 'search'; state.sidebarVisible = true; return }
  if (a === 'find-file') { setFocusMode(false); state.activeActivity = 'search'; state.sidebarVisible = true; return }

  // ── 终端 ──
  if (a === 'new-terminal') { state.bottomPanelVisible = true; state.bottomPanelTab = 'terminal'; return }
  if (a === 'clear-terminal') { window.dispatchEvent(new CustomEvent('clear-terminal')); return }

  // ── 工具 ──
  if (a === 'open-settings') {
    if (showSettingsModal) showSettingsModal.value = true
    return
  }
  if (a === 'open-marketplace') {
    switchActivity('marketplace')
    return
  }

  // ── 帮助 ──
  if (a === 'help-faq') {
    if (showHelpModal) { showHelpModal.value = 'faq'; return }
    return
  }
  if (a === 'help-getting-started') {
    if (showHelpModal) { showHelpModal.value = 'getting-started'; return }
    return
  }
  if (a === 'help-docs') {
    if (showHelpModal) showHelpModal.value = true
    return
  }
  if (a === 'help-features') {
    if (showHelpModal) { showHelpModal.value = 'features'; return }
    return
  }
  if (a === 'help-api') {
    if (showHelpModal) { showHelpModal.value = 'api'; return }
    return
  }
  if (a === 'help-tools') {
    if (showHelpModal) { showHelpModal.value = 'tools'; return }
    return
  }
  if (a === 'help-shortcuts') {
    if (showHelpModal) { showHelpModal.value = 'shortcuts'; return }
    return
  }
  if (a === 'about') {
    if (showAboutModal) showAboutModal.value = true
    return
  }
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
/* ★ 2026-09-24 新 UI 风格适配（本组件按用户指令回归顶栏 th206，40px 高）：
   按钮收敛为 h=24 / 圆角 full / 11px 的低调胶囊，与顶栏其它入口
   （.tb-nav-pill / .plugin-slot-item）同规格；下拉面板半径 4→8、内边距 4px 0→4px、
   菜单项 13→12px 且带自身圆角；hover 由 accent 实底改为克制的 --bg-hover
   （与顶栏导航胶囊 hover 一致，不抢视觉焦点）。 */
.menubar { display: flex; flex-direction: row; align-items: center; height: 100%; gap: 0; }
.menu-group { position: relative; }
.menu-btn {
  display: inline-flex; align-items: center; gap: 4px; height: 24px; padding: 0 8px 0 10px; font-size: 11px;
  color: var(--text-muted); background: none; border: none; cursor: pointer; user-select: none; white-space: nowrap;
  border-radius: 999px; transition: color .15s, background .15s;
}
.menu-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
/* 展开中：与 hover 同态 —— 鼠标移入下拉面板后按钮仍保持高亮，明示「当前打开项」 */
.menu-btn.open { background: var(--bg-hover); color: var(--text-primary); }
/* 展开指示箭头：常驻低对比（不抢焦点），hover/展开时提亮，展开时旋转 180° */
.menu-btn-caret { display: inline-flex; align-items: center; transition: transform .15s; opacity: .7; }
.menu-btn:hover .menu-btn-caret, .menu-btn.open .menu-btn-caret { opacity: 1; }
.menu-btn.open .menu-btn-caret { transform: rotate(180deg); }
.menu-dropdown {
  position: fixed;
  min-width: 220px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 4px;
  box-shadow: var(--shadow-lg);
  z-index: 9999;
}
.menu-item {
  display: flex; align-items: center; padding: 6px 10px; cursor: pointer;
  font-size: 12px; color: var(--text-primary); gap: 24px;
  border-radius: 6px;
}
.menu-item:hover { background: var(--bg-hover); color: var(--text-primary); }
.menu-item.disabled { color: var(--text-muted); cursor: default; }
.menu-item.disabled:hover { background: none; color: var(--text-muted); }
.menu-item-label { flex: 1; white-space: nowrap; }
.menu-item-shortcut { font-size: 11px; color: var(--text-muted); flex-shrink: 0; }
.menu-item:hover .menu-item-shortcut { color: var(--text-muted); }
.menu-divider { height: 1px; background: var(--border-color); margin: 4px 6px; opacity: 0.6; }
</style>
