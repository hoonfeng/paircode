<template>
  <div class="debug-log-panel">
    <div class="dlp-header">
      <span class="dlp-title"><SvgIcon name="bug" :size="14" /> Debug 日志</span>
      <div class="dlp-actions">
        <span class="dlp-count">{{ filteredLogs.length }} 条</span>
        <span class="dlp-filter-btn" :class="{ active: autoRefresh }" @click="toggleAutoRefresh" title="自动刷新">
          <SvgIcon name="refresh" :size="12" /> 自动
        </span>
        <span class="dlp-filter-btn" @click="fetchLogs" title="刷新">
          <SvgIcon name="refresh" :size="12" />
        </span>
        <span class="dlp-close-btn" @click="$emit('close')" title="关闭">
          <SvgIcon name="close" :size="12" />
        </span>
      </div>
    </div>
    <div class="dlp-filters">
      <span v-for="f in filterOptions" :key="f.value"
        :class="['dlp-filter', { active: currentFilter === f.value }]"
        @click="currentFilter = f.value">
        <span :class="['dlp-level-dot', f.dotClass]"></span>
        {{ f.label }}
      </span>
    </div>
    <div class="dlp-body" ref="bodyRef">
      <div v-if="filteredLogs.length === 0" class="dlp-empty">
        <SvgIcon name="check-circle" :size="16" class="dlp-empty-icon" />
        <span>暂无错误日志</span>
      </div>
      <div v-for="(log, i) in filteredLogs" :key="log.id"
        :class="['dlp-entry', { 'dlp-entry-expanded': expandedId === log.id }]"
        @click="toggleExpand(log.id)">
        <!-- 紧凑行 -->
        <div class="dlp-entry-row">
          <span :class="['dlp-level-badge', 'level-' + log.level]">{{ logLevelLabel(log.level) }}</span>
          <span class="dlp-entry-source">{{ log.source }}</span>
          <span class="dlp-entry-time">{{ formatTime(log.time) }}</span>
          <span class="dlp-entry-msg">{{ log.message }}</span>
        </div>
        <!-- 展开详情 -->
        <div v-if="expandedId === log.id" class="dlp-entry-detail">
          <div v-if="log.stack" class="dlp-detail-section">
            <div class="dlp-detail-title">堆栈</div>
            <pre class="dlp-detail-pre">{{ log.stack }}</pre>
          </div>
          <div v-if="log.context && log.context.buildOutput" class="dlp-detail-section">
            <div class="dlp-detail-title">构建输出</div>
            <pre class="dlp-detail-pre">{{ log.context.buildOutput }}</pre>
          </div>
          <div v-if="log.context" class="dlp-detail-section">
            <div class="dlp-detail-title">上下文</div>
            <div class="dlp-detail-ctx">
              <div v-for="(v, k) in log.context" :key="k" class="dlp-ctx-item">
                <span class="dlp-ctx-key">{{ k }}:</span>
                <span class="dlp-ctx-val" v-if="k !== 'buildOutput'">{{ v }}</span>
              </div>
            </div>
          </div>
          <div class="dlp-detail-time">时间: {{ log.time }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import SvgIcon from './SvgIcon.vue'

defineEmits(['close'])

const logs = ref([])
const expandedId = ref(null)
const currentFilter = ref('error')
const autoRefresh = ref(false)
const bodyRef = ref(null)
let refreshTimer = null

const filterOptions = [
  { value: 'all', label: '全部', dotClass: '' },
  { value: 'error', label: '错误', dotClass: 'dot-error' },
  { value: 'panic', label: '崩溃', dotClass: 'dot-panic' },
  { value: 'build', label: '构建', dotClass: 'dot-build' },
  { value: 'warning', label: '警告', dotClass: 'dot-warning' },
]

const filteredLogs = computed(() => {
  if (currentFilter.value === 'all') return logs.value
  return logs.value.filter(l => l.level === currentFilter.value)
})

function logLevelLabel(level) {
  const map = {
    'error': 'ERR',
    'panic': 'PANIC',
    'build': 'BUILD',
    'warning': 'WARN',
    'info': 'INFO',
    'debug': 'DEBUG',
    'debugop': 'DBGOP',
  }
  return map[level] || level
}

function formatTime(iso) {
  if (!iso) return ''
  try {
    const d = new Date(iso)
    return d.toLocaleTimeString()
  } catch {
    return iso
  }
}

function toggleExpand(id) {
  expandedId.value = expandedId.value === id ? null : id
}

function toggleAutoRefresh() {
  autoRefresh.value = !autoRefresh.value
  if (autoRefresh.value) {
    refreshTimer = setInterval(fetchLogs, 10000)
  } else {
    if (refreshTimer) clearInterval(refreshTimer)
    refreshTimer = null
  }
}

async function fetchLogs() {
  try {
    const url = currentFilter.value === 'all'
      ? '/api/debug/logs?limit=100'
      : `/api/debug/logs?level=${currentFilter.value}&limit=100`
    const resp = await fetch(url)
    const data = await resp.json()
    if (data && data.logs) {
      logs.value = data.logs
    }
  } catch (err) {
    // 静默失败
  }
}

onMounted(() => {
  fetchLogs()
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<style scoped>
.debug-log-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  font-size: 12px;
}

.dlp-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 8px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}
.dlp-title {
  display: flex;
  align-items: center;
  gap: 4px;
  font-weight: 600;
  color: var(--text-primary);
}
.dlp-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}
.dlp-count {
  color: var(--text-muted);
  font-size: 11px;
}
.dlp-filter-btn {
  cursor: pointer;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 1px 4px;
  border-radius: 3px;
}
.dlp-filter-btn:hover,
.dlp-filter-btn.active {
  color: var(--text-primary);
  background: var(--bg-hover);
}
.dlp-close-btn {
  cursor: pointer;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  padding: 1px 4px;
  border-radius: 3px;
}
.dlp-close-btn:hover {
  color: var(--text-primary);
  background: var(--bg-hover);
}

