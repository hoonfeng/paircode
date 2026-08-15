// Command toolcfg_probe 打开工具配置弹窗，查询弹窗各滚动容器几何
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
	// 打开工具配置弹窗（确保右侧面板可见）
	fmt.Println("[open] " + js(wv, `(function(){
		var st = window.__state;
		if (st && !st.rightPanelVisible) { st.rightPanelVisible = true; }
		var btn = document.querySelector('.obtn-review-config');
		if (!btn) return 'no btn';
		var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev);
		return 'clicked: ' + !!btn;
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)

	fmt.Println("[geom] " + js(wv, `(function(){
		var pop = document.querySelector('.tool-config-popover');
		var panel = document.querySelector('.tcp-panel');
		var list = document.querySelector('.tcp-switch-list');
		if (!pop) return 'no popover; state.rightPanel=' + (window.__state ? window.__state.rightPanelVisible : '?') + ' toolcfg=' + (window.__state ? window.__state.toolConfigOpen : '?');
		function g(el) {
			if (!el) return null;
			var r = el.getBoundingClientRect();
			var cs = getComputedStyle(el);
			return {cls: el.className, x: r.x, y: r.y, w: r.width, h: r.height,
				scrollH: el.scrollHeight, clientH: el.clientHeight,
				overflow: cs.overflow, overflowY: cs.overflowY, display: cs.display,
				flex: cs.flex, pos: cs.position, maxH: cs.maxHeight, maxW: cs.maxWidth};
		}
		return JSON.stringify({pop: g(pop), panel: g(panel), list: g(list),
			items: document.querySelectorAll('.tcp-switch-item').length,
			cats: document.querySelectorAll('.tcp-cat-header').length});
	})()`))
	// 测试滚动
	fmt.Println("[scroll] " + js(wv, `(function(){
		var list = document.querySelector('.tcp-switch-list');
		if (!list) return 'no list';
		list.scrollTop = 200;
		return 'scrollTop set: ' + list.scrollTop + ' / max=' + (list.scrollHeight - list.clientHeight);
	})()`))
	runJobs(wv)
	fmt.Println("[scroll2] " + js(wv, `(function(){
		var list = document.querySelector('.tcp-switch-list');
		return JSON.stringify({scrollTop: list.scrollTop,
			firstCatY: (function(){
				var c = document.querySelector('.tcp-cat-header');
				if (!c) return -1;
				var r = c.getBoundingClientRect();
				return Math.round(r.y);
			})()});
	})()`))

	// 截图（滚动后）
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	pngBytes, err := wv.Render()
	if err == nil {
		wd, _ := os.Getwd()
		out := filepath.Join(wd, "dev", "desktop_probe", "toolcfg_scroll.png")
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
	// 滚回顶部再截图
	fmt.Println("[reset] " + js(wv, `(function(){
		var list = document.querySelector('.tcp-switch-list');
		list.scrollTop = 0;
		return 'reset';
	})()`))
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	if pngBytes, err := wv.Render(); err == nil {
		wd, _ := os.Getwd()
		out := filepath.Join(wd, "dev", "desktop_probe", "toolcfg_top.png")
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
}
