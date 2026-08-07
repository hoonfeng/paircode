// Command select_ui_shot 打开设置面板（含多个 <select> 下拉），截取整个
// 面板渲染 PNG，并定位各 select 元素的位置——用于与 Edge 原生渲染逐像素对照。
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

	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func runJobs(wv *webkit.WebView) {
	for i := 0; i < 15; i++ {
		wv.JSInterpreter().RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		time.Sleep(20 * time.Millisecond)
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
	for i := 0; i < 4; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
		time.Sleep(600 * time.Millisecond)
		runJobs(wv)
	}

	// 打开设置面板
	fmt.Println("[open] " + js(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		if (!btn) return 'no btn';
		var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev);
		return 'clicked';
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)

	// 列出所有 select 的几何 + 样式
	fmt.Println("[selects] " + js(wv, `(function(){
		var sels = Array.prototype.slice.call(document.querySelectorAll('select'));
		if (!sels.length) return 'NO select elements in DOM';
		return JSON.stringify(sels.map(function(s, i){
			var r = s.getBoundingClientRect();
			var cs = getComputedStyle(s);
			return {i: i, cls: s.className, x: Math.round(r.x), y: Math.round(r.y),
				w: Math.round(r.width), h: Math.round(r.height),
				border: cs.borderTopColor + ' ' + cs.borderTopWidth,
				bg: cs.backgroundColor, color: cs.color,
				value: s.value, options: s.options.length};
		}));
	})()`))

	// 截图
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	pngBytes, err := wv.Render()
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	out := filepath.Join(wd, "dev", "desktop_probe", "settings_select.png")
	f, _ := os.Create(out)
	if f != nil {
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
		f.Close()
		fmt.Println("[shot] → " + out)
	}
}
