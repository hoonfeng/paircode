// ─── REST API ────────────────────────────────────────────────

const BASE = '/api'

function apiURL(path, params = {}) {
  // 确保 path 以 / 开头，避免 BASE + path 拼成 /apitools（应为 /api/tools）
  if (!path.startsWith('/')) path = '/' + path
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

async function apiPost(path, body = {}, params = {}) {
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
const WS_MAX_RECONNECT = 20  // 从 5 提升到 20，减少因短暂网络波动导致永久断连
let wsCallbacks = null
let wsManuallyClosed = false
let wsPongTimer = null  // 检测后端 ping 超时：45s 无响应则主动重连（从 90s 缩短）

// initWebSocket 建立全局 WebSocket 连接，接收所有会话事件。
// callbacks: { onStatus(payload), onEvent(convId, data), onDone(convId, data) }
//   - onStatus(payload): payload = { runningConvs: [...], runningByWorkspace: {wsRoot: count} }
// 连接断开时自动重连（指数退避，最多 5 次）。
function initWebSocket(callbacks) {
  wsCallbacks = callbacks
  wsManuallyClosed = false
  wsReconnectCount = 0  // 新开连接时重置计数
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
    window.dispatchEvent(new CustomEvent('ws-connection-change', { detail: { connected: true } }))
    // 后端每 30s 发 ping，45s（1.5次）未收到则主动重连
    if (wsPongTimer) clearTimeout(wsPongTimer)
    wsPongTimer = setTimeout(() => {
      console.warn('[WS] 45s 未收到 pong，触发重连')
      wsSocket.close()
    }, 45000)
  }
  wsSocket.onmessage = (ev) => {
    // 收到任何消息都重置 pong 超时（后端 ping 帧也会触发 onmessage）
    if (wsPongTimer) { clearTimeout(wsPongTimer) }
    wsPongTimer = setTimeout(() => {
      console.warn('[WS] 45s 无消息，触发重连')
      if (wsSocket) wsSocket.close()
    }, 45000)
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
    window.dispatchEvent(new CustomEvent('ws-connection-change', { detail: { connected: false } }))
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
    // ★ 通知前端所有 running 会话已中断（后端进程已不在）
    if (wsCallbacks?.onDisconnected) {
      wsCallbacks.onDisconnected()
    }
    return
  }
  wsReconnectCount++
  // 更快重连：初始 500ms，最大 5s（原策略 1s→8s）
  const delay = Math.min(500 * Math.pow(1.5, wsReconnectCount - 1), 5000)
  console.warn('[WS] ' + reason + '，' + delay + 'ms 后重连 (' + wsReconnectCount + '/' + WS_MAX_RECONNECT + ')')
  if (wsReconnectTimer) clearTimeout(wsReconnectTimer)
  wsReconnectTimer = setTimeout(() => { doWsConnect() }, delay)
}

// 主动重启 WebSocket 连接（工作区切换、检测到连接异常等场景）
// 与 closeWebSocket 不同，此函数会在关闭后立即重建连接并恢复回调。
function reconnectWebSocket() {
  if (wsSocket) {
    wsSocket.onclose = null  // 防止触发自动重连
    wsSocket.close()
    wsSocket = null
  }
  if (wsReconnectTimer) { clearTimeout(wsReconnectTimer); wsReconnectTimer = null }
  if (wsPongTimer) { clearTimeout(wsPongTimer); wsPongTimer = null }
  wsReconnectCount = 0
  wsManuallyClosed = false
  doWsConnect()
}

function closeWebSocket() {
  wsManuallyClosed = true
  if (wsReconnectTimer) { clearTimeout(wsReconnectTimer); wsReconnectTimer = null }
  if (wsPongTimer) { clearTimeout(wsPongTimer); wsPongTimer = null }
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

// 等待 WebSocket 连接就绪（用于发送消息前确保能接收 WS 事件）。
// 若已连接立即返回 true；若正在连接则等待最多 timeout ms；
// 若断开则触发重连并等待。返回是否就绪。
async function waitForWebSocket(timeout = 3000) {
  if (wsSocket && wsSocket.readyState === WebSocket.OPEN) return true
  if (wsManuallyClosed) return false
  // 如果断开或从未连接，触发连接
  if (!wsSocket || wsSocket.readyState === WebSocket.CLOSED) {
    wsReconnectCount = 0
    doWsConnect()
  }
  // 等待 CONNECTING → OPEN
  const deadline = Date.now() + timeout
  while (Date.now() < deadline) {
    if (wsSocket && wsSocket.readyState === WebSocket.OPEN) return true
    await new Promise(r => setTimeout(r, 100))
  }
  return !!(wsSocket && wsSocket.readyState === WebSocket.OPEN)
}

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

// 回滚到指定用户消息前的状态：恢复文件快照 + 删除后续对话历史
async function chatRollback(convId, msgIdx) {
  return apiPost('/chat/rollback', { convId, msgIdx })
}

// 请求当前运行中的对话在下一轮迭代压缩上下文
async function chatCompact(convId) {
  return apiPost('/chat/compact?convId=' + encodeURIComponent(convId), {})
}

// ─── 对话消息懒加载 ──────────────────────────────────────────

// 获取对话消息（分页）：默认拉最新 limit 条；传 before 时向前翻页
// 返回 { messages: [...], total: N }，messages 中每条含 segments 字段
async function getMessages(convId, { limit = 50, before = null } = {}) {
  const params = { limit }
  if (before !== null && before !== undefined) params.before = before
  return apiGet('/conversations/' + encodeURIComponent(convId) + '/messages', params)
}

// 获取对话消息总数
async function getMessagesCount(convId) {
  return apiGet('/conversations/' + encodeURIComponent(convId) + '/messages/count')
}

// ─── 模型列表 ──────────────────────────────────────────────

async function getModels() {
  return apiGet('/models')
}

// ─── MCP 配置（从后端 API 获取，不再用 localStorage） ────────

async function getMcpList(level = 'all') {
  return apiGet('/mcp/list', { level })
}

// 保存单个 MCP 配置（action: save/delete）
async function saveMcpItem({ action, name, command, args, level }) {
  return apiPost('/mcp/save', { action, name, command, args: args || [], level: level || 'user' })
}

// ─── Skills API ────────────────────────────────────────────

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

export default { apiGet, apiPost, apiPut, apiDelete, initWebSocket, reconnectWebSocket, closeWebSocket, isWebSocketOpen, waitForWebSocket, chatStart, answerChat, approveChat, sendFeedback, chatRollback, chatCompact, chatStop, getMessages, getMessagesCount, getModels, getMcpList, saveMcpItem, getSkillsList, readSkill, deleteSkill, saveSkillStatus, getInstructions, saveInstructions, getPhilosophy, savePhilosophy }