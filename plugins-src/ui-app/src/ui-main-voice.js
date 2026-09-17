// ui-main-voice.js — tool-voice 插件「人声」面板 bundle 入口
// ★ 2026-09-17 合并：本面板原属独立包 ui-voice，现与工具面（tool-voice/index.js）同包；
// 编译为 IIFE 产物 plugins-dist/tool-voice/assets/voice-panel.js（window.VoicePanel），
// 由该包 client.js 注册 ui.registerPanel/ui.registerView 后调用 mount(el) 挂载。
// 数据源与 Agent 相同：tool-voice 的 voice.project.json 与旁挂 f0/wave/recipe 文件。
import { createApp } from 'vue'
import VoicePanel from './components/VoicePanel.vue'

export function mount(el) {
  const app = createApp(VoicePanel)
  app.mount(el)
  return () => { app.unmount() }
}
