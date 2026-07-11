<template>
  <div class="sab-wrapper" :class="[`sab-type-${type}`, `sab-mode-${mode}`]">
    <template v-if="mode === 'minimal'">
      <div class="sab-minimal" @click="expandFromMinimal" title="展开查看（仅工作模式，内容已累积）">
        <span class="sab-dot"></span>
        <span class="sab-minimal-hint">▸</span>
        <span v-if="hasContent" class="sab-minimal-pulse"></span>
      </div>
    </template>

    <template v-else>
      <div class="sab-header" @click="toggleCollapse">
        <span class="sab-dot"></span>
        <span class="sab-chevron">{{ mode === 'expanded' ? '▾' : '▸' }}</span>
        <SvgIcon v-if="icon" :name="icon" :size="12" class="sab-icon" />
        <span class="sab-label">{{ label }}</span>
        <button class="sab-mode-btn" @click.stop="toggleMinimal" title="仅工作模式（隐藏内容，agent 继续运行）">
          <SvgIcon name="eye-off" :size="11" />
        </button>
        <span v-if="mode === 'collapsed' && summary" class="sab-preview">{{ summary }}</span>
      </div>

      <div v-if="mode === 'expanded'" class="sab-body">
        <div class="sab-body-inner" ref="bodyInnerRef">
          <slot />
        </div>
        <div class="sab-body-footer">
          <button class="sab-foot-min-btn" @click="toggleMinimal">
            <SvgIcon name="eye-off" :size="10" />
            <span>仅工作</span>
          </button>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  type: { type: String, default: 'tool_call' },
  label: { type: String, default: '' },
  icon: { type: String, default: '' },
  summary: { type: String, default: '' },
  mode: { type: String, default: 'collapsed' },
})

const emit = defineEmits(['toggle', 'update:mode'])

const bodyInnerRef = ref(null)
const hasContent = ref(false)

function checkContent() {
  if (bodyInnerRef.value) {
    hasContent.value = bodyInnerRef.value.children.length > 0 ||
      (bodyInnerRef.value.textContent || '').trim().length > 0
  }
}

function toggleCollapse() {
  const next = props.mode === 'expanded' ? 'collapsed' : 'expanded'
  emit('toggle')
  emit('update:mode', next)
}

function toggleMinimal() {
  const next = props.mode === 'minimal' ? 'expanded' : 'minimal'
  emit('update:mode', next)
}

function expandFromMinimal() {
  emit('update:mode', 'expanded')
}

defineExpose({ checkContent, toggleMinimal, expandFromMinimal })
</script>

