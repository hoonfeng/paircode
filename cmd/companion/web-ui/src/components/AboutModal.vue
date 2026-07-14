<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content about-modal">
      <div class="modal-header">
        <h2><SvgIcon name="info" :size="18" /> 关于 PairCode</h2>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>
      <div class="modal-body">
        <!-- Logo + 标题 -->
        <div class="about-hero">
          <div class="about-logo">
            <SvgIcon name="code" :size="48" color="#58a6ff" />
          </div>
          <div class="about-title">PairCode IDE</div>
          <div class="about-version">版本 {{ version }}</div>
        </div>

        <!-- 描述 -->
        <div class="about-section">
          <p class="about-description">
            PairCode IDE 是一个 AI 辅助编程的 Web 版本集成开发环境。
            它将 AI 对话能力深度融入编码工作流，你可以像和资深开发者对话一样，
            通过自然语言描述需求，AI 自动理解上下文并生成代码、修复错误、优化结构。
            无需切换工具，在同一个界面内完成编码、调试、版本控制全部环节。
          </p>
        </div>

        <!-- 特性亮点 -->
        <div class="about-section">
          <div class="section-title">主要特性</div>
          <ul class="feature-list">
            <li><SvgIcon name="bot" :size="14" color="var(--accent)" /> AI 对话编程 — 用自然语言与 AI 对话，自动生成与重构代码</li>
            <li><SvgIcon name="file" :size="14" color="var(--accent)" /> 智能代码编辑器 — 多语言语法高亮，流畅的编辑体验</li>
            <li><SvgIcon name="git-branch" :size="14" color="var(--accent)" /> Git 版本控制 — 在对话中完成全部 Git 操作</li>
            <li><SvgIcon name="terminal" :size="14" color="var(--accent)" /> 内置终端 — 无需离开 IDE 即可执行命令</li>
            <li><SvgIcon name="search" :size="14" color="var(--accent)" /> 全局搜索 — 快速搜索文件与代码内容</li>
            <li><SvgIcon name="settings" :size="14" color="var(--accent)" /> 自主 Agent 模式 — AI 主动分析项目并自动执行任务</li>
            <li><SvgIcon name="grid" :size="14" color="var(--accent)" /> 对话历史管理 — 自动保存、回溯与继续历史对话</li>
            <li><SvgIcon name="tool" :size="14" color="var(--accent)" /> Skills / MCP 扩展 — 通过技能市场扩展 IDE 能力</li>
          </ul>
        </div>

        <!-- 技术栈 -->
        <div class="about-section">
          <div class="section-title">技术栈</div>
          <div class="tech-stack">
            <span class="tech-badge">Go</span>
            <span class="tech-badge">Vue 3</span>
            <span class="tech-badge">WebSocket</span>
            <span class="tech-badge">MCP</span>
            <span class="tech-badge">CodeMirror</span>
            <span class="tech-badge">REST API</span>
          </div>
        </div>

        <!-- 系统信息 -->
        <div class="about-section">
          <div class="section-title">系统信息</div>
          <div class="sys-info" v-if="!sysLoading">
            <div class="info-row"><span class="info-label">主机名</span><span>{{ sysInfo.hostname }}</span></div>
            <div class="info-row"><span class="info-label">操作系统</span><span>{{ sysInfo.os }}</span></div>
            <div class="info-row"><span class="info-label">工作区</span><span class="info-path">{{ sysInfo.workspace }}</span></div>
            <div class="info-row"><span class="info-label">平台信息</span><span>{{ sysInfo.goos }}</span></div>
          </div>
          <div v-else class="loading-info">加载中...</div>
        </div>
      </div>

      <!-- 底部 -->
      <div class="modal-footer">
        <button class="btn-primary" @click="$emit('openHelp')" v-if="showHelpBtn">
          <SvgIcon name="book-open" :size="14" /> 查看帮助文档
        </button>
        <button class="btn-secondary" @click="$emit('close')">关闭</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import SvgIcon from './SvgIcon.vue'
import api from '../api.js'

const emit = defineEmits(['close', 'openHelp'])

defineProps({
  showHelpBtn: { type: Boolean, default: true },
})

const version = ref('v0.1.0')
const sysInfo = ref({})
const sysLoading = ref(true)

onMounted(async () => {
  try {
    sysInfo.value = await api.apiGet('/system/info')
  } catch {}
  sysLoading.value = false
})
</script>

<style scoped>
.modal-overlay {
  position: fixed; top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.5); z-index: 2000;
  display: flex; align-items: center; justify-content: center;
}
.modal-content {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  width: 520px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0,0,0,0.4);
}
.modal-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 20px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}
.modal-header h2 {
  font-size: 16px; font-weight: 600;
  display: flex; align-items: center; gap: 8px;
}
.modal-close {
  background: none; border: none;
  color: var(--text-secondary);
  font-size: 22px; cursor: pointer;
  width: 28px; height: 28px;
  display: flex; align-items: center; justify-content: center;
  border-radius: 4px;
}
.modal-close:hover { background: var(--bg-hover); color: var(--text-primary); }

.modal-body {
  flex: 1; overflow-y: auto; padding: 20px;
}
.modal-footer {
  display: flex; align-items: center; justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--border-color);
  flex-shrink: 0;
}

/* Hero */
.about-hero {
  text-align: center;
  padding: 16px 0 20px;
}
.about-logo {
  margin-bottom: 8px;
}
.about-title {
  font-size: 22px; font-weight: 700;
  color: var(--text-primary);
}
.about-version {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 4px;
}

/* 描述 */
.about-section {
  margin: 16px 0;
}
.about-description {
  font-size: 14px;
  line-height: 1.7;
  color: var(--text-secondary);
}
.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
  padding-bottom: 4px;
  border-bottom: 1px solid var(--border-color);
}

/* 特性列表 */
.feature-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.feature-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 0;
  font-size: 13px;
  color: var(--text-secondary);
}

/* 技术栈 */
.tech-stack {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.tech-badge {
  background: var(--accent-bg);
  color: var(--accent-light);
  border: 1px solid var(--accent);
  padding: 3px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

/* 系统信息 */
.sys-info {
  font-size: 13px;
}
.info-row {
  display: flex;
  padding: 5px 0;
  border-bottom: 1px solid var(--border-color);
}
.info-label {
  width: 80px;
  color: var(--text-muted);
  flex-shrink: 0;
}
.info-path {
  word-break: break-all;
  font-family: var(--font-code);
  font-size: 12px;
}
.loading-info {
  color: var(--text-muted);
  font-size: 13px;
}

/* 按钮 */
.btn-primary {
  display: flex; align-items: center; gap: 6px;
  background: var(--accent);
  border: 1px solid var(--accent);
  color: #000;
  padding: 7px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
}
.btn-primary:hover { filter: brightness(1.1); }
.btn-secondary {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 7px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
}
.btn-secondary:hover { background: var(--bg-hover); }
</style>
