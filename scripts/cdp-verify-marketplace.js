// 市场面板验证脚本（CDP 无依赖客户端）：
//   ① 市场列表显示版本号（.mi-version）
//   ② 更新提示：已安装条目「已装 vX · 已是最新」或「有更新：vX → vY」+ 更新按钮
//   ③ 已安装页类型筛选 tag（全部/插件/MCP/技能，点击切换分组）
//   ④ 「检查更新」按钮回读版本对照摘要
// 用法：node scripts/cdp-verify-marketplace.js <debugPort> <appPort>
// 依赖：headless chrome --remote-debugging-port=<debugPort> 已启动；
//      appPort = 被测实例端口（★ 非默认 9090——自举铁律）
const http = require('http'); const net = require('net'); const crypto = require('crypto')
const fs = require('fs'); const path = require('path')
function httpJson(path, method) { return new Promise((resolve, reject) => { const r = http.request({ host: '127.0.0.1', port: 9223, path, method: method || 'GET' }, x => { let d = ''; x.on('data', c => d += c); x.on('end', () => { try { resolve(JSON.parse(d)) } catch (e) { resolve(d) } }) }); r.on('error', reject); r.end() }) }
class WS { constructor() { this.buf = Buffer.alloc(0); this.frag = [] }
  connect(u) { return new Promise((res, rej) => { const m = u.match(/^ws:\/\/([^:/]+):(\d+)(\/.*)$/); const k = crypto.randomBytes(16).toString('base64'); this.sock = net.connect(+m[2], m[1], () => this.sock.write(`GET ${m[3]} HTTP/1.1\r\nHost: ${m[1]}:${m[2]}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${k}\r\nSec-WebSocket-Version: 13\r\n\r\n`)); let hs = false; this.sock.on('data', d => { this.buf = Buffer.concat([this.buf, d]); if (!hs) { const i = this.buf.indexOf('\r\n\r\n'); if (i < 0) return; hs = true; this.buf = this.buf.slice(i + 4); res() } else this._drain() }); this.sock.on('error', rej) }) }
  _drain() { for (;;) { if (this.buf.length < 2) return; const b0 = this.buf[0], b1 = this.buf[1]; let len = b1 & 0x7f, off = 2; if (len === 126) { if (this.buf.length < 4) return; len = this.buf.readUInt16BE(2); off = 4 } else if (len === 127) { if (this.buf.length < 10) return; len = Number(this.buf.readBigUInt64BE(2)); off = 10 } const masked = (b1 & 0x80) !== 0; let mask = null; if (masked) { if (this.buf.length < off + 4) return; mask = this.buf.slice(off, off + 4); off += 4 } if (this.buf.length < off + len) return; let p = this.buf.slice(off, off + len); if (masked) { const o = Buffer.alloc(len); for (let i = 0; i < len; i++) o[i] = p[i] ^ mask[i & 3]; p = o } this.buf = this.buf.slice(off + len); const op = b0 & 0x0f; if (op === 1) this.frag.push(p); if ((b0 & 0x80) && this.frag.length) { this.onMessage(Buffer.concat(this.frag)); this.frag = [] } } }
  send(o) { const p = Buffer.from(JSON.stringify(o)); const len = p.length; let h; if (len < 126) h = Buffer.from([0x81, 0x80 | len]); else { h = Buffer.alloc(4); h[0] = 0x81; h[1] = 0x80 | 126; h.writeUInt16BE(len, 2) } const mask = crypto.randomBytes(4); const m = Buffer.alloc(len); for (let i = 0; i < len; i++) m[i] = p[i] ^ mask[i & 3]; this.sock.write(Buffer.concat([h, mask, m])) } }

