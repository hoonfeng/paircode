// ui-main-modals.js — modals 区域 bundle 入口（ui-modals 插件承载）
// 渲染 6 个全局模态框 + GlobalDialogs + overlay 叠加槽位（fixed 浮层）。
import { createApp } from 'vue'
import UiModals from './components/UiModals.vue'

export function mount(el) {
  const app = createApp(UiModals)
  app.mount(el)
  return () => { app.unmount() }
}
