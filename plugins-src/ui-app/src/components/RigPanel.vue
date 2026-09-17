<template>
  <PanelShell
    v-model:file="projectPath"
    title="角色"
    icon="user"
    placeholder="rig.model.iki.json"
    :badge="verifyBadge"
    :badge-tone="verifyTone"
    :metrics="metrics"
    :loading="loading"
    :error="model ? error : ''"
    :empty="!model"
    reload-title="重新载入工程、预览与校验报告"
    footnote="只读面板：部件/骨骼/绑定编辑一律经 Agent 的 rig_edit（命令式 op）；预览 iframe 可拖参数滑块试效果，不写工作区。"
    :source="projectPath"
    @reload="load"
  >
    <template #empty>
      <div class="pn-empty-title">还没有角色工程</div>
      <div class="pn-tip">
        面板读工作区里的 <code>rig.model.iki.json</code>（.iki 上游格式，与 Agent 共用同一份真相源），当前没有这个文件。
      </div>
      <div class="pn-card">
        <div class="pn-card-title">三步开始</div>
        <div class="pn-empty-step"><i>1</i><span>让 Agent 执行 <code>rig_model action=create</code> 建空工程，或 <code>rig_import source=psd src=角色.psd</code> 自动分层。</span></div>
        <div class="pn-empty-step"><i>2</i><span>用 <code>rig_edit</code> 绑骨骼与参数（bind.auto / part.bind …）。</span></div>
        <div class="pn-empty-step"><i>3</i><span><code>rig_export format=html</code> 出动态预览、<code>rig_verify</code> 生成校验报告。</span></div>
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

    <!-- 左栏：参数 / 骨骼（含 warp 与物理）/ 部件 / 校验 -->
    <template #side>
      <div v-if="tab === 'params'" class="pn-list">
        <div v-for="p in paramRows" :key="p.id" class="pn-kv">
          <span class="pn-mono pn-mini">{{ p.name || p.id }}</span>
          <span class="pn-spacer"></span>
          <b class="pn-mini">{{ p.min }}..{{ p.max }}<span class="pn-dim"> 默认 {{ p.default }}</span></b>
        </div>
        <div v-if="!paramRows.length" class="pn-tip">工程里还没有参数。</div>
        <div v-else class="pn-tip mp-side-hint">
          <span class="pn-tag">标准</span> = 上游标准参数；<span class="pn-tag">物理输出</span> = 由二级运动驱动（宿主不直接设）。
        </div>
        <div class="mp-side-block">
          <div class="pn-card-title">驱动来源</div>
          <div class="pn-tip">{{ driverText }}</div>
        </div>
      </div>

      <div v-else-if="tab === 'bones'" class="pn-list">
        <div v-for="b in boneRows" :key="b.id" class="rp-bone" :style="{ paddingLeft: (b.depth * 12) + 'px' }">
          <span class="pn-mono pn-mini">{{ b.id }}</span>
          <span class="pn-dim pn-mini"> pivot({{ b.pivot }}) {{ b.bindText }}</span>
        </div>
        <div v-if="!boneRows.length" class="pn-tip">工程里还没有骨骼（matrix deformer）。</div>
        <div class="mp-side-block">
          <div class="pn-card-title">warp 形变</div>
          <div class="pn-tip">{{ warpText }}</div>
        </div>
        <div v-if="physics.length || chains.length" class="mp-side-block">
          <div class="pn-card-title">二级运动</div>
          <div v-for="s in physics" :key="s.id" class="pn-kv">
            <span class="pn-mono pn-mini">{{ s.id }}</span>
            <b class="pn-dim pn-mini mp-wrap">{{ s.text }}</b>
          </div>
          <div v-for="c in chains" :key="c.id" class="pn-kv">
            <span class="pn-mono pn-mini">{{ c.id }}</span>
            <b class="pn-dim pn-mini mp-wrap">{{ c.text }}</b>
          </div>
          <div class="pn-tip">输出参数由物理驱动，宿主不直接设。</div>
        </div>
      </div>

      <div v-else-if="tab === 'parts'" class="pn-list">
        <div v-for="p in partRows" :key="p.id" class="pn-kv">
          <span>
            <span class="rp-idx">{{ p.order }}</span>
            <span class="pn-mono pn-mini">{{ p.id }}</span>
            <span class="pn-dim pn-mini"> {{ p.role }}</span>
          </span>
          <span class="pn-spacer"></span>
          <b class="pn-mini">
            {{ p.size }}
            <span class="pn-dim">{{ p.deformer || '未挂骨骼' }}</span>
            <span v-if="p.binds" class="pn-tag">{{ p.binds }}</span>
            <span v-if="p.mesh" class="pn-tag">网格</span>
          </b>
        </div>
        <div v-if="!partRows.length" class="pn-tip">工程里还没有部件。</div>
        <div v-else class="pn-tip mp-side-hint">绘制顺序自后向前（数字小 = 更靠后）。</div>
      </div>

      <div v-else class="pn-list">
        <template v-if="verify">
          <div v-for="c in verify.checks" :key="c.id" class="pn-check">
            <span class="pn-dot" :class="{ 'pn-dot-ok': c.pass }"></span>
            <b>{{ c.id }}</b>
            <span class="pn-check-name">{{ c.name }}</span>
            <span class="pn-check-metric" :title="c.metric">{{ c.metric }}</span>
          </div>
          <div v-if="warnChecks.length" class="pn-tip pn-warn">警告：{{ warnChecks.join('；') }}</div>
          <div class="pn-tip mp-side-hint">报告时间：{{ verify.ts }}</div>
        </template>
        <div v-else class="pn-tip">
          还没有校验报告。让 Agent 执行 <code>rig_verify</code> 生成旁挂的 rig.verify.json。
        </div>
      </div>
    </template>

    <!-- 主区：动态预览 + 命令 -->
    <template #main>
      <div class="pn-scroll rp-main">
        <div class="pn-row-gap">
          <span class="pn-strong">{{ model.name }}</span>
          <span class="pn-dim pn-mini pn-mono">{{ canvasText }}</span>
          <span class="pn-spacer"></span>
          <span class="pn-dim pn-mini pn-mono">{{ previewReady ? 'rig.preview.html' : '未导出' }}</span>
        </div>
        <div class="rp-stage">
          <iframe
            v-if="previewHtml"
            class="rp-frame"
            :srcdoc="previewHtml"
            sandbox="allow-scripts"
            referrerpolicy="no-referrer"
          ></iframe>
          <div v-else class="pn-tip rp-stage-empty">
            未找到预览产物。让 Agent 执行 <code>rig_export format=html</code> 生成（默认 rig.preview.html）。
          </div>
        </div>
        <div class="pn-tip">
          iframe 内可拖参数滑块、开关演示动画（呼吸/眨眼）—— 预览需要脚本，故 sandbox 给了 allow-scripts
          （但无 allow-same-origin，它拿不到宿主 DOM）；本面板不写工作区。
        </div>
        <div class="pn-tip">
          .iki v{{ model.version }}（上游 @ikijs/format 契约）｜模型空间原点 = 画布中心、+y 向上。
        </div>
        <div class="pn-card">
          <div class="pn-card-title">
            交给 Agent 执行
            <span class="pn-spacer"></span>
            <button class="pn-btn" @click="copyCmd">{{ copied ? '已复制 ✓' : '复制命令' }}</button>
          </div>
          <pre class="pn-code">{{ sampleCmd }}</pre>
        </div>
      </div>
    </template>
  </PanelShell>
