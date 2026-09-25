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
    <!-- ★ 2026-09-25 对齐设计稿 shell-midnight th198：原 main-tabs（8 个 tab ＋
         「视图/并排对比」工具区）已整体移除 —— 导航收敛为顶栏的 4 个胶囊
         （UiTitlebar：对话/编辑器/设计/市场）；其余视图（工具集/画板/3D/音乐/
         角色/市场）经活动栏图标进入，不再占用顶栏。主区内容（.main-views）
         不受影响，仍按 state.panels.mainTab 切换。 -->
    <div class="main-area" :class="{ 'panel-only': panelMode }">
      <!-- tab 栏已提到顶栏（P4 合并），此处不再渲染 -->
      <!-- tab 栏（工具栏：对话 / 编辑器 / 市场） -->

      <!-- ★ 内容区：单栏（tab 互斥切换）或并排两栏（对话 + 当前视图）─────────
           split 时 = 一栏对话（可在左/右，换边按钮切换）+ 一栏当前激活视图；
           各 pane 常驻挂载（v-show 切换，绝不 unmount）→ CM6/终端 WS 不重连。 -->
      <div class="main-views"
           :class="{ split: splitActive, 'split-chat-right': state.panels.splitChatSide === 'right' }">
      <!-- conversation 宿主（单槽，常驻挂载；v-show 按 tab 切换；并排时常显侧栏） -->
      <div class="view-pane view-pane-chat" v-show="mainView === 'conversation' || splitActive">
        <div v-if="(state.rightPanelVisible || panelMode) && !slots.conversation.owner.value"
             class="slot-empty conversation-container"
             :class="{ 'panel-only': panelMode }"><span>对话面板未装配（ui-right-panel）</span><button class="escape-link" @click="pluginsOpen = true">打开插件面板</button></div>
        <div v-else-if="(state.rightPanelVisible || panelMode)"
             :ref="slots.conversation.hostRef" class="plugin-slot-host conversation-container"
             :class="{ 'panel-only': panelMode }"></div>
      </div>

      <!-- editor 宿主（单槽，常驻挂载；v-show 按 tab 切换，从不 unmount） -->
      <div v-if="!panelMode" class="view-pane" v-show="mainView === 'editor'">
        <div v-if="!slots.editor.owner.value" class="slot-empty editor-container">
          <span>编辑器未装配（ui-editor）</span><button class="escape-link" @click="pluginsOpen = true">打开插件面板</button></div>
        <div v-else :ref="slots.editor.hostRef" class="plugin-slot-host editor-container"></div>
      </div>

      <!-- ★ market 宿主（市场面板 tab 内容）：marketplace bundle 动态挂载；
           与对话/编辑器同为主区视图（v-show 切换，不占用槽），bundle 未就绪自动重试 -->
      <div v-if="!panelMode && state.marketTabOpen" class="view-pane" v-show="mainView === 'market'">
        <div ref="marketHost" class="plugin-slot-host market-container"></div>
      </div>

      <!-- ★ 工具集宿主（主区 tab）：ToolsetPanel 静态组件直接挂载（非 bundle）；
           v-if 控制 tab 打开时才挂载（首次打开加载数据），v-show 切换保持不销毁 -->
      <div v-if="!panelMode && state.toolsetsTabOpen" class="view-pane" v-show="mainView === 'toolsets'">
        <div class="plugin-slot-host toolsets-container"><ToolsetPanel /></div>
      </div>

      <!-- ★ 插件中间区域视图容器（registerView）：激活过的视图懒挂载（bundle.render →
           容器；挂上后保持，切 tab 不卸载 → 保住 3D 视角/滚动等状态），关闭 tab 才卸载 -->
      <div v-for="v in openViews" :key="v.key" class="view-pane view-pane-plugin"
           v-show="mainView === v.key" :ref="(el) => setViewHostEl(v.key, el)"></div>
      </div>
    </div>

    <!-- ═══ 右栏（grid col 4）：设计稿 th178 ═══
         运行统计(188) / 任务进度(152) / 上下文构成(116，水平分段条) / 底提示(72)。
         设计稿为常驻信息栏（宽 288），随主题令牌自动换肤。 -->
    <!-- ★ 右栏收纳（2026-09-25）：折叠=网格列宽 0（gridStyle railW），不 v-if —— 与
         sidebar/编辑器同一折叠哲学：保持挂载，专注模式进出右栏不重挂、卡片状态不丢。 -->
    <div v-if="!panelMode" class="right-rail-host">
      <StatsRail />
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
import { useSingleSlot, boot, startPolling, stopPolling, loadAssemblyFile,
         setViewMount, isViewOpen, getUIFor } from './plugin-runtime.js'
