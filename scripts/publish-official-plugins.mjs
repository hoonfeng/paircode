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
import crypto from 'node:crypto'
import { execSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const pluginsDir = path.join(root, '.pair', 'plugins')
const publishDir = path.join(root, '.pair', 'publish')
const REGISTRY = process.env.PAIRCODE_NPM_REGISTRY || 'https://registry.npmjs.org'

// ── 代理配置：PAIRCODE_PROXY → HTTPS_PROXY → HTTP_PROXY ──
const PROXY = process.env.PAIRCODE_PROXY || process.env.HTTPS_PROXY || process.env.HTTP_PROXY || ''
const proxyArgs = PROXY ? ` --proxy=${PROXY} --https-proxy=${PROXY}` : ''

const args = process.argv.slice(2)
const DO_PUBLISH = args.includes('--publish')
const DO_CHECK = args.includes('--check')
const otpIdx = args.indexOf('--otp')
const OTP = otpIdx >= 0 ? args[otpIdx + 1] : ''
const onlyIdx = args.indexOf('--only')
const ONLY = onlyIdx >= 0 ? args[onlyIdx + 1].split(',').map((s) => s.trim()).filter(Boolean) : null

// 拷贝文件/目录（白名单：只拷发布所需，排除 node_modules/.git 等）
// ★ 2026-08-20 修复：topLevel 只在顶层做 PUBLISH_FILES 过滤；
//   递归进入 assets/bin 等子目录时复制全部内容（否则 UI 构建产物全被过滤掉）。
const PUBLISH_FILES = ['index.js', 'client.js', 'assets', 'bin', 'package.json', 'README.md']
function copyDir(src, dst, topLevel = true) {
  fs.mkdirSync(dst, { recursive: true })
  for (const ent of fs.readdirSync(src, { withFileTypes: true })) {
    if (ent.name === 'node_modules' || ent.name === '.git') continue
    if (topLevel && !PUBLISH_FILES.includes(ent.name)) continue
    const s = path.join(src, ent.name)
    const d = path.join(dst, ent.name)
    if (ent.isDirectory()) copyDir(s, d, false)
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

// ── 内容指纹：对打包白名单文件（index.js/client.js/assets/bin/package.json/README.md）
//    计算目录级 SHA-256——解决「文件改了但版本没 bump → 误判已一致跳过」问题 ──
function dirHash(dir) {
  const files = []
  ;(function walk(p, rel) {
    for (const ent of fs.readdirSync(p, { withFileTypes: true })) {
      if (ent.name === 'node_modules' || ent.name === '.git') continue
      if (rel === '' && !PUBLISH_FILES.includes(ent.name)) continue
      const s = path.join(p, ent.name)
      const r = rel ? path.join(rel, ent.name) : ent.name
      if (ent.isDirectory()) walk(s, r)
      else files.push(r)
    }
  })(dir, '')
  files.sort()
  const h = crypto.createHash('sha256')
  for (const f of files) {
    h.update(f)
    h.update('\x00')
    h.update(fs.readFileSync(path.join(dir, f)))
    h.update('\x00')
  }
  return h.digest('hex')
}

// ── 发布记录（.pair/publish/.content-hashes.json）：上次发布时的版本 + 内容指纹 ──
const HASH_FILE = path.join(publishDir, '.content-hashes.json')
function loadHashes() {
  try { return JSON.parse(fs.readFileSync(HASH_FILE, 'utf8')) } catch { return {} }
}
function saveHashes(hashes) {
  fs.mkdirSync(publishDir, { recursive: true })
  fs.writeFileSync(HASH_FILE, JSON.stringify(hashes, null, 2))
}

// 自动 bump patch 版本（写回源 .pair/plugins/<name>/package.json，版本随源码持久化）
function bumpPatch(pkgPath) {
  const raw = JSON.parse(fs.readFileSync(pkgPath, 'utf8'))
  const v = String(raw.version || '0.0.0').split('.')
  v[2] = (parseInt(v[2] || '0', 10) + 1).toString()
  raw.version = v.join('.')
  fs.writeFileSync(pkgPath, JSON.stringify(raw, null, 2) + '\n')
  return raw.version
}

// 检查 npm 上是否已存在（返回已发布版本或 null）
function npmExists(pkgName) {
  // 用 dist-tags 端点检测（registry 对新包 metadata 可能 404，但 dist-tags 立即可查）
  try {
    const out = execSync(`curl -s${PROXY ? ` -x "${PROXY}"` : ''} -w "\\n%{http_code}" "${REGISTRY}/-/package/${pkgName}/dist-tags"`, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] })
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

  // ★ 自动重新构建 UI（2026-08-21）：检测到待处理插件含 assets（UI 构建产物）时，
  //   发布前自动跑 build-ui.mjs，避免「改了 UI 源码却把旧 assets 发出去」。
  //   --no-build 可跳过；构建失败直接中止（不带旧产物发新版本）。
  if (!process.argv.includes('--no-build') && plugins.some((p) => fs.existsSync(path.join(p.dir, 'assets')))) {
    console.log('\n── 检测到 UI 插件（含 assets），发布前自动重新构建 UI（node scripts/build-ui.mjs）──')
    try {
      execSync(`node ${path.join(root, 'scripts', 'build-ui.mjs')}`, { cwd: root, stdio: 'inherit', timeout: 600000 })
    } catch (e) {
      console.error('\n✗ UI 构建失败，中止发布（避免带旧 assets 发新版本）')
      process.exit(1)
    }
  }

  const ok = []
  const fail = []
  const hashes = loadHashes()
  for (const p of plugins) {
    const pkgName = `@paircode/${p.name}`
    const dst = path.join(publishDir, p.name)
    // ★ 指纹感知跳过（2026-08-21）：不仅比版本号，还比打包内容指纹。
    //   - 线上无 → 发布；线上版本不同 → 发布；
    //   - 版本相同 + 指纹相同 → 真一致，跳过；
    //   - 版本相同 + 无记录（首次运行）→ 建立指纹基线，跳过；
    //   - 版本相同 + 指纹变化（改了代码没 bump 版本）→ 自动 bump patch 后发布。
    const existing = npmExists(pkgName)
    const srcHash = dirHash(p.dir)
    const rec = hashes[pkgName] || {}
    if (existing && existing === p.pkg.version) {
      if (rec.hash === srcHash) {
        console.log(`  ⏭ ${pkgName} 已一致（@${existing}，内容指纹相同），跳过`)
        ok.push({ name: p.name, pkgName, out: `skip@${existing}` })
        continue
      }
      if (!rec.hash) {
        hashes[pkgName] = { version: p.pkg.version, hash: srcHash }
        saveHashes(hashes)
        console.log(`  ⏭ ${pkgName} 已存在（@${existing}），首次建立内容指纹基线`)
        ok.push({ name: p.name, pkgName, out: `skip@${existing}（基线）` })
        continue
      }
      // 版本相同但内容变了 → 自动 bump patch（写回源 package.json，随源码持久化）
      const newVer = bumpPatch(path.join(p.dir, 'package.json'))
      console.log(`  ✚ ${pkgName} 内容已变化但版本未 bump（线上 @${existing}），自动升 patch → ${newVer}`)
      p.pkg.version = newVer
    }
    if (existing && existing !== p.pkg.version) {
      console.log(`  ↗ ${pkgName} npm 已有 @${existing} ≠ 本地 ${p.pkg.version}，重新打包${DO_PUBLISH ? '并发布' : '验证'}`)
    } else if (!existing) {
      console.log(`  🆕 ${pkgName} 未发布，准备打包${DO_PUBLISH ? '并发布' : '验证'}`)
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
      // 3. 验证/发布（支持代理：npm --proxy / --https-proxy）
      const cmd = DO_PUBLISH
        ? `npm publish ${path.join(dst, 'package.json').replace(/package\.json$/, '').replace(/\\/g, '/')} --registry=${REGISTRY} --access public${OTP ? ` --otp=${OTP}` : ''}${proxyArgs}`
        : `npm pack ${path.join(dst, 'package.json').replace(/package\.json$/, '').replace(/\\/g, '/')} --dry-run --json${proxyArgs}`
      const out = execSync(cmd, { encoding: 'utf8', cwd: dst, stdio: ['ignore', 'pipe', 'ignore'] })
      hashes[pkgName] = { version: pkg.version, hash: srcHash }
      saveHashes(hashes)
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
