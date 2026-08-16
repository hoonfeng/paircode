<template>
  <div class="plugin-panel">
    <!-- 头部 -->
    <div class="pp-header">
      <span class="pp-title"><SvgIcon name="puzzle" :size="14" /> 插件</span>
      <div class="pp-actions">
        <button class="pp-icon-btn" @click="refresh" title="刷新"><SvgIcon name="refresh" :size="13" :class="{ spinning: refreshing }" /></button>
        <button class="pp-icon-btn" :class="{ active: showToolset }" @click="showToolset = !showToolset" title="工具集管理（插件化：加插件/删插件/摘工具）"><SvgIcon name="layers" :size="13" /></button>
        <button class="pp-icon-btn" @click="showNew = !showNew" title="新建插件"><SvgIcon name="plus" :size="14" /></button>
      </div>
    </div>

    <!-- 工具集管理（插件化思路：add_plugin / rm_plugin / rm_tool / enable_tool） -->
    <div v-if="showToolset" class="pp-toolset">
      <div class="pp-ts-head">
        <select v-model="tsName" class="pp-input pp-lang" @change="loadToolsetDetail">
          <option value="">选择工具集…</option>
          <optgroup label="工作区工具集（本项目）">
            <option v-for="t in toolsetMetas.filter(x => x.scope !== 'builtin')" :key="t.name" :value="t.name">{{ t.name }}（{{ t.pluginCount }} 插件）</option>
            <option v-for="t in toolsetMetas.filter(x => x.scope === 'builtin')" :key="t.name" :value="t.name">{{ t.name }}（{{ t.pluginCount }} 插件·内置默认）</option>
          </optgroup>
        </select>
        <button class="pp-btn" @click="loadToolsets">刷新</button>
      </div>
      <div v-if="tsDetail" class="pp-ts-body">
        <div class="pp-ts-title">
          {{ tsDetail.name }}
          <span class="pp-ts-scope">{{ tsDetail.project ? tsDetail.project + '·' : '' }}{{ tsDetail.description || '工具集' }}</span>
        </div>
        <div v-for="pl in tsDetail.plugins" :key="pl.name" class="pp-ts-plugin">
          <div class="pp-ts-prow">
            <span class="pp-ts-pname">{{ pl.name }}</span>
            <button v-if="tsDetail.scope !== 'builtin'" class="pp-btn danger" @click="edit({ action: 'rm_plugin', plugin_name: pl.name })">移出工具集</button>
            <span v-else class="pp-ts-muted">内置</span>
          </div>
          <div v-if="pl.purpose" class="pp-ts-purpose">{{ pl.purpose }}</div>
          <div v-if="pluginToolsOf(pl.name).length" class="pp-ts-tools">
            <label v-for="t in pluginToolsOf(pl.name)" :key="t" class="pp-ts-tool" :title="isToolDisabled(pl, t) ? '已摘除（对 agent 不可见），点击恢复' : '点击摘除（插件保留、工具不可见）'">
              <input type="checkbox" :checked="!isToolDisabled(pl, t)" @change="toggleTool(pl, t)" />
              <span :class="{ off: isToolDisabled(pl, t) }">{{ t }}</span>
            </label>
          </div>
          <div v-else class="pp-ts-muted">（插件未运行或无工具）</div>
        </div>
        <div class="pp-ts-add">
          <select v-model="addPluginName" class="pp-input pp-lang">
            <option value="">把宿主插件加入工具集…</option>
            <option v-for="p in addablePlugins" :key="p.name" :value="p.name">{{ p.name }}<template v-if="p.tools && p.tools.length">（{{ p.tools.length }} 工具）</template></option>
          </select>
          <button class="pp-btn primary" :disabled="!addPluginName" @click="doAddPlugin">加入</button>
        </div>
      </div>
      <div v-else class="pp-ts-empty">选择上方工具集查看/编辑其插件与工具</div>
    </div>

    <!-- 新建插件表单 -->
    <div v-if="showNew" class="pp-new">
      <div class="pp-new-title">新建 JS 动态插件</div>
      <input v-model="newForm.purpose" placeholder="用途说明（必填）" class="pp-input" />
      <textarea v-model="newForm.code" placeholder="host 半代码（必填）：(async () => { return { name, apply(ctx, config) } })()" class="pp-textarea code" rows="6"></textarea>
      <textarea v-model="newForm.client" placeholder="client 半代码（可选）：(ui) => { ui.registerPanel({...}); ui.on('ui:xxx', fn) }" class="pp-textarea code" rows="4"></textarea>
      <div class="pp-new-foot">
        <select v-model="newForm.language" class="pp-input pp-lang">
          <option value="">语言(自动)</option>
          <option value="js">js</option>
          <option value="ts">ts</option>
        </select>
        <label class="pp-check"><input type="checkbox" v-model="newForm.run" /> 定义后立即装载</label>
        <button class="pp-btn primary" :disabled="defining || !newForm.purpose || !newForm.code" @click="doDefine">
          {{ defining ? '定义中…' : '定义' }}
        </button>
      </div>
      <div v-if="newMsg" class="pp-new-msg" :class="{ err: newMsgErr }">{{ newMsg }}</div>
    </div>

    <!-- 客户端面板区（client 半注册的自定义面板） -->
    <div v-if="clientPanels.length > 0" class="pp-client">
      <div class="pp-client-tabs">
        <div v-for="p in clientPanels" :key="p.id"
             :class="['pp-client-tab', { active: activePanelId === p.id }]"
             @click="selectPanel(p.id)">
          <SvgIcon :name="p.icon || 'sparkles'" :size="12" />
          <span class="pp-client-tab-title">{{ p.title }}</span>
        </div>
      </div>
      <div ref="clientPanelEl" class="pp-client-body"></div>
    </div>

    <!-- UI 槽位区（Slot 系统：client 半注册的可替换界面区域，如底部状态栏） -->
    <div v-if="slotGroups.length > 0" class="pp-slots">
      <div class="pp-slots-head" @click="slotsOpen = !slotsOpen" :title="slotsOpen ? '点击收起 UI 槽位列表' : '点击展开 UI 槽位列表'" style="cursor:pointer">
        <span class="pp-slots-title"><SvgIcon name="layers" :size="13" /> UI 槽位</span>
        <span class="pp-slots-sub">插件可替换的界面区域</span>
        <SvgIcon name="chevron-right" :size="11" class="pp-chevron" :class="{ open: slotsOpen }" />
      </div>
      <template v-if="slotsOpen">
      <div v-for="g in slotGroups" :key="g.slotId + '::' + g.kind" class="pp-slot-row">
        <div class="pp-slot-info">
          <div class="pp-slot-title-row">
            <span class="pp-slot-id">{{ g.slotId }}</span>
            <span class="pp-slot-kind" :class="g.kind === 'list' ? 'kind-list' : 'kind-single'">{{ g.kind === 'list' ? '叠加' : '替换' }}</span>
          </div>
          <span class="pp-slot-owner" :class="{ builtin: !g.owner && g.kind !== 'list' }">
            {{ g.kind === 'list'
              ? (g.candidates.length ? g.candidates.length + ' 个叠加条目' : '（无叠加条目）')
              : (g.owner ? g.owner : (g.builtin ? '内置组件' : '（无宿主）')) }}
          </span>
        </div>
        <!-- single 槽位：下拉切换占用者（内置默认 / 插件占用者） -->
        <select v-if="g.kind !== 'list'" class="pp-input pp-slot-select" :value="g.owner" @change="switchSlot(g.slotId, $event.target.value)"
                :title="'切换 ' + g.slotId + ' 区域的渲染者'">
          <option value="">{{ g.builtin ? '内置组件（默认）' : '（未占用）' }}</option>
          <option v-for="c in g.candidates" :key="c.pluginName" :value="c.pluginName">{{ c.pluginName }} · {{ c.title }}</option>
        </select>
        <div v-else class="pp-slot-list">
          <label v-for="c in g.candidates" :key="c.pluginName" class="pp-slot-list-item">
            <input type="checkbox" :checked="overlayActive(g.slotId, c.pluginName)" @change="toggleOverlay(g.slotId, c.pluginName, $event.target.checked)" />
            <span>{{ c.pluginName }} · {{ c.title }}</span>
          </label>
          <span v-if="!g.candidates.length" class="pp-slot-empty">（无叠加条目）</span>
        </div>
      </div>
      </template>
    </div>
    <div class="pp-list">
      <div v-if="loading && plugins.length === 0" class="pp-loading">
        <SvgIcon name="refresh" :size="16" class="spinner" /><span>加载插件…</span>
      </div>
      <div v-else-if="loadError" class="pp-empty">
        <SvgIcon name="puzzle" :size="22" color="var(--text-muted)" />
        <span>插件列表加载失败</span>
        <button class="pp-btn primary" @click="refresh">重试</button>
      </div>
      <div v-else-if="plugins.length === 0 && !loading" class="pp-empty">
        <SvgIcon name="puzzle" :size="22" color="var(--text-muted)" />
        <span>暂无插件</span>
        <span class="pp-empty-sub">点击上方 + 新建 JS 动态插件，或用对话 cordis_define 定义</span>
      </div>
      <div v-for="p in plugins" :key="p.name" class="pp-item">
        <div class="pp-item-row" @click="toggleDetail(p)">
          <span class="pp-state" :class="p.state === 'running' ? 'on' : 'off'"></span>
          <span class="pp-name" :title="p.purpose || p.name">{{ p.name }}</span>
          <span class="pp-src" :class="p.source">{{ p.source }}</span>
          <span v-if="p.scope === 'global'" class="pp-badge" title="全局插件：跨工作区生效（UI 类），不属于任何工具集">全局</span>
          <span v-if="p.hasClient && p.clientApproved" class="pp-badge" title="含 client 半（浏览器 UI，已批准装载）">UI</span>
          <span v-else-if="p.hasClient && p.state === 'running'" class="pp-badge pp-badge-warn" title="client 半待激活批准：在对话中用 cordis_run 装载该插件触发审批">UI 待批准</span>
          <span v-else-if="p.hasClient" class="pp-badge" title="含 client 半（浏览器 UI；装载后需批准）">UI</span>
          <span v-if="p.tools && p.tools.length" class="pp-count" :title="p.tools.join(', ')">{{ p.tools.length }} 工具</span>
          <template v-if="p.hasClient && p.clientApproved && uiSlotsOf(p.name).length">
            <span class="pp-ui-label" :class="{ on: uiPluginActive(p.name) }">{{ uiPluginActive(p.name) ? 'UI 已启用' : 'UI 未启用' }}</span>
            <label class="pp-switch" :title="uiPluginActive(p.name) ? '停用该插件的 UI（恢复内置界面）' : '启用该插件的 UI（替换对应界面区域）'">
              <input type="checkbox" :checked="uiPluginActive(p.name)" @change="toggleUiPlugin(p, $event.target.checked)" @click.stop />
              <span class="pp-switch-track"></span>
            </label>
          </template>
          <SvgIcon name="chevron-right" :size="12" class="pp-chevron" :class="{ open: expanded[p.name] }" />
          </div>
          <div v-if="expanded[p.name]" class="pp-detail">
            <div v-if="p.purpose" class="pp-d-purpose">{{ p.purpose }}</div>
            <div v-if="p.defId" class="pp-d-line">定义: {{ p.defId }}<span v-if="p.version"> · {{ p.version }}</span></div>
            <div v-if="p.provides && p.provides.length" class="pp-d-line">服务: {{ p.provides.join(', ') }}</div>
            <div v-if="p.sections && p.sections.length" class="pp-d-line">提示片段: {{ p.sections.join(', ') }}</div>
            <div v-if="p.tools && p.tools.length" class="pp-d-tools">
              <div class="pp-d-tools-title">工具（{{ p.tools.length }}）· 开关控制 agent 可见性</div>
              <div v-for="t in p.tools" :key="t" class="pp-d-tool">
                <span class="pp-d-tname" :title="t">{{ t }}</span>
                <label class="pp-switch" :title="pluginToolOn(p, t) ? '对 agent 可见；点击禁用（不影响插件运行）' : '对 agent 不可见；点击启用'">
                  <input type="checkbox" :checked="pluginToolOn(p, t)" @change="togglePluginTool(p, t)" />
                  <span class="pp-switch-track"></span>
                </label>
              </div>
            </div>
            <div v-if="p.clientCode" class="pp-d-code">
              <div class="pp-d-code-head">
              <span>client 半源码</span>
              <button class="pp-icon-btn" @click="copyText(p.clientCode)" title="复制"><SvgIcon name="copy" :size="11" /></button>
            </div>
            <pre>{{ p.clientCode }}</pre>
          </div>
          <div class="pp-d-actions">
            <!-- UI 类插件（client 半已装载且有槽位）不再显示「停止插件」：
                 UI 可见性已由勾选/UI 开关控制（取消勾选=隐藏，勾选=恢复），
                 stop 会卸载 client 半并清空槽位条目 → 勾选框消失无法再启用。
                 stopped 状态仍保留「启动插件」按钮作为恢复路径。 -->
            <template v-if="p.state === 'running'">
              <button v-if="!(p.hasClient && p.clientApproved && uiSlotsOf(p.name).length)" class="pp-btn" title="停止整个插件（其全部工具对 agent 不可见）；单工具请用上方工具开关" @click="doAction(p, 'stop')">停止插件</button>
            </template>
            <button v-else class="pp-btn primary" @click="doAction(p, 'start')">启动插件</button>
            <button v-if="p.source === 'js'" class="pp-btn danger" @click="doAction(p, 'undefine')">删除定义</button>
          </div>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'
