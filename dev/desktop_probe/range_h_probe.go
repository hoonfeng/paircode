package main

// Command range_h_probe 输出 wb-ui 中 range input 的实际布局尺寸
import (
	"fmt"
	"log"
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
	html := `<html><head><style>
		body { margin: 0; background: #0d1117; font-family: "Segoe UI", sans-serif; }
		.wrap { padding: 60px 80px; }
		.row { display: flex; align-items: center; gap: 8px; padding: 6px 0; }
		.row label { width: 120px; font-size: 13px; color: #e6edf3; }
		.row input[type="range"] { flex: 1; }
	</style></head><body>
	<div class="wrap">
	  <div class="row"><label>温度</label><input type="range" min="0" max="2" step="0.1" value="0.3" /></div>
	  <div class="row"><label>音量</label><input type="range" min="0" max="100" step="1" value="42" /></div>
	</div>
	</body></html>`

	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style2, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}, text)
	}

	wv := webkit.NewWebView()
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	wv.SetConsoleLogger(&jsc.BufferLogger{})
	desktopbridge.Init(wv)
	wv.LoadHTML(html)
	for i := 0; i < 2; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
		time.Sleep(350 * time.Millisecond)
		runJobs(wv)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)

	fmt.Println("[r1] " + js(wv, `(function(){
		var r = document.querySelector('input[type="range"]');
		var rc = r.getBoundingClientRect();
		return JSON.stringify({x: Math.round(rc.left), y: Math.round(rc.top), w: Math.round(rc.width), h: Math.round(rc.height)});
	})()`))
	fmt.Println("[row] " + js(wv, `(function(){
		var row = document.querySelector('.row');
		var rc = row.getBoundingClientRect();
		return JSON.stringify({x: Math.round(rc.left), y: Math.round(rc.top), w: Math.round(rc.width), h: Math.round(rc.height)});
	})()`))
	fmt.Println("[cs] " + js(wv, `(function(){
		var r = document.querySelector('input[type="range"]');
		var cs = getComputedStyle(r);
		return JSON.stringify({h: cs.height, minH: cs.minHeight, boxSizing: cs.boxSizing, padding: cs.padding, margin: cs.margin});
	})()`))
}
