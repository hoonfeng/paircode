// ═══════════════════════════════════════════════════════════════
// tool-harness — harness 核心协议工具（read/write/edit/glob/grep/
// bash/str_replace_editor/run_code）
//
// 迁移来源（2026-08-16）：内置 RegisterHarnessTools（internal/agent/
// harness_tools.go）→ 磁盘外置插件。2026-08-16 第二轮：7 个工具的 execute
// 由 ctx.hostTool（宿主 Go 执行器）改为 **JS 原生化**（调用实现在插件内，
// 底层能力 ctx.fs/ctx.bash，对齐 tool-core 模式）；run_code 保持 hostTool
// ——其「node + tools.xxx 嵌套调度」需 goja VM 宿主运行时（runCodeNested），
// 属框架运行时能力，JS 沙箱不可复刻。
//
// 装配：.pair/plugins/ 启动扫描（LoadGlobalPlugins）→ define + load。
// 停用本插件（cordis_stop tool-harness）即回收全部 8 个工具。
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

// filePath 兼容参数名：DSH 参考用 file_path，repo 旧调用方用 path——两者都接受（file_path 优先）。
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

// read：DSH 对齐（2026-09 Round2 R2-7）——file_path 参数 + 行号输出块：
//   <path>/<type>/<content> 内每行 "number: text"，末尾 footer
//   （(End of file - total N lines) 或 (Showing lines X-Y of N. Use offset=… to continue.)）。
// offset(默认 1)/limit(默认 2000) 分页；兼容旧参数名 path。
function readFile(ctx, args) {
  const path = projPath(ctx, args, filePath(args))
  const text = readFileText(ctx, args, path)
  const lines = text.split('\n')
  // 去掉末尾空行（split 尾随 \n 产生）——与 DSH 行计数一致
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
// replace_all=true 时替换全部出现处（DSH 对齐，2026-09 Round2 R2-7），
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

// str_replace_editor：view/create/str_replace/insert
function sreView(ctx, args, path) {
  const full = projPath(ctx, args, path)
  if (ctx.fs.stat(full).isDirectory) {
    // 目录：列出非隐藏项（最多 2 层）
    const out = []
    const walk = (dir, depth) => {
      for (const name of ctx.fs.readdir(dir)) {
        if (name.startsWith('.')) continue
        const child = dir.endsWith('/') ? dir + name : dir + '/' + name
        const st = ctx.fs.stat(child)
        out.push('  '.repeat(depth) + name + (st.isDirectory ? '/' : ''))
        if (st.isDirectory && depth < 1) walk(child, depth + 1)
      }
    }
    walk(full, 0)
    return out.join('\n') || '（空目录）'
  }
  const text = readFileText(ctx, args, full)
  const lines = text.split('\n')
  const vr = args.view_range
  let start = 0, end = lines.length
  if (Array.isArray(vr) && vr.length >= 1) {
    const a = Number(vr[0])
    if (a === -1) { start = 0; end = lines.length }
    else if (a >= 1) {
      start = a - 1
      end = vr.length >= 2 && Number(vr[1]) >= a ? Number(vr[1]) : lines.length
    }
  }
  const shown = lines.slice(start, end)
  return shown.map((l, i) => String(start + i + 1).padStart(5) + '\t' + l).join('\n')
}

function sreCreate(ctx, args, path) {
  const full = projPath(ctx, args, path)
  if (ctx.fs.exists(full)) throw new Error('文件已存在，create 拒绝覆盖: ' + path)
  writeFile(ctx, { path: full, content: args.file_text || '' })
  return '已创建 ' + path + '（' + (args.file_text || '').length + ' 字节）'
}

function sreReplace(ctx, args, path) {
  const full = projPath(ctx, args, path)
  const text = readFileText(ctx, args, full)
  const oldStr = args.old_str
  if (oldStr == null) throw new Error('str_replace 需要 old_str')
  const newStr = args.new_str == null ? '' : String(args.new_str)
  let idx = text.indexOf(oldStr)
  if (idx < 0) throw new Error('old_str 未找到（须精确匹配，含空白）')
  if (text.indexOf(oldStr, idx + Math.max(oldStr.length, 1)) >= 0) throw new Error('old_str 不唯一，拒绝替换')
  ctx.fs.writeFile(full, text.slice(0, idx) + newStr + text.slice(idx + oldStr.length))
  return '已替换 ' + path
}

function sreInsert(ctx, args, path) {
  const full = projPath(ctx, args, path)
  const text = readFileText(ctx, args, full)
  const lines = text.split('\n')
  const ln = Number(args.insert_line)
  if (!(ln >= 0)) throw new Error('insert 需要 insert_line（1 基）')
  if (ln > lines.length) throw new Error('insert_line ' + ln + ' 超出文件行数 ' + lines.length)
  const newStr = args.new_str == null ? '' : String(args.new_str)
  lines.splice(ln, 0, newStr)
  ctx.fs.writeFile(full, lines.join('\n'))
  return '已在第 ' + ln + ' 行后插入 ' + path
}

function strReplaceEditor(ctx, args) {
  const cmd = args.command
  switch (cmd) {
    case 'view': return sreView(ctx, args, args.path)
    case 'create': return sreCreate(ctx, args, args.path)
    case 'str_replace': return sreReplace(ctx, args, args.path)
    case 'insert': return sreInsert(ctx, args, args.path)
    default: throw new Error('未知 command: ' + cmd + '（view/create/str_replace/insert）')
  }
}

// bash：ctx.bash（120s 超时 + 输出截断由宿主保证）。
// ★ 2026-09 Round2 R2-7 DSH 对齐：description 可选（参考必填，repo 兼容旧调用方
//   保持可选）+ timeoutMs 可选（>0 覆盖默认 120s，传给 ctx.bash.exec 第三参秒数）。
function runCommand(ctx, args) {
  const cwd = args.project ? '../' + String(args.project).replace(/[\\/]+$/, '') + (args.cwd ? '/' + args.cwd : '') : (args.cwd || '')
  let timeoutSec = 0
  if (args.timeoutMs != null) {
    const ms = Math.round(Number(args.timeoutMs))
    if (Number.isFinite(ms) && ms > 0) timeoutSec = Math.max(1, Math.round(ms / 1000))
  }
  const res = timeoutSec > 0 ? ctx.bash.exec(String(args.command || ''), cwd, timeoutSec) : ctx.bash.exec(String(args.command || ''), cwd)
  if (res.error) return res.output + (res.output ? '\n' : '') + '[stderr] ' + res.error
  return res.output
}

// run_code：统一二进制承载（tool-binary 注册了 run_code——node+tools.xxx
// 嵌套 goja 调度 + 外部进程执行，二进制进程内自持 goja 运行时）
function runCode(ctx, args) {
  const opts = { timeout: 120000 }
  return ctx.binary.exec('run_code', args || {}, opts).text
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
    description: '读取文件内容（对齐 deepseek-harness read）。file_path 为工作区内路径（兼容旧参数名 path）；可选 offset(起始行,1 基，默认 1)+limit(行数，默认 2000)读片段；输出带行号（"行号: 内容"）与文件总行数 footer。',
    usageGuide: 'harness 标准读工具：读取文件内容（行号输出对齐参考实现）。路径越界自动拦截，二进制自动拒绝（改用 inspect_binary）。大文件用 offset+limit 分页。',
    category: '文件',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        file_path: { type: 'string', description: '文件路径（工作区内；DSH 参考参数名，与 path 等价）' },
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
        file_path: { type: 'string', description: '文件路径（DSH 参考参数名，与 path 等价）' },
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
    description: '把文件中唯一一处 old_string 替换为 new_string（对齐 deepseek-harness edit）；replace_all=true 时替换全部出现处。内置智能匹配（CRLF 归一化）；匹配失败优先用 line_start/line_end 行号定位。',
    usageGuide: 'harness 标准编辑工具：小改动（≤5 行）用精确替换（须唯一；多处出现可设 replace_all=true 全部替换）；大改动请用 write 写整段。替换前会自动快照。',
    category: '文件',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        file_path: { type: 'string', description: '文件路径（DSH 参考参数名，与 path 等价）' },
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
    description: '按通配符递归查找文件，返回相对路径列表（对齐 deepseek-harness glob）。pattern 含 / 或 ** 时按路径模式（如 internal/**/*.go），否则匹配任意深度文件名（如 *.go）；path 限定子目录。',
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
    description: '在工作区内按正则搜索文件内容，返回「相对路径:行号: 行文本」（对齐 deepseek-harness grep）。pattern 为 RE2 正则；path 限定子目录；glob 按文件名过滤；case_insensitive 忽略大小写。',
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
    name: 'bash',
    description: '同步执行一条 shell 命令并返回输出（对齐 deepseek-harness bash）。每次调用在独立 shell 中运行（无状态持久）。禁止用于长期进程（dev server/watch/tcp 监听）——请用 run_background。description 可选（参考实现必填，用于 UI 展示命令意图）；timeoutMs 可选（覆盖默认 120s）。',
    usageGuide: 'harness 标准 bash 工具：执行命令（构建/测试/查询等短命令）。默认 120s 超时自动终止，可用 timeoutMs 调整。长期进程用 run_background/read_output/kill_process。',
    category: '执行',
    parameters: {
      type: 'object',
      properties: {
        command: { type: 'string', description: '要执行的命令' },
        description: { type: 'string', description: '可选：命令意图说明（5-10 词，DSH 参考为必填；repo 兼容旧调用方保持可选）' },
        timeoutMs: { type: 'integer', description: '可选：超时毫秒数（>0 覆盖默认 120000；0=不超时）' },
        cwd: { type: 'string', description: '可选工作目录（工作区内，省略=根）' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['command'],
    },
    impl: runCommand,
  },
  {
    name: 'str_replace_editor',
    description: 'Custom editing tool for viewing, creating and editing files（对齐 deepseek-harness str_replace_editor）\n' +
      '* `command` 必填：view / create / str_replace / insert\n' +
      '* `view` 显示文件内容（带行号）；path 为目录时列出非隐藏文件/目录最多 2 层\n' +
      '* `create` 创建新文件（path 已存在则报错）；内容在 `file_text`\n' +
      '* `str_replace` 把 `old_str` 替换为 `new_str`——old_str 必须精确匹配且唯一（含空白！不唯一则拒绝）\n' +
      '* `insert` 在 `insert_line` 之后插入 `new_str`\n' +
      '* `view` 支持 `view_range` 数组限定行范围（如 [11,12]，[-1] 到文件尾）\n' +
      '* 长输出会被截断并标记 `<response clipped>`',
    usageGuide: 'harness 标准命令式编辑器（Claude 系工具）：view 查看、create 创建、str_replace 精确替换（唯一匹配）、insert 行后插入。与 edit 相比更适合『需要先查看行号、再精确替换』的流程；带行号输出方便后续定位。',
    category: '文件',
    parameters: {
      type: 'object',
      properties: {
        command: { type: 'string', description: '要执行的命令：view / create / str_replace / insert（必填）' },
        path: { type: 'string', description: '文件或目录路径（工作区内；view 支持目录，其他命令须为文件）' },
        file_text: { type: 'string', description: 'create 命令的文件内容' },
        insert_line: { type: 'integer', description: 'insert 命令：在此行之后插入 new_str（1 基）' },
        new_str: { type: 'string', description: 'str_replace 的替换新内容 / insert 的插入内容' },
        old_str: { type: 'string', description: 'str_replace 的原文（须精确且唯一匹配）' },
        view_range: { type: 'array', items: { type: 'integer' }, description: 'view 命令行范围，如 [11,12]；[-1] 表示到文件尾' },
        project: { type: 'string', description: '可选：目标项目。省略 = 主项目。' },
      },
      required: ['command', 'path'],
    },
    impl: strReplaceEditor,
  },
  {
    name: 'run_code',
    description: '执行一段代码并返回输出（对齐 deepseek-harness run_code）。language: auto（默认，按内容探测）/ go / python / node。',
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
  purpose: 'harness 核心协议工具（read/write/edit/glob/grep/bash/str_replace_editor/run_code）——迁移自内置 RegisterHarnessTools',
  inject: ['fs', 'bash'],
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
        // JS 原生化：调用实现在插件内（ctx.fs/ctx.bash）
        toolDef.execute = (args) => t.impl(ctx, args || {})
      } else {
        // run_code：宿主 Go 执行器（嵌套 goja VM 调度）
        toolDef.execute = (args) => ctx.hostTool.exec(t.name, args || {})
      }
      ctx.tools.register(toolDef)
    }
  },
}
