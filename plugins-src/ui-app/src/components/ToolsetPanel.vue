<template>
  <!-- 工具集管理面板（2026-09 Round5）：
       列表（名称/插件数/描述）→ 点选详情 → 插件行（工具 chips 开关/移出/加插件）+
       新建（build 模板生成）/ 导入（粘贴或文件）/ 导出（下载）/ 删除 -->
  <div class="tp-panel">
    <!-- 头部 -->
    <div class="tp-header">
      <span class="tp-title"><SvgIcon name="layers" :size="14" /> 工具集</span>
      <div class="tp-actions">
        <button class="tp-icon-btn" @click="loadAll" title="刷新"><SvgIcon name="refresh" :size="13" :class="{ spinning: refreshing }" /></button>
        <button class="tp-icon-btn" @click="openImport = true" title="导入工具集（粘贴 JSON 或选择文件）"><SvgIcon name="download" :size="13" /></button>
        <button class="tp-icon-btn" @click="openBuild = true" title="新建工具集（模板构建 + AI 分析）"><SvgIcon name="plus" :size="14" /></button>
      </div>
    </div>

    <!-- 主体：左栏工具集列表 + 右栏详情（PC master-detail 左右分栏） -->
    <div class="tp-body">
      <!-- 列表 -->
      <div class="tp-list">
      <div v-if="!metas.length && !loading" class="tp-empty">
        暂无工具集。<br />点右上角 + 新建（描述你的项目用途，AI 分析后模板生成插件集）。
      </div>
      <div v-for="m in metas" :key="m.name" class="tp-item" :class="{ active: m.name === selName, builtin: m.scope === 'builtin' }" @click="select(m)">
        <div class="tp-item-main">
          <span class="tp-item-name">{{ m.name }}</span>
          <span class="tp-item-count">{{ m.pluginCount }} 插件</span>
        </div>
        <div class="tp-item-desc">{{ m.description || (m.scope === 'builtin' ? '内置工具包（core/git/codegraph/…）' : '—') }}</div>
      </div>
      <div v-if="loading" class="tp-empty">加载中…</div>
    </div>

    <!-- 详情 -->
    <div v-if="detail" class="tp-detail">
      <div class="tp-dhead">
        <div class="tp-dtitle">
          {{ detail.name }}
          <span v-if="detail.scope === 'builtin'" class="tp-badge">内置</span>
          <span v-else class="tp-badge proj">工作区</span>
        </div>
        <div class="tp-dactions">
          <button class="tp-btn" @click="doExport" title="导出发布 JSON 到本地下载"><SvgIcon name="upload" :size="11" /> 导出</button>
          <button v-if="detail.scope !== 'builtin'" class="tp-btn danger" @click="doRemove" title="删除该工具集（含固化的插件定义）"><SvgIcon name="trash" :size="11" /> 删除</button>
        </div>
        <div v-if="detail.description" class="tp-ddesc">{{ detail.description }}</div>
      </div>

      <!-- 插件行 -->
      <div class="tp-plugins">
        <div class="tp-section-title">插件（{{ (detail.plugins || []).length }}）</div>
        <div v-for="pl in detail.plugins || []" :key="pl.name" class="tp-plugin">
          <div class="tp-prow">
            <span class="tp-pname">{{ pl.name }}</span>
            <button v-if="detail.scope !== 'builtin'" class="tp-btn tiny danger" @click="rmPlugin(pl)" title="从工具集移出（插件定义仍留在宿主）">移出</button>
            <span v-else class="tp-muted">内置</span>
          </div>
          <div v-if="pl.purpose" class="tp-ppurpose">{{ pl.purpose }}</div>
          <div v-if="pluginToolsOf(pl.name).length" class="tp-tools">
            <button v-for="t in pluginToolsOf(pl.name)" :key="t" class="tp-tool" :class="{ off: isToolDisabled(pl, t) }"
                    :title="isToolDisabled(pl, t) ? '已摘除（对 agent 不可见），点击恢复' : '点击摘除（插件保留、工具不可见）'"
                    @click="toggleTool(pl, t)">{{ t }}</button>
          </div>
          <div v-else class="tp-muted">（插件未运行或无工具）</div>
        </div>
        <div v-if="!(detail.plugins || []).length" class="tp-muted">空工具集：点下方「添加插件」加入宿主插件</div>
      </div>

      <!-- 添加插件（浮层卡片列表） -->
      <div v-if="detail.scope !== 'builtin'" class="tp-add">
        <button class="tp-btn tp-add-btn" @click="openAddPlugin = true"><SvgIcon name="plus" :size="12" /> 添加插件</button>
      </div>
    </div>
      <div v-else class="tp-empty tp-side-empty">选择左侧工具集查看详情</div>
    </div>

    <!-- 新建弹层 -->
    <Teleport to="body">
      <Transition name="tp-fade">
        <div v-if="openBuild" class="tp-overlay" @click.self="openBuild = false">
          <div class="tp-sheet" @click.stop>
            <div class="tp-sheet-head">
              <span class="tp-sheet-title">新建工具集</span>
              <button class="tp-cancel" @click="openBuild = false">取消</button>
            </div>
            <div class="tp-sheet-body">
              <label class="tp-field">
                <span class="tp-field-label">名称（小写字母/数字/-/_）</span>
                <input v-model="buildForm.name" class="tp-input" placeholder="如 web-dev / go-backend" />
              </label>
              <label class="tp-field">
                <span class="tp-field-label">描述（可选）</span>
                <input v-model="buildForm.description" class="tp-input" placeholder="工具集用途说明" />
              </label>
              <label class="tp-field">
                <span class="tp-field-label">需求说明（可选，AI 分析项目时参考）</span>
                <textarea v-model="buildForm.requirement" class="tp-input tp-textarea" rows="3" placeholder="如：需要前后端脚手架 + 数据库迁移工具"></textarea>
              </label>
              <label v-if="buildHint" class="tp-hint">检测到同名工具集已存在，勾选后覆盖重建（原插件先卸载）</label>
              <div class="tp-sheet-actions">
                <button class="tp-btn" @click="openBuild = false">取消</button>
                <button class="tp-btn primary" :disabled="buildBusy || !validName(buildForm.name)" @click="doBuild">{{ buildBusy ? '构建中…' : '构建并装载' }}</button>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 导入弹层 -->
    <Teleport to="body">
      <Transition name="tp-fade">
        <div v-if="openImport" class="tp-overlay" @click.self="openImport = false">
          <div class="tp-sheet" @click.stop>
            <div class="tp-sheet-head">
              <span class="tp-sheet-title">导入工具集</span>
              <button class="tp-cancel" @click="openImport = false">取消</button>
            </div>
            <div class="tp-sheet-body">
              <div class="tp-file-row">
                <input ref="fileInput" type="file" accept=".json,application/json" class="tp-hidden-input" @change="onFile" />
                <button class="tp-btn" @click="fileInput && fileInput.click()">选择 JSON 文件…</button>
                <span v-if="fileName" class="tp-muted">{{ fileName }}</span>
              </div>
              <label class="tp-field">
                <span class="tp-field-label">或粘贴工具集 JSON</span>
                <textarea v-model="importJSON" class="tp-input tp-textarea code" rows="9" placeholder='{"name":"my-tools","description":"…","plugins":[…]}'></textarea>
              </label>
              <div class="tp-sheet-actions">
                <button class="tp-btn" @click="openImport = false">取消</button>
                <button class="tp-btn primary" :disabled="importBusy || !importJSON.trim()" @click="doImport">{{ importBusy ? '导入中…' : '导入' }}</button>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 添加插件浮层（插件卡片列表） -->
    <Teleport to="body">
      <Transition name="tp-fade">
        <div v-if="openAddPlugin" class="tp-overlay" @click.self="openAddPlugin = false">
          <div class="tp-sheet tp-add-sheet" @click.stop>
            <div class="tp-sheet-head">
              <span class="tp-sheet-title"><SvgIcon name="puzzle" :size="13" /> 添加宿主插件（{{ filteredAddable.length }}）</span>
              <button class="tp-cancel" @click="openAddPlugin = false">取消</button>
            </div>
            <div class="tp-add-search">
              <SvgIcon name="search" :size="13" class="tp-add-search-icon" />
              <input v-model="addFilter" class="tp-input tp-add-search-input" placeholder="搜索插件名 / 用途…" />
            </div>
            <div class="tp-add-grid">
              <button v-for="p in filteredAddable" :key="p.name" class="tp-pcard" @click="doAddPlugin(p.name)">
                <div class="tp-pcard-head">
                  <span class="tp-pcard-name">{{ p.name }}</span>
                  <span class="tp-pcard-count">{{ p.toolCount }} 工具</span>
                </div>
                <div class="tp-pcard-desc">{{ p.purpose }}</div>
                <div class="tp-pcard-add"><SvgIcon name="plus" :size="11" /> 加入</div>
              </button>
              <div v-if="!filteredAddable.length" class="tp-empty">无匹配插件（或宿主插件已全部加入）</div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import SvgIcon from './SvgIcon.vue'
