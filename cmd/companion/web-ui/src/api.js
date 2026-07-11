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

// ─── SSE 聊天（含自动重连）─────────────────────────────────

// chatSSE 发起 SSE 流式聊天并返回中止函数。
// 在网络断连时会自动重试（最多 retries 次），对端未启动或已停止则不重试。
function chatSSE(message, sessionId, autonomous, convId, callbacks, retries = 2) {
  let aborted = false
  let retryCount = 0
  let retryTimer = null

  function scheduleRetry(errMsg) {
    if (aborted || retryCount >= retries) {
      callbacks.onError?.(errMsg || '连接已断开，重试已达上限')
      return
    }
    retryCount++
    const delay = Math.min(1000 * Math.pow(2, retryCount - 1), 5000) // 指数退避 1s→2s→4s
    callbacks.onReconnect?.(retryCount, retries, delay)
    retryTimer = setTimeout(() => {
      if (!aborted) doFetch()
    }, delay)
  }

  async function doFetch() {
    if (aborted) return
    let r
    try {
      r = await fetch(apiURL('/chat/send'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message, sessionId, autonomous, convId }),
      })
    } catch (networkErr) {
      // 网络级失败（如连接被拒绝）→ 重试
      if (!aborted) {
        scheduleRetry(networkErr.message || '网络连接失败')
      }
      return
    }
    if (!r.ok) {
      let errMsg = r.statusText
      try {
        const e = await r.json()
        errMsg = e.error || errMsg
      } catch { /* ignore */ }
      // 4xx 错误不重试（参数错误、未配置等）
      if (r.status >= 400 && r.status < 500) {
        callbacks.onError?.(errMsg)
        return
      }
      scheduleRetry(errMsg)
      return
    }
    retryCount = 0 // 连接成功 → 重置重试计数

    const reader = r.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      let result
      try {
        result = await reader.read()
      } catch (readErr) {
        // 流读取失败（连接断开）→ 重试
        if (!aborted) {
          scheduleRetry(readErr.message || '流读取中断')
        }
        return
      }
      const { done, value } = result
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const parts = buffer.split('\n')
      buffer = parts.pop() || ''
      for (const line of parts) {
        const trimmed = line.trim()
        // 跳过注释行（心跳）
        if (trimmed.startsWith(':')) continue
        if (trimmed.startsWith('data: ')) {
          try {
            const data = JSON.parse(trimmed.slice(6))
            callbacks.onEvent?.(data)
          } catch { /* ignore parse errors */ }
        }
      }
    }
    // 正常结束
    callbacks.onDone?.()
  }

  doFetch()

  // 返回中止函数
  return () => {
    aborted = true
    if (retryTimer) clearTimeout(retryTimer)
  }
}

function stopChat(sessionId) {
  return apiGet('/chat/stop', { sessionId })
}

// 回答 ask_user 问题
async function answerChat(sessionId, answer) {
  return apiPost('/chat/answer', { sessionId, answer })
}

// 审批写工具
async function approveChat(sessionId, callId, approved) {
  return apiPost('/chat/approve', { sessionId, callId, approved })
}

// 运行时反馈：Agent 执行中用户可补充/纠正
async function sendFeedback(sessionId, content) {
  return apiPost('/chat/feedback', { sessionId, content })
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

export default { apiGet, apiPost, apiPut, apiDelete, chatSSE, stopChat, answerChat, approveChat, sendFeedback, getModels, getInstructions, saveInstructions, getPhilosophy, savePhilosophy }
