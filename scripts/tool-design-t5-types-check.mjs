// tool-design-t5-types-check.mjs — tool-design 新增类型（progress/avatar/chart）端到端自检
//
// 用法：node scripts/tool-design-t5-types-check.mjs
//
// 背景：t5 给 LEAF_TYPES 增加 progress / avatar / chart（14 → 17 类）——展示层（预览渲染）
//   与代码生成（HTML/Vue/React 三目标）都必须同步支持，否则「设计稿里能看、导出的代码里消失」。
//   本脚本以与 goja 沙箱相同的「函数体」语义加载插件源码，注册全部工具后**真实调用**：
//   design_project(create) → design_verify → design_export(html) → design_codegen(html/vue/react)，
//   断言新类型在四类产物里都有对应标记（进度条 role/aria、头像首字、图表内联 SVG、JSX 驼峰属性）。
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

// 真实文件系统适配（同宿主 ctx.fs 语义：路径一律绝对）
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
    const st = String(e && e.stack ? e.stack : '').split('\n').slice(0, 4).join(' | ')
    console.error('[stack ' + tool + '] ' + st)
    throw e
  }
}

const tmp = path.join(root, '_temp', 'design-t5-verify')
const projPath = path.join(tmp, 'design.project.json')
const tokensPath = path.join(tmp, 'design.tokens.json')
const outDir = path.join(tmp, 'out')
fs.rmSync(tmp, { recursive: true, force: true })
fs.mkdirSync(tmp, { recursive: true })

// ── 1. 建工程：progress（含 fill 宽）+ avatar（中文首字 / 拉丁首字母）+ chart（bar/line/donut）
const SCREEN_ROOT = {
  type: 'col', props: { pad: '$space.md', gap: '$space.md', w: 'fill' }, children: [
    { type: 'text', props: { text: '构建进度', size: '$fontSize.lg', weight: '$fontWeight.semibold' } },
    { type: 'progress', props: { value: 72, tone: 'success', h: 8, w: 'fill' } },
    { type: 'row', props: { gap: '$space.sm', align: 'center' }, children: [
      { type: 'avatar', props: { name: '张三', size: 40 } },
      { type: 'avatar', props: { name: 'Ada Lovelace', size: 32, tone: 'info' } },
      { type: 'text', props: { text: '张三 · 构建中', size: '$fontSize.md' } },
    ] },
    { type: 'chart', props: { kind: 'bar', data: [4, 7, 5, 9, 6, 3], w: 360, h: 120 } },
    { type: 'chart', props: { kind: 'line', data: [2, 5, 3, 8, 6], w: 360, h: 100 } },
    { type: 'row', props: { gap: '$space.md', align: 'center' }, children: [
      { type: 'chart', props: { kind: 'donut', data: [3, 5, 2], w: 120, h: 120 } },
      { type: 'text', props: { text: '环形占比', size: '$fontSize.sm', color: '$color.muted' } },
    ] },
  ],
}

let created = null
try {
  created = await run('design_project', {
    mode: 'create', path: projPath, tokens: tokensPath,
    title: 't5 验证：progress/avatar/chart', overwrite: true,
    screens: [{ id: 's1', name: '验证屏', width: 400, height: 360, root: SCREEN_ROOT }],
  })
  check('B1 工程可创建（新类型被接受为合法节点）', true, '')
} catch (e) {
  check('B1 工程可创建（新类型被接受为合法节点）', false, String(e && e.message))
}

// ── 2. 校验：D1-D11 无 FAIL
try {
  const v = await run('design_verify', { path: projPath })
  const txt = typeof v === 'string' ? v : JSON.stringify(v)
  const fails = (txt.match(/FAIL/g) || []).length
  check('B2 design_verify 对新类型工程无 FAIL', fails === 0, fails ? txt.slice(0, 600) : '无 FAIL')
} catch (e) {
  check('B2 design_verify 对新类型工程无 FAIL', false, String(e && e.message))
}

