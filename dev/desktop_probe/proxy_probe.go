//go:build ignore

package main

// Command proxy_probe 验证 wb-ui jsc 引擎中 JS Proxy 的支持情况
// （Vue 3 reactive 依赖 Proxy，若 Proxy set trap 不触发 → 组件不重渲染）
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

	// Proxy 测试
	fmt.Println("[proxy] typeof Proxy = " + js(wv, `typeof Proxy`))
	fmt.Println("[proxy] Proxy 基本测试:")
	fmt.Println("  " + js(wv, `(function(){
		try {
			var target = {a: 1};
			var sets = [];
			var p = new Proxy(target, {
				get: function(t, k) { return t[k]; },
				set: function(t, k, v) { sets.push(k + '=' + v); t[k] = v; return true; }
			});
			p.a = 2;
			p.b = 3;
			return JSON.stringify({targetA: target.a, sets: sets});
		} catch(e) { return 'ERR ' + (e && e.message ? e.message : String(e)); }
	})()`))

	// reactive 深层嵌套 + Vue 常用操作（Object.keys / has / ownKeys）
	fmt.Println("[proxy] Vue reactive 常用 trap 测试:")
	fmt.Println("  " + js(wv, `(function(){
		try {
			var p = new Proxy({x: {y: 1}}, {
				get: function(t, k, r) { return Reflect.get(t, k, r); },
				set: function(t, k, v, r) { return Reflect.set(t, k, v, r); },
				has: function(t, k) { return k in t; },
				ownKeys: function(t) { return Reflect.ownKeys(t); }
			});
			p.x.y = 42;
			var keys = Object.keys(p);
			var hasY = 'y' in p.x;
			return JSON.stringify({xy: p.x.y, keys: keys, hasY: hasY});
		} catch(e) { return 'ERR ' + (e && e.message ? e.message : String(e)); }
	})()`))

	// Vue 3 reactive 核心：effect 收集依赖（简化模拟）
	fmt.Println("[proxy] 模拟 Vue effect 依赖收集:")
	fmt.Println("  " + js(wv, `(function(){
		try {
			var target = {a: 1};
			var deps = [];
			var p = new Proxy(target, {
				get: function(t, k, r) { deps.push('get:' + k); return Reflect.get(t, k, r); },
				set: function(t, k, v, r) { deps.push('set:' + k); return Reflect.set(t, k, v, r); }
			});
			var sum = p.a + 1;
			p.a = 10;
			return JSON.stringify({sum: sum, deps: deps});
		} catch(e) { return 'ERR ' + (e && e.message ? e.message : String(e)); }
	})()`))
}
