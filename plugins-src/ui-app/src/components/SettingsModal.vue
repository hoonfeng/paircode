<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content">
      <!-- ★ 2026-09-24 对齐设计稿 appearance-theme th782（头部 = 标题 + 副标题两行）：
           副标题显示**当前分类名**（真实数据，非装饰文案）。 -->
      <div class="modal-head">
        <div class="mh-text">
          <div class="mh-title"><SvgIcon name="settings" :size="16" /> 设置</div>
          <div class="mh-sub">{{ (tabs.find(t => t.key === activeTab) || {}).title || '所有分类' }}</div>
        </div>
        <button class="modal-close" @click="$emit('close')">×</button>
      </div>
      <div class="modal-body">
        <!-- ═══ 纯 schema 驱动：所有配置 tab 由插件 ctx.registerSettings 注册 ═══ -->
        <div v-if="tabs.length" class="settings-tabs">
          <!-- 设计稿 th783：分组标题「设置分类」 -->
          <div class="settings-tabs-title">设置分类</div>
          <div v-if="tabs.length > 6" class="settings-tabs-filter-wrap">
            <input v-model="tabQuery" class="settings-tabs-filter" type="text" placeholder="筛选设置…" />
          </div>
          <!-- 设计稿 th755-th773：项 32 高 / radius 8 / 前置图标（选中态=check） -->
          <button v-for="t in filteredTabs" :key="t.key" :class="['settings-tab', { active: activeTab === t.key }]"
                  @click="activeTab = t.key">
            <SvgIcon :name="activeTab === t.key ? 'check' : 'chevron-right'" :size="14" />
            <span class="st-label">{{ t.title }}</span>
          </button>
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

                      <!-- theme-gallery（主题画廊：缩略色卡选择）
                           ★ 值仍是 string 主题 id，binding 直连 AppSettings.theme —— 与 select 同数据模型，
                             仅换一种更直观的呈现（色卡 + 明暗标识）。无联动字段故不调 onSelectChange。 -->
                      <div v-else-if="f.type === 'theme-gallery'" class="theme-gallery">
                        <button v-for="it in galleryItems(f)" :key="it.value" type="button"
                                class="tg-item" :class="{ active: form[tab.key][f.name] === it.value }"
                                :title="it.label" @click="form[tab.key][f.name] = it.value">
                          <!-- 设计稿 th584：预览区 48 高（主题代表色横向平铺） -->
                          <span class="tg-preview">
                            <i v-for="(c, ci) in it.colors" :key="ci" :style="{ background: c }"></i>
                          </span>
                          <span class="tg-text">
                            <span class="tg-label">{{ it.label }}</span>
                            <!-- ★ 2026-09-25：定位文案（设计稿每张卡 = 预览 + 名称 + 一句定位，
                                 如「默认暗色，久看不累」）。由插件 swatches[].desc 提供。 -->
                            <span v-if="it.desc" class="tg-desc">{{ it.desc }}</span>
                          </span>
                        </button>
                      </div>

                      <!-- color-dots（色点选择：强调色覆盖等 —— 圆点 + 选中描边，悬停看名称）
                           ★ 值仍是 string，与 select / theme-gallery 同数据模型，仅换呈现形态。 -->
                      <div v-else-if="f.type === 'color-dots'" class="color-dots">
                        <button v-for="it in galleryItems(f)" :key="it.value" type="button"
                                class="cd-item" :class="{ active: String(form[tab.key][f.name] || '') === it.value }"
                                :title="it.label" @click="form[tab.key][f.name] = it.value">
                          <i class="cd-dot" :style="{ background: (it.colors && it.colors[0]) || 'transparent' }"></i>
                        </button>
                      </div>

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
                      <ProviderManager v-else-if="f.type === 'provider-manager'" :model-param-fields="f.modelParamFields || []" :model-editor="f.modelEditor || {}"
                                       :protocol-label="f.protocolLabel || 'LLM 协议'" :protocol-options="f.protocolOptions || []" :protocol-hint="f.protocolHint || ''"
                                       @saved="onProvidersSaved" />

                      <!-- preset-manager（AI 配置预设面板：CRUD /api/ai-presets，独立保存，不参与普通表单） -->
                      <PresetManager v-else-if="f.type === 'preset-manager'" :key="presetRev" :preset-fields="f.presetFields || []" @saved="onPresetSaved" />
                      
                      <!-- link（纯展示：href 经 linkHref 白名单校验；文本用 {{ }} 插值防注入） -->
                      <a v-else-if="f.type === 'link'" class="field-link" :href="linkHref(f)" target="_blank" rel="noopener noreferrer">{{ f.linkText || '打开' }}</a>

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
      <!-- ★ 2026-09-24 对齐设计稿 th790（底 48：说明 + 信息）：左侧补说明文本，
           与「撤销 / 保存设置」按钮同行（设计稿未画按钮，但 schema 驱动需显式保存）。 -->
      <div class="modal-footer">
        <span class="mf-note">配置项由插件注册，修改后点「保存设置」生效</span>
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

