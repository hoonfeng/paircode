<template>
  <div class="code-editor-container">
    <FindPanel ref="searchPanelRef" :view="view" @close="onSearchPanelClose" />
    <div class="code-editor-wrapper" ref="wrapperRef"></div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { EditorView, basicSetup } from 'codemirror'
import { EditorState, Transaction, Prec } from '@codemirror/state'
import { syntaxHighlighting, HighlightStyle } from '@codemirror/language'
import { tags } from '@lezer/highlight'
import { javascript } from '@codemirror/lang-javascript'
import { python } from '@codemirror/lang-python'
import { html } from '@codemirror/lang-html'
import { css } from '@codemirror/lang-css'
import { json } from '@codemirror/lang-json'
import { markdown } from '@codemirror/lang-markdown'
import { xml } from '@codemirror/lang-xml'
import { sql } from '@codemirror/lang-sql'
import { keymap } from '@codemirror/view'
import { indentWithTab } from '@codemirror/commands'
import { oneDark } from '@codemirror/theme-one-dark'
import { closeBrackets } from '@codemirror/autocomplete'
import { highlightSelectionMatches, search } from '@codemirror/search'
import { state } from '../main.js'
import api from '../api.js'
import FindPanel from './FindPanel.vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  path: { type: String, default: '' },
  readonly: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue', 'save', 'cursorPos', 'contextmenu-selection'])

const wrapperRef = ref(null)
const searchPanelRef = ref(null)
let view = null

function getLang(path) {
  if (!path) return null
  const ext = path.split('.').pop().toLowerCase()
  const langMap = {
    js: javascript, jsx: javascript, mjs: javascript, cjs: javascript,
    ts: () => javascript({ typescript: true }),
    tsx: () => javascript({ jsx: true, typescript: true }),
    py: python, html: html, htm: html,
    css: css, scss: css, less: css,
    json: json, md: markdown, xml: xml, svg: xml,
    sql: sql, go: javascript, rs: javascript, java: javascript,
    c: javascript, cpp: javascript, h: javascript, hpp: javascript,
    vue: html, svelte: html, php: html, rb: javascript,
    yaml: markdown, yml: markdown, toml: markdown,
    sh: javascript, bash: javascript, ps1: javascript,
    swift: javascript, kt: javascript,
  }
  return langMap[ext] || null
}

