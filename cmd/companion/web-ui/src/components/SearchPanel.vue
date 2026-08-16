<template>
  <div class="search-panel">
    <!-- 模式切换：搜索 / 替换 -->
    <div class="sp-mode-bar">
      <button :class="['sp-mode-btn', { active: mode === 'search' }]" @click="mode = 'search'">查找</button>
      <button :class="['sp-mode-btn', { active: mode === 'replace' }]" @click="mode = 'replace'">替换</button>
    </div>

    <!-- 搜索输入 -->
    <div class="sp-field">
      <div class="sp-input-wrap">
        <SvgIcon name="search" :size="13" class="sp-input-icon" />
        <input type="text" v-model="query" placeholder="搜索..."
               @keydown.enter="doSearch" class="sp-input" />
      </div>
      <button @click="doSearch" class="sp-go-btn">查找</button>
    </div>

    <!-- 替换输入（替换模式下显示）-->
    <div v-if="mode === 'replace'" class="sp-field">
      <div class="sp-input-wrap">
        <SvgIcon name="edit" :size="13" class="sp-input-icon" />
        <input type="text" v-model="replaceText" placeholder="替换为..."
               @keydown.enter="replaceAll" class="sp-input" />
      </div>
      <button @click="replaceAll" class="sp-replace-btn"
              :disabled="!state.searchResults.length || !query.trim()">全部替换</button>
    </div>

    <!-- 搜索路径 -->
    <div class="sp-path-row">
      <input type="text" v-model="searchPath" placeholder="搜索路径（默认工作区）" class="sp-path-input" />
      <button v-if="searchPath" class="sp-clear-btn" @click="searchPath = ''" title="清除路径">×</button>
    </div>

    <!-- 选项行 -->
    <div class="sp-options">
      <label class="sp-opt" title="区分大小写">
        <input type="checkbox" v-model="caseSensitive" />
        <span>Aa</span>
      </label>
      <label class="sp-opt" title="全词匹配">
        <input type="checkbox" v-model="wholeWord" />
        <span>全词</span>
      </label>
      <label class="sp-opt" title="使用正则表达式">
        <input type="checkbox" v-model="useRegex" />
        <span>正则</span>
      </label>
    </div>

    <!-- 结果区域 -->
    <div class="sp-results" v-if="state.searchResults.length > 0">
      <div class="sp-result-header">
        <span class="sp-result-count">{{ state.searchResults.length }} 个文件匹配</span>
        <button v-if="mode === 'replace'" class="sp-replace-all-sm" @click="replaceAll">全部替换</button>
      </div>
      <!-- 按文件分组 -->
      <div v-for="(group, gi) in groupedResults" :key="gi" class="sp-file-group">
        <div class="sp-file-title" @click="toggleGroup(gi)">
          <SvgIcon :name="group.expanded ? 'chevron-down' : 'chevron-right'" :size="10" />
          <span class="sp-file-path">{{ group.file }}</span>
          <span class="sp-file-count">{{ group.items.length }} 处</span>
        </div>
        <div v-if="group.expanded" class="sp-file-items">
          <div v-for="(r, ri) in group.items" :key="ri" class="sp-result-row"
               @click="openResult(r)" :title="r.text">
            <span class="sp-result-line">{{ r.line }}</span>
            <span class="sp-result-text" v-html="highlightMatch(r.text, query)"></span>
          </div>
        </div>
      </div>
    </div>
    <div v-else-if="searched" class="sp-no-results">
      <SvgIcon name="search-off" :size="20" class="sp-no-icon" />
      <span>未找到匹配内容</span>
    </div>
    <div v-else class="sp-hint">
      <span>输入关键词后按 Enter 搜索</span>
    </div>

    <!-- 替换状态提示 -->
    <div v-if="replaceStatus" :class="['sp-status', replaceStatus.type]">
      {{ replaceStatus.message }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed, inject } from 'vue'
import { state } from '../ui-state.js'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'

const query = ref('')
const replaceText = ref('')
const searchPath = ref('')
const searched = ref(false)
const caseSensitive = ref(false)
const wholeWord = ref(false)
const useRegex = ref(false)
const mode = ref('search')  // 'search' | 'replace'
const replaceStatus = ref(null)

// 分组展开状态
const groupExpanded = ref({})

const groupedResults = computed(() => {
  const map = {}
  for (const r of state.searchResults) {
    if (!map[r.file]) {
      map[r.file] = { file: r.file, items: [], expanded: groupExpanded.value[r.file] !== false }
    }
    map[r.file].items.push(r)
  }
  const list = Object.values(map)
  // 同步 expanded 状态
  for (const g of list) {
    if (groupExpanded.value[g.file] === undefined) groupExpanded.value[g.file] = true
  }
  return list
})

