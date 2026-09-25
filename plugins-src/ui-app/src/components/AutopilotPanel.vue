<template>
  <!-- ═══ 监督者看板（自主模式 autopilot）════════════════════════════════
       卡面与右栏卡片同设计语言：radius 8 + border + 11/12px 字号 + 令牌配色。
       ★ 2026-09-24 按新 UI 风格重做（用户指令）：折叠箭头改用 SvgIcon（与右栏同源），
         裁决标记由描边式 .ap-badge 改为填充胶囊 .ap-chip。 -->
  <div class="ap-panel" :class="{ collapsed: !expanded }">
    <div class="ap-header" @click="$emit('toggle')">
      <SvgIcon name="chevron-down" :size="14" class="ap-chevron" />
      <SvgIcon name="eye" :size="14" class="ap-eye" />
      <span class="ap-title">监督者</span>
      <span class="ap-count">{{ rounds.length }} 轮</span>
      <span v-if="lastActionLabel" :class="['ap-chip', 'ap-' + lastAction]">{{ lastActionLabel }}</span>
    </div>
    <div v-if="expanded" class="ap-body" ref="bodyRef">
      <!-- ★ 2026-09-25 关键信息优先（用户指令：「只呈现关键信息、可快速了解」）：
           每轮 = 可折叠行（<details>），**仅最新一轮默认 open** —— 默认视图直接给出
           当前裁决 + 结论 + 下一步；历史轮压成单行摘要（#序号 + 裁决胶囊 + 耗时/步数），
           点开才看全文与证据。原实现每轮无条件全展开、三条 details 竖排堆叠 → 轮次一多
           就是一大块文本堆（「内容拥挤、结构性差」）。 -->
      <details v-for="(r, i) in rounds" :key="(r.startedAt || '') + '#' + r.round"
               class="ap-round" :class="{ 'is-current': i === rounds.length - 1 }"
               :open="i === rounds.length - 1" @toggle="onRoundToggle(i, $event)">
        <summary class="ap-sum">
          <!-- 会话内累计序号（唯一可辨）；r.round 是本次运行内序号（每次运行从 1 重数）→ 悬停提示 -->
          <span class="ap-round-no" :title="'本次运行的第 ' + r.round + ' 次监督'">#{{ i + 1 }}</span>
          <span :class="['ap-chip', 'ap-' + badgeKey(r)]">{{ badgeLabel(r) }}</span>
          <span v-if="i === rounds.length - 1" class="ap-now">当前</span>
          <span class="ap-meta">
            {{ fmtDuration(r.durationMs) }}<template v-if="r.steps || r.toolCalls"> · {{ r.steps }} 步 / {{ r.toolCalls }} 工具</template><template v-if="tokens(r)"> · {{ tokens(r) }} tokens</template>
          </span>
          <span v-if="r.error" class="ap-err" :title="r.error">异常</span>
        </summary>
        <!-- ★ 2026-09-25 性能：正文**懒挂载**（v-if）—— 收起的历史轮不再创建 DOM。
             实测本面板曾占全页文本 324 万字符 / 7216 节点（全页 8424 节点、322 万字符
             的 99.8%），其中「轨迹」一家就 302 万字符（4281 个 span）全部无条件渲染。 -->
        <div v-if="roundBodyShown(i)" class="ap-round-body">
          <div v-if="r.assessment" class="ap-text">{{ r.assessment }}</div>
          <div v-if="r.nextTask" class="ap-next">
            <span class="ap-next-label">下一步</span>
            <span class="ap-next-text">{{ r.nextTask }}</span>
          </div>
          <!-- 详情三件套：横排一行、默认收起（原为三行竖排堆叠，是「拥挤」的主因）；
               任一项展开时该项独占整行（.ap-more[open] → flex 1 0 100%）。 -->
          <div v-if="r.evidence || (r.trace && r.trace.length) || r.workerReport" class="ap-more-row">
            <details v-if="r.evidence" class="ap-more" @toggle="onMoreToggle(i, 'evidence', $event)">
              <summary>证据</summary>
              <pre v-if="moreShown(i, 'evidence')" class="ap-pre">{{ r.evidence }}</pre>
            </details>
            <details v-if="r.trace && r.trace.length" class="ap-more" @toggle="onMoreToggle(i, 'trace', $event)">
              <summary>轨迹 {{ r.trace.length }}</summary>
              <div v-if="moreShown(i, 'trace')" class="ap-trace-list">
                <div v-for="(t, ti) in traceShown(r, i)" :key="ti" class="ap-trace">
                  <span v-if="t.type === 'tool_call'" class="ap-trace-tool">{{ t.name }}</span>
                  <span class="ap-trace-text">{{ t.type === 'tool_call' ? (t.result || t.args) : t.content }}</span>
                </div>
                <button v-if="traceHidden(r, i)" class="ap-trace-more" @click="expandTrace(i)">
                  还有 {{ traceHidden(r, i) }} 条更早的轨迹 · 点此全部展开
                </button>
              </div>
            </details>
            <details v-if="r.workerReport" class="ap-more" @toggle="onMoreToggle(i, 'worker', $event)">
              <summary>工作侧 {{ r.workerTurns }} 轮</summary>
              <pre v-if="moreShown(i, 'worker')" class="ap-pre">{{ r.workerReport }}</pre>
            </details>
          </div>
        </div>
      </details>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  rounds: { type: Array, default: () => [] },
  expanded: { type: Boolean, default: true },
})
defineEmits(['toggle'])

