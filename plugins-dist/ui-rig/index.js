// ui-rig — host 半（纯 UI 插件，host 侧无逻辑）
//
// 装配链路：<InstallDir>/.pair/plugins/ 启动扫描 → define + load → /api/plugins 下发 clientCode →
// 前端 plugin-runtime.js syncClientHalves 执行 client 半 → ui.registerPanel({id:'rig-panel'})
// → PluginPanel.vue 渲染（点击面板项 → render(el) 挂载 Vue 应用）。
//
// 数据面：面板不新增 host 接口，直接读 tool-rig 的工程 / 预览产物与旁挂校验报告
// （rig.model.iki.json / rig.preview.html / rig.verify.json），经 GET /api/fs/read 取内容
// —— 与 Agent 看到的是同一份真相源（上游 .iki v1 契约）。
//
// 写路径纪律：面板只读工作区（部件/骨骼/绑定编辑一律经 Agent 的 rig_edit 命令式 op）；
// 唯一的"产出"动作是复制命令文本，不写回工作区，避免「工具写工程 / 面板写工程」真相源分叉。
return {
  name: 'ui-rig',
  purpose: '2D 角色面板：可交互预览 / 骨骼树 / 部件与参数 / 物理 / 9 项校验',
  apply(ctx) {
    // client 半负责渲染（见 client.js）
  },
}
