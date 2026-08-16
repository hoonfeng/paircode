<template>
  <div class="app-root" :class="{ 'panel-only': panelMode }">
    <!-- ═══ 按槽位细粒度装配（2026-08-16）：壳 = 纯布局骨架容器 ═══
         每个区域一个单槽位挂载点：owner 为空 → 空态提示（该区域插件未装配）；
         owner 有 → 渲染插件到 hostRef。全部 7 个区域由独立磁盘插件
         （ui-titlebar/ui-activitybar/ui-sidebar/ui-editor/ui-right-panel/
         ui-statusbar/ui-modals）装配，可独立装卸/替换。 -->

    <!-- titlebar 槽位（single）：标题栏整条 -->
    <div v-if="!panelMode && !slots.titlebar.owner.value" class="slot-empty plugin-area-titlebar">
      <span>标题栏未装配（ui-titlebar）</span>
    </div>
    <div v-else-if="!panelMode" :ref="slots.titlebar.hostRef"
         class="plugin-slot-host plugin-area-titlebar"></div>

    <!-- activitybar 槽位（single）：左侧活动栏竖列 -->
    <div v-if="!panelMode && !slots.activitybar.owner.value" class="slot-empty plugin-area-activitybar">
      <span>⦿</span>
    </div>
    <div v-else-if="!panelMode" :ref="slots.activitybar.hostRef"
         class="plugin-slot-host plugin-area-activitybar"></div>

    <!-- sidebar 槽位（single）：左侧栏（文件/搜索/Git），sidebarVisible 控制 -->
    <div v-if="!panelMode && state.sidebarVisible && !slots.sidebar.owner.value"
         class="slot-empty plugin-area-sidebar"><span>侧栏未装配（ui-sidebar）</span></div>
    <div v-else-if="!panelMode && state.sidebarVisible" :ref="slots.sidebar.hostRef"
         class="plugin-slot-host plugin-area-sidebar"></div>

    <!-- editor 槽位（single）：主编辑区，focusMode 隐藏 -->
    <div v-if="!panelMode && !state.focusMode && !slots.editor.owner.value"
         class="slot-empty main-area"><span>编辑器未装配（ui-editor）</span></div>
    <div v-else-if="!panelMode && !state.focusMode" :ref="slots.editor.hostRef"
         class="plugin-slot-host main-area"></div>

    <!-- right-panel 槽位（single）：右侧对话容器 -->
    <div v-if="(state.rightPanelVisible || panelMode) && !slots.rightPanel.owner.value"
         class="slot-empty right-container"
         :class="{ 'focus-mode': state.focusMode, 'panel-only': panelMode }"><span>对话面板未装配（ui-right-panel）</span></div>
    <div v-else-if="(state.rightPanelVisible || panelMode)" :ref="slots.rightPanel.hostRef"
         class="plugin-slot-host right-container"
         :class="{ 'focus-mode': state.focusMode, 'panel-only': panelMode }"></div>

    <!-- statusbar 槽位（single）：底部状态栏 -->
    <div v-if="!panelMode && !slots.statusbar.owner.value" class="slot-empty app-statusbar-host">
      <span>状态栏未装配（ui-statusbar）</span>
    </div>
    <div v-else-if="!panelMode" :ref="slots.statusbar.hostRef"
         class="plugin-slot-host app-statusbar-host"></div>

    <!-- modals 槽位（single）：全局模态框/浮动层（fixed 不占 grid 格） -->
    <div v-if="!slots.modals.owner.value" class="modals-empty"></div>
    <div v-else :ref="slots.modals.hostRef" class="plugin-slot-host modals-host"></div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted } from 'vue'
import { useSingleSlot, syncClientHalves, startPolling, stopPolling, registerBuiltinSlot } from './plugin-runtime.js'
import { state } from './ui-state.js'
import api from './api.js'
import { initAppGlobals, cleanupAppGlobals, desktopPrefetch, loadWsList } from './app-actions.js'

// ★ 桌面端面板独立模式：desktopbridge 注入 window.__DESKTOP_PANEL_MODE__，
//   此时只渲染右侧面板占满全屏，隐藏 IDE 其他区域。
const panelMode = typeof window !== 'undefined' && window.__DESKTOP_PANEL_MODE__ === true

