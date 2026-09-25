// ═══════════════════════════════════════════════════════════════
// focus-probe.mjs — 专注模式「收纳完备性」探针（headless Edge + 手写 CDP，无第三方依赖）
//
// 用途：Ctrl+K 专注模式的自动检查 —— 进入前后两轮快照对比，输出：
//   ① 登记表期望断言（FOCUS_PANELS：4 面板收起 / 对话面板显示 / mainTab=对话）；
//   ② 直系区域通用枚举：列出「专注后仍占空间且不在恒显白名单内」的区域（UNCOLLECTED 告警）——
//      「以后新增组件被自动发现」的机制保障；
//   ③ 退出专注还原校验；结束关闭浏览器（无残留，独立 profile 不影响用户浏览器）。
//
// 用法：node scripts/source-update/focus-probe.mjs [--web-port 9090] [--cdp-port 9227]
//               [--timeout 120000] [--keep-open]
// 退出码：0=通过 / 3=发现未收纳（告警级） / 9=脚本异常/超时 / 10=环境不可用（浏览器/页面未就绪）
//
// 集成：update.mjs 部署成功后调用 runFocusProbe()（结果记录到 update.log /
//   last-result.json；异常仅告警不阻断部署流程——本模块被 import 时绝不调用 process.exit）。
// ═══════════════════════════════════════════════════════════════
import fs from 'node:fs'
import os from 'node:os'
import net from 'node:net'
import path from 'node:path'
import http from 'node:http'
import crypto from 'node:crypto'
import { spawn } from 'node:child_process'
import { pathToFileURL } from 'node:url'
import { parseArgs } from './lib.mjs'

// ─── 恒显骨架白名单（与 ShellApp.vue 模板/网格定位对应；宁可误报不可漏报）──
//   plugin-area-titlebar    顶栏（跨列 40px）
//   plugin-area-activitybar 活动栏（48px 竖条）
//   main-area               主区（对话/编辑器/市场/工具集/插件视图容器）
//   app-statusbar-host      状态栏（跨列 28px）
//   modals-host/modals-empty 全屏浮层容器（fixed 不占网格格；不占空间时自然被忽略）
const WHITELIST = [
  'plugin-area-titlebar', 'plugin-area-activitybar', 'main-area',
  'app-statusbar-host', 'modals-host', 'modals-empty',
]

// ─── 登记表期望值（与 ui-state.js FOCUS_PANELS 对应；专注态）──
const EXPECT_FOLD = {
  sidebarVisible: false,
  convListVisible: false,
  statsRailVisible: false,
  bottomPanelVisible: false,
  rightPanelVisible: true,
  mainTab: 'conversation',
}

const sleep = (ms) => new Promise(r => setTimeout(r, ms))

function httpJson(port, p) {
  return new Promise((res, rej) => {
    http.get({ host: '127.0.0.1', port, path: p }, x => {
      let d = ''
      x.on('data', c => d += c)
      x.on('end', () => { try { res(JSON.parse(d)) } catch (e) { rej(e) } })
    }).on('error', rej)
  })
}

// ─── 最小 WebSocket 客户端（CDP 用；移植自项目内测试脚本，无第三方依赖）──
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
      // error 挂 handler 防 uncaught（连接期的失败 reject 首次 promise；连接后的失败由 pend 兜底）
      this.sock.on('error', (e) => { try { rej(e) } catch { /* ignore */ } })
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
      let p = this.buf.slice(off, off + len)
      if (masked) { const o = Buffer.alloc(len); for (let i = 0; i < len; i++) o[i] = p[i] ^ mask[i & 3]; p = o }
      this.buf = this.buf.slice(off + len)
      const op = b0 & 0x0f
      if (op === 1) this.frag.push(p)
      if ((b0 & 0x80) && this.frag.length) {
        const msg = Buffer.concat(this.frag); this.frag = []
        // ★ 防护：未设回调的连接（如仅用于发送 Browser.close 的浏览器连接）忽略入站消息
        if (typeof this.onMessage === 'function') this.onMessage(msg)
      }
    }
  }
  send(o) {
    try {
      const p = Buffer.from(JSON.stringify(o))
      const len = p.length
      let h
      if (len < 126) h = Buffer.from([0x81, 0x80 | len])
      else { h = Buffer.alloc(4); h[0] = 0x81; h[1] = 0x80 | 126; h.writeUInt16BE(len, 2) }
      const mask = crypto.randomBytes(4)
      const m = Buffer.alloc(len)
      for (let i = 0; i < len; i++) m[i] = p[i] ^ mask[i & 3]
      this.sock.write(Buffer.concat([h, mask, m]))
    } catch { /* socket 已关闭等场景静默（由 pend reject 兜底） */ }
  }
  close() { try { this.sock && this.sock.destroy() } catch { /* ignore */ } }
}

