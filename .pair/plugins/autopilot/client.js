// ═══════════════════════════════════════════════════════════════
// autopilot — client 半（UI 面：标题栏状态徽标 + 独立看板视图）
//
// 两个挂载点，共用本实例内的同一份数据源：
//   ① titlebar-right 槽位（list 型）—— 标题栏右侧状态徽标：
//        运行中圆点脉冲 + 「自主模式 · N 轮 · 裁决」；点击 → 打开/激活看板视图 tab。
//   ② registerView('autopilot-board') —— 主内容区独立 tab 的完整看板：
//        与「对话/编辑器/市场/工具集」同级，可单独看、也可与对话并排（壳「并排对话」）。
//
// 数据面（全部来自 host 半，client 半不自造数据）：
//   · GET /api/autopilot/rounds?convId=<当前会话>  —— 历史监督回合（切会话/刷新恢复）
//   · ui.on('ui:autopilot:round', record)          —— 实时回合（host 半每轮 emit）
//   · 当前会话与运行态：window.__state.currentConvId / agentRunningByConv[convId]
//
// ★ 渲染纪律：回合文本来自 LLM（可能含 < > & 等字符）→ 一律 textContent 赋值，
//   绝不用 innerHTML 拼插值；本文件仅样式块经 innerHTML/textContent 注入静态 CSS。
// ★ 会话隔离：record 自带 convId；后台会话的回合不进当前看板（避免串台）。
// ═══════════════════════════════════════════════════════════════
(ui) => {
  const PLUGIN = 'autopilot'
  const SLOT_ID = 'titlebar-right'
  const VIEW_ID = 'autopilot-board'
  const VIEW_TITLE = '自主模式'
  const POLL_MS = 3000          // 会话切换探测 + 运行态刷新（数据靠事件，不靠轮询）

  // ── 数据源：轮次表 + 会话跟随 + 运行态 ─────────────────────────
  const subs = new Set()
  let rounds = []
  let loadedConvId = ''
  let loadSeq = 0

  const coreState = () => (typeof window !== 'undefined' && window.__state) || {}
  const currentConvId = () => String(coreState().currentConvId || '')
  function isRunning() {
    const id = currentConvId()
    const map = coreState().agentRunningByConv
    return !!(id && map && map[id] === true)
  }
  // 通知订阅者：kind='rounds'（数据变化 → 看板重绘）| 'state'（运行态/会话 → 徽标刷新）
  function notify(kind) {
    for (const fn of Array.from(subs)) {
      try { fn(kind) } catch (e) { console.warn('[autopilot] 订阅者异常', e) }
    }
  }
  function subscribe(fn) {
    subs.add(fn)
    return () => { subs.delete(fn) }
  }
  const keyOf = (r) => String((r && r.startedAt) || '') + '#' + String((r && r.round) || '')

  // load 拉取指定会话的回合表（缺省=当前会话）；期间切会话/又发起新拉取 → 丢弃本次结果。
  async function load(convId) {
    const target = (convId === undefined) ? currentConvId() : String(convId || '')
    const seq = ++loadSeq
    if (!target) { rounds = []; loadedConvId = ''; notify('rounds'); return }
    let next = []
    try {
      const res = await ui.http.get('/autopilot/rounds', { convId: target })
      next = (res && Array.isArray(res.rounds)) ? res.rounds : []
    } catch (e) {
      next = []   // 接口不可用 / 无记录：视作空表（不打断 UI）
    }
    if (seq !== loadSeq || currentConvId() !== target) return
    const same = JSON.stringify(next) === JSON.stringify(rounds)
    rounds = next
    loadedConvId = target
    notify(same ? 'state' : 'rounds')
  }

  // append 实时追加/替换一个回合（幂等键 = startedAt#round：同轮次重复事件只更新不重复）
  function append(rec) {
    if (!rec) return
    const cur = currentConvId()
    if (rec.convId && cur && String(rec.convId) !== cur) return
    const k = keyOf(rec)
    const i = rounds.findIndex((r) => keyOf(r) === k)
    rounds = (i >= 0) ? rounds.map((r, j) => (j === i ? rec : r)) : rounds.concat([rec])
    notify('rounds')
  }

  // ── 裁决语义（与 AutopilotPanel.vue 一致：异常优先于收尾，避免掩盖失败）──
  function verdictKey(r) {
    if (!r) return ''
    if (r.error || r.ended === 'error') return 'error'
    if (r.action === 'continue') return 'continue'
    if (r.action === 'done') return 'done'
    return 'error'
  }
  const verdictLabel = (k) => (k === 'done' ? '收尾' : k === 'continue' ? '续跑' : k === 'error' ? '异常' : '')
  function fmtDuration(ms) {
    const n = Number(ms) || 0
    if (n < 1000) return n + 'ms'
    if (n < 60000) return (n / 1000).toFixed(1) + 's'
    return Math.round(n / 60000) + 'm' + Math.round((n % 60000) / 1000) + 's'
  }
  const metaOf = (r) => {
    if (!r) return ''
    const t = (Number(r.promptTokens) || 0) + (Number(r.outputTokens) || 0)
    let s = fmtDuration(r.durationMs)
    if (r.steps || r.toolCalls) s += ' · ' + (r.steps || 0) + ' 步 / ' + (r.toolCalls || 0) + ' 工具'
    if (t) s += ' · ' + t + ' tokens'
    return s
  }
  function badgeTitle(running, last) {
    if (running) return '自主模式运行中（监督者审核中）—— 点击打开看板'
    if (!last) return '自主模式监督看板（本会话暂无监督回合）—— 点击打开'
    return '自主模式：共 ' + rounds.length + ' 轮监督，末次裁决 ' + (verdictLabel(verdictKey(last)) || '—') +
      '（' + metaOf(last) + '）—— 点击打开看板'
  }

  // ── 打开看板视图（壳 ctx.uiLayout / __PAIRCODE_CORE.layout 是权威面）──
  function openBoard() {
    const core = (typeof window !== 'undefined') ? window.__PAIRCODE_CORE : null
    const lay = core && core.layout
    if (lay && typeof lay.openViewTab === 'function') {
      lay.openViewTab(PLUGIN, VIEW_ID, { activate: true })
      return
    }
    // 兜底：旧内核无 view 服务 → 直接持久化打开态并切主视图（尽力而为）
    try { localStorage.setItem('viewOpen:' + PLUGIN + ':' + VIEW_ID, '1') } catch (e) { /* 忽略 */ }
    const st = coreState()
    if (st.panels) st.panels.mainTab = 'view:' + PLUGIN + ':' + VIEW_ID
  }

  // ── 样式（前缀隔离：ap-t* = 标题栏徽标，apb-* = 看板视图；色值全用既有令牌/语义色）──
  const CSS = [
    /* 标题栏徽标 */
    '.ap-tbadge{display:inline-flex;align-items:center;gap:5px;height:20px;padding:0 8px;',
    'border:1px solid transparent;border-radius:10px;background:transparent;color:var(--text-secondary);',
    'font-size:11px;line-height:1;cursor:pointer;font-family:inherit;white-space:nowrap}',
    '.ap-tbadge:hover{background:var(--bg-active);color:var(--text-primary)}',
    '.ap-tbadge .ap-tdot{width:6px;height:6px;border-radius:50%;background:var(--text-muted);flex-shrink:0}',
    '.ap-tbadge.running{border-color:color-mix(in srgb, currentColor 35%, transparent);color:var(--text-primary)}',
    '.ap-tbadge.running .ap-tdot{background:#d4a74e;animation:ap-pulse 1.4s ease-in-out infinite}',
    '@keyframes ap-pulse{0%,100%{opacity:1}50%{opacity:.3}}',
    '.ap-tbadge .ap-tcount{font-variant-numeric:tabular-nums;color:var(--text-muted)}',
    '.ap-tverdict{font-weight:600}',
    '.ap-tverdict.done{color:#6a9955}',
    '.ap-tverdict.continue{color:#d4a74e}',
    '.ap-tverdict.error{color:var(--danger,#e5534b)}',
    /* 看板视图 */
    '.apb-root{display:flex;flex-direction:column;height:100%;min-height:0;background:var(--bg-primary)}',
    '.apb-head{display:flex;align-items:center;gap:8px;flex-shrink:0;padding:8px 12px;',
    'border-bottom:1px solid var(--border-color);background:var(--bg-secondary)}',
    '.apb-title{font-size:13px;font-weight:600;color:var(--text-primary)}',
    '.apb-sub{font-size:11px;color:var(--text-muted);font-variant-numeric:tabular-nums}',
    '.apb-spacer{flex:1}',
    '.apb-btn{font:inherit;font-size:11px;color:var(--text-secondary);background:transparent;',
    'border:1px solid var(--border-color);border-radius:3px;padding:2px 8px;cursor:pointer}',
    '.apb-btn:hover{background:var(--bg-active);color:var(--text-primary)}',
    '.apb-body{flex:1;min-height:0;overflow-y:auto;padding:10px 12px}',
    '.apb-empty{max-width:520px;margin:28px auto;font-size:12px;line-height:1.7;color:var(--text-muted);text-align:center}',
    '.apb-round{border:1px solid var(--border-color);border-radius:var(--border-radius);',
    'background:var(--bg-secondary);padding:8px 10px;font-size:12px}',
    '.apb-round+.apb-round{margin-top:8px}',
    '.apb-round-head{display:flex;align-items:center;gap:6px;flex-wrap:wrap}',
    '.apb-no{color:var(--text-muted);font-variant-numeric:tabular-nums}',
    '.apb-badge{flex-shrink:0;font-size:11px;line-height:1.5;padding:0 6px;border-radius:8px;border:1px solid currentColor}',
    '.apb-badge.done{color:#6a9955}',
    '.apb-badge.continue{color:#d4a74e}',
    '.apb-badge.error{color:var(--danger,#e5534b)}',
    '.apb-meta{color:var(--text-muted);font-size:11px}',
    '.apb-text{margin-top:4px;color:var(--text-primary);line-height:1.5;word-break:break-word;white-space:pre-wrap}',
    '.apb-next{margin-top:4px;display:flex;gap:6px;align-items:flex-start;background:var(--bg-active);border-radius:3px;padding:3px 6px}',
    '.apb-next-label{flex-shrink:0;font-size:11px;color:#d4a74e}',
    '.apb-next-text{color:var(--text-secondary);line-height:1.5;word-break:break-word;white-space:pre-wrap}',
    '.apb-more{margin-top:4px}',
    '.apb-more>summary{cursor:pointer;font-size:11px;color:var(--text-muted);user-select:none}',
    '.apb-more>summary:hover{color:var(--text-secondary)}',
    '.apb-pre{margin:4px 0 0;padding:4px 6px;background:var(--bg-primary);border-radius:3px;font-size:11px;',
    'color:var(--text-secondary);white-space:pre-wrap;word-break:break-word;max-height:200px;overflow-y:auto}',
    '.apb-trace{display:flex;gap:6px;padding:2px 0;font-size:11px}',
    '.apb-trace-tool{flex-shrink:0;color:var(--accent);font-family:var(--font-mono,monospace)}',
    '.apb-trace-text{color:var(--text-muted);flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}',
  ].join('\n')

  const el = (tag, cls, text) => {
    const n = document.createElement(tag)
    if (cls) n.className = cls
    if (text !== undefined && text !== null && text !== '') n.textContent = String(text)
    return n
  }

  // ── 挂载点①：标题栏状态徽标（titlebar-right / list 型）─────────
  ui.registerSlot({
    slotId: SLOT_ID,
    kind: 'list',
    title: '自主模式状态徽标（点击打开监督看板）',
    render(host) {
      host.textContent = ''
      const style = document.createElement('style')
      style.textContent = CSS
      host.appendChild(style)

      const btn = el('button', 'ap-tbadge')
      btn.type = 'button'
      btn.setAttribute('aria-label', '自主模式监督看板')
      const dot = el('span', 'ap-tdot')
      const label = el('span', null, '自主模式')
      const count = el('span', 'ap-tcount')
      const verdict = el('span', 'ap-tverdict')
      btn.appendChild(dot)
      btn.appendChild(label)
      btn.appendChild(count)
      btn.appendChild(verdict)
      btn.addEventListener('click', openBoard)
      host.appendChild(btn)

      const paint = () => {
        const running = isRunning()
        const last = rounds.length ? rounds[rounds.length - 1] : null
        const k = verdictKey(last)
        const vLabel = verdictLabel(k)
        btn.classList.toggle('running', running)
        count.textContent = rounds.length ? rounds.length + ' 轮' : ''
        count.hidden = rounds.length === 0
        verdict.textContent = vLabel
        verdict.className = 'ap-tverdict' + (k ? ' ' + k : '')
        verdict.hidden = !vLabel
        btn.title = badgeTitle(running, last)
      }

      const off = subscribe(() => paint())
      paint()
      return () => {
        off()
        btn.removeEventListener('click', openBoard)
      }
    },
  })

  // ── 挂载点②：独立看板视图（registerView → 主内容区 tab）───────
  ui.registerView({
    id: VIEW_ID,
    title: VIEW_TITLE,
    icon: 'eye',
    order: 90,          // 排在壳内置视图之后（内置 < 100）
    open: false,        // 不默认打开（由徽标/视图菜单打开）
    render(host) {
      host.textContent = ''
      const style = document.createElement('style')
      style.textContent = CSS
      host.appendChild(style)

      const root = el('div', 'apb-root')
      const head = el('div', 'apb-head')
      const title = el('span', 'apb-title', '自主模式 · 监督者')
      const sub = el('span', 'apb-sub')
      const spacer = el('div', 'apb-spacer')
      const refresh = el('button', 'apb-btn', '刷新')
      refresh.type = 'button'
      refresh.title = '重新拉取本会话的监督回合（GET /api/autopilot/rounds）'
      refresh.addEventListener('click', () => { load() })
      head.appendChild(title)
      head.appendChild(sub)
      head.appendChild(spacer)
      head.appendChild(refresh)

      const body = el('div', 'apb-body')
      root.appendChild(head)
      root.appendChild(body)
      host.appendChild(root)

      const empty = () => {
        body.textContent = ''
        const box = el('div', 'apb-empty')
        box.appendChild(el('div', null, '本会话暂无监督回合'))
        const hint = el('div', null, '自主模式下，工作 agent 每次自然结束时监督者会审核、评判并决定下一步；' +
          '每轮记录都会落盘（.pair/autopilot/<会话 id>.jsonl）并显示在这里。')
        box.appendChild(hint)
        body.appendChild(box)
      }

      // buildRound 渲染单轮卡片（结构对齐 RightPanel 内嵌面板 AutopilotPanel.vue）
      const buildRound = (r, idx) => {
        const card = el('div', 'apb-round')
        const h = el('div', 'apb-round-head')
        const no = el('span', 'apb-no', '#' + (idx + 1))
        no.title = '本次运行的第 ' + (r.round || 1) + ' 次监督（round 为「运行内序号」，# 为会话内累计序号）'
        const k = verdictKey(r)
        h.appendChild(no)
        h.appendChild(el('span', 'apb-badge ' + k, verdictLabel(k)))
        h.appendChild(el('span', 'apb-meta', metaOf(r)))
        if (r.error) {
          const err = el('span', 'apb-badge error', '异常')
          err.title = String(r.error)
          h.appendChild(err)
        }
        card.appendChild(h)

        if (r.assessment) card.appendChild(el('div', 'apb-text', r.assessment))
        if (r.nextTask) {
          const next = el('div', 'apb-next')
          next.appendChild(el('span', 'apb-next-label', '下一步'))
          next.appendChild(el('span', 'apb-next-text', r.nextTask))
          card.appendChild(next)
        }
        if (r.objective) {
          const d = document.createElement('details')
          d.className = 'apb-more'
          d.appendChild(el('summary', null, '用户目标'))
          d.appendChild(el('pre', 'apb-pre', r.objective))
          card.appendChild(d)
        }
        if (r.evidence) {
          const d = document.createElement('details')
          d.className = 'apb-more'
          d.appendChild(el('summary', null, '证据'))
          d.appendChild(el('pre', 'apb-pre', r.evidence))
          card.appendChild(d)
        }
        if (r.trace && r.trace.length) {
          const d = document.createElement('details')
          d.className = 'apb-more'
          d.appendChild(el('summary', null, '监督者轨迹（' + r.trace.length + '）'))
          for (const t of r.trace) {
            const row = el('div', 'apb-trace')
            if (t.type === 'tool_call') {
              row.appendChild(el('span', 'apb-trace-tool', t.name))
              row.appendChild(el('span', 'apb-trace-text', t.result || t.args))
            } else {
              row.appendChild(el('span', 'apb-trace-text', t.content))
            }
            d.appendChild(row)
          }
          card.appendChild(d)
        }
        if (r.workerReport) {
          const d = document.createElement('details')
          d.className = 'apb-more'
          d.appendChild(el('summary', null, '工作 agent 侧（' + (r.workerTurns || 0) + ' 轮）'))
          d.appendChild(el('pre', 'apb-pre', r.workerReport))
          card.appendChild(d)
        }
        return card
      }

      // drawHead 只更新头部副标题（会话/运行态）——不重建列表，避免每 3s 的 state
      // 通知把用户已展开的 <details>（证据/轨迹）折叠回去。
      const drawHead = () => {
        const conv = currentConvId()
        sub.textContent = rounds.length
          ? rounds.length + ' 轮监督' + (conv ? ' · ' + conv : '') + (isRunning() ? ' · 运行中' : '')
          : (conv ? '会话 ' + conv : '')
      }
      // drawBody 重建轮次列表；原本贴近底部（或首屏）时保持「跟随最新」。
      const drawBody = () => {
        if (!rounds.length) { empty(); return }
        const stick = body.scrollHeight - body.scrollTop - body.clientHeight < 24
        body.textContent = ''
        rounds.forEach((r, i) => { body.appendChild(buildRound(r, i)) })
        if (stick) body.scrollTop = body.scrollHeight
      }
      const draw = () => { drawHead(); drawBody() }

      const offRounds = subscribe((kind) => { if (kind === 'rounds') draw() })
      const offState = subscribe((kind) => { if (kind === 'state') drawHead() })
      if (currentConvId() && loadedConvId !== currentConvId()) load()
      draw()
      return () => { offRounds(); offState() }
    },
  })

  // ── 启动：首拉 + 实时事件 + 会话切换探测 ──────────────────────
  ui.on('ui:autopilot:round', (rec) => append(rec))
  load()
  setInterval(() => {
    const id = currentConvId()
    if (id !== loadedConvId) load(id)
    else notify('state')
  }, POLL_MS)
}
