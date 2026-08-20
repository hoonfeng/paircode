<template>
  <div class="mgm-manager">
    <!-- 工具栏 -->
    <div class="mgm-toolbar">
      <span class="mgm-count">{{ groups.length }} 个模型组</span>
      <button class="mgm-btn mgm-primary" @click="startAdd">+ 新增模型组</button>
    </div>

    <!-- 新增/编辑表单（工具栏下方展开，紧邻按钮不跳动） -->
    <div v-if="editing" class="mgm-edit">
      <div class="mgm-edit-title">{{ editingIsNew ? '新增模型组' : '编辑模型组：' + editing.oldName }}</div>
      <div class="mgm-field">
        <span class="mgm-field-label">模型组名称（自定义，如「主力」「备用」）</span>
        <input v-model="editName" placeholder="输入模型组名称…" />
      </div>
      <div class="mgm-field">
        <span class="mgm-field-label">组内实例（勾选加入；实例 = 服务商连接：Key + BaseURL + 模型列表）</span>
        <div v-if="allInstances.length" class="mgm-instance-list">
          <label v-for="inst in allInstances" :key="inst.name" class="mgm-instance-item"
                 :class="{ 'mgm-instance-checked': editInstances.includes(inst.name) }">
            <input type="checkbox" :value="inst.name" v-model="editInstances" />
            <span class="mgm-inst-name">{{ inst.name }}</span>
            <span class="mgm-inst-meta">{{ inst.models.length }} 模型{{ inst.apiKey ? ' · Key ✓' : ' · 无 Key' }}</span>
          </label>
        </div>
        <div v-else class="mgm-empty">暂无实例（服务商）。请先在下方「实例（服务商）」面板添加服务商，再回来挂载到模型组。</div>
      </div>
      <div class="mgm-field" v-if="editInstances.length">
        <span class="mgm-field-label">组内模型汇总（对话面板选模型组后可见）</span>
        <div class="mgm-model-summary">
          <span v-for="m in groupedModels" :key="m" class="mgm-tag">{{ m }}</span>
        </div>
      </div>
      <div class="mgm-edit-actions">
        <button class="mgm-btn mgm-primary" :disabled="saving" @click="saveEdit">
          {{ saving ? '保存中…' : '保存模型组' }}
        </button>
        <button class="mgm-btn" @click="cancelEdit">取消</button>
      </div>
    </div>

    <!-- 模型组卡片列表 -->
    <div v-if="groups.length" class="mgm-cards">
      <div v-for="g in groups" :key="g.name" class="mgm-card">
        <div class="mgm-card-head">
          <span class="mgm-name" :title="g.name">{{ g.name }}</span>
          <div class="mgm-ops">
            <button class="mgm-btn mgm-small" @click="startEdit(g)">编辑</button>
            <button class="mgm-btn mgm-small mgm-danger" @click="removeGroup(g)">删除</button>
          </div>
        </div>
        <div class="mgm-instances">
          <span v-if="!g.instances.length" class="mgm-none">（未挂载实例）</span>
          <span v-for="inst in g.instances" :key="inst" class="mgm-inst-tag">{{ inst }}</span>
        </div>
        <div class="mgm-models">
          <span v-if="!g.modelCount" class="mgm-none">（无模型）</span>
          <template v-else>
            <span class="mgm-model-label">{{ g.modelCount }} 个模型：</span>
            <span v-for="m in g.models" :key="m" class="mgm-tag">{{ m }}</span>
          </template>
        </div>
      </div>
    </div>
    <div v-else-if="!editing" class="mgm-empty">暂无模型组。新增一个模型组并挂载实例（服务商），对话面板即可按「模型组 → 模型」选择，无需再选服务商。</div>

    <div v-if="error" class="mgm-error">{{ error }}</div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '../api.js'

// 模型组管理面板：维护 config/model-groups.json（GET/PUT /api/model-groups）。
// 模型组 = 用户命名的实例集合；实例 = models.json 服务商条目（Key + BaseURL + 模型列表）。
// 对话面板：选模型组 → 选模型（组内模型）→ 自动匹配所属实例 Key，无需选服务商。
const emit = defineEmits(['saved'])

