// cdp-verify-sidebar-switch.js — 左侧栏三视图切换（会话 ↔ 文件树）性能/正确性回归
//
// 用法：node scripts/cdp-verify-sidebar-switch.js [appPort]     默认 9096
// 前置：headless chrome --remote-debugging-port=9223（同 scripts/cdp-verify.js 惯例）
//
// 背景（2026-10-15 卡顿修复）：
//   ① Sidebar 原用 v-if/v-else-if 链 → 切视图即销毁重建组件；FileTreeItem 的 children
//      是组件局部 ref，重建后 state.expandedDirs 的展开态还在而子节点为空 →
//      「切过来是空树/点了没反应」。
//   ② app-actions 每 5s 派发 refresh-tree → FileTreeItem 对**每个**已展开目录重发
//      /fs/list（切到别的视图后照发不误）。
//
// 断言：
//   A1 首次进入文件树：树有节点
//   A2 展开一个子目录：节点数增长（深层内容就位）
//   A3 切到会话视图静置 6.5s（跨过周期刷新）：/fs/list 请求数**不增长**（门控生效）
//   A4 切回文件树：节点数不减少（深层展开内容仍在）+ /fs/list 不再新增（KeepAlive+缓存）
//   A5 再往返一轮：节点数稳定
const http = require('http'); const net = require('net'); const crypto = require('crypto')

const APP_PORT = process.argv[2] || '9096'
const CHROME_PORT = 9223
const URL_APP = 'http://localhost:' + APP_PORT + '/'

function httpJson(path, method) {
  return new Promise((resolve, reject) => {
    const r = http.request({ host: '127.0.0.1', port: CHROME_PORT, path, method: method || 'GET' }, x => {
      let d = ''
      x.on('data', c => d += c)
      x.on('end', () => { try { resolve(JSON.parse(d)) } catch (e) { resolve(d) } })
    })
    r.on('error', reject); r.end()
  })
}

