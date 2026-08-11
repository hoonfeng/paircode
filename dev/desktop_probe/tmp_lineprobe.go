// Command tmp_lineprobe：dump .cm-content 前 8 行的渲染树文本位置，与像素对比。
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
	}
	for i := 0; i < 8; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 50); })`)
		time.Sleep(60 * time.Millisecond)
	}

	code := "package main\n\n"
	for i := 1; i <= 40; i++ {
		code += fmt.Sprintf("// 函数 fn%d 处理中文注释测试\nfunc fn%d(a, b int) int {\n\t// 第 %d 个函数，返回值 a+b\n\treturn a + b\n}\n\n", i, i, i)
	}
	code = strings.ReplaceAll(code, "\\", "\\\\")
	code = strings.ReplaceAll(code, "\n", "\\n")
	code = strings.ReplaceAll(code, "'", "\\'")
	_, _ = wv.JSInterpreter().RunJS(`(function(){
		var p = '/workspace/tmp.go';
		var st = window.__state;
		st.openFiles = [p];
		st.activeFile = p;
		st.fileContents[p] = '` + code + `';
		return 'injected';
	})()`)
	for i := 0; i < 6; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 100); })`)
		time.Sleep(120 * time.Millisecond)
	}
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
	time.Sleep(200 * time.Millisecond)

	// DOM 行文本
	domText, _ := wv.JSInterpreter().RunJS(`(function(){
		var ls = document.querySelectorAll('.cm-content .cm-line');
		var out = [];
		for (var i = 0; i < ls.length && i < 14; i++) {
			out.push(i + ':' + JSON.stringify(ls[i].textContent));
		}
		return out.join('|');
	})()`)
	fmt.Println("[dom-lines] " + domText.ToString())

	// 渲染树 dump：找 .cm-line 的 RenderObject，打印文本 + y
	rv := wv.RenderView()
	if rv != nil {
		var walk func(ro rendering.RenderObject, depth int)
		walk = func(ro rendering.RenderObject, depth int) {
			if ro == nil {
				return
			}
			// RenderBlockFlow with class cm-line 或含文本
			if rb, ok := ro.(*rendering.RenderBlockFlow); ok {
				el := rb.Node()
				if el != nil {
					if el2, ok2 := el.(*dom.Element); ok2 {
						if strings.Contains(el2.ClassName(), "cm-line") {
							// 收集子 RenderText
							var texts []string
							var walkT func(ro2 rendering.RenderObject)
							walkT = func(ro2 rendering.RenderObject) {
								if rt, ok3 := ro2.(*rendering.RenderText); ok3 {
									t := rt.OriginalText()
									if len(t) > 26 {
										t = t[:26]
									}
									texts = append(texts, fmt.Sprintf("%q", t))
								}
								for c := ro2.FirstChild(); c != nil; c = c.NextSibling() {
									walkT(c)
								}
							}
							walkT(rb)
							fmt.Printf("[rt-line] y=%.0f h=%.1f %s\n", rb.Y(), rb.Height(), strings.Join(texts, " "))
						}
					}
				}
			}
			for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c, depth+1)
			}
		}
		walk(rv, 0)
	}

	// ★ 实验：RebuildRenderTree 后再次 dump + 截图对比
	if false { // ★ 实验开关：禁用 rebuild 对比
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.RebuildRenderTree()
			}
		}
	}
	interp.RunJobs()
	time.Sleep(200 * time.Millisecond)
	// 重建后 DOM .cm-line 数量
	domAfter, _ := wv.JSInterpreter().RunJS(`(function(){
		var ls = document.querySelectorAll('.cm-content .cm-line');
		var out = ['count=' + ls.length];
		for (var i = 0; i < ls.length && i < 3; i++) {
			out.push(i + ':' + JSON.stringify(ls[i].textContent));
		}
		return out.join('|');
	})()`)
	fmt.Println("[dom-after] " + domAfter.ToString())
	rv2 := wv.RenderView()
	if rv2 != nil {
		var walk2 func(ro rendering.RenderObject, depth int)
		walk2 = func(ro rendering.RenderObject, depth int) {
			if ro == nil {
				return
			}
			if rb, ok := ro.(*rendering.RenderBlockFlow); ok {
				el := rb.Node()
				if el != nil {
					if el2, ok2 := el.(*dom.Element); ok2 {
						if strings.Contains(el2.ClassName(), "cm-line") {
							var texts []string
							var walkT func(ro2 rendering.RenderObject)
							walkT = func(ro2 rendering.RenderObject) {
								if rt, ok3 := ro2.(*rendering.RenderText); ok3 {
									t := rt.OriginalText()
									if len(t) > 26 {
										t = t[:26]
									}
									texts = append(texts, fmt.Sprintf("%q", t))
								}
								for c := ro2.FirstChild(); c != nil; c = c.NextSibling() {
									walkT(c)
								}
							}
							walkT(rb)
							fmt.Printf("[rt-line2] y=%.0f h=%.1f %s\n", rb.Y(), rb.Height(), strings.Join(texts, " "))
						}
					}
				}
			}
			for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
				walk2(c, depth+1)
			}
		}
		walk2(rv2, 0)
	}

	// 截图
	pngBytes, err := wv.Render()
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	out := filepath.Join(wd, "dev", "desktop_probe", "tmp_line.png")
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
	fmt.Println("[shot] tmp_line.png")
	fmt.Println("DONE")
}
