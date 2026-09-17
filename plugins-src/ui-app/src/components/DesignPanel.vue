<template>
  <div class="dp-panel">
    <div class="dp-bar">
      <span class="dp-title">设计</span>
      <input v-model="projectPath" class="dp-input" placeholder="design.project.json" @keyup.enter="load" />
      <button class="dp-icon-btn" :disabled="loading" title="重新载入工程、令牌、预览与校验报告" @click="load">
        <SvgIcon name="refresh" :size="12" :class="{ spinning: loading }" />
      </button>
    </div>

    <div v-if="error" class="dp-msg dp-err">{{ error }}</div>
    <div v-if="!proj && !error" class="dp-msg">
      未找到设计工程。先让 Agent 执行 <code>design_project</code> 创建（生成 design.project.json + design.tokens.json）。
    </div>

    <template v-if="proj">
      <!-- 概要 -->
      <section class="dp-sec">
        <div class="dp-sec-title">{{ title }}</div>
        <div class="dp-kv"><span>屏幕 / 节点 / 导航</span><b>{{ screens.length }} · {{ nodeText }} · {{ flowCount }}</b></div>
        <div class="dp-kv"><span>令牌 / 变更</span><b>{{ tokenCount }} 项 · {{ updatedAt }}</b></div>
        <div class="dp-kv"><span>文件</span><b class="dp-mono">{{ projectPath }}</b></div>
      </section>

      <!-- 预览（iframe srcdoc：sandbox 无 allow-scripts → 预览脚本一律不执行）；两种视图：整页 / 逐屏并排 -->
      <section class="dp-sec">
        <div class="dp-row">
          <span class="dp-sec-title">预览</span>
          <span class="dp-modes">
            <button class="dp-tab" :class="{ on: viewMode === 'page' }" @click="viewMode = 'page'">整页</button>
            <button
              class="dp-tab" :class="{ on: viewMode === 'split' }" :disabled="!screenDocs.length"
              :title="screenDocs.length ? '每屏一个独立 iframe，便于逐屏对比' : '需要先导出 design.html'"
              @click="viewMode = 'split'"
            >并排（{{ screenDocs.length }} 屏）</button>
          </span>
          <span class="dp-spacer"></span>
          <span class="dp-dim">{{ htmlReady ? 'design.html' : '未导出' }}</span>
        </div>
        <template v-if="htmlText">
          <iframe v-if="viewMode === 'page'" class="dp-frame" :srcdoc="htmlText" sandbox="" referrerpolicy="no-referrer"></iframe>
          <div v-else class="dp-split">
            <div v-for="d in screenDocs" :key="d.id" class="dp-split-col" :style="{ width: colWidth(d) + 'px' }">
              <div class="dp-split-head">
                <b class="dp-mono">{{ d.id }}</b>
                <span class="dp-dim">{{ d.name }} · {{ d.w }}×{{ d.h }}</span>
                <span class="dp-spacer"></span>
                <span class="dp-dim dp-mono">{{ colWidth(d) }}px</span>
              </div>
              <iframe class="dp-split-frame" :srcdoc="d.doc" sandbox="" referrerpolicy="no-referrer"></iframe>
              <span
                class="dp-split-grip" title="拖动调整该栏宽度（双击复位为设计稿宽度）"
                @mousedown="startResize(d, $event)" @dblclick="resetCol(d)"
              ></span>
            </div>
          </div>
        </template>
        <div v-else class="dp-dim dp-msg">
          未找到 HTML 预览。让 Agent 执行 <code>design_export format=html</code> 生成（默认 design.html）。
        </div>
        <div class="dp-dim">
          {{ viewMode === 'page'
            ? 'iframe 内可滚动查看全部屏幕（尺寸为设计稿原尺寸）；本面板只读，不写工作区。'
            : '并排视图：每屏一个独立 iframe（复用同一份 design.html 的样式 + 该屏 DOM 切片），横向滚动查看全部；本面板只读。' }}
        </div>
      </section>

      <!-- 设计令牌：色板 + 对比度（面板端按 WCAG 复算，与 D3 同口径） -->
      <section class="dp-sec">
        <div class="dp-sec-title">颜色令牌 <span class="dp-dim">（{{ colorRows.length }} 项；对比度对 $color.bg）</span></div>
        <div v-for="c in colorRows" :key="c.name" class="dp-kv">
          <span>
            <span class="dp-swatch" :style="{ background: c.value }"></span>
            <b class="dp-mono">{{ c.name }}</b>
            <span class="dp-dim"> {{ c.value }}</span>
          </span>
          <b v-if="c.ratio !== null" :class="c.pass ? 'dp-pass' : 'dp-fail'">
            {{ c.ratio.toFixed(2) }}:1<span v-if="!c.pass" class="dp-dim"> · 低于 4.5</span>
          </b>
        </div>
        <div class="dp-dim">数值令牌：字号 {{ sizeText }}｜间距 {{ spaceText }}</div>
      </section>

      <!-- 屏幕 -->
      <section class="dp-sec">
        <div class="dp-sec-title">屏幕 <span class="dp-dim">（{{ screens.length }}；来自工程真相源）</span></div>
        <div v-for="s in screens" :key="s.id" class="dp-kv">
          <span class="dp-mono">{{ s.id }}</span>
          <b>{{ s.name }} <span class="dp-dim">{{ s.w }}×{{ s.h }} · {{ s.nodes }} 节点 · 背景 {{ s.bg }}</span></b>
        </div>
        <div v-if="flowRows.length" class="dp-dim">导航：{{ flowRows.join(' ｜ ') }}</div>
      </section>

      <!-- 校验（旁挂报告设计）/ 10 项判据（D10 = 交互声明） -->
      <section v-if="verify" class="dp-sec">
        <div class="dp-row">
          <span class="dp-sec-title">校验</span>
          <span class="dp-spacer"></span>
          <span :class="['dp-badge', passedAll ? 'dp-ok' : 'dp-bad']">{{ passedCount }}/{{ verify.checks.length }}</span>
        </div>
        <div v-for="c in verify.checks" :key="c.id" class="dp-check">
          <span :class="['dp-dot', c.pass ? 'dp-ok' : 'dp-bad']"></span>
          <b>{{ c.id }}</b>
          <span class="dp-check-name">{{ c.name }}</span>
          <span class="dp-dim dp-check-metric" :title="c.metric">{{ c.metric }}</span>
        </div>
        <div class="dp-dim">报告时间：{{ verify.ts }}</div>
      </section>

      <!-- ★ 前端代码（design_codegen 落地产物）：设计 → 工程代码的落地状态 -->
      <section class="dp-sec">
        <div class="dp-row">
          <span class="dp-sec-title">前端代码</span>
          <span class="dp-spacer"></span>
          <span class="dp-dim">{{ codegen ? codegen.targets.join(' / ') : '未生成' }}</span>
        </div>
        <template v-if="codegen">
          <div class="dp-kv">
            <span>落点 / 文件</span>
            <b class="dp-mono">{{ codegen.targetDir }}/ <span class="dp-dim">{{ codegen.files.length }} 个 · {{ codegenSize }}</span></b>
          </div>
          <div class="dp-kv">
            <span>生成时间</span>
            <b>{{ codegenTime }}<span v-if="staleCodegen" class="dp-stale"> · 设计已变更，建议重新生成</span></b>
          </div>
          <div class="dp-kv">
            <span>屏幕</span><b class="dp-mono">{{ (codegen.screens || []).join(' · ') }}</b>
          </div>
          <div v-if="(codegen.components || []).length" class="dp-kv">
            <span>抽出组件</span><b class="dp-mono">{{ codegenComponents }}</b>
          </div>
          <div v-if="(codegen.duplicates || []).length" class="dp-kv">
            <span>重复区块候选</span>
            <b class="dp-mono">{{ codegenDuplicates }}</b>
          </div>
          <div v-if="(codegen.nearDuplicates || []).length" class="dp-kv">
            <span>近似区块</span>
            <b class="dp-mono">{{ codegenNear }}</b>
          </div>
          <div v-if="(codegen.interactions || []).length" class="dp-kv">
            <span>交互</span>
            <b class="dp-mono">{{ codegenInteractions }}</b>
          </div>
          <div v-if="codegenInteractionsBad" class="dp-dim dp-warn">{{ codegenInteractionsBad }}</div>
          <div v-for="f in codegenFiles" :key="f.path" class="dp-kv">
            <span class="dp-mono">{{ f.path }}</span>
            <b class="dp-dim">{{ f.kb }}</b>
          </div>
          <div v-if="(codegen.backups || []).length" class="dp-dim">
            覆盖备份 {{ codegen.backups.length }} 个（.bak，可人工回滚）
          </div>
          <div v-for="(w, i) in (codegen.warnings || [])" :key="'w' + i" class="dp-dim dp-warn">{{ w }}</div>
        </template>
        <div v-else class="dp-dim dp-msg">
          尚未生成前端代码。用 <b class="dp-mono">design_codegen</b> 从设计生成 HTML+CSS / Vue SFC / React 组件，
          并落到 <b class="dp-mono">design-export/</b>（或指定的 targetDir）。
        </div>
        <pre class="dp-code">{{ codegenCmd }}</pre>
        <button class="dp-btn" @click="copyCodegenCmd">{{ codegenCopied ? '已复制 ✓' : '复制生成命令' }}</button>
      </section>

      <!-- 面板只读：需要改动时给出可直接复制的命令 -->
      <section class="dp-sec">
        <div class="dp-sec-title">交给 Agent 执行</div>
        <div class="dp-dim">面板不写工作区（避免第二条写路径）；下面是常用命令，复制后交给 Agent：</div>
        <pre class="dp-code">{{ sampleCmd }}</pre>
        <button class="dp-btn" @click="copyCmd">{{ copied ? '已复制 ✓' : '复制命令' }}</button>
      </section>
    </template>
  </div>
