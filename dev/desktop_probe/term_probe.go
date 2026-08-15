// Command term_probe loads the terminal reference page
// (cmd/companion/web-ui/ide_ref_term.html) through wb-ui (the SAME engine
// as cmd/desktop) — it renders FIXED cmd-like content via xterm.js with the
// DOM renderer, then dumps the complete element tree (geometry + computed
// styles) for comparison against Edge headless (edge_term_ref.go →
// term_tree_edge.json).
//
// Output: dev/desktop_probe/term_tree_wb.json + stdout tree
//
// Run: go run ./dev/desktop_probe/term_probe.go
//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
	"wb-ui/webkit"
)

type TNode struct {
	Tag      string   `json:"tag"`
	ID       string   `json:"id"`
	Class    string   `json:"class"`
	X        float64  `json:"x"`
	Y        float64  `json:"y"`
	W        float64  `json:"w"`
	H        float64  `json:"h"`
	Display  string   `json:"display"`
	Color    string   `json:"color"`
	BG       string   `json:"bg"`
	FontSz   string   `json:"fontSz"`
	Lh       string   `json:"lh,omitempty"`
	Text     string   `json:"text,omitempty"`
	Depth    int      `json:"depth"`
	Idx      int      `json:"idx"`
	Children []TNode  `json:"children,omitempty"`
}

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	webDir := filepath.Join(wd, "cmd", "companion", "web-ui")
	htmlData, err := os.ReadFile(filepath.Join(webDir, "ide_ref_term.html"))
	if err != nil {
		log.Fatalf("read ide_ref_term.html: %v", err)
	}
	log.Printf("[term_probe] webDir=%s html=%d bytes", webDir, len(htmlData))

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
	setupTermLoaders(wv, webDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()

	wv.LoadHTML(string(htmlData))
	// wb-ui 引擎不自动派发 window load 事件——手动触发参照页 onload
	//（与 Edge 自动触发对齐）。
	_, _ = wv.JSInterpreter().RunJS(`
		if (typeof window.onload === 'function') { window.onload(); }
	`)
	// ★ 驱动事件循环：xterm 的 write 是异步的（write 数据先进
	// _writeBuffer，经 microtask/parser 处理进入 buffer），DOM 渲染器
	// 依赖 requestAnimationFrame 驱动 RenderDebouncer。真实 desktop 由
	// host.go 每帧调用 EventLoop.ProcessTasks；term_probe 无 host 循环，
	// 手动驱动：ProcessTasks 处理宏任务/微任务/rAF，再跑渲染管线。
	interp := wv.JSInterpreter()
	if el := interp.GetEventLoop(); el != nil {
		for i := 0; i < 10; i++ {
			el.ProcessTasks(0)
			_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 200); })`)
			el.ProcessTasks(0)
		}
	} else {
		log.Println("[term_probe] WARN: no event loop")
	}
	if out := wv.ConsoleOutput(); out != "" {
		fmt.Println("[CONSOLE]")
		fmt.Println(out)
	}
	// 检查 xterm 是否成功渲染
	_, _ = wv.JSInterpreter().RunJS(`
		var host = document.getElementById('term-host');
		var rows = host && host.querySelector('.xterm-rows');
		console.log('[TERM] host=' + !!host + ' rows=' + (rows ? rows.children.length : 'null'));
		if (window.Terminal) console.log('[TERM] xterm loaded');
		else console.log('[TERM] xterm NOT loaded');
		// 诊断 xterm 测量（Terminal 包装对象字段在 _core 下）
		try {
			var t = window.__term;
			if (t) {
				var core = t._core || t;
				var rs = core._renderService;
				var d = rs ? rs.dimensions : null;
				console.log('[TERM] dims css.cell=' + (d && d.css ? d.css.cell.width + 'x' + d.css.cell.height : 'null') +
					' device.cell=' + (d && d.device ? d.device.cell.width + 'x' + d.device.cell.height : 'null'));
				var cs = core._charSizeService;
				if (cs) {
					console.log('[TERM] charSize w=' + cs.width + ' h=' + cs.height + ' valid=' + cs.hasValidSize);
				} else {
					var keys = [];
					for (var k in core) { if (k.indexOf('char') >= 0 || k.indexOf('render') >= 0) keys.push(k); }
					console.log('[TERM] no charSize; keys=' + keys.join(','));
				}
				console.log('[TERM] hasRenderer=' + (rs ? rs.hasRenderer() : 'nors'));
				if (rs && rs._renderer && rs._renderer.value) {
					console.log('[TERM] renderer ctor=' + (rs._renderer.value.constructor ? rs._renderer.value.constructor.name : 'anon'));
				}
				var me = host.querySelector('.xterm-char-measure-element');
				if (me) {
					console.log('[TERM] measure offsetW=' + me.offsetWidth + ' offsetH=' + me.offsetHeight +
						' rect=' + me.getBoundingClientRect().width + 'x' + me.getBoundingClientRect().height);
				}
				console.log('[TERM] dpr=' + (window.devicePixelRatio || 'undef'));
				var rowsEl = host.querySelector('.xterm-rows');
				if (rowsEl) {
					console.log('[TERM] row0 children=' + (rowsEl.children[0] ? rowsEl.children[0].childElementCount : 'none') +
						' html=' + (rowsEl.children[0] ? rowsEl.children[0].innerHTML.slice(0, 120) : ''));
				}
				console.log('[TERM] OffscreenCanvas=' + (typeof OffscreenCanvas));
				try { var oc = new OffscreenCanvas(10,10); console.log('[TERM] OCNEW ok, ctx=' + (oc.getContext('2d') ? 'yes' : 'no')); } catch(e2) { console.log('[TERM] OCNEW err: ' + e2.message); }
				console.log('[TERM] cv2d=' + (function(){ try { var c2=document.createElement('canvas'); var cx2=c2.getContext('2d'); if(!cx2) return 'no'; cx2.font = '13px Consolas'; var mt=cx2.measureText('W'); return 'yes w=' + mt.width + ' fba=' + mt.fontBoundingBoxAscent + ' fbd=' + mt.fontBoundingBoxDescent; } catch(e3) { return 'err:'+e3.message; } })());
			console.log('[TERM] cv2d-empty=' + (function(){ try { var c3=document.createElement('canvas'); var cx3=c3.getContext('2d'); if(!cx3) return 'no'; cx3.font = '13px Consolas'; var mt3=cx3.measureText('WWWW'); return 'yes w=' + mt3.width; } catch(e4) { return 'err:'+e4.message; } })());
			console.log('[TERM] span-measure=' + (function(){
				try {
					var doc2 = document;
					var s2 = doc2.createElement('span');
					s2.style.cssText = 'position:absolute;visibility:hidden;white-space:pre;display:inline-block;';
					s2.style.fontSize = '13px'; s2.style.fontFamily = 'Consolas';
					s2.textContent = 'W';
					var holder2 = doc2.createElement('div');
					holder2.style.cssText = 'position:absolute;left:0;top:0;visibility:hidden;';
					holder2.appendChild(s2);
					doc2.body.appendChild(holder2);
					var r2 = s2.getBoundingClientRect();
					var res = 'rect=' + r2.width + 'x' + r2.height + ' off=' + s2.offsetWidth + 'x' + s2.offsetHeight;
					doc2.body.removeChild(holder2);
					return res;
				} catch(e5) { return 'err:'+e5.message; }
			})());
			console.log('[TERM] xterm-measure=' + (function(){
				try {
					var xm = document.querySelector('.xterm-char-measure-element');
					if (!xm) return 'none';
					var r3 = xm.getBoundingClientRect();
					return 'rect=' + r3.width + 'x' + r3.height + ' off=' + xm.offsetWidth + 'x' + xm.offsetHeight;
				} catch(e6) { return 'err:'+e6.message; }
			})());
				try {
				// 公开 API：term.buffer.active 访问当前 buffer
				var ba = t.buffer ? t.buffer.active : null;
				console.log('[TERM] buffer active=' + (ba ? 'yes' : 'no') + ' rows=' + (ba ? ba.length : '?') + ' cursorY=' + (ba ? ba.cursorY : '?'));
				for (var bi = 0; bi < 3; bi++) {
					try {
						var bl = ba ? ba.getLine(bi) : null;
						console.log('[TERM] line' + bi + '=[' + (bl ? bl.translateToString(true) : 'null') + ']');
					} catch(e) { console.log('[TERM] line' + bi + ' err: ' + e.message); }
				}
				// xterm 渲染后的行 textContent
				var rd = host.querySelectorAll('.xterm-rows > div');
				for (var ri = 0; ri < Math.min(rd.length, 3); ri++) {
					console.log('[TERM] rowdiv' + ri + '=[' + (rd[ri].textContent || '').slice(0, 60) + ']');
				}
			} catch(e4) { console.log('[TERM] buf err: ' + e4.message); }
				try {
				// 模拟 fit：先改小再改回，触发 options onMultipleOptionChange → measure
				console.log('[TERM] forcing font re-measure...');
				t.options.fontSize = 13;
				t.options.fontSize = 14;
				t.options.fontSize = 13;
				var cs2 = core._charSizeService;
				console.log('[TERM] after optchange charSize w=' + (cs2 ? cs2.width : 'nocs') + ' h=' + (cs2 ? cs2.height : ''));
				var d2 = core._renderService ? core._renderService.dimensions : null;
				console.log('[TERM] after dims css.cell=' + (d2 && d2.css ? d2.css.cell.width + 'x' + d2.css.cell.height : 'null'));
			} catch(e5) { console.log('[TERM] optchange err: ' + e5.message); }
			try { console.log('[TERM] refresh()...'); core.refresh(0, core.rows-1); console.log('[TERM] refresh done, row0 children now=' + (rowsEl.children[0] ? rowsEl.children[0].childElementCount : 'none')); } catch(e6) { console.log('[TERM] refresh err: ' + e6.message); }
			}
		} catch (e) { console.log('[TERM] diag err: ' + (e && e.stack || e)); }
	`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 800); })`)
	if out := wv.ConsoleOutput(); out != "" {
		fmt.Println("[CONSOLE2]")
		fmt.Println(out)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}
	state := rv.LayoutState()

	var build func(o rendering.RenderObject, depth, idx int) *TNode
	build = func(o rendering.RenderObject, depth, idx int) *TNode {
		if o == nil {
			return nil
		}
		n := &TNode{Depth: depth, Idx: idx}
		st := o.Style()
		if st != nil {
			d := displayName(st.Display)
			if lb := o.LayoutBox(); lb != nil && lb.Parent() != nil {
				if pcs := lb.Parent().Style(); pcs != nil {
					pd := pcs.Display
					if (pd == style.DisplayFlex || pd == style.DisplayInlineFlex) && !lb.IsAbsolutelyPositioned() {
						switch st.Display {
						case style.DisplayInline:
							d = "block"
						case style.DisplayInlineBlock:
							d = "block"
						case style.DisplayInlineFlex:
							d = "flex"
						}
					}
				}
			}
			n.Display = d
			n.Color = hexColor(st.Color)
			n.BG = hexColor(st.BackgroundColor)
			if st.FontSize.Value > 0 {
				n.FontSz = fmt.Sprintf("%.1f", st.FontSize.Value)
			}
			if st.LineHeight.Value > 0 {
				n.Lh = fmt.Sprintf("%.1f", st.LineHeight.Value)
			}
		}
		if nn := o.Node(); nn != nil {
			if el, ok := nn.(*dom.Element); ok {
				n.Tag = el.LocalName()
				n.ID = el.GetAttribute("id")
				n.Class = el.GetAttribute("class")
			} else {
				n.Tag = nn.NodeName()
				if rt, ok := o.(*rendering.RenderText); ok {
					t := strings.TrimSpace(rt.Text())
					if t != "" {
						if len(t) > 40 {
							t = t[:40] + "…"
						}
						n.Text = t
					}
				}
			}
		}
		if lb := o.LayoutBox(); lb != nil && state != nil {
			g := state.GeometryForBox(lb)
			n.X, n.Y = g.Left(), g.Top()
			n.W, n.H = g.BorderBoxWidth(), g.BorderBoxHeight()
		}
		ci := 0
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			if ch := build(c, depth+1, ci); ch != nil {
				n.Children = append(n.Children, *ch)
			}
			ci++
		}
		return n
	}

	root := build(rendering.RenderObject(rv), 0, 0)
	if root == nil {
		log.Fatal("empty tree")
	}
	jsonOut, _ := json.MarshalIndent(root, "", " ")
	os.WriteFile(filepath.Join("dev", "desktop_probe", "term_tree_wb.json"), jsonOut, 0o644)
	printTree(root, 0)
	fmt.Printf("\n[term_probe] tree nodes=%d saved=term_tree_wb.json\n", countNodes(root))
}

