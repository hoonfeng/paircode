// cdp-verify-runstats-persist.js — 端到端验证：运行统计（token 速度/步数）「常态显示 + 跨刷新持久化」
//
// 覆盖（本轮「常态显示」改动的验收点）：
//   ① 无数据时不显示统计条（对照：不凭空造数据）
//   ② 一次运行结束（agentEvents.endRun 真实路径）→ 定格值落盘 localStorage
//   ③ 内存清空（模拟刷新后的空 state）+ hydrateRunStats() → 统计条常驻显示「上次运行」
//   ④ 真实 Page.reload 后仍显示（刷新页面/重启 IDE 后依然在 = 用户诉求）
//   ⑤ 运行中文案「执行中…」+ 进度条可见；结束后文案「上次运行」+ 进度条隐藏
//
// 用法：node scripts/cdp-verify-runstats-persist.js [port=9097]
//   前置：① headless chrome 监听 127.0.0.1:9223
//            （chrome.exe --headless=new --remote-debugging-port=9223 --user-data-dir=.chrome-test）
//        ② 被测实例已在 <port> 监听（★ 勿用默认 9090；本脚本只读页面，不启停实例）
//
// ★ 为什么用 CDP 直驱而不是 web_debug：web_debug 的 eval 不支持 async/promise
//   （JSON.stringify(promise) → {}），而本验证需要「eval 内 await DOM 更新 / 刷新后重读」。
const http = require('http')
const net = require('net')
const crypto = require('crypto')

const PORT = Number(process.argv[2] || 9097)
const CHROME_PORT = 9223
const BASE = 'http://localhost:' + PORT
const LS_KEY = 'paircode-run-stats'

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
const results = []
function check(name, ok, extra) {
  results.push({ name, ok, extra })
  console.log((ok ? '  PASS  ' : '  FAIL  ') + name + (extra ? '  —— ' + extra : ''))
}

