// 「已安装」类型筛选 tag 最小验证（CDP 无依赖客户端）：
//   打开市场 → 已安装 tab → 依次点击 插件/MCP/技能/全部 tag，
//   断言分组按类型过滤、空类型给出空态提示。
// 用法：node scripts/cdp-verify-market-filter-tag.js <debugPort> <appPort>
// 依赖：headless chrome --remote-debugging-port=<debugPort> 已启动；
//      appPort = 被测实例端口（★ 非默认 9090——自举铁律）
const http = require('http'); const net = require('net'); const crypto = require('crypto')
const fs = require('fs'); const path = require('path')
function httpJson(p) { return new Promise((resolve, reject) => { const r = http.request({ host: '127.0.0.1', port: 9223, path: p, method: 'GET' }, x => { let d = ''; x.on('data', c => d += c); x.on('end', () => { try { resolve(JSON.parse(d)) } catch (e) { resolve(d) } }) }); r.on('error', reject); r.end() }) }
class WS { constructor() { this.buf = Buffer.alloc(0); this.frag = [] }
  connect(u) { return new Promise((res, rej) => { const m = u.match(/^ws:\/\/([^:/]+):(\d+)(\/.*)$/); const k = crypto.randomBytes(16).toString('base64'); this.sock = net.connect(+m[2], m[1], () => this.sock.write(`GET ${m[3]} HTTP/1.1\r\nHost: ${m[1]}:${m[2]}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${k}\r\nSec-WebSocket-Version: 13\r\n\r\n`)); let hs = false; this.sock.on('data', d => { this.buf = Buffer.concat([this.buf, d]); if (!hs) { const i = this.buf.indexOf('\r\n\r\n'); if (i < 0) return; hs = true; this.buf = this.buf.slice(i + 4); res() } else this._drain() }); this.sock.on('error', rej) }) }
  _drain() { for (;;) { if (this.buf.length < 2) return; const b0 = this.buf[0], b1 = this.buf[1]; let len = b1 & 0x7f, off = 2; if (len === 126) { if (this.buf.length < 4) return; len = this.buf.readUInt16BE(2); off = 4 } else if (len === 127) { if (this.buf.length < 10) return; len = Number(this.buf.readBigUInt64BE(2)); off = 10 } const masked = (b1 & 0x80) !== 0; let mask = null; if (masked) { if (this.buf.length < off + 4) return; mask = this.buf.slice(off, off + 4); off += 4 } if (this.buf.length < off + len) return; let p = this.buf.slice(off, off + len); if (masked) { const o = Buffer.alloc(len); for (let i = 0; i < len; i++) o[i] = p[i] ^ mask[i & 3]; p = o } this.buf = this.buf.slice(off + len); const op = b0 & 0x0f; if (op === 1) this.frag.push(p); if ((b0 & 0x80) && this.frag.length) { this.onMessage(Buffer.concat(this.frag)); this.frag = [] } } }
  send(o) { const p = Buffer.from(JSON.stringify(o)); const len = p.length; let h; if (len < 126) h = Buffer.from([0x81, 0x80 | len]); else { h = Buffer.alloc(4); h[0] = 0x81; h[1] = 0x80 | 126; h.writeUInt16BE(len, 2) } const mask = crypto.randomBytes(4); const m = Buffer.alloc(len); for (let i = 0; i < len; i++) m[i] = p[i] ^ mask[i & 3]; this.sock.write(Buffer.concat([h, mask, m])) } }

