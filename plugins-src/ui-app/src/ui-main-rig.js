// ui-main-rig.js — tool-rig 插件「角色」面板 bundle 入口
// ★ 2026-09-17 合并：本面板原属独立包 ui-rig，现与工具面（tool-rig/index.js）同包；
// 编译为 IIFE 产物 plugins-dist/tool-rig/assets/rig-panel.js（window.RigPanel），
// 由该包 client.js 注册 ui.registerPanel/ui.registerView 后调用 mount(el) 挂载。
// 数据源与 Agent 完全相同：rig.model.iki.json / rig.preview.html / rig.verify.json。
import { createApp } from 'vue'
import RigPanel from './components/RigPanel.vue'

export function mount(el) {
  const app = createApp(RigPanel)
  app.mount(el)
  return () => { app.unmount() }
}
