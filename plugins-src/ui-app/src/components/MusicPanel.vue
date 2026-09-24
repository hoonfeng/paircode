<template>
  <PanelShell
    v-model:file="projectPath"
    title="音乐工程"
    icon="list"
    placeholder="music.project.json"
    :badge="verifyBadge"
    :badge-tone="verifyTone"
    :metrics="metrics"
    :loading="loading"
    :error="proj ? error : ''"
    :empty="!proj"
    reload-title="重新载入工程与校验报告"
    footnote="只读面板：音符编辑一律经 Agent 的 music_edit（命令式 op）；「播放」用原生 Web Audio 试听，不改动工程数据。"
    :source="projectPath"
    @reload="load"
  >
    <template #head-actions>
      <button class="pn-btn" :disabled="!noteCount" :title="'用原生 Web Audio 试听（不写工作区）'" @click="playing ? stopPlay() : play()">
        {{ playing ? '停止' : '播放' }}
      </button>
    </template>

    <template #empty>
      <div class="pn-empty-title">还没有音乐工程</div>
      <div class="pn-tip">
        面板读工作区根目录的 <code>music.project.json</code>（与 Agent 共用同一份真相源），当前没有这个文件，所以没有可看的内容。
      </div>
      <div class="pn-card">
        <div class="pn-card-title">三步开始</div>
        <div class="pn-empty-step"><i>1</i><span>让 Agent 执行 <code>music_project</code> 创建工程（速度 / 拍号 / 调号 / 轨道）。</span></div>
        <div class="pn-empty-step"><i>2</i><span>用 <code>music_edit</code> 写音符（note.add / track.add …）。</span></div>
        <div class="pn-empty-step"><i>3</i><span><code>music_export</code> 出产物（MIDI / 谱面），<code>music_verify</code> 生成校验报告。</span></div>
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

    <!-- 左栏：轨道 / 产物 / 校验 -->
    <template #side>
      <div v-if="tab === 'tracks'" class="pn-list">
        <div v-for="tr in trackRows" :key="tr.id" class="mp-track">
          <div class="mp-track-h">
            <span class="pn-item-k">{{ tr.id }}</span>
            <span class="pn-dim pn-mini">{{ tr.name }}</span>
            <span class="pn-spacer"></span>
            <span class="pn-dim pn-mini">{{ tr.count }} 音符</span>
          </div>
          <div class="pn-tip pn-mini mp-track-meta">
            通道 <span class="pn-mono">{{ tr.channel }}</span> · 音色 <span class="pn-mono">{{ tr.program }}</span> · 音域 <span class="pn-mono">{{ tr.range }}</span>
          </div>
        </div>
        <div v-if="!trackRows.length" class="pn-tip">工程里还没有轨道。</div>
        <div v-else class="pn-tip mp-side-hint">音色为 MIDI program 号（0 起）；音域按实际音符算。</div>
      </div>

      <div v-else-if="tab === 'out'" class="pn-list">
        <div v-for="a in artifacts" :key="a.kind" class="pn-kv">
          <span>{{ a.kind }}</span>
          <b class="pn-mono mp-wrap">{{ a.path }}</b>
        </div>
        <div v-if="!artifacts.length" class="pn-tip">
          还没有产物。让 Agent 执行 <code>music_export</code>（默认出 MIDI / 谱面等）。
        </div>
        <div v-else class="pn-tip mp-side-hint">产物由 <code>music_export</code> 生成。</div>
      </div>

      <div v-else class="pn-list">
        <template v-if="verify">
          <div v-for="c in verify.checks || []" :key="c.id" class="pn-check">
            <span class="pn-dot" :class="{ 'pn-dot-ok': c.pass }"></span>
            <b>{{ c.id }}</b>
            <span class="pn-check-name">{{ c.name }}</span>
            <span class="pn-check-metric" :title="c.metric">{{ c.metric }}</span>
          </div>
          <div class="pn-tip mp-side-hint">报告时间：{{ verify.ts }}</div>
        </template>
        <div v-else class="pn-tip">
          还没有校验报告。让 Agent 执行 <code>music_verify</code> 生成旁挂的 music.verify.json。
        </div>
      </div>
    </template>

    <!-- 主区：谱面 / 卷帘 -->
    <template #main>
      <div class="pn-scroll mp-main">
        <div class="pn-row-gap">
          <span class="pn-strong">{{ title }}</span>
          <span class="pn-spacer"></span>
          <button class="pn-btn" :class="{ 'pn-btn-on': viewMode === 'staff' }" @click="switchView('staff')">五线谱</button>
          <button class="pn-btn" :class="{ 'pn-btn-on': viewMode === 'roll' }" @click="switchView('roll')">钢琴卷帘</button>
        </div>
        <canvas ref="scoreCanvas" class="mp-canvas"></canvas>
        <div class="pn-tip">
          {{ viewMode === 'staff'
            ? '简易高音谱表（E4 为最下线）：符头位置 = 音高，横轴 = 小节网格；每轨一组五线。'
            : '钢琴卷帘：纵轴 = 音高，横轴 = 时间；用于快速核对音程与节奏密度。' }}
        </div>
        <div class="pn-tip">
          播放走原生 Web Audio（三角波 + 包络），只读工程数据；谱面与卷帘都是本地绘制的示意，不是排版级乐谱。
        </div>
      </div>
    </template>
  </PanelShell>
