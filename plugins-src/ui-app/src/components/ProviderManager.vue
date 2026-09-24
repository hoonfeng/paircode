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
        <span class="pm-field-label">API URL（基础地址或完整端点）</span>
        <input v-model="editForm.baseURL" placeholder="https://api.deepseek.com/v1（基础地址；旧完整端点亦兼容）" />
      </div>
      <div class="pm-field">
        <span class="pm-field-label">{{ protocolLabel }}</span>
        <select v-model="editForm.protocol" :title="protocolHint">
          <option v-for="opt in protocolOptions" :key="'p'+opt" :value="opt">{{ opt || '默认' }}</option>
        </select>
        <span v-if="protocolHint" class="pm-protocol-hint">{{ protocolHint }}</span>
      </div>
      <div class="pm-field">
        <span class="pm-field-label">上下文大小（Token）</span>
        <input v-model="editForm.contextMaxTokens" type="number" min="0" step="1000" placeholder="0=不限制（模型级未配置时的默认窗口）" />
      </div>
      <div class="pm-field">
        <span class="pm-field-label">默认温度</span>
        <input v-model="editForm.temperature" placeholder="如 0.3（空=不设；模型级可覆盖）" />
      </div>
      <div class="pm-field">
        <span class="pm-field-label">默认输出 Token（最大输出）</span>
        <input v-model="editForm.maxTokens" type="number" min="0" step="1024" placeholder="0=不设（模型级可覆盖）" />
      </div>
<ModelEditor :models="editModels" :label="modelEditor.label || '可用模型（回车或逗号分隔添加；支持整段粘贴）'" :placeholder="modelEditor.placeholder || '输入模型名，回车添加…'" @change="onModelsChange" />
<div class="pm-params">
            <div class="pm-params-title">模型参数（每模型独立配置；对话里也可临时切换思考档位）</div>
            <div v-if="editModels.length" class="pm-param-rows">
              <div v-for="m in editModels" :key="m" class="pm-param-row">
                <span class="pm-param-model" :title="m">{{ m }}</span>
                <!-- ★ 2026-08-21 schema 驱动：按 modelParamFields 动态渲染（checkbox/select/number/text） -->
                <template v-for="f in modelParamFields" :key="f.name">
                  <label v-if="f.type === 'checkbox'" class="pm-param-check" :title="f.hint || f.label">
                    <input type="checkbox" v-model="editParams[m][f.name]" /> {{ f.label }}
                  </label>
                  <select v-else-if="f.type === 'select'" v-model="editParams[m][f.name]" :title="f.hint || f.label">
                    <option v-for="opt in (f.options || [])" :key="'o'+opt" :value="opt">{{ opt === '' ? (f.label + '默认') : opt }}</option>
                  </select>
                  <input v-else-if="f.type === 'number'" v-model.number="editParams[m][f.name]" type="number" :min="f.min ?? 0" :step="f.step ?? 1" :placeholder="f.label" :title="f.hint || f.label" />
                  <input v-else v-model="editParams[m][f.name]" type="text" :placeholder="f.label" :title="f.hint || f.label" />
                </template>
              </div>
            </div>
            <div v-else class="pm-params-empty">添加模型后，可逐模型配置参数（温度/思考/输出/上下文/多模态…）</div>
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
            <input v-model="editForm.name" placeholder="如 deepseek" />
            <span class="pm-hint">改名会一并同步引用该服务商的 AI 配置（连接信息不丢）；空=保持原名</span>
          </div>
          <div class="pm-field">
            <span class="pm-field-label">API URL（基础地址或完整端点）</span>
            <input v-model="editForm.baseURL" placeholder="https://api.deepseek.com/v1（基础地址；旧完整端点亦兼容）" />
          </div>
          <div class="pm-field">
            <span class="pm-field-label">{{ protocolLabel }}</span>
            <select v-model="editForm.protocol" :title="protocolHint">
              <option v-for="opt in protocolOptions" :key="'p'+opt" :value="opt">{{ opt || '默认' }}</option>
            </select>
            <span v-if="protocolHint" class="pm-protocol-hint">{{ protocolHint }}</span>
          </div>
          <div class="pm-field">
            <span class="pm-field-label">上下文大小（Token）</span>
            <input v-model="editForm.contextMaxTokens" type="number" min="0" step="1000" placeholder="0=不限制（模型级未配置时的默认窗口）" />
          </div>
          <div class="pm-field">
            <span class="pm-field-label">默认温度</span>
            <input v-model="editForm.temperature" placeholder="如 0.3（空=不设；模型级可覆盖）" />
          </div>
          <div class="pm-field">
            <span class="pm-field-label">默认输出 Token（最大输出）</span>
            <input v-model="editForm.maxTokens" type="number" min="0" step="1024" placeholder="0=不设（模型级可覆盖）" />
          </div>
