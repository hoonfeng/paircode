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
  inject: ['fs', 'install', 'logger', 'commands', 'http', 'bash', 'agents', 'timer'],
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
    // ★ 2026-10-17 使用面收窄（用户决策）：流程只在「任务看板」里使用。
    //   会话侧（agent 工具面 flow_* / 斜杠命令 /flow* / 每轮提示注入 / 门禁钩子）全部关闭；
    //   看板侧（flow-board 视图 + flow_action / taskboard_action RPC + taskboard_* 工具）保持不变。
    //   恢复会话侧能力只需把本开关改成 true（代码保留不删，便于回滚与将来按场景开启）。
    var CONVERSATION_FACING = false
    var BOARD_KINDS = ['kanban', 'checklist', 'timeline', 'swimlane']
    var SCOPES = ['project', 'global', 'preset']                 // 解析优先级
    // 子任务状态（阶段内可细化的最小执行单元；cancelled 不计入进度口径）
    var SUB_STATUS = ['pending', 'in_progress', 'completed', 'blocked', 'cancelled']
    // 实例生命周期：active 进行中 / completed 已完成 / interrupted 已中断（可 resume）
    //              archived 已归档（只读留档，移出活跃列表）/ deleted 已删除（软删可恢复）
    var RUN_STATUS = ['active', 'completed', 'interrupted', 'archived', 'deleted']

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

    // ── ②b 任务看板数据层（照 github.com/cloader/dsh-taskboard 重建，2026-10-17）──
    // 照参考：任务归**项目（工作区）**、不归会话（会话只出现在 claimedBy＝认领执行者）；
    //         七态严格状态机；人建卡 → agent 认领 → 交报告 → **人验收**（agent 不能自己 done）。
    // 结合流程（本 IDE 增量）：任务可绑定一个流程模板作**执行骨架** ——
    //         阶段进度显示在卡片/详情；流程门禁 → 任务验收清单（DoD）；流程走完 → 任务转「待验收」。
    // 落点：<ws>/.pair/taskboard/tasks.json（单文件 ledger + revision，照参考）
    // 镜像：每条任务同步 <ws>/.pair/tasks/flowtb_<id>.json → 右栏「任务进度」面板可见
    var P_TB = '.pair/taskboard/tasks.json'
    // ── 多实例共存（9090/9096 等共享同一工作区）的隔离与互斥 ──
    // 同一份 tasks.json 被两个进程各自 60s 扫描会重复派发 + last-write-wins 覆盖写。
    // 方案：① 定时扫描文件锁（90s 心跳）——同一时刻只有一个实例扫 schedule；
    //       ② 到期任务「先推进 nextAt 再执行」（CAS 重读 revision），抢到的实例才执行；
    //       ③ 最近活跃会话池——定时执行没有显式会话时的兜底投递目标。
    var P_TB_CONV = '.pair/taskboard/.last-conv.json'
    var P_TB_LOCK = '.pair/taskboard/.scheduler.lock'
    // 驱动模式（手动 / 全流程自动）：工作区级配置，9090/9096 多实例共享。
    // 独立文件而不写 tasks.json —— 避免与任务写入的 revision CAS 互相踩。
    var P_TB_DRIVER = '.pair/taskboard/driver.json'
    var TB_INSTANCE = 'tb-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2, 8)
    var TB_LOCK_TTL_MS = 90000
    var TB_MAIN = ['backlog', 'todo', 'in_progress', 'in_review', 'done']
    var TB_SECONDARY = ['canceled', 'archived']
    var TB_STATUS = TB_MAIN.concat(TB_SECONDARY)
    var TB_LABEL = { backlog: '待规划', todo: '待办', in_progress: '进行中', in_review: '待验收', done: '已完成', canceled: '已取消', archived: '已归档' }
    var TB_URGENCY = ['urgent', 'normal', 'relaxed']
    // 合法迁移（照参考 TRANSITIONS 原样）；archived 为终态
    var TB_TRANS = {
      backlog: ['todo', 'canceled'],
      todo: ['in_progress', 'backlog', 'canceled'],
      in_progress: ['in_review', 'todo', 'canceled'],
      in_review: ['in_progress', 'todo', 'done', 'canceled'],
      done: ['archived', 'todo'],
      canceled: ['archived', 'todo'],
      archived: []
    }
    function tbIsClaim(from, to) { return from === 'todo' && to === 'in_progress' }
    function tbCanTransition(from, to) {
      var t = TB_TRANS[from]
      return !!(t && t.indexOf(to) >= 0)
    }
    function tbLabel(s) { return TB_LABEL[s] || String(s == null ? '' : s) }
    function tbActor(by) {
      if (by && typeof by === 'object') {
        return { kind: String(by.kind || 'user'), sessionId: String(by.sessionId || ''), name: String(by.name || '') }
      }
      return { kind: by === 'agent' ? 'agent' : 'user', sessionId: '', name: '' }
    }
    function tbWorkspaceId() {
      var r = String(wsRoot() || '').replace(/[\\/]+$/, '')
      var i = Math.max(r.lastIndexOf('/'), r.lastIndexOf('\\'))
      return i < 0 ? (r || 'workspace') : r.slice(i + 1)
    }
    /**
     * 最近活跃会话池：任何带 convId 的看板 RPC（建卡/认领/执行/设定时…）都会记一笔。
     * 定时执行没有显式会话时，按「最近活跃会话」投递（ctx.agents.followup），
     * 这样「新建卡 → 设定时 → 到点自动跑」不需要先人工点一次「执行」。
     */
    function tbRememberConv(convId) {
      var c = String(convId || '').trim()
      if (!c) return
      var cur = fsReadJson(P_TB_CONV) || {}
      var list = Array.isArray(cur.convs) ? cur.convs : []
      var out = []
      for (var i = 0; i < list.length; i++) {
        var it = list[i] || {}
        if (String(it.id || '') === c) continue
        if (it.id) out.push({ id: String(it.id), at: String(it.at || '') })
      }
      out.unshift({ id: c, at: nowIso() })
      if (out.length > 20) out = out.slice(0, 20)
      fsWriteJson(P_TB_CONV, { version: 1, convs: out, updatedAt: nowIso() })
    }
    function tbLastConv() {
      var cur = fsReadJson(P_TB_CONV) || {}
      var list = Array.isArray(cur.convs) ? cur.convs : []
      for (var i = 0; i < list.length; i++) {
        var c = String((list[i] && list[i].id) || '')
        if (c) return c
      }
      return ''
    }
    /**
     * 定时扫描实例锁（近似互斥）：写自己的 holder → 立刻读回确认。
     * 90s 心跳窗口内其它实例看到「活锁」就跳过本轮扫描；过期锁可被抢占。
     * 说明：goja 沙箱没有 O_EXCL，读回校验存在极小竞态；因此调度侧还有
     *       「先推进 nextAt（CAS 重读 revision）再执行」的第二道防线。
     */
    function tbLockAcquire() {
      var now = Date.now()
      var cur = fsReadJson(P_TB_LOCK) || null
      if (cur && cur.holder && String(cur.holder) !== TB_INSTANCE) {
        var at = new Date(String(cur.at || '')).getTime()
        if (isFinite(at) && now - at < TB_LOCK_TTL_MS) return false
      }
      fsWriteJson(P_TB_LOCK, { holder: TB_INSTANCE, at: nowIso(), note: 'taskboard schedule scan lock' })
      var back = fsReadJson(P_TB_LOCK) || null
      return !!(back && String(back.holder) === TB_INSTANCE)
    }
    function tbLoad() {
      var l = fsReadJson(P_TB)
      if (!l || typeof l !== 'object' || !Array.isArray(l.tasks)) return { version: 1, revision: 0, tasks: [] }
      if (!isNum(l.revision)) l.revision = 0
      return l
    }
    function tbSave(l) {
      l.version = 1
      l.revision = (isNum(l.revision) ? l.revision : 0) + 1
      return fsWriteJson(P_TB, l)
    }
    function tbNewId() { return 't-' + Date.now() + '-' + Math.random().toString(36).slice(2, 6) }
    function tbFind(l, id) {
      for (var i = 0; i < l.tasks.length; i++) if (l.tasks[i].id === id) return l.tasks[i]
      return null
    }
    function tbMirrorName(id) { return 'flowtb_' + String(id).replace(/[^A-Za-z0-9]/g, '_') }
    /** 宿主镜像（右栏任务面板读 .pair/tasks/<id>.json，字段沿用宿主契约 §5.3）。 */
    function tbMirror(t) {
      if (!t) return
      var st = t.status === 'done' ? 'completed'
        : (t.status === 'in_progress' ? 'in_progress'
          : ((t.status === 'canceled' || t.status === 'archived') ? 'cancelled' : 'pending'))
      fsWriteJson(P_TASKS + '/' + tbMirrorName(t.id) + '.json', {
        id: tbMirrorName(t.id), subject: String(t.title || ''), description: String(t.description || ''),
        status: st, dependencies: [], convId: String(t.claimedBy || ''),
        created_at: t.createdAt || '', updated_at: t.updatedAt || '', boardTaskId: t.id
      })
    }
    /** 流程门禁 → 任务验收清单（DoD）：每个门禁一条清单项，阶段名做前缀。 */
    function tbFlowChecklist(def) {
      var out = []
      var stages = (def && def.stages) || []
      for (var i = 0; i < stages.length; i++) {
        var gs = stages[i].gates || []
        for (var j = 0; j < gs.length; j++) {
          out.push({
            id: String(gs[j].id),
            text: '[' + String(stages[i].title || stages[i].id) + '] ' + String(gs[j].text || gs[j].id),
            stage: String(stages[i].id), gate: String(gs[j].id),
            checked: false, evidence: '', by: ''
          })
        }
      }
      return out
    }
    // ── ②c 卡片驱动执行：定时（schedule）/ 子项（subtasks）/ 执行态（runState）──
    // schedule 语义：interval=每 N 分钟；daily=每天 HH:MM；at=一次性时间点。
    // nextAt 由插件计算（磁盘上是真相），定时器只做「到点就执行」的扫描。
    function tbNextAt(sch, from) {
      var base = from ? new Date(from) : new Date()
      if (!(base instanceof Date) || isNaN(base.getTime())) base = new Date()
      if (sch.mode === 'interval') {
        return new Date(base.getTime() + sch.everyMinutes * 60000).toISOString()
      }
      if (sch.mode === 'daily') {
        var m = String(sch.atTime || '09:00').match(/^(\d{1,2}):(\d{2})$/)
        var h = m ? Math.max(0, Math.min(23, parseInt(m[1], 10))) : 9
        var mi = m ? Math.max(0, Math.min(59, parseInt(m[2], 10))) : 0
        var d = new Date(base.getTime())
        d.setHours(h, mi, 0, 0)
        if (d.getTime() <= base.getTime()) d.setDate(d.getDate() + 1)
        return d.toISOString()
      }
      var at = sch.atISO ? new Date(sch.atISO) : null
      if (at && !isNaN(at.getTime())) return at.toISOString()
      return ''
    }
    function tbNormSchedule(raw) {
      var s = (raw && typeof raw === 'object') ? raw : {}
      var mode = ['interval', 'daily', 'at'].indexOf(String(s.mode || '')) >= 0 ? String(s.mode) : 'interval'
      var out = {
        enabled: !!s.enabled,
        mode: mode,
        everyMinutes: isNum(s.everyMinutes) ? Math.max(1, Math.min(10080, Math.floor(s.everyMinutes))) : 60,
        atTime: /^\d{1,2}:\d{2}$/.test(String(s.atTime || '')) ? String(s.atTime) : '09:00',
        atISO: String(s.atISO || ''),
        convId: String(s.convId || ''),
        nextAt: String(s.nextAt || ''),
        lastAt: String(s.lastAt || '')
      }
      if (!out.enabled) { out.nextAt = ''; return out }
      var nx = out.nextAt ? new Date(out.nextAt) : null
      if (!nx || isNaN(nx.getTime()) || nx.getTime() <= Date.now() - 60000) out.nextAt = tbNextAt(out)
      return out
    }
    /** agent 拆解的子项（由执行回合写入，只做展示与进度；人可在详情里勾选推进）。 */
    function tbNormSubtasks(raw) {
      var a = Array.isArray(raw) ? raw : []
      var out = []
      for (var i = 0; i < a.length && out.length < 50; i++) {
        var s = a[i]
        if (!s) continue
        var text = String(typeof s === 'string' ? s : (s.text || s.title || '')).trim()
        if (!text) continue
        var st = String((s && s.status) || 'pending')
        if (SUB_STATUS.indexOf(st) < 0) st = 'pending'
        out.push({
          id: String((s && s.id) || ('s-' + i + '-' + Math.random().toString(36).slice(2, 5))),
          text: text, status: st,
          by: String((s && s.by) || ''), at: String((s && s.at) || '')
        })
      }
      return out
    }
    function tbCreate(o) {
      var a = (o && typeof o === 'object') ? o : {}
      var title = String(a.title || a.subject || '').trim()
      if (!title) return { ok: false, error: '任务标题不能为空' }
      var l = tbLoad()
      var st = String(a.status || '')
      var sch0 = tbNormSchedule(a.schedule)
      if (sch0.enabled && !sch0.convId) sch0.convId = String(a.convId || '')
      var t = {
        id: tbNewId(), title: title,
        description: String(a.description || ''), prompt: String(a.prompt || ''),
        workspace: String(a.workspace || tbWorkspaceId()),
        urgency: TB_URGENCY.indexOf(String(a.urgency || '')) >= 0 ? String(a.urgency) : 'normal',
        status: TB_MAIN.indexOf(st) >= 0 ? st : 'todo',
        blocked: !!a.blocked,
        flowId: String(a.flowId || ''), flowScope: '', flowVersion: 0, runId: '',
        checklist: [], comments: [], executions: [],
        claimedBy: '', claimedAt: '', claimedByKind: '',
        schedule: sch0,
        runState: { running: false, startedAt: '', startedBy: '', lastEndAt: '', lastError: '', runCount: 0 },
        subtasks: tbNormSubtasks(a.subtasks),
        version: 1,
        createdAt: nowIso(), updatedAt: nowIso(),
        createdBy: tbActor(a.by), updatedBy: tbActor(a.by), trashedAt: ''
      }
      if (t.flowId) {
        var f = resolveFlow(t.flowId)
        if (f) { t.flowScope = f.scope; t.flowVersion = f.def.version || 1; t.checklist = tbFlowChecklist(f.def) }
      }
      l.tasks.push(t)
      if (!tbSave(l)) return { ok: false, error: '任务库写入失败（' + P_TB + '）' }
      tbMirror(t)
      return { ok: true, task: t, revision: l.revision }
    }
    function tbUpdate(id, patch, o) {
      var p = (patch && typeof patch === 'object') ? patch : {}
      var a = (o && typeof o === 'object') ? o : {}
      var l = tbLoad()
      var t = tbFind(l, id)
      if (!t) return { ok: false, error: '任务不存在：' + id }
      if (p.title !== undefined && String(p.title).trim()) t.title = String(p.title).trim()
      if (typeof p.description === 'string') t.description = p.description
      if (typeof p.prompt === 'string') t.prompt = p.prompt
      if (p.schedule !== undefined) {
        t.schedule = tbNormSchedule(p.schedule)
        // 每次设置都从「现在」重算下次执行时间（避免沿用过期 nextAt）
        t.schedule.nextAt = t.schedule.enabled ? tbNextAt(t.schedule) : ''
      }
      if (Array.isArray(p.subtasks)) t.subtasks = tbNormSubtasks(p.subtasks)
      if (TB_URGENCY.indexOf(String(p.urgency || '')) >= 0) t.urgency = String(p.urgency)
      if (typeof p.blocked === 'boolean') t.blocked = p.blocked
      if (typeof p.flowId === 'string' && p.flowId !== t.flowId) {
        t.flowId = p.flowId
        var f = p.flowId ? resolveFlow(p.flowId) : null
        t.flowScope = f ? f.scope : ''
        t.flowVersion = f ? (f.def.version || 1) : 0
        if (f) t.checklist = tbFlowChecklist(f.def)
      }
      t.version = (isNum(t.version) ? t.version : 1) + 1
      t.updatedAt = nowIso()
      t.updatedBy = tbActor(a.by)
      if (!tbSave(l)) return { ok: false, error: '任务库写入失败' }
      tbMirror(t)
      return { ok: true, task: t, revision: l.revision }
    }
    /**
     * 改状态：看板拖拽 / 卡片按钮 / agent claim 全走这里。
     * 协议闸（照参考）：agent 永远不能把卡移到 done —— 验收只能由人做。
     */
    function tbMove(id, to, o) {
      var a = (o && typeof o === 'object') ? o : {}
      var l = tbLoad()
      var t = tbFind(l, id)
      if (!t) return { ok: false, error: '任务不存在：' + id }
      var s = String(to || '')
      if (TB_STATUS.indexOf(s) < 0) return { ok: false, error: '未知状态：' + s }
      var by = tbActor(a.by)
      // 「已完成」默认只能由人验收；a.autoAccept 仅由「全流程自动」驱动内部传入
      // （只在 tbAutoAccept 里出现，agent 工具面无法构造该参数）。
      if (s === 'done' && by.kind !== 'user' && !a.autoAccept) {
        return { ok: false, error: '「已完成」只能由人验收 —— agent 不能自己把任务移到已完成' }
      }
      if (t.status === s) return { ok: true, task: t, unchanged: true, revision: l.revision }
      if (!tbCanTransition(t.status, s)) {
        return { ok: false, error: '无法从「' + tbLabel(t.status) + '」移到「' + tbLabel(s) + '」' }
      }
      var from = t.status
      t.status = s
      // 拖回「待规划」= 需要重新规划：已有规划历史的卡把阶段从 final/review 退回 drafting
      // （历史版本与 finalAt/acceptedAt 保留，作为审计痕迹，不在重规划时抹掉）
      if (s === 'backlog' && t.plan && typeof t.plan === 'object' &&
          Array.isArray(t.plan.rounds) && t.plan.rounds.length > 0) {
        t.plan.stage = 'drafting'
        t.plan.updatedAt = nowIso()
      }
      if (s === 'in_progress') {
        t.claimedBy = String(a.sessionId || by.sessionId || '')
        t.claimedAt = nowIso()
        t.claimedByKind = String(a.claimKind || by.kind || '')
      }
      if (from === 'in_progress') {
        t.claimedBy = ''; t.claimedAt = ''; t.claimedByKind = ''            // 离开进行中即释放认领（照参考）
        if (t.runState && t.runState.running) { t.runState.running = false; t.runState.lastEndAt = nowIso() }
      }
      t.version = (isNum(t.version) ? t.version : 1) + 1
      t.updatedAt = nowIso()
      t.updatedBy = by
      if (!tbSave(l)) return { ok: false, error: '任务库写入失败' }
      tbMirror(t)
      return { ok: true, task: t, from: from, revision: l.revision }
    }
    /**
     * 认领：待办/待规划 → 进行中，并记认领者。
     * 语义（卡片驱动 agent）：人点「认领」= 自己接手；agent 经 taskboard_claim 认领 = 会话持有。
     * 被他人持有时不可抢；「进行中」再认领视为幂等成功（同一认领者）。
     */
    function tbClaim(id, by, convId) {
      var l = tbLoad()
      var t = tbFind(l, id)
      if (!t) return { ok: false, error: '任务不存在：' + id }
      if (t.status === 'in_progress') {
        if (t.claimedBy && convId && t.claimedBy !== String(convId)) {
          return { ok: false, error: '任务 ' + t.id + ' 已被 ' + t.claimedBy + ' 认领，不能抢' }
        }
        return { ok: true, task: t, unchanged: true, revision: l.revision }
      }
      if (t.status !== 'todo' && t.status !== 'backlog') {
        return { ok: false, error: '只有「待办 / 待规划」的任务可以认领（当前「' + tbLabel(t.status) + '」）' }
      }
      return tbMove(id, 'in_progress', { by: by, sessionId: String(convId || ''), claimKind: tbActor(by).kind })
    }
    /** 释放认领：进行中 → 待办，并清空认领者（tbMove 已处理）。 */
    function tbRelease(id, by) {
      var l = tbLoad()
      var t = tbFind(l, id)
      if (!t) return { ok: false, error: '任务不存在：' + id }
      if (t.status !== 'in_progress') return { ok: false, error: '任务不在「进行中」，无需释放' }
      return tbMove(id, 'todo', { by: by })
    }
    function tbComment(id, text, by) {
      var l = tbLoad()
      var t = tbFind(l, id)
      if (!t) return { ok: false, error: '任务不存在：' + id }
      var s = String(text || '').trim()
      if (!s) return { ok: false, error: '评论内容不能为空' }
      if (!Array.isArray(t.comments)) t.comments = []
      t.comments.push({ id: 'c-' + Date.now() + '-' + Math.random().toString(36).slice(2, 5), text: s, at: nowIso(), by: tbActor(by) })
      t.updatedAt = nowIso()
      if (!tbSave(l)) return { ok: false, error: '任务库写入失败' }
      tbMirror(t)
      return { ok: true, task: t }
    }
    // ── ②c 规划阶段：「待规划」列可执行化（2026-10-17 用户诉求）──────────
    // 改造前：backlog 只能看 / 只能直接认领去做，没有「规划」语义。
    // 改造后：backlog 卡由 agent 做**真实规划**（调研 → 产出可执行拆解），
    //   多轮讨论可反复修订（每轮留版本），定稿后移入「待办」，拆解即执行骨架。
    // 阶段：''（未规划）→ drafting（规划中）→ review（待确认）→ final（已定稿）
    var PLAN_STAGE_LABEL = { '': '未规划', drafting: '规划中', review: '待确认', final: '已定稿' }
    /** 规划状态（惰性初始化；老卡无 plan 字段 → 视为未规划）。 */
    function tbPlanState(t) {
      if (!t.plan || typeof t.plan !== 'object') t.plan = {}
      var p = t.plan
      if (!Array.isArray(p.rounds)) p.rounds = []
      if (!Array.isArray(p.questions)) p.questions = []
      if (!Array.isArray(p.risks)) p.risks = []
      if (!Array.isArray(p.discussion)) p.discussion = []
      if (typeof p.stage !== 'string') p.stage = ''
      if (typeof p.summary !== 'string') p.summary = ''
      if (typeof p.convId !== 'string') p.convId = ''
      // 最近一次派发状态：{at, via, ok, error, convId, repliedAt?}（无则 null）
      if (!p.dispatch || typeof p.dispatch !== 'object') p.dispatch = null
      return p
    }
    function tbPlanStageLabel(s) { return PLAN_STAGE_LABEL[String(s || '')] || String(s || '') }
    /**
     * 规划任务书：让 agent 产出**可执行规划**（本轮不做实现）。
     * 与执行任务书的区别：不要求动手，要求调研 + 拆解 + 待确认问题；定稿走 final=true。
     */
    function tbPlanTaskText(t) {
      var p = tbPlanState(t)
      var lines = []
      lines.push('你在为「任务看板」上的一张【待规划】卡做规划 —— 本轮**不做实现**，唯一产物是规划。')
      lines.push('卡片：' + t.id + ' 「' + String(t.title || '') + '」')
      lines.push('')
      if (String(t.description || '').trim()) {
        lines.push('## 任务描述')
        lines.push(String(t.description))
        lines.push('')
      }
      if (String(t.prompt || '').trim()) {
        lines.push('## 已有执行提示词（规划时参考；发现不足请在规划里指出或改写建议）')
        lines.push(String(t.prompt))
        lines.push('')
      }
      if (p.rounds.length) {
        var last = p.rounds[p.rounds.length - 1]
        lines.push('## 当前规划（第 ' + last.n + ' 版，' + String(last.at || '') + '）')
        if (last.summary) lines.push('概要：' + String(last.summary))
        var ls = Array.isArray(last.subtasks) ? last.subtasks : []
        for (var i = 0; i < ls.length; i++) lines.push((i + 1) + '. ' + String(ls[i].text || ''))
        if ((last.questions || []).length) {
          lines.push('')
          lines.push('待确认问题：')
          for (var q = 0; q < last.questions.length; q++) lines.push('- ' + String(last.questions[q]))
        }
        if ((last.risks || []).length) {
          lines.push('')
          lines.push('已识别风险：')
          for (var rk = 0; rk < last.risks.length; rk++) lines.push('- ' + String(last.risks[rk]))
        }
        lines.push('')
      }
      if (p.discussion.length) {
        lines.push('## 用户已提出的规划意见（必须逐条回应后再改）')
        var ds = p.discussion.slice(-10)
        for (var d = 0; d < ds.length; d++) {
          lines.push('- [' + String(ds[d].at || '') + '] ' + String(ds[d].text || ''))
        }
        lines.push('')
      }
      lines.push('## 规划协议（必须遵守）')
      lines.push('1. 先调研再规划：读相关代码/文档，确认现状、影响面、可复用点；不要凭想象编步骤。')
      lines.push('2. 写规划：调用 taskboard_plan(id="' + t.id + '", summary="一句话方案", subtasks=["1 …","2 …"], questions=["需要用户拍板的点"], risks=["风险"])。')
      lines.push('3. 多轮迭代：每轮修订都再调用一次 taskboard_plan（自动记为新版本）；用户在看板「规划讨论」里的意见要先回应再改。')
      lines.push('4. 定稿：规划成熟后调用 taskboard_plan(id="' + t.id + '", final=true, ...) —— 卡进入「待确认 / 待办」。')
      lines.push('5. 不确定的点写进 questions 并结束本轮，不要替用户拍板；问题没答完不要定稿。')
      lines.push('6. 本阶段不要执行任务本身（不写业务代码）；定稿后才由执行流程接手。')
      return lines.join('\n')
    }
    /**
     * 派发规划任务书：followup 是唯一真实通道（多轮对话的载体）。
     * ★ 不自欺（2026-09 第 2 次监督）：只有 followup 明确成功（未返回 ok:false 且不抛错）
     *   才算派发；无会话 / 投递被拒 / 抛错 三种情形统一记进 plan.dispatch 并返回 ok:false。
     *   历史实现里的 ctx.subagent 兜底已删除 —— RPC 路径上 SubagentRuntimeFor("") 恒 nil
     *   （只在 autopilot 决策器期间绑定），且 subagent.run 同步阻塞插件 VM（timer 回调
     *   在 p.withLock + runJSWithTimeout 内执行，会锁死看板全部 RPC）。保留只是伪兜底。
     * 只动 plan 阶段与 convId，不改任务状态（状态仍 backlog，等定稿才入 todo）。
     */
    function tbPlanDispatch(id, opts) {
      var a = (opts && typeof opts === 'object') ? opts : {}
      var l = tbLoad()
      var t = tbFind(l, id)
      if (!t) return { ok: false, error: '任务不存在：' + id }
      if (t.status !== 'backlog') {
        return { ok: false, error: '只有「待规划」的任务可以规划（当前「' + tbLabel(t.status) + '」）；执行中的卡请用 taskboard_execute' }
      }
      var p = tbPlanState(t)
      if (p.stage === 'review') {
        return { ok: false, error: '规划已提交、等待确认；如需继续修订，请先在「规划讨论」里补充意见（会自动回到规划中）' }
      }
      var pick = tbExecResolveConv(a, t, { sessionId: p.convId })
      var convId = pick.convId
      var text = tbPlanTaskText(t)
      var via = ''
      var queued = false
      var err = ''
      if (!convId) {
        err = '没有可用的规划会话：请先在对话里打开（或新建）一个会话，再点「开始规划」'
      } else {
        try {
          var ag = ctx.agents
          if (!ag || typeof ag.followup !== 'function') {
            err = '宿主没有提供会话投递能力（ctx.agents.followup 不可用），请重启 IDE 后再试'
          } else {
            var rr = ag.followup(convId, text)
            if (rr && rr.ok === false) {
              err = '会话投递被拒：' + String(rr.error || '未知原因') + '（请确认该会话仍在对话列表里）'
            } else {
              queued = !!(rr && rr.queued)
              via = queued ? 'followup-queued' : 'followup'
              tbRememberConv(convId)
            }
          }
        } catch (e) {
          err = '会话投递失败：' + String((e && e.message) || e)
          logWarn('规划派发 followup 失败: ' + e)
        }
      }
      // 最近一次派发状态（供看板徽标 / 重试按钮读；只保留最近一条）
      p.dispatch = { at: nowIso(), via: via, ok: !err, error: err, convId: convId }
      if (!err) {
        if (!p.startedAt) p.startedAt = nowIso()
        if (p.stage !== 'review') p.stage = 'drafting'
        if (convId) p.convId = convId
      }
      p.updatedAt = nowIso()
      t.updatedAt = nowIso()
      if (!tbSave(l)) return { ok: false, error: '任务库写入失败' }
      tbMirror(t)
      if (err) return { ok: false, dispatched: false, error: err, convId: convId, convSource: pick.source, task: t, plan: p }
      return { ok: true, dispatched: true, via: via, queued: queued, convId: convId, convSource: pick.source, task: t, plan: p }
    }
    /**
     * 规划定稿：stage → review（待确认）或 final（已定稿 + 移入待办）。
     * 直接入待办的三个来源：人显式确认、driver.planAutoAccept=true、驱动内部 autoAccept。
     * 其余情况停在 review，等人在看板点「确认规划 → 待办」（agent 不能替用户拍板）。
     */
    function tbPlanFinalize(id, by, autoAccept) {
      var l = tbLoad()
      var t = tbFind(l, id)
      if (!t) return { ok: false, error: '任务不存在：' + id }
      if (t.status !== 'backlog') {
        return { ok: false, error: '只有「待规划」的任务需要规划定稿（当前「' + tbLabel(t.status) + '」）' }
      }
      var p = tbPlanState(t)
      if (!p.rounds.length) return { ok: false, error: '还没有任何规划版本：先调用 taskboard_plan 写一版，不能空定稿' }
      var who = tbActor(by)
      var drv = tbDriverLoad()
      var autoMove = !!autoAccept || drv.planAutoAccept === true || who.kind === 'user'
      p.finalAt = nowIso()
      p.finalBy = who
      p.updatedAt = p.finalAt
      if (autoMove) {
        p.stage = 'final'
        p.acceptedAt = p.finalAt
        p.acceptedBy = who
      } else {
        p.stage = 'review'
      }
      t.updatedAt = nowIso()
      if (!tbSave(l)) return { ok: false, error: '任务库写入失败' }
      var moved = false
      if (autoMove) {
        var mr = tbMove(id, 'todo', { by: by })
        if (!mr.ok) return mr
        moved = true
        t = mr.task
      }
      tbMirror(t)
      return { ok: true, task: t, plan: tbPlanState(t), stage: p.stage, moved: moved }
    }
    /** 人确认规划 → 移入「待办」（等同定稿 + 直接流转）。 */
    function tbPlanAccept(id, by) { return tbPlanFinalize(id, by || { kind: 'user' }, true) }
    /** 规划讨论：人提意见 → 记进 plan.discussion 并投递到规划会话（agent 据此修订）。 */
    function tbPlanDiscuss(id, text, by, convId) {
      var s = String(text || '').trim()
      if (!s) return { ok: false, error: '讨论内容不能为空' }
      var l = tbLoad()
      var t = tbFind(l, id)
      if (!t) return { ok: false, error: '任务不存在：' + id }
      if (t.status !== 'backlog') {
        return { ok: false, error: '规划讨论只用于「待规划」的卡（当前「' + tbLabel(t.status) + '」）' }
      }
      var p = tbPlanState(t)
      p.discussion.push({ at: nowIso(), by: tbActor(by).kind, text: s })
      if (p.discussion.length > 50) p.discussion = p.discussion.slice(-50)
      // 新意见 → 回到规划中（原来在「待确认」也要让 agent 再改一版）
      p.stage = 'drafting'
      var target = String(convId || p.convId || tbLastConv() || '')
      if (target) p.convId = target
      p.updatedAt = nowIso()
      t.updatedAt = nowIso()
      if (!tbSave(l)) return { ok: false, error: '任务库写入失败' }
      tbMirror(t)
      var dispatched = false
      var via = ''
      var why = ''
      if (!target) {
        why = '最近活跃会话池为空'
      } else {
        try {
          var ag = ctx.agents
          if (ag && typeof ag.followup === 'function') {
            var msg = '用户在「任务看板」任务卡 ' + t.id + '「' + String(t.title || '') + '」的规划讨论里提出：\n\n' + s +
              '\n\n请据此修订规划（先逐条回应这条意见），调用 taskboard_plan 写新版本；仍有不确定的点写进 questions。'
            var rr = ag.followup(target, msg)
            if (rr && rr.ok === false) {
              why = '会话投递被拒：' + String(rr.error || '未知原因')
            } else {
              dispatched = true
              via = (rr && rr.queued) ? 'followup-queued' : 'followup'
              tbRememberConv(target)
            }
          } else {
            why = '宿主没有提供会话投递能力（ctx.agents.followup 不可用）'
          }
        } catch (e) {
          why = String((e && e.message) || e)
          logWarn('规划讨论 followup 投递失败: ' + e)
        }
      }
      // 最近一次派发状态（与 tbPlanDispatch 同构；失败可被看板读到并重试）
      p.dispatch = { at: nowIso(), via: via, ok: dispatched, error: dispatched ? '' : why, convId: target }
      p.updatedAt = nowIso()
      t.updatedAt = nowIso()
      if (!tbSave(l)) return { ok: false, error: '任务库写入失败' }
      tbMirror(t)
      if (!dispatched) {
        // 意见已落盘（不回滚），但必须把「未投递」如实告诉用户
        return {
          ok: false, recorded: true, dispatched: false, via: '', convId: target,
          error: '意见已记录在规划讨论里，但没有投递成功——请先在对话里打开（或新建）一个会话，再点「开始规划」重新派发' +
            (why ? '（' + why + '）' : ''),
          task: t, plan: tbPlanState(t)
        }
      }
      return { ok: true, task: t, plan: tbPlanState(t), dispatched: true, via: via, convId: target }
    }
    /**
     * 写规划 / 写执行拆解（同一工具两种语义，由卡状态决定）：
     *   · status=backlog（待规划）→ 规划模式：记一轮版本（summary/questions/risks/subtasks）；
     *     final=true → 定稿（directly 入待办 or 停在待确认，见 tbPlanFinalize）。
     *   · 其它状态 → 执行模式：写执行拆解（原语义不变）。
     */
    function tbPlan(id, subtasks, by, opts) {
      var o = (opts && typeof opts === 'object') ? opts : {}
      var l = tbLoad()
      var t = tbFind(l, id)
      if (!t) return { ok: false, error: '任务不存在：' + id }
      var who = tbActor(by)
      var list = tbNormSubtasks(subtasks)
      for (var i = 0; i < list.length; i++) {
        list[i].by = who.kind
        if (!list[i].at) list[i].at = nowIso()
      }
      var qs = []
      if (Array.isArray(o.questions)) {
        for (var qi = 0; qi < o.questions.length; qi++) {
          var qt = String(o.questions[qi] || '').trim()
          if (qt) qs.push(qt)
        }
      }
      var rk = []
      if (Array.isArray(o.risks)) {
        for (var ri = 0; ri < o.risks.length; ri++) {
          var rt = String(o.risks[ri] || '').trim()
          if (rt) rk.push(rt)
        }
      }
      var summary = String(o.summary || '').trim()
      if (!list.length && !summary && !qs.length && !o.final) {
        return { ok: false, error: '拆解不能为空：subtasks 需为非空字符串数组（规划模式可只给 summary / questions）' }
      }
      // ── 规划模式（待规划卡）──
      if (t.status === 'backlog') {
        var p = tbPlanState(t)
        var n = p.rounds.length + 1
        var roundSubs = []
        for (var ri2 = 0; ri2 < list.length; ri2++) roundSubs.push({ text: list[ri2].text, status: list[ri2].status })
        p.rounds.push({ n: n, at: nowIso(), by: who.kind, summary: summary, subtasks: roundSubs, questions: qs, risks: rk })
        if (p.rounds.length > 20) p.rounds = p.rounds.slice(-20)
        p.summary = summary
        p.questions = qs
        p.risks = rk
        p.stage = 'drafting'
        // 有新版本回写 → 清掉「派发失败」错误（派发已产生真实结果，不是失败态）
        if (p.dispatch && typeof p.dispatch === 'object' && p.dispatch.error) {
          p.dispatch.ok = true
          p.dispatch.error = ''
          p.dispatch.repliedAt = nowIso()
        }
        p.updatedAt = nowIso()
        if (list.length) t.subtasks = list          // 规划产物同时作执行骨架预览
        t.updatedAt = nowIso()
        if (!tbSave(l)) return { ok: false, error: '任务库写入失败' }
        tbMirror(t)
        if (o.final) {
          var fr = tbPlanFinalize(id, by, !!o.autoAccept)
          if (!fr.ok) return fr
          return { ok: true, task: fr.task, plan: fr.plan, round: n, stage: fr.stage, movedToTodo: !!fr.moved }
        }
        return { ok: true, task: t, plan: tbPlanState(t), round: n, stage: 'drafting' }
      }
      // ── 执行模式（原语义：写执行拆解）──
      if (!list.length) return { ok: false, error: '拆解不能为空：subtasks 需为非空字符串数组' }
      t.subtasks = list
      var pe = tbPlanState(t)
      if (summary) pe.summary = summary
      if (o.final) {
        pe.stage = 'final'
        pe.finalAt = pe.finalAt || nowIso()
        pe.finalBy = pe.finalBy || who
      }
      t.updatedAt = nowIso()
      if (!tbSave(l)) return { ok: false, error: '任务库写入失败' }
      tbMirror(t)
      return { ok: true, task: t, plan: pe }
    }
    /** 勾选/取消验收清单项（agent 可勾并附证据；人可全量编辑）。 */
    function tbCheck(id, itemId, checked, note, by) {
      var l = tbLoad()
      var t = tbFind(l, id)
      if (!t) return { ok: false, error: '任务不存在：' + id }
      var list = Array.isArray(t.checklist) ? t.checklist : []
      var it = null
      for (var i = 0; i < list.length; i++) if (String(list[i].id) === String(itemId)) it = list[i]
      if (!it) return { ok: false, error: '清单项不存在：' + itemId }
      it.checked = !!checked
      it.evidence = String(note || '')
      it.by = tbActor(by).kind
      t.updatedAt = nowIso()
      if (!tbSave(l)) return { ok: false, error: '任务库写入失败' }
      tbMirror(t)
      return { ok: true, task: t }
    }
    /** agent 交报告：留执行记录 + 任务转「待验收」（照参考 report 语义）。 */
    function tbReport(id, o) {
      var a = (o && typeof o === 'object') ? o : {}
      var l = tbLoad()
      var t = tbFind(l, id)
      if (!t) return { ok: false, error: '任务不存在：' + id }
      if (!Array.isArray(t.executions)) t.executions = []
      t.executions.push({
        id: 'x-' + Date.now() + '-' + Math.random().toString(36).slice(2, 5),
        at: nowIso(), by: tbActor(a.by || 'agent'),
        outcome: String(a.outcome || 'succeeded'),
        summary: String(a.summary || ''), files: Array.isArray(a.files) ? a.files : [], risks: String(a.risks || '')
      })
      if (t.executions.length > 20) t.executions = t.executions.slice(-20)
      var rs = tbExecState(t)
      rs.running = false
      rs.lastEndAt = nowIso()
      rs.lastError = String(a.outcome || 'succeeded') === 'failed' ? String(a.risks || '执行失败') : ''
      var moved = false
      if (t.status === 'in_progress') { t.status = 'in_review'; moved = true }
      t.updatedAt = nowIso()
      if (!tbSave(l)) return { ok: false, error: '任务库写入失败' }
      tbMirror(t)
      return { ok: true, task: t, movedToReview: moved }
    }
    /**
     * 任务 ↔ 流程绑定（启动流程时用）：回写 runId；卡上未设流程时补 flowId/flowScope/flowVersion
     * 并按该流程门禁生成验收清单（DoD）。所有启动入口（flow_start 工具 / flow_action.start /
     * taskboard_action.startFlow）都经 startRun 调这里 —— 绑定只有这一处，避免漏写。
     */
    function tbBindRun(id, runId, flowMeta) {
      var l = tbLoad()
      var t = tbFind(l, id)
      if (!t) return { ok: false, error: '任务不存在：' + id }
      var fm = (flowMeta && typeof flowMeta === 'object') ? flowMeta : null
      if (fm && fm.flowId && !t.flowId) {
        var f = resolveFlow(String(fm.flowId))
        t.flowId = String(fm.flowId)
        t.flowScope = String(fm.flowScope || (f ? f.scope : ''))
        t.flowVersion = isNum(fm.flowVersion) ? fm.flowVersion : (f ? (f.def.version || 1) : 1)
        if (f && (!Array.isArray(t.checklist) || !t.checklist.length)) t.checklist = tbFlowChecklist(f.def)
      }
      t.runId = String(runId || '')
      t.updatedAt = nowIso()
      if (!tbSave(l)) return { ok: false, error: '任务库写入失败' }
      tbMirror(t)
      return { ok: true, task: t }
    }
    /**
     * 流程结合（用户选定方案）：流程推进后把结果同步到绑定它的任务。
     *   · 流程门禁通过 → 勾选对应验收清单项（DoD，by=flow）
     *   · 流程全部走完 → 任务自动转「待验收」（等人工验收，绝不自动 completed）
     */
    function tbSyncFromRun(run) {
      if (!run || !run.runId) return null
      var l = tbLoad()
      var t = null
      for (var i = 0; i < l.tasks.length; i++) if (l.tasks[i].runId === run.runId) t = l.tasks[i]
      if (!t) return null
      var changed = false
      var cl = Array.isArray(t.checklist) ? t.checklist : []
      for (var k = 0; k < cl.length; k++) {
        var item = cl[k]
        if (!item || !item.gate || item.checked) continue
        var passed = false
        for (var si = 0; si < (run.stages || []).length; si++) {
          var gr = gateResult(run.stages[si], item.gate)
          if (gr && (gr.status === 'pass' || gr.status === 'approved')) { passed = true; break }
        }
        if (passed) { item.checked = true; item.by = 'flow'; changed = true }
      }
      if (run.status === 'completed' && t.status !== 'in_review' && t.status !== 'done' && t.status !== 'archived' && t.status !== 'canceled') {
        t.status = 'in_review'
        changed = true
      }
      if (changed) {
        t.updatedAt = nowIso()
        if (tbSave(l)) tbMirror(t)
      }
      return { ok: true, task: t, changed: changed }
    }
    // ── ②d 卡片驱动 agent：执行 / 定时 / 拆解（2026-10-17 用户诉求）──
    // 通道：ctx.agents.followup(convId, 任务书) → 工作会话起一轮 →
    //   工作 agent 自然结束时 autopilot 监督者接管裁决（continue 继续 / done 收尾），
    //   形成「一张卡 = 一个被监督的任务」的多轮闭环；
    //   工作 agent 通过 taskboard_plan / taskboard_check / taskboard_report 把拆解、进度、报告回填看板。
    // 唯一通道就是 followup：RPC 路径没有可用的 subagent 运行环境（详见 tbExecute 注释），不做伪兜底。
    function tbExecState(t) {
      if (!t.runState || typeof t.runState !== 'object') {
        t.runState = { running: false, startedAt: '', startedBy: '', lastEndAt: '', lastError: '', sessionId: '', runCount: 0 }
      }
      if (!isNum(t.runState.runCount)) t.runState.runCount = 0
      return t.runState
    }
    /** 任务书：描述 + 执行提示词 + DoD + 看板协议（执行 agent 按此干活并回填）。 */
    function tbExecTaskText(t) {
      var lines = []
      lines.push('你现在在执行「任务看板」上的一张任务卡：' + t.id + ' 「' + String(t.title || '') + '」。')
      lines.push('请直接动手完成，不要只给建议。')
      lines.push('')
      if (String(t.description || '').trim()) {
        lines.push('## 任务描述')
        lines.push(String(t.description))
        lines.push('')
      }
      if (String(t.prompt || '').trim()) {
        lines.push('## 执行提示词（必须遵守）')
        lines.push(String(t.prompt))
        lines.push('')
      }
      var cl = Array.isArray(t.checklist) ? t.checklist : []
      if (cl.length) {
        lines.push('## 验收清单（DoD）')
        for (var i = 0; i < cl.length; i++) {
          lines.push('- [' + (cl[i].checked ? 'x' : ' ') + '] ' + cl[i].id + ' · ' + String(cl[i].text || ''))
        }
        lines.push('')
        lines.push('每完成一项，调用 taskboard_check 勾选并附证据（note）；不要一次性全勾。')
        lines.push('')
      }
      lines.push('## 看板协议（必须遵守）')
      lines.push('1. 开工前调用 taskboard_plan(id="' + t.id + '", subtasks=["...","..."]) 写下执行拆解（3-8 条）。')
      lines.push('2. 过程中用 taskboard_comment 记录关键进展或卡点；子项状态随进展更新。')
      lines.push('3. 结束时调用 taskboard_report(id="' + t.id + '", summary="...", files=[...], risks="...") 交报告，任务会自动转「待验收」。')
      lines.push('4. 不要用 taskboard_move 把任务移到 done —— 验收只能由人做。')
      if (t.flowId) {
        var fp = t.flowProgress || null
        lines.push('')
        lines.push('## 流程骨架')
        lines.push('流程 ' + t.flowId + (fp && fp.label ? '（当前阶段：' + fp.label + '）' : '') + '；门禁即上面的 DoD，按顺序推进。')
      }
      // 全流程自动：告知 agent 验收由看板自动完成 + 流程阶段自己推进（approval 仍须等人）
      var drv = null
      try { drv = tbDriverLoad() } catch (eD) { drv = null }
      if (drv && drv.mode === 'auto') {
        lines.push('')
        lines.push('## 自动驱动模式（用户已在看板开启「全流程自动」）')
        lines.push('- 本卡完成后看板会自动验收（用户已预授权），但你仍必须调用 taskboard_report 交报告，不能跳过。')
        if (t.flowId) {
          lines.push('- 流程按骨架逐阶段推进：本阶段任务与门禁完成后调用 flow_advance 推进到下一阶段，直到全部阶段完成。')
          lines.push('- approval 类门禁**必须停下等人批准**，不得自行通过（插件与 agent 都无权自动批准）。')
        }
      }
      return lines.join('\n')
    }
    /**
     * 解析执行会话：显式参数 → 卡上 execConvId → 上次执行会话 → 定时指定会话 → 最近活跃会话池。
     * 「新建卡 → 设定时 → 到点自动跑」靠最后一档兜底（看板 RPC 都会把 convId 记进会话池）。
     */
    function tbExecResolveConv(a, t, rs) {
      var explicit = String((a && a.convId) || '')
      if (explicit) return { convId: explicit, source: 'explicit' }
      if (t.execConvId) return { convId: String(t.execConvId), source: 'execConvId' }
      var lastSess = String((rs && rs.sessionId) || '')
      if (lastSess && lastSess.indexOf('subagent') !== 0) return { convId: lastSess, source: 'runState' }
      var schConv = (t.schedule && t.schedule.convId) ? String(t.schedule.convId) : ''
      if (schConv) return { convId: schConv, source: 'schedule' }
      var last = tbLastConv()
      if (last) return { convId: last, source: 'lastActive' }
      return { convId: '', source: 'none' }
    }
    /**
     * 执行一张卡：**先投递任务书 → 投递成功后才认领（进行中）→ 记 runState**。
     * 唯一通道：ctx.agents.followup(convId) 唤醒已存在的工作会话（autopilot 监督者多轮接管）。
     * 不保留 subagent 兜底：RPC 路径上 SubagentRuntimeFor("") 恒 nil、available() 恒 false，
     *   且 sub.run 同步阻塞插件 VM —— 看起来能兜底，实则永远不跑（伪兜底）。
     * 失败语义（不允许「进行中但没人跑」）：
     *   ok:false + dispatched:false + runState.running=false + lastError 可读；
     *   卡若已被其它路径置于「进行中」，一并退回「待办」，用户可原地点「执行」重试。
     * 完成由工作 agent 的 taskboard_report 回填（running=false + 转待验收）。
     */
    function tbExecute(id, opts) {
      var a = (opts && typeof opts === 'object') ? opts : {}
      var l = tbLoad()
      var t = tbFind(l, id)
      if (!t) return { ok: false, error: '任务不存在：' + id }
      if (t.status === 'done' || t.status === 'archived' || t.status === 'canceled') {
        return { ok: false, error: '任务已结束（' + tbLabel(t.status) + '），不能执行' }
      }
      var rs = tbExecState(t)
      if (rs.running) return { ok: false, error: '任务正在执行中（开始于 ' + rs.startedAt + '）' }
      var pick = tbExecResolveConv(a, t, rs)
      var convId = pick.convId
      // ① 先投递：成功之前不碰卡状态（失败只需回滚执行态，不会留下「进行中没人跑」）
      var text = tbExecTaskText(t)
      var via = ''
      var queued = false
      var err = ''
      if (!convId) {
        err = '没有可用的执行会话：请在对话中打开一个会话，或在「定时」里指定执行会话'
      } else {
        try {
          var ag = ctx.agents
          if (!ag || typeof ag.followup !== 'function') {
            err = '执行通道不可用：宿主没有提供会话唤醒能力（ctx.agents.followup）'
          } else {
            var rr = ag.followup(convId, text)
            if (rr && rr.ok === false) {
              err = '会话投递被拒：' + String(rr.error || '未知原因')
            } else {
              queued = !!(rr && rr.queued)
              via = 'followup'
            }
          }
        } catch (e) {
          err = '会话投递失败：' + String((e && e.message) || e)
        }
      }
      // ② 投递失败：不留脏状态 —— running=false + lastError 可读；进行中的卡退回待办
      if (err) {
        var lf = tbLoad()
        var tf = tbFind(lf, id)
        if (!tf) return { ok: false, error: err }
        var rsf = tbExecState(tf)
        rsf.running = false
        rsf.startedAt = ''
        rsf.lastEndAt = nowIso()
        rsf.lastError = err
        rsf.sessionId = ''
        rsf.via = ''
        rsf.convSource = pick.source
        tf.updatedAt = nowIso()
        if (tbSave(lf)) tbMirror(tf)
        var restored = ''
        if (tf.status === 'in_progress') {
          var mv = tbMove(id, 'todo', { by: tbActor(a.by || 'user') })
          if (mv && mv.ok && mv.task) { tf = mv.task; restored = 'todo' }
        }
        logWarn('执行派发失败 ' + id + '：' + err)
        return { ok: false, dispatched: false, error: err, convSource: pick.source, restored: restored, task: tf }
      }
      // ③ 投递成功后才认领（待办 / 待规划 → 进行中）；此处失败不会产生「进行中没人跑」
      if (t.status === 'todo' || t.status === 'backlog') {
        var cr = tbClaim(id, tbActor(a.by || 'user'), convId)
        if (!cr.ok) {
          logWarn('执行已投递但认领失败 ' + id + '：' + cr.error)
          return { ok: false, dispatched: true, via: via, error: cr.error, convId: convId, convSource: pick.source, task: cr.task || null }
        }
      }
      // ④ 记执行态（running=true，等 taskboard_report 收尾）
      var l3 = tbLoad()
      var t3 = tbFind(l3, id)
      if (!t3) return { ok: false, dispatched: true, via: via, error: '任务库读取失败', convId: convId }
      var rs3 = tbExecState(t3)
      rs3.running = true
      rs3.startedAt = nowIso()
      rs3.startedBy = String(a.by || 'user')
      rs3.sessionId = convId
      rs3.via = via
      rs3.convSource = pick.source
      rs3.lastError = ''
      rs3.runCount = (isNum(rs3.runCount) ? rs3.runCount : 0) + 1
      if (convId) t3.execConvId = convId
      t3.updatedAt = nowIso()
      if (tbSave(l3)) tbMirror(t3)
      tbRememberConv(convId)
      return { ok: true, dispatched: true, via: via, queued: queued, convId: convId, convSource: pick.source, task: t3 }
    }
    /**
     * 定时调度扫描（插件内 60s 一次）：
     *   · 实例锁：同一时刻只允许一个进程扫描（9090/9096 共享 tasks.json 时防重复派发）
     *   · schedule 到点 → 先推进 nextAt 落盘（CAS 重读 revision），抢到的实例才执行
     *   · running 超 30 分钟无 report → 标记超时恢复，避免永久卡在「执行中」
     */
    // ── ②e 驱动模式：手动 / 全流程自动（2026-10-17 用户诉求）──────────────
    // 手动（manual，默认）：每张卡由人点「执行 / 认领 / 定时」，与原行为一致。
    // 全流程自动（auto）：60s 扫描自动把「待办」卡派发给 agent（无需点执行、无需设定时），
    //   可选「自动验收」（in_review → done，留 executions 痕迹）；并发上限 maxConcurrent。
    // 配置落 <ws>/.pair/taskboard/driver.json（工作区级，多实例共享）。
    function tbDriverLoad() {
      var raw = fsReadJson(P_TB_DRIVER) || {}
      var mode = String(raw.mode || 'manual')
      if (mode !== 'auto') mode = 'manual'
      var mc = parseInt(raw.maxConcurrent, 10)
      if (!isFinite(mc) || mc < 1) mc = 1
      if (mc > 5) mc = 5
      return {
        mode: mode,
        autoAccept: raw.autoAccept === true,
        // 规划定稿后是否直接移入「待办」（默认 false：停在「待确认」，由人点确认）
        planAutoAccept: raw.planAutoAccept === true,
        maxConcurrent: mc,
        updatedAt: String(raw.updatedAt || ''),
        updatedBy: String(raw.updatedBy || '')
      }
    }
    function tbDriverSave(patch, by) {
      var cur = tbDriverLoad()
      var p = (patch && typeof patch === 'object') ? patch : {}
      if (p.mode !== undefined) {
        var m = String(p.mode)
        if (m !== 'manual' && m !== 'auto') return { ok: false, error: '未知驱动模式：' + m + '（manual / auto）' }
        cur.mode = m
      }
      if (p.autoAccept !== undefined) cur.autoAccept = !!p.autoAccept
      if (p.planAutoAccept !== undefined) cur.planAutoAccept = !!p.planAutoAccept
      if (p.maxConcurrent !== undefined) {
        var mc = parseInt(p.maxConcurrent, 10)
        if (!isFinite(mc) || mc < 1) return { ok: false, error: '并发数需为 1-5' }
        cur.maxConcurrent = mc > 5 ? 5 : mc
      }
      cur.updatedAt = nowIso()
      cur.updatedBy = String((by && (by.name || by.sessionId || by.kind)) || 'user')
      var ok = fsWriteJson(P_TB_DRIVER, {
        version: 1, mode: cur.mode, autoAccept: cur.autoAccept, planAutoAccept: cur.planAutoAccept,
        maxConcurrent: cur.maxConcurrent,
        updatedAt: cur.updatedAt, updatedBy: cur.updatedBy
      })
      if (!ok) return { ok: false, error: '驱动配置写入失败' }
      logInfo('驱动模式 → ' + (cur.mode === 'auto'
        ? ('全流程自动（自动验收=' + (cur.autoAccept ? '开' : '关') + '，规划自动入待办=' + (cur.planAutoAccept ? '开' : '关') + '，并发=' + cur.maxConcurrent + '）')
        : '手动'))
      return { ok: true, driver: cur }
    }
    /** 自动验收：in_review → done（留 autoAcceptedAt + executions 痕迹，便于追溯是机器验收）。 */
    function tbAutoAccept(id) {
      var r = tbMove(id, 'done', { by: { kind: 'agent', name: 'auto-accept' }, autoAccept: true })
      if (!r.ok) return r
      var l = tbLoad(); var t = tbFind(l, id)
      if (!t) return r
      if (!Array.isArray(t.executions)) t.executions = []
      t.executions.push({
        id: 'x-' + Date.now() + '-' + Math.random().toString(36).slice(2, 5),
        at: nowIso(), by: { kind: 'agent', sessionId: '', name: 'auto-accept' },
        outcome: 'accepted',
        summary: '全流程自动模式自动验收（用户已在看板开启「自动验收」）',
        files: [], risks: ''
      })
      if (t.executions.length > 20) t.executions = t.executions.slice(-20)
      t.autoAcceptedAt = nowIso()
      t.updatedAt = nowIso()
      if (tbSave(l)) tbMirror(t)
      return { ok: true, task: t }
    }
    /**
     * 全流程自动扫描（在调度锁内、定时派发之后执行）：
     *   ① 自动验收：in_review 且最后一条执行记录不是 failed → done（失败卡留人工看）
     *   ② 自动派发：status=todo 且未在跑的卡，紧急度优先、创建早的先跑，
     *      最多 maxConcurrent - 正在执行数 张 → tbExecute（等价于点「执行」）
     */
    function tbAutoScan(l, now) {
      var drv = tbDriverLoad()
      if (drv.mode !== 'auto') return { mode: drv.mode, accepted: [], dispatched: [], errors: [] }
      var accepted = []
      var cands = []
      var runningCount = 0
      for (var i = 0; i < l.tasks.length; i++) {
        var t = l.tasks[i]
        if (t.trashedAt) continue
        var rs = t.runState || {}
        if (rs.running) { runningCount++; continue }
        if (drv.autoAccept && t.status === 'in_review') {
          var ex = Array.isArray(t.executions) ? t.executions : []
          var last = ex.length ? ex[ex.length - 1] : null
          if (last && String(last.outcome || 'succeeded') !== 'failed') accepted.push(t.id)
          continue
        }
        if (t.status === 'todo') cands.push(t)
      }
      var errors = []
      for (var a = 0; a < accepted.length; a++) {
        var ar = tbAutoAccept(accepted[a])
        if (!ar.ok) errors.push('验收 ' + accepted[a] + '：' + ar.error)
      }
      var rank = { urgent: 0, normal: 1, relaxed: 2 }
      cands.sort(function (x, y) {
        var rx = rank[x.urgency] === undefined ? 1 : rank[x.urgency]
        var ry = rank[y.urgency] === undefined ? 1 : rank[y.urgency]
        if (rx !== ry) return rx - ry
        var cx = new Date(String(x.createdAt || '')).getTime()
        var cy = new Date(String(y.createdAt || '')).getTime()
        if (!isFinite(cx)) cx = 0
        if (!isFinite(cy)) cy = 0
        return cx - cy
      })
      var dispatched = []
      var slots = drv.maxConcurrent - runningCount
      for (var c = 0; c < cands.length && slots > 0; c++, slots--) {
        var cr = tbExecute(cands[c].id, { by: 'agent', auto: true })
        if (cr.ok) {
          dispatched.push(cands[c].id)
          logInfo('自动驱动派发 ' + cands[c].id + '「' + String(cands[c].title || '') + '」via=' + cr.via + ' conv=' + (cr.convId || '-'))
        } else {
          errors.push('派发 ' + cands[c].id + '：' + cr.error)
          logWarn('自动驱动派发失败 ' + cands[c].id + '：' + cr.error)
        }
      }
      if (accepted.length) logInfo('自动验收 ' + accepted.join(',') + ' → 已完成')
      return { mode: 'auto', running: runningCount, accepted: accepted, dispatched: dispatched, errors: errors }
    }
    function tbScheduleScan() {
      if (!tbLockAcquire()) return { ok: true, skipped: 'lock-held', due: 0, instance: TB_INSTANCE }
      var l = tbLoad()
      var now = Date.now()
      var due = []
      for (var i = 0; i < l.tasks.length; i++) {
        var t = l.tasks[i]
        if (t.trashedAt) continue
        var mt = t.runState
        if (mt && mt.running) {
          var st = new Date(String(mt.startedAt || '')).getTime()
          if (isFinite(st) && now - st > 30 * 60000) {
            mt.running = false
            mt.lastEndAt = nowIso()
            mt.lastError = '执行超时（30 分钟未收到 taskboard_report）'
            t.updatedAt = nowIso()
            tbMirror(t)
          }
        }
        var sch = t.schedule
        if (!sch || !sch.enabled || !sch.nextAt) continue
        if (t.runState && t.runState.running) continue
        var nx = new Date(String(sch.nextAt)).getTime()
        if (!isFinite(nx) || nx > now) continue
        // ★ 先推进 nextAt（立刻落盘）：即使两个实例同时扫到，也只有先写者能保留这个推进
        due.push({ id: t.id, convId: String(sch.convId || '') })
        sch.lastAt = nowIso()
        if (sch.mode === 'at') { sch.enabled = false; sch.nextAt = '' }
        else sch.nextAt = tbNextAt(sch, new Date(now))
        t.updatedAt = nowIso()
      }
      if (due.length) {
        // CAS：写前重读文件，revision 变了说明别的实例已写 → 放弃本轮（下一轮按新 nextAt 重扫）
        var chk = tbLoad()
        if (isNum(l.revision) && chk.revision !== l.revision) {
          return { ok: true, skipped: 'concurrent-write', due: 0, instance: TB_INSTANCE }
        }
        if (!tbSave(l)) return { ok: false, error: '任务库写入失败', due: 0, instance: TB_INSTANCE }
      }
      for (var d = 0; d < due.length; d++) {
        var r = tbExecute(due[d].id, { by: 'agent', convId: due[d].convId })
        var l2 = tbLoad()
        var t2 = tbFind(l2, due[d].id)
        if (!t2 || !t2.schedule) continue
        t2.schedule.lastError = r.ok ? '' : String(r.error || '')
        t2.updatedAt = nowIso()
        if (tbSave(l2)) tbMirror(t2)
        logInfo('定时执行 ' + due[d].id + '：' + (r.ok ? ('已派发 via=' + r.via + ' conv=' + (r.convId || '-')) : ('失败 ' + r.error)))
      }
      // 全流程自动：定时派发之后，自动派发待办卡 + 可选自动验收（仍在本轮扫描锁内）
      var autoRes = null
      try { autoRes = tbAutoScan(tbLoad(), now) } catch (eA) { logWarn('自动驱动扫描失败: ' + eA) }
      return { ok: true, due: due.length, instance: TB_INSTANCE, auto: autoRes }
    }
    /** 任务看板状态快照（client 半渲染用；带流程阶段进度）。 */
    function tbSnapshot(q) {
      var qq = (q && typeof q === 'object') ? q : {}
      var l = tbLoad()
      var want = String(qq.workspace || '')
      var rows = []
      for (var i = 0; i < l.tasks.length; i++) {
        var t = l.tasks[i]
        if (t.trashedAt) continue
        if (want && t.workspace !== want) continue
        rows.push(t)
      }
      // 绑定了流程的任务：带上阶段进度（卡片上显示「流程名 · x/y」）
      var runs = listRuns ? listRuns('all') : []
      var byRun = {}
      for (var r = 0; r < runs.length; r++) byRun[runs[r].runId] = runs[r]
      for (var k = 0; k < rows.length; k++) {
        var row = rows[k]
        row.flowProgress = null
        var run = row.runId ? byRun[row.runId] : null
        if (run) {
          var sp = stageProgress(run)
          var rp = runProgress(run)
          row.flowProgress = { runId: run.runId, status: run.status, label: sp.label, percent: rp.percent, stagesDone: rp.stagesDone, stagesTotal: rp.stagesTotal }
        } else if (row.flowId) {
          row.flowProgress = { runId: '', status: 'idle', label: '未启动', percent: 0 }
        }
      }
      return { ok: true, at: nowIso(), revision: l.revision, workspace: want || tbWorkspaceId(), tasks: rows, flowLibrary: allFlows(), driver: tbDriverLoad() }
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

    // ── ⑥b 子任务（subtasks：模板骨架 + agent 运行时细化）────────
    // 语义：流程定义 stage.plan（兼容旧 stage.tasks）只是「子任务模板骨架」，
    //   进入阶段时 seed 成 run.stages[].subtasks —— 这是子任务的唯一真相；
    //   agent 在运行时用 flow_task 工具增删改（source='agent'），把粗阶段拆细。
    // 镜像：每条子任务同步建/更宿主任务（.pair/tasks/<id>.json）→ 右栏任务面板可见。
    function subStatusOf(s) { return String((s && s.status) || 'pending') }
    function nextSubId(stageRun) {
      var max = 0
      var list = (stageRun && stageRun.subtasks) || []
      for (var i = 0; i < list.length; i++) {
        var m = /^s(\d+)$/.exec(String((list[i] && list[i].id) || ''))
        if (m) { var n = parseInt(m[1], 10); if (n > max) max = n }
      }
      return 's' + (max + 1)
    }
    function subMirrorStatus(st) {
      var s = subStatusOf(st)
      if (s === 'completed') return 'completed'
      if (s === 'in_progress') return 'in_progress'
      return 'pending'
    }
    function subMirror(run, stageDef, stageRun, st) {
      try {
        if (!st.hostTaskId) {
          var title = (stageDef && stageDef.title) || (stageRun && stageRun.id) || '阶段'
          st.hostTaskId = createTask('[' + title + '] ' + st.subject,
            '流程 ' + run.flowId + ' · 阶段 ' + title + ' · 子任务 ' + st.id + (st.detail ? ('：' + st.detail) : ''),
            run.convId)
          stageRun.taskIds = (stageRun.taskIds || []).concat([st.hostTaskId])
        } else {
          updateTaskStatus(st.hostTaskId, subMirrorStatus(st))
        }
      } catch (e) { logWarn('子任务镜像宿主任务失败 ' + st.id + ': ' + e) }
    }
    function seedSubtasks(run, stageDef, stageRun) {
      if (!stageRun.subtasks) stageRun.subtasks = []
      if (stageRun.seeded) return []                       // 幂等：同一阶段只播一次种
      stageRun.seeded = true
      var plan = []
      if (stageDef && Array.isArray(stageDef.plan) && stageDef.plan.length) plan = stageDef.plan
      else if (stageDef && Array.isArray(stageDef.tasks)) plan = stageDef.tasks
      var out = []
      for (var i = 0; i < plan.length; i++) {
        var raw = plan[i]
        var subject = (raw && typeof raw === 'object') ? String(raw.subject || raw.title || '') : String(raw || '')
        subject = subject.replace(/^\s+|\s+$/g, '')
        if (!subject) continue
        var st = {
          id: nextSubId(stageRun), subject: subject,
          detail: (raw && typeof raw === 'object') ? String(raw.detail || '') : '',
          status: 'pending', source: 'plan', hostTaskId: '',
          note: '', evidence: '',
          createdAt: nowIso(), startedAt: '', completedAt: '',
        }
        stageRun.subtasks.push(st)
        subMirror(run, stageDef, stageRun, st)
        out.push(st.id)
      }
      return out
    }
    function subStats(stageRun) {
      var list = (stageRun && stageRun.subtasks) || []
      var o = { total: list.length, done: 0, doing: 0, blocked: 0, cancelled: 0, pending: 0 }
      for (var i = 0; i < list.length; i++) {
        var s = subStatusOf(list[i])
        if (s === 'completed') o.done++
        else if (s === 'in_progress') o.doing++
        else if (s === 'blocked') o.blocked++
        else if (s === 'cancelled') o.cancelled++
        else o.pending++
      }
      o.active = o.total - o.cancelled
      o.label = o.done + '/' + o.active
      o.percent = o.active ? Math.round((o.done / o.active) * 100) : (o.total ? 100 : 0)
      return o
    }
    function runSubStats(run) {
      var list = (run && run.stages) || []
      var o = { total: 0, done: 0, doing: 0, blocked: 0, cancelled: 0, pending: 0 }
      for (var i = 0; i < list.length; i++) {
        var s = subStats(list[i])
        o.total += s.total; o.done += s.done; o.doing += s.doing
        o.blocked += s.blocked; o.cancelled += s.cancelled; o.pending += s.pending
      }
      o.active = o.total - o.cancelled
      o.label = o.done + '/' + o.active
      o.percent = o.active ? Math.round((o.done / o.active) * 100) : (o.total ? 100 : 0)
      return o
    }
    // 「此刻在做什么」：优先进行中，其次待办，再次阻塞
    function focusSubtask(run) {
      var list = (run && run.stages) || []
      var st = list[stageIdx(run)] || null
      var subs = (st && st.subtasks) || []
      for (var i = 0; i < subs.length; i++) if (subStatusOf(subs[i]) === 'in_progress') return subs[i]
      for (var j = 0; j < subs.length; j++) if (subStatusOf(subs[j]) === 'pending') return subs[j]
      for (var k = 0; k < subs.length; k++) if (subStatusOf(subs[k]) === 'blocked') return subs[k]
      return null
    }
    // 总进度：阶段为主权重 + 当前阶段子任务细化（0-100）
    function runProgress(run) {
      var list = (run && run.stages) || []
      var total = list.length || 1
      var idx = stageIdx(run)
      var cur = list[idx] || null
      var stagesDone = 0
      for (var i = 0; i < list.length; i++) if (list[i].status === 'completed') stagesDone++
      var curStats = subStats(cur)
      var frac = 0
      if (cur && cur.status !== 'completed' && curStats.active > 0) frac = curStats.done / curStats.active
      var percent = Math.round(((stagesDone + (stagesDone < total ? frac : 0)) / total) * 100)
      if (percent > 100) percent = 100
      if (percent < 0) percent = 0
      if (run && run.status === 'completed') percent = 100
      return {
        stagesDone: stagesDone, stagesTotal: total,
        stageIdx: idx, stageLabel: Math.min(idx + 1, total) + '/' + total,
        currentStage: cur ? cur.id : '',
        currentSub: curStats,
        subs: runSubStats(run),
        percent: percent,
      }
    }
    function resolveStageRef(run, ref) {
      var list = (run && run.stages) || []
      if (ref === undefined || ref === null || ref === '') return stageIdx(run)
      if (isNum(ref)) { var n = Number(ref); return (n >= 0 && n < list.length) ? n : -1 }
      var str = String(ref)
      for (var i = 0; i < list.length; i++) if (String(list[i].id) === str) return i
      return -1
    }
    function subtaskText(stageRun) {
      var subs = (stageRun && stageRun.subtasks) || []
      if (!subs.length) return ''
      var marks = { completed: '[x]', in_progress: '[>]', blocked: '[!]', cancelled: '[-]', pending: '[ ]' }
      return subs.map(function (s) {
        return (marks[subStatusOf(s)] || '[ ]') + ' ' + s.id + ' ' + s.subject + (s.note ? ('（' + s.note + '）') : '')
      }).join('\n')
    }
    // 子任务操作（flow_task 工具与看板 RPC 共用；run.stages[].subtasks 为唯一真相）
    function taskAction(o) {
      var a = o || {}
      var op = String(a.op || 'list')
      var run = pickRun(a)
      if (!run) return { ok: false, error: '没有可用的流程实例（flow_start 启动）' }
      var si = resolveStageRef(run, a.stage)
      if (si < 0) return { ok: false, error: '阶段不存在: ' + String(a.stage) }
      var stageRun = run.stages[si]
      if (!stageRun.subtasks) stageRun.subtasks = []
      var flow = runFlow(run)
      var stageDef = (flow && flow.def && flow.def.stages) ? flow.def.stages[si] : null
      var stageTitle = (stageDef && stageDef.title) || stageRun.id
      if (op === 'list') {
        var st0 = subStats(stageRun)
        return {
          ok: true, runId: run.runId, stage: stageRun.id, subtasks: stageRun.subtasks, stats: st0,
          text: '阶段「' + stageTitle + '」子任务 ' + st0.label + '（进行中 ' + st0.doing + ' · 待办 ' + st0.pending +
            ' · 阻塞 ' + st0.blocked + ' · 已取消 ' + st0.cancelled + '）：\n' +
            (subtaskText(stageRun) || '（无子任务 —— 用 op=add 拆解）'),
        }
      }
      if (run.status === 'archived' || run.status === 'deleted') {
        return { ok: false, error: '实例已' + run.status + '（只读留档）——先恢复再改子任务' }
      }
      if (op === 'add') {
        var subject = String(a.subject || '').replace(/^\s+|\s+$/g, '')
        if (!subject) return { ok: false, error: 'subject 必填（子任务标题）' }
        var ns0 = String(a.status || 'pending')
        if (SUB_STATUS.indexOf(ns0) < 0) ns0 = 'pending'
        var st = {
          id: nextSubId(stageRun), subject: subject, detail: String(a.detail || ''),
          status: ns0, source: 'agent', hostTaskId: '',
          note: String(a.note || ''), evidence: '',
          createdAt: nowIso(), startedAt: ns0 === 'in_progress' ? nowIso() : '', completedAt: '',
        }
        stageRun.subtasks.push(st)
        subMirror(run, stageDef, stageRun, st)
        saveRun(run)
        var s1 = subStats(stageRun)
        return { ok: true, subtask: st, stats: s1, text: '已拆解子任务 ' + st.id + '：' + st.subject + '\n阶段「' + stageTitle + '」进度 ' + s1.label }
      }
      if (op === 'update') {
        var id = String(a.id || a.subId || '')
        var target = null
        for (var i = 0; i < stageRun.subtasks.length; i++) if (String(stageRun.subtasks[i].id) === id) target = stageRun.subtasks[i]
        if (!target) return {
          ok: false,
          error: '子任务不存在: ' + id + '（现有: ' + (stageRun.subtasks.map(function (s) { return s.id }).join(', ') || '无') + '）',
        }
        if (a.status !== undefined) {
          var ns = String(a.status)
          if (SUB_STATUS.indexOf(ns) < 0) return { ok: false, error: 'status 非法（可选 ' + SUB_STATUS.join(' | ') + '）' }
          target.status = ns
          if (ns === 'in_progress' && !target.startedAt) target.startedAt = nowIso()
          if (ns === 'completed') target.completedAt = nowIso()
          else target.completedAt = ''
        }
        if (a.subject !== undefined) target.subject = String(a.subject)
        if (a.detail !== undefined) target.detail = String(a.detail)
        if (a.note !== undefined) target.note = String(a.note)
        if (a.evidence !== undefined) target.evidence = String(a.evidence)
        subMirror(run, stageDef, stageRun, target)
        saveRun(run)
        var s2 = subStats(stageRun)
        var all2 = runSubStats(run)
        return {
          ok: true, subtask: target, stats: s2, runStats: all2,
          text: '子任务 ' + target.id + ' → ' + target.status + '：' + target.subject +
            '\n阶段「' + stageTitle + '」' + s2.label + ' · 全流程 ' + all2.label,
        }
      }
      if (op === 'remove') {
        var id2 = String(a.id || a.subId || '')
        var idx2 = -1
        for (var j = 0; j < stageRun.subtasks.length; j++) if (String(stageRun.subtasks[j].id) === id2) idx2 = j
        if (idx2 < 0) return { ok: false, error: '子任务不存在: ' + id2 }
        var gone = stageRun.subtasks.splice(idx2, 1)[0]
        if (gone.hostTaskId) { try { updateTaskStatus(gone.hostTaskId, 'cancelled') } catch (e2) { } }
        saveRun(run)
        return { ok: true, removed: gone, text: '已移除子任务 ' + id2 + '：' + gone.subject }
      }
      return { ok: false, error: '未知 op（支持 list | add | update | remove）' }
    }

    // ── ⑦ 流程实例 / 门禁 / 证据 / 推进 ──────────────────────────
    function runPath(runId) { return P_RUNS + '/' + safeName(runId) + '.json' }
    function loadRun(runId) { return fsReadJson(runPath(runId)) }
    function saveRun(run) { run.updatedAt = nowIso(); fsWriteJson(runPath(run.runId), run); return run }
    // 归档/删除 = 只读留档：默认从活跃视图隐藏（只在「运行记录」里出现）；'all' 取全部
    function runHidden(r) { return r.status === 'archived' || r.status === 'deleted' }
    function listRuns(scope) {
      var out = fsListNames(P_RUNS).filter(function (x) { return /\.json$/.test(x) })
        .map(function (x) { return fsReadJson(P_RUNS + '/' + x) })
        .filter(function (r) { return r && r.runId })
      if (String(scope || '') !== 'all') out = out.filter(function (r) { return !runHidden(r) })
      return out.sort(function (a, b) { return String(a.updatedAt || '') < String(b.updatedAt || '') ? -1 : 1 })
    }
    function activeRuns() { return listRuns().filter(function (r) { return r.status === 'active' }) }
    function pickRun(opts) {
      var o = opts || {}
      if (o.runId) return loadRun(o.runId)
      var act = activeRuns()
      if (o.convId) {
        var mine = act.filter(function (r) { return r.convId === o.convId })
        if (mine.length) return mine[mine.length - 1]
        // ★ 显式指定会话却没有活跃实例时**不越界**到别的会话/项目（否则对已归档实例的
        //   操作会打到别人身上）；退回该会话的任意实例（含归档/删除），由调用方按状态给准确提示。
        var anyMine = listRuns('all').filter(function (r) { return r.convId === o.convId })
        return anyMine.length ? anyMine[anyMine.length - 1] : null
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
      var allDone = run.status === 'completed'
      return {
        idx: idx, total: list.length,
        label: allDone
          ? ('全部完成 ' + list.length + '/' + list.length)
          : ((idx + 1) + '/' + list.length + (def ? ' ' + def.title : '')),
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
      // 任务绑定：taskId/boardTaskId 必须真实存在（不存在就明确报错，不静默成功）
      var taskId = String(o.taskId || o.boardTaskId || '')
      if (taskId && !tbFind(tbLoad(), taskId)) {
        return { ok: false, error: '看板任务不存在：' + taskId + '（taskId 须为任务看板上的卡 id，如 t-…；先用 taskboard_list 查）' }
      }
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
      seedSubtasks(run, defStages[0], stages[0])
      saveRun(run)
      // 绑定看板卡（唯一入口）：启动成功即回写 runId；卡未设流程时补 flowId 并生成 DoD 清单
      var boundTask = null
      if (taskId) {
        var bRes = tbBindRun(taskId, runId, { flowId: run.flowId, flowScope: run.flowScope, flowVersion: run.flowVersion })
        if (!bRes.ok) return { ok: false, error: '流程已启动但绑定看板任务失败：' + bRes.error + '（实例 ' + runId + '）' }
        boundTask = bRes.task
      }
      return { ok: true, run: run, source: picked.source, flowScope: flow.scope, task: boundTask }
    }

    // 推进（含门禁记录）
    function advanceRun(opts) {
      var o = opts || {}
      var run = pickRun(o)
      if (!run) return { ok: false, error: '没有可用的流程实例（用 flow_start 启动）' }
      if (run.status !== 'active') {
        var why = '流程实例不是进行中状态（' + run.status + '）'
        if (run.status === 'interrupted') why += '——用 resume 恢复后继续'
        else if (run.status === 'archived') why += '——先取消归档（unarchive）'
        else if (run.status === 'deleted') why += '——先恢复删除（restore）'
        else if (run.status === 'completed') why += '——实例已完成'
        return { ok: false, error: why, run: run }
      }
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
            ok: true, advanced: false, gate: gateRec, run: run,
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
          ok: true, advanced: false, gate: gateRec, run: run,
          pending: pending.map(function (g) { return { id: g.id, kind: g.kind, text: g.text, cmd: g.cmd || '', artifact: g.artifact || '' } }),
          text: '阶段「' + stageDef.title + '」还有 ' + pending.length + ' 个必过门禁未通过，不能推进：\n' +
            pending.map(function (g) { return '  · ' + g.id + '（' + g.kind + '）' + g.text + (g.cmd ? ' | cmd: ' + g.cmd : (g.artifact ? ' | artifact: ' + g.artifact : '')) }).join('\n') +
            '\n处置：check 门禁用 flow_advance {gateId, evidence}；command 门禁用 flow_advance {gateId} 代跑；artifact 门禁确认产物后 flow_advance {gateId}；approval 门禁必须用户批准（approve=true）。',
        }
      }

      // ③ 完成本阶段 → 进入下一阶段
      // 子任务保留真实状态（未完成的**不**代填 completed）：进度要可信，遗留要看见
      var stDone = subStats(stageRun)
      var leftOver = stDone.active - stDone.done
      stageRun.status = 'completed'
      stageRun.completedAt = nowIso()
      completeStageTasks(stageRun)
      var nextIdx = idx + 1
      if (nextIdx >= run.stages.length) {
        run.status = 'completed'
        run.currentStage = ''
        saveRun(run)
        var allDone = runSubStats(run)
        return {
          ok: true, advanced: true, completed: true, gate: gateRec, run: run,
          text: '流程「' + flow.def.title + '」全部阶段已完成 [全部通过]（run ' + run.runId + '）' +
            '\n子任务合计 ' + allDone.label + ' · 总进度 100%' +
            (leftOver > 0 ? ('\n[注意] 末阶段还有 ' + leftOver + ' 个子任务未完成（保留真实状态）。') : ''),
        }
      }
      var nextRun = run.stages[nextIdx]
      var nextDef = flow.def.stages[nextIdx]
      nextRun.status = 'in_progress'
      nextRun.startedAt = nowIso()
      run.currentStage = nextDef.id
      seedSubtasks(run, nextDef, nextRun)
      saveRun(run)
      return {
        ok: true, advanced: true, stage: nextDef.id, title: nextDef.title, gate: gateRec, run: run,
        text: '阶段 ' + (idx + 1) + '/' + run.stages.length + '「' + stageDef.title + '」已完成 → 进入阶段 ' + (nextIdx + 1) + '「' + nextDef.title + '」' +
          (nextDef.goal ? '\n阶段目标：' + nextDef.goal : '') +
          '\n门禁：' + ((nextDef.gates || []).map(function (g) { return g.id + '(' + g.kind + ')' }).join('、') || '无') +
          '\n子任务骨架 ' + subStats(nextRun).label + '（已镜像到右栏任务面板；用 flow_task add/update 可细化与推进）：\n' +
          (subtaskText(nextRun) || '（本阶段无模板子任务 —— 可用 flow_task 自行拆解）') +
          (leftOver > 0 ? ('\n[注意] 上一阶段还有 ' + leftOver + ' 个子任务未完成（保留真实状态，未自动标完成）。') : ''),
      }
    }
    // ── ⑦b 实例生命周期（中断 / 恢复 / 归档 / 取消归档 / 删除 / 恢复删除 / 彻底删除）──
    // 语义：中断=可恢复暂停（进度全留）；归档=移出活跃列表的只读留档；
    //       删除=软删（默认隐藏、可恢复）；彻底删除=物理删 run 与证据（需 confirm=true）。
    function lifecycleRun(opName, o) {
      var a = o || {}
      var run = a.runId ? loadRun(String(a.runId)) : pickRun(a)
      if (!run) return { ok: false, error: '实例不存在' + (a.runId ? (': ' + a.runId) : '（没有活跃实例）') }
      var prev = String(run.status || 'active')
      var pg = runProgress(run)
      if (opName === 'interrupt') {
        if (prev !== 'active') return { ok: false, error: '只有进行中的实例可中断（当前 ' + prev + '）', run: run }
        run.status = 'interrupted'; run.statusBefore = prev
        run.interruptedAt = nowIso(); run.interruptReason = String(a.reason || '')
        saveRun(run)
        return {
          ok: true, run: run,
          text: '流程已中断（进度保留，可 resume 恢复）：' + run.runId +
            '\n停在 阶段 ' + pg.stageLabel + ' · 子任务 ' + pg.subs.label + ' · 总进度 ' + pg.percent + '%' +
            (a.reason ? ('\n原因：' + a.reason) : ''),
        }
      }
      if (opName === 'resume') {
        if (prev !== 'interrupted') return { ok: false, error: '只有已中断的实例可恢复（当前 ' + prev + '）', run: run }
        run.status = 'active'; run.resumedAt = nowIso()
        saveRun(run)
        return { ok: true, run: run, text: '流程已恢复运行：' + run.runId + '\n继续 阶段 ' + pg.stageLabel + '（子任务 ' + pg.subs.label + '）' }
      }
      if (opName === 'archive') {
        if (prev === 'archived') return { ok: false, error: '该实例已归档', run: run }
        if (prev === 'deleted') return { ok: false, error: '该实例已删除 —— 先「恢复」再归档', run: run }
        run.status = 'archived'; run.statusBefore = prev; run.archivedAt = nowIso()
        saveRun(run)
        return { ok: true, run: run, text: '已归档（移出活跃列表，记录与证据保留，只读）：' + run.runId + '（原状态 ' + prev + '）' }
      }
      if (opName === 'unarchive') {
        if (prev !== 'archived') return { ok: false, error: '该实例未归档（当前 ' + prev + '）', run: run }
        run.status = String(run.statusBefore || 'interrupted'); run.archivedAt = ''
        saveRun(run)
        return { ok: true, run: run, text: '已取消归档：' + run.runId + ' → 状态 ' + run.status }
      }
      if (opName === 'delete') {
        if (prev === 'deleted') return { ok: false, error: '该实例已是删除状态', run: run }
        run.status = 'deleted'; run.statusBefore = prev; run.deletedAt = nowIso()
        saveRun(run)
        return { ok: true, run: run, text: '已删除记录（软删：默认从流程视图隐藏，可在「运行记录」里恢复）：' + run.runId }
      }
      if (opName === 'restore') {
        if (prev !== 'deleted') return { ok: false, error: '该实例不是删除状态（当前 ' + prev + '）', run: run }
        run.status = String(run.statusBefore || 'interrupted'); run.deletedAt = ''
        saveRun(run)
        return { ok: true, run: run, text: '已恢复删除的记录：' + run.runId + ' → 状态 ' + run.status }
      }
      if (opName === 'purge') {
        if (a.confirm !== true) return { ok: false, error: '彻底删除需 confirm=true（物理删除实例与证据文件，不可恢复）', run: run }
        var gone = []
        var fail = []
        var rp = runPath(run.runId)
        try { ctx.fs.rm(rp, false); gone.push(rp) } catch (e) { fail.push(rp + ': ' + e) }
        var ed = String(run.evidenceDir || '').replace(/\/+$/, '') || (P_EVID + '/' + safeName(run.runId))
        if (fsExists(ed)) { try { ctx.fs.rm(ed, true); gone.push(ed) } catch (e2) { fail.push(ed + ': ' + e2) } }
        return {
          ok: !fail.length, run: null, removed: gone, errors: fail,
          text: '已彻底删除实例 ' + run.runId + '\n删除：' + (gone.join('\n') || '（无）') +
            (fail.length ? ('\n失败：' + fail.join('；')) : '') +
            '\n（宿主任务文件 .pair/tasks/<id>.json 保留为历史，不影响流程真相）',
        }
      }
      return { ok: false, error: '未知生命周期操作: ' + opName + '（支持 interrupt | resume | archive | unarchive | delete | restore | purge）' }
    }
    // 运行记录（含归档/删除；「运行记录」页签与 flow_status all 用）
    function runsRecord() {
      var all = listRuns('all')
      var out = []
      for (var i = all.length - 1; i >= 0; i--) {
        var r = all[i]
        var pg = runProgress(r)
        var flow = runFlow(r)
        out.push({
          runId: r.runId, flowId: r.flowId, title: flow ? (flow.def.title || r.flowId) : r.flowId,
          status: r.status, statusBefore: r.statusBefore || '',
          project: r.project || '', convId: r.convId || '',
          stageLabel: pg.stageLabel, currentStage: pg.currentStage,
          stagesDone: pg.stagesDone, stagesTotal: pg.stagesTotal,
          subs: pg.subs, percent: pg.percent,
          createdAt: r.createdAt || '', updatedAt: r.updatedAt || '',
          archivedAt: r.archivedAt || '', deletedAt: r.deletedAt || '',
          interruptedAt: r.interruptedAt || '', interruptReason: r.interruptReason || '',
          gitHead: r.gitHead || '', evidenceDir: r.evidenceDir || '',
        })
      }
      return out
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
      var pg = runProgress(run)
      var lines = []
      lines.push('流程 ' + run.flowId + (flow ? '「' + flow.def.title + '」' : '') + '（' + (run.flowScope || '?') + ' 级 v' + run.flowVersion + '）· 状态 ' + run.status)
      lines.push('进度：阶段 ' + sp.label + ' · 子任务 ' + pg.subs.label + ' · 总进度 ' + pg.percent + '%' + (run.convId ? ' · 会话 ' + run.convId : ''))
      if (run.status === 'interrupted') lines.push('已中断' + (run.interruptedAt ? ('于 ' + run.interruptedAt) : '') + (run.interruptReason ? ('：' + run.interruptReason) : '') + '（flow_lifecycle resume 恢复）')
      if (run.status === 'archived') lines.push('已归档（只读留档）' + (run.archivedAt ? ('于 ' + run.archivedAt) : '') + '（flow_lifecycle unarchive 取消归档）')
      if (run.status === 'deleted') lines.push('已删除（软删）' + (run.deletedAt ? ('于 ' + run.deletedAt) : '') + '（flow_lifecycle restore 恢复）')
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
        var ss = subStats(st)
        var subTxt = ss.total
          ? ('  <子任务 ' + ss.label + (ss.doing ? (' · 进行中' + ss.doing) : '') + (ss.blocked ? (' · 阻塞' + ss.blocked) : '') + '>')
          : '  <无子任务>'
        lines.push('  ' + (st.status === 'completed' ? '[x]' : (st.status === 'in_progress' ? '[>]' : '[ ]')) + ' ' + (i + 1) + '. ' + (def ? def.title : st.id) + (marks.length ? '  ' + marks.join(' ') : '') + subTxt)
        // 当前阶段展开子任务明细（一眼看清「此刻在做哪一条」）
        if (i === pg.stageIdx && (st.subtasks || []).length) {
          var subs0 = st.subtasks
          for (var j = 0; j < subs0.length; j++) {
            var sb0 = subs0[j]
            var mk0 = sb0.status === 'completed' ? '[x]' : (sb0.status === 'in_progress' ? '[>]' : (sb0.status === 'blocked' ? '[!]' : (sb0.status === 'cancelled' ? '[-]' : '[ ]')))
            lines.push('        ' + mk0 + ' ' + sb0.id + ' ' + sb0.subject + (sb0.note ? ('（' + sb0.note + '）') : '') + (sb0.source === 'agent' ? ' [agent 拆解]' : ''))
          }
        }
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
        var planL = Array.isArray(st.plan) ? st.plan : (Array.isArray(st.tasks) ? st.tasks : [])
        if (planL.length) L.push('- 子任务模板：' + planL.map(function (x) { return (x && typeof x === 'object') ? String(x.subject || x.title || '') : String(x) }).join('；'))
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
        var pg = runProgress(run)
        var stageSubs = subStats(stageRun)
        var s = '[流程] ' + run.flowId + (flow ? '「' + flow.def.title + '」' : '') + '（' + (run.flowScope || '?') + ' 级 v' + run.flowVersion + '）· 阶段 ' + sp.label + ' · 总进度 ' + pg.percent + '%'
        if (run.status === 'interrupted') s += '\n[注意] 该实例已中断' + (run.interruptReason ? ('（' + run.interruptReason + '）') : '') + ' —— flow_lifecycle {op:"resume"} 可恢复。'
        if (stageDef && stageDef.goal) s += '\n阶段目标：' + stageDef.goal
        // 子任务闭环（本流程模型的核心执行面）：拆解 → 推进 → 收口，进度实时可见
        if (stageSubs.total) {
          s += '\n子任务 ' + stageSubs.label + (stageSubs.doing ? ('（进行中 ' + stageSubs.doing + '）') : '') + (stageSubs.blocked ? ('（阻塞 ' + stageSubs.blocked + '）') : '') + '：'
          var subs1 = stageRun.subtasks || []
          for (var si = 0; si < subs1.length; si++) {
            var sb = subs1[si]
            var mk = sb.status === 'completed' ? '[x]' : (sb.status === 'in_progress' ? '[>]' : (sb.status === 'blocked' ? '[!]' : (sb.status === 'cancelled' ? '[-]' : '[ ]')))
            s += '\n  ' + mk + ' ' + sb.id + ' ' + sb.subject
          }
          s += '\n维护方式：flow_task {op:"add",subject:"…"} 拆解 / {op:"update",id:"s1",status:"in_progress|completed"} 推进 / {op:"list"} 查看。'
        } else {
          s += '\n本阶段暂无子任务 —— 建议先用 flow_task {op:"add",subject:"…"} 把阶段拆成可执行的小步（拆解后本提示会实时带上进度）。'
        }
        if (pending.length) {
          s += '\n未过门禁：' + pending.map(function (g) { return g.id + '（' + g.text + '，kind=' + g.kind + '）' }).join('；')
          s += '\n下一步：先把门禁做实（command 门禁可 flow_advance {gateId} 代跑并留证；check/artifact 需给出证据；approval 必须用户批准），再 flow_advance 推进阶段。'
        } else {
          s += '\n本阶段必过门禁已全部通过 → 可用 flow_advance 推进到下一阶段。'
        }
        return s
      } catch (e) { return '' }
    }
    // ★ 2026-10-17 使用面收窄：会话侧不再注入流程状态（流程只在任务看板里使用）。
    if (CONVERSATION_FACING) {
      try {
        ctx.systemPrompt.section({ name: 'flow', order: 95, text: '{{paircode_flow_state}}', alwaysVisible: true })
        ctx.systemPrompt.variable({ name: 'paircode_flow_state', provider: function () { return flowPromptText() } })
      } catch (e) { logWarn('systemPrompt 注册失败: ' + e) }
    }

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
            var planL2 = Array.isArray(st.plan) ? st.plan : (Array.isArray(st.tasks) ? st.tasks : [])
            if (planL2.length) lines.push('   子任务模板：' + planL2.map(function (x) { return (x && typeof x === 'object') ? String(x.subject || x.title || '') : String(x) }).join('；'))
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
            taskId: { type: 'string', description: '可选：绑定任务看板上的卡 id（如 t-…）——启动成功后回写 runId；卡未设流程时自动补 flowId/flowScope/flowVersion 并生成验收清单。id 不存在会明确报错' },
            boardTaskId: { type: 'string', description: '可选：taskId 的别名（同义）' },
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
              (r.task ? ('\n已绑定看板任务：' + r.task.id + '（' + String(r.task.title || '') + '）——门禁通过会自动勾选验收清单，流程走完转「待验收」') : '') +
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
          // 流程门禁 → 看板任务（DoD 回填 / 全部走完转「待验收」）：与 flow_action RPC 完全同语义
          try { if (r.run) tbSyncFromRun(r.run) } catch (e) { logWarn('任务同步失败: ' + e) }
          var extra = ''
          if (r.rollback) extra = '\n\n[注意] 回滚建议（插件不自动执行，高风险命令必须用户批准）：\n' + r.rollback.rollbackOptions.map(function (o) { return '  · [' + o.risk + '] ' + o.text + ' → ' + o.cmd }).join('\n') + (r.rollback.note ? ('\n  ' + r.rollback.note) : '')
          return { text: r.text + extra, result: r }
        },
      },
      {
        name: 'flow_task',
        description: '阶段子任务：把当前阶段拆成可执行的小步并实时推进进度（拆解 → 推进 → 收口）。op=list 列出（默认）；add 新增（subject 必填）；update 改状态/备注/证据（id 形如 s1）；remove 移除。子任务状态是看板与每轮提示里的进度真相（cancelled 不计入进度口径）。',
        parameters: {
          type: 'object',
          properties: {
            op: { type: 'string', description: 'list | add | update | remove（默认 list）' },
            subject: { type: 'string', description: 'add：子任务标题（必填）' },
            detail: { type: 'string', description: '可选：细化说明（做到什么算完成）' },
            id: { type: 'string', description: 'update/remove：子任务 id（如 s1）' },
            status: { type: 'string', description: 'update：pending | in_progress | completed | blocked | cancelled' },
            note: { type: 'string', description: '可选：备注（如卡在哪里、为什么阻塞）' },
            evidence: { type: 'string', description: '可选：完成证据（文件路径 / 命令 / 结论）' },
            stage: { type: 'string', description: '可选：阶段 id 或索引（缺省=当前阶段）' },
            runId: { type: 'string', description: '可选：指定流程实例' },
            convId: { type: 'string', description: '可选：按会话定位实例' },
          },
        },
        execute: function (args) {
          var r = taskAction(args || {})
          if (!r.ok) return { text: r.error, error: r.error }
          return { text: r.text, subtasks: r.subtasks || null, subtask: r.subtask || null, stats: r.stats || null, runStats: r.runStats || null }
        },
      },
      {
        name: 'flow_lifecycle',
        description: '流程实例生命周期：interrupt 中断（可恢复暂停，进度与证据全留）/ resume 恢复运行 / archive 归档（移出活跃列表、只读留档）/ unarchive 取消归档 / delete 删除记录（软删，可恢复）/ restore 恢复删除 / purge 彻底删除（物理删实例与证据文件，需 confirm=true，不可恢复）。',
        parameters: {
          type: 'object',
          properties: {
            op: { type: 'string', description: 'interrupt | resume | archive | unarchive | delete | restore | purge' },
            reason: { type: 'string', description: '可选：中断原因（interrupt）' },
            confirm: { type: 'boolean', description: 'purge 必填 true（确认物理删除，不可恢复）' },
            runId: { type: 'string', description: '可选：指定实例（缺省取会话/最近活跃实例；已归档/已删除的实例必须显式给 runId）' },
            convId: { type: 'string', description: '可选：按会话定位实例' },
          },
          required: ['op'],
        },
        execute: function (args) {
          var a2 = args || {}
          var rl2 = lifecycleRun(String(a2.op || ''), a2)
          if (!rl2.ok) return { text: rl2.error, error: rl2.error, run: rl2.run || null }
          return { text: rl2.text, run: rl2.run || null, removed: rl2.removed || null }
        },
      },
    ]
    // ★ 2026-10-17 使用面收窄：flow_* 工具不再注册进 agent 工具面（流程只在任务看板里使用）。
    //   看板的所有写操作走 ui.invoke → flow_action / taskboard_action RPC，不依赖这里的注册。
    if (CONVERSATION_FACING) {
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
    } else {
      logInfo('会话侧流程工具已关闭（仅任务看板可用）：未注册 ' + TOOLS.length + ' 个 flow_* 工具')
    }

    // ── ⑫b 任务看板：接口 / agent 工具 / 写 RPC（照参考重建）──────────
    function tbTrash(id, by) {
      var l = tbLoad()
      var t = tbFind(l, id)
      if (!t) return { ok: false, error: '任务不存在：' + id }
      t.trashedAt = nowIso()
      t.updatedAt = nowIso()
      t.updatedBy = tbActor(by)
      if (!tbSave(l)) return { ok: false, error: '任务库写入失败' }
      try { ctx.fs.rm(P_TASKS + '/' + tbMirrorName(t.id) + '.json') } catch (e) { }
      return { ok: true, task: t }
    }
    function tbToolLine(t) {
      var bits = [t.id, '[' + tbLabel(t.status) + ']', t.title]
      if (t.urgency === 'urgent') bits.push('（紧急）')
      if (t.blocked) bits.push('（阻塞）')
      if (t.flowId) bits.push('流程 ' + t.flowId + (t.flowProgress && t.flowProgress.label ? ' · ' + t.flowProgress.label : ''))
      if (t.claimedBy) bits.push('认领 ' + t.claimedBy)
      return bits.join(' ')
    }
    var TB_TOOLS = [
      {
        name: 'taskboard_list',
        description: '列出任务看板上的任务（默认当前项目；可只列某状态）。任务归项目，不归会话。',
        parameters: {
          type: 'object',
          properties: {
            status: { type: 'string', description: '可选：backlog/todo/in_progress/in_review/done/canceled/archived' },
            workspace: { type: 'string', description: '可选：项目（工作区）名；默认当前项目' },
            limit: { type: 'number', description: '可选：最多返回条数（默认 50）' }
          },
          required: []
        },
        execute: function (args) {
          var a = args || {}
          var snap = tbSnapshot({ workspace: a.workspace ? String(a.workspace) : '' })
          var rows = snap.tasks || []
          if (a.status) rows = rows.filter(function (t) { return t.status === String(a.status) })
          var lim = isNum(a.limit) ? Math.max(1, Math.min(200, a.limit)) : 50
          rows = rows.slice(0, lim)
          var text = rows.length
            ? ('任务看板（' + snap.workspace + '，rev ' + snap.revision + '）：\n' + rows.map(function (t) { return '· ' + tbToolLine(t) }).join('\n'))
            : ('任务看板（' + snap.workspace + '）暂无任务。可用 taskboard_create 建卡。')
          return { text: text, tasks: rows }
        }
      },
      {
        name: 'taskboard_create',
        description: '在看板上建一张任务卡（归当前项目）。可选绑定一个流程模板作为执行骨架。',
        parameters: {
          type: 'object',
          properties: {
            title: { type: 'string', description: '任务标题（必填，也是卡片名）' },
            description: { type: 'string', description: '可选：任务描述' },
            prompt: { type: 'string', description: '可选：执行时额外的提示词' },
            urgency: { type: 'string', description: '可选：urgent/normal/relaxed（默认 normal）' },
            status: { type: 'string', description: '可选：建卡入列 —— todo（默认，直接待办）/ backlog（待规划，先由 agent 规划再入待办）' },
            flowId: { type: 'string', description: '可选：绑定的流程模板 id（如 plugin-dev），门禁会成为验收清单' }
          },
          required: ['title']
        },
        execute: function (args) {
          var a = args || {}
          var r = tbCreate({
            title: a.title, description: a.description, prompt: a.prompt,
            urgency: a.urgency, flowId: a.flowId, status: a.status,
            by: { kind: 'agent', sessionId: String(a.convId || '') }
          })
          if (!r.ok) return { text: r.error, error: r.error }
          return { text: '已建卡：' + r.task.id + ' ' + r.task.title + '（状态「' + tbLabel(r.task.status) + '」' + (r.task.flowId ? '，流程 ' + r.task.flowId : '') + '）', task: r.task }
        }
      },
      {
        name: 'taskboard_claim',
        description: '认领一个待办任务（todo → 进行中），认领者记为该会话。被他人持有时不可抢。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '任务 id' },
            convId: { type: 'string', description: '可选：认领会话 id' }
          },
          required: ['id']
        },
        execute: function (args) {
          var a = args || {}
          var l = tbLoad()
          var t = tbFind(l, String(a.id || ''))
          if (!t) return { text: '任务不存在：' + a.id, error: 'not_found' }
          if (t.status === 'in_progress' && t.claimedBy && a.convId && t.claimedBy !== String(a.convId)) {
            return { text: '任务 ' + t.id + ' 已被 ' + t.claimedBy + ' 认领，不能抢', error: 'held' }
          }
          var r = tbMove(t.id, 'in_progress', { by: { kind: 'agent', sessionId: String(a.convId || '') } })
          if (!r.ok) return { text: r.error, error: r.error }
          return { text: '已认领 ' + r.task.id + '「' + r.task.title + '」→ 进行中', task: r.task }
        }
      },
      {
        name: 'taskboard_move',
        description: '改任务状态（受七态状态机约束）。★ agent 不能把任务移到 done —— 验收只能由人做。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '任务 id' },
            to: { type: 'string', description: '目标状态：backlog/todo/in_progress/in_review/done/canceled/archived' },
            convId: { type: 'string', description: '可选：执行会话 id' }
          },
          required: ['id', 'to']
        },
        execute: function (args) {
          var a = args || {}
          var r = tbMove(String(a.id || ''), String(a.to || ''), { by: { kind: 'agent', sessionId: String(a.convId || '') } })
          if (!r.ok) return { text: r.error, error: r.error }
          return { text: '任务 ' + r.task.id + ' → 「' + tbLabel(r.task.status) + '」', task: r.task }
        }
      },
      {
        name: 'taskboard_report',
        description: '交执行报告（摘要/改动文件/风险），任务自动转「待验收」等人验收（不能自己标完成）。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '任务 id' },
            summary: { type: 'string', description: '做了什么、结果如何' },
            files: { type: 'array', items: { type: 'string' }, description: '可选：改动文件列表' },
            risks: { type: 'string', description: '可选：风险/未完成项' },
            outcome: { type: 'string', description: '可选：succeeded/failed/cancelled（默认 succeeded）' }
          },
          required: ['id', 'summary']
        },
        execute: function (args) {
          var a = args || {}
          var r = tbReport(String(a.id || ''), { summary: a.summary, files: a.files, risks: a.risks, outcome: a.outcome, by: { kind: 'agent' } })
          if (!r.ok) return { text: r.error, error: r.error }
          return { text: '已交报告，任务 ' + r.task.id + ' 状态「' + tbLabel(r.task.status) + '」' + (r.movedToReview ? '（转待验收）' : ''), task: r.task }
        }
      },
      {
        name: 'taskboard_check',
        description: '勾选/取消任务的验收清单项（DoD），可附证据说明。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '任务 id' },
            itemId: { type: 'string', description: '清单项 id' },
            checked: { type: 'boolean', description: 'true 勾选 / false 取消' },
            note: { type: 'string', description: '可选：证据说明' }
          },
          required: ['id', 'itemId']
        },
        execute: function (args) {
          var a = args || {}
          var r = tbCheck(String(a.id || ''), String(a.itemId || ''), a.checked !== false, a.note, { kind: 'agent' })
          if (!r.ok) return { text: r.error, error: r.error }
          var done = (r.task.checklist || []).filter(function (x) { return x.checked }).length
          return { text: '清单已更新：' + done + '/' + (r.task.checklist || []).length, task: r.task }
        }
      },
      {
        name: 'taskboard_comment',
        description: '给任务加一条评论（协作/卡点记录）。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '任务 id' },
            text: { type: 'string', description: '评论内容' }
          },
          required: ['id', 'text']
        },
        execute: function (args) {
          var a = args || {}
          var r = tbComment(String(a.id || ''), String(a.text || ''), { kind: 'agent' })
          if (!r.ok) return { text: r.error, error: r.error }
          return { text: '已评论任务 ' + r.task.id, task: r.task }
        }
      },
      {
        name: 'taskboard_plan',
        description: '写规划 / 执行拆解。★ 任务在「待规划」(backlog) 时：写的是**规划**（可多轮修订，每轮记一版），' +
          '规划成熟后传 final=true 定稿 → 卡移入「待办」（未开 planAutoAccept 时停在「待确认」，等人看板确认）。' +
          '任务在「待办 / 进行中」时：写的是执行拆解（原语义）。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '任务 id' },
            subtasks: {
              type: 'array',
              items: {
                type: 'object',
                properties: {
                  text: { type: 'string', description: '子项内容' },
                  status: { type: 'string', description: 'pending/in_progress/completed/blocked/cancelled' }
                }
              },
              description: '子项列表（规划 3-8 条；字符串或 {text,status}）'
            },
            summary: { type: 'string', description: '规划概要：一句话方案（规划模式建议必填）' },
            questions: { type: 'array', items: { type: 'string' }, description: '需要用户拍板的待确认问题（规划模式）' },
            risks: { type: 'array', items: { type: 'string' }, description: '已识别的风险 / 前置依赖（规划模式）' },
            final: { type: 'boolean', description: '规划模式：true=定稿（移入待办或待确认）；false/缺省=记新版本' },
            start: { type: 'boolean', description: 'true=开始/继续规划：把规划任务书派发给规划会话（仅待规划卡）' }
          },
          required: ['id']
        },
        execute: function (args) {
          var a = args || {}
          var id = String(a.id || '')
          if (a.start) {
            var d = tbPlanDispatch(id, { by: { kind: 'agent' }, convId: String(a.convId || '') })
            if (!d.ok) return { text: d.error, error: d.error }
            return { text: '已派发规划（via=' + d.via + (d.queued ? '，排队' : '') + '）：agent 会调研后调用 taskboard_plan 写第一版', task: d.task, plan: d.plan }
          }
          var r = tbPlan(id, a.subtasks, { kind: 'agent' }, { summary: a.summary, questions: a.questions, risks: a.risks, final: a.final === true })
          if (!r.ok) return { text: r.error, error: r.error }
          if (r.stage) {
            var nm = (r.plan && r.plan.rounds) ? r.plan.rounds.length : 0
            if (r.movedToTodo) {
              return { text: '规划已定稿（第 ' + nm + ' 版）并移入「待办」：' + (r.task.subtasks || []).length + ' 步', task: r.task, plan: r.plan }
            }
            if (r.stage === 'review') {
              return { text: '规划已提交（第 ' + nm + ' 版）→ 等待用户在「待确认」里确认后移入待办', task: r.task, plan: r.plan }
            }
            return { text: '已写入规划第 ' + (r.round || 1) + ' 版：' + (r.task.subtasks || []).length + ' 步' +
              ((r.plan && r.plan.questions && r.plan.questions.length) ? ('，待确认问题 ' + r.plan.questions.length + ' 个') : ''), task: r.task, plan: r.plan }
          }
          return { text: '已写入执行拆解：' + (r.task.subtasks || []).length + ' 条', task: r.task }
        }
      },
      {
        name: 'taskboard_execute',
        description: '执行一张任务卡：把任务书投递到工作会话，由 agent 按卡片描述/执行提示词/DoD 执行（有监督者时自动多轮驱动）。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '任务 id' },
            convId: { type: 'string', description: '可选：执行会话 id（缺省用卡上记录的会话）' }
          },
          required: ['id']
        },
        execute: function (args) {
          var a = args || {}
          var r = tbExecute(String(a.id || ''), { by: 'agent', convId: String(a.convId || '') })
          if (!r.ok) return { text: r.error, error: r.error }
          return { text: '已派发执行：' + r.task.id + ' → 会话 ' + r.convId + (r.queued ? '（排队）' : ''), task: r.task }
        }
      },
      {
        name: 'taskboard_schedule',
        description: '给任务设/取消定时执行：interval=每 N 分钟；daily=每天 HH:MM；at=一次性时间点。',
        parameters: {
          type: 'object',
          properties: {
            id: { type: 'string', description: '任务 id' },
            enabled: { type: 'boolean', description: 'true 启用 / false 取消' },
            mode: { type: 'string', description: 'interval / daily / at' },
            everyMinutes: { type: 'number', description: 'interval 模式的间隔分钟数' },
            atTime: { type: 'string', description: 'daily 模式的时间 HH:MM' },
            atISO: { type: 'string', description: 'at 模式的一次性 ISO 时间' }
          },
          required: ['id', 'enabled']
        },
        execute: function (args) {
          var a = args || {}
          var r = tbUpdate(String(a.id || ''), {
            schedule: { enabled: a.enabled !== false, mode: a.mode, everyMinutes: a.everyMinutes, atTime: a.atTime, atISO: a.atISO }
          }, { kind: 'agent' })
          if (!r.ok) return { text: r.error, error: r.error }
          var s = r.task.schedule || {}
          return { text: '定时已更新：' + (s.enabled ? ('下次 ' + s.nextAt) : '已关闭'), task: r.task }
        }
      }
    ]
    for (var tbI = 0; tbI < TB_TOOLS.length; tbI++) {
      (function (t) {
        try {
          ctx.tools.register({
            name: t.name,
            description: t.description,
            parameters: t.parameters,
            execute: function (args) { return t.execute(args || {}) },
          })
        } catch (e) { logWarn('任务看板工具注册失败 ' + t.name + ': ' + e) }
      })(TB_TOOLS[tbI])
    }

    try {
      ctx.http.register('GET', '/api/taskboard/state', function (req) {
        var q = parseQuery(req && req.query)
        var body = ''
        try { body = JSON.stringify(tbSnapshot(q)) }
        catch (e) {
          return { status: 500, body: JSON.stringify({ ok: false, error: '任务看板快照失败: ' + e }), headers: { 'Content-Type': 'application/json' } }
        }
        return { status: 200, body: body, headers: { 'Content-Type': 'application/json; charset=utf-8' } }
      })
    } catch (e) { logWarn('接口注册失败 /api/taskboard/state: ' + e) }

    // 定时调度器：每 60s 扫描 schedule 到期任务（插件卸载时宿主自动清理）
    try {
      var tbTimer = null
      try { tbTimer = ctx.timer || (typeof ctx.get === 'function' ? ctx.get('timer') : null) } catch (e) { tbTimer = null }
      if (tbTimer && typeof tbTimer.interval === 'function') {
        tbTimer.interval(function () {
          try { tbScheduleScan() } catch (e) { logWarn('定时扫描失败: ' + e) }
        }, 60000)
        logInfo('任务看板定时调度器已启动（60s 扫描 schedule）')
      } else {
        logWarn('任务看板定时调度器未启动：ctx.get("timer") 不可用')
      }
    } catch (e) { logWarn('定时调度器启动失败: ' + e) }

    try {
      ctx.registerClientMethod('taskboard_action', function (args) {
        var a = (args && typeof args === 'object') ? args : {}
        var op = String(a.op || '')
        var byUser = { kind: 'user', sessionId: String(a.convId || '') }
        // 任何带会话的看板操作都记进「最近活跃会话池」：定时执行无显式会话时按它投递
        if (a.convId) tbRememberConv(String(a.convId))
        if (op === 'state') return tbSnapshot(a)
        if (op === 'create') {
          return tbCreate({
            title: a.title, description: a.description, prompt: a.prompt,
            urgency: a.urgency, flowId: a.flowId, workspace: a.workspace,
            status: a.status,
            schedule: a.schedule, convId: a.convId, by: byUser
          })
        }
        if (op === 'update') return tbUpdate(String(a.id || ''), a.patch || a, { by: byUser })
        if (op === 'move') return tbMove(String(a.id || ''), String(a.to || ''), { by: byUser, sessionId: a.convId })
        if (op === 'claim') return tbClaim(String(a.id || ''), byUser, String(a.convId || ''))
        if (op === 'release') return tbRelease(String(a.id || ''), byUser)
        if (op === 'execute') return tbExecute(String(a.id || ''), { by: 'user', convId: String(a.convId || '') })
        if (op === 'schedule') {
          var schIn = a.schedule || a.patch || a
          if (schIn && typeof schIn === 'object' && !schIn.convId && a.convId) schIn.convId = String(a.convId)
          return tbUpdate(String(a.id || ''), { schedule: schIn }, { by: byUser })
        }
        if (op === 'plan') {
          return tbPlan(String(a.id || ''), a.subtasks, byUser, {
            summary: a.summary, questions: a.questions, risks: a.risks, final: a.final === true
          })
        }
        // 「待规划」列的可执行入口：开始/继续规划、讨论、定稿、确认
        if (op === 'planStart') return tbPlanDispatch(String(a.id || ''), { by: byUser, convId: String(a.convId || '') })
        if (op === 'planDiscuss') return tbPlanDiscuss(String(a.id || ''), String(a.text || ''), byUser, String(a.convId || ''))
        if (op === 'planFinalize') return tbPlanFinalize(String(a.id || ''), byUser, false)
        if (op === 'planAccept') return tbPlanAccept(String(a.id || ''), byUser)
        if (op === 'comment') return tbComment(String(a.id || ''), a.text, byUser)
        if (op === 'check') return tbCheck(String(a.id || ''), String(a.itemId || ''), a.checked !== false, a.note, byUser)
        if (op === 'trash') return tbTrash(String(a.id || ''), byUser)
        if (op === 'driver') return tbDriverSave(a.patch || a, byUser)
        if (op === 'driverState') return { ok: true, driver: tbDriverLoad() }
        if (op === 'startFlow') {
          var l0 = tbLoad()
          var tk = tbFind(l0, String(a.id || ''))
          if (!tk) return { ok: false, error: '任务不存在：' + a.id }
          if (!tk.flowId) return { ok: false, error: '任务未绑定流程（建卡时可选流程模板）' }
          var sr = startRun({ flowId: tk.flowId, convId: String(a.convId || '') })
          if (!sr || !sr.ok) return { ok: false, error: (sr && sr.error) || '流程启动失败' }
          var rid = (sr.run && sr.run.runId) || sr.runId || ''
          if (rid) tbBindRun(tk.id, rid)
          return { ok: true, runId: rid, run: sr.run || null }
        }
        return { ok: false, error: '未知操作：' + op }
      })
    } catch (e) { logWarn('RPC 注册失败 taskboard_action: ' + e) }

    // ── ⑫ 只读接口（给看板）─────────────────────────────────────
    function stateSnapshot(q) {
      var convId = String(q.convId || '')
      var run = pickRun({ convId: convId })
      // 归档/删除 = 只读留档：不出现在活跃视图（只在「运行记录」页签里）。
      // （生命周期/子任务操作内部仍走 pickRun：会拿到归档实例并给出准确的状态提示。）
      if (run && runHidden(run)) run = null
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
            tasks: def ? (def.tasks || []) : [],
            plan: def ? (def.plan || def.tasks || []) : [],
            status: st.status,
            startedAt: st.startedAt || '', completedAt: st.completedAt || '',
            taskIds: st.taskIds || [], gates: gates,
            subtasks: (st.subtasks || []).map(function (s) {
              return {
                id: s.id, subject: s.subject, detail: s.detail || '', status: subStatusOf(s),
                source: s.source || 'plan', note: s.note || '', evidence: s.evidence || '',
                createdAt: s.createdAt || '', startedAt: s.startedAt || '', completedAt: s.completedAt || '',
                hostTaskId: s.hostTaskId || '',
              }
            }),
            subStats: subStats(st),
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
          status: run.status, statusBefore: run.statusBefore || '',
          currentStage: run.currentStage, project: run.project, convId: run.convId || '',
          gitHead: run.gitHead || '', modifiedFiles: run.modifiedFiles || [],
          evidenceDir: run.evidenceDir || '', stageProgress: sp,
          progress: runProgress(run),
          focus: focusSubtask(run),
          createdAt: run.createdAt || '', updatedAt: run.updatedAt || '',
          interruptedAt: run.interruptedAt || '', interruptReason: run.interruptReason || '',
          archivedAt: run.archivedAt || '', deletedAt: run.deletedAt || '',
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
        runs: runsRecord(),
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
          try { if (r.run) tbSyncFromRun(r.run) } catch (e) { logWarn('任务同步失败: ' + e) }
          return { ok: true, result: { advanced: !!r.advanced, completed: !!r.completed, stage: r.stage || '', text: r.text || '', gate: r.gate || null, pending: r.pending || null, rollback: r.rollback || null } }
        }
        if (op === 'approve') {
          if (a.approve !== true) return { ok: false, error: 'approve 必须为 true（审批卡点由用户在界面显式批准）' }
          var r2 = advanceRun({ runId: a.runId, convId: a.convId, gateId: a.gateId, approve: true, note: a.note })
          if (!r2.ok) return { ok: false, error: r2.error }
          try { if (r2.run) tbSyncFromRun(r2.run) } catch (e) { logWarn('任务同步失败: ' + e) }
          return { ok: true, result: { gate: r2.gate || null, advanced: !!r2.advanced, text: r2.text || '' } }
        }
        // 子任务（看板勾选/增删 → 与 flow_task 工具同一实现）
        // 注意：顶层 op='task' 是**通道名**，子操作由 action（兼容 subOp）指定：list | add | update | remove
        if (op === 'task') {
          var ta = {}
          for (var tk in a) { if (Object.prototype.hasOwnProperty.call(a, tk)) ta[tk] = a[tk] }
          ta.op = String(a.action || a.subOp || 'list')
          var rt = taskAction(ta)
          if (!rt.ok) return { ok: false, error: rt.error }
          return {
            ok: true,
            result: {
              subtask: rt.subtask || null, removed: rt.removed || null,
              stats: rt.stats || null, runStats: rt.runStats || null,
              subtasks: rt.subtasks || null, stage: rt.stage || '',
              text: rt.text || '',
            },
          }
        }
        // 实例生命周期（中断/恢复/归档/取消归档/删除/恢复删除/彻底删除）
        if (op === 'interrupt' || op === 'resume' || op === 'archive' || op === 'unarchive' ||
            op === 'delete' || op === 'restore' || op === 'purge') {
          var rl = lifecycleRun(op, a)
          if (!rl.ok) return { ok: false, error: rl.error }
          return {
            ok: true,
            result: {
              status: rl.run ? rl.run.status : 'purged',
              runId: (rl.run && rl.run.runId) || String(a.runId || ''),
              removed: rl.removed || null, text: rl.text || '',
            },
          }
        }
        if (op === 'start') {
          var r3 = startRun({ flowId: a.flowId, convId: a.convId, project: a.project, force: !!a.force, taskId: a.taskId || a.boardTaskId })
          if (!r3.ok) return { ok: false, error: r3.error }
          return { ok: true, result: { runId: r3.run.runId, flowId: r3.run.flowId, taskId: (r3.task && r3.task.id) || '', text: '流程已启动：' + r3.run.flowId } }
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
        return { ok: false, error: '未知 op（支持 advance | approve | start | task | interrupt | resume | archive | unarchive | delete | restore | purge | attach | detach | show | validate | define | clone | publish | import）' }
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
    // ★ 2026-10-17 使用面收窄：门禁钩子只服务于会话内进程推进，随会话侧一起关闭。
    if (CONVERSATION_FACING) {
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
    }

    // ── ⑮ 命令面（§14.6-②）──────────────────────────────────────
    // ★ 2026-10-17 使用面收窄：/flow、/flow-board、/flow-import 不再注册（流程只在任务看板里使用）。
    if (CONVERSATION_FACING) {
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
    } else {
      logInfo('会话侧流程命令已关闭（仅任务看板可用）：未注册 /flow、/flow-board、/flow-import')
    }

    // ── ⑯ 装载摘要（诊断用）─────────────────────────────────────
    logInfo('装载完成：预设 ' + presetIds().length + ' · 全局 ' + libIds().length + ' · 项目 ' + projectIds().length +
      ' · 工具 ' + TOOLS.length + ' · 实例 ' + listRuns().length + ' · 强度 ' + gateStrength())
  },
}
