<template>
  <div class="conv-sidebar" :class="{ 'conv-sidebar-horizontal': horizontal }" :style="horizontal ? {} : { width: width + 'px' }">
    <div class="conv-sidebar-header">
      <!-- ★ 2026-09-25 对齐设计稿 th23：标题「会话」(sm/600) + 右侧 2 图标
           （th20 muted 14 / th21 accent 14）。 -->
      <span class="csh-title">会话</span>
      <div class="csh-actions">
        <button class="csh-btn" title="刷新会话列表" @click="refreshConvs"><SvgIcon name="refresh" :size="14" /></button>
        <button class="csh-btn csh-btn-accent" title="新建对话" @click="$emit('new-conversation')"><SvgIcon name="plus" :size="14" /></button>
      </div>
    </div>
    <div class="conv-sb-divider"></div>

    <!-- ★ 2026-09-25 对齐设计稿 th61：分组标签（th25「今天」/ th41「更早」）＋
         行结构对齐 th30（row 248×32, pad=8, bg=surface-3 选中）：
         [icon(accent 14)] + [标题(sm, fg, w156)] + [计数(52, right, muted, xs)]。
         ★ 监督纠正：去掉设计稿没有的**时间**列；「运行中/未完成」保留为极小的
           状态点（功能必需信息，设计稿无对应节点，已在 notes §26 说明理由）。 -->
    <div class="conv-list">
      <template v-if="todayConvs.length">
        <div class="conv-group-label">今天</div>
        <div v-for="conv in todayConvs" :key="conv.id" class="conv-item"
             :class="{ active: conv.id === currentConvId }"
             @click="$emit('switch-conversation', conv.id)">
          <SvgIcon class="conv-row-icon" name="message-square" :size="14" />
          <span class="conv-title">{{ conv.title }}</span>
          <span v-if="loadingByConv[conv.id]" class="conv-running-dot" title="Agent 运行中"></span>
          <span v-else-if="conv.interrupted" class="conv-interrupted-dot" title="上次任务未完成"></span>
          <span class="conv-msg-count">{{ conv.msgCount || 0 }}</span>
          <button class="conv-del" @click.stop="$emit('delete-conversation', conv.id)" title="删除对话">×</button>
        </div>
      </template>
      <template v-if="earlierConvs.length">
        <div class="conv-group-label">更早</div>
        <div v-for="conv in earlierConvs" :key="conv.id" class="conv-item"
             :class="{ active: conv.id === currentConvId }"
             @click="$emit('switch-conversation', conv.id)">
          <SvgIcon class="conv-row-icon" name="message-square" :size="14" />
          <span class="conv-title">{{ conv.title }}</span>
          <span v-if="loadingByConv[conv.id]" class="conv-running-dot" title="Agent 运行中"></span>
          <span v-else-if="conv.interrupted" class="conv-interrupted-dot" title="上次任务未完成"></span>
          <span class="conv-msg-count">{{ conv.msgCount || 0 }}</span>
          <button class="conv-del" @click.stop="$emit('delete-conversation', conv.id)" title="删除对话">×</button>
        </div>
      </template>
      <div v-if="localConvs.length === 0" class="conv-empty">暂无对话</div>
    </div>

    <!-- ★ 2026-09-25 用户指令：token 统计**统一到右栏**（StatsRail「运行统计」卡下方的
         「Token 统计」卡 = 环形缓存图 + 数字明细）。左栏不再重复渲染同一份数据，
         原 .cs-tokens / .cache-ring 结构已移除，避免同一统计两处展示。 -->
    <!-- ★ 2026-09-25 对齐设计稿 th60：底部＝胶囊（row 248×32, pad=8, gap=8,
         bg=surface-2, radius=8, border）＝ icon(accent 14) + 文案(200, muted, xs)
         「全栈开发 · 14 插件」。★ 监督纠正：原先的「市场/设置」两按钮形态不符，
         已改为设计稿的单胶囊形态。 -->
    <div class="conv-footer-pill" :title="footerTitle">
      <SvgIcon name="layers" :size="14" />
      <span class="cfp-text">{{ footerLabel }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import SvgIcon from './SvgIcon.vue'
import { state, showSettings } from '../ui-state.js'
import { switchActivity } from '../app-actions.js'

function openMarketplace() {
  switchActivity('marketplace')
}
function openSettings() {
  showSettings.value = true
}

const props = defineProps({
  conversations: { type: Array, default: () => [] },
  currentConvId: { type: String, default: '' },
  loadingByConv: { type: Object, default: () => ({}) },
  wsTokenStats: { type: Object, default: () => ({ totalTokens: 0, promptTokens: 0, completionTokens: 0, cacheHitTokens: 0, cacheMissTokens: 0 }) },
  convCtxStats: { type: Object, default: () => ({ promptTokens: 0, completionTokens: 0, cacheHitTokens: 0, cacheMissTokens: 0, systemTokens: 0, skillsTokens: 0, mcpTokens: 0, toolTokens: 0, historyTokens: 0, otherTokens: 0 }) },
  ctxMaxTokensVal: { type: Number, default: 64000 },
  width: { type: Number, default: 250 },
  horizontal: { type: Boolean, default: false },
})

defineEmits(['new-conversation', 'switch-conversation', 'delete-conversation'])

// ★ wb-ui(goja) workaround：父组件 state.conversations 整体赋值
//   （state.conversations = list）触发不了 v-for 的渲染 effect（Vue 3 的
//   v-for 依赖数组内部 length/indices，整体赋值 set 属性不触发）。watch
//   在 goja 里正常触发（显式依赖），这里把 prop 复制到本地 ref——ref 的
//   set 能触发 v-for 重新渲染。
const localConvs = ref([])

// ── ★ 2026-09-25 对齐设计稿 th61：分组（th25「今天」/ th41「更早」）──
function isToday(iso) {
  if (!iso) return false
  const d = new Date(iso)
  if (isNaN(d.getTime())) return false
  const n = new Date()
  return d.getFullYear() === n.getFullYear() && d.getMonth() === n.getMonth() && d.getDate() === n.getDate()
}
const todayConvs = computed(() => localConvs.value.filter(c => isToday(c.updatedAt)))
const earlierConvs = computed(() => localConvs.value.filter(c => !isToday(c.updatedAt)))

// 表头 th20 图标：刷新会话列表（触发持久化事件让上层重新同步，不新增轮询）
function refreshConvs() { window.dispatchEvent(new Event("save-conversations")) }

// ── ★ 底部胶囊 th60：「全栈开发 · 14 插件」＝ 当前场景（会话级工具集）+ 该场景的插件数 ──
// ★ 2026-09-25 接线修复：原读 settings.scenarioName / toolsetName / defaultToolset ——
//   这三个字段全项目从未被写入（grep 仅此处 1 处引用）→ 胶囊恒显示硬编码字面量
//   「默认场景」；且读的是**全局 settings** 而非当前会话 → 切换对话也不变；插件数取的
//   还是「全部已装插件数」（与本场景无关）。
//   现改为读 state.convToolsetInfo —— 唯一写方 = RightPanel.publishConvToolsetInfo()，
//   权威源 GET /api/toolsets/active（agent.ResolveConvToolsetActive：会话元数据未显式
//   选择时回落默认集合「基础」）→ 场景名与插件数均随当前对话变化。
const pluginCount = computed(() => (state.pluginSchemas ? state.pluginSchemas.length : 0))
const footerLabel = computed(() => {
  const info = state.convToolsetInfo || {}
  // ★ 未解析（页面首帧 / 刚切换对话）：显示"解析中"而非任何场景名 —— 切换对话时
  //   真实场景要等异步请求返回（切到大对话时主线程繁忙，实测可达数秒），若此时
  //   沿用上一个对话的值即为**错误信息**（该窗口已用 CDP 时间线 + fetch 探针实测确认）。
  if (!info.resolved) return "场景解析中…"
  // ① 已收敛到具体集合：场景名 + 该集合装配的插件数（与右侧选择器 toolsetLabel 同源）
  if (info.name) return info.name + " · " + (info.pluginCount || 0) + " 插件"
  // ② 未收敛（集合文件缺失/无集合配置）：不按集合收敛工具面，全部工具可用
  return "全部工具可用 · " + pluginCount.value + " 插件"
})
const footerTitle = computed(() => {
  const info = state.convToolsetInfo || {}
  if (info.resolved && info.name) {
    return "当前场景与已装配插件：" + footerLabel.value +
      (info.isDefault ? "（默认集合" + (info.defaultName ? "：" + info.defaultName : "") + "）" : "（本对话已选择）")
  }
  return "当前场景与已装配插件：" + footerLabel.value
})
watch(() => props.conversations, (v) => {
  localConvs.value = Array.isArray(v) ? v.slice() : []
}, { immediate: true, deep: true })

// ★ 2026-09-25 用户指令：token 统计统一到右栏后，左栏的 cacheRate / cacheRingDash /
//   shortTokens 已无消费方（右栏 StatsRail 用自带的 cacheRate/cacheRingDash + fmt），故移除。
//   props.wsTokenStats 声明仍保留：Sidebar.vue 继续透传，删声明会产生 extraneous prop 噪声。

</script>

<style scoped>
.conv-sidebar {
  flex-shrink: 0;
  border-left: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg-tertiary);
  position: relative;
  min-width: 200px;
}
.conv-sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 10px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
  letter-spacing: 0.3px;
}
.conv-list {
  flex: 1 1 auto;
  overflow-y: auto;
  /* ★ 右侧 padding 8px：wb-ui 滚动条为 overlay（不占内容空间），
     conv-item 默认延伸至容器右缘会盖住滚动条（选中/悬停背景）。
     加 padding-right 后 item 右缘退到滚动条左侧，与 Edge 一致
     （Edge 滚动条占 8px 内容空间，item 右缘同样距滚动条 12px）。 */
  padding: 4px 8px 4px 0;
  min-height: 0;
}
.conv-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 10px;
  cursor: pointer;
  font-size: 12px;
  color: var(--text-secondary);
  position: relative;
  transition: background 0.12s, padding-left 0.15s;
  margin: 1px 4px;
  border-radius: 6px;
  border-left: 2px solid transparent;
}
.conv-item.active {
  background: var(--bg-active);
  color: var(--text-primary);
  font-weight: 500;
  border-left-color: var(--accent);
  padding-left: 12px;
}
.conv-item:hover { background: var(--bg-hover); }
.conv-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  line-height: 1.4;
}
@keyframes conv-pulse {
  0%, 100% { opacity: 0.4; transform: scale(0.8); }
  50% { opacity: 1; transform: scale(1.2); }
}
.conv-msg-count {
  font-size: 10px;
  color: var(--text-muted);
  background: var(--bg-primary);
  padding: 0 4px;
  border-radius: 6px;
  line-height: 16px;
  min-width: 16px;
  text-align: center;
}
.conv-del {
  display: none;
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 14px;
  padding: 0 2px;
  line-height: 1;
  opacity: 0.5;
  transition: opacity 0.12s;
}
.conv-item:hover .conv-del { display: block; }
.conv-del:hover { opacity: 1; color: var(--color-danger); }
.conv-empty {
  padding: 16px 8px;
  font-size: 11px;
  color: var(--text-muted);
  text-align: center;
  font-style: italic;
}

