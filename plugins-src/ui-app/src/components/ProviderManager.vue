<template>
  <div class="provider-manager">
    <!-- 编辑表单（新增/编辑共用） -->
    <div v-if="editing" class="pm-edit">
      <div class="pm-edit-title">{{ editMode === 'add' ? '新增服务商' : '编辑服务商' }}</div>

      <div class="pm-field">
        <span class="pm-field-label">服务商名称</span>
        <input v-model="editForm.name" placeholder="如 deepseek" :disabled="editMode === 'edit'" />
      </div>
      <div class="pm-field">
        <span class="pm-field-label">Base URL</span>
        <input v-model="editForm.baseURL" placeholder="https://api.deepseek.com/v1" />
      </div>

      <!-- 模型编辑器：输入/回车/粘贴添加，点 × 删除 -->
      <div class="pm-field">
        <span class="pm-field-label">可用模型（回车或逗号分隔添加；支持整段粘贴）</span>
        <div class="pm-models-editor">
          <div class="pm-me-input-row">
            <input v-model="modelInput" class="pm-me-input" placeholder="输入模型名，回车添加…"
                   @keydown.enter.prevent="addModelFromInput" @paste="onPasteModels" />
            <button class="pm-btn pm-small" @click="addModelFromInput">添加</button>
          </div>
          <div class="pm-me-tags">
            <span v-if="!editModels.length" class="pm-me-empty">暂无模型——添加后 AI tab 的模型下拉会按服务商显示</span>
            <span v-for="(m, i) in editModels" :key="m + i" class="pm-me-tag">
              {{ m }}
              <button class="pm-me-x" title="移除" @click="editModels.splice(i, 1)">×</button>
            </span>
          </div>
        </div>
      </div>

      <div class="pm-edit-actions">
        <button class="pm-btn pm-primary" :disabled="saving" @click="saveEdit">
          {{ saving ? '保存中…' : '保存服务商' }}
        </button>
        <button class="pm-btn" @click="cancelEdit">取消</button>
      </div>
    </div>

    <!-- 服务商卡片列表 -->
    <div class="pm-toolbar">
      <span class="pm-count">{{ providers.length }} 个服务商</span>
      <button class="pm-btn pm-primary" @click="startAdd">+ 新增服务商</button>
    </div>

    <div v-if="providers.length" class="pm-cards">
      <div v-for="p in providers" :key="p.name" class="pm-card">
        <div class="pm-card-head">
          <span class="pm-name" :title="p.name">{{ p.name }}</span>
          <div class="pm-ops">
            <button class="pm-btn pm-small" @click="startEdit(p)">编辑</button>
            <button class="pm-btn pm-small pm-danger" @click="removeProvider(p)">删除</button>
          </div>
        </div>
        <div class="pm-url" :title="p.baseURL">{{ p.baseURL || '未配置 Base URL' }}</div>
        <div class="pm-models">
          <span v-if="!p.models.length" class="pm-none">（未配置模型）</span>
          <span v-for="m in p.models" :key="m" class="pm-tag">{{ m }}</span>
        </div>
      </div>
    </div>
    <div v-else-if="!editing" class="pm-empty">暂无服务商，点「+ 新增服务商」添加</div>

    <div v-if="error" class="pm-error">{{ error }}</div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api.js'

// 服务商管理面板：维护 config/models.json（GET/POST /api/models）。
// 保存后 emit('saved') 通知父组件刷新 AI tab 的 provider/模型下拉。
const emit = defineEmits(['saved'])

const providers = ref([])
const editing = ref(false)
const editMode = ref('add') // 'add' | 'edit'
const editForm = ref({ name: '', baseURL: '' })
const editModels = ref([])
const modelInput = ref('')
const error = ref('')
const saving = ref(false)

async function load() {
  try {
    const d = await api.getModels()
    providers.value = (d.providers || []).map(name => ({
      name,
      baseURL: (d.providerBaseURLs || {})[name] || '',
      models: (d.models || {})[name] || [],
    }))
    error.value = ''
  } catch (e) {
    error.value = '加载服务商失败: ' + (e.message || e)
  }
}
onMounted(load)