// ─── 定位 msedge ─────────────────────────────────────────────
function findEdge() {
  const cands = [
    'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',
    'C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe',
  ]
  for (const c of cands) { if (fs.existsSync(c)) return c }
  return 'msedge' // 交给 PATH 解析（找不到时 spawn 报错，由调用方提示环境问题）
}

// ─── 快照表达式（页面内执行；返回 JSON 字符串）────────────────
const SNAPSHOT_EXPR = `(() => {
  const root = document.querySelector('.app-root')
  const rect = el => { if (!el) return null; const r = el.getBoundingClientRect(); return [Math.round(r.width), Math.round(r.height)] }
  const children = root ? Array.from(root.children).map(el => ({
    cls: (el.className && String(el.className)) || el.tagName.toLowerCase(),
    w: Math.round(el.getBoundingClientRect().width),
    h: Math.round(el.getBoundingClientRect().height),
  })) : []
  const st = window.__state || {}
  return JSON.stringify({
    focusMode: st.focusMode,
    sidebarVisible: st.sidebarVisible,
    convListVisible: st.convListVisible,
    statsRailVisible: st.statsRailVisible,
    bottomPanelVisible: st.bottomPanelVisible,
    rightPanelVisible: st.rightPanelVisible,
    mainTab: st.panels && st.panels.mainTab,
    children,
    conv: rect(document.querySelector('.conversation-container')),
  })
})()`

