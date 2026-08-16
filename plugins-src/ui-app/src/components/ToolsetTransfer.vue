<template>
  <!-- 工作区工具集（builtin）穿梭框：左=未加入，右=已加入，勾选批量加入/移出 -->
  <div class="dialog-overlay" @click.self="close">
    <div class="dialog-box ts-transfer-box">
      <div class="dialog-title">
        <span class="dialog-title-main"><SvgIcon name="package" :size="14" /> 管理工作区工具集</span>
        <span class="dialog-title-sub">插件工具 · 勾选后加入 / 移出工作区工具集</span>
      </div>
      <div class="ts-transfer-body">
        <!-- 左：未加入 -->
        <div class="ts-transfer-col">
          <div class="ts-transfer-col-head">
            <span class="ts-col-title">未加入</span>
            <span class="ts-col-hint">勾选后 → 加入</span>
            <span class="ts-col-head-actions">
              <button class="ts-btn mini" @click="selectAllLeft">全选</button>
            </span>
          </div>
          <div class="ts-transfer-list">
            <div v-for="g in leftGroups" :key="g.name" class="ts-transfer-group">
              <div class="ts-transfer-group-head">
                <label class="ts-transfer-check">
                  <input type="checkbox" :checked="groupAllChecked(g, true)" @change="toggleGroup(g, true)" />
                  <span class="ts-transfer-group-name">{{ g.name }}</span>
                  <span class="ts-tool-desc">{{ g.tools.length }} 工具</span>
                </label>
                <button class="ts-btn mini" @click="addGroup(g)">整组加入</button>
              </div>
              <label v-for="t in g.tools" :key="t.name" class="ts-transfer-check ts-transfer-tool" :title="t.desc">
                <input type="checkbox" :checked="leftSelected[t.name]" @change="toggleSelect(t.name, true)" />
                <span>{{ t.name }}</span>
              </label>
            </div>
            <div v-if="!leftGroups.length" class="ts-empty">全部已加入</div>
          </div>
        </div>
        <!-- 中间操作列 -->
        <div class="ts-transfer-ops">
          <button class="ts-btn primary" @click="addSelected" :disabled="!anyLeftSelected" title="把选中的工具加入工作区工具集">加入 →</button>
          <button class="ts-btn danger" @click="removeSelected" :disabled="!anyRightSelected" title="把选中的工具移出工作区工具集">← 移出</button>
        </div>
        <!-- 右：已加入 -->
        <div class="ts-transfer-col">
          <div class="ts-transfer-col-head">
            <span class="ts-col-title">已加入</span>
            <span class="ts-col-hint">勾选后 ← 移出</span>
            <span class="ts-col-head-actions">
              <button class="ts-btn mini" @click="selectAllRight">全选</button>
            </span>
          </div>
          <div class="ts-transfer-list">
            <div v-for="g in joinedGroups" :key="g.name" class="ts-transfer-group">
              <div class="ts-transfer-group-head">
                <label class="ts-transfer-check">
                  <input type="checkbox" :checked="groupAllChecked(g, false)" @change="toggleGroup(g, false)" />
                  <span class="ts-transfer-group-name">{{ g.name }}</span>
                  <span class="ts-tool-desc">{{ g.tools.length }} 工具</span>
                </label>
                <button class="ts-btn mini danger" @click="removeGroup(g)">整组移出</button>
              </div>
              <label v-for="t in g.tools" :key="t.name" class="ts-transfer-check ts-transfer-tool" :title="t.desc">
                <input type="checkbox" :checked="rightSelected[t.name]" @change="toggleSelect(t.name, false)" />
                <span>{{ t.name }}</span>
              </label>
            </div>
            <div v-if="manualTools.length" class="ts-transfer-group">
              <div class="ts-transfer-group-head">
                <label class="ts-transfer-check">
                  <input type="checkbox" :checked="groupAllChecked({ name: '_manual', tools: manualTools }, false)" @change="toggleGroup({ name: '_manual', tools: manualTools }, false)" />
                  <span class="ts-transfer-group-name">_manual（手动）</span>
                  <span class="ts-tool-desc">{{ manualTools.length }} 工具</span>
                </label>
                <button class="ts-btn mini danger" @click="removeManualTools">整组移出</button>
              </div>
              <label v-for="t in manualTools" :key="t" class="ts-transfer-check ts-transfer-tool">
                <input type="checkbox" :checked="rightSelected[t]" @change="toggleSelect(t, false)" />
                <span>{{ t }}</span>
              </label>
            </div>
            <div v-if="!joinedGroups.length && !manualTools.length" class="ts-empty">暂无已加入工具</div>
          </div>
        </div>
      </div>
      <div v-if="busy" class="ts-msg">操作中…</div>
      <div v-if="msg" class="ts-msg" :class="{ err: msgErr }">{{ msg }}</div>
      <div class="dialog-footer">
        <button class="ts-btn ghost" @click="close">关闭</button>
      </div>
    </div>
  </div>
