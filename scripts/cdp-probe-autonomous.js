// cdp-probe-autonomous.js — 「自主模式开关 → 发送参数」联动探针（只读探针，不发真消息）
//
// 用法：node scripts/cdp-probe-autonomous.js [appPort]     默认 9098
// 前置：headless chrome --remote-debugging-port=9223（同 scripts/cdp-verify.js 惯例）
//
// 背景（2026-10-16 用户反馈：「开关是开的，但 agent 跑完一轮就停、不进入监督续轮」）：
//   输入区右侧的圆形按钮（.obtn-agent）是自主模式开关，其值经
//   RightPanel.vue 的 toggleAuto() 写入 AppSettings.autonomous，并在 sendMessage()
//   中作为第 3 参传给 api.chatStart() → POST /api/chat/send 的 body.autonomous。
//   本探针在页面内劫持 window.fetch，**拦下** /chat/send 请求并记录 body（不产生
//   任何 LLM 调用、不写入任何会话数据），据此断言「界面开关状态」与「实际发出的
//   autonomous 字段」是否一致。
//
// 断言：
//   A1 开关可定位，初始 active 态与后端 GET /api/settings 的 autonomous 一致
//   A2 点击开关 → active 类翻转，且后端值随之落盘（PUT /api/settings 生效）
//   A3 点击后发送 → body.autonomous === true（本探针核心：开关开必须传 true）
//   A4 再点回来发送 → body.autonomous === false
//   A5 无控制台错误
const http = require('http'); const net = require('net'); const crypto = require('crypto')

const APP_PORT = process.argv[2] || '9098'
const CHROME_PORT = 9223
const URL_APP = 'http://localhost:' + APP_PORT + '/'

function reqJson(port, path, method, body) {
  return new Promise((resolve, reject) => {
    const data = body ? JSON.stringify(body) : null
    const r = http.request({
      host: '127.0.0.1', port, path, method: method || 'GET',
      headers: data ? { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(data) } : {},
    }, x => { let d = ''; x.on('data', c => d += c); x.on('end', () => { try { resolve(JSON.parse(d)) } catch (e) { resolve(d) } }) })
    r.on('error', reject); if (data) r.write(data); r.end()
  })
}
const sleep = ms => new Promise(r => setTimeout(r, ms))

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

const results = []
function check(name, ok, detail) { results.push({ name, ok: !!ok, detail }) }

// 页面内：劫持 fetch，记录 /chat/send 的 body 并拒绝该请求（不发真消息）
const HOOK = `(function(){
  window.__apProbe = { bodies: [], blocked: 0 };
  try {
    var _f = window.fetch;
    if (_f) window.fetch = function(u, o){
      try {
        var url = (u && u.url) || u || '';
        if (String(url).indexOf('/chat/send') >= 0) {
          var b = null; try { b = o && o.body } catch(e){}
          window.__apProbe.bodies.push(typeof b === 'string' ? b : String(b));
          window.__apProbe.blocked++;
          return Promise.reject(new Error('probe-blocked'));
        }
      } catch(e){}
      return _f.apply(this, arguments);
    };
  } catch(e){ window.__apProbe.err = String(e); }
  // ★ 2026-10-16 错误定位增强：收集未捕获错误的**行列号 + 堆栈前几帧** ——
  //   否则只有 CDP console 的首行（如「at yt」），在 esbuild 压缩产物里无法定位
  //   TDZ / 未定义变量究竟出在哪一段代码。
  try {
    window.__apProbeErrs = [];
    window.addEventListener('error', function(e){
      var st = ''; try { st = (e.error && e.error.stack) ? String(e.error.stack).split('\\n').slice(0, 4).join(' << ') : '' } catch(x){}
      window.__apProbeErrs.push(String(e.message) + ' @' + e.lineno + ':' + e.colno + (st ? ' :: ' + st : ''));
    });
    window.addEventListener('unhandledrejection', function(e){
      var r = e.reason || {}; window.__apProbeErrs.push('unhandled: ' + String(r.message || r));
    });
  } catch(e){}
  return 'hooked';
})()`

const BTN_STATE = `(function(){
  var b = document.querySelector('.obtn-agent');
  if (!b) return JSON.stringify({ found: false });
  return JSON.stringify({ found: true, cls: String(b.className), active: String(b.className).indexOf('active') >= 0,
    title: b.getAttribute('title') || '' });
})()`

// 发送动作走「真实输入路径」（CDP Input 域）：
//   DOM 直接赋 value + dispatchEvent('input') 在前端为自定义受控输入时不生效
//   （Vue 的 v-model 走原生 input 事件，但部分实现读的是内部状态），实测点击发送
//   按钮后前端未发出请求。故改用 focus + Input.insertText（等效真人敲字）+ 点击发送。

