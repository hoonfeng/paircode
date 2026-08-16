<template>
  <div v-if="plan.length > 0" class="plan-panel" :class="{ collapsed: !expanded, 'all-done': allDone }">
    <div class="plan-header" @click="$emit('toggle')">
      <svg v-if="!expanded" class="plan-chevron" viewBox="0 0 8 8" width="9" height="9" fill="currentColor" aria-hidden="true"><path d="M2.6 1.2 L6.8 4 L2.6 6.8 Z"/></svg>
      <svg v-else class="plan-chevron" viewBox="0 0 8 8" width="9" height="9" fill="currentColor" aria-hidden="true"><path d="M1.2 2.6 L4 6.8 L6.8 2.6 Z"/></svg>
      <SvgIcon name="list" :size="12" />
      <span class="plan-title">执行计划</span>
      <span class="plan-progress">{{ doneCount }}/{{ plan.length }}</span>
      <span class="plan-bar">
        <span class="plan-bar-fill" :style="{ width: pct + '%' }"></span>
      </span>
    </div>
    <div v-if="expanded" class="plan-body">
      <div v-for="(step, si) in plan" :key="si" class="plan-step-group">
        <div class="plan-step" :class="'step-' + step.status" @click="toggleStep(si)">
          <svg v-if="!expandedSteps.has(si)" class="step-chevron" viewBox="0 0 8 8" width="8" height="8" fill="currentColor" aria-hidden="true"><path d="M2.6 1.2 L6.8 4 L2.6 6.8 Z"/></svg>
          <svg v-else class="step-chevron" viewBox="0 0 8 8" width="8" height="8" fill="currentColor" aria-hidden="true"><path d="M1.2 2.6 L4 6.8 L6.8 2.6 Z"/></svg>
          <span class="step-icon">
            <SvgIcon v-if="step.status === 'done' || step.status === 'completed'" name="check" :size="12" class="icon-done" />
            <SvgIcon v-else-if="step.status === 'in_progress'" name="cycle" :size="12" class="icon-in-progress" />
            <SvgIcon v-else name="clock" :size="12" class="icon-pending" />
          </span>
          <span class="step-text">{{ si + 1 }}. {{ cleanText(step.step || step.description || step.subject || '') }}</span>
          <span v-if="subTasksByStep[si]" class="step-sub-progress">{{ subDoneByStep[si] }}/{{ subTasksByStep[si].length }}</span>
        </div>
        <div v-if="expandedSteps.has(si) && subTasksByStep[si]" class="sub-tasks">
          <div v-for="(t, ti) in subTasksByStep[si]" :key="t._taskId || t.id || ti" class="sub-task" :class="'task-' + t.status">
            <span class="sub-task-icon">
              <SvgIcon v-if="t.status === 'completed' || t.status === 'done'" name="check" :size="10" class="icon-done" />
              <SvgIcon v-else-if="t.status === 'in_progress'" name="cycle" :size="10" class="icon-run" />
              <SvgIcon v-else name="clock" :size="10" class="icon-pending" />
            </span>
            <span class="sub-task-text">{{ t.subject || t.step || t.description || '(无标题)' }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  plan: { type: Array, default: () => [] },
  tasks: { type: Array, default: () => [] },
  expanded: { type: Boolean, default: true },
})
defineEmits(['toggle'])

// 每个步骤是否展开子任务
const expandedSteps = ref(new Set())

// 当 plan 或 tasks 变化时，自动展开当前 in_progress 步骤
watch([() => props.plan, () => props.tasks], () => {
  const s = new Set()
  props.plan.forEach((step, si) => {
    if (step.status === 'in_progress') s.add(si)
  })
  // 如果没有任何 in_progress 步骤，展开第一个
  if (s.size === 0 && props.plan.length > 0) s.add(0)
  expandedSteps.value = s
}, { immediate: true })

