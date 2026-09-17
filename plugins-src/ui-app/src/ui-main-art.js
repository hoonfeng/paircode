// ui-main-art.js — ui-art 插件「画板」面板 bundle 入口
// 编译为独立 IIFE 产物 plugins-dist/ui-art/assets/art-panel.js（window.ArtPanel），
// client.js 注册 ui.registerPanel 后由 PluginPanel 调用 mount(el) 挂载。
// 数据源与 Agent 完全相同：tool-art 的 art.project.json 与旁挂 art.verify.json。
import { createApp } from 'vue'
import ArtPanel from './components/ArtPanel.vue'

export function mount(el) {
  const app = createApp(ArtPanel)
  app.mount(el)
  return () => { app.unmount() }
}
