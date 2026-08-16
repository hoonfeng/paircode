// ═══════════════════════════════════════════════════════════════
// tool-bug — BUG 检测与修复（bug_detect/bug_analyze/bug_fix）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.binary 复用统一宿主二进制（.pair/plugins/tool-binary/bin/，源码 cmd/plugins/tool-binary/，承载全部内置工具组实现）。
// 工具清单：bug_analyze、bug_detect、bug_fix
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "bug_analyze",
    "description": "分析构建/测试/运行输出，提取错误位置和代码上下文。接受 output（构建输出文本），output_type（build/test/run），返回解析后的错误列表（含文件路径、行号、消息和代码上下文）。由 Detector 在自动检测时调用，也可供 agent 手动分析构建日志。",
    "usageGuide": "分析构建/测试/运行输出，提取错误位置和代码上下文。配合 bug_detect 使用：先 bug_detect 全量检测，拿到输出后 bug_analyze 定位根因。比肉眼扫日志更快（结构化提取错误位置+行号）。",
    "parameters": {
      "properties": {
        "output": {
          "description": "构建/测试/运行的完整输出文本",
          "type": "string"
        },
        "output_type": {
          "description": "输出类型：build（编译输出）/ test（测试输出）/ run（运行时输出），默认 build",
          "type": "string"
        }
      },
      "required": [
        "output"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "bug_detect",
    "description": "全量检测项目中是否存在 BUG。自动运行 go vet → go build → go test，输出解析后的错误列表（含文件路径、行号、错误消息和代码上下文）。用于自动发现编译/测试/运行时的 BUG。集成在自主模式的编排循环中。",
    "usageGuide": "全量检测项目 BUG：自动运行 go vet → go build → go test，聚合所有错误。修改代码后验证无错误的推荐工具。比手动分别运行更高效（一站式检测）。",
    "parameters": {
      "properties": {},
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "bug_fix",
    "description": "自动检测项目 BUG（编译/测试/运行时错误），生成详细的修复任务文本。返回包含错误位置、代码上下文和修复指南的完整修复任务。可用于自主模式中在 loop 之间自动检测并修复项目问题。",
    "usageGuide": "自动检测并修复项目 BUG。运行编译/测试后定位错误，生成修复方案并 apply。max_attempts 控制最大尝试次数（默认 3）。注意：自动修复可能引入新问题，改完需验证。",
    "parameters": {
      "properties": {
        "max_attempts": {
          "description": "可选：最大修复尝试次数，默认 3",
          "type": "integer"
        }
      },
      "type": "object"
    }
  }
];

return {
  name: 'tool-bug',
  purpose: 'BUG 检测与修复（bug_detect/bug_analyze/bug_fix）（自动生成，迁移自内置 Go 工具组）',
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
        execute: (args) => ctx.binary.exec(t.name, args || {}),
      })
    }
  },
}