function toggleGroup(file) {
  groupExpanded.value[file] = !groupExpanded.value[file]
}

function highlightMatch(text, q) {
  if (!text || !q) return text
  try {
    const escaped = q.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    const re = new RegExp('(' + escaped + ')', caseSensitive.value ? 'g' : 'gi')
    return text.replace(re, '<mark class="sp-match">$1</mark>')
  } catch {
    return text
  }
}

const doSearch = async () => {
  if (!query.value.trim()) return
  searched.value = true
  state.searchResults = []
  replaceStatus.value = null
  try {
    const params = { q: query.value, path: searchPath.value || state.workspaceRoot }
    if (caseSensitive.value) params.case_sensitive = '1'
    if (wholeWord.value) params.whole_word = '1'
    if (useRegex.value) params.regex = '1'
    const results = await api.apiGet('/fs/search', params)
    state.searchResults = results || []
  } catch (err) {
    state.searchResults = []
    console.error('搜索失败:', err)
  }
}

const openResult = (result) => {
  const path = result.file
  if (!state.openFiles.includes(path)) state.openFiles.push(path)
  state.activeFile = path
  if (!state.fileContents[path]) {
    api.apiGet('/fs/read', { path }).then(d => {
      state.fileContents[path] = d.content || ''
      state.fileSavedContent[path] = state.fileContents[path]
      state.fileDirty[path] = false
    }).catch(e => console.warn('[搜索] 读取文件失败:', path, e))
  }
}

const replaceAll = async () => {
  if (!query.value.trim() || state.searchResults.length === 0) return
  replaceStatus.value = { type: 'info', message: '正在替换...' }

  // 按文件分组，逐文件替换
  const files = {}
  for (const r of state.searchResults) {
    if (!files[r.file]) files[r.file] = new Set()
    files[r.file].add(r.line)
  }

  let successCount = 0
  let failCount = 0
  const fileList = Object.keys(files)

  for (const filePath of fileList) {
    try {
      // 读取文件完整内容
      const data = await api.apiGet('/fs/read', { path: filePath })
      let content = data.content || ''

      // 构建替换正则
      let pattern = query.value
      if (!useRegex.value) pattern = pattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
      if (wholeWord.value) pattern = '\\b' + pattern + '\\b'
      const flags = caseSensitive.value ? 'g' : 'gi'
      const re = new RegExp(pattern, flags)

      const newContent = content.replace(re, replaceText.value)
      if (newContent === content) continue

      // 写回文件
      await api.apiPost('/fs/write', { path: filePath, content: newContent })

      // 更新编辑器缓存
      const normalized = newContent.replace(/\r\n/g, '\n')
      if (state.openFiles.includes(filePath)) {
        state.fileContents[filePath] = normalized
        state.fileSavedContent[filePath] = normalized
        state.fileDirty[filePath] = false
      }
      successCount++
    } catch (err) {
      console.warn('[替换] 文件替换失败:', filePath, err)
      failCount++
    }
  }

  // 重新搜索获取最新结果
  await doSearch()

  replaceStatus.value = {
    type: failCount > 0 ? 'warn' : 'success',
    message: failCount > 0
      ? `完成：${successCount} 个文件已替换，${failCount} 个失败`
      : `✅ 全部完成：${successCount} 个文件已替换`,
  }

  // 触发文件树刷新
  window.dispatchEvent(new CustomEvent('refresh-tree'))

  // 3秒后清除状态提示
  setTimeout(() => { replaceStatus.value = null }, 5000)
}
</script>

<style scoped>
.search-panel {
  padding: 8px;
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  font-size: 13px;
}

/* 模式切换栏 */
.sp-mode-bar {
  display: flex;
  margin-bottom: 8px;
  border-bottom: 1px solid var(--border-color);
  gap: 0;
}
.sp-mode-btn {
  flex: 1;
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--text-muted);
  padding: 4px 0 6px;
  cursor: pointer;
  font-size: 12px;
  transition: all 0.15s;
}
.sp-mode-btn.active {
  color: var(--accent);
  border-bottom-color: var(--accent);
  font-weight: 600;
}
.sp-mode-btn:hover:not(.active) {
  color: var(--text-secondary);
}

