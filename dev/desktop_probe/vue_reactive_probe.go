//go:build ignore

package main

// Command vue_reactive_probe 验证 wb-ui 引擎中 Vue 3 reactive 依赖追踪：
//   模拟 Vue 的 reactive + effect（依赖收集 + set 触发），确认 set 后 effect 是否重跑。
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

	// 模拟 Vue 3 reactive：依赖收集 + set 触发
	fmt.Println("[vue] 模拟 Vue reactive + effect 依赖追踪:")
	fmt.Println("  " + js(wv, `(function(){
		try {
			// 最小 Vue 模拟
			var targetMap = new Map(); // target -> Map(key -> Set(effects))
			var activeEffect = null;
			function track(target, key) {
				if (!activeEffect) return;
				var depsMap = targetMap.get(target);
				if (!depsMap) { depsMap = new Map(); targetMap.set(target, depsMap); }
				var dep = depsMap.get(key);
				if (!dep) { dep = new Set(); depsMap.set(key, dep); }
				dep.add(activeEffect);
			}
			function trigger(target, key) {
				var depsMap = targetMap.get(target);
				if (!depsMap) return;
				var dep = depsMap.get(key);
				if (!dep) return;
				dep.forEach(function(eff) { eff(); });
			}
			function reactive(obj) {
				return new Proxy(obj, {
					get: function(t, k, r) { track(t, k); return Reflect.get(t, k, r); },
					set: function(t, k, v, r) { var res = Reflect.set(t, k, v, r); trigger(t, k); return res; }
				});
			}
			function effect(fn) {
				activeEffect = fn;
				fn();
				activeEffect = null;
				return fn;
			}

			var runs = [];
			var state = reactive({ baseURL: '', provider: '' });
			effect(function() { runs.push('effect: provider=' + state.provider + ' baseURL=' + state.baseURL); });
			state.provider = 'deepseek';
			state.baseURL = 'https://api.deepseek.com/v1';
			return JSON.stringify({runs: runs});
		} catch(e) { return 'ERR ' + (e && e.message ? e.message : String(e)); }
	})()`))

	// 用真实 Vue（dist 里的 vue.global）测试
	fmt.Println("[vue-real] 真实 Vue reactive 测试:")
	_, _ = wv.JSInterpreter().RunJS(`(function(){
		window.__vueReactiveTest = 'pending';
		try {
			// dist 里 Vue 可能挂在 window.Vue 或通过模块导入，尝试直接构造
			var V = window.Vue;
			if (!V) { window.__vueReactiveTest = 'no window.Vue'; return; }
			var state = V.reactive({ baseURL: '', provider: '' });
			var runs = [];
			V.effect(function() { runs.push('effect: ' + state.provider + '/' + state.baseURL); });
			state.provider = 'deepseek';
			state.baseURL = 'https://x.com';
			window.__vueReactiveTest = JSON.stringify({runs: runs, hasVue: true});
		} catch(e) { window.__vueReactiveTest = 'ERR ' + (e && e.message ? e.message : String(e)); }
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 200); })`)
	time.Sleep(300 * time.Millisecond)
	runJobs(wv)
	fmt.Println("  " + js(wv, `window.__vueReactiveTest`))
}
