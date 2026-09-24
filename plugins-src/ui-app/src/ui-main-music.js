// ui-main-music.js — tool-music 插件「音乐」面板 bundle 入口
// ★ 2026-09-17 合并：本面板原属独立包 ui-music，现与工具面（tool-music/index.js）同包；
// 编译为 IIFE 产物 plugins-dist/tool-music/assets/music-panel.js（window.MusicPanel），
// 由该包 client.js 注册 ui.registerPanel/ui.registerView 后调用 mount(el) 挂载。
// 数据源与 Agent 完全相同：tool-music 的 music.project.json 与旁挂 music.verify.json。
import { createApp } from 'vue'
import MusicPanel from './components/MusicPanel.vue'

export function mount(el) {
  const app = createApp(MusicPanel)
  app.mount(el)
  return () => { app.unmount() }
}
