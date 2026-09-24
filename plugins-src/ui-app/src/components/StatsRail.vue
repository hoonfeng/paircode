<template>
  <!-- ═══ 右栏（设计稿 shell-midnight th178：col 288×832，pad=space.md，gap=space.sm）═══
       四张卡逐项对齐设计稿：
         ① 运行统计   th148 264×188（标题 + accent 摘要行 + divider + 4 行明细）
         ①b Token 统计（★ 2026-09-25 用户指令：token 统计**统一收到右栏**，
                       紧接「运行统计」卡下方 —— 原左栏 ConvSidebar 的 .cache-ring
                       环形缓存图已迁到此处，左栏副本删除）
         ② 任务进度   th164 264×152（标题 + accent「已完成 / 总数」+ 任务行）
         ③ 上下文构成 th173 264×116（标题 + 水平分段条 + 构成明细 + 剩余）
         ④ 底提示     th177 264×72 （bg=surface-2，两行静态提示）
       卡 ①②③ 依次排列，spacer 把 ④ 推到底部（设计稿 th174 spacer h=244）。 -->
  <aside class="stats-rail">
    <!-- ① 运行统计
         ★ 数据源 = **后端运行统计** GET /api/conversations/{id}/run-stats
           （state.runStatsByConv ← agent-events.fetchRunStats ← 后端 agent 循环埋点累加，
            落盘 .pair/run-stats.json；输出速度 t/s 亦由后端按生成阶段耗时派生）。
         ★ 与消息区顶部 phase-bar 的 currentRunStat 是同一份数据 —— 本卡只做展示格式化，
           不累加、不派生、不兜底。（旧实现取的是 /token-stats 的上下文 token 统计，
           并把 settings.contextMaxTokens 当上限、`|| 1000000` 兜底 —— 口径错位 + 臆测，已删。） -->
    <section class="sr-card">
      <div class="sr-head">
        <span class="sr-title">运行统计</span>
        <button class="sr-icon-btn" :title="folded ? '展开明细' : '收起明细'" @click="folded = !folded">
          <SvgIcon :name="folded ? 'chevron-down' : 'chevron-up'" :size="16" />
        </button>
      </div>
      <div class="sr-summary">{{ runSummary }}</div>
      <template v-if="!folded">
        <div class="sr-divider"></div>
        <div class="sr-row"><span class="sr-k">步数</span><span class="sr-v">{{ runSteps }}</span></div>
        <div class="sr-row"><span class="sr-k">工具调用</span><span class="sr-v">{{ runToolCalls }}</span></div>
        <div class="sr-row"><span class="sr-k">LLM 调用</span><span class="sr-v">{{ runLLMCalls }}</span></div>
        <div class="sr-row"><span class="sr-k">输出速度</span><span class="sr-v sr-v-ok">{{ runSpeed }}</span></div>
      </template>
    </section>

    <!-- ①b Token 统计（★ 2026-09-25 用户指令：token 统计统一到右栏，落在「运行统计」下方）
         形态＝SVG donut（r=18，viewBox 48 缩放到 72px）+ 中心「NN% 缓存命中」+ 右侧数字明细。
         ★ 数据源订正（2026-09-24）：state.wsTokenStatsByWs[workspaceRoot] 来自
           **工作区级** GET /api/tokens/stats（TokenStats 结构），与 Sidebar.vue
           `:ws-token-stats`、RightPanel 的 wsTokenStats 是同一份 —— 即本卡五个数字
           属**工作区累计**口径（非当前会话）；会话级口径见「上下文构成」卡所用的
           /conversations/{id}/token-stats（contextMaxTokens 亦由该接口下发）。
           旧注释把它误标为会话级 token-stats —— 声称与事实不符即隐患，已订正。
         命中率 = cacheHitTokens / (cacheHitTokens + cacheMissTokens)：后端只下发计数、
           不下发比率，故该除法由前端完成（数学定义唯一，非估算、非兜底）。 -->
    <section class="sr-card">
      <div class="sr-head">
        <span class="sr-title">Token 统计</span>
        <span class="sr-badge">命中 {{ cacheRate }}%</span>
      </div>
      <div class="sr-tokens">
        <div class="cache-ring-wrap"
             :title="'缓存命中率 ' + cacheRate + '%（命中 ' + fmt(wsStats.cacheHitTokens) + ' / 未命中 ' + fmt(wsStats.cacheMissTokens) + '）'">
          <svg class="cache-ring" viewBox="0 0 48 48" width="72" height="72" aria-hidden="true">
            <circle cx="24" cy="24" r="18" fill="none" stroke="var(--border-color)" stroke-width="4" />
            <circle cx="24" cy="24" r="18" fill="none" stroke="var(--accent)" stroke-width="4"
                    :stroke-dasharray="cacheRingDash" stroke-linecap="round"
                    transform="rotate(-90 24 24)" style="transition: stroke-dasharray 0.3s;" />
          </svg>
          <div class="cache-ring-label">
            <span class="cache-ring-pct">{{ cacheRate }}%</span>
            <span class="cache-ring-text">缓存命中</span>
          </div>
        </div>
        <div class="sr-token-detail">
          <div class="sr-row"><span class="sr-k">输入</span><span class="sr-v">{{ fmt(wsStats.promptTokens) }}</span></div>
          <div class="sr-row"><span class="sr-k">● 缓存命中</span><span class="sr-v sr-v-ok">{{ fmt(wsStats.cacheHitTokens) }}</span></div>
          <div class="sr-row"><span class="sr-k">● 未命中</span><span class="sr-v">{{ fmt(wsStats.cacheMissTokens) }}</span></div>
          <div class="sr-row"><span class="sr-k">输出</span><span class="sr-v">{{ fmt(wsStats.completionTokens) }}</span></div>
          <div class="sr-divider"></div>
          <div class="sr-row sr-token-total"><span class="sr-k">总 Token</span><span class="sr-v">{{ fmt(wsStats.totalTokens) }}</span></div>
        </div>
      </div>
    </section>

    <!-- ② 任务进度（设计稿 th164 264×152：标题 24 + 胶囊「已完成 / 总数」+ 进度条 8 + 3 行任务）-->
    <section class="sr-card">
      <div class="sr-head">
        <span class="sr-title">任务进度</span>
        <span class="sr-badge" :title="'已完成 ' + doneCount + ' 项 / 共 ' + tasks.length + ' 项'">{{ doneCount }} / {{ tasks.length }}</span>
      </div>
      <!-- 设计稿 th154/th153：底 surface-2 + accent 填充，宽度按已完成比例 -->
      <div v-if="tasks.length > 0" class="sr-bar">
        <div class="sr-bar-fill" :style="{ width: donePct + '%' }"></div>
      </div>
      <div v-if="tasks.length === 0" class="sr-empty">暂无任务</div>
      <!-- ★ 2026-09-25 用户指令：任务进度＝**可滚动的完整列表**。
           全部任务逐条列出（不裁剪行数、不加省略号、不显示「还有 N 项…」），
           超出时在卡内滚动查看；任务文本换行显示完整内容。 -->
      <div v-else class="sr-tasks">
        <div v-for="(t, i) in tasks" :key="i" class="sr-task">
          <span class="sr-task-text">{{ t.subject }}</span>
          <span class="sr-task-st" :class="statusClass(t.status)">{{ statusText(t.status) }}</span>
        </div>
      </div>
    </section>

    <!-- ③ 上下文构成 -->
    <section class="sr-card">
      <div class="sr-head"><span class="sr-title">上下文构成</span></div>
      <div class="sr-segbar">
        <span v-for="(s, i) in segs" :key="i" class="sr-seg" :style="{ width: s.pct + '%', background: s.color }"></span>
      </div>
      <div class="sr-note">{{ composeText }}</div>
      <div v-if="ctxMax > 0" class="sr-note">剩余 {{ fmt(remainTokens) }}（{{ remainPct }}%）</div>
    </section>

    <div class="sr-filler"></div>

    <!-- ④ 底提示 -->
    <section class="sr-card sr-card-tip">
      <div class="sr-tip-main">外观可在「设置 → 外观」调整</div>
      <div class="sr-tip-sub">主题 / 背景图 / 面板不透明度</div>
    </section>
  </aside>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { state } from '../ui-state.js'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'

