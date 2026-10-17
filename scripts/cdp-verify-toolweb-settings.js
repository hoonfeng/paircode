// 验证「设置 → 网络工具」页的 API 实例下拉（tool-web 插件设置，2026-10-17）
// 用法：CDP_PORT=9225 WEB_PORT=9096 node scripts/cdp-verify-toolweb-settings.js
// 前置：已启动 chrome --headless=new --remote-debugging-port=9225 打开 http://localhost:9096/
const http = require('http'), net = require('net'), crypto = require('crypto'), fs = require('fs'), path = require('path')
const PORT = Number(process.env.CDP_PORT || 9225)
const WEB_PORT = Number(process.env.WEB_PORT || 9096)
const ROOT = path.resolve(__dirname, '..')

const json = p => new Promise((res, rej) => {
  http.get({ host: '127.0.0.1', port: PORT, path: p }, x => { let d = ''; x.on('data', c => d += c); x.on('end', () => { try { res(JSON.parse(d)) } catch (e) { rej(e) } }) }).on('error', rej)
})

class WS {
  constructor() { this.buf = Buffer.alloc(0) }
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
      const b1 = this.buf[1]
      let len = b1 & 0x7f, off = 2
      if (len === 126) { if (this.buf.length < 4) return; len = this.buf.readUInt16BE(2); off = 4 }
      else if (len === 127) { if (this.buf.length < 10) return; len = Number(this.buf.readBigUInt64BE(2)); off = 10 }
      const masked = (b1 & 0x80) !== 0
      let mask = null
      if (masked) { if (this.buf.length < off + 4) return; mask = this.buf.slice(off, off + 4); off += 4 }
      if (this.buf.length < off + len) return
      let p = this.buf.slice(off, off + len)
      this.buf = this.buf.slice(off + len)
      if (masked) { const c = Buffer.alloc(len); for (let i = 0; i < len; i++) c[i] = p[i] ^ mask[i % 4]; p = c }
      const op = this.buf0op === undefined ? null : null
      this._onFrame(p, b1 & 0x0f)
    }
  }
  _onFrame(payload, opcode) {
    if (opcode === 0x1 && this.onMessage) { try { this.onMessage(payload) } catch (e) { console.error('onMessage 异常:', e.message) } }
  }
  send(obj) {
    const data = Buffer.from(JSON.stringify(obj))
    const mask = crypto.randomBytes(4)
    let head
    if (data.length < 126) head = Buffer.from([0x81, 0x80 | data.length])
    else if (data.length < 65536) { head = Buffer.alloc(4); head[0] = 0x81; head[1] = 0x80 | 126; head.writeUInt16BE(data.length, 2) }
    else { head = Buffer.alloc(10); head[0] = 0x81; head[1] = 0x80 | 127; head.writeBigUInt64BE(BigInt(data.length), 2) }
    const masked = Buffer.alloc(data.length)
    for (let i = 0; i < data.length; i++) masked[i] = data[i] ^ mask[i % 4]
    this.sock.write(Buffer.concat([head, mask, masked]))
  }
}

const sleep = ms => new Promise(r => setTimeout(r, ms))

;(async () => {
  const list = await json('/json/list')
  const tab = list.find(t => t.type === 'page' && new RegExp('localhost:' + WEB_PORT).test(t.url))
  if (!tab) throw new Error('未找到 ' + WEB_PORT + ' 页面 tab')
  const ws = new WS()
  await ws.connect(tab.webSocketDebuggerUrl)
  let id = 0; const pend = new Map()
  ws.onMessage = b => {
    const m = JSON.parse(b.toString())
    if (m.id && pend.has(m.id)) { const f = pend.get(m.id); pend.delete(m.id); f(m) }
  }
  const send = (method, params) => new Promise((res, rej) => {
    const i = ++id
    const timer = setTimeout(() => { pend.delete(i); rej(new Error('CDP 超时: ' + method)) }, 20000)
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
  }

  await send('Page.enable').catch(() => {})
  console.log('标题:', await ev('document.title'))

  // 1) 找设置入口
  const entry = await ev(`(() => {
    const cand = [...document.querySelectorAll('button,[role=button],[title],a')]
      .map(e => ({ e, t: ((e.textContent||'') + ' ' + (e.getAttribute('title')||'') + ' ' + (e.getAttribute('aria-label')||'')).trim() }))
      .filter(x => /设置|Settings/i.test(x.t) && (x.e.offsetWidth||0) > 0)
    if (!cand.length) return 'not-found'
    cand[0].e.click()
    return 'clicked:' + cand[0].t.slice(0, 40)
  })()`)
  console.log('设置入口:', entry)
  await sleep(1800)

  // 2) 切到「网络工具」tab
  const tabR = await ev(`(() => {
    const hit = [...document.querySelectorAll('*')].find(e => e.children.length === 0 && (e.textContent||'').trim() === '网络工具')
    if (!hit) return 'not-found:' + (document.body.innerText||'').slice(0, 200).replace(/\\n/g,' / ')
    hit.click()
    return 'clicked'
  })()`)
  console.log('网络工具 tab:', tabR)
  await sleep(1200)

  // 3) 读取该页所有 select 的选项 + 字段标签
  const dump = await ev(`(() => ({
    labels: [...document.querySelectorAll('.field-label')].map(e => e.textContent.trim()),
    selects: [...document.querySelectorAll('select')].map(s => ({ value: s.value, opts: [...s.options].map(o => o.textContent.trim()) })),
    inputs: [...document.querySelectorAll('.field-input')].map(i => ({ type: i.type, value: i.value, ph: i.placeholder })),
    hints: [...document.querySelectorAll('.setting-hint')].map(e => e.textContent.trim().slice(0, 80))
  }))()`)
  console.log('页面字段:', JSON.stringify(dump, null, 1).slice(0, 2500))
  await shot('settings-tool-web-9096')
})().catch(e => { console.error('失败:', e && e.message || e); process.exitCode = 1 })
