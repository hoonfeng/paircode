// ui-main-rig.js — ui-rig 插件「角色」面板 bundle 入口
// 编译为独立 IIFE 产物 plugins-dist/ui-rig/assets/rig-panel.js（window.RigPanel），
// client.js 注册 ui.registerPanel 后由 PluginPanel 调用 mount(el) 挂载。
// 数据源与 Agent 完全相同：rig.model.iki.json / rig.preview.html / rig.verify.json。
import { createApp } from 'vue'
import RigPanel from './components/RigPanel.vue'

export function mount(el) {
  const app = createApp(RigPanel)
  app.mount(el)
  return () => { app.unmount() }
}
