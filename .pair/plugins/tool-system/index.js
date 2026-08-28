// ═══════════════════════════════════════════════════════════════
// tool-system — 系统内部工具（SystemTool + Skills/MCP/市场：update_tasks/update_plan/tool_stats/history_*/skill_*/mcp_*）——全部可更换
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// ★ 2026-08-29（t2 集成修复）：移除 generate_commit_message——宿主无对应 Go 实现
//   （零消费方），claimTool 无存档导致 hostTool 执行必失败；与生成器白名单对齐。
// 工具清单：skill_list、load_skill、load_skill_resource、skill_write、skill_delete、mcp_list、mcp_add、mcp_remove、history_search、history_list、history_count、update_plan、tool_stats、update_tasks
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
    "description": "创建或更新一个技能（写入 .pair/skills/\u003c名\u003e/SKILL.md）。",
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
    "name": "update_plan",
    "description": "维护任务计划清单：传入完整步骤列表（每步 step 描述 + status：pending/in_progress/done）。复杂任务应先用它列出计划，执行中随时更新状态（每次传全量整份清单）。清单会展示给用户。",
    "parameters": {
      "properties": {
        "plan": {
          "description": "完整计划步骤（全量；状态变化时重传整份）",
          "items": {
            "properties": {
              "status": {
                "description": "状态",
                "enum": [
                  "pending",
                  "in_progress",
                  "done"
                ],
                "type": "string"
              },
              "step": {
                "description": "步骤描述",
                "type": "string"
              }
            },
            "required": [
              "step",
              "status"
            ],
            "type": "object"
          },
          "type": "array"
        }
      },
      "required": [
        "plan"
      ],
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
    "name": "todo_write",
    "description": "维护任务列表：传入完整任务清单（全量替换），系统自动持久化到磁盘（DSH todo_write 别名，语义同 update_tasks）。每项包含 subject（必填）、status（pending/in_progress/completed/cancelled）、description（可选）、dependencies（可选）、plan_step_index（可选，整数）。",
    "usageGuide": "管理持久化任务列表（全量替换模式，DSH todo_write 别名）。复杂任务（3+ 步）必须拆解为子任务并逐项追踪。每次传入完整清单，状态变化时重传整份。",
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
              "plan_step_index": {
                "description": "所属 plan 步骤索引（0 基；自主模式下绑定到 update_plan 的某步）",
                "type": "integer"
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
    "name": "update_tasks",
    "description": "维护任务列表：传入完整任务清单（全量替换），系统自动持久化到磁盘。每项包含 subject（必填）、status（pending/in_progress/completed/cancelled）、description（可选）、dependencies（可选）、plan_step_index（可选，整数）。plan_step_index 用于在自主模式下将子任务绑定到 update_plan 的某个步骤（0=第1步，1=第2步…）。普通模式下忽略此字段。",
    "usageGuide": "管理持久化任务列表（全量替换模式）。复杂任务（3+ 步）必须拆解为子任务并逐项追踪。每次传入完整清单，状态变化时重传整份。系统自动持久化到磁盘。plan_step_index 用于自主模式下绑定到 update_plan 的步骤。",
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
              "plan_step_index": {
                "description": "所属 plan 步骤索引（0 基；自主模式下绑定到 update_plan 的某步）",
                "type": "integer"
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
  }
];

return {
  name: 'tool-system',
  purpose: '系统内部工具（SystemTool + Skills/MCP/市场：update_tasks/todo_write/update_plan/tool_stats/history_*/skill_*/mcp_*）——全部可更换（自动生成，迁移自内置 Go 工具组）',
  apply(ctx) {
    // ★ R2-7 别名（DSH 命名对齐）：todo_write → update_tasks（宿主执行器同名承载）
    const hostExec = { todo_write: 'update_tasks' }
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
        execute: (args) => ctx.hostTool.exec(hostExec[t.name] || t.name, args || {}),
      })
    }
  },
}
