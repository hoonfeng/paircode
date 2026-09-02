// ═══════════════════════════════════════════════════════════════
// tool-core — 核心文件工具（multi_edit/move_file/delete_file）
//
// 迁移来源（2026-08-16）：内置 registerCoreTools（internal/agent/tools.go）
// → 磁盘外置插件。2026-08-16 JS 原生化：调用实现在插件内（ctx.fs/ctx.bash），
// 宿主只留底层文件/进程服务。保留 requiresApproval 元数据（审核门在 agent
// 层，loop.go 按元数据触发审批）。差异（记录 project.md）：无编辑快照/行号
// 偏移追踪（editHistory）、无变更回调（UI 刷新）；project 多项目路由用
// ../<project>/ 相对路径近似（resolvePath 多根归属检查路由）。
// ★ 2026-09 Round2：read/write/edit/bash 等 harness 名工具由 tool-harness
//   承载，本插件仅保留 multi_edit/move_file/delete_file 三个组合工具。
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    name: 'multi_edit',
    description: '对一个文件按顺序应用多处替换（edits：每项 old_string→new_string 或 line_start/line_end 行号定位）。匹配策略同 edit。原子：任一步失败则全部不写。保留文件原换行风格。',
    usageGuide: '按顺序对一个文件应用多处替换。比多次 edit 更高效（原子提交：任一步失败全部回滚）。编辑项较多时用 multi_edit 替代多次 edit 调用。',
    category: '文件',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '文件路径' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
        edits: {
          type: 'array',
          description: '按顺序应用的替换列表',
          items: {
            type: 'object',
            properties: {
              old_string: { type: 'string', description: '待替换原文（须唯一；line_start>0 时可省略或作校验）' },
              new_string: { type: 'string', description: '替换后的新文' },
              line_start: { type: 'integer', description: '可选：1 基起始行号，>0 时启用行号定位模式' },
              line_end: { type: 'integer', description: '可选：1 基结束行号（含）；省略只替换 line_start 一行' },
            },
            required: ['new_string'],
          },
        },
      },
      required: ['path', 'edits'],
    },
  },
  {
    name: 'move_file',
    description: '移动/重命名文件（from → to，工作区内）。',
    usageGuide: '移动/重命名文件。路径越界自动拦截+变更回调。',
    category: '文件',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        from: { type: 'string', description: '源路径' },
        to: { type: 'string', description: '目标路径' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['from', 'to'],
    },
  },
  {
    name: 'delete_file',
    description: '删除一个文件（工作区内，不可恢复，谨慎）。为安全不删目录。',
    usageGuide: '删除工作区内的文件（不可恢复，谨慎）。为安全不删目录（删除目录请用 bash rmdir）。需审核批准。',
    category: '文件',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '要删除的文件路径' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['path'],
    },
  },
]


// ─── JS 原生化实现（ctx.fs / ctx.bash） ────────────────────

// 项目路由：project 非空 → 以 ../<project>/ 前缀拼相对路径（多根归属由
// ctx.fs.resolve 检查）；绝对路径直接传（越界由 resolve 拦截）。
function projPath(ctx, args, path) {
  const project = args.project
  if (project && path && !/^[a-zA-Z]:[\/]/.test(path) && !path.startsWith('/')) {
    return '../' + String(project).replace(/[\/]+$/, '') + '/' + path.replace(/^[\/]+/, '')
  }
  return path
}

// 检测文件主要换行风格（CRLF vs LF），用于保留原风格
function detectEOL(text) {
  const crlf = (text.match(/\r\n/g) || []).length
  const lf = (text.match(/\n/g) || []).length - crlf
  return crlf > 0 && crlf >= lf ? '\r\n' : '\n'
}

// 读文件（含二进制保护：含 NUL 拒绝）
function readFileText(ctx, args, path) {
  const text = ctx.fs.readFile(path)
  if (text.includes('\0')) {
    throw new Error('「' + args.path + '」是二进制文件，read 不支持读取二进制内容；请用 inspect_binary 工具查看（hexdump/类型嗅探）')
  }
  return text
}

// read：offset/limit 分页 + 2000 行截断
function readFile(ctx, args) {
  const path = projPath(ctx, args, args.path)
  const text = readFileText(ctx, args, path)
  const offset = Number(args.offset || 0)
  const limit = Number(args.limit || 0)
  if (offset <= 0 && limit <= 0) {
    const lines = text.split('\n')
    if (lines.length > 2000) {
      return lines.slice(0, 2000).join('\n') + '\n…[文件共 ' + lines.length + ' 行，仅显示前 2000；用 offset/limit 读其余]'
    }
    return text
  }
  const lines = text.split('\n')
  const start = offset - 1
  if (start < 0) start = 0
  if (start >= lines.length) throw new Error('offset ' + offset + ' 超出文件行数 ' + lines.length)
  let end = lines.length
  if (limit > 0 && start + limit < end) end = start + limit
  return lines.slice(start, end).join('\n')
}

