<template>
  <div class="music-panel">
    <div class="mp-bar">
      <span class="mp-title">音乐工程</span>
      <input v-model="projectPath" class="mp-input" placeholder="music.project.json" @keyup.enter="load" />
      <button class="mp-icon-btn" :disabled="loading" title="重新载入工程与校验报告" @click="load">
        <SvgIcon name="refresh" :size="12" :class="{ spinning: loading }" />
      </button>
    </div>

    <div v-if="error" class="mp-msg mp-err">{{ error }}</div>
    <div v-else-if="!proj" class="mp-msg">
      未找到音乐工程。先让 Agent 执行 <code>music_project</code> 创建工程（生成 music.project.json）。
    </div>

    <template v-else>
      <!-- 工程信息 -->
      <section class="mp-sec">
        <div class="mp-sec-title">{{ title }}</div>
        <div class="mp-kv"><span>速度 / 拍号 / 调号</span><b>{{ tempo }} BPM · {{ meterText }} · {{ proj.key || 'C' }}</b></div>
        <div class="mp-kv"><span>ppq / 轨道 / 音符</span><b>{{ ppq }} · {{ tracks.length }} · {{ noteCount }}</b></div>
        <div class="mp-kv"><span>时长 / 音域</span><b>{{ durationText }} · {{ rangeText }}</b></div>
        <div class="mp-kv"><span>文件</span><b class="mp-mono">{{ projectPath }}</b></div>
      </section>

      <!-- 视图切换 + 播放 -->
      <section class="mp-sec">
        <div class="mp-row">
          <button :class="['mp-tab', viewMode === 'staff' ? 'mp-tab-on' : '']" @click="switchView('staff')">五线谱</button>
          <button :class="['mp-tab', viewMode === 'roll' ? 'mp-tab-on' : '']" @click="switchView('roll')">钢琴卷帘</button>
          <span class="mp-spacer"></span>
          <button class="mp-btn" :disabled="!noteCount" @click="playing ? stopPlay() : play()">
            {{ playing ? '停止' : '播放' }}
          </button>
        </div>
        <canvas ref="scoreCanvas" class="mp-canvas"></canvas>
        <div class="mp-dim">
          {{ viewMode === 'staff'
            ? '简易高音谱表（E4 为最下线）：符头位置 = 音高，横轴 = 小节网格'
            : '钢琴卷帘：纵轴 = 音高，横轴 = 时间；用于快速核对音程与节奏密度' }}
        </div>
      </section>

      <!-- 轨道 -->
      <section class="mp-sec">
        <div class="mp-sec-title">轨道 <span class="mp-dim">（{{ tracks.length }}）</span></div>
        <table class="mp-table">
          <thead>
            <tr><th>id</th><th>名称</th><th>通道</th><th>音色</th><th>音符</th><th>音域</th></tr>
          </thead>
          <tbody>
            <tr v-for="tr in trackRows" :key="tr.id">
              <td class="mp-mono">{{ tr.id }}</td>
              <td>{{ tr.name }}</td>
              <td>{{ tr.channel }}</td>
              <td>{{ tr.program }}</td>
              <td>{{ tr.count }}</td>
              <td class="mp-mono">{{ tr.range }}</td>
            </tr>
          </tbody>
        </table>
      </section>

      <!-- 校验报告（旁挂文件 music.verify.json，由 music_verify 写入） -->
      <section class="mp-sec">
        <div class="mp-sec-title">
          工程校验
          <span v-if="verify" :class="['mp-badge', verify.pass ? 'mp-ok' : 'mp-bad']">
            {{ verifyPassed }}/{{ verifyTotal }}
          </span>
          <span v-else class="mp-dim">（未校验）</span>
        </div>
        <div v-if="!verify" class="mp-dim">让 Agent 执行 <code>music_verify</code> 生成校验报告（music.verify.json）。</div>
        <div v-else v-for="c in verify.checks || []" :key="c.id" class="mp-check">
          <span :class="['mp-dot', c.pass ? 'mp-ok' : 'mp-bad']"></span>
          <b>{{ c.id }}</b>
          <span class="mp-check-name">{{ c.name }}</span>
          <span class="mp-dim mp-check-metric">{{ c.metric }}</span>
        </div>
        <div v-if="verify" class="mp-dim">报告时间：{{ verify.ts }}</div>
      </section>

      <!-- 产物 -->
      <section v-if="artifacts.length" class="mp-sec">
        <div class="mp-sec-title">产物 <span class="mp-dim">（由 music_export 生成）</span></div>
        <div v-for="a in artifacts" :key="a.kind" class="mp-kv mp-kv-left">
          <span>{{ a.kind }}</span><b class="mp-mono">{{ a.path }}</b>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup>
