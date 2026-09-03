// 添加插件浮层截图（打开工具集 tab → 选非内置集 → 点「添加插件」→ 截浮层）
// 用法：node scripts/cdp-shot-add-plugin.js <out.png> [searchTerm]
const http = require('http'); const net = require('net'); const crypto = require('crypto'); const fs = require('fs')
function httpJson(p) { return new Promise((res, rej) => { http.get({ host: '127.0.0.1', port: 9223, path: p }, x => { let d = ''; x.on('data', c => d += c); x.on('end', () => res(JSON.parse(d))) }).on('error', rej) }) }
class WS { constructor() { this.buf = Buffer.alloc(0); this.frag = [] }
  connect(u) { return new Promise((res, rej) => { const m = u.match(/^ws:\/\/([^:/]+):(\d+)(\/.*)$/); const k = crypto.randomBytes(16).toString('base64'); this.sock = net.connect(+m[2], m[1], () => this.sock.write(`GET ${m[3]} HTTP/1.1\r\nHost: ${m[1]}:${m[2]}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${k}\r\nSec-WebSocket-Version: 13\r\n\r\n`)); let hs = false; this.sock.on('data', d => { this.buf = Buffer.concat([this.buf, d]); if (!hs) { const i = this.buf.indexOf('\r\n\r\n'); if (i < 0) return; hs = true; this.buf = this.buf.slice(i + 4); res() } else this._drain() }); this.sock.on('error', rej) }) }
  _drain() { for (;;) { if (this.buf.length < 2) return; const b0 = this.buf[0], b1 = this.buf[1]; let len = b1 & 0x7f, off = 2; if (len === 126) { if (this.buf.length < 4) return; len = this.buf.readUInt16BE(2); off = 4 } else if (len === 127) { if (this.buf.length < 10) return; len = Number(this.buf.readBigUInt64BE(2)); off = 10 } const masked = (b1 & 0x80) !== 0; let mask = null; if (masked) { if (this.buf.length < off + 4) return; mask = this.buf.slice(off, off + 4); off += 4 } if (this.buf.length < off + len) return; let p = this.buf.slice(off, off + len); if (masked) { const o = Buffer.alloc(len); for (let i = 0; i < len; i++) o[i] = p[i] ^ mask[i & 3]; p = o } this.buf = this.buf.slice(off + len); const op = b0 & 0x0f; if (op === 1) this.frag.push(p); if ((b0 & 0x80) && this.frag.length) { this.onMessage(Buffer.concat(this.frag)); this.frag = [] } } }
  send(o) { const p = Buffer.from(JSON.stringify(o)); const len = p.length; let h; if (len < 126) h = Buffer.from([0x81, 0x80 | len]); else { h = Buffer.alloc(4); h[0] = 0x81; h[1] = 0x80 | 126; h.writeUInt16BE(len, 2) } const mask = crypto.randomBytes(4); const m = Buffer.alloc(len); for (let i = 0; i < len; i++) m[i] = p[i] ^ mask[i & 3]; this.sock.write(Buffer.concat([h, mask, m])) } }
;(async () => {
  const list = await httpJson('/json/list'); const tab = list.find(t => t.type === 'page')
  const ws = new WS(); await ws.connect(tab.webSocketDebuggerUrl)
  let id = 0; const pend = new Map()
  ws.onMessage = b => { const m = JSON.parse(b.toString()); if (m.id && pend.has(m.id)) { pend.get(m.id)(m); pend.delete(m.id) } }
  const send = (method, params) => new Promise((res, rej) => { const i = ++id; pend.set(i, x => x.error ? rej(new Error(x.error.message)) : res(x.result)); ws.send({ id: i, method, params }) })
  const ev = async e => (await send('Runtime.evaluate', { expression: e, returnByValue: true, awaitPromise: true })).result.value
  const out = process.argv[2] || '.chrome-test/add-plugin.png'
  const search = process.argv[3] || ''
  await ev(`(async () => { const b = [...document.querySelectorAll('.activity-bar button')].find(x => (x.title||'')==='工具集'); if (b) b.click(); await new Promise(r => setTimeout(r, 1500)); const items = [...document.querySelectorAll('.tp-item')]; const nb = items.find(x => !x.classList.contains('builtin')); if (nb) nb.click(); await new Promise(r => setTimeout(r, 1200)); const add = document.querySelector('.tp-add .tp-add-btn'); if (add) add.click(); await new Promise(r => setTimeout(r, 700)); return 1 })()`)
  if (search) {
    await ev(`(async () => { const inp = document.querySelector('.tp-add-search-input'); if (inp) { inp.value = ${JSON.stringify(search)}; inp.dispatchEvent(new Event('input', { bubbles: true })); } await new Promise(r => setTimeout(r, 500)); return 1 })()`)
  }
  const shot = await send('Page.captureScreenshot', { format: 'png' })
  fs.writeFileSync(out, Buffer.from(shot.data, 'base64'))
  console.log('saved', out, fs.statSync(out).size)
  process.exit(0)
})().catch(e => { console.error(e.message); process.exit(1) })
