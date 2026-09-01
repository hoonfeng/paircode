<template>
  <div class="pm-manager">
    <!-- 工具栏 -->
    <div class="mgm-toolbar">
      <span class="mgm-count">{{ names.length }} 个配置</span>
      <button class="mgm-btn mgm-primary" @click="openAdd" title="添加一条 AI 配置（服务商 + API Key）">＋ 添加新配置</button>
    </div>

    <!-- 添加 / 编辑表单（点击添加/编辑才弹出） -->
    <div v-if="showForm" class="mgm-edit">
      <div class="mgm-edit-title">{{ editingName ? '编辑配置：' + editingName : '添加新配置' }}</div>
      <div class="mgm-field">
        <span class="mgm-field-label">配置名称</span>
        <input v-model="form.name" type="text" placeholder="如：主力 / 写作备用…" @keydown.enter="confirmSave" />
      </div>
      <div v-for="f in presetFields" :key="f.name" class="mgm-field">
        <span class="mgm-field-label">{{ f.label }}<span v-if="f.required" class="mgm-required">*</span></span>
        <!-- select 类型（服务商选择） -->
        <select v-if="f.type === 'select'" v-model="form[f.name]" class="mgm-select" @change="onFieldChange(f)">
          <option v-for="opt in fieldOptions(f)" :key="opt" :value="opt">{{ opt }}</option>
        </select>
        <!-- password 类型（API Key） -->
        <input v-else-if="f.type === 'password'" v-model="form[f.name]" type="password" :placeholder="f.placeholder || ''" />
        <!-- text 兜底 -->
        <input v-else v-model="form[f.name]" type="text" :placeholder="f.placeholder || ''" />
        <span v-if="f.hint" class="mgm-field-hint">{{ f.hint }}</span>
      </div>
      <div class="mgm-edit-actions">
        <button class="mgm-btn mgm-primary" :disabled="saving" @click="confirmSave">
          {{ saving ? '保存中…' : '保存配置' }}
        </button>
        <button class="mgm-btn" @click="closeForm">取消</button>
      </div>
    </div>

    <!-- 配置卡片列表（主视图） -->
    <div v-if="names.length" class="mgm-cards">
      <div v-for="n in names" :key="n" class="mgm-card" :class="{ 'pm-active': n === activeName }">
        <div class="mgm-card-head">
          <span class="mgm-name" :title="n">{{ n }}<span v-if="n === activeName" class="pm-active-badge">使用中</span></span>
          <div class="mgm-ops">
            <button class="mgm-btn mgm-small" :disabled="applying === n" @click="applyPreset(n)">
              {{ applying === n ? '应用中…' : '应用' }}
            </button>
            <button class="mgm-btn mgm-small" @click="openEdit(n)">编辑</button>
            <button class="mgm-btn mgm-small mgm-danger" @click="removePreset(n)">删除</button>
          </div>
        </div>
        <div class="pm-preview">
          <div class="pm-snap-row"><span>服务商</span><b>{{ (presets[n] || {}).provider || '—' }}</b></div>
          <div class="pm-snap-row"><span>API Key</span><b>{{ (presets[n] || {}).apiKey ? '已配置' : '未配置' }}</b></div>
        </div>
      </div>
    </div>
    <div v-else-if="!showForm" class="mgm-empty">
      <div class="mgm-empty-box">
        <div class="mgm-empty-title">还没有 AI 配置</div>
        <div class="mgm-empty-sub">添加一条服务商 + API Key，保存后即可在对话面板中选模型。</div>
        <button class="mgm-btn mgm-primary" @click="openAdd">＋ 添加新配置</button>
      </div>
    </div>

    <div v-if="error" class="mgm-error">{{ error }}</div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import api from '../api.js'

