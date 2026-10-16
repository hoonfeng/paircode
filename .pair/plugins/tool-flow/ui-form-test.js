// ═══════════════════════════════════════════════════════════════
// tool-flow — D4-④ 看板「字段级表单编辑」UI 断言（DOM 桩驱动**真实** client.js）
//
// 目的：不装 jsdom、不开浏览器，用最小 DOM 桩执行真实 client.js，并走真实
//       host 半 RPC（rpcs['flow_action']）与真实快照接口（GET /api/flow/state），
//       断言以下验收点（监督者 D4-④）：
//         ① 默认表单模式（无需手写 JSON）
//         ② 纯表单新建 ≥2 阶段 ≥3 门禁的流程并保存成功（落盘 .pair/flows/）
//         ③ 表单可增删阶段 / 门禁、可上下移
//         ④ kind 收敛字段 + kind 缺字段（command 无 cmd / artifact 无产物）被拦
//         ⑤ 表单 ↔ JSON 双向切换不丢字段（表单未渲染字段如 board 原样保留）
//         ⑥ preset 只读层无「编辑」入口（只能「克隆到项目」后编辑）
//
// 用法：由 smoke-test.js 末尾调用（module.exports = async (check, env) => {}）。
//       独立运行不可行（需要 host 半 ctx 桩与 RPC），请跑 node smoke-test.js。
// ═══════════════════════════════════════════════════════════════
const fs = require('fs')
const path = require('path')

// ── 最小 DOM 桩（只实现 client.js 实际用到的语义）───────────────
function makeEl(tag) {
  const e = {
    tagName: String(tag).toUpperCase(), children: [], parent: null,
    className: '', value: '', title: '', type: '', placeholder: '',
    disabled: false, checked: false, selected: false, min: '', spellcheck: true,
    style: {}, _text: '', _listeners: {},
  }
  Object.defineProperty(e, 'textContent', {
    get() { return e._text },
    set(v) { e._text = String(v); e.children.length = 0 },   // 同浏览器：赋值即清空子节点
  })
  e.appendChild = (c) => { c.parent = e; e.children.push(c); return c }
  e.addEventListener = (ev, fn) => { (e._listeners[ev] = e._listeners[ev] || []).push(fn) }
  e.dispatch = (ev) => { for (const fn of (e._listeners[ev] || []).slice()) fn({ target: e }) }
  return e
}
function all(root, out) { out = out || []; if (!root) return out; out.push(root); for (const c of root.children) all(c, out); return out }
function nodes(root, pred) { return all(root).filter(pred) }
function byClass(root, cls) { return nodes(root, (n) => String(n.className).split(/\s+/).indexOf(cls) >= 0) }
function btnOf(root, label) { return root ? (nodes(root, (n) => n.tagName === 'BUTTON' && n._text === label)[0] || null) : null }
function btnsOf(root, label) { return root ? nodes(root, (n) => n.tagName === 'BUTTON' && n._text === label) : [] }
function textAll(root) { return nodes(root, () => true).map((n) => n._text).filter(Boolean).join(' | ') }
function rowOf(root, flowId) { return root ? (byClass(root, 'ftb-row').filter((r) => textAll(r).indexOf(flowId) >= 0)[0] || null) : null }
// 按 label 文本在子树内定位输入控件（比按顺序取更稳）
function fieldInput(scope, labelText) {
  for (const f of byClass(scope, 'ftb-field')) {
    const lab = nodes(f, (n) => String(n.className).indexOf('ftb-label') >= 0)[0]
    if (lab && lab._text === labelText) {
      return nodes(f, (n) => n.tagName === 'INPUT' || n.tagName === 'SELECT')[0] || null
    }
  }
  return null
}
function setVal(node, v) { node.value = v; node.dispatch('input') }
function setSel(node, v) { node.value = v; node.dispatch('change') }
// .ftb-block 同时用于「流程元信息块」与「阶段块」——按是否含「阶段 id」字段区分
function stageBlocks(root) { return byClass(root, 'ftb-block').filter((b) => !!fieldInput(b, '阶段 id')) }
function stageBlockByTitle(root, title) {
  return stageBlocks(root).filter((b) => { const i = fieldInput(b, '标题'); return !!i && i.value === title })[0] || null
}
// 按阶段 id 定位（标题会被测试改写，id 更稳）
function stageBlockById(root, id) {
  return stageBlocks(root).filter((b) => { const i = fieldInput(b, '阶段 id'); return !!i && i.value === id })[0] || null
}

