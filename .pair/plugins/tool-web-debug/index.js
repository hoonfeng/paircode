// ═══════════════════════════════════════════════════════════════
// tool-web-debug — 网页验证（web_debug）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool
// 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：web_debug
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "web_debug",
    "description": "一站式网页验证工具：在无头浏览器中打开 URL，捕获控制台错误/警告、网络请求失败（404/500/CORS）、DOM 结构概览、元素查询（标签/样式/尺寸/可见性/属性）、可选输入文字、点击元素、执行 JS、提取页面可见文字，最后截图保存。用于验证前端改动是否正常工作（白屏、JS 异常、接口报错、样式错乱等）。截图保存到 screenshots/ 目录，返回文件路径可用 image_analyze 进一步分析。注意：首次使用会自动下载 Chromium（约 150MB），后续复用缓存。",
    "usageGuide": "一站式网页验证工具：在无头浏览器中打开 URL，检查控制台错误+网络请求失败+截图。支持交互操作（click_selector/type_selector+type_text）、JS 求值（eval）、文字提取(text_extract)、元素查询(element_query)。前端改动验证首选工具，比手动打开浏览器检查更全自动化。",
    "parameters": {
      "properties": {
        "click_selector": {
          "description": "可选：页面加载后点击的 CSS 选择器（如 '#submit-btn'）",
          "type": "string"
        },
        "element_query": {
          "description": "可选：CSS 选择器，查询匹配元素的详细信息（标签/类/样式/尺寸/可见性/属性/文本）",
          "type": "string"
        },
        "eval": {
          "description": "可选：在页面上执行的 JavaScript 表达式（如 'document.title' 或 'JSON.stringify(window.appState)'）",
          "type": "string"
        },
        "screenshot": {
          "description": "可选：是否截图（默认 true）。截图保存到 screenshots/ 目录",
          "type": "boolean"
        },
        "text_extract": {
          "description": "可选：提取页面可见纯文本内容（默认 false，内容过多时自动截断）",
          "type": "boolean"
        },
        "type_selector": {
          "description": "可选：要输入文字的 input/textarea 的 CSS 选择器",
          "type": "string"
        },
        "type_text": {
          "description": "可选：要输入的文字内容（需配合 type_selector）",
          "type": "string"
        },
        "url": {
          "description": "要验证的网页 URL（如 http://localhost:9090）",
          "type": "string"
        },
        "viewport_height": {
          "description": "可选：视口高度（默认 900）",
          "type": "integer"
        },
        "viewport_width": {
          "description": "可选：视口宽度（默认 1280）",
          "type": "integer"
        },
        "wait": {
          "description": "可选：页面加载后等待毫秒数（默认 2000，给 JS 渲染和异步请求时间）",
          "type": "integer"
        }
      },
      "required": [
        "url"
      ],
      "type": "object"
    },
    "readOnly": true
  }
];

return {
  name: 'tool-web-debug',
  purpose: '网页验证（web_debug）（自动生成，迁移自内置 Go 工具组）',
  apply(ctx) {
    for (const t of tools) {
      ctx.tools.register({
        name: t.name,
        description: t.description,
        usageGuide: t.usageGuide,
        category: t.category,
        readOnly: t.readOnly,
        requiresApproval: t.requiresApproval,
        parameters: t.parameters,
        execute: (args) => ctx.hostTool.exec(t.name, args || {}),
      })
    }
  },
}
