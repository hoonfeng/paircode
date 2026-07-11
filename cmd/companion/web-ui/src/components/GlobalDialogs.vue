<template>
  <!-- Toast 通知区域 -->
  <div class="toast-container">
    <div v-for="t in dialogState.toasts" :key="t.id"
         :class="['toast-item', 'toast-' + (t.type || 'info')]">
      {{ t.message }}
    </div>
  </div>

  <!-- Confirm 对话框 -->
  <div v-if="dialogState.show && dialogState.type === 'confirm'" class="dlg-overlay" @click.self="handleCancel">
    <div class="dlg-box" style="max-width:400px">
      <div class="dlg-title">{{ dialogState.title }}</div>
      <div class="dlg-body">{{ dialogState.message }}</div>
      <div class="dlg-actions">
        <button class="dlg-btn" @click="handleCancel">{{ dialogState.cancelText }}</button>
        <button class="dlg-btn primary" @click="handleConfirm">{{ dialogState.confirmText }}</button>
      </div>
    </div>
  </div>

  <!-- Prompt 对话框 -->
  <div v-if="dialogState.show && dialogState.type === 'prompt'" class="dlg-overlay" @click.self="handleCancel">
    <div class="dlg-box" style="max-width:420px">
      <div class="dlg-title">{{ dialogState.title }}</div>
      <div class="dlg-body" style="display:flex;flex-direction:column;gap:8px">
        <span style="font-size:13px;color:var(--text-secondary)">{{ dialogState.message }}</span>
        <input ref="promptInputRef" v-model="dialogState.inputValue"
               :placeholder="dialogState.inputPlaceholder"
               class="dlg-input"
               @keyup.enter="handleConfirm"
               @keyup.escape="handleCancel" />
      </div>
      <div class="dlg-actions">
        <button class="dlg-btn" @click="handleCancel">{{ dialogState.cancelText }}</button>
        <button class="dlg-btn primary" @click="handleConfirm">{{ dialogState.confirmText }}</button>
      </div>
    </div>
  </div>

  <!-- Alert 信息框 -->
  <div v-if="dialogState.show && dialogState.type === 'alert'" class="dlg-overlay" @click.self="handleConfirm">
    <div class="dlg-box" style="max-width:400px">
      <div class="dlg-title">{{ dialogState.title }}</div>
      <div class="dlg-body" style="white-space:pre-line">{{ dialogState.message }}</div>
      <div class="dlg-actions">
        <button class="dlg-btn primary" @click="handleConfirm">确定</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, watch } from 'vue'
import { dialogState } from '../main.js'

const promptInputRef = ref(null)

watch(() => dialogState.show, (v) => {
  if (v && dialogState.type === 'prompt') {
    nextTick(() => {
      promptInputRef.value?.focus()
      promptInputRef.value?.select()
    })
  }
})

function handleConfirm() {
  const val = dialogState.type === 'prompt' ? dialogState.inputValue : true
  dialogState.show = false
  if (dialogState.resolve) dialogState.resolve(val)
  dialogState.resolve = null
}

function handleCancel() {
  const val = dialogState.type === 'prompt' ? null : false
  dialogState.show = false
  if (dialogState.resolve) dialogState.resolve(val)
  dialogState.resolve = null
}
</script>

<style scoped>
.toast-container {
  position: fixed; top: 40px; right: 16px; z-index: 10000;
  display: flex; flex-direction: column; gap: 6px; max-width: 320px;
}
.toast-item {
  font-size: 13px; padding: 8px 14px; border-radius: var(--border-radius);
  box-shadow: var(--shadow-md); animation: toastIn .25s ease-out;
  word-break: break-word;
}
.toast-info { background: var(--bg-secondary); color: var(--text-primary); border: 1px solid var(--border-color); border-left: 3px solid var(--accent); }
.toast-success { background: var(--bg-secondary); color: var(--text-primary); border: 1px solid var(--border-color); border-left: 3px solid #2ea043; }
.toast-warning { background: var(--bg-secondary); color: var(--text-primary); border: 1px solid var(--border-color); border-left: 3px solid #d4a74e; }
.toast-error { background: var(--bg-secondary); color: var(--text-primary); border: 1px solid var(--border-color); border-left: 3px solid #e74c3c; }
@keyframes toastIn { from { opacity:0;transform:translateX(20px) } to { opacity:1;transform:translateX(0) } }

.dlg-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.5); z-index: 10001;
  display: flex; align-items: center; justify-content: center;
}
.dlg-box {
  background: var(--bg-secondary); border: 1px solid var(--border-color);
  border-radius: var(--border-radius-lg); padding: 20px 24px; width: 90%;
  box-shadow: var(--shadow-md);
}
.dlg-title { font-size: 15px; font-weight: 600; color: var(--text-primary); margin-bottom: 12px; }
.dlg-body { font-size: 13px; color: var(--text-primary); margin-bottom: 16px; line-height: 1.5; }
.dlg-actions { display: flex; gap: 8px; justify-content: flex-end; }
.dlg-btn {
  background: var(--bg-tertiary); border: 1px solid var(--border-color);
  color: var(--text-primary); padding: 7px 20px; border-radius: 4px;
  cursor: pointer; font-size: 13px;
}
.dlg-btn:hover { background: var(--bg-hover); }
.dlg-btn.primary { background: var(--accent); color: #000; border-color: var(--accent); }
.dlg-btn.primary:hover { filter: brightness(1.1); }
.dlg-input {
  width: 100%; box-sizing: border-box;
  background: var(--bg-primary); border: 1px solid var(--border-color);
  color: var(--text-primary); padding: 8px 10px; border-radius: 4px; font-size: 13px; outline: none;
}
.dlg-input:focus { border-color: var(--accent); }
</style>
