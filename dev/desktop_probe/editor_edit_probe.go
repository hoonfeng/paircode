// Command editor_edit_probe 验证 CM6 编辑器输入链路：
//   DOM Selection 设置 → bindings.InsertTextAtSelection（光标处插文本节点）
//   → input 事件 → CM6 DOMObserver readDOMChange 同步 state。
// 输出：
//   1. 初始 .cm-content/.cm-line 结构与文本
//   2. Selection 设置后 rangeCount（sstate.ranges 是否填充）
//   3. InsertTextAtSelection 返回值
//   4. 插入后 DOM 是否出现字符、CM6 是否同步（.cm-line 文本）
//   5. gutter 宽度 dump（行号栏宽度对比用）
//   6. PNG 截图
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
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

// dumpGutterTree 遍历渲染树，打印 .cm-gutters 子树（含 gutter 元素文本与几何）。
func dumpGutterTree(wv *webkit.WebView) {
	dumpSubtree(wv, "cm-gutters", "=== gutter render tree ===")
}

// dumpSubtree 遍历渲染树，打印指定 class 的子树。
func dumpSubtree(wv *webkit.WebView, clsFilter, title string) {
	rv := wv.RenderView()
	if rv == nil {
		fmt.Println("[rtree] no RenderView")
		return
	}
	fmt.Println("[rtree] " + title)
	var walk func(ro rendering.RenderObject, depth int, inGutter bool)
	walk = func(ro rendering.RenderObject, depth int, inGutter bool) {
		if ro == nil {
			return
		}
		var cls, txt string
		if el, ok := ro.Node().(*dom.Element); ok {
			cls = el.ClassName()
		}
		if rt, ok := ro.(*rendering.RenderText); ok {
			txt = rt.Text()
		}
		if strings.Contains(cls, clsFilter) {
			inGutter = true
		}
		var x, y, w, h float64
		if lb := ro.LayoutBox(); lb != nil {
			if ls := rv.LayoutState(); ls != nil {
				g := ls.GeometryForBox(lb)
				x, y, w, h = g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight()
			}
		}
		if inGutter {
			kind := ro.RenderName()
			if _, ok := ro.(*rendering.RenderText); ok {
				kind = "RenderText"
			}
			styleInfo := ""
			if cs := ro.Style(); cs != nil {
				styleInfo = fmt.Sprintf(" styleH=%v styleLH=%v disp=%v pos=%v", cs.Height, cs.LineHeight, cs.Display, cs.Position)
			}
			nodeInfo := ""
			if ro.Node() != nil {
				if el, ok := ro.Node().(*dom.Element); ok {
					nodeInfo = " <" + el.LocalName() + ">"
				} else {
					nodeInfo = " [#text]"
				}
			}
			if txt != "" {
				fmt.Printf("[rtree] %*s%s text=%q (%.1f,%.1f) %.1fx%.1f%s%s\n", depth*2, "", kind, txt, x, y, w, h, styleInfo, nodeInfo)
			} else {
				fmt.Printf("[rtree] %*s%s cls=%q (%.1f,%.1f) %.1fx%.1f%s%s\n", depth*2, "", kind, cls, x, y, w, h, styleInfo, nodeInfo)
			}
		}
		for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, depth+1, inGutter)
		}
	}
	walk(rv, 0, false)
	fmt.Println("[rtree] === end ===")
}

