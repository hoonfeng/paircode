//go:build ignore

package main

// Command interact_probe 验证设置面板中表单控件的交互：
//   1. checkbox 点击 → checked 属性 + Vue 状态
//   2. select 点击 → 是否有下拉响应（change 事件）
//   3. range 点击/键盘 → value 变化
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

	// 1. checkbox 交互：切到 Agent tab 找 checkbox
	fmt.Println("[tab] 切 Agent tab:")
	fmt.Println("  " + js(wv, `(function(){
		var btns = document.querySelectorAll('.settings-tabs button');
		if (btns.length > 1) { var ev = new Event('click', {bubbles:true}); btns[1].dispatchEvent(ev); }
		return 'clicked';
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	fmt.Println("[cb] Agent tab checkbox 状态:")
	fmt.Println("  " + js(wv, `(function(){
		var cbs = document.querySelectorAll('.settings-modal input[type="checkbox"]');
		var out = {count: cbs.length, items: []};
		for (var i=0;i<cbs.length && i<5;i++) {
			out.items.push({checked: cbs[i].checked, attr: cbs[i].hasAttribute('checked')});
		}
		return JSON.stringify(out);
	})()`))

	// 点击第一个 checkbox（破坏性操作需确认 → requireHumanApprovalForDestructive=true 初始应 checked）
	fmt.Println("[cb2] 点击第一个 checkbox:")
	fmt.Println("  " + js(wv, `(function(){
		var cb = document.querySelector('.settings-modal input[type="checkbox"]');
		if (!cb) return 'no checkbox';
		var before = cb.checked;
		var ev = new Event('click', {bubbles:true});
		cb.dispatchEvent(ev);
		return JSON.stringify({before: before, after: cb.checked});
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("  [after] " + js(wv, `(function(){
		var cb = document.querySelector('.settings-modal input[type="checkbox"]');
		return cb ? JSON.stringify({checked: cb.checked, attr: cb.hasAttribute('checked')}) : 'gone';
	})()`))

	// 2. select 点击测试（切回 AI tab）
	fmt.Println("[tab2] 切回 AI tab:")
	fmt.Println("  " + js(wv, `(function(){
		var btns = document.querySelectorAll('.settings-tabs button');
		if (btns.length > 0) { var ev = new Event('click', {bubbles:true}); btns[0].dispatchEvent(ev); }
		return 'clicked';
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("[sel] select 点击后是否有下拉容器:")
	fmt.Println("  " + js(wv, `(function(){
		var sel = document.querySelector('.settings-modal select');
		if (!sel) return 'no select';
		var ev = new Event('click', {bubbles:true});
		sel.dispatchEvent(ev);
		var out = {selValue: sel.value};
		// 检查是否有 popup/列表容器出现
		var popups = document.querySelectorAll('.settings-modal select option[selected]');
		out.selectedOpt = popups.length;
		// DOM 中 option 是否可点
		var opts = sel.querySelectorAll('option');
		out.optionCount = opts.length;
		return JSON.stringify(out);
	})()`))
}
