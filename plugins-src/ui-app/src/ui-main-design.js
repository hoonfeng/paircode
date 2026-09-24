// ui-main-design.js — tool-design 插件「设计」面板 bundle 入口
// ★ 2026-09-17 合并：本面板原属独立包 ui-design，现与工具面（tool-design/index.js）同包；
// 编译为 IIFE 产物 plugins-dist/tool-design/assets/design-panel.js（window.DesignPanel），
// 由该包 client.js 注册 ui.registerPanel/ui.registerView 后调用 mount(el) 挂载。
// 数据源与 Agent 完全相同：design.tokens.json / design.project.json / design.html / design.verify.json。
import { createApp } from 'vue'
import DesignPanel from './components/DesignPanel.vue'

export function mount(el) {
  const app = createApp(DesignPanel)
  app.mount(el)
  return () => { app.unmount() }
}