const folded = ref(false)
const tasks = ref([])

// ── ① 运行统计（数据源＝后端 GET /api/conversations/{id}/run-stats，见模板注释）──
//   state.runStatsByConv 由 agent-events.fetchRunStats 写入（后端唯一真源，
//   持久化 .pair/run-stats.json）。本卡只格式化展示，不做任何累加/派生/兜底。
const run = computed(() => state.runStatsByConv[state.currentConvId] || null)
const runTick = ref(0)        // 运行中每秒 +1，驱动耗时重算
let runTickTimer = null
// 有运行记录的判定：后端 startAt 存在且已有任一计数（避免把"无记录"显示成 0）
const runHasData = computed(() => {
  const r = run.value
  return !!(r && r.startAt && (r.steps > 0 || r.toolCalls > 0 || r.llmCalls > 0 || r.completionTokens > 0))
})
// 耗时：运行中用后端 startAt 与本地时钟差（同机时间），结束后定格为后端 durationMs
const runElapsedMs = computed(() => {
  const r = run.value
  void runTick.value
  if (!r || !r.startAt) return 0
  if (r.endAt || !r.running) return r.durationMs || (r.endAt > r.startAt ? r.endAt - r.startAt : 0)
  return Math.max(0, Date.now() - r.startAt)
})
// 摘要行：状态 · 耗时 · 输出速度（三项均取自后端字段，前端不做除法）
const runSummary = computed(() => {
  const r = run.value
  if (!runHasData.value) return '暂无运行记录'
  const seg = [r.running ? '运行中' : '已完成']
  if (runElapsedMs.value >= 1000) seg.push(fmtDuration(runElapsedMs.value))
  if (r.tokensPerSecond > 0) seg.push(fmtSpeed(r.tokensPerSecond))
  return seg.join(' · ')
})
const runSteps = computed(() => (runHasData.value ? run.value.steps : '—'))
const runToolCalls = computed(() => (runHasData.value ? run.value.toolCalls : '—'))
const runLLMCalls = computed(() => (runHasData.value ? run.value.llmCalls : '—'))
const runSpeed = computed(() => (runHasData.value && run.value.tokensPerSecond > 0
  ? fmtSpeed(run.value.tokensPerSecond) : '—'))

