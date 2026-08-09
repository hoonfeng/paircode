// Command term_resize_probe simulates the real "terminal panel resize"
// flow on wb-ui: after xterm+fit is set up on a fixed container, the
// container height changes (panel drag) → ResizeObserver fires → fit()
// re-runs → cols/rows must be recomputed.
//
// Real flow (TerminalPanel.vue observeActiveTerm):
//   new ResizeObserver(() => debounce 100ms → fit() + ws.resize())
// Engine: host main loop calls bindings.ResizeObserverCheck() every frame
// after layout → detects size change → fires observer callback.
//
// This probe registers a ResizeObserver on the term-host container exactly
// like TerminalPanel, then changes the container height via JS (simulating
// bottom-panel drag), runs ResizeObserverCheck (like the host loop), waits
// for the debounced fit(), and reports whether rows/cols were recomputed.
//
// Run: go run ./dev/desktop_probe/term_resize_probe.go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"wb-ui/bindings"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	webDir := filepath.Join(wd, "cmd", "companion", "web-ui")
	htmlData, err := os.ReadFile(filepath.Join(webDir, "ide_ref_termfit.html"))
	if err != nil {
		log.Fatalf("read ide_ref_termfit.html: %v", err)
	}
	log.Printf("[resize] webDir=%s html=%d bytes", webDir, len(htmlData))

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
	setupLoaders(wv, webDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()

	wv.LoadHTML(string(htmlData))
	_, _ = wv.JSInterpreter().RunJS(`
		if (typeof window.onload === 'function') { window.onload(); }
	`)
	interp := wv.JSInterpreter()
	el := interp.GetEventLoop()
	runLoop := func(times int, ms int) {
		for i := 0; i < times; i++ {
			el.ProcessTasks(0)
			_, _ = interp.RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
			el.ProcessTasks(0)
		}
	}
	runLoop(12, 200) // 等 xterm 创建 + fit 完成

	if out := wv.ConsoleOutput(); out != "" {
		fmt.Println("[CONSOLE-INIT]")
		fmt.Println(out)
	}

	// 1) 注册 ResizeObserver（模拟 TerminalPanel.observeActiveTerm：回调 debounce 100ms 后 fit）
	_, _ = interp.RunJS(`
		(function(){
			var host = document.getElementById('term-host');
			window.__ro = new ResizeObserver(function(entries){
				console.log('[RO] callback fired entries=' + entries.length);
				var t = window.__term;
				if (!t) { console.log('[RO] no term'); return; }
				var fit = window.__fit;
				var before = {cols: t.cols, rows: t.rows};
				try { fit.fit(); } catch(e) { console.log('[RO] fit err ' + e); }
			});
			window.__ro.observe(host);
			console.log('[RO] observer registered on term-host');
		})();
	`)
	el.ProcessTasks(0)
	// 建立 RO 初始快照（真实 desktop 主循环每帧调 ResizeObserverCheck）
	for i := 0; i < 3; i++ {
		el.ProcessTasks(0)
		bindings.ResizeObserverCheck(interp)
		el.ProcessTasks(0)
	}

	diag := func(label string) {
		d, _ := interp.RunJS(`(function(){
			var t = window.__term; if (!t) return 'no term';
			var core = t._core;
			var d = core._renderService && core._renderService.dimensions;
			var host = document.getElementById('term-host');
			var hr = host.getBoundingClientRect();
			return JSON.stringify({cols:t.cols, rows:t.rows, cssCellH: d ? d.css.cell.height : null, hostH: Math.round(hr.height*10)/10});
		})()`)
		fmt.Printf("[DIAG-%s] %s\n", label, d.ToString())
	}

	diag("BEFORE")

	// 2) 模拟面板拉伸：高度 160 → 400（JS 改样式 = Vue bottomPanelHeight 更新）
	_, _ = interp.RunJS(`
		(function(){
			var host = document.getElementById('term-host');
			host.style.height = '400px';
			console.log('[RESIZE] host height set to 400px');
		})();
	`)
	// 让 Vue patch/样式生效 → 布局
	wv.EnsureLayout()

	// 3) 模拟宿主主循环：每帧 ResizeObserverCheck（布局后）
	for i := 0; i < 6; i++ {
		el.ProcessTasks(0)
		bindings.ResizeObserverCheck(interp)
		el.ProcessTasks(0)
		_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 150); })`)
		el.ProcessTasks(0)
	}

	diag("AFTER-RESIZE")

	if out := wv.ConsoleOutput(); out != "" {
		fmt.Println("[CONSOLE-AFTER]")
		fmt.Println(out)
	}
	fmt.Println("[RESIZE] done")
}

func setupLoaders(wv *webkit.WebView, webDir string) {
	absDir, _ := filepath.Abs(webDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := src
				for _, p := range []string{"file://", "./"} {
					clean = trimPrefix(clean, p)
				}
				data, err := os.ReadFile(filepath.Join(absDir, clean))
				if err != nil {
					return "", err
				}
				code := string(data)
				if i := index(code, "//# sourceMappingURL="); i >= 0 {
					code = code[:i]
				}
				return code, nil
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				clean := href
				for _, p := range []string{"file://", "./"} {
					clean = trimPrefix(clean, p)
				}
				data, err := os.ReadFile(filepath.Join(absDir, clean))
				return string(data), err
			}
		}
	}
}

func trimPrefix(s, p string) string {
	if len(s) >= len(p) && s[:len(p)] == p {
		return s[len(p):]
	}
	return s
}

func index(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
