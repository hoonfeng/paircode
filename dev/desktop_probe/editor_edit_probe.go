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

	"wb-ui/bindings"
	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

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
		return out.join(' | ');
	})()`))

	// ② 设置 DOM Selection：定位到第一行文本第 8 字符处
	fmt.Println("[sel] " + js(wv, `(function(){
		var co = document.querySelector('.cm-content');
		var line = co.querySelector('.cm-line');
		if (!line) return 'no line';
		var txt = line.firstChild;
		var node = txt;
		while (node && node.nodeType !== 3) { node = node.firstChild; }
		if (!node) return 'no text node, firstChildType=' + txt.nodeType + ' tag=' + (txt.tagName || '');
		var s = window.getSelection();
		s.removeAllRanges();
		var rng = document.createRange();
		rng.setStart(node, 8);
		rng.setEnd(node, 8);
		s.addRange(rng);
		return 'rangeCount=' + s.rangeCount + ' anchor=' + (s.anchorNode ? s.anchorNode.nodeName : 'null') + '/' + s.anchorOffset + ' len=' + node.nodeValue.length;
	})()`))

	// ③ Go 侧 InsertTextAtSelection（等价 host EventChar 的 contenteditable 分支）
	ok := bindings.InsertTextAtSelection("ZZ")
	fmt.Printf("[insert] InsertTextAtSelection='ZZ' → %v\n", ok)

	// ④ 派发 input 事件（host EventChar 分支后续动作）
	if cmEl, err := findContentEditable(wv); err == nil {
		cmEl.DispatchEvent(dom.NewInputEvent("insertText", "ZZ", false))
		fmt.Println("[input] dispatched input → cm-content")
	} else {
		fmt.Println("[input] no cm-content: " + err.Error())
	}
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
