// Command scroll_editor_probe 验证 CM6 长文档滚动后行号 gutter 是否更新
// （用户报告「滚动时初始超出区域的行号都没有绘制」）。
// 流程：加载 120 行文档（含中文行）→ 初始行号 → scrollTop=400 滚动
// → CM6 虚拟化重渲染 → dump 行号 + 截图。
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

	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
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

	// 120 行 Go 文档（含中文注释行）
	var sb strings.Builder
	sb.WriteString("package main\n\n")
	for i := 1; i <= 110; i++ {
		sb.WriteString(fmt.Sprintf("// 函数 fn%d 处理中文注释测试\n", i))
		sb.WriteString(fmt.Sprintf("func fn%d(a, b int) int {\n", i))
		sb.WriteString(fmt.Sprintf("\t// 第 %d 个函数，返回值 a+b\n", i))
		sb.WriteString("\treturn a + b\n}\n\n")
	}
	code := sb.String()
	codeEsc := strings.ReplaceAll(code, "\\", "\\\\")
	codeEsc = strings.ReplaceAll(codeEsc, "\n", "\\n")
	codeEsc = strings.ReplaceAll(codeEsc, "'", "\\'")
	js(wv, `(function(){
		var p = '/workspace/long.go';
		var st = window.__state;
		st.openFiles = [p];
		st.activeFile = p;
		st.fileContents[p] = '`+codeEsc+`';
		return 'injected';
	})()`)
	runJobs(wv)
	time.Sleep(300 * time.Millisecond)
	runJobs(wv)

	dumpGutters := func(tag string) {
		fmt.Println("["+tag+"] " + js(wv, `(function(){
			var out = [];
			var g = document.querySelector('.cm-gutters');
			if (!g) return 'NO gutters';
			var sc = document.querySelector('.cm-scroller');
			out.push('st=' + (sc ? sc.scrollTop.toFixed(0) : '?') + ' sh=' + (sc ? sc.scrollHeight.toFixed(0) : '?') + ' ch=' + (sc ? sc.clientHeight.toFixed(0) : '?'));
			out.push('evt=' + (window.__scrollCount || 0));
			var v = window.__editorView;
			if (v) {
				var vs = v.viewState;
				if (vs && vs.viewport) out.push('viewport=' + vs.viewport.from + '-' + vs.viewport.to);
				if (vs && vs.heightOracle) out.push('oracleLH=' + (vs.heightOracle.lineHeight ? vs.heightOracle.lineHeight.toFixed(2) : '?'));
				if (v.scrollDOM) {
					out.push('scrollDOM=' + (v.scrollDOM.className || v.scrollDOM.tagName) + ' st=' + v.scrollDOM.scrollTop.toFixed(0));
				}
				if (v.observer) {
					out.push('obsInt=' + v.observer.intersecting + ' obsActive=' + v.observer.active);
				}
				out.push('vpEvt=' + (window.__vpAfterEvent || '?') + ' manM=' + (window.__manualMeasure || '?') + ' vpMan=' + (window.__vpAfterManual || '?'));
				if (vs && vs.pixelViewport) {
					out.push('pv=' + vs.pixelViewport.top.toFixed(0) + '-' + vs.pixelViewport.bottom.toFixed(0));
				}
				var cd = document.querySelector('.cm-content');
				if (cd) {
					var cr = cd.getBoundingClientRect();
					out.push('cRect=' + cr.top.toFixed(0) + '-' + cr.bottom.toFixed(0) + ' ih=' + window.innerHeight + ' iw=' + window.innerWidth);
				}
			}
			var ge = g.querySelectorAll('.cm-gutterElement');
			var arr = [];
			for (var i = 0; i < ge.length && i < 10; i++) {
				var r = ge[i].getBoundingClientRect();
				arr.push('"' + ge[i].textContent + '"@' + r.top.toFixed(0) + 'h' + r.height.toFixed(0));
			}
			out.push('first10: ' + arr.join('|'));
			var ge2 = g.querySelectorAll('.cm-gutterElement');
			var last = ge2.length > 0 ? ge2[ge2.length - 1] : null;
			if (last) { var rl = last.getBoundingClientRect(); out.push('last: "' + last.textContent + '"@' + rl.top.toFixed(0)); }
			out.push('elCount=' + ge2.length);
			return out.join(' ');
		})()`))
	}

	// ① 初始（未滚动）
	dumpGutters("init")
	// ★ 行3 中文注释的 RenderText segment dump（中英对齐异常排查）
	{
		rv := wv.RenderView()
		if rv != nil {
			var walk func(ro rendering.RenderObject, depth int)
			walk = func(ro rendering.RenderObject, depth int) {
				if ro == nil {
					return
				}
				if rt, ok := ro.(*rendering.RenderText); ok {
					txt := rt.Text()
					if len(txt) > 0 && strings.Contains(txt, "函数") && !strings.Contains(txt, "fn2") {
						runes := []rune(txt)
						var segs []string
						for _, s := range rt.Segments() {
							var sub string
							if s.Start >= 0 && s.Start < len(runes) {
								end := s.Start + s.Len
								if end > len(runes) {
									end = len(runes)
								}
								sub = string(runes[s.Start:end])
							}
							segs = append(segs, fmt.Sprintf("%q@%.0f w%.1f", sub, s.X, s.Width))
						}
						fmt.Printf("[line3] text=%q segs=[%s]\n", txt, strings.Join(segs, " "))
					}
					for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
						walk(c, depth+1)
					}
					return
				}
				for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
					walk(c, depth+1)
				}
			}
			walk(rv, 0)
		}
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("[shot0] 初始截图")
	shot(wv, wd, "edit_scroll0.png")
	// ★ 重建后再次 dump line3（对比重建前后 segment 是否漂移）
	{
		rv2 := wv.RenderView()
		if rv2 != nil {
			var walk2 func(ro rendering.RenderObject)
			walk2 = func(ro rendering.RenderObject) {
				if ro == nil {
					return
				}
				if rt, ok := ro.(*rendering.RenderText); ok {
					txt := rt.Text()
					if len(txt) > 0 && strings.Contains(txt, "函数") && !strings.Contains(txt, "fn2") {
						runes := []rune(txt)
						var segs []string
						for _, s := range rt.Segments() {
							var sub string
							if s.Start >= 0 && s.Start < len(runes) {
								end := s.Start + s.Len
								if end > len(runes) {
									end = len(runes)
								}
								sub = string(runes[s.Start:end])
							}
							segs = append(segs, fmt.Sprintf("%q@%.0f", sub, s.X))
						}
						fmt.Printf("[line3b] text=%q segs=[%s]\n", txt, strings.Join(segs, " "))
					}
					for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
						walk2(c)
					}
					return
				}
				for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
					walk2(c)
				}
			}
			walk2(rv2)
		}
	}

	// ② 滚动 400px（JS scrollTop + 手动派发 scroll 事件——浏览器里赋值即派发）
	js(wv, `(function(){
		var sc = document.querySelector('.cm-scroller');
		if (!sc) return 'no-scroller';
		window.__scrollCount = 0;
		sc.addEventListener('scroll', function(){ window.__scrollCount = (window.__scrollCount || 0) + 1; });
		window.__ioCb = 0;
		try {
			var io = new IntersectionObserver(function(){ window.__ioCb++; });
			io.observe(document.querySelector('.cm-editor') || document.body);
		} catch(e) { window.__ioErr = e.message; }
		sc.scrollTop = 400;
		sc.dispatchEvent(new Event('scroll'));
		var v = window.__editorView;
		if (v && v.requestMeasure) { try { v.requestMeasure(); } catch(e){ window.__rmErr = e.message; } }
		window.__vpAfterEvent = (v && v.viewState && v.viewState.viewport) ? v.viewState.viewport.from + '-' + v.viewState.viewport.to : '?';
		if (v && typeof v.measure === 'function') {
			try { v.measure(); window.__manualMeasure = 'ok'; }
			catch(e) { window.__manualMeasure = 'ERR:' + e.message; }
		} else { window.__manualMeasure = 'no-fn'; }
		window.__vpAfterManual = (v && v.viewState && v.viewState.viewport) ? v.viewState.viewport.from + '-' + v.viewState.viewport.to : '?';
		return 'set-' + sc.scrollTop + ' evt=' + window.__scrollCount;
	})()`)
	runJobs(wv)
	time.Sleep(200 * time.Millisecond)
	runJobs(wv)
	// ★ Go 侧验证：scroller 的滚动偏移 + contentDOM 祖先链
	{
		rv := wv.RenderView()
		if rv != nil {
			if fr := wv.MainFrame().Frame(); fr != nil {
				if doc := fr.Document(); doc != nil {
					els := doc.GetElementsByClassName("cm-scroller")
					if len(els) > 0 {
						if sb := rv.FindRenderBoxForNode(els[0]); sb != nil {
							ox, oy := rv.BoxScrollOffset(sb)
							fmt.Printf("[go] scroller box offset=(%.0f,%.0f) name=%s\n", ox, oy, sb.RenderName())
						} else {
							fmt.Println("[go] scroller box NOT FOUND in render tree")
						}
					}
					els2 := doc.GetElementsByClassName("cm-content")
					if len(els2) > 0 {
						if cb := rv.FindRenderBoxForNode(els2[0]); cb != nil {
							var chain []string
							for par := cb.Parent(); par != nil; par = par.Parent() {
								var ox, oy float64
								if pb, ok := par.(*rendering.RenderBox); ok {
									ox, oy = rv.BoxScrollOffset(pb)
								}
								chain = append(chain, fmt.Sprintf("%s@%s(%.0f,%.0f)", par.RenderName(), func() string {
									if par.Node() != nil {
										if el, ok := par.Node().(*dom.Element); ok {
											return el.ClassName()
										}
										return "?"
									}
									return "nil-node"
								}(), ox, oy))
							}
							fmt.Println("[go] content parent chain: " + strings.Join(chain, " → "))
							// ★ Node 指针一致性验证
							if len(els) > 0 {
								sb := rv.FindRenderBoxForNode(els[0])
								cbParent := cb.Parent()
								if sb != nil && cbParent != nil {
									same := sb.Node() == cbParent.Node()
									ox1, oy1 := rv.BoxScrollOffset(sb)
									var ox2, oy2 float64
									if pb2, ok := cbParent.(*rendering.RenderBox); ok {
										ox2, oy2 = rv.BoxScrollOffset(pb2)
									}
									fmt.Printf("[go] same=%v sbOff=(%.0f,%.0f) parOff=(%.0f,%.0f) cnt=%d\n",
										same, ox1, oy1, ox2, oy2, rv.ScrollOffsetCount())
									// 直接查 map（绕过 box）
									if rb, ok2 := cbParent.(*rendering.RenderBox); ok2 {
										n := rb.Node()
										fmt.Printf("[go] parentNode=%p scrollerNode=%p\n", n, sb.Node())
									}
								}
							}
						}
					}
				}
			}
		}
	}
	dumpGutters("scrolled400")
	fmt.Println("[shot1] 滚动后截图（无 rebuild）")
	shot(wv, wd, "edit_scroll1.png")

	// ③ 滚动到中部（1200px）
	js(wv, `(function(){
		var sc = document.querySelector('.cm-scroller');
		if (sc) {
			sc.scrollTop = 1200;
			sc.dispatchEvent(new Event('scroll'));
			return 'set-' + sc.scrollTop;
		}
		return 'no-scroller';
	})()`)
	runJobs(wv)
	time.Sleep(200 * time.Millisecond)
	runJobs(wv)
	dumpGutters("scrolled1200")
	// ★ paint 状态：dirty check 是否启用（决定 walk 剪枝）
	if rv := wv.RenderView(); rv != nil {
		fmt.Printf("[g2] pre-shot IsDirty=%v HasBoxScroll=%v offsetCount=%d scrollY=%.0f\n",
			rv.IsDirty(), rv.HasBoxScrollOffset(), rv.ScrollOffsetCount(), func() float64 {
				_, sy := rv.ScrollOffset()
				return sy
			}())
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("[shot2] 滚动1200截图（rebuild）")
	shot(wv, wd, "edit_scroll2.png")

	// ★ 滚动 1200 后渲染树 gutter 元素 dump（行号元素实际位置）
	{
		rv := wv.RenderView()
		if rv != nil {
			found := 0
			textCount := 0
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
					textCount++
				}
				if strings.Contains(cls, "cm-gutters") {
					inGutter = true
				}
				if inGutter {
					found++
					if rt, ok := ro.(*rendering.RenderText); ok && txt != "" {
						var x, y float64
						nseg := 0
						if segs := rt.Segments(); len(segs) > 0 {
							x, y = segs[0].X, segs[0].Y
							nseg = len(segs)
						}
						fmt.Printf("[g2t] %q @(%.0f,%.0f) segs=%d\n", txt, x, y, nseg)
					}
				}
				for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
					walk(c, depth+1, inGutter)
				}
			}
			walk(rv, 0, false)
			fmt.Printf("[g2] gutter 内对象数=%d RenderText=%d\n", found, textCount)
		}
		// ★ 验证：重建后 scroller 的 BoxScrollOffset 是否保留（painter 平移依据）
		if fr := wv.MainFrame().Frame(); fr != nil {
			if doc := fr.Document(); doc != nil {
				els := doc.GetElementsByClassName("cm-scroller")
				if len(els) > 0 {
					if sb := rv.FindRenderBoxForNode(els[0]); sb != nil {
						ox, oy := rv.BoxScrollOffset(sb)
						fmt.Printf("[g2] scroller BoxScrollOffset=(%.0f,%.0f) after-rebuild\n", ox, oy)
					} else {
						fmt.Println("[g2] scroller box NOT FOUND after rebuild")
					}
				}
				// ★ gutter DOM 存在性 vs 渲染树
				gs := doc.GetElementsByClassName("cm-gutters")
				fmt.Printf("[g2] cm-gutters DOM 数=%d\n", len(gs))
				for _, g := range gs {
					gb := rv.FindRenderBoxForNode(g)
					if gb != nil {
						fmt.Printf("[g2] gutter box found: %s (%.0f,%.0f) %.0fx%.0f\n",
							gb.RenderName(), gb.X(), gb.Y(), gb.Width(), gb.Height())
					} else {
						fmt.Println("[g2] gutter box NOT in render tree")
					}
				}
				// 内容行 DOM vs 渲染树
				cs := doc.GetElementsByClassName("cm-content")
				for _, c := range cs {
					cb := rv.FindRenderBoxForNode(c)
					if cb != nil {
						fmt.Printf("[g2] cm-content box found: %s (%.0f,%.0f)\n",
							cb.RenderName(), cb.X(), cb.Y())
					} else {
						fmt.Println("[g2] cm-content box NOT in render tree")
					}
				}
			}
		}
		// ★ JS dump 全部 gutter 元素（文本 + rect + 父列）
		fmt.Println("[g3] " + js(wv, `(function(){
			var g = document.querySelector('.cm-gutters');
			if (!g) return 'no-gutters';
			var out = [];
			out.push('childCount=' + g.children.length);
			for (var c = 0; c < g.children.length; c++) {
				var col = g.children[c];
				var els = col.querySelectorAll('.cm-gutterElement');
				out.push('col' + c + '=[' + col.className + '] els=' + els.length);
				for (var i = 0; i < els.length && i < 3; i++) {
					var r = els[i].getBoundingClientRect();
					out.push('  "' + els[i].textContent + '"@' + r.top.toFixed(0));
				}
			}
			// ★ 诊断：.cm-gutters 的 minHeight / offsetHeight / rect
			var gh = document.querySelector('.cm-gutters');
			if (gh) {
				var gs = getComputedStyle(gh);
				out.push('gutters[minH]=' + (gs.minHeight||'?') + ' offH=' + gh.offsetHeight + ' rectH=' + gh.getBoundingClientRect().height.toFixed(0) + ' disp=' + (gs.display||'?') + ' h=' + (gs.height||'?'));
				var gp = gh.parentElement;
				if (gp) out.push('gutters[parent]=' + gp.className + ' rectH=' + gp.getBoundingClientRect().height.toFixed(0) + ' disp=' + getComputedStyle(gp).display);
			}
			return out.join('|');
		})()`))
	}
	fmt.Println("DONE")
}

func shot(wv *webkit.WebView, wd, name string) {
	pngBytes, err := wv.Render()
	if err != nil {
		log.Printf("render: %v", err)
		return
	}
	out := filepath.Join(wd, "dev", "desktop_probe", name)
	f, err := os.Create(out)
	if err != nil {
		log.Printf("create: %v", err)
		return
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
}