onMounted(() => { runTickTimer = setInterval(() => { runTick.value++ }, 1000) })
onUnmounted(() => { if (runTickTimer) clearInterval(runTickTimer) })

// ── 上下文统计（全局 state：agent-events.getConvCtxStats 写入；字段全部来自后端）──
const EMPTY = {
  promptTokens: 0, completionTokens: 0,
  cacheHitTokens: 0, cacheMissTokens: 0,
  systemTokens: 0, skillsTokens: 0, mcpTokens: 0,
  toolTokens: 0, historyTokens: 0, otherTokens: 0,
  contextMaxTokens: 0,
}
const ctx = computed(() => state.convCtxStatsByConv[state.currentConvId] || EMPTY)
// ★ 上下文上限 = **后端**装配口径真值：/conversations/{id}/token-stats 的 contextMaxTokens
//   字段（后端 = agent.ContextWindow：服务商级 models.json 装配结果 > 机制常量
//   core.DefaultContextWindow）。不再读 settings 顶层 contextMaxTokens —— 该字段
//   2026-09-20 已迁入插件注册域、按机制「不得参与窗口取值」，是过期副本；
//   也不再 `|| 1000000` 兜底 —— 那是硬编码臆测，会显示与后端不一致的假上限。
const ctxMax = computed(() => ctx.value.contextMaxTokens || 0)
// 上下文占用 = 输入侧 promptTokens（后端字段直接采用）。旧实现额外 + completionTokens，
//   但模型输出不占上下文窗口 —— 那是前端拼接出来的口径，已删。
const ctxUsed = computed(() => ctx.value.promptTokens || 0)

