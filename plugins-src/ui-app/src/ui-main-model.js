// ui-main-model.js — tool-model 插件「3D 模型」面板 bundle 入口
// ★ 2026-09-17 合并：本面板原属独立包 ui-model，现与工具面（tool-model/index.js）同包；
// 编译为 IIFE 产物 plugins-dist/tool-model/assets/model-panel.js（window.ModelPanel），
// 由该包 client.js 注册 ui.registerPanel/ui.registerView 后调用 mount(el) 挂载。
// 数据源：host 半 clientMethod（listArtifacts/readArtifact/buildProject）+ 「识别到的文件」实时预览。
import { createApp } from 'vue'
import ModelPanel from './components/ModelPanel.vue'

// mount(el, opts)：opts.fill = true 时预览占满可用高度（主内容区 tab 场景；
// 插件面板里侧栏很窄则用固定高度，避免把面板撑爆）。
export function mount(el, opts) {
  const app = createApp(ModelPanel, { fill: !!(opts && opts.fill) })
  app.mount(el)
  return () => { app.unmount() }
}
