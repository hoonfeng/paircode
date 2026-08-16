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
      <button class="escape-link" @click="pluginsOpen = true">打开插件面板</button>
    </div>
    <div v-else-if="!panelMode" :ref="slots.titlebar.hostRef"
         class="plugin-slot-host plugin-area-titlebar"></div>

    <!-- activitybar 槽位（single）：左侧活动栏竖列 -->
    <div v-if="!panelMode && !slots.activitybar.owner.value" class="slot-empty plugin-area-activitybar">
      <span>⦿</span>
      <button class="escape-link" @click="pluginsOpen = true">面板</button>
    </div>
    <div v-else-if="!panelMode" :ref="slots.activitybar.hostRef"
         class="plugin-slot-host plugin-area-activitybar"></div>

    <!-- sidebar 槽位（single）：左侧栏（文件/搜索/Git），sidebarVisible 控制 -->
    <div v-if="!panelMode && state.sidebarVisible && !slots.sidebar.owner.value"
         class="slot-empty plugin-area-sidebar"><span>侧栏未装配（ui-sidebar）</span><button class="escape-link" @click="pluginsOpen = true">打开插件面板</button></div>
    <div v-else-if="!panelMode && state.sidebarVisible" :ref="slots.sidebar.hostRef"
         class="plugin-slot-host plugin-area-sidebar"></div>

    <!-- editor 槽位（single）：主编辑区，focusMode 隐藏 -->
    <div v-if="!panelMode && !state.focusMode && !slots.editor.owner.value"
         class="slot-empty main-area"><span>编辑器未装配（ui-editor）</span><button class="escape-link" @click="pluginsOpen = true">打开插件面板</button></div>
    <div v-else-if="!panelMode && !state.focusMode" :ref="slots.editor.hostRef"
         class="plugin-slot-host main-area"></div>

    <!-- right-panel 槽位（single）：右侧对话容器 -->
    <div v-if="(state.rightPanelVisible || panelMode) && !slots.rightPanel.owner.value"
         class="slot-empty right-container"
         :class="{ 'focus-mode': state.focusMode, 'panel-only': panelMode }"><span>对话面板未装配（ui-right-panel）</span><button class="escape-link" @click="pluginsOpen = true">打开插件面板</button></div>
    <div v-else-if="(state.rightPanelVisible || panelMode)" :ref="slots.rightPanel.hostRef"
         class="plugin-slot-host right-container"
         :class="{ 'focus-mode': state.focusMode, 'panel-only': panelMode }"></div>

    <!-- statusbar 槽位（single）：底部状态栏 -->
    <div v-if="!panelMode && !slots.statusbar.owner.value" class="slot-empty app-statusbar-host">
      <span>状态栏未装配（ui-statusbar）</span>
      <button class="escape-link" @click="pluginsOpen = true">打开插件面板</button>
    </div>
    <div v-else-if="!panelMode" :ref="slots.statusbar.hostRef"
         class="plugin-slot-host app-statusbar-host"></div>

    <!-- modals 槽位（single）：全局模态框/浮动层（fixed 不占 grid 格） -->
    <div v-if="!slots.modals.owner.value" class="modals-empty"></div>
    <div v-else :ref="slots.modals.hostRef" class="plugin-slot-host modals-host"></div>
  </div>

  <!-- ═══ 壳级逃生口（不依赖任何插件）：插件面板浮动入口 ═══
       插件面板由 ui-sidebar 插件承载（Sidebar.vue 的 plugins 页）——若该插件
       UI 被停用/异常，面板入口即消失（死锁）。此按钮常驻壳层，即使全部
       7 个区域插件都被停用，也能重新打开插件面板恢复装配。
       浮动面板直接渲染 PluginPanel（壳编译副本），与侧边栏内面板共享状态。 -->
  <Teleport to="body">
    <button v-if="!panelMode" class="plugin-escape-btn" title="插件面板（壳级入口，不受插件停用影响）" @click="pluginsOpen = true">
      <svg viewBox="0 0 16 16" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.5">
        <rect x="2" y="6" width="12" height="8" rx="1.5"/><path d="M5 6V4.5a3 3 0 0 1 6 0V6"/>
      </svg>
    </button>
    <div v-if="pluginsOpen" class="plugin-escape-overlay" @click.self="pluginsOpen = false">
      <div class="plugin-escape-panel">
        <div class="plugin-escape-head">
          <span>插件面板（壳级入口）</span>
          <button class="plugin-escape-close" title="关闭" @click="pluginsOpen = false">✕</button>
        </div>
        <div class="plugin-escape-body">
          <PluginPanel />
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { useSingleSlot, syncClientHalves, startPolling, stopPolling, registerBuiltinSlot, loadAssemblyFile } from './plugin-runtime.js'
import { state } from './ui-state.js'
import api from './api.js'
import { initAppGlobals, cleanupAppGlobals, desktopPrefetch, loadWsList } from './app-actions.js'
import PluginPanel from './components/PluginPanel.vue'

