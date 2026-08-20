<template>
  <div class="provider-manager">
    <!-- 工具栏 -->
    <div class="pm-toolbar">
      <span class="pm-count">{{ providers.length }} 个服务商</span>
      <button class="pm-btn pm-primary" @click="startAdd">+ 新增服务商</button>
    </div>

    <!-- 新增表单（工具栏下方展开，紧邻按钮不跳动） -->
    <div v-if="editingName === '__new__'" class="pm-edit">
      <div class="pm-edit-title">新增服务商</div>
      <div class="pm-field">
        <span class="pm-field-label">服务商名称</span>
        <input v-model="editForm.name" placeholder="如 deepseek" />
      </div>
      <div class="pm-field">
        <span class="pm-field-label">Base URL</span>
        <input v-model="editForm.baseURL" placeholder="https://api.deepseek.com/v1" />
      </div>
      <div class="pm-field">
        <span class="pm-field-label">API Key（该服务商独立保存）</span>
        <input v-model="editForm.apiKey" type="password" placeholder="sk-…" />
      </div>
      <div class="pm-field">
        <span class="pm-field-label">上下文大小（Token）</span>
        <input v-model="editForm.contextMaxTokens" type="number" min="0" step="1000" placeholder="0=不限制（模型级未配置时的默认窗口）" />
      </div>
      <ModelEditor :models="editModels" @change="onModelsChange" />
          <div class="pm-params">
            <div class="pm-params-title">模型参数（每模型独立配置；对话里也可临时切换思考档位）</div>
            <div v-if="editModels.length" class="pm-param-rows">
              <div v-for="m in editModels" :key="m" class="pm-param-row">
                <span class="pm-param-model" :title="m">{{ m }}</span>
                <select v-model="editParams[m].temperature" title="温度（随机性）">
                  <option v-for="t in TEMPS" :key="'t' + t" :value="t">{{ t === '' ? '温度默认' : t }}</option>
                </select>
                <select v-model="editParams[m].thinkingMode" title="思考档位（OpenAI 定义）">
                  <option v-for="th in THINK_TIERS" :key="'k' + th.v" :value="th.v">{{ th.label }}</option>
                </select>
                <input v-model.number="editParams[m].maxTokens" type="number" min="0" step="1024" placeholder="输出 Token" title="最大输出 Token（0=默认）" />
                <input v-model.number="editParams[m].contextMaxTokens" type="number" min="0" step="4096" placeholder="上下文" title="上下文窗口（0=默认）" />
              </div>
            </div>
            <div v-else class="pm-params-empty">添加模型后，可逐模型配置温度/思考档位/输出上限/上下文窗口</div>
          </div>

      <div class="pm-edit-actions">
        <button class="pm-btn pm-primary" :disabled="saving" @click="saveEdit">
          {{ saving ? '保存中…' : '保存服务商' }}
        </button>
        <button class="pm-btn" @click="cancelEdit">取消</button>
      </div>
    </div>

    <!-- 服务商卡片列表（编辑时在卡片位置就地展开表单，不跳顶） -->
    <div v-if="providers.length" class="pm-cards">
      <template v-for="p in providers" :key="p.name">
        <div v-if="editingName === p.name" class="pm-edit">
          <div class="pm-edit-title">编辑服务商：{{ p.name }}</div>
          <div class="pm-field">
            <span class="pm-field-label">服务商名称</span>
            <input :value="p.name" disabled />
          </div>
          <div class="pm-field">
            <span class="pm-field-label">Base URL</span>
            <input v-model="editForm.baseURL" placeholder="https://api.deepseek.com/v1" />
          </div>
          <div class="pm-field">
            <span class="pm-field-label">API Key（该服务商独立保存）</span>
            <input v-model="editForm.apiKey" type="password" placeholder="sk-…" />
          </div>
          <div class="pm-field">
            <span class="pm-field-label">上下文大小（Token）</span>
            <input v-model="editForm.contextMaxTokens" type="number" min="0" step="1000" placeholder="0=不限制（模型级未配置时的默认窗口）" />
          </div>
          <ModelEditor :models="editModels" @change="onModelsChange" />
          <div class="pm-params">
            <div class="pm-params-title">模型参数（每模型独立配置；对话里也可临时切换思考档位）</div>
            <div v-if="editModels.length" class="pm-param-rows">
              <div v-for="m in editModels" :key="m" class="pm-param-row">
                <span class="pm-param-model" :title="m">{{ m }}</span>
                <select v-model="editParams[m].temperature" title="温度（随机性）">
                  <option v-for="t in TEMPS" :key="'t' + t" :value="t">{{ t === '' ? '温度默认' : t }}</option>
                </select>
                <select v-model="editParams[m].thinkingMode" title="思考档位（OpenAI 定义）">
                  <option v-for="th in THINK_TIERS" :key="'k' + th.v" :value="th.v">{{ th.label }}</option>
                </select>
                <input v-model.number="editParams[m].maxTokens" type="number" min="0" step="1024" placeholder="输出 Token" title="最大输出 Token（0=默认）" />
                <input v-model.number="editParams[m].contextMaxTokens" type="number" min="0" step="4096" placeholder="上下文" title="上下文窗口（0=默认）" />
              </div>
            </div>
            <div v-else class="pm-params-empty">添加模型后，可逐模型配置温度/思考档位/输出上限/上下文窗口</div>
          </div>

          <div class="pm-edit-actions">
            <button class="pm-btn pm-primary" :disabled="saving" @click="saveEdit">
              {{ saving ? '保存中…' : '保存服务商' }}
            </button>
            <button class="pm-btn" @click="cancelEdit">取消</button>
          </div>
        </div>
        <div v-else class="pm-card">
          <div class="pm-card-head">
            <span class="pm-name" :title="p.name">{{ p.name }}</span>
            <div class="pm-ops">
              <button class="pm-btn pm-small" @click="startEdit(p)">编辑</button>
              <button class="pm-btn pm-small pm-danger" @click="removeProvider(p)">删除</button>
            </div>
          </div>
          <div class="pm-url" :title="p.baseURL">{{ p.baseURL || '未配置 Base URL' }}</div>
          <div class="pm-key" :class="{ 'pm-key-ok': p.apiKey }" :title="p.apiKey ? '已配置 API Key' : '未配置 API Key'">{{ p.apiKey ? 'API Key 已配置' : '未配置 API Key' }}</div>
          <div class="pm-ctx">{{ p.contextMaxTokens > 0 ? ('上下文 ' + (p.contextMaxTokens / 1000).toFixed(0) + 'K Token') : '上下文 未限制' }}</div>
          <div class="pm-models">
            <span v-if="!p.models.length" class="pm-none">（未配置模型）</span>
            <span v-for="m in p.models" :key="m" class="pm-tag">{{ m }}</span>
          </div>
          <div v-if="paramsSummary(p.name)" class="pm-params-summary">{{ paramsSummary(p.name) }}</div>
        </div>
      </template>
    </div>
    <div v-else-if="editingName !== '__new__'" class="pm-empty">暂无服务商，点「+ 新增服务商」添加</div>

    <div v-if="error" class="pm-error">{{ error }}</div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { state } from '../ui-state.js'
