<template>
  <!-- 锚定 popover 选择器（2026-09-05 重构，替代 bottom-sheet）：
       触发条 → 在按钮附近弹出小菜单（分组标题 + 列表项 + ✓），
       点外部/Esc/滚动关闭，空间不足自动向上翻转；参考 deepseek 聊天框模型选择器。 -->
  <div class="sp-root" ref="rootRef">
    <button class="sp-trigger" :class="{ 'sp-trigger-open': open }" @click.stop="toggle" :title="title">
      <span class="sp-value" :class="{ 'sp-muted': !currentLabel }">{{ currentLabel || placeholder }}</span>
      <SvgIcon name="chevron-down" :size="11" class="sp-chevron" :class="{ up: open }" />
    </button>
  </div>

  <Teleport to="body">
    <!-- 透明遮罩：捕获外部点击关闭，不遮挡视觉（z-index 低于弹层） -->
    <div v-if="open" class="sp-pop-mask" @click="close"></div>
    <!-- 锚定 popover（position:fixed + 动态 top/left） -->
    <Transition name="sp-pop-fade">
      <div v-if="open" class="sp-pop" :style="popStyle" ref="popRef" @click.stop>
        <div class="sp-pop-body">
          <template v-for="(item, i) in items" :key="item.value">
            <div v-if="item.group && item.group !== (items[i - 1] || {}).group" class="sp-group">{{ item.group }}</div>
            <div class="sp-item" :class="{ active: item.value === modelValue }" @click="pick(item)">
              <div class="sp-item-main">
                <span class="sp-item-label">{{ item.label }}</span>
                <span v-if="item.desc" class="sp-item-desc">{{ item.desc }}</span>
              </div>
              <SvgIcon v-if="item.value === modelValue" name="check" :size="14" class="sp-check" />
            </div>
          </template>
          <div v-if="!items.length" class="sp-empty">{{ emptyText }}</div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  items: { type: Array, default: () => [] },       // [{value,label,desc?,group?}]
  title: { type: String, default: '请选择' },
  placeholder: { type: String, default: '点击选择…' },
  emptyText: { type: String, default: '暂无可用选项' },
  width: { type: Number, default: 260 },           // popover 目标宽度 px
  align: { type: String, default: 'start' },       // 'start' 左对齐 | 'end' 右对齐
})
const emit = defineEmits(['update:modelValue', 'change'])

const open = ref(false)
const rootRef = ref(null)
const popRef = ref(null)
const popStyle = ref({})

const currentLabel = computed(() => {
  const it = props.items.find(x => x.value === props.modelValue)
  return it ? it.label : ''
})

function toggle() {
  open.value = !open.value
  if (open.value) nextTick(() => position())
}

function close() { open.value = false }

function pick(item) {
  emit('update:modelValue', item.value)
  emit('change', item.value)
  close()
}

// 锚定定位：优先向下弹出，底部空间不足向上翻转；水平越界则收回到视口内
function position() {
  const r = rootRef.value && rootRef.value.getBoundingClientRect()
  if (!r) return
  const pop = popRef.value
  const pw = pop ? pop.offsetWidth : props.width
  const ph = pop ? pop.offsetHeight : 320
  const gap = 6
  const pad = 8

  let left = props.align === 'end' ? r.right - pw : r.left
  if (left + pw > window.innerWidth - pad) left = Math.max(pad, window.innerWidth - pw - pad)
  if (left < pad) left = pad

  let top = r.bottom + gap
  if (top + ph > window.innerHeight - pad) {
    // 向上翻转
    top = r.top - ph - gap
    if (top < pad) top = pad
  }
  popStyle.value = { left: left + 'px', top: top + 'px' }
}

function onDocClick(e) {
  // 点外部关闭（遮罩层已兜底，这里再兜一次）
  if (open.value && rootRef.value && !rootRef.value.contains(e.target)) close()
}
function onKeydown(e) {
  if (e.key === 'Escape') close()
}
function onScroll() { close() }  // 滚动时关闭，避免弹层与按钮错位漂移
function onResize() { close() }

