// Command modal_hscroll_probe 复现「拖动 range 滑块时设置窗口出现水平滚动条」：
//   1. 打开设置面板，检查 settings-body / modal-body / modal-content 的水平滚动
//   2. 模拟 Press+Move 拖动温度 range（含 value 越过 0.8 触发 ⚠️ hint）
//   3. 每步用 HorizontalScrollbarMetrics.MaxScroll 检测水平滚动条是否出现
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

// hscroll 检查一个 DOM 元素对应的 RenderBox 是否有水平滚动条
func hscroll(rv *rendering.RenderView, el *dom.Element, name string) {
	if el == nil {
		fmt.Printf("[hscroll] %-14s N/A\n", name)
		return
	}
	box := rv.FindRenderBoxForNode(el)
	if box == nil {
		fmt.Printf("[hscroll] %-14s no box\n", name)
		return
	}
	m := rendering.HorizontalScrollbarMetrics(rv, box)
	sx, _ := rv.BoxScrollOffset(box)
	flag := "none"
	if m.OK && m.MaxScroll > 0 {
		flag = "★ HSCROLL"
	}
	fmt.Printf("[hscroll] %-14s view=%.0f total=%.0f maxSx=%.0f sx=%.0f %s\n",
		name, box.Width(), m.TotalLen, m.MaxScroll, sx, flag)
}

