<template>
  <div class="ask-user-card">
    <!-- ★ 2026-09-17：会话已结束（服务端不存在）失效提示——回答无法提交，可继续发新消息 -->
    <div v-if="stale" class="ask-stale-hint">⚠ 该会话已结束（服务端不存在），回答无法提交；可直接在下方输入框发送新消息继续。</div>
    <!-- ★ Round3 ⑤ 多问题模式：questions 数组渲染列表，一次「提交」回灌 answers -->
    <div v-if="questions && questions.length" class="ask-multi-list">
      <div v-for="(q, qi) in questions" :key="q.id || qi" class="ask-multi-item">
        <div class="ask-user-question">{{ qi + 1 }}. {{ q.question }}</div>
        <!-- 有选项：单选/多选（multi_select 区分） -->
        <div v-if="q.options && q.options.length" class="ask-user-options">
          <label v-for="opt in q.options" :key="opt"
                 :class="['ask-option', { selected: isMultiSelected(q, opt) }]"
                 @click="toggleMultiOption(q, opt)">
            <span v-if="q.multiSelect" :class="['ask-checkbox', { checked: isMultiSelected(q, opt) }]">
              <span v-if="isMultiSelected(q, opt)" class="ask-checkmark">✓</span>
            </span>
            <span v-else :class="['ask-radio-circle', { checked: isMultiSelected(q, opt) }]"></span>
            <span class="ask-option-text">{{ opt }}</span>
          </label>
        </div>
        <!-- 无选项：文本输入 -->
        <div v-else class="ask-user-input-row">
          <input v-model="multiTexts[q.id]" class="ask-user-input" type="text"
                 placeholder="输入回答..." @keydown.enter="submitMultiForm" :disabled="answered || stale" />
        </div>
        <!-- ★ 2026-09-25 历史回显：该问题当时的回答（选择项/自定义文本） -->
        <div v-if="answeredTextFor(q)" class="ask-answered-line"
             :class="{ 'ask-answered-failed': isFailedAnswer(answeredTextFor(q)) }">
          <span class="ask-answered-label">{{ isFailedAnswer(answeredTextFor(q)) ? '未收到回答' : '你的回答' }}</span>
          <span class="ask-answered-text">{{ answeredTextFor(q) }}</span>
        </div>
      </div>
      <div class="ask-multi-actions">
        <button class="ask-user-btn" @click="submitMultiForm" :disabled="answered || stale || !multiFormValid">
          {{ stale ? '会话已结束' : (answered ? '已回答' : '提交回答') }}
        </button>
      </div>
    </div>

    <!-- 单问题模式（原路径不变） -->
    <div v-else>
    <div class="ask-user-question">{{ question }}</div>

    <!-- ★ 2026-09-25 历史回显：显示当时的选择项 / 自定义输入 / 文本回答 -->
    <div v-if="answered && answeredShow" class="ask-answered-line"
         :class="{ 'ask-answered-failed': isFailedAnswer(answeredShow) }">
      <span class="ask-answered-label">{{ isFailedAnswer(answeredShow) ? '未收到回答' : '你的回答' }}</span>
      <span class="ask-answered-text">{{ answeredShow }}</span>
    </div>

    <!-- 单选 (radio)：options 为空时降级为文本输入（见下方兜底） -->
    <div v-if="askType === 'single' && hasOptions" class="ask-user-options">
      <label v-for="(opt, i) in options" :key="i"
             :class="['ask-option', { selected: selectedOpt === opt }]"
             @click="selectOption(opt)">
        <span class="ask-radio-circle" :class="{ checked: selectedOpt === opt }"></span>
        <span class="ask-option-text">{{ opt }}</span>
      </label>
      <button class="ask-user-btn" @click="submitOption"
              :disabled="answered || stale || !selectedOpt">
        {{ stale ? '会话已结束' : (answered ? '已回答' : '提交选择') }}
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
                :disabled="answered || stale || selectedMulti.length === 0">
          {{ stale ? '会话已结束' : (answered ? '已回答' : '提交选择') }}
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
               placeholder="输入自定义回答..." @keydown.enter="submitCustom" :disabled="answered || stale" />
        <button class="ask-user-btn" @click="submitCustom"
                :disabled="answered || stale || (!selectedOpt && !customInput.trim())">
          {{ stale ? '会话已结束' : (answered ? '已回答' : '发送') }}
        </button>
      </div>
    </div>

    <!-- 纯文本输入 (默认/兜底：任何未识别类型都显示输入框，保证可回答) -->
    <!-- ★ options 空兜底：选择类 askType 但未提供选项时降级为文本输入，附提示，
         避免出现「只有提交按钮、点不了」的死卡片（模型漏填 options 的常见情况） -->
    <div v-else class="ask-user-input-row">
      <input v-model="textInput" class="ask-user-input" type="text"
             :placeholder="noOptionsHint" @keydown.enter="submitText" :disabled="answered || stale" />
      <button class="ask-user-btn" @click="submitText"
              :disabled="answered || stale || !textInput.trim()">
        {{ stale ? '会话已结束' : (answered ? '已回答' : '发送') }}
      </button>
    </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  question: { type: String, default: '' },
  askType: { type: String, default: 'text' },
  options: { type: Array, default: () => [] },
  callId: { type: String, default: '' },
  answered: { type: Boolean, default: false },
  // ★ 2026-09-17：会话已结束（服务端不存在，提交必然失败）→ 卡片置失效态
  stale: { type: Boolean, default: false },
  // ★ Round3 ⑤ 多问题：questions 数组 [{id, question, options?, multiSelect?}]
  questions: { type: Array, default: () => [] },
  // ★ 2026-09-25 历史回显：已答内容（后端 segments 提供，见 message_store.go
  //   SegmentsFromMessage 的 ask_user 分支——从 tool 结果解析 Answer/Answers）。
  //   answer=单问题答案文本；answers=[{id, answer}]=多问题答案数组。
  //   此前两者只用于 answered 判定、未参与渲染 → 回看历史看不到当时选了什么。
  answer: { type: String, default: '' },
  answers: { type: Array, default: () => [] },
})
const emit = defineEmits(['answer'])

