<template>
  <div class="mp-panel">
    <div class="mp-bar">
      <span class="mp-title">3D 模型</span>
      <input v-model="projectPath" class="mp-input" placeholder="model.json" @keyup.enter="load" />
      <button class="mp-icon-btn" :disabled="loading" title="重新载入工程、预览与判据报告" @click="load">
        <SvgIcon name="refresh" :size="12" :class="{ spinning: loading }" />
      </button>
    </div>

    <div v-if="error" class="mp-msg mp-err">{{ error }}</div>
    <div v-if="!model && !error" class="mp-msg">
      未找到 CAD 工程。先让 Agent 执行 <code>model_doc action=new name=part</code>，
      再 <code>model_add id=base type=cuboid size=[40,40,10]</code>，
      最后 <code>model_export format=all</code> + <code>model_verify</code>。
    </div>

    <template v-if="model">
      <section class="mp-sec">
        <div class="mp-sec-title">{{ model.name || '未命名' }}</div>
        <div class="mp-kv"><span>单位 / 坐标</span><b>{{ model.unit || 'mm' }} · {{ model.up || 'z' }}-up</b></div>
        <div class="mp-kv"><span>部件 / 三角面</span><b>{{ partRows.length }} · {{ triTotal }}</b></div>
        <div class="mp-kv"><span>参数</span><b>{{ paramRows.length }}</b></div>
        <div v-if="summary" class="mp-kv"><span>体积合计</span><b>{{ summary.volume }} mm³</b></div>
        <div v-if="summary" class="mp-kv"><span>包围盒</span><b>{{ bboxText }}</b></div>
        <div class="mp-dim">
          工程即模型：几何字段用表达式引用参数（改参数 → 全模型自动重算）；产物是公开格式
          glTF 2.0 / binary STL / OBJ，预览由 tool-model 生成的自包含 WebGL 页面渲染。
        </div>
      </section>

      <section class="mp-sec">
        <div class="mp-row">
          <span class="mp-sec-title">3D 预览</span>
          <span class="mp-spacer"></span>
          <span class="mp-dim">{{ previewReady ? 'model.preview.html' : '未导出' }}</span>
        </div>
        <iframe v-if="previewHtml" class="mp-frame" :srcdoc="previewHtml" sandbox="allow-scripts" referrerpolicy="no-referrer"></iframe>
        <div v-else class="mp-dim mp-msg">
          未找到预览产物。让 Agent 执行 <code>model_export format=html</code>（默认 model.preview.html）。
        </div>
        <div class="mp-dim">iframe 内可拖拽旋转、滚轮缩放、右键平移、<code>W</code> 切线框。预览页自包含，无外链。</div>
      </section>

      <section class="mp-sec">
        <div class="mp-sec-title">部件 <span class="mp-dim">（{{ partRows.length }} 个）</span></div>
        <div v-for="p in partRows" :key="p.id" class="mp-kv">
          <span>
            <span class="mp-idx">{{ p.idx }}</span>
            <span class="mp-mono">{{ p.id }}</span>
            <span class="mp-dim"> {{ p.type }}</span>
            <span v-if="p.ops" class="mp-tag">{{ p.ops }} 运算</span>
            <span v-if="p.repeat" class="mp-tag">{{ p.repeat }} 阵列</span>
            <span v-if="!p.visible" class="mp-tag">隐藏</span>
          </span>
          <b>
            <span v-if="p.triangles">{{ p.triangles.toLocaleString() }} 面</span>
            <span v-else class="mp-dim">未构建</span>
            <span v-if="p.size" class="mp-dim"> {{ p.size }}</span>
          </b>
        </div>
        <div v-if="!partRows.length" class="mp-dim">还没有部件 —— 用 <code>model_add</code> 加几何。</div>
      </section>

      <section class="mp-sec">
        <div class="mp-sec-title">参数 <span class="mp-dim">（{{ paramRows.length }} 项；值可写表达式引用其它参数）</span></div>
        <div v-for="p in paramRows" :key="p.id" class="mp-kv">
          <span class="mp-mono">{{ p.name || p.id }}</span>
          <b>
            {{ p.value }}
            <span v-if="p.range" class="mp-dim">({{ p.range }})</span>
            <span v-if="p.refs" class="mp-tag">引用 {{ p.refs }}</span>
          </b>
        </div>
        <div v-if="!paramRows.length" class="mp-dim">未声明参数（尺寸为字面量）。</div>
      </section>

      <section class="mp-sec">
        <div class="mp-row">
          <span class="mp-sec-title">判据 M1–M9</span>
          <span class="mp-spacer"></span>
          <span v-if="verify" class="mp-badge" :class="verify.ok ? 'mp-ok' : 'mp-bad'">{{ passedCount }}/9</span>
        </div>
        <div v-if="!verify" class="mp-dim">未找到判据报告（model.verify.json）。让 Agent 跑 <code>model_verify</code>。</div>
        <template v-else>
          <div v-for="c in checks" :key="c.id" class="mp-check">
            <span class="mp-dot" :class="c.ok ? 'mp-ok' : (c.level === 'fail' ? 'mp-bad' : 'mp-warn')"></span>
            <b>{{ c.id }}</b>
            <span class="mp-check-name">{{ c.title }}</span>
            <span class="mp-check-metric mp-dim">{{ c.detail }}</span>
          </div>
          <div v-if="problemItems.length" class="mp-dim mp-items">
            <div v-for="(it, i) in problemItems" :key="i">· {{ it }}</div>
          </div>
        </template>
      </section>

      <section class="mp-sec">
        <div class="mp-sec-title">产物</div>
        <div v-if="!products.length" class="mp-dim">未发现导出产物。让 Agent 跑 <code>model_export format=all</code>。</div>
        <div v-for="p in products" :key="p" class="mp-kv"><span class="mp-mono">{{ shortPath(p) }}</span><b class="mp-dim">{{ kindOf(p) }}</b></div>
      </section>

      <section class="mp-sec">
        <div class="mp-row">
          <span class="mp-sec-title">Agent 命令</span>
          <span class="mp-spacer"></span>
          <button class="mp-btn" @click="copyCmd">{{ copied ? '已复制' : '复制' }}</button>
        </div>
        <pre class="mp-code">{{ sampleCmd }}</pre>
      </section>
    </template>
  </div>
