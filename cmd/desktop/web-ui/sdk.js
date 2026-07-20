// ─── 双模式通信 SDK ───────────────────────────────────────────
// 统一前端 API 调用和事件通道，屏蔽 Web 模式（HTTP+WS）与桌面模式（Go 直调）的差异。
//
// 检测方式：
//   Web 模式：window.__DESKTOP_MODE__ 不存在或为 false
//   桌面模式：window.__DESKTOP_MODE__ = true，通过 window.desktopBridge 调用 Go
//
// desktopBridge 约定：
//   - call(method, path, bodyJSON?, paramsJSON?) → Promise<JSON>
//     对应 REST 语义：GET/POST/PUT/DELETE
//   - onAgentEvent(convId, data)  由 Go 端主动调用 → 推送 agent 事件
//   - onStatus(payload)           由 Go 端主动调用 → 推送运行状态
// ─────────────────────────────────────────────────────────────

const isDesktop = typeof window !== 'undefined' && window.__DESKTOP_MODE__ === true
const bridge = isDesktop ? (window.desktopBridge || null) : null

function debug(...args) {
  if (isDesktop) {
    console.log('[SDK:desktop]', ...args)
  }
}

// ─── HTTP 基础（Web 模式） ──────────────────────────────────

const BASE = '/api'

function apiURL(path, params = {}) {
  const u = new URL(BASE + path, location.origin)
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') u.searchParams.set(k, v)
  }
  return u.toString()
}

async function httpGet(path, params = {}) {
  const r = await fetch(apiURL(path, params))
  if (!r.ok) {
    const e = await r.json().catch(() => ({ error: r.statusText }))
    throw new Error(e.error || e.message || r.statusText)
  }
  return r.json()
}

async function httpPost(path, body = {}, params = {}) {
  const r = await fetch(apiURL(path, params), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!r.ok) {
    const e = await r.json().catch(() => ({ error: r.statusText }))
    throw new Error(e.error || e.message || r.statusText)
  }
  return r.json()
}

