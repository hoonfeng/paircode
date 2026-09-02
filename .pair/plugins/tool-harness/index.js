// ═══════════════════════════════════════════════════════════════
// tool-harness — harness 核心协议工具（read/write/edit/glob/grep/run_code
// + 后台进程 run_background/read_output/kill_process/job_list）
//
// 迁移来源（2026-08-16）：内置 RegisterHarnessTools（internal/agent/
// harness_tools.go）→ 磁盘外置插件。2026-08-16 第二轮：7 个工具的 execute
// 由 ctx.hostTool（宿主 Go 执行器）改为 **JS 原生化**（调用实现在插件内，
// 底层能力 ctx.fs/ctx.bash，对齐 tool-core 模式）；run_code 保持 hostTool
// ——其「node + tools.xxx 嵌套调度」需 goja VM 宿主运行时（runCodeNested），
// 属框架运行时能力，JS 沙箱不可复刻。
//
// ★ 2026-09 Round3 ③.4 插件瘦身合并：
//   - tool-shell 的 6 个后台进程工具（run_background/read_output/
//     kill_process/job_output/job_list/job_kill）并入本插件（实现同源
//     ctx.process 宿主服务，globalBG 跨轮次存活）；
//   - bash 工具移除：短查询不再暴露 bash 工具名（长进程误用风险），
//     宿主 bash 服务本身保留（fs-api/git-api 的 ctx.bash 仍有依赖）。
// ★ 2026-09 Round4 工具面瘦身：job_output/job_kill（read_output/kill_process
//   纯别名）已删除，仅保留 job_list（独立清单语义）。
//
// 装配：.pair/plugins/ 启动扫描（LoadGlobalPlugins）→ define + load。
// 停用本插件（cordis_stop tool-harness）即回收全部 13 个工具。
// ═══════════════════════════════════════════════════════════════

// ─── JS 原生化实现（ctx.fs / ctx.bash） ────────────────────

// 项目路由：project 非空 → 以 ../<project>/ 前缀拼相对路径（多根归属由
// ctx.fs.resolve 检查）；绝对路径直接传（越界由 resolve 拦截）。
function projPath(ctx, args, path) {
  const project = args.project
  if (project && path && !/^[a-zA-Z]:[\\/]/.test(path) && !path.startsWith('/')) {
    return '../' + String(project).replace(/[\\/]+$/, '') + '/' + path.replace(/^[\\/]+/, '')
  }
  return path
}

// filePath 兼容参数名：参考用 file_path，repo 旧调用方用 path——两者都接受（file_path 优先）。
function filePath(args) {
  const p = args.file_path != null ? args.file_path : args.path
  return p == null ? '' : String(p)
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

// read：约定对齐（2026-09 Round2 R2-7）——file_path 参数 + 行号输出块：
//   <path>/<type>/<content> 内每行 "number: text"，末尾 footer
//   （(End of file - total N lines) 或 (Showing lines X-Y of N. Use offset=… to continue.)）。
// offset(默认 1)/limit(默认 2000) 分页；兼容旧参数名 path。
function readFile(ctx, args) {
  const path = projPath(ctx, args, filePath(args))
  const text = readFileText(ctx, args, path)
  const lines = text.split('\n')
  // 去掉末尾空行（split 尾随 \n 产生）——与 行计数一致
  if (lines.length > 0 && lines[lines.length - 1] === '' ) lines.pop()
  const totalLines = lines.length
  let offset = Math.round(Number(args.offset || 0))
  let limit = Math.round(Number(args.limit || 0))
  if (offset <= 0) offset = 1
  if (limit <= 0) limit = 2000
  let start = offset - 1
  if (start >= totalLines && !(totalLines === 0 && offset === 1)) {
    throw new Error('offset ' + offset + ' 超出文件行数 ' + totalLines)
  }
  if (start < 0) start = 0
  let end = start + limit
  if (end > totalLines) end = totalLines
  const shown = lines.slice(start, end)
  const endLine = shown.length > 0 ? start + shown.length : Math.max(0, offset - 1)
  let footer
  if (endLine < totalLines) footer = '(Showing lines ' + offset + '-' + endLine + ' of ' + totalLines + '. Use offset=' + (endLine + 1) + ' to continue.)'
  else footer = '(End of file - total ' + totalLines + ' lines)'
  const body = shown.length > 0 ? shown.map((l, i) => (start + i + 1) + ': ' + l).join('\n') + '\n\n' + footer : footer
  return '<path>' + (args.file_path != null ? args.file_path : args.path) + '</path>\n<type>file</type>\n<content>\n' + body + '\n</content>'
}

// write：父目录自动创建（file_path/path 双参数名兼容）
function writeFile(ctx, args) {
  const path = projPath(ctx, args, filePath(args))
  if (!path) throw new Error('缺少文件路径（file_path 或 path）')
  const slash = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'))
  if (slash > 0) {
    const dir = path.slice(0, slash)
    if (dir && !ctx.fs.exists(dir)) ctx.fs.mkdir(dir, true)
  }
  ctx.fs.writeFile(path, args.content == null ? '' : String(args.content))
  return '已写入 ' + (args.file_path != null ? args.file_path : args.path)
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
// replace_all=true 时替换全部出现处（约定对齐，2026-09 Round2 R2-7），
// 不再要求 old_string 唯一；默认 false 保持「须唯一」安全语义。
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
  // 精确匹配（统计全部出现位置，>1 报唯一性错误；replace_all 时全部替换）
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
    return { ok: false, err: '未找到待替换文本（尝试精确/CRLF 归一化均失败）；请改用 line_start/line_end 行号定位' }
  }
  if (candidates.length > 1 && !ed.replace_all) {
    return { ok: false, err: 'old_string 在文件中出现 ' + candidates.length + ' 处（须唯一；或设 replace_all=true 全部替换）；请用 line_start/line_end 行号定位' }
  }
  if (candidates.length === 1) {
    return { ok: true, text: text.slice(0, candidates[0]) + newStr + text.slice(candidates[0] + oldStr.length) }
  }
  // replace_all：从后往前替换（偏移不受前次替换影响）
  let out = text
  for (let i = candidates.length - 1; i >= 0; i--) {
    out = out.slice(0, candidates[i]) + newStr + out.slice(candidates[i] + oldStr.length)
  }
  return { ok: true, text: out }
}

