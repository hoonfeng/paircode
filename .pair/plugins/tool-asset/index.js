// ═══════════════════════════════════════════════════════════════
// tool-asset — 智能资产管理（asset_list/asset_search/asset_delete：经验胶囊 + 技能基因）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：asset_list、asset_search、asset_delete
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "asset_list",
    "description": "列出所有智能资产（经验胶囊 + 技能基因）。可按作用域（global/project）和资产类型（capsules/genes）过滤。",
    "parameters": {
      "properties": {
        "detail": {
          "description": "可选：是否显示详细信息（包括描述、标签等），默认 false 只显示概览",
          "type": "boolean"
        },
        "scope": {
          "description": "可选：作用域过滤，\"global\"（全局）或 \"project\"（项目级），不传则列出全部",
          "type": "string"
        },
        "type": {
          "description": "可选：资产类型过滤，\"capsules\"（胶囊）| \"genes\"（基因），不传则列出全部",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "asset_search",
    "description": "搜索智能资产（经验胶囊 + 技能基因）。按名称、描述、标签进行模糊匹配。",
    "parameters": {
      "properties": {
        "query": {
          "description": "搜索关键词（匹配名称、描述、标签）",
          "type": "string"
        },
        "scope": {
          "description": "可选：作用域过滤，\"global\" 或 \"project\"，不传则全部",
          "type": "string"
        },
        "type": {
          "description": "可选：资产类型过滤，\"capsules\" | \"genes\"，不传则全部",
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
    "name": "asset_delete",
    "description": "删除指定智能资产（经验胶囊 / 技能基因）。此操作不可逆。使用 asset_list 查看资产列表获取 ID。",
    "parameters": {
      "properties": {
        "id": {
          "description": "资产 ID（胶囊或基因的 ID 字段）",
          "type": "string"
        },
        "scope": {
          "description": "作用域：\"global\" 或 \"project\"，默认 \"global\"",
          "type": "string"
        },
        "type": {
          "description": "资产类型：\"capsules\" | \"genes\"",
          "type": "string"
        }
      },
      "required": [
        "id",
        "type"
      ],
      "type": "object"
    },
    "requiresApproval": true
  }
];

return {
  name: 'tool-asset',
  purpose: '智能资产管理（asset_list/asset_search/asset_delete：经验胶囊 + 技能基因）（自动生成，迁移自内置 Go 工具组）',
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
