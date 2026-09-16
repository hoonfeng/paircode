// ═══════════════════════════════════════════════════════════════
// wechat-bridge — client 半：(ui) => void
//
// 标题栏 titlebar-right 槽位「微信」按钮 + 桥管理浮层面板：
//   · 状态区：桥运行状态（pid/端口/PairCode 地址/重启次数）
//   · 控制区：启动 / 停止 / 重启 / 打开扫码页
//   · 账号区：别名/阶段/会话 ID；重新登录 / 移除；添加账号
//   · 日志区：最近桥日志（滚动）
//   · 实时性：host 半 ctx.emit('ui:wechat-bridge/state') → ui.on 订阅事件驱动刷新
//     （事件经浏览器侧 2s 事件轮询带过来，面板打开时无需额外轮询）
//   · 数据：ui.invoke('getState'/'start'/'stop'/'restart')（host 方法）；
//     账号管理走同源 fetch（壳 HTTP 面 /api/ext/wechat-bridge/*）
// 纯 DOM + 主题 CSS 变量（禁 emoji 图标；SVG 内联）。
// ═══════════════════════════════════════════════════════════════
(ui) => {
  const PLUGIN = 'wechat-bridge'
  const SLOT = 'titlebar-right'
  const BASE = '/api/ext/wechat-bridge'

  // ─── SVG 图标（禁止 emoji）───────────────────────────────
  const SVG = {
    wechat: '<svg viewBox="0 0 16 16" width="13" height="13" fill="currentColor" aria-hidden="true"><path d="M8 1.5C4.1 1.5 1 4.1 1 7.2c0 1.8 1 3.4 2.6 4.4l-.7 2.4 2.6-1.3c.8.2 1.6.3 2.5.3 3.9 0 7-2.6 7-5.8S11.9 1.5 8 1.5zM5.4 8.7c-.6 0-1-.5-1-1s.4-1 1-1 1 .5 1 1-.4 1-1 1zm5.2 0c-.6 0-1-.5-1-1s.4-1 1-1 1 .5 1 1-.4 1-1 1z"/></svg>',
    close: '<svg viewBox="0 0 16 16" width="12" height="12" fill="currentColor" aria-hidden="true"><path d="M4.2 2.8 8 6.6l3.8-3.8 1.4 1.4L9.4 8l3.8 3.8-1.4 1.4L8 9.4l-3.8 3.8-1.4-1.4L6.6 8 2.8 4.2l1.4-1.4z"/></svg>',
    external: '<svg viewBox="0 0 16 16" width="10" height="10" fill="currentColor" aria-hidden="true"><path d="M10 1h5v5h-1.6V3.6L8 9l-1-1 5.4-5.4H10V1zM3.5 2.5h3V4h-3V12.5h8.5v-3H14v3A1.5 1.5 0 0 1 12.5 14h-9A1.5 1.5 0 0 1 2 12.5V4A1.5 1.5 0 0 1 3.5 2.5z"/></svg>',
  }

  // ─── 样式（幂等注入）─────────────────────────────────────
  const STYLE_ID = 'wxbridge-style'
  if (!document.getElementById(STYLE_ID)) {
    const s = document.createElement('style')
    s.id = STYLE_ID
    s.textContent = `
.wxbridge-btn{display:inline-flex;align-items:center;gap:4px;height:22px;padding:0 8px;font-size:11px;color:var(--text-secondary,#c9d1d9);background:var(--bg-tertiary,#21262d);border:1px solid var(--border-color,#30363d);border-radius:4px;cursor:pointer;white-space:nowrap}
.wxbridge-btn:hover{color:var(--text-primary,#e6edf3);background:var(--bg-hover,#2d333b);border-color:var(--accent-color,#4f8cff)}
.wxbridge-btn .wxbridge-dot{width:7px;height:7px;border-radius:50%;background:#6e7681;flex-shrink:0}
.wxbridge-btn .wxbridge-dot.on{background:#3fb950;box-shadow:0 0 5px rgba(63,185,80,.7)}
.wxbridge-btn .wxbridge-dot.err{background:#f85149}
.wxbridge-panel{position:fixed;z-index:9000;top:34px;right:10px;width:min(520px,calc(100vw - 24px));height:min(620px,calc(100vh - 48px));display:flex;flex-direction:column;background:var(--bg-secondary,#1c2128);border:1px solid var(--border-color,#30363d);border-radius:10px;box-shadow:0 10px 34px rgba(0,0,0,.5);overflow:hidden;font-size:12px;color:var(--text-primary,#e6edf3)}
.wxbridge-head{display:flex;align-items:center;gap:7px;padding:8px 12px;background:var(--bg-tertiary,#21262d);border-bottom:1px solid var(--border-color,#30363d);font-size:13px;font-weight:600}
.wxbridge-head .sp{flex:1}
.wxbridge-head button{display:flex;align-items:center;justify-content:center;width:22px;height:22px;border:none;background:none;color:var(--text-muted,#8b949e);cursor:pointer;border-radius:4px;padding:0}
.wxbridge-head button:hover{color:var(--text-primary,#e6edf3);background:var(--bg-hover,#2d333b)}
.wxbridge-body{overflow-y:auto;flex:1;min-height:0;padding:10px 12px}
.wxbridge-sec{margin-bottom:12px}
.wxbridge-sec .ttl{font-size:10px;font-weight:600;color:var(--text-muted,#8b949e);text-transform:uppercase;letter-spacing:.4px;margin-bottom:6px}
.wxbridge-kv{display:flex;align-items:center;gap:6px;padding:3px 0;font-size:11.5px}
.wxbridge-kv .k{color:var(--text-muted,#8b949e);min-width:86px}
.wxbridge-kv .v{color:var(--text-secondary,#c9d1d9);font-family:Consolas,Menlo,monospace;font-size:11px;word-break:break-all}
.wxbridge-ops{display:flex;gap:6px;flex-wrap:wrap}
.wxbridge-btnx{padding:4px 10px;font-size:11px;border-radius:4px;border:1px solid var(--border-color,#30363d);background:var(--bg-secondary,#161b22);color:var(--text-secondary,#c9d1d9);cursor:pointer}
.wxbridge-btnx:hover:not(:disabled){border-color:var(--accent-color,#4f8cff);color:var(--text-primary,#e6edf3)}
.wxbridge-btnx:disabled{opacity:.5;cursor:default}
.wxbridge-btnx.primary{background:rgba(79,140,255,.14);border-color:var(--accent-color,#4f8cff);color:#79b8ff}
.wxbridge-btnx.primary:hover:not(:disabled){background:rgba(79,140,255,.22)}
.wxbridge-btnx.danger{color:#ff7b72;border-color:rgba(255,123,114,.35)}
.wxbridge-btnx.danger:hover:not(:disabled){background:rgba(255,123,114,.1)}
.wxbridge-acct{display:flex;align-items:center;gap:8px;padding:6px 8px;margin-bottom:5px;border:1px solid var(--border-color,#30363d);border-radius:7px;background:var(--bg-tertiary,#21262d)}
.wxbridge-acct .dot{width:8px;height:8px;border-radius:50%;background:#6e7681;flex-shrink:0}
.wxbridge-acct .dot.running{background:#3fb950}
.wxbridge-acct .dot.login{background:#d29922;animation:wxbridge-pulse 1.2s ease-in-out infinite}
.wxbridge-acct .dot.stale{background:#f0883e}
.wxbridge-acct .dot.offline{background:#f85149}
@keyframes wxbridge-pulse{0%,100%{opacity:1}50%{opacity:.3}}
.wxbridge-acct .info{flex:1;min-width:0}
.wxbridge-acct .nm{font-size:12px;font-weight:600}
.wxbridge-acct .sub{font-size:10px;color:var(--text-muted,#8b949e);font-family:Consolas,Menlo,monospace;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.wxbridge-acct .acts{display:flex;gap:4px;flex-shrink:0}
.wxbridge-logs{background:var(--bg-primary,#0d1117);border:1px solid var(--border-color,#30363d);border-radius:7px;padding:8px 10px;max-height:180px;overflow-y:auto;font-family:Consolas,Menlo,monospace;font-size:10.5px;line-height:1.7;color:var(--text-secondary,#c9d1d9)}
.wxbridge-logs .lw{color:#d29922}
.wxbridge-logs .le{color:#ff7b72}
.wxbridge-logs .lt{color:var(--text-muted,#8b949e);margin-right:6px}
.wxbridge-empty{padding:14px 6px;text-align:center;color:var(--text-muted,#8b949e);font-size:11.5px;line-height:1.7}
.wxbridge-hint{padding:6px 8px;margin-top:6px;font-size:10.5px;color:var(--text-muted,#8b949e);border-left:2px solid var(--border-color,#30363d);line-height:1.6}
`
    document.head.appendChild(s)
  }

  // ─── 状态 ────────────────────────────────────────────────
  let panelEl = null
  let btnEl = null
  let st = null        // 最近一次状态快照
  let busy = false

  function esc(s) {
    return String(s == null ? '' : s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
  }

  function fmtTime(ms) {
    try {
      const d = new Date(ms)
      const p = (n) => String(n).padStart(2, '0')
      return p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    } catch (e) { return '' }
  }

  // ─── 数据 ────────────────────────────────────────────────
  async function invoke(method, args) {
    try { return await ui.invoke(PLUGIN, method, args || {}) } catch (e) { return null }
  }

  async function api(method, path, body) {
    const opts = { method: method, cache: 'no-store' }
    if (body !== undefined) {
      opts.headers = { 'Content-Type': 'application/json' }
      opts.body = JSON.stringify(body)
    }
    const res = await fetch(BASE + path, opts)
    let data = null
    try { data = await res.json() } catch (e) { /* 非 JSON */ }
    return { ok: res.ok, status: res.status, data: data }
  }

  async function loadState() {
    const s = await invoke('getState')
    if (s) { st = s; refreshUi() }
    return s
  }

  function isOn() { return !!(st && st.running) }

  // ─── 面板渲染 ────────────────────────────────────────────
  function statusDotClass() {
    if (!st) return ''
    if (st.running) return 'on'
    return st.lastError ? 'err' : ''
  }

  function phaseDot(phase) {
    const p = String(phase || '')
    if (p === 'running') return 'running'
    if (p === 'login') return 'login'
    if (p === 'stale') return 'stale'
    if (p === 'offline') return 'offline'
    return ''
  }

  function renderBody() {
    if (!panelEl) return
    const body = panelEl.querySelector('.wxbridge-body')
    if (!body) return
    const s = st || {}
    const accts = Array.isArray(s.accounts) ? s.accounts : []
    const logs = Array.isArray(s.logs) ? s.logs.slice(-40).reverse() : []

    let html = ''

    // ── 状态 ──
    html += '<div class="wxbridge-sec"><div class="ttl">状态</div>'
    html += '<div class="wxbridge-kv"><span class="k">桥进程</span><span class="v">' +
      (s.running ? ('运行中 · pid=' + esc(s.pid) + ' · 端口 ' + esc(s.port) + (s.reused ? '（复用实例）' : '')) : '已停止') + '</span></div>'
    html += '<div class="wxbridge-kv"><span class="k">PairCode</span><span class="v">' + esc(s.paircodeUrl || '（未启动）') + '</span></div>'
    if (s.restarts) html += '<div class="wxbridge-kv"><span class="k">自动重启</span><span class="v">' + esc(s.restarts) + ' 次</span></div>'
    if (s.lastError) html += '<div class="wxbridge-kv"><span class="k">最近错误</span><span class="v" style="color:#ff7b72">' + esc(s.lastError) + '</span></div>'
    html += '</div>'

    // ── 控制 ──
    html += '<div class="wxbridge-sec"><div class="ttl">控制</div><div class="wxbridge-ops">'
    html += '<button class="wxbridge-btnx primary" data-op="start"' + (isOn() || busy ? ' disabled' : '') + '>启动</button>'
    html += '<button class="wxbridge-btnx" data-op="stop"' + (!isOn() || busy ? ' disabled' : '') + '>停止</button>'
    html += '<button class="wxbridge-btnx" data-op="restart"' + (!isOn() || busy ? ' disabled' : '') + '>重启</button>'
    html += '<button class="wxbridge-btnx" data-op="qr"' + (!isOn() ? ' disabled' : '') + '>' + SVG.external + ' 打开扫码页</button>'
    html += '</div>'
    if (!isOn()) {
      html += '<div class="wxbridge-hint">桥未运行。若从未启用：请到「设置 → 插件 → 微信桥」勾选「启用微信桥」后回来点「启动」。</div>'
    }
    html += '</div>'

    // ── 账号 ──
    html += '<div class="wxbridge-sec"><div class="ttl">账号（' + accts.length + '）</div>'
    if (!accts.length) {
      html += '<div class="wxbridge-empty">' + (isOn() ? '暂无账号——点下方「添加账号」用手机微信扫码绑定。' : '桥未运行。') + '</div>'
    } else {
      for (const a of accts) {
        const id = String((a && a.id) || '')
        const alias = String((a && a.alias) || id)
        const phase = String((a && a.phase) || '')
        const conv = String((a && a.convId) || '')
        html += '<div class="wxbridge-acct">'
        html += '<span class="dot ' + phaseDot(phase) + '"></span>'
        html += '<span class="info"><span class="nm">' + esc(alias) + '</span>' +
          '<div class="sub">' + esc(id) + ' · ' + esc(phase) + (conv ? ' · ' + esc(conv) : '') + '</div></span>'
        html += '<span class="acts">'
        html += '<button class="wxbridge-btnx" data-op="relogin" data-account="' + esc(id) + '"' + (!isOn() || busy ? ' disabled' : '') + '>重新登录</button>'
        html += '<button class="wxbridge-btnx danger" data-op="remove" data-account="' + esc(id) + '"' + (!isOn() || busy ? ' disabled' : '') + '>移除</button>'
        html += '</span></div>'
      }
    }
    html += '<div class="wxbridge-ops" style="margin-top:6px">'
    html += '<button class="wxbridge-btnx" data-op="add"' + (!isOn() || busy ? ' disabled' : '') + '>添加账号</button>'
    html += '</div></div>'

    // ── 日志 ──
    html += '<div class="wxbridge-sec"><div class="ttl">日志（最近 ' + logs.length + ' 条）</div>'
    if (!logs.length) {
      html += '<div class="wxbridge-empty">暂无日志。</div>'
    } else {
      html += '<div class="wxbridge-logs">'
      for (const l of logs) {
        const lv = l && l.level === 'warn' ? 'lw' : (l && l.level === 'error' ? 'le' : '')
        html += '<div class="' + lv + '"><span class="lt">' + esc(fmtTime((l && l.t) || 0)) + '</span>' + esc((l && l.msg) || '') + '</div>'
      }
      html += '</div>'
    }
    html += '</div>'

    body.innerHTML = html
  }

  function refreshUi() {
    if (btnEl) {
      const dot = btnEl.querySelector('.wxbridge-dot')
      if (dot) {
        dot.className = 'wxbridge-dot ' + statusDotClass()
      }
    }
    renderBody()
  }

  // ─── 操作 ────────────────────────────────────────────────
  async function withBusy(fn) {
    if (busy) return
    busy = true
    renderBody()
    try {
      await fn()
    } finally {
      busy = false
      await loadState()
    }
  }

  async function onOp(op, account) {
    if (op === 'start') {
      await withBusy(async () => {
        const r = await invoke('start')
        if (r && r.ok === false) alert('启动失败：' + (r.error || '未知错误'))
      })
    } else if (op === 'stop') {
      await withBusy(async () => { await invoke('stop') })
    } else if (op === 'restart') {
      await withBusy(async () => { await invoke('restart') })
    } else if (op === 'qr') {
      window.open(BASE + '/qr', '_blank')
    } else if (op === 'add') {
      await withBusy(async () => {
        const r = await api('POST', '/login-new', {})
        if (r.ok && r.data && r.data.ok) {
          alert('已创建账号（' + r.data.id + '），请在打开的扫码页完成绑定')
          window.open(BASE + '/qr', '_blank')
        } else {
          alert('添加失败：' + ((r.data && r.data.error) || ('HTTP ' + r.status)))
        }
      })
    } else if (op === 'relogin') {
      if (!confirm('重新登录「' + account + '」？\n旧凭据将被清除，需要用手机微信重新扫码。')) return
      await withBusy(async () => {
        const r = await api('POST', '/relogin', { account: account })
        if (r.ok && r.data && r.data.ok) {
          window.open(BASE + '/qr', '_blank')
        } else {
          alert('发起失败：' + ((r.data && r.data.error) || ('HTTP ' + r.status)))
        }
      })
    } else if (op === 'remove') {
      if (!confirm('移除账号「' + account + '」？\n将停止该账号的桥循环并删除本地凭据（不可恢复）。')) return
      await withBusy(async () => {
        const r = await api('DELETE', '/accounts?id=' + encodeURIComponent(account))
        if (!(r.ok && r.data && r.data.ok)) {
          alert('移除失败：' + ((r.data && r.data.error) || ('HTTP ' + r.status)))
        }
      })
    }
  }

  // ─── 面板 ────────────────────────────────────────────────
  function openPanel() {
    if (panelEl) return
    panelEl = document.createElement('div')
    panelEl.className = 'wxbridge-panel'
    panelEl.innerHTML =
      '<div class="wxbridge-head">' + SVG.wechat + '<span>微信桥</span><span class="sp"></span>' +
      '<button data-op="close" title="关闭">' + SVG.close + '</button></div>' +
      '<div class="wxbridge-body"></div>'
    document.body.appendChild(panelEl)

    panelEl.addEventListener('click', (ev) => {
      const t = ev.target && ev.target.closest ? ev.target.closest('button[data-op]') : null
      if (!t) return
      const op = t.getAttribute('data-op')
      if (op === 'close') { closePanel(); return }
      onOp(op, t.getAttribute('data-account') || '')
    })

    renderBody()
    loadState()
  }

  function closePanel() {
    if (panelEl) { panelEl.remove(); panelEl = null }
  }

  // ─── 事件订阅（host → 浏览器）───────────────────────────
  try {
    ui.on('ui:wechat-bridge/state', (payload) => {
      if (payload && typeof payload === 'object') {
        st = payload
        refreshUi()
      }
    })
  } catch (e) { /* 旧宿主无事件桥时仅靠打开面板时拉取 */ }

  // ─── 注册标题栏按钮 ──────────────────────────────────────
  ui.registerSlot({
    slotId: SLOT,
    title: '微信桥（wechat-bridge）',
    kind: 'list',
    render(el) {
      btnEl = document.createElement('button')
      btnEl.className = 'wxbridge-btn'
      btnEl.innerHTML = SVG.wechat + '<span>微信</span><span class="wxbridge-dot"></span>'
      btnEl.addEventListener('click', () => { panelEl ? closePanel() : openPanel() })
      el.appendChild(btnEl)

      // 初始化：上报浏览器 origin（host 用它校准桥连接的 PairCode 地址）+ 拉状态
      invoke('reportOrigin', { origin: location.origin })
      loadState()

      return () => {
        closePanel()
        if (btnEl) { btnEl.remove(); btnEl = null }
      }
    },
  })
}
