<template>
  <div class="code-editor-container">
    <FindPanel ref="searchPanelRef" :view="view" @close="onSearchPanelClose" />
    <div class="code-editor-wrapper" ref="wrapperRef"></div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { EditorView } from 'codemirror'
import { EditorState, Transaction, Prec } from '@codemirror/state'
import { syntaxHighlighting, HighlightStyle, foldGutter, indentOnInput, defaultHighlightStyle, bracketMatching, foldKeymap } from '@codemirror/language'
import { history, defaultKeymap, historyKeymap } from '@codemirror/commands'
import { closeBrackets, autocompletion, closeBracketsKeymap, completionKeymap } from '@codemirror/autocomplete'
import { highlightSelectionMatches, search, searchKeymap } from '@codemirror/search'
import { lintKeymap } from '@codemirror/lint'
import { lineNumbers, highlightActiveLineGutter, highlightSpecialChars, drawSelection, dropCursor, rectangularSelection, crosshairCursor, highlightActiveLine, keymap } from '@codemirror/view'
import { tags } from '@lezer/highlight'
import { javascript } from '@codemirror/lang-javascript'
import { python } from '@codemirror/lang-python'
import { html } from '@codemirror/lang-html'
import { css } from '@codemirror/lang-css'
import { json } from '@codemirror/lang-json'
import { markdown } from '@codemirror/lang-markdown'
import { xml } from '@codemirror/lang-xml'
import { sql } from '@codemirror/lang-sql'
import { indentWithTab } from '@codemirror/commands'
import { state, isDarkTheme } from '../ui-state.js'
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
    // ★ 手动拼 basicSetup（codemirror 6.0.2 源码照搬），其中默认
    // foldGutter() 替换为 foldGutter({openText:'∨', closedText:'›'})——
    // 浏览器默认 openText '⌄'(U+2304) 在 wb-ui 系统符号字体是 notdef
    // （豆腐块矩形）；▾(U+25BE)/▸(U+25B8) 是实心三角（用户要求箭头风格，
    // 与浏览器参照的箭头一致）。∨(U+2228 逻辑或=V 形向下)/›(U+203A 右尖
    // 引号=V 形右) 在 Consolas/Segoe UI Symbol 均有真字形，视觉为线框箭头。
    lineNumbers(),
    highlightActiveLineGutter(),
    highlightSpecialChars(),
    history(),
    foldGutter({ openText: '∨', closedText: '›' }),
    drawSelection(),
    dropCursor(),
    EditorState.allowMultipleSelections.of(true),
    indentOnInput(),
    syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
    bracketMatching(),
    closeBrackets(),
    autocompletion(),
    rectangularSelection(),
    crosshairCursor(),
    highlightActiveLine(),
    highlightSelectionMatches(),
    keymap.of([
      ...closeBracketsKeymap,
      ...defaultKeymap,
      ...searchKeymap,
      ...historyKeymap,
      ...foldKeymap,
      ...completionKeymap,
      ...lintKeymap,
    ]),
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

  // ── 主题与语法高亮（★ 全部走令牌） ──
  // 旧实现按 'dark'/'light'/'warm'/'night' 四个硬编码主题名分支，而 state.theme 的
  // 实际取值是 8 套 UI 主题名（midnight/graphite/obsidian/aurora/daylight/paper/sand/slate）
  // → 四个分支永不命中 → themeExt/syntaxExt 恒为 null，语法高亮与编辑器主题从未生效。
  // 现改为：明暗两档由 isDarkTheme 判定（仅用于告知 CodeMirror 当前明/暗），
  // 颜色本身**全部走令牌**（--syn-* 语法角色 + --color-* 编辑器 chrome）。
  // 令牌是 CSS 变量 → 由浏览器在渲染时解析 → 切换 UI 主题即时跟随，无需重建编辑器。
  const dark = isDarkTheme(state.theme)
  const themeExt = EditorView.theme({
    '&': { backgroundColor: 'transparent', color: 'var(--color-fg)' },
    '.cm-content': { caretColor: 'var(--color-accent)' },
    '.cm-gutters': {
      backgroundColor: 'var(--color-surface)',
      borderRight: '1px solid var(--color-border)',
      color: 'var(--color-muted)',
    },
    '.cm-activeLineGutter': { backgroundColor: 'var(--color-surface-2)' },
    '.cm-activeLine': { backgroundColor: 'var(--color-surface-2)' },
    '&.cm-focused .cm-cursor': { borderLeftColor: 'var(--color-accent)' },
    '.cm-selectionBackground, &.cm-focused .cm-selectionBackground, ::selection': { backgroundColor: 'var(--color-accent-soft)' },
    '.cm-matchingBracket, &.cm-focused .cm-matchingBracket': {
      backgroundColor: 'var(--color-accent-bg)',
      outline: '1px solid var(--color-accent-ring)',
    },
    '.cm-nonmatchingBracket': { color: 'var(--color-danger)' },
  }, { dark })
  const syntaxExt = syntaxHighlighting(HighlightStyle.define([
    { tag: [tags.keyword, tags.definitionKeyword, tags.modifier, tags.operatorKeyword, tags.controlKeyword], color: 'var(--syn-keyword)' },
    { tag: [tags.typeName, tags.className, tags.namespace, tags.definition(tags.typeName)], color: 'var(--syn-type)' },
    { tag: [tags.function(tags.variableName), tags.function(tags.propertyName), tags.labelName], color: 'var(--syn-function)' },
    { tag: [tags.propertyName, tags.attributeName, tags.definition(tags.propertyName)], color: 'var(--syn-property)' },
    { tag: [tags.number, tags.bool, tags.atom, tags.null, tags.constant(tags.variableName)], color: 'var(--syn-constant)' },
    { tag: [tags.string, tags.attributeValue, tags.regexp, tags.special(tags.string)], color: 'var(--syn-string)' },
    { tag: [tags.variableName, tags.definition(tags.variableName), tags.local(tags.variableName)], color: 'var(--syn-variable)' },
    { tag: [tags.comment, tags.lineComment, tags.blockComment, tags.meta, tags.processingInstruction], color: 'var(--syn-comment)', fontStyle: 'italic' },
    { tag: tags.invalid, color: 'var(--syn-invalid)' },
    { tag: [tags.bracket, tags.paren, tags.squareBracket, tags.brace, tags.separator, tags.operator], color: 'var(--syn-bracket)' },
    { tag: [tags.link, tags.url], color: 'var(--syn-link)', textDecoration: 'underline' },
    { tag: [tags.heading, tags.heading1, tags.heading2, tags.heading3, tags.heading4, tags.strong], color: 'var(--syn-heading)', fontWeight: 'bold' },
    { tag: tags.emphasis, fontStyle: 'italic' },
    { tag: tags.strikethrough, textDecoration: 'line-through' },
    { tag: tags.quote, color: 'var(--syn-comment)', fontStyle: 'italic' },
    { tag: tags.escape, color: 'var(--syn-constant)' },
  ]))

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
        // ★ 折叠箭头（∨/› 文本字符）缩小：13px 时 › 宽 7.2px/∨ 宽 10.2px
        // 在 12.2px 宽的行号 gutter 列里显得过大；缩到 8px 后 ∨≈6.3px、
        // ›≈4.4px，接近原生 select 下拉箭头（wb-ui paintSelectArrow：
        // ~6px 宽 7px 高 stroke 2px 的 V 形 chevron）的视觉尺寸。
        '.cm-foldGutter .cm-gutterElement': { fontSize: '8px' },
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
