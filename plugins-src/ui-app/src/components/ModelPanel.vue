<template>
  <div class="mp-panel" :class="{ 'mp-fill': fill }">
    <div class="mp-bar">
      <span class="mp-title">3D 模型</span>
      <span class="mp-chip" :class="bridge.mode === 'invoke' ? 'mp-chip-ok' : 'mp-chip-warn'">
        {{ bridge.mode === 'invoke' ? '工具+UI 同包直连' : '只读降级' }}
      </span>
      <span class="mp-spacer"></span>
      <button class="mp-icon-btn" :disabled="loading" title="重载「识别到的文件」" @click="refresh(scanning)">
        <SvgIcon name="refresh" :size="12" :class="{ spinning: loading }" />
      </button>
    </div>

    <!-- ── 指引：这个面板是什么 / 怎么打开预览（用户第一眼就该看到）──────── -->
    <section class="mp-sec mp-guide">
      <div class="mp-row mp-guide-head" @click="guideOpen = !guideOpen">
        <span class="mp-sec-title">这个面板是什么 · 怎么用</span>
        <span class="mp-spacer"></span>
        <span class="mp-dim">{{ guideOpen ? '收起' : '展开' }}</span>
      </div>
      <template v-if="guideOpen">
        <div class="mp-dim">
          工具写出的**每一个文件**（工程 JSON / glTF / GLB / STL / OBJ / 自包含预览 HTML / 复验报告）
          都会自动出现在下面「识别到的文件」里 —— 点文件名即登记预览，页面**实时跟随文件变化重绘**。
        </div>
        <div class="mp-steps">
          <div class="mp-step"><b>1</b><span>让 Agent 产出文件：<code>model_export format=all</code> → <code>model_verify</code></span></div>
          <div class="mp-step"><b>2</b><span>文件自动出现在「识别到的文件」（历史/手工文件点<code>扫描工作区</code>也能找到）</span></div>
          <div class="mp-step"><b>3</b><span>点文件名 → 当场实时预览（改工程/重导出后无需手点刷新）</span></div>
        </div>
        <div class="mp-open">
          <div class="mp-dim"><b>从工程里打开预览的三种入口：</b></div>
          <div class="mp-dim">① 主内容区顶部 tab「<b>3D 模型</b>」（默认已后台打开，可与对话并排）</div>
          <div class="mp-dim">② 插件面板里的「<b>3D 模型</b>」面板（本页）</div>
          <div class="mp-dim">③ 直接双击导出的 <code>*.preview.html</code>（自包含，浏览器即可打开）</div>
        </div>
        <div class="mp-dim">{{ bridge.modeNote }}</div>
      </template>
    </section>

    <div v-if="error" class="mp-msg mp-err">{{ error }}</div>

    <!-- ── 识别到的文件 ───────────────────────────────────────────── -->
    <section class="mp-sec mp-sec-list">
      <div class="mp-row">
        <span class="mp-sec-title">识别到的文件 <span class="mp-dim">（{{ items.length }}）</span></span>
        <span class="mp-spacer"></span>
        <button class="mp-btn" :disabled="loading" @click="scanNow">
          {{ scanning ? '扫描工作区：开' : '仅工具产出' }}
        </button>
      </div>
      <div v-if="!items.length" class="mp-dim mp-msg">
        还没有识别到文件。让 Agent 执行 <code>model_export format=all</code>，或点上面的「仅工具产出/扫描工作区」切换扫描。
      </div>
      <div
        v-for="it in items"
        :key="it.path"
        class="mp-item"
        :class="{ 'mp-item-on': it.path === selected }"
        @click="select(it.path)"
      >
        <span class="mp-kind" :class="'mp-kind-' + (it.kind || 'x')">{{ kindShort(it.kind) }}</span>
        <span class="mp-mono mp-item-path" :title="it.path">{{ shortPath(it.path) }}</span>
        <span class="mp-spacer"></span>
        <span v-if="it.source === 'project'" class="mp-src">工具·工程</span>
        <span v-else-if="it.source === 'export'" class="mp-src">工具·产物</span>
        <span v-else-if="it.source === 'verify'" class="mp-src">工具·报告</span>
        <span v-else-if="it.source === 'claim'" class="mp-src">已登记</span>
        <span v-else class="mp-src mp-src-dim">扫描</span>
        <span class="mp-dim mp-item-meta">{{ fmtSize(it.bytes) }}<span v-if="it.mtime"> · {{ fmtTime(it.mtime) }}</span></span>
      </div>
    </section>

    <!-- ── 预览（点选即实时渲染）─────────────────────────────────── -->
    <section class="mp-sec mp-sec-preview">
      <div class="mp-row">
        <span class="mp-sec-title">预览</span>
        <span class="mp-spacer"></span>
        <span class="mp-dim" v-if="selected">{{ metaLabel }}</span>
      </div>
      <div v-if="!selected" class="mp-dim mp-msg">点上面任一文件即在此实时预览。</div>
      <template v-else>
        <div class="mp-dim mp-fileline">
          <span class="mp-mono">{{ selected }}</span>
          <span v-if="meta.bytes"> · {{ fmtSize(meta.bytes) }}</span>
          <span v-if="meta.mtime"> · 更新于 {{ fmtTime(meta.mtime) }}</span>
          <span class="mp-live">实时</span>
        </div>
        <div v-if="preview.loading" class="mp-dim">读取中…</div>
        <div v-else-if="preview.error" class="mp-msg mp-err">{{ preview.error }}</div>
        <template v-else>
          <ModelViewer
            v-if="geo.length"
            :parts="geo"
            :fill="fill"
            :key="'geo:' + selected + ':' + renderStamp"
          />
          <iframe
            v-else-if="frame"
            class="mp-frame"
            :srcdoc="frame"
            sandbox="allow-scripts"
            referrerpolicy="no-referrer"
          ></iframe>
          <template v-else-if="verifyInfo">
            <div class="mp-row">
              <span class="mp-dim">判据 M1–M9</span>
              <span class="mp-spacer"></span>
              <span class="mp-badge" :class="verifyInfo.ok ? 'mp-ok' : 'mp-bad'">{{ passedCount }}/9 通过</span>
            </div>
            <div v-for="c in (verifyInfo.checks || [])" :key="c.id" class="mp-check">
              <span class="mp-dot" :class="c.ok ? 'mp-ok' : (c.level === 'fail' ? 'mp-bad' : 'mp-warn')"></span>
              <b>{{ c.id }}</b>
              <span class="mp-check-name">{{ c.title }}</span>
              <span class="mp-check-metric mp-dim">{{ c.detail }}</span>
            </div>
            <div v-if="verifyInfo.parts && verifyInfo.parts.length" class="mp-kv mp-parthead">
              <span>部件（{{ verifyInfo.parts.length }}）</span><b>面数 / 体积 mm³</b>
            </div>
            <div v-for="p in (verifyInfo.parts || [])" :key="p.id" class="mp-kv">
              <span class="mp-mono">{{ p.id }} <span class="mp-dim">{{ p.type }}</span></span>
              <b>{{ (p.triangles || 0).toLocaleString() }} / {{ p.volume }}</b>
            </div>
          </template>
          <pre v-else-if="rawText" class="mp-code">{{ rawText }}</pre>
          <div v-else class="mp-dim">该文件没有可渲染内容（可在文本编辑器中查看）。</div>
          <div v-if="parseNote" class="mp-dim">{{ parseNote }}</div>
        </template>
      </template>
    </section>

    <!-- ── 工程详情（选中工程 / 点击 model.json 时）────────────────── -->
    <section v-if="projInfo" class="mp-sec">
      <div class="mp-row">
        <span class="mp-sec-title">工程 {{ projInfo.name }}</span>
        <span class="mp-spacer"></span>
        <span class="mp-dim">{{ projInfo.unit }} · {{ projInfo.up }}-up</span>
      </div>
      <div class="mp-kv"><span>部件 / 三角面</span><b>{{ projInfo.parts.length }} · {{ (projInfo.totalTris || 0).toLocaleString() }}</b></div>
      <div class="mp-kv"><span>体积合计</span><b>{{ projInfo.totalVolume }} mm³</b></div>
      <div class="mp-kv"><span>包围盒</span><b>{{ bboxText }}</b></div>
      <div class="mp-sec-title mp-sub">参数（{{ projInfo.params.length }}）</div>
      <div v-for="p in projInfo.params" :key="p.id" class="mp-kv">
        <span class="mp-mono">{{ p.name || p.id }}</span>
        <b>{{ p.value }}<span v-if="p.range" class="mp-dim"> ({{ p.range }})</span></b>
      </div>
      <div v-if="!projInfo.params.length" class="mp-dim">未声明参数（尺寸为字面量）。</div>
      <div class="mp-dim mp-note">
        几何由 host 半内核现场构建（同一份代码，与 Agent 看到的一致）；工程或参数一变，这里立即重绘。
      </div>
    </section>

    <!-- ── 命令样例（复制给 Agent 用）────────────────────────────── -->
    <section class="mp-sec">
      <div class="mp-row">
        <span class="mp-sec-title">Agent 命令</span>
        <span class="mp-spacer"></span>
        <button class="mp-btn" @click="copyCmd">{{ copied ? '已复制' : '复制' }}</button>
      </div>
      <pre class="mp-code">{{ sampleCmd }}</pre>
    </section>
  </div>