</template>

<script setup>
// MusicPanel — 音乐工程只读视图（与 Agent 共用同一份真相源：music.project.json + 旁挂 music.verify.json）。
// 设计取舍（与 tool-voice 一致）：面板不做写操作——音符编辑一律经 Agent 的 music_edit（命令式 op），
// 避免出现「工具写工程 / 面板写工程」两条写路径导致真相源分叉。
// 播放用原生 Web Audio（零第三方依赖），是「试听」能力，不改动任何工程数据。
// 布局（2026-09-18 改版）：套 PanelShell 统一外壳 —— 顶栏 / 指标条 / 左栏分段（轨道·产物·校验）+ 主区谱面 / 底栏。
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import api from '../api.js'
import PanelShell from './PanelShell.vue'
import { state } from '../ui-state.js'

const projectPath = ref('music.project.json')
const verifyPath = ref('music.verify.json')
const loading = ref(false)
const error = ref('')
const proj = ref(null)
const verify = ref(null)
const viewMode = ref('staff')
const playing = ref(false)
const scoreCanvas = ref(null)
const tab = ref('tracks')

const tracks = computed(() => (proj.value && proj.value.tracks) || [])
const ppq = computed(() => (proj.value && proj.value.ppq) || 480)
const tempo = computed(() => (proj.value && proj.value.tempo) || 120)
const title = computed(() => ((proj.value && proj.value.meta && proj.value.meta.title) || '未命名乐曲'))
const meterText = computed(() => {
  const m = (proj.value && proj.value.meter) || [4, 4]
  return m[0] + '/' + m[1]
})
const noteCount = computed(() => tracks.value.reduce((s, t) => s + ((t.notes && t.notes.length) || 0), 0))
const totalTicks = computed(() => {
  let end = 0
  tracks.value.forEach((t) => (t.notes || []).forEach((n) => { if (n.start + n.dur > end) end = n.start + n.dur }))
  return end || ppq.value
})
const durationText = computed(() => {
  const sec = totalTicks.value / ppq.value * (60 / tempo.value)
  return sec.toFixed(2) + 's'
})
const rangeText = computed(() => {
  let lo = 999
  let hi = -1
  tracks.value.forEach((t) => (t.notes || []).forEach((n) => { if (n.pitch < lo) lo = n.pitch; if (n.pitch > hi) hi = n.pitch }))
  return hi >= lo ? pitchName(lo) + '..' + pitchName(hi) : '（空）'
})
const trackRows = computed(() => tracks.value.map((t) => {
  let lo = 999
  let hi = -1
  ;(t.notes || []).forEach((n) => { if (n.pitch < lo) lo = n.pitch; if (n.pitch > hi) hi = n.pitch })
  return {
    id: t.id,
    name: t.name || '',
    channel: t.channel,
    program: t.program,
    count: (t.notes || []).length,
    range: hi >= lo ? pitchName(lo) + '..' + pitchName(hi) : '（空）',
  }
}))
const verifyPassed = computed(() => ((verify.value && verify.value.checks) || []).filter((c) => c.pass).length)
const verifyTotal = computed(() => ((verify.value && verify.value.checks) || []).length)
const verifyBadge = computed(() => (verify.value ? '校验 ' + verifyPassed.value + '/' + verifyTotal.value : '未校验'))
const verifyTone = computed(() => (!verify.value ? 'muted' : (verifyPassed.value === verifyTotal.value && verifyTotal.value > 0 ? 'ok' : 'bad')))
const artifacts = computed(() => {
  const a = (proj.value && proj.value.artifacts) || {}
  return Object.keys(a).map((k) => ({ kind: k, path: a[k] }))
})

