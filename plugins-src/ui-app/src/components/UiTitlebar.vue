<template>
  <!-- ═══ 顶栏（设计稿 shell-midnight th206 / th199）══════════════════════════
       整条 grid-row:1 全宽 40px，bg=surface，pad=space.sm(8)：
         th182 [icon accent 18] + 「PairCode」(fg sm 600)
         th198 [「对话」(选中) 「编辑器」「设计」「市场」]  ← 导航胶囊组，h=24 r=full
         左段 = [品牌] │ 「帮助」菜单；右段 = 插件叠加槽位 titlebar-right
       ★ 2026-09-25 按设计稿重建：移除旧 main-tabs（8 个 tab + 视图/并排工具），
         导航胶囊收敛为设计稿的 4 项；其余视图（工具集/画板/3D/音乐…）改由活动栏图标进入。
       ★ 2026-09-25 用户指令：右上那组内置图标（搜索/账户/帮助/设置）不要了 ——
         原 th205 的 4 个 muted 16 图标整块移除。
       ★ 2026-09-24 用户指令（本轮）：
         ① 删除「项目名/分支名」胶囊（原设计稿 th188「gou-ide」+「master」）—— 与底栏
            StatusBar 的「folder 工作区名 + link 分支名」完全重复；工作区/分支属**环境信息**，
            保留在底栏（IDE 惯例，th211 带 folder/link 图标语义）更合适，顶栏腾出的
            空间留给帮助入口与插件入口。
          ② 把「帮助」菜单放回顶栏：重新挂载 MenuBar.vue（menus 仅含「帮助」一项）。
        ★ 2026-09-25（本轮用户指令）：帮助菜单**由右段移到左段**（紧跟品牌、以 1px 竖直
          分隔线隔开）—— 见 .tb-left / .tb-left-sep 样式注释与 .tb-right-wrap 注释。 -->
  <div class="titlebar">
    <!-- ── 左：logo + 品牌 + 帮助菜单 ──────────────────────────────────────────
         ★ 2026-09-25（本轮用户指令）：帮助菜单由右段移到此处（品牌之后）。
         IDE 惯例：菜单栏贴着 logo 居于最左；右段专供运行时「状态/动作」入口
         （插件叠加槽位 titlebar-right），两侧语义不再混杂。
         ▸ .tb-nav 是绝对居中（脱离 flex 流），左段加宽不影响导航居中。 -->
    <div class="tb-left">
      <img :src="logoUrl" class="tb-logo" alt="PairCode" />
      <span class="tb-brand">PairCode</span>
      <span class="tb-left-sep" aria-hidden="true"></span>
      <MenuBar />
    </div>

    <!-- ── 中右：导航胶囊（内置对话/编辑器/市场 + 插件注册视图，点击切主区视图）── -->
    <nav class="tb-nav">
      <button v-for="n in navs" :key="n.key" class="tb-nav-pill"
              :class="{ active: isNavActive(n) }" :title="n.title"
              @click="onNav(n)">{{ n.label }}</button>
    </nav>

    <!-- ── 右：插件叠加槽位 titlebar-right（右对齐）────────────────────────────
         ★ 2026-09-25（用户纠正）：该槽位语义就是「标题栏右侧按钮区」，
         今日一度被迁到 StatusBar 承载 → 用户两次指出「还在状态栏」，故在此接回宿主。
         占用者：agent-teams「团队」/ autopilot「自主模式 N 轮」/ ui-quick-exec「快速执行」。
         ▸ 内置图标（搜索/账户/设置）仍按用户早前指令保持移除；既有路径：
           搜索→活动栏「搜索」/Ctrl+K；账户与团队→活动栏「团队」；
           设置→Ctrl+,（ui-state.showSettings）。动作函数保留在 <script> 中
           （命令面板/快捷键仍复用），仅不在此出图。
         ▸ 2026-09-24：「帮助」曾回顶栏右段（与插件槽位同容器）；2026-09-25 按用户
           指令**移到左段**，本容器回归单一职责 = 插件入口。
           ▸ 容器内无插件占用时为空容器，无害：margin-left:auto 仍把右段推到最右。 -->
    <div class="tb-right-wrap">
      <div ref="tbRightEl" class="tb-right plugin-slot-host plugin-slot-titlebar"></div>
    </div>
  </div>
</template>