</template>

<script setup>
// ModelPanel — 3D/CAD 面板（★ 2026-09-17 与工具合并进同一插件 tool-model）
//
// 口径：工具操作到**具体文件** → 每次写盘 host 半立即登记并广播 →
//   本面板列出「识别到的文件」→ 点击即登记（claimArtifact）+ 当场预览 + **实时**跟随变化。
//   预览渲染有三条路：① 工程 JSON → host 内核现场构建几何 → ModelViewer；
//   ② glTF/GLB/STL/OBJ → 前端解析（model-parsers）→ ModelViewer；
//   ③ 自包含预览 HTML → iframe srcdoc；复验报告 → 判据列表。
// 写路径纪律：面板不写工作区（几何/参数一律经 Agent 的 model_add / model_edit / model_doc），
//   避免「工具写工程 / 面板写工程」两条写路径把真相源写分叉。
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import SvgIcon from './SvgIcon.vue'
import ModelViewer from './ModelViewer.vue'
import { createBridge } from '../model-bridge.js'
import { parseArtifact } from '../model-parsers.js'

const props = defineProps({ fill: { type: Boolean, default: false } })

const bridge = createBridge()
const items = ref([])
const selected = ref('')
const scanMode = ref(true)
const loading = ref(false)
const error = ref('')
// 主内容区（fill）里重点是预览：指引默认收起（侧栏面板空间宽裕，默认展开）
const guideOpen = ref(!props.fill)
const copied = ref(false)
const geo = ref([])
const frame = ref('')
const projInfo = ref(null)
const verifyInfo = ref(null)
const rawText = ref('')
const parseNote = ref('')
const meta = ref({ kind: '', label: '', bytes: 0, mtime: '' })
const preview = ref({ loading: false, error: '' })
const renderStamp = ref(0)

