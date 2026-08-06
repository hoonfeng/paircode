// Command edit_probe 复现「文件打开/编辑破坏布局显示」。
// 加载真实 dist + desktopbridge：
//   1. 文件树点击打开 go.mod（CodeMirror 6）
//   2. 查询每个 .cm-line 的 getBoundingClientRect → 行高/行距一致性
//      （浏览器正常：均匀行高 ~19px；异常：大块粘合/重叠/错位）
//   3. 模拟 wb-ui 输入路径（setFocusedElementValue 对 contenteditable =
//      SetTextContent 全文替换）→ 检查 .cm-content 结构破坏情况
//   4. 离屏渲染 PNG 留存
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	csspkg "wb-ui/css"
	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/rendering"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

const hookJS = `
window.__errs = [];
window.addEventListener('error', function(e){ window.__errs.push('error: ' + (e && (e.message || e.type))); }, true);
window.addEventListener('unhandledrejection', function(e){ window.__errs.push('rejection: ' + ((e && e.reason && e.reason.message) || String(e))); }, true);
var _ce = console.error;
console.error = function(){
  window.__errs.push('console.error: ' + Array.prototype.slice.call(arguments).map(function(a){
    var m = typeof a === 'string' ? a : ((a && a.message) || String(a));
    if (a && a.stack) m += ' | STACK: ' + String(a.stack).split('\n').slice(0, 6).join(' <- ');
    return m;
  }).join(' | ').slice(0, 600));
  return _ce.apply(console, arguments);
};
`

const openGoModJS = `(function(){
  var rows = document.querySelectorAll('.file-tree-item .item-row');
  for (var i = 0; i < rows.length; i++) {
    var nm = rows[i].querySelector('.item-name');
    if (nm && nm.textContent.trim() === 'go.mod') return String(i);
  }
  return '-1';
})()`

const clickIdxJS = `(function(){
  var rows = document.querySelectorAll('.file-tree-item .item-row');
  var el = rows[IDX];
  if (!el) return;
  el.dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
})()`

const lineGeoJS = `(function(){
  var lines = document.querySelectorAll('.cm-content .cm-line');
  var out = [];
  for (var i = 0; i < lines.length; i++) {
    var r = lines[i].getBoundingClientRect();
    var txt = (lines[i].textContent || '').slice(0, 40);
    out.push(i + ':' + Math.round(r.top) + ',' + Math.round(r.height) + ' "' + txt + '"');
  }
  return out.join('\n');
})()`

const styleDiagJS = `(function(){
  var o = {};
  o.styleTags = document.querySelectorAll('style').length;
  o.headStyles = document.head ? document.head.querySelectorAll('style').length : -1;
  o.styleSheets = document.styleSheets ? document.styleSheets.length : -1;
  // head 里含 cm-editor 规则的 style
  var cmStyles = [];
  if (document.head) {
    var ss = document.head.querySelectorAll('style');
    for (var i = 0; i < ss.length; i++) {
      var t = ss[i].textContent || '';
      cmStyles.push('style#' + i + ' len=' + t.length + ' head120=' + JSON.stringify(t.slice(0, 120)));
      if (t.indexOf('cm-editor') >= 0) {
        var rules = [];
        var re = /[^{}]*\.cm-editor[^{}]*\{[^}]*\}/g;
        var m;
        while ((m = re.exec(t)) !== null && rules.length < 8) {
          rules.push(m[0].replace(/\s+/g, ' ').slice(0, 150));
        }
        cmStyles.push('  .cm-editor rules=[' + rules.join(' || ') + ']');
      }
    }
  }
  o.cmStyles = cmStyles;
  var ed = document.querySelector('.cm-editor');
  if (ed) {
    var cs = getComputedStyle(ed);
    o.edPos = cs.position; o.edDisplay = cs.display; o.edHeight = cs.height;
    o.edInline = ed.getAttribute('style') || '(none)';
    o.edClientH = ed.clientHeight; o.edOffsetH = ed.offsetHeight;
  }
  var sc = document.querySelector('.cm-scroller');
  if (sc) {
    var cs2 = getComputedStyle(sc);
    o.scPos = cs2.position; o.scDisplay = cs2.display; o.scOverflow = cs2.overflow;
    o.scInline = sc.getAttribute('style') || '(none)';
    o.scClientH = sc.clientHeight; o.scOffsetH = sc.offsetHeight; o.scScrollH = sc.scrollHeight;
  }
  var wr = document.querySelector('.code-editor-wrapper');
  if (wr) {
    o.wrClientH = wr.clientHeight; o.wrOffsetH = wr.offsetHeight; o.wrClientW = wr.clientWidth;
    var csw = getComputedStyle(wr);
    o.wrH = csw.height;
  }
  var ea = document.querySelector('.editor-area');
  if (ea) {
    o.eaClientH = ea.clientHeight; o.eaOffsetH = ea.offsetHeight; o.eaClientW = ea.clientWidth;
  }
  var co = document.querySelector('.cm-content');
  if (co) {
    var cs3 = getComputedStyle(co);
    o.coPos = cs3.position; o.coDisplay = cs3.display;
    o.coInline = co.getAttribute('style') || '(none)';
    o.coClientH = co.clientHeight; o.coScrollH = co.scrollHeight;
  }
  return JSON.stringify(o);
})()`

