// ═══════════════════════════════════════════════════════════════
// plugin-runtime.js — 浏览器侧插件 client 半运行时（host/client 双半的 client 侧）
//
// 契约（与后端 DefineJSCodeFull / cordis_define 的 client 参数一致）：
//   client 半形态：(ui) => void
//     ui.on(event, fn)            收 host → 浏览器事件（ui:/client: 前缀）
//     ui.emit(event, payload)     发事件回 host（host: 前缀给 host 插件消费）
//     ui.invoke(plugin, method, args?)  远程调用 host 半注册方法（D11 invoke RPC）
//     ui.reportFailure(phase, message)  client 半失败上报（render/guard/boot）
//     ui.registerPanel({id,title,icon,render,props}) 注册自定义面板（渲染进插件面板 client 区）
//     ui.http.get(path)/post(path, body)       受限后端 API 调用
//     ui.log(...)                 日志（控制台）
//
// 事件流：
//   host → 浏览器：EventBus 里 ui:/client: 前缀事件入 host 队列，本模块每 2s
//                 轮询 /api/plugins/client-events 取增量，分发给各 client 半的 on 监听器。
//   浏览器 → host：ui.emit 调 POST /api/plugins/event（host 插件 ctx.on('host:xxx') 消费）。
//   invoke RPC：ui.invoke 调 POST /api/plugins/invoke（host 半 ctx.registerClientMethod 注册）。
//
// 面板渲染：client 半注册的面板显示在插件面板的「客户端面板」区；render(el, ui) 在
// 挂载时调用，el 为容器 DOM 元素（可自由操作 DOM），ui 为当前 client 半的沙箱对象。
// props 声明面板数据契约（轻量 Slot：{field: type}，宿主可注入外部数据）。
//
// ★ 宿主预定义 UI 槽位（slotId，插件用 ui.registerSlot({slotId,...}) 注册占用）：
//   ── 替换型（single，宿主区域整体由插件渲染，面板里下拉切换占用者）──
//     titlebar     标题栏整条（含 logo/菜单/标题，App 顶部）
//     activitybar  活动栏整条竖列（App 左侧竖条）
//     sidebar      左侧文件栏整栏（App 左侧）
//     editor       主编辑区整块（App 中部：编辑器 tab + 代码 + 底部终端面板）
//     right-panel  右侧容器整块（App 右侧：含对话面板/外壳）
//     chat         对话面板整区（RightPanel rp-body，含消息+输入区）
//     statusbar    底部状态栏整栏（App 底部）
//   ── 叠加型（list，宿主容器内每个占用者渲染一个小条目，面板里勾选激活）──
//     overlay          浮动层（fixed 全屏，不挡交互，badge/toast）
//     titlebar-right   标题栏右侧按钮区（App 内置标题栏，工作区切换按钮旁）
//     activitybar      活动栏图标列（ActivityBar 顶部，图标+tooltip 入口）
//     editor-toolbar   编辑器标签栏尾部（undo/redo 按钮旁）
//     chat-tools       对话输入区上方工具条（发送按钮上方一行快捷工具）
//     statusbar-items  内置状态栏内叠加条目（左侧信息与右侧信息之间）
//   ★ 同一 slotId 可同时存在 single 与 list 两类占用（如 activitybar）：
//     机制按 kind 分流（getSlotOwner 只管 single，getSlotUIList 只管 list）。
// ═══════════════════════════════════════════════════════════════

import api from './api.js'
import { ref, nextTick } from 'vue'

// ─── 运行中的 client 半实例 ──────────────────────────────────
// instances: [{ name, defId, source, status, error?, onHandlers: Map<event, [fn]> }]
const instances = []

// ─── 事件轮询状态 ────────────────────────────────────────────
let pollTimer = null
let lastSeq = 0
let pollInterval = 2000

// ─── 面板注册表 ──────────────────────────────────────────────
// panels: [{ id, title, icon, render, pluginName }]
export const clientPanels = []
// 面板容器元素注册（PluginPanel 挂载后调用）
let panelMountFn = null

