<template>
  <div v-if="visible" class="find-panel" @keydown="onPanelKeydown">
    <div class="find-row">
      <!-- 搜索图标 -->
      <svg class="fp-icon" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="11" cy="11" r="7"/><path d="m21 21-4.35-4.35"/>
      </svg>
      <input
        ref="findInput"
        v-model="searchText"
        type="text"
        placeholder="查找"
        @input="onSearchChange"
        @keydown.enter.prevent="onFindNext"
        @keydown.shift.enter.prevent="onFindPrev"
        class="fp-input"
        spellcheck="false"
        main-field="true"
      />
      <span class="fp-count">{{ matchCount }}</span>
      <!-- 上一个 -->
      <button class="fp-btn" @click="onFindPrev" title="上一个 (Shift+Enter)">
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="18 15 12 9 6 15"/>
        </svg>
      </button>
      <!-- 下一个 -->
      <button class="fp-btn" @click="onFindNext" title="下一个 (Enter)">
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="6 9 12 15 18 9"/>
        </svg>
      </button>
      <!-- 切换替换 -->
      <button class="fp-btn fp-toggle" :class="{ active: showReplace }" @click="toggleReplace" title="替换 (Ctrl+H)">
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M16 5l3-3 3 3"/><path d="M19 2v7a4 4 0 0 1-4 4H5"/><path d="M8 16l-3 3 3 3"/><path d="M5 13v7a4 4 0 0 0 4 4h10"/>
        </svg>
      </button>
      <!-- 关闭 -->
      <button class="fp-close" @click="onClose" title="关闭 (Esc)">
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M18 6 6 18"/><path d="m6 6 12 12"/>
        </svg>
      </button>
    </div>
    <div v-if="showReplace" class="replace-row">
      <svg class="fp-icon" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M16 5l3-3 3 3"/><path d="M19 2v7a4 4 0 0 1-4 4H5"/><path d="M8 16l-3 3 3 3"/><path d="M5 13v7a4 4 0 0 0 4 4h10"/>
      </svg>
      <input
        ref="replaceInput"
        v-model="replaceText"
        type="text"
        placeholder="替换"
        @keydown.enter.prevent="onReplaceNext"
        class="fp-input"
        spellcheck="false"
      />
      <button class="fp-btn fp-action" @click="onReplaceNext" title="替换下一个">替换</button>
      <button class="fp-btn" @click="onReplaceAll" title="全部替换">全部替换</button>
    </div>
    <div class="fp-options">
      <!-- 大小写敏感 -->
      <label class="fp-opt" :class="{ active: caseSensitive }" title="大小写敏感">
        <input type="checkbox" v-model="caseSensitive" @change="onSearchChange" />
        <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="4" y="4" width="16" height="16" rx="2"/><path d="m8 16 2-4m4 4-2-4m0 0L12 8l1 4"/><path d="m10 12h4"/>
        </svg>
      </label>
      <!-- 正则表达式 -->
      <label class="fp-opt" :class="{ active: regexp }" title="正则表达式">
        <input type="checkbox" v-model="regexp" @change="onSearchChange" />
        <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 3v18"/><path d="M5 12h14"/><path d="M9 7l-4-4"/><path d="M15 7l4-4"/><path d="M9 17l-4 4"/><path d="M15 17l4 4"/>
        </svg>
      </label>
      <!-- 全词匹配 -->
      <label class="fp-opt" :class="{ active: wholeWord }" title="全词匹配">
        <input type="checkbox" v-model="wholeWord" @change="onSearchChange" />
        <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M4 8V6a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v2"/><path d="M4 16v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2"/><path d="M8 12h8"/><path d="M10 9v6"/><path d="M14 9v6"/>
        </svg>
      </label>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, watch } from 'vue'
import { SearchQuery, setSearchQuery, findNext, findPrevious, replaceNext, replaceAll, getSearchQuery, closeSearchPanel } from '@codemirror/search'

const props = defineProps({
  view: { type: Object, default: null },
})

const emit = defineEmits(['close'])

const visible = ref(false)
const showReplace = ref(false)
const searchText = ref('')
const replaceText = ref('')
const caseSensitive = ref(false)
const regexp = ref(false)
const wholeWord = ref(false)
const matchCount = ref('')
const findInput = ref(null)
const replaceInput = ref(null)

// 更新搜索查询到编辑器状态
function applySearchQuery() {
  if (!props.view) return
  const v = props.view
  const query = new SearchQuery({
    search: searchText.value || '',
    caseSensitive: caseSensitive.value,
    regexp: regexp.value,
    wholeWord: wholeWord.value,
    replace: replaceText.value,
  })
  v.dispatch({ effects: setSearchQuery.of(query) })
  updateMatchCount(v)
}

// 更新匹配计数
function updateMatchCount(view) {
  if (!view || !searchText.value) {
    matchCount.value = ''
    return
  }
  const query = getSearchQuery(view.state)
  if (!query.valid) {
    matchCount.value = '—'
    return
  }
  try {
    const cursor = query.getCursor(view.state)
    let count = 0
    let current = 0
    const sel = view.state.selection.main
    while (!cursor.next().done) {
      count++
      if (cursor.value.from <= sel.from && cursor.value.to >= sel.to) {
        current = count
      }
    }
    matchCount.value = count > 0 ? `${current}/${count}` : '0/0'
  } catch {
    matchCount.value = '—'
  }
}

