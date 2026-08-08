package main

// Command settings_load_probe 定位 loadSettings 未生效的根因：
//   1. 打开面板前/后 settingsLoaded 状态
//   2. 点击「撤销」按钮（resetForm→loadSettings）后 input 是否更新
//   3. 手动 watch 触发验证
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

	// 打开前 settingsLoaded
	fmt.Println("[before] 打开面板前 state:")
	fmt.Println("  " + js(wv, `(function(){
		var s = window.__state;
		return JSON.stringify({settingsLoaded: s.settingsLoaded,
			hasSettings: Object.keys(s.settings || {}).length,
			provider: s.settings.provider, maxTokens: s.settings.maxTokens});
	})()`))

	// 打开设置面板
	fmt.Println("[open] 打开设置面板:")
	fmt.Println("  " + js(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		var out = {btnFound: !!btn};
		if (btn) { var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev); }
		return JSON.stringify(out);
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 打开后立即读
	fmt.Println("[after] 打开后 input 值:")
	fmt.Println("  " + js(wv, `(function(){
		var out = [];
		var els = document.querySelectorAll('.settings-modal input, .settings-modal select');
		for (var i=0;i<els.length && i<10;i++) {
			var el = els[i];
			out.push({t: el.tagName, ty: el.getAttribute('type'), v: el.value});
		}
		return JSON.stringify(out);
	})()`))

	// 等 1 秒再看
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 800); })`)
	time.Sleep(900 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("[wait] 等 900ms 后 input 值:")
	fmt.Println("  " + js(wv, `(function(){
		var out = [];
		var els = document.querySelectorAll('.settings-modal input, .settings-modal select');
		for (var i=0;i<els.length && i<10;i++) {
			var el = els[i];
			out.push({t: el.tagName, ty: el.getAttribute('type'), v: el.value});
		}
		return JSON.stringify(out);
	})()`))

	// 点击「撤销」按钮（resetForm → loadSettings）
	fmt.Println("[reset] 点击撤销按钮:")
	fmt.Println("  " + js(wv, `(function(){
		var btns = document.querySelectorAll('.modal-footer button');
		var out = {found: btns.length, labels: []};
		for (var i=0;i<btns.length;i++) out.labels.push(btns[i].textContent.trim());
		if (btns.length > 0) { var ev = new Event('click', {bubbles:true}); btns[0].dispatchEvent(ev); }
		return JSON.stringify(out);
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("[reset-after] 撤销后 input 值:")
	fmt.Println("  " + js(wv, `(function(){
		var out = [];
		var els = document.querySelectorAll('.settings-modal input, .settings-modal select');
		for (var i=0;i<els.length && i<10;i++) {
			var el = els[i];
			out.push({t: el.tagName, ty: el.getAttribute('type'), v: el.value});
		}
		return JSON.stringify(out);
	})()`))
}
