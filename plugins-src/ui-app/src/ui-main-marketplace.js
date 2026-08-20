// ui-main-marketplace.js — marketplace 插件市场面板 bundle 入口（2026-08-20）
// 市场功能全插件化：UI 与市场接口同插件（marketplace）——编译为独立 IIFE 产物
// .pair/plugins/marketplace/assets/marketplace-panel.js，client.js 注入后
// Sidebar（activeActivity==='marketplace'）动态挂载（与 git-api 的 GitPanel 同模式）。
import { createApp } from 'vue'
import MarketplacePanel from './components/MarketplacePanel.vue'

export function mount(el) {
  const app = createApp(MarketplacePanel)
  app.mount(el)
  return () => { app.unmount() }
}