// ── 显示辅助 ───────────────────────────────────────────────
const KIND_SHORT = {
  project: '工程', 'preview-html': '预览HTML', gltf: 'glTF', glb: 'GLB',
  stl: 'STL', obj: 'OBJ', verify: '报告', json: 'JSON',
}
const KIND_LABEL = {
  project: 'CAD 工程（paircode.model/1）', 'preview-html': '自包含 WebGL 预览页',
  gltf: 'glTF 2.0（Khronos）', glb: 'GLB 单文件二进制', stl: 'STL（3D 打印）',
  obj: 'Wavefront OBJ', verify: '复验报告 M1–M9', json: 'JSON 文档',
}
function kindShort(k) { return KIND_SHORT[k] || (k || '?') }
function shortPath(p) {
  const parts = String(p).split(/[\\/]/)
  return parts.length <= 2 ? String(p) : '…/' + parts.slice(-2).join('/')
}
function fmtSize(n) {
  const b = Number(n || 0)
  if (b < 1024) return b + ' B'
  if (b < 1024 * 1024) return (b / 1024).toFixed(1) + ' KB'
  return (b / 1024 / 1024).toFixed(2) + ' MB'
}
function fmtTime(s) {
  const t = String(s || '')
  const m = t.match(/T(\d{2}:\d{2}:\d{2})/)
  return m ? m[1] : t
}
const metaLabel = computed(() => {
  const k = meta.value.kind
  return (KIND_LABEL[k] || kindShort(k)) + (preview.value.loading ? ' · 读取中' : '')
})
const passedCount = computed(() => ((verifyInfo.value && verifyInfo.value.checks) || []).filter((c) => c.ok).length)
const bboxText = computed(() => {
  const p = projInfo.value
  if (!p || !p.bbox) return '—'
  const size = [p.bbox[1][0] - p.bbox[0][0], p.bbox[1][1] - p.bbox[0][1], p.bbox[1][2] - p.bbox[0][2]]
  return size.map((v) => Math.round(v * 1000) / 1000).join(' × ') + ' mm'
})
const sampleCmd = computed(() => [
  '# 改工程 → 重算 → 复验 → 出产物（面板会立刻刷新并可点开预览）',
  'model_export format=all',
  'model_verify',
].join('\n'))