// ★ 桌面端面板独立模式：desktopbridge 注入 window.__DESKTOP_PANEL_MODE__，
//   此时只渲染右侧面板占满全屏，隐藏 IDE 其他区域。
const panelMode = typeof window !== 'undefined' && window.__DESKTOP_PANEL_MODE__ === true
const pluginsOpen = ref(false) // 壳级逃生口：插件面板浮动层开关

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
    // ★ 先合并磁盘装配文件（.pair/ui-assembly.json，用户可编辑的逃生通道）——
    //   再装载 client 半（注册槽位时 getSlotOwner/getSlotUIList 读到合并后的状态）。
    await loadAssemblyFile()
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
  /* ★ 列保护：编辑器列 minmax(340px,1fr) 保证主区不被右侧/侧边栏挤压；
     右侧列 var(--right-w) 由 ui-right-panel 包（拖拽/初始化）同步，
     宿主与 bundle 根宽度一致（无空余）。--right-w = rpw+205
     (chat rpw + conv 200 + resizer/border 5)，rpw 上限 360 → 右侧 ≤565px，
     1280 窗口编辑器 ≥ 1280-48-280-565 = 387px。 */
  grid-template-columns: 48px auto minmax(340px, 1fr) var(--right-w, 525px);
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
  display: flex; flex-direction: row; gap: 8px;
  align-items: center; justify-content: center;
  color: var(--text-muted); font-size: 12px;
  background: var(--bg-primary);
  border: 1px dashed var(--border-color);
  min-height: 0;
}
/* activitybar 是竖条（~48px 宽）：空态改纵向排列 */
.plugin-area-activitybar.slot-empty { flex-direction: column; gap: 4px; padding: 4px; }
.plugin-area-activitybar.slot-empty .escape-link { font-size: 11px; padding: 2px 8px; }
/* 空态内的「打开插件面板」恢复入口（上下文感知注入：只在区域未装配时出现，
   插件全正常时零干扰；与常驻逃生按钮互为双保险） */
.escape-link {
  background: none; border: 1px solid var(--border-color);
  color: var(--accent, #4f8cff); font-size: 12px;
  padding: 3px 12px; border-radius: 4px; cursor: pointer;
  opacity: .85; transition: opacity .15s;
}
.escape-link:hover { opacity: 1; background: rgba(79,140,255,.12); }
/* ─── 壳级逃生口：插件面板浮动入口 ───
   常驻极小按钮位于左下角（状态栏上方）：右侧面板占 grid 第 4 列，
   左下角处于第 1-2 列区域（activitybar/sidebar 底部），focusMode 全屏对话
   也不覆盖——不再遮挡右侧对话输入区。半透明弱化，hover 全显。
   点击打开浮动插件面板（Fixed 560px 居中）。 */
.plugin-escape-btn {
  position: fixed; left: 6px; bottom: 26px; z-index: 300;
  width: 22px; height: 22px; border-radius: 5px;
  display: flex; align-items: center; justify-content: center;
  background: var(--bg-elevated, #2a2d36); color: var(--text-muted);
  border: 1px solid var(--border-color); cursor: pointer;
  opacity: .3; transition: opacity .15s;
}
.plugin-escape-btn:hover { opacity: 1; color: var(--accent, #4f8cff); }
.plugin-escape-overlay {
  position: fixed; inset: 0; z-index: 400;
  background: rgba(0,0,0,.45);
  display: flex; align-items: center; justify-content: center;
}
.plugin-escape-panel {
  width: 560px; max-width: 92vw; height: 70vh; max-height: 640px;
  background: var(--bg-primary); border: 1px solid var(--border-color);
  border-radius: 10px; box-shadow: 0 8px 40px rgba(0,0,0,.5);
  display: flex; flex-direction: column; overflow: hidden;
}
.plugin-escape-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 6px 10px; font-size: 12px; color: var(--text-muted);
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-elevated, #262932);
}
.plugin-escape-close {
  border: none; background: none; color: var(--text-muted);
  cursor: pointer; font-size: 13px; padding: 2px 6px; border-radius: 4px;
}
.plugin-escape-close:hover { background: rgba(255,255,255,.08); color: #fff; }
.plugin-escape-body { flex: 1; overflow: auto; }
.plugin-escape-body .plugin-panel { height: 100%; border: none; }
</style>
