<template>
  <div class="chat-view">
    <div class="chat-layout">
      <div class="chat-main">
        <div class="chat-container">
          <div class="chat-header">
            <h2>对话</h2>
            <span v-if="convId" class="conv-id">{{ convId }}</span>
          </div>
          <div class="chat-messages">
            <p class="welcome-text">选择一个对话开始交流</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useRoute } from 'vue-router'
const route = useRoute()
const convId = route.params?.convId
</script>

<style scoped>
.chat-view {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}
.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
}
.chat-container {
  flex: 1;
  display: flex;
  flex-direction: column;
}
.chat-header {
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
}
.chat-header h2 {
  margin: 0;
  font-size: 16px;
}
.conv-id {
  font-size: 12px;
  color: var(--text-muted);
}
.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}
.welcome-text {
  color: var(--text-muted);
  text-align: center;
  margin-top: 40px;
}
</style>
