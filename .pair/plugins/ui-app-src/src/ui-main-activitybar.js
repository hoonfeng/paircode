// ui-main-activitybar.js — activitybar 区域 bundle 入口（ui-activitybar 插件承载）
import { createApp } from 'vue'
import ActivityBar from './components/ActivityBar.vue'

export function mount(el) {
  const app = createApp(ActivityBar)
  app.mount(el)
  return () => { app.unmount() }
}
