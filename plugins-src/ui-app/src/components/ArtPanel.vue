<template>
  <PanelShell
    v-model:file="projectPath"
    title="画板"
    icon="grid"
    placeholder="art.project.json"
    :badge="verifyBadge"
    :badge-tone="verifyTone"
    :metrics="metrics"
    :loading="loading"
    :error="proj ? error : ''"
    :notice="info"
    :empty="!proj"
    reload-title="重新载入工程、SVG 与校验报告"
    footnote="只读面板：改动画板请让 Agent 执行 art_edit（命令式 op）—— 面板不写工作区，避免与工具形成两条写路径。"
    :source="projectPath"
    @reload="load"
  >
    <template #head-actions>
      <button class="pn-btn" :disabled="!svgText || !imgReady" title="用浏览器 canvas 绘制已渲染的 SVG 并下载 PNG" @click="exportPng">
        导出 PNG
      </button>
    </template>

    <!-- 空态：说清为什么空 + 怎么做 -->
    <template #empty>
      <div class="pn-empty-title">还没有画板工程</div>
      <div class="pn-tip">
        面板读工作区根目录的 <code>art.project.json</code>（与 Agent 共用同一份真相源），当前没有这个文件，所以没有可看的内容。
      </div>
      <div v-if="error" class="pn-tip pn-mono">{{ error }}</div>
      <div class="pn-card">
        <div class="pn-card-title">三步开始</div>
        <div class="pn-empty-step"><i>1</i><span>让 Agent 执行 <code>art_project</code> 创建工程（画布尺寸 + 图层）。</span></div>
        <div class="pn-empty-step"><i>2</i><span>用 <code>art_edit</code> 添加图元（rect / circle / text / line …）。</span></div>
        <div class="pn-empty-step"><i>3</i><span><code>art_export</code> 导出 SVG、<code>art_verify</code> 生成校验报告。</span></div>
      </div>
      <div class="pn-tip">做完第 1 步后点右上角刷新，即可看到画布。</div>
    </template>

    <template #segments>
      <button
        v-for="t in tabs"
        :key="t.id"
        class="pn-tab"
        :class="{ 'pn-tab-on': tab === t.id }"
        :title="t.title"
        @click="tab = t.id"
      >{{ t.name }}<span v-if="t.count" class="pn-dim"> {{ t.count }}</span></button>
    </template>

    <!-- 左栏：图元 / 图层 / 颜色 / 校验 -->
    <template #side>
      <div v-if="tab === 'shapes'" class="pn-list">
        <button
          v-for="s in shapeRows"
          :key="s.id"
          class="pn-item"
          :class="{ 'pn-item-on': s.id === selectedId }"
          @click="select(s.id)"
        >
          <span class="pn-swatch" :style="{ background: s.fillCss }"></span>
          <span class="pn-item-k">{{ s.id }}</span>
          <span class="pn-dim pn-mini">{{ s.type }} · {{ s.layer }}</span>
          <span class="pn-item-v">{{ s.size }}</span>
        </button>
        <div v-if="!shapeRows.length" class="pn-tip">还没有图元。</div>
        <div v-else class="pn-tip art-side-hint">点行可在右侧画布高亮定位。</div>
      </div>

      <div v-else-if="tab === 'layers'" class="pn-list">
        <div v-for="l in layerRows" :key="l.id" class="pn-kv">
          <span class="pn-mono">{{ l.id }}</span>
          <b>{{ l.name }} <span class="pn-dim">{{ l.count }} 图元{{ l.visible ? '' : ' · 隐藏' }}</span></b>
        </div>
        <div class="pn-tip art-side-hint">顺序 = 从下到上（后画的在上）。</div>
      </div>

      <div v-else-if="tab === 'colors'" class="pn-list">
        <div v-for="c in colorList" :key="c.raw" class="pn-kv">
          <span class="pn-swatch" :style="{ background: c.raw }"></span>
          <b class="pn-mono">{{ c.raw }}</b>
          <span class="pn-dim">×{{ c.count }}</span>
          <span class="pn-spacer"></span>
          <b v-if="c.ratio !== null" :class="c.pass ? 'pn-pass' : 'pn-warn'">
            {{ c.ratio.toFixed(2) }}:1{{ c.pass ? ' AA' : ' 低于 4.5' }}
          </b>
          <span v-else class="pn-dim">—</span>
        </div>
        <div class="pn-tip art-side-hint">按「色 vs 画板底色（{{ backgroundText }}）」估算；文字实际底色由校验 V6 精确判定。</div>
      </div>

      <div v-else class="pn-list">
        <template v-if="verify">
          <div v-for="c in verify.checks || []" :key="c.id" class="pn-check">
            <span class="pn-dot" :class="{ 'pn-dot-ok': c.pass }"></span>
            <b>{{ c.id }}</b>
            <span class="pn-check-name">{{ c.name }}</span>
            <span class="pn-check-metric" :title="c.metric">{{ c.metric }}</span>
          </div>
          <div class="pn-tip art-side-hint">报告时间：{{ verify.ts }}</div>
        </template>
        <div v-else class="pn-tip">
          还没有校验报告。让 Agent 执行 <code>art_verify</code> 生成旁挂的 art.verify.json。
        </div>
      </div>
    </template>

    <!-- 主区：画布预览 + 选中详情 -->
    <template #main>
      <div class="pn-scroll art-main">
        <div class="pn-row-gap">
          <span class="pn-strong">{{ artTitle }}</span>
          <span class="pn-dim pn-mini pn-mono">{{ canvasW }}×{{ canvasH }}</span>
          <span class="pn-spacer"></span>
          <span v-if="selected" class="pn-dim pn-mini">已选中 <span class="pn-mono">{{ selected.id }}</span></span>
        </div>

        <div ref="wrapEl" class="pn-preview">
          <img v-if="svgText" ref="imgEl" class="art-canvas" :src="svgUrl" alt="画板预览" @load="onImgLoad" />
          <div v-else class="pn-tip">
            未找到 SVG 产物。让 Agent 执行 <code>art_export</code> 生成（默认 art.svg）。
          </div>
          <div v-if="hlBox" class="art-hl" :style="hlBox"></div>
        </div>
        <div class="pn-tip">
          覆盖范围 <span class="pn-mono">{{ coverInfo.range }}</span>（占画布 {{ coverInfo.pct }}%）；浏览器用 img + data URI 渲染 SVG（脚本不执行）。
        </div>

        <div v-if="selected" class="pn-card">
          <div class="pn-card-title">
            选中 <span class="pn-mono">{{ selected.id }}</span>
            <span class="pn-dim">{{ selected.type }}</span>
            <span class="pn-spacer"></span>
            <button class="pn-btn" @click="selectedId = ''">取消</button>
          </div>
          <div v-for="r in selectedRows" :key="r.k" class="pn-kv">
            <span>{{ r.k }}</span><b class="pn-mono">{{ r.v }}</b>
          </div>
          <div class="pn-tip">面板只读 —— 复制下面命令交给 Agent 执行即可改这份真相源：</div>
          <pre class="pn-code">{{ selectedOp }}</pre>
          <div>
            <button class="pn-btn" @click="copyOp">{{ copied ? '已复制 ✓' : '复制 op' }}</button>
          </div>
        </div>
        <div v-else class="pn-tip">点左栏「图元」里的任意一行：画布上会高亮定位，这里会显示它的完整属性与可直接复制的 op 命令。</div>
      </div>
    </template>
  </PanelShell>