const editorGeoJS = `(function(){
  var parts = [];
  function R(sel) {
    var el = document.querySelector(sel);
    if (!el) return sel + '=null';
    var r = el.getBoundingClientRect();
    return sel + '=(' + Math.round(r.left) + ',' + Math.round(r.top) + ' ' + Math.round(r.width) + 'x' + Math.round(r.height) + ')';
  }
  return [R('.app-root'), R('.main-area'), R('.right-panel'), R('.rp-body'), R('.editor-area'), R('.editor-body'), R('.editor-wrapper'), R('.code-editor-wrapper'), R('.cm-editor'), R('.cm-scroller'), R('.cm-content')].join(' ');
})()`

const chainGeoJS = `(function(){
  var el = document.querySelector('.editor-area');
  var out = [];
  while (el && out.length < 10) {
    var r = el.getBoundingClientRect();
    var cs = getComputedStyle(el);
    out.push(el.className || el.tagName + '=' + Math.round(r.left) + ',' + Math.round(r.top) + ' ' + Math.round(r.width) + 'x' + Math.round(r.height) + ' display=' + cs.display + ' flex=' + cs.flex + ' w=' + cs.width + ' minw=' + cs.minWidth + ' pos=' + cs.position);
    el = el.parentElement;
  }
  return out.join('\\n');
})()`

const contentStateJS = `(function(){
  var co = document.querySelector('.cm-content');
  if (!co) return 'no-content';
  return 'children=' + co.children.length + ' firstChild=' + (co.children[0] ? co.children[0].className : '-') + ' textLen=' + (co.textContent||'').length + ' html0=' + (co.innerHTML||'').slice(0, 120);
})()`

const replaceTextJS = `(function(){
  var co = document.querySelector('.cm-content');
  if (!co) return;
  co.textContent = co.textContent;  // 等价 wb-ui setFocusedElementValue：SetTextContent 全文替换
})()`

func setupLoaders4(wv *webkit.WebView, distDir string) {
	absDist, _ := filepath.Abs(distDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				return string(data), err
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				if err != nil {
					return "", err
				}
				return string(data), nil
			}
		}
	}
}