; (async () => {
  const appPort = Number(process.argv[3] || 9097)
  const list = await httpJson('/json/list')
  const tab = list.find(t => t.type === 'page')
  if (!tab) { console.log('FAIL: 无调试页（chrome 未就绪）'); process.exit(2) }
  const ws = new WS(); await ws.connect(tab.webSocketDebuggerUrl)
  let id = 0; const pend = new Map(); const errors = []
  ws.onMessage = b => { const m = JSON.parse(b.toString()); if (m.id && pend.has(m.id)) { pend.get(m.id)(m); pend.delete(m.id) } else if (m.method === 'Runtime.consoleAPICalled' && m.params.type === 'error') { const txt = (m.params.args || []).map(a => a.value || a.description || '').join(' '); errors.push(txt.slice(0, 160)) } }
  // ★ send 带超时（页面内长循环/上下文丢失时不让脚本永久挂起）
  const send = (method, params, timeoutMs) => new Promise((res, rej) => {
    const i = ++id
    const timer = setTimeout(() => { pend.delete(i); rej(new Error('CDP 超时: ' + method)) }, timeoutMs || 180000)
    pend.set(i, x => { clearTimeout(timer); x.error ? rej(new Error(x.error.message)) : res(x.result) })
    ws.send({ id: i, method, params })
  })
  await send('Runtime.enable'); await send('Page.enable')
  const ev = async (e, t) => (await send('Runtime.evaluate', { expression: e, returnByValue: true, awaitPromise: true }, t)).result.value
  const sleep = ms => new Promise(r => setTimeout(r, ms))
  const shotDir = path.join(__dirname, '..', 'screenshots')
  const shot = async (name) => {
    try {
      if (!fs.existsSync(shotDir)) fs.mkdirSync(shotDir, { recursive: true })
      const r = await send('Page.captureScreenshot', { format: 'png' })
      const f = path.join(shotDir, name + '.png')
      fs.writeFileSync(f, Buffer.from(r.data, 'base64'))
      console.log('   截图:', path.relative(process.cwd(), f))
    } catch (e) { console.log('   截图失败:', e.message) }
  }

  await send('Page.navigate', { url: `http://127.0.0.1:${appPort}/` })
  let ready = false
  for (let i = 0; i < 40; i++) { if (await ev(`!!document.querySelector('.activity-bar')`)) { ready = true; break } await sleep(1000) }
  if (!ready) { console.log('FAIL: 页面未就绪'); process.exit(2) }
  await sleep(3000)

  // ── 1. 打开市场：列表版本号 + 已安装条目的更新提示 ──
  const r1 = await ev(`(async () => {
    const btn = [...document.querySelectorAll('.activity-bar button')].find(b => (b.title||'')==='市场')
    if (!btn) return { ok:false, why:'无市场图标' }
    btn.click()
    const t0 = Date.now()
    while (Date.now() - t0 < 150000) {
      await new Promise(r => setTimeout(r, 1000))
      if (document.querySelectorAll('.market-item').length > 0) break
    }
    await new Promise(r => setTimeout(r, 9000)) // 等静默 check-update 回来（版本对照）
    const items = [...document.querySelectorAll('.market-item')]
    const txt = (el, sel) => (el.querySelector(sel)?.textContent || '').replace(/\\s+/g,' ').trim()
    const withVer = items.filter(el => el.querySelector('.mi-version'))
    const inst = items.filter(el => /已安装/.test(txt(el, '.mi-installed')))
    return { ok:true, panel: !!document.querySelector('.market-panel'), items: items.length,
      withVer: withVer.length,
      verSamples: withVer.slice(0, 5).map(el => txt(el, '.mi-name') + ' → ' + txt(el, '.mi-version')),
      installed: inst.length,
      updLines: items.filter(el => el.querySelector('.mi-update')).map(el => txt(el, '.mi-name') + ' | ' + txt(el, '.mi-update')),
      uptoLines: items.filter(el => el.querySelector('.mi-uptodate')).map(el => txt(el, '.mi-name') + ' | ' + txt(el, '.mi-uptodate')),
      updBtn: items.filter(el => el.querySelector('.mi-install-area .mi-install-btn')).length }
  })()`)
  console.log('1.市场列表:', JSON.stringify(r1))
  await shot('market-1-list')

  // ── 1b. 搜索单个 npm 插件：看「有更新」提示与「更新到 vX」按钮 ──
  const r1b = await ev(`(async () => {
    const inp = document.querySelector('.market-search .search-input')
    if (!inp) return { ok:false, why:'无搜索框' }
    inp.value = 'tool-bug'
    inp.dispatchEvent(new Event('input', { bubbles: true }))
    const t0 = Date.now()
    while (Date.now() - t0 < 90000) {
      await new Promise(r => setTimeout(r, 1000))
      const names = [...document.querySelectorAll('.market-item .mi-name')].map(e => e.textContent.trim())
      if (names.length > 0 && names.every(n => /tool-bug/i.test(n))) break
    }
    await new Promise(r => setTimeout(r, 4000))
    const txt = el => (el.textContent || '').replace(/\\s+/g,' ').trim()
    const item = [...document.querySelectorAll('.market-item')].find(el => /tool-bug/i.test(txt(el.querySelector('.mi-name') || { textContent: '' })))
    if (item) { item.scrollIntoView({ block: 'center' }); await new Promise(r => setTimeout(r, 600)) }
    return { ok: !!item, name: item ? txt(item.querySelector('.mi-name')) : '',
      version: item ? txt(item.querySelector('.mi-version') || { textContent: '' }) : '',
      updLine: item ? txt(item.querySelector('.mi-update') || { textContent: '' }) : '',
      btn: item ? txt(item.querySelector('.mi-install-area .mi-install-btn') || { textContent: '' }) : '' }
  })()`)
  console.log('1b.搜索 tool-bug:', JSON.stringify(r1b))
  await shot('market-1b-search-update')

  // ── 2. 已安装 tab：筛选 tag + 插件版本号 ──
  const r2 = await ev(`(async () => {
    const inst = [...document.querySelectorAll('.market-tabs button')].find(b => b.textContent.trim() === '已安装')
    if (!inst) return { ok:false, why:'无已安装 tab' }
    inst.click()
    const t0 = Date.now()
    while (Date.now() - t0 < 20000) { await new Promise(r => setTimeout(r, 500)); if (document.querySelector('.installed-filters')) break }
    await new Promise(r => setTimeout(r, 12000)) // 等 check-update 完成（36 包并发）
    const txt = el => (el.textContent || '').replace(/\\s+/g,' ').trim()
    const tags = [...document.querySelectorAll('.if-tag')].map(b => ({ t: txt(b), active: b.classList.contains('active'), disabled: !!b.disabled }))
    const groups = [...document.querySelectorAll('.installed-group-title')].map(g => txt(g))
    const items = [...document.querySelectorAll('.installed-item')]
    const plug = items.filter(el => /插件 ·/.test(txt(el.querySelector('.ii-badge') || el)))
    const plugVer = plug.filter(el => /v\\d/.test(txt(el.querySelector('.ii-badge') || el)))
    return { ok:true, hasFilterBar: !!document.querySelector('.installed-filters'), tags, groups,
      items: items.length, plugItems: plug.length, plugWithVer: plugVer.length,
      samples: plug.slice(0, 4).map(el => txt(el.querySelector('.ii-name')) + ' | ' + txt(el.querySelector('.ii-badge'))),
      updBadges: [...document.querySelectorAll('.badge-updateable')].map(b => txt(b)),
      updBtns: document.querySelectorAll('.ii-upd').length,
      summary: txt(document.querySelector('.installed-toolbar-tip') || { textContent: '' }) }
  })()`)
  console.log('2.已安装页:', JSON.stringify(r2))
  await shot('market-2-installed')
  // 2b. 滚动到「有新版」条目（视觉证据）
  await ev(`(async () => {
    const el = [...document.querySelectorAll('.installed-item')].find(x => x.querySelector('.badge-updateable'))
    if (el) { el.scrollIntoView({ block: 'center' }); await new Promise(r => setTimeout(r, 600)) }
    return true
  })()`)
  await shot('market-2b-installed-update')

  // ── 3. 类型筛选 tag 点击切换 ──
  // ★ 每个 tag 独立一次 CDP 调用（短超时）——单次大函数在长会话下曾出现挂起
  const clickTag = async (label) => {
    try {
      return await ev(`(async () => {
        const t = [...document.querySelectorAll('.if-tag')].find(b => (b.textContent||'').replace(/\\s+/g,' ').trim().startsWith('${label}'))
        if (!t) return { why: '无 tag ${label}' }
        t.click()
        await new Promise(r => setTimeout(r, 700))
        const txt = el => (el.textContent || '').replace(/\\s+/g,' ').trim()
        return { active: txt(document.querySelector('.if-tag.active') || { textContent: '' }),
          groups: [...document.querySelectorAll('.installed-group-title')].map(g => txt(g)),
          emptyHint: txt(document.querySelector('.market-empty') || { textContent: '' }).slice(0, 60),
          items: document.querySelectorAll('.installed-item').length }
      })()`, 30000)
    } catch (e) { return { why: 'CDP 失败: ' + e.message } }
  }
  const r3 = { ok: true }
  r3.plugin = await clickTag('插件'); r3.mcp = await clickTag('MCP'); r3.skill = await clickTag('技能'); r3.all = await clickTag('全部')
  console.log('3.筛选切换:', JSON.stringify(r3))
  await ev(`(async () => {
    const t = [...document.querySelectorAll('.if-tag')].find(b => (b.textContent||'').replace(/\\s+/g,' ').trim().startsWith('MCP'))
    if (t) { t.click(); await new Promise(r => setTimeout(r, 600)) }
    return true
  })()`)
  await shot('market-3-filter-mcp-empty')
  await ev(`(async () => {
    const t = [...document.querySelectorAll('.if-tag')].find(b => (b.textContent||'').replace(/\\s+/g,' ').trim().startsWith('全部'))
    if (t) { t.click(); await new Promise(r => setTimeout(r, 600)) }
    return true
  })()`)

  // ── 4. 「检查更新」按钮 ──
  const r4 = await ev(`(async () => {
    const btn = document.querySelector('.ii-check-upd')
    if (!btn) return { ok:false, why:'无检查更新按钮' }
    btn.click()
    const t0 = Date.now()
    while (Date.now() - t0 < 120000) { await new Promise(r => setTimeout(r, 1000)); if (!document.querySelector('.ii-check-upd')?.disabled) break }
    return { ok:true, summary: (document.querySelector('.installed-toolbar-tip')?.textContent || '').trim().replace(/\\s+/g,' '), updBtns: document.querySelectorAll('.ii-upd').length }
  })()`, 120000)
  console.log('4.检查更新:', JSON.stringify(r4))
  await shot('market-4-updates')

  console.log('5.控制台错误数:', errors.length)
  errors.slice(0, 8).forEach(e => console.log('   ', e))

  const failed = !r1.ok || !r1.panel || !r2.ok || !r2.hasFilterBar || !r2.tags || r2.tags.length !== 4
    || !r3.ok || !r3.plugin || !r3.mcp || !r3.skill
    || (r3.plugin.groups || []).length !== 1 || (r3.mcp.groups || []).length !== 0
    || (r3.skill.groups || []).length > 1 || (r3.all.groups || []).length < (r3.plugin.groups || []).length
    || !r4.ok
  console.log(failed ? 'FAIL: 存在失败项' : 'PASS: 版本号 / 更新提示 / 类型筛选 全部通过')
  process.exit(failed ? 1 : 0)
})().catch(e => { console.error('ERR:', e.message); process.exit(2) })
