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
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("[shot0] 初始截图")
	shot(wv, wd, "edit_scroll0.png")

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
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("[shot1] 滚动后截图")
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
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("[shot2] 滚动1200截图")
	shot(wv, wd, "edit_scroll2.png")
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