// ── 指标条 ──
const metrics = computed(() => [
  { k: '速度', v: tempo.value + ' BPM' },
  { k: '拍号', v: meterText.value },
  { k: '调号', v: (proj.value && proj.value.key) || 'C' },
  { k: 'ppq', v: String(ppq.value) },
  { k: '轨道', v: String(tracks.value.length) },
  { k: '音符', v: String(noteCount.value) },
  { k: '时长', v: durationText.value },
  { k: '音域', v: rangeText.value, mono: true },
])
const tabs = computed(() => [
  { id: 'tracks', name: '轨道', count: tracks.value.length, title: '轨道清单（通道 / 音色 / 音符 / 音域）' },
  { id: 'out', name: '产物', count: artifacts.value.length, title: 'music_export 产出的文件' },
  { id: 'checks', name: '校验', count: verify.value ? verifyPassed.value + '/' + verifyTotal.value : 0, title: '工程校验报告' },
])

const NOTE_NAMES = ['C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B']
function pitchName(p) {
  return NOTE_NAMES[((p % 12) + 12) % 12] + (Math.floor(p / 12) - 1)
}

function absPath(rel) {
  const root = String((state && state.workspaceRoot) || '').replace(/[\\/]+$/, '')
  if (!root) return rel
  return root + '/' + String(rel).replace(/^[\\/]+/, '')
}

async function readJSON(rel) {
  // ★ apiURL 会自动加 /api 前缀（api.js: BASE='/api'）——再写 /api 会拼成
  //   /api/api/fs/read → 404（实测面板「载入失败：Not Found」正是此因）。
  const r = await api.apiGet('/fs/read', { path: absPath(rel) })
  return JSON.parse(r.content)
}

async function load() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    proj.value = await readJSON(projectPath.value)
    verify.value = null
    try { verify.value = await readJSON(verifyPath.value) } catch (_) { /* 报告可选 */ }
    await nextTick()
    draw()
  } catch (e) {
    proj.value = null
    error.value = '载入失败：' + ((e && e.message) || e)
  } finally {
    loading.value = false
  }
}

function switchView(mode) {
  viewMode.value = mode
  nextTick(draw)
}

// 颜色一律取自设计系统变量（不另起一套配色）
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
  const w = cv.clientWidth || 320
  cv.width = Math.max(1, Math.round(w * dpr))
  cv.height = Math.round(h * dpr)
  const g = cv.getContext('2d')
  g.setTransform(dpr, 0, 0, dpr, 0, 0)
  g.clearRect(0, 0, w, h)
  return { g, w, h }
}

function draw() {
  if (viewMode.value === 'staff') drawStaff()
  else drawRoll()
}

