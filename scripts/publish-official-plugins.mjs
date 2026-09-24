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
//   node scripts/publish-official-plugins.mjs --only marketplace,tool-web   # 只发指定插件
//   node scripts/publish-official-plugins.mjs --check       # 只检查现有 npm 包版本（不打包）
//
//   node ... --pkg-prefix paircode-plugin-   # 裸名前缀发布（无 scope 账号用；默认 @paircode/，须以 / 或 - 结尾）
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
// ★ 2026-09-17 独立发布插件（市场分发）真源目录：与 .pair/plugins 同权参与发布，
//   但不进 IDE 发布包（packager.json exclude 这些包名）。
const distPluginsDir = path.join(root, 'plugins-dist')
const publishDir = path.join(root, '.pair', 'publish')
const REGISTRY = process.env.PAIRCODE_NPM_REGISTRY || 'https://registry.npmjs.org'

// ── 代理配置：PAIRCODE_PROXY → HTTPS_PROXY → HTTP_PROXY ──
const PROXY = process.env.PAIRCODE_PROXY || process.env.HTTPS_PROXY || process.env.HTTP_PROXY || ''
const proxyArgs = PROXY ? ` --proxy=${PROXY} --https-proxy=${PROXY}` : ''

// ── npm 认证：显式引用 .pair/publish/.npmrc（与 plugin-publisher 共享同一 token 文件）──
//   ★ 不引用时 npm 走全局 ~/.npmrc——用户改 token 时只改一处，两脚本同步生效。
//   ★ T3（2026-09-12）：PAIRCODE_NPM_USERCONFIG 环境变量可覆盖——CI/临时环境用
//     一次性 userconfig（内容写 `//registry.npmjs.org/:_authToken=${NPM_TOKEN}`，
//     npm 自行做环境变量插值）注入凭据，token 明文不落任何文件。
const npmrcPathRaw = process.env.PAIRCODE_NPM_USERCONFIG || path.join(publishDir, '.npmrc')
// ★ 绝对路径锚定（2026-09-12 实测坑）：npm 命令的 cwd 会切到目标包目录（execSync cwd=dst），
//   相对路径的 userconfig 解析错位 → 认证失败（ENEEDAUTH：「need auth ... registry.npmjs.org」）。
//   故一律 resolve 到仓库根绝对路径后再传给 --userconfig。
const npmrcPath = path.isAbsolute(npmrcPathRaw) ? npmrcPathRaw : path.resolve(root, npmrcPathRaw)

// 同步冷却（npm 限流防护）：Atomics.wait 无子进程开销，Node 主线程可用
function sleepSync(ms) {
  try { Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, ms) } catch {}
}

const args = process.argv.slice(2)
const DO_PUBLISH = args.includes('--publish')
const DO_CHECK = args.includes('--check')
const DO_PLAN = args.includes('--plan')
const DO_ORPHANS = args.includes('--orphans')
const DO_DEPRECATE_ORPHANS = args.includes('--deprecate-orphans')
const otpIdx = args.indexOf('--otp')
const OTP = otpIdx >= 0 ? args[otpIdx + 1] : ''
const onlyIdx = args.indexOf('--only')
const ONLY = onlyIdx >= 0 ? args[onlyIdx + 1].split(',').map((s) => s.trim()).filter(Boolean) : null

// ── 包名前缀（2026-09-24）：默认 @paircode/（scope 包，唯一权威）；
//    --pkg-prefix paircode-plugin- 切裸名官方形态（市场 searchNpmPlugins 第二约定：
//    paircode-plugin-* 裸名包，须带 paircode 关键词，无需 scope 账号即可发布）。
const prefixIdx = args.indexOf('--pkg-prefix')
const PKG_PREFIX = prefixIdx >= 0 ? String(args[prefixIdx + 1] || '').trim() : '@paircode/'
if (!PKG_PREFIX || /\s/.test(PKG_PREFIX) || !/[\/-]$/.test(PKG_PREFIX)) {
  console.error(`--pkg-prefix 非法（应形如 @paircode/ 或 paircode-plugin-，须以 / 或 - 结尾）: ${PKG_PREFIX || '(空)'}`)
  process.exit(1)
}