</template>

<script setup>
// DesignPanel — UI 设计工程只读视图（与 Agent 共用同一份真相源：design.project.json + design.tokens.json
// + 产物 design.html + 旁挂 design.verify.json）。
// 设计取舍（与 tool-art / tool-music / tool-voice 一致）：面板不写工作区 —— 节点/令牌编辑一律经 Agent 的
// design_edit（命令式 op），避免「工具写工程 / 面板写工程」两条写路径导致真相源分叉。
import { ref, computed, onMounted, watch } from 'vue'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'
import { state } from '../ui-state.js'

const projectPath = ref('design.project.json')
const verifyPath = ref('design.verify.json')
const loading = ref(false)
const error = ref('')
const proj = ref(null)
const tokens = ref(null)
const verify = ref(null)
const codegen = ref(null)       // design_codegen 落地报告（codegen.report.json）
const htmlText = ref('')
const copied = ref(false)
const codegenCopied = ref(false)

const title = computed(() => ((proj.value && proj.value.meta && proj.value.meta.title) || '未命名设计'))
// ★ 映射为视图模型：工程里是 width/height/background/root，模板用 w/h/bg/nodes
//   （直接透传原始屏幕对象会让模板读不到字段 → 显示成"× · 节点 · 背景"空白，实测踩过）
const screens = computed(() => ((proj.value && proj.value.screens) || []).map((s) => ({
  id: s.id,
  name: s.name || s.id,
  w: s.width,
  h: s.height,
  bg: s.background || '$color.bg',
  nodes: countNodes(s.root),
})))
const updatedAt = computed(() => ((proj.value && proj.value.meta && proj.value.meta.updatedAt) || '—').slice(0, 19).replace('T', ' '))
const htmlReady = computed(() => htmlText.value.length > 0)
const flowCount = computed(() => (((proj.value && proj.value.flow) || []).length) + ' 条')
const passedAll = computed(() => ((verify.value && verify.value.checks) || []).every((c) => c.pass))
const passedCount = computed(() => ((verify.value && verify.value.checks) || []).filter((c) => c.pass).length)

