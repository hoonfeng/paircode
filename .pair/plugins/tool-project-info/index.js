// ═══════════════════════════════════════════════════════════════
// tool-project-info — 项目知识库（单工具 project_info(op=write/read/list/tree/search/delete/explore)；2026-09 由 7 工具合并）
//
// 迁移（2026-08-22 Round2）：binary 形态 → JS 原生（对齐 tool-core 模式）。
// 原 execute 调 ctx.binary.exec 复用插件目录 bin/ 下独立二进制（已归档
// bin/legacy-plugin-bins/），现实现完全在插件内（ctx.fs 读写 .pair/project-info/），
// 不再依赖 ctx.binary。行为复刻 internal/agent/projectinfo.go（树形路径分级、
// notes/ 前缀镜像、渐进式披露、多项目 project 路由）。
// 工具清单：project_info（op=write/read/list/tree/search/delete/explore）
// ═══════════════════════════════════════════════════════════════
// ★ 2026-08-29（候选 A 创造需求）：改用 mini Node API——require('path')
//   统一路径拼接（手工 d+'/'+name 在 Windows 分隔符下有隐患）。
const path = require('path')
const tools = [
  {
    "name": "project_info",
    "description": "项目知识库（.pair/project-info/ 树形文档）统一入口。op=write 写入/更新条目（path+content）；op=read 读某篇全文（path）；op=list 列出全部条目的总览；op=tree 返回完整树形结构；op=search 按关键词搜索（query）；op=delete 删除某篇（path）；op=explore 返回项目目录结构概览（构建知识库的起点）。",
    "usageGuide": "项目知识库操作（渐进式披露）：先 list/tree 看有哪些文档，search 查关键词，read 读全文；explore 扫描项目结构起步；write 记录结构化理解（顶层分支：目标/架构/实现/关键点/设计思想，根为 概览；path 带分支前缀如 架构/模块-agent，兼容 notes/ 前缀镜像 .agents/notes/）；delete 清理过时条目。比记在脑子里可靠（持久化+跨会话可见）。多项目工作区用 project 指定项目。",
    "parameters": {
      "properties": {
        "content": {
          "description": "write 用：Markdown 正文（首行用 # 标题）",
          "type": "string"
        },
        "op": {
          "description": "操作：write 写入/read 读取/list 总览/tree 树形/search 搜索/delete 删除/explore 项目结构扫描",
          "enum": [
            "write",
            "read",
            "list",
            "tree",
            "search",
            "delete",
            "explore"
          ],
          "type": "string"
        },
        "path": {
          "description": "write/read/delete 用：条目路径（中文，带顶层分支前缀：目标/架构/实现/关键点/设计思想，如 架构/模块-agent），不含 .md；用 / 嵌套为细节篇",
          "type": "string"
        },
        "project": {
          "description": "可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。",
          "type": "string"
        },
        "query": {
          "description": "search 用：关键词",
          "type": "string"
        }
      },
      "required": [
        "op"
      ],
      "type": "object"
    }
  },
  // （project_info_read / project_info_list 已并入单工具 project_info——op=read/list）
  // （project_info_tree / project_info_search 已并入单工具 project_info——op=tree/search）
  // （project_info_delete / project_info_explore 已并入单工具 project_info——op=delete/explore）
];


// ─── JS 原生化实现（ctx.fs） ────────────────────

// infoDirFromArgs 解析 project 参数 → 目标项目知识库目录（缺省 = 主项目）。
// 多项目工作区中知识库按项目隔离（.pair/project-info/ 在各项目根下）。
function infoDir(ctx, args) {
  const project = args && args.project
  if (project && !/^[a-zA-Z]:[\/]/.test(project) && !project.startsWith('/')) {
    return '../' + String(project).replace(/[\/]+$/, '') + '/.pair/project-info'
  }
  return '.pair/project-info'
}

// safeInfoPath 规范化条目路径：去 .md、清理、禁路径穿越（..、绝对路径），允许 / 嵌套。
function cleanSegs(p) {
  const out = []
  for (const s of String(p).split('/')) {
    if (s === '' || s === '.') continue
    if (s === '..') { out.pop(); continue }
    out.push(s)
  }
  return out.join('/')
}
function safeInfoPath(p) {
  let s = String(p == null ? '' : p).replace(/\.md\s*$/, '').trim()
  s = s.replace(/\\/g, '/')
  return cleanSegs('/' + s).replace(/^\/+|\/+$/g, '')
}