<ModelEditor :models="editModels" :label="modelEditor.label || '可用模型（回车或逗号分隔添加；支持整段粘贴）'" :placeholder="modelEditor.placeholder || '输入模型名，回车添加…'" @change="onModelsChange" />
<div class="pm-params">
            <div class="pm-params-title">模型参数（每模型独立配置；对话里也可临时切换思考档位）</div>
            <div v-if="editModels.length" class="pm-param-rows">
              <div v-for="m in editModels" :key="m" class="pm-param-row">
                <span class="pm-param-model" :title="m">{{ m }}</span>
                <!-- ★ 2026-08-21 schema 驱动：按 modelParamFields 动态渲染（checkbox/select/number/text） -->
                <template v-for="f in modelParamFields" :key="f.name">
                  <label v-if="f.type === 'checkbox'" class="pm-param-check" :title="f.hint || f.label">
                    <input type="checkbox" v-model="editParams[m][f.name]" /> {{ f.label }}
                  </label>
                  <select v-else-if="f.type === 'select'" v-model="editParams[m][f.name]" :title="f.hint || f.label">
                    <option v-for="opt in (f.options || [])" :key="'o'+opt" :value="opt">{{ opt === '' ? (f.label + '默认') : opt }}</option>
                  </select>
                  <input v-else-if="f.type === 'number'" v-model.number="editParams[m][f.name]" type="number" :min="f.min ?? 0" :step="f.step ?? 1" :placeholder="f.label" :title="f.hint || f.label" />
                  <input v-else v-model="editParams[m][f.name]" type="text" :placeholder="f.label" :title="f.hint || f.label" />
                </template>
              </div>

            </div>
            <div v-else class="pm-params-empty">添加模型后，可逐模型配置参数（温度/思考/输出/上下文/多模态…）</div>
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
          <div class="pm-url" :title="p.baseURL">{{ p.baseURL || '未配置 API URL' }}</div>
          <div v-if="p.protocol" class="pm-protocol" :title="protocolHint">协议 {{ p.protocol }}</div>
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
import api from '../api.js'
import ModelEditor from './ModelEditor.vue'

// 服务商管理面板：维护 config/models.json（GET/POST /api/models）。
// 保存后 emit('saved') 通知父组件刷新 AI tab 的 provider/模型下拉。
// ★ 2026-08-21 模型参数区 schema 驱动：modelParamFields prop 声明逐模型参数字段
//   （temperature/thinkingMode/maxTokens/contextMaxTokens/multimodal…），
//   模板按 schema 动态渲染（checkbox/select/number/text），新增参数无需改本组件。
const emit = defineEmits(['saved'])
const props = defineProps({
  modelParamFields: { type: Array, default: () => [] }, // [{name,label,type,default,options,hint,min,max,step}]
  modelEditor: { type: Object, default: () => ({}) },   // ★ 2026-08-21 schema 驱动：{label, placeholder} 声明模型编辑器
  // ★ 2026-09-02 LLM 协议：选项/文案由插件注册配置（agentloop registerSettings 的 provider-manager 字段），前端不硬编码协议清单
  protocolLabel: { type: String, default: 'LLM 协议' },
  protocolOptions: { type: Array, default: () => [] },   // ['', 'openai-completions', 'openai-responses', 'anthropic-messages']…空=默认
  protocolHint: { type: String, default: '' },
})

const providers = ref([])
const editingName = ref('')        // '' = 不编辑；'__new__' = 新增；其他 = 编辑该服务商（就地展开）
// ★ 2026-09-19 生成参数（温度/最大输出）改为服务商级字段，随 models.json 一起保存
const editForm = ref({ name: '', baseURL: '', contextMaxTokens: 0, temperature: '', maxTokens: 0, protocol: '' })
const editModels = ref([])
const editParams = ref({})   // 模型级参数：{模型: {字段名: 值}} → models.json 该服务商的 modelParams
const error = ref('')
const saving = ref(false)

// 思考档位（OpenAI ReasoningEffort）+ 温度档位（兼容旧硬编码，schema 未声明时兜底）
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

// ★ 2026-08-21 按 schema 生成某模型的默认参数键（模板 v-model 需要键存在）
function defaultParamKeys() {
  const keys = {}
  for (const f of props.modelParamFields) {
    if (f.type === 'checkbox') keys[f.name] = false
    else if (f.type === 'number') keys[f.name] = 0
    else keys[f.name] = ''
  }
  return keys
}

function readProviderParams(providerName) {
  // ★ 2026-09-19：模型参数唯一来源 = models.json（服务商条目下的 modelParams）
  const p = providers.value.find(x => x.name === providerName)
  return JSON.parse(JSON.stringify((p && p.modelParams) || {}))
}

