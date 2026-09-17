<template>
  <PanelShell
    v-model:file="projectPath"
    title="人声工程"
    icon="sparkles"
    placeholder="voice.project.json"
    :badge="checksBadge"
    :badge-tone="checksTone"
    :metrics="metrics"
    :loading="loading"
    :error="proj ? error : ''"
    :empty="!proj"
    reload-title="重新载入工程与旁挂文件"
    footnote="只读面板：编辑一律经 Agent 的 voice_edit（命令式 op）追加到 edits 链；面板只做展示与试算结果呈现。"
    :source="projectPath"
    @reload="load"
  >
    <template #empty>
      <div class="pn-empty-title">还没有人声工程</div>
      <div class="pn-tip">
        面板读工作区根目录的 <code>voice.project.json</code>（与 Agent 共用同一份真相源），当前没有这个文件。
      </div>
      <div class="pn-card">
        <div class="pn-card-title">三步开始</div>
        <div class="pn-empty-step"><i>1</i><span>让 Agent 执行 <code>voice_import</code> 导入素材（自动算波形峰值、F0、音符段、切片）。</span></div>
        <div class="pn-empty-step"><i>2</i><span>用 <code>voice_edit</code> 追加编辑命令（剪辑 / 变调 / 响度 …）到 edits 链。</span></div>
        <div class="pn-empty-step"><i>3</i><span><code>voice_render</code> 出成品与 sourceMap；工程自检结果会显示在顶栏徽章。</span></div>
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

    <!-- 左栏：音符段（含切片）/ 编辑链 / 产物 / 自检 -->
    <template #side>
      <div v-if="tab === 'notes'" class="pn-list">
        <div v-for="(n, i) in notes" :key="i" class="vp-note">
          <div class="vp-note-h">
            <span class="pn-mono pn-mini">{{ n.t0.toFixed(2) }}–{{ n.t1.toFixed(2) }}</span>
            <span class="pn-spacer"></span>
            <b class="pn-mini">{{ n.f0 }} Hz</b>
          </div>
          <div class="pn-tip pn-mini">MIDI <span class="pn-mono">{{ n.midiFloat }}</span> · conf <span class="pn-mono">{{ n.conf }}</span></div>
        </div>
        <div v-if="!notes.length" class="pn-tip">还没有音符段（F0 分析未产出，或整段都是清音）。</div>
        <div v-if="slices.length" class="mp-side-block">
          <div class="pn-card-title">切片 {{ slices.length }} 段<span class="pn-dim"> / 候选 {{ analysis.sliceCandidates || 0 }}</span></div>
          <div class="vp-chips">
            <span v-for="(s, i) in slices" :key="i" class="vp-chip pn-mono">{{ s.a }}–{{ s.b }}</span>
          </div>
        </div>
      </div>

      <div v-else-if="tab === 'edits'" class="pn-list">
        <template v-if="edits.length">
          <div v-for="(o, i) in edits" :key="i" class="vp-op">
            <span class="rp-idx">{{ i + 1 }}</span>
            <code class="pn-mini">{{ o.op }}</code>
            <span class="pn-dim pn-mini mp-wrap"> {{ paramsOf(o) }}</span>
          </div>
        </template>
        <div v-else class="pn-tip">（空）—— 让 Agent 执行 <code>voice_edit</code> 追加编辑命令。</div>
        <div v-if="recipes.length" class="mp-side-block">
          <div class="pn-card-title">渲染配方</div>
          <div class="pn-tip pn-mono">{{ recipes.map((r) => r.algo + '@' + r.version).join(' → ') }}</div>
        </div>
        <div class="pn-tip mp-side-hint">源 + edits 是纯函数关系：同样的 edits 链必然渲染出同一个结果。</div>
      </div>

      <div v-else-if="tab === 'out'" class="pn-list">
        <template v-if="render.out">
          <div class="pn-kv"><span>输出</span><b class="pn-mono mp-wrap">{{ render.out }}</b></div>
          <div class="pn-kv"><span>时长 / 位深</span><b>{{ render.seconds }}s · {{ render.bitDepth }}bit</b></div>
          <div class="pn-kv"><span>sourceMap</span><b class="pn-mono mp-wrap">{{ render.sourceMap }}</b></div>
          <div class="pn-kv"><span>recipe</span><b class="pn-mono mp-wrap">{{ render.recipe }}</b></div>
          <div v-if="analysis.loudness" class="pn-kv">
            <span>响度</span>
            <b>{{ analysis.loudness.lufs }} LUFS · {{ analysis.loudness.truePeak }} dBTP<span class="pn-dim">（{{ analysis.loudness.standard }}）</span></b>
          </div>
        </template>
        <div v-else class="pn-tip">
          还没有渲染产物。让 Agent 执行 <code>voice_render</code> 出成品（会同时写 sourceMap 与 recipe）。
        </div>
      </div>

      <div v-else class="pn-list">
        <template v-if="checks.length">
          <div v-for="c in checks" :key="c.id" class="pn-check">
            <span class="pn-dot" :class="{ 'pn-dot-ok': c.pass }"></span>
            <b>{{ c.id }}</b>
            <span class="pn-check-name">{{ c.name }}</span>
            <span class="pn-check-metric" :title="c.metric">{{ c.metric }}</span>
          </div>
          <div class="pn-tip mp-side-hint">六项自检随工程内嵌（不需要单独的 verify 文件）。</div>
        </template>
        <div v-else class="pn-tip">工程里还没有自检结果。</div>
      </div>
    </template>

    <!-- 主区：源素材 + 波形 + F0 -->
    <template #main>
      <div class="pn-scroll vp-main">
        <div class="pn-row-gap">
          <span class="pn-strong">{{ source ? source.path : '（无源素材）' }}</span>
          <span class="pn-spacer"></span>
          <span class="pn-dim pn-mini pn-mono">{{ shortHash }}</span>
        </div>

        <div v-if="source" class="pn-card">
          <div class="pn-card-title">源素材</div>
          <div class="pn-kv">
            <span>格式</span>
            <b>{{ source.seconds }}s · {{ source.fs }} Hz · {{ source.channels }}ch · {{ source.bitDepth }}bit</b>
          </div>
          <div class="pn-kv"><span>峰值 / RMS / DC</span><b class="pn-mono">{{ source.peak }} / {{ source.rms }} / {{ source.dc }}</b></div>
          <div class="pn-kv"><span>sha256</span><b class="pn-mono mp-wrap">{{ shortHash }}</b></div>
        </div>

        <div v-if="wave" class="vp-block">
          <div class="pn-card-title">波形 <span class="pn-dim">（{{ wave.peaks.length }} 帧 / {{ wave.frameMs }} ms）</span></div>
          <canvas ref="waveCanvas" class="vp-canvas" height="76"></canvas>
        </div>

        <div v-if="f0Values.length" class="vp-block">
          <div class="pn-card-title">
            F0 曲线 / 音符段
            <span class="pn-dim">（{{ notes.length }} 段 · hop {{ f0Meta.hop }} · 浊音比 {{ pct(f0Meta.voicedRatio) }}）</span>
          </div>
          <canvas ref="f0Canvas" class="vp-canvas" height="118"></canvas>
          <div class="pn-tip">浅色竖带 = 识别到的音符段，折线 = 逐帧 F0；两者叠加便于核对切段是否贴合音高变化。</div>
        </div>

        <div v-if="!wave && !f0Values.length" class="pn-tip">
          还没有波形/F0 旁挂数据。让 Agent 执行 <code>voice_import</code>（会写峰值缓存与 F0 文件）。
        </div>
      </div>
    </template>
  </PanelShell>
