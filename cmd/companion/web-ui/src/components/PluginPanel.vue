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
          <option v-for="t in toolsetMetas" :key="t.name" :value="t.name">{{ t.name }}（{{ t.pluginCount }} 插件{{ t.scope === 'global' ? '·全局' : '' }}）</option>
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
            <button class="pp-btn danger" @click="edit({ action: 'rm_plugin', plugin_name: pl.name })">移出工具集</button>
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

    <!-- 内置工具（被过滤的 pair 独有工具按内置插件组管理；文件浏览器工具集区同源展示） -->
    <div class="pp-builtin">
      <div class="pp-builtin-head" @click="builtinOpen = !builtinOpen" title="点击折叠/展开">
        <span class="pp-builtin-title"><SvgIcon name="package" :size="13" /> 内置工具 <span class="pp-builtin-sub">{{ builtinInfo ? builtinInfo.enabledTotal + '/' + builtinInfo.toolTotal + ' 启用' : '' }}</span></span>
        <div class="pp-builtin-actions" @click.stop>
          <label class="pp-check" title="强制全部内置工具组加入工作区（所有被过滤工具对 agent 可见）">
            <input type="checkbox" :checked="builtinForceAll" @change="forceAllBuiltin" />
            <span>强制全部</span>
          </label>
          <button class="pp-icon-btn" @click="loadBuiltin" title="刷新内置工具状态"><SvgIcon name="refresh" :size="11" :class="{ spinning: builtinLoading }" /></button>
          </div>
          <SvgIcon name="chevron-right" :size="11" class="pp-chevron" :class="{ open: builtinOpen }" />
        </div>
      <div v-if="builtinOpen && builtinLoading && !builtinInfo" class="pp-loading"><SvgIcon name="refresh" :size="12" class="spinner" /><span>加载…</span></div>
      <div v-else-if="builtinOpen && builtinInfo && builtinInfo.groups.length" class="pp-builtin-groups">
        <div v-for="g in builtinInfo.groups" :key="g.name" class="pp-builtin-group">
          <div class="pp-builtin-grow">
            <span class="pp-builtin-gname" :class="{ off: !g.enabled && !g.partial }">{{ g.title }}</span>
            <span class="pp-builtin-gdesc" :title="g.desc">{{ g.desc }}</span>
            <span class="pp-builtin-gtools">{{ g.tools.length }} 工具<template v-if="g.partial">（部分）</template></span>
          </div>
          <label class="pp-switch" :title="g.enabled ? '组内工具全部对 agent 可见；点击移出（恢复默认过滤）' : '加入工作区：组内工具全部对 agent 可见'">
            <input type="checkbox" :checked="g.enabled" @change="toggleBuiltinGroup(g)" />
            <span class="pp-switch-track"></span>
          </label>
        </div>
      </div>
      <div v-else-if="builtinOpen" class="pp-builtin-empty">内置工具包未加载（启动后自动可用）</div>
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
          <span v-if="p.hasClient && p.clientApproved" class="pp-badge" title="含 client 半（浏览器 UI，已批准装载）">UI</span>
          <span v-else-if="p.hasClient && p.state === 'running'" class="pp-badge pp-badge-warn" title="client 半待激活批准：在对话中用 cordis_run 装载该插件触发审批">UI 待批准</span>
          <span v-else-if="p.hasClient" class="pp-badge" title="含 client 半（浏览器 UI；装载后需批准）">UI</span>
          <span v-if="p.tools && p.tools.length" class="pp-count" :title="p.tools.join(', ')">{{ p.tools.length }} 工具</span>
            <SvgIcon name="chevron-right" :size="12" class="pp-chevron" :class="{ open: expanded[p.name] }" />
          </div>
          <div v-if="expanded[p.name]" class="pp-detail">
            <div v-if="p.purpose" class="pp-d-purpose">{{ p.purpose }}</div>
            <div v-if="p.defId" class="pp-d-line">定义: {{ p.defId }}<span v-if="p.version"> · {{ p.version }}</span></div>
            <div v-if="p.provides && p.provides.length" class="pp-d-line">服务: {{ p.provides.join(', ') }}</div>
            <div v-if="p.sections && p.sections.length" class="pp-d-line">提示片段: {{ p.sections.join(', ') }}</div>
            <div v-if="p.tools && p.tools.length" class="pp-d-line">工具: {{ p.tools.join(', ') }}</div>
            <div v-if="p.clientCode" class="pp-d-code">
              <div class="pp-d-code-head">
              <span>client 半源码</span>
              <button class="pp-icon-btn" @click="copyText(p.clientCode)" title="复制"><SvgIcon name="copy" :size="11" /></button>
            </div>
            <pre>{{ p.clientCode }}</pre>
          </div>
          <div class="pp-d-actions">
            <button v-if="p.state === 'running'" class="pp-btn" @click="doAction(p, 'stop')">停止</button>
            <button v-else class="pp-btn primary" @click="doAction(p, 'start')">启动</button>
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
import { clientPanels, syncClientHalves, unloadClientHalf, startPolling, stopPolling, setPanelMount, getUIFor } from '../plugin-runtime.js'

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
const builtinForceAll = ref(false)

