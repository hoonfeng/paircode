// cdp-verify-runstats-backend.js — 端到端验证：运行统计「后端真源 + 刷新不丢」
//
// 覆盖（★ 2026-09-12 后端统计改造的验收点）：
//   ① 后端 run-stats 接口：真实跑一次 agent 后返回 steps/toolCalls/llmCalls/
//      genMs/tokensPerSecond 等字段（含工具次数与耗时、LLM 生成耗时）
//   ② token 速度口径：tokensPerSecond ≈ completionTokens ÷ (genMs/1000)
//      —— 分母是 LLM 生成耗时，**不是**整段运行墙钟（旧前端口径）
//   ③ 前端统计条消费后端数据：切到该会话显示「上次运行」+ 步数 + t/s，
//      且显示的 t/s 与后端返回一致（前端不再自行除法）
//   ④ Page.reload 后统计条仍在（后端 .pair/run-stats.json 持久化 = 刷新不丢）
//   ⑤ 前端不再使用 localStorage 'paircode-run-stats'（旧前端累加/持久化已删除）
//
// 用法：node scripts/cdp-verify-runstats-backend.js [port=9097] [exe=companion-runstats.exe]
//   前置：① headless chrome 监听 127.0.0.1:9223
//        ② 被测实例已在 <port> 监听（★ 勿用 9090）
//        ③ 该实例的 LLM 配置可用（可用本地 mock：node _temp/mock-llm.js 8899）
const http = require('http')
const net = require('net')
const crypto = require('crypto')
const path = require('path')
const { spawn, execSync } = require('child_process')

const PORT = Number(process.argv[2] || 9097)
const EXE = process.argv[3] || 'companion-runstats.exe'
const CHROME_PORT = 9223
const BASE = 'http://localhost:' + PORT
const ROOT = 'F:\\syproject\\gou-ide'
const STAMP = Date.now()
const CONV_ID = 'conv_runstats_e2e_' + STAMP
const CONV_TITLE = 'VERIFY 运行统计（后端）'
const TASK = '请先看项目里的 md 文档，然后输出一份不少于 1500 字的说明。'
const LS_KEY = 'paircode-run-stats'

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

// ── 极简 WebSocket 客户端（CDP）──
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
      else if (opcode === 9) this._sendFrame(10, payload)
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

async function apiJson(p, opts) {
  const r = await fetch(BASE + p, opts)
  const t = await r.text()
  let j = null
  try { j = JSON.parse(t) } catch (e) { /* ignore */ }
  return { status: r.status, body: j, text: t }
}

function runStatsUrl(convId, wsRoot) {
  return '/api/conversations/' + encodeURIComponent(convId) + '/run-stats?workspaceRoot=' + encodeURIComponent(wsRoot)
}

