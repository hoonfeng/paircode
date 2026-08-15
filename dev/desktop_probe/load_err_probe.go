//go:build ignore

package main

// Command load_err_probe 打开设置面板后打印全部 console 输出（含错误），
// 并直接手动执行 loadSettings 等价逻辑，定位抛异常位置。
import (
	"fmt"
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
	logger := &jsc.BufferLogger{}
	wv.SetConsoleLogger(logger)
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	for i := 0; i < 4; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
		time.Sleep(600 * time.Millisecond)
		runJobs(wv)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)

	fmt.Println("[before-console] 加载后 console:")
	fmt.Println(logger.String())

	// 打开设置面板
	fmt.Println("[open]")
	fmt.Println("  " + js(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		if (btn) { var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev); }
		return 'clicked';
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("[after-console] 打开面板后 console:")
	fmt.Println(logger.String())

	// 手动执行 loadSettings 等价逻辑，逐行定位异常
	fmt.Println("[manual] 手动执行 loadSettings 关键步骤:")
	fmt.Println("  " + js(wv, `(function(){
		var s = window.__state.settings;
		var out = {keys: Object.keys(s).length, provider: s.provider, baseURL: s.baseURL};
		try {
			var p2 = s.provider || '';
			var b2 = s.baseURL || '';
			var k2 = s.apiKey || '';
			var em = s.executeModel || s.model || '';
			var t = s.temperature ?? 0.3;
			var mt = s.maxTokens || 16384;
			var c = s.contextMaxTokens || 1000000;
			var tm = s.thinkingMode || 'thinking';
			var id = (s.ignoreDirs || []).join(', ');
			out.step = 'all ok';
			out.values = {p2: p2, b2: b2, em: em, t: t, mt: mt, c: c, tm: tm, id: id};
		} catch(e) {
			out.step = 'EXC: ' + (e && e.message ? e.message : String(e));
		}
		return JSON.stringify(out);
	})()`))
}
