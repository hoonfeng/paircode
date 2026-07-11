<template>
  <teleport to="body">
    <div v-if="visible" class="context-menu-overlay" @mousedown.self="close" @contextmenu.prevent="close">
      <div class="context-menu" :style="menuStyle" @contextmenu.prevent>
        <div v-if="title" class="ctx-title">{{ title }}</div>
        <template v-for="(item, i) in items" :key="i">
          <!-- 分隔线：独立元素，不包裹在 ctx-item 中 -->
          <div v-if="item.separator" class="ctx-separator"></div>
          <!-- 普通菜单项 -->
          <div v-else
               class="ctx-item"
               :class="{ disabled: item.disabled }"
               @click="onItemClick(item)">
            <SvgIcon v-if="item.icon" :name="item.icon" :size="14" class="ctx-icon" />
            <span class="ctx-label">{{ item.label }}</span>
            <span v-if="item.shortcut" class="ctx-shortcut">{{ item.shortcut }}</span>
          </div>
        </template>
      </div>
    </div>
  </teleport>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import SvgIcon from './SvgIcon.vue'

const visible = ref(false)
const position = ref({ x: 0, y: 0 })
const items = ref([])
const title = ref('')
let resolvePromise = null

const menuStyle = computed(() => ({
  left: position.value.x + 'px',
  top: position.value.y + 'px',
}))

function show(opts) {
  return new Promise((resolve) => {
    const { x, y, items: menuItems, title: t } = opts
    let px = x
    let py = y
    // 预计算大致高度避免超出视口
    const itemCount = menuItems.filter(m => !m.separator).length
    const sepCount = menuItems.filter(m => m.separator).length
    const menuHeight = itemCount * 28 + sepCount * 8 + 40
    if (px + 200 > window.innerWidth - 10) px = Math.max(10, window.innerWidth - 210)
    if (py + menuHeight > window.innerHeight - 10) py = Math.max(10, window.innerHeight - menuHeight - 10)
    position.value = { x: px, y: py }
    items.value = menuItems
    title.value = t || ''
    visible.value = true
    resolvePromise = resolve
  })
}

function close(result = null) {
  visible.value = false
  if (resolvePromise) {
    resolvePromise(result)
    resolvePromise = null
  }
}

function onItemClick(item) {
  if (item.disabled) return
  close(item.action || item.label)
}

function handleKeydown(e) {
  if (e.key === 'Escape' && visible.value) close(null)
}

onMounted(() => document.addEventListener('keydown', handleKeydown))
onUnmounted(() => document.removeEventListener('keydown', handleKeydown))

defineExpose({ show, close })
</script>

<style scoped>
.context-menu-overlay {
  position: fixed;
  inset: 0;
  z-index: 10000;
  background: transparent;
}
.context-menu {
  position: absolute;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  box-shadow: var(--shadow-md);
  padding: 4px 0;
  min-width: 180px;
  max-width: 340px;
  font-family: var(--font-ui);
  font-size: var(--font-size-sm);
}
.ctx-title {
  padding: 4px 12px;
  font-size: 11px;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border-color);
  margin-bottom: 2px;
}
.ctx-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 12px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-primary);
  white-space: nowrap;
  min-height: 26px;
}
.ctx-item:hover:not(.disabled) {
  background: var(--accent);
  color: var(--status-text);
}
.ctx-item.disabled {
  opacity: 0.4;
  cursor: default;
}
.ctx-icon {
  flex-shrink: 0;
  color: var(--text-muted);
  width: 16px;
  text-align: center;
}
.ctx-item:hover:not(.disabled) .ctx-icon {
  color: rgba(255,255,255,0.7);
}
.ctx-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ctx-shortcut {
  font-size: 11px;
  color: var(--text-muted);
  margin-left: auto;
  padding-left: 16px;
}
.ctx-item:hover:not(.disabled) .ctx-shortcut {
  color: rgba(255,255,255,0.7);
}

/* ── 独立分隔线（不再包裹在 ctx-item 中）── */
.ctx-separator {
  height: 1px;
  background: var(--border-color);
  margin: 4px 8px;
  padding: 0;
  min-height: 1px;
  flex-shrink: 0;
}
</style>
