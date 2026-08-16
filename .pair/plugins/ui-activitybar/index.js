// ui-activitybar — host 半（纯 UI 插件，host 侧无逻辑）
// 装配链路：<InstallDir>/.pair/plugins/ 启动扫描 → define + load →
// /api/plugins 下发 → 前端 plugin-runtime.js syncClientHalves 装载 client 半
// → ui.registerSlot('activitybar') 占用 → 壳 ShellApp 渲染到 activitybar 槽位。
return {
  name: 'ui-activitybar',
  purpose: '活动栏竖列（左侧图标导航 + activitybar 叠加挂载）',
  apply(ctx) {
    // client 半负责渲染（见 client.js）
  },
}