// ─── UI 槽位注册表（Slot 系统：插件可替换的预定义界面区域）───
// slots: [{ slotId, pluginName, title, render, defId }]
// 宿主预定义 slotId：'statusbar'（底部状态栏）、'chat'（对话面板，RightPanel rp-body 整区）。
// 占用者选择（getSlotOwner/setSlotOwner）持久化 localStorage；
// owner = '' 表示使用内置组件。
export const clientSlots = []
const slotOwnerKey = (id) => 'paircode-slot-' + id

// ─── 内置槽位注册表（一切皆插件：内置 UI 区域 = 内置默认实现，与插件占用统一装配）───
// builtinSlotDefs: { slotId: { slotId, title, desc, kind, builtin:true } }
// ★ 宿主模板直挂内置组件（titlebar/activitybar/sidebar/editor/right-panel/statusbar/chat），
//   装配视图与插件占用统一：面板展示内置区域清单 + 覆盖者，插件占用优先于内置。
//   外部 JS（插件 client 半 / 用户自定义脚本）可调用 registerBuiltinSlot 覆盖注册表项，
//   无需重新编译（注册表是运行时 JS 数据结构）。
const builtinSlotDefs = {}
// registerBuiltinSlot 注册内置区域定义（宿主启动时调用）。
// def: { slotId, title, desc?, kind?('single'|'list', 默认 single) }
export function registerBuiltinSlot(slotId, def) {
  builtinSlotDefs[slotId] = { slotId, kind: def.kind || 'single', builtin: true, title: def.title || slotId, desc: def.desc || '' }
}
// getBuiltinSlots 全部内置区域定义（插件面板装配视图用）。
export function getBuiltinSlots() { return Object.values(builtinSlotDefs) }
// getSlotImplInfo 某槽位当前装配信息：owner='' 内置默认 / 插件占用者名。
export function getSlotImplInfo(slotId) {
  return { slotId, owner: getSlotOwner(slotId), builtin: !!builtinSlotDefs[slotId] }
}
// 槽位变化监听（支持多个订阅者：App 状态栏容器 + 插件面板管理区）
let slotMountFns = []

// setSlotMount 订阅槽位变化（返回取消函数；fn 立即收到当前槽位表）。
export function setSlotMount(fn) {
  if (!fn) { slotMountFns = []; return () => {} }
  slotMountFns.push(fn)
  try { fn(clientSlots) } catch (e) { console.warn('[slot] 初始通知失败', e) }
  return () => {
    const i = slotMountFns.indexOf(fn)
    if (i >= 0) slotMountFns.splice(i, 1)
  }
}
export function emitSlotChanged() {
  for (const fn of slotMountFns) {
    try { fn(clientSlots) } catch (e) { console.warn('[slot] 通知失败', e) }
  }
}

// list 型槽位（叠加）条目激活状态：勾选 = 参与渲染（localStorage 持久化，跨刷新保留）。
// ★ 未显式设置 = 默认激活（注册即显示，对齐参考项目 shell.overlay 语义）。
function overlayKey(slotId, pluginName) { return 'slotOverlay:' + slotId + ':' + pluginName }
export function isOverlayActive(slotId, pluginName) {
  try {
    const v = localStorage.getItem(overlayKey(slotId, pluginName))
    if (v === null) return true // 从未设置：默认激活
    return v === '1'
  } catch (e) { return false }
}
export function setOverlayActive(slotId, pluginName, on) {
  try { localStorage.setItem(overlayKey(slotId, pluginName), on ? '1' : '0') } catch (e) { /* 忽略 */ }
  emitSlotChanged() // 通知宿主重渲染 overlay 槽位
}