// ── 3. 预览导出（HTML）：新类型必须真实出现（不是空盒子）
try {
  const htmlPath = path.join(tmp, 'design.html')
  await run('design_export', { path: projPath, format: 'html', out: htmlPath, overwrite: true })
  const html = fs.readFileSync(htmlPath, 'utf8')
  const hasBar = /role="progressbar"/.test(html)
  const hasAria = /aria-valuenow="72"/.test(html)
  const hasFill = /width:\s*72%/.test(html)
  const hasAvatar1 = html.indexOf('张') >= 0
  const hasAvatar2 = html.indexOf('A') >= 0
  const svgCount = (html.match(/<svg/g) || []).length
  const hasRect = /<rect/.test(html)
  const hasPolyline = /<polyline/.test(html)
  const hasCircle = /<circle/.test(html)
  check('B3 预览 HTML：进度条语义与填充宽度',
    hasBar && hasAria && hasFill, `role=${hasBar} aria=${hasAria} fill=${hasFill}`)
  check('B4 预览 HTML：头像首字', hasAvatar1 && hasAvatar2, `中文首字=${hasAvatar1} 拉丁首字母=${hasAvatar2}`)
  check('B5 预览 HTML：三种图表形态内联 SVG',
    svgCount >= 3 && hasRect && hasPolyline && hasCircle,
    `svg=${svgCount} rect=${hasRect} polyline=${hasPolyline} circle=${hasCircle}`)
} catch (e) {
  check('B3 预览 HTML：进度条语义与填充宽度', false, String(e && e.message))
}

// ── 4. 代码生成三目标：新类型必须都在产物里（HTML/Vue 用 data/aria；React 用驼峰 SVG 属性）
try {
  await run('design_codegen', { path: projPath, targets: ['html', 'vue', 'react'], targetDir: outDir, overwrite: true })
  const files = []
  const walk = d => {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      const p = path.join(d, e.name)
      if (e.isDirectory()) walk(p); else files.push(p)
    }
  }
  walk(outDir)
  const pick = ext => files.filter(f => f.endsWith(ext))
  const htmlF = pick('.html')[0]
  const vueF = pick('.vue')[0]
  const jsxF = pick('.jsx')[0]

  const htmlC = htmlF ? fs.readFileSync(htmlF, 'utf8') : ''
  const vueC = vueF ? fs.readFileSync(vueF, 'utf8') : ''
  const jsxC = jsxF ? fs.readFileSync(jsxF, 'utf8') : ''
  const cssAll = pick('.css').map(f => fs.readFileSync(f, 'utf8')).join('\n')

  check('B6 codegen(html)：进度条 / 头像 / 图表标记',
    /role="progressbar"/.test(htmlC) && /aria-valuenow="72"/.test(htmlC) &&
    htmlC.indexOf('张') >= 0 && /<svg/.test(htmlC) && /width="360"/.test(htmlC),
    `progressbar=${/role="progressbar"/.test(htmlC)} avatar=${htmlC.indexOf('张') >= 0} svg=${/<svg/.test(htmlC)}`)
  check('B7 codegen(vue)：进度条骨架 + 头像文本 + 图表 SVG',
    /<span><\/span>/.test(vueC) && vueC.indexOf('张') >= 0 && /<svg/.test(vueC),
    `span=${/<span><\/span>/.test(vueC)} avatar=${vueC.indexOf('张') >= 0} svg=${/<svg/.test(vueC)}`)
  check('B8 codegen(react)：SVG 属性为驼峰（stroke-width 非法）',
    /strokeWidth=/.test(jsxC) && !/stroke-width=/.test(jsxC),
    `驼峰=${/strokeWidth=/.test(jsxC)} 连字符残留=${/stroke-width=/.test(jsxC)}`)
  check('B9 CSS：进度条填充规则（.cls > span 宽度 = value%）',
    /width:\s*72%/.test(cssAll), cssAll ? `css 文件 ${pick('.css').length} 个` : '无 CSS 产物')
} catch (e) {
  check('B6 codegen(html)：进度条 / 头像 / 图表标记', false, String(e && e.message))
}

// ── 汇总
let pass = 0
for (const r of results) {
  if (r.ok) pass++
  console.log((r.ok ? 'PASS  ' : 'FAIL  ') + r.name + (r.detail ? '  —— ' + r.detail : ''))
}
console.log(`\n结果：${pass}/${results.length} 通过`)
process.exit(pass === results.length ? 0 : 1)
