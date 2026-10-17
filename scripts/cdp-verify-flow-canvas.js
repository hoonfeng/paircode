// 9096 实机验证：任务看板 → 流程库 = 左列表 + 右画布（可写流程全链路）
//   覆盖：列表加载 / 画布节点数 / 点选切换 / 拖拽重排（含还原）/ 节点增删 / 门禁增删 /
//         新建可写流程 → 改 id → 保存 → 落盘 + 页面回显一致 → 清理
// 用法：CDP_PORT=9224 WEB_PORT=9096 node scripts/cdp-verify-flow-canvas.js
const http = require('http'); const net = require('net'); const crypto = require('crypto'); const fs = require('fs'); const path = require('path')
const PORT = Number(process.env.CDP_PORT || 9224)
const WEB_PORT = Number(process.env.WEB_PORT || 9096)
const TMP_ID = 'canvas-e2e-tmp'
const ROOT = path.resolve(__dirname, '..')

function httpJson(p, method) { return new Promise((resolve, reject) => { const r = http.request({ host: '127.0.0.1', port: PORT, path: p, method: method || 'GET' }, x => { let d = ''; x.on('data', c => d += c); x.on('end', () => { try { resolve(JSON.parse(d)) } catch (e) { resolve(d) } }) }); r.on('error', reject); r.end() }) }
class WS { constructor() { this.buf = Buffer.alloc(0); this.frag = [] }
  connect(u) { return new Promise((res, rej) => { const m = u.match(/^ws:\/\/([^:/]+):(\d+)(\/.*)$/); const k = crypto.randomBytes(16).toString('base64'); this.sock = net.connect(+m[2], m[1], () => this.sock.write(`GET ${m[3]} HTTP/1.1\r\nHost: ${m[1]}:${m[2]}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ${k}\r\nSec-WebSocket-Version: 13\r\n\r\n`)); let hs = false; this.sock.on('data', d => { this.buf = Buffer.concat([this.buf, d]); if (!hs) { const i = this.buf.indexOf('\r\n\r\n'); if (i < 0) return; hs = true; this.buf = this.buf.slice(i + 4); res() } else this._drain() }); this.sock.on('error', rej) }) }
  _drain() { for (;;) { if (this.buf.length < 2) return; const b0 = this.buf[0], b1 = this.buf[1]; let len = b1 & 0x7f, off = 2; if (len === 126) { if (this.buf.length < 4) return; len = this.buf.readUInt16BE(2); off = 4 } else if (len === 127) { if (this.buf.length < 10) return; len = Number(this.buf.readBigUInt64BE(2)); off = 10 } const masked = (b1 & 0x80) !== 0; let mask = null; if (masked) { if (this.buf.length < off + 4) return; mask = this.buf.slice(off, off + 4); off += 4 } if (this.buf.length < off + len) return; let p = this.buf.slice(off, off + len); if (masked) { const o = Buffer.alloc(len); for (let i = 0; i < len; i++) o[i] = p[i] ^ mask[i & 3]; p = o } this.buf = this.buf.slice(off + len); const op = b0 & 0x0f; if (op === 1) this.frag.push(p); if ((b0 & 0x80) && this.frag.length) { this.onMessage(Buffer.concat(this.frag)); this.frag = [] } } }
  send(o) { const p = Buffer.from(JSON.stringify(o)); const len = p.length; let h; if (len < 126) h = Buffer.from([0x81, 0x80 | len]); else { h = Buffer.alloc(4); h[0] = 0x81; h[1] = 0x80 | 126; h.writeUInt16BE(len, 2) } const mask = crypto.randomBytes(4); const m = Buffer.alloc(len); for (let i = 0; i < len; i++) m[i] = p[i] ^ mask[i & 3]; this.sock.write(Buffer.concat([h, mask, m])) } }

let pass = 0, fail = 0
function check(name, ok, detail) { if (ok) pass++; else fail++; console.log(`${ok ? 'PASS' : 'FAIL'} ${name}${detail !== undefined ? ' · ' + JSON.stringify(detail) : ''}`) }