</template>

<script setup>
// ToolsetTransfer — 工作区工具集穿梭框：未加入 ↔ 已加入 批量管理。
// ★ 2026-08-17：数据源从「内置分组（builtin groups）」改为「插件面板中存在工具的
//   插件分组（props.groups = /api/plugins/builtin 的 plugins 字段，source=plugin）」——
//   每个有工具的插件一组，组内是其注册的工具（含 enabled=agent 可见性）。
//   左=插件中未加入（enabled=false）的工具；右=已加入（enabled=true）的工具。
//   加入：插件未加入工具集 → toolsetEdit add_plugin（tools 白名单=勾选）；
//        插件已加入 → enable_tool 逐个恢复被摘除工具。
//   移出：rm_tool（摘除单工具）/ rm_plugin（整插件移出）。
//   手动工具（_manual）仍走 POST /api/plugins/builtin 工具级开关。
import { ref, computed, reactive } from 'vue'
import api from '../api.js'

const props = defineProps({
  groups: { type: Array, default: () => [] },   // 插件分组（source=plugin，含 tools[].enabled）
  joined: { type: Array, default: () => [] },   // 兼容保留（内置组名，不再用于插件分组判定）
  manualTools: { type: Array, default: () => [] },
})
const emit = defineEmits(['close', 'changed'])

// 左（未加入）：有未启用工具（enabled=false）的插件分组 → 只保留未启用工具
const leftGroups = computed(() => {
  return props.groups
    .filter(g => (g.tools || []).some(t => !t.enabled))
    .map(g => ({ ...g, tools: (g.tools || []).filter(t => !t.enabled) }))
})
// 右（已加入）：有已启用工具（enabled=true）的插件分组 → 只保留已启用工具
const joinedGroups = computed(() => {
  return props.groups
    .filter(g => (g.tools || []).some(t => t.enabled))
    .map(g => ({ ...g, tools: (g.tools || []).filter(t => t.enabled) }))
})
const manualTools = computed(() => props.manualTools)

const leftSelected = reactive({})
const rightSelected = reactive({})
const busy = ref(false)
const msg = ref('')
const msgErr = ref(false)

const anyLeftSelected = computed(() => Object.values(leftSelected).some(Boolean))
const anyRightSelected = computed(() => Object.values(rightSelected).some(Boolean))

function groupAllChecked(g, left) {
  const sel = left ? leftSelected : rightSelected
  return g.tools.length > 0 && g.tools.every(t => sel[t.name || t])
}
function toggleGroup(g, left) {
  const sel = left ? leftSelected : rightSelected
  const target = !groupAllChecked(g, left)
  for (const t of g.tools) sel[t.name || t] = target
}
function toggleSelect(name, left) {
  const sel = left ? leftSelected : rightSelected
  sel[name] = !sel[name]
}
function selectAllLeft() {
  const target = !leftGroups.value.every(g => groupAllChecked(g, true))
  for (const g of leftGroups.value) {
    for (const t of g.tools) leftSelected[t.name] = target
  }
}
function selectAllRight() {
  const groups = [...joinedGroups.value, ...(manualTools.value.length ? [{ tools: manualTools.value }] : [])]
  const target = !groups.every(g => groupAllChecked(g, false))
  for (const g of groups) {
    for (const t of g.tools) rightSelected[t.name || t] = target
  }
}

let once = null
async function callOnce(fn) {
  if (once) return
  once = true
  busy.value = true
  try {
    await fn()
    msg.value = ''; msgErr.value = false
  } catch (e) {
    msg.value = String(e && e.message || e); msgErr.value = true
  } finally {
    busy.value = false; once = null
  }
}