</template>

<script setup>
// ArtPanel — 矢量画板只读视图（与 Agent 共用同一份真相源：art.project.json + art.svg + 旁挂 art.verify.json）。
// 设计取舍（与 tool-music / tool-voice 一致）：面板不写工作区——图元编辑一律经 Agent 的 art_edit（命令式 op），
// 避免「工具写工程 / 面板写工程」两条写路径导致真相源分叉；需要改动时面板给出可直接复制的 op 片段。
// 唯一产出动作是浏览器侧 PNG 导出（canvas 绘制已渲染的 SVG → 本地下载），同样不写回工作区。
// 布局（2026-09-18 改版）：统一外壳 PanelShell —— 顶栏 / 指标条 / 左栏分段 + 主区预览 / 底栏，去掉「一路往下铺」的长滚动。
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import api from '../api.js'
import PanelShell from './PanelShell.vue'
import { state } from '../ui-state.js'

const projectPath = ref('art.project.json')
const verifyPath = ref('art.verify.json')
const loading = ref(false)
const error = ref('')
const info = ref('')
const proj = ref(null)
const verify = ref(null)
const selectedId = ref('')
const copied = ref(false)
const svgText = ref('')
const imgReady = ref(false)
const dispWidth = ref(0)
const tab = ref('shapes')

