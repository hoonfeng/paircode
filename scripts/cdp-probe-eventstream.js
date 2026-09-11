// cdp-probe-eventstream.js — 事件流连续性探针（排查「执行中前端突然收不到 agent 输出」）
//
// 原理：经 CDP 新建标签页，并在页面脚本执行**之前**注入 WebSocket hook
//   （Page.addScriptToEvaluateOnNewDocument 包装 window.WebSocket）——
//   记录每次 onmessage 的时间戳/类型/长度，以及连接建立/关闭次数。
//   再（可选）在页面内发起一次真实任务，采样一段时间后输出：
//     · 收到消息总数与类型分布（thinking/content/tool_call/status/done…）
//     · >GAP_MS 的静默间隔（后端在跑却收不到事件 = 通讯断裂）
//     · 连接建立/关闭次数（频繁重连 = 看门狗误判/服务端主动断）
//
// 用法：node scripts/cdp-probe-eventstream.js [port=9097] [durationSec=90] [task]
//   · 省略 task → 只观测（不发起任务，适合观察某个正在跑的会话）
//   · 提供 task  → 页面内 POST /api/chat/send 发起任务后观测
//   前置：headless chrome 监听 127.0.0.1:9223；被测实例已在 <port> 监听（勿用 9090）
const http = require('http')

const PORT = Number(process.argv[2] || 9097)
const DURATION = Number(process.argv[3] || 90)
const TASK = process.argv[4] || ''
const CHROME_PORT = 9223
const CONV_ID = 'conv_evt_probe_' + Date.now()
const GAP_MS = 8000   // 超过此时长无任何 WS 帧 → 记为可疑静默
// 目标工作区（发起任务用；默认本项目。可用 PROBE_WS_ROOT 覆盖）
const WS_ROOT = process.env.PROBE_WS_ROOT || 'F:\\syproject\\gou-ide'

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

// ── CDP HTTP 助手 ──
function cdpHttp(path, method = 'PUT') {
  return new Promise((resolve, reject) => {
    const req = http.request({ host: '127.0.0.1', port: CHROME_PORT, path, method }, (res) => {
      let d = ''
      res.on('data', (c) => (d += c))
      res.on('end', () => { try { resolve(JSON.parse(d)) } catch (e) { resolve(null) } })
    })
    req.on('error', reject)
    req.end()
  })
}

// ── CDP WebSocket 助手（原生实现，避免额外依赖）──
const crypto = require('crypto')
const net = require('net')
class CDP {
  constructor(wsUrl) {
    this.url = wsUrl
    this.id = 0
    this.pending = new Map()
    this.handlers = []
  }
  connect() {
    return new Promise((resolve, reject) => {
      const u = new URL(this.url)
      this.sock = net.connect(Number(u.port), u.hostname, () => {
        const key = crypto.randomBytes(16).toString('base64')
        this.sock.write(
          `GET ${u.pathname}${u.search} HTTP/1.1\r\n` +
          `Host: ${u.host}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n` +
          `Sec-WebSocket-Key: ${key}\r\nSec-WebSocket-Version: 13\r\n\r\n`
        )
      })
      this.buf = Buffer.alloc(0)
      this.handshaken = false
      this.sock.on('data', (chunk) => {
        this.buf = Buffer.concat([this.buf, chunk])
        if (!this.handshaken) {
          const idx = this.buf.indexOf('\r\n\r\n')
          if (idx < 0) return
          this.handshaken = true
          this.buf = this.buf.slice(idx + 4)
          resolve()
        }
        this.drain()
      })
      this.sock.on('error', reject)
    })
  }
  drain() {
    while (this.buf.length >= 2) {
      const b1 = this.buf[1]
      let len = b1 & 0x7f
      let off = 2
      if (len === 126) { if (this.buf.length < 4) return; len = this.buf.readUInt16BE(2); off = 4 }
      else if (len === 127) { if (this.buf.length < 10) return; len = Number(this.buf.readBigUInt64BE(2)); off = 10 }
      if (this.buf.length < off + len) return
      const payload = this.buf.slice(off, off + len).toString('utf8')
      this.buf = this.buf.slice(off + len)
      let msg = null
      try { msg = JSON.parse(payload) } catch (e) { continue }
      if (msg.id && this.pending.has(msg.id)) {
        const { resolve } = this.pending.get(msg.id)
        this.pending.delete(msg.id)
        resolve(msg)
      } else if (msg.method) {
        for (const h of this.handlers) h(msg)
      }
    }
  }
  send(method, params = {}, sessionId) {
    const id = ++this.id
    const payload = { id, method, params }
    if (sessionId) payload.sessionId = sessionId
    const data = Buffer.from(JSON.stringify(payload), 'utf8')
    const mask = crypto.randomBytes(4)
    let header
    if (data.length < 126) header = Buffer.from([0x81, 0x80 | data.length])
    else if (data.length < 65536) {
      header = Buffer.alloc(4)
      header[0] = 0x81; header[1] = 0x80 | 126; header.writeUInt16BE(data.length, 2)
    } else {
      header = Buffer.alloc(10)
      header[0] = 0x81; header[1] = 0x80 | 127; header.writeBigUInt64BE(BigInt(data.length), 2)
    }
    const masked = Buffer.alloc(data.length)
    for (let i = 0; i < data.length; i++) masked[i] = data[i] ^ mask[i % 4]
    this.sock.write(Buffer.concat([header, mask, masked]))
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject })
      setTimeout(() => { if (this.pending.has(id)) { this.pending.delete(id); reject(new Error('CDP 超时: ' + method)) } }, 30000)
    })
  }
}

