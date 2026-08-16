// ui-sidebar — host 半（纯 UI 插件，host 侧无逻辑）
// 装配链路：<InstallDir>/.pair/plugins/ 启动扫描 → define + load →
// /api/plugins 下发 → 前端 plugin-runtime.js syncClientHalves 装载 client 半
// → ui.registerSlot('sidebar') 占用 → 壳 ShellApp 渲染到 sidebar 槽位。
return {
  name: 'ui-sidebar',
  purpose: '文件/搜索/Git/插件 四面板侧栏',
  apply(ctx) {
    // client 半负责渲染（见 client.js）
  },
}
