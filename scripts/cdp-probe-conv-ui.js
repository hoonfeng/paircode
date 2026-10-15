// cdp-probe-conv-ui.js — 「会话 UI 行为」探针（新建会话立即出现 / 停止后状态不回弹）
//
// 用法：node scripts/cdp-probe-conv-ui.js [appPort]     默认 9098
// 前置：headless chrome --remote-debugging-port=9223（同 scripts/cdp-probe-autonomous.js 惯例）
//
// 背景（2026-10-16 用户反馈的两个独立症状）：
//   ① 新建会话后列表要刷新才出现 —— wb-ui 内嵌引擎对数组**原地变动**（unshift/push）
//      不可靠地触发依赖收集；修法 = 整体替换数组（state.conversations = [新, ...旧]）。
//   ② 点「停止」后仍显示运行中、停止按钮不消失 —— 停止是**异步取消**，后端在途的
//      status 帧仍把该会话报成 running，processStatus 会把 loadingByConv 置回 true
//      （→ chatLoading 恢复 → 停止按钮又出现）。修法 = userStoppedConvs 标记：
//      用户停止后，在该会话真正结束（done）或重新发送前忽略这类回流。
//
// 断言（全部在真实页面内驱动**真实导出函数**，不发消息、不调 LLM、不写会话消息数据）：
//   B1  点「新建对话」→ state.conversations 长度 +1 且 DOM 中「新对话」**立即**出现（不等刷新）
//   B2a 对照：无标记时 processStatus(runningConvs) 会把会话置回运行中（证明回流机制本身有效，
//       排除「断言恒真」的假通过）
//   B2b 实验：有「用户已停止」标记时同一帧被忽略（loadingByConv / chatLoading 不被置回 true）
//   B2c processAgentDone 结束后标记被清除（不影响下一次运行）
//   B3  无控制台错误
const http = require('http'); const net = require('net'); const crypto = require('crypto')

const APP_PORT = process.argv[2] || '9098'
const CHROME_PORT = 9223
const URL_APP = 'http://localhost:' + APP_PORT + '/'

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

