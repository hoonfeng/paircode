<template>
  <div class="markdown-renderer" ref="renderRef">
    <!-- 分块渲染：普通 HTML + 图表占位块 -->
    <template v-for="(block, i) in blocks" :key="i">
      <!-- 普通 Markdown HTML -->
      <div v-if="block.type === 'html'" class="md-html" v-html="block.content"></div>

      <!-- Mermaid 图表 -->
      <div v-else-if="block.type === 'mermaid'" class="chart-block chart-mermaid"
           :class="{ 'chart-error': mermaidErrors[i] }">
        <div class="chart-label">
          <span class="chart-icon">📊</span>
          <span class="chart-title">{{ block.title || '图表' }}</span>
          <button v-if="mermaidErrors[i]" class="chart-retry-btn"
                  @click="retryMermaid(i)" title="重试渲染">↻</button>
        </div>
        <div v-if="!mermaidErrors[i]" :id="'mermaid-' + uid + '-' + i"
             class="mermaid-container">
          <pre class="mermaid-src">{{ block.code }}</pre>
        </div>
        <div v-else class="chart-error-msg">
          <pre><code>{{ block.code }}</code></pre>
          <p class="chart-error-hint">⚠️ 图表渲染失败，已显示源码。语法修正后自动恢复。</p>
        </div>
      </div>

      <!-- UI 布局可视化（增强版：支持所有布局类型） -->
      <div v-else-if="block.type === 'layout'" class="layout-block"
           :class="'layout-style-' + block.layoutStyle">
        <div class="chart-label">
          <span class="chart-icon">{{ layoutIcon(block.layoutStyle) }}</span>
          <span class="chart-title">{{ block.title }}</span>
          <span class="chart-type-badge">{{ layoutTypeLabel(block.layoutStyle) }}</span>
          <button v-if="block.layoutStyle === 'box' && block.items.length > 0" class="chart-retry-btn"
                  @click="toggleBoxZoom(i)" title="切换缩放">🔍</button>
        </div>
        <div class="layout-canvas-wrap" :class="{ 'layout-zoomed': boxZoomed[i] }">
          <div class="layout-preview" :style="block.containerStyle">
            <!-- 布局项渲染 -->
            <div v-for="(item, ii) in block.items" :key="ii"
                 class="layout-item"
                 :style="item.style"
                 :title="item.label + (item.sub ? ' - ' + item.sub : '')">
              <!-- 带图标渲染 -->
              <span v-if="item.icon" class="layout-item-icon">{{ item.icon }}</span>
              <span class="layout-item-label">{{ item.label }}</span>
              <span v-if="item.sub" class="layout-item-sub">{{ item.sub }}</span>
            </div>
            <!-- 空布局提示 -->
            <div v-if="block.items.length === 0" class="layout-empty-hint">
              <span>💡 按格式定义布局元素：每行一个，支持前缀 > . # + - *</span>
            </div>
          </div>
        </div>
        <details class="chart-table-toggle">
          <summary>📋 查看布局源码</summary>
          <div class="layout-code-wrap"><pre><code>{{ block.rawCode }}</code></pre></div>
        </details>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, nextTick, onBeforeUnmount } from 'vue'
import { marked } from 'marked'
import { isDarkTheme } from '../ui-state.js'

// ══════════════════════════════════════════════════════════════
// ★★ 2026-09-25 性能修复：mermaid 由「静态 import」改为「按需惰性加载」★★
//   根因：mermaid.min.js 单体 3.6MB，静态 import 会被打进**每一个**引用本组件的
//   区域包（实测 ui-right-panel.js 3.6MB 几乎全是它、ui-editor.js 4.6MB 亦含它）
//   → 首屏必须下载+编译约 7.2MB JS。区域包为 IIFE 单文件 lib 构建，动态 import()
//   会被 rollup 内联，故**无法**靠 code-splitting 拆分，只能改为运行时按需注入。
//   而绝大多数对话/文件根本不含 mermaid 图 → 改成「真有图要渲染时才注入 <script>」，
//   从运行目录 /vendor/mermaid.min.js 取（mermaid 官方 dist，尾部
//   `globalThis["mermaid"] = ...`，onload 后 window.mermaid 即可用）。
//   ★ 副产品：任何区域包都不再打包 mermaid（体积骤降），加载时机与引用方无关。
let _mermaidPromise = null
function loadMermaid() {
  if (window.mermaid) return Promise.resolve(window.mermaid)
  if (_mermaidPromise) return _mermaidPromise        // 并发去重：多个渲染器共享同一次加载
  _mermaidPromise = new Promise((resolve, reject) => {
    // 相对 document.baseURI 解析（子路径部署同样正确）
    const url = new URL('vendor/mermaid.min.js', document.baseURI).href
    const s = document.createElement('script')
    s.src = url
    s.async = true
    s.onload = () => { window.mermaid ? resolve(window.mermaid) : reject(new Error('全局 mermaid 未暴露')) }
    s.onerror = () => { _mermaidPromise = null; reject(new Error('mermaid 资源加载失败: ' + url)) }
    document.head.appendChild(s)
  })
  return _mermaidPromise
}

const props = defineProps({
  text: { type: String, default: '' },
  // ★ 主题名 = UI 主题名（midnight/graphite/obsidian/aurora/daylight/paper/sand/slate）。
  //   旧注释的 'dark'|'light'|'warm'|'cute' 是扩到 8 套主题前的旧值域，已失效：
  //   调用方传的是 state.theme（UI 主题名）→ 旧 switch 永不命中 → mermaid 恒用暗色主题，
  //   于是亮色 UI 主题下图表仍是深色块。现改为按 isDarkTheme 双档 + 令牌取色。
  theme: { type: String, default: 'midnight' },
})

const uid = ref('md_' + Date.now().toString(36) + '_' + Math.random().toString(36).slice(2, 6))
const renderRef = ref(null)
const mermaidErrors = ref({})
const boxZoomed = ref({})

// ── Mermaid 初始化配置（按 UI 主题明暗双档，色值全部取自令牌） ──
// ★ 旧实现用 4 个硬编码主题名分支（'dark'|'light'|'warm'|'cute'），而 props.theme 实际收到的
//   是 UI 主题名（midnight/graphite/…/slate）→ switch 永不命中 → 恒落 default 的暗色主题
//   → 亮色 UI 主题下 mermaid 图仍渲染成深色块（亮色主题「深色割裂块」的主因之一）。
// ★ mermaid 生成 SVG 时会对颜色做运算（派生深浅色），故必须传入**真实色值**而非 var(--x)；
//   用 cssVar() 读取计算后的令牌值 → 既跟随主题、又是合法色值。
function cssVar(name, fallback) {
  try {
    const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
    return v || fallback
  } catch (e) { return fallback }
}

