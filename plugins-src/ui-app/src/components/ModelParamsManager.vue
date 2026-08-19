<template>
  <div class="mp-manager">
    <!-- 选择器：服务商 + 模型 -->
    <div class="mp-pickers">
      <label class="mp-pick">
        <span>服务商</span>
        <select v-model="provider" @change="onProviderChange">
          <option value="">请选择…</option>
          <option v-for="p in providers" :key="p" :value="p">{{ p }}</option>
        </select>
      </label>
      <label class="mp-pick">
        <span>模型</span>
        <select v-model="model">
          <option value="">请选择…</option>
          <option v-for="m in models" :key="m" :value="m">{{ m }}</option>
        </select>
      </label>
    </div>

    <!-- 参数表单 -->
    <div v-if="provider && model" class="mp-form">
      <div class="mp-form-title">参数配置：{{ provider }} / {{ model }}</div>
      <div class="mp-form-grid">
        <label class="mp-field">
          <span>温度（随机性）</span>
          <select v-model="form.temperature">
            <option value="">默认</option>
            <option v-for="t in TEMPS" :key="t" :value="t">{{ t }}</option>
          </select>
        </label>
        <label class="mp-field">
          <span>思考模式</span>
          <select v-model="form.thinkingMode">
            <option value="">默认</option>
            <option value="thinking">thinking（深度思考，更慢更准）</option>
            <option value="non-thinking">non-thinking（快速响应）</option>
          </select>
        </label>
        <label class="mp-field">
          <span>最大输出 Token</span>
          <input type="number" v-model.number="form.maxTokens" placeholder="0=默认" min="0" max="131072" step="1024" />
        </label>
        <label class="mp-field">
          <span>上下文窗口</span>
          <input type="number" v-model.number="form.contextMaxTokens" placeholder="0=默认" min="0" max="200000" step="4096" />
        </label>
      </div>
      <div class="mp-actions">
        <button class="mp-btn mp-primary" :disabled="saving" @click="save">
          {{ saving ? '保存中…' : '保存该模型参数' }}
        </button>
        <button v-if="isConfigured" class="mp-btn mp-danger" @click="clearConfig">清空（恢复默认）</button>
      </div>
      <div v-if="msg" class="mp-msg">{{ msg }}</div>
    </div>
    <div v-else class="mp-empty">选择服务商与模型后，可为此模型单独配置生成参数</div>

    <!-- 已配置列表 -->
    <div v-if="configuredList.length" class="mp-list">
      <div class="mp-list-title">已配置参数的模型（{{ configuredList.length }}）</div>
      <div v-for="(c, i) in configuredList" :key="i" class="mp-item">
        <span class="mp-item-name">{{ c.provider }} / {{ c.model }}</span>
        <code class="mp-item-summary">{{ c.summary }}</code>
        <button class="mp-del" title="删除该模型配置" @click="removeConfig(c.provider, c.model)">×</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { state } from '../ui-state.js'
import api from '../api.js'

// 模型级参数配置：settings.json 顶层 modelParams（{服务商: {模型: {temperature, thinkingMode, maxTokens, contextMaxTokens}}}）。
// 装配器（agentloop providerFactory）按 服务商+模型 精确匹配覆盖；未配置的模型沿用默认。
const TEMPS = ['0', '0.1', '0.2', '0.3', '0.4', '0.5', '0.6', '0.7', '0.8', '0.9', '1.0', '1.2', '1.5', '2.0']

const provider = ref('')
const model = ref('')
const providers = ref([])
const models = ref([])
const form = ref({ temperature: '', thinkingMode: '', maxTokens: 0, contextMaxTokens: 0 })
const saving = ref(false)
const msg = ref('')