.dlp-filters {
  display: flex;
  gap: 2px;
  padding: 3px 8px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}
.dlp-filter {
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 3px;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 3px;
  font-size: 11px;
}
.dlp-filter:hover {
  color: var(--text-primary);
  background: var(--bg-hover);
}
.dlp-filter.active {
  color: var(--text-primary);
  background: var(--bg-active);
}
.dlp-level-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-muted);
}
.dot-error, .dot-panic, .dot-build { background: #e74c3c; }
.dot-warning { background: #f39c12; }

.dlp-body {
  flex: 1;
  overflow-y: auto;
}
.dlp-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-muted);
  gap: 8px;
}
.dlp-empty-icon { opacity: 0.3; }

.dlp-entry {
  border-bottom: 1px solid var(--border-color);
  cursor: pointer;
}
.dlp-entry:hover {
  background: var(--bg-hover);
}
.dlp-entry-expanded {
  background: var(--bg-active);
}

.dlp-entry-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 8px;
  font-size: 11px;
}
.dlp-level-badge {
  font-size: 9px;
  padding: 1px 4px;
  border-radius: 2px;
  font-weight: 600;
  flex-shrink: 0;
  min-width: 36px;
  text-align: center;
}
.level-error, .level-panic, .level-build {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}
.level-warning {
  background: rgba(243, 156, 18, 0.15);
  color: #f39c12;
}
.level-info, .level-debug, .level-debugop {
  background: rgba(149, 165, 166, 0.15);
  color: var(--text-muted);
}

.dlp-entry-source {
  color: var(--text-secondary);
  flex-shrink: 0;
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dlp-entry-time {
  color: var(--text-muted);
  flex-shrink: 0;
  font-family: monospace;
}
.dlp-entry-msg {
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dlp-entry-detail {
  padding: 4px 8px 8px;
  border-top: 1px solid var(--border-color);
  background: var(--bg-primary);
}
.dlp-detail-section {
  margin-bottom: 4px;
}
.dlp-detail-title {
  font-size: 10px;
  color: var(--text-muted);
  text-transform: uppercase;
  margin-bottom: 2px;
}
.dlp-detail-pre {
  font-family: 'Consolas', monospace;
  font-size: 11px;
  background: var(--bg-code);
  padding: 4px 6px;
  border-radius: 3px;
  max-height: 120px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
  color: var(--text-secondary);
}
.dlp-detail-ctx {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 12px;
}
.dlp-ctx-item {
  font-size: 11px;
  color: var(--text-secondary);
}
.dlp-ctx-key {
  color: var(--text-muted);
}
.dlp-detail-time {
  font-size: 10px;
  color: var(--text-muted);
  margin-top: 4px;
}
</style>
