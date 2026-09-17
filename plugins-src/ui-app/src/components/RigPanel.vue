<template>
  <div class="rp-panel">
    <div class="rp-bar">
      <span class="rp-title">角色</span>
      <input v-model="projectPath" class="rp-input" placeholder="rig.model.iki.json" @keyup.enter="load" />
      <button class="rp-icon-btn" :disabled="loading" title="重新载入工程、预览与校验报告" @click="load">
        <SvgIcon name="refresh" :size="12" :class="{ spinning: loading }" />
      </button>
    </div>

    <div v-if="error" class="rp-msg rp-err">{{ error }}</div>
    <div v-if="!model && !error" class="rp-msg">
      未找到角色工程。先让 Agent 执行 <code>rig_model action=create</code>，或直接
      <code>rig_import source=psd src=角色.psd</code>（自动分层 + 按命名约定自动绑定）。
    </div>

    <template v-if="model">
      <!-- 概要 -->
      <section class="rp-sec">
        <div class="rp-sec-title">{{ model.name }}</div>
        <div class="rp-kv"><span>画布</span><b>{{ canvasText }}</b></div>
        <div class="rp-kv"><span>部件 / 骨骼</span><b>{{ model.parts.length }} · {{ deformers.length }}</b></div>
        <div class="rp-kv"><span>参数 / 物理</span><b>{{ model.parameters.length }} · {{ physics.length }} 弹簧 + {{ chains.length }} 角链</b></div>
        <div class="rp-dim">
          .iki v{{ model.version }}（上游 @ikijs/format 契约）｜模型空间原点 = 画布中心、+y 向上｜拖右侧预览里的滑块即可试参数。
        </div>
      </section>

      <!-- 预览：动态预览含参数滑块，必须允许脚本（sandbox=allow-scripts，无 same-origin） -->
      <section class="rp-sec">
        <div class="rp-row">
          <span class="rp-sec-title">预览</span>
          <span class="rp-spacer"></span>
          <span class="rp-dim">{{ previewReady ? 'rig.preview.html' : '未导出' }}</span>
        </div>
        <iframe v-if="previewHtml" class="rp-frame" :srcdoc="previewHtml" sandbox="allow-scripts" referrerpolicy="no-referrer"></iframe>
        <div v-else class="rp-dim rp-msg">
          未找到预览产物。让 Agent 执行 <code>rig_export format=html</code> 生成（默认 rig.preview.html）。
        </div>
        <div class="rp-dim">iframe 内可拖参数滑块、开关演示动画（呼吸/眨眼）；本面板只读，不写工作区。</div>
      </section>

      <!-- 参数：范围 + 驱动来源 -->
      <section class="rp-sec">
        <div class="rp-sec-title">参数 <span class="rp-dim">（{{ model.parameters.length }} 项；标注是否上游标准参数与被谁驱动）</span></div>
        <div v-for="p in paramRows" :key="p.id" class="rp-kv">
          <span class="rp-mono">{{ p.name || p.id }}</span>
          <b>
            {{ p.min }}..{{ p.max }} <span class="rp-dim">默认 {{ p.default }}</span>
            <span v-if="p.standard" class="rp-tag">标准</span>
            <span v-if="p.physicsOut" class="rp-tag">物理输出</span>
          </b>
        </div>
        <div class="rp-dim">驱动来源：{{ driverText }}</div>
      </section>

      <!-- 骨骼树（= .iki 的 matrix deformer 树，转轴 pivot 决定旋转中心） -->
      <section class="rp-sec">
        <div class="rp-sec-title">骨骼树 <span class="rp-dim">（deformer；缩进 = 父子层级，括号内为转轴与参数绑定）</span></div>
        <div v-for="b in boneRows" :key="b.id" class="rp-bone" :style="{ paddingLeft: (b.depth * 12) + 'px' }">
          <span class="rp-mono">{{ b.id }}</span>
          <span class="rp-dim"> pivot({{ b.pivot }}) {{ b.bindText }}</span>
        </div>
        <div class="rp-dim">warp 形变：{{ warpText }}</div>
      </section>

      <!-- 部件（绘制顺序自后向前） -->
      <section class="rp-sec">
        <div class="rp-sec-title">部件 <span class="rp-dim">（{{ model.parts.length }} 个，绘制顺序自后向前）</span></div>
        <div v-for="p in partRows" :key="p.id" class="rp-kv">
          <span>
            <span class="rp-idx">{{ p.order }}</span>
            <span class="rp-mono">{{ p.id }}</span>
            <span class="rp-dim"> {{ p.role }}</span>
          </span>
          <b>
            {{ p.size }}
            <span class="rp-dim">{{ p.deformer || '未挂骨骼' }}</span>
            <span v-if="p.binds" class="rp-tag">{{ p.binds }}</span>
            <span v-if="p.mesh" class="rp-tag">网格</span>
          </b>
        </div>
      </section>

      <!-- 物理 -->
      <section v-if="physics.length || chains.length" class="rp-sec">
        <div class="rp-sec-title">二级运动 <span class="rp-dim">（弹簧 / 多段角链；输出参数由物理驱动，宿主不直接设）</span></div>
        <div v-for="s in physics" :key="s.id" class="rp-kv">
          <span class="rp-mono">{{ s.id }}</span>
          <b class="rp-dim">{{ s.text }}</b>
        </div>
        <div v-for="c in chains" :key="c.id" class="rp-kv">
          <span class="rp-mono">{{ c.id }}</span>
          <b class="rp-dim">{{ c.text }}</b>
        </div>
      </section>

      <!-- 校验（旁挂报告） -->
      <section v-if="verify" class="rp-sec">
        <div class="rp-row">
          <span class="rp-sec-title">校验</span>
          <span class="rp-spacer"></span>
          <span :class="['rp-badge', passedAll ? 'rp-ok' : 'rp-bad']">{{ passedCount }}/{{ verify.checks.length }}</span>
        </div>
        <div v-for="c in verify.checks" :key="c.id" class="rp-check">
          <span :class="['rp-dot', c.pass ? 'rp-ok' : 'rp-bad']"></span>
          <b>{{ c.id }}</b>
          <span class="rp-check-name">{{ c.name }}</span>
          <span class="rp-dim rp-check-metric" :title="c.metric">{{ c.metric }}</span>
        </div>
        <div v-if="warnChecks.length" class="rp-dim">
          警告：{{ warnChecks.join('；') }}
        </div>
        <div class="rp-dim">报告时间：{{ verify.ts }}</div>
      </section>

      <!-- 面板只读：改动交 Agent -->
      <section class="rp-sec">
        <div class="rp-sec-title">交给 Agent 执行</div>
        <div class="rp-dim">面板不写工作区（避免第二条写路径）；常用命令：</div>
        <pre class="rp-code">{{ sampleCmd }}</pre>
        <button class="rp-btn" @click="copyCmd">{{ copied ? '已复制' : '复制命令' }}</button>
      </section>
    </template>
  </div>