// ── 数据流 ─────────────────────────────────────────────────
let offEvent = null
let timer = 0
let lastSig = ''
let stopFlag = false

function sigOf(it) { return String(it && (it.bytes + '|' + it.mtime)) }

async function refresh(scan = scanMode.value) {
  if (stopFlag) return
  loading.value = true
  error.value = ''
  try {
    const r = await bridge.listArtifacts({ scan })
    const scanned = (items.value || []).filter((x) => x.source === 'scan')
    const reg = (r && r.items) || []
    const seen = new Set(reg.map((x) => x.path))
    items.value = reg.concat(scan ? scanned.filter((x) => !seen.has(x.path)) : [])
  } catch (e) {
    error.value = '读取识别到的文件失败：' + ((e && e.message) || e)
  } finally {
    loading.value = false
  }
}

async function scanNow() {
  scanMode.value = !scanMode.value
  await refresh(scanMode.value)
  if (!selected.value) autoPick()
}

function autoPick() {
  const list = items.value || []
  const proj = list.find((x) => x.kind === 'project')
  const pick = proj || list.find((x) => x.kind === 'preview-html') || list.find((x) => x.kind !== 'verify') || list[0]
  if (pick) select(pick.path)
}

async function renderFor(art) {
  geo.value = []
  frame.value = ''
  projInfo.value = null
  verifyInfo.value = null
  rawText.value = ''
  parseNote.value = ''
  const kind = art.kind
  if (kind === 'project') {
    const bp = await bridge.buildProject(art.path)
    projInfo.value = bp
    geo.value = bp.parts || []
    parseNote.value = '几何由 tool-model 内核现场构建（' + (bp.totalTris || 0).toLocaleString() + ' 面 / ' + (bp.parts || []).length + ' 部件）'
    return
  }
  if (kind === 'preview-html') { frame.value = art.text || ''; return }
  if (kind === 'gltf' || kind === 'glb' || kind === 'stl' || kind === 'obj') {
    const r = parseArtifact(art)
    geo.value = r.parts
    parseNote.value = '前端当场解析：' + r.format + ' · ' + r.note
    return
  }
  if (kind === 'verify') {
    try { verifyInfo.value = JSON.parse(art.text) } catch (e) { rawText.value = String(art.text || '').slice(0, 4000) }
    return
  }
  // 普通 JSON：若是 CAD 工程（format=paircode.model/1）就按工程渲染
  try {
    const doc = JSON.parse(art.text)
    if (doc && (doc.format === 'paircode.model/1' || (doc.parts && doc.params))) {
      const bp = await bridge.buildProject(art.path)
      projInfo.value = bp
      geo.value = bp.parts || []
      parseNote.value = '按 CAD 工程渲染（内核现场构建 ' + (bp.totalTris || 0).toLocaleString() + ' 面）'
      return
    }
    rawText.value = JSON.stringify(doc, null, 2).slice(0, 4000)
  } catch (e) {
    rawText.value = String(art.text || '').slice(0, 4000)
  }
}

async function select(path) {
  if (!path) return
  selected.value = path
  preview.value = { loading: true, error: '' }
  try {
    // 点选即登记为「识别到的文件」（host 侧登记 + 广播，其它面板也能看到）
    Promise.resolve(bridge.claimArtifact(path)).catch(() => {})
    const art = await bridge.readArtifact(path)
    // 读后校正类型（.json 可能是 CAD 工程；host 按 format 判定后才准）→ 列表标签同步
    if (art.kind) {
      const it = (items.value || []).find((x) => x.path === path)
      if (it && it.kind !== art.kind) { it.kind = art.kind; it.label = art.label || KIND_LABEL[art.kind] || '' }
    }
    meta.value = {
      kind: art.kind || '',
      label: art.label || KIND_LABEL[art.kind] || '',
      bytes: art.bytes || 0,
      mtime: art.mtime || '',
    }
    lastSig = sigOf(art)
    await renderFor(art)
    renderStamp.value++
  } catch (e) {
    preview.value.error = '预览失败：' + ((e && e.message) || e)
  } finally {
    preview.value.loading = false
  }
}

