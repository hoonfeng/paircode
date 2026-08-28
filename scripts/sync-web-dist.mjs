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
import { cpSync, rmSync, readdirSync, readFileSync, existsSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const src = path.join(repoRoot, '.pair', 'assets', 'runtime', 'web')
const dst = path.join(repoRoot, 'cmd', 'companion', 'web-ui', 'dist')

if (!existsSync(src)) {
  console.error(`[sync-web-dist] 源不存在: ${src}（请先运行 plugins-src/ui-app 的 vite build）`)
  process.exit(1)
}
rmSync(dst, { recursive: true, force: true })
cpSync(src, dst, { recursive: true })

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
