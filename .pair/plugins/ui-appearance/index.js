// ui-appearance — 外观设置插件（2026-08-19）
// ★ 配置分散化：主题/界面字体等外观配置由本插件注册（全局 UI 外观对应插件）
//   · theme → AppSettings.theme（前端实时应用）
//   · uiFont* → AppSettings 顶层（界面字体风格）
// ★ 2026-09（t1 报告 F1 缺口闭环）：选项与壳主题 CSS 对齐——web/index.html 内
//   4 套主题（.theme-dark/.theme-light/.theme-warm/.theme-night）硬编码在壳，
//   设置面板此前只提供 dark/light 两项（warm/night 无法选择）。补全 4 项；
//   「主题内容插件化」（CSS 变量随插件下发）列为后续演进（壳 CSS 迁移 ui 插件时
//   本选项表与壳 class 一一对应，保持同源）。
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
          options: ['dark', 'light', 'warm', 'night'],
          hint: '深色 / 浅色 / 暖色 / 夜间 主题（与壳 4 套主题 CSS 一一对应）' },
      ],
    })
  },
}