<script setup>
// UiTitlebar — 顶栏（ui-titlebar 插件承载），2026-09-25 按设计稿 shell-midnight 重建。
// 旧实现 = logo + MenuBar(帮助菜单) + 居中工作区名 + 右侧叠加槽位（30px 高）；
// 现形态 = 左段（品牌 + 帮助菜单）· 居中 4 导航胶囊 · 右段（插件叠加槽位），40px 高。
//   ★ 2026-09-24（用户指令）：th188 项目/分支胶囊（与底栏重复）与 th205 右上 4 图标
//     均已移除；「帮助」菜单（MenuBar）回归顶栏。
//   ★ 2026-09-25（本轮用户指令）：帮助菜单由右段移到左段（品牌之后）。
//   ★ 2026-09-25（本轮用户指令）：导航胶囊改为**按插件自动注册** —— 除内置
//     「对话/编辑器/市场」三项外，每一项都来自插件的 ui.registerView 注册表
//     （clientViews），插件装载/卸载 → 胶囊自动增删（详见下方 navs）。
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { state, layout, showSettings } from '../ui-state.js'
import MenuBar from './MenuBar.vue'
import { clientViews, setViewMount, mountListSlot } from '../plugin-runtime.js'
import { switchActivity } from '../app-actions.js'
import api from '../api.js'
import logoUrl from '../assets/logo.svg'

// titlebar-right 槽位宿主（右段）：插件入口渲染目标 + 退订句柄
const tbRightEl = ref(null)
let tbRightUnsub = null
let navSubUnsub = null   // 视图注册表订阅退订句柄（胶囊自动增删）

// ── 导航胶囊：内置 3 项 + 插件注册视图（动态，2026-09-25 用户指令）──────────
//   设计稿 th191/193/195/197 只给了 4 个胶囊（对话/编辑器/设计/市场），此前实现
//   把它们写死，并**单独猜名**认领「设计」一个插件视图（title/id 命中即算）。
//   问题：插件不止「设计」一个 —— 六域（画板/设计/3D 模型/音乐/角色/人声）与
//   autopilot 看板都注册了主区视图，却只有「设计」在标题栏有入口，其余只能去
//   主区 tab 栏/视图菜单里找，可发现性差且每加一个插件都要改壳代码。
//   现方案：胶囊 = 内置「对话/编辑器」+ registerView 注册视图（按 regions 筛）+ 内置「市场」，
//   数据源是 plugin-runtime 的 clientViews（跨 bundle 共享的同一数组）。
//     · 刷新：setViewMount(订阅) → 插件注册/卸载视图时 emitViewChanged → viewTick++；
//     · 顺序：注册表已按 order 排序（六域 10..60、autopilot 90、默认 100），
//       插件视图排在「编辑器」与「市场」之间，与设计稿四项的相对次序一致；
//     · ★ 入驻区域（regions，2026-09）：只收声明含 'titlebar' 的视图 —— 插件可用
//       regions:['mainTab'] 表示「只要主区内容、别占顶栏」（自带入口者如 autopilot
//       在其活动栏图标里自开视图）；**不声明时默认两项都进**，故已发布插件无感。
//       （'mainTab' = 主区内容出口，含渲染必要条件，故 plugin-runtime 侧缺失即补回。）
//     · 点击：layout.openViewTab(pluginName, id)（未打开则打开并激活，与主区内容
//       出口同源，状态存 plugin-runtime 的 viewOpen:* 持久化）。
const viewTick = ref(0)   // 注册表版本号：computed 依赖它才能在插件增删时重算

const pluginNavs = computed(() => {
  viewTick.value   // 显式读一次建立依赖（clientViews 是普通数组，本身非响应式）
  return (clientViews || [])
    // ★ 入驻区域过滤（regions）：只渲染声明含 'titlebar' 的视图。单插件视图自带
    //   入口（活动栏图标）时可声明 regions:['mainTab'] 主动让位；老注册表项
    //   （页内残留 / 未带 regions 字段）兜底显示，避免升级瞬间胶囊整体消失。
    .filter(v => (Array.isArray(v.regions) ? v.regions.includes('titlebar') : true))
    .map(v => ({
      key: layout.viewTabKey(v.pluginName, v.id),   // 'view:<插件名>:<视图 id>'
      label: v.title,
      title: '插件视图 · ' + v.pluginName,
      pluginName: v.pluginName,
      id: v.id,
    }))
})