async function main() {
  try { await cdpHttp('/json/version') } catch (e) { console.error('FAIL: CDP 不可用（127.0.0.1:' + CHROME_PORT + '）'); process.exit(2) }
  const pid = portPid(PORT)
  if (!pid) { console.error(`FAIL: 端口 ${PORT} 未监听（先启动被测实例 WEB_PORT=${PORT} ./${EXE}）`); process.exit(2) }
  console.log(`被测实例: port=${PORT} pid=${pid}，测试会话 ${CONV_ID}`)

  // 0. 建会话
  const conv = await apiJson('/api/conversations', {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id: CONV_ID, title: CONV_TITLE, workspaceRoot: ROOT }),
  })
  console.log('  创建测试会话:', conv.status)

  // 1. 打开页面
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
    if (r && r.exceptionDetails) throw new Error('evalJS 异常: ' + JSON.stringify(r.exceptionDetails.exception && r.exceptionDetails.exception.description))
    return r && r.result ? r.result.value : undefined
  }
  await send('Runtime.enable', {})
  await send('Page.enable', {})
  await send('Page.navigate', { url: BASE + '/' })
  await sleep(5000)

  // 2. 通过页面发起真实任务（走同一 WS/事件链路）
  const sent = await evalJS(`(async () => {
    const r = await fetch('/api/chat/send', { method:'POST', headers:{'Content-Type':'application/json'},
      body: JSON.stringify({ message: ${JSON.stringify(TASK)}, convId: ${JSON.stringify(CONV_ID)}, workspaceRoot: ${JSON.stringify(ROOT)} }) })
    return r.status
  })()`)
  check('通过页面发起任务（POST /api/chat/send）', sent === 200, 'status=' + sent)

  // 3. 等运行结束（轮询后端 run-stats：running=false 且已有数据）
  let stats = null
  const deadline = Date.now() + 90000
  while (Date.now() < deadline) {
    await sleep(1500)
    const r = await apiJson(runStatsUrl(CONV_ID, ROOT))
    if (r.body && r.body.startAt > 0 && r.body.running === false) { stats = r.body; break }
  }
  if (!stats) { console.error('FAIL: 90s 内未拿到已定格的运行统计（LLM 配置可用？mock 在跑？）'); }
  console.log('  后端 run-stats:', JSON.stringify(stats))

  // ① 字段完整性
  check('后端统计：步数 > 0', !!stats && stats.steps > 0, 'steps=' + (stats && stats.steps))
  check('后端统计：LLM 调用次数 > 0', !!stats && stats.llmCalls > 0, 'llmCalls=' + (stats && stats.llmCalls))
  check('后端统计：工具调用次数 > 0（本次任务含工具）', !!stats && stats.toolCalls > 0, 'toolCalls=' + (stats && stats.toolCalls))
  check('后端统计：工具累计耗时与生成耗时均有值', !!stats && stats.genMs > 0 && stats.llmMs > 0, `genMs=${stats && stats.genMs} llmMs=${stats && stats.llmMs}`)
  check('后端统计：输出 token > 0', !!stats && stats.completionTokens > 0, 'completionTokens=' + (stats && stats.completionTokens))
  check('后端统计：已定格（endAt/墙钟耗时 > 0）', !!stats && stats.endAt > 0 && stats.durationMs > 0, `endAt=${stats && stats.endAt} durationMs=${stats && stats.durationMs}`)

  // ② 速度口径：分母为生成耗时（不是墙钟）
  let speedOk = false, speedDetail = ''
  if (stats && stats.completionTokens > 0 && stats.genMs > 0) {
    const expect = stats.completionTokens / (stats.genMs / 1000)
    const byWall = stats.completionTokens / (stats.durationMs / 1000)
    speedOk = Math.abs(stats.tokensPerSecond - expect) / expect < 0.02
    speedDetail = `tps=${stats.tokensPerSecond.toFixed(1)} 期望=${expect.toFixed(1)} 墙钟口径=${byWall.toFixed(1)}`
  }
  check('token 速度 = 输出 token ÷ LLM 生成耗时（非整段墙钟）', speedOk, speedDetail)

  // 4. 前端消费后端数据：切到该会话 → 统计条
  const gotoConv = async () => {
    const r = await evalJS(`(() => {
      const items = Array.from(document.querySelectorAll('.conv-item'))
      const hit = items.find(el => { const t = el.querySelector('.conv-title'); return t && t.textContent && t.textContent.indexOf(${JSON.stringify(CONV_TITLE)}) >= 0 })
      if (hit) { hit.click(); return { ok: true, via: 'click' } }
      window.__PAIRCODE_CORE.uiState.state.currentConvId = ${JSON.stringify(CONV_ID)}
      return { ok: true, via: 'state', total: items.length }
    })()`)
    await sleep(3000)
    return r
  }
  await gotoConv()
  const readBar = () => evalJS(`(() => {
    const bar = document.querySelector('.phase-bar')
    if (!bar) return null
    return {
      text: (bar.querySelector('.phase-text') || {}).textContent || '',
      stats: Array.from(bar.querySelectorAll('.phs-item')).map(e => e.textContent.replace(/\\s+/g, ' ').trim()),
      hasTrack: !!bar.querySelector('.phase-bar-track'),
    }
  })()`)
  let bar = await readBar()
  check('切到该会话后统计条出现', !!bar, bar ? JSON.stringify(bar) : 'null')
  check('空闲态文案为「上次运行」', !!bar && bar.text.indexOf('上次运行') >= 0, bar && bar.text)
  const barText = bar ? bar.stats.join(' | ') : ''
  check('统计条显示步数', /\d+\s*步/.test(barText), barText)
  check('统计条显示 token 速度（t/s）', /t\/s/.test(barText), barText)
  check('统计条显示工具次数', /\d+\s*次/.test(barText), barText)
  // ③ 前端显示的 t/s 与后端一致（前端不再自行除法）
  let tpsMatch = false, tpsDetail = ''
  if (bar && stats && stats.tokensPerSecond > 0) {
    const m = barText.match(/(\d+(?:\.\d+)?)\s*t\/s/)
    if (m) {
      const shown = Number(m[1])
      const expect = stats.tokensPerSecond >= 100 ? Math.round(stats.tokensPerSecond) : Number(stats.tokensPerSecond.toFixed(1))
      tpsMatch = Math.abs(shown - expect) < 1
      tpsDetail = `前端显示=${shown} 后端=${stats.tokensPerSecond.toFixed(1)}`
    }
  }
  check('前端展示的 t/s 与后端返回一致（未自行计算）', tpsMatch, tpsDetail)

  // 5. 刷新页面 → 统计条仍在（后端持久化 = 刷新不丢）
  console.log('  刷新页面（Page.reload）')
  await send('Page.reload', { ignoreCache: false })
  await sleep(6000)
  await gotoConv()
  bar = await readBar()
  check('刷新后统计条仍显示（后端持久化）', !!bar && (bar.text.indexOf('上次运行') >= 0), bar ? bar.text + ' / ' + bar.stats.join(' | ') : 'null')
  const barText2 = bar ? bar.stats.join(' | ') : ''
  check('刷新后仍显示 t/s 与步数', /t\/s/.test(barText2) && /\d+\s*步/.test(barText2), barText2)

  // ⑤ 旧的前端 localStorage 机制已删除（不得再写入）
  const lsVal = await evalJS(`localStorage.getItem(${JSON.stringify(LS_KEY)})`)
  check('前端不再写入 localStorage ' + LS_KEY + '（旧机制已删除）', lsVal === null, '值为 ' + JSON.stringify(lsVal))

  // 控制台错误
  const errs = logs.filter((l) => l.type === 'error')
  check('无控制台错误', errs.length === 0, errs.slice(0, 3).map((e) => e.text).join(' / '))

  // 清理
  try { await apiJson('/api/conversations/' + encodeURIComponent(CONV_ID), { method: 'DELETE' }) } catch (e) { /* ignore */ }
  const pass = results.filter((r) => r.ok).length
  console.log(`\n=== 结果：${pass}/${results.length} 通过 ===`)
  results.filter((r) => !r.ok).forEach((r) => console.log('  FAIL: ' + r.name + '  —— ' + (r.extra || '')))
  try { ws.sock.end() } catch (e) { /* ignore */ }
  process.exit(pass === results.length ? 0 : 1)
}

main().catch((e) => { console.error('探针异常:', e); process.exit(1) })
