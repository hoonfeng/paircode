// ui-editor — host 半（纯 UI 插件：EditorArea + 底部终端面板 + 拖拽 resizer）
// ★ 配置分散化（2026-08-19）：编辑器设置由本插件注册（binding → AppSettings 顶层）
//   · tabSize / fontSize：前端 EditorArea 组件实时消费（state.settings.tabSize/fontSize）
//   · 终端无配置注册（2026-08-19 清理：defaultShell/termFontSize/termEncoding 无消费方，
//     终端面板 shell 选择走 localStorage；保留 UI 行为，移除无效配置项）
// 装配链路：<InstallDir>/.pair/plugins/ 启动扫描 → define + load →
// /api/plugins 下发 → 前端 plugin-runtime.js syncClientHalves 装载 client 半
// → ui.registerSlot('editor') 占用 → 壳 ShellApp 渲染到 editor 槽位。
return {
  name: 'ui-editor',
  purpose: 'EditorArea + 底部终端面板 + 拖拽 resizer（含编辑器配置注册）',
  apply(ctx) {
    // ── 编辑器设置（binding → AppSettings 顶层）──
    ctx.registerSettings({
      key: 'editor',
      title: '编辑器',
      fields: [
        { name: 'tabSize', label: '制表符宽度', type: 'number', binding: 'tabSize', min: 1, max: 8 },
        // ★ 字段名对齐前端消费（EditorArea 读 state.settings.fontSize；旧注册名 editorFontSize 不生效）
        { name: 'fontSize', label: '字号', type: 'number', binding: 'fontSize', min: 10, max: 28 },
      ],
    })

    // client 半负责渲染（见 client.js）
  },
}
