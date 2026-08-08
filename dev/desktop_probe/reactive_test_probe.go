package main

// Command reactive_test_probe 直接在 wb-ui 引擎中测试 Vue 3 reactive 的依赖追踪：
//   1. 用真实 Vue（从页面 window 取）创建 reactive + effect
//   2. 修改属性 → effect 是否重跑
//   3. 用页面的 state（reactive）修改 settingsLoaded → 是否有响应
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

	// 1. 从 window 找 Vue（可能挂在 window.Vue）
	fmt.Println("[vue] window.Vue = " + js(wv, `typeof window.Vue`))

	// 2. 直接测页面的 __state.settings 是否 reactive（修改后读）
	fmt.Println("[state] state.settings 修改测试:")
	fmt.Println("  " + js(wv, `(function(){
		var s = window.__state;
		var before = s.settings.temperature;
		s.settings.temperature = 'CHANGED-' + Date.now();
		var after = s.settings.temperature;
		// 恢复
		s.settings.temperature = before;
		return JSON.stringify({before: before, after: after, reactive: after !== before});
	})()`))

	// 3. 手动注入 Vue 的 reactive 测试（如果 dist 的 Vue 没挂 window，尝试 import）
	// 直接测试页面内是否有全局 Vue 模块
	fmt.Println("[modules] 查找 Vue 运行时:")
	fmt.Println("  " + js(wv, `(function(){
		var out = {};
		out.windowVue = typeof window.Vue;
		// vite 打包后 Vue 不挂 window，但可以通过 __vue_app__ 拿
		var appEl = document.querySelector('#app');
		out.appVue = appEl && appEl.__vue_app__ ? 'yes' : 'no';
		return JSON.stringify(out);
	})()`))
}
