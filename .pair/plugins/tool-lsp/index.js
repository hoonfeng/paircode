// ═══════════════════════════════════════════════════════════════
// tool-lsp — LSP 代码导航（lsp_definition/references/hover/diagnostics）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool
// 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：lsp_definition、lsp_references、lsp_hover、lsp_diagnostics
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "lsp_definition",
    "description": "Jump to where a symbol is defined. Give the file path (relative to workspace root), the 1-based line number the symbol appears on, and the symbol text itself.",
    "parameters": {
      "properties": {
        "file": {
          "description": "Path to the source file, relative to the workspace root or absolute.",
          "type": "string"
        },
        "line": {
          "description": "1-based line number the symbol appears on.",
          "type": "integer"
        },
        "symbol": {
          "description": "The exact symbol text on that line, e.g. \"executeBatch\".",
          "type": "string"
        }
      },
      "required": [
        "file",
        "line",
        "symbol"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "lsp_references",
    "description": "List every reference to a symbol across the workspace. Give the file path (relative to workspace root), the 1-based line number, and the symbol text.",
    "parameters": {
      "properties": {
        "file": {
          "description": "Path to the source file, relative to the workspace root or absolute.",
          "type": "string"
        },
        "line": {
          "description": "1-based line number the symbol appears on.",
          "type": "integer"
        },
        "symbol": {
          "description": "The exact symbol text on that line, e.g. \"executeBatch\".",
          "type": "string"
        }
      },
      "required": [
        "file",
        "line",
        "symbol"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "lsp_hover",
    "description": "Show the type signature and documentation for a symbol. Give the file path (relative to workspace root), the 1-based line number, and the symbol text.",
    "parameters": {
      "properties": {
        "file": {
          "description": "Path to the source file, relative to the workspace root or absolute.",
          "type": "string"
        },
        "line": {
          "description": "1-based line number the symbol appears on.",
          "type": "integer"
        },
        "symbol": {
          "description": "The exact symbol text on that line, e.g. \"executeBatch\".",
          "type": "string"
        }
      },
      "required": [
        "file",
        "line",
        "symbol"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "lsp_diagnostics",
    "description": "Report compiler/linter diagnostics (errors, warnings) for a file from its language server. Use after editing to check the change compiles.",
    "parameters": {
      "properties": {
        "file": {
          "description": "Path to the source file, relative to the workspace root or absolute.",
          "type": "string"
        }
      },
      "required": [
        "file"
      ],
      "type": "object"
    },
    "readOnly": true
  }
];

return {
  name: 'tool-lsp',
  purpose: 'LSP 代码导航（lsp_definition/references/hover/diagnostics）（自动生成，迁移自内置 Go 工具组）',
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
