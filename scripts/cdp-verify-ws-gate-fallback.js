// cdp-verify-ws-gate-fallback.js — 验证「WS 事件门控兜底」（修复执行中前端不再输出 agent 内容）
//
// 背景：agent-events.js 的刷新门控（historyLoadedConvs）原先无兜底——历史加载失败/超时
//   （apiLoadAndBuildConv 返回 null 时不调用 markHistoryLoaded）会让该会话此后**所有**
//   WS 事件永久积压，界面表现为「agent 仍在跑，但前端再也不出新内容」。
//   本脚本精确复现该故障条件，并验证兜底后事件仍会渲染。
//
// 步骤：
//   ① 页面内拦截 /messages 请求（返回 500）→ 模拟历史加载失败
//   ② 直接切到测试会话（触发 switchConv → apiLoadAndBuildConv 失败 → 不 markHistoryLoaded）
//   ③ 断言确实拦截到了消息请求（故障条件成立）
//   ④ 发起任务 → 断言：3s 兜底触发（console 出现兜底警告）
//   ⑤ 断言：agent 输出真的渲染到页面（含 mock 内容标记）—— 旧代码此处为空白
//   ⑥ 断言：统计条出现（后端统计照常消费）
//
// 用法：node scripts/cdp-verify-ws-gate-fallback.js [port=9097]
//   前置：headless chrome 127.0.0.1:9223；实例在 <port> 监听；LLM 可用（mock 亦可）
const http = require('http')
const net = require('net')
const crypto = require('crypto')
const { execSync } = require('child_process')

const PORT = Number(process.argv[2] || 9097)
const CHROME_PORT = 9223
const BASE = 'http://localhost:' + PORT
const ROOT = 'F:\\syproject\\gou-ide'
const CONV_ID = 'conv_gate_probe_' + Date.now()
const CONV_TITLE = 'VERIFY 门控兜底'
const TASK = '请先看项目里的 md 文档，然后输出一份不少于 1500 字的说明。'
const MARK = '内容片段'   // mock/真实模型输出都可能命中；真实模型用宽松断言
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

const results = []
function check(name, ok, extra) {
  results.push({ name, ok, extra })
  console.log((ok ? '  PASS  ' : '  FAIL  ') + name + (extra ? '  —— ' + extra : ''))
}

function cdpHttp(p) {
  return new Promise((resolve, reject) => {
    const req = http.request({ host: '127.0.0.1', port: CHROME_PORT, path: p, method: 'PUT' }, (res) => {
      let d = ''
      res.on('data', (c) => (d += c))
      res.on('end', () => { try { resolve(JSON.parse(d)) } catch (e) { reject(e) } })
    })
    req.on('error', reject)
    req.end()
  })
}