</template>

<script setup>
// RigPanel — 2D 角色工程只读视图（与 Agent 共用真相源：rig.model.iki.json + rig.preview.html + rig.verify.json）
// 设计取舍（与 tool-design / tool-art / tool-music 一致）：面板不写工作区 —— 部件/骨骼/绑定编辑一律经 Agent 的
// rig_edit（命令式 op），避免"工具写工程 / 面板写工程"两条写路径导致真相源分叉。
// 预览 iframe 用 sandbox="allow-scripts"（无 allow-same-origin）：预览内需要脚本才能拖参数滑块、
// 跑弹簧/角链二级运动，但它拿不到宿主 DOM，且产物本身无外链。
// 布局（2026-09-18 改版）：套 PanelShell 统一外壳 —— 顶栏 / 指标条 / 左栏分段（参数·骨骼·部件·校验）+ 主区预览 / 底栏。
import { ref, computed, onMounted } from 'vue'
import api from '../api.js'
import PanelShell from './PanelShell.vue'
import { state } from '../ui-state.js'

const STANDARD = ['ParamMouthOpenY', 'ParamMouthForm', 'ParamEyeLOpen', 'ParamEyeROpen', 'ParamEyeBallX', 'ParamEyeBallY',
  'ParamAngleX', 'ParamAngleY', 'ParamAngleZ', 'ParamBreath', 'ParamBrowLY', 'ParamBrowRY', 'ParamBrowLAngle',
  'ParamBrowRAngle', 'ParamHairSwayX', 'ParamHairSwayZ']