// 登记表刷新（工具产出条目；工作区扫描条目保留）
async function refreshRegistry() {
  const r = await bridge.listArtifacts({ scan: false })
  const reg = (r && r.items) || []
  const seen = new Set(reg.map((x) => x.path))
  items.value = reg.concat((items.value || []).filter((x) => (x.source === 'scan' || x.source === 'http') && !seen.has(x.path)))
  if (!selected.value && items.value.length) autoPick()
}

// 实时：宿主广播（事件驱动，工具写文件即到）+ 定时复核选中文件的 size/mtime
// （兜底：① 无事件桥的旧宿主；② 工作区里被 Agent/外部编辑器改动的文件——不在登记表也能刷新）
let pollCount = 0
async function pollTick() {
  if (stopFlag || preview.value.loading) return
  pollCount++
  try {
    if (selected.value) {
      const st = await bridge.statArtifact(selected.value)
      if (st && st.exists === false) {
        preview.value.error = '文件已不存在（' + selected.value + '）'
      } else if (st && sigOf(st) !== lastSig) {
        await select(selected.value)   // 内容变了 → 重新读取 + 重绘（实时）
      }
    }
    // 登记表刷新：降级模式每次都刷（本地 HTTP 很轻）；直连模式每 6 秒一次
    if (bridge.mode !== 'invoke' || pollCount % 3 === 0) await refreshRegistry()
  } catch (_) { /* 轮询失败静默：下次再试 */ }
}

let scheduleTimer = 0
function scheduleRefresh() {
  if (scheduleTimer) return
  scheduleTimer = setTimeout(async () => {
    scheduleTimer = 0
    if (stopFlag) return
    await refresh(scanMode.value)
    // 选中文件内容变了 → 立即重绘（实时）
    const it = (items.value || []).find((x) => x.path === selected.value)
    if (it && sigOf(it) !== lastSig) await select(selected.value)
    else if (!selected.value && items.value.length) autoPick()
  }, 150)
}

function copyCmd() {
  const text = sampleCmd.value
  try {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(() => {
        copied.value = true
        setTimeout(() => { copied.value = false }, 1500)
      })
      return
    }
  } catch (_) { /* 回退：不复制 */ }
  copied.value = false
}

onMounted(async () => {
  offEvent = bridge.onArtifacts(() => scheduleRefresh())
  timer = setInterval(pollTick, 2000)
  await refresh(true)
  autoPick()
})

onBeforeUnmount(() => {
  stopFlag = true
  if (offEvent) { try { offEvent() } catch (_) { /* ignore */ } }
  if (timer) clearInterval(timer)
  if (scheduleTimer) clearTimeout(scheduleTimer)
})
</script>

<style scoped>
.mp-panel { padding: 8px 10px; font-size: 12px; color: var(--text-primary); overflow-y: auto; height: 100%; }
/* ★ 主内容区视图（fill）：面板做成纵向 flex —— 指引/列表按内容高度（列表限高可滚动），
   预览区吃掉剩余空间，模型始终占满可视区（此前预览被前面的内容挤出屏幕，要滚动才看得见）。 */
