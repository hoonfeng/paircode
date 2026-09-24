<template>
  <PanelShell
    v-model:file="projectPath"
    title="设计"
    icon="layers"
    placeholder="design.project.json"
    :badge="verifyBadge"
    :badge-tone="verifyTone"
    :metrics="metrics"
    :loading="loading"
    :error="proj ? error : ''"
    :empty="!proj"
    reload-title="重新载入工程、令牌、预览与校验报告"
    footnote="只读面板：改设计请让 Agent 执行 design_edit（命令式 op）；预览是静态 HTML（脚本不执行）。"
    :source="projectPath"
    @reload="load"
  >
    <template #head-actions>
      <button v-if="htmlText" class="pn-btn" title="在浏览器新标签打开完整预览（原始尺寸，可全屏感受）" @click="openPreview">
        在新窗口打开
      </button>
    </template>

    <template #empty>
      <div class="pn-empty-title">还没有设计工程</div>
      <div class="pn-tip">
        面板读工作区根目录的 <code>design.project.json</code>（与 Agent 共用同一份真相源），当前没有这个文件，所以没有可看的内容。
      </div>
      <div class="pn-card">
        <div class="pn-card-title">三步开始</div>
        <div class="pn-empty-step"><i>1</i><span>让 Agent 执行 <code>design_project</code> 创建工程（屏幕 id / 尺寸 / 背景）。</span></div>
        <div class="pn-empty-step"><i>2</i><span>用 <code>design_edit</code> 加节点与令牌（node.add / token.set，样式一律引用令牌）。</span></div>
        <div class="pn-empty-step"><i>3</i><span><code>design_verify</code> 自检 11 项，<code>design_export format=html</code> 出预览。</span></div>
      </div>
      <div v-if="error" class="pn-tip pn-mono">{{ error }}</div>
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

    <!-- 左栏：屏幕 / 令牌 / 校验 / 代码 -->
    <template #side>
      <div v-if="tab === 'screens'" class="pn-list">
        <button
          v-for="s in screens"
          :key="s.id"
          class="pn-item"
          :title="s.id + ' · ' + s.name + '（点击滚动到该屏）'"
          @click="gotoScreen(s.id)"
        >
          <span class="pn-item-k">{{ s.id }}</span>
          <span class="pn-dim pn-mini">{{ s.w }}×{{ s.h }} · {{ s.nodes }}</span>
        </button>
        <div class="pn-tip dp-side-hint">{{ title }}</div>
        <div v-if="flowRows.length" class="dp-side-block">
          <div class="pn-card-title">导航 {{ flowRows.length }} 条</div>
          <div v-for="(f, i) in flowRows" :key="i" class="pn-tip pn-mono">{{ f }}</div>
        </div>
        <div v-else class="pn-tip dp-side-hint">无屏幕间导航（flow 为空）。</div>
      </div>

      <div v-else-if="tab === 'tokens'" class="pn-list">
        <div v-for="c in colorRows" :key="c.name" class="pn-kv">
          <span class="pn-swatch" :style="{ background: c.value }"></span>
          <b class="pn-mono pn-mini">{{ c.name }}</b>
          <span class="pn-spacer"></span>
          <b v-if="c.ratio !== null" :class="c.pass ? '' : 'pn-warn'">{{ c.ratio.toFixed(2) }}:1</b>
          <span v-else class="pn-dim">—</span>
        </div>
        <div class="dp-side-block">
          <div class="pn-card-title">字号</div>
          <div class="pn-tip pn-mono">{{ sizeText }}</div>
        </div>
        <div class="dp-side-block">
          <div class="pn-card-title">间距</div>
          <div class="pn-tip pn-mono">{{ spaceText }}</div>
        </div>
        <div class="pn-tip dp-side-hint">对比度对 <span class="pn-mono">$color.bg</span> 复算（与校验 D3 同口径）。</div>
      </div>

      <div v-else-if="tab === 'checks'" class="pn-list">
        <template v-if="verify">
          <div v-for="c in verify.checks || []" :key="c.id" class="pn-check">
            <span class="pn-dot" :class="{ 'pn-dot-ok': c.pass }"></span>
            <b>{{ c.id }}</b>
            <span class="pn-check-name">{{ c.name }}</span>
            <span class="pn-check-metric" :title="c.metric">{{ c.metric }}</span>
          </div>
          <div class="pn-tip dp-side-hint">报告时间：{{ verify.ts }}</div>
        </template>
        <div v-else class="pn-tip">
          还没有校验报告。让 Agent 执行 <code>design_verify</code> 生成旁挂的 design.verify.json。
        </div>
      </div>

      <div v-else class="pn-list">
        <template v-if="codegen">
          <div class="pn-kv"><span>落点</span><b class="pn-mono">{{ codegen.targetDir }}/</b></div>
          <div class="pn-kv"><span>产物</span><b>{{ codegen.files.length }} 个 · {{ codegenSize }}</b></div>
          <div class="pn-kv">
            <span>生成时间</span>
            <b>{{ codegenTime }}<span v-if="staleCodegen" class="pn-warn"> · 已过期</span></b>
          </div>
          <div class="pn-kv"><span>形态</span><b class="pn-mono">{{ codegen.targets.join(' / ') }}</b></div>
          <div class="pn-kv"><span>屏幕</span><b class="pn-mono dp-wrap">{{ (codegen.screens || []).join(' · ') }}</b></div>
          <div v-if="(codegen.components || []).length" class="pn-kv">
            <span>组件</span><b class="pn-mono dp-wrap">{{ codegenComponents }}</b>
          </div>
          <div v-if="(codegen.duplicates || []).length" class="pn-kv">
            <span>重复候选</span><b class="pn-mono dp-wrap">{{ codegenDuplicates }}</b>
          </div>
          <div v-if="(codegen.nearDuplicates || []).length" class="pn-kv">
            <span>近似区块</span><b class="pn-mono dp-wrap">{{ codegenNear }}</b>
          </div>
          <div v-if="(codegen.interactions || []).length" class="pn-kv">
            <span>交互</span><b class="pn-mono dp-wrap">{{ codegenInteractions }}</b>
          </div>
          <div v-if="codegenInteractionsBad" class="pn-tip pn-warn">{{ codegenInteractionsBad }}</div>
          <div v-for="f in codegenFiles" :key="f.path" class="pn-kv">
            <span class="pn-mono pn-mini">{{ f.path }}</span><b class="pn-dim">{{ f.kb }}</b>
          </div>
          <div v-if="(codegen.backups || []).length" class="pn-tip">
            覆盖备份 {{ codegen.backups.length }} 个（.bak，可人工回滚）
          </div>
          <div v-for="(w, i) in (codegen.warnings || [])" :key="'w' + i" class="pn-tip pn-warn">{{ w }}</div>
        </template>
        <template v-else>
          <div class="pn-tip">
            尚未生成前端代码。用 <code>design_codegen</code> 从设计生成 HTML+CSS / Vue SFC / React，
            默认落到 <code>design-export/</code>（或指定 targetDir）。
          </div>
        </template>
        <div class="dp-side-block">
          <pre class="pn-code">{{ codegenCmd }}</pre>
          <button class="pn-btn" @click="copyCodegenCmd">{{ codegenCopied ? '已复制 ✓' : '复制生成命令' }}</button>
        </div>
        <div class="dp-side-block">
          <div class="pn-card-title">常用命令</div>
          <pre class="pn-code">{{ sampleCmd }}</pre>
          <button class="pn-btn" @click="copyCmd">{{ copied ? '已复制 ✓' : '复制命令' }}</button>
        </div>
      </div>
    </template>

    <!-- 主区：预览（整页 / 并排 + 属性标注） -->
    <template #main>
      <div class="dp-toolbar">
        <div class="pn-row-gap">
          <button class="pn-btn" :class="{ 'pn-btn-on': viewMode === 'page' }" title="整份预览竖向排列，等比缩放到面板宽度" @click="viewMode = 'page'">整页</button>
          <button
            class="pn-btn"
            :class="{ 'pn-btn-on': viewMode === 'split' }"
            :disabled="!screenDocs.length"
            :title="screenDocs.length ? '每屏一栏，各自等比缩放完整显示；可拖右边缘改栏宽' : '需要先导出 HTML 预览'"
            @click="viewMode = 'split'"
          >并排 {{ screenDocs.length }} 屏</button>
        </div>

        <div v-if="viewMode === 'page' && htmlText" class="pn-row-gap">
          <button class="pn-btn" :class="{ 'pn-btn-on': zoomMode === 'fit' }" title="等比缩放到预览区宽度（看整体）" @click="zoomMode = 'fit'">适应宽度</button>
          <button class="pn-btn" :class="{ 'pn-btn-on': zoomMode === '1x' }" title="按设计稿原始像素显示（看细节，可滚动）" @click="zoomMode = '1x'">1:1</button>
          <span class="pn-dim pn-mini pn-mono">{{ Math.round(scale * 100) }}%</span>
        </div>
        <div v-else-if="viewMode === 'split' && screenDocs.length" class="pn-row-gap">
          <button class="pn-btn" :class="{ 'pn-btn-on': splitFit }" title="每栏按栏宽等比缩放，完整看到整屏" @click="splitFit = true">适应栏宽</button>
          <button class="pn-btn" :class="{ 'pn-btn-on': !splitFit }" title="按设计稿原始像素（可拖栏宽看细节）" @click="splitFit = false">1:1</button>
        </div>

        <button
          v-if="viewMode === 'page' && htmlText"
          class="pn-btn"
          :class="{ 'pn-btn-on': annotOn }"
          title="在每个元素旁标出实际尺寸 / 字号 / 颜色（悬停看完整信息）；纯本地叠加层，不写工作区"
          @click="annotOn = !annotOn"
        >属性标注{{ annots.length ? ' ' + annots.length : '' }}</button>

        <span class="pn-spacer"></span>
        <span class="pn-dim pn-mini">{{ viewHint }}</span>
      </div>

      <div ref="stageEl" class="dp-stage-wrap">
        <template v-if="htmlText">
          <div v-if="viewMode === 'page'" class="dp-stage" :style="{ height: stageH + 'px' }">
            <iframe
              ref="frameEl"
              class="dp-frame"
              :srcdoc="htmlText"
              sandbox="allow-same-origin"
              referrerpolicy="no-referrer"
              :style="{ width: contentSize.w + 'px', height: contentSize.h + 'px', transform: 'scale(' + scale + ')' }"
              @load="onFrameLoad"
            ></iframe>
            <!-- 属性标注层：坐标来自 iframe 内真实 DOM（同源只读），随 scale 缩放对齐 -->
            <div v-if="annotOn" class="dp-annot-layer">
              <div
                v-for="(a, i) in annots"
                :key="i"
                class="dp-annot"
                :class="{ 'dp-annot-txt': a.isText }"
                :style="a.style"
                :title="a.title"
              >
                <span class="dp-annot-l">{{ a.label }}</span>
                <span v-if="a.color" class="dp-annot-c" :style="{ background: a.color }"></span>
              </div>
            </div>
          </div>

          <div v-else class="dp-split">
            <div v-for="d in screenDocs" :key="d.id" class="dp-split-col" :style="{ width: colWidth(d) + 'px' }">
              <div class="dp-split-head">
                <b class="pn-mono">{{ d.id }}</b>
                <span class="pn-dim">{{ d.name }}</span>
                <span class="pn-spacer"></span>
                <span class="pn-dim pn-mono">{{ d.w }}×{{ d.h }}</span>
                <span v-if="splitFit" class="pn-dim pn-mono">{{ Math.round(colScale(d) * 100) }}%</span>
              </div>
              <div class="dp-split-stage" :style="{ height: colStageH(d) + 'px' }">
                <iframe
                  class="dp-split-frame"
                  :srcdoc="d.doc"
                  sandbox="allow-same-origin"
                  referrerpolicy="no-referrer"
                  :style="{ width: d.w + 'px', height: d.h + 'px', transform: 'scale(' + colScale(d) + ')' }"
                ></iframe>
              </div>
              <span
                class="dp-split-grip" title="拖动调整该栏宽度（双击复位为设计稿宽度）"
                @mousedown="startResize(d, $event)" @dblclick="resetCol(d)"
              ></span>
            </div>
          </div>
        </template>
        <div v-else class="pn-tip dp-stage-empty">
          未找到 HTML 预览。让 Agent 执行 <code>design_export format=html</code> 生成
          （工程 <span class="pn-mono">artifacts.html</span> 指向它，默认 design.html）。
        </div>
      </div>
    </template>
  </PanelShell>