;(async () => {
  const list = await new Promise((res, rej) => {
    const r = http.request({ host: '127.0.0.1', port: CHROME_PORT, path: '/json/list' }, x => {
      let d = ''; x.on('data', c => d += c); x.on('end', () => { try { res(JSON.parse(d)) } catch (e) { rej(e) } })
    }); r.on('error', rej); r.end()
  })
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

  await send('Runtime.enable'); await send('Page.enable')
  // ★ 禁缓存：区域包/壳 bundle 的 URL 无内容 hash，否则可能仍在断言旧产物（假结论）
  await send('Network.enable'); await send('Network.setCacheDisabled', { cacheDisabled: true })
  await send('Page.navigate', { url: URL_APP })

  // 等页面就绪（共享核心 + 会话列表）
  let ready = false
  for (let i = 0; i < 40; i++) {
    await sleep(500)
    if (await ev(`!!(window.__PAIRCODE_CORE && window.__PAIRCODE_CORE.uiState && window.__PAIRCODE_CORE.uiState.state.currentConvId)`)) { ready = true; break }
  }
  if (!ready) throw new Error('页面 20s 内未就绪（__PAIRCODE_CORE / 当前会话缺失）')
  await sleep(1200)

  // 页面内辅助面（只读状态 + 定位「新建对话」按钮）
  await ev(`(function(){
    window.__cq = {
      S: function(){ return window.__PAIRCODE_CORE.uiState.state },
      A: function(){ return window.__PAIRCODE_CORE.agentEvents },
      btn: function(){
        var bs = [].slice.call(document.querySelectorAll('button'));
        return bs.filter(function(b){
          var t = String(b.title || '') + ' ' + String(b.getAttribute('aria-label') || '');
          var r = b.getBoundingClientRect();
          return r && r.width > 0 && r.height > 0 && /新建(对话|会话)/.test(t);
        })[0] || null;
      },
      count: function(){ return (document.body.innerText.match(/新对话/g) || []).length }
    };
    return 'ready';
  })()`)

  // ── B1 新建会话：列表应立即出现（不等刷新）──
  const b1a = JSON.parse(await ev(`JSON.stringify({ len: __cq.S().conversations.length, txt: __cq.count(), hasBtn: !!__cq.btn() })`))
  const clicked = await ev(`(function(){ var b = __cq.btn(); if (!b) return 'no-btn'; b.click(); return 'click:' + String(b.className) })()`)
  await sleep(1500)
  const b1b = JSON.parse(await ev(`JSON.stringify({ len: __cq.S().conversations.length, txt: __cq.count(), curId: __cq.S().currentConvId, top: (__cq.S().conversations[0]||{}).title })`))
  check('B1 新建会话后列表立即出现（无需刷新）',
    b1a.hasBtn && b1b.len === b1a.len + 1 && b1b.txt > b1a.txt,
    `按钮=${clicked} 列表 ${b1a.len} → ${b1b.len}（DOM「新对话」出现次数 ${b1a.txt} → ${b1b.txt}）top="${b1b.top}"`)

  // ── B2 停止后状态不回弹（对照 + 实验 + 清除）──
  const b2 = JSON.parse(await ev(`(function(){
    var A = __cq.A(), S = __cq.S();
    var id = S.currentConvId;
    var out = { convId: id, chatLoadingWritable: true };
    function reset(){ S.loadingByConv[id] = false; S.agentRunningByConv[id] = false; try { S.chatLoading = false } catch (e) { out.chatLoadingWritable = false } }
    // ★ 「停止按钮是否回弹」的真实链条 = processStatus（回流置 loadingByConv）
    //   → switchConv:2404 的同步语句 state.chatLoading = state.loadingByConv[id] || false
    //   → chatLoading → 停止按钮。此处以 syncConv() 复现该行，使对照实验覆盖完整链条
    //   （processStatus 本身不直接写 chatLoading，只写 loadingByConv）。
    function syncConv(){ try { S.chatLoading = S.loadingByConv[id] || false } catch (e) {} }
    // 对照：无标记 → 回流生效（loadingByConv=true → syncConv → chatLoading=true，即会回弹）
    reset();
    A.processStatus({ runningConvs: [id], runningByWorkspace: {} });
    syncConv();
    out.ctrl_loading = S.loadingByConv[id] === true;
    out.ctrl_chatLoading = S.chatLoading === true;
    // 实验：打上「用户已停止」标记 → 同一帧被忽略（loadingByConv 保持 false → 不回弹）
    A.userStoppedConvs.add(id);
    reset();
    A.processStatus({ runningConvs: [id], runningByWorkspace: {} });
    syncConv();
    out.marked_loading = S.loadingByConv[id] === true;
    out.marked_chatLoading = S.chatLoading === true;
    // 会话真正结束 → 标记应被清除（不影响下一次运行）
    A.processAgentDone(id, { doneReason: 'stopped' });
    out.markCleared = A.userStoppedConvs.has(id) === false;
    return JSON.stringify(out);
  })()`))
  check('B2a 对照：无标记时回流经 switchConv 恢复 chatLoading', b2.ctrl_loading === true && b2.ctrl_chatLoading === true,
    `loadingByConv=${b2.ctrl_loading} chatLoading=${b2.ctrl_chatLoading}（应为 true，证明回流链条本身有效，排除断言恒真）`)
  check('B2b 已停止的会话忽略运行中回流（切回会话停止按钮不回弹）',
    b2.marked_loading === false && b2.marked_chatLoading === false,
    `loadingByConv=${b2.marked_loading} chatLoading=${b2.marked_chatLoading}（应为 false，即停止态保持、按钮不回弹）`)
  check('B2c 会话结束后清除停止标记', b2.markCleared === true,
    `userStoppedConvs.has(${b2.convId}) = ${!b2.markCleared}（应为 false）`)

  check('B3 无控制台错误', pageErrors.length === 0, pageErrors.length ? pageErrors.slice(0, 3).join(' | ') : '无')

  console.log('\n=== 会话 UI 行为探针（应用端口 ' + APP_PORT + '）===')
  let fail = 0
  for (const r of results) { if (!r.ok) fail++; console.log((r.ok ? 'PASS  ' : 'FAIL  ') + r.name + '  —— ' + r.detail) }
  console.log('\n结果：' + (results.length - fail) + '/' + results.length + ' 通过')
  console.log('（说明：B1 会真实创建一个空会话；B2 只驱动前端导出函数，不发消息、不调 LLM）')
  process.exit(fail ? 1 : 0)
})().catch(e => { console.error('探针异常：', e && e.message); process.exit(2) })
