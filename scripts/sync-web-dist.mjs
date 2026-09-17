// ═══════════════════════════════════════════════════════════════
// sync-web-dist.mjs — 前端壳产物同步到 embed 兜底目录（2026-08-29）
//
// 背景：vite 壳构建输出到 .pair/assets/runtime/web（宿主外部优先加载，
// 改 UI 无需重编译 Go）；cmd/companion/web-ui/dist 为 //go:embed 兜底
// （单文件分发）。本脚本把壳产物镜像到 dist，并清理未被 index.html
// 引用的历史 index-*.js bundle（减小 embed 体积）。
//
// 运行：node scripts/sync-web-dist.mjs（cwd=仓库根）
// ═══════════════════════════════════════════════════════════════
import { copyFileSync, mkdirSync, rmSync, readdirSync, readFileSync, existsSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const src = path.join(repoRoot, '.pair', 'assets', 'runtime', 'web')
const dst = path.join(repoRoot, 'cmd', 'companion', 'web-ui', 'dist')

if (!existsSync(src)) {
  console.error(`[sync-web-dist] 源不存在: ${src}（请先运行 plugins-src/ui-app 的 vite build）`)
  process.exit(1)
}
// ★ 2026-09-17：不用 fs.cpSync(recursive)——本机 Node v24.14.0 + Windows 下触发 native 层静默
//   崩溃（exit 127：无异常可捕获、无 exit 事件，进程直接被终止；连两个小文件的目录递归拷也崩，
//   已实测复现）。改为逐文件递归拷贝（copyFileSync/mkdirSync 实测稳定）。
//   dst 为 repoRoot 派生常量；先校验未越过仓库根（防误删）再 rmSync。
if (path.relative(repoRoot, dst).startsWith('..')) {
  console.error(`[sync-web-dist] 非法 dst（在仓库外）: ${dst}`)
  process.exit(1)
}
rmSync(dst, { recursive: true, force: true })
function copyTree(from, to) {
  try {
    mkdirSync(to, { recursive: true })
    for (const entry of readdirSync(from, { withFileTypes: true })) {
      const sp = path.join(from, entry.name)
      const dp = path.join(to, entry.name)
      if (entry.isSymbolicLink()) { console.warn(`[sync-web-dist] 跳过符号链接: ${sp}`); continue }
      if (entry.isDirectory()) copyTree(sp, dp)
      else copyFileSync(sp, dp)
    }
  } catch (e) {
    console.error(`[sync-web-dist] 拷贝失败 ${from} → ${to}: ${e.message}`)
    process.exit(1)
  }
}
copyTree(src, dst)

// 清理未被引用的历史 bundle
const indexPath = path.join(dst, 'index.html')
if (existsSync(indexPath)) {
  const html = readFileSync(indexPath, 'utf8')
  const refs = new Set([...html.matchAll(/index-[A-Za-z0-9_-]+\.js/g)].map((m) => m[0]))
  const assetsDir = path.join(dst, 'assets')
  if (existsSync(assetsDir)) {
    let removed = 0
    for (const f of readdirSync(assetsDir)) {
      if (/^index-.*\.js$/.test(f) && !refs.has(f)) {
        rmSync(path.join(assetsDir, f), { force: true })
        removed++
      }
    }
    if (removed > 0) console.log(`[sync-web-dist] 清理 ${removed} 个未引用历史 bundle`)
  }
}
console.log(`[sync-web-dist] 已同步 ${src} → ${dst}`)
