// ═══════════════════════════════════════════════════════════════
// tool-memory — 跨会话记忆（memory_write/read/list/search/delete）
//
// 迁移（2026-08-22 Round2）：binary 形态 → JS 原生（对齐 tool-core 模式）。
// 原 execute 调 ctx.binary.exec 复用插件目录 bin/ 下独立二进制（已归档
// bin/legacy-plugin-bins/），现实现完全在插件内（ctx.fs 读写 .pair/memory/），
// 不再依赖 ctx.binary。行为复刻 internal/agent/memory.go（frontmatter 格式、
// MEMORY.md 索引、碎片化提醒、多项目 project 路由）。
// 工具清单：memory_write、memory_delete、memory_read、memory_list、memory_search
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "memory_write",
    "description": "写入或【更新】一条持久记忆（跨会话保留在 .pair/memory/）。**先 memory_search/list 查有无相关记忆——有则用其同名覆盖来更新（先 memory_read 读旧的、融合后写回），别为同一主题反复新建、造成碎片化**。name 唯一标识；type: user(用户偏好)/feedback(纠正与确认的做法)/project(项目决策约束)/reference(外部资源指针)；description 一句话摘要；content 正文。",
    "usageGuide": "写入或更新一条持久记忆（跨会话保留）。先 memory_search 查有无相关记忆，有则读旧→融合→同名更新，别反复新建造成碎片化。用于记录用户偏好、项目决策、修复方案等。需审核批准。多项目工作区可用 project 参数指定目标项目。",
    "parameters": {
      "properties": {
        "content": {
          "description": "正文",
          "type": "string"
        },
        "description": {
          "description": "一句话摘要",
          "type": "string"
        },
        "name": {
          "description": "唯一名，用【简短中文】命名（如 数据库连接池配置）；更新已有记忆请用其原名",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        },
        "type": {
          "description": "user/feedback/project/reference",
          "type": "string"
        }
      },
      "required": [
        "name",
        "description",
        "content"
      ],
      "type": "object"
    },
    "requiresApproval": true
  },
  {
    "name": "memory_delete",
    "description": "删除一条过时/错误的记忆（按 name）。保持记忆库精简准确，别让过时信息长期误导。",
    "usageGuide": "删除一条过时/错误的记忆。保持记忆库精简。需审核批准。删除前建议先 memory_read 确认是该条。",
    "parameters": {
      "properties": {
        "name": {
          "description": "记忆名",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
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
    "name": "memory_read",
    "description": "按 name 读取一条记忆的全文。",
    "usageGuide": "按 name 读一条记忆全文。渐进式披露：先 memory_list 看总览，再用此工具读具体细则。比直接读 .pair/memory/ 文件更方便（自动解析 YAML front-matter）。",
    "parameters": {
      "properties": {
        "name": {
          "description": "记忆名",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
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
    "name": "memory_list",
    "description": "列出所有记忆的【总览】（名 + 摘要，渐进式披露的总览层）；要某条细则用 memory_read 读全文。",
    "usageGuide": "列出所有记忆的总览（名+摘要）。先调此工具看有什么记忆，再决定用 memory_read 读哪条。比 bash dir .pair/memory 更友好（渐进式披露+自动维护索引）。",
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
    "name": "memory_search",
    "description": "按关键词搜索记忆（匹配名/摘要/正文），返回命中条目的名+摘要。",
    "usageGuide": "按关键词搜索记忆（匹配名/摘要/正文）。要查某个主题是否已有记忆时优先用此工具，比 memory_list 遍历更高效。",
    "parameters": {
      "properties": {
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        },
        "query": {
          "description": "关键词",
          "type": "string"
        }
      },
      "required": [
        "query"
      ],
      "type": "object"
    },
    "readOnly": true
  }
];


// ─── JS 原生化实现（ctx.fs） ────────────────────

// memDirFromArgs 解析 project 参数 → 目标项目记忆目录（缺省 = 主项目）。
// 多项目工作区中 memory 按项目隔离（.pair/memory/ 在各项目根下）。
function memDir(ctx, args) {
  const project = args && args.project
  if (project && !/^[a-zA-Z]:[\/]/.test(project) && !project.startsWith('/')) {
    return '../' + String(project).replace(/[\/]+$/, '') + '/.pair/memory'
  }
  return '.pair/memory'
}

// safeMemName 把名字里的路径危险字符(/ \ : . 空格)换成 -，防路径穿越；保留 CJK 等其它字符。
function safeMemName(s) {
  return String(s == null ? '' : s).replace(/[\/\\:. ]/g, '-').trim()
}

// frontmatterField 取 frontmatter 中 key: 字段值。
function frontmatterField(text, key) {
  for (const ln of String(text).split('\n')) {
    if (ln.startsWith(key + ':')) return ln.slice(key.length + 1).trim()
  }
  return ''
}

// isMemFile 是否为一条记忆文件（.md 且非索引 MEMORY.md 本身）。
function isMemFile(name) {
  return name.endsWith('.md') && name !== 'MEMORY.md'
}

const memIndexHeader =
  '# 记忆索引（总览）\n\n' +
  '> 类型：user 用户偏好 / feedback 纠正与确认 / project 项目决策 / reference 外部资源\n' +
  '> 渐进式披露：先看本总览，需要细则再读对应条目文件（memory_read）。\n\n'

function memIndexLine(name, desc) {
  return desc ? '- [' + name + '](' + name + '.md) — ' + desc : '- [' + name + '](' + name + '.md)'
}

// listMemFiles 列出目录下全部记忆文件（名 + 全文），目录不存在返回 []。
function listMemFiles(ctx, dir) {
  let names = []
  try {
    names = ctx.fs.readdir(dir)
  } catch (e) {
    return []
  }
  const out = []
  for (const n of names) {
    if (!isMemFile(n)) continue
    let st = null
    try { st = ctx.fs.stat(dir + '/' + n) } catch (e) { /* 忽略 */ }
    if (st && st.isDir) continue
    let text = ''
    try { text = ctx.fs.readFile(dir + '/' + n) } catch (e) { continue }
    out.push({ name: n.slice(0, -3), text })
  }
  return out
}

// genMemIndex 扫描目录现生成「记忆总览」内容（只读，不写盘）；无记忆→""。
function genMemIndex(ctx, dir) {
  const files = listMemFiles(ctx, dir)
  if (files.length === 0) return ''
  const lines = files.map(f => memIndexLine(f.name, frontmatterField(f.text, 'description')))
  return memIndexHeader + lines.join('\n') + '\n'
}

// rebuildIndex 重建并写盘 MEMORY.md（记忆写入/删除后调，保持总览文件实时）。
function rebuildIndex(ctx, dir) {
  let c = genMemIndex(ctx, dir)
  if (c === '') c = memIndexHeader
  ctx.fs.writeFile(dir + '/MEMORY.md', c)
}

// taskTokens 从文本提取检索 token：ASCII 词（长度>1，去停用词）+ CJK 二元组。
const memStopWords = new Set(['the', 'and', 'for', 'with', 'you', 'this', 'that', '请', '帮我', '一下', '这个', '那个', '怎么', '如何'])
function taskTokens(s) {
  const toks = new Set()
  const lower = String(s).toLowerCase()
  let m
  const asciiRe = /[a-z0-9]+/g
  while ((m = asciiRe.exec(lower)) !== null) {
    if (m[0].length > 1 && !memStopWords.has(m[0])) toks.add(m[0])
  }
  const cjkRe = /[\u4e00-\u9fff]+/g
  while ((m = cjkRe.exec(lower)) !== null) {
    const run = m[0]
    if (run.length === 1) toks.add(run)
    else for (let i = 0; i + 1 < run.length; i++) toks.add(run.slice(i, i + 2))
  }
  return toks
}

// similarMemory 找与给定文本最相关的已有记忆名（token 重叠 ≥3，排除 exclude）。
function similarMemory(ctx, dir, exclude, text) {
  const files = listMemFiles(ctx, dir)
  const toks = taskTokens(text)
  if (toks.size < 3) return ''
  let best = '', bestScore = 2
  for (const f of files) {
    if (f.name === exclude) continue
    const lower = f.text.toLowerCase()
    let score = 0
    for (const tok of toks) {
      if (lower.includes(tok)) score++
    }
    if (score > bestScore) { best = f.name; bestScore = score }
  }
  return best
}

// listMemories 列出 dir 下记忆（filter 非空则按关键词过滤名/摘要/正文）。
function listMemories(ctx, dir, filter) {
  const files = listMemFiles(ctx, dir)
  if (files.length === 0) return '（暂无记忆）'
  filter = String(filter || '').toLowerCase()
  const lines = []
  for (const f of files) {
    if (filter && !f.text.toLowerCase().includes(filter)) continue
    lines.push('- ' + f.name + '：' + frontmatterField(f.text, 'description'))
  }
  if (lines.length === 0) return '（无匹配记忆）'
  return lines.join('\n')
}

// memory_write：写入/更新（frontmatter + 正文），同步维护 MEMORY.md 索引；
// 新建且已有相关记忆时提醒优先更新（防碎片化）。
function memoryWrite(ctx, args) {
  const dir = memDir(ctx, args)
  const name = safeMemName(args.name)
  if (!name) throw new Error('name 不能为空')
  const typ = args.type || 'project'
  if (!ctx.fs.exists(dir)) ctx.fs.mkdir(dir, true)
  const path = dir + '/' + name + '.md'
  const updating = ctx.fs.exists(path)
  const desc = args.description == null ? '' : String(args.description)
  const content = args.content == null ? '' : String(args.content)
  const body = '---\nname: ' + name + '\ntype: ' + typ + '\ndescription: ' + desc + '\n---\n\n' + content + '\n'
  ctx.fs.writeFile(path, body)
  rebuildIndex(ctx, dir)
  if (updating) return '已更新记忆：' + name
  const sim = similarMemory(ctx, dir, name, name + ' ' + desc + ' ' + content)
  if (sim) {
    return '已新建记忆：' + name + '\n⚠ 已有相关记忆「' + sim +
      '」——若属同一主题，建议改用 memory_read 读它、融合后用「' + sim + '」更新，而非新建，避免记忆碎片化。'
  }
  return '已记忆：' + name
}

function memoryDelete(ctx, args) {
  const dir = memDir(ctx, args)
  const name = safeMemName(args.name)
  const path = dir + '/' + name + '.md'
  if (!ctx.fs.exists(path)) throw new Error('无此记忆: ' + name)
  ctx.fs.rm(path, false)
  rebuildIndex(ctx, dir)
  return '已删除记忆：' + name
}

function memoryRead(ctx, args) {
  const dir = memDir(ctx, args)
  const name = safeMemName(args.name)
  const path = dir + '/' + name + '.md'
  if (!ctx.fs.exists(path)) throw new Error('无此记忆: ' + name)
  return ctx.fs.readFile(path)
}

function memoryList(ctx, args) {
  const dir = memDir(ctx, args)
  const c = genMemIndex(ctx, dir)
  return c !== '' ? c : '（暂无记忆）'
}

function memorySearch(ctx, args) {
  const q = String(args.query == null ? '' : args.query).trim()
  if (q === '') throw new Error('query 不能为空')
  return listMemories(ctx, memDir(ctx, args), q)
}

const impls = {
  memory_write: memoryWrite,
  memory_delete: memoryDelete,
  memory_read: memoryRead,
  memory_list: memoryList,
  memory_search: memorySearch,
}


return {
  name: 'tool-memory',
  inject: ['fs'],
  purpose: '跨会话记忆（memory_write/read/list/search/delete）——迁移自内置 Go 工具组；调用实现（JS 编排 ctx.fs）完全在插件内（Round2 JS 原生化）',
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