func waitJS4(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func runJobs4(wv *webkit.WebView) {
	for i := 0; i < 8; i++ {
		wv.JSInterpreter().RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func runJS4(wv *webkit.WebView, script string) string {
	v, err := wv.JSInterpreter().RunJS(script)
	if err != nil {
		return "[err] " + err.Error()
	}
	return v.ToString()
}

// diagEditorClip 诊断编辑器滚动容器的裁剪链路：
//   - cm-scroller / cm-content 的 render box：overflow、padding-box、scroll offset
//   - 层树中 cm-* 相关 layer 的 CalculateRects clip
// 用于验证「cm-scroller(overflow:auto) 是否裁剪 cm-content 溢出内容」。
func diagEditorClip(wv *webkit.WebView) {
	fr := wv.MainFrame()
	if fr == nil {
		return
	}
	f := fr.Frame()
	if f == nil {
		return
	}
	rv := f.RenderView()
	if rv == nil {
		return
	}
	// 1) 渲染树中找 cm-scroller / cm-content 的 render box
	var scBox, coBox *rendering.RenderBox
	var walkRB func(ro rendering.RenderObject)
	walkRB = func(ro rendering.RenderObject) {
		if ro == nil {
			return
		}
		if rb, ok := ro.(*rendering.RenderBox); ok && rb.Style() != nil {
			if n := rb.Node(); n != nil {
				if el, ok := n.(*dom.Element); ok {
					cls := el.GetAttribute("class")
					if strings.Contains(cls, "cm-scroller") && scBox == nil {
						scBox = rb
					}
					if strings.Contains(cls, "cm-content") && coBox == nil {
						coBox = rb
					}
				}
			}
		}
		for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
			walkRB(c)
		}
	}
	walkRB(rendering.RenderObject(rv))
	report := func(tag string, b *rendering.RenderBox) {
		if b == nil {
			fmt.Printf("[CLIP] %s: NOT FOUND\n", tag)
			return
		}
		st := b.Style()
		ox, oy := -1, -1
		if st != nil {
			ox, oy = int(st.OverflowX), int(st.OverflowY)
		}
		pb := b.PaddingBoxRect()
		sx, sy := 0.0, 0.0
		if rv != nil {
			sx, sy = rv.BoxScrollOffset(b)
		}
		fmt.Printf("[CLIP] %s box=(%.0f,%.0f %.0fx%.0f) pb=(%.0f,%.0f %.0fx%.0f) ovf=(%d,%d) scroll=(%.0f,%.0f)\n",
			tag, b.X(), b.Y(), b.Width(), b.Height(), pb.X, pb.Y, pb.Width, pb.Height, ox, oy, sx, sy)
	}
	report("cm-scroller", scBox)
	report("cm-content", coBox)
	// 2) 层树中找 cm-scroller 相关 layer，打印 CalculateRects clip
	rootL := rv.RootLayer()
	var dumpL func(l *rendering.RenderLayer, depth int)
	dumpL = func(l *rendering.RenderLayer, depth int) {
		if l == nil || l.Owner() == nil {
			return
		}
		name := "?"
		if n := l.Owner().Node(); n != nil {
			if el, ok := n.(*dom.Element); ok {
				name = el.LocalName() + "." + el.GetAttribute("class")
			}
		}
		if strings.Contains(name, "cm-") || strings.Contains(name, "editor-") || strings.Contains(name, "main-area") ||
			strings.Contains(name, "right-") || strings.Contains(name, "chat-") || strings.Contains(name, "conv-") ||
			strings.Contains(name, "rp-") || strings.Contains(name, "input-") || strings.Contains(name, "bottom-") ||
			strings.Contains(name, "msg") || strings.Contains(name, "term-") || strings.Contains(name, "comp-") ||
			strings.Contains(name, "tool-") || strings.Contains(name, "obtn") || strings.Contains(name, "send-") {
			_, clip := l.CalculateRects()
			rb := l.Owner()
			var bx, by, bw, bh float64
			if rbx, ok := rb.(*rendering.RenderBox); ok {
				bx, by, bw, bh = rbx.X(), rbx.Y(), rbx.Width(), rbx.Height()
			}
			fmt.Printf("[LAYER] %s%s box=(%.0f,%.0f %.0fx%.0f) clip=(%.0f,%.0f %.0fx%.0f)\n",
				strings.Repeat("  ", depth), name, bx, by, bw, bh, clip.X, clip.Y, clip.Width, clip.Height)
		}
		for c := l.FirstChild(); c != nil; c = c.NextSibling() {
			dumpL(c, depth+1)
		}
	}
	dumpL(rootL, 0)
}

func pngEncode(width, height int, rgba []byte) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	copy(img.Pix, rgba)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// diagEditorBox 从布局树查 cm-editor 的实际 ComputedStyle 与几何（Go 侧，区别于 JS getComputedStyle）。
func diagEditorBox(wv *webkit.WebView) {
	fr := wv.MainFrame()
	if fr == nil {
		return
	}
	f := fr.Frame()
	if f == nil {
		return
	}
	rv := f.RenderView()
	doc := f.Document()
	if doc == nil || doc.DocumentElement() == nil {
		return
	}
	var found *dom.Element
	var walk func(el *dom.Element)
	walk = func(el *dom.Element) {
		if found != nil {
			return
		}
		if el != nil && strings.Contains(el.GetAttribute("class"), "cm-editor") {
			found = el
			return
		}
		for c := el.FirstChild(); c != nil; c = c.NextSibling() {
			if ce, ok := c.(*dom.Element); ok {
				walk(ce)
			}
		}
	}
	walk(doc.DocumentElement())
	if found == nil {
		log.Printf("diag: cm-editor DOM not found")
		return
	}
	// 遍历渲染树（RenderObject）找 cm-editor 节点，检查 nodeRenderMap 是否缺失
	var roNode dom.Node
	var walkRO func(ro rendering.RenderObject)
	walkRO = func(ro rendering.RenderObject) {
		if roNode != nil {
			return
		}
		if ro == nil {
			return
		}
		if ro.Node() == found {
			roNode = ro.Node()
			return
		}
		for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
			walkRO(c)
		}
	}
	walkRO(rv)
	if roNode == nil {
		log.Printf("diag: cm-editor NOT in render tree (DOM has %d children)", func() int {
			n := 0
			for c := found.FirstChild(); c != nil; c = c.NextSibling() {
				n++
			}
			return n
		}())
	} else {
		// 打印渲染树节点的 ComputedStyle
		if roEl, ok := roNode.(*dom.Element); ok {
			var walkS func(ro rendering.RenderObject)
			walkS = func(ro rendering.RenderObject) {
				if ro == nil {
					return
				}
				if ro.Node() == roNode {
					if st := ro.Style(); st != nil {
						log.Printf("diag: renderObj cm-editor Style: display=%v height=%s pos=%s", st.Display, st.Height.String(), st.Position)
					} else {
						log.Printf("diag: renderObj cm-editor Style nil")
					}
					return
				}
				for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
					walkS(c)
				}
			}
			walkS(rv)
			_ = roEl
		}
		rb2 := rv.FindRenderBoxForNode(roNode)
		if rb2 == nil {
			log.Printf("diag: cm-editor IS in render tree but FindRenderBoxForNode(node) nil — nodeRenderMap key mismatch")
			// 统计渲染树中缺失于 nodeRenderMap 的节点
			missing := 0
			total := 0
			var scan func(ro rendering.RenderObject, path string)
			scan = func(ro rendering.RenderObject, path string) {
				if ro == nil {
					return
				}
				total++
				label := "?"
				if n := ro.Node(); n != nil {
					if el, ok := n.(*dom.Element); ok {
						cls := el.GetAttribute("class")
						if cls == "" {
							cls = el.LocalName()
						}
						label = cls
					}
				} else {
					label = "(anon)"
				}
				if ro.Node() != nil && rv.FindRenderBoxForNode(ro.Node()) == nil {
					missing++
					if strings.HasPrefix(label, "cm-") || strings.Contains(path, "cm-editor") {
						log.Printf("diag:   MISSING cm node %s (%s) nodePtr=%p", path+"/"+label, ro.RenderName(), ro.Node())
					}
					if missing <= 12 {
						log.Printf("diag:   MISSING map node %s (%s)", path+"/"+label, ro.RenderName())
					}
				}
				for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
					scan(c, path+"/"+label)
				}
			}
			scan(rv, "")
			log.Printf("diag: render tree nodes total=%d missing=%d", total, missing)
		} else {
			log.Printf("diag: cm-editor render box found: X=%.1f Y=%.1f W=%.1f H=%.1f", rb2.X(), rb2.Y(), rb2.Width(), rb2.Height())
		}
	}
	// 对比 wrapper 的渲染树/布局树子节点（定位 syncChildren 跳过 cm-editor 的原因）
	var wrapRO rendering.RenderObject
	var walkRO2 func(ro rendering.RenderObject, want *dom.Element)
	walkRO2 = func(ro rendering.RenderObject, want *dom.Element) {
		if wrapRO != nil || ro == nil {
			return
		}
		if ro.Node() == want {
			wrapRO = ro
			return
		}
		for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
			walkRO2(c, want)
		}
	}
	var wrapEl *dom.Element
	var walkDOM func(el *dom.Element)
	walkDOM = func(el *dom.Element) {
		if wrapEl != nil || el == nil {
			return
		}
		if strings.Contains(el.GetAttribute("class"), "code-editor-wrapper") {
			wrapEl = el
			return
		}
		for c := el.FirstChild(); c != nil; c = c.NextSibling() {
			if ce, ok := c.(*dom.Element); ok {
				walkDOM(ce)
			}
		}
	}
	walkDOM(doc.DocumentElement())
	if wrapEl != nil {
		walkRO2(rv, wrapEl)
		var roKids []string
		if wrapRO != nil {
			for c := wrapRO.FirstChild(); c != nil; c = c.NextSibling() {
				label := "(anon)"
				if n := c.Node(); n != nil {
					if e, ok := n.(*dom.Element); ok {
						cls := e.GetAttribute("class")
						if cls == "" {
							cls = e.LocalName()
						}
						label = cls
					}
				}
				roKids = append(roKids, label)
			}
		}
		// 布局树 wrapper
		var wrapLB *layout.ElementBox
		var walkLB func(box *layout.ElementBox)
		walkLB = func(box *layout.ElementBox) {
			if wrapLB != nil || box == nil {
				return
			}
			if box.Element() == wrapEl {
				wrapLB = box
				return
			}
			for _, c := range box.Children() {
				if eb, ok := c.(*layout.ElementBox); ok {
					walkLB(eb)
				}
			}
		}
		walkLB(rv.LayoutBox())
		var lbKids []string
		if wrapLB != nil {
			for _, c := range wrapLB.Children() {
				if eb, ok := c.(*layout.ElementBox); ok {
					label := "(anon)"
					if eb.Element() != nil {
						cls := eb.Element().GetAttribute("class")
						if cls == "" {
							cls = eb.Element().LocalName()
						}
						label = cls
					}
					lbKids = append(lbKids, label)
				}
			}
		}
		log.Printf("diag: wrapper renderKids(%d)=%v", len(roKids), roKids)
		log.Printf("diag: wrapper layoutKids(%d)=%v", len(lbKids), lbKids)
	}
	// 直接用 resolver 解析 cm-editor 样式
	frame := fr.Frame()
	log.Printf("diag: frame=%p resolver=%v", frame, frame.Resolver())
	if rs := frame.Resolver(); rs != nil {
		if css2 := rs.ResolveElement(found); css2 != nil {
			log.Printf("diag: resolver cm-editor: display=%v set=%v height=%s pos=%v", css2.Display, css2.DisplaySet, css2.Height.String(), css2.Position)
			// 用 SelectorChecker 直接匹配 .ͼ1（CM6 module class）
			chk := csspkg.NewSelectorChecker()
			// 解析 style#0 的规则（含 .ͼ1）
			var style0 string
			if doc.Head() != nil {
				for c := doc.Head().FirstChild(); c != nil; c = c.NextSibling() {
					if se, ok := c.(*dom.Element); ok && se.LocalName() == "style" {
						style0 = se.TextContent()
						break
					}
				}
			}
			if style0 != "" {
				pr := csspkg.NewParser(style0)
				rules := pr.ParseStyleSheet()
				matchedAny := false
				for _, rr := range rules {
					if sr, ok := rr.(*csspkg.StyleRule); ok && sr.Selectors != nil {
						if chk.MatchList(sr.Selectors, found) {
							matchedAny = true
							var ds []string
							for _, d := range sr.Declarations {
								ds = append(ds, d.Name+":"+d.ValueString())
							}
							log.Printf("diag: style#0 rule %v MATCHED cm-editor decls=%v", sr.Selectors, ds)
						}
					}
				}
				if !matchedAny {
					log.Printf("diag: style#0 NO rule matched cm-editor (rules=%d)", len(rules))
				}
			} else {
				log.Printf("diag: style#0 empty/not found")
			}
			// 也解析 cm-scroller
			var scEl *dom.Element
			var walkSc func(el *dom.Element)
			walkSc = func(el *dom.Element) {
				if scEl != nil || el == nil {
					return
				}
				if strings.Contains(el.GetAttribute("class"), "cm-scroller") {
					scEl = el
					return
				}
				for c := el.FirstChild(); c != nil; c = c.NextSibling() {
					if ce, ok := c.(*dom.Element); ok {
						walkSc(ce)
					}
				}
			}
			walkSc(doc.DocumentElement())
			if scEl != nil {
				if css3 := rs.ResolveElement(scEl); css3 != nil {
					log.Printf("diag: resolver cm-scroller: display=%v set=%v height=%s", css3.Display, css3.DisplaySet, css3.Height.String())
				}
			}
		}
	}
	// 遍历布局树（ElementBox 树）找 DOM 元素匹配的 box
	rootEb := rv.LayoutBox()
	if rootEb == nil {
		log.Printf("diag: layout root nil")
		return
	}
	var matchEb *layout.ElementBox
	var walkEb func(box *layout.ElementBox, depth int)
	walkEb = func(box *layout.ElementBox, depth int) {
		if matchEb != nil {
			return
		}
		if box != nil && box.Element() == found {
			matchEb = box
			return
		}
		if box == nil {
			return
		}
		for _, c := range box.Children() {
			if eb, ok := c.(*layout.ElementBox); ok {
				walkEb(eb, depth+1)
			}
		}
	}
	walkEb(rootEb, 0)
	if matchEb == nil {
		log.Printf("diag: cm-editor not in layout tree (DOM has %d children)", func() int {
			n := 0
			for c := found.FirstChild(); c != nil; c = c.NextSibling() {
				n++
			}
			return n
		}())
		return
	}
	st := matchEb.Style()
	var hStr, dStr string
	if st != nil {
		hStr = st.Height.String()
		dStr = st.Display.String()
	}
	g := rv.LayoutState().GeometryForBox(matchEb)
	log.Printf("diag: cm-editor layout: height=%s display=%s borderH=%.1f contentH=%.1f top=%.1f",
		hStr, dStr, g.BorderBoxHeight(), g.ContentHeight(), g.Top())
	// 子链（scroller/content）
	var walkKids func(box *layout.ElementBox, depth int)
	walkKids = func(box *layout.ElementBox, depth int) {
		if depth > 4 || box == nil {
			return
		}
		for _, c := range box.Children() {
			if eb, ok := c.(*layout.ElementBox); ok {
				cs := eb.Style()
				var hh, dd string
				if cs != nil {
					hh = cs.Height.String()
					dd = cs.Display.String()
				}
				eg := rv.LayoutState().GeometryForBox(eb)
				log.Printf("diag:   child[%d] %s h=%s disp=%s L=%.1f T=%.1f W=%.1f H=%.1f",
					depth, func() string {
						if eb.Element() != nil {
							cls := eb.Element().GetAttribute("class")
							if cls == "" {
								return eb.Element().LocalName()
							}
							return cls
						}
						return "(anon)"
					}(), hh, dd, eg.Left(), eg.Top(), eg.BorderBoxWidth(), eg.BorderBoxHeight())
				walkKids(eb, depth+1)
			}
		}
	}
	walkKids(matchEb, 0)
	// 父链
	for p := matchEb.Parent(); p != nil && p.Parent() != nil; p = p.Parent() {
		ps := p.Style()
		pg := rv.LayoutState().GeometryForBox(p)
		var ph string
		if ps != nil {
			ph = ps.Height.String()
		}
		log.Printf("diag:   parent box height=%s borderH=%.1f contentH=%.1f top=%.1f", ph, pg.BorderBoxHeight(), pg.ContentHeight(), pg.Top())
	}
}

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, err := os.ReadFile(filepath.Join(distDir, "index.html"))
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style2, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}, text)
	}
	layout.FontMetricsFunc = func(family string, size float64, weight int, style2 string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	wv := webkit.NewWebView()
	wv.Resize(1280, 800)
	setupLoaders4(wv, distDir)
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	waitJS4(wv, 1500)
	runJobs4(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 全局错误捕获（LoadHTML 之后注入）
	runJS4(wv, hookJS)

	// 1. 打开 go.mod
	idx := runJS4(wv, openGoModJS)
	log.Printf("openGoMod idx=%s", idx)
	runJS4(wv, strings.ReplaceAll(clickIdxJS, "IDX", idx))
	waitJS4(wv, 1200)
	runJobs4(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 2. 所有 .cm-line 的布局几何
	fmt.Println("[LINE-GEO]")
	fmt.Println(runJS4(wv, lineGeoJS))

	diagEditorBox(wv)
	diagEditorClip(wv)

	// 3. 编辑器区域整体几何 + 父链 + 样式注入诊断
	log.Printf("editorGeo: %s", runJS4(wv, editorGeoJS))
	log.Printf("styleDiag: %s", runJS4(wv, styleDiagJS))
	fmt.Println("[CHAIN-GEO]")
	fmt.Println(runJS4(wv, chainGeoJS))

	// 4. 模拟 wb-ui 输入（SetTextContent 全文替换 contenteditable）
	log.Printf("cm-content BEFORE: %s", runJS4(wv, contentStateJS))
	runJS4(wv, replaceTextJS)
	waitJS4(wv, 600)
	runJobs4(wv)
	log.Printf("cm-content AFTER: %s", runJS4(wv, contentStateJS))

	// CM6 的 input/observer 是否恢复结构
	waitJS4(wv, 1200)
	runJobs4(wv)
	log.Printf("cm-content AFTER2(CM6 recovery): %s", runJS4(wv, contentStateJS))
	log.Printf("errs: %s", runJS4(wv, `window.__errs.join(' ;; ')`))

	// 5. 离屏渲染 PNG
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs4(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rgba, err := wv.Render()
	if err == nil && len(rgba) == 1280*800*4 {
		if err2 := os.WriteFile("_edit_probe.png", pngEncode(1280, 800, rgba), 0o644); err2 == nil {
			log.Printf("saved _edit_probe.png")
		}
	}
	log.Printf("完成")
}
