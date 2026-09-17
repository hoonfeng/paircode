// ui-main-model.js — ui-model 插件「3D 模型」面板 bundle 入口
// 编译为独立 IIFE 产物 plugins-dist/ui-model/assets/model-panel.js（window.ModelPanel），
// client.js 注册 ui.registerPanel 后由 PluginPanel 调用 mount(el) 挂载。
// 数据源与 Agent 完全相同：model.json / model.preview.html / model.verify.json。
import { createApp } from 'vue'
import ModelPanel from './components/ModelPanel.vue'

export function mount(el) {
  const app = createApp(ModelPanel)
  app.mount(el)
  return () => { app.unmount() }
}
