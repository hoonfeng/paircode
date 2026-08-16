// ═══════════════════════════════════════════════════════════════
// tool-core — 核心文件工具（read_file/write_file/edit_file/
// multi_edit/run_command/move_file/delete_file）
//
// 迁移来源（2026-08-16）：内置 registerCoreTools（internal/agent/tools.go）
// → 磁盘外置插件。api 声明在插件，execute 调 ctx.hostTool 复用宿主 Go
// 执行器（快照/行号追踪/CRLF 保留/审批语义保持一致）。
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    name: 'read_file',
    description: '读取文件内容。path 为工作区内路径。可选 offset(起始行,1 基)+limit(行数)读片段；省略则读全文(超 2000 行只返回前 2000 行并提示用 offset/limit 翻页)。',
    usageGuide: '读取文件内容，限工作区内路径。大文件用 offset+limit 分页读取，避免撑爆上下文。二进制文件会自动拒绝读取，请改用 inspect_binary。',
    category: '文件',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '文件路径（工作区内）' },
        offset: { type: 'integer', description: '可选：起始行号(1 基)' },
        limit: { type: 'integer', description: '可选：读取行数' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['path'],
    },
  },
  {
    name: 'write_file',
    description: '把 content 完整写入 path（覆盖；父目录自动创建）。',
    usageGuide: '写入文件，父目录自动创建。需审核批准。比 os.WriteFile 更安全（自动快照+路径越界拦截+变更回调）。',
    category: '文件',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '文件路径' },
        content: { type: 'string', description: '完整文件内容' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['path', 'content'],
    },
  },
  {
    name: 'edit_file',
    description: '把文件中唯一一处 old_string 替换为 new_string。匹配策略（自动）：精确→CRLF归一化→空白折叠；全部失败时返回带行号上下文的诊断。替代方案：用 line_start/line_end 行号定位整段替换（最可靠）。保留文件原换行风格。',
    usageGuide: '把文件中唯一一处 old_string 替换为 new_string。内置智能匹配（CRLF 归一化+空白折叠）。匹配失败时优先用 line_start/line_end 行号定位（最可靠）。仅用于小改动（≤5 行），大改动请用 write_file 写整段。',
    category: '文件',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '文件路径' },
        old_string: { type: 'string', description: '待替换原文（须在文件中唯一；line_start>0 时可省略或作校验）' },
        new_string: { type: 'string', description: '替换后的新文' },
        line_start: { type: 'integer', description: '可选：1 基起始行号，>0 时启用行号定位模式（与 old_string 二选一或并用）' },
        line_end: { type: 'integer', description: '可选：1 基结束行号（含）；省略或 < line_start 时只替换 line_start 一行' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['path', 'new_string'],
    },
  },
  {
    name: 'multi_edit',
    description: '对一个文件按顺序应用多处替换（edits：每项 old_string→new_string 或 line_start/line_end 行号定位）。匹配策略同 edit_file。原子：任一步失败则全部不写。保留文件原换行风格。',
    usageGuide: '按顺序对一个文件应用多处替换。比多次 edit_file 更高效（原子提交：任一步失败全部回滚）。编辑项较多时用 multi_edit 替代多次 edit_file 调用。',
    category: '文件',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '文件路径' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
        edits: {
          type: 'array',
          description: '按顺序应用的替换列表',
          items: {
            type: 'object',
            properties: {
              old_string: { type: 'string', description: '待替换原文（须唯一；line_start>0 时可省略或作校验）' },
              new_string: { type: 'string', description: '替换后的新文' },
              line_start: { type: 'integer', description: '可选：1 基起始行号，>0 时启用行号定位模式' },
              line_end: { type: 'integer', description: '可选：1 基结束行号（含）；省略只替换 line_start 一行' },
            },
            required: ['new_string'],
          },
        },
      },
      required: ['path', 'edits'],
    },
  },
  {
    name: 'run_command',
    description: '同步执行一条 shell 命令并返回输出。每次调用独立 shell（无状态持久）。禁止长期进程（用 run_background）。',
    usageGuide: '同步执行一条 shell 命令并返回输出（构建/测试/查询等短命令）。120s 超时自动终止。长期进程用 run_background/read_output/kill_process。',
    category: '执行',
    parameters: {
      type: 'object',
      properties: {
        command: { type: 'string', description: '要执行的命令' },
        cwd: { type: 'string', description: '可选工作目录（工作区内）' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['command'],
    },
  },
  {
    name: 'move_file',
    description: '移动/重命名文件（from → to，工作区内）。',
    usageGuide: '移动/重命名文件。路径越界自动拦截+变更回调。',
    category: '文件',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        from: { type: 'string', description: '源路径' },
        to: { type: 'string', description: '目标路径' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['from', 'to'],
    },
  },
  {
    name: 'delete_file',
    description: '删除一个文件（工作区内，不可恢复，谨慎）。为安全不删目录。',
    usageGuide: '删除工作区内的文件（不可恢复，谨慎）。为安全不删目录（删除目录请用 run_command rmdir）。需审核批准。',
    category: '文件',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '要删除的文件路径' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['path'],
    },
  },
]

return {
  name: 'tool-core',
  purpose: '核心文件工具（read_file/write_file/edit_file/multi_edit/run_command/move_file/delete_file）——迁移自内置 registerCoreTools',
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
    // 日志已省略（logger 需 inject 声明）
  },
}
