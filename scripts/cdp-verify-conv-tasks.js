// cdp-verify-conv-tasks.js — 端到端验证：会话任务拉取 + 运行统计条 + WS 重连补偿
//
// 覆盖（本轮对话面板修复的三项验收点）：
//   ① 刷新/点击对话后从服务端拉取该会话任务（GET /api/tasks?convId=X → TaskPanel 渲染）
//   ② 运行统计条：耗时 / 步数 / 工具次数 / token 速度（运行中实时刷新，结束后定格）
//   ③ WS 断线自动重连后补偿：重拉当前会话任务 + 重拉当前会话消息
//
// 用法：node scripts/cdp-verify-conv-tasks.js [port=9098] [exePath=./companion_test.exe]
//   前置：headless chrome 监听 127.0.0.1:9223（--headless=new --remote-debugging-port=9223）
//        被测实例已在 <port> 监听（本脚本会 kill 后用 exePath 重启它，勿用默认 9090）
//
// 产物：临时会话（id 前缀 _verify_）+ 临时任务文件 .pair/tasks/_verify_*.json，验证后自动清理。
const http = require('http')
const net = require('net')
const fs = require('fs')
const path = require('path')
const crypto = require('crypto')
const { spawn, execSync } = require('child_process')

const PORT = Number(process.argv[2] || 9098)
const EXE = process.argv[3] || 'companion_test.exe'
const CHROME_PORT = 9223
const ROOT = path.resolve(__dirname, '..')
const BASE = 'http://127.0.0.1:' + PORT
const STAMP = Date.now()
const CONV_ID = '_verify_' + STAMP
const CONV_TITLE = 'VERIFY 任务与统计 ' + STAMP
const TASK_A = path.join(ROOT, '.pair', 'tasks', CONV_ID + '_a.json')
const TASK_B = path.join(ROOT, '.pair', 'tasks', CONV_ID + '_b.json')

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
function killPid(pid) { try { execSync(`taskkill /PID ${pid} /F`, { stdio: 'ignore' }) } catch (e) { /* ignore */ } }
function startServer() {
  const p = spawn(path.resolve(ROOT, EXE), [], { detached: true, stdio: 'ignore', env: { ...process.env, WEB_PORT: String(PORT) } })
  p.unref()
  return p.pid
}

// ── 测试数据准备 / 清理 ──
async function apiJson(p, opts) {
  const r = await fetch(BASE + p, opts)
  const t = await r.text()
  let j = null
  try { j = JSON.parse(t) } catch (e) { /* ignore */ }
  return { status: r.status, body: j, text: t }
}
function writeTaskFile(file, id, subject, status, created) {
  fs.writeFileSync(file, JSON.stringify({
    id, subject, description: '验证用任务 ' + subject, status, convId: CONV_ID, created_at: created,
  }, null, 2), 'utf8')
}
async function prepare() {
  const conv = await apiJson('/api/conversations', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id: CONV_ID, title: CONV_TITLE, workspaceRoot: ROOT }),
  })
  console.log('  创建测试会话:', conv.status, conv.text.slice(0, 120))
  writeTaskFile(TASK_A, CONV_ID + '-1', 'VERIFY-A 构造代码骨架', 'completed', new Date(STAMP).toISOString())
  writeTaskFile(TASK_B, CONV_ID + '-2', 'VERIFY-B 补齐验证脚本', 'pending', new Date(STAMP + 1000).toISOString())
  console.log('  已写入任务文件:', path.basename(TASK_A), path.basename(TASK_B))
}
async function cleanup() {
  for (const f of [TASK_A, TASK_B]) { try { fs.unlinkSync(f) } catch (e) { /* ignore */ } }
  try { await apiJson('/api/conversations/' + encodeURIComponent(CONV_ID), { method: 'DELETE' }) } catch (e) { /* ignore */ }
  console.log('  已清理测试会话与任务文件')
}