import { clientPanels, clientSlots, syncClientHalves, unloadClientHalf, startPolling, stopPolling, setPanelMount, setSlotMount, getUIFor, getSlotCandidates, getSlotOwner, setSlotOwner, emitSlotChanged, isOverlayActive, setOverlayActive, isPluginUIEnabled, setPluginUIEnabled } from '../plugin-runtime.js'

const plugins = ref([])
const loading = ref(false)
const refreshing = ref(false)
const loadError = ref(false)
const expanded = reactive({})
const showNew = ref(false)
const defining = ref(false)
const slotsOpen = ref(false) // UI 槽位区默认折叠：打开面板直接看到插件列表
const newMsg = ref('')
const newMsgErr = ref(false)
const activePanelId = ref('')
const clientPanelEl = ref(null)
// 工具集管理状态
const showToolset = ref(false)
const toolsetMetas = ref([])
const tsName = ref('')
const tsDetail = ref(null)
const addPluginName = ref('')

const newForm = reactive({ purpose: '', code: '', client: '', language: '', run: true })


// ─── 工具集管理（插件化：add_plugin / rm_plugin / rm_tool / enable_tool）──
async function loadToolsets() {
  try {
    const list = (await api.getToolsets()) || []
    // ★ 工具集管理只显示工作区（project）工具集 + 内置默认工具包（builtin）；
    //    全局工具集（global，UI 插件等跨项目插件）不在此展示——自动装载、无需管理。
    toolsetMetas.value = list.filter(t => t.scope !== 'global')
  } catch (e) {
    toolsetMetas.value = []
  }
  if (tsName.value && !toolsetMetas.value.some(t => t.name === tsName.value)) tsName.value = ''
  if (tsName.value) await loadToolsetDetail()
  else tsDetail.value = null
}

