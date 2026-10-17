// 9096 实机：流程（tool-flow）× 任务看板 多场景端到端测试
// 通道：POST /api/plugins/invoke { plugin:"tool-flow", method:"flow_action" | "taskboard_action" }
// 用法：node scripts/e2e-flow-taskboard-9096.js [--keep]
//   --keep 保留测试数据（默认清理：purge run / trash 卡 / 删测试流程文件）
const http = require('http')
const fs = require('fs')
const path = require('path')

const PORT = Number(process.env.WEB_PORT || 9096)
const CONV = 'e2e-flowtb-9096'
const CONV_SF = CONV + '-startflow'
const FLOW_ID = 'e2e-scenario-flow'
const ROOT = path.resolve(__dirname, '..')
const KEEP = process.argv.includes('--keep')

function request(p, method, body) {
  return new Promise((resolve, reject) => {
    const data = body === undefined ? '' : JSON.stringify(body)
    const r = http.request({
      host: '127.0.0.1', port: PORT, path: p, method: method || 'GET',
      headers: { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(data) },
    }, x => {
      let d = ''
      x.on('data', c => d += c)
      x.on('end', () => { try { resolve(JSON.parse(d)) } catch (e) { resolve({ _raw: d, _status: x.statusCode }) } })
    })
    r.on('error', reject)
    if (data) r.write(data)
    r.end()
  })
}
const get = p => request(p, 'GET')

async function invoke(method, args) {
  const r = await request('/api/plugins/invoke', 'POST', { plugin: 'tool-flow', method, args })
  if (!r || r.ok !== true) return { ok: false, error: (r && (r.error || r._raw)) || 'invoke 失败' }
  return r.value || { ok: false, error: '空返回' }
}
const flow = a => invoke('flow_action', a)
const tb = a => invoke('taskboard_action', a)

let pass = 0, fail = 0
const failed = []
function check(name, ok, detail) {
  if (ok) pass++
  else { fail++; failed.push(name) }
  console.log(`${ok ? 'PASS' : 'FAIL'} ${name}${detail !== undefined ? ' · ' + JSON.stringify(detail).slice(0, 300) : ''}`)
}
const sum = (arr, k) => (arr || []).reduce((s, x) => s + (Number(x[k]) || 0), 0)

const created = { tasks: [], runs: [], flowFiles: [] }

