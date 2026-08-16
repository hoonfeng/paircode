// ui-modals — host 半（纯 UI 插件，host 侧无逻辑）
// 装配链路：<InstallDir>/.pair/plugins/ 启动扫描 → define + load →
// /api/plugins 下发 → 前端 plugin-runtime.js syncClientHalves 装载 client 半
// → ui.registerSlot('modals') 占用 → 壳 ShellApp 渲染到 modals 槽位。
return {
  name: 'ui-modals',
  purpose: '6 个全局模态框 + GlobalDialogs + overlay 叠加挂载',
  apply(ctx) {
    // client 半负责渲染（见 client.js）
  },
}