</template>

<script setup>
// VoicePanel — 人声工程只读视图（与 Agent 共用同一份真相源：工程文件 + 旁挂文件）。
// 设计取舍：面板不做写操作（编辑一律经 Agent 工具或后续的拖拽命令模型），
// 因此不会与 tool-voice 的「源 + edits 纯函数渲染」契约产生第二条写路径。
// 布局（2026-09-18 改版）：套 PanelShell 统一外壳 —— 顶栏 / 指标条 / 左栏分段（音符段·编辑链·产物·自检）+ 主区波形 / 底栏。
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import api from '../api.js'
import PanelShell from './PanelShell.vue'
import { state } from '../ui-state.js'

const projectPath = ref('voice.project.json')
const loading = ref(false)
const error = ref('')
const proj = ref(null)
const wave = ref(null)
const f0Values = ref([])
const recipeFile = ref(null)
const waveCanvas = ref(null)
const f0Canvas = ref(null)
const tab = ref('notes')

const source = computed(() => (proj.value && proj.value.sources && proj.value.sources[0]) || null)
const analysis = computed(() => {
  if (!proj.value || !source.value) return {}
  return (proj.value.analysis && proj.value.analysis[source.value.id]) || {}
})
const f0Meta = computed(() => analysis.value.f0 || {})
const notes = computed(() => analysis.value.notes || [])
const slices = computed(() => analysis.value.slices || [])
const edits = computed(() => (proj.value && proj.value.edits) || [])
const render = computed(() => (proj.value && proj.value.render) || {})
const recipes = computed(() => (recipeFile.value && recipeFile.value.recipes) || [])
const checks = computed(() => {
  const c = proj.value && proj.value.checks
  return (c && c.checks) || []
})
const passed = computed(() => checks.value.filter((c) => c.pass).length)
const total = computed(() => checks.value.length)
const checksPass = computed(() => total.value > 0 && passed.value === total.value)
const checksBadge = computed(() => (total.value ? '自检 ' + passed.value + '/' + total.value : '未自检'))
const checksTone = computed(() => (!total.value ? 'muted' : (checksPass.value ? 'ok' : 'bad')))
const shortHash = computed(() => {
  const h = source.value && source.value.sha256
  return h ? h.slice(0, 16) + '…' : '—'
})