const PHYSICS_OUT = ['ParamHairSwayX', 'ParamHairSwayZ']

const projectPath = ref('rig.model.iki.json')
const verifyPath = ref('rig.verify.json')
const previewPath = ref('rig.preview.html')
const loading = ref(false)
const error = ref('')
const model = ref(null)
const verify = ref(null)
const previewHtml = ref('')
const copied = ref(false)
const tab = ref('params')

const canvasText = computed(() => {
  const c = (model.value && model.value.canvas) || {}
  return (c.width || '?') + '×' + (c.height || '?')
})
const deformers = computed(() => (model.value && model.value.deformers) || [])
const physics = computed(() => ((model.value && model.value.physics) || []).map((p) => ({
  id: p.id,
  text: p.input.parameter + ' ×' + p.input.weight + ' → ' + p.output.parameter + ' ×' + p.output.scale
    + '（mass ' + p.mass + '，stiffness ' + p.stiffness + '，damping ' + p.damping + '）',
})))
const chains = computed(() => ((model.value && model.value.physicsChains) || []).map((c) => ({
  id: c.id,
  text: '锚 ' + c.anchorDeformer + '，' + c.segments.length + ' 段 → '
    + c.segments.map((s) => s.output.parameter + (s.scale !== 1 ? ' ×' + s.scale : '')).join(' / ')
    + '（重力 ' + c.gravity.angle + '° ×' + c.gravity.strength + '）',
})))
const previewReady = computed(() => previewHtml.value.length > 0)
const passedAll = computed(() => ((verify.value && verify.value.checks) || []).every((c) => c.pass))
const passedCount = computed(() => ((verify.value && verify.value.checks) || []).filter((c) => c.pass).length)
const verifyTotal = computed(() => ((verify.value && verify.value.checks) || []).length)
const verifyBadge = computed(() => (verify.value ? '校验 ' + passedCount.value + '/' + verifyTotal.value : '未校验'))
const verifyTone = computed(() => (!verify.value ? 'muted' : (passedAll.value && verifyTotal.value > 0 ? 'ok' : 'bad')))
const warnChecks = computed(() => ((verify.value && verify.value.checks) || [])
  .filter((c) => c.warns > 0).map((c) => c.id + ' ' + c.name + '（' + c.warns + ' 条）'))

const paramRows = computed(() => ((model.value && model.value.parameters) || []).map((p) => ({
  id: p.id, name: p.name, min: p.min, max: p.max, default: p.default,
  standard: STANDARD.indexOf(p.id) >= 0, physicsOut: PHYSICS_OUT.indexOf(p.id) >= 0,
})))

