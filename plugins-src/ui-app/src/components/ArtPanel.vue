<template>
  <div class="art-panel">
    <div class="ap-bar">
      <span class="ap-title">画板</span>
      <input v-model="projectPath" class="ap-input" placeholder="art.project.json" @keyup.enter="load" />
      <button class="ap-icon-btn" :disabled="loading" title="重新载入工程、SVG 与校验报告" @click="load">
        <SvgIcon name="refresh" :size="12" :class="{ spinning: loading }" />
      </button>
    </div>

    <div v-if="error" class="ap-msg ap-err">{{ error }}</div>
    <div v-else-if="info" class="ap-msg ap-info">{{ info }}</div>
    <div v-if="!proj && !error" class="ap-msg">
      未找到画板工程。先让 Agent 执行 <code>art_project</code> 创建工程（生成 art.project.json）。
    </div>

    <template v-if="proj">
      <!-- 画板信息 -->
      <section class="ap-sec">
        <div class="ap-sec-title">{{ title }}</div>
        <div class="ap-kv"><span>画布 / 背景</span><b>{{ canvasW }}×{{ canvasH }} · {{ backgroundText }}</b></div>
        <div class="ap-kv"><span>图层 / 图元 / 用色</span><b>{{ layers.length }} · {{ shapes.length }} · {{ colorList.length }}</b></div>
        <div class="ap-kv"><span>覆盖范围</span><b class="ap-mono">{{ coverText }}</b></div>
        <div class="ap-kv"><span>文件</span><b class="ap-mono">{{ projectPath }}</b></div>
      </section>

      <!-- 预览 + 导出 -->
      <section class="ap-sec">
        <div class="ap-row">
          <span class="ap-sec-title">画布预览</span>
          <span class="ap-spacer"></span>
          <button class="ap-btn" :disabled="!svgText || !imgReady" title="用浏览器 canvas 绘制 SVG 并下载 PNG" @click="exportPng">
            导出 PNG
          </button>
        </div>
        <div ref="wrapEl" class="ap-canvas-wrap">
          <img v-if="svgText" ref="imgEl" class="ap-canvas" :src="svgUrl" alt="画板预览" @load="onImgLoad" />
          <div v-else class="ap-dim ap-msg">未找到 SVG 产物。让 Agent 执行 <code>art_export</code> 生成（默认 art.svg）。</div>
          <div v-if="hlBox" class="ap-hl" :style="hlBox"></div>
        </div>
        <div class="ap-dim">
          浏览器渲染 SVG（img + data URI，脚本不执行）；点下方图元行可高亮定位。导出的 PNG 尺寸 =
          画布尺寸，由本地面板直接下载，不写回工作区。
        </div>
      </section>

      <!-- 图元表 -->
      <section class="ap-sec">
        <div class="ap-sec-title">图元 <span class="ap-dim">（{{ shapes.length }}；点行选中）</span></div>
        <table class="ap-table">
          <thead>
            <tr><th>id</th><th>类型</th><th>位置</th><th>尺寸</th><th>填充</th><th>层</th></tr>
          </thead>
          <tbody>
            <tr
              v-for="s in shapeRows"
              :key="s.id"
              :class="{ 'ap-row-sel': s.id === selectedId }"
              @click="select(s.id)"
            >
              <td class="ap-mono">{{ s.id }}</td>
              <td>{{ s.type }}</td>
              <td class="ap-mono">{{ s.pos }}</td>
              <td class="ap-mono">{{ s.size }}</td>
              <td>
                <span class="ap-swatch" :style="{ background: s.fillCss }"></span>
                <span class="ap-mono ap-fill">{{ s.fill }}</span>
              </td>
              <td class="ap-mono">{{ s.layer }}</td>
            </tr>
          </tbody>
        </table>
      </section>

      <!-- 选中详情 + op 片段 -->
      <section v-if="selected" class="ap-sec">
        <div class="ap-sec-title">
          选中：<span class="ap-mono">{{ selected.id }}</span> <span class="ap-dim">{{ selected.type }}</span>
          <span class="ap-spacer"></span>
          <button class="ap-btn" @click="selectedId = ''">取消</button>
        </div>
        <div v-for="r in selectedRows" :key="r.k" class="ap-kv ap-kv-left">
          <span>{{ r.k }}</span><b class="ap-mono">{{ r.v }}</b>
        </div>
        <div class="ap-dim">面板只读（不产生第二条写路径）——复制下面命令交给 Agent 执行即可改这份真相源：</div>
        <pre class="ap-code">{{ selectedOp }}</pre>
        <button class="ap-btn" @click="copyOp">{{ copied ? '已复制 ✓' : '复制 op' }}</button>
      </section>

      <!-- 图层 -->
      <section class="ap-sec">
        <div class="ap-sec-title">图层 <span class="ap-dim">（{{ layers.length }}；顺序 = 从下到上）</span></div>
        <div v-for="l in layerRows" :key="l.id" class="ap-kv">
          <span class="ap-mono">{{ l.id }}</span>
          <b>{{ l.name }} <span class="ap-dim">{{ l.count }} 图元{{ l.visible ? '' : ' · 隐藏' }}</span></b>
        </div>
      </section>

      <!-- 颜色 -->
      <section class="ap-sec">
        <div class="ap-sec-title">颜色 <span class="ap-dim">（{{ colorList.length }} 种，对比度按画板背景 {{ backgroundText }} 计算）</span></div>
        <div v-for="c in colorList" :key="c.raw" class="ap-kv">
          <span>
            <span class="ap-swatch" :style="{ background: c.raw }"></span>
            <b class="ap-mono">{{ c.raw }}</b>
            <span class="ap-dim"> ×{{ c.count }}</span>
          </span>
          <b v-if="c.ratio !== null" :class="c.pass ? 'ap-pass' : 'ap-fail'">
            {{ c.ratio.toFixed(2) }}:1 {{ c.pass ? 'AA' : '低于 4.5' }}
          </b>
          <b v-else class="ap-dim">描边/无填充</b>
        </div>
        <div class="ap-dim">此处按“色 vs 画板底色”估算；文字实际底色（可能是下层图形）由校验 V6 精确判定。</div>
      </section>

      <!-- 校验报告（旁挂 art.verify.json，由 art_verify 写入） -->
      <section class="ap-sec">
        <div class="ap-sec-title">
          工程校验
          <span v-if="verify" :class="['ap-badge', verify.pass ? 'ap-ok' : 'ap-bad']">
            {{ verifyPassed }}/{{ verifyTotal }}
          </span>
          <span v-else class="ap-dim">（未校验）</span>
        </div>
        <div v-if="!verify" class="ap-dim">
          让 Agent 执行 <code>art_verify</code> 生成校验报告（art.verify.json）。
        </div>
        <div v-else>
          <div v-for="c in verify.checks || []" :key="c.id" class="ap-check">
            <span :class="['ap-dot', c.pass ? 'ap-ok' : 'ap-bad']"></span>
            <b>{{ c.id }}</b>
            <span class="ap-check-name">{{ c.name }}</span>
            <span class="ap-dim ap-check-metric" :title="c.metric">{{ c.metric }}</span>
          </div>
          <div class="ap-dim">报告时间：{{ verify.ts }}</div>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup>
