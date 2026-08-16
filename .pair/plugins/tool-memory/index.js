// ═══════════════════════════════════════════════════════════════
// tool-memory — 跨会话记忆（memory_write/read/list/search）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool
// 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：memory_write、memory_delete、memory_read、memory_list、memory_search
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "memory_write",
    "description": "写入或【更新】一条持久记忆（跨会话保留在 .pair/memory/）。**先 memory_search/list 查有无相关记忆——有则用其同名覆盖来更新（先 memory_read 读旧的、融合后写回），别为同一主题反复新建、造成碎片化**。name 唯一标识；type: user(用户偏好)/feedback(纠正与确认的做法)/project(项目决策约束)/reference(外部资源指针)；description 一句话摘要；content 正文。",
    "usageGuide": "写入或更新一条持久记忆（跨会话保留）。先 memory_search 查有无相关记忆，有则读旧→融合→同名更新，别反复新建造成碎片化。用于记录用户偏好、项目决策、修复方案等。需审核批准。多项目工作区可用 project 参数指定目标项目。",
    "parameters": {
      "properties": {
        "content": {
          "description": "正文",
          "type": "string"
        },
        "description": {
          "description": "一句话摘要",
          "type": "string"
        },
        "name": {
          "description": "唯一名，用【简短中文】命名（如 数据库连接池配置）；更新已有记忆请用其原名",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        },
        "type": {
          "description": "user/feedback/project/reference",
          "type": "string"
        }
      },
      "required": [
        "name",
        "description",
        "content"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "memory_delete",
    "description": "删除一条过时/错误的记忆（按 name）。保持记忆库精简准确，别让过时信息长期误导。",
    "usageGuide": "删除一条过时/错误的记忆。保持记忆库精简。需审核批准。删除前建议先 memory_read 确认是该条。",
    "parameters": {
      "properties": {
        "name": {
          "description": "记忆名",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "required": [
        "name"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "memory_read",
    "description": "按 name 读取一条记忆的全文。",
    "usageGuide": "按 name 读一条记忆全文。渐进式披露：先 memory_list 看总览，再用此工具读具体细则。比直接读 .pair/memory/ 文件更方便（自动解析 YAML front-matter）。",
    "parameters": {
      "properties": {
        "name": {
          "description": "记忆名",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "required": [
        "name"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "memory_list",
    "description": "列出所有记忆的【总览】（名 + 摘要，渐进式披露的总览层）；要某条细则用 memory_read 读全文。",
    "usageGuide": "列出所有记忆的总览（名+摘要）。先调此工具看有什么记忆，再决定用 memory_read 读哪条。比 run_command dir .pair/memory 更友好（渐进式披露+自动维护索引）。",
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
    "name": "memory_search",
    "description": "按关键词搜索记忆（匹配名/摘要/正文），返回命中条目的名+摘要。",
    "usageGuide": "按关键词搜索记忆（匹配名/摘要/正文）。要查某个主题是否已有记忆时优先用此工具，比 memory_list 遍历更高效。",
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
  }
];

return {
  name: 'tool-memory',
  purpose: '跨会话记忆（memory_write/read/list/search）（自动生成，迁移自内置 Go 工具组）',
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
