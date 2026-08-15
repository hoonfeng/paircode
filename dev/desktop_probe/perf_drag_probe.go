// perf_drag_probe 量化「textarea resize 拖拽不跟手」的每帧耗时分布：
// 在真实设置面板「指令」tab 模拟拖拽 30 帧，逐帧测量：
//   - MockEventCursorMove（事件处理 + SetAttribute("style",height)）
//   - RebuildRenderTree（DOM → RenderObject 全量重建）
//   - EnsureLayout（全树布局）
//   - runJobs（JS 微任务）
//   - 整帧合计
// 输出 avg/max/p95，定位瓶颈阶段（重建 vs 布局 vs JS）。
// 用法: set CGO_ENABLED=1 && go run ./dev/desktop_probe/perf_drag_probe.go
//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
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

// stats 收集一段阶段的耗时样本（avg/max/p95 统计）。
type stats struct {
	name string
	vals []float64
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

	// 切到「指令」tab
	js(wv, `(function(){
		var btns = document.querySelectorAll('.settings-tabs button');
		for (var i=0;i<btns.length;i++){
			if (btns[i].textContent.indexOf('指令') >= 0) {
				var ev = new Event('click', {bubbles:true}); btns[i].dispatchEvent(ev); break;
			}
		}
	})()`)
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

	var ta *dom.Element
	for _, el := range doc.GetElementsByTagName("textarea") {
		if strings.Contains(el.GetAttribute("class"), "inst-textarea") {
			ta = el
			break
		}
	}
	if ta == nil {
		fmt.Println("[perf] NO .inst-textarea found")
		os.Exit(1)
	}
	box := rv.FindRenderBoxForNode(ta)
	if box == nil {
		fmt.Println("[perf] NO RenderBox for textarea")
		os.Exit(1)
	}
	bx, by, bw, bh := rendering.BoxViewportRect(rv, box)
	objCount := countRO(rendering.RenderObject(rv))
	fmt.Printf("[perf] textarea viewport=(%.0f,%.0f) %.0fx%.0f renderObjects=%d\n",
		bx, by, bw, bh, objCount)

	h := app.NewHostForTest(wv, 1280, 800)
	handleX := bx + bw - 5
	startY := by + bh - 5
	h.MockTextareaResizePress(ta, rv, startY)

	s := func(n string) *stats { return &stats{name: n} }
	evSt, rbSt, jsSt, totSt := s("event"), s("layout"), s("jobs"), s("total")

	frames := 30
	// 帧序：事件（内部增量 RebuildStyleForElement，不重建渲染树）→ layout → jobs
	// （模拟真实主循环：事件在帧首处理，布局后绘制）
	for i := 0; i < frames; i++ {
		dy := 10.0 * float64(i+1)
		t0 := time.Now()
		h.MockEventCursorMove(wv, handleX, startY+dy)
		t1 := time.Now()
		wv.EnsureLayout()
		t2 := time.Now()
		runJobs(wv)
		t3 := time.Now()
		evSt.vals = append(evSt.vals, ms(t0, t1))
		rbSt.vals = append(rbSt.vals, ms(t1, t2))
		jsSt.vals = append(jsSt.vals, ms(t2, t3))
		totSt.vals = append(totSt.vals, ms(t0, t3))
		if os.Getenv("WB_LAYOUT_PROFILE") != "" {
			layout.DumpLayoutProfile()
		}
	}
	// 基准：空闲帧（无事件无 dirty）整帧成本
	idle := make([]float64, 0, frames)
	for i := 0; i < frames; i++ {
		t0 := time.Now()
		wv.EnsureLayout()
		t1 := time.Now()
		idle = append(idle, ms(t0, t1))
	}

	fmt.Println("\n[perf] 拖拽帧耗时分布（30 帧，单位 ms）")
	report(evSt)
	report(rbSt)
	report(jsSt)
	report(totSt)
	fmt.Printf("[perf] 空闲帧 EnsureLayout 仅: avg=%.2f max=%.2f\n", avg(idle), maxV(idle))
}

func runJobs(wv *webkit.WebView) {
	if wv.JSInterpreter() != nil {
		wv.JSInterpreter().RunJobs()
	}
}

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

func ms(a, b time.Time) float64 { return float64(b.Sub(a).Nanoseconds()) / 1e6 }

func report(st *stats) {
	v := st.vals
	sort.Float64s(v)
	p95 := v[len(v)*95/100]
	fmt.Printf("  %-8s avg=%6.2f  max=%6.2f  p95=%6.2f  [%6.2f .. %6.2f]\n",
		st.name, avg(v), maxV(v), p95, v[0], v[len(v)-1])
}

func avg(v []float64) float64 {
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}

func maxV(v []float64) float64 {
	m := 0.0
	for _, x := range v {
		if x > m {
			m = x
		}
	}
	return m
}

func countRO(ro rendering.RenderObject) int {
	n := 0
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil {
			return
		}
		n++
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(ro)
	return n
}
