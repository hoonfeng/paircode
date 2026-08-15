//go:build ignore

package main

// Command comp_inst_probe 通过 _vnode 链查找 SettingsModal 组件实例，读取 setupState.local
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

	// 遍历 _vnode 链找 SettingsModal 的 componentInstance
	fmt.Println("[vnode] 查找组件实例:")
	fmt.Println("  " + js(wv, `(function(){
		var el = document.querySelector('.settings-modal');
		if (!el) return 'no modal';
		var out = {};
		var keys = Object.keys(el);
		out.elKeys = keys.filter(function(k){ return k.indexOf('__v') === 0 || k.indexOf('_vnode') === 0; });
		var vnode = el._vnode;
		out.hasVnode = !!vnode;
		if (vnode) {
			out.vnodeKeys = Object.keys(vnode).slice(0, 25);
			out.vnodeComp = vnode.component ? Object.keys(vnode.component).slice(0, 25) : null;
			if (vnode.component && vnode.component.setupState) {
				var ss = vnode.component.setupState;
				out.setupKeys = Object.keys(ss);
				if (ss.local) {
					out.local = {
						provider: ss.local.provider,
						baseURL: ss.local.baseURL,
						maxTokens: ss.local.maxTokens,
						temperature: ss.local.temperature
					};
				}
			}
		}
		return JSON.stringify(out);
	})()`))

	// 尝试从 document 挂载的 app 找（可能挂在 #app 元素）
	fmt.Println("[app2] #app 的 _vnode:")
	fmt.Println("  " + js(wv, `(function(){
		var app = document.querySelector('#app');
		if (!app) return 'no #app';
		var out = {keys: Object.keys(app).filter(function(k){ return k.indexOf('__v') === 0 || k.indexOf('_vnode') === 0; })};
		var vn = app._vnode;
		if (vn && vn.component && vn.component.setupState) {
			out.setupKeys = Object.keys(vn.component.setupState);
		}
		return JSON.stringify(out);
	})()`))
}