/* ── 统计/上下文面板 ── */
/* ★ 2026-09-25（设计稿屏 1 ②）：上下文折叠态的**一行摘要** ——
   已用/上限 · 占用比例 · 缓存命中率（后者直接决定要不要压缩历史）。
   沿用 .conv-stats-pct 的等宽数字排版，但信息量从 1 个百分比升到 3 项。 */

/* ★ 2026-09-25 用户指令：token 统计统一到右栏 —— 左栏 .cs-tokens / .cache-ring-* /
   .conv-stats-detail / .cs-row 等样式已随之移除（对应结构迁至 StatsRail.vue 卡 ①b）。 */

/* ── 上下文窗口条 ── */

/* ── 构成占比横条 ── */

/* ── rp-btn 复用 ── */
.rp-btn {
  background: none; border: 1px solid transparent; color: var(--text-secondary);
  padding: 2px 6px; cursor: pointer; border-radius: 3px; display: flex; align-items: center;
}
.rp-btn:hover { background: var(--bg-hover); color: var(--text-primary); }

/* ── 技能管理区域 ── */
.skill-mgr-wrap {
  margin-top: 8px;
  padding-top: 6px;
  border-top: 1px solid var(--border-color);
}
.skill-mgr-header {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.3px;
  margin-bottom: 6px;
}
.skill-token-badge {
  margin-left: auto;
  font-family: var(--font-code);
  font-size: 10px;
  color: var(--color-success);
  background: var(--color-cat-amber-bg);
  padding: 0 5px;
  border-radius: 6px;
  line-height: 16px;
}
.skill-mgr-actions {
  display: flex;
  gap: 4px;
  margin-bottom: 4px;
}
.skill-mgr-btn {
  display: flex;
  align-items: center;
  gap: 3px;
  padding: 3px 8px;
  font-size: 11px;
  color: var(--text-secondary);
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.12s, color 0.12s;
}
.skill-mgr-btn:hover {
  background: var(--bg-hover);
  color: var(--accent);
  border-color: var(--accent);
}
.skill-mgr-empty {
  font-size: 10px;
  color: var(--text-muted);
  font-style: italic;
  padding: 2px 0;
}