/* 输入行 */
.sp-field {
  display: flex;
  gap: 4px;
  margin-bottom: 4px;
}
.sp-input-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 0 6px;
  transition: border-color 0.15s;
}
.sp-input-wrap:focus-within {
  border-color: var(--accent);
}
.sp-input-icon {
  color: var(--text-muted);
  margin-right: 4px;
  flex-shrink: 0;
}
.sp-input {
  flex: 1;
  background: none;
  border: none;
  color: var(--text-primary);
  padding: 5px 0;
  font-size: 13px;
  outline: none;
  min-width: 0;
}
.sp-go-btn {
  background: var(--accent);
  border: none;
  color: #fff;
  padding: 4px 10px;
  cursor: pointer;
  border-radius: 4px;
  font-size: 12px;
  white-space: nowrap;
}
.sp-go-btn:hover {
  filter: brightness(1.1);
}
.sp-replace-btn {
  background: #e06c75;
  border: none;
  color: #fff;
  padding: 4px 10px;
  cursor: pointer;
  border-radius: 4px;
  font-size: 12px;
  white-space: nowrap;
}
.sp-replace-btn:hover:not(:disabled) {
  filter: brightness(1.1);
}
.sp-replace-btn:disabled {
  opacity: 0.4;
  cursor: default;
}

/* 路径行 */
.sp-path-row {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-bottom: 6px;
}
.sp-path-input {
  flex: 1;
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-muted);
  padding: 3px 6px;
  font-size: 11px;
  outline: none;
  border-radius: 3px;
}
.sp-path-input:focus {
  border-color: var(--accent);
  color: var(--text-primary);
}
.sp-clear-btn {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 2px 4px;
  font-size: 14px;
  line-height: 1;
}
.sp-clear-btn:hover {
  color: var(--text-primary);
}

/* 选项 */
.sp-options {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  padding: 0 2px;
}
.sp-opt {
  display: flex;
  align-items: center;
  gap: 3px;
  cursor: pointer;
  font-size: 11px;
  color: var(--text-muted);
  user-select: none;
}
.sp-opt input[type="checkbox"] {
  margin: 0;
  accent-color: var(--accent);
}
.sp-opt:hover {
  color: var(--text-secondary);
}

/* 结果区域 */
.sp-results {
  flex: 1;
  overflow-y: auto;
  margin-top: 4px;
}
.sp-result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 2px 4px;
  font-size: 11px;
  color: var(--text-muted);
  margin-bottom: 2px;
}
.sp-result-count {
  font-weight: 500;
}
.sp-replace-all-sm {
  background: none;
  border: 1px solid var(--border-color);
  color: #e06c75;
  padding: 1px 6px;
  font-size: 10px;
  cursor: pointer;
  border-radius: 3px;
}
.sp-replace-all-sm:hover {
  background: #e06c7520;
}

/* 文件分组 */
.sp-file-group {
  margin-bottom: 4px;
}
.sp-file-title {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 4px;
  cursor: pointer;
  border-radius: 3px;
  font-size: 12px;
}
.sp-file-title:hover {
  background: var(--bg-hover);
}
.sp-file-path {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--accent-light);
}
.sp-file-count {
  font-size: 10px;
  color: var(--text-muted);
  flex-shrink: 0;
}
.sp-file-items {
  margin-left: 14px;
}
.sp-result-row {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 2px 4px;
  cursor: pointer;
  border-radius: 2px;
  font-size: 12px;
  line-height: 1.4;
}
.sp-result-row:hover {
  background: var(--bg-hover);
}
.sp-result-line {
  color: var(--text-muted);
  font-size: 10px;
  min-width: 28px;
  text-align: right;
  flex-shrink: 0;
  font-family: var(--font-editor, monospace);
  line-height: 1.4;
}
.sp-result-text {
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.4;
}
:deep(.sp-match) {
  background: #e2b71433;
  color: #e2b714;
  border-radius: 2px;
  padding: 0 1px;
}

/* 空状态 */
.sp-no-results {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 40px;
  color: var(--text-muted);
  font-size: 13px;
}
.sp-no-icon {
  opacity: 0.4;
}
.sp-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 40px;
  color: var(--text-muted);
  font-size: 12px;
  opacity: 0.6;
}

/* 状态提示 */
.sp-status {
  padding: 6px 10px;
  margin-top: 6px;
  border-radius: 4px;
  font-size: 12px;
  flex-shrink: 0;
}
.sp-status.info {
  background: var(--accent-bg);
  color: var(--accent);
}
.sp-status.success {
  background: #1b3a2d;
  color: #7ec8a3;
}
.sp-status.warn {
  background: #3a2d1b;
  color: #d4a05a;
}
</style>