// ── 指标条 ──
const metrics = computed(() => {
  const out = []
  if (source.value) {
    out.push({ k: '时长', v: source.value.seconds + 's' })
    out.push({ k: '采样', v: source.value.fs + ' Hz' })
    out.push({ k: '声道', v: source.value.channels + 'ch · ' + source.value.bitDepth + 'bit' })
    out.push({ k: '峰值/RMS', v: source.value.peak + ' / ' + source.value.rms, mono: true })
  }
  out.push({ k: '音符段', v: String(notes.value.length) })
  if (slices.value.length) out.push({ k: '切片', v: String(slices.value.length) })
  if (analysis.value.loudness) out.push({ k: '响度', v: analysis.value.loudness.lufs + ' LUFS' })
  if (edits.value.length) out.push({ k: '编辑', v: edits.value.length + ' 步' })
  return out
})
const tabs = computed(() => [
  { id: 'notes', name: '音符段', count: notes.value.length, title: 'F0 逐段结果（起止 / 频率 / MIDI / 置信度）与切片' },
  { id: 'edits', name: '编辑链', count: edits.value.length, title: 'voice_edit 追加的编辑命令与渲染配方' },
  { id: 'out', name: '产物', count: render.value.out ? 1 : 0, title: 'voice_render 的成品与 sourceMap' },
  { id: 'checks', name: '自检', count: total.value ? passed.value + '/' + total.value : 0, title: '工程内嵌的六项自检' },
])

function pct(v) {
  return typeof v === 'number' ? (v * 100).toFixed(1) + '%' : '—'
}

function paramsOf(op) {
  const skip = new Set(['op'])
  return Object.keys(op || {})
    .filter((k) => !skip.has(k))
    .map((k) => k + '=' + JSON.stringify(op[k]))
    .join(' ')
}

function absPath(rel) {
  const root = String((state && state.workspaceRoot) || '').replace(/[\\/]+$/, '')
  if (!root) return rel
  return root + '/' + String(rel).replace(/^[\\/]+/, '')
}

async function readJSON(rel) {
  // ★ 同上：apiURL 已加 /api 前缀，写成 '/api/fs/read' 会 404（tool-voice 面板此前一直读不到文件）。
  const r = await api.apiGet('/fs/read', { path: absPath(rel) })
  return JSON.parse(r.content)
}

async function load() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    const p = await readJSON(projectPath.value)
    proj.value = p
    wave.value = null
    f0Values.value = []
    recipeFile.value = null
    const src = (p.sources || [])[0]
    if (src && src.wave) {
      try { wave.value = await readJSON(src.wave) } catch (_) { /* 缓存可选 */ }
    }
    const an = src && p.analysis ? p.analysis[src.id] : null
    if (an && an.f0 && an.f0.file) {
      try {
        const arr = await readJSON(an.f0.file)
        f0Values.value = Array.isArray(arr) ? arr : []
      } catch (_) { /* 旁挂可选 */ }
    }
    if (p.render && p.render.recipe) {
      try { recipeFile.value = await readJSON(p.render.recipe) } catch (_) { /* 可选 */ }
    }
    await nextTick()
    drawWave()
    drawF0()
  } catch (e) {
    proj.value = null
    error.value = '载入失败：' + ((e && e.message) || e)
  } finally {
    loading.value = false
  }
}

// 颜色一律取自设计系统变量（避免另起一套配色）
function cssVar(el, name, fallback) {
  try {
    const v = getComputedStyle(el).getPropertyValue(name).trim()
    return v || fallback
  } catch (_) {
    return fallback
  }
}

function prepareCanvas(cv, h) {
  const dpr = window.devicePixelRatio || 1
  const w = cv.clientWidth || 300
  cv.width = Math.max(1, Math.round(w * dpr))
  cv.height = Math.round(h * dpr)
  const g = cv.getContext('2d')
  g.setTransform(dpr, 0, 0, dpr, 0, 0)
  g.clearRect(0, 0, w, h)
  return { g, w, h }
}

