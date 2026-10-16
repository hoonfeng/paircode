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
  const VIEW_TITLE = '流程'
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

  const STATUS_LABEL = { pass: 'PASS', approved: 'APPROVED', pending: 'PENDING', fail: 'FAIL', todo: 'TODO' }
  const STAGE_LABEL = { completed: 'DONE', in_progress: 'ACTIVE', pending: 'TODO', cancelled: 'CANCELLED' }
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
    const sp = run.stageProgress || { idx: 0, total: 1 }
    const total = sp.total || 1
    const done = (snap.stages || []).filter(s => s.status === 'completed').length
    const bar = el('div', 'ftb-bar')
    const fill = el('div', 'ftb-bar-fill')
    fill.style.width = Math.round((done / total) * 100) + '%'
    bar.appendChild(fill)
    body.appendChild(bar)

    const info = el('div', 'ftb-note', '实例 ' + run.runId + ' · 流程 ' + run.flowId + '（' + (run.flowScope || '?') + ' 级 v' + (run.flowVersion || 1) + '）· 状态 ' + run.status +
      (run.gitHead ? (' · git 基线 ' + run.gitHead) : '') +
      ((run.modifiedFiles || []).length ? (' · 已改动 ' + run.modifiedFiles.length + ' 个文件') : ''))
    body.appendChild(info)

    ;(snap.stages || []).forEach((st, i) => {
      const card = el('div', 'ftb-stage' + (st.status === 'in_progress' ? ' active' : ''))
      const head = el('div', 'ftb-stage-head')
      head.appendChild(el('span', 'ftb-idx', String(i + 1) + '/' + total))
      head.appendChild(el('span', 'ftb-name', st.title || st.id))
      const sc = st.status === 'completed' ? 'done' : (st.status === 'in_progress' ? 'active' : 'todo')
      head.appendChild(badge(sc, STAGE_LABEL[st.status] || st.status))
      if (st.completedAt) head.appendChild(el('span', 'ftb-cell', '完成于 ' + st.completedAt))
      card.appendChild(head)
      if (st.goal) card.appendChild(el('div', 'ftb-goal', st.goal))
      if ((st.tasks || []).length) card.appendChild(el('div', 'ftb-goal', '任务：' + st.tasks.join('；')))

      ;(st.gates || []).forEach(g => {
        const row = el('div', 'ftb-gate')
        row.appendChild(el('span', 'ftb-gate-id', g.id))
        row.appendChild(el('span', 'ftb-gate-kind', g.kind + (g.required ? '·必过' : '') + (g.blocking ? '·阻塞' : '')))
        row.appendChild(el('span', 'ftb-gate-text', g.text || ''))
        row.appendChild(gateBadge(g))

        // 门禁动作（写操作一律走 RPC）
        const isPassed = g.status === 'pass' || g.status === 'approved'
        if (!isPassed && run.status === 'active') {
          if (g.kind === 'command') {
            row.appendChild(btn('代跑', '执行 ' + (g.cmd || '') + ' 并留证' + (g.cmd ? '' : '（未声明 cmd）'), () => {
              act({ op: 'advance', runId: run.runId, gateId: g.id }, '执行门禁 ' + g.id + '…')
            }, true))
          } else if (g.kind === 'approval') {
            const b = btn('批准', '审批卡点：由用户显式批准（会记 approvedBy=user）', () => {
              if (!window.confirm('确认批准门禁 ' + g.id + '？\n' + (g.text || ''))) return
              act({ op: 'approve', runId: run.runId, gateId: g.id, approve: true, note: '看板批准' }, '批准 ' + g.id + '…')
            }, true)
            row.appendChild(b)
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
          const pre = el('pre', 'ftb-pre', (g.summary || '') + (g.evidenceFile ? ('\n文件：' + g.evidenceFile) : '') + (g.by ? ('\n批准人：' + g.by + (g.note ? ('（' + g.note + '）') : '')) : ''))
          d.appendChild(pre)
          card.appendChild(d)
        }
      })

      if (st.status === 'in_progress' && run.status === 'active') {
        const foot = el('div', 'ftb-gate')
        foot.appendChild(btn('推进阶段', '检查本阶段必过门禁；全部通过则进入下一阶段', () => {
          act({ op: 'advance', runId: run.runId }, '推进阶段…')
        }, true))
        foot.appendChild(el('span', 'ftb-cell', (sp.pending ? ('未过必过门禁 ' + sp.pending + ' 个') : '门禁已清')))
        card.appendChild(foot)
      }
      body.appendChild(card)
    })
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
      if (Array.isArray(st.tasks)) block.appendChild(field('任务', lock(txtInput(st.tasks.join('；'), (v) => { st.tasks = splitList(v) }), ro), '「；」分隔多条：进入该阶段时按条建宿主任务'))

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
      stages.push({ id: 'stage-' + (stages.length + 1), title: '新阶段', goal: '', tasks: [], gates: [] })
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

  // ── 视图挂载 ─────────────────────────────────────────────────
  ui.registerView({
    id: VIEW_ID,
    title: VIEW_TITLE,
    icon: 'eye',
    order: 20,
    open: false,
    render(host) {
      host.textContent = ''
      const style = document.createElement('style')
      style.textContent = CSS
      host.appendChild(style)

      const root = el('div', 'ftb-root')
      const head = el('div', 'ftb-head')
      const title = el('span', 'ftb-title', '工程流程')
      const sub = el('span', 'ftb-sub')
      const spacer = el('div', 'ftb-spacer')
      const refresh = btn('刷新', '重新拉取 GET /api/flow/state', () => load(true))
      head.appendChild(title)
      head.appendChild(sub)
      head.appendChild(spacer)
      head.appendChild(refresh)

      const tabs = el('div', 'ftb-tabs')
      const TAB_DEFS = [['current', '当前流程'], ['matrix', '项目 × 流程'], ['library', '流程库']]
      const tabBtns = {}
      TAB_DEFS.forEach(([id, label]) => {
        const b = el('button', 'ftb-tab' + (tab === id ? ' active' : ''), label)
        b.type = 'button'
        b.addEventListener('click', () => { tab = id; if (editor) editor.dirty = true; draw() })
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
        TAB_DEFS.forEach(([id]) => { tabBtns[id].className = 'ftb-tab' + (tab === id ? ' active' : '') })
        const run = snap && snap.run
        sub.textContent = run
          ? (run.flowId + ' · ' + ((run.stageProgress && run.stageProgress.label) || '') + (run.status === 'active' ? '' : (' · ' + run.status)))
          : (snap ? '未接入流程' : '加载中…')
        msgBox.textContent = message || ''
        msgBox.style.display = message ? '' : 'none'
        // ★ 编辑态保护：编辑面板打开且无结构变化时不重绘 —— poll 每 4s 触发一次 load→notify，
        //   若重绘会重建输入控件，打断正在输入的表单/JSON（结构变化由 touchEditor() 标记）。
        if (editor && !editor.dirty) return
        if (editor) editor.dirty = false
        body.textContent = ''
        if (tab === 'matrix') renderMatrix(body)
        else if (tab === 'library') renderLibrary(body)
        else renderCurrent(body)
      }

      const off = subscribe(draw)
      draw()
      load(true)
      const timer = setInterval(() => {
        const conv = currentConvId()
        if (conv !== loadedConv) { load(true); return }
        load(false)
      }, POLL_MS)

      return () => {
        off()
        clearInterval(timer)
      }
    },
  })
}
