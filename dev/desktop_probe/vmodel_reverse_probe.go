//go:build ignore

package main

// Command vmodel_reverse_probe 反向验证 input 的 v-model 绑定是否活着：
//   修改 input.value + dispatch input 事件 → Vue 应把值写回 local（进而 state）
//   如果 state.settings 没有变化 → input 不是 Vue 管理的元素！
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

	// 1. 反向绑定测试：改 input.value + input 事件 → 检查 state.settings 是否变化
	fmt.Println("[rev] 反向绑定测试（改 input → Vue 写回 state?）:")
	fmt.Println("  " + js(wv, `(function(){
		var inp = document.querySelector('.settings-modal input[type="text"]');
		if (!inp) return 'no input';
		inp.value = 'REVERSE-TEST-VALUE';
		var ev = new Event('input', {bubbles: true});
		inp.dispatchEvent(ev);
		return 'dispatched';
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	fmt.Println("  state.settings.baseURL = " + js(wv, `window.__state.settings.baseURL`))

	// 2. 检查 input 是否有 Vue 挂的 _assigning / vnode 属性
	fmt.Println("[vm2] input 元素 Vue 属性:")
	fmt.Println("  " + js(wv, `(function(){
		var inp = document.querySelector('.settings-modal input[type="text"]');
		var out = {keys: []};
		for (var k in inp) {
			if (k.indexOf('__v') === 0 || k.indexOf('_v') === 0 || k.indexOf('__') === 0 || k === '_assigning' || k === 'composing') {
				out.keys.push(k);
			}
		}
		return JSON.stringify(out);
	})()`))

	// 3. 检查 select 的 v-model 反向（select.value + change → local.provider?）
	fmt.Println("[rev-sel] select 反向绑定:")
	fmt.Println("  " + js(wv, `(function(){
		var sel = document.querySelector('.settings-modal select');
		if (!sel) return 'no select';
		// 改为 anthropic
		sel.value = 'anthropic';
		var ev = new Event('change', {bubbles: true});
		sel.dispatchEvent(ev);
		return 'dispatched, select.value=' + sel.value;
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	fmt.Println("  state.settings.provider = " + js(wv, `window.__state.settings.provider`))
}
