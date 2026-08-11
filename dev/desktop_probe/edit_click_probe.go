// Command edit_click_probe 验证真实桌面链路：点击编辑器 → CM6 同步
// DOM selection → 键盘输入成功。
//   MockFocus（模拟 FocusElement）→ JS mousedown（detail:1）
//   → runJobs（CM6 measure + updateSelection 同步 DOM selection）
//   → 检查 getSelection().getRangeAt(0)（sstate.ranges）
//   → MockKeyChar → 检查 DOM/state 是否出现字符
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/app"
	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

func runJobs(wv *webkit.WebView) {
	for i := 0; i < 6; i++ {
		interp := wv.JSInterpreter()
		interp.RunJobs()
		if el := interp.GetEventLoop(); el != nil {
			el.ProcessTasks(0)
		}
		interp.RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
	}
}

func js(wv *webkit.WebView, code string) string {
	iv, err := wv.JSInterpreter().RunJS(code)
	if err != nil {
		return "[ERR] " + err.Error()
	}
	return iv.ToString()
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
	layout.FontMetricsFunc = func(family string, size float64, weight int, style string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	wv := webkit.NewWebView()
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
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
				return string(data), err
			}
		}
	}
	wv.LoadHTML(string(htmlData))
	for i := 0; i < 6; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
		time.Sleep(400 * time.Millisecond)
		runJobs(wv)
	}

	code := `package main

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
	code = strings.ReplaceAll(code, "\\", "\\\\")
	code = strings.ReplaceAll(code, "\n", "\\n")
	code = strings.ReplaceAll(code, "'", "\\'")
	js(wv, `(function(){
		var p = '/workspace/main.go';
		var st = window.__state;
		st.openFiles = [p];
		st.activeFile = p;
		st.fileContents[p] = '`+code+`';
		return 'injected';
	})()`)
	runJobs(wv)
	time.Sleep(100 * time.Millisecond)
	runJobs(wv)

	// ① MockFocus（模拟真实 desktop 点击聚焦）
	host := app.NewHostForTest(wv, 1280, 800)
	cmEl, err := findContentEditable(wv)
	if err != nil {
		log.Fatalf("no cm-content: %v", err)
	}
	host.MockFocus(cmEl)
	fmt.Println("[focus] contenteditable=" + cmEl.GetAttribute("contenteditable"))
	// hasFocus / activeElement（CM6 updateSelection 依赖）
	fmt.Println("[focus] " + js(wv, `(function(){
		var d = document;
		return 'hasFocus=' + d.hasFocus() + ' activeEl=' + (d.activeElement ? d.activeElement.className : 'null') +
			' isContent=' + (d.activeElement === document.querySelector('.cm-content'));
	})()`))

	// ② JS mousedown 点击行 1 中部（detail:1 = 单击）
	fmt.Println("[click] " + js(wv, `(function(){
		var co = document.querySelector('.cm-content');
		var l1 = co.querySelectorAll('.cm-line')[0];
		var r = l1.getBoundingClientRect();
		var cx = r.left + 40, cy = r.top + r.height/2;
		var ev = new MouseEvent('mousedown', {clientX: cx, clientY: cy, bubbles: true, cancelable: true, button: 0, detail: 1});
		var err = null;
		try { l1.dispatchEvent(ev); } catch(e) { err = e.message; }
		var v = window.__editorView;
		var h = v ? v.state.selection.main.head : -1;
		return 'click@' + cx.toFixed(1) + ',' + cy.toFixed(1) + ' head=' + h + ' err=' + err;
	})()`))
	runJobs(wv)
	time.Sleep(50 * time.Millisecond)
	runJobs(wv)
	// ②b 几何 dump：行 1 结构 + span/text 的 rect + activeLine + cursor
	fmt.Println("[geom] " + js(wv, `(function(){
		var out = [];
		var co = document.querySelector('.cm-content');
		var l1 = co ? co.querySelectorAll('.cm-line')[0] : null;
		if (!l1) return 'no-line';
		var lr = l1.getBoundingClientRect();
		out.push('line1=(' + lr.top.toFixed(1) + ' h=' + lr.height.toFixed(1) + ')');
		var kids = l1.children;
		out.push('kids=' + kids.length);
		for (var i = 0; i < kids.length && i < 6; i++) {
			var k = kids[i];
			var kr = k.getBoundingClientRect();
			out.push(i + ':<' + k.tagName.toLowerCase() + '.' + (k.className||'').split(' ')[0] + '> y=' + kr.top.toFixed(1) + ' h=' + kr.height.toFixed(1));
		}
		var al = document.querySelector('.cm-activeLine');
		if (al) { var ar = al.getBoundingClientRect(); out.push('activeLine=(' + ar.top.toFixed(1) + ' h=' + ar.height.toFixed(1) + ')'); }
		var cu = document.querySelector('.cm-cursor');
		if (cu) {
			var cur = cu.getBoundingClientRect();
			var ccs = getComputedStyle(cu);
			out.push('cursor=(' + cur.left.toFixed(1) + ',' + cur.top.toFixed(1) + ' ' + cur.width.toFixed(1) + 'x' + cur.height.toFixed(1) + ') disp=' + ccs.display + ' pos=' + ccs.position + ' borderL=' + ccs.borderLeft + ' color=' + ccs.color + ' vis=' + ccs.visibility + ' anim=' + (cu.style.animationName || ccs.animationName || '') + ' opacity=' + ccs.opacity);
			var layer = cu.parentElement;
			if (layer) {
				var lcs = getComputedStyle(layer);
				out.push('cursorLayer=(' + layer.className + ') disp=' + lcs.display + ' pos=' + lcs.position + ' vis=' + lcs.visibility + ' top=' + lcs.top + ' left=' + lcs.left + ' right=' + lcs.right + ' bottom=' + lcs.bottom + ' w=' + lcs.width + ' h=' + lcs.height + ' ovf=' + lcs.overflow + ' inset=' + (layer.style.inset || '') + ' cssText=' + (layer.getAttribute('style') || '').slice(0, 120));
			}
		} else out.push('cursor=NOT-FOUND');
		var fd = document.querySelectorAll('.cm-foldGutter .cm-gutterElement, .cm-foldGutter');
		out.push('foldGutter=' + fd.length);
		var fe = document.querySelector('.cm-foldGutter .cm-gutterElement');
		if (fe) {
			var fer = fe.getBoundingClientRect();
			out.push('foldEl=(' + fer.width.toFixed(1) + 'x' + fer.height.toFixed(1) + ') vis=' + getComputedStyle(fe).visibility + ' txt=[' + (fe.textContent||'').slice(0,4) + ']');
		}
		return out.join(' | ');
	})()`))

	// ③ 检查 CM6 同步后的 DOM selection（sstate.ranges）
	// ③ 检查 CM6 同步后的 DOM selection（sstate.ranges）
	fmt.Println("[sel] " + js(wv, `(function(){
		var s = window.getSelection();
		if (!s) return 'no-sel';
		var out = ['rc=' + s.rangeCount];
		out.push('anchor=' + (s.anchorNode ? s.anchorNode.nodeName : String(s.anchorNode)) + '/' + s.anchorOffset);
		if (s.rangeCount == 0) return out.join(' | ');
		var r0 = s.getRangeAt(0);
		if (r0 === null || r0 === undefined) { out.push('r0=' + r0); }
		else {
			var n = r0.startContainer;
			var txt = n && n.nodeType == 3 ? '[' + n.nodeValue.slice(0, 12) + ']' : (n ? '<' + n.nodeName + '>' : String(n));
			out.push('start=#' + (n ? n.nodeName : '?') + '/' + r0.startOffset + ' ' + txt);
		}
		var v = window.__editorView;
		if (v && v.observer && v.observer.selectionRange) {
			var sr = v.observer.selectionRange;
			out.push('obsRange=' + (sr.anchorNode ? sr.anchorNode.nodeName : 'null') + '/' + sr.anchorOffset +
				' focus=' + (sr.focusNode ? sr.focusNode.nodeName : 'null') + '/' + sr.focusOffset);
		}
		// ★ CM6 updateSelection 的 rawSel = getSelection(view.root) 链路
		if (v) {
			var rt2 = v.root;
			out.push('root=' + (rt2 ? rt2.nodeName + '/' + rt2.nodeType : 'null') + ' str=' + String(rt2) + ' hasGetSel=' + (rt2 && typeof rt2.getSelection) +
				' isDoc=' + (rt2 === document) + ' sameFn=' + (rt2 && rt2.getSelection === window.getSelection));
			var selX = null;
			try {
				selX = rt2 && rt2.nodeType == 9 ? rt2.defaultView.getSelection() : rt2.getSelection();
			} catch(e) { out.push('selXErr=' + e.message); }
			out.push('selX=' + (selX ? typeof selX + '/' + (selX.collapse ? 'has-collapse' : 'NO-collapse') : 'null'));
			var ws2 = window.getSelection();
			var dgs = document.getSelection();
			out.push('winSel=' + (ws2 ? (ws2.collapse ? 'has-collapse' : 'NO-collapse') : 'null') +
				' same=' + (ws2 === selX) +
				' rc=' + ws2.rangeCount + ' collapseType=' + typeof ws2.collapse +
				' docSel=' + (dgs ? (dgs.collapse ? 'has-collapse' : 'NO-collapse') : 'null') +
				' docSameWin=' + (dgs === ws2) +
				' src=[' + String(document.getSelection).slice(0, 90) + ']');
		}
		if (v && v.docView) {
			try {
				var d = v.docView.inlineDOMNearPos(5, 1);
				out.push('inline=' + (d && d.node ? d.node.nodeName : 'null') + '/' + (d ? d.offset : '?'));
			} catch(e) { out.push('inlineErr=' + e.message); }
			try {
				v.docView.updateSelection(false, true);
				out.push('us=ok');
			} catch(e) { out.push('usErr=' + e.message); }
		}
		return out.join(' | ');
	})()`))

	// ④ 键盘输入
	host.MockKeyChar('Z')
	host.MockKeyChar('Z')
	runJobs(wv)
	time.Sleep(100 * time.Millisecond)
	runJobs(wv)

	// ⑤ 结果
	fmt.Println("[after] " + js(wv, `(function(){
		var out = [];
		var co = document.querySelector('.cm-content');
		var t = co.textContent;
		out.push('hasZZ=' + (t.indexOf('ZZ') >= 0));
		out.push('line1=[' + t.split('\n')[0] + ']');
		var v = window.__editorView;
		if (v) {
			var h = v.state.selection.main.head;
			var line = v.state.doc.lineAt(h);
			out.push('head=' + h + ' line=' + line.number);
			out.push('stateLen=' + v.state.doc.length);
			out.push('stateHasZZ=' + (v.state.doc.toString().indexOf('ZZ') >= 0));
		}
		return out.join(' | ');
	})()`))

	// ⑤b Go 侧：dump cursor/cursorLayer 的渲染树 box
	dumpBox := func(name, cls string) {
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				if doc := fr.Document(); doc != nil {
					els := doc.GetElementsByClassName(cls)
					if len(els) == 0 {
						fmt.Printf("[%s-box] NOT-FOUND\n", name)
						return
					}
					el := els[0]
					if rv := wv.RenderView(); rv != nil {
						ro := rv.FindRenderObjectForNode(el)
						if ro == nil {
							fmt.Printf("[%s-box] no-render-object\n", name)
							return
						}
						// 父链：cursor 应挂在 cursorLayer 下
						chain := ""
						for p := ro.Parent(); p != nil && len(chain) < 200; p = p.Parent() {
							nm := "?"
							if n2 := p.Node(); n2 != nil {
								if e2, ok := n2.(*dom.Element); ok {
									nm = e2.ClassName()
								}
							}
							chain += " <- " + nm
						}
						fmt.Printf("[%s-box] parentChain=%s\n", name, chain)
						if b, ok := ro.(interface{ AsRenderBox() *rendering.RenderBox }); ok {
							if bb := b.AsRenderBox(); bb != nil {
								fmt.Printf("[%s-box] (%v,%v %vx%v) visible=%v disp=%v vis=%v\n", name, bb.X(), bb.Y(), bb.Width(), bb.Height(), bb.IsVisible(), bb.Style().Display, bb.Style().Visibility)
								return
							}
						}
						fmt.Printf("[%s-box] renderobj type=%T (inline?)\n", name, ro)
					}
				}
			}
		}
	}
	dumpBox("cursor", "cm-cursor")
	dumpBox("cursorLayer", "cm-cursorLayer")
	dumpBox("activeLine", "cm-activeLine")

	// ⑤c 触发渲染（paint）——验证 cursor 是否被 PaintBorder 绘制
	if pngBytes, err := wv.Render(); err != nil {
		fmt.Println("[render] err:", err)
	} else {
		fmt.Println("[render] ok bytes=" + fmt.Sprint(len(pngBytes)))
		wd2, _ := os.Getwd()
		outf, ferr := os.Create(filepath.Join(wd2, "dev", "desktop_probe", "click_probe.png"))
		if ferr == nil {
			img := image.NewRGBA(image.Rect(0, 0, wv.Width(), wv.Height()))
			for y := 0; y < wv.Height(); y++ {
				for x := 0; x < wv.Width(); x++ {
					off := (y*wv.Width() + x) * 4
					if off+3 < len(pngBytes) {
						img.SetRGBA(x, y, color.RGBA{R: pngBytes[off], G: pngBytes[off+1], B: pngBytes[off+2], A: pngBytes[off+3]})
					}
				}
			}
			_ = png.Encode(outf, img)
			outf.Close()
			fmt.Println("[render] saved click_probe.png")
		}
	}

	// ⑤c layer 树：找 cursor/cursorLayer 的 layer
	if rv := wv.RenderView(); rv != nil {
		var findLayer func(l *rendering.RenderLayer, depth int)
		findLayer = func(l *rendering.RenderLayer, depth int) {
			if l == nil || depth > 60 {
				return
			}
			if o := l.Owner(); o != nil {
				if n := o.Node(); n != nil {
					if el, ok := n.(*dom.Element); ok {
						if el.HasClassName("cm-cursor") || el.HasClassName("cm-cursorLayer") {
							nm := "other"
							if el.HasClassName("cm-cursor") {
								nm = "cursor"
							} else {
								nm = "cursorLayer"
							}
							fmt.Printf("[layer] %s depth=%d\n", nm, depth)
						}
					}
				}
			}
			for c := l.FirstChild(); c != nil; c = c.NextSibling() {
				findLayer(c, depth+1)
			}
		}
		findLayer(rv.RootLayer(), 0)
	}
}

func findContentEditable(wv *webkit.WebView) (*dom.Element, error) {
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			if doc := fr.Document(); doc != nil {
				els := doc.GetElementsByClassName("cm-content")
				if len(els) > 0 {
					return els[0], nil
				}
				return nil, fmt.Errorf("no cm-content element")
			}
		}
	}
	return nil, fmt.Errorf("no document")
}
