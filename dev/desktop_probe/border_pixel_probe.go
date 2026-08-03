// 独立像素对比测试：渲染单个 div（border-left: 2px solid #58a6ff;
// border-radius: 6px; background: #1f2b3d），输出 RGBA 像素矩阵，
// 与 Edge headless 渲染同一元素逐像素对比。
package main

import (
	"fmt"
	"math"
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

	pix := canvas.Pixels()
	out := make([]byte, 0, len(pix))
	out = append(out, pix...)
	os.WriteFile("tmp/border_px_wb.rgba", out, 0o644)
	fmt.Println("saved tmp/border_px_wb.rgba 300x200")

	// 背景精确 RGB（y=104, x=98-112）
	fmt.Print("bgRGB y=104:")
	for xx := 98; xx < 112; xx++ {
		p := canvas.PixelAt(xx, 104)
		fmt.Printf(" (%d,%d,%d)", p.R, p.G, p.B)
	}
	fmt.Println()
	// 完整模拟：背景 + clip(box) + 矩形 + 内弧
	canvas.ClearRect(0, 0, 300, 200)
	rendering.PaintBackground(box, info)
	canvas.Save()
	canvas.Clip(graphics.Rect{X: 100, Y: 100, Width: 100, Height: 40})
	canvas.FillRect(100, 104, 2, 32, graphics.Color{R: 88, G: 166, B: 255, A: 255})
	ptsA := []graphics.Point{}
	for i := 0; i <= 6; i++ {
		a := 3*3.14159265/2 + (3.14159265-3*3.14159265/2)*float64(i)/float64(6)
		ptsA = append(ptsA, graphics.Point{X: 104 + 4*math.Cos(a), Y: 104 + 4*math.Sin(a)})
	}
	canvas.StrokePath(ptsA, 2, graphics.Color{R: 88, G: 166, B: 255, A: 255}, "butt", "miter")
	canvas.Restore()
	fmt.Print("fullSim y=103-106:")
	for y := 103; y <= 106; y++ {
		fmt.Printf(" y=%d:[", y)
		for x := 99; x <= 105; x++ {
			p := canvas.PixelAt(x, y)
			fmt.Printf("(%d,%d,%d)", p.R, p.G, p.B)
		}
		fmt.Print("]")
	}
	fmt.Println()
	canvas.ClearRect(0, 0, 300, 200)
	// 新 canvas 单独测 PaintBackground + PaintBorder
	c2 := graphics.NewCanvas(300, 200)
	defer c2.Release()
	info2 := rendering.NewPaintInfo(c2, rendering.Rect{X: 0, Y: 0, Width: 300, Height: 200})
	rendering.PaintBackground(box, info2)
	rendering.PaintBorder(box, info2)
	fmt.Print("freshCanvas y=103-106:")
	for y := 103; y <= 106; y++ {
		fmt.Printf(" y=%d:[", y)
		for x := 99; x <= 105; x++ {
			p := c2.PixelAt(x, y)
			fmt.Printf("(%d,%d,%d)", p.R, p.G, p.B)
		}
		fmt.Print("]")
	}
	fmt.Println()
	fmt.Println("=== wb-ui 左上角 (x=95..130, y=95..145) ===")
	for yy := 95; yy < 145; yy++ {
		row := fmt.Sprintf("y=%3d:", yy)
		for xx := 95; xx < 130; xx++ {
			px := canvas.PixelAt(xx, yy)
			ch := "."
			if px.A > 200 {
				if px.B > 200 && px.R < 160 {
					ch = "B"
				} else {
					ch = "#"
				}
			} else if px.A > 60 {
				ch = "b"
			}
			row += " " + ch
		}
		fmt.Println(row)
	}
}
