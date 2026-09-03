// 聊天输入选择器验证（SheetPicker 锚定 popover 版）
// 用法：node scripts/cdp-verify-chat-input.js <port>
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
  for (let i = 0; i < 20; i++) { const ok = await ev(`!!document.querySelector('.chat-input')`); if (ok) break; await sleep(1000) }
  // 1. 输入区无传统 select + 有 popover 触发条
  const r1 = await ev(`(() => {
    const area = document.querySelector('.chat-input-area')
    if (!area) return { ok: false, why: '无输入区' }
    return { ok: true, selectCount: area.querySelectorAll('select').length,
      triggerCount: area.querySelectorAll('.sp-trigger').length }
  })()`)
  console.log('1.输入区:', JSON.stringify(r1))
  // 2. 点模型触发条 → 锚定 popover 弹出 + 位置/尺寸断言
  const r2 = await ev(`(async () => {
    const area = document.querySelector('.chat-input-area')
    const tr = area.querySelectorAll('.sp-trigger')[0]
    tr.click(); await new Promise(r => setTimeout(r, 500))
    const pop = document.querySelector('.sp-pop')
    if (!pop) return { ok: false, why: '无 popover', triggerText: tr.textContent }
    const pr = pop.getBoundingClientRect()
    const br = tr.getBoundingClientRect()
    const groups = [...pop.querySelectorAll('.sp-group')].map(g => g.textContent)
    const items = pop.querySelectorAll('.sp-item').length
    const check = !!pop.querySelector('.sp-check')
    // 锚定断言：popover 顶部应贴近按钮底部（或上方翻转），水平与按钮重叠
    const anchoredV = Math.abs(pr.top - (br.bottom + 6)) < 40 || Math.abs((pr.bottom + 6) - br.top) < 40
    const anchoredH = !(pr.right < br.left - 20 || pr.left > br.right + 20)
    const small = pr.width <= 300 && pr.height <= 360
    return { ok: true, groups: groups.slice(0, 4), groupsCount: groups.length, items, check,
      popW: Math.round(pr.width), popH: Math.round(pr.height), anchoredV, anchoredH, small,
      popTop: Math.round(pr.top), btnBottom: Math.round(br.bottom) }
  })()`)
  console.log('2.模型popover:', JSON.stringify(r2))
  // 3. 选模型 → 触发条更新 + 关闭
  const r3 = await ev(`(async () => {
    const pop = document.querySelector('.sp-pop')
    if (!pop) return { ok: false, why: '弹层已关' }
    const item = pop.querySelectorAll('.sp-item')[0]
    if (!item) return { ok: false, why: '无选项' }
    const label = item.querySelector('.sp-item-label')?.textContent
    item.click(); await new Promise(r => setTimeout(r, 500))
    const area = document.querySelector('.chat-input-area')
    const cur = area.querySelectorAll('.sp-trigger')[0].textContent.trim()
    return { ok: true, picked: label, triggerShows: cur, closed: !document.querySelector('.sp-pop') }
  })()`)
  console.log('3.选模型:', JSON.stringify(r3))
  // 4. 工具集选择器 popover
  const r4 = await ev(`(async () => {
    const area = document.querySelector('.chat-input-area')
    const trs = area.querySelectorAll('.sp-trigger')
    if (trs.length < 2) return { ok: false, why: '无工具集触发条', n: trs.length }
    trs[1].click(); await new Promise(r => setTimeout(r, 500))
    const pop = document.querySelector('.sp-pop')
    if (!pop) return { ok: false, why: '工具集弹层未开' }
    const items = [...pop.querySelectorAll('.sp-item')].map(i => i.querySelector('.sp-item-label')?.textContent)
    return { ok: true, items: items.slice(0, 6), n: items.length }
  })()`)
  console.log('4.工具集popover:', JSON.stringify(r4))
  // 5. 点击遮罩关闭
  const r5 = await ev(`(async () => {
    const mask = document.querySelector('.sp-pop-mask')
    if (!mask) return { ok: false, why: '无遮罩' }
    mask.click(); await new Promise(r => setTimeout(r, 300))
    return { ok: true, closed: !document.querySelector('.sp-pop') }
  })()`)
  console.log('5.遮罩关闭:', JSON.stringify(r5))
  // 6. Esc 关闭
  const r6 = await ev(`(async () => {
    const area = document.querySelector('.chat-input-area')
    area.querySelectorAll('.sp-trigger')[0].click(); await new Promise(r => setTimeout(r, 400))
    const opened = !!document.querySelector('.sp-pop')
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await new Promise(r => setTimeout(r, 300))
    return { ok: true, opened, escClosed: !document.querySelector('.sp-pop') }
  })()`)
  console.log('6.Esc关闭:', JSON.stringify(r6))
  console.log('7.控制台错误数:', errors.length)
  errors.slice(0, 6).forEach(e => console.log('   ', e))
  const failed = !r1.ok || r1.selectCount !== 0 || r1.triggerCount < 2 || !r2.ok || !r2.anchoredV || !r2.anchoredH || !r2.small || !r3.ok || r3.closed !== true || !r4.ok || r4.n === 0 || !r5.closed || !r6.ok || r6.escClosed !== true
  console.log(failed ? 'FAIL: 存在失败项' : 'PASS: 全部通过')
  process.exit(failed ? 1 : 0)
})().catch(e => { console.error('ERR:', e.message); process.exit(2) })