// getSlotCandidates 某槽位的全部候选（占用者）。
export function getSlotCandidates(slotId) {
  return clientSlots.filter(s => s.slotId === slotId)
}
// getSlotOwner 当前激活占用者（'' = 内置组件）。
// ★ 只考虑 single 型（kind!=='list'）：叠加型走 isOverlayActive 独立激活，
//   不会被误设为替换 owner（否则 getSlotUI 查不到 → 空容器）。
export function getSlotOwner(slotId) {
  let v = ''
  try { v = localStorage.getItem(slotOwnerKey(slotId)) || '' } catch (e) { /* 忽略 */ }
  if (v && !clientSlots.some(s => s.slotId === slotId && s.kind !== 'list' && s.pluginName === v)) v = ''
  if (v) return v
  // ★ 从未显式选择：仅一个 single 候选时自动激活（对齐参考项目「注册槽位=替换」语义；
  //   用户在面板选过「内置组件」后存了 ''，走 localStorage 命中，不再自动激活）
  let neverChosen = true
  try { neverChosen = localStorage.getItem(slotOwnerKey(slotId)) === null } catch (e) { /* 忽略 */ }
  if (neverChosen) {
    const cands = clientSlots.filter(s => s.slotId === slotId && s.kind !== 'list' && typeof s.render === 'function')
    if (cands.length === 1) return cands[0].pluginName
  }
  return ''
}
// setSlotOwner 切换占用者（'' = 恢复内置）。
export function setSlotOwner(slotId, pluginName) {
  try { localStorage.setItem(slotOwnerKey(slotId), pluginName || '') } catch (e) { /* 忽略 */ }
  emitSlotChanged()
}
// getSlotUI 取激活占用者的渲染信息 {render, ui, pluginName}；无激活返回 null。
export function getSlotUI(slotId) {
  const owner = getSlotOwner(slotId)
  if (!owner) return null
  const s = clientSlots.find(x => x.slotId === slotId && x.pluginName === owner)
  if (!s || typeof s.render !== 'function') return null
  return { render: s.render, ui: getUIFor(owner), pluginName: owner }
}

// getSlotUIList 取 list 型槽位（叠加）的全部占用者渲染信息数组（kind='list'）。
export function getSlotUIList(slotId) {
  return clientSlots
    .filter(s => s.slotId === slotId && s.kind === 'list' && typeof s.render === 'function')
    .map(s => ({ render: s.render, ui: getUIFor(s.pluginName), pluginName: s.pluginName }))
}

// mountListSlot 挂载 list 型槽位宿主（细粒度叠加注入通用入口）。
// hostRef：Vue ref（.value 为容器 DOM）；slotId：宿主预定义槽位 id；
// opts.isActive(pluginName) 可选过滤（默认全部激活，同 overlay 语义）。
// 每个占用者渲染进独立 div（class=plugin-slot-item plugin-slot-<id>-item, data-plugin=名）；
// render 返回的 cleanup 在重渲染前调用。返回取消订阅函数（组件卸载前调用）。
export function mountListSlot(hostRef, slotId, opts = {}) {
  const cleanups = new Map()
  function render() {
    const host = hostRef && hostRef.value
    if (!host) return
    for (const [name, c] of cleanups) {
      try { c() } catch (e) { console.warn('[slot] ' + name + ' cleanup 失败', e) }
    }
    cleanups.clear()
    host.innerHTML = ''
    for (const s of getSlotUIList(slotId)) {
      if (opts.isActive && !opts.isActive(s.pluginName)) continue
      const item = document.createElement('div')
      item.className = 'plugin-slot-item plugin-slot-' + slotId + '-item'
      item.dataset.plugin = s.pluginName
      host.appendChild(item)
      try {
        const ret = s.render(item, s.ui)
        if (typeof ret === 'function') cleanups.set(s.pluginName, ret)
      } catch (e) {
        console.warn('[slot] ' + slotId + ' 渲染失败', e)
        item.innerHTML = '<span style="color:var(--text-muted);font-size:11px">插件条目渲染失败</span>'
      }
    }
  }
  return setSlotMount(() => { render() })
}