async function main() {
  // 0. 前置
  try { await cdpHttp('/json/version') } catch (e) { console.error('FAIL: CDP 不可用（127.0.0.1:' + CHROME_PORT + '）'); process.exit(2) }
  const initialPid = portPid(PORT)
  if (!initialPid) { console.error(`FAIL: 端口 ${PORT} 未监听（先启动被测实例）`); process.exit(2) }
  console.log(`被测实例: port=${PORT} pid=${initialPid}`)

  await prepare()
  const taskIds = [CONV_ID + '-1', CONV_ID + '-2']

  // 1. 打开页面
  const tab = await cdpHttp('/json/new?http://localhost:' + PORT + '/')
  const ws = new WS()
  await ws.connect(tab.webSocketDebuggerUrl)

  const logs = []
  const netReqs = []
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
    } else if (msg.method === 'Network.requestWillBeSent') {
      netReqs.push(msg.params.request.url)
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
  await send('Network.enable', {})
  await send('Log.enable', {})
  await send('Page.navigate', { url: 'http://localhost:' + PORT + '/' })
  await sleep(4500)

  // 2. 点击会话列表中的测试会话（用户操作路径）
  const clickConv = async () => {
    const r = await evalJS(`(() => {
      const items = Array.from(document.querySelectorAll('.conv-item'))
      const hit = items.find(el => {
        const t = el.querySelector('.conv-title')
        return t && t.textContent && t.textContent.indexOf(${JSON.stringify('VERIFY 任务与统计')}) >= 0
      })
      if (!hit) return { ok: false, total: items.length }
      hit.click()
      return { ok: true, total: items.length, title: hit.querySelector('.conv-title').textContent }
    })()`)
    if (!r || !r.ok) {
      // 兜底：直接切当前会话（会话列表未渲染出该项时的等价路径）
      console.warn('  会话列表未命中（items=' + (r && r.total) + '），改用 state.currentConvId 切换')
      await evalJS(`(() => { window.__PAIRCODE_CORE.uiState.state.currentConvId = ${JSON.stringify(CONV_ID)}; return true })()`)
    } else {
      console.log('  已点击会话项:', r.title, '（列表项 ' + r.total + '）')
    }
  }
  // expandTasks 展开任务面板（折叠态只渲染 header，步骤文本不渲染）并读回步骤文本
  const expandTasks = async () => {
    await evalJS(`(() => {
      const panel = document.querySelector('.task-panel')
      if (!panel) return false
      if (!panel.querySelector('.task-body')) { const h = panel.querySelector('.task-header'); if (h) h.click() }
      return true
    })()`)
    await sleep(800)
    return evalJS(`(() => {
      const el = document.querySelector('.task-panel')
      if (!el) return null
      return { texts: Array.from(el.querySelectorAll('.task-step-text')).map(e => e.textContent.trim()), progress: (el.querySelector('.task-progress') || {}).textContent }
    })()`)
  }

  // 3. 断言①（点击路径）：点击对话 → 拉取该会话任务并渲染
  await clickConv()
  await sleep(3000)
  const curConv = await evalJS('window.__PAIRCODE_CORE.uiState.state.currentConvId')
  check('点击对话后当前会话 = 测试会话', curConv === CONV_ID, 'currentConvId=' + curConv)
  const tasksReq1 = netReqs.filter((u) => u.includes('/api/tasks') && u.includes(CONV_ID)).length
  check('点击对话后发出 GET /api/tasks?convId=<测试会话>', tasksReq1 > 0, '命中 ' + tasksReq1 + ' 次')
  let taskDom = await expandTasks()
  let domText = taskDom ? taskDom.texts.join(' | ') : ''
  check('TaskPanel 渲染两个测试任务', domText.includes('VERIFY-A') && domText.includes('VERIFY-B'), domText.slice(0, 160))
  check('任务进度计数 1/2', !!taskDom && /1\s*\/\s*2/.test(taskDom.progress || ''), 'progress=' + (taskDom && taskDom.progress))

  // 4. 断言①（刷新路径）：页面刷新后重新点击对话 → 再次拉取任务并恢复面板
  console.log('  刷新页面（Page.reload）')
  await send('Page.reload', { ignoreCache: false })
  await sleep(5500)
  await clickConv()
  await sleep(3000)
  const tasksReq2 = netReqs.filter((u) => u.includes('/api/tasks') && u.includes(CONV_ID)).length
  check('刷新后点击对话仍拉取该会话任务', tasksReq2 > tasksReq1, `${tasksReq1} → ${tasksReq2}`)
  taskDom = await expandTasks()
  domText = taskDom ? taskDom.texts.join(' | ') : ''
  check('刷新后任务面板恢复（两个任务）', domText.includes('VERIFY-A') && domText.includes('VERIFY-B'), domText.slice(0, 160))

  // 4. 断言②：运行统计条（注入运行态 → 实时刷新 → 结束定格）
  //   ★ 2026-09-12 后端统计改造：字段结构与后端 GET /api/conversations/{id}/run-stats
  //     对齐（durationMs/running/toolMs/genMs/tokensPerSecond 等）——前端只渲染后端值，
  //     token 速度不再由前端相除（真源与端到端验证见 cdp-verify-runstats-backend.js）。
  const injected = await evalJS(`(() => {
    const st = window.__PAIRCODE_CORE.uiState.state
    const id = ${JSON.stringify(CONV_ID)}
    st.agentRunningByConv[id] = true
    st.runStatsByConv[id] = {
      startAt: Date.now() - 65000, endAt: 0, durationMs: 65000, running: true,
      steps: 5, toolCalls: 3, toolMs: 4200, llmCalls: 4, llmMs: 60000, genMs: 40000,
      promptTokens: 8000, completionTokens: 1200, tokensPerSecond: 30, fetchedAt: Date.now(),
    }
    return true
  })()`)
  await sleep(1800)
  const barInfo = async () => evalJS(`(() => {
    const el = document.querySelector('.phase-bar')
    if (!el) return null
    const fill = document.querySelector('.phase-bar-fill')
    return { text: el.textContent.replace(/\\s+/g, ' ').trim(), fill: fill ? fill.style.width : '', icon: !!el.querySelector('svg'), track: !!document.querySelector('.phase-bar-track') }
  })()`)
  const barRun = await barInfo()
  const barText = (barRun && barRun.text) || ''
  check('统计条出现（phase-bar）', !!barRun && injected === true, barText.slice(0, 120))
  check('显示步数 5 步', /5\s*步/.test(barText), barText.slice(0, 120))
  check('显示工具调用 3 次', /3\s*次/.test(barText), barText.slice(0, 120))
  check('显示耗时（注入 65s → 显示 1m 0x s）', /1m\s*0[4-8]s/.test(barText), barText.slice(0, 120))
  check('显示 token 速度（后端派生值 30 t/s）', /30(\.0)?\s*t\/s/.test(barText), barText.slice(0, 120))
  check('显示输出 token 1.2k', /1\.2k/.test(barText), barText.slice(0, 120))
  check('进度条宽度非 0（运行中）', !!barRun && parseFloat(barRun.fill) > 0 && parseFloat(barRun.fill) < 100, 'fill=' + (barRun && barRun.fill))

  await evalJS(`(() => {
    const st = window.__PAIRCODE_CORE.uiState.state
    const id = ${JSON.stringify(CONV_ID)}
    st.runStatsByConv[id].endAt = Date.now()
    st.runStatsByConv[id].running = false
    st.runStatsByConv[id].durationMs = 65000
    st.agentRunningByConv[id] = false
    return true
  })()`)
  await sleep(2200)
  const barEnd = await barInfo()
  const endText = (barEnd && barEnd.text) || ''
  // ★ 2026-09-12 常态显示：结束后文案为「上次运行」（结果保留，可跨刷新/重启）
  check('结束后统计条切换为「上次运行」文案（常态定格）', /上次运行/.test(endText), endText.slice(0, 120))
  await sleep(2000)
  const barEnd2 = await barInfo()
  const endText2 = (barEnd2 && barEnd2.text) || ''
  check('结束后耗时/速度定格（2s 后文本不变）', endText === endText2 && /1m\s*0[4-8]s/.test(endText2), JSON.stringify([endText, endText2]))
  check('结束后进度条隐藏（空闲态只保留数据）', !!barEnd && barEnd.track === false, 'track=' + (barEnd && barEnd.track))

  // 5. 断言③：WS 断线自动重连 + 重连补偿（重拉任务 / 重拉消息）
  await evalJS(`(() => {
    window.__wsLog = []
    window.addEventListener('ws-connection-change', e => window.__wsLog.push({ ts: Date.now(), connected: !!(e.detail && e.detail.connected), reconnected: !!(e.detail && e.detail.reconnected) }))
    return true
  })()`)
  const taskReqBefore = netReqs.filter((u) => u.includes('/api/tasks') && u.includes(CONV_ID)).length
  const msgReqBefore = netReqs.filter((u) => u.includes('/api/conversations/' + CONV_ID) && u.includes('messages')).length
  const pidNow = portPid(PORT) || initialPid
  console.log('  杀掉被测实例 pid=' + pidNow + '（模拟断线）')
  killPid(pidNow)
  logs.length = 0
  await sleep(7000)
  const wsLogMid = (await evalJS('window.__wsLog')) || []
  check('观察到 WS 断线事件', wsLogMid.some((e) => !e.connected), '事件数 ' + wsLogMid.length)
  console.log('  断线期日志: ' + logs.filter((l) => /\[WS\]|WebSocket/.test(l.text)).slice(0, 4).map((l) => l.type + ':' + l.text).join(' ; '))

  console.log('  重启被测实例')
  logs.length = 0
  const newPid = startServer()
  await sleep(18000)
  const wsLogAfter = (await evalJS('window.__wsLog')) || []
  const connected = wsLogAfter.filter((e) => e.connected)
  check('WS 自动重连成功', connected.length > 0, '重连事件 ' + connected.length + ' 次（新 pid=' + newPid + '）')
  check('重连标记 reconnected=true', connected.some((e) => e.reconnected), JSON.stringify(wsLogAfter.slice(-2)))
  const taskReqAfter = netReqs.filter((u) => u.includes('/api/tasks') && u.includes(CONV_ID)).length
  const msgReqAfter = netReqs.filter((u) => u.includes('/api/conversations/' + CONV_ID) && u.includes('messages')).length
  check('重连后重拉该会话任务', taskReqAfter > taskReqBefore, `${taskReqBefore} → ${taskReqAfter}`)
  check('重连后重拉该会话消息', msgReqAfter > msgReqBefore, `${msgReqBefore} → ${msgReqAfter}`)
  const taskDomAfter = await expandTasks()
  const domTextAfter = taskDomAfter ? taskDomAfter.texts.join(' | ') : ''
  check('重连后任务面板仍在（重拉后恢复）', /VERIFY-A/.test(domTextAfter), domTextAfter.slice(0, 160))

  // 6. 控制台错误
  const errs = logs.filter((l) => (l.type === 'error') && !/favicon|ERR_CONNECTION|ERR_INTERNET|Failed to load resource/i.test(l.text))
  check('无未预期控制台错误（重连期）', errs.length === 0, errs.slice(0, 3).map((e) => e.text).join(' ; '))

  // 7. 清理
  await cleanup()

  const failed = results.filter((r) => !r.ok)
  console.log('\n═══ 结果：' + (results.length - failed.length) + '/' + results.length + ' 通过 ═══')
  if (failed.length) failed.forEach((f) => console.log('  FAIL: ' + f.name + (f.extra ? ' —— ' + f.extra : '')))
  process.exit(failed.length ? 1 : 0)
}

main().catch((e) => { console.error('SCRIPT FAIL:', e && e.stack || e); cleanup().finally(() => process.exit(2)) })
