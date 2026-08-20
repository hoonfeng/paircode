// publish-retry-forever.mjs — 轮次制补发官方插件（限流多久都等，1 小时间隔）
// ★ 2026-08-20 增强（第 3 版）：
//   1. 开头自动重新打包（跑 publish-official-plugins.mjs 默认模式，从 .pair/plugins
//      复制最新内容到 .pair/publish/，避免发旧版 UI 资源）
//   2. 版本感知跳过：仅当 npm 已存在「相同版本」才跳过（本地 bump 过的新版本会正常发布）
//   3. ★ 轮次制：每轮把所有待发包各试一次（包间 20s 冷却）→ 失败的统一等 1 小时 → 下一轮
//      （旧版逐包等 1 小时，12 个包最坏 12 小时；轮次制 1 小时一轮，效率高得多）
//   4. ★ 修复 npmExists 的 CRLF bug：Windows cmd 下 curl 输出 \r\n，code 行带 \r
//      导致所有已存在包被误判为「未发布」→ 重复 publish 同版本失败
// 用法：node scripts/publish-retry-forever.mjs
import { execSync } from 'node:child_process'
import { readFileSync, existsSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const REGISTRY = 'https://registry.npmjs.org'
const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), '..')
const PUBLISH_DIR = join(REPO_ROOT, '.pair', 'publish')
const REMAIN = ['marketplace','ui-activitybar','ui-appearance','ui-editor','ui-modals','ui-quick-exec','ui-right-panel','ui-sidebar','ui-statusbar','ui-statusbar-conn','ui-titlebar','web-api']
const MAX_ROUNDS = 48 // 最多 48 轮（48 小时），仍不够可手动再跑
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
const log = (m) => console.log(new Date().toISOString() + ' ' + m)

// 查询 npm 已发布的最新版本（修 CRLF：curl 在 Windows cmd 输出 \r\n）
function npmExists(pkgName) {
  try {
    const out = execSync(`curl -s -w "\\n%{http_code}" "${REGISTRY}/-/package/${pkgName}/dist-tags"`, { encoding: 'utf8', stdio: ['ignore','pipe','ignore'] })
    const lines = out.trim().split('\n')
    const code = (lines[lines.length - 1] || '').trim() // ★ 必须 trim 掉 \r
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

// ── 步骤 2：轮次制补发 ──
log('══ 步骤 2/2：轮次制补发未发布/新版本包 ══')
for (let round = 1; round <= MAX_ROUNDS; round++) {
  // 本轮待发 = 未发布 或 版本不同
  const pending = REMAIN.filter((name) => {
    const ver = npmExists(`@paircode/${name}`)
    const local = localVersion(name)
    if (ver && ver === local) { log(`SKIP @paircode/${name}@${ver} 已存在（与本地一致）`); return false }
    return true
  })
  if (!pending.length) { log('=== 全部已发布，完成 ==='); break }
  log(`第 ${round} 轮：${pending.length} 个待发布`)
  const failed = []
  for (const name of pending) {
    const pkgName = `@paircode/${name}`
    const dir = join(PUBLISH_DIR, name)
    const want = localVersion(name) || '未知'
    try {
      execSync(`npm publish "${dir}/" --registry=${REGISTRY} --access public`, {
        timeout: 120000, encoding: 'utf8', stdio: ['ignore','pipe','ignore'],
      })
      log(`OK ${pkgName}@${want} 发布成功`)
    } catch (e) {
      // 提取真实错误（npm publish 的 stderr 被 ignore，错误在 message 摘要里）
      const msg = String(e.message || e).split('\n')[0]
      log(`FAIL ${pkgName}@${want}: ${msg}`)
      failed.push(name)
    }
    await sleep(20000) // 包间 20s 冷却
  }
  if (!failed.length) { log('=== 本轮全部成功，完成 ==='); break }
  const waitH = failed.length
  log(`剩余 ${failed.length} 个失败（${failed.join(', ')}），等待 1 小时后下一轮重试`)
  await sleep(3600000)
}
log('=== 补发循环结束 ===')
