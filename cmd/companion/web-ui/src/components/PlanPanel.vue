<template>
  <div v-if="plan.length > 0" class="plan-panel" :class="{ collapsed: !expanded }">
    <div class="plan-header" @click="$emit('toggle')">
      <span class="plan-chevron">{{ expanded ? '▾' : '▸' }}</span>
      <SvgIcon name="list" :size="12" />
      <span class="plan-title">任务计划</span>
      <span class="plan-progress">{{ doneCount }}/{{ plan.length }}</span>
      <span class="plan-bar">
        <span class="plan-bar-fill" :style="{ width: pct + '%' }"></span>
      </span>
    </div>
    <div v-if="expanded" class="plan-body">
      <div v-for="(step, si) in plan" :key="si" class="plan-step" :class="'step-' + step.status">
        <span class="step-icon">
          <SvgIcon v-if="step.status === 'done'" name="check" :size="12" class="icon-done" />
          <SvgIcon v-else-if="step.status === 'in_progress'" name="cycle" :size="12" class="icon-in-progress" />
          <SvgIcon v-else name="clock" :size="12" class="icon-pending" />
        </span>
        <span class="step-text">{{ si + 1 }}. {{ cleanText(step.step || step.description || step.subject || '') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  plan: { type: Array, default: () => [] },
  expanded: { type: Boolean, default: true },
})
defineEmits(['toggle'])

const doneCount = computed(() => props.plan.filter(s => s.status === 'done').length)
const pct = computed(() => {
  const total = props.plan.length
  return total > 0 ? Math.round(doneCount.value / total * 100) : 0
})

function cleanText(raw) {
  if (!raw) return ''
  return raw.replace(/^(步骤|任务|Step|Task)\s*\d+[. :：、-]*\s*/i, '')
    .replace(/^\d+[. :：、-]+\s*/, '')
}
</script>

<style scoped>
.plan-panel {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
  margin: 4px 8px;
  overflow: hidden;
  flex-shrink: 0;
}
.plan-panel.collapsed .plan-body { display: none; }
.plan-header {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 10px; cursor: pointer; user-select: none;
  font-size: 12px; color: var(--text-secondary);
}
.plan-header:hover { background: var(--bg-active); }
.plan-chevron { width: 10px; text-align: center; font-size: 10px; }
.plan-title { font-weight: 600; flex: 1; color: var(--text-primary); }
.plan-progress { font-variant-numeric: tabular-nums; margin-right: 4px; }
.plan-bar {
  width: 40px; height: 4px; background: var(--border-color); border-radius: 2px; overflow: hidden; display: inline-block; vertical-align: middle;
}
.plan-bar-fill { height: 100%; background: var(--accent); border-radius: 2px; transition: width 0.3s; }
.plan-body { border-top: 1px solid var(--border-color); padding: 4px 0; max-height: 200px; overflow-y: auto; }
.plan-step { display: flex; align-items: flex-start; gap: 6px; padding: 3px 10px; font-size: 12px; }
.plan-step.step-done { opacity: 0.6; }
.plan-step.step-in_progress { background: var(--bg-active); }
.step-icon { flex-shrink: 0; width: 16px; text-align: center; line-height: 1.4; display: flex; align-items: center; justify-content: center; }
.step-text { color: var(--text-primary); line-height: 1.4; word-break: break-word; }
.icon-done { color: var(--accent); }
.icon-in-progress { color: #d4a74e; }
.icon-pending { color: var(--text-muted); }
</style>
