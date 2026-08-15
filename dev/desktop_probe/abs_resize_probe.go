// Command abs_resize_probe verifies that an absolutely-positioned element
// filling its parent (top:0;left:0;right:0;bottom:0 — like .term-xterm-wrap
// inside the terminal panel) gets its laid-out size updated when the parent
// height changes (bottom-panel drag).
//
// If the engine does not relayout the absolute child when the parent's
// explicit height changes, ResizeObserver never fires for the wrap element
// → FitAddon.fit() never re-runs → terminal stays at initial cols/rows.
//
// Run: go run ./dev/desktop_probe/abs_resize_probe.go
//go:build ignore

package main

import (
	"fmt"
	"log"

	"wb-ui/bindings"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(log.Ltime)
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

	// 模拟真实布局：panel（面板，height 可变）> tabs（33px）> content（flex:1）> wrap（absolute 填满）
	html := `<html><head><style>
		.panel { position:absolute; left:0; top:100px; width:800px; height:200px; display:flex; flex-direction:column; background:#222; }
		.tabs { height:33px; flex-shrink:0; background:#333; }
		.content { flex:1; min-height:0; position:relative; background:#111; }
		.wrap { position:absolute; top:0; left:0; right:0; bottom:0; background:#0d1117; }
	</style></head><body style="margin:0">
		<div id="panel" class="panel">
			<div class="tabs" id="tabs"></div>
			<div class="content" id="content">
				<div class="wrap" id="wrap"></div>
			</div>
		</div>
	</body></html>`

	wv := webkit.NewWebView()
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	wv.LoadHTML(html)
	interp := wv.JSInterpreter()
	el := interp.GetEventLoop()
	for i := 0; i < 4; i++ {
		el.ProcessTasks(0)
		_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 100); })`)
		el.ProcessTasks(0)
	}
	wv.EnsureLayout()

	box := func(id string) string {
		d, _ := interp.RunJS(`(function(){
			var el = document.getElementById('` + id + `');
			var r = el.getBoundingClientRect();
			return Math.round(r.height*10)/10;
		})()`)
		return d.ToString()
	}

	// 注册 ResizeObserver 观察 wrap
	_, _ = interp.RunJS(`
		window.__ro = new ResizeObserver(function(entries){
			var e = entries[0];
			console.log('[RO] wrap size=' + Math.round(e.contentRect.height*10)/10);
		});
		window.__ro.observe(document.getElementById('wrap'));
	`)
	// 建立快照
	for i := 0; i < 3; i++ {
		el.ProcessTasks(0)
		bindings.ResizeObserverCheck(interp)
		el.ProcessTasks(0)
	}
	fmt.Printf("[ABS] initial: panel=%s tabs=%s content=%s wrap=%s\n",
		box("panel"), box("tabs"), box("content"), box("wrap"))

	// 面板高度 200 → 500（模拟拉伸）
	_, _ = interp.RunJS(`document.getElementById('panel').style.height = '500px';`)
	wv.EnsureLayout()

	// 检测变化
	for i := 0; i < 4; i++ {
		el.ProcessTasks(0)
		bindings.ResizeObserverCheck(interp)
		el.ProcessTasks(0)
	}
	fmt.Printf("[ABS] after-resize: panel=%s tabs=%s content=%s wrap=%s\n",
		box("panel"), box("tabs"), box("content"), box("wrap"))

	if out := wv.ConsoleOutput(); out != "" {
		fmt.Println("[CONSOLE]")
		fmt.Println(out)
	}
	fmt.Println("[ABS] done")
}