function startAdd() {
  editMode.value = 'add'
  editForm.value = { name: '', baseURL: '' }
  editModels.value = []
  modelInput.value = ''
  error.value = ''
  editing.value = true
}
function startEdit(p) {
  editMode.value = 'edit'
  editForm.value = { name: p.name, baseURL: p.baseURL }
  editModels.value = [...(p.models || [])]
  modelInput.value = ''
  error.value = ''
  editing.value = true
}
function cancelEdit() { editing.value = false; error.value = '' }

// ─── 模型编辑器 ───
function addModelFromInput() {
  const parts = modelInput.value.split(/[\n,，]/).map(s => s.trim()).filter(Boolean)
  for (const m of parts) {
    if (!editModels.value.includes(m)) editModels.value.push(m)
  }
  modelInput.value = ''
}
function onPasteModels(ev) {
  // 粘贴整段（逗号/换行分隔）自动拆分添加，阻止默认（避免塞进输入框）
  const text = (ev.clipboardData || window.clipboardData).getData('text')
  if (/[,\n，]/.test(text)) {
    ev.preventDefault()
    const parts = text.split(/[\n,，]/).map(s => s.trim()).filter(Boolean)
    for (const m of parts) {
      if (!editModels.value.includes(m)) editModels.value.push(m)
    }
    modelInput.value = ''
  }
}

// 当前列表 → 全量快照 map（供 POST /api/models）
function snapshot() {
  const map = {}
  for (const p of providers.value) map[p.name] = { baseURL: p.baseURL, models: p.models }
  return map
}

async function saveEdit() {
  const name = editForm.value.name.trim()
  if (!name) { error.value = '服务商名称不能为空'; return }
  const map = snapshot()
  if (editMode.value === 'add' && map[name]) { error.value = `服务商「${name}」已存在`; return }
  map[name] = { baseURL: editForm.value.baseURL.trim(), models: editModels.value }
  saving.value = true
  try {
    await api.saveModels(map)
    editing.value = false
    await load()
    emit('saved') // AI tab 下拉同步刷新
  } catch (e) {
    error.value = '保存失败: ' + (e.message || e)
  } finally { saving.value = false }
}

async function removeProvider(p) {
  if (!window.confirm(`删除服务商「${p.name}」？\n（AI tab 将不再可选该服务商）`)) return
  const map = snapshot()
  delete map[p.name]
  try {
    await api.saveModels(map)
    await load()
    emit('saved')
  } catch (e) {
    error.value = '删除失败: ' + (e.message || e)
  }
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

/* ─── 编辑表单（卡片式分组）─── */
.pm-edit {
  display: flex; flex-direction: column; gap: 12px;
  padding: 16px; border: 1px solid var(--border-color, #3a3a4a);
  border-radius: 10px; background: var(--bg-tertiary, rgba(0,0,0,.15));
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

/* ─── 模型编辑器 ─── */
.pm-models-editor {
  display: flex; flex-direction: column; gap: 8px;
  border: 1px solid var(--border-color, #3a3a4a); border-radius: 8px;
  padding: 10px; background: var(--input-bg, #14141f);
}
.pm-me-input-row { display: flex; gap: 8px; }
.pm-me-input {
  flex: 1; min-width: 0; box-sizing: border-box;
  background: rgba(0,0,0,.2);
  border: 1px solid var(--border-color, #3a3a4a);
  color: var(--text-primary, #eee); border-radius: 6px;
  padding: 6px 10px; font-size: 13px; outline: none; font-family: inherit;
}
.pm-me-input:focus { border-color: var(--accent, #4f8cff); }
.pm-me-tags { display: flex; flex-wrap: wrap; gap: 6px; min-height: 24px; }
.pm-me-tag {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 12px; padding: 3px 6px 3px 10px; border-radius: 12px;
  background: rgba(79,140,255,.12); color: #8ab4ff;
  border: 1px solid rgba(79,140,255,.25);
}
.pm-me-x {
  width: 16px; height: 16px; display: inline-flex; align-items: center; justify-content: center;
  border: none; background: none; color: #8ab4ff; cursor: pointer;
  font-size: 13px; line-height: 1; border-radius: 50%;
  padding: 0; opacity: .6;
}
.pm-me-x:hover { opacity: 1; background: rgba(79,140,255,.25); color: #fff; }
.pm-me-empty { font-size: 12px; color: var(--text-secondary, #777); padding: 4px 0; }

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
</style>
