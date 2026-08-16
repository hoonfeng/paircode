<template>
  <!-- 工作区工具集（builtin）穿梭框：左=未加入，右=已加入，勾选批量加入/移出 -->
  <div class="dialog-overlay" @click.self="close">
    <div class="dialog-box ts-transfer-box">
      <div class="dialog-title">
        管理工作区工具集（builtin）
        <span class="ts-transfer-count">已加入 {{ joinedTotal }}/{{ toolTotal }} 工具</span>
      </div>
      <div class="ts-transfer-body">
        <!-- 左：未加入 -->
        <div class="ts-transfer-col">
          <div class="ts-transfer-col-head">
            <span>未加入（{{ leftGroups.length }} 组 / {{ leftCount }} 工具）</span>
            <button class="ts-btn mini" @click="selectAllLeft">全选</button>
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
          <button class="ts-btn" @click="addSelected" :disabled="!anyLeftSelected" title="把选中的工具加入工作区工具集">→ 加入</button>
          <button class="ts-btn" @click="removeSelected" :disabled="!anyRightSelected" title="把选中的工具移出工作区工具集">← 移出</button>
        </div>
        <!-- 右：已加入 -->
        <div class="ts-transfer-col">
          <div class="ts-transfer-col-head">
            <span>已加入（{{ joinedTotal }} 工具）</span>
            <button class="ts-btn mini" @click="selectAllRight">全选</button>
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
        <button class="ts-btn" @click="close">关闭</button>
      </div>
    </div>
  </div>
</template>

<script setup>
// ToolsetTransfer — 工作区工具集（builtin）穿梭框：未加入 ↔ 已加入 批量管理。
// props.groups：builtin 全量分组（含工具名/描述）；props.joined：已加入组名数组；
// props.manualTools：手动加入的工具名数组。
// 加入/移出走 POST /api/plugins/builtin（组级或工具级），完成后 emit('changed') 通知父级刷新。
import { ref, computed, reactive } from 'vue'
import api from '../api.js'

const props = defineProps({
  groups: { type: Array, default: () => [] },
  joined: { type: Array, default: () => [] },
  manualTools: { type: Array, default: () => [] },
})
const emit = defineEmits(['close', 'changed'])

const joinedSet = computed(() => new Set(props.joined))
// 左（未加入）：非 joined 组全量
const leftGroups = computed(() => {
  return props.groups
    .filter(g => !joinedSet.value.has(g.name))
    .map(g => ({ ...g, tools: g.tools }))
})
// 右（已加入）：joined 组的工具
const joinedGroups = computed(() => {
  return props.groups
    .filter(g => joinedSet.value.has(g.name))
    .map(g => ({ ...g, tools: g.tools }))
})
const leftCount = computed(() => leftGroups.value.reduce((n, g) => n + g.tools.length, 0))
const toolTotal = computed(() => props.groups.reduce((n, g) => n + g.tools.length, 0) + props.manualTools.length)
const joinedTotal = computed(() => joinedGroups.value.reduce((n, g) => n + g.tools.length, 0) + props.manualTools.length)

const leftSelected = reactive({})
const rightSelected = reactive({})
const anyLeftSelected = computed(() => Object.keys(leftSelected).some(k => leftSelected[k]))
const anyRightSelected = computed(() => Object.keys(rightSelected).some(k => rightSelected[k]))

const busy = ref(false)
const msg = ref('')
const msgErr = ref(false)

function selectAllLeft() {
  const all = leftGroups.value.every(g => g.tools.every(t => leftSelected[t.name]))
  for (const g of leftGroups.value) for (const t of g.tools) leftSelected[t.name] = !all
}
function selectAllRight() {
  const all = joinedGroups.value.every(g => g.tools.every(t => rightSelected[t.name]))
    && props.manualTools.every(t => rightSelected[t])
  for (const g of joinedGroups.value) for (const t of g.tools) rightSelected[t.name] = !all
  for (const t of props.manualTools) rightSelected[t] = !all
}
function toggleSelect(name, isLeft) {
  const sel = isLeft ? leftSelected : rightSelected
  sel[name] = !sel[name]
}
function groupAllChecked(g, isLeft) {
  return g.tools.length > 0 && g.tools.every(t => (isLeft ? leftSelected : rightSelected)[t.name])
}
function toggleGroup(g, isLeft) {
  const all = groupAllChecked(g, isLeft)
  const sel = isLeft ? leftSelected : rightSelected
  for (const t of g.tools) sel[t.name] = !all
}

function show(m, err) { msg.value = m; msgErr.value = !!err }

async function callOnce(fn) {
  busy.value = true
  try {
    const res = await fn()
    if (res && res.message) show(res.message)
  } catch (e) {
    show(e.message || String(e), true)
    busy.value = false
    throw e
  }
  busy.value = false
}

function addGroup(g) {
  callOnce(() => api.builtinPlugins({ group: g.name, enabled: true })).then(() => emit('changed')).catch(() => {})
}
function removeGroup(g) {
  callOnce(() => api.builtinPlugins({ group: g.name, enabled: false })).then(() => emit('changed')).catch(() => {})
}
async function addSelected() {
  const names = leftGroups.value.flatMap(g => g.tools).filter(t => leftSelected[t.name]).map(t => t.name)
  try {
    for (const n of names) await callOnce(() => api.builtinPlugins({ tool: n, enabled: true }))
    emit('changed')
  } catch (e) { /* callOnce 已上报 */ }
}
async function removeSelected() {
  const names = [...joinedGroups.value.flatMap(g => g.tools).map(t => t.name), ...props.manualTools]
    .filter(n => rightSelected[n])
  try {
    for (const n of names) await callOnce(() => api.builtinPlugins({ tool: n, enabled: false }))
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
.ts-transfer-box { width: 780px; max-width: 92vw; }
.ts-transfer-count { font-size: 10px; color: var(--text-muted); font-weight: normal; margin-left: 8px; }
.ts-transfer-body { display: flex; gap: 8px; min-height: 320px; max-height: 60vh; }
.ts-transfer-col { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.ts-transfer-col-head {
  display: flex; align-items: center; justify-content: space-between;
  font-size: 11px; font-weight: 600; color: var(--text-secondary);
  padding: 4px 6px; border-bottom: 1px solid var(--border-color);
}
.ts-transfer-list { flex: 1; overflow: auto; padding: 4px; display: flex; flex-direction: column; gap: 4px; border: 1px solid var(--border-color); border-radius: 4px; background: var(--bg-tertiary); }
.ts-transfer-group { border: 1px solid var(--border-color); border-radius: 4px; background: var(--bg-primary); overflow: hidden; }
.ts-transfer-group-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 3px 6px; background: var(--bg-hover); gap: 4px;
}
.ts-transfer-group-name { font-size: 11px; font-weight: 600; color: var(--accent); }
.ts-transfer-check { display: flex; align-items: center; gap: 5px; font-size: 11px; color: var(--text-primary); cursor: pointer; }
.ts-transfer-tool { padding: 2px 8px 2px 14px; font-family: var(--font-code); }
.ts-transfer-tool:hover { background: var(--bg-hover); }
.ts-transfer-ops { display: flex; flex-direction: column; justify-content: center; gap: 8px; flex-shrink: 0; }
.ts-msg { padding: 6px 8px; font-size: 11px; color: var(--text-secondary); }
.ts-msg.err { color: #e06c75; }
</style>
