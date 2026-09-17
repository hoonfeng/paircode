// ui-main-design.js — ui-design 插件「设计」面板 bundle 入口
// 编译为独立 IIFE 产物 plugins-dist/ui-design/assets/design-panel.js（window.DesignPanel），
// client.js 注册 ui.registerPanel 后由 PluginPanel 调用 mount(el) 挂载。
// 数据源与 Agent 完全相同：design.tokens.json / design.project.json / design.html / design.verify.json。
import { createApp } from 'vue'
import DesignPanel from './components/DesignPanel.vue'

export function mount(el) {
  const app = createApp(DesignPanel)
  app.mount(el)
  return () => { app.unmount() }
}
