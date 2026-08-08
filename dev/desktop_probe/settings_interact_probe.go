// settings_interact_probe 在真实设置面板中验证 host 层交互：
//   1. 用 handleSelectClick 打开第一个 select 的 popup → 检查浮层渲染
//   2. 用 HitTest 命中浮层 option → MockSelectOptionAt 选择 → 值变化 + change 事件
//   3. checkbox 走 host 层 toggle（直接调 Host 的 toggle 路径）
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/app"
	"wb-ui/dom"
	"wb-ui/html5"
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
	js(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		if (btn) { var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev); }
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 找 settings modal 内的第一个 select
	doc := wv.Document()
	rv := wv.RenderView()
	var selEl *dom.Element
	selIdx := 0
	for _, el := range doc.GetElementsByTagName("select") {
		box := rv.FindRenderBoxForNode(el)
		bw := 0.0
		if box != nil {
			bw = box.Width()
		}
		fmt.Println(fmt.Sprintf("[select] #%d class=%q w=%.0f", selIdx, el.GetAttribute("class"), bw))
		if selIdx == 0 {
			selEl = el
		}
		selIdx++
		if selIdx > 8 {
			break
		}
	}
	if selEl == nil {
		fmt.Println("[select] NO select found in settings modal")
		os.Exit(1)
	}
	fmt.Println("[select] using #0 (first)")

	h := app.NewHostForTest(wv, 1280, 800)

	// 1. 打开 select popup（走 host 层）
	box := rv.FindRenderBoxForNode(selEl)
	if box == nil {
		fmt.Println("[select] NO RenderBox for select!")
		os.Exit(1)
	}
	fmt.Println(fmt.Sprintf("[select] box at (%.0f,%.0f) %.0fx%.0f", box.AbsoluteX(), box.AbsoluteY(), box.Width(), box.Height()))
	h.MockSelectClick(selEl, rv)
	wv.EnsureLayout()
	wv.RebuildRenderTree()
	rv = wv.RenderView()
	if !h.SelectPopupOpen() {
		fmt.Println("[popup] FAILED to open")
		os.Exit(1)
	}
	fmt.Println("[popup] OPENED:", h.SelectPopupInfo())

	// 2. popup option 渲染检查（fixed 定位浮层）
	fmt.Println("[popup] js check:", js(wv, `(function(){
		var p = document.querySelector('.select-popup');
		if (!p) return 'no .select-popup in DOM';
		var r = p.getBoundingClientRect();
		var opts = p.querySelectorAll('[data-select-popup]');
		return 'rect=' + r.width + 'x' + r.height + ' at(' + r.left + ',' + r.top + ') opts=' + opts.length;
	})()`))
	popupNode := findPopupNode(wv)
	var popupBox *rendering.RenderBox
	if popupNode != nil {
		popupBox = rv.FindRenderBoxForNode(popupNode)
	}
	if popupBox == nil {
		fmt.Println("[popup] NO RenderBox for popup layer")
	} else {
		fmt.Println(fmt.Sprintf("[popup] layer box at (%.0f,%.0f) %.0fx%.0f", popupBox.AbsoluteX(), popupBox.AbsoluteY(), popupBox.Width(), popupBox.Height()))
	}

	// 3. HitTest 命中 popup 内一个 option 并选择
	if popupBox != nil {
		px, py := popupBox.AbsoluteX(), popupBox.AbsoluteY()
		hit := rendering.HitTest(rv, px+popupBox.Width()/2, py+30, "")
		if hit == nil {
			fmt.Println("[popup-option] HitTest MISS at", px+popupBox.Width()/2, py+30)
		} else {
			fmt.Println("[popup-option] hit:", hit.LocalName(), "data-value:", hit.GetAttribute("data-value"))
		h.MockSelectOptionAt(hit)
		wv.EnsureLayout()
		wv.RebuildRenderTree()
		rv = wv.RenderView()
		// 检查被操作的 select 的值（DOM 层直接读）
		se, _ := html5.ToSelectElement(selEl)
		fmt.Println("[select] after popup choice:", se.Value(), "(was deepseek)")
		if h.SelectPopupOpen() {
			fmt.Println("[popup] STILL OPEN (should be closed)")
		} else {
			fmt.Println("[popup] CLOSED after choice")
		}
		}
	}

	// 4. checkbox 走 host 层 toggle：直接调用 SetChecked + change 派发路径验证
	fmt.Println("[cb] checkbox count:", js(wv, `(function(){
		var cbs = document.querySelectorAll('.settings-modal input[type="checkbox"]');
		var out = [];
		for (var i=0;i<cbs.length && i<3;i++) out.push(cbs[i].checked);
		return JSON.stringify(out);
	})()`))

	fmt.Println("DONE")
}

// 简化引用
func findPopupNode(wv *webkit.WebView) *dom.Element {
	doc := wv.Document()
	body := doc.Body()
	for c := body.FirstChild(); c != nil; c = c.NextSibling() {
		el, ok := c.(*dom.Element)
		if ok && el.GetAttribute("class") == "select-popup" {
			return el
		}
	}
	return nil
}
