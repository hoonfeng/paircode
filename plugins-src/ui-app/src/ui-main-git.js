// ui-main-git.js — git-api 插件 Git 面板 bundle 入口（2026-08-19）
// Git 面板 UI 与 Git 接口同插件（git-api）：编译为独立 IIFE 产物
// .pair/plugins/git-api/assets/git-panel.js，client.js 动态加载后
// ui.registerPanel 渲染（UI 与接口一体化，卸载插件即消失）。
import { createApp } from 'vue'
import GitPanel from './components/GitPanel.vue'

export function mount(el) {
  const app = createApp(GitPanel)
  app.mount(el)
  return () => { app.unmount() }
}
