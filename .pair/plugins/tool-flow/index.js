// ═══════════════════════════════════════════════════════════════
// tool-flow — 工程流程与看板（host 半）
//
// 设计真源：docs/flow-board-plugin-design.md
//   §5 数据契约（FlowDef / FlowRun）· §7 预设与项目类型 · §9 任务接线
//   §10 门禁（check/command/artifact/approval + 证据留档 + 回滚建议）
//   §14 扩展（三层流程库 / 目录 bundle 导出导入 / 项目绑定矩阵 / 三感知）
//
// 硬约束：框架核心零改动 —— 只消费既有 ctx 能力：
//   tools.register（工具面）· http.register（只读接口）· registerClientMethod（写 RPC）
//   systemPrompt.section + variable（agent 每轮感知）· hooks.register（门禁）
//   commands.register（/flow*）· registerSettings（强度配置）
//   fs（工作区）· install（安装目录数据区：全局流程库 flowlib/ + 预设 assets/）
//
// 数据落点（真相全部在磁盘，L0 唯一真相）：
//   预设 preset  <InstallDir>/.pair/plugins/tool-flow/assets/flows/*.json   只读 ctx.install
//   全局 global  <InstallDir>/.pair/flowlib/*.json                         读写 ctx.install
//   项目 project <ws>/.pair/flows/*.json                                   读写 ctx.fs
//   绑定         <ws>/.pair/flows/_binding.json
//   实例         <ws>/.pair/flow-runs/<runId>.json
//   证据         <ws>/.pair/flow-runs/evidence/<runId>/<gateId>.{json,log}
//   任务         <ws>/.pair/tasks/<id>.json（复用宿主任务真相：subject/status/convId）
//
// [注意] 诚实限制（已核核心源码）：
//   ① hook.IsBlocking 仅对 PreToolUse / UserPromptSubmit 生效 → 设计 §10.1 的
//      「Stop 钩子硬拦会话收尾」在当前核心不可实现（需改核心，超出零改动约束）。
//      本插件以「systemPrompt 每轮注入 + PreToolUse 拦破坏性操作 + 工具返回摘要」替代。
//   ② 系统提示 provider 在组装期求值、无会话上下文 → 注入按**项目级**（最近活跃 run）聚合；
//      多会话并行时以显式传 convId 的工具调用为准。
// ═══════════════════════════════════════════════════════════════
return {
  name: 'tool-flow',
  purpose: '工程流程与看板：三层流程库（预设/全局/项目）+ 阶段门禁（check/command/artifact/approval）+ 证据留档 + 任务真相复用 + 项目绑定矩阵 + bundle 导出导入',
  inject: ['fs', 'install', 'logger', 'commands', 'http', 'bash'],
  apply(ctx, config) {

    // ── ① 日志与常量 ─────────────────────────────────────────────
    var LOG = null
    try { LOG = ctx.logger('tool-flow') } catch (e) { LOG = null }
    function logInfo(m) { try { if (LOG && LOG.info) LOG.info(m) } catch (e) { } }
    function logWarn(m) { try { if (LOG && LOG.warn) LOG.warn(m) } catch (e) { } }

    var P_FLOWS = '.pair/flows'
    var P_BINDING = '.pair/flows/_binding.json'
    var P_RUNS = '.pair/flow-runs'
    var P_EVID = '.pair/flow-runs/evidence'
    var P_TASKS = '.pair/tasks'
    var P_EXPORT = '.pair/flow-exports'
    var I_LIB = '.pair/flowlib'                                  // 相对安装根（ctx.install）
    var I_PRESET = '.pair/plugins/tool-flow/assets/flows'
    var SCHEMA = 1
    var GATE_KINDS = ['check', 'command', 'artifact', 'approval']
    var BOARD_KINDS = ['kanban', 'checklist', 'timeline', 'swimlane']
    var SCOPES = ['project', 'global', 'preset']                 // 解析优先级

    // ── ② 基础助手 ───────────────────────────────────────────────
    function nowIso() { return new Date().toISOString() }
    function pad3(n) { var s = String(n); while (s.length < 3) s = '0' + s; return s }
    function isNum(v) { return typeof v === 'number' && isFinite(v) }
    function clone(o) { return JSON.parse(JSON.stringify(o)) }
    function safeName(s) {
      var t = String(s == null ? '' : s).replace(/[^A-Za-z0-9._-]/g, '_')
      t = t.replace(/^\.+/, '_')
      return t || 'unnamed'
    }
    function parseQuery(raw) {
      var out = {}
      var parts = String(raw == null ? '' : raw).split('&')
      for (var i = 0; i < parts.length; i++) {
        var kv = parts[i]
        if (!kv) continue
        var j = kv.indexOf('=')
        var k = j < 0 ? kv : kv.slice(0, j)
        var v = j < 0 ? '' : kv.slice(j + 1)
        try { out[decodeURIComponent(k)] = decodeURIComponent(v) } catch (e) { out[k] = v }
      }
      return out
    }
    function sep(p) { return p.charAt(p.length - 1) === '/' ? p : p + '/' }

    // 工作区（ctx.fs）——注意：ctx.fs.writeFile **不**自动建父目录，写前必须 ensureDir
    function ensureDir(dir) { try { if (!ctx.fs.exists(dir)) ctx.fs.mkdir(dir, true) } catch (e) { } }
    function fsExists(p) { try { return !!ctx.fs.exists(p) } catch (e) { return false } }
    function fsRead(p) { try { var s = ctx.fs.readFile(p); return (typeof s === 'string' && s.length) ? s : null } catch (e) { return null } }
    function fsReadJson(p) { var s = fsRead(p); if (!s) return null; try { return JSON.parse(s) } catch (e) { return null } }
    function fsWrite(p, s) { ensureDir(parentOf(p)); try { ctx.fs.writeFile(p, s); return true } catch (e) { logWarn('写失败 ' + p + ': ' + e); return false } }
    function fsWriteJson(p, o) { return fsWrite(p, JSON.stringify(o, null, 2)) }
    function fsListNames(dir) { try { var n = ctx.fs.readdir(dir); return Array.isArray(n) ? n : [] } catch (e) { return [] } }
    function parentOf(p) { var i = String(p).lastIndexOf('/'); return i < 0 ? '.' : String(p).slice(0, i) }
    function wsRoot() { try { return String(ctx.app.workspaceRoot || '') } catch (e) { return '' } }

    // 安装数据区（ctx.install）——writeFile 自动建父目录
    function libIds() {
      try {
        var n = ctx.install.readdir(I_LIB)
        return (Array.isArray(n) ? n : []).filter(function (x) { return /\.json$/.test(x) })
          .map(function (x) { return x.replace(/\.json$/, '') }).sort()
      } catch (e) { return [] }
    }
    function libRead(id) {
      try {
        var p = I_LIB + '/' + safeName(id) + '.json'
        if (!ctx.install.exists(p)) return null
        return JSON.parse(ctx.install.readFile(p))
      } catch (e) { return null }
    }
    function libWrite(id, def) {
      try { ctx.install.writeFile(I_LIB + '/' + safeName(id) + '.json', JSON.stringify(def, null, 2)); return true }
      catch (e) { logWarn('全局库写失败 ' + id + ': ' + e); return false }
    }
    function libRemove(id) {
      try { ctx.install.rm(I_LIB + '/' + safeName(id) + '.json'); return true } catch (e) { return false }
    }
    function presetIds() {
      try {
        var n = ctx.install.readdir(I_PRESET)
        return (Array.isArray(n) ? n : []).filter(function (x) { return /\.json$/.test(x) })
          .map(function (x) { return x.replace(/\.json$/, '') }).sort()
      } catch (e) { return [] }
    }
    function presetRead(id) {
      try {
        var p = I_PRESET + '/' + safeName(id) + '.json'
        if (!ctx.install.exists(p)) return null
        return JSON.parse(ctx.install.readFile(p))
      } catch (e) { return null }
    }

    // ── ③ 三层流程库（project > global > preset）─────────────────
    function projectIds() {
      return fsListNames(P_FLOWS).filter(function (x) { return /\.json$/.test(x) && x.charAt(0) !== '_' })
        .map(function (x) { return x.replace(/\.json$/, '') }).sort()
    }
    function idsAt(scope) {
      if (scope === 'project') return projectIds()
      if (scope === 'global') return libIds()
      if (scope === 'preset') return presetIds()
      return []
    }
    function flowAt(scope, id) {
      if (!id) return null
      if (scope === 'project') return fsReadJson(P_FLOWS + '/' + safeName(id) + '.json')
      if (scope === 'global') return libRead(id)
      if (scope === 'preset') return presetRead(id)
      return null
    }
    function flowPathAt(scope, id) {
      if (scope === 'project') return P_FLOWS + '/' + safeName(id) + '.json'
      if (scope === 'global') return I_LIB + '/' + safeName(id) + '.json'
      return null
    }
    function resolveFlow(id) {
      for (var i = 0; i < SCOPES.length; i++) {
        var d = flowAt(SCOPES[i], id)
        if (d && typeof d === 'object') return { def: d, scope: SCOPES[i], id: id }
      }
      return null
    }
    // allFlows：三层合并（上层覆盖下层，标 shadowed）
    function allFlows() {
      var out = []
      var seen = {}
      for (var i = SCOPES.length - 1; i >= 0; i--) {          // preset → global → project
        var scope = SCOPES[i]
        var ids = idsAt(scope)
        for (var j = 0; j < ids.length; j++) {
          var id = ids[j]
          var def = flowAt(scope, id)
          if (!def) continue
          if (seen[id]) continue
          var rec = { id: id, title: def.title || id, version: def.version || 1, scope: scope, readOnly: scope === 'preset', stages: (def.stages || []).length, projectTypes: def.projectTypes || (def.appliesTo && def.appliesTo.types) || [], taskTypes: def.taskTypes || [] }
          seen[id] = 1
          for (var k = 0; k < SCOPES.length; k++) {           // 标「被上层同名的层」
            if (SCOPES[k] === scope) continue
            var hi = SCOPES.indexOf(scope), lo = SCOPES.indexOf(SCOPES[k])
            if (lo > hi && flowAt(SCOPES[k], id)) { rec.overriddenBy = SCOPES[k]; break }
          }
          out.push(rec)
        }
      }
      return out.sort(function (a, b) { return a.id < b.id ? -1 : (a.id > b.id ? 1 : 0) })
    }

    // ── 校验（导入/编辑/启动共用）─────────────────────────────────
    function validateFlow(def) {
      var errs = []
      if (!def || typeof def !== 'object') return { ok: false, errors: ['流程定义不是对象'] }
      if (!def.id || typeof def.id !== 'string') errs.push('缺少 id')
      else if (!/^[A-Za-z0-9._-]+$/.test(def.id)) errs.push('id 只允许字母/数字/._-（当前: ' + def.id + '）')
      if (!Array.isArray(def.stages) || !def.stages.length) errs.push('stages 必须是非空数组')
      var stageIds = {}
      var stages = Array.isArray(def.stages) ? def.stages : []
      for (var i = 0; i < stages.length; i++) {
        var st = stages[i] || {}
        if (!st.id) errs.push('阶段 #' + (i + 1) + ' 缺少 id')
        else if (stageIds[st.id]) errs.push('阶段 id 重复: ' + st.id)
        else stageIds[st.id] = 1
        var gates = Array.isArray(st.gates) ? st.gates : []
        var gateIds = {}
        for (var j = 0; j < gates.length; j++) {
          var g = gates[j] || {}
          if (!g.id) errs.push('阶段 ' + st.id + ' 的第 ' + (j + 1) + ' 个门禁缺少 id')
          else if (gateIds[g.id]) errs.push('门禁 id 重复: ' + g.id)
          else gateIds[g.id] = 1
          if (GATE_KINDS.indexOf(g.kind) < 0) errs.push('门禁 ' + (g.id || '?') + ' kind 非法: ' + g.kind + '（合法: ' + GATE_KINDS.join('/') + '）')
          if (g.kind === 'command' && !(g.cmd && String(g.cmd).replace(/\s/g, ''))) errs.push('command 门禁 ' + g.id + ' 缺少 cmd')
          if (g.kind === 'artifact' && !(g.artifact && String(g.artifact).replace(/\s/g, ''))) errs.push('artifact 门禁 ' + g.id + ' 缺少 artifact')
          if (g.onFail && ['advise', 'rollback'].indexOf(g.onFail) < 0) errs.push('门禁 ' + g.id + ' onFail 非法: ' + g.onFail)
        }
      }
      if (def.board && def.board.kind && BOARD_KINDS.indexOf(def.board.kind) < 0) errs.push('board.kind 非法: ' + def.board.kind)
      return { ok: errs.length === 0, errors: errs }
    }

    // ── ④ 项目类型识别与推荐（§7）────────────────────────────────
    var TYPE_SIGNALS = [
      { type: 'plugin', signals: ['.pair/plugins/', 'plugins-dist/', 'plugins-src/'] },
      { type: 'go', signals: ['go.mod', 'go.work'] },
      { type: 'web', signals: ['vite.config.js', 'vite.config.mjs', 'vite.config.ts', 'next.config.js', 'package.json'] },
      { type: 'python', signals: ['requirements.txt', 'pyproject.toml', 'setup.py'] },
      { type: 'lib', signals: ['Cargo.toml', 'cmd/', 'bin/'] },
      { type: 'docs', signals: ['docs/'] },
    ]
    var TYPE_TO_FLOW = {
      plugin: 'plugin-dev', go: 'go-backend-feature', web: 'web-frontend-feature',
      python: 'python-feature', lib: 'lib-cli', cli: 'lib-cli', docs: 'docs-office',
      any: 'generic',
    }
    function detectTypes() {
      var found = []
      for (var i = 0; i < TYPE_SIGNALS.length; i++) {
        var t = TYPE_SIGNALS[i]
        for (var j = 0; j < t.signals.length; j++) {
          if (fsExists(t.signals[j])) { found.push(t.type); break }
        }
      }
      return found
    }
    function detectCachePath() { return P_FLOWS + '/_detected.json' }
    function recommendFlow() {
      var types = detectTypes()
      var rec = null
      for (var i = 0; i < types.length; i++) {
        if (TYPE_TO_FLOW[types[i]]) { rec = { flowId: TYPE_TO_FLOW[types[i]], reason: '检测到 ' + types[i] + ' 项目特征', types: types }; break }
      }
      if (!rec) rec = { flowId: 'generic', reason: '未识别到特定项目类型信号，回落通用流程', types: types }
      var d = resolveFlow(rec.flowId)
      rec.flowTitle = d ? (d.def.title || rec.flowId) : rec.flowId
      rec.available = !!d
      if (!d) rec.reason += '（注意：该流程当前不在三层库中）'
      fsWriteJson(detectCachePath(), { at: nowIso(), types: types, recommended: rec.flowId })
      return rec
    }

    // ── ⑤ 项目绑定（§14.4）───────────────────────────────────────
    function loadBinding() {
      var b = fsReadJson(P_BINDING)
      if (!b || typeof b !== 'object') b = { version: 1, bindings: {} }
      if (!b.bindings || typeof b.bindings !== 'object') b.bindings = {}
      return b
    }
    function saveBinding(b) { return fsWriteJson(P_BINDING, b) }
    function bindingFor(project) { var b = loadBinding(); return b.bindings[project] || null }
    function projectRoots() {
      var out = []
      try { var f = ctx.app.folders; if (Array.isArray(f)) out = f.map(String).filter(function (x) { return !!x }) } catch (e) { }
      if (!out.length) { var w = wsRoot(); if (w) out = [w] }
      return out
    }
    function bindingsMatrix() {
      var b = loadBinding()
      var roots = projectRoots()
      var keys = Object.keys(b.bindings)
      for (var i = 0; i < keys.length; i++) if (roots.indexOf(keys[i]) < 0) roots.push(keys[i])
      var runs = listRuns()
      return roots.map(function (root) {
        var bd = b.bindings[root] || null
        var mine = runs.filter(function (r) { return r.project === root && r.status === 'active' })
        var run = mine.length ? mine[mine.length - 1] : null
        var stageInfo = run ? stageProgress(run) : null
        return {
          project: root,
          flowId: bd ? bd.flowId : '',
          scope: bd ? (bd.scope || 'project') : '',
          note: bd ? (bd.note || '') : '',
          runId: run ? run.runId : '',
          stage: stageInfo ? stageInfo.label : '',
          blocked: stageInfo ? stageInfo.pending : 0,
        }
      })
    }

    // ── ⑥ 任务（复用宿主任务真相 §5.3 / §9 路径 A）────────────────
    function newTaskId() { return String(Date.now()) + pad3(Math.floor(Math.random() * 1000)) }
    function createTask(subject, description, convId) {
      var id = newTaskId()
      var t = {
        id: id, subject: String(subject || ''),
        description: String(description || ''),
        status: 'pending', dependencies: [],
        convId: String(convId || ''),
        created_at: nowIso(), updated_at: nowIso(),
      }
      fsWriteJson(P_TASKS + '/' + id + '.json', t)
      return id
    }
    function updateTaskStatus(id, status) {
      var p = P_TASKS + '/' + id + '.json'
      var t = fsReadJson(p)
      if (!t) return false
      t.status = status
      t.updated_at = nowIso()
      fsWriteJson(p, t)
      return true
    }
    function createStageTasks(run, stageDef, stageRun) {
      var ids = []
      var list = Array.isArray(stageDef && stageDef.tasks) ? stageDef.tasks : []
      for (var i = 0; i < list.length; i++) {
        ids.push(createTask('[' + stageDef.title + '] ' + String(list[i]), '流程 ' + run.flowId + ' · 阶段 ' + stageDef.id, run.convId))
      }
      stageRun.taskIds = (stageRun.taskIds || []).concat(ids)
      return ids
    }
    function completeStageTasks(stageRun) {
      var ids = stageRun.taskIds || []
      for (var i = 0; i < ids.length; i++) {
        var p = P_TASKS + '/' + ids[i] + '.json'
        if (!fsExists(p)) continue
        updateTaskStatus(ids[i], 'completed')
      }
    }

    // ── ⑦ 流程实例 / 门禁 / 证据 / 推进 ──────────────────────────
    function runPath(runId) { return P_RUNS + '/' + safeName(runId) + '.json' }
    function loadRun(runId) { return fsReadJson(runPath(runId)) }
    function saveRun(run) { run.updatedAt = nowIso(); fsWriteJson(runPath(run.runId), run); return run }
    function listRuns() {
      return fsListNames(P_RUNS).filter(function (x) { return /\.json$/.test(x) })
        .map(function (x) { return fsReadJson(P_RUNS + '/' + x) })
        .filter(function (r) { return r && r.runId })
        .sort(function (a, b) { return String(a.updatedAt || '') < String(b.updatedAt || '') ? -1 : 1 })
    }
    function activeRuns() { return listRuns().filter(function (r) { return r.status === 'active' }) }
    function pickRun(opts) {
      var o = opts || {}
      if (o.runId) return loadRun(o.runId)
      var act = activeRuns()
      if (o.convId) {
        var mine = act.filter(function (r) { return r.convId === o.convId })
        if (mine.length) return mine[mine.length - 1]
      }
      if (act.length) return act[act.length - 1]
      var all = listRuns()
      return all.length ? all[all.length - 1] : null
    }
    function latestActiveRun() { var a = activeRuns(); return a.length ? a[a.length - 1] : null }
    function runFlow(run) { return resolveFlow(run.flowId) }
    function stageIdx(run) {
      var list = run.stages || []
      for (var i = 0; i < list.length; i++) if (list[i].id === run.currentStage) return i
      for (var j = 0; j < list.length; j++) if (list[j].status === 'in_progress') return j
      return 0
    }
    function stageProgress(run) {
      var flow = runFlow(run)
      var list = run.stages || []
      var idx = stageIdx(run)
      var st = list[idx] || {}
      var def = flow && flow.def.stages ? flow.def.stages[idx] : null
      var pending = def ? pendingRequired(st, def) : []
      return {
        idx: idx, total: list.length,
        label: (idx + 1) + '/' + list.length + (def ? ' ' + def.title : ''),
        stageId: st.id || '', pending: pending.length,
      }
    }
    function findGate(stageDef, gateId) {
      var gates = (stageDef && stageDef.gates) || []
      for (var i = 0; i < gates.length; i++) if (gates[i].id === gateId) return gates[i]
      return null
    }
    function gateResult(stageRun, gateId) {
      var list = stageRun.gateResults || []
      for (var i = list.length - 1; i >= 0; i--) if (list[i].id === gateId) return list[i]
      return null
    }
    // 未通过的 required 门禁（推进的硬条件）
    function pendingRequired(stageRun, stageDef) {
      var gates = (stageDef && stageDef.gates) || []
      var out = []
      for (var i = 0; i < gates.length; i++) {
        var g = gates[i]
        if (!g.required) continue
        var r = gateResult(stageRun, g.id)
        var ok = r && (r.status === 'pass' || r.status === 'approved')
        if (!ok) out.push(g)
      }
      return out
    }
    function blockingPending(stageRun, stageDef) {
      return pendingRequired(stageRun, stageDef).filter(function (g) { return g.blocking })
    }

    // 证据留档（§10.3）
    function writeEvidence(run, gate, rec) {
      var dir = P_EVID + '/' + safeName(run.runId)
      ensureDir(dir)
      if (rec.output) {
        var log = String(rec.output)
        if (log.length > 40000) log = log.slice(log.length - 40000)
        fsWrite(dir + '/' + safeName(gate.id) + '.log', log)
        rec.evidenceFile = dir + '/' + safeName(gate.id) + '.log'
      }
      fsWriteJson(dir + '/' + safeName(gate.id) + '.json', rec)
      if (!rec.evidenceFile) rec.evidenceFile = dir + '/' + safeName(gate.id) + '.json'
      return rec.evidenceFile
    }
    // 执行/记录单个门禁
    function evaluateGate(run, gate, opts) {
      var o = opts || {}
      var rec = { id: gate.id, kind: gate.kind, text: gate.text || '', at: nowIso() }
      if (gate.kind === 'command') {
        rec.cmd = String(gate.cmd || '')
        var r = null
        try { r = ctx.bash.exec(rec.cmd, '', 0) } catch (e) { rec.status = 'fail'; rec.summary = '执行异常: ' + e }
        if (!rec.status) {
          var out = String((r && r.output) || '')
          var err = String((r && r.error) || '')
          rec.output = out + (err ? ('\n[stderr] ' + err) : '')
          if (err) { rec.status = 'fail'; rec.summary = '命令失败: ' + err.slice(0, 400) }
          else { rec.status = 'pass'; rec.summary = '命令通过（输出 ' + rec.output.length + " 字符）" }
        }
      } else if (gate.kind === 'artifact') {
        var target = String(gate.artifact || '')
        var isDir = /\/$/.test(target)
        var exists = false
        if (isDir) exists = fsListNames(target.replace(/\/$/, '')).length > 0
        else exists = fsExists(target)
        rec.artifact = target
        rec.status = exists ? 'pass' : 'fail'
        rec.summary = exists ? '产物存在: ' + target : '产物不存在: ' + target
      } else if (gate.kind === 'approval') {
        if (o.approve === true) {
          rec.status = 'approved'; rec.by = 'user'; rec.note = String(o.note || '')
          rec.summary = '用户已批准' + (o.note ? ('（' + o.note + '）') : '')
        } else {
          rec.status = 'pending'; rec.summary = '待用户批准（approval 门禁不得自动通过）'
        }
      } else {                                                 // check
        var ev = String(o.evidence || '').trim()
        if (ev) { rec.status = 'pass'; rec.summary = ev.slice(0, 800) }
        else { rec.status = 'pending'; rec.summary = '待提供证据（check 门禁需说明如何确认）' }
      }
      rec.evidence = String(o.evidence || '') || rec.summary
      writeEvidence(run, gate, rec)
      return rec
    }
    function upsertGateResult(stageRun, rec) {
      if (!stageRun.gateResults) stageRun.gateResults = []
      var replaced = false
      for (var i = 0; i < stageRun.gateResults.length; i++) {
        if (stageRun.gateResults[i].id === rec.id) { stageRun.gateResults[i] = rec; replaced = true; break }
      }
      if (!replaced) stageRun.gateResults.push(rec)
    }
    // 失败回滚建议（§10.3；生成建议，绝不自动执行高风险命令）
    function rollbackAdvice(run, gate, rec) {
      var files = (run.modifiedFiles || []).slice(0, 10)
      var opts = [
        { kind: 'inspect', cmd: 'git diff --stat', risk: 'none', text: '先看改动范围' },
        { kind: 'stash', cmd: 'git stash push -m flow-' + run.runId, risk: 'medium', text: '暂存本次改动以便对比' },
      ]
      if (files.length) opts.push({ kind: 'discard', cmd: 'git checkout -- ' + files.join(' '), risk: 'high', text: '丢弃本次改动（必须用户批准）' })
      if (run.gitHead) opts.push({ kind: 'reset', cmd: 'git reset --hard ' + run.gitHead, risk: 'high', text: '回到流程起点（危险，必须用户批准）' })
      return {
        failedGate: gate.id, reason: rec.summary || '门禁未通过',
        evidenceFile: rec.evidenceFile || '', modifiedFiles: files,
        note: run.gitHead ? '' : '无 git 基线（未记录 gitHead）→ 仅能按 modifiedFiles 人工处理',
        rollbackOptions: opts,
      }
    }
    function gitHead() {
      try {
        var r = ctx.bash.exec('git rev-parse HEAD', '', 30)
        if (r && !r.error) return String(r.output || '').trim().slice(0, 40)
      } catch (e) { }
      return ''
    }

    // 启动流程（§14.4 解析顺序：会话级临时 → 项目绑定 → 类型推荐 → 全局默认）
    function startRun(opts) {
      var o = opts || {}
      var project = String(o.project || wsRoot() || '')
      var picked = null
      if (o.flowId) picked = { flowId: String(o.flowId), source: 'explicit' }
      if (!picked) {
        var bd = bindingFor(project)
        if (bd && bd.flowId) picked = { flowId: bd.flowId, source: 'binding' }
      }
      if (!picked) {
        var rec = recommendFlow()
        picked = { flowId: rec.flowId, source: 'recommend' }
      }
      var flow = resolveFlow(picked.flowId)
      if (!flow) return { ok: false, error: '流程不存在（三层库里没有）: ' + picked.flowId }
      var v = validateFlow(flow.def)
      if (!v.ok) return { ok: false, error: '流程定义校验失败：' + v.errors.join('；') }
      // 判重按「会话」优先：同会话已有活跃实例才拒绝；无 convId（项目级实例）时按项目判重。
      // （多会话并行各自跑自己的流程是设计允许的 —— §5.2 flow run 按 convId 绑定）
      var exist = activeRuns().filter(function (r) {
        if (o.convId) return r.convId === o.convId
        return !!(project && r.project === project)
      })
      if (exist.length && !o.force) {
        return {
          ok: false, error: '已有进行中的流程实例：' + exist[0].runId + '（流程 ' + exist[0].flowId + '）。',
          activeRun: exist[0], hint: 'force=true 可另起一个实例（不推荐）；flow_status 查看进度。',
        }
      }
      var runId = 'run-' + Date.now() + '-' + Math.random().toString(36).slice(2, 6)
      var stages = []
      var defStages = flow.def.stages
      for (var i = 0; i < defStages.length; i++) {
        stages.push({
          id: defStages[i].id, status: i === 0 ? 'in_progress' : 'pending',
          startedAt: i === 0 ? nowIso() : '', completedAt: '',
          taskIds: [], gateResults: [],
        })
      }
      var run = {
        runId: runId, convId: String(o.convId || ''), project: project,
        flowId: flow.def.id, flowVersion: flow.def.version || 1, flowScope: flow.scope,
        status: 'active', currentStage: defStages[0].id,
        stages: stages, createdAt: nowIso(), updatedAt: nowIso(),
        gitHead: gitHead(), modifiedFiles: [],
        evidenceDir: P_EVID + '/' + runId + '/',
      }
      createStageTasks(run, defStages[0], stages[0])
      saveRun(run)
      return { ok: true, run: run, source: picked.source, flowScope: flow.scope }
    }

    // 推进（含门禁记录）
    function advanceRun(opts) {
      var o = opts || {}
      var run = pickRun(o)
      if (!run) return { ok: false, error: '没有可用的流程实例（用 flow_start 启动）' }
      if (run.status !== 'active') return { ok: false, error: '流程实例已结束: ' + run.status, run: run }
      var flow = runFlow(run)
      if (!flow) return { ok: false, error: '流程定义已不存在: ' + run.flowId }
      var idx = stageIdx(run)
      var stageRun = run.stages[idx]
      var stageDef = flow.def.stages[idx]
      if (!stageDef || !stageRun) return { ok: false, error: '阶段状态与流程定义不一致（索引 ' + idx + '）' }

      // ① 指定门禁 → 执行/记录
      var gateRec = null
      if (o.gateId) {
        var gate = findGate(stageDef, o.gateId)
        if (!gate) return { ok: false, error: '阶段「' + stageDef.title + '」中不存在门禁 ' + o.gateId + '（现有: ' + ((stageDef.gates || []).map(function (g) { return g.id }).join(', ') || '无') + '）' }
        gateRec = evaluateGate(run, gate, o)
        upsertGateResult(stageRun, gateRec)
        saveRun(run)
        var passed = gateRec.status === 'pass' || gateRec.status === 'approved'
        if (!passed) {
          var out = {
            ok: true, advanced: false, gate: gateRec,
            text: '门禁 ' + gate.id + ' 未通过：' + (gateRec.summary || '') + '\n' + gateRecText(run, gate, gateRec, stageDef.title),
          }
          if (gate.onFail === 'rollback') out.rollback = rollbackAdvice(run, gate, gateRec)
          return out
        }
      }

      // ② 检查本阶段 required 门禁
      var pending = pendingRequired(stageRun, stageDef)
      if (pending.length && !o.force) {
        return {
          ok: true, advanced: false, gate: gateRec,
          pending: pending.map(function (g) { return { id: g.id, kind: g.kind, text: g.text, cmd: g.cmd || '', artifact: g.artifact || '' } }),
          text: '阶段「' + stageDef.title + '」还有 ' + pending.length + ' 个必过门禁未通过，不能推进：\n' +
            pending.map(function (g) { return '  · ' + g.id + '（' + g.kind + '）' + g.text + (g.cmd ? ' | cmd: ' + g.cmd : (g.artifact ? ' | artifact: ' + g.artifact : '')) }).join('\n') +
            '\n处置：check 门禁用 flow_advance {gateId, evidence}；command 门禁用 flow_advance {gateId} 代跑；artifact 门禁确认产物后 flow_advance {gateId}；approval 门禁必须用户批准（approve=true）。',
        }
      }

      // ③ 完成本阶段 → 进入下一阶段
      stageRun.status = 'completed'
      stageRun.completedAt = nowIso()
      completeStageTasks(stageRun)
      var nextIdx = idx + 1
      if (nextIdx >= run.stages.length) {
        run.status = 'completed'
        run.currentStage = ''
        saveRun(run)
        return { ok: true, advanced: true, completed: true, gate: gateRec, text: '流程「' + flow.def.title + '」全部阶段已完成 [全部通过]（run ' + run.runId + '）' }
      }
      var nextRun = run.stages[nextIdx]
      var nextDef = flow.def.stages[nextIdx]
      nextRun.status = 'in_progress'
      nextRun.startedAt = nowIso()
      run.currentStage = nextDef.id
      createStageTasks(run, nextDef, nextRun)
      saveRun(run)
      return {
        ok: true, advanced: true, stage: nextDef.id, title: nextDef.title, gate: gateRec,
        text: '阶段 ' + (idx + 1) + '/' + run.stages.length + '「' + stageDef.title + '」已完成 → 进入阶段 ' + (nextIdx + 1) + '「' + nextDef.title + '」' +
          (nextDef.goal ? '\n阶段目标：' + nextDef.goal : '') +
          '\n门禁：' + ((nextDef.gates || []).map(function (g) { return g.id + '(' + g.kind + ')' }).join('、') || '无') +
          '\n已为该阶段建任务（见右栏任务面板）。',
      }
    }
    function gateRecText(run, gate, rec, stageTitle) {
      var lines = []
      lines.push('阶段：' + stageTitle + ' · 门禁：' + gate.id + '（' + gate.kind + '，' + (gate.required ? '必过' : '可选') + '）')
      if (rec.cmd) lines.push('命令：' + rec.cmd)
      if (rec.evidenceFile) lines.push('证据：' + rec.evidenceFile)
      if (gate.kind === 'approval') lines.push('审批卡点：需用户显式批准（对话确认或看板「批准」按钮），插件不会自动通过。')
      if (gate.kind === 'check') lines.push('请在 evidence 参数中写明「如何确认该门禁成立」。')
      if (gate.kind === 'artifact') lines.push('确认产物路径存在：' + (gate.artifact || ''))
      return lines.join('\n')
    }
    function statusText(run) {
      if (!run) return '当前没有流程实例。用 flow_start 启动；flow_list 查看可用流程。'
      var flow = runFlow(run)
      var sp = stageProgress(run)
      var lines = []
      lines.push('流程 ' + run.flowId + (flow ? '「' + flow.def.title + '」' : '') + '（' + (run.flowScope || '?') + ' 级 v' + run.flowVersion + '）· 状态 ' + run.status)
      lines.push('进度：阶段 ' + sp.label + (run.convId ? ' · 会话 ' + run.convId : ''))
      var stages = run.stages || []
      for (var i = 0; i < stages.length; i++) {
        var st = stages[i]
        var def = flow && flow.def.stages ? flow.def.stages[i] : null
        var gates = (def && def.gates) || []
        var marks = gates.map(function (g) {
          var r = gateResult(st, g.id)
          var s = r ? r.status : 'todo'
          var icon = (s === 'pass' || s === 'approved') ? 'PASS' : (s === 'fail' ? 'FAIL' : (s === 'pending' ? 'PEND' : '----'))
          return icon + g.id
        })
        lines.push('  ' + (st.status === 'completed' ? '[x]' : (st.status === 'in_progress' ? '[>]' : '[ ]')) + ' ' + (i + 1) + '. ' + (def ? def.title : st.id) + (marks.length ? '  ' + marks.join(' ') : ''))
      }
      if (run.modifiedFiles && run.modifiedFiles.length) lines.push('本次改动文件（' + run.modifiedFiles.length + '）：' + run.modifiedFiles.slice(0, 8).join(', ') + (run.modifiedFiles.length > 8 ? ' …' : ''))
      if (run.gitHead) lines.push('git 基线：' + run.gitHead)
      return lines.join('\n')
    }

    // ── ⑧ 目录 bundle 导出/导入（§14.11）─────────────────────────
    function bundleDirOf(id) { return P_EXPORT + '/flow-' + safeName(id) }
    function bundleReadme(def) {
      var L = []
      L.push('# ' + (def.title || def.id) + '（流程包）')
      L.push('')
      L.push('> 由 tool-flow 导出（schemaVersion ' + SCHEMA + '）。导入：flow_import {dir:"' + bundleDirOf(def.id) + '", scope:"project"}')
      L.push('')
      if (def.library && (def.library.tags || []).length) L.push('- 标签：' + (def.library.tags || []).join(', '))
      if (def.library && def.library.author) L.push('- 作者：' + def.library.author)
      L.push('- 版本：v' + (def.version || 1))
      L.push('- 适用项目类型：' + ((def.projectTypes || (def.appliesTo && def.appliesTo.types) || ['any']).join(', ')))
      L.push('')
      L.push('## 阶段与门禁')
      L.push('')
      var stages = def.stages || []
      for (var i = 0; i < stages.length; i++) {
        var st = stages[i]
        L.push('### ' + (i + 1) + '. ' + (st.title || st.id) + '（' + st.id + '）')
        if (st.goal) L.push('- 目标：' + st.goal)
        if (st.tasks && st.tasks.length) L.push('- 任务：' + st.tasks.join('；'))
        var gates = st.gates || []
        if (gates.length) {
          L.push('- 门禁：')
          for (var j = 0; j < gates.length; j++) {
            var g = gates[j]
            L.push('  - `' + g.id + '`（' + g.kind + (g.required ? '，必过' : '') + (g.blocking ? '，阻塞' : '') + '）' + (g.text || '') + (g.cmd ? '　cmd: `' + g.cmd + '`' : '') + (g.artifact ? '　artifact: `' + g.artifact + '`' : ''))
          }
        }
        L.push('')
      }
      L.push('## 如何过关')
      L.push('')
      L.push('- `check`：`flow_advance {gateId, evidence:"如何确认"}`')
      L.push('- `command`：`flow_advance {gateId}`（插件代跑，输出落在 `.pair/flow-runs/evidence/<runId>/<gateId>.log`）')
      L.push('- `artifact`：产物就位后 `flow_advance {gateId}`')
      L.push('- `approval`：必须用户显式批准（看板按钮或对话确认后 `approve:true`）')
      return L.join('\n')
    }
    function exportBundle(id, scope, to) {
      var found = resolveFlow(id)
      if (!found) return { ok: false, error: '流程不存在: ' + id }
      if (scope && scope !== found.scope) {
        var alt = flowAt(scope, id)
        if (!alt) return { ok: false, error: '在 ' + scope + ' 层找不到流程 ' + id }
        found = { def: alt, scope: scope, id: id }
      }
      var dir = to ? String(to) : bundleDirOf(id)
      var def = clone(found.def)
      def.schemaVersion = SCHEMA
      fsWriteJson(dir + '/flow.json', def)
      fsWrite(dir + '/README.md', bundleReadme(def))
      var stages = def.stages || []
      for (var i = 0; i < stages.length; i++) {
        var gates = stages[i].gates || []
        for (var j = 0; j < gates.length; j++) {
          var g = gates[j]
          if (g.kind === 'command' && g.cmd) fsWrite(dir + '/gates/' + safeName(g.id) + '.cmd', String(g.cmd) + '\n')
          if (g.required) {
            fsWrite(dir + '/templates/' + safeName(g.id) + '.md',
              '# 证据模板 · ' + g.id + '\n\n- 阶段：' + stages[i].title + '\n- 门禁：' + (g.text || '') + '\n- 类型：' + g.kind + '\n- 确认方式：\n- 命令/产物：' + (g.cmd || g.artifact || '-') + '\n- 结论：pass / fail\n- 备注：\n')
          }
        }
      }
      return { ok: true, dir: dir, scope: found.scope, files: 2 + stages.length }
    }
    // ── ⑧b 单包发布（D9 · §14.13：元数据形态对齐 toolset_export）──────
    //   本地能力：产出可分享的单 JSON 发布包（不推送外部市场——推送由
    //   市场插件/npm 通道负责）。发布包由 flow_import 消费（file/json）。
    function sha8(s) {
      var h = 5381
      var str = String(s)
      for (var i = 0; i < str.length; i++) h = ((h * 33) ^ str.charCodeAt(i)) >>> 0
      return ('00000000' + h.toString(16)).slice(-8)
    }
    function publishPathOf(id) { return P_EXPORT + '/' + safeName(id) + '.publish.json' }
    function publishPack(o) {
      var a = o || {}
      var id = String(a.id || '')
      var found = resolveFlow(id)
      if (!found) return { ok: false, error: '流程不存在: ' + id }
      if (a.scope) {
        var alt = flowAt(String(a.scope), id)
        if (!alt) return { ok: false, error: '在 ' + a.scope + ' 层找不到流程 ' + id }
        found = { def: alt, scope: String(a.scope), id: id }
      }
      var def = clone(found.def)
      def.schemaVersion = SCHEMA
      var v = validateFlow(def)
      if (!v.ok) return { ok: false, error: '流程校验失败（发布前必须通过）：' + v.errors.join('；') }
      var lib = def.library || {}
      var tags = []
      if (Array.isArray(a.tags)) tags = a.tags.map(String)
      else if (typeof a.tags === 'string' && a.tags) tags = a.tags.split(',').map(function (x) { return x.replace(/^\s+|\s+$/g, '') }).filter(function (x) { return !!x })
      if (!tags.length && Array.isArray(lib.tags)) tags = lib.tags.map(String)
      var author = String(a.author || lib.author || '')
      if (!author) return { ok: false, error: '缺少 author（发布必须署名：传 author，或在流程定义 library.author 中给出）' }
      var repo = String(a.repo || lib.repo || '')
      var version = Number(a.version || def.version || 1)
      if (!isNum(version) || version < 1) return { ok: false, error: 'version 必须是 ≥1 的数字' }
      var stages = def.stages || []
      var gateCount = 0
      for (var i = 0; i < stages.length; i++) gateCount += (stages[i].gates || []).length
      def.version = version
      def.library = { author: author, tags: tags, repo: repo }
      var pack = {
        kind: 'paircode.flow',                                     // 包类型（对齐工具集发布的显式 kind）
        spec: 'paircode-flow/1',
        schemaVersion: SCHEMA,
        name: def.id,
        title: def.title || def.id,
        version: version,
        author: author,
        tags: tags,
        repo: repo,
        publishedAt: nowIso(),
        source: { plugin: 'tool-flow', scope: found.scope },
        stats: { stages: stages.length, gates: gateCount },
        flow: def,
      }
      pack.checksum = sha8(JSON.stringify(pack.flow))
      var to = String(a.to || publishPathOf(id))
      if (!fsWriteJson(to, pack)) return { ok: false, error: '发布包写盘失败: ' + to }
      return {
        ok: true, path: to, pack: pack,
        manifest: {
          kind: pack.kind, spec: pack.spec, name: pack.name, title: pack.title, version: version,
          author: author, tags: tags, repo: repo, scope: found.scope,
          stages: stages.length, gates: gateCount, checksum: pack.checksum,
        },
      }
    }
    function dryRunText(r) {
      var rep = r.report || []
      var hits = rep.filter(function (x) { return x.exists })
      var lines = rep.map(function (x) {
        var s = '· ' + x.id + ' → ' + x.scope + ' 层 · 动作 ' + x.action + '\n    原因：' + x.reason
        if (x.exists) s += '\n    版本：现有 v' + x.currentVersion + ' ← 导入包 v' + x.incomingVersion
        if (x.author || (x.tags && x.tags.length)) s += '\n    元数据：' + (x.author || '（无作者）') + (x.tags && x.tags.length ? (' · 标签 ' + x.tags.join(', ')) : '')
        return s
      })
      return 'dryRun（未落盘）：' + (hits.length ? ('发现 ' + hits.length + ' 处冲突') : '无冲突（将新建）') + '\n' + lines.join('\n')
    }
    function importBundle(o) {
      var dir = String(o.dir || '')
      var file = String(o.file || '')
      var payload = null
      if (o.json) {
        try { payload = (typeof o.json === 'string') ? JSON.parse(o.json) : o.json } catch (e) { return { ok: false, error: 'json 解析失败: ' + e } }
      } else if (file) {
        var raw = fsRead(file)
        if (!raw) return { ok: false, error: '读取文件失败（不存在或不可读）：' + file }
        try { payload = JSON.parse(raw) } catch (e) { return { ok: false, error: '文件不是合法 JSON：' + file + '（' + e + '）' } }
      } else if (dir) {
        payload = fsReadJson(sep(dir) + 'flow.json')
        if (!payload && fsExists(sep(dir) + 'flow.json')) return { ok: false, error: '读取 ' + dir + '/flow.json 失败（内容不是合法 JSON）' }
        if (!payload) return { ok: false, error: '缺少 ' + dir + '/flow.json（目录 bundle 必须含 flow.json）' }
      } else return { ok: false, error: '需要 dir（目录 bundle）/ file（单文件）/ json（文本）之一' }
      // 发布包（D9 §14.13）：kind="paircode.flow" + flow 主体 + checksum
      var pubKind = String(payload.kind || '')
      var pubMeta = null
      if (pubKind === 'paircode.flow') {
        if (!payload.flow || typeof payload.flow !== 'object') return { ok: false, error: '发布包缺少 flow 主体（kind="paircode.flow" 必须含 flow）' }
        var wantCs = String(payload.checksum || '')
        if (wantCs) {
          var gotCs = sha8(JSON.stringify(payload.flow))
          if (wantCs !== gotCs) return { ok: false, error: '发布包校验和不符（文件被改动或损坏）：期望 ' + wantCs + '，实际 ' + gotCs }
        }
        pubMeta = {
          name: payload.name, title: payload.title, version: payload.version,
          author: payload.author, tags: payload.tags || [], repo: payload.repo || '',
          publishedAt: payload.publishedAt || '', stats: payload.stats || null,
        }
        payload = payload.flow
      } else if (pubKind && payload.flow && typeof payload.flow === 'object') {
        return { ok: false, error: '不支持的包 kind: ' + pubKind + '（本插件可消费 kind="paircode.flow" 的发布包，或裸流程定义 / 目录 bundle）' }
      } else if (payload.flow && typeof payload.flow === 'object') {
        payload = payload.flow                                          // 兼容无 kind 的单文件形态（§14.3）
      }
      var ver = payload.schemaVersion
      if (ver !== undefined && Number(ver) !== SCHEMA) return { ok: false, error: 'schemaVersion 不支持: ' + ver + '（本插件支持 ' + SCHEMA + '）——拒绝导入，不猜测降级' }
      var v = validateFlow(payload)
      if (!v.ok) return { ok: false, error: '流程校验失败：' + v.errors.join('；') }
      var scope = String(o.scope || 'project')
      if (['project', 'global'].indexOf(scope) < 0) return { ok: false, error: 'scope 必须是 project | global（preset 只读，不可写）' }
      var onConflict = String(o.onConflict || 'skip')
      if (['skip', 'overwrite', 'rename'].indexOf(onConflict) < 0) return { ok: false, error: 'onConflict 必须是 skip | overwrite | rename' }
      var id = String(payload.id)
      var cur0 = flowAt(scope, id)
      var exists = !!cur0
      var targetId = id
      var action = exists ? onConflict : 'create'
      if (exists && onConflict === 'rename') {
        targetId = id + '-imported-' + String(Date.now()).slice(-5)
        action = 'rename'
      }
      var report = [{
        id: id, targetId: targetId, scope: scope, action: action, exists: exists,
        packKind: pubKind || 'flow',
        currentVersion: cur0 ? (cur0.version || 1) : null,
        incomingVersion: Number(payload.version || 1),
        author: String((pubMeta && pubMeta.author) || (payload.library && payload.library.author) || ''),
        tags: (pubMeta && pubMeta.tags && pubMeta.tags.length) ? pubMeta.tags : ((payload.library && payload.library.tags) || []),
        reason: exists
          ? ('目标层已有同名流程（现 v' + (cur0.version || 1) + '）：' + (onConflict === 'skip' ? '默认 skip，不覆盖' : '按 ' + onConflict + ' 处理'))
          : '目标层无同名流程，将新建',
      }]
      if (o.dryRun === true) return { ok: true, dryRun: true, report: report, exists: exists, conflicts: exists ? report : [] }
      if (exists && onConflict === 'skip') return { ok: true, skipped: true, report: report, conflicts: report, text: '已存在同名流程（' + scope + ' 层）：' + id + '（现 v' + (cur0.version || 1) + '）→ 按 skip 跳过（未落盘）。需要覆盖用 onConflict:"overwrite"。' }
      var def = clone(payload)
      def.id = targetId
      def.schemaVersion = SCHEMA
      if (scope === 'project') {
        if (!fsWriteJson(flowPathAt('project', targetId), def)) return { ok: false, error: '写项目库失败' }
      } else {
        if (!libWrite(targetId, def)) return { ok: false, error: '写全局库失败（' + I_LIB + '）' }
      }
      return { ok: true, action: action, id: targetId, scope: scope, report: report, text: '已导入流程 ' + targetId + ' → ' + scope + ' 层' + (targetId !== id ? '（原 id ' + id + ' 冲突，已重命名）' : '') }
    }

    // ── ⑨ 设置（强度）────────────────────────────────────────────
    function settings() {
      try { var s = ctx.getSettings('tool-flow'); return (s && typeof s === 'object') ? s : {} } catch (e) { return {} }
    }
    function gateStrength() {
      var v = String(settings().gateStrength || 'enforce')
      return (v === 'warn' || v === 'off') ? v : 'enforce'
    }
    try {
      ctx.registerSettings({
        key: 'tool-flow',
        title: '工程流程与看板',
        fields: [
          { name: 'gateStrength', label: '门禁强度', type: 'select', options: ['enforce', 'warn', 'off'], default: 'enforce', group: '门禁', hint: 'enforce=拦截破坏性操作+每轮注入未过门禁；warn=只提示不拦截；off=只记录（不拦不注入，证据与看板照常留档）' },
          { name: 'autoRecommend', label: '未接入流程时自动推荐', type: 'boolean', default: true, group: '流程' },
          { name: 'trackModifiedFiles', label: '记录 run 期间改动文件（回滚建议用）', type: 'boolean', default: true, group: '证据' },
        ],
      })
    } catch (e) { logWarn('registerSettings 失败: ' + e) }

    // ── ⑩ 提示注入（agent 每轮感知 §14.6-①）──────────────────────
    function flowPromptText() {
      try {
        // 强度 off = 只记录（§10.1）：不主动注入流程提示；证据留档与看板展示不受影响。
        // warn / enforce 才注入（enforce 另在 PreToolUse 拦截破坏性操作）。
        if (gateStrength() === 'off') return ''
        if (settings().autoRecommend === false) return ctx._flowPromptSkip || ''
        var run = latestActiveRun()
        if (!run) {
          var rec = recommendFlow()
          return '[流程] 当前项目尚未接入流程。推荐：' + rec.flowId + '（' + rec.flowTitle + '，依据：' + rec.reason + '）。\n' +
            '可用 flow_start 启动它，flow_list 查看三层流程库，flow_attach 把流程绑定到项目。未接入不影响正常对话。'
        }
        var flow = runFlow(run)
        var sp = stageProgress(run)
        var idx = sp.idx
        var stageRun = run.stages[idx] || {}
        var stageDef = flow && flow.def.stages ? flow.def.stages[idx] : null
        var pending = stageDef ? pendingRequired(stageRun, stageDef) : []
        var s = '[流程] ' + run.flowId + (flow ? '「' + flow.def.title + '」' : '') + '（' + (run.flowScope || '?') + ' 级 v' + run.flowVersion + '）· 阶段 ' + sp.label
        if (stageDef && stageDef.goal) s += '\n阶段目标：' + stageDef.goal
        if (pending.length) {
          s += '\n未过门禁：' + pending.map(function (g) { return g.id + '（' + g.text + '，kind=' + g.kind + '）' }).join('；')
          s += '\n下一步：先把门禁做实（command 门禁可 flow_advance {gateId} 代跑并留证；check/artifact 需给出证据；approval 必须用户批准），再 flow_advance 推进阶段。'
        } else {
          s += '\n本阶段必过门禁已全部通过 → 可用 flow_advance 推进到下一阶段。'
        }
        return s
      } catch (e) { return '' }
    }
    try {
      ctx.systemPrompt.section({ name: 'flow', order: 95, text: '{{paircode_flow_state}}', alwaysVisible: true })
      ctx.systemPrompt.variable({ name: 'paircode_flow_state', provider: function () { return flowPromptText() } })
    } catch (e) { logWarn('systemPrompt 注册失败: ' + e) }

    // ── ⑪ 工具面（§14.5）────────────────────────────────────────
    var TOOLS = [
      {
        name: 'flow_list',
        description: '列出三层流程库（预设 preset 只读 / 全局 global / 项目 project；同名上层覆盖下层）。返回 id、标题、版本、层、阶段数、适用项目类型；另附当前项目类型识别与推荐。',
        parameters: {
          type: 'object',
          properties: {
            scope: { type: 'string', description: '可选：只看某一层（preset|global|project）' },
            type: { type: 'string', description: '可选：按适用项目类型过滤（如 go/web/python/plugin）' },
          },
        },
        execute: function (args) {
          var a = args || {}
          var list = allFlows()
          if (a.scope) list = list.filter(function (x) { return x.scope === a.scope })
          if (a.type) list = list.filter(function (x) { return (x.projectTypes || []).indexOf(a.type) >= 0 })
          var rec = recommendFlow()
          var lines = list.map(function (x) {
            return '· ' + x.id + '「' + x.title + '」v' + x.version + ' · ' + x.scope + (x.readOnly ? '(只读)' : '') + ' · ' + x.stages + ' 阶段 · 适用 ' + ((x.projectTypes || ['any']).join(',')) + (x.overriddenBy ? (' ←被 ' + x.overriddenBy + ' 同名覆盖') : '')
          })
          return {
            text: '流程库共 ' + list.length + ' 个：\n' + (lines.join('\n') || '（空）') +
              '\n\n项目类型识别：' + ((rec.types || []).join(', ') || '（无信号）') +
              '\n推荐流程：' + rec.flowId + '（' + rec.flowTitle + '）· 依据：' + rec.reason +
              '\n（预设层位于 <InstallDir>/.pair/plugins/tool-flow/assets/flows/，全局层 <InstallDir>/.pair/flowlib/，项目层 .pair/flows/）',
            flows: list, recommend: rec,
          }
        },
      },
      {
        name: 'flow_show',
        description: '查看某个流程的完整定义（阶段、任务、门禁、看板声明）+ 校验结果。默认按三层优先级解析；可用 scope 指定层。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '流程 id' },
            scope: { type: 'string', description: '可选：preset|global|project（缺省按优先级解析）' },
          },
          required: ['id'],
        },
        execute: function (args) {
          var a = args || {}
          var found = a.scope ? (function () { var d = flowAt(a.scope, a.id); return d ? { def: d, scope: a.scope, id: a.id } : null })() : resolveFlow(a.id)
          if (!found) return { text: '找不到流程：' + a.id + (a.scope ? ('（层 ' + a.scope + '）') : '') }
          var v = validateFlow(found.def)
          var def = found.def
          var lines = []
          lines.push(def.id + '「' + (def.title || '') + '」v' + (def.version || 1) + ' · 层 ' + found.scope + (found.scope === 'preset' ? '（只读）' : ''))
          if (def.library) lines.push('库元数据：' + JSON.stringify(def.library))
          lines.push('校验：' + (v.ok ? '[通过]' : '[未通过] ' + v.errors.join('；')))
          var stages = def.stages || []
          for (var i = 0; i < stages.length; i++) {
            var st = stages[i]
            lines.push('')
            lines.push((i + 1) + '. ' + (st.title || st.id) + '（' + st.id + '）' + (st.goal ? ' — ' + st.goal : ''))
            if (st.tasks && st.tasks.length) lines.push('   任务：' + st.tasks.join('；'))
            var gates = st.gates || []
            for (var j = 0; j < gates.length; j++) {
              var g = gates[j]
              lines.push('   门禁 ' + g.id + '（' + g.kind + (g.required ? '，必过' : '') + (g.blocking ? '，阻塞' : '') + (g.onFail ? '，onFail=' + g.onFail : '') + '）：' + (g.text || '') + (g.cmd ? '　cmd: ' + g.cmd : '') + (g.artifact ? '　artifact: ' + g.artifact : ''))
            }
          }
          if (def.board) lines.push('\n看板：' + JSON.stringify(def.board))
          return { text: lines.join('\n'), flow: def, scope: found.scope, validation: v }
        },
      },
      {
        name: 'flow_validate',
        description: '独立校验流程定义：阶段 id 唯一、门禁 kind 合法、command 门禁须有 cmd、artifact 须有 artifact、onFail 合法、board.kind 合法。可校验已有流程（id）或直接给 JSON（json）。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '可选：校验库中已有流程' },
            scope: { type: 'string', description: '可选：配合 id 指定层' },
            json: { type: 'string', description: '可选：直接校验一段流程 JSON 文本' },
          },
        },
        execute: function (args) {
          var a = args || {}
          var def = null
          if (a.json) { try { def = (typeof a.json === 'string') ? JSON.parse(a.json) : a.json } catch (e) { return { text: 'json 解析失败：' + e } } }
          else if (a.id) {
            var found = a.scope ? (function () { var d = flowAt(a.scope, a.id); return d ? { def: d } : null })() : resolveFlow(a.id)
            if (!found) return { text: '找不到流程：' + a.id }
            def = found.def
          } else return { text: '需要 id 或 json 之一' }
          var v = validateFlow(def)
          return { text: v.ok ? '[通过] 校验通过（' + ((def.stages || []).length) + ' 阶段）' : '[未通过] 校验未通过：\n' + v.errors.map(function (e) { return '  · ' + e }).join('\n'), validation: v }
        },
      },
      {
        name: 'flow_define',
        description: '创建或更新流程定义（项目层需 scope:"project"，全局层 scope:"global"；preset 只读不可写）。落盘前做完整 schema 校验；带 expectedVersion 时做乐观锁（版本不一致则拒绝并提示重拉）。',
        parameters: {
          type: 'object',
          properties: {
            json: { type: 'string', description: '流程定义 JSON 文本（FlowDef：id/title/version/stages[]/board/library…）' },
            scope: { type: 'string', description: 'project（工作区 .pair/flows/）或 global（<InstallDir>/.pair/flowlib/）' },
            expectedVersion: { type: 'number', description: '可选：乐观锁——当前版本号，不一致则拒绝' },
          },
          required: ['json', 'scope'],
        },
        execute: function (args) {
          var a = args || {}
          var scope = String(a.scope || '')
          if (['project', 'global'].indexOf(scope) < 0) return { text: 'scope 必须是 project | global（preset 层只读，不可写）' }
          var def = null
          try { def = (typeof a.json === 'string') ? JSON.parse(a.json) : a.json } catch (e) { return { text: 'JSON 解析失败：' + e } }
          var v = validateFlow(def)
          if (!v.ok) return { text: '[未通过] 校验失败，未落盘：\n' + v.errors.map(function (e) { return '  · ' + e }).join('\n'), validation: v }
          var cur = flowAt(scope, def.id)
          if (cur && a.expectedVersion !== undefined && Number(a.expectedVersion) !== Number(cur.version || 1)) {
            return { text: '版本冲突：当前 ' + scope + ' 层 ' + def.id + ' 为 v' + (cur.version || 1) + '，你给的是 v' + a.expectedVersion + ' → 拒绝写入（请重拉后再改）。', conflict: true, currentVersion: cur.version || 1 }
          }
          var existed = !!cur
          def.schemaVersion = SCHEMA
          if (scope === 'project') { if (!fsWriteJson(flowPathAt('project', def.id), def)) return { text: '写入失败：' + flowPathAt('project', def.id) } }
          else { if (!libWrite(def.id, def)) return { text: '写入全局库失败（' + I_LIB + '）' } }
          var path = scope === 'project' ? flowPathAt('project', def.id) : (I_LIB + '/' + safeName(def.id) + '.json')
          return { text: (existed ? '已更新' : '已创建') + '流程 ' + def.id + '（' + scope + ' 层 v' + (def.version || 1) + '）→ ' + path + '\n用 flow_start 启动，或 flow_attach 绑定到项目。' }
        },
      },
      {
        name: 'flow_clone',
        description: '以预设/全局/项目流程为模板，生成一份副本到目标层（「改自己的，不动上游」）。新 id 默认 <原id>-custom。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '源流程 id' },
            newId: { type: 'string', description: '可选：新 id（默认 <id>-custom）' },
            scope: { type: 'string', description: '目标层（默认 project）' },
            from: { type: 'string', description: '可选：源层（默认按优先级解析）' },
          },
          required: ['id'],
        },
        execute: function (args) {
          var a = args || {}
          var src = a.from ? (function () { var d = flowAt(a.from, a.id); return d ? { def: d, scope: a.from } : null })() : resolveFlow(a.id)
          if (!src) return { text: '找不到源流程：' + a.id }
          var scope = String(a.scope || 'project')
          if (['project', 'global'].indexOf(scope) < 0) return { text: 'scope 必须是 project | global' }
          var newId = String(a.newId || (a.id + '-custom'))
          if (flowAt(scope, newId)) return { text: scope + ' 层已存在同名流程：' + newId + '（换 newId 或用 flow_define 覆盖）' }
          var def = clone(src.def)
          def.id = newId
          def.title = (src.def.title || a.id) + '（自定义）'
          def.version = 1
          def.schemaVersion = SCHEMA
          if (scope === 'project') fsWriteJson(flowPathAt('project', newId), def)
          else libWrite(newId, def)
          return { text: '已从 ' + src.scope + ' 层 ' + a.id + ' 克隆为 ' + scope + ' 层 ' + newId + '（v1，可自由修改，不影响上游）' }
        },
      },
      {
        name: 'flow_delete',
        description: '删除某一层的流程定义（preset 只读不可删）。删除前检查是否被项目绑定或被运行实例引用，有引用则警告但仍执行（引用方会退化为「找不到流程」）。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '流程 id' },
            scope: { type: 'string', description: 'project | global（默认 project）' },
          },
          required: ['id'],
        },
        execute: function (args) {
          var a = args || {}
          var scope = String(a.scope || 'project')
          if (scope === 'preset') return { text: 'preset 层只读，不可删除（它是插件包内随包发布的模板）' }
          if (['project', 'global'].indexOf(scope) < 0) return { text: 'scope 必须是 project | global' }
          if (!flowAt(scope, a.id)) return { text: scope + ' 层不存在流程：' + a.id }
          var warns = []
          var b = loadBinding()
          var keys = Object.keys(b.bindings)
          for (var i = 0; i < keys.length; i++) if (b.bindings[keys[i]].flowId === a.id) warns.push('项目绑定引用：' + keys[i])
          var runs = listRuns().filter(function (r) { return r.flowId === a.id && r.status === 'active' })
          if (runs.length) warns.push('进行中的实例引用：' + runs.map(function (r) { return r.runId }).join(', '))
          var ok = false
          if (scope === 'project') { try { ctx.fs.rm(flowPathAt('project', a.id)); ok = true } catch (e) { ok = false } }
          else ok = libRemove(a.id)
          return { text: (ok ? '已删除 ' : '删除失败 ') + scope + ' 层流程 ' + a.id + (warns.length ? ('\n[注意] ' + warns.join('；')) : '') }
        },
      },
      {
        name: 'flow_sync',
        description: '显式把上游流程的新版本拉到下游层（preset→global/project，global→project）。默认不覆盖下游已有改动；force=true 才强制覆盖（升级不回冲原则）。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '流程 id（两侧同名）' },
            from: { type: 'string', description: '源层：preset|global' },
            to: { type: 'string', description: '目标层：global|project' },
            force: { type: 'boolean', description: '可选：目标已存在时是否强制覆盖（默认 false）' },
          },
          required: ['id', 'from', 'to'],
        },
        execute: function (args) {
          var a = args || {}
          var order = { preset: 0, global: 1, project: 2 }
          if (order[a.from] === undefined || order[a.to] === undefined || order[a.from] >= order[a.to]) return { text: 'from/to 非法：必须是上游→下游（preset→global→project）' }
          var src = flowAt(a.from, a.id)
          if (!src) return { text: a.from + ' 层没有流程：' + a.id }
          var dst = flowAt(a.to, a.id)
          if (dst && !a.force) {
            return { text: a.to + ' 层已存在 ' + a.id + '（v' + (dst.version || 1) + '），上游为 v' + (src.version || 1) + ' → 默认不回冲，未写入。需要覆盖用 force:true（会丢弃下游改动）。', skipped: true }
          }
          var def = clone(src)
          def.schemaVersion = SCHEMA
          if (a.to === 'project') fsWriteJson(flowPathAt('project', a.id), def)
          else libWrite(a.id, def)
          return { text: '已同步 ' + a.id + '：' + a.from + '（v' + (src.version || 1) + '）→ ' + a.to + (dst ? '（覆盖原 v' + (dst.version || 1) + '）' : '（新建）') }
        },
      },
      {
        name: 'flow_export',
        description: '把流程导出为目录 bundle（flow.json + README.md + gates/ + templates/），可 diff/评审/共享；默认导出到工作区 .pair/flow-exports/flow-<id>/。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '流程 id' },
            scope: { type: 'string', description: '可选：指定导出的层（默认按优先级）' },
            to: { type: 'string', description: '可选：目标目录（工作区相对路径）' },
          },
          required: ['id'],
        },
        execute: function (args) {
          var a = args || {}
          var r = exportBundle(a.id, a.scope, a.to)
          if (!r.ok) return { text: '导出失败：' + r.error }
          return { text: '已导出目录 bundle：' + r.dir + '（层 ' + r.scope + '）\n包含 flow.json / README.md' + ((r.files > 2) ? ' / gates/ / templates/' : '') + '\n导入：flow_import {dir:"' + r.dir + '", scope:"project"}' }
        },
      },
      {
        name: 'flow_publish',
        description: '把流程打成可分享的单 JSON 发布包（kind="paircode.flow"：author/tags/repo/version 元数据 + 流程主体 + checksum 完整性校验），元数据形态对齐工具集导出；产出可直接由 flow_import 回导。本地能力——不推送外部市场（推送走市场插件/npm 通道，见设计 §14.13）。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '流程 id' },
            scope: { type: 'string', description: '可选：指定导出的层（默认按优先级解析）' },
            to: { type: 'string', description: '可选：发布包输出路径（工作区相对，默认 .pair/flow-exports/<id>.publish.json）' },
            author: { type: 'string', description: '可选：作者署名（缺省取流程 library.author；都缺则拒绝——发布必须署名）' },
            tags: { type: 'string', description: '可选：标签（逗号分隔；缺省取流程 library.tags）' },
            repo: { type: 'string', description: '可选：来源仓库（如 github:owner/repo，供市场通道用）' },
            version: { type: 'number', description: '可选：发布版本号（缺省取流程定义 version）' },
          },
          required: ['id'],
        },
        execute: function (args) {
          var r = publishPack(args || {})
          if (!r.ok) return { text: '发布失败：' + r.error }
          var m = r.manifest
          return {
            text: '已生成发布包：' + r.path + '\nkind=' + m.kind + '（' + m.spec + '）· name=' + m.name + ' · v' + m.version +
              '\n作者：' + m.author + ' · 标签：' + ((m.tags && m.tags.length) ? m.tags.join(', ') : '（无）') + (m.repo ? (' · 仓库：' + m.repo) : '') +
              '\n规模：' + m.stages + ' 阶段 / ' + m.gates + ' 门禁 · 校验和 ' + m.checksum +
              '\n回导：flow_import {file:"' + r.path + '", scope:"project"}（先加 dryRun:true 看冲突）',
            manifest: m, path: r.path,
          }
        },
      },
      {
        name: 'flow_import',
        description: '导入流程到指定层：支持目录 bundle（dir）、单文件路径（file，含 flow_publish 产出的发布包）、或 JSON 文本（json）。发布包会校验 kind 与 checksum；schemaVersion 不符直接拒绝；onConflict 必须显式（默认 skip，绝不静默覆盖）；dryRun=true 只报告冲突与将执行的动作而不落盘。',
        parameters: {
          type: 'object',
          properties: {
            dir: { type: 'string', description: '目录 bundle 路径（含 flow.json）' },
            file: { type: 'string', description: '可选：单文件路径（发布包 *.publish.json 或裸流程 JSON）' },
            json: { type: 'string', description: '可选：JSON 文本（与 dir/file 三选一）' },
            scope: { type: 'string', description: '目标层：project（默认）| global' },
            onConflict: { type: 'string', description: 'skip（默认）| overwrite | rename' },
            dryRun: { type: 'boolean', description: '可选：只报告不落盘' },
          },
        },
        execute: function (args) {
          var a = args || {}
          var r = importBundle(a)
          if (!r.ok) return { text: '导入失败：' + r.error, report: r.report || null }
          if (r.dryRun) return { text: dryRunText(r), report: r.report, conflicts: r.conflicts || [] }
          if (r.skipped) return { text: r.text, report: r.report, conflicts: r.conflicts || [] }
          return { text: r.text, report: r.report }
        },
      },
      {
        name: 'flow_attach',
        description: '把流程绑定到项目（工作区某根）。绑定只影响后续新建实例，不会清空既有实例进度（随时可切换）。缺省 project 为当前主工作区根。',
        parameters: {
          type: 'object',
          properties: {
            project: { type: 'string', description: '可选：项目根绝对路径（默认当前主工作区根）' },
            flowId: { type: 'string', description: '要绑定的流程 id' },
            scope: { type: 'string', description: '可选：流程所在层（记录用，默认按优先级解析到的层）' },
            note: { type: 'string', description: '可选：备注' },
          },
          required: ['flowId'],
        },
        execute: function (args) {
          var a = args || {}
          var project = String(a.project || wsRoot() || '')
          if (!project) return { text: '拿不到项目根（工作区未打开？）' }
          var found = resolveFlow(a.flowId)
          if (!found) return { text: '流程不存在（三层库里没有）：' + a.flowId }
          var b = loadBinding()
          var old = b.bindings[project] || null
          b.bindings[project] = { flowId: a.flowId, scope: String(a.scope || found.scope), since: nowIso(), note: String(a.note || '') }
          saveBinding(b)
          return { text: '已绑定：' + project + ' → ' + a.flowId + '（' + b.bindings[project].scope + ' 层）' + (old ? ('\n（原绑定 ' + old.flowId + ' 已替换；既有实例进度不受影响）') : '') }
        },
      },
      {
        name: 'flow_detach',
        description: '解除项目与流程的绑定（既有实例进度保留）。',
        parameters: {
          type: 'object',
          properties: { project: { type: 'string', description: '可选：项目根（默认当前主工作区根）' } },
        },
        execute: function (args) {
          var a = args || {}
          var project = String(a.project || wsRoot() || '')
          var b = loadBinding()
          if (!b.bindings[project]) return { text: '该项目未绑定流程：' + project }
          var old = b.bindings[project].flowId
          delete b.bindings[project]
          saveBinding(b)
          return { text: '已解除绑定：' + project + '（原 ' + old + '）' }
        },
      },
      {
        name: 'flow_bindings',
        description: '查看「项目 × 流程」矩阵：每个项目根的绑定流程、进行中实例、当前阶段与未过门禁数（数据源：.pair/flows/_binding.json + 各 run）。',
        parameters: { type: 'object', properties: {} },
        execute: function () {
          var m = bindingsMatrix()
          if (!m.length) return { text: '没有可用项目根。' }
          var lines = m.map(function (x) {
            return '· ' + x.project + ' → ' + (x.flowId || '（未绑定流程）') + (x.scope ? ('（' + x.scope + ' 层）') : '') +
              (x.runId ? (' · 实例 ' + x.runId + ' · 阶段 ' + x.stage + (x.blocked ? (' · ⛔ ' + x.blocked + ' 门禁阻塞') : '')) : ' · 无进行中实例')
          })
          return { text: '项目 × 流程矩阵（' + m.length + ' 项）：\n' + lines.join('\n'), matrix: m }
        },
      },
      {
        name: 'flow_recommend',
        description: '按项目特征文件（go.mod/package.json/…）识别项目类型并推荐预设流程；返回推荐 id、依据、以及是否已在三层库中。',
        parameters: {
          type: 'object',
          properties: { project: { type: 'string', description: '可选：项目根（当前实现按主工作区根识别）' } },
        },
        execute: function () {
          var rec = recommendFlow()
          return { text: '项目类型：' + ((rec.types || []).join(', ') || '（未识别到信号）') + '\n推荐流程：' + rec.flowId + '（' + rec.flowTitle + '）\n依据：' + rec.reason + (rec.available ? '' : '\n[注意] 推荐的流程不在三层库中，可用 flow_import 导入或 flow_define 创建。'), recommend: rec }
        },
      },
      {
        name: 'flow_start',
        description: '启动一个流程实例（按会话/项目绑定）。解析顺序：显式 flowId → 项目绑定 → 项目类型推荐。启动会记录 git HEAD 基线、为第一阶段建任务（写 .pair/tasks，出现在右栏任务面板）。',
        parameters: {
          type: 'object',
          properties: {
            flowId: { type: 'string', description: '可选：流程 id（缺省按绑定/推荐）' },
            convId: { type: 'string', description: '可选：绑定会话（多会话并行时必填，缺省为项目级实例）' },
            project: { type: 'string', description: '可选：项目根（默认当前主工作区根）' },
            force: { type: 'boolean', description: '可选：已有进行中实例时仍另起一个（不推荐）' },
          },
        },
        execute: function (args) {
          var a = args || {}
          var r = startRun(a)
          if (!r.ok) return { text: r.error + (r.hint ? ('\n' + r.hint) : ''), run: r.activeRun || null }
          var run = r.run
          var sp = stageProgress(run)
          return {
            text: '流程已启动：' + run.flowId + '（' + r.flowScope + ' 层 · 来源 ' + r.source + '）\n实例：' + run.runId + '\n阶段 ' + sp.label +
              '\n已建任务 ' + ((run.stages[0].taskIds || []).length) + ' 条（右栏任务面板可见）' +
              (run.gitHead ? '\ngit 基线：' + run.gitHead : '\n（未取到 git 基线：非 git 仓库，回滚只能人工处理）') +
              '\n用 flow_status 看全貌，flow_advance 过门禁/推进。',
            run: run,
          }
        },
      },
      {
        name: 'flow_status',
        description: '查看流程实例进度：各阶段状态、门禁通过情况（PASS/PEND/FAIL/----）、改动文件、git 基线；缺省取最近活跃实例（多会话请传 convId 或 runId）。',
        parameters: {
          type: 'object',
          properties: {
            convId: { type: 'string', description: '可选：按会话查' },
            runId: { type: 'string', description: '可选：按实例 id 查' },
            all: { type: 'boolean', description: '可选：列出全部实例（含已结束）' },
          },
        },
        execute: function (args) {
          var a = args || {}
          if (a.all) {
            var all = listRuns()
            if (!all.length) return { text: '没有任何流程实例（flow_start 启动）。' }
            var lines = all.map(function (r) {
              var sp = stageProgress(r)
              return '· ' + r.runId + ' · ' + r.flowId + ' · ' + r.status + ' · 阶段 ' + sp.label + (r.convId ? (' · ' + r.convId) : '') + ' · 更新 ' + r.updatedAt
            })
            return { text: '流程实例（' + all.length + '）：\n' + lines.join('\n'), runs: all }
          }
          var run = pickRun(a)
          return { text: statusText(run), run: run || null }
        },
      },
      {
        name: 'flow_advance',
        description: '流程推进的单一入口：① 带 gateId → 执行/记录该门禁（command 自动代跑并存证；check 需 evidence 说明；artifact 校验产物存在；approval 必须 approve:true 由用户批准）；② 不带 gateId → 检查当前阶段全部必过门禁，通过则推进到下一阶段。force:true 可跳过未过门禁（需谨慎，会记录在证据里）。',
        parameters: {
          type: 'object',
          properties: {
            gateId: { type: 'string', description: '可选：要执行/记录的门禁 id' },
            evidence: { type: 'string', description: '可选：证据说明（check 门禁必填；command 门禁可附注）' },
            approve: { type: 'boolean', description: '可选：approval 门禁的批准动作（必须由用户明确同意后才能传 true）' },
            note: { type: 'string', description: '可选：审批附言' },
            force: { type: 'boolean', description: '可选：跳过未过门禁直接推进（谨慎）' },
            runId: { type: 'string', description: '可选：指定实例' },
            convId: { type: 'string', description: '可选：按会话定位实例' },
          },
        },
        execute: function (args) {
          var r = advanceRun(args || {})
          if (!r.ok) return { text: r.error, run: r.run || null }
          var extra = ''
          if (r.rollback) extra = '\n\n[注意] 回滚建议（插件不自动执行，高风险命令必须用户批准）：\n' + r.rollback.rollbackOptions.map(function (o) { return '  · [' + o.risk + '] ' + o.text + ' → ' + o.cmd }).join('\n') + (r.rollback.note ? ('\n  ' + r.rollback.note) : '')
          return { text: r.text + extra, result: r }
        },
      },
    ]
    for (var ti = 0; ti < TOOLS.length; ti++) {
      (function (t) {
        try {
          ctx.tools.register({
            name: t.name,
            description: t.description,
            parameters: t.parameters,
            execute: function (args) { return t.execute(args || {}) },
          })
        } catch (e) { logWarn('工具注册失败 ' + t.name + ': ' + e) }
      })(TOOLS[ti])
    }

    // ── ⑫ 只读接口（给看板）─────────────────────────────────────
    function stateSnapshot(q) {
      var convId = String(q.convId || '')
      var run = pickRun({ convId: convId })
      var flow = run ? runFlow(run) : null
      var stagesOut = []
      if (run) {
        for (var i = 0; i < (run.stages || []).length; i++) {
          var st = run.stages[i]
          var def = flow && flow.def.stages ? flow.def.stages[i] : null
          var gates = ((def && def.gates) || []).map(function (g) {
            var r = gateResult(st, g.id)
            return {
              id: g.id, kind: g.kind, text: g.text || '', required: !!g.required, blocking: !!g.blocking,
              onFail: g.onFail || '', cmd: g.cmd || '', artifact: g.artifact || '',
              status: r ? r.status : 'todo', summary: r ? (r.summary || '') : '', at: r ? r.at : '',
              evidenceFile: r ? (r.evidenceFile || '') : '', by: r ? (r.by || '') : '', note: r ? (r.note || '') : '',
            }
          })
          stagesOut.push({
            id: st.id, title: def ? def.title : st.id, goal: def ? (def.goal || '') : '',
            tasks: def ? (def.tasks || []) : [], status: st.status,
            startedAt: st.startedAt || '', completedAt: st.completedAt || '',
            taskIds: st.taskIds || [], gates: gates,
          })
        }
      }
      var sp = run ? stageProgress(run) : null
      return {
        ok: true, at: nowIso(),
        convId: convId,
        run: run ? {
          runId: run.runId, flowId: run.flowId, flowScope: run.flowScope, flowVersion: run.flowVersion,
          title: flow ? (flow.def.title || run.flowId) : run.flowId,
          status: run.status, currentStage: run.currentStage, project: run.project,
          gitHead: run.gitHead || '', modifiedFiles: run.modifiedFiles || [],
          evidenceDir: run.evidenceDir || '', stageProgress: sp,
        } : null,
        stages: stagesOut,
        runMap: (function () {
          var m = {}
          var runs = listRuns()
          for (var i = 0; i < runs.length; i++) {
            var r = runs[i]
            var x = stageProgress(r)
            m[r.runId] = { runId: r.runId, flowId: r.flowId, status: r.status, stage: x.label, pending: x.pending }
          }
          return m
        })(),
        library: allFlows(),
        matrix: bindingsMatrix(),
        recommend: (settings().autoRecommend === false) ? null : recommendFlow(),
        settings: { gateStrength: gateStrength() },
      }
    }
    try {
      ctx.http.register('GET', '/api/flow/state', function (req) {
        var q = parseQuery(req && req.query)
        var body = ''
        try { body = JSON.stringify(stateSnapshot(q)) }
        catch (e) { return { status: 500, body: JSON.stringify({ ok: false, error: '状态快照失败: ' + e }), headers: { 'Content-Type': 'application/json' } } }
        return { status: 200, body: body, headers: { 'Content-Type': 'application/json; charset=utf-8' } }
      })
    } catch (e) { logWarn('接口注册失败 /api/flow/state: ' + e) }

    // ── ⑬ 写 RPC（看板按钮 → host 半）───────────────────────────
    try {
      ctx.registerClientMethod('flow_action', function (args) {
        var a = (args && typeof args === 'object') ? args : {}
        var op = String(a.op || '')
        if (op === 'advance') {
          var r = advanceRun(a)
          if (!r.ok) return { ok: false, error: r.error }
          return { ok: true, result: { advanced: !!r.advanced, completed: !!r.completed, stage: r.stage || '', text: r.text || '', gate: r.gate || null, pending: r.pending || null, rollback: r.rollback || null } }
        }
        if (op === 'approve') {
          if (a.approve !== true) return { ok: false, error: 'approve 必须为 true（审批卡点由用户在界面显式批准）' }
          var r2 = advanceRun({ runId: a.runId, convId: a.convId, gateId: a.gateId, approve: true, note: a.note })
          if (!r2.ok) return { ok: false, error: r2.error }
          return { ok: true, result: { gate: r2.gate || null, advanced: !!r2.advanced, text: r2.text || '' } }
        }
        if (op === 'start') {
          var r3 = startRun({ flowId: a.flowId, convId: a.convId, project: a.project, force: !!a.force })
          if (!r3.ok) return { ok: false, error: r3.error }
          return { ok: true, result: { runId: r3.run.runId, flowId: r3.run.flowId, text: '流程已启动：' + r3.run.flowId } }
        }
        if (op === 'show') {
          var f0 = resolveFlow(String(a.id || ''))
          if (!f0) return { ok: false, error: '流程不存在: ' + a.id }
          return { ok: true, result: { id: f0.id, scope: f0.scope, version: f0.def.version || 1, readOnly: f0.scope === 'preset', def: f0.def } }
        }
        if (op === 'validate') {
          var dv = null
          try { dv = (typeof a.json === 'string') ? JSON.parse(a.json) : a.json } catch (e) { return { ok: false, error: 'JSON 解析失败：' + e } }
          return { ok: true, result: validateFlow(dv) }
        }
        if (op === 'define') {
          var scope2 = String(a.scope || 'project')
          if (['project', 'global'].indexOf(scope2) < 0) return { ok: false, error: 'scope 必须是 project | global（preset 只读，不可写）' }
          var dd = null
          try { dd = (typeof a.json === 'string') ? JSON.parse(a.json) : a.json } catch (e) { return { ok: false, error: 'JSON 解析失败：' + e } }
          var vv = validateFlow(dd)
          if (!vv.ok) return { ok: false, error: '校验未通过：' + vv.errors.join('；') }
          var cur0 = flowAt(scope2, dd.id)
          if (cur0 && a.expectedVersion !== undefined && a.expectedVersion !== null && Number(a.expectedVersion) !== Number(cur0.version || 1)) {
            return { ok: false, error: '版本冲突：当前 v' + (cur0.version || 1) + '，提交基于 v' + a.expectedVersion + '（请重新加载后再改）' }
          }
          dd.schemaVersion = SCHEMA
          if (scope2 === 'project') { if (!fsWriteJson(flowPathAt('project', dd.id), dd)) return { ok: false, error: '写项目库失败' } }
          else if (!libWrite(dd.id, dd)) return { ok: false, error: '写全局库失败（' + I_LIB + '）' }
          return { ok: true, result: { id: dd.id, scope: scope2, version: dd.version || 1, text: '已保存流程 ' + dd.id + '（' + scope2 + ' 层 v' + (dd.version || 1) + '）' } }
        }
        if (op === 'clone') {
          var src0 = resolveFlow(String(a.id || ''))
          if (!src0) return { ok: false, error: '源流程不存在: ' + a.id }
          var target0 = String(a.scope || 'project')
          if (['project', 'global'].indexOf(target0) < 0) return { ok: false, error: 'scope 必须是 project | global' }
          var newId0 = String(a.newId || (a.id + '-custom'))
          if (flowAt(target0, newId0)) return { ok: false, error: target0 + ' 层已存在：' + newId0 }
          var dc = clone(src0.def)
          dc.id = newId0
          dc.title = (src0.def.title || a.id) + '（自定义）'
          dc.version = 1
          dc.schemaVersion = SCHEMA
          if (target0 === 'project') fsWriteJson(flowPathAt('project', newId0), dc)
          else libWrite(newId0, dc)
          return { ok: true, result: { id: newId0, scope: target0, text: '已从 ' + src0.scope + ' 层克隆为 ' + target0 + ' 层 ' + newId0 + '（可自由编辑，不影响上游）' } }
        }
        if (op === 'attach') {
          var found = resolveFlow(String(a.flowId || ''))
          if (!found) return { ok: false, error: '流程不存在: ' + a.flowId }
          var project = String(a.project || wsRoot() || '')
          var b = loadBinding()
          b.bindings[project] = { flowId: a.flowId, scope: found.scope, since: nowIso(), note: String(a.note || '') }
          saveBinding(b)
          return { ok: true, result: { project: project, flowId: a.flowId, text: '已绑定 ' + project + ' → ' + a.flowId } }
        }
        if (op === 'detach') {
          var project2 = String(a.project || wsRoot() || '')
          var b2 = loadBinding()
          delete b2.bindings[project2]
          saveBinding(b2)
          return { ok: true, result: { project: project2, text: '已解绑 ' + project2 } }
        }
        // D9（§14.13）：发布包与回导也走同一 RPC 通道（看板/外部一致）
        if (op === 'publish') {
          var rp = publishPack(a)
          if (!rp.ok) return { ok: false, error: rp.error }
          return {
            ok: true,
            result: {
              path: rp.path, manifest: rp.manifest,
              text: '已生成发布包：' + rp.path + '（kind=' + rp.manifest.kind + ' · v' + rp.manifest.version +
                ' · ' + rp.manifest.stages + ' 阶段 / ' + rp.manifest.gates + ' 门禁）',
            },
          }
        }
        if (op === 'import') {
          var ri = importBundle(a)
          if (!ri.ok) return { ok: false, error: ri.error }
          if (ri.dryRun) return { ok: true, result: { dryRun: true, report: ri.report, conflicts: ri.conflicts || [], text: dryRunText(ri) } }
          return { ok: true, result: { report: ri.report, conflicts: ri.conflicts || [], skipped: !!ri.skipped, text: ri.text || '' } }
        }
        return { ok: false, error: '未知 op（支持 advance | approve | start | attach | detach | show | validate | define | clone | publish | import）' }
      })
    } catch (e) { logWarn('RPC 注册失败 flow_action: ' + e) }

    // ── ⑭ 门禁钩子（§10.1：PreToolUse 拦破坏性操作 + PostToolUse 记改动）──
    function isRiskyTool(name, args) {
      var n = String(name || '')
      var blob = ''
      try { blob = JSON.stringify(args || {}) } catch (e) { blob = '' }
      if (n === 'bash' || n === 'exec_command') {
        if (/git\s+(reset\s+--hard|checkout\s+--|clean\s+-[a-z]*f)/.test(blob)) return 'git 破坏性操作'
        if (/rm\s+-[a-z]*r[a-z]*f|rm\s+-[a-z]*f[a-z]*r/.test(blob)) return 'rm -rf 递归删除'
      }
      if (/^(delete|rm_file|remove_file)$/.test(n)) return '文件删除'
      return ''
    }
    try {
      ctx.hooks.register('PreToolUse', function (payload) {
        try {
          if (gateStrength() !== 'enforce') return null
          var p = payload || {}
          var run = pickRun({})
          if (!run || run.status !== 'active') return null
          var flow = runFlow(run)
          if (!flow) return null
          var idx = stageIdx(run)
          var stageDef = flow.def.stages[idx]
          var stageRun = run.stages[idx]
          if (!stageDef || !stageRun) return null
          var risk = isRiskyTool(p.toolName, p.toolArgs)
          if (!risk) return null
          // 拦截条件 = 本阶段存在未过的**必过**门禁（required）；blocking 只是加重标记。
          // 只认 blocking 会让「必过但未标 blocking」的门禁形同虚设（D1「重」强度要求真拦截）。
          var notPassed = pendingRequired(stageRun, stageDef)
          if (!notPassed.length) return null
          var blocking = notPassed.filter(function (g) { return !!g.blocking })
          return {
            block: true,
            feedback: '【流程门禁】阶段「' + stageDef.title + '」还有 ' + notPassed.length + ' 个未过的必过门禁' +
              (blocking.length ? ('（其中阻塞门禁 ' + blocking.length + ' 个）') : '') + '：' +
              notPassed.map(function (g) { return g.id + '（' + (g.text || '') + (g.blocking ? '，阻塞' : '') + '）' }).join('；') +
              '。检测到破坏性操作（' + risk + '）——请先让门禁通过（flow_advance），或由用户明确批准该操作后再执行。',
          }
        } catch (e) { return null }
      })
      ctx.hooks.register('PostToolUse', function (payload) {
        try {
          if (settings().trackModifiedFiles === false) return null
          var p = payload || {}
          var n = String(p.toolName || '')
          if (['write', 'edit', 'apply_patch', 'multi_edit', 'str_replace'].indexOf(n) < 0) return null
          var run = pickRun({})
          if (!run || run.status !== 'active') return null
          var f = ''
          var a = p.toolArgs || {}
          var keys = ['file_path', 'path', 'filename', 'file']
          for (var i = 0; i < keys.length; i++) if (a && typeof a[keys[i]] === 'string') { f = a[keys[i]]; break }
          if (!f && typeof p.toolArgs === 'string') {
            var m = String(p.toolArgs).match(/"path"\s*:\s*"([^"]+)"/)
            if (m) f = m[1]
          }
          if (!f) return null
          if (!run.modifiedFiles) run.modifiedFiles = []
          if (run.modifiedFiles.indexOf(f) < 0) {
            run.modifiedFiles.push(f)
            saveRun(run)
          }
        } catch (e) { }
        return null
      })
    } catch (e) { logWarn('hooks 注册失败: ' + e) }

    // ── ⑮ 命令面（§14.6-②）──────────────────────────────────────
    try {
      ctx.commands.register({
        name: 'flow',
        description: '工程流程：查看进度 / 列出流程库 / 推进阶段（用法：/flow [status|list|advance|bind <flowId>]）',
        handler: function (argsStr) {
          var argv = String(argsStr || '').trim().split(/\s+/).filter(function (x) { return !!x })
          var sub = argv[0] || 'status'
          if (sub === 'list') {
            var list = allFlows()
            return '流程库（' + list.length + '）:\n' + list.map(function (x) { return '· ' + x.id + '「' + x.title + '」v' + x.version + ' · ' + x.scope + (x.readOnly ? '(只读)' : ''); }).join('\n')
          }
          if (sub === 'advance') {
            var r = advanceRun({ force: argv.indexOf('--force') >= 0 })
            if (!r.ok) return r.error
            return r.text
          }
          if (sub === 'bind') {
            var fid = argv[1]
            if (!fid) return '用法：/flow bind <flowId>'
            var found = resolveFlow(fid)
            if (!found) return '流程不存在：' + fid
            var b = loadBinding()
            var project = wsRoot()
            b.bindings[project] = { flowId: fid, scope: found.scope, since: nowIso(), note: '' }
            saveBinding(b)
            return '已绑定 ' + project + ' → ' + fid
          }
          return statusText(pickRun({}))
        },
      })
      ctx.commands.register({
        name: 'flow-board',
        description: '打开流程看板（主内容区「流程」tab；无实例时显示流程库与接入向导）',
        handler: function () {
          try {
            ctx.emit('client:openView', { id: 'flow-board' })
          } catch (e) { }
          var run = pickRun({})
          return (run ? ('流程看板已打开（实例 ' + run.runId + '）。\n' + statusText(run)) : '流程看板已打开。当前没有流程实例：可在看板「流程库」页签选一个流程并启动。')
        },
      })
      ctx.commands.register({
        name: 'flow-import',
        description: '导入流程包或目录 bundle（用法：/flow-import <目录|发布包.json> [project|global]）',
        handler: function (argsStr) {
          var argv = String(argsStr || '').trim().split(/\s+/).filter(function (x) { return !!x })
          if (!argv.length) return '用法：/flow-import <目录 bundle 路径|发布包 .json> [project|global]'
          var src = argv[0]
          var opts = { scope: argv[1] || 'project', onConflict: 'skip' }
          if (/\.json$/i.test(src)) opts.file = src
          else opts.dir = src
          var r = importBundle(opts)
          if (!r.ok) return '导入失败：' + r.error
          return r.text
        },
      })
    } catch (e) { logWarn('commands 注册失败: ' + e) }

    // ── ⑯ 装载摘要（诊断用）─────────────────────────────────────
    logInfo('装载完成：预设 ' + presetIds().length + ' · 全局 ' + libIds().length + ' · 项目 ' + projectIds().length +
      ' · 工具 ' + TOOLS.length + ' · 实例 ' + listRuns().length + ' · 强度 ' + gateStrength())
  },
}
