// ═══════════════════════════════════════════════════════════════
// publish-official-plugins.mjs — PairCode 官方插件发布脚本（2026-08-20）
//
// 把 .pair/plugins/ 下磁盘插件发布为 npm 官方包 @paircode/<name>：
//   - 命名约定对齐市场 searchNpmPlugins：@paircode/<name>（权威前缀）
//   - 整目录打包（index.js + client.js + assets/ + bin/ + package.json），
//     安装端 npmMarketInstall 拉 tarball → 固化为磁盘插件（目录名=去 scope）
//
// 用法（cwd=仓库根）：
//   node scripts/publish-official-plugins.mjs               # 打包 + npm pack 验证（默认）
//   node scripts/publish-official-plugins.mjs --publish     # 真实发布（需 npm 登录 + 2FA 就绪）
//   node scripts/publish-official-plugins.mjs --only marketplace,tool-git   # 只发指定插件
//   node scripts/publish-official-plugins.mjs --check       # 只检查现有 npm 包版本（不打包）
//
// ★ 2FA：账号开启 2FA 时真实发布需一次性 OTP：
//   node ... --publish --otp 123456
// 或改用 bypass-2FA 的 granular token（写入 .npmrc 后无需 --otp）。
// ═══════════════════════════════════════════════════════════════
import fs from 'node:fs'
import path from 'node:path'
import { execSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const pluginsDir = path.join(root, '.pair', 'plugins')
const publishDir = path.join(root, '.pair', 'publish')
const REGISTRY = process.env.PAIRCODE_NPM_REGISTRY || 'https://registry.npmjs.org'

const args = process.argv.slice(2)
const DO_PUBLISH = args.includes('--publish')
const DO_CHECK = args.includes('--check')
const otpIdx = args.indexOf('--otp')
const OTP = otpIdx >= 0 ? args[otpIdx + 1] : ''
const onlyIdx = args.indexOf('--only')
const ONLY = onlyIdx >= 0 ? args[onlyIdx + 1].split(',').map((s) => s.trim()).filter(Boolean) : null

// 拷贝文件/目录（白名单：只拷发布所需，排除 node_modules/.git 等）
const PUBLISH_FILES = ['index.js', 'client.js', 'assets', 'bin', 'package.json', 'README.md']
function copyDir(src, dst) {
  fs.mkdirSync(dst, { recursive: true })
  for (const ent of fs.readdirSync(src, { withFileTypes: true })) {
    if (ent.name === 'node_modules' || ent.name === '.git') continue
    if (!PUBLISH_FILES.includes(ent.name)) continue
    const s = path.join(src, ent.name)
    const d = path.join(dst, ent.name)
    if (ent.isDirectory()) copyDir(s, d)
    else fs.copyFileSync(s, d)
  }
}

// 插件目录清单（跳过空目录 = 已废弃的 market-mcp/market-plugin/market-skill）
function listPlugins() {
  const out = []
  for (const ent of fs.readdirSync(pluginsDir, { withFileTypes: true })) {
    if (!ent.isDirectory()) continue
    const dir = path.join(pluginsDir, ent.name)
    const pkgPath = path.join(dir, 'package.json')
    if (!fs.existsSync(path.join(dir, 'index.js')) && !fs.existsSync(pkgPath)) continue
    let pkg = {}
    if (fs.existsSync(pkgPath)) {
      try { pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8')) } catch { pkg = {} }
    }
    out.push({ name: ent.name, dir, pkg })
  }
  return out.sort((a, b) => a.name.localeCompare(b.name))
}

// 检查 npm 上是否已存在（返回已发布版本或 null）
function npmExists(pkgName) {
  // 用 dist-tags 端点检测（registry 对新包 metadata 可能 404，但 dist-tags 立即可查）
  try {
    const out = execSync(`curl -s -w "\\n%{http_code}" "${REGISTRY}/-/package/${pkgName}/dist-tags"`, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] })
    const lines = out.trim().split('\n')
    const code = lines[lines.length - 1]
    if (code !== '200') return null
    try { return JSON.parse(lines.slice(0, -1).join('\n')).latest || null } catch { return null }
  } catch { return null }
}

function main() {
  const plugins = listPlugins().filter((p) => !ONLY || ONLY.includes(p.name))
  console.log(`══ PairCode 官方插件发布 ══ registry=${REGISTRY} mode=${DO_PUBLISH ? 'PUBLISH' : 'pack-验证'}`)
  if (ONLY) console.log(`精选: ${ONLY.join(', ')}`)
  console.log(`待处理插件 ${plugins.length} 个：`)
  for (const p of plugins) console.log(`  - ${p.name}`)

  if (DO_CHECK) {
    console.log('\n── 现有 npm 版本检查 ──')
    for (const p of plugins) {
      const full = `@paircode/${p.name}`
      const ver = npmExists(full)
      console.log(`  ${ver ? `@paircode/${p.name}@${ver}（已存在）` : `${full}（未发布）`}`)
    }
    return
  }

  const ok = []
  const fail = []
  for (const p of plugins) {
    const pkgName = `@paircode/${p.name}`
    const dst = path.join(publishDir, p.name)
    // 已存在则跳过（避免 403 cannot publish over versions）
    const existing = npmExists(pkgName)
    if (existing) {
      console.log(`  ⏭ ${pkgName} 已存在（@${existing}），跳过`)
      ok.push({ name: p.name, pkgName, out: `skip@${existing}` })
      continue
    }
    try {
      // 1. 整目录拷贝（保留 index.js/client.js/assets/bin/）
      copyDir(p.dir, dst)
      // 2. package.json 改造为 npm 官方包
      const pkg = { ...p.pkg, name: pkgName }
      pkg.description = pkg.purpose || pkg.description || `PairCode 官方插件 ${p.name}`
      delete pkg.purpose
      pkg.keywords = ['paircode']
      pkg.license = pkg.license || 'MIT'
      pkg.publishConfig = { access: 'public' }
      pkg.files = ['index.js', 'client.js', 'assets', 'bin', 'package.json']
      fs.writeFileSync(path.join(dst, 'package.json'), JSON.stringify(pkg, null, 2))
      // 3. 验证/发布
      const cmd = DO_PUBLISH
        ? `npm publish ${path.join(dst, 'package.json').replace(/package\.json$/, '').replace(/\\/g, '/')} --registry=${REGISTRY} --access public${OTP ? ` --otp=${OTP}` : ''}`
        : `npm pack ${path.join(dst, 'package.json').replace(/package\.json$/, '').replace(/\\/g, '/')} --dry-run --json`
      const out = execSync(cmd, { encoding: 'utf8', cwd: dst, stdio: ['ignore', 'pipe', 'ignore'] })
      const size = fs.statSync(path.join(dst, 'package.json')).size
      ok.push({ name: p.name, pkgName, out })
      console.log(`  ✅ ${pkgName} ${DO_PUBLISH ? '已发布' : '打包验证通过'}（${pkg.version}）`)
    } catch (e) {
      fail.push({ name: p.name, pkgName, err: String(e.stderr || e.message).split('\n').slice(0, 4).join(' | ') })
      console.log(`  ❌ ${pkgName}: ${fail[fail.length - 1].err}`)
    }
  }

  console.log(`\n══ 结果：成功 ${ok.length} / 失败 ${fail.length} ══`)
  if (fail.length) {
    console.log('失败明细：')
    for (const f of fail) console.log(`  - ${f.pkgName}: ${f.err}`)
    process.exitCode = 1
  }
}

main()