// ★ AI 配置列表（2026-09-01 改变模式）：AI tab 主视图 = 已添加的配置列表；
//   点「＋ 添加新配置」弹出表单设置 名称/服务商/Key 后保存；
//   「应用」= 整套配置写回 settings（preset 名），装配时按 preset 展开。Key 随配置存（ai-presets.json）。
//   数据经 /api/ai-presets（config/ai-presets.json），每条 = 完整 AI 配置快照（服务商 + Key）。

// ★ 2026-09-01 schema 驱动：presetFields 由插件（agentloop/index.js）注册，
//   前端按此动态渲染表单字段（新增/修改字段只改插件，不碰前端组件）。
const props = defineProps({
  presetFields: { type: Array, default: () => [] },
})
const emit = defineEmits(['saved'])
// { 配置名: AiPreset } 配置快照（provider/baseURL/apiKey/composer 模型等），load() 从 /api/ai-presets 填充
const presets = ref({})
const names = computed(() => Object.keys(presets.value || {}))
const activeName = ref('')     // settings.preset（当前使用中的配置名）
const showForm = ref(false)
const editingName = ref('')    // '' = 添加；非空 = 编辑该配置
const saving = ref(false)
const applying = ref('')
const error = ref('')

// 服务商数据（/api/models）
const modelData = ref(null)
const providers = computed(() => (modelData.value && modelData.value.providers) || [])
// form 初始：配置名（内置）+ presetFields 各字段（schema 驱动）
const form = ref({ name: '', provider: '', baseURL: '', apiKey: '' })

// 按 presetFields 构建空白表单字段（新增字段自动出现）
function blankForm(extra = {}) {
  const base = { name: '', provider: '', baseURL: '', apiKey: '' }
  for (const f of props.presetFields) {
    if (!(f.name in base)) base[f.name] = ''
  }
  return Object.assign(base, extra)
}

function showError(msg) { error.value = msg; setTimeout(() => { if (error.value === msg) error.value = '' }, 4000) }

async function load() {
  try {
    const [p, st, md] = await Promise.all([
      api.getAiPresets().catch(() => ({ presets: {} })),
      api.apiGet('/settings').catch(() => ({ settings: {} })),
      api.getModels().catch(() => null),
    ])
    presets.value = (p && p.presets) || {}
    activeName.value = (st && st.settings && st.settings.preset) || ''
    modelData.value = md
  } catch (e) {
    showError('加载失败: ' + (e.message || e))
  }
}

function providerInfo(prov) {
  const md = modelData.value || {}
  return {
    baseURL: (md.providerBaseURLs && md.providerBaseURLs[prov]) || '',
    models: ((md.models && md.models[prov]) || []),
  }
}

// 打开添加表单（服务商默认当前生效的，带出 BaseURL/Key/模型）
// ★ 优先带出激活预设（settings.preset → ai-presets 整套配置），
//   其次 settings 当前生效值，最后 models.json 服务商默认。
function openAdd() {
  const s = (window && window.__PAIRCODE_CORE && window.__PAIRCODE_CORE.uiState && window.__PAIRCODE_CORE.uiState.state
    && window.__PAIRCODE_CORE.uiState.state.settings) || {}
  let base = {}
  if (s.preset && presets.value && presets.value[s.preset]) base = presets.value[s.preset]
  const prov = (base.provider || s.provider || providers.value[0] || '')
  const info = providerInfo(prov)
  form.value = blankForm({
    provider: prov,
    baseURL: (base.baseURL || s.baseURL || info.baseURL || ''),
    apiKey: base.apiKey || '',
  })
  editingName.value = ''
  showForm.value = true
}

// 打开编辑表单（预填配置快照）
function openEdit(name) {
  const p = (presets.value && presets.value[name]) || {}
  form.value = blankForm({
    name,
    provider: p.provider || '',
    baseURL: p.baseURL || '',
    apiKey: p.apiKey || '',
  })
  editingName.value = name
  showForm.value = true
}

function closeForm() { showForm.value = false; editingName.value = '' }