// ── 上下文构成（设计稿 th171「提示词 12K · 历史 6K · 工具 3K · 其他 9K」）──
// ★ 段口径 = 后端 PromptBreakdown 各分项（types.go：估算 prompt 内各类构成，
//   经 NormalizeBreakdown 归一化到 prompt_tokens）—— 即
//   system+skills+mcp+tool+history+other ≈ promptTokens。
//   故 ① 不得再叠加 promptTokens（旧实现 `promptTokens + system + skills + mcp`
//   把总数重复计入，分段条与"提示词"数值双双虚高）；② 输出 token 也不属上下文构成
//   （旧实现 `other + completionTokens`）。两处臆测口径均已修正。
const parts = computed(() => {
  const c = ctx.value
  return {
    prompt: (c.systemTokens || 0) + (c.skillsTokens || 0) + (c.mcpTokens || 0),
    history: c.historyTokens || 0,
    tool: c.toolTokens || 0,
    other: c.otherTokens || 0,
  }
})
const segs = computed(() => {
  const p = parts.value
  const tot = p.prompt + p.history + p.tool + p.other
  if (tot <= 0) return [{ pct: 100, color: 'var(--border-color)' }]
  // 设计稿 th166-th169 四段配色的语义顺序：accent / success / warning / surface-3
  return [
    { pct: p.prompt / tot * 100, color: 'var(--accent)' },
    { pct: p.history / tot * 100, color: 'var(--success, #4FD8A4)' },
    { pct: p.tool / tot * 100, color: 'var(--warning, #F0C158)' },
    { pct: p.other / tot * 100, color: 'var(--bg-active)' },
  ]
})
const composeText = computed(() => {
  const p = parts.value
  return `提示词 ${fmt(p.prompt)} · 历史 ${fmt(p.history)} · 工具 ${fmt(p.tool)} · 其他 ${fmt(p.other)}`
})
const remainTokens = computed(() => Math.max(0, ctxMax.value - ctxUsed.value))
const remainPct = computed(() => ctxMax.value > 0 ? Math.round(remainTokens.value / ctxMax.value * 100) : 0)

// ── ★ 2026-09-25 用户指令：token 统计统一到右栏（原左栏 ConvSidebar 的环形缓存图迁到此）
//   数据源 state.wsTokenStatsByWs[workspaceRoot] —— 后端 GET /conversations/{id}/token-stats
//   真源，与 Sidebar.vue `:ws-token-stats`、RightPanel 的 wsTokenStats 同一份（禁止客户端估算）。
const EMPTY_WS_TOKENS = { totalTokens: 0, promptTokens: 0, completionTokens: 0, cacheHitTokens: 0, cacheMissTokens: 0 }
const wsStats = computed(() => state.wsTokenStatsByWs[state.workspaceRoot] || EMPTY_WS_TOKENS)
const cacheDenom = computed(() => (wsStats.value.cacheHitTokens || 0) + (wsStats.value.cacheMissTokens || 0))
const cacheRate = computed(() =>
  cacheDenom.value > 0 ? ((wsStats.value.cacheHitTokens / cacheDenom.value) * 100).toFixed(1) : '0.0')
// r=18 → 周长 ≈113.1（与 viewBox 用户坐标一致）
const cacheRingDash = computed(() => {
  const circ = 2 * Math.PI * 18
  if (cacheDenom.value <= 0) return `0 ${circ}`
  const hit = circ * (wsStats.value.cacheHitTokens / cacheDenom.value)
  return `${hit} ${circ - hit}`
})

// ── 任务进度（GET /tasks?convId=…，与 RightPanel.loadConvTasks 同源）──
const doneCount = computed(() => tasks.value.filter(t => t.status === 'completed').length)
// ★ 2026-09-25 用户指令：任务列表＝**全量 tasks 的可滚动列表**。
//   原实现 `slice(0, 3)` 裁剪 3 行 + 单行省略号 + 「还有 N 项…」提示已全部删除
//   —— 用户要看的是全部任务，不是被截断的摘要。
// 进度条填充比例 = 已完成 / 总数（无任务时不渲染该条，避免 0/0 产生 NaN 宽度）。
const donePct = computed(() => tasks.value.length > 0 ? (doneCount.value / tasks.value.length) * 100 : 0)
const statusText = (s) => (s === 'completed' ? '完成' : s === 'in_progress' ? '进行中' : s === 'cancelled' ? '已取消' : '待执行')
const statusClass = (s) => (s === 'completed' ? 'sr-v-ok' : s === 'in_progress' ? 'sr-v-run' : 'sr-v-dim')