</template>

<script setup>
// DesignPanel — UI 设计工程只读视图（与 Agent 共用同一份真相源：design.project.json + design.tokens.json
// + 产物 design.html + 旁挂 design.verify.json）。
// 设计取舍（与 tool-art / tool-music / tool-voice 一致）：面板不写工作区 —— 节点/令牌编辑一律经 Agent 的
// design_edit（命令式 op），避免「工具写工程 / 面板写工程」两条写路径导致真相源分叉。
//
// 2026-09-18 改版（两件事）：
//  ① 布局：套 PanelShell 统一外壳（顶栏 / 指标条 / 左栏分段 + 主区预览 / 底栏），七域形态一致；
//  ② 预览：整页加「属性标注」层（逐元素标出实际尺寸/字号/颜色，悬停看完整信息）；
//     并排改成「每屏等比缩放、完整显示」+ 适应栏宽/1:1 两档（此前每栏写死 380px 高，屏内容被裁）。
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import api from '../api.js'
import PanelShell from './PanelShell.vue'
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
const tab = ref('screens')

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
const verifyBadge = computed(() => (verify.value ? '校验 ' + passedCount.value + '/' + ((verify.value.checks || []).length) : '未校验'))
const verifyTone = computed(() => (!verify.value ? 'muted' : (passedAll.value ? 'ok' : 'bad')))

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
const splitFit = ref(true)        // 并排：每栏是否等比缩放到栏宽（false = 原始像素 + 横向滚动）
// 整页：属性标注层**默认开** —— 用户诉求就是「整页时看不到各项设计属性」，一点开面板即见；
// 标注是本地叠加（iframe 同源只读），不写工作区，不喜欢可一键关掉。
const annotOn = ref(true)
const annots = ref([])            // 标注项（来自 iframe 内真实 DOM，坐标已乘 scale）

