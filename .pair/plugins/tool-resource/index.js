// ═══════════════════════════════════════════════════════════════
// tool-resource — 资源管理（resource_list/resource_search/resource_stats）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：resource_list、resource_search、resource_stats
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "resource_list",
    "description": "列出所有智能资源（经验胶囊、技能基因、记忆、项目知识库、技能、MCP服务器、Lua工具）。可按类型（type）和作用域（scope）过滤。不传 type 则列出全部类型。",
    "parameters": {
      "properties": {
        "detail": {
          "description": "可选：是否显示详细信息（含描述、标签等），默认 false",
          "type": "boolean"
        },
        "scope": {
          "description": "可选：作用域过滤，\"global\"|\"project\"，不传=全部",
          "type": "string"
        },
        "type": {
          "description": "可选：资源类型过滤，如 \"capsules\"|\"genes\"|\"memory\"|\"project-info\"|\"skills\"|\"mcp-servers\"|\"lua-tools\"，不传=全部",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "resource_search",
    "description": "跨类型搜索智能资源（经验胶囊、技能基因、记忆、知识库等）。按名称、描述、标签模糊匹配。可用 type 和 scope 缩小范围。",
    "parameters": {
      "properties": {
        "query": {
          "description": "搜索关键词（必填，匹配名称/描述/标签）",
          "type": "string"
        },
        "scope": {
          "description": "可选：作用域过滤，\"global\"|\"project\"，不传=全部",
          "type": "string"
        },
        "type": {
          "description": "可选：资源类型过滤，如 \"capsules\"|\"genes\"|\"memory\"，不传=全部",
          "type": "string"
        }
      },
      "required": [
        "query"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "resource_stats",
    "description": "查看各类智能资源的数量统计。直观展示经验胶囊、基因、记忆、知识库、技能、MCP服务器、Lua工具的数量分布。",
    "parameters": {
      "properties": {
        "scope": {
          "description": "可选：作用域过滤，\"global\"|\"project\"，不传=全部",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  }
];

return {
  name: 'tool-resource',
  purpose: '资源管理（resource_list/resource_search/resource_stats）（自动生成，迁移自内置 Go 工具组）',
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
