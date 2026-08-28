<template>
  <div class="ask-user-card">
    <div class="ask-user-question">{{ question }}</div>

    <!-- 单选 (radio)：options 为空时降级为文本输入（见下方兜底） -->
    <div v-if="askType === 'single' && hasOptions" class="ask-user-options">
      <label v-for="(opt, i) in options" :key="i"
             :class="['ask-option', { selected: selectedOpt === opt }]"
             @click="selectOption(opt)">
        <span class="ask-radio-circle" :class="{ checked: selectedOpt === opt }"></span>
        <span class="ask-option-text">{{ opt }}</span>
      </label>
      <button class="ask-user-btn" @click="submitOption"
              :disabled="answered || !selectedOpt">
        {{ answered ? '已回答' : '提交选择' }}
      </button>
    </div>

    <!-- 多选 (checkbox)：options 为空时降级为文本输入 -->
    <div v-else-if="askType === 'multi' && hasOptions" class="ask-user-options">
      <label v-for="(opt, i) in options" :key="i"
             :class="['ask-option', { selected: selectedMulti.includes(opt) }]"
             @click="toggleMulti(opt)">
        <span class="ask-checkbox" :class="{ checked: selectedMulti.includes(opt) }">
          <span v-if="selectedMulti.includes(opt)" class="ask-checkmark">✓</span>
        </span>
        <span class="ask-option-text">{{ opt }}</span>
      </label>
      <div class="ask-multi-actions">
        <button class="ask-user-btn" @click="submitMulti"
                :disabled="answered || selectedMulti.length === 0">
          {{ answered ? '已回答' : '提交选择' }}
        </button>
      </div>
    </div>

    <!-- 单选 + 自由输入：options 为空时降级为文本输入 -->
    <div v-else-if="askType === 'single-with-input' && hasOptions" class="ask-user-wrapper">
      <div class="ask-user-options">
        <label v-for="(opt, i) in options" :key="i"
               :class="['ask-option', { selected: selectedOpt === opt }]"
               @click="selectOption(opt)">
          <span class="ask-radio-circle" :class="{ checked: selectedOpt === opt }"></span>
          <span class="ask-option-text">{{ opt }}</span>
        </label>
      </div>
      <div class="ask-user-or-divider"><span>或自定义输入</span></div>
      <div class="ask-user-input-row">
        <input v-model="customInput" class="ask-user-input" type="text"
               placeholder="输入自定义回答..." @keydown.enter="submitCustom" :disabled="answered" />
        <button class="ask-user-btn" @click="submitCustom"
                :disabled="answered || (!selectedOpt && !customInput.trim())">
          {{ answered ? '已回答' : '发送' }}
        </button>
      </div>
    </div>

    <!-- 纯文本输入 (默认/兜底：任何未识别类型都显示输入框，保证可回答) -->
    <!-- ★ options 空兜底：选择类 askType 但未提供选项时降级为文本输入，附提示，
         避免出现「只有提交按钮、点不了」的死卡片（模型漏填 options 的常见情况） -->
    <div v-else class="ask-user-input-row">
      <input v-model="textInput" class="ask-user-input" type="text"
             :placeholder="noOptionsHint" @keydown.enter="submitText" :disabled="answered" />
      <button class="ask-user-btn" @click="submitText"
              :disabled="answered || !textInput.trim()">
        {{ answered ? '已回答' : '发送' }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  question: { type: String, default: '' },
  askType: { type: String, default: 'text' },
  options: { type: Array, default: () => [] },
  callId: { type: String, default: '' },
  answered: { type: Boolean, default: false },
})
const emit = defineEmits(['answer'])

// hasOptions：选项列表非空（选择类卡片仅在有选项时渲染选项）
const hasOptions = computed(() => Array.isArray(props.options) && props.options.length > 0)
// noOptionsHint：选择类 askType 但无选项时，输入框提示语
const noOptionsHint = computed(() => {
  const t = String(props.askType || '')
  if (t === 'single' || t === 'multi' || t === 'single-with-input') {
    return '（未提供选项，请直接输入你的回答）'
  }
  return '输入回答...'
})

const selectedOpt = ref('')
const selectedMulti = ref([])
const customInput = ref('')
const textInput = ref('')

