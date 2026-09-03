<template>
  <!-- 科技风原生 <select> 选择器（2026-09-05 从锚定 popover 改回传统下拉）：
       appearance:none 去默认箭头 + 自定义 chevron + 发光 focus + 暗色下拉（color-scheme）。
       原生下拉由浏览器渲染，不因菜单内部滚动而关闭，稳定可靠；支持 optgroup 分组。 -->
  <div class="sp-wrap">
    <select
      class="sp-select"
      :value="modelValue"
      :title="title"
      :disabled="!items.length"
      :style="selectStyle"
      @change="onChange"
    >
      <option v-if="!items.length" value="" disabled>{{ emptyText }}</option>
      <option v-else-if="placeholder" value="" disabled>{{ placeholder }}</option>
      <template v-for="(g, gi) in groupedItems" :key="gi">
        <optgroup v-if="g.group" :label="g.group">
          <option v-for="it in g.items" :key="it.value" :value="it.value" :title="it.desc">{{ optionText(it) }}</option>
        </optgroup>
        <template v-else>
          <option v-for="it in g.items" :key="it.value" :value="it.value" :title="it.desc">{{ optionText(it) }}</option>
        </template>
      </template>
    </select>
    <SvgIcon name="chevron-down" :size="11" class="sp-chevron" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  items: { type: Array, default: () => [] },       // [{value,label,desc?,group?}]
  title: { type: String, default: '请选择' },
  placeholder: { type: String, default: '请选择…' },
  emptyText: { type: String, default: '暂无可用选项' },
  width: { type: Number, default: 0 },             // >0 时限制触发条最大宽度 px
})
const emit = defineEmits(['update:modelValue', 'change'])

// 按 group 分组（连续同 group 归一组；无 group 项平铺）
const groupedItems = computed(() => {
  const out = []
  for (const it of props.items) {
    const g = it.group || ''
    const last = out[out.length - 1]
    if (g && last && last.group === g) last.items.push(it)
    else out.push({ group: g, items: [it] })
  }
  return out
})

function optionText(it) {
  return it.desc ? `${it.label} · ${it.desc}` : it.label
}

function onChange(e) {
  const v = e.target.value
  emit('update:modelValue', v)
  emit('change', v)
}

const selectStyle = computed(() => (props.width > 0 ? { maxWidth: props.width + 'px' } : {}))
</script>

<style scoped>
/* ── 科技风原生 select：去默认外观 + 渐变底 + 发光 focus ── */
.sp-wrap { position: relative; display: inline-flex; align-items: center; }
.sp-select {
  appearance: none;
  -webkit-appearance: none;
  -moz-appearance: none;
  max-width: 240px;
  padding: 5px 26px 5px 10px;
  border-radius: 8px;
  border: 1px solid var(--border-color, #3a3a4a);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.045), rgba(255, 255, 255, 0.01)),
    var(--bg-tertiary, rgba(0, 0, 0, 0.15));
  color: var(--text-primary, #eee);
  font-size: 12px;
  font-family: inherit;
  line-height: 1.4;
  cursor: pointer;
  outline: none;
  text-overflow: ellipsis;
  white-space: nowrap;
  color-scheme: dark;
  transition: border-color 0.15s, box-shadow 0.15s, background 0.15s;
}
.sp-select:hover { border-color: var(--accent, #4f8cff); }
.sp-select:focus {
  border-color: var(--accent, #4f8cff);
  box-shadow: 0 0 0 3px rgba(79, 140, 255, 0.18);
}
.sp-select:disabled { opacity: 0.55; cursor: not-allowed; }
.sp-chevron {
  position: absolute;
  right: 9px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--accent, #4f8cff);
  pointer-events: none;
}
</style>
