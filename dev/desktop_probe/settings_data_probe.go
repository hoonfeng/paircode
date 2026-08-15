//go:build ignore

package main

// Command settings_data_probe 验证 /api/settings 数据加载链路：
//   1. fetch('/api/settings') 返回什么结构
//   2. state.settings 是否被正确赋值
//   3. SettingsModal 的 local 是否能取到值
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
	wv.SetConsoleLogger(&jsc.BufferLogger{})
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

	// 1. fetch /api/settings 原始返回
	fmt.Println("[fetch] /api/settings:")
	fmt.Println("  " + js(wv, `(function(){
		return fetch('/api/settings').then(function(r){ return r.text(); });
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 200); })`)
	time.Sleep(300 * time.Millisecond)
	runJobs(wv)

	// 2. state.settings 内容
	fmt.Println("[state] state.settings 键列表:")
	fmt.Println("  " + js(wv, `(function(){
		var s = window.__state ? window.__state.settings : null;
		if (!s) return 'no state';
		var keys = Object.keys(s);
		return JSON.stringify({keys: keys.slice(0, 30), loaded: window.__state.settingsLoaded,
			provider: s.provider, baseURL: s.baseURL, temperature: s.temperature,
			hasSettings: typeof s.settings === 'object' && s.settings !== null,
			innerKeys: s.settings ? Object.keys(s.settings).slice(0, 30) : []});
	})()`))

	// 3. 打开设置面板看 local 值
	fmt.Println("[modal] 打开设置面板:")
	fmt.Println("  " + js(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		var out = {btnFound: !!btn};
		if (btn) { var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev); }
		return JSON.stringify(out);
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(350 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("  " + js(wv, `(function(){
		var out = {};
		var inp = document.querySelector('.setting-row input[type="text"]');
		out.firstInput = inp ? inp.value : null;
		var range = document.querySelector('.setting-row input[type="range"]');
		out.rangeVal = range ? range.value : null;
		out.rangeMin = range ? range.min : null;
		out.rangeMax = range ? range.max : null;
		var sel = document.querySelector('.setting-row select');
		out.selectVal = sel ? sel.value : null;
		var temp = document.querySelector('.range-val');
		out.rangeValSpan = temp ? temp.textContent.trim() : null;
		return JSON.stringify(out);
	})()`))

	// 4. API 可用性：直接 api.apiGet
	fmt.Println("[api] 通过 api.apiGet('/settings'):")
	fmt.Println("  " + js(wv, `(function(){
		return window.__api ? 'api exists' : 'no __api';
	})()`))
}