import api from '../api.js'

const metas = ref([])          // 列表 [{name, scope, pluginCount, description}]
const detail = ref(null)       // 选中工具集详情
const selName = ref('')
const plugins = ref([])        // 宿主插件（/api/plugins）
const loading = ref(false)
const refreshing = ref(false)

const openBuild = ref(false)
const buildForm = reactive({ name: '', description: '', requirement: '' })
const buildBusy = ref(false)
const buildHint = ref(false)

const openImport = ref(false)
const importJSON = ref('')
const fileName = ref('')
const fileInput = ref(null)
const importBusy = ref(false)

const openAddPlugin = ref(false)
const addFilter = ref('')

// ─── 数据加载 ────────────────────────────────────────────────
async function loadAll() {
  refreshing.value = true
  try {
    const list = (await api.getToolsets()) || []
    metas.value = list
    // 选中保持：列表变化后若选中项没了则清空详情
    if (selName.value && !list.some(m => m.name === selName.value)) {
      selName.value = ''
      detail.value = null
    } else if (selName.value && detail.value) {
      await loadDetail(selName.value)
    }
  } catch (e) {
    toastErr('加载工具集列表失败: ' + (e.message || e))
  } finally {
    refreshing.value = false
    loading.value = false
  }
}

async function loadDetail(name) {
  try {
    detail.value = await api.getToolsets(name)
  } catch (e) {
    toastErr('加载工具集详情失败: ' + (e.message || e))
  }
}