.mp-fill { padding-bottom: 10px; display: flex; flex-direction: column; overflow-y: auto; }
.mp-fill .mp-sec { flex: none; }
.mp-fill .mp-sec-list { max-height: 30vh; overflow-y: auto; }
.mp-fill .mp-sec-preview { flex: 1 1 auto; min-height: 420px; display: flex; flex-direction: column; }
.mp-fill .mp-sec-preview .mp-fileline,
.mp-fill .mp-sec-preview .mp-row { flex: none; }
.mp-fill .mv-full { flex: 1 1 auto; height: auto; min-height: 320px; }
.mp-fill .mp-frame { flex: 1 1 auto; height: auto; min-height: 320px; }
.mp-bar { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
.mp-title { font-size: 12px; font-weight: 500; color: var(--text-secondary); flex: none; }
.mp-spacer { flex: 1; }
.mp-chip { padding: 1px 6px; border-radius: 8px; font-size: 10px; border: 1px solid var(--border-color); color: var(--text-muted); }
.mp-chip-ok { color: var(--text-secondary); }
.mp-chip-warn { color: var(--text-muted); opacity: 0.85; }
.mp-icon-btn {
  flex: none; display: inline-flex; align-items: center; justify-content: center; width: 22px; height: 22px;
  color: var(--text-muted); background: transparent; border: 1px solid var(--border-color);
  border-radius: 4px; cursor: pointer;
}
.mp-icon-btn:hover:not(:disabled) { background: var(--bg-hover); }
.mp-icon-btn:disabled { cursor: default; opacity: 0.6; }
.mp-sec { margin-bottom: 12px; padding-bottom: 10px; border-bottom: 1px solid var(--border-color); }
.mp-sec:last-child { border-bottom: none; }
.mp-sec-title { font-size: 12px; font-weight: 500; color: var(--text-secondary); margin-bottom: 6px; }
.mp-sub { margin-top: 8px; }
.mp-kv { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; padding: 2px 0; }
.mp-kv > span { color: var(--text-muted); flex: none; }
.mp-kv > b { color: var(--text-primary); font-weight: 500; text-align: right; overflow: hidden; text-overflow: ellipsis; }
.mp-mono { font-family: ui-monospace, Consolas, monospace; }
.mp-dim { color: var(--text-muted); font-weight: 400; font-size: 11px; line-height: 1.6; }
.mp-row { display: flex; align-items: center; gap: 6px; }
.mp-btn {
  padding: 3px 10px; font-size: 11px; color: var(--text-secondary);
  background: transparent; border: 1px solid var(--border-color); border-radius: 4px; cursor: pointer;
}
.mp-btn:hover:not(:disabled) { background: var(--bg-hover); }
.mp-frame { display: block; width: 100%; height: 420px; background: #14161a; border: 1px solid var(--border-color); border-radius: 4px; }
.mp-code {
  margin: 0; padding: 6px; font-family: ui-monospace, Consolas, monospace;
  font-size: 10px; line-height: 1.5; color: var(--text-secondary);
  background: var(--bg-primary); border: 1px solid var(--border-color);
  border-radius: 4px; overflow-x: auto; white-space: pre;
}
.mp-badge { padding: 1px 6px; border-radius: 8px; font-size: 11px; }
.mp-dot { width: 7px; height: 7px; border-radius: 50%; flex: none; display: inline-block; }
.mp-ok { background: var(--accent); color: var(--bg-primary); }
.mp-bad { background: var(--text-muted); color: var(--bg-primary); }
.mp-warn { background: var(--text-muted); opacity: 0.55; }
.mp-check { display: flex; align-items: center; gap: 6px; padding: 1px 0; }
.mp-check > b { flex: none; color: var(--text-secondary); }
.mp-check-name { flex: none; }
.mp-check-metric { flex: 1; text-align: right; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mp-msg { padding: 4px 0; }
.mp-err { color: var(--text-secondary); }
.mp-guide { background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 4px; padding: 6px 8px; }
.mp-guide-head { cursor: pointer; }
.mp-steps { margin: 4px 0; }
.mp-step { display: flex; gap: 6px; padding: 1px 0; }
.mp-step > b {
  flex: none; width: 14px; height: 14px; margin-top: 2px; border-radius: 50%;
  font-size: 10px; line-height: 14px; text-align: center;
  color: var(--text-muted); border: 1px solid var(--border-color);
}
.mp-open { margin-top: 4px; }
.mp-item {
  display: flex; align-items: center; gap: 6px; padding: 3px 4px;
  border: 1px solid transparent; border-radius: 4px; cursor: pointer;
}
.mp-item:hover { background: var(--bg-hover); }
.mp-item-on { border-color: var(--text-muted); background: var(--bg-hover); }
.mp-item-path { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mp-item-meta { flex: none; }
.mp-kind {
  flex: none; min-width: 44px; padding: 0 4px; font-size: 10px; text-align: center;
  color: var(--text-muted); border: 1px solid var(--border-color); border-radius: 3px;
}
.mp-kind-project { color: var(--text-secondary); }
.mp-src { flex: none; font-size: 10px; color: var(--text-muted); }
.mp-src-dim { opacity: 0.6; }
.mp-fileline { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.mp-live { color: var(--text-secondary); font-size: 10px; border: 1px solid var(--border-color); border-radius: 3px; padding: 0 4px; }
.mp-parthead { margin-top: 6px; }
.mp-note { margin-top: 6px; }
</style>