const groups = ref([])        // [{ name, instances, models, modelCount }]
const allInstances = ref([])  // [{ name, models, apiKey }]（来自 /api/models）
const editing = ref(null)     // null=不编辑；{ oldName, isNew }
const editName = ref('')
const editInstances = ref([])
const error = ref('')
const saving = ref(false)

const editingIsNew = computed(() => !!(editing.value && editing.value.isNew))
const groupedModels = computed(() => {
  const instSet = new Set(editInstances.value)
  const out = []
  for (const inst of allInstances.value) {
    if (instSet.has(inst.name)) {
      for (const m of inst.models) if (!out.includes(m)) out.push(m)
    }
  }
  return out
})

async function load() {
  try {
    const [g, m] = await Promise.all([api.getModelGroups(), api.getModels()])
    const instByName = {}
    for (const p of (m.providers || [])) {
      instByName[p] = { name: p, models: (m.models || {})[p] || [], apiKey: (m.providerKeys || {})[p] || '' }
    }
    allInstances.value = (m.providers || []).map(p => instByName[p]).filter(Boolean)
    groups.value = Object.entries(g.groups || {}).map(([name, insts]) => {
      const models = []
      for (const inst of insts) {
        const im = instByName[inst] && instByName[inst].models
        if (im) for (const mm of im) if (!models.includes(mm)) models.push(mm)
      }
      return { name, instances: insts || [], models, modelCount: models.length }
    })
    error.value = ''
  } catch (e) {
    error.value = '加载模型组失败: ' + (e.message || e)
  }
}
onMounted(load)

function startAdd() {
  editing.value = { oldName: '', isNew: true }
  editName.value = ''
  editInstances.value = []
  error.value = ''
}
function startEdit(g) {
  editing.value = { oldName: g.name, isNew: false }
  editName.value = g.name
  editInstances.value = [...(g.instances || [])]
  error.value = ''
}
function cancelEdit() { editing.value = null; error.value = '' }

async function saveEdit() {
  const name = editName.value.trim()
  if (!name) { error.value = '模型组名称不能为空'; return }
  const map = {}
  for (const g of groups.value) map[g.name] = g.instances
  if (editing.value.oldName && editing.value.oldName !== name && map[name] !== undefined) {
    error.value = `模型组「${name}」已存在`
    return
  }
  if (editing.value.oldName && editing.value.oldName !== name) delete map[editing.value.oldName]
  map[name] = [...editInstances.value]
  saving.value = true
  try {
    await api.saveModelGroups(map)
    editing.value = null
    await load()
    emit('saved')
  } catch (e) {
    error.value = '保存失败: ' + (e.message || e)
  } finally { saving.value = false }
}

async function removeGroup(g) {
  if (!window.confirm(`删除模型组「${g.name}」？\n（实例本身不会被删除，仍可在服务商面板维护）`)) return
  const map = {}
  for (const gg of groups.value) if (gg.name !== g.name) map[gg.name] = gg.instances
  try {
    await api.saveModelGroups(map)
    await load()
    emit('saved')
  } catch (e) {
    error.value = '删除失败: ' + (e.message || e)
  }
}
</script>