async function loadTasks() {
  const convId = state.currentConvId
  if (!convId) { tasks.value = []; return }
  try {
    const data = await api.apiGet('/tasks', { convId })
    if (state.currentConvId !== convId) return
    const list = (data && Array.isArray(data.tasks)) ? data.tasks : []
    tasks.value = list.map(t => ({ subject: t.subject || t.step || '', status: t.status || 'pending' }))
  } catch (e) {
    if (state.currentConvId === convId) tasks.value = []
  }
}

onMounted(loadTasks)
watch(() => state.currentConvId, loadTasks)

// ── 数字简写（与设计稿 44.4K / 1.0M 同格式）──
function fmt(n) {
  const v = Number(n) || 0
  if (v >= 1000000) return (v / 1000000).toFixed(1) + 'M'
  if (v >= 1000) return (v / 1000).toFixed(1) + 'K'
  return String(v)
}
// fmtDuration 耗时人性化（12s / 1m 23s / 1h 05m；与 RightPanel.formatDuration 同口径）
function fmtDuration(ms) {
  const sec = Math.floor((Number(ms) || 0) / 1000)
  if (sec < 60) return sec + 's'
  const min = Math.floor(sec / 60)
  if (min < 60) return min + 'm ' + String(sec % 60).padStart(2, '0') + 's'
  return Math.floor(min / 60) + 'h ' + String(min % 60).padStart(2, '0') + 'm'
}
// fmtSpeed 输出速度（后端派生 tokensPerSecond，前端只做格式：≥100 取整，否则 1 位小数）
function fmtSpeed(tps) {
  const v = Number(tps) || 0
  return (v >= 100 ? Math.round(v) : v.toFixed(1)) + ' t/s'
}
</script>

<style scoped>
/* ═══ 设计稿 th178：col w=288, bg=color.bg, pad=space.md(12), gap=space.sm(8) ═══ */
.stats-rail {
  width: 288px; height: 100%;
  display: flex; flex-direction: column; gap: 8px;
  padding: 12px;
  background: var(--bg-primary);
  border-left: 1px solid var(--border-color);
  overflow-y: auto; overflow-x: hidden;
  box-sizing: border-box;
}
/* 设计稿 th148/th164/th173：bg=surface, radius=8, border, pad=12, gap=4 */
.sr-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 12px;
  display: flex; flex-direction: column; gap: 4px;
  flex: 0 0 auto;
}
.sr-head { display: flex; align-items: center; justify-content: space-between; height: 24px; }
/* 设计稿 th131/th151/th165：字号 sm(12) + 字重 600 + fg */
.sr-title { font-size: 12px; font-weight: 600; color: var(--text-primary); }
/* 设计稿 th150（任务进度「4 / 7」）与 th77/th85（主区结果徽章）：
   64×24 居中胶囊 —— radius=full、bg=surface-3(→ --bg-active)、accent 文字、等宽数字。
   本类同时供「Token 统计」卡的「命中 NN%」使用，两处形态统一。 */
