// ═══════════════════════════════════════════════════════════════
// wechat-bridge — 微信 ClawBot 桥（插件壳 · host 半）
//
// 职责（对照 docs/wechat-bridge-go-plan.md §2.2）：
//   1. 拉起/监督 Go 常驻桥（bin/wxbridge.exe），stdout 行 JSON 事件流 →
//      内存状态 + 浏览器事件（client:wechat-bridge:state）；
//   2. HTTP 面（/api/ext/wechat-bridge/*）：状态/账号/联系人/发送/扫码入口，
//      全部代理到桥的本地回环（127.0.0.1:<port>，端口对页面隐藏只留重定向）；
//   3. Agent 工具：wechat_send（主动发送）/ wechat_contacts（联系人查询）；
//   4. 生命周期：正常停机（stop 命令 → 2s 超时强杀）/ 插件卸载（ctx.effect）/
//      宿主退出（内核 KillAllBackgroundProcesses + 桥 stdin EOF 自杀，双保险）。
//
// 进程模型：ctx.process.runBackground 起 bash -c 包装（Git Bash 优先），stdin/stdout
// 管道直连桥的 stdio 控制面（桥为纯行协议：stdout 只出事件 JSON，日志走事件）。
// 读增量用 writeStdin({chars:''})（readNew 增量游标；yieldMs 极小=纯轮询）。
//
// 已知边界：
//   · /qr.png 为二进制，插件路由响应体仅支持字符串（goja 无 Buffer）——故扫码入口
//     用 302 重定向到桥的真实 /qr 页面（页面内相对资源在桥域下自洽），不做二进制代理；
//   · 桥 stdout 事件若被管道从中切断（多字节字符跨界）理论上可能损坏单行——壳按行
//     缓冲 + 状态以 /status 权威接口校正（面板每次刷新先拉 HTTP 再叠事件）。
// ═══════════════════════════════════════════════════════════════

const PLUGIN = 'wechat-bridge'
const HTTP_BASE = '/api/ext/wechat-bridge'
const BRIDGE_PORT = 9097            // 桥默认端口（被占时桥自动 +1..+9，以 hello 事件为准）
const POLL_MS = 1500                // stdout 轮询间隔
const LOG_KEEP = 120                // 壳内日志环形条数
const RESTART_DELAYS = [2000, 5000, 15000, 30000, 60000] // 监督重启退避（2s→60s）
const MAX_RESTARTS = 5              // 连续重启上限（超限转手动）
const STABLE_RESET_MS = 60000       // 稳定运行 60s 后重启计数归零

const DEFAULT_SETTINGS = {
  enabled: false,
  autoStart: true,
  defaultConvId: 'conv_wx_clawbot',
  workspaceRoot: '',
  paircodeUrl: '',
}

