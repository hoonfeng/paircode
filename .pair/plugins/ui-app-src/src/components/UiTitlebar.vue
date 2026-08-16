<template>
  <div class="titlebar" @click="closeAllMenus">
    <div class="app-logo">
      <img :src="logoUrl" class="logo-img" alt="PairCode" />
    </div>
    <MenuBar ref="menuBarRef" />
    <div class="title-center">{{ state.workspaceName }}</div>
    <div class="title-right">
      <button v-if="wsList.length > 1" class="ws-quick-btn"
              @click="showQuickSwitcher = !showQuickSwitcher" title="快速切换工作区">
        <SvgIcon name="folder" :size="14" />
      </button>
      <!-- ★ titlebar-right 槽位（list 型）：标题栏右侧细粒度叠加（插件加小按钮/状态） -->
      <div ref="titlebarRightEl" class="plugin-slot-host plugin-slot-titlebar"></div>
    </div>
  </div>
</template>

<script setup>
// UiTitlebar — titlebar 槽位默认实现（ui-titlebar 插件承载）。
// 从原 App.vue 标题栏模板拆出：logo + MenuBar + 工作区标题 + titlebar-right 叠加槽位。
import { ref, onMounted, onUnmounted } from 'vue'
import { state, showQuickSwitcher } from '../ui-state.js'
import MenuBar from './MenuBar.vue'
import SvgIcon from './SvgIcon.vue'
import { mountListSlot } from '../plugin-runtime.js'
import logoUrl from '../assets/logo.svg'

const menuBarRef = ref(null)
const titlebarRightEl = ref(null)
let titlebarRightUnsub = null
const wsList = state.wsList

const closeAllMenus = () => { if (menuBarRef.value) menuBarRef.value.closeMenu?.() }

onMounted(() => {
  titlebarRightUnsub = mountListSlot(titlebarRightEl, 'titlebar-right')
})
onUnmounted(() => {
  if (titlebarRightUnsub) { titlebarRightUnsub(); titlebarRightUnsub = null }
})
</script>

<style scoped>
.titlebar {
  grid-column: 1 / -1; grid-row: 1;
  display: flex; align-items: center; height: 30px;
  background: var(--bg-tertiary);
  border-bottom: 1px solid var(--border-color);
  z-index: 100; overflow: visible;
  -webkit-app-region: drag;
}
.app-logo {
  width: 48px; display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  -webkit-app-region: no-drag;
}
.logo-img { width: 18px; height: 18px; }
.title-center {
  flex: 1; text-align: center; font-size: 12px; color: var(--text-muted);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap; padding: 0 8px;
}
.title-right {
  display: flex; align-items: center; padding-right: 8px; gap: 6px;
  -webkit-app-region: no-drag;
}
.ws-quick-btn {
  background: none; border: 1px solid var(--border-color); color: var(--text-secondary);
  padding: 2px 8px; border-radius: 3px; cursor: pointer; display: flex; align-items: center;
}
.ws-quick-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.plugin-slot-titlebar { display: flex; align-items: center; gap: 4px; height: auto; }
</style>