async function httpPut(path, body = {}) {
  const r = await fetch(apiURL(path), {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!r.ok) {
    const e = await r.json().catch(() => ({ error: r.statusText }))
    throw new Error(e.error || e.message || r.statusText)
  }
  return r.json()
}

async function httpDelete(path) {
  const r = await fetch(apiURL(path), { method: 'DELETE' })
  if (!r.ok) {
    const e = await r.json().catch(() => ({ error: r.statusText }))
    throw new Error(e.error || e.message || r.statusText)
  }
  return r.json()
}

// ─── 桌面桥接调用 ──────────────────────────────────────────

async function bridgeCall(method, path, body = null, params = {}) {
  if (!bridge) {
    throw new Error('[SDK] 桌面模式但 desktopBridge 未初始化')
  }
  try {
    const bodyJSON = body !== null && body !== undefined ? JSON.stringify(body) : ''
    const paramsJSON = Object.keys(params).length > 0 ? JSON.stringify(params) : ''
    const result = bridge.call(method, path, bodyJSON, paramsJSON)
    // bridge.call 可能返回 Promise 或同步值
    const str = await Promise.resolve(result)
    if (!str || str === '') return null
    return JSON.parse(str)
  } catch (e) {
    throw new Error('[SDK] 桌面调用失败: ' + (e.message || e))
  }
}

// ─── 统一 API 调用 ──────────────────────────────────────────

async function apiGet(path, params = {}) {
  if (isDesktop) {
    return bridgeCall('GET', path, null, params)
  }
  return httpGet(path, params)
}

async function apiPost(path, body = {}, params = {}) {
  if (isDesktop) {
    return bridgeCall('POST', path, body, params)
  }
  return httpPost(path, body, params)
}

async function apiPut(path, body = {}) {
  if (isDesktop) {
    return bridgeCall('PUT', path, body)
  }
  return httpPut(path, body)
}

async function apiDelete(path) {
  if (isDesktop) {
    return bridgeCall('DELETE', path)
  }
  return httpDelete(path)
}

// ─── 事件通道（WebSocket / 桌面桥接回调） ────────────────────
//
// 统一接口：
//   initChannel(callbacks)   — 建立事件连接
//   closeChannel()           — 关闭事件连接
//   isChannelOpen()          — 连接是否打开
//
// callbacks: { onStatus(payload), onEvent(convId, data), onDone(convId, data) }
// ─────────────────────────────────────────────────────────────

// ── Web 模式：WebSocket ──

let wsSocket = null
let wsReconnectTimer = null
let wsReconnectCount = 0
const WS_MAX_RECONNECT = 5
let wsCallbacks = null
let wsManuallyClosed = false

function doWsConnect() {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = proto + '//' + location.host + '/ws'
  try {
    wsSocket = new WebSocket(url)
  } catch (e) {
    scheduleWsReconnect('WebSocket 创建失败: ' + (e.message || e))
    return
  }
  wsSocket.onopen = () => {
    wsReconnectCount = 0
    if (wsReconnectTimer) { clearTimeout(wsReconnectTimer); wsReconnectTimer = null }
  }
  wsSocket.onmessage = (ev) => {
    let data
    try { data = JSON.parse(ev.data) } catch { return }
    if (!data) return
    if (data.type === 'status' && data.runningConvs) {
      wsCallbacks?.onStatus?.({
        runningConvs: data.runningConvs,
        runningByWorkspace: data.runningByWorkspace || {},
      })
      return
    }
    const convId = data.convId
    if (!convId) return
    if (data.type === 'done') {
      wsCallbacks?.onDone?.(convId, data)
    } else {
      wsCallbacks?.onEvent?.(convId, data)
    }
  }
  wsSocket.onclose = () => {
    wsSocket = null
    if (!wsManuallyClosed) scheduleWsReconnect('连接已关闭')
  }
  wsSocket.onerror = () => {
    // onclose 会随后触发
  }
}

function scheduleWsReconnect(reason) {
  if (wsManuallyClosed) return
  if (wsReconnectCount >= WS_MAX_RECONNECT) {
    console.warn('[SDK:WS] 重连已达上限:', reason)
    return
  }
  wsReconnectCount++
  const delay = Math.min(1000 * Math.pow(2, wsReconnectCount - 1), 8000)
  console.warn('[SDK:WS] ' + reason + '，' + delay + 'ms 后重连 (' + wsReconnectCount + '/' + WS_MAX_RECONNECT + ')')
  if (wsReconnectTimer) clearTimeout(wsReconnectTimer)
  wsReconnectTimer = setTimeout(() => { doWsConnect() }, delay)
}

function initWebChannel(callbacks) {
  wsCallbacks = callbacks
  wsManuallyClosed = false
  if (wsSocket && (wsSocket.readyState === WebSocket.OPEN || wsSocket.readyState === WebSocket.CONNECTING)) {
    return
  }
  doWsConnect()
}

function closeWebChannel() {
  wsManuallyClosed = true
  if (wsReconnectTimer) { clearTimeout(wsReconnectTimer); wsReconnectTimer = null }
  wsCallbacks = null
  wsReconnectCount = WS_MAX_RECONNECT
  if (wsSocket) {
    wsSocket.onclose = null
    wsSocket.close()
    wsSocket = null
  }
}

function isWebChannelOpen() {
  return !!(wsSocket && wsSocket.readyState === WebSocket.OPEN)
}

// ── 桌面模式：桥接回调 ──

let desktopCallbacks = null

function initDesktopChannel(callbacks) {
  desktopCallbacks = callbacks
  if (!bridge) return
  debug('初始化桌面事件通道')

  // Go 端通过设置 bridge 上的回调来推送事件
  bridge.onAgentEvent = (convId, dataJSON) => {
    try {
      const data = typeof dataJSON === 'string' ? JSON.parse(dataJSON) : dataJSON
      if (data.type === 'done') {
        desktopCallbacks?.onDone?.(convId, data)
      } else {
        desktopCallbacks?.onEvent?.(convId, data)
      }
    } catch (e) {
      console.error('[SDK] 桌面事件解析失败:', e)
    }
  }

  bridge.onStatus = (payloadJSON) => {
    try {
      const payload = typeof payloadJSON === 'string' ? JSON.parse(payloadJSON) : payloadJSON
      desktopCallbacks?.onStatus?.({
        runningConvs: payload.runningConvs || [],
        runningByWorkspace: payload.runningByWorkspace || {},
      })
    } catch (e) {
      console.error('[SDK] 桌面状态解析失败:', e)
    }
  }
}

function closeDesktopChannel() {
  desktopCallbacks = null
  if (bridge) {
    bridge.onAgentEvent = null
    bridge.onStatus = null
  }
}

function isDesktopChannelOpen() {
  return desktopCallbacks !== null
}

// ── 统一事件通道 ──

function initChannel(callbacks) {
  if (isDesktop) {
    initDesktopChannel(callbacks)
  } else {
    initWebChannel(callbacks)
  }
}

function closeChannel() {
  if (isDesktop) {
    closeDesktopChannel()
  } else {
    closeWebChannel()
  }
}

function isChannelOpen() {
  if (isDesktop) {
    return isDesktopChannelOpen()
  }
  return isWebChannelOpen()
}

// ─── 业务 API ──────────────────────────────────────────────

async function chatStart(convId, message, autonomous, workspaceRoot) {
  const body = { convId, message, autonomous }
  if (workspaceRoot) body.workspaceRoot = workspaceRoot
  return apiPost('/chat/send', body)
}

async function chatStop(convId) {
  return apiPost('/chat/stop?convId=' + encodeURIComponent(convId), {})
}

async function answerChat(convId, answer) {
  return apiPost('/chat/answer', { convId, answer })
}

async function approveChat(convId, approved) {
  return apiPost('/chat/approve', { convId, approved })
}

async function sendFeedback(convId, content) {
  return apiPost('/chat/feedback', { convId, content })
}

async function chatRollback(convId, msgIdx) {
  return apiPost('/chat/rollback', { convId, msgIdx })
}

async function getMessages(convId, { limit = 50, before = null } = {}) {
  const params = { limit }
  if (before !== null && before !== undefined) params.before = before
  return apiGet('/conversations/' + encodeURIComponent(convId) + '/messages', params)
}

async function getMessagesCount(convId) {
  return apiGet('/conversations/' + encodeURIComponent(convId) + '/messages/count')
}

async function getModels() {
  return apiGet('/models')
}

async function getMcpList(level = 'all') {
  return apiGet('/mcp/list', { level })
}

async function saveMcpItem({ action, name, command, args, level }) {
  return apiPost('/mcp/save', { action, name, command, args: args || [], level: level || 'user' })
}

async function getSkillsList() {
  return apiGet('/skills/list')
}

async function readSkill(name, level) {
  const params = { name }
  if (level) params.level = level
  return apiGet('/skills/read', params)
}

async function deleteSkill(name) {
  return apiPost('/skills/delete', { name })
}

async function saveSkillStatus(name, level, status) {
  return apiPost('/skills/save', { action: 'set-status', name, level, status })
}

async function getInstructions(scope = 'system') {
  return apiGet('/instructions', { scope })
}

async function saveInstructions(scope, content) {
  return apiPut('/instructions' + '?scope=' + scope, { content })
}

async function getPhilosophy() {
  return apiGet('/philosophy')
}

async function savePhilosophy(data) {
  return apiPut('/philosophy', data)
}

// ─── 导出 ──────────────────────────────────────────────────
export default {
  // 运行时检测（供外部判断环境用）
  isDesktop,
  bridge,

  // 底层 API
  apiGet, apiPost, apiPut, apiDelete,

  // 事件通道
  initChannel, closeChannel, isChannelOpen,

  // 业务 API
  chatStart, chatStop, answerChat, approveChat, sendFeedback, chatRollback,
  getMessages, getMessagesCount,
  getModels,
  getMcpList, saveMcpItem,
  getSkillsList, readSkill, deleteSkill, saveSkillStatus,
  getInstructions, saveInstructions,
  getPhilosophy, savePhilosophy,
}