// ── 多问题模式（Round3 ⑤） ──
const multiTexts = ref({})        // qid → 文本输入
const multiSelections = ref({})   // qid → 已选项数组
function isMultiSelected(q, opt) {
  return (multiSelections.value[q.id] || []).includes(opt)
}
function toggleMultiOption(q, opt) {
  const list = multiSelections.value[q.id] || []
  if (q.multiSelect) {
    const idx = list.indexOf(opt)
    multiSelections.value[q.id] = idx >= 0 ? list.filter(o => o !== opt) : [...list, opt]
  } else {
    multiSelections.value[q.id] = [opt]
  }
}
const multiFormValid = computed(() => {
  const qs = props.questions || []
  return qs.length > 0 && qs.every(q => {
    if (q.options && q.options.length) return (multiSelections.value[q.id] || []).length > 0
    return !!((multiTexts.value[q.id] || '').trim())
  })
})
function submitMultiForm() {
  const qs = props.questions || []
  const answers = qs.map(q => {
    let answer = ''
    if (q.options && q.options.length) answer = (multiSelections.value[q.id] || []).join(', ')
    else answer = (multiTexts.value[q.id] || '').trim()
    return { id: q.id, answer }
  })
  emit('answer', { callId: props.callId, answers })
}

// ── 单问题模式（原逻辑不变） ──

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

// ── ★ 2026-09-25 历史回显：已答内容回填（选项高亮 + 自定义文本/纯文本） ──
// 缺陷：回答提交后仅后端落盘（tool 结果 → segments.answer/answers），前端卡片
// 从未消费这两者 → 回看历史时看不到当时选了哪一项、自定义输入了什么。

// splitAnswer 拆分答案文本（多选/多问题提交时以 ', ' 连接，见 submitMulti/submitMultiForm）。
const splitAnswer = (s) => String(s || '').split(',').map(x => x.trim()).filter(Boolean)