class WS {
  constructor() { this.buf = Buffer.alloc(0); this.frag = [] }
  connect(wsUrl) {
    return new Promise((resolve, reject) => {
      const m = wsUrl.match(/^ws:\/\/([^:/]+):(\d+)(\/.*)$/)
      if (!m) return reject(new Error('bad ws url'))
      const key = crypto.randomBytes(16).toString('base64')
      this.sock = net.connect(Number(m[2]), m[1], () => {
        this.sock.write(`GET ${m[3]} HTTP/1.1\r\nHost: ${m[1]}:${m[2]}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${key}\r\nSec-WebSocket-Version: 13\r\n\r\n`)
      })
      let hs = false
      this.sock.on('data', (d) => {
        this.buf = Buffer.concat([this.buf, d])
        if (!hs) {
          const idx = this.buf.indexOf('\r\n\r\n')
          if (idx < 0) return
          if (!/101/.test(this.buf.slice(0, idx).toString())) return reject(new Error('handshake failed'))
          hs = true
          this.buf = this.buf.slice(idx + 4)
          this._drain(); resolve()
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
      const op = b0 & 0x0f
      if (op === 1) this.frag.push(payload)
      else if (op === 9) this._sendFrame(10, payload)
      else if (op === 8) { this.sock.end(); return }
      if ((b0 & 0x80) && this.frag.length) { this.onMessage(Buffer.concat(this.frag)); this.frag = [] }
    }
  }
  _sendFrame(op, payload) {
    const len = payload.length
    let header
    if (len < 126) header = Buffer.from([0x80 | op, 0x80 | len])
    else if (len < 65536) { header = Buffer.alloc(4); header[0] = 0x80 | op; header[1] = 0x80 | 126; header.writeUInt16BE(len, 2) }
    else { header = Buffer.alloc(10); header[0] = 0x80 | op; header[1] = 0x80 | 127; header.writeBigUInt64BE(BigInt(len), 2) }
    const mask = crypto.randomBytes(4)
    const out = Buffer.alloc(len)
    for (let i = 0; i < len; i++) out[i] = payload[i] ^ mask[i & 3]
    this.sock.write(Buffer.concat([header, mask, out]))
  }
  send(obj) { this._sendFrame(1, Buffer.from(JSON.stringify(obj))) }
}

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

async function main() {
  try { await cdpHttp('/json/version') } catch (e) { console.error('FAIL: CDP 不可用'); process.exit(2) }
  if (!portPid(PORT)) { console.error(`FAIL: 端口 ${PORT} 未监听`); process.exit(2) }
  console.log(`被测实例 port=${PORT}，测试会话 ${CONV_ID}`)

  const conv = await fetch(BASE + '/api/conversations', {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id: CONV_ID, title: CONV_TITLE, workspaceRoot: ROOT }),
  })
  console.log('  创建测试会话:', conv.status)

  const tab = await cdpHttp('/json/new?http://localhost:' + PORT + '/')
  const ws = new WS()
  await ws.connect(tab.webSocketDebuggerUrl)
  let id = 0
  const pending = new Map()
  const logs = []
  ws.onMessage = (buf) => {
    let msg
    try { msg = JSON.parse(buf.toString()) } catch (e) { return }
    if (msg.id && pending.has(msg.id)) { pending.get(msg.id)(msg); pending.delete(msg.id); return }
    if (msg.method === 'Runtime.consoleAPICalled') {
      const a = (msg.params.args || []).map((x) => (x.value !== undefined ? x.value : (x.description || x.type))).join(' ')
      logs.push({ type: msg.params.type, text: a })
    }
  }
  const send = (method, params) => new Promise((res, rej) => {
    const i = ++id
    pending.set(i, (m) => (m.error ? rej(new Error(m.error.message)) : res(m.result)))
    ws.send({ id: i, method, params })
  })
  const evalJS = async (expr) => {
    const r = await send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true })
    if (r && r.exceptionDetails) throw new Error('evalJS 异常: ' + (r.exceptionDetails.exception && r.exceptionDetails.exception.description))
    return r && r.result ? r.result.value : undefined
  }
  await send('Runtime.enable', {})
  await send('Page.enable', {})
  await send('Page.navigate', { url: BASE + '/' })
  await sleep(5000)

  // ① 拦截 /messages（模拟历史加载失败）
  const patched = await evalJS(`(() => {
    if (window.__fetchPatched) return 'exists';
    window.__fetchPatched = true; window.__blockedMsgReqs = 0;
    const orig = window.fetch;
    window.fetch = function (u, o) {
      const url = typeof u === 'string' ? u : ((u && u.url) || '');
      if (url.indexOf('/messages') >= 0) {
        window.__blockedMsgReqs++;
        return Promise.resolve(new Response('{"error":"blocked-for-test"}', { status: 500, headers: { 'Content-Type': 'application/json' } }));
      }
      return orig.apply(this, arguments);
    };
    return 'ok';
  })()`)
  check('注入 /messages 拦截（模拟历史加载失败）', patched === 'ok' || patched === 'exists', String(patched))

