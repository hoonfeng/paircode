<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content">
      <h2><SvgIcon name="settings" :size="18" /> 设置
        <button class="modal-close" @click="$emit('close')">×</button>
      </h2>
      <div class="modal-body">
        <!-- ═══ 纯 schema 驱动：所有配置 tab 由插件 ctx.registerSettings 注册 ═══ -->
        <div v-if="tabs.length" class="settings-tabs">
          <div v-if="tabs.length > 6" class="settings-tabs-filter-wrap">
            <input v-model="tabQuery" class="settings-tabs-filter" type="text" placeholder="筛选设置…" />
          </div>
          <button v-for="t in filteredTabs" :key="t.key" :class="['settings-tab', { active: activeTab === t.key }]"
                  @click="activeTab = t.key">{{ t.title }}</button>
          <div v-if="filteredTabs.length === 0" class="settings-tabs-none">无匹配设置</div>
        </div>
        <div class="settings-content">
          <template v-for="tab in tabs" :key="tab.key">
            <div v-if="activeTab === tab.key">
              <div v-for="grp in tab.groups" :key="grp.title || '__main'" class="setting-group">
                <div v-if="grp.title" class="group-title">{{ grp.title }}</div>
                <div v-for="f in grp.fields" :key="f.name" class="setting-row"
                     :class="{ 'row-toggle': f.type === 'checkbox' }">
                  <!-- checkbox：label 与开关同行 -->
                  <template v-if="f.type === 'checkbox'">
                    <label class="field-label" :title="f.hint">{{ f.label }}</label>
                    <label class="pp-switch" :title="f.hint">
                      <input type="checkbox" v-model="form[tab.key][f.name]" />
                      <span class="pp-switch-track"></span>
                    </label>
                  </template>

                  <!-- 其他类型：label 在上、控件在下、说明文字在控件下方（不挤占输入区） -->
                  <template v-else>
                    <label class="field-label" :title="f.hint">{{ f.label }}</label>
                    <div class="field-control">
                      <!-- text / password -->
                      <input v-if="f.type === 'text' || f.type === 'password'" class="field-input" :type="f.type === 'password' ? 'password' : 'text'"
                             v-model="form[tab.key][f.name]" :placeholder="f.placeholder" />

                      <!-- number -->
                      <input v-else-if="f.type === 'number'" class="field-input" type="number" v-model.number="form[tab.key][f.name]"
                             :min="f.min" :max="f.max" :step="f.step" />

                      <!-- select（optionsSource 驱动动态数据源：models=按服务商模型列表 / providers=服务商列表） -->
                      <select v-else-if="f.type === 'select'" v-model="form[tab.key][f.name]" class="field-select" @change="onSelectChange(f)">
                        <option v-for="o in dynamicOptions(tab.key, f)" :key="o" :value="o">{{ o }}</option>
                      </select>

                      <!-- textarea -->
                      <textarea v-else-if="f.type === 'textarea'" v-model="form[tab.key][f.name]" class="field-textarea"
                                rows="4" :placeholder="f.placeholder"></textarea>

                      <!-- slider -->
                      <div v-else-if="f.type === 'slider'" class="slider-row">
                        <input type="range" v-model.number="form[tab.key][f.name]"
                               :min="f.min != null ? f.min : 0" :max="f.max != null ? f.max : 100" :step="f.step || 1" />
                        <span class="slider-val">{{ form[tab.key][f.name] }}</span>
                      </div>

                      <!-- color -->
                      <div v-else-if="f.type === 'color'" class="color-row">
                        <input type="color" v-model="form[tab.key][f.name]" />
                        <code class="color-code">{{ form[tab.key][f.name] }}</code>
                      </div>

                      <!-- tags（逗号分隔数组） -->
                      <input v-else-if="f.type === 'tags'" type="text" class="field-input"
                             :value="tagsText(tab.key, f)" @input="onTagsInput(tab.key, f, $event)"
                             :placeholder="f.placeholder || '逗号分隔'" />

                      <!-- project（平台特殊：项目级指令，经 /api/instructions 读写） -->
                      <textarea v-else-if="f.type === 'project'" v-model="projectInst" class="field-textarea"
                                rows="4" :placeholder="f.placeholder"></textarea>

                      <!-- provider-manager（服务商维护面板：CRUD /api/models，独立保存，不参与普通表单） -->
                      <ProviderManager v-else-if="f.type === 'provider-manager'" :model-param-fields="f.modelParamFields || []" :model-editor="f.modelEditor || {}" @saved="loadModels" />

                      <!-- preset-manager（AI 配置预设面板：CRUD /api/ai-presets，独立保存，不参与普通表单） -->
                      <PresetManager v-else-if="f.type === 'preset-manager'" :preset-fields="f.presetFields || []" @saved="onPresetSaved" />
                      
                      <!-- 兜底 text -->
                      <input v-else class="field-input" type="text" v-model="form[tab.key][f.name]" />
                    </div>

                    <span v-if="f.hint" class="setting-hint">{{ f.hint }}</span>
                  </template>
                </div>
              </div>
            </div>
          </template>
          <div v-if="!tabs.length" class="settings-empty">暂无配置项（等待插件注册…）</div>
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn-secondary" @click="resetForm">撤销</button>
        <button class="btn-primary" @click="saveSettings">保存设置</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { state, applyTheme } from '../ui-state.js'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'
