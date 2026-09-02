<template>
  <!-- 移动端风格选择器（bottom-sheet 弹层）：
       触发条 → 底部弹层滑入：分组标题 + 列表项（label/desc/勾选）；替代传统 <select> -->
  <button class="sp-trigger" :class="{ 'sp-trigger-open': open }" @click="open = true" :title="title">
    <span class="sp-value" :class="{ 'sp-muted': !currentLabel }">{{ currentLabel || placeholder }}</span>
    <SvgIcon name="chevron-down" :size="11" class="sp-chevron" :class="{ up: open }" />
  </button>

  <Teleport to="body">
    <Transition name="sp-fade">
      <div v-if="open" class="sp-overlay" @click.self="close">
        <div class="sp-sheet" @click.stop>
          <div class="sp-grabber"></div>
          <div class="sp-head">
            <span class="sp-title">{{ title }}</span>
            <button class="sp-cancel" @click="close">取消</button>
          </div>
          <div class="sp-body">
            <template v-for="(item, i) in items" :key="item.value">
              <div v-if="item.group && item.group !== (items[i - 1] || {}).group" class="sp-group">{{ item.group }}</div>
              <div class="sp-item" :class="{ active: item.value === modelValue }" @click="pick(item)">
                <div class="sp-item-main">
                  <span class="sp-item-label">{{ item.label }}</span>
                  <span v-if="item.desc" class="sp-item-desc">{{ item.desc }}</span>
                </div>
                <SvgIcon v-if="item.value === modelValue" name="check" :size="15" class="sp-check" />
              </div>
            </template>
            <div v-if="!items.length" class="sp-empty">{{ emptyText }}</div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, ref } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  items: { type: Array, default: () => [] },       // [{value,label,desc?,group?}]
  title: { type: String, default: '请选择' },
  placeholder: { type: String, default: '点击选择…' },
  emptyText: { type: String, default: '暂无可用选项' },
})
const emit = defineEmits(['update:modelValue', 'change'])

const open = ref(false)
const currentLabel = computed(() => {
  const it = props.items.find(x => x.value === props.modelValue)
  return it ? it.label : ''
})
function pick(item) {
  emit('update:modelValue', item.value)
  emit('change', item.value)
  close()
}
function close() { open.value = false }
</script>

<style scoped>
/* ── 触发条：胶囊样式（融合设计；点击展开底部弹层） ── */
.sp-trigger {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  max-width: 220px;
  padding: 4px 9px;
  border-radius: 999px;
  border: 1px solid var(--border-color, #3a3a4a);
  background: var(--bg-tertiary, rgba(0, 0, 0, 0.12));
  color: var(--text-primary, #eee);
  font-size: 12px;
  font-family: inherit;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.sp-trigger:hover { border-color: var(--accent, #4f8cff); background: var(--bg-hover, rgba(255,255,255,0.06)); }
.sp-trigger-open { border-color: var(--accent, #4f8cff); }
.sp-value {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sp-muted { color: var(--text-muted, #888); }
.sp-chevron { flex-shrink: 0; color: var(--text-muted, #888); transition: transform 0.2s; }
.sp-chevron.up { transform: rotate(180deg); }

/* ── 底部弹层（移动端 sheet） ── */
.sp-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  z-index: 10050;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}
.sp-sheet {
  width: min(560px, 96vw);
  max-height: 72vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-secondary, #1c1c28);
  border: 1px solid var(--border-color, #3a3a4a);
  border-bottom: none;
  border-radius: 18px 18px 0 0;
  box-shadow: 0 -10px 40px rgba(0, 0, 0, 0.4);
  animation: sp-sheet-in 0.22s ease-out;
}
@keyframes sp-sheet-in {
  from { transform: translateY(24px); opacity: 0.6; }
  to { transform: translateY(0); opacity: 1; }
}
.sp-grabber {
  width: 36px;
  height: 4px;
  border-radius: 2px;
  background: var(--border-color, #3a3a4a);
  margin: 8px auto 2px;
  flex-shrink: 0;
}
.sp-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px 10px;
  border-bottom: 1px solid var(--border-color, #333);
  flex-shrink: 0;
}
.sp-title { font-size: 14px; font-weight: 600; color: var(--text-primary, #eee); }
.sp-cancel {
  background: none;
  border: none;
  color: var(--accent, #4f8cff);
  font-size: 13px;
  cursor: pointer;
  font-family: inherit;
  padding: 2px 4px;
}
.sp-body { overflow-y: auto; padding: 4px 8px 14px; }
.sp-group {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted, #888);
  text-transform: uppercase;
  letter-spacing: 0.4px;
  padding: 10px 10px 4px;
}
.sp-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 9px 10px;
  border-radius: 10px;
  cursor: pointer;
  transition: background 0.12s;
}
.sp-item:hover { background: var(--bg-hover, rgba(255, 255, 255, 0.06)); }
.sp-item.active { background: color-mix(in srgb, var(--accent, #4f8cff) 14%, transparent); }
.sp-item-main { display: flex; flex-direction: column; gap: 1px; min-width: 0; }
.sp-item-label { font-size: 13px; color: var(--text-primary, #eee); }
.sp-item.active .sp-item-label { color: var(--accent-light, #8ab4ff); }
.sp-item-desc { font-size: 11px; color: var(--text-muted, #888); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sp-check { color: var(--accent, #4f8cff); flex-shrink: 0; }
.sp-empty { text-align: center; color: var(--text-muted, #888); font-size: 12px; padding: 24px 0; }

/* 弹层过渡 */
.sp-fade-enter-active, .sp-fade-leave-active { transition: opacity 0.18s; }
.sp-fade-enter-active .sp-sheet, .sp-fade-leave-active .sp-sheet { transition: transform 0.18s ease-out; }
.sp-fade-enter-from, .sp-fade-leave-to { opacity: 0; }
.sp-fade-enter-from .sp-sheet, .sp-fade-leave-to .sp-sheet { transform: translateY(24px); }
</style>