function getMermaidTheme(themeName) {
  const dark = isDarkTheme(themeName)
  // fallback 取默认主题（:root = midnight）的对应档值，仅在令牌未就绪的极端情况生效
  const f = dark
    ? { fg: '#E7ECF5', muted: '#8D9BB3', surface: '#151B29', surface2: '#1C2436', surface3: '#232b3c', bg: '#0F1420', border: '#2A3550', blue: '#6FA8FF', green: '#6BD98A', amber: '#F0C158', purple: '#C0A6F5', teal: '#3FD0E0', danger: '#FF8085' }
    : { fg: '#101418', muted: '#5A6B85', surface: '#FFFFFF', surface2: '#EEF1F6', surface3: '#E4E9F2', bg: '#F5F7FB', border: '#DCE3ED', blue: '#1B5FD9', green: '#186418', amber: '#8A5A00', purple: '#6B3FA0', teal: '#0A6E7A', danger: '#C42B2B' }
  const V = (n, k) => cssVar(n, f[k])
  return {
    theme: dark ? 'dark' : 'default',
    themeVariables: {
      primaryColor: V('--color-surface-2', 'surface2'),
      primaryTextColor: V('--color-fg', 'fg'),
      primaryBorderColor: V('--color-border', 'border'),
      lineColor: V('--color-cat-blue', 'blue'),
      secondaryColor: V('--color-surface', 'surface'),
      tertiaryColor: V('--color-surface-3', 'surface3'),
      clusterBkg: V('--color-bg', 'bg'),
      clusterBorder: V('--color-border', 'border'),
      edgeLabelBackground: V('--color-surface', 'surface'),
      nodeBorder: V('--color-cat-blue', 'blue'),
      nodeTextColor: V('--color-fg', 'fg'),
      background: 'transparent',
      mainBkg: V('--color-surface', 'surface'),
      signalColor: V('--color-cat-blue', 'blue'),
      signalTextColor: V('--color-fg', 'fg'),
      labelTextColor: V('--color-muted', 'muted'),
      noteBkgColor: V('--color-surface-2', 'surface2'),
      noteBorderColor: V('--color-border', 'border'),
      noteTextColor: V('--color-fg', 'fg'),
      // git 图 8 色：以品类色为主（保持可区分），不足处以语义色补齐
      git0: V('--color-cat-blue', 'blue'),
      git1: V('--color-cat-green', 'green'),
      git2: V('--color-cat-amber', 'amber'),
      git3: V('--color-danger', 'danger'),
      git4: V('--color-cat-purple', 'purple'),
      git5: V('--color-muted', 'muted'),
      git6: V('--color-cat-teal', 'teal'),
      git7: V('--color-warning', 'amber'),
    },
  }
}

// ── 布局类型图标映射 ──
function layoutIcon(lang) {
  const icons = {
    layout: '🎨', flex: '↔️', grid: '🔲', box: '📦', wireframe: '🏗️',
    card: '🃏', list: '📋', navbar: '🧭', sidebar: '📑', tabs: '📌',
    dashboard: '📊', form: '📝', modal: '🪟', tree: '🌳', flow: '➡️', map: '🧠',
  }
  return icons[lang] || '🎨'
}

// ── 布局类型中文标签 ──
function layoutTypeLabel(lang) {
  const labels = {
    layout: 'UI 布局', flex: 'Flexbox 弹性布局', grid: 'Grid 网格布局',
    box: 'Box Model 盒模型', wireframe: '线框图',
    card: '卡片布局', list: '列表布局', navbar: '导航栏',
    sidebar: '侧边栏', tabs: '标签页', dashboard: '仪表盘',
    form: '表单', modal: '弹窗', tree: '树形结构', flow: '流程图', map: '思维导图',
  }
  return labels[lang] || 'UI 布局'
}

// ── 解析 Markdown → 分块 ──
const blocks = computed(() => {
  const raw = props.text || ''
  const parts = []
  const chartLangRe = /^(?:mermaid|layout|flex|grid|box|wireframe|card|list|navbar|sidebar|tabs|dashboard|form|modal|tree|flow|map)$/i

  let lastIdx = 0
  const codeBlockRe = /```(\w*)\s*\n([\s\S]*?)```/g
  let match

  while ((match = codeBlockRe.exec(raw)) !== null) {
    if (match.index > lastIdx) {
      const text = raw.slice(lastIdx, match.index)
      const html = renderMd(text)
      parts.push({ type: 'html', content: html })
    }

    const lang = match[1].trim().toLowerCase()
    const code = match[2]

    if (chartLangRe.test(lang)) {
      if (lang === 'mermaid') {
        parts.push({ type: 'mermaid', code: code.trim(), title: detectChartTitle(code) })
      } else if (lang === 'layout' || lang === 'flex' || lang === 'grid' || lang === 'box' || lang === 'wireframe' || lang === 'card' || lang === 'list' || lang === 'navbar' || lang === 'sidebar' || lang === 'tabs' || lang === 'dashboard' || lang === 'form' || lang === 'modal' || lang === 'tree' || lang === 'flow' || lang === 'map') {
        const layoutResult = parseLayout(code.trim(), lang)
        parts.push({
          type: 'layout', title: layoutResult.title || layoutTypeLabel(lang),
          layoutStyle: lang, containerStyle: layoutResult.containerStyle,
          items: layoutResult.items, rawCode: code.trim(),
        })
      } else {
        const html = renderMd(match[0])
        parts.push({ type: 'html', content: html })
      }
    } else {
      const html = renderMd(match[0])
      parts.push({ type: 'html', content: html })
    }

    lastIdx = match.index + match[0].length
  }

  if (lastIdx < raw.length) {
    const text = raw.slice(lastIdx)
    const html = renderMd(text)
    parts.push({ type: 'html', content: html })
  }

  return parts
})

// ── 用 marked 渲染普通 markdown ──
function renderMd(text) {
  if (!text) return ''
  try { return marked.parse(text, { async: false }) } catch { return text }
}

