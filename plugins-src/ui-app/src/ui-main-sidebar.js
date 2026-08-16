// ui-main-sidebar.js — sidebar 区域 bundle 入口（ui-sidebar 插件承载）
import { createApp } from 'vue'
import Sidebar from './components/Sidebar.vue'

export function mount(el) {
  const app = createApp(Sidebar)
  app.mount(el)
  return () => { app.unmount() }
}