// 驱动来源索引：参数 → 谁在用它（骨骼绑定 / 部件绑定 / warp / 物理输出）
const drivers = computed(() => {
  const idx = {}
  const push = (pid, who) => { if (!idx[pid]) idx[pid] = []; idx[pid].push(who) }
  const m = model.value
  if (!m) return idx
  for (const p of m.parts || []) {
    for (const b of p.bindings || []) push(b.parameter, p.id + '.' + b.channel)
    for (const w of p.warps || []) push(w.parameter, p.id + '.warp')
  }
  for (const d of m.deformers || []) {
    for (const b of d.bindings || []) push(b.parameter, d.id + '.' + b.channel)
    for (const w of d.warps || []) push(w.parameter, d.id + '.warp')
    if (d.warp2d) { push(d.warp2d.parameter, d.id + '.warp2d'); push(d.warp2d.parameterY, d.id + '.warp2d') }
  }
  for (const s of m.physics || []) push(s.output.parameter, '弹簧 ' + s.id)
  for (const c of m.physicsChains || []) for (const sg of c.segments || []) push(sg.output.parameter, '角链 ' + c.id)
  return idx
})
const driverText = computed(() => {
  const rows = []
  for (const p of paramRows.value) {
    const d = drivers.value[p.id]
    if (d) rows.push((p.name || p.id).replace(/（.*?）/g, '') + ' ← ' + d.slice(0, 3).join('、') + (d.length > 3 ? ' 等 ' + d.length + ' 处' : ''))
  }
  return rows.length ? rows.slice(0, 6).join('；') + (rows.length > 6 ? '…' : '') : '（暂无参数被引用）'
})

// 骨骼树：按父链缩进（同层按 id 稳定排序）
const boneRows = computed(() => {
  const list = deformers.value.filter((d) => d.kind === undefined || d.kind === 'matrix')
  const byId = {}
  for (const d of list) byId[d.id] = d
  const rows = []
  const walk = (d, depth) => {
    const binds = (d.bindings || []).map((b) => b.parameter + '→' + b.channel)
    rows.push({
      id: d.id,
      depth,
      pivot: (d.pivot ? d.pivot.x + ',' + d.pivot.y : '0,0'),
      bindText: binds.length ? '｜' + binds.join(' ') : '',
    })
    for (const kid of list.filter((x) => x.parent === d.id)) walk(kid, depth + 1)
  }
  for (const root of list.filter((d) => !d.parent || !byId[d.parent])) walk(root, 0)
  return rows
})
const warpText = computed(() => {
  const ws = deformers.value.filter((d) => d.kind === 'warp')
  if (!ws.length) return '无（未做网格形变）'
  return ws.map((d) => d.id + '（grid ' + (d.grid.cols + 1) + '×' + (d.grid.rows + 1)
    + (d.warps && d.warps.length ? '，← ' + d.warps[0].parameter : '') + '）').join('；')
})

const ROLE_LABEL = {
  hair: '头发', head: '头', face: '脸', neck: '脖子', body: '身体', cloth: '衣物', accessory: '配件',
  mouth: '嘴', brow: '眉', eyelid: '眼睑', eyeball: '眼球', arm: '手臂', hand: '手', leg: '腿', unclassified: '未分类',
}
function roleLabel(id) {
  const cls = classify(id)
  return ROLE_LABEL[cls] || cls
}
// 与服务端一致的部位词根判定（仅用于展示分组）
function classify(id) {
  const n = String(id || '')
  if (/hair|ahoge|ponytail|braid|bang/.test(n)) return 'hair'
  if (/iris|pupil|eyeball|highlight/.test(n)) return 'eyeball'
  if (/eyelid|lash|eyewhite|eye_white|eye($|_)/.test(n)) return 'eyelid'
  if (/brow/.test(n)) return 'brow'
  if (/mouth|lips|teeth|tongue|jaw/.test(n)) return 'mouth'
  if (/(^|_)head|(^|_)face($|_)|skull/.test(n)) return 'head'
  if (/neck/.test(n)) return 'neck'
  if (/arm|shoulder|elbow|forearm|wrist/.test(n)) return 'arm'
  if (/hand|finger|palm/.test(n)) return 'hand'
  if (/(thigh|leg|knee|shin|calf|foot|toe|ankle|shoe)/.test(n)) return 'leg'
  if (/(cloth|skirt|dress|coat|cape|scarf|tie|ribbon|belt|sleeve|pants)/.test(n)) return 'cloth'
  if (/(tail|wing|horn|accessor|weapon|sword|staff|prop)/.test(n)) return 'accessory'
  if (/(body|torso|trunk|chest|hip|waist|spine|back)/.test(n)) return 'body'
  return '未分类'
}
const partRows = computed(() => ((model.value && model.value.parts) || [])
  .slice().sort((a, b) => a.order - b.order)
  .map((p) => ({
    id: p.id,
    order: p.order,
    role: roleLabel(p.id),
    size: Math.round(p.width) + '×' + Math.round(p.height),
    deformer: p.deformer || '',
    binds: (p.bindings || []).length || 0,
    mesh: !!p.mesh,
  })))