// ── 检测图表标题（从代码的第一行注释或上下文） ──
function detectChartTitle(code) {
  const lines = code.split('\n')
  for (const line of lines) {
    const trimmed = line.trim()
    if (trimmed.startsWith('---') || trimmed.startsWith('%%') || trimmed.startsWith('#')) continue
    if (trimmed.startsWith('title ')) return trimmed.slice(6).trim().replace(/["']/g, '')
    if (trimmed.length > 0 && !trimmed.startsWith('flowchart') && !trimmed.startsWith('sequenceDiagram') &&
        !trimmed.startsWith('classDiagram') && !trimmed.startsWith('stateDiagram') && !trimmed.startsWith('gantt') &&
        !trimmed.startsWith('pie') && !trimmed.startsWith('erDiagram') && !trimmed.startsWith('journey') &&
        !trimmed.startsWith('gitgraph') && !trimmed.startsWith('mindmap') && !trimmed.startsWith('timeline') &&
        !trimmed.startsWith('quadrantChart') && !trimmed.startsWith('xychart') && !trimmed.startsWith('block') &&
        !trimmed.startsWith('sankey') && !trimmed.startsWith('requirement')) {
      return trimmed.length < 50 ? trimmed : ''
    }
    break
  }
  return ''
}

// ══════════════════════════════════════════════════════════════
//  UI 布局解析器（增强版：支持所有布局类型）
// ══════════════════════════════════════════════════════════════
// 支持的代码块标记: layout / flex / grid / box / wireframe / card / list /
//                  navbar / sidebar / tabs / dashboard / form / modal / tree / flow / map
//
// 语法格式：每行一个布局元素
//   前缀（语义）:  > 弹性区  . 固定区  # 强调区  + 内联项  - 次要区  * 装饰区
//   标签:  "显示文字" 或 直接文字
//   说明:  (括号内的辅助说明)
//   样式:  key:value key:value（如 bg:#1f2937 w:200px h:50px flex:1）
//
// 示例：
//   // 后台管理布局
//   #header "顶部导航" bg:#1f2937 h:50px
//   .sidebar "侧边栏" w:200px bg:#161b22
//   > main "主内容区" bg:#0d1117
//   footer "底部" h:40px bg:#1f2937
// ══════════════════════════════════════════════════════════════

function parseLayout(code, lang) {
  const lines = code.trim().split('\n').filter(l => l.trim())
  const items = []
  let title = ''
  const containerStyle = { display: 'flex', flexDirection: 'column', gap: '0px' }
  const isDark = getIsDark()

  // ── 按布局类型设置容器样式 ──
  switch (lang) {
    case 'flex': setupFlexContainer(code, containerStyle); break
    case 'grid': setupGridContainer(code, containerStyle); break
    case 'box': setupBoxContainer(containerStyle); break
    case 'wireframe': setupWireframeContainer(containerStyle); break
    case 'card': setupCardContainer(containerStyle); break
    case 'list': setupListContainer(containerStyle); break
    case 'navbar': setupNavbarContainer(containerStyle); break
    case 'sidebar': setupSidebarContainer(containerStyle); break
    case 'tabs': setupTabsContainer(containerStyle); break
    case 'dashboard': setupDashboardContainer(containerStyle); break
    case 'form': setupFormContainer(containerStyle); break
    case 'modal': setupModalContainer(containerStyle); break
    case 'tree': setupTreeContainer(containerStyle); break
    case 'flow': setupFlowContainer(containerStyle); break
    case 'map': setupMapContainer(containerStyle); break
    default: setupGenericContainer(containerStyle); break
  }

  // 提取标题
  const titleMatch = code.match(/\/\/\s*(.+)/)
  if (titleMatch) title = titleMatch[1].trim()

  // ── 解析 items（跳过声明行和注释行） ──
  const itemLines = filterItemLines(lines, lang)
  const bgColors = getBgColors(isDark)

  for (const line of itemLines) {
    const item = parseLayoutItem(line.trim(), bgColors, items.length, lang, isDark)
    if (item) items.push(item)
  }

  // ── 特殊布局的后处理 ──
  if (lang === 'box' && items.length === 0) {
    generateBoxModelItems(code, containerStyle, items, isDark)
  }
  if (lang === 'card' && items.length === 0) {
    generateCardItems(code, containerStyle, items, bgColors, isDark)
  }
  if (lang === 'list' && items.length === 0) {
    generateListItems(code, containerStyle, items, bgColors, isDark)
  }
  if (lang === 'tabs' && items.length === 0) {
    generateTabsItems(code, containerStyle, items, isDark)
  }

  return { title, containerStyle, items }
}

// ══════════════════════════════════════════════════════════════
//  各布局类型的容器样式设置
// ══════════════════════════════════════════════════════════════

function setupFlexContainer(code, style) {
  let dir = 'row', wrap = 'nowrap', justify = 'flex-start', align = 'stretch'
  const dirM = code.match(/direction\s*[=:]\s*(\w+)/i)
  if (dirM) dir = dirM[1] === 'column' ? 'column' : 'row'
  const wrapM = code.match(/wrap\s*[=:]\s*(\w+)/i)
  if (wrapM) wrap = wrapM[1] === 'wrap' ? 'wrap' : 'nowrap'
  const jM = code.match(/justify\s*[=:]\s*(\S+)/i)
  if (jM) justify = jM[1]
  const aM = code.match(/align\s*[=:]\s*(\S+)/i)
  if (aM) align = aM[1]
  Object.assign(style, {
    display: 'flex', flexDirection: dir, flexWrap: wrap,
    justifyContent: justify, alignItems: align, gap: '8px',
    padding: '12px', minHeight: '80px',
    background: 'var(--bg-secondary)', border: '1px dashed var(--border-color)', borderRadius: '6px',
  })
}

function setupGridContainer(code, style) {
  let cols = 'repeat(auto-fill, minmax(120px, 1fr))', rows = 'auto', gap = '8px'
  const cM = code.match(/columns?\s*[=:]\s*(.+)/i)
  if (cM) cols = cM[1].trim()
  const rM = code.match(/rows?\s*[=:]\s*(.+)/i)
  if (rM) rows = rM[1].trim()
  const gM = code.match(/gap\s*[=:]\s*(.+)/i)
  if (gM) gap = gM[1].trim()
  Object.assign(style, {
    display: 'grid', gridTemplateColumns: cols, gridTemplateRows: rows, gap,
    padding: '12px', minHeight: '80px',
    background: 'var(--bg-secondary)', border: '1px dashed var(--border-color)', borderRadius: '6px',
  })
}

function setupBoxContainer(style) {
  Object.assign(style, {
    position: 'relative', padding: '0', margin: '0', minHeight: '200px',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    background: 'var(--bg-tertiary)', borderRadius: '6px',
  })
}

function setupWireframeContainer(style) {
  Object.assign(style, {
    display: 'flex', flexDirection: 'column', gap: '0', padding: '0',
    minHeight: '100px', border: '1px dashed var(--border-color)',
    background: 'var(--bg-primary)', borderRadius: '4px',
  })
}

function setupCardContainer(style) {
  Object.assign(style, {
    display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))',
    gap: '12px', padding: '16px', minHeight: '100px',
    background: 'transparent',
  })
}

