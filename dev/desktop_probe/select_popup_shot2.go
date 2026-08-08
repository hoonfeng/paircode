// Command select_popup_shot2 在真实设置面板中点击服务商 select，检查
// popup 打开位置（应在 select 底部下方）与 option 渲染，并截图对比。
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
	js(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		if (!btn) return 'no btn';
		var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev);
		return 'clicked';
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)

	// 定位服务商 select（value=deepseek 的那个）
	fmt.Println("[selects] " + js(wv, `(function(){
		var sels = Array.prototype.slice.call(document.querySelectorAll('select'));
		return JSON.stringify(sels.map(function(s, i){
			var r = s.getBoundingClientRect();
			return {i: i, x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width), h: Math.round(r.height), value: s.value, txt: s.selectedOptions.length ? s.selectedOptions[0].textContent : '?'};
		}));
	})()`))

	// 通过 JS 直接 dispatch click 打开 popup？—— popup 由 host 层打开，
	// 这里改用 app.Host 模拟。但设置面板 host 复杂，先直接验证：
	// 用 JS 点击 select（会触发 Vue 但不会打开 popup），再调用 host 层。
	// 简化：直接用 host 层 mock。
	fmt.Println("DONE-PHASE1")
}