const imgEl = ref(null)
const wrapEl = ref(null)

const layers = computed(() => (proj.value && proj.value.layers) || [])
const shapes = computed(() => (proj.value && proj.value.shapes) || [])
const canvasW = computed(() => ((proj.value && proj.value.canvas) || {}).width || 0)
const canvasH = computed(() => ((proj.value && proj.value.canvas) || {}).height || 0)
const artTitle = computed(() => ((proj.value && proj.value.meta && proj.value.meta.title) || '未命名画板'))
const backgroundText = computed(() => {
  const bg = (proj.value && proj.value.canvas && proj.value.canvas.background) || 'none'
  return bg
})
const verifyPassed = computed(() => ((verify.value && verify.value.checks) || []).filter((c) => c.pass).length)
const verifyTotal = computed(() => ((verify.value && verify.value.checks) || []).length)
const verifyBadge = computed(() => (verify.value ? '校验 ' + verifyPassed.value + '/' + verifyTotal.value : '未校验'))
const verifyTone = computed(() => (!verify.value ? 'muted' : (verifyPassed.value === verifyTotal.value ? 'ok' : 'bad')))

// SVG 预览：用 data URI 交给 <img> 渲染 —— SVG 内的脚本不会执行（面板不做信任假设）
const svgUrl = computed(() => (svgText.value ? 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svgText.value) : ''))

function absPath(rel) {
  const root = String((state && state.workspaceRoot) || '').replace(/[\\/]+$/, '')
  if (!root) return rel
  return root + '/' + String(rel).replace(/^[\\/]+/, '')
}

async function readText(rel) {
  // ★ apiURL 会自动加 /api 前缀（api.js: BASE='/api'）——再写 /api 会拼成 /api/api/fs/read → 404
  const r = await api.apiGet('/fs/read', { path: absPath(rel) })
  return r.content
}

async function load() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  info.value = ''
  try {
    proj.value = JSON.parse(await readText(projectPath.value))
    verify.value = null
    try { verify.value = JSON.parse(await readText(verifyPath.value)) } catch (_) { /* 报告可选 */ }
    // SVG 产物：优先取工程登记的 artifacts.svg，其次同名 art.svg
    const svgRel = (proj.value.artifacts && proj.value.artifacts.svg) || String(projectPath.value).replace(/\.json$/i, '.svg')
    svgText.value = ''
    imgReady.value = false
    try { svgText.value = await readText(svgRel) } catch (_) { /* 未导出时保持空 */ }
    selectedId.value = ''
  } catch (e) {
    proj.value = null
    svgText.value = ''
    error.value = '载入失败：' + ((e && e.message) || e)
  } finally {
    loading.value = false
  }
}

