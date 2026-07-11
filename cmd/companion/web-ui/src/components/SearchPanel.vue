<template>
  <div class="search-panel">
    <div class="search-box">
      <input type="text" v-model="query" placeholder="搜索文件内容..."
             @keydown.enter="doSearch" class="search-input" />
      <button @click="doSearch" class="search-btn"><SvgIcon name="search" :size="14" /></button>
    </div>
    <div class="search-path">
      <input type="text" v-model="searchPath" placeholder="搜索路径（默认工作区）" class="path-input" />
    </div>
    <div class="search-results" v-if="state.searchResults.length > 0">
      <div class="result-count">{{ state.searchResults.length }} 个结果</div>
      <div v-for="(r, i) in state.searchResults" :key="i" class="result-item" @click="openResult(r)">
        <span class="result-file">{{ r.file }}</span>
        <span class="result-line">{{ r.line }}</span>
        <span class="result-text">{{ r.text }}</span>
      </div>
    </div>
    <div v-else-if="searched" class="no-results">无结果</div>
  </div>
</template>

<script setup>
import { ref, inject } from 'vue'
import { state } from '../main.js'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'

const query = ref('')
const searchPath = ref('')
const searched = ref(false)
const showSettings = inject('showSettings')
const showSystem = inject('showSystem')
const switchActivity = inject('switchActivity')

const doSearch = async () => {
  if (!query.value.trim()) return
  searched.value = true
  state.searchResults = []
  try {
    const results = await api.apiGet('/fs/search', { q: query.value, path: searchPath.value || state.workspaceRoot })
    state.searchResults = results || []
  } catch (err) {
    state.searchResults = []
    console.error('搜索失败:', err)
  }
}

const openResult = (result) => {
  // 打开文件并跳转到行
  const path = result.file
  const existing = state.openFiles.find(f => f === path)
  if (!existing) state.openFiles.push(path)
  state.activeFile = path
  if (!state.fileContents[path]) {
    api.apiGet('/fs/read', { path }).then(d => {
      state.fileContents[path] = d.content || ''
      state.fileDirty[path] = false
    }).catch(e => console.warn('[搜索] 读取文件失败:', path, e))
  }
}
</script>

<style scoped>
.search-panel { padding: 8px; }
.search-box { display: flex; gap: 4px; }
.search-input, .path-input {
  flex: 1;
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 4px 8px;
  font-size: 13px;
  outline: none;
  border-radius: 3px;
}
.search-input:focus, .path-input:focus { border-color: var(--accent); }
.search-btn {
  background: var(--accent);
  border: none;
  color: var(--status-text);
  padding: 4px 10px;
  cursor: pointer;
  border-radius: var(--border-radius);
}
.search-path { margin-top: 6px; }
.search-results { margin-top: 8px; }
.result-count { font-size: 11px; color: var(--text-muted); margin-bottom: 4px; }
.result-item {
  padding: 3px 4px;
  cursor: pointer;
  font-size: 12px;
  border-radius: 2px;
}
.result-item:hover { background: var(--bg-hover); }
.result-file { color: var(--accent-light); margin-right: 4px; }
.result-line { color: var(--text-muted); margin-right: 4px; font-size: 11px; }
.result-text { color: var(--text-primary); }
.no-results { margin-top: 16px; text-align: center; color: var(--text-muted); }
</style>