/* ── 已安装列表 ── */
.installed-list {
  margin-top: 2px;
}
.installed-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 6px;
  border-radius: 4px;
  font-size: 11px;
  transition: background 0.1s;
}
.installed-item:hover {
  background: var(--bg-hover);
}
.installed-info {
  flex: 1;
  min-width: 0;
  overflow: hidden;
}
.installed-name {
  font-weight: 500;
  color: var(--text-primary);
  display: block;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.installed-detail {
  font-size: 10px;
  color: var(--text-muted);
  display: block;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.installed-del-btn {
  flex-shrink: 0;
  background: none;
  border: 1px solid transparent;
  color: var(--text-muted);
  width: 22px;
  height: 22px;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.12s;
}
.installed-del-btn:hover {
  color: var(--color-danger);
  border-color: var(--color-danger-bg);
  background: var(--color-danger-bg);
}

/* ── 底部快捷入口 ── */
.conv-footer-actions {
  display: flex;
  gap: 2px;
  padding: 8px 10px;
  border-top: 1px solid var(--border-color);
  flex-shrink: 0;
}
.conv-footer-btn {
  flex: 1;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-muted);
  padding: 5px 8px;
  border-radius: 4px;
  font-size: 10px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  transition: all 0.12s;
}
.conv-footer-btn:hover {
  color: var(--text-primary);
  background: var(--bg-hover);
  border-color: var(--accent);
}
.conv-footer-btn:hover {
  color: var(--text-primary);
  background: var(--bg-hover);
  border-color: var(--accent);
}

/* ── 水平横条布局 ── */
.conv-sidebar-horizontal {
  display: flex !important;
  flex-direction: row !important;
  width: 100% !important;
  height: 100% !important;
  border-left: none !important;
  min-width: 0 !important;
  overflow: hidden;
}
.conv-sidebar-horizontal .conv-sidebar-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  gap: 4px;
  padding: 6px 8px;
  border-bottom: none;
  border-right: 1px solid var(--border-color);
  min-width: 44px;
  flex-shrink: 0;
}
.conv-sidebar-horizontal .conv-sidebar-header span {
  writing-mode: vertical-lr;
  font-size: 11px;
  letter-spacing: 2px;
}
.conv-sidebar-horizontal .conv-list {
  display: flex;
  flex-direction: row !important;
  gap: 4px;
  padding: 6px 8px;
  overflow-x: auto;
  overflow-y: hidden;
  flex: 1;
  min-width: 0;
  align-items: stretch;
}
.conv-sidebar-horizontal .conv-item {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 3px;
  padding: 6px 10px;
  min-width: 120px;
  max-width: 180px;
  flex-shrink: 0;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  margin: 0;
  background: var(--bg-primary);
}
.conv-sidebar-horizontal .conv-item.active {
  border-color: var(--accent);
  background: var(--accent-bg);
}
.conv-sidebar-horizontal .conv-title {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 12px;
  font-weight: 500;
  width: 100%;
}
.conv-sidebar-horizontal .conv-del {
  position: absolute;
  top: 2px;
  right: 4px;
}
.conv-sidebar-horizontal .conv-empty {
  font-size: 11px;
  padding: 8px 16px;
  white-space: nowrap;
}
.conv-sidebar-horizontal .cs-row {
  padding: 1px 0;
}
.conv-sidebar-horizontal .cs-val {
  font-size: 10px;
}
.conv-sidebar-horizontal .ctx-bar {
  height: 8px;
}
.conv-sidebar-horizontal .comp-bar {
  height: 6px;
}
.conv-sidebar-horizontal .comp-bar-title {
  font-size: 9px;
}
.conv-sidebar-horizontal .ctx-detail {
  font-size: 10px;
}
.conv-sidebar-horizontal .conv-footer-actions {
  padding: 4px 8px;
}
.conv-sidebar-horizontal .conv-footer-btn {
  font-size: 9px;
  padding: 3px 6px;
}


