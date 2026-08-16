// ═══════════════════════════════════════════════════════════════
// main.js — 壳入口（全 UI 插件化后 web-ui 只保留壳）
//
// ★ 2026-08-16：整个 IDE UI（App.vue + 全部组件 + 状态）已迁出为磁盘插件
//   ui-app（产物 .pair/plugins/ui-app/assets/ui-app.js，Vite 提前编译的
//   IIFE bundle）。本文件只做三件事：
//     1. createApp 挂载壳组件 ShellApp（app-root 槽位容器 + 空态）
//     2. ShellApp onMounted 装载 client 半（含 ui-app → 加载 UI bundle）
//     3. 主题 CSS 变量等全局资源仍在 index.html（壳与 UI bundle 共享）
//   原 UI 侧状态（state/dialogState/主题/持久化）已拆至 ./ui-state.js，
//   随 UI bundle 编译，壳不引用。
// ═══════════════════════════════════════════════════════════════
import { createApp } from 'vue'
import ShellApp from './ShellApp.vue'
// 确保 plugin-runtime 注册表初始化（window.__SLOT_REGISTRY）
import './plugin-runtime.js'

createApp(ShellApp).mount('#app')
