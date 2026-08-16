// ═══════════════════════════════════════════════════════════════
// tool-project-info — 项目知识库（project_info_write/read/list/search/delete/explore）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool
// 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：project_info_write、project_info_read、project_info_list、project_info_tree、project_info_search、project_info_delete、project_info_explore
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "project_info_write",
    "description": "写入/更新项目知识库的一篇（.pair/project-info/\u003c路径\u003e.md）——记录项目架构/模块职责/数据流/设计决策等结构化理解，跨会话复用、你和用户都能看。★树形路径：顶层分支 目标/架构/实现/关键点/设计思想，根条目用 概览（如 架构/模块-agent / 设计思想/决策-渲染架构）；兼容参考项目 notes/ 前缀路径（自动映射分支+镜像 .agents/notes/）。",
    "usageGuide": "写入/更新项目知识库条目，跨会话复用。★知识库是树：顶层分支 = 目标/架构/实现/关键点/设计思想（根为 概览）——路径带分支前缀（如 架构/模块-agent / 设计思想/决策-渲染架构）。也可用参考项目风格路径 notes/implemented/architecture/x（自动归入树分支 架构/x 并镜像 .agents/notes/）。读完关键文件后立即写入，积累项目的结构化理解。比记在脑子里可靠（持久化+跨会话可见）。多项目工作区可用 project 参数指定目标项目。",
    "parameters": {
      "properties": {
        "content": {
          "description": "Markdown 正文（首行用 # 标题）",
          "type": "string"
        },
        "path": {
          "description": "条目路径（中文，带顶层分支前缀：目标/架构/实现/关键点/设计思想，如 架构/模块-agent），不含 .md；用 / 嵌套为细节篇",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "required": [
        "path",
        "content"
      ],
      "type": "object"
    }
  },
  {
    "name": "project_info_read",
    "description": "读取知识库某篇的全文（按路径，如 概览 / 模块-agent）。渐进式披露的细节层。",
    "usageGuide": "读取知识库某篇全文。渐进式披露：先 project_info_list 看总览，再用此工具读具体细则。比翻目录更方便（自动解析路径+内容格式化）。",
    "parameters": {
      "properties": {
        "path": {
          "description": "条目路径，不含 .md",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
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
    "name": "project_info_list",
    "description": "列出知识库所有条目的【总览】（路径 + 标题 + 分级）。渐进式披露的总览层。",
    "usageGuide": "列出知识库所有条目的总览（路径+标题+分级）。新项目先调此工具查看已有哪些文档，避免重复写入。",
    "parameters": {
      "properties": {
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "project_info_tree",
    "description": "返回知识库完整树形结构（缩进树：目标/架构/实现/关键点/设计思想 分支 + 条目）。人可读的树形导航。",
    "usageGuide": "查看知识库完整树形结构（分支/子类/条目缩进树）。比 project_info_list 更直观：先看树定位条目，再 project_info_read 读全文。",
    "parameters": {
      "properties": {
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "project_info_search",
    "description": "按关键词搜索知识库（匹配路径/标题/正文），返回命中条目。",
    "usageGuide": "按关键词搜索知识库（匹配路径/标题/正文）。想查某个模块/概念是否已有文档时优先用此工具。",
    "parameters": {
      "properties": {
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        },
        "query": {
          "description": "关键词",
          "type": "string"
        }
      },
      "required": [
        "query"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "project_info_delete",
    "description": "删除知识库某篇（按路径）。",
    "usageGuide": "删除知识库某篇（按路径）。知识库条目过时/错误时用此工具清理。删除前建议先 project_info_read 确认。",
    "parameters": {
      "properties": {
        "path": {
          "description": "条目路径，不含 .md",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "required": [
        "path"
      ],
      "type": "object"
    }
  },
  {
    "name": "project_info_explore",
    "description": "返回项目目录结构概览（根目录关键文件、顶层目录及文件数）——构建知识库的起点；据此用 read_file 读关键文件分析，再 project_info_write 写入 概览/模块-*/决策-*。",
    "usageGuide": "扫描项目目录结构概览——构建知识库的起点。新项目首次接触时先调此工具了解项目全貌，再用 read_file 读关键文件，最后 project_info_write 写入结构化理解。",
    "parameters": {
      "properties": {},
      "type": "object"
    },
    "readOnly": true
  }
];

return {
  name: 'tool-project-info',
  purpose: '项目知识库（project_info_write/read/list/search/delete/explore）（自动生成，迁移自内置 Go 工具组）',
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
