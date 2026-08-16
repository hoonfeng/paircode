// ═══════════════════════════════════════════════════════════════
// ui-statusbar-conn — host 半（纯 UI 插件，host 侧无逻辑）
//
// 迁移来源：web-ui StatusBar.vue 内置「连接状态」指示（已从内置模板迁出，
// 由本插件经 statusbar-items 槽位装配，前端无需重编译即可更新/替换）。
//
// 装配链路：<InstallDir>/.pair/plugins/ 启动扫描（LoadGlobalPlugins）
//   → define + load → /api/plugins 下发 → 前端 plugin-runtime.js
//   syncClientHalves 装载 client 半 → ui.registerSlot 占用槽位。
// ═══════════════════════════════════════════════════════════════
return {
  name: 'ui-statusbar-conn',
  purpose: '状态栏连接状态指示（statusbar-items 槽位）',
  apply(ctx) {
    // host 半无工具/服务需求；client 半负责渲染（见 client.js）
  },
}
