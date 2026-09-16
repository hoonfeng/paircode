// ═══════════════════════════════════════════════════════════════
// build-wx-bridge.mjs — 微信桥（wechat-bridge）构建与插件包同步
//
// 职责（对照 docs/wechat-bridge-go-plan.md 交付物④/P5-1）：
//   1. 构建 Go 桥二进制（plugins-src/plugins/wechat-bridge → wxbridge-<os>-<arch>）；
//   2. 拷贝到插件包 bin/（.pair/plugins/wechat-bridge/bin/，Windows 为 wxbridge.exe）；
//   3. --sync：把整个插件包同步到安装目录（<InstallDir>/.pair/plugins/wechat-bridge/，
//      宿主只装载安装目录的全局插件；默认 InstallDir=D:/PairCode，可 --install-dir 覆盖）；
//   4. exe 占用检测（Windows 运行中 exe 无法覆盖——明确提示先停桥）。
//
// 用法（cwd=仓库根）：
//   node scripts/build-wx-bridge.mjs                 # 构建 + 更新仓库插件包 bin/
//   node scripts/build-wx-bridge.mjs --sync          # 再同步插件包到安装目录
//   node scripts/build-wx-bridge.mjs --install-dir "D:/PairCode"
//   node scripts/build-wx-bridge.mjs --skip-build    # 不重编译（仅拷贝/同步现有产物）
// ═══════════════════════════════════════════════════════════════
import { execFileSync } from 'node:child_process'
import { copyFileSync, existsSync, mkdirSync, readdirSync, rmSync, statSync } from 'node:fs'
import path from 'node:path'
import os from 'node:os'
import { fileURLToPath } from 'node:url'

const argv = process.argv.slice(2)
const argOf = (n) => { const i = argv.indexOf(n); return i >= 0 ? argv[i + 1] : null }
const SKIP_BUILD = argv.includes('--skip-build')
const SYNC = argv.includes('--sync')

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const srcDir = path.join(repoRoot, 'plugins-src', 'plugins', 'wechat-bridge')
const pluginDir = path.join(repoRoot, '.pair', 'plugins', 'wechat-bridge')
const binDir = path.join(pluginDir, 'bin')
const isWin = os.platform() === 'win32'
const binName = isWin ? 'wxbridge.exe' : 'wxbridge'
const stagedName = `wxbridge-${os.platform()}-${os.arch}${isWin ? '.exe' : ''}`
const installDir = (argOf('--install-dir') || 'D:/PairCode').replace(/\\/g, '/')

function log(msg) { console.log(`[build-wx-bridge] ${msg}`) }
function fail(msg) { console.error(`[build-wx-bridge] 错误：${msg}`); process.exit(1) }

// ── 1. 构建 ──
const stagedPath = path.join(repoRoot, 'temp', 'wx-bridge-go', stagedName)
if (SKIP_BUILD) {
  if (!existsSync(stagedPath)) fail(`--skip-build 但暂存产物不存在：${stagedPath}`)
  log(`跳过构建（使用现有产物 ${stagedPath}）`)
} else {
  mkdirSync(path.dirname(stagedPath), { recursive: true })
  log(`构建 ${srcDir} → ${stagedPath}`)
  execFileSync('go', ['build', '-o', stagedPath, './plugins-src/plugins/wechat-bridge/'], {
    cwd: repoRoot, stdio: 'inherit',
    env: { ...process.env, CGO_ENABLED: process.env.CGO_ENABLED || '1' },
  })
  log('构建完成')
}

// ── 2. 拷贝到插件包 bin/（占用检测）──
const targetBin = path.join(binDir, binName)
mkdirSync(binDir, { recursive: true })
try {
  copyFileSync(stagedPath, targetBin)
} catch (e) {
  if (e && (e.code === 'EBUSY' || e.code === 'EPERM' || String(e.message).includes('busy'))) {
    fail(`目标被占用（桥进程正在运行？）：${targetBin}\n  → 请先在插件面板停止微信桥（或任务管理器结束 wxbridge.exe）后重试`)
  }
  fail(`拷贝失败：${e.message}`)
}
const size = (statSync(targetBin).size / 1048576).toFixed(1)
log(`已更新插件包二进制：${targetBin}（${size} MB）`)

// ── 3. --sync：同步插件包到安装目录 ──
if (SYNC) {
  const dstDir = path.join(installDir, '.pair', 'plugins', 'wechat-bridge')
  if (!existsSync(path.join(installDir, '.pair', 'plugins'))) {
    fail(`安装目录插件路径不存在：${path.join(installDir, '.pair', 'plugins')}（--install-dir 指定是否正确？）`)
  }
  mkdirSync(dstDir, { recursive: true })
  let n = 0
  const walk = (src, dst) => {
    mkdirSync(dst, { recursive: true })
    for (const ent of readdirSync(src, { withFileTypes: true })) {
      if (ent.name === 'node_modules' || ent.name === '.git') continue
      const s = path.join(src, ent.name)
      const d = path.join(dst, ent.name)
      if (ent.isDirectory()) { walk(s, d); continue }
      try {
        copyFileSync(s, d)
        n++
      } catch (e) {
        if (e && (e.code === 'EBUSY' || e.code === 'EPERM')) {
          fail(`同步被占用：${d}\n  → 桥可能正在运行；请先停止桥再同步`)
        }
        throw e
      }
    }
  }
  walk(pluginDir, dstDir)
  log(`已同步插件包 → ${dstDir}（${n} 个文件）`)
  log('提示：宿主需重启（或在插件面板停用→启用）才会重装插件；改 JS 文件后重启宿主一次即可。')
} else {
  log(`（未同步到安装目录；加 --sync 同步到 ${installDir}/.pair/plugins/wechat-bridge/）`)
}
