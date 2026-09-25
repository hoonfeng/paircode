<template>
  <div class="sidebar" :style="{ width: sidebarWidth + 'px' }">
    <div class="sidebar-header">
      <span>{{ headerTitle }}</span>
    </div>
    <div class="sidebar-content">
      <!-- ★ 2026-09-25 对齐设计稿 shell-midnight th61：左栏（264px）默认 = 会话列表
           （标题「会话」/ 分组「今天·更早」/ 条目含消息数）。数据与动作全部走全局
           state + api：切换会话只改 state.currentConvId，由 RightPanel 的 watch
           负责加载消息与统计（不再复制其 switchConv 逻辑）。 -->
      <ConvSidebar v-if="state.activeActivity === 'chat'"
        :conversations="state.conversations"
        :current-conv-id="state.currentConvId"
        :loading-by-conv="state.loadingByConv"
        :ws-token-stats="wsTokenStats"
        :conv-ctx-stats="convCtxStats"
        :ctx-max-tokens-val="ctxMaxTokens"
        :width="sidebarWidth"
        @new-conversation="onNewConv"
        @switch-conversation="onSwitchConv"
        @delete-conversation="onDeleteConv" />
      <FileExplorer v-else-if="state.activeActivity === 'explorer'" />
      <SearchPanel v-else-if="state.activeActivity === 'search'" />
      <!-- Git 源代码管理面板：由 git-api 插件加载 bundle 到 window.GitPanel，
           本组件动态挂载（跨 bundle，不能静态 import） -->
      <div v-else-if="state.activeActivity === 'source'" ref="gitHost" class="git-host"></div>
      <!-- 市场面板已迁至主内容区 tab（2026-09）：不再于侧边栏显示 -->
      <PluginPanel v-else-if="state.activeActivity === 'plugins'" />
      <div v-else class="sidebar-placeholder">
        <span>面板加载中...</span>
      </div>
    </div>
    <!-- 拖拽分隔条（放在 Sidebar 内，绝对定位在右侧边缘） -->
    <div class="sidebar-resizer" @mousedown.prevent="startResize"></div>
  </div>
</template>

<script setup>
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import { state, sidebarWidth } from '../ui-state.js'
import api from '../api.js'
import { getConvCtxStats } from '../agent-events.js'
import FileExplorer from './FileExplorer.vue'
import SearchPanel from './SearchPanel.vue'
import PluginPanel from './PluginPanel.vue'
import ConvSidebar from './ConvSidebar.vue'

const headerTitle = computed(() => {
  const titles = { chat: '会话', explorer: '文件浏览器', search: '搜索', source: '源代码管理', marketplace: '市场', plugins: '插件' }
  return titles[state.activeActivity] || ''
})

// ─── 会话列表（设计稿 th61）数据/动作：与 RightPanel 同源，避免状态分叉 ───
// ★ 上下文上限 = 后端装配口径真值（/conversations/{id}/token-stats 的 contextMaxTokens
//   字段，即 agent.ContextWindow：服务商级 models.json > 机制常量 DefaultContextWindow）。
//   不再读 settings 顶层 contextMaxTokens（2026-09-20 已迁入插件注册域、不参与取值），
//   也不再 `|| 1000000` 硬编码兜底。后端正常总会给出 > 0 的真值。
const ctxMaxTokens = computed(() => (state.convCtxStatsByConv[state.currentConvId]
  && state.convCtxStatsByConv[state.currentConvId].contextMaxTokens) || 0)
const wsTokenStats = computed(() => state.wsTokenStatsByWs[state.workspaceRoot] || {
  totalTokens: 0, promptTokens: 0, completionTokens: 0,
  cacheHitTokens: 0, cacheMissTokens: 0, systemTokens: 0, skillsTokens: 0,
  mcpTokens: 0, toolTokens: 0, historyTokens: 0, otherTokens: 0,
})
const convCtxStats = computed(() => getConvCtxStats(state.currentConvId))

// 切换会话：只改 currentConvId —— RightPanel watch 会加载消息/统计并同步 currentTasks
function onSwitchConv(id) {
  if (id && id !== state.currentConvId) state.currentConvId = id
}

