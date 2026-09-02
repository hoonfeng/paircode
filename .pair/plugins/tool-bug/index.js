// ═══════════════════════════════════════════════════════════════
// tool-bug — BUG 检测与修复（bug_detect/bug_fix）
// ★ Round4 削减：bug_analyze 删除（bug_detect/bug_fix 已覆盖构建输出→错误定位链）
//
// 迁移（2026-08-22 Round2）：binary 形态 → JS 原生（对齐 tool-core 模式）。
// 原 execute 调 ctx.binary.exec 复用插件目录 bin/ 下独立二进制（已归档
// bin/legacy-plugin-bins/），现实现完全在插件内（bug_detect/bug_fix 经 ctx.bash 编排
// go vet/build/test，内置 analyzeVet/analyzeBuild/analyzeTest 解析器），不再依赖 ctx.binary。
// 行为复刻 internal/agent/bugdetect.go（编译/测试/运行三形态解析、vet 回退、
// 摘要与修复提示格式、build 失败短路跳过 test）。
// 工具清单：bug_detect、bug_fix
// ═══════════════════════════════════════════════════════════════
const tools = [

  {
    "name": "bug_detect",
    "description": "全量检测项目中是否存在 BUG。自动运行 go vet → go build → go test，输出解析后的错误列表（含文件路径、行号、错误消息和代码上下文）。用于自动发现编译/测试/运行时的 BUG。集成在自主模式的编排循环中。",
    "usageGuide": "全量检测项目 BUG：自动运行 go vet → go build → go test，聚合所有错误。修改代码后验证无错误的推荐工具。比手动分别运行更高效（一站式检测）。",
    "parameters": {
      "properties": {},
      "type": "object"
    },
    "readOnly": true
  },
  {
    "name": "bug_fix",
    "description": "自动检测项目 BUG（编译/测试/运行时错误），生成详细的修复任务文本。返回包含错误位置、代码上下文和修复指南的完整修复任务。可用于自主模式中在 loop 之间自动检测并修复项目问题。",
    "usageGuide": "自动检测并修复项目 BUG。运行编译/测试后定位错误，生成修复方案并 apply。max_attempts 控制最大尝试次数（默认 3）。注意：自动修复可能引入新问题，改完需验证。",
    "parameters": {
      "properties": {
        "max_attempts": {
          "description": "可选：最大修复尝试次数，默认 3",
          "type": "integer"
        }
      },
      "type": "object"
    }
  }
];


// ─── JS 原生化实现（纯解析 + ctx.bash） ────────────────────