function selectOption(opt) {
  selectedOpt.value = opt
}

function toggleMulti(opt) {
  const idx = selectedMulti.value.indexOf(opt)
  if (idx >= 0) {
    selectedMulti.value = selectedMulti.value.filter(o => o !== opt)
  } else {
    selectedMulti.value = [...selectedMulti.value, opt]
  }
}

function submitOption() {
  if (!selectedOpt.value) return
  emit('answer', { callId: props.callId, answer: selectedOpt.value })
}

function submitMulti() {
  if (selectedMulti.value.length === 0) return
  emit('answer', { callId: props.callId, answer: selectedMulti.value.join(', ') })
}

function submitCustom() {
  const ans = selectedOpt.value && !customInput.value.trim()
    ? selectedOpt.value
    : customInput.value.trim()
  if (!ans) return
  emit('answer', { callId: props.callId, answer: ans })
  customInput.value = ''
}

function submitText() {
  if (!textInput.value.trim()) return
  emit('answer', { callId: props.callId, answer: textInput.value.trim() })
  textInput.value = ''
}
</script>

<style scoped>
.ask-user-card { padding: 0; }
.ask-user-question {
  font-size: 14px; color: var(--text-primary); margin-bottom: 10px;
  line-height: 1.5; white-space: pre-wrap;
}

/* ── 选项列表 ── */
.ask-user-options { display: flex; flex-direction: column; gap: 4px; margin-bottom: 10px; }
.ask-option {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 10px; border-radius: 6px;
  cursor: pointer; user-select: none;
  border: 1px solid var(--border-color);
  background: var(--bg-primary);
  transition: all 0.12s;
}
.ask-option:hover { background: var(--bg-hover); border-color: var(--accent); }
.ask-option.selected {
  background: rgba(126, 184, 218, 0.08);
  border-color: var(--accent);
}

/* ── 单选圆圈 ── */
.ask-radio-circle {
  width: 16px; height: 16px; border-radius: 50%;
  border: 2px solid var(--border-color);
  flex-shrink: 0; position: relative;
  transition: all 0.12s;
}
.ask-radio-circle.checked {
  border-color: var(--accent);
  background: var(--accent);
}
.ask-radio-circle.checked::after {
  content: ''; position: absolute;
  top: 3px; left: 3px; width: 6px; height: 6px;
  border-radius: 50%; background: #fff;
}

/* ── 多选方框 ── */
.ask-checkbox {
  width: 16px; height: 16px; border-radius: 3px;
  border: 2px solid var(--border-color);
  flex-shrink: 0; display: flex;
  align-items: center; justify-content: center;
  transition: all 0.12s;
}
.ask-checkbox.checked { border-color: var(--accent); background: var(--accent); }
.ask-checkmark { color: #fff; font-size: 11px; font-weight: 700; }

.ask-option-text { font-size: 13px; color: var(--text-primary); }

/* ── 输入行 ── */
.ask-user-input-row { display: flex; gap: 6px; }
.ask-user-input {
  flex: 1; padding: 7px 10px;
  background: var(--input-bg); border: 1px solid var(--border-color);
  color: var(--text-primary); font-size: 13px; outline: none; border-radius: 4px;
}
.ask-user-input:focus { border-color: var(--accent); }

/* ── 按钮 ── */
.ask-user-btn {
  padding: 7px 16px; background: var(--accent); color: #fff;
  border: none; border-radius: 4px; font-size: 13px; cursor: pointer;
  white-space: nowrap; transition: opacity 0.12s;
}
.ask-user-btn:disabled { opacity: 0.5; cursor: default; }
.ask-user-btn:hover:not(:disabled) { opacity: 0.85; }

/* ── 分割线 ── */
.ask-user-or-divider {
  display: flex; align-items: center; gap: 8px;
  margin: 8px 0; color: var(--text-muted); font-size: 11px;
}
.ask-user-or-divider::before,
.ask-user-or-divider::after {
  content: ''; flex: 1; height: 1px; background: var(--border-color);
}

/* ── 单选+自由输入包装 ── */
.ask-user-wrapper { display: flex; flex-direction: column; gap: 0; }

/* ── 多选按钮区 ── */
.ask-multi-actions { margin-top: 4px; }
</style>