// 切换服务商 → 带出该服务商 BaseURL/Key + 模型列表（模型保留当前值若在新列表中，否则默认第一个）
function fieldOptions(f) {
  // select 类型的选项来源
  if (f.source === 'providers') return providers.value || []
  return f.options || []
}

function onFieldChange(f) {
  // 服务商切换 → 自动带出 BaseURL
  if (f.name === 'provider' && form.value.provider) {
    const info = providerInfo(form.value.provider)
    form.value.baseURL = info.baseURL || ''
  }
}

// ★ 联动双保险：除 @change 外再挂 watch（ov 非空 = 用户在表单内切换服务商，
//   跳过 openAdd/openEdit 初始化赋值；兼容 wb-ui 引擎 select change 事件缺失场景）
watch(() => form.value.provider, (nv, ov) => {
  if (!showForm.value || ov === '') return
  if (nv !== ov) onFieldChange({ name: 'provider' })
})

async function confirmSave() {
  const name = form.value.name.trim()
  if (!name) { showError('请输入配置名称'); return }
  const prov = form.value.provider || ''
  if (!prov) { showError('请选择服务商'); return }
  // 校验 presetFields 中标记 required 的字段
  for (const f of props.presetFields) {
    if (f.required && !form.value[f.name]) {
      showError('请填写' + f.label)
      return
    }
  }
  const info = providerInfo(prov)
  if (!form.value.baseURL) form.value.baseURL = info.baseURL || ''
  saving.value = true
  error.value = ''
  try {
  const preset = {
      provider: prov,
      baseURL: form.value.baseURL,
      apiKey: form.value.apiKey,
    }
    // ★ 附加 presetFields 中注册的其他字段（schema 驱动，自动透传）
    for (const f of props.presetFields) {
      if (f.name !== 'provider' && f.name !== 'apiKey') preset[f.name] = form.value[f.name]
    }
    if (editingName.value && editingName.value !== name) {
      // 改名：PUT 全量（新名入库 + 旧名删除），并同步 settings.preset
      const map = { ...(presets.value || {}) }
      map[name] = preset
      delete map[editingName.value]
      const r = await api.saveAiPresets(map)
      if (!(r && r.ok)) { showError((r && r.error) || '保存失败'); return }
      presets.value = map
      if (activeName.value === editingName.value) {
        activeName.value = name
        // ★ 修复：不再用 UI 缓存 settings 整体 PUT（过期缓存会覆盖后端新值）。
        //   只提交变更字段 preset——后端 applyTopSettings 为合并语义，其余字段保留后端现值。
        await api.apiPut('/settings', { settings: { preset: name }, pluginSettings: {} }).catch(() => {})
      }
    } else {
      const r = await api.saveAiPreset('save', name, preset)
      if (!(r && r.ok)) { showError((r && r.error) || '保存失败'); return }
      presets.value = r.presets || presets.value
    }
    closeForm()
    emit('saved')
  } catch (e) {
    showError('保存失败: ' + (e.message || e))
  } finally {
    saving.value = false
  }
}

async function applyPreset(name) {
  applying.value = name
  error.value = ''
  try {
    const r = await api.saveAiPreset('apply', name)
    if (r && r.ok) {
      activeName.value = name
      emit('saved')   // 通知父级刷新（settings 已整套写回）
    } else {
      showError((r && r.error) || '应用失败')
    }
  } catch (e) {
    showError('应用失败: ' + (e.message || e))
  } finally {
    applying.value = ''
  }
}

async function removePreset(name) {
  if (!confirm('删除配置「' + name + '」？')) return
  error.value = ''
  try {
    const r = await api.saveAiPreset('delete', name)
    if (r && r.ok) {
      presets.value = r.presets || presets.value
      if (activeName.value === name) activeName.value = ''
      emit('saved')
    } else {
      showError((r && r.error) || '删除失败')
    }
  } catch (e) {
    showError('删除失败: ' + (e.message || e))
  }
}