// ══════════════════════════════════════════════════════════════
// ★★ 2026-09-25 性能修复：折叠区**懒挂载**（v-if，而非仅靠 CSS 隐藏）★★
//   根因：<details> 收起只影响显示，Vue 仍会把内部 DOM 全量创建 —— 实测本面板曾占
//   全页文本 324 万字符 / 7216 节点（全页 8424 节点、322 万字符中的 99.8%），其中
//   「轨迹」列表一家就 302 万字符（4281 个 <span>）全部无条件渲染 → 实测 GC 3.5s、
//   首屏 FCP 13.6s、滚动/输入掉帧。
//   现改为：正文与三件套详情**首次展开才挂载**（toggle 事件同步 open 态），
//   默认视图只剩「摘要行 + 当前轮结论/下一步」。
const openRounds = ref({})   // { [轮索引]: true } 正文已挂载过
const openMore = ref({})     // { ['轮索引:键']: true } 详情已挂载过

function onRoundToggle(i, e) {
  const open = !!(e && e.target && e.target.open)
  const next = { ...openRounds.value }
  if (open) next[i] = true; else delete next[i]
  openRounds.value = next
}
function onMoreToggle(i, key, e) {
  const k = i + ':' + key
  const open = !!(e && e.target && e.target.open)
  const next = { ...openMore.value }
  if (open) next[k] = true; else delete next[k]
  openMore.value = next
}
// 最后一轮恒为展开态（模板 :open="i === rounds.length - 1"）→ 正文必须同步挂载
function roundBodyShown(i) { return !!openRounds.value[i] || i === props.rounds.length - 1 }
function moreShown(i, key) { return !!openMore.value[i + ':' + key] }

// 轨迹截断：单轮轨迹可达数千条（每条数百字符）→ 默认只挂**最后 200 条**（最新进展），
// 其余折进「还有 N 条」按钮；点按钮即时展开全部（用户显式动作，不做静默丢弃）。
const TRACE_TAIL = 200
const traceExpanded = ref({})
function traceShown(r, i) {
  const arr = (r && r.trace) || []
  if (traceExpanded.value[i] || arr.length <= TRACE_TAIL) return arr
  return arr.slice(arr.length - TRACE_TAIL)
}
function traceHidden(r, i) {
  const n = ((r && r.trace) || []).length
  return traceExpanded.value[i] ? 0 : Math.max(0, n - TRACE_TAIL)
}
function expandTrace(i) { traceExpanded.value = { ...traceExpanded.value, [i]: true } }

