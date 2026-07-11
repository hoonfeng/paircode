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

      <!-- 数据图表（从表格自动生成） -->
      <div v-else-if="block.type === 'table-chart'" class="chart-block chart-data">
        <div class="chart-label">
          <span class="chart-icon">📈</span>
          <span class="chart-title">{{ block.title }}</span>
        </div>
        <div class="chart-canvas-wrap">
          <canvas :id="'chart-' + uid + '-' + i"
                  class="data-chart-canvas"
                  :data-chart-data="block.chartData ? JSON.stringify(block.chartData) : ''"
                  :data-chart-type="block.chartType">
          </canvas>
        </div>
        <details class="chart-table-toggle">
          <summary>📋 查看源数据表</summary>
          <div class="chart-table-wrap" v-html="block.tableHtml"></div>
        </details>
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
import mermaid from 'mermaid'

const props = defineProps({
  text: { type: String, default: '' },
  // 图表主题：'dark' | 'light' | 'warm' | 'cute'
  theme: { type: String, default: 'dark' },
})

const uid = ref('md_' + Date.now().toString(36) + '_' + Math.random().toString(36).slice(2, 6))
const renderRef = ref(null)
const mermaidErrors = ref({})
const boxZoomed = ref({})

// ── Mermaid 初始化配置（按主题） ──
function getMermaidTheme(themeName) {
  switch (themeName) {
    case 'dark': return { theme: 'dark',
      themeVariables: {
        primaryColor: '#1f2937', primaryTextColor: '#e6edf3', primaryBorderColor: '#30363d',
        lineColor: '#58a6ff', secondaryColor: '#161b22', tertiaryColor: '#21262d',
        clusterBkg: '#0d1117', clusterBorder: '#30363d', edgeLabelBackground: '#161b22',
        nodeBorder: '#58a6ff', nodeTextColor: '#e6edf3', background: 'transparent',
        mainBkg: '#161b22', signalColor: '#58a6ff', signalTextColor: '#e6edf3',
        labelTextColor: '#8b949e', noteTextColor: '#e6edf3', noteBkgColor: '#21262d',
        noteBorderColor: '#30363d', git0: '#1f6feb', git1: '#3fb950', git2: '#d29922',
        git3: '#f85149', git4: '#bc8cff', git5: '#6e7681', git6: '#58a6ff', git7: '#da3633',
      },
    }
    case 'light': return { theme: 'default',
      themeVariables: {
        primaryColor: '#e8eaed', primaryTextColor: '#1a1a2e', primaryBorderColor: '#dadce0',
        lineColor: '#1a73e8', secondaryColor: '#f8f9fa', tertiaryColor: '#f0f1f3',
        clusterBkg: '#ffffff', clusterBorder: '#dadce0', edgeLabelBackground: '#f8f9fa',
        nodeBorder: '#1a73e8', nodeTextColor: '#1a1a2e', background: 'transparent',
        mainBkg: '#f8f9fa',
      },
    }
    case 'warm': return { theme: 'default',
      themeVariables: {
        primaryColor: '#efe4d4', primaryTextColor: '#3d2c1e', primaryBorderColor: '#d6c8b8',
        lineColor: '#b87333', secondaryColor: '#f5ece0', tertiaryColor: '#efe4d4',
        clusterBkg: '#faf3e8', clusterBorder: '#d6c8b8', edgeLabelBackground: '#f5ece0',
        nodeBorder: '#b87333', nodeTextColor: '#3d2c1e', background: 'transparent',
        mainBkg: '#f5ece0',
      },
    }
    case 'cute': return { theme: 'default',
      themeVariables: {
        primaryColor: '#fce4ec', primaryTextColor: '#4a1a2e', primaryBorderColor: '#e8b8c8',
        lineColor: '#e84393', secondaryColor: '#fff5f7', tertiaryColor: '#f8d7e0',
        clusterBkg: '#fff5f7', clusterBorder: '#e8b8c8', edgeLabelBackground: '#fce4ec',
        nodeBorder: '#e84393', nodeTextColor: '#4a1a2e', background: 'transparent',
        mainBkg: '#fce4ec',
      },
    }
    default: return { theme: 'dark' }
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
  const chartLangRe = /^(?:mermaid|chart|chart-bar|chart-line|chart-pie|chart-radar|graphviz|vega|vega-lite|layout|flex|grid|box|wireframe|card|list|navbar|sidebar|tabs|dashboard|form|modal|tree|flow|map)$/i

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
      } else if (lang.startsWith('chart-')) {
        const chartType = lang.replace('chart-', '')
        const chartData = parseTableTextToChartData(code, chartType)
        parts.push({
          type: 'table-chart', chartType, chartData,
          title: detectChartTitle(code) || `${chartType.toUpperCase()} 图表`,
          tableHtml: code.trim() ? renderMd('```\n' + code.trim() + '\n```') : '',
        })
      } else {
        const html = renderMd(match[0])
        parts.push({ type: 'html', content: html })
      }
    } else if (lang === '' || lang === 'csv' || lang === 'data') {
      const chartData = tryParseTableData(code.trim())
      if (chartData) {
        parts.push({
          type: 'table-chart', chartType: 'bar', chartData,
          title: detectChartTitle(code) || '数据图表',
          tableHtml: renderMd('```\n' + code.trim() + '\n```'),
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

  // 检测 markdown 表格并转换为图表
  const finalParts = []
  for (const part of parts) {
    if (part.type === 'html') {
      const tableCharts = extractTablesFromHtml(part.content)
      if (tableCharts.length > 0) {
        let remaining = part.content
        for (const tc of tableCharts) {
          const before = remaining.slice(0, tc.offset)
          const after = remaining.slice(tc.offset + tc.html.length)
          if (before.trim()) finalParts.push({ type: 'html', content: before })
          finalParts.push({
            type: 'table-chart', chartType: 'bar', chartData: tc.data,
            title: tc.title || '数据图表', tableHtml: tc.html,
          })
          remaining = after
        }
        if (remaining.trim()) finalParts.push({ type: 'html', content: remaining })
      } else {
        finalParts.push(part)
      }
    } else {
      finalParts.push(part)
    }
  }

  return finalParts
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
    boxShadow: '0 8px 32px rgba(0,0,0,0.3)', maxWidth: '500px', margin: '0 auto',
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
  return document.documentElement.classList.contains('dark') ||
         document.body.classList.contains('dark-mode') ||
         window.matchMedia?.('(prefers-color-scheme: dark)')?.matches
}

function getBgColors(isDark) {
  return isDark
    ? ['#1f6feb', '#3fb950', '#d29922', '#f85149', '#bc8cff', '#58a6ff', '#6e7681', '#da3633']
    : ['#4285f4', '#34a853', '#fbbc04', '#ea4335', '#ab47bc', '#1a73e8', '#5f6368', '#e37400']
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
    case '!': itemStyle.fontWeight = 700; itemStyle.borderColor = '#f85149'; break
    case '@': itemStyle.borderStyle = 'dotted'; itemStyle.opacity = '0.7'; break
  }

  // 布局类型特殊样式
  if (lang === 'card') {
    itemStyle.borderRadius = '8px'
    itemStyle.padding = '12px 16px'
    itemStyle.minHeight = '60px'
    itemStyle.boxShadow = '0 1px 4px rgba(0,0,0,0.1)'
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
      itemStyle.background = isDark ? 'rgba(31,111,235,0.2)' : 'rgba(66,133,244,0.15)'
      itemStyle.borderBottom = '2px solid ' + (isDark ? '#58a6ff' : '#4285f4')
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
    itemStyle.boxShadow = '0 1px 4px rgba(0,0,0,0.1)'
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
      background: isDark ? 'rgba(248,81,73,0.12)' : 'rgba(234,67,53,0.08)',
      border: '2px dashed ' + (isDark ? '#f85149' : '#ea4335'),
      borderRadius: '4px', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0', }
  })
  const bOff = margin
  items.push({
    label: 'border: ' + border + 'px solid', sub: '', icon: '',
    style: { position: 'absolute', top: bOff + 'px', left: bOff + 'px', right: bOff + 'px', bottom: bOff + 'px',
      background: isDark ? 'rgba(212,167,78,0.12)' : 'rgba(251,188,4,0.08)',
      border: border + 'px solid ' + (isDark ? '#d29922' : '#fbbc04'), borderRadius: '3px',
      display: 'flex', alignItems: 'center', justifyContent: 'center', }
  })
  const pOff = bOff + border
  items.push({
    label: 'padding: ' + padding + 'px', sub: '', icon: '',
    style: { position: 'absolute', top: pOff + 'px', left: pOff + 'px', right: pOff + 'px', bottom: pOff + 'px',
      background: isDark ? 'rgba(63,185,80,0.12)' : 'rgba(52,168,83,0.08)',
      border: '2px dashed ' + (isDark ? '#3fb950' : '#34a853'), borderRadius: '2px',
      display: 'flex', alignItems: 'center', justifyContent: 'center', }
  })
  const cOff = pOff + padding
  items.push({
    label: contentText, sub: '', icon: '',
    style: { position: 'absolute', top: cOff + 'px', left: cOff + 'px', right: cOff + 'px', bottom: cOff + 'px',
      background: isDark ? 'rgba(88,166,255,0.18)' : 'rgba(66,133,244,0.1)',
      border: '1px solid ' + (isDark ? '#58a6ff' : '#4285f4'), borderRadius: '2px',
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
          boxShadow: '0 1px 4px rgba(0,0,0,0.08)', }
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
        background: i === 0 ? (isDark ? 'rgba(31,111,235,0.2)' : 'rgba(66,133,244,0.12)') : 'transparent',
        border: '1px solid var(--border-color)', borderBottom: i === 0 ? '2px solid ' + (isDark ? '#58a6ff' : '#4285f4') : '1px solid var(--border-color)',
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
function parseTableTextToChartData(text, chartType) {
  const lines = text.trim().split('\n').filter(l => l.trim())
  if (lines.length < 2) return null
  const sep = lines[0].includes('\t') ? '\t' : ','
  const headers = lines[0].split(sep).map(h => h.trim()).filter(Boolean)
  if (headers.length < 2) return null
  const labels = []
  const datasets = []
  for (let i = 1; i < headers.length; i++) {
    datasets.push({ label: headers[i], data: [], backgroundColor: getChartColor(i - 1, 'bg'), borderColor: getChartColor(i - 1, 'border') })
  }
  for (let r = 1; r < lines.length; r++) {
    const cells = lines[r].split(sep).map(c => c.trim())
    if (cells.length < 2) continue
    labels.push(cells[0])
    for (let c = 1; c < cells.length && c - 1 < datasets.length; c++) {
      const val = parseFloat(cells[c])
      datasets[c - 1].data.push(isNaN(val) ? 0 : val)
    }
  }
  if (labels.length === 0) return null
  return { labels, datasets, chartType }
}

// ── 尝试将纯文本解析为表格数据 ──
function tryParseTableData(text) {
  if (!text) return null
  const lines = text.split('\n').filter(l => l.trim())
  if (lines.length < 2) return null
  if (lines.some(l => l.includes('|'))) {
    const mdTableLines = lines.filter(l => l.trim().startsWith('|') || l.trim().includes('|'))
    if (mdTableLines.length >= 3) {
      const dataLines = mdTableLines.filter(l => !/^[\s|:-]+$/.test(l.trim()))
      if (dataLines.length >= 2) {
        const csvLines = dataLines.map(l => l.split('|').slice(1, -1).map(c => c.trim()).join(','))
        return parseTableTextToChartData(csvLines.join('\n'), 'bar')
      }
    }
  }
  const sep = lines[0].includes('\t') ? '\t' : ','
  const firstRow = lines[0].split(sep).map(c => c.trim()).filter(Boolean)
  if (firstRow.length < 2) return null
  let numericCount = 0
  for (let r = 1; r < Math.min(lines.length, 20); r++) {
    const cells = lines[r].split(sep).map(c => c.trim()).filter(Boolean)
    if (cells.length >= 2 && !isNaN(parseFloat(cells[1]))) numericCount++
  }
  if (numericCount >= 1) return parseTableTextToChartData(text, 'bar')
  return null
}

// ── 从已渲染的 HTML 中提取表格数据 ──
function extractTablesFromHtml(html) {
  const tables = []
  const tableRe = /<table>([\s\S]*?)<\/table>/g
  let match
  while ((match = tableRe.exec(html)) !== null) {
    const tableHtml = match[0]; const inner = match[1]
    const rows = inner.match(/<tr>([\s\S]*?)<\/tr>/g)
    if (!rows || rows.length < 2) continue
    const headers = []
    const headerMatch = rows[0].match(/<th[^>]*>([\s\S]*?)<\/th>/g)
    if (headerMatch) { for (const th of headerMatch) { const t = th.replace(/<[^>]+>/g, '').trim(); if (t) headers.push(t) } }
    else {
      const tdMatch = rows[0].match(/<td[^>]*>([\s\S]*?)<\/td>/g)
      if (tdMatch) { for (const td of tdMatch) { const t = td.replace(/<[^>]+>/g, '').trim(); if (t) headers.push(t) } }
    }
    if (headers.length < 2) continue
    const labels = []
    const datasets = []
    for (let h = 1; h < headers.length; h++) datasets.push({ label: headers[h], data: [], backgroundColor: getChartColor(h - 1, 'bg'), borderColor: getChartColor(h - 1, 'border') })
    let hasData = false
    for (let r = 1; r < rows.length; r++) {
      const cells = rows[r].match(/<t[dh][^>]*>([\s\S]*?)<\/t[dh]>/g)
      if (!cells || cells.length < 2) continue
      const label = cells[0].replace(/<[^>]+>/g, '').trim()
      labels.push(label)
      for (let c = 1; c < cells.length && c - 1 < datasets.length; c++) {
        const val = parseFloat(cells[c].replace(/<[^>]+>/g, '').trim())
        if (!isNaN(val)) { datasets[c - 1].data.push(val); hasData = true } else datasets[c - 1].data.push(0)
      }
    }
    if (hasData && labels.length > 0) {
      const beforeHtml = html.slice(0, match.index)
      const titleMatch = beforeHtml.match(/<(h[2-4])[^>]*>([^<]+)<\/\1>[^<]*$/i)
      tables.push({ html: tableHtml, offset: match.index, data: { labels, datasets, chartType: 'bar' }, title: titleMatch ? titleMatch[2].trim() : '数据图表' })
    }
  }
  return tables
}

// ── 图表颜色方案 ──
const CHART_COLORS = [
  { bg: 'rgba(88, 166, 255, 0.7)', border: '#58a6ff' },
  { bg: 'rgba(63, 185, 80, 0.7)', border: '#3fb950' },
  { bg: 'rgba(210, 153, 34, 0.7)', border: '#d29922' },
  { bg: 'rgba(188, 140, 255, 0.7)', border: '#bc8cff' },
  { bg: 'rgba(248, 81, 73, 0.7)', border: '#f85149' },
  { bg: 'rgba(110, 118, 129, 0.7)', border: '#6e7681' },
]
function getChartColor(idx, type) { const c = CHART_COLORS[idx % CHART_COLORS.length]; return c[type] }

// ── 重试 Mermaid ──
function retryMermaid(i) { mermaidErrors.value = { ...mermaidErrors.value, [i]: false }; nextTick(() => initMermaid()) }

// ── 初始化 Mermaid 渲染 ──
async function initMermaid() {
  if (!renderRef.value) return
  const themeConfig = getMermaidTheme(props.theme)
  try { mermaid.initialize({ startOnLoad: false, ...themeConfig, securityLevel: 'loose', fontFamily: '"Inter", system-ui, sans-serif' }) } catch {}
  const containers = renderRef.value.querySelectorAll('.mermaid-container')
  for (const el of containers) {
    const id = el.id; if (!id) continue
    const srcEl = el.querySelector('.mermaid-src'); if (!srcEl) continue
    if (el.querySelector('svg')) continue
    const code = srcEl.textContent.trim(); if (!code) continue
    try {
      const { svg } = await mermaid.render(id + '-svg', code)
      el.innerHTML = svg
    } catch (err) {
      console.warn('[Mermaid] 渲染失败:', err.message)
      const parts = id.replace('mermaid-', '').split('-')
      mermaidErrors.value = { ...mermaidErrors.value, [parseInt(parts[1])]: true }
    }
  }
}

// ── 使用 Canvas API 渲染简易图表 ──
function renderDataCharts() {
  if (!renderRef.value) return
  const canvases = renderRef.value.querySelectorAll('.data-chart-canvas')
  for (const canvas of canvases) {
    const chartDataStr = canvas.dataset.chartData; const chartType = canvas.dataset.chartType
    if (!chartDataStr) continue
    try { drawChart(canvas, JSON.parse(chartDataStr), chartType) } catch {}
  }
}

function drawChart(canvas, data, chartType) {
  const ctx = canvas.getContext('2d'); if (!ctx) return
  const dpr = window.devicePixelRatio || 1
  const rect = canvas.getBoundingClientRect()
  canvas.width = rect.width * dpr; canvas.height = rect.height * dpr; ctx.scale(dpr, dpr)
  const w = rect.width; const h = rect.height
  const isDark = props.theme === 'dark'
  const textColor = isDark ? '#e6edf3' : '#1a1a2e'
  const gridColor = isDark ? 'rgba(48,54,61,0.5)' : 'rgba(218,220,224,0.5)'
  ctx.clearRect(0, 0, w, h)
  const pad = { top: 20, right: 20, bottom: 40, left: 50 }
  const chartW = w - pad.left - pad.right; const chartH = h - pad.top - pad.bottom
  if (chartW <= 0 || chartH <= 0) return
  const datasets = data.datasets || []; const labels = data.labels || []
  if (labels.length === 0 || datasets.length === 0) return
  let maxVal = 0; for (const ds of datasets) { for (const v of ds.data) { if (v > maxVal) maxVal = v } }
  maxVal = Math.ceil(maxVal * 1.2) || 1
  ctx.font = '10px Inter, system-ui, sans-serif'; ctx.textAlign = 'right'; ctx.textBaseline = 'middle'
  const gridLines = 5
  for (let i = 0; i <= gridLines; i++) {
    const y = pad.top + chartH - (chartH / gridLines) * i; const val = (maxVal / gridLines) * i
    ctx.strokeStyle = gridColor; ctx.lineWidth = 1; ctx.beginPath(); ctx.moveTo(pad.left, y); ctx.lineTo(w - pad.right, y); ctx.stroke()
    ctx.fillStyle = textColor; ctx.globalAlpha = 0.6; ctx.fillText(formatNum(val), pad.left - 5, y); ctx.globalAlpha = 1
  }
  if (chartType === 'pie') { renderPieChart(ctx, datasets, labels, pad, w, h, textColor); return }
  const barWidth = chartW / labels.length * 0.7; const groupGap = chartW / labels.length * 0.3; const barGap = 2
  for (let li = 0; li < labels.length; li++) {
    const xBase = pad.left + (chartW / labels.length) * li + groupGap / 2
    ctx.fillStyle = textColor; ctx.globalAlpha = 0.6; ctx.textAlign = 'center'; ctx.textBaseline = 'top'
    ctx.font = '10px Inter, system-ui, sans-serif'
    ctx.fillText(labels[li].length > 10 ? labels[li].slice(0,10)+'…' : labels[li], xBase + (chartW / labels.length - groupGap) / 2, h - pad.bottom + 8)
    ctx.globalAlpha = 1
    for (let di = 0; di < datasets.length; di++) {
      const val = datasets[di].data[li] || 0; const barH = (val / maxVal) * chartH
      const barX = xBase + (barWidth + barGap) * di; const barY = pad.top + chartH - barH
      ctx.fillStyle = datasets[di].backgroundColor || CHART_COLORS[di % CHART_COLORS.length].bg
      if (chartType === 'line' || chartType === 'radar') {
        ctx.beginPath(); ctx.arc(barX + barWidth / 2, barY, 4, 0, Math.PI * 2); ctx.fill()
        ctx.strokeStyle = datasets[di].borderColor || CHART_COLORS[di % CHART_COLORS.length].border; ctx.lineWidth = 2; ctx.stroke()
      } else {
        ctx.beginPath(); ctx.moveTo(barX + 2, pad.top + chartH); ctx.lineTo(barX + 2, barY + 2); ctx.quadraticCurveTo(barX, barY + 2, barX, barY)
        ctx.lineTo(barX + barWidth - 2, barY); ctx.quadraticCurveTo(barX + barWidth, barY, barX + barWidth, barY + 2)
        ctx.lineTo(barX + barWidth, pad.top + chartH); ctx.closePath(); ctx.fill()
      }
      if (val > 0) {
        ctx.fillStyle = textColor; ctx.globalAlpha = 0.8; ctx.textAlign = 'center'; ctx.textBaseline = 'bottom'
        ctx.font = '9px Inter, system-ui, sans-serif'; ctx.fillText(formatNum(val), barX + barWidth / 2, barY - 2); ctx.globalAlpha = 1
      }
    }
  }
  // 图例
  const legendY = h - 6; let legendX = pad.left
  ctx.textBaseline = 'bottom'; ctx.textAlign = 'left'; ctx.font = '10px Inter, system-ui, sans-serif'
  for (let di = 0; di < datasets.length; di++) {
    ctx.fillStyle = datasets[di].backgroundColor || CHART_COLORS[di % CHART_COLORS.length].bg
    ctx.fillRect(legendX, legendY - 8, 10, 10)
    ctx.fillStyle = textColor; ctx.globalAlpha = 0.7
    ctx.fillText(datasets[di].label || ('系列'+(di+1)), legendX + 14, legendY)
    ctx.globalAlpha = 1; legendX += ctx.measureText(datasets[di].label || '').width + 30
  }
  // 折线连接
  if (chartType === 'line' || chartType === 'radar') {
    for (let di = 0; di < datasets.length; di++) {
      ctx.strokeStyle = datasets[di].borderColor || CHART_COLORS[di % CHART_COLORS.length].border; ctx.lineWidth = 2; ctx.beginPath()
      for (let li = 0; li < labels.length; li++) {
        const val = datasets[di].data[li] || 0
        const xBase = pad.left + (chartW / labels.length) * li + groupGap / 2
        const barX = xBase + (barWidth + barGap) * di; const barY = pad.top + chartH - (val / maxVal) * chartH
        if (li === 0) ctx.moveTo(barX + barWidth / 2, barY); else ctx.lineTo(barX + barWidth / 2, barY)
      }
      ctx.stroke()
    }
  }
}

function renderPieChart(ctx, datasets, labels, pad, w, h, textColor) {
  const cx = w / 2; const cy = h / 2; const radius = Math.min(w, h) / 2 - 40
  if (datasets.length === 0 || !datasets[0].data) return
  const data = datasets[0].data; const total = data.reduce((a, b) => a + b, 0)
  if (total === 0) return
  let startAngle = -Math.PI / 2
  for (let i = 0; i < data.length; i++) {
    const sliceAngle = (data[i] / total) * Math.PI * 2
    ctx.beginPath(); ctx.moveTo(cx, cy); ctx.arc(cx, cy, radius, startAngle, startAngle + sliceAngle); ctx.closePath()
    ctx.fillStyle = CHART_COLORS[i % CHART_COLORS.length].bg; ctx.fill()
    ctx.strokeStyle = 'rgba(0,0,0,0.1)'; ctx.lineWidth = 1; ctx.stroke()
    if (data[i] > 0) {
      const midAngle = startAngle + sliceAngle / 2; const lr = radius * 0.65
      ctx.fillStyle = '#fff'; ctx.textAlign = 'center'; ctx.textBaseline = 'middle'
      ctx.font = 'bold 12px Inter, system-ui, sans-serif'
      ctx.fillText(Math.round((data[i]/total)*100)+'%', cx + Math.cos(midAngle)*lr, cy + Math.sin(midAngle)*lr)
    }
    startAngle += sliceAngle
  }
  const legendY = h - 6; let legendX = pad.left
  ctx.textBaseline = 'bottom'; ctx.textAlign = 'left'; ctx.font = '10px Inter, system-ui, sans-serif'
  for (let i = 0; i < Math.min(labels.length, data.length); i++) {
    ctx.fillStyle = CHART_COLORS[i % CHART_COLORS.length].bg
    ctx.fillRect(legendX, legendY - 8, 10, 10)
    ctx.fillStyle = textColor; ctx.globalAlpha = 0.7
    const label = labels[i].length > 15 ? labels[i].slice(0,15)+'…' : labels[i]
    ctx.fillText(label + ' ('+data[i]+')', legendX + 14, legendY); ctx.globalAlpha = 1
    legendX += ctx.measureText(label + ' ('+data[i]+')').width + 30
  }
}

function formatNum(n) {
  if (n >= 1000000) return (n/1000000).toFixed(1)+'M'
  if (n >= 1000) return (n/1000).toFixed(1)+'K'
  return n.toFixed(0)
}

// ── 生命周期 ──
onMounted(() => { nextTick(() => { initMermaid(); renderDataCharts() }) })
onBeforeUnmount(() => {
  // 清理 Mermaid SVG 实例（防止内存泄漏）
  if (renderRef.value) {
    renderRef.value.querySelectorAll('.mermaid-container svg').forEach(el => el.remove())
    renderRef.value.querySelectorAll('.data-chart-canvas').forEach(canvas => {
      const ctx = canvas.getContext('2d')
      if (ctx) ctx.clearRect(0, 0, canvas.width, canvas.height)
    })
  }
  mermaidErrors.value = {}
  boxZoomed.value = {}
})
watch(() => props.text, () => { mermaidErrors.value = {}; boxZoomed.value = {}; nextTick(() => { initMermaid(); renderDataCharts() }) })
watch(() => props.theme, () => {
  nextTick(() => {
    if (renderRef.value) renderRef.value.querySelectorAll('.mermaid-container svg').forEach(el => el.remove())
    initMermaid(); renderDataCharts()
  })
})
</script>

<style scoped>
/* ═══════════════ Markdown 渲染器 ═══════════════ */
.markdown-renderer { width: 100%; max-width: 100%; overflow-x: hidden; }
.md-html { line-height: 1.6; white-space: pre-wrap; word-break: break-word; overflow-wrap: break-word; }

/* ── Markdown 内容样式 ── */
.md-html :deep(h1), .md-html :deep(h2), .md-html :deep(h3), .md-html :deep(h4) { margin: 8px 0 4px; font-weight: 600; }
.md-html :deep(h1) { font-size: 16px; }
.md-html :deep(h2) { font-size: 15px; }
.md-html :deep(h3) { font-size: 14px; }
.md-html :deep(p) { margin: 4px 0; }
.md-html :deep(ul), .md-html :deep(ol) { margin: 4px 0; padding-left: 20px; }
.md-html :deep(li) { margin: 2px 0; }
.md-html :deep(code) { background: rgba(0,0,0,0.15); padding: 1px 4px; border-radius: 3px; font-family: var(--font-code); font-size: 12px; }
.md-html :deep(pre) { background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 6px; padding: 8px 10px; margin: 6px 0; overflow-x: auto; font-size: 12px; line-height: 1.4; white-space: pre-wrap; word-break: break-word; max-width: 100%; }
.md-html :deep(pre code) { background: none; padding: 0; border-radius: 0; white-space: pre-wrap; word-break: break-word; }
.md-html :deep(blockquote) { border-left: 3px solid var(--accent); padding-left: 8px; margin: 6px 0; color: var(--text-secondary); font-style: italic; }
.md-html :deep(a) { color: var(--accent-light); text-decoration: none; }
.md-html :deep(a:hover) { text-decoration: underline; }
.md-html :deep(table) { border-collapse: collapse; margin: 6px 0; font-size: 12px; width: 100%; }
.md-html :deep(th), .md-html :deep(td) { border: 1px solid var(--border-color); padding: 4px 8px; text-align: left; }
.md-html :deep(th) { background: var(--bg-tertiary); font-weight: 600; }
.md-html :deep(hr) { border: none; border-top: 1px solid var(--border-color); margin: 8px 0; }
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
.chart-error-hint { margin: 6px 0 0; font-size: 11px; color: #f85149; opacity: 0.7; }

/* ── 数据图表 Canvas 容器 ── */
.chart-canvas-wrap { padding: 12px; min-height: 200px; position: relative; }
.data-chart-canvas { width: 100%; height: 200px; display: block; }

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
  box-shadow: 0 2px 16px rgba(0,0,0,0.25);
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
.layout-style-modal .layout-preview { background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 12px; overflow: hidden; box-shadow: 0 8px 32px rgba(0,0,0,0.2); }
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
