// 聊天输入选择器验证（原生 <select> 科技风版：无 popover、optgroup 分组、appearance:none）
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
  for (let i = 0; i < 20; i++) { const ok = await ev(`!!document.querySelector('.chat-input-area')`); if (ok) break; await sleep(1000) }

  // 1. 输入区：原生 select 存在、无旧 popover 触发条/弹层
  const r1 = await ev(`(() => {
    const area = document.querySelector('.chat-input-area')
    if (!area) return { ok: false, why: '无输入区' }
    return { ok: true, selectCount: area.querySelectorAll('select.sp-select').length,
      triggerCount: area.querySelectorAll('.sp-trigger').length,
      popCount: document.querySelectorAll('.sp-pop').length }
  })()`)
  console.log('1.输入区:', JSON.stringify(r1))

  // 2. 模型 select：原生 + optgroup 分组 + appearance:none
  const r2 = await ev(`(() => {
    const sels = document.querySelectorAll('.chat-input-area select.sp-select')
    const sel = sels[0]
    if (!sel) return { ok: false, why: '无模型 select' }
    const cs = getComputedStyle(sel)
    const firstOpt = sel.querySelector('optgroup option') || sel.querySelector('option')
    return { ok: true, tag: sel.tagName, optgroups: sel.querySelectorAll('optgroup').length,
      opts: sel.querySelectorAll('option').length, appearance: cs.appearance, radius: cs.borderRadius,
      sample: firstOpt ? firstOpt.textContent.slice(0, 40) : '' }
  })()`)
  console.log('2.模型select:', JSON.stringify(r2))

  // 3. 工具集 select
  const r3 = await ev(`(() => {
    const sels = document.querySelectorAll('.chat-input-area select.sp-select')
    const sel = sels[1]
    if (!sel) return { ok: false, why: '无工具集 select', n: sels.length }
    const opt = sel.querySelector('option:not([value=""])')
    return { ok: true, tag: sel.tagName, opts: sel.querySelectorAll('option').length,
      sample: opt ? opt.textContent.slice(0, 40) : '' }
  })()`)
  console.log('3.工具集select:', JSON.stringify(r3))

  console.log('4.控制台错误数:', errors.length)
  errors.slice(0, 6).forEach(e => console.log('   ', e))

  const failed =
    !r1.ok || r1.selectCount < 1 || r1.triggerCount !== 0 || r1.popCount !== 0 ||
    !r2.ok || r2.tag !== 'SELECT' || r2.optgroups < 1 || r2.opts < 1 ||
    (r3.ok && r3.tag !== 'SELECT')
  console.log(failed ? 'FAIL: 存在失败项' : 'PASS: 全部通过')
  process.exit(failed ? 1 : 0)
})().catch(e => { console.error('ERR:', e.message); process.exit(2) })
