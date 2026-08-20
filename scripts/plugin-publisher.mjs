#!/usr/bin/env node
// ═══════════════════════════════════════════════════════════════
// plugin-publisher.mjs — 插件上传发布工具（独立单文件脚本）
//
// ★ 本工具与 PairCode IDE 项目本体无关：不依赖项目任何模块/内核/插件体系，
//   仅用 Node 标准库；可整体拷到任意环境单独运行。它扫描并发布的是
//   「任意 PairCode 项目」的磁盘插件目录（.pair/plugins/），默认当前工作区。
//
// 功能：
//   1. token 管理：直接填写/粘贴 npm token 保存（.pair/publish/.npmrc，
//      发布时 --userconfig 引用，不污染全局 ~/.npmrc）；
//   2. 插件列表：扫描 .pair/plugins/*/package.json，自动检测 npm 线上版本
//      （dist-tags）→ 状态：未发布 / 已一致 / 本地有更新 / 线上更高；
//   3. 内容指纹：对打包白名单文件计算 SHA-256 记录到 .content-hashes.json——
//      「版本相同 + 内容已变」不再误判跳过，发布时自动升 patch 版本；
//   4. 自动构建：含 assets 的 UI 插件发布前自动跑 scripts/build-ui.mjs（项目内），
//      避免发旧 UI 构建产物；
//   5. 自动发布：未发布 → 自动发布；本地有更新 / 内容已变 → 自动更新版本发布；
//      已一致 / 线上更高 → 自动跳过（返回原因）。
//   6. 代理配置：Web UI 可保存代理到 .pair/publish/.proxy（优先级高于环境变量
//      PAIRCODE_PROXY / HTTPS_PROXY / HTTP_PROXY），curl 检测与 npm 发布均走该代理。
//
// 用法（node 需 ≥18，fetch 内置）：
//   node scripts/plugin-publisher.mjs                交互式 CLI 菜单
//   node scripts/plugin-publisher.mjs --list         只列出插件与版本状态
//   node scripts/plugin-publisher.mjs --publish      自动发布全部待发（交互确认）
//   node scripts/plugin-publisher.mjs --set-token    交互输入 token 并保存
//   node scripts/plugin-publisher.mjs --set-token <token>  直接保存
//   node scripts/plugin-publisher.mjs --serve [port] 本地 Web UI（默认 8787）
//   node scripts/plugin-publisher.mjs --root <dir>   指定 PairCode 项目目录（默认当前目录）
//
// 环境变量：PAIRCODE_NPM_REGISTRY 可覆盖 registry（如本地 verdaccio 测试）
// ═══════════════════════════════════════════════════════════════
import fs from 'node:fs'
import path from 'node:path'
import crypto from 'node:crypto'
import { execSync } from 'node:child_process'
import { createInterface } from 'node:readline/promises'
import http from 'node:http'

// ── 路径与常量（--root 指向任意 PairCode 项目）──
let root = process.cwd()
const args = process.argv.slice(2)
const rootIdx = args.indexOf('--root')
if (rootIdx >= 0 && args[rootIdx + 1]) root = path.resolve(args[rootIdx + 1])

const pluginsDir = path.join(root, '.pair', 'plugins')
const publishDir = path.join(root, '.pair', 'publish')
const npmrcPath = path.join(publishDir, '.npmrc')
const REG = String(process.env.PAIRCODE_NPM_REGISTRY || '').replace(/\/+$/, '') || 'https://registry.npmjs.org'
const SCOPED = '@paircode'
const PUBLISH_FILES = ['index.js', 'client.js', 'assets', 'bin', 'package.json', 'README.md']
const COOLDOWN_MS = 15000 // 包间冷却（npm 限流防护）
// ── 代理配置：Web 配置(.pair/publish/.proxy 文件) → PAIRCODE_PROXY → HTTPS_PROXY → HTTP_PROXY ──
// ★ node fetch 不读 HTTP(S)_PROXY 环境变量，故线上查询改走 curl（天然支持 -x）
const proxyPath = path.join(publishDir, '.proxy')
function getProxy() {
  // 1) Web UI 保存的代理配置（最高优先级，用户显式配置）
  try { const p = String(fs.readFileSync(proxyPath, 'utf8') || '').trim(); if (p) return p } catch {}
  // 2) 环境变量回退
  return process.env.PAIRCODE_PROXY || process.env.HTTPS_PROXY || process.env.HTTP_PROXY || ''
}
function getProxyArgs() { const p = getProxy(); return p ? ` --proxy=${p} --https-proxy=${p}` : '' }
function proxySource() {
  try { if (String(fs.readFileSync(proxyPath, 'utf8') || '').trim()) return 'web-config' } catch {}
  if (process.env.PAIRCODE_PROXY || process.env.HTTPS_PROXY || process.env.HTTP_PROXY) return 'env'
  return ''
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
const log = (m = '') => console.log(m)

// ═══════════════════════════════════════════════════════════════
// 核心逻辑（CLI 与 Web UI 共用）
// ═══════════════════════════════════════════════════════════════

function regHost() {
  const m = /^https?:\/\/([^/]+)/.exec(REG)
  return m ? m[1] : REG
}

// 插件目录清单（跳过无 package.json 且无 index.js 的目录）
function listPlugins() {
  if (!fs.existsSync(pluginsDir)) return []
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
    out.push({
      name: ent.name,
      version: pkg.version || '0.0.0',
      purpose: pkg.purpose || pkg.description || '',
      scope: pkg.scope || 'project',
    })
  }
  return out.sort((a, b) => a.name.localeCompare(b.name))
}

