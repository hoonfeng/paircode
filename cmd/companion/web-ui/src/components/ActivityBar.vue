<template>
  <div class="activity-bar">
    <div class="activity-top">
      <button v-for="item in items" :key="item.id"
              :class="{ active: isActive(item.id), highlight: item.id === 'chat' && state.rightPanelVisible }"
              :title="item.label"
              @click="switchIt(item.id)">
        <SvgIcon :name="item.icon" :size="18" />
      </button>
      <!-- ★ activitybar 槽位（list 型）：插件在活动栏顶部叠加图标入口 -->
      <div ref="activitySlotEl" class="plugin-slot-host plugin-slot-activitybar"></div>
    </div>
    <div class="activity-bottom">
      <button title="设置" @click="switchIt('settings')">
        <SvgIcon name="settings" :size="18" />
      </button>
    </div>
  </div>
</template>

<script setup>
import { inject, onMounted, onUnmounted, ref } from 'vue'
import { state } from '../main.js'
import SvgIcon from './SvgIcon.vue'
import { mountListSlot } from '../plugin-runtime.js'

const switchActivity = inject('switchActivity')
const activitySlotEl = ref(null)
let activityUnsub = null

onMounted(() => {
  activityUnsub = mountListSlot(activitySlotEl, 'activitybar')
})
onUnmounted(() => {
  if (activityUnsub) { activityUnsub(); activityUnsub = null }
})

const items = [
  { id: 'explorer', label: '文件浏览器', icon: 'folder' },
  { id: 'search', label: '搜索', icon: 'search' },
  { id: 'source', label: '源代码管理', icon: 'source-control' },
  { id: 'plugins', label: '插件', icon: 'puzzle' },
  { id: 'marketplace', label: '市场', icon: 'package' },
  { id: 'chat', label: '对话', icon: 'chat' },
]

const isActive = (id) => {
  if (id === 'chat') return false
  return state.activeActivity === id
}

const switchIt = (id) => { switchActivity(id) }
</script>

<style scoped>
.activity-bar {
  width: 48px;
  background: var(--activity-bar-bg);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 4px 0;
}
.activity-top, .activity-bottom {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}
.activity-bottom { margin-top: auto; }
.activity-bar button {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  border-left: 2px solid transparent;
  cursor: pointer;
  color: var(--text-muted);
  transition: all 0.1s;
}
.activity-bar button:hover { color: var(--text-primary); }
.activity-bar button.active {
  color: var(--text-primary);
  border-left-color: var(--text-primary);
}
.activity-bar button.highlight {
  color: var(--accent-light);
  border-left-color: var(--accent);
}
/* activitybar 槽位（list 型）：插件图标入口与内置按钮同列，不撑满高度 */
.plugin-slot-activitybar {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  height: auto;
  margin-top: 4px;
}
.plugin-slot-activitybar .plugin-slot-item {
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