// ★ 一切皆插件：壳注册 8 个内置槽位定义（插件面板装配视图可见；占用统一）。
registerBuiltinSlot('titlebar', { title: '标题栏（含 logo/菜单/标题）', desc: '默认实现：ui-titlebar 插件' })
registerBuiltinSlot('activitybar', { title: '活动栏（左侧竖条）', desc: '默认实现：ui-activitybar 插件' })
registerBuiltinSlot('sidebar', { title: '左侧栏（文件/搜索/Git）', desc: '默认实现：ui-sidebar 插件' })
registerBuiltinSlot('editor', { title: '主编辑区（编辑器+终端）', desc: '默认实现：ui-editor 插件' })
registerBuiltinSlot('right-panel', { title: '右侧容器（对话外壳）', desc: '默认实现：ui-right-panel 插件' })
registerBuiltinSlot('statusbar', { title: '状态栏（底部）', desc: '默认实现：ui-statusbar 插件' })
registerBuiltinSlot('chat', { title: '对话面板（rp-body 区）', desc: 'ui-right-panel 内对话+输入区' })
registerBuiltinSlot('modals', { title: '全局模态框/浮动层', desc: '默认实现：ui-modals 插件' })

const slots = {
  titlebar: useSingleSlot('titlebar'),
  activitybar: useSingleSlot('activitybar'),
  sidebar: useSingleSlot('sidebar'),
  editor: useSingleSlot('editor'),
  rightPanel: useSingleSlot('right-panel'),
  statusbar: useSingleSlot('statusbar'),
  modals: useSingleSlot('modals'),
}
for (const s of Object.values(slots)) s.init()

onMounted(async () => {
  for (const s of Object.values(slots)) s.start()
  desktopPrefetch()
  initAppGlobals()
  loadWsList()

  // ★ 全局装载 client 半（必须最前执行，不依赖其他 await——否则 ui-* 插件
  //   的 client 半永不装载，壳永远停在空态）。链路：
  //   listPlugins → 补 clientCode → syncClientHalves → 各 ui-* client 半执行
  //   → registerSlot → 槽位变化通知本组件 → 渲染插件到各 hostRef 容器。
  try {
    const list = (await api.listPlugins()) || []
    for (const p of list) {
      if (p.hasClient && !p.clientCode) {
        try {
          const d = await api.getPluginDetail(p.name)
          if (d && d.clientCode) p.clientCode = d.clientCode
        } catch (e) { /* 忽略：detail 失败跳过 */ }
      }
    }
    await syncClientHalves(list)
    startPolling() // 事件轮询全局启动（host→client 事件分发；幂等）
  } catch (e) {
    console.warn('[shell] client 半装载失败', e)
  }
})

onUnmounted(() => {
  for (const s of Object.values(slots)) s.stop()
  stopPolling()
  cleanupAppGlobals()
})
</script>

<style scoped>
.app-root {
  display: grid;
  grid-template-columns: 48px auto 1fr auto;
  grid-template-rows: 30px 1fr 22px;
  width: 100%; height: 100%;
  background: var(--bg-primary);
  color: var(--text-primary);
  overflow: hidden;
  font-family: var(--font-ui);
}
/* ★ 桌面端面板独立模式：只渲染右侧面板，占满整个窗口 */
.app-root.panel-only {
  grid-template-columns: 1fr;
  grid-template-rows: 1fr;
}
.app-root.panel-only .right-container {
  grid-column: 1; grid-row: 1;
  width: 100% !important;
  height: 100%;
}
/* 整区替换槽位（single）宿主：与内置区域同 grid 位置/尺寸 */
.plugin-area-titlebar { grid-column: 1 / -1; grid-row: 1; height: 30px; }
.plugin-area-activitybar { grid-column: 1; grid-row: 2; width: 48px; }
.plugin-area-sidebar { grid-column: 2; grid-row: 2; height: 100%; overflow: hidden; }
.main-area {
  grid-column: 3; grid-row: 2;
  display: flex; flex-direction: column; min-width: 0; overflow: hidden;
}
.right-container {
  grid-column: 4; grid-row: 2;
  display: flex; flex-direction: row; overflow: hidden; position: relative;
}
.right-container.focus-mode { grid-column: 3 / -1; }
.app-statusbar-host { grid-column: 1 / -1; grid-row: 3; z-index: 30; height: 22px; }
.plugin-slot-host { height: 100%; overflow: hidden; }
/* ★ 插件渲染的子元素必须撑满宿主（bundle 根 auto 宽度不随宿主 grid 拉伸——
   focus-mode 下宿主被 grid 拉到 3/-1 全宽，子元素保持内容宽 → 右侧大片空余） */
.plugin-slot-host.right-container > * { width: 100%; min-width: 0; }
/* modals 槽位：fixed 全屏浮层容器（不占 grid 格） */
.modals-host { position: fixed; inset: 0; z-index: 200; pointer-events: none; }
.modals-host > * { pointer-events: auto; }
.modals-empty { display: none; }
/* 空态占位（区域插件未装配时显示） */
.slot-empty {
  display: flex; align-items: center; justify-content: center;
  color: var(--text-muted); font-size: 12px;
  background: var(--bg-primary);
  border: 1px dashed var(--border-color);
  min-height: 0;
}
</style>