import ProviderManager from './ProviderManager.vue'
import PresetManager from './PresetManager.vue'

const emit = defineEmits(['close'])
const activeTab = ref('')
const tabQuery = ref('') // ★ 2026-09-01 tab 筛选（tab 太多时快速定位）

// ─── tabs：全部来自插件注册 schema（配置本身无内置）───
const tabs = computed(() => {
  const list = (state.pluginSchemas || []).map(s => ({
    key: s.key,
    title: s.title || s.key,
    groups: groupFields(s.fields || []),
  }))
  if (list.length && !activeTab.value) activeTab.value = list[0].key
  return list
})

// 按标题/key 模糊筛选 tab（active tab 命中时不隐藏，保证内容可见）
const filteredTabs = computed(() => {
  const q = tabQuery.value.trim().toLowerCase()
  if (!q) return tabs.value
  const active = tabs.value.find(t => t.key === activeTab.value)
  const hit = tabs.value.filter(t => (t.title || t.key).toLowerCase().includes(q))
  if (active && !hit.includes(active)) hit.unshift(active)
  return hit
})

function groupFields(fields) {
  const groups = []
  const map = {}
  for (const f of fields) {
    const g = f.group || ''
    if (!map[g]) { map[g] = []; groups.push({ title: g, fields: map[g] }) }
    map[g].push(f)
  }
  return groups
}

// ─── 动态数据源（schema 属性 optionsSource/linkField 驱动；复用已有 /api/models，无新后端）───
// modelData = { providers:[...], models:{provider:[...]}, providerBaseURLs:{provider:url} }
// 由插件注册声明字段行为：optionsSource='providers' 服务商列表 / 'models' 按服务商模型列表；
// linkField='xxx' 选择变化时用 providerBaseURLs 联动填充目标字段（如 provider→baseURL）。
const modelData = ref(null)
let lastProvider = '' // linkField 联动：记录上一个服务商（判断目标字段是否用户自定义）
async function loadModels() {
  try {
    modelData.value = await api.getModels()
  } catch { modelData.value = null }
}
// 按服务商取模型列表（optionsSource='models' 数据源）
function modelsFor(provider) {
  if (!provider) return []
  const m = (modelData.value && modelData.value.models) || {}
  return m[provider] || []
}
// 通用选项计算：f.optionsSource ∈ 'models' | 'providers' | 缺省（静态 f.options）
function dynamicOptions(tabKey, f) {
  if (f.optionsSource === 'models') {
    const cur = form[tabKey]?.[f.name]
    const list = modelsFor(form['ai']?.provider)
    if (cur && !list.includes(cur)) return [...list, cur] // 自定义值兜底显示
    return list
  }
  if (f.optionsSource === 'providers') {
    const list = (modelData.value && modelData.value.providers) || []
    if (list.length) {
      const cur = form[tabKey]?.[f.name]
      if (cur && !list.includes(cur)) return [...list, cur] // 自定义服务商兜底显示（如用户手填的网关名）
      return list
    }
    return f.options || []
  }
  return f.options || []
}
// 通用联动：f.linkField 存在时（如 provider→baseURL），用 providerBaseURLs 填充目标字段
// （目标字段为空 或 等于旧服务商默认端点时自动填充，用户自定义过的值不动）
function onSelectChange(f) {
  if (!form['ai']) return
  const ai = form['ai']
  // linkFields 数组（多字段联动，如 provider → baseURL + apiKey）；兼容旧单 linkField
  const fields = f.linkFields || (f.linkField ? [f.linkField] : [])
  if (!fields.length) return
  const md = modelData.value || {}
  const urls = md.providerBaseURLs || {}
  const keys = md.providerKeys || {} // 服务商独立 API Key
  const newP = ai.provider
  const oldDefault = urls[lastProvider]
  for (const name of fields) {
    if (name === 'apiKey') {
      // 服务商密钥：始终带出该服务商保存的 key（用户可改，保存设置时写回）
      ai[name] = keys[newP] || ''
    } else {
      const cur = ai[name]
      if (cur === undefined || cur === '' || (oldDefault && cur === oldDefault)) {
        ai[name] = urls[newP] || ''
      }
    }
  }
  lastProvider = newP
}