function drawWave() {
  const cv = waveCanvas.value
  if (!cv || !wave.value || !wave.value.peaks) return
  const { g, w, h } = prepareCanvas(cv, 76)
  const peaks = wave.value.peaks
  const mid = h / 2
  const accent = cssVar(cv, '--accent', '#4b8bf5')
  const bw = Math.max(1, w / Math.max(1, peaks.length))
  g.globalAlpha = 0.85
  g.fillStyle = accent
  for (let i = 0; i < peaks.length; i++) {
    const a = Math.min(1, peaks[i]) * (mid - 3)
    g.fillRect(i * bw, mid - a, Math.max(1, bw * 0.8), a * 2)
  }
  g.globalAlpha = 1
  g.strokeStyle = cssVar(cv, '--border-color', 'rgba(128,128,128,0.3)')
  g.beginPath()
  g.moveTo(0, mid)
  g.lineTo(w, mid)
  g.stroke()
}

function drawF0() {
  const cv = f0Canvas.value
  if (!cv) return
  const { g, w, h } = prepareCanvas(cv, 118)
  const arr = f0Values.value
  if (!arr.length) return
  const fs = Number(f0Meta.value.fs) || (source.value && source.value.fs) || 48000
  const hop = Number(f0Meta.value.hop) || 512
  const pts = []
  for (let i = 0; i < arr.length; i++) {
    const f = arr[i]
    if (typeof f === 'number' && Number.isFinite(f) && f > 0) pts.push({ t: (i * hop) / fs, f })
  }
  if (!pts.length) return
  const pad = 10
  const tMax = Math.max(pts[pts.length - 1].t, 0.001)
  const fMin = Math.min.apply(null, pts.map((p) => p.f))
  const fMax = Math.max.apply(null, pts.map((p) => p.f))
  const span = Math.max(1e-6, fMax - fMin)
  const X = (t) => pad + (t / tMax) * (w - 2 * pad)
  const Y = (f) => h - pad - ((f - fMin) / span) * (h - 2 * pad)
  const accent = cssVar(cv, '--accent', '#4b8bf5')

  // 音符段背景（半透明，用 globalAlpha 而非拼色值）
  g.globalAlpha = 0.14
  g.fillStyle = accent
  for (const n of notes.value) {
    g.fillRect(X(n.t0), pad, Math.max(1, X(n.t1) - X(n.t0)), h - 2 * pad)
  }
  g.globalAlpha = 1

  // F0 折线
  g.strokeStyle = accent
  g.lineWidth = 1.2
  g.beginPath()
  let started = false
  for (const p of pts) {
    const x = X(p.t)
    const y = Y(p.f)
    if (!started) { g.moveTo(x, y); started = true } else g.lineTo(x, y)
  }
  g.stroke()

  // 频率刻度
  g.fillStyle = cssVar(cv, '--text-muted', '#888')
  g.font = '10px system-ui, sans-serif'
  g.fillText(fMax.toFixed(1) + ' Hz', 4, 11)
  g.fillText(fMin.toFixed(1) + ' Hz', 4, h - 4)
}

function redraw() {
  drawWave()
  drawF0()
}

onMounted(() => {
  load()
  window.addEventListener('resize', redraw)
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', redraw)
})
</script>

<style scoped>
/* 局部样式（其余复用 PanelShell 共享类；配色取设计系统变量） */
.vp-main { display: flex; flex-direction: column; gap: 10px; padding: 10px; }
.vp-block { display: flex; flex-direction: column; gap: 4px; }
.vp-canvas { display: block; width: 100%; background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 4px; }
.vp-note { display: flex; flex-direction: column; gap: 1px; padding: 4px 6px; border-radius: 4px; }
.vp-note:hover { background: var(--bg-hover); }
.vp-note-h { display: flex; align-items: baseline; gap: 6px; }
.vp-op { display: flex; gap: 6px; align-items: baseline; padding: 3px 2px; line-height: 1.6; }
.vp-chips { display: flex; flex-wrap: wrap; gap: 4px; }
.vp-chip {
  padding: 0 5px; font-size: 10px; color: var(--text-muted);
  border: 1px solid var(--border-color); border-radius: 3px;
}
.rp-idx {
  display: inline-block; min-width: 16px; text-align: right;
  color: var(--text-muted); font-variant-numeric: tabular-nums; font-size: 11px; flex: none;
}
.mp-wrap { white-space: normal; overflow-wrap: anywhere; text-align: left; }
.mp-side-hint { margin-top: 6px; }
.mp-side-block { margin-top: 10px; display: flex; flex-direction: column; gap: 4px; }
</style>
