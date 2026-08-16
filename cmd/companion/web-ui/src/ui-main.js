// ═══════════════════════════════════════════════════════════════
// ui-main.js — UI bundle 入口（由 vite.ui.config.js 编译为 IIFE）
//
// ★ 2026-08-16 全 UI 插件化：整个 IDE UI（App.vue + 全部组件 + 状态）
//   由本入口提前编译为单个 IIFE bundle（window.__PAIRCODE_UI），
//   产物放 .pair/plugins/ui-app/assets/ui-app.js，ui-app 插件 client 半
//   加载后调用 window.__PAIRCODE_UI.mount(rootEl) 挂载。
//
//   - 运行时零编译：Vue SFC 模板已在构建期编译为 render 函数
//   - 与壳共享槽位/实例表：plugin-runtime.js 的 instances/clientSlots/
//     clientPanels 挂 window.__SLOT_REGISTRY，本 bundle 与壳各自打包该
//     模块时仍指向同一数组（外部插件与 bundle 内组件看到同一装配视图）
//   - 壳只负责装载 client 半与提供根容器，不包含任何 UI 实现
// ═══════════════════════════════════════════════════════════════
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'
import { applyTheme, loadPersistentState, state } from './ui-state.js'

// mount 挂载整个 IDE UI 到指定容器（ui-app 插件 client 半调用）。
// 返回 cleanup（卸载时调用）。
export function mount(rootEl) {
  if (!rootEl) return () => {}
  // 与原 main.js 顶层行为一致：先应用主题，再恢复持久化偏好
  applyTheme(state.theme || 'dark')
  loadPersistentState()
  const app = createApp(App)
  app.use(createPinia())
  app.use(router)
  app.mount(rootEl)
  return () => { app.unmount() }
}

// 全局入口：ui-app 插件 client 半通过 window.__PAIRCODE_UI.mount 挂载
if (typeof window !== 'undefined') {
  window.__PAIRCODE_UI = { mount }
}

export default { mount }
