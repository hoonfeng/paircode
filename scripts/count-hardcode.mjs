#!/usr/bin/env node
/**
 * 前端硬编码颜色统计（可复现口径 —— 用于验收，不允许出现无法复算的数字）
 *
 * 用法：
 *   node scripts/count-hardcode.mjs              # 组件源码 + index.html 非令牌行
 *   node scripts/count-hardcode.mjs --src        # 仅组件源码
 *   node scripts/count-hardcode.mjs --html       # 仅 index.html
 *   node scripts/count-hardcode.mjs --files      # 附按文件明细（降序）
 *   node scripts/count-hardcode.mjs --exclude-comments   # 排除纯注释行
 *
 * 口径（务必写清，便于独立复核）：
 *  1. 计数单位 = 「出现次数」，同一行多个 hex 各计一次（旧自述的 421→70 系「每行只记首个」，偏低，已废弃）。
 *  2. hex 须为合法颜色长度（3/4/6/8 位），且前一个字符不能是 [\w#&-]，
 *     以排除 CSS 选择器 #id、URL fragment 等非颜色用途。
 *  3. rgb()/rgba() 一律计入。
 *  4. index.html：仅统计「非令牌定义行」。形如 `--name: value;` 的变量定义行不计
 *     （它们就是令牌真相源本身）；规则体里的字面色计入。
 *  5. 注释行默认计入（口径从严）；加 --exclude-comments 可排除。
 */
import fs from 'node:fs'
import path from 'node:path'

const ROOT = path.resolve(process.cwd())
const SRC = path.join(ROOT, 'plugins-src/ui-app/src')
const HTML = path.join(ROOT, 'plugins-src/ui-app/index.html')
const argv = process.argv.slice(2)
const onlySrc = argv.includes('--src')
const onlyHtml = argv.includes('--html')
const showFiles = argv.includes('--files')
const exclComments = argv.includes('--exclude-comments')

const RE_HEX = /(^|[^\w#&-])#([0-9a-fA-F]{3,8})\b/g
const RE_RGB = /\brgba?\([^)]*\)/g
const isColorLen = h => /^([0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/.test(h)
const isComment = l => /^\s*(\/\*|\*|\/\/) /.test(l)

function countText(text, { skipTokenDefLines = false } = {}) {
  let hex = 0, rgb = 0
  for (const line of text.split(/\r?\n/)) {
    if (skipTokenDefLines && /^\s*--[a-zA-Z0-9-]+\s*:/.test(line)) continue
    if (exclComments && isComment(line)) continue
    let m
    RE_HEX.lastIndex = 0
    while ((m = RE_HEX.exec(line))) if (isColorLen(m[2])) hex++
    RE_RGB.lastIndex = 0
    while (RE_RGB.exec(line)) rgb++
  }
  return { hex, rgb, total: hex + rgb }
}

function walk(dir) {
  const out = []
  for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, e.name)
    if (e.isDirectory()) { if (e.name === 'node_modules' || e.name === 'assets') continue; out.push(...walk(p)) }
    else if (/\.(vue|js|ts|css)$/.test(e.name)) out.push(p)
  }
  return out
}

let srcLines = [], htmlRes = null
if (!onlyHtml) {
  for (const f of walk(SRC)) {
    const r = countText(fs.readFileSync(f, 'utf8'))
    if (r.total) srcLines.push({ f: path.relative(ROOT, f).replace(/\\/g, '/'), ...r })
  }
}
if (!onlySrc) htmlRes = countText(fs.readFileSync(HTML, 'utf8'), { skipTokenDefLines: true })

const sSum = srcLines.reduce((a, b) => a + b.total, 0)
console.log('== 前端硬编码颜色统计（口径见文件头注释）==')
if (!onlyHtml) console.log('组件源码 plugins-src/ui-app/src/**  : hex=' + srcLines.reduce((a, b) => a + b.hex, 0) + ' rgb=' + srcLines.reduce((a, b) => a + b.rgb, 0) + ' 合计=' + sSum + '（' + srcLines.length + ' 文件）')
if (!onlySrc) console.log('index.html 非令牌定义行              : 合计=' + htmlRes.total)
if (!onlySrc && !onlyHtml) console.log('----------------------------------------\n总计 = ' + (sSum + htmlRes.total))
if (showFiles && srcLines.length) {
  console.log('\n按文件（降序）：')
  for (const x of srcLines.sort((a, b) => b.total - a.total)) console.log('  ' + String(x.total).padStart(4) + '  (hex ' + String(x.hex).padStart(3) + '/rgb ' + String(x.rgb).padStart(3) + ')  ' + x.f)
}