// isFailedAnswer 工具失败/超时文本（后端把工具 error 文本当结果解析成 answer，如
// "Error: JS 工具 ask_user 执行失败: ... 等待用户回答超时（5 分钟）"）——那不是用户
// 答案：展示为「未收到回答」，且不参与选项高亮/输入框回填。
const isFailedAnswer = (s) => /^Error:|^error:|等待用户回答超时/.test(String(s || '').trim())

// answeredTextFor 多问题模式：取该问题当时的答案文本（按 question id 关联）。
function answeredTextFor(q) {
  const hit = (props.answers || []).find(a => a && a.id === (q && q.id))
  return hit ? String(hit.answer || '').trim() : ''
}

// answeredShow 单问题模式的答案文本。
const answeredShow = computed(() => String(props.answer || '').trim())

// restoreFromAnswer 把已答内容回填到本地交互状态，使历史态外观与实时态一致：
//   有选项 → 高亮当时所选（多选回填多项）；无选项/选项外文本 → 回填输入框。
function restoreFromAnswer() {
  const qs = props.questions || []
  if (qs.length > 0) {
    const sel = { ...multiSelections.value }
    const txt = { ...multiTexts.value }
    for (const q of qs) {
      const ans = answeredTextFor(q)
      if (!ans || isFailedAnswer(ans)) continue
      if (q.options && q.options.length) {
        const matched = splitAnswer(ans).filter(p => q.options.includes(p))
        sel[q.id] = matched
        if (matched.length === 0) txt[q.id] = ans // 选项外文本（模型自定义/历史手输）
      } else if (!txt[q.id]) {
        txt[q.id] = ans
      }
    }
    multiSelections.value = sel
    multiTexts.value = txt
    return
  }
  const ans = answeredShow.value
  if (!ans || isFailedAnswer(ans)) return
  const opts = Array.isArray(props.options) ? props.options : []
  const matched = opts.length > 0 ? splitAnswer(ans).filter(p => opts.includes(p)) : []
  if (matched.length > 0) {
    selectedOpt.value = matched[0]
    selectedMulti.value = matched
  } else if (opts.length > 0) {
    customInput.value = ans // single-with-input：自定义文本回填
  } else {
    textInput.value = ans // text：纯文本回答回填
  }
}

// 历史渲染（segments 带 answer/answers）与实时回答（onAskAnswer 写入 seg）都会触发；
// immediate 保证首帧回显，deep 保证 answers 数组内容变化也能同步。
watch(() => [props.answer, props.answers], restoreFromAnswer, { immediate: true, deep: true })
</script>

<style scoped>
.ask-user-card { padding: 0; }
/* ★ 2026-09-17：会话已结束失效提示 */
.ask-stale-hint {
  margin-bottom: 8px; padding: 6px 10px; border-radius: 4px;
  font-size: 12px; line-height: 1.5;
  color: var(--warning, #e6a23c); background: rgba(230, 162, 60, 0.08);
  border: 1px solid rgba(230, 162, 60, 0.3);
}
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
  background: var(--color-accent-bg);
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
  border-radius: 50%; background: var(--color-accent-fg);
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
.ask-checkmark { color: var(--color-accent-fg); font-size: 11px; font-weight: 700; }

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
  padding: 7px 16px; background: var(--accent); color: var(--color-accent-fg);
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

/* ── ★ 2026-09-25 历史回显：已答内容行（当时的选择 / 自定义输入） ── */
.ask-answered-line {
  display: flex; align-items: baseline; gap: 6px;
  margin-top: 6px; padding: 6px 8px;
  background: var(--bg-hover);
  border-left: 2px solid var(--accent);
  border-radius: 3px;
  font-size: 12.5px; line-height: 1.5;
}
.ask-answered-label { flex-shrink: 0; color: var(--text-muted); font-size: 11px; }
.ask-answered-text { color: var(--text-primary); white-space: pre-wrap; word-break: break-word; }
.ask-answered-failed { border-left-color: var(--text-muted); }
.ask-answered-failed .ask-answered-text { color: var(--text-muted); font-style: italic; }

/* 已答态（历史回看）输入框回填内容可读性：disabled 默认偏灰 */
.ask-user-input:disabled { opacity: 1; color: var(--text-primary); }
</style>
