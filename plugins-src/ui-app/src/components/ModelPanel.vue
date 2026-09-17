<template>
  <PanelShell
    title="3D 模型"
    icon="package"
    :badge="modeBadge"
    :badge-tone="bridge.mode === 'invoke' ? 'ok' : 'muted'"
    :metrics="metrics"
    :loading="loading"
    :error="error"
    :empty="!items.length"
    reload-title="重载「识别到的文件」"
    footnote="只读面板：几何/参数改动一律经 Agent 的 model_add / model_edit / model_doc；面板实时跟随文件变化。"
    source="model.*"
    @reload="refreshNow"
  >
    <template #head-actions>
      <button class="pn-btn" :disabled="loading" title="在「仅工具产出」与「扫描工作区」之间切换" @click="scanNow">
        {{ scanMode ? '扫描工作区：开' : '仅工具产出' }}
      </button>
    </template>

    <!-- 空态：把原来那个「这个面板是什么 · 怎么用」的指引放到这里，第一眼就有答案 -->
    <template #empty>
      <div class="pn-empty-title">还没有识别到 3D / CAD 文件</div>
      <div class="pn-tip">
        工具写出的<b>每一个文件</b>（工程 JSON / glTF / GLB / STL / OBJ / 自包含预览 HTML / 复验报告）
        都会自动出现在左栏「文件」里 —— 点文件名即登记预览，页面<b>实时跟随文件变化重绘</b>。
      </div>
      <div class="pn-card">
        <div class="pn-card-title">三步开始</div>
        <div class="pn-empty-step"><i>1</i><span>让 Agent 产出文件：<code>model_export format=all</code> → <code>model_verify</code>。</span></div>
        <div class="pn-empty-step"><i>2</i><span>文件自动出现在左栏（历史/手工文件点顶部「仅工具产出」切到「扫描工作区」也能找到）。</span></div>
        <div class="pn-empty-step"><i>3</i><span>点文件名 → 当场实时预览；改工程或重导出后无需手点刷新。</span></div>
      </div>
      <div class="pn-tip">
        也可以直接双击导出的 <code>*.preview.html</code>（自包含，浏览器即可打开）。
      </div>
      <div class="pn-tip">{{ bridge.modeNote }}</div>
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

    <!-- 左栏：文件 / 部件 / 参数 / 复验 -->
    <template #side>
      <div v-if="tab === 'files'" class="pn-list">
        <button
          v-for="it in items"
          :key="it.path"
          class="pn-item"
          :class="{ 'pn-item-on': it.path === selected }"
          :title="it.path"
          @click="select(it.path)"
        >
          <span class="mp-kind" :class="'mp-kind-' + (it.kind || 'x')">{{ kindShort(it.kind) }}</span>
          <span class="pn-item-k mp-path">{{ shortPath(it.path) }}</span>
          <span class="mp-src">{{ srcLabel(it.source) }}</span>
          <span class="pn-item-v">{{ fmtSize(it.bytes) }}</span>
        </button>
        <div v-if="!items.length" class="pn-tip">没有识别到文件。</div>
        <div v-else class="pn-tip mp-side-hint">点文件名即在此登记并实时预览。</div>
      </div>

      <div v-else-if="tab === 'parts'" class="pn-list">
        <template v-if="partRows.length">
          <div v-for="p in partRows" :key="p.id" class="pn-kv">
            <span class="pn-mono">{{ p.id }}</span>
            <b class="pn-dim pn-mini">{{ p.type }}</b>
            <span class="pn-spacer"></span>
            <b>{{ p.tri }}</b>
          </div>
          <div class="pn-tip mp-side-hint">面数来自内核现场构建；体积见指标条与参数。</div>
        </template>
        <div v-else class="pn-tip">
          当前文件没有部件几何。选中 <code>model.json</code>（CAD 工程）或 glTF/GLB/STL/OBJ 才会出现。
        </div>
      </div>

      <div v-else-if="tab === 'params'" class="pn-list">
        <template v-if="projInfo && projInfo.params.length">
          <div v-for="p in projInfo.params" :key="p.id" class="pn-kv">
            <span class="pn-mono">{{ p.name || p.id }}</span>
            <span class="pn-spacer"></span>
            <b>{{ p.value }}<span v-if="p.range" class="pn-dim"> ({{ p.range }})</span></b>
          </div>
          <div class="pn-tip mp-side-hint">参数改动经 Agent 的 <code>model_edit</code>；面板只读。</div>
        </template>
        <div v-else class="pn-tip">未声明参数（尺寸为字面量），或当前文件不是 CAD 工程。</div>
      </div>

      <div v-else class="pn-list">
        <template v-if="verifyInfo">
          <div v-for="c in (verifyInfo.checks || [])" :key="c.id" class="pn-check">
            <span class="pn-dot" :class="{ 'pn-dot-ok': c.ok }"></span>
            <b>{{ c.id }}</b>
            <span class="pn-check-name">{{ c.title }}</span>
            <span class="pn-check-metric" :title="c.detail">{{ c.detail }}</span>
          </div>
          <div class="pn-tip mp-side-hint">判据 M1–M9，来自 <code>model_verify</code> 的复验报告。</div>
        </template>
        <div v-else class="pn-tip">
          当前文件不是复验报告。点左栏「文件」里的报告类文件（<code>*.verify.json</code>）即可看到 M1–M9。
        </div>
      </div>
    </template>

    <!-- 主区：预览 + 工程简介 + 指引 + 命令 -->
    <template #main>
      <div class="pn-scroll mp-main">
        <div class="pn-row-gap">
          <span class="pn-strong">{{ selected ? shortPath(selected) : '未选择文件' }}</span>
          <span class="pn-dim pn-mini">{{ metaLabel }}</span>
          <span class="pn-spacer"></span>
          <span v-if="selected" class="pn-dim pn-mini pn-mono">{{ fmtSize(meta.bytes) }}<span v-if="meta.mtime"> · {{ fmtTime(meta.mtime) }}</span></span>
          <span v-if="selected" class="pn-badge pn-badge-accent">实时</span>
        </div>

        <div class="mp-stage">
          <div v-if="!selected" class="pn-tip">点左栏「文件」里任一文件，即在此实时预览。</div>
          <div v-else-if="preview.loading" class="pn-tip">读取中…</div>
          <div v-else-if="preview.error" class="pn-msg-inline">{{ preview.error }}</div>
          <ModelViewer
            v-else-if="geo.length"
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
          <div v-else-if="verifyInfo" class="pn-tip">
            这是复验报告（无几何可渲染）。判据 M1–M9 见左栏「复验」分段 —— 通过 {{ passedCount }}/{{ (verifyInfo.checks || []).length }}。
          </div>
          <pre v-else-if="rawText" class="pn-code mp-raw">{{ rawText }}</pre>
          <div v-else class="pn-tip">该文件没有可渲染内容（可在文本编辑器中查看）。</div>
        </div>
        <div v-if="parseNote" class="pn-tip">{{ parseNote }}</div>

        <template v-if="projInfo">
          <div class="pn-card">
            <div class="pn-card-title">
              工程 {{ projInfo.name }}
              <span class="pn-dim">{{ projInfo.unit }} · {{ projInfo.up }}-up</span>
              <span class="pn-spacer"></span>
              <span class="pn-dim pn-mini">{{ projInfo.parts.length }} 部件 · {{ (projInfo.totalTris || 0).toLocaleString() }} 面</span>
            </div>
            <div class="pn-kv"><span>体积合计</span><b>{{ projInfo.totalVolume }} mm³</b></div>
            <div class="pn-kv"><span>包围盒</span><b class="pn-mono">{{ bboxText }}</b></div>
            <div class="pn-tip">
              几何由 host 半内核现场构建（同一份代码，与 Agent 看到的一致）；工程或参数一变，这里立即重绘。
            </div>
          </div>
        </template>

        <div class="pn-card">
          <div class="pn-card-title">
            这个面板是什么 · 怎么用
            <span class="pn-spacer"></span>
            <button class="pn-btn" @click="guideOpen = !guideOpen">{{ guideOpen ? '收起' : '展开' }}</button>
          </div>
          <template v-if="guideOpen">
            <div class="pn-tip">
              工具写出的每一个文件都会自动出现在左栏「文件」里；点文件名即登记预览，页面实时跟随文件变化重绘。
              从工程里打开预览的三种入口：① 主内容区顶部 tab「3D 模型」；② 本面板；③ 直接双击导出的 <code>*.preview.html</code>。
            </div>
            <div class="pn-tip">{{ bridge.modeNote }}</div>
          </template>
        </div>

        <div class="pn-card">
          <div class="pn-card-title">
            Agent 命令
            <span class="pn-spacer"></span>
            <button class="pn-btn" @click="copyCmd">{{ copied ? '已复制 ✓' : '复制' }}</button>
          </div>
          <pre class="pn-code">{{ sampleCmd }}</pre>
        </div>
      </div>
    </template>
  </PanelShell>
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
// 布局（2026-09-18 改版）：套 PanelShell 统一外壳 —— 顶栏 / 指标条 / 左栏分段（文件·部件·参数·复验）+ 主区预览 / 底栏。
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import PanelShell from './PanelShell.vue'
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
const tab = ref('files')

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
function srcLabel(s) {
  if (s === 'project') return '工具·工程'
  if (s === 'export') return '工具·产物'
  if (s === 'verify') return '工具·报告'
  if (s === 'claim') return '已登记'
  return '扫描'
}
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
const modeBadge = computed(() => (bridge.mode === 'invoke' ? '工具直连' : '只读降级'))
const metaLabel = computed(() => {
  const k = meta.value.kind
  return (KIND_LABEL[k] || kindShort(k)) + (preview.value.loading ? ' · 读取中' : '')
})
const passedCount = computed(() => ((verifyInfo.value && verifyInfo.value.checks) || []).filter((c) => c.ok).length)
const partRows = computed(() => {
  // 复验报告里也带部件（面数/体积），优先用它；否则用内核构建的几何
  const fromVerify = ((verifyInfo.value && verifyInfo.value.parts) || []).map((p) => ({
    id: p.id, type: p.type, tri: (p.triangles || 0).toLocaleString() + ' 面',
  }))
  if (fromVerify.length) return fromVerify
  return (projInfo.value && projInfo.value.parts || []).map((p) => ({
    id: p.id, type: p.type || '', tri: (p.triangles || p.tris || 0).toLocaleString() + ' 面',
  }))
})
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