import api from '../api.js'
import ModelEditor from './ModelEditor.vue'

// 服务商管理面板：维护 config/models.json（GET/POST /api/models）。
// 保存后 emit('saved') 通知父组件刷新 AI tab 的 provider/模型下拉。
const emit = defineEmits(['saved'])

const providers = ref([])
const editingName = ref('')        // '' = 不编辑；'__new__' = 新增；其他 = 编辑该服务商（就地展开）
const editForm = ref({ name: '', baseURL: '', apiKey: '', contextMaxTokens: 0 })
const editModels = ref([])
const editParams = ref({})   // 模型级参数：{模型: {temperature, thinkingMode, maxTokens, contextMaxTokens}} → settings.json modelParams
const error = ref('')
const saving = ref(false)

// 思考档位（OpenAI ReasoningEffort）+ 温度档位
const THINK_TIERS = [
  { v: '', label: '默认' },
  { v: 'none', label: 'none（关闭）' },
  { v: 'minimal', label: 'minimal（极简）' },
  { v: 'low', label: 'low（低）' },
  { v: 'medium', label: 'medium（中）' },
  { v: 'high', label: 'high（高）' },
  { v: 'xhigh', label: 'xhigh（超高）' },
  { v: 'max', label: 'max（最大化）' },
]
const TEMPS = ['', '0', '0.1', '0.2', '0.3', '0.4', '0.5', '0.6', '0.7', '0.8', '0.9', '1.0', '1.2', '1.5', '2.0']

