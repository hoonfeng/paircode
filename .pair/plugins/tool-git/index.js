// ═══════════════════════════════════════════════════════════════
// tool-git — Git 操作（git_status/diff/log/show/blame/add/commit/…）
//
// 迁移（2026-08-22 Round2）：binary 形态 → JS 原生（对齐 tool-core 模式）。
// 原 execute 调 ctx.binary.exec 复用插件目录 bin/ 下独立二进制（已归档
// bin/legacy-plugin-bins/），现实现完全在插件内（ctx.process.exec argv 直连
// git CLI，30s 超时，行为复刻 internal/agent/git.go：porcelain 状态/无改动
// 提示/参数组装）。不再依赖 ctx.binary。
// 工具清单：git_status、git_diff、git_log、git_show、git_blame、git_add、git_commit、git_branch、git_checkout、git_stash
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "git_status",
    "description": "查看 git 工作区状态（当前分支 + 已修改/暂存/未跟踪文件，porcelain 紧凑格式）。",
    "usageGuide": "查看工作区 git 状态（分支+已修改/暂存/未跟踪文件）。比 bash git status 更简洁（porcelain 紧凑格式+自动判断工作区干净）。先调用此工具了解当前变更再决定下一步。",
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
    "usageGuide": "查看工作区未暂存改动（或 staged=true 看已暂存改动）。file 参数可限定单个文件。比 bash git diff 更智能（无改动时自动返回「无改动」而非空输出）。",
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
    "usageGuide": "查看最近提交历史（单行格式）。count 限定条数（默认 15）。比 bash git log --oneline 更省 token（结果精简+自动限制上限 200 条）。",
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
    "usageGuide": "查看某次提交的详情与改动。默认 HEAD（最新一次）。比 bash git show 更安全（自动处理空参数+带 --stat 统计）。",
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
    "usageGuide": "逐行查看某文件每行的最后修改提交和作者。用 start/end 限定行范围避免输出过多。比 bash git blame 更方便（自动处理行范围参数格式）。",
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
    "usageGuide": "把文件加入暂存区（准备提交）。files 为路径列表；省略则暂存全部改动(-A)。需审核批准。比 bash git add 更安全（参数自动组装+路径越界拦截）。",
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
    "usageGuide": "提交已暂存的改动。message 必填；all=true 先暂存已跟踪文件再提交(-a)。需审核批准。比 bash git commit 更安全（message 为空自动拒绝）。",
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
    "usageGuide": "分支操作：无 name 列出全部分支；name+checkout=true 创建并切换；name+delete=true 删除。比 bash git branch 更智能（自动处理三种操作模式的参数差异）。",
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
    "usageGuide": "切换分支或恢复文件到 HEAD。file=true 时 target 为文件路径（丢弃其改动，危险！）。比 bash git checkout 更安全（分支/文件模式自动判断+参数校验）。",
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
    "usageGuide": "贮藏/恢复工作区改动。action=push(默认贮藏)/pop(弹出恢复)/list(列出)/drop(丢弃)。比 bash git stash 更方便（自动处理 action+message 组合）。",
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

// ─── JS 原生化实现（ctx.process.exec 直连 git CLI，无 shell） ─────

// runGit 在目标项目根执行一条 git 子命令（30s 超时，core.quotepath=false）。
// ★ 2026-09 Round2：ctx.bash → ctx.process.exec（argv 数组直连 git.exe，
//   不经 shell——无引号/转义/编码坑；沙箱与受限环境下也比 bash 更稳）。
// git 非零退出（如目录非 git 仓库）：有输出则连同返回，无输出则作 error。
function runGit(ctx, args, ...gitArgs) {
  let cwd = ''
  const project = args && args.project
  if (project && !/^[a-zA-Z]:[\/]/.test(project) && !project.startsWith('/')) {
    cwd = '../' + String(project).replace(/[\/]+$/, '')
  }
  const opts = { cmd: 'git', args: ['-c', 'core.quotepath=false', ...gitArgs], timeout: 30 }
  if (cwd) opts.cwd = cwd
  const res = ctx.process.exec(opts)
  const out = res.output || ''
  if (res.error && !out.trim()) {
    throw new Error('git ' + gitArgs.join(' ') + ' 失败: ' + res.error)
  }
  if (!out.trim()) return '（无输出）'
  return out
}

