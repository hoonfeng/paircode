<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content sys-modal">
      <div class="modal-header">
        <h2>ℹ 系统信息</h2>
        <button class="modal-close" @click="$emit('close')">×</button>
      </div>
      <div class="modal-body">
        <div v-if="loading" class="loading">加载中...</div>
        <div v-else class="sys-info">
          <div class="info-row"><label>主机名</label><span>{{ info.hostname }}</span></div>
          <div class="info-row"><label>当前目录</label><span>{{ info.cwd }}</span></div>
          <div class="info-row"><label>操作系统</label><span>{{ info.os }}</span></div>
          <div class="info-row"><label>Go 版本</label><span>{{ info.goos }}</span></div>
          <div class="info-row"><label>工作区</label><span>{{ info.workspace }}</span></div>
          <div class="info-row"><label>文件夹</label><span>{{ (info.folders || []).join(', ') }}</span></div>
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn-secondary" @click="$emit('close')">关闭</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api.js'

const emit = defineEmits(['close'])
const loading = ref(true)
const info = ref({})

onMounted(async () => {
  try {
    info.value = await api.apiGet('/system/info')
  } catch {}
  loading.value = false
})
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal-content {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  width: 500px;
  max-height: 60vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
}
.modal-header h2 { font-size: 16px; }
.modal-close { background: none; border: none; color: var(--text-secondary); font-size: 20px; cursor: pointer; }
.modal-body { flex: 1; overflow: auto; padding: 16px; }
.loading { text-align: center; color: var(--text-muted); }
.info-row { display: flex; padding: 6px 0; border-bottom: 1px solid var(--border-color); }
.info-row label { width: 100px; color: var(--text-secondary); font-size: 13px; flex-shrink: 0; }
.info-row span { color: var(--text-primary); font-size: 13px; word-break: break-all; }
.modal-footer { padding: 10px 16px; border-top: 1px solid var(--border-color); display: flex; justify-content: flex-end; }
.btn-secondary {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 6px 16px;
  cursor: pointer;
  border-radius: 3px;
}
</style>
