// ═══════════════════════════════════════════════════════════════
// verify-dist-isolation.mjs — 独立发布插件「不入 IDE 发布包」护栏（2026-09-17）
//
// 背景：独立发布插件（市场分发，npm 上的 @paircode/<name>）真源从
//   .pair/plugins/ 迁到 **plugins-dist/**。原因：packager.json 的 dist.include
//   含 { src: ".pair/plugins", recursive: true } —— 放在 .pair 内的插件会**随
//   发布包出厂**，与「独立发版、用户按需安装」的定位冲突（体积 + 版本双份锁死）。
//
// 本护栏在打包管线最先执行，断言四件事（任一失败即中止打包）：
//   ① plugins-dist/<name> 的每个插件都在 packager.json
//      dist.include[.pair/plugins].exclude 中（排除清单不残缺）；
//   ② .pair/plugins/<name> 不存在同名**真实副本**（残留 = 仍会进包）；
//      开发态 symlink/junction 挂载放行（packager copyDistEntry 显式跳过 symlink）；
//   ③ packager.json exclude 中的条目都真实存在（无陈旧条目，防删插件后漏改）；
//   ④ plugins-dist 目录形态合法（目录 + package.json）。
//
// 运行：node scripts/verify-dist-isolation.mjs   （cwd 任意，路径以脚本位置推导）
// ═══════════════════════════════════════════════════════════════
import { existsSync, lstatSync, readdirSync, readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const distDir = path.join(repoRoot, 'plugins-dist')
const baselineDir = path.join(repoRoot, '.pair', 'plugins')
const cfgPath = path.join(repoRoot, 'packager.json')
const BASELINE_ENTRY_SRC = '.pair/plugins'

// dirNames 列出目录名（跳过隐藏目录 / node_modules / -src 源码目录）。
// ★ 纳入 symlink/junction（开发挂载）——Windows 上 junction 的 Dirent.isDirectory()
//   可能为 false，纳入后才能在护栏里明确区分「挂载（不进包）」与「真实副本（进包）」。
function dirNames(p) {
  if (!existsSync(p)) return []
  return readdirSync(p, { withFileTypes: true })
    .filter((e) => (e.isDirectory() || e.isSymbolicLink()) && !e.name.startsWith('.') && e.name !== 'node_modules' && !e.name.endsWith('-src'))
    .map((e) => e.name)
    .sort()
}

// packageMap 目录名 → package.json 解析结果（无 package.json 的目录跳过，与装载判定同源）。
function packageMap(base) {
  const out = new Map()
  for (const name of dirNames(base)) {
    const pkgPath = path.join(base, name, 'package.json')
    if (!existsSync(pkgPath)) continue
    let pkg = {}
    try { pkg = JSON.parse(readFileSync(pkgPath, 'utf8')) } catch { pkg = {} }
    out.set(name, pkg)
  }
  return out
}

const distPlugins = packageMap(distDir)
const baselinePlugins = new Set(packageMap(baselineDir).keys())

if (!existsSync(cfgPath)) {
  console.error(`[verify-dist-isolation] 找不到 packager.json: ${cfgPath}`)
  process.exit(1)
}
const cfg = JSON.parse(readFileSync(cfgPath, 'utf8'))
const entry = (cfg?.dist?.include || []).find((e) => e.src === BASELINE_ENTRY_SRC)
if (!entry) {
  console.error(`[verify-dist-isolation] packager.json 的 dist.include 缺少 { src: "${BASELINE_ENTRY_SRC}" } —— 打包基线插件链路已变，请同步本护栏`)
  process.exit(1)
}
const excluded = new Set(entry.exclude || [])

const problems = []
const notes = []

// ① + ②：每个独立插件都必须被 exclude，且基线目录不得有同名副本
for (const name of distPlugins.keys()) {
  const pkg = distPlugins.get(name) || {}
  if (!pkg.name) {
    notes.push(`plugins-dist/${name}/package.json 缺 name 字段（发布脚本会规范化，仅提示）`)
  }
  if (!excluded.has(name)) {
    problems.push(`packager.json → dist.include[${BASELINE_ENTRY_SRC}].exclude 缺少 "${name}"：该独立插件会被打进 IDE 发布包`)
  }
  if (baselinePlugins.has(name)) {
    let lst = null
    try { lst = lstatSync(path.join(baselineDir, name)) } catch { lst = null }
    if (lst && lst.isSymbolicLink()) {
      notes.push(`.pair/plugins/${name} 是开发挂载（symlink/junction）：packager 复制时跳过 symlink，不进发布包 ✓`)
    } else {
      problems.push(`.pair/plugins/${name} 与 plugins-dist/${name} 同名并存（真实副本）：基线目录里的副本仍会进包 —— 开发副本请用 scripts/dev-sync-dist-plugins.mjs --clean 清理`)
    }
  }
}

// ③：exclude 条目必须真实存在（防陈旧条目掩盖真实缺口）
for (const name of excluded) {
  if (!distPlugins.has(name)) {
    problems.push(`packager.json exclude 中的 "${name}" 在 plugins-dist 下不存在：陈旧条目（插件已转正/改名/删除？）`)
  }
}

console.log(`[verify-dist-isolation] 独立发布插件 ${distPlugins.size} 个：${[...distPlugins.keys()].join(', ') || '（无）'}`)
console.log(`[verify-dist-isolation] 基线插件 ${baselinePlugins.size} 个；packager exclude ${excluded.size} 项`)
for (const n of notes) console.log(`[verify-dist-isolation] 提示：${n}`)
if (problems.length > 0) {
  for (const p of problems) console.error(`[verify-dist-isolation] ✗ ${p}`)
  console.error(`[verify-dist-isolation] 失败：${problems.length} 项 —— 打包已中止（独立插件不得进发布包）`)
  process.exit(1)
}
console.log('[verify-dist-isolation] PASS：独立发布插件与 IDE 发布包已隔离')
