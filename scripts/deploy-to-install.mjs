// 部署仓库产物到 PairCode「安装目录」（新 pair.exe + agentloop 插件）
//
// 为什么需要这个脚本
// ------------------
// 安装版运行实例（默认端口 9090 的 pair.exe）加载的是：
//     ① <安装目录>/pair.exe
//     ② <安装目录>/.pair/plugins/**
// 仓库里的改动**不会自动进入安装版**（仓库 .pair/plugins 是开发运行时真源，
// 安装目录是独立副本）——改完插件/内核后界面看不到新配置项，通常是这个原因
// （2026-09-12「双闸门设置项不可见」即由此而起）。
//
// 用法
// ----
//     node scripts/deploy-to-install.mjs                        # dry-run：只打印计划，不写任何文件
//     node scripts/deploy-to-install.mjs --apply                # 部署 exe + agentloop 插件（自动备份）
//     node scripts/deploy-to-install.mjs --apply --only-plugin  # 只同步插件（免关程序，插件的 JS 不被锁定）
//     node scripts/deploy-to-install.mjs --apply --install-dir "D:/PairCode"
//
// ★ 换 exe 必须先关闭 PairCode（Windows 不允许覆盖运行中的 exe）；
//   插件文件可在运行中覆盖，但需重启 PairCode（或在插件面板对 agentloop stop → start）才生效。
import fs from 'node:fs'
import path from 'node:path'
import os from 'node:os'
import crypto from 'node:crypto'
import { execFileSync } from 'node:child_process'

const argv = process.argv.slice(2)
const APPLY = argv.includes('--apply')
const ONLY_PLUGIN = argv.includes('--only-plugin')
const argOf = (n) => { const i = argv.indexOf(n); return i >= 0 ? argv[i + 1] : null }

const ROOT = process.cwd()
const SRC_EXE = argOf('--exe') || path.join(ROOT, 'bin', 'pair.exe')
const SRC_PLUGIN = path.join(ROOT, '.pair', 'plugins', 'agentloop')

const CANDIDATES = [
  'E:/Program Files (x86)/PairCode',
  'E:/Program Files/PairCode',
  'C:/Program Files (x86)/PairCode',
  'C:/Program Files/PairCode',
  path.join(os.homedir(), 'PairCode'),
]

const md5 = (p) => crypto.createHash('md5').update(fs.readFileSync(p)).digest('hex')
const mb = (n) => (n / 1048576).toFixed(1) + ' MB'

function detectInstallDir() {
  const explicit = argOf('--install-dir')
  if (explicit) return explicit.replace(/\\/g, '/')
  for (const c of CANDIDATES) {
    try {
      if (fs.existsSync(path.join(c, 'pair.exe')) &&
          fs.existsSync(path.join(c, '.pair', 'plugins', 'agentloop', 'index.js'))) return c
    } catch { /* 权限/不存在：跳过 */ }
  }
  return null
}

function copyTree(src, dst) {
  const st = fs.statSync(src)
  if (st.isDirectory()) {
    fs.mkdirSync(dst, { recursive: true })
    for (const e of fs.readdirSync(src)) copyTree(path.join(src, e), path.join(dst, e))
  } else {
    fs.mkdirSync(path.dirname(dst), { recursive: true })
    fs.copyFileSync(src, dst)
  }
}

function listListeners(port) {
  try {
    const out = execFileSync('netstat', ['-ano'], { encoding: 'utf8' })
    return out.split(/\r?\n/)
      .filter((l) => l.includes(`:${port}`) && /LISTENING/i.test(l))
      .map((l) => l.trim().split(/\s+/).pop())
      .filter((v, i, a) => a.indexOf(v) === i)
  } catch { return [] }
}

const install = detectInstallDir()
console.log('══ 部署计划（仓库 → 安装目录）══')
console.log('模式          :', APPLY ? (ONLY_PLUGIN ? 'APPLY（仅插件）' : 'APPLY（exe + 插件）') : 'DRY-RUN（不写文件）')
console.log('仓库根        :', ROOT)
console.log('安装目录      :', install || '❌ 未探测到（请用 --install-dir 指定）')
if (!install) process.exit(1)

