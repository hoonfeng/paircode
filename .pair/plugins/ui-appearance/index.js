// ui-appearance — 外观设置插件（2026-08-19）
// ★ 配置分散化：主题/界面字体等外观配置由本插件注册（全局 UI 外观对应插件）
//   · theme → AppSettings.theme（前端实时应用）
//   · uiFont* → AppSettings 顶层（界面字体风格）
return {
  name: 'ui-appearance',
  purpose: '外观设置：主题与界面字体（配置注册化，设置面板纯 schema 驱动）',
  apply(ctx) {
    // ── 外观：主题 + 界面字体 ──
    ctx.registerSettings({
      key: 'appearance',
      title: '外观',
      fields: [
        { name: 'theme', label: '主题', type: 'select', binding: 'theme',
          options: ['dark', 'light'], hint: '深色 / 浅色主题' },
        { name: 'uiFontFamily', label: '界面字体', type: 'text', binding: 'uiFontFamily', group: '界面字体',
          placeholder: 'Cascadia Code, Consolas, monospace' },
        { name: 'uiFontBold', label: '粗体', type: 'checkbox', binding: 'uiFontBold', group: '界面字体' },
        { name: 'uiFontItalic', label: '斜体', type: 'checkbox', binding: 'uiFontItalic', group: '界面字体' },
        { name: 'uiFontUnderline', label: '下划线', type: 'checkbox', binding: 'uiFontUnderline', group: '界面字体' },
      ],
    })
  },
}
