<template>
  <div class="status-bar">
    <div class="status-left">
      <span class="status-item" v-if="state.workspaceRoot">
        <SvgIcon name="home" :size="12" />
        {{ state.workspaceRoot.split('\\').filter(Boolean).pop() || '工作区' }}
      </span>
      <span class="status-item" v-else>未加载</span>
      <!-- Git 分支 -->
      <span class="status-item git-branch-item" v-if="gitBranch" @click="switchToGit">
        <SvgIcon name="git-branch" :size="11" />
        {{ gitBranch }}
      </span>
      <span class="status-item git-status-icons" v-if="gitChanges > 0" @click="switchToGit">
        <SvgIcon name="source-control" :size="11" />
        {{ gitChanges }}
      </span>
    </div>
    <div class="status-right">
      <span class="status-item" v-if="state.activeFile">
        <SvgIcon name="file-code" :size="12" />
        {{ displayPath }}
      </span>
      <span class="status-item" v-if="state.openFiles.length > 0">Ln {{ state.cursorLine }}, Col {{ state.cursorCol }}</span>
      <span class="status-item">UTF-8</span>
      <span class="status-item" :class="{ connected }">
        <span class="status-dot" :class="{ on: connected }"></span>
        {{ connected ? '已连接' : '断开' }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { state } from '../main.js'
import SvgIcon from './SvgIcon.vue'
import api from '../api.js'

const connected = ref(false)
const gitBranch = ref('')
const gitChanges = ref(0)
let gitTimer = null

const displayPath = computed(() => {
  const p = state.activeFile
  if (!p) return ''
  const parts = p.replace(/\\/g, '/').split('/')
  const name = parts.pop()
  if (parts.length > 2) return '.../' + parts.slice(-2).join('/') + '/' + name
  return parts.length > 0 ? parts.join('/') + '/' + name : name
})

async function loadGitInfo() {
  try {
    const res = await api.apiGet('/git/status')
    if (res.isRepo) {
      gitBranch.value = res.branch || ''
      gitChanges.value = (res.staged?.length || 0) + (res.modified?.length || 0) + (res.untracked?.length || 0)
    } else {
      gitBranch.value = ''
      gitChanges.value = 0
    }
  } catch {
    gitBranch.value = ''
    gitChanges.value = 0
  }
}

function switchToGit() {
  window.dispatchEvent(new CustomEvent('switch-activity', { detail: { id: 'source' } }))
}

onMounted(async () => {
  const check = async () => {
    try { const r = await fetch('/api/health'); connected.value = r.ok } catch { connected.value = false }
  }
  await check()
  setInterval(check, 30000)
  // Load git info
  await loadGitInfo()
  gitTimer = setInterval(loadGitInfo, 15000)
})

onUnmounted(() => {
  if (gitTimer) clearInterval(gitTimer)
})
</script>

<style scoped>
.status-bar {
  height: 22px;
  background: var(--accent);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 8px;
  font-size: 11px;
  color: var(--status-text);
}
.status-left, .status-right { display: flex; align-items: center; gap: 8px; }
.status-item { opacity: 0.9; display: flex; align-items: center; gap: 4px; }
.status-item:hover { opacity: 1; }
.git-branch-item, .git-status-icons { cursor: pointer; }
.git-branch-item:hover, .git-status-icons:hover { text-decoration: underline; }
.status-dot {
  display: inline-block; width: 8px; height: 8px; border-radius: 50%;
  background: rgba(255,255,255,0.4);
}
.status-dot.on { background: var(--status-text); }
</style>
