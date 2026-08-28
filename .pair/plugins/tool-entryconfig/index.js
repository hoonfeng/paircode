// ═══════════════════════════════════════════════════════════════
// tool-entryconfig — 入口与配置定位（find_entry_points/find_config_files）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：find_entry_points、find_config_files
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "find_entry_points",
    "description": "列出项目中所有检测到的入口文件（main.go、index.ts、app.py 等）。通过扫描项目目录树匹配常见入口文件名实现，支持 Go / TypeScript / JavaScript / Python / Rust / C/C++ / Java 等语言。可选参数：path 限定子目录，maxResults 限制返回数量。",
    "parameters": {
      "properties": {
        "maxResults": {
          "description": "可选：最大返回结果数（默认 50，最大 200）",
          "type": "string"
        },
        "path": {
          "description": "可选：限定扫描的子目录路径，留空则扫描整个项目",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "find_config_files",
    "description": "列出项目中所有检测到的配置文件（go.mod、package.json、tsconfig.json、.gitignore、Makefile、Dockerfile 等）。通过扫描项目目录树匹配常见配置文件名和扩展名实现。可选参数：path 限定子目录，maxResults 限制返回数量。",
    "parameters": {
      "properties": {
        "maxResults": {
          "description": "可选：最大返回结果数（默认 50，最大 200）",
          "type": "string"
        },
        "path": {
          "description": "可选：限定扫描的子目录路径，留空则扫描整个项目",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  }
];

return {
  name: 'tool-entryconfig',
  purpose: '入口与配置定位（find_entry_points/find_config_files）（自动生成，迁移自内置 Go 工具组）',
  apply(ctx) {
    for (const t of tools) {
      ctx.tools.register({
        name: t.name,
        description: t.description,
        usageGuide: t.usageGuide,
        category: t.category,
        readOnly: t.readOnly,
        requiresApproval: t.requiresApproval,
        systemTool: t.systemTool,
        parameters: t.parameters,
        execute: (args) => ctx.hostTool.exec(t.name, args || {}),
      })
    }
  },
}
