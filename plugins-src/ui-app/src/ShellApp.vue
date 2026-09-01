<template>
  <div class="app-root" :class="{ 'panel-only': panelMode }" :style="gridStyle">
    <!-- ═══ chat 优先薄壳（2026-08 重构）：壳 = 纯几何骨架容器 ═══
         对齐 AppFrame 三列：sidebar | conversation(主) | details(editor)。
         编辑器为「辅助/details 列」：默认折叠（width:0 不占空间），点文件树按需打开，
         且永远保持挂载（绝不 unmount，规避 CM6/终端 WS 重挂断连坑）。
         每个区域一个单槽位挂载点：owner 为空 → 空态提示（该区域插件未装配）。 -->

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

    <!-- sidebar 槽位（single）：左侧文件/会话栏（chat 优先下的左列）
         ★ 折叠=收缩列宽（CSS 宽度切换，不 v-if）：保持 DOM 不卸载插件 UI（历史 bug:
            v-if 隐藏会销毁宿主 div，重新显示时 useSingleSlot 判定 owner 未变跳过重渲染 →
            面板空白需整页刷新）。 -->
    <div v-if="!panelMode && !slots.sidebar.owner.value" class="slot-empty plugin-area-sidebar">
      <span>侧栏未装配（ui-sidebar）</span><button class="escape-link" @click="pluginsOpen = true">打开插件面板</button>
    </div>
    <div v-else-if="!panelMode" :ref="slots.sidebar.hostRef"
         class="plugin-slot-host plugin-area-sidebar"></div>

    <!-- ★ main 区（col 3）：对话 / 编辑器 用 tab 切换（chat 优先薄壳）
         · 对话与编辑器两者常驻挂载（壳 v-show 切换，绝不 unmount）→ CM6/终端 WS 不重连，
           二者互不影响。
         · 激活 tab 由 state.panels.editorOpen 决定（false=对话 tab，true=编辑器 tab）。
         · 点文件树 openEditor → editorOpen=true → 切到编辑器 tab；对话 tab 可手动切回。 -->
    <div class="main-area" :class="{ 'panel-only': panelMode }">
      <!-- tab 栏（工具栏：对话 / 编辑器 / 市场） -->
      <div class="main-tabs" v-if="!panelMode">
        <button class="main-tab" :class="{ active: mainView === 'conversation' }"
                @click="layout.setMainView('conversation')">对话</button>
        <button class="main-tab" :class="{ active: mainView === 'editor' }"
                @click="layout.setMainView('editor')">编辑器<span class="main-tab-close" title="关闭" @click.stop="layout.closeEditor()">×</span></button>
        <!-- ★ 市场 tab（2026-09）：点击活动栏「市场」打开；× 关闭后回对话主视图 -->
        <button v-if="state.marketTabOpen" class="main-tab" :class="{ active: mainView === 'market' }"
                @click="layout.setMainView('market')">市场<span class="main-tab-close" title="关闭" @click.stop="closeMarketTab()">×</span></button>
      </div>

      <!-- conversation 宿主（单槽，常驻挂载；v-show 按 tab 切换） -->
      <div v-if="(state.rightPanelVisible || panelMode) && !slots.conversation.owner.value"
           v-show="mainView === 'conversation'" class="slot-empty conversation-container"
           :class="{ 'panel-only': panelMode }"><span>对话面板未装配（ui-right-panel）</span><button class="escape-link" @click="pluginsOpen = true">打开插件面板</button></div>
      <div v-else-if="(state.rightPanelVisible || panelMode)" v-show="mainView === 'conversation'"
           :ref="slots.conversation.hostRef" class="plugin-slot-host conversation-container"
           :class="{ 'panel-only': panelMode }"></div>

      <!-- editor 宿主（单槽，常驻挂载；v-show 按 tab 切换，从不 unmount） -->
      <div v-if="!panelMode && !slots.editor.owner.value"
           v-show="mainView === 'editor'" class="slot-empty editor-container">
        <span>编辑器未装配（ui-editor）</span><button class="escape-link" @click="pluginsOpen = true">打开插件面板</button></div>
      <div v-else-if="!panelMode" v-show="mainView === 'editor'"
           :ref="slots.editor.hostRef" class="plugin-slot-host editor-container"></div>

      <!-- ★ market 宿主（市场面板 tab 内容）：marketplace bundle 动态挂载；
           与对话/编辑器同为主区视图（v-show 切换，不占用槽），bundle 未就绪自动重试 -->
      <div v-if="!panelMode && state.marketTabOpen" v-show="mainView === 'market'"
           ref="marketHost" class="plugin-slot-host market-container"></div>
    </div>

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
       常驻壳层，即使全部区域插件都被停用，也能重新打开插件面板恢复装配。 -->
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
import { onMounted, onUnmounted, ref, computed, nextTick, watch } from 'vue'
import { useSingleSlot, boot, startPolling, stopPolling, loadAssemblyFile } from './plugin-runtime.js'
import { state, sidebarWidth, layout } from './ui-state.js'
import { initAppGlobals, cleanupAppGlobals, desktopPrefetch, loadWsList, closeMarketTab } from './app-actions.js'
import PluginPanel from './components/PluginPanel.vue'

