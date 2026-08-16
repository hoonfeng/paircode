<template>
  <div class="right-container"
       :class="{ 'focus-mode': state.focusMode, 'panel-only': panelMode }"
       :style="(state.focusMode || panelMode) ? {} : { width: (rightPanelWidth + 4 + 1 + 250) + 'px' }">
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

let dragging = false
let startX = 0, startW = 0

const startRightResize = (e) => {
  dragging = true; startX = e.clientX; startW = rightPanelWidth.value
  document.addEventListener('mousemove', onRightMove)
  document.addEventListener('mouseup', stopRightResize)
}
const onRightMove = (e) => {
  if (!dragging) return
  rightPanelWidth.value = Math.max(260, Math.min(900, startW + (startX - e.clientX)))
}
const stopRightResize = () => {
  dragging = false
  document.removeEventListener('mousemove', onRightMove)
  document.removeEventListener('mouseup', stopRightResize)
  savePanelSize()
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