function createEditor() {
  if (!wrapperRef.value) return

  const lang = getLang(props.path)

  const extensions = [
    basicSetup,
    search(),               // 初始化搜索状态，不显示默认面板
    keymap.of([indentWithTab]),
    closeBrackets(),
    highlightSelectionMatches(),
    // 拦截 Ctrl+F/H 显示自定义中文搜索面板
    Prec.high(keymap.of([
      {
        key: 'Ctrl-f',
        run: () => {
          const selectedText = view ? view.state.sliceDoc(view.state.selection.main.from, view.state.selection.main.to) : ''
          searchPanelRef.value?.open(selectedText)
          return true
        },
      },
      {
        key: 'Ctrl-h',
        run: () => {
          const selectedText = view ? view.state.sliceDoc(view.state.selection.main.from, view.state.selection.main.to) : ''
          searchPanelRef.value?.openReplace(selectedText)
          return true
        },
      },
    ])),
    EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        emit('update:modelValue', update.state.doc.toString())
      }
      // 光标位置变化时通知
      if (update.selectionSet) {
        const pos = update.state.selection.main.head
        const line = update.state.doc.lineAt(pos)
        emit('cursorPos', { line: line.number, col: pos - line.from + 1 })
      }
    }),
  ]

  // 主题：为每个模式创建完整的语法高亮
  let themeExt = null
  let syntaxExt = null

  if (state.theme === 'dark') {
    // 暗色主题：使用 oneDark + 微调
    themeExt = oneDark
    syntaxExt = syntaxHighlighting(HighlightStyle.define([])) // oneDark 自带高亮
  } else if (state.theme === 'light') {
    themeExt = EditorView.theme({
      '&': { backgroundColor: '#ffffff', color: '#1a1a2e' },
      '.cm-gutters': { backgroundColor: '#f8f9fa', borderRight: '1px solid #dadce0', color: '#6e7681' },
      '.cm-activeLineGutter': { backgroundColor: '#e8eaed' },
      '.cm-activeLine': { backgroundColor: '#f0f4ff' },
      '&.cm-focused .cm-cursor': { borderLeftColor: '#1a73e8' },
      '.cm-selectionBackground': { background: '#1a73e830 !important' },
      '.cm-matchingBracket': { background: '#1a73e820', outline: '1px solid #1a73e840' },
    })
    syntaxExt = syntaxHighlighting(HighlightStyle.define([
      { tag: tags.keyword, color: '#d73a49' },
      { tag: [tags.definitionKeyword, tags.modifier], color: '#d73a49' },
      { tag: tags.typeName, color: '#005cc5' },
      { tag: tags.className, color: '#6f42c1' },
      { tag: tags.function(tags.variableName), color: '#6f42c1' },
      { tag: tags.function(tags.propertyName), color: '#6f42c1' },
      { tag: tags.definition(tags.propertyName), color: '#005cc5' },
      { tag: tags.definition(tags.typeName), color: '#6f42c1' },
      { tag: tags.propertyName, color: '#005cc5' },
      { tag: tags.attributeName, color: '#005cc5' },
      { tag: tags.attributeValue, color: '#032f62' },
      { tag: tags.number, color: '#005cc5' },
      { tag: tags.string, color: '#032f62' },
      { tag: tags.bool, color: '#005cc5' },
      { tag: tags.regexp, color: '#032f62' },
      { tag: tags.variableName, color: '#e36209' },
      { tag: tags.comment, color: '#6a737d', fontStyle: 'italic' },
      { tag: tags.invalid, color: '#d73a49' },
      { tag: tags.operator, color: '#d73a49' },
      { tag: tags.bracket, color: '#1a1a2e' },
      { tag: tags.paren, color: '#1a1a2e' },
      { tag: tags.separator, color: '#1a1a2e' },
      { tag: tags.link, color: '#032f62', textDecoration: 'underline' },
      { tag: tags.strong, fontWeight: 'bold' },
      { tag: tags.emphasis, fontStyle: 'italic' },
      { tag: tags.strikethrough, textDecoration: 'line-through' },
      { tag: tags.heading, color: '#005cc5', fontWeight: 'bold' },
      { tag: tags.processingInstruction, color: '#6a737d' },
      { tag: tags.meta, color: '#6a737d' },
    ]))
  } else if (state.theme === 'warm') {
    themeExt = EditorView.theme({
      '&': { backgroundColor: '#faf3e8', color: '#3d2c1e' },
      '.cm-gutters': { backgroundColor: '#f5ece0', borderRight: '1px solid #d6c8b8', color: '#a09080' },
      '.cm-activeLineGutter': { backgroundColor: '#e8dbcb' },
      '.cm-activeLine': { backgroundColor: '#f0e8d8' },
      '&.cm-focused .cm-cursor': { borderLeftColor: '#b87333' },
      '.cm-selectionBackground': { background: '#b8733330 !important' },
      '.cm-matchingBracket': { background: '#b8733320', outline: '1px solid #b8733340' },
    })
    syntaxExt = syntaxHighlighting(HighlightStyle.define([
      { tag: tags.keyword, color: '#8b6f47' },
      { tag: [tags.definitionKeyword, tags.modifier], color: '#8b6f47' },
      { tag: tags.typeName, color: '#6b4c7a' },
      { tag: tags.className, color: '#6b4c7a' },
      { tag: tags.function(tags.variableName), color: '#8b5c3a' },
      { tag: tags.function(tags.propertyName), color: '#8b5c3a' },
      { tag: tags.definition(tags.propertyName), color: '#5a7a4a' },
      { tag: tags.definition(tags.typeName), color: '#6b4c7a' },
      { tag: tags.propertyName, color: '#5a7a4a' },
      { tag: tags.attributeName, color: '#5a7a4a' },
      { tag: tags.attributeValue, color: '#7a6a5a' },
      { tag: tags.number, color: '#b87333' },
      { tag: tags.string, color: '#7a5a3a' },
      { tag: tags.bool, color: '#b87333' },
      { tag: tags.regexp, color: '#7a5a3a' },
      { tag: tags.variableName, color: '#3d2c1e' },
      { tag: tags.comment, color: '#a09080', fontStyle: 'italic' },
      { tag: tags.invalid, color: '#c04040' },
      { tag: tags.operator, color: '#8b6f47' },
      { tag: tags.bracket, color: '#5a4a3a' },
      { tag: tags.paren, color: '#5a4a3a' },
      { tag: tags.link, color: '#7a5a3a', textDecoration: 'underline' },
      { tag: tags.heading, color: '#8b6f47', fontWeight: 'bold' },
    ]))
  } else if (state.theme === 'night') {
    themeExt = EditorView.theme({
      '&': { backgroundColor: '#12101a', color: '#d8d4e0' },
      '.cm-gutters': { backgroundColor: '#1a1726', borderRight: '1px solid #2d2940', color: '#6a6680' },
      '.cm-activeLineGutter': { backgroundColor: '#252235' },
      '.cm-activeLine': { backgroundColor: '#1e1b30' },
      '&.cm-focused .cm-cursor': { borderLeftColor: '#9b8ec4' },
      '.cm-selectionBackground': { background: '#9b8ec430 !important' },
      '.cm-matchingBracket': { background: '#9b8ec420', outline: '1px solid #9b8ec440' },
    })
    syntaxExt = syntaxHighlighting(HighlightStyle.define([
      { tag: tags.keyword, color: '#c4b8e8' },
      { tag: [tags.definitionKeyword, tags.modifier], color: '#c4b8e8' },
      { tag: tags.typeName, color: '#8ab8d4' },
      { tag: tags.className, color: '#b8add4' },
      { tag: tags.function(tags.variableName), color: '#b8add4' },
      { tag: tags.function(tags.propertyName), color: '#b8add4' },
      { tag: tags.definition(tags.propertyName), color: '#8ab8d4' },
      { tag: tags.definition(tags.typeName), color: '#b8add4' },
      { tag: tags.propertyName, color: '#8ab8d4' },
      { tag: tags.attributeName, color: '#8ab8d4' },
      { tag: tags.attributeValue, color: '#a8b4c0' },
      { tag: tags.number, color: '#b8add4' },
      { tag: tags.string, color: '#a8b4c0' },
      { tag: tags.bool, color: '#b8add4' },
      { tag: tags.regexp, color: '#a8b4c0' },
      { tag: tags.variableName, color: '#d8d4e0' },
      { tag: tags.comment, color: '#6a6680', fontStyle: 'italic' },
      { tag: tags.invalid, color: '#d08080' },
      { tag: tags.operator, color: '#c4b8e8' },
      { tag: tags.bracket, color: '#8884a0' },
      { tag: tags.paren, color: '#8884a0' },
      { tag: tags.link, color: '#a8b4c0', textDecoration: 'underline' },
      { tag: tags.heading, color: '#c4b8e8', fontWeight: 'bold' },
    ]))
  }

  if (themeExt) extensions.push(themeExt)
  if (syntaxExt) extensions.push(syntaxExt)

  // 语言
  const langImpl = lang ? lang() : null
  if (langImpl) extensions.push(langImpl)

  // 只读
  if (props.readonly) extensions.push(EditorView.editable.of(false))

  const tabSize = state.settings?.tabSize || 2
  const fontSize = state.settings?.fontSize || 13

  const editorState = EditorState.create({
    doc: props.modelValue || '',
    extensions: [
      ...extensions,
      EditorState.tabSize.of(tabSize),
      EditorView.theme({
        '&': { fontSize: fontSize + 'px' },
        '.cm-scroller': { fontFamily: "'JetBrains Mono','Fira Code','Cascadia Code','Consolas',monospace" },
      }),
    ],
  })

  view = new EditorView({
    state: editorState,
    parent: wrapperRef.value,
  })

  // ★ 调试探针：暴露 CM6 view（probe 验证编辑链路 state 同步；真实运行无害）
  if (typeof window !== 'undefined') window.__editorView = view

  // 监听编辑器区域的右键事件 — 无论有无选中都发射
  wrapperRef.value.addEventListener('contextmenu', (e) => {
    if (!view) return
    const sel = view.state.selection.main
    const selectedText = view.state.sliceDoc(sel.from, sel.to)
    // 计算选中文本的行号范围
    let lineStart = 0, lineEnd = 0
    if (selectedText) {
      lineStart = view.state.doc.lineAt(sel.from).number
      lineEnd = view.state.doc.lineAt(sel.to).number
    } else {
      lineStart = lineEnd = view.state.doc.lineAt(sel.from).number
    }
    emit('contextmenu', {
      text: selectedText || '',
      hasSelection: !!(selectedText && selectedText.length > 0),
      lineStart,
      lineEnd,
      x: e.clientX,
      y: e.clientY,
      path: props.path,
    })
  })
}