// ★ 桌面端面板独立模式：desktopbridge 注入 window.__DESKTOP_PANEL_MODE__，
//   此时只渲染右侧对话面板占满全屏，隐藏 IDE 其他区域。
const panelMode = typeof window !== 'undefined' && window.__DESKTOP_PANEL_MODE__ === true
const pluginsOpen = ref(false) // 壳级逃生口：插件面板浮动层开关

const slots = {
  titlebar: useSingleSlot('titlebar'),
  activitybar: useSingleSlot('activitybar'),
  sidebar: useSingleSlot('sidebar'),
  editor: useSingleSlot('editor'),
  conversation: useSingleSlot('conversation'),
  statusbar: useSingleSlot('statusbar'),
  modals: useSingleSlot('modals'),
}
for (const s of Object.values(slots)) s.init()

// ─── ★ host.main 子槽声明（spec §5.2 「声明即认领」，一个子槽一位认领者）───
//   对齐 AppFrame：槽位名 = manifest dsh.ui.slot = 运行时 registerSlot slotId。
//   壳只声明几何骨架 + 具名子槽；内容由含 dsh.ui 段的区域插件 registerSlot 认领。
//   kind：single=替换型（面板切换，一位占用者）；list=叠加型（多位占用者同时渲染）。
const hostMainChildren = {
  titlebar:    { kind: 'single', scope: 'root' },
  activitybar: { kind: 'single', scope: 'root' },
  sidebar:     { kind: 'single', scope: 'root' },
  conversation:{ kind: 'single', scope: 'root' },   // chat 优先主视图（ui-right-panel）
  editor:      { kind: 'single', scope: 'root' },   // 辅助/details 列，默认折叠（ui-editor）
  statusbar:   { kind: 'single', scope: 'root' },
  modals:      { kind: 'single', scope: 'root' },   // fixed 浮层，不占 grid 格
  overlay:     { kind: 'list',   scope: 'root' },   // 浮动层（toast/approval/badge，叠加）
  // ★ git-api / marketplace：dsh.ui.slot 已对齐为 'sidebar'（真实宿主，spec §5.2 无
  //   'activitybar-panel' 子槽）。二者并非 registerSlot 单槽占用者（kind='list'，client.js
  //   仅注入 window.GitPanel / window.MarketplacePanel bundle，不 registerSlot），而是经
  //   activitybar 活动图标激活、由 Sidebar.vue 在 activeActivity==='source'/'marketplace'
  //   时动态挂载到 sidebar 面板区。故其 slot 声明指向承载它们的宿主子槽 'sidebar'，
  //   消除「伪声明 activitybar-panel」：manifest slot == 真实宿主槽名（sidebar）。
  //   （如后续要求 registerSlot 认领，需在 client.js 增加 registerSlot 并把 slot 改为
  //     对应壳级子槽，并在本表联动。）
}

// ─── ★ chat 优先薄壳几何：conversation 主列 + editor 辅助/details 列 ───
// grid 列：activitybar(48) | sidebar | conversation(minmax 0 1fr 主) | details(editor)
// · sidebar 列宽：focusMode 或折叠 → 0；否则 sidebarWidth（280）
// · details(editor) 列宽：focusMode 或 editorOpen=false → 0；否则 editorWidth
//   ★ 编辑器折叠=列宽收缩（CSS），宿主 DOM 不卸载（CM6/终端 WS 保持挂载）。
// ★ 主视图 tab（对话 ⇄ 编辑器 ⇄ 市场）：state.panels.mainTab 是单一事实源。
//   三者常驻挂载（模板 v-show 切换），互不影响。
const mainView = computed(() => state.panels.mainTab)