// ── 预览缩放（解决「窄面板里看不到完整设计」）──────────────────
// 导出 HTML 的结构：body padding 24 + 每个 .screen 固定 width/height + 下间距 24；
// 所以内容总宽 = max(屏宽) + 48，总高 = 24 + Σ(屏高 + 24)。
// 直接把它塞进面板（宽度只有几百 px）会被横向裁掉 —— 等比缩放后一眼看全；要看细节用 1:1，要看全貌用「在新窗口打开」。
const stageEl = ref(null)
const frameEl = ref(null)
const stageW = ref(0)
const zoomMode = ref('fit')          // 'fit' 适应宽度 | '1x' 原始像素
const contentSize = computed(() => {
  const ss = screens.value
  if (!ss.length) return { w: 320, h: 200 }
  const w = Math.max(...ss.map((s) => Number(s.w) || 0)) + 48
  const h = 24 + ss.reduce((a, s) => a + (Number(s.h) || 0) + 24, 0)
  return { w: w || 320, h: h || 200 }
})
const scale = computed(() => {
  if (zoomMode.value === '1x') return 1
  const avail = stageW.value || 640
  return Math.min(1, Math.max(0.1, avail / contentSize.value.w))
})
const stageH = computed(() => Math.max(160, Math.round(contentSize.value.h * scale.value)))
const viewHint = computed(() => (viewMode.value === 'page'
  ? '整页：等比缩放看整体；1:1 看像素；属性标注逐元素标尺寸/字号/颜色'
  : '并排：每屏一栏、各自等比缩放完整显示；拖右边缘改栏宽、双击复位'))