async function loadToolsetDetail() {
  if (!tsName.value) { tsDetail.value = null; return }
  try {
    tsDetail.value = await api.getToolsets(tsName.value)
  } catch (e) {
    window.$toast && window.$toast('加载工具集失败: ' + (e.message || e), 'error')
  }
}

// 插件工具清单（从宿主插件列表取——工具是插件运行时注册的）
function pluginToolsOf(pname) {
  const p = plugins.value.find(x => x.name === pname)
  return (p && p.tools) || []
}

function isToolDisabled(pl, t) {
  return (pl.disabledTools || []).includes(t)
}

// 工具集编辑（即时热装载 + 回写固化 JSON）
async function edit(data) {
  try {
    const res = await api.toolsetEdit({ name: tsName.value, ...data })
    window.$toast && window.$toast((res && res.message) || '操作成功', 'info')
    await loadToolsetDetail()
    await refresh()
  } catch (e) {
    window.$toast && window.$toast(e.message || '操作失败', 'error')
  }
}

function toggleTool(pl, t) {
  edit({ action: isToolDisabled(pl, t) ? 'enable_tool' : 'rm_tool', plugin_name: pl.name, tool: t })
}

async function doAddPlugin() {
  if (!addPluginName.value) return
  await edit({ action: 'add_plugin', plugin_name: addPluginName.value })
  addPluginName.value = ''
}

