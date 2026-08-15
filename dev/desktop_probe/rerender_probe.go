//go:build ignore

package main

// Command rerender_probe 手动触发组件重渲染，验证：
//   1. 改 activeTab（ref）→ 组件 re-render → input 的 vModelText 是否更新？
//   2. 直接改 local（通过 select value 触发）？
//   3. 关键：input._assign（Symbol）是否存在
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

	// 1. 检查 input 的 Symbol 属性（_assign 可能是 Symbol）
	fmt.Println("[sym] input Symbol 属性:")
	fmt.Println("  " + js(wv, `(function(){
		var inp = document.querySelector('.settings-modal input[type="text"]');
		if (!inp) return 'no input';
		var syms = Object.getOwnPropertySymbols(inp);
		var out = {symbols: syms.length};
		out.details = syms.map(function(s) {
			return {desc: String(s), val: String(inp[s]).slice(0, 30)};
		});
		return JSON.stringify(out);
	})()`))

	// 2. 切 tab 触发组件重渲染
	fmt.Println("[tab] 切到 Agent tab 触发重渲染:")
	fmt.Println("  " + js(wv, `(function(){
		var btns = document.querySelectorAll('.settings-tabs button');
		if (btns.length > 1) { var ev = new Event('click', {bubbles:true}); btns[1].dispatchEvent(ev); }
		return 'clicked tab 1';
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 切回 AI tab，检查 input 是否更新
	fmt.Println("  " + js(wv, `(function(){
		var btns = document.querySelectorAll('.settings-tabs button');
		if (btns.length > 0) { var ev = new Event('click', {bubbles:true}); btns[0].dispatchEvent(ev); }
		return 'back to AI';
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("  [ai] " + js(wv, `(function(){
		var inp = document.querySelector('.settings-modal input[type="text"]');
		return inp ? JSON.stringify({value: inp.value, attr: inp.getAttribute('value')}) : 'gone';
	})()`))
}
