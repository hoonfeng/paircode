<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content help-modal">
      <!-- 头部 -->
      <div class="modal-header">
        <h2><SvgIcon name="book-open" :size="18" /> 帮助文档</h2>
        <div class="header-actions">
          <input type="text" class="doc-search-input" v-model="searchQuery"
                 placeholder="搜索文档..." @input="filterDocs" />
          <button class="modal-close" @click="$emit('close')">&times;</button>
        </div>
      </div>

      <!-- 主体 -->
      <div class="modal-body">
        <!-- 侧边导航 -->
        <div class="doc-sidebar">
          <div v-for="doc in filteredDocs" :key="doc.id"
               :class="['doc-nav-item', { active: activeDoc === doc.id }]"
               @click="activeDoc = doc.id">
            <SvgIcon :name="doc.icon" :size="16" />
            <span>{{ doc.title }}</span>
          </div>
        </div>

        <!-- 文档内容 -->
        <div class="doc-content">
          <div class="doc-content-inner" ref="contentRef">
            <div v-if="activeDoc === 'features'" class="doc-markdown" v-html="renderedFeatures"></div>
            <div v-else-if="activeDoc === 'api'" class="doc-markdown" v-html="renderedApi"></div>
            <div v-else-if="activeDoc === 'tools'" class="doc-markdown" v-html="renderedTools"></div>
            <div v-else-if="activeDoc === 'shortcuts'" class="doc-markdown" v-html="renderedShortcuts"></div>
          </div>

          <!-- 底部翻页 -->
          <div class="doc-pagination">
            <button class="page-btn" @click="prevDoc" :disabled="!hasPrev">
              <SvgIcon name="chevron-left" :size="14" /> 上一页
            </button>
            <span class="page-info">{{ currentDocIndex + 1 }} / {{ filteredDocs.length }}</span>
            <button class="page-btn" @click="nextDoc" :disabled="!hasNext">
              下一页 <SvgIcon name="chevron-right" :size="14" />
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import SvgIcon from './SvgIcon.vue'
import featuresMd from '../docs/features.md?raw'
import apiDocsMd from '../docs/api-docs.md?raw'
import toolsMd from '../docs/tools.md?raw'
import shortcutsMd from '../docs/shortcuts.md?raw'
import { marked } from 'marked'

const props = defineProps({
  initialDoc: { type: String, default: 'features' },
})

const emit = defineEmits(['close'])
const activeDoc = ref(props.initialDoc)
const searchQuery = ref('')
const contentRef = ref(null)

const docsList = [
  { id: 'features', title: '功能介绍', icon: 'info' },
  { id: 'api', title: 'API 文档', icon: 'code' },
  { id: 'tools', title: '工具文档', icon: 'tool' },
  { id: 'shortcuts', title: '快捷键', icon: 'keyboard' },
]

const filteredDocs = ref(docsList)

function filterDocs() {
  const q = searchQuery.value.toLowerCase().trim()
  if (!q) {
    filteredDocs.value = docsList
    return
  }
  filteredDocs.value = docsList.filter(d =>
    d.title.toLowerCase().includes(q) || d.id.includes(q)
  )
  if (filteredDocs.value.length > 0 && !filteredDocs.value.find(d => d.id === activeDoc.value)) {
    activeDoc.value = filteredDocs.value[0].id
  }
}

const renderMd = (md) => {
  const html = marked(md, { breaks: true, gfm: true })
  // 给表格加 class
  return html.replace(/<table>/g, '<table class="doc-table">')
}

const renderedFeatures = computed(() => renderMd(featuresMd))
const renderedApi = computed(() => renderMd(apiDocsMd))
const renderedTools = computed(() => renderMd(toolsMd))
const renderedShortcuts = computed(() => renderMd(shortcutsMd))

const currentDocIndex = computed(() => filteredDocs.value.findIndex(d => d.id === activeDoc.value))
const hasPrev = computed(() => currentDocIndex.value > 0)
const hasNext = computed(() => currentDocIndex.value < filteredDocs.value.length - 1)

function prevDoc() {
  if (hasPrev.value) {
    activeDoc.value = filteredDocs.value[currentDocIndex.value - 1].id
    contentRef.value?.scrollTo(0, 0)
  }
}
function nextDoc() {
  if (hasNext.value) {
    activeDoc.value = filteredDocs.value[currentDocIndex.value + 1].id
    contentRef.value?.scrollTo(0, 0)
  }
}

onMounted(() => {
  filterDocs()
})
</script>