// 线上版本检测（npm packument，404=未发布）→ {ok, version?, reason?}
// ★ 2026-08-21：用 packument 端点而非 dist-tags——npm registry 对「未发布的
//   scoped 包」dist-tags 返回 401（非 404），误判为检测失败；packument 正确
//   返回 404=未发布 / 200=已发布。
async function remoteCheck(name) {
  const pkgName = SCOPED + '/' + name
  const url = `${REG}/${encodeURIComponent(pkgName)}`
  try {
    // curl 查询（packument）：404=未发布 / 200=已发布；支持代理 -x
    const px = getProxy()
    const out = execSync(`curl -s${px ? ` -x "${px}"` : ''} -w "\\n%{http_code}" "${url}"`, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] })
    const lines = out.trim().split('\n')
    const code = (lines[lines.length - 1] || '').trim()
    if (code === '404') return { ok: true, version: null }
    if (code !== '200') return { ok: false, reason: `HTTP ${code}` }
    const j = JSON.parse(lines.slice(0, -1).join('\n'))
    return { ok: true, version: (j['dist-tags'] && j['dist-tags'].latest) || null }
  } catch (e) {
    return { ok: false, reason: String(e.message || e).slice(0, 80) }
  }
}

// semver 简单比较（主.次.补丁[.预发布]），a>b→1, a<b→-1, 相等→0
function cmpSemver(a, b) {
  const pa = String(a || '').trim().replace(/^v/, '').split(/[.-]/)
  const pb = String(b || '').trim().replace(/^v/, '').split(/[.-]/)
  for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
    const x = pa[i] === undefined ? 0 : (/^\d+$/.test(pa[i]) ? +pa[i] : pa[i])
    const y = pb[i] === undefined ? 0 : (/^\d+$/.test(pb[i]) ? +pb[i] : pb[i])
    if (x > y) return 1
    if (x < y) return -1
  }
  return 0
}

// 状态计算：未发布 / 已一致 / 本地有更新 / 线上更高 / 检测失败
function statusOf(local, remote) {
  if (remote === null) return { code: 'unpublished', label: '未发布' }
  if (remote === local) return { code: 'same', label: '已一致' }
  if (cmpSemver(local, remote) > 0) return { code: 'update', label: '本地有更新' }
  return { code: 'ahead', label: '线上更高' }
}

// 扫描全部插件 + 检测（并发检测，快）；含内容指纹感知（同版本下区分真一致/内容已变/首次基线）
async function scanAll() {
  const plugins = listPlugins()
  const checks = await Promise.all(plugins.map((p) => remoteCheck(p.name)))
  const hashes = readHashes()
  const autoRefresh = [] // 仅产物变化（重编译/重打包）→ 统一刷新基线，不显示不一致
  const out = []
  for (let i = 0; i < plugins.length; i++) {
    const p = plugins[i]
    const rc = checks[i]
    const pkgName = SCOPED + '/' + p.name
    let h = { src: '', artifact: '' }
    try { h = dirHashSplit(path.join(pluginsDir, p.name)) } catch {}
    const rec = migrateRec(hashes[pkgName])
    const srcSame = !!rec && !!h.src && rec.src === h.src
    const artSame = !!rec && !!h.artifact && rec.artifact === h.artifact
    const st = rc.ok ? statusOf(p.version, rc.version) : { code: 'check-failed', label: '检测失败' }
    let note = ''
    // 版本相同但指纹不同 → 细分：源码变=实质变更（自动 bump）；仅产物变=重编译/重打包（自动刷新基线，不显示不一致）
    if (rc.ok && st.code === 'same') {
      if (rec && !srcSame) {
        st.code = 'changed'; st.label = '内容已变'
        note = `源码指纹与上次发布不一致（${rec.src.slice(0, 8)}… ≠ ${h.src.slice(0, 8)}…），版本未 bump，发布时将自动升 patch`
      } else if (rec && srcSame && !artSame) {
        // 仅构建产物变化（重编译/重打包）→ 自动刷新基线，状态视为已一致（不打扰）
        autoRefresh.push({ pkgName, src: h.src, artifact: h.artifact, version: p.version })
        st.code = 'same'; st.label = '已一致'
        note = '构建产物已变化（重编译/重打包），基线已自动刷新；如需发布请先 bump 版本'
      } else if (!rec) {
        st.code = 'baseline'; st.label = '基线'
        note = '线上已是最新版本，首次建立内容指纹基线（本次跳过）'
        // 无记录（含旧格式迁移）→ 一并写盘建基线，下次扫描即稳定为「已一致」
        autoRefresh.push({ pkgName, src: h.src, artifact: h.artifact, version: p.version })
      }
    }
    out.push({
      ...p,
      remote: rc.ok ? rc.version : null,
      status: st.code,
      statusLabel: st.label,
      note,
      srcHash: h.src,
      artifactHash: h.artifact,
      checkError: rc.ok ? '' : rc.reason,
      publishable: rc.ok && (st.code === 'unpublished' || st.code === 'update' || st.code === 'changed'),
    })
  }
  // 仅产物变化（重编译/重打包）统一写盘刷新基线（幂等：刷新后下次扫描即一致）
  if (autoRefresh.length > 0) {
    const hh = readHashes()
    for (const r of autoRefresh) hh[r.pkgName] = { version: r.version, src: r.src, artifact: r.artifact }
    saveHashes(hh)
  }
  return { plugins: out, registry: REG, pkgPrefix: SCOPED + '/', root }
}

