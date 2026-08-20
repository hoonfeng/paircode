<template>
  <div class="pm-manager">
    <!-- 工具栏 -->
    <div class="mgm-toolbar">
      <span class="mgm-count">{{ names.length }} 个配置</span>
      <button class="mgm-btn mgm-primary" @click="openAdd">＋ 添加新配置</button>
    </div>

    <!-- 添加 / 编辑表单（点击添加/编辑才弹出） -->
    <div v-if="showForm" class="mgm-edit">
      <div class="mgm-edit-title">{{ editingName ? '编辑配置：' + editingName : '添加新配置' }}</div>
      <div class="mgm-field">
        <span class="mgm-field-label">配置名称</span>
        <input v-model="form.name" placeholder="如：主力 / 写作备用…" @keydown.enter="confirmSave" />
      </div>
      <div class="mgm-field">
        <span class="mgm-field-label">服务商</span>
        <select v-model="form.provider" class="mgm-select" @change="onProviderChange">
          <option v-for="p in providers" :key="p" :value="p">{{ p }}</option>
        </select>
      </div>
      <div class="mgm-field">
        <span class="mgm-field-label">Base URL</span>
        <input v-model="form.baseURL" placeholder="https://api.deepseek.com/v1" />
      </div>
      <div class="mgm-field">
        <span class="mgm-field-label">API Key</span>
        <input v-model="form.apiKey" type="password" placeholder="sk-…" />
      </div>
      <div class="mgm-field">
        <span class="mgm-field-label">执行模型</span>
        <select v-model="form.executeModel" class="mgm-select">
          <option v-for="m in formModels" :key="m" :value="m">{{ m }}</option>
        </select>
      </div>
      <div class="mgm-field">
        <span class="mgm-field-label">规划模型</span>
        <select v-model="form.planModel" class="mgm-select">
          <option v-for="m in formModels" :key="m" :value="m">{{ m }}</option>
        </select>
      </div>
      <div class="mgm-field">
        <span class="mgm-field-label">审核模型</span>
        <select v-model="form.reviewModel" class="mgm-select">
          <option v-for="m in formModels" :key="m" :value="m">{{ m }}</option>
        </select>
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
          <div class="pm-snap-row"><span>执行模型</span><b>{{ (presets[n] || {}).executeModel || '—' }}</b></div>
          <div class="pm-snap-row"><span>规划 / 审核</span><b>{{ (presets[n] || {}).planModel || '—' }} / {{ (presets[n] || {}).reviewModel || '—' }}</b></div>
          <div class="pm-snap-row"><span>API Key</span><b>{{ keyMask((presets[n] || {}).apiKey) }}</b></div>
        </div>
      </div>
    </div>
    <div v-else-if="!showForm" class="mgm-empty">还没有 AI 配置。点「＋ 添加新配置」去设置模型和 Key，保存后即可应用。</div>

    <div v-if="error" class="mgm-error">{{ error }}</div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '../api.js'

// ★ AI 配置列表（2026-08-20 改变模式）：AI tab 主视图 = 已添加的配置列表；
//   点「＋ 添加新配置」弹出表单设置 名称/服务商/模型/Key 后保存；
//   「应用」= 整套配置写回 settings（provider/baseURL/apiKey/模型），对话面板随 settings 自动同步。
//   数据经 /api/ai-presets（config/ai-presets.json），每条 = 完整 AI 配置快照。
const emit = defineEmits(['saved'])

const presets = ref({})        // { 配置名: AiPreset }
const names = computed(() => Object.keys(presets.value || {}))
const activeName = ref('')     // settings.preset（当前使用中的配置名）
const showForm = ref(false)
const editingName = ref('')    // '' = 添加；非空 = 编辑该配置
const saving = ref(false)
const applying = ref('')
const error = ref('')

// 服务商/模型数据（/api/models）
const modelData = ref(null)
const providers = computed(() => (modelData.value && modelData.value.providers) || [])
const formModels = computed(() => {
  const m = (modelData.value && modelData.value.models) || {}
  return m[form.value.provider] || []
})
const form = ref({ name: '', provider: '', baseURL: '', apiKey: '', executeModel: '', planModel: '', reviewModel: '' })