// galleryItems：主题画廊条目。优先用插件给的 swatches（含缩略色卡色值），
// 缺失时回退为 options 的纯文本条目 —— 老插件/未升级的 schema 也能正常渲染。
function galleryItems(f) {
  if (Array.isArray(f.swatches) && f.swatches.length) return f.swatches
  return (f.options || []).map(o => ({ value: o, label: o, colors: [] }))
}

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
      if (f.type === 'project' || f.type === 'link' || f.type === 'provider-manager' || f.type === 'model-params-manager' || f.type === 'preset-manager') { continue }
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
// link 字段：href 安全化——仅允许 http/https 与同源相对路径；其余协议
// （javascript:/data:/vbscript: 等）一律拒绝返回 '#'（防 XSS/开放重定向）。
function linkHref(f) {
  const href = String(f.href || f.default || '').trim()
  if (!href) return '#'
  if (/^https?:\/\//i.test(href)) return href
  if (/^[a-z][a-z0-9+.-]*:/i.test(href)) return '#'
  return location.origin + (href.startsWith('/') ? href : '/' + href)
}

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

// ─── 服务商数据变更后（新增/编辑/改名/删除）：重载服务商数据 + 重建 AI 配置面板 ───
// ★ 2026-09-20：服务商改名会同步改写 ai-presets.json 里的 provider 引用，预设面板必须重新
//   拉取才会显示新名（:key 变化 → PresetManager 重建并重新 getAiPresets）。
const presetRev = ref(0)
async function onProvidersSaved() {
  await loadModels()
  presetRev.value++
}

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
        if (f.type === 'link' || f.type === 'provider-manager' || f.type === 'model-params-manager' || f.type === 'preset-manager') {
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

// ★ 同步构建：setup 阶段（首次渲染前）就绪，字段挂载时 form[tabKey] 必然存在，
//   避免 onMounted 中 await loadModels() 前的首次渲染对 form[tabKey][f.name] 读 undefined 报错
buildForm()

onMounted(async () => {
  await loadModels()
  loadSettings()
})
</script>

<style scoped>
.modal-overlay {
  position: fixed; top: 0; left: 0; right: 0; bottom: 0;
  background: var(--color-scrim);
  display: flex; align-items: center; justify-content: center; z-index: 1000;
}
.modal-content {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  /* ★ 2026-09-01 固定高宽：切换 tab 不再来回改变面板尺寸（内容在内部滚动） */
  width: min(94vw, 880px);
  height: min(86vh, 720px);
  display: flex; flex-direction: column;
  box-shadow: var(--shadow-lg);
  overflow: hidden;
}
/* 设计稿 th782：头部左「标题 + 副标题」两行、右侧操作区（原 h2 单行 + × 按钮） */
.modal-head {
  display: flex; align-items: center; gap: 12px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border-color);
}
.mh-text { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.mh-title {
  display: flex; align-items: center; gap: 8px;
  font-size: 15px; font-weight: 600; color: var(--text-primary);
}
.mh-sub { font-size: 11px; color: var(--text-muted); }
.modal-close {
  margin-left: auto; background: none; border: none;
  color: var(--text-secondary); font-size: 18px; cursor: pointer;
  width: 28px; height: 28px; border-radius: 6px; line-height: 1;
}
.modal-close:hover { background: var(--bg-hover); color: var(--color-fg); }
.modal-body { display: flex; flex: 1; min-height: 0; }
.settings-tabs {
  display: flex; flex-direction: column; gap: 4px;
  padding: 12px 10px; width: 184px; flex-shrink: 0;
  border-right: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  overflow-y: auto;
}
/* 设计稿 th783：分组标题 xs(11)/muted/600 */
.settings-tabs-title {
  font-size: 11px; font-weight: 600; color: var(--text-muted);
  padding: 2px 6px 4px;
}
.settings-tabs-filter-wrap { margin-bottom: 6px; }
.settings-tabs-filter {
  width: 100%; box-sizing: border-box;
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  border-radius: 6px; padding: 5px 9px; font-size: 12px;
  outline: none; transition: border-color .15s;
}
.settings-tabs-filter:focus { border-color: var(--accent); }
.settings-tabs-none { color: var(--text-secondary); font-size: 12px; text-align: center; padding: 12px 4px; }
/* 设计稿 th755-th773：分类项 32 高、radius 8、pad sm(8)、gap sm(8)、前置图标 14；
   选中态 = surface-3 底(→ --bg-active) + accent 文字（**不是** accent 实底），
   图标随之由 chevron-right 换成 check（th762）。 */
.settings-tab {
  display: flex; align-items: center; gap: 8px;
  height: 32px; padding: 0 8px; border: none; border-radius: 8px;
  background: none; color: var(--text-primary);
  font-size: 12px; text-align: left; cursor: pointer;
  transition: background .15s, color .15s;
}
.settings-tab .svg-icon { flex-shrink: 0; color: var(--text-muted); }
.st-label { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.settings-tab:hover { background: var(--bg-hover); }
.settings-tab.active { background: var(--bg-active); color: var(--accent); font-weight: 600; }
.settings-tab.active .svg-icon { color: var(--accent); }
.settings-content { flex: 1; padding: 16px 18px; overflow-y: auto; }
.settings-empty { color: var(--text-secondary); text-align: center; padding: 40px 0; font-size: 13px; }

.setting-group { margin-bottom: 14px; }
.group-title {
  font-size: 12px; font-weight: 600; letter-spacing: .4px;
  color: var(--text-secondary); margin-bottom: 8px;
  text-transform: uppercase; opacity: .85;
}
.setting-row {
  display: flex; flex-direction: column; gap: 5px;
  padding: 8px 10px; margin-bottom: 6px; border-radius: 7px;
  transition: background .12s;
}
.setting-row:hover { background: var(--bg-hover); }
.setting-row.row-toggle { flex-direction: row; align-items: center; justify-content: space-between; }
.field-label {
  font-size: 13px; color: var(--text-primary); font-weight: 500;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.field-control { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.setting-row .field-input,
.setting-row input[type="text"],
.setting-row input[type="password"],
.setting-row input[type="number"],
.setting-row .field-select {
  width: 100%; box-sizing: border-box;
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  border-radius: 6px; padding: 6px 10px; font-size: 13px;
  outline: none; transition: border-color .15s;
}
.setting-row input:focus,
.setting-row select:focus { border-color: var(--accent); }
.setting-hint {
  font-size: 11px; color: var(--text-secondary);
  line-height: 1.45; min-width: 0;
}
.field-link {
  display: inline-flex; align-items: center; height: 28px; padding: 0 10px;
  font-size: 12px; color: var(--accent, #4f8cff); text-decoration: none;
  border: 1px solid rgba(79, 140, 255, .35); border-radius: 6px;
  background: rgba(79, 140, 255, .08); transition: background .15s;
}
.field-link:hover { background: rgba(79, 140, 255, .16); }
.field-textarea {
  width: 100%; box-sizing: border-box; resize: vertical;
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary); border-radius: 6px;
  padding: 8px 10px; font-size: 13px; font-family: inherit; outline: none;
}
.field-textarea:focus { border-color: var(--accent); }

/* 开关（对齐 pp-switch 风格） */
.pp-switch { position: relative; display: inline-flex; align-items: center; cursor: pointer; flex-shrink: 0; }
.pp-switch input { position: absolute; opacity: 0; width: 0; height: 0; }
.pp-switch-track {
  width: 34px; height: 18px; border-radius: 9px;
  background: var(--border-color); position: relative; transition: background .18s;
}
.pp-switch-track::after {
  content: ''; position: absolute; top: 2px; left: 2px;
  width: 14px; height: 14px; border-radius: 50%;
  background: #fff; transition: transform .18s;
}
.pp-switch input:checked + .pp-switch-track { background: var(--accent); }
.pp-switch input:checked + .pp-switch-track::after { transform: translateX(16px); }

/* slider */
.slider-row { flex: 1; display: flex; align-items: center; gap: 10px; min-width: 0; }
.slider-row input[type="range"] { flex: 1; accent-color: var(--accent); }
.slider-val {
  min-width: 36px; text-align: right;
  font-size: 12px; color: var(--text-primary); font-variant-numeric: tabular-nums;
}

/* color */
.color-row { flex: 1; display: flex; align-items: center; gap: 8px; min-width: 0; }
.color-row input[type="color"] {
  width: 34px; height: 24px; border: 1px solid var(--border-color);
  border-radius: 5px; background: none; padding: 1px; cursor: pointer;
}
.color-code { font-size: 12px; color: var(--text-secondary); }

/* theme-gallery（主题画廊：缩略色卡）—— 颜色全部走设计令牌，跟随当前主题 */
/* 设计稿 th589/th608/th627：主题卡 224×128、radius 12、pad sm(8)、gap sm(8)；
   预览区 48 高（主题代表色平铺）+ 名称(sm/600) + 定位文案(xs/muted)；
   选中卡以 accent 描边（设计稿 border=$color.accent）。 */
.theme-gallery {
  flex: 1; min-width: 0;
  display: grid; gap: 12px;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
}
.tg-item {
  display: flex; flex-direction: column; gap: 8px; padding: 8px;
  min-height: 118px; text-align: left;
  background: var(--bg-secondary); border: 1px solid var(--border-color);
  border-radius: 12px; cursor: pointer; transition: border-color .15s, background .15s;
}
.tg-item:hover { background: var(--bg-hover); border-color: var(--text-muted); }
.tg-item.active { border-color: var(--accent); }
.tg-preview { display: flex; height: 48px; border-radius: 8px; overflow: hidden; flex: none; }
.tg-preview i { display: block; flex: 1 1 0; min-width: 0; }
.tg-text { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.tg-label { font-size: 12px; font-weight: 600; color: var(--text-primary);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tg-desc { font-size: 11px; color: var(--text-muted); line-height: 1.5; }

/* color-dots（色点选择） */
.color-dots { flex: 1; display: flex; align-items: center; gap: 8px; min-width: 0; }
.cd-item {
  display: flex; padding: 3px; background: none;
  border: 2px solid transparent; border-radius: 999px; cursor: pointer; transition: all .15s;
}
.cd-item:hover { border-color: var(--text-muted); }
.cd-item.active { border-color: var(--accent); }
.cd-dot {
  display: block; width: 18px; height: 18px; border-radius: 999px;
  border: 1px solid var(--border-color);
}

/* 设计稿 th790：底部 48 高（pad md=12），左「说明文本」+ 右「操作按钮」同行 */
.modal-footer {
  display: flex; align-items: center; justify-content: flex-end; gap: 10px;
  padding: 12px 18px; border-top: 1px solid var(--border-color);
}
.mf-note { margin-right: auto; font-size: 11px; color: var(--text-muted); }
.btn-secondary, .btn-primary {
  padding: 7px 16px; border-radius: 7px; font-size: 13px;
  cursor: pointer; border: 1px solid var(--border-color); transition: all .15s;
}
.btn-secondary { background: none; color: var(--text-primary); }
.btn-secondary:hover { background: var(--bg-hover); }
.btn-primary {
  background: var(--accent); color: var(--color-accent-fg); border-color: var(--accent); font-weight: 600;
}
.btn-primary:hover { filter: brightness(1.12); }
</style>
