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

	// 2.5 长小数验证：拖到 33% → val=0.66 → step=0.1 吸附 0.7。
	//    期望 "0.7"（1 位小数），绝不出 "0.7000000000000001"。
	p33 := left + box.Width()*0.33
	v25 := h.MockRangeMove(p33)
	dp := strings.IndexByte(v25, '.')
	decLen := 0
	if dp >= 0 {
		decLen = len(v25) - dp - 1
	}
	fmt.Printf("[range] drag → 33%% x=%.0f → value=%q (want \"0.7\") 小数位=%d\n", p33, v25, decLen)
	if decLen > 1 {
		fmt.Printf("[range] ★ 长小数 BUG：%q\n", v25)
	} else {
		fmt.Println("[range] 值格式干净 ✓")
	}

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

	// 几何像素测量：thumb 直径（期望 14px）与 track 高度（期望 6px）
	pngBytes, err := wv.Render()
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	pxAt := func(x, y int) (int, int, int, int) {
		o := (y*wv.Width() + x) * 4
		if o+3 < len(pngBytes) {
			return int(pngBytes[o]), int(pngBytes[o+1]), int(pngBytes[o+2]), int(pngBytes[o+3])
		}
		return -1, -1, -1, -1
	}
	isBlue := func(r, g, b int) bool { return b > 180 && r < 80 && g < 170 && b > r+60 }
	isGrey := func(r, g, b int) bool { return r >= 228 && r <= 248 && g >= 228 && g <= 248 && b >= 228 && b <= 248 && abs(r-g) < 4 && abs(g-b) < 4 }
	// thumb 垂直直径：thumbX 列连续 blue 行数
	tx := int(thumbX)
	ty := int(thumbY)
	tTop, tBot := ty, ty
	for tTop > 0 {
		r, g, b, _ := pxAt(tx, tTop-1)
		if isBlue(r, g, b) {
			tTop--
		} else {
			break
		}
	}
	for tBot < wv.Height()-1 {
		r, g, b, _ := pxAt(tx, tBot+1)
		if isBlue(r, g, b) {
			tBot++
		} else {
			break
		}
	}
	thumbD := tBot - tTop + 1
	// track 高度：thumb 右侧（thumbX+12，已过 thumb）扫 grey 行数
	gx := tx + 12
	gTop, gBot := ty, ty
	for gTop > 0 {
		r, g, b, _ := pxAt(gx, gTop-1)
		if isGrey(r, g, b) {
			gTop--
		} else {
			break
		}
	}
	for gBot < wv.Height()-1 {
		r, g, b, _ := pxAt(gx, gBot+1)
		if isGrey(r, g, b) {
			gBot++
		} else {
			break
		}
	}
	trackH := gBot - gTop + 1
	// fill 段高度：track 起点（left+10）扫 blue 行数
	fx := int(left) + 10
	fTop, fBot := ty, ty
	for fTop > 0 {
		r, g, b, _ := pxAt(fx, fTop-1)
		if isBlue(r, g, b) {
			fTop--
		} else {
			break
		}
	}
	for fBot < wv.Height()-1 {
		r, g, b, _ := pxAt(fx, fBot+1)
		if isBlue(r, g, b) {
			fBot++
		} else {
			break
		}
	}
	fillH := fBot - fTop + 1
	// thumb 水平直径：thumb 右侧边缘半径×2（thumbX 左侧被 fill 蓝色污染，
	// 只能测右半径：从圆心向右到第一个非 blue 像素）
	rT := tx
	for rT < wv.Width()-1 {
		r, g, b, _ := pxAt(rT+1, ty)
		if isBlue(r, g, b) {
			rT++
		} else {
			break
		}
	}
	thumbW := (rT - tx) * 2
	fmt.Printf("[geo] thumb 直径 %dpx (期望 14)  track 高 %dpx (期望 6)  fill 高 %dpx (期望 6)\n",
		thumbD, trackH, fillH)
	if abs(thumbD-14) <= 2 && abs(thumbW-14) <= 2 {
		fmt.Printf("[geo] thumb 垂直 %d / 水平 %d 均≈14 ✓\n", thumbD, thumbW)
	} else {
		fmt.Printf("[geo] ★ thumb 尺寸异常 (垂直 %d / 水平 %d)\n", thumbD, thumbW)
	}
	if abs(trackH-6) <= 2 && abs(fillH-6) <= 2 {
		fmt.Printf("[geo] track %d / fill %d 均≈6 ✓\n", trackH, fillH)
	} else {
		fmt.Printf("[geo] ★ track/fill 高度异常 (track %d / fill %d)\n", trackH, fillH)
	}

	// ── hover 分开验证（浏览器语义，用户实测 Chromium 深色主题）──
	//   A. hover 到条（fill 段非圆）→ 只蓝条 fill 变暗 (0,99,216)，
	//      圆 thumb 不变 (0,117,255)
	//   B. hover 到圆（thumb 内）→ 圆 thumb 变暗 (0,99,216)（整体）
	//   C. 移出 hover → 全部恢复
	// 渲染 hover 前帧
	px := func(x, y float64) string {
		r, g, b, a := pxAt(int(x), int(y))
		return fmt.Sprintf("(%d,%d,%d,%d)", r, g, b, a)
	}
	// thumb 圆心右侧 4px（半径 7 内，避开可能的光晕）
	thumbC := thumbX + 4
	// fill 段内、远离 thumb 的点（frac=0.9，fill 覆盖 0..90%，45% 处非圆内）
	fillX := left + box.Width()*0.45
	fmt.Printf("[thumb-normal] (%d,%d) rgba=%s 期望 (0,117,255)\n", int(thumbC), int(thumbY), px(thumbC, thumbY))
	fmt.Printf("[fill-normal]   (%d,%d) rgba=%s 期望 (0,117,255)\n", int(fillX), int(thumbY), px(fillX, thumbY))

	// A. hover 到 fill 段（非圆）
	hitA := h.MockMouseMove(wv, fillX, thumbY)
	if hitA == nil {
		fmt.Println("[hover-A] HitTest MISS on fill")
	} else {
		wv.RebuildRenderTree()
		wv.EnsureLayout()
		pngBytes, err = wv.Render()
		if err != nil {
			log.Fatalf("render: %v", err)
		}
		fillColA := px(fillX, thumbY)
		thumbColA := px(thumbC, thumbY)
		fmt.Printf("[hover-A fill]  (%d,%d) rgba=%s 期望变暗 (0,99,216)±2\n", int(fillX), int(thumbY), fillColA)
		fmt.Printf("[hover-A thumb] (%d,%d) rgba=%s 期望不变 (0,117,255)\n", int(thumbC), int(thumbY), thumbColA)
	}

	// B. hover 到圆（thumb 圆心）
	hitB := h.MockMouseMove(wv, thumbX, thumbY)
	curRV := wv.RenderView()
	cx, cy := curRV.CursorPos()
	fmt.Printf("[hover-B] cursor=(%.0f,%.0f) thumb=(%.0f,%.0f) hit=%v\n", cx, cy, thumbX, thumbY, hitB)
	if hitB == nil {
		fmt.Println("[hover-B] HitTest MISS on thumb")
	} else {
		wv.RebuildRenderTree()
		wv.EnsureLayout()
		pngBytes, err = wv.Render()
		if err != nil {
			log.Fatalf("render: %v", err)
		}
		thumbColB := px(thumbC, thumbY)
		fillColB := px(fillX, thumbY)
		fmt.Printf("[hover-B thumb] (%d,%d) rgba=%s 期望变暗 (0,99,216)±2\n", int(thumbC), int(thumbY), thumbColB)
		fmt.Printf("[hover-B fill]  (%d,%d) rgba=%s 期望变暗 (0,99,216)±2\n", int(fillX), int(thumbY), fillColB)
	}

	// C. 移出 hover → 恢复
	h.MockMouseMove(wv, 100, 100)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	pngBytes, err = wv.Render()
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	fmt.Printf("[thumb-out] (%d,%d) rgba=%s 期望恢复 (0,117,255)\n", int(thumbC), int(thumbY), px(thumbC, thumbY))
	fmt.Println("DONE")
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
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
