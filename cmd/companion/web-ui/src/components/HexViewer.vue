<template>
  <div class="hex-viewer">
    <div class="hex-toolbar">
      <span class="hex-file-info">{{ fileName }} · {{ formatSize(fileSize) }}</span>
      <span class="hex-spacer"></span>
      <button class="hex-btn" @click="loadPrev" :disabled="currentPage <= 0" title="上一页 (PageUp)">▲</button>
      <span class="hex-pos">第 {{ currentPage + 1 }}/{{ totalPages }} 页</span>
      <button class="hex-btn" @click="loadNext" :disabled="currentPage >= totalPages - 1" title="下一页 (PageDown)">▼</button>
      <span class="hex-sep"></span>
      <label class="hex-goto">
        偏移:
        <input type="text" v-model="gotoInput" class="hex-goto-input"
               placeholder="0x or decimal" @keydown.enter="goToOffset" />
        <button class="hex-btn" @click="goToOffset" title="跳转">跳转</button>
      </label>
      <span class="hex-sep"></span>
      <button class="hex-btn" @click="loadAll" :disabled="loading" title="加载全部">全部</button>
    </div>
    <div class="hex-body">
      <div v-if="loading" class="hex-loading">加载中...</div>
      <pre v-else class="hex-dump"><code>{{ hexDump }}</code></pre>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import api from '../api.js'

const props = defineProps({
  path: { type: String, required: true },
})

const hexDump = ref('')
const currentPage = ref(0)
const totalPages = ref(0)
const fileSize = ref(0)
const loading = ref(false)
const chunkSize = 512
const gotoInput = ref('')

const fileName = computed(() => {
  const p = props.path || ''
  return p.split('\\').pop() || p.split('/').pop() || p
})

function formatSize(s) {
  if (s < 1024) return s + ' B'
  if (s < 1024 * 1024) return (s / 1024).toFixed(1) + ' KB'
  return (s / (1024 * 1024)).toFixed(1) + ' MB'
}

async function fetchPage(page) {
  loading.value = true
  try {
    const offset = page * chunkSize
    const res = await api.apiGet('/fs/hex', {
      path: props.path,
      offset: String(offset),
      length: String(chunkSize),
    })
    hexDump.value = res.hex || ''
    currentPage.value = page
    fileSize.value = res.fileSize || 0
    const total = Math.ceil((res.fileSize || 0) / chunkSize)
    totalPages.value = total > 0 ? total : 1
  } catch (e) {
    hexDump.value = `加载失败: ${e.message || e}`
  } finally {
    loading.value = false
  }
}

function loadNext() {
  if (currentPage.value < totalPages.value - 1) fetchPage(currentPage.value + 1)
}

function loadPrev() {
  if (currentPage.value > 0) fetchPage(currentPage.value - 1)
}

async function loadAll() {
  if (loading.value) return
  loading.value = true
  try {
    let allHex = ''
    let page = 0
    for (;;) {
      const offset = page * chunkSize
      const res = await api.apiGet('/fs/hex', {
        path: props.path,
        offset: String(offset),
        length: String(chunkSize),
      })
      if (allHex) allHex += '\n'
      allHex += res.hex || ''
      page++
      if (!res.hasMore) break
    }
    hexDump.value = allHex
    fileSize.value = 0
    totalPages.value = 0
    currentPage.value = 0
  } catch (e) {
    hexDump.value = `加载失败: ${e.message || e}`
  } finally {
    loading.value = false
  }
}

function goToOffset() {
  let raw = gotoInput.value.trim()
  if (!raw) return
  let offset = 0
  if (raw.startsWith('0x') || raw.startsWith('0X')) {
    offset = parseInt(raw, 16)
  } else {
    offset = parseInt(raw, 10)
  }
  if (isNaN(offset) || offset < 0) {
    window.$toast?.('无效的偏移量', 'error')
    return
  }
  const page = Math.floor(offset / chunkSize)
  fetchPage(page)
}

onMounted(() => {
  fetchPage(0)
})

watch(() => props.path, () => {
  hexDump.value = ''
  currentPage.value = 0
  totalPages.value = 0
  fileSize.value = 0
  gotoInput.value = ''
  fetchPage(0)
})
</script>

<style scoped>
.hex-viewer {
  display: flex; flex-direction: column; height: 100%;
  background: var(--bg-primary); color: var(--text-primary);
  font-family: var(--font-code, 'Cascadia Code', 'Consolas', monospace);
}
.hex-toolbar {
  display: flex; align-items: center; gap: 6px;
  padding: 4px 8px; background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0; font-size: 12px;
  flex-wrap: wrap;
}
.hex-file-info { color: var(--accent-light); }
.hex-spacer { flex: 1; }
.hex-sep { width: 1px; height: 18px; background: var(--border-color); }
.hex-btn {
  background: var(--bg-tertiary); border: 1px solid var(--border-color);
  color: var(--text-secondary); padding: 2px 8px; border-radius: 3px;
  cursor: pointer; font-size: 11px;
}
.hex-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.hex-btn:disabled { opacity: 0.4; cursor: default; }
.hex-pos { color: var(--text-muted); font-size: 11px; min-width: 80px; text-align: center; white-space: nowrap; }
.hex-goto { display: flex; align-items: center; gap: 4px; font-size: 11px; color: var(--text-muted); }
.hex-goto-input {
  width: 90px; background: var(--input-bg); border: 1px solid var(--border-color);
  color: var(--text-primary); padding: 1px 6px; font-size: 11px; outline: none;
  border-radius: 3px; font-family: var(--font-code);
}
.hex-goto-input:focus { border-color: var(--accent); }
.hex-body { flex: 1; overflow: auto; padding: 0; }
.hex-loading { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--text-muted); font-size: 13px; }
.hex-dump {
  margin: 0; padding: 8px 12px;
  font-size: 12px; line-height: 1.5;
  white-space: pre; user-select: text;
}
.hex-dump code { font-family: inherit; }
</style>
