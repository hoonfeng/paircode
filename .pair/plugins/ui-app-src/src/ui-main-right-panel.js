// ui-main-right-panel.js — right-panel 区域 bundle 入口（ui-right-panel 插件承载）
import { createApp } from 'vue'
import UiRightPanel from './components/UiRightPanel.vue'

export function mount(el) {
  const app = createApp(UiRightPanel)
  app.mount(el)
  return () => { app.unmount() }
}