function countNodes(node) {
  if (!node) return 0
  let n = 1
  ;(node.children || []).forEach((k) => { n += countNodes(k) })
  return n
}
const nodeText = computed(() => screens.value.reduce((acc, s) => acc + s.nodes, 0) + ' 个')

// 多视图并排：把 design.html 按 [data-screen] 切成每屏一份独立文档（head 样式整份复用），
//   用面板自己的元数据补屏幕名与尺寸 —— 不额外调用工具，也不写工作区（面板仍是只读）。
const viewMode = ref('page')
const colWidths = ref({})          // { 屏幕 id: 像素宽 } —— 并排视图每栏宽度（默认 = 设计稿宽度，可拖拽）
function colWidth(d) { return colWidths.value[d.id] || (Number(d.w) || 320) }
function resetCol(d) {
  const next = { ...colWidths.value }
  delete next[d.id]
  colWidths.value = next
}
function startResize(d, ev) {
  ev.preventDefault()
  const startX = ev.clientX
  const startW = colWidth(d)
  const onMove = (e) => {
    const w = Math.max(160, Math.min(1400, Math.round(startW + (e.clientX - startX))))
    colWidths.value = { ...colWidths.value, [d.id]: w }
  }
  const onUp = () => {
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('mouseup', onUp)
  }
  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseup', onUp)
}
const screenDocs = computed(() => {
  const html = htmlText.value
  if (!html) return []
  try {
    const doc = new DOMParser().parseFromString(html, 'text/html')
    const head = doc.head ? doc.head.innerHTML : ''
    const meta = new Map(screens.value.map((s) => [String(s.id), s]))
    return Array.from(doc.body.querySelectorAll('[data-screen]')).map((el) => {
      const id = String(el.getAttribute('data-screen') || '')
      const m = meta.get(id) || {}
      return {
        id,
        name: m.name || id,
        w: m.w || '?',
        h: m.h || '?',
        doc: '<!doctype html><html><head>' + head + '</head><body style="margin:0">' + el.outerHTML + '</body></html>',
      }
    })
  } catch (e) { return [] }
})