// MusicPanel — 音乐工程只读视图（与 Agent 共用同一份真相源：music.project.json + 旁挂 music.verify.json）。
// 设计取舍（与 tool-voice 一致）：面板不做写操作——音符编辑一律经 Agent 的 music_edit（命令式 op），
// 避免出现「工具写工程 / 面板写工程」两条写路径导致真相源分叉。
// 播放用原生 Web Audio（零第三方依赖），是「试听」能力，不改动任何工程数据。
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'
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
const artifacts = computed(() => {
  const a = (proj.value && proj.value.artifacts) || {}
  return Object.keys(a).map((k) => ({ kind: k, path: a[k] }))
})

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
.music-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 10px;
  font-size: 12px;
  color: var(--text-primary);
  overflow-y: auto;
  height: 100%;
}
.mp-bar { display: flex; align-items: center; gap: 6px; }
.mp-title { font-weight: 600; color: var(--text-secondary); }
.mp-input {
  flex: 1;
  min-width: 0;
  padding: 3px 6px;
  font-size: 11px;
  color: var(--text-primary);
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  border-radius: 4px;
}
.mp-icon-btn {
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
.mp-icon-btn:hover { background: var(--bg-hover); }
.spinning { animation: mp-spin 1s linear infinite; }
@keyframes mp-spin { to { transform: rotate(360deg); } }

.mp-msg { padding: 8px; color: var(--text-muted); line-height: 1.6; }
.mp-err { color: var(--text-primary); border-left: 2px solid var(--accent); background: var(--bg-tertiary); }
.mp-sec { display: flex; flex-direction: column; gap: 4px; padding: 8px; background: var(--bg-tertiary); border-radius: 6px; }
.mp-sec-title { font-weight: 600; color: var(--text-secondary); margin-bottom: 2px; }
.mp-kv { display: flex; gap: 8px; justify-content: space-between; }
.mp-kv > span { color: var(--text-muted); flex: none; }
.mp-kv > b { font-weight: 500; text-align: right; word-break: break-all; }
/* 产物路径：左对齐 + 允许任意位置换行（右对齐会被容器宽度截断，实测「music.mid」显示不全） */
.mp-kv-left > b { text-align: left; overflow-wrap: anywhere; }
.mp-mono { font-family: ui-monospace, Consolas, monospace; font-size: 11px; color: var(--text-secondary); }
.mp-dim { color: var(--text-muted); font-weight: 400; }
.mp-row { display: flex; align-items: center; gap: 6px; }
.mp-spacer { flex: 1; }
.mp-tab, .mp-btn {
  padding: 3px 10px;
  font-size: 11px;
  color: var(--text-secondary);
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  cursor: pointer;
}
.mp-tab-on { color: var(--text-primary); background: var(--bg-hover); border-color: var(--accent); }
.mp-btn:hover, .mp-tab:hover { background: var(--bg-hover); }
.mp-canvas { width: 100%; display: block; background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 4px; }
.mp-table { width: 100%; border-collapse: collapse; font-size: 11px; }
.mp-table th { color: var(--text-muted); font-weight: 500; text-align: left; padding: 2px 4px; border-bottom: 1px solid var(--border-color); }
.mp-table td { padding: 2px 4px; border-bottom: 1px solid var(--bg-hover); }
.mp-badge { padding: 1px 6px; border-radius: 8px; font-size: 11px; }
.mp-dot { width: 7px; height: 7px; border-radius: 50%; flex: none; display: inline-block; }
.mp-ok { background: var(--accent); color: var(--bg-primary); }
.mp-bad { background: var(--text-muted); color: var(--bg-primary); }
.mp-check { display: flex; align-items: center; gap: 6px; }
.mp-check > b { flex: none; color: var(--text-secondary); }
.mp-check-name { flex: none; }
.mp-check-metric { flex: 1; text-align: right; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
