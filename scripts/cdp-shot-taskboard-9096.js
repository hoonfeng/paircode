// 9096 页面截图（极简 CDP）：任务看板视图 + 流程库画布
// 用法：CDP_PORT=9224 WEB_PORT=9096 node scripts/cdp-shot-taskboard-9096.js
const http = require('http'), net = require('net'), crypto = require('crypto'), fs = require('fs'), path = require('path')
const PORT = Number(process.env.CDP_PORT || 9224)
const WEB_PORT = Number(process.env.WEB_PORT || 9096)
const ROOT = path.resolve(__dirname, '..')

const json = p => new Promise((res, rej) => {
  http.get({ host: '127.0.0.1', port: PORT, path: p }, x => { let d = ''; x.on('data', c => d += c); x.on('end', () => { try { res(JSON.parse(d)) } catch (e) { rej(e) } }) }).on('error', rej)
})

class WS {
  constructor() { this.buf = Buffer.alloc(0); this.frag = [] }
  connect(u) {
    return new Promise((res, rej) => {
      const m = u.match(/^ws:\/\/([^:/]+):(\d+)(\/.*)$/)
      const k = crypto.randomBytes(16).toString('base64')
      this.sock = net.connect(+m[2], m[1], () => this.sock.write(`GET ${m[3]} HTTP/1.1\r\nHost: ${m[1]}:${m[2]}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${k}\r\nSec-WebSocket-Version: 13\r\n\r\n`))
      let hs = false
      this.sock.setTimeout(12000, () => rej(new Error('WS 连接超时')))
      this.sock.on('data', d => {
        this.buf = Buffer.concat([this.buf, d])
        if (!hs) { const i = this.buf.indexOf('\r\n\r\n'); if (i < 0) return; hs = true; this.buf = this.buf.slice(i + 4); this.sock.setTimeout(0); res() }
        else this._drain()
      })
      this.sock.on('error', rej)
    })
  }
  _drain() {
    for (;;) {
      if (this.buf.length < 2) return
      const b0 = this.buf[0], b1 = this.buf[1]
      let len = b1 & 0x7f, off = 2
      if (len === 126) { if (this.buf.length < 4) return; len = this.buf.readUInt16BE(2); off = 4 }
      else if (len === 127) { if (this.buf.length < 10) return; len = Number(this.buf.readBigUInt64BE(2)); off = 10 }
      const masked = (b1 & 0x80) !== 0
      let mask = null
      if (masked) { if (this.buf.length < off + 4) return; mask = this.buf.slice(off, off + 4); off += 4 }
      if (this.buf.length < off + len) return
      let p = this.buf.slice(off, off + len)
      if (masked) { const o = Buffer.alloc(len); for (let i = 0; i < len; i++) o[i] = p[i] ^ mask[i & 3]; p = o }
      this.buf = this.buf.slice(off + len)
      const op = b0 & 0x0f
      if (op === 1) this.frag.push(p)
      if ((b0 & 0x80) && this.frag.length) { this.onMessage(Buffer.concat(this.frag)); this.frag = [] }
    }
  }
  send(o) {
    const p = Buffer.from(JSON.stringify(o))
    const len = p.length
    let h
    if (len < 126) h = Buffer.from([0x81, 0x80 | len])
    else { h = Buffer.alloc(4); h[0] = 0x81; h[1] = 0x80 | 126; h.writeUInt16BE(len, 2) }
    const mask = crypto.randomBytes(4)
    const m = Buffer.alloc(len)
    for (let i = 0; i < len; i++) m[i] = p[i] ^ mask[i & 3]
    this.sock.write(Buffer.concat([h, mask, m]))
  }
}

const sleep = ms => new Promise(r => setTimeout(r, ms))