onMounted(() => {
  createEditor()
  // ★ wb-ui 引擎兼容：CM6 的 HeightOracle 行高探测依赖 createRange/
  // observer.ignore/forceFlush/getClientRects/defaultView/Window 等 DOM API
  // + rAF 时序，wb-ui 已补齐 API 但 measure 在引擎里的首轮触发仍可能
  // 停在默认 lineHeight=14（行号栏按 14px/行步进而内容 ~18.2px 错位）。
  // 挂载后用一个真实行块高度校准 oracle——浏览器里 CM6 自测正常（值相等
  // 不覆盖，零副作用）；引擎里 measure 未更新时兜底对齐行号与内容。
  setTimeout(() => {
    if (view && view.contentDOM && view.viewState) {
      try {
        // ★ 兼容层校准（仅引擎内生效）：CM6 在 wb-ui 引擎里首次 measure
        // 因构造时序（Vue onMounted 微任务 vs 渲染树下一帧重建）落空，
        // HeightOracle 停留默认 lineHeight=14 → 行号 14px/行 vs 内容
        // ~18.2px 逐行错位。用真实行块高度校准 oracle 并强制 heightMap
        // 全量重算（全文档替换 remote 事务，不进 undo）。浏览器里 CM6
        // 自测正常（值相等不覆盖，零副作用）。
        const tile = view.contentDOM.firstChild
        if (tile) {
          const h = tile.getBoundingClientRect().height
          if (h > 0 && Math.abs(view.defaultLineHeight - h) > 0.3) {
            const oracle = view.viewState.heightOracle
            oracle.lineHeight = h
            oracle.textHeight = h
            oracle.heightSamples = {}
            const docStr = view.state.doc.toString()
            view.dispatch({
              changes: { from: 0, to: view.state.doc.length, insert: docStr },
              annotations: Transaction.remote
            })
            view.requestMeasure()
          }
        }
      } catch (e) { /* 兼容层失败不影响编辑器 */ }
    }
  }, 100)
})

