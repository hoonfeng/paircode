package main

// Command local_read_probe 通过 Vue vnode 组件实例读取 SettingsModal 的 local 状态，
// 确认 loadSettings 是否执行、local 各字段值，并对比 DOM。
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

	// 通过 vnode 找 SettingsModal 组件实例
	fmt.Println("[inst] 查找 SettingsModal 组件实例:")
	fmt.Println("  " + js(wv, `(function(){
		try {
			// 方法1：从 .settings-modal 元素向上找 __vueParentComponent
			var el = document.querySelector('.settings-modal');
			var chain = [];
			var cur = el;
			for (var i = 0; i < 8 && cur; i++) {
				var keys = Object.keys(cur).filter(function(k){ return k.indexOf('__vue') === 0 || k.indexOf('_vnode') === 0 || k.indexOf('__v') === 0; });
				chain.push({tag: cur.tagName, keys: keys});
				cur = cur.parentElement;
			}
			return JSON.stringify(chain);
		} catch(e) { return 'ERR ' + (e && e.message ? e.message : String(e)); }
	})()`))

	// 尝试 Vue devtools 全局
	fmt.Println("[devtools] __VUE__ 全局:")
	fmt.Println("  " + js(wv, `(function(){
		var out = {};
		out.hasVue = typeof window.Vue !== 'undefined';
		out.hasDevtools = typeof window.__VUE_DEVTOOLS_GLOBAL_HOOK__ !== 'undefined';
		out.hasApp = typeof window.__vue_app__ !== 'undefined';
		return JSON.stringify(out);
	})()`))
}