function setupListContainer(style) {
  Object.assign(style, {
    display: 'flex', flexDirection: 'column', gap: '6px',
    padding: '8px', minHeight: '60px',
    background: 'var(--bg-primary)', border: '1px solid var(--border-color)', borderRadius: '6px',
  })
}

function setupNavbarContainer(style) {
  Object.assign(style, {
    display: 'flex', flexDirection: 'row', gap: '4px',
    padding: '8px 16px', minHeight: '44px',
    background: 'var(--bg-primary)', border: '1px solid var(--border-color)',
    borderRadius: '8px', alignItems: 'center',
  })
}

function setupSidebarContainer(style) {
  Object.assign(style, {
    display: 'flex', flexDirection: 'row', gap: '0',
    padding: '0', minHeight: '200px',
    background: 'var(--bg-primary)', border: '1px solid var(--border-color)', borderRadius: '8px',
    overflow: 'hidden',
  })
}

function setupTabsContainer(style) {
  Object.assign(style, {
    display: 'flex', flexDirection: 'row', gap: '2px',
    padding: '8px 8px 0', minHeight: '36px',
    background: 'var(--bg-primary)', borderBottom: '2px solid var(--border-color)',
    alignItems: 'flex-end',
  })
}

function setupDashboardContainer(style) {
  Object.assign(style, {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))',
    gap: '12px', padding: '16px', minHeight: '200px',
    background: 'var(--bg-secondary)', borderRadius: '8px',
  })
}

function setupFormContainer(style) {
  Object.assign(style, {
    display: 'flex', flexDirection: 'column', gap: '10px',
    padding: '20px', minHeight: '150px',
    background: 'var(--bg-primary)', border: '1px solid var(--border-color)',
    borderRadius: '8px', maxWidth: '400px',
  })
}

function setupModalContainer(style) {
  Object.assign(style, {
    display: 'flex', flexDirection: 'column', gap: '0',
    padding: '0', minHeight: '200px',
    background: 'var(--bg-primary)', border: '1px solid var(--border-color)',
    borderRadius: '12px', overflow: 'hidden',
    boxShadow: 'var(--shadow-lg)', maxWidth: '500px', margin: '0 auto',
  })
}

function setupTreeContainer(style) {
  Object.assign(style, {
    display: 'flex', flexDirection: 'column', gap: '4px',
    padding: '12px', minHeight: '80px',
    background: 'transparent', position: 'relative',
  })
}

function setupFlowContainer(style) {
  Object.assign(style, {
    display: 'flex', flexDirection: 'row', gap: '6px',
    padding: '16px', minHeight: '60px', alignItems: 'center',
    flexWrap: 'wrap', justifyContent: 'center',
    background: 'var(--bg-secondary)', borderRadius: '8px',
  })
}

function setupMapContainer(style) {
  Object.assign(style, {
    display: 'flex', flexDirection: 'column', gap: '6px',
    padding: '16px', minHeight: '100px',
    background: 'transparent', position: 'relative',
  })
}

function setupGenericContainer(style) {
  Object.assign(style, {
    display: 'flex', flexDirection: 'column', gap: '6px',
    padding: '12px', background: 'var(--bg-secondary)',
    border: '1px dashed var(--border-color)', borderRadius: '6px',
  })
}

// ══════════════════════════════════════════════════════════════
//  布局项过滤与解析
// ══════════════════════════════════════════════════════════════

function filterItemLines(lines, lang) {
  const skipPrefixes = ['//', 'direction', 'wrap', 'justify', 'align', 'columns', 'rows', 'gap', 'title']
  return lines.filter(l => {
    const t = l.trim()
    return !skipPrefixes.some(p => t.startsWith(p))
  })
}

function getIsDark() {
  // ★ 原实现查 documentElement 的 'dark' class、body 的 'dark-mode' class 与
  //   prefers-color-scheme。但主题 class 实为 theme-<名>（如 theme-daylight），
  //   前两者恒为 false；末者跟随**操作系统**而非 UI 主题 → 配色与 UI 主题脱钩，
  //   造成亮色 UI 主题下仍用暗色档（割裂块成因）。改为按 props.theme 判定。
  return isDarkTheme(props.theme)
}

function getBgColors(isDark) {
  // 布局/流程图节点用可区分的类别色（8 类）。改用令牌 → 跟随 UI 主题，不再两档硬编码。
  // ★ 调用方按 `bgColors[ci] + '20'` / `+ '55'` 拼 alpha（#6FA8FF + 20 → #6FA8FF20），
  //   故此处必须返回 **6 位 hex**：令牌值均为 hex，符合要求；cssVar 失败则回退同档 hex。
  // ★ 类别色须保持可区分，不得塌成 4 个语义色（详见 design/ui-theme.notes.md §18）。
  const fb = isDark
    ? ['#6FA8FF', '#6BD98A', '#F0C158', '#FF8085', '#C0A6F5', '#3FD0E0', '#8D9BB3', '#4FD8A4']
    : ['#1B5FD9', '#186418', '#8A5A00', '#C42B2B', '#6B3FA0', '#0A6E7A', '#5A6B85', '#0F7A4A']
  const names = ['--color-cat-blue', '--color-cat-green', '--color-cat-amber', '--color-danger',
    '--color-cat-purple', '--color-cat-teal', '--color-muted', '--color-success']
  return names.map((n, i) => {
    const v = cssVar(n, fb[i])
    return /^#[0-9a-fA-F]{6}$/.test(v) ? v : fb[i]
  })
}

