<template>
  <div class="right-container"
       :class="{ 'focus-mode': state.focusMode, 'panel-only': panelMode }"
       :style="(state.focusMode || panelMode) ? {} : { width: (rightPanelWidth + 4 + 1 + 200) + 'px' }">
    <div v-if="!panelMode && !state.focusMode" class="right-panel-resizer"
         @mousedown.prevent="startRightResize"></div>
    <RightPanel :panel-mode="panelMode" />
  </div>
</template>

<script setup>
// UiRightPanel — right-panel 槽位默认实现（ui-right-panel 插件承载）。
// 从原 App.vue 右侧容器模板拆出：右侧 resizer + RightPanel（对话面板）。
// ★ 桌面端面板独立模式：只渲染右侧面板占满全屏（隐藏 IDE 其他区域）。
import { state, rightPanelWidth, savePanelSize } from '../ui-state.js'
import RightPanel from './RightPanel.vue'

const panelMode = typeof window !== 'undefined' && window.__DESKTOP_PANEL_MODE__ === true

// ★ 同步壳 grid 列 4 宽度（ShellApp: var(--right-w)）：宿主与 bundle 根
//   宽度一致，拖拽时实时更新，避免宿主/内容 87px 空余（历史坑）。
const TOTAL_EXTRA = 4 + 1 + 200 // resizer + border + ConvSidebar 200
const syncRightWidth = () => {
  document.documentElement.style.setProperty('--right-w', (rightPanelWidth.value + TOTAL_EXTRA) + 'px')
}
syncRightWidth()

let dragging = false
let startX = 0, startW = 0

const startRightResize = (e) => {
  dragging = true; startX = e.clientX; startW = rightPanelWidth.value
  document.addEventListener('mousemove', onRightMove)
  document.addEventListener('mouseup', stopRightResize)
}
const onRightMove = (e) => {
  if (!dragging) return
  // ★ 下限 160：聊天区最小可用宽度（右侧总宽 = 160+205 = 365px，
  //   1280 窗口编辑器 ≥ 587px）。上限 400：右侧总宽 ≤ 605px（400+205），
  //   1280 窗口编辑器 ≥ 347px。
  //   拖拽到 900（旧值）→ 右侧 1105px → 编辑器被挤到负值/溢出。
  rightPanelWidth.value = Math.max(160, Math.min(400, startW + (startX - e.clientX)))
  syncRightWidth()
}
const stopRightResize = () => {
  dragging = false
  document.removeEventListener('mousemove', onRightMove)
  document.removeEventListener('mouseup', stopRightResize)
  savePanelSize()
  syncRightWidth()
}
</script>

<style scoped>
.right-container {
  grid-column: 4; grid-row: 2;
  display: flex; flex-direction: row; overflow: hidden; position: relative;
}
/* ★ 专注模式：编辑器（main-area 列 3）隐藏，对话面板从列 3 占满——文件侧边栏（列 2）仍显示 */
.right-container.focus-mode { grid-column: 3 / -1; }
.right-panel-resizer {
  width: 4px; cursor: ew-resize; background: transparent; flex-shrink: 0; z-index: 10;
}
.right-panel-resizer:hover { background: var(--accent); }
</style>
