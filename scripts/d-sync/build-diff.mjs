#!/usr/bin/env node
// 生成 E:\paircode-master(master 基准) ↔ D:\PairCode 的完整逐文件差异清单
// 产出：diff-report.json（机器可读） + diff-report.md（人可读）
// 只读，不修改任何一方文件。
import fs from 'node:fs'
import path from 'node:path'
import crypto from 'node:crypto'

const ROOT = 'E:/paircode-master'
const OUT = path.join(ROOT, 'temp/sync-D-PairCode')
const E = ROOT
const D = 'D:/PairCode'
const IGNORE = new Set(['node_modules', '.git', '.ui-backup'])

// ── 同步候选分组（对齐 packager.json 的 dist.include）──
const GROUPS = [
  { key: '.pair/plugins',             e: '.pair/plugins',             d: '.pair/plugins',             kind: '插件产物',       note: 'D 多出 9 个本地插件 + agentloop 的 .bak' },
  { key: '.pair/assets/runtime/web',  e: '.pair/assets/runtime/web',  d: '.pair/assets/runtime/web',  kind: '壳产物',         note: '壳 js 文件名不同（DjWrYO0q vs CTZe2XhI）' },
  { key: '.pair/assets/runtime',      e: '.pair/assets/runtime',      d: '.pair/assets/runtime',      kind: '运行时运行时资产', note: 'bridge_node.js / cordis.bundle.js / README.md' },
  { key: 'plugins-src/ui-app',        e: 'plugins-src/ui-app',        d: 'plugins-src/ui-app',        kind: 'UI 源码',        note: 'D 含 setFocusMode/convListVisible 本地改动（未提交）' },
  { key: 'plugins-src/plugins',       e: 'plugins-src/plugins',       d: 'plugins-src/plugins',       kind: 'Go 插件源码',    note: 'D 多 tool-debug/tool-git' },
  { key: 'docs',                      e: 'docs',                      d: 'docs',                      kind: '文档',           note: '' },
  { key: 'config/skills',             e: 'config/skills',             d: 'config/skills',             kind: '内置技能',       note: '' },
  { key: 'assets',                    e: 'assets',                    d: 'assets',                    kind: '图标等',         note: '' },
  { key: 'scripts',                   e: 'scripts',                   d: 'scripts',                   kind: '脚本',           note: 'D 只有 2 个构建脚本' },
  { key: 'README.md',                 e: 'README.md',                 d: 'README.md',                 kind: '单文件',         file: true },
  { key: 'config/models → models',    e: 'config/models',             d: 'models',                    kind: '模型配置',       note: '' },
  { key: 'config/skills → bin?',      e: 'bin',                       d: 'bin',                       kind: '二进制配置',     note: '' },
]

// D 中「不同步」的目录/文件（用户数据/运行时产物，仅供报告列出）
const PROTECTED = ['.env', 'config/settings.json', 'config/mcp.json', 'config/models.json', 'config/roles', 'config/philosophy',
  'config/skills(用户自定义)', 'logs', 'temp', 'tools', 'fonts', 'lib', 'pair.exe', '.pair/skills', '.pair/toolsets',
  '.pair/.ui-backup', 'models', 'bin']

const sha = f => { try { return crypto.createHash('sha1').update(fs.readFileSync(f)).digest('hex').slice(0, 12) } catch { return null } }
const isText = f => /\.(js|mjs|cjs|json|md|txt|html|css|vue|svg|go|mod|sum|yml|yaml|json5|ts|sh|cmd|rc|gitignore|npmignore|env)$/i.test(f)
// ★ 行尾归一化比较：D 侧多为 LF，E 侧（git checkout）多为 CRLF
//   → 若用原始字节比较会把大量「仅行尾不同」的文件误报为内容差异。
const normBuf = f => { const b = fs.readFileSync(f); return isText(f) ? Buffer.from(b.toString('utf8').replace(/\r\n/g, '\n'), 'utf8') : b }
const shaN = f => { try { return crypto.createHash('sha1').update(normBuf(f)).digest('hex').slice(0, 12) } catch { return null } }