.sr-badge {
  display: inline-flex; align-items: center; justify-content: center;
  min-width: 64px; height: 24px; padding: 0 10px;
  border-radius: 999px; background: var(--bg-active);
  font-size: 11px; color: var(--accent); font-variant-numeric: tabular-nums;
}
.sr-icon-btn {
  border: none; background: none; color: var(--text-muted); cursor: pointer;
  display: inline-flex; align-items: center; padding: 0; opacity: .75;
}
.sr-icon-btn:hover { opacity: 1; color: var(--text-primary); }
/* 设计稿 th134：accent 色摘要行 */
.sr-summary { font-size: 11px; color: var(--accent); line-height: 1.5; }
.sr-divider { height: 1px; background: var(--border-color); margin: 2px 0; }
/* 设计稿 th138/th141/…：行高 24，左 muted 右 fg */
.sr-row { display: flex; align-items: center; justify-content: space-between; height: 24px; gap: 8px; }
.sr-k { font-size: 11px; color: var(--text-muted); min-width: 0; }
.sr-v { font-size: 11px; color: var(--text-primary); white-space: nowrap; }
.sr-v-ok { color: var(--success, #4FD8A4); }
.sr-v-run { color: var(--accent); }
.sr-v-dim { color: var(--text-muted); }
.sr-empty { font-size: 11px; color: var(--text-muted); padding: 4px 0; }
.sr-note { font-size: 11px; color: var(--text-muted); line-height: 1.5; }
/* 设计稿 th154/th153：h=8、圆角 4 的进度条（底 = surface-2 → --bg-hover；填充 = accent） */
.sr-bar { height: 8px; border-radius: 4px; background: var(--bg-hover); overflow: hidden; }
.sr-bar-fill { height: 100%; border-radius: 4px; background: var(--accent); transition: width .25s; }
/* ── ★ 任务进度列表（2026-09-25 用户指令）：**完整可滚动列表** ──
   全部任务逐条列出；行数超出时在卡内 max-height 内滚动（细滚动条），
   任务文本换行显示完整内容 —— 不裁剪行数、不加省略号、不显示「还有 N 项…」。 */
.sr-tasks {
  flex: 1 1 auto; min-height: 0;
  max-height: 240px;
  overflow-y: auto; overflow-x: hidden;
  display: flex; flex-direction: column; gap: 2px;
  scrollbar-gutter: stable;
  /* ★ 必须显式置 auto：见下方滚动条注释（全局继承的 scrollbar-color 会废掉 webkit 伪元素） */
  scrollbar-color: auto;
}
/* 细滚动条（与 GitPanel 同口径：4px + --scrollbar-thumb）。
   ★★ 两个实测坑（否则滚动条根本不是 4px）：
     ① index.html 全局给 html 设了 scrollbar-color，而**该属性可继承** —— 只要继承到
        非 auto 的 scrollbar-color，Chromium 就忽略 ::-webkit-scrollbar 伪元素，
        滚动条回退平台默认粗度（实测 17px）⇒ 必须在本元素显式 scrollbar-color: auto；
     ② 一旦设了标准 scrollbar-width（如 thin），同样会让 ::-webkit-scrollbar 失效
        （实测 thin = 11px）⇒ 本元素刻意不写 scrollbar-width。 */
.sr-tasks::-webkit-scrollbar { width: 4px; }
.sr-tasks::-webkit-scrollbar-track { background: transparent; }
.sr-tasks::-webkit-scrollbar-thumb { background: var(--scrollbar-thumb); border-radius: 2px; }
.sr-tasks::-webkit-scrollbar-thumb:hover { background: var(--scrollbar-thumb-hover); }
/* 单条任务：左＝文本（换行完整显示）｜右＝状态（不换行、不被压缩） */
.sr-task { display: flex; align-items: flex-start; justify-content: space-between; gap: 8px; padding: 2px 0; }
.sr-task-text {
  flex: 1 1 auto; min-width: 0;
  font-size: 11px; line-height: 1.45; color: var(--text-muted);
  word-break: break-word;
}
.sr-task-st { flex: 0 0 auto; font-size: 11px; line-height: 1.45; white-space: nowrap; }
/* 设计稿 th170：h=8, gap=4, 圆角 4 的水平分段条（非圆环） */
.sr-segbar { display: flex; gap: 4px; height: 8px; margin: 4px 0; }
.sr-seg { height: 8px; border-radius: 4px; min-width: 2px; transition: width .25s; }
/* ── ★ Token 统计（卡 ①b）：环形缓存图 + 数字明细 ── */
.sr-tokens { display: flex; align-items: center; gap: 10px; padding-top: 2px; }
.cache-ring-wrap {
  position: relative; flex: 0 0 auto;
  width: 72px; height: 72px;
  display: flex; align-items: center; justify-content: center;
}
.cache-ring { display: block; }
.cache-ring-label {
  position: absolute;
  display: flex; flex-direction: column; align-items: center;
  line-height: 1.25;
}
.cache-ring-pct {
  font-size: 14px; font-weight: 700;
  color: var(--accent); font-family: var(--font-code);
}
.cache-ring-text { font-size: 9px; color: var(--text-muted); }
.sr-token-detail { flex: 1 1 auto; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.sr-token-detail .sr-row { height: 20px; }
.sr-token-total .sr-k, .sr-token-total .sr-v { font-weight: 600; color: var(--text-primary); }
.sr-filler { flex: 1 1 auto; min-height: 0; }
/* 设计稿 th177：bg=surface-2 */
.sr-card-tip { background: var(--bg-hover); }
.sr-tip-main { font-size: 11px; color: var(--text-primary); line-height: 1.5; }
.sr-tip-sub { font-size: 11px; color: var(--text-muted); line-height: 1.5; }
</style>
