// ─── REST API ────────────────────────────────────────────────

const BASE = '/api'

function apiURL(path, params = {}) {
  const u = new URL(BASE + path, location.origin)
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') u.searchParams.set(k, v)
  }
  return u.toString()
}

async function apiGet(path, params = {}) {
  const r = await fetch(apiURL(path, params))
  if (!r.ok) {
    const e = await r.json().catch(() => ({ error: r.statusText }))
    throw new Error(e.error || e.message || r.statusText)
  }
  return r.json()
}

async function apiPost(path, body = {}) {
  const r = await fetch(apiURL(path), {
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

async function apiPut(path, body = {}) {
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

async function apiDelete(path) {
  const r = await fetch(apiURL(path), { method: 'DELETE' })
  if (!r.ok) {
    const e = await r.json().catch(() => ({ error: r.statusText }))
    throw new Error(e.error || e.message || r.statusText)
  }
  return r.json()
}

// ─── WebSocket（替代 SSE）：单一全局连接推送所有会话事件 ────

let wsSocket = null
let wsReconnectTimer = null
let wsReconnectCount = 0
const WS_MAX_RECONNECT = 5
let wsCallbacks = null
let wsManuallyClosed = false

// initWebSocket 建立全局 WebSocket 连接，接收所有会话事件。
// callbacks: { onStatus(payload), onEvent(convId, data), onDone(convId, data) }
//   - onStatus(payload): payload = { runningConvs: [...], runningByWorkspace: {wsRoot: count} }
// 连接断开时自动重连（指数退避，最多 5 次）。
function initWebSocket(callbacks) {
  wsCallbacks = callbacks
  wsManuallyClosed = false
  if (wsSocket && (wsSocket.readyState === WebSocket.OPEN || wsSocket.readyState === WebSocket.CONNECTING)) {
    return
  }
  doWsConnect()
}

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
    // onclose 会随后触发，重连由 onclose 处理
  }
}

function scheduleWsReconnect(reason) {
  if (wsManuallyClosed) return
  if (wsReconnectCount >= WS_MAX_RECONNECT) {
    console.warn('[WS] 重连已达上限:', reason)
    return
  }
  wsReconnectCount++
  const delay = Math.min(1000 * Math.pow(2, wsReconnectCount - 1), 8000)
  console.warn('[WS] ' + reason + '，' + delay + 'ms 后重连 (' + wsReconnectCount + '/' + WS_MAX_RECONNECT + ')')
  if (wsReconnectTimer) clearTimeout(wsReconnectTimer)
  wsReconnectTimer = setTimeout(() => { doWsConnect() }, delay)
}

function closeWebSocket() {
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

function isWebSocketOpen() {
  return !!(wsSocket && wsSocket.readyState === WebSocket.OPEN)
}

// ─── 多会话并行：非阻塞启动 + WebSocket 事件流 ──────────────

// 非阻塞启动指定对话的 agent（后端返回 {ok, convId}）
async function chatStart(convId, message, autonomous, workspaceRoot) {
  const body = { convId, message, autonomous }
  if (workspaceRoot) body.workspaceRoot = workspaceRoot
  return apiPost('/chat/send', body)
}

// 停止指定对话的 agent
async function chatStop(convId) {
  return apiPost('/chat/stop?convId=' + encodeURIComponent(convId), {})
}

// 回答 ask_user 问题
async function answerChat(convId, answer) {
  return apiPost('/chat/answer', { convId, answer })
}

// 审批写工具
async function approveChat(convId, approved) {
  return apiPost('/chat/approve', { convId, approved })
}

// 运行时反馈：Agent 执行中用户可补充/纠正
async function sendFeedback(convId, content) {
  return apiPost('/chat/feedback', { convId, content })
}

// ─── 模型列表 ──────────────────────────────────────────────

async function getModels() {
  return apiGet('/models')
}

// ─── 指令管理 ──────────────────────────────────────────────

async function getInstructions(scope = 'system') {
  return apiGet('/instructions', { scope })
}

async function saveInstructions(scope, content) {
  return apiPut('/instructions' + '?scope=' + scope, { content })
}

// ─── 思想配置 ──────────────────────────────────────────────

async function getPhilosophy() {
  return apiGet('/philosophy')
}

async function savePhilosophy(data) {
  return apiPut('/philosophy', data)
}

export default { apiGet, apiPost, apiPut, apiDelete, initWebSocket, closeWebSocket, isWebSocketOpen, chatStart, answerChat, approveChat, sendFeedback, chatStop, getModels, getInstructions, saveInstructions, getPhilosophy, savePhilosophy }
