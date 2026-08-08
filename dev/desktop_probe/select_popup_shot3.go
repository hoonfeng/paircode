// Command select_popup_shot3 在真实设置面板中验证 select popup 打开位置：
// 点击服务商 select（#1）后，popup 应从其底部（y=236+27=263）下方展开，
// 不覆盖组件本身。
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/app"
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
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 定位服务商 select（第 2 个 select，i=1）
	doc := wv.Document()
	sels := doc.GetElementsByTagName("select")
	if len(sels) < 2 {
		fmt.Println("NO selects in panel")
		os.Exit(1)
	}
	sel := sels[1] // 服务商
	rv := wv.RenderView()
	h := app.NewHostForTest(wv, 1280, 800)

	// 打开 popup
	h.MockSelectClick(sel, rv)
	wv.EnsureLayout()
	wv.RebuildRenderTree()
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	if h.SelectPopupOpen() {
		fmt.Println("[popup] OPENED")
	} else {
		fmt.Println("[popup] NOT OPENED")
		os.Exit(1)
	}

	// popup 几何 vs select 几何
	fmt.Println("[popup-info] " + js(wv, `(function(){
		var p = document.querySelector('.select-popup');
		if (!p) return 'no popup';
		var pr = p.getBoundingClientRect();
		var sels = document.querySelectorAll('select');
		var s = sels[1]; // 服务商
		var sr = s.getBoundingClientRect();
		return 'select at(' + sr.x + ',' + sr.y + ') w=' + sr.width + ' h=' + sr.height +
			' popup at(' + pr.x + ',' + pr.y + ') w=' + pr.width + ' h=' + pr.height +
			' options=' + p.children.length +
			' gap=' + Math.round(pr.y - (sr.y + sr.height));
	})()`))
	fmt.Println("DONE")
}
