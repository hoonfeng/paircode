// 主 tab 工具集面板验证脚本（CDP 无依赖客户端）
// 用法：node scripts/cdp-verify-toolset-tab.js <port>
// 依赖：headless chrome --remote-debugging-port=<port> 已启动
const http = require('http'); const net = require('net'); const crypto = require('crypto')
function httpJson(path, method) { return new Promise((resolve, reject) => { const r = http.request({ host: '127.0.0.1', port: 9223, path, method: method || 'GET' }, x => { let d = ''; x.on('data', c => d += c); x.on('end', () => { try { resolve(JSON.parse(d)) } catch (e) { resolve(d) } }) }); r.on('error', reject); r.end() }) }
class WS { constructor() { this.buf = Buffer.alloc(0); this.frag = [] }
  connect(u) { return new Promise((res, rej) => { const m = u.match(/^ws:\/\/([^:/]+):(\d+)(\/.*)$/); const k = crypto.randomBytes(16).toString('base64'); this.sock = net.connect(+m[2], m[1], () => this.sock.write(`GET ${m[3]} HTTP/1.1\r\nHost: ${m[1]}:${m[2]}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${k}\r\nSec-WebSocket-Version: 13\r\n\r\n`)); let hs = false; this.sock.on('data', d => { this.buf = Buffer.concat([this.buf, d]); if (!hs) { const i = this.buf.indexOf('\r\n\r\n'); if (i < 0) return; hs = true; this.buf = this.buf.slice(i + 4); res() } else this._drain() }); this.sock.on('error', rej) }) }
  _drain() { for (;;) { if (this.buf.length < 2) return; const b0 = this.buf[0], b1 = this.buf[1]; let len = b1 & 0x7f, off = 2; if (len === 126) { if (this.buf.length < 4) return; len = this.buf.readUInt16BE(2); off = 4 } else if (len === 127) { if (this.buf.length < 10) return; len = Number(this.buf.readBigUInt64BE(2)); off = 10 } const masked = (b1 & 0x80) !== 0; let mask = null; if (masked) { if (this.buf.length < off + 4) return; mask = this.buf.slice(off, off + 4); off += 4 } if (this.buf.length < off + len) return; let p = this.buf.slice(off, off + len); if (masked) { const o = Buffer.alloc(len); for (let i = 0; i < len; i++) o[i] = p[i] ^ mask[i & 3]; p = o } this.buf = this.buf.slice(off + len); const op = b0 & 0x0f; if (op === 1) this.frag.push(p); if ((b0 & 0x80) && this.frag.length) { this.onMessage(Buffer.concat(this.frag)); this.frag = [] } } }
  send(o) { const p = Buffer.from(JSON.stringify(o)); const len = p.length; let h; if (len < 126) h = Buffer.from([0x81, 0x80 | len]); else { h = Buffer.alloc(4); h[0] = 0x81; h[1] = 0x80 | 126; h.writeUInt16BE(len, 2) } const mask = crypto.randomBytes(4); const m = Buffer.alloc(len); for (let i = 0; i < len; i++) m[i] = p[i] ^ mask[i & 3]; this.sock.write(Buffer.concat([h, mask, m])) } }