const colorRows = computed(() => {
  const c = (tokens.value && tokens.value.color) || {}
  const bg = parseColor(c.bg) || { r: 255, g: 255, b: 255 }
  return Object.keys(c).map((name) => {
    const fg = parseColor(c[name])
    const r = fg ? contrast(fg, bg) : null
    return { name, value: c[name], ratio: r, pass: r === null ? true : r >= 4.5 }
  })
})
const tokenCount = computed(() => {
  if (!tokens.value) return 0
  return ['color', 'space', 'radius', 'fontSize', 'fontWeight', 'lineHeight', 'shadow', 'font']
    .reduce((n, g) => n + Object.keys(tokens.value[g] || {}).length, 0)
})
const sizeText = computed(() => {
  const s = (tokens.value && tokens.value.fontSize) || {}
  return Object.keys(s).map((k) => k + '=' + s[k]).join(' ') || '—'
})
const spaceText = computed(() => {
  const s = (tokens.value && tokens.value.space) || {}
  return Object.keys(s).map((k) => k + '=' + s[k]).join(' ') || '—'
})
const flowRows = computed(() => (((proj.value && proj.value.flow) || []).map((e) => e.from + ' → ' + e.to + (e.label ? '（' + e.label + '）' : ''))))
// ── 前端代码（design_codegen）落地状态 ──
// 报告路径：工程 artifacts.codegen 优先，否则默认落点 design-export/codegen.report.json
const codegenPath = computed(() => {
  const art = (proj.value && proj.value.artifacts) || {}
  return art.codegen || 'design-export/codegen.report.json'
})
function humanKb(n) {
  const v = Number(n) || 0
  return v < 1024 ? v + ' B' : (Math.round(v / 102.4) / 10) + ' KB'
}
const codegenFiles = computed(() => (((codegen.value && codegen.value.files) || []).map((f) => ({ path: f.path, kb: humanKb(f.bytes) }))))
// 抽出的组件（design_codegen component=<节点 id|auto>）：组件名 [×实例数] ← 原型屏幕/节点 [（N props）] [（插槽）]
const codegenComponents = computed(() => (((codegen.value && codegen.value.components) || [])
  .map((c) => {
    const insts = c.instances || []
    const n = insts.length || 1
    const first = insts[0] || { screen: c.screen, id: c.id }
    const ps = (c.props || []).length
    // 插槽：新版报告给 slots 数组（可多个具名插槽），旧报告只有 slot（首个槽名）→ 兼容读取
    const slots = (c.slots && c.slots.length) ? c.slots : (c.slot ? [{ name: c.slot }] : [])
    const slotTxt = slots.length
      ? '（' + slots.length + ' 插槽：' + slots.map((s) => s.name).join(' / ') + '）'
      : ''
    // 近似组（component=near）：配色差异提升为 CSS 变量（面板给出字段 → 变量名，便于按名覆盖）
    const nearVars = (c.nearVars || [])
    const nearTxt = (c.near && nearVars.length)
      ? '（配色 → 变量 ' + nearVars.map((v) => v.fields.map((f) => f.cssVar).join('/')).join('、') + '）'
      : ''
    return c.name + (n > 1 ? ' ×' + n : '') + ' ← ' + first.screen + '/' + first.id +
      (ps ? '（' + ps + ' props）' : '') + slotTxt + nearTxt
  }).join(' · ')))