// ── 几何（与 tool-art 同规则的精简版：支持 transform 矩阵的平移/旋转/缩放） ──
function numOf(v, d) { const n = typeof v === 'number' ? v : parseFloat(v); return isFinite(n) ? n : d }

function localBox(s) {
  const t = s.type
  if (t === 'rect') return { x: numOf(s.x, 0), y: numOf(s.y, 0), w: numOf(s.w, 0), h: numOf(s.h, 0) }
  if (t === 'circle') { const r = numOf(s.r, 0); return { x: numOf(s.cx, 0) - r, y: numOf(s.cy, 0) - r, w: 2 * r, h: 2 * r } }
  if (t === 'ellipse') {
    const rx = numOf(s.rx, 0); const ry = numOf(s.ry, 0)
    return { x: numOf(s.cx, 0) - rx, y: numOf(s.cy, 0) - ry, w: 2 * rx, h: 2 * ry }
  }
  if (t === 'line') {
    const x1 = numOf(s.x1, 0); const y1 = numOf(s.y1, 0); const x2 = numOf(s.x2, 0); const y2 = numOf(s.y2, 0)
    return { x: Math.min(x1, x2), y: Math.min(y1, y2), w: Math.abs(x2 - x1), h: Math.abs(y2 - y1) }
  }
  if (t === 'path') {
    const vals = String(s.d || '').match(/-?\d+(\.\d+)?/g) || []
    let mnx = Infinity; let mny = Infinity; let mxx = -Infinity; let mxy = -Infinity
    for (let i = 0; i + 1 < vals.length; i += 2) {
      mnx = Math.min(mnx, vals[i]); mxx = Math.max(mxx, vals[i])
      mny = Math.min(mny, vals[i + 1]); mxy = Math.max(mxy, vals[i + 1])
    }
    if (!isFinite(mnx)) return { x: 0, y: 0, w: 0, h: 0 }
    return { x: mnx, y: mny, w: mxx - mnx, h: mxy - mny }
  }
  if (t === 'text') {
    const fs = numOf(s.fontSize, 16)
    const text = String(s.text || '')
    let units = 0
    for (let i = 0; i < text.length; i++) units += text.charCodeAt(i) > 0x2e80 ? 1 : 0.55
    const bold = String(s.fontWeight || '400') === 'bold' || numOf(s.fontWeight, 400) >= 600
    const w = units * fs * (bold ? 1.05 : 1)
    const anchor = s.anchor || 'start'
    const x0 = anchor === 'middle' ? numOf(s.x, 0) - w / 2 : (anchor === 'end' ? numOf(s.x, 0) - w : numOf(s.x, 0))
    return { x: x0, y: numOf(s.y, 0) - fs * 0.8, w, h: fs }
  }
  return { x: 0, y: 0, w: 0, h: 0 }
}

function applyM(m, x, y) {
  return { x: m[0] * x + m[2] * y + m[4], y: m[1] * x + m[3] * y + m[5] }
}

function shapeBox(s) {
  const b = localBox(s)
  const m = s.transform
  if (!m || !m.length) return b
  const isRd = s.type === 'circle' || s.type === 'ellipse'
  const pts = isRd
    ? Array.from({ length: 16 }, (_, i) => {
      const a = Math.PI * 2 * i / 16
      const rx = s.type === 'circle' ? numOf(s.r, 0) : numOf(s.rx, 0)
      const ry = s.type === 'circle' ? numOf(s.r, 0) : numOf(s.ry, 0)
      return [numOf(s.cx, 0) + rx * Math.cos(a), numOf(s.cy, 0) + ry * Math.sin(a)]
    })
    : [[b.x, b.y], [b.x + b.w, b.y], [b.x + b.w, b.y + b.h], [b.x, b.y + b.h]]
  let mnx = Infinity; let mny = Infinity; let mxx = -Infinity; let mxy = -Infinity
  pts.forEach((p) => {
    const q = applyM(m, p[0], p[1])
    mnx = Math.min(mnx, q.x); mxx = Math.max(mxx, q.x)
    mny = Math.min(mny, q.y); mxy = Math.max(mxy, q.y)
  })
  return { x: mnx, y: mny, w: mxx - mnx, h: mxy - mny }
}

