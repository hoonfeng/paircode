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
      <div class="pp-slots-head">
        <span class="pp-slots-title"><SvgIcon name="layers" :size="13" /> UI 槽位</span>
        <span class="pp-slots-sub">插件可替换的界面区域</span>
      </div>
      <div v-for="g in slotGroups" :key="g.slotId + '::' + g.kind" class="pp-slot-row">
        <div class="pp-slot-info">
          <span class="pp-slot-id">{{ g.slotId }}</span>
          <span class="pp-slot-kind">{{ g.kind === 'list' ? '叠加' : '替换' }}</span>
          <span class="pp-slot-owner" :class="{ builtin: !g.owner }">{{ g.owner ? g.owner : '内置组件' }}</span>
        </div>
        <!-- single 槽位：下拉切换占用者；list 槽位：勾选叠加（全部渲染） -->
        <select v-if="g.kind !== 'list'" class="pp-input pp-slot-select" :value="g.owner" @change="switchSlot(g.slotId, $event.target.value)"
                :title="'切换 ' + g.slotId + ' 区域的渲染者'">
          <option value="">内置组件（默认）</option>
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
    </div>

    <!-- 内置工具（全部内置工具：分组 + 工具级开关 + 搜索；文件浏览器工具集区同源展示） -->
    <div class="pp-builtin">
      <div class="pp-builtin-head" @click="builtinOpen = !builtinOpen" :title="builtinOpen ? '点击收起' : '点击展开工具列表'">
        <span class="pp-builtin-title"><SvgIcon name="package" :size="13" /> 内置工具 <span class="pp-builtin-sub">{{ builtinInfo ? builtinInfo.enabledTotal + ' 启用' : '' }}</span></span>
        <div class="pp-builtin-actions" @click.stop>
          <input v-model="builtinQuery" class="pp-input pp-builtin-search" placeholder="搜索工具…" />
          <button class="pp-icon-btn" @click="loadBuiltin" title="刷新内置工具状态"><SvgIcon name="refresh" :size="11" :class="{ spinning: builtinLoading }" /></button>
        </div>
        <SvgIcon name="chevron-right" :size="11" class="pp-chevron" :class="{ open: builtinOpen }" />
      </div>
      <div v-if="builtinOpen && builtinLoading && !builtinInfo" class="pp-loading"><SvgIcon name="refresh" :size="12" class="spinner" /><span>加载…</span></div>
      <template v-else-if="builtinOpen && builtinInfo">
        <!-- 搜索模式：扁平工具列表（跨组） -->
        <div v-if="builtinQuery.trim()" class="pp-builtin-tools">
          <div v-for="t in filteredBuiltinTools" :key="t.name" class="pp-builtin-tool">
            <span class="pp-builtin-tgroup">{{ t.group }}</span>
            <span class="pp-builtin-tname" :class="{ off: !t.enabled }" :title="t.desc">{{ t.name }}</span>
            <span class="pp-builtin-tdesc">{{ t.desc }}</span>
            <label class="pp-switch" :title="t.enabled ? '对 agent 可见；点击移除（恢复默认过滤）' : '加入 agent 可用'">
              <input type="checkbox" :checked="t.enabled" @change="toggleBuiltinTool(t)" />
              <span class="pp-switch-track"></span>
            </label>
          </div>
          <div v-if="!filteredBuiltinTools.length" class="pp-builtin-empty">无匹配工具</div>
        </div>
        <!-- 分组模式：组折叠 + 组内工具行 -->
        <div v-else-if="builtinInfo.groups.length" class="pp-builtin-groups">
          <div v-for="g in builtinInfo.groups" :key="g.name" class="pp-builtin-group">
            <div class="pp-builtin-group-head" @click="builtinGroupOpen[g.name] = !builtinGroupOpen[g.name]" :title="'点击' + (builtinGroupOpen[g.name] ? '收起' : '展开') + '组内工具（工具级开关控制 agent 可见性）'">
              <SvgIcon name="chevron-right" :size="10" class="pp-chevron pp-group-chevron" :class="{ open: builtinGroupOpen[g.name] }" />
              <span class="pp-builtin-gname" :class="{ off: !g.enabled && !g.partial }">{{ g.title }}</span>
              <span class="pp-builtin-gtools">{{ g.tools.length }} 工具<template v-if="g.partial">（部分）</template></span>
              <span class="pp-builtin-gdesc" :title="g.desc">{{ g.desc }}</span>
            </div>
            <div v-if="builtinGroupOpen[g.name]" class="pp-builtin-tools">
              <div v-for="t in g.tools" :key="t.name" class="pp-builtin-tool">
                <span class="pp-builtin-tgroup">{{ g.name }}</span>
                <span class="pp-builtin-tname" :class="{ off: !t.enabled }" :title="t.desc">{{ t.name }}</span>
                <span class="pp-builtin-tdesc">{{ t.desc }}</span>
                <label class="pp-switch" :title="t.enabled ? '对 agent 可见；点击移除（恢复默认过滤）' : '加入 agent 可用'">
                  <input type="checkbox" :checked="t.enabled" @change="toggleBuiltinTool(t)" />
                  <span class="pp-switch-track"></span>
                </label>
              </div>
            </div>
          </div>
        </div>
        <div v-else class="pp-builtin-empty">内置工具包未加载（启动后自动可用）</div>
      </template>
    </div>

    <!-- 插件列表 -->
    <div class="pp-list">
      <div v-if="loading && plugins.length === 0" class="pp-loading">
        <SvgIcon name="refresh" :size="16" class="spinner" /><span>加载插件…</span>
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
            <button v-if="p.state === 'running'" class="pp-btn" title="停止整个插件（其全部工具对 agent 不可见）；单工具请用上方工具开关" @click="doAction(p, 'stop')">停止插件</button>
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
import { clientPanels, clientSlots, syncClientHalves, unloadClientHalf, startPolling, stopPolling, setPanelMount, setSlotMount, getUIFor, getSlotCandidates, getSlotOwner, setSlotOwner, emitSlotChanged, isOverlayActive, setOverlayActive } from '../plugin-runtime.js'