// edit
function editFile(ctx, args) {
  const path = projPath(ctx, args, filePath(args))
  if (!path) throw new Error('缺少文件路径（file_path 或 path）')
  const text = readFileText(ctx, args, path)
  const r = applyEdit(text, args)
  if (!r.ok) throw new Error(r.err)
  ctx.fs.writeFile(path, r.text)
  return '已编辑 ' + (args.file_path != null ? args.file_path : args.path)
}

// ★ Round4：str_replace_editor（命令式壳）已删除——read/write/edit/multi_edit 全覆盖，避免重复工具面。


// run_code：统一二进制承载（tool-binary 注册了 run_code——node+tools.xxx
// 嵌套 goja 调度 + 外部进程执行，二进制进程内自持 goja 运行时）
function runCode(ctx, args) {
  const opts = { timeout: 120000 }
  return ctx.binary.exec('run_code', args || {}, opts).text
}

// ─── 后台进程（合并自 tool-shell，同源 ctx.process 宿主服务）───

// run_background：后台启动长命令，返回进程 id
async function runBackground(ctx, args) {
  const command = String(args.command || '').trim()
  if (!command) throw new Error('command 不能为空')
  const { id } = await ctx.process.runBackground(command, args.cwd || '')
  return `已后台启动 id=${id}。用 read_output(id=${id}) 看输出、kill_process(id=${id}) 停止。`
}

// read_output：读取后台进程累积输出与状态
async function readOutput(ctx, args) {
  const { output, done, exitErr, status } = await ctx.process.readOutput(Number(args.id))
  let line = `[${status}]`
  if (done && exitErr) line += `（${exitErr}）`
  const capped = output.length > 16000 ? output.slice(0, 16000) + '\n…[输出截断]' : output
  return `${line}\n${capped}`
}

// kill_process：停止后台进程
async function killProcess(ctx, args) {
  await ctx.process.kill(Number(args.id))
  return `已停止 id=${args.id}`
}

// job_list：列出全部后台进程（job_list 对齐，R2-7）
async function jobList(ctx, args) {
  const jobs = await ctx.process.list()
  if (!jobs || jobs.length === 0) return '（无后台进程）'
  return jobs.map(j => `- id=${j.id} 状态=${j.status}${j.error ? ' 错误=' + j.error : ''}`).join('\n')
}

// glob/grep：ctx.fs（复用 glob/grep 宿主实现）
function globFiles(ctx, args) {
  const opts = {}
  if (args.path) opts.path = args.path
  if (args.language) opts.language = args.language
  if (args.max_results) opts.max_results = args.max_results
  return ctx.fs.glob(args.pattern, opts)
}

