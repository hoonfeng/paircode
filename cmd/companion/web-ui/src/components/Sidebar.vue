<template>
  <div class="sidebar" :style="{ width: '280px' }">
    <div class="sidebar-header">
      <span>{{ headerTitle }}</span>
    </div>
    <div class="sidebar-content">
      <FileExplorer v-if="state.activeActivity === 'explorer'" />
      <SearchPanel v-else-if="state.activeActivity === 'search'" />
      <GitPanel v-else-if="state.activeActivity === 'source'" />
      <div v-else class="sidebar-placeholder">
        <span>面板加载中...</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { state } from '../main.js'
import FileExplorer from './FileExplorer.vue'
import SearchPanel from './SearchPanel.vue'
import GitPanel from './GitPanel.vue'

const headerTitle = computed(() => {
  const titles = { explorer: '文件浏览器', search: '搜索', source: '源代码管理' }
  return titles[state.activeActivity] || ''
})
</script>

<style scoped>
.sidebar {
  background: var(--sidebar-bg);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.sidebar-header {
  height: 32px;
  display: flex;
  align-items: center;
  padding: 0 12px;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}
.sidebar-content {
  flex: 1;
  overflow: auto;
}
.sidebar-placeholder {
  padding: 20px;
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
}
</style>
