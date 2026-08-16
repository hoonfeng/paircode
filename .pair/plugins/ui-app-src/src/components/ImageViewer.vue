<template>
  <div class="image-viewer">
    <div class="img-toolbar">
      <span class="img-file-info">{{ fileName }} · {{ formatSize(fileSize) }}</span>
      <span class="img-spacer"></span>
      <button class="img-btn" @click="zoomOut" title="缩小">−</button>
      <span class="img-zoom">{{ Math.round(zoom * 100) }}%</span>
      <button class="img-btn" @click="zoomIn" title="放大">+</button>
      <button class="img-btn" @click="zoomFit" title="适应窗口">⊡</button>
      <button class="img-btn" @click="zoomReset" title="原始大小">1:1</button>
    </div>
    <div class="img-body" ref="containerRef" @wheel.prevent="onWheel">
      <div v-if="loading" class="img-loading">加载中...</div>
      <div v-else-if="error" class="img-error">{{ error }}</div>
      <img v-else ref="imgRef" :src="imageUrl" :style="imgStyle" class="img-display"
           @load="onImgLoad" @error="onImgError" alt="预览" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'

const props = defineProps({
  path: { type: String, required: true },
})

const loading = ref(true)
const error = ref('')
const zoom = ref(1)
const naturalW = ref(0)
const naturalH = ref(0)
const containerRef = ref(null)
const imgRef = ref(null)
const fileSize = ref(0)

const fileName = computed(() => {
  const p = props.path || ''
  return p.split('\\').pop() || p.split('/').pop() || p
})

const imageUrl = computed(() => {
  return '/api/fs/image?path=' + encodeURIComponent(props.path)
})

const imgStyle = computed(() => ({
  transform: `scale(${zoom.value})`,
  transformOrigin: 'top left',
  maxWidth: zoom.value <= 1 ? '100%' : 'none',
}))

function formatSize(s) {
  if (s < 1024) return s + ' B'
  if (s < 1024 * 1024) return (s / 1024).toFixed(1) + ' KB'
  return (s / (1024 * 1024)).toFixed(1) + ' MB'
}

function zoomIn() { zoom.value = Math.min(zoom.value * 1.25, 10) }
function zoomOut() { zoom.value = Math.max(zoom.value / 1.25, 0.1) }
function zoomReset() { zoom.value = 1 }
function zoomFit() {
  if (!containerRef.value || !naturalW.value) return
  const cw = containerRef.value.clientWidth - 16
  const ch = containerRef.value.clientHeight - 16
  const scaleX = cw / naturalW.value
  const scaleY = ch / naturalH.value
  zoom.value = Math.min(scaleX, scaleY, 1)
}

function onWheel(e) {
  if (e.ctrlKey || e.metaKey) {
    if (e.deltaY < 0) zoomIn()
    else zoomOut()
  }
}

function onImgLoad() {
  loading.value = false
  error.value = ''
  if (imgRef.value) {
    naturalW.value = imgRef.value.naturalWidth
    naturalH.value = imgRef.value.naturalHeight
  }
  // 获取文件大小（通过 HTTP Content-Length 或 X-File-Size）
  fetch(imageUrl.value, { method: 'HEAD' }).then(r => {
    const cl = r.headers.get('Content-Length')
    if (cl) fileSize.value = parseInt(cl)
  }).catch(() => {})
}

function onImgError() {
  loading.value = false
  error.value = '图片加载失败，文件可能已损坏或格式不支持'
}

onMounted(() => { loading.value = true; error.value = '' })
watch(() => props.path, () => { loading.value = true; error.value = ''; zoom.value = 1; fileSize.value = 0 })
</script>

<style scoped>
.image-viewer {
  display: flex; flex-direction: column; height: 100%;
  background: #1e1e1e; color: #d4d4d4;
}
.img-toolbar {
  display: flex; align-items: center; gap: 6px;
  padding: 4px 8px; background: #252526; border-bottom: 1px solid #3c3c3c;
  flex-shrink: 0; font-size: 12px;
}
.img-file-info { color: #9cdcfe; }
.img-spacer { flex: 1; }
.img-btn {
  background: #3c3c3c; border: 1px solid #555; color: #d4d4d4;
  padding: 2px 8px; border-radius: 3px; cursor: pointer; font-size: 13px; min-width: 28px; text-align: center;
}
.img-btn:hover { background: #505050; }
.img-zoom { color: #6a9955; font-size: 11px; min-width: 48px; text-align: center; }
.img-body {
  flex: 1; overflow: auto; display: flex; align-items: flex-start; justify-content: flex-start;
  padding: 8px; background: #1e1e1e;
}
.img-loading, .img-error {
  display: flex; align-items: center; justify-content: center;
  width: 100%; height: 100%; color: #888; font-size: 14px;
}
.img-error { color: #f48771; }
.img-display { flex-shrink: 0; user-select: none; }
</style>