function allParams() {
  return (state.settings && state.settings.modelParams) || {}
}
function currentEntry() {
  if (!provider.value || !model.value) return null
  const byProv = allParams()[provider.value]
  return byProv ? (byProv[model.value] || null) : null
}
const isConfigured = computed(() => !!currentEntry())
const configuredList = computed(() => {
  const out = []
  const mp = allParams()
  for (const p of Object.keys(mp || {})) {
    for (const m of Object.keys(mp[p] || {})) {
      const e = mp[p][m] || {}
      const parts = []
      if (e.temperature !== undefined && e.temperature !== '') parts.push('温度 ' + e.temperature)
      if (e.thinkingMode) parts.push('思考 ' + e.thinkingMode)
      if (e.maxTokens) parts.push('输出 ' + e.maxTokens)
      if (e.contextMaxTokens) parts.push('上下文 ' + e.contextMaxTokens)
      out.push({ provider: p, model: m, summary: parts.join(' · ') || '（空配置）' })
    }
  }
  return out
})

async function load() {
  try {
    const d = await api.getModels()
    providers.value = d.providers || []
    // 默认选中当前服务商
    if (!provider.value && state.settings && state.settings.provider && providers.value.includes(state.settings.provider)) {
      provider.value = state.settings.provider
      onProviderChange()
    }
  } catch {}
}
function onProviderChange() {
  model.value = ''
  const d = null // 模型列表从 modelData 拿？组件独立拉
  refreshModels()
}
async function refreshModels() {
  if (!provider.value) { models.value = []; return }
  try {
    const d = await api.getModels()
    const ms = (d.models || {})[provider.value] || []
    // 并入已配置但可能已从列表移除的模型
    const mp = allParams()[provider.value] || {}
    for (const m of Object.keys(mp)) if (!ms.includes(m)) ms.push(m)
    models.value = ms
    // 自动选中当前执行模型
    const cur = state.settings && state.settings.executeModel
    if (cur && ms.includes(cur)) model.value = cur
    else if (ms.length === 1) model.value = ms[0]
    else if (ms.length) model.value = ''
  } catch {}
}
function syncForm() {
  const e = currentEntry()
  form.value = {
    temperature: e && e.temperature !== undefined && e.temperature !== null ? String(e.temperature) : '',
    thinkingMode: (e && e.thinkingMode) || '',
    maxTokens: (e && e.maxTokens) || 0,
    contextMaxTokens: (e && e.contextMaxTokens) || 0,
  }
}
async function save() {
  saving.value = true
  msg.value = ''
  try {
    const mp = JSON.parse(JSON.stringify(allParams()))
    if (!mp[provider.value]) mp[provider.value] = {}
    mp[provider.value][model.value] = {
      temperature: form.value.temperature,
      thinkingMode: form.value.thinkingMode,
      maxTokens: form.value.maxTokens || 0,
      contextMaxTokens: form.value.contextMaxTokens || 0,
    }
    const top = { ...(state.settings || {}), modelParams: mp }
    await api.apiPut('/settings', { settings: top, pluginSettings: (state.settings && state.settings.pluginSettings) || {} })
    state.settings = top
    msg.value = `已保存：${provider.value} / ${model.value} 的参数`
    setTimeout(() => { msg.value = '' }, 2500)
  } catch (e) {
    msg.value = '保存失败: ' + (e.message || e)
  } finally { saving.value = false }
}
function clearConfig() {
  const mp = JSON.parse(JSON.stringify(allParams()))
  if (mp[provider.value]) delete mp[provider.value][model.value]
  if (mp[provider.value] && !Object.keys(mp[provider.value]).length) delete mp[provider.value]
  const top = { ...(state.settings || {}), modelParams: mp }
  api.apiPut('/settings', { settings: top, pluginSettings: (state.settings && state.settings.pluginSettings) || {} })
    .then(() => {
      state.settings = top
      syncForm()
      msg.value = '已清空该模型参数'
      setTimeout(() => { msg.value = '' }, 2000)
    }).catch(e => { msg.value = '清空失败: ' + (e.message || e) })
}
function removeConfig(p, m) {
  const mp = JSON.parse(JSON.stringify(allParams()))
  if (mp[p]) delete mp[p][m]
  if (mp[p] && !Object.keys(mp[p]).length) delete mp[p]
  const top = { ...(state.settings || {}), modelParams: mp }
  api.apiPut('/settings', { settings: top, pluginSettings: (state.settings && state.settings.pluginSettings) || {} })
    .then(() => { state.settings = top; if (p === provider.value && m === model.value) syncForm() })
    .catch(e => { msg.value = '删除失败: ' + (e.message || e) })
}