async function loadBuiltin() {
  builtinLoading.value = true
  try {
    builtinInfo.value = await api.builtinPlugins()
    builtinForceAll.value = !!(builtinInfo.value && builtinInfo.value.joined && builtinInfo.value.joined.length === (builtinInfo.value.groups || []).length && builtinInfo.value.joined.length > 0)
  } catch (e) {
    builtinInfo.value = null
  } finally {
    builtinLoading.value = false
  }
}

async function toggleBuiltinGroup(g) {
  const target = !g.enabled
  try {
    const res = await api.builtinPlugins({ group: g.name, enabled: target })
    window.$toast && window.$toast((res && res.message) || (target ? '已加入' : '已移出') + ' ' + g.name, 'info')
  } catch (e) {
    window.$toast && window.$toast(e.message || '操作失败', 'error')
  }
  await loadBuiltin()
  await refresh()
}

async function forceAllBuiltin(ev) {
  const target = !!ev.target.checked
  if (!target) {
    // 关闭强制全部：保持当前状态（仅提示如何单个移出）
    window.$toast && window.$toast('已取消强制全部（保持当前启用状态；可在工具集面板 builtin 分组逐个移出）', 'info')
    loadBuiltin()
    return
  }
  try {
    const res = await api.builtinPlugins({ forceAll: true })
    window.$toast && window.$toast((res && res.message) || '已强制全部加入', 'info')
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
    // 内置工具包 builtin 在插件面板顶部「内置工具」区块管理，工具集下拉不重复展示
    toolsetMetas.value = list.filter(t => t.scope !== 'builtin')
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

onMounted(() => {
  setPanelMount(onPanelsChanged)
  startPolling()
  refresh()
  loadBuiltin()
})

onUnmounted(() => {
  stopPolling()
  setPanelMount(null)
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
.pp-chevron { transition: transform .15s; flex-shrink: 0; }
.pp-chevron.open { transform: rotate(90deg); }

.pp-detail { padding: 4px 10px 10px 24px; background: var(--bg-tertiary); }
.pp-d-purpose { font-size: 12px; color: var(--text-secondary); margin-bottom: 4px; }
.pp-d-line { font-size: 11px; color: var(--text-muted); margin: 2px 0; word-break: break-all; }
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
  display: flex; align-items: center; justify-content: space-between; gap: 6px;
  border: 1px solid var(--border-color); border-radius: 4px;
  background: var(--bg-primary); padding: 3px 8px;
}
.pp-builtin-grow { display: flex; align-items: baseline; gap: 6px; min-width: 0; flex: 1; }
.pp-builtin-gname { font-size: 11px; font-weight: 600; color: var(--text-primary); white-space: nowrap; }
.pp-builtin-gname.off { color: var(--text-muted); opacity: .6; }
.pp-builtin-gdesc { font-size: 10px; color: var(--text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; }
.pp-builtin-gtools { font-size: 10px; color: var(--text-secondary); white-space: nowrap; }
.pp-builtin-empty { font-size: 10px; color: var(--text-muted); padding: 2px 0; }

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
