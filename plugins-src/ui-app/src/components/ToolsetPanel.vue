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

      <!-- 添加插件 -->
      <div v-if="detail.scope !== 'builtin'" class="tp-add">
        <SheetPicker v-model="addName" :items="addableItems" title="选择要加入的宿主插件" placeholder="+ 添加插件…" empty-text="宿主无可加入插件" @change="doAddPlugin" />
      </div>
    </div>
    <div v-else-if="!metas.length" class="tp-empty"></div>

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
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import SvgIcon from './SvgIcon.vue'
import SheetPicker from './SheetPicker.vue'
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

const addName = ref('')

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
  addName.value = ''
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

// 可加入的宿主插件（排除已在工具集 + 已加的 node-bridge 按 name 匹配）
const addableItems = computed(() => {
  const inTs = new Set((detail.value && detail.value.plugins || []).map(p => p.name))
  return plugins.value
    .filter(p => !inTs.has(p.name))
    .map(p => ({ value: p.name, label: p.name, desc: p.purpose || ((p.tools || []).length + ' 工具') }))
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
    addName.value = ''
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

/* ── 列表 ── */
.tp-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 34%;
  overflow-y: auto;
  flex-shrink: 0;
}
.tp-item {
  padding: 6px 8px;
  border-radius: 8px;
  border: 1px solid transparent;
  cursor: pointer;
  transition: background 0.12s, border-color 0.12s;
}
.tp-item:hover { background: var(--bg-hover, rgba(255,255,255,0.05)); }
.tp-item.active { background: var(--bg-tertiary, rgba(0,0,0,0.15)); border-color: var(--accent, #4f8cff); }
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

/* ── 详情 ── */
.tp-detail {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow: hidden;
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

/* ── 空态 ── */
.tp-empty {
  font-size: 11.5px; color: var(--text-muted, #888);
  padding: 16px 8px; text-align: center; line-height: 1.7;
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