async function select(m) {
  selName.value = m.name
  addFilter.value = ''
  openAddPlugin.value = false
  await loadDetail(m.name)
}

async function loadHostPlugins() {
  try {
    const list = await api.listPlugins()
    plugins.value = Array.isArray(list) ? list : []
  } catch (e) {
    plugins.value = []
  }
}

// ─── 插件工具清单（从宿主插件列表取——工具是插件运行时注册的）──
function pluginToolsOf(pname) {
  const p = plugins.value.find(x => x.name === pname)
  return (p && (p.tools || [])) || []
}

function isToolDisabled(pl, t) {
  return (pl.disabledTools || []).includes(t)
}

// 可加入的宿主插件（排除已在工具集的）——浮层卡片列表数据源
const addablePlugins = computed(() => {
  const inTs = new Set((detail.value && detail.value.plugins || []).map(p => p.name))
  return plugins.value
    .filter(p => !inTs.has(p.name))
    .map(p => ({ name: p.name, purpose: p.purpose || '（无描述）', toolCount: (p.tools || []).length }))
})

// 浮层内按搜索词过滤
const filteredAddable = computed(() => {
  const q = addFilter.value.trim().toLowerCase()
  if (!q) return addablePlugins.value
  return addablePlugins.value.filter(p =>
    p.name.toLowerCase().includes(q) || p.purpose.toLowerCase().includes(q))
})

// ─── 编辑动作（即时热装载 + 回写固化 JSON）──────────────────
async function edit(data) {
  if (!selName.value) return
  try {
    const res = await api.toolsetEdit({ name: selName.value, ...data })
    window.$toast && window.$toast((res && res.message) || '操作成功', 'info')
    await loadDetail(selName.value)
    await loadAll()
  } catch (e) {
    toastErr(e.message || '操作失败')
  }
}

function toggleTool(pl, t) {
  edit({ action: isToolDisabled(pl, t) ? 'enable_tool' : 'rm_tool', plugin_name: pl.name, tool: t })
}

function rmPlugin(pl) {
  if (!window.confirm(`将插件「${pl.name}」移出工具集 ${selName.value}？`)) return
  edit({ action: 'rm_plugin', plugin_name: pl.name })
}

async function doAddPlugin(name) {
  if (!name) return
  try {
    const res = await api.toolsetEdit({ name: selName.value, action: 'add_plugin', plugin_name: name })
    window.$toast && window.$toast((res && res.message) || `已加入 ${name}`, 'info')
    openAddPlugin.value = false
    addFilter.value = ''
    await loadDetail(selName.value)
    await loadAll()
  } catch (e) {
    toastErr(e.message || '添加插件失败')
  }
}