function toggleStep(si) {
  const s = new Set(expandedSteps.value)
  if (s.has(si)) s.delete(si)
  else s.add(si)
  expandedSteps.value = s
}

// 按 planStepIndex 分组子任务
const subTasksByStep = computed(() => {
  const groups = {}
  for (const t of props.tasks) {
    const idx = t.planStepIndex ?? t.plan_step_index
    if (idx !== undefined && idx !== null) {
      if (!groups[idx]) groups[idx] = []
      groups[idx].push(t)
    }
  }
  return groups
})

const subDoneByStep = computed(() => {
  const done = {}
  for (const [si, tasks] of Object.entries(subTasksByStep.value)) {
    done[si] = tasks.filter(t => t.status === 'completed' || t.status === 'done').length
  }
  return done
})

const doneCount = computed(() => props.plan.filter(s => s.status === 'done' || s.status === 'completed').length)
const allDone = computed(() => props.plan.length > 0 && doneCount.value === props.plan.length)
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
  overflow: hidden;
  flex-shrink: 0;
}
.plan-panel.collapsed .plan-body { display: none; }
.plan-panel.all-done { opacity: 0.7; }
.plan-panel.all-done .plan-bar-fill { background: #6a9955; }
.plan-panel.all-done .plan-header .plan-progress::after { content: ' 全部完成'; font-size: 10px; color: #6a9955; }
.plan-header {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 10px; cursor: pointer; user-select: none;
  font-size: 12px; color: var(--text-secondary);
}
.plan-header:hover { background: var(--bg-active); }
.plan-chevron { width: 10px; flex-shrink: 0; display: block; color: var(--text-muted); }
.plan-title { font-weight: 600; flex: 1; color: var(--text-primary); }
.plan-progress { font-variant-numeric: tabular-nums; margin-right: 4px; }
.plan-bar {
  width: 40px; height: 4px; background: var(--border-color); border-radius: 2px; overflow: hidden; display: inline-block; vertical-align: middle;
}
.plan-bar-fill { height: 100%; background: var(--accent); border-radius: 2px; transition: width 0.3s; }
.plan-body { border-top: 1px solid var(--border-color); padding: 4px 0; max-height: 300px; overflow-y: auto; }
.plan-step-group { }
.plan-step { display: flex; align-items: flex-start; gap: 5px; padding: 4px 10px; font-size: 12px; cursor: pointer; }
.plan-step:hover { background: var(--bg-active); }
.plan-step.step-done { opacity: 0.7; }
.plan-step.step-completed { opacity: 0.7; }
.plan-step.step-in_progress { background: var(--bg-active); }
.step-chevron { width: 10px; flex-shrink: 0; display: block; color: var(--text-muted); }
.step-icon { flex-shrink: 0; width: 14px; text-align: center; line-height: 1.4; display: flex; align-items: center; justify-content: center; }
.step-text { color: var(--text-primary); line-height: 1.4; word-break: break-word; flex: 1; min-width: 0; }
.step-sub-progress { flex-shrink: 0; font-size: 10px; color: var(--text-muted); font-variant-numeric: tabular-nums; }
/* 子任务（缩进显示） */
.sub-tasks { border-top: 1px solid var(--border-color-subtle, rgba(128,128,128,0.15)); }
.sub-task { display: flex; align-items: center; gap: 4px; padding: 3px 10px 3px 38px; font-size: 11px; opacity: 0.85; }
.sub-task.task-completed, .sub-task.task-done { opacity: 0.55; text-decoration: line-through; }
.sub-task.task-in_progress { background: var(--bg-active); opacity: 1; }
.sub-task-icon { flex-shrink: 0; width: 14px; text-align: center; }
.sub-task-text { color: var(--text-secondary); line-height: 1.3; word-break: break-word; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.icon-done { color: var(--accent); }
.icon-in-progress { color: #d4a74e; }
.icon-run { color: #d4a74e; }
.icon-pending { color: var(--text-muted); }
</style>
