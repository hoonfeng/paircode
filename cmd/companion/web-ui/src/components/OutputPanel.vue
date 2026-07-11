<template>
  <div class="output-panel">
    <div class="output-header">
      <span>应用输出日志</span>
      <button class="clear-btn" @click="logs = []">清除</button>
    </div>
    <div class="output-body" ref="outputRef">
      <div v-for="(log, i) in logs" :key="i" class="log-line">[{{ log.time }}] {{ log.text }}</div>
      <div v-if="logs.length === 0" class="log-empty">暂无日志</div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'

const logs = ref([])
const outputRef = ref(null)

const addLog = (text) => {
  const now = new Date()
  const time = now.toLocaleTimeString()
  logs.value.push({ time, text })
  nextTick(() => {
    if (outputRef.value) outputRef.value.scrollTop = outputRef.value.scrollHeight
  })
}

// 初始日志
onMounted(() => {
  addLog('PairCode Web IDE 已启动')
  addLog('后端: http://localhost:9090')
})

// 暴露 addLog 供全局使用
window.__outputLog = addLog
</script>

<style scoped>
.output-panel { height: 100%; display: flex; flex-direction: column; }
.output-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 2px 8px;
  font-size: 11px;
  color: var(--text-muted);
  flex-shrink: 0;
}
.clear-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 11px;
}
.clear-btn:hover { color: var(--text-primary); }
.output-body {
  flex: 1;
  overflow: auto;
  padding: 4px 8px;
  font-family: 'Consolas', monospace;
  font-size: 12px;
}
.log-line { color: var(--text-secondary); padding: 1px 0; }
.log-empty { color: var(--text-muted); font-style: italic; }
</style>
