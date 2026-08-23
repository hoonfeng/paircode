// ─── REST API ────────────────────────────────────────────────

const BASE = '/api'
const API_TIMEOUT = 30000 // ★ 2026-08-21 请求超时：后端卡死时前端不无限等待（AbortController）

function apiURL(path, params = {}) {

  // 确保 path 以 / 开头，避免 BASE + path 拼成 /apitools（应为 /api/tools）

  if (!path.startsWith('/')) path = '/' + path

  const u = new URL(BASE + path, location.origin)

  for (const [k, v] of Object.entries(params)) {

    if (v !== undefined && v !== null && v !== '') u.searchParams.set(k, v)

  }

  return u.toString()

}

async function apiGet(path, params = {}, opts = {}) {
  const r = await fetch(apiURL(path, params), { signal: apiSignal(opts) })
  if (!r.ok) {
    const e = await r.json().catch(() => ({ error: r.statusText }))
    throw new Error(e.error || e.message || r.statusText)
  }
  return r.json()
}

async function apiPost(path, body = {}, params = {}, opts = {}) {
  const r = await fetch(apiURL(path, params), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    signal: apiSignal(opts),
  })
  if (!r.ok) {
    const e = await r.json().catch(() => ({ error: r.statusText }))
    throw new Error(e.error || e.message || r.statusText)
  }
  return r.json()
}

async function apiPut(path, body = {}, opts = {}) {
  const r = await fetch(apiURL(path), {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    signal: apiSignal(opts),
  })
  if (!r.ok) {
    const e = await r.json().catch(() => ({ error: r.statusText }))
    throw new Error(e.error || e.message || r.statusText)
  }
  return r.json()
}

async function apiDelete(path, opts = {}) {
  const r = await fetch(apiURL(path), { method: 'DELETE', signal: apiSignal(opts) })
  if (!r.ok) {
    const e = await r.json().catch(() => ({ error: r.statusText }))
    throw new Error(e.error || e.message || r.statusText)
  }
  return r.json()
}

// apiSignal 构造超时信号：opts.timeout（毫秒，默认 30s）；opts.signal 可传入外部取消信号。
function apiSignal(opts = {}) {
  const timeout = opts.timeout || API_TIMEOUT
  const ctrl = new AbortController()
  const timer = setTimeout(() => ctrl.abort(new Error('请求超时(' + (timeout / 1000) + 's)')), timeout)
  if (opts.signal) {
    opts.signal.addEventListener('abort', () => ctrl.abort(), { once: true })
  }
  return ctrl.signal
}

// ─── WebSocket（替代 SSE）：单一全局连接推送所有会话事件 ────

let wsSocket = null

let wsReconnectTimer = null

let wsReconnectCount = 0

const WS_MAX_RECONNECT = 20  // 从 5 提升到 20，减少因短暂网络波动导致永久断连

let wsCallbacks = null

let wsManuallyClosed = false

