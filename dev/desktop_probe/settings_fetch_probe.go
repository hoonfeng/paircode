package main

// Command settings_fetch_probe 直接测 fetch('/api/settings') 异步结果 + location.origin + URL 构造
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

	// 1. location.origin
	fmt.Println("[env] location.origin = " + js(wv, `location.origin`))
	fmt.Println("[env] location.href = " + js(wv, `location.href`))
	fmt.Println("[env] typeof URL = " + js(wv, `typeof URL`))
	fmt.Println("[env] new URL test = " + js(wv, `(function(){ try { var u = new URL('/api/settings', location.origin); return u.toString(); } catch(e) { return 'URL-ERR: ' + (e.message||e); } })()`))
	fmt.Println("[env] new URL /api = " + js(wv, `(function(){ try { var u = new URL('/api/settings', 'http://localhost:9090'); return u.toString(); } catch(e) { return 'URL-ERR: ' + (e.message||e); } })()`))

	// 2. fetch 异步 await 结果
	fmt.Println("[fetch] await fetch('/api/settings'):")
	_, _ = wv.JSInterpreter().RunJS(`(function(){
		window.__fetchResult = 'pending';
		fetch('/api/settings').then(function(r){
			window.__fetchResult = 'ok status=' + r.status + ' body=' + JSON.stringify(r.body);
		}).catch(function(e){
			window.__fetchResult = 'ERR ' + (e && e.message ? e.message : String(e));
		});
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	fmt.Println("  result = " + js(wv, `window.__fetchResult`))

	// 3. apiURL 逻辑复现
	fmt.Println("[apiURL] 复现 api.js apiURL:")
	fmt.Println("  " + js(wv, `(function(){
		var BASE = '/api';
		try {
			var u = new URL(BASE + '/settings', location.origin);
			return u.toString();
		} catch(e) { return 'ERR: ' + (e.message||e); }
	})()`))

	// 4. api.apiGet 全链路（App.vue onMounted 调用）
	fmt.Println("[apiGet] 完整 apiGet 链路:")
	_, _ = wv.JSInterpreter().RunJS(`(function(){
		window.__apiGetResult = 'pending';
		var BASE = '/api';
		function apiURL(path, params) {
			if (!path.startsWith('/')) path = '/' + path;
			var u = new URL(BASE + path, location.origin);
			return u.toString();
		}
		fetch(apiURL('/settings')).then(function(r){
			if (!r.ok) { window.__apiGetResult = 'not ok ' + r.status; return; }
			return r.json();
		}).then(function(data){
			window.__apiGetResult = 'data=' + JSON.stringify(data).slice(0, 400);
		}).catch(function(e){
			window.__apiGetResult = 'ERR ' + (e && e.message ? e.message : String(e));
		});
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	fmt.Println("  result = " + js(wv, `window.__apiGetResult`))
}
