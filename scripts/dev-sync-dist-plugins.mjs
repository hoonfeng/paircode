// ═══════════════════════════════════════════════════════════════
// dev-sync-dist-plugins.mjs — 独立发布插件「本地装载挂载」（2026-09-17）
//
// 背景：独立发布插件真源在 plugins-dist/（不进 IDE 发布包），而运行时装载目录
//   固定为 <InstallDir>/.pair/plugins —— exe 位于 bin/ 时 InstallDir() 回退到
//   仓库根（internal/core/settings.go:94），故本地装载目录 = 仓库根 .pair/plugins。
//   本脚本把两者连通，仅服务于**本地开发/验收**。
//
// 三种模式：
//   node scripts/dev-sync-dist-plugins.mjs          # 默认：junction 挂载（改源码即生效）
//   node scripts/dev-sync-dist-plugins.mjs --copy   # 复制副本（junction 不可用时；测完务必 --clean）
//   node scripts/dev-sync-dist-plugins.mjs --clean  # 移除挂载/副本
//
// ★ 为什么不能直接常驻 .pair/plugins：packager.json 的 dist.include 会递归复制
//   .pair/plugins 进发布包 —— 这正是本次迁移的原因。junction 是例外：
//   packager copyDistEntry 对 symlink 一律跳过（main.go:422）→ 挂载态天然不进包；
//   护栏 scripts/verify-dist-isolation.mjs 对此放行，对 --copy 产生的真实副本报错。
//
// 注意：--clean 只删除 symlink/副本，**不删除** plugins-dist 下的真源。
// ═══════════════════════════════════════════════════════════════
import { copyFileSync, existsSync, lstatSync, mkdirSync, readFileSync, readdirSync, rmSync, symlinkSync, unlinkSync, writeFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const distDir = path.join(repoRoot, 'plugins-dist')
const targetDir = path.join(repoRoot, '.pair', 'plugins')

const args = process.argv.slice(2)
const DO_COPY = args.includes('--copy')
const DO_CLEAN = args.includes('--clean')
const DO_FORCE = args.includes('--force')

// ── 本地 git 忽略维护：挂载/副本都落在 .pair/plugins 白名单目录内（.gitignore
//    `!.pair/plugins/` 会让它们以未跟踪项出现，易被误提交）。写到 .git/info/exclude
//    （仅本机、不入库、不改动仓库 .gitignore），由本脚本自管一个 marker 块。
const GIT_EXCLUDE_MARKER = '# paircode：独立发布插件开发挂载/副本（scripts/dev-sync-dist-plugins.mjs 维护；真源在 plugins-dist/）'

function updateGitExclude(add, names) {
  const gitDir = path.join(repoRoot, '.git')
  if (!existsSync(gitDir)) return
  const infoDir = path.join(gitDir, 'info')
  mkdirSync(infoDir, { recursive: true })
  const file = path.join(infoDir, 'exclude')
  let raw = ''
  try { raw = readFileSync(file, 'utf8') } catch { raw = '' }
  // 先摘除旧管理块（marker 行 + 紧随其后的 /.pair/plugins/ 条目）
  const kept = []
  let skipping = false
  for (const line of raw.split(/\r?\n/)) {
    const t = line.trim()
    if (t === GIT_EXCLUDE_MARKER) { skipping = true; continue }
    if (skipping) {
      if (t.startsWith('/.pair/plugins/')) continue
      skipping = false
    }
    kept.push(line)
  }
  while (kept.length > 0 && kept[kept.length - 1].trim() === '') kept.pop()
  if (add && names.length > 0) {
    kept.push('', GIT_EXCLUDE_MARKER)
    for (const n of names) kept.push(`/.pair/plugins/${n}`)
  }
  writeFileSync(file, kept.join('\n') + '\n')
  console.log(`[dev-sync-dist-plugins] .git/info/exclude ${add ? '已登记' : '已移除'} ${names.length} 条独立插件忽略规则（防误提交）`)
}

// pluginNames plugins-dist 下的插件目录名（含 package.json 的目录）。
function pluginNames() {
  if (!existsSync(distDir)) return []
  return readdirSync(distDir, { withFileTypes: true })
    .filter((e) => e.isDirectory() && !e.name.startsWith('.') && e.name !== 'node_modules')
    .map((e) => e.name)
    .filter((n) => existsSync(path.join(distDir, n, 'package.json')))
    .sort()
}

// mirrorCopy 递归复制（跳过 .git；保留 node_modules —— Node 桥插件运行时需要）。
function mirrorCopy(src, dst) {
  mkdirSync(dst, { recursive: true })
  let files = 0
  for (const ent of readdirSync(src, { withFileTypes: true })) {
    if (ent.name === '.git') continue
    const s = path.join(src, ent.name)
    const d = path.join(dst, ent.name)
    let lst
    try { lst = lstatSync(s) } catch { continue }
    if (lst.isDirectory()) { files += mirrorCopy(s, d); continue }
    if (lst.isFile()) { copyFileSync(s, d); files++ }
  }
  return files
}

const names = pluginNames()
if (names.length === 0) {
  console.log('[dev-sync-dist-plugins] plugins-dist 下无插件，无需操作')
  process.exit(0)
}

if (!existsSync(targetDir)) mkdirSync(targetDir, { recursive: true })

let done = 0
for (const name of names) {
  const src = path.join(distDir, name)
  const dst = path.join(targetDir, name)
  let lst = null
  try { lst = lstatSync(dst) } catch { lst = null }

  if (DO_CLEAN) {
    if (!lst) { console.log(`  · ${name}：未挂载，跳过`); continue }
    if (lst.isSymbolicLink()) {
      unlinkSync(dst)
      console.log(`  ✓ ${name}：已移除挂载（symlink）`)
      done++
      continue
    }
    if (DO_FORCE) {
      rmSync(dst, { recursive: true, force: true })
      console.log(`  ✓ ${name}：已删除真实副本（--force）`)
      done++
      continue
    }
    console.error(`  ✗ ${name}：.pair/plugins/${name} 是真实目录，非挂载 —— 为避免误删数据须显式 --force`)
    continue
  }

  if (lst) {
    if (lst.isSymbolicLink()) {
      console.log(`  · ${name}：已挂载（symlink/junction），跳过`)
      continue
    }
    console.error(`  ✗ ${name}：.pair/plugins/${name} 已存在（真实目录）—— 先 --clean --force，或用 --copy 覆盖前手工清理`)
    continue
  }

  if (DO_COPY) {
    const n = mirrorCopy(src, dst)
    console.log(`  ✓ ${name}：已复制副本（${n} 文件；★ 打包前必须 --clean，否则会进发布包）`)
    done++
    continue
  }

  try {
    symlinkSync(src, dst, process.platform === 'win32' ? 'junction' : 'dir')
    console.log(`  ✓ ${name}：已 junction 挂载 → ${path.relative(repoRoot, src)}`)
    done++
  } catch (e) {
    console.error(`  ✗ ${name}：挂载失败（${e.message}）—— 可改用 --copy`)
  }
}

const mode = DO_CLEAN ? '--clean' : DO_COPY ? '--copy' : 'link'
updateGitExclude(!DO_CLEAN, names)
console.log(`[dev-sync-dist-plugins] ${mode} 完成：${done}/${names.length} 个插件（共 ${names.length} 个独立插件）`)
if (!DO_CLEAN && DO_COPY) {
  console.log('[dev-sync-dist-plugins] ⚠ 复制模式会在 .pair/plugins 留下真实副本：打包前请执行 --clean（护栏 verify-dist-isolation.mjs 会拦）')
}
