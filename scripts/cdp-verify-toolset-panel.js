// 工具集面板验证（左右分栏 master-detail + 添加插件浮层卡片 + 居中 modal + 无 checkbox chip）
// 用法：node scripts/cdp-verify-toolset-panel.js <port>
const http = require('http'); const net = require('net'); const crypto = require('crypto')
function httpJson(path, method) { return new Promise((resolve, reject) => { const r = http.request({ host: '127.0.0.1', port: 9223, path, method: method || 'GET' }, x => { let d = ''; x.on('data', c => d += c); x.on('end', () => { try { resolve(JSON.parse(d)) } catch (e) { resolve(d) } }) }); r.on('error', reject); r.end() }) }
class WS { constructor() { this.buf = Buffer.alloc(0); this.frag = [] }
  connect(u) { return new Promise((res, rej) => { const m = u.match(/^ws:\/\/([^:/]+):(\d+)(\/.*)$/); const k = crypto.randomBytes(16).toString('base64'); this.sock = net.connect(+m[2], m[1], () => this.sock.write(`GET ${m[3]} HTTP/1.1\r\nHost: ${m[1]}:${m[2]}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${k}\r\nSec-WebSocket-Version: 13\r\n\r\n`)); let hs = false; this.sock.on('data', d => { this.buf = Buffer.concat([this.buf, d]); if (!hs) { const i = this.buf.indexOf('\r\n\r\n'); if (i < 0) return; hs = true; this.buf = this.buf.slice(i + 4); res() } else this._drain() }); this.sock.on('error', rej) }) }
  _drain() { for (;;) { if (this.buf.length < 2) return; const b0 = this.buf[0], b1 = this.buf[1]; let len = b1 & 0x7f, off = 2; if (len === 126) { if (this.buf.length < 4) return; len = this.buf.readUInt16BE(2); off = 4 } else if (len === 127) { if (this.buf.length < 10) return; len = Number(this.buf.readBigUInt64BE(2)); off = 10 } const masked = (b1 & 0x80) !== 0; let mask = null; if (masked) { if (this.buf.length < off + 4) return; mask = this.buf.slice(off, off + 4); off += 4 } if (this.buf.length < off + len) return; let p = this.buf.slice(off, off + len); if (masked) { const o = Buffer.alloc(len); for (let i = 0; i < len; i++) o[i] = p[i] ^ mask[i & 3]; p = o } this.buf = this.buf.slice(off + len); const op = b0 & 0x0f; if (op === 1) this.frag.push(p); if ((b0 & 0x80) && this.frag.length) { this.onMessage(Buffer.concat(this.frag)); this.frag = [] } } }
  send(o) { const p = Buffer.from(JSON.stringify(o)); const len = p.length; let h; if (len < 126) h = Buffer.from([0x81, 0x80 | len]); else { h = Buffer.alloc(4); h[0] = 0x81; h[1] = 0x80 | 126; h.writeUInt16BE(len, 2) } const mask = crypto.randomBytes(4); const m = Buffer.alloc(len); for (let i = 0; i < len; i++) m[i] = p[i] ^ mask[i & 3]; this.sock.write(Buffer.concat([h, mask, m])) } }