function walk(root, out = [], depth = 0) {
  if (depth > 14) return out
  let es; try { es = fs.readdirSync(root, { withFileTypes: true }) } catch { return out }
  for (const e of es) {
    if (IGNORE.has(e.name)) continue
    const f = path.join(root, e.name)
    if (e.isDirectory()) walk(f, out, depth + 1); else out.push(f)
  }
  return out
}

function lineDiff(a, b) {
  const al = a.split(/\r?\n/).filter(l => l.trim()), bl = b.split(/\r?\n/).filter(l => l.trim())
  const sb = new Set(bl), sa = new Set(al)
  const eOnly = al.filter(l => !sb.has(l)), dOnly = bl.filter(l => !sa.has(l))
  return { eOnlyLines: eOnly.length, dOnlyLines: dOnly.length, eSamples: eOnly.slice(0, 3).map(s => s.trim().slice(0, 110)), dSamples: dOnly.slice(0, 3).map(s => s.trim().slice(0, 110)) }
}

const report = { generatedAt: new Date().toISOString(), baseline: { remote: 'e6c1e625 (GitHub hoonfeng/paircode master)', E, D }, summary: {}, groups: [], protectedPaths: PROTECTED }

for (const g of GROUPS) {
  const ea = path.join(E, g.e), da = path.join(D, g.d)
  const st = f => { try { return fs.statSync(f).isDirectory() ? 'dir' : 'file' } catch { return 'missing' } }
  const rec = { key: g.key, kind: g.kind, note: g.note, ePath: ea, dPath: da, eState: st(ea), dState: st(da), files: [], onlyE: [], onlyD: [], diff: [], eolOnly: [], skipped: false }
  if (rec.eState === 'missing' || rec.dState === 'missing') { rec.skipped = true; report.groups.push(rec); continue }

  const eFiles = st(ea) === 'file' ? [ea] : walk(ea)
  const dFiles = st(da) === 'file' ? [da] : walk(da)
  const relE = eFiles.map(f => path.relative(ea, f)), relD = dFiles.map(f => path.relative(da, f))
  const setD = new Set(relD), setE = new Set(relE)

  rec.eCount = relE.length; rec.dCount = relD.length
  rec.onlyE = relE.filter(x => !setD.has(x))
  rec.onlyD = relD.filter(x => !setE.has(x))
  for (const rel of relE) {
    if (!setD.has(rel)) continue
    const fa = path.join(ea, rel), fb = path.join(da, rel)
    const ha = sha(fa), hb = sha(fb)
    if (ha === hb) continue
    // 字节不同但行尾归一化后相同 ⇒ 仅行尾差异（CRLF↔LF），不算内容差异
    if (shaN(fa) === shaN(fb)) { rec.eolOnly.push(rel); continue }
    const sa = fs.statSync(fa), sb = fs.statSync(fb)
    const item = { rel, sizeE: sa.size, sizeD: sb.size, mtimeE: sa.mtime.toISOString(), mtimeD: sb.mtime.toISOString() }
    if (isText(fa) && isText(fb) && sa.size < 900000 && sb.size < 900000) {
      try { Object.assign(item, lineDiff(fs.readFileSync(fa, 'utf8'), fs.readFileSync(fb, 'utf8'))) } catch {}
    }
    rec.diff.push(item)
  }
  report.groups.push(rec)
}

// 汇总
let s = { groups: 0, onlyE: 0, onlyD: 0, diff: 0, eol: 0, skipped: 0 }
for (const g of report.groups) {
  if (g.skipped) { s.skipped++; continue }
  s.groups++; s.onlyE += g.onlyE.length; s.onlyD += g.onlyD.length; s.diff += g.diff.length; s.eol += g.eolOnly.length
}
report.summary = s

fs.mkdirSync(OUT, { recursive: true })
fs.writeFileSync(path.join(OUT, 'diff-report.json'), JSON.stringify(report, null, 2), 'utf8')

