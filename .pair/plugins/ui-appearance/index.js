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
      ],
    })
  },
}
