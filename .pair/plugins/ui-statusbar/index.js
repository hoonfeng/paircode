// ui-statusbar — host 半（纯 UI 插件，host 侧无逻辑）
// 装配链路：<InstallDir>/.pair/plugins/ 启动扫描 → define + load →
// /api/plugins 下发 → 前端 plugin-runtime.js syncClientHalves 装载 client 半
// → ui.registerSlot('statusbar') 占用 → 壳 ShellApp 渲染到 statusbar 槽位。
return {
  name: 'ui-statusbar',
  purpose: '工作区/git/文件信息 + statusbar-items 叠加挂载',
  apply(ctx) {
    // client 半负责渲染（见 client.js）
  },
}