/* ═══════════════════════════════════════════════════════════════
   ★ 2026-09-25 对齐设计稿 th61（监督纠正后重写）
   th23 表头 / th25·th41 分组标签 / th30 会话行 / th60 底部胶囊
   ═══════════════════════════════════════════════════════════════ */
/* th23：row 248×32 — 标题 sm/600 + 右侧 2 图标（gap=space.xs） */
.conv-sidebar-header { display: flex; align-items: center; justify-content: space-between; height: 32px; padding: 0 8px; }
.csh-title { font-size: 12px; font-weight: 600; color: var(--text-primary); }
.csh-actions { display: flex; align-items: center; gap: 4px; }
.csh-btn { display: inline-flex; align-items: center; justify-content: center; width: 20px; height: 20px; border: none; background: none; cursor: pointer; border-radius: 4px; color: var(--text-muted); }
.csh-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
/* th21 = accent 图标（新建对话） */
.csh-btn-accent { color: var(--accent); }
/* th24 divider h=1 */
.conv-sb-divider { height: 1px; background: var(--border-color); margin: 2px 0 4px; }
/* th25 / th41：分组标签 */
.conv-group-label { font-size: 11px; color: var(--text-muted); padding: 6px 10px 2px; }
/* th30：row 248×32, pad=8；选中 bg=surface-3 */
.conv-list { flex: 1; min-height: 0; overflow-y: auto; overflow-x: hidden; }
.conv-item { display: flex; align-items: center; gap: 8px; height: 32px; padding: 0 8px; border-radius: 6px; cursor: pointer; background: var(--bg-secondary); position: relative; }
.conv-item:hover { background: var(--bg-hover); }
.conv-item.active { background: var(--bg-active); }
.conv-row-icon { flex: 0 0 auto; color: var(--accent); }
/* 标题占 w156 语义：flex:1 分配，多余省略 */
.conv-item .conv-title { flex: 1 1 auto; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; color: var(--text-primary); }
/* th29：计数 52/右对齐/muted/xs */
.conv-item .conv-msg-count { flex: 0 0 auto; font-size: 11px; color: var(--text-muted); }
/* 运行中/未完成：极小状态点（设计稿无对应节点，功能必需，见 notes §26） */
.conv-running-dot { flex: 0 0 auto; width: 6px; height: 6px; border-radius: 50%; background: var(--accent); }
.conv-interrupted-dot { flex: 0 0 auto; width: 6px; height: 6px; border-radius: 50%; background: var(--warning, #F0C158); }
/* th60：底部胶囊 row 248×32, pad=8, gap=8, bg=surface-2, radius=8, border */
.conv-footer-pill { display: flex; align-items: center; gap: 8px; height: 32px; padding: 0 8px; margin: 4px 8px 8px; border-radius: 8px; background: var(--bg-hover); border: 1px solid var(--border-color); color: var(--accent); flex: 0 0 auto; }
.cfp-text { font-size: 11px; color: var(--text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

</style>
