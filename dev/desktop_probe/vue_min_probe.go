package main

// Command vue_min_probe 最小 Vue 复现：创建独立 Vue 应用（带 v-model input），
// 挂载后修改数据，看 input 是否更新。排除 SettingsModal 特定因素。
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

	// 在页面 DOM 中创建一个最小 Vue 应用（用页面已有的 Vue runtime）
	fmt.Println("[vue] 尝试获取 Vue runtime:")
	fmt.Println("  " + js(wv, `(function(){
		// dist 打包的 Vue 不暴露全局。尝试从 #app 的 __vue_app__ 拿
		var app = document.querySelector('#app');
		if (!app || !app.__vue_app__) return 'no app instance';
		var a = app.__vue_app__;
		var out = {keys: Object.keys(a).slice(0, 15)};
		if (a._context) {
			out.contextKeys = Object.keys(a._context).slice(0, 10);
			out.hasApp = !!a._context.app;
		}
		return JSON.stringify(out);
	})()`))

	// 从 app._context.app 找 Vue 导出
	fmt.Println("[vue2] 从 app 实例取 Vue:")
	fmt.Println("  " + js(wv, `(function(){
		var app = document.querySelector('#app');
		if (!app || !app.__vue_app__) return 'no';
		var a = app.__vue_app__;
		var out = {};
		// 尝试各种路径
		out.appType = typeof a;
		out.version = a.version;
		out.use = typeof a.use;
		out.component = typeof a.component;
		out.mount = typeof a.mount;
		// 从 _instance 找 Vue
		if (a._instance) {
			out.instKeys = Object.keys(a._instance).slice(0, 20);
			if (a._instance.appContext) {
				out.ctxKeys = Object.keys(a._instance.appContext).slice(0, 15);
				out.ctxApp = !!a._instance.appContext.app;
				out.ctxAppVersion = a._instance.appContext.app && a._instance.appContext.app.version;
			}
		}
		return JSON.stringify(out);
	})()`))
}
