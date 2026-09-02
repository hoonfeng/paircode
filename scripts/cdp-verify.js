// cdp-verify.js — 无依赖 CDP 验证脚本（headless Chrome 驱动）
// 用途：打开 http://localhost:9097/ → 点击活动栏「工具集」→ 验证 ToolsetPanel 渲染
//       → 收集 console 错误。用法: node cdp-verify.js <port>
const http = require('http')
const net = require('net')
const crypto = require('crypto')

const TARGET = process.argv[2] || '9097'

function httpJson(path) {
  return new Promise((resolve, reject) => {
    const req = http.request({ host: '127.0.0.1', port: 9223, path, method: "PUT" }, res => {
      let d = ''
      res.on('data', c => d += c)
      res.on('end', () => resolve(JSON.parse(d)))
    })
    req.on('error', reject)
    req.end()
  })
}

// ── 极简 WebSocket 客户端（CDP 用；仅 text frame）──
class WS {
  constructor() { this.buf = Buffer.alloc(0); this.frag = [] }
  connect(wsUrl) {
    return new Promise((resolve, reject) => {
      const m = wsUrl.match(/^ws:\/\/([^:/]+):(\d+)(\/.*)$/)
      if (!m) return reject(new Error('bad ws url'))
      const key = crypto.randomBytes(16).toString('base64')
      this.sock = net.connect(Number(m[2]), m[1], () => {
        this.sock.write(
          `GET ${m[3]} HTTP/1.1\r\nHost: ${m[1]}:${m[2]}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${key}\r\nSec-WebSocket-Version: 13\r\n\r\n`
        )
      })
      let handshaken = false
      this.sock.on('data', d => {
        this.buf = Buffer.concat([this.buf, d])
        if (!handshaken) {
          const idx = this.buf.indexOf('\r\n\r\n')
          if (idx < 0) return
          const head = this.buf.slice(0, idx).toString()
          if (!/101/.test(head)) return reject(new Error('handshake failed: ' + head.split('\r\n')[0]))
          handshaken = true
          this.buf = this.buf.slice(idx + 4)
          this._drain()
          resolve()
        } else this._drain()
      })
      this.sock.on('error', reject)
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
      let payload = this.buf.slice(off, off + len)
      if (masked) { const out = Buffer.alloc(len); for (let i = 0; i < len; i++) out[i] = payload[i] ^ mask[i & 3]; payload = out }
      this.buf = this.buf.slice(off + len)
      const opcode = b0 & 0x0f
      if (opcode === 1) this.frag.push(payload)
      else if (opcode === 9) { this._sendFrame(10, payload) } // ping→pong
      else if (opcode === 8) { this.sock.end(); return }
      if (opcode === 8) return
      if ((b0 & 0x80) && this.frag.length) { this.onMessage(Buffer.concat(this.frag)); this.frag = [] }
    }
  }
  _sendFrame(opcode, payload) {
    const len = payload.length
    let header
    if (len < 126) { header = Buffer.from([0x80 | opcode, 0x80 | len]) }
    else if (len < 65536) { header = Buffer.alloc(4); header[0] = 0x80 | opcode; header[1] = 0x80 | 126; header.writeUInt16BE(len, 2) }
    else { header = Buffer.alloc(10); header[0] = 0x80 | opcode; header[1] = 0x80 | 127; header.writeBigUInt64BE(BigInt(len), 2) }
    const mask = crypto.randomBytes(4)
    const masked = Buffer.alloc(len)
    for (let i = 0; i < len; i++) masked[i] = payload[i] ^ mask[i & 3]
    this.sock.write(Buffer.concat([header, mask, masked]))
  }
  send(obj) { this._sendFrame(1, Buffer.from(JSON.stringify(obj))) }
}

async function main() {
  // 1. 开新标签页
  const tab = await httpJson('/json/new?http://localhost:' + TARGET + '/')
  const ws = new WS()
  await ws.connect(tab.webSocketDebuggerUrl)

  let id = 0
  const pending = new Map()
  const events = []
  ws.onMessage = buf => {
    let msg
    try { msg = JSON.parse(buf.toString()) } catch (e) { return }
    if (msg.id && pending.has(msg.id)) { pending.get(msg.id)(msg); pending.delete(msg.id) }
    else if (msg.method) {
      if (msg.method === 'Runtime.consoleAPICalled') {
        const a = (msg.params.args || []).map(x => x.value !== undefined ? x.value : (x.description || x.type)).join(' ')
        events.push(`[console.${msg.params.type}] ${a}`)
        if (msg.params.type === 'error') process.exitCode = 1
      }
      if (msg.method === 'Log.entryAdded') {
        events.push(`[log.${msg.params.entry.level}] ${msg.params.entry.text}`)
        if (msg.params.entry.level === 'error') process.exitCode = 1
      }
    }
  }
  const send = (method, params) => new Promise((res, rej) => { const i = ++id; pending.set(i, m => m.error ? rej(new Error(m.error.message)) : res(m.result)); ws.send({ id: i, method, params }) })

  await send('Runtime.enable', {})
  await send('Log.enable', {})
  await send('Page.enable', {})
  await send('Page.navigate', { url: 'http://localhost:' + TARGET + '/' })
  await new Promise(r => setTimeout(r, 3500))

  const evalJS = async (expr) => {
    const r = await send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true })
    return r && r.result ? r.result.value : undefined
  }

  // 2. 检查活动栏入口
  const bar = await evalJS(`(() => {
    const btns = [...document.querySelectorAll('.activity-bar button')]
    const t = btns.find(b => (b.title || '') === '工具集')
    if (!t) return { found: false, titles: btns.map(b => b.title) }
    t.click()
    return { found: true }
  })()`)
  console.log('活动栏工具集入口:', JSON.stringify(bar))
  if (!bar.found) { console.log('FAIL: 活动栏无工具集入口'); process.exit(1) }

  await new Promise(r => setTimeout(r, 2500))

  // 3. 检查面板渲染
  const panel = await evalJS(`(() => {
    const p = document.querySelector('.tp-panel')
    if (!p) return { ok: false }
    const items = [...document.querySelectorAll('.tp-item')].map(x => ({
      name: (x.querySelector('.tp-item-name') || {}).textContent,
      count: (x.querySelector('.tp-item-count') || {}).textContent,
    }))
    const header = (document.querySelector('.tp-header .tp-title') || {}).textContent || ''
    return { ok: true, header, items, itemCount: items.length }
  })()`)
  console.log('ToolsetPanel 渲染:', JSON.stringify(panel))
  if (!panel.ok) { console.log('FAIL: 面板未渲染'); process.exit(1) }

  // 4. 点选第一个工具集 → 详情
  if (panel.itemCount > 0) {
    await evalJS(`(() => { const it = document.querySelector('.tp-item'); if (it) it.click(); return true })()`)
    await new Promise(r => setTimeout(r, 1800))
    const detail = await evalJS(`(() => {
      const d = document.querySelector('.tp-detail')
      if (!d) return { ok: false }
      const plugs = [...document.querySelectorAll('.tp-plugin')].map(x => (x.querySelector('.tp-pname') || {}).textContent)
      const actions = [...document.querySelectorAll('.tp-dactions .tp-btn')].map(b => b.textContent.trim())
      return { ok: true, title: (document.querySelector('.tp-dtitle') || {}).textContent, plugs, plugCount: plugs.length, actions }
    })()`)
    console.log('详情渲染:', JSON.stringify(detail))
  }

  // 5. 新建弹层可打开
  const build = await evalJS(`(() => {
    const btn = [...document.querySelectorAll('.tp-icon-btn')].find(b => (b.title || '').includes('新建'))
    if (!btn) return { ok: false }
    btn.click()
    return { ok: true }
  })()`)
  await new Promise(r => setTimeout(r, 600))
  const sheet = await evalJS(`(() => {
    const s = document.querySelector('.tp-sheet')
    if (!s) return { open: false }
    return { open: true, title: (s.querySelector('.tp-sheet-title') || {}).textContent, fields: s.querySelectorAll('.tp-input').length }
  })()`)
  console.log('新建弹层:', JSON.stringify({ ...build, ...sheet }))
  await evalJS(`(() => { const c = [...document.querySelectorAll('.tp-cancel')]; if (c.length) c[c.length-1].click(); return true })()`)

  // 6. 汇总
  const errs = events.filter(e => e.startsWith('[console.error]') || e.startsWith('[log.error]'))
  console.log('控制台错误数:', errs.length)
  errs.slice(0, 10).forEach(e => console.log('  ' + e))
  console.log(errs.length === 0 ? 'PASS: 无控制台错误' : 'FAIL: 存在控制台错误')
  process.exit(errs.length === 0 ? 0 : 1)
}

main().catch(e => { console.error('SCRIPT FAIL:', e.message); process.exit(2) })
