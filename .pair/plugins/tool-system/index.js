// ═══════════════════════════════════════════════════════════════
// tool-system — 系统内部工具（update_tasks/update_plan/tool_stats/history_search/history_list/history_count）——SystemTool 同样可更换
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// 工具清单：history_search、history_list、history_count、update_plan、tool_stats、update_tasks
// ═══════════════════════════════════════════════════════════════
const tools = [
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
  }
];

return {
  name: 'tool-system',
  purpose: '系统内部工具（update_tasks/update_plan/tool_stats/history_search/history_list/history_count）——SystemTool 同样可更换（自动生成，迁移自内置 Go 工具组）',
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
