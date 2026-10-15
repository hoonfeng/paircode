// tool-design-t6-agent-surface-check.mjs — tool-design 新增类型在「agent 工具面」上的可用性自检
//
// 用法：node scripts/tool-design-t6-agent-surface-check.mjs
//
// 背景：t5 让 progress/avatar/chart 成为合法节点类型，但 agent 是通过**工具面**用它们的：
//   design_add(node.add) 加节点、design_edit(node.set) 改属性、design_project(show) 看结构树、
//   design_schema 读字段全表与工具描述。任何一环没同步（字段表缺失、结构树不认、描述写着 14 类），
//   agent 就会「猜」——这正是本次增强要消灭的返工源。
//   本脚本以与 goja 沙箱相同的「函数体」语义加载插件，走真实工具调用链验证这条路径。
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
// ★ 2026-10-16：设计插件转正为基线插件，真源从 plugins-dist/ 迁回 .pair/plugins/。
const file = path.join(root, '.pair', 'plugins', 'tool-design', 'index.js')
const src = fs.readFileSync(file, 'utf8')

const fn = new Function('module', 'exports', 'require', 'ctx', '__dirname', '__filename', src)
const mod = { exports: {} }
let ret
try {
  ret = fn(mod, mod.exports, () => { throw new Error('沙箱无 require') }, undefined, path.dirname(file), file)
} catch (e) {
  console.error('✗ 加载插件失败：' + (e && e.message))
  process.exit(2)
}
const PLUGIN = (mod.exports && mod.exports.name) ? mod.exports : ret

const fsImpl = new Proxy({}, {
  get(_t, name) {
    return (...args) => {
      const p = String(args[0] === undefined ? '' : args[0])
      if (name === 'exists') return fs.existsSync(p)
      if (name === 'readFile') return fs.readFileSync(p, 'utf8')
      if (name === 'writeFile') {
        fs.mkdirSync(path.dirname(p), { recursive: true })
        fs.writeFileSync(p, String(args[1] === undefined ? '' : args[1]))
        return
      }
      if (name === 'mkdir' || name === 'mkdirAll' || name === 'ensureDir') { fs.mkdirSync(p, { recursive: true }); return }
      if (name === 'list' || name === 'readdir') return fs.readdirSync(p)
      if (name === 'stat') return fs.statSync(p)
      if (name === 'remove' || name === 'unlink' || name === 'rm') { fs.rmSync(p, { recursive: true, force: true }); return }
      throw new Error('未支持的 fs 方法: ' + String(name))
    }
  },
})

const defs = []
const ctx = {
  tools: { register(t) { defs.push(t) } },
  fs: fsImpl,
  logger: () => ({ info: () => {}, warn: () => {}, error: () => {} }),
}
PLUGIN.apply(ctx)

const byName = {}
for (const d of defs) byName[d.name] = d

const results = []
const check = (name, ok, detail) => results.push({ name, ok: !!ok, detail })
const run = async (tool, args) => {
  const d = byName[tool]
  if (!d) throw new Error('工具未注册: ' + tool)
  try {
    return await d.execute(args || {})
  } catch (e) {
    console.error('[stack ' + tool + '] ' + String(e && e.stack ? e.stack : '').split('\n').slice(0, 4).join(' | '))
    throw e
  }
}

const tmp = path.join(root, '_temp', 'design-t6-verify')
const projPath = path.join(tmp, 'design.project.json')
const tokensPath = path.join(tmp, 'design.tokens.json')
fs.rmSync(tmp, { recursive: true, force: true })
fs.mkdirSync(tmp, { recursive: true })

// ── C1 design_add(node.add) 路径：agent 逐节点增量添加新类型
let added = ''
try {
  await run('design_project', {
    mode: 'create', path: projPath, tokens: tokensPath, overwrite: true,
    title: 't6 验证：agent 工具面',
    screens: [{
      id: 's1', name: '验证屏', width: 360, height: 320,
      root: { type: 'col', props: { pad: '$space.md', gap: '$space.md', w: 'fill' } },
    }],
  })
  const r = await run('design_add', {
    path: projPath, screen: 's1',
    nodes: [
      { type: 'progress', props: { label: '构建进度', value: 45, tone: 'info', h: 8, w: 'fill' } },
      { type: 'avatar', props: { name: '李四', size: 36 } },
      { type: 'chart', props: { kind: 'donut', data: [5, 3, 2], w: 120, h: 120 } },
    ],
  })
  added = typeof r === 'string' ? r : JSON.stringify(r)
  check('C1 design_add 增量添加 progress/avatar/chart', true, added.slice(0, 120))
} catch (e) {
  check('C1 design_add 增量添加 progress/avatar/chart', false, String(e && e.message))
}

