// cdp-verify-ctxmenu-chain.cjs — 实证编辑器右键链路（CodeEditor ↔ EditorArea）
// ★ 2026-09-25 修复后：本脚本已由「抓缺陷」转为「回归守卫」——断言方向改为
//   修复后的正确行为：① contextmenu 监听器幂等（改字号后恒为 1）；
//   ② 有选中时菜单走「有选中」分支（不含 格式化文档/命令面板）；
//   ③ 「AI: 添加到对话」载荷 type=selection（含 lineStart/lineEnd/content）。
// 原始发现（修复前，用于回归时对照）：
//   ① CodeEditor.createEditor() 里 wrapperRef.addEventListener('contextmenu') 每次重建都注册、
//      从不移除 → 累积（累积路径 = state.settings.fontSize 变化，因为 props.path 变化会换 :key
//      导致组件整体重建、监听器随节点回收）
//   ② defineEmits 声明的是 'contextmenu-selection'，实际 emit 的是 'contextmenu'（未声明）
//   ③ 父组件 EditorArea 用 @contextmenu（绑在 CodeEditor 组件上）→ Vue fallthrough 到根元素
//      成为**原生**监听器 → 收到的是原生 MouseEvent（无 hasSelection/text/lineStart/lineEnd）
//      → 「有选中文本」分支永不进入 → 点「AI: 添加到对话」时按整文件处理（降级）
// 用法：node scripts/cdp-verify-ctxmenu-chain.cjs [webPort=9099] [chromePort=9223]
//   前置：独立实例（cd _temp/verify-gen && WEB_PORT=9099 ./companion_gen.exe）
//        + headless chrome（--headless=new --remote-debugging-port=9223）；9090 全程不动。
const WEB = Number(process.argv[2] || 9099)
const CHROME = Number(process.argv[3] || 9223)
const FILE1 = 'plugins-src/ui-app/src/api.js'
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
const results = []
const check = (name, ok, extra) => { results.push({ name, ok, extra }); console.log((ok ? '  PASS  ' : '  FAIL  ') + name + (extra ? '  —— ' + extra : '')) }

class CDP {
  constructor(url) { this.url = url; this.id = 0; this.pending = new Map(); this.consoleMsgs = [] }
  connect() {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(this.url)
      this.ws.addEventListener('open', () => resolve())
      this.ws.addEventListener('error', (e) => reject(new Error('ws error')))
      this.ws.addEventListener('message', (ev) => {
        const m = JSON.parse(ev.data)
        if (m.id && this.pending.has(m.id)) {
          const { res, rej } = this.pending.get(m.id); this.pending.delete(m.id)
          m.error ? rej(new Error(JSON.stringify(m.error))) : res(m.result)
        } else if (m.method === 'Runtime.consoleAPICalled') {
          const txt = (m.params.args || []).map((a) => a.value ?? a.description ?? '').join(' ')
          this.consoleMsgs.push({ type: m.params.type, text: txt })
        }
      })
    })
  }
  send(method, params = {}) {
    const id = ++this.id
    return new Promise((res, rej) => { this.pending.set(id, { res, rej }); this.ws.send(JSON.stringify({ id, method, params })) })
  }
  async eval(expression, byValue = true) {
    const r = await this.send('Runtime.evaluate', { expression, returnByValue: byValue, awaitPromise: true })
    if (r.exceptionDetails) throw new Error('eval 异常: ' + JSON.stringify(r.exceptionDetails.exception?.description || r.exceptionDetails.text))
    return byValue ? r.result.value : r.result
  }
}

