// ui-appearance — 外观设置插件
// ★ 配置分散化：主题/界面字体等外观配置由本插件注册（全局 UI 外观对应插件）
//   · theme → AppSettings.theme（前端实时应用；binding 直连，零框架改动）
//   · uiFont* → AppSettings 顶层（界面字体风格）
//
// ★ P2「主题 CSS 插件化」（2026-09-25）：
//   8 套主题的完整 CSS 已从壳 index.html 迁入本插件包（theme-<id>.css），由 client.js
//   按当前主题注入 <link href="/plugins-assets/ui-appearance/theme-<id>.css">；
//   壳仅保留一份 :root 基准令牌兜底。故此处选项表与 theme-<id>.css 一一对应。
//   ★ 旧 4 主题 id（dark/light/warm/night）作为别名保留在各主题 CSS 的选择器里，
//     行为与迁移前一致（设置面板不再提供，但既有配置值与持久化数据仍可用）。
return {
  name: 'ui-appearance',
  purpose: '外观设置：主题与界面字体（8 套主题画廊；主题 CSS 随插件下发）',
  apply(ctx) {
    ctx.registerSettings({
      key: 'appearance',
      title: '外观',
      fields: [
        { name: 'theme', label: '主题', type: 'theme-gallery', binding: 'theme',
          options: ['midnight', 'graphite', 'obsidian', 'aurora', 'daylight', 'paper', 'sand', 'slate'],
          hint: '8 套主题（4 暗 / 4 亮）。完整 CSS 随本插件下发，切换即时生效。',
          swatches: [
            {
              "value": "midnight",
              "label": "Midnight 午夜蓝（暗）",
              "desc": "默认暗色，久看不累",
              "scheme": "dark",
              "colors": [
                "#0F1420",
                "#151B29",
                "#6FA8FF",
                "#6FA8FF",
                "#6BD98A"
              ]
            },
            {
              "value": "graphite",
              "label": "Graphite 石墨（暗）",
              "desc": "中性无彩，专注编码",
              "scheme": "dark",
              "colors": [
                "#17191D",
                "#1D2025",
                "#78C2D6",
                "#6FA8FF",
                "#6BD98A"
              ]
            },
            {
              "value": "obsidian",
              "label": "Obsidian 黑曜石（暗）",
              "desc": "近黑高对比，OLED 友好",
              "scheme": "dark",
              "colors": [
                "#07080A",
                "#0F1115",
                "#B49BFF",
                "#6FA8FF",
                "#6BD98A"
              ]
            },
            {
              "value": "aurora",
              "label": "Aurora 极光（暗）",
              "desc": "紫青夜色，灵感模式",
              "scheme": "dark",
              "colors": [
                "#12101C",
                "#191625",
                "#5FE3C4",
                "#6FA8FF",
                "#6BD98A"
              ]
            },
            {
              "value": "daylight",
              "label": "Daylight 日光（亮）",
              "desc": "默认亮色，清晰通透",
              "scheme": "light",
              "colors": [
                "#F5F7FB",
                "#FFFFFF",
                "#1B5FD9",
                "#1B5FD9",
                "#186418"
              ]
            },
            {
              "value": "paper",
              "label": "Paper 纸白（亮）",
              "desc": "中性纸感，低蓝光",
              "scheme": "light",
              "colors": [
                "#F8F6F2",
                "#FFFDFA",
                "#8A4B12",
                "#1B5FD9",
                "#186418"
              ]
            },
            {
              "value": "sand",
              "label": "Sand 沙丘（亮）",
              "desc": "暖调米色，阅读舒适",
              "scheme": "light",
              "colors": [
                "#F6F0E7",
                "#FFFBF5",
                "#9A4F16",
                "#1B5FD9",
                "#186418"
              ]
            },
            {
              "value": "slate",
              "label": "Slate 石板（亮）",
              "desc": "冷灰建筑感，克制",
              "scheme": "light",
              "colors": [
                "#EEF1F6",
                "#FFFFFF",
                "#3B45C9",
                "#1B5FD9",
                "#186418"
              ]
            }
          ] },

        // ── 跟随系统（设计稿屏 4）──
        // ★ 作「覆盖开关」而非替换主题字段：开启后 client 半按 prefers-color-scheme
        //   自动在亮/暗主题间切换；关闭则回到用户在画廊里选的那一套。
        { name: 'themeFollowSystem', label: '跟随系统', type: 'checkbox',
          hint: '随系统深色 / 浅色自动切换（浅色 → Daylight / 深色 → Midnight）。开启后以系统偏好为准。' },

        // ── 强调色覆盖（设计稿屏 4）──
        // ★ 只覆盖 accent 系列槽位，不动整套色板；「默认」= 完全跟随主题自带强调色。
        //   值走 color-dots 类型（圆点 + 选中描边），与 select / theme-gallery 同数据模型。
        { name: 'accentOverride', label: '强调色', type: 'color-dots',
          default: 'default',
          hint: '可选覆盖主题强调色（按钮 / 选中 / 焦点）。「默认」跟随主题自带强调色。',
          swatches: [
            { value: 'default', label: '默认（跟随主题）', colors: ['#6FA8FF'] },
            { value: 'teal', label: '青绿', colors: ['#22B8A6'] },
            { value: 'amber', label: '琥珀', colors: ['#D8A03C'] },
            { value: 'crimson', label: '朱红', colors: ['#D9534F'] },
            { value: 'violet', label: '紫罗兰', colors: ['#8B6CF0'] },
          ] },

        // ── 应用背景（P3）──
        // ★ 架构：壳只提供 #app-bg 渲染层与 --app-bg-* / --app-surface-alpha 语义槽
        //   （见 index.html），本插件负责把设置写进这些槽位 —— 背景能力可独立演进，壳无需变更。
        // ★ 对比度守护：client 半按「最坏情况背景」反推所需遮罩强度，
        //   保证任意不透明度下正文对比度仍 ≥4.5:1（大字 ≥3:1），见 client.js computeScrim。
        // ★ 2026-09-25 参数面对齐设计稿屏 5：来源 4 种 / 填充方式 / 模糊 0–40 /
        //   暗化提亮 / 背景可见度 / 面板不透明度 / 每主题独立背景。
        { name: 'appBgSource', label: '背景来源', type: 'select',
          options: ['none', 'gradient', 'color', 'image'],
          default: 'none',
          hint: '无 / 主题渐变（跟随主题双色氛围）/ 纯色 / 自定义图片。默认无（与旧版行为一致）。' },
        { name: 'appBgColor', label: '背景纯色', type: 'color', default: '#0F1420',
          hint: '「纯色」来源下的底色。' },
        { name: 'appBgImage', label: '背景图 URL', type: 'text',
          placeholder: 'https://… 或 /assets/xxx.jpg（仅「自定义图片」来源生效）',
          hint: '支持远程 URL 与本机静态路径。跨域图片仅取平均亮度受限时会走更保守的守护强度。' },
        { name: 'appBgFit', label: '填充方式', type: 'select',
          options: ['cover', 'contain', 'repeat', 'center'],
          default: 'cover',
          hint: '覆盖（铺满，可能裁切）/ 适应（完整显示，可能留边）/ 平铺 / 居中（原始尺寸）。' },
        { name: 'appBgBlur', label: '模糊强度 (px)', type: 'slider',
          min: 0, max: 40, step: 2,
          default: 0,
          hint: '毛玻璃效果；虚化可降低背景细节对文字的干扰。' },
        { name: 'appBgDim', label: '暗化 / 提亮 (%)', type: 'slider',
          min: -50, max: 60, step: 5,
          default: 0,
          hint: '负值压暗、正值提亮。暗色主题建议 30~60 压暗，亮色主题可用负值提亮。' },
        { name: 'appBgOpacity', label: '背景可见度 (%)', type: 'slider',
          min: 0, max: 100, step: 5,
          default: 35,
          hint: '0=完全不显示背景，100=完全显示。数值越大背景越明显，文字可读性由对比度守护自动兜底。' },
        { name: 'appSurfaceAlpha', label: '面板不透明度 (%)', type: 'slider',
          min: 40, max: 100, step: 4,
          default: 100,
          hint: '面板底色透明度：越低越能透出背景（玻璃化）。低于 60% 时自动补描边保证边界可读。' },
        { name: 'appBgPerTheme', label: '每主题独立背景', type: 'checkbox',
          hint: '开启后亮色主题可用另一张背景图（暗色主题用上面的 URL）。' },
        { name: 'appBgImageLight', label: '亮色主题背景图', type: 'text',
          placeholder: 'https://…（仅「每主题独立背景」开启时生效）',
          hint: '仅在开启「每主题独立背景」时生效；亮色主题用此图，暗色主题用上面的 URL。' },
      ],
    })
  },
}