// 新标签打开：用 Blob 造 URL（预览 HTML 已自包含，不依赖工作区静态服务，file/工作区路径都无需暴露）
function openPreview() {
  if (!htmlText.value) return
  const url = URL.createObjectURL(new Blob([htmlText.value], { type: 'text/html' }))
  window.open(url, '_blank', 'noopener')
  setTimeout(() => URL.revokeObjectURL(url), 120000)
}
let stageRO = null
function bindStageRO() {
  const el = stageEl.value
  if (!el || typeof ResizeObserver === 'undefined') return
  stageW.value = el.clientWidth
  if (!stageRO) {
    stageRO = new ResizeObserver(() => { stageW.value = stageEl.value ? stageEl.value.clientWidth : 0 })
    stageRO.observe(el)
  }
}
onMounted(bindStageRO)
onBeforeUnmount(() => { if (stageRO) stageRO.disconnect() })
watch(htmlText, async () => { await nextTick(); bindStageRO() })

// ── 属性标注层（整页模式）────────────────────────────────────
// 预览 iframe 只加 sandbox="allow-same-origin"（**不给 allow-scripts** → 预览里的脚本照旧不执行），
// 于是面板能只读地读到 iframe 内的真实 DOM：每个 .nd 元素的实际宽高、字号、文字色/背景色。
// 标出来的就是「浏览器最终算出来的值」，比照设计稿更可信（也正好回答「整页时看不到设计属性」）。
const ACT = { bg: '#111827', fg: '#F9FAFB', dim: '#9CA3AF' }
function rgbToHex(s) {
  const m = /^rgba?\(([^)]+)\)$/.exec(String(s || ''))
  if (!m) return ''
  const p = m[1].split(/[\s,\/]+/).filter((x) => x !== '').map(Number)
  if (p.length < 3 || p.some((n) => !isFinite(n))) return ''
  if (p.length >= 4 && p[3] === 0) return ''
  const h = (n) => Math.round(n).toString(16).padStart(2, '0')
  return '#' + h(p[0]) + h(p[1]) + h(p[2])
}
function hexLum(hex) {
  const s = String(hex).replace('#', '')
  if (s.length !== 6) return 0
  const c = [0, 2, 4].map((i) => parseInt(s.slice(i, i + 2), 16) / 255)
    .map((v) => (v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)))
  return 0.2126 * c[0] + 0.7152 * c[1] + 0.0722 * c[2]
}
function onFrameLoad() { buildAnnots() }
function buildAnnots() {
  if (!annotOn.value) { annots.value = []; return }
  const f = frameEl.value
  const doc = f && f.contentDocument
  if (!doc || !doc.body) { annots.value = []; return }
  const win = doc.defaultView
  const out = []
  const dense = scale.value >= 0.9        // 1:1 附近才连纯布局容器一起标（缩放小时只标"看得见"的东西）
  doc.body.querySelectorAll('.nd').forEach((el) => {
    const r = el.getBoundingClientRect()
    const w = Math.round(r.width)
    const h = Math.round(r.height)
    if (w < 8 || h < 8) return                             // 细线 / 小点不标（否则糊成一片）
    const cs = win.getComputedStyle(el)
    const hasText = Array.from(el.childNodes).some((n) => n.nodeType === 3 && n.textContent.trim())
    const hasSvg = !!el.querySelector('svg')
    const fs = parseFloat(cs.fontSize) || 0
    const fg = hasText ? rgbToHex(cs.color) : ''
    const bg = rgbToHex(cs.backgroundColor)
    const hasBorder = (parseFloat(cs.borderTopWidth) || 0) > 0
    // 整屏背景层不标（它只是底，标了只挡内容）；纯布局撑开层（无文字/无图标/无底色/无边框）默认也不标 ——
    // 972 个全标会把设计稿糊住。放大到 1:1 附近再连容器一起标，兼顾「看整体」与「量到每一层」。
    const scr = el.closest('.screen')
    if (scr) {
      const sr = scr.getBoundingClientRect()
      if (w >= sr.width * 0.94 && h >= sr.height * 0.94) return
    }
    if (!hasText && !hasSvg && !bg && !hasBorder && !dense) return
    const parts = [w + '×' + h]
    if (hasText && fs) parts.push(Math.round(fs) + 'px')
    if (fs && (parseFloat(cs.paddingTop) || parseFloat(cs.paddingLeft))) {
      parts.push('pad ' + Math.round(parseFloat(cs.paddingTop) || 0) + '/' + Math.round(parseFloat(cs.paddingLeft) || 0))
    }
    const tip = [
      '尺寸 ' + w + '×' + h,
      hasText ? '字号 ' + Math.round(fs) + 'px' : '',
      fg ? '文字色 ' + fg : '',
      bg ? '背景 ' + bg : '',
      '位置 (' + Math.round(r.left) + ', ' + Math.round(r.top) + ')',
    ].filter(Boolean).join('\n')
    out.push({
      isText: hasText,
      label: parts.join(' · '),
      color: bg || fg,
      title: tip,
      style: {
        left: (r.left * scale.value) + 'px',
        top: (r.top * scale.value) + 'px',
        // 标签底色按该元素自身颜色取反，压在浅底深底上都看得清
        background: (hexLum(bg || fg || '#FFFFFF') > 0.55 ? ACT.bg : ACT.fg),
        color: (hexLum(bg || fg || '#FFFFFF') > 0.55 ? ACT.fg : ACT.bg),
        fontSize: Math.max(8, Math.min(12, Math.round(10 * Math.min(1.4, Math.max(0.8, scale.value + 0.3))))) + 'px',
      },
    })
  })
  annots.value = out
}
watch(annotOn, async (v) => {
  if (!v) { annots.value = []; return }
  await nextTick()
  buildAnnots()
})
// 缩放变化后标注层跟着重排（位置要重乘 scale）
watch(scale, () => { if (annotOn.value) buildAnnots() })