;(async () => {
  const list = await httpJson('/json/list')
  const tab = list.find(t => t.type === 'page' && new RegExp('localhost:' + WEB_PORT).test(t.url))
  if (!tab) throw new Error('未找到 ' + WEB_PORT + ' 页面 tab')
  const ws = new WS(); await ws.connect(tab.webSocketDebuggerUrl)
  let id = 0; const pend = new Map(); const consoleErrors = []
  ws.onMessage = b => {
    const m = JSON.parse(b.toString())
    if (m.id && pend.has(m.id)) { pend.get(m.id)(m); pend.delete(m.id); return }
    if (m.method === 'Runtime.consoleAPICalled' && m.params.type === 'error') consoleErrors.push((m.params.args || []).map(a => a.value || a.description || '').join(' '))
    if (m.method === 'Runtime.exceptionThrown') consoleErrors.push('EXC: ' + (m.params.exceptionDetails && m.params.exceptionDetails.text))
  }
  const send = (method, params) => new Promise((res, rej) => { const i = ++id; pend.set(i, x => x.error ? rej(new Error(x.error.message)) : res(x.result)); ws.send({ id: i, method, params }) })
  const ev = async e => { const r = await send('Runtime.evaluate', { expression: e, returnByValue: true, awaitPromise: true }); if (r.exceptionDetails) throw new Error('EVAL: ' + JSON.stringify(r.exceptionDetails.text || r.exceptionDetails)); return r.result && r.result.value }
  const sleep = ms => new Promise(r => setTimeout(r, ms))
  const shot = async (name) => { const r = await send('Page.captureScreenshot', { format: 'png' }); const f = path.join(ROOT, 'screenshots', name + '.png'); fs.mkdirSync(path.dirname(f), { recursive: true }); fs.writeFileSync(f, Buffer.from(r.data, 'base64')); console.log('截图:', f) }

  await send('Runtime.enable'); await send('Page.enable')
  await send('Page.reload', { ignoreCache: true })
  await sleep(5000)

  // 1. 打开「任务看板」→「流程库」
  const viewClicked = await ev(`(() => { const b=[...document.querySelectorAll('button,[role=tab],a,div')].find(x=>/任务看板/.test((x.textContent||'').trim()) && x.children.length<6 && (x.offsetWidth||0)<220); if(!b) return {ok:false}; b.click(); return {ok:true} })()`)
  await sleep(1800)
  const tabClicked = await ev(`(() => { const b=[...document.querySelectorAll('.ftb-tab,button')].find(x=>/流程库/.test((x.textContent||'').trim()) && (x.offsetWidth||0)<160); if(!b) return {ok:false}; b.click(); return {ok:true} })()`)
  await sleep(1600)
  check('打开任务看板视图', !!viewClicked.ok, viewClicked)
  check('切到「流程库」页签', !!tabClicked.ok, tabClicked)

  const shell = await ev(`(() => ({ items:document.querySelectorAll('.ftb-lib-side .ftb-flowitem').length,
    leftW:Math.round((document.querySelector('.ftb-lib-side')||{getBoundingClientRect:()=>({width:0})}).getBoundingClientRect().width),
    canvasW:Math.round((document.querySelector('.ftb-canvas')||{getBoundingClientRect:()=>({width:0})}).getBoundingClientRect().width) }))()`)
  check('左侧流程列表已渲染', shell.items >= 8, shell)
  check('左列表在画布左侧（布局）', shell.leftW > 150 && shell.canvasW > 200, shell)

  // 2. 新建可写流程（预设只读，编辑能力需在项目层验证）
  const newFlow = await ev(`(() => { const b=[...document.querySelectorAll('.ftb-lib-side button')].find(x=>/新建流程/.test(x.textContent||'')); if(!b) return {ok:false}; b.click(); return {ok:true} })()`)
  await sleep(1200)
  const blank = await ev(`(() => ({ nodes:[...document.querySelectorAll('.ftb-canvas .ftb-node')].map(n=>n.getAttribute('data-stage-id')),
    isNew:/新建流程/.test((document.querySelector('.ftb-cv-bar .ftb-cv-name')||{}).textContent||''),
    hasAdd:!!document.querySelector('.ftb-cv-add'), hasSave:[...document.querySelectorAll('.ftb-cv-bar button')].some(x=>/^保存$/.test((x.textContent||'').trim())),
    roNodes:document.querySelectorAll('.ftb-canvas .ftb-node.ro').length }))()`)
  check('新建流程进入可写编辑态', newFlow.ok && blank.isNew && blank.hasAdd && blank.hasSave && blank.roNodes === 0, blank)

  // 3. 添加节点 → 4 个
  const add3 = await ev(`(() => { const out=[]; for(let k=0;k<3;k++){ const b=document.querySelector('.ftb-cv-add'); if(!b) return {ok:false,out}; b.click(); out.push(document.querySelectorAll('.ftb-canvas .ftb-node').length) } return {ok:true,out} })()`)
  await sleep(900)
  const st4 = await ev(`(() => { const ns=[...document.querySelectorAll('.ftb-canvas .ftb-node')]; return { n:ns.length, ids:ns.map(n=>n.getAttribute('data-stage-id')), sel:ns.findIndex(n=>n.classList.contains('sel')), insp:(document.querySelector('.ftb-insp .ftb-count')||{}).textContent||'' } })()`)
  check('画布新增节点（连续 +3 且末尾自动选中）', add3.ok && st4.n === 4 && st4.sel === 3 && /#4/.test(st4.insp), st4)
  check('节点 id 唯一（删除后新增不复用重复 id）', new Set(st4.ids).size === st4.ids.length, st4.ids)

  // 4. 点击第 2 个节点 → 选中 + 检查器联动
  await ev(`(() => { const n=document.querySelectorAll('.ftb-canvas .ftb-node')[1]; if(n) n.click(); return true })()`)
  await sleep(900)
  const clickState = await ev(`(() => { const ns=[...document.querySelectorAll('.ftb-canvas .ftb-node')]; return { sel:ns.findIndex(n=>n.classList.contains('sel')), insp:(document.querySelector('.ftb-insp .ftb-count')||{}).textContent||'', gates:document.querySelectorAll('.ftb-insp .ftb-gate-form').length } })()`)
  check('点击画布节点切换选中（检查器跟随 #2）', clickState.sel === 1 && /#2/.test(clickState.insp), clickState)

  // 5. 拖拽重排：节点2 → 位置1，再还原
  const drag = await ev(`(() => {
    const ns=[...document.querySelectorAll('.ftb-canvas .ftb-node')]; if(ns.length<2) return {ok:false}
    const before=ns.map(n=>n.getAttribute('data-stage-id'))
    const dt=new DataTransfer(); const from=ns[1], to=ns[0]
    from.dispatchEvent(new DragEvent('dragstart',{bubbles:true,dataTransfer:dt}))
    to.dispatchEvent(new DragEvent('dragover',{bubbles:true,cancelable:true,dataTransfer:dt}))
    to.dispatchEvent(new DragEvent('drop',{bubbles:true,cancelable:true,dataTransfer:dt}))
    const after=[...document.querySelectorAll('.ftb-canvas .ftb-node')]
    return {ok:true, before, ids:after.map(n=>n.getAttribute('data-stage-id')), sel:after.findIndex(n=>n.classList.contains('sel'))}
  })()`)
  await sleep(700)
  check('拖动节点重排（顺序交换 + 选中跟随到新位置）', drag.ok && drag.ids[0] === drag.before[1] && drag.ids[1] === drag.before[0] && drag.sel === 0, drag)
  await ev(`(() => { const ns=[...document.querySelectorAll('.ftb-canvas .ftb-node')]; if(ns.length<2) return false; const dt=new DataTransfer(); ns[0].dispatchEvent(new DragEvent('dragstart',{bubbles:true,dataTransfer:dt})); ns[1].dispatchEvent(new DragEvent('dragover',{bubbles:true,cancelable:true,dataTransfer:dt})); ns[1].dispatchEvent(new DragEvent('drop',{bubbles:true,cancelable:true,dataTransfer:dt})); return true })()`)
  await sleep(700)
  const restored = await ev(`(() => [...document.querySelectorAll('.ftb-canvas .ftb-node')].map(n=>n.getAttribute('data-stage-id')))()`)
  check('拖拽还原为原始顺序', JSON.stringify(restored) === JSON.stringify(st4.ids), restored)

  // 6. 节点增删（悬停快捷按钮 ×）
  await ev(`(() => { const b=document.querySelector('.ftb-cv-add'); if(b) b.click(); return true })()`)
  await sleep(800)
  const addOne = await ev(`(() => { const ns=[...document.querySelectorAll('.ftb-canvas .ftb-node')]; return { n:ns.length, sel:ns.findIndex(n=>n.classList.contains('sel')) } })()`)
  check('画布新增节点（末尾 +1 且自动选中）', addOne.n === 5 && addOne.sel === 4, addOne)
  const delNode = await ev(`(() => { const ns=[...document.querySelectorAll('.ftb-canvas .ftb-node')]; const last=ns[ns.length-1]; if(!last) return {ok:false}; const b=[...last.querySelectorAll('.ftb-node-act button')].find(x=>/×/.test(x.textContent||'')); if(!b) return {ok:false}; b.click(); return {ok:true} })()`)
  await sleep(800)
  const afterDel = await ev(`(() => [...document.querySelectorAll('.ftb-canvas .ftb-node')].map(n=>n.getAttribute('data-stage-id')))()`)
  check('画布删除节点（恢复节点集）', delNode.ok && JSON.stringify(afterDel) === JSON.stringify(st4.ids), afterDel)

  // 7. 门禁增删（当前选中节点 #2）
  const gateBefore = await ev(`document.querySelectorAll('.ftb-insp .ftb-gate-form').length`)
  const addGate = await ev(`(() => { const b=[...document.querySelectorAll('.ftb-insp button')].find(x=>/加门禁/.test(x.textContent||'')); if(!b) return {ok:false}; b.click(); return {ok:true} })()`)
  await sleep(800)
  const gateAfter = await ev(`(() => ({ n:document.querySelectorAll('.ftb-insp .ftb-gate-form').length, chip:(document.querySelector('.ftb-node.sel .ftb-flowitem-meta .ftb-chipm')||{}).textContent||'' }))()`)
  check('检查器新增门禁（+1，画布节点门禁计数同步）', addGate.ok && gateAfter.n === gateBefore + 1 && /门禁/.test(gateAfter.chip), { before: gateBefore, after: gateAfter.n, chip: gateAfter.chip })
  const delGate = await ev(`(() => { const bs=[...document.querySelectorAll('.ftb-insp .ftb-gate-form button')].filter(x=>/删除门禁/.test(x.textContent||'')); const b=bs[bs.length-1]; if(!b) return {ok:false}; b.click(); return {ok:true} })()`)
  await sleep(800)
  const gateNow = await ev(`document.querySelectorAll('.ftb-insp .ftb-gate-form').length`)
  check('检查器删除门禁（恢复原数量）', delGate.ok && gateNow === gateBefore, { now: gateNow })

  // 8. 编辑节点属性（标题改到画布节点上实时回显）
  const titleEdit = await ev(`(() => { const inp=[...document.querySelectorAll('.ftb-insp input[type=text]')][1]; if(!inp) return {ok:false}; inp.value='E2E 标题'; inp.dispatchEvent(new Event('input',{bubbles:true})); return {ok:true} })()`)
  await sleep(600)
  const titleState = await ev(`(() => ({ node:(document.querySelector('.ftb-node.sel .ftb-node-title')||{}).textContent||'', insp:(document.querySelector('.ftb-insp .ftb-node-title')||{}).textContent||'' }))()`)
  check('检查器改标题 → 画布节点实时回显', titleEdit.ok && titleState.node === 'E2E 标题' && titleState.insp === 'E2E 标题', titleState)

  await shot('flow-canvas-9096')

  // 9. 改流程 id → 保存 → 落盘 + 回显一致
  const setId = await ev(`(() => { const inp=document.querySelector('.ftb-canvas .ftb-block input[type=text]'); if(!inp) return {ok:false}; inp.value=${JSON.stringify(TMP_ID)}; inp.dispatchEvent(new Event('input',{bubbles:true})); return {ok:true, v:inp.value} })()`)
  await sleep(500)
  const saveClick = await ev(`(() => { const b=[...document.querySelectorAll('.ftb-cv-bar button')].find(x=>/^保存$/.test((x.textContent||'').trim())); if(!b) return {ok:false}; b.click(); return {ok:true} })()`)
  await sleep(3500)
  const readSave = () => ev(`(() => ({ id:(document.querySelector('.ftb-cv-bar .ftb-gate-id')||{}).textContent||'', nodes:document.querySelectorAll('.ftb-canvas .ftb-node').length, name:(document.querySelector('.ftb-cv-bar .ftb-cv-name')||{}).textContent||'', result:(document.querySelector('.ftb-cv-bar')||{textContent:''}).textContent.slice(0,60), wrap:(document.querySelector('.ftb-wrap')||{textContent:''}).textContent.slice(0,200) }))()`)
  let afterSave = await readSave()
  if (!afterSave || !afterSave.id) { console.log('保存后首次读数为空，等待 3s 再读…', JSON.stringify(afterSave)); await sleep(3000); afterSave = await readSave() }
  const file = path.join(ROOT, '.pair', 'flows', TMP_ID + '.json')
  const fileOk = fs.existsSync(file)
  let fileDef = null
  if (fileOk) { try { fileDef = JSON.parse(fs.readFileSync(file, 'utf8')) } catch (e) { fileDef = { parseError: String(e) } } }
  check('保存落盘 .pair/flows/<新 id>.json', saveClick.ok && fileOk && !!fileDef, { file: fileOk, id: fileDef && fileDef.id, stages: fileDef && (fileDef.stages || []).length })
  check('落盘定义内 id 与文件名一致', !!fileDef && fileDef.id === TMP_ID, { fileId: fileDef && fileDef.id })
  check('保存后页面回显新流程 id（不再回退 my-flow）', afterSave.id === TMP_ID, afterSave)
  check('保存后画布节点数保持一致', afterSave.nodes === 4 && (fileDef.stages || []).length === 4, { dom: afterSave.nodes, file: fileDef && (fileDef.stages || []).length })
  check('节点标题修改已随保存落盘', !!fileDef && (fileDef.stages || []).some(s => s && s.title === 'E2E 标题'), (fileDef.stages || []).map(s => s.title))

  // 10. 清理
  try { if (fileOk) fs.unlinkSync(file) } catch (e) { console.log('清理失败:', e.message) }
  await ev(`(() => { const b=[...document.querySelectorAll('.ftb-cv-bar button')].find(x=>/^取消$/.test((x.textContent||'').trim())); if(b) b.click(); return true })()`)
  await sleep(800)
  check('临时流程文件已清理', !fs.existsSync(file))

  const realErrors = consoleErrors.filter(e => e && !/favicon|ResizeObserver|Failed to load resource.*404/i.test(e))
  check('无控制台报错', realErrors.length === 0, realErrors.slice(0, 5))

  console.log(`\n=== 结果：${pass} 通过 / ${fail} 失败 ===`)
  process.exit(fail ? 1 : 0)
})().catch(e => { console.error('ERR', e.message); process.exit(2) })