;(async () => {
  const list = await httpJson('/json/list')
  const tab = list.find(t => t.type === 'page')
  const ws = new WS(); await ws.connect(tab.webSocketDebuggerUrl)
  let id = 0; const pend = new Map(); const errors = []
  ws.onMessage = b => { const m = JSON.parse(b.toString()); if (m.id && pend.has(m.id)) { pend.get(m.id)(m); pend.delete(m.id) } else if (m.method === 'Runtime.consoleAPICalled' && m.params.type === 'error') { const txt = (m.params.args || []).map(a => a.value || a.description || '').join(' '); errors.push(txt.slice(0, 140)) } }
  const send = (method, params) => new Promise((res, rej) => { const i = ++id; pend.set(i, x => x.error ? rej(new Error(x.error.message)) : res(x.result)); ws.send({ id: i, method, params }) })
  await send('Runtime.enable')
  const ev = async e => (await send('Runtime.evaluate', { expression: e, returnByValue: true, awaitPromise: true })).result.value
  const sleep = ms => new Promise(r => setTimeout(r, ms))
  for (let i = 0; i < 20; i++) { const ok = await ev(`!!document.querySelector('.activity-bar')`); if (ok) break; await sleep(1000) }
  await sleep(1500)
  await send('Page.reload', { ignoreCache: true })
  await sleep(4500)
  for (let i = 0; i < 20; i++) { const ok = await ev(`!!document.querySelector('.activity-bar')`); if (ok) break; await sleep(1000) }
  // 打开工具集 tab
  await ev(`(async () => { const b = [...document.querySelectorAll('.activity-bar button')].find(x => (x.title||'')==='工具集'); if (b) b.click(); await new Promise(r => setTimeout(r, 1800)); return 1 })()`)

  // 1. 左右分栏 master-detail（先不选中，检查结构）
  const r1 = await ev(`(async () => {
    const body = document.querySelector('.tp-body')
    const list = document.querySelector('.tp-list')
    if (!body || !list) return { ok: false, why: '无分栏结构' }
    return { ok: true, flexDirection: getComputedStyle(body).flexDirection, listWidth: Math.round(list.getBoundingClientRect().width) }
  })()`)
  console.log('1.分栏结构:', JSON.stringify(r1))

  // 2. 选中一个工具集 → 检查工具 chip 无 checkbox + 左右排列
  const r2 = await ev(`(async () => {
    const items = document.querySelectorAll('.tp-item')
    if (!items.length) return { ok: false, why: '无工具集列表' }
    items[0].click(); await new Promise(r => setTimeout(r, 1200))
    const tools = document.querySelectorAll('.tp-tool')
    const hasCheckbox = !!document.querySelector('.tp-tool input[type=checkbox]')
    const isBtn = tools.length ? tools[0].tagName === 'BUTTON' : null
    // 左右排列：列表右边界 <= 详情左边界，且顶部对齐
    const list = document.querySelector('.tp-list')
    const detail = document.querySelector('.tp-detail')
    let horizontal = false
    if (list && detail) {
      const lr = list.getBoundingClientRect(); const dr = detail.getBoundingClientRect()
      horizontal = lr.left < dr.left && Math.abs(lr.top - dr.top) < 20
    }
    return { ok: true, toolCount: tools.length, hasCheckbox, firstTag: tools.length ? tools[0].tagName : null, horizontal }
  })()`)
  console.log('2.工具chip:', JSON.stringify(r2))

  // 3. 新建弹层居中 modal
  const r3 = await ev(`(async () => {
    const btn = [...document.querySelectorAll('.tp-icon-btn')].find(b => (b.title||'').includes('新建'))
    if (!btn) return { ok: false, why: '无新建按钮' }
    btn.click(); await new Promise(r => setTimeout(r, 600))
    const ov = document.querySelector('.tp-overlay')
    const sheet = document.querySelector('.tp-sheet')
    if (!ov || !sheet) return { ok: false, why: '无弹层' }
    const os = getComputedStyle(ov)
    const sr = sheet.getBoundingClientRect()
    const vh = window.innerHeight; const vw = window.innerWidth
    const centeredH = Math.abs((sr.left + sr.width / 2) - vw / 2) < 60
    const centeredV = Math.abs((sr.top + sr.height / 2) - vh / 2) < 120
    const grabber = !!sheet.querySelector('.tp-grabber')
    const radius = getComputedStyle(sheet).borderRadius
    return { ok: true, align: os.alignItems, centeredH, centeredV, grabber, radius, w: Math.round(sr.width), h: Math.round(sr.height) }
  })()`)
  console.log('3.新建modal:', JSON.stringify(r3))

  // 4. 关闭新建 → 添加插件浮层卡片（非 select/popover）
  const r4 = await ev(`(async () => {
    const cancel = document.querySelector('.tp-sheet .tp-cancel')
    if (cancel) cancel.click()
    await new Promise(r => setTimeout(r, 400))
    // 选一个非内置工具集（有添加插件入口）
    const items = [...document.querySelectorAll('.tp-item')]
    const nonBuiltin = items.find(x => !x.classList.contains('builtin'))
    if (nonBuiltin) { nonBuiltin.click(); await new Promise(r => setTimeout(r, 1200)) }
    const addBtn = document.querySelector('.tp-add .tp-add-btn')
    if (!addBtn) return { ok: true, skipped: true, why: '无添加插件入口（仅内置集）' }
    const popover = document.querySelector('.sp-pop')
    addBtn.click(); await new Promise(r => setTimeout(r, 600))
    const sheet = document.querySelector('.tp-add-sheet')
    if (!sheet) return { ok: false, why: '点击添加插件后无浮层' }
    const cards = sheet.querySelectorAll('.tp-pcard')
    const grid = sheet.querySelector('.tp-add-grid')
    const hasSearch = !!sheet.querySelector('.tp-add-search-input')
    const first = cards[0]
    const firstInfo = first ? {
      name: (first.querySelector('.tp-pcard-name') || {}).textContent || '',
      count: (first.querySelector('.tp-pcard-count') || {}).textContent || '',
      desc: !!(first.querySelector('.tp-pcard-desc')),
    } : null
    return { ok: true, hasGrid: !!grid, cardCount: cards.length, hasSearch, popoverStill: !!popover, firstInfo }
  })()`)
  console.log('4.添加插件浮层:', JSON.stringify(r4))

  console.log('5.控制台错误数:', errors.length)
  errors.slice(0, 6).forEach(e => console.log('   ', e))

  const failed =
    !r1.ok || r1.flexDirection !== 'row' ||
    !r2.ok || r2.hasCheckbox !== false || (r2.toolCount > 0 && r2.firstTag !== 'BUTTON') || r2.horizontal !== true ||
    !r3.ok || r3.align !== 'center' || r3.grabber !== false || !r3.centeredH ||
    !r4.ok || (r4.skipped ? false : (r4.hasGrid !== true || r4.hasSearch !== true || r4.popoverStill === true))
  console.log(failed ? 'FAIL: 存在失败项' : 'PASS: 全部通过')
  process.exit(failed ? 1 : 0)
})().catch(e => { console.error('ERR:', e.message); process.exit(2) })