// 解析单个布局项（增强版）
function parseLayoutItem(line, bgColors, idx, lang, isDark) {
  // 语法: [前缀] "标签" (说明) key:value key:value
  let prefix = '', label = '', sub = '', icon = ''
  let remaining = line

  // 提取前缀
  const prefixMatch = remaining.match(/^([.>#+\-*!@])\s*/)
  if (prefixMatch) { prefix = prefixMatch[1]; remaining = remaining.slice(prefixMatch[0].length) }

  // 提取引号标签
  const quoteMatch = remaining.match(/"([^"]*)"/)
  if (quoteMatch) { label = quoteMatch[1]; remaining = remaining.slice(quoteMatch[0].length).trim() }
  else {
    const wordMatch = remaining.match(/^(\S+)/)
    if (wordMatch) { label = wordMatch[1]; remaining = remaining.slice(wordMatch[0].length).trim() }
  }

  // 提取 sub（括号说明）
  const subMatch = remaining.match(/\(([^)]+)\)/)
  if (subMatch) { sub = subMatch[1]; remaining = remaining.slice(subMatch[0].length).trim() }

  if (!label) return null

  // 解析样式键值对
  const customStyle = {}
  const stylePairs = remaining.match(/(\S+):(\S+)/g)
  if (stylePairs) {
    for (const pair of stylePairs) {
      const sepIdx = pair.indexOf(':')
      const key = pair.slice(0, sepIdx)
      let val = pair.slice(sepIdx + 1)
      const cssKey = key.replace(/-([a-z])/g, (_, c) => c.toUpperCase())
      customStyle[cssKey] = val
    }
  }

  // 构建基础样式
  const itemStyle = {
    borderRadius: '6px', display: 'flex', flexDirection: 'column',
    alignItems: 'center', justifyContent: 'center',
    padding: '8px 12px', fontSize: '12px', fontWeight: 500,
    textAlign: 'center', minHeight: '36px', overflow: 'hidden',
    position: 'relative', transition: 'all 0.15s ease',
    ...customStyle,
  }

  // 自动配色
  if (!customStyle.background && !customStyle.backgroundColor && !customStyle.bg) {
    const ci = idx % bgColors.length
    itemStyle.background = bgColors[ci] + '20'
    itemStyle.border = '1px solid ' + bgColors[ci] + '55'
    itemStyle.color = 'var(--text-primary)'
  } else if (customStyle.bg) {
    itemStyle.background = customStyle.bg
    delete itemStyle.bg
  }

  // 简写：w→width, h→height
  if (customStyle.w) { itemStyle.width = customStyle.w; delete itemStyle.w }
  if (customStyle.h) { itemStyle.height = customStyle.h; delete itemStyle.h }

  // 根据前缀调整样式
  switch (prefix) {
    case '>': itemStyle.flex = '1'; itemStyle.borderStyle = 'dashed'; break
    case '.': itemStyle.flexShrink = '0'; break
    case '#': itemStyle.fontWeight = 700; itemStyle.borderWidth = '2px'; break
    case '+': itemStyle.flexShrink = '0'; itemStyle.flexGrow = '0'; break
    case '-': itemStyle.opacity = '0.65'; itemStyle.fontSize = '11px'; break
    case '*': itemStyle.fontStyle = 'italic'; itemStyle.opacity = '0.8'; break
    case '!': itemStyle.fontWeight = 700; itemStyle.borderColor = 'var(--color-danger)'; break
    case '@': itemStyle.borderStyle = 'dotted'; itemStyle.opacity = '0.7'; break
  }

  // 布局类型特殊样式
  if (lang === 'card') {
    itemStyle.borderRadius = '8px'
    itemStyle.padding = '12px 16px'
    itemStyle.minHeight = '60px'
    itemStyle.boxShadow = 'var(--shadow-sm)'
    if (idx === 0) itemStyle.gridColumn = '1 / -1' // 首项可能跨列
  }
  if (lang === 'navbar') {
    itemStyle.padding = '6px 14px'
    itemStyle.minHeight = '32px'
    itemStyle.borderRadius = '6px'
    itemStyle.fontSize = '12px'
    itemStyle.flexShrink = '0'
  }
  if (lang === 'sidebar') {
    itemStyle.borderRadius = '0'
    itemStyle.minHeight = '40px'
    itemStyle.justifyContent = 'flex-start'
    itemStyle.padding = '8px 16px'
    itemStyle.textAlign = 'left'
    itemStyle.alignItems = 'flex-start'
    if (prefix === '.' || idx === 0) itemStyle.width = '180px'; itemStyle.flexShrink = '0'
    if (prefix === '>') itemStyle.flex = '1'
  }
  if (lang === 'tabs') {
    itemStyle.borderRadius = '6px 6px 0 0'
    itemStyle.padding = '6px 16px'
    itemStyle.minHeight = '32px'
    itemStyle.fontSize = '12px'
    itemStyle.flexShrink = '0'
    if (idx === 0) {
      itemStyle.background = 'var(--color-accent-bg)'
      itemStyle.borderBottom = '2px solid var(--color-cat-blue)'
    }
  }
  if (lang === 'form') {
    itemStyle.borderRadius = '4px'
    itemStyle.padding = '10px 14px'
    itemStyle.minHeight = '36px'
    itemStyle.alignItems = 'flex-start'
    itemStyle.textAlign = 'left'
    // 模拟表单输入框
    if (prefix === '-') {
      itemStyle.background = 'var(--bg-secondary)'
      itemStyle.border = '1px solid var(--border-color)'
      itemStyle.cursor = 'text'
      itemStyle.minHeight = '32px'
    }
  }
  if (lang === 'modal') {
    if (idx === 0) { // 标题区
      itemStyle.borderRadius = '12px 12px 0 0'
      itemStyle.padding = '16px 20px'
      itemStyle.fontWeight = 700
      itemStyle.fontSize = '14px'
      itemStyle.justifyContent = 'flex-start'
      itemStyle.textAlign = 'left'
      itemStyle.alignItems = 'flex-start'
    } else if (idx === items.length - 1) { // 底部
      itemStyle.borderRadius = '0 0 12px 12px'
      itemStyle.padding = '12px 20px'
      itemStyle.flexDirection = 'row'
      itemStyle.justifyContent = 'flex-end'
      itemStyle.gap = '8px'
    } else { // 内容区
      itemStyle.padding = '20px'
      itemStyle.flex = '1'
      itemStyle.alignItems = 'flex-start'
      itemStyle.textAlign = 'left'
    }
  }
  if (lang === 'tree') {
    itemStyle.padding = '4px 12px'
    itemStyle.minHeight = '28px'
    itemStyle.flexDirection = 'row'
    itemStyle.justifyContent = 'flex-start'
    itemStyle.textAlign = 'left'
    itemStyle.alignItems = 'center'
    itemStyle.gap = '6px'
    itemStyle.marginLeft = (idx * 20) + 'px'
    itemStyle.borderRadius = '4px'
    if (idx === 0) itemStyle.fontWeight = 700
  }
  if (lang === 'flow') {
    itemStyle.borderRadius = '20px'
    itemStyle.padding = '8px 18px'
    itemStyle.minHeight = '36px'
    itemStyle.flexShrink = '0'
    if (idx < items.length - 1) itemStyle.marginRight = '12px'
  }
  if (lang === 'map') {
    itemStyle.padding = '6px 14px'
    itemStyle.minHeight = '28px'
    itemStyle.flexDirection = 'row'
    itemStyle.justifyContent = 'flex-start'
    itemStyle.textAlign = 'left'
    itemStyle.alignItems = 'center'
    itemStyle.gap = '4px'
    itemStyle.borderRadius = '20px'
    itemStyle.marginLeft = (idx * 24) + 'px'
  }
  if (lang === 'dashboard') {
    itemStyle.borderRadius = '8px'
    itemStyle.padding = '16px'
    itemStyle.minHeight = '80px'
    itemStyle.boxShadow = 'var(--shadow-sm)'
    if (idx === 0) itemStyle.gridColumn = '1 / -1'
    if (idx <= 2) itemStyle.minHeight = '60px'
  }
  if (lang === 'list') {
    itemStyle.borderRadius = '4px'
    itemStyle.padding = '8px 14px'
    itemStyle.minHeight = '32px'
    itemStyle.flexDirection = 'row'
    itemStyle.justifyContent = 'flex-start'
    itemStyle.textAlign = 'left'
    itemStyle.alignItems = 'center'
    itemStyle.gap = '8px'
    itemStyle.border = 'none'
    itemStyle.borderBottom = '1px solid var(--border-color)'
    if (idx === items.length - 1) itemStyle.borderBottom = 'none'
  }

  return { label, sub, icon, style: itemStyle }
}

// ══════════════════════════════════════════════════════════════
//  特殊布局类型的默认内容生成器（无用户定义项时的兜底）
// ══════════════════════════════════════════════════════════════

function generateBoxModelItems(code, containerStyle, items, isDark) {
  const margin = parseInt(code.match(/margin\s*[=:]\s*(\d+)/i)?.[1] || '20')
  const border = parseInt(code.match(/border\s*[=:]\s*(\d+)/i)?.[1] || '3')
  const padding = parseInt(code.match(/padding\s*[=:]\s*(\d+)/i)?.[1] || '15')
  const contentText = code.match(/content\s*[=:]\s*"([^"]+)"/)?.[1] || '内容区域'
  const totalSize = margin * 2 + border * 2 + padding * 2 + 60
  containerStyle.minHeight = totalSize + 'px'

  items.push({
    label: 'margin: ' + margin + 'px', sub: '', icon: '',
    style: { position: 'absolute', top: '0', left: '0', right: '0', bottom: '0',
      background: 'var(--color-danger-bg)',
      border: '2px dashed var(--color-danger)',
      borderRadius: '4px', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0', }
  })
  const bOff = margin
  items.push({
    label: 'border: ' + border + 'px solid', sub: '', icon: '',
    style: { position: 'absolute', top: bOff + 'px', left: bOff + 'px', right: bOff + 'px', bottom: bOff + 'px',
      background: 'var(--color-warning-bg)',
      border: border + 'px solid var(--color-warning)', borderRadius: '3px',
      display: 'flex', alignItems: 'center', justifyContent: 'center', }
  })
  const pOff = bOff + border
  items.push({
    label: 'padding: ' + padding + 'px', sub: '', icon: '',
    style: { position: 'absolute', top: pOff + 'px', left: pOff + 'px', right: pOff + 'px', bottom: pOff + 'px',
      background: 'var(--color-success-bg)',
      border: '2px dashed var(--color-success)', borderRadius: '2px',
      display: 'flex', alignItems: 'center', justifyContent: 'center', }
  })
  const cOff = pOff + padding
  items.push({
    label: contentText, sub: '', icon: '',
    style: { position: 'absolute', top: cOff + 'px', left: cOff + 'px', right: cOff + 'px', bottom: cOff + 'px',
      background: 'var(--color-info-bg)',
      border: '1px solid var(--color-info)', borderRadius: '2px',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      fontSize: '12px', fontWeight: 600, color: 'var(--text-primary)', }
  })
}

function generateCardItems(code, containerStyle, items, bgColors, isDark) {
  const cards = code.split('\n').filter(l => l.trim() && !l.trim().startsWith('//'))
  if (cards.length === 0) {
    for (let i = 0; i < 3; i++) {
      items.push({
        label: '卡片 ' + (i + 1), sub: '卡片描述内容', icon: '',
        style: { borderRadius: '8px', padding: '14px 16px', minHeight: '70px',
          background: bgColors[i % bgColors.length] + '18',
          border: '1px solid ' + bgColors[i % bgColors.length] + '44',
          display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
          boxShadow: 'var(--shadow-sm)', }
      })
    }
  }
}

function generateListItems(code, containerStyle, items, bgColors, isDark) {
  const entries = code.split('\n').filter(l => l.trim() && !l.trim().startsWith('//'))
  if (entries.length === 0) {
    const labels = ['项目一', '项目二', '项目三', '项目四']
    for (let i = 0; i < labels.length; i++) {
      items.push({
        label: labels[i], sub: '', icon: '',
        style: { borderRadius: '4px', padding: '8px 14px', minHeight: '32px',
          flexDirection: 'row', justifyContent: 'flex-start', textAlign: 'left',
          alignItems: 'center', gap: '8px',
          background: i % 2 === 0 ? 'var(--bg-hover)' : 'transparent',
          border: 'none', borderBottom: '1px solid var(--border-color)',
          fontSize: '12px', }
      })
    }
    if (items.length > 0) items[items.length - 1].style.borderBottom = 'none'
  }
}

function generateTabsItems(code, containerStyle, items, isDark) {
  const tabs = ['标签一', '标签二', '标签三']
  for (let i = 0; i < tabs.length; i++) {
    items.push({
      label: tabs[i], sub: '', icon: '',
      style: { borderRadius: '6px 6px 0 0', padding: '6px 18px', minHeight: '32px',
        fontSize: '12px', flexShrink: '0', cursor: 'pointer',
        background: i === 0 ? 'var(--color-accent-bg)' : 'transparent',
        border: '1px solid var(--border-color)', borderBottom: i === 0 ? '2px solid var(--color-cat-blue)' : '1px solid var(--border-color)',
        marginBottom: i === 0 ? '0' : '-1px', color: 'var(--text-primary)', }
    })
  }
}

// ══════════════════════════════════════════════════════════════
//  Box Model 缩放切换
// ══════════════════════════════════════════════════════════════
function toggleBoxZoom(i) {
  boxZoomed.value = { ...boxZoomed.value, [i]: !boxZoomed.value[i] }
}

// ── 解析表格文本为 Chart.js 数据格式 ──

function retryMermaid(i) { mermaidErrors.value = { ...mermaidErrors.value, [i]: false }; nextTick(() => initMermaid()) }

// ── 初始化 Mermaid 渲染 ──
async function initMermaid() {
  if (!renderRef.value) return
  // ★ 先收集**待渲染**容器：一个都没有就直接返回 —— 绝不加载 3.6MB 的 mermaid。
  const pending = []
  for (const el of renderRef.value.querySelectorAll('.mermaid-container')) {
    if (el.querySelector('svg')) continue                  // 已渲染
    const srcEl = el.querySelector('.mermaid-src'); if (!srcEl) continue
    const code = srcEl.textContent.trim(); if (!code) continue
    pending.push({ el, code })
  }
  if (pending.length === 0) return
  let mm
  try { mm = await loadMermaid() } catch (e) {
    console.warn('[Mermaid] 加载失败:', e && e.message)
    return
  }
  const themeConfig = getMermaidTheme(props.theme)
  try { mm.initialize({ startOnLoad: false, ...themeConfig, securityLevel: 'loose', fontFamily: '"Inter", system-ui, sans-serif' }) } catch {}
  for (const { el, code } of pending) {
    const id = el.id; if (!id) continue
    try {
      const { svg } = await mm.render(id + '-svg', code)
      el.innerHTML = svg
    } catch (err) {
      console.warn('[Mermaid] 渲染失败:', err.message)
      const parts = id.replace('mermaid-', '').split('-')
      mermaidErrors.value = { ...mermaidErrors.value, [parseInt(parts[1])]: true }
    }
  }
}

// ── 生命周期 ──
onMounted(() => { nextTick(() => { initMermaid() }) })
onBeforeUnmount(() => {
  // 清理 Mermaid SVG 实例（防止内存泄漏）
  if (renderRef.value) {
    renderRef.value.querySelectorAll('.mermaid-container svg').forEach(el => el.remove())
  }
  mermaidErrors.value = {}
  boxZoomed.value = {}
})
watch(() => props.text, () => { mermaidErrors.value = {}; boxZoomed.value = {}; nextTick(() => { initMermaid() }) })
watch(() => props.theme, () => {
  nextTick(() => {
    if (renderRef.value) renderRef.value.querySelectorAll('.mermaid-container svg').forEach(el => el.remove())
    initMermaid()
  })
})
</script>

<style scoped>
/* ═══════════════ Markdown 渲染器 ═══════════════ */
.markdown-renderer { width: 100%; max-width: 100%; overflow-x: hidden; }
.md-html { line-height: 1.6; word-break: break-word; overflow-wrap: break-word; }

/* ── Markdown 内容样式（紧凑间距） ── */
.md-html :deep(h1), .md-html :deep(h2), .md-html :deep(h3), .md-html :deep(h4) { margin: 6px 0 2px; font-weight: 600; }
.md-html :deep(h1) { font-size: 15px; }
.md-html :deep(h2) { font-size: 14px; }
.md-html :deep(h3) { font-size: 13px; }
.md-html :deep(p) { margin: 2px 0; }
.md-html :deep(ul) { margin: 2px 0; padding-left: 18px; }
.md-html :deep(ol) { margin: 2px 0; padding-left: 28px; }
.md-html :deep(li) { margin: 1px 0; }
.md-html :deep(code) { background: var(--color-surface-2); padding: 1px 4px; border-radius: 3px; font-family: var(--font-code); font-size: 12px; }
.md-html :deep(pre) { background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 6px; padding: 6px 10px; margin: 4px 0; overflow-x: auto; font-size: 12px; line-height: 1.4; white-space: pre-wrap; word-break: break-word; max-width: 100%; }
.md-html :deep(pre code) { background: none; padding: 0; border-radius: 0; white-space: pre-wrap; word-break: break-word; }
.md-html :deep(blockquote) { border-left: 3px solid var(--accent); padding-left: 8px; margin: 4px 0; color: var(--text-secondary); font-style: italic; }
.md-html :deep(a) { color: var(--accent-light); text-decoration: none; }
.md-html :deep(a:hover) { text-decoration: underline; }
.md-html :deep(table) { border-collapse: collapse; margin: 4px 0; font-size: 12px; width: 100%; }
.md-html :deep(th), .md-html :deep(td) { border: 1px solid var(--border-color); padding: 3px 6px; text-align: left; }
.md-html :deep(th) { background: var(--bg-tertiary); font-weight: 600; }
.md-html :deep(hr) { border: none; border-top: 1px solid var(--border-color); margin: 6px 0; }
.md-html :deep(img) { max-width: 100%; border-radius: 4px; }
.md-html :deep(strong) { font-weight: 600; }
.md-html :deep(em) { font-style: italic; }

/* ── 图表公共样式 ── */
.chart-block {
  margin: 12px 0; border: 1px solid var(--border-color); border-radius: 8px;
  overflow: hidden; background: var(--bg-primary);
  transition: all 0.2s ease; animation: chartFadeIn 0.3s ease-out;
}
@keyframes chartFadeIn { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: translateY(0); } }
.chart-block:hover { border-color: var(--accent); box-shadow: var(--shadow-sm); }
.chart-label {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 12px; background: var(--bg-tertiary);
  border-bottom: 1px solid var(--border-color);
  font-size: 12px; font-weight: 500; color: var(--text-secondary);
}
.chart-icon { font-size: 14px; flex-shrink: 0; }
.chart-title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.chart-type-badge {
  font-size: 10px; padding: 1px 6px; border-radius: 3px;
  background: var(--bg-hover); color: var(--text-muted);
  border: 1px solid var(--border-color); flex-shrink: 0;
}
.chart-retry-btn {
  background: var(--bg-hover); border: 1px solid var(--border-color);
  color: var(--text-muted); padding: 1px 6px; border-radius: 3px;
  cursor: pointer; font-size: 11px;
}
.chart-retry-btn:hover { color: var(--text-primary); border-color: var(--accent); }