  // ② 点击会话列表中的测试会话 → switchConv → 历史加载失败 → 不 markHistoryLoaded（故障条件）
  //   ★ 必须走 DOM 点击：ui-right-panel 是独立构建的区域包，其 ui-state 与壳
  //     （window.__PAIRCODE_CORE.uiState）不是同一实例，直接改 state 不会触发切换。
  const clicked = await evalJS(`(() => {
    const items = Array.from(document.querySelectorAll('.conv-item'))
    const hit = items.find(el => { const t = el.querySelector('.conv-title'); return t && t.textContent && t.textContent.indexOf(${JSON.stringify(CONV_TITLE)}) >= 0 })
    if (!hit) return { ok: false, total: items.length, titles: items.slice(0, 10).map(e => (e.querySelector('.conv-title') || {}).textContent) }
    hit.click()
    return { ok: true, total: items.length }
  })()`)
  check('点击测试会话（触发切换与历史加载）', !!clicked && clicked.ok, JSON.stringify(clicked))
  await sleep(4000)
  const blocked = await evalJS('window.__blockedMsgReqs')
  check('历史加载确实被阻断（故障条件成立）', blocked > 0, '拦截次数=' + blocked)

  // ③ 发起任务
  const sent = await evalJS(`(async () => {
    const r = await fetch('/api/chat/send', { method:'POST', headers:{'Content-Type':'application/json'},
      body: JSON.stringify({ message: ${JSON.stringify(TASK)}, convId: ${JSON.stringify(CONV_ID)}, workspaceRoot: ${JSON.stringify(ROOT)} }) });
    return r.status;
  })()`)
  check('发起任务', sent === 200, 'status=' + sent)

  // ④ 等兜底触发（WS_PENDING_MAX_MS=3000）并渲染
  await sleep(12000)
  const fallbackLog = logs.find((l) => l.text.indexOf('兜底 flush') >= 0 || l.text.indexOf('历史加载未在') >= 0)
  check('门控兜底已触发（console 记录兜底 flush）', !!fallbackLog, fallbackLog ? fallbackLog.text.slice(0, 120) : '无')

  const flushLog = logs.find((l) => l.text.indexOf('flush pending') >= 0)
  check('pending 事件被重放（flush pending 日志）', !!flushLog, flushLog ? flushLog.text.slice(0, 120) : '无')

  // ⑤ agent 输出确实渲染（旧代码此处空白）
  const bodyText = await evalJS('document.body.innerText')
  const rendered = !!bodyText && (bodyText.indexOf(MARK) >= 0 || bodyText.length > 2000)
  check('agent 输出已渲染到页面（修复前此处无内容）', rendered, '页面文本长度=' + (bodyText ? bodyText.length : 0))

  const bar = await evalJS(`(() => {
    const b = document.querySelector('.phase-bar')
    if (!b) return null
    return { text: (b.querySelector('.phase-text')||{}).textContent||'', stats: Array.from(b.querySelectorAll('.phs-item')).map(e=>e.textContent.replace(/\\s+/g,' ').trim()) }
  })()`)
  check('统计条已显示（后端统计正常消费）', !!bar && bar.stats.length > 0, bar ? JSON.stringify(bar) : 'null')

  const errs = logs.filter((l) => l.type === 'error')
  check('无控制台错误', errs.length === 0, errs.slice(0, 3).map((e) => e.text).join(' / '))

  try { await fetch(BASE + '/api/conversations/' + encodeURIComponent(CONV_ID), { method: 'DELETE' }) } catch (e) { /* ignore */ }
  const pass = results.filter((r) => r.ok).length
  console.log(`\n=== 结果：${pass}/${results.length} 通过 ===`)
  results.filter((r) => !r.ok).forEach((r) => console.log('  FAIL: ' + r.name + '  —— ' + (r.extra || '')))
  try { ws.sock.end() } catch (e) { /* ignore */ }
  process.exit(pass === results.length ? 0 : 1)
}

main().catch((e) => { console.error('探针异常:', e); process.exit(1) })
