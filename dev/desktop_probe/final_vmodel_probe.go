//go:build ignore

package main

// Command final_vmodel_probe 终极验证：
//   1. 页面上是否有多个 settings-modal（重复渲染？）
//   2. input 手动赋值后是否保持（确认 setter 与渲染引擎联动）
//   3. 直接检查 Vue 是否真的更新了 select（读 option selected + 手动清除后重渲染）
//   4. 打开面板后手动执行 loadSettings 等价操作，对比前后 DOM
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

	// 1. 多个 modal？
	fmt.Println("[multi] settings-modal 数量: " + js(wv, `document.querySelectorAll('.settings-modal').length`))
	fmt.Println("[multi] modal-overlay 数量: " + js(wv, `document.querySelectorAll('.modal-overlay').length`))

	// 2. select 与 input 对比
	fmt.Println("[cmp] select/input 状态:")
	fmt.Println("  " + js(wv, `(function(){
		var sel = document.querySelector('.settings-modal select');
		var inp = document.querySelector('.settings-modal input[type="text"]');
		var out = {};
		out.selectValue = sel ? sel.value : null;
		out.inputValue = inp ? inp.value : null;
		// select options
		out.options = [];
		if (sel) {
			for (var c = sel.firstChild; c; c = c.nextSibling) {
				if (c.nodeType === 1 && c.nodeName === 'OPTION') {
					out.options.push({v: c.getAttribute('value'), s: c.hasAttribute('selected')});
				}
			}
		}
		return JSON.stringify(out);
	})()`))

	// 3. 手动设置 input.value 保持测试（含 RebuildRenderTree）
	fmt.Println("[set] 手动 input.value = state.settings.baseURL:")
	fmt.Println("  " + js(wv, `(function(){
		var inp = document.querySelector('.settings-modal input[type="text"]');
		if (!inp) return 'no input';
		inp.value = window.__state.settings.baseURL || '';
		return 'set: ' + inp.value;
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 200); })`)
	time.Sleep(300 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("  after: " + js(wv, `(function(){
		var inp = document.querySelector('.settings-modal input[type="text"]');
		return inp ? JSON.stringify({value: inp.value, attr: inp.getAttribute('value')}) : 'gone';
	})()`))

	// 4. 手动清除 select 的 selected 后重渲染看 fallback
	fmt.Println("[sel-clear] 清除全部 selected 后渲染:")
	fmt.Println("  " + js(wv, `(function(){
		var sel = document.querySelector('.settings-modal select');
		if (!sel) return 'no select';
		for (var c = sel.firstChild; c; c = c.nextSibling) {
			if (c.nodeType === 1 && c.nodeName === 'OPTION') c.removeAttribute('selected');
		}
		return 'cleared, value=' + sel.value;
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 200); })`)
	time.Sleep(300 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("  after: " + js(wv, `(function(){
		var sel = document.querySelector('.settings-modal select');
		return sel ? JSON.stringify({value: sel.value, opts: (function(){
			var r = [];
			for (var c = sel.firstChild; c; c = c.nextSibling) {
				if (c.nodeType === 1 && c.nodeName === 'OPTION') r.push({v: c.getAttribute('value'), s: c.hasAttribute('selected')});
			}
			return r;
		})()}) : 'gone';
	})()`))
}