// write：父目录自动创建
function writeFile(ctx, args) {
  const path = projPath(ctx, args, args.path)
  const slash = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'))
  if (slash > 0) {
    const dir = path.slice(0, slash)
    if (dir && !ctx.fs.exists(dir)) ctx.fs.mkdir(dir, true)
  }
  ctx.fs.writeFile(path, args.content)
  return '已写入 ' + args.path
}

// 行号定位替换：返回替换后的行数组
function applyLineReplace(lines, ed) {
  const ls = Number(ed.line_start || 0)
  if (ls <= 0) return { ok: false }
  const le = ed.line_end && Number(ed.line_end) >= ls ? Number(ed.line_end) : ls
  if (ls > lines.length) return { ok: false, err: 'line_start ' + ls + ' 超出文件行数 ' + lines.length }
  const end = Math.min(le, lines.length)
  const newText = String(ed.new_string == null ? '' : ed.new_string).split('\n')
  return { ok: true, lines: lines.slice(0, ls - 1).concat(newText, lines.slice(end)) }
}

// edit 匹配：精确 → CRLF 归一化；返回 {ok, text} 或 {ok:false, err}
function applyEdit(text, ed) {
  const newStr = String(ed.new_string == null ? '' : ed.new_string)
  // 行号定位模式（最可靠）
  if (ed.line_start && Number(ed.line_start) > 0) {
    const r = applyLineReplace(text.split('\n'), ed)
    if (!r.ok) return r.err ? { ok: false, err: r.err } : { ok: false, err: '行号定位失败' }
    return { ok: true, text: r.lines.join('\n') }
  }
  if (ed.old_string == null) return { ok: false, err: '缺少 old_string（且未用 line_start 行号定位）' }
  const oldStr = String(ed.old_string)
  // 精确匹配（统计全部出现位置，>1 报唯一性错误）
  const candidates = []
  {
    let from = 0
    for (;;) {
      const i = text.indexOf(oldStr, from)
      if (i < 0) break
      candidates.push(i)
      from = i + Math.max(oldStr.length, 1)
    }
  }
  if (candidates.length === 0) {
    // CRLF 归一化：全文 \r\n → \n 后匹配，替换时保留原换行风格
    const norm = text.replace(/\r\n/g, '\n')
    const oldNorm = oldStr.replace(/\r\n/g, '\n')
    const ni = norm.indexOf(oldNorm)
    if (ni >= 0) {
      const eol = detectEOL(text)
      const newNorm = newStr.replace(/\r\n/g, '\n').split('\n').join(eol)
      return { ok: true, text: norm.slice(0, ni) + newNorm + norm.slice(ni + oldNorm.length) }
    }
  }
  if (candidates.length === 0) return { ok: false, err: '未找到待替换文本（尝试精确/CRLF 归一化均失败）；请改用 line_start/line_end 行号定位' }
  if (candidates.length > 1) return { ok: false, err: 'old_string 在文件中出现 ' + candidates.length + ' 处（须唯一）；请用 line_start/line_end 行号定位' }
  return { ok: true, text: text.slice(0, candidates[0]) + newStr + text.slice(candidates[0] + oldStr.length) }
}

// edit
function editFile(ctx, args) {
  const path = projPath(ctx, args, args.path)
  const text = readFileText(ctx, args, path)
  const r = applyEdit(text, args)
  if (!r.ok) throw new Error(r.err)
  ctx.fs.writeFile(path, r.text)
  return '已编辑 ' + args.path
}

// multi_edit：原子（内存中逐项应用，全部成功才写回）
function multiEdit(ctx, args) {
  const path = projPath(ctx, args, args.path)
  let text = readFileText(ctx, args, path)
  const edits = Array.isArray(args.edits) ? args.edits : []
  for (let i = 0; i < edits.length; i++) {
    const r = applyEdit(text, edits[i])
    if (!r.ok) throw new Error('第 ' + (i + 1) + ' 项编辑失败: ' + r.err)
    text = r.text
  }
  ctx.fs.writeFile(path, text)
  return '已应用 ' + edits.length + ' 处编辑到 ' + args.path
}

// move_file：rename
function moveFile(ctx, args) {
  const from = projPath(ctx, args, args.from)
  const to = projPath(ctx, args, args.to)
  ctx.fs.rename(from, to)
  return '已移动 ' + args.from + ' → ' + args.to
}

// delete_file
function deleteFile(ctx, args) {
  const path = projPath(ctx, args, args.path)
  ctx.fs.rm(path, false)
  return '已删除 ' + args.path
}

const impls = {
  multi_edit: multiEdit,
  move_file: moveFile,
  delete_file: deleteFile,
}


return {
  name: 'tool-core',
  inject: ['fs'],
  purpose: '核心文件工具（multi_edit/move_file/delete_file）——迁移自内置 registerCoreTools；调用实现（JS 编排 ctx.fs）完全在插件内（Round2：read/write/edit 由 tool-harness 承载；③.4：bash 从工具侧移除）',
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
        execute: (args) => impls[t.name](ctx, args || {}),
      })
    }
    // 日志已省略（logger 需 inject 声明）
  },
}