const infoBranches = ['目标', '架构', '实现', '关键点', '设计思想']
function isInfoBranch(head) {
  return infoBranches.indexOf(head) >= 0
}

// notesToBranchRel 把决策树路径映射到知识库树分支路径（复刻 Go 实现）。
function notesToBranchRel(n) {
  n = String(n).replace(/^notes\//, '').replace(/^\//, '')
  if (n === '') return null
  const segs = n.split('/')
  const leaf = segs[segs.length - 1]
  let branch
  if (segs.length >= 2 && segs[0] === 'implemented' && segs[1] === 'architecture') branch = '架构'
  else if (segs.length >= 2 && segs[0] === 'implemented' && (segs[1] === 'decision' || segs[1] === 'decisions')) branch = '设计思想'
  else if (segs.length >= 2 && segs[0] === 'implemented' && segs[1] === 'feature') branch = '实现'
  else if (segs.length >= 2 && segs[0] === 'implemented') branch = '关键点'
  else if (segs[0] === 'decision' || segs[0] === 'decisions') branch = '设计思想'
  else if (segs.length >= 2 && segs[0] === 'inbox') branch = '实现'
  else branch = '实现'
  return path.join(branch, leaf)
}

// firstHeading 取 Markdown 首行 # 标题。
function firstHeading(md, fallback) {
  for (const ln of String(md).split('\n')) {
    const s = ln.trim()
    if (s.startsWith('# ')) return s.slice(2).trim()
  }
  return fallback
}

// infoLevel 按路径分级：overview = 根条目；module = 1 层；detail = 2+ 层。
function infoLevel(rel) {
  const low = String(rel).toLowerCase()
  if (low === 'overview' || rel === '概览' || rel === '项目概览') return 'overview'
  if (String(rel).split('/').length >= 3) return 'detail'
  return 'module'
}

// walkMd 递归扫描目录下的 .md 条目（返回 {rel, title, content} 列表）。
function walkMd(ctx, base, prefix) {
  let names = []
  try { names = ctx.fs.readdir(base) } catch (e) { return [] }
  const out = []
  for (const n of names) {
    const full = path.join(base, n)
    let st = null
    try { st = ctx.fs.stat(full) } catch (e) { continue }
    if (st.isDir) {
      out.push(...walkMd(ctx, full, path.join(prefix, n)))
      continue
    }
    if (!n.endsWith('.md')) continue
    let content = ''
    try { content = ctx.fs.readFile(full) } catch (e) { continue }
    const rel = (prefix ? prefix + '/' : '') + n.slice(0, -3)
    out.push({ rel, title: firstHeading(content, rel), level: infoLevel(rel), content })
  }
  return out
}

// scanInfoEntries 递归扫描知识库目录；附加源 .agents/notes/ 并入（路径前缀 notes/）。
function scanInfoEntries(ctx, dir) {
  const out = walkMd(ctx, dir, '')
  // 项目根：dir 为 <root>/.pair/project-info → root = 去掉后缀
  const root = String(dir).replace(/\/\.pair\/project-info$/, '')
  const notes = root + '/.agents/notes'
  if (notes !== dir) {
    for (const e of walkMd(ctx, notes, 'notes')) {
      const br = notesToBranchRel(e.rel)
      if (!br) continue
      // 树中已有镜像副本 → 跳过，避免重复
      if (ctx.fs.exists(path.join(dir, br + '.md'))) continue
      out.push({ rel: e.rel, title: e.title, level: infoLevel(e.rel), content: e.content })
    }
  }
  out.sort((a, b) => (a.rel < b.rel ? -1 : a.rel > b.rel ? 1 : 0))
  return out
}

// partsOf 取条目路径末段（文件名段）。
function partsOf(rel) {
  const i = String(rel).lastIndexOf('/')
  return i >= 0 ? rel.slice(i + 1) : rel
}

// infoTree 构建知识库条目树（分支=目录，叶子=条目），返回缩进树文本。
function infoTree(entries, showLevel) {
  const root = { children: {} }
  for (const e of entries) {
    const parts = e.rel.split('/')
    let cur = root
    for (let i = 0; i < parts.length - 1; i++) {
      const seg = parts[i]
      if (!cur.children[seg]) cur.children[seg] = { name: seg, children: {} }
      cur = cur.children[seg]
    }
    const leaf = parts[parts.length - 1]
    if (!cur.children[leaf]) cur.children[leaf] = { name: leaf, children: {} }
    cur.children[leaf].entry = e
  }
  let b = ''
  const walk = (n, prefix) => {
    const keys = Object.keys(n.children).sort()
    for (let i = 0; i < keys.length; i++) {
      const ch = n.children[keys[i]]
      const last = i === keys.length - 1
      let conn = '├── ', nextPrefix = prefix + '│   '
      if (last) { conn = '└── '; nextPrefix = prefix + '    ' }
      if (ch.entry) {
        let mark = ''
        if (showLevel) mark = ' [' + ch.entry.level + ']'
        let title = ch.entry.title
        const leaf = partsOf(ch.entry.rel)
        if (leaf && leaf !== title) title += '（' + leaf + '）'
        b += prefix + conn + title + mark + '\n'
      } else {
        b += prefix + conn + keys[i] + '/\n'
      }
      walk(ch, nextPrefix)
    }
  }
  walk(root, '')
  return b
}

// infoKeyFile 是否为「根目录关键文件」。
function infoKeyFile(n) {
  const low = String(n).toLowerCase()
  if (low.startsWith('readme') || low.startsWith('makefile')) return true
  return ['go.mod', 'package.json', 'cargo.toml', 'pyproject.toml', 'pom.xml',
    'main.go', 'agents.md', 'claude.md', 'go.sum', 'tsconfig.json'].indexOf(low) >= 0
}

// isSkipDir 常见忽略目录（explore/文件树统计用）。
const skipDirs = new Set(['node_modules', '.git', 'dist', 'build', 'release', '.cache', 'logs', 'tmp', '_temp', '.verify-tmp', '.agent-teams', 'bin', '.pair', '.agents'])
function isSkipDir(n) {
  return skipDirs.has(String(n).toLowerCase())
}

// countDirFiles 数目录下文件（递归，跳过依赖/产物目录，上限 2000 防卡）。
function countDirFiles(ctx, dir) {
  let n = 0
  const walk = (d) => {
    let names = []
    try { names = ctx.fs.readdir(d) } catch (e) { return }
    for (const name of names) {
      if (n >= 2000) return
      let st = null
      try { st = ctx.fs.stat(path.join(d, name)) } catch (e) { continue }
      if (st.isDir) {
        if (!isSkipDir(name)) walk(path.join(d, name))
      } else {
        n++
      }
    }
  }
  walk(dir)
  return n
}

// exploreProjectStructure 轻量项目结构概览（复刻 Go exploreProjectStructure）。
function exploreProjectStructure(ctx) {
  let names = []
  try { names = ctx.fs.readdir('.') } catch (e) {
    return '无法读取项目根目录：' + (e && e.message ? e.message : String(e))
  }
  const keyFiles = names.filter(n => !isSkipDir(n) && !n.startsWith('.') && infoKeyFile(n))
  const dirs = names.filter(n => !isSkipDir(n) && !n.startsWith('.'))
  let b = '# 项目结构概览（供分析后写入知识库）\n\n## 根目录关键文件\n'
  for (const f of keyFiles) b += '- ' + f + '\n'
  b += '\n## 顶层目录（约略文件数）\n'
  for (const d of dirs) {
    let st = null
    try { st = ctx.fs.stat(d) } catch (e) { continue }
    if (!st.isDir) continue
    b += '- ' + d + '/（约 ' + countDirFiles(ctx, d) + ' 文件）\n'
  }
  b += '\n建议：用 read 读关键文件分析后，project_info(op=write) 写入「概览」「模块-<名>」「决策-<主题>」等中文条目。'
  return b
}

// projectInfoWrite（op=write）：写/更新 + notes/ 镜像 + 非分支路径提示。
function projectInfoWrite(ctx, args) {
  const dir = infoDir(ctx, args)
  const rel = safeInfoPath(args.path)
  if (rel === '') throw new Error('path 不能为空')
  let branchRel = rel, mirrorRel = ''
  if (rel.startsWith('notes/')) {
    const br = notesToBranchRel(rel)
    if (br) branchRel = br
    mirrorRel = rel.replace(/^notes\//, '')
  }
  const fp = path.join(dir, branchRel + '.md')
  const slash = Math.max(fp.lastIndexOf('/'), fp.lastIndexOf('\\'))
  if (slash > 0) {
    const pd = fp.slice(0, slash)
    if (pd && !ctx.fs.exists(pd)) ctx.fs.mkdir(pd, true)
  }
  const updating = ctx.fs.exists(fp)
  ctx.fs.writeFile(fp, String(args.content == null ? '' : args.content))
  if (mirrorRel) { // 镜像：.agents/notes/<原相对路径>.md
    const root = String(dir).replace(/\/\.pair\/project-info$/, '')
    const nfp = root + '/.agents/notes/' + mirrorRel + '.md'
    const nsl = Math.max(nfp.lastIndexOf('/'), nfp.lastIndexOf('\\'))
    if (nsl > 0) {
      const npd = nfp.slice(0, nsl)
      if (npd && !ctx.fs.exists(npd)) ctx.fs.mkdir(npd, true)
    }
    ctx.fs.writeFile(nfp, String(args.content == null ? '' : args.content))
  }
  let head = branchRel
  const hi = head.indexOf('/')
  if (hi > 0) head = head.slice(0, hi)
  let hint = ''
  if (!rel.startsWith('notes/') && head !== '概览' && !isInfoBranch(head)) {
    hint = '（提示：知识库是树，建议用顶层分支 目标/架构/实现/关键点/设计思想 开头，如 架构/' + branchRel + '）'
  }
  const verb = updating ? '已更新知识库' : '已写入知识库'
  if (mirrorRel) return verb + '：' + branchRel + '（notes/ 参考路径已镜像 .agents/notes/' + mirrorRel + '）'
  return verb + '：' + branchRel + hint
}

function projectInfoRead(ctx, args) {
  const dir = infoDir(ctx, args)
  const rel = safeInfoPath(args.path)
  const fp = path.join(dir, rel + '.md')
  if (!ctx.fs.exists(fp)) throw new Error('无此知识库条目：' + rel + '（用 project_info(op=list) 看全部）')
  return ctx.fs.readFile(fp)
}

function projectInfoList(ctx, args) {
  const entries = scanInfoEntries(ctx, infoDir(ctx, args))
  if (entries.length === 0) return '（知识库为空。用 project_info(op=explore) 起步、project_info(op=write) 写入，或菜单「探索项目知识库」。）'
  return infoTree(entries, true)
}

function projectInfoTree(ctx, args) {
  const entries = scanInfoEntries(ctx, infoDir(ctx, args))
  if (entries.length === 0) return '（知识库为空）'
  return '# 项目知识库（树）\n' + infoTree(entries, false)
}

function projectInfoSearch(ctx, args) {
  const q = String(args.query == null ? '' : args.query).toLowerCase().trim()
  if (q === '') throw new Error('query 不能为空')
  const lines = []
  for (const e of scanInfoEntries(ctx, infoDir(ctx, args))) {
    if ((e.rel + e.title + e.content).toLowerCase().includes(q)) {
      lines.push('- ' + e.title + '（' + e.rel + '）')
    }
  }
  if (lines.length === 0) return '（无匹配条目）'
  return lines.join('\n')
}

function projectInfoDelete(ctx, args) {
  const dir = infoDir(ctx, args)
  const rel = safeInfoPath(args.path)
  const fp = path.join(dir, rel + '.md')
  if (!ctx.fs.exists(fp)) throw new Error('无此知识库条目：' + rel)
  ctx.fs.rm(fp, false)
  return '已删除知识库条目：' + rel
}

function projectInfoExplore(ctx, args) {
  return exploreProjectStructure(ctx)
}

const impls = {
  write: projectInfoWrite,
  read: projectInfoRead,
  list: projectInfoList,
  tree: projectInfoTree,
  search: projectInfoSearch,
  delete: projectInfoDelete,
  explore: projectInfoExplore,
}


return {
  name: 'tool-project-info',
  inject: ['fs'],
  purpose: '项目知识库（project_info(op=write/read/list/tree/search/delete/explore)）——迁移自内置 Go 工具组；调用实现（JS 编排 ctx.fs）完全在插件内（Round2 JS 原生化；2026-09 由 7 工具合并为单工具）',
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
        execute: (args) => {
          const a = args || {}
          const fn = impls[a.op]
          if (!fn) {
            return Promise.resolve(
              'project_info：op 无效（可用 write/read/list/tree/search/delete/explore）——未执行任何操作'
            )
          }
          return fn(ctx, a)
        },
      })
    }
  },
}