function showError(msg) { error.value = msg; setTimeout(() => { if (error.value === msg) error.value = '' }, 4000) }
function keyMask(k) { return k ? (k.slice(0, 10) + '…') : '—' }

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
    apiKey: (md.providerKeys && md.providerKeys[prov]) || '',
    models: ((md.models && md.models[prov]) || []),
  }
}

// 打开添加表单（服务商默认当前生效的，带出 BaseURL/Key）
function openAdd() {
  const s = (window && window.__PAIRCODE_CORE && window.__PAIRCODE_CORE.uiState && window.__PAIRCODE_CORE.uiState.state
    && window.__PAIRCODE_CORE.uiState.state.settings) || {}
  const prov = (s.provider && providers.value.includes(s.provider)) ? s.provider : (providers.value[0] || '')
  const info = providerInfo(prov)
  form.value = {
    name: '',
    provider: prov,
    baseURL: info.baseURL || '',
    apiKey: info.apiKey || '',
    executeModel: info.models.includes(s.executeModel) ? s.executeModel : (info.models[0] || ''),
    planModel: info.models.includes(s.planModel) ? s.planModel : (info.models[0] || ''),
    reviewModel: info.models.includes(s.reviewModel) ? s.reviewModel : (info.models[0] || ''),
  }
  editingName.value = ''
  showForm.value = true
}

// 打开编辑表单（预填配置快照）
function openEdit(name) {
  const p = (presets.value && presets.value[name]) || {}
  form.value = {
    name,
    provider: p.provider || '',
    baseURL: p.baseURL || '',
    apiKey: p.apiKey || '',
    executeModel: p.executeModel || '',
    planModel: p.planModel || '',
    reviewModel: p.reviewModel || '',
  }
  editingName.value = name
  showForm.value = true
}

function closeForm() { showForm.value = false; editingName.value = '' }

// 切换服务商 → 带出该服务商 BaseURL/Key + 模型列表默认选中
function onProviderChange() {
  const info = providerInfo(form.provider)
  form.value.baseURL = info.baseURL
  form.value.apiKey = info.apiKey
  form.value.executeModel = info.models[0] || ''
  form.value.planModel = info.models[0] || ''
  form.value.reviewModel = info.models[0] || ''
}

async function confirmSave() {
  const name = form.value.name.trim()
  if (!name) { showError('请输入配置名称'); return }
  if (!form.value.provider) { showError('请选择服务商'); return }
  if (!form.value.apiKey) { showError('请填写 API Key'); return }
  saving.value = true
  error.value = ''
  try {
    const preset = {
      provider: form.value.provider,
      baseURL: form.value.baseURL,
      apiKey: form.value.apiKey,
      executeModel: form.value.executeModel,
      planModel: form.value.planModel,
      reviewModel: form.value.reviewModel,
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
        await api.apiPut('/settings', { settings: { ...(window.__PAIRCODE_CORE?.uiState?.state?.settings || {}), preset: name }, pluginSettings: {} })
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
.mgm-toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
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
.mgm-field input[type="text"], .mgm-field input[type="password"], .mgm-field .mgm-select {
  width: 100%; padding: 6px 8px; border: 1px solid var(--bd, #333); border-radius: 4px;
  background: var(--bg, #111); color: var(--txt, #ccc); box-sizing: border-box;
}
.mgm-field .mgm-select { appearance: auto; }
.mgm-edit-actions { display: flex; gap: 8px; }
.mgm-cards { display: flex; flex-direction: column; gap: 8px; }
.mgm-card { border: 1px solid var(--bd, #333); border-radius: 6px; padding: 10px 12px; }
.mgm-card.pm-active { border-color: var(--accent, #3b82f6); }
.mgm-card-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.mgm-name { font-weight: 600; }
.pm-active-badge { margin-left: 8px; font-size: 11px; color: var(--accent, #3b82f6); border: 1px solid var(--accent, #3b82f6); border-radius: 3px; padding: 0 4px; }
.mgm-ops { display: flex; gap: 6px; }
.mgm-empty { color: var(--txt-dim, #8a8f98); padding: 12px 0; font-size: 12px; }
.mgm-error { color: #e05555; margin-top: 8px; font-size: 12px; }
.pm-preview { border: 1px dashed var(--bd, #333); border-radius: 4px; padding: 8px; display: flex; flex-direction: column; gap: 4px; }
.pm-snap-row { display: flex; justify-content: space-between; font-size: 12px; }
.pm-snap-row span { color: var(--txt-dim, #8a8f98); }
</style>
