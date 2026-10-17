// 探针：任务看板详情 textarea 逐字符输入时 DOM 是否被重建 / 焦点是否丢失
// 用法：node scripts/cdp-probe-taskboard-input.js [port]
const http = require('http'); const net = require('net'); const crypto = require('crypto')
const PORT = Number(process.argv[2] || 9224)
function httpJson(path, method) { return new Promise((resolve, reject) => { const r = http.request({ host: '127.0.0.1', port: PORT, path, method: method || 'GET' }, x => { let d = ''; x.on('data', c => d += c); x.on('end', () => { try { resolve(JSON.parse(d)) } catch (e) { resolve(d) } }) }); r.on('error', reject); r.end() }) }
class WS { constructor() { this.buf = Buffer.alloc(0); this.frag = [] }
  connect(u) { return new Promise((res, rej) => { const m = u.match(/^ws:\/\/([^:/]+):(\d+)(\/.*)$/); const k = crypto.randomBytes(16).toString('base64'); this.sock = net.connect(+m[2], m[1], () => this.sock.write(`GET ${m[3]} HTTP/1.1\r\nHost: ${m[1]}:${m[2]}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${k}\r\nSec-WebSocket-Version: 13\r\n\r\n`)); let hs = false; this.sock.on('data', d => { this.buf = Buffer.concat([this.buf, d]); if (!hs) { const i = this.buf.indexOf('\r\n\r\n'); if (i < 0) return; hs = true; this.buf = this.buf.slice(i + 4); res() } else this._drain() }); this.sock.on('error', rej) }) }
  _drain() { for (;;) { if (this.buf.length < 2) return; const b0 = this.buf[0], b1 = this.buf[1]; let len = b1 & 0x7f, off = 2; if (len === 126) { if (this.buf.length < 4) return; len = this.buf.readUInt16BE(2); off = 4 } else if (len === 127) { if (this.buf.length < 10) return; len = Number(this.buf.readBigUInt64BE(2)); off = 10 } const masked = (b1 & 0x80) !== 0; let mask = null; if (masked) { if (this.buf.length < off + 4) return; mask = this.buf.slice(off, off + 4); off += 4 } if (this.buf.length < off + len) return; let p = this.buf.slice(off, off + len); if (masked) { const o = Buffer.alloc(len); for (let i = 0; i < len; i++) o[i] = p[i] ^ mask[i & 3]; p = o } this.buf = this.buf.slice(off + len); const op = b0 & 0x0f; if (op === 1) this.frag.push(p); if ((b0 & 0x80) && this.frag.length) { this.onMessage(Buffer.concat(this.frag)); this.frag = [] } } }
  send(o) { const p = Buffer.from(JSON.stringify(o)); const len = p.length; let h; if (len < 126) h = Buffer.from([0x81, 0x80 | len]); else { h = Buffer.alloc(4); h[0] = 0x81; h[1] = 0x80 | 126; h.writeUInt16BE(len, 2) } const mask = crypto.randomBytes(4); const m = Buffer.alloc(len); for (let i = 0; i < len; i++) m[i] = p[i] ^ mask[i & 3]; this.sock.write(Buffer.concat([h, mask, m])) } }
;(async () => {
  const list = await httpJson('/json/list')
  const tab = list.find(t => t.type === 'page' && /localhost:9096/.test(t.url))
  if (!tab) throw new Error('未找到 9096 页面 tab')
  const ws = new WS(); await ws.connect(tab.webSocketDebuggerUrl)
  let id = 0; const pend = new Map()
  ws.onMessage = b => { const m = JSON.parse(b.toString()); if (m.id && pend.has(m.id)) { pend.get(m.id)(m); pend.delete(m.id) } }
  const send = (method, params) => new Promise((res, rej) => { const i = ++id; pend.set(i, x => x.error ? rej(new Error(x.error.message)) : res(x.result)); ws.send({ id: i, method, params }) })
  const ev = async e => { const r = await send('Runtime.evaluate', { expression: e, returnByValue: true, awaitPromise: true }); return r.result && r.result.value }
  const sleep = ms => new Promise(r => setTimeout(r, ms))

  // 1. 切到任务看板页签
  console.log('页签:', JSON.stringify(await ev(`(() => { const btns=[...document.querySelectorAll('button,[role=tab],a')].filter(b=>/任务看板|流程库/.test(b.textContent||'')); return btns.map(b=>({t:(b.textContent||'').trim().slice(0,20),cls:b.className,title:b.title||''})) })()`)))
  const clicked = await ev(`(() => { const b=[...document.querySelectorAll('button,[role=tab],a,div')].find(x=>/任务看板/.test((x.textContent||'').trim()) && x.children.length<6 && (x.offsetWidth||0)<200); if(!b) return false; b.click(); return true })()`)
  console.log('点击任务看板:', clicked)
  await sleep(1500)
  const snap = await ev(`(() => ({ cards: document.querySelectorAll('.ftb-tcard').length, tas: document.querySelectorAll('.ftb-ta').length, board: !!document.querySelector('.ftb-board'), html: (document.querySelector('.ftb-wrap')||document.body).className }))()`)
  console.log('看板:', JSON.stringify(snap))

  // 2. 打开第一张卡详情
  // 详情是 toggle：若已展开，先点「关闭」，保证下面这次点击是「打开」而不是「关闭」
  const closed = await ev(`(() => { const b=[...document.querySelectorAll('.ftb-detail button')].find(x=>/关闭/.test(x.textContent||'')); if(!b) return false; b.click(); return true })()`)
  if (closed) await sleep(700)
  const opened = await ev(`(() => { const c=document.querySelector('.ftb-tcard'); if(!c) return false; c.click(); return true })()`)
  console.log('打开卡片:', opened)
  await sleep(1200)
  const probe = await ev(`(() => { const tas=[...document.querySelectorAll('.ftb-ta')]; return { n: tas.length, items: tas.map(t=>({ph:t.placeholder.slice(0,18), rows:t.rows, w:Math.round(t.getBoundingClientRect().width), h:Math.round(t.getBoundingClientRect().height), x:Math.round(t.getBoundingClientRect().left), y:Math.round(t.getBoundingClientRect().top)})) } })()`)
  console.log('textarea 几何:', JSON.stringify(probe))

  // 3. 聚焦描述 textarea，打标记，逐字符输入
  const focusInfo = await ev(`(() => { const tas=[...document.querySelectorAll('.ftb-ta')]; const ta=tas.find(t=>!t.classList.contains('prompt'))||tas[0]; if(!ta) return null; ta.focus(); ta._tbMark='M1'; ta.setSelectionRange(ta.value.length, ta.value.length); return { before: ta.value.slice(-12), active: document.activeElement===ta } })()`)
  console.log('聚焦:', JSON.stringify(focusInfo))
  const text = '自动化探针-多行输入'
  for (const ch of text) {
    await send('Input.insertText', { text: ch })
    await sleep(120)
  }
  await sleep(800)
  const after = await ev(`(() => { const all=[...document.querySelectorAll('.ftb-ta')]; const marked=all.filter(t=>t._tbMark==='M1'); const ae=document.activeElement; return { marked: marked.length, total: all.length, activeTag: ae&&ae.tagName, activeClass: ae&&ae.className, values: all.map(t=>t.value.slice(-24)) } })()`)
  console.log('输入后:', JSON.stringify(after))
  process.exit(0)
})().catch(e => { console.error('ERR', e.message); process.exit(1) })
