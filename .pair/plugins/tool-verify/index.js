// ═══════════════════════════════════════════════════════════════
// tool-verify — 知识库过期验证（memory_verify/project_info_verify）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool
// 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：memory_verify、project_info_verify
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "memory_verify",
    "description": "验证所有记忆条目中引用的文件和目录是否仍然存在。如果条目引用了已不存在的文件，可能是过时信息，建议更新或删除。返回验证报告，包含每个过期条目的问题描述。",
    "usageGuide": "验证所有记忆条目引用的文件和目录是否仍然存在。过时记忆会误导 agent，建议定期运行。比手动检查更高效（自动解析引用路径并检测有效性）。",
    "parameters": {
      "properties": {},
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "project_info_verify",
    "description": "验证所有知识库条目中引用的文件和目录是否仍然存在。如果条目引用了已不存在的文件/目录，可能是过时信息，建议更新或删除。返回验证报告，包含每个过期条目的问题描述。",
    "usageGuide": "验证知识库条目引用的文件和目录是否仍然存在。项目重构后文件移动可能导致旧引用失效，运行此工具可发现并清理过时条目。",
    "parameters": {
      "properties": {},
      "type": "object"
    },
    "readOnly": true
  }
];

return {
  name: 'tool-verify',
  purpose: '知识库过期验证（memory_verify/project_info_verify）（自动生成，迁移自内置 Go 工具组）',
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
