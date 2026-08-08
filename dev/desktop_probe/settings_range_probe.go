// settings_range_probe 在真实设置面板中验证 <input type="range"> 交互：
//   1. 打开设置面板 → 找温度 range（min=0 max=2 step=0.1）
//   2. MockRangePress 点击 75% 位置 → value 应吸附到 1.5（step 取整）
//   3. MockRangeMove 拖到 25% → value 实时更新 0.5（input 事件）
//   4. MockRangeRelease → 派发 change，最终值 0.5
//   5. hover 像素：thumb 正常 vs hover 变亮对比
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

	doc := wv.Document()
	rv := wv.RenderView()

	// 找温度 range
	var rangeEl *dom.Element
	for _, el := range doc.GetElementsByTagName("input") {
		if el.GetAttribute("type") == "range" {
			rangeEl = el
			break
		}
	}
	if rangeEl == nil {
		fmt.Println("[range] NO input[type=range] found in settings modal")
		os.Exit(1)
	}
	box := rv.FindRenderBoxForNode(rangeEl)
	if box == nil {
		fmt.Println("[range] NO RenderBox")
		os.Exit(1)
	}
	left, top := box.AbsoluteX(), box.AbsoluteY()
	_, so := rv.BoxScrollOffset(box)
	left -= so
	fmt.Printf("[range] box at (%.0f,%.0f) %.0fx%.0f min/max=%s/%s step=%s value=%s\n",
		left, top, box.Width(), box.Height(),
		rangeEl.GetAttribute("min"), rangeEl.GetAttribute("max"),
		rangeEl.GetAttribute("step"), rangeEl.GetAttribute("value"))

	h := app.NewHostForTest(wv, 1280, 800)

	// 1. Press 75% 位置 → 期望 0+0.75*2=1.5
	p75 := left + box.Width()*0.75
	v1 := h.MockRangePress(rangeEl, rv, p75)
	fmt.Printf("[range] press @75%% x=%.0f → value=%s (want 1.5)\n", p75, v1)
	_ = v1

	// 2. Move 到 25% → 期望 0.5
	p25 := left + box.Width()*0.25
	v2 := h.MockRangeMove(p25)
	fmt.Printf("[range] drag → 25%% x=%.0f → value=%s (want 0.5)\n", p25, v2)

	// 3. Move 到 90% → 期望 1.8
	p90 := left + box.Width()*0.90
	v3 := h.MockRangeMove(p90)
	fmt.Printf("[range] drag → 90%% x=%.0f → value=%s (want 1.8)\n", p90, v3)

	// 4. Release → change
	v4 := h.MockRangeRelease()
	fmt.Printf("[range] release → value=%s (change dispatched)\n", v4)

	// 5. 重新 Press + hover 像素验证 thumb 高亮
	//    先 RebuildRenderTree 让 thumb 位置更新
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv = wv.RenderView()
	box = rv.FindRenderBoxForNode(rangeEl)
	if box == nil {
		fmt.Println("[range] NO RenderBox after rebuild")
		os.Exit(1)
	}
	left = box.AbsoluteX()
	// 当前 value=1.8 → frac=0.9
	cur := rangeEl.GetAttribute("value")
	frac := parseF(cur, 0) / 2.0 // max=2
	thumbX := left + box.Width()*frac
	thumbY := box.AbsoluteY() + box.Height()/2
	fmt.Printf("[range] thumb at (%.0f,%.0f) value=%s\n", thumbX, thumbY, cur)

	// hover：MockMouseMove 到 thumb 上，像素验证 thumb 变亮
	// （Chromium 深色主题 hover 亮化 ~16% 混白）
	// 先渲染 hover 前帧
	pngBytes, err := wv.Render()
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	px := func(x, y float64) string {
		o := (int(y)*wv.Width() + int(x)) * 4
		if o+3 < len(pngBytes) {
			return fmt.Sprintf("(%d,%d,%d,%d)", pngBytes[o], pngBytes[o+1], pngBytes[o+2], pngBytes[o+3])
		}
		return "OOB"
	}
	// thumb 圆心（半径 h*0.5≈10.5，取圆心右侧 4px 避开正中心可能的光晕）
	thumbC := thumbX + 4
	fmt.Printf("[thumb-normal] (%d,%d) rgba=%s 期望 (0,117,255)\n", int(thumbC), int(thumbY), px(thumbC, thumbY))

	hit := h.MockMouseMove(wv, thumbX, thumbY)
	if hit == nil {
		fmt.Println("[hover] HitTest MISS on thumb")
	} else {
		fmt.Printf("[hover] hit=%s.%s (hovered=%v)\n", hit.LocalName(), hit.GetAttribute("type"), hit.IsHovered())
		// 再走一次完整渲染流程拿到 hover 高亮帧
		wv.RebuildRenderTree()
		wv.EnsureLayout()
		pngBytes, err = wv.Render()
		if err != nil {
			log.Fatalf("render: %v", err)
		}
		fmt.Printf("[thumb-hover] (%d,%d) rgba=%s 期望变亮 (41,140,255)±\n", int(thumbC), int(thumbY), px(thumbC, thumbY))
		// 移出 hover → 恢复
		h.MockMouseMove(wv, 100, 100)
		wv.RebuildRenderTree()
		wv.EnsureLayout()
		pngBytes, err = wv.Render()
		if err != nil {
			log.Fatalf("render: %v", err)
		}
		fmt.Printf("[thumb-out] (%d,%d) rgba=%s 期望恢复 (0,117,255)\n", int(thumbC), int(thumbY), px(thumbC, thumbY))
	}
	fmt.Println("DONE")
}

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

func parseF(s string, def float64) float64 {
	var v float64
	if _, err := fmt.Sscanf(s, "%g", &v); err != nil {
		return def
	}
	return v
}
