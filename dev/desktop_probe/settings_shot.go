// Command settings_shot 加载真实 dist，打开设置面板与工具配置弹窗，
// 在每个关键阶段输出 PNG 截图（settings_main.png / settings_panel.png /
// settings_toolcfg.png），用于肉眼/ASCII 检查视觉布局。
//go:build ignore

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

func shot(wv *webkit.WebView, name string) {
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	pngBytes, err := wv.Render()
	if err != nil {
		log.Printf("Render %s: %v", name, err)
		return
	}
	w, h := wv.Width(), wv.Height()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := (y*w + x) * 4
			if off+3 < len(pngBytes) {
				img.SetRGBA(x, y, color.RGBA{R: pngBytes[off], G: pngBytes[off+1], B: pngBytes[off+2], A: pngBytes[off+3]})
			}
		}
	}
	wd, _ := os.Getwd()
	out := filepath.Join(wd, "dev", "desktop_probe", name)
	f, err := os.Create(out)
	if err != nil {
		log.Printf("create %s: %v", out, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Printf("encode %s: %v", out, err)
		return
	}
	log.Printf("shot %dx%d → %s", w, h, out)
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
	shot(wv, "settings_main.png")

	// 打开设置面板
	fmt.Println("[settings] 打开设置面板:")
	fmt.Println("  " + js(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		var out = {btnFound: !!btn};
		if (btn) { var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev); }
		return JSON.stringify(out);
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	shot(wv, "settings_panel.png")

	// 打开工具配置弹窗（先关闭设置面板，模拟真实用户操作）
	fmt.Println("[toolcfg] 关闭设置面板 + 打开工具配置弹窗:")
	fmt.Println("  " + js(wv, `(function(){
		// 关闭设置面板
		var st = window.__state;
		if (st && st.settingsOpen !== undefined) { st.settingsOpen = false; }
		var close = document.querySelector('.modal-close');
		if (close) { var ev = new Event('click', {bubbles:true}); close.dispatchEvent(ev); }
		// 确保右侧面板可见
		if (st && !st.rightPanelVisible) { st.rightPanelVisible = true; }
		return 'settings closed, rightPanel: ' + (st ? !!st.rightPanelVisible : 'no-state');
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	fmt.Println("  " + js(wv, `(function(){
		var btn = document.querySelector('.obtn-review-config');
		var out = {btnFound: !!btn};
		if (btn) { var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev); }
		return JSON.stringify(out);
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	shot(wv, "settings_toolcfg.png")
	log.Println("done")
}
