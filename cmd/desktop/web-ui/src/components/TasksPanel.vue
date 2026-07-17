<template>
  <div class="tasks-panel">
    <div class="tasks-toolbar">
      <input type="text" v-model="taskCommand" placeholder="输入命令..."
             @keydown.enter="runTask" class="task-input" />
      <button @click="runTask" class="task-btn"><SvgIcon name="send" :size="12" /> 运行</button>
    </div>
    <div class="tasks-list">
      <div v-for="task in state.tasks" :key="task.id" class="task-item">
        <div class="task-header">
          <span class="task-cmd">{{ task.cmd }}</span>
          <span class="task-status" :class="task.status">{{ task.status }}</span>
          <button v-if="task.status === 'running'" @click="stopTask(task.id)" class="stop-btn"><SvgIcon name="close" :size="10" /></button>
        </div>
        <pre class="task-output">{{ task.output }}</pre>
      </div>
    </div>
    <div v-if="state.tasks.length === 0" class="tasks-empty">运行命令以查看输出</div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { state } from '../main.js'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'

const taskCommand = ref('')

const runTask = async () => {
  if (!taskCommand.value.trim()) return
  const cmd = taskCommand.value.trim()
  taskCommand.value = ''
  try {
    const result = await api.apiPost('/tasks', { command: cmd })
    if (result.id) {
      state.tasks.unshift({
        id: result.id,
        cmd,
        status: 'running',
        output: '',
        pid: result.pid,
      })
      // 简版：一次执行
    } else {
      state.tasks.unshift({
        id: Date.now().toString(),
        cmd,
        status: 'done',
        output: result.output || '(无输出)',
      })
    }
  } catch (err) {
    state.tasks.unshift({
      id: Date.now().toString(),
      cmd,
      status: 'error',
      output: err.message,
    })
  }
}

const stopTask = async (id) => {
  try {
    await api.apiPost(`/tasks/${id}/kill`)
  } catch {}
}
</script>

<style scoped>
.tasks-panel { height: 100%; display: flex; flex-direction: column; }
.tasks-toolbar { display: flex; gap: 4px; padding: 4px; flex-shrink: 0; }
.task-input {
  flex: 1;
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 4px 8px;
  font-size: 12px;
  outline: none;
  font-family: 'Consolas', monospace;
}
.task-input:focus { border-color: var(--accent); }
.task-btn { background: var(--accent); border: none; color: #fff; padding: 4px 10px; cursor: pointer; font-size: 12px; }
.tasks-list { flex: 1; overflow: auto; }
.task-item { margin-bottom: 4px; padding: 4px; background: var(--bg-tertiary); border-radius: 3px; }
.task-header { display: flex; align-items: center; gap: 8px; font-size: 12px; }
.task-cmd { font-family: monospace; color: var(--text-primary); flex: 1; }
.task-status { font-size: 11px; padding: 1px 6px; border-radius: 3px; }
.task-status.running { background: #0a4; color: #fff; }
.task-status.done { background: var(--bg-hover); color: var(--text-secondary); }
.task-status.error { background: #c03; color: #fff; }
.stop-btn { background: none; border: 1px solid #c03; color: #c03; cursor: pointer; padding: 0 4px; }
.task-output {
  font-size: 11px;
  color: var(--text-secondary);
  margin-top: 2px;
  max-height: 100px;
  overflow: auto;
  white-space: pre-wrap;
  font-family: monospace;
}
.tasks-empty { padding: 20px; text-align: center; color: var(--text-muted); font-size: 12px; }
</style>
