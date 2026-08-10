// Command help_probe 在 wb-ui（jsc）引擎中加载真实 dist，诊断「标题栏帮助点击无反应」：
//   1. 标题栏/帮助按钮是否渲染（rect）
//   2. 初始状态真实坐标 hit-test（帮助按钮区域命中什么）
//   3. 引擎 MouseEvent 派发（handleClick 等价路径）→ dropdown 是否出现
//   4. JS new MouseEvent 派发（对照）→ dropdown 是否出现
//   5. 点击菜单项「功能介绍」→ .help-modal 是否渲染（marked 解析是否报错）
//   6. 输出 console 错误
// 用法：go run ./dev/desktop_probe/help_probe.go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
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

func wait(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(time.Duration(ms) * time.Millisecond)
	runJobs(wv)
}

func dropdownState(wv *webkit.WebView) string {
	return js(wv, `(function(){
		var dd = document.querySelector('.menu-dropdown');
		return JSON.stringify({dropdown: dd ? 'visible' : 'MISSING', items: document.querySelectorAll('.menu-dropdown .menu-item').length});
	})()`)
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
		wait(wv, 600)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// ── 1. 标题栏/帮助按钮是否渲染 ──
	fmt.Println("[1] 标题栏结构:")
	fmt.Println("  " + js(wv, `(function(){
		var out = {};
		var tb = document.querySelector('.titlebar');
		out.titlebar = tb ? rect(tb) : null;
		var mb = document.querySelector('.menubar');
		out.menubar = mb ? rect(mb) : null;
		var btns = document.querySelectorAll('.menu-btn');
		out.menuBtns = [];
		for (var i=0;i<btns.length;i++) {
			var b = btns[i];
			out.menuBtns.push({label: b.textContent.trim(), rect: rect(b), display: cs(b).display});
		}
		return JSON.stringify(out);
		function rect(el){ var r = el.getBoundingClientRect(); return {x:Math.round(r.left), y:Math.round(r.top), w:Math.round(r.width), h:Math.round(r.height)}; }
		function cs(el){ return getComputedStyle(el); }
	})()`))

	// ── 2. 初始状态真实坐标 hit-test ──
	fmt.Println("[2] 初始状态 hit-test（帮助按钮 rect 48,0,46,30 → 中心 71,15）:")
	rv := wv.RenderView()
	pts := [][2]float64{{71, 15}, {48, 5}, {94, 29}, {60, 15}, {130, 15}, {10, 15}}
	for _, pt := range pts {
		el := rendering.HitTest(rv, pt[0], pt[1], "")
		name := "<nil>"
		if el != nil {
			name = el.LocalName() + "." + el.GetAttribute("class")
		}
		fmt.Printf("  hit(%.0f,%.0f) → %s\n", pt[0], pt[1], name)
	}

	// ── 3. 引擎 MouseEvent 派发（handleClick 等价路径）──
	fmt.Println("[3] 引擎 MouseEvent 派发（handleClick 等价）:")
	el := rendering.HitTest(rv, 71, 15, "")
	if el != nil {
		el.DispatchEvent(dom.NewMouseEvent(dom.EventClick, true, true, false))
		wait(wv, 350)
		wv.RebuildRenderTree()
		wv.EnsureLayout()
		fmt.Println("  " + dropdownState(wv))
	} else {
		fmt.Println("  hit-test 未命中帮助按钮！")
	}

	// ── 4. JS new MouseEvent 派发（对照）──
	// 先关闭 dropdown（点 titlebar 触发 closeAllMenus）
	_ = js(wv, `(function(){ var tb = document.querySelector('.titlebar'); if (tb) tb.dispatchEvent(new MouseEvent('click', {bubbles:true})); })()`)
	wait(wv, 200)
	fmt.Println("[4] JS new MouseEvent 派发（对照）:")
	fmt.Println("  " + js(wv, `(function(){
		var btn = document.querySelector('.menu-btn');
		var out = {btnFound: !!btn};
		if (btn) btn.dispatchEvent(new MouseEvent('click', {bubbles:true, cancelable:true}));
		return JSON.stringify(out);
	})()`))
	wait(wv, 300)
	fmt.Println("  " + dropdownState(wv))

	// ── 5. 点击「功能介绍」菜单项 → HelpModal ──
	fmt.Println("[5] 点击功能介绍菜单项后 HelpModal:")
	fmt.Println("  " + js(wv, `(function(){
		var items = document.querySelectorAll('.menu-dropdown .menu-item');
		var out = {items: items.length};
		for (var i=0;i<items.length;i++) {
			var t = items[i].textContent.trim();
			if (t.indexOf('功能介绍') >= 0) {
				items[i].dispatchEvent(new MouseEvent('click', {bubbles:true, cancelable:true}));
				out.clicked = t;
			}
		}
		return JSON.stringify(out);
	})()`))
	wait(wv, 400)
	fmt.Println("  " + js(wv, `(function(){
		var out = {};
		var m = document.querySelector('.help-modal');
		out.helpModal = m ? {rect: rect(m), display: cs(m).display} : null;
		var ov = document.querySelector('.modal-overlay');
		out.overlay = ov ? {rect: rect(ov)} : null;
		var docs = document.querySelectorAll('.doc-nav-item');
		out.docNav = docs.length;
		var content = document.querySelector('.doc-content');
		out.contentHTML = content ? content.innerHTML.slice(0, 200) : null;
		out.helpModalCount = document.querySelectorAll('.help-modal').length;
		return JSON.stringify(out);
		function rect(el){ var r = el.getBoundingClientRect(); return {x:Math.round(r.left), y:Math.round(r.top), w:Math.round(r.width), h:Math.round(r.height)}; }
		function cs(el){ return getComputedStyle(el); }
	})()`))

	// ── 6. console 错误 ──
	fmt.Println("[6] console 错误:")
	cout := wv.ConsoleOutput()
	lines := strings.Split(cout, "\n")
	errLines := 0
	for _, ln := range lines {
		low := strings.ToLower(ln)
		if strings.Contains(low, "error") || strings.Contains(low, "cannot") || strings.Contains(low, "undefined") || strings.Contains(low, "typeerror") || strings.Contains(low, "is not a function") {
			fmt.Println("  [jserr]", ln)
			errLines++
			if errLines > 15 {
				break
			}
		}
	}
	fmt.Println("  [console] total lines:", len(lines))
}