// useSingleSlot 单槽位（替换型 single）装配组合函数——收敛宿主组件的装配样板。
// 宿主组件模板（v-if 内置 / v-else 插件容器）：
//   <内置 v-if="!slot.owner" />
//   <div v-else ref="slot.hostRef" class="plugin-slot-host plugin-slot-<id>"></div>
// 组件 onMounted 调 start()（订阅槽位变化 + 首次渲染），onUnmounted 调 stop()（退订+清理）。
// owner 变化时宿主 v-if 自动切换，nextTick 后渲染插件内容到 hostRef 容器。
export function useSingleSlot(slotId) {
  const owner = ref('')
  const hostRef = ref(null)
  let cleanup = null
  let unsub = null
  let started = false

  function render() {
    const host = hostRef.value
    if (!host) return
    if (typeof cleanup === 'function') { try { cleanup() } catch (e) {} cleanup = null }
    host.innerHTML = ''
    const s = getSlotUI(slotId)
    if (s && typeof s.render === 'function') {
      try {
        const ret = s.render(host, s.ui)
        if (typeof ret === 'function') cleanup = ret
      } catch (e) {
        console.warn('[slot] ' + slotId + ' 渲染失败', e)
        host.innerHTML = '<div style="padding:8px;font-size:12px;color:var(--text-muted)">插件「' + slotId + '」渲染失败</div>'
      }
    }
  }

  function refresh() {
    const prev = owner.value
    owner.value = getSlotOwner(slotId)
    if (owner.value !== prev && typeof cleanup === 'function') {
      // 占用者变化（含切回内置）：先清理旧插件渲染
      try { cleanup() } catch (e) {} cleanup = null
    }
    nextTick(() => { if (owner.value) render() })
  }

  return {
    owner,
    hostRef,
    // ★ setup 顶层同步调用：初始化 owner（避免首帧先 mount 内置组件再切插件
    //   分支——复杂组件（CM6 编辑器等）的「mount 后立即卸载」会触发错误）。
    //   在组件 setup 里 useSingleSlot(...) 后立即调用。
    init() { owner.value = getSlotOwner(slotId) },
    start() {
      if (started) return
      started = true
      refresh()
      unsub = setSlotMount(refresh)
    },
    stop() {
      started = false
      if (unsub) { unsub(); unsub = null }
      if (typeof cleanup === 'function') { try { cleanup() } catch (e) {} cleanup = null }
    },
  }
}

// setPanelMount 供 PluginPanel 注入「渲染 client 面板」的回调。
// 回调签名 (panels) => void，面板列表变化时触发。
export function setPanelMount(fn) {
  panelMountFn = fn
  if (panelMountFn) panelMountFn(clientPanels)
}

// emitPanelChanged 面板列表变化时通知 PluginPanel 重渲染。
function emitPanelChanged() {
  if (panelMountFn) panelMountFn(clientPanels)
}

// ─── 沙箱 ui 对象构建 ────────────────────────────────────────

