<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content about-modal">
      <div class="modal-header">
        <h2><SvgIcon name="info" :size="18" /> 关于 PairCode IDE</h2>
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
            PairCode IDE 是一款集成 AI 助手的现代化编程环境。
            以「对话即编程」为核心理念，将 AI 代码生成能力与 IDE 的编辑、调试、版本管理等功能无缝融合，
            让开发者通过自然语言与代码进行交互，显著提升开发效率。
          </p>
        </div>

        <!-- 特性亮点 -->
        <div class="about-section">
          <div class="section-title">主要特性</div>
          <ul class="feature-list">
            <li><SvgIcon name="bot" :size="14" color="var(--accent)" /> AI 对话编程 — 自然语言驱动代码生成与重构</li>
            <li><SvgIcon name="file" :size="14" color="var(--accent)" /> 多语言代码编辑器 — 基于 CodeMirror 6</li>
            <li><SvgIcon name="git-branch" :size="14" color="var(--accent)" /> Git 集成 — 可视化的版本管理</li>
            <li><SvgIcon name="terminal" :size="14" color="var(--accent)" /> 内建终端 — 无需离开 IDE 执行命令</li>
            <li><SvgIcon name="search" :size="14" color="var(--accent)" /> 智能搜索 — 全文搜索与代码导航</li>
            <li><SvgIcon name="settings" :size="14" color="var(--accent)" /> BUG 自动检测 — 自动发现并修复代码问题</li>
            <li><SvgIcon name="tool" :size="14" color="var(--accent)" /> 丰富工具链 — 文件处理/网络/截图/办公文档</li>
            <li><SvgIcon name="grid" :size="14" color="var(--accent)" /> Skills / MCP 扩展 — 可插拔的工具生态</li>
          </ul>
        </div>

        <!-- 技术栈 -->
        <div class="about-section">
          <div class="section-title">技术栈</div>
          <div class="tech-stack">
            <span class="tech-badge">Go</span>
            <span class="tech-badge">Vue 3</span>
            <span class="tech-badge">CodeMirror 6</span>
            <span class="tech-badge">GWui</span>
            <span class="tech-badge">WebSocket</span>
            <span class="tech-badge">MCP</span>
          </div>
        </div>

        <!-- 系统信息 -->
        <div class="about-section">
          <div class="section-title">系统信息</div>
          <div class="sys-info" v-if="!sysLoading">
            <div class="info-row"><span class="info-label">主机名</span><span>{{ sysInfo.hostname }}</span></div>
            <div class="info-row"><span class="info-label">操作系统</span><span>{{ sysInfo.os }}</span></div>
            <div class="info-row"><span class="info-label">工作区</span><span class="info-path">{{ sysInfo.workspace }}</span></div>
            <div class="info-row"><span class="info-label">Go 版本</span><span>{{ sysInfo.goos }}</span></div>
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

const version = ref('1.0.0')
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
