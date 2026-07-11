<template>
  <div v-if="waiting" class="approval-bar">
    <div class="approval-bar-icon"><SvgIcon name="shield" :size="14" /></div>
    <div class="approval-bar-info">
      <span class="approval-bar-tool">{{ tool }}</span>
      <span class="approval-bar-args">{{ displayArgs }}</span>
    </div>
    <div class="approval-bar-actions">
      <button class="approval-btn approval-btn-allow" @click="$emit('resolve', true)">允许</button>
      <button class="approval-btn approval-btn-deny" @click="$emit('resolve', false)">拒绝</button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  waiting: { type: Boolean, default: false },
  tool: { type: String, default: '' },
  args: { type: String, default: '' },
})
defineEmits(['resolve'])

const displayArgs = computed(() => {
  const s = props.args || ''
  return s.length > 80 ? s.slice(0, 80) + '…' : s
})
</script>

<style scoped>
.approval-bar {
  display: flex; align-items: center; gap: 6px;
  margin: 4px 8px; padding: 6px 10px;
  background: var(--bg-warning, #fff3cd); border: 1px solid var(--border-warning, #ffc107);
  border-radius: var(--border-radius); flex-shrink: 0;
}
.approval-bar-icon { flex-shrink: 0; color: #cc7b1e; }
.approval-bar-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.approval-bar-tool { font-size: 12px; font-weight: 600; color: var(--text-primary); }
.approval-bar-args { font-size: 11px; color: var(--text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.approval-bar-actions { display: flex; gap: 4px; flex-shrink: 0; }
.approval-btn {
  padding: 4px 12px; border: none; border-radius: 3px; font-size: 12px; cursor: pointer;
  transition: opacity 0.15s;
}
.approval-btn:hover { opacity: 0.85; }
.approval-btn-allow { background: #2ea043; color: #fff; }
.approval-btn-deny { background: #da3633; color: #fff; }
</style>