function fmt(n) {
  const r = Math.round(numOf(n, 0) * 100) / 100
  return String(r)
}

const shapeRows = computed(() => shapes.value.map((s) => {
  const b = shapeBox(s)
  const isLine = s.type === 'line'
  return {
    id: s.id,
    type: s.type,
    layer: s.layer,
    pos: fmt(b.x) + ',' + fmt(b.y),
    size: fmt(b.w) + '×' + fmt(b.h),
    fill: isLine ? '（线）' : String(s.fill === undefined ? 'none' : s.fill),
    fillCss: isLine || !s.fill || s.fill === 'none' ? 'transparent' : s.fill,
  }
}))

const layerRows = computed(() => layers.value.map((l, i) => {
  let count = 0
  shapes.value.forEach((s) => {
    const known = layers.value.some((x) => x.id === s.layer)
    const eff = known ? s.layer : (layers.value[0] || {}).id
    if (eff === l.id) count++
  })
  return { id: l.id, name: l.name || '', index: i, count, visible: l.visible !== false }
}))

// 覆盖范围：所有图元包围盒的并集（相对画布占比）
const coverInfo = computed(() => {
  if (!shapes.value.length) return { range: '（空）', pct: '0.0' }
  let union = null
  shapes.value.forEach((s) => {
    const b = shapeBox(s)
    union = union
      ? {
        x: Math.min(union.x, b.x), y: Math.min(union.y, b.y),
        w: Math.max(union.x + union.w, b.x + b.w) - Math.min(union.x, b.x),
        h: Math.max(union.y + union.h, b.y + b.h) - Math.min(union.y, b.y),
      }
      : { x: b.x, y: b.y, w: b.w, h: b.h }
  })
  const pct = canvasW.value && canvasH.value
    ? Math.min(100, (union.w * union.h) / (canvasW.value * canvasH.value) * 100)
    : 0
  return {
    range: 'x ' + fmt(union.x) + '..' + fmt(union.x + union.w) + '，y ' + fmt(union.y) + '..' + fmt(union.y + union.h),
    pct: pct.toFixed(1),
  }
})

// ── 指标条（关键数字提到顶栏下方，不必滚动去找） ──
const metrics = computed(() => [
  { k: '画布', v: canvasW.value + '×' + canvasH.value, mono: true },
  { k: '底色', v: backgroundText.value, mono: true },
  { k: '图层', v: String(layers.value.length) },
  { k: '图元', v: String(shapes.value.length) },
  { k: '用色', v: String(colorList.value.length) },
  { k: '覆盖', v: coverInfo.value.pct + '%' },
])

const tabs = computed(() => [
  { id: 'shapes', name: '图元', count: shapes.value.length, title: '图元清单（点行高亮定位）' },
  { id: 'layers', name: '图层', count: layers.value.length, title: '图层清单（从下到上）' },
  { id: 'colors', name: '颜色', count: colorList.value.length, title: '用色与对比度' },
  { id: 'checks', name: '校验', count: verify.value ? verifyPassed.value + '/' + verifyTotal.value : 0, title: '工程校验报告' },
])

