// ═══════════════════════════════════════════════════════════════
// tool-snapshot — 会话快照（restore_snapshot/list_snapshots）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：restore_snapshot、list_snapshots
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "restore_snapshot",
    "description": "从快照恢复指定文件。快照在 edit_file/multi_edit/write_file 修改前自动创建。默认恢复到最旧快照（原始文件）。可用 list_snapshots 查看快照列表。指定 index 参数恢复特定版本（0=最旧原始文件，-1=最新，1~N=第 N 份从最旧算）。",
    "parameters": {
      "properties": {
        "index": {
          "description": "可选快照索引：0=最旧(原始/默认)，-1=最新，1~N=第 N 份",
          "type": "string"
        },
        "path": {
          "description": "要恢复的文件路径（工作区相对路径，如 \"cmd/main.go\"）",
          "type": "string"
        }
      },
      "required": [
        "path"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "list_snapshots",
    "description": "列出指定文件的所有可用快照（按时间倒序，带索引号）。用 restore_snapshot 的 index 参数可恢复指定版本。",
    "parameters": {
      "properties": {
        "path": {
          "description": "文件路径（工作区相对路径）",
          "type": "string"
        }
      },
      "required": [
        "path"
      ],
      "type": "object"
    },
    "readOnly": true
  }
];

return {
  name: 'tool-snapshot',
  purpose: '会话快照（restore_snapshot/list_snapshots）（自动生成，迁移自内置 Go 工具组）',
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
