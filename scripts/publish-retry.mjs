// publish-retry.mjs — 冷却后低频补发剩余官方插件（限流容错重试）
import { execSync } from 'node:child_process'
const REGISTRY = 'https://registry.npmjs.org'
const PUBLISH_DIR = 'F:/syproject/gou-ide/.pair/publish'
const REMAIN = ['ui-appearance','ui-editor','ui-modals','ui-quick-exec','ui-right-panel','ui-sidebar','ui-statusbar','ui-statusbar-conn','ui-titlebar','web-api']
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
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
  const ver = npmExists(pkgName)
  if (ver) { console.log(`${new Date().toISOString()} SKIP ${pkgName}@${ver} 已存在`); continue }
  let ok = false
  for (let i = 1; i <= 6; i++) {
    try {
      execSync(`npm publish "${dir}/" --registry=${REGISTRY} --access public`, { timeout: 30000, encoding: 'utf8', stdio: ['ignore','pipe','ignore'] })
      console.log(`${new Date().toISOString()} OK ${pkgName} 发布成功`)
      ok = true; break
    } catch (e) {
      console.log(`${new Date().toISOString()} FAIL ${pkgName} 尝试${i}/6: ${String(e.message||e).split('\n')[0]}`)
      if (i < 6) await sleep(150000)
    }
  }
  if (!ok) console.log(`${new Date().toISOString()} GIVEUP ${pkgName}`)
  await sleep(20000)
}
console.log('=== 重试结束 ===')