async function load() {
  try {
    const d = await api.getModels()
    const mp = d.providerModelParams || {}
    const temps = d.providerTemperatures || {}
    const maxes = d.providerMaxTokens || {}
    providers.value = (d.providers || []).map(name => ({
      name,
      baseURL: (d.providerBaseURLs || {})[name] || '',
      contextMaxTokens: (d.providerContexts || {})[name] || 0, // ★ 服务商级默认上下文窗口
      protocol: (d.providerProtocols || {})[name] || '',        // ★ 2026-09-02 LLM 协议（插件注册配置选项）
      models: (d.models || {})[name] || [],
      // ★ 2026-09-19 生成参数唯一来源 = models.json（服务商级 + 模型级）
      temperature: temps[name] || '',
      maxTokens: maxes[name] || 0,
      modelParams: mp[name] || {},
    }))
    error.value = ''
  } catch (e) {
    error.value = '加载服务商失败: ' + (e.message || e)
  }
}
onMounted(load)

function startAdd() {
  editingName.value = '__new__'
  editForm.value = { name: '', baseURL: '', contextMaxTokens: 0, temperature: '', maxTokens: 0, protocol: '' }
  editModels.value = []
  editParams.value = {}
  error.value = ''
}
function startEdit(p) {
  editingName.value = p.name
  editForm.value = {
    name: p.name, baseURL: p.baseURL, contextMaxTokens: p.contextMaxTokens || 0,
    temperature: p.temperature || '', maxTokens: p.maxTokens || 0, protocol: p.protocol || '',
  }
  editModels.value = [...(p.models || [])]
  const params = readProviderParams(p.name)
  // 为所有模型补默认参数键（模板 v-model 需要键存在；★ 2026-08-21 按 schema 生成）
  const dk = defaultParamKeys()
  for (const m of editModels.value) if (!params[m]) params[m] = { ...dk }
  editParams.value = params
  error.value = ''
}

// 模型列表变化：新模型补参数键、移除模型清理参数
function onModelsChange(list) {
  const params = { ...editParams.value }
  const dk = defaultParamKeys()
  for (const m of list) if (!params[m]) params[m] = { ...dk }
  for (const m of Object.keys(params)) if (!list.includes(m)) delete params[m]
  editParams.value = params
  editModels.value = list
}
function cancelEdit() { editingName.value = ''; error.value = '' }

// 当前列表 → 全量快照 map（供 POST /api/models）
function snapshot() {
  const map = {}
  // ★ 2026-09-19：快照必须带上服务商级参数与模型级参数——否则保存任一服务商会丢掉其余服务商的参数
  for (const p of providers.value) {
    map[p.name] = {
      baseURL: p.baseURL,
      models: p.models,
      contextMaxTokens: p.contextMaxTokens || 0,
      protocol: p.protocol || '',
      temperature: p.temperature || '',
      maxTokens: p.maxTokens || 0,
      modelParams: p.modelParams || {},
    }
  }
  return map
}

async function saveEdit() {
  const oldName = editingName.value        // '' | '__new__' | 被编辑的服务商原名
  const isRename = oldName !== '__new__' && oldName !== ''
  const name = editForm.value.name.trim() || (isRename ? oldName : '')
  if (!name) { error.value = '服务商名称不能为空'; return }
  if (name !== oldName && providers.value.some(p => p.name === name)) {
    error.value = `服务商「${name}」已存在`; return
  }
  saving.value = true
  try {
    // ★ 2026-09-20 改名先走后端：由它迁移 models.json 的键并同步 AI 配置里的 provider 引用。
    //   失败即中止（否则后面的全量保存会把旧键写回去，等于改名没发生）。
    let renamed = null
    if (isRename && name !== oldName) renamed = await api.renameProvider(oldName, name)
    const map = snapshot()
    if (isRename && name !== oldName) delete map[oldName] // 旧键已由后端改名，快照里必须清掉
    map[name] = {
      baseURL: editForm.value.baseURL.trim(),
      models: editModels.value,
      contextMaxTokens: Math.max(0, Number(editForm.value.contextMaxTokens) || 0),
      protocol: (editForm.value.protocol || '').trim(),
      temperature: String(editForm.value.temperature ?? '').trim(),
      maxTokens: Math.max(0, Number(editForm.value.maxTokens) || 0),
      modelParams: cleanModelParams(),
    }
    await api.saveModels(map)
    editingName.value = ''
    await load()
    emit('saved') // AI tab 下拉同步刷新
    if (renamed && renamed.renamed) {
      const n = (renamed.updatedPresets || []).length
      window.$toast(`已改名为「${name}」` + (n ? `，同步更新 ${n} 条 AI 配置` : ''), 'success')
    }
  } catch (e) {
    error.value = (isRename && name !== oldName ? '改名失败: ' : '保存失败: ') + (e.message || e)
  } finally { saving.value = false }
}