async function onNewConv() {
  try {
    const conv = await api.apiPost('/conversations', { title: '新对话', workspaceRoot: state.workspaceRoot })
    if (!conv || !conv.id) return
    state.conversations.unshift({ id: conv.id, title: conv.title || '新对话', msgCount: 0, createdAt: conv.createdAt, updatedAt: conv.updatedAt })
    if (!state.messagesByConv[conv.id]) state.messagesByConv[conv.id] = []
    state.currentConvId = conv.id
  } catch (e) {
    window.$toast && window.$toast('新建会话失败：' + (e && e.message ? e.message : e), 'error')
  }
}

async function onDeleteConv(id) {
  try {
    await api.apiDelete('/conversations/' + id)
    state.conversations = state.conversations.filter(c => c.id !== id)
    delete state.messagesByConv[id]
    delete state.convCtxStatsByConv[id]
    delete state.loadingByConv[id]
    window.dispatchEvent(new Event('save-conversations'))
    if (state.currentConvId === id) state.currentConvId = ''
  } catch (e) {
    window.$toast && window.$toast('删除会话失败：' + (e && e.message ? e.message : e), 'error')
  }
}

// ─── Git 面板动态挂载（git-api 插件 bundle → window.GitPanel）───
// 2026-08-20：Git 面板从插件面板「客户端面板」区移出，改为活动栏 source 图标
// 打开的侧边栏独立面板。bundle 由 git-api 插件 client 半注入（插件停用即消失），
// 本组件只负责在 activeActivity==='source' 时取 window.GitPanel 挂载。
const gitHost = ref(null)
let gitUnmount = null
let gitRetryTimer = null

function mountGitPanel() {
  const el = gitHost.value
  if (!el) return
  el.innerHTML = ''
  const mod = window.GitPanel
  if (mod && typeof mod.mount === 'function') {
    try {
      gitUnmount = mod.mount(el)
      return
    } catch (e) {
      console.warn('[sidebar] Git 面板挂载失败', e)
      el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">挂载失败: ' + (e && e.message || e) + '</div>'
      return
    }
  }
  // bundle 未就绪（git-api 插件未启用/正在加载）：提示 + 短暂自动重试
  if (gitRetryTimer) return
  let tries = 0
  gitRetryTimer = setInterval(() => {
    tries++
    if (window.GitPanel) {
      clearInterval(gitRetryTimer); gitRetryTimer = null
      mountGitPanel()
      return
    }
    if (tries >= 8) {
      clearInterval(gitRetryTimer); gitRetryTimer = null
      el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">Git 面板未就绪（git-api 插件未启用）</div>'
    }
  }, 800)
  el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">Git 面板加载中...</div>'
}

function unmountGitPanel() {
  if (gitRetryTimer) { clearInterval(gitRetryTimer); gitRetryTimer = null }
  if (gitUnmount) { try { gitUnmount() } catch (e) {} gitUnmount = null }
}

watch(() => state.activeActivity, (a) => {
  if (a === 'source') {
    nextTick(mountGitPanel)
  } else {
    unmountGitPanel()
  }
})

onUnmounted(() => {
  unmountGitPanel()
})

let dragging = false
let startX = 0
let startW = 0

function startResize(e) {
  dragging = true
  startX = e.clientX
  startW = sidebarWidth.value
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'ew-resize'
  document.body.style.userSelect = 'none'
}

function onMove(e) {
  if (!dragging) return
  sidebarWidth.value = Math.max(120, Math.min(800, startW + (e.clientX - startX)))
}

function stopResize() {
  dragging = false
  document.removeEventListener('mousemove', onMove)
  document.removeEventListener('mouseup', stopResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  try {
    localStorage.setItem('paircode-sidebar-width', String(sidebarWidth.value))
  } catch {}
}
</script>

<style scoped>
.sidebar {
  background: var(--sidebar-bg);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
  /* ★ bundle 根撑满宿主：无 height:100% 时高度=内容（FileExplorer 1216px
     溢出窗口 → 工具集被裁到窗口外；短面板 → 底部空余）。 */
  height: 100%;
}
.sidebar-header {
  height: 32px;
  display: flex;
  align-items: center;
  padding: 0 12px;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}
.sidebar-content {
  flex: 1;
  overflow: auto;
}
.git-host { height: 100%; }

.sidebar-placeholder {
  padding: 20px;
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
}
.sidebar-resizer {
  position: absolute;
  right: -2px;
  top: 0;
  width: 6px;
  height: 100%;
  cursor: ew-resize;
  z-index: 10;
  background: transparent;
}
.sidebar-resizer:hover {
  background: var(--accent);
}
</style>
