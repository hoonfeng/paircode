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
  grid-column: 3; grid-row: 2;
  display: flex; flex-direction: column; min-width: 0; overflow: hidden;
  /* ★ bundle 根必须撑满宿主（plugin-slot-host 748px），否则 flex 高度 =
     内容高度 → 编辑区/终端下方大段空余（历史坑：520 vs 748 → 228px 空白） */
  height: 100%;
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
