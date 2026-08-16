// ui-main-titlebar.js — titlebar 区域 bundle 入口（ui-titlebar 插件承载）
// 构建：IIFE 挂 window.__PAIRCODE_UI_TITLEBAR；external 共享核心从
// __PAIRCODE_CORE 取（Vue 单例/同一状态）。client.js loadScript 后调 mount。
import { createApp } from 'vue'
import UiTitlebar from './components/UiTitlebar.vue'

export function mount(el) {
  const app = createApp(UiTitlebar)
  app.mount(el)
  return () => { app.unmount() }
}
