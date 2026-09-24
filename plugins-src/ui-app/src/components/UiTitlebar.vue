<template>
  <!-- ═══ 顶栏（设计稿 shell-midnight th206 / th199）══════════════════════════
       整条 grid-row:1 全宽 40px，bg=surface，pad=space.sm(8)：
         th182 [icon accent 18] + 「PairCode」(fg sm 600)
         th198 [「对话」(选中) 「编辑器」「设计」「市场」]  ← 4 个导航胶囊，h=24 r=full
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

    <!-- ── 中右：4 个导航胶囊（点击切主区视图）── -->
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
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { state, layout, showSettings } from '../ui-state.js'
import MenuBar from './MenuBar.vue'
import { clientViews, mountListSlot } from '../plugin-runtime.js'
import { switchActivity } from '../app-actions.js'
import api from '../api.js'
import logoUrl from '../assets/logo.svg'

// titlebar-right 槽位宿主（右段）：插件入口渲染目标 + 退订句柄
const tbRightEl = ref(null)
let tbRightUnsub = null

// ── 4 个导航胶囊（设计稿 th191/193/195/197：对话 / 编辑器 / 设计 / 市场）──
const navs = [
  { key: 'conversation', label: '对话', title: '对话主视图' },
  { key: 'editor', label: '编辑器', title: '代码编辑器' },
  { key: 'design', label: '设计', title: '设计画板视图' },
  { key: 'market', label: '市场', title: '插件市场' },
]

// 设计视图来自插件注册表（registerView）：title/id 命中「设计」即认领该胶囊
const designView = computed(() =>
  (clientViews || []).find(v => v && (v.title === '设计' || v.id === 'design' || v.id === 'art')) || null)

const mainTab = computed(() => state.panels.mainTab)

function isNavActive(n) {
  if (n.key === 'design') {
    const v = designView.value
    return !!v && mainTab.value === layout.viewTabKey(v.pluginName, v.id)
  }
  return mainTab.value === n.key
}

function onNav(n) {
  if (n.key === 'design') {
    const v = designView.value
    if (!v) { window.$toast && window.$toast('设计视图未注册（插件未装配）', 'info'); return }
    layout.openViewTab(v.pluginName, v.id)   // 未打开则后台打开再激活
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
})
onUnmounted(() => {
  if (tbRightUnsub) { tbRightUnsub(); tbRightUnsub = null }
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
}
.tb-nav-pill {
  display: inline-flex; align-items: center;
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
