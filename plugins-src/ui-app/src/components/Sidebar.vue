<template>
  <div class="sidebar" :style="{ width: sidebarWidth + 'px' }">
    <div class="sidebar-header">
      <span>{{ headerTitle }}</span>
    </div>
    <div class="sidebar-content">
      <FileExplorer v-if="state.activeActivity === 'explorer'" />
      <SearchPanel v-else-if="state.activeActivity === 'search'" />
      <PluginPanel v-else-if="state.activeActivity === 'plugins'" />
      <div v-else class="sidebar-placeholder">
        <span>面板加载中...</span>
      </div>
    </div>
    <!-- 拖拽分隔条（放在 Sidebar 内，绝对定位在右侧边缘） -->
    <div class="sidebar-resizer" @mousedown.prevent="startResize"></div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { state, sidebarWidth } from '../ui-state.js'
import FileExplorer from './FileExplorer.vue'
import SearchPanel from './SearchPanel.vue'
import PluginPanel from './PluginPanel.vue'

const headerTitle = computed(() => {
  const titles = { explorer: '文件浏览器', search: '搜索', source: '源代码管理', plugins: '插件' }
  return titles[state.activeActivity] || ''
})

let dragging = false
let startX = 0
let startW = 0

function startResize(e) {
  dragging = true
  startX = e.clientX
  startW = sidebarWidth.value
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'ew-resize'
  document.body.style.userSelect = 'none'
}

function onMove(e) {
  if (!dragging) return
  sidebarWidth.value = Math.max(120, Math.min(800, startW + (e.clientX - startX)))
}

function stopResize() {
  dragging = false
  document.removeEventListener('mousemove', onMove)
  document.removeEventListener('mouseup', stopResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  try {
    localStorage.setItem('paircode-sidebar-width', String(sidebarWidth.value))
  } catch {}
}
</script>

<style scoped>
.sidebar {
  background: var(--sidebar-bg);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
  /* ★ bundle 根撑满宿主：无 height:100% 时高度=内容（FileExplorer 1216px
     溢出窗口 → 工具集被裁到窗口外；短面板 → 底部空余）。 */
  height: 100%;
}
.sidebar-header {
  height: 32px;
  display: flex;
  align-items: center;
  padding: 0 12px;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}
.sidebar-content {
  flex: 1;
  overflow: auto;
}
.sidebar-placeholder {
  padding: 20px;
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
}
.sidebar-resizer {
  position: absolute;
  right: -2px;
  top: 0;
  width: 6px;
  height: 100%;
  cursor: ew-resize;
  z-index: 10;
  background: transparent;
}
.sidebar-resizer:hover {
  background: var(--accent);
}
</style>