// ArtPanel — 矢量画板只读视图（与 Agent 共用同一份真相源：art.project.json + art.svg + 旁挂 art.verify.json）。
// 设计取舍（与 tool-music / tool-voice 一致）：面板不写工作区——图元编辑一律经 Agent 的 art_edit（命令式 op），
// 避免「工具写工程 / 面板写工程」两条写路径导致真相源分叉；需要改动时面板给出可直接复制的 op 片段。
// 唯一产出动作是浏览器侧 PNG 导出（canvas 绘制已渲染的 SVG → 本地下载），同样不写回工作区。
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'
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

const imgEl = ref(null)
const wrapEl = ref(null)

const layers = computed(() => (proj.value && proj.value.layers) || [])
const shapes = computed(() => (proj.value && proj.value.shapes) || [])
const canvasW = computed(() => ((proj.value && proj.value.canvas) || {}).width || 0)
const canvasH = computed(() => ((proj.value && proj.value.canvas) || {}).height || 0)
const title = computed(() => ((proj.value && proj.value.meta && proj.value.meta.title) || '未命名画板'))
const backgroundText = computed(() => {
  const bg = (proj.value && proj.value.canvas && proj.value.canvas.background) || 'none'
  return bg
})
const verifiedAt = computed(() => (verify.value && verify.value.ts) || '')
const verifyPassed = computed(() => ((verify.value && verify.value.checks) || []).filter((c) => c.pass).length)
const verifyTotal = computed(() => ((verify.value && verify.value.checks) || []).length)

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
  if (t === 'polyline' || t === 'polygon') {
    const pts = s.points || []
    if (!pts.length) return { x: 0, y: 0, w: 0, h: 0 }
    const xs = pts.map((p) => numOf(p[0], 0)); const ys = pts.map((p) => numOf(p[1], 0))
    const mnx = Math.min(...xs); const mxx = Math.max(...xs); const mny = Math.min(...ys); const mxy = Math.max(...ys)
    return { x: mnx, y: mny, w: mxx - mnx, h: mxy - mny }
  }
  if (t === 'path') {
    // 简化：用 d 中的数字对做保守包围盒（与工具侧同思路，不含相对命令推演）
    const nums = String(s.d || '').match(/-?\d*\.?\d+/g) || []
    const vals = nums.map(Number).filter((n) => isFinite(n))
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

const coverText = computed(() => {
  if (!shapes.value.length) return '（空）'
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
  return 'x ' + fmt(union.x) + '..' + fmt(union.x + union.w) + '，y ' + fmt(union.y) + '..' + fmt(union.y + union.h) +
    '（占 ' + pct.toFixed(1) + '%）'
})

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
/* 配色一律取自设计系统变量（不另起一套配色），与 tool-music / tool-voice 保持一致 */
.art-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 10px;
  font-size: 12px;
  color: var(--text-primary);
  height: 100%;
  overflow-y: auto;
  box-sizing: border-box;
}
.ap-bar { display: flex; align-items: center; gap: 6px; }
.ap-title { font-weight: 600; color: var(--text-secondary); flex: none; }
.ap-input {
  flex: 1;
  min-width: 0;
  padding: 3px 6px;
  font-size: 11px;
  font-family: ui-monospace, Consolas, monospace;
  color: var(--text-primary);
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
}
.ap-icon-btn {
  display: flex;
  align-items: center;
  padding: 3px 6px;
  color: var(--text-secondary);
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  cursor: pointer;
}
.ap-icon-btn:hover { background: var(--bg-hover); }
.spinning { animation: ap-spin 1s linear infinite; }
@keyframes ap-spin { to { transform: rotate(360deg); } }