// 勾选工具加入工作区工具集（按插件分组处理）：
//   - 插件未加入工具集（g.joined=false）→ add_plugin（tools 白名单=勾选集合）——
//     插件整体装载，白名单外工具自动摘除（DisabledTools），只加入勾选工具；
//   - 插件已加入（g.joined=true，工具被摘除）→ enable_tool 逐个恢复。
async function addSelected() {
  const byPlugin = {} // 插件名 → 勾选工具名数组
  for (const g of leftGroups.value) {
    const names = (g.tools || []).map(t => t.name).filter(n => leftSelected[n])
    if (names.length) byPlugin[g.name] = { joined: !!g.joined, names }
  }
  try {
    for (const [pn, info] of Object.entries(byPlugin)) {
      if (info.joined) {
        for (const tn of info.names) {
          await callOnce(() => api.toolsetEdit({ name: 'default', action: 'enable_tool', plugin_name: pn, tool: tn }))
        }
      } else {
        await callOnce(() => api.toolsetEdit({ name: 'default', action: 'add_plugin', plugin_name: pn, tools: info.names.join(',') }))
      }
    }
    emit('changed')
  } catch (e) { /* callOnce 已上报 */ }
}
// 勾选工具移出工作区工具集：插件工具 → rm_tool（摘除单工具）；
// 手动工具（_manual）→ POST /api/plugins/builtin 工具级开关。
async function removeSelected() {
  const byPlugin = {}
  for (const g of joinedGroups.value) {
    const names = (g.tools || []).map(t => t.name).filter(n => rightSelected[n])
    if (names.length) byPlugin[g.name] = names
  }
  const manualNames = manualTools.value.filter(n => rightSelected[n])
  try {
    for (const [pn, names] of Object.entries(byPlugin)) {
      for (const tn of names) {
        await callOnce(() => api.toolsetEdit({ name: 'default', action: 'rm_tool', plugin_name: pn, tool: tn }))
      }
    }
    for (const n of manualNames) {
      await callOnce(() => api.builtinPlugins({ tool: n, enabled: false }))
    }
    emit('changed')
  } catch (e) { /* callOnce 已上报 */ }
}
// 整组加入：插件未加入工具集 → add_plugin（不带 tools = 整插件全部工具加入）；
// 插件已加入 → enable_tool 逐个恢复。
async function addGroup(g) {
  try {
    if (g.joined) {
      for (const t of g.tools) {
        await callOnce(() => api.toolsetEdit({ name: 'default', action: 'enable_tool', plugin_name: g.name, tool: t.name }))
      }
    } else {
      await callOnce(() => api.toolsetEdit({ name: 'default', action: 'add_plugin', plugin_name: g.name }))
    }
    emit('changed')
  } catch (e) { /* callOnce 已上报 */ }
}
// 整组移出：rm_plugin（插件移出工具集，其工具恢复默认过滤）
async function removeGroup(g) {
  try {
    await callOnce(() => api.toolsetEdit({ name: 'default', action: 'rm_plugin', plugin_name: g.name }))
    emit('changed')
  } catch (e) { /* callOnce 已上报 */ }
}
async function removeManualTools() {
  try {
    for (const n of props.manualTools) {
      await callOnce(() => api.builtinPlugins({ tool: n, enabled: false }))
    }
    emit('changed')
  } catch (e) { /* callOnce 已上报 */ }
}

function close() { emit('close') }
</script>

<style scoped>
/* ── 弹窗遮罩/盒子（独立组件自带，不能依赖父组件 scoped 样式） ── */
.dialog-overlay {
  position: fixed; inset: 0; z-index: 1000;
  background: rgba(1, 4, 9, 0.62);
  backdrop-filter: blur(3px);
  display: flex; align-items: center; justify-content: center;
}
.dialog-box {
  position: relative;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 18px 20px 14px;
  min-width: 320px; max-width: 600px; width: 90%;
  box-shadow: 0 16px 48px rgba(0,0,0,0.5), 0 2px 8px rgba(0,0,0,0.3);
  overflow: hidden;
}
/* 顶部 accent 高亮渐变条 */
.dialog-box::before {
  content: ''; position: absolute; top: 0; left: 0; right: 0; height: 2px;
  background: linear-gradient(90deg, var(--accent), var(--accent-light) 55%, transparent);
}
.dialog-title {
  display: flex; align-items: center; gap: 10px;
  font-size: 14px; font-weight: 600; color: var(--text-primary);
  margin-bottom: 12px; padding-top: 2px;
}
.dialog-title-main { display: flex; align-items: center; gap: 6px; }
.dialog-title-main svg { color: var(--accent); }
.dialog-title-sub { font-size: 10px; font-weight: 400; color: var(--text-muted); margin-left: auto; }
.dialog-footer { display: flex; justify-content: flex-end; gap: 8px; margin-top: 12px; }

.ts-transfer-box { width: 780px; max-width: 92vw; }
/* ★ 滚动约束：body 固定高度区（min→max 取实际），col 必须允许收缩（min-height: 0），
   list 内部滚动——否则 flex 子项默认 min-height:auto 被内容撑破，列表不滚动而是挤压 */
