// ═══════════════════════════════════════════════════════════════
// sync-plugins-to-bin.mjs — 安装根 .pair → bin/.pair 镜像同步（2026-09-12）
//
// 背景：bin/.pair 是旧版「从 bin 启动」部署留下的数据目录；启动时
// core.MigrateLegacyBinPairData 会把它「只补缺」迁移到安装根 —— 若
// bin/.pair 残留旧插件/旧工具集/旧资产，会在启动时复活污染安装根
// （2026 教训：tool-debug 磁盘插件曾因此复活）。
//
// 本脚本保证 bin/.pair 的 plugins / toolsets / assets 三个子树与安装根
// 完全一致（镜像语义：先删后拷），替换 packager 旧管线中 windows-only
// 的 robocopy 步骤（跨平台一致）。
//
// 跳过规则（与 MigrateLegacyBinPairData / packager copyDistEntry 对齐）：
//   - node_modules、.git  目录不镜像
//   - 符号链接 / junction  不镜像（无文件内容，防越界）
//
// 运行：node scripts/sync-plugins-to-bin.mjs（cwd=仓库根）
// ═══════════════════════════════════════════════════════════════
import { copyFileSync, mkdirSync, readdirSync, rmSync, existsSync, lstatSync, statSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const srcRoot = path.join(repoRoot, '.pair')
const dstRoot = path.join(repoRoot, 'bin', '.pair')

// mirrorDir 递归镜像目录（跳过 node_modules/.git/符号链接），返回拷贝文件数。
function mirrorDir(src, dst) {
  mkdirSync(dst, { recursive: true })
  let files = 0
  for (const ent of readdirSync(src, { withFileTypes: true })) {
    if (ent.name === 'node_modules' || ent.name === '.git') continue
    const s = path.join(src, ent.name)
    const d = path.join(dst, ent.name)
    let lst
    try { lst = lstatSync(s) } catch { continue }
    if (lst.isSymbolicLink()) continue
    if (lst.isDirectory()) { files += mirrorDir(s, d); continue }
    if (lst.isFile()) { copyFileSync(s, d); files++ }
  }
  return files
}

const SUBS = ['plugins', 'toolsets', 'assets']
let synced = 0
for (const sub of SUBS) {
  const src = path.join(srcRoot, sub)
  const dst = path.join(dstRoot, sub)
  if (!existsSync(src) || !statSync(src).isDirectory()) {
    console.log(`[sync-plugins-to-bin] 跳过（源不存在）: .pair/${sub}`)
    continue
  }
  rmSync(dst, { recursive: true, force: true })
  const n = mirrorDir(src, dst)
  synced++
  console.log(`[sync-plugins-to-bin] 已镜像 .pair/${sub} → bin/.pair/${sub}（${n} 个文件）`)
}
if (synced === 0) {
  console.error('[sync-plugins-to-bin] 错误：无任何目录被同步')
  process.exit(1)
}
console.log(`[sync-plugins-to-bin] 完成（${synced} 个目录）`)