// ── 颜色 + 对比度（面板侧估算：色 vs 画板底色） ──
const NAMED = {
  black: [0, 0, 0], white: [255, 255, 255], gray: [128, 128, 128], grey: [128, 128, 128],
  red: [255, 0, 0], green: [0, 128, 0], blue: [0, 0, 255], orange: [255, 165, 0], yellow: [255, 255, 0],
}
function parseColor(v) {
  const s = String(v || '').trim().toLowerCase()
  if (!s || s === 'none' || s === 'transparent') return null
  if (s[0] === '#') {
    const hex = s.slice(1)
    if (hex.length === 3) return [parseInt(hex[0] + hex[0], 16), parseInt(hex[1] + hex[1], 16), parseInt(hex[2] + hex[2], 16)]
    if (hex.length >= 6) return [parseInt(hex.slice(0, 2), 16), parseInt(hex.slice(2, 4), 16), parseInt(hex.slice(4, 6), 16)]
    return null
  }
  const m = /^rgba?\(([^)]+)\)$/.exec(s)
  if (m) {
    const p = m[1].split(/[\s,\/]+/).filter((x) => x !== '')
    if (p.length < 3) return null
    return [parseFloat(p[0]), parseFloat(p[1]), parseFloat(p[2])]
  }
  return NAMED[s] || null
}
function lum(c) {
  const f = (v) => { const x = v / 255; return x <= 0.03928 ? x / 12.92 : Math.pow((x + 0.055) / 1.055, 2.4) }
  return 0.2126 * f(c[0]) + 0.7152 * f(c[1]) + 0.0722 * f(c[2])
}
function contrast(a, b) {
  const la = lum(a); const lb = lum(b)
  return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05)
}

const colorList = computed(() => {
  const bgRaw = (proj.value && proj.value.canvas && proj.value.canvas.background) || 'none'
  const bg = parseColor(bgRaw) || [255, 255, 255]
  const acc = {}
  shapes.value.forEach((s) => {
    if (s.fill && s.fill !== 'none' && s.type !== 'line') acc[s.fill] = (acc[s.fill] || 0) + 1
    if (s.stroke && s.stroke !== 'none') acc[s.stroke] = (acc[s.stroke] || 0) + 1
  })
  return Object.keys(acc).map((raw) => {
    const c = parseColor(raw)
    const ratio = c ? contrast(c, bg) : null
    return { raw, count: acc[raw], ratio, pass: ratio !== null && ratio >= 4.5 }
  }).sort((a, b) => b.count - a.count)
})

// ── 选中详情与 op 片段 ──
const selected = computed(() => shapes.value.find((s) => s.id === selectedId.value) || null)

const selectedRows = computed(() => {
  const s = selected.value
  if (!s) return []
  const b = shapeBox(s)
  const rows = [
    { k: '图层 / z', v: String(s.layer) + ' / ' + String(s.z) },
    { k: '包围盒', v: fmt(b.x) + ',' + fmt(b.y) + ' ' + fmt(b.w) + '×' + fmt(b.h) },
  ]
  const keys = ['x', 'y', 'w', 'h', 'rx', 'cx', 'cy', 'r', 'rx', 'ry', 'x1', 'y1', 'x2', 'y2', 'd', 'text', 'fontSize', 'fontWeight', 'anchor']
  const seen = {}
  keys.forEach((k) => {
    if (seen[k] || s[k] === undefined) return
    seen[k] = true
    rows.push({ k, v: String(s[k]) })
  })
  if (s.points) rows.push({ k: 'points', v: s.points.length + ' 点' })
  if (s.type !== 'line') rows.push({ k: 'fill', v: String(s.fill === undefined ? 'none' : s.fill) })
  rows.push({ k: 'stroke', v: String(s.stroke === undefined ? 'none' : s.stroke) + (Number(s.strokeWidth) > 0 ? ' / ' + s.strokeWidth : '') })
  if (Number(s.opacity) < 1) rows.push({ k: 'opacity', v: String(s.opacity) })
  if (s.transform) rows.push({ k: 'transform', v: s.transform.map(fmt).join(' ') })
  return rows
})

