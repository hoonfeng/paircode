// ui-voice — host 半（纯 UI 插件，host 侧无逻辑）
//
// 装配链路：<InstallDir>/.pair/plugins/ 启动扫描 → define + load →
// /api/plugins 下发 clientCode → 前端 plugin-runtime.js syncClientHalves 执行
// client 半 → ui.registerPanel({id:'voice-panel'}) 注册进「插件面板」区
// → PluginPanel.vue 渲染（点击面板项 → render(el) 挂载 Vue 应用）。
//
// 数据面：面板不新增 host 接口，直接读 tool-voice 的工程文件与旁挂文件
// （voice.project.json / *.f0.json / *.wave.json / *.recipe.json），
// 经 GET /api/fs/read 取内容 —— 与 Agent 看到的是**同一份真相源**。
return {
  name: 'ui-voice',
  purpose: '人声面板：源素材/波形/F0 曲线/音符段/切片/编辑链/渲染产物/六项自检结果',
  apply(ctx) {
    // client 半负责渲染（见 client.js）
  },
}
