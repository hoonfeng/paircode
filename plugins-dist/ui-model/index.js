// ui-model — host 半（纯 UI 插件，host 侧无逻辑）
//
// 装配链路：<InstallDir>/.pair/plugins/ 启动扫描 → define + load → /api/plugins 下发 clientCode →
// 前端 plugin-runtime.js syncClientHalves 执行 client 半 → ui.registerPanel({id:'model-panel'})
// → PluginPanel.vue 渲染（点击面板项 → render(el) 挂载 Vue 应用）。
//
// 数据面：面板不新增 host 接口，直接读 tool-model 的工程 / 预览产物 / 旁挂判据报告
// （model.json / model.preview.html / model.verify.json），经 GET /api/fs/read 取内容
// —— 与 Agent 看到的是同一份真相源（工程即模型，预览即可打印性的一手证据）。
//
// 写路径纪律：面板只读工作区（几何/参数/布尔全部经 Agent 的 model_add / model_edit）；
// 唯一的"动作"是复制命令文本，不写回工作区，避免「工具写工程 / 面板写工程」真相源分叉。
return {
  name: 'ui-model',
  purpose: '3D/CAD 面板：3D 预览 / 部件表 / 参数 / 9 项判据 / 产物清单',
  apply(ctx) {
    // client 半负责渲染（见 client.js）
  },
}
