// ═══════════════════════════════════════════════════════════════
// tool-binary — 二进制读写（inspect_binary/write_binary）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.binary 复用统一宿主二进制（.pair/plugins/tool-binary/bin/，源码 cmd/plugins/tool-binary/，承载全部内置工具组实现）。
// 工具清单：inspect_binary、write_binary
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "inspect_binary",
    "description": "分析二进制文件而不撑爆上下文：返回大小 + 嗅探类型（按 magic bytes）+ 指定区段的十六进制/ASCII 预览（hexdump 风格）。读图片/可执行/压缩包/字体等二进制用它，别用 read_file。",
    "usageGuide": "分析二进制文件：大小 + 类型嗅探（magic bytes）+ hexdump 预览。二进制文件（图片/可执行/压缩包/字体等）只能用此工具，不可用 read_file（read_file 会拒绝含 NULL 字节的文件）。比直接读原始字节安全（预览有界不撑爆上下文）。",
    "parameters": {
      "properties": {
        "length": {
          "description": "可选：预览字节数（默认 256，上限 4096）",
          "type": "integer"
        },
        "offset": {
          "description": "可选：起始字节偏移（默认 0）",
          "type": "integer"
        },
        "path": {
          "description": "文件路径（工作区内）",
          "type": "string"
        }
      },
      "required": [
        "path"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "write_binary",
    "description": "把 base64 编码的字节写入文件（path；覆盖；父目录自动创建）。用于写二进制内容。",
    "usageGuide": "把 base64 编码的字节写入文件。用于写二进制内容（图片/字体/编译产物等）。需审核批准。比 write_file 更省 token（base64 比文本转义更紧凑）。",
    "parameters": {
      "properties": {
        "base64": {
          "description": "base64 编码的字节",
          "type": "string"
        },
        "path": {
          "description": "文件路径",
          "type": "string"
        }
      },
      "required": [
        "path",
        "base64"
      ],
      "type": "object"
    },
    "requiresApproval": true
  }
];

return {
  name: 'tool-binary',
  purpose: '二进制读写（inspect_binary/write_binary）（自动生成，迁移自内置 Go 工具组）',
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
        execute: (args) => ctx.binary.exec(t.name, args || {}, {bin: 'tool-binary'}),
      })
    }
  },
}