func setupTermLoaders(wv *webkit.WebView, webDir string) {
	absDir, _ := filepath.Abs(webDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDir, clean))
				if err != nil {
					return "", err
				}
				// 剥掉 sourceMappingURL 注释——引擎默认 source map loader
				// 按相对路径读不到（probe 本地加载无 map 需要）。
				code := string(data)
				if i := strings.Index(code, "//# sourceMappingURL="); i >= 0 {
					code = code[:i]
				}
				return code, nil
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDir, clean))
				return string(data), err
			}
		}
	}
}

func printTree(n *TNode, depth int) {
	pad := strings.Repeat("  ", depth)
	cls := ""
	if n.Class != "" {
		cls = "." + strings.Fields(n.Class)[0]
	}
	id := ""
	if n.ID != "" {
		id = "#" + n.ID
	}
	text := ""
	if n.Text != "" {
		text = fmt.Sprintf(" %q", n.Text)
	}
	fmt.Printf("%s%s%s%s x=%.0f y=%.0f w=%.0f h=%.0f [%s] fs=%s%s%s\n",
		pad, n.Tag, id, cls, n.X, n.Y, n.W, n.H, n.Display, n.FontSz, text, colorInfo(n))
	for _, c := range n.Children {
		printTree(&c, depth+1)
	}
}

func colorInfo(n *TNode) string {
	if n.BG != "" && n.BG != "#000000" {
		return " bg=" + n.BG
	}
	return ""
}

func countNodes(n *TNode) int {
	c := 1
	for i := range n.Children {
		c += countNodes(&n.Children[i])
	}
	return c
}

func displayName(d style.DisplayType) string {
	switch d {
	case style.DisplayBlock:
		return "block"
	case style.DisplayInline:
		return "inline"
	case style.DisplayInlineBlock:
		return "inline-block"
	case style.DisplayFlex:
		return "flex"
	case style.DisplayInlineFlex:
		return "inline-flex"
	case style.DisplayGrid:
		return "grid"
	case style.DisplayNone:
		return "none"
	}
	return fmt.Sprintf("d%d", d)
}

func hexColor(c style.Color) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}
