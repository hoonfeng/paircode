// 「关于 → 软件更新」端到端验证（CDP 无依赖客户端）：
//   打开关于弹窗 → 读更新卡片 → 点「检查更新」→ 点「下载并安装」→ 跟踪进度到就绪
//   → 点「预览替换清单」读替换计划 → 截图。
//   ★ 不点「安装并重启」（被测实例的 .pair/plugins 可能是 junction，安装会覆盖真源）。
// 用法：node scripts/cdp-verify-update.js <debugPort> <appPort>
// 依赖：headless chrome --remote-debugging-port=<debugPort>；appPort = 被测实例端口（非 9090）
const http = require('http'); const net = require('net'); const crypto = require('crypto')
const fs = require('fs'); const path = require('path')

const DEBUG_PORT = Number(process.argv[2] || 9223)
function httpJson(p) { return new Promise((resolve, reject) => { const r = http.request({ host: '127.0.0.1', port: DEBUG_PORT, path: p, method: 'GET' }, x => { let d = ''; x.on('data', c => d += c); x.on('end', () => { try { resolve(JSON.parse(d)) } catch (e) { resolve(d) } }) }); r.on('error', reject); r.end() }) }

class WS {
  constructor() { this.buf = Buffer.alloc(0); this.frag = [] }
  connect(u) {
    return new Promise((res, rej) => {
      const m = u.match(/^ws:\/\/([^:/]+):(\d+)(\/.*)$/); const k = crypto.randomBytes(16).toString('base64')
      this.sock = net.connect(+m[2], m[1], () => this.sock.write(`GET ${m[3]} HTTP/1.1\r\nHost: ${m[1]}:${m[2]}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${k}\r\nSec-WebSocket-Version: 13\r\n\r\n`))
      let hs = false
      this.sock.on('data', d => { this.buf = Buffer.concat([this.buf, d]); if (!hs) { const i = this.buf.indexOf('\r\n\r\n'); if (i < 0) return; hs = true; this.buf = this.buf.slice(i + 4); res() } else this._drain() })
      this.sock.on('error', rej)
    })
  }
  _drain() {
    for (;;) {
      if (this.buf.length < 2) return
      const b0 = this.buf[0], b1 = this.buf[1]; let len = b1 & 0x7f, off = 2
      if (len === 126) { if (this.buf.length < 4) return; len = this.buf.readUInt16BE(2); off = 4 } else if (len === 127) { if (this.buf.length < 10) return; len = Number(this.buf.readBigUInt64BE(2)); off = 10 }
      const masked = (b1 & 0x80) !== 0; let mask = null
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
    const p = Buffer.from(JSON.stringify(o)); const len = p.length; let h
    if (len < 126) h = Buffer.from([0x81, 0x80 | len])
    else { h = Buffer.alloc(4); h[0] = 0x81; h[1] = 0x80 | 126; h.writeUInt16BE(len, 2) }
    const mask = crypto.randomBytes(4); const m = Buffer.alloc(len)
    for (let i = 0; i < len; i++) m[i] = p[i] ^ mask[i & 3]
    this.sock.write(Buffer.concat([h, mask, m]))
  }
}

; (async () => {
  const appPort = Number(process.argv[3] || 9099)
  const list = await httpJson('/json/list')
  const tab = list.find(t => t.type === 'page')
  const ws = new WS(); await ws.connect(tab.webSocketDebuggerUrl)
  let id = 0; const pend = new Map(); const errors = []
  ws.onMessage = b => {
    const m = JSON.parse(b.toString())
    if (m.id && pend.has(m.id)) { pend.get(m.id)(m); pend.delete(m.id) }
    else if (m.method === 'Runtime.consoleAPICalled' && m.params.type === 'error') errors.push(((m.params.args || []).map(a => a.value || a.description || '').join(' ')).slice(0, 160))
  }
  const send = (method, params, timeoutMs) => new Promise((res, rej) => {
    const i = ++id
    const timer = setTimeout(() => { pend.delete(i); rej(new Error('CDP 超时: ' + method)) }, timeoutMs || 30000)
    pend.set(i, x => { clearTimeout(timer); x.error ? rej(new Error(x.error.message)) : res(x.result) })
    ws.send({ id: i, method, params })
  })
  await send('Runtime.enable'); await send('Page.enable')
  // 大视口（默认 800x600 会裁掉 580px 弹窗的底部——避免把「视口小」误判为「布局溢出」）
  await send('Emulation.setDeviceMetricsOverride', { width: 1600, height: 1100, deviceScaleFactor: 1, mobile: false }, 20000)
  const ev = async (e, t) => (await send('Runtime.evaluate', { expression: e, returnByValue: true, awaitPromise: true }, t)).result.value
  const sleep = ms => new Promise(r => setTimeout(r, ms))
  const shotDir = path.join(__dirname, '..', 'screenshots')
  const shot = async (name) => {
    if (!fs.existsSync(shotDir)) fs.mkdirSync(shotDir, { recursive: true })
    const r = await send('Page.captureScreenshot', { format: 'png' }, 20000)
    fs.writeFileSync(path.join(shotDir, name + '.png'), Buffer.from(r.data, 'base64'))
    console.log('   截图: screenshots/' + name + '.png')
  }
  const readCard = () => ev(`(() => {
    const el = document.querySelector('.uc'); if (!el) return { why: 'no .uc' }
    const txt = e => (e ? (e.textContent || '').replace(/\\s+/g, ' ').trim() : '')
    return {
      text: txt(el).slice(0, 220),
      title: txt(el.querySelector('.uc-title')),
      current: txt(el.querySelector('.uc-cur')),
      newVer: txt(el.querySelector('.uc-new')),
      state: txt(el.querySelector('.uc-state')),
      progress: txt(el.querySelector('.uc-pct')),
      plan: txt(el.querySelector('.uc-plan')),
      notes: !!el.querySelector('.uc-notes'),
      buttons: [...el.querySelectorAll('button')].map(b => txt(b)),
      barWidth: el.querySelector('.uc-fill') ? el.querySelector('.uc-fill').style.width : '',
    }
  })()`, 20000)
  const clickBtn = (t) => ev(`(() => {
    const b = [...document.querySelectorAll('.uc button')].find(x => (x.textContent || '').includes('${t}'))
    if (!b) return { why: 'no btn', btns: [...document.querySelectorAll('.uc button')].map(x => (x.textContent || '').trim()) }
    b.click(); return { ok: true }
  })()`, 20000)

  console.log('1. 打开页面 http://127.0.0.1:' + appPort)
  await send('Page.navigate', { url: `http://127.0.0.1:${appPort}/` }, 20000)
  let ready = false
  for (let i = 0; i < 30; i++) { if (await ev(`!!document.querySelector('.menubar')`, 15000)) { ready = true; break } await sleep(1000) }
  if (!ready) { console.log('FAIL: 页面未就绪'); process.exit(2) }
  await sleep(2500)

  console.log('2. 打开「关于」弹窗（遍历菜单找「关于」项）')
  const open = await ev(`(async () => {
    const btns = [...document.querySelectorAll('.menu-btn')]
    for (const b of btns) {
      b.click(); await new Promise(r => setTimeout(r, 350))
      const item = [...document.querySelectorAll('.menu-item')].find(i => (i.textContent || '').includes('关于'))
      if (item) { item.click(); return { menu: (b.textContent || '').trim(), ok: true } }
      b.click(); await new Promise(r => setTimeout(r, 150))
    }
    return { ok: false, menus: btns.map(x => (x.textContent || '').trim()) }
  })()`, 40000)
  console.log('   →', JSON.stringify(open))
  if (!open.ok) { console.log('FAIL: 未打开关于弹窗'); process.exit(1) }
  for (let i = 0; i < 20; i++) { if (await ev(`!!document.querySelector('.uc')`, 15000)) break; await sleep(400) }

  const initial = await readCard()
  console.log('3. 更新卡片初始态:', JSON.stringify(initial))
  await shot('update-1-about')

  let afterCheck = initial
  if ((initial.buttons || []).some(b => b.includes('检查更新'))) {
    console.log('4. 点击「检查更新」')
    console.log('   →', JSON.stringify(await clickBtn('检查更新')))
    await sleep(3000)
    afterCheck = await readCard()
    console.log('   检查后:', JSON.stringify(afterCheck))
  } else {
    console.log('4. 跳过「检查更新」（卡片已处于就绪态）:', (initial.buttons || []).join(','))
  }

  const btns0 = afterCheck.buttons || []
  if (btns0.some(b => b.includes('下载并安装'))) {
    console.log('5. 点击「下载并安装」')
    console.log('   →', JSON.stringify(await clickBtn('下载并安装')))
    let last = ''
    for (let i = 0; i < 90; i++) {
      await sleep(1000)
      const c = await readCard()
      const line = (c.progress || '') + ' | ' + (c.state || '') + ' | ' + (c.buttons || []).join(',')
      if (line !== last) { console.log('   +' + i + 's', line); last = line }
      if ((c.buttons || []).some(b => b.includes('立即安装并重启'))) { console.log('   ✓ 已就绪（按钮出现「立即安装并重启」）'); break }
      if ((c.state || '').includes('失败') || (c.state || '').includes('错误')) { console.log('   ✗ ' + c.state); break }
    }
  } else if (btns0.some(b => b.includes('立即安装并重启'))) {
    console.log('5. 已有就绪的更新包（复用缓存，未重复下载）')
  } else if (btns0.some(b => b.includes('重新检查'))) {
    console.log('5. 已是最新版本，无更新可下载')
  } else {
    console.log('5. 未知按钮集:', JSON.stringify(btns0))
  }

  await shot('update-2-ready')
  const readyCard = await readCard()
  console.log('6. 就绪态:', JSON.stringify(readyCard))

  if ((readyCard.buttons || []).some(b => b.includes('预览替换清单'))) {
    console.log('7. 点击「预览替换清单」（★ 不安装——避免改动被测安装目录）')
    console.log('   →', JSON.stringify(await clickBtn('预览替换清单')))
    await sleep(5000)
    const withPlan = await readCard()
    console.log('   计划:', JSON.stringify({ plan: withPlan.plan, state: withPlan.state, buttons: withPlan.buttons }))
    await shot('update-3-plan')
  }

  console.log('\n控制台错误数:', errors.length)
  errors.slice(0, 5).forEach(e => console.log('   !', e))
  process.exit(0)
})()
