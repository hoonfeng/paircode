// ═══════════════════════════════════════════════════════════════
// ask-user-events-check.mjs — ask_user 实时事件归一化自检（2026-10-15）
//
// 背景（真实缺陷）：模型按插件 schema 生成的是 snake_case 参数
//   （questions:[{id, question, options, multi_select, with_input}]），而卡片与历史接口
//   用 camelCase（multiSelect / withInput）。实时路径此前把 args.questions **原样**
//   塞进 segment → AskUserCard 读 q.multiSelect 恒为 undefined → 多选问题被渲染成
//   单选圆点（用户报告「前端 ask_user 不支持多选」）；漏填 id 时还会导致答案提交
//   被后端 validateAskAnswers 拒绝。
//
// 本脚本在 node 下加载真实前端模块（agent-events.js），驱动 processAgentEvent 走
// 「WS tool_call 事件 → ask_user 段」路径并断言归一化结果。
//   node scripts/ask-user-events-check.mjs   # 全 PASS 退出码 0
// ═══════════════════════════════════════════════════════════════
globalThis.window = globalThis.window || {
  dispatchEvent() {}, addEventListener() {}, removeEventListener() {},
  localStorage: { getItem: () => null, setItem() {}, removeItem() {} },
  location: { href: 'http://localhost/', origin: 'http://localhost' },
}
globalThis.localStorage = globalThis.localStorage || globalThis.window.localStorage
globalThis.document = globalThis.document || {
  addEventListener() {}, createElement: () => ({ style: {}, setAttribute() {}, appendChild() {} }),
  documentElement: { style: {} }, body: { appendChild() {} },
}

const ae = await import('../plugins-src/ui-app/src/agent-events.js')
const { state } = await import('../plugins-src/ui-app/src/ui-state.js')

const convId = 'conv-verify'
state.messagesByConv[convId] = []
const key = ae.createAssistantPlaceholder(convId, 'k1')
ae.startConvRuntime(convId, key, 'hi')
// ★ 刷新门控：历史未加载的会话事件会进 pending（不处理），先标记历史已加载
ae.markHistoryLoaded(convId)

// ── 场景 1：多问题（插件 schema 原始 snake_case）──
const multiArgs = JSON.stringify({
  questions: [
    { id: 'auth', question: '认证方式？', options: ['JWT', 'Session'], multi_select: true },
    { question: '部署目标？', options: ['Web', '桌面', 'CLI'], multi_select: false, with_input: true },
    { id: 'note', question: '补充说明？' },
  ],
})
ae.processAgentEvent(convId, { type: 'tool_call', tool: 'ask_user', args: multiArgs, callId: 'c1' })

const segs = state.messagesByConv[convId][0].segments
const multiSeg = segs.find(s => s.type === 'ask_user')
const qs = multiSeg.questions
const checks = [
  ['多选标记 multi_select → multiSelect', qs[0].multiSelect === true],
  ['单选保持 false', qs[1].multiSelect === false],
  ['with_input → withInput', qs[1].withInput === true],
  ['缺 id 兜底 q{序号}', qs[1].id === 'q2'],
  ['显式 id 保留', qs[0].id === 'auth' && qs[2].id === 'note'],
  ['options 保留', JSON.stringify(qs[0].options) === JSON.stringify(['JWT', 'Session'])],
  ['多问题段清空单问题 question', multiSeg.question === ''],
]

// ── 场景 2：单问题多选（askType=multi）──
ae.processAgentEvent(convId, {
  type: 'tool_call', tool: 'ask_user', callId: 'c2',
  args: JSON.stringify({ question: '要跑哪些测试？', askType: 'multi', options: ['单测', '集成', ''] }),
})
const singleSeg = state.messagesByConv[convId][0].segments.filter(s => s.type === 'ask_user').pop()
checks.push(
  ['单问题 askType=multi 保留', singleSeg.askType === 'multi'],
  ['空选项被过滤', JSON.stringify(singleSeg.options) === JSON.stringify(['单测', '集成'])],
)

// ── 场景 3：单问题 single-with-input ──
ae.processAgentEvent(convId, {
  type: 'tool_call', tool: 'ask_user', callId: 'c3',
  args: JSON.stringify({ question: '选方案', askType: 'single_with_input', options: ['A', 'B'] }),
})
const customSeg = state.messagesByConv[convId][0].segments.filter(s => s.type === 'ask_user').pop()
checks.push(['snake askType 归一化 single-with-input', customSeg.askType === 'single-with-input'])

let failed = 0
for (const [name, ok] of checks) {
  if (!ok) failed++
  console.log((ok ? 'PASS' : 'FAIL') + '  ' + name)
}
console.log('\n多问题段渲染数据:', JSON.stringify(qs))
process.exit(failed === 0 ? 0 : 1)