/* ── Mermaid 图表容器 ── */
.mermaid-container { padding: 16px; overflow-x: auto; overflow-y: hidden; display: flex; justify-content: center; min-height: 60px; max-width: 100%; }
.mermaid-container svg { max-width: 100%; height: auto; }
@media (max-width: 600px) { .mermaid-container { padding: 8px; } }
.mermaid-src { display: none; }
.chart-error .mermaid-container { display: none; }
.chart-error-msg { padding: 12px; }
.chart-error-msg pre { margin: 0; padding: 8px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 4px; font-size: 11px; overflow-x: auto; }
.chart-error-msg code { font-family: var(--font-code); white-space: pre; }
.chart-error-hint { margin: 6px 0 0; font-size: 11px; color: var(--color-danger); opacity: 0.7; }

/* ── 数据图表 Canvas 容器 ── */

/* ── 源数据表切换 ── */
.chart-table-toggle { border-top: 1px solid var(--border-color); }
.chart-table-toggle summary { padding: 6px 12px; font-size: 11px; color: var(--text-muted); cursor: pointer; user-select: none; transition: background 0.12s; }
.chart-table-toggle summary:hover { background: var(--bg-hover); color: var(--text-secondary); }
.chart-table-wrap { padding: 8px 12px; overflow-x: auto; }
.chart-table-wrap :deep(table) { width: 100%; border-collapse: collapse; font-size: 11px; }
.chart-table-wrap :deep(th), .chart-table-wrap :deep(td) { border: 1px solid var(--border-color); padding: 3px 6px; text-align: left; }
.chart-table-wrap :deep(th) { background: var(--bg-tertiary); font-weight: 600; }

