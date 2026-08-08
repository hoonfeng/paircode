package main

// Command settings_vmodel_probe 验证 SettingsModal 中 v-model 对 input.value 的更新：
//   1. 打开面板后 local 值（通过 Vue 组件实例不可得，改为读 DOM value）
//   2. 对比 select（有值）与 input（空）的差异
//   3. 检查 wb-ui 对 input value 属性的渲染更新
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
	fmt.Println("[open] 打开设置面板:")
	fmt.Println("  " + js(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		var out = {btnFound: !!btn};
		if (btn) { var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev); }
		return JSON.stringify(out);
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 等待更多时间让 Vue 更新
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
	time.Sleep(600 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 读取所有 input 的 value 属性 + 属性值
	fmt.Println("[inputs] 面板所有 input/select 值:")
	fmt.Println("  " + js(wv, `(function(){
		var out = [];
		var els = document.querySelectorAll('.settings-modal input, .settings-modal select');
		for (var i=0;i<els.length && i<15;i++) {
			var el = els[i];
			out.push({
				tag: el.tagName,
				type: el.getAttribute('type'),
				val: el.value,
				attrVal: el.getAttribute('value'),
				placeholder: el.getAttribute('placeholder') ? el.getAttribute('placeholder').slice(0,20) : null
			});
		}
		return JSON.stringify(out);
	})()`))

	// state.settings 关键字段
	fmt.Println("[state] state.settings:")
	fmt.Println("  " + js(wv, `(function(){
		var s = window.__state.settings;
		return JSON.stringify({provider: s.provider, baseURL: s.baseURL, apiKey: s.apiKey ? '***' : s.apiKey,
			executeModel: s.executeModel, temperature: s.temperature, maxTokens: s.maxTokens});
	})()`))

	// 手动设置 input.value 测试 v-model 反向（input 事件）
	fmt.Println("[vm] 手动修改 input.value + dispatch input 事件:")
	fmt.Println("  " + js(wv, `(function(){
		var inp = document.querySelector('.settings-modal input[type="text"]');
		if (!inp) return 'no input';
		var before = inp.value;
		inp.value = 'http://test.example.com/v1';
		var ev = new Event('input', {bubbles: true});
		inp.dispatchEvent(ev);
		return JSON.stringify({before: before, after: inp.value});
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 200); })`)
	time.Sleep(300 * time.Millisecond)
	runJobs(wv)
	fmt.Println("  " + js(wv, `(function(){
		var inp = document.querySelector('.settings-modal input[type="text"]');
		return 'value after input event: ' + (inp ? inp.value : 'gone');
	})()`))
}
