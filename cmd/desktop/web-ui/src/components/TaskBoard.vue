<template>
  <div class="task-board">
    <div class="task-header">
      <h2>任务看板</h2>
      <button class="back-btn" @click="$router.push('/')">返回</button>
    </div>
    <p class="hint">任务列表管理界面 — 可查看各任务的进度和状态</p>
    <div class="task-columns">
      <div class="task-column">
        <h3>待办</h3>
        <div class="task-card-placeholder">
          <p>暂无待办任务</p>
        </div>
      </div>
      <div class="task-column">
        <h3>进行中</h3>
        <div class="task-card-placeholder">
          <p>暂无进行中的任务</p>
        </div>
      </div>
      <div class="task-column">
        <h3>已完成</h3>
        <div class="task-card-placeholder">
          <p>暂无已完成的任务</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
</script>

<style scoped>
.task-board {
  padding: 24px;
  max-width: 900px;
  margin: 0 auto;
}
.task-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}
.task-header h2 { margin: 0; font-size: 20px; }
.back-btn {
  background: none;
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  padding: 4px 12px;
  border-radius: 4px;
  cursor: pointer;
}
.back-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.hint { color: var(--text-muted); font-size: 13px; margin-bottom: 24px; }
.task-columns {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}
.task-column {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 16px;
  min-height: 200px;
}
.task-column h3 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: var(--text-primary);
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-color);
}
.task-card-placeholder {
  color: var(--text-muted);
  font-size: 13px;
  text-align: center;
  padding: 20px 0;
}
</style>