class WS {
  constructor() { this.buf = Buffer.alloc(0); this.frag = [] }
  connect(u) {
    return new Promise((res, rej) => {
      const m = u.match(/^ws:\/\/([^:/]+):(\d+)(\/.*)$/)
      const k = crypto.randomBytes(16).toString('base64')
      this.sock = net.connect(+m[2], m[1], () => this.sock.write(
        `GET ${m[3]} HTTP/1.1\r\nHost: ${m[1]}:${m[2]}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${k}\r\nSec-WebSocket-Version: 13\r\n\r\n`))
      let hs = false
      this.sock.on('data', d => {
        this.buf = Buffer.concat([this.buf, d])
        if (!hs) {
          const i = this.buf.indexOf('\r\n\r\n'); if (i < 0) return
          hs = true; this.buf = this.buf.slice(i + 4); res()
        } else this._drain()
      })
      this.sock.on('error', rej)
    })
  }
  _drain() {
    for (;;) {
      if (this.buf.length < 2) return
      const b0 = this.buf[0], b1 = this.buf[1]
      let len = b1 & 0x7f, off = 2
      if (len === 126) { if (this.buf.length < 4) return; len = this.buf.readUInt16BE(2); off = 4 }
      else if (len === 127) { if (this.buf.length < 10) return; len = Number(this.buf.readBigUInt64BE(2)); off = 10 }
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

// 页面内注入：统计 /fs/list 请求（fetch 与 XHR 双路）
const HOOK = `(function(){
  window.__fsCalls = { list: 0 };
  window.__fsCallUrls = [];
  function note(u){ try { if (String(u).indexOf('/fs/list') >= 0) { window.__fsCalls.list++; window.__fsCallUrls.push(String(u)) } } catch(e){} }
  try {
    var _f = window.fetch;
    if (_f) window.fetch = function(){
      note((arguments[0] && arguments[0].url) || arguments[0] || '');
      return _f.apply(this, arguments);
    };
    var _o = XMLHttpRequest.prototype.open;
    XMLHttpRequest.prototype.open = function(m, u){ note(u); return _o.apply(this, arguments); };
  } catch(e){}
})()`

const results = []
function check(name, ok, detail) { results.push({ name: name, ok: !!ok, detail: detail }); }

;(async () => {
  const list = await httpJson('/json/list')
  const tab = (Array.isArray(list) ? list : []).find(t => t.type === 'page')
  if (!tab) throw new Error('未找到 CDP page tab（chrome 是否以 --remote-debugging-port=9223 启动？）')
  const ws = new WS(); await ws.connect(tab.webSocketDebuggerUrl)
  let id = 0; const pend = new Map(); const pageErrors = []
  ws.onMessage = b => {
    const m = JSON.parse(b.toString())
    if (m.id && pend.has(m.id)) { pend.get(m.id)(m); pend.delete(m.id) }
    else if (m.method === 'Runtime.consoleAPICalled' && m.params.type === 'error') {
      pageErrors.push((m.params.args || []).map(a => a.value || a.description || '').join(' ').slice(0, 140))
    }
  }
  const send = (method, params) => new Promise((res, rej) => {
    const i = ++id; pend.set(i, x => x.error ? rej(new Error(x.error.message)) : res(x.result))
    ws.send({ id: i, method, params })
  })
  const ev = async e => (await send('Runtime.evaluate', { expression: e, returnByValue: true, awaitPromise: true })).result.value
  const sleep = ms => new Promise(r => setTimeout(r, ms))

  const treeCount = `document.querySelectorAll('.file-tree-item').length`
  const fsList = `(window.__fsCalls ? window.__fsCalls.list : -1)`
  // 已展开目录数（决定「一轮补刷」的请求上限）：按 DOM 里 chevron.expanded 计
  // （根节点 defaultExpanded=true 不进 state.expandedDirs，只能从渲染结果数）。
  const expandedCount = `[...document.querySelectorAll('.file-tree-item > .item-row')].filter(r => r.querySelector('.chevron.expanded')).length`
  const urlsTail = `JSON.stringify((window.__fsCallUrls || []).slice(-12))`
  const clickByTitle = t => `(() => { const b = [...document.querySelectorAll('.activity-bar button')].find(x => x.title === ${JSON.stringify(t)}); if (!b) return 'no-btn'; b.click(); return 'ok' })()`
  const clickNthDir = n => `(() => {
    const rows = [...document.querySelectorAll('.file-tree-item > .item-row')].filter(r => r.querySelector('.chevron-wrap'));
    if (rows.length <= ${n}) return 'only-' + rows.length;
    rows[${n}].click(); return 'ok'
  })()`

  await send('Runtime.enable')
  await send('Page.enable')
  await send('Page.addScriptToEvaluateOnNewDocument', { source: HOOK })
  await send('Page.navigate', { url: URL_APP })
  for (let i = 0; i < 30; i++) { if (await ev(`!!document.querySelector('.activity-bar')`)) break; await sleep(500) }
  await sleep(2500)

  // ── A1 首次进入文件树 ──
  await ev(clickByTitle('文件浏览器'))
  await sleep(2500)
  const t1 = await ev(treeCount), c1 = await ev(fsList)
  check('A1 首次进入文件树有节点', t1 > 0, `tree=${t1} fs/list=${c1}`)

  // ── A2 展开一个子目录（制造「深层已加载内容」）──
  const dirClick = await ev(clickNthDir(2))
  await sleep(1800)
  const t2 = await ev(treeCount), c2 = await ev(fsList)
  check('A2 展开子目录后节点增长', t2 > t1, `click=${dirClick} tree=${t2}（前 ${t1}） fs/list=${c2}`)

  // ── A3 切到会话视图静置 6.5s（跨过 5s 周期刷新）──
  await ev(clickByTitle('会话'))
  await sleep(500)
  const c3a = await ev(fsList)
  await sleep(6500)
  const c3b = await ev(fsList)
  const convItems = await ev(`document.querySelectorAll('.conv-item').length`)
  check('A3 非文件树视图不再后台刷树', c3b === c3a && convItems >= 0,
    `会话视图：fs/list ${c3a} → ${c3b}（静置 6.5s，期望不增长）；conv=${convItems}`)

  // ── A4 切回文件树：内容保留 + 零新增请求 ──
  const t0 = Date.now()
  await ev(clickByTitle('文件浏览器'))
  let t4 = 0
  for (let i = 0; i < 40; i++) { t4 = await ev(treeCount); if (t4 > 0) break; await sleep(25) }
  const switchMs = Date.now() - t0
  await sleep(700)
  const c4 = await ev(fsList)
  const expCount = await ev(expandedCount)
  check('A4a 切回文件树内容仍在（非空树）', t4 >= t1, `tree=${t4}（首次 ${t1} / 展开后 ${t2}），切换耗时 ${switchMs}ms`)
  // 切走期间若周期刷新被跳过（dirty），切回会**补刷一轮**；此外 5s 周期刷新也可能
  // 恰落在切回窗口内（切回后 isFileTreeVisible() 变 true，本来就该刷）→ 最多两轮。
  // 判据（区分设计行为 vs 旧的重建式风暴）：
  //   · 同一轮内**同一目录不重复请求**（去重后 ≤ 已展开目录数）
  //   · 总量 ≤ 2 轮上限；「多轮往返零新增」由 A5 断言
  const deltaUrls = JSON.parse(await ev(`JSON.stringify((window.__fsCallUrls||[]).slice(${c3b}))`))
  const uniqDirs = [...new Set(deltaUrls.map(u => (u.match(/path=([^&]*)/) || [])[1] || u))].length
  check('A4b 切回请求规模受限且同轮无重复目录',
    deltaUrls.length <= 2 * Math.max(expCount, 1) && uniqDirs <= Math.max(expCount, 1),
    `fs/list ${c3b} → ${c4}（+${deltaUrls.length}），去重目录 ${uniqDirs} / 已展开 ${expCount}`)
  check('A4c 深层展开内容未丢失', t4 >= t2, `tree=${t4} ≥ 展开后 ${t2}`)

  // ── A5 再往返一轮：应零新增请求（dirty 已消费 + KeepAlive/缓存命中）──
  await ev(clickByTitle('会话')); await sleep(400)
  await ev(clickByTitle('文件浏览器')); await sleep(700)
  const t5 = await ev(treeCount), c5 = await ev(fsList)
  check('A5 再往返一轮：内容稳定且零新增请求', t5 >= t1 && c5 === c4, `tree=${t5} fs/list=${c4} → ${c5}（期望相等）`)

  check('A6 无新增控制台错误', pageErrors.length === 0, pageErrors.length ? pageErrors.join(' | ') : '无')

  console.log('\n=== 左侧栏切换验证（应用端口 ' + APP_PORT + '）===')
  let fail = 0
  for (const r of results) { if (!r.ok) fail++; console.log((r.ok ? 'PASS  ' : 'FAIL  ') + r.name + '  —— ' + r.detail) }
  console.log('\n结果：' + (results.length - fail) + '/' + results.length + ' 通过')
  process.exit(fail ? 1 : 0)
})().catch(e => { console.error('验证脚本异常：', e && e.message); process.exit(2) })