// ── token 管理 ──
function readToken() {
  try {
    const c = String(fs.readFileSync(npmrcPath, 'utf8') || '')
    // host 允许含端口（[^/]+）；key 形态 //host/:_authToken=xxx
    const m = /^\/\/([^/]+)\/:_authToken=(\S+)/m.exec(c)
    return m ? m[2] : ''
  } catch { return '' }
}
function maskedToken() {
  const t = readToken()
  return t ? `${t.slice(0, 4)}****${t.slice(-4)}` : ''
}
function saveToken(token) {
  const t = String(token || '').trim()
  if (!t) return { ok: false, error: 'token 不能为空' }
  if (!/^[A-Za-z0-9_\-.]{4,}$/.test(t)) return { ok: false, error: 'token 格式非法（应为 npm token）' }
  try {
    fs.mkdirSync(publishDir, { recursive: true })
    let lines = []
    try { lines = String(fs.readFileSync(npmrcPath, 'utf8') || '').split('\n') } catch {}
    let found = false
    const host = regHost()
    const out = lines.map((ln) => {
      if (/^\/\/.*:\/_authToken=/.test(ln.trim())) { found = true; return `//${host}/:_authToken=${t}` }
      return ln
    })
    if (!found) out.push(`//${host}/:_authToken=${t}`)
    fs.writeFileSync(npmrcPath, out.join('\n').trim() + '\n')
    return { ok: true }
  } catch (e) {
    return { ok: false, error: String(e.message || e).slice(0, 200) }
  }
}
function clearToken() {
  try {
    if (!fs.existsSync(npmrcPath)) return { ok: true }
    const lines = String(fs.readFileSync(npmrcPath, 'utf8') || '').split('\n')
    const out = lines.filter((ln) => !/^\/\/.*:\/_authToken=/.test(ln.trim()))
    fs.writeFileSync(npmrcPath, out.join('\n').trim() + '\n')
    return { ok: true }
  } catch (e) {
    return { ok: false, error: String(e.message || e).slice(0, 200) }
  }
}

// ── 内容指纹：对打包白名单文件（index.js/client.js/assets/bin/package.json/README.md）
//    计算目录级 SHA-256——解决「文件改了但版本没 bump → 误判已一致跳过」问题 ──
const hashFile = path.join(publishDir, '.content-hashes.json')
// ★ 指纹分层（2026-08-21）：src（源码）+ artifact（构建产物 assets/bin）分开
//   背景：Go 二进制重编译会嵌入 git VCS 信息（vcs.revision/vcs.time）→ 每次重编译必变；
//         前端重打包（esbuild 确定性压缩）→ 源码不变则产物不变。
//   判定：源码变 → 实质变更 → 自动 bump 发布；仅产物变（重编译/重打包）→ 刷新产物基线，不误报不误发。
const ARTIFACT_DIRS = ['assets', 'bin']
function isArtifactRel(rel) {
  const top = String(rel).split(/[\\/]/)[0]
  return ARTIFACT_DIRS.includes(top)
}
// 目录指纹拆分：返回 { src, artifact } 两组 SHA-256（walk 逻辑同 dirHash）
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
  const h = crypto.createHash('sha256')
  for (const f of files) { h.update(f.rel); h.update('\0'); h.update(f.data) }
  return h.digest('hex')
}
function readHashes() {
  try { return JSON.parse(fs.readFileSync(hashFile, 'utf8')) } catch { return {} }
}
function saveHashes(h) {
  try { fs.mkdirSync(publishDir, { recursive: true }); fs.writeFileSync(hashFile, JSON.stringify(h, null, 2)) } catch (e) {
    console.warn('[publisher] 指纹记录写盘失败:', e.message)
  }
}
// 自动升 patch（1.2.3 → 1.2.4；非标准版本追加 .1）
function bumpPatch(v) {
  const m = /^(\d+)\.(\d+)\.(\d+)/.exec(String(v || '').trim().replace(/^v/, ''))
  if (!m) return String(v || '0.0.0') + '.1'
  return `${m[1]}.${m[2]}.${+m[3] + 1}`
}
// 过滤 npm notice 噪音行（notice 是正常打包清单，非错误），提取真实错误信息
function cleanNpmError(raw) {
  const lines = String(raw || '').split('\n')
  const errLines = lines.filter((l) => !/^\s*npm notice/i.test(l) && l.trim())
  return errLines.join(' | ').slice(0, 500) || String(raw || '').slice(0, 500)
}

// ── 打包（复制白名单文件 + package.json 改造）──
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
function buildPackage(name) {
  const src = path.join(pluginsDir, name)
  const dst = path.join(publishDir, name)
  try {
    copyDir(src, dst)
    const pkgPath = path.join(dst, 'package.json')
    let pkg = {}
    try { pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8')) } catch { return { ok: false, error: '缺少合法 package.json' } }
    pkg.name = SCOPED + '/' + name
    pkg.description = pkg.purpose || pkg.description || `PairCode 官方插件 ${name}`
    delete pkg.purpose
    pkg.keywords = pkg.keywords || []
    if (!pkg.keywords.includes('paircode')) pkg.keywords.push('paircode')
    pkg.license = pkg.license || 'MIT'
    pkg.publishConfig = { access: 'public' }
    pkg.files = ['index.js', 'client.js', 'assets', 'bin', 'package.json']
    fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2))
    return { ok: true, version: pkg.version }
  } catch (e) {
    return { ok: false, error: String(e.message || e).slice(0, 200) }
  }
}

