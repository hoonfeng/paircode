// ui-music — host 半（纯 UI 插件，host 侧无逻辑）
//
// 装配链路：<InstallDir>/.pair/plugins/ 启动扫描 → define + load →
// /api/plugins 下发 clientCode → 前端 plugin-runtime.js syncClientHalves 执行
// client 半 → ui.registerPanel({id:'music-panel'}) 注册进「插件面板」区
// → PluginPanel.vue 渲染（点击面板项 → render(el) 挂载 Vue 应用）。
//
// 数据面：面板不新增 host 接口，直接读 tool-music 的工程文件与旁挂校验报告
// （music.project.json / music.verify.json），经 GET /api/fs/read 取内容 ——
// 与 Agent 看到的是**同一份真相源**。
//
// 写路径纪律：面板只读（音符编辑一律经 Agent 的 music_edit 命令式 op），
// 避免出现「工具写工程 / 面板写工程」两条写路径导致真相源分叉。
return {
  name: 'ui-music',
  purpose: '音乐面板：工程信息/五线谱/钢琴卷帘/试听/轨道表/7 项校验结果',
  apply(ctx) {
    // client 半负责渲染（见 client.js）
  },
}
