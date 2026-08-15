//go:build ignore

package main

// Command settings_mount_probe 验证 SettingsModal onMounted 是否执行：
//   1. 切到「指令」tab 看 wsRoot（onMounted 赋值 state.workspaceRoot）
//   2. 直接执行 loadSettings 逻辑（模拟）看 Vue 是否刷新 DOM
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
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 切到「指令」tab（index 5）
	fmt.Println("[tab] 切换到指令 tab:")
	fmt.Println("  " + js(wv, `(function(){
		var btns = document.querySelectorAll('.settings-tabs button');
		var out = {found: btns.length};
		if (btns.length > 5) { var ev = new Event('click', {bubbles:true}); btns[5].dispatchEvent(ev); }
		return JSON.stringify(out);
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	fmt.Println("[inst] 指令 tab 内容（wsRoot 来自 onMounted）:")
	fmt.Println("  " + js(wv, `(function(){
		var hint = document.querySelector('.project-inst-hint');
		var out = {hintFound: !!hint};
		if (hint) out.hintText = hint.textContent.trim().slice(0, 100);
		var textareas = document.querySelectorAll('.inst-textarea');
		out.textareaCount = textareas.length;
		out.firstTA = textareas.length > 0 ? textareas[0].value.slice(0, 30) : null;
		return JSON.stringify(out);
	})()`))

	// state.workspaceRoot
	fmt.Println("[state] workspaceRoot = " + js(wv, `window.__state ? window.__state.workspaceRoot : 'no state'`))

	// 直接修改一个 input 的 value 属性测试 Vue 渲染（模拟 v-model 更新）
	fmt.Println("[vm] 测试 input.value 直接赋值是否被渲染引擎读取:")
	fmt.Println("  " + js(wv, `(function(){
		var inp = document.querySelector('.settings-modal input[type="number"]');
		if (!inp) return 'no number input';
		inp.value = '99999';
		var ev = new Event('input', {bubbles: true});
		inp.dispatchEvent(ev);
		return 'set 99999, now=' + inp.value;
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("  after: " + js(wv, `(function(){
		var inp = document.querySelector('.settings-modal input[type="number"]');
		return inp ? inp.value : 'gone';
	})()`))
}
