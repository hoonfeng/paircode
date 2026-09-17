<template>
  <div class="voice-panel">
    <div class="vp-bar">
      <span class="vp-title">人声工程</span>
      <input v-model="projectPath" class="vp-input" placeholder="voice.project.json" @keyup.enter="load" />
      <button class="vp-icon-btn" :disabled="loading" title="重新载入工程与旁挂文件" @click="load">
        <SvgIcon name="refresh" :size="12" :class="{ spinning: loading }" />
      </button>
    </div>

    <div v-if="error" class="vp-msg vp-err">{{ error }}</div>
    <div v-else-if="!proj" class="vp-msg">
      未找到人声工程。先让 Agent 执行 <code>voice_import</code> 导入素材（生成 voice.project.json）。
    </div>

    <template v-else>
      <!-- 源素材 -->
      <section v-if="source" class="vp-sec">
        <div class="vp-sec-title">源素材</div>
        <div class="vp-kv"><span>文件</span><b>{{ source.path }}</b></div>
        <div class="vp-kv">
          <span>格式</span>
          <b>{{ source.seconds }}s · {{ source.fs }} Hz · {{ source.channels }}ch · {{ source.bitDepth }}bit</b>
        </div>
        <div class="vp-kv">
          <span>峰值/RMS/DC</span>
          <b>{{ source.peak }} / {{ source.rms }} / {{ source.dc }}</b>
        </div>
        <div class="vp-kv"><span>sha256</span><b class="vp-mono">{{ shortHash }}</b></div>
      </section>

      <!-- 波形（峰值缓存，10 ms/帧） -->
      <section v-if="wave" class="vp-sec">
        <div class="vp-sec-title">波形 <span class="vp-dim">（{{ wave.peaks.length }} 帧 / {{ wave.frameMs }} ms）</span></div>
        <canvas ref="waveCanvas" class="vp-canvas" height="76"></canvas>
      </section>

      <!-- F0 曲线 + 音符段 -->
      <section v-if="f0Values.length" class="vp-sec">
        <div class="vp-sec-title">
          F0 曲线 / 音符段
          <span class="vp-dim">（{{ notes.length }} 段 · hop {{ f0Meta.hop }} · 浊音比 {{ pct(f0Meta.voicedRatio) }}）</span>
        </div>
        <canvas ref="f0Canvas" class="vp-canvas" height="118"></canvas>
      </section>

      <!-- 音符段表 -->
      <section v-if="notes.length" class="vp-sec">
        <div class="vp-sec-title">音符段</div>
        <table class="vp-table">
          <thead><tr><th>起</th><th>止</th><th>F0</th><th>MIDI</th><th>conf</th></tr></thead>
          <tbody>
            <tr v-for="(n, i) in notes" :key="i">
              <td>{{ n.t0.toFixed(3) }}</td>
              <td>{{ n.t1.toFixed(3) }}</td>
              <td>{{ n.f0 }}</td>
              <td>{{ n.midiFloat }}</td>
              <td>{{ n.conf }}</td>
            </tr>
          </tbody>
        </table>
      </section>

      <!-- 切片 -->
      <section v-if="slices.length" class="vp-sec">
        <div class="vp-sec-title">
          切片 <span class="vp-dim">（{{ slices.length }} 段 / 候选 {{ analysis.sliceCandidates || 0 }}）</span>
        </div>
        <div class="vp-chips">
          <span v-for="(s, i) in slices" :key="i" class="vp-chip">{{ s.a }}–{{ s.b }}</span>
        </div>
      </section>

      <!-- 响度 -->
      <section v-if="analysis.loudness" class="vp-sec">
        <div class="vp-sec-title">响度 <span class="vp-dim">（{{ analysis.loudness.standard }}）</span></div>
        <div class="vp-kv"><span>LUFS / true peak</span><b>{{ analysis.loudness.lufs }} / {{ analysis.loudness.truePeak }} dBTP</b></div>
      </section>

      <!-- 编辑链 -->
      <section class="vp-sec">
        <div class="vp-sec-title">编辑链 <span class="vp-dim">（{{ edits.length }} 步）</span></div>
        <div v-if="!edits.length" class="vp-dim">（空 —— 让 Agent 执行 voice_edit 追加编辑命令）</div>
        <ol v-else class="vp-ops">
          <li v-for="(o, i) in edits" :key="i">
            <code>{{ o.op }}</code><span class="vp-dim"> {{ paramsOf(o) }}</span>
          </li>
        </ol>
        <div v-if="recipes.length" class="vp-dim vp-mono">recipe: {{ recipes.map(r => r.algo + '@' + r.version).join(' → ') }}</div>
      </section>

      <!-- 渲染产物 -->
      <section v-if="render.out" class="vp-sec">
        <div class="vp-sec-title">渲染产物</div>
        <div class="vp-kv"><span>输出</span><b>{{ render.out }}（{{ render.seconds }}s · {{ render.bitDepth }}bit）</b></div>
        <div class="vp-kv"><span>sourceMap</span><b class="vp-mono">{{ render.sourceMap }}</b></div>
        <div class="vp-kv"><span>recipe</span><b class="vp-mono">{{ render.recipe }}</b></div>
      </section>

      <!-- 六项自检 -->
      <section v-if="checks.length" class="vp-sec">
        <div class="vp-sec-title">
          六项自检
          <span :class="['vp-badge', checksPass ? 'vp-ok' : 'vp-bad']">{{ passed }}/{{ total }}</span>
        </div>
        <div v-for="c in checks" :key="c.id" class="vp-check">
          <span :class="['vp-dot', c.pass ? 'vp-ok' : 'vp-bad']"></span>
          <b>{{ c.id }}</b>
          <span class="vp-check-name">{{ c.name }}</span>
          <span class="vp-dim vp-check-metric">{{ c.metric }}</span>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup>
