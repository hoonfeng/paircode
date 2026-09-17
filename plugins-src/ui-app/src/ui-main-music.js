// ui-main-music.js — ui-music 插件「音乐」面板 bundle 入口
// 编译为独立 IIFE 产物 plugins-dist/ui-music/assets/music-panel.js（window.MusicPanel），
// client.js 注册 ui.registerPanel 后由 PluginPanel 调用 mount(el) 挂载。
// 数据源与 Agent 完全相同：tool-music 的 music.project.json 与旁挂 music.verify.json。
import { createApp } from 'vue'
import MusicPanel from './components/MusicPanel.vue'

export function mount(el) {
  const app = createApp(MusicPanel)
  app.mount(el)
  return () => { app.unmount() }
}
