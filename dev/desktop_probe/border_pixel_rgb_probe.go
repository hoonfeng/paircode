// 独立像素对比测试：渲染单个 div（border-left: 2px solid #58a6ff;
// border-radius: 6px; background: #1f2b3d），输出目标区域精确 RGBA，
// 与浏览器（web_debug 截图）同区域逐像素对比。
package main

import (
	"fmt"
	"os"

	"wb-ui/dom"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
)

func main() {
	x, y, w, h := 100.0, 100.0, 100.0, 40.0
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
	}
	doc := dom.NewDocument()
	el := doc.CreateElement("div")
	st := style.NewComputedStyle()
	blue := style.Color{R: 88, G: 166, B: 255, A: 255}
	trans := style.Color{A: 0}
	bg := style.Color{R: 31, G: 43, B: 61, A: 255}
	st.BackgroundColor = bg
	st.BorderLeftWidth = style.Length{Value: 2, Unit: "px"}
	st.BorderTopWidth = style.Length{Value: 0, Unit: "px"}
	st.BorderRightWidth = style.Length{Value: 0, Unit: "px"}
	st.BorderBottomWidth = style.Length{Value: 0, Unit: "px"}
	st.BorderLeftColor = blue
	st.BorderTopColor = trans
	st.BorderRightColor = trans
	st.BorderBottomColor = trans
	st.BorderRadius = style.Length{Value: 6, Unit: "px"}
	st.BorderLeftStyle = "solid"
	st.BorderTopStyle = "none"
	st.BorderRightStyle = "none"
	st.BorderBottomStyle = "none"

	box := rendering.NewRenderBox(el, st)
	box.SetLocation(x, y)
	box.SetSize(w, h)

	canvas := graphics.NewCanvas(300, 200)
	defer canvas.Release()
	info := rendering.NewPaintInfo(canvas, rendering.Rect{X: 0, Y: 0, Width: 300, Height: 200})
	rendering.PaintBackground(box, info)
	rendering.PaintBorder(box, info)

	// 分步诊断：只画背景 / 只画边框，看 y=103 行（中段）范围
	cb := graphics.NewCanvas(300, 200)
	defer cb.Release()
	ib := rendering.NewPaintInfo(cb, rendering.Rect{X: 0, Y: 0, Width: 300, Height: 200})
	rendering.PaintBackground(box, ib)
	fmt.Print("bgOnly y=103: ")
	for xx := 96; xx <= 112; xx++ {
		p := cb.PixelAt(xx, 103)
		if p.A < 8 {
			fmt.Print(" --")
		} else {
			fmt.Printf(" %02x%02x%02x", p.R, p.G, p.B)
		}
	}
	fmt.Println()
	// 主 canvas y=103 行完整 RGBA（定位 62% 蓝来源）
	fmt.Print("main y=103 RGBA: ")
	for xx := 96; xx <= 112; xx++ {
		p := canvas.PixelAt(xx, 103)
		fmt.Printf("x%d=%02x%02x%02x/%d ", xx, p.R, p.G, p.B, p.A)
	}
	fmt.Println()
	cf := graphics.NewCanvas(300, 200)
	defer cf.Release()
	inf := rendering.NewPaintInfo(cf, rendering.Rect{X: 0, Y: 0, Width: 300, Height: 200})
	rendering.PaintBorder(box, inf)
	fmt.Print("borderOnly y=103: ")
	for xx := 96; xx <= 112; xx++ {
		p := cf.PixelAt(xx, 103)
		if p.A < 8 {
			fmt.Print(" --")
		} else {
			fmt.Printf(" %02x%02x%02x", p.R, p.G, p.B)
		}
	}
	fmt.Println()

	// 最小测试：FillRect 1x33（中段矩形）覆盖哪些 x（诊断宽 2px 偏移）
	c4 := graphics.NewCanvas(300, 200)
	defer c4.Release()
	c4.FillRect(100, 103, 1, 33, graphics.Color{R: 88, G: 166, B: 255, A: 255})
	c4.FillRect(101, 103, 1, 33, graphics.Color{R: 88, G: 166, B: 255, A: 158})
	fmt.Print("midRect1x33: ")
	for xx := 98; xx <= 105; xx++ {
		p := c4.PixelAt(xx, 110)
		fmt.Printf("x=%d(%d,%d,%d,%d) ", xx, p.R, p.G, p.B, p.A)
	}
	fmt.Println()

	// 预热：先完整读一遍，确保 pixelCache 刷新到最新（PixelAt 首次大量
	// 读取偶发读到旧表面数据，导致区域输出与后续 RGBA 诊断不一致）
	for yy := 99; yy <= 110; yy++ {
		for xx := 96; xx <= 112; xx++ {
			_ = canvas.PixelAt(xx, yy)
		}
	}
	out := os.Stdout
	for yy := 99; yy <= 110; yy++ {
		fmt.Fprintf(out, "y=%3d:", yy)
		for xx := 96; xx <= 112; xx++ {
			p := canvas.PixelAt(xx, yy)
			if p.A < 8 {
				fmt.Fprintf(out, " ffffff")
				continue
			}
			fmt.Fprintf(out, " %02x%02x%02x", p.R, p.G, p.B)
		}
		fmt.Fprintln(out)
	}
	// 底部区域（y=133..142，左下弧带对称验证）
	for yy := 133; yy <= 142; yy++ {
		fmt.Fprintf(out, "by=%3d:", yy)
		for xx := 96; xx <= 112; xx++ {
			p := canvas.PixelAt(xx, yy)
			if p.A < 8 {
				fmt.Fprintf(out, " ffffff")
				continue
			}
			fmt.Fprintf(out, " %02x%02x%02x", p.R, p.G, p.B)
		}
		fmt.Fprintln(out)
	}
}
