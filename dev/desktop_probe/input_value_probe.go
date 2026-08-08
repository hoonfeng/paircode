package main

// Command input_value_probe 精确验证：
//   1. 打开面板后 select option selected 状态（确认 loadSettings 是否生效）
//   2. 手动 setAttribute value 后 GetAttribute 是否返回
//   3. el.value = x 赋值（Vue v-model 路径）是否更新 attribute
//   4. option 的 v-for 是否渲染了 deepseek 项
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

	// 1. select option 状态
	fmt.Println("[sel] 服务商 select options:")
	fmt.Println("  " + js(wv, `(function(){
		var sel = document.querySelector('.settings-modal select');
		if (!sel) return 'no select';
		var opts = [];
		for (var c = sel.firstElementChild; c; c = c.nextElementSibling) {
			opts.push({val: c.getAttribute('value'), text: c.textContent.trim().slice(0, 10),
				selected: c.hasAttribute('selected')});
		}
		return JSON.stringify({opts: opts.slice(0, 5), selValue: sel.value});
	})()`))

	// 2. text input value setter 测试
	fmt.Println("[inp] input value setter 测试:")
	fmt.Println("  " + js(wv, `(function(){
		var inp = document.querySelector('.settings-modal input[type="text"]');
		if (!inp) return 'no input';
		var out = {};
		out.before = inp.value;
		out.attrBefore = inp.getAttribute('value');
		inp.setAttribute('value', 'https://test.abc/v1');
		out.attrAfter = inp.getAttribute('value');
		out.valueAfter = inp.value;
		// 恢复
		inp.removeAttribute('value');
		return JSON.stringify(out);
	})()`))

	// 3. el.value = x（Vue v-model 路径）
	fmt.Println("[vm] el.value = x 赋值路径:")
	fmt.Println("  " + js(wv, `(function(){
		var inp = document.querySelector('.settings-modal input[type="text"]');
		if (!inp) return 'no input';
		inp.value = 'hello-world-value';
		return JSON.stringify({value: inp.value, attr: inp.getAttribute('value')});
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 200); })`)
	time.Sleep(300 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 4. 渲染后是否显示
	fmt.Println("[paint] 渲染后 input 值:")
	fmt.Println("  " + js(wv, `(function(){
		var inp = document.querySelector('.settings-modal input[type="text"]');
		return inp ? JSON.stringify({value: inp.value, attr: inp.getAttribute('value')}) : 'gone';
	})()`))

	// 5. 手动执行 loadSettings 逻辑（绕过 Vue，直接验证 DOM 属性）
	fmt.Println("[direct] 直接改 state 触发 loadSettings 的等价值:")
	fmt.Println("  " + js(wv, `(function(){
		var s = window.__state;
		var out = {settingsLoaded: s.settingsLoaded};
		return JSON.stringify(out);
	})()`))
}
