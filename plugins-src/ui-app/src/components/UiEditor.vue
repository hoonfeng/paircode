<template>
  <div class="main-area">
    <EditorArea />
    <div class="bottom-panel" v-if="state.bottomPanelVisible"
         :style="{ height: bottomPanelHeight + 'px' }">
      <div class="panel-content">
        <TerminalPanel @close-panel="state.bottomPanelVisible = false" />
      </div>
      <div class="panel-resizer" @mousedown.prevent="startBottomResize"></div>
    </div>
  </div>
</template>

<script setup>
// UiEditor — editor 槽位默认实现（ui-editor 插件承载）。
// 从原 App.vue 主编辑区模板拆出：EditorArea + 底部终端面板 + 底部拖拽 resizer。
import { state, bottomPanelHeight, savePanelSize } from '../ui-state.js'
import EditorArea from './EditorArea.vue'
import TerminalPanel from './TerminalPanel.vue'

let dragging = false
let startY = 0, startH = 0

const startBottomResize = (e) => {
  dragging = true; startY = e.clientY; startH = bottomPanelHeight.value
  document.addEventListener('mousemove', onBottomMove)
  document.addEventListener('mouseup', stopBottomResize)
}
const onBottomMove = (e) => {
  if (!dragging) return
  bottomPanelHeight.value = Math.max(60, Math.min(800, startH + (startY - e.clientY)))
}
const stopBottomResize = () => {
  dragging = false
  document.removeEventListener('mousemove', onBottomMove)
  document.removeEventListener('mouseup', stopBottomResize)
  savePanelSize()
}
</script>

<style scoped>
.main-area {
  /* ★ chat 优先薄壳（2026-08 重构）：editor 槽位现为「details 辅助列」（ShellApp
     grid-column:4）。此处不再写 grid-column/grid-row（宿主 details-area 是 grid
     项，非 grid 容器，子元素 grid 定位无效）；改为撑满宿主（width/height 100%）。 */
  display: flex; flex-direction: column; min-width: 0; overflow: hidden;
  width: 100%; height: 100%;
}
.main-area > :first-child { flex: 1; }
.bottom-panel {
  position: relative; background: var(--bg-secondary);
  border-top: 1px solid var(--border-color);
  display: flex; flex-direction: column; min-height: 60px;
}
.panel-content { flex: 1; overflow: hidden; padding: 0; }
.panel-resizer { position: absolute; top: -3px; left: 0; right: 0; height: 6px; cursor: ns-resize; z-index: 10; }
</style>