import { state, sidebarWidth, layout, syncFocusTarget } from './ui-state.js'
import { initAppGlobals, cleanupAppGlobals, desktopPrefetch, loadWsList, closeMarketTab, closeToolsetsTab } from './app-actions.js'
import PluginPanel from './components/PluginPanel.vue'
import ToolsetPanel from './components/ToolsetPanel.vue'
import StatsRail from './components/StatsRail.vue'

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

// ─── ★ 薄壳几何：activitybar | sidebar | main（主视图 tab 区）| rail（右栏）───
// grid 列：activitybar(48) | sidebar | main(minmax(0, 1fr)) | rail(288) —— 四列，见下方 gridStyle
// · sidebar 列宽：折叠（state.sidebarVisible=false）→ 0px；否则 sidebarWidth（默认 280）
//   ★ 专注模式不进列宽公式：Ctrl+K 由 ui-state.setFocusMode 按 FOCUS_PANELS 登记表
//     直接收起 sidebarVisible / statsRailVisible 等（退出还原用户原值），故此处只认可见性标志。
//   ★ 编辑器不再独立列：在 main 区内与对话 tab 切换（v-show，宿主 DOM 不卸载）。
// ★ 主视图 tab（对话 ⇄ 编辑器 ⇄ 市场 ⇄ 工具集 ⇄ 插件视图）：state.panels.mainTab 是单一事实源。
//   各视图常驻挂载（模板 v-show 切换），互不影响。
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

// ─── ★ 插件中间区域视图（ui.registerView / clientViews，2026-09）──────────
//   视图 tab 与内置 4 视图同级（对话/编辑器/市场/工具集）：
//   · 列表来源 = plugin-runtime 的 clientViews（共享注册表，跨 bundle 副本一致）；
//   · 打开状态 = isViewOpen()（localStorage viewOpen:* 优先，否则注册时的 open 默认）；
//   · 渲染 = 视图 render(el, ui) → 容器；**懒挂载 + 保持**（激活过就挂上，切 tab 不
//     卸载 —— 保住 3D 视角/滚动位置等），关闭 tab 或插件卸载才 cleanup；
//   · 与对话并排 = layout.toggleSplit()（状态在 ui-state，本组件只渲染几何）。
const views = ref([])          // [{ key, id, pluginName, title, icon, open, render }]
const viewMenuOpen = ref(false)
const viewHostEls = new Map()  // key → 容器 DOM
const viewMounts = new Map()   // key → cleanup（render 返回值）
let viewUnsub = null

