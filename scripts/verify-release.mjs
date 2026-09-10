// ═══════════════════════════════════════════════════════════════
// verify-release.mjs — 发布输入完整性校验（2026-09-12）
//
// 定位：packager pipeline 中 build-go / package 之前的「守门员」。
// 校验“完整发布”所需的全部输入就位，任何硬缺失直接 fail（exit 1），
// 避免打出残包（历史教训：壳 idx 与资产不同步、bin 镜像缺插件导致
// 发布包插件面残缺、UI 区域产物缺失等）。
//
// 校验项：
//   1. packager.json dist.include：必需项（optional!=true）全部存在；
//      可选项缺失列出提示
//   2. 插件基座：.pair/plugins 每个插件含 index.js + package.json；
//      含 dsh.ui.build 的 UI 区域插件产物（outDir/fileName.js）存在
//   3. 前端：.pair/assets/runtime/web/index.html 存在且引用 assets/*.js；
//      cmd/companion/web-ui/dist（//go:embed 兜底）index.html 存在
//   4. JS 运行时基座：bridge_node.js / cordis.bundle.js 存在且非空
//   5. 死内容守卫：config/philosophy 若存在仅警告（已移出发布清单）
//
// 注：插件自 2026-08-27 起全量 JS 化（.pair/plugins/<name>/index.js），
// 宿主执行走内嵌 Go 内核，不再编译独立二进制（旧脚本 build-plugin-bins.bat
// 已废弃、管线步骤已移除，禁止恢复）。
//
// 运行：node scripts/verify-release.mjs（cwd=仓库根）
// ═══════════════════════════════════════════════════════════════
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const errors = []
const warns = []
const ok = (m) => console.log('  ✔ ' + m)
const bad = (m) => { errors.push(m); console.log('  ✘ ' + m) }
const warn = (m) => { warns.push(m); console.log('  ⚠ ' + m) }

console.log('[verify-release] 校验发布输入完整性…')

// ── 1) packager.json dist.include ──────────────────────────────
let cfg
try {
  cfg = JSON.parse(fs.readFileSync(path.join(repoRoot, 'packager.json'), 'utf8'))
} catch (e) {
  bad(`packager.json 无法解析: ${e.message}`)
}
let optMissing = 0
if (cfg) {
  const include = (cfg.dist && cfg.dist.include) || []
  for (const e of include) {
    if (fs.existsSync(path.join(repoRoot, e.src))) continue
    if (e.optional) { optMissing++; continue }
    bad(`dist.include 必需项缺失: ${e.src}`)
  }
  ok(`dist.include 检查完成（${include.length} 条；可选缺失 ${optMissing} 条，已跳过）`)
}

// ── 2) 插件基座 ────────────────────────────────────────────────
const pluginsDir = path.join(repoRoot, '.pair', 'plugins')
let pluginCount = 0
if (!fs.existsSync(pluginsDir)) {
  bad('.pair/plugins 不存在')
} else {
  for (const name of fs.readdirSync(pluginsDir).sort()) {
    const dir = path.join(pluginsDir, name)
    let lst
    try { lst = fs.statSync(dir) } catch { continue }
    if (!lst.isDirectory()) continue
    pluginCount++
    const idx = path.join(dir, 'index.js')
    const pkgPath = path.join(dir, 'package.json')
    if (!fs.existsSync(idx)) bad(`插件 ${name} 缺 index.js`)
    if (!fs.existsSync(pkgPath)) { bad(`插件 ${name} 缺 package.json`); continue }
    try {
      const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'))
      const b = pkg.dsh && pkg.dsh.ui && pkg.dsh.ui.build
      if (b && b.outDir && b.fileName) {
        const js = path.join(repoRoot, b.outDir, b.fileName + '.js')
        if (!fs.existsSync(js)) bad(`UI 区域插件 ${name} 构建产物缺失: ${b.outDir}/${b.fileName}.js（先运行 node scripts/build-ui.mjs）`)
      }
    } catch (e) {
      bad(`插件 ${name} package.json 解析失败: ${e.message}`)
    }
  }
  ok(`插件基座检查完成（${pluginCount} 个插件）`)
}

// ── 3) 前端壳（外置 + embed 兜底）─────────────────────────────
const webIndex = path.join(repoRoot, '.pair', 'assets', 'runtime', 'web', 'index.html')
if (!fs.existsSync(webIndex)) {
  bad('外置壳缺失: .pair/assets/runtime/web/index.html（先运行壳构建）')
} else {
  const html = fs.readFileSync(webIndex, 'utf8')
  const refs = [...html.matchAll(/assets\/[A-Za-z0-9._-]+\.js/g)].map((m) => m[0])
  if (refs.length === 0) bad('外置壳 index.html 未引用任何 assets/*.js')
  else ok(`外置壳 index.html 正常（引用 ${refs.length} 个 bundle）`)
}
const embedIndex = path.join(repoRoot, 'cmd', 'companion', 'web-ui', 'dist', 'index.html')
if (!fs.existsSync(embedIndex)) {
  bad('embed 兜底壳缺失: cmd/companion/web-ui/dist/index.html（先运行 node scripts/sync-web-dist.mjs）')
} else {
  ok('embed 兜底壳存在（cmd/companion/web-ui/dist）')
}

// ── 4) JS 运行时基座 ──────────────────────────────────────────
for (const f of ['bridge_node.js', 'cordis.bundle.js']) {
  const p = path.join(repoRoot, '.pair', 'assets', 'runtime', f)
  if (!fs.existsSync(p)) { bad(`运行时基座缺失: .pair/assets/runtime/${f}`); continue }
  const sz = fs.statSync(p).size
  if (sz === 0) bad(`运行时基座为空文件: ${f}`)
  else ok(`运行时基座 ${f} 正常（${(sz / 1024).toFixed(0)} KB）`)
}

// ── 5) 死内容守卫 ─────────────────────────────────────────────
if (fs.existsSync(path.join(repoRoot, 'config', 'philosophy'))) {
  warn('config/philosophy 仍存在（已从发布清单移除，建议清理开发树）')
}

// ── 汇总 ──────────────────────────────────────────────────────
if (errors.length > 0) {
  console.error(`\n[verify-release] ✘ 校验失败（${errors.length} 项硬错误）：`)
  for (const e of errors) console.error('  - ' + e)
  process.exit(1)
}
console.log(`\n[verify-release] ✔ 全部通过${warns.length ? `（${warns.length} 条警告）` : ''}`)