// 同构重复区块候选（design_codegen detect=true 时写入报告）：只列前 2 组（长文本会撑破面板行），其余计数
const codegenDuplicates = computed(() => {
  const list = (codegen.value && codegen.value.duplicates) || []
  if (!list.length) return ''
  const head = list.slice(0, 2).map((d) => d.name + '（' + d.type + '）×' + d.count).join(' · ')
  return '共 ' + list.length + ' 组：' + head + (list.length > 2 ? ' 等' : '') + '（component=auto 可抽）'
})
// 近似同构区块（结构相同、只有颜色类字段不同）：**不自动抽**（组件默认值只能一份，配色会被压平），
//   面板只提示差异字段 —— 用户可按提示对齐配色后再 auto，或确认是有意区分
const codegenNear = computed(() => {
  const list = (codegen.value && codegen.value.nearDuplicates) || []
  if (!list.length) return ''
  const head = list.slice(0, 2)
    .map((d) => d.name + '（' + d.type + '）×' + d.count + ' 差异 ' + (d.diffKeys || []).join('/'))
    .join(' · ')
  return '共 ' + list.length + ' 组：' + head + (list.length > 2 ? ' 等' : '') + '（配色不同 → 默认不抽；要复用传 component=near）'
})
// 交互（节点 props.action）：link/emit/toggle 计数 + 有问题的声明（与 D10 同源，报告里已带 error）
const codegenInteractions = computed(() => {
  const list = (codegen.value && codegen.value.interactions) || []
  if (!list.length) return ''
  const c = { link: 0, emit: 0, toggle: 0, invalid: 0 }
  list.forEach((x) => { c[x.verb] = (c[x.verb] || 0) + 1 })
  const parts = []
  if (c.link) parts.push('link ×' + c.link)
  if (c.emit) parts.push('emit ×' + c.emit)
  if (c.toggle) parts.push('toggle ×' + c.toggle)
  if (c.invalid) parts.push('非法 ×' + c.invalid)
  const bad = list.filter((x) => x.error).length - c.invalid
  if (bad > 0) parts.push('有问题 ×' + bad)     // 语法合法但类型不适配（如把 input 当链接）
  return list.length + ' 处：' + parts.join(' / ')
})
const codegenInteractionsBad = computed(() => {
  const bad = ((codegen.value && codegen.value.interactions) || []).filter((x) => x.error)
  if (!bad.length) return ''
  const head = bad.slice(0, 3).map((x) => x.screen + '/' + x.id + '（' + x.action + '）→ ' + x.error).join('；')
  return '交互问题：' + head + (bad.length > 3 ? '；另 ' + (bad.length - 3) + ' 处' : '')
})
const codegenSize = computed(() => humanKb(((codegen.value && codegen.value.files) || []).reduce((n, f) => n + (Number(f.bytes) || 0), 0)))
const codegenTime = computed(() => String((codegen.value && codegen.value.ts) || '—').slice(0, 19).replace('T', ' '))
// 设计变更时间晚于生成时间 → 代码可能过期（面板只提示，不自动生成）
const staleCodegen = computed(() => {
  const a = String((proj.value && proj.value.meta && proj.value.meta.updatedAt) || '')
  const b = String((codegen.value && codegen.value.ts) || '')
  return !!a && !!b && a > b
})
const codegenCmd = computed(() => 'design_codegen targets=["html","vue","react"]' +
  (codegen.value && codegen.value.targetDir ? ' targetDir=' + codegen.value.targetDir : '') +
  '\n\n# 只要一种：targets=["vue"]｜落到源码目录：targetDir=src/components｜先看不写盘：dryRun=true')

const sampleCmd = computed(() => JSON.stringify({
  op: 'node.add', screen: (screens.value[0] || {}).id || 'home', parent: 'root',
  node: { type: 'text', text: '新文案', size: '$fontSize.md', color: '$color.fg' },
}) + '\n\n# 交互（节点 props.action）："link:#屏幕" / "link:https://…" / "emit:名字" / "toggle"'
  + '\n# 近似区块复用（配色不同 → CSS 变量）：component=near'
  + '\n\n# 校验 + 出预览\ndesign_verify\ndesign_export format=html overwrite=true')

