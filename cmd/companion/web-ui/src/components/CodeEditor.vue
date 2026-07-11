<template>
  <div class="code-editor-wrapper" ref="wrapperRef"></div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch, inject } from 'vue'
import { EditorView, basicSetup } from 'codemirror'
import { EditorState } from '@codemirror/state'
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
import { highlightSelectionMatches } from '@codemirror/search'
import { state } from '../main.js'
import api from '../api.js'

const props = defineProps({
  modelValue: { type: String, default: '' },
  path: { type: String, default: '' },
  readonly: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue', 'save', 'cursorPos', 'contextmenu-selection'])

const wrapperRef = ref(null)
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
    keymap.of([indentWithTab]),
    closeBrackets(),
    highlightSelectionMatches(),
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

  // 主题
  if (state.theme === 'dark') {
    extensions.push(oneDark)
  } else if (state.theme === 'warm') {
    extensions.push(EditorView.theme({
      '&': { backgroundColor: '#faf3e8', color: '#3d2c1e' },
      '.cm-gutters': { backgroundColor: '#f5ece0', borderRight: '1px solid #d6c8b8' },
      '&.cm-focused .cm-cursor': { borderLeftColor: '#b87333' },
    }))
  } else if (state.theme === 'cute') {
    extensions.push(EditorView.theme({
      '&': { backgroundColor: '#fff5f7', color: '#4a1a2e' },
      '.cm-gutters': { backgroundColor: '#fce4ec', borderRight: '1px solid #e8b8c8' },
      '&.cm-focused .cm-cursor': { borderLeftColor: '#e84393' },
    }))
  } else {
    // light
    extensions.push(EditorView.theme({
      '&': { backgroundColor: '#ffffff', color: '#1a1a2e' },
      '.cm-gutters': { backgroundColor: '#f8f9fa', borderRight: '1px solid #dadce0' },
    }))
  }

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

defineExpose({
  getEditor: () => view,
  focus: () => view?.focus(),
  execSave: () => emit('save'),
})
</script>

<style scoped>
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
</style>