// 单个插件发布：检测 → 自动 bump（内容变了版本没变时）→ 自动构建 UI（含 assets 时）
//             → 打包 → npm publish（自动跳过已一致/线上更高）；成功后更新内容指纹
async function publishOne(name, { onLog = log } = {}) {
  let local = '0.0.0'
  const srcDir = path.join(pluginsDir, name)
  const pkgPath = path.join(srcDir, 'package.json')
  try {
    local = JSON.parse(fs.readFileSync(pkgPath, 'utf8')).version || '0.0.0'
  } catch {}
  const rc = await remoteCheck(name)
  if (!rc.ok) return { name, ok: false, error: `线上版本检测失败: ${rc.reason}` }
  const st = statusOf(local, rc.version)

  // ── 内容指纹：版本相同 + 指纹不同 → 自动 bump patch；无记录 → 建基线跳过 ──
  const hashes = readHashes()
  const pkgName = SCOPED + '/' + name
  let h = { src: '', artifact: '' }
  try { h = dirHashSplit(srcDir) } catch {}
  const rec = migrateRec(hashes[pkgName])
  if (st.code === 'same' && rec && rec.src !== h.src) {
    // 源码变了 → 自动 bump（实质变更）
    const bumped = bumpPatch(local)
    onLog(`  ✚ ${pkgName} 源码已变化但版本未 bump（线上 @${rc.version}），自动升 patch → ${bumped}`)
    try {
      const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'))
      pkg.version = bumped
      fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n')
      local = bumped
      try { h = dirHashSplit(srcDir) } catch {}
    } catch (e) {
      return { name, ok: false, error: `自动 bump 版本失败: ${String(e.message || e).slice(0, 120)}` }
    }
  } else if (st.code === 'same' && rec && rec.src === h.src && rec.artifact !== h.artifact) {
    // 仅构建产物变化（重编译/重打包）→ 刷新产物基线，不 bump 不发布（视为已一致）
    hashes[pkgName] = { version: local, src: h.src, artifact: h.artifact }
    saveHashes(hashes)
    return { name, ok: false, skipped: true, reason: '已一致（构建产物基线已自动刷新，重编译/重打包不触发发布）' }
  } else if (st.code === 'same' && !rec) {
    // 首次基线：记录本次指纹并跳过（与线上内容一致，无需发布）
    hashes[pkgName] = { version: local, src: h.src, artifact: h.artifact }
    saveHashes(hashes)
    return { name, ok: false, skipped: true, reason: `已一致（线上 @${rc.version}），首次建立内容指纹基线` }
  } else if (st.code === 'same') {
    return { name, ok: false, skipped: true, reason: `已一致（线上 @${rc.version}，内容指纹相同）` }
  }
  if (st.code === 'ahead') return { name, ok: false, skipped: true, reason: `线上更高（${rc.version} > 本地 ${local}），请先本地 bump 版本` }

  // ── 自动构建 UI：含 assets（UI 构建产物）且项目有 build-ui.mjs → 发布前重建，避免发旧 UI ──
  if (fs.existsSync(path.join(srcDir, 'assets')) && fs.existsSync(path.join(root, 'scripts', 'build-ui.mjs'))) {
    onLog('  ⚙ 检测到 UI 插件（含 assets），发布前自动重新构建 UI（node scripts/build-ui.mjs）...')
    try {
      execSync('node scripts/build-ui.mjs', { cwd: root, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'], timeout: 600000 })
      onLog('  ✅ UI 构建完成（全部区域）')
      try { h = dirHashSplit(srcDir) } catch {}
    } catch (e) {
      const msg = cleanNpmError(e.stderr || e.message || e)
      return { name, ok: false, error: `UI 构建失败，中止发布（避免发旧 UI 资产）: ${msg.slice(0, 200)}` }
    }
  }

  onLog(`  ⚙ 打包 ${SCOPED}/${name}@${local} ...`)
  const b = buildPackage(name)
  if (!b.ok) return { name, ok: false, error: b.error }

  if (!readToken()) return { name, ok: false, error: '未保存 npm token（请先设置 token）' }
  const pkgDir = path.join(publishDir, name)
  // 发布：绝对路径的包目录 + userconfig + 代理（相对路径+反斜杠会导致 npm 找不到配置卡认证）
  const cmd = `npm publish "${pkgDir}" --registry=${REG} --userconfig=${npmrcPath} --access public${getProxyArgs()}`
  onLog(`  ⬆ 发布 ${SCOPED}/${name}@${b.version} ...`)
  try {
    const out = execSync(cmd, { cwd: pkgDir, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'], timeout: 120000 })
    // 发布成功 → 更新内容指纹记录（新版本 + 构建后最新 src/artifact）
    hashes[pkgName] = { version: b.version, src: h.src, artifact: h.artifact }
    saveHashes(hashes)
    onLog(`  ✅ ${SCOPED}/${name}@${b.version} 发布成功`)
    return { name, ok: true, action: st.code === 'unpublished' ? 'published' : 'updated', version: b.version, remote: rc.version, out: out.slice(0, 300) }
  } catch (e) {
    const msg = cleanNpmError(e.stderr || e.message || e)
    return { name, ok: false, error: msg.slice(0, 400), version: b.version }
  }
}

// 发布多个（串行 + 冷却；names 空=全部待发）
async function publishMany(names = [], { onLog = log } = {}) {
  let targets = names.filter(Boolean)
  if (targets.length === 0) {
    const { plugins } = await scanAll()
    targets = plugins.filter((p) => p.publishable).map((p) => p.name)
  }
  if (targets.length === 0) {
    onLog('没有需要发布的插件（全部已一致或线上更高）')
    return []
  }
  const results = []
  for (let i = 0; i < targets.length; i++) {
    const name = targets[i]
    if (i > 0) { onLog(`（冷却 ${COOLDOWN_MS / 1000}s 防 npm 限流）`); await sleep(COOLDOWN_MS) }
    results.push(await publishOne(name, { onLog }))
  }
  return results
}

// ═══════════════════════════════════════════════════════════════
// CLI
// ═══════════════════════════════════════════════════════════════

function printList(scan) {
  const w = (s, n) => String(s).padEnd(n)
  log('')
  log(`══ 插件列表（root=${scan.root}）══`)
  log(`  Registry: ${scan.registry}  Token: ${readToken() ? maskedToken() : '未保存'}`)
  log('')
  log(`  ${w('名称', 22)}${w('本地', 10)}${w('线上', 10)}状态`)
  log('  ' + '-'.repeat(60))
  if (scan.plugins.length === 0) { log('  （.pair/plugins/ 下无插件）'); return }
  for (const p of scan.plugins) {
    const remote = p.remote || (p.checkError ? '—' : '未发布')
    const note = p.note ? `  ${p.note}` : ''
    log(`  ${w(p.name, 22)}${w(p.version, 10)}${w(remote, 10)}${p.statusLabel}${p.checkError ? '（' + p.checkError + '）' : ''}${note}`)
  }
  const pend = scan.plugins.filter((p) => p.publishable)
  log('')
  log(`  待发布 ${pend.length} 个：${pend.map((p) => p.name).join(', ') || '（无）'}`)
  log('')
}

async function interactive() {
  const rl = createInterface({ input: process.stdin, output: process.stdout })
  try {
    while (true) {
      const scan = await scanAll()
      printList(scan)
      const tk = readToken() ? maskedToken() : '未保存'
      console.log(`[1] 发布全部待发   [2] 指定插件发布   [3] 更新 token（当前: ${tk}）`)
      console.log(`[4] 打开 Web UI（--serve 8787）   [q] 退出`)
      const ans = (await rl.question('请选择: ')).trim().toLowerCase()
      if (ans === 'q') break
      if (ans === '1') {
        const pend = scan.plugins.filter((p) => p.publishable)
        if (pend.length === 0) { log('没有待发布插件'); continue }
        const yn = (await rl.question(`确认发布 ${pend.map((p) => p.name).join(', ')}？(y/N) `)).trim().toLowerCase()
        if (yn !== 'y') continue
        await publishMany([], { onLog: log })
        await sleep(500)
      } else if (ans === '2') {
        const names = (await rl.question('输入插件名（逗号分隔，可用 * 表示全部待发）: ')).trim()
        if (!names) continue
        if (names === '*') { await publishMany([], { onLog: log }); await sleep(500); continue }
        await publishMany(names.split(/[,，\s]+/).filter(Boolean), { onLog: log })
        await sleep(500)
      } else if (ans === '3') {
        const t = (await rl.question('粘贴 npm token（留空取消）: ')).trim()
        if (!t) continue
        const r = saveToken(t)
        log(r.ok ? `✅ token 已保存（${maskedToken()}）→ ${npmrcPath}` : `❌ ${r.error}`)
      }
    }
  } finally {
    rl.close()
  }
}

// ═══════════════════════════════════════════════════════════════
// Web UI（--serve：本地独立服务，内嵌页面，可直接填 token）
// ═══════════════════════════════════════════════════════════════

const PAGE_HTML = `<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="utf-8">
<title>PairCode 插件上传发布工具</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
:root{--bg:#1e1e2e;--bg2:#252537;--bg3:#2f2f46;--fg:#e4e4f0;--muted:#9a9ab0;--border:#3a3a52;
--accent:#7c7cf0;--ok:#3ddc84;--err:#f87171;--warn:#f59e0b;--blue:#60a5fa;--mono:Consolas,monospace}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--fg);font:13px/1.5 system-ui,sans-serif}
.wrap{max-width:900px;margin:0 auto;padding:20px}
h1{font-size:17px;display:flex;align-items:center;gap:8px}
h1 .dot{width:10px;height:10px;border-radius:50%;background:var(--accent);display:inline-block}
.sub{color:var(--muted);font-size:12px;margin:-4px 0 14px 18px}
.card{background:var(--bg2);border:1px solid var(--border);border-radius:8px;padding:12px 14px;margin-bottom:14px}
.card h2{font-size:13px;margin:0 0 8px;color:var(--fg)}
.row{display:flex;align-items:center;gap:8px;flex-wrap:wrap}
input[type=password],input[type=text]{background:var(--bg3);border:1px solid var(--border);color:var(--fg);
border-radius:5px;padding:6px 8px;font-size:13px;flex:1;min-width:160px;outline:none}
input:focus{border-color:var(--accent)}
button{background:var(--bg3);border:1px solid var(--border);color:var(--fg);border-radius:5px;
padding:6px 12px;cursor:pointer;font-size:13px;display:inline-flex;align-items:center;gap:5px}
button:hover:not(:disabled){border-color:var(--accent)}button:disabled{opacity:.45;cursor:default}
button.primary{background:var(--accent);border-color:var(--accent);color:#fff}
button.danger{color:var(--err)}button.sm{padding:3px 8px;font-size:12px}
.hint{color:var(--muted);font-size:11px;margin-top:6px}
code{font-family:var(--mono);background:var(--bg3);padding:1px 5px;border-radius:3px;font-size:12px}
table{width:100%;border-collapse:collapse;font-size:12px}
th{text-align:left;color:var(--muted);font-weight:500;font-size:11px;padding:5px 6px;border-bottom:1px solid var(--border)}
td{padding:6px;border-bottom:1px solid var(--border);vertical-align:top}
.pkg{word-break:break-all}.ver{font-family:var(--mono);white-space:nowrap}
.purpose{color:var(--muted);font-size:11px;max-width:280px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;display:block}
.badge{display:inline-block;padding:1px 8px;border-radius:10px;font-size:11px;white-space:nowrap}
.b-unpublished{background:var(--warn);color:#1a1a1a}.b-same{background:var(--bg3);color:var(--muted)}
.b-update{background:var(--blue);color:#1a1a1a}.b-ahead{background:var(--bg3);color:var(--muted)}
.b-check-failed{background:var(--err);color:#fff}
#log{max-height:180px;overflow:auto;font:12px/1.6 var(--mono)}
#log .ok{color:var(--ok)}#log .err{color:var(--err)}#log .skip{color:var(--muted)}#log .info{color:var(--muted)}
.spin{display:inline-block;width:12px;height:12px;border:2px solid var(--accent);border-top-color:transparent;
border-radius:50%;animation:sp 1s linear infinite;vertical-align:-2px}
@keyframes sp{to{transform:rotate(360deg)}}
</style></head><body><div class="wrap">
<h1><span class="dot"></span>PairCode 插件上传发布工具</h1>
<div class="sub">独立构建工具 · 扫描 .pair/plugins/ → 自动检测版本 → 一键发布到 npm（@paircode/*）</div>

<div class="card">
  <h2>NPM Token</h2>
  <div class="row">
    <input id="token" type="password" placeholder="粘贴 npm token（granular，无需 2FA）">
    <button id="saveToken" class="primary">保存</button>
    <button id="clearToken" class="danger" style="display:none">清除</button>
  </div>
  <div class="hint" id="tokenHint">保存到 <code>.pair/publish/.npmrc</code>（工作区内），发布时自动引用，不污染全局 ~/.npmrc</div>
</div>

<div class="card">
  <h2>网络代理</h2>
  <div class="row">
    <input id="proxy" type="text" placeholder="如 http://127.0.0.1:7890（留空=直连）">
    <button id="saveProxy" class="primary">保存</button>
    <button id="clearProxy" class="danger" style="display:none">清除</button>
  </div>
  <div class="hint" id="proxyHint">保存到 <code>.pair/publish/.proxy</code>（工作区内），线上检测（curl）与发布（npm）均走该代理 · 优先级高于环境变量 PAIRCODE_PROXY</div>
</div>

<div class="card">
  <div class="row" style="justify-content:space-between">
    <h2 style="margin:0">插件列表 <span id="count" class="hint"></span></h2>
    <div class="row">
      <button id="refresh">↻ 刷新</button>
      <button id="publishAll" class="primary">发布全部待发</button>
    </div>
  </div>
  <table id="table"><thead><tr>
    <th style="width:26px"></th><th>插件</th><th style="width:70px">本地</th>
    <th style="width:70px">线上</th><th>状态</th><th style="width:36px"></th>
  </tr></thead><tbody id="tbody"><tr><td colspan="6" style="color:var(--muted)">加载中...</td></tr></tbody></table>
</div>

<div class="card" id="logCard" style="display:none">
  <h2>发布日志</h2>
  <div id="log"></div>
</div>
</div>
<script>
const $=(s)=>document.querySelector(s)
const api=async(p,o={})=>{const r=await fetch(p,{method:o.method||'GET',
headers:{'Content-Type':'application/json'},body:o.body?JSON.stringify(o.body):undefined});
const j=await r.json().catch(()=>({}));if(!r.ok)throw new Error(j.error||('HTTP '+r.status));return j}
let mask=''
async function loadToken(){try{const j=await api('/api/token');mask=j.masked||'';
const inp=$('#token');inp.placeholder=mask?('已保存: '+mask+'（留空则保存新 token）'):'粘贴 npm token';
$('#clearToken').style.display=j.has?'':'none';$('#tokenHint').innerHTML='保存到 <code>.pair/publish/.npmrc</code>（工作区内） · registry: <code>'+(j.registry||'')+'</code>'}catch(e){}}
async function loadProxy(){try{const j=await api('/api/proxy');
const inp=$('#proxy');inp.placeholder=j.proxy?('已配置: '+j.proxy):'如 http://127.0.0.1:7890（留空=直连）';
$('#clearProxy').style.display=j.proxy?'':'none';
$('#proxyHint').innerHTML='保存到 <code>.pair/publish/.proxy</code> · 当前'+(j.proxy?('生效: <code>'+j.proxy+'</code>（'+j.source+'）'):'未配置代理（直连）')}catch(e){}}
const badge={unpublished:['b-unpublished','未发布'],same:['b-same','已一致'],update:['b-update','本地有更新'],ahead:['b-ahead','线上更高'],'check-failed':['b-check-failed','检测失败'],changed:['b-update','内容已变'],'artifact-changed':['b-ahead','仅产物变'],baseline:['b-same','基线']}
async function loadList(){const j=await api('/api/list');
$('#count').textContent='· '+j.plugins.length+' 个 · 待发布 '+j.plugins.filter(p=>p.publishable).length+' 个';
const tb=$('#tbody');tb.innerHTML=''
if(!j.plugins.length){tb.innerHTML='<tr><td colspan="6" style="color:var(--muted)">未发现磁盘插件</td></tr>';return}
for(const p of j.plugins){const b=badge[p.status]||['b-same',p.status];
const tr=document.createElement('tr')
tr.innerHTML='<td>'+(p.publishable?'<input type="checkbox" checked data-n="'+p.name+'">':'')+'</td>'+
'<td><span class="pkg">'+(j.pkgPrefix||'@paircode/')+p.name+'</span><span class="purpose" title="'+esc(p.purpose)+'">'+esc(p.purpose)+'</span></td>'+
'<td class="ver">'+p.version+'</td><td class="ver">'+(p.remote||(p.checkError?'—':'未发布'))+'</td>'+
'<td><span class="badge '+b[0]+'" title="'+(p.note?esc(p.note):b[1])+'">'+b[1]+'</span>'+(p.note?'<div class="purpose" title="'+esc(p.note)+'">'+esc(p.note)+'</div>':'')+'</td>'+
'<td>'+(p.publishable?'<button class="sm" data-one="'+p.name+'">发布</button>':'')+'</td>'
tb.appendChild(tr)}
tb.querySelectorAll('[data-one]').forEach(b=>b.onclick=()=>publish([b.dataset.one]))
}
const esc=s=>String(s||'').replace(/[&<>"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]))
function addLog(msg,cls){const l=$('#log');$('#logCard').style.display=''
const d=document.createElement('div');d.className=cls||'info';d.textContent=msg;l.appendChild(d);l.scrollTop=l.scrollHeight}
async function publish(names){const btn=$('#publishAll');btn.disabled=true
$('#log').innerHTML='';addLog('开始发布 '+(names.length?names.join(', '):'全部待发')+' ...')
try{const j=await api('/api/publish',{method:'POST',body:{names}})
for(const r of (j.results||[])){if(r.ok)addLog('✅ '+r.name+' 发布成功 @'+r.version,'ok')
else if(r.skipped)addLog('⏭ '+r.name+' 跳过：'+r.reason,'skip')
else addLog('❌ '+r.name+': '+(r.error||'未知错误'),'err')}
if(j.error)addLog('❌ '+(j.error||''),'err')
if(j.message)addLog('ℹ '+j.message,'info')
await loadList()}catch(e){addLog('❌ '+e.message,'err')}finally{btn.disabled=false}}
$('#saveToken').onclick=async()=>{const t=$('#token').value.trim()
if(!t){addLog('❌ 请先粘贴 token','err');return}
try{await api('/api/token',{method:'POST',body:{token:t}});$('#token').value='';await loadToken();addLog('✅ token 已保存','ok')}catch(e){addLog('❌ 保存失败: '+e.message,'err')}}
$('#clearToken').onclick=async()=>{try{await api('/api/token/clear',{method:'POST',body:{}});await loadToken();addLog('ℹ token 已清除','info')}catch(e){addLog('❌ '+e.message,'err')}}
$('#saveProxy').onclick=async()=>{const p=$('#proxy').value.trim()
try{await api('/api/proxy',{method:'POST',body:{proxy:p}});$('#proxy').value='';await loadProxy();addLog(p?('✅ 代理已保存: '+p):'ℹ 代理已清除（直连）','ok')}catch(e){addLog('❌ 保存失败: '+e.message,'err')}}
$('#clearProxy').onclick=async()=>{try{await api('/api/proxy',{method:'POST',body:{proxy:''}});await loadProxy();addLog('ℹ 代理已清除','info')}catch(e){addLog('❌ '+e.message,'err')}}
$('#refresh').onclick=async()=>{try{await loadList();addLog('↻ 状态已刷新','info')}catch(e){addLog('❌ '+e.message,'err')}}
$('#publishAll').onclick=()=>{const checked=[...document.querySelectorAll('tbody input[type=checkbox]:checked')].map(i=>i.dataset.n)
publish(checked)}
loadToken();loadProxy();loadList().catch(e=>addLog('❌ 加载列表失败: '+e.message,'err'))
</script></body></html>`

function startServer(port) {
  const server = http.createServer(async (req, res) => {
    const u = new URL(req.url, 'http://localhost')
    const send = (code, obj) => {
      res.writeHead(code, { 'Content-Type': 'application/json; charset=utf-8' })
      res.end(JSON.stringify(obj))
    }
    try {
      if (u.pathname === '/' || u.pathname === '/index.html') {
        res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' })
        res.end(PAGE_HTML)
        return
      }
      if (u.pathname === '/api/list' && req.method === 'GET') {
        const scan = await scanAll()
        send(200, { ...scan, token: { has: !!readToken(), masked: maskedToken() } })
        return
      }
      if (u.pathname === '/api/token' && req.method === 'GET') {
        send(200, { has: !!readToken(), masked: maskedToken(), registry: REG })
        return
      }
      if (u.pathname === '/api/token' && req.method === 'POST') {
        let body = {}
        try { body = JSON.parse(await readBody(req)) } catch {}
        const r = saveToken(body.token)
        if (!r.ok) return send(400, { error: r.error })
        send(200, { ok: true })
        return
      }
      if (u.pathname === '/api/token/clear' && req.method === 'POST') {
        const r = clearToken()
        if (!r.ok) return send(400, { error: r.error })
        send(200, { ok: true })
        return
      }
      if (u.pathname === '/api/proxy' && req.method === 'GET') {
        send(200, { proxy: getProxy(), source: proxySource() === 'web-config' ? 'web 配置' : (proxySource() === 'env' ? '环境变量' : '') })
        return
      }
      if (u.pathname === '/api/proxy' && req.method === 'POST') {
        let body = {}
        try { body = JSON.parse(await readBody(req)) } catch {}
        const p = String(body.proxy || '').trim()
        try {
          if (p) { fs.mkdirSync(publishDir, { recursive: true }); fs.writeFileSync(proxyPath, p + '\n') }
          else if (fs.existsSync(proxyPath)) fs.unlinkSync(proxyPath)
          send(200, { ok: true, proxy: p })
        } catch (e) {
          send(500, { error: String(e.message || e).slice(0, 200) })
        }
        return
      }
      if (u.pathname === '/api/publish' && req.method === 'POST') {
        let body = {}
        try { body = JSON.parse(await readBody(req)) } catch {}
        if (!readToken()) return send(400, { error: '尚未保存 npm token，请先在上方填写 token' })
        const names = Array.isArray(body.names) ? body.names.filter(Boolean) : []
        // 发布是长任务（npm publish 30-120s/个），同步串行执行；期间其他请求排队（本地单用户可接受）
        const results = await publishMany(names, { onLog: (m) => console.log('[publisher] ' + m) })
        send(200, { ok: true, results })
        return
      }
      send(404, { error: 'not found' })
    } catch (e) {
      send(500, { error: String(e.message || e).slice(0, 300) })
    }
  })
  server.listen(port, () => {
    log(`\n══ PairCode 插件上传发布工具（Web UI）══`)
    log(`  地址: http://127.0.0.1:${port}`)
    log(`  项目: ${root}`)
    log(`  Registry: ${REG} · Token: ${readToken() ? maskedToken() : '未保存'}`)
    log(`  Ctrl+C 退出\n`)
  })
  return server
}
function readBody(req) {
  return new Promise((resolve, reject) => {
    let d = ''
    req.on('data', (c) => { d += c; if (d.length > 1e6) req.destroy() })
    req.on('end', () => resolve(d))
    req.on('error', reject)
  })
}

// ═══════════════════════════════════════════════════════════════
// 入口
// ═══════════════════════════════════════════════════════════════

async function main() {
  if (args.includes('--list')) {
    const scan = await scanAll()
    printList(scan)
    return
  }
  if (args.includes('--publish')) {
    // 支持 --publish <name> 单发；无名字 = 发布全部待发
    const pIdx = args.indexOf('--publish')
    const target = args[pIdx + 1] && !args[pIdx + 1].startsWith('--') ? args[pIdx + 1].trim() : ''
    if (target) {
      const r = await publishOne(target, { onLog: log })
      log(r.ok ? `✅ ${SCOPED}/${target}@${r.version} ${r.action === 'published' ? '首次发布' : '更新'}成功` : (r.skipped ? `⏭ 跳过：${r.reason}` : `❌ ${target} 发布失败: ${r.error}`))
      return
    }
    const scan = await scanAll()
    printList(scan)
    const pend = scan.plugins.filter((p) => p.publishable)
    if (pend.length === 0) { log('\n没有待发布插件（全部已一致或线上更高）'); return }
    log(`\n将发布 ${pend.length} 个：${pend.map((p) => p.name).join(', ')}`)
    await publishMany([], { onLog: log })
    return
  }
  if (args.includes('--set-token')) {
    const tkIdx = args.indexOf('--set-token')
    const inline = args[tkIdx + 1] && !args[tkIdx + 1].startsWith('--') ? args[tkIdx + 1] : ''
    let t = inline
    if (!t) {
      const rl = createInterface({ input: process.stdin, output: process.stdout })
      t = (await rl.question('粘贴 npm token（输入后回车保存）: ')).trim()
      rl.close()
    }
    const r = saveToken(t)
    log(r.ok ? `✅ token 已保存（${maskedToken()}）→ ${npmrcPath}` : `❌ ${r.error}`)
    return
  }
  if (args.includes('--serve')) {
    const pIdx = args.indexOf('--serve')
    const port = args[pIdx + 1] && /^\d+$/.test(args[pIdx + 1]) ? +args[pIdx + 1] : 8787
    startServer(port)
    return
  }
  // 默认：交互式 CLI
  await interactive()
}

// 入口（仅直接运行时执行；被 import 时不触发）
const isMain = (() => {
  try { return process.argv[1] && import.meta.url === new URL('file://' + path.resolve(process.argv[1]).replace(/\\/g, '/')).href } catch { return true }
})()
if (isMain) {
  main().catch((e) => { console.error('错误:', e.message || e); process.exit(1) })
}