// 输入变化
function onSearchChange() {
  applySearchQuery()
}

// 查找下一个
function onFindNext() {
  if (!props.view || !searchText.value) return
  applySearchQuery()
  findNext(props.view)
  nextTick(() => updateMatchCount(props.view))
}

// 查找上一个
function onFindPrev() {
  if (!props.view || !searchText.value) return
  applySearchQuery()
  findPrevious(props.view)
  nextTick(() => updateMatchCount(props.view))
}

// 替换下一个
function onReplaceNext() {
  if (!props.view || !searchText.value) return
  applySearchQuery()
  replaceNext(props.view)
  nextTick(() => updateMatchCount(props.view))
}

// 全部替换
function onReplaceAll() {
  if (!props.view || !searchText.value) return
  applySearchQuery()
  replaceAll(props.view)
  nextTick(() => updateMatchCount(props.view))
}

// 切换替换区域
function toggleReplace() {
  showReplace.value = !showReplace.value
  if (showReplace.value) {
    nextTick(() => replaceInput.value?.focus())
  } else {
    nextTick(() => findInput.value?.focus())
  }
}

// 面板键盘事件
function onPanelKeydown(e) {
  if (e.key === 'Escape') {
    e.preventDefault()
    e.stopPropagation()
    onClose()
  }
}

// 关闭
function onClose() {
  visible.value = false
  showReplace.value = false
  searchText.value = ''
  replaceText.value = ''
  if (props.view) {
    // 清除搜索高亮
    const emptyQuery = new SearchQuery({ search: '' })
    props.view.dispatch({ effects: setSearchQuery.of(emptyQuery) })
    closeSearchPanel(props.view)
    props.view.focus()
  }
  matchCount.value = ''
  emit('close')
}

// 打开搜索面板
function open(initialText) {
  visible.value = true
  showReplace.value = false
  searchText.value = initialText || ''
  replaceText.value = ''
  caseSensitive.value = false
  regexp.value = false
  wholeWord.value = false
  nextTick(() => {
    findInput.value?.focus()
    findInput.value?.select()
    if (props.view && searchText.value) {
      applySearchQuery()
      findNext(props.view)
    }
  })
}

// 打开替换面板
function openReplace(initialText) {
  open(initialText)
  showReplace.value = true
  nextTick(() => {
    replaceInput.value?.focus()
    replaceInput.value?.select()
  })
}

// 监听视图变化后更新计数
watch(() => props.view, (v) => {
  if (v && visible.value) {
    nextTick(() => updateMatchCount(v))
  }
})

defineExpose({ open, openReplace, close: onClose })
</script>

<style scoped>
.find-panel {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  background: var(--bg-secondary, #1e1e2e);
  border-bottom: 1px solid var(--border-color, #333);
  padding: 6px 10px;
  display: flex;
  flex-direction: column;
  gap: 5px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.3);
  font-size: 13px;
}

.find-row, .replace-row {
  display: flex;
  align-items: center;
  gap: 4px;
}

.fp-icon {
  flex-shrink: 0;
  color: var(--text-muted, #888);
  opacity: 0.7;
}

.fp-input {
  flex: 1;
  min-width: 0;
  background: var(--bg-primary, #111);
  border: 1px solid var(--border-color, #444);
  color: var(--text-primary, #ddd);
  padding: 3px 8px;
  border-radius: 3px;
  font-size: 13px;
  font-family: inherit;
  outline: none;
  transition: border-color 0.15s;
}
.fp-input:focus {
  border-color: var(--accent, #4a9eff);
}

.fp-count {
  font-size: 11px;
  color: var(--text-muted, #888);
  min-width: 40px;
  text-align: center;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.fp-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-tertiary, #252535);
  border: 1px solid var(--border-color, #444);
  color: var(--text-secondary, #aaa);
  padding: 2px 8px;
  border-radius: 3px;
  cursor: pointer;
  font-size: 12px;
  line-height: 1.6;
  white-space: nowrap;
  transition: all 0.12s;
}
.fp-btn:hover {
  background: var(--bg-hover, #333);
  color: var(--text-primary, #eee);
}
.fp-btn:active {
  transform: scale(0.96);
}
.fp-toggle.active {
  background: var(--accent, #4a9eff);
  color: #fff;
  border-color: var(--accent, #4a9eff);
}
.fp-action {
  background: var(--accent, #4a9eff);
  color: #fff;
  border-color: var(--accent, #4a9eff);
}
.fp-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  color: var(--text-muted, #888);
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
  opacity: 0.6;
  transition: opacity 0.12s;
}
.fp-close:hover {
  opacity: 1;
  color: var(--text-primary, #eee);
}

.fp-options {
  display: flex;
  gap: 8px;
  padding-left: 18px;
}

.fp-opt {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 11px;
  color: var(--text-muted, #888);
  cursor: pointer;
  padding: 1px 6px;
  border-radius: 3px;
  border: 1px solid transparent;
  transition: all 0.12s;
  user-select: none;
  line-height: 1.4;
}
.fp-opt input {
  display: none;
}
.fp-opt.active {
  color: var(--accent, #4a9eff);
  border-color: var(--accent, #4a9eff);
  background: color-mix(in srgb, var(--accent, #4a9eff) 10%, transparent);
}
.fp-opt:hover {
  color: var(--text-secondary, #aaa);
}
</style>
