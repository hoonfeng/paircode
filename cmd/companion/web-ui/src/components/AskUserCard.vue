<template>
  <div class="ask-user-card">
    <div class="ask-user-question">{{ question }}</div>
    <div class="ask-user-input-row">
      <input v-model="answer" class="ask-user-input" type="text"
             placeholder="输入回答..." @keydown.enter="submit" :disabled="answered" />
      <button class="ask-user-btn" @click="submit" :disabled="answered || !answer.trim()">
        {{ answered ? '已回答' : '发送' }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  question: { type: String, default: '' },
  callId: { type: String, default: '' },
  answered: { type: Boolean, default: false },
})
const emit = defineEmits(['answer'])

const answer = ref('')

function submit() {
  if (props.answered || !answer.value.trim()) return
  emit('answer', { callId: props.callId, answer: answer.value.trim() })
  answer.value = ''
}
</script>

<style scoped>
.ask-user-card {
  padding: 0;
}
.ask-user-question {
  font-size: 14px; color: var(--text-primary); margin-bottom: 8px;
  line-height: 1.5; white-space: pre-wrap;
}
.ask-user-input-row {
  display: flex; gap: 6px;
}
.ask-user-input {
  flex: 1; padding: 6px 10px;
  background: var(--input-bg); border: 1px solid var(--border-color);
  color: var(--text-primary); font-size: 13px; outline: none; border-radius: 3px;
}
.ask-user-input:focus { border-color: var(--accent); }
.ask-user-btn {
  padding: 6px 14px; background: var(--accent); color: #fff;
  border: none; border-radius: 3px; font-size: 12px; cursor: pointer;
  white-space: nowrap;
}
.ask-user-btn:disabled { opacity: 0.5; cursor: default; }
.ask-user-btn:hover:not(:disabled) { opacity: 0.85; }
</style>
