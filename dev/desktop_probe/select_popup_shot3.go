// Command select_popup_shot3 在真实设置面板中验证 select popup 打开位置：
// 点击服务商 select（#1）后，popup 应从其底部（y=236+27=263）下方展开，
// 不覆盖组件本身。
//go:build ignore

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
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

	// 打开设置面板
	js(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		if (!btn) return 'no btn';
		var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev);
		return 'clicked';
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 定位服务商 select（第 2 个 select，i=1）
	doc := wv.Document()
	sels := doc.GetElementsByTagName("select")
	if len(sels) < 2 {
		fmt.Println("NO selects in panel")
		os.Exit(1)
	}
	sel := sels[1] // 服务商
	rv := wv.RenderView()
	h := app.NewHostForTest(wv, 1280, 800)

	// 打开 popup
	h.MockSelectClick(sel, rv)
	wv.EnsureLayout()
	wv.RebuildRenderTree()
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	// MockSelectClick 内部 RebuildRenderTree 会创建新树，重新获取 rv
	rv = wv.RenderView()

	if h.SelectPopupOpen() {
		fmt.Println("[popup] OPENED")
	} else {
		fmt.Println("[popup] NOT OPENED")
		os.Exit(1)
	}

	// popup 几何 vs select 几何
	fmt.Println("[popup-info] " + js(wv, `(function(){
		var p = document.querySelector('.select-popup');
		if (!p) return 'no popup';
		var pr = p.getBoundingClientRect();
		var sels = document.querySelectorAll('select');
		var s = sels[1]; // 服务商
		var sr = s.getBoundingClientRect();
		return 'select at(' + sr.x + ',' + sr.y + ') w=' + sr.width + ' h=' + sr.height +
			' popup at(' + pr.x + ',' + pr.y + ') w=' + pr.width + ' h=' + pr.height +
			' options=' + p.children.length +
			' gap=' + Math.round(pr.y - (sr.y + sr.height));
	})()`))
	// popup 内每个 option 的文本/value/类/背景
	fmt.Println("[popup-options] " + js(wv, `(function(){
		var p = document.querySelector('.select-popup');
		if (!p) return 'no popup';
		var out = [];
		for (var i = 0; i < p.children.length; i++) {
			var c = p.children[i];
			out.push(i + ':' + c.textContent + '|val=' + c.getAttribute('data-value') +
				'|cls=' + c.className + '|bg=' + getComputedStyle(c).backgroundColor);
		}
		return out.join(' ; ');
	})()`))
	// option div 的 computed 背景来源
	fmt.Println("[opt-cs] " + js(wv, `(function(){
		var p = document.querySelector('.select-popup');
		var c = p.children[1]; // 非选中 option
		var cs = getComputedStyle(c);
		return 'bg=' + cs.backgroundColor +
			' bg-prop=' + cs.getPropertyValue('background-color') +
			' bg-shorthand=' + cs.getPropertyValue('background') +
			' color=' + cs.color +
			' inherit?=' + (c.parentElement ? getComputedStyle(c.parentElement).backgroundColor : 'none');
	})()`))
	// select 当前 value 与 selectedIndex
	fmt.Println("[sel-state] " + js(wv, `(function(){
		var s = document.querySelectorAll('select')[1];
		return 'value=' + s.value + ' idx=' + s.selectedIndex +
			' selOpts=' + (s.selectedOptions ? s.selectedOptions.length : 'undef') +
			' selText=' + (s.selectedOptions && s.selectedOptions[0] ? s.selectedOptions[0].textContent : 'none');
	})()`))
	// popup 的 computed border/background
	fmt.Println("[popup-cs] " + js(wv, `(function(){
		var p = document.querySelector('.select-popup');
		if (!p) return 'no popup';
		var cs = getComputedStyle(p);
		return 'border=' + cs.borderTopWidth + ' ' + cs.borderTopColor +
			' radius=' + cs.borderRadius +
			' bg=' + cs.backgroundColor +
			' shadow=' + cs.boxShadow +
			' color=' + cs.color +
			' position=' + cs.position +
			' overflow=' + cs.overflowY;
	})()`))
	// 渲染盒级诊断：popup 的 box border/背景样式
	if box := rv.FindRenderBoxForNode(sel); box != nil {
		fmt.Println("[sel-box] x=", box.AbsoluteX(), "y=", box.AbsoluteY(), "w=", box.Width(), "h=", box.Height())
		st := box.Style()
		fmt.Printf("[sel-style] border=(%.0f,%.0f,%.0f,%.0f) color=%s bg=%s\n",
			st.BorderTopWidth.Value, st.BorderRightWidth.Value, st.BorderBottomWidth.Value, st.BorderLeftWidth.Value,
			st.Color.String(), st.BackgroundColor.String())
	}
	// 给 popup 加 id 以便 GetElementById 获取（与渲染树节点引用一致）
	js(wv, `var p = document.querySelector('.select-popup'); if (p) p.id = 'selpopup';`)
	popupEl := wv.Document().GetElementById("selpopup")
	if pbox := findBoxByClass(rv, "select-popup"); pbox != nil {
		fmt.Println("[popup-box] x=", pbox.AbsoluteX(), "y=", pbox.AbsoluteY(), "w=", pbox.Width(), "h=", pbox.Height())
		st := pbox.Style()
		fmt.Printf("[popup-style] border=(%.0f,%.0f,%.0f,%.0f) borderColorTop=%s bg=%s radius=%s color=%s\n",
			st.BorderTopWidth.Value, st.BorderRightWidth.Value, st.BorderBottomWidth.Value, st.BorderLeftWidth.Value,
			st.BorderTopColor.String(), st.BackgroundColor.String(), st.BorderRadius.String(), st.Color.String())
	} else {
		fmt.Println("[popup-box] NOT FOUND in render tree (tree walk)")
	}
	_ = popupEl

	// ── hover 验证：模拟鼠标移到 option 2（index 1，非选中）──
	// 浏览器标准：option:hover 应有高亮背景（--bg-hover）。走真实
	// 交互路径 MockMouseMove（HitTest → SetHovered → hoverStyleFastPath，
	// 与 EventCursorMove 一致，不重建渲染树）。
	hoverPX, hoverPY := 640.0, 300.0 // popup 内 option 2 行中心（popup at 525,264 行高24）
	hoverEl := h.MockMouseMove(wv, hoverPX, hoverPY)
	if hoverEl != nil {
		fmt.Printf("[hover] mock-move → (%.0f,%.0f) hit=%s.%s\n", hoverPX, hoverPY, hoverEl.LocalName(), hoverEl.ClassName())
	} else {
		fmt.Printf("[hover] mock-move → (%.0f,%.0f) hit=<nil>\n", hoverPX, hoverPY)
	}
	pngBytes, err := wv.Render()
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	off := (int(hoverPY)*wv.Width() + int(hoverPX)) * 4
	if off+3 < len(pngBytes) {
		fmt.Printf("[hover-px] (%d,%d) rgba=(%d,%d,%d,%d) 期望 #1c2333(--bg-hover)\n",
			int(hoverPX), int(hoverPY), pngBytes[off], pngBytes[off+1], pngBytes[off+2], pngBytes[off+3])
	}
	// 同时扫 hover 行左侧（无文字处）与相邻非 hover 行对比
	for _, p := range [][2]int{{int(hoverPX) + 40, int(hoverPY)}, {int(hoverPX) + 40, int(hoverPY) + 24}} {
		o := (p[1]*wv.Width() + p[0]) * 4
		if o+3 < len(pngBytes) {
			fmt.Printf("[hover-px] (%d,%d) rgba=(%d,%d,%d,%d)\n", p[0], p[1], pngBytes[o], pngBytes[o+1], pngBytes[o+2], pngBytes[o+3])
		}
	}
	// 移出 hover → 背景应还原为 popup 深色
	h.MockMouseMove(wv, 100, 100)
	pngBytes, err = wv.Render()
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	o := (int(hoverPY)*wv.Width() + int(hoverPX)) * 4
	if o+3 < len(pngBytes) {
		fmt.Printf("[hover-out] (%d,%d) rgba=(%d,%d,%d,%d) 期望还原 #161b22(--bg-secondary)\n",
			int(hoverPX), int(hoverPY), pngBytes[o], pngBytes[o+1], pngBytes[o+2], pngBytes[o+3])
	}

	// 渲染截图 + 像素扫描：确认 popup（主题深色背景）在 modal 之上可见
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	pngBytes, err = wv.Render()
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	fmt.Printf("[render] bytes=%d nonzero=%d\n", len(pngBytes), countNonzero(pngBytes))
	// modal-content 位置从 popup-info 的 select 推算：select 在 modal 内
	// popup 区域（option 行之间）应为白色；modal 遮罩区域应为深色
	for _, p := range [][2]int{
		{10, 10},    // modal-overlay 遮罩（页面边缘）→ 深色半透明
		{540, 300},  // popup 内 option 行（非选中）→ 白色 #fff
		{540, 340},  // popup 内 option 行 → 白色
		{300, 100},  // modal 外/页面 → 深色
	} {
		off := (p[1]*wv.Width() + p[0]) * 4
		if off+3 < len(pngBytes) {
			fmt.Printf("[px] (%d,%d) rgba=(%d,%d,%d,%d)\n", p[0], p[1], pngBytes[off], pngBytes[off+1], pngBytes[off+2], pngBytes[off+3])
		}
	}
	// 保存截图供 PIL 分析
	f, _ := os.Create(filepath.Join(wd, "dev", "desktop_probe", "select_popup_shot3.png"))
	if f != nil {
		img := image.NewRGBA(image.Rect(0, 0, wv.Width(), wv.Height()))
		for y := 0; y < wv.Height(); y++ {
			for x := 0; x < wv.Width(); x++ {
				off := (y*wv.Width() + x) * 4
				if off+3 < len(pngBytes) {
					img.SetRGBA(x, y, color.RGBA{R: pngBytes[off], G: pngBytes[off+1], B: pngBytes[off+2], A: pngBytes[off+3]})
				}
			}
		}
		_ = png.Encode(f, img)
		f.Close()
		fmt.Println("[shot] → dev/desktop_probe/select_popup_shot3.png")
	}
	fmt.Println("DONE")
}