let wsPongTimer = null  // 检测后端 ping 超时：45s 无响应则主动重连（从 90s 缩短）
// ★ 运行中会话集合（onStatus 更新）：agent 运行期间连接健康由后端 30s 文本
//   ping 保证（触发 onmessage 重置定时器）；此集合作为「无业务消息不断开」的
//   双保险——LLM 思考/重试可能长时间无业务事件，若此时因无消息断开将丢失事件。
let wsRunningConvs = null

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

      // ★ 双保险：agent 运行中（有 running 会话）不因无业务消息断开——

      //   LLM 思考/重试期间可能长时间无事件，断开将导致事件丢失（无响应）。

      //   连接健康由后端 30s 文本 ping 持续重置本定时器保证；只有 ping 也

      //   停止（后端假死/网络断开）时才会走到这里。

      if (wsRunningConvs && wsRunningConvs.size > 0) {

        console.warn('[WS] 45s 无业务消息但 agent 运行中，保持连接（等待后端 ping）')

        return

      }

      console.warn('[WS] 45s 无消息，触发重连')

      if (wsSocket) wsSocket.close()

    }, 45000)

    let data

    try { data = JSON.parse(ev.data) } catch { return }

    if (!data) return

    // ★ 2026-08-21 WS 断线补偿：服务端重连时推送批量快照数组
    //   [{convId, type:'snapshot', content, reasoning, toolSegments:[...]}]
    if (Array.isArray(data)) {
      for (const snap of data) {
        if (snap && snap.type === 'snapshot' && snap.convId) {
          wsCallbacks?.onEvent?.(snap.convId, snap)
        }
      }
      return
    }

    if (data.type === 'ping') {

      // 后端心跳帧：仅用于连接健康检测，无业务含义

      return

    }

    if (data.type === 'status' && data.runningConvs) {

      wsRunningConvs = new Set(data.runningConvs)

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
async function chatStart(convId, message, autonomous, workspaceRoot, images) {
  const body = { convId, message, autonomous }
  if (workspaceRoot) body.workspaceRoot = workspaceRoot
  if (images && images.length > 0) body.images = images // ★ 2026-08-21 多模态：结构化图片数组（{data,mimeType,detail}）
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

// ★ 2026-08-23 工作区隔离：workspaceRoot 参数透传（对话消息回放按所属工作区路由，
// 运行中切换工作区后回放旧工作区对话不再落到新工作区存储）

async function getMessages(convId, { limit = 50, before = null, workspaceRoot = '' } = {}) {

  const params = { limit }

  if (before !== null && before !== undefined) params.before = before

  if (workspaceRoot) params.workspaceRoot = workspaceRoot

  return apiGet('/conversations/' + encodeURIComponent(convId) + '/messages', params)

}

// 获取对话消息总数

async function getMessagesCount(convId, workspaceRoot = '') {

  const params = {}

  if (workspaceRoot) params.workspaceRoot = workspaceRoot

  return apiGet('/conversations/' + encodeURIComponent(convId) + '/messages/count', params)

}

// ─── 模型列表 ──────────────────────────────────────────────

async function getModels() {

  return apiGet('/models')

}

// saveModels 全量保存服务商与模型列表（providers 快照 → config/models.json）
async function saveModels(providers) {

  return apiPost('/models', { providers })

}

// getAiPresets 获取 AI 配置预设列表（预设名 → 完整配置快照；config/ai-presets.json）
async function getAiPresets() {

  return apiGet('/ai-presets')

}

// saveAiPreset 保存一条预设（action: save/apply/delete/rename）
async function saveAiPreset(action, name, preset) {

  const body = { action, name }
  if (preset) body.preset = preset
  return apiPost('/ai-presets', body)

}

// saveAiPresets 全量保存 AI 配置预设（presets → config/ai-presets.json）
async function saveAiPresets(presets) {

  return apiPut('/ai-presets', { presets })

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

export default { apiGet, apiPost, apiPut, apiDelete, initWebSocket, reconnectWebSocket, closeWebSocket, isWebSocketOpen, waitForWebSocket, chatStart, answerChat, approveChat, sendFeedback, chatRollback, chatCompact, chatStop, getMessages, getMessagesCount, getModels, saveModels, getAiPresets, saveAiPreset, saveAiPresets, getMcpList, saveMcpItem, getSkillsList, readSkill, deleteSkill, saveSkillStatus, getInstructions, saveInstructions, listPlugins, getPluginDetail, pluginAction, definePlugin, pluginEmit, pluginClientEvents, pluginClientState, pluginInvoke, pluginClientFailure, builtinPlugins, pluginToolToggle, getToolsets, toolsetEdit }

// ─── 插件（管理 + 使用 + host/client 事件桥）──────────────

// listPlugins 插件列表（含状态/工具/服务/client 有无，不含源码）。

async function listPlugins() {

  return apiGet('/plugins')

}

// builtinPlugins 内置工具包（被过滤工具按内置插件组管理——插件面板开关）：

// GET 返回 {groups, joined, toolTotal, enabledTotal}；

// POST 切换分组 {group, enabled} 或强制全部 {forceAll:true}。

async function builtinPlugins(data, workspaceRoot) {

  // ★ workspaceRoot：目标工作区（工作区隔离——管理弹窗按当前工作区操作，缺省主工作区）

  const ws = workspaceRoot || ''

  if (data) {

    const body = { ...data }

    if (ws) body.workspaceRoot = ws

    return apiPost('/plugins/builtin', body)

  }

  return apiGet('/plugins/builtin', ws ? { workspaceRoot: ws } : undefined)

}

// pluginToolToggle 通用工具级开关（任意已注册工具，agent 可见性）：{tool, enabled}。

async function pluginToolToggle(tool, enabled) {

  return apiPost('/plugins/tool', { tool, enabled })

}

// getPluginDetail 单插件详情（?id= 插件名或 dyn id，含 client 半源码）。

async function getPluginDetail(id) {

  return apiGet('/plugins/detail', { id })

}

// pluginAction 启停/删除：action = start|stop|undefine。

async function pluginAction(id, action) {

  return apiPost('/plugins/action', { id, action })

}

// definePlugin 直接定义 JS 动态插件：{purpose, code, client?, language?, run?}。

async function definePlugin(data) {

  return apiPost('/plugins/define', data)

}

// pluginEmit 浏览器 client 半 → host 事件桥（{event, payload}）。

async function pluginEmit(event, payload) {

  return apiPost('/plugins/event', { event, payload })

}

// pluginClientEvents host → 浏览器事件轮询（since=seq）。

async function pluginClientEvents(since) {

  return apiGet('/plugins/client-events', { since })

}

// pluginClientState 浏览器上报/读取 client 半运行快照（client inspect provider 数据源）。

async function pluginClientState(snapshot) {

  if (snapshot) return apiPost('/plugins/client-state', snapshot)

  return apiGet('/plugins/client-state')

}

// pluginInvoke 浏览器 client 半远程调用 host 半注册方法（D11 invoke RPC）。

// {plugin, method, args} → {ok, value} 或 {ok:false, error}。

async function pluginInvoke(plugin, method, args) {

  return apiPost('/plugins/invoke', { plugin, method, args: args === undefined ? null : args })

}

// pluginClientFailure 上报 client 半失败（render/guard/boot 阶段；供 Agent inspect 修复）。

async function pluginClientFailure(plugin, phase, message) {

  return apiPost('/plugins/client-failure', { plugin, phase, message })

}

// ─── 工具集（手动管理：插件化思路）──────────────────────

// getToolsets 工具集列表；传 name 返回该工具集完整详情（含插件与 disabledTools）。

async function getToolsets(name, workspaceRoot) {

  const ws = workspaceRoot || ''

  const params = {}

  if (name) params.name = name

  if (ws) params.workspaceRoot = ws

  return apiGet('/toolsets', params)

}

// toolsetEdit 手动编辑工具集：{name, scope?, action, plugin_name?, from_toolset?, tool?, plugin_json?, overwrite?}

// action: add_plugin / rm_plugin / rm_tool / enable_tool。

async function toolsetEdit(data) {

  return apiPost('/toolsets/edit', data)

}