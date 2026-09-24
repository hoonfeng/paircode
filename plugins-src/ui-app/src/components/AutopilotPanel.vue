<template>
  <div class="ap-panel" :class="{ collapsed: !expanded }">
    <div class="ap-header" @click="$emit('toggle')">
      <svg v-if="!expanded" class="ap-chevron" viewBox="0 0 8 8" width="9" height="9" fill="currentColor" aria-hidden="true"><path d="M2.6 1.2 L6.8 4 L2.6 6.8 Z"/></svg>
      <svg v-else class="ap-chevron" viewBox="0 0 8 8" width="9" height="9" fill="currentColor" aria-hidden="true"><path d="M1.2 2.6 L4 6.8 L6.8 2.6 Z"/></svg>
      <SvgIcon name="eye" :size="12" />
      <span class="ap-title">监督者</span>
      <span class="ap-count">{{ rounds.length }} 轮</span>
      <span v-if="lastActionLabel" :class="['ap-badge', 'ap-' + lastAction]">{{ lastActionLabel }}</span>
    </div>
    <div v-if="expanded" class="ap-body">
      <div v-for="(r, i) in rounds" :key="(r.startedAt || '') + '#' + r.round" class="ap-round">
        <div class="ap-round-head">
          <!-- 会话内累计序号（唯一可辨）；r.round 是本次运行内序号（每次运行从 1 重数）→ 悬停提示 -->
          <span class="ap-round-no" :title="'本次运行的第 ' + r.round + ' 次监督'">#{{ i + 1 }}</span>
          <span :class="['ap-badge', 'ap-' + badgeKey(r)]">{{ badgeLabel(r) }}</span>
          <span class="ap-meta">
            {{ fmtDuration(r.durationMs) }}<template v-if="r.steps || r.toolCalls"> · {{ r.steps }} 步 / {{ r.toolCalls }} 工具</template><template v-if="tokens(r)"> · {{ tokens(r) }} tokens</template>
          </span>
          <span v-if="r.error" class="ap-err" :title="r.error">异常</span>
        </div>
        <div v-if="r.assessment" class="ap-text">{{ r.assessment }}</div>
        <div v-if="r.nextTask" class="ap-next">
          <span class="ap-next-label">下一步</span>
          <span class="ap-next-text">{{ r.nextTask }}</span>
        </div>
        <details v-if="r.evidence" class="ap-more">
          <summary>证据</summary>
          <pre class="ap-pre">{{ r.evidence }}</pre>
        </details>
        <details v-if="r.trace && r.trace.length" class="ap-more">
          <summary>监督者轨迹（{{ r.trace.length }}）</summary>
          <div v-for="(t, i) in r.trace" :key="i" class="ap-trace">
            <span v-if="t.type === 'tool_call'" class="ap-trace-tool">{{ t.name }}</span>
            <span class="ap-trace-text">{{ t.type === 'tool_call' ? (t.result || t.args) : t.content }}</span>
          </div>
        </details>
        <details v-if="r.workerReport" class="ap-more">
          <summary>工作 agent 侧（{{ r.workerTurns }} 轮）</summary>
          <pre class="ap-pre">{{ r.workerReport }}</pre>
        </details>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  rounds: { type: Array, default: () => [] },
  expanded: { type: Boolean, default: true },
})
defineEmits(['toggle'])

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
/* 与 TaskPanel 同规格（bg/border/字号令牌一致），裁决色沿用既有语义色：
   续跑=#d4a74e（进行中）、收尾=#6a9955（完成）、异常=--danger */
.ap-panel {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
  overflow: hidden;
  flex-shrink: 0;
  margin-top: 4px;
}
.ap-panel.collapsed .ap-body { display: none; }
.ap-header {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 10px; cursor: pointer; user-select: none;
  font-size: 12px; color: var(--text-secondary);
}
.ap-header:hover { background: var(--bg-active); }
.ap-chevron { width: 10px; flex-shrink: 0; display: block; color: var(--text-muted); }
.ap-title { font-weight: 600; flex: 1; color: var(--text-primary); }
.ap-count { font-variant-numeric: tabular-nums; margin-right: 4px; font-size: 11px; }
.ap-badge {
  flex-shrink: 0; font-size: 11px; line-height: 1.5; padding: 0 6px;
  border-radius: 8px; border: 1px solid currentColor;
}
.ap-continue { color: #d4a74e; }
.ap-done { color: #6a9955; }
.ap-error { color: var(--danger, #e5534b); }
.ap-body { border-top: 1px solid var(--border-color); padding: 4px 0; max-height: 260px; overflow-y: auto; }
.ap-round { padding: 6px 10px; font-size: 12px; }
.ap-round + .ap-round { border-top: 1px solid var(--border-color); }
.ap-round-head { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.ap-round-no { color: var(--text-muted); font-variant-numeric: tabular-nums; }
.ap-meta { color: var(--text-muted); font-size: 11px; }
.ap-err { color: var(--danger, #e5534b); font-size: 11px; }
.ap-text {
  margin-top: 4px; color: var(--text-primary); line-height: 1.5;
  word-break: break-word; white-space: pre-wrap;
}
.ap-next {
  margin-top: 4px; display: flex; gap: 6px; align-items: flex-start;
  background: var(--bg-active); border-radius: 3px; padding: 3px 6px;
}
.ap-next-label { flex-shrink: 0; font-size: 11px; color: #d4a74e; }
.ap-next-text { color: var(--text-secondary); line-height: 1.5; word-break: break-word; white-space: pre-wrap; }
.ap-more { margin-top: 4px; }
.ap-more > summary { cursor: pointer; font-size: 11px; color: var(--text-muted); user-select: none; }
.ap-more > summary:hover { color: var(--text-secondary); }
.ap-pre {
  margin: 4px 0 0; padding: 4px 6px; background: var(--bg-primary);
  border-radius: 3px; font-size: 11px; color: var(--text-secondary);
  white-space: pre-wrap; word-break: break-word; max-height: 160px; overflow-y: auto;
}
.ap-trace { display: flex; gap: 6px; padding: 2px 0; font-size: 11px; }
.ap-trace-tool { flex-shrink: 0; color: var(--accent); font-family: var(--font-mono, monospace); }
.ap-trace-text {
  color: var(--text-muted); flex: 1; min-width: 0;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
</style>