;(async () => {
  const port = process.argv[2] ? Number(process.argv[2]) : 9223
  const list = await httpJson('/json/list')
  const tab = list.find(t => t.type === 'page')
  const ws = new WS(); await ws.connect(tab.webSocketDebuggerUrl)
  let id = 0; const pend = new Map(); const errors = []
  ws.onMessage = b => { const m = JSON.parse(b.toString()); if (m.id && pend.has(m.id)) { pend.get(m.id)(m); pend.delete(m.id) } else if (m.method === 'Runtime.consoleAPICalled' && m.params.type === 'error') { const txt = (m.params.args || []).map(a => a.value || a.description || '').join(' '); errors.push(txt.slice(0, 120)) } }
  const send = (method, params) => new Promise((res, rej) => { const i = ++id; pend.set(i, x => x.error ? rej(new Error(x.error.message)) : res(x.result)); ws.send({ id: i, method, params }) })
  await send('Runtime.enable')
  const ev = async e => (await send('Runtime.evaluate', { expression: e, returnByValue: true, awaitPromise: true })).result.value
  const sleep = ms => new Promise(r => setTimeout(r, ms))
  for (let i = 0; i < 20; i++) { const ok = await ev(`!!document.querySelector('.activity-bar')`); if (ok) break; await sleep(1000) }
  await sleep(2500)
  // 1. 初始状态
  const r1 = await ev(`(() => {
    const tabs = [...document.querySelectorAll('.main-tabs .main-tab')].map(b => b.textContent.trim().replace('×',''))
    const side = document.querySelector('.sidebar')
    return { tabs, tsInSide: side ? !!side.textContent.match(/工具集/) : false,
      hasIcon: [...document.querySelectorAll('.activity-bar button')].some(b => (b.title||'')==='工具集') }
  })()`)
  console.log('1.初始状态:', JSON.stringify(r1))
  if (r1.tsInSide) { console.log('FAIL: 侧边栏仍含工具集'); process.exit(1) }
  // 2. 点击活动栏工具集 → 主 tab 打开
  const r2 = await ev(`(async () => {
    const btn = [...document.querySelectorAll('.activity-bar button')].find(b => (b.title||'')==='工具集')
    if (!btn) return { ok:false, why:'无工具集图标' }
    btn.click(); await new Promise(r => setTimeout(r, 1800))
    const tabs = [...document.querySelectorAll('.main-tabs .main-tab')].map(b => b.textContent.trim().replace('×',''))
    const active = document.querySelector('.main-tab.active')?.textContent.trim().replace('×','')
    const panel = document.querySelector('.toolsets-container')
    const items = panel ? panel.querySelectorAll('[class*=tp-], .toolset-card, .tset-item').length : 0
    return { ok:true, tabs, active, panel: !!panel, items }
  })()`)
  console.log('2.打开工具集tab:', JSON.stringify(r2))
  // 3. 切回对话
  const r3 = await ev(`(async () => {
    const tabs = [...document.querySelectorAll('.main-tabs .main-tab')]
    const conv = tabs.find(b => b.textContent.includes('对话')); conv.click()
    await new Promise(r => setTimeout(r, 400))
    const p = document.querySelector('.toolsets-container')
    return { active: document.querySelector('.main-tab.active')?.textContent.trim().replace('×',''),
      panelHidden: p ? getComputedStyle(p).display === 'none' : null }
  })()`)
  console.log('3.切回对话:', JSON.stringify(r3))
  // 4. 切回工具集
  const r4 = await ev(`(async () => {
    const tabs = [...document.querySelectorAll('.main-tabs .main-tab')]
    const ts = tabs.find(b => b.textContent.includes('工具集')); ts.click()
    await new Promise(r => setTimeout(r, 500))
    const p = document.querySelector('.toolsets-container')
    return { visible: p ? getComputedStyle(p).display !== 'none' : false }
  })()`)
  console.log('4.切回工具集:', JSON.stringify(r4))
  // 5. 关闭 tab（×）
  const r5 = await ev(`(async () => {
    const tabs = [...document.querySelectorAll('.main-tabs .main-tab')]
    const ts = tabs.find(b => b.textContent.includes('工具集'))
    if (!ts) return { ok:false, why:'无工具集tab' }
    const close = ts.querySelector('.main-tab-close'); if (!close) return { ok:false, why:'无关闭按钮' }
    close.click(); await new Promise(r => setTimeout(r, 400))
    const tabs2 = [...document.querySelectorAll('.main-tabs .main-tab')].map(b => b.textContent.trim().replace('×',''))
    return { ok:true, tabs2, active: document.querySelector('.main-tab.active')?.textContent.trim().replace('×',''),
      panelGone: !document.querySelector('.toolsets-container') }
  })()`)
  console.log('5.关闭tab:', JSON.stringify(r5))
  // 6. 再次打开
  const r6 = await ev(`(async () => {
    const btn = [...document.querySelectorAll('.activity-bar button')].find(b => (b.title||'')==='工具集')
    btn.click(); await new Promise(r => setTimeout(r, 1200))
    const p = document.querySelector('.toolsets-container')
    return { panel: !!p, items: p ? p.querySelectorAll('[class*=tp-]').length : 0,
      active: document.querySelector('.main-tab.active')?.textContent.trim().replace('×','') }
  })()`)
  console.log('6.再打开:', JSON.stringify(r6))
  // 7. FileExplorer 无旧卷帘
  const r7 = await ev(`(async () => {
    const btn = [...document.querySelectorAll('.activity-bar button')].find(b => (b.title||'')==='文件浏览器')
    if (btn) { btn.click(); await new Promise(r => setTimeout(r, 1800)) }
    const fe = document.querySelector('.file-explorer')
    return { fe: !!fe, hasTs: fe ? !!fe.querySelector('.ts-divider, .ts-header') : null }
  })()`)
  console.log('7.FileExplorer:', JSON.stringify(r7))
  console.log('8.控制台错误数:', errors.length)
  errors.slice(0, 6).forEach(e => console.log('   ', e))
  const failed = !r2.ok || !r2.panel || r2.active !== '工具集' || !r3.panelHidden || !r4.visible || !r5.ok || r5.active !== '对话' || !r6.panel || (r7.hasTs === true)
  console.log(failed ? 'FAIL: 存在失败项' : 'PASS: 全部通过')
  process.exit(failed ? 1 : 0)
})().catch(e => { console.error('ERR:', e.message); process.exit(2) })
