// ═══════════════════════════════════════════════════════════════
// tool-system — 系统内部工具（SystemTool + Skills/MCP/市场：update_tasks/tool_stats/history_*/skill_*/mcp_*）——全部可更换
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。
// ★ 2026-08-29（t2 集成修复）：移除 generate_commit_message——宿主无对应 Go 实现
//   （零消费方），claimTool 无存档导致 hostTool 执行必失败；与生成器白名单对齐。
// 工具清单：skill_list、load_skill（path 区分 L2 正文/L3 子资源）、skill_write、skill_delete、mcp_list、mcp_add、mcp_remove、
//         history_search、history_list、history_count、tool_stats、update_tasks、ask_user、
//         progress_checker + goal(op=create/get/edit/pause/resume/complete/blocked)
// ★ 2026-09-12（剩余插件审查轮）：task_create 工具面已移除——任务写入收敛到
//   update_tasks（全量替换）。原描述引用不存在的 task_update 属误导；宿主
//   session_manager 会话级注册与会话桥路由（能力层）保留，需要时可按名恢复。
// ★ 2026-09-04 合并：tool-progress 插件（进度检查）并入本插件；tool-progress 插件目录已删除。
// ★ 2026-09 Round3 ③.4 合并：tool-goal 插件（create_goal/get_goal/update_goal，
//   宿主 goal.go 状态机 + 自动续轮）并入本插件；tool-goal 插件目录已删除。
// ★ 2026-09 工具面合并：goal 3 工具合并为单工具 goal(op=create/get/edit/pause/
//   resume/complete/blocked)——op 分派到宿主内部路由名执行器（goal.go 保留
//   create_goal/get_goal/update_goal 三个 ArchiveHostTool 存档）。
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
    "description": "加载某技能的完整 SKILL.md 正文（L2 渐进式披露），或该技能的子资源文件（L3：传 path）。所有层级（内置/工作区/全局）同名时工作区优先。",
    "usageGuide": "加载技能全文或子资源（渐进式披露）：先看系统提示中的技能清单（名/描述），需要细则时 load_skill name=xxx 取正文；references/assets/scripts 子文件用 path 指定。",
    "parameters": {
      "properties": {
        "name": {
          "description": "技能名",
          "type": "string"
        },
        "path": {
          "description": "可选：技能内子资源相对路径（如 references/xxx.md）；省略则加载 SKILL.md 正文",
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
    "name": "skill_write",
    "description": "创建或更新一个技能（目录式 <skills>/名/SKILL.md）。默认写当前工作区级（.pair/skills/）；传 scope=global 写全局（跨工作区生效，随程序安装目录共享）。",
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
        },
        "scope": {
          "description": "层级：project=工作区级（默认，仅当前工作区）/ global=全局（跨工作区，<InstallDir>/.pair/skills/）",
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
    "description": "删除一个技能（工作区级默认；scope=global 删全局，scope=system 删内置）。",
    "parameters": {
      "properties": {
        "name": {
          "description": "技能名",
          "type": "string"
        },
        "scope": {
          "description": "层级：project=工作区级（默认）/ global=全局 / system=内置",
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
    "description": "删除一个 MCP 服务器。scope 可选 user 或 project（默认 user；project 需当前工作区有该服务器）。",
    "parameters": {
      "properties": {
        "name": {
          "description": "服务器名",
          "type": "string"
        },
        "scope": {
          "description": "层级：user（默认，全局）或 project（工作区级）",
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
    name: 'goal',
    description:
      '同会话完成目标管理（goal 范式）：创建/查看/更新目标状态。op=create 创建（objective 必填）；op=get 查看当前目标；op=edit 改 objective/max_goal_rounds；op=pause 暂停自动续轮；op=resume 恢复；op=complete 标记完成；op=blocked 标记阻塞（blocked_reason 必填）。创建后每轮结束自动续轮推进，直到 complete/blocked 或达轮次上限。',
    parameters: {
      type: 'object',
      properties: {
        op: { type: 'string', enum: ['create', 'get', 'edit', 'pause', 'resume', 'complete', 'blocked'], description: '操作：create 创建 / get 查看 / edit 修改 / pause 暂停续轮 / resume 恢复续轮 / complete 完成 / blocked 阻塞' },
        objective: { type: 'string', description: 'create/edit 用：目标描述（祈使句，直接给出，如「修复登录超时 bug」）' },
        max_goal_rounds: { type: 'integer', description: 'create/edit 用：自动续轮上限（默认 3；0=不限——慎用，会无限续轮）' },
        revision: { type: 'integer', description: 'edit/pause/resume/complete/blocked 用：当前 revision（goal(op=get) 返回；乐观锁，冲突拒绝）' },
        goal_id: { type: 'string', description: '可选：目标 ID（=会话 ID；通常省略——按当前会话自动路由）' },
        blocked_reason: { type: 'string', description: 'blocked 用：阻塞原因（必填）' },
      },
      required: ['op'],
    },
    systemTool: true,
  },
];

return {
  name: 'tool-system',
  purpose: '系统内部工具（SystemTool + Skills/MCP/市场 + 进度检查 + goal：update_tasks/todo_write/tool_stats/history_*/skill_*/mcp_*/progress_checker/goal(op=…)）（tool-progress/tool-goal 已并入）',
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
        // goal/load_skill 单工具：按参数分派到宿主内部路由名执行器
        // （goal.go 三个 ArchiveHostTool；skill 双执行器 load_skill/load_skill_resource）。
        execute: t.name === 'goal'
          ? (args) => {
              const a = Object.assign({}, args || {})
              const op = a.op
              delete a.op
              if (op === 'create') return ctx.hostTool.exec('create_goal', a)
              if (op === 'get') return ctx.hostTool.exec('get_goal', a)
              if (['edit', 'pause', 'resume', 'complete', 'blocked'].includes(op)) {
                a.action = op
                return ctx.hostTool.exec('update_goal', a)
              }
              return Promise.resolve(
                'goal：op 无效（可用 create/get/edit/pause/resume/complete/blocked）——未执行任何操作'
              )
            }
          : t.name === 'load_skill'
            ? (args) => {
                const a = Object.assign({}, args || {})
                if (a.path) return ctx.hostTool.exec('load_skill_resource', a)
                return ctx.hostTool.exec('load_skill', a)
              }
            : (args) => ctx.hostTool.exec(t.name, args || {}),
      })
    }
  },
}