// ── ★ 市场面板（主区 tab）动态挂载：marketplace 插件 bundle → window.MarketplacePanel ──
// 2026-09：市场面板从侧边栏迁至主内容区 tab；跨 bundle 挂载（与 GitPanel 同模式）。
const marketHost = ref(null)
let marketUnmount = null
let marketRetryTimer = null

function mountMarketPanel() {
  const el = marketHost.value
  if (!el) return
  el.innerHTML = ''
  const mod = window.MarketplacePanel
  if (mod && typeof mod.mount === 'function') {
    try {
      marketUnmount = mod.mount(el)
      return
    } catch (e) {
      console.warn('[shell] 市场面板挂载失败', e)
      el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">挂载失败: ' + (e && e.message || e) + '</div>'
      return
    }
  }
  if (marketRetryTimer) return
  let tries = 0
  marketRetryTimer = setInterval(() => {
    tries++
    if (window.MarketplacePanel) {
      clearInterval(marketRetryTimer); marketRetryTimer = null
      mountMarketPanel()
      return
    }
    if (tries >= 8) {
      clearInterval(marketRetryTimer); marketRetryTimer = null
      el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">市场面板未就绪（marketplace 插件未启用）</div>'
    }
  }, 800)
  el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">市场面板加载中...</div>'
}

function unmountMarketPanel() {
  if (marketRetryTimer) { clearInterval(marketRetryTimer); marketRetryTimer = null }
  if (marketUnmount) { try { marketUnmount() } catch (e) {} marketUnmount = null }
}

// 市场 tab 激活/离开时挂载/卸载面板（激活时 may 尚未 mount 完成，nextTick 兜底）
watch(mainView, (v) => {
  if (v === 'market' && state.marketTabOpen) nextTick(mountMarketPanel)
  else unmountMarketPanel()
})

const gridStyle = computed(() => {
  if (panelMode) return { gridTemplateColumns: '1fr', gridTemplateRows: '1fr' }
  // · sidebar 列宽：折叠（sidebarVisible=false）→ 0；否则 sidebarWidth（280）
  // · main 列（col 3）：对话/编辑器 tab 区，占主导（无独立 editor 列）。
  //   ★ 编辑器不再是独立 details 列，而是 main 区内与对话 tab 切换（见 main-area）。
  const sidebarW = state.sidebarVisible ? (sidebarWidth.value + 'px') : '0px'
  return {
    gridTemplateColumns: `48px ${sidebarW} minmax(0, 1fr)`,
    gridTemplateRows: '30px 1fr 22px',
  }
})

onMounted(async () => {
  for (const s of Object.values(slots)) s.start()
  desktopPrefetch()
  initAppGlobals()
  loadWsList()

  // ★ 外部兼容（spec M4）：boot()（plugin-runtime.js）为薄壳装载【单入口，两源合并】：
  //   链路：自取 /api/ui-boot → 校验 __PAIRCODE_CORE 就绪 → 预取 immediately bundle
  //   → ① loadClientHalvesFromManifest(entries) 按图装配 dsh.ui 区域包 client 半；
  //   → ② syncClientHalves(await api.listPlugins()) 同时装载无 dsh.ui 段的旧直载包
  //      （agent-teams/ui-quick-exec/ui-statusbar-conn → titlebar-right/statusbar-items），
  //      两类并存、首屏即刻恢复（spec §7 向后兼容 —— 非 PluginPanel 面板延迟同步）。
  //   ★ PluginPanel.vue 的 syncClientHalves 调用保留为面板路径的兜底同步（不删）。
  try {
    // ★ 先合并磁盘装配文件（.pair/ui-assembly.json，用户可编辑的逃生通道）——
    //   再装载 client 半（注册槽位时 getSlotOwner/getSlotUIList 读到合并后的状态）。
    await loadAssemblyFile()
    await boot()
    startPolling() // 事件轮询全局启动（host→client 事件分发；幂等）
  } catch (e) {
    console.warn('[shell] client 半装载失败（/api/ui-boot）', e)
  }
})

onUnmounted(() => {
  for (const s of Object.values(slots)) s.stop()
  stopPolling()
  cleanupAppGlobals()
  unmountMarketPanel()
})
</script>

