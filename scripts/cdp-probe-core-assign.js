// cdp-probe-core-assign.js — 诊断探针：记录页面里 __PAIRCODE_CORE（及其 actions）被赋值的历史
//
// 用途：排查「区域包报 xxx is not a function，但页面里 window.__PAIRCODE_CORE.actions 明明有该函数」
// 这类**跨版本 core 快照**问题（区域包在 IIFE 求值时捕获 core 引用）。
// 用法：node scripts/cdp-probe-core-assign.js [appPort]     默认 9096
const http = require('http'); const net = require('net'); const crypto = require('crypto')
const APP_PORT = process.argv[2] || '9096'
const CHROME_PORT = 9223

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
        if (!hs) { const i = this.buf.indexOf('\r\n\r\n'); if (i < 0) return; hs = true; this.buf = this.buf.slice(i + 4); res() }
        else this._drain()
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
      if ((b0 & 0x0f) === 1) this.frag.push(p)
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

// 注入：拦截 __PAIRCODE_CORE 与其 actions 的赋值，记录「谁在什么时候装了缺函数的对象」
const HOOK = `(function(){
  window.__coreLog = [];
  var _c;
  function snap(tag, v){
    try {
      var a = v && v.actions;
      window.__coreLog.push({ tag: tag, t: Math.round(performance.now()),
        hasActions: !!a, keys: a ? Object.keys(a).length : -1,
        hasConsume: !!(a && typeof a.consumeTreeDirty === 'function'),
        hasMark: !!(a && typeof a.markTreeDirty === 'function') });
    } catch(e) { window.__coreLog.push({ tag: tag, err: String(e) }) }
  }
  try {
    Object.defineProperty(window, '__PAIRCODE_CORE', {
      configurable: true,
      get: function(){ return _c },
      set: function(v){ snap('set-core', v); _c = v }
    });
  } catch(e) {}
  // 记录区域包脚本注入时刻（对照 core 赋值时序）
  var _append = Element.prototype.appendChild;
  Element.prototype.appendChild = function(el){
    try { if (el && el.tagName === 'SCRIPT' && el.src) window.__coreLog.push({ tag: 'script', src: String(el.src).slice(-40), t: Math.round(performance.now()) }) } catch(e){}
    return _append.apply(this, arguments);
  };
})()`

;(async () => {
  const list = await httpJson('/json/list')
  const tab = (Array.isArray(list) ? list : []).find(t => t.type === 'page')
  if (!tab) throw new Error('未找到 CDP page tab')
  const ws = new WS(); await ws.connect(tab.webSocketDebuggerUrl)
  let id = 0; const pend = new Map()
  ws.onMessage = b => { const m = JSON.parse(b.toString()); if (m.id && pend.has(m.id)) { pend.get(m.id)(m); pend.delete(m.id) } }
  const send = (method, params) => new Promise((res, rej) => {
    const i = ++id; pend.set(i, x => x.error ? rej(new Error(x.error.message)) : res(x.result)); ws.send({ id: i, method, params })
  })
  const ev = async e => (await send('Runtime.evaluate', { expression: e, returnByValue: true, awaitPromise: true })).result.value
  const sleep = ms => new Promise(r => setTimeout(r, ms))

  await send('Runtime.enable'); await send('Page.enable')
  await send('Page.addScriptToEvaluateOnNewDocument', { source: HOOK })
  await send('Page.navigate', { url: 'http://localhost:' + APP_PORT + '/' })
  await sleep(4000)

  // ① core 各子对象的 consumeTreeDirty 可见性 + 属性描述符（可枚举性！）
  const survey = await ev(`(() => {
    const c = window.__PAIRCODE_CORE || {}
    const out = {}
    for (const k of Object.keys(c)) {
      const o = c[k]
      const d = (o && typeof o === 'object') ? Object.getOwnPropertyDescriptor(o, 'consumeTreeDirty') : null
      out[k] = {
        has: !!(o && typeof o.consumeTreeDirty === 'function'),
        own: !!(o && Object.prototype.hasOwnProperty.call(o, 'consumeTreeDirty')),
        enumerable: d ? d.enumerable : null,
        writable: d ? d.writable : null,
      }
    }
    return JSON.stringify(out)
  })()`)

  // ② 给**所有** core 子对象（含 actions）挂同名拦截函数，看区域包到底调用谁
  const armed = await ev(`(() => {
    window.__probeHit = []
    const c = window.__PAIRCODE_CORE || {}
    const armedKeys = []
    for (const k of Object.keys(c)) {
      const o = c[k]
      if (!o || typeof o !== 'object') continue
      try {
        Object.defineProperty(o, 'consumeTreeDirty', {
          configurable: true, enumerable: true, writable: true,
          value: function(){ window.__probeHit.push(k); return false },
        })
        armedKeys.push(k)
      } catch (e) { armedKeys.push(k + '!ERR') }
    }
    return JSON.stringify(armedKeys)
  })()`)

  // ③ 触发一次 chat → explorer → chat → explorer（onActivated 路径）
  const click = t => `(() => { const b = [...document.querySelectorAll('.activity-bar button')].find(x => x.title === ${JSON.stringify('文件浏览器')}); if (b) b.click(); return !!b })()`
  const clickChat = `(() => { const b = [...document.querySelectorAll('.activity-bar button')].find(x => x.title === '会话'); if (b) b.click(); return !!b })()`
  await ev(click()); await sleep(1500)
  await ev(clickChat); await sleep(600)
  await ev(click()); await sleep(900)

  const hit = await ev(`JSON.stringify(window.__probeHit || [])`)
  const log = await ev(`JSON.stringify((window.__coreLog || []).slice(0, 8))`)
  const now = await ev(`JSON.stringify({ hasConsume: typeof (window.__PAIRCODE_CORE && window.__PAIRCODE_CORE.actions && window.__PAIRCODE_CORE.actions.consumeTreeDirty), keys: Object.keys((window.__PAIRCODE_CORE||{}).actions||{}).length })`)

  console.log('=== ① core 子对象的 consumeTreeDirty 属性描述符 ===')
  console.log(survey)
  console.log('=== ② 被挂探针的（缺函数）子对象 ===', armed)
  console.log('=== ③ onActivated 实际命中的键 ===', hit)
  console.log('=== core 赋值/脚本注入时序（前 8 条）===')
  for (const l of JSON.parse(log)) console.log(JSON.stringify(l))
  console.log('=== 页面终态 ===', now)
  process.exit(0)
})().catch(e => { console.error('探针异常：', e && e.message); process.exit(2) })