// ── 指标条 ──
const metrics = computed(() => {
  const out = [{ k: '文件', v: String(items.value.length) }]
  if (projInfo.value) {
    out.push({ k: '部件', v: String(projInfo.value.parts.length) })
    out.push({ k: '三角面', v: (projInfo.value.totalTris || 0).toLocaleString() })
    out.push({ k: '体积', v: projInfo.value.totalVolume + ' mm³' })
  }
  if (verifyInfo.value) {
    out.push({ k: '复验', v: passedCount.value + '/' + ((verifyInfo.value.checks || []).length) })
  }
  if (meta.value.kind) out.push({ k: '当前', v: kindShort(meta.value.kind) })
  return out
})
const tabs = computed(() => [
  { id: 'files', name: '文件', count: items.value.length, title: '识别到的文件（点选即预览）' },
  { id: 'parts', name: '部件', count: partRows.value.length, title: '部件与面数' },
  { id: 'params', name: '参数', count: (projInfo.value && projInfo.value.params || []).length, title: '工程参数' },
  { id: 'verify', name: '复验', count: verifyInfo.value ? passedCount.value + '/' + ((verifyInfo.value.checks || []).length) : 0, title: '复验判据 M1–M9' },
])

// ── 数据流 ─────────────────────────────────────────────────
let offEvent = null
let timer = 0
let lastSig = ''
let stopFlag = false