</template>

<script setup>
// ModelPanel — 3D/CAD 工程只读视图（与 Agent 共用真相源：model.json + model.preview.html + model.verify.json）
// 设计取舍（与 ui-rig / ui-design / ui-art / ui-music 一致）：面板不写工作区 —— 几何/参数/布尔一律经 Agent 的
// model_add / model_edit（命令式 op），避免"工具写工程 / 面板写工程"两条写路径导致真相源分叉。
// 预览 iframe 用 sandbox="allow-scripts"（无 allow-same-origin）：预览需要脚本才能跑 WebGL 与交互，
// 但它拿不到宿主 DOM，且产物本身自包含（无外链）。
import { ref, computed, onMounted } from 'vue'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'
import { state } from '../ui-state.js'

const projectPath = ref('model.json')
const verifyPath = ref('model.verify.json')
const previewPath = ref('model.preview.html')
const loading = ref(false)
const error = ref('')
const model = ref(null)
const verify = ref(null)
const previewHtml = ref('')
const copied = ref(false)

const KIND = { '.gltf': 'glTF 2.0（Khronos）', '.glb': 'GLB（单文件二进制）', '.stl': 'binary STL（3D 打印）', '.ascii.stl': 'ASCII STL', '.obj': 'Wavefront OBJ', '.preview.html': '自包含 WebGL 预览' }

const summary = computed(() => (verify.value && verify.value.summary) || null)
const checks = computed(() => (verify.value && verify.value.checks) || [])
const passedCount = computed(() => checks.value.filter((c) => c.ok).length)
const products = computed(() => (verify.value && verify.value.products) || [])
const previewReady = computed(() => previewHtml.value.length > 0)
const problemItems = computed(() => {
  const out = []
  for (const c of checks.value) if (!c.ok) for (const it of (c.items || []).slice(0, 4)) out.push(c.id + ' ' + it)
  return out.slice(0, 8)
})
const triTotal = computed(() => {
  const s = summary.value
  if (s && s.triangles) return Number(s.triangles).toLocaleString()
  return partRows.value.reduce((a, p) => a + (p.triangles || 0), 0).toLocaleString()
})
const bboxText = computed(() => {
  const s = summary.value
  if (!s || !s.size) return '—'
  return s.size.map((v) => v).join(' × ') + ' mm'
})

// 部件表：工程侧（类型/运算/阵列/可见性）+ 报告侧（面数/体积/尺寸）
const partRows = computed(() => {
  const rep = {}
  for (const p of (verify.value && verify.value.parts) || []) rep[p.id] = p
  return ((model.value && model.value.parts) || []).map((p, i) => {
    const r = rep[p.id] || {}
    return {
      idx: i + 1,
      id: p.id,
      type: (p.shape && p.shape.type) || (r.type || 'cuboid'),
      ops: (p.ops || []).length || r.ops || 0,
      repeat: p.repeat ? (p.repeat.mode || 'linear') : (r.repeat || ''),
      visible: p.visible !== false,
      triangles: r.triangles || 0,
      size: r.size ? r.size.join('×') : '',
    }
  })
})

