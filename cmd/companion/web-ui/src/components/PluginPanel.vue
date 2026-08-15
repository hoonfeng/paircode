<template>
  <div class="plugin-panel">
    <!-- 头部 -->
    <div class="pp-header">
      <span class="pp-title"><SvgIcon name="puzzle" :size="14" /> 插件</span>
      <div class="pp-actions">
        <button class="pp-icon-btn" @click="refresh" title="刷新"><SvgIcon name="refresh" :size="13" :class="{ spinning: refreshing }" /></button>
        <button class="pp-icon-btn" @click="showNew = !showNew" title="新建插件"><SvgIcon name="plus" :size="14" /></button>
      </div>
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

    <!-- 工具集（动态构建/固化/导出/导入） -->
    <div class="pp-toolsets">
      <div class="pp-ts-header">
        <span class="pp-ts-title"><SvgIcon name="package" :size="13" /> 工具集</span>
        <div class="pp-ts-actions">
          <button class="pp-icon-btn" @click="loadToolsets" title="刷新工具集"><SvgIcon name="refresh" :size="12" :class="{ spinning: tsRefreshing }" /></button>
          <button class="pp-icon-btn" @click="showTsBuild = !showTsBuild" title="构建工具集"><SvgIcon name="plus" :size="13" /></button>
        </div>
      </div>

      <!-- 构建表单 -->
      <div v-if="showTsBuild" class="pp-ts-build">
        <div class="pp-ts-build-title">动态构建工具集（分析项目 → 模板组合生成插件 → 固化到 .pair/toolsets/）</div>
        <input v-model="tsForm.name" placeholder="工具集名（如 web-dev；默认 default）" class="pp-input" />
        <input v-model="tsForm.description" placeholder="用途描述（可选）" class="pp-input" />
        <input v-model="tsForm.requirement" placeholder="要求（可选）：如「Web 前端脚手架 + 接口调试」" class="pp-input" />
        <div class="pp-ts-build-foot">
          <label class="pp-check"><input type="checkbox" v-model="tsForm.overwrite" /> 覆盖已固化同名工具集</label>
          <button class="pp-btn primary" :disabled="tsBuilding" @click="buildToolset">
            {{ tsBuilding ? '构建中…' : '构建并固化' }}
          </button>
        </div>
        <div v-if="tsMsg" class="pp-new-msg" :class="{ err: tsMsgErr }">{{ tsMsg }}</div>
      </div>

      <!-- 工具集列表 -->
      <div v-if="toolsets.length" class="pp-ts-list">
        <div v-for="ts in toolsets" :key="ts.name + '-' + ts.scope" class="pp-ts-item">
          <div class="pp-ts-item-row">
            <span class="pp-state" :class="ts.scope === 'global' ? 'on' : ''"></span>
            <span class="pp-name" :title="ts.description">{{ ts.name }}</span>
            <span class="pp-src">{{ ts.scope === 'global' ? '全局' : '工作区' }}</span>
            <span class="pp-count">{{ ts.pluginCount }} 插件</span>
          </div>
          <div class="pp-ts-item-desc">{{ ts.description }}</div>
          <div class="pp-ts-item-actions">
            <button class="pp-btn" @click="exportToolset(ts)">导出</button>
            <button class="pp-btn danger" @click="removeToolset(ts)">删除</button>
          </div>
        </div>
      </div>
      <div v-else-if="!tsRefreshing" class="pp-ts-empty">
        暂无工具集。点 + 构建（分析项目自动组合插件），或到市场安装插件工具集。
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, nextTick } from 'vue'
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

const newForm = reactive({ purpose: '', code: '', client: '', language: '', run: true })

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
  loadToolsets()
})

// ─── 工具集管理 ─────────────────────────────────────────────
const toolsets = ref([])
const tsRefreshing = ref(false)
const tsBuilding = ref(false)
const showTsBuild = ref(false)
const tsMsg = ref('')
const tsMsgErr = ref(false)
const tsForm = reactive({ name: '', description: '', requirement: '', overwrite: false })

async function loadToolsets() {
  tsRefreshing.value = true
  try {
    const list = await api.apiGet('/toolsets')
    toolsets.value = Array.isArray(list) ? list : []
  } catch (e) {
    console.warn('[toolset] 加载失败', e)
  } finally {
    tsRefreshing.value = false
  }
}

async function buildToolset() {
  tsBuilding.value = true
  tsMsg.value = ''
  tsMsgErr.value = false
  try {
    const res = await api.apiPost('/toolsets/build', {
      name: tsForm.name,
      description: tsForm.description,
      requirement: tsForm.requirement,
      overwrite: tsForm.overwrite,
    })
    tsMsg.value = `已构建并固化「${res.name}」（${res.pluginCount} 个插件）`
    tsForm.name = ''
    tsForm.description = ''
    tsForm.requirement = ''
    tsForm.overwrite = false
    showTsBuild.value = false
    await loadToolsets()
    await refresh() // 插件列表同步（工具集插件已装载）
  } catch (err) {
    tsMsgErr.value = true
    tsMsg.value = '构建失败: ' + (err.message || err)
  } finally {
    tsBuilding.value = false
  }
}

function exportToolset(ts) {
  // 下载发布 JSON（可提交 GitHub 发布市场 / toolset_import 导入）
  const a = document.createElement('a')
  a.href = `/api/toolsets/export?name=${encodeURIComponent(ts.name)}`
  a.download = ts.name + '.toolset.json'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

async function removeToolset(ts) {
  if (!window.confirm(`删除工具集「${ts.name}」（${ts.scope}）？已装载插件将卸载。`)) return
  try {
    await api.apiPost('/toolsets/remove', { name: ts.name, scope: ts.scope === 'global' ? 'global' : 'project' })
    window.$toast?.('已删除工具集 ' + ts.name, 'success')
    await loadToolsets()
    await refresh()
  } catch (err) {
    window.$toast?.('删除失败: ' + (err.message || err), 'error')
  }
}
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