const navs = computed(() => [
  { key: 'conversation', label: '对话', title: '对话主视图' },
  { key: 'editor', label: '编辑器', title: '代码编辑器' },
  ...pluginNavs.value,
  { key: 'market', label: '市场', title: '插件市场' },
])

const mainTab = computed(() => state.panels.mainTab)

function isNavActive(n) {
  // 插件视图的 key 就是 viewTabKey（'view:<插件名>:<id>'），与 mainTab 同一取值
  // 空间 → 内置项与插件项可统一比较（mainTab 是主视图单一事实源）。
  return mainTab.value === n.key
}

function onNav(n) {
  if (n.pluginName && n.id) {
    // 插件视图：未打开则打开并激活（activate 默认 true）。插件被卸载后该胶囊随
    // 注册表消失，故此处无需再判空；仍保留注册表项时 openViewTab 必能命中。
    layout.openViewTab(n.pluginName, n.id)
    return
  }
  if (n.key === 'market') { state.marketTabOpen = true; layout.setMainView('market'); return }
  layout.setMainView(n.key)
}

// ── 右侧图标动作 ──
function openSearch() { switchActivity('search') }
function openAccount() { window.$alert && window.$alert('账户与团队：多智能体协作面板由 agent-teams 插件提供，入口在活动栏「团队」。', '账户与团队') }
// ★ 迁移自 RightPanel.rp-header（该行已按设计删除）：新对话 / 会话列表 / 专注 / Debug 日志
async function newConversation() {
  try {
    const conv = await api.apiPost('/conversations', { title: '新对话', workspaceRoot: state.workspaceRoot })
    if (!conv || !conv.id) return
    state.conversations.unshift({ id: conv.id, title: conv.title || '新对话', msgCount: 0, createdAt: conv.createdAt, updatedAt: conv.updatedAt })
    if (!state.messagesByConv[conv.id]) state.messagesByConv[conv.id] = []
    state.currentConvId = conv.id
  } catch (e) { window.$toast && window.$toast('新建会话失败：' + (e && e.message ? e.message : e), 'error') }
}
function openSettings() { showSettings.value = true }
function showHelp() {
  window.$alert &&
    window.$alert('命令面板 Ctrl+K · 专注模式 Ctrl+Shift+L · 设置 Ctrl+, · 会话列表 Ctrl+Shift+C', '快捷键')
}

// ★ 2026-09-25（用户纠正）：接回 titlebar-right 槽位宿主。
//   该槽位语义 = 标题栏右侧按钮区，占用者：agent-teams「团队」/
//   autopilot「自主模式 N 轮」/ ui-quick-exec「快速执行」，落点 .tb-right-wrap 内。
//   mountListSlot 对宿主缺失容错（host 为 null 直接 return），且从全局 clientSlots 取全部
//   list 占用者渲染 —— 故**同一 slotId 只能有一处宿主**，与 StatusBar 不得并存（会重复渲染）。
onMounted(() => {
  tbRightUnsub = mountListSlot(tbRightEl, 'titlebar-right')
  // ★ 视图注册表订阅（registerView）：插件注册/卸载视图 → pluginNavs 重算。
  //   setViewMount 为多订阅者（ShellApp 的视图 tab 栏亦订阅同一事件），
  //   返回退订函数；此处无需初始回调 —— 首帧渲染时 computed 直接读注册表现状。
  navSubUnsub = setViewMount(() => { viewTick.value++ })
})
onUnmounted(() => {
  if (tbRightUnsub) { tbRightUnsub(); tbRightUnsub = null }
  if (navSubUnsub) { navSubUnsub(); navSubUnsub = null }
})
</script>

<style scoped>
/* ═══ 设计稿 th206：row 1440×40, bg=color.surface, pad=space.sm(8), align=center ═══ */
.titlebar {
  grid-column: 1 / -1; grid-row: 1;
  display: flex; align-items: center; height: 40px;
  padding: 0 8px; gap: 8px;
  position: relative;              /* ★ 承载 .tb-nav 的绝对居中（见 .tb-nav 注释） */
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  z-index: 100; overflow: visible;
  -webkit-app-region: drag;
}
.tb-left { display: flex; align-items: center; gap: 8px; flex: 0 0 auto; -webkit-app-region: no-drag; }
/* ★ 2026-09-25（用户指令：帮助菜单移到标题栏左侧）：
   品牌区与菜单区之间的 1px 竖直分隔（h=12，与 40px 顶栏比例协调，取最低调的
   --border-color）—— 划分「这是谁」(品牌) 与「能做什么」(菜单) 两个语义区，
   同时避免品牌文字与菜单按钮视觉上连成一串。 */
