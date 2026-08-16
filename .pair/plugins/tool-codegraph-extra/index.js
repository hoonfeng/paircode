// ═══════════════════════════════════════════════════════════════
// tool-codegraph-extra — 图谱扩展（codegraph_find_by_signature/explore）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool
// 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：codegraph_find_entry_points、codegraph_find_hot_paths、codegraph_find_by_imports、codegraph_get_detailed_symbol、codegraph_find_dead_imports、codegraph_search_by_error、codegraph_index_markdown、codegraph_search_docs、codegraph_verify_design、codegraph_pr_context、codegraph_find_by_signature、codegraph_semantic_search、codegraph_explore
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "codegraph_find_entry_points",
    "description": "发现应用程序入口点和执行起点。",
    "usageGuide": "发现应用程序入口点（main 函数、HTTP handler、CLI 命令）。新项目先调此工具了解从哪启动。",
    "parameters": {
      "properties": {
        "entryType": {
          "description": "可选：main/http_handler/cli_command/all，默认 all",
          "type": "string"
        },
        "limit": {
          "description": "可选：最大返回数（默认 50）",
          "type": "integer"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_find_hot_paths",
    "description": "查找最常被调用的函数。",
    "usageGuide": "查找最常被调用的函数（按调用者数量排序）。了解核心热点代码，优化优先考虑高频路径。",
    "parameters": {
      "properties": {
        "limit": {
          "description": "可选：最大返回数（默认 20）",
          "type": "integer"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_find_by_imports",
    "description": "查找所有导入指定模块的文件。",
    "usageGuide": "查找所有导入指定模块的文件。想了解某包被哪些文件引用时用。比 search_content 搜索 import 语句更精确（基于解析的 import 关系）。",
    "parameters": {
      "properties": {
        "limit": {
          "description": "可选：最大返回数（默认 50）",
          "type": "integer"
        },
        "matchMode": {
          "description": "可选：exact/prefix/contains/fuzzy，默认 contains",
          "type": "string"
        },
        "moduleName": {
          "description": "模块/包名",
          "type": "string"
        }
      },
      "required": [
        "moduleName"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_get_detailed_symbol",
    "description": "获取符号详细上下文（源码+调用者+被调用者）。",
    "usageGuide": "获取某符号的完整上下文：源码+调用者+被调用者。比分别调 codegraph_callers/callees 更省 token（一站式）。",
    "parameters": {
      "properties": {
        "includeSource": {
          "description": "可选：包含源码（默认 true）",
          "type": "boolean"
        },
        "query": {
          "description": "符号名",
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
    "name": "codegraph_find_dead_imports",
    "description": "查找已导入但从未使用的模块。",
    "usageGuide": "查找已导入但从未使用的模块。改完代码后运行可发现残留 import。比 goimports 更灵活（指定文件或全量扫描）。",
    "parameters": {
      "properties": {
        "file": {
          "description": "可选：指定文件",
          "type": "string"
        },
        "limit": {
          "description": "可选：最大返回数（默认 50）",
          "type": "integer"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_search_by_error",
    "description": "查找抛出或处理错误的函数。",
    "usageGuide": "查找抛出或处理特定错误的函数。mode=throws 找谁抛了错误，catches 找谁处理了。错误分析定位根因时用。",
    "parameters": {
      "properties": {
        "errorType": {
          "description": "可选：错误类型过滤",
          "type": "string"
        },
        "limit": {
          "description": "可选：最大返回数（默认 50）",
          "type": "integer"
        },
        "mode": {
          "description": "可选：throws/catches/any，默认 any",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_index_markdown",
    "description": "索引 Markdown 文档，按标题分段。有 ONNX 则计算嵌入向量。",
    "usageGuide": "索引 Markdown 文档到知识库。之后可用 codegraph_search_docs 语义搜索。新加文档后需重新索引才能搜到。",
    "parameters": {
      "properties": {
        "path": {
          "description": "可选：文件路径；省略则扫描全部 .md 文件",
          "type": "string"
        }
      },
      "type": "object"
    }
  },
  {
    "name": "codegraph_search_docs",
    "description": "搜索已索引文档。优先向量语义搜索，回退关键词。",
    "usageGuide": "搜索已索引的 Markdown 文档。有 ONNX 时做语义搜索（理解意图），否则关键词回退。比全文搜索更智能。",
    "parameters": {
      "properties": {
        "limit": {
          "description": "可选：最大返回数（默认 5）",
          "type": "integer"
        },
        "query": {
          "description": "搜索关键词",
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
    "name": "codegraph_verify_design",
    "description": "检查设计文档中的代码引用是否存在。",
    "usageGuide": "检查设计文档中的代码引用是否仍然有效。重构后运行可发现过期的文档引用。",
    "parameters": {
      "properties": {
        "docFile": {
          "description": "设计文档路径",
          "type": "string"
        }
      },
      "required": [
        "docFile"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_pr_context",
    "description": "分析分支变更影响范围。",
    "usageGuide": "分析当前分支与 baseBranch 的变更影响范围。提交 PR 前运行可了解变更波及哪些文件/函数。",
    "parameters": {
      "properties": {
        "baseBranch": {
          "description": "可选：基准分支（默认 main）",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_find_by_signature",
    "description": "按结构特征（参数数、返回类型、名称模式）查找函数。",
    "usageGuide": "按结构特征查找函数：参数个数/返回类型/名称模式。想找「接收 string 返回 error」的函数时用。比 search_content 更原子化（基于签名匹配）。",
    "parameters": {
      "properties": {
        "limit": {
          "description": "可选：最大返回数（默认 50）",
          "type": "integer"
        },
        "maxParams": {
          "description": "可选：最多参数个数",
          "type": "integer"
        },
        "minParams": {
          "description": "可选：最少参数个数",
          "type": "integer"
        },
        "namePattern": {
          "description": "可选：函数名通配模式，如 'get*'、'*Handler'",
          "type": "string"
        },
        "paramCount": {
          "description": "可选：精确参数个数",
          "type": "integer"
        },
        "returnType": {
          "description": "可选：返回类型，如 'error'、'string'",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_semantic_search",
    "description": "基于语义理解搜索代码（需 ONNX 嵌入模型）。支持自然语言查询，如「读取文件的函数」「处理错误的逻辑」。结果按语义相似度排序，比关键词搜索更准确。",
    "usageGuide": "基于语义理解搜索代码（需 ONNX 嵌入模型）。支持自然语言查询如「读取配置文件」「处理 HTTP 请求」。比关键词搜索更智能。",
    "parameters": {
      "properties": {
        "limit": {
          "description": "可选：最大返回数（默认 10）",
          "type": "integer"
        },
        "query": {
          "description": "自然语言查询，如「读取配置文件」「处理 HTTP 请求」",
          "type": "string"
        },
        "reindex": {
          "description": "可选：强制重新索引代码实体（默认 false）",
          "type": "boolean"
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
    "name": "codegraph_explore",
    "description": "一站式代码理解工具。用自然语言或符号名探索代码，返回相关源码和位置。分析代码的首选工具。",
    "usageGuide": "一站式代码理解工具。用自然语言或符号名探索代码，返回相关源码和位置。新接触项目时用此工具了解代码比逐个 read_file 更高效。",
    "parameters": {
      "properties": {
        "maxFiles": {
          "description": "可选：最大返回文件数（默认 8）",
          "type": "integer"
        },
        "query": {
          "description": "自然语言问题或符号名",
          "type": "string"
        }
      },
      "required": [
        "query"
      ],
      "type": "object"
    },
    "readOnly": true
  }
];

return {
  name: 'tool-codegraph-extra',
  purpose: '图谱扩展（codegraph_find_by_signature/explore）（自动生成，迁移自内置 Go 工具组）',
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