function makeUI(inst) {
  const ui = {
    // 收 host → 浏览器事件（ui:/client: 前缀由后端转发）
    on(event, fn) {
      if (typeof fn !== 'function') return
      if (!inst.onHandlers.has(event)) inst.onHandlers.set(event, [])
      inst.onHandlers.get(event).push(fn)
      // 登记事件监听（上报快照用）
      if (!inst.events.includes(event)) inst.events.push(event)
      return () => {
        const arr = inst.onHandlers.get(event) || []
        const i = arr.indexOf(fn)
        if (i >= 0) arr.splice(i, 1)
        if ((inst.onHandlers.get(event) || []).length === 0) {
          inst.onHandlers.delete(event)
          const ei = inst.events.indexOf(event)
          if (ei >= 0) inst.events.splice(ei, 1)
        }
      }
    },
    // 发事件回 host（host: 前缀约定；host 插件 ctx.on 消费）
    async emit(event, payload) {
      try {
        await api.pluginEmit(event, payload)
      } catch (e) {
        console.warn('[plugin] emit 失败', event, e)
      }
    },
    // D11 invoke RPC：远程调用 host 半注册的方法（ctx.registerClientMethod）。
    // 返回 {ok, value} 或 {ok:false, error}；插件可再 await 值。
    async invoke(plugin, method, args) {
      const res = await api.pluginInvoke(plugin, method, args)
      if (!res || !res.ok) {
        throw new Error((res && res.error) || 'invoke 失败: ' + method)
      }
      return res.value
    },
    // D11 失败上报：client 半 render/guard/boot 失败 → 后端记诊断，Agent 经
    // cordis_inspect 发现修复（不中断 host 半运行）。
    reportFailure(phase, message) {
      const ph = (phase === 'guard' || phase === 'boot') ? phase : 'render'
      api.pluginClientFailure(inst.name, ph, String(message || 'unknown error')).catch(() => {})
    },
    // 注册自定义面板（显示在插件面板 client 区）
    registerPanel(spec) {
      if (!spec || !spec.id || !spec.title) {
        console.warn('[plugin] registerPanel 需要 {id, title, render?}')
        return
      }
      const existing = clientPanels.findIndex(p => p.id === spec.id)
      const panel = {
        id: spec.id,
        title: spec.title,
        icon: spec.icon || 'sparkles',
        render: typeof spec.render === 'function' ? spec.render : null,
        // 轻量 Slot：props 声明面板数据契约（{field: type}），宿主/其他插件可注入
        props: spec.props && typeof spec.props === 'object' ? { ...spec.props } : null,
        pluginName: inst.name,
      }
      if (existing >= 0) clientPanels[existing] = panel
      else clientPanels.push(panel)
      emitPanelChanged()
      return {
        // 更新面板内容（重新渲染）
        update() { emitPanelChanged() },
        // 移除面板
        remove() {
          const i = clientPanels.findIndex(p => p.id === panel.id)
          if (i >= 0) { clientPanels.splice(i, 1); emitPanelChanged() }
        },
      }
    },
    // 注册 UI 槽位占用（Slot 系统：替换宿主预定义界面区域，如 'statusbar'/'chat'；
    // kind='list' 槽位为叠加型——多个占用者同时渲染，如 'overlay' 浮动层）。
    // 同插件重复注册同槽位 → 替换。single 槽位宿主按 getSlotOwner 决定激活哪个
    // 占用者；list 槽位宿主渲染全部占用者。激活后调 render(el, ui)；render 可
    // 返回 cleanup 函数（宿主下次重渲染前调用）。
    registerSlot(spec) {
      if (!spec || !spec.slotId || !spec.title) {
        console.warn('[plugin] registerSlot 需要 {slotId, title, render?}')
        return
      }
      const idx = clientSlots.findIndex(s => s.slotId === spec.slotId && s.pluginName === inst.name)
      const slot = {
        slotId: spec.slotId,
        pluginName: inst.name,
        title: spec.title,
        kind: spec.kind === 'list' ? 'list' : 'single',
        render: typeof spec.render === 'function' ? spec.render : null,
        defId: inst.defId,
      }
      if (idx >= 0) clientSlots[idx] = slot
      else clientSlots.push(slot)
      emitSlotChanged()
      return {
        update() { emitSlotChanged() },
        remove() {
          const i = clientSlots.findIndex(s => s.slotId === spec.slotId && s.pluginName === inst.name)
          if (i >= 0) { clientSlots.splice(i, 1); emitSlotChanged() }
        },
      }
    },
    // 受限后端 API（相对路径）
    http: {
      get: (path, params) => api.apiGet(path, params),
      post: (path, body) => api.apiPost(path, body),
    },
    log: (...args) => console.log('[plugin:' + inst.name + ']', ...args),
  }
  return ui
}

// ─── 装载/卸载 ───────────────────────────────────────────────