;(async () => {
  console.log(`# 9096 流程 × 任务看板 多场景测试（应到 ${PORT}）\n`)
  const baseTb = await get('/api/taskboard/state')
  const baseFlow = await get('/api/flow/state')
  if (!baseTb || !baseTb.ok) throw new Error('9096 任务看板不可用: ' + JSON.stringify(baseTb).slice(0, 200))

  // ───────────────────────── 场景 0：现状盘点（只读） ─────────────────────────
  console.log('\n## 场景 0 · 现状盘点')
  check('任务看板快照可读', baseTb.ok === true, { revision: baseTb.revision, tasks: (baseTb.tasks || []).length })
  check('流程库可读且含预设流程', baseFlow.ok === true && (baseFlow.library || []).length >= 8, { flows: (baseFlow.library || []).length })
  const preCards = (baseTb.tasks || []).filter(t => !t.trashedAt)
  check('存在已绑流程的演示卡', preCards.some(t => t.flowId && t.status === 'in_progress'), preCards.filter(t => t.flowId).map(t => `${t.id}:${t.status}`))
  check('存在未绑流程的待验收对照卡', preCards.some(t => !t.flowId && t.status === 'in_review'), preCards.filter(t => !t.flowId && t.status === 'in_review').map(t => t.id))

  // ───────────────────────── 场景 1：流程定义与校验 ─────────────────────────
  console.log('\n## 场景 1 · 流程定义与校验（define/show/validate）')
  const def = {
    schemaVersion: 1, id: FLOW_ID, title: 'E2E 场景流程（测试）', version: 1,
    projectTypes: ['any'], taskTypes: ['test'],
    appliesTo: { types: ['any'], signals: [] },
    stages: [
      {
        id: 'prep', title: '准备', goal: '场景准备', plan: ['建卡', '绑流程'],
        gates: [
          { id: 'g-check-1', kind: 'check', text: '准备就绪（check 需证据）', required: true, blocking: true },
          { id: 'g-cmd-1', kind: 'command', text: '命令门禁 echo', cmd: 'echo e2e-gate-ok', required: true, blocking: true },
        ],
      },
      {
        id: 'impl', title: '实施', goal: '实施修改', plan: ['改代码'],
        gates: [
          { id: 'g-art-1', kind: 'artifact', text: '产物 .pair/project.md 存在', artifact: '.pair/project.md', required: true, blocking: true },
          { id: 'g-appr-1', kind: 'approval', text: '用户批准实施', required: true, blocking: true },
        ],
      },
      {
        id: 'wrap', title: '收尾', goal: '收尾验收', plan: ['出报告'],
        gates: [
          { id: 'g-art-bad', kind: 'artifact', text: '（非必过）不存在的产物', artifact: '.pair/__not_exist_e2e__.md', required: false, blocking: false },
          { id: 'g-check-2', kind: 'check', text: '收尾完成', required: true, blocking: true },
        ],
      },
    ],
  }
  let r = await flow({ op: 'define', scope: 'project', json: def })
  const flowFile = path.join(ROOT, '.pair', 'flows', FLOW_ID + '.json')
  if (r.ok && fs.existsSync(flowFile)) created.flowFiles.push(flowFile)
  check('合法流程保存成功（project 层）', r.ok === true, r.error || r.result)

  let sh = await flow({ op: 'show', id: FLOW_ID })
  check('回读：3 阶段 / 6 门禁', sh.ok === true && sh.result.def.stages.length === 3 && sum(sh.result.def.stages.map(s => s.gates), 'id') >= 0 && sh.result.def.stages.reduce((n, s) => n + (s.gates || []).length, 0) === 6,
    { stages: sh.ok && sh.result.def.stages.map(s => s.id + ':' + (s.gates || []).length) })

  const badDef = JSON.parse(JSON.stringify(def))
  badDef.id = 'e2e-bad-flow'
  badDef.stages = [{ id: 's1', title: 'S1', gates: [{ id: 'g-bad', kind: 'command', text: '缺 cmd', required: true }] }]
  r = await flow({ op: 'define', scope: 'project', json: badDef })
  const badFile = path.join(ROOT, '.pair', 'flows', 'e2e-bad-flow.json')
  if (fs.existsSync(badFile)) created.flowFiles.push(badFile)
  check('command 门禁缺 cmd 被拒绝', r.ok === false, r.error)
  check('非法定义未落盘', !fs.existsSync(badFile))

  r = await flow({ op: 'define', scope: 'preset', json: def })
  check('preset 只读层拒绝写入', r.ok === false, r.error)

  r = await flow({ op: 'validate', json: '{不是 JSON' })
  check('非法 JSON 校验报错', r.ok === false, r.error)

  // ───────────────────────── 场景 2：建卡 → 流程启动 → 绑定与 DoD ─────────────────────────
  console.log('\n## 场景 2 · 建卡 + 流程启动 + 绑定与 DoD')
  let c = await tb({ op: 'create', title: 'E2E 场景卡·流程绑定', description: '端到端场景测试用卡，勿手工处理', convId: CONV })
  check('看板建卡成功', c.ok === true, c.error)
  const taskId = c.ok ? c.task.id : ''
  if (taskId) created.tasks.push(taskId)

  let s = await flow({ op: 'start', flowId: FLOW_ID, taskId, convId: CONV })
  check('流程启动并绑定任务卡', s.ok === true, s.error)
  const runId = s.ok ? s.result.runId : ''
  if (runId) created.runs.push(runId)

  let snap = await get('/api/taskboard/state')
  let card = (snap.tasks || []).find(t => t.id === taskId)
  check('卡 runId 已回写', !!card && card.runId === runId, card && { runId: card.runId })
  check('卡 flowId 已补全', !!card && card.flowId === FLOW_ID, card && { flowId: card.flowId, scope: card.flowScope, v: card.flowVersion })
  check('DoD 清单 = 流程门禁数（6）', !!card && (card.checklist || []).length === 6, card && (card.checklist || []).map(x => x.gate))
  check('DoD 初始全部未勾选', !!card && (card.checklist || []).every(x => !x.checked))

  const dup = await flow({ op: 'start', flowId: FLOW_ID, taskId, convId: CONV })
  check('同会话重复启动被拒（判重）', dup.ok === false && /已有进行中的流程实例/.test(dup.error), dup.error)

  // ───────────────────────── 场景 3：门禁未过 → 推进被拦截 ─────────────────────────
  console.log('\n## 场景 3 · 门禁未过时推进被拦截')
  let adv = await flow({ op: 'advance', runId, convId: CONV })
  check('推进被拦截（advanced=false）', adv.ok === true && adv.result.advanced === false)
  check('列出 2 个必过门禁', adv.ok && (adv.result.pending || []).length === 2, adv.result && (adv.result.pending || []).map(p => p.id))
  check('拦截文案给出处置指引', adv.ok && /处置：/.test(adv.result.text || ''))

  // ───────────────────────── 场景 4：四类门禁逐个验证 ─────────────────────────
  console.log('\n## 场景 4 · 四类门禁（check / command / artifact / approval）')
  let g = await flow({ op: 'advance', runId, gateId: 'g-check-1' })
  check('check 无证据 → pending 且不推进', g.ok === true && g.result.gate.status === 'pending' && g.result.advanced === false, g.result && g.result.gate.summary)

  g = await flow({ op: 'advance', runId, gateId: 'g-check-1', evidence: 'E2E 实机证据：准备阶段检查项逐条核对完成' })
  check('check 带证据 → pass', g.ok === true && g.result.gate.status === 'pass', g.result && g.result.gate.summary)

  g = await flow({ op: 'advance', runId, gateId: 'g-cmd-1' })
  check('command 门禁自动执行 → pass', g.ok === true && g.result.gate.status === 'pass', g.result && { s: g.result.gate.summary, out: (g.result.gate.output || '').slice(0, 40) })
  check('command 输出包含 echo 内容', g.ok && /e2e-gate-ok/.test((g.result.gate.output || '') + (g.result.gate.summary || '')))
  check('最后一个必过门禁通过 → 连带推进到 impl', g.ok === true && g.result.advanced === true && g.result.stage === 'impl', g.result && { stage: g.result.stage, text: String(g.result.text || '').slice(0, 40) })

  g = await flow({ op: 'advance', runId, gateId: 'g-no-such-gate' })
  check('未知门禁 id 报错并提示现有门禁', g.ok === false && /不存在门禁/.test(g.error), g.error)

  adv = await flow({ op: 'advance', runId, convId: CONV })
  check('impl 阶段门禁未过再推进被拦截', adv.ok === true && adv.result.advanced === false && (adv.result.pending || []).length === 2, adv.result && (adv.result.pending || []).map(p => p.id))

  snap = await get('/api/taskboard/state')
  card = (snap.tasks || []).find(t => t.id === taskId)
  const checkedNow = card ? (card.checklist || []).filter(x => x.checked) : []
  check('卡 DoD 自动回填前 2 项', checkedNow.length === 2 && checkedNow.every(x => ['g-check-1', 'g-cmd-1'].includes(x.gate)), checkedNow.map(x => x.gate))
  check('回填项标记来源 flow', checkedNow.every(x => x.by === 'flow'))

  g = await flow({ op: 'advance', runId, gateId: 'g-art-1' })
  check('artifact 门禁（产物存在）→ pass', g.ok === true && g.result.gate.status === 'pass', g.result && g.result.gate.summary)

  g = await flow({ op: 'advance', runId, gateId: 'g-appr-1' })
  check('approval 门禁未批准 → pending', g.ok === true && g.result.gate.status === 'pending', g.result && g.result.gate.summary)

  g = await flow({ op: 'advance', runId, gateId: 'g-appr-1', approve: false })
  check('advance 传 approve=false 不产生批准（仍 pending）', g.ok === true && g.result.gate.status === 'pending', g.result && g.result.gate.summary)

  g = await flow({ op: 'approve', runId, gateId: 'g-appr-1' })
  check('approve 缺 approve:true 被拒绝', g.ok === false, g.error)

  g = await flow({ op: 'approve', runId, gateId: 'g-appr-1', approve: true, note: 'E2E 用户显式批准' })
  check('用户批准 → approved', g.ok === true && g.result.gate.status === 'approved', g.result && g.result.gate.summary)
  check('批准最后一个必过门禁 → 连带推进到 wrap', g.ok === true && g.result.advanced === true && /收尾/.test(String(g.result.text || '')), g.result && String(g.result.text || '').slice(0, 50))

  // ───────────────────────── 场景 5：中断 / 恢复 ─────────────────────────
  console.log('\n## 场景 5 · 中断与恢复')
  let it = await flow({ op: 'interrupt', runId, reason: 'E2E 中断测试' })
  check('中断实例成功', it.ok === true, it.error)
  let blocked = await flow({ op: 'advance', runId })
  check('中断后推进被拒', blocked.ok === false && /不是进行中/.test(blocked.error), blocked.error)
  let rs = await flow({ op: 'resume', runId })
  check('恢复运行成功', rs.ok === true, rs.error)
  adv = await flow({ op: 'advance', runId, convId: CONV })
  check('恢复后推进仍被收尾门禁拦截（可继续操作）', adv.ok === true && adv.result.advanced === false && (adv.result.pending || []).some(p => p.id === 'g-check-2'), adv.result && (adv.result.pending || []).map(p => p.id))

  // ───────────────────────── 场景 6：流程完成 → 卡转待验收 ─────────────────────────
  console.log('\n## 场景 6 · 流程完成 → 卡自动转待验收')
  g = await flow({ op: 'advance', runId, gateId: 'g-art-bad' })
  check('artifact 门禁（产物不存在）→ fail', g.ok === true && g.result.gate.status === 'fail', g.result && g.result.gate.summary)
  check('非必过门禁失败不阻断推进', g.ok === true && g.result.advanced === false, g.result && { pendingLen: (g.result.pending || []).length })

  g = await flow({ op: 'advance', runId, gateId: 'g-check-2', evidence: 'E2E 收尾核对完成' })
  check('收尾 check → pass', g.ok === true && g.result.gate.status === 'pass')
  check('最后一个必过门禁通过 → 全流程完成（completed=true）', g.ok === true && g.result.completed === true, g.result && String(g.result.text || '').slice(0, 60))

  snap = await get('/api/taskboard/state')
  card = (snap.tasks || []).find(t => t.id === taskId)
  check('流程完成 → 卡自动转「待验收」', !!card && card.status === 'in_review', card && card.status)
  const checkedAll = card ? (card.checklist || []).filter(x => x.checked) : []
  check('6 项 DoD 中 5 项已勾选（失败门禁不勾）', checkedAll.length === 5 && !checkedAll.some(x => x.gate === 'g-art-bad'), checkedAll.map(x => x.gate))

  const advDone = await flow({ op: 'advance', runId, convId: CONV })
  check('已完成实例不可再推进', advDone.ok === false && /已完成/.test(advDone.error), advDone.error)

  // ───────────────────────── 场景 7：验收权限 + 未绑流程对照 ─────────────────────────
  console.log('\n## 场景 7 · 验收权限与未绑流程对照')
  let mv = await tb({ op: 'move', id: taskId, to: 'done', convId: CONV })
  check('人可将「待验收」卡移到「已完成」', mv.ok === true, mv.error)

  const cCard = (snap.tasks || []).find(t => !t.flowId && t.status === 'in_review' && t.id !== taskId)
  check('未绑流程待验收卡存在（对照样本）', !!cCard, cCard && cCard.id)
  check('未绑流程卡无 DoD 清单', !!cCard && (cCard.checklist || []).length === 0)

  let c2 = await tb({ op: 'create', title: 'E2E 未绑流程对照卡', convId: CONV })
  check('未绑流程建卡：无 flowId / 无清单', c2.ok === true && !c2.task.flowId && (c2.task.checklist || []).length === 0, c2.error)
  const c2id = c2.ok ? c2.task.id : ''
  if (c2id) created.tasks.push(c2id)

  let jump = await tb({ op: 'move', id: c2id, to: 'done' })
  check('状态机禁止 todo → done 跳级', jump.ok === false && /无法从/.test(jump.error), jump.error)
  await tb({ op: 'move', id: c2id, to: 'in_progress' })
  await tb({ op: 'move', id: c2id, to: 'in_review' })
  let okDone = await tb({ op: 'move', id: c2id, to: 'done' })
  check('未绑流程卡 in_review → done 允许（无门禁约束）', okDone.ok === true, okDone.error)

  // ───────────────────────── 场景 8：看板 startFlow 入口 ─────────────────────────
  console.log('\n## 场景 8 · 看板「启动流程」入口（startFlow）')
  let c3 = await tb({ op: 'create', title: 'E2E startFlow 入口卡', convId: CONV_SF })
  const c3id = c3.ok ? c3.task.id : ''
  if (c3id) created.tasks.push(c3id)
  let sf1 = await tb({ op: 'startFlow', id: c3id, convId: CONV_SF })
  check('未绑流程卡 startFlow 被拒（提示先绑流程）', sf1.ok === false && /未绑定流程/.test(sf1.error), sf1.error)
  let upd = await tb({ op: 'update', id: c3id, patch: { flowId: FLOW_ID }, convId: CONV_SF })
  check('卡可直接指定 flowId（不启动）', upd.ok === true && upd.task.flowId === FLOW_ID, upd.error)
  let sf2 = await tb({ op: 'startFlow', id: c3id, convId: CONV_SF })
  check('startFlow 启动流程并回写 runId', sf2.ok === true && !!sf2.runId, sf2.error || sf2.runId)
  if (sf2.ok && sf2.runId) created.runs.push(sf2.runId)
  let snap2 = await get('/api/taskboard/state')
  const c3now = (snap2.tasks || []).find(t => t.id === c3id)
  check('startFlow 后卡 runId 已绑定', !!c3now && c3now.runId === (sf2.runId || ''), c3now && c3now.runId)

  // ───────────────────────── 场景 9：边界与异常 ─────────────────────────
  console.log('\n## 场景 9 · 边界与异常')
  let e1 = await flow({ op: 'start', flowId: FLOW_ID, taskId: 't-no-such-task', convId: CONV + '-x' })
  check('绑定不存在的卡 → 明确报错', e1.ok === false && /任务不存在/.test(e1.error), e1.error)
  let e2 = await flow({ op: 'start', flowId: 'no-such-flow', convId: CONV + '-y' })
  check('不存在的流程 → 明确报错', e2.ok === false && /流程不存在/.test(e2.error), e2.error)
  let e3 = await flow({ op: 'advance', runId: 'run-0000000000000-zzzz' })
  check('不存在的 runId → 报错', e3.ok === false, e3.error)
  let e4 = await tb({ op: 'move', id: taskId, to: 'no-such-status' })
  check('未知看板状态 → 报错', e4.ok === false && /未知状态/.test(e4.error), e4.error)
  let e5 = await flow({ op: 'advance', convId: 'e2e-no-such-conv' })
  check('显式会话无实例 → 不越界到其他会话（明确报错）', e5.ok === false && /没有可用的流程实例/.test(e5.error), e5.error)

  // ───────────────────────── 清理 ─────────────────────────
  console.log('\n## 清理测试数据')
  if (KEEP) {
    console.log('（--keep：保留测试数据）', JSON.stringify(created))
  } else {
    for (const id of created.tasks) {
      const t = await tb({ op: 'trash', id })
      console.log('trash', id, t.ok ? 'ok' : (t.error || ''))
    }
    for (const rid of created.runs) {
      const p = await flow({ op: 'purge', runId: rid, confirm: true })
      console.log('purge', rid, p.ok ? 'ok' : (p.error || ''))
    }
    for (const f of created.flowFiles) {
      try { if (fs.existsSync(f)) fs.unlinkSync(f); console.log('删除流程文件', path.relative(ROOT, f)) } catch (e) { console.log('删除失败', f, String(e)) }
    }
    const leftRuns = (await get('/api/flow/state')).runs || []
    check('测试实例已清理（无残留）', !leftRuns.some(x => created.runs.includes(x.runId)), leftRuns.map(x => x.runId))
  }

  // ───────────────────────── 汇总 ─────────────────────────
  console.log('\n' + '─'.repeat(52))
  console.log(`结果：${pass} 通过 / ${fail} 失败`)
  if (failed.length) console.log('失败项：\n - ' + failed.join('\n - '))
  process.exitCode = fail ? 1 : 0
})().catch(e => { console.error('异常中断：', e && e.stack || e); process.exitCode = 2 })
