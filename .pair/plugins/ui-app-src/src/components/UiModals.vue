<template>
  <SettingsModal v-if="showSettings.value" @close="showSettings.value = false" />
  <SystemModal v-if="showSystem.value" @close="showSystem.value = false" />
  <SourceModal v-if="showSource.value" @close="showSource.value = false" />
  <MarketplaceModal v-if="showMarketplace.value" @close="showMarketplace.value = false" />
  <HelpModal v-if="showHelp.value" @close="showHelp.value = false" @openAbout="onHelpOpenAbout" :initialDoc="helpDocTarget.value" />
  <AboutModal v-if="showAbout.value" @close="showAbout.value = false" @openHelp="onAboutOpenHelp" />
  <GlobalDialogs />
  <!-- ★ overlay 槽位（list 型）：插件注册的浮动层条目叠加渲染（badge/toast/status pill 等） -->
  <div ref="overlaySlotEl" class="plugin-overlay-host"></div>
</template>

<script setup>
// UiModals — modals 槽位默认实现（ui-modals 插件承载）。
// 从原 App.vue 模态框部分拆出：6 个全局模态框 + GlobalDialogs + overlay 叠加槽位。
// 开关状态来自共享 ui-state（titlebar 菜单 / activitybar 设置钮打开）。
import { ref, onMounted, onUnmounted } from 'vue'
import {
  showSettings, showSystem, showSource, showMarketplace,
  showHelp, showAbout, helpDocTarget,
} from '../ui-state.js'
import { mountListSlot, isOverlayActive } from '../plugin-runtime.js'
import SettingsModal from './SettingsModal.vue'
import SystemModal from './SystemModal.vue'
import SourceModal from './SourceModal.vue'
import MarketplaceModal from './MarketplaceModal.vue'
import HelpModal from './HelpModal.vue'
import AboutModal from './AboutModal.vue'
import GlobalDialogs from './GlobalDialogs.vue'

const overlaySlotEl = ref(null)
let overlayUnsub = null

function onAboutOpenHelp() {
  showAbout.value = false
  showHelp.value = true
  helpDocTarget.value = 'getting-started'
}
function onHelpOpenAbout() {
  showHelp.value = false
  showAbout.value = true
}

onMounted(() => {
  overlayUnsub = mountListSlot(overlaySlotEl, 'overlay', { isActive: n => isOverlayActive('overlay', n) })
})
onUnmounted(() => {
  if (overlayUnsub) { overlayUnsub(); overlayUnsub = null }
})
</script>

<style scoped>
/* overlay 槽位（list 型）：浮动层，条目叠加（badge/toast/status pill），不挡交互 */
.plugin-overlay-host { position: fixed; top: 0; left: 0; right: 0; bottom: 0; pointer-events: none; z-index: 9999; }
.plugin-overlay-item { pointer-events: auto; }
</style>