// 页面加载后注入的观测脚本：独立开一条 /ws 订阅（后端全局事件流对所有订阅者 fan-out），
// 记录每帧时间戳/类型，以及连接建立/关闭次数与 >3s 的静默间隔。
//   ★ 为何不用「加载前 hook window.WebSocket」：注入时机依赖 CDP 脚本注入点，
//     实测在部分页面上下文中拿不到（window.__wsProbe undefined）；独立订阅连接
//     与页面自带连接等价（同一后端 fan-out），且完全可控。
const HOOK = `
(function () {
  if (window.__wsProbe) return 'exists';
  var p = { connected: 0, closed: 0, errors: 0, msgs: 0, firstAt: 0, lastAt: 0, gaps: [], types: {}, urls: [] };
  window.__wsProbe = p;
  function bump(ty) { p.types[ty] = (p.types[ty] || 0) + 1 }
  function connect() {
    var url = (location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/ws';
    var ws = new WebSocket(url);
    p.connected++;
    p.urls.push(url);
    ws.onmessage = function (ev) {
      var t = Date.now();
      if (p.lastAt && (t - p.lastAt) > 3000) p.gaps.push({ gap: t - p.lastAt, at: new Date(t).toISOString() });
      p.lastAt = t;
      p.msgs++;
      if (!p.firstAt) p.firstAt = t;
      var ty = 'raw';
      try { var d = JSON.parse(ev.data); ty = Array.isArray(d) ? 'snapshot[]' : (d && d.type ? d.type : 'unknown') } catch (e) {}
      bump(ty);
    };
    ws.onclose = function () { p.closed++; setTimeout(connect, 3000) };
    ws.onerror = function () { p.errors++ };
    p.sock = ws;
    return ws;
  }
  connect();
  return 'ok';
})();
`

async function main() {
  console.log('=== 事件流探针：port=%d duration=%ds task=%s ===', PORT, DURATION, TASK || '(仅观测)')
  const ver = await cdpHttp('/json/version')
  if (!ver) { console.error('CDP %d 不可用（先启动 headless chrome）', CHROME_PORT); process.exit(1) }

  const target = await cdpHttp('/json/new?about:blank')
  if (!target || !target.webSocketDebuggerUrl) { console.error('创建标签页失败'); process.exit(1) }
  const cdp = new CDP(target.webSocketDebuggerUrl)
  await cdp.connect()
  await cdp.send('Page.enable')
  await cdp.send('Runtime.enable')
  await cdp.send('Page.navigate', { url: 'http://localhost:' + PORT + '/' })
  console.log('· 新标签页已打开，等待页面初始化…')
  await sleep(9000)

  const evalIn = async (expr, awaitPromise = false) => {
    const r = await cdp.send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise })
    // CDP 响应结构：{ id, result: { result: { type, value }, exceptionDetails } }
    const inner = r && r.result
    if (!inner) return null
    if (inner.exceptionDetails) return { error: (inner.exceptionDetails.exception || {}).description || inner.exceptionDetails.text }
    return inner.result ? inner.result.value : null
  }

  const installed = await evalIn(HOOK)
  console.log('· 观测连接注入:', installed)
  await sleep(2000)

  let probe = await evalIn('JSON.stringify(window.__wsProbe)')
  console.log('· 初始探针状态:', probe)

  if (TASK) {
    // 页面内发起任务（与用户操作同源，事件走同一个 WS）
    const sendExpr = `
      (async () => {
        const body = { message: ${JSON.stringify(TASK)}, convId: ${JSON.stringify(CONV_ID)},
                       workspaceRoot: ${JSON.stringify(WS_ROOT)} };
        const r = await fetch('/api/chat/send', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
        return JSON.stringify({ status: r.status, text: (await r.text()).slice(0, 200) });
      })()
    `
    const res = await evalIn(sendExpr, true)
    console.log('· 已发起任务:', res, ' convId=', CONV_ID)
  }

  // 周期采样：记录快照，最后输出汇总
  const t0 = Date.now()
  let lastMsgs = -1
  let stalls = []
  while (Date.now() - t0 < DURATION * 1000) {
    await sleep(3000)
    const raw = await evalIn('JSON.stringify(window.__wsProbe)')
    let p = null
    try { p = JSON.parse(raw || 'null') } catch (e) {}
    if (!p) continue
    const elapsed = Math.round((Date.now() - t0) / 1000)
    const idle = p.lastAt ? Math.round((Date.now() - p.lastAt) / 1000) : -1
    console.log('  [' + elapsed + 's] msgs=' + p.msgs + ' closed=' + p.closed + ' 最近帧距今=' + idle + 's 类型=' + JSON.stringify(p.types))
    if (lastMsgs === p.msgs && elapsed > 10) stalls.push(elapsed)
    lastMsgs = p.msgs
  }

  const finalRaw = await evalIn('JSON.stringify(window.__wsProbe)')
  console.log('\n=== 汇总 ===')
  console.log('快照:', finalRaw)
  console.log('静默采样点(msgs 无增长的采样时刻，秒):', stalls.join(',') || '无')
  await cdp.send('Page.close').catch(() => {})
  try { cdp.sock.end() } catch (e) {}
}

main().catch((e) => { console.error('探针异常:', e); process.exit(1) })
