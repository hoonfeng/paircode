// publish-retry-forever.mjs — 无限重试补发官方插件（限流多久都等）
// ★ 2026-08-20 增强：
//   1. 开头自动重新打包（跑 publish-official-plugins.mjs 默认模式，从 .pair/plugins
//      复制最新内容到 .pair/publish/，避免发旧版 UI 资源）
//   2. 版本感知跳过：仅当 npm 已存在「相同版本」才跳过（本地 bump 过的新版本会正常发布）
// 用法：node scripts/publish-retry-forever.mjs
import { execSync } from 'node:child_process'
import { readFileSync, existsSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const REGISTRY = 'https://registry.npmjs.org'
const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), '..')
const PUBLISH_DIR = join(REPO_ROOT, '.pair', 'publish')
//   3. 清单含 marketplace/ui-activitybar（本地版本 ≠ npm 版本 → 自动重发修复内容）
const REMAIN = ['marketplace','ui-activitybar','ui-appearance','ui-editor','ui-modals','ui-quick-exec','ui-right-panel','ui-sidebar','ui-statusbar','ui-statusbar-conn','ui-titlebar','web-api']
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
const log = (m) => console.log(new Date().toISOString() + ' ' + m)

function npmExists(pkgName) {
  try {
    const out = execSync(`curl -s -w "\n%{http_code}" "${REGISTRY}/-/package/${pkgName}/dist-tags"`, { encoding: 'utf8', stdio: ['ignore','pipe','ignore'] })
    const lines = out.trim().split('\n'); const code = lines[lines.length - 1]
    if (code !== '200') return null
    try { return JSON.parse(lines.slice(0,-1).join('\n')).latest || null } catch { return null }
  } catch { return null }
}

// 本地待发布版本（.pair/publish/<name>/package.json）
function localVersion(name) {
  const p = join(PUBLISH_DIR, name, 'package.json')
  if (!existsSync(p)) return null
  try { return JSON.parse(readFileSync(p, 'utf8')).version || null } catch { return null }
}

// ── 步骤 1：重新打包（从 .pair/plugins 刷新 .pair/publish/，含最新 UI 构建产物）──
log('══ 步骤 1/2：重新打包官方插件（publish-official-plugins.mjs 默认=验证模式）══')
try {
  execSync(`node ${join(REPO_ROOT, 'scripts', 'publish-official-plugins.mjs')}`, {
    cwd: REPO_ROOT, encoding: 'utf8', stdio: ['ignore', 'inherit', 'ignore'], timeout: 300000,
  })
} catch (e) {
  log(`重新打包失败: ${String(e.message || e).split('\n')[0]}（继续用现有 .pair/publish/ 内容）`)
}

// ── 步骤 2：逐个补发（每包 3 分钟重试直到成功）──
log('══ 步骤 2/2：补发未发布/新版本包 ══')
for (const name of REMAIN) {
  const pkgName = `@paircode/${name}`
  const dir = join(PUBLISH_DIR, name)
  const want = localVersion(name) || '未知'
  let tries = 0
  while (true) {
    const ver = npmExists(pkgName)
    // 版本感知跳过：npm 已有「相同版本」才算已发布；本地版本不同（bump 过）→ 发布新版本
    if (ver === localVersion(name) && ver) { log(`OK ${pkgName}@${ver} 已存在（与本地一致）`); break }
    tries++
    try {
      execSync(`npm publish "${dir}/" --registry=${REGISTRY} --access public`, { timeout: 30000, encoding: 'utf8', stdio: ['ignore','pipe','ignore'] })
      log(`OK ${pkgName}@${want} 发布成功（第${tries}次尝试）`)
      break
    } catch (e) {
      log(`FAIL ${pkgName} 尝试${tries}: ${String(e.message||e).split('\n')[0]}`)
      await sleep(180000) // 3 分钟
    }
  }
  await sleep(20000)
}
log('=== 全部补发完成 ===')