// ─── 值模型：binding → 顶层 AppSettings；非 binding → 插件命名空间 ───
const form = reactive({})       // form[tabKey][fieldName] = 值
const projectInst = ref('')     // 项目级指令（type=project 字段）

function zeroValue(type) {
  switch (type) {
    case 'checkbox': return false
    case 'number': return 0
    case 'tags': return []
    default: return ''
  }
}

function buildForm() {
  for (const key of Object.keys(form)) delete form[key]
  const top = state.settings || {}
  lastProvider = top.provider || '' // ★ 联动基准：当前服务商（切走时 baseURL 判断是否覆盖旧默认）
  const pvals = (top.pluginSettings || {})
  for (const s of (state.pluginSchemas || [])) {
    form[s.key] = {}
    for (const f of (s.fields || [])) {
      let v
      if (f.type === 'project' || f.type === 'provider-manager' || f.type === 'model-params-manager' || f.type === 'preset-manager') { continue }
      if (f.binding) {
        v = top[f.binding] !== undefined ? top[f.binding] : f.default
      } else {
        const cur = pvals[s.key] || {}
        v = cur[f.name] !== undefined ? cur[f.name] : f.default
      }
      if (v === undefined) v = zeroValue(f.type)
      // 类型规整
      if (f.type === 'checkbox') v = !!v
      if (f.type === 'number') v = typeof v === 'number' ? v : Number(v) || 0
      if (f.type === 'tags') v = Array.isArray(v) ? v : []
      form[s.key][f.name] = v
    }
  }
  // 平台特殊字段：项目级指令
  const hasProject = (state.pluginSchemas || []).some(s => (s.fields || []).some(f => f.type === 'project'))
  projectInst.value = ''
  if (hasProject) loadProjectInstructions()
}

// tags 显示/输入
function tagsText(tabKey, f) {
  const v = form[tabKey]?.[f.name]
  return Array.isArray(v) ? v.join(', ') : (v || '')
}
function onTagsInput(tabKey, f, ev) {
  form[tabKey][f.name] = ev.target.value.split(',').map(s => s.trim()).filter(Boolean)
}

async function loadProjectInstructions() {
  try {
    const proj = await api.getInstructions('project')
    projectInst.value = proj.content || ''
  } catch {}
}

// ─── 加载 ───
function loadSettings() {
  buildForm()
  if (state.settings?.theme) applyTheme(state.settings.theme)
}

async function reloadProjectInst() { await loadProjectInstructions() }

const resetForm = () => { loadSettings() }

// ─── AI 配置预设变更后：重新拉 settings（应用预设已整套写回）并重建表单 ───
async function onPresetSaved() {
  try {
    const r = await api.apiGet('/settings')
    if (r && r.settings) {
      state.settings = r.settings
      await loadModels()
      loadSettings()
    }
  } catch {}
}

