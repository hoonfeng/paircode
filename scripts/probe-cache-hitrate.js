// probe-cache-hitrate.js — 缓存命中率排查探针（无浏览器，纯 HTTP 驱动）
//
// 目的：同一会话连续多轮，观察 ① mock/prov 侧每次请求的前缀命中率
//   ② 后端 [cache-diag] 的消息级前缀诊断（历史是否被原地改写）。
//   命中率骤降 + 前缀断裂日志 = 已发送历史字节被改写。
//
// 用法：node scripts/probe-cache-hitrate.js [port=9097] [rounds=3] [workspaceRoot=F:\syproject\gou-ide]
// 前置：被测实例在 <port> 监听；其 LLM 指向本地缓存模拟 mock
//   （node _temp/mock-llm-cache.js 8899）——mock 会打印每请求 hit/miss。
const http = require('http')

const PORT = Number(process.argv[2] || 9097)
const ROUNDS = Number(process.argv[3] || 3)
const ROOT = process.argv[4] || 'F:\\syproject\\gou-ide'
const BASE = 'http://localhost:' + PORT
const STAMP = Date.now()
const CONV_ID = 'conv_cache_probe_' + STAMP

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

function req(method, path, body) {
  return new Promise((resolve, reject) => {
    const data = body ? JSON.stringify(body) : null
    const r = http.request({
      host: '127.0.0.1', port: PORT, path, method,
      headers: data ? { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(data) } : {},
    }, (res) => {
      let d = ''
      res.on('data', (c) => (d += c))
      res.on('end', () => {
        let parsed = null
        try { parsed = JSON.parse(d) } catch (e) { parsed = d }
        resolve({ status: res.statusCode, body: parsed })
      })
    })
    r.on('error', reject)
    if (data) r.write(data)
    r.end()
  })
}

const runStatsUrl = (id) => `/api/conversations/${encodeURIComponent(id)}/run-stats?workspaceRoot=${encodeURIComponent(ROOT)}`

async function waitIdle(id, timeoutMs) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    await sleep(1200)
    const r = await req('GET', runStatsUrl(id))
    const b = r.body
    if (b && b.startAt > 0 && b.running === false) return b
  }
  return null
}

;(async () => {
  console.log('=== 缓存命中率排查探针 ===')
  console.log('port=%d rounds=%d conv=%s', PORT, ROUNDS, CONV_ID)

  const c = await req('POST', '/api/conversations', { id: CONV_ID, title: 'PROBE 缓存命中率', workspaceRoot: ROOT })
  console.log('创建会话:', c.status)

  for (let r = 1; r <= ROUNDS; r++) {
    const task = `第${r}轮：先看项目里的 markdown 文档，再简述结论（这是第 ${r} 轮，请给出不同侧重的分析）。`
    const t0 = Date.now()
    const sent = await req('POST', '/api/chat/send', { message: task, convId: CONV_ID, workspaceRoot: ROOT })
    if (sent.status !== 200) {
      console.error('第%d轮 chat/send 失败: %s %s', r, sent.status, JSON.stringify(sent.body))
      process.exit(1)
    }
    const stats = await waitIdle(CONV_ID, 120000)
    if (!stats) {
      console.error('第%d轮 120s 内未结束', r)
      process.exit(1)
    }
    console.log('第%d轮 完成：墙钟 %ds，LLM 调用 %d 次，工具 %d 次，prompt=%d completion=%d',
      r, Math.round((Date.now() - t0) / 1000), stats.llmCalls, stats.toolCalls, stats.promptTokens, stats.completionTokens)
    await sleep(1000) // 让落盘完成
  }

  // 落盘历史（供比对）
  const msgs = await req('GET', `/api/conversations/${encodeURIComponent(CONV_ID)}/messages?workspaceRoot=${encodeURIComponent(ROOT)}`)
  const n = Array.isArray(msgs.body) ? msgs.body.length : (msgs.body && msgs.body.messages ? msgs.body.messages.length : '?')
  console.log('会话消息数:', n)
  console.log('CONV_ID=%s', CONV_ID)
  console.log('=== 完成（请对照 mock 日志的 hit/miss 与后端 [cache-diag] 前缀诊断）===')
})().catch((e) => { console.error('探针异常:', e); process.exit(1) })
