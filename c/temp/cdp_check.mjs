// CDP 检查脚本：连 Edge，刷新页面，收集 console 错误，触发市场，检查弹窗
const TARGET = process.env.TARGET || 'http://localhost:9333'

async function getPageWs() {
  const res = await fetch(TARGET + '/json')
  const pages = await res.json()
  const page = pages.find(p => p.type === 'page' && p.url.includes('9096'))
  if (!page) throw new Error('未找到 9096 页面: ' + JSON.stringify(pages.filter(p=>p.type==='page').map(p=>p.url)))
  return page.webSocketDebuggerUrl
}

const wsUrl = await getPageWs()
console.log('连接:', wsUrl)
const ws = new WebSocket(wsUrl)
let msgId = 0
const pending = new Map()
const consoleErrors = []
const runtimeErrors = []
const send = (method, params = {}) => new Promise((resolve) => {
  const id = ++msgId
  pending.set(id, resolve)
  ws.send(JSON.stringify({ id, method, params }))
})

ws.onmessage = (ev) => {
  const msg = JSON.parse(ev.data)
  if (msg.id && pending.has(msg.id)) {
    pending.get(msg.id)(msg.result || msg.error)
    pending.delete(msg.id)
    return
  }
  if (msg.method === 'Runtime.consoleAPICalled') {
    const type = msg.params.type
    const text = (msg.params.args || []).map(a => a.value ?? a.description ?? '').join(' ')
    if (type === 'error' || type === 'warning') consoleErrors.push(`[${type}] ${text}`)
  }
  if (msg.method === 'Runtime.exceptionThrown') {
    const d = msg.params.exceptionDetails
    runtimeErrors.push((d.exception?.description || d.text || '').slice(0, 300))
  }
  if (msg.method === 'Log.entryAdded') {
    const e = msg.params.entry
    if (e.level === 'error') consoleErrors.push(`[log-error] ${e.text}`)
  }
}

await new Promise((r) => { ws.onopen = r })
await send('Runtime.enable')
await send('Console.enable')
await send('Log.enable')
await send('Page.enable')

// 刷新页面等加载
await send('Page.reload', { ignoreCache: true })
await new Promise(r => setTimeout(r, 7000))

const evalJs = async (expr) => {
  const r = await send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true })
  return r?.result?.value ?? r
}

console.log('=== 页面状态 ===')
console.log('__PAIRCODE_CORE:', await evalJs('!!window.__PAIRCODE_CORE'))
console.log('UiModals bundle:', await evalJs('typeof window.UiModals'))
console.log('showMarketplace 初值:', await evalJs('window.__PAIRCODE_CORE ? window.__PAIRCODE_CORE.uiState.showMarketplace.value : "N/A"'))
console.log('modals 槽位 DOM:', await evalJs(`(function(){const el=document.querySelector('.modals-host');return el?('存在, 子节点数='+el.children.length):'不存在'})()`))

// 检查 ui-modals bundle script 是否加载
console.log('ui-modals script 标签:', await evalJs(`(function(){const s=[...document.querySelectorAll('script')].filter(x=>x.src.includes('ui-modals'));return s.length?('已加载 '+s.length+' 个'):'未加载'})()`))
console.log('ui-titlebar script 标签:', await evalJs(`(function(){const s=[...document.querySelectorAll('script')].filter(x=>x.src.includes('ui-titlebar'));return s.length?('已加载 '+s.length+' 个'):'未加载'})()`))

// 触发市场打开
console.log('\n=== 触发市场 ===')
await evalJs(`window.dispatchEvent(new CustomEvent('open-marketplace'))`)
await new Promise(r => setTimeout(r, 1500))
console.log('触发后 showMarketplace:', await evalJs('window.__PAIRCODE_CORE.uiState.showMarketplace.value'))
console.log('触发后 modals 槽位:', await evalJs(`(function(){const el=document.querySelector('.modals-host');if(!el) return '无 modals-host';return '子节点='+el.children.length+', html头200: '+el.innerHTML.slice(0,200)})()`))
console.log('市场弹窗存在:', await evalJs(`(function(){return !!document.querySelector('.modal, .marketplace-modal, [class*=market]')})()`))

console.log('\n=== console 错误 (' + consoleErrors.length + ') ===')
consoleErrors.slice(0, 20).forEach(e => console.log(' -', e))
console.log('\n=== 运行时异常 (' + runtimeErrors.length + ') ===')
runtimeErrors.slice(0, 10).forEach(e => console.log(' -', e.slice(0, 400)))

ws.close()
process.exit(0)
