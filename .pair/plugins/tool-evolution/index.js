// ═══════════════════════════════════════════════════════════════
// tool-evolution — 进化系统（evolution_save_capsule/evolution_search_capsules/evolution_save_gene/evolution_status）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：evolution_save_capsule、evolution_search_capsules、evolution_save_gene、evolution_status
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "evolution_save_capsule",
    "description": "保存经验胶囊：将修复经验持久化到记忆系统供未来复用。当 Agent 成功修复一个错误后调用此工具，将修复过程编码为经验胶囊。",
    "parameters": {
      "properties": {
        "contextTags": {
          "description": "上下文标签，如 [\"typescript\",\"electron\",\"react\"]",
          "items": {
            "type": "string"
          },
          "type": "array"
        },
        "errorPattern": {
          "description": "错误消息模式（用于模糊匹配），如 \"Cannot find module '...'\"",
          "type": "string"
        },
        "errorType": {
          "description": "错误类型",
          "enum": [
            "模块未找到",
            "类型错误",
            "运行时错误",
            "语法错误",
            "网络错误",
            "权限错误",
            "其他"
          ],
          "type": "string"
        },
        "keyChanges": {
          "description": "核心代码变更摘要",
          "type": "string"
        },
        "scope": {
          "description": "作用域：\"global\"（跨项目共享）或 \"project\"（仅当前项目），默认 \"global\"",
          "type": "string"
        },
        "steps": {
          "description": "修复步骤列表",
          "items": {
            "type": "string"
          },
          "type": "array"
        },
        "summary": {
          "description": "修复摘要，一句话描述修复了什么",
          "type": "string"
        },
        "toolName": {
          "description": "失败的工具名称",
          "type": "string"
        }
      },
      "required": [
        "errorType",
        "errorPattern",
        "toolName",
        "summary",
        "steps",
        "keyChanges"
      ],
      "type": "object"
    }
  },
  {
    "name": "evolution_search_capsules",
    "description": "搜索匹配的经验胶囊：根据错误信息检索历史修复方案。Agent 遇到错误时调用此工具查找已知的修复方案。",
    "parameters": {
      "properties": {
        "contextTags": {
          "description": "上下文标签，如 [\"typescript\",\"electron\"]",
          "items": {
            "type": "string"
          },
          "type": "array"
        },
        "errorMessage": {
          "description": "错误消息全文，用于模糊匹配历史修复方案",
          "type": "string"
        },
        "toolName": {
          "description": "当前工具名称",
          "type": "string"
        }
      },
      "required": [
        "errorMessage",
        "toolName"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "evolution_save_gene",
    "description": "保存技能基因：记录可跨项目复用的编程最佳实践。当发现通用的编程模式或最佳实践时调用此工具。",
    "parameters": {
      "properties": {
        "body": {
          "description": "技能详细描述（Markdown）",
          "type": "string"
        },
        "category": {
          "description": "技能分类",
          "enum": [
            "架构设计",
            "设计模式",
            "配置管理",
            "调试技巧",
            "工作流程"
          ],
          "type": "string"
        },
        "description": {
          "description": "技能描述",
          "type": "string"
        },
        "examples": {
          "description": "示例代码片段",
          "items": {
            "type": "string"
          },
          "type": "array"
        },
        "frameworks": {
          "description": "适用框架，如 [\"electron\", \"react\"]",
          "items": {
            "type": "string"
          },
          "type": "array"
        },
        "languages": {
          "description": "适用语言，如 [\"typescript\"]",
          "items": {
            "type": "string"
          },
          "type": "array"
        },
        "name": {
          "description": "技能中文名称，如「Electron IPC 安全调用模式」",
          "type": "string"
        },
        "scope": {
          "description": "作用域：\"global\"（跨项目共享）或 \"project\"（仅当前项目），默认 \"global\"",
          "type": "string"
        },
        "tags": {
          "description": "自定义标签",
          "items": {
            "type": "string"
          },
          "type": "array"
        }
      },
      "required": [
        "name",
        "description",
        "category",
        "body"
      ],
      "type": "object"
    }
  },
  {
    "name": "evolution_status",
    "description": "查看 BES（Bugee Evolution System）状态。返回当前进化引擎的运行状态、项目指纹和已积累的进化资产数量。",
    "parameters": {
      "properties": {},
      "type": "object"
    },
    "readOnly": true
  }
];

return {
  name: 'tool-evolution',
  purpose: '进化系统（evolution_save_capsule/evolution_search_capsules/evolution_save_gene/evolution_status）（自动生成，迁移自内置 Go 工具组）',
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