async function main() {
  const page = await (await fetch(`http://127.0.0.1:${CHROME}/json/new?${encodeURIComponent('about:blank')}`, { method: 'PUT' })).json()
  const cdp = new CDP(page.webSocketDebuggerUrl)
  await cdp.connect()
  await cdp.send('Runtime.enable')
  await cdp.send('Page.enable')
  await cdp.send('Page.navigate', { url: 'http://127.0.0.1:' + WEB })

  // 等 UI 就绪
  for (let i = 0; i < 60; i++) {
    await sleep(500)
    const ok = await cdp.eval('!!(window.__PAIRCODE_CORE && document.querySelector(".editor-body"))').catch(() => false)
    if (ok) break
  }
  const wsRoot = await cdp.eval('window.__PAIRCODE_CORE.uiState.state.workspaceRoot || ""')
  console.log(`  工作区: ${wsRoot}`)

  // 打开文件 → 渲染 CodeEditor
  await cdp.eval(`(() => { const st = window.__PAIRCODE_CORE.uiState.state; st.openFiles = [${JSON.stringify(FILE1)}]; st.activeFile = ${JSON.stringify(FILE1)}; return st.openFiles.length })()`)
  let ready = false
  for (let i = 0; i < 40; i++) {
    await sleep(400)
    ready = await cdp.eval('!!document.querySelector(".code-editor-wrapper .cm-content")').catch(() => false)
    if (ready) break
  }
  check('编辑器（CM6）已渲染', ready, ready ? '' : '未出现 .cm-content')

  const countListeners = async () => {
    const obj = await cdp.eval('document.querySelector(".code-editor-wrapper")', false)
    if (!obj.objectId) return -1
    const r = await cdp.send('DOMDebugger.getEventListeners', { objectId: obj.objectId, depth: 0 })
    return (r.listeners || []).filter((l) => l.type === 'contextmenu').length
  }
  const ctxCount = await countListeners()
  check('基线：wrapper 上 contextmenu 原生监听器 = 1', ctxCount === 1, `实测 ${ctxCount}`)

  // ★ 累积路径：改 fontSize 触发 watch → createEditor 重建（:key 不变 → 组件不重建）
  for (const fs of [13, 14, 15, 16]) {
    await cdp.eval(`(() => { const s = window.__PAIRCODE_CORE.uiState.state; if (!s.settings) s.settings = {}; s.settings.fontSize = ${fs}; return s.settings.fontSize })()`)
    await sleep(350)
  }
  const ctxCount2 = await countListeners()
  check('★ 改字号 4 次后监听器仍为 1（幂等注册；修复前为 1→5 累积）', ctxCount2 === 1, `实际 ${ctxCount} → ${ctxCount2}`)

  // ★ 有选中文本 + 右键 → 最终菜单版本
  const menuProbe = `(() => {
    const deepest = (txt) => [...document.querySelectorAll('div,span,li,button')]
      .filter(el => el.children.length === 0 && el.textContent.trim() === txt)
      .pop() || null
    const labels = [...document.querySelectorAll('div,span,li,button')]
      .filter(el => el.children.length === 0 && el.offsetParent !== null && el.textContent.trim().length > 1 && el.textContent.trim().length < 12)
      .map(el => el.textContent.trim())
    return {
      hasNoSelOnly: labels.includes('格式化文档'),
      hasCopyLine: labels.includes('复制行内容'),
      hasCmdPalette: labels.includes('命令面板'),
      hasAddToChat: labels.includes('AI: 添加到对话'),
      labels: [...new Set(labels)].slice(0, 20),
    }
  })()`
  await cdp.eval(`window.__probe = null; window.addEventListener('add-to-chat', (e) => { window.__probe = e.detail })`)
  // 选中一段文本（★ 文档可能为空——读文件失败时 doc.length=0，选区越界会抛 RangeError；先补内容）
  const selOk = await cdp.eval(`(() => {
    const v = window.__editorView; if (!v) return 'no-view'
    if (v.state.doc.length < 20) v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: 'right click probe line one\\nsecond line here\\n' } })
    const head = Math.min(12, v.state.doc.length)
    v.dispatch({ selection: { anchor: 0, head } })
    const s = v.state.selection.main
    return { text: v.state.sliceDoc(s.from, s.to), docLen: v.state.doc.length }
  })()`)
  console.log(`  已选中文本: ${JSON.stringify(selOk)}`)
  await cdp.eval(`document.querySelector('.code-editor-wrapper .cm-content').dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 460, clientY: 320 }))`)
  await sleep(500)
  const menu = await cdp.eval(menuProbe)
  check('右键后菜单已弹出', !!menu && (menu.hasAddToChat || menu.hasCmdPalette), JSON.stringify(menu.labels))
  check('★ 有选中文本时菜单呈「有选中」版（不含 格式化文档/命令面板 → 载荷 hasSelection 生效）',
    !!menu && !menu.hasNoSelOnly && !menu.hasCmdPalette, `格式化文档=${menu?.hasNoSelOnly} 命令面板=${menu?.hasCmdPalette} 添加到对话=${menu?.hasAddToChat}`)

  // ★ 点「AI: 添加到对话」→ 看 detail.type（selection = 正确；file = 降级）
  const clicked = await cdp.eval(`(() => {
    const el = [...document.querySelectorAll('div,span,li,button')].filter(e => e.children.length === 0 && e.textContent.trim() === 'AI: 添加到对话').pop()
    if (!el) return 'not-found'; el.click(); return 'clicked'
  })()`)
  await sleep(600)
  const probe = await cdp.eval('window.__probe ? JSON.stringify(window.__probe) : null')
  console.log(`  点击结果: ${clicked}；add-to-chat 载荷: ${probe}`)
  let detail = null
  try { detail = probe ? JSON.parse(probe) : null } catch {}
  check('★ 「AI: 添加到对话」载荷 type=selection（期望）', !!detail && detail.type === 'selection', detail ? `实际 type=${detail.type}, 字段=${Object.keys(detail).join(',')}` : '未收到 add-to-chat 事件')

  const errs = cdp.consoleMsgs.filter((m) => m.type === 'error')
  console.log(`  控制台 error ${errs.length} 条` + (errs.length ? ':\n' + errs.slice(0, 5).map((e) => '      ' + e.text.slice(0, 140)).join('\n') : ''))
  const warns = cdp.consoleMsgs.filter((m) => m.type === 'warning')
  console.log(`  控制台 warning ${warns.length} 条` + (warns.length ? ':\n' + warns.slice(0, 5).map((e) => '      ' + e.text.slice(0, 140)).join('\n') : ''))

  const pass = results.filter((r) => r.ok).length
  console.log(`\n═══ ${pass}/${results.length} 项断言通过 ═══`)
  try { await cdp.send('Page.close') } catch {}
  process.exit(0)
}
main().catch((e) => { console.error('脚本异常:', e.message); process.exit(1) })