// ── 指标条 ──
const metrics = computed(() => {
  const m = model.value
  if (!m) return []
  return [
    { k: '画布', v: canvasText.value, mono: true },
    { k: '部件', v: String((m.parts || []).length) },
    { k: '骨骼', v: String(deformers.value.length) },
    { k: '参数', v: String((m.parameters || []).length) },
    { k: '物理', v: physics.value.length + ' 弹簧 · ' + chains.value.length + ' 角链' },
    { k: '格式', v: '.iki v' + m.version },
  ]
})
const tabs = computed(() => [
  { id: 'params', name: '参数', count: paramRows.value.length, title: '参数范围 / 默认值 / 是否标准参数 / 驱动来源' },
  { id: 'bones', name: '骨骼', count: boneRows.value.length, title: '骨骼树（含 warp 形变与二级运动）' },
  { id: 'parts', name: '部件', count: partRows.value.length, title: '部件与绘制顺序' },
  { id: 'checks', name: '校验', count: verify.value ? passedCount.value + '/' + verifyTotal.value : 0, title: '工程校验报告' },
])

const sampleCmd = computed(() => {
  const first = (model.value && model.value.parts && model.value.parts[0]) || {}
  return JSON.stringify({ op: 'bind.auto' }) + '\n'
    + JSON.stringify({ op: 'part.bind', id: first.id || 'hair_front', parameter: 'ParamAngleZ', channel: 'rotate', from: -3, to: 3 }) + '\n'
    + '# 校验 + 出预览\nrig_verify\nrig_export format=html overwrite=true'
})

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
    model.value = JSON.parse(await readText(projectPath.value))
    verify.value = null
    const vTxt = await readIfExists(verifyPath.value)
    if (vTxt) { try { verify.value = JSON.parse(vTxt) } catch (_) { /* 报告可选 */ } }
    previewHtml.value = await readIfExists(previewPath.value)
  } catch (e) {
    model.value = null
    previewHtml.value = ''
    verify.value = null
    error.value = '读取角色工程失败：' + ((e && e.message) || e) + '（路径 ' + projectPath.value + '）'
  } finally {
    loading.value = false
  }
}
function copyCmd() {
  const text = sampleCmd.value
  try {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(() => { copied.value = true; setTimeout(() => { copied.value = false }, 1500) })
      return
    }
  } catch (_) { /* 回退 */ }
  copied.value = true
  setTimeout(() => { copied.value = false }, 1500)
}
onMounted(load)
</script>

<style scoped>
/* 局部样式（其余复用 PanelShell 共享类；配色取设计系统变量） */
.rp-main { display: flex; flex-direction: column; gap: 10px; padding: 10px; }
.rp-stage { flex: 1; min-height: 300px; display: flex; flex-direction: column; }
.rp-stage > * { flex: 1; min-height: 0; }
.rp-frame {
  display: block; width: 100%; min-height: 300px; background: #fff;
  border: 1px solid var(--border-color); border-radius: 4px;
}
.rp-stage-empty { margin: auto; }
.rp-bone { padding: 1px 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; line-height: 1.7; }
.rp-idx {
  display: inline-block; min-width: 16px; margin-right: 4px; text-align: right;
  color: var(--text-muted); font-variant-numeric: tabular-nums; font-size: 11px;
}
.pn-tag {
  margin-left: 4px; padding: 0 4px; font-size: 10px; color: var(--text-muted);
  border: 1px solid var(--border-color); border-radius: 3px;
}
.mp-wrap { white-space: normal; overflow-wrap: anywhere; text-align: left; }
.mp-side-hint { margin-top: 6px; }
.mp-side-block { margin-top: 10px; display: flex; flex-direction: column; gap: 4px; }
.pn-warn { color: var(--text-primary); }
</style>