// 监听选择变化：切换服务商/模型时同步表单
import { watch } from 'vue'
watch(provider, refreshModels)
watch(model, syncForm)
onMounted(load)
</script>

<style scoped>
.mp-manager { display: flex; flex-direction: column; gap: 14px; }
.mp-pickers { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.mp-pick { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--text-secondary, #999); }
.mp-pick select {
  width: 100%; box-sizing: border-box;
  background: var(--input-bg, #14141f);
  border: 1px solid var(--border-color, #3a3a4a);
  color: var(--text-primary, #eee); border-radius: 6px;
  padding: 6px 10px; font-size: 13px; outline: none;
}
.mp-pick select:focus { border-color: var(--accent, #4f8cff); }
.mp-form {
  display: flex; flex-direction: column; gap: 12px;
  padding: 14px; border: 1px solid var(--border-color, #3a3a4a);
  border-radius: 10px; background: var(--bg-tertiary, rgba(0,0,0,.15));
}
.mp-form-title { font-size: 13px; font-weight: 600; color: var(--text-primary, #eee);
  padding-bottom: 8px; border-bottom: 1px solid var(--border-color, #333); }
.mp-form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.mp-field { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--text-secondary, #999); }
.mp-field select, .mp-field input {
  width: 100%; box-sizing: border-box;
  background: var(--input-bg, #14141f);
  border: 1px solid var(--border-color, #3a3a4a);
  color: var(--text-primary, #eee); border-radius: 6px;
  padding: 6px 10px; font-size: 13px; outline: none; font-family: inherit;
}
.mp-field select:focus, .mp-field input:focus { border-color: var(--accent, #4f8cff); }
.mp-actions { display: flex; gap: 8px; }
.mp-btn {
  padding: 6px 14px; border-radius: 7px; font-size: 13px; cursor: pointer;
  border: 1px solid var(--border-color, #444); background: none;
  color: var(--text-primary, #ddd); transition: all .15s;
}
.mp-btn:hover { background: var(--bg-hover, rgba(255,255,255,.06)); }
.mp-btn:disabled { opacity: .5; cursor: not-allowed; }
.mp-btn.mp-primary { background: var(--accent, #4f8cff); color: #fff; border-color: var(--accent, #4f8cff); font-weight: 600; }
.mp-btn.mp-primary:hover { filter: brightness(1.12); background: var(--accent, #4f8cff); }
.mp-btn.mp-danger { color: #e06c6c; border-color: rgba(224,108,108,.4); }
.mp-btn.mp-danger:hover { background: rgba(224,108,108,.12); }
.mp-msg { font-size: 12px; color: #7ecb7e; }
.mp-empty { color: var(--text-secondary, #888); text-align: center; padding: 24px 0; font-size: 13px; }
.mp-list { display: flex; flex-direction: column; gap: 6px; }
.mp-list-title { font-size: 12px; font-weight: 600; color: var(--text-secondary, #999); }
.mp-item {
  display: flex; align-items: center; gap: 10px;
  padding: 7px 10px; border: 1px solid var(--border-color, #333);
  border-radius: 7px; background: var(--bg-tertiary, rgba(0,0,0,.1));
  font-size: 12px; color: var(--text-primary, #ddd);
}
.mp-item-name { font-weight: 600; white-space: nowrap; }
.mp-item-summary { color: var(--text-secondary, #aaa); font-size: 11px; flex: 1; }
.mp-del {
  width: 20px; height: 20px; border: none; background: none; cursor: pointer;
  color: #e06c6c; font-size: 14px; border-radius: 4px; flex-shrink: 0;
}
.mp-del:hover { background: rgba(224,108,108,.15); }
</style>