// loadClientHalf 装载一个 client 半（new Function 求值 + 调用 (ui) => void）。
// source: { name, defId, clientCode, source }
export function loadClientHalf(source) {
  if (!source || !source.clientCode || !String(source.clientCode).trim()) return null
  const code = String(source.clientCode)
  // 语法预检（后端已预检，这里兜底）
  let fn
  try {
    // ★ client 半是「函数」形态（(ui) => {...} / function(ui) {...}）：
    //   必须包装为 return (code)(ui) 立即调用——否则 new Function 只是创建
    //   箭头函数而不执行，registerSlot 静默失效（status=loaded 但槽位为空，
    //   UI 插件看起来「没生效」）。多语句/自执行形态代码原样执行。
    // ★ 先剥离开头的行注释/块注释（代码可能以多行注释开头，注释会破坏函数
    //   形态检测——只剥一行不够，要剥全部前置注释）。
    const t = code.trim().replace(/^(?:\s*(?:\/\/[^\r\n]*|\/\*[\s\S]*?\*\/)\r?\n?)+/, '').trim()
    const isFnExpr = /^\(?\s*(async\s+)?(\(?\s*ui\s*\)?\s*=>|function\s*\()/.test(t)
    fn = new Function('ui', '"use strict";\n' + (isFnExpr ? 'return (' + code + ')(ui)' : code))
  } catch (e) {
    console.warn('[plugin] client 半语法错误', source.name, e)
    return null
  }
  const inst = {
    name: source.name,
    defId: source.defId,
    source: source.source || 'js',
    status: 'loaded',
    events: [],
    onHandlers: new Map(),
  }
  // 缓存沙箱 ui（供面板渲染 render(el, ui) 取用）
  inst.ui = makeUI(inst)
  try {
    fn(inst.ui)
  } catch (e) {
    console.warn('[plugin] client 半执行错误', source.name, e)
    inst.status = 'error'
    inst.error = String(e && e.message || e)
    instances.push(inst)
    // D11 失败上报：boot 阶段执行错误 → 后端记诊断（Agent inspect 修复）
    api.pluginClientFailure(source.name, 'boot', inst.error).catch(() => {})
    emitPanelChanged()
    emitSlotChanged()
    return inst
  }
  instances.push(inst)
  // 重新装载时同步一次面板与槽位
  emitPanelChanged()
  emitSlotChanged()
  return inst
}

// unloadClientHalf 卸载（stop/undefine 时清理实例与面板）。
export function unloadClientHalf(nameOrDefId) {
  const i = instances.findIndex(inst => inst.name === nameOrDefId || inst.defId === nameOrDefId)
  if (i >= 0) {
    const name = instances[i].name
    instances.splice(i, 1)
    // 移除该插件注册的面板
    for (let j = clientPanels.length - 1; j >= 0; j--) {
      if (clientPanels[j].pluginName === name) {
        clientPanels.splice(j, 1)
      }
    }
    // 移除该插件注册的槽位占用
    for (let j = clientSlots.length - 1; j >= 0; j--) {
      if (clientSlots[j].pluginName === name) {
        clientSlots.splice(j, 1)
      }
    }
    emitPanelChanged()
    emitSlotChanged()
  }
  reportState()
}

// syncClientHalves 与后端插件列表对齐：装载已批准的新 client 半，卸载已消失的。
// ★ client 激活审批：仅装载 clientApproved=true 的插件（后端 cordis_run 经
//   现有审批门后标记批准；未批准=hasClient && !clientApproved 显示「待批准」，
//   Agent 侧 cordis_run 触发审批，前端轮询/刷新后自动装载）。
export async function syncClientHalves(plugins) {
  if (!plugins || !Array.isArray(plugins)) return
  const active = new Set()
  for (const p of plugins) {
    if (p.hasClient && p.state === 'running' && p.clientApproved && p.clientCode) {
      active.add(p.name)
      const exists = instances.find(inst => inst.name === p.name)
      // error 实例重试装载（可能是瞬时执行错误）
      if (!exists || exists.status === 'error') {
        if (exists) {
          const ei = instances.indexOf(exists)
          if (ei >= 0) instances.splice(ei, 1)
        }
        loadClientHalf({ name: p.name, defId: p.defId, clientCode: p.clientCode })
      }
    }
  }
  // 卸载已停止/删除的
  for (let i = instances.length - 1; i >= 0; i--) {
    if (!active.has(instances[i].name)) {
      instances.splice(i, 1)
    }
  }
  // 清理孤儿面板
  const liveNames = new Set(instances.map(x => x.name))
  for (let j = clientPanels.length - 1; j >= 0; j--) {
    if (!liveNames.has(clientPanels[j].pluginName)) {
      clientPanels.splice(j, 1)
    }
  }
  // 清理孤儿槽位注册
  for (let j = clientSlots.length - 1; j >= 0; j--) {
    if (!liveNames.has(clientSlots[j].pluginName)) {
      clientSlots.splice(j, 1)
    }
  }
  emitPanelChanged()
  emitSlotChanged()
  reportState()
}

// ─── host → 浏览器事件轮询 ──────────────────────────────────

// dispatchHostEvent 把一条 host 事件分发给所有 client 半的 on 监听器。
function dispatchHostEvent(ev) {
  for (const inst of instances) {
    const fns = inst.onHandlers.get(ev.name)
    if (!fns) continue
    for (const fn of fns) {
      try {
        fn(ev.payload)
      } catch (e) {
        console.warn('[plugin] 事件处理错误', inst.name, ev.name, e)
      }
    }
  }
}

// startPolling 启动事件轮询（每 2s 一次）；顺带上报 client 半运行快照。
export function startPolling() {
  if (pollTimer) return
  pollTimer = setInterval(async () => {
    try {
      const res = await api.pluginClientEvents(lastSeq)
      if (res && Array.isArray(res.events)) {
        for (const ev of res.events) dispatchHostEvent(ev)
        if (typeof res.lastSeq === 'number') lastSeq = res.lastSeq
      }
    } catch (e) {
      // 静默：后端未就绪时跳过
    }
    reportState()
  }, pollInterval)
}

// ─── client 半运行快照上报（client inspect provider 数据源）──

// buildSnapshot 汇总当前 client 半实例 → ClientRuntimeSnapshot。
export function buildSnapshot() {
  const panels = clientPanels.map(p => p.id)
  const plugins = instances.map(inst => ({
    name: inst.name,
    status: inst.status,
    version: inst.defId || '',
    ...(inst.error ? { error: inst.error } : {}),
    ...(inst.events && inst.events.length ? { events: [...inst.events] } : {}),
  }))
  // 面板归属并入对应插件
  for (const p of plugins) {
    const mine = clientPanels.filter(cp => cp.pluginName === p.name).map(cp => cp.id)
    if (mine.length) p.panels = mine
    const mineSlots = clientSlots.filter(cs => cs.pluginName === p.name).map(cs => cs.slotId)
    if (mineSlots.length) p.slots = mineSlots
  }
  const slots = clientSlots.map(s => s.slotId)
  return { plugins, ...(panels.length ? { panels } : {}), ...(slots.length ? { slots } : {}) }
}

// reportState 上报快照（节流：轮询周期内自动去重，无需独立节流）。
export async function reportState() {
  try {
    await api.pluginClientState(buildSnapshot())
  } catch (e) {
    // 静默：后端未就绪
  }
}

// stopPolling 停止轮询。
export function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// getInstances 运行中的 client 半实例（供面板展示）。
export function getInstances() {
  return instances.map(x => ({ name: x.name, defId: x.defId, source: x.source }))
}

// getUIFor 按插件名取 client 半的沙箱 ui 对象（面板渲染 render(el, ui) 用；
// 未装载返回 undefined）。ui 含 on/emit/invoke/reportFailure/registerPanel/http/log。
export function getUIFor(pluginName) {
  const inst = instances.find(x => x.name === pluginName)
  return inst ? inst.ui : undefined
}

export default {
  loadClientHalf, unloadClientHalf, syncClientHalves,
  startPolling, stopPolling, dispatchHostEvent,
  getInstances, setPanelMount, clientPanels,
  clientSlots, setSlotMount, getSlotCandidates, getSlotOwner, setSlotOwner, getSlotUI, getSlotUIList,
  emitSlotChanged, isOverlayActive, setOverlayActive, mountListSlot,
}

// ★ 调试/验证暴露（生产保留，无害）：window.__pluginRuntime 供浏览器控制台
// 与自动化（web_debug）检查 client 半装载/槽位注册实时状态。
if (typeof window !== 'undefined') {
  window.__pluginRuntime = {
    instances: () => instances.map(i => ({ name: i.name, status: i.status, error: i.error || '' })),
    clientSlots: () => clientSlots.map(s => ({ slotId: s.slotId, pluginName: s.pluginName, title: s.title, hasRender: typeof s.render === 'function' })),
    clientPanels: () => clientPanels.map(p => ({ id: p.id, pluginName: p.pluginName })),
    getSlotOwner,
  }
}
