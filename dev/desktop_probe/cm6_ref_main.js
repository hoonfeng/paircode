// cm6_ref_main.js — 最小 CM6 参照页入口（Edge headless 基准数据）
// 与 CodeEditor.vue 相同的扩展组合 + 测试代码，dump gutter/行号几何
// 与 # 注释行字符 x 偏移，写入 document.title 供 --dump-dom 提取。
import { EditorView, basicSetup } from 'codemirror'
import { EditorState } from '@codemirror/state'
import { javascript } from '@codemirror/lang-javascript'
import { oneDark } from '@codemirror/theme-one-dark'

const code = `package main

import "fmt"

// main is the entry point.
func main() {
	fmt.Println("hello")
}

# hash comment test
func add(a, b int) int {
	return a + b
}
`

const view = new EditorView({
  parent: document.getElementById('app'),
  state: EditorState.create({
    doc: code,
    extensions: [basicSetup, javascript(), oneDark]
  })
})
window.__refView = view

function dump() {
  const out = []
  const g = document.querySelector('.cm-gutters')
  if (g) { const r = g.getBoundingClientRect(); out.push('guttersW=' + r.width.toFixed(1) + ' x=' + r.left.toFixed(1)) }
  const ln = document.querySelector('.cm-lineNumbers')
  if (ln) {
    const r = ln.getBoundingClientRect(); out.push('lineNumbersW=' + r.width.toFixed(1))
    const cs = getComputedStyle(ln)
    out.push('lnPad=' + cs.paddingLeft + '/' + cs.paddingRight + ' minW=' + cs.minWidth + ' ff=' + cs.fontFamily)
  }
  const ge = document.querySelectorAll('.cm-gutterElement')
  if (ge.length) {
    const r0 = ge[0].getBoundingClientRect()
    out.push('ge0H=' + r0.height.toFixed(1) + ' w=' + r0.width.toFixed(1))
    const cs0 = getComputedStyle(ge[0])
    out.push('ge0Pad=' + cs0.paddingLeft + '/' + cs0.paddingRight + ' minW=' + cs0.minWidth + ' ta=' + cs0.textAlign)
    out.push('ge0Text=[' + ge[0].textContent + ']')
    const ys = []
    for (let i = 0; i < 6 && i < ge.length; i++) ys.push(ge[i].getBoundingClientRect().top.toFixed(1))
    out.push('geYs=' + ys.join(','))
  }
  const co = document.querySelector('.cm-content')
  if (co) { const r = co.getBoundingClientRect(); out.push('contentX=' + r.left.toFixed(1) + ' w=' + r.width.toFixed(1)) }
  const lines = co.querySelectorAll('.cm-line')
  out.push('lines=' + lines.length)
  const hashLine = Array.from(lines).find(l => l.textContent.indexOf('#') >= 0)
  if (hashLine) {
    const r = hashLine.getBoundingClientRect()
    out.push('hashLineY=' + r.top.toFixed(1) + ' h=' + r.height.toFixed(1))
    const txt = hashLine.textContent
    const idx = txt.indexOf('#')
    out.push('hashLineText=[' + txt + '] idx=' + idx)
    try {
      const walker = document.createTreeWalker(hashLine, NodeFilter.SHOW_TEXT)
      let off = 0, target = null
      while (walker.nextNode()) {
        const n = walker.currentNode
        if (off + n.nodeValue.length > idx) { target = n; off = idx - off; break }
        off += n.nodeValue.length
      }
      if (target) {
        const rng = document.createRange()
        rng.setStart(target, off); rng.setEnd(target, off + 1)
        const rc = rng.getClientRects()
        if (rc && rc.length) out.push('hashCharX=' + rc[0].left.toFixed(1))
        // 测量行首到 # 的宽度
        const r2 = document.createRange()
        r2.setStart(target, 0)
        r2.setEnd(target, off)
        const rc2 = r2.getClientRects()
        if (rc2 && rc2.length) out.push('prefixW=' + rc2[0].width.toFixed(1))
      }
    } catch (e) { out.push('charErr=' + e.message) }
  }
  // 第一行文本宽度 + 空格宽度（# 后空格）
  const first = lines[0]
  if (first) {
    const r = first.getBoundingClientRect()
    out.push('line0Y=' + r.top.toFixed(1) + ' h=' + r.height.toFixed(1))
    out.push('line0Text=[' + first.textContent + ']')
  }
  document.title = 'CM6-REF:' + out.join('|')
}
setTimeout(dump, 300)
setTimeout(dump, 1500)
