// ui-main-statusbar.js — statusbar 区域 bundle 入口（ui-statusbar 插件承载）
import { createApp } from 'vue'
import StatusBar from './components/StatusBar.vue'

export function mount(el) {
  const app = createApp(StatusBar)
  app.mount(el)
  return () => { app.unmount() }
}
