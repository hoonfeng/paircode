<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="$emit('close')">
      <div class="modal-container" :style="{ maxWidth: maxWidth }">
        <div class="modal-header">
          <span class="modal-title"><slot name="title">提示</slot></span>
          <button class="modal-close" @click="$emit('close')">
            <SvgIcon name="close" :size="14" />
          </button>
        </div>
        <div class="modal-body">
          <slot />
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import SvgIcon from './SvgIcon.vue'

defineProps({
  maxWidth: { type: String, default: '480px' },
})

const emit = defineEmits(['close'])
const visible = ref(true)

function handleKeydown(e) {
  if (e.key === 'Escape') emit('close')
}
onMounted(() => document.addEventListener('keydown', handleKeydown))
onUnmounted(() => document.removeEventListener('keydown', handleKeydown))
</script>

<style scoped>
.modal-overlay {
  position: fixed; top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.5); z-index: 2000;
  display: flex; align-items: center; justify-content: center;
}
.modal-container {
  background: var(--bg-primary); border: 1px solid var(--border-color);
  border-radius: 8px; min-width: 320px; max-width: 520px;
  box-shadow: 0 8px 32px rgba(0,0,0,0.4);
  display: flex; flex-direction: column; max-height: 80vh;
}
.modal-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 16px; border-bottom: 1px solid var(--border-color);
}
.modal-title { font-size: 13px; font-weight: 600; color: var(--text-primary); }
.modal-close {
  display: flex; align-items: center; justify-content: center;
  width: 28px; height: 28px; cursor: pointer; border: none; background: none;
  color: var(--text-muted); border-radius: 4px;
}
.modal-close:hover { background: var(--bg-hover); color: var(--text-primary); }
.modal-body { flex: 1; overflow-y: auto; }
</style>