<style scoped>
.modal-overlay {
  position: fixed; top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.5); z-index: 2000;
  display: flex; align-items: center; justify-content: center;
}
.modal-content {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  width: 850px;
  height: 75vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0,0,0,0.4);
}
.modal-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}
.modal-header h2 {
  font-size: 16px; font-weight: 600;
  display: flex; align-items: center; gap: 8px;
}
.header-actions {
  display: flex; align-items: center; gap: 12px;
}
.doc-search-input {
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 5px 10px;
  border-radius: 4px;
  font-size: 12px;
  width: 180px;
  outline: none;
}
.doc-search-input:focus {
  border-color: var(--accent);
}
.modal-close {
  background: none; border: none;
  color: var(--text-secondary);
  font-size: 22px; cursor: pointer;
  width: 28px; height: 28px;
  display: flex; align-items: center; justify-content: center;
  border-radius: 4px;
}
.modal-close:hover { background: var(--bg-hover); color: var(--text-primary); }

.modal-body {
  flex: 1; display: flex; overflow: hidden;
}

/* 侧边导航 */
.doc-sidebar {
  width: 190px; flex-shrink: 0;
  border-right: 1px solid var(--border-color);
  padding: 8px 0;
  overflow-y: auto;
  background: var(--bg-tertiary);
}
.doc-nav-item {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 16px;
  cursor: pointer; font-size: 13px;
  color: var(--text-secondary);
  transition: all 0.15s;
}
.doc-nav-item:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}
.doc-nav-item.active {
  background: var(--accent-bg);
  color: var(--accent);
  border-right: 2px solid var(--accent);
}

/* 文档内容 */
.doc-content {
  flex: 1; display: flex; flex-direction: column; overflow: hidden;
}
.doc-content-inner {
  flex: 1; overflow-y: auto; padding: 20px 28px;
}
.doc-markdown {
  font-size: 14px; line-height: 1.7;
  color: var(--text-primary);
}
.doc-markdown :deep(h1) {
  font-size: 22px; font-weight: 700;
  margin: 0 0 16px 0;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-color);
  color: var(--text-primary);
}
.doc-markdown :deep(h2) {
  font-size: 18px; font-weight: 600;
  margin: 24px 0 12px 0;
  color: var(--accent-light);
}
.doc-markdown :deep(h3) {
  font-size: 15px; font-weight: 600;
  margin: 20px 0 8px 0;
  color: var(--text-primary);
}
.doc-markdown :deep(p) {
  margin: 8px 0;
}
.doc-markdown :deep(code) {
  font-family: var(--font-code);
  font-size: 13px;
  background: var(--bg-primary);
  padding: 2px 6px;
  border-radius: 3px;
  border: 1px solid var(--border-color);
}
.doc-markdown :deep(pre) {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 14px 16px;
  overflow-x: auto;
  margin: 12px 0;
}
.doc-markdown :deep(pre code) {
  background: none;
  border: none;
  padding: 0;
  font-size: 12.5px;
  line-height: 1.5;
}
.doc-markdown :deep(table.doc-table) {
  width: 100%;
  border-collapse: collapse;
  margin: 12px 0;
  font-size: 13px;
}
.doc-markdown :deep(table.doc-table th) {
  background: var(--bg-tertiary);
  padding: 8px 12px;
  text-align: left;
  border: 1px solid var(--border-color);
  font-weight: 600;
}
.doc-markdown :deep(table.doc-table td) {
  padding: 6px 12px;
  border: 1px solid var(--border-color);
}
.doc-markdown :deep(table.doc-table tr:nth-child(even)) {
  background: var(--bg-tertiary);
}
.doc-markdown :deep(ul), .doc-markdown :deep(ol) {
  margin: 6px 0;
  padding-left: 24px;
}
.doc-markdown :deep(li) {
  margin: 3px 0;
}
.doc-markdown :deep(blockquote) {
  border-left: 3px solid var(--accent);
  padding: 8px 16px;
  margin: 12px 0;
  background: var(--accent-bg);
  border-radius: 0 4px 4px 0;
  color: var(--text-secondary);
}
.doc-markdown :deep(strong) {
  font-weight: 600;
  color: var(--text-primary);
}
.doc-markdown :deep(a) {
  color: var(--accent);
  text-decoration: none;
}
.doc-markdown :deep(a:hover) {
  text-decoration: underline;
}

/* 翻页 */
.doc-pagination {
  display: flex; align-items: center; justify-content: center;
  gap: 16px;
  padding: 10px 16px;
  border-top: 1px solid var(--border-color);
  flex-shrink: 0;
}
.page-btn {
  display: flex; align-items: center; gap: 4px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  padding: 4px 12px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
}
.page-btn:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text-primary);
}
.page-btn:disabled {
  opacity: 0.4; cursor: default;
}
.page-info {
  font-size: 12px; color: var(--text-muted);
}
</style>