const dstExe = path.join(install, 'pair.exe')
const dstPlugin = path.join(install, '.pair', 'plugins', 'agentloop')

// ── 计划 ──
const plan = []
const stepExe = { what: 'pair.exe', src: SRC_EXE, dst: dstExe, needed: false, note: '' }
try {
  if (!fs.existsSync(SRC_EXE)) { stepExe.note = '源不存在（先构建：go build -o bin/pair.exe ./cmd/companion）' }
  else if (!fs.existsSync(dstExe)) { stepExe.needed = true; stepExe.note = '目标缺失' }
  else {
    const a = md5(SRC_EXE), b = md5(dstExe)
    stepExe.needed = a !== b
    stepExe.note = stepExe.needed ? `md5 不同（仓库 ${a.slice(0, 8)} / 安装 ${b.slice(0, 8)}，${mb(fs.statSync(SRC_EXE).size)}）` : 'md5 相同，已是最新'
  }
} catch (e) { stepExe.note = '读取失败：' + e.message }
plan.push(stepExe)

const stepPlugin = { what: 'agentloop 插件', src: SRC_PLUGIN, dst: dstPlugin, needed: false, note: '' }
try {
  const a = path.join(SRC_PLUGIN, 'index.js')
  const b = path.join(dstPlugin, 'index.js')
  if (!fs.existsSync(a)) { stepPlugin.note = '源不存在' }
  else if (!fs.existsSync(b)) { stepPlugin.needed = true; stepPlugin.note = '目标缺失' }
  else {
    const ma = md5(a), mb2 = md5(b)
    stepPlugin.needed = ma !== mb2
    stepPlugin.note = stepPlugin.needed ? `md5 不同（仓库 ${ma.slice(0, 8)} / 安装 ${mb2.slice(0, 8)}）` : 'md5 相同，已是最新'
  }
} catch (e) { stepPlugin.note = '读取失败：' + e.message }
plan.push(stepPlugin)

for (const p of plan) console.log(`  · ${p.what.padEnd(16)} ${p.needed ? '需同步' : '跳过  '}  ${p.note}`)

const running = listListeners(9090)
console.log('9090 运行实例 :', running.length ? `运行中（PID ${running.join(',')}）` : '未运行')

if (!APPLY) {
  console.log('\n（DRY-RUN 结束，未写入任何文件；确认后加 --apply）')
  process.exit(0)
}

// ── 备份 ──
const ts = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
const bak = path.join(ROOT, '_temp', `install-backup-${ts}`)
fs.mkdirSync(bak, { recursive: true })
let backed = 0
if (fs.existsSync(dstExe)) { copyTree(dstExe, path.join(bak, 'pair.exe')); backed++ }
if (fs.existsSync(dstPlugin)) { copyTree(dstPlugin, path.join(bak, '.pair', 'plugins', 'agentloop')); backed++ }
console.log(`\n已备份 ${backed} 项 → ${path.relative(ROOT, bak)}`)

// ── 同步 ──
let failed = 0
if (!ONLY_PLUGIN && stepExe.needed) {
  try { fs.copyFileSync(SRC_EXE, dstExe); console.log('✓ pair.exe 已覆盖') }
  catch (e) {
    failed++
    console.log(`✗ pair.exe 覆盖失败：${e.message}`)
    console.log('  → exe 正在运行时不允覆盖：先关闭 PairCode 再执行；或改用 --only-plugin（只同步插件）')
  }
}
if (stepPlugin.needed) {
  try { copyTree(SRC_PLUGIN, dstPlugin); console.log('✓ agentloop 插件已覆盖') }
  catch (e) { failed++; console.log(`✗ agentloop 插件覆盖失败：${e.message}`) }
}

console.log('\n══ 生效方式 ══')
console.log('  · 关闭并重新打开 PairCode（换 exe 必须；插件改动也需重启装配）')
console.log('  · 或在插件面板对 agentloop 执行 stop → start（仅插件改动时可免重启）')
console.log('  · 验证：curl -s http://127.0.0.1:9090/api/settings | 看 schemas[agentloop].fields 是否含 stepBudget')
process.exit(failed ? 1 : 0)