onMounted(load)
defineExpose({ load })
</script>

<style scoped>
/* 复用 mgm-* 样式前缀保证设置面板整体一致 */
.pm-manager { font-size: 13px; }
.mgm-toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; padding-bottom: 8px; border-bottom: 1px solid var(--bd, #333); }
.mgm-count { color: var(--txt-dim, #8a8f98); font-size: 12px; }
.mgm-btn { padding: 4px 10px; border: 1px solid var(--bd, #333); border-radius: 4px; background: transparent; color: var(--txt, #ccc); cursor: pointer; font-size: 12px; }
.mgm-btn:hover { background: var(--bd-hover, #2a2a2a); }
.mgm-primary { background: var(--accent, #3b82f6); border-color: var(--accent, #3b82f6); color: #fff; }
.mgm-primary:hover { background: var(--accent-hover, #2f6fe0); }
.mgm-danger { color: #e05555; border-color: #5a3333; }
.mgm-small { padding: 2px 8px; font-size: 12px; }
.mgm-edit { border: 1px solid var(--bd, #333); border-radius: 6px; padding: 12px; margin-bottom: 12px; background: var(--bg-2, #1a1a1f); }
.mgm-edit-title { font-weight: 600; margin-bottom: 10px; }
.mgm-field { margin-bottom: 10px; }
.mgm-field-label { display: block; font-size: 12px; color: var(--txt-dim, #8a8f98); margin-bottom: 6px; }
.mgm-required { color: #e05555; margin-left: 2px; }
.mgm-field-hint { display: block; font-size: 11px; color: var(--txt-dim, #8a8f98); margin-top: 4px; }
/* ★ 用元素选择器（不用 input[type=…] 属性选择器）：模板部分输入框无显式 type 时
   属性选择器匹配不上（CSS 属性选择器只匹配显式属性）→ 输入框丢失全部样式 */
.mgm-field input, .mgm-field select {
  width: 100%; padding: 6px 8px; border: 1px solid var(--bd, #333); border-radius: 4px;
  background: var(--bg, #111); color: var(--txt, #ccc); box-sizing: border-box;
  font-size: 13px; font-family: inherit;
}
.mgm-field select { appearance: auto; }
.mgm-field input:focus, .mgm-field select:focus {
  border-color: var(--accent, #3b82f6); outline: none;
}
.mgm-edit-actions { display: flex; gap: 8px; }

.mgm-cards { display: flex; flex-direction: column; gap: 8px; }
.mgm-card { border: 1px solid var(--bd, #333); border-radius: 6px; padding: 10px 12px; }
.mgm-card.pm-active { border-color: var(--accent, #3b82f6); }
.mgm-card-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.mgm-name { font-weight: 600; }
.pm-active-badge { margin-left: 8px; font-size: 11px; color: var(--accent, #3b82f6); border: 1px solid var(--accent, #3b82f6); border-radius: 3px; padding: 0 4px; }
.mgm-ops { display: flex; gap: 6px; }
.mgm-empty { display: flex; justify-content: center; }
.mgm-empty-box {
  display: flex; flex-direction: column; align-items: center; gap: 10px;
  margin: 18px 0; padding: 26px 24px; max-width: 340px; width: 100%;
  border: 1px dashed var(--bd, #333); border-radius: 8px; text-align: center;
}
.mgm-empty-title { font-size: 13px; font-weight: 600; color: var(--txt, #ccc); }
.mgm-empty-sub { font-size: 12px; color: var(--txt-dim, #8a8f98); line-height: 1.5; }
.mgm-error { color: #e05555; margin-top: 8px; font-size: 12px; }
.pm-preview { border: 1px dashed var(--bd, #333); border-radius: 4px; padding: 8px; display: flex; flex-direction: column; gap: 4px; }
.pm-snap-row { display: flex; justify-content: space-between; font-size: 12px; }
.pm-snap-row span { color: var(--txt-dim, #8a8f98); }
</style>
