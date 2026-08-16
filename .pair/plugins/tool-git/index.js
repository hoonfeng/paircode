// ═══════════════════════════════════════════════════════════════
// tool-git — Git 操作（git_status/diff/log/show/blame/add/commit/…）
//
// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.binary 复用本插件目录 bin/ 下的独立二进制（源码 cmd/plugins/<name>/，改实现重编译即更换）。
// 工具清单：git_status、git_diff、git_log、git_show、git_blame、git_add、git_commit、git_branch、git_checkout、git_stash
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "git_status",
    "description": "查看 git 工作区状态（当前分支 + 已修改/暂存/未跟踪文件，porcelain 紧凑格式）。",
    "usageGuide": "查看工作区 git 状态（分支+已修改/暂存/未跟踪文件）。比 run_command git status 更简洁（porcelain 紧凑格式+自动判断工作区干净）。先调用此工具了解当前变更再决定下一步。",
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
    "name": "git_diff",
    "description": "查看 git 改动。file 可选（限定单个文件）；staged=true 看已暂存(--cached)的改动，否则看工作区未暂存改动。",
    "usageGuide": "查看工作区未暂存改动（或 staged=true 看已暂存改动）。file 参数可限定单个文件。比 run_command git diff 更智能（无改动时自动返回「无改动」而非空输出）。",
    "parameters": {
      "properties": {
        "file": {
          "description": "可选：限定单个文件路径",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        },
        "staged": {
          "description": "看已暂存(--cached)改动，默认看未暂存",
          "type": "boolean"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "git_log",
    "description": "查看最近提交历史（单行格式）。count 限定条数（默认 15）；file 可选（限定某文件的历史）。",
    "usageGuide": "查看最近提交历史（单行格式）。count 限定条数（默认 15）。比 run_command git log --oneline 更省 token（结果精简+自动限制上限 200 条）。",
    "parameters": {
      "properties": {
        "count": {
          "description": "条数（默认 15）",
          "type": "integer"
        },
        "file": {
          "description": "可选：限定某文件的提交历史",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "git_show",
    "description": "查看某次提交的详情与改动。commit=提交哈希或引用（默认 HEAD）。",
    "usageGuide": "查看某次提交的详情与改动。默认 HEAD（最新一次）。比 run_command git show 更安全（自动处理空参数+带 --stat 统计）。",
    "parameters": {
      "properties": {
        "commit": {
          "description": "提交哈希/引用，默认 HEAD",
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
    "name": "git_blame",
    "description": "逐行查看某文件每行的最后修改提交/作者。file 必填；可选 start/end 限定行范围。",
    "usageGuide": "逐行查看某文件每行的最后修改提交和作者。用 start/end 限定行范围避免输出过多。比 run_command git blame 更方便（自动处理行范围参数格式）。",
    "parameters": {
      "properties": {
        "end": {
          "description": "结束行",
          "type": "integer"
        },
        "file": {
          "description": "文件路径",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        },
        "start": {
          "description": "起始行",
          "type": "integer"
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
    "name": "git_add",
    "description": "把文件加入暂存区。files 为路径列表；省略则暂存全部改动(-A)。",
    "usageGuide": "把文件加入暂存区（准备提交）。files 为路径列表；省略则暂存全部改动(-A)。需审核批准。比 run_command git add 更安全（参数自动组装+路径越界拦截）。",
    "parameters": {
      "properties": {
        "files": {
          "description": "文件路径列表（省略=全部）",
          "items": {
            "type": "string"
          },
          "type": "array"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "git_commit",
    "description": "提交已暂存的改动。message 必填；all=true 先暂存所有已跟踪文件改动再提交(-a)。",
    "usageGuide": "提交已暂存的改动。message 必填；all=true 先暂存已跟踪文件再提交(-a)。需审核批准。比 run_command git commit 更安全（message 为空自动拒绝）。",
    "parameters": {
      "properties": {
        "all": {
          "description": "先 -a 暂存已跟踪改动",
          "type": "boolean"
        },
        "message": {
          "description": "提交信息",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "required": [
        "message"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "git_branch",
    "description": "分支操作。无 name=列出全部分支；name+checkout=true 创建并切换；name+delete=true 删除；仅 name=创建。",
    "usageGuide": "分支操作：无 name 列出全部分支；name+checkout=true 创建并切换；name+delete=true 删除。比 run_command git branch 更智能（自动处理三种操作模式的参数差异）。",
    "parameters": {
      "properties": {
        "checkout": {
          "description": "创建后切换过去",
          "type": "boolean"
        },
        "delete": {
          "description": "删除该分支",
          "type": "boolean"
        },
        "name": {
          "description": "分支名（创建/删除时）",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "git_checkout",
    "description": "切换分支，或把文件恢复到 HEAD。target=分支名(切换)；file=true 时 target 为文件路径(丢弃其改动，危险)。",
    "usageGuide": "切换分支或恢复文件到 HEAD。file=true 时 target 为文件路径（丢弃其改动，危险！）。比 run_command git checkout 更安全（分支/文件模式自动判断+参数校验）。",
    "parameters": {
      "properties": {
        "file": {
          "description": "target 是文件(恢复/丢弃改动)",
          "type": "boolean"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        },
        "target": {
          "description": "分支名或文件路径",
          "type": "string"
        }
      },
      "required": [
        "target"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "git_stash",
    "description": "贮藏工作区改动。action：push(默认,贮藏) / pop(弹出恢复) / list(列出) / drop(丢弃最近一条)。",
    "usageGuide": "贮藏/恢复工作区改动。action=push(默认贮藏)/pop(弹出恢复)/list(列出)/drop(丢弃)。比 run_command git stash 更方便（自动处理 action+message 组合）。",
    "parameters": {
      "properties": {
        "action": {
          "description": "push/pop/list/drop，默认 push",
          "type": "string"
        },
        "message": {
          "description": "push 时的备注",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        }
      },
      "type": "object"
    },
    "requiresApproval": true
  }
];

return {
  name: 'tool-git',
  purpose: 'Git 操作（git_status/diff/log/show/blame/add/commit/…）（自动生成，迁移自内置 Go 工具组）',
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
        execute: (args) => ctx.binary.exec(t.name, args || {}),
      })
    }
  },
}
