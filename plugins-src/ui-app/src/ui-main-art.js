// ui-main-art.js — tool-art 插件「画板」面板 bundle 入口
// ★ 2026-09-17 合并：本面板原属独立包 ui-art，现与工具面（tool-art/index.js）同包；
// 编译为 IIFE 产物 plugins-dist/tool-art/assets/art-panel.js（window.ArtPanel），
// 由该包 client.js 注册 ui.registerPanel/ui.registerView 后调用 mount(el) 挂载。
// 数据源与 Agent 完全相同：tool-art 的 art.project.json 与旁挂 art.verify.json。
import { createApp } from 'vue'
import ArtPanel from './components/ArtPanel.vue'

export function mount(el) {
  const app = createApp(ArtPanel)
  app.mount(el)
  return () => { app.unmount() }
}
