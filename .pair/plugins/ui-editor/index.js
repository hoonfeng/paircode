// ui-editor — host 半（纯 UI 插件，host 侧无逻辑）
// 装配链路：<InstallDir>/.pair/plugins/ 启动扫描 → define + load →
// /api/plugins 下发 → 前端 plugin-runtime.js syncClientHalves 装载 client 半
// → ui.registerSlot('editor') 占用 → 壳 ShellApp 渲染到 editor 槽位。
return {
  name: 'ui-editor',
  purpose: 'EditorArea + 底部终端面板 + 拖拽 resizer',
  apply(ctx) {
    // client 半负责渲染（见 client.js）
  },
}