// refreshViews 视图表变化（插件装载/卸载、用户开关 tab）→ 重建列表 + 清理已关闭项。
function refreshViews(list) {
  const next = (list || []).map(v => ({
    key: 'view:' + v.pluginName + ':' + v.id,
    id: v.id,
    pluginName: v.pluginName,
    title: v.title,
    icon: v.icon,
    // ★ 入驻区域（regions，2026-09）：'mainTab' 才在主区建内容容器；
    //   老注册表项无该字段时兜底两项都算（与 plugin-runtime 的默认一致）。
    regions: Array.isArray(v.regions) ? v.regions : ['titlebar', 'mainTab'],
    render: v.render,
    open: isViewOpen(v),
  }))
  views.value = next
  const live = new Set(next.filter(v => v.open).map(v => v.key))
  // 已关闭的 tab / 已卸载插件的视图 → 卸载挂载（防残留 DOM 与轮询）
  for (const [key, cleanup] of [...viewMounts]) {
    if (live.has(key)) continue
    try { cleanup() } catch (e) { console.warn('[shell] 视图卸载失败', key, e) }
    viewMounts.delete(key)
    viewHostEls.delete(key)
  }
  // 兜底：正激活的视图 tab 消失（插件被卸载）→ 回对话主视图
  if (String(mainView.value).startsWith('view:') && !live.has(mainView.value)) {
    state.panels.mainTab = 'conversation'
    // ★ 专注态内同步「退出专注」还原目标：避免退出时还原到已卸载视图（空白）。
    //   syncFocusTarget 由 ui-state.js 导出（非专注态为无操作）。
    syncFocusTarget('panels.mainTab')
  }
}

// openViews 建立「主区内容容器（pane）」的视图清单：① 已打开 ② 声明含 mainTab 区域。
//   二者缺一都不建容器 —— 未打开 = 不该占主区；无 mainTab = 无内容出口。
//   （plugin-runtime 侧对缺失 mainTab 会自动补回 + 告警，故后者实为防御性过滤。）
const openViews = computed(() => views.value.filter(v => v.open && v.regions.includes('mainTab')))
// 并排开关可用性：对话本身是当前视图时无意义（会变成两栏对话）
const canSplit = computed(() => mainView.value !== 'conversation' || state.panels.splitView)
const splitActive = computed(() => layout.isSplitActive())

function setViewHostEl(key, el) {
  if (el) viewHostEls.set(key, el)
  else viewHostEls.delete(key)
}

// mountViewIfNeeded 懒挂载（已挂载则幂等跳过；render 返回值作 cleanup）。
function mountViewIfNeeded(v) {
  if (!v || viewMounts.has(v.key)) return
  const el = viewHostEls.get(v.key)
  if (!el) return
  if (typeof v.render !== 'function') {
    el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">视图「' +
      v.title + '」未提供 render（插件 client 半声明 registerView 时需给 render）</div>'
    return
  }
  try {
    const ret = v.render(el, getUIFor(v.pluginName))
    viewMounts.set(v.key, typeof ret === 'function' ? ret : () => {})
  } catch (e) {
    console.warn('[shell] 视图挂载失败', v.key, e)
    el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">视图「' +
      v.title + '」挂载失败: ' + ((e && e.message) || e) + '</div>'
  }
}

// 视图菜单勾选：打开（后台 tab，不抢主视图）/ 关闭
function onToggleView(v, checked) {
  if (checked) layout.openViewTab(v.pluginName, v.id, { activate: false })
  else layout.closeViewTab(v.pluginName, v.id)
}
// 并排：切换对话所在侧（左 ⇄ 右）
function swapSplitSide() {
  layout.setSplitChatSide(state.panels.splitChatSide === 'left' ? 'right' : 'left')
}

// 市场 tab 激活/离开时挂载/卸载面板（激活时 may 尚未 mount 完成，nextTick 兜底）
watch(mainView, (v) => {
  if (v === 'market' && state.marketTabOpen) nextTick(mountMarketPanel)
  else unmountMarketPanel()
  // ★ 插件视图：激活时懒挂载（挂上后保持 —— 切 tab 不卸载）
  if (String(v).startsWith('view:')) {
    const target = openViews.value.find(x => x.key === v)
    if (target) nextTick(() => mountViewIfNeeded(target))
  }
})

