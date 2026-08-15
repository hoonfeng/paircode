//go:build ignore

package main

// Command settings_fetch2_probe 获取 fetch('/api/settings') 的完整响应细节（status+body）
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

	// fetch /api/settings 完整响应
	_, _ = wv.JSInterpreter().RunJS(`(function(){
		window.__r = 'pending';
		fetch('/api/settings').then(function(r){
			return r.text().then(function(t){
				window.__r = 'status=' + r.status + ' ok=' + r.ok + ' body=' + t.slice(0, 300);
			});
		}).catch(function(e){
			window.__r = 'ERR ' + (e && e.message ? e.message : String(e));
		});
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	fmt.Println("[fetch] /api/settings 完整响应:")
	fmt.Println("  " + js(wv, `window.__r`))

	// 用完整 URL http://localhost:9090/api/settings
	_, _ = wv.JSInterpreter().RunJS(`(function(){
		window.__r2 = 'pending';
		fetch('http://localhost:9090/api/settings').then(function(r){
			return r.text().then(function(t){
				window.__r2 = 'status=' + r.status + ' ok=' + r.ok + ' body=' + t.slice(0, 300);
			});
		}).catch(function(e){
			window.__r2 = 'ERR ' + (e && e.message ? e.message : String(e));
		});
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	fmt.Println("[fetch] 完整 URL http://localhost:9090/api/settings:")
	fmt.Println("  " + js(wv, `window.__r2`))

	// 直接测试其他 API（health）
	_, _ = wv.JSInterpreter().RunJS(`(function(){
		window.__r3 = 'pending';
		fetch('/api/health').then(function(r){
			return r.text().then(function(t){
				window.__r3 = 'status=' + r.status + ' body=' + t.slice(0, 200);
			});
		}).catch(function(e){
			window.__r3 = 'ERR ' + (e && e.message ? e.message : String(e));
		});
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	fmt.Println("[fetch] /api/health:")
	fmt.Println("  " + js(wv, `window.__r3`))
}
