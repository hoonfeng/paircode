// Command term_panel_resize_probe fully simulates the real terminal panel:
//
//   .panel (height: Npx, flex column)          ← bottom-panel (drag target)
//     .tabs (33px)
//     .content (flex:1, min-height:0, position:relative)
//       .wrap (position:absolute, fills content)  ← .term-xterm-wrap
//         xterm opens here (like TerminalPanel.vue)
//
// Then the panel height changes (drag) and we check whether ResizeObserver
// fires for .wrap and whether FitAddon.fit() recomputes rows — exactly the
// user-reported issue: "terminal panel resized but terminal stays initial".
//
// Run: go run ./dev/desktop_probe/term_panel_resize_probe.go
//go:build ignore

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

	html := `<html><head>
	<link rel="stylesheet" href="./node_modules/@xterm/xterm/css/xterm.css" />
	<script src="./node_modules/@xterm/xterm/lib/xterm.js"></script>
	<script src="./node_modules/@xterm/addon-fit/lib/addon-fit.js"></script>
	<style>
		/* 完整复刻 TerminalPanel 高度链：bottom-panel > panel-content > terminal-panel(height:100%) > term-content > wrap */
		.panel { position:absolute; left:0; top:60px; width:800px; height:200px;
			display:flex; flex-direction:column; background:#161b22; }
		.tabs { height:33px; flex-shrink:0; background:#21262d; }
		.panel-content { flex:1; overflow:hidden; }
		.terminal-panel { height:100%; display:flex; flex-direction:column; }
		.term-content { flex:1; min-height:0; position:relative; overflow:hidden; }
		.wrap { position:absolute; top:0; left:0; right:0; bottom:0; }
	</style></head><body style="margin:0">
	<div id="panel" class="panel">
		<div class="tabs"></div>
		<div class="panel-content">
			<div class="terminal-panel">
				<div class="term-content" id="content">
					<div class="wrap" id="wrap"></div>
				</div>
			</div>
		</div>
	</div>
	</body></html>`

	wv := webkit.NewWebView()
	setupLoaders(wv, webDir)
	_ = wv.JSInterpreter()
	wv.LoadHTML(html)
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
	// 模拟 TerminalPanel.vue：open + 立即 fit + RO 观察 wrap
	if _, err := interp.RunJS(`
		(function(){
			var host = document.getElementById('wrap');
			var FitCtor = window.FitAddon && (window.FitAddon.FitAddon || window.FitAddon);
			var term = new window.Terminal({cursorBlink:false, cursorStyle:'bar', fontSize:13,
				fontFamily:"'Consolas', monospace", cols:80, rows:24, scrollback:200, convertEol:true, rendererType:'dom'});
			window.__term = term;
			var fit = new FitCtor();
			term.loadAddon(fit);
			window.__fit = fit;
			term.open(host);
			fit.fit();
			console.log('[PANEL] initial fit: cols=' + term.cols + ' rows=' + term.rows +
				' host=' + Math.round(host.getBoundingClientRect().height*10)/10);
			window.__ro = new ResizeObserver(function(entries){
				console.log('[PANEL] RO fired');
				var t = window.__term;
				var b = {cols: t.cols, rows: t.rows};
				try { window.__fit.fit(); } catch(e) { console.log('[PANEL] fit err ' + e); }
				console.log('[PANEL] FIT before=' + JSON.stringify(b) + ' after=' + t.cols + 'x' + t.rows +
					' host=' + Math.round(host.getBoundingClientRect().height*10)/10);
			});
			window.__ro.observe(host);
		})();
	`); err != nil {
		log.Printf("[PANEL] RunJS err: %v", err)
	}
	runLoop(10, 200) // 等 xterm + fit 完成
	// 建立 RO 初始快照（真实 desktop 主循环每帧调 ResizeObserverCheck）
	for i := 0; i < 3; i++ {
		el.ProcessTasks(0)
		bindings.ResizeObserverCheck(interp)
		el.ProcessTasks(0)
	}
	st, _ := interp.RunJS(`(function(){
		return JSON.stringify({hasTerm: !!window.__term, hasFit: !!window.__fit,
			wrapChildren: document.getElementById('wrap').children.length});
	})()`)
	fmt.Println("[STATUS] " + st.ToString())

	diag := func(label string) {
		d, _ := interp.RunJS(`(function(){
			var t = window.__term;
			var host = document.getElementById('wrap');
			var panel = document.getElementById('panel');
			var pr = panel.getBoundingClientRect();
			var hr = host.getBoundingClientRect();
			return JSON.stringify({cols:t.cols, rows:t.rows, panelH: Math.round(pr.height*10)/10, wrapH: Math.round(hr.height*10)/10});
		})()`)
		fmt.Printf("[DIAG-%s] %s\n", label, d.ToString())
	}

	diag("BEFORE")

	// 面板拉伸：200 → 500（模拟拖 panel-resizer → bottomPanelHeight 更新）
	_, _ = interp.RunJS(`document.getElementById('panel').style.height = '500px';`)
	wv.EnsureLayout()

	// 检查 computedStyle 的 height（fit 依赖）
	csd, _ := interp.RunJS(`(function(){
		var wrap = document.getElementById('wrap');
		var cs = window.getComputedStyle(wrap);
		var el = wrap.getBoundingClientRect();
		return JSON.stringify({gcsH: cs.getPropertyValue('height'), gcsW: cs.getPropertyValue('width'),
			rectH: Math.round(el.height*10)/10, cssH: cs.height, cssW: cs.width});
	})()`)
	fmt.Println("[GCS-AFTER-RESIZE] " + csd.ToString())
	// 驱动事件循环真实时间（等 100ms debounce 到期 + RO fire + fit）
	for i := 0; i < 10; i++ {
		el.ProcessTasks(0)
		bindings.ResizeObserverCheck(interp)
		el.ProcessTasks(0)
		_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 250); })`)
		el.ProcessTasks(0)
	}

	diag("AFTER-RESIZE")

	if out := wv.ConsoleOutput(); out != "" {
		fmt.Println("[CONSOLE]")
		fmt.Println(out)
	}
	fmt.Println("[PANEL] done")
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