// ─── 保存：分拣 binding → 顶层 / 非 binding → 插件命名空间 ───
const saveSettings = async () => {
  try {
    // ★ 修复：PUT 前先拉后端最新 settings 作基底（state.settings 是启动时快照+本地增量，
    //   AI 配置「应用」/对话面板切模型/服务商面板等已把新值写入后端——用过期缓存整体
    //   PUT 会把这些新值覆盖回旧值，导致「配置永远不会修改」）
    let base = {}
    try {
      const latest = await api.apiGet('/settings')
      base = (latest && latest.settings) || {}
    } catch {}
    const top = { ...base }
    const pluginOut = { ...((base.pluginSettings) || {}) }
    let themeChanged = false
    for (const s of (state.pluginSchemas || [])) {
      const vals = form[s.key] || {}
      for (const f of (s.fields || [])) {
        if (f.type === 'project') {
          await api.saveInstructions('project', projectInst.value)
          continue
        }
        if (f.type === 'provider-manager' || f.type === 'model-params-manager' || f.type === 'preset-manager') {
          // 服务商/模型参数/AI 配置预设维护走独立面板（各自内部保存），不并入通用表单保存
          continue
        }
        const v = vals[f.name]
        if (f.binding) {
          if (f.name === 'theme' && v !== top[f.binding]) themeChanged = true
          top[f.binding] = v
        } else {
          if (!pluginOut[s.key]) pluginOut[s.key] = {}
          pluginOut[s.key][f.name] = v
        }
      }
    }
    await api.apiPut('/settings', { settings: top, pluginSettings: pluginOut })
    state.settings = top
    if (themeChanged) applyTheme(top.theme)
    window.$toast('设置已保存', 'success')
    emit('close')
  } catch (err) {
    window.$toast('保存失败: ' + err.message, 'error')
  }
}

onMounted(async () => {
  await loadModels()
  loadSettings()
})
</script>

