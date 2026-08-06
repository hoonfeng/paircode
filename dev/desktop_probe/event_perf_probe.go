// Command event_perf_probe measures the event-handling chain cost vs content
// size: HitTest, hover-switch (RebuildRenderTree + Layout), and Paint, with
// N message combos injected into the real dist. Answers "why does UI response
// get slower as more content loads" by showing which stage grows with N.
package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func setupLoaders(wv *webkit.WebView, distDir string) {
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
				if err != nil {
					return "", err
				}
				return string(data), nil
			}
		}
	}
}

func countObjects(ro rendering.RenderObject) int {
	n := 1
	for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
		n += countObjects(c)
	}
	return n
}

// injectMessages sets st.messages to N synthetic combos (user + assistant with
// thinking/tool_call/content segments) matching the front-end's StoredMessage
// shape, then returns a JS expression that does the injection.
func injectJS(n int) string {
	const tpl = `(function(){
var st = window.__state;
if (!st) return JSON.stringify({err:'no __state'});
var msgs = [];
var think = '分析用户需求并规划执行步骤。首先确认问题背景，然后拆解为可执行的小任务，逐步推进并验证结果。';
var content = '已完成分析。根据当前项目状态，我检查了相关代码并定位到问题根因，接下来将实施修复并补充回归测试。';
var tool = 'run_command';
var argsRaw = '{"command":"go test ./..."}';
for (var i = 0; i < __N__; i++) {
  var base = i * 4;
  msgs.push({role:'user', _idx:base, content:'用户问题 ' + i + '：请帮我检查这段代码并优化性能，涉及文件较多需要仔细处理。'});
  msgs.push({role:'assistant', _idx:base+1, segments:[
    {type:'thinking', content:think},
    {type:'tool_call', name:tool, argsRaw:argsRaw, result:'ok', duration_ms:120},
    {type:'content', content:content}
  ]});
}
st.messages = msgs;
st.messagesByConv['perf_conv'] = msgs;
return JSON.stringify({n: msgs.length});
})()`
	return strings.Replace(tpl, "__N__", fmt.Sprint(n), 1)
}

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	absDist, _ := filepath.Abs(distDir)
	htmlData, err := os.ReadFile(filepath.Join(absDist, "index.html"))
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
	layout.FontMetricsFunc = func(family string, size float64, weight int, style2 string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}
	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	waitJS := func(ms int) {
		_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	}
	waitJS(400)
	waitJS(600)

	// workspace (needed by RightPanel)
	_, _ = wv.JSInterpreter().RunJS(`(function(){ var st=window.__state; st.workspaceRoot='F:\\syproject\\gou-ide'; st.workspaceName='gou-ide'; st.workspaceFolders=['F:\\syproject\\gou-ide']; st.currentConvId='perf_conv'; return 'ok'; })()`)

	W, H := 1280, 800
	cv := graphics.NewCanvas(W, H)

	fmt.Printf("%-8s %-10s %-10s %-12s %-12s %-12s %-10s\n", "combos", "objects", "hitTest", "rebuild", "layout", "paint", "hoverTotal")
	for _, n := range []int{5, 20, 50, 100, 200} {
		_, _ = wv.JSInterpreter().RunJS(injectJS(n))
		waitJS(700) // Vue patch + async render
		wv.RebuildRenderTree()
		wv.EnsureLayout()
		rv := wv.RenderView()
		if rv == nil {
			fmt.Println("no rv")
			continue
		}
		objCount := countObjects(rendering.RenderObject(rv))

		// ★ consume the full-viewport dirty left by EnsureLayout (a real host
		// paints every frame and clears it); afterwards MarkDirty() regions
		// are genuinely local.
		rendering.Paint(rv, cv, rendering.Rect{X: 0, Y: 0, Width: float64(W), Height: float64(H)})

		// --- HitTest: 60 random points, median-ish avg ---
		rng := rand.New(rand.NewSource(42))
		t0 := time.Now()
		const HT = 60
		var hits int
		for i := 0; i < HT; i++ {
			x := rng.Float64() * 400 // right panel area
			y := rng.Float64() * 700
			if rendering.HitTest(rv, x, y, "") != nil {
				hits++
			}
		}
		htAvg := time.Since(t0) / HT

		// --- hover switch: FAST PATH (new) — re-resolve only old/new hover
		// element styles, swap onto render objects, mark region dirty. No
		// render-tree rebuild, no layout (unless geometry-affecting). ---
		var target *dom.Element
		var walk func(ro rendering.RenderObject)
		walk = func(ro rendering.RenderObject) {
			if target != nil {
				return
			}
			if el, ok := ro.Node().(*dom.Element); ok {
				cn := el.GetAttribute("class")
				if strings.Contains(cn, "msg-item") || strings.Contains(cn, "conv-item") {
					target = el
					return
				}
			}
			for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c)
			}
		}
		walk(rendering.RenderObject(rv))
		if target == nil {
			fmt.Printf("no hover target for n=%d\n", n)
			continue
		}
		fr := wv.MainFrame().Frame()
		resolver := fr.Resolver()
		var findRO func(root rendering.RenderObject, node dom.Node) rendering.RenderObject
		findRO = func(root rendering.RenderObject, node dom.Node) rendering.RenderObject {
			if node == nil {
				return nil
			}
			for cur := root; cur != nil; cur = cur.NextInPreOrder() {
				if cur.Node() == node {
					return cur
				}
			}
			return nil
		}
		t0 = time.Now()
		target.SetHovered(true)
		resolver.Invalidate(target)
		newCS := resolver.ResolveElement(target)
		ro := findRO(rendering.RenderObject(rv), target)
		oldCS := ro.Style()
		if n == 50 {
			oc, nc := style.Color{}, style.Color{}
			if oldCS != nil {
				oc = oldCS.BackgroundColor
			}
			if newCS != nil {
				nc = newCS.BackgroundColor
			}
			fmt.Printf("[fastpath] hover target=%s bg #%02x%02x%02x → #%02x%02x%02x (changed=%v)\n",
				target.GetAttribute("class"), oc.R, oc.G, oc.B, nc.R, nc.G, nc.B,
				oldCS == nil || newCS == nil || oc != nc)
		}
		ro.SetStyle(newCS)
		if lb := ro.LayoutBox(); lb != nil && rv.LayoutState() != nil {
			g := rv.LayoutState().GeometryForBox(lb)
			rv.MarkDirty(rendering.Rect{X: g.Left() - 2, Y: g.Top() - 2, Width: g.BorderBoxWidth() + 4, Height: g.BorderBoxHeight() + 4})
		}
		rebuildT := time.Since(t0)

		// --- Paint (dirty-rect local repaint) ---
		if n == 100 {
			sx0, sy0 := rv.ScrollOffset()
			fmt.Printf("[paint-diag] n=%d dirty=%v dirtyRect=%.0f,%.0f %.0fx%.0f scrollOffsets=%d pageScroll=(%.0f,%.0f)\n",
				n, rv.IsDirty(), rv.GetDirtyRect().X, rv.GetDirtyRect().Y, rv.GetDirtyRect().Width, rv.GetDirtyRect().Height,
				rv.ScrollOffsetCount(), sx0, sy0)
		}
		t0 = time.Now()
		scBefore := cv.SaveCount()
		if n == 200 {
			f, _ := os.Create("paint_cpu.pprof")
			pprof.StartCPUProfile(f)
		}
		rendering.Paint(rv, cv, rendering.Rect{X: 0, Y: 0, Width: float64(W), Height: float64(H)})
		if n == 200 {
			pprof.StopCPUProfile()
			log.Printf("[pprof] paint_cpu.pprof written")
		}
		scAfter := cv.SaveCount()
		if scAfter != scBefore {
			log.Printf("[saveleak] n=%d SaveCount %d → %d (leak %d)", n, scBefore, scAfter, scAfter-scBefore)
		}
		paintT := time.Since(t0)

		// restore hover off via fast path (clean for next iteration)
		target.SetHovered(false)
		resolver.Invalidate(target)
		newCS2 := resolver.ResolveElement(target)
		ro.SetStyle(newCS2)
		if lb := ro.LayoutBox(); lb != nil && rv.LayoutState() != nil {
			g := rv.LayoutState().GeometryForBox(lb)
			rv.MarkDirty(rendering.Rect{X: g.Left() - 2, Y: g.Top() - 2, Width: g.BorderBoxWidth() + 4, Height: g.BorderBoxHeight() + 4})
		}
		rendering.Paint(rv, cv, rendering.Rect{X: 0, Y: 0, Width: float64(W), Height: float64(H)})
		rv.ClearDirty()

		fmt.Printf("%-8d %-10d %-10s %-12s %-12s %-12s %-10s\n",
			n, objCount, htAvg.Round(time.Microsecond), rebuildT.Round(time.Microsecond),
			time.Since(t0).Round(time.Microsecond), paintT.Round(time.Microsecond),
			(rebuildT + paintT).Round(time.Microsecond))
	}
	fmt.Println("done")
}