// scanHScroll 遍历整个渲染树，找出所有存在水平滚动条的容器
// （任何层级的隐藏水平溢出都能被发现，不限于已知 class）。
func scanHScroll(rv *rendering.RenderView) int {
	count := 0
	root := rendering.RenderObject(rv)
	for obj := root; obj != nil; obj = obj.NextInPreOrder() {
		if obj == nil || !obj.IsBox() {
			continue
		}
		box, ok := obj.(*rendering.RenderBox)
		if !ok {
			continue
		}
		m := rendering.HorizontalScrollbarMetrics(rv, box)
		if m.OK && m.MaxScroll > 0 {
			count++
			cls, id, tag := "", "", ""
			if n := box.Node(); n != nil {
				if el, ok := n.(*dom.Element); ok {
					cls = el.ClassName()
					id = el.GetId()
					tag = el.LocalName()
				}
			}
			sx, _ := rv.BoxScrollOffset(box)
			fmt.Printf("  ★ HSCROLL tag=%-8s class=%-40q id=%q view=%.0f total=%.0f maxSx=%.0f sx=%.0f\n",
				tag, cls, id, box.Width(), m.TotalLen, m.MaxScroll, sx)
		}
	}
	if count == 0 {
		fmt.Println("  (全树无水平滚动条)")
	}
	return count
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
	layout.FontMetricsFunc = func(family string, size float64, weight int, style string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
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

	doc := wv.Document()
	rv := wv.RenderView()

	byCls := func(cls string) *dom.Element {
		for _, e := range doc.GetElementsByClassName(cls) {
			if e.ClassName() == cls {
				return e
			}
		}
		return nil
	}
	// 容器层级：settings-body → modal-body → modal-content
	sb := byCls("settings-body")
	mb := byCls("modal-body")
	mc := byCls("modal-content")
	selBody := byCls("modal-body")
	_ = selBody

	var rangeEl *dom.Element
	for _, el := range doc.GetElementsByTagName("input") {
		if el.GetAttribute("type") == "range" {
			rangeEl = el
			break
		}
	}
	if rangeEl == nil {
		fmt.Println("[range] NO input[type=range]")
		os.Exit(1)
	}
	rbox := rv.FindRenderBoxForNode(rangeEl)
	if rbox == nil {
		fmt.Println("[range] NO RenderBox")
		os.Exit(1)
	}
	left := rbox.AbsoluteX()
	_, so := rv.BoxScrollOffset(rbox)
	left -= so
	fmt.Printf("[range] box at (%.0f,%.0f) %.0fx%.0f value=%s\n",
		left, rbox.AbsoluteY(), rbox.Width(), rbox.Height(), rangeEl.GetAttribute("value"))

	// DOM 侧 scrollWidth/clientWidth 对比（模拟浏览器检测）
	domW := func() string {
		return js(wv, `(function(){
			var out = {};
			['.settings-body','.modal-body','.modal-content'].forEach(function(s){
				var el = document.querySelector(s);
				if (el) out[s] = {cw: el.clientWidth, sw: el.scrollWidth, diff: el.scrollWidth-el.clientWidth, sx: el.scrollLeft};
			});
			return JSON.stringify(out);
		})()`)
	}

	fmt.Println("[0] 初始（value=默认）")
	hscroll(rv, sb, "settings-body")
	hscroll(rv, mb, "modal-body")
	hscroll(rv, mc, "modal-content")
	fmt.Println("[0] DOM: " + domW())

	h := app.NewHostForTest(wv, 1280, 800)

	// 1. Press 75% → value=1.5（无 hint，1.5>0.8 有 hint）
	_ = h.MockRangePress(rangeEl, rv, left+rbox.Width()*0.75)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("[1] Press 75% → value=" + rangeEl.GetAttribute("value"))
	scanHScroll(rv)
	hscroll(rv, sb, "settings-body")
	hscroll(rv, sb, "settings-body")
	hscroll(rv, mb, "modal-body")
	hscroll(rv, mc, "modal-content")
	fmt.Println("[1] DOM: " + domW())

	// 2. Move 拖到 90% → value=1.8（>0.8 触发 ⚠️ hint 出现）
	_ = h.MockRangeMove(left + rbox.Width()*0.90)
	fmt.Println("[2] Move 90% → value=" + rangeEl.GetAttribute("value"))
	scanHScroll(rv)
	hscroll(rv, sb, "settings-body")
	runJobs(wv)
	fmt.Println("[2] Move 90% → value=" + rangeEl.GetAttribute("value"))
	hscroll(rv, sb, "settings-body")
	hscroll(rv, mb, "modal-body")
	hscroll(rv, mc, "modal-content")
	fmt.Println("[2] DOM: " + domW())

	fmt.Println("[3] Move 200% → value=" + rangeEl.GetAttribute("value"))
	scanHScroll(rv)
	hscroll(rv, sb, "settings-body")
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
	fmt.Println("[3] Move 200% → value=" + rangeEl.GetAttribute("value"))
	hscroll(rv, sb, "settings-body")
	hscroll(rv, mb, "modal-body")
	hscroll(rv, mc, "modal-content")
	fmt.Println("[3] DOM: " + domW())

	// 3.5 拖动中鼠标移出 range（hover 到设置面板其他区域）——真实拖动
	fmt.Println("[3.5] 拖动中 hover 移出 range（仍按住）")
	scanHScroll(rv)
	hscroll(rv, sb, "settings-body")
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
	fmt.Println("[3.5] 拖动中 hover 移出 range（仍按住）")
	hscroll(rv, sb, "settings-body")
	hscroll(rv, mb, "modal-body")
	hscroll(rv, mc, "modal-content")
	fmt.Println("[3.5] DOM: " + domW())

	fmt.Println("[4] Release → value=" + rangeEl.GetAttribute("value"))
	scanHScroll(rv)
	hscroll(rv, sb, "settings-body")
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
	fmt.Println("[4] Release → value=" + rangeEl.GetAttribute("value"))
	hscroll(rv, sb, "settings-body")
	hscroll(rv, mb, "modal-body")
	hscroll(rv, mc, "modal-content")
	fmt.Println("[4] DOM: " + domW())

	// 5. 关闭设置面板，打开工具配置弹窗，检查水平滚动（tcp-switch-list）
	js(wv, `(function(){
		var close = document.querySelector('.modal-close');
		if (close) { var ev = new Event('click', {bubbles:true}); close.dispatchEvent(ev); }
		var st = window.__state;
		if (st && !st.rightPanelVisible) { st.rightPanelVisible = true; }
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
	js(wv, `(function(){
		var btn = document.querySelector('.obtn-review-config');
		if (btn) { var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev); }
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
	fmt.Println("[5] 工具配置弹窗")
	for _, cls := range []string{"modal-content", "modal-body", "tool-list", "tcp-switch-list"} {
		hscroll(rv, byCls(cls), cls)
	}
	fmt.Println("[5] DOM: " + js(wv, `(function(){
		var out = {};
		['.modal-content','.modal-body','.tool-list','.tcp-switch-list','[class*=tool-list]'].forEach(function(s){
			var el = document.querySelector(s);
			if (el) out[s] = {cw: el.clientWidth, sw: el.scrollWidth, diff: el.scrollWidth-el.clientWidth};
		});
		return JSON.stringify(out);
	})()`))
	fmt.Println("DONE")
}