// 按 schema 清洗当前编辑的模型参数（仅保留非空项）→ 随服务商一起写入 models.json
// ★ 2026-09-19：不再写 settings.json（生成参数唯一来源 = models.json，服务商配置为准）。
// 字段语义：checkbox 存 true；number 存 >0；select/text 存非空字符串。
function cleanModelParams() {
  const clean = {}
  for (const [m, cfg] of Object.entries(editParams.value)) {
    const c = cfg || {}
    const out = {}
    for (const f of props.modelParamFields) {
      const v = c[f.name]
      if (f.type === 'checkbox') {
        if (v === true) out[f.name] = true
      } else if (f.type === 'number') {
        if (Number(v) > 0) out[f.name] = Number(v)
      } else {
        if (v !== '' && v !== undefined && v !== null) out[f.name] = v
      }
    }
    if (Object.keys(out).length) clean[m] = out
  }
  return clean
}

async function removeProvider(p) {
  // ★ 2026-09-20：删除前列出仍引用该服务商的 AI 配置——配置自带完整连接快照，删掉服务商后
  //   这些配置仍能聊天，但它们在对话面板的模型分组会消失（分组取自 models.json 的服务商模型
  //   列表），用户会以为配置坏了。此处提前告知并给出去向建议。
  let affected = []
  try {
    const d = await api.getAiPresets()
    const presets = (d && d.presets) || {}
    affected = Object.keys(presets).filter(n => (presets[n] || {}).provider === p.name).sort()
  } catch {}
  let msg = `删除服务商「${p.name}」？\n（AI tab 将不再可选该服务商）`
  if (affected.length) {
    msg += `\n\n以下 ${affected.length} 条 AI 配置仍引用它：\n· ${affected.join('\n· ')}`
      + '\n\n删除后它们仍可继续聊天（配置是完整快照），但在对话面板里会失去模型分组；'
      + '建议先在「AI 配置」里把它们改选到其他服务商。'
  }
  if (!window.confirm(msg)) return
  const map = snapshot()
  delete map[p.name]
  try {
    // ★ 2026-09-19：模型参数存于 models.json 的服务商条目内，随服务商一并删除
    await api.saveModels(map)
    await load()
    emit('saved')
  } catch (e) {
    error.value = '删除失败: ' + (e.message || e)
  }
}

function paramsSummary(providerName) {
  const p = providers.value.find(x => x.name === providerName)
  if (!p) return ''
  const n = Object.keys(p.modelParams || {}).length
  const svc = []
  if (p.temperature) svc.push('温度 ' + p.temperature)
  if (p.maxTokens > 0) svc.push('输出上限 ' + p.maxTokens)
  if (n) svc.push('模型参数 ' + n + ' 个')
  return svc.join(' · ')
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
.pm-field > select {
  width: 100%; box-sizing: border-box;
  background: var(--input-bg, #14141f);
  border: 1px solid var(--border-color, #3a3a4a);
  color: var(--text-primary, #eee); border-radius: 6px;
  padding: 7px 10px; font-size: 13px; outline: none; font-family: inherit;
}
.pm-field > select:focus { border-color: var(--accent, #4f8cff); }

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
.pm-protocol {
  font-size: 11px; color: #4cc9a0;
  border: 1px solid rgba(76,201,160,.3);
  border-radius: 4px; padding: 1px 7px; display: inline-block; margin-bottom: 4px;
}
.pm-protocol-hint { font-size: 11px; color: var(--text-secondary, #888); margin-top: 2px; }
.pm-hint { font-size: 11px; color: var(--text-secondary, #888); margin-top: 2px; line-height: 1.5; }
.pm-ctx { font-size: 11px; color: var(--text-secondary, #999); }
.pm-models { display: flex; flex-wrap: wrap; gap: 5px; }
.pm-tag {
  font-size: 11px; padding: 2px 9px; border-radius: 10px;
  background: rgba(79,140,255,.1); color: #8ab4ff;
  border: 1px solid rgba(79,140,255,.22); white-space: nowrap;
}
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
  border-radius: 4px; color: var(--text-primary, #ddd);
  padding: 3px 6px; font-size: 11px;
}
.pm-param-check { display: inline-flex; align-items: center; gap: 4px; font-size: 11px; color: #4cc9a0; cursor: pointer; white-space: nowrap; }
.pm-param-check input { accent-color: #4cc9a0; }
.pm-param-row select { flex: 1.1; min-width: 0; }
.pm-param-row input { flex: 0 0 84px; width: 84px; }
.pm-param-row select:focus, .pm-param-row input:focus { border-color: var(--accent, #4f8cff); }
.pm-params-empty { font-size: 12px; color: var(--text-secondary, #777); padding: 4px 0; }
.pm-params-summary { font-size: 11px; color: var(--text-secondary, #999); }
</style>
