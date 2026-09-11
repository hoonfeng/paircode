// ═══════════════════════════════════════════════════════════════
// tool-harness — harness 核心协议工具（read/write/apply_patch/glob/grep/run_code + list/restore_snapshot）
// 迁移来源（2026-08-16）：内置 RegisterHarnessTools（internal/agent/
// harness_tools.go）→ 磁盘外置插件。2026-08-16 第二轮：7 个工具的 execute
// 由 ctx.hostTool（宿主 Go 执行器）改为 **JS 原生化**（调用实现在插件内，
// 底层能力 ctx.fs/ctx.bash，对齐 tool-core 模式）；run_code 保持 hostTool
// ——其「node + tools.xxx 嵌套调度」需 goja VM 宿主运行时（runCodeNested），
// 属框架运行时能力，JS 沙箱不可复刻。
//
// ★ 2026-09 工具重构（对齐 codex unified exec）：后台进程 4 件套
//   （run_background/read_output/kill_process/job_list）移交 tool-exec
//   （exec_command/write_stdin/kill_process 会话式模型）；本插件聚焦
//   文件与工作区工具（read/write/apply_patch/glob/grep/run_code）。
// 装配：.pair/plugins/ 启动扫描（LoadGlobalPlugins）→ define + load。
// 停用本插件（cordis(op=stop) tool-harness）即回收全部 6 个工具。
// ═══════════════════════════════════════════════════════════════

// ─── JS 原生化实现（ctx.fs / ctx.bash） ────────────────────

// 是否绝对路径（Windows 盘符 / Unix 根 / UNC 反斜杠开头）。
function isAbsPath(p) {
  const s = String(p == null ? '' : p)
  return /^[a-zA-Z]:[\\/]/.test(s) || s.startsWith('/') || s.startsWith('\\')
}