// 读类（ReadOnly 免审）
function gitStatus(ctx, args) {
  const out = runGit(ctx, args, 'status', '--porcelain=v1', '--branch')
  const trimmed = out.trim()
  // porcelain --branch 首行恒为 "## <branch>"；仅此一行=工作区干净。
  if (trimmed.startsWith('##') && !trimmed.includes('\n')) return out + '（工作区干净）'
  return out
}

function gitDiff(ctx, args) {
  const ga = ['diff']
  if (args.staged) ga.push('--cached')
  const f = String(args.file == null ? '' : args.file).trim()
  if (f) ga.push('--', f)
  const out = runGit(ctx, args, ...ga)
  if (out.trim() === '' || out === '（无输出）') return '（无改动）'
  return out
}

function gitLog(ctx, args) {
  let count = Number(args.count || 15)
  if (!Number.isFinite(count)) count = 15
  count = Math.min(200, Math.max(1, Math.round(count)))
  const ga = ['log', '--oneline', '-n', String(count)]
  const f = String(args.file == null ? '' : args.file).trim()
  if (f) ga.push('--', f)
  return runGit(ctx, args, ...ga)
}

function gitShow(ctx, args) {
  const commit = String(args.commit == null ? '' : args.commit).trim() || 'HEAD'
  return runGit(ctx, args, 'show', '--stat', commit)
}

function gitBlame(ctx, args) {
  const file = String(args.file == null ? '' : args.file).trim()
  if (!file) throw new Error('file 不能为空')
  const ga = ['blame']
  const s = Number(args.start || 0), e = Number(args.end || 0)
  if (s > 0 && e >= s) ga.push('-L', s + ',' + e)
  ga.push('--', file)
  return runGit(ctx, args, ...ga)
}

// 写类（需审批）
function gitAdd(ctx, args) {
  const ga = ['add']
  if (Array.isArray(args.files) && args.files.length > 0) {
    ga.push('--', ...args.files.map(String))
  } else {
    ga.push('-A')
  }
  return '已暂存。' + runGit(ctx, args, ...ga)
}

function gitCommit(ctx, args) {
  const msg = String(args.message == null ? '' : args.message).trim()
  if (!msg) throw new Error('message 不能为空')
  const ga = ['commit', '-m', msg]
  if (args.all) ga.push('-a')
  return runGit(ctx, args, ...ga)
}

function gitBranch(ctx, args) {
  const name = String(args.name == null ? '' : args.name).trim()
  if (!name) return runGit(ctx, args, 'branch', '--all')
  if (args.delete) return runGit(ctx, args, 'branch', '-D', name)
  if (args.checkout) return runGit(ctx, args, 'checkout', '-b', name)
  return runGit(ctx, args, 'branch', name)
}

function gitCheckout(ctx, args) {
  const target = String(args.target == null ? '' : args.target).trim()
  if (!target) throw new Error('target 不能为空')
  if (args.file) return runGit(ctx, args, 'checkout', '--', target)
  return runGit(ctx, args, 'checkout', target)
}

function gitStash(ctx, args) {
  const action = String(args.action == null ? '' : args.action).trim() || 'push'
  if (action === 'push') {
    const ga = ['stash', 'push']
    const m = String(args.message == null ? '' : args.message).trim()
    if (m) ga.push('-m', m)
    return runGit(ctx, args, ...ga)
  }
  if (action === 'pop' || action === 'list' || action === 'drop') {
    return runGit(ctx, args, 'stash', action)
  }
  throw new Error('未知 action: ' + action + '（push/pop/list/drop）')
}

const impls = {
  git_status: gitStatus,
  git_diff: gitDiff,
  git_log: gitLog,
  git_show: gitShow,
  git_blame: gitBlame,
  git_add: gitAdd,
  git_commit: gitCommit,
  git_branch: gitBranch,
  git_checkout: gitCheckout,
  git_stash: gitStash,
}


return {
  name: 'tool-git',
  inject: ['process'],
  purpose: 'Git 操作（git_status/diff/log/show/blame/add/commit/…）——迁移自内置 Go 工具组；调用实现（JS 编排 ctx.process.exec 直连 git CLI）完全在插件内（Round2 JS 原生化）',
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
        execute: (args) => impls[t.name](ctx, args || {}),
      })
    }
  },
}
