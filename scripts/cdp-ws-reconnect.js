// cdp-ws-reconnect.js — WS 断线自动重连验证（自闭环：打开页面 → 杀服务 → 重启 → 观察恢复）
//
// 用途：验证前端 /ws 连接在服务端中断后是否自动重连，并在重连成功后标记 reconnected
//       （供 agent-events 断线补偿消费）。
// 用法：node scripts/cdp-ws-reconnect.js <port> [exePath]
//   前置：headless chrome `--headless=new --remote-debugging-port=9223 --user-data-dir=.chrome-test`
//   缺省 exePath = ./companion_test.exe（detached 重启，验证结束保留实例）
//
// 输出：console 日志（含 [WS] 前缀）+ ws-connection-change 事件序列 + PASS/FAIL。
const http = require('http')
const net = require('net')
const crypto = require('crypto')
const { spawn, execSync } = require('child_process')
const path = require('path')

const TARGET = process.argv[2] || '9098'
const EXE = process.argv[3] || 'companion_test.exe'
const CHROME_PORT = 9223

function httpJson(p) {
  return new Promise((resolve, reject) => {
    const req = http.request({ host: '127.0.0.1', port: CHROME_PORT, path: p, method: 'PUT' }, res => {
      let d = ''
      res.on('data', c => d += c)
      res.on('end', () => { try { resolve(JSON.parse(d)) } catch (e) { reject(e) } })
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
      else if (opcode === 9) { this._sendFrame(10, payload) }
      if (opcode === 8) { this.sock.end(); return }
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

const sleep = ms => new Promise(r => setTimeout(r, ms))

function portPid(port) {
  try {
    const out = execSync('netstat -ano', { encoding: 'utf8' })
    for (const line of out.split('\n')) {
      if (line.includes('LISTENING') && new RegExp(':' + port + '\\s').test(line)) {
        const cols = line.trim().split(/\s+/)
        return cols[cols.length - 1]
      }
    }
  } catch (e) { /* ignore */ }
  return ''
}

function killPid(pid) {
  try { execSync(`taskkill /PID ${pid} /F`, { stdio: 'ignore' }) } catch (e) { /* ignore */ }
}

function startServer(port) {
  const p = spawn(path.resolve(EXE), [], {
    detached: true,
    stdio: 'ignore',
    env: { ...process.env, WEB_PORT: String(port) },
  })
  p.unref()
  return p.pid
}

async function main() {
  // 0. 前置检查
  try { await httpJson('/json/version') } catch (e) { console.error('FAIL: CDP 不可用（127.0.0.1:' + CHROME_PORT + '）'); process.exit(2) }
  const initialPid = portPid(TARGET)
  if (!initialPid) { console.error(`FAIL: 端口 ${TARGET} 未监听（先启动被测实例）`); process.exit(2) }
  console.log(`被测实例: port=${TARGET} pid=${initialPid}`)

  // 1. 打开页面
  const tab = await httpJson('/json/new?http://localhost:' + TARGET + '/')
  const ws = new WS()
  await ws.connect(tab.webSocketDebuggerUrl)
  const logs = []
  ws.onMessage = buf => {
    let msg
    try { msg = JSON.parse(buf.toString()) } catch (e) { return }
    if (msg.method === 'Runtime.consoleAPICalled') {
      const a = (msg.params.args || []).map(x => x.value !== undefined ? x.value : (x.description || x.type)).join(' ')
      logs.push('[console.' + msg.params.type + '] ' + a)
    }
  }
  let id = 0
  const pending = new Map()
  const send = (method, params) => new Promise((res, rej) => {
    const i = ++id
    pending.set(i, m => m.error ? rej(new Error(m.error.message)) : res(m.result))
    ws.send({ id: i, method, params })
  })
  ws.onMessage = buf => {
    let msg
    try { msg = JSON.parse(buf.toString()) } catch (e) { return }
    if (msg.id && pending.has(msg.id)) { pending.get(msg.id)(msg); pending.delete(msg.id); return }
    if (msg.method === 'Runtime.consoleAPICalled') {
      const a = (msg.params.args || []).map(x => x.value !== undefined ? x.value : (x.description || x.type)).join(' ')
      logs.push('[console.' + msg.params.type + '] ' + a)
    }
  }
  await send('Runtime.enable', {})
  await send('Page.enable', {})
  await send('Page.navigate', { url: 'http://localhost:' + TARGET + '/' })
  await sleep(3500)

  const evalJS = async (expr) => {
    const r = await send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true })
    return r && r.result ? r.result.value : undefined
  }

  // 2. 注入 ws-connection-change 探针
  await evalJS(`(() => {
    window.__wsLog = []
    if (!window.__wsProbe) {
      window.__wsProbe = true
      window.addEventListener('ws-connection-change', e => window.__wsLog.push({
        ts: Date.now(), connected: !!(e.detail && e.detail.connected), reconnected: !!(e.detail && e.detail.reconnected),
      }))
    }
    return true
  })()`)
  await sleep(1200)
  const before = await evalJS('({log: window.__wsLog, open: !!document.querySelector("#app")})')
  console.log('阶段1 初始连接: ws-connection-change 事件数 =', (before.log || []).length, '（无断线事件为正常）')

  // 3. 杀掉服务端 → 观察断线 + 重连尝试日志
  const pidNow = portPid(TARGET) || initialPid
  console.log('阶段2 杀掉服务端 pid=' + pidNow)
  killPid(pidNow)
  logs.length = 0
  await sleep(6000)
  const mid = await evalJS('window.__wsLog')
  const disconnects = (mid || []).filter(e => !e.connected).length
  console.log('  断线事件数 =', disconnects)
  console.log('  断线期日志:')
  logs.filter(l => l.includes('[WS]') || l.includes('WebSocket')).slice(0, 8).forEach(l => console.log('    ' + l))

  // 4. 重启服务端 → 观察自动重连
  console.log('阶段3 重启服务端')
  logs.length = 0
  const newPid = startServer(TARGET)
  await sleep(12000)
  const after = await evalJS('window.__wsLog')
  const reconnects = (after || []).filter(e => e.connected).length
  console.log('  重连成功事件数 =', reconnects, '（新 pid=' + newPid + '）')
  console.log('  重连期日志:')
  logs.filter(l => l.includes('[WS]') || l.includes('WebSocket')).slice(0, 10).forEach(l => console.log('    ' + l))

  const ok = disconnects > 0 && reconnects > 0
  console.log(ok ? 'PASS: WS 断线后自动重连成功' : 'FAIL: 未观察到断线自动重连（断线事件=' + disconnects + ' 重连事件=' + reconnects + '）')
  process.exit(ok ? 0 : 1)
}

main().catch(e => { console.error('SCRIPT FAIL:', e && e.message); process.exit(2) })
