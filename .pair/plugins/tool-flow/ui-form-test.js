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
    style: {}, _text: '', _listeners: {}, _attrs: {},
  }
  Object.defineProperty(e, 'textContent', {
    get() { return e._text },
    set(v) { e._text = String(v); e.children.length = 0 },   // 同浏览器：赋值即清空子节点
  })
  e.appendChild = (c) => { c.parent = e; e.children.push(c); return c }
  e.addEventListener = (ev, fn) => { (e._listeners[ev] = e._listeners[ev] || []).push(fn) }
  e.dispatch = (ev) => { for (const fn of (e._listeners[ev] || []).slice()) fn({ target: e }) }
  e.setAttribute = (k, v) => { e._attrs[String(k)] = String(v) }
  e.getAttribute = (k) => (String(k) in e._attrs ? e._attrs[String(k)] : null)
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
  let queryRoot = null
  const doc = {
    createElement: (t) => makeEl(t),
    // client.js 用 querySelectorAll('[data-tb-desc="id"]') 同步卡片摘要（编辑保存后不整页重渲染）
    querySelectorAll: (sel) => {
      if (!queryRoot) return []
      const m = /^\[data-tb-([a-z]+)="([^"]*)"\]$/.exec(String(sel || ''))
      if (!m) return []
      const key = 'data-tb-' + m[1]
      return nodes(queryRoot, (n) => !!n._attrs && n._attrs[key] === m[2])
    },
  }
  const toasts = []
  const win = {
    __state: { currentConvId: 'conv_form' },
    $toast: (text, type) => toasts.push({ text: text, type: type }),
    confirm: () => true,
  }
  let view = null
  const ui = {
    registerView: (v) => { view = v },
    http: {
      // 面板主体已是任务看板：/taskboard/state 走任务快照；流程快照仍供「流程库」页签用
      get: async (p, params) => {
        if (String(p || '').indexOf('taskboard') >= 0) return env.boardState ? env.boardState() : null
        return env.snapshot(params && params.convId)
      },
    },
    invoke: async (plugin, method, args) => {
      if (String(method) === 'taskboard_action' && env.boardRpc) return env.boardRpc(args || {})
      return env.rpc(args || {})
    },
  }
  const flush = async (n) => { for (let i = 0; i < (n || 4); i++) await new Promise((r) => setTimeout(r, 3)) }
  const noop = () => { }

  const code = fs.readFileSync(env.clientPath, 'utf8')
  const load = new Function('ui', 'window', 'document', 'setInterval', 'clearInterval', 'console',
    'return (' + code + ')(ui)')
  load(ui, win, doc, setInterval, clearInterval, { log: noop, warn: noop, error: noop })
  check('[UI] client.js 装载并注册看板视图（id=flow-board）', !!view && view.id === 'flow-board', view && view.id)

  const host = makeEl('div')
  queryRoot = host
  const unmount = view.render(host)
  await flush()

  // ── ① 进入流程库页签（左列表 + 右画布）──────────────────────
  const libTab = btnOf(host, '流程库')
  check('[UI] 顶部页签渲染（任务看板 / 流程库；旧流程视图已下线）', !!libTab && !!btnOf(host, '任务看板') && !btnOf(host, '当前流程'))
  libTab.dispatch('click')
  await flush(6)
  const items = () => byClass(host, 'ftb-flowitem')
  const nodeCards = () => byClass(host, 'ftb-node')
  const insp = () => byClass(host, 'ftb-insp')[0] || null
  check('[UI] 流程库 = 左列表 + 右画布（.ftb-lib-side / .ftb-canvas 并存）',
    byClass(host, 'ftb-lib').length === 1 && byClass(host, 'ftb-lib-side').length === 1 && byClass(host, 'ftb-canvas').length === 1,
    'lib=' + byClass(host, 'ftb-lib').length + ' side=' + byClass(host, 'ftb-lib-side').length + ' canvas=' + byClass(host, 'ftb-canvas').length)
  check('[UI] 左列表列出三层流程（每项带层标签与节点数）',
    items().length >= 5 && byClass(host, 'ftb-scope').length >= 5 && /节点/.test(textAll(items()[0])),
    'items=' + items().length + ' first=' + (items()[0] ? textAll(items()[0]).slice(0, 80) : '（无）'))
  const newBtn = btnOf(host, '＋ 新建流程')
  check('[UI] 流程库提供「＋ 新建流程」入口（画布落地路径）', !!newBtn)

  // ── ② 默认画布模式 + 空白骨架（1 节点 / 检查器 1 门禁）──────
  newBtn.dispatch('click')
  await flush()
  const seg = byClass(host, 'ftb-seg')[0]
  const canvasTab = seg ? btnOf(seg, '画布') : null
  check('[UI] 默认进入画布模式（有「画布 / JSON」切换，画布态无 textarea）',
    !!seg && !!canvasTab && /active/.test(canvasTab.className) && byClass(host, 'ftb-textarea').length === 0 && !!btnOf(seg, 'JSON'),
    seg ? textAll(seg) : '（无切换控件）')
  check('[UI] 空白骨架 = 1 个节点 + 右栏检查器 1 个门禁',
    nodeCards().length === 1 && byClass(host, 'ftb-gate-form').length === 1 && !!insp(),
    'nodes=' + nodeCards().length + ' gates=' + byClass(host, 'ftb-gate-form').length + ' insp=' + !!insp())
  check('[UI] 画布节点绘制：序号 + 标题 + 阶段 id（可视化节点）',
    nodeCards().length === 1 && !!byClass(nodeCards()[0], 'ftb-node-idx')[0] && !!byClass(nodeCards()[0], 'ftb-node-title')[0] &&
    !!byClass(nodeCards()[0], 'ftb-node-sid')[0],
    nodeCards()[0] ? textAll(nodeCards()[0]) : '（无节点）')

  // ── ③ 节点增删改与拖拽重排 ───────────────────────────────────
  btnOf(host, '＋ 添加节点').dispatch('click')
  await flush()
  check('[UI] 「＋ 添加节点」→ 2 个节点 + 1 段连线',
    nodeCards().length === 2 && byClass(host, 'ftb-cv-link').length === 1,
    'nodes=' + nodeCards().length + ' links=' + byClass(host, 'ftb-cv-link').length)

  // 点击第一个节点 → 选中（.sel）且检查器切到 #1
  nodeCards()[0].dispatch('click')
  await flush()
  check('[UI] 点击节点选中：.sel 高亮 + 检查器切换到该节点',
    /\bsel\b/.test(nodeCards()[0].className) && /#1/.test(textAll(insp() || host)),
    'cls=' + (nodeCards()[0] ? nodeCards()[0].className : '') + ' insp=' + (insp() ? textAll(insp()).slice(0, 60) : '（无）'))

  // 节点「↑ / ↓」重排（新阶段从末位移到首位）
  check('[UI] 每个节点提供 ↑ / ↓ / × 快捷按钮',
    !!btnOf(nodeCards()[1], '↑') && !!btnOf(nodeCards()[1], '↓') && !!btnOf(nodeCards()[1], '×'))
  btnOf(nodeCards()[1], '↑').dispatch('click')
  await flush()
  check('[UI] 节点「↑」上移生效（新阶段移到首位，节点数不变）',
    nodeCards().length === 2 && textAll(nodeCards()[0]).indexOf('新阶段') >= 0,
    nodeCards()[0] ? textAll(nodeCards()[0]).slice(0, 80) : '（无节点）')
  btnOf(nodeCards()[0], '↓').dispatch('click')
  await flush()
  check('[UI] 节点「↓」下移恢复原序', nodeCards().length === 2 && textAll(nodeCards()[1]).indexOf('新阶段') >= 0,
    nodeCards()[1] ? textAll(nodeCards()[1]).slice(0, 80) : '（无节点）')

  // 拖拽重排：dragstart(第 2 个) → dragover/drop(第 1 个)
  nodeCards()[1].dispatch('dragstart')
  nodeCards()[0].dispatch('dragover')
  check('[UI] 拖拽悬停目标节点 → drop-on 落点高亮', /\bdrop-on\b/.test(nodeCards()[0].className), nodeCards()[0].className)
  nodeCards()[0].dispatch('drop')
  await flush()
  check('[UI] 拖拽 drop 后顺序交换（新阶段到首位）', textAll(nodeCards()[0]).indexOf('新阶段') >= 0,
    nodeCards()[0] ? textAll(nodeCards()[0]).slice(0, 80) : '（无节点）')

  // 删除节点（此时首位是刚新增的 stage-2；删它，再添加验证 id 不重复）
  btnOf(nodeCards()[0], '×').dispatch('click')
  await flush()
  check('[UI] 「×」删除节点 → 节点数减少', nodeCards().length === 1, 'nodes=' + nodeCards().length)
  btnOf(host, '＋ 添加节点').dispatch('click')
  await flush()
  check('[UI] 删除后可再添加节点（回到 2 个，且 id 不重复）',
    nodeCards().length === 2 && !/id 重复/.test(textAll(host)), 'nodes=' + nodeCards().length)

  // 选中第二个节点（stage-2）继续编辑
  nodeCards()[1].dispatch('click')
  await flush()
  const s2 = () => insp()
  const s2gates = () => byClass(s2(), 'ftb-gate-form')
  btnOf(s2(), '＋ 加门禁').dispatch('click')
  await flush()
  btnOf(s2(), '＋ 加门禁').dispatch('click')
  await flush()
  check('[UI] 检查器「＋ 加门禁」×2 → 选中节点门禁 2 个（全流程共 3 个）',
    byClass(host, 'ftb-gate-form').length === 2 && /3 门禁|2 门禁/.test(textAll(nodeCards()[1])),
    'inspGates=' + byClass(host, 'ftb-gate-form').length + ' node=' + textAll(nodeCards()[1]).slice(0, 80))

  // 删除一个门禁（验证「可删」）后再加回，最终仍为 3
  btnOf(s2gates()[1], '删除门禁').dispatch('click')
  await flush()
  check('[UI] 「删除门禁」生效 → 选中节点门禁 1 个', byClass(host, 'ftb-gate-form').length === 1, 'gates=' + byClass(host, 'ftb-gate-form').length)
  btnOf(s2(), '＋ 加门禁').dispatch('click')
  await flush()
  check('[UI] 删后再加 → 选中节点门禁回到 2 个', byClass(host, 'ftb-gate-form').length === 2)

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
  const errText = textAll(byClass(host, 'ftb-lib-main')[0] || host)
  check('[UI] kind 缺字段被拦：command 未填 cmd → 提示且未落盘',
    /必须填「命令 cmd」/.test(errText) && !fs.existsSync(target),
    errText.slice(0, 220))
  check('[UI] 校验失败时给出可读错误（.ftb-err 错误样式）', byClass(host, 'ftb-err').length >= 1)

  // 补齐必填 → 保存成功（纯画布新建）
  setVal(fieldInput(s2gates()[0], '命令 cmd'), 'go test ./...')
  setVal(fieldInput(s2gates()[1], '门禁 id'), 'g-2-2')
  setVal(fieldInput(s2gates()[1], '产物路径'), 'dist/app.exe')
  btnOf(host, '保存').dispatch('click')
  await flush(8)
  if (!fs.existsSync(target)) {
    const eb = byClass(host, 'ftb-lib-main')[0]
    console.log('  [debug] 未落盘 → 画布面板文本：' + (eb ? textAll(eb) : '（无画布面板）').slice(0, 500))
  }
  check('[UI] 纯画布新建保存成功（落盘项目层 .pair/flows/my-flow.json）', fs.existsSync(target), target)
  const saved = fs.existsSync(target) ? JSON.parse(fs.readFileSync(target, 'utf8')) : null
  const gateCount = saved ? saved.stages.reduce((n, s) => n + (s.gates || []).length, 0) : -1
  const st2 = saved ? saved.stages.filter((s) => s.id === 'stage-2')[0] : null
  check('[UI] 落盘内容 = 画布输入（2 阶段 / 3 门禁 / command 带 cmd / artifact 带产物路径）',
    !!saved && saved.stages.length === 2 && gateCount === 3 && !!st2 && st2.title === '实施' &&
    (st2.gates || []).length === 2 &&
    st2.gates[0].id === 'g-2-1' && st2.gates[0].kind === 'command' && st2.gates[0].cmd === 'go test ./...' &&
    st2.gates[1].id === 'g-2-2' && st2.gates[1].kind === 'artifact' && st2.gates[1].artifact === 'dist/app.exe',
    saved ? JSON.stringify({ stages: saved.stages.map((s) => s.id), gates: gateCount, stage2: st2 }).slice(0, 360) : '（未落盘）')
  check('[UI] 保存后壳内提示（$toast 收到成功通知）', toasts.some((t) => /已创建流程|已保存流程/.test(t.text)), JSON.stringify(toasts.slice(-2)))

  // ── ⑤ 画布 ↔ JSON 双向切换不丢字段 ──────────────────────────
  const flowItemOf = (id) => items().filter((r) => textAll(r).indexOf(id) >= 0)[0] || null
  const myRow = flowItemOf('my-flow')
  check('[UI] 流程库出现新建流程项（可编辑层）', !!myRow, myRow ? textAll(myRow).slice(0, 120) : '（无）')
  // 点左列表项即加载该流程到右侧画布（无需「编辑」按钮）
  if (myRow) myRow.dispatch('click')
  await flush()
  check('[UI] 点列表项 → 右栏画布默认画布模式', byClass(host, 'ftb-seg').length >= 1 &&
    !!btnOf(byClass(host, 'ftb-seg')[0], '画布') && byClass(host, 'ftb-textarea').length === 0)
  // 点第二个节点（stage-2）验证节点选中切换与定义完整回显
  if (nodeCards()[1]) nodeCards()[1].dispatch('click')
  await flush()
  check('[UI] 画布回显既有定义（2 节点；选中节点 2 门禁）', nodeCards().length === 2 && byClass(host, 'ftb-gate-form').length === 2,
    'nodes=' + nodeCards().length + ' gates=' + byClass(host, 'ftb-gate-form').length +
    ' cards=' + nodeCards().map((n) => textAll(n).slice(0, 40)).join(' || ') +
    ' insp=' + (insp() ? textAll(insp()).slice(0, 160) : '（无）'))

  btnOf(byClass(host, 'ftb-seg')[0], 'JSON').dispatch('click')
  await flush()
  let ta = byClass(host, 'ftb-textarea')[0]
  check('[UI] 切到 JSON 模式：文本 = 画布当前定义（含阶段与门禁 id）',
    !!ta && /"stages"/.test(ta.value) && /stage-2/.test(ta.value) && /g-2-1/.test(ta.value),
    ta ? ta.value.slice(0, 160) : '（无 textarea）')
  // 在 JSON 态改一个字段 + 追加画布未渲染字段（board）
  const parsed = JSON.parse(ta.value)
  parsed.title = 'JSON 改过的标题'
  parsed.board = { kind: 'kanban' }
  ta.value = JSON.stringify(parsed, null, 2)
  ta.dispatch('input')
  btnOf(byClass(host, 'ftb-seg')[0], '画布').dispatch('click')
  await flush()
  const titleInput = fieldInput(byClass(host, 'ftb-block')[0], '标题')
  check('[UI] JSON → 画布同步：标题跟随 JSON 改动、节点/门禁数不变',
    blockTitleOK(host) && byClass(host, 'ftb-gate-form').length === 2,
    'titleField=' + (titleInput ? titleInput.value : '（无）') + ' gates=' + byClass(host, 'ftb-gate-form').length)
  btnOf(byClass(host, 'ftb-seg')[0], 'JSON').dispatch('click')
  await flush()
  ta = byClass(host, 'ftb-textarea')[0]
  check('[UI] 双向切换不丢字段：画布未渲染的 board 仍在 JSON 中',
    !!ta && /"board"/.test(ta.value) && /"kanban"/.test(ta.value) && /"title": "JSON 改过的标题"/.test(ta.value),
    ta ? ta.value.slice(0, 200) : '（无 textarea）')
  btnOf(host, '取消').dispatch('click')
  await flush()

  // ── ⑥ preset 只读层入口 ─────────────────────────────────────
  const presetRow = items().filter((r) => /preset·只读/.test(textAll(r)))[0] || null
  check('[UI] preset 层列表项只读：无「编辑」入口，只有「克隆到项目」',
    !!presetRow && !btnOf(presetRow, '编辑') && !!btnOf(presetRow, '克隆到项目'),
    presetRow ? textAll(presetRow).slice(0, 160) : '（无 preset 项）')
  btnOf(presetRow, '克隆到项目').dispatch('click')
  await flush(8)
  const clonedRow = items().filter((r) => /（自定义）/.test(textAll(r)))[0] || null
  check('[UI] 克隆到项目后出现可编辑副本项（只读层不可直接编辑）',
    !!clonedRow && !/·只读/.test(textAll(clonedRow)),
    clonedRow ? textAll(clonedRow).slice(0, 160) : '（无克隆项）')

  // ── ⑦ 任务看板（2026-10-17 重建：照 github.com/cloader/dsh-taskboard）──
  //   五列状态机 + 人建卡 + 认领/验收协议闸 + 流程绑定（门禁→DoD 清单）
  //   旧流程视图（当前流程 / 运行记录 / 项目 × 流程）已下线，流程库并入顶部页签。
  win.__state.currentConvId = 'conv_board1'
  const tabLabels = () => byClass(host, 'ftb-tab').map((b) => b._text)
  check('[UI-看板] 顶部页签 = 任务看板 / 流程库（旧流程视图已下线）',
    tabLabels().indexOf('任务看板') >= 0 && tabLabels().indexOf('流程库') >= 0 &&
    tabLabels().indexOf('当前流程') < 0 && tabLabels().indexOf('运行记录') < 0,
    'tabs=' + tabLabels().join('/'))
  const tabBoard = btnOf(host, '任务看板')
  if (tabBoard) tabBoard.dispatch('click')
  await flush(10)

  const cols = () => byClass(host, 'ftb-tcol')
  const colByTitle = (t) => cols().filter((c) => textAll(c).indexOf(t) >= 0)[0] || null
  check('[UI-看板] 五列主看板（待规划/待办/进行中/待验收/已完成），列头各带计数',
    cols().length === 5 && ['待规划', '待办', '进行中', '待验收', '已完成'].every((t) => !!colByTitle(t)) &&
    byClass(host, 'ftb-tcol-count').length === 5,
    'cols=' + cols().length)
  check('[UI-看板] 空列有指引性占位「无任务」（不是空白格）',
    byClass(host, 'ftb-tcol-empty').length >= 1 && /无任务/.test(textAll(host)),
    'empty=' + byClass(host, 'ftb-tcol-empty').length)

  // 驱动条：规划定稿自动入待办开关（修 2）——真实 RPC 落盘 + 状态回显往返
  const paBtn = btnOf(host, '规划定稿自动入待办')
  check('[UI-看板] 驱动条含「规划定稿自动入待办」开关（默认关，title 说明停在待确认）',
    !!paBtn && !/active/.test(paBtn.className) && /规划待确认/.test(String(paBtn.title || '')),
    paBtn ? ('cls=' + paBtn.className + ' title=' + paBtn.title) : '（无按钮）')
  if (paBtn) paBtn.dispatch('click')
  await flush(14)
  const drvOn = ((env.boardState ? env.boardState() : {}) || {}).driver || {}
  check('[UI-看板] 点击开关 → /api/taskboard/state 的 driver.planAutoAccept=true',
    drvOn.planAutoAccept === true, JSON.stringify(drvOn))
  const paBtnOn = btnOf(host, '规划定稿自动入待办')
  check('[UI-看板] 开启后按钮呈 active 态（状态回显 + title 改为免人工确认）',
    !!paBtnOn && /active/.test(paBtnOn.className) && /免人工确认/.test(String(paBtnOn.title || '')),
    paBtnOn ? ('cls=' + paBtnOn.className + ' title=' + paBtnOn.title) : '（无按钮）')
  if (paBtnOn) paBtnOn.dispatch('click')
  await flush(14)
  const drvOff = ((env.boardState ? env.boardState() : {}) || {}).driver || {}
  check('[UI-看板] 再点一次 → driver.planAutoAccept 回到 false',
    drvOff.planAutoAccept === false, JSON.stringify(drvOff))

  // 人建卡
  const tbNewBtn = btnOf(host, '＋ 新建任务')
  check('[UI-看板] 有「＋ 新建任务」入口（人建卡）', !!tbNewBtn)
  if (tbNewBtn) tbNewBtn.dispatch('click')
  await flush(8)
  // 新版建卡表单：标题是裸 input（placeholder 标明语义），描述/执行提示词是 textarea
  const titleIn = nodes(host, (n) => n.tagName === 'INPUT' && /一句话任务名/.test(String(n.placeholder || '')))[0] || null
  check('[UI-看板] 建卡表单有「标题」输入框', !!titleIn)
  if (titleIn) setVal(titleIn, '看板断言：写一个测试任务')
  const draftTas = nodes(host, (n) => n.tagName === 'TEXTAREA')
  check('[UI-看板] 建卡表单：描述与执行提示词是多行编辑框（textarea, rows>=3）',
    draftTas.length >= 2 && draftTas.every((t) => Number(t.rows || 0) >= 3),
    'textareas=' + draftTas.length + ' rows=' + draftTas.map((t) => String(t.rows)).join(','))
  const createBtn = btnOf(host, '创建任务')
  if (createBtn) createBtn.dispatch('click')
  await flush(16)
  const board1 = env.boardState ? env.boardState() : { tasks: [] }
  const made = (board1.tasks || []).filter((t) => t.title === '看板断言：写一个测试任务')[0] || null
  check('[UI-看板] 建卡成功（真 RPC 落 .pair/taskboard/tasks.json，状态 todo）',
    !!made && made.status === 'todo', made ? made.id + ' ' + made.status : '（无）')
  check('[UI-看板] 卡片出现在「待办」列',
    !!made && !!colByTitle('待办') && textAll(colByTitle('待办')).indexOf('看板断言') >= 0,
    'todoCol=' + (colByTitle('待办') ? textAll(colByTitle('待办')).slice(0, 140) : '（无列）') +
    ' | cards=' + byClass(host, 'ftb-tcard').length + ' | made=' + (made ? made.id : '（无）') +
    ' | tcols=' + byClass(host, 'ftb-tcol').length + ' | bar=' + byClass(host, 'ftb-tb-bar').length +
    ' | err=' + (/看板渲染失败[^|]{0,160}/.exec(textAll(host)) || [''])[0] +
    ' | bodyText=' + textAll(host).slice(0, 180))

  const mirrorFile = made ? path.join(env.ws, '.pair', 'tasks', 'flowtb_' + String(made.id).replace(/[^A-Za-z0-9]/g, '_') + '.json') : ''
  check('[UI-看板] 任务镜像到宿主 .pair/tasks（右栏任务进度面板可见）',
    !!made && fs.existsSync(mirrorFile), mirrorFile ? path.basename(mirrorFile) : '（无）')

  // 拖拽：待办 → 进行中（合法迁移）
  const cardOf = (title) => byClass(host, 'ftb-tcard').filter((c) => textAll(c).indexOf(title) >= 0)[0] || null
  let card = cardOf('看板断言')
  check('[UI-看板] 卡片可拖拽（draggable=true）', !!card && card.draggable === true)
  if (card) card.dispatch('dragstart')
  const colDoing = colByTitle('进行中')
  if (colDoing) colDoing.dispatch('dragover')
  check('[UI-看板] 拖拽悬停目标列 → 落点虚线（drop-on）',
    !!colDoing && /drop-on/.test(colDoing.className), colDoing ? colDoing.className : '（无列）')
  if (colDoing) colDoing.dispatch('drop')
  await flush(16)
  const board2 = env.boardState ? env.boardState() : { tasks: [] }
  const moved = (board2.tasks || []).filter((t) => t.id === (made && made.id))[0] || null
  check('[UI-看板] 拖到「进行中」→ 状态真的变了（dragstart + drop 全链路）',
    !!moved && moved.status === 'in_progress',
    moved ? moved.status + ' claimed=' + moved.claimedBy : '（无）')

  // 非法迁移被状态机拦住（照参考 drag.forbidden）
  const refreshBtn = btnOf(host, '刷新')
  if (refreshBtn) refreshBtn.dispatch('click')
  await flush(12)
  card = cardOf('看板断言')
  if (card) card.dispatch('dragstart')
  const colDone = colByTitle('已完成')
  if (colDone) colDone.dispatch('drop')
  await flush(12)
  const board3 = env.boardState ? env.boardState() : { tasks: [] }
  const still = (board3.tasks || []).filter((t) => t.id === (made && made.id))[0] || null
  check('[UI-看板] 非法迁移被拦：进行中 → 已完成 不允许，给可读提示且状态不变',
    !!still && still.status === 'in_progress' &&
    /无法从「进行中」移到「已完成」/.test(textAll(host) + '|' + textAll(byClass(host, 'ftb-msg')[0] || host)),
    still ? still.status : '（无）')

  // 详情（清单 / 评论 / 状态机按钮）
  if (refreshBtn) refreshBtn.dispatch('click')
  await flush(12)
  card = cardOf('看板断言')
  if (card) card.dispatch('click')
  await flush(10)
  check('[UI-看板] 点卡片 → 展开详情（状态机迁移按钮 + 评论入口）',
    !!byClass(host, 'ftb-detail')[0] && !!btnOf(host, '移到「待验收」') && !!btnOf(host, '评论'),
    'detail=' + byClass(host, 'ftb-detail').length)
  const detailTas = nodes(host, (n) => n.tagName === 'TEXTAREA')
  check('[UI-看板] 详情含执行/认领/定时入口 + 描述与执行提示词多行编辑',
    !!btnOf(host, '执行') && (!!btnOf(host, '释放认领') || !!btnOf(host, '认领')) &&
    detailTas.length >= 2 && nodes(host, (n) => n.tagName === 'SELECT').length >= 1,
    'exec=' + !!btnOf(host, '执行') + ' claim=' + (!!btnOf(host, '释放认领') || !!btnOf(host, '认领')) +
    ' textareas=' + detailTas.length + ' selects=' + nodes(host, (n) => n.tagName === 'SELECT').length)

  // ★ 编辑回归（监督者实测缺陷）：逐字符 input 不得重建 textarea（本地草稿 + 不重渲染）
  const taDesc = detailTas[0]
  const taCountBefore = detailTas.length
  const probeText = '自动化探针-多行输入'
  let typed = ''
  for (const ch of probeText) { typed += ch; setVal(taDesc, typed) }
  await flush(2)
  const taAfter = nodes(host, (n) => n.tagName === 'TEXTAREA')
  check('[UI-看板] 逐字符输入不重建 textarea（同一节点仍在树中）且值完整',
    taAfter.indexOf(taDesc) >= 0 && taDesc.value === probeText && taAfter.length === taCountBefore,
    'inTree=' + (taAfter.indexOf(taDesc) >= 0) + ' value=' + JSON.stringify(taDesc.value) +
    ' count=' + taCountBefore + '->' + taAfter.length)
  // 失焦保存 → 只把 task 合并进快照，不整页 notify（节点不被替换）
  taDesc.dispatch('blur')
  await flush(8)
  const taAfter2 = nodes(host, (n) => n.tagName === 'TEXTAREA')
  check('[UI-看板] 保存后不整页重建（同一 textarea 节点保持、值不丢）',
    taAfter2.indexOf(taDesc) >= 0 && taDesc.value === probeText,
    'inTree=' + (taAfter2.indexOf(taDesc) >= 0) + ' value=' + JSON.stringify(taDesc.value))
  // 认领者可读身份：展示「你」而不是完整 convId（完整 id 在 title 里）
  const detailText = byClass(host, 'ftb-detail')[0] ? textAll(byClass(host, 'ftb-detail')[0]) : ''
  check('[UI-看板] 详情认领展示可读身份「你」（完整 convId 在 title）',
    detailText.indexOf('认领 你') >= 0 && detailText.indexOf('conv_form') < 0,
    detailText.slice(0, 160))

  // 协议闸：agent 不能移到 done（host 工具层强制）；人能移到待验收
  const agentTry = env.tool ? await env.tool('taskboard_move', { id: made && made.id, to: 'done' }) : null
  check('[UI-看板] 协议闸：agent 不能把任务移到「已完成」（验收只能由人做）',
    !!agentTry && /只能由人验收/.test(String(agentTry.text || agentTry.error || '')),
    JSON.stringify(agentTry || {}).slice(0, 140))
  const humanReview = env.boardRpc ? env.boardRpc({ op: 'move', id: made && made.id, to: 'in_review' }) : null
  check('[UI-看板] 人可把任务移到「待验收」', !!humanReview && humanReview.ok === true,
    JSON.stringify(humanReview || {}).slice(0, 120))

  // 流程绑定：建卡带流程 → 流程门禁成为验收清单（DoD）
  const fx = env.boardRpc ? env.boardRpc({ op: 'create', title: '看板断言：绑定流程的卡', flowId: 'generic' }) : null
  check('[UI-看板] 建卡可绑定流程，流程门禁成为验收清单（DoD）',
    !!fx && fx.ok === true && (fx.task.checklist || []).length > 0,
    fx && fx.task ? '清单 ' + (fx.task.checklist || []).length + ' 项' : JSON.stringify(fx || {}).slice(0, 120))

  // 清理：不留测试卡给真实工作区
  if (env.boardRpc && made && made.id) env.boardRpc({ op: 'trash', id: made.id })
  if (env.boardRpc && fx && fx.task && fx.task.id) env.boardRpc({ op: 'trash', id: fx.task.id })
  await flush(6)
  win.__state.currentConvId = 'conv_form'
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
