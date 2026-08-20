// publish-retry-forever.mjs — 无限重试补发剩余官方插件（限流多久都等）
// 每个包：每 3 分钟重试一次直到发布成功；全部完成后结束。
import { execSync } from 'node:child_process'
const REGISTRY = 'https://registry.npmjs.org'
const PUBLISH_DIR = 'F:/syproject/gou-ide/.pair/publish'
const REMAIN = ['ui-appearance','ui-editor','ui-modals','ui-quick-exec','ui-right-panel','ui-sidebar','ui-statusbar','ui-statusbar-conn','ui-titlebar','web-api']
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
for (const name of REMAIN) {
  const pkgName = `@paircode/${name}`
  const dir = `${PUBLISH_DIR}/${name}`
  let tries = 0
  while (true) {
    const ver = npmExists(pkgName)
    if (ver) { log(`OK ${pkgName}@${ver} 已存在`); break }
    tries++
    try {
      execSync(`npm publish "${dir}/" --registry=${REGISTRY} --access public`, { timeout: 30000, encoding: 'utf8', stdio: ['ignore','pipe','ignore'] })
      log(`OK ${pkgName} 发布成功（第${tries}次尝试）`)
      break
    } catch (e) {
      log(`FAIL ${pkgName} 尝试${tries}: ${String(e.message||e).split('\n')[0]}`)
      await sleep(180000) // 3 分钟
    }
  }
  await sleep(20000)
}
log('=== 全部补发完成 ===')