function readProviderParams(providerName) {
  const mp = (state.settings && state.settings.modelParams) || {}
  return JSON.parse(JSON.stringify(mp[providerName] || {}))
}

async function load() {
  try {
    const d = await api.getModels()
    providers.value = (d.providers || []).map(name => ({
      name,
      baseURL: (d.providerBaseURLs || {})[name] || '',
      apiKey: (d.providerKeys || {})[name] || '',
      contextMaxTokens: (d.providerContexts || {})[name] || 0, // ★ 服务商级默认上下文窗口
      models: (d.models || {})[name] || [],
    }))
    error.value = ''
  } catch (e) {
    error.value = '加载服务商失败: ' + (e.message || e)
  }
}
onMounted(load)

function startAdd() {
  editingName.value = '__new__'
  editForm.value = { name: '', baseURL: '', apiKey: '', contextMaxTokens: 0 }
  editModels.value = []
  editParams.value = {}
  error.value = ''
}
function startEdit(p) {
  editingName.value = p.name
  editForm.value = { name: p.name, baseURL: p.baseURL, apiKey: p.apiKey || '', contextMaxTokens: p.contextMaxTokens || 0 }
  editModels.value = [...(p.models || [])]
  const params = readProviderParams(p.name)
  // 为所有模型补默认参数键（模板 v-model 需要键存在）
  for (const m of editModels.value) if (!params[m]) params[m] = { temperature: '', thinkingMode: '', maxTokens: 0, contextMaxTokens: 0 }
  editParams.value = params
  error.value = ''
}

// 模型列表变化：新模型补参数键、移除模型清理参数
function onModelsChange(list) {
  const params = { ...editParams.value }
  for (const m of list) if (!params[m]) params[m] = { temperature: '', thinkingMode: '', maxTokens: 0, contextMaxTokens: 0 }
  for (const m of Object.keys(params)) if (!list.includes(m)) delete params[m]
  editParams.value = params
  editModels.value = list
}
function cancelEdit() { editingName.value = ''; error.value = '' }

// 当前列表 → 全量快照 map（供 POST /api/models）
function snapshot() {
  const map = {}
  for (const p of providers.value) map[p.name] = { baseURL: p.baseURL, models: p.models, apiKey: p.apiKey || '', contextMaxTokens: p.contextMaxTokens || 0 }
  return map
}

async function saveEdit() {
  const name = editForm.value.name.trim() || (editingName.value !== '__new__' ? editingName.value : '')
  if (!name) { error.value = '服务商名称不能为空'; return }
  const map = snapshot()
  if (editingName.value === '__new__' && map[name]) { error.value = `服务商「${name}」已存在`; return }
  map[name] = {
    baseURL: editForm.value.baseURL.trim(),
    models: editModels.value,
    apiKey: (editForm.value.apiKey || '').trim(),
    contextMaxTokens: Math.max(0, Number(editForm.value.contextMaxTokens) || 0), // ★ 服务商级默认上下文窗口
  }
  saving.value = true
  try {
    await api.saveModels(map)
    await saveModelParams(name) // ★ 模型参数同步 settings.modelParams
    editingName.value = ''
    await load()
    emit('saved') // AI tab 下拉同步刷新
  } catch (e) {
    error.value = '保存失败: ' + (e.message || e)
  } finally { saving.value = false }
}

