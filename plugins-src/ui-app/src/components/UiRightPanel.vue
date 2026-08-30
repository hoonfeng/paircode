<template>
  <div class="right-container"
       :class="{ 'panel-only': panelMode }">
    <RightPanel :panel-mode="panelMode" />
  </div>
</template>

<script setup>
// UiRightPanel — right-panel 槽位默认实现（ui-right-panel 插件承载）。
// ★ chat 优先薄壳（2026-08 重构）：right-panel 槽位即「conversation 对话主视图」，
//   占薄壳主列（ShellApp grid-column:3）。它撑满该列（width:100%），不再自设固定宽/
//   不再拖拽 rightPanelWidth（那是旧 4 列 IDE 网格下 chat 作为右侧 details 列的尺寸
//   语义；chat 现为主视图，宽度跟随列并随窗口自适应）。
// 桌面端面板独立模式：desktopbridge 注入 panelMode，只渲染对话面板占满全屏。
import RightPanel from './RightPanel.vue'

const panelMode = typeof window !== 'undefined' && window.__DESKTOP_PANEL_MODE__ === true
</script>

<style scoped>
.right-container {
  display: flex; flex-direction: row; overflow: hidden; position: relative;
  /* ★ 撑满 conversation 主列（ShellApp grid-column:3, minmax(0,1fr)） */
  width: 100%; min-width: 0;
}
</style>
