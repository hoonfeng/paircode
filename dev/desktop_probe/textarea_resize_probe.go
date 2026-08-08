// textarea_resize_probe 验证「多行输入框（textarea）拖动高度不跟手」
// （用户历史反馈，明确纠正过对象是 textarea 而非滑块）：
// 在真实设置面板「指令」tab 中模拟真实事件循环拖动 textarea 右下角
// resize 手柄——
//   每帧: RebuildRenderTree（style height 变更生效）→ EnsureLayout →
//         box 高度采样 → 注入真实 EventCursorMove 路径
//         （MockEventCursorMove，含 resize 拖拽分支：
//         resolveDragRV → newH=startH+(cssY-startY) → 写 style height）
//         → runJobs（JS 微任务）
// 输出每帧 box 高度 vs 期望（startH+dy），验证:
//   A. 拖动中 box 高度实时跟随鼠标（每帧 +10px，无滞后）
//   B. 渲染树重建后（RV 快照过期场景）高度计算不跳变
//   C. min-height clamp 生效（向下拖到 <60px 被夹住）
//   D. 释放后高度保持（style height 生效，不再回弹）
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
	wv.RebuildRenderTree()
	wv.EnsureLayout()

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

	// 切到「指令」tab（id=instructions，含 .inst-textarea）
	js(wv, `(function(){
		var btns = document.querySelectorAll('.settings-tabs button');
		var info = [];
		for (var i=0;i<btns.length;i++){
			info.push(i + ':' + JSON.stringify(btns[i].textContent) + ':' + btns[i].className);
			if (btns[i].textContent.indexOf('指令') >= 0) {
				var ev = new Event('click', {bubbles:true}); btns[i].dispatchEvent(ev); break;
			}
		}
		window.__tabInfo = info.join(' | ');
	})()`)
	fmt.Println("[resize] tab buttons:", js(wv, `window.__tabInfo || ''`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	doc := wv.Document()
	rv := wv.RenderView()

	// 找第一个 .inst-textarea（系统指令 textarea）
	var ta *dom.Element
	for _, el := range doc.GetElementsByTagName("textarea") {
		cls := el.GetAttribute("class")
		if strings.Contains(cls, "inst-textarea") {
			ta = el
			break
		}
	}
	if ta == nil {
		fmt.Println("[resize] NO .inst-textarea found in instructions tab")
		os.Exit(1)
	}
	box := rv.FindRenderBoxForNode(ta)
	if box == nil {
		fmt.Println("[resize] NO RenderBox for textarea")
		os.Exit(1)
	}
	// 手柄 = box 视口矩形右下角（BoxViewportRect 已减祖先滚动偏移）
	bx, by, bw, bh := rendering.BoxViewportRect(rv, box)
	fmt.Printf("[resize] textarea viewport=(%.0f,%.0f) %.0fx%.0f  styleH=%q rows=%s resize=%v\n",
		bx, by, bw, bh, ta.GetAttribute("style"), ta.GetAttribute("rows"),
		func() string {
			if st := box.Style(); st != nil {
				return fmt.Sprintf("%v", rendering.ResizeModeOf(st))
			}
			return "?"
		}())

	h := app.NewHostForTest(wv, 1280, 800)
	handleX := bx + bw - 5
	handleY := by + bh - 5

	// press 手柄：y 向下拖动 dy → 期望新高度 = startH + dy
	startH := bh
	startY := handleY
	h.MockTextareaResizePress(ta, rv, startY)
	fmt.Printf("[resize] press handle at (%.0f,%.0f) startH=%.1f\n", handleX, handleY, startH)

	// 事件循环 10 帧，每帧 +10px Y
	fmt.Printf("\n[resize] 事件循环模拟（每帧 move +10px Y，检查高度跟手）\n")
	frames := 10
	var prevPtr string
	mismatch := 0
	for i := 0; i < frames; i++ {
		// 真实主循环时序：重建 dirty 渲染树（style height 变更生效）→ 布局
		wv.RebuildRenderTree()
		wv.EnsureLayout()
		rv = wv.RenderView()
		box = rv.FindRenderBoxForNode(ta)
		if box == nil {
			fmt.Printf("[frame %02d] NO BOX\n", i)
			continue
		}
		ptr := fmt.Sprintf("%p", box)
		rebuilt := ""
		if prevPtr != "" && prevPtr != ptr {
			rebuilt = " ★REBUILT"
		}
		prevPtr = ptr
		dy := 10.0 * float64(i+1)
		expectH := startH + dy
		actualH := box.Height()
		styleH := ta.GetAttribute("style")
		// 帧末注入真实 move 事件（resize 拖拽分支）
		h.MockEventCursorMove(wv, handleX, startY+dy)
		runJobs(wv)
		ok := "OK"
		if abs(actualH-expectH) > 1.5 {
			ok = "✗ MISMATCH"
			mismatch++
		}
		fmt.Printf("[frame %02d] boxH=%.1f expect=%.1f dy=%+0.f style=%q %s%s\n",
			i, actualH, expectH, dy, styleH, ok, rebuilt)
	}
	fmt.Printf("\n[resize] mismatch=%d/%d（容差 1.5px，取整偏差可忽略）\n", mismatch, frames)

	// 释放后验证：高度保持（style height 生效，不再回弹）
	h.MockTextareaResizeRelease()
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv = wv.RenderView()
	box = rv.FindRenderBoxForNode(ta)
	if box == nil {
		fmt.Println("[resize] NO BOX after release")
		os.Exit(1)
	}
	finalH := box.Height()
	expectFinal := startH + 10.0*float64(frames)
	fmt.Printf("[resize] release: finalH=%.1f expect=%.1f style=%q %s\n",
		finalH, expectFinal, ta.GetAttribute("style"),
		map[bool]string{true: "OK（保持）", false: "✗ 回弹"}[abs(finalH-expectFinal) <= 1.5])

	// min-height clamp 验证：向下拖 < 60px 应被夹住
	// （.inst-textarea min-height:60px）
	h.MockTextareaResizePress(ta, rv, finalH)
	js(wv, `(function(){ var t = document.querySelector('textarea.inst-textarea'); t.style.height='30px'; })()`)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv = wv.RenderView()
	box = rv.FindRenderBoxForNode(ta)
	if box != nil {
		fmt.Printf("[resize] clamp: style=30px → boxH=%.1f（min-height:60px）%s\n",
			box.Height(), map[bool]string{true: "OK clamp 生效", false: "✗ 未 clamp"}[abs(box.Height()-60) <= 1.5])
	}
	h.MockTextareaResizeRelease()
	fmt.Println("\n[resize] done")
}

// ── 通用小工具（与 drag_repaint_probe 一致）──

func js(wv *webkit.WebView, code string) string {
	if wv.JSInterpreter() == nil {
		return ""
	}
	v, err := wv.JSInterpreter().RunJS(code)
	if err != nil {
		return "ERR:" + err.Error()
	}
	return v.ToString()
}

func runJobs(wv *webkit.WebView) {
	if wv.JSInterpreter() != nil {
		wv.JSInterpreter().RunJobs()
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