// ─── 新建（build 模板生成 + 装载 + 固化）─────────────────────
function validName(n) { return /^[a-z0-9_-]+$/.test((n || '').trim()) }

async function doBuild() {
  const name = buildForm.name.trim()
  if (!validName(name)) { toastErr('名称只能含小写字母/数字/-/_'); return }
  buildBusy.value = true
  try {
    const res = await api.apiPost('/toolsets/build', {
      name, description: buildForm.description.trim(),
      requirement: buildForm.requirement.trim(),
      overwrite: buildHint.value,
    })
    window.$toast && window.$toast(`工具集 ${name} 构建完成（${res.pluginCount || 0} 插件）`, 'info')
    openBuild.value = false
    buildForm.name = ''; buildForm.description = ''; buildForm.requirement = ''; buildHint.value = false
    selName.value = name
    await loadAll()
    await loadDetail(name)
  } catch (e) {
    toastErr(e.message || '构建失败')
  } finally {
    buildBusy.value = false
  }
}

// ─── 删除 ───────────────────────────────────────────────────
async function doRemove() {
  if (!selName.value || detail.value.scope === 'builtin') return
  if (!window.confirm(`删除工具集「${selName.value}」？该操作不可撤销。`)) return
  try {
    await api.apiPost('/toolsets/remove', { name: selName.value })
    window.$toast && window.$toast('工具集已删除', 'info')
    selName.value = ''
    detail.value = null
    await loadAll()
  } catch (e) {
    toastErr(e.message || '删除失败')
  }
}

// ─── 导出/导入 ───────────────────────────────────────────────
function doExport() {
  if (!selName.value) return
  window.open('/api/toolsets/export?name=' + encodeURIComponent(selName.value), '_blank')
}

function onFile(e) {
  const f = e.target.files && e.target.files[0]
  if (!f) return
  fileName.value = f.name
  const fr = new FileReader()
  fr.onload = () => { importJSON.value = String(fr.result || '') }
  fr.onerror = () => toastErr('读取文件失败')
  fr.readAsText(f)
}

async function doImport() {
  const json = importJSON.value.trim()
  if (!json) { toastErr('请提供工具集 JSON'); return }
  importBusy.value = true
  try {
    const res = await api.apiPost('/toolsets/import', { json, scope: 'project' })
    const msg = (res && res.message) || '导入成功'
    window.$toast && window.$toast(msg, 'info')
    openImport.value = false
    importJSON.value = ''
    fileName.value = ''
    // 导入成功后可尝试选中（未知 name 则刷新列表）
    await loadAll()
  } catch (e) {
    toastErr(e.message || '导入失败')
  } finally {
    importBusy.value = false
  }
}

function toastErr(msg) {
  console.warn('[toolset]', msg)
  window.$toast && window.$toast(msg, 'error')
}

onMounted(async () => {
  loading.value = true
  await Promise.all([loadAll(), loadHostPlugins()])
})
</script>

<style scoped>
.tp-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  padding: 8px;
  gap: 8px;
}

