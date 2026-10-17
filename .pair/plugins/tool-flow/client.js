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
  // 任务看板（照 github.com/cloader/dsh-taskboard 重建，2026-10-17 用户决策）
  //   · 任务归**项目（工作区）**、不归会话；会话只出现在「认领」
  //   · 七态严格状态机（照参考 TRANSITIONS）：backlog→todo→in_progress→in_review→done
  //     加 canceled / archived；拖拽只允许合法迁移，非法时照参考提示「无法从「A」移到「B」」
  //   · 人建卡 → agent 认领 → 交报告 → **人验收**（done 只有人能放）
  //   · 结合流程：卡片可绑定流程模板；流程阶段进度显示在卡上，门禁 = 验收清单（DoD）
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
  const TB_CSS = [
    '.ftb-tb-bar{display:flex;align-items:center;gap:6px;flex-wrap:wrap;margin-bottom:8px}',
    '.ftb-tb-cols{display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:8px;align-items:start}',
    '.ftb-tb-cols.sec{grid-template-columns:repeat(2,minmax(0,1fr))}',
    '.ftb-tcol{display:flex;flex-direction:column;min-width:0;border:1px solid var(--border-color);border-radius:var(--border-radius);background:var(--bg-secondary)}',
    '.ftb-tcol.drop-on{border-color:#6a9955;border-style:dashed;background:var(--bg-active)}',
    '.ftb-tcol-head{display:flex;align-items:center;gap:6px;padding:5px 8px;border-bottom:1px solid var(--border-color);font-size:11px;color:var(--text-secondary);flex-shrink:0}',
    '.ftb-tcol-count{margin-left:auto;font-size:10px;color:var(--text-muted);font-variant-numeric:tabular-nums;border:1px solid var(--border-color);border-radius:8px;padding:0 5px}',
    '.ftb-tcol-body{display:flex;flex-direction:column;gap:6px;padding:6px;min-height:70px}',
    '.ftb-tcol-empty{color:var(--text-muted);font-size:10px;text-align:center;padding:10px 4px;line-height:1.6}',
    '.ftb-tcard{border:1px solid var(--border-color);border-left:3px solid var(--text-muted);border-radius:3px;background:var(--bg-primary);padding:6px 7px;cursor:pointer}',
    '.ftb-tcard:hover{background:var(--bg-active)}',
    '.ftb-tcard.dragging{opacity:.5}',
    '.ftb-tcard.u-urgent{border-left-color:var(--danger,#e5534b)}',
    '.ftb-tcard.u-normal{border-left-color:#569cd6}',
    '.ftb-tcard.blocked{outline:1px solid var(--danger,#e5534b);outline-offset:-1px}',
    '.ftb-tcard.sel{border-color:#569cd6}',
    '.ftb-tcard-title{font-size:12px;color:var(--text-primary);line-height:1.45;word-break:break-word}',
    '.ftb-tcard-meta{display:flex;gap:4px;flex-wrap:wrap;margin-top:4px}',
    '.ftb-tchip{font-size:10px;color:var(--text-muted);border:1px solid var(--border-color);border-radius:8px;padding:0 5px;line-height:1.5;flex-shrink:0}',
    '.ftb-tchip.flow{color:#569cd6}',
    '.ftb-tchip.blocked{color:var(--danger,#e5534b)}',
    '.ftb-tchip.review{color:#d4a74e}',
    '.ftb-badge.u-urgent{color:var(--danger,#e5534b)}',
    '.ftb-newbox,.ftb-detail{border:1px solid var(--border-color);border-radius:var(--border-radius);background:var(--bg-secondary);padding:8px 10px;margin-bottom:8px}',
    '.ftb-detail{border-left:3px solid #569cd6}',
    '.ftb-clist{margin:4px 0;padding:0;list-style:none}',
    '.ftb-clist li{display:flex;align-items:flex-start;gap:6px;font-size:11px;color:var(--text-secondary);line-height:1.6;padding:2px 0}',
    '.ftb-clist li.done{color:var(--text-muted);text-decoration:line-through}',
    '.ftb-tb-meta{font-size:11px;color:var(--text-muted);line-height:1.7;margin:4px 0}',
  ].join('\n')
  let tbSnap = null
  let tbTab = 'board'          // board | library
  let tbUrg = ''               // 紧急度筛选
  let tbSec = false            // false=主看板 true=已取消/已归档
  let tbDetail = ''            // 展开详情的任务 id
  let tbNew = false            // 新建任务表单是否展开
  let tbDraft = { title: '', description: '', urgency: 'normal', flowId: '' }
  let tbMsg = ''
  let tbDrag = null

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
  function tbTasks() { return (tbSnap && tbSnap.tasks) || [] }
  function tbIsSec(t) { return t.status === 'canceled' || t.status === 'archived' }
  function tbVisible() {
    let rows = tbTasks().filter((t) => tbSec ? tbIsSec(t) : !tbIsSec(t))
    if (tbUrg) rows = rows.filter((t) => t.urgency === tbUrg)
    return rows
  }
  function tbCanMove(from, to) { const t = TB_TRANS[from]; return !!(t && t.indexOf(to) >= 0) }
  function tbFlowLibrary() { return (tbSnap && tbSnap.flowLibrary) || [] }
  function shortTime2(iso) {
    const s = String(iso || '')
    const m = s.match(/T(\d{2}:\d{2})/)
    return m ? m[1] : s.slice(0, 10)
  }

  // 单张任务卡：紧急度左色条 + 标题 + 徽标行（流程进度/阻塞/认领/清单/执行结果）
  function tbCard(t, host) {
    const node = el('div', 'ftb-tcard u-' + (t.urgency || 'normal') + (t.blocked ? ' blocked' : '') + (tbDetail === t.id ? ' sel' : ''))
    node.draggable = true
    node.title = '拖到其他列改状态 · 点开看详情'
    node.addEventListener('dragstart', () => { tbDrag = t.id; node.className += ' dragging' })
    node.addEventListener('dragend', () => { tbDrag = null; node.className = node.className.replace(' dragging', '') })
    node.addEventListener('click', () => { tbDetail = (tbDetail === t.id) ? '' : t.id; tbNew = false; notify() })
    node.appendChild(el('div', 'ftb-tcard-title', t.title || '(无标题)'))
    const meta = el('div', 'ftb-tcard-meta')
    if (t.urgency && t.urgency !== 'normal') meta.appendChild(el('span', 'ftb-badge u-' + t.urgency, TB_URG[t.urgency] || t.urgency))
    if (t.flowId) {
      const fp = t.flowProgress || null
      const fpTxt = fp && fp.label && fp.label !== '未启动' ? (' · ' + fp.label) : ''
      meta.appendChild(el('span', 'ftb-tchip flow', '流程 ' + t.flowId + fpTxt))
    }
    if (t.blocked) meta.appendChild(el('span', 'ftb-tchip blocked', '阻塞'))
    if (t.claimedBy) meta.appendChild(el('span', 'ftb-tchip', '认领 ' + String(t.claimedBy).slice(0, 14)))
    const cl = t.checklist || []
    if (cl.length) {
      const done = cl.filter((x) => x.checked).length
      meta.appendChild(el('span', 'ftb-tchip', '清单 ' + done + '/' + cl.length))
    }
    const exs = t.executions || []
    if (exs.length) meta.appendChild(el('span', 'ftb-tchip', '执行 ' + (exs[exs.length - 1].outcome || '')))
    if ((t.comments || []).length) meta.appendChild(el('span', 'ftb-tchip', '💬 ' + t.comments.length))
    if (meta.children && meta.children.length) node.appendChild(meta)
    host.appendChild(node)
  }

  // 一列：列头（名称 + 计数）+ 卡片堆 + 空列占位；拖拽只放合法迁移
  function tbColumn(key, label, rows, container) {
    const col = el('div', 'ftb-tcol')
    const head = el('div', 'ftb-tcol-head')
    head.appendChild(el('span', null, label))
    head.appendChild(el('span', 'ftb-tcol-count', String(rows.length)))
    col.appendChild(head)
    const box = el('div', 'ftb-tcol-body')
    if (!rows.length) box.appendChild(el('div', 'ftb-tcol-empty', '无任务'))
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
      const t = tbTasks().filter((x) => x.id === id)[0]
      if (!t || t.status === key) return
      if (!tbCanMove(t.status, key)) {
        tbMsg = '无法从「' + (TB_LABEL[t.status] || t.status) + '」移到「' + (TB_LABEL[key] || key) + '」'
        notify()
        return
      }
      tbDo({ op: 'move', id: id, to: key }, '已移到「' + (TB_LABEL[key] || key) + '」')
    })
    container.appendChild(col)
  }

  // 主看板：工具栏（紧急度筛选 / 主·次要看板 / 新建）+ 五列
  function renderTaskBoard(body) {
    const bar = el('div', 'ftb-tb-bar')
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
    bar.appendChild(el('div', 'ftb-spacer'))
    bar.appendChild(btn(tbNew ? '取消新建' : '＋ 新建任务', '人建卡：填标题即可，可选绑定流程', () => {
      tbNew = !tbNew
      tbDetail = ''
      if (tbNew) tbDraft = { title: '', description: '', urgency: 'normal', flowId: '' }
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
        : '看板还没有任务 —— 点右上「＋ 新建任务」建一张卡（人建卡 → agent 认领执行 → 你验收）。'))
    }
    if (tbDetail) {
      const t = tbTasks().filter((x) => x.id === tbDetail)[0]
      if (t) renderTaskDetail(body, t)
    }
  }

  // 建卡表单（照参考的 composer：标题必填，其余可选；可选绑定流程）
  function renderNewTask(body) {
    const box = el('div', 'ftb-newbox')
    box.appendChild(el('div', 'ftb-name', '新建任务'))
    const f1 = el('div', 'ftb-field')
    f1.appendChild(el('span', 'ftb-label', '标题'))
    const inp = document.createElement('input')
    inp.type = 'text'
    inp.placeholder = '做什么（一句话任务名）'
    inp.value = tbDraft.title
    inp.addEventListener('input', () => { tbDraft.title = inp.value })
    f1.appendChild(inp)
    box.appendChild(f1)

    const f2 = el('div', 'ftb-field')
    f2.appendChild(el('span', 'ftb-label', '描述'))
    const inp2 = document.createElement('input')
    inp2.type = 'text'
    inp2.placeholder = '可选：背景 / 验收标准'
    inp2.value = tbDraft.description
    inp2.addEventListener('input', () => { tbDraft.description = inp2.value })
    f2.appendChild(inp2)
    box.appendChild(f2)

    const f3 = el('div', 'ftb-field')
    f3.appendChild(el('span', 'ftb-label', '紧急度'))
    const sel = document.createElement('select')
    ;[['urgent', '紧急'], ['normal', '一般'], ['relaxed', '不急']].forEach(([v, label]) => {
      const o = document.createElement('option')
      o.value = v
      o.textContent = label
      if (tbDraft.urgency === v) o.selected = true
      sel.appendChild(o)
    })
    sel.addEventListener('change', () => { tbDraft.urgency = sel.value })
    f3.appendChild(sel)
    box.appendChild(f3)

    const f4 = el('div', 'ftb-field')
    f4.appendChild(el('span', 'ftb-label', '流程'))
    const sel2 = document.createElement('select')
    const o0 = document.createElement('option')
    o0.value = ''
    o0.textContent = '不绑定流程'
    sel2.appendChild(o0)
    tbFlowLibrary().forEach((f) => {
      const o = document.createElement('option')
      o.value = f.id
      o.textContent = f.title + '（' + f.stages + ' 阶段 · ' + f.scope + '）'
      if (tbDraft.flowId === f.id) o.selected = true
      sel2.appendChild(o)
    })
    sel2.addEventListener('change', () => { tbDraft.flowId = sel2.value; notify() })
    f4.appendChild(sel2)
    box.appendChild(f4)
    if (tbDraft.flowId) {
      box.appendChild(el('div', 'ftb-note', '绑定流程后：流程门禁 = 这张卡的「验收清单（DoD）」；流程走完卡片自动转「待验收」，由你验收。'))
    }
    const acts = el('div', 'ftb-toolbar')
    acts.appendChild(btn('创建任务', '建卡进入「待办」列', async () => {
      if (!String(tbDraft.title || '').trim()) { tbMsg = '任务标题不能为空'; notify(); return }
      const r = await tbDo({ op: 'create', title: tbDraft.title, description: tbDraft.description, urgency: tbDraft.urgency, flowId: tbDraft.flowId }, '已建卡')
      if (r && r.ok !== false) { tbNew = false; tbDraft = { title: '', description: '', urgency: 'normal', flowId: '' }; notify() }
    }, true))
    acts.appendChild(btn('取消', '不建卡', () => { tbNew = false; tbMsg = ''; notify() }))
    box.appendChild(acts)
    body.appendChild(box)
  }

  // 任务详情：流程进度 / 验收清单（DoD）/ 评论 / 执行记录 / 状态机迁移按钮
  function renderTaskDetail(body, t) {
    const box = el('div', 'ftb-detail')
    const head = el('div', 'ftb-block-head')
    head.appendChild(el('span', 'ftb-name', t.title || '(无标题)'))
    head.appendChild(badge(t.status === 'done' ? 'done' : (t.status === 'in_progress' ? 'active' : 'todo'), TB_LABEL[t.status] || t.status))
    head.appendChild(el('span', 'ftb-badge', TB_URG[t.urgency] || '一般'))
    if (t.blocked) head.appendChild(el('span', 'ftb-badge fail', '阻塞'))
    head.appendChild(el('div', 'ftb-spacer'))
    head.appendChild(btn('关闭', '收起详情', () => { tbDetail = ''; notify() }))
    box.appendChild(head)

    const meta = el('div', 'ftb-tb-meta')
    meta.textContent = '项目 ' + (t.workspace || '-') + ' · 创建 ' + shortTime2(t.createdAt) +
      (t.claimedBy ? (' · 认领 ' + t.claimedBy) : '') + ' · ' + t.id
    box.appendChild(meta)
    if (t.description) box.appendChild(el('div', 'ftb-note', t.description))

    if (t.flowId) {
      const fp = t.flowProgress || {}
      box.appendChild(el('div', 'ftb-tb-meta', '流程：' + t.flowId + '（' + (t.flowScope || '-') + ' 级 v' + (t.flowVersion || 1) + '）· ' +
        (fp.label || '未启动') + (fp.percent != null ? (' · ' + fp.percent + '%') : '')))
      if (fp.runId) box.appendChild(el('div', 'ftb-note', '运行实例 ' + fp.runId + '（状态 ' + (fp.status || '') + '）'))
    }

    const cl = t.checklist || []
    if (cl.length) {
      const ch = el('div', 'ftb-block-head')
      ch.appendChild(el('span', 'ftb-name', '验收清单（DoD）'))
      const doneN = cl.filter((x) => x.checked).length
      ch.appendChild(el('span', 'ftb-count', doneN + '/' + cl.length + (doneN === cl.length ? ' · 全部完成' : ' · ' + (cl.length - doneN) + ' 项未完成')))
      box.appendChild(ch)
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
      box.appendChild(ul)
    }

    const acts = el('div', 'ftb-toolbar')
    const moves = TB_TRANS[t.status] || []
    moves.forEach((to) => {
      acts.appendChild(btn('移到「' + TB_LABEL[to] + '」', '按状态机迁移（只能走合法迁移）', () => tbDo({ op: 'move', id: t.id, to: to }, '已移到「' + TB_LABEL[to] + '」'), to === 'done'))
    })
    if (t.flowId && !(t.flowProgress && t.flowProgress.runId)) {
      acts.appendChild(btn('启动流程', '用绑定的流程模板启动一个实例；阶段进度会显示在卡片上', () => tbDo({ op: 'startFlow', id: t.id }, '流程已启动'), true))
    }
    acts.appendChild(btn('紧急度：' + (TB_URG[t.urgency] || '一般'), '在 紧急 / 一般 / 不急 之间切换', () => {
      const cycle = { urgent: 'normal', normal: 'relaxed', relaxed: 'urgent' }
      tbDo({ op: 'update', id: t.id, patch: { urgency: cycle[t.urgency] || 'normal' } }, '紧急度已更新')
    }))
    acts.appendChild(btn(t.blocked ? '解除阻塞' : '标记阻塞', '水平标记：现在做不下去', () => tbDo({ op: 'update', id: t.id, patch: { blocked: !t.blocked } }, '已更新')))
    acts.appendChild(btn('删除', '从看板移除该卡', () => {
      if (typeof window.confirm === 'function' && !window.confirm('删除任务「' + t.title + '」？')) return
      tbDo({ op: 'trash', id: t.id }, '已删除')
      tbDetail = ''
    }))
    box.appendChild(acts)

    const cs = t.comments || []
    if (cs.length) {
      box.appendChild(el('div', 'ftb-name', '评论'))
      cs.forEach((c) => {
        const who = (c.by && c.by.kind === 'agent') ? '🤖 ' : '👤 '
        box.appendChild(el('div', 'ftb-note', who + String(c.text || '') + '（' + shortTime2(c.at) + '）'))
      })
    }
    const cf = el('div', 'ftb-field')
    const cin = document.createElement('input')
    cin.type = 'text'
    cin.placeholder = '加一条评论（回车提交）'
    cin.addEventListener('keydown', (e) => {
      if ((e && e.key) === 'Enter' && String(cin.value || '').trim()) tbDo({ op: 'comment', id: t.id, text: cin.value }, '已评论')
    })
    cf.appendChild(cin)
    cf.appendChild(btn('评论', '提交评论', () => {
      if (String(cin.value || '').trim()) tbDo({ op: 'comment', id: t.id, text: cin.value }, '已评论')
    }))
    box.appendChild(cf)

    const exs = t.executions || []
    if (exs.length) {
      box.appendChild(el('div', 'ftb-name', '执行记录'))
      exs.slice().reverse().forEach((x) => {
        box.appendChild(el('div', 'ftb-note', '· ' + String(x.outcome || '') + ' @' + shortTime2(x.at) + '：' + String(x.summary || '') +
          (x.risks ? ('（风险：' + x.risks + '）') : '') + ((x.files || []).length ? ('（改动 ' + x.files.length + ' 文件）') : '')))
      })
    }
    body.appendChild(box)
  }

  // ── 视图挂载（2026-10-17 重建：面板主体 = 任务看板；旧流程视图下线，流程库并入顶部）──
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
      const refresh = btn('刷新', '重新拉取 GET /api/taskboard/state', () => { loadBoard(); if (tbTab === 'library') load(true) })
      head.appendChild(title)
      head.appendChild(count)
      head.appendChild(sub)
      head.appendChild(spacer)
      head.appendChild(refresh)

      const tabs = el('div', 'ftb-tabs')
      const TAB_DEFS = [['board', '任务看板'], ['library', '流程库']]
      const tabBtns = {}
      TAB_DEFS.forEach(([id, label]) => {
        const b = el('button', 'ftb-tab' + (tab === id ? ' active' : ''), label)
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
        count.textContent = tbSnap ? (rows.length + ' 任务 · rev ' + (tbSnap.revision || 0)) : ''
        sub.textContent = tbSnap
          ? ('进行中 ' + doing + ' · 待验收 ' + review + ' · 项目 ' + (tbSnap.workspace || '-'))
          : '加载中…'
        msgBox.textContent = tbMsg || message || ''
        msgBox.style.display = (tbMsg || message) ? '' : 'none'
        // ★ 编辑态保护：编辑面板打开且无结构变化时不重绘 —— poll 每 4s 触发一次 load→notify，
        //   若重绘会重建输入控件，打断正在输入的表单/JSON（结构变化由 touchEditor() 标记）。
        // 编辑态保护只作用于「流程库」页签：任务看板的重绘不受流程编辑器状态影响
        // （旧实现无条件 return，导致 ①-⑥ 段打开过编辑器后，任务看板再也画不出来）
        if (tbTab === 'library') {
          if (editor && !editor.dirty) return
          if (editor) editor.dirty = false
        }
        body.textContent = ''
        try {
          if (tbTab === 'library') {
            if (!snap) load(true)               // 流程库页签才拉流程快照（面板主体已不依赖它）
            renderLibrary(body)
          } else {
            renderTaskBoard(body)
          }
        } catch (e) {
          // 渲染异常不能导致「整块白板」：把错误显示在看板里（也便于测试定位）
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
