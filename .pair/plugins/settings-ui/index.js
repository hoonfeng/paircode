// ═══════════════════════════════════════════════════════════════
// settings-ui — 配置 UI 插件（2026-08-19）
//
// ★ 配置本身无内置：所有配置项（组件类型/值/名称/分组）由本插件经
//   ctx.registerSettings 注册，前端设置面板纯 schema 驱动渲染。
//   · binding 字段 → AppSettings 顶层（settings.json，宿主运行时读取）
//   · 非 binding 字段 → pluginSettings[key]（插件私有命名空间）
//
// 组件类型规范：text|password|number|checkbox|select|textarea|slider|
//   color|tags(逗号分隔数组)|roles(键值对文本区)
// ═══════════════════════════════════════════════════════════════
return {
  name: 'settings-ui',
  purpose: '配置 UI：注册全部配置 schema（组件类型/值/名称/分组），设置面板纯 schema 驱动',
  inject: ['app'],
  apply(ctx) {
    // ── AI：服务商与模型 ──
    ctx.registerSettings({
      key: 'ai',
      title: 'AI',
      fields: [
        { name: 'provider', label: '服务商', type: 'select', binding: 'provider',
          options: ['deepseek', 'openai', 'ollama', 'anthropic', 'azure', 'custom'],
          hint: '模型服务商（决定默认 Base URL 与模型列表）' },
        { name: 'baseURL', label: 'Base URL', type: 'text', binding: 'baseURL',
          placeholder: 'https://api.deepseek.com/v1', hint: 'API 端点（custom 服务商必填）' },
        { name: 'apiKey', label: 'API Key', type: 'password', binding: 'apiKey',
          placeholder: 'sk-…', hint: '服务商密钥，仅本地保存' },
        { name: 'executeModel', label: '执行模型', type: 'text', binding: 'executeModel',
          placeholder: 'deepseek-v4-flash', hint: '执行 Agent 使用的模型' },
        { name: 'planModel', label: '规划模型', type: 'text', binding: 'planModel',
          placeholder: 'deepseek-v4-pro', hint: '规划 Agent 使用的模型（更强推理）' },
        { name: 'reviewModel', label: '审核模型', type: 'text', binding: 'reviewModel',
          placeholder: 'deepseek-v4-pro', hint: '审核 Agent 使用的模型' },
        { name: 'temperature', label: '温度', type: 'select', binding: 'temperature',
          options: ['0', '0.1', '0.3', '0.5', '0.7', '1.0'],
          hint: '随机性：越低越确定，越高越发散' },
        { name: 'thinkingMode', label: '思考模式', type: 'select', binding: 'thinkingMode',
          options: ['thinking', 'non-thinking'], hint: 'thinking=深度思考（更慢更准）' },
        { name: 'maxTokens', label: '最大输出 Token', type: 'number', binding: 'maxTokens', min: 1024, max: 131072, step: 1024 },
        { name: 'contextMaxTokens', label: '上下文窗口', type: 'number', binding: 'contextMaxTokens', min: 4096, max: 200000, step: 4096,
          hint: '历史注入的上下文上限' },
      ],
    })

    // ── Agent：行为 ──
    ctx.registerSettings({
      key: 'agent',
      title: 'Agent',
      fields: [
        { name: 'reviewMode', label: '审核模式', type: 'select', binding: 'reviewMode',
          options: ['auto', 'manual', 'off'],
          hint: 'auto=AI 审核、manual=手动审批、off=全部放行' },
        { name: 'autonomous', label: '自主执行', type: 'checkbox', binding: 'autonomous',
          hint: '开启后 agent 连续自主执行到完成' },
        { name: 'autoCollapse', label: '自动折叠', type: 'checkbox', binding: 'autoCollapse' },
        { name: 'maxIterations', label: '最大迭代', type: 'number', binding: 'maxIterations', min: 5, max: 500, step: 5 },
        { name: 'autoIterateOnRejection', label: '拒绝后自动迭代', type: 'checkbox', binding: 'autoIterateOnRejection' },
        { name: 'searxngUrl', label: 'SearXNG URL', type: 'text', binding: 'searxngUrl',
          placeholder: 'http://localhost:8080', hint: '可选：自建搜索实例' },
        { name: 'ignoreDirs', label: '忽略目录', type: 'tags', binding: 'ignoreDirs',
          hint: '逗号分隔（node_modules, dist, .git…）' },
        { name: 'autoConnectMCP', label: '自动连接 MCP', type: 'checkbox', binding: 'autoConnectMCP',
          hint: '启动时自动连接已配置的 MCP 服务器' },
      ],
    })

    // ── 编辑器 ──
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

    // ── 终端 ──
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

    // ── 外观 ──
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

    // ── 指令 ──
    ctx.registerSettings({
      key: 'instructions',
      title: '指令',
      fields: [
        { name: 'systemInstructions', label: '系统级指令（所有工作区共享）', type: 'textarea', binding: 'systemInstructions',
          placeholder: '输入全局系统指令…' },
      ],
    })

    // client 半可调：取当前全部配置值（顶层 + 插件命名空间）
    ctx.registerClientMethod('getValues', () => {
      return { top: ctx.app.settingsTop || {}, plugin: {} }
    })
  },
}