return {
  name: PLUGIN,
  purpose: '微信 ClawBot 桥：手机微信 ⇄ PairCode 双向（多账号/媒体/主动发送）',
  inject: ['fs', 'logger', 'web'],

  apply(ctx) {
    const logRef = ctx.logger(PLUGIN)

    // ── 每实例运行时状态（闭包持有；插件重载 = 全新实例）──
    const st = {
      want: false,        // 期望运行（false 时轮询不再拉起）
      procId: 0,          // ctx.process.runBackground 进程 id
      running: false,     // 桥已启动（乐观置位，以事件/poll 校准）
      port: 0,            // 桥实际端口（hello 事件）
      pid: 0,             // 桥进程 pid（hello 事件）
      reused: false,      // 复用了既有实例（单实例锁命中）
      accounts: [],       // 桥账号状态（status/account 事件）
      logs: [],           // 壳侧日志环形
      restarts: 0,        // 当前连续重启次数
      lastError: '',
      external: false,    // 桥由外部实例提供（复用探活；本壳不托管该进程）
      token: '',          // 桥写操作 token（启动时生成注入）
      paircodeUrl: '',    // 桥连接的 PairCode 地址
      uiOrigin: '',       // client 半上报的浏览器 origin（最真实的宿主地址）
      lineBuf: '',        // stdout 按行缓冲（半行不出）
      pollTimer: null,
      restartTimer: null,
      stableTimer: null,
      emitPending: false,
    }

    // ─────────────────────────── 设置 ───────────────────────────

    ctx.registerSettings({
      key: PLUGIN,
      title: '微信桥（wechat-bridge）',
      fields: [
        { name: 'enabled', label: '启用微信桥', type: 'checkbox', default: false, hint: '总开关：未启用时「启动」会被拒绝' },
        { name: 'autoStart', label: '启动时自动运行', type: 'checkbox', default: true, hint: '宿主装载插件后自动拉起桥（需同时勾选「启用微信桥」）' },
        { name: 'defaultConvId', label: '默认会话 ID', type: 'text', default: 'conv_wx_clawbot', hint: '桥在 PairCode 内使用的会话名（每账号独立）' },
        { name: 'workspaceRoot', label: '默认工作区（空=主工作区）', type: 'text', default: '', hint: '桥数据目录落在此工作区的 .pair/wechat-bridge/' },
        { name: 'paircodeUrl', label: 'PairCode 地址（空=自动探测）', type: 'text', default: '', hint: '一般不需要填；自动顺序：浏览器上报 > WEB_PORT 环境变量 > http://127.0.0.1:9090' },
        { name: 'qrLink', label: '扫码登录', type: 'link', href: HTTP_BASE + '/qr', linkText: '打开扫码页' },
      ],
    })

    function loadCfg() {
      let cur = {}
      try { cur = ctx.getSettings(PLUGIN) || {} } catch (e) { cur = {} }
      return Object.assign({}, DEFAULT_SETTINGS, cur)
    }

    // ─────────────────────────── 小工具 ───────────────────────────

    function now() { return Date.now() }

    function genToken() {
      return 'wx' + now().toString(36) + Math.random().toString(36).slice(2, 10)
    }

    function toSlash(p) { return String(p || '').replace(/\\/g, '/').replace(/\/+$/, '') }

    function log(msg, level) {
      const entry = { t: now(), level: level || 'info', msg: String(msg).slice(0, 400) }
      st.logs.push(entry)
      if (st.logs.length > LOG_KEEP) st.logs = st.logs.slice(-LOG_KEEP)
      try {
        if (level === 'error') logRef.error(entry.msg)
        else if (level === 'warn') logRef.warn(entry.msg)
        else logRef.info(entry.msg)
      } catch (e) { /* 日志失败不影响主流程 */ }
    }

    function snapshot() {
      return {
        running: st.running, pid: st.pid, port: st.port, reused: st.reused,
        external: st.external,
        want: st.want, restarts: st.restarts, lastError: st.lastError,
        paircodeUrl: st.paircodeUrl, accounts: st.accounts,
        logs: st.logs.slice(-40), ts: now(),
      }
    }

    // 状态事件推送（400ms 合并节流：桥活跃时日志密集，防刷爆浏览器事件队列）
    // ★ 事件名统一 ui: 前缀（惯例，见 agent-teams 的 'ui:agent-teams/change'）。
    //   client 半（client.js）同步监听 'ui:wechat-bridge/state'——两者一同新增，
    //   不存在旧监听方（本插件首次交付）。try/catch 保留：事件推送异常不中断节流。
    function emitState() {
      if (st.emitPending) return
      st.emitPending = true
      ctx.timeout(() => {
        st.emitPending = false
        try { ctx.emit('ui:wechat-bridge/state', snapshot()) } catch (e) { /* ignore */ }
      }, 400)
    }

    // ─────────────────────── 二进制 / 地址探测 ───────────────────────

    function findBinary() {
      // ★ 2026-09-16 修复：ctx.fs.exists 走「工作区归属检查」（resolvePath 只认
      //   工作区根内路径）——本插件目录在安装目录侧（D:\PairCode\.pair\plugins\
      //   wechat-bridge），位于工作区（如 E:\paircode-master）之外 → 探测恒为
      //   false，doStart 误报「未找到桥二进制」而无法启动。
      //   改为直接基于 ctx.binary.dir()（宿主给出的插件目录绝对路径）拼候选；
      //   存在性由 runBackground 执行结果体现（缺文件时 shell 报 No such file /
      //   command not found，错误随启动日志带出）。两候选保持原有兼容：.exe 优先
      //   （Windows 构建产物，build-wx-bridge.mjs 产出），无扩展名兜底保留。
      const dir = toSlash(ctx.binary.dir())
      return dir + '/bin/wxbridge.exe'
    }

    function detectPaircodeUrl(cfg) {
      if (cfg.paircodeUrl) return toSlash(cfg.paircodeUrl)
      if (st.uiOrigin) return toSlash(st.uiOrigin)
      let envPort = ''
      try {
        if (typeof process !== 'undefined' && process.env) envPort = String(process.env.WEB_PORT || '')
      } catch (e) { /* 沙箱无 process 时兜底默认 */ }
      if (envPort) return 'http://127.0.0.1:' + envPort
      return 'http://127.0.0.1:9090'
    }

    // ─────────────────────── 进程生命周期 ───────────────────────

    function doStart(isRestart) {
      if (st.running || st.procId) return { ok: true, already: true }
      const cfg = loadCfg()
      if (!cfg.enabled) {
        return { ok: false, error: '微信桥未启用：请先在「设置 → 插件 → 微信桥」勾选「启用微信桥」' }
      }
      const bin = findBinary()
      if (!bin) {
        return { ok: false, error: '未找到桥二进制（bin/wxbridge.exe）——请先运行 scripts/build-wx-bridge.mjs 构建' }
      }
      const wsRoot = toSlash(cfg.workspaceRoot || ctx.app.workspaceRoot || ctx.app.root || '')
      if (!wsRoot) return { ok: false, error: '无法确定工作区根（设置「默认工作区」或打开一个工作区）' }
      const dataDir = wsRoot + '/.pair/wechat-bridge'
      try { if (!ctx.fs.exists(dataDir)) ctx.fs.mkdir(dataDir, true) } catch (e) { /* 桥自身也会建 */ }

      st.paircodeUrl = detectPaircodeUrl(cfg)
      st.token = genToken()
      const cmd = '"' + bin + '" --data-dir "' + dataDir + '" --paircode "' + st.paircodeUrl +
        '" --port ' + BRIDGE_PORT + ' --token "' + st.token + '" --workspace "' + wsRoot + '"'
      log('启动桥：' + cmd)

      let r
      try {
        r = ctx.process.runBackground(cmd, wsRoot)
      } catch (e) {
        return { ok: false, error: '启动失败：' + String(e && e.message || e) }
      }
      st.procId = Number(r && r.id) || 0
      if (!st.procId) return { ok: false, error: 'runBackground 未返回进程 id' }
      st.want = true
      st.running = true        // 乐观置位；poll 首轮即校准
      if (!isRestart) st.restarts = 0
      startPoll()
      emitState()
      return { ok: true, id: st.procId }
    }

    function stop(reason) {
      if (st.external && !st.procId) {
        // 桥由外部实例提供：本壳无该进程句柄，不能代为停止（避免误报成功）
        log('桥由外部实例提供（pid=' + st.pid + '），无法从插件停止——请手动结束该进程', 'warn')
        emitState()
        return
      }
      st.want = false
      stopPolling()
      if (st.restartTimer) { try { st.restartTimer() } catch (e) { } st.restartTimer = null }
      const id = st.procId
      st.procId = 0
      st.running = false
      st.port = 0
      st.pid = 0
      if (id) {
        log('停止桥（' + (reason || '手动') + '）')
        // 优雅：发 stop 命令（桥收令后走自身退出流程）
        try { ctx.process.writeStdin({ id: id, chars: '{"cmd":"stop"}\n', yieldMs: 1 }) } catch (e) { /* ignore */ }
        // 兜底：1.5s 后若仍存活则强杀（先探 done 防 PID 复用误杀）
        ctx.timeout(() => {
          try {
            const rr = ctx.process.readOutput(id)
            if (rr && rr.done) return
          } catch (e) { return } // 进程记录已清理 → 无事可做
          try { ctx.process.kill(id) } catch (e) { /* ignore */ }
        }, 1500)
      }
      emitState()
    }

    function restart() {
      if (st.external) {
        // 外部实例：无句柄可停——放弃托管声明后重新探测（新探活进程会再次报告复用）
        log('放弃外部实例托管声明，重新探测…')
        st.external = false
        st.running = false
        st.port = 0
        st.pid = 0
      }
      stop('重启')
      // 等强杀兜底完成后再拉起
      ctx.timeout(() => { const r = doStart(false); if (!r.ok) log('重启失败：' + r.error, 'error') }, 2600)
      return { ok: true, restarting: true }
    }

    function startPoll() {
      if (st.pollTimer) return
      st.pollTimer = ctx.interval(pollOnce, POLL_MS)
      ctx.timeout(() => { try { pollOnce() } catch (e) { /* ignore */ } }, 300) // 首轮尽快拿 hello
    }

    function stopPolling() {
      if (st.pollTimer) {
        try { st.pollTimer() } catch (e) { /* ignore */ }
        st.pollTimer = null
      }
    }

    function pollOnce() {
      if (!st.procId) return
      let r
      try {
        r = ctx.process.writeStdin({ id: st.procId, chars: '', yieldMs: 1 })
      } catch (e) {
        handleExit('进程句柄失效：' + String(e && e.message || e))
        return
      }
      if (r && r.output) feedLines(String(r.output))
      if (r && !r.running) handleExit(String(r.exitErr || ''))
    }

    function feedLines(chunk) {
      st.lineBuf += chunk
      let idx
      while ((idx = st.lineBuf.indexOf('\n')) >= 0) {
        const line = st.lineBuf.slice(0, idx).trim()
        st.lineBuf = st.lineBuf.slice(idx + 1)
        if (line) handleEventLine(line)
      }
      if (st.lineBuf.length > 65536) st.lineBuf = st.lineBuf.slice(-4096) // 半行失控保护
    }

    function handleEventLine(line) {
      let evt
      try {
        evt = JSON.parse(line)
      } catch (e) {
        log('（桥输出非 JSON，已忽略）' + line.slice(0, 200), 'warn')
        return
      }
      switch (evt.evt) {
        case 'hello':
          st.port = Number(evt.port) || 0
          st.pid = Number(evt.pid) || 0
          st.reused = !!evt.reused
          st.lastError = ''
          log('桥已就绪（pid=' + st.pid + ' port=' + st.port + (st.reused ? '，复用既有实例' : '') + '）')
          armStableReset()
          break
        case 'status':
          st.accounts = Array.isArray(evt.accounts) ? evt.accounts : []
          break
        case 'account':
          upsertAccount(evt)
          break
        case 'log':
          log('[bridge] ' + String(evt.msg || ''), evt.level === 'warn' || evt.level === 'error' ? evt.level : 'info')
          break
        case 'error':
          st.lastError = String(evt.msg || '')
          log('[bridge·错误] ' + st.lastError, 'error')
          break
        default:
          log('（未知事件 ' + String(evt.evt) + '）' + line.slice(0, 160), 'warn')
      }
      emitState()
    }

    function upsertAccount(evt) {
      const id = String(evt.account || '')
      if (!id) return
      const list = st.accounts.slice()
      const i = list.findIndex((a) => a && a.id === id)
      const item = Object.assign({}, i >= 0 ? list[i] : { id: id }, {
        id: id,
        phase: evt.phase || (i >= 0 ? list[i].phase : ''),
      })
      if (i >= 0) list[i] = item; else list.push(item)
      st.accounts = list
    }

    function armStableReset() {
      if (st.stableTimer) { try { st.stableTimer() } catch (e) { /* ignore */ } }
      st.stableTimer = ctx.timeout(() => { st.restarts = 0 }, STABLE_RESET_MS)
    }

    function handleExit(exitErr) {
      if (st.reused) {
        // 复用外部实例：本壳拉起的是「探活进程」，它报告复用后正常退出；
        // 桥本体在外部运行 —— 保持「运行中」语义，但不启动自动重启（防死循环）。
        st.procId = 0
        st.external = true
        st.running = true
        st.want = false
        stopPolling()
        log('桥由外部实例提供（pid=' + st.pid + ' · 端口 ' + st.port + '），本插件不托管该进程')
        emitState()
        return
      }
      const wasWant = st.want
      st.running = false
      st.port = 0
      st.pid = 0
      st.procId = 0
      stopPolling()
      log('桥进程已退出' + (exitErr ? '：' + exitErr : ''), 'warn')
      if (wasWant) {
        if (st.restarts >= MAX_RESTARTS) {
          st.lastError = '桥连续 ' + MAX_RESTARTS + ' 次异常退出，已停止自动重启（可在面板手动启动）'
          log(st.lastError, 'error')
          st.want = false
        } else {
          const delay = RESTART_DELAYS[Math.min(st.restarts, RESTART_DELAYS.length - 1)]
          st.restarts += 1
          log('将在 ' + Math.round(delay / 1000) + 's 后自动重启（第 ' + st.restarts + ' 次）', 'warn')
          st.restartTimer = ctx.timeout(() => {
            st.restartTimer = null
            if (st.want) {
              const r = doStart(true)
              if (!r.ok) log('自动重启失败：' + r.error, 'error')
            }
          }, delay)
        }
      }
      emitState()
    }

    // ─────────────────────── 桥 HTTP 客户端 ───────────────────────

    function bridgeBase() {
      return 'http://127.0.0.1:' + st.port
    }

    function bridgeGetText(path) {
      const r = ctx.web.fetch(bridgeBase() + path)
      return r && r.text != null ? String(r.text) : ''
    }

    function bridgeJson(path) {
      return JSON.parse(bridgeGetText(path))
    }

    function proxyGet(path) {
      const r = ctx.web.fetch(bridgeBase() + path)
      return { status: (r && r.status) || 200, headers: { 'Content-Type': 'application/json; charset=utf-8' }, body: String((r && r.text) || '') }
    }

    function proxyPost(path, body) {
      const r = ctx.web.post(bridgeBase() + path,
        { 'Content-Type': 'application/json', 'X-Bridge-Token': st.token },
        JSON.stringify(body || {}))
      return { status: (r && r.status) || 200, headers: { 'Content-Type': 'application/json; charset=utf-8' }, body: String((r && r.text) || '') }
    }

    function jsonResp(obj, status) {
      return { status: status || 200, headers: { 'Content-Type': 'application/json; charset=utf-8' }, body: JSON.stringify(obj) }
    }

    function bridgeDown(sub) {
      return jsonResp({ ok: false, error: '微信桥未运行（' + sub + ' 不可用），请先在插件面板启动' }, 503)
    }

    // ─────────────────────── HTTP 面（浏览器/外部） ───────────────────────

    function handleHttp(req) {
      const method = String((req && req.method) || 'GET').toUpperCase()
      const full = String((req && req.path) || '')
      const sub = full.slice(HTTP_BASE.length) || '/'
      try {
        // 壳自身操作（不依赖桥）
        if (sub === '/start' && method === 'POST') return jsonResp(doStart(false))
        if (sub === '/stop' && method === 'POST') { stop('面板停用'); return jsonResp({ ok: true }) }
        if (sub === '/restart' && method === 'POST') return jsonResp(restart())
        if (sub === '/logs' && method === 'GET') return jsonResp({ ok: true, logs: st.logs.slice(-60), shell: snapshot() })

        // 扫码入口：302 到桥的真实页面（HTML 文本代理 + PNG 二进制受限的务实解；
        // 页面内相对资源在桥域下自洽，二维码实时渲染不落盘）
        if (sub === '/qr' && method === 'GET') {
          if (!st.running || !st.port) {
            return {
              status: 200,
              headers: { 'Content-Type': 'text/html; charset=utf-8' },
              body: '<!doctype html><meta charset="utf-8"><title>微信桥</title>' +
                '<body style="font-family:system-ui;padding:40px;color:#333">' +
                '<h3>微信桥未运行</h3><p>请先在 PairCode 插件面板启动微信桥，再刷新本页。</p></body>',
            }
          }
          return { status: 302, headers: { 'Location': bridgeBase() + '/qr' }, body: '' }
        }

        if (!st.running || !st.port) return bridgeDown(sub)

        // 代理桥端点（JSON 文本）
        if (sub === '/status' && method === 'GET') {
          const out = { ok: true, shell: snapshot(), bridge: null }
          try { out.bridge = bridgeJson('/status') } catch (e) { out.bridgeError = String(e && e.message || e) }
          return jsonResp(out)
        }
        if (sub === '/accounts') {
          if (method === 'GET') return proxyGet('/accounts')
          if (method === 'POST') return proxyPost('/accounts', readBody(req))
          if (method === 'DELETE') {
            // ctx.web 无 DELETE：走桥的 POST 别名 /accounts/remove（body 携带 id）
            return proxyPost('/accounts/remove', { id: queryParam(req, 'id') })
          }
        }
        if (sub === '/contacts' && method === 'GET') return proxyGet('/contacts')
        if (sub === '/send' && method === 'POST') return proxyPost('/send', readBody(req))
        if (sub === '/relogin' && method === 'POST') return proxyPost('/relogin', readBody(req))
        if (sub === '/login-new' && method === 'POST') return proxyPost('/login/new', readBody(req))
        if (sub === '/refresh' && method === 'POST') return proxyPost('/qr/refresh', readBody(req))
        return jsonResp({ ok: false, error: '未知路径: ' + sub }, 404)
      } catch (e) {
        return jsonResp({ ok: false, error: String(e && e.message || e) }, 500)
      }
    }

    function readBody(req) {
      try { return req.json() } catch (e) { return {} }
    }

    function queryParam(req, name) {
      const q = String((req && req.query) || '')
      for (const pair of q.split('&')) {
        const eq = pair.indexOf('=')
        if (eq > 0 && decodeURIComponent(pair.slice(0, eq)) === name) {
          return decodeURIComponent(pair.slice(eq + 1))
        }
      }
      return ''
    }

    let unregisterHttp = null
    try {
      unregisterHttp = ctx.http.register('*', HTTP_BASE + '/*', handleHttp)
    } catch (e) {
      log('HTTP 路由注册失败：' + String(e && e.message || e), 'error')
    }

    // ─────────────────────── client 半对接 ───────────────────────

    ctx.registerClientMethod('getState', () => snapshot())
    ctx.registerClientMethod('reportOrigin', (args) => {
      const o = args && args.origin ? String(args.origin) : ''
      if (o && /^https?:\/\//.test(o)) st.uiOrigin = toSlash(o)
      return { ok: true }
    })
    ctx.registerClientMethod('start', () => doStart(false))
    ctx.registerClientMethod('stop', () => { stop('面板停用'); return { ok: true } })
    ctx.registerClientMethod('restart', () => restart())

    // ─────────────────────── Agent 工具 ───────────────────────

    ctx.tools.register({
      name: 'wechat_send',
      category: 'web',
      description: '主动向微信联系人发送消息。to 用 wechat_contacts 查询；text 纯文本；file 为工作区内文件路径（图片/视频/文档，自动上传发送）。',
      parameters: {
        type: 'object',
        properties: {
          to: { type: 'string', description: '联系人 id（来自 wechat_contacts）' },
          text: { type: 'string', description: '消息内容（纯文本）' },
          file: { type: 'string', description: '工作区内文件路径（可选；图片/视频/pdf 等，经微信 CDN 上传发送）' },
        },
        required: ['to'],
      },
      execute: function (args) {
        if (!st.running || !st.port) return '微信桥未运行：请在插件面板启动微信桥后重试。'
        try {
          const r = ctx.web.post(bridgeBase() + '/send',
            { 'Content-Type': 'application/json', 'X-Bridge-Token': st.token },
            JSON.stringify({ to: String((args && args.to) || ''), text: String((args && args.text) || ''), file: args && args.file ? String(args.file) : '' }))
          const d = JSON.parse(String((r && r.text) || '{}'))
          if (d && d.ok) return '已发送给 ' + String(args.to) + '（' + (d.sentAt || '') + '）'
          return '发送失败：' + ((d && d.error) || ('HTTP ' + ((r && r.status) || '?')))
        } catch (e) {
          return '发送失败：' + String(e && e.message || e)
        }
      },
    })

    ctx.tools.register({
      name: 'wechat_contacts',
      category: 'web',
      description: '列出微信联系人（id / 是否可发送），供 wechat_send 选择收件人。',
      parameters: { type: 'object', properties: {} },
      execute: function () {
        if (!st.running || !st.port) return '微信桥未运行：请在插件面板启动微信桥后重试。'
        try {
          const d = JSON.parse(bridgeGetText('/contacts'))
          const list = (d && d.contacts) || []
          if (!list.length) return '暂无联系人（对方先给桥发过消息后才会出现）。'
          return list.map((c) => '- ' + c.id + (c.hasSession ? '（可发送）' : '')).join('\n')
        } catch (e) {
          return '获取联系人失败：' + String(e && e.message || e)
        }
      },
    })

    // ─────────────────────── 清理（插件卸载路径） ───────────────────────

    ctx.effect(() => {
      st.want = false
      try { stopPolling() } catch (e) { /* ignore */ }
      try { if (st.restartTimer) st.restartTimer() } catch (e) { /* ignore */ }
      try { if (st.stableTimer) st.stableTimer() } catch (e) { /* ignore */ }
      try { if (unregisterHttp) unregisterHttp() } catch (e) { /* ignore */ }
      const id = st.procId
      if (id) {
        log('插件卸载：停止桥进程 id=' + id)
        // 先发 stop（尽力让其优雅退出），随即强杀（卸载路径不等待；桥 stdin-EOF 自杀兜底）
        try { ctx.process.writeStdin({ id: id, chars: '{"cmd":"stop"}\n', yieldMs: 1 }) } catch (e) { /* ignore */ }
        try { ctx.process.kill(id) } catch (e) { /* ignore */ }
      }
    })

    // ─────────────────────── 自动启动 ───────────────────────

    const bootCfg = loadCfg()
    if (bootCfg.enabled && bootCfg.autoStart) {
      const r = doStart(false)
      if (r.ok) log('autoStart：桥已拉起' + (r.already ? '（已在运行）' : ''))
      else log('autoStart 跳过：' + r.error, 'warn')
    } else {
      log('插件已装载（未自动启动：enabled=' + !!bootCfg.enabled + ' autoStart=' + !!bootCfg.autoStart + '）')
    }
  },
}