// ★ 2026-09-25「关键信息优先」的必要配套：面板高 ≤320px（约 10 行摘要），而**当前轮
//   永远排在最后** —— 不自动滚动时，展开面板第一眼看到的是最早的历史轮（实测 44 轮
//   时视口里只有 #1~#10，当前轮 #44 在视口外），与用户诉求「只呈现关键信息、快速了解」
//   正好相反。故：展开时 / 首次挂载 → 滚到最新一轮；运行中新轮追加 → 仅在用户已处于
//   底部附近（40px 容差）时跟随，避免打断正在翻阅历史的用户。
const bodyRef = ref(null)
function scrollToLatest(force) {
  const el = bodyRef.value
  if (!el) return
  const nearBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 40
  if (!force && !nearBottom) return
  // ★ 对齐「当前轮的**顶部**」而非绝对底部：直接 el.scrollTop = scrollHeight 时，视口
  //   显示的是该轮的**尾部**（结论结尾 + 详情行），头部摘要行「#N 收尾 当前 1m19s…」
  //   被裁到视口上方 —— 实测 44 轮展开态确实如此（结论中段起头，看不到是哪一轮）。
  const lastEl = el.lastElementChild
  if (!lastEl) { el.scrollTop = el.scrollHeight; return }
  const delta = lastEl.getBoundingClientRect().top - el.getBoundingClientRect().top
  // 上限仍取最大滚动量：当前轮比视口矮时，轮顶对齐会露白 → 此时退回滚到底
  el.scrollTop = Math.min(el.scrollTop + delta, el.scrollHeight - el.clientHeight)
}
watch(() => props.expanded, async (v) => {
  if (!v) return
  await nextTick()
  scrollToLatest(true)
}, { immediate: true })
watch(() => props.rounds.length, async () => {
  await nextTick()
  scrollToLatest(false)
})

const last = computed(() => (props.rounds.length ? props.rounds[props.rounds.length - 1] : null))
const lastAction = computed(() => (last.value ? badgeKey(last.value) : ''))
const lastActionLabel = computed(() => (last.value ? badgeLabel(last.value) : ''))

// badgeKey：裁决徽标类型（裁决缺失或监督回合本身报错 → error，避免「收尾」掩盖异常）
function badgeKey(r) {
  if (!r) return 'done'
  if (r.error) return 'error'
  if (r.ended === 'error') return 'error'
  if (r.action === 'continue') return 'continue'
  if (r.action === 'done') return 'done'
  return 'error'
}
function badgeLabel(r) {
  const k = badgeKey(r)
  if (k === 'continue') return '续跑'
  if (k === 'done') return '收尾'
  return '异常'
}
function fmtDuration(ms) {
  const n = Number(ms) || 0
  if (n < 1000) return n + 'ms'
  if (n < 60000) return (n / 1000).toFixed(1) + 's'
  return Math.round(n / 60000) + 'm' + Math.round((n % 60000) / 1000) + 's'
}
function tokens(r) {
  const t = (Number(r.promptTokens) || 0) + (Number(r.outputTokens) || 0)
  return t > 0 ? t : 0
}
</script>

<style scoped>
/* ═══ 监督者看板样式：与右栏卡片同一设计语言（radius 8 / border / 11-12px / 令牌配色）═══
   裁决语义色：续跑(进行中)=warning、收尾(完成)=success、异常=danger。
   ★ 2026-09-24 按新 UI 风格重做（用户指令）：原实现是描边式徽章（border 1px currentColor）、
   radius 3 色块与 var(--border-radius) 混搭，与设计稿的**填充胶囊 + 统一圆角**不一致。 */