// ── Markdown 报告 ──
let md = `# D:\\PairCode ↔ master 差异清单\n\n生成时间：${report.generatedAt}\n基准：GitHub hoonfeng/paircode master = \`e6c1e625\`（E 侧已逐字节一致）\n\n`
md += `## 汇总\n\n比较口径：**行尾归一化**（CRLF↔LF 视为相同）；\`仅行尾\` 列 = 内容实质相同、只是换行符不同 → 不建议覆盖。\n\n| 组 | E 文件 | D 文件 | 仅 E（D 缺） | 仅 D（D 本地） | **内容不同** | 仅行尾 |\n|---|---|---|---|---|---|---|\n`
for (const g of report.groups) {
  md += g.skipped ? `| ${g.key} | — | — | — | — | — | — *(跳过：E=${g.eState} D=${g.dState})* |\n`
    : `| ${g.key} | ${g.eCount} | ${g.dCount} | ${g.onlyE.length} | ${g.onlyD.length} | **${g.diff.length}** | ${g.eolOnly.length} |\n`
}
md += `\n合计：仅 E（D 缺失）**${s.onlyE}** 个、仅 D（本地独有）**${s.onlyD}** 个、内容不同**${s.diff}** 个、仅行尾不同**${s.eol}** 个。\n`
md += `\n> ⚠️ 本报告**只读**产出，未修改 E 或 D 任何文件。\n`

for (const g of report.groups) {
  md += `\n---\n\n## ${g.key}（${g.kind}）\n\n`
  if (g.note) md += `> ${g.note}\n\n`
  if (g.skipped) { md += `跳过：E=${g.eState} D=${g.dState}\n`; continue }
  if (g.onlyD.length) {
    md += `### ▶ 仅 D 存在（${g.onlyD.length}）— 覆盖同步将**删除**这些\n\n`
    for (const x of g.onlyD) md += `- \`${x}\`\n`
  }
  if (g.onlyE.length) {
    md += `\n### ◀ 仅 E 存在（${g.onlyE.length}）— D 缺失，同步将**新增**\n\n`
    for (const x of g.onlyE) md += `- \`${x}\`\n`
  }
  if (g.diff.length) {
    md += `\n### ✗ 内容不同（${g.diff.length}）\n\n`
    md += `| # | 文件 | E 大小 | D 大小 | E独有行 | D独有行 | D 独有行示例 |\n|---|---|---|---|---|---|---|\n`
    g.diff.forEach((it, i) => {
      const samp = it.dSamples ? it.dSamples.map(x => x.replace(/\|/g, '\\|')).join(' ⏎ ').slice(0, 120) : '—'
      md += `| ${i + 1} | \`${it.rel}\` | ${it.sizeE} | ${it.sizeD} | ${it.eOnlyLines ?? '—'} | ${it.dOnlyLines ?? '—'} | ${samp} |\n`
    })
  }
  if (g.eolOnly && g.eolOnly.length) {
    md += `\n### ␍ 仅行尾差异（${g.eolOnly.length}）— 内容实质相同（CRLF↔LF），**不建议**为此覆盖\n\n`
    for (const x of g.eolOnly) md += `- \`${x}\`\n`
  }
  if (!g.onlyD.length && !g.onlyE.length && !g.diff.length && !(g.eolOnly || []).length) md += `✓ 完全一致\n`
}

md += `\n---\n\n## 🛡 不在同步范围（用户数据/运行时，勿覆盖）\n\n`
for (const x of PROTECTED) md += `- \`${x}\`\n`
fs.writeFileSync(path.join(OUT, 'diff-report.md'), md, 'utf8')

console.log(`报告已生成：${OUT}`)
console.log(`diff-report.json / diff-report.md`)
console.log(`汇总：仅E=${s.onlyE} 仅D=${s.onlyD} 内容不同=${s.diff} 仅行尾=${s.eol} 跳过组=${s.skipped}`)
for (const g of report.groups) {
  if (g.skipped) { console.log(`  [跳过] ${g.key} (E=${g.eState} D=${g.dState})`); continue }
  console.log(`  ${g.key}: E=${g.eCount} D=${g.dCount} 仅E=${g.onlyE.length} 仅D=${g.onlyD.length} 内容不同=${g.diff.length} 仅行尾=${g.eolOnly.length}`)
}