;(async () => {
  const list = await json('/json/list')
  const tab = list.find(t => t.type === 'page' && new RegExp('localhost:' + WEB_PORT).test(t.url))
  if (!tab) throw new Error('未找到 ' + WEB_PORT + ' 页面 tab')
  console.log('tab:', tab.url)
  const ws = new WS()
  await ws.connect(tab.webSocketDebuggerUrl)
  console.log('WS 已连接')

  let id = 0; const pend = new Map()
  ws.onMessage = b => {
    const m = JSON.parse(b.toString())
    if (m.id && pend.has(m.id)) { const f = pend.get(m.id); pend.delete(m.id); f(m) }
  }
  const send = (method, params) => new Promise((res, rej) => {
    const i = ++id
    const timer = setTimeout(() => { pend.delete(i); rej(new Error('CDP 超时: ' + method)) }, 15000)
    pend.set(i, x => { clearTimeout(timer); x.error ? rej(new Error(x.error.message)) : res(x.result) })
    ws.send({ id: i, method, params })
  })
  const ev = async e => {
    const r = await send('Runtime.evaluate', { expression: e, returnByValue: true, awaitPromise: true })
    if (r.exceptionDetails) throw new Error('EVAL: ' + JSON.stringify(r.exceptionDetails.text || r.exceptionDetails))
    return r.result && r.result.value
  }
  const shot = async name => {
    const r = await send('Page.captureScreenshot', { format: 'png' })
    const f = path.join(ROOT, 'screenshots', name + '.png')
    fs.mkdirSync(path.dirname(f), { recursive: true })
    fs.writeFileSync(f, Buffer.from(r.data, 'base64'))
    console.log('截图:', f)
    return f
  }

  console.log('页面标题:', await ev('document.title'))
  await send('Page.enable').catch(e => console.log('Page.enable 跳过:', e.message))

  // 1. 打开任务看板
  const c1 = await ev(`(() => { const b=[...document.querySelectorAll('button,[role=tab],a,div')].find(x=>/任务看板/.test((x.textContent||'').trim()) && x.children.length<6 && (x.offsetWidth||0)<220); if(b){b.click(); return 'clicked'} return 'not-found' })()`)
  console.log('打开任务看板:', c1)
  await sleep(2200)
  console.log('看板 DOM:', JSON.stringify(await ev(`(() => ({ cards:document.querySelectorAll('.ftb-tcard').length, tabs:[...document.querySelectorAll('.ftb-tab')].map(x=>x.textContent.trim()) }))()`)))
  await shot('taskboard-board-9096')

  // 2. 切流程库
  const c2 = await ev(`(() => { const b=[...document.querySelectorAll('.ftb-tab,button')].find(x=>/流程库/.test((x.textContent||'').trim()) && (x.offsetWidth||0)<160); if(b){b.click(); return 'clicked'} return 'not-found' })()`)
  console.log('切流程库:', c2)
  await sleep(1800)
  console.log('流程库 DOM:', JSON.stringify(await ev(`(() => ({ items:document.querySelectorAll('.ftb-flowitem').length, nodes:document.querySelectorAll('.ftb-node').length, name:(document.querySelector('.ftb-cv-name')||{}).textContent||'' }))()`)))
  await shot('taskboard-flowlib-9096')

  // 3. 切回「任务看板」页签，截卡片视图
  const c3 = await ev(`(() => { const b=[...document.querySelectorAll('.ftb-tab')].find(x=>/任务看板/.test((x.textContent||'').trim())); if(b){b.click(); return 'clicked'} return 'not-found' })()`)
  console.log('切任务看板页签:', c3)
  await sleep(2200)
  console.log('看板卡片 DOM:', JSON.stringify(await ev(`(() => ({ cards:document.querySelectorAll('.ftb-tcard').length, cols:[...document.querySelectorAll('.ftb-col-head')].map(x=>x.textContent.trim()).slice(0,6) }))()`)))
  await shot('taskboard-cards-9096')
})().catch(e => { console.error('失败:', e && e.message || e); process.exitCode = 1 })