func countNonzero(b []byte) int {
	n := 0
	for i := 0; i < len(b); i += 4 {
		if b[i] != 0 || b[i+1] != 0 || b[i+2] != 0 || b[i+3] != 0 {
			n++
		}
	}
	return n
}

func findClass(doc *dom.Document, cls string) *dom.Element {
	for _, el := range doc.GetElementsByClassName(cls) {
		return el
	}
	return nil
}

// findBoxByClass 遍历渲染树找 class 匹配的 RenderBox
func findBoxByClass(rv *rendering.RenderView, cls string) *rendering.RenderBox {
	var found *rendering.RenderBox
	dump := func() {}
	_ = dump
	var walk func(ro rendering.RenderObject, depth int)
	walk = func(ro rendering.RenderObject, depth int) {
		if found != nil {
			return
		}
		var tag, class string
		if el, ok := ro.Node().(*dom.Element); ok {
			tag = el.LocalName()
			class = el.GetAttribute("class")
			if el.HasClassName(cls) {
				if b, ok := ro.(interface{ AsRenderBox() *rendering.RenderBox }); ok {
					found = b.AsRenderBox()
					return
				}
			}
		} else if ro.Node() != nil {
			tag = "text"
		}
		if depth < 4 && (class != "" || tag == "body" || tag == "html") {
			fmt.Printf("[tree d%d] %s class=%q\n", depth, tag, class)
		}
		for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, depth+1)
		}
	}
	for c := rv.FirstChild(); c != nil; c = c.NextSibling() {
		walk(c, 0)
	}
	return found
}