const selectedOp = computed(() => {
  const s = selected.value
  if (!s) return ''
  const bg = parseColor((proj.value.canvas && proj.value.canvas.background) || 'none')
  const fill = s.fill && s.fill !== 'none' ? s.fill : '#2563EB'
  return [
    JSON.stringify({ op: 'shape.move', id: s.id, dx: 0, dy: 0 }),
    JSON.stringify({ op: 'shape.set', id: s.id, set: { fill: fill } }),
    JSON.stringify({ op: 'shape.set', id: s.id, set: { stroke: '#111827', strokeWidth: 2 } }),
    JSON.stringify({ op: 'shape.z', id: s.id, mode: 'front' }),
    JSON.stringify({ op: 'shape.remove', id: s.id }),
    '',
    '// ' + s.id + ' 当前 fill ' + String(s.fill === undefined ? 'none' : s.fill) +
      '（与画板底色对比度 ' + (() => {
        const c = parseColor(s.fill)
        return c && bg ? contrast(c, bg).toFixed(2) + ':1' : 'n/a'
      })() + '，AA 需 ≥ 4.5:1）',
  ].join('\n')
})

async function copyOp() {
  try {
    await navigator.clipboard.writeText(selectedOp.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1600)
  } catch (e) {
    info.value = '复制失败（浏览器未授权剪贴板）：可手动选中上面的命令文本。'
  }
}

function select(id) {
  selectedId.value = selectedId.value === id ? '' : id
  if (selectedId.value) updateDispWidth()
}

// ── 预览高亮框 ──
const hlBox = computed(() => {
  const s = selected.value
  if (!s || !dispWidth.value) return null
  const b = shapeBox(s)
  const scale = dispWidth.value / (canvasW.value || 1)
  return {
    left: (b.x * scale) + 'px',
    top: (b.y * scale) + 'px',
    width: Math.max(2, b.w * scale) + 'px',
    height: Math.max(2, b.h * scale) + 'px',
  }
})

function updateDispWidth() {
  const img = imgEl.value
  if (img) dispWidth.value = img.clientWidth || 0
}

function onImgLoad() {
  imgReady.value = true
  updateDispWidth()
}

// ── 导出 PNG（浏览器 canvas 绘制已渲染的 SVG → 本地下载；不写工作区） ──
function exportPng() {
  const img = imgEl.value
  if (!img || !imgReady.value) { error.value = '预览尚未就绪，稍后再试'; return }
  try {
    const cv = document.createElement('canvas')
    cv.width = canvasW.value
    cv.height = canvasH.value
    const g = cv.getContext('2d')
    const bg = backgroundText.value
    if (bg && bg !== 'none' && parseColor(bg)) { g.fillStyle = bg; g.fillRect(0, 0, cv.width, cv.height) }
    g.drawImage(img, 0, 0, cv.width, cv.height)
    const url = cv.toDataURL('image/png')
    const a = document.createElement('a')
    a.href = url
    a.download = String(projectPath.value).replace(/\.json$/i, '') + '.png'
    document.body.appendChild(a)
    a.click()
    a.remove()
    info.value = '已导出 PNG（' + cv.width + '×' + cv.height + '）：' + a.download + '（浏览器下载目录）'
  } catch (e) {
    error.value = 'PNG 导出失败：' + ((e && e.message) || e)
  }
}

function onResize() { updateDispWidth() }

onMounted(() => {
  window.addEventListener('resize', onResize)
  load()
})
onBeforeUnmount(() => { window.removeEventListener('resize', onResize) })
watch(selectedId, () => { updateDispWidth() })
</script>

<style scoped>
/* 局部样式（其余一律复用 PanelShell 的共享类；配色取设计系统变量） */
.art-main { display: flex; flex-direction: column; gap: 10px; padding: 10px; }
.art-canvas { display: block; width: 100%; height: auto; }
/* 选中高亮框：覆盖在预览之上，不拦截鼠标事件 */
.art-hl { position: absolute; border: 1px dashed var(--accent); pointer-events: none; box-sizing: border-box; }
.art-side-hint { margin-top: 6px; }
.pn-pass { color: var(--text-secondary); }
</style>