// 拷贝文件/目录（白名单：只拷发布所需，排除 node_modules/.git 等）
// ★ 2026-08-20 修复：topLevel 只在顶层做 PUBLISH_FILES 过滤；
//   递归进入 assets/bin 等子目录时复制全部内容（否则 UI 构建产物全被过滤掉）。
// ★ 2026-09（L1 人声插件试点）：新增 'lib' —— Node 桥插件的多文件实现必须随包发布。
//   背景：tool-voice 是第一个使用 lib/（wav/dsp/ops/verify/…）的插件，此前白名单
//   只放行 index.js/client.js/assets/bin，会导致发布包缺模块、市场安装后运行即失败。
const PUBLISH_FILES = ['index.js', 'client.js', 'assets', 'bin', 'lib', 'package.json', 'README.md']
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
// ★ 扫描面（2026-09-17）：.pair/plugins（IDE 基线）+ plugins-dist（独立发布插件）。
//   同名同时存在于两处时以基线为准并告警——避免同一包名两个源头的发布歧义。
function listPlugins() {
  const out = []
  const seen = new Set()
  for (const base of [pluginsDir, distPluginsDir]) {
    if (!fs.existsSync(base)) continue
    for (const ent of fs.readdirSync(base, { withFileTypes: true })) {
      if (!ent.isDirectory()) continue
      if (seen.has(ent.name)) {
        console.warn(`[listPlugins] 同名插件同时存在于 .pair/plugins 与 plugins-dist：${ent.name}（以 .pair/plugins 为准，已忽略 ${base}）`)
        continue
      }
      const dir = path.join(base, ent.name)
      const pkgPath = path.join(dir, 'package.json')
      if (!fs.existsSync(path.join(dir, 'index.js')) && !fs.existsSync(pkgPath)) continue
      seen.add(ent.name)
      let pkg = {}
      if (fs.existsSync(pkgPath)) {
        try { pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8')) } catch { pkg = {} }
      }
      out.push({ name: ent.name, dir, pkg })
    }
  }
  return out.sort((a, b) => a.name.localeCompare(b.name))
}

// ── 内容指纹：对打包白名单文件（index.js/client.js/assets/bin/package.json/README.md）
//    计算目录级 SHA-256——解决「文件改了但版本没 bump → 误判已一致跳过」问题
// ★ 指纹分层（2026-08-21）：src（源码）+ artifact（构建产物 assets/bin）分开
//   背景：Go 二进制重编译会嵌入 git VCS 信息（vcs.revision/vcs.time）→ 每次重编译必变；
//         前端重打包（esbuild 确定性压缩）→ 源码不变则产物不变。
//   判定：源码变 → 实质变更 → 自动 bump 发布；仅产物变（重编译/重打包）→ 刷新产物基线，不误报不误发。
//   ★ 与 scripts/plugin-publisher.mjs 的 dirHashSplit 保持同一算法（rel + '\0' + data），共享记录可互相读取。
const ARTIFACT_DIRS = ['assets', 'bin']
function isArtifactRel(rel) {
  const top = String(rel).split(/[\\/]/)[0]
  return ARTIFACT_DIRS.includes(top)
}
function dirHashSplit(dir) {
  const files = []
  ;(function walk(p, rel) {
    for (const ent of fs.readdirSync(p, { withFileTypes: true })) {
      if (ent.name === 'node_modules' || ent.name === '.git') continue
      if (rel === '' && !PUBLISH_FILES.includes(ent.name)) continue
      const s = path.join(p, ent.name)
      const r = rel ? path.join(rel, ent.name) : ent.name
      if (ent.isDirectory()) walk(s, r)
      else files.push({ rel: r, data: fs.readFileSync(s) })
    }
  })(dir, '')
  files.sort((a, b) => (a.rel < b.rel ? -1 : 1))
  const src = crypto.createHash('sha256')
  const art = crypto.createHash('sha256')
  let hasArtifact = false
  for (const f of files) {
    if (isArtifactRel(f.rel)) { art.update(f.rel); art.update('\0'); art.update(f.data); hasArtifact = true }
    else { src.update(f.rel); src.update('\0'); src.update(f.data) }
  }
  // 空产物归一化：无 bin/assets 的纯 js 插件 artifact 恒为 ''（与旧记录迁移值一致，不误报）
  return { src: src.digest('hex'), artifact: hasArtifact ? art.digest('hex') : '' }
}
// 旧记录迁移：旧格式 {version, hash} 的 hash 是「完整内容旧算法」，与新分层 src（只算源码）
// 算法不兼容 → 强行比对必误判 → 视为无记录重建基线（保守不 bump、不发布）
function migrateRec(rec) {
  if (!rec) return null
  if (rec.src) return rec // 新格式 {version, src, artifact} 直接用
  return null
}
function dirHash(dir) {
  return dirHashSplit(dir).src + dirHashSplit(dir).artifact
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
// 过滤 npm notice 噪音行（notice 是正常打包清单，非错误），提取真实错误信息
function cleanNpmError(raw) {
  const lines = String(raw || '').split('\n')
  const errLines = lines.filter((l) => !/^\s*npm notice/i.test(l) && l.trim())
  return errLines.join(' | ').slice(0, 500) || String(raw || '').slice(0, 500)
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

// ── T1（2026-09-12）：本地版本 < 线上版本的处理 ──
// 场景：线上更高（历史发布后本地版本号被回退/分叉）。npm 不允许对已发布版本重发，
// 直接发必 EPUBLISHCONFLICT。故自动 bump 到「线上版本的下一个 patch」再发布。
// 前置纪律：本地内容必须 ≥ 线上（否则是降级发布）——用 --compare 逐包核对内容。
function semverCmp(a, b) {
  const pa = String(a || '0').split('.').map((x) => parseInt(x, 10) || 0)
  const pb = String(b || '0').split('.').map((x) => parseInt(x, 10) || 0)
  for (let i = 0; i < 3; i++) {
    if ((pa[i] || 0) !== (pb[i] || 0)) return (pa[i] || 0) - (pb[i] || 0)
  }
  return 0
}
function bumpAbove(pkgPath, onlineVer) {
  const raw = JSON.parse(fs.readFileSync(pkgPath, 'utf8'))
  raw.version = nextPatch(onlineVer)
  fs.writeFileSync(pkgPath, JSON.stringify(raw, null, 2) + '\n')
  return raw.version
}
function nextPatch(ver) {
  const v = String(ver || '0.0.0').split('.')
  v[2] = (parseInt(v[2] || '0', 10) + 1).toString()
  return v.join('.')
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

// ── T2（2026-09-12）：孤儿包（本地已删、registry 仍在）检测与 deprecate ──
// 替代关系映射（本地孤儿名 → 处置说明）；未列出者给通用文案。
const ORPHAN_HINTS = {
  'tool-vision': '已并入 @paircode/tool-web（read_image 工具）',
  'tool-vision-llm': '已并入 @paircode/tool-web（read_image 工具）',
  'tool-screenshot': '已并入 @paircode/tool-web（screenshot 工具）',
  'tool-web-debug': '已并入 @paircode/tool-web（web_debug 工具）',
  'tool-shell': '已由 @paircode/tool-exec 取代（exec_command/write_stdin/kill_process）',
  'tool-core': '已移除：multi_edit/move_file/delete_file 由 apply_patch 覆盖',
  'tool-debug': '已移除（纯命令行包装壳，无独立价值）',
  'tool-git': '已移除：git 操作统一走命令执行（exec_command）',
  'tool-verify': '已并入 @paircode/tool-resource',
  'tool-codegraph-extra': '已并入 @paircode/tool-codegraph',
  'host-capability-probe': '已移除（宿主能力探针，验证完成后删除）',
  'web-api': '已移除（/api/ext 示例插件，能力由 ext_routes + 插件生态覆盖）',
  // ★ 2026-09-17「UI 与工具同包」六域合并：独立 UI 包停更，功能已并入对应 tool-* 包
  'ui-art': '已并入 @paircode/tool-art（UI 与工具同包，2026-09-17）',
  'ui-design': '已并入 @paircode/tool-design（UI 与工具同包，2026-09-17）',
  'ui-model': '已并入 @paircode/tool-model（UI 与工具同包，2026-09-17）',
  'ui-music': '已并入 @paircode/tool-music（UI 与工具同包，2026-09-17）',
  'ui-rig': '已并入 @paircode/tool-rig（UI 与工具同包，2026-09-17）',
  'ui-voice': '已并入 @paircode/tool-voice（UI 与工具同包，2026-09-17）',
}

// registry 上全部官方包（keywords:paircode 全集；search 索引可能滞后于新发布，
// 故只用于「孤儿」判定——多出的名字才是孤儿，缺名字不代表未发布）
function npmOfficialPackageNames() {
  const url = `${REGISTRY}/-/v1/search?text=keywords:paircode&size=250`
  try {
    const out = execSync(`curl -s${PROXY ? ` -x "${PROXY}"` : ''} "${url}"`, {
      encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'], timeout: 60000,
    })
    const j = JSON.parse(out)
    return (j.objects || []).map((o) => o.package && o.package.name).filter(Boolean)
  } catch { return [] }
}

// 孤儿 = registry 有、本地插件目录无（去掉 @paircode/ 前缀比对）
function findOrphans(localNames) {
  const local = new Set(localNames)
  return npmOfficialPackageNames()
    .filter((n) => n.startsWith('@paircode/'))
    .map((n) => n.slice('@paircode/'.length))
    .filter((n) => n && !local.has(n))
    .sort()
}

function main() {
  const plugins = listPlugins().filter((p) => !ONLY || ONLY.includes(p.name))
  console.log(`══ PairCode 官方插件发布 ══ registry=${REGISTRY} mode=${DO_PUBLISH ? 'PUBLISH' : 'pack-验证'} prefix=${PKG_PREFIX}`)
  if (ONLY) console.log(`精选: ${ONLY.join(', ')}`)
  console.log(`待处理插件 ${plugins.length} 个：`)
  for (const p of plugins) console.log(`  - ${p.name}`)

  if (DO_CHECK) {
    console.log('\n── 现有 npm 版本检查 ──')
    for (const p of plugins) {
      const full = `${PKG_PREFIX}${p.name}`
      const ver = npmExists(full)
      console.log(`  ${ver ? `${PKG_PREFIX}${p.name}@${ver}（已存在）` : `${full}（未发布）`}`)
    }
    return
  }

  // ── T2：孤儿包（registry 有、本地已删）检测 / deprecate ──
  if (DO_ORPHANS || DO_DEPRECATE_ORPHANS) {
    const orphans = findOrphans(plugins.map((p) => p.name))
    console.log(`\n── 孤儿包检查（registry 有 / 本地插件目录无）──`)
    if (!orphans.length) console.log('  无孤儿包（registry 与本地一致）')
    for (const o of orphans) console.log(`  - @paircode/${o}：${ORPHAN_HINTS[o] || '本地已移除（不再维护）'}`)
    if (DO_DEPRECATE_ORPHANS) {
      console.log(`\n── 执行 npm deprecate（${orphans.length} 个）──`)
      for (const o of orphans) {
        const msg = `不再维护：${ORPHAN_HINTS[o] || '本地已移除（不再维护）'}。`
        try {
          execSync(`npm deprecate @paircode/${o}@* "${msg}" --registry=${REGISTRY} --userconfig=${npmrcPath}${proxyArgs}`, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'], timeout: 120000 })
          console.log(`  ✅ @paircode/${o} 已标注废弃`)
        } catch (e) {
          console.log(`  ❌ @paircode/${o}: ${cleanNpmError(e.stderr || e.message)}`)
          process.exitCode = 1
        }
      }
    }
    return
  }

  // ── T4：发布计划（只读；不发不打包）──
  if (DO_PLAN) {
    const pub = []; const update = []; const bump = []; const same = []
    for (const p of plugins) {
      const existing = npmExists(`@paircode/${p.name}`)
      if (!existing) pub.push(`${p.name}@${p.pkg.version}`)
      else if (semverCmp(p.pkg.version, existing) < 0) bump.push(`${p.name}（本地 ${p.pkg.version} < 线上 ${existing} → 自动升到 ${nextPatch(existing)}）`)
      else if (existing !== p.pkg.version) update.push(`${p.name}（${existing} → ${p.pkg.version}）`)
      else same.push(`${p.name}@${existing}`)
    }
    const orphans = findOrphans(plugins.map((p) => p.name))
    console.log('\n── 发布计划（只读，不写 registry）──')
    console.log(`  ① 未发布 ${pub.length}：${pub.join(', ') || '无'}`)
    console.log(`  ② 本地更高 ${update.length}：${update.join(', ') || '无'}`)
    console.log(`  ③ 本地更低 ${bump.length}（自动 bump patch 后发布）：${bump.join(', ') || '无'}`)
    console.log(`  ④ 已一致 ${same.length}（跳过）：${same.join(', ') || '无'}`)
    console.log(`  ⑤ 孤儿包 ${orphans.length}（建议 --deprecate-orphans）：${orphans.map((o) => '@paircode/' + o).join(', ') || '无'}`)
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
  for (let pi = 0; pi < plugins.length; pi++) {
    const p = plugins[pi]
    // 发布模式下包间冷却 15s（npm 限流防护；验证模式为本地打包，无需冷却）
    if (pi > 0 && DO_PUBLISH) {
      console.log(`（冷却 15s 防 npm 限流）`)
      sleepSync(15000)
    }
    const pkgName = `${PKG_PREFIX}${p.name}`
    const dst = path.join(publishDir, p.name)
    // ★ 指纹感知跳过（2026-08-21）：版本 + 内容指纹（src 源码 / artifact 构建产物分层）
    //   - 线上无 → 发布；线上版本不同 → 发布；
    //   - 版本相同 + src/artifact 全同 → 真一致，跳过；
    //   - 版本相同 + 无记录（首次运行）→ 建立指纹基线，跳过；
    //   - 版本相同 + src 同 + artifact 不同（重编译/重打包）→ 刷新产物基线，不发布；
    //   - 版本相同 + src 不同（改了源码没 bump 版本）→ 自动 bump patch 后发布。
    const existing = npmExists(pkgName)
    let h = { src: '', artifact: '' }
    try { h = dirHashSplit(p.dir) } catch {}
    const rec = migrateRec(hashes[pkgName]) || {}
    // ★ T1（2026-09-12）：本地版本低于线上（历史回退/分叉）→ npm 不允许对已发布版本重发，
    //   直接发必 EPUBLISHCONFLICT。自动 bump 到「线上版本的下一个 patch」再发布。
    //   前置纪律：本地内容必须 ≥ 线上（用 --compare 逐包核对过内容才允许走此路径）。
    if (existing && semverCmp(p.pkg.version, existing) < 0) {
      const forced = bumpAbove(path.join(p.dir, 'package.json'), existing)
      console.log(`  ⤴ ${pkgName} 本地 ${p.pkg.version} < 线上 ${existing} → 自动升到 ${forced}（npm 不可降版本重发）`)
      p.pkg.version = forced
    }
    if (existing && existing === p.pkg.version) {
      if (rec.src === h.src && rec.artifact === h.artifact) {
        console.log(`  ⏭ ${pkgName} 已一致（@${existing}，内容指纹相同），跳过`)
        ok.push({ name: p.name, pkgName, out: `skip@${existing}` })
        continue
      }
      if (!rec.src) {
        hashes[pkgName] = { version: p.pkg.version, src: h.src, artifact: h.artifact }
        saveHashes(hashes)
        console.log(`  ⏭ ${pkgName} 已存在（@${existing}），首次建立内容指纹基线`)
        ok.push({ name: p.name, pkgName, out: `skip@${existing}（基线）` })
        continue
      }
      if (rec.src === h.src && rec.artifact !== h.artifact) {
        // 仅构建产物变化（重编译/重打包）→ 刷新产物基线，不 bump 不发布（视为已一致，不打扰）
        hashes[pkgName] = { version: p.pkg.version, src: h.src, artifact: h.artifact }
        saveHashes(hashes)
        console.log(`  ⏭ ${pkgName} 已一致（构建产物基线自动刷新：重编译/重打包不触发发布）`)
        ok.push({ name: p.name, pkgName, out: `skip@${existing}（产物基线刷新）` })
        continue
      }
      // 源码变了 → 自动 bump patch（写回源 package.json，随源码持久化）
      const newVer = bumpPatch(path.join(p.dir, 'package.json'))
      console.log(`  ✚ ${pkgName} 源码已变化但版本未 bump（线上 @${existing}），自动升 patch → ${newVer}`)
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
      pkg.files = ['index.js', 'client.js', 'assets', 'bin', 'lib', 'package.json']
      fs.writeFileSync(path.join(dst, 'package.json'), JSON.stringify(pkg, null, 2))
      // 3. 验证/发布（支持代理：npm --proxy / --https-proxy）
      // ★ stderr 必须 pipe 捕获——否则 npm 真实错误（401/网络/权限）被丢弃，只剩 "Command failed" 外壳
      const cmd = DO_PUBLISH
        ? `npm publish ${path.join(dst, 'package.json').replace(/package\.json$/, '').replace(/\\/g, '/')} --registry=${REGISTRY} --access public --userconfig=${npmrcPath}${OTP ? ` --otp=${OTP}` : ''}${proxyArgs}`
        : `npm pack ${path.join(dst, 'package.json').replace(/package\.json$/, '').replace(/\\/g, '/')} --dry-run --json${proxyArgs}`
      const out = execSync(cmd, { encoding: 'utf8', cwd: dst, stdio: ['ignore', 'pipe', 'pipe'], timeout: 120000 })
      hashes[pkgName] = { version: pkg.version, src: h.src, artifact: h.artifact }
      saveHashes(hashes)
      ok.push({ name: p.name, pkgName, out })
      console.log(`  ✅ ${pkgName} ${DO_PUBLISH ? '已发布' : '打包验证通过'}（${pkg.version}）`)
    } catch (e) {
      fail.push({ name: p.name, pkgName, err: cleanNpmError(e.stderr || e.message) })
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
