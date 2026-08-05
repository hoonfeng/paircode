<template>
  <div class="task-panel" :class="{ collapsed: !expanded }">
    <div class="task-header" @click="$emit('toggle')">
      <svg v-if="!expanded" class="task-chevron" viewBox="0 0 8 8" width="9" height="9" fill="currentColor" aria-hidden="true"><path d="M2.6 1.2 L6.8 4 L2.6 6.8 Z"/></svg>
      <svg v-else class="task-chevron" viewBox="0 0 8 8" width="9" height="9" fill="currentColor" aria-hidden="true"><path d="M1.2 2.6 L4 6.8 L6.8 2.6 Z"/></svg>
      <SvgIcon name="list" :size="12" />
      <span class="task-title">任务进度</span>
      <span class="task-progress">{{ doneCount }}/{{ tasks.length }}</span>
      <span class="task-bar">
        <span class="task-bar-fill" :style="{ width: pct + '%' }"></span>
      </span>
    </div>
    <div v-if="expanded" class="task-body">
      <div v-for="(task, ti) in tasks" :key="task._taskId || ti"
           :class="['task-step', 'task-' + (task.status === 'done' ? 'completed' : task.status)]">
        <span class="task-step-icon">
          <SvgIcon v-if="task.status === 'completed' || task.status === 'done'" name="check" :size="12" class="ti-done" />
          <SvgIcon v-else-if="task.status === 'in_progress'" name="cycle" :size="12" class="ti-run" />
          <SvgIcon v-else name="clock" :size="12" class="ti-pending" />
        </span>
        <span class="task-step-text">{{ ti + 1 }}. {{ task.step || task.subject || task.description || '(无标题)' }}</span>
        <span class="task-step-status">{{ statusLabel(task.status) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  tasks: { type: Array, default: () => [] },
  expanded: { type: Boolean, default: true },
})
defineEmits(['toggle'])

const activeCount = computed(() => props.tasks.filter(t => t.status === 'in_progress').length)
const doneCount = computed(() => props.tasks.filter(t => t.status === 'completed' || t.status === 'done').length)
const pct = computed(() => {
  const total = props.tasks.length
  return total > 0 ? Math.round(doneCount.value / total * 100) : 0
})

function statusLabel(s) {
  if (s === 'completed' || s === 'done') return '完成'
  if (s === 'in_progress') return '进行中'
  if (s === 'cancelled') return '已取消'
  return '待处理'
}
</script>

<style scoped>
.task-panel {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
  overflow: hidden;
  flex-shrink: 0;
  margin-top: 4px;
}
.task-panel.collapsed .task-body { display: none; }
.task-header {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 10px; cursor: pointer; user-select: none;
  font-size: 12px; color: var(--text-secondary);
}
.task-header:hover { background: var(--bg-active); }
.task-chevron { width: 10px; flex-shrink: 0; display: block; color: var(--text-muted); }
.task-title { font-weight: 600; flex: 1; color: var(--text-primary); }
.task-progress { font-variant-numeric: tabular-nums; margin-right: 4px; font-size: 11px; }
.task-bar {
  width: 40px; height: 4px; background: var(--border-color); border-radius: 2px; overflow: hidden; display: inline-block; vertical-align: middle;
}
.task-bar-fill { height: 100%; background: #6a9955; border-radius: 2px; transition: width 0.3s; }
.task-body { border-top: 1px solid var(--border-color); padding: 4px 0; max-height: 200px; overflow-y: auto; }
.task-step { display: flex; align-items: center; gap: 6px; padding: 4px 10px; font-size: 12px; }
.task-step.task-completed { opacity: 0.6; }
.task-step.task-done { opacity: 0.6; }
.task-step.task-in_progress { background: var(--bg-active); }
.task-step-icon { flex-shrink: 0; width: 16px; text-align: center; display: flex; align-items: center; justify-content: center; }
.task-step-text { color: var(--text-primary); line-height: 1.4; word-break: break-word; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.task-step-status { flex-shrink: 0; font-size: 11px; color: var(--text-muted); }
.ti-done { color: var(--accent); }
.ti-run { color: #d4a74e; }
.ti-pending { color: var(--text-muted); }
</style>