// ── CDP 侧 HTTP（PUT /json/...）──
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
      this.sock.on('data', (d) => {
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
      else if (opcode === 8) { this.sock.end(); return }
      if ((b0 & 0x80) && this.frag.length) { this.onMessage(Buffer.concat(this.frag)); this.frag = [] }
    }
  }
  _sendFrame(opcode, payload) {
    const len = payload.length
    let header
    if (len < 126) header = Buffer.from([0x80 | opcode, 0x80 | len])
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
  try { await cdpHttp('/json/version') } catch (e) {
    console.error('FAIL: CDP 不可用（127.0.0.1:' + CHROME_PORT + '）——先启动 headless chrome')
    process.exit(2)
  }

  const tab = await cdpHttp('/json/new?' + encodeURIComponent(BASE + '/'))
  const ws = new WS()
  await ws.connect(tab.webSocketDebuggerUrl)

  const logs = []
  let id = 0
  const pending = new Map()
  ws.onMessage = (buf) => {
    let msg
    try { msg = JSON.parse(buf.toString()) } catch (e) { return }
    if (msg.id && pending.has(msg.id)) { pending.get(msg.id)(msg); pending.delete(msg.id); return }
    if (msg.method === 'Runtime.consoleAPICalled') {
      const a = (msg.params.args || []).map((x) => (x.value !== undefined ? x.value : (x.description || x.type))).join(' ')
      logs.push({ type: msg.params.type, text: a })
    } else if (msg.method === 'Log.entryAdded') {
      logs.push({ type: msg.params.entry.level, text: msg.params.entry.text })
    }
  }
  const send = (method, params) => new Promise((res, rej) => {
    const i = ++id
    pending.set(i, (m) => (m.error ? rej(new Error(m.error.message)) : res(m.result)))
    ws.send({ id: i, method, params })
  })
  const evalJS = async (expr) => {
    const r = await send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true })
    if (r && r.exceptionDetails) {
      throw new Error('evalJS 异常: ' + JSON.stringify(r.exceptionDetails.exception && r.exceptionDetails.exception.description))
    }
    return r && r.result ? r.result.value : undefined
  }
  // barInfo 统计条快照：文案 / 进度条是否存在 / 填充宽度
  const barInfo = () => evalJS(`(() => {
    const el = document.querySelector('.phase-bar')
    const track = document.querySelector('.phase-bar-track')
    const fill = document.querySelector('.phase-bar-fill')
    return { exists: !!el, text: el ? el.textContent.replace(/\\s+/g, ' ').trim() : '', track: !!track, fill: fill ? fill.style.width : '' }
  })()`)

  await send('Runtime.enable', {})
  await send('Page.enable', {})
  await send('Log.enable', {})
  await sleep(4500)

  const convId = await evalJS('window.__PAIRCODE_CORE && window.__PAIRCODE_CORE.uiState.state.currentConvId')
  if (!convId) { console.error('FAIL: 页面未就绪（无 currentConvId）'); process.exit(2) }
  console.log('被测页面: ' + BASE + '  当前会话: ' + convId)

  // ── ⓪ 基线：清掉持久化 + 清内存 → 无数据不显示统计条 ──
  await evalJS(`(() => {
    localStorage.removeItem(${JSON.stringify(LS_KEY)})
    const st = window.__PAIRCODE_CORE.uiState.state
    delete st.runStatsByConv[${JSON.stringify(convId)}]
    return true
  })()`)
  await sleep(600)
  const barEmpty = await barInfo()
  check('① 无运行数据时不显示统计条（对照）', !barEmpty.exists, JSON.stringify(barEmpty))

  // ── ① 一次运行结束（真实 endRun 路径）→ 定格值落盘 ──
  const runInfo = await evalJS(`(() => {
    const ae = window.__PAIRCODE_CORE.agentEvents
    const st = window.__PAIRCODE_CORE.uiState.state
    const id = ${JSON.stringify(convId)}
    ae.resetRunStat(id)
    const rs = ae.beginRun(id)
    rs.steps = 7; rs.toolCalls = 5; rs.llmCalls = 7
    rs.promptTokens = 12345; rs.completionTokens = 2400
    rs.startAt = Date.now() - 65000          // 伪装：本次运行已跑 65s
    ae.endRun(id)                            // ★ 真实结束路径（内部触发节流落盘）
    return { id, steps: rs.steps, endAt: rs.endAt }
  })()`)
  await sleep(1500)
  const ls1 = await evalJS(`JSON.parse(localStorage.getItem(${JSON.stringify(LS_KEY)}) || '{}')`)
  const entry = ls1[runInfo.id]
  check('② 运行结束 → 定格值落盘 localStorage', !!entry, entry ? JSON.stringify(entry) : '（无条目）')
  check('② 落盘口径：步数 7 / 输出 token 2400 / 工具有 endAt',
    !!entry && entry.steps === 7 && entry.completionTokens === 2400 && entry.endAt > 0,
    entry ? JSON.stringify(entry) : '')

  // ── ② 内存清空（模拟刷新后的空 state）→ hydrate 恢复 → 常态显示 ──
  await evalJS(`(() => { delete window.__PAIRCODE_CORE.uiState.state.runStatsByConv[${JSON.stringify(convId)}]; return true })()`)
  await sleep(500)
  const barDeleted = await barInfo()
  check('③ 内存清空后统计条消失（确认数据确实来自持久化）', !barDeleted.exists, JSON.stringify(barDeleted))
  await evalJS('window.__PAIRCODE_CORE.agentEvents.hydrateRunStats()')
  await sleep(600)
  const barHydrated = await barInfo()
  check('③ hydrate 恢复后统计条常驻显示「上次运行」', /上次运行/.test(barHydrated.text), barHydrated.text.slice(0, 160))
  check('③ 常态展示内容：7 步 / 5 次 / 耗时 1m 0x s / token 速度 / 2.4k',
    /7\s*步/.test(barHydrated.text) && /5\s*次/.test(barHydrated.text) && /1m\s*0[4-9]s/.test(barHydrated.text)
    && /t\/s/.test(barHydrated.text) && /2\.4k/.test(barHydrated.text),
    barHydrated.text.slice(0, 200))

  // ── ③ 真实刷新页面（最贴近用户诉求：刷新/重启后依然在）──
  await send('Page.reload', { ignoreCache: false })
  await sleep(7000)
  const ls2 = await evalJS(`JSON.parse(localStorage.getItem(${JSON.stringify(LS_KEY)}) || '{}')`)
  check('④ 刷新后持久化数据仍在', !!ls2[convId], Object.keys(ls2).join(',') || '（空）')
  const curAfter = await evalJS('window.__PAIRCODE_CORE.uiState.state.currentConvId')
  if (curAfter !== convId) {
    await evalJS(`(() => { window.__PAIRCODE_CORE.uiState.state.currentConvId = ${JSON.stringify(convId)}; return true })()`)
    await sleep(2000)
  }
  const barReload = await barInfo()
  check('④ 刷新页面后统计条自动恢复（常态显示）', /上次运行/.test(barReload.text) && /7\s*步/.test(barReload.text), barReload.text.slice(0, 200))
  check('④ 空闲态不显示进度条（避免误读为进行中）', barReload.track === false, 'track=' + barReload.track)

  // ── ④ 运行中 / 结束后的文案与进度条切换 ──
  await evalJS(`(() => {
    const ae = window.__PAIRCODE_CORE.agentEvents
    const st = window.__PAIRCODE_CORE.uiState.state
    const id = ${JSON.stringify(convId)}
    st.agentRunningByConv[id] = true
    const rs = ae.beginRun(id)
    rs.steps = 3; rs.completionTokens = 900; rs.startAt = Date.now() - 5000
    return true
  })()`)
  await sleep(1800)
  const barRunning = await barInfo()
  check('⑤ 运行中文案「执行中…」+ 步数实时显示 3 步',
    /执行中/.test(barRunning.text) && /3\s*步/.test(barRunning.text), barRunning.text.slice(0, 160))
  check('⑤ 运行中进度条可见', barRunning.track === true, 'track=' + barRunning.track + ' fill=' + barRunning.fill)
  await evalJS(`(() => {
    const ae = window.__PAIRCODE_CORE.agentEvents
    const st = window.__PAIRCODE_CORE.uiState.state
    const id = ${JSON.stringify(convId)}
    ae.endRun(id)
    st.agentRunningByConv[id] = false
    return true
  })()`)
  await sleep(1500)
  const barEnded = await barInfo()
  check('⑤ 结束后切回「上次运行」文案', /上次运行/.test(barEnded.text), barEnded.text.slice(0, 160))
  check('⑤ 结束后进度条隐藏、数据定格', barEnded.track === false && /3\s*步/.test(barEnded.text), barEnded.text.slice(0, 160))

  // ── ⑤ 控制台错误 ──
  const errs = logs.filter((l) => l.type === 'error' && !/favicon|ERR_CONNECTION|ERR_INTERNET|Failed to load resource/i.test(l.text))
  check('无未预期控制台错误', errs.length === 0, errs.slice(0, 3).map((e) => e.text).join(' ; '))

  const failed = results.filter((r) => !r.ok)
  console.log('\n═══ 结果：' + (results.length - failed.length) + '/' + results.length + ' 通过 ═══')
  if (failed.length) failed.forEach((f) => console.log('  FAIL: ' + f.name + (f.extra ? ' —— ' + f.extra : '')))
  try { ws.sock.end() } catch (e) { /* ignore */ }
  process.exit(failed.length ? 1 : 0)
}

main().catch((e) => { console.error('SCRIPT FAIL:', (e && e.stack) || e); process.exit(2) })