// 项目路由：project 非空 → 目标项目（工作区另一根）下的路径。
//   · path 为绝对路径 → 原样返回（越界由宿主 resolve 拦截）
//   · project 为绝对路径 → 直接作为前缀拼接 path
//   · 其余 → 以 ../<project>/ 前缀拼相对路径（多根归属由宿主 resolve 检查，
//     与直接传绝对路径等价）
//   path 为空时原样返回（保持「省略 path」语义：搜索类工具用 project 参数走
//   宿主 projRootFromArgs 路由，见 globFiles/grepFiles）。
function projPath(ctx, args, path) {
  const project = args.project
  if (!project) return path
  if (isAbsPath(path)) return path
  const rel = String(path == null ? '' : path).replace(/^[\\/]+/, '')
  if (!rel) return path
  const proj = String(project).replace(/[\\/]+$/, '')
  if (isAbsPath(proj)) return proj + '/' + rel
  return '../' + proj.replace(/^[\\/]+/, '') + '/' + rel
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

// apply_patch：codex 语法自由格式补丁（2026-09 工具重构 Phase B——取代 edit/multi_edit）。
// 宿主能力 ctx.fs.applyPatch（Go ApplyPatchText：解析 + 应用 + 写前快照 + 变更回调）：
// 免 JSON 转义、免 old_string 唯一性焦虑、免行号依赖（上下文行定位）；支持
// Add/Delete/Update/Move 四类操作及多文件单次调用。
function applyPatch(ctx, args) {
  const patch = String(args.patch == null ? '' : args.patch)
  if (!patch.trim()) throw new Error('patch 不能为空（codex 语法：*** Begin Patch … *** End Patch）')
  return ctx.fs.applyPatch(patch)
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
  // ★ project 透传宿主：searchFilesHandler 经 projRootFromArgs 路由到目标项目根
  //   （此前被丢弃 → 传 project 无效，同工作区跨项目搜索只能靠绝对路径）。
  if (args.project) opts.project = args.project
  return ctx.fs.glob(args.pattern, opts)
}

function grepFiles(ctx, args) {
  const opts = {}
  if (args.path) opts.path = args.path
  if (args.glob) opts.glob = args.glob
  if (args.case_insensitive) opts.case_insensitive = true
  if (args.max_results) opts.max_results = args.max_results
  // ★ project 透传宿主（同上，跨项目搜索路由）。
  if (args.project) opts.project = args.project
  return ctx.fs.grep(args.pattern, opts)
}

const tools = [
  {
    name: 'read',
    description: '读取文件内容（对齐 read）。file_path 为工作区内路径，相对「主项目根」解析——多项目工作区读其他项目文件请传 project（项目目录名）或绝对路径（兼容旧参数名 path）。可选 offset(起始行,1 基，默认 1)+limit(行数，默认 2000)读片段；输出带行号（"行号: 内容"）与文件总行数 footer。',
    usageGuide: 'harness 标准读工具：读取文件内容（行号输出对齐约定）。路径相对主项目根（跨项目传 project 或绝对路径，勿反复重试相对路径）；越界自动拦截，二进制自动拒绝（改用 inspect_binary）。大文件用 offset+limit 分页。',
    category: '文件',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        file_path: { type: 'string', description: '文件路径（工作区内；相对主项目根，跨项目用 project 或绝对路径；参考参数名，与 path 等价）' },
        path: { type: 'string', description: '文件路径（工作区内；相对主项目根，跨项目用 project 或绝对路径；旧参数名，file_path 优先）' },
        offset: { type: 'integer', description: '可选：起始行号(1 基，默认 1)' },
        limit: { type: 'integer', description: '可选：读取行数（默认 2000）' },
        project: { type: 'string', description: '可选：目标项目（工作区项目目录名如 ref/wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目；不传时路径相对主项目根解析。多项目工作区访问其他项目必须传本参数或绝对路径。' },
      },
    },
    impl: readFile,
  },
  {
    name: 'write',
    description: '把 content 完整写入 file_path（覆盖；父目录自动创建）。路径相对「主项目根」解析——多项目工作区写其他项目文件请传 project（项目目录名）或绝对路径。需审核批准。',
    usageGuide: 'harness 标准写工具：整文件写入（覆盖）。路径相对主项目根（跨项目传 project 或绝对路径）。写类操作需人工确认。如需追加请先 read 再 write 覆盖。',
    category: '文件',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        file_path: { type: 'string', description: '文件路径（相对主项目根，跨项目用 project 或绝对路径；参考参数名，与 path 等价）' },
        path: { type: 'string', description: '文件路径（相对主项目根，跨项目用 project 或绝对路径；旧参数名，file_path 优先）' },
        content: { type: 'string', description: '完整文件内容' },
        project: { type: 'string', description: '可选：目标项目（目录名/相对主项目的路径/绝对路径）。省略 = 主项目；写入其他项目必须传本参数或绝对路径。' },
      },
      required: ['content'],
    },
    impl: writeFile,
  },
  {
    name: 'apply_patch',
    description: '应用 codex 语法自由格式补丁修改文件（*** Begin Patch / Add File / Update File / Delete File / *** End Patch）。一次调用可含多个文件、四类操作（新增/更新/删除/移动）；Update 用上下文行定位（免行号、免 JSON 转义）——@@ 段内：空格前缀=上下文行、- 前缀=删除行、+ 前缀=新增行。',
    usageGuide: '编辑文件的主工具（取代 edit/multi_edit）。小改动用 Update File+上下文行；新增文件用 Add File（每行 + 前缀，内容须一次写全）；删除文件用 Delete File；移动用 Update File + *** Move to: 行。修改前先 read 确认上下文行逐字一致，否则该 hunk 匹配失败。',
    category: '文件',
    requiresApproval: true,
    parameters: {
      type: 'object',
      properties: {
        patch: { type: 'string', description: '补丁文本（*** Begin Patch … *** End Patch）' },
      },
      required: ['patch'],
    },
    impl: applyPatch,
  },
  {
    name: 'glob',
    description: '按通配符递归查找文件，返回相对路径列表（对齐 glob）。pattern 含 / 或 ** 时按路径模式（如 internal/**/*.go），否则匹配任意深度文件名（如 *.go）；path 限定子目录（相对「主项目根」解析——多项目工作区搜其他项目请传 project 或绝对路径）。',
    usageGuide: 'harness 标准 glob 工具：按路径模式发现文件。path 相对主项目根，跨项目传 project（目录名）或绝对路径。自动跳过依赖库（node_modules/vendor/.venv…）、构建产物（dist/build/out/target…）、VCS（.git…）与项目根下的 IDE 运行数据目录（_temp/logs/bin/release/screenshots…）；跳过只作用于递归下降，把 path 指进该目录仍可搜。比 shell find 更精确（结构化、防撑爆）。',
    category: '代码搜索',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        pattern: { type: 'string', description: '文件名/路径通配符，如 *.go' },
        path: { type: 'string', description: '可选：限定子目录（省略=主项目根；相对主项目根，跨项目用 project 或绝对路径）' },
        language: { type: 'string', description: '可选：按语言过滤，如 "go"、"typescript"' },
        project: { type: 'string', description: '可选：目标项目（目录名如 ref，或相对主项目的路径/绝对路径）。省略 = 主项目（path 相对主项目根解析）。★ 跨项目搜索必须传本参数，否则搜不到其他项目的文件。' },
      },
      required: ['pattern'],
    },
    impl: globFiles,
  },
  {
    name: 'grep',
    description: '在工作区内按正则搜索文件内容，返回「相对路径:行号: 行文本」（对齐 grep）。pattern 为 RE2 正则；path 限定子目录（相对「主项目根」解析——多项目工作区搜其他项目请传 project 或绝对路径）；glob 按文件名过滤；case_insensitive 忽略大小写。',
    usageGuide: 'harness 标准 grep 工具：正则全文搜索。path 相对主项目根，跨项目传 project（目录名）或绝对路径（不传 project 时只搜主项目）。自动跳过依赖库/构建产物/VCS/项目根下的 IDE 运行数据目录（_temp/logs/bin/release/screenshots…）与二进制/超大文件；跳过只作用于递归下降（把 path 指进该目录仍可搜）。搜索函数/类型定义请优先用 codegraph_search（AST 级更精确）。',
    category: '代码搜索',
    readOnly: true,
    parameters: {
      type: 'object',
      properties: {
        pattern: { type: 'string', description: 'RE2 正则表达式' },
        path: { type: 'string', description: '可选：限定子目录（省略=主项目根；相对主项目根，跨项目用 project 或绝对路径）' },
        glob: { type: 'string', description: '可选：文件名通配过滤，如 *.go' },
        case_insensitive: { type: 'boolean', description: '可选：忽略大小写' },
        max_results: { type: 'integer', description: '可选：结果行数上限（默认 200）' },
        project: { type: 'string', description: '可选：目标项目（目录名如 ref，或相对主项目的路径/绝对路径）。省略 = 主项目（path 相对主项目根解析）。★ 跨项目搜索必须传本参数，否则搜不到其他项目的文件。' },
      },
      required: ['pattern'],
    },
    impl: grepFiles,
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
  // ─── 会话快照（2026-09-12 自 tool-snapshot 并入）：写前快照的查询/恢复 ───
  {
    "name": "restore_snapshot",
    "description": "从快照恢复指定文件。快照在 write/apply_patch 修改前自动创建。默认恢复到最旧快照（原始文件）。可用 list_snapshots 查看快照列表。指定 index 参数恢复特定版本（0=最旧原始文件，-1=最新，1~N=第 N 份从最旧算）。",
    "parameters": {
      "properties": {
        "index": {
          "description": "可选快照索引：0=最旧(原始/默认)，-1=最新，1~N=第 N 份",
          "type": "string"
        },
        "path": {
          "description": "要恢复的文件路径（相对主项目根，如 \"cmd/main.go\"；跨项目请传绝对路径）",
          "type": "string"
        }
      },
      "required": [
        "path"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "list_snapshots",
    "description": "列出指定文件的所有可用快照（按时间倒序，带索引号）。用 restore_snapshot 的 index 参数可恢复指定版本。",
    "parameters": {
      "properties": {
        "path": {
          "description": "文件路径（相对主项目根；跨项目请传绝对路径）",
          "type": "string"
        }
      },
      "required": [
        "path"
      ],
      "type": "object"
    },
    "readOnly": true
  },
]

return {
  name: 'tool-harness',
  purpose: 'harness 核心协议工具（read/write/apply_patch/glob/grep/run_code + 快照 list/restore_snapshot）——文件与工作区工具；执行工具由 tool-exec 承载（2026-09 工具重构）；2026-09-12 并入 tool-snapshot',
  inject: ['fs'],
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