// ─── 主流程 ──────────────────────────────────────────────────
// opts: { webPort=9090, cdpPort=9227, timeoutMs=120000, keepOpen=false, log }
// 返回：{ ok, uncollected:[], missing:[], note, envFail?, error? }
export async function runFocusProbe(opts = {}) {
  const webPort = Number(opts.webPort || 9090)
  const cdpPort = Number(opts.cdpPort || 9227)
  const timeoutMs = Number(opts.timeoutMs || 120000)
  const say = opts.log || ((m) => console.log(m))
  const errors = []          // 登记表期望不符（断言失败）
  const uncollected = []     // 直系枚举发现的未收纳区域（通用告警）
  const notes = []
  let timedOut = false
  let cleaned = false

  const profileDir = path.join(os.tmpdir(), 'paircode-focus-probe-' + process.pid)
  const edgePath = findEdge()
  say('[focus-probe] web=http://127.0.0.1:' + webPort + ' cdp=' + cdpPort + ' edge=' + edgePath)

  let proc = null
  let ws = null
  let browserWs = null
  let pend = null

  // 幂等清理：先优雅关浏览器（Browser.close）→ 关 socket（并 reject 全部挂起）→ kill 兜底 → 删临时 profile。
  async function cleanup() {
    if (cleaned) return
    cleaned = true
    if (!opts.keepOpen) {
      try { if (browserWs) browserWs.send({ id: 9001, method: 'Browser.close', params: {} }) } catch { /* ignore */ }
      await sleep(800)
    }
    if (pend) { for (const [, p] of pend) { try { p.rej(new Error('focus-probe cleanup')) } catch { /* ignore */ } } pend.clear() }
    try { ws && ws.close() } catch { /* ignore */ }
    try { browserWs && browserWs.close() } catch { /* ignore */ }
    if (!opts.keepOpen) {
      try { proc && proc.kill() } catch { /* ignore */ }
      await sleep(300)
      try { fs.rmSync(profileDir, { recursive: true, force: true }) } catch { /* ignore */ }
    }
  }

  // 超时护栏：标记 + 清理（将挂起的调用以 reject 收尾）；仅 CLI 模式才 exit。
  const hardTimer = setTimeout(() => {
    timedOut = true
    say('[focus-probe] 超时（' + timeoutMs + 'ms），强制收尾')
    cleanup()
  }, timeoutMs)
  hardTimer.unref?.()

  try {
    // ① 启动 headless Edge（独立 profile；窗口 1440x900 对齐设计稿四列布局）
    fs.mkdirSync(profileDir, { recursive: true })
    proc = spawn(edgePath, [
      '--headless=new', '--disable-gpu', '--no-first-run', '--no-default-browser-check',
      '--disable-extensions', '--disable-sync', '--window-size=1440,900',
      '--user-data-dir=' + profileDir,
      '--remote-debugging-port=' + cdpPort,
      'http://127.0.0.1:' + webPort + '/',
    ], { stdio: 'ignore', windowsHide: true })

    // ② 等 CDP 端口就绪
    let ver = null
    for (let i = 0; i < 40; i++) {
      try { ver = await httpJson(cdpPort, '/json/version'); break } catch { await sleep(500) }
    }
    if (!ver) { say('[focus-probe] CDP 未就绪（端口 ' + cdpPort + '）：浏览器启动失败或端口被占用，可 --cdp-port 换端口'); return { ok: false, envFail: true, uncollected, missing: errors, note: 'CDP 未就绪' } }

    // ③ 找页面 tab（启动参数已导航到目标 URL）
    let tab = null
    for (let i = 0; i < 30; i++) {
      try {
        const list = await httpJson(cdpPort, '/json/list')
        tab = list.find(t => t.type === 'page' && String(t.url).includes(':' + webPort))
        if (tab) break
      } catch { /* ignore */ }
      await sleep(500)
    }
    if (!tab) { say('[focus-probe] 未找到页面 tab（webPort=' + webPort + '）'); return { ok: false, envFail: true, uncollected, missing: errors, note: '未找到页面 tab' } }

    // ④ 连页面 WS + 浏览器 WS（后者仅用于优雅关闭）
    ws = new WS()
    await ws.connect(tab.webSocketDebuggerUrl)
    let id = 0
    pend = new Map()
    ws.onMessage = b => {
      const m = JSON.parse(b.toString())
      if (m.id && pend.has(m.id)) { const p = pend.get(m.id); pend.delete(m.id); m.error ? p.rej(new Error(m.error.message)) : p.res(m.result) }
    }
    const send = (method, params) => new Promise((res, rej) => {
      const i = ++id
      pend.set(i, { res, rej })
      ws.send({ id: i, method, params })
    })
    const ev = async e => (await send('Runtime.evaluate', { expression: e, returnByValue: true, awaitPromise: true })).result.value
    try {
      const bwsUrl = ver && ver.webSocketDebuggerUrl
      if (bwsUrl) { browserWs = new WS(); await browserWs.connect(bwsUrl) }
    } catch { /* 关闭走 kill 兜底 */ }

    // ⑤ 等壳就绪
    say('[focus-probe] 等待页面壳就绪…')
    let ready = false
    for (let i = 0; i < 45; i++) {
      try {
        const r = await ev(`(!!window.__state && !!document.querySelector('.app-root') && !!document.querySelector('.conversation-container')) ? 'ready' : 'no'`)
        if (r === 'ready') { ready = true; break }
      } catch { /* ignore */ }
      await sleep(700)
    }
    if (!ready) { say('[focus-probe] 页面壳未就绪（__state/.app-root/.conversation-container）'); return { ok: false, envFail: true, uncollected, missing: errors, note: '页面壳未就绪' } }
    await sleep(1200) // 插件客户端半二次渲染稳定

    // ★ 自测模式（CLI --inject-test）：向 .app-root 注入一个假「未知区域」——
    //   验证探针枚举能报出未收纳元素（负向测试探针自身有效性；常规运行不用）。
    if (opts.injectTest) {
      await ev(`(() => { const d = document.createElement('div'); d.className = 'test-unknown-region'; d.style.cssText = 'width:40px;height:40px;background:red'; document.querySelector('.app-root').appendChild(d); return 'injected' })()`)
      say('[focus-probe] 自测模式：已注入假区域 .test-unknown-region（40x40）')
    }

    const snapshot = async () => JSON.parse(await ev(SNAPSHOT_EXPR))
    const pressCtrlK = () => ev(`(document.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', code: 'KeyK', ctrlKey: true, bubbles: true, cancelable: true })), 'ok')`)

    const isSpace = (c) => !!c && c.w > 0 && c.h > 0
    const isWhitelisted = (c) => {
      const words = String(c.cls || '').split(/\s+/)
      return WHITELIST.some(w => words.includes(w))
    }

    // ⑥ 基线快照 + 初始状态
    const S0 = await snapshot()
    const startedFocused = S0.focusMode === true
    say('[focus-probe] 基线快照：直系区域 ' + S0.children.length + ' 个，初始 focusMode=' + startedFocused)

    let S1 = S0
    if (!startedFocused) {
      await pressCtrlK(); await sleep(700) // 进入专注
      S1 = await snapshot()
      say('[focus-probe] 已进入专注（Ctrl+K 合成事件）')
    } else {
      notes.push('页面初始即为专注态；跳过进入动作，仅做当前态检查、不做往返还原校验')
      say('[focus-probe] 页面初始即专注态：跳过进入动作')
    }

    // ⑦ 判定 A：登记表期望断言
    const stateMap = { sidebarVisible: S1.sidebarVisible, convListVisible: S1.convListVisible, statsRailVisible: S1.statsRailVisible, bottomPanelVisible: S1.bottomPanelVisible, rightPanelVisible: S1.rightPanelVisible, mainTab: S1.mainTab }
    for (const [k, expected] of Object.entries(EXPECT_FOLD)) {
      const actual = stateMap[k]
      if (actual !== expected) errors.push(k + '=' + JSON.stringify(actual) + '（期望 ' + JSON.stringify(expected) + '）')
    }
    say('[focus-probe] 状态断言：' + (errors.length === 0 ? '全部符合登记表期望 ✓' : ('不符 ' + errors.length + ' 项：' + errors.join('; '))))

    // ⑧ 判定 B：直系区域通用枚举（未收纳 = 专注前占空间 且 专注后仍占空间 且 非白名单）
    for (const c0 of S0.children) {
      if (isWhitelisted(c0)) continue
      if (!isSpace(c0)) continue
      const c1 = S1.children.find(x => x.cls === c0.cls)
      if (c1 && isSpace(c1)) uncollected.push({ cls: c0.cls, before: [c0.w, c0.h], after: [c1.w, c1.h] })
    }
    if (uncollected.length) {
      say('[focus-probe] ⚠ 发现未收纳区域 ' + uncollected.length + ' 个：')
      for (const u of uncollected) {
        say('    ' + u.cls + '  (before ' + u.before[0] + 'x' + u.before[1] + ' → after ' + u.after[0] + 'x' + u.after[1] + ')')
      }
      say('    → 请按 ui-state.js FOCUS_PANELS 登记规则处理（登记一行 + 同步 ShellApp.gridStyle 列宽）')
    } else {
      say('[focus-probe] 直系区域枚举：白名单外无残留占位 ✓')
    }

    // ⑨ 判定 C：对话容器仍可见（纯对话）
    const convOk = S1.conv && S1.conv[0] > 0 && S1.conv[1] > 0
    if (!convOk) errors.push('对话容器不可见（纯对话语义破坏）：conv=' + JSON.stringify(S1.conv))
    say('[focus-probe] 对话容器可见性：' + (convOk ? '✓' : '✗ ' + JSON.stringify(S1.conv)))

    // ⑩ 退出专注 + 还原校验（仅当本探针执行了进入动作）
    if (!startedFocused) {
      await pressCtrlK(); await sleep(700)
      const S2 = await snapshot()
      const restoreBad = []
      for (const [k, v0] of Object.entries({ sidebarVisible: S0.sidebarVisible, convListVisible: S0.convListVisible, statsRailVisible: S0.statsRailVisible, bottomPanelVisible: S0.bottomPanelVisible, rightPanelVisible: S0.rightPanelVisible, mainTab: S0.mainTab })) {
        if (S2[k] !== v0) restoreBad.push(k + '=' + JSON.stringify(S2[k]) + '（原 ' + JSON.stringify(v0) + '）')
      }
      for (const c0 of S0.children) {
        if (isWhitelisted(c0)) continue
        const c2 = S2.children.find(x => x.cls === c0.cls)
        const was = isSpace(c0), now = c2 ? isSpace(c2) : false
        if (was !== now) restoreBad.push(c0.cls + ' 占位状态未复原（原 ' + was + ' → 现 ' + now + '）')
      }
      if (restoreBad.length) { errors.push('退出还原不符：' + restoreBad.join('; ')); say('[focus-probe] 退出还原校验：✗ ' + restoreBad.join('; ')) }
      else say('[focus-probe] 退出还原校验：✓（状态与直系区域占位复原）')
    }

    await cleanup()

    const ok = errors.length === 0 && uncollected.length === 0 && !timedOut
    const note = timedOut ? 'TIMEOUT'
      : ok
        ? ('PASS' + (startedFocused ? '（初始专注态，未做往返）' : '（含往返还原校验）') + (notes.length ? '；' + notes.join('；') : ''))
        : ('WARN' + (uncollected.length ? ' 未收纳 ' + uncollected.length + ' 个' : '') + (errors.length ? ' 断言不符 ' + errors.length + ' 项' : ''))
    say('[focus-probe] RESULT: ' + note)
    return { ok, uncollected, missing: errors, note }
  } catch (e) {
    say('[focus-probe] 异常：' + (e && e.message || e))
    await cleanup()
    return { ok: false, error: String(e && e.message || e), uncollected, missing: errors, note: timedOut ? 'TIMEOUT' : '脚本异常' }
  } finally {
    clearTimeout(hardTimer)
    await cleanup()
  }
}

// ─── CLI 入口（仅直接运行本文件时生效；被 import 时不触发）───
const isCli = process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href
if (isCli) {
  const args = parseArgs(process.argv.slice(2))
  const res = await runFocusProbe({
    webPort: Number(args['web-port'] || 9090),
    cdpPort: Number(args['cdp-port'] || 9227),
    timeoutMs: Number(args['timeout'] || 120000),
    keepOpen: !!args['keep-open'],
    injectTest: !!args['inject-test'],
  })
  process.exit(res.envFail ? 10 : (res.ok ? 0 : (res.error ? 9 : 3)))
}