</template>

<script setup>
// RigPanel — 2D 角色工程只读视图（与 Agent 共用真相源：rig.model.iki.json + rig.preview.html + rig.verify.json）
// 设计取舍（与 tool-design / tool-art / tool-music 一致）：面板不写工作区 —— 部件/骨骼/绑定编辑一律经 Agent 的
// rig_edit（命令式 op），避免"工具写工程 / 面板写工程"两条写路径导致真相源分叉。
// 预览 iframe 用 sandbox="allow-scripts"（无 allow-same-origin）：预览内需要脚本才能拖参数滑块、
// 跑弹簧/角链二级运动，但它拿不到宿主 DOM，且产物本身无外链。
import { ref, computed, onMounted } from 'vue'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'
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
.rp-panel { padding: 8px 10px; font-size: 12px; color: var(--text-primary); overflow-y: auto; height: 100%; }
.rp-bar { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
.rp-title { font-size: 12px; font-weight: 500; color: var(--text-secondary); flex: none; }
.rp-input {
  flex: 1; min-width: 0; padding: 2px 6px; font-size: 11px; color: var(--text-primary);
  background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 4px;
  font-family: ui-monospace, Consolas, monospace;
}
.rp-icon-btn {
  flex: none; display: inline-flex; align-items: center; justify-content: center; width: 22px; height: 22px;
  color: var(--text-muted); background: transparent; border: 1px solid var(--border-color);
  border-radius: 4px; cursor: pointer;
}
.rp-icon-btn:hover:not(:disabled) { background: var(--bg-hover); }
.rp-icon-btn:disabled { cursor: default; opacity: 0.6; }
.rp-msg { color: var(--text-muted); padding: 6px 2px; line-height: 1.6; }
.rp-err { color: var(--text-primary); }
.rp-sec { margin-bottom: 12px; padding-bottom: 10px; border-bottom: 1px solid var(--border-color); }
.rp-sec:last-child { border-bottom: none; }
.rp-sec-title { font-size: 12px; font-weight: 500; color: var(--text-secondary); margin-bottom: 6px; }
.rp-kv { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; padding: 2px 0; }
.rp-kv > span { color: var(--text-muted); flex: none; }
.rp-kv > b { color: var(--text-primary); font-weight: 500; text-align: right; overflow: hidden; text-overflow: ellipsis; }
.rp-mono { font-family: ui-monospace, Consolas, monospace; }
.rp-dim { color: var(--text-muted); font-weight: 400; font-size: 11px; line-height: 1.6; }
.rp-row { display: flex; align-items: center; gap: 6px; }
.rp-spacer { flex: 1; }
.rp-btn {
  padding: 3px 10px; font-size: 11px; color: var(--text-secondary);
  background: transparent; border: 1px solid var(--border-color); border-radius: 4px; cursor: pointer;
}
.rp-btn:hover { background: var(--bg-hover); }
.rp-frame {
  display: block; width: 100%; height: 520px; background: #fff;
  border: 1px solid var(--border-color); border-radius: 4px;
}
.rp-bone { padding: 1px 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.rp-idx {
  display: inline-block; min-width: 16px; margin-right: 4px; text-align: right;
  color: var(--text-muted); font-variant-numeric: tabular-nums;
}
.rp-tag {
  margin-left: 4px; padding: 0 4px; font-size: 10px; color: var(--text-muted);
  border: 1px solid var(--border-color); border-radius: 3px;
}
.rp-code {
  margin: 0 0 6px; padding: 6px; font-family: ui-monospace, Consolas, monospace;
  font-size: 10px; line-height: 1.5; color: var(--text-secondary);
  background: var(--bg-primary); border: 1px solid var(--border-color);
  border-radius: 4px; overflow-x: auto; white-space: pre;
}
.rp-badge { padding: 1px 6px; border-radius: 8px; font-size: 11px; }
.rp-dot { width: 7px; height: 7px; border-radius: 50%; flex: none; display: inline-block; }
.rp-ok { background: var(--accent); color: var(--bg-primary); }
.rp-bad { background: var(--text-muted); color: var(--bg-primary); }
.rp-check { display: flex; align-items: center; gap: 6px; padding: 1px 0; }
.rp-check > b { flex: none; color: var(--text-secondary); }
.rp-check-name { flex: none; }
.rp-check-metric { flex: 1; text-align: right; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
