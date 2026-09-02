// 聊天输入现代化验证（SheetPicker bottom-sheet 替换传统 select）
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
  await sleep(2000)
  // 强刷页面加载新区域 bundle
  await send('Page.reload', { ignoreCache: true })
  await sleep(5000)
  for (let i = 0; i < 20; i++) { const ok = await ev(`!!document.querySelector('.chat-input')`); if (ok) break; await sleep(1000) }
  // 1. 输入区无传统 select + 有 SheetPicker 触发条
  const r1 = await ev(`(() => {
    const area = document.querySelector('.chat-input-area')
    if (!area) return { ok: false, why: '无输入区' }
    const sel = area.querySelectorAll('select.cmp-sel, select.cmp-model, select.cmp-toolset')
    const triggers = area.querySelectorAll('.sp-trigger')
    const sendBtn = area.querySelector('.send-btn')
    const sendStyle = sendBtn ? getComputedStyle(sendBtn) : null
    return { ok: true, selectCount: sel.length, triggerCount: triggers.length,
      sendRound: sendStyle ? sendStyle.borderRadius : null, sendSize: sendStyle ? sendStyle.width : null,
      hasSelectAnywhere: !!area.querySelector('select') }
  })()`)
  console.log('1.输入区现代化:', JSON.stringify(r1))
  // 2. 点模型选择器触发条 → bottom-sheet 弹出
  const r2 = await ev(`(async () => {
    const area = document.querySelector('.chat-input-area')
    const triggers = area.querySelectorAll('.sp-trigger')
    if (!triggers.length) return { ok: false, why: '无触发条' }
    triggers[0].click(); await new Promise(r => setTimeout(r, 400))
    const sheet = document.querySelector('.sp-sheet')
    if (!sheet) return { ok: false, why: '无弹层', title: triggers[0].textContent }
    const groups = [...sheet.querySelectorAll('.sp-group')].map(g => g.textContent)
    const items = sheet.querySelectorAll('.sp-item').length
    const title = sheet.querySelector('.sp-title')?.textContent
    return { ok: true, title, groups: groups.slice(0, 4), groupsCount: groups.length, items, check: !!sheet.querySelector('.sp-check') }
  })()`)
  console.log('2.模型bottom-sheet:', JSON.stringify(r2))
  // 3. 选一个模型 → 触发条更新
  const r3 = await ev(`(async () => {
    const sheet = document.querySelector('.sp-sheet')
    if (!sheet) return { ok: false, why: '弹层已关' }
    const item = sheet.querySelectorAll('.sp-item')[0]
    if (!item) return { ok: false, why: '无选项' }
    const label = item.querySelector('.sp-item-label')?.textContent
    item.click(); await new Promise(r => setTimeout(r, 600))
    const area = document.querySelector('.chat-input-area')
    const triggers = area.querySelectorAll('.sp-trigger')
    const cur = triggers[0].textContent.trim()
    const sheetGone = !document.querySelector('.sp-sheet')
    return { ok: true, picked: label, triggerShows: cur, sheetClosed: sheetGone }
  })()`)
  console.log('3.选择模型:', JSON.stringify(r3))
  // 4. 工具集选择器
  const r4 = await ev(`(async () => {
    const area = document.querySelector('.chat-input-area')
    const triggers = area.querySelectorAll('.sp-trigger')
    if (triggers.length < 2) return { ok: false, why: '无工具集触发条', triggers: triggers.length }
    triggers[1].click(); await new Promise(r => setTimeout(r, 400))
    const sheet = document.querySelector('.sp-sheet')
    if (!sheet) return { ok: false, why: '工具集弹层未开' }
    const items = [...sheet.querySelectorAll('.sp-item')].map(i => i.querySelector('.sp-item-label')?.textContent)
    const title = sheet.querySelector('.sp-title')?.textContent
    const closeBtn = sheet.querySelector('.sp-cancel')
    return { ok: true, title, items: items.slice(0, 5) }
  })()`)
  console.log('4.工具集bottom-sheet:', JSON.stringify(r4))
  // 5. 取消关闭
  const r5 = await ev(`(async () => {
    const sheet = document.querySelector('.sp-sheet')
    if (!sheet) return { ok: true, closed: true, why: '已关' }
    const cancel = sheet.querySelector('.sp-cancel'); cancel.click()
    await new Promise(r => setTimeout(r, 300))
    return { ok: true, closed: !document.querySelector('.sp-sheet') }
  })()`)
  console.log('5.取消关闭:', JSON.stringify(r5))
  // 6. 发送按钮样式（圆形 FAB）
  const r6 = await ev(`(() => {
    const area = document.querySelector('.chat-input-area')
    const sendBtn = area.querySelector('.send-btn')
    if (!sendBtn) return { ok: false, why: '无发送按钮' }
    const s = getComputedStyle(sendBtn)
    return { ok: true, radius: s.borderRadius, w: s.width, h: s.height }
  })()`)
  console.log('6.发送按钮:', JSON.stringify(r6))
  console.log('7.控制台错误数:', errors.length)
  errors.slice(0, 6).forEach(e => console.log('   ', e))
  const failed = !r1.ok || r1.selectCount !== 0 || r1.triggerCount < 2 || !r2.ok || !r3.ok || !r4.ok || r4.items.length === 0 || !r5.closed || !r6.ok || r6.radius !== '50%'
  console.log(failed ? 'FAIL: 存在失败项' : 'PASS: 全部通过')
  process.exit(failed ? 1 : 0)
})().catch(e => { console.error('ERR:', e.message); process.exit(2) })