// VoicePanel — 人声工程只读视图（与 Agent 共用同一份真相源：工程文件 + 旁挂文件）。
// 设计取舍：面板不做写操作（编辑一律经 Agent 工具或后续的拖拽命令模型），
// 因此不会与 tool-voice 的「源 + edits 纯函数渲染」契约产生第二条写路径。
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'
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
const shortHash = computed(() => {
  const h = source.value && source.value.sha256
  return h ? h.slice(0, 16) + '…' : '—'
})

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
.voice-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 10px;
  font-size: 12px;
  color: var(--text-primary);
  overflow-y: auto;
  height: 100%;
}
.vp-bar { display: flex; align-items: center; gap: 6px; }
.vp-title { font-weight: 600; color: var(--text-secondary); }
.vp-input {
  flex: 1;
  min-width: 0;
  padding: 3px 6px;
  font-size: 11px;
  color: var(--text-primary);
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  border-radius: 4px;
}
.vp-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
  color: var(--text-secondary);
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  cursor: pointer;
}
.vp-icon-btn:hover { background: var(--bg-hover); }
.spinning { animation: vp-spin 1s linear infinite; }
@keyframes vp-spin { to { transform: rotate(360deg); } }

.vp-msg { padding: 8px; color: var(--text-muted); line-height: 1.6; }
.vp-err { color: var(--text-primary); border-left: 2px solid var(--accent); background: var(--bg-tertiary); }
.vp-sec { display: flex; flex-direction: column; gap: 4px; padding: 8px; background: var(--bg-tertiary); border-radius: 6px; }
.vp-sec-title { font-weight: 600; color: var(--text-secondary); margin-bottom: 2px; }
.vp-kv { display: flex; gap: 8px; justify-content: space-between; }
.vp-kv > span { color: var(--text-muted); flex: none; }
.vp-kv > b { font-weight: 500; text-align: right; word-break: break-all; }
.vp-mono { font-family: ui-monospace, Consolas, monospace; font-size: 11px; color: var(--text-secondary); }
.vp-dim { color: var(--text-muted); font-weight: 400; }
.vp-canvas { width: 100%; display: block; background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 4px; }
.vp-table { width: 100%; border-collapse: collapse; font-size: 11px; }
.vp-table th { color: var(--text-muted); font-weight: 500; text-align: left; padding: 2px 4px; border-bottom: 1px solid var(--border-color); }
.vp-table td { padding: 2px 4px; border-bottom: 1px solid var(--bg-hover); }
.vp-chips { display: flex; flex-wrap: wrap; gap: 4px; }
.vp-chip { padding: 1px 6px; font-size: 11px; color: var(--text-secondary); background: var(--bg-hover); border-radius: 10px; }
.vp-ops { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 2px; }
.vp-ops code { color: var(--accent); font-size: 11px; }
.vp-badge { padding: 1px 6px; border-radius: 8px; font-size: 11px; }
.vp-dot { width: 7px; height: 7px; border-radius: 50%; flex: none; display: inline-block; }
.vp-ok { background: var(--accent); color: var(--bg-primary); }
.vp-bad { background: var(--text-muted); color: var(--bg-primary); }
.vp-check { display: flex; align-items: center; gap: 6px; }
.vp-check > b { flex: none; color: var(--text-secondary); }
.vp-check-name { flex: none; }
.vp-check-metric { flex: 1; text-align: right; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