// 可加入的宿主插件（排除已在工具集的）
const addablePlugins = computed(() => {
  const inTs = new Set((tsDetail.value && tsDetail.value.plugins || []).map(p => p.name))
  return plugins.value.filter(p => !inTs.has(p.name))
})

watch(showToolset, v => { if (v) loadToolsets() })

// ─── 列表加载 ────────────────────────────────────────────────
// 用 XHR 而非 fetch：兼容性最好（旧 Edge/无头环境都可靠），自带 timeout
function fetchPluginsJSON() {
  return new Promise((resolve, reject) => {
    const x = new XMLHttpRequest()
    x.open('GET', '/api/plugins', true)
    x.timeout = 8000
    x.onload = () => {
      if (x.status >= 200 && x.status < 300) {
        try { resolve(JSON.parse(x.responseText)) } catch (e) { reject(e) }
      } else reject(new Error('HTTP ' + x.status))
    }
    x.onerror = () => reject(new Error('network error'))
    x.ontimeout = () => reject(new Error('timeout'))
    x.send()
  })
}
async function refresh() {
  refreshing.value = true
  loading.value = true
  loadError.value = false
  try {
    // 失败自动重试 1 次；仍失败显示重试按钮（不静默吞掉）
    let list = []
    for (let attempt = 0; attempt < 2; attempt++) {
      try {
        const data = await fetchPluginsJSON()
        if (Array.isArray(data)) { list = data; break }
      } catch (e) {
        list = []
      }
    }
    plugins.value = list
    if (!list.length) loadError.value = true
    // 列表接口省略 clientCode：详情**并行**补取（失败跳过，不阻塞列表渲染）
    const detailTargets = plugins.value.filter(p => p.hasClient && !p.clientCode)
    await Promise.allSettled(detailTargets.map(async (p) => {
      try {
        const d = await api.getPluginDetail(p.name)
        if (d && d.clientCode) p.clientCode = d.clientCode
      } catch (e) { /* 详情失败跳过 */ }
    }))
    await syncClientHalves(plugins.value)
  } catch (e) {
    console.warn('[plugin] 加载失败', e)
    loadError.value = true
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

// ─── 详情折叠 ────────────────────────────────────────────────
async function toggleDetail(p) {
  expanded[p.name] = !expanded[p.name]
  // 展开且缺 clientCode 时补取
  if (expanded[p.name] && p.hasClient && !p.clientCode) {
    try {
      const d = await api.getPluginDetail(p.name)
      if (d) Object.assign(p, d)
    } catch (e) {}
  }
}

// ─── 插件详情：单个工具开关（agent 可见性；不影响插件运行）──────────
function pluginToolOn(p, t) {
  return !(p.toolStates && p.toolStates[t] === false)
}

async function togglePluginTool(p, t) {
  const target = !pluginToolOn(p, t)
  try {
    const res = await api.pluginToolToggle(t, target)
    window.$toast && window.$toast((res && res.message) || (target ? '已启用' : '已禁用') + ' ' + t, 'info')
    if (!p.toolStates) p.toolStates = {}
    p.toolStates[t] = target
  } catch (e) {
    window.$toast && window.$toast(e.message || '操作失败', 'error')
  }
}

// ─── UI 插件启用开关（Slot 系统）：勾选=激活该插件注册的全部槽位 ──
//  single 槽位（titlebar/activitybar/sidebar/editor/right-panel/statusbar/modals）：
//    激活=插件级 UI 启用（渲染时检查 isPluginUIEnabled）；停用=禁用标记 → 区域空态。
//    ★ 不再用 setSlotOwner('')「恢复内置」：壳是纯骨架无内置可恢复，空 owner 会
//      自动回退重新激活唯一候选 → 永远停不掉（历史 bug：单槽插件开关无效）。
//  list 槽位（overlay）：激活=勾选叠加显示（槽位级 overlay），停用=隐藏。
function uiSlotsOf(pname) {
  return clientSlots.filter(s => s.pluginName === pname)
}
function uiPluginActive(pname) {
  const slots = uiSlotsOf(pname)
  if (!slots.length) return false
  return slots.some(s => {
    if (s.kind === 'list') return isOverlayActive(s.slotId, s.pluginName)
    return isPluginUIEnabled(s.pluginName)
  })
}
function toggleUiPlugin(p, on) {
  const slots = uiSlotsOf(p.name)
  for (const s of slots) {
    if (s.kind === 'list') {
      setOverlayActive(s.slotId, s.pluginName, on)
    } else {
      setPluginUIEnabled(s.pluginName, on)
      if (on) setSlotOwner(s.slotId, s.pluginName) // 启用时显式占用该区域
    }
  }
  // ★ ui-sidebar 承载插件面板：停用后侧边栏（含本面板）立即卸载，入口消失。
  //   恢复途径：右下角壳级逃生口按钮（ShellApp 浮动面板，不依赖插件）。
  const recover = p.name === 'ui-sidebar' && !on ? '（已停用；恢复入口：右下角壳级按钮）' : ''
  window.$toast && window.$toast(on ? '已启用 ' + p.name + ' 的 UI（' + slots.map(s => s.slotId).join(', ') + '）' : '已停用 ' + p.name + ' 的 UI（区域恢复空态）' + recover, on ? 'info' : 'warn')
  emitSlotChanged()
}

// ─── 操作 ────────────────────────────────────────────────────
async function doAction(p, action) {
  try {
    await api.pluginAction(p.name, action)
    if (action === 'undefine') {
      unloadClientHalf(p.name)
      delete expanded[p.name]
      plugins.value = plugins.value.filter(x => x.name !== p.name)
    } else {
      await refresh()
    }
    window.$toast && window.$toast(`${action === 'start' ? '已启动' : action === 'stop' ? '已停止' : '已删除'} ${p.name}`, 'info')
  } catch (e) {
    window.$toast && window.$toast(e.message || '操作失败', 'error')
  }
}

// ─── 新建 ────────────────────────────────────────────────────
async function doDefine() {
  defining.value = true
  newMsg.value = ''
  newMsgErr.value = false
  try {
    const res = await api.definePlugin({
      purpose: newForm.purpose,
      code: newForm.code,
      client: newForm.client || undefined,
      language: newForm.language || undefined,
      run: newForm.run,
    })
    newMsg.value = `已定义 ${res.id}（${res.state}）`
    showNew.value = false
    await refresh()
  } catch (e) {
    newMsgErr.value = true
    newMsg.value = e.message || '定义失败'
  } finally {
    defining.value = false
  }
}

function copyText(t) {
  try {
    navigator.clipboard.writeText(t)
    window.$toast && window.$toast('已复制', 'info')
  } catch (e) {}
}

// ─── client 面板渲染 ─────────────────────────────────────────
function selectPanel(id) {
  activePanelId.value = id
  renderActivePanel()
}

async function renderActivePanel() {
  await nextTick()
  const el = clientPanelEl.value
  if (!el) return
  el.innerHTML = ''
  const panel = clientPanels.find(p => p.id === activePanelId.value)
  if (panel && panel.render) {
    try {
      // render(el, ui)：ui 为该 client 半的沙箱对象（invoke/emit/on 等）
      panel.render(el, getUIFor(panel.pluginName))
    } catch (e) {
      console.warn('[plugin] 面板渲染错误', panel.id, e)
      el.innerHTML = '<div style="color:var(--text-muted);padding:8px;font-size:12px">面板渲染失败</div>'
    }
  }
}

// 面板列表变化时重渲染
function onPanelsChanged(panels) {
  if (!activePanelId.value || !panels.some(p => p.id === activePanelId.value)) {
    activePanelId.value = panels.length ? panels[0].id : ''
  }
  renderActivePanel()
}

// ─── UI 槽位管理（Slot 系统：内置区域 + 插件占用统一装配视图）────────────────
const slotGroups = ref([])
let slotUnsub = null

function refreshSlots() {
  // ★ 按 (slotId, kind) 分组：同一区域可同时有 single 替换与 list 叠加占用
  //   （如 activitybar），两类控件（下拉/勾选）互不干扰。
  // ★ 一切皆插件（2026-08-16）：槽位完全由磁盘插件 client 半注册
  //   （clientSlots）；壳不再硬编码内置槽位（registerBuiltinSlot 已移除），
  //   面板只展示插件注册的槽位与占用者。
  const keys = [...new Set(clientSlots.map(s => s.slotId + '::' + s.kind))]
  slotGroups.value = keys.map(k => {
    const [slotId, kind] = k.split('::')
    const candidates = getSlotCandidates(slotId).filter(c => c.kind === kind)
    return { slotId, kind, owner: getSlotOwner(slotId), candidates, builtin: null }
  })
}

// list 槽位（叠加）：勾选 = 该条目参与渲染（localStorage 持久化）
function overlayActive(slotId, pluginName) { return isOverlayActive(slotId, pluginName) }
function toggleOverlay(slotId, pluginName, on) { setOverlayActive(slotId, pluginName, on) }

function switchSlot(slotId, pluginName) {
  setSlotOwner(slotId, pluginName || '')
  refreshSlots()
}

onMounted(() => {
  setPanelMount(onPanelsChanged)
  slotUnsub = setSlotMount(refreshSlots)
  startPolling()
  refresh()
})

onUnmounted(() => {
  stopPolling()
  setPanelMount(null)
  if (slotUnsub) { slotUnsub(); slotUnsub = null }
})
</script>

<style scoped>
.plugin-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  font-size: 13px;
}
.pp-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}
.pp-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  font-size: 12px;
  color: var(--text-primary);
}
.pp-actions { display: flex; gap: 4px; }
.pp-icon-btn {
  background: none; border: none; cursor: pointer;
  color: var(--text-muted); padding: 2px 4px; border-radius: 3px;
  display: flex; align-items: center;
}
.pp-icon-btn:hover { background: var(--bg-hover); color: var(--text-primary); }