// 点击左栏屏幕 → 整页模式滚动到该屏
function gotoScreen(id) {
  const el = stageEl.value
  if (!el) return
  let top = 24
  for (const s of screens.value) {
    if (s.id === id) break
    top += (Number(s.h) || 0) + 24
  }
  const target = el.querySelector('.dp-stage')
  const scroller = target || el
  scroller.scrollTop = Math.max(0, top * scale.value - 8)
}

const colWidths = ref({})          // { 屏幕 id: 像素宽 } —— 并排视图每栏宽度（默认 = 设计稿宽度，可拖拽）
function colWidth(d) { return colWidths.value[d.id] || (Number(d.w) || 320) }
function colScale(d) {
  if (!splitFit.value) return 1
  const avail = Math.max(80, colWidth(d))
  return Math.min(1, avail / (Number(d.w) || 1))
}
function colStageH(d) { return Math.max(80, Math.round((Number(d.h) || 100) * colScale(d))) }
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

// ── 指标条 ──
const metrics = computed(() => [
  { k: '屏幕', v: String(screens.value.length) },
  { k: '节点', v: String(screens.value.reduce((a, s) => a + s.nodes, 0)) },
  { k: '导航', v: ((proj.value && proj.value.flow) || []).length + ' 条' },
  { k: '令牌', v: tokenCount.value + ' 项' },
  { k: '预览', v: htmlReady.value ? screenDocs.value.length + ' 屏' : '未导出' },
  { k: '变更', v: updatedAt.value },
])
const tabs = computed(() => [
  { id: 'screens', name: '屏幕', count: screens.value.length, title: '屏幕清单（点击滚动到该屏）与导航' },
  { id: 'tokens', name: '令牌', count: tokenCount.value, title: '设计令牌（色板 / 字号 / 间距）' },
  { id: 'checks', name: '校验', count: verify.value ? passedCount.value + '/' + (verify.value.checks || []).length : 0, title: '11 项判据自检' },
  { id: 'code', name: '代码', count: codegen.value ? (codegen.value.files || []).length : 0, title: 'design_codegen 落地产物状态' },
])

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
    annots.value = []
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
    annots.value = []
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
/* 局部样式（其余复用 PanelShell 共享类；配色取设计系统变量） */
.dp-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: none;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border-color);
  flex-wrap: wrap;
}
.dp-stage-wrap { flex: 1; min-height: 0; display: flex; flex-direction: column; padding: 10px; }
.dp-stage-empty { margin: auto; }

