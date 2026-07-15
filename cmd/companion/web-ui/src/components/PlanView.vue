<template>
  <div class="plan-view">
    <div class="plan-header">
      <h2>计划详情</h2>
      <button class="back-btn" @click="$router.push('/')">返回</button>
    </div>
    <div v-if="planId" class="plan-content">
      <p>计划 ID：<code>{{ planId }}</code></p>
      <p class="hint">此页面将展示计划的完整步骤和进度</p>
    </div>
    <div v-else class="plan-empty">
      <p>请选择一个计划</p>
    </div>
  </div>
</template>

<script setup>
import { useRoute } from 'vue-router'
const route = useRoute()
const planId = route.params?.planId
</script>

<style scoped>
.plan-view {
  padding: 24px;
  max-width: 800px;
  margin: 0 auto;
}
.plan-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
}
.plan-header h2 { margin: 0; font-size: 20px; }
.back-btn {
  background: none;
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  padding: 4px 12px;
  border-radius: 4px;
  cursor: pointer;
}
.back-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.plan-content code {
  background: var(--bg-tertiary);
  padding: 2px 6px;
  border-radius: 3px;
  font-family: var(--font-code);
}
.hint { color: var(--text-muted); font-size: 13px; }
.plan-empty { color: var(--text-muted); text-align: center; margin-top: 40px; }
</style>