;(async () => {
  const list = await reqJson(CHROME_PORT, '/json/list')
  const tab = (Array.isArray(list) ? list : []).find(t => t.type === 'page')
  if (!tab) throw new Error('未找到 CDP page tab（chrome 是否以 --remote-debugging-port=9223 启动？）')
  const ws = new WS(); await ws.connect(tab.webSocketDebuggerUrl)
  let id = 0; const pend = new Map(); const pageErrors = []
  ws.onMessage = b => {
    const m = JSON.parse(b.toString())
    if (m.id && pend.has(m.id)) { pend.get(m.id)(m); pend.delete(m.id) }
    else if (m.method === 'Runtime.consoleAPICalled' && m.params.type === 'error') {
      pageErrors.push((m.params.args || []).map(a => a.value || a.description || '').join(' ').slice(0, 200))
    }
  }
  const send = (method, params) => { const mid = ++id; ws.send({ id: mid, method, params: params || {} }); return new Promise(res => pend.set(mid, res)) }
  async function ev(expr) {
    const r = await send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true })
    if (r.result && r.result.exceptionDetails) throw new Error('eval 异常: ' + JSON.stringify(r.result.exceptionDetails).slice(0, 300))
    return r.result && r.result.result ? r.result.result.value : null
  }

  // doSend：真实输入 + 触发发送；返回诊断串（含输入框实际值），不产生真请求（被 hook 拦截）
  async function doSend() {
    // ★ 2026-10-16 修正「输入未写入 → 发不出请求」：CDP Input.insertText 依赖页面
    //   聚焦状态（headless 下实测写入为空 → 输入框空且无请求）；且**真实输入框不是
    //   textarea，而是 contenteditable 的 `.chat-input`**（RightPanel.vue:308，
    //   @input → onInput → serializeInput → inputText）—— 此前探针写的是页面上另一个
    //   textarea，inputText 从未更新 → 发送按钮 :disabled="!inputText.trim()" 恒为真
    //   → click() 不触发 @click → 无请求。现改为在 .chat-input 上 execCommand('insertText')
    //   （等效真人敲字，会触发 input 事件），失败则兜底设 textContent + 手动派发 input。
    const focused = await ev(`(function(){
      var el = document.querySelector('.chat-input');
      if (!el) return 'no-chat-input';
      el.focus();
      try {
        var sel = window.getSelection(); var range = document.createRange();
        range.selectNodeContents(el); sel.removeAllRanges(); sel.addRange(range);
      } catch (e) {}
      var ok = false;
      try { ok = document.execCommand('insertText', false, '[探针] 联动检查（不会真正发送）') } catch (e) {}
      if (!ok || !String(el.textContent || '').trim()) el.textContent = '[探针] 联动检查（不会真正发送）';
      el.dispatchEvent(new Event('input', { bubbles: true }));
      return 'chat-input' + (ok ? '+execText' : '+fallback');
    })()`)
    await sleep(400)
    const val = await ev(`((document.querySelector('.chat-input') || {}).textContent || '')`)
    const r = await ev(`(function(){
      var el = document.querySelector('.chat-input');
      if (el) el.dispatchEvent(new Event('input', { bubbles: true }));
      var sb = document.querySelector('.send-btn');
      if (!sb) return 'no-send-btn(chatLoading=' + !!(window.__PAIRCODE_CORE && window.__PAIRCODE_CORE.uiState.state.chatLoading) + ')';
      var wasDisabled = !!sb.disabled;
      sb.click();
      return 'click:send-btn wasDisabled=' + wasDisabled;
    })()`)
    return String(focused) + ' | ' + String(r) + ' | input="' + String(val).slice(0, 40) + '"'
  }

  await send('Runtime.enable'); await send('Page.enable')
  // ★ 2026-10-16 禁缓存：区域包/壳 bundle 的 URL 无内容 hash（如 ui-right-panel.js），
  //   浏览器会复用旧产物 —— 不禁用的话「改完重跑探针」可能仍在断言旧代码（假结论）。
  await send('Network.enable'); await send('Network.setCacheDisabled', { cacheDisabled: true })
  await send('Page.navigate', { url: URL_APP })
  // 等开关出现（插件装载 + 首屏）
  let appeared = false
  for (let i = 0; i < 40; i++) { await sleep(500); if (await ev(`!!document.querySelector('.obtn-agent')`)) { appeared = true; break } }
  if (!appeared) throw new Error('页面 20s 内未渲染出 .obtn-agent（插件未装载？）')
  await sleep(1200)

  await ev(HOOK)
  check('hook 已注入', (await ev(`!!window.__apProbe`)) === true, 'window.__apProbe 就绪（拦截 /chat/send）')

  // ── 会话级权威值读取（★ 2026-10-16 自主模式已改为会话级：按钮真值 = 当前会话
  //    元数据的 autonomous；全局 settings.autonomous 不再是权威值）──
  async function convAuto() {
    const convId = await ev(`(window.__PAIRCODE_CORE && window.__PAIRCODE_CORE.uiState.state.currentConvId) || ''`)
    const ws = await ev(`(window.__PAIRCODE_CORE && window.__PAIRCODE_CORE.uiState.state.workspaceRoot) || ''`)
    const list = await reqJson(APP_PORT, '/api/conversations?workspace=' + encodeURIComponent(ws))
    const arr = Array.isArray(list) ? list : (list && list.conversations) || []
    const conv = arr.filter(c => c && c.id === convId)[0]
    return { convId: convId, hasField: !!(conv && Object.prototype.hasOwnProperty.call(conv, 'autonomous')), on: conv ? !!conv.autonomous : null }
  }

  // ── A1 初始态 vs 会话级权威值 ──
  const gl0 = await reqJson(APP_PORT, '/api/settings')
  const gl0Auto = !!(gl0 && gl0.settings && gl0.settings.autonomous)
  const ca0 = await convAuto()
  const st0 = JSON.parse(await ev(BTN_STATE))
  check('A1 开关初始态与会话级值一致', st0.found && ca0.hasField && st0.active === ca0.on,
    `按钮 active=${st0.active}（title=${st0.title}） / 会话 ${ca0.convId} autonomous=${ca0.on} / 全局 autonomous=${gl0Auto}（非权威）`)

  // ── A2 点击开关 → 翻转并落盘 ──
  await ev(`document.querySelector('.obtn-agent').click()`)
  await sleep(900)
  const st1 = JSON.parse(await ev(BTN_STATE))
  const ca1 = await convAuto()
  check('A2a 点击后按钮态翻转', st1.active === !st0.active, `active: ${st0.active} → ${st1.active}`)
  check('A2b 点击后会话级值同步落盘', ca1.on === st1.active,
    `会话 ${ca1.convId} autonomous=${ca1.on} / 按钮 active=${st1.active}`)

  // ── A3 发送 → body.autonomous 应等于当前按钮态 ──
  await ev(`window.__apProbe.bodies = []`)
  const sendRes1 = await doSend()
  await sleep(1500)
  const probe1 = JSON.parse(await ev(`JSON.stringify(window.__apProbe)`))
  let body1 = null; try { body1 = JSON.parse(probe1.bodies[0] || 'null') } catch (e) {}
  check('A3 发送的 autonomous 与开关一致（第一次）',
    !!body1 && body1.autonomous === st1.active,
    `sendRes=${sendRes1} body.autonomous=${body1 ? body1.autonomous : '(无请求)'} / 按钮 active=${st1.active}`)

  // ── A4 再点回来 → 再发 ──
  await ev(`document.querySelector('.obtn-agent').click()`)
  await sleep(900)
  const st2 = JSON.parse(await ev(BTN_STATE))
  await ev(`window.__apProbe.bodies = []`)
  const sendRes2 = await doSend()
  await sleep(1500)
  const probe2 = JSON.parse(await ev(`JSON.stringify(window.__apProbe)`))
  let body2 = null; try { body2 = JSON.parse(probe2.bodies[0] || 'null') } catch (e) {}
  check('A4 再点回来：发送的 autonomous 同步翻转',
    !!body2 && body2.autonomous === st2.active && st2.active === st0.active,
    `sendRes=${sendRes2} body.autonomous=${body2 ? body2.autonomous : '(无请求)'} / 按钮 active=${st2.active}（初值 ${st0.active}）`)

  // ── A6 会话级切换不得污染全局 settings（会话级改造的核心不变量）──
  const gl1 = await reqJson(APP_PORT, '/api/settings')
  const gl1Auto = !!(gl1 && gl1.settings && gl1.settings.autonomous)
  check('A6 会话级切换不污染全局 settings', gl1Auto === gl0Auto,
    `全局 autonomous: ${gl0Auto} → ${gl1Auto}（应保持不变）`)

  const perrsStr = await ev(`JSON.stringify(window.__apProbeErrs || [])`)
  let perrs = []; try { perrs = JSON.parse(perrsStr) } catch (e) {}
  check('A5 无控制台错误', pageErrors.length === 0 && perrs.length === 0,
    (pageErrors.slice(0, 2).join(' | ') + (perrs.length ? ' || err-event: ' + perrs.slice(0, 3).join(' | ') : '')) || '无')

  console.log('\n=== 自主模式开关联动探针（应用端口 ' + APP_PORT + '）===')
  let fail = 0
  for (const r of results) { if (!r.ok) fail++; console.log((r.ok ? 'PASS  ' : 'FAIL  ') + r.name + '  —— ' + r.detail) }
  console.log('\n结果：' + (results.length - fail) + '/' + results.length + ' 通过')
  console.log('（说明：探针拦截了 /chat/send，未产生任何 LLM 调用与会话写入）')
  process.exit(fail ? 1 : 0)
})().catch(e => { console.error('探针异常：', e && e.message); process.exit(2) })
