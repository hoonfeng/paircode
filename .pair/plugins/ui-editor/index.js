// ui-editor — host 半（纯 UI 插件：EditorArea + 底部终端面板 + 拖拽 resizer）
// ★ 配置分散化（2026-08-19）：编辑器设置 + 终端设置由本插件注册
//   （配置项对应本插件的功能区域；前端设置面板纯 schema 驱动渲染）
// 装配链路：<InstallDir>/.pair/plugins/ 启动扫描 → define + load →
// /api/plugins 下发 → 前端 plugin-runtime.js syncClientHalves 装载 client 半
// → ui.registerSlot('editor') 占用 → 壳 ShellApp 渲染到 editor 槽位。
return {
  name: 'ui-editor',
  purpose: 'EditorArea + 底部终端面板 + 拖拽 resizer（含编辑器/终端配置注册）',
  apply(ctx) {
    // ── 编辑器设置（binding → AppSettings 顶层）──
    ctx.registerSettings({
      key: 'editor',
      title: '编辑器',
      fields: [
        { name: 'tabSize', label: '制表符宽度', type: 'number', binding: 'tabSize', min: 1, max: 8 },
        { name: 'wordWrap', label: '自动换行', type: 'checkbox', binding: 'wordWrap' },
        { name: 'hideMinimap', label: '隐藏缩略图', type: 'checkbox', binding: 'hideMinimap' },
        { name: 'fontFamily', label: '字体', type: 'text', binding: 'fontFamily', group: '字体风格',
          placeholder: 'Cascadia Code, Consolas, monospace' },
        { name: 'editorFontSize', label: '字号', type: 'number', binding: 'editorFontSize', min: 10, max: 28, group: '字体风格' },
        { name: 'editorFontBold', label: '粗体', type: 'checkbox', binding: 'editorFontBold', group: '字体风格' },
        { name: 'editorFontItalic', label: '斜体', type: 'checkbox', binding: 'editorFontItalic', group: '字体风格' },
        { name: 'editorFontUnderline', label: '下划线', type: 'checkbox', binding: 'editorFontUnderline', group: '字体风格' },
      ],
    })

    // ── 终端设置（本插件管底部终端面板）──
    ctx.registerSettings({
      key: 'terminal',
      title: '终端',
      fields: [
        { name: 'defaultShell', label: '默认 Shell', type: 'text', binding: 'defaultShell',
          placeholder: 'auto', hint: 'auto=系统默认；或 powershell/cmd/bash 路径' },
        { name: 'termFontSize', label: '字号', type: 'number', binding: 'termFontSize', min: 10, max: 24 },
        { name: 'termEncoding', label: '编码', type: 'select', binding: 'termEncoding',
          options: ['auto', 'utf-8', 'gbk'] },
      ],
    })

    // client 半负责渲染（见 client.js）
  },
}
