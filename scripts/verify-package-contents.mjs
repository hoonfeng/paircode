// ═══════════════════════════════════════════════════════════════
// verify-package-contents.mjs — 发布产物 zip 内容断言（2026-10-17）
//
// 定位：packager pipeline 中 type=package 的「下游守门员」。
// build-go/package 跑完后校验 release/ 下当前版本的三平台 zip，
// 确保测试夹具/依赖/VCS 元数据没有混进发布物（v1.6.18 事故：
// 三平台包各混入 98 个 .pair/plugins/tool-flow/.smoke/**）。
//
// 断言一（禁入内容）：每个 zip 的条目中，以下片段计数必须为 0
//   .smoke/       插件测试夹具（含嵌套 .git 与任务快照）
//   .git/         VCS 元数据
//   node_modules/ 依赖目录
//   .pair/tasks/  运行时任务快照（测试残留，不应出厂）
//
// 断言二（发行版隔离，2026-10-19）：web 包与 gui 包必须各自只含自己的主程序，
// 不允许再出现「一个包里两个 exe」的旧形态：
//   PairCode-web-* : 必须含 pair-server[.exe]，不得含 PairCode[.exe]、不得含 libSkiaSharp*
//   PairCode-gui-* : 必须含 PairCode[.exe]，不得含 pair-server[.exe]
//
// 实现：直接解析 zip 的 EOCD + 中央目录（不依赖 unzip/PowerShell，
// Windows/Linux/macOS 一致；本仓库发布物 < 4GB，无 ZIP64）。
//
// 用法：
//   node scripts/verify-package-contents.mjs              # 校验 packager.json 版本的六包
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

// 发行版隔离规则：must 至少命中一条，forbid 一条都不许命中。
// 注意：zip 条目是「dist 目录内相对路径」（顶层没有 PairCode-web/ 前缀），
// 与旧包（PairCode.exe / pair-server.exe 直接在根）保持一致。
const EDITION_RULES = [
  {
    name: 'web',
    re: /^PairCode-web-.*\.zip$/,
    must: [/^pair-server(\.exe)?$/],
    forbid: [
      { label: 'GUI 主程序', re: /^PairCode(\.exe)?$/ },
      { label: 'Skia 运行库', re: /^libSkiaSharp/ },
    ],
  },
  {
    name: 'gui',
    re: /^PairCode-gui-.*\.zip$/,
    // gui 包 2026-10-19 起改为「目录形态」：主程序 + Skia 运行库 + .pair 插件/
    // 前端资源 + bin 运行时配置缺一不可（单文件自解压 stub 已不再用于发布包，
    // 用户明确要求解压即见完整运行目录，且 GUI 自身起 HTTP/插件服务）。
    must: [
      /^PairCode(\.exe)?$/,
      /^libSkiaSharp/,
      /^\.pair\/plugins\//,
      /^\.pair\/assets\//,
      /^bin\/config\//,
    ],
    forbid: [{ label: '服务端主程序', re: /^pair-server(\.exe)?$/ }],
  },
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
  // web / gui × windows / linux / darwin = 六个独立包（2026-10-19 起）
  const names = ['windows', 'linux', 'darwin'].flatMap((os) => [
    `PairCode-web-${version}-${os}.zip`,
    `PairCode-gui-${version}-${os}.zip`,
  ])
  zips = names.map((n) => path.join(repoRoot, 'release', n))
} else {
  zips = zips.map((p) => (path.isAbsolute(p) ? p : path.join(repoRoot, p)))
}

console.log(`[verify-package-contents] 校验发布产物 zip（${zips.length} 个）…`)

// nativeOnly 版（如 gui：cgo + Skia 只能原生构建）在非本机平台不会产出 zip，
// 这属于预期跳过而非校验失败。
const nativeOnlyEditions = (() => {
  try {
    const cfg = JSON.parse(fs.readFileSync(path.join(repoRoot, 'packager.json'), 'utf8'))
    return new Set((cfg.dist?.editions || []).filter((e) => e.nativeOnly).map((e) => e.name))
  } catch {
    return new Set()
  }
})()
const currentOs = process.platform === 'win32' ? 'windows' : process.platform === 'darwin' ? 'darwin' : 'linux'

for (const zip of zips) {
  const base = path.basename(zip)
  if (!fs.existsSync(zip)) {
    const m = base.match(/^PairCode-([a-z0-9]+)-[\d.]+-(windows|linux|darwin)\.zip$/i)
    if (m && nativeOnlyEditions.has(m[1]) && m[2] !== currentOs) {
      console.log(`  ⏭ ${base}: 跳过（${m[1]} 版 nativeOnly，需在 ${m[2]} 原生构建）`)
      continue
    }
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

  // 发行版隔离断言：web 包不许混入 GUI 主程序/Skia，gui 包不许混入 pair-server。
  const rule = EDITION_RULES.find((r) => r.re.test(base))
  if (rule) {
    const missing = rule.must.filter((re) => !names.some((x) => re.test(x)))
    if (missing.length > 0) {
      errors.push(`${base} 缺少 ${rule.name} 版必需内容（${missing.map((r) => r.source).join(', ')}）`)
      console.log(`  ✘ ${base}: 缺少 ${rule.name} 版必需内容`)
    }
    for (const f of rule.forbid) {
      const hit = names.filter((x) => f.re.test(x))
      if (hit.length > 0) {
        errors.push(`${base} 混入${f.label}: ${hit.slice(0, 3).join(', ')}（${hit.length} 项）`)
        console.log(`  ✘ ${base}: 混入${f.label}（${hit.length} 项）`)
      }
    }
  }
}

if (errors.length > 0) {
  console.error(`\n[verify-package-contents] ✘ 失败（${errors.length} 项）：`)
  for (const e of errors) console.error('  - ' + e)
  process.exit(1)
}
console.log('\n[verify-package-contents] ✔ 全部通过（六包：无禁入内容，且 web/gui 主程序互不混装）')
