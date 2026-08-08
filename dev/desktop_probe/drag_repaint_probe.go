// drag_repaint_probe 验证「拖拽是否因不重绘而跟手性差」（用户疑问）：
// 在真实设置面板中模拟【真实事件循环时序】拖动 range 滑块——
//   每帧: EnsureLayout → 取 rv → 按需渲染判定 needPaint=rv.IsDirty()
//         → needPaint 时离屏 Paint + 采样 thumb 像素 → 注入真实
//         EventCursorMove 路径（MockEventCursorMove，含无条件
//         MarkAllDirty + range 拖拽分支）→ runJobs（JS 微任务）
// 输出每帧 needPaint 与 thumb 位置，验证:
//   A. 拖动中每帧 needPaint=true（不重绘假设不成立）
//   B. thumb 像素位置随鼠标实时推进（值变化当帧生效，非延迟到释放）
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
		fmt.Println("[drag] NO input[type=range] found in settings modal")
		os.Exit(1)
	}
	box := rv.FindRenderBoxForNode(rangeEl)
	if box == nil {
		fmt.Println("[drag] NO RenderBox")
		os.Exit(1)
	}
	left, top := box.AbsoluteX(), box.AbsoluteY()
	_, so := rv.BoxScrollOffset(box)
	left -= so
	bw := box.Width()
	yMid := top + box.Height()/2
	fmt.Printf("[drag] range box at (%.0f,%.0f) %.0fx%.0f min/max=%s/%s step=%s value=%s\n",
		left, top, bw, box.Height(),
		rangeEl.GetAttribute("min"), rangeEl.GetAttribute("max"),
		rangeEl.GetAttribute("step"), rangeEl.GetAttribute("value"))

	h := app.NewHostForTest(wv, 1280, 800)

	// 进入拖拽态：press 在 30% 位置
	x0 := left + bw*0.30
	v0 := h.MockRangePress(rangeEl, rv, x0)
	// press 后真实代码会 RebuildRenderTree + EnsureLayout
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Printf("[drag] press @30%% x=%.0f → value=%s (drag active)\n", x0, v0)

	// 离屏 canvas（模拟主循环的 gpuCanvas）
	canvas := graphics.NewCanvas(wv.Width(), wv.Height())
	defer canvas.Release()
	paintRect := rendering.Rect{X: 0, Y: 0, Width: float64(wv.Width()), Height: float64(wv.Height())}

	// 期望 thumb 位置（paintRangeSlider: thumbX = boxLeft + frac*boxWidth, frac=(val-min)/(max-min)）
	expectThumbX := func(el *dom.Element, bx *rendering.RenderBox, boxLeft, boxW float64) float64 {
		minV := parseF(el.GetAttribute("min"), 0)
		maxV := parseF(el.GetAttribute("max"), 100)
		val := parseF(el.GetAttribute("value"), minV)
		frac := 0.0
		if maxV > minV {
			frac = (val - minV) / (maxV - minV)
		}
		return boxLeft + frac*boxW
	}
	// 像素检测 thumb（蓝色系圆）：扫 yMid 行找蓝色像素的【右端】。
	// 蓝色像素包含 filled track 段（x..x+frac*w）与 thumb 圆
	// （thumb 右缘 = 圆心的最右），取 maxX 即 thumb 右缘，中心=maxX-7。
	thumbBlueX := func() float64 {
		w, hgt := canvas.Width(), canvas.Height()
		if yMid < 0 || int(yMid) >= hgt {
			return -1
		}
		maxX := -1
		rowY := int(yMid)
		for x := 0; x < w; x++ {
			p := canvas.PixelAt(x, rowY)
			// SliderFill 蓝 #0075FF，悬停/按下变暗仍是蓝色系：B 明显 > R 且 G 中等
			if int(p.B) > 150 && int(p.B) > int(p.R)+60 {
				maxX = x
			}
		}
		if maxX < 0 {
			return -1
		}
		return float64(maxX) - 7 // thumb 中心 = 右缘 - 半径 7
	}

	// 模拟真实事件循环 22 帧，x 从 30% 匀速拖到 70%（每帧 +1.8%）。
	// ★ 用【固定初始几何】计算 xNew（模拟真实鼠标在屏幕上的坐标——
	// 屏幕轨道宽度不变），setRangeValueFromX 内部用当前 box 几何。
	// 若重建后 box 宽度变化（420→171.6），两者错配会暴露为 value 跳变。
	initLeft, initWidth := left, box.Width()
	fmt.Printf("\n[drag] 事件循环模拟（每帧 EnsureLayout→needPaint 判定→Paint→move 事件）\n")
	frames := 22
	var prevBoxPtr string
	for i := 0; i < frames; i++ {
		// 帧首：EnsureLayout + 取 rv
		wv.EnsureLayout()
		rv = wv.RenderView()
		box = rv.FindRenderBoxForNode(rangeEl)
		if box == nil {
			fmt.Printf("[frame %02d] NO RenderBox (tree rebuilt?)", i)
			continue
		}
		left = box.AbsoluteX()
		_, so = rv.BoxScrollOffset(box)
		left -= so
		boxPtr := fmt.Sprintf("%p", box)
		rebuilt := "-"
		if prevBoxPtr != "" && prevBoxPtr != boxPtr {
			rebuilt = "★ REBUILT " + prevBoxPtr + "→" + boxPtr
		}
		prevBoxPtr = boxPtr
		// 诊断：原始 AbsoluteX 与滚动偏移 (sx, sy)
		absRaw := box.AbsoluteX()
		sox, soy := rv.BoxScrollOffset(box)
		boxW := box.Width()
		wMark := ""
		if abs(boxW-initWidth) > 1 {
			wMark = fmt.Sprintf(" ★ WIDTH %.1f→%.1f", initWidth, boxW)
		}
		parentInfo := ""
		if pEl := findSettingRow(rangeEl); pEl != nil {
			if pbb := rv.FindRenderBoxForNode(pEl); pbb != nil {
				parentInfo = fmt.Sprintf(" parentRow w=%.1f disp=%v", pbb.Width(), pbb.Style().Display)
			} else {
				parentInfo = " parentRow=nil"
			}
		}
		styleInfo := ""
		if st := box.Style(); st != nil {
			styleInfo = fmt.Sprintf(" flexGrow=%v w=%v disp=%v", st.FlexGrow, st.Width.String(), st.Display)
		}
		fmt.Printf("[frame %02d] absX=%.1f w=%.1f scroll=(%.1f,%.1f)%s %s%s%s\n", i, absRaw, boxW, sox, soy, wMark, rebuilt, parentInfo, styleInfo)
		// 打印 range 的 DOM 父链（class 标识），确认结构是否随重建变化
		chain := ""
		for cur := rangeEl.ParentNode(); cur != nil; cur = cur.ParentNode() {
			if ce, ok := cur.(*dom.Element); ok {
				cls := ce.GetAttribute("class")
				if len(cls) > 24 {
					cls = cls[:24]
				}
				chain += ce.LocalName() + "." + cls + " <- "
			}
		}
		fmt.Printf("        domChain: %s\n", chain)

		// 按需渲染判定
		needPaint := rv.IsDirty()
		if needPaint {
			// Paint 前快照 DOM value（paint 会读它）
			rendering.Paint(rv, canvas, paintRect)
			valNow := rangeEl.GetAttribute("value")
			expTx := expectThumbX(rangeEl, box, left, box.Width())
			pxTx := thumbBlueX()

			// 帧末：注入真实 move 事件（无条件 MarkAllDirty + 拖拽分支）
			// 用固定屏幕坐标（初始几何），模拟真实鼠标
			xNew := initLeft + initWidth*(0.30+0.0182*float64(i))
			valBefore := rangeEl.GetAttribute("value")
			h.MockEventCursorMove(wv, xNew, yMid)
			valAfterMove := rangeEl.GetAttribute("value")
			runJobs(wv)
			valAfterJobs := rangeEl.GetAttribute("value")

			status := "OK"
			if pxTx < 0 {
				status = "★ NO THUMB PIXEL"
			} else if expTx > 0 && pxTx > 0 && abs(pxTx-expTx) > 3 {
				status = fmt.Sprintf("★ PIXEL MISMATCH exp=%.1f px=%.1f", expTx, pxTx)
			}
			if valBefore != valAfterMove || valAfterMove != valAfterJobs {
				status += fmt.Sprintf(" ← val: pre=%s move→%s jobs→%s", valBefore, valAfterMove, valAfterJobs)
			}
			fmt.Printf("[frame %02d] needPaint=%-5v val=%-5s expectTx=%.1f pixelTx=%.1f %s %s\n",
				i, needPaint, valNow, expTx, pxTx, status, rebuilt)
		} else {
			fmt.Printf("[frame %02d] needPaint=false box=%s %s\n", i, boxPtr, rebuilt)
		}
	}

	// 释放
	h.MockRangeRelease()
	// 释放后 unfreeze → 下帧重建（Vue 变更生效）。验证重建后 range 宽度
	// 是否仍 420（布局断链是否只在拖拽序列中触发）。
	wv.EnsureLayout()
	rv = wv.RenderView()
	if bx := rv.FindRenderBoxForNode(rangeEl); bx != nil {
		fmt.Printf("[drag] release 后重建: range w=%.1f value=%s %s\n",
			bx.Width(), rangeEl.GetAttribute("value"),
			func() string {
				if abs(bx.Width()-initWidth) > 1 {
					return "（v-if hint: value>0.8 显示警告挤占轨道——浏览器行为）"
				}
				return "✓ 宽度保持"
			}())
	}
	// 再重建一次（模拟后续 DOM 操作）
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv = wv.RenderView()
	if bx := rv.FindRenderBoxForNode(rangeEl); bx != nil {
		fmt.Printf("[drag] 再次重建: range w=%.1f value=%s %s\n",
			bx.Width(), rangeEl.GetAttribute("value"),
			func() string {
				if abs(bx.Width()-initWidth) > 1 {
					return "（v-if hint 仍显示——与 Edge 一致）"
				}
				return "✓ 宽度保持"
			}())
	}
}

func findSettingRow(el *dom.Element) *dom.Element {
	for cur := el.ParentNode(); cur != nil; cur = cur.ParentNode() {
		if ce, ok := cur.(*dom.Element); ok {
			if strings.Contains(ce.GetAttribute("class"), "setting-row") {
				return ce
			}
		}
	}
	return nil
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func parseF(s string, def float64) float64 {
	if s == "" {
		return def
	}
	var v float64
	if _, err := fmt.Sscanf(s, "%g", &v); err != nil {
		return def
	}
	return v
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
