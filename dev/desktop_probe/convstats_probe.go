// Command convstats_probe 精确测量 ConvSidebar 中 conv-stats-header 的
// ▸ 三角渲染位置是否垂直居中：
//   1. 加载真实 dist（同 folded_probe 管线）
//   2. 注入 conversations + 点击 conv-item（保持折叠态）
//   3. dump conv-stats / conv-stats-header / conv-stats-chevron 的 box 几何
//   4. 渲染 PNG，供像素级校验 ▸ 字形中心 vs header 内容区中心
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

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func setupLoadersC(wv *webkit.WebView, distDir string) {
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

func waitJSC(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func runJobsC(wv *webkit.WebView) {
	for i := 0; i < 10; i++ {
		wv.JSInterpreter().RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		time.Sleep(12 * time.Millisecond)
	}
}

func dumpBoxes(wv *webkit.WebView, tag string) {
	rv := wv.RenderView()
	if rv == nil {
		log.Fatalf("no render view at %s", tag)
	}
	var walk func(o rendering.RenderObject, depth int)
	walk = func(o rendering.RenderObject, depth int) {
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			if strings.Contains(cn, "conv-stats") && depth < 40 {
				lb := o.LayoutBox()
				line := ""
				if lb != nil && rv.LayoutState() != nil {
					g2 := rv.LayoutState().GeometryForBox(lb)
					line = fmt.Sprintf(" rect=(%.1f,%.1f %.1fx%.1f)", g2.Left(), g2.Top(), g2.BorderBoxWidth(), g2.BorderBoxHeight())
				}
				// 文本内容
				texts := ""
				var walkT func(s rendering.RenderObject, d int)
				walkT = func(s rendering.RenderObject, d int) {
					if d > 2 {
						return
					}
					if rt, ok := s.(*rendering.RenderText); ok {
						t := strings.TrimSpace(rt.Text())
						if t != "" && len(t) < 30 {
							sb := rt.LayoutBox()
							sg := ""
							if sb != nil && rv.LayoutState() != nil {
								g3 := rv.LayoutState().GeometryForBox(sb)
								sg = fmt.Sprintf("@(%.1f,%.1f %.1fx%.1f)", g3.Left(), g3.Top(), g3.BorderBoxWidth(), g3.BorderBoxHeight())
							}
							texts += fmt.Sprintf(" [%q %s]", t, sg)
						}
					}
					for c := s.FirstChild(); c != nil; c = c.NextSibling() {
						walkT(c, d+1)
					}
				}
				walkT(o, 0)
				fmt.Printf("[convstats] %s %s cls=%-24s%s%s\n", tag, strings.Repeat(" ", depth), cn, line, texts)
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, depth+1)
		}
	}
	walk(rendering.RenderObject(rv), 0)
}

func renderPNGC(wv *webkit.WebView, path string) {
	if pngBytes, err := wv.Render(); err != nil {
		log.Printf("Render: %v", err)
		return
	} else {
		w, h := wv.Width(), wv.Height()
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				off := (y*w + x) * 4
				if off+3 < len(pngBytes) {
					img.SetRGBA(x, y, color.RGBA{R: pngBytes[off], G: pngBytes[off+1], B: pngBytes[off+2], A: pngBytes[off+3]})
				}
			}
		}
		f, _ := os.Create(path)
		defer f.Close()
		_ = png.Encode(f, img)
		fmt.Printf("[convstats] rendered %dx%d -> %s\n", w, h, path)
	}
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
	setupLoadersC(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	waitJSC(wv, 500)
	waitJSC(wv, 500)
	runJobsC(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 注入 conversations
	iv, _ := wv.JSInterpreter().RunJS(`(function(){
		var st = window.__state;
		if (!st) { return JSON.stringify({fatal: 'no state'}); }
		st.workspaceRoot = 'F:\\syproject\\gou-ide';
		st.workspaceName = 'gou-ide';
		st.conversations = [
			{id:'c1', title:'测试对话一', updatedAt:'2026-08-05T10:00:00Z'},
			{id:'c2', title:'测试对话二', updatedAt:'2026-08-05T09:00:00Z'},
			{id:'c3', title:'测试对话三', updatedAt:'2026-08-05T08:00:00Z'}
		];
		// ★ 折叠状态：两个统计面板都折叠 → chevron 显示 ▸
		if (typeof st.convStatsExpanded !== 'undefined') st.convStatsExpanded = false;
		if (typeof st.ctxStatsExpanded !== 'undefined') st.ctxStatsExpanded = false;
		return JSON.stringify({ok:1, convs: st.conversations.length, cse: st.convStatsExpanded, ctxe: st.ctxStatsExpanded});
	})()`)
	fmt.Printf("[convstats] inject: %s\n", iv.ToString())
	waitJSC(wv, 400)
	runJobsC(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 点击第一个 conv-item
	rv := wv.RenderView()
	var target *dom.Element
	var tx, ty float64
	var findConv func(o rendering.RenderObject)
	findConv = func(o rendering.RenderObject) {
		if target != nil {
			return
		}
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			if strings.Contains(cn, "conv-item") {
				lb := o.LayoutBox()
				if lb != nil && rv.LayoutState() != nil {
					g := rv.LayoutState().GeometryForBox(lb)
					tx = g.Left() + 40
					ty = g.Top() + 6
					target = el
					return
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			findConv(c)
		}
	}
	findConv(rendering.RenderObject(rv))
	if target == nil {
		log.Fatal("no conv-item rendered")
	}
	fmt.Printf("[convstats] click conv at (%.0f, %.0f)\n", tx, ty)
	elHit := rendering.HitTest(rv, tx, ty, "onclick")
	if elHit != nil {
		elHit.DispatchEvent(dom.NewMouseEvent(dom.EventClick, true, true, false))
	}
	runJobsC(wv)
	waitJSC(wv, 800)
	runJobsC(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	dumpBoxes(wv, "chat")
	renderPNGC(wv, filepath.Join(wd, "dev", "desktop_probe", "convstats_chat.png"))

	// 返回列表视图（再次点击或切换）——检查列表视图的 conv-stats 位置
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var st = window.__state;
		if (st) { st.currentView = st.currentView || 'chat'; }
		return JSON.stringify({view: st ? st.currentView : 'no-state'});
	})()`)
	fmt.Printf("[convstats] current view: %s\n", iv.ToString())

	fmt.Printf("[convstats] done\n")
}
