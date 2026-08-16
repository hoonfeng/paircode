// ═══════════════════════════════════════════════════════════════
// tool-search — 搜索工具（search_content/search_files）
//
// 迁移来源（2026-08-16）：内置 registerSearchTools（internal/agent/search.go）
// → 磁盘外置插件。2026-08-16 二进制定位：execute 调 ctx.binary 复用统一
// 宿主二进制（fs-search 组在 RegisterDefaultTools，编码探测/跳过目录在 Go 实现）。
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    name: 'search_content',
    description: '在工作区内按正则搜索文件内容，返回匹配的「相对路径:行号: 行文本」。pattern 为 RE2 正则；path 限定子目录（省略=根）；glob 按文件名过滤（如 *.go）；case_insensitive 忽略大小写；max_results 上限（默认 200）。自动跳过 .git/node_modules 等与二进制/超大文件。',
    usageGuide: '搜索文件内容（全文搜索）。比 run_command findstr/grep 更精确（跳过 .git/node_modules、自动处理编码、结果结构化）。搜索函数/类型定义请优先用 codegraph_search（基于 AST，更精确）。',
    category: '代码搜索',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        pattern: { type: 'string', description: 'RE2 正则表达式' },
        path: { type: 'string', description: '限定子目录（省略=工作区根）' },
        glob: { type: 'string', description: '文件名通配过滤，如 *.go' },
        case_insensitive: { type: 'boolean', description: '忽略大小写' },
        max_results: { type: 'integer', description: '结果行数上限（默认 200）' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['pattern'],
    },
  },
  {
    name: 'search_files',
    description: '在工作区内按通配符递归查找文件，返回相对路径列表（已排序）。pattern 为通配符：不含 / 时匹配文件名（如 *.go、*config*），含 / 时匹配相对路径（如 internal/*/main.go）；path 限定子目录；language 可选按语言过滤；max_results 上限（默认 500）。跳过 .git/node_modules 等。',
    usageGuide: '按文件名/路径模式搜索文件。比 run_command dir /s 更高效。配合 language 参数可按语言过滤。',
    category: '代码搜索',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        pattern: { type: 'string', description: '文件名/路径通配符，如 *.go' },
        path: { type: 'string', description: '限定子目录（省略=工作区根）' },
        language: { type: 'string', description: '可选：按语言过滤，如 "go"、"typescript"、"python"' },
        max_results: { type: 'integer', description: '结果上限（默认 500）' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['pattern'],
    },
  },
]

return {
  name: 'tool-search',
  purpose: '搜索工具（search_content/search_files）——迁移自内置 registerSearchTools，api 声明在插件、执行走统一宿主二进制（fs-search 组）',
  apply(ctx) {
    for (const t of tools) {
      ctx.tools.register({
        name: t.name,
        description: t.description,
        usageGuide: t.usageGuide,
        category: t.category,
        readOnly: t.readOnly,
        parameters: t.parameters,
        execute: (args) => ctx.binary.exec(t.name, args || {}),
      })
    }
    // 日志已省略（logger 需 inject 声明）
  },
}
