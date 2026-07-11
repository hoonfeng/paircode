<template>
  <div class="conv-sidebar" :style="{ width: width + 'px' }">
    <div class="conv-sidebar-header">
      <span>会话</span>
      <button class="rp-btn" @click="$emit('new-conversation')" title="新对话"><SvgIcon name="plus" :size="12" /></button>
    </div>
    <div class="conv-list">
      <div v-for="conv in conversations" :key="conv.id"
           :class="['conv-item', { active: conv.id === currentConvId }]"
           @click="$emit('switch-conversation', conv.id)">
        <div class="conv-title">{{ conv.title }}</div>
        <div class="conv-meta">
          <span class="conv-msg-count">{{ conv.msgCount || 0 }}</span>
          <span class="conv-time">{{ conv.updatedAt ? conv.updatedAt.slice(11,16) : '' }}</span>
        </div>
        <button class="conv-del" @click.stop="$emit('delete-conversation', conv.id)" title="删除对话">×</button>
      </div>
      <div v-if="conversations.length === 0" class="conv-empty">暂无对话</div>
    </div>

    <!-- Token 统计面板 -->
    <div class="conv-stats cs-tokens">
      <div class="conv-stats-header" @click="toggleTokens">
        <span class="conv-stats-chevron">{{ convStatsExpanded ? '▾' : '▸' }}</span>
        <SvgIcon name="code" :size="11" />
        <span>Token 统计</span>
        <span class="conv-stats-total">{{ shortTokens(wsTokenStats.totalTokens) }}</span>
      </div>
      <div v-if="convStatsExpanded" class="conv-stats-body">
        <div class="cache-ring-wrap">
          <svg class="cache-ring" viewBox="0 0 48 48" width="96" height="96">
            <circle cx="24" cy="24" r="18" fill="none" stroke="var(--border-color)" stroke-width="4" />
            <circle cx="24" cy="24" r="18" fill="none" stroke="#6a9955" stroke-width="4"
              :stroke-dasharray="cacheRingDash" stroke-linecap="round"
              transform="rotate(-90 24 24)" style="transition: stroke-dasharray 0.3s;" />
          </svg>
          <div class="cache-ring-label">
            <span class="cache-ring-pct">{{ cacheRate || 0 }}%</span>
            <span class="cache-ring-text">缓存命中</span>
          </div>
        </div>
        <div class="conv-stats-detail">
          <div class="cs-row">
            <span class="cs-label">输入</span>
            <span class="cs-val cs-prompt">{{ shortTokens(wsTokenStats.promptTokens) }}</span>
          </div>
          <div class="cs-row">
            <span class="cs-label cs-cachelbl">● 缓存命中</span>
            <span class="cs-val cs-cache">{{ shortTokens(wsTokenStats.cacheHitTokens) }}</span>
          </div>
          <div class="cs-row">
            <span class="cs-label cs-misslbl">● 缓存未命中</span>
            <span class="cs-val cs-miss">{{ shortTokens(wsTokenStats.cacheMissTokens) }}</span>
          </div>
          <div class="cs-row">
            <span class="cs-label">输出</span>
            <span class="cs-val cs-out">{{ shortTokens(wsTokenStats.completionTokens) }}</span>
          </div>
          <div class="cs-divider"></div>
          <div class="cs-row cs-total">
            <span class="cs-label">总 Token</span>
            <span class="cs-val">{{ shortTokens(wsTokenStats.totalTokens) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 上下文窗口 + 构成占比 -->
    <div class="conv-stats cs-context">
      <div class="conv-stats-header" @click="toggleCtx">
        <span class="conv-stats-chevron">{{ ctxStatsExpanded ? '▾' : '▸' }}</span>
        <SvgIcon name="layers" :size="11" />
        <span>上下文</span>
        <span class="conv-stats-pct">{{ ctxUsagePct }}%</span>
      </div>
      <div v-if="ctxStatsExpanded" class="conv-stats-body ctx-body">
        <div class="ctx-bar-wrap">
          <div class="ctx-bar">
            <div class="ctx-bar-fill" :style="{ width: ctxUsagePct + '%' }"></div>
          </div>
          <div class="ctx-bar-labels">
            <span>已用: {{ shortTokens(convCtxStats.promptTokens) }}</span>
            <span>上限: {{ shortTokens(ctxMaxTokens) }}</span>
          </div>
        </div>
        <div class="ctx-detail">
          <div class="ctx-row">
            <span class="cs-label">上下文大小</span>
            <span class="cs-val">{{ shortTokens(convCtxStats.promptTokens) }} / {{ shortTokens(ctxMaxTokens) }}</span>
          </div>
          <div class="ctx-row">
            <span class="cs-label">占用比例</span>
            <span class="cs-val" :class="ctxPctClass">{{ ctxUsagePct }}%</span>
          </div>
          <div class="ctx-row">
            <span class="cs-label">剩余空间</span>
            <span class="cs-val cs-remain">{{ shortTokens(ctxRemaining) }}</span>
          </div>
        </div>
        <div class="comp-bar-wrap">
          <div class="comp-bar-title">上下文构成</div>
          <div class="comp-bar">
            <div v-if="convCtxStats.systemTokens > 0" class="comp-bar-seg comp-system" :style="{ width: compSystemPct + '%' }"
                 :title="'提示词: ' + shortTokens(convCtxStats.systemTokens)"></div>
            <div v-if="convCtxStats.skillsTokens > 0" class="comp-bar-seg comp-skills" :style="{ width: compSkillsPct + '%' }"
                 :title="'技能: ' + shortTokens(convCtxStats.skillsTokens)"></div>
            <div v-if="convCtxStats.mcpTokens > 0" class="comp-bar-seg comp-mcp" :style="{ width: compMCPPct + '%' }"
                 :title="'MCP: ' + shortTokens(convCtxStats.mcpTokens)"></div>
            <div v-if="convCtxStats.toolTokens > 0" class="comp-bar-seg comp-tool" :style="{ width: compToolPct + '%' }"
                 :title="'工具: ' + shortTokens(convCtxStats.toolTokens)"></div>
            <div v-if="convCtxStats.historyTokens > 0" class="comp-bar-seg comp-history" :style="{ width: compHistoryPct + '%' }"
                 :title="'历史: ' + shortTokens(convCtxStats.historyTokens)"></div>
            <div v-if="convCtxStats.otherTokens > 0" class="comp-bar-seg comp-other" :style="{ width: compOtherPct + '%' }"
                 :title="'其他: ' + shortTokens(convCtxStats.otherTokens)"></div>
          </div>
          <div class="comp-legend">
            <span v-if="convCtxStats.systemTokens > 0" class="comp-leg-item"><span class="leg-dot comp-system-dot"></span>提示词 {{ shortTokens(convCtxStats.systemTokens) }}</span>
            <span v-if="convCtxStats.skillsTokens > 0" class="comp-leg-item"><span class="leg-dot comp-skills-dot"></span>技能 {{ shortTokens(convCtxStats.skillsTokens) }}</span>
            <span v-if="convCtxStats.mcpTokens > 0" class="comp-leg-item"><span class="leg-dot comp-mcp-dot"></span>MCP {{ shortTokens(convCtxStats.mcpTokens) }}</span>
            <span v-if="convCtxStats.toolTokens > 0" class="comp-leg-item"><span class="leg-dot comp-tool-dot"></span>工具 {{ shortTokens(convCtxStats.toolTokens) }}</span>
            <span v-if="convCtxStats.historyTokens > 0" class="comp-leg-item"><span class="leg-dot comp-history-dot"></span>历史 {{ shortTokens(convCtxStats.historyTokens) }}</span>
            <span v-if="convCtxStats.otherTokens > 0" class="comp-leg-item"><span class="leg-dot comp-other-dot"></span>其他 {{ shortTokens(convCtxStats.otherTokens) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  conversations: { type: Array, default: () => [] },
  currentConvId: { type: String, default: '' },
  wsTokenStats: { type: Object, default: () => ({ totalTokens: 0, promptTokens: 0, completionTokens: 0, cacheHitTokens: 0, cacheMissTokens: 0 }) },
  convCtxStats: { type: Object, default: () => ({ promptTokens: 0, completionTokens: 0, cacheHitTokens: 0, cacheMissTokens: 0, systemTokens: 0, skillsTokens: 0, mcpTokens: 0, toolTokens: 0, historyTokens: 0, otherTokens: 0 }) },
  ctxMaxTokensVal: { type: Number, default: 64000 },
  width: { type: Number, default: 250 },
})

defineEmits(['new-conversation', 'switch-conversation', 'delete-conversation'])

const convStatsExpanded = ref(true)
const ctxStatsExpanded = ref(true)

function toggleTokens() { convStatsExpanded.value = !convStatsExpanded.value }
function toggleCtx() { ctxStatsExpanded.value = !ctxStatsExpanded.value }

// ── Token 工具 ──
function shortTokens(n) {
  if (n >= 999950) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K'
  return String(n)
}

const cacheRate = computed(() => {
  const stats = props.wsTokenStats
  const denom = stats.cacheHitTokens + stats.cacheMissTokens
  if (denom > 0) {
    return ((stats.cacheHitTokens / denom) * 100).toFixed(1)
  }
  return 0
})

const cacheRingDash = computed(() => {
  const stats = props.wsTokenStats
  const denom = stats.cacheHitTokens + stats.cacheMissTokens
  if (denom <= 0) return '0 113.1'
  const ratio = stats.cacheHitTokens / denom
  const circ = 2 * Math.PI * 18 // r=18 → ≈113.1
  const hit = circ * ratio
  const miss = circ - hit
  return `${hit} ${miss}`
})

// ── 上下文计算 ──
const ctxMaxTokens = computed(() => {
  return (props.ctxMaxTokensVal && props.ctxMaxTokensVal > 0) ? props.ctxMaxTokensVal : 64000
})
const ctxUsagePct = computed(() => {
  if (!ctxMaxTokens.value) return 0
  const prompt = props.convCtxStats.promptTokens
  if (prompt <= 0) return 0
  return Math.min(100, Math.round((prompt / ctxMaxTokens.value) * 100))
})
const ctxRemaining = computed(() => {
  return Math.max(0, ctxMaxTokens.value - props.convCtxStats.promptTokens)
})
const ctxPctClass = computed(() => {
  const pct = ctxUsagePct.value
  if (pct >= 90) return 'cs-danger'
  if (pct >= 70) return 'cs-warn'
  return 'cs-safe'
})

const compTotalCtx = computed(() => {
  const s = props.convCtxStats
  return s.systemTokens + s.skillsTokens + s.mcpTokens +
    s.toolTokens + s.historyTokens + s.otherTokens || 1
})
const compSystemPct = computed(() => ((props.convCtxStats.systemTokens / compTotalCtx.value) * 100).toFixed(1))
const compSkillsPct = computed(() => ((props.convCtxStats.skillsTokens / compTotalCtx.value) * 100).toFixed(1))
const compMCPPct = computed(() => ((props.convCtxStats.mcpTokens / compTotalCtx.value) * 100).toFixed(1))
const compToolPct = computed(() => ((props.convCtxStats.toolTokens / compTotalCtx.value) * 100).toFixed(1))
const compHistoryPct = computed(() => ((props.convCtxStats.historyTokens / compTotalCtx.value) * 100).toFixed(1))
const compOtherPct = computed(() => ((props.convCtxStats.otherTokens / compTotalCtx.value) * 100).toFixed(1))
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
  padding: 4px 0;
  min-height: 0;
}
.conv-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px 8px 12px;
  cursor: pointer;
  font-size: 12px;
  color: var(--text-secondary);
  border-left: 3px solid transparent;
  position: relative;
  transition: background 0.12s, border-color 0.12s;
  margin: 1px 4px;
  border-radius: 4px;
}
.conv-item.active {
  background: var(--bg-active);
  color: var(--text-primary);
  border-left-color: var(--accent);
  font-weight: 500;
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
.conv-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
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
.conv-time {
  font-size: 10px;
  color: var(--text-muted);
  flex-shrink: 0;
  opacity: 0.7;
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
.conv-del:hover { opacity: 1; color: #c03; }
.conv-empty {
  padding: 16px 8px;
  font-size: 11px;
  color: var(--text-muted);
  text-align: center;
  font-style: italic;
}

/* ── 统计/上下文面板 ── */
.conv-stats {
  border-top: 1px solid var(--border-color);
  flex-shrink: 0;
  user-select: none;
  background: var(--bg-tertiary);
}
.conv-stats-header {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
  font-size: 11px;
  color: var(--text-secondary);
  cursor: pointer;
}
.conv-stats-header:hover { background: var(--bg-hover); }
.conv-stats-chevron { font-size: 10px; color: var(--text-muted); width: 10px; text-align: center; }
.conv-stats-total {
  margin-left: auto;
  font-family: var(--font-code);
  font-size: 12px;
  color: var(--accent);
  font-weight: 700;
}
.conv-stats-pct {
  margin-left: auto;
  font-family: var(--font-code);
  font-size: 11px;
  color: var(--accent-light);
  font-weight: 600;
}
.conv-stats-body {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px 10px;
}
.ctx-body {
  flex-direction: column;
  align-items: stretch;
  gap: 10px;
}

/* ── 环形缓存图 ── */
.cache-ring-wrap {
  position: relative;
  flex-shrink: 0;
  width: 96px;
  height: 96px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.cache-ring { display: block; }
.cache-ring-label {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
  line-height: 1.3;
}
.cache-ring-pct {
  font-size: 18px;
  font-weight: 700;
  color: #6a9955;
  font-family: var(--font-code);
}
.cache-ring-text { font-size: 10px; color: var(--text-muted); }

/* ── 数字明细 ── */
.conv-stats-detail {
  flex: 1;
  min-width: 0;
  font-size: 12px;
}
.cs-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 2px 0;
}
.cs-label { color: var(--text-muted); font-size: 11px; }
.cs-cachelbl { color: #6a9955; font-size: 11px; }
.cs-misslbl { color: #c586c0; font-size: 11px; }
.cs-val {
  font-family: var(--font-code);
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 600;
}
.cs-prompt { color: var(--text-primary); }
.cs-cache { color: #6a9955; }
.cs-miss { color: #c586c0; }
.cs-out { color: var(--accent-light); }
.cs-divider { height: 1px; background: var(--border-color); margin: 4px 0; }
.cs-total .cs-label { font-weight: 600; color: var(--text-primary); font-size: 12px; }
.cs-total .cs-val { color: var(--accent); font-weight: 700; font-size: 14px; }
.cs-safe { color: #6a9955; }
.cs-warn { color: #d4a74e; }
.cs-danger { color: #c03; }

/* ── 上下文窗口条 ── */
.ctx-bar-wrap { display: flex; flex-direction: column; gap: 4px; }
.ctx-bar {
  height: 12px;
  background: var(--bg-primary);
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid var(--border-color);
}
.ctx-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #6a9955, #d4a74e, #c03);
  border-radius: 6px;
  transition: width 0.4s ease;
  min-width: 4px;
}
.ctx-bar-labels {
  display: flex;
  justify-content: space-between;
  font-size: 10px;
  color: var(--text-muted);
  font-family: var(--font-code);
}
.ctx-detail { font-size: 12px; display: flex; flex-direction: column; gap: 2px; }
.ctx-row { display: flex; justify-content: space-between; align-items: center; padding: 1px 0; }
.ctx-row .cs-val { font-size: 12px; }
.cs-remain { color: var(--accent-light); }

/* ── 构成占比横条 ── */
.comp-bar-wrap { margin-top: 8px; padding-top: 6px; border-top: 1px solid var(--border-color); }
.comp-bar-title { font-size: 10px; color: var(--text-muted); margin-bottom: 4px; text-transform: uppercase; letter-spacing: 0.3px; }
.comp-bar { display: flex; height: 10px; background: var(--bg-primary); border-radius: 5px; overflow: hidden; border: 1px solid var(--border-color); }
.comp-bar-seg { height: 100%; transition: width 0.3s ease; min-width: 2px; }
.comp-system { background: var(--accent); opacity: 0.7; }
.comp-skills { background: #6a9955; }
.comp-mcp { background: #c586c0; }
.comp-tool { background: var(--accent-light); }
.comp-history { background: #d4a74e; }
.comp-other { background: #888; }
.comp-legend { display: flex; flex-wrap: wrap; gap: 4px 8px; margin-top: 5px; font-size: 10px; color: var(--text-muted); }
.comp-leg-item { display: flex; align-items: center; gap: 3px; }
.leg-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.comp-system-dot { background: var(--accent); opacity: 0.7; }
.comp-skills-dot { background: #6a9955; }
.comp-mcp-dot { background: #c586c0; }
.comp-tool-dot { background: var(--accent-light); }
.comp-history-dot { background: #d4a74e; }
.comp-other-dot { background: #888; }

/* ── rp-btn 复用 ── */
.rp-btn {
  background: none; border: 1px solid transparent; color: var(--text-secondary);
  padding: 2px 6px; cursor: pointer; border-radius: 3px; display: flex; align-items: center;
}
.rp-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
</style>
