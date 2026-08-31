// ═══════════════════════════════════════════════════════════════
// agent-teams — client 半：(ui) => void
//
// 标题栏 titlebar-right 槽位「团队」按钮 + 活动浮层：
//   · 点击展开团队活动面板（成员卡片/任务 DAG/进度/未读）
//   · 实时性：宿主写团队状态即广播 ui:agent-teams/change → ui.on 订阅 → 事件驱动刷新
//     （不经周期轮询，避免占用 goja VM 锁；事件由浏览器侧 2s 事件轮询带过来）
//   · 任务 DAG：按依赖深度分层展示，每层显示依赖链/阻塞来源；终态任务可删除
//     （ui.invoke 'deleteTask'，宿主清理依赖引用）
//   · 折叠/收缩：团队卡片 / 成员区 / 任务区 / DAG 层均可点击折叠（DOM 纯切换）
//   · staged 计划：批准并运行 / 返回对话修改 / 废弃
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
    external: '<svg viewBox="0 0 16 16" width="10" height="10" fill="currentColor" aria-hidden="true"><path d="M10 1h5v5h-1.6V3.6L8 9l-1-1 5.4-5.4H10V1zM3.5 2.5h3V4h-3V12.5h8.5v-3H14v3A1.5 1.5 0 0 1 12.5 14h-9A1.5 1.5 0 0 1 2 12.5V4A1.5 1.5 0 0 1 3.5 2.5z"/></svg>',
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
.agteams-panel{position:fixed;z-index:9000;top:34px;right:10px;width:min(560px,calc(100vw - 24px));height:min(680px,calc(100vh - 48px));display:flex;flex-direction:column;background:var(--bg-secondary,#1c2128);border:1px solid var(--border-color,#30363d);border-radius:10px;box-shadow:0 10px 34px rgba(0,0,0,.5);overflow:hidden;font-size:12px;color:var(--text-primary,#e6edf3)}
.agteams-panel-head{display:flex;align-items:center;justify-content:space-between;padding:8px 12px;background:var(--bg-tertiary,#21262d);border-bottom:1px solid var(--border-color,#30363d);font-size:13px;font-weight:600}
.agteams-panel-head .agteams-ico{display:flex;align-items:center;justify-content:center;width:22px;height:22px;border:none;background:none;color:var(--text-muted,#8b949e);cursor:pointer;border-radius:4px;padding:0}
.agteams-panel-head .agteams-ico:hover{color:var(--text-primary,#e6edf3);background:var(--bg-hover,#2d333b)}
.agteams-body{overflow-y:auto;flex:1;min-height:0;padding:10px 12px}
.agteams-empty{padding:22px 8px;text-align:center;color:var(--text-muted,#8b949e);font-size:12px;line-height:1.7}
.agteams-empty .t{color:var(--text-secondary,#c9d1d9);font-weight:600}
.agteams-card{margin-bottom:10px;border:1px solid var(--border-color,#30363d);border-radius:8px;background:var(--bg-tertiary,#21262d);overflow:hidden}
.agteams-card-head{display:flex;align-items:center;gap:8px;padding:7px 10px;border-bottom:1px solid var(--border-color,#30363d);cursor:pointer}
.agteams-card-head:hover{background:var(--bg-hover,#2d333b)}
.agteams-card-name{flex:1;min-width:0;font-weight:600;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.agteams-phase{display:inline-flex;align-items:center;gap:3px;padding:1px 7px;font-size:10px;border-radius:8px;border:1px solid var(--border-color,#30363d);color:var(--text-secondary,#c9d1d9)}
.agteams-phase.running{color:#7ee787;border-color:rgba(126,231,135,.35)}
.agteams-phase.staged{color:#f0883e;border-color:rgba(240,136,62,.35)}
.agteams-phase.halted{color:#ff7b72;border-color:rgba(255,123,114,.35)}
.agteams-card-desc{padding:6px 10px;font-size:11px;color:var(--text-muted,#8b949e);line-height:1.5;border-bottom:1px solid var(--border-color,#30363d);max-height:none;overflow:visible;white-space:normal;word-break:break-word}
.agteams-progress{height:5px;background:var(--bg-secondary,#161b22);border-radius:3px;margin:8px 10px 4px;overflow:hidden}
.agteams-progress i{display:block;height:100%;background:var(--accent-color,#4f8cff);border-radius:3px;transition:width .4s}
.agteams-stats{padding:0 10px 8px;font-size:10px;color:var(--text-muted,#8b949e);display:flex;gap:12px}
.agteams-sec{padding:6px 10px;font-size:10px;font-weight:600;color:var(--text-muted,#8b949e);border-top:1px solid var(--border-color,#30363d);display:flex;align-items:center;gap:5px;cursor:pointer}
.agteams-sec:hover{background:var(--bg-hover,#2d333b)}
.agteams-sec .hint{font-weight:400;font-size:9px;margin-left:auto}
.agteams-sec-body{overflow:hidden}
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
.agteams-foot-note{color:var(--text-muted,#8b949e)}
.agteams-spin{animation:agteams-rot 1s linear infinite;display:inline-block}
@keyframes agteams-rot{to{transform:rotate(360deg)}}
@keyframes agteams-pulse{0%,100%{opacity:1}50%{opacity:.35}}
/* ★ 折叠 / DAG 状态徽章 / 删除按钮 / 实时脚注（2026-08-31）★ */
.agteams-chevr{display:inline-flex;align-items:center;transition:transform .15s;flex-shrink:0;color:var(--text-muted,#8b949e)}
.agteams-card-head .agteams-chevr{transform:rotate(90deg)}
.agteams-card-head.collapsed .agteams-chevr{transform:rotate(0deg)}
.agteams-card-head.collapsed + .agteams-card-body{display:none}
.agteams-sec .agteams-chevr{transform:rotate(90deg)}
.agteams-sec.collapsed .agteams-chevr{transform:rotate(0deg)}
.agteams-sec.collapsed + .agteams-sec-body{display:none}
.agteams-layer{overflow:hidden}
.agteams-layer-head{display:flex;align-items:center;gap:5px;padding:4px 10px;font-size:9px;font-weight:600;color:var(--text-muted,#8b949e);cursor:pointer;border-top:1px solid var(--border-color,#30363d);background:var(--bg-secondary,#161b22)}
.agteams-layer-head:hover{background:var(--bg-hover,#2d333b)}
.agteams-layer-head .agteams-chevr{transform:rotate(90deg)}
.agteams-layer-head.collapsed .agteams-chevr{transform:rotate(0deg)}
.agteams-layer-head.collapsed + .agteams-layer-body{display:none}
.agteams-layer-head .cnt{margin-left:auto;font-weight:400}
.agteams-layer-head .hint{font-weight:400;font-size:8px}
.agteams-task .tb{display:inline-flex;align-items:center;padding:0 5px;font-size:9px;border-radius:7px;border:1px solid var(--border-color,#30363d);color:var(--text-muted,#8b949e);flex-shrink:0;line-height:14px;white-space:nowrap}
.agteams-task.completed .tb{color:#7ee787;border-color:rgba(126,231,135,.35)}
.agteams-task.failed .tb{color:#ff7b72;border-color:rgba(255,123,114,.35)}
.agteams-task.cancelled .tb{color:#8b949e}
.agteams-task.in_progress .tb{color:#3fb950;border-color:rgba(63,185,80,.35)}
.agteams-task.claimed .tb{color:#f0883e;border-color:rgba(240,136,62,.35)}
.agteams-task .depblk{font-size:9px;color:#f0883e;flex-shrink:0;white-space:nowrap}
.agteams-task .depdone{font-size:9px;color:var(--text-muted,#8b949e);flex-shrink:0;white-space:nowrap}
.agteams-task .agteams-del{display:none;align-items:center;justify-content:center;width:16px;height:16px;border:none;background:none;color:var(--text-muted,#8b949e);cursor:pointer;border-radius:3px;padding:0;flex-shrink:0}
.agteams-task:hover .agteams-del{display:inline-flex}
.agteams-task .agteams-del:hover{color:#ff7b72;background:rgba(255,123,114,.12)}
/* ── 成员卡片：子会话打开入口（openMember）── */\n.agteams-member.clickable{cursor:pointer}\n.agteams-member .open{display:inline-flex;align-items:center;justify-content:center;width:14px;height:14px;color:var(--text-muted,#8b949e);border-radius:3px;flex-shrink:0;margin-left:2px}\n.agteams-member.clickable:hover .open{color:var(--accent-color,#4f8cff)}\n/* ── 任务 DAG 流程图（compact 左→右；对齐 dagViewport/dagEdges/dagNode）── */
.agteams-viewport{overflow-x:auto;padding:6px 10px 8px;scrollbar-width:thin}
.agteams-canvas{position:relative}
.agteams-edges{position:absolute;inset:0;overflow:visible;pointer-events:none}
.agteams-edges path{fill:none;stroke:var(--border-color,#3d444d);stroke-width:1;transition:opacity 140ms ease,stroke 140ms ease,stroke-width 140ms ease}
.agteams-edges path[data-active='true']{stroke:var(--accent-color,#4f8cff);stroke-width:1.6}
.agteams-edges path[data-dimmed='true']{opacity:0.24}
.agteams-node{position:absolute;display:flex;flex-direction:column;justify-content:center;gap:1px;padding:0 6px;box-sizing:border-box;border:1px solid var(--border-color,#30363d);border-radius:6px;background:var(--bg-tertiary,#21262d);color:var(--text-primary,#e6edf3);cursor:pointer;transition:border-color 140ms ease,background-color 140ms ease,opacity 140ms ease}
.agteams-node:hover,.agteams-node[data-focused='true']{border-color:var(--accent-color,#4f8cff);background:color-mix(in srgb,var(--accent-color,#4f8cff) 6%,var(--bg-tertiary,#21262d))}
.agteams-node[data-dimmed='true']{opacity:0.3}
.agteams-node[data-state='running'][data-dimmed='true']{opacity:0.58}
.agteams-node[data-state='completed']{border-color:rgba(126,231,135,.48)}
.agteams-node[data-state='blocked']{border-color:rgba(240,136,62,.52)}
.agteams-node[data-state='failed']{border-color:rgba(255,123,114,.56)}
.agteams-node-head{display:flex;align-items:center;gap:4px;color:var(--text-primary,#e6edf3);font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:9.5px;font-weight:700}
.agteams-node-dot{flex:none;width:5px;height:5px;border-radius:1.5px;background:var(--border-color,#3d444d)}
.agteams-node[data-state='running'] .agteams-node-dot{background:var(--accent-color,#4f8cff)}
.agteams-node[data-state='running'] .agteams-node-head{padding-right:12px}
.agteams-node-run{position:absolute;top:4px;right:5px;display:inline-flex;width:9px;height:9px;align-items:center;justify-content:center;color:var(--accent-color,#4f8cff);animation:agteams-spin 1s linear infinite;pointer-events:none}
@keyframes agteams-spin{to{transform:rotate(360deg)}}
.agteams-node[data-state='blocked'] .agteams-node-dot{background:#f0883e}
.agteams-node[data-state='completed'] .agteams-node-dot{background:#7ee787}
.agteams-node[data-state='failed'] .agteams-node-dot{background:#ff7b72}
.agteams-node-label{overflow:hidden;color:var(--text-muted,#8b949e);font-size:8.5px;line-height:11px;text-overflow:ellipsis;white-space:nowrap}
.agteams-node .agteams-del{display:none;position:absolute;top:2px;right:2px;width:13px;height:13px;border:none;background:none;color:var(--text-muted,#8b949e);cursor:pointer;border-radius:3px;padding:0;z-index:2}
.agteams-node:hover .agteams-del{display:inline-flex;align-items:center;justify-content:center}
.agteams-node .agteams-del:hover{color:#ff7b72;background:rgba(255,123,114,.12)}
`
    document.head.appendChild(st)
  }

  // ─── 状态 ────────────────────────────────────────────────
  let panelEl = null
  let timerId = null
  let busy = false

  // 折叠状态记忆（render 每次重绘后恢复；面板生命周期内保留）
  const collapsedHeads = new Set()

  // ─── 任务 DAG 聚焦状态（pin=点击固定 / hover=悬停预览；render 依赖二者）───
  let pinnedTaskId = null
  let hoverTaskId = null
  let hoverTimer = null

  // relatedTaskIds 计算某任务的全部上下游相关链（上游依赖 + 下游被依赖；环安全）。
  // 对齐 activity-model.relatedTaskIds：focus 一个任务时高亮其整条链、其余降亮。
  function relatedTaskIds(id, tasks) {
    const byId = {}
    tasks.forEach((t) => { byId[t.id] = t })
    if (!byId[id]) return null
    const dependents = {}
    tasks.forEach((t) => (t.dependencies || []).forEach((d) => { (dependents[d] = dependents[d] || []).push(t.id) }))
    const related = new Set()
    const upSeen = new Set()
    const downSeen = new Set()
    const walkUp = (i) => {
      if (upSeen.has(i)) return
      upSeen.add(i)
      related.add(i)
      ;(byId[i].dependencies || []).forEach(walkUp)
    }
    const walkDown = (i) => {
      if (downSeen.has(i)) return
      downSeen.add(i)
      related.add(i)
      ;(dependents[i] || []).forEach(walkDown)
    }
    walkUp(id)
    walkDown(id)
    return related
  }

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
    body.innerHTML = teams.map((team) => {
      const members = (team.members || []).filter((m) => m.status !== 'removed')
      const total = members.length
      const done = members.filter((m) => m.done > 0 && m.done === m.total && m.total > 0).length
      const pct = total === 0 ? 0 : Math.round((done / total) * 100)
      const tasks = (team.tasks || [])
      const doneTasks = tasks.filter((t) => t.status === 'completed').length
      const phaseCls = team.halted ? 'halted' : (team.phase === 'staged' ? 'staged' : 'running')
      const phaseTxt = team.halted ? '已暂停' : (team.phase === 'staged' ? '待批准' : '运行中')
      const cardKey = 'card:' + team.teamId
      // ── 成员区 ──
      const memberHtml = members.map((m) => {
        const initial = (m.name || '?').slice(0, 1).toUpperCase()
        const activity = m.activity === 'working' ? 'working' : ''
        const st = m.activity === 'working' ? '工作中' : (m.activity === 'idle' ? '空闲' : '—')
        const pg = m.total === 0 ? 0 : Math.round((m.done / m.total) * 100)
        const unread = m.unread > 0 ? '<span style="color:#f0883e">' + m.unread + ' 未读</span>' : ''
        // ★ 2026-08-31 子会话入口：成员会话（conv_sub_*）不出现在顶层会话列表，
        //   从团队面板内 openMember 打开（派发 open-conversation 事件 → 前端切换会话）。
        return '<div class="agteams-member' + (m.id ? ' clickable' : '') + ' ' + activity + '"' +
          (m.id ? ' data-act="openMember" data-conv="' + esc(m.id) + '" title="打开成员会话（子会话，不占顶层列表）"' : '') + '>' +
          '<span class="av">' + esc(initial) + '</span>' +
          '<span class="nm">' + esc(m.name) + '</span>' +
          (m.role ? '<span class="rl">' + esc(m.role) + '</span>' : '<span class="rl"></span>') +
          '<span class="pg"><i style="width:' + pg + '%"></i></span>' +
          '<span class="st"><span class="dot"></span>' + st + (m.currentTask ? ' · ' + esc(m.currentTask) : '') + ' ' + unread + '</span>' +
          (m.id ? '<span class="open" title="打开会话">' + SVG.external + '</span>' : '') +
        '</div>'
      }).join('')
      // ── 任务 DAG 区：compact 左→右依赖流程图（节点卡片 + SVG 贝塞尔连线，对齐 compactDagLayout）──
      const STATUS_TXT = { pending: '待办', claimed: '已认领', in_progress: '执行中', completed: '完成', failed: '失败', cancelled: '已取消' }
      const TERMINAL = ['completed', 'failed', 'cancelled']
      // 常量（与 activity-model COMPACT_DAG_* 一致）
      const DAG = { NODE_W: 92, NODE_H: 30, COL_GAP: 26, ROW_GAP: 8 }
      // 列 = 依赖深度 stage；列内按任务 ID 稳定排序 → 节点绝对坐标
      const byDepth = {}
      tasks.forEach((t) => { (byDepth[t.depth] = byDepth[t.depth] || []).push(t) })
      const depths = Object.keys(byDepth).sort((a, b) => Number(a) - Number(b))
      const pos = new Map()
      const dagNodes = []
      depths.forEach((d, col) => {
        const rowTasks = byDepth[d].slice().sort((a, b) => a.id.localeCompare(b.id, 'en', { numeric: true }))
        rowTasks.forEach((t, row) => {
          pos.set(t.id, { x: col * (DAG.NODE_W + DAG.COL_GAP), y: row * (DAG.NODE_H + DAG.ROW_GAP) })
          dagNodes.push({ task: t, x: pos.get(t.id).x, y: pos.get(t.id).y })
        })
      })
      // 边：源节点右缘中点 → 目标节点左缘中点，三次贝塞尔曲线（fan-in 保持可读）
      const dagEdges = []
      tasks.forEach((t) => {
        const dst = pos.get(t.id)
        if (!dst) return
        ;(t.dependencies || []).forEach((d) => {
          const src = pos.get(d)
          if (!src) return
          const x1 = src.x + DAG.NODE_W
          const y1 = src.y + DAG.NODE_H / 2
          const x2 = dst.x
          const y2 = dst.y + DAG.NODE_H / 2
          dagEdges.push({ from: d, to: t.id, path: 'M' + x1 + ' ' + y1 + 'C' + (x1 + 14) + ' ' + y1 + ',' + (x2 - 14) + ' ' + y2 + ',' + x2 + ' ' + y2 })
        })
      })
      const dagRows = Math.max(1, ...depths.map((d) => byDepth[d].length))
      const dagW = depths.length ? depths.length * DAG.NODE_W + (depths.length - 1) * DAG.COL_GAP : 0
      const dagH = dagRows * DAG.NODE_H + (dagRows - 1) * DAG.ROW_GAP
      // 聚焦任务（pin 优先于 hover）：计算其上下游相关链，其余节点/边降亮
      const focusId = pinnedTaskId || hoverTaskId
      const related = focusId && dagNodes.length ? relatedTaskIds(focusId, tasks) : null
      const nodeTitle = (t) => {
        const parts = [t.id + ' [' + (STATUS_TXT[t.status] || t.status) + ']']
        if (t.kind && t.kind !== 'work') parts.push('[' + t.kind + (t.round ? ' r' + t.round : '') + ']')
        if (t.assignee) parts.push('@' + t.assignee)
        if (t.state === 'blocked') parts.push('⛔ 等待依赖完成')
        return esc(parts.join(' '))
      }
      const taskNodeHtml = (n) => {
        const t = n.task
        const dim = related !== null && !related.has(t.id)
        return '<div class="agteams-node" data-act="focusTask" data-task="' + esc(t.id) + '" data-team="' + esc(team.teamId) + '"' +
          ' data-state="' + esc(t.state) + '" data-focused="' + (t.id === focusId ? 'true' : 'false') + '"' +
          ' data-dimmed="' + (dim ? 'true' : 'false') + '" style="left:' + n.x + 'px;top:' + n.y + 'px"' +
          ' title="' + nodeTitle(t) + '">' +
          (t.state === 'running' ? '<span class="agteams-node-run">' + SVG.spin + '</span>' : '') +
          '<span class="agteams-node-head"><span class="agteams-node-dot"></span>' + esc(t.id) + '</span>' +
          '<span class="agteams-node-label">' + esc(t.subject.length > 12 ? t.subject.slice(0, 11) + '…' : t.subject) + '</span>' +
          (TERMINAL.includes(t.status)
            ? '<button class="agteams-del" data-act="deleteTask" data-team="' + esc(team.teamId) + '" data-task="' + esc(t.id) + '" title="删除任务（清理依赖引用）">' + SVG.xmark + '</button>'
            : '') +
        '</div>'
      }
      const taskHtml = dagNodes.length
        ? '<div class="agteams-viewport"><div class="agteams-canvas" style="width:' + dagW + 'px;height:' + dagH + 'px">' +
            '<svg class="agteams-edges" width="' + dagW + '" height="' + dagH + '">' +
              dagEdges.map((e) => '<path d="' + e.path + '" data-from="' + esc(e.from) + '" data-to="' + esc(e.to) + '"' +
                (related !== null && related.has(e.from) && related.has(e.to) ? ' data-active="true"' : (related !== null ? ' data-dimmed="true"' : '')) + '></path>').join('') +
            '</svg>' +
            dagNodes.map(taskNodeHtml).join('') +
          '</div></div>'
        : '<div class="agteams-empty" style="padding:8px">（暂无任务）</div>'
      // ── 队长信箱预览 ──
      const mailPreview = (team.captainInbox || []).slice(-2).map((m) => {
        return '<div class="agteams-mail">' + esc(m.from) + ' → 队长: ' + esc(m.content.length > 60 ? m.content.slice(0, 57) + '…' : m.content) + '</div>'
      }).join('')
      // ── 操作区 ──
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
        '<div class="agteams-card-head' + (collapsedHeads.has(cardKey) ? ' collapsed' : '') + '" data-act="toggleCard" data-team="' + esc(team.teamId) + '" title="点击折叠/展开">' +
          '<span class="agteams-chevr">' + SVG.caret + '</span>' +
          '<span class="agteams-card-name">' + esc(team.name) + '</span>' +
          '<span class="agteams-phase ' + phaseCls + '">' + phaseTxt + '</span>' +
        '</div>' +
        '<div class="agteams-card-body">' +
        (team.description ? '<div class="agteams-card-desc">' + esc(team.description) + '</div>' : '') +
        '<div class="agteams-progress"><i style="width:' + pct + '%"></i></div>' +
        '<div class="agteams-stats">成员 ' + total + ' · 任务 ' + doneTasks + '/' + tasks.length + ' 完成' + (team.escalated ? ' · <span style="color:#f0883e">已升级</span>' : '') + '</div>' +
        (members.length ? '<div class="agteams-sec' + (collapsedHeads.has('members:' + team.teamId) ? ' collapsed' : '') + '" data-act="toggleSec" data-sec="members:' + team.teamId + '" data-team="' + esc(team.teamId) + '"><span class="agteams-chevr">' + SVG.caret + '</span>成员<span class="hint">' + total + '</span></div><div class="agteams-sec-body">' + memberHtml + '</div>' : '') +
        '<div class="agteams-sec' + (collapsedHeads.has('tasks:' + team.teamId) ? ' collapsed' : '') + '" data-act="toggleSec" data-sec="tasks:' + team.teamId + '" data-team="' + esc(team.teamId) + '"><span class="agteams-chevr">' + SVG.caret + '</span>任务 DAG<span class="hint">' + tasks.length + '</span></div>' +
        '<div class="agteams-sec-body">' + taskHtml + '</div>' +
        mailPreview +
        actions +
        '</div>' +
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

  // 实时刷新：宿主写团队状态即广播 ui:agent-teams/change（事件驱动，无轮询锁竞争）。
  // 多个事件合并为一次刷新（600ms 节流），事件由浏览器侧 2s 事件轮询带过来。
  let refreshTimer = null
  function scheduleRefresh() {
    if (refreshTimer) return
    refreshTimer = setTimeout(() => { refreshTimer = null; refresh() }, 600)
  }
  try { ui.on('ui:agent-teams/change', () => scheduleRefresh()) } catch (e) { /* 旧宿主无事件桥 */ }

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
      '<div class="agteams-foot"><span>实时 · 事件驱动刷新</span><span class="agteams-foot-note">agent-teams</span></div>'
    document.body.appendChild(panelEl)

    // 事件代理（折叠 / 批准 / 废弃 / 暂停 / 删除任务 / 关闭）
    panelEl.addEventListener('click', async (e) => {
      const t = e.target.closest('[data-act]')
      if (!t) return
      const act = t.getAttribute('data-act')
      if (act === 'close') { closePanel(); return }
      // 折叠/展开（团队卡片 / 分区 / DAG 层）：纯 DOM，不发后端请求
      if (act === 'toggleCard' || act === 'toggleSec') {
        const key = t.getAttribute('data-sec') || ('card:' + (t.getAttribute('data-team') || ''))
        t.classList.toggle('collapsed')
        if (t.classList.contains('collapsed')) collapsedHeads.add(key)
        else collapsedHeads.delete(key)
        return
      }
      // 任务节点点击：固定/取消固定该任务的聚焦链（pin；再次点击同一节点取消）
      if (act === 'focusTask') {
        const taskId = t.getAttribute('data-task') || ''
        pinnedTaskId = (pinnedTaskId === taskId ? null : taskId)
        refresh()
        return
      }
      // 成员卡片点击：打开成员子会话（openMember）——成员会话不在顶层会话列表，
      // 经 open-conversation 全局事件切到前端会话视图（app-actions handler 设置
      // state.currentConvId → RightPanel watch → switchConv 加载消息）。
      if (act === 'openMember') {
        const convId = t.getAttribute('data-conv') || ''
        if (!convId) return
        window.dispatchEvent(new CustomEvent('open-conversation', { detail: { id: convId } }))
        return
      }
      const teamId = t.getAttribute('data-team') || ''
      t.disabled = true
      try {
        if (act === 'approve') {
          await ui.invoke(PLUGIN, 'approve', { teamId })
        } else if (act === 'discard') {
          if (!confirm('废弃该暂存计划？（不可恢复，将归档）')) { t.disabled = false; return }
          await ui.invoke(PLUGIN, 'discard', { teamId })
          alert('已废弃并归档该团队计划（' + teamId + '）')
        } else if (act === 'continue') {
          await ui.invoke(PLUGIN, 'continuePlanning', { teamId })
        } else if (act === 'halt') {
          if (!confirm('暂停团队并取消所有未完成任务？')) { t.disabled = false; return }
          await ui.invoke(PLUGIN, 'halt', { teamId })
          alert('已暂停团队（' + teamId + '）')
        } else if (act === 'deleteTask') {
          const taskId = t.getAttribute('data-task') || ''
          if (!confirm('删除任务 ' + taskId + '？（仅已结束任务可删，依赖引用将被清理）')) { t.disabled = false; return }
          await ui.invoke(PLUGIN, 'deleteTask', { teamId, taskId })
        }
      } catch (err) {
        console.warn('[agent-teams] ' + act + ' 失败', err)
        alert('操作失败: ' + ((err && err.message) || err))
      }
      t.disabled = false
      refresh()
    })

    // ── 任务节点悬停预览：180ms 防抖后高亮上下游相关链（对齐 scheduleHover）──
    panelEl.addEventListener('mouseover', (e) => {
      const n = e.target.closest('.agteams-node')
      if (!n) return
      const id = n.getAttribute('data-task') || ''
      if (!id) return
      if (hoverTimer) clearTimeout(hoverTimer)
      hoverTimer = setTimeout(() => { hoverTimer = null; hoverTaskId = id; refresh() }, 180)
    })
    panelEl.addEventListener('mouseout', (e) => {
      const from = e.target.closest('.agteams-node')
      if (!from) return
      const to = e.relatedTarget && e.relatedTarget.closest ? e.relatedTarget.closest('.agteams-node') : null
      if (to) return // 移到另一节点：mouseover 会重新调度
      if (hoverTimer) { clearTimeout(hoverTimer); hoverTimer = null }
      if (hoverTaskId) { hoverTaskId = null; refresh() }
    })

    refresh()
    // ★ 实时性：宿主写团队状态即广播 ui:agent-teams/change（上方 ui.on 订阅），
    //   事件到达后合并刷新。此处保留打开时取一次 + 操作后手动 refresh（动作保底）。
    //   不做周期轮询：agent-teams 在单个 goja VM 锁内串行，周期 GET 会持续占用
    //   VM 锁推高争抢；事件由浏览器侧 2s 事件轮询带过来，不占宿主 VM 锁。
  }

  function closePanel() {
    if (timerId) { clearInterval(timerId); timerId = null }
    if (hoverTimer) { clearTimeout(hoverTimer); hoverTimer = null }
    pinnedTaskId = null
    hoverTaskId = null
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