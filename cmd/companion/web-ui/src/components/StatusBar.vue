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
    <!-- ★ statusbar-items 槽位（list 型）：内置状态栏内细粒度叠加条目（插件加小状态/快捷入口） -->
    <div ref="statusItemsEl" class="plugin-slot-host plugin-slot-status-items"></div>
    <div class="status-right">
      <span class="status-item" v-if="state.activeFile">
        <SvgIcon name="file-code" :size="12" />
        {{ displayPath }}
      </span>
      <span class="status-item" v-if="state.openFiles.length > 0">Ln {{ state.cursorLine }}, Col {{ state.cursorCol }}</span>
      <span class="status-item">UTF-8</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { state } from '../main.js'
import SvgIcon from './SvgIcon.vue'
import api from '../api.js'
import { mountListSlot } from '../plugin-runtime.js'

const gitBranch = ref('')
const gitChanges = ref(0)
const statusItemsEl = ref(null)
let statusItemsUnsub = null
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
  // statusbar-items 槽位（list 型）：状态栏内细粒度叠加。
  // ★ 连接状态指示已迁移为磁盘插件 ui-statusbar-conn（.pair/plugins/），
  //   不再内置——前端经 /api/plugins 装载 client 半后由插件渲染。
  statusItemsUnsub = mountListSlot(statusItemsEl, 'statusbar-items')
  // Load git info
  await loadGitInfo()
  gitTimer = setInterval(loadGitInfo, 15000)
})

onUnmounted(() => {
  if (gitTimer) clearInterval(gitTimer)
  if (statusItemsUnsub) { statusItemsUnsub(); statusItemsUnsub = null }
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
/* statusbar-items 槽位（list 型）：状态栏中间叠加区，与左右信息同行 */
.plugin-slot-status-items {
  display: flex;
  align-items: center;
  gap: 8px;
  height: auto;
  margin: 0 auto;
}
.plugin-slot-status-items .plugin-slot-item {
  display: flex;
  align-items: center;
  opacity: 0.9;
  font-size: 11px;
  gap: 4px;
}
.plugin-slot-status-items .plugin-slot-item:hover { opacity: 1; }
.status-item { opacity: 0.9; display: flex; align-items: center; gap: 4px; }
.status-item:hover { opacity: 1; }
.git-branch-item, .git-status-icons { cursor: pointer; }
.git-branch-item:hover, .git-status-icons:hover { text-decoration: underline; }
</style>