// 参数表：值 + 范围 + 在工程里被几何引用的次数（面板侧统计，不改工作区）
const paramRows = computed(() => {
  const text = JSON.stringify(model.value || {})
  return ((model.value && model.value.params) || []).map((p) => {
    let refs = 0
    try {
      const re = new RegExp('"' + p.id + '"', 'g')
      refs = (text.match(re) || []).length
    } catch (_) { refs = 0 }
    return {
      id: p.id,
      name: p.name,
      value: p.value,
      range: (p.min === undefined && p.max === undefined) ? '' : (p.min + '..' + p.max),
      refs: Math.max(0, refs - 1), // 减去定义处自身
    }
  })
})

const sampleCmd = computed(() => {
  const first = partRows.value[0] || {}
  return [
    '# 改参数 → 全模型重算 → 复验 → 出产物',
    JSON.stringify({ op: 'param.set', id: (paramRows.value[0] || {}).id || 'wall', value: 4 }),
    JSON.stringify({ op: 'part.transform', id: first.id || 'base', transform: { translate: [0, 0, 5] } }),
    'model_verify',
    'model_export format=all',
  ].join('\n')
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
    previewHtml.value = ''
    const vTxt = await readIfExists(verifyPath.value)
    if (vTxt) { try { verify.value = JSON.parse(vTxt) } catch (_) { /* 报告可选 */ } }
    previewHtml.value = await readIfExists(previewPath.value)
  } catch (e) {
    model.value = null
    verify.value = null
    previewHtml.value = ''
    error.value = '读取 CAD 工程失败：' + ((e && e.message) || e) + '（路径 ' + projectPath.value + '）'
  } finally {
    loading.value = false
  }
}
function shortPath(p) {
  return String(p).split(/[\\/]/).slice(-1)[0]
}
function kindOf(p) {
  const f = shortPath(p)
  return KIND['.' + f.split('.').slice(1).join('.')] || ''
}
function copyCmd() {
  const text = sampleCmd.value
  try {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(() => { copied.value = true; setTimeout(() => { copied.value = false }, 1500) })
      return
    }
  } catch (_) { /* 回退 */ }
  copied.value = false
}
onMounted(load)
</script>

<style scoped>
.mp-panel { padding: 8px 10px; font-size: 12px; color: var(--text-primary); overflow-y: auto; height: 100%; }
.mp-bar { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
.mp-title { font-size: 12px; font-weight: 500; color: var(--text-secondary); flex: none; }
.mp-input {
  flex: 1; min-width: 0; padding: 2px 6px; font-size: 11px; color: var(--text-primary);
  background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 4px;
  font-family: ui-monospace, Consolas, monospace;
}
.mp-icon-btn {
  flex: none; display: inline-flex; align-items: center; justify-content: center; width: 22px; height: 22px;
  color: var(--text-muted); background: transparent; border: 1px solid var(--border-color);
  border-radius: 4px; cursor: pointer;
}
.mp-icon-btn:hover:not(:disabled) { background: var(--bg-hover); }
.mp-icon-btn:disabled { cursor: default; opacity: 0.6; }
.mp-msg { color: var(--text-muted); padding: 6px 2px; line-height: 1.6; }
.mp-err { color: var(--text-primary); }
.mp-sec { margin-bottom: 12px; padding-bottom: 10px; border-bottom: 1px solid var(--border-color); }
.mp-sec:last-child { border-bottom: none; }
.mp-sec-title { font-size: 12px; font-weight: 500; color: var(--text-secondary); margin-bottom: 6px; }
.mp-kv { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; padding: 2px 0; }
.mp-kv > span { color: var(--text-muted); flex: none; }
.mp-kv > b { color: var(--text-primary); font-weight: 500; text-align: right; overflow: hidden; text-overflow: ellipsis; }
.mp-mono { font-family: ui-monospace, Consolas, monospace; }
.mp-dim { color: var(--text-muted); font-weight: 400; font-size: 11px; line-height: 1.6; }
.mp-row { display: flex; align-items: center; gap: 6px; }
.mp-spacer { flex: 1; }
.mp-btn {
  padding: 3px 10px; font-size: 11px; color: var(--text-secondary);
  background: transparent; border: 1px solid var(--border-color); border-radius: 4px; cursor: pointer;
}
.mp-btn:hover { background: var(--bg-hover); }
.mp-frame {
  display: block; width: 100%; height: 460px; background: #14161a;
  border: 1px solid var(--border-color); border-radius: 4px;
}
.mp-idx {
  display: inline-block; min-width: 16px; margin-right: 4px; text-align: right;
  color: var(--text-muted); font-variant-numeric: tabular-nums;
}
.mp-tag {
  margin-left: 4px; padding: 0 4px; font-size: 10px; color: var(--text-muted);
  border: 1px solid var(--border-color); border-radius: 3px;
}
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
.mp-items { margin-top: 6px; }
</style>
