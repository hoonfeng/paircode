#!/usr/bin/env node
// audit-plugin-ui-register.cjs — 创作域插件「client 半注册面」审计（2026-09-18）
//
// 口径与宿主同源：
//   · 宿主前端 plugin-runtime.js 的 syncClientHalves 用 `new Function` **求值** client.js
//     源码（源码形态是 `(ui) => void` 表达式），得到函数后以 ui 对象调用 → 收集注册调用。
//   · 本脚本照做（eval 求值 + 假 ui/document/window 替身），断言六域**只注册主内容区视图**
//     （ui.registerView），不再注册插件面板（ui.registerPanel）。
//
// 依据：docs/plugin-development.md §12 分工表 —— registerPanel = 「管理/总览类面板（藏在
//   插件面板里）」，registerView = 「需要长期占用主工作区的界面（编辑器式工作台）」；
//   六域（画布/波形/3D/角色/人声）属后者，面板区那份是重复入口。详见 .pair/project.md
//   「六域不再注册「插件面板」——只留主内容区视图」。
//
// 用法：
//   node scripts/audit-plugin-ui-register.cjs             # 审计六域（默认）
//   node scripts/audit-plugin-ui-register.cjs --brief     # 只列不合格项
//   node scripts/audit-plugin-ui-register.cjs tool-art    # 单包
// 退出码：0 = 全部符合预期；1 = 有告警。

const fs = require('fs')
const path = require('path')

const ROOT = path.resolve(__dirname, '..')
const DEFAULT_PKGS = ['tool-art', 'tool-design', 'tool-model', 'tool-music', 'tool-rig', 'tool-voice']
// 期望 view id 形如 '<域>-view'（域短名 = 包名剥 tool- 前缀）
const VIEW_ID_RE = /^(art|design|model|music|rig|voice)-view$/

// auditOne 求值单包 client.js 并收集注册调用，返回 {pkg, calls, errors}。
function auditOne(pkg) {
  const out = { pkg, file: path.join('plugins-dist', pkg, 'client.js'), calls: [], errors: [] }
  const p = path.join(ROOT, out.file)
  if (!fs.existsSync(p)) {
    out.errors.push('client.js 缺失')
    return out
  }
  let fn
  try {
    // eslint-disable-next-line no-eval
    fn = eval(fs.readFileSync(p, 'utf8'))
  } catch (e) {
    out.errors.push('语法错误: ' + e.message)
    return out
  }
  if (typeof fn !== 'function') {
    out.errors.push('client 半不是函数表达式（应为 (ui) => void）')
    return out
  }

  const calls = []
  const ui = {
    registerPanel: (s) => calls.push({ kind: 'panel', id: s && s.id, title: s && s.title }),
    registerView: (s) =>
      calls.push({ kind: 'view', id: s && s.id, title: s && s.title, order: s && s.order, open: s && s.open }),
    registerSlot: (s) => calls.push({ kind: 'slot', id: s && s.slotId }),
    reportFailure: (stage, msg) => calls.push({ kind: 'failure', stage, msg }),
  }
  // 宿主前端最小替身：bundle 已就绪（window[GLOBAL].mount 是函数）→ register() 立即成功，
  // 不进 script 注入兜底分支（该分支依赖真实 DOM 事件，测不到注册面）。
  global.window = new Proxy({}, { get: () => ({ mount() {} }) })
  global.document = {
    querySelector: () => null,
    createElement: () => ({ setAttribute() {} }),
    head: { appendChild() {} },
  }
  try {
    fn(ui)
  } catch (e) {
    out.errors.push('执行异常: ' + e.message)
  }

  out.calls = calls
  const views = calls.filter((c) => c.kind === 'view')
  const panels = calls.filter((c) => c.kind === 'panel')
  const fails = calls.filter((c) => c.kind === 'failure')
  if (views.length !== 1) {
    out.errors.push(`期望恰好 1 条 registerView，实际 ${views.length}`)
  } else {
    if (!VIEW_ID_RE.test(views[0].id || '')) out.errors.push(`view id 不符规范 '<域>-view': ${views[0].id}`)
    if (views[0].open !== true) out.errors.push('view 未声明 open:true（应默认后台打开，不抢对话主视图）')
    if (typeof views[0].order !== 'number') out.errors.push('view 未声明 order（tab 顺序）')
  }
  if (panels.length) out.errors.push(`仍注册插件面板: ${panels.map((c) => c.id).join(', ')}`)
  if (fails.length) out.errors.push(`向宿主上报失败（说明走到了兜底分支）: ${JSON.stringify(fails)}`)
  return out
}

function main() {
  const args = process.argv.slice(2)
  const brief = args.includes('--brief')
  const named = args.filter((a) => !a.startsWith('-'))
  const targets = named.length ? named : DEFAULT_PKGS
  let bad = 0
  const orders = []
  for (const pkg of targets) {
    const r = auditOne(pkg)
    const view = r.calls.find((c) => c.kind === 'view')
    if (r.errors.length) {
      bad++
      console.log(`✗ ${r.pkg}`)
      for (const e of r.errors) console.log(`    ${e}`)
    } else if (!brief) {
      orders.push(view.order)
      console.log(
        `✓ ${r.pkg}: registerView{id:${view.id}, title:${view.title}, order:${view.order}, open:${view.open}}；无 registerPanel`
      )
    }
  }
  console.log(`\n${targets.length - bad}/${targets.length} 包符合预期（只注册主内容区视图、无插件面板注册）`)
  if (!brief && orders.length === targets.length && targets.length > 1) {
    const sorted = orders.slice().sort((a, b) => a - b)
    console.log(`view tab 顺序: ${sorted.join(' < ')}${JSON.stringify(sorted) === JSON.stringify(orders) ? '（按包序递增）' : '（注意：与包序不一致）'}`)
  }
  process.exit(bad ? 1 : 0)
}

main()