// 将当前编辑的模型参数写回 settings.json 顶层 modelParams（仅保留非空项）
async function saveModelParams(providerName) {
  // ★ 先拉后端最新 settings 作基底，避免过期缓存覆盖其他字段
  let base = {};
  try { const l = await api.apiGet('/settings'); base = (l && l.settings) || {} } catch {}
  const mp = JSON.parse(JSON.stringify((base.modelParams) || {}))
  const clean = {}
  for (const [m, cfg] of Object.entries(editParams.value)) {
    const c = cfg || {}
    const out = {}
    if (c.temperature !== '' && c.temperature !== undefined && c.temperature !== null) out.temperature = c.temperature
    if (c.thinkingMode) out.thinkingMode = c.thinkingMode
    if (Number(c.maxTokens) > 0) out.maxTokens = Number(c.maxTokens)
    if (Number(c.contextMaxTokens) > 0) out.contextMaxTokens = Number(c.contextMaxTokens)
    if (Object.keys(out).length) clean[m] = out
  }
  if (Object.keys(clean).length) mp[providerName] = clean
  else delete mp[providerName]
  const top = { ...base, modelParams: mp }
  await api.apiPut('/settings', { settings: top, pluginSettings: (base.pluginSettings) || {} })
  state.settings = top
}

async function removeProvider(p) {
  if (!window.confirm(`删除服务商「${p.name}」？\n（AI tab 将不再可选该服务商）`)) return
  const map = snapshot()
  delete map[p.name]
  try {
    await api.saveModels(map)
    // 同步清理该服务商的模型参数（先拉后端最新 settings 作基底，避免缓存覆盖）
    let base = {};
    try { const l = await api.apiGet('/settings'); base = (l && l.settings) || {} } catch {}
    const mp = JSON.parse(JSON.stringify((base.modelParams) || {}))
    if (mp[p.name]) {
      delete mp[p.name]
      const top = { ...base, modelParams: mp }
      await api.apiPut('/settings', { settings: top, pluginSettings: (base.pluginSettings) || {} })
      state.settings = top
    }
    await load()
    emit('saved')
  } catch (e) {
    error.value = '删除失败: ' + (e.message || e)
  }
}

function paramsSummary(providerName) {
  const mp = (state.settings && state.settings.modelParams) || {}
  const by = mp[providerName] || {}
  const n = Object.keys(by).length
  return n ? '模型参数已配置 ' + n + ' 个' : ''
}
</script>