.tb-left-sep { width: 1px; height: 12px; background: var(--border-color); flex: 0 0 auto; }
.tb-logo { width: 18px; height: 18px; }
/* 设计稿 th181：字号 sm(12) + 600 + fg */
.tb-brand { font-size: 12px; font-weight: 600; color: var(--text-primary); }
/* ★ 2026-09-24：设计稿 th185/th187 的「项目名」「分支名」胶囊已按用户指令删除
   （与底栏 StatusBar 的 th207-th210 重复）—— 如需恢复，从 git 历史取回
   .tb-pill / .tb-pill-branch 两段样式即可。 */
/* ★ 2026-09-25 缺陷 5 收口 + 用户指令（移除右上图标）：
   顶栏导航胶囊用**绝对居中** —— 相对 .titlebar（已设 position:relative）以
   left:50% + translateX(-50%) 定位，与设计稿四胶囊 left 622/670/730/778（1440 画布居中区）一致。
   ▸ 早前左段（品牌 + 项目/分支胶囊）宽于右段导致流式居中偏右 46px；2026-09-24 删除
     项目/分支胶囊后左段只剩品牌，且绝对居中本就不受左右段宽度影响，问题彻底消除。 */
.tb-nav {
  display: flex; align-items: center; gap: 4px;
  position: absolute; left: 50%; transform: translateX(-50%);
  flex: 0 0 auto; -webkit-app-region: no-drag;
  /* ★ 2026-09-25（动态 tag）：胶囊数 = 内置 3 + 每个插件注册视图 1（插件可增删，
     六域 + autopilot 已达 7 个）→ 限宽 + 横向滚动兜底，避免绝对居中的胶囊组
     压住左段（品牌/帮助菜单）与右段（titlebar-right 插件槽位）。
     滚动条隐藏：顶栏内出现横向滚动条会撑高整条顶栏、视觉跳动。 */
  max-width: min(60vw, 760px);
  overflow-x: auto; overflow-y: hidden;
  scrollbar-width: none;
}
.tb-nav::-webkit-scrollbar { display: none; }
.tb-nav-pill {
  display: inline-flex; align-items: center;
  flex: 0 0 auto; white-space: nowrap;   /* 滚动容器内不被压缩、文字不换行 */
  height: 24px; padding: 0 10px; border-radius: 999px;
  border: none; background: none; cursor: pointer;
  font-size: 11px; font-weight: 600; color: var(--text-muted);
  transition: color .15s, background .15s;
}
.tb-nav-pill:hover { color: var(--text-primary); background: var(--bg-hover); }
/* 选中：surface-3 底 + accent 文字（设计稿 th191/th190） */
.tb-nav-pill.active { color: var(--accent); background: var(--bg-active); }
/* ── 右段：插件叠加槽位 titlebar-right（右对齐）──
   ★ 2026-09-25（用户纠正）：顶栏承载该槽位；th205 的 4 个内置图标
   （搜索/账户/设置）保持移除，此处只放**插件**注册的入口（帮助菜单已于
   2026-09-25 移往左段）。
   margin-left:auto 由 .tb-right-wrap 承担（.tb-nav 绝对居中、脱离 flex 流）——
   即便容器内无插件占用（空容器）也照常把右端锚定，不影响左段与导航。 */
/* 右段容器（插件入口）：margin-left:auto 推到最右。
   ★ .tb-nav 是绝对居中（脱离 flex 流），本容器内容宽度不影响导航居中。 */
.tb-right-wrap {
  margin-left: auto;
  display: flex; align-items: center; gap: 8px;
  flex: 0 0 auto; height: 100%;
  -webkit-app-region: no-drag;
}
.tb-right { display: flex; align-items: center; gap: 8px; flex: 0 0 auto; height: auto; }
/* 插件条目：mountListSlot 动态创建的 .plugin-slot-item 不带 scoped 属性 → 必须走 :deep */
.tb-right :deep(.plugin-slot-item) { display: inline-flex; align-items: center; gap: 4px; }
</style>
