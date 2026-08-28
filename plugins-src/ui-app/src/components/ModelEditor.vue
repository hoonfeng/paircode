<template>
  <div class="me-field">
    <span class="me-label">{{ label }}</span>
    <div class="me-editor">
      <div class="me-input-row">
        <input v-model="input" class="me-input" :placeholder="placeholder"
               @keydown.enter.prevent="addModels" @paste="onPaste" />
        <button class="me-btn" @click="addModels">添加</button>
      </div>
      <div class="me-tags">
        <span v-if="!local.length" class="me-empty">暂无模型——添加后 AI tab 的模型下拉会按服务商显示</span>
        <span v-for="(m, i) in local" :key="m + i" class="me-tag">
          {{ m }}
          <button class="me-x" title="移除" @click="removeAt(i)">×</button>
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

// 模型 tag 编辑器：输入/回车/逗号/粘贴批量添加，点 × 删除。
// props.models 同步外部数组；每次变更 emit('change', 新数组)。
// ★ 2026-08-21 schema 驱动：label/placeholder 由配置注册时声明（providers.modelEditor），
//   前端按声明渲染，无需改组件即可调整文案。
const props = defineProps({
  models: { type: Array, default: () => [] },
  label: { type: String, default: '可用模型（回车或逗号分隔添加；支持整段粘贴）' },
  placeholder: { type: String, default: '输入模型名，回车添加…' },
})
const emit = defineEmits(['change'])

const input = ref('')
const local = ref([...props.models])

watch(() => props.models, v => { local.value = [...v] })

function addModels() {
  const parts = input.value.split(/[\n,，]/).map(s => s.trim()).filter(Boolean)
  let changed = false
  for (const m of parts) {
    if (!local.value.includes(m)) { local.value.push(m); changed = true }
  }
  if (changed) emit('change', [...local.value])
  input.value = ''
}
function onPaste(ev) {
  // 粘贴整段（逗号/换行分隔）自动拆分添加，阻止默认（避免塞进输入框）
  const text = (ev.clipboardData || window.clipboardData).getData('text')
  if (/[,\n，]/.test(text)) {
    ev.preventDefault()
    const parts = text.split(/[\n,，]/).map(s => s.trim()).filter(Boolean)
    let changed = false
    for (const m of parts) {
      if (!local.value.includes(m)) { local.value.push(m); changed = true }
    }
    if (changed) emit('change', [...local.value])
    input.value = ''
  }
}
function removeAt(i) {
  local.value.splice(i, 1)
  emit('change', [...local.value])
}
</script>

<style scoped>
.me-field { display: flex; flex-direction: column; gap: 5px; }
.me-label { font-size: 12px; color: var(--text-secondary, #999); font-weight: 500; }
.me-editor {
  display: flex; flex-direction: column; gap: 8px;
  border: 1px solid var(--border-color, #3a3a4a); border-radius: 8px;
  padding: 10px; background: var(--input-bg, #14141f);
}
.me-input-row { display: flex; gap: 8px; }
.me-input {
  flex: 1; min-width: 0; box-sizing: border-box;
  background: rgba(0,0,0,.2);
  border: 1px solid var(--border-color, #3a3a4a);
  color: var(--text-primary, #eee); border-radius: 6px;
  padding: 6px 10px; font-size: 13px; outline: none; font-family: inherit;
}
.me-input:focus { border-color: var(--accent, #4f8cff); }
.me-btn {
  padding: 6px 12px; border-radius: 6px; font-size: 12px; cursor: pointer;
  border: 1px solid var(--border-color, #444); background: none;
  color: var(--text-primary, #ddd); transition: all .15s; flex-shrink: 0;
}
.me-btn:hover { background: var(--bg-hover, rgba(255,255,255,.06)); }
.me-tags { display: flex; flex-wrap: wrap; gap: 6px; min-height: 24px; }
.me-tag {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 12px; padding: 3px 6px 3px 10px; border-radius: 12px;
  background: rgba(79,140,255,.12); color: #8ab4ff;
  border: 1px solid rgba(79,140,255,.25);
}
.me-x {
  width: 16px; height: 16px; display: inline-flex; align-items: center; justify-content: center;
  border: none; background: none; color: #8ab4ff; cursor: pointer;
  font-size: 13px; line-height: 1; border-radius: 50%;
  padding: 0; opacity: .6;
}
.me-x:hover { opacity: 1; background: rgba(79,140,255,.25); color: #fff; }
.me-empty { font-size: 12px; color: var(--text-secondary, #777); padding: 4px 0; }
</style>
