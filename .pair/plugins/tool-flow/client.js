// ═══════════════════════════════════════════════════════════════
// tool-flow — client 半（工程流程看板：主内容区「流程」tab）
//
// 设计真源：docs/flow-board-plugin-design.md §6 看板 UI / §14.6-② 用户感知
//   · 三个页签：「当前流程」（阶段泳道 + 门禁 + 证据展开）/「项目 × 流程矩阵」/「流程库」
//   · 数据面全部来自 host 半只读接口 GET /api/flow/state（事实数据走持久化接口）
//   · 写操作（过门禁/推进/批准/启动/绑定）走 ui.invoke RPC → host 半 flow_action
//
// ★ 渲染纪律：一切文本（含 agent/用户产出）走 textContent —— 绝不用 innerHTML 拼插值；
//   本文件仅静态 CSS 经 style.textContent 注入。
// ★ 会话跟随：快照按当前会话请求（convId），切会话自动重拉；多会话并行时不串台。
// ★ 样式：沿用壳的 CSS 变量（--bg-*/--text-*/--border-*）与插件既有语义色
//   （#6a9955 通过 / #d4a74e 待办），不自造配色方案。
// ═══════════════════════════════════════════════════════════════
(ui) => {
  const PLUGIN = 'tool-flow'
  const VIEW_ID = 'flow-board'
  const VIEW_TITLE = '任务看板'
  const POLL_MS = 4000

  const CSS = [
    '.ftb-root{display:flex;flex-direction:column;height:100%;min-height:0;background:var(--bg-primary)}',
    '.ftb-head{display:flex;align-items:center;gap:8px;flex-wrap:wrap;flex-shrink:0;padding:8px 12px;border-bottom:1px solid var(--border-color);background:var(--bg-secondary)}',
    '.ftb-title{font-size:13px;font-weight:600;color:var(--text-primary)}',
    '.ftb-sub{font-size:11px;color:var(--text-muted);font-variant-numeric:tabular-nums}',
    '.ftb-spacer{flex:1}',
    '.ftb-btn{font:inherit;font-size:11px;color:var(--text-secondary);background:transparent;border:1px solid var(--border-color);border-radius:3px;padding:2px 8px;cursor:pointer}',
    '.ftb-btn:hover{background:var(--bg-active);color:var(--text-primary)}',
    '.ftb-btn:disabled{opacity:.45;cursor:default}',
    '.ftb-btn.primary{color:var(--text-primary);border-color:currentColor}',
    '.ftb-tabs{display:flex;gap:2px;flex-shrink:0;padding:6px 12px 0;border-bottom:1px solid var(--border-color);background:var(--bg-secondary)}',
    '.ftb-tab{font:inherit;font-size:11px;color:var(--text-muted);background:transparent;border:1px solid transparent;border-bottom:none;border-radius:3px 3px 0 0;padding:3px 10px;cursor:pointer}',
    '.ftb-tab:hover{color:var(--text-secondary)}',
    '.ftb-tab.active{color:var(--text-primary);background:var(--bg-primary);border-color:var(--border-color)}',
    '.ftb-msg{font-size:11px;line-height:1.6;color:var(--text-secondary);background:var(--bg-active);border-radius:3px;padding:4px 8px;margin:8px 12px 0;white-space:pre-wrap;word-break:break-word;max-height:180px;overflow:auto}',
    '.ftb-body{flex:1;min-height:0;overflow-y:auto;padding:10px 12px}',
    '.ftb-empty{max-width:560px;margin:28px auto;font-size:12px;line-height:1.7;color:var(--text-muted);text-align:center}',
    '.ftb-bar{height:6px;border-radius:3px;background:var(--bg-active);overflow:hidden;margin:6px 0 10px}',
    '.ftb-bar-fill{height:100%;background:#6a9955}',
    '.ftb-stage{border:1px solid var(--border-color);border-radius:var(--border-radius);background:var(--bg-secondary);padding:8px 10px;font-size:12px}',
    '.ftb-stage+.ftb-stage{margin-top:8px}',
    '.ftb-stage-head{display:flex;align-items:center;gap:6px;flex-wrap:wrap}',
    '.ftb-idx{color:var(--text-muted);font-variant-numeric:tabular-nums}',
    '.ftb-name{color:var(--text-primary);font-weight:600}',
    '.ftb-goal{margin-top:3px;color:var(--text-muted);font-size:11px;line-height:1.5}',
    '.ftb-badge{flex-shrink:0;font-size:10px;line-height:1.5;padding:0 5px;border-radius:8px;border:1px solid currentColor;letter-spacing:.3px}',
    '.ftb-badge.done,.ftb-badge.pass,.ftb-badge.approved,.ftb-badge.active{color:#6a9955}',
    '.ftb-badge.todo,.ftb-badge.pending{color:#d4a74e}',
    '.ftb-badge.fail{color:var(--danger,#e5534b)}',
    '.ftb-gate{display:flex;align-items:center;gap:6px;flex-wrap:wrap;margin-top:5px;padding:4px 6px;background:var(--bg-primary);border-radius:3px}',
    '.ftb-gate-id{color:var(--text-secondary);font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:11px}',
    '.ftb-gate-text{color:var(--text-primary);font-size:11px;flex:1;min-width:140px}',
    '.ftb-gate-kind{color:var(--text-muted);font-size:10px}',
    '.ftb-input{font:inherit;font-size:11px;color:var(--text-primary);background:var(--bg-secondary);border:1px solid var(--border-color);border-radius:3px;padding:2px 6px;flex:1;min-width:140px}',
    '.ftb-more{margin-top:4px}',
    '.ftb-more>summary{cursor:pointer;font-size:11px;color:var(--text-muted);user-select:none}',
    '.ftb-more>summary:hover{color:var(--text-secondary)}',
    '.ftb-pre{margin:4px 0 0;padding:4px 6px;background:var(--bg-primary);border-radius:3px;font-size:11px;line-height:1.6;color:var(--text-secondary);white-space:pre-wrap;word-break:break-word;max-height:240px;overflow:auto}',
    '.ftb-row{display:flex;gap:8px;align-items:center;padding:5px 6px;border-bottom:1px solid var(--border-color);font-size:12px}',
    '.ftb-row:hover{background:var(--bg-active)}',
    '.ftb-cell{color:var(--text-secondary);font-size:11px;flex-shrink:0}',
    '.ftb-cell.wide{flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}',
    '.ftb-note{color:var(--text-muted);font-size:11px;line-height:1.6;margin:6px 0}',
    '.ftb-editor{border:1px solid var(--border-color);border-radius:var(--border-radius);background:var(--bg-secondary);padding:8px 10px}',
    '.ftb-textarea{width:100%;min-height:340px;margin-top:6px;box-sizing:border-box;font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:11px;line-height:1.5;color:var(--text-primary);background:var(--bg-primary);border:1px solid var(--border-color);border-radius:3px;padding:6px;resize:vertical}',
    '.ftb-toolbar{display:flex;align-items:center;gap:6px;flex-wrap:wrap;margin:6px 0}',
    '.ftb-seg{display:inline-flex;border:1px solid var(--border-color);border-radius:3px;overflow:hidden}',
    '.ftb-seg>button{font:inherit;font-size:11px;padding:2px 10px;color:var(--text-muted);background:transparent;border:none;cursor:pointer}',
    '.ftb-seg>button.active{color:var(--text-primary);background:var(--bg-active)}',
    '.ftb-block{border:1px solid var(--border-color);border-radius:var(--border-radius);background:var(--bg-secondary);padding:8px 10px;margin-top:8px}',
    '.ftb-block-head{display:flex;align-items:center;gap:6px;flex-wrap:wrap;margin-bottom:4px}',
    '.ftb-field{display:flex;align-items:center;gap:6px;margin:4px 0}',
    '.ftb-field>.ftb-label{flex-shrink:0;width:92px;color:var(--text-muted);font-size:11px;text-align:right}',
    '.ftb-field>.ftb-label.inline{width:auto;text-align:left;color:var(--text-secondary)}',
    '.ftb-field>input[type=text],.ftb-field>input[type=number],.ftb-field>select{font:inherit;font-size:11px;color:var(--text-primary);background:var(--bg-primary);border:1px solid var(--border-color);border-radius:3px;padding:2px 6px;flex:1;min-width:120px;box-sizing:border-box}',
    '.ftb-field>input[type=checkbox]{flex-shrink:0}',
    '.ftb-field>input:disabled,.ftb-field>select:disabled{opacity:.6;cursor:not-allowed}',
    '.ftb-gate-box{border-top:1px solid var(--border-color);margin-top:6px;padding-top:6px}',
    '.ftb-gate-form{margin-top:5px;padding:6px;background:var(--bg-primary);border-radius:3px}',
    '.ftb-gate-form .ftb-field>.ftb-label{width:72px}',
    '.ftb-err{color:var(--danger,#e5534b);font-size:11px;line-height:1.6;margin:6px 0;white-space:pre-wrap}',
    '.ftb-readonly{color:#d4a74e;font-size:11px;line-height:1.6;margin:6px 0}',
    '.ftb-mini{font-size:11px;padding:1px 6px}',
    // ── 焦点条：一眼看清「此刻在哪、在做什么、卡在哪」────────────────
    '.ftb-focus{border:1px solid var(--border-color);border-left:3px solid #d4a74e;border-radius:var(--border-radius);background:var(--bg-secondary);padding:10px 12px;margin-bottom:10px}',
    '.ftb-focus.done{border-left-color:#6a9955}',
    '.ftb-focus.err{border-left-color:var(--danger,#e5534b)}',
    '.ftb-focus-title{font-size:13px;font-weight:600;color:var(--text-primary);display:flex;align-items:center;gap:6px;flex-wrap:wrap}',
    '.ftb-focus-meta{margin-top:3px;font-size:11px;color:var(--text-muted);line-height:1.7}',
    '.ftb-focus-now{margin-top:6px;font-size:12px;color:var(--text-primary);display:flex;align-items:center;gap:6px;flex-wrap:wrap}',
    '.ftb-focus-acts{display:flex;align-items:center;gap:6px;flex-wrap:wrap;margin-top:8px;padding-top:8px;border-top:1px solid var(--border-color)}',
    '.ftb-pbar{height:8px;flex-shrink:0;border-radius:4px;background:var(--bg-active);overflow:hidden;margin:8px 0 2px}',
    '.ftb-pbar-fill{height:100%;background:#6a9955;transition:width .18s ease}',
    // ── 阶段时间线（垂直）────────────────────────────────────────
    '.ftb-tl{margin-top:2px}',
    '.ftb-node{display:flex;gap:9px}',
    '.ftb-node-rail{flex-shrink:0;width:12px;display:flex;flex-direction:column;align-items:center}',
    '.ftb-dot{width:11px;height:11px;border-radius:50%;border:2px solid var(--border-color);background:var(--bg-primary);margin-top:5px;box-sizing:border-box;flex-shrink:0}',
    '.ftb-dot.done{border-color:#6a9955;background:#6a9955}',
    '.ftb-dot.active{border-color:#d4a74e;background:var(--bg-primary);box-shadow:0 0 0 3px rgba(212,167,78,.16)}',
    '.ftb-line{flex:1;width:2px;background:var(--border-color);margin:2px 0}',
    '.ftb-line.done{background:#6a9955}',
    '.ftb-node-body{flex:1;min-width:0;padding-bottom:10px}',
    '.ftb-node-head{display:flex;align-items:center;gap:6px;flex-wrap:wrap;min-height:21px}',
    '.ftb-node-title{font-size:12px;font-weight:600;color:var(--text-secondary)}',
    '.ftb-node.cur .ftb-node-title{color:var(--text-primary)}',
    '.ftb-card{border:1px solid var(--border-color);border-radius:var(--border-radius);background:var(--bg-secondary);padding:7px 9px;margin-top:5px}',
    // 子任务卡（借鉴 dsh-taskboard：左侧状态色条 + 卡片式 + 悬停提升）
    '.ftb-sub{position:relative;display:flex;flex-direction:column;gap:4px;padding:5px 7px 5px 10px;font-size:12px;color:var(--text-primary);background:var(--bg-secondary);border:1px solid var(--border-color);border-radius:4px}',
    '.ftb-sub:hover{border-color:var(--text-secondary)}',
    '.ftb-sub::before{content:"";position:absolute;left:0;top:0;bottom:0;width:3px;background:var(--border-color)}',
    '.ftb-sub.s-pending::before{background:var(--text-muted)}',
    '.ftb-sub.s-doing::before{background:#d4a74e}',
    '.ftb-sub.s-done::before{background:#6a9955}',
    '.ftb-sub.s-blocked::before{background:var(--danger,#e5534b)}',
    '.ftb-sub.dragging{opacity:.45}',
    '.ftb-sub-row{display:flex;align-items:center;gap:6px;min-width:0}',
    '.ftb-sub-note{font-size:10.5px;line-height:1.5;color:var(--text-muted);word-break:break-word}',
    '.ftb-sub-mark{flex-shrink:0;width:20px;text-align:center;font-size:10px;color:var(--text-muted);font-family:ui-monospace,SFMono-Regular,Consolas,monospace}',
    '.ftb-sub-mark.done{color:#6a9955}',
    '.ftb-sub-mark.doing{color:#d4a74e}',
    '.ftb-sub-mark.blocked{color:var(--danger,#e5534b)}',
    '.ftb-sub-text{flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}',
    '.ftb-sub-text.done{color:var(--text-muted);text-decoration:line-through}',
    '.ftb-sub-acts{display:flex;gap:4px;flex-shrink:0;justify-content:flex-end;opacity:.5}',
    '.ftb-sub:hover .ftb-sub-acts{opacity:1}',
    '.ftb-chip{font-size:10px;padding:0 5px;border-radius:9px;border:1px solid currentColor;color:var(--text-muted);flex-shrink:0;line-height:1.5}',
    '.ftb-chip.agent{color:#6a9955}',
    '.ftb-chip.blocked{color:var(--danger,#e5534b)}',
    // ── 运行记录 ────────────────────────────────────────────────
    '.ftb-rec{display:flex;gap:8px;align-items:center;padding:7px 8px;border-bottom:1px solid var(--border-color);font-size:12px;flex-wrap:wrap}',
    '.ftb-rec:hover{background:var(--bg-active)}',
    '.ftb-rec.layer{opacity:.66}',
    '.ftb-rec-main{flex:1;min-width:200px}',
    '.ftb-rec-id{font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:11px;color:var(--text-secondary);margin-right:6px}',
    '.ftb-rec-sub{margin-top:3px;font-size:11px;color:var(--text-muted);line-height:1.6}',
    '.ftb-rec-acts{display:flex;gap:4px;flex-shrink:0;flex-wrap:wrap;justify-content:flex-end}',
    '.ftb-danger{color:var(--danger,#e5534b)!important;border-color:var(--danger,#e5534b)!important}',
    // ── 子任务看板列（借鉴 dsh-taskboard：列头计数 / 空列占位 / 可折叠 / 拖拽落点反馈）──
    '.ftb-subcols{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:8px;margin-top:6px}',
    '.ftb-subcol{display:flex;flex-direction:column;gap:5px;border:1px solid var(--border-color);border-radius:4px;background:var(--bg-primary);padding:6px;min-width:0}',
    '.ftb-subcol.drop-on{border-style:dashed;border-color:var(--text-secondary);background:var(--bg-active)}',
    '.ftb-subcol-head{display:flex;align-items:center;gap:5px;font-size:11px;color:var(--text-secondary);min-height:18px}',
    '.ftb-subcol.collapsible .ftb-subcol-head{cursor:pointer;user-select:none}',
    '.ftb-subcol.collapsible .ftb-subcol-head:hover{color:var(--text-primary)}',
    '.ftb-subcol-count{font-size:10px;line-height:1.5;padding:0 5px;border-radius:8px;border:1px solid var(--border-color);color:var(--text-muted);font-variant-numeric:tabular-nums;flex-shrink:0}',
    '.ftb-subcol-caret{font-size:9px;color:var(--text-muted);flex-shrink:0}',
    '.ftb-subcol-body{display:flex;flex-direction:column;gap:5px;min-height:32px}',
    '.ftb-subcol.collapsed .ftb-subcol-body{display:none}',
    '.ftb-subcol-empty{border:1px dashed var(--border-color);border-radius:4px;padding:8px 6px;font-size:10.5px;line-height:1.6;color:var(--text-muted);text-align:center}',
    // ── 工具栏：计数徽标 / 搜索 / 筛选胶囊 ────────────────────────
    '.ftb-count{font-size:10.5px;line-height:1.6;padding:0 6px;border-radius:9px;border:1px solid var(--border-color);color:var(--text-muted);font-variant-numeric:tabular-nums;flex-shrink:0}',
    '.ftb-count.strong{color:var(--text-secondary)}',
    '.ftb-search{font:inherit;font-size:11px;color:var(--text-primary);background:var(--bg-primary);border:1px solid var(--border-color);border-radius:3px;padding:2px 6px;width:170px;box-sizing:border-box;flex-shrink:0}',
    '.ftb-search::placeholder{color:var(--text-muted)}',
    '.ftb-fchip{font:inherit;font-size:10.5px;line-height:1.6;padding:1px 8px;border-radius:9px;border:1px solid var(--border-color);background:transparent;color:var(--text-muted);cursor:pointer;flex-shrink:0}',
    '.ftb-fchip:hover{color:var(--text-secondary)}',
    '.ftb-fchip.active{color:var(--text-primary);background:var(--bg-active);border-color:currentColor}',
    // 焦点条元信息胶囊（把长句 meta 拆成可扫读的 chips）
    '.ftb-chips{display:flex;align-items:center;gap:5px;flex-wrap:wrap;margin-top:5px}',
    '.ftb-chipm{font-size:10.5px;line-height:1.7;padding:0 7px;border-radius:9px;background:var(--bg-active);color:var(--text-muted);font-variant-numeric:tabular-nums}',
    '.ftb-chipm.brand{color:var(--text-secondary)}',
  ].join('')

  // ── DOM 助手（一律 textContent）──────────────────────────────
  function el(tag, cls, text) {
    const n = document.createElement(tag)
    if (cls) n.className = cls
    if (text !== undefined && text !== null) n.textContent = String(text)
    return n
  }
  function btn(label, title, onClick, primary) {
    const b = el('button', 'ftb-btn' + (primary ? ' primary' : ''), label)
    b.type = 'button'
    if (title) b.title = title
    b.addEventListener('click', onClick)
    return b
  }
  function badge(cls, text, title) {
    const b = el('span', 'ftb-badge ' + cls, text)
    if (title) b.title = title
    return b
  }
  // 表单控件（D4-④）：值一律经 textContent / value 赋值，绝不拼字符串
  function field(labelText, control, hint) {
    const f = el('div', 'ftb-field')
    const l = el('label', 'ftb-label', labelText)
    if (hint) l.title = hint
    f.appendChild(l)
    f.appendChild(control)
    return f
  }
  function txtInput(value, onInput, placeholder) {
    const i = document.createElement('input')
    i.type = 'text'
    i.value = value === undefined || value === null ? '' : String(value)
    if (placeholder) i.placeholder = placeholder
    i.addEventListener('input', () => onInput(i.value))
    return i
  }
  function numInput(value, onInput, min) {
    const i = document.createElement('input')
    i.type = 'number'
    i.min = String(min === undefined ? 1 : min)
    i.value = value === undefined || value === null ? '' : String(value)
    i.addEventListener('input', () => onInput(i.value))
    return i
  }
  function checkInput(checked, labelText, onChange, hint) {
    const w = el('div', 'ftb-field')
    const i = document.createElement('input')
    i.type = 'checkbox'
    i.checked = !!checked
    i.addEventListener('change', () => onChange(!!i.checked))
    const l = el('label', 'ftb-label inline', labelText)
    if (hint) l.title = hint
    w.appendChild(i)
    w.appendChild(l)
    return w
  }
  function selInput(options, value, onChange, hint) {
    const s = document.createElement('select')
    if (hint) s.title = hint
    options.forEach((o) => {
      const op = document.createElement('option')
      op.value = o
      op.textContent = o
      if (o === value) op.selected = true
      s.appendChild(op)
    })
    s.addEventListener('change', () => onChange(s.value))
    return s
  }
  function lock(node, readOnly) {
    if (readOnly) { node.disabled = true; node.title = '预设层只读：请先「克隆到项目」' }
    return node
  }

  // ── 数据源（事实数据走持久化接口）────────────────────────────
  const coreState = () => (typeof window !== 'undefined' && window.__state) || {}
  const currentConvId = () => String(coreState().currentConvId || '')

  let snap = null
  let loadSeq = 0
  let loadedConv = ''
  let tab = 'current'
  let message = ''
  let busy = false
  // 看板内流程编辑器（D4-④）：{id, scope, version, readOnly, mode:'form'|'json', def, json, result, dirty}
  // ★ def 是唯一真相源（表单直接改它，未渲染字段原样保留 → 双向切换不丢字段）；json 是文本视图；
  //   dirty=true 表示需要重绘（结构变化/模式切换/结果更新）——编辑态下 poll 不重绘，避免打断输入。
  let editor = null
  const GATE_KINDS = ['check', 'command', 'artifact', 'approval']
  const KIND_TEXT_LABEL = {
    check: '确认说明（做什么算通过）',
    command: '说明（代跑命令验证什么）',
    artifact: '说明（产物要求）',
    approval: '审批提示语（给用户的批准理由）',
  }
  const KIND_LABEL_SHORT = { check: '确认说明', command: '说明', artifact: '说明', approval: '审批提示语' }
  function splitList(v) { return String(v === undefined || v === null ? '' : v).split(/[；;\n]/).map((x) => x.trim()).filter((x) => !!x) }
  const subs = new Set()

  function notify() { for (const fn of Array.from(subs)) { try { fn() } catch (e) { console.warn('[tool-flow] 订阅者异常', e) } } }
  function subscribe(fn) { subs.add(fn); return () => { subs.delete(fn) } }

  async function load(force) {
    const conv = currentConvId()
    if (!force && loadedConv && loadedConv !== conv) { /* 会话已切换 → 重拉 */ }
    const seq = ++loadSeq
    try {
      const res = await ui.http.get('/flow/state', { convId: conv })
      if (seq !== loadSeq) return
      snap = res || null
      loadedConv = conv
    } catch (e) {
      if (seq !== loadSeq) return
      snap = null
    }
    notify()
  }

  // ★ P4 用户感知（D4-④ / R10）：操作结果与阶段变更通知 ——
  //   优先壳内既有 window.$toast（ui-state.js），不可用时兜底写顶部消息条。
  function notifyUser(text, type) {
    const line = String(text || '').split('\n')[0].slice(0, 160)
    if (!line) return
    try {
      if (typeof window !== 'undefined' && typeof window.$toast === 'function') {
        window.$toast(line, type || 'info', 4000)
        return
      }
    } catch (e) { }
    message = line
  }

  async function rpc(payload) {
    const r = await ui.invoke(PLUGIN, 'flow_action', payload || {})
    if (r && r.ok === false) throw new Error((r && r.error) || '未知错误')
    return (r && r.result) || {}
  }

  // ── 看板内流程编辑（D4-④；一律经 RPC，壳/看板不落业务规则）──
  async function openEditor(id, opts) {
    const o = opts || {}
    if (o.blank === true) {
      // 纯表单新建（D4-④ 验收①）：给最小可用骨架，其余全靠表单增删
      const def = {
        schemaVersion: 1, id: 'my-flow', title: '新流程', version: 1,
        projectTypes: ['any'],
        stages: [{
          id: 'stage-1', title: '准备', goal: '', tasks: [],
          gates: [{ id: 'g-1-1', kind: 'check', text: '确认准备完成', required: true, blocking: false }],
        }],
      }
      editor = {
        id: def.id, scope: 'project', version: 1, readOnly: false, isNew: true,
        mode: 'form', def: def, json: JSON.stringify(def, null, 2),
        result: '新建模式：id 可改（默认 my-flow）。保存写入项目层 .pair/flows/<id>.json', dirty: true,
      }
      notify()
      return
    }
    try {
      const r0 = await rpc({ op: 'show', id: id })
      const def = (r0 && r0.def) || {}
      editor = {
        id: r0.id || id, scope: r0.scope || 'project', version: r0.version || 1, readOnly: !!r0.readOnly,
        mode: 'form', def: def, json: JSON.stringify(def, null, 2), result: '', dirty: true,
      }
      notify()
    } catch (e) { message = '打开编辑器失败：' + e; notify() }
  }
  async function cloneFlow(id) {
    try {
      const out = await rpc({ op: 'clone', id: id })
      notifyUser(out.text || '已克隆', 'success')
      await load(true); notify()
    } catch (e) { message = '克隆失败：' + e; notify() }
  }
  // 结构/模式变化 → 标记需重绘一次（字段输入不是结构变化，故不重绘、不打断输入）
  function touchEditor() { if (editor) { editor.dirty = true; notify() } }
  function setMode(next) {
    if (!editor || editor.mode === next) return
    if (next === 'json') {
      editor.json = JSON.stringify(editor.def, null, 2)
      editor.mode = 'json'
    } else {
      try {
        editor.def = JSON.parse(editor.json)
        editor.mode = 'form'
        editor.result = '已按 JSON 同步到表单（表单未渲染的字段仍保留在定义中，保存时会一并写回）'
      } catch (e) {
        editor.result = 'JSON 解析失败，未切到表单：' + ((e && e.message) || e)
        touchEditor()
        return
      }
    }
    touchEditor()
  }
  // 保存/校验前把当前模式的内容同步为 JSON 文本
  function currentJson() {
    if (!editor) return null
    if (editor.mode === 'json') {
      try { editor.def = JSON.parse(editor.json) } catch (e) {
        editor.result = 'JSON 解析失败：' + ((e && e.message) || e)
        return null
      }
      return editor.json
    }
    editor.json = JSON.stringify(editor.def, null, 2)
    return editor.json
  }
  // 表单本地校验（与 host 半 validateFlow 同规则：必填 + kind 相关必填 + 唯一性）
  function formErrors() {
    const def = (editor && editor.def) || {}
    const errs = []
    if (!def.id) errs.push('流程 id 不能为空')
    const stages = Array.isArray(def.stages) ? def.stages : []
    if (!stages.length) errs.push('至少需要 1 个阶段')
    const seenS = {}
    stages.forEach((st, i) => {
      const sl = '阶段 #' + (i + 1)
      if (!st || !st.id) errs.push(sl + ' 缺少 id')
      else if (seenS[st.id]) errs.push('阶段 id 重复：' + st.id)
      else seenS[st.id] = 1
      const gates = (st && Array.isArray(st.gates)) ? st.gates : []
      const seenG = {}
      gates.forEach((g, j) => {
        const gl = (st && st.id ? st.id : sl) + ' 的门禁 #' + (j + 1)
        if (!g || !g.id) errs.push(gl + ' 缺少 id')
        else if (seenG[g.id]) errs.push('门禁 id 重复：' + g.id)
        else seenG[g.id] = 1
        const kind = g && g.kind
        if (GATE_KINDS.indexOf(kind) < 0) errs.push((g && g.id ? g.id : gl) + '：kind 非法 ' + String(kind) + '（合法：' + GATE_KINDS.join('/') + '）')
        if (kind === 'command' && !String((g && g.cmd) || '').trim()) errs.push('command 门禁 ' + ((g && g.id) || gl) + ' 必须填「命令 cmd」')
        if (kind === 'artifact' && !String((g && g.artifact) || '').trim()) errs.push('artifact 门禁 ' + ((g && g.id) || gl) + ' 必须填「产物路径」')
      })
    })
    return errs
  }
  async function editorValidate() {
    if (!editor) return
    if (editor.mode === 'form') {
      const local = formErrors()
      if (local.length) { editor.result = '表单校验未通过（' + local.length + ' 项）：' + local.join('；'); touchEditor(); return }
    }
    const json = currentJson()
    if (!json) { touchEditor(); return }
    try {
      const v = await rpc({ op: 'validate', json: json })
      editor.result = (v && v.ok) ? '校验通过（提交前会再经 host 侧 flow_validate）' : ('校验未通过：' + ((v && v.errors) || []).join('；'))
    } catch (e) { editor.result = '校验失败：' + e }
    touchEditor()
  }
  async function editorSave() {
    if (!editor || editor.readOnly) return
    if (editor.isNew) {
      const nid = String((editor.def && editor.def.id) || '').trim()
      if (!nid) { editor.result = '新流程必须填「流程 id」'; touchEditor(); return }
      const lib = (snap && snap.library) || []
      if (lib.some((f) => f.id === nid && f.scope === 'project')) {
        editor.result = '项目层已存在同名流程：' + nid + '（请换一个 id，或取消后对该流程用「编辑」）'
        touchEditor()
        return
      }
    }
    if (editor.mode === 'form') {
      const local = formErrors()
      if (local.length) { editor.result = '表单校验未通过（' + local.length + ' 项）：' + local.join('；'); touchEditor(); return }
    }
    const json = currentJson()
    if (!json) { touchEditor(); return }
    try {
      const v = await rpc({ op: 'validate', json: json })
      if (!v || v.ok !== true) { editor.result = '校验未通过：' + ((v && v.errors) || ['未知原因']).join('；'); touchEditor(); return }
      // 新建不带 expectedVersion（目标尚不存在）；编辑带乐观锁
      const out = await rpc({ op: 'define', json: json, scope: editor.scope, expectedVersion: editor.isNew ? undefined : editor.version })
      editor = null
      notifyUser(out.text || '已保存', 'success')
      await load(true); notify()
    } catch (e) { editor.result = '保存失败：' + e; touchEditor() }
  }

  async function act(payload, tip) {
    if (busy) return
    busy = true
    message = tip || '正在执行…'
    notify()
    try {
      const r = await ui.invoke(PLUGIN, 'flow_action', payload || {})
      if (r && r.ok === false) message = '失败：' + (r.error || '未知错误')
      else {
        const out = (r && r.result) || {}
        message = out.text || (tip || '已执行')
        if (out.advanced) notifyUser(out.completed ? '流程已全部完成' : ('已进入阶段：' + (out.stage || out.title || '')), 'success')
        else if (out.gate) notifyUser(out.gate.id + ' → ' + out.gate.status, (out.gate.status === 'pass' || out.gate.status === 'approved') ? 'success' : 'warn')
      }
    } catch (e) {
      message = '调用失败：' + e
    }
    busy = false
    await load(true)
    notify()
  }

  // 子任务 / 生命周期：看板快捷通道（统一经 RPC，host 半是唯一真相）
  function actTask(payload, tip) { return act(Object.assign({ op: 'task' }, payload || {}), tip) }
  function actLife(op, extra, tip) { return act(Object.assign({ op: op }, extra || {}), tip) }

  const STATUS_LABEL = { pass: 'PASS', approved: 'APPROVED', pending: 'PENDING', fail: 'FAIL', todo: 'TODO' }
  const STAGE_LABEL = { completed: 'DONE', in_progress: 'ACTIVE', pending: 'TODO', cancelled: 'CANCELLED' }
  // 实例生命周期 / 子任务：徽章与文案（与 host 半 RUN_STATUS / SUB_STATUS 一致）
  const RUN_LABEL = { active: '进行中', completed: '已完成', interrupted: '已中断', archived: '已归档', deleted: '已删除' }
  const RUN_BADGE = { active: 'active', completed: 'done', interrupted: 'pending', archived: 'todo', deleted: 'fail' }
  const SUB_LABEL = { pending: '待办', in_progress: '进行中', completed: '已完成', blocked: '阻塞', cancelled: '已取消' }
  function subMark(s) {
    if (s === 'completed') return '[x]'
    if (s === 'in_progress') return '[>]'
    if (s === 'blocked') return '[!]'
    if (s === 'cancelled') return '[-]'
    return '[ ]'
  }
  function subMarkCls(s) {
    if (s === 'completed') return 'done'
    if (s === 'in_progress') return 'doing'
    if (s === 'blocked') return 'blocked'
    return ''
  }
  function shortTime(iso) {
    if (!iso) return '—'
    try {
      const d = new Date(iso)
      if (isNaN(d.getTime())) return String(iso).slice(0, 16)
      const p = (n) => (n < 10 ? '0' + n : String(n))
      return p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    } catch (e) { return String(iso).slice(0, 16) }
  }
  function gateBadge(g) {
    const s = g.status || 'todo'
    const label = STATUS_LABEL[s] || String(s).toUpperCase()
    const extra = s === 'approved' && g.by === 'user' ? '·用户' : ''
    return badge(s, label + extra, (g.summary || '') + (g.at ? ('\n' + g.at) : ''))
  }

  // ── 页签 1：当前流程 ─────────────────────────────────────────
  function renderCurrent(body) {
    const run = snap && snap.run
    if (!run) {
      const box = el('div', 'ftb-empty')
      box.appendChild(el('div', null, '当前没有进行中的流程实例'))
      const rec = snap && snap.recommend
      if (rec) {
        box.appendChild(el('div', null, '项目类型识别：' + ((rec.types || []).join(', ') || '（无信号）')))
        box.appendChild(el('div', null, '推荐流程：' + rec.flowId + '（' + (rec.flowTitle || '') + '）'))
        box.appendChild(el('div', null, '依据：' + (rec.reason || '')))
        box.appendChild(btn('启动推荐流程', '调 flow_action start', () => act({ op: 'start', flowId: rec.flowId, convId: currentConvId() }, '启动 ' + rec.flowId + '…'), true))
      } else {
        box.appendChild(el('div', null, '可在「流程库」页签选择流程启动，或用对话里的 flow_start 工具启动。'))
      }
      body.appendChild(box)
      return
    }
    const pg = run.progress || {}
    const total = pg.stagesTotal || 1
    const stages = snap.stages || []
    const curIdx = typeof pg.stageIdx === 'number' ? pg.stageIdx : 0
    const curStage = stages[curIdx] || null
    const subs = pg.subs || { label: '0/0', doing: 0, blocked: 0 }
    const focus = run.focus || null
    const gs0 = (curStage && curStage.gates) || []
    const gateLeft = gs0.filter(g => !(g.status === 'pass' || g.status === 'approved'))

    // ── ① 焦点卡：此刻在哪 / 在做什么 / 卡在哪 + 操作入口 ──────────
    const fc = el('div', 'ftb-focus' + (run.status === 'completed' ? ' done' : (gateLeft.length ? ' err' : '')))
    const ft = el('div', 'ftb-focus-title')
    ft.appendChild(el('span', null, '第 ' + (curIdx + 1) + '/' + total + ' 阶段 · ' + ((curStage && curStage.title) || run.currentStage || '（已完成）')))
    ft.appendChild(badge(RUN_BADGE[run.status] || 'todo', RUN_LABEL[run.status] || run.status))
    fc.appendChild(ft)

    const pbar = el('div', 'ftb-pbar')
    const pfill = el('div', 'ftb-pbar-fill')
    pfill.style.width = Math.max(0, Math.min(100, pg.percent || 0)) + '%'
    pbar.appendChild(pfill)
    fc.appendChild(pbar)

    // 元信息胶囊：把原来的一行长句拆成可扫读的 chips（借鉴 dsh-taskboard 的 chips 行）
    const chips = el('div', 'ftb-chips')
    chips.appendChild(el('span', 'ftb-chipm brand', '总进度 ' + (pg.percent || 0) + '%'))
    chips.appendChild(el('span', 'ftb-chipm', '阶段 ' + (pg.stagesDone || 0) + '/' + total + ' 已完成'))
    chips.appendChild(el('span', 'ftb-chipm', '子任务 ' + subs.label +
      (subs.doing ? (' · 进行中 ' + subs.doing) : '') + (subs.blocked ? (' · 阻塞 ' + subs.blocked) : '')))
    chips.appendChild(el('span', 'ftb-chipm', '实例 ' + run.runId))
    if (run.convId) chips.appendChild(el('span', 'ftb-chipm', '会话 ' + run.convId))
    if (run.gitHead) chips.appendChild(el('span', 'ftb-chipm', 'git ' + String(run.gitHead).slice(0, 8)))
    if ((run.modifiedFiles || []).length) chips.appendChild(el('span', 'ftb-chipm', '已改动 ' + run.modifiedFiles.length + ' 文件'))
    fc.appendChild(chips)

    if (run.status === 'completed') {
      fc.appendChild(el('div', 'ftb-focus-now', '全部阶段已完成 —— 可在「运行记录」页签归档或删除该记录。'))
    } else {
      const now = el('div', 'ftb-focus-now')
      if (focus) {
        now.appendChild(el('span', 'ftb-sub-mark ' + subMarkCls(focus.status), subMark(focus.status)))
        now.appendChild(el('span', null, '当前子任务 ' + focus.id + '：' + focus.subject))
        if (focus.note) now.appendChild(el('span', 'ftb-cell', '（' + focus.note + '）'))
        if (run.status === 'active' && focus.status !== 'completed') {
          now.appendChild(btn('开始', '标记该子任务为进行中', () => actTask({ action: 'update', id: focus.id, status: 'in_progress' }, '开始子任务…')))
          now.appendChild(btn('完成', '标记该子任务已完成', () => actTask({ action: 'update', id: focus.id, status: 'completed' }, '完成子任务…'), true))
        }
      } else if (curStage && (curStage.subtasks || []).length) {
        now.appendChild(el('span', null, '本阶段子任务已全部处理'))
      } else {
        now.appendChild(el('span', null, '本阶段还没有子任务 —— 可在阶段详情里「＋ 拆解子任务」把活拆细'))
      }
      fc.appendChild(now)
      fc.appendChild(el('div', 'ftb-focus-meta', gateLeft.length
        ? ('卡点：' + gateLeft.length + ' 个门禁未通过（' + gateLeft.map(g => g.id).join('、') + '）')
        : '本阶段门禁已清'))
    }

    const facs = el('div', 'ftb-focus-acts')
    if (run.status === 'active') {
      facs.appendChild(btn('推进阶段', '检查本阶段必过门禁；全部通过则进入下一阶段', () => act({ op: 'advance', runId: run.runId }, '推进阶段…'), true))
      facs.appendChild(btn('中断', '暂停该流程：进度与证据全留，可随时恢复', () => {
        if (!window.confirm('中断流程 ' + run.runId + '？\n进度会完整保留，可随时「恢复运行」。')) return
        actLife('interrupt', { runId: run.runId }, '中断流程…')
      }))
    }
    if (run.status === 'interrupted') {
      facs.appendChild(btn('恢复运行', '从断点继续（进度全留）', () => actLife('resume', { runId: run.runId }, '恢复流程…'), true))
    }
    if (run.status !== 'archived' && run.status !== 'deleted') {
      facs.appendChild(btn('归档', '移出活跃列表、只读留档（记录与证据保留，可取消归档）', () => {
        if (!window.confirm('归档流程 ' + run.runId + '？\n会移出活跃列表，可在「运行记录」里取消归档。')) return
        actLife('archive', { runId: run.runId }, '归档流程…')
      }))
    }
    if (run.status !== 'deleted') {
      facs.appendChild(btn('删除记录', '软删除：从流程视图隐藏，可在「运行记录」里恢复', () => {
        if (!window.confirm('删除流程记录 ' + run.runId + '？\n这是软删除（可在「运行记录」恢复）；物理删除请用「运行记录」里的「彻底删除」。')) return
        actLife('delete', { runId: run.runId }, '删除记录…')
      }))
    }
    if (run.status === 'archived' || run.status === 'deleted') facs.appendChild(el('span', 'ftb-cell', '（只读留档：子任务与门禁不可改）'))
    fc.appendChild(facs)
    body.appendChild(fc)

    // ── ② 阶段时间线：逐级看清「走到哪一步、每步还剩什么」──────────
    const tl = el('div', 'ftb-tl')
    stages.forEach((st, i) => {
      const isCur = i === curIdx && run.status !== 'completed' && run.status !== 'archived' && run.status !== 'deleted'
      const stDone = st.status === 'completed'
      const node = el('div', 'ftb-node' + (isCur ? ' cur' : ''))
      const rail = el('div', 'ftb-node-rail')
      rail.appendChild(el('div', 'ftb-dot ' + (stDone ? 'done' : (isCur ? 'active' : 'todo'))))
      if (i < stages.length - 1) rail.appendChild(el('div', 'ftb-line' + (stDone ? ' done' : '')))
      node.appendChild(rail)

      const nb = el('div', 'ftb-node-body')
      const nh = el('div', 'ftb-node-head')
      nh.appendChild(el('span', 'ftb-idx', (i + 1) + '.'))
      nh.appendChild(el('span', 'ftb-node-title', st.title || st.id))
      nh.appendChild(badge(stDone ? 'done' : (isCur ? 'active' : 'todo'), STAGE_LABEL[st.status] || st.status))
      const ss = st.subStats || { total: 0, label: '0/0', doing: 0, blocked: 0 }
      const gs = st.gates || []
      const gPass = gs.filter(g => (g.status === 'pass' || g.status === 'approved')).length
      if (ss.total) nh.appendChild(el('span', 'ftb-cell', '子任务 ' + ss.label + (ss.doing ? (' · 进行中' + ss.doing) : '') + (ss.blocked ? (' · 阻塞' + ss.blocked) : '')))
      if (gs.length) nh.appendChild(el('span', 'ftb-cell', '门禁 ' + gPass + '/' + gs.length))
      if (st.completedAt) nh.appendChild(el('span', 'ftb-cell', '完成 ' + shortTime(st.completedAt)))
      // 节点头必须在 node-body 内：node 是 flex 行（rail + body），
      // 若 head 直接挂到 node 上，它会以内容宽度独占空间，把 body（flex:1）压到几十像素，
      // 三列子任务看板随之塌缩成逐字竖排（真实浏览器实测 width 21px/列）。
      nb.appendChild(nh)

      // 看板常显（2026-10-17 用户实测反馈）：此前本块仅在「有子任务或门禁」时才渲染，
      // 空阶段只剩一句提示、三列看板整体消失 —— 用户完全看不到看板。
      // dsh-taskboard 的列同样常显（空列给指引性占位），此处对齐。
      if (ss.total || gs.length || isCur) {
        const det = document.createElement('details')
        det.className = 'ftb-more'
        det.open = isCur
        det.appendChild(el('summary', null, '子任务 ' + ss.label + ' · 门禁 ' + gPass + '/' + gs.length))
        const card = el('div', 'ftb-card')
        renderSubs(card, st, run)
        renderGates(card, st, run)
        det.appendChild(card)
        nb.appendChild(det)
      }
      node.appendChild(nb)
      tl.appendChild(node)
    })
    body.appendChild(tl)
  }

  // 子任务区：状态可见 + 一键推进（开始/完成/移除/拆解）
  // ── 子任务看板（三列：待办 / 进行中 / 已完成）────────────────────
  // 设计借鉴 dsh-taskboard（github.com/cloader/dsh-taskboard）：
  //   · 列头带计数徽标、空列有指引性占位文案
  //   · 卡片左侧状态色条、卡片式承载、悬停才显操作
  //   · 列间**拖拽改状态**，落点虚线反馈
  // 适配判断：流程「阶段」是严格串行的，不能照搬五列状态看板（顺序由上方时间线唯一表达）；
  //   但「同一阶段内的子任务」彼此无先后依赖，因此列间拖拽改状态是安全的、也是最快的操作路径。
  const SUB_COLS = [
    { key: 'todo', label: '待办', status: 'pending', empty: '没有待办子任务 —— 点「＋ 拆解子任务」，或让 agent 用 flow_task 拆解', match: (s) => s.status !== 'in_progress' && s.status !== 'completed' },
    { key: 'doing', label: '进行中', status: 'in_progress', empty: '没有进行中的子任务 —— 点「开始」，或把待办卡片拖到这里', match: (s) => s.status === 'in_progress' },
    { key: 'done', label: '已完成', status: 'completed', empty: '还没有完成的子任务', match: (s) => s.status === 'completed' },
  ]
  const SUB_COL_HINT = { todo: '拖到这里 = 退回待办', doing: '拖到这里 = 标记进行中', done: '拖到这里 = 标记已完成' }
  const SUB_CARD_CLS = { pending: 's-pending', in_progress: 's-doing', completed: 's-done', blocked: 's-blocked' }
  const collapsedCols = { done: true }   // 「已完成」列默认收起：长流程不淹没当前在做的事
  let dragSub = null                     // 正在拖拽的子任务（不用 dataTransfer：无浏览器环境同样可断言）

  /** 单张子任务卡：状态色条 + 标题行 + 备注行 + 悬停操作行。 */
  function subCard(s, st, run, ro) {
    const node = el('div', 'ftb-sub ' + (SUB_CARD_CLS[s.status] || 's-pending'))
    const r1 = el('div', 'ftb-sub-row')
    r1.appendChild(el('span', 'ftb-sub-mark ' + subMarkCls(s.status), subMark(s.status)))
    const tx = el('span', 'ftb-sub-text' + (s.status === 'completed' ? ' done' : ''), s.id + ' ' + s.subject)
    tx.title = [s.detail, s.note ? ('备注：' + s.note) : '', s.evidence ? ('证据：' + s.evidence) : '', s.startedAt ? ('开始：' + shortTime(s.startedAt)) : '', s.completedAt ? ('完成：' + shortTime(s.completedAt)) : '']
      .filter(x => !!x).join('\n') || s.subject
    r1.appendChild(tx)
    if (s.status === 'blocked') r1.appendChild(el('span', 'ftb-chip blocked', '阻塞'))
    if (s.source === 'agent') r1.appendChild(el('span', 'ftb-chip agent', 'agent 拆解'))
    node.appendChild(r1)
    if (s.note) node.appendChild(el('div', 'ftb-sub-note', '备注：' + s.note))
    if (!ro) {
      node.draggable = true
      node.title = '可拖到其他列直接改状态'
      node.addEventListener('dragstart', () => { dragSub = { stage: st.id, id: s.id }; node.className += ' dragging' })
      node.addEventListener('dragend', () => { dragSub = null; node.className = node.className.replace(' dragging', '') })
      const acts = el('div', 'ftb-sub-acts')
      if (s.status !== 'in_progress' && s.status !== 'completed') acts.appendChild(btn('开始', '标记为进行中', () => actTask({ action: 'update', stage: st.id, id: s.id, status: 'in_progress' }, '更新子任务…')))
      if (s.status !== 'completed') acts.appendChild(btn('完成', '标记为已完成', () => actTask({ action: 'update', stage: st.id, id: s.id, status: 'completed' }, '更新子任务…')))
      else acts.appendChild(btn('重开', '退回进行中', () => actTask({ action: 'update', stage: st.id, id: s.id, status: 'in_progress' }, '更新子任务…')))
      acts.appendChild(btn('备注', '记录卡点/备注', () => {
        const n = window.prompt('备注（如卡在哪里）：', s.note || '')
        if (n === null) return
        actTask({ action: 'update', stage: st.id, id: s.id, note: n }, '更新备注…')
      }))
      acts.appendChild(btn('移除', '从本阶段移除该子任务（宿主任务标记为取消）', () => {
        if (!window.confirm('移除子任务 ' + s.id + '：' + s.subject + '？')) return
        actTask({ action: 'remove', stage: st.id, id: s.id }, '移除子任务…')
      }))
      node.appendChild(acts)
    }
    return node
  }

  // 子任务区：三列看板（列头计数 / 空列占位 / 拖拽改状态）
  function renderSubs(card, st, run) {
    const list = st.subtasks || []
    const ro = run.status !== 'active' && run.status !== 'interrupted'
    const head = el('div', 'ftb-block-head')
    head.appendChild(el('span', 'ftb-name', '子任务'))
    const doneN = list.filter(c => c.status === 'completed').length
    head.appendChild(el('span', 'ftb-count', list.length ? (doneN + '/' + list.length + ' 已完成') : '无'))
    head.appendChild(el('div', 'ftb-spacer'))
    if (!ro) {
      head.appendChild(btn('＋ 拆解子任务', '在当前阶段新增一条子任务（agent 也可用 flow_task 工具拆解）', () => {
        const t = window.prompt('子任务（做到什么算完成）：')
        if (!t || !t.trim()) return
        actTask({ action: 'add', stage: st.id, subject: t.trim() }, '拆解子任务…')
      }, true))
    }
    card.appendChild(head)
    if (!list.length) {
      // 无子任务也照样渲染三列（空列各有指引性占位）——看板不再整体消失，用户始终看得到它。
      card.appendChild(el('div', 'ftb-note', '本阶段暂无子任务 —— 点右上「＋ 拆解子任务」，或让 agent 用 flow_task {op:"add"} 拆解。'))
    }
    const cols = el('div', 'ftb-subcols')
    SUB_COLS.forEach(c => {
      const items = list.filter(c.match)
      const collapsible = c.key === 'done' && items.length > 0
      const col = el('div', 'ftb-subcol' + (collapsible ? ' collapsible' : '') + (collapsible && collapsedCols.done ? ' collapsed' : ''))
      const ch = el('div', 'ftb-subcol-head')
      if (collapsible) ch.appendChild(el('span', 'ftb-subcol-caret', collapsedCols.done ? '▸' : '▾'))
      ch.appendChild(el('span', null, c.label))
      ch.appendChild(el('span', 'ftb-subcol-count', String(items.length)))
      if (collapsible) {
        ch.title = collapsedCols.done ? '展开「已完成」列' : '收起「已完成」列'
        ch.addEventListener('click', () => { collapsedCols.done = !collapsedCols.done; notify() })
      }
      col.appendChild(ch)
      const bodyEl = el('div', 'ftb-subcol-body')
      if (!items.length) bodyEl.appendChild(el('div', 'ftb-subcol-empty', c.empty))
      items.forEach(s => bodyEl.appendChild(subCard(s, st, run, ro)))
      col.appendChild(bodyEl)
      if (!ro) {
        col.title = SUB_COL_HINT[c.key] || ''
        col.addEventListener('dragover', (e) => {
          if (e && typeof e.preventDefault === 'function') e.preventDefault()
          if (dragSub && col.className.indexOf('drop-on') < 0) col.className += ' drop-on'
        })
        col.addEventListener('dragleave', () => { col.className = col.className.replace(' drop-on', '') })
        col.addEventListener('drop', (e) => {
          if (e && typeof e.preventDefault === 'function') e.preventDefault()
          col.className = col.className.replace(' drop-on', '')
          const d = dragSub
          dragSub = null
          if (!d) return
          const src = (st.subtasks || []).filter(x => x.id === d.id)[0]
          if (!src || src.status === c.status) return   // 同列放下：无变化，不发无意义写入
          actTask({ action: 'update', stage: d.stage, id: d.id, status: c.status }, '把 ' + d.id + ' 移到「' + c.label + '」…')
        })
      }
      cols.appendChild(col)
    })
    card.appendChild(cols)
  }

  // 门禁区：状态 + 过门禁动作（只读留档态不提供动作）
  function renderGates(card, st, run) {
    ;(st.gates || []).forEach(g => {
      const row = el('div', 'ftb-gate')
      row.appendChild(el('span', 'ftb-gate-id', g.id))
      row.appendChild(el('span', 'ftb-gate-kind', g.kind + (g.required ? '·必过' : '') + (g.blocking ? '·阻塞' : '')))
      row.appendChild(el('span', 'ftb-gate-text', g.text || ''))
      row.appendChild(gateBadge(g))

      const isPassed = g.status === 'pass' || g.status === 'approved'
      if (!isPassed && run.status === 'active') {
        if (g.kind === 'command') {
          row.appendChild(btn('代跑', '执行 ' + (g.cmd || '') + ' 并留证' + (g.cmd ? '' : '（未声明 cmd）'), () => {
            act({ op: 'advance', runId: run.runId, gateId: g.id }, '执行门禁 ' + g.id + '…')
          }, true))
        } else if (g.kind === 'approval') {
          row.appendChild(btn('批准', '审批卡点：由用户显式批准（会记 approvedBy=user）', () => {
            if (!window.confirm('确认批准门禁 ' + g.id + '？\n' + (g.text || ''))) return
            act({ op: 'approve', runId: run.runId, gateId: g.id, approve: true, note: '看板批准' }, '批准 ' + g.id + '…')
          }, true))
        } else {
          const inp = el('input', 'ftb-input')
          inp.type = 'text'
          inp.placeholder = g.kind === 'artifact' ? ('产物路径（默认 ' + (g.artifact || '') + '）') : '证据说明（如何确认成立）'
          inp.value = g.kind === 'artifact' ? (g.artifact || '') : ''
          row.appendChild(inp)
          row.appendChild(btn('提交', '以 evidence 记录该门禁', () => {
            act({ op: 'advance', runId: run.runId, gateId: g.id, evidence: inp.value }, '记录门禁 ' + g.id + '…')
          }))
        }
      }
      card.appendChild(row)

      if (g.summary || g.evidenceFile || g.at) {
        const d = document.createElement('details')
        d.className = 'ftb-more'
        d.appendChild(el('summary', null, '证据（' + (g.at || '未记录') + '）'))
        d.appendChild(el('pre', 'ftb-pre', (g.summary || '') + (g.evidenceFile ? ('\n文件：' + g.evidenceFile) : '') + (g.by ? ('\n批准人：' + g.by + (g.note ? ('（' + g.note + '）') : '')) : '')))
        card.appendChild(d)
      }
    })
  }

  // ── 页签 2：运行记录（全部实例 + 生命周期操作）───────────────
  // 运行记录筛选：状态胶囊（各自带计数）+ 关键词搜索（借鉴 dsh-taskboard 工具栏）
  const RUN_FILTERS = [
    { key: 'all', label: '全部', hint: '不筛选状态', match: () => true },
    { key: 'active', label: '进行中', hint: '活跃实例：可推进 / 中断', match: (r) => r.status === 'active' },
    { key: 'completed', label: '已完成', hint: '全部阶段已完成', match: (r) => r.status === 'completed' },
    { key: 'interrupted', label: '已中断', hint: '可恢复暂停（进度全留）', match: (r) => r.status === 'interrupted' },
    { key: 'archived', label: '已归档', hint: '只读留档', match: (r) => r.status === 'archived' },
    { key: 'deleted', label: '已删除', hint: '软删，可恢复', match: (r) => r.status === 'deleted' },
  ]
  let runsFilter = 'all'
  let runsQuery = ''

  function renderRuns(body) {
    const all = (snap && snap.runs) || []
    body.appendChild(el('div', 'ftb-note',
      '全部流程实例（含已归档 / 已删除）。语义：中断=可恢复暂停（进度全留）；归档=移出活跃列表、只读留档；删除=软删（可恢复）；彻底删除=物理删除实例与证据文件（不可恢复）。'))
    const bar = el('div', 'ftb-toolbar')
    bar.appendChild(btn('刷新记录', '重新拉取 GET /api/flow/state', () => load(true), true))
    const inp = el('input', 'ftb-search')
    inp.type = 'text'
    inp.placeholder = '搜索 runId / 流程 / 项目…'
    inp.title = '按 runId / 流程名 / 项目路径过滤'
    inp.value = runsQuery
    bar.appendChild(inp)
    const chipNodes = []
    RUN_FILTERS.forEach(f => {
      const n = f.key === 'all' ? all.length : all.filter(f.match).length
      const c = el('button', 'ftb-fchip' + (runsFilter === f.key ? ' active' : ''), f.label + ' ' + n)
      c.type = 'button'
      c.title = f.hint
      c.addEventListener('click', () => {
        runsFilter = f.key
        chipNodes.forEach(x => { x.node.className = 'ftb-fchip' + (x.key === f.key ? ' active' : '') })
        paint()
      })
      chipNodes.push({ key: f.key, node: c })
      bar.appendChild(c)
    })
    bar.appendChild(el('div', 'ftb-spacer'))
    bar.appendChild(el('span', 'ftb-count strong', '共 ' + all.length + ' 条'))
    body.appendChild(bar)
    const listEl = el('div', 'ftb-runs-list')
    body.appendChild(listEl)

    // ★ 局部重绘：搜索 / 筛选只刷新列表，不重建工具栏 —— 输入时保住焦点与光标
    const paint = () => {
      listEl.textContent = ''
      const fdef = RUN_FILTERS.filter(x => x.key === runsFilter)[0] || RUN_FILTERS[0]
      const q = runsQuery.trim().toLowerCase()
      const recs = all.filter(fdef.match).filter(f => !q ||
        [f.runId, f.flowId, f.title, f.project].filter(Boolean).join(' ').toLowerCase().indexOf(q) >= 0)
      if (!all.length) {
        listEl.appendChild(el('div', 'ftb-empty', '还没有任何流程实例 —— 在「当前流程」或「流程库」页签启动一个，或用对话里的 flow_start 工具。'))
        return
      }
      if (!recs.length) {
        listEl.appendChild(el('div', 'ftb-empty', '没有匹配的实例 —— 换个关键词，或点「全部」清除状态筛选。'))
        return
      }
      recs.forEach(rc => {
      const layered = rc.status === 'archived' || rc.status === 'deleted'
      const row = el('div', 'ftb-rec' + (layered ? ' layer' : ''))
      const main = el('div', 'ftb-rec-main')
      const l1 = el('div')
      l1.appendChild(el('span', 'ftb-rec-id', rc.runId))
      l1.appendChild(badge(RUN_BADGE[rc.status] || 'todo', RUN_LABEL[rc.status] || rc.status))
      main.appendChild(l1)
      main.appendChild(el('div', 'ftb-rec-sub',
        (rc.title || rc.flowId) + ' · 阶段 ' + (rc.stageLabel || '-') + '（' + (rc.stagesDone || 0) + '/' + (rc.stagesTotal || 0) + ' 完成）· 子任务 ' +
        ((rc.subs && rc.subs.label) || '0/0') + ' · 总进度 ' + (rc.percent || 0) + '%'))
      main.appendChild(el('div', 'ftb-rec-sub',
        '更新 ' + shortTime(rc.updatedAt) + (rc.project ? (' · ' + rc.project) : '') +
        (rc.status === 'interrupted' ? (' · 中断于 ' + shortTime(rc.interruptedAt) + (rc.interruptReason ? ('（' + rc.interruptReason + '）') : '')) : '') +
        (rc.status === 'archived' ? (' · 归档于 ' + shortTime(rc.archivedAt)) : '') +
        (rc.status === 'deleted' ? (' · 删除于 ' + shortTime(rc.deletedAt)) : '')))
      row.appendChild(main)

      const mp = el('div', 'ftb-pbar')
      mp.style.width = '64px'
      mp.style.margin = '0'
      const mf = el('div', 'ftb-pbar-fill')
      mf.style.width = Math.max(0, Math.min(100, rc.percent || 0)) + '%'
      mp.appendChild(mf)
      row.appendChild(mp)

      const acts = el('div', 'ftb-rec-acts')
      if (rc.status === 'interrupted') acts.appendChild(btn('恢复', '从断点继续（可恢复暂停）', () => actLife('resume', { runId: rc.runId }, '恢复流程…'), true))
      if (rc.status === 'archived') acts.appendChild(btn('取消归档', '恢复归档前状态', () => actLife('unarchive', { runId: rc.runId }, '取消归档…'), true))
      if (rc.status === 'deleted') acts.appendChild(btn('恢复', '撤销删除（回到删除前状态）', () => actLife('restore', { runId: rc.runId }, '恢复记录…'), true))
      if (rc.status === 'active') {
        acts.appendChild(btn('中断', '暂停该流程（进度保留）', () => {
          if (!window.confirm('中断流程 ' + rc.runId + '？')) return
          actLife('interrupt', { runId: rc.runId }, '中断流程…')
        }))
      }
      if (!layered) {
        acts.appendChild(btn('归档', '移出活跃列表、只读留档', () => {
          if (!window.confirm('归档 ' + rc.runId + '？\n可在本页「取消归档」。')) return
          actLife('archive', { runId: rc.runId }, '归档…')
        }))
      }
      if (rc.status !== 'deleted') {
        acts.appendChild(btn('删除', '软删除（可在本页恢复）', () => {
          if (!window.confirm('删除记录 ' + rc.runId + '？\n软删除：可在本页「恢复」。')) return
          actLife('delete', { runId: rc.runId }, '删除记录…')
        }))
      }
      if (layered) {
        acts.appendChild(btn('彻底删除', '物理删除实例文件与证据目录（不可恢复），需输入 runId 确认', () => {
          if (!window.confirm('彻底删除 ' + rc.runId + '？\n会物理删除实例文件与证据目录，不可恢复！')) return
          const t = window.prompt('请输入 runId 以确认彻底删除：', '')
          if (t === null) return
          if (t.trim() !== rc.runId) { message = '已取消：输入的 runId 与目标不一致'; notify(); return }
          actLife('purge', { runId: rc.runId, confirm: true }, '彻底删除…')
        }))
      }
      row.appendChild(acts)
      listEl.appendChild(row)
      })
    }
    inp.addEventListener('input', () => { runsQuery = inp.value; paint() })
    inp.addEventListener('change', () => { runsQuery = inp.value; paint() })
    paint()
  }

  // ── 页签 2：项目 × 流程矩阵 ──────────────────────────────────
  function renderMatrix(body) {
    const rows = (snap && snap.matrix) || []
    body.appendChild(el('div', 'ftb-note', '来自 .pair/flows/_binding.json 与各 run；绑定变更只影响后续新建实例，不清空既有进度。'))
    if (!rows.length) { body.appendChild(el('div', 'ftb-empty', '没有可用项目根')) ; return }
    const head = el('div', 'ftb-row')
    head.appendChild(el('span', 'ftb-cell h wide', '项目'))
    head.appendChild(el('span', 'ftb-cell h', '绑定流程'))
    head.appendChild(el('span', 'ftb-cell h', '实例'))
    head.appendChild(el('span', 'ftb-cell h', '阶段'))
    head.appendChild(el('span', 'ftb-cell h', '阻塞'))
    body.appendChild(head)
    rows.forEach(r => {
      const row = el('div', 'ftb-row')
      const p = el('span', 'ftb-cell wide', r.project)
      p.title = r.project
      row.appendChild(p)
      row.appendChild(el('span', 'ftb-cell', r.flowId ? (r.flowId + (r.scope ? '（' + r.scope + '）' : '')) : '未接入'))
      row.appendChild(el('span', 'ftb-cell', r.runId || '-'))
      row.appendChild(el('span', 'ftb-cell', r.stage || '-'))
      row.appendChild(r.blocked ? badge('fail', String(r.blocked)) : el('span', 'ftb-cell', '0'))
      if (r.flowId && !r.runId) {
        row.appendChild(btn('启动', '按该项目绑定启动流程实例', () => act({ op: 'start' }, '启动流程…')))
      }
      if (r.flowId) {
        row.appendChild(btn('解绑', '解除项目与流程的绑定', () => act({ op: 'detach', project: r.project }, '解绑 ' + r.project + '…')))
      }
      body.appendChild(row)
    })
  }

  // ── 页签 3：流程库（三层）────────────────────────────────────
  function renderLibrary(body) {
    if (editor) { renderEditor(body); return }
    const list = (snap && snap.library) || []
    const bar = el('div', 'ftb-toolbar')
    bar.appendChild(btn('＋ 新建流程', '用字段级表单从空白开始新建一个流程（保存到项目层 .pair/flows/）', () => openEditor('', { blank: true }), true))
    bar.appendChild(el('span', 'ftb-count strong', list.length + ' 个流程'))
    bar.appendChild(el('span', 'ftb-note', '预设层流程只读 —— 要改先「克隆到项目」；已有流程点「编辑」（表单 / JSON 双模式）。'))
    body.appendChild(bar)
    body.appendChild(el('div', 'ftb-note', '三层：项目（.pair/flows/，可写）> 全局（<InstallDir>/.pair/flowlib/，可写）> 预设（插件包 assets/flows/，只读）。同名上层覆盖下层；升级不回冲，需显式 flow_sync。'))
    if (!list.length) { body.appendChild(el('div', 'ftb-empty', '流程库为空')); return }
    const head = el('div', 'ftb-row')
    head.appendChild(el('span', 'ftb-cell h wide', '流程'))
    head.appendChild(el('span', 'ftb-cell h', '层'))
    head.appendChild(el('span', 'ftb-cell h', '阶段'))
    head.appendChild(el('span', 'ftb-cell h', '适用类型'))
    body.appendChild(head)
    list.forEach(f => {
      const row = el('div', 'ftb-row')
      const name = el('span', 'ftb-cell wide', f.id + '「' + f.title + '」v' + f.version)
      name.title = f.id + '\n' + (f.projectTypes || []).join(', ')
      row.appendChild(name)
      row.appendChild(el('span', 'ftb-cell', f.scope + (f.readOnly ? '(只读)' : '')))
      row.appendChild(el('span', 'ftb-cell', String(f.stages)))
      row.appendChild(el('span', 'ftb-cell', (f.projectTypes || ['any']).join(',')))
      row.appendChild(btn('启动', '启动该流程的新实例（会记录 git 基线并建任务）', () => act({ op: 'start', flowId: f.id, convId: currentConvId() }, '启动 ' + f.id + '…')))
      row.appendChild(btn('绑定到当前项目', '把该流程绑定到主工作区根', () => act({ op: 'attach', flowId: f.id }, '绑定 ' + f.id + '…')))
      if (f.readOnly) {
        row.appendChild(btn('克隆到项目', '以预设为模板生成项目级副本后即可编辑（改自己的，不动上游）', () => cloneFlow(f.id)))
      } else {
        row.appendChild(btn('编辑', '在看板内编辑流程 JSON（先校验，再带乐观锁保存）', () => openEditor(f.id)))
      }
      if (f.overrideBy) row.appendChild(el('span', 'ftb-cell', '被 ' + f.overrideBy + ' 覆盖'))
      body.appendChild(row)
    })
  }

  // 编辑面板（D4-④：看板内「表单 / JSON」双模式，默认表单；写走 RPC + 乐观锁）
  function renderEditor(body) {
    const box = el('div', 'ftb-editor')
    const head = el('div', 'ftb-gate')
    head.appendChild(el('span', 'ftb-gate-id', editor.id))
    head.appendChild(el('span', 'ftb-cell', editor.scope + ' 层 · v' + editor.version))
    head.appendChild(el('span', 'ftb-cell', '保存带 expectedVersion 乐观锁'))
    box.appendChild(head)

    const bar = el('div', 'ftb-toolbar')
    const seg = el('div', 'ftb-seg')
    ;[['form', '表单'], ['json', 'JSON']].forEach(([m, label]) => {
      const b = el('button', 'ftb-mini' + (editor.mode === m ? ' active' : ''), label)
      b.type = 'button'
      b.title = m === 'form'
        ? '字段级表单：按流程定义逐字段编辑阶段与门禁（默认；无需手写 JSON）'
        : '直接编辑流程 JSON（同样的保存链路：flow_validate → flow_define）'
      b.addEventListener('click', () => setMode(m))
      seg.appendChild(b)
    })
    bar.appendChild(seg)
    if (editor.readOnly) bar.appendChild(el('span', 'ftb-readonly', '预设层只读（表单/JSON 均不可保存）——请先「克隆到项目」'))
    else bar.appendChild(el('span', 'ftb-note', '保存链路：表单本地校验 / JSON 语法 → flow_validate → flow_define'))
    box.appendChild(bar)

    if (editor.mode === 'form') {
      if (!editor.readOnly) box.appendChild(el('div', 'ftb-note', '表单只渲染 流程 / 阶段 / 门禁 字段；未渲染字段（如 board、appliesTo、onFail、library）在切换与保存时原样保留，不会被表单抹掉。'))
      renderForm(box)
    } else {
      const ta = document.createElement('textarea')
      ta.className = 'ftb-textarea'
      ta.spellcheck = false
      ta.value = editor.json
      ta.addEventListener('input', () => { if (editor) editor.json = ta.value })
      box.appendChild(ta)
    }

    if (editor.result) {
      const bad = /未通过|失败|非法|必须填|不能为空|重复/.test(editor.result)
      box.appendChild(el('div', bad ? 'ftb-err' : 'ftb-note', editor.result))
    }
    const foot = el('div', 'ftb-gate')
    foot.appendChild(btn('校验', 'flow_validate：阶段/门禁契约校验（不落盘；表单模式先做同规则本地校验）', () => editorValidate(), false))
    if (!editor.readOnly) foot.appendChild(btn('保存', 'flow_define：校验通过后带乐观锁写入该层', () => editorSave(), true))
    foot.appendChild(btn('取消', '放弃本次编辑', () => { editor = null; notify() }, false))
    box.appendChild(foot)
    body.appendChild(box)
  }

  // 表单渲染（D4-④）：阶段（id/标题/目标/任务，可增删 + 上下移）与门禁（kind 收敛字段，可增删）
  function renderForm(box) {
    const def = editor.def || (editor.def = {})
    const ro = !!editor.readOnly
    const stages = Array.isArray(def.stages) ? def.stages : (def.stages = [])

    const meta = el('div', 'ftb-block')
    const mh = el('div', 'ftb-block-head')
    mh.appendChild(el('span', 'ftb-name', '流程'))
    mh.appendChild(el('div', 'ftb-spacer'))
    mh.appendChild(el('span', 'ftb-cell', stages.length + ' 阶段'))
    meta.appendChild(mh)
    const idLocked = !editor.isNew
    meta.appendChild(field('流程 id', lock(txtInput(def.id, (v) => { if (editor.isNew) def.id = v }, ''), idLocked),
      idLocked ? '流程 id 不在此处改（改名等于另建流程）；需要复制请用「克隆到项目」' : '新流程 id（保存后落项目层 .pair/flows/<id>.json）'))
    meta.appendChild(field('标题', lock(txtInput(def.title, (v) => { def.title = v }), ro), '流程标题（title）'))
    meta.appendChild(field('版本', lock(numInput(def.version || 1, (v) => { def.version = Number(v) || 1 }, 1), ro), '流程定义版本（version）——乐观锁与发布包都取它'))
    if (Array.isArray(def.projectTypes)) {
      meta.appendChild(field('适用类型', lock(txtInput(def.projectTypes.join(','), (v) => { def.projectTypes = splitList(v.replace(/,/g, '；')) }), ro), '逗号/分号分隔：go / web / python / plugin / docs / any'))
    }
    box.appendChild(meta)

    stages.forEach((st, i) => {
      if (!st || typeof st !== 'object') { st = stages[i] = { id: '', title: '', gates: [] } }
      const gates = Array.isArray(st.gates) ? st.gates : (st.gates = [])
      const block = el('div', 'ftb-block')
      const bh = el('div', 'ftb-block-head')
      bh.appendChild(el('span', 'ftb-idx', '阶段 ' + (i + 1)))
      bh.appendChild(el('span', 'ftb-name', st.title || st.id || '（未命名）'))
      bh.appendChild(el('span', 'ftb-cell', gates.length + ' 门禁'))
      bh.appendChild(el('div', 'ftb-spacer'))
      if (!ro) {
        bh.appendChild(btn('↑', '上移该阶段（阶段顺序即流程顺序）', () => { if (i > 0) { stages.splice(i - 1, 0, stages.splice(i, 1)[0]); touchEditor() } }, false))
        bh.appendChild(btn('↓', '下移该阶段', () => { if (i < stages.length - 1) { stages.splice(i + 1, 0, stages.splice(i, 1)[0]); touchEditor() } }, false))
        bh.appendChild(btn('删除阶段', '删除该阶段（连同其门禁）', () => { stages.splice(i, 1); touchEditor() }, false))
      }
      block.appendChild(bh)
      block.appendChild(field('阶段 id', lock(txtInput(st.id, (v) => { st.id = v }), ro), '阶段唯一 id（必填；阶段内不得重复）'))
      block.appendChild(field('标题', lock(txtInput(st.title, (v) => { st.title = v }), ro), '阶段标题（title）'))
      block.appendChild(field('目标', lock(txtInput(st.goal, (v) => { st.goal = v }), ro), '阶段目标（goal）——会注入每轮提示，给 agent 上下文'))
      // 子任务模板（plan；兼容旧定义的 tasks）：进入该阶段时按条**播种**为子任务（幂等），
      // agent 之后可用 flow_task 在运行时继续拆解与推进状态。
      var planKey = Array.isArray(st.plan) ? 'plan' : (Array.isArray(st.tasks) ? 'tasks' : null)
      if (planKey) {
        const curPlan = st[planKey]
        const asText = (x) => (x && typeof x === 'object') ? String(x.subject || x.title || '') : String(x || '')
        block.appendChild(field('子任务模板', lock(txtInput(curPlan.map(asText).join('；'), (v) => {
          const keep = {}
          curPlan.forEach((x) => { if (x && typeof x === 'object' && x.subject) keep[x.subject] = x.detail || '' })
          st[planKey] = splitList(v).map((s) => (keep[s] !== undefined ? { subject: s, detail: keep[s] } : s))
        }), ro), '「；」分隔多条：进入该阶段时按条播种为子任务（agent 可再用 flow_task 增删改与推进）'))
      }

      const gbox = el('div', 'ftb-gate-box')
      gbox.appendChild(el('div', 'ftb-note', '门禁 ' + gates.length + ' 个 —— kind=command 必填命令、artifact 必填产物路径、approval 不自动通过（须用户批准）；必过（required）门禁未过时不允许推进。'))
      gates.forEach((g, j) => gbox.appendChild(renderGateForm(st, g, i, j, ro)))
      if (!ro) gbox.appendChild(btn('＋ 加门禁', '在该阶段末尾新增一个 check 门禁', () => {
        gates.push({ id: 'g-' + (st.id || 'new') + '-' + (gates.length + 1), kind: 'check', text: '', required: true, blocking: false })
        touchEditor()
      }, false))
      block.appendChild(gbox)
      box.appendChild(block)
    })

    if (ro) box.appendChild(el('div', 'ftb-readonly', '预设层只读：表单仅展示，「保存」按钮已隐藏。点「取消」返回后，用流程库里的「克隆到项目」生成可编辑副本。'))
    else box.appendChild(btn('＋ 加阶段', '在末尾新增一个阶段', () => {
      stages.push({ id: 'stage-' + (stages.length + 1), title: '新阶段', goal: '', plan: [], gates: [] })
      touchEditor()
    }, true))
  }

  // 门禁表单块：按 kind 收敛字段（command→cmd、artifact→path、approval→提示语）
  function renderGateForm(st, g, si, gi, ro) {
    if (!g || typeof g !== 'object') { g = st.gates[gi] = { id: '', kind: 'check' } }
    const wrap = el('div', 'ftb-gate-form')
    const head = el('div', 'ftb-block-head')
    head.appendChild(el('span', 'ftb-idx', (si + 1) + '.' + (gi + 1)))
    head.appendChild(el('span', 'ftb-cell', 'kind=' + String(g.kind || '') + (g.required === false ? ' · 非必过' : ' · 必过') + (g.blocking ? ' · 阻塞' : '')))
    head.appendChild(el('div', 'ftb-spacer'))
    if (!ro) head.appendChild(btn('删除门禁', '删除该门禁', () => { st.gates.splice(gi, 1); touchEditor() }, false))
    wrap.appendChild(head)
    wrap.appendChild(field('门禁 id', lock(txtInput(g.id, (v) => { g.id = v }), ro), '门禁唯一 id（必填；阶段内不得重复）'))
    wrap.appendChild(field('kind', lock(selInput(GATE_KINDS, g.kind || 'check', (v) => { g.kind = v; touchEditor() }, 'check=人工确认 / command=插件代跑命令 / artifact=校验产物存在 / approval=用户批准'), ro), '切换 kind 会按类型收敛字段'))
    wrap.appendChild(field(KIND_LABEL_SHORT[g.kind] || '说明', lock(txtInput(g.text, (v) => { g.text = v }), ro), KIND_TEXT_LABEL[g.kind] || '门禁说明（会出现在看板与每轮提示里）'))
    if (g.kind === 'command') wrap.appendChild(field('命令 cmd', lock(txtInput(g.cmd, (v) => { g.cmd = v }), ro), '必填：flow_advance 时由插件代跑并留证（.pair/flow-runs/evidence/<runId>/<gateId>.log）'))
    if (g.kind === 'artifact') wrap.appendChild(field('产物路径', lock(txtInput(g.artifact, (v) => { g.artifact = v }), ro), '必填：过门禁时校验该路径存在（工作区相对路径）'))
    if (g.kind === 'approval') wrap.appendChild(el('div', 'ftb-note', '审批卡点：不会自动通过；用户在看板点「批准」或在对话中确认后落库（记 approvedBy=user）。'))
    const flags = el('div', 'ftb-field')
    flags.appendChild(el('span', 'ftb-label', '标记'))
    flags.appendChild(lock(checkInput(g.required !== false, '必过 required', (v) => { g.required = v }, '必过门禁未通过 → 不允许推进/收尾'), ro))
    flags.appendChild(lock(checkInput(!!g.blocking, '阻塞 blocking', (v) => { g.blocking = v }, '加重标记：enforce 强度下破坏性操作被拦截'), ro))
    wrap.appendChild(flags)
    return wrap
  }
  // ═══════════════════════════════════════════════════════════════
  // 任务看板 —— 看板驱动 agent（2026-10-17 重做，用户诉求）
  //   · 一张卡 = 一个可执行任务：描述 + 执行提示词（多行、一一对应）+ 验收清单（DoD）
  //   · 闭环：人建卡 → 认领 → 执行（任务书投递到工作会话，由 agent / 监督者多轮驱动）
  //          → agent 用 taskboard_* 回填拆解 / 进度 / 报告 → 人验收（done 只有人能放）
  //   · 卡面直接可见：认领人 / 执行中 / 定时下次时间 / 拆解进度 / DoD 进度
  // ═══════════════════════════════════════════════════════════════
  const TB_MAIN = [['backlog', '待规划'], ['todo', '待办'], ['in_progress', '进行中'], ['in_review', '待验收'], ['done', '已完成']]
  const TB_SEC = [['canceled', '已取消'], ['archived', '已归档']]
  const TB_LABEL = { backlog: '待规划', todo: '待办', in_progress: '进行中', in_review: '待验收', done: '已完成', canceled: '已取消', archived: '已归档' }
  const TB_URG = { urgent: '紧急', normal: '一般', relaxed: '不急' }
  const TB_TRANS = {
    backlog: ['todo', 'canceled'],
    todo: ['in_progress', 'backlog', 'canceled'],
    in_progress: ['in_review', 'todo', 'canceled'],
    in_review: ['in_progress', 'todo', 'done', 'canceled'],
    done: ['archived', 'todo'],
    canceled: ['archived', 'todo'],
    archived: [],
  }
  const TB_SUB_LABEL = { pending: '待办', in_progress: '进行中', completed: '已完成', blocked: '阻塞', cancelled: '取消' }
  const TB_DOT = { backlog: 'var(--text-muted)', todo: '#d4a74e', in_progress: '#569cd6', in_review: '#d4a74e', done: '#6a9955' }
  const TB_CSS = [
    // ── 工具栏 ────────────────────────────────────────────────
    '.ftb-tb-bar{display:flex;align-items:center;gap:8px;flex-wrap:wrap;margin-bottom:12px}',
    '.ftb-seg{display:inline-flex;border:1px solid var(--border-color);border-radius:8px;overflow:hidden}',
    '.ftb-seg .ftb-btn{border:none;border-radius:0;padding:5px 10px}',
    '.ftb-seg .ftb-btn.active{background:var(--bg-active);color:var(--text-primary)}',
    '.ftb-search{font:inherit;font-size:12px;color:var(--text-primary);background:var(--bg-primary);border:1px solid var(--border-color);border-radius:8px;padding:5px 10px;width:210px;box-sizing:border-box;outline:none}',
    '.ftb-search:focus{border-color:#569cd6}',
    '.ftb-search::placeholder{color:var(--text-muted)}',
    // ── 看板列 ────────────────────────────────────────────────
    '.ftb-tb-cols{display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:12px;align-items:start}',
    '.ftb-tb-cols.sec{grid-template-columns:repeat(2,minmax(0,1fr))}',
    '.ftb-tcol{display:flex;flex-direction:column;min-width:0;background:var(--bg-secondary);border:1px solid var(--border-color);border-radius:10px;padding:6px}',
    '.ftb-tcol.drop-on{border-color:#569cd6;border-style:dashed;background:var(--bg-active)}',
    '.ftb-tcol-head{display:flex;align-items:center;gap:6px;padding:6px 8px 8px;font-size:12px;font-weight:600;color:var(--text-secondary)}',
    '.ftb-tcol-dot{width:7px;height:7px;border-radius:50%;background:var(--text-muted);flex-shrink:0}',
    '.ftb-tcol-count{margin-left:auto;font-size:11px;color:var(--text-muted);background:var(--bg-active);border-radius:10px;padding:1px 7px;min-width:20px;text-align:center;font-variant-numeric:tabular-nums}',
    '.ftb-tcol-body{display:flex;flex-direction:column;gap:8px;padding:2px;min-height:60px}',
    '.ftb-tcol-empty{color:var(--text-muted);font-size:11px;text-align:center;padding:16px 6px;border:1px dashed var(--border-color);border-radius:8px;line-height:1.6}',
    // ── 任务卡 ────────────────────────────────────────────────
    '.ftb-tcard{position:relative;display:flex;flex-direction:column;gap:6px;border:1px solid var(--border-color);border-left:3px solid var(--text-muted);border-radius:8px;background:var(--bg-primary);padding:9px 10px;cursor:pointer;transition:border-color .12s,box-shadow .12s,transform .12s}',
    '.ftb-tcard:hover{border-color:var(--text-muted);box-shadow:0 2px 8px rgba(0,0,0,.18);transform:translateY(-1px)}',
    '.ftb-tcard.dragging{opacity:.45}',
    '.ftb-tcard.sel{border-color:#569cd6;box-shadow:0 0 0 1px #569cd6 inset}',
    '.ftb-tcard.u-urgent{border-left-color:#e5534b}',
    '.ftb-tcard.u-normal{border-left-color:#569cd6}',
    '.ftb-tcard.u-relaxed{border-left-color:#6a9955}',
    '.ftb-tcard.blocked{outline:1px solid #e5534b;outline-offset:-1px}',
    '.ftb-tcard-title{font-size:13px;font-weight:600;line-height:1.45;color:var(--text-primary);word-break:break-word}',
    '.ftb-tcard-desc{font-size:11px;line-height:1.55;color:var(--text-muted);display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden;word-break:break-word}',
    '.ftb-tcard-meta{display:flex;align-items:center;gap:5px;flex-wrap:wrap}',
    '.ftb-tchip{display:inline-flex;align-items:center;gap:3px;font-size:10px;line-height:1.7;color:var(--text-muted);background:var(--bg-active);border-radius:10px;padding:0 7px;flex-shrink:0}',
    '.ftb-tchip.flow{color:#569cd6}',
    '.ftb-tchip.claim{color:#d4a74e}',
    '.ftb-tchip.run{color:#6a9955}',
    '.ftb-tchip.timer{color:#569cd6}',
    '.ftb-tchip.blocked{color:#e5534b}',
    '.ftb-tchip.demo{color:#d4a74e;border:1px dashed rgba(212,167,78,.55)}',
    '.ftb-tcard-bar{height:3px;border-radius:2px;background:var(--bg-active);overflow:hidden}',
    '.ftb-tcard-bar i{display:block;height:100%;background:#6a9955;border-radius:2px;transition:width .2s}',
    // ── 详情面板 ──────────────────────────────────────────────
    '.ftb-detail{max-width:980px;margin:12px auto 0;border:1px solid var(--border-color);border-radius:12px;background:var(--bg-secondary);padding:14px 16px;box-shadow:0 6px 24px rgba(0,0,0,.16);container-type:inline-size}',
    '.ftb-detail-head{display:flex;align-items:center;gap:8px;flex-wrap:wrap;margin-bottom:4px}',
    '.ftb-detail-title{font-size:15px;font-weight:700;color:var(--text-primary);word-break:break-word}',
    '.ftb-detail-meta{font-size:11px;color:var(--text-muted);line-height:1.7;margin:2px 0 10px}',
    '.ftb-sec{margin-top:14px;padding-top:12px;border-top:1px solid var(--border-color)}',
    '.ftb-sec-head{display:flex;align-items:center;gap:8px;margin-bottom:8px;flex-wrap:wrap}',
    '.ftb-sec-title{font-size:12px;font-weight:600;color:var(--text-secondary);letter-spacing:.2px}',
    '.ftb-sec-hint{font-size:11px;color:var(--text-muted);margin-left:auto;font-variant-numeric:tabular-nums}',
    '.ftb-ta{display:block;width:100%;box-sizing:border-box;resize:vertical;min-height:70px;font:inherit;font-size:12px;line-height:1.6;color:var(--text-primary);background:var(--bg-primary);border:1px solid var(--border-color);border-radius:8px;padding:8px 10px;outline:none}',
    '.ftb-ta:focus{border-color:#569cd6}',
    '.ftb-ta.prompt{min-height:96px;border-color:#3d6d96}',
    '.ftb-pair{display:grid;grid-template-columns:1fr;gap:10px;align-items:start}',
    '.ftb-pair-item{display:flex;flex-direction:column;gap:5px}',
    '.ftb-pair-label{font-size:11px;color:var(--text-muted);display:flex;align-items:center;gap:6px;height:22px;min-height:22px;box-sizing:border-box;overflow:hidden}',
    '.ftb-mini{font:inherit;font-size:10px;line-height:18px;height:18px;padding:0 8px;color:var(--text-secondary);background:var(--bg-active);border:1px solid var(--border-color);border-radius:9px;cursor:pointer;flex-shrink:0}',
    '.ftb-mini:hover{color:var(--text-primary);border-color:var(--text-muted)}',
    // 真双栏：面板可用宽度 ≥860px 时左右并排（描述左 / 执行提示词右）；窄屏自动降级单列
    '@media (min-width:1100px){.ftb-pair{grid-template-columns:1fr 1fr;align-items:start}}',
    '@container (min-width:860px){.ftb-pair{grid-template-columns:1fr 1fr;align-items:start}}',
    '.ftb-row-actions{display:flex;align-items:center;gap:6px;flex-wrap:wrap;margin-top:10px}',
    // ── 执行条 / 定时条 ───────────────────────────────────────
    '.ftb-exec{display:flex;align-items:center;gap:8px;flex-wrap:wrap;background:var(--bg-primary);border:1px solid var(--border-color);border-radius:10px;padding:10px 12px}',
    '.ftb-exec-state{font-size:11px;color:var(--text-muted)}',
    '.ftb-exec-state.run{color:#6a9955}',
    '.ftb-exec-state.err{color:#e5534b}',
    '.ftb-sched{display:flex;align-items:center;gap:6px;flex-wrap:wrap;margin-top:8px}',
    '.ftb-sched input[type=number],.ftb-sched input[type=time],.ftb-sched input[type=text],.ftb-sched select{font:inherit;font-size:11px;color:var(--text-primary);background:var(--bg-primary);border:1px solid var(--border-color);border-radius:6px;padding:3px 7px}',
    '.ftb-sched-tag{font-size:10px;color:#569cd6;background:var(--bg-active);border-radius:9px;padding:0 7px}',
    // ── 子项 / DoD ───────────────────────────────────────────
    '.ftb-subs{display:flex;flex-direction:column;gap:6px}',
    '.ftb-sub{display:flex;align-items:center;gap:8px;font-size:12px;color:var(--text-secondary);background:var(--bg-primary);border:1px solid var(--border-color);border-radius:8px;padding:6px 9px;cursor:pointer}',
    '.ftb-sub:hover{border-color:var(--text-muted)}',
    '.ftb-sub.completed{color:var(--text-muted);text-decoration:line-through}',
    '.ftb-sub-dot{width:8px;height:8px;border-radius:50%;background:var(--text-muted);flex-shrink:0}',
    '.ftb-sub.in_progress .ftb-sub-dot{background:#569cd6}',
    '.ftb-sub.completed .ftb-sub-dot{background:#6a9955}',
    '.ftb-sub.blocked .ftb-sub-dot{background:#e5534b}',
    '.ftb-clist{margin:4px 0;padding:0;list-style:none}',
    '.ftb-clist li{display:flex;align-items:flex-start;gap:6px;font-size:12px;color:var(--text-secondary);line-height:1.6;padding:3px 0}',
    '.ftb-clist li.done{color:var(--text-muted);text-decoration:line-through}',
    // ── 通用 ─────────────────────────────────────────────────
    '.ftb-note{color:var(--text-muted);font-size:11px;line-height:1.7;white-space:pre-wrap;word-break:break-word}',
    '.ftb-note.agent{border-left:2px solid #569cd6;padding-left:8px}',
    '.ftb-note.user{border-left:2px solid #6a9955;padding-left:8px}',
    '.ftb-tb-meta{font-size:11px;color:var(--text-muted);line-height:1.7;margin:4px 0}',
  ].join('\n')
  let tbSnap = null
  let tbTab = 'board'          // board | library
  let tbUrg = ''               // 紧急度筛选
  let tbSec = false            // false=主看板 true=已取消/已归档
  let tbDetail = ''            // 展开详情的任务 id
  let tbNew = false            // 新建任务表单是否展开
  let tbDraft = null
  let tbMsg = ''
  let tbDrag = null
  let tbSearch = ''
  let tbPending = ''           // 正在操作的任务 id（防重复点击）
  let tbRoundCache = {}        // convId → 监督回合摘要（懒加载）

  function tbBlankDraft() {
    return {
      title: '', description: '', prompt: '', urgency: 'normal', flowId: '', convId: '',
      schedule: { enabled: false, mode: 'interval', everyMinutes: 60, atTime: '09:00', atISO: '' },
    }
  }
  tbDraft = tbBlankDraft()

  async function loadBoard() {
    try { tbSnap = (await ui.http.get('/taskboard/state', {})) || null }
    catch (e) { tbSnap = null }
    notify()
  }
  async function tbCall(payload) {
    const r = await ui.invoke(PLUGIN, 'taskboard_action', payload || {})
    if (r && r.ok === false) throw new Error(r.error || '未知错误')
    return r || {}
  }
  async function tbDo(payload, okMsg) {
    try {
      const r = await tbCall(payload)
      tbMsg = ''
      await loadBoard()
      if (okMsg) notifyUser(okMsg, 'info')
      return r
    } catch (e) {
      tbMsg = String((e && e.message) || e)
      notify()
      notifyUser(tbMsg, 'error')
      return null
    }
  }
  /** 驱动模式切换（手动 / 全流程自动 + 自动验收 / 并发）。 */
  async function tbSetDriver(patch) {
    const p = patch || {}
    const msg = p.mode
      ? (p.mode === 'auto'
        ? '已切到「全流程自动」：待办卡会自动派发给 agent，无需点执行'
        : '已切到「手动」：每张卡由人点执行 / 认领 / 定时')
      : (p.autoAccept !== undefined
        ? ('自动验收已' + (p.autoAccept ? '开启' : '关闭'))
        : ('自动并发已设为 ' + p.maxConcurrent))
    await tbDo({ op: 'driver', patch: p, convId: currentConvId() }, msg)
  }
  async function tbRun(id) {
    if (tbPending) return
    tbPending = id
    notify()
    try {
      const r = await tbDo({ op: 'execute', id: id, convId: currentConvId() }, '已派发执行（agent 开始工作，进度会实时回填）')
      if (r && r.ok === false) tbMsg = r.error || '执行派发失败'
    } finally {
      tbPending = ''
      notify()
    }
  }

  function tbTasks() { return (tbSnap && tbSnap.tasks) || [] }
  function tbFind(id) { return tbTasks().filter((x) => x.id === id)[0] || null }
  function tbIsSec(t) { return t.status === 'canceled' || t.status === 'archived' }
  function tbCanMove(from, to) { const t = TB_TRANS[from]; return !!(t && t.indexOf(to) >= 0) }
  function tbFlowLibrary() { return (tbSnap && tbSnap.flowLibrary) || [] }
  function tbVisible() {
    let rows = tbTasks().filter((t) => tbSec ? tbIsSec(t) : !tbIsSec(t))
    if (tbUrg) rows = rows.filter((t) => t.urgency === tbUrg)
    const q = String(tbSearch || '').trim().toLowerCase()
    if (q) {
      rows = rows.filter((t) => (String(t.title || '') + ' ' + String(t.description || '') + ' ' + String(t.prompt || '')).toLowerCase().indexOf(q) >= 0)
    }
    return rows
  }
  function tbShort(iso) {
    const s = String(iso || '')
    const m = s.match(/T(\d{2}:\d{2})/)
    if (m) return m[1]
    const d = new Date(s)
    if (isNaN(d.getTime())) return s.slice(0, 10)
    return (d.getMonth() + 1) + '/' + d.getDate()
  }
  function tbSchedText(s) {
    if (!s || !s.enabled) return ''
    if (s.mode === 'interval') return '每 ' + s.everyMinutes + ' 分钟'
    if (s.mode === 'daily') return '每天 ' + s.atTime
    return '一次 ' + (s.atISO ? tbShort(s.atISO) : '')
  }
  function tbSubStats(subs) {
    const list = Array.isArray(subs) ? subs : []
    const done = list.filter((x) => x.status === 'completed').length
    return { total: list.length, done: done, pct: list.length ? Math.round(done * 100 / list.length) : 0 }
  }
  // ── 详情编辑内核：本地草稿 + 防抖 600ms（★ 输入时不重渲染）──
  // 关键：input 只更新本地草稿，绝不 notify/loadBoard —— 否则面板重建会替换 textarea 节点，
  //       逐字输入必丢焦点与内容（监督者 CDP 实测 12 字只剩 1 字即此因）。
  // 保存：字段级串行（inflight 时只置 dirty，完成后补发「最后一次值」）→ 同一波输入只发 1 次 RPC；
  //       响应只把 task 合并进 tbSnap + 同步卡片摘要文本，不整页 notify。
  const tbEdit = {}
  function tbEditState(id) {
    if (!tbEdit[id]) tbEdit[id] = { description: null, prompt: null, timer: null, inflight: false, dirty: false }
    return tbEdit[id]
  }
  function tbEditSeed(t, field) {
    const st = tbEditState(t.id)
    const saved = String(t[field] == null ? '' : t[field])
    if (!st.timer && !st.inflight && !st.dirty) st[field] = saved
    else if (st[field] == null) st[field] = saved
    return st
  }
  function tbMergeTask(task) {
    if (!task || !tbSnap || !Array.isArray(tbSnap.tasks)) return
    for (let i = 0; i < tbSnap.tasks.length; i++) {
      if (tbSnap.tasks[i].id === task.id) { tbSnap.tasks[i] = task; return }
    }
  }
  function tbSyncCardText(id, field, value) {
    const nodes = document.querySelectorAll('[data-tb-' + field + '="' + String(id).replace(/"/g, '') + '"]')
    for (let i = 0; i < nodes.length; i++) nodes[i].textContent = value
  }
  function tbEditFlush(t, field) {
    const st = tbEditState(t.id)
    if (st.timer) { clearTimeout(st.timer); st.timer = null }
    const val = st[field]
    const saved = String(t[field] == null ? '' : t[field])
    if (val == null || val === saved) return
    if (st.inflight) { st.dirty = true; return }
    st.inflight = true
    const patch = {}
    patch[field] = val
    tbCall({ op: 'update', id: t.id, patch: patch })
      .then((r) => {
        if (r && r.task) {
          tbMergeTask(r.task)
          t[field] = String(r.task[field] == null ? '' : r.task[field])
          if (field === 'description') tbSyncCardText(t.id, 'desc', String(r.task.description == null ? '' : r.task.description).trim())
          if (field === 'prompt') tbSyncCardText(t.id, 'prompt', String(r.task.prompt == null ? '' : r.task.prompt))
        }
      })
      .catch((e) => { tbMsg = String((e && e.message) || e) })
      .then(() => {
        st.inflight = false
        if (st.dirty) { st.dirty = false; tbEditFlush(t, field) }
      })
  }
  function tbEditBind(t, field, ta) {
    tbEditSeed(t, field)
    ta.addEventListener('input', () => {
      const st = tbEditState(t.id)
      st[field] = ta.value
      if (st.timer) clearTimeout(st.timer)
      st.timer = setTimeout(() => tbEditFlush(t, field), 600)
    })
    ta.addEventListener('blur', () => tbEditFlush(t, field))
  }
  // 认领者可读身份：user → 「你」；agent → 「agent·会话后 4 位」；完整 convId 放 title
  function tbWho(t) {
    const kind = String(t.claimedByKind || '')
    const id = String(t.claimedBy || '')
    if (kind === 'user') return '你'
    if (kind === 'agent') return id ? ('agent·' + id.slice(-4)) : 'agent'
    if (!id) return '你'
    return 'agent·' + id.slice(-4)
  }
  function tbWhoTitle(t) {
    return t.claimedBy ? ('认领会话：' + t.claimedBy) : ''
  }
  function tbTextarea(value, placeholder, rows, cls, onInput) {
    const ta = document.createElement('textarea')
    ta.className = 'ftb-ta' + (cls ? ' ' + cls : '')
    ta.rows = rows || 3
    ta.placeholder = placeholder || ''
    ta.value = value == null ? '' : String(value)
    if (onInput) ta.addEventListener('input', () => onInput(ta.value))
    return ta
  }

  // ── 单张任务卡：标题 + 描述摘要 + 状态/进度徽标（认领/执行/定时/拆解/DoD）──
  function tbCard(t, host) {
    const node = el('div', 'ftb-tcard u-' + (t.urgency || 'normal') + (t.blocked ? ' blocked' : '') + (tbDetail === t.id ? ' sel' : ''))
    node.draggable = true
    node.title = '点开详情：认领 / 执行 / 定时 · 拖到其他列改状态'
    node.addEventListener('dragstart', () => { tbDrag = t.id; node.className += ' dragging' })
    node.addEventListener('dragend', () => { tbDrag = null; node.className = node.className.replace(' dragging', '') })
    node.addEventListener('click', () => {
      tbDetail = (tbDetail === t.id) ? '' : t.id
      tbNew = false
      notify()
    })
    const titleNode = el('div', 'ftb-tcard-title', t.title || '(无标题)')
    titleNode.setAttribute('data-tb-title', t.id)
    node.appendChild(titleNode)
    const desc = String(t.description || '').trim()
    if (desc) {
      const descNode = el('div', 'ftb-tcard-desc', desc)
      descNode.setAttribute('data-tb-desc', t.id)
      node.appendChild(descNode)
    }
    const meta = el('div', 'ftb-tcard-meta')
    if (t.demo) meta.appendChild(el('span', 'ftb-tchip demo', 'demo'))
    if (t.urgency && t.urgency !== 'normal') meta.appendChild(el('span', 'ftb-tchip', TB_URG[t.urgency] || t.urgency))
    if (t.flowId) {
      const fp = t.flowProgress || null
      const fpTxt = fp && fp.label && fp.label !== '未启动' ? (' · ' + fp.label) : ''
      meta.appendChild(el('span', 'ftb-tchip flow', '流程 ' + t.flowId + fpTxt))
    }
    if (t.blocked) meta.appendChild(el('span', 'ftb-tchip blocked', '阻塞'))
    const rs = t.runState || {}
    if (rs.running) meta.appendChild(el('span', 'ftb-tchip run', '执行中…'))
    else if (t.status === 'in_progress') {
      const claimChip = el('span', 'ftb-tchip claim', '认领 ' + tbWho(t))
      const wt = tbWhoTitle(t)
      if (wt) claimChip.title = wt
      meta.appendChild(claimChip)
    }
    const sch = t.schedule
    if (sch && sch.enabled) meta.appendChild(el('span', 'ftb-tchip timer', '⏱ ' + tbSchedText(sch) + (sch.nextAt ? ' · ' + tbShort(sch.nextAt) : '')))
    const st = tbSubStats(t.subtasks)
    if (st.total) meta.appendChild(el('span', 'ftb-tchip', '拆解 ' + st.done + '/' + st.total))
    const cl = t.checklist || []
    if (cl.length) {
      const done = cl.filter((x) => x.checked).length
      meta.appendChild(el('span', 'ftb-tchip', 'DoD ' + done + '/' + cl.length))
    }
    if (rs.lastEndAt && !rs.running) meta.appendChild(el('span', 'ftb-tchip', (rs.lastError ? '执行失败' : '已执行 ' + tbShort(rs.lastEndAt))))
    if ((t.comments || []).length) meta.appendChild(el('span', 'ftb-tchip', '评论 ' + t.comments.length))
    if (meta.children && meta.children.length) node.appendChild(meta)
    // 进度条：优先拆解，其次 DoD
    if (st.total) {
      const bar = el('div', 'ftb-tcard-bar')
      const fill = el('i')
      fill.style.width = st.pct + '%'
      bar.appendChild(fill)
      node.appendChild(bar)
    } else if (cl.length) {
      const done = cl.filter((x) => x.checked).length
      const bar = el('div', 'ftb-tcard-bar')
      const fill = el('i')
      fill.style.width = Math.round(done * 100 / cl.length) + '%'
      bar.appendChild(fill)
      node.appendChild(bar)
    }
    host.appendChild(node)
  }

  // ── 一列：列头（状态点 + 名称 + 计数）+ 卡片堆 + 空列占位 ──
  function tbColumn(key, label, rows, container) {
    const col = el('div', 'ftb-tcol')
    const head = el('div', 'ftb-tcol-head')
    const dot = el('span', 'ftb-tcol-dot')
    dot.style.background = TB_DOT[key] || 'var(--text-muted)'
    head.appendChild(dot)
    head.appendChild(el('span', null, label))
    head.appendChild(el('span', 'ftb-tcol-count', String(rows.length)))
    col.appendChild(head)
    const box = el('div', 'ftb-tcol-body')
    if (!rows.length) box.appendChild(el('div', 'ftb-tcol-empty', '暂无任务'))
    rows.forEach((t) => tbCard(t, box))
    col.appendChild(box)
    col.addEventListener('dragover', (e) => {
      if (e && typeof e.preventDefault === 'function') e.preventDefault()
      if (tbDrag && col.className.indexOf('drop-on') < 0) col.className += ' drop-on'
    })
    col.addEventListener('dragleave', () => { col.className = col.className.replace(' drop-on', '') })
    col.addEventListener('drop', (e) => {
      if (e && typeof e.preventDefault === 'function') e.preventDefault()
      col.className = col.className.replace(' drop-on', '')
      const id = tbDrag
      tbDrag = null
      if (!id) return
      const t = tbFind(id)
      if (!t || t.status === key) return
      if (!tbCanMove(t.status, key)) {
        tbMsg = '无法从「' + (TB_LABEL[t.status] || t.status) + '」移到「' + (TB_LABEL[key] || key) + '」'
        notify()
        return
      }
      tbDo({ op: 'move', id: id, to: key, convId: currentConvId() }, '已移到「' + (TB_LABEL[key] || key) + '」')
    })
    container.appendChild(col)
  }

  // ── 主看板：工具栏（搜索/紧急度/主次看板/新建）+ 五列 + 详情 ──
  function renderTaskBoard(body) {
    const bar = el('div', 'ftb-tb-bar')
    const search = document.createElement('input')
    search.type = 'text'
    search.className = 'ftb-search'
    search.placeholder = '搜索标题 / 描述 / 执行提示词'
    search.value = tbSearch
    search.addEventListener('input', () => { tbSearch = search.value; notify() })
    bar.appendChild(search)
    const seg = el('div', 'ftb-seg')
    ;[['', '全部'], ['urgent', '紧急'], ['normal', '一般'], ['relaxed', '不急']].forEach(([k, label]) => {
      const b = el('button', 'ftb-btn' + (tbUrg === k ? ' active' : ''), label)
      b.type = 'button'
      b.addEventListener('click', () => { tbUrg = k; notify() })
      seg.appendChild(b)
    })
    bar.appendChild(seg)
    const seg2 = el('div', 'ftb-seg')
    ;[[false, '主看板'], [true, '已取消 / 已归档']].forEach(([k, label]) => {
      const b = el('button', 'ftb-btn' + (tbSec === k ? ' active' : ''), label)
      b.type = 'button'
      b.addEventListener('click', () => { tbSec = k; tbDetail = ''; notify() })
      seg2.appendChild(b)
    })
    bar.appendChild(seg2)
    // 驱动模式：手动（每卡点执行）/ 全流程自动（自动派发待办 + 可选自动验收）
    const drv = (tbSnap && tbSnap.driver) || { mode: 'manual', autoAccept: false, maxConcurrent: 1 }
    const segDrv = el('div', 'ftb-seg')
    ;[['manual', '手动'], ['auto', '全流程自动']].forEach(([k, label]) => {
      const b = el('button', 'ftb-btn' + (drv.mode === k ? ' active' : ''), label)
      b.type = 'button'
      b.title = k === 'auto'
        ? '自动派发「待办」卡给 agent（无需点执行），可再开「自动验收」'
        : '每张卡由人点「执行 / 认领 / 定时」'
      b.addEventListener('click', () => {
        if (drv.mode === k) return
        tbSetDriver({ mode: k })
      })
      segDrv.appendChild(b)
    })
    bar.appendChild(segDrv)
    if (drv.mode === 'auto') {
      const ac = el('button', 'ftb-btn' + (drv.autoAccept ? ' active' : ''), '自动验收')
      ac.type = 'button'
      ac.title = 'agent 交报告后自动转「已完成」（执行记录留 auto-accept 痕迹；执行失败的卡不自动验收）'
      ac.addEventListener('click', () => tbSetDriver({ autoAccept: !drv.autoAccept }))
      bar.appendChild(ac)
      const cc = el('button', 'ftb-btn', '并发 ' + drv.maxConcurrent)
      cc.type = 'button'
      cc.title = '自动模式下同时执行的卡数（点击在 1-3 之间循环）'
      cc.addEventListener('click', () => tbSetDriver({ maxConcurrent: drv.maxConcurrent >= 3 ? 1 : drv.maxConcurrent + 1 }))
      bar.appendChild(cc)
    }
    bar.appendChild(el('div', 'ftb-spacer'))
    bar.appendChild(btn(tbNew ? '取消新建' : '＋ 新建任务', '建卡：填标题 + 描述 + 执行提示词', () => {
      tbNew = !tbNew
      tbDetail = ''
      if (tbNew) tbDraft = tbBlankDraft()
      notify()
    }, !tbNew))
    body.appendChild(bar)

    if (tbNew) renderNewTask(body)
    const rows = tbVisible()
    const cols = el('div', 'ftb-tb-cols' + (tbSec ? ' sec' : ''))
    if (tbSec) {
      TB_SEC.forEach(([k, label]) => tbColumn(k, label, rows.filter((t) => t.status === k), cols))
    } else {
      TB_MAIN.forEach(([k, label]) => tbColumn(k, label, rows.filter((t) => t.status === k), cols))
    }
    body.appendChild(cols)
    if (!rows.length) {
      body.appendChild(el('div', 'ftb-note', tbTasks().length
        ? '当前筛选下没有任务。'
        : (drv.mode === 'auto'
          ? '看板是空的 —— 建一张卡即可：「全流程自动」会在 60s 内自动把它派发给 agent，跑完自动转「待验收」（开了自动验收则直接完成）。'
          : '看板还没有任务 —— 点「＋ 新建任务」建一张卡：写清描述与执行提示词，然后点卡片里的「执行」，agent 会按卡工作并回填进度。')))
    }
    if (tbDetail) {
      const t = tbFind(tbDetail)
      if (t) renderTaskDetail(body, t)
    }
  }

  // ── 建卡表单：标题 + 描述 + 执行提示词（多行、一一对应）+ 紧急度/流程/定时 ──
  function renderNewTask(body) {
    const box = el('div', 'ftb-detail')
    box.appendChild(el('div', 'ftb-detail-title', '新建任务'))
    box.appendChild(el('div', 'ftb-detail-meta', '描述 = 要做什么 / 背景与验收标准；执行提示词 = 交给 agent 的具体要求。两者一对一，都会进入执行任务书。'))

    const f1 = el('div', 'ftb-pair-item')
    f1.appendChild(el('div', 'ftb-pair-label', '任务标题'))
    const inp = document.createElement('input')
    inp.type = 'text'
    inp.placeholder = '一句话任务名（卡片名）'
    inp.value = tbDraft.title
    inp.style.cssText = 'font:inherit;font-size:13px;color:var(--text-primary);background:var(--bg-primary);border:1px solid var(--border-color);border-radius:8px;padding:7px 10px;outline:none'
    inp.addEventListener('input', () => { tbDraft.title = inp.value })
    f1.appendChild(inp)
    box.appendChild(f1)

    const pair = el('div', 'ftb-pair')
    const d1 = el('div', 'ftb-pair-item')
    d1.appendChild(el('div', 'ftb-pair-label', '描述（多行）'))
    d1.appendChild(tbTextarea(tbDraft.description, '背景、范围、验收标准…（可多行）', 3, '', (v) => { tbDraft.description = v }))
    pair.appendChild(d1)
    const d2 = el('div', 'ftb-pair-item')
    const lab2 = el('div', 'ftb-pair-label')
    lab2.appendChild(el('span', null, '执行提示词（多行）'))
    lab2.appendChild(btn('用描述生成', '把描述整理成给 agent 的执行指令', () => {
      tbDraft.prompt = tbPromptFromDesc(tbDraft.title, tbDraft.description)
      notify()
    }))
    d2.appendChild(lab2)
    d2.appendChild(tbTextarea(tbDraft.prompt, '告诉 agent 怎么干：步骤、边界、产物、验证方式…（可多行）', 4, 'prompt', (v) => { tbDraft.prompt = v }))
    pair.appendChild(d2)
    box.appendChild(pair)

    const opts = el('div', 'ftb-row-actions')
    const sel = document.createElement('select')
    ;[['urgent', '紧急'], ['normal', '一般'], ['relaxed', '不急']].forEach(([v, label]) => {
      const o = document.createElement('option')
      o.value = v
      o.textContent = '紧急度：' + label
      if (tbDraft.urgency === v) o.selected = true
      sel.appendChild(o)
    })
    sel.addEventListener('change', () => { tbDraft.urgency = sel.value })
    opts.appendChild(sel)
    const sel2 = document.createElement('select')
    const o0 = document.createElement('option')
    o0.value = ''
    o0.textContent = '流程：不绑定'
    sel2.appendChild(o0)
    tbFlowLibrary().forEach((f) => {
      const o = document.createElement('option')
      o.value = f.id
      o.textContent = '流程：' + f.title + '（' + f.stages + ' 阶段）'
      if (tbDraft.flowId === f.id) o.selected = true
      sel2.appendChild(o)
    })
    sel2.addEventListener('change', () => { tbDraft.flowId = sel2.value; notify() })
    opts.appendChild(sel2)
    // 执行会话（可选）：留空 = 当前会话；用于「新建卡 → 定时 → 到点自动跑」
    const convIn = document.createElement('input')
    convIn.type = 'text'
    convIn.placeholder = '执行会话（留空=当前会话）'
    convIn.value = tbDraft.convId || ''
    convIn.title = '定时/执行时投递到哪个会话；留空用当前会话，定时时再回落到最近活跃会话'
    convIn.style.cssText = 'font:inherit;font-size:11px;color:var(--text-primary);background:var(--bg-primary);border:1px solid var(--border-color);border-radius:6px;padding:3px 7px;width:170px'
    convIn.addEventListener('input', () => { tbDraft.convId = convIn.value })
    opts.appendChild(convIn)
    box.appendChild(opts)
    if (tbDraft.flowId) {
      box.appendChild(el('div', 'ftb-note', '绑定流程后：流程门禁 = 这张卡的验收清单（DoD）；agent 执行时按门禁推进，流程走完自动转「待验收」。'))
    }

    const acts = el('div', 'ftb-row-actions')
    acts.appendChild(btn('创建任务', '建卡进入「待办」', async () => {
      if (!String(tbDraft.title || '').trim()) { tbMsg = '任务标题不能为空'; notify(); return }
      const r = await tbDo({
        op: 'create', title: tbDraft.title, description: tbDraft.description, prompt: tbDraft.prompt,
        urgency: tbDraft.urgency, flowId: tbDraft.flowId,
        convId: String(tbDraft.convId || '').trim() || currentConvId(),
      }, '已建卡')
      if (r && r.ok !== false) { tbNew = false; tbDraft = tbBlankDraft(); notify() }
    }, true))
    acts.appendChild(btn('取消', '不建卡', () => { tbNew = false; tbMsg = ''; notify() }))
    box.appendChild(acts)
    body.appendChild(box)
  }

  function tbPromptFromDesc(title, desc) {
    const d = String(desc || '').trim()
    const head = '请完成「' + String(title || '任务').trim() + '」。'
    if (!d) return head + '\n\n要求：动手实现并自行验证；结束时用 taskboard_report 交报告（summary / files / risks）。'
    return head + '\n\n' + d + '\n\n要求：\n1. 按上面的描述直接动手，不要只给建议。\n2. 先调用 taskboard_plan 写下执行拆解。\n3. 完成后逐项勾选 DoD（taskboard_check），并用 taskboard_report 交报告。'
  }

  // ── 详情：执行条（认领/执行/定时）+ 描述/提示词（多行一对一）+ 拆解 + DoD + 记录 ──
  function renderTaskDetail(body, t) {
    const box = el('div', 'ftb-detail')
    const rs = t.runState || {}
    const head = el('div', 'ftb-detail-head')
    head.appendChild(el('span', 'ftb-detail-title', t.title || '(无标题)'))
    head.appendChild(badge(t.status === 'done' ? 'done' : (t.status === 'in_progress' ? 'active' : 'todo'), TB_LABEL[t.status] || t.status))
    if (t.demo) head.appendChild(el('span', 'ftb-badge', 'demo'))
    head.appendChild(el('span', 'ftb-badge', TB_URG[t.urgency] || '一般'))
    if (rs.running) head.appendChild(el('span', 'ftb-badge active', '执行中'))
    if (t.blocked) head.appendChild(el('span', 'ftb-badge fail', '阻塞'))
    head.appendChild(el('div', 'ftb-spacer'))
    head.appendChild(btn('关闭', '收起详情', () => { tbDetail = ''; notify() }))
    box.appendChild(head)
    const metaLine = el('div', 'ftb-detail-meta', '项目 ' + (t.workspace || '-') + ' · 创建 ' + tbShort(t.createdAt) +
      (t.status === 'in_progress' ? (' · 认领 ' + tbWho(t)) : '') + ' · ' + t.id)
    const metaTitle = tbWhoTitle(t)
    if (metaTitle) metaLine.title = metaTitle
    box.appendChild(metaLine)

    // ── 执行条：认领 / 执行 / 状态 ──
    const exec = el('div', 'ftb-exec')
    if (t.status === 'in_progress') {
      exec.appendChild(btn('释放认领', '回到「待办」并清空认领', () => tbDo({ op: 'release', id: t.id }, '已释放'), false))
    } else if (t.status === 'todo' || t.status === 'backlog') {
      exec.appendChild(btn('认领', '接手这张卡（移到「进行中」）', () => tbDo({ op: 'claim', id: t.id, convId: currentConvId() }, '已认领'), false))
    }
    const canRun = t.status !== 'done' && t.status !== 'archived' && t.status !== 'canceled' && !rs.running
    exec.appendChild(btn('执行', '把任务书投递到当前会话，由 agent 按卡执行（可配合监督者多轮驱动）', () => tbRun(t.id), canRun))
    const state = el('div', 'ftb-exec-state' + (rs.running ? ' run' : (rs.lastError ? ' err' : '')))
    state.textContent = rs.running
      ? ('执行中… 开始于 ' + tbShort(rs.startedAt) + (rs.sessionId ? ' · 会话 ' + String(rs.sessionId).slice(0, 16) : ''))
      : (rs.lastError ? ('上次执行失败：' + rs.lastError) : (rs.lastEndAt ? ('上次执行完成 ' + tbShort(rs.lastEndAt) + ' · 共 ' + (rs.runCount || 0) + ' 次') : '尚未执行'))
    exec.appendChild(state)
    if (rs.sessionId) {
      exec.appendChild(btn('监督回合', '拉取该执行会话的监督者裁决记录', () => {
        ui.http.get('/autopilot/rounds', { convId: rs.sessionId })
          .then((r) => { tbRoundCache[rs.sessionId] = r || null; notify() })
          .catch(() => { tbRoundCache[rs.sessionId] = { ok: false, error: '监督者插件未启用或接口不可用' }; notify() })
      }))
    }
    box.appendChild(exec)

    // ── 定时执行条 ──
    const sch = t.schedule || { enabled: false, mode: 'interval', everyMinutes: 60, atTime: '09:00', atISO: '' }
    const srow = el('div', 'ftb-sched')
    srow.appendChild(el('span', 'ftb-pair-label', '定时'))
    srow.appendChild(btn(sch.enabled ? '已开启' : '已关闭', '开/关定时执行', () => {
      const next = Object.assign({}, sch, { enabled: !sch.enabled })
      tbDo({ op: 'schedule', id: t.id, schedule: next }, next.enabled ? '定时已开启' : '定时已关闭')
    }, sch.enabled))
    const modeSel = document.createElement('select')
    ;[['interval', '每 N 分钟'], ['daily', '每天'], ['at', '一次']].forEach(([v, label]) => {
      const o = document.createElement('option')
      o.value = v
      o.textContent = label
      if (sch.mode === v) o.selected = true
      modeSel.appendChild(o)
    })
    srow.appendChild(modeSel)
    const valWrap = el('span')
    const valInput = document.createElement('input')
    if (sch.mode === 'interval') {
      valInput.type = 'number'
      valInput.min = '1'
      valInput.value = String(sch.everyMinutes || 60)
      valInput.placeholder = '分钟'
      valInput.style.width = '64px'
    } else if (sch.mode === 'daily') {
      valInput.type = 'time'
      valInput.value = sch.atTime || '09:00'
    } else {
      valInput.type = 'text'
      valInput.value = sch.atISO || ''
      valInput.placeholder = '2026-10-18T09:00'
      valInput.style.width = '160px'
    }
    valWrap.appendChild(valInput)
    srow.appendChild(valWrap)
    // 执行会话（可选）：定时到点投递到哪个会话；留空 = 最近活跃会话 → subagent 兜底
    const convIn = document.createElement('input')
    convIn.type = 'text'
    convIn.placeholder = '执行会话（留空=自动）'
    convIn.value = sch.convId || ''
    convIn.title = '定时到点的投递会话；留空用最近活跃会话；仍无则走 subagent 兜底'
    convIn.style.width = '150px'
    srow.appendChild(convIn)
    srow.appendChild(btn('保存定时', '按当前模式与值保存', () => {
      const next = Object.assign({}, sch, { enabled: true, mode: modeSel.value, convId: String(convIn.value || '').trim() })
      if (modeSel.value === 'interval') next.everyMinutes = Math.max(1, parseInt(valInput.value || '60', 10) || 60)
      if (modeSel.value === 'daily') next.atTime = valInput.value || '09:00'
      if (modeSel.value === 'at') next.atISO = String(valInput.value || '').trim()
      tbDo({ op: 'schedule', id: t.id, schedule: next }, '定时已保存')
    }))
    modeSel.addEventListener('change', () => {
      sch.mode = modeSel.value
      notify()
    })
    if (sch.enabled && sch.nextAt) srow.appendChild(el('span', 'ftb-sched-tag', '下次 ' + tbShort(sch.nextAt)))
    box.appendChild(srow)
    box.appendChild(el('div', 'ftb-note', '定时执行按「指定执行会话 → 卡上上次会话 → 最近活跃会话」投递；都没有时走 subagent 兜底。未执行过的新卡也能到点自动跑。'))

    // ── 描述 + 执行提示词（一一对应，多行；本地草稿 + 防抖 600ms 保存，输入不重渲染）──
    const content = el('div', 'ftb-sec')
    const chead = el('div', 'ftb-sec-head')
    chead.appendChild(el('span', 'ftb-sec-title', '任务内容'))
    chead.appendChild(el('span', 'ftb-sec-hint', '输入后 600ms 自动保存 · 失焦立即保存 · 两栏都会进入执行任务书'))
    content.appendChild(chead)
    const pair = el('div', 'ftb-pair')
    const d1 = el('div', 'ftb-pair-item')
    d1.appendChild(el('div', 'ftb-pair-label', '描述'))
    const taD = tbTextarea(tbEditSeed(t, 'description').description, '背景、范围、验收标准…（多行）', 4, '')
    tbEditBind(t, 'description', taD)
    d1.appendChild(taD)
    pair.appendChild(d1)
    const d2 = el('div', 'ftb-pair-item')
    const lab2 = el('div', 'ftb-pair-label')
    lab2.appendChild(el('span', null, '执行提示词'))
    const genBtn = el('button', 'ftb-mini', '用描述生成')
    genBtn.type = 'button'
    genBtn.title = '用「描述（含未保存草稿） + 标题」重写执行提示词'
    genBtn.addEventListener('click', () => {
      const stD = tbEditState(t.id)
      const descNow = stD.description != null ? stD.description : String(t.description || '')
      tbDo({ op: 'update', id: t.id, patch: { prompt: tbPromptFromDesc(t.title, descNow) } }, '已用描述生成执行提示词')
    })
    lab2.appendChild(genBtn)
    d2.appendChild(lab2)
    const taP = tbTextarea(tbEditSeed(t, 'prompt').prompt, '告诉 agent 怎么干：步骤、边界、产物、验证方式…（多行）', 5, 'prompt')
    tbEditBind(t, 'prompt', taP)
    d2.appendChild(taP)
    pair.appendChild(d2)
    content.appendChild(pair)
    box.appendChild(content)

    // ── 执行拆解（agent 写，人可点状态推进）──
    const subs = Array.isArray(t.subtasks) ? t.subtasks : []
    const ssec = el('div', 'ftb-sec')
    const shead = el('div', 'ftb-sec-head')
    shead.appendChild(el('span', 'ftb-sec-title', '执行拆解'))
    const st = tbSubStats(subs)
    shead.appendChild(el('span', 'ftb-sec-hint', subs.length ? (st.done + '/' + st.total + ' 完成') : '执行时由 agent 自动拆解（taskboard_plan）'))
    ssec.appendChild(shead)
    if (subs.length) {
      const list = el('div', 'ftb-subs')
      subs.forEach((s) => {
        const row = el('div', 'ftb-sub ' + s.status)
        row.title = '点击切换：待办 → 进行中 → 已完成'
        row.appendChild(el('span', 'ftb-sub-dot'))
        row.appendChild(el('span', null, s.text))
        row.appendChild(el('span', 'ftb-sec-hint', TB_SUB_LABEL[s.status] || s.status))
        row.addEventListener('click', () => {
          const cycle = { pending: 'in_progress', in_progress: 'completed', completed: 'pending', blocked: 'pending', cancelled: 'pending' }
          const next = subs.map((x) => x.id === s.id ? Object.assign({}, x, { status: cycle[x.status] || 'pending' }) : x)
          tbDo({ op: 'plan', id: t.id, subtasks: next }, '拆解状态已更新')
        })
        list.appendChild(row)
      })
      ssec.appendChild(list)
    }
    box.appendChild(ssec)

    // ── 验收清单（DoD）──
    const cl = t.checklist || []
    if (cl.length) {
      const dsec = el('div', 'ftb-sec')
      const dhead = el('div', 'ftb-sec-head')
      dhead.appendChild(el('span', 'ftb-sec-title', '验收清单（DoD）'))
      const doneN = cl.filter((x) => x.checked).length
      dhead.appendChild(el('span', 'ftb-sec-hint', doneN + '/' + cl.length + (doneN === cl.length ? ' · 全部完成' : ' · ' + (cl.length - doneN) + ' 项未完成')))
      dsec.appendChild(dhead)
      const ul = el('ul', 'ftb-clist')
      cl.forEach((it) => {
        const li = el('li', it.checked ? 'done' : '')
        const cb = document.createElement('input')
        cb.type = 'checkbox'
        cb.checked = !!it.checked
        cb.addEventListener('change', () => { tbDo({ op: 'check', id: t.id, itemId: it.id, checked: cb.checked }, '清单已更新') })
        li.appendChild(cb)
        li.appendChild(el('span', null, String(it.text || '')))
        if (it.by === 'flow') li.appendChild(el('span', 'ftb-note', ' · 流程回填'))
        if (it.evidence) li.appendChild(el('span', null, '（证据：' + it.evidence + '）'))
        ul.appendChild(li)
      })
      dsec.appendChild(ul)
      box.appendChild(dsec)
    }

    // ── 状态机 / 标记 ──
    const acts = el('div', 'ftb-row-actions')
    const moves = TB_TRANS[t.status] || []
    moves.forEach((to) => {
      acts.appendChild(btn('移到「' + TB_LABEL[to] + '」', '按状态机迁移', () => tbDo({ op: 'move', id: t.id, to: to, convId: currentConvId() }, '已移到「' + TB_LABEL[to] + '」'), to === 'done'))
    })
    if (t.flowId && !(t.flowProgress && t.flowProgress.runId)) {
      acts.appendChild(btn('启动流程', '用绑定模板启动流程实例；门禁即 DoD', () => tbDo({ op: 'startFlow', id: t.id, convId: currentConvId() }, '流程已启动'), true))
    }
    acts.appendChild(btn('紧急度：' + (TB_URG[t.urgency] || '一般'), '在 紧急 / 一般 / 不急 间切换', () => {
      const cycle = { urgent: 'normal', normal: 'relaxed', relaxed: 'urgent' }
      tbDo({ op: 'update', id: t.id, patch: { urgency: cycle[t.urgency] || 'normal' } }, '紧急度已更新')
    }))
    acts.appendChild(btn(t.blocked ? '解除阻塞' : '标记阻塞', '当前做不下去', () => tbDo({ op: 'update', id: t.id, patch: { blocked: !t.blocked } }, '已更新')))
    acts.appendChild(btn('删除', '从看板移除该卡', () => {
      if (typeof window.confirm === 'function' && !window.confirm('删除任务「' + t.title + '」？')) return
      tbDo({ op: 'trash', id: t.id }, '已删除')
      tbDetail = ''
    }))
    box.appendChild(acts)

    // ── 评论 ──
    const cs = t.comments || []
    if (cs.length) {
      const csec = el('div', 'ftb-sec')
      csec.appendChild(el('div', 'ftb-sec-title', '评论'))
      cs.slice(-10).forEach((c) => {
        const who = (c.by && c.by.kind === 'agent') ? 'agent' : 'user'
        csec.appendChild(el('div', 'ftb-note ' + who, (who === 'agent' ? 'agent：' : '我：') + String(c.text || '') + '（' + tbShort(c.at) + '）'))
      })
      box.appendChild(csec)
    }
    const cf = el('div', 'ftb-row-actions')
    const cin = document.createElement('input')
    cin.type = 'text'
    cin.placeholder = '加一条评论（回车提交）'
    cin.style.cssText = 'flex:1;min-width:200px;font:inherit;font-size:12px;color:var(--text-primary);background:var(--bg-primary);border:1px solid var(--border-color);border-radius:8px;padding:6px 10px;outline:none'
    cin.addEventListener('keydown', (e) => {
      if ((e && e.key) === 'Enter' && String(cin.value || '').trim()) tbDo({ op: 'comment', id: t.id, text: cin.value }, '已评论')
    })
    cf.appendChild(cin)
    cf.appendChild(btn('评论', '提交评论', () => {
      if (String(cin.value || '').trim()) tbDo({ op: 'comment', id: t.id, text: cin.value }, '已评论')
    }))
    box.appendChild(cf)

    // ── 执行记录 + 监督回合 ──
    const exs = t.executions || []
    const esec = el('div', 'ftb-sec')
    esec.appendChild(el('div', 'ftb-sec-title', '执行记录'))
    if (!exs.length) esec.appendChild(el('div', 'ftb-note', '还没有执行记录 —— 点上方「执行」派发给 agent。'))
    exs.slice().reverse().slice(0, 5).forEach((x) => {
      esec.appendChild(el('div', 'ftb-note agent',
        '· ' + String(x.outcome || '') + ' @' + tbShort(x.at) + '：' + String(x.summary || '').slice(0, 500) +
        (x.risks ? ('（风险：' + x.risks + '）') : '') + ((x.files || []).length ? ('（改动 ' + x.files.length + ' 文件）') : '')))
    })
    if (rs.sessionId && tbRoundCache[rs.sessionId]) {
      const rc = tbRoundCache[rs.sessionId]
      const rounds = (rc && rc.rounds) || []
      if (!rc.ok) esec.appendChild(el('div', 'ftb-note', '监督回合：' + (rc.error || '不可用')))
      else if (!rounds.length) esec.appendChild(el('div', 'ftb-note', '监督回合：该会话暂无记录（需在设置里启用自主模式）'))
      else {
        esec.appendChild(el('div', 'ftb-note', '监督回合（最近 ' + rounds.length + ' 条）：'))
        rounds.slice(-3).reverse().forEach((r) => {
          const v = r.verdict || {}
          esec.appendChild(el('div', 'ftb-note agent', '· 第 ' + r.round + ' 轮 · ' + String(v.action || r.action || '') +
            (v.assessment ? (' · ' + String(v.assessment).slice(0, 160)) : '') +
            (v.nextTask ? (' → ' + String(v.nextTask).slice(0, 160)) : '')))
        })
      }
    }
    box.appendChild(esec)
    body.appendChild(box)
  }

  // ── 视图挂载（面板主体 = 任务看板；流程库并入顶部）──
  ui.registerView({
    id: VIEW_ID,
    title: VIEW_TITLE,
    icon: 'eye',
    order: 20,
    open: false,
    render(host) {
      host.textContent = ''
      const style = document.createElement('style')
      style.textContent = CSS + '\n' + TB_CSS
      host.appendChild(style)

      const root = el('div', 'ftb-root')
      const head = el('div', 'ftb-head')
      const title = el('span', 'ftb-title', '任务看板')
      const count = el('span', 'ftb-count')
      const sub = el('span', 'ftb-sub')
      const spacer = el('div', 'ftb-spacer')
      const refresh = btn('刷新', '重新拉取看板状态', () => { loadBoard(); if (tbTab === 'library') load(true) })
      head.appendChild(title)
      head.appendChild(count)
      head.appendChild(sub)
      head.appendChild(spacer)
      head.appendChild(refresh)

      const tabs = el('div', 'ftb-tabs')
      const TAB_DEFS = [['board', '任务看板'], ['library', '流程库']]
      const tabBtns = {}
      TAB_DEFS.forEach(([id, label]) => {
        const b = el('button', 'ftb-tab' + (tbTab === id ? ' active' : ''), label)
        b.type = 'button'
        b.addEventListener('click', () => { tbTab = id; if (editor) editor.dirty = true; draw() })
        tabBtns[id] = b
        tabs.appendChild(b)
      })

      const msgBox = el('div', 'ftb-msg')
      const body = el('div', 'ftb-body')
      root.appendChild(head)
      root.appendChild(tabs)
      root.appendChild(msgBox)
      root.appendChild(body)
      host.appendChild(root)

      const draw = () => {
        TAB_DEFS.forEach(([id]) => { tabBtns[id].className = 'ftb-tab' + (tbTab === id ? ' active' : '') })
        const rows = tbTasks()
        const doing = rows.filter((t) => t.status === 'in_progress').length
        const review = rows.filter((t) => t.status === 'in_review').length
        const running = rows.filter((t) => t.runState && t.runState.running).length
        count.textContent = tbSnap ? (rows.length + ' 任务 · rev ' + (tbSnap.revision || 0)) : ''
        sub.textContent = tbSnap
          ? ('进行中 ' + doing + ' · 待验收 ' + review + (running ? ' · 执行中 ' + running : '') + ' · 项目 ' + (tbSnap.workspace || '-'))
          : '加载中…'
        msgBox.textContent = tbMsg || message || ''
        msgBox.style.display = (tbMsg || message) ? '' : 'none'
        if (tbTab === 'library') {
          if (editor && !editor.dirty) return
          if (editor) editor.dirty = false
        }
        // ★ 输入保护（监督者实测缺陷）：4s 轮询 loadBoard→notify 会重建面板，替换正在输入的
        //   textarea（焦点丢失 + 丢字）。焦点落在本视图的编辑控件上时只更新头部数据，不重建 body。
        const ae = (typeof document !== 'undefined' && document.activeElement) || null
        const editing = !!(ae && body && typeof body.contains === 'function' && body.contains(ae) &&
          /^(TEXTAREA|INPUT|SELECT)$/.test(String(ae.tagName || '')))
        if (editing) return
        body.textContent = ''
        try {
          if (tbTab === 'library') {
            if (!snap) load(true)
            renderLibrary(body)
          } else {
            renderTaskBoard(body)
          }
        } catch (e) {
          const stk = String((e && e.stack) || '').split('\n').slice(0, 3).join(' / ')
          body.appendChild(el('div', 'ftb-err', '看板渲染失败：' + String((e && e.message) || e) + ' @ ' + stk))
          try { console.warn('[tool-flow] 看板渲染异常', e) } catch (e2) { }
        }
      }

      const off = subscribe(draw)
      draw()
      loadBoard()
      const timer = setInterval(() => {
        if (tbTab === 'library') { load(false); return }
        loadBoard()
      }, POLL_MS)

      return () => {
        off()
        clearInterval(timer)
      }
    },
  })
}