const plugins = ref([])
const loading = ref(false)
const refreshing = ref(false)
const expanded = reactive({})
const showNew = ref(false)
const defining = ref(false)
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

// ─── 内置工具（被过滤工具按内置插件组管理——插件面板开关）──
const builtinInfo = ref(null)
const builtinLoading = ref(false)
const builtinOpen = ref(false) // 内置工具区默认收起：点击头部「内置工具 N 启用」展开工具列表

async function loadBuiltin() {
  builtinLoading.value = true
  try {
    builtinInfo.value = await api.builtinPlugins()
  } catch (e) {
    builtinInfo.value = null
  } finally {
    builtinLoading.value = false
  }
}

// ─── 内置工具：工具级视图 + 搜索 ───
const builtinQuery = ref('')
const builtinGroupOpen = reactive({}) // 分组展开状态（默认全折叠）

// 扁平工具列表（跨组，搜索用）
const flatBuiltinTools = computed(() => {
  if (!builtinInfo.value) return []
  const out = []
  for (const g of builtinInfo.value.groups || []) {
    for (const t of g.tools) out.push({ ...t, group: g.name })
  }
  return out
})
const filteredBuiltinTools = computed(() => {
  const q = builtinQuery.value.trim().toLowerCase()
  if (!q) return []
  return flatBuiltinTools.value.filter(t =>
    (t.name + ' ' + (t.desc || '') + ' ' + t.group).toLowerCase().includes(q))
})

// 工具级开关（agent 可见性——内存态，走 /api/plugins/tool；不固化工作区工具集。
// 内置工具包=过滤落点：默认全量可见（全勾），取消勾选=临时过滤；持久化加入
// 请用工具集机制 toolset_edit add_builtin（固化 .pair/toolsets/*.json））
async function toggleBuiltinTool(t) {
  const target = !t.enabled
  try {
    const res = await api.pluginToolToggle(t.name, target)
    window.$toast && window.$toast((res && res.message) || (target ? '已启用' : '已禁用') + ' ' + t.name, 'info')
  } catch (e) {
    window.$toast && window.$toast(e.message || '操作失败', 'error')
  }
  await loadBuiltin()
  await refresh()
}

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
async function refresh() {
  refreshing.value = true
  loading.value = true
  try {
    const list = await api.listPlugins()
    plugins.value = Array.isArray(list) ? list : []
    // 列表接口省略 clientCode：详情按需补取，供 client 半装载
    for (const p of plugins.value) {
      if (p.hasClient && !p.clientCode) {
        try {
          const d = await api.getPluginDetail(p.name)
          if (d && d.clientCode) p.clientCode = d.clientCode
        } catch (e) { /* 详情失败跳过 */ }
      }
    }
    await syncClientHalves(plugins.value)
  } catch (e) {
    console.warn('[plugin] 加载失败', e)
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
//  single 槽位（statusbar/chat/sidebar）：激活=设为该槽位占用者，停用=恢复内置；
//  list 槽位（overlay）：激活=勾选叠加显示，停用=隐藏。
function uiSlotsOf(pname) {
  return clientSlots.filter(s => s.pluginName === pname)
}
function uiPluginActive(pname) {
  const slots = uiSlotsOf(pname)
  if (!slots.length) return false
  return slots.some(s => {
    if (s.kind === 'list') return isOverlayActive(s.slotId, s.pluginName)
    return getSlotOwner(s.slotId) === s.pluginName
  })
}
function toggleUiPlugin(p, on) {
  const slots = uiSlotsOf(p.name)
  for (const s of slots) {
    if (s.kind === 'list') setOverlayActive(s.slotId, s.pluginName, on)
    else setSlotOwner(s.slotId, on ? s.pluginName : '')
  }
  window.$toast && window.$toast(on ? '已启用 ' + p.name + ' 的 UI（' + slots.map(s => s.slotId).join(', ') + '）' : '已停用 ' + p.name + ' 的 UI（恢复内置界面）', 'info')
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

// ─── UI 槽位管理（Slot 系统）────────────────────────────────
const slotGroups = ref([])
let slotUnsub = null

function refreshSlots() {
  // ★ 按 (slotId, kind) 分组：同一区域可同时有 single 替换与 list 叠加占用
  //   （如 activitybar），两类控件（下拉/勾选）互不干扰。
  const keys = [...new Set(clientSlots.map(s => s.slotId + '::' + s.kind))]
  slotGroups.value = keys.map(k => {
    const [slotId, kind] = k.split('::')
    const candidates = getSlotCandidates(slotId).filter(c => c.kind === kind)
    return { slotId, kind, owner: getSlotOwner(slotId), candidates }
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
  loadBuiltin()
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
  gap: 4px;
  max-height: 30%;
  overflow-y: auto;
}
.pp-slots-head {
  display: flex; align-items: center; gap: 6px;
  font-size: 11px; color: var(--text-primary); font-weight: 600;
}
.pp-slots-sub { font-weight: 400; font-size: 10px; color: var(--text-muted); }
.pp-slot-row {
  display: flex; align-items: center; justify-content: space-between; gap: 8px;
  background: var(--bg-hover); border-radius: 4px; padding: 4px 8px;
}
.pp-slot-info { display: flex; flex-direction: column; gap: 1px; min-width: 0; }
.pp-slot-id {
  font-family: var(--font-mono, monospace); font-size: 11px; color: var(--text-primary);
  text-transform: uppercase; letter-spacing: 0.3px;
}
.pp-slot-owner { font-size: 10px; color: var(--accent); }
.pp-slot-owner.builtin { color: var(--text-muted); }
.pp-slot-kind { font-size: 9px; color: var(--text-muted); background: var(--bg-input, var(--bg-elevated)); border: 1px solid var(--border-color); border-radius: 3px; padding: 0 4px; align-self: flex-start; }
.pp-slot-list { display: flex; flex-direction: column; gap: 2px; align-items: flex-end; flex-shrink: 0; }
.pp-slot-list-item { display: flex; align-items: center; gap: 4px; font-size: 10px; color: var(--text-secondary); cursor: pointer; max-width: 220px; }
.pp-slot-list-item span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pp-slot-empty { font-size: 10px; color: var(--text-muted); }
.pp-slot-select {
  width: 160px; font-size: 11px; padding: 2px 4px;
  background: var(--bg-input, var(--bg-elevated)); color: var(--text-primary);
  border: 1px solid var(--border-color); border-radius: 3px; flex-shrink: 0;
}
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

/* ─── 内置工具（被过滤工具按内置插件组管理） ─── */
.pp-builtin {
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  padding: 6px 10px;
  display: flex;
  flex-direction: column;
  gap: 5px;
  flex-shrink: 0;
  max-height: 34%;
  overflow: auto;
}
.pp-builtin-head { display: flex; align-items: center; justify-content: space-between; gap: 6px; cursor: pointer; user-select: none; }
.pp-builtin-head .pp-builtin-actions { cursor: default; }
.pp-builtin-title { display: flex; align-items: center; gap: 4px; font-size: 11px; font-weight: 600; color: var(--text-primary); }
.pp-builtin-sub { font-weight: 400; color: var(--text-muted); font-size: 10px; }
.pp-builtin-actions { display: flex; align-items: center; gap: 6px; }
.pp-builtin-groups { display: flex; flex-direction: column; gap: 3px; }
.pp-builtin-group {
  display: flex; flex-direction: column; align-items: stretch; gap: 2px;
  border: 1px solid var(--border-color); border-radius: 4px;
  background: var(--bg-primary); padding: 3px 8px;
}
.pp-builtin-grow { display: flex; align-items: baseline; gap: 6px; min-width: 0; flex: 1; }
.pp-builtin-gname { font-size: 11px; font-weight: 600; color: var(--text-primary); white-space: nowrap; }
.pp-builtin-gname.off { color: var(--text-muted); opacity: .6; }
.pp-builtin-gdesc { font-size: 10px; color: var(--text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; }
.pp-builtin-gtools { font-size: 10px; color: var(--text-secondary); white-space: nowrap; }
.pp-builtin-empty { font-size: 10px; color: var(--text-muted); padding: 2px 0; }

/* 内置工具：工具级视图 + 搜索 */
.pp-builtin-search { width: 110px; font-size: 11px; padding: 2px 6px; flex-shrink: 0; }
.pp-builtin-group-head { display: flex; align-items: center; gap: 6px; padding: 2px 0; cursor: pointer; }
.pp-builtin-group-head:hover .pp-builtin-gname { color: var(--accent-light); }
.pp-group-chevron { color: var(--text-muted); flex-shrink: 0; }
.pp-builtin-tools { display: flex; flex-direction: column; gap: 1px; margin-top: 2px; }
.pp-builtin-tool {
  display: flex; align-items: center; gap: 6px; padding: 2px 6px; border-radius: 3px;
  font-size: 11px;
}
.pp-builtin-tool:hover { background: var(--bg-primary); }
.pp-builtin-tgroup {
  font-size: 8px; padding: 0 4px; border-radius: 3px; flex-shrink: 0;
  background: rgba(97, 175, 239, .15); color: #61afef; font-family: var(--font-code);
}
.pp-builtin-tname {
  font-family: var(--font-code); color: var(--text-primary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 55%;
}
.pp-builtin-tname.off { color: var(--text-muted); opacity: .6; }
.pp-builtin-tdesc { font-size: 10px; color: var(--text-muted); flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

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