<style scoped>
.app-root {
  display: grid;
  /* ★ chat 优先薄壳（替换原 4 列 IDE 网格）：conversation 为 minmax(0,1fr) 主列，
     editor 为 details 辅助列（--editor-w），折叠=0px 不占空间但 DOM 保持挂载。
     gridStyle computed 会覆盖此默认值（聚焦/折叠时动态调整列宽）。 */
  grid-template-columns: 48px var(--sidebar-w, 280px) minmax(0, 1fr);
  grid-template-rows: 30px 1fr 22px;
  width: 100%; height: 100%;
  background: var(--bg-primary);
  color: var(--text-primary);
  overflow: hidden;
  font-family: var(--font-ui);
}
/* ★ 桌面端面板独立模式：只渲染右侧对话面板，占满整个窗口 */
.app-root.panel-only {
  grid-template-columns: 1fr;
  grid-template-rows: 1fr;
}
.app-root.panel-only .main-area {
  grid-column: 1; grid-row: 1;
  width: 100% !important;
  height: 100%;
}
.app-root.panel-only .main-tabs { display: none; }
/* 整区替换槽位（single）宿主：与内置区域同 grid 位置/尺寸 */
.plugin-area-titlebar { grid-column: 1 / -1; grid-row: 1; height: 30px; }
.plugin-area-activitybar { grid-column: 1; grid-row: 2; width: 48px; }
.plugin-area-sidebar { grid-column: 2; grid-row: 2; height: 100%; overflow: hidden; }
/* ★ main 区（col 3）：对话 / 编辑器 tab 切换（chat 优先薄壳主视图） */
.main-area {
  grid-column: 3; grid-row: 2;
  display: flex; flex-direction: column; min-width: 0; overflow: hidden; position: relative;
}
/* tab 栏：对话 / 编辑器 */
.main-tabs {
  display: flex; flex-shrink: 0; height: 30px;
  background: var(--bg-secondary); border-bottom: 1px solid var(--border-color);
}
.main-tab {
  /* ★ 不用均分（flex:1 会造成 50/50 平分、视觉难看）：宽度随内容自适应，左对齐 */
  flex: 0 0 auto; min-width: 0; border: none; background: none; cursor: pointer;
  color: var(--text-muted); font-size: 12px; font-weight: 600;
  padding: 0 18px; border-bottom: 2px solid transparent;
  transition: color .15s, background .15s;
}
.main-tab:hover { color: var(--text-primary); background: var(--bg-hover); }
.main-tab.active { color: var(--text-primary); border-bottom-color: var(--accent); background: var(--bg-active); }
/* conversation（对话）宿主：常驻挂载，v-show 切换；填满 main 区（tab 栏下方） */
.conversation-container {
  flex: 1; min-width: 0; min-height: 0;
  display: flex; flex-direction: row; overflow: hidden; position: relative;
}
/* editor（编辑器）宿主：常驻挂载，v-show 切换；填满 main 区（tab 栏下方），永不 unmount */
.editor-container {
  flex: 1; min-width: 0; min-height: 0;
  display: flex; flex-direction: column; overflow: hidden;
}
/* market（市场面板）宿主：主区第三视图，v-show 切换；bundle 动态挂载 */
.market-container {
  flex: 1; min-width: 0; min-height: 0;
  display: flex; flex-direction: column; overflow: hidden;
}
/* 主区 tab 内嵌关闭按钮（编辑器 / 市场）× */
.main-tab-close {
  display: inline-flex; align-items: center; justify-content: center;
  margin-left: 6px; font-size: 13px; line-height: 1;
  width: 16px; height: 16px; border-radius: 3px; opacity: 0.55;
}
.main-tab-close:hover { opacity: 1; background: var(--bg-hover); color: var(--text-primary); }
.app-statusbar-host { grid-column: 1 / -1; grid-row: 3; z-index: 30; height: 22px; }
.plugin-slot-host { height: 100%; overflow: hidden; }
/* ★ 插件渲染的子元素必须撑满宿主（bundle 根 auto 宽度不随宿主 grid 拉伸）。
   以 <conversation> 主列为例：宿主占列 3 → 子元素撑满，避免右侧空余。 */
.plugin-slot-host.conversation-container > * { width: 100%; min-width: 0; }
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
   常驻极小按钮位于左下角（状态栏上方）；半透明弱化，hover 全显。
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