const gridStyle = computed(() => {
  if (panelMode) return { gridTemplateColumns: '1fr', gridTemplateRows: '1fr' }
  // ★★★ 网格登记规则（新增网格列/常驻区域必须执行，2026-09-25 起）★★★
  //   1. 在 ui-state.js 的 FOCUS_PANELS 登记一行（fold = 专注态取值）——
  //      专注模式（Ctrl+K）的收起/还原由该表驱动，不登记 = 专注不收；
  //   2. 在本 computed 中登记列宽公式（可见性开关 → '0px'）；
  //   3. 跑收纳探针：node scripts/source-update/focus-probe.mjs（更新流程部署后自动跑，告警不阻断）。
  //   背景：右栏 StatsRail（v1.6.7 引入）曾未登记专注收纳（4fa3e367 修复）。
  // · sidebar 列宽：折叠（sidebarVisible=false）→ 0；否则 sidebarWidth（280）
  // · main 列（col 3）：对话/编辑器 tab 区，占主导（无独立 editor 列）。
  //   ★ 编辑器不再是独立 details 列，而是 main 区内与对话 tab 切换（见 main-area）。
  const sidebarW = state.sidebarVisible ? (sidebarWidth.value + 'px') : '0px'
  // ★ 2026-09-25 对齐设计稿 shell-midnight（th178）：新增第 4 列 = 右栏 288px。
  //   设计稿四列 = 活动栏 48 ｜ 会话 264 ｜ 主区 840 ｜ 右栏 288（1440 总宽）。
  const railW = state.statsRailVisible === false ? '0px' : '288px'
  return {
    gridTemplateColumns: `48px ${sidebarW} minmax(0, 1fr) ${railW}`,
    //   注意此处是内联 style，优先级高于 <style> 里的 .app-root 规则 —— 改行高必须改这里，
    //   否则 CSS 改了也不生效（曾实测 gridTemplateRows 仍为 30px）。
    // ★ 2026-09-25 对齐设计稿 shell-midnight：底栏 28px（th219 row h=28）。
    gridTemplateRows: '40px 1fr 28px',
  }
})

