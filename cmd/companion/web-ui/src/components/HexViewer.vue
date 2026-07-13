<template>
  <div class="hex-viewer">
    <div class="hex-toolbar">
      <span class="hex-file-info">{{ fileName }} · {{ formatSize(fileSize) }}</span>
      <span class="hex-spacer"></span>
      <button class="hex-btn" @click="loadPrev" :disabled="offset <= 0" title="上一页">▲ 上页</button>
      <span class="hex-pos">偏移 {{ formatOffset(offset) }}</span>
      <button class="hex-btn" @click="loadNext" :disabled="!hasMore" title="下一页">▼ 下页</button>
      <button class="hex-btn" @click="loadAll" :disabled="loading" title="加载全部">加载全部</button>
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
const offset = ref(0)
const fileSize = ref(0)
const hasMore = ref(false)
const loading = ref(false)
const loadedAll = ref(false)
const chunkSize = 512 // 每页字节数

const fileName = computed(() => {
  const p = props.path || ''
  return p.split('\\').pop() || p.split('/').pop() || p
})

function formatSize(s) {
  if (s < 1024) return s + ' B'
  if (s < 1024 * 1024) return (s / 1024).toFixed(1) + ' KB'
  return (s / (1024 * 1024)).toFixed(1) + ' MB'
}

function formatOffset(o) {
  return '0x' + o.toString(16).toUpperCase().padStart(8, '0')
}

async function fetchHex(newOffset) {
  loading.value = true
  try {
    const res = await api.apiGet('/fs/hex', {
      path: props.path,
      offset: String(newOffset),
      length: String(chunkSize),
    })
    if (newOffset === 0) {
      hexDump.value = res.hex || ''
    } else {
      hexDump.value += '\n' + (res.hex || '')
    }
    offset.value = newOffset + (res.length || 0)
    fileSize.value = res.fileSize || 0
    hasMore.value = res.hasMore || false
  } catch (e) {
    hexDump.value = `加载失败: ${e.message || e}`
  } finally {
    loading.value = false
  }
}

function loadNext() {
  if (!hasMore.value || loading.value) return
  fetchHex(offset.value)
}

function loadPrev() {
  if (offset.value <= chunkSize) {
    // 重新从 0 开始
    hexDump.value = ''
    fetchHex(0)
    return
  }
  const newOffset = Math.max(0, offset.value - chunkSize * 2)
  hexDump.value = ''
  fetchHex(newOffset)
}

async function loadAll() {
  if (loadedAll.value || loading.value) return
  loading.value = true
  loadedAll.value = true
  try {
    let allHex = ''
    let off = 0
    for (;;) {
      const res = await api.apiGet('/fs/hex', {
        path: props.path,
        offset: String(off),
        length: String(chunkSize),
      })
      if (allHex) allHex += '\n'
      allHex += res.hex || ''
      off += res.length || 0
      if (!res.hasMore) break
    }
    hexDump.value = allHex
    offset.value = off
    fileSize.value = 0
    hasMore.value = false
  } catch (e) {
    hexDump.value = `加载失败: ${e.message || e}`
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchHex(0)
})

watch(() => props.path, () => {
  hexDump.value = ''
  offset.value = 0
  fileSize.value = 0
  hasMore.value = false
  loadedAll.value = false
  fetchHex(0)
})
</script>

<style scoped>
.hex-viewer {
  display: flex; flex-direction: column; height: 100%;
  background: #1e1e1e; color: #d4d4d4; font-family: var(--font-code, 'Cascadia Code', 'Consolas', monospace);
}
.hex-toolbar {
  display: flex; align-items: center; gap: 8px;
  padding: 4px 8px; background: #252526; border-bottom: 1px solid #3c3c3c;
  flex-shrink: 0; font-size: 12px;
}
.hex-file-info { color: #9cdcfe; }
.hex-spacer { flex: 1; }
.hex-btn {
  background: #3c3c3c; border: 1px solid #555; color: #d4d4d4;
  padding: 2px 8px; border-radius: 3px; cursor: pointer; font-size: 11px;
}
.hex-btn:hover { background: #505050; }
.hex-btn:disabled { opacity: 0.4; cursor: default; }
.hex-pos { color: #6a9955; font-size: 11px; min-width: 80px; text-align: center; }
.hex-body { flex: 1; overflow: auto; padding: 0; }
.hex-loading { display: flex; align-items: center; justify-content: center; height: 100%; color: #888; font-size: 13px; }
.hex-dump {
  margin: 0; padding: 8px 12px;
  font-size: 12px; line-height: 1.5;
  white-space: pre; user-select: text;
}
.hex-dump code { font-family: inherit; }
</style>
