// ═══════════════════════════════════════════════════════════════
// agent-teams — client 半：(ui) => void
//
// 标题栏 titlebar-right 槽位「团队」按钮 + 活动浮层：
//   · 点击展开团队活动面板（成员卡片/任务 DAG/进度/未读）
//   · 每 3s 轮询 GET /api/agent-teams/teams 获取磁盘真相快照
//   · staged 计划：批准并运行 / 返回对话修改 / 废弃（ui.invoke 调宿主）
//   · 正在运行的团队：暂停（halt）/ 展开成员列 / 任务状态徽章
// 纯 DOM 实现（不依赖 Vue bundle），CSS 变量跟随 IDE 主题。
// ═══════════════════════════════════════════════════════════════
(ui) => {
  const PLUGIN = 'agent-teams'
  const SLOT = 'titlebar-right'
  const API = '/api/agent-teams/teams'

  // ─── SVG 图标（禁止 emoji）───────────────────────────────
  const SVG = {
    team: '<svg viewBox="0 0 16 16" width="13" height="13" fill="currentColor" aria-hidden="true"><path d="M5.5 4a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5zm5 4a2 2 0 1 1 0 4 2 2 0 0 1 0-4zm-7.7 8a2.7 2.7 0 0 1 5.4 0H2.8zm10.4 0a2.9 2.9 0 0 0-2.2-2.8 6 6 0 0 1 1.7 2.8h.5z"/></svg>',
    close: '<svg viewBox="0 0 16 16" width="12" height="12" fill="currentColor" aria-hidden="true"><path d="M4.2 2.8 8 6.6l3.8-3.8 1.4 1.4L9.4 8l3.8 3.8-1.4 1.4L8 9.4l-3.8 3.8-1.4-1.4L6.6 8 2.8 4.2l1.4-1.4z"/></svg>',
    caret: '<svg viewBox="0 0 16 16" width="8" height="8" fill="currentColor" aria-hidden="true"><path d="M4 6l4 4 4-4H4z"/></svg>',
    check: '<svg viewBox="0 0 16 16" width="10" height="10" fill="currentColor" aria-hidden="true"><path d="M6.5 11.2 3.3 8l-1.1 1.1 4.3 4.3 7.3-7.3-1.1-1.1z"/></svg>',
    xmark: '<svg viewBox="0 0 16 16" width="10" height="10" fill="currentColor" aria-hidden="true"><path d="M4.2 2.8 8 6.6l3.8-3.8 1.4 1.4L9.4 8l3.8 3.8-1.4 1.4L8 9.4l-3.8 3.8-1.4-1.4L6.6 8 2.8 4.2l1.4-1.4z"/></svg>',
    play: '<svg viewBox="0 0 16 16" width="10" height="10" fill="currentColor" aria-hidden="true"><path d="M4 2l10 6-10 6V2z"/></svg>',
    stop: '<svg viewBox="0 0 16 16" width="10" height="10" fill="currentColor" aria-hidden="true"><path d="M3 3h10v10H3V3z"/></svg>',
    inbox: '<svg viewBox="0 0 16 16" width="11" height="11" fill="currentColor" aria-hidden="true"><path d="M2 3h12v10H2V3zm1.5 1.5v7h2.3a2.4 2.4 0 0 0 4.4 0h2.3v-7H3.5z"/></svg>',
    spin: '<svg viewBox="0 0 16 16" width="10" height="10" fill="currentColor" aria-hidden="true"><path d="M8 1a7 7 0 1 0 7 7h-1.6A5.4 5.4 0 1 1 8 2.6V1z"/></svg>',
  }

  // 全局样式（幂等）
  const STYLE_ID = 'agteams-style'
  if (!document.getElementById(STYLE_ID)) {
    const st = document.createElement('style')
    st.id = STYLE_ID
    st.textContent = `
.agteams-btn{display:inline-flex;align-items:center;gap:4px;height:22px;padding:0 8px;font-size:11px;color:var(--text-secondary,#c9d1d9);background:var(--bg-tertiary,#21262d);border:1px solid var(--border-color,#30363d);border-radius:4px;cursor:pointer;white-space:nowrap}
.agteams-btn:hover{color:var(--text-primary,#e6edf3);background:var(--bg-hover,#2d333b);border-color:var(--accent-color,#4f8cff)}
.agteams-btn .agteams-badge{display:inline-flex;align-items:center;justify-content:center;min-width:14px;height:14px;padding:0 3px;font-size:9px;font-weight:600;color:#fff;background:var(--accent-color,#4f8cff);border-radius:7px}
.agteams-btn .agteams-badge.hot{background:#f0883e}
.agteams-panel{position:fixed;z-index:9000;top:34px;right:10px;width:min(560px,calc(100vw - 24px));max-height:calc(100vh - 48px);display:flex;flex-direction:column;background:var(--bg-secondary,#1c2128);border:1px solid var(--border-color,#30363d);border-radius:10px;box-shadow:0 10px 34px rgba(0,0,0,.5);overflow:hidden;font-size:12px;color:var(--text-primary,#e6edf3)}
.agteams-panel-head{display:flex;align-items:center;justify-content:space-between;padding:8px 12px;background:var(--bg-tertiary,#21262d);border-bottom:1px solid var(--border-color,#30363d);font-size:13px;font-weight:600}
.agteams-panel-head .agteams-ico{display:flex;align-items:center;justify-content:center;width:22px;height:22px;border:none;background:none;color:var(--text-muted,#8b949e);cursor:pointer;border-radius:4px;padding:0}
.agteams-panel-head .agteams-ico:hover{color:var(--text-primary,#e6edf3);background:var(--bg-hover,#2d333b)}
.agteams-body{overflow-y:auto;flex:1;padding:10px 12px}
.agteams-empty{padding:22px 8px;text-align:center;color:var(--text-muted,#8b949e);font-size:12px;line-height:1.7}
.agteams-empty .t{color:var(--text-secondary,#c9d1d9);font-weight:600}
.agteams-card{margin-bottom:10px;border:1px solid var(--border-color,#30363d);border-radius:8px;background:var(--bg-tertiary,#21262d);overflow:hidden}
.agteams-card-head{display:flex;align-items:center;gap:8px;padding:7px 10px;border-bottom:1px solid var(--border-color,#30363d)}
.agteams-card-name{flex:1;min-width:0;font-weight:600;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.agteams-phase{display:inline-flex;align-items:center;gap:3px;padding:1px 7px;font-size:10px;border-radius:8px;border:1px solid var(--border-color,#30363d);color:var(--text-secondary,#c9d1d9)}
.agteams-phase.running{color:#7ee787;border-color:rgba(126,231,135,.35)}
.agteams-phase.staged{color:#f0883e;border-color:rgba(240,136,62,.35)}
.agteams-phase.halted{color:#ff7b72;border-color:rgba(255,123,114,.35)}
.agteams-card-desc{padding:6px 10px;font-size:11px;color:var(--text-muted,#8b949e);line-height:1.5;border-bottom:1px solid var(--border-color,#30363d);max-height:44px;overflow:hidden}
.agteams-progress{height:5px;background:var(--bg-secondary,#161b22);border-radius:3px;margin:8px 10px 4px;overflow:hidden}
.agteams-progress i{display:block;height:100%;background:var(--accent-color,#4f8cff);border-radius:3px;transition:width .4s}
.agteams-stats{padding:0 10px 8px;font-size:10px;color:var(--text-muted,#8b949e);display:flex;gap:12px}
.agteams-sec{padding:6px 10px;font-size:10px;font-weight:600;color:var(--text-muted,#8b949e);border-top:1px solid var(--border-color,#30363d);display:flex;align-items:center;justify-content:space-between}
.agteams-sec .hint{font-weight:400;font-size:9px}
.agteams-member{display:flex;align-items:center;gap:7px;padding:5px 10px;border-left:2px solid transparent}
.agteams-member:hover{background:var(--bg-hover,#2d333b)}
.agteams-member.working{border-left-color:var(--accent-color,#4f8cff)}
.agteams-member .av{display:inline-flex;align-items:center;justify-content:center;width:18px;height:18px;border-radius:50%;background:var(--bg-secondary,#161b22);border:1px solid var(--border-color,#30363d);color:var(--text-secondary,#c9d1d9);font-size:9px;font-weight:700;flex-shrink:0}
.agteams-member .nm{flex:1;min-width:0;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;color:var(--text-primary,#e6edf3)}
.agteams-member .rl{font-size:10px;color:var(--text-muted,#8b949e);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:110px}
.agteams-member .st{font-size:10px;color:var(--text-muted,#8b949e);flex-shrink:0}
.agteams-member .st .dot{display:inline-block;width:7px;height:7px;border-radius:50%;margin-right:4px;background:#6e7681}
.agteams-member.working .st .dot{background:#3fb950;animation:agteams-pulse 1.2s ease-in-out infinite}
.agteams-member .pg{width:56px;height:4px;background:var(--bg-secondary,#161b22);border-radius:2px;overflow:hidden;flex-shrink:0}
.agteams-member .pg i{display:block;height:100%;background:var(--accent-color,#4f8cff)}
.agteams-task{display:flex;align-items:center;gap:6px;padding:4px 10px;font-size:11px}
.agteams-task:hover{background:var(--bg-hover,#2d333b)}
.agteams-task .dot{width:7px;height:7px;border-radius:50%;flex-shrink:0}
.agteams-task.pending .dot{background:#6e7681}
.agteams-task.claimed .dot{background:#f0883e}
.agteams-task.in_progress .dot{background:#3fb950;animation:agteams-pulse 1.2s ease-in-out infinite}
.agteams-task.completed .dot{background:#58a6ff}
.agteams-task.failed .dot{background:#f85149}
.agteams-task.cancelled .dot{background:#484f58}
.agteams-task.blocked{opacity:.55}
.agteams-task .tid{font-family:Consolas,Menlo,monospace;font-size:10px;color:var(--text-muted,#8b949e);flex-shrink:0}
.agteams-task .sub{flex:1;min-width:0;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.agteams-task .as{font-size:10px;color:var(--text-muted,#8b949e);flex-shrink:0}
.agteams-mail{padding:4px 10px;font-size:10px;color:var(--text-muted,#8b949e)}
.agteams-actions{display:flex;gap:6px;padding:8px 10px;border-top:1px solid var(--border-color,#30363d)}
.agteams-btnx{padding:4px 10px;font-size:11px;border-radius:4px;border:1px solid var(--border-color,#30363d);background:var(--bg-secondary,#161b22);color:var(--text-secondary,#c9d1d9);cursor:pointer}
.agteams-btnx:hover{border-color:var(--accent-color,#4f8cff);color:var(--text-primary,#e6edf3)}
.agteams-btnx.primary{background:rgba(79,140,255,.14);border-color:var(--accent-color,#4f8cff);color:#79b8ff}
.agteams-btnx.primary:hover{background:rgba(79,140,255,.22)}
.agteams-btnx.danger{color:#ff7b72;border-color:rgba(255,123,114,.35)}
.agteams-btnx.danger:hover{background:rgba(255,123,114,.1)}
.agteams-btnx:disabled{opacity:.5;cursor:default}
.agteams-foot{padding:5px 10px;font-size:9px;color:var(--text-muted,#8b949e);border-top:1px solid var(--border-color,#30363d);display:flex;justify-content:space-between;align-items:center}
.agteams-spin{animation:agteams-rot 1s linear infinite;display:inline-block}
@keyframes agteams-rot{to{transform:rotate(360deg)}}
@keyframes agteams-pulse{0%,100%{opacity:1}50%{opacity:.35}}
`
    document.head.appendChild(st)
  }

  // ─── 状态 ────────────────────────────────────────────────
  let panelEl = null
  let timerId = null
  let busy = false

  // 取快照（fetch 失败返回 null）
  async function fetchTeams() {
    try {
      const res = await fetch(API, { cache: 'no-store' })
      if (!res.ok) return null
      const data = await res.json()
      return Array.isArray(data.teams) ? data.teams : null
    } catch (e) {
      return null
    }
  }

  function esc(s) {
    return String(s == null ? '' : s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
  }

  function render(teams) {
    if (!panelEl) return
    const body = panelEl.querySelector('.agteams-body')
    if (!teams || teams.length === 0) {
      body.innerHTML = '<div class="agteams-empty"><div class="t">当前没有进行中的多智能体团队</div><div>在对话里说「用 AgentTeams 做 X」即可让当前会话成为队长</div></div>'
      return
    }
    body.innerHTML = teams.map((team, idx) => {
      const members = (team.members || []).filter((m) => m.status !== 'removed')
      const total = members.length
      const done = members.filter((m) => m.done > 0 && m.done === m.total && m.total > 0).length
      const pct = total === 0 ? 0 : Math.round((done / total) * 100)
      const tasks = (team.tasks || [])
      const doneTasks = tasks.filter((t) => t.status === 'completed').length
      const phaseCls = team.halted ? 'halted' : (team.phase === 'staged' ? 'staged' : 'running')
      const phaseTxt = team.halted ? '已暂停' : (team.phase === 'staged' ? '待批准' : '运行中')
      const memberHtml = members.map((m) => {
        const initial = (m.name || '?').slice(0, 1).toUpperCase()
        const activity = m.activity === 'working' ? 'working' : ''
        const st = m.activity === 'working' ? '工作中' : (m.activity === 'idle' ? '空闲' : '—')
        const pg = m.total === 0 ? 0 : Math.round((m.done / m.total) * 100)
        const unread = m.unread > 0 ? '<span style="color:#f0883e">' + m.unread + ' 未读</span>' : ''
        return '<div class="agteams-member ' + activity + '">' +
          '<span class="av">' + esc(initial) + '</span>' +
          '<span class="nm">' + esc(m.name) + '</span>' +
          (m.role ? '<span class="rl">' + esc(m.role) + '</span>' : '<span class="rl"></span>') +
          '<span class="pg"><i style="width:' + pg + '%"></i></span>' +
          '<span class="st"><span class="dot"></span>' + st + (m.currentTask ? ' · ' + esc(m.currentTask) : '') + ' ' + unread + '</span>' +
        '</div>'
      }).join('')
      const taskHtml = tasks.map((t) => {
        const depBlocked = t.state === 'blocked'
        return '<div class="agteams-task ' + esc(t.status) + (depBlocked ? ' blocked' : '') + '">' +
          '<span class="dot"></span>' +
          '<span class="tid">' + esc(t.id) + '</span>' +
          '<span class="sub">' + esc(t.subject) + (t.kind && t.kind !== 'work' ? ' [' + esc(t.kind) + (t.round ? ' r' + t.round : '') + ']' : '') + '</span>' +
          '<span class="as">' + (t.assignee ? '@' + esc(t.assignee) : '') + '</span>' +
        '</div>'
      }).join('')
      const mailPreview = (team.captainInbox || []).slice(-2).map((m) => {
        return '<div class="agteams-mail">' + esc(m.from) + ' → 队长: ' + esc(m.content.length > 60 ? m.content.slice(0, 57) + '…' : m.content) + '</div>'
      }).join('')
      let actions = ''
      if (team.phase === 'staged' && !team.halted) {
        actions = '<div class="agteams-actions">' +
          '<button class="agteams-btnx primary" data-act="approve" data-team="' + esc(team.teamId) + '">' + SVG.play + ' 批准并运行</button>' +
          '<button class="agteams-btnx" data-act="continue" data-team="' + esc(team.teamId) + '">返回对话修改</button>' +
          '<button class="agteams-btnx danger" data-act="discard" data-team="' + esc(team.teamId) + '">废弃</button>' +
        '</div>'
      } else if (!team.halted) {
        actions = '<div class="agteams-actions">' +
          '<button class="agteams-btnx danger" data-act="halt" data-team="' + esc(team.teamId) + '">' + SVG.stop + ' 暂停团队</button>' +
        '</div>'
      } else {
        actions = '<div class="agteams-actions"><span style="font-size:10px;color:var(--text-muted,#8b949e)">团队已暂停（队长可用 agent_teams_resume 恢复）</span></div>'
      }
      return '<div class="agteams-card">' +
        '<div class="agteams-card-head">' +
          '<span class="agteams-card-name">' + esc(team.name) + '</span>' +
          '<span class="agteams-phase ' + phaseCls + '">' + phaseTxt + '</span>' +
        '</div>' +
        (team.description ? '<div class="agteams-card-desc">' + esc(team.description) + '</div>' : '') +
        '<div class="agteams-progress"><i style="width:' + pct + '%"></i></div>' +
        '<div class="agteams-stats">成员 ' + total + ' · 任务 ' + doneTasks + '/' + tasks.length + ' 完成' + (team.escalated ? ' · <span style="color:#f0883e">已升级</span>' : '') + '</div>' +
        (members.length ? '<div class="agteams-sec">成员</div>' + memberHtml : '') +
        (tasks.length ? '<div class="agteams-sec">任务 <span class="hint">依赖未完成的显示为弱化</span></div>' + taskHtml : '') +
        mailPreview +
        actions +
      '</div>'
    }).join('')
  }

  async function refresh() {
    if (busy) return
    busy = true
    try {
      const teams = await fetchTeams()
      render(teams)
      updateBadge(teams)
    } finally {
      busy = false
    }
  }

  function updateBadge(teams) {
    // 找到当前工作区的团队数（面板按钮徽标）
    const active = (teams || []).filter((t) => !t.halted).length
    const staged = (teams || []).filter((t) => t.phase === 'staged' && !t.halted).length
    const badge = document.getElementById('agteams-badge')
    if (badge) {
      if (active === 0) { badge.style.display = 'none' }
      else {
        badge.style.display = 'inline-flex'
        badge.textContent = active
        badge.className = 'agteams-badge' + (staged > 0 ? ' hot' : '')
      }
    }
  }

  function openPanel() {
    if (panelEl) { closePanel(); return }
    panelEl = document.createElement('div')
    panelEl.className = 'agteams-panel'
    panelEl.innerHTML =
      '<div class="agteams-panel-head"><span>多智能体团队活动</span>' +
      '<button class="agteams-ico" data-act="close">' + SVG.close + '</button></div>' +
      '<div class="agteams-body"></div>' +
      '<div class="agteams-foot"><span>磁盘真相（' + API + '）· 3s 轮询</span><span>agent-teams</span></div>'
    document.body.appendChild(panelEl)

    // 事件代理
    panelEl.addEventListener('click', async (e) => {
      const t = e.target.closest('[data-act]')
      if (!t) return
      const act = t.getAttribute('data-act')
      if (act === 'close') { closePanel(); return }
      const teamId = t.getAttribute('data-team') || ''
      t.disabled = true
      try {
        if (act === 'approve') {
          await ui.invoke(PLUGIN, 'approve', { teamId })
        } else if (act === 'discard') {
          if (!confirm('废弃该暂存计划？（不可恢复，将归档）')) { t.disabled = false; return }
          await ui.invoke(PLUGIN, 'discard', { teamId })
        } else if (act === 'continue') {
          await ui.invoke(PLUGIN, 'continuePlanning', { teamId })
        } else if (act === 'halt') {
          if (!confirm('暂停团队并取消所有未完成任务？')) { t.disabled = false; return }
          await ui.invoke(PLUGIN, 'halt', { teamId })
        }
      } catch (err) {
        console.warn('[agent-teams] ' + act + ' 失败', err)
        alert('操作失败: ' + ((err && err.message) || err))
      }
      t.disabled = false
      refresh()
    })

    refresh()
    timerId = setInterval(refresh, 3000)
  }

  function closePanel() {
    if (timerId) { clearInterval(timerId); timerId = null }
    if (panelEl) { panelEl.remove(); panelEl = null }
  }

  // ─── 注册标题栏按钮 ──────────────────────────────────────
  ui.registerSlot({
    slotId: SLOT,
    title: '多智能体团队（agent-teams）',
    kind: 'list',
    render(el) {
      const btn = document.createElement('button')
      btn.className = 'agteams-btn'
      btn.innerHTML = SVG.team + '<span>团队</span><span class="agteams-badge" id="agteams-badge" style="display:none">0</span>' + SVG.caret
      btn.addEventListener('click', openPanel)
      el.appendChild(btn)

      // 初始徽标
      fetchTeams().then((teams) => updateBadge(teams)).catch(() => {})

      return () => {
        closePanel()
        btn.remove()
      }
    },
  })
}