// ── 颜色工具（与 tool-design D3 同口径，仅用于展示） ──
function parseColor(v) {
  const s = String(v || '').trim()
  const m6 = /^#([0-9a-fA-F]{6})$/.exec(s)
  if (m6) return { r: parseInt(m6[1].slice(0, 2), 16), g: parseInt(m6[1].slice(2, 4), 16), b: parseInt(m6[1].slice(4, 6), 16) }
  const m3 = /^#([0-9a-fA-F]{3})$/.exec(s)
  if (m3) return { r: parseInt(m3[1][0] + m3[1][0], 16), g: parseInt(m3[1][1] + m3[1][1], 16), b: parseInt(m3[1][2] + m3[1][2], 16) }
  return null
}
function lum(c) {
  const f = (v) => { const x = v / 255; return x <= 0.03928 ? x / 12.92 : Math.pow((x + 0.055) / 1.055, 2.4) }
  return 0.2126 * f(c.r) + 0.7152 * f(c.g) + 0.0722 * f(c.b)
}
function contrast(a, b) {
  const la = lum(a); const lb = lum(b)
  return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05)
}

function absPath(rel) {
  const root = String((state && state.workspaceRoot) || '').replace(/[\\/]+$/, '')
  if (!root) return rel
  return root + '/' + String(rel).replace(/^[\\/]+/, '')
}
async function readText(rel) {
  // ★ apiURL 自带 /api 前缀（api.js BASE='/api'）——再写 /api 会拼成 /api/api/fs/read → 404
  const r = await api.apiGet('/fs/read', { path: absPath(rel) })
  return r.content
}
async function readIfExists(rel) {
  try { return await readText(rel) } catch (_) { return '' }
}

async function load() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    proj.value = JSON.parse(await readText(projectPath.value))
    tokens.value = null
    const tkRel = (proj.value && proj.value.tokens) || 'design.tokens.json'
    const tkTxt = await readIfExists(tkRel)
    if (tkTxt) { try { tokens.value = JSON.parse(tkTxt) } catch (_) { /* 令牌损坏时面板仍显示工程 */ } }
    verify.value = null
    const vTxt = await readIfExists(verifyPath.value)
    if (vTxt) { try { verify.value = JSON.parse(vTxt) } catch (_) { /* 报告可选 */ } }
    htmlText.value = ''
    const art = (proj.value && proj.value.artifacts) || {}
    const htmlRel = art.html || String(projectPath.value).replace(/\.json$/i, '.html')
    htmlText.value = await readIfExists(htmlRel)
    codegen.value = null
    const cgTxt = await readIfExists(codegenPath.value)
    if (cgTxt) { try { codegen.value = JSON.parse(cgTxt) } catch (_) { /* 落地报告可选 */ } }
  } catch (e) {
    proj.value = null
    htmlText.value = ''
    codegen.value = null
    error.value = '载入失败：' + ((e && e.message) || e)
  } finally {
    loading.value = false
  }
}

async function copyCmd() {
  try {
    await navigator.clipboard.writeText(sampleCmd.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1500)
  } catch (_) { /* 剪贴板不可用时静默 */ }
}
async function copyCodegenCmd() {
  try {
    await navigator.clipboard.writeText(codegenCmd.value)
    codegenCopied.value = true
    setTimeout(() => { codegenCopied.value = false }, 1500)
  } catch (_) { /* 剪贴板不可用时静默 */ }
}

// 工作区/工程切换时自动重载
watch(projectPath, () => { load() })
watch(() => state.workspaceRoot, () => { load() })
onMounted(load)
</script>