.ap-panel {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
  margin-top: 4px;
  /* ★ 2026-09-25 高度模型修复（用户报「展开被输入框盖住一部分」）：
     原 .ap-body 写死 max-height:300px ⇒ 面板最高 32(header)+300+4(margin)=336px，
     而父容器 .autopilot-container 只给 max-height:320px 且未设 overflow ⇒ 溢出的
     16px+ 落到容器外，被**后绘制**的不透明 .chat-input-area 背景盖住。
     现改为：面板自身 flex 列 + 可收缩（min-height:0），.ap-body 用 flex:1 吃掉
     剩余高度并内部滚动 ⇒ 面板高度恒 ≤ 容器高度，溢出源头消失。 */
  display: flex; flex-direction: column;
  /* ★ flex:0 1 auto —— 关键是那个 1：面板必须**可被容器压缩**（原 flex-shrink:0
     让面板拒绝压缩，内容多时直接撑破容器边界；min-height:0 解除 flex 最小尺寸下限，
     否则内容高度仍是硬下限、压不下去）。 */
  flex: 0 1 auto; min-height: 0;
}
.ap-panel.collapsed .ap-body { display: none; }
.ap-header {
  display: flex; align-items: center; gap: 6px;
  height: 32px; padding: 0 10px; cursor: pointer; user-select: none;
  flex: 0 0 auto;
}
.ap-header:hover { background: var(--bg-hover); }
.ap-chevron { flex-shrink: 0; color: var(--text-muted); transition: transform .15s; }
.ap-panel.collapsed .ap-chevron { transform: rotate(-90deg); }
.ap-eye { flex-shrink: 0; color: var(--accent); }
.ap-title { font-size: 12px; font-weight: 600; flex: 1; color: var(--text-primary); }
.ap-count { font-size: 11px; color: var(--text-muted); font-variant-numeric: tabular-nums; }
/* 裁决胶囊（填充式）：底 surface-3(→ --bg-active) + 语义色文字 —— 与设计稿 th150/th77 同族 */
.ap-chip {
  flex-shrink: 0; display: inline-flex; align-items: center;
  height: 20px; padding: 0 8px; border-radius: 999px;
  background: var(--bg-active);
  font-size: 11px; font-weight: 600; white-space: nowrap;
}
.ap-continue { color: var(--color-warning, #F0C158); }
.ap-done { color: var(--color-success, #4FD8A4); }
.ap-error { color: var(--color-danger, #FF8085); }
/* 唯一滚动区：flex 自适应（不再写死高度）+ 4px 细滚动条。
   ★ scrollbar-color: auto 必写：index.html 给 html 设的 scrollbar-color 是**可继承**属性，
   继承到非 auto 值会让 Chromium 忽略 ::-webkit-scrollbar 全部样式（滚动条回退 17px 平台默认）。
   详见 .pair/project.md「滚动条样式：两个坑」。 */
.ap-body {
  flex: 1 1 auto; min-height: 0; overflow-y: auto;
  border-top: 1px solid var(--border-color);
  scrollbar-gutter: stable; scrollbar-color: auto;
}
.ap-body::-webkit-scrollbar { width: 4px; }
.ap-body::-webkit-scrollbar-track { background: transparent; }
.ap-body::-webkit-scrollbar-thumb { background: var(--scrollbar-thumb); border-radius: 2px; }
.ap-body::-webkit-scrollbar-thumb:hover { background: var(--scrollbar-thumb-hover); }
/* 轮次行（可折叠）：摘要恒为一行的「#序号 + 裁决胶囊 + 指标」，展开才出结论/下一步/详情 */
.ap-round + .ap-round { border-top: 1px solid var(--border-color); }
.ap-sum {
  display: flex; align-items: center; gap: 6px; flex-wrap: wrap;
  min-height: 28px; padding: 4px 10px; box-sizing: border-box;
  cursor: pointer; user-select: none; list-style: none;
  font-size: 11px; color: var(--text-muted);
}
.ap-sum::-webkit-details-marker { display: none; }
.ap-sum::before { content: '▸'; flex-shrink: 0; font-size: 10px; color: var(--text-muted); }
.ap-round[open] > .ap-sum::before { content: '▾'; }
.ap-sum:hover { background: var(--bg-hover); }
.ap-round[open] > .ap-sum { color: var(--text-secondary); }
.ap-round-no { flex-shrink: 0; font-variant-numeric: tabular-nums; }
/* 指标不截断：空间不足时换行（保持与任务进度卡「零省略号」同一约定） */
.ap-meta { flex: 1; min-width: 0; }
.ap-now { flex-shrink: 0; font-weight: 600; color: var(--accent); }
.ap-err { flex-shrink: 0; color: var(--color-danger, #FF8085); }
.ap-round-body { padding: 0 10px 8px; }
.ap-text {
  font-size: 12px; color: var(--text-primary); line-height: 1.55;
  word-break: break-word; white-space: pre-wrap;
}
/* 「下一步」：底 surface-2(→ --bg-hover)、radius 6（旧实现 radius 3 与圆角体系不符） */
.ap-next {
  margin-top: 6px; display: flex; gap: 6px; align-items: flex-start;
  background: var(--bg-hover); border-radius: 6px; padding: 6px 8px;
}
.ap-next-label { flex-shrink: 0; font-size: 11px; font-weight: 600; color: var(--color-warning, #F0C158); }
.ap-next-text { font-size: 11px; color: var(--text-secondary); line-height: 1.55; word-break: break-word; white-space: pre-wrap; }
/* 详情三件套：横排一行（原为竖排三行堆叠 = 「内容拥挤」主因）；任一项展开时独占整行 */
.ap-more-row { margin-top: 6px; display: flex; align-items: flex-start; gap: 10px; flex-wrap: wrap; }
/* details/summary：去掉浏览器默认三角，改 ▸/▾ 前缀 + 悬停变亮（11px muted） */
.ap-more > summary {
  cursor: pointer; font-size: 11px; color: var(--text-muted); user-select: none;
  list-style: none; display: inline-flex; align-items: center; gap: 4px;
}
.ap-more > summary::-webkit-details-marker { display: none; }
.ap-more > summary::before { content: '▸'; font-size: 10px; }
.ap-more[open] > summary::before { content: '▾'; }
.ap-more > summary:hover { color: var(--text-primary); }
.ap-more[open] { flex: 1 0 100%; }
.ap-pre {
  margin: 6px 0 0; padding: 8px; background: var(--bg-primary);
  border: 1px solid var(--border-color); border-radius: 6px;
  font-size: 11px; color: var(--text-secondary); line-height: 1.55;
  white-space: pre-wrap; word-break: break-word; max-height: 200px; overflow-y: auto;
  /* ★ 滚动条统一 4px（项目约定，与任务进度卡/GitPanel 同口径）：scrollbar-color:auto
     必写 —— 否则继承 index.html 给 html 设的全局值，Chromium 会忽略 ::-webkit-scrollbar
     全部样式，实测本元素回退到 10px（比同面板其它滚动条明显粗，视觉不一致）。 */
  scrollbar-gutter: stable; scrollbar-color: auto;
}
.ap-pre::-webkit-scrollbar { width: 4px; }
.ap-pre::-webkit-scrollbar-track { background: transparent; }
.ap-pre::-webkit-scrollbar-thumb { background: var(--scrollbar-thumb); border-radius: 2px; }
/* 轨迹列表：内部滚动（长轨迹不撑高面板），文本换行不截断 */
.ap-trace-list { max-height: 240px; overflow-y: auto; scrollbar-gutter: stable; scrollbar-color: auto; }
.ap-trace-list::-webkit-scrollbar { width: 4px; }
.ap-trace-list::-webkit-scrollbar-track { background: transparent; }
.ap-trace-list::-webkit-scrollbar-thumb { background: var(--scrollbar-thumb); border-radius: 2px; }
.ap-trace { display: flex; gap: 6px; padding: 3px 0; font-size: 11px; }
.ap-trace-tool { flex-shrink: 0; color: var(--accent); font-family: var(--font-mono, monospace); }
.ap-trace-text {
  color: var(--text-muted); flex: 1; min-width: 0;
  word-break: break-word;
}
/* ★ 轨迹截断提示按钮：沿用 .ap-more > summary 的排版规格（11px muted / hover 提亮） */
.ap-trace-more {
  display: block; width: 100%; margin: 4px 0 0; padding: 3px 0;
  background: transparent; border: none; text-align: left;
  font-size: 11px; color: var(--text-muted); cursor: pointer;
}
.ap-trace-more:hover { color: var(--text-primary); text-decoration: underline; }
</style>
