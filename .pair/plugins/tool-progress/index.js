// ═══════════════════════════════════════════════════════════════
// tool-progress — 进度检查（progress_checker）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：progress_checker
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "progress_checker",
    "description": "检查当前任务完成进度，输出结构化进度报告，识别未完成的任务并给出执行建议。使用场景：任务列表较长时、Agent 不确定下一步做什么时、或用户要求查看进度时。",
    "parameters": {
      "properties": {
        "detail": {
          "description": "可选：详细模式，设为 \"full\" 显示每个任务的详细信息（含描述）",
          "enum": [
            "summary",
            "full"
          ],
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  }
];

return {
  name: 'tool-progress',
  purpose: '进度检查（progress_checker）（自动生成，迁移自内置 Go 工具组）',
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
