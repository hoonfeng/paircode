// ═══════════════════════════════════════════════════════════════
// main.js — web-ui 壳入口（2026-08-16 按槽位细粒度拆分）
//
// 职责（壳只做三件事）：
//   1. 把共享核心挂 window.__PAIRCODE_CORE（vue/api/ui-state/plugin-runtime/
//      agent-events/app-actions）——7 个区域插件 bundle 全部 external 这些
//      模块，从全局取同一份（Vue 单例 / 同一 reactive state / 同一槽位注册表）。
//   2. createApp 壳组件（ShellApp：8 个区域槽位容器 + 装载器）。
//   3. 装载 client 半 → 各 ui-* 插件 registerSlot → 区域渲染。
//
// ★ 区域 bundle 构建：vite lib mode IIFE + external 上述模块（globals 映射
//   到 __PAIRCODE_CORE.xxx），见 scripts/build-ui.mjs。
// ═══════════════════════════════════════════════════════════════
import { createApp } from 'vue'
import * as Vue from 'vue'
import api from './api.js'
import * as uiState from './ui-state.js'
import * as pluginRuntime from './plugin-runtime.js'
import * as agentEvents from './agent-events.js'
import * as actions from './app-actions.js'
import ShellApp from './ShellApp.vue'

// ★ 共享核心：全部区域 bundle external 的模块从这里取（Vue 单例 + 状态单例）。
//   DSH 兼容契约（spec §4.3）：除 Vue/uiState/api/pluginRuntime/agentEvents/actions 外，
//   新增 `version` 锚（区域包兼容检测）与 `layout` 服务面（ctx.uiLayout，侧栏折叠/
//   编辑器按需打开的权威面）。保持单例关键：本对象里的引用必须与所有区域 bundle
//   运行时经 `window.__PAIRCODE_CORE` 取的为同一实例（Vue/reactive/槽位注册表跨副本共享）。
window.__PAIRCODE_CORE = {
  Vue,
  api,
  uiState,
  pluginRuntime,
  agentEvents,
  actions,
  layout: uiState.layout,    // ★ ctx.uiLayout 服务面（见 ui-state.js `layout`）
  version: '1.0.0',          // ★ 共享核心契约版本锚（region 包兼容检测）
}

createApp(ShellApp).mount('#app')