function sigOf(it) { return String(it && (it.bytes + '|' + it.mtime)) }

function refreshNow() { return refresh(scanMode.value) }

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
/* 局部样式（其余复用 PanelShell 共享类） */
.mp-main { display: flex; flex-direction: column; gap: 10px; padding: 10px; }
.mp-stage { min-height: 260px; display: flex; flex-direction: column; }
.mp-stage > * { flex: 1; min-height: 0; }
/* 预览缺产物时的提示居中显示（.mp-stage 是 flex 列，直接留白会贴在左上角） */
.mp-stage > .pn-tip { flex: none; margin: auto; }
.mp-frame {
  display: block; width: 100%; min-height: 260px; background: #14161a;
  border: 1px solid var(--border-color); border-radius: 4px;
}
.mp-raw { max-height: 420px; overflow: auto; }
.mp-kind {
  flex: none; min-width: 42px; padding: 0 4px; font-size: 10px; text-align: center;
  color: var(--text-muted); border: 1px solid var(--border-color); border-radius: 3px;
}
.mp-kind-project { color: var(--text-secondary); }
.mp-src { flex: none; font-size: 10px; color: var(--text-muted); }
.mp-path { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mp-side-hint { margin-top: 6px; }
.mp-msg-inline { color: var(--text-primary); border-left: 2px solid var(--accent); padding-left: 8px; }
</style>