/* ── 头部 ── */
.tp-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 2px 2px 6px;
  border-bottom: 1px solid var(--border-color, #3a3a4a);
}
.tp-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary, #eee);
}
.tp-actions { display: inline-flex; gap: 4px; }
.tp-icon-btn {
  width: 26px; height: 26px;
  display: inline-flex; align-items: center; justify-content: center;
  background: none; border: none; border-radius: 6px;
  color: var(--text-muted, #888);
  cursor: pointer;
}
.tp-icon-btn:hover { color: var(--text-primary, #eee); background: var(--bg-hover, rgba(255,255,255,0.06)); }

/* ── 主体（左栏列表 + 右栏详情） ── */
.tp-body {
  flex: 1;
  min-height: 0;
  display: flex;
  gap: 8px;
}

/* ── 列表（左栏） ── */
.tp-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 200px;
  flex-shrink: 0;
  overflow-y: auto;
  padding-right: 6px;
  border-right: 1px solid var(--border-color, #3a3a4a);
}
.tp-item {
  padding: 6px 8px;
  border-radius: 8px;
  border: 1px solid transparent;
  cursor: pointer;
  transition: background 0.12s, border-color 0.12s;
}
.tp-item:hover { background: var(--bg-hover, rgba(255,255,255,0.05)); }
.tp-item.active {
  background: var(--bg-tertiary, rgba(0,0,0,0.15));
  border-color: var(--accent, #4f8cff);
  box-shadow: inset 2px 0 0 var(--accent, #4f8cff);
}
.tp-item-main { display: flex; align-items: center; gap: 6px; }
.tp-item-name { font-size: 12.5px; font-weight: 600; color: var(--text-primary, #eee); }
.tp-item-count {
  font-size: 10px; color: var(--text-muted, #888);
  background: var(--bg-tertiary, rgba(0,0,0,0.2));
  padding: 1px 6px; border-radius: 999px;
}
.tp-item-desc {
  font-size: 11px; color: var(--text-muted, #888);
  margin-top: 2px;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}

/* ── 详情（右栏） ── */
.tp-detail {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow: hidden;
  padding-left: 4px;
}
.tp-dhead { padding: 0 2px; }
.tp-dtitle {
  display: flex; align-items: center; gap: 6px;
  font-size: 13px; font-weight: 600; color: var(--text-primary, #eee);
}
.tp-badge {
  font-size: 9.5px; padding: 1px 6px; border-radius: 999px;
  background: var(--accent, #4f8cff); color: #fff; font-weight: 500;
}
.tp-badge.proj { background: rgba(79, 140, 255, 0.25); color: var(--accent-light, #9dc0ff); }
.tp-dactions { display: flex; gap: 6px; margin-top: 6px; }
.tp-ddesc { font-size: 11px; color: var(--text-muted, #888); margin-top: 4px; }

.tp-section-title {
  font-size: 10px; color: var(--text-muted, #888);
  text-transform: uppercase; letter-spacing: 0.5px; font-weight: 500;
  margin: 2px 0 4px;
}
.tp-plugins {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 0 2px;
}
.tp-plugin {
  border: 1px solid var(--border-color, #3a3a4a);
  border-radius: 8px;
  padding: 6px 8px;
  background: var(--bg-secondary, rgba(255,255,255,0.02));
}
.tp-prow { display: flex; align-items: center; justify-content: space-between; gap: 6px; }
.tp-pname { font-size: 12px; font-weight: 600; color: var(--text-primary, #eee); font-family: var(--font-code, monospace); }
.tp-ppurpose { font-size: 11px; color: var(--text-muted, #888); margin-top: 2px; }
.tp-tools {
  display: flex; flex-wrap: wrap; gap: 4px; margin-top: 5px;
}
.tp-tool {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 11px;
  color: var(--text-secondary, #bbb);
  background: var(--bg-tertiary, rgba(0,0,0,0.15));
  border: 1px solid var(--border-color, #3a3a4a);
  border-radius: 999px;
  padding: 2px 9px;
  cursor: pointer;
  user-select: none;
  font-family: inherit;
  line-height: 1.5;
  transition: border-color 0.12s, color 0.12s, opacity 0.12s;
}
.tp-tool:hover { border-color: var(--accent, #4f8cff); color: var(--text-primary, #eee); }
.tp-tool.off { text-decoration: line-through; opacity: 0.45; border-color: transparent; }
.tp-muted { font-size: 11px; color: var(--text-muted, #888); }

/* ── 按钮 ── */
.tp-btn {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 11.5px; font-family: inherit;
  color: var(--text-primary, #eee);
  background: var(--bg-tertiary, rgba(0,0,0,0.15));
  border: 1px solid var(--border-color, #3a3a4a);
  border-radius: 6px;
  padding: 4px 10px;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.tp-btn:hover { border-color: var(--accent, #4f8cff); }
.tp-btn.tiny { padding: 2px 7px; font-size: 10.5px; }
.tp-btn.primary { background: var(--accent, #4f8cff); border-color: var(--accent, #4f8cff); color: #fff; }
.tp-btn.primary:disabled { opacity: 0.5; cursor: not-allowed; }
.tp-btn.danger { color: #ff6b6b; }
.tp-btn.danger:hover { border-color: #ff6b6b; }

/* ── 添加插件 ── */
.tp-add { padding: 2px; flex-shrink: 0; }
.tp-add-btn {
  width: 100%; justify-content: center;
  border-style: dashed; color: var(--accent, #4f8cff);
}
.tp-add-btn:hover { background: rgba(79, 140, 255, 0.08); }

/* ── 添加插件浮层（卡片 grid） ── */
/* 双 class 提高优先级，覆盖 .tp-sheet 的 480px（浮层更宽以容纳卡片墙） */
.tp-sheet.tp-add-sheet { width: min(680px, 92vw); }
.tp-add-search { position: relative; margin: 0 16px 10px; }
.tp-add-search-icon {
  position: absolute; left: 10px; top: 50%; transform: translateY(-50%);
  color: var(--text-muted, #888); pointer-events: none;
}
.tp-add-search-input { width: 100%; padding-left: 30px; }
.tp-add-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 8px;
  padding: 4px 16px 16px;
  overflow-y: auto;
  max-height: 60vh;
}
.tp-pcard {
  display: flex; flex-direction: column; gap: 5px;
  text-align: left;
  padding: 9px 10px;
  background: var(--bg-tertiary, rgba(0, 0, 0, 0.15));
  border: 1px solid var(--border-color, #3a3a4a);
  border-radius: 9px;
  cursor: pointer;
  font-family: inherit;
  transition: border-color 0.13s, background 0.13s, box-shadow 0.13s;
}
.tp-pcard:hover {
  border-color: var(--accent, #4f8cff);
  background: rgba(79, 140, 255, 0.06);
  box-shadow: 0 0 0 3px rgba(79, 140, 255, 0.12);
}
.tp-pcard-head { display: flex; align-items: center; justify-content: space-between; gap: 6px; }
.tp-pcard-name {
  font-size: 12px; font-weight: 600; color: var(--text-primary, #eee);
  font-family: var(--font-code, monospace);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.tp-pcard-count {
  font-size: 9.5px; color: var(--text-muted, #888);
  background: var(--bg-secondary, rgba(255, 255, 255, 0.05));
  padding: 1px 6px; border-radius: 999px; flex-shrink: 0;
}
.tp-pcard-desc {
  font-size: 11px; color: var(--text-muted, #888); line-height: 1.45;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;
  min-height: 32px;
}
.tp-pcard-add {
  display: inline-flex; align-items: center; gap: 3px;
  font-size: 10.5px; color: var(--accent, #4f8cff);
  margin-top: 2px;
}

/* ── 空态 ── */
.tp-empty {
  font-size: 11.5px; color: var(--text-muted, #888);
  padding: 16px 8px; text-align: center; line-height: 1.7;
}
.tp-side-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* ── 弹层（居中 modal，桌面端更合理；2026-09-05 从 bottom-sheet 改造） ── */
.tp-overlay {
  position: fixed; inset: 0;
  background: rgba(0, 0, 0, 0.45);
  z-index: 10050;
  display: flex; align-items: center; justify-content: center;
  padding: 16px;
}
.tp-sheet {
  width: min(480px, 94vw);
  max-height: 84vh;
  display: flex; flex-direction: column;
  background: var(--bg-secondary, #1c1c28);
  border: 1px solid var(--border-color, #3a3a4a);
  border-radius: 14px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.45);
  animation: tp-sheet-in 0.18s ease-out;
}
@keyframes tp-sheet-in {
  from { transform: translateY(10px) scale(0.98); opacity: 0.6; }
  to { transform: translateY(0) scale(1); opacity: 1; }
}
.tp-sheet-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 16px 8px;
}
.tp-sheet-title { font-size: 14px; font-weight: 600; color: var(--text-primary, #eee); }
.tp-cancel { background: none; border: none; color: var(--accent, #4f8cff); font-size: 13px; cursor: pointer; font-family: inherit; }
.tp-sheet-body { padding: 4px 16px 16px; display: flex; flex-direction: column; gap: 10px; overflow-y: auto; }
.tp-field { display: flex; flex-direction: column; gap: 4px; }
.tp-field-label { font-size: 11px; color: var(--text-muted, #888); }
.tp-input {
  font-size: 12.5px; font-family: inherit;
  color: var(--text-primary, #eee);
  background: var(--bg-tertiary, rgba(0,0,0,0.15));
  border: 1px solid var(--border-color, #3a3a4a);
  border-radius: 8px;
  padding: 7px 10px;
  outline: none;
}
.tp-input:focus { border-color: var(--accent, #4f8cff); }
.tp-textarea { resize: vertical; min-height: 40px; }
.tp-textarea.code { font-family: var(--font-code, monospace); font-size: 11.5px; }
.tp-hint { font-size: 11px; color: #ffb84d; }
.tp-sheet-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 4px; }
.tp-file-row { display: flex; align-items: center; gap: 8px; }
.tp-hidden-input { display: none; }

.tp-fade-enter-active, .tp-fade-leave-active { transition: opacity 0.18s; }
.tp-fade-enter-from, .tp-fade-leave-to { opacity: 0; }

.spinning { animation: tp-spin 0.8s linear infinite; }
@keyframes tp-spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
</style>
