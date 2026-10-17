// ═══════════════════════════════════════════════════════════════
// verify-package-contents.mjs — 发布产物 zip 内容断言（2026-10-17）
//
// 定位：packager pipeline 中 type=package 的「下游守门员」。
// build-go/package 跑完后校验 release/ 下当前版本的三平台 zip，
// 确保测试夹具/依赖/VCS 元数据没有混进发布物（v1.6.18 事故：
// 三平台包各混入 98 个 .pair/plugins/tool-flow/.smoke/**）。
//
// 断言：每个 zip 的条目中，以下片段计数必须为 0
//   .smoke/       插件测试夹具（含嵌套 .git 与任务快照）
//   .git/         VCS 元数据
//   node_modules/ 依赖目录
//   .pair/tasks/  运行时任务快照（测试残留，不应出厂）
//
// 实现：直接解析 zip 的 EOCD + 中央目录（不依赖 unzip/PowerShell，
// Windows/Linux/macOS 一致；本仓库发布物 < 4GB，无 ZIP64）。
//
// 用法：
//   node scripts/verify-package-contents.mjs              # 校验 packager.json 版本的三包
//   node scripts/verify-package-contents.mjs --version 1.6.20
//   node scripts/verify-package-contents.mjs release/a.zip release/b.zip
// ═══════════════════════════════════════════════════════════════
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')

const FORBIDDEN = [
  { label: '.smoke', re: /(^|\/)\.smoke(\/|$)/ },
  { label: '.git', re: /(^|\/)\.git(\/|$)/ },
  { label: 'node_modules', re: /(^|\/)node_modules(\/|$)/ },
  { label: '.pair/tasks', re: /(^|\/)\.pair\/tasks(\/|$)/ },
]

// 读 zip 条目名：EOCD → 中央目录
function readZipEntryNames(buf) {
  const minEocd = 22
  if (buf.length < minEocd) throw new Error('文件过小，不是有效 zip')
  const scanStart = Math.max(0, buf.length - 65557 - minEocd)
  let eocd = -1
  for (let i = buf.length - minEocd; i >= scanStart; i--) {
    if (buf.readUInt32LE(i) === 0x06054b50) { eocd = i; break }
  }
  if (eocd < 0) throw new Error('未找到 EOCD（不是 zip 或文件损坏）')
  const total = buf.readUInt16LE(eocd + 10)
  const cdOff = buf.readUInt32LE(eocd + 16)
  if (total === 0xffff || cdOff === 0xffffffff) {
    throw new Error('ZIP64 中央目录，超出本脚本支持范围')
  }
  const names = []
  let p = cdOff
  for (let i = 0; i < total; i++) {
    if (p + 46 > buf.length || buf.readUInt32LE(p) !== 0x02014b50) {
      throw new Error(`中央目录头损坏（第 ${i + 1} 条，offset=${p}）`)
    }
    const nameLen = buf.readUInt16LE(p + 28)
    const extraLen = buf.readUInt16LE(p + 30)
    const commentLen = buf.readUInt16LE(p + 32)
    names.push(buf.toString('utf8', p + 46, p + 46 + nameLen))
    p += 46 + nameLen + extraLen + commentLen
  }
  return names
}

// ── 解析参数：--version / 显式 zip 路径 ────────────────────────
const argv = process.argv.slice(2)
const errors = []
let version = ''
const explicit = []
for (let i = 0; i < argv.length; i++) {
  if (argv[i] === '--version') version = argv[++i]
  else explicit.push(argv[i])
}

let zips = explicit
if (zips.length === 0) {
  if (!version) {
    try {
      version = JSON.parse(fs.readFileSync(path.join(repoRoot, 'packager.json'), 'utf8')).version
    } catch (e) {
      console.error(`[verify-package-contents] ✘ 无法读取 packager.json: ${e.message}`)
      process.exit(1)
    }
  }
  const names = [`PairCode-${version}.zip`, `PairCode-linux-${version}.zip`, `PairCode-darwin-${version}.zip`]
  zips = names.map((n) => path.join(repoRoot, 'release', n))
} else {
  zips = zips.map((p) => (path.isAbsolute(p) ? p : path.join(repoRoot, p)))
}

console.log(`[verify-package-contents] 校验发布产物 zip（${zips.length} 个）…`)

for (const zip of zips) {
  const base = path.basename(zip)
  if (!fs.existsSync(zip)) {
    errors.push(`${base} 不存在（packager package 阶段未产出？）`)
    console.log(`  ✘ ${base}: 文件不存在`)
    continue
  }
  let names
  try {
    names = readZipEntryNames(fs.readFileSync(zip))
  } catch (e) {
    errors.push(`${base} 解析失败: ${e.message}`)
    console.log(`  ✘ ${base}: 解析失败（${e.message}）`)
    continue
  }
  const counts = FORBIDDEN.map(({ label, re }) => ({ label, n: names.filter((x) => re.test(x)).length }))
  const badOnes = counts.filter((c) => c.n > 0)
  const summary = counts.map((c) => `${c.label}=${c.n}`).join(' ')
  if (badOnes.length > 0) {
    errors.push(`${base} 含禁入内容: ${badOnes.map((c) => `${c.label}(${c.n})`).join(', ')}`)
    console.log(`  ✘ ${base}: ${names.length} 条目 | ${summary}`)
    const sample = names.filter((x) => FORBIDDEN.some(({ re }) => re.test(x))).slice(0, 5)
    for (const s of sample) console.log(`      样本: ${s}`)
  } else {
    console.log(`  ✔ ${base}: ${names.length} 条目 | ${summary}`)
  }
}

if (errors.length > 0) {
  console.error(`\n[verify-package-contents] ✘ 失败（${errors.length} 项）：`)
  for (const e of errors) console.error('  - ' + e)
  process.exit(1)
}
console.log('\n[verify-package-contents] ✔ 全部通过（三平台包均无测试夹具/依赖/VCS 元数据）')