// 五线谱：E4(64) = 最下线；每半音半格；每小节一条竖线
function drawStaff() {
  const cv = scoreCanvas.value
  if (!cv || !proj.value) return
  const n = Math.max(1, tracks.value.length)
  const h = 26 + n * 78
  cv.style.height = h + 'px'
  const { g, w, h: ch } = prepareCanvas(cv, h)
  const cLine = cssVar(cv, '--border-color', '#ddd')
  const cNote = cssVar(cv, '--text-primary', '#333')
  const cDim = cssVar(cv, '--text-muted', '#888')
  const cBg = cssVar(cv, '--bg-primary', '#fff')
  g.fillStyle = cBg
  g.fillRect(0, 0, w, ch)

  const LG = 9
  const left = 56
  const right = w - 14
  const usable = Math.max(20, right - left)
  const meter = (proj.value.meter || [4, 4])
  const measureTicks = Math.round(ppq.value * 4 * meter[0] / meter[1])
  const total = totalTicks.value

  tracks.value.forEach((tr, ti) => {
    const top = 20 + ti * 78
    // 五线
    g.strokeStyle = cLine
    g.lineWidth = 1
    for (let i = 0; i < 5; i++) {
      g.beginPath()
      g.moveTo(left - 12, top + i * LG + 0.5)
      g.lineTo(right, top + i * LG + 0.5)
      g.stroke()
    }
    // 轨道名
    g.fillStyle = cDim
    g.font = '10px system-ui, sans-serif'
    g.fillText(String(tr.name || tr.id).slice(0, 6), 4, top + 4 * LG)

    // 小节线
    const measures = Math.max(1, Math.ceil(total / measureTicks))
    for (let m = 1; m <= measures; m++) {
      const x = Math.round(left + (m * measureTicks / total) * usable)
      g.beginPath()
      g.moveTo(x + 0.5, top)
      g.lineTo(x + 0.5, top + 4 * LG)
      g.stroke()
    }

    // 音符
    const notes = (tr.notes || []).slice().sort((a, b) => a.start - b.start)
    notes.forEach((nt) => {
      const x = left + (nt.start / total) * usable
      const y = top + 4 * LG - (nt.pitch - 64) * (LG / 2)
      // 加线（超出五线范围）
      if (y < top || y > top + 4 * LG) {
        g.strokeStyle = cNote
        g.lineWidth = 1
        g.beginPath()
        g.moveTo(x - 5, Math.round(y) + 0.5)
        g.lineTo(x + 5, Math.round(y) + 0.5)
        g.stroke()
      }
      // 符头
      const filled = nt.dur <= ppq.value
      g.beginPath()
      g.ellipse(x, y, 4.2, 3.1, 0, 0, Math.PI * 2)
      if (filled) { g.fillStyle = cNote; g.fill() } else { g.strokeStyle = cNote; g.lineWidth = 1.2; g.stroke() }
      // 符干
      const up = nt.pitch < 71
      g.strokeStyle = cNote
      g.lineWidth = 1.1
      g.beginPath()
      g.moveTo(x + (up ? 4 : -4), y)
      g.lineTo(x + (up ? 4 : -4), up ? y - 26 : y + 26)
      g.stroke()
    })
  })
}

// 钢琴卷帘：纵轴音高、横轴时间
function drawRoll() {
  const cv = scoreCanvas.value
  if (!cv || !proj.value) return
  let lo = 127
  let hi = 0
  tracks.value.forEach((t) => (t.notes || []).forEach((n) => { if (n.pitch < lo) lo = n.pitch; if (n.pitch > hi) hi = n.pitch }))
  if (hi < lo) { lo = 60; hi = 72 }
  const span = Math.max(6, hi - lo + 1)
  const h = Math.min(460, Math.max(180, span * 9 + 30))
  cv.style.height = h + 'px'
  const { g, w, h: ch } = prepareCanvas(cv, h)
  const cLine = cssVar(cv, '--border-color', '#ddd')
  const cNote = cssVar(cv, '--text-primary', '#333')
  const cDim = cssVar(cv, '--text-muted', '#888')
  const cBg = cssVar(cv, '--bg-primary', '#fff')
  const cAccent = cssVar(cv, '--accent', '#4a90d9')
  g.fillStyle = cBg
  g.fillRect(0, 0, w, ch)

  const left = 44
  const rowH = (ch - 22) / span
  // 八度分隔线 + 音名
  g.font = '9px system-ui, sans-serif'
  for (let p = lo; p <= hi; p++) {
    const y = ch - 12 - (p - lo + 0.5) * rowH
    if (p % 12 === 0) {
      g.strokeStyle = cLine
      g.lineWidth = 1
      g.beginPath()
      g.moveTo(left, y + rowH / 2)
      g.lineTo(w - 8, y + rowH / 2)
      g.stroke()
      g.fillStyle = cDim
      g.fillText(pitchName(p), 4, y + 3)
    }
  }
  // 时间网格（每拍）
  const total = totalTicks.value
  const usable = w - left - 8
  for (let t = 0; t <= total; t += ppq.value) {
    const x = Math.round(left + (t / total) * usable)
    g.strokeStyle = cLine
    g.lineWidth = 1
    g.beginPath()
    g.moveTo(x + 0.5, 8)
    g.lineTo(x + 0.5, ch - 12)
    g.stroke()
  }
  // 音符块（各轨用不同透明度区分，颜色统一取 accent）
  tracks.value.forEach((tr, ti) => {
    const alpha = 1 - Math.min(0.55, ti * 0.22)
    g.globalAlpha = alpha
    ;(tr.notes || []).forEach((nt) => {
      const x = left + (nt.start / total) * usable
      const bw = Math.max(2, (nt.dur / total) * usable)
      const y = ch - 12 - (nt.pitch - lo + 1) * rowH
      g.fillStyle = cAccent
      g.fillRect(x, y + 0.5, bw, Math.max(2, rowH - 1))
    })
  })
  g.globalAlpha = 1
  // cNote 目前未用于卷帘（音符块统一用 accent），保留取色以便后续区分声部
  void cNote
}