// ── C2 结构树摘要（agent 看结构的主要入口）：三类新节点带关键属性出现
try {
  const show = await run('design_project', { mode: 'show', path: projPath })
  const s = typeof show === 'string' ? show : JSON.stringify(show)
  const hasP = s.indexOf('progress') >= 0 && /45/.test(s)
  // 摘要与头像渲染同源：显示首字（'李'），不是全名
  const hasA = s.indexOf('avatar') >= 0 && s.indexOf('李') >= 0
  const hasC = s.indexOf('chart') >= 0 && s.indexOf('donut') >= 0
  check('C2 结构树摘要显示新类型与关键属性', hasP && hasA && hasC,
    `progress=${hasP} avatar=${hasA} chart=${hasC}`)
} catch (e) {
  check('C2 结构树摘要显示新类型与关键属性', false, String(e && e.message))
}

// ── C3 design_edit(node.set) 改新类型属性 → 产物随之变化（值真的接进了渲染管线）
try {
  const proj = JSON.parse(fs.readFileSync(projPath, 'utf8'))
  const screen = (proj.screens || []).find(x => x.id === 's1') || (proj.screens || [])[0]
  let prog = null
  const walk = n => {
    if (!n) return
    if (n.type === 'progress') prog = n
    for (const k of (n.children || [])) walk(k)
  }
  walk(screen && screen.root)
  if (!prog) throw new Error('未找到 progress 节点')
  await run('design_edit', {
    path: projPath,
    ops: [{ op: 'node.set', id: prog.id, props: { value: 90, tone: 'danger' } }],
  })
  const htmlPath = path.join(tmp, 'design.html')
  await run('design_export', { path: projPath, format: 'html', out: htmlPath, overwrite: true })
  const html = fs.readFileSync(htmlPath, 'utf8')
  check('C3 design_edit 改 value/tone 后产物同步（宽度与配色）',
    /aria-valuenow="90"/.test(html) && /width:90%/.test(html) && /danger|E5E7EB|F3F4F6/.test(html),
    `aria90=${/aria-valuenow="90"/.test(html)} fill90=${/width:90%/.test(html)}`)
} catch (e) {
  check('C3 design_edit 改 value/tone 后产物同步（宽度与配色）', false, String(e && e.message))
}

// ── C4 自省与描述：agent 读到的「结构事实」必须已含 17 类
try {
  const schema = byName['design_schema']
  const nodesTxt = schema.execute({ topic: 'nodes' })
  const allTxt = schema.execute({ topic: 'all' })
  const rows = nodesTxt.split('\n').filter(l => /^\| (progress|avatar|chart) /.test(l)).length
  const declared = (allTxt.match(/## 节点类型（(\d+) 类）/) || [])[1]
  const descOk = String(schema.description || '').indexOf('17 类节点') >= 0
  const addDesc = String(byName['design_add'].description || '')
  const hideStale = addDesc.indexOf('14 类') < 0
  check('C4 design_schema 表格/声明/描述 = 17 类且无 14 类残留',
    rows === 3 && declared === '17' && descOk && hideStale,
    `表格新行=${rows} 声明=${declared} 描述17=${descOk} 无14残留=${hideStale}`)
} catch (e) {
  check('C4 design_schema 表格/声明/描述 = 17 类且无 14 类残留', false, String(e && e.message))
}

// ── C5 未知类型仍被拒绝（防「放行一切」的过度宽松）
try {
  let rejected = false
  try {
    await run('design_add', { path: projPath, screen: 's1', nodes: [{ type: 'timeline' }] })
  } catch (e) {
    rejected = /未知节点类型/.test(String(e && e.message))
  }
  check('C5 未实现类型仍被拒绝（类型白名单未被放宽）', rejected, rejected ? 'timeline 被拒' : '未拒绝')
} catch (e) {
  check('C5 未实现类型仍被拒绝（类型白名单未被放宽）', false, String(e && e.message))
}

let pass = 0
for (const r of results) {
  if (r.ok) pass++
  console.log((r.ok ? 'PASS  ' : 'FAIL  ') + r.name + (r.detail ? '  —— ' + r.detail : ''))
}
console.log(`\n结果：${pass}/${results.length} 通过`)
process.exit(pass === results.length ? 0 : 1)