func runJobs(wv *webkit.WebView) {
	for i := 0; i < 8; i++ {
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
		time.Sleep(15 * time.Millisecond)
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
	wv.SetConsoleLogger(&jsc.BufferLogger{})
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	for i := 0; i < 6; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
		time.Sleep(450 * time.Millisecond)
		runJobs(wv)
	}

	// 注入打开 Go 文件
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
	time.Sleep(200 * time.Millisecond)
	runJobs(wv)

	// ① 初始 dump：CM6 是否构造、内容、gutter 宽度（Edge 同款测量）
	fmt.Println("[init] " + js(wv, `(function(){
		var out = [];
		var co = document.querySelector('.cm-content');
		if (!co) return 'NO cm-content';
		var lines = co.querySelectorAll('.cm-line');
		out.push('lines=' + lines.length);
		var first = lines[0];
		if (first) out.push('line1text=[' + first.textContent.slice(0, 40) + ']');
		var g = document.querySelector('.cm-gutters');
		if (g) {
			var gr = g.getBoundingClientRect();
			out.push('guttersW=' + gr.width.toFixed(1) + ' x=' + gr.left.toFixed(1));
			var ge = g.querySelectorAll('.cm-gutterElement');
			out.push('gutterEls=' + ge.length + ' firstH=' + (ge[0] ? ge[0].getBoundingClientRect().height.toFixed(1) : '?'));
			if (ge[0]) {
				var cs0 = getComputedStyle(ge[0]);
				out.push('ge0Pad=' + cs0.paddingLeft + '/' + cs0.paddingRight + ' minW=' + cs0.minWidth + ' ta=' + cs0.textAlign + ' mt=' + cs0.marginTop);
				out.push('ge0Text=[' + ge[0].textContent + '] w=' + ge[0].getBoundingClientRect().width.toFixed(1));
				var ys = [];
				for (var i = 0; i < 6 && i < ge.length; i++) ys.push(ge[i].getBoundingClientRect().top.toFixed(1));
				out.push('geYs=' + ys.join(','));
			}
			var ln = document.querySelector('.cm-lineNumbers');
			if (ln) {
				var lnr = ln.getBoundingClientRect();
				out.push('lineNumbersW=' + lnr.width.toFixed(1));
				var cs = getComputedStyle(ln);
				out.push('lnPad=' + cs.paddingLeft + '/' + cs.paddingRight + ' minW=' + cs.minWidth + ' ff=' + cs.fontFamily + ' fs=' + cs.fontSize);
			}
		}
		var sc = document.querySelector('.cm-scroller');
		if (sc) { var scr = sc.getBoundingClientRect(); out.push('scrollerX=' + scr.left.toFixed(1) + ' w=' + scr.width.toFixed(1)); }
		// # 注释行：字符 x 偏移（Edge 同款测量）
		var hashLine = Array.from(lines).find(function(l){ return l.textContent.indexOf('#') >= 0; });
		if (hashLine) {
			var hr = hashLine.getBoundingClientRect();
			out.push('hashLineY=' + hr.top.toFixed(1) + ' h=' + hr.height.toFixed(1));
			out.push('hashLineText=[' + hashLine.textContent + ']');
			try {
				var walker = document.createTreeWalker(hashLine, NodeFilter.SHOW_TEXT);
				var off = 0, target = null;
				while (walker.nextNode()) {
					var n = walker.currentNode;
					if (off + n.nodeValue.length > 0) { target = n; off = 0; break; }
					off += n.nodeValue.length;
				}
				if (target) {
					var rng = document.createRange();
					rng.setStart(target, 0); rng.setEnd(target, 1);
					var rc = rng.getClientRects();
					if (rc && rc.length) out.push('hashCharX=' + rc[0].left.toFixed(1));
				}
			} catch(e) { out.push('hashErr=' + e.message); }
		}
		if (first) {
			var fr = first.getBoundingClientRect();
			out.push('line0Y=' + fr.top.toFixed(1) + ' h=' + fr.height.toFixed(1));
			var flcs = getComputedStyle(first);
			out.push('line0Pad(pv)=' + (flcs.getPropertyValue ? flcs.getPropertyValue('padding-left') : '?') + '/' + (flcs.getPropertyValue ? flcs.getPropertyValue('padding-top') : '?') + ' dir=' + (flcs.paddingLeft !== undefined));
		}
		var ccs = getComputedStyle(co);
		out.push('contentPad(pv)=' + (ccs.getPropertyValue ? ccs.getPropertyValue('padding-top') : '?') + '/' + (ccs.getPropertyValue ? ccs.getPropertyValue('padding-bottom') : '?'));
		// gutter 子结构：每个 .cm-gutter 的宽（foldGutter vs lineNumbers）
		if (g) {
		if (g) {
			var gd = [];
			for (var i = 0; i < g.children.length; i++) {
				var ch = g.children[i];
				var gr2 = ch.getBoundingClientRect();
				gd.push((ch.className || ch.tagName) + '=' + gr2.width.toFixed(1));
				if (ch.className && ch.className.indexOf('foldGutter') >= 0) {
					var fge = ch.querySelector('.cm-gutterElement');
					if (fge) {
						var fgeR = fge.getBoundingClientRect();
						var fcs = getComputedStyle(fge);
						gd.push('foldEl0: w=' + fgeR.width.toFixed(1) + ' text=[' + fge.textContent + '] ff=' + (fcs.fontFamily || ''));
						var sp = fge.querySelector('span');
						if (sp) { var spR = sp.getBoundingClientRect(); gd.push('foldSpan: w=' + spR.width.toFixed(1) + ' text=[' + sp.textContent + ']'); }
					}
				}
			}
			out.push('gutters=[' + gd.join(',') + ']');
		}
		}
		// ★ dump 所有 gutter 元素：文本 + top + width + style.top（定位诊断）
		var gd2 = [];
		if (g) {
			var ge2 = g.querySelectorAll('.cm-gutterElement');
			for (var i = 0; i < ge2.length; i++) {
				var r2 = ge2[i].getBoundingClientRect();
				var st2 = ge2[i].style && ge2[i].style.top;
				var sh2 = ge2[i].style && ge2[i].style.height;
				var sm2 = ge2[i].style && ge2[i].style.marginTop;
				gd2.push('[' + i + ']"' + ge2[i].textContent + '"@' + r2.top.toFixed(1) + 'h' + r2.height.toFixed(1) + 'w' + r2.width.toFixed(1) + 'st=' + st2 + ' sh=' + sh2 + ' sm=' + sm2);
			}
			out.push('ALLGUTTER=' + gd2.join('|'));
		}
		// ★ 空行 cm-line 的高度（br 布局）
		var lines2 = co.querySelectorAll('.cm-line');
		var empty = [];
		for (var i = 0; i < lines2.length; i++) {
			if (lines2[i].textContent.length === 0) {
				var er = lines2[i].getBoundingClientRect();
				var ebr = lines2[i].querySelector('br');
				empty.push('L' + (i + 1) + ' h=' + er.height.toFixed(1) + ' br=' + (ebr ? ebr.getBoundingClientRect().height.toFixed(1) : 'none'));
			}
		}
		out.push('EMPTYLINES=' + empty.join('|'));
		return out.join(' | ');
	})()`))

	// ② 设置 DOM Selection：★ 真实 CM6 路径用 getSelection().collapse(node, off)
	// （CM6 点击/光标移动写 selection 的方式），而非手动 addRange——验证
	// collapse 同步 sstate.ranges（InsertTextAtSelection 依赖）。
	fmt.Println("[sel] " + js(wv, `(function(){
		var co = document.querySelector('.cm-content');
		var line = co.querySelector('.cm-line');
		if (!line) return 'no line';
		var txt = line.firstChild;
		var node = txt;
		while (node && node.nodeType !== 3) { node = node.firstChild; }
		if (!node) return 'no text node, firstChildType=' + txt.nodeType + ' tag=' + (txt.tagName || '');
		var s = window.getSelection();
		s.collapse(node, 8);
		var r0 = null;
		try { r0 = s.getRangeAt(0); } catch(e) { return 'getRangeAt err: ' + e.message; }
		return 'rangeCount=' + s.rangeCount + ' anchor=' + (s.anchorNode ? s.anchorNode.nodeName : 'null') + '/' + s.anchorOffset +
			' range0=' + (r0 && r0.startContainer ? r0.startContainer.nodeName : 'null') + '/' + (r0 ? r0.startOffset : '?') + ' len=' + node.nodeValue.length;
	})()`))

	// ③ ★ 真实键盘链路：FocusElement 设置 imeFocusedEl（点击聚焦）→
	// MockKeyChar 走 handleCharInput（processEvents 的 EventChar 分支同一实现）
	// → contenteditable 分支 InsertTextAtSelection → input 事件。
	// ★ 输入前渲染树 dump（对比输入后 gutter 高度是否被 CM6 重置）
	fmt.Println("[prekey] === render tree before keys ===")
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	dumpGutterTree(wv)
	dumpSubtree(wv, "cm-line", "=== cm-line render tree (prekey) ===")
	host := app.NewHostForTest(wv, 1280, 800)
	cmEl, err := findContentEditable(wv)
	if err != nil {
		log.Fatalf("no cm-content: %v", err)
	}
	host.MockFocus(cmEl)
	fmt.Printf("[focus] contenteditable=%q\n", cmEl.GetAttribute("contenteditable"))
	host.MockKeyChar('Z')
	host.MockKeyChar('Z')
	fmt.Println("[keys] MockKeyChar('Z')×2 → handleCharInput")
	// ⑤ 等 MutationObserver → CM6 readDOMChange
	runJobs(wv)
	time.Sleep(150 * time.Millisecond)
	runJobs(wv)

	// ⑥ dump：插入后 DOM 与 CM6 同步状态
	fmt.Println("[after] " + js(wv, `(function(){
		var out = [];
		var co = document.querySelector('.cm-content');
		if (!co) return 'NO cm-content';
		var lines = co.querySelectorAll('.cm-line');
		out.push('lines=' + lines.length);
		var first = lines[0];
		if (first) out.push('line1text=[' + first.textContent.slice(0, 50) + ']');
		var all = co.textContent;
		out.push('hasZZ=' + (all.indexOf('ZZ') >= 0));
		out.push('textLen=' + all.length);
		// ★ CM6 state 同步验证（readDOMChange 是否把 DOM 变化同步进 state）
		var v = window.__editorView;
		if (v) {
			var docStr = v.state.doc.toString();
			out.push('STATE=' + docStr.slice(0, 40).replace(/\n/g, '\\n'));
			out.push('stateHasZZ=' + (docStr.indexOf('ZZ') >= 0));
			out.push('stateLen=' + docStr.length);
			try {
				out.push('canUndo=' + v.undoManager.canUndo);
			} catch(e) { out.push('undoErr=' + e.message); }
		} else { out.push('noView'); }
		return out.join(' | ');
	})()`))

	// ⑦ 渲染截图
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	dumpGutterTree(wv)
	// ★ 第二次重建：验证 ComputedStyle 缓存是否失效（首次用旧值 0px）
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("[rtree2] === second rebuild ===")
	dumpGutterTree(wv)

	// ⑧ 滚动测试：设置 scroller.scrollTop → CM6 虚拟化重渲染行号
	js(wv, `(function(){
		var sc = document.querySelector('.cm-scroller');
		if (sc) { sc.scrollTop = 100; return 'scrolled-' + sc.scrollTop; }
		return 'no-scroller';
	})()`)
	runJobs(wv)
	time.Sleep(150 * time.Millisecond)
	runJobs(wv)
	fmt.Println("[scroll] " + js(wv, `(function(){
		var out = [];
		var g = document.querySelector('.cm-gutters');
		if (!g) return 'no-gutters';
		var ge = g.querySelectorAll('.cm-gutterElement');
		var sc = document.querySelector('.cm-scroller');
		out.push('st=' + (sc ? sc.scrollTop.toFixed(0) : '?'));
		var arr = [];
		for (var i = 0; i < ge.length && i < 8; i++) {
			var r = ge[i].getBoundingClientRect();
			arr.push(ge[i].textContent + '@' + r.top.toFixed(0));
		}
		out.push(arr.join('|'));
		return out.join(' ');
	})()`))
	pngBytes, err := wv.Render()
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	out := filepath.Join(wd, "dev", "desktop_probe", "edit_probe.png")
	f, err := os.Create(out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, wv.Width(), wv.Height()))
	for y := 0; y < wv.Height(); y++ {
		for x := 0; x < wv.Width(); x++ {
			off := (y*wv.Width() + x) * 4
			if off+3 < len(pngBytes) {
				img.SetRGBA(x, y, color.RGBA{R: pngBytes[off], G: pngBytes[off+1], B: pngBytes[off+2], A: pngBytes[off+3]})
			}
		}
	}
	_ = png.Encode(f, img)
	fmt.Println("[shot] → dev/desktop_probe/edit_probe.png")
	fmt.Println("DONE")
}
