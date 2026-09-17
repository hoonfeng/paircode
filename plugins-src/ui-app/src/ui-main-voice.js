// ui-main-voice.js — ui-voice 插件「人声」面板 bundle 入口
// 编译为独立 IIFE 产物 .pair/plugins/ui-voice/assets/voice-panel.js（window.VoicePanel），
// client.js 注册 ui.registerPanel 后由 PluginPanel 调用 mount(el) 挂载。
// 数据源与 Agent 相同：tool-voice 的 voice.project.json 与旁挂 f0/wave/recipe 文件。
import { createApp } from 'vue'
import VoicePanel from './components/VoicePanel.vue'

export function mount(el) {
  const app = createApp(VoicePanel)
  app.mount(el)
  return () => { app.unmount() }
}
