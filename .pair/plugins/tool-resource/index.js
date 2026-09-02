// ═══════════════════════════════════════════════════════════════
// tool-resource — 资源管理（resource_list/resource_search/resource_stats）
//               + 知识库过期验证（memory_verify/project_info_verify）
//
// 合并说明（2026-09-04）：tool-verify 插件已并入本插件（验证对象同为
// .pair/memory 与 .pair/project-info 资产，与资源管理同源），JS 实现完全搬迁，
// 行为不变；tool-verify 插件目录已删除。
//
// resource_* 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go
// 自动生成，schema 完整外置拷贝）。api 声明在插件，execute 调 ctx.hostTool 复用宿主 Go 执行器；
// verify_* 为 JS 原生化（编排+能力均在插件内，复刻 pkg/verify 规则）。
// 工具清单：resource_list、resource_search、resource_stats、memory_verify、project_info_verify
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    "name": "resource_list",
    "description": "列出所有智能资源（经验胶囊、技能基因、记忆、项目知识库、技能、MCP服务器、Lua工具）。可按类型（type）和作用域（scope）过滤。不传 type 则列出全部类型。",
    "parameters": {
      "properties": {
        "detail": {
          "description": "可选：是否显示详细信息（含描述、标签等），默认 false",
          "type": "boolean"
        },
        "scope": {
          "description": "可选：作用域过滤，\"global\"|\"project\"，不传=全部",
          "type": "string"
        },
        "type": {
          "description": "可选：资源类型过滤，如 \"capsules\"|\"genes\"|\"memory\"|\"project-info\"|\"skills\"|\"mcp-servers\"|\"lua-tools\"，不传=全部",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "resource_search",
    "description": "跨类型搜索智能资源（经验胶囊、技能基因、记忆、知识库等）。按名称、描述、标签模糊匹配。可用 type 和 scope 缩小范围。",
    "parameters": {
      "properties": {
        "query": {
          "description": "搜索关键词（必填，匹配名称/描述/标签）",
          "type": "string"
        },
        "scope": {
          "description": "可选：作用域过滤，\"global\"|\"project\"，不传=全部",
          "type": "string"
        },
        "type": {
          "description": "可选：资源类型过滤，如 \"capsules\"|\"genes\"|\"memory\"，不传=全部",
          "type": "string"
        }
      },
      "required": [
        "query"
      ],
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "resource_stats",
    "description": "查看各类智能资源的数量统计。直观展示经验胶囊、基因、记忆、知识库、技能、MCP服务器、Lua工具的数量分布。",
    "parameters": {
      "properties": {
        "scope": {
          "description": "可选：作用域过滤，\"global\"|\"project\"，不传=全部",
          "type": "string"
        }
      },
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "memory_verify",
    "description": "验证所有记忆条目中引用的文件和目录是否仍然存在。如果条目引用了已不存在的文件，可能是过时信息，建议更新或删除。返回验证报告，包含每个过期条目的问题描述。",
    "usageGuide": "验证所有记忆条目引用的文件和目录是否仍然存在。过时记忆会误导 agent，建议定期运行。比手动检查更高效（自动解析引用路径并检测有效性）。",
    "parameters": {
      "properties": {},
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "project_info_verify",
    "description": "验证所有知识库条目中引用的文件和目录是否仍然存在。如果条目引用了已不存在的文件/目录，可能是过时信息，建议更新或删除。返回验证报告，包含每个过期条目的问题描述。",
    "usageGuide": "验证知识库条目引用的文件和目录是否仍然存在。项目重构后文件移动可能导致旧引用失效，运行此工具可发现并清理过时条目。",
    "parameters": {
      "properties": {},
      "type": "object"
    },
    "readOnly": true
  }
];

// isHistoricalRecord 判断标题是否属于历史记录类（修复/排查/改造/决策等过程记录）。
// 这些条目是「发生了什么」的历史事实，引用文件的存在性随时间自然失效，不视为过期。
const histKeywords = [
  '修复记录', '修复：', '修复', '排查记录', '历史', '评估报告', '异常报告',
  '性能', '体检', '验证', '冒烟', '升级',
  '改造记录', '实施记录', '重写方案', '决策', '方案', '设计',
]
function isHistoricalRecord(title) {
  for (const kw of histKeywords) {
    if (String(title).includes(kw)) return true
  }
  return false
}

// 引用提取正则（复刻 pkg/verify）：
// fullPathRE 要求至少一层目录 + 文件扩展名（排除裸文件名误报）。
const fullPathRE = /(?:\w+\/)[\w./-]*\.(?:go|ts|vue|jsx|json|yaml|xml|markdown|md|css|html|rs|py|java|rb|php|swift|kt|dart|lua|sh|sql|js)\b/g
// dirPathRE 匹配目录引用（行内的 cmd/companion/ 模式）。
const dirPathRE = /\b(?:cmd\/[a-zA-Z0-9_/-]+|internal\/[a-zA-Z0-9_/-]+|pkg\/[a-zA-Z0-9_/-]+|config\/[a-zA-Z0-9_/-]+|scripts\/[a-zA-Z0-9_/-]+)\b/g

const versionRE = /^v?\d+\.\d+\.\d+$/
const importPrefixes = ['github.com/', 'golang.org/', 'google.golang.org/', 'gopkg.in/', 'pkg.go.dev/', 'npmjs.com/', 'pypi.org/', 'crates.io/']

function looksLikeVersion(s) { return versionRE.test(s) }
function looksLikeURL(s) { return s.indexOf('://') >= 0 }
function looksLikeImport(s) {
  for (const p of importPrefixes) if (s.startsWith(p)) return true
  return false
}

// cleanRel 归一化引用路径（去 ./ 与重复 /）。
function cleanRel(p) {
  const segs = String(p).split('/')
  const out = []
  for (const s of segs) {
    if (s === '' || s === '.') continue
    if (s === '..') { out.pop(); continue }
    out.push(s)
  }
  return out.join('/')
}

// extractFileRefs 从文本中提取可能的文件路径引用（复刻 Go 过滤规则）。
function extractFileRefs(text) {
  const seen = new Set()
  const refs = []
  fullPathRE.lastIndex = 0
  let m
  while ((m = fullPathRE.exec(text)) !== null) {
    const s = m[0]
    if (looksLikeVersion(s) || looksLikeURL(s) || looksLikeImport(s)) continue
    if (s.startsWith('pair/') || s.startsWith('.pair/')) continue
    if (m.index > 0 && text[m.index - 1] === '.') continue // 点扩展名序列（.js/.ts）
    const clean = cleanRel(s)
    if (!seen.has(clean)) {
      seen.add(clean)
      refs.push(clean)
    }
  }
  return refs
}

// extractDirRefs 从文本中提取目录引用。
function extractDirRefs(text) {
  const seen = new Set()
  const refs = []
  dirPathRE.lastIndex = 0
  let m
  while ((m = dirPathRE.exec(text)) !== null) {
    if (text[m.index + m[0].length] === '.') continue // 实际是文件路径前缀
    const clean = cleanRel(m[0])
    if (!seen.has(clean)) {
      seen.add(clean)
      refs.push(clean)
    }
  }
  return refs
}

// existsAny 检查引用在工作区任一根下存在（文件或目录）。
function existsAny(ctx, ref) {
  let roots = []
  try { roots = ctx.fs.roots() } catch (e) { roots = [] }
  if (roots.length === 0) roots = ['.']
  for (const r of roots) {
    try {
      const st = ctx.fs.stat(r + '/' + ref)
      if (st) return true
    } catch (e) { /* 不存在，试下一个根 */ }
  }
  return false
}

// checkEntry 检查单条文本中的引用是否有效（复刻 pkg/verify.checkEntry）。
function checkEntry(ctx, title, content) {
  const issues = []
  const text = String(title) + '\n' + String(content)
  if (isHistoricalRecord(title)) return issues
  for (const ref of extractFileRefs(text)) {
    if (!existsAny(ctx, ref)) issues.push('引用的文件已不存在: ' + ref)
  }
  for (const ref of extractDirRefs(text)) {
    if (!existsAny(ctx, ref)) issues.push('引用的目录已不存在: ' + ref)
  }
  return issues
}

// fmtTime 格式化为 yyyy-MM-dd HH:mm:ss。
function fmtTime(d) {
  const p = n => String(n).padStart(2, '0')
  return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
}

// formatReport 格式化验证报告（复刻 formatVerifyReport 输出形态）。
function formatReport(source, checkedAt, entries, stale) {
  let b = '## ' + source + ' 验证报告 (' + checkedAt + ')\n'
  b += '共检查 ' + entries.length + ' 条，正常 ' + (entries.length - stale.length) + ' 条'
  if (stale.length > 0) {
    b += '，发现 **' + stale.length + ' 条可能过期**：\n\n'
    stale.forEach((s, i) => {
      b += (i + 1) + '. **' + s.title + '** (' + s.id + ')\n'
      for (const issue of s.issues) b += '   - ' + issue + '\n'
    })
    b += '\n建议：\n'
    b += '- 对过期条目可删除或用更新类工具刷新（工具名称与用法见 tools 参数 schema）\n'
    b += '- 对知识库过期条目可删除或用更新类工具刷新\n'
    b += '- 定期执行过期检查类工具保持数据新鲜\n'
  } else {
    b += '，全部正常。\n'
  }
  return b
}

// listMdEntries 扫描目录下 .md 条目（递归）→ [{id, title, content}]。
function listMdEntries(ctx, dir) {
  const out = []
  const walk = (base, prefix) => {
    let names = []
    try { names = ctx.fs.readdir(base) } catch (e) { return }
    for (const n of names) {
      const full = base + '/' + n
      let st = null
      try { st = ctx.fs.stat(full) } catch (e) { continue }
      if (st.isDir) { walk(full, prefix + '/' + n); continue }
      if (!n.endsWith('.md')) continue
      if (n === 'MEMORY.md') continue // 记忆索引非条目
      let content = ''
      try { content = ctx.fs.readFile(full) } catch (e) { continue }
      let title = firstHeading(content, n.slice(0, -3))
      out.push({ id: (prefix ? prefix + '/' : '') + n.slice(0, -3), title, content })
    }
  }
  walk(dir, '')
  return out
}

function firstHeading(md, fallback) {
  for (const ln of String(md).split('\n')) {
    const s = ln.trim()
    if (s.startsWith('# ')) return s.slice(2).trim()
  }
  return fallback
}

// verifyDir 通用验证：扫描 dir 条目 → 引用校验 → 报告。
function verifyDir(ctx, source, dir) {
  const entries = listMdEntries(ctx, dir)
  const checkedAt = fmtTime(new Date())
  const stale = []
  for (const e of entries) {
    const issues = checkEntry(ctx, e.title, e.content)
    if (issues.length > 0) {
      stale.push({ id: e.id, title: e.title, issues })
    }
  }
  stale.sort((a, b) => (a.title < b.title ? -1 : a.title > b.title ? 1 : 0))
  return formatReport(source, checkedAt, entries, stale)
}

function memoryVerify(ctx) {
  return verifyDir(ctx, '记忆', '.pair/memory')
}

function projectInfoVerify(ctx) {
  return verifyDir(ctx, '知识库', '.pair/project-info')
}

const impls = {
  memory_verify: memoryVerify,
  project_info_verify: projectInfoVerify,
}


return {
  name: 'tool-resource',
  inject: ['fs'],
  purpose: '资源管理（resource_list/resource_search/resource_stats）+ 知识库过期验证（memory_verify/project_info_verify）（tool-verify 已并入）',
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
        execute: (args) => impls[t.name] ? impls[t.name](ctx, args || {}) : ctx.hostTool.exec(t.name, args || {}),
      })
    }
  },
}