watch(() => props.path, () => {
  if (view) {
    view.destroy()
    view = null
  }
  createEditor()
})

watch(() => props.modelValue, (newVal) => {
  if (view && newVal !== view.state.doc.toString()) {
    view.dispatch({
      changes: { from: 0, to: view.state.doc.length, insert: newVal || '' }
    })
  }
})

watch(() => state.settings?.tabSize, (val) => {
  if (view && val) {
    view.dispatch({ effects: EditorState.tabSize.reconfigure(val) })
  }
})

watch(() => state.settings?.fontSize, (val) => {
  if (view && val) {
    view.destroy()
    view = null
    createEditor()
  }
})

onBeforeUnmount(() => {
  if (view) {
    view.destroy()
    view = null
  }
})

// 搜索面板关闭时聚焦回编辑器
function onSearchPanelClose() {
  view?.focus()
}

defineExpose({
  getEditor: () => view,
  focus: () => view?.focus(),
  execSave: () => emit('save'),
  openFind: (text) => searchPanelRef.value?.open(text),
})
</script>

<style scoped>
.code-editor-container {
  position: relative;
  height: 100%;
  overflow: hidden;
}
.code-editor-wrapper {
  height: 100%;
  overflow: hidden;
}
.code-editor-wrapper :deep(.cm-editor) {
  height: 100%;
  background: var(--bg-primary);
  color: var(--text-primary);
}
.code-editor-wrapper :deep(.cm-editor.cm-focused) {
  outline: none;
}
.code-editor-wrapper :deep(.cm-scroller) {
  overflow: auto;
  font-family: var(--font-editor);
  font-size: var(--font-size-base);
}
.code-editor-wrapper :deep(.cm-gutters) {
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-color);
  color: var(--text-muted);
  font-family: var(--font-editor);
}
.code-editor-wrapper :deep(.cm-activeLineGutter) {
  background: var(--bg-hover);
}
.code-editor-wrapper :deep(.cm-activeLine) {
  background: var(--accent-bg);
}
.code-editor-wrapper :deep(.cm-cursor) {
  border-left-color: var(--text-primary);
}
.code-editor-wrapper :deep(.cm-selectionBackground) {
  background: var(--accent) !important;
  opacity: 0.25;
}
.code-editor-wrapper :deep(.cm-matchingBracket) {
  background: var(--accent-bg);
  outline: 1px solid var(--accent);
}

/* 隐藏 CodeMirror 默认搜索面板 */
:deep(.cm-panel.cm-search) {
  display: none !important;
}
</style>