<style scoped>
.sab-wrapper {
  position: relative;
  margin: 2px 0;
  font-size: 13px;
}
.sab-header {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 3px 6px 3px 0;
  cursor: pointer;
  user-select: none;
  border-radius: 4px;
  min-height: 24px;
  transition: background 0.12s;
}
.sab-header:hover {
  background: var(--bg-hover);
}
.sab-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
  border: 2px solid var(--border-color);
  background: var(--bg-primary);
  transition: all 0.15s;
  position: relative;
  z-index: 1;
}
.sab-wrapper.sab-mode-collapsed .sab-dot { background: var(--bg-tertiary); }
.sab-type-thinking .sab-dot { border-color: var(--accent); background: var(--accent-bg); }
.sab-type-thinking.sab-mode-collapsed .sab-dot { background: var(--accent); }
.sab-type-thinking.sab-mode-minimal .sab-dot { background: var(--accent); animation: sabDotPulse 2s infinite; }
.sab-type-tool_call .sab-dot { border-color: #d4a74e; background: rgba(212,167,78,0.15); }
.sab-type-tool_call.sab-mode-collapsed .sab-dot { background: #d4a74e; }
.sab-type-tool_call.sab-mode-minimal .sab-dot { background: #d4a74e; animation: sabDotPulse 2s infinite; }
.sab-type-content .sab-dot { border-color: #6a9955; background: rgba(106,153,85,0.15); }
.sab-type-content.sab-mode-collapsed .sab-dot { background: #6a9955; }
.sab-type-content.sab-mode-minimal .sab-dot { background: #6a9955; animation: sabDotPulse 2s infinite; }
.sab-type-subagent .sab-dot { border-color: #c586c0; background: rgba(197,134,192,0.15); }
.sab-type-subagent.sab-mode-collapsed .sab-dot { background: #c586c0; }
.sab-type-subagent.sab-mode-minimal .sab-dot { background: #c586c0; animation: sabDotPulse 2s infinite; }
.sab-wrapper:not(.sab-mode-collapsed):not(.sab-mode-minimal) .sab-dot {
  box-shadow: 0 0 0 3px color-mix(in srgb, currentColor 15%, transparent);
}
@keyframes sabDotPulse {
  0%,100% { opacity: 0.6; box-shadow: 0 0 0 0 color-mix(in srgb, currentColor 30%, transparent); }
  50% { opacity: 1; box-shadow: 0 0 0 4px color-mix(in srgb, currentColor 20%, transparent); }
}
.sab-chevron { font-size: 9px; color: var(--text-muted); width: 8px; text-align: center; flex-shrink: 0; }
.sab-icon { flex-shrink: 0; color: var(--text-secondary); }
.sab-label { font-size: 12px; font-weight: 500; color: var(--text-primary); font-family: var(--font-code); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; min-width: 40px; }
.sab-type-thinking .sab-label, .sab-type-content .sab-label { font-family: var(--font-ui); font-weight: 600; }
.sab-type-thinking .sab-label { color: var(--accent); }
.sab-type-content .sab-label { color: #6a9955; }
.sab-preview { font-size: 11px; color: var(--text-muted); font-style: italic; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 200px; flex-shrink: 1; min-width: 0; }

.sab-minimal { display: flex; align-items: center; gap: 0; padding: 2px 0; cursor: pointer; position: relative; transition: all 0.15s; }
.sab-minimal .sab-dot { width: 10px; height: 10px; }
.sab-minimal:hover { opacity: 1; }
.sab-minimal:not(:hover) .sab-minimal-hint { opacity: 0; }
.sab-minimal-hint { font-size: 8px; color: var(--text-muted); margin-left: 2px; transition: opacity 0.15s; opacity: 0.6; }
.sab-minimal-pulse { position: absolute; left: 3px; top: 0; width: 6px; height: 6px; border-radius: 50%; background: var(--accent); opacity: 0.4; animation: sabMinPulse 1.5s infinite; }
@keyframes sabMinPulse { 0%,100% { transform: scale(1); opacity: 0.4; } 50% { transform: scale(1.8); opacity: 0; } }

.sab-mode-btn { display: flex; align-items: center; justify-content: center; background: none; border: 1px solid transparent; color: var(--text-muted); cursor: pointer; border-radius: 3px; padding: 1px 3px; opacity: 0.3; transition: all 0.12s; flex-shrink: 0; }
.sab-header:hover .sab-mode-btn { opacity: 0.7; }
.sab-mode-btn:hover { opacity: 1 !important; background: var(--bg-active); color: var(--text-secondary); border-color: var(--border-color); }

.sab-body { margin: 0 0 0 18px; border-left: none; overflow: hidden; max-height: 280px; border-radius: 0 0 6px 6px; }
.sab-body-inner { padding: 6px 10px; font-size: 13px; line-height: 1.6; word-break: break-word; overflow-wrap: break-word; max-height: 240px; overflow-y: auto; background: var(--bg-primary); border: 1px solid var(--border-color); border-top: none; border-radius: 0 0 6px 6px; }
.sab-wrapper.sab-mode-expanded { position: relative; }
.sab-wrapper.sab-mode-expanded::before { content: ''; position: absolute; left: 4px; top: 14px; bottom: auto; width: 2px; height: 6px; background: var(--border-color); opacity: 0.3; border-radius: 1px; z-index: 0; }

.sab-body-footer { display: flex; justify-content: flex-end; padding: 2px 6px 4px; background: var(--bg-primary); border: 1px solid var(--border-color); border-top: none; border-radius: 0 0 6px 6px; margin-top: -1px; }
.sab-foot-min-btn { display: flex; align-items: center; gap: 3px; background: none; border: 1px solid transparent; color: var(--text-muted); cursor: pointer; font-size: 10px; padding: 2px 6px; border-radius: 3px; opacity: 0.5; transition: all 0.12s; }
.sab-foot-min-btn:hover { opacity: 1; background: var(--bg-active); color: var(--text-secondary); border-color: var(--border-color); }
</style>
