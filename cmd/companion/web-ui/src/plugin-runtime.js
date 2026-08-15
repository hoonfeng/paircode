// ═══════════════════════════════════════════════════════════════
// plugin-runtime.js — 浏览器侧插件 client 半运行时（host/client 双半的 client 侧）
//
// 契约（与后端 DefineJSCodeFull / cordis_define 的 client 参数一致）：
//   client 半形态：(ui) => void
//     ui.on(event, fn)            收 host → 浏览器事件（ui:/client: 前缀）
//     ui.emit(event, payload)     发事件回 host（host: 前缀给 host 插件消费）
//     ui.registerPanel({id,title,icon,render}) 注册自定义面板（渲染进插件面板 client 区）
//     ui.http.get(path)/post(path, body)       受限后端 API 调用
//     ui.log(...)                 日志（控制台）
//
// 事件流：
//   host → 浏览器：EventBus 里 ui:/client: 前缀事件入 host 队列，本模块每 2s
//                 轮询 /api/plugins/client-events 取增量，分发给各 client 半的 on 监听器。
//   浏览器 → host：ui.emit 调 POST /api/plugins/event（host 插件 ctx.on('host:xxx') 消费）。
//
// 面板渲染：client 半注册的面板显示在插件面板的「客户端面板」区；render(el) 在
// 挂载时调用，el 为容器 DOM 元素（可自由操作 DOM）。
// ═══════════════════════════════════════════════════════════════

import api from './api.js'

// ─── 运行中的 client 半实例 ──────────────────────────────────
// instances: [{ name, defId, source, onHandlers: Map<event, [fn]> }]
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
      return () => {
        const arr = inst.onHandlers.get(event) || []
        const i = arr.indexOf(fn)
        if (i >= 0) arr.splice(i, 1)
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
    onHandlers: new Map(),
  }
  try {
    fn(makeUI(inst))
  } catch (e) {
    console.warn('[plugin] client 半执行错误', source.name, e)
    return null
  }
  instances.push(inst)
  // 重新装载时同步一次面板
  emitPanelChanged()
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
    emitPanelChanged()
  }
}

// syncClientHalves 与后端插件列表对齐：装载新出现的 client 半，卸载已消失的。
export async function syncClientHalves(plugins) {
  if (!plugins || !Array.isArray(plugins)) return
  const active = new Set()
  for (const p of plugins) {
    if (p.hasClient && p.state === 'running' && p.clientCode) {
      active.add(p.name)
      const exists = instances.find(inst => inst.name === p.name)
      if (!exists) {
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
  emitPanelChanged()
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

// startPolling 启动事件轮询（每 2s 一次）。
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
  }, pollInterval)
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

export default {
  loadClientHalf, unloadClientHalf, syncClientHalves,
  startPolling, stopPolling, dispatchHostEvent,
  getInstances, setPanelMount, clientPanels,
}