; (async () => {
  const appPort = Number(process.argv[3] || 9097)
  const list = await httpJson('/json/list')
  const tab = list.find(t => t.type === 'page')
  const ws = new WS(); await ws.connect(tab.webSocketDebuggerUrl)
  let id = 0; const pend = new Map(); const errors = []
  ws.onMessage = b => { const m = JSON.parse(b.toString()); if (m.id && pend.has(m.id)) { pend.get(m.id)(m); pend.delete(m.id) } else if (m.method === 'Runtime.consoleAPICalled' && m.params.type === 'error') { errors.push(((m.params.args || []).map(a => a.value || a.description || '').join(' ')).slice(0, 160)) } }
  const send = (method, params, timeoutMs) => new Promise((res, rej) => {
    const i = ++id
    const timer = setTimeout(() => { pend.delete(i); rej(new Error('CDP 超时: ' + method)) }, timeoutMs || 30000)
    pend.set(i, x => { clearTimeout(timer); x.error ? rej(new Error(x.error.message)) : res(x.result) })
    ws.send({ id: i, method, params })
  })
  await send('Runtime.enable'); await send('Page.enable')
  const ev = async (e, t) => (await send('Runtime.evaluate', { expression: e, returnByValue: true, awaitPromise: true }, t)).result.value
  const sleep = ms => new Promise(r => setTimeout(r, ms))
  const shotDir = path.join(__dirname, '..', 'screenshots')
  const shot = async (name) => {
    if (!fs.existsSync(shotDir)) fs.mkdirSync(shotDir, { recursive: true })
    const r = await send('Page.captureScreenshot', { format: 'png' }, 20000)
    fs.writeFileSync(path.join(shotDir, name + '.png'), Buffer.from(r.data, 'base64'))
    console.log('   截图: screenshots/' + name + '.png')
  }

  await send('Page.navigate', { url: `http://127.0.0.1:${appPort}/` }, 20000)
  let ok = false
  for (let i = 0; i < 30; i++) { if (await ev(`!!document.querySelector('.activity-bar')`, 15000)) { ok = true; break } await sleep(1000) }
  if (!ok) { console.log('FAIL: 页面未就绪'); process.exit(2) }
  await sleep(2500)

  // 打开市场 → 已安装
  const open = await ev(`(async () => {
    const btn = [...document.querySelectorAll('.activity-bar button')].find(b => (b.title||'')==='市场')
    if (!btn) return { why: '无市场图标' }
    btn.click()
    for (let i = 0; i < 20; i++) { await new Promise(r => setTimeout(r, 300)); if (document.querySelector('.market-tabs')) break }
    const inst = [...document.querySelectorAll('.market-tabs button')].find(b => b.textContent.trim() === '已安装')
    if (!inst) return { why: '无已安装 tab' }
    inst.click()
    for (let i = 0; i < 20; i++) { await new Promise(r => setTimeout(r, 300)); if (document.querySelector('.installed-filters')) break }
    return { ok: true, hasFilters: !!document.querySelector('.installed-filters'),
      tags: [...document.querySelectorAll('.if-tag')].map(b => (b.textContent || '').replace(/\\s+/g,' ').trim()) }
  })()`, 40000)
  console.log('1.打开已安装页:', JSON.stringify(open))
  if (!open.ok) { console.log('FAIL: 无法打开已安装页'); process.exit(1) }

  const probe = async (label) => await ev(`(async () => {
    const t = [...document.querySelectorAll('.if-tag')].find(b => (b.textContent||'').replace(/\\s+/g,' ').trim().startsWith('${label}'))
    if (!t) return { why: '无 tag ${label}' }
    t.click()
    await new Promise(r => setTimeout(r, 700))
    const txt = el => (el.textContent || '').replace(/\\s+/g,' ').trim()
    const empt = document.querySelector('.market-empty')
    return { active: txt(document.querySelector('.if-tag.active') || { textContent: '' }),
      groups: [...document.querySelectorAll('.installed-group-title')].map(g => txt(g)),
      items: document.querySelectorAll('.installed-item').length,
      emptyHint: empt ? txt(empt).slice(0, 50) : '' }
  })()`, 20000)

  const r = {}
  r.plugin = await probe('插件'); await shot('filter-1-plugin')
  r.mcp = await probe('MCP'); await shot('filter-2-mcp-empty')
  r.skill = await probe('技能'); await shot('filter-3-skill')
  r.all = await probe('全部'); await shot('filter-4-all')
  for (const k of ['plugin', 'mcp', 'skill', 'all']) console.log(`2.${k}:`, JSON.stringify(r[k]))
  console.log('3.控制台错误数:', errors.length)

  const failed = !r.plugin || !r.mcp || !r.skill || !r.all
    || (r.plugin.groups || []).length !== 1                                    // 只显示插件分组
    || (r.mcp.groups || []).length !== 0 || (r.mcp.items || 0) !== 0           // MCP 无内容 → 空
    || !/MCP/.test(r.mcp.emptyHint || '')                                       // 应给出空态提示
    || (r.skill.groups || []).some(g => /插件/.test(g))                         // 技能筛选不含插件组
    || (r.all.groups || []).length !== (r.skill.groups || []).length + 1        // 全部 = 插件 + 技能 组
  console.log(failed ? 'FAIL: 类型筛选存在失败项' : 'PASS: 类型筛选 tag 点击切换全部通过')
  process.exit(failed ? 1 : 0)
})().catch(e => { console.error('ERR:', e.message); process.exit(2) })