<style scoped>
.mgm-manager { display: flex; flex-direction: column; gap: 14px; }
.mgm-toolbar { display: flex; align-items: center; justify-content: space-between; }
.mgm-count { font-size: 12px; color: var(--text-secondary, #999); }
.mgm-btn {
  padding: 6px 14px; border-radius: 7px; font-size: 13px; cursor: pointer;
  border: 1px solid var(--border-color, #444); background: none;
  color: var(--text-primary, #ddd); transition: all .15s;
}
.mgm-btn:hover { background: var(--bg-hover, rgba(255,255,255,.06)); }
.mgm-btn:disabled { opacity: .5; cursor: not-allowed; }
.mgm-btn.mgm-primary {
  background: var(--accent, #4f8cff); color: #fff; border-color: var(--accent, #4f8cff); font-weight: 600;
}
.mgm-btn.mgm-primary:hover { filter: brightness(1.12); background: var(--accent, #4f8cff); }
.mgm-btn.mgm-small { padding: 3px 10px; font-size: 12px; }
.mgm-btn.mgm-danger { color: #e06c6c; border-color: rgba(224,108,108,.4); }
.mgm-btn.mgm-danger:hover { background: rgba(224,108,108,.12); }

.mgm-edit {
  display: flex; flex-direction: column; gap: 12px;
  padding: 16px; border: 1px solid var(--border-color, #3a3a4a);
  border-radius: 10px; background: var(--bg-tertiary, rgba(0,0,0,.15));
}
.mgm-edit-title {
  font-size: 14px; font-weight: 600; color: var(--text-primary, #eee);
  padding-bottom: 10px; border-bottom: 1px solid var(--border-color, #333);
}
.mgm-field { display: flex; flex-direction: column; gap: 6px; }
.mgm-field-label { font-size: 12px; color: var(--text-secondary, #999); font-weight: 500; }
.mgm-field input[type="text"] {
  width: 100%; box-sizing: border-box;
  background: var(--input-bg, #14141f);
  border: 1px solid var(--border-color, #3a3a4a);
  color: var(--text-primary, #eee); border-radius: 6px;
  padding: 7px 10px; font-size: 13px; outline: none; font-family: inherit;
}
.mgm-field input[type="text"]:focus { border-color: var(--accent, #4f8cff); }
.mgm-edit-actions { display: flex; gap: 8px; justify-content: flex-end; padding-top: 4px; }

.mgm-instance-list { display: flex; flex-direction: column; gap: 4px; max-height: 220px; overflow-y: auto; }
.mgm-instance-item {
  display: flex; align-items: center; gap: 8px;
  padding: 6px 10px; border: 1px solid var(--border-color, #333);
  border-radius: 6px; cursor: pointer; background: var(--bg-tertiary, rgba(0,0,0,.08));
  transition: border-color .15s, background .15s;
}
.mgm-instance-item:hover { border-color: var(--border-color, #4a4a5a); }
.mgm-instance-checked { border-color: rgba(79,140,255,.4); background: rgba(79,140,255,.08); }
.mgm-instance-item input { accent-color: var(--accent, #4f8cff); }
.mgm-inst-name { font-size: 13px; font-weight: 500; color: var(--text-primary, #eee); }
.mgm-inst-meta { font-size: 11px; color: var(--text-secondary, #999); }

.mgm-cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 10px; }
.mgm-card {
  display: flex; flex-direction: column; gap: 8px;
  padding: 12px 14px; border: 1px solid var(--border-color, #333);
  border-radius: 9px; background: var(--bg-tertiary, rgba(0,0,0,.12));
  transition: border-color .15s, background .15s;
}
.mgm-card:hover { border-color: var(--border-color, #4a4a5a); background: var(--bg-tertiary, rgba(0,0,0,.18)); }
.mgm-card-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.mgm-name { font-weight: 600; color: var(--text-primary, #eee); font-size: 13px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.mgm-ops { display: flex; gap: 6px; flex-shrink: 0; }
.mgm-instances { display: flex; flex-wrap: wrap; gap: 5px; }
.mgm-inst-tag {
  font-size: 11px; padding: 2px 9px; border-radius: 10px;
  background: rgba(126,203,126,.1); color: #7ecb7e;
  border: 1px solid rgba(126,203,126,.22); white-space: nowrap;
}
.mgm-models { display: flex; flex-wrap: wrap; gap: 5px; align-items: center; }
.mgm-model-label { font-size: 11px; color: var(--text-secondary, #999); }
.mgm-tag {
  font-size: 11px; padding: 2px 9px; border-radius: 10px;
  background: rgba(79,140,255,.1); color: #8ab4ff;
  border: 1px solid rgba(79,140,255,.22); white-space: nowrap;
}
.mgm-none { color: var(--text-secondary, #777); font-size: 12px; }
.mgm-empty { color: var(--text-secondary, #888); text-align: center; padding: 24px 0; font-size: 13px; line-height: 1.8; }
.mgm-error {
  color: #e06c6c; font-size: 12px; padding: 8px 10px;
  border: 1px solid rgba(224,108,108,.3); border-radius: 6px;
  background: rgba(224,108,108,.08);
}
.mgm-model-summary { display: flex; flex-wrap: wrap; gap: 5px; }
</style>