<style scoped>
.modal-overlay {
  position: fixed; top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex; align-items: center; justify-content: center; z-index: 1000;
}
.modal-content {
  background: var(--bg-secondary, #1e1e2e);
  border: 1px solid var(--border-color, #333);
  border-radius: 10px;
  /* ★ 2026-09-01 固定高宽：切换 tab 不再来回改变面板尺寸（内容在内部滚动） */
  width: min(94vw, 880px);
  height: min(86vh, 720px);
  display: flex; flex-direction: column;
  box-shadow: 0 12px 40px rgba(0,0,0,.4);
  overflow: hidden;
}
h2 {
  display: flex; align-items: center; gap: 8px;
  margin: 0; padding: 14px 18px;
  font-size: 15px; font-weight: 600;
  border-bottom: 1px solid var(--border-color, #333);
  color: var(--text-primary, #eee);
}
.modal-close {
  margin-left: auto; background: none; border: none;
  color: var(--text-secondary, #999); font-size: 18px; cursor: pointer;
  width: 28px; height: 28px; border-radius: 6px; line-height: 1;
}
.modal-close:hover { background: var(--bg-hover, rgba(255,255,255,.08)); color: #fff; }
.modal-body { display: flex; flex: 1; min-height: 0; }
.settings-tabs {
  display: flex; flex-direction: column; gap: 3px;
  padding: 12px 10px; width: 184px; flex-shrink: 0;
  border-right: 1px solid var(--border-color, #333);
  background: var(--bg-tertiary, rgba(0,0,0,.15));
  overflow-y: auto;
}
.settings-tabs-filter-wrap { margin-bottom: 6px; }
.settings-tabs-filter {
  width: 100%; box-sizing: border-box;
  background: var(--input-bg, #14141f);
  border: 1px solid var(--border-color, #3a3a4a);
  color: var(--text-primary, #eee);
  border-radius: 6px; padding: 5px 9px; font-size: 12px;
  outline: none; transition: border-color .15s;
}
.settings-tabs-filter:focus { border-color: var(--accent, #4f8cff); }
.settings-tabs-none { color: var(--text-secondary, #888); font-size: 12px; text-align: center; padding: 12px 4px; }
.settings-tab {
  text-align: left; padding: 9px 12px; border: none; border-radius: 7px;
  background: none; color: var(--text-secondary, #aaa);
  font-size: 13px; line-height: 1.3; min-height: 18px;
  cursor: pointer; transition: all .15s;
  word-break: break-word;
}
.settings-tab:hover { background: var(--bg-hover, rgba(255,255,255,.06)); color: var(--text-primary, #eee); }
.settings-tab.active {
  background: var(--accent, #4f8cff); color: #fff; font-weight: 600;
}
.settings-content { flex: 1; padding: 16px 18px; overflow-y: auto; }
.settings-empty { color: var(--text-secondary, #888); text-align: center; padding: 40px 0; font-size: 13px; }

.setting-group { margin-bottom: 14px; }
.group-title {
  font-size: 12px; font-weight: 600; letter-spacing: .4px;
  color: var(--text-secondary, #999); margin-bottom: 8px;
  text-transform: uppercase; opacity: .85;
}
.setting-row {
  display: flex; flex-direction: column; gap: 5px;
  padding: 8px 10px; margin-bottom: 6px; border-radius: 7px;
  transition: background .12s;
}
.setting-row:hover { background: var(--bg-hover, rgba(255,255,255,.04)); }
.setting-row.row-toggle { flex-direction: row; align-items: center; justify-content: space-between; }
.field-label {
  font-size: 13px; color: var(--text-primary, #ddd); font-weight: 500;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.field-control { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.setting-row .field-input,
.setting-row input[type="text"],
.setting-row input[type="password"],
.setting-row input[type="number"],
.setting-row .field-select {
  width: 100%; box-sizing: border-box;
  background: var(--input-bg, #14141f);
  border: 1px solid var(--border-color, #3a3a4a);
  color: var(--text-primary, #eee);
  border-radius: 6px; padding: 6px 10px; font-size: 13px;
  outline: none; transition: border-color .15s;
}
.setting-row input:focus,
.setting-row select:focus { border-color: var(--accent, #4f8cff); }
.setting-hint {
  font-size: 11px; color: var(--text-secondary, #888);
  line-height: 1.45; min-width: 0;
}
.field-textarea {
  width: 100%; box-sizing: border-box; resize: vertical;
  background: var(--input-bg, #14141f);
  border: 1px solid var(--border-color, #3a3a4a);
  color: var(--text-primary, #eee); border-radius: 6px;
  padding: 8px 10px; font-size: 13px; font-family: inherit; outline: none;
}
.field-textarea:focus { border-color: var(--accent, #4f8cff); }

/* 开关（对齐 pp-switch 风格） */
.pp-switch { position: relative; display: inline-flex; align-items: center; cursor: pointer; flex-shrink: 0; }
.pp-switch input { position: absolute; opacity: 0; width: 0; height: 0; }
.pp-switch-track {
  width: 34px; height: 18px; border-radius: 9px;
  background: var(--border-color, #444); position: relative; transition: background .18s;
}
.pp-switch-track::after {
  content: ''; position: absolute; top: 2px; left: 2px;
  width: 14px; height: 14px; border-radius: 50%;
  background: #fff; transition: transform .18s;
}
.pp-switch input:checked + .pp-switch-track { background: var(--accent, #4f8cff); }
.pp-switch input:checked + .pp-switch-track::after { transform: translateX(16px); }

/* slider */
.slider-row { flex: 1; display: flex; align-items: center; gap: 10px; min-width: 0; }
.slider-row input[type="range"] { flex: 1; accent-color: var(--accent, #4f8cff); }
.slider-val {
  min-width: 36px; text-align: right;
  font-size: 12px; color: var(--text-primary, #eee); font-variant-numeric: tabular-nums;
}

/* color */
.color-row { flex: 1; display: flex; align-items: center; gap: 8px; min-width: 0; }
.color-row input[type="color"] {
  width: 34px; height: 24px; border: 1px solid var(--border-color, #3a3a4a);
  border-radius: 5px; background: none; padding: 1px; cursor: pointer;
}
.color-code { font-size: 12px; color: var(--text-secondary, #aaa); }

.modal-footer {
  display: flex; justify-content: flex-end; gap: 10px;
  padding: 12px 18px; border-top: 1px solid var(--border-color, #333);
}
.btn-secondary, .btn-primary {
  padding: 7px 16px; border-radius: 7px; font-size: 13px;
  cursor: pointer; border: 1px solid var(--border-color, #444); transition: all .15s;
}
.btn-secondary { background: none; color: var(--text-primary, #ddd); }
.btn-secondary:hover { background: var(--bg-hover, rgba(255,255,255,.06)); }
.btn-primary {
  background: var(--accent, #4f8cff); color: #fff; border-color: var(--accent, #4f8cff); font-weight: 600;
}
.btn-primary:hover { filter: brightness(1.12); }
</style>
