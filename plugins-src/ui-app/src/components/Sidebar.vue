<template>
  <div class="sidebar" :style="{ width: sidebarWidth + 'px' }">
    <div class="sidebar-header">
      <span>{{ headerTitle }}</span>
    </div>
    <div class="sidebar-content">
      <FileExplorer v-if="state.activeActivity === 'explorer'" />
      <SearchPanel v-else-if="state.activeActivity === 'search'" />
      <!-- Git 源代码管理面板：由 git-api 插件加载 bundle 到 window.GitPanel，
           本组件动态挂载（跨 bundle，不能静态 import） -->
      <div v-else-if="state.activeActivity === 'source'" ref="gitHost" class="git-host"></div>
      <!-- 市场面板：由 marketplace 插件加载 bundle 到 window.MarketplacePanel，
           本组件动态挂载（与 GitPanel 同模式） -->
      <div v-else-if="state.activeActivity === 'marketplace'" ref="marketHost" class="market-host"></div>
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
import FileExplorer from './FileExplorer.vue'
import SearchPanel from './SearchPanel.vue'
import PluginPanel from './PluginPanel.vue'

const headerTitle = computed(() => {
  const titles = { explorer: '文件浏览器', search: '搜索', source: '源代码管理', marketplace: '市场', plugins: '插件' }
  return titles[state.activeActivity] || ''
})

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
  if (a === 'marketplace') {
    nextTick(mountMarketPanel)
  } else {
    unmountMarketPanel()
  }
})

onUnmounted(() => {
  unmountGitPanel()
  unmountMarketPanel()
})

// ─── 市场面板动态挂载（marketplace 插件 bundle → window.MarketplacePanel）───
// 2026-08-20：市场功能全插件化——面板 bundle 由 marketplace 插件 client 半注入，
// 本组件只负责在 activeActivity==='marketplace' 时取 window.MarketplacePanel 挂载。
const marketHost = ref(null)
let marketUnmount = null
let marketRetryTimer = null

function mountMarketPanel() {
  const el = marketHost.value
  if (!el) return
  el.innerHTML = ''
  const mod = window.MarketplacePanel
  if (mod && typeof mod.mount === 'function') {
    try {
      marketUnmount = mod.mount(el)
      return
    } catch (e) {
      console.warn('[sidebar] 市场面板挂载失败', e)
      el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">挂载失败: ' + (e && e.message || e) + '</div>'
      return
    }
  }
  if (marketRetryTimer) return
  let tries = 0
  marketRetryTimer = setInterval(() => {
    tries++
    if (window.MarketplacePanel) {
      clearInterval(marketRetryTimer); marketRetryTimer = null
      mountMarketPanel()
      return
    }
    if (tries >= 8) {
      clearInterval(marketRetryTimer); marketRetryTimer = null
      el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">市场面板未就绪（marketplace 插件未启用）</div>'
    }
  }, 800)
  el.innerHTML = '<div style="padding:12px;font-size:12px;color:var(--text-muted)">市场面板加载中...</div>'
}

function unmountMarketPanel() {
  if (marketRetryTimer) { clearInterval(marketRetryTimer); marketRetryTimer = null }
  if (marketUnmount) { try { marketUnmount() } catch (e) {} marketUnmount = null }
}

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