.ts-transfer-body {
  display: flex; gap: 10px;
  height: min(60vh, 560px);
  min-height: 320px;
}
/* 左右列 = 独立面板卡片 */
.ts-transfer-col {
  flex: 1; display: flex; flex-direction: column; min-width: 0;
  min-height: 0; /* ★ 允许 flex 收缩，配合 list 滚动 */
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
}
.ts-transfer-col-head {
  display: flex; align-items: center; gap: 8px;
  padding: 6px 10px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}
.ts-col-title {
  font-size: 11px; font-weight: 700; color: var(--text-primary);
  letter-spacing: 0.5px;
  display: flex; align-items: center; gap: 6px;
}
.ts-col-title::before {
  content: ''; width: 3px; height: 11px; border-radius: 2px;
  background: var(--accent);
}
.ts-col-hint { font-size: 9px; color: var(--text-muted); }
.ts-col-head-actions { margin-left: auto; display: flex; align-items: center; }
.ts-transfer-list {
  flex: 1; min-height: 0; /* ★ 允许收缩 → 溢出滚动 */
  overflow-y: auto; overflow-x: hidden;
  padding: 6px;
  display: flex; flex-direction: column; gap: 8px;
  scrollbar-gutter: stable;
}
.ts-transfer-list::-webkit-scrollbar { width: 8px; }
.ts-transfer-list::-webkit-scrollbar-thumb { background: var(--border-color); border-radius: 4px; }
.ts-transfer-group {
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-tertiary);
  overflow: hidden;
  flex-shrink: 0;
}
.ts-transfer-group-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 6px 8px 6px 10px; gap: 6px;
  background: color-mix(in srgb, var(--accent) 7%, var(--bg-secondary));
  border-left: 2px solid var(--accent);
}
.ts-transfer-group-name { font-size: 11px; font-weight: 600; color: var(--accent-light); }
.ts-tool-desc { font-size: 9px; color: var(--text-muted); font-weight: 400; margin-left: 2px; }
.ts-transfer-check {
  display: flex; align-items: center; gap: 5px;
  font-size: 11px; color: var(--text-primary); cursor: pointer;
}
.ts-transfer-check input[type='checkbox'] { accent-color: var(--accent); margin: 0; }
.ts-transfer-tool { padding: 4px 10px 4px 18px; font-family: var(--font-code); transition: background .1s; }
.ts-transfer-tool:hover { background: var(--bg-hover); }
.ts-transfer-tool:has(input:checked) { background: var(--accent-bg); }
.ts-transfer-ops { display: flex; flex-direction: column; justify-content: center; gap: 10px; flex-shrink: 0; }
.ts-msg { padding: 6px 8px; font-size: 11px; color: var(--text-secondary); }
.ts-msg.err { color: #f47067; }

/* 按钮体系：基础 / mini / primary（accent 实心）/ danger / ghost */
.ts-btn {
  background: var(--bg-primary); border: 1px solid var(--border-color); color: var(--text-secondary);
  border-radius: 5px; padding: 4px 12px; font-size: 11px; cursor: pointer;
  transition: all .12s;
}
.ts-btn:hover { background: var(--bg-hover); color: var(--text-primary); border-color: var(--text-muted); }
.ts-btn:disabled { opacity: .45; cursor: not-allowed; }
.ts-btn.mini { font-size: 10px; padding: 1px 8px; border-radius: 10px; }
.ts-btn.mini:hover { color: var(--accent-light); border-color: var(--accent); background: var(--accent-bg); }
.ts-btn.mini.danger { color: #f47067; border-color: rgba(244,112,103,.4); }
.ts-btn.mini.danger:hover { background: rgba(244,112,103,.12); border-color: #f47067; color: #f47067; }
.ts-btn.primary {
  background: var(--accent); border-color: var(--accent);
  color: #0d1117; font-weight: 600;
}
.ts-btn.primary:hover { background: var(--accent-light); border-color: var(--accent-light); color: #0d1117; }
.ts-btn.primary:disabled { background: color-mix(in srgb, var(--accent) 40%, transparent); border-color: transparent; color: rgba(13,17,23,.5); }
.ts-btn.danger {
  background: transparent; border-color: rgba(244,112,103,.5); color: #f47067;
}
.ts-btn.danger:hover { background: rgba(244,112,103,.12); border-color: #f47067; color: #f47067; }
.ts-btn.ghost { background: transparent; }
.ts-empty { padding: 14px 4px; text-align: center; color: var(--text-muted); font-size: 11px; }
</style>
