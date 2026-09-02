// ═══════════════════════════════════════════════════════════════
// tool-system — 系统内部工具（SystemTool + Skills/MCP/市场：update_tasks/tool_stats/history_*/skill_*/mcp_*）——全部可更换
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// ★ 2026-08-29（t2 集成修复）：移除 generate_commit_message——宿主无对应 Go 实现
//   （零消费方），claimTool 无存档导致 hostTool 执行必失败；与生成器白名单对齐。
// 工具清单：skill_list、load_skill、load_skill_resource、skill_write、skill_delete、mcp_list、mcp_add、mcp_remove、
//         history_search、history_list、history_count、tool_stats、todo_write、update_tasks、ask_user、task_create、
//         progress_checker + goal 组 create_goal/get_goal/update_goal
// ★ 2026-09-04 合并：tool-progress 插件（进度检查）并入本插件；tool-progress 插件目录已删除。
// ★ 2026-09 Round3 ③.4 合并：tool-goal 插件（create_goal/get_goal/update_goal，
//   宿主 goal.go 状态机 + 自动续轮）并入本插件；tool-goal 插件目录已删除。
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "skill_list",
    "description": "列出所有可用技能（名/描述/激活模式/层级）。",
    "parameters": {
      "properties": {},
      "required": [],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "load_skill",
    "description": "加载某技能的完整 SKILL.md 正文（L2 渐进式披露）。",
    "parameters": {
      "properties": {
        "name": {
          "description": "技能名",
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
    "name": "load_skill_resource",
    "description": "加载某技能的子资源文件（L3 渐进式披露）。",
    "parameters": {
      "properties": {
        "name": {
          "description": "技能名",
          "type": "string"
        },
        "path": {
          "description": "资源相对路径",
          "type": "string"
        }
      },
      "required": [
        "name",
        "path"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "skill_write",
    "description": "创建或更新一个技能（写入 .pair/skills/<名>/SKILL.md）。",
    "parameters": {
      "properties": {
        "content": {
          "description": "技能正文",
          "type": "string"
        },
        "description": {
          "description": "一句话描述",
          "type": "string"
        },
        "mode": {
          "description": "激活模式：auto/always/manual，默认 auto",
          "type": "string"
        },
        "name": {
          "description": "技能名",
          "type": "string"
        }
      },
      "required": [
        "name",
        "content"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "skill_delete",
    "description": "删除一个项目级技能。",
    "parameters": {
      "properties": {
        "name": {
          "description": "技能名",
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
    "name": "mcp_list",
    "description": "列出已配置的 MCP 服务器。",
    "parameters": {
      "properties": {},
      "required": [],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "mcp_add",
    "description": "新增一个 MCP 服务器。scope 可选 user 或 project。",
    "parameters": {
      "properties": {
        "args": {
          "items": {
            "type": "string"
          },
          "type": "array"
        },
        "command": {
          "description": "启动命令",
          "type": "string"
        },
        "name": {
          "description": "服务器名",
          "type": "string"
        },
        "scope": {
          "description": "user/project",
          "type": "string"
        }
      },
      "required": [
        "name",
        "command"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "mcp_remove",
    "description": "删除一个 MCP 服务器。",
    "parameters": {
      "properties": {
        "name": {
          "description": "服务器名",
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
    "name": "history_search",
    "description": "按关键词搜索已完成对话的历史记录（标题/摘要/标签/关键点）。",
    "parameters": {
      "properties": {
        "query": {
          "description": "搜索关键词",
          "type": "string"
        }
      },
      "required": [],
      "type": "object"
    },
    "readOnly": true,
    "systemTool": true
  },
  {
    "name": "history_list",
    "description": "列出所有已完成对话的历史记录（按完成时间倒序）。",
    "parameters": {
      "properties": {},
      "required": [],
      "type": "object"
    },
    "readOnly": true,
    "systemTool": true
  },
  {
    "name": "history_count",
    "description": "查询已完成对话的历史记录总数。",
    "parameters": {
      "properties": {},
      "required": [],
      "type": "object"
    },
    "readOnly": true,
    "systemTool": true
  },
  {
    "name": "tool_stats",
    "description": "查看工具调用统计（成功率、调用次数）。按工具名聚合，显示每个工具的调用次数/成功数/失败数/成功率。可使用 min_calls 过滤低频工具，recent 查看最近调用记录。Agent 可用此数据识别高频失败工具，主动优化或创建新工具替代。",
    "parameters": {
      "properties": {
        "min_calls": {
          "description": "可选：最少调用次数过滤（默认0=全部显示）",
          "type": "integer"
        },
        "recent": {
          "description": "可选：显示最近 N 条调用记录（不传则不显示）",
          "type": "integer"
        },
        "source": {
          "description": "可选：按来源过滤，\"builtin\" | \"mcp\"（不传=全部）",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true,
    "systemTool": true
  },

  {
    "name": "update_tasks",
    "description": "维护任务列表：传入完整任务清单（全量替换），系统自动持久化到磁盘。每项包含 subject（必填）、status（pending/in_progress/completed/cancelled）、description（可选）、dependencies（可选）、",
    "usageGuide": "管理持久化任务列表（全量替换模式）。复杂任务（3+ 步）必须拆解为子任务并逐项追踪。每次传入完整清单，状态变化时重传整份。系统自动持久化到磁盘。",
    "parameters": {
      "properties": {
        "tasks": {
          "description": "完整任务列表（全量；状态变化时重传整份）",
          "items": {
            "properties": {
              "dependencies": {
                "description": "依赖的任务 ID 列表（可选）",
                "items": {
                  "type": "string"
                },
                "type": "array"
              },
              "description": {
                "description": "详细描述（可选）：做什么、涉及哪些文件",
                "type": "string"
              },
              "id": {
                "description": "任务 ID（可选，不传则自动生成）",
                "type": "string"
              },
              "status": {
                "description": "状态",
                "enum": [
                  "pending",
                  "in_progress",
                  "completed",
                  "cancelled"
                ],
                "type": "string"
              },
              "subject": {
                "description": "任务标题，用祈使句（如\"修复登录超时\"）",
                "type": "string"
              }
            },
            "required": [
              "subject",
              "status"
            ],
            "type": "object"
          },
          "type": "array"
        }
      },
      "required": [
        "tasks"
      ],
      "type": "object"
    },
    "systemTool": true
  },
  {
    "name": "ask_user",
    "description": "向用户提问并等待回答（用于关键决策、歧义澄清，别滥用）。question 必填（或 questions 数组多问题）；askType 可选(text/single/multi/single-with-input)，默认 text 纯文本输入；options 可选(选择类 question 的选项列表；single-with-input 时用户可另选或自定义输入)。多问题：questions:[{id, question, options?, multi_select?}]（questions 优先，缺省回落单问题；Round3 ⑤ 前端已支持多问题渲染与 answers 回灌）。调用会阻塞直到用户回答。",
    "parameters": {
      "properties": {
        "askType": {
          "description": "提问类型：text(纯文本)/single(单选)/multi(多选)/single-with-input(单选+自由输入)",
          "enum": [
            "text",
            "single",
            "multi",
            "single-with-input"
          ],
          "type": "string"
        },
        "options": {
          "description": "选择类问题用：可选项列表",
          "items": {
            "type": "string"
          },
          "type": "array"
        },
        "question": {
          "description": "向用户提出的问题（单问题路径；与 questions 二选一）",
          "type": "string"
        },
        "questions": {
          "description": "多问题数组（与 question 二选一；questions 优先，前端按多问题卡片渲染、一次提交 answers 回灌）",
          "items": {
            "properties": {
              "id": {
                "description": "问题稳定 id（回答回显用）",
                "type": "string"
              },
              "multi_select": {
                "description": "是否多选",
                "type": "boolean"
              },
              "options": {
                "description": "可选项（选择题用）",
                "items": {
                  "type": "string"
                },
                "type": "array"
              },
              "question": {
                "description": "问题文本",
                "type": "string"
              }
            },
            "required": [
              "id",
              "question"
            ],
            "type": "object"
          },
          "type": "array"
        }
      },
      "required": [
        "question"
      ],
      "type": "object"
    },
    "systemTool": true
  },
  {
    "name": "task_create",
    "description": "创建新的子任务。创建后必须立即执行该任务：先调用 task_update 标记为 in_progress 开始执行，执行完成后调用 task_update 标记为 completed 并说明结果。重复此流程直到所有子任务完成。",
    "parameters": {
      "properties": {
        "dependencies": {
          "description": "依赖的任务 ID 列表",
          "items": {
            "type": "string"
          },
          "type": "array"
        },
        "description": {
          "description": "详细描述：做什么、涉及哪些文件。不要包含文件原始内容，只写摘要。",
          "type": "string"
        },
        "subject": {
          "description": "任务标题，用祈使句（如\"修复登录超时\"）",
          "type": "string"
        }
      },
      "required": [
        "subject",
        "description"
      ],
      "type": "object"
    },
    "systemTool": true,
    "usageGuide": "创建子任务并追踪执行进度。复杂任务（3+ 步）必须拆解为子任务，每完成一项更新状态（in_progress→completed）。依赖项用 dependencies 参数关联。比手动记清单更可靠（持久化到磁盘+状态自动管理）。"
  },
  {
    "name": "progress_checker",
    "description": "检查当前任务完成进度，输出结构化进度报告，识别未完成的任务并给出执行建议。使用场景：任务列表较长时、Agent 不确定下一步做什么时、或用户要求查看进度时。",
    "parameters": {
      "properties": {
        "detail": {
          "description": "可选：详细模式，设为 \"full\" 显示每个任务的详细信息（含描述）",
          "enum": [
            "summary",
            "full"
          ],
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  }
];

// ─── goal 工具（2026-09 Round3 ③.4 并入自 tool-goal：宿主 goal.go 执行器）───
// 编排在插件、能力在宿主：schema/描述在插件，状态机与自动续轮在宿主
// （internal/agent/goal.go）。execute 经 ctx.hostTool.exec 路由回宿主执行器
// （_convID 由宿主工具执行链自动注入，多会话并发不串）。

const goalTools = [
  {
    name: 'create_goal',
    description:
      '创建同会话完成目标（对齐 goal）。objective 必填（直接给出目标，不做推断）；max_goal_rounds 可选（自动续轮上限，默认 3）。创建后会话将在每轮结束后自动续轮推进，直到 update_goal complete/blocked 或达轮次上限。',
    parameters: {
      type: 'object',
      properties: {
        objective: { type: 'string', description: '目标描述（祈使句，直接给出，如「修复登录超时 bug」）' },
        max_goal_rounds: { type: 'integer', description: '可选：自动续轮上限（默认 3；0=不限——慎用，会无限续轮）' },
      },
      required: ['objective'],
    },
    systemTool: true,
  },
  {
    name: 'get_goal',
    description:
      '读取当前会话目标（goal_id/revision/objective/phase/rounds/roundLimit/blockerReason/armed）。无目标返回提示。',
    parameters: { type: 'object', properties: {}, required: [] },
    readOnly: true,
    systemTool: true,
  },
  {
    name: 'update_goal',
    description:
      '更新当前会话目标（对齐 goal update）。action ∈ {edit,pause,resume,complete,blocked}；revision 必传（乐观锁，冲突拒绝）。edit 可改 objective/max_goal_rounds；pause 停续轮、resume 重挂；complete 标记完成；blocked 标记阻塞（blocked_reason 必填说明）。',
    parameters: {
      type: 'object',
      properties: {
        goal_id: { type: 'string', description: '目标 ID（=会话 ID；get_goal 可查）' },
        revision: { type: 'integer', description: '当前 revision（get_goal 返回；冲突时拒绝）' },
        action: { type: 'string', description: 'edit / pause / resume / complete / blocked' },
        objective: { type: 'string', description: 'edit 用：新目标描述（可选）' },
        max_goal_rounds: { type: 'integer', description: 'edit 用：新自动续轮上限（可选）' },
        blocked_reason: { type: 'string', description: 'blocked 用：阻塞原因（必填）' },
      },
      required: ['goal_id', 'revision', 'action'],
    },
    systemTool: true,
  },
];

return {
  name: 'tool-system',
  purpose: '系统内部工具（SystemTool + Skills/MCP/市场 + 进度检查 + goal：update_tasks/todo_write/tool_stats/history_*/skill_*/mcp_*/progress_checker/create_goal/get_goal/update_goal）（tool-progress/tool-goal 已并入）',
  apply(ctx) {
    const all = tools.concat(goalTools)
    for (const t of all) {
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