/* ═══════════════ UI 布局可视化（增强版） ═══════════════ */
.layout-block {
  margin: 12px 0; border: 1px solid var(--border-color); border-radius: 8px;
  overflow: hidden; background: var(--bg-primary);
  transition: all 0.2s ease; animation: chartFadeIn 0.3s ease-out;
}
.layout-block:hover { border-color: var(--accent); box-shadow: var(--shadow-sm); }

.layout-canvas-wrap {
  padding: 16px; min-height: 60px;
  overflow-x: auto; overflow-y: auto; max-height: 450px;
  transition: all 0.2s ease;
}
.layout-zoomed { max-height: 800px; }

.layout-preview { position: relative; min-height: 50px; transition: all 0.15s ease; }

.layout-item {
  cursor: default;
  transition: transform 0.12s ease, box-shadow 0.12s ease;
  position: relative;
}
.layout-item:hover {
  transform: scale(1.03); z-index: 2;
  box-shadow: var(--shadow-md);
}
.layout-item-icon { font-size: 14px; margin-bottom: 2px; pointer-events: none; }
.layout-item-label { font-size: 11px; font-weight: 600; line-height: 1.3; pointer-events: none; }
.layout-item-sub { font-size: 9px; opacity: 0.65; pointer-events: none; margin-top: 2px; }

