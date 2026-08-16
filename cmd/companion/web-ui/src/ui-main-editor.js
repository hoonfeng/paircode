// ui-main-editor.js — editor 区域 bundle 入口（ui-editor 插件承载）
import { createApp } from 'vue'
import UiEditor from './components/UiEditor.vue'

export function mount(el) {
  const app = createApp(UiEditor)
  app.mount(el)
  return () => { app.unmount() }
}