/* 新建表单 */
.pp-new {
  padding: 10px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  gap: 6px;
  background: var(--bg-tertiary);
  flex-shrink: 0;
  max-height: 45%;
  overflow: auto;
}
.pp-new-title { font-size: 12px; font-weight: 600; color: var(--text-secondary); }
.pp-input, .pp-textarea {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  border-radius: 4px;
  padding: 5px 8px;
  font-size: 12px;
  width: 100%;
  box-sizing: border-box;
}
.pp-textarea.code {
  font-family: var(--font-code);
  font-size: 11px;
  line-height: 1.5;
  resize: vertical;
}
.pp-new-foot { display: flex; align-items: center; gap: 8px; }
.pp-lang { width: auto; flex-shrink: 0; }
.pp-check { display: flex; align-items: center; gap: 4px; font-size: 11px; color: var(--text-secondary); white-space: nowrap; }
.pp-new-msg { font-size: 11px; color: var(--accent-light); word-break: break-all; }
.pp-new-msg.err { color: var(--error, #e06c75); }

/* client 面板区 */
.pp-client {
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  max-height: 40%;
}
/* UI 槽位区（Slot 系统） */
.pp-slots {
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
  padding: 6px 8px;
  display: flex;
  flex-direction: column;
  gap: 5px;
  max-height: 30%;
  overflow-y: auto;
}
.pp-slots-head {
  display: flex; align-items: center; gap: 6px;
  font-size: 11px; color: var(--text-primary); font-weight: 600;
}
.pp-slots-head svg { color: var(--accent); }
.pp-slots-sub { font-weight: 400; font-size: 10px; color: var(--text-muted); }
.pp-slot-row {
  display: flex; align-items: center; justify-content: space-between; gap: 8px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-left: 2px solid var(--accent);
  border-radius: 6px; padding: 5px 8px;
  transition: border-color .12s, background .12s;
}
.pp-slot-row:hover {
  border-color: color-mix(in srgb, var(--accent) 45%, var(--border-color));
  background: var(--bg-hover);
}
.pp-slot-info { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.pp-slot-title-row { display: flex; align-items: center; gap: 6px; min-width: 0; }
.pp-slot-id {
  font-family: var(--font-mono, monospace); font-size: 11px; color: var(--accent-light);
  text-transform: uppercase; letter-spacing: 0.5px; font-weight: 600;
}
.pp-slot-owner { font-size: 10px; color: var(--accent); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 180px; }
.pp-slot-owner.builtin { color: var(--text-muted); }
.pp-slot-kind {
  font-size: 9px; border-radius: 4px; padding: 0 5px; align-self: flex-start;
  line-height: 15px; flex-shrink: 0; font-weight: 600; letter-spacing: .3px;
}
.pp-slot-kind.kind-single { color: var(--accent-light); background: color-mix(in srgb, var(--accent) 14%, transparent); border: 1px solid color-mix(in srgb, var(--accent) 35%, transparent); }
.pp-slot-kind.kind-list { color: #3fb950; background: rgba(63, 185, 80, .10); border: 1px solid rgba(63, 185, 80, .30); }
.pp-slot-list { display: flex; flex-direction: column; gap: 3px; align-items: flex-end; flex-shrink: 0; }
.pp-slot-list-item {
  display: flex; align-items: center; gap: 4px; font-size: 10px; color: var(--text-secondary);
  cursor: pointer; max-width: 220px; padding: 1px 4px; border-radius: 4px;
  transition: background .1s, color .1s;
}
.pp-slot-list-item:hover { color: var(--text-primary); background: var(--bg-hover); }
.pp-slot-list-item input[type='checkbox'] { accent-color: var(--accent); margin: 0; }
.pp-slot-list-item span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pp-slot-empty { font-size: 10px; color: var(--text-muted); }
.pp-slot-select {
  width: 170px; font-size: 11px; padding: 3px 6px;
  background: var(--bg-secondary); color: var(--text-primary);
  border: 1px solid var(--border-color); border-radius: 5px; flex-shrink: 0;
  cursor: pointer; transition: border-color .12s, box-shadow .12s;
}
.pp-slot-select:hover { border-color: var(--accent); }
.pp-slot-select:focus { outline: none; border-color: var(--accent); box-shadow: 0 0 0 2px var(--focus-ring); }
.pp-client-tabs {
  display: flex;
  gap: 2px;
  padding: 4px 8px 0;
  border-bottom: 1px solid var(--border-color);
  overflow-x: auto;
}
.pp-client-tab {
  display: flex; align-items: center; gap: 4px;
  padding: 4px 10px;
  font-size: 11px;
  color: var(--text-secondary);
  cursor: pointer;
  border: 1px solid transparent;
  border-bottom: none;
  border-radius: 4px 4px 0 0;
  white-space: nowrap;
}
.pp-client-tab.active {
  background: var(--bg-primary);
  color: var(--text-primary);
  border-color: var(--border-color);
}
.pp-client-tab-title { max-width: 120px; overflow: hidden; text-overflow: ellipsis; }
.pp-client-body {
  min-height: 80px;
  max-height: 200px;
  overflow: auto;
  padding: 6px 8px;
  font-size: 12px;
}

/* 列表 */
.pp-list { flex: 1; overflow: auto; padding: 4px 0; }
.pp-loading, .pp-empty {
  display: flex; flex-direction: column; align-items: center; gap: 6px;
  padding: 24px 12px; color: var(--text-muted); font-size: 12px;
}
.pp-empty-sub { font-size: 11px; color: var(--text-muted); text-align: center; }
.pp-item { border-bottom: 1px solid var(--border-color); }
.pp-item-row {
  display: flex; align-items: center; gap: 6px;
  padding: 7px 10px;
  cursor: pointer;
}
.pp-item-row:hover { background: var(--bg-hover); }
.pp-state {
  width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0;
}
.pp-state.on { background: #4caf50; box-shadow: 0 0 4px rgba(76, 175, 80, .6); }
.pp-state.off { background: var(--text-muted); opacity: .4; }
.pp-name {
  flex: 1; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.pp-src {
  font-size: 9px; padding: 1px 5px; border-radius: 3px;
  font-family: var(--font-code); text-transform: uppercase;
}
.pp-src.js { background: rgba(240, 219, 79, .15); color: #e5c07b; }
.pp-src.go { background: rgba(0, 178, 255, .12); color: #61afef; }
.pp-badge {
  font-size: 9px; padding: 1px 5px; border-radius: 3px;
  background: rgba(198, 120, 221, .15); color: #c678dd;
  flex-shrink: 0;
}
.pp-badge-warn {
  background: rgba(229, 192, 123, .18); color: #e5c07b;
  cursor: help;
}
.pp-count { font-size: 10px; color: var(--text-muted); flex-shrink: 0; }
.pp-ui-label { font-size: 10px; color: var(--text-muted); flex-shrink: 0; }
.pp-ui-label.on { color: var(--accent, #4c9aff); }
.pp-chevron { transition: transform .15s; flex-shrink: 0; }
.pp-chevron.open { transform: rotate(90deg); }

.pp-detail { padding: 4px 10px 10px 24px; background: var(--bg-tertiary); }
.pp-d-purpose { font-size: 12px; color: var(--text-secondary); margin-bottom: 4px; }
.pp-d-line { font-size: 11px; color: var(--text-muted); margin: 2px 0; word-break: break-all; }
.pp-d-tools { display: flex; flex-direction: column; gap: 1px; margin: 4px 0; padding: 4px 6px; border: 1px solid var(--border-color); border-radius: 4px; background: var(--bg-primary); }
.pp-d-tools-title { font-size: 10px; color: var(--text-muted); margin-bottom: 2px; }
.pp-d-tool { display: flex; align-items: center; justify-content: space-between; gap: 8px; padding: 1px 2px; border-radius: 3px; }
.pp-d-tool:hover { background: var(--bg-secondary); }
.pp-d-tname { font-family: var(--font-code); font-size: 11px; color: var(--text-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pp-d-code {
  margin-top: 6px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  overflow: hidden;
}
.pp-d-code-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 3px 8px;
  background: var(--bg-primary);
  font-size: 10px; color: var(--text-secondary);
  border-bottom: 1px solid var(--border-color);
}
.pp-d-code pre {
  margin: 0; padding: 6px 8px;
  font-family: var(--font-code); font-size: 10px;
  line-height: 1.5;
  color: var(--text-secondary);
  overflow: auto;
  max-height: 160px;
  white-space: pre-wrap;
  word-break: break-all;
}
.pp-d-actions { display: flex; gap: 6px; margin-top: 8px; }

.pp-btn {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  border-radius: 4px;
  padding: 3px 10px;
  font-size: 11px;
  cursor: pointer;
}
.pp-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.pp-btn.primary { border-color: var(--accent); color: var(--accent-light); }
.pp-btn.danger { border-color: #e06c75; color: #e06c75; }
.pp-btn:disabled { opacity: .5; cursor: not-allowed; }
.spinner { animation: pp-spin 1s linear infinite; }
@keyframes pp-spin { to { transform: rotate(360deg); } }
</style>

/* ─── 工具集管理区 ─── */
.pp-toolset {
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  padding: 8px 10px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex-shrink: 0;
  max-height: 46%;
  overflow: auto;
}
.pp-ts-head { display: flex; gap: 6px; align-items: center; }
.pp-ts-head select { flex: 1; }
.pp-ts-body { display: flex; flex-direction: column; gap: 6px; }
.pp-ts-title { font-size: 12px; font-weight: 600; color: var(--text-primary); }
.pp-ts-scope { font-weight: 400; color: var(--text-muted); font-size: 11px; }
.pp-ts-plugin {
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--bg-primary);
  padding: 6px 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.pp-ts-prow { display: flex; align-items: center; justify-content: space-between; gap: 6px; }
.pp-ts-pname { font-weight: 600; font-size: 12px; word-break: break-all; }
.pp-ts-purpose { font-size: 11px; color: var(--text-muted); }
.pp-ts-tools { display: flex; flex-wrap: wrap; gap: 4px 10px; }
.pp-ts-tool {
  display: flex; align-items: center; gap: 3px;
  font-size: 11px; color: var(--text-secondary);
  cursor: pointer;
}
.pp-ts-tool input { margin: 0; cursor: pointer; }
.pp-ts-tool span.off { text-decoration: line-through; color: var(--text-muted); opacity: .6; }
.pp-ts-muted { font-size: 10px; color: var(--text-muted); }
.pp-ts-add { display: flex; gap: 6px; align-items: center; margin-top: 2px; }
.pp-ts-add select { flex: 1; }
.pp-ts-empty { font-size: 11px; color: var(--text-muted); padding: 4px 0; }
.pp-icon-btn.active { color: var(--accent-light); background: var(--bg-hover); }



/* 开关（pp-switch） */
.pp-switch { position: relative; display: inline-flex; align-items: center; cursor: pointer; flex-shrink: 0; }
.pp-switch input { position: absolute; opacity: 0; width: 0; height: 0; }
.pp-switch-track {
  width: 26px; height: 14px;
  background: var(--border-color);
  border-radius: 7px;
  transition: background .15s;
  position: relative;
}
.pp-switch-track::after {
  content: '';
  position: absolute; top: 2px; left: 2px;
  width: 10px; height: 10px;
  background: #fff; border-radius: 50%;
  transition: transform .15s;
}
.pp-switch input:checked + .pp-switch-track { background: var(--accent); }
.pp-switch input:checked + .pp-switch-track::after { transform: translateX(12px); }