function grepFiles(ctx, args) {
  const opts = {}
  if (args.path) opts.path = args.path
  if (args.glob) opts.glob = args.glob
  if (args.case_insensitive) opts.case_insensitive = true
  if (args.max_results) opts.max_results = args.max_results
  return ctx.fs.grep(args.pattern, opts)
}

const tools = [
  {
    name: 'read',
    description: '读取文件内容（对齐 read）。file_path 为工作区内路径（兼容旧参数名 path）；可选 offset(起始行,1 基，默认 1)+limit(行数，默认 2000)读片段；输出带行号（"行号: 内容"）与文件总行数 footer。',
    usageGuide: 'harness 标准读工具：读取文件内容（行号输出对齐约定）。路径越界自动拦截，二进制自动拒绝（改用 inspect_binary）。大文件用 offset+limit 分页。',
    category: '文件',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        file_path: { type: 'string', description: '文件路径（工作区内；参考参数名，与 path 等价）' },
        path: { type: 'string', description: '文件路径（工作区内；旧参数名，file_path 优先）' },
        offset: { type: 'integer', description: '可选：起始行号(1 基，默认 1)' },
        limit: { type: 'integer', description: '可选：读取行数（默认 2000）' },
        project: { type: 'string', description: '可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。' },
      },
    },
    impl: readFile,
  },
  {
    name: 'write',
    description: '把 content 完整写入 file_path（覆盖；父目录自动创建）。需审核批准。',
    usageGuide: 'harness 标准写工具：整文件写入（覆盖）。写类操作需人工确认。如需追加请先 read 再 write 覆盖。',
    category: '文件',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        file_path: { type: 'string', description: '文件路径（参考参数名，与 path 等价）' },
        path: { type: 'string', description: '文件路径（旧参数名，file_path 优先）' },
        content: { type: 'string', description: '完整文件内容' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['content'],
    },
    impl: writeFile,
  },
  {
    name: 'edit',
    description: '把文件中唯一一处 old_string 替换为 new_string（对齐 edit）；replace_all=true 时替换全部出现处。内置智能匹配（CRLF 归一化）；匹配失败优先用 line_start/line_end 行号定位。',
    usageGuide: 'harness 标准编辑工具：小改动（≤5 行）用精确替换（须唯一；多处出现可设 replace_all=true 全部替换）；大改动请用 write 写整段。替换前会自动快照。',
    category: '文件',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        file_path: { type: 'string', description: '文件路径（参考参数名，与 path 等价）' },
        path: { type: 'string', description: '文件路径（旧参数名，file_path 优先）' },
        new_string: { type: 'string', description: '替换后的新文本' },
        old_string: { type: 'string', description: '待替换原文（默认须唯一；replace_all=true 时替换全部；line_start>0 时可省略或作校验）' },
        replace_all: { type: 'boolean', description: '可选：true 时替换 old_string 的全部出现处（默认 false 须唯一）' },
        line_start: { type: 'integer', description: '可选：1 基起始行号（含）；省略或 < line_start 时只替换 line_start 一行' },
        line_end: { type: 'integer', description: '可选：1 基结束行号（含）' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['new_string'],
    },
    impl: editFile,
  },
  {
    name: 'glob',
    description: '按通配符递归查找文件，返回相对路径列表（对齐 glob）。pattern 含 / 或 ** 时按路径模式（如 internal/**/*.go），否则匹配任意深度文件名（如 *.go）；path 限定子目录。',
    usageGuide: 'harness 标准 glob 工具：按路径模式发现文件。跳过 .git/node_modules 等目录。比 shell find 更精确（结构化、防撑爆）。',
    category: '代码搜索',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        pattern: { type: 'string', description: '文件名/路径通配符，如 *.go' },
        path: { type: 'string', description: '可选：限定子目录（省略=工作区根）' },
        language: { type: 'string', description: '可选：按语言过滤，如 "go"、"typescript"' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['pattern'],
    },
    impl: globFiles,
  },
  {
    name: 'grep',
    description: '在工作区内按正则搜索文件内容，返回「相对路径:行号: 行文本」（对齐 grep）。pattern 为 RE2 正则；path 限定子目录；glob 按文件名过滤；case_insensitive 忽略大小写。',
    usageGuide: 'harness 标准 grep 工具：正则全文搜索。搜索函数/类型定义请优先用 codegraph_search（AST 级更精确）。',
    category: '代码搜索',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        pattern: { type: 'string', description: 'RE2 正则表达式' },
        path: { type: 'string', description: '可选：限定子目录（省略=工作区根）' },
        glob: { type: 'string', description: '可选：文件名通配过滤，如 *.go' },
        case_insensitive: { type: 'boolean', description: '可选：忽略大小写' },
        max_results: { type: 'integer', description: '可选：结果行数上限（默认 200）' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['pattern'],
    },
    impl: grepFiles,
  },
  {
    name: 'run_background',
    description: '在后台启动一条长命令，不阻塞 agent 循环（推荐用于 dev server、watch 模式、调试服务等）。返回进程 id，随后用 read_output 读输出、kill_process 停止。如果命令会长期运行或保持监听状态，优先用此工具。短查询请用其他宿主执行通道。',
    usageGuide: '后台启动一条长命令，不阻塞 agent 循环。用于 dev server、npm run dev/watch 模式、调试服务、TCP 监听——这些场景只能用此工具。返回进程 id，之后用 read_output/kill_process 控制。',
    category: '执行',
    parameters: {
      type: 'object',
      properties: {
        command: { type: 'string', description: '要后台执行的命令' },
        cwd: { type: 'string', description: '可选工作目录（工作区内）' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['command'],
    },
    impl: runBackground,
  },
  {
    name: 'read_output',
    description: '读取某后台进程（id）累积的输出与运行状态（运行中/已结束）。',
    usageGuide: '读取后台进程的累积输出与运行状态。需先用 run_background 启动进程获得 id。比直接看终端更方便（自动截断保护+状态标记运行中/已结束）。',
    category: '执行',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        id: { type: 'integer', description: '进程 id' },
      },
      required: ['id'],
    },
    impl: readOutput,
  },
  {
    name: 'kill_process',
    description: '停止某后台进程（id）。只能杀死通过 run_background 启动的进程，无法操作外部进程。',
    usageGuide: '停止某后台进程（仅限通过 run_background 启动的）。进程跑偏/卡死/已不需要时用此工具停止。',
    category: '执行',
    parameters: {
      type: 'object',
      properties: {
        id: { type: 'integer', description: '进程 id' },
      },
      required: ['id'],
    },
    impl: killProcess,
  },

  {
    name: 'job_list',
    description: '列出全部后台任务（id + 状态 running/done/error）（job_list 对齐）。',
    usageGuide: '列出全部后台任务（id+状态）。配合 read_output/kill_process 管理后台任务。',
    category: '执行',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {},
    },
    impl: jobList,
  },


  {
    name: 'run_code',
    description: '执行一段代码并返回输出（对齐 run_code）。language: auto（默认，按内容探测）/ go / python / node。',
    usageGuide: 'harness 标准代码执行工具：快速验证算法/处理数据/调用本地库，不用写临时文件。与 bash 的区别：直接执行代码片段（自动建临时文件）。',
    category: '执行',
    parameters: {
      type: 'object',
      properties: {
        code: { type: 'string', description: '要执行的代码（必填）' },
        language: { type: 'string', description: '可选：auto（默认，按内容探测）/ go / python / node' },
      },
      required: ['code'],
    },
    impl: runCode, // 统一二进制承载（node 嵌套 goja 调度 + 外部进程执行）
  },
]

return {
  name: 'tool-harness',
  purpose: 'harness 核心协议工具（read/write/edit/glob/grep/run_code + 后台进程 run_background/read_output/kill_process/job_list）——迁移自内置 RegisterHarnessTools，2026-09 并入 tool-shell（Round4：str_replace_editor/job_output/job_kill 冗余删除）',
  inject: ['fs', 'bash', 'process'], // ctx.process 后台进程服务（globalBG，跨轮次存活）
  apply(ctx) {
    for (const t of tools) {
      const toolDef = {
        name: t.name,
        description: t.description,
        usageGuide: t.usageGuide,
        category: t.category,
        readOnly: t.readOnly,
        requiresApproval: t.requiresApproval,
        parameters: t.parameters,
      }
      if (t.impl) {
        // JS 原生化：调用实现在插件内（ctx.fs/ctx.bash/ctx.process）
        toolDef.execute = (args) => t.impl(ctx, args || {})
      } else {
        // run_code：宿主 Go 执行器（嵌套 goja VM 调度）
        toolDef.execute = (args) => ctx.hostTool.exec(t.name, args || {})
      }
      ctx.tools.register(toolDef)
    }
  },
}
