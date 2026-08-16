// ═══════════════════════════════════════════════════════════════
// tool-codegraph — 代码知识图谱（codegraph_build/search/impact/…）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool
// 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：codegraph_build、codegraph_stats、codegraph_file_structure、codegraph_function、codegraph_class、codegraph_callers、codegraph_callees、codegraph_impact、codegraph_search、codegraph_git_history、codegraph_entity_history、codegraph_get_edit_context、codegraph_find_related_tests、codegraph_analyze_complexity、codegraph_search_by_pattern、codegraph_trace_call_chain、codegraph_find_dead_code、codegraph_module_architecture
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "codegraph_build",
    "description": "构建或重建代码知识图谱。解析项目所有 Go 源文件，提取文件、包、函数、方法、结构体、接口、变量、常量等实体，以及包含、定义、调用、导入等关系。支持增量更新（只重新解析变更的文件）。参数 rebuild=true 强制全量重建。多项目工作区（gou-ide/wb-ui/ref）用 project 指定项目，默认主项目。",
    "usageGuide": "构建或重建代码知识图谱。项目代码变更后运行此工具让图谱保持最新。之后可用其他 codegraph_* 工具做符号级精确搜索。比全文搜索更精确（基于 AST 多语言解析）。多项目工作区用 project 参数指定目标项目（如 wb-ui），各项目图谱独立存储/独立查询。",
    "parameters": {
      "properties": {
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        },
        "rebuild": {
          "description": "可选：强制全量重建（默认 false，增量更新）",
          "type": "boolean"
        }
      },
      "type": "object"
    }
  },
  {
    "name": "codegraph_stats",
    "description": "获取代码知识图谱的统计信息：文件数、实体数、关系数，以及按类型（函数/方法/结构体/接口…）的分布。",
    "usageGuide": "获取当前项目代码知识图谱的统计信息（实体/关系数量、按类型分布）。构建后运行了解图谱覆盖度。支持 project 指定项目（多项目工作区）。",
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
    "name": "codegraph_file_structure",
    "description": "获取指定文件的实体结构树（文件→函数/类型→方法/字段的层次结构）。用于理解文件内部的组织结构。",
    "usageGuide": "获取指定文件的实体结构树（文件→函数/类型→方法/字段的层次）。比 list_files 更深入（了解文件内部组织）。",
    "parameters": {
      "properties": {
        "file": {
          "description": "文件路径（工作区相对路径，如 'cmd/companion/main.go'）",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "required": [
        "file"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_function",
    "description": "按名称查找函数/方法的定义位置。支持函数名、包名.函数名、或接收者.方法名。返回文件路径、行号、签名等信息。",
    "usageGuide": "按名称查找函数/方法的定义位置，携带函数签名。支持包名.函数名、接收者.方法名。比 search_content 全文搜索更精确（基于 AST 直接定位）。搜函数定义首选此工具。",
    "parameters": {
      "properties": {
        "name": {
          "description": "函数名（如 'main'、'ServeHTTP'、'foo.Bar'）",
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
    "name": "codegraph_class",
    "description": "获取类型（struct/interface）的完整层次结构：字段、方法、嵌入类型。支持结构体名或接口名。",
    "usageGuide": "获取类型（struct/interface）的完整层次结构：字段、方法、嵌入类型。比 read_file 逐个文件翻更高效（聚合所有相关定义）。",
    "parameters": {
      "properties": {
        "name": {
          "description": "类型名（如 'Server'、'Handler'）",
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
    "name": "codegraph_callers",
    "description": "查询哪些函数调用了指定的函数/方法。用于理解函数被使用的情况。返回调用者的文件路径和行号。",
    "usageGuide": "查询哪些函数调用了指定的函数。修改函数签名/行为前必调此工具了解调用方，防止漏改。比 search_content 搜索引用更精确（基于调用图）。",
    "parameters": {
      "properties": {
        "name": {
          "description": "函数/方法名（如 'SendRequest'、'handler.Handle'）",
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
    "name": "codegraph_callees",
    "description": "查询指定的函数/方法调用了哪些其他函数。用于理解函数的内部调用情况。返回被调用者的名称和调用位置。",
    "usageGuide": "查询指定函数内部调用了哪些函数。理解函数实现逻辑时用。比 read_file 手动翻更快（聚合被调函数列表）。",
    "parameters": {
      "properties": {
        "name": {
          "description": "函数/方法名（如 'handleRequest'）",
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
    "name": "codegraph_impact",
    "description": "分析修改某个函数/类型/文件后可能影响的范围。基于调用图进行可达性分析，返回受影响的文件、函数列表和传播路径。用于回答「修改这个函数会影响哪些地方？」",
    "usageGuide": "分析修改某函数/类型/文件后的影响范围（传递调用链）。修改核心代码前必调此工具。比 check_impact 更精确（函数级调用链而非文件级导入链）。",
    "parameters": {
      "properties": {
        "entity": {
          "description": "实体标识（函数名、类型名或文件路径，如 'SendRequest'、'cmd/main.go'）",
          "type": "string"
        },
        "maxDepth": {
          "description": "可选：搜索深度（默认 10，限制传递链长度）",
          "type": "integer"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "required": [
        "entity"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_search",
    "description": "在代码知识图谱中搜索实体（函数、类型、变量、文件等）。支持按名称搜索和按类型过滤。返回匹配实体的位置、签名和相关度评分。比 search_content 更精确，因为基于结构化理解而非纯文本匹配。",
    "usageGuide": "在代码知识图谱中搜索实体（函数/类型/变量/文件等）。scope 限定类型（function/type/variable/file）。搜函数/类型定义首选此工具，其次才是 search_content。比全文搜索精确一个数量级。",
    "parameters": {
      "properties": {
        "maxResults": {
          "description": "可选：最大返回数（默认 20）",
          "type": "integer"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        },
        "query": {
          "description": "搜索关键词（函数名、类型名、变量名等）",
          "type": "string"
        },
        "scope": {
          "description": "可选：搜索范围，可选值: all(全部)/file(文件)/function(函数)/type(类型)/variable(变量)/package(包)，默认 all",
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
    "name": "codegraph_git_history",
    "description": "查询 Git 提交历史，并关联到代码实体。可以查询最近提交、影响某个文件的提交，或者某个实体的变更历史。用于回答「这个函数是谁改的？」、「这个 bug 是哪次提交引入的？」",
    "usageGuide": "查询 Git 提交历史并关联到代码实体。file 参数限定文件；count 控制条数。比 git_log 更丰富（关联实体变更信息）。",
    "parameters": {
      "properties": {
        "count": {
          "description": "可选：返回提交数（默认 20，最大 100）",
          "type": "integer"
        },
        "file": {
          "description": "可选：文件路径，查询影响该文件的提交历史",
          "type": "string"
        },
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
    "name": "codegraph_entity_history",
    "description": "查询指定代码实体的完整变更历史：谁在什么时候修改了它，以及对应的提交消息。实体可以是函数名、类型名或文件路径。",
    "usageGuide": "查询指定代码实体的完整变更历史：谁在什么时候修改了它。比 git_blame 更友好（聚合到实体级别而非行级别）。",
    "parameters": {
      "properties": {
        "entity": {
          "description": "实体名（函数名、类型名或文件路径）",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "required": [
        "entity"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_get_edit_context",
    "description": "获取修改某个代码位置所需的完整上下文。一次调用返回：符号源码、调用者列表、关联测试、近期 Git 历史、相关记忆。比分别调用多个工具更高效。参数 maxTokens 控制返回内容的 token 预算。",
    "usageGuide": "获取修改某代码位置所需的完整上下文。调用 edit_file 前先用此工具获取周边代码，减少多次 read_file 的 token 消耗。",
    "parameters": {
      "properties": {
        "file": {
          "description": "文件路径（工作区相对路径，如 'cmd/main.go'）",
          "type": "string"
        },
        "line": {
          "description": "行号（1 基，目标函数/类型所在行）",
          "type": "integer"
        },
        "maxTokens": {
          "description": "可选：token 预算上限（默认 4000，0 不限）",
          "type": "integer"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "required": [
        "file",
        "line"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_find_related_tests",
    "description": "查找与指定函数/方法关联的测试。发现方式：（1）测试函数调用了目标函数；（2）命名约定匹配（TestXxx ↔ Xxx）。返回测试文件路径、行号和源码片段。",
    "usageGuide": "查找与某函数关联的测试。按两种方式发现：（1）测试调用了目标函数 （2）测试名与目标名相近。改完函数后调此工具找到需要运行的测试。",
    "parameters": {
      "properties": {
        "function": {
          "description": "函数/方法名（如 'SendRequest'、'handler.Handle'）",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "required": [
        "function"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_analyze_complexity",
    "description": "测量代码圈复杂度，用于评估重构优先级。返回每个函数的复杂度评分（1=最低）、等级（A-E）和行数。复杂度 \u003e10 建议考虑重构，\u003e20 为高风险。file 指定文件分析单个文件，省略则分析所有函数。",
    "usageGuide": "测量代码圈复杂度，评估重构优先级。高复杂度函数优先重构。比人工评估更客观（基于控制流图计算）。",
    "parameters": {
      "properties": {
        "file": {
          "description": "可选：文件路径（工作区相对路径），分析单个文件；省略则分析全部",
          "type": "string"
        },
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
    "name": "codegraph_search_by_pattern",
    "description": "用正则表达式在代码实体的名称、签名、文档注释中搜索。比 codegraph_search 更精确，支持 scope 过滤（name/signature/docstring/any）。支持按实体类型过滤（function/method/struct/interface/variable）。",
    "usageGuide": "用正则表达式在代码实体名、签名、文档注释中搜索。比 search_content 更结构化（只搜实体级元信息而非全文）。scope 可选 name/signature/docstring。",
    "parameters": {
      "properties": {
        "entityKind": {
          "description": "可选：实体类型过滤，如 function/method/struct/interface/variable",
          "type": "string"
        },
        "maxResults": {
          "description": "可选：最大返回数（默认 50）",
          "type": "integer"
        },
        "pattern": {
          "description": "正则表达式，如 'unwrap\\(\\)'、'SELECT .* FROM'、'TODO'",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        },
        "scope": {
          "description": "可选：搜索范围，any(默认)/name/signature/docstring",
          "type": "string"
        }
      },
      "required": [
        "pattern"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_trace_call_chain",
    "description": "追踪函数/方法的调用链。支持 callers（反向追踪谁调用了它）、callees（正向追踪它调用了谁）、both（双向）。maxDepth 控制追踪深度（默认 5）。返回树形调用链。",
    "usageGuide": "追踪函数调用链：callers（反向：谁调了我）、callees（正向：我调了谁）、both（双向）。比 codegraph_callers/callees 更灵活（支持多级深度追踪）。",
    "parameters": {
      "properties": {
        "direction": {
          "description": "可选：callers(反向)/callees(正向)/both(双向)，默认 callers",
          "type": "string"
        },
        "function": {
          "description": "函数/方法名（如 'SendRequest'、'handler.Handle'）",
          "type": "string"
        },
        "maxDepth": {
          "description": "可选：最大深度（默认 5）",
          "type": "integer"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "required": [
        "function"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "codegraph_find_dead_code",
    "description": "检测项目中疑似没有被调用的函数、类型、变量。判定方式：函数无 incoming RelCalls 边 + 无其他引用。注意：Go 反射和接口分发可能误报，结果仅供参考。",
    "usageGuide": "检测项目中疑似未被调用的函数、类型、变量。定期运行清理死代码，保持项目整洁。",
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
    "name": "codegraph_module_architecture",
    "description": "获取一个目录/模块的架构概览。返回：文件数、函数数、导出函数列表、类型列表、外部依赖、内部依赖、复杂度热点。用于快速理解一个模块的职责和结构。",
    "usageGuide": "获取某目录/模块的架构概览：文件列表+导出符号+导入关系。新接触一个目录时先用此工具了解整体结构。",
    "parameters": {
      "properties": {
        "path": {
          "description": "目录路径（工作区相对路径，如 'cmd/companion/agent'）",
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
  }
];

return {
  name: 'tool-codegraph',
  purpose: '代码知识图谱（codegraph_build/search/impact/…）（自动生成，迁移自内置 Go 工具组）',
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