/* 预览舞台：内容按设计稿原尺寸渲染，再整体 transform: scale 缩放（宽高由脚本按屏尺寸算） */
.dp-stage {
  position: relative;
  overflow: auto;
  max-height: 100%;
  background: #E5E7EB;
  border: 1px solid var(--border-color);
  border-radius: 4px;
}
.dp-frame {
  position: absolute; top: 0; left: 0; display: block; background: #fff;
  border: 0; transform-origin: 0 0;
}

/* 属性标注层：覆盖在预览之上，逐元素标出实际尺寸/字号/颜色 */
.dp-annot-layer { position: absolute; inset: 0; pointer-events: none; }
.dp-annot {
  position: absolute;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 0 3px;
  border-radius: 2px;
  line-height: 1.5;
  white-space: nowrap;
  opacity: 0.92;
  pointer-events: auto;
  font-family: ui-monospace, Consolas, monospace;
}
.dp-annot:hover { opacity: 1; outline: 1px dashed currentColor; }
.dp-annot-l { letter-spacing: -0.2px; }
.dp-annot-c {
  width: 6px; height: 6px; border-radius: 1px; flex: none;
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.35);
}
.dp-annot-txt { font-weight: 600; }

/* 并排：每栏等比缩放，完整显示整屏 */
.dp-split { display: flex; gap: 12px; overflow-x: auto; overflow-y: hidden; padding: 2px 2px 8px; height: 100%; }
.dp-split-col { flex: 0 0 auto; position: relative; display: flex; flex-direction: column; min-height: 0; }
.dp-split-head { display: flex; align-items: baseline; gap: 6px; margin-bottom: 4px; font-size: 11px; flex: none; }
.dp-split-stage {
  position: relative;
  overflow: hidden;
  background: #E5E7EB;
  border: 1px solid var(--border-color);
  border-radius: 4px;
}
.dp-split-frame {
  position: absolute; top: 0; left: 0; display: block;
  border: 0; transform-origin: 0 0;
}
.dp-split-grip {
  position: absolute; top: 18px; right: -8px; width: 8px; height: calc(100% - 18px);
  cursor: col-resize; border-radius: 2px;
}
.dp-split-grip:hover { background: var(--accent); opacity: .5; }

.dp-side-hint { margin-top: 6px; }
.dp-side-block { margin-top: 10px; display: flex; flex-direction: column; gap: 4px; }
.dp-wrap { white-space: normal; overflow-wrap: anywhere; }
.pn-warn { color: var(--text-primary); }
</style>