// 解析正则（复刻 internal/agent/bugdetect.go）
const goCompileRe = /^(.+?\.go):(\d+)(?::(\d+))?:\s*(.+)$/
const goVetRe = /^(.+\.go):(\d+):(\d+)?:\s*(.+)$/
const goTestFailRe = /^---\s+FAIL:\s+(.+)\s+\(/
const goTestStackRe = /^\s*(.+\.go):(\d+)\s+\(0x[0-9a-f]+\)/
const goPanicFileRe = /^\s*(.+\.go):(\d+)(?:\s+\+0x[0-9a-f]+)?\s*$/

function truncate(s, max) {
  s = String(s)
  return s.length <= max ? s : s.slice(0, max) + '...'
}

// errorContext 读错误位置附近代码（5 行，失败返回空串）。
function errorContext(ctx, file, lineNum, ctxLines) {
  if (!file || !(lineNum > 0)) return ''
  let text
  try { text = ctx.fs.readFile(file) } catch (e) { return '' }
  const lines = text.split('\n')
  let start = lineNum - ctxLines - 1
  if (start < 0) start = 0
  let end = lineNum + ctxLines
  if (end > lines.length) end = lines.length
  let b = ''
  for (let i = start; i < end; i++) {
    const marker = i + 1 === lineNum ? '→' : ' '
    b += '  ' + marker + ' ' + String(i + 1).padStart(4) + ': ' + lines[i] + '\n'
  }
  return b
}

// analyzeBuild 解析 go build 输出（file:line:col: msg；或 error/cannot/undefined 行）。
function analyzeBuild(ctx, output) {
  const symptoms = []
  for (const raw of String(output).split('\n')) {
    const line = raw.trim()
    if (!line) continue
    const m = goCompileRe.exec(line)
    if (m) {
      symptoms.push({ type: 'compile', severity: 'error', msg: m[4], file: m[1], line: +m[2], col: m[3] ? +m[3] : 0, ctx: errorContext(ctx, m[1], +m[2], 5) })
    } else if (/error|cannot|undefined|unexpected/i.test(line)) {
      symptoms.push({ type: 'compile', severity: 'error', msg: line })
    }
  }
  return symptoms
}

// analyzeVet 解析 go vet 输出。
function analyzeVet(ctx, output) {
  const symptoms = []
  for (const raw of String(output).split('\n')) {
    const line = raw.trim()
    if (!line) continue
    const m = goVetRe.exec(line)
    if (m) {
      symptoms.push({ type: 'vet', severity: 'warning', msg: m[4], file: m[1], line: +m[2], col: m[3] ? +m[3] : 0, ctx: errorContext(ctx, m[1], +m[2], 3) })
    }
  }
  return symptoms
}

// analyzeTest 解析 go test 输出（--- FAIL: + panic/栈帧）。
function analyzeTest(ctx, output) {
  const symptoms = []
  let currentTest = '', inStack = false, stack = []
  const seen = new Set()
  for (const line of String(output).split('\n')) {
    const m = goTestFailRe.exec(line)
    if (m) {
      if (seen.has(m[1])) continue
      seen.add(m[1])
      currentTest = m[1]
      stack = []
      inStack = true
      continue
    }
    if (!inStack) continue
    if (line.startsWith('panic:') || line.startsWith('fatal error:')) {
      let msg = line.trim()
      if (currentTest) msg = '测试 ' + currentTest + ' 中 ' + msg
      symptoms.push({ type: 'panic', severity: 'error', msg })
      inStack = false
      continue
    }
    const sm = goTestStackRe.exec(line)
    if (sm) {
      symptoms.push({ type: 'test', severity: 'error', msg: '测试失败: ' + currentTest, file: sm[1], line: +sm[2], ctx: errorContext(ctx, sm[1], +sm[2], 5) })
      stack.push(line)
      inStack = false
      continue
    }
    if (line.trim() === '' && stack.length > 0) inStack = false
    else stack.push(line)
  }
  if (symptoms.length === 0 && (/FAIL/.test(output) || /panic/.test(output))) {
    symptoms.push({ type: 'test', severity: 'error', msg: '测试失败（原始输出见 BuildOutput）' })
  }
  return symptoms
}



// bash 在项目根执行命令（输出截断 16000 由宿主保证）。
function bash(ctx, cwd, command, timeoutSec) {
  const res = ctx.bash.exec(command, cwd || '', timeoutSec || 120)
  return (res.output || '').trim()
}

// detectProjectErrors 复刻 DetectProjectErrors：vet → build（失败短路）→ test。
function detectProjectErrors(ctx) {
  if (!ctx.fs.exists('go.mod')) return { skipped: '（非 Go 项目，跳过检测）', symptoms: [] }
  const symptoms = []
  const vet = bash(ctx, '', 'go vet -tags webonly ./cmd/companion', 60)
  if (vet) symptoms.push(...analyzeVet(ctx, vet))
  const build = bash(ctx, '', 'go build -tags webonly ./cmd/companion', 120)
  if (build) {
    symptoms.push(...analyzeBuild(ctx, build))
    return { skipped: '', symptoms }
  }
  const test = bash(ctx, '', 'go test -count=1 -timeout 30s ./cmd/companion/agent', 60)
  if (test && (/FAIL/.test(test) || /panic:/.test(test))) {
    symptoms.push(...analyzeTest(ctx, test))
  }
  return { skipped: '', symptoms }
}

// buildSummary 复刻 buildSummary 摘要格式。
function buildSummary(symptoms) {
  let b = '[失败] 发现 ' + symptoms.length + ' 个问题:\n\n'
  const tc = {}
  for (const s of symptoms) tc[s.type] = (tc[s.type] || 0) + 1
  for (const t of Object.keys(tc)) {
    if (t === 'compile') b += '  - 编译错误: ' + tc[t] + ' 个\n'
    else if (t === 'test') b += '  - 测试失败: ' + tc[t] + ' 个\n'
    else if (t === 'panic') b += '  - 运行时 panic: ' + tc[t] + ' 个\n'
    else if (t === 'vet') b += '  - go vet 警告: ' + tc[t] + ' 个\n'
    else b += '  - 其他问题: ' + tc[t] + ' 个\n'
  }
  let shown = 0
  for (const s of symptoms) {
    if (shown >= 5) { b += '  ... 还有 ' + (symptoms.length - shown) + ' 个问题\n'; break }
    b += s.file ? '  - ' + s.file + ':' + s.line + ': ' + truncate(s.msg, 80) + '\n' : '  - ' + truncate(s.msg, 80) + '\n'
    shown++
  }
  return b
}

function bugDetect(ctx, args) {
  const r = detectProjectErrors(ctx)
  if (r.skipped) return r.skipped
  if (r.symptoms.length === 0) return '[成功] 项目检测通过，未发现错误。'
  return buildSummary(r.symptoms)
}

// bug_fix：检测 + 生成修复任务文本（复刻 BuildDetailedFixPrompt 形态）。
function bugFix(ctx, args) {
  const r = detectProjectErrors(ctx)
  if (r.skipped) return r.skipped
  if (r.symptoms.length === 0) return '[成功] 项目检测通过，无需修复。'
  let b = '# 自动检测到项目 BUG\n\n项目构建/测试失败，请分析并修复以下问题。\n\n## 错误摘要\n'
  b += buildSummary(r.symptoms) + '\n\n## 详细错误列表\n\n'
  r.symptoms.forEach((s, i) => {
    b += '### 错误 ' + (i + 1) + ': ' + s.msg + '\n\n'
    if (s.file) b += '- **文件**: `' + s.file + ':' + s.line + '`\n'
    b += '- **类型**: ' + s.type + '\n'
    b += '- **严重程度**: ' + (s.severity || 'error') + '\n\n'
    if (s.ctx) b += '**代码上下文**:\n```\n' + s.ctx + '```\n\n'
  })
  b += '## 修复指南\n\n'
  b += '1. 读取每个错误位置附近的代码（工具名称与用法见 tools 参数 schema）\n'
  b += '2. 分析错误原因：是语法错误、类型不匹配、未定义标识符还是其他问题\n'
  b += '3. 用编辑类工具修复\n'
  b += '4. 修改后运行构建验证是否通过\n'
  b += '5. 如果仍有错误，继续修复\n'
  b += '6. 所有错误修复完成后，确认全部通过，然后输出完成总结。\n\n请开始修复。'
  return b
}

const impls = {
  bug_detect: bugDetect,
  bug_fix: bugFix,
}


return {
  name: 'tool-bug',
  inject: ['fs', 'bash'],
  purpose: 'BUG 检测与修复（bug_detect/bug_analyze/bug_fix）——迁移自内置 Go 工具组；调用实现（JS 编排 ctx.fs/ctx.bash）完全在插件内（Round2 JS 原生化）',
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