module.exports = async function runUiFormTest(check, env) {
  const doc = { createElement: (t) => makeEl(t) }
  const toasts = []
  const win = {
    __state: { currentConvId: 'conv_form' },
    $toast: (text, type) => toasts.push({ text: text, type: type }),
    confirm: () => true,
  }
  let view = null
  const ui = {
    registerView: (v) => { view = v },
    http: { get: async (p, params) => env.snapshot(params && params.convId) },
    invoke: async (plugin, method, args) => env.rpc(args || {}),
  }
  const flush = async (n) => { for (let i = 0; i < (n || 4); i++) await new Promise((r) => setTimeout(r, 3)) }
  const noop = () => { }

  const code = fs.readFileSync(env.clientPath, 'utf8')
  const load = new Function('ui', 'window', 'document', 'setInterval', 'clearInterval', 'console',
    'return (' + code + ')(ui)')
  load(ui, win, doc, setInterval, clearInterval, { log: noop, warn: noop, error: noop })
  check('[UI] client.js 装载并注册看板视图（id=flow-board）', !!view && view.id === 'flow-board', view && view.id)

  const host = makeEl('div')
  const unmount = view.render(host)
  await flush()

  // ── ① 进入流程库页签 ─────────────────────────────────────────
  const libTab = btnOf(host, '流程库')
  check('[UI] 看板三页签渲染（当前流程 / 项目 × 流程 / 流程库）', !!libTab && !!btnOf(host, '当前流程') && !!btnOf(host, '项目 × 流程'))
  libTab.dispatch('click')
  await flush()
  const newBtn = btnOf(host, '＋ 新建流程')
  check('[UI] 流程库提供「＋ 新建流程」入口（纯表单落地路径）', !!newBtn)

  // ── ② 默认表单模式 + 空白骨架 ───────────────────────────────
  newBtn.dispatch('click')
  await flush()
  const seg = byClass(host, 'ftb-seg')[0]
  const formTab = seg ? btnOf(seg, '表单') : null
  check('[UI] 默认进入表单模式（有「表单 / JSON」切换，表单态无 textarea）',
    !!seg && !!formTab && /active/.test(formTab.className) && byClass(host, 'ftb-textarea').length === 0 && !!btnOf(seg, 'JSON'),
    seg ? textAll(seg) : '（无切换控件）')
  check('[UI] 空白骨架渲染为 1 阶段 / 1 门禁的表单块',
    stageBlocks(host).length === 1 && byClass(host, 'ftb-gate-form').length === 1,
    'stages=' + stageBlocks(host).length + ' gates=' + byClass(host, 'ftb-gate-form').length)

  // ── ③ 表单增删：加阶段、上下移、加门禁、删门禁 ───────────────
  btnOf(host, '＋ 加阶段').dispatch('click')
  await flush()
  check('[UI] 表单「＋ 加阶段」→ 阶段块 2 个', stageBlocks(host).length === 2, 'stages=' + stageBlocks(host).length)

  const sbs = stageBlocks(host)
  const downBtn = btnOf(sbs[sbs.length - 1], '↓')
  check('[UI] 阶段块提供上下移按钮（↑/↓）', !!btnOf(sbs[0], '↑') && !!downBtn)
  downBtn.dispatch('click')                                  // 末尾阶段下移 → 无操作
  await flush()
  check('[UI] 末尾阶段下移为无操作（阶段数不变）', stageBlocks(host).length === 2)
  // ★ 用标题定位（而非索引）做移动断言，避免索引错位造成的假通过/假失败
  btnOf(stageBlockByTitle(host, '新阶段'), '↑').dispatch('click')
  await flush()
  const firstTitle = fieldInput(stageBlocks(host)[0], '标题')
  check('[UI] 阶段「↑」上移生效（新阶段移到首位）', !!firstTitle && firstTitle.value === '新阶段', firstTitle ? firstTitle.value : '')
  btnOf(stageBlockByTitle(host, '新阶段'), '↓').dispatch('click')
  await flush()
  const backTitle = fieldInput(stageBlocks(host)[1], '标题')
  check('[UI] 阶段「↓」下移恢复原序（新阶段回到末位）', !!backTitle && backTitle.value === '新阶段', backTitle ? backTitle.value : '')

  const addGates = btnsOf(host, '＋ 加门禁')
  check('[UI] 每个阶段各有一个「＋ 加门禁」', addGates.length === 2, 'count=' + addGates.length)
  // 新加的阶段模板 id 即 'stage-2'；用 id 定位（标题随后会被改写）
  const s2 = () => stageBlockById(host, 'stage-2')
  const s2gates = () => byClass(s2(), 'ftb-gate-form')
  btnOf(s2(), '＋ 加门禁').dispatch('click')
  await flush()
  btnOf(s2(), '＋ 加门禁').dispatch('click')
  await flush()
  check('[UI] 表单「＋ 加门禁」×2 → 门禁块合计 3 个（≥3 门禁）', byClass(host, 'ftb-gate-form').length === 3, 'gates=' + byClass(host, 'ftb-gate-form').length)

  // 删除一个门禁（验证「可删」）后再加回，最终仍为 3
  btnOf(s2gates()[1], '删除门禁').dispatch('click')
  await flush()
  check('[UI] 「删除门禁」生效 → 门禁块 2 个', byClass(host, 'ftb-gate-form').length === 2, 'gates=' + byClass(host, 'ftb-gate-form').length)
  btnOf(s2(), '＋ 加门禁').dispatch('click')
  await flush()
  check('[UI] 删后再加 → 门禁块回到 3 个', byClass(host, 'ftb-gate-form').length === 3)

  // ── ④ 逐字段填写（含 kind 收敛）──────────────────────────────
  setVal(fieldInput(s2(), '阶段 id'), 'stage-2')
  setVal(fieldInput(s2(), '标题'), '实施')
  setVal(fieldInput(s2gates()[0], '门禁 id'), 'g-2-1')
  setVal(fieldInput(s2gates()[0], '确认说明'), '实现完成并自测通过')
  setSel(fieldInput(s2gates()[0], 'kind'), 'command')        // 切 kind → 表单重绘
  await flush()
  check('[UI] kind 收敛：切到 command 后出现「命令 cmd」字段且无「产物路径」',
    !!fieldInput(s2gates()[0], '命令 cmd') && !fieldInput(s2gates()[0], '产物路径'))
  setSel(fieldInput(s2gates()[1], 'kind'), 'artifact')
  await flush()
  check('[UI] kind 收敛：切到 artifact 后出现「产物路径」字段', !!fieldInput(s2gates()[1], '产物路径'))

  // kind 缺字段 → 本地校验拦下、不落盘
  const target = path.join(env.ws, '.pair', 'flows', 'my-flow.json')
  btnOf(host, '保存').dispatch('click')
  await flush()
  const errText = textAll(byClass(host, 'ftb-editor')[0] || host)
  check('[UI] kind 缺字段被拦：command 未填 cmd → 提示且未落盘',
    /必须填「命令 cmd」/.test(errText) && !fs.existsSync(target),
    errText.slice(0, 220))
  check('[UI] 校验失败时给出可读错误（.ftb-err 错误样式）', byClass(host, 'ftb-err').length >= 1)

  // 补齐必填 → 保存成功（纯表单新建）
  setVal(fieldInput(s2gates()[0], '命令 cmd'), 'go test ./...')
  setVal(fieldInput(s2gates()[1], '门禁 id'), 'g-2-2')
  setVal(fieldInput(s2gates()[1], '产物路径'), 'dist/app.exe')
  btnOf(host, '保存').dispatch('click')
  await flush(8)
  if (!fs.existsSync(target)) {
    const eb = byClass(host, 'ftb-editor')[0]
    console.log('  [debug] 未落盘 → 编辑器面板文本：' + (eb ? textAll(eb) : '（无编辑器面板）').slice(0, 500))
  }
  check('[UI] 纯表单新建保存成功（落盘项目层 .pair/flows/my-flow.json）', fs.existsSync(target), target)
  const saved = fs.existsSync(target) ? JSON.parse(fs.readFileSync(target, 'utf8')) : null
  const gateCount = saved ? saved.stages.reduce((n, s) => n + (s.gates || []).length, 0) : -1
  const st2 = saved ? saved.stages.filter((s) => s.id === 'stage-2')[0] : null
  check('[UI] 落盘内容 = 表单输入（2 阶段 / 3 门禁 / command 带 cmd / artifact 带产物路径）',
    !!saved && saved.stages.length === 2 && gateCount === 3 && !!st2 && st2.title === '实施' &&
    (st2.gates || []).length === 2 &&
    st2.gates[0].id === 'g-2-1' && st2.gates[0].kind === 'command' && st2.gates[0].cmd === 'go test ./...' &&
    st2.gates[1].id === 'g-2-2' && st2.gates[1].kind === 'artifact' && st2.gates[1].artifact === 'dist/app.exe',
    saved ? JSON.stringify({ stages: saved.stages.map((s) => s.id), gates: gateCount, stage2: st2 }).slice(0, 360) : '（未落盘）')
  check('[UI] 保存后壳内提示（$toast 收到成功通知）', toasts.some((t) => /已创建流程|已保存流程/.test(t.text)), JSON.stringify(toasts.slice(-2)))

  // ── ⑤ 表单 ↔ JSON 双向切换不丢字段 ──────────────────────────
  const myRow = rowOf(host, 'my-flow')
  check('[UI] 流程库出现新建流程行（可编辑层）', !!myRow, myRow ? textAll(myRow).slice(0, 120) : '（无）')
  btnOf(myRow, '编辑').dispatch('click')
  await flush()
  check('[UI] 编辑既有流程默认仍是表单模式', byClass(host, 'ftb-seg').length >= 1 && byClass(host, 'ftb-textarea').length === 0)
  check('[UI] 表单回显既有定义（2 阶段 3 门禁）', byClass(host, 'ftb-block').length >= 3 && byClass(host, 'ftb-gate-form').length === 3,
    'stages=' + stageBlocks(host).length + ' gates=' + byClass(host, 'ftb-gate-form').length)

  btnOf(byClass(host, 'ftb-seg')[0], 'JSON').dispatch('click')
  await flush()
  let ta = byClass(host, 'ftb-textarea')[0]
  check('[UI] 切到 JSON 模式：文本 = 表单当前定义（含阶段与门禁 id）',
    !!ta && /"stages"/.test(ta.value) && /stage-2/.test(ta.value) && /g-2-1/.test(ta.value),
    ta ? ta.value.slice(0, 160) : '（无 textarea）')
  // 在 JSON 态改一个字段 + 追加表单未渲染字段（board）
  const parsed = JSON.parse(ta.value)
  parsed.title = 'JSON 改过的标题'
  parsed.board = { kind: 'kanban' }
  ta.value = JSON.stringify(parsed, null, 2)
  ta.dispatch('input')
  btnOf(byClass(host, 'ftb-seg')[0], '表单').dispatch('click')
  await flush()
  const titleInput = fieldInput(byClass(host, 'ftb-block')[0], '标题')
  check('[UI] JSON → 表单同步：标题跟随 JSON 改动、阶段/门禁数不变',
    blockTitleOK(host) && byClass(host, 'ftb-gate-form').length === 3,
    'titleField=' + (titleInput ? titleInput.value : '（无）') + ' gates=' + byClass(host, 'ftb-gate-form').length)
  btnOf(byClass(host, 'ftb-seg')[0], 'JSON').dispatch('click')
  await flush()
  ta = byClass(host, 'ftb-textarea')[0]
  check('[UI] 双向切换不丢字段：表单未渲染的 board 仍在 JSON 中',
    !!ta && /"board"/.test(ta.value) && /"kanban"/.test(ta.value) && /"title": "JSON 改过的标题"/.test(ta.value),
    ta ? ta.value.slice(0, 200) : '（无 textarea）')
  btnOf(host, '取消').dispatch('click')
  await flush()

  // ── ⑥ preset 只读层入口 ─────────────────────────────────────
  const presetRow = byClass(host, 'ftb-row').filter((r) => /preset\(只读\)/.test(textAll(r)))[0] || null
  check('[UI] preset 层行只读：无「编辑」入口，只有「克隆到项目」',
    !!presetRow && !btnOf(presetRow, '编辑') && !!btnOf(presetRow, '克隆到项目'),
    presetRow ? textAll(presetRow).slice(0, 160) : '（无 preset 行）')
  btnOf(presetRow, '克隆到项目').dispatch('click')
  await flush(8)
  const clonedRow = byClass(host, 'ftb-row').filter((r) => /（自定义）/.test(textAll(r)))[0] || null
  check('[UI] 克隆到项目后该行才出现「编辑」入口（只读层不可直接编辑）',
    !!clonedRow && !!btnOf(clonedRow, '编辑'),
    clonedRow ? textAll(clonedRow).slice(0, 160) : '（无克隆行）')

  unmount && unmount()
  return
}

// 表单里「阶段 1 标题」= JSON 改过的标题（位于第一个阶段块的标题字段）
function blockTitleOK(host) {
  const blocks = byClass(host, 'ftb-block')
  if (!blocks.length) return false
  const t = (function find(scope) {
    for (const f of byClass(scope, 'ftb-field')) {
      const lab = all(f).filter((n) => String(n.className).indexOf('ftb-label') >= 0)[0]
      if (lab && lab._text === '标题' && f.parent === scope) {
        const i = all(f).filter((n) => n.tagName === 'INPUT')[0]
        if (i) return i.value
      }
    }
    return ''
  })(blocks[0])
  return t === 'JSON 改过的标题'
}
