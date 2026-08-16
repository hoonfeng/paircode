// ═══════════════════════════════════════════════════════════════
// tool-office — 办公文档（csv_read/csv_write/json_to_table/table_stats/text_report/word_read）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool
// 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：csv_read、csv_write、json_to_table、table_stats、text_report、word_read、word_write、read_xlsx、write_xlsx、read_pdf、markdown_to_html
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "csv_read",
    "description": "读取 CSV/TSV 文件并以 Markdown 表格形式返回内容。参数 delimiter 可选 \"comma\"（逗号, 默认）或 \"tab\"（制表符）。columns 按列索引过滤（从 0 开始，逗号分隔，如 \"0,2,3\"）。limit 限制返回行数（默认 100，-1=全部），offset 跳过前 N 行。",
    "usageGuide": "读取 CSV/TSV 文件并以 Markdown 表格形式返回。比直接 read_file 读 CSV 更友好（自动解析分隔符+格式化表格）。delimiter 可指定 comma/tab。",
    "parameters": {
      "properties": {
        "columns": {
          "description": "可选：要显示的列索引（从 0 开始，逗号分隔），省略显示全部",
          "type": "string"
        },
        "delimiter": {
          "description": "可选：分隔符，\"comma\"（逗号）或 \"tab\"（制表符），默认 \"comma\"",
          "type": "string"
        },
        "limit": {
          "description": "可选：最大返回行数（默认 100，-1 表示全部）",
          "type": "integer"
        },
        "offset": {
          "description": "可选：跳过前 N 行（默认 0）",
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
    "name": "csv_write",
    "description": "将表格数据写入 CSV/TSV 文件。data 为 JSON 二维数组（如 [[\"列1\",\"列2\"],[\"值1\",\"值2\"]]）或 Markdown 表格文本。delimiter 可选 \"comma\"（逗号, 默认）或 \"tab\"（制表符）。header 为可选的表头行 JSON 数组，省略则从 data 首行自动提取。",
    "usageGuide": "将表格数据写入 CSV/TSV 文件。data 参数支持 JSON 二维数组或 Markdown 表格文本。比手动拼接 CSV 更高效（自动处理转义+分隔符）。需审核批准。",
    "parameters": {
      "properties": {
        "data": {
          "description": "表格数据：JSON 二维数组字符串，或 Markdown 表格文本",
          "type": "string"
        },
        "delimiter": {
          "description": "可选：分隔符 \"comma\" 或 \"tab\"，默认 \"comma\"",
          "type": "string"
        },
        "header": {
          "description": "可选：表头行 JSON 数组，如 \"[\"姓名\",\"年龄\"]\"",
          "type": "string"
        },
        "path": {
          "description": "文件路径（工作区内）",
          "type": "string"
        }
      },
      "required": [
        "path",
        "data"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "json_to_table",
    "description": "将 JSON 数组字符串转为 Markdown 表格。columns 指定列名和顺序（逗号分隔，如 \"name,age\"），省略则使用全部键并按字母序排列。limit 限制行数（默认 100，-1=全部），title 可选表格标题。",
    "usageGuide": "将 JSON 数组字符串转为 Markdown 表格。columns 参数指定列名和顺序。比手动格式化更高效（自动生成表头+对齐）。",
    "parameters": {
      "properties": {
        "columns": {
          "description": "可选：显示的键名（逗号分隔），省略则用全部键",
          "type": "string"
        },
        "json": {
          "description": "JSON 数组字符串（必填，如 [{\"name\":\"张三\",\"age\":30}]）",
          "type": "string"
        },
        "limit": {
          "description": "可选：最大行数（默认 100，-1=全部）",
          "type": "integer"
        },
        "title": {
          "description": "可选：表格标题",
          "type": "string"
        }
      },
      "required": [
        "json"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "table_stats",
    "description": "对表格数据的数值列做基本统计（求和、均值、最大值、最小值、计数）。data 为 CSV 文本、JSON 数组或文件路径。format 指定数据格式：\"csv\"（CSV 文本）\"json\"（JSON 数组）\"file\"（文件路径）。group_by 按指定列分组统计（可选）。",
    "usageGuide": "对表格数据的数值列做基本统计（求和/均值/最大/最小/计数）。group_by 可按某列分组统计。比手动计算更快（自动识别数值列+分组聚合）。",
    "parameters": {
      "properties": {
        "data": {
          "description": "数据：CSV 文本、JSON 数组字符串、或文件路径（根据 format）",
          "type": "string"
        },
        "format": {
          "description": "数据格式：\"csv\"（默认）/ \"json\" / \"file\"",
          "type": "string"
        },
        "group_by": {
          "description": "可选：按此列名分组统计",
          "type": "string"
        }
      },
      "required": [
        "data"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "text_report",
    "description": "扫描工作区目录树，按文件扩展名分组统计行数。支持统计总行数、代码行（非空非纯注释）、注释行、空行。path 限定扫描子目录（默认工作区根）；extensions 限定文件扩展名（逗号分隔，如 \".go,.ts,.vue\"）；group_by 分组方式：\"ext\"（按扩展名，默认）或 \"dir\"（按目录）。自动跳过 .git/node_modules/vendor 等目录。",
    "usageGuide": "扫描目录树，按文件扩展名或目录分组统计代码行数。快速了解项目规模和技术栈分布。比 run_command wc -l 更智能（自动跳过 .git/node_modules+按类型分组）。",
    "parameters": {
      "properties": {
        "extensions": {
          "description": "可选：限定文件扩展名，逗号分隔（如 \".go,.ts,.vue\"）",
          "type": "string"
        },
        "group_by": {
          "description": "可选：分组方式 \"ext\"（按扩展名，默认）或 \"dir\"（按目录）",
          "type": "string"
        },
        "max_files": {
          "description": "可选：最大扫描文件数（默认 5000）",
          "type": "integer"
        },
        "path": {
          "description": "可选：要扫描的目录路径（默认工作区根）",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "word_read",
    "description": "读取 Microsoft Word (.docx) 文件的内容，以纯文本或 Markdown 格式返回。支持段落文本、表格、列表等基本结构提取。format 可选 \"text\"（纯文本，默认）或 \"markdown\"（Markdown 格式）。limit 限制返回字符数（默认 10000，防止内容过长）。",
    "usageGuide": "读取 Microsoft Word (.docx) 文件内容，以纯文本或 Markdown 格式返回。比手动打开 Word 更高效（直接提取文本到上下文）。",
    "parameters": {
      "properties": {
        "format": {
          "description": "可选：输出格式 \"text\"（纯文本，默认）或 \"markdown\"",
          "type": "string"
        },
        "limit": {
          "description": "可选：最大返回字符数（默认 10000，-1=全部）",
          "type": "integer"
        },
        "path": {
          "description": "Word 文件路径（工作区内，.docx 格式）",
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
    "name": "word_write",
    "description": "生成 Microsoft Word (.docx) 文档。content 为 Markdown 格式文本（支持 # 标题、普通段落、- 列表项、| 表格），系统自动将其转换为 OOXML 格式写入 .docx 文件。title 为可选的文档标题（默认无）。",
    "usageGuide": "生成 Microsoft Word (.docx) 文档。content 为 Markdown 格式文本。用于输出报告/文档。比手动排版更高效（Markdown 转 Word 格式）。需审核批准。",
    "parameters": {
      "properties": {
        "content": {
          "description": "文档内容（Markdown 格式：标题用 #、列表用 -、表格用 |）",
          "type": "string"
        },
        "path": {
          "description": "输出文件路径（工作区内，.docx 扩展名）",
          "type": "string"
        },
        "title": {
          "description": "可选：文档标题",
          "type": "string"
        }
      },
      "required": [
        "path",
        "content"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "read_xlsx",
    "description": "读取 Microsoft Excel (.xlsx) 文件的内容，以 Markdown 表格形式返回各工作表。sheet 指定工作表名称（默认第一个）；limit 限制行数（默认 200，-1=全部）。纯 Go 标准库实现（解析 ZIP + XML），零外部依赖。",
    "usageGuide": "读取 Excel (.xlsx) 文件内容，以 Markdown 表格形式返回。sheet 参数指定工作表名。比直接 read_file 更友好（自动解析+多 sheet 支持）。",
    "parameters": {
      "properties": {
        "limit": {
          "description": "可选：最大行数（默认 200，-1=全部）",
          "type": "integer"
        },
        "path": {
          "description": "Excel 文件路径（工作区内，.xlsx 格式）",
          "type": "string"
        },
        "sheet": {
          "description": "可选：工作表名称（默认第一个工作表）",
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
    "name": "write_xlsx",
    "description": "创建 Microsoft Excel (.xlsx) 文件。data 为 JSON 二维数组（如 [[\"列1\",\"列2\"],[\"值1\",\"值2\"]]）或 Markdown 表格文本。sheet 为工作表名称（默认 \"Sheet1\"）。纯 Go 标准库实现（生成 ZIP + XML），零外部依赖。",
    "usageGuide": "创建 Excel (.xlsx) 文件。data 为 JSON 二维数组或 Markdown 表格文本。比手动拼接 CSV 更专业（支持多 sheet+格式）。需审核批准。",
    "parameters": {
      "properties": {
        "data": {
          "description": "表格数据：JSON 二维数组字符串 或 Markdown 表格文本",
          "type": "string"
        },
        "path": {
          "description": "输出文件路径（工作区内，.xlsx 扩展名）",
          "type": "string"
        },
        "sheet": {
          "description": "可选：工作表名称（默认 \"Sheet1\"）",
          "type": "string"
        }
      },
      "required": [
        "path",
        "data"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "read_pdf",
    "description": "提取 PDF 文件的文本内容。先尝试纯文本提取（解析 PDF 流对象）；如提取结果为空或内容极少（\u003c50 字符），自动调用 pdftoppm/mutool 将页面渲染为图片 + Tesseract OCR 识别文字。page 指定页码（从 1 开始，默认全部）；limit 限制返回字符数（默认 10000）。",
    "usageGuide": "提取 PDF 文本内容。先尝试纯文本提取，失败则回退 OCR。比手动打开 PDF 复制更高效（直接提取到上下文）。",
    "parameters": {
      "properties": {
        "limit": {
          "description": "可选：最大返回字符数（默认 10000，-1=全部）",
          "type": "integer"
        },
        "page": {
          "description": "可选：页码（从 1 开始），省略则提取全部页面",
          "type": "integer"
        },
        "path": {
          "description": "PDF 文件路径（工作区内）",
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
    "name": "markdown_to_html",
    "description": "将 Markdown 文本转换为 HTML 片段。支持 # 标题、**粗体**、*斜体*、`行内代码`、```代码块```、- 无序列表、1. 有序列表、| 表格、\u003e 引用、[链接](url)、![图片](url)。full_html 为 true 时输出完整 HTML 文档（含 DOCTYPE + head + body），否则只输出 body 内的 HTML 片段。",
    "usageGuide": "将 Markdown 文本转为 HTML。支持 full_html 参数输出完整 HTML 文档（含 title）。比手动转换更快（内置渲染器+代码高亮）。",
    "parameters": {
      "properties": {
        "full_html": {
          "description": "可选：是否输出完整 HTML 文档（默认 false，只输出片段）",
          "type": "boolean"
        },
        "markdown": {
          "description": "Markdown 文本（必填）",
          "type": "string"
        },
        "title": {
          "description": "可选：完整 HTML 时的页面标题",
          "type": "string"
        }
      },
      "required": [
        "markdown"
      ],
      "type": "object"
    },
    "readOnly": true
  }
];

return {
  name: 'tool-office',
  purpose: '办公文档（csv_read/csv_write/json_to_table/table_stats/text_report/word_read）（自动生成，迁移自内置 Go 工具组）',
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