// ── 播放（原生 Web Audio，零依赖；不改动工程数据）──
let audioCtx = null
let oscs = []
let endTimer = null

function ensureAudio() {
  if (!audioCtx) {
    const Ctor = window.AudioContext || window.webkitAudioContext
    if (!Ctor) return null
    audioCtx = new Ctor()
  }
  return audioCtx
}

function play() {
  const ctx = ensureAudio()
  if (!ctx) { error.value = '当前浏览器不支持 Web Audio，无法试听'; return }
  if (ctx.state === 'suspended') ctx.resume()
  stopPlay()
  const tickSec = (60 / tempo.value) / ppq.value
  const t0 = ctx.currentTime + 0.08
  const master = ctx.createGain()
  master.gain.value = 0.18
  master.connect(ctx.destination)
  let end = 0
  tracks.value.forEach((tr) => {
    (tr.notes || []).forEach((n) => {
      const start = t0 + n.start * tickSec
      const dur = Math.max(0.06, n.dur * tickSec)
      const osc = ctx.createOscillator()
      osc.type = 'triangle'
      osc.frequency.value = 440 * Math.pow(2, (n.pitch - 69) / 12)
      const gain = ctx.createGain()
      const vel = ((n.vel === undefined ? 90 : n.vel) / 127)
      gain.gain.setValueAtTime(0, start)
      gain.gain.linearRampToValueAtTime(0.5 * vel, start + 0.012)
      gain.gain.setTargetAtTime(0.0001, start + Math.max(0.03, dur * 0.72), 0.05)
      osc.connect(gain)
      gain.connect(master)
      osc.start(start)
      osc.stop(start + dur + 0.3)
      oscs.push(osc)
      if (n.start + n.dur > end) end = n.start + n.dur
    })
  })
  playing.value = true
  endTimer = setTimeout(() => { playing.value = false }, (end * tickSec + 0.5) * 1000)
}

function stopPlay() {
  oscs.forEach((o) => { try { o.stop() } catch (_) { /* 已停止 */ } })
  oscs = []
  if (endTimer) { clearTimeout(endTimer); endTimer = null }
  playing.value = false
}

function redraw() {
  draw()
}

onMounted(() => {
  load()
  window.addEventListener('resize', redraw)
})
onBeforeUnmount(() => {
  stopPlay()
  window.removeEventListener('resize', redraw)
})
</script>

<style scoped>
/* 局部样式（其余复用 PanelShell 共享类；配色取设计系统变量） */
.mp-main { display: flex; flex-direction: column; gap: 10px; padding: 10px; }
.mp-canvas { width: 100%; display: block; background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 4px; }
.mp-track { display: flex; flex-direction: column; gap: 1px; padding: 5px 6px; border-radius: 4px; }
.mp-track:hover { background: var(--bg-hover); }
.mp-track-h { display: flex; align-items: baseline; gap: 6px; font-size: 11px; }
.mp-track-meta { line-height: 1.5; }
.mp-side-hint { margin-top: 6px; }
.mp-wrap { white-space: normal; overflow-wrap: anywhere; text-align: left; }
</style>