.ap-msg { padding: 8px; color: var(--text-muted); line-height: 1.6; }
.ap-err { color: var(--text-primary); border-left: 2px solid var(--accent); background: var(--bg-tertiary); }
.ap-info { color: var(--text-secondary); border-left: 2px solid var(--accent); background: var(--bg-tertiary); }
.ap-sec { display: flex; flex-direction: column; gap: 4px; padding: 8px; background: var(--bg-tertiary); border-radius: 6px; }
.ap-sec-title { display: flex; align-items: center; gap: 6px; font-weight: 600; color: var(--text-secondary); margin-bottom: 2px; }
.ap-kv { display: flex; gap: 8px; justify-content: space-between; }
.ap-kv > span { color: var(--text-muted); flex: none; }
.ap-kv > b { font-weight: 500; text-align: right; word-break: break-all; }
.ap-kv-left > b { text-align: left; overflow-wrap: anywhere; }
.ap-mono { font-family: ui-monospace, Consolas, monospace; font-size: 11px; color: var(--text-secondary); }
.ap-dim { color: var(--text-muted); font-weight: 400; }
.ap-pass { color: var(--text-secondary); }
.ap-fail { color: var(--text-primary); }
.ap-row { display: flex; align-items: center; gap: 6px; }
.ap-spacer { flex: 1; }
.ap-btn {
  padding: 3px 10px;
  font-size: 11px;
  color: var(--text-secondary);
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  cursor: pointer;
}
.ap-btn:hover:not(:disabled) { background: var(--bg-hover); }
.ap-btn:disabled { opacity: 0.5; cursor: default; }

.ap-canvas-wrap { position: relative; width: 100%; }
.ap-canvas { display: block; width: 100%; height: auto; background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 4px; }
/* 选中高亮框：覆盖在预览之上，不拦截鼠标事件 */
.ap-hl { position: absolute; border: 1px dashed var(--accent); pointer-events: none; box-sizing: border-box; }

.ap-table { width: 100%; border-collapse: collapse; font-size: 11px; }
.ap-table th { color: var(--text-muted); font-weight: 500; text-align: left; padding: 2px 4px; border-bottom: 1px solid var(--border-color); }
.ap-table td { padding: 2px 4px; border-bottom: 1px solid var(--bg-hover); }
.ap-table tbody tr { cursor: pointer; }
.ap-table tbody tr:hover { background: var(--bg-hover); }
.ap-row-sel { background: var(--bg-hover); }
.ap-fill { margin-left: 4px; }
.ap-swatch {
  display: inline-block;
  width: 10px;
  height: 10px;
  border: 1px solid var(--border-color);
  border-radius: 2px;
  vertical-align: middle;
  margin-right: 4px;
}
.ap-code {
  margin: 0;
  padding: 6px;
  font-family: ui-monospace, Consolas, monospace;
  font-size: 10px;
  line-height: 1.5;
  color: var(--text-secondary);
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  overflow-x: auto;
  white-space: pre;
}
.ap-badge { padding: 1px 6px; border-radius: 8px; font-size: 11px; }
.ap-dot { width: 7px; height: 7px; border-radius: 50%; flex: none; display: inline-block; }
.ap-ok { background: var(--accent); color: var(--bg-primary); }
.ap-bad { background: var(--text-muted); color: var(--bg-primary); }
.ap-check { display: flex; align-items: center; gap: 6px; }
.ap-check > b { flex: none; color: var(--text-secondary); }
.ap-check-name { flex: none; }
.ap-check-metric { flex: 1; text-align: right; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