<style scoped>
.dp-panel { padding: 8px; font-size: 12px; color: var(--text-primary); }
.dp-bar { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
.dp-title { font-size: 12px; color: var(--text-secondary); flex: none; }
.dp-input {
  flex: 1; min-width: 0; padding: 2px 6px; font-size: 11px;
  color: var(--text-primary); background: var(--bg-primary);
  border: 1px solid var(--border-color); border-radius: 4px;
}
.dp-icon-btn {
  display: inline-flex; align-items: center; justify-content: center;
  width: 22px; height: 22px; color: var(--text-secondary);
  background: transparent; border: 1px solid var(--border-color);
  border-radius: 4px; cursor: pointer;
}
.dp-icon-btn:hover:not(:disabled) { background: var(--bg-hover); }
.dp-icon-btn:disabled { opacity: 0.5; cursor: default; }

.dp-msg { color: var(--text-muted); padding: 6px 2px; line-height: 1.6; }
.dp-err { color: var(--text-primary); }
.dp-sec { margin-bottom: 12px; padding-bottom: 10px; border-bottom: 1px solid var(--border-color); }
.dp-sec:last-child { border-bottom: none; }
.dp-sec-title { font-size: 12px; font-weight: 500; color: var(--text-secondary); margin-bottom: 6px; }
.dp-kv { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; padding: 2px 0; }
.dp-kv > span { color: var(--text-muted); flex: none; }
.dp-kv > b { color: var(--text-primary); font-weight: 500; text-align: right; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dp-mono { font-family: ui-monospace, Consolas, monospace; }
.dp-dim { color: var(--text-muted); font-weight: 400; font-size: 11px; line-height: 1.6; }
/* 前端代码区：过期/告警提示用主题强调色（项目主题只有 text/border/accent 三系，
   不引入新色值） */
.dp-stale { color: var(--accent); font-size: 11px; }
.dp-warn { color: var(--accent); }
.dp-pass { color: var(--text-secondary); }
.dp-fail { color: var(--text-primary); }
.dp-row { display: flex; align-items: center; gap: 6px; }
.dp-spacer { flex: 1; }
.dp-btn {
  padding: 3px 10px; font-size: 11px; color: var(--text-secondary);
  background: transparent; border: 1px solid var(--border-color);
  border-radius: 4px; cursor: pointer;
}
.dp-btn:hover:not(:disabled) { background: var(--bg-hover); }
.dp-frame {
  display: block; width: 100%; height: 460px; background: #fff;
  border: 1px solid var(--border-color); border-radius: 4px;
}
.dp-modes { display: inline-flex; gap: 3px; }
.dp-tab {
  padding: 2px 8px; font-size: 11px; color: var(--text-secondary); cursor: pointer;
  background: transparent; border: 1px solid var(--border-color); border-radius: 4px;
}
.dp-tab:hover:not(:disabled) { background: var(--bg-hover); }
.dp-tab.on { color: var(--text-primary); border-color: var(--accent); }
.dp-tab:disabled { opacity: .45; cursor: default; }
.dp-split { display: flex; gap: 10px; overflow-x: auto; padding: 2px 2px 8px; }
.dp-split-col { flex: 0 0 auto; position: relative; }
.dp-split-head { display: flex; align-items: baseline; gap: 6px; margin-bottom: 4px; font-size: 11px; }
.dp-split-frame {
  display: block; width: 100%; height: 380px; background: #fff;
  border: 1px solid var(--border-color); border-radius: 4px;
}
.dp-split-grip {
  position: absolute; top: 18px; right: -7px; width: 7px; height: calc(100% - 18px);
  cursor: col-resize; border-radius: 2px;
}
.dp-split-grip:hover { background: var(--accent); opacity: .5; }
.dp-swatch {
  display: inline-block; width: 10px; height: 10px; margin-right: 4px;
  border: 1px solid var(--border-color); border-radius: 2px; vertical-align: middle;
}
.dp-code {
  margin: 0 0 6px; padding: 6px; font-family: ui-monospace, Consolas, monospace;
  font-size: 10px; line-height: 1.5; color: var(--text-secondary);
  background: var(--bg-primary); border: 1px solid var(--border-color);
  border-radius: 4px; overflow-x: auto; white-space: pre;
}
.dp-badge { padding: 1px 6px; border-radius: 8px; font-size: 11px; }
.dp-dot { width: 7px; height: 7px; border-radius: 50%; flex: none; display: inline-block; }
.dp-ok { background: var(--accent); color: var(--bg-primary); }
.dp-bad { background: var(--text-muted); color: var(--bg-primary); }
.dp-check { display: flex; align-items: center; gap: 6px; padding: 1px 0; }
.dp-check > b { flex: none; color: var(--text-secondary); }
.dp-check-name { flex: none; }
.dp-check-metric { flex: 1; text-align: right; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
