// ═══════════════════════════════════════════════════════════════
// tool-harness — harness 核心协议工具（read/write/edit/glob/grep/
// bash/str_replace_editor/run_code）
//
// 迁移来源（2026-08-16）：内置 RegisterHarnessTools（internal/agent/
// harness_tools.go）→ 磁盘外置插件。api（name/description/parameters）
// 与本插件声明；execute 编排调 ctx.hostTool 复用宿主 Go 执行器
// （对齐 harness seam：工具编排在插件、底层能力在宿主；宿主执行器
// 由 PluginHost 在插件接管时自动存档）。
//
// 装配：.pair/plugins/ 启动扫描（LoadGlobalPlugins）→ define + load。
// 停用本插件（cordis_stop tool-harness）即回收全部 8 个工具。
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    name: 'read',
    description: '读取文件内容（对齐 deepseek-harness read）。path 为工作区内路径；可选 offset(起始行,1 基)+limit(行数)读片段；省略则读全文(超 2000 行只返回前 2000 行并提示翻页)。',
    usageGuide: 'harness 标准读工具：读取文件内容。路径越界自动拦截，二进制自动拒绝（改用 inspect_binary）。大文件用 offset+limit 分页。',
    category: '文件',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '文件路径（工作区内）' },
        offset: { type: 'integer', description: '可选：起始行号(1 基)' },
        limit: { type: 'integer', description: '可选：读取行数' },
        project: { type: 'string', description: '可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。' },
      },
      required: ['path'],
    },
  },
  {
    name: 'write',
    description: '把 content 完整写入 path（覆盖；父目录自动创建）。需审核批准。',
    usageGuide: 'harness 标准写工具：整文件写入（覆盖）。写类操作需人工确认。如需追加请先 read 再 write 覆盖。',
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
    name: 'edit',
    description: '把文件中唯一一处 old_string 替换为 new_string（对齐 deepseek-harness edit）。内置智能匹配（CRLF 归一化+空白折叠）；匹配失败优先用 line_start/line_end 行号定位。',
    usageGuide: 'harness 标准编辑工具：小改动（≤5 行）用精确替换；大改动请用 write 写整段。替换前会自动快照。',
    category: '文件',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '文件路径' },
        new_string: { type: 'string', description: '替换后的新文本' },
        old_string: { type: 'string', description: '待替换原文（须在文件中唯一；line_start>0 时可省略或作校验）' },
        line_start: { type: 'integer', description: '可选：1 基起始行号（含）；省略或 < line_start 时只替换 line_start 一行' },
        line_end: { type: 'integer', description: '可选：1 基结束行号（含）' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['path', 'new_string'],
    },
  },
  {
    name: 'glob',
    description: '按通配符递归查找文件，返回相对路径列表（对齐 deepseek-harness glob）。pattern 含 / 或 ** 时按路径模式（如 internal/**/*.go），否则匹配任意深度文件名（如 *.go）；path 限定子目录。',
    usageGuide: 'harness 标准 glob 工具：按路径模式发现文件。跳过 .git/node_modules 等目录。比 shell find 更精确（结构化、防撑爆）。',
    category: '代码搜索',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        pattern: { type: 'string', description: '文件名/路径通配符，如 *.go' },
        path: { type: 'string', description: '可选：限定子目录（省略=工作区根）' },
        language: { type: 'string', description: '可选：按语言过滤，如 "go"、"typescript"' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['pattern'],
    },
  },
  {
    name: 'grep',
    description: '在工作区内按正则搜索文件内容，返回「相对路径:行号: 行文本」（对齐 deepseek-harness grep）。pattern 为 RE2 正则；path 限定子目录；glob 按文件名过滤；case_insensitive 忽略大小写。',
    usageGuide: 'harness 标准 grep 工具：正则全文搜索。搜索函数/类型定义请优先用 codegraph_search（AST 级更精确）。',
    category: '代码搜索',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        pattern: { type: 'string', description: 'RE2 正则表达式' },
        path: { type: 'string', description: '可选：限定子目录（省略=工作区根）' },
        glob: { type: 'string', description: '可选：文件名通配过滤，如 *.go' },
        case_insensitive: { type: 'boolean', description: '可选：忽略大小写' },
        max_results: { type: 'integer', description: '可选：结果行数上限（默认 200）' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['pattern'],
    },
  },
  {
    name: 'bash',
    description: '同步执行一条 shell 命令并返回输出（对齐 deepseek-harness bash）。每次调用在独立 shell 中运行（无状态持久）。禁止用于长期进程（dev server/watch/tcp 监听）——请用 run_background。',
    usageGuide: 'harness 标准 bash 工具：执行命令（构建/测试/查询等短命令）。120s 超时自动终止。长期进程用 run_background/read_output/kill_process。',
    category: '执行',
    parameters: {
      type: 'object',
      properties: {
        command: { type: 'string', description: '要执行的命令' },
        cwd: { type: 'string', description: '可选工作目录（工作区内，省略=根）' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['command'],
    },
  },
  {
    name: 'str_replace_editor',
    description: 'Custom editing tool for viewing, creating and editing files（对齐 deepseek-harness str_replace_editor）\n' +
      '* `command` 必填：view / create / str_replace / insert\n' +
      '* `view` 显示文件内容（带行号）；path 为目录时列出非隐藏文件/目录最多 2 层\n' +
      '* `create` 创建新文件（path 已存在则报错）；内容在 `file_text`\n' +
      '* `str_replace` 把 `old_str` 替换为 `new_str`——old_str 必须精确匹配且唯一（含空白！不唯一则拒绝）\n' +
      '* `insert` 在 `insert_line` 之后插入 `new_str`\n' +
      '* `view` 支持 `view_range` 数组限定行范围（如 [11,12]，[-1] 到文件尾）\n' +
      '* 长输出会被截断并标记 `<response clipped>`',
    usageGuide: 'harness 标准命令式编辑器（Claude 系工具）：view 查看、create 创建、str_replace 精确替换（唯一匹配）、insert 行后插入。与 edit_file 相比更适合『需要先查看行号、再精确替换』的流程；带行号输出方便后续定位。',
    category: '文件',
    parameters: {
      type: 'object',
      properties: {
        command: { type: 'string', description: '要执行的命令：view / create / str_replace / insert（必填）' },
        path: { type: 'string', description: '文件或目录路径（工作区内；view 支持目录，其他命令须为文件）' },
        file_text: { type: 'string', description: 'create 命令的文件内容' },
        insert_line: { type: 'integer', description: 'insert 命令：在此行之后插入 new_str（1 基）' },
        new_str: { type: 'string', description: 'str_replace 的替换新内容 / insert 的插入内容' },
        old_str: { type: 'string', description: 'str_replace 的原文（须精确且唯一匹配）' },
        view_range: { type: 'array', items: { type: 'integer' }, description: 'view 命令行范围，如 [11,12]；[-1] 表示到文件尾' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['command', 'path'],
    },
  },
  {
    name: 'run_code',
    description: '执行一段代码并返回输出（对齐 deepseek-harness run_code）。language: auto（默认，按内容探测）/ go / python / node。',
    usageGuide: 'harness 标准代码执行工具：快速验证算法/处理数据/调用本地库，不用写临时文件。与 bash 的区别：直接执行代码片段（自动建临时文件）。',
    category: '执行',
    parameters: {
      type: 'object',
      properties: {
        code: { type: 'string', description: '要执行的代码（必填）' },
        language: { type: 'string', description: '可选：auto（默认，按内容探测）/ go / python / node' },
      },
      required: ['code'],
    },
  },
]

return {
  name: 'tool-harness',
  purpose: 'harness 核心协议工具（read/write/edit/glob/grep/bash/str_replace_editor/run_code）——迁移自内置 RegisterHarnessTools',
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
        // execute：编排调宿主执行器（宿主 Go 实现经 claimTool 存档；二进制方案）
        execute: (args) => ctx.hostTool.exec(t.name, args || {}),
      })
    }
    // 日志已省略（logger 需 inject 声明）
  },
}