function bindGlobal() {
  document.addEventListener('click', onDocClick)
  document.addEventListener('keydown', onKeydown)
  window.addEventListener('scroll', onScroll, true)
  window.addEventListener('resize', onResize)
}
function unbindGlobal() {
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onKeydown)
  window.removeEventListener('scroll', onScroll, true)
  window.removeEventListener('resize', onResize)
}

// 打开时绑定全局监听，关闭时解绑
watch(open, (v) => { v ? bindGlobal() : unbindGlobal() })
onBeforeUnmount(unbindGlobal)
</script>

<style scoped>
/* ── 触发条：小巧胶囊芯片（融合设计） ── */
.sp-root { display: inline-flex; }
.sp-trigger {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  max-width: 220px;
  padding: 4px 10px;
  border-radius: 999px;
  border: 1px solid var(--border-color, #3a3a4a);
  background: var(--bg-tertiary, rgba(0, 0, 0, 0.12));
  color: var(--text-primary, #eee);
  font-size: 12px;
  font-family: inherit;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
  line-height: 1.4;
}
.sp-trigger:hover { border-color: var(--accent, #4f8cff); background: var(--bg-hover, rgba(255, 255, 255, 0.06)); }
.sp-trigger-open { border-color: var(--accent, #4f8cff); background: var(--bg-hover, rgba(255,255,255,0.06)); }
.sp-value {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sp-muted { color: var(--text-muted, #888); }
.sp-chevron { flex-shrink: 0; color: var(--text-muted, #888); transition: transform 0.2s; }
.sp-chevron.up { transform: rotate(180deg); }

/* ── 透明遮罩（捕获外部点击，不遮挡视觉） ── */
.sp-pop-mask {
  position: fixed;
  inset: 0;
  z-index: 10040;
  background: transparent;
}

/* ── 锚定 popover 小菜单 ── */
.sp-pop {
  position: fixed;
  z-index: 10050;
  width: 260px;
  max-width: calc(100vw - 16px);
  max-height: 320px;
  display: flex;
  flex-direction: column;
  background: var(--bg-secondary, #1c1c28);
  border: 1px solid var(--border-color, #3a3a4a);
  border-radius: 12px;
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.4);
  overflow: hidden;
}
.sp-pop-body { overflow-y: auto; padding: 5px; }
.sp-group {
  font-size: 10px;
  font-weight: 600;
  color: var(--text-muted, #888);
  text-transform: uppercase;
  letter-spacing: 0.4px;
  padding: 8px 10px 3px;
  flex-shrink: 0;
}
.sp-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 7px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.12s;
}
.sp-item:hover { background: var(--bg-hover, rgba(255, 255, 255, 0.06)); }
.sp-item.active { background: color-mix(in srgb, var(--accent, #4f8cff) 14%, transparent); }
.sp-item-main { display: flex; flex-direction: column; gap: 1px; min-width: 0; }
.sp-item-label { font-size: 12.5px; color: var(--text-primary, #eee); }
.sp-item.active .sp-item-label { color: var(--accent-light, #8ab4ff); }
.sp-item-desc { font-size: 10.5px; color: var(--text-muted, #888); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sp-check { color: var(--accent, #4f8cff); flex-shrink: 0; }
.sp-empty { text-align: center; color: var(--text-muted, #888); font-size: 12px; padding: 18px 0; }

/* 弹层过渡（缩放淡入） */
.sp-pop-fade-enter-active, .sp-pop-fade-leave-active { transition: opacity 0.14s; }
.sp-pop-fade-enter-active .sp-pop, .sp-pop-fade-leave-active .sp-pop { transition: transform 0.14s ease-out, opacity 0.14s; }
.sp-pop-fade-enter-from, .sp-pop-fade-leave-to { opacity: 0; }
.sp-pop-fade-enter-from .sp-pop, .sp-pop-fade-leave-to .sp-pop { transform: translateY(-6px) scale(0.98); }
</style>