<style scoped>
.provider-manager { display: flex; flex-direction: column; gap: 14px; }
.pm-toolbar { display: flex; align-items: center; justify-content: space-between; }
.pm-count { font-size: 12px; color: var(--text-secondary, #999); }
.pm-btn {
  padding: 6px 14px; border-radius: 7px; font-size: 13px; cursor: pointer;
  border: 1px solid var(--border-color, #444); background: none;
  color: var(--text-primary, #ddd); transition: all .15s;
}
.pm-btn:hover { background: var(--bg-hover, rgba(255,255,255,.06)); }
.pm-btn:disabled { opacity: .5; cursor: not-allowed; }
.pm-btn.pm-primary {
  background: var(--accent, #4f8cff); color: #fff; border-color: var(--accent, #4f8cff); font-weight: 600;
}
.pm-btn.pm-primary:hover { filter: brightness(1.12); background: var(--accent, #4f8cff); }
.pm-btn.pm-small { padding: 3px 10px; font-size: 12px; }
.pm-btn.pm-danger { color: #e06c6c; border-color: rgba(224,108,108,.4); }
.pm-btn.pm-danger:hover { background: rgba(224,108,108,.12); }

/* ─── 编辑表单（卡片式分组；在卡片列表内 grid-column 跨整行，就地展开）─── */
.pm-edit {
  display: flex; flex-direction: column; gap: 12px;
  padding: 16px; border: 1px solid var(--border-color, #3a3a4a);
  border-radius: 10px; background: var(--bg-tertiary, rgba(0,0,0,.15));
  grid-column: 1 / -1; /* 列表内编辑：占满整行，就地展开不跳动 */
}
.pm-edit-title {
  font-size: 14px; font-weight: 600; color: var(--text-primary, #eee);
  padding-bottom: 10px; border-bottom: 1px solid var(--border-color, #333);
}
.pm-field { display: flex; flex-direction: column; gap: 5px; }
.pm-field-label { font-size: 12px; color: var(--text-secondary, #999); font-weight: 500; }
.pm-field input[type="text"], .pm-field > input {
  width: 100%; box-sizing: border-box;
  background: var(--input-bg, #14141f);
  border: 1px solid var(--border-color, #3a3a4a);
  color: var(--text-primary, #eee); border-radius: 6px;
  padding: 7px 10px; font-size: 13px; outline: none; font-family: inherit;
  transition: border-color .15s;
}
.pm-field > input:focus { border-color: var(--accent, #4f8cff); }
.pm-field > input:disabled { opacity: .5; }

.pm-edit-actions { display: flex; gap: 8px; justify-content: flex-end; padding-top: 4px; }

/* ─── 服务商卡片 ─── */
.pm-cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 10px; }
.pm-card {
  display: flex; flex-direction: column; gap: 8px;
  padding: 12px 14px; border: 1px solid var(--border-color, #333);
  border-radius: 9px; background: var(--bg-tertiary, rgba(0,0,0,.12));
  transition: border-color .15s, background .15s;
}
.pm-card:hover { border-color: var(--border-color, #4a4a5a); background: var(--bg-tertiary, rgba(0,0,0,.18)); }
.pm-card-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.pm-name { font-weight: 600; color: var(--text-primary, #eee); font-size: 13px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.pm-ops { display: flex; gap: 6px; flex-shrink: 0; }
.pm-url {
  font-size: 11px; color: var(--text-secondary, #999);
  word-break: break-all; line-height: 1.5;
}
.pm-ctx { font-size: 11px; color: var(--text-secondary, #999); }
.pm-models { display: flex; flex-wrap: wrap; gap: 5px; }
.pm-tag {
  font-size: 11px; padding: 2px 9px; border-radius: 10px;
  background: rgba(79,140,255,.1); color: #8ab4ff;
  border: 1px solid rgba(79,140,255,.22); white-space: nowrap;
}
.pm-key { font-size: 11px; color: var(--text-secondary, #777); }
.pm-key-ok { color: #7ecb7e; }
.pm-none { color: var(--text-secondary, #777); font-size: 12px; }
.pm-empty { color: var(--text-secondary, #888); text-align: center; padding: 30px 0; font-size: 13px; }
.pm-error {
  color: #e06c6c; font-size: 12px; padding: 8px 10px;
  border: 1px solid rgba(224,108,108,.3); border-radius: 6px;
  background: rgba(224,108,108,.08);
}
.pm-params { display: flex; flex-direction: column; gap: 6px; }
.pm-params-title { font-size: 12px; color: var(--text-secondary, #999); font-weight: 600; }
.pm-param-rows { display: flex; flex-direction: column; gap: 6px; }
.pm-param-row {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 8px; border: 1px solid var(--border-color, #333);
  border-radius: 6px; background: var(--bg-tertiary, rgba(0,0,0,.1));
}
.pm-param-model {
  flex: 0 0 auto; max-width: 140px; font-size: 12px; font-weight: 500;
  color: var(--text-primary, #ddd); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.pm-param-row select, .pm-param-row input {
  background: var(--input-bg, #14141f);
  border: 1px solid var(--border-color, #3a3a4a);
  color: var(--text-primary, #eee); border-radius: 5px;
  padding: 4px 6px; font-size: 12px; outline: none; font-family: inherit;
}
.pm-param-row select { flex: 1.1; min-width: 0; }
.pm-param-row input { flex: 0 0 84px; width: 84px; }
.pm-param-row select:focus, .pm-param-row input:focus { border-color: var(--accent, #4f8cff); }
.pm-params-empty { font-size: 12px; color: var(--text-secondary, #777); padding: 4px 0; }
.pm-params-summary { font-size: 11px; color: var(--text-secondary, #999); }
</style>