onMounted(async () => {
  for (const s of Object.values(slots)) s.start()
  // ★ 订阅视图注册表（registerView）：tab 栏 + 懒挂载随插件注册/卸载实时更新
  viewUnsub = setViewMount(refreshViews)
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
  if (viewUnsub) { viewUnsub(); viewUnsub = null }
  for (const [, cleanup] of viewMounts) { try { cleanup() } catch (e) {} }
  viewMounts.clear()
  viewHostEls.clear()
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
  /* ★ 2026-09-25 对齐设计稿：活动栏 48 ｜ 会话 264 ｜ 主区 1fr ｜ 右栏 288。 */
  grid-template-columns: 48px var(--sidebar-w, 264px) minmax(0, 1fr) var(--rail-w, 288px);
  /* ★ 2026-09-25：顶栏为整条（titlebar 跨全部列，见 UiTitlebar）；原 main-tabs 已移除。 */
  /* ★ 2026-09-25 对齐设计稿：底栏 28px（th219）；顶栏 titlebar 跨全部列（整条 40px）。 */
  grid-template-rows: 40px 1fr 28px;
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
/* 整区替换槽位（single）宿主：与内置区域同 grid 位置/尺寸
   ★ 2026-09-25：设计稿顶栏是【整条】(th206 w=1440)，故 titlebar 跨全部列；
   main-tabs 改为浮在顶栏行右侧的导航胶囊组（见下方 justify-self:end）。 */
.plugin-area-titlebar { grid-column: 1 / -1; grid-row: 1; height: 40px; }
.slot-empty.plugin-area-titlebar { grid-column: 1 / -1; grid-row: 1; height: 40px; }
.plugin-area-activitybar { grid-column: 1; grid-row: 2; width: 48px; }
.plugin-area-sidebar { grid-column: 2; grid-row: 2; height: 100%; overflow: hidden; }
/* ★ 右栏（设计稿第 4 列）：288px 常驻信息栏 */
.right-rail-host { grid-column: 4; grid-row: 2; min-width: 0; height: 100%; overflow: hidden; }
/* ★ main 区（col 3）：对话 / 编辑器 tab 切换（chat 优先薄壳主视图） */
.main-area {
  grid-column: 3; grid-row: 2;
  display: flex; flex-direction: column; min-width: 0; overflow: hidden; position: relative;
}
/* tab 栏：对话 / 编辑器 */
/* ★ 内容区（2026-09）：单栏（tab 互斥切换）或并排两栏（对话 + 当前视图） */
.main-views {
  flex: 1; min-width: 0; min-height: 0;
  display: flex; flex-direction: column; overflow: hidden; position: relative;
}
/* 并排：横排两栏；split-chat-right 用 row-reverse 把对话栏换到右侧
   （对话 pane 始终是 DOM 首个子元素：换边只改方向，不动挂载顺序） */
.main-views.split { flex-direction: row; }
.main-views.split.split-chat-right { flex-direction: row-reverse; }
.view-pane {
  flex: 1 1 auto; min-width: 0; min-height: 0;
  display: flex; flex-direction: column; overflow: hidden; position: relative;
}
/* 并排时对话栏固定比例（当前视图栏占剩余空间） */
.main-views.split .view-pane-chat {
  flex: 0 0 var(--split-chat-w, 42%);
  border-right: 1px solid var(--border-color);
}
.main-views.split.split-chat-right .view-pane-chat {
  border-right: none; border-left: 1px solid var(--border-color);
}
/* 插件中间区域视图容器（registerView）：bundle 挂载点，撑满所在栏 */
.view-pane-plugin { background: var(--bg-primary); }
/* tab 栏右侧工具区（视图列表 + 并排开关） */
/* 视图列表浮层（勾选 = tab 打开；后台打开，不抢对话主视图） */
/* 菜单外点关闭背板（仅菜单打开时存在，点一下就关） */
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
/* ★ 2026-09-25 对齐设计稿 th219（row h=28）。 */
.app-statusbar-host { grid-column: 1 / -1; grid-row: 3; z-index: 30; height: 28px; }
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
  color: var(--accent); font-size: 12px;
  padding: 3px 12px; border-radius: 4px; cursor: pointer;
  opacity: .85; transition: opacity .15s;
}
.escape-link:hover { opacity: 1; background: var(--color-accent-bg); }
/* ─── 壳级逃生口：插件面板浮动入口 ───
   常驻极小按钮位于左下角（状态栏上方）；半透明弱化，hover 全显。
   点击打开浮动插件面板（Fixed 560px 居中）。 */
.plugin-escape-btn {
  position: fixed; left: 6px; bottom: 26px; z-index: 300;
  width: 22px; height: 22px; border-radius: 5px;
  display: flex; align-items: center; justify-content: center;
  background: var(--bg-elevated); color: var(--text-muted);
  border: 1px solid var(--border-color); cursor: pointer;
  opacity: .3; transition: opacity .15s;
}
.plugin-escape-btn:hover { opacity: 1; color: var(--accent); }
.plugin-escape-overlay {
  position: fixed; inset: 0; z-index: 400;
  background: var(--color-scrim);
  display: flex; align-items: center; justify-content: center;
}
.plugin-escape-panel {
  width: 560px; max-width: 92vw; height: 70vh; max-height: 640px;
  background: var(--bg-primary); border: 1px solid var(--border-color);
  border-radius: 10px; box-shadow: var(--shadow-lg);
  display: flex; flex-direction: column; overflow: hidden;
}
.plugin-escape-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 6px 10px; font-size: 12px; color: var(--text-muted);
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-elevated);
}
.plugin-escape-close {
  border: none; background: none; color: var(--text-muted);
  cursor: pointer; font-size: 13px; padding: 2px 6px; border-radius: 4px;
}
.plugin-escape-close:hover { background: var(--bg-hover); color: var(--text-primary); }
.plugin-escape-body { flex: 1; overflow: auto; }
.plugin-escape-body .plugin-panel { height: 100%; border: none; }
</style>