.layout-empty-hint {
  display: flex; align-items: center; justify-content: center;
  padding: 24px; color: var(--text-muted); font-size: 12px;
  font-style: italic; opacity: 0.6;
}

.layout-code-wrap { padding: 8px 12px; overflow-x: auto; }
.layout-code-wrap pre { margin: 0; padding: 8px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 4px; font-size: 11px; overflow-x: auto; }
.layout-code-wrap code { font-family: var(--font-code); white-space: pre; }

/* ── 各布局类型特殊样式 ── */
.layout-style-flex .layout-preview,
.layout-style-grid .layout-preview { background: var(--bg-secondary); border: 1px dashed var(--border-color); border-radius: 6px; }
.layout-style-box .layout-preview { min-height: 200px; background: var(--bg-tertiary); border-radius: 6px; }
.layout-style-wireframe .layout-preview { background: var(--bg-primary); }
.layout-style-wireframe .layout-item { border: 1px solid var(--border-color); background: var(--bg-secondary) !important; min-height: 40px; border-radius: 3px; }
.layout-style-wireframe .layout-item:hover { background: var(--bg-hover) !important; }
.layout-style-card .layout-preview { background: transparent; }
.layout-style-card .layout-item { min-height: 60px; }
.layout-style-list .layout-preview { background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 6px; }
.layout-style-list .layout-item { border-bottom: 1px solid var(--border-color) !important; }
.layout-style-list .layout-item:last-child { border-bottom: none !important; }
.layout-style-navbar .layout-preview { background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 8px; }
.layout-style-sidebar .layout-preview { background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 8px; overflow: hidden; }
.layout-style-sidebar .layout-item:first-child { border-right: 1px solid var(--border-color); }
.layout-style-tabs .layout-preview { background: transparent; }
.layout-style-tabs .layout-item { border: 1px solid var(--border-color); }
.layout-style-dashboard .layout-preview { background: var(--bg-secondary); border-radius: 8px; }
.layout-style-dashboard .layout-item { min-height: 60px; }
.layout-style-form .layout-preview { background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 8px; }
.layout-style-modal .layout-preview { background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 12px; overflow: hidden; box-shadow: var(--shadow-lg); }
.layout-style-tree .layout-preview { background: transparent; }
.layout-style-tree .layout-item { border-left: 2px solid var(--border-color); }
.layout-style-flow .layout-preview { background: var(--bg-secondary); border-radius: 8px; }
.layout-style-flow .layout-item { min-height: 32px; position: relative; }
.layout-style-flow .layout-item::after {
  content: '→'; position: absolute; right: -12px; top: 50%;
  transform: translateY(-50%); color: var(--text-muted); font-size: 14px; opacity: 0.5;
}
.layout-style-flow .layout-item:last-child::after { display: none; }
.layout-style-map .layout-preview { background: transparent; }
.layout-style-map .layout-item { border-radius: 20px; }
</style>
