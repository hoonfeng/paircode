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
// ═══════════════════════════════════════════════════════════════

import api from './api.js'

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
// 宿主预定义 slotId：'statusbar'（底部状态栏）。
// 占用者选择（getSlotOwner/setSlotOwner）持久化 localStorage；
// owner = '' 表示使用内置组件。
export const clientSlots = []
const slotOwnerKey = (id) => 'paircode-slot-' + id
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
function emitSlotChanged() {
  for (const fn of slotMountFns) {
    try { fn(clientSlots) } catch (e) { console.warn('[slot] 通知失败', e) }
  }
}

// getSlotCandidates 某槽位的全部候选（占用者）。
export function getSlotCandidates(slotId) {
  return clientSlots.filter(s => s.slotId === slotId)
}
// getSlotOwner 当前激活占用者（'' = 内置组件）。
export function getSlotOwner(slotId) {
  let v = ''
  try { v = localStorage.getItem(slotOwnerKey(slotId)) || '' } catch (e) { /* 忽略 */ }
  if (v && !clientSlots.some(s => s.slotId === slotId && s.pluginName === v)) v = ''
  return v
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
    // 注册 UI 槽位占用（Slot 系统：替换宿主预定义界面区域，如 'statusbar'）。
    // 同插件重复注册同槽位 → 替换。宿主按 getSlotOwner 决定激活哪个占用者，
    // 激活后调 render(el, ui)；render 可返回 cleanup 函数（宿主下次重渲染前调用）。
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
    fn = new Function('ui', '"use strict";\n' + code)
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
  clientSlots, setSlotMount, getSlotCandidates, getSlotOwner, setSlotOwner, getSlotUI,
}
