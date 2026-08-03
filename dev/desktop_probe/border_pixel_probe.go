// 独立像素对比测试：渲染单个 div（border-left: 2px solid #58a6ff;
// border-radius: 6px; background: #1f2b3d），输出 RGBA 像素矩阵，
// 与 Edge headless 渲染同一元素逐像素对比。
package main

import (
	"fmt"
	"math"

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

	// 验证 FillPath 月牙渲染位置：圆心 (106,106)，外弧 r6 + 内弧 r4 反向
	c3 := graphics.NewCanvas(300, 200)
	defer c3.Release()
	var pts []graphics.Point
	for i := 0; i <= 24; i++ {
		a := 3*3.14159265/2 + (3.14159265-3*3.14159265/2)*float64(i)/float64(24)
		pts = append(pts, graphics.Point{X: 106 + 6*math.Cos(a), Y: 106 + 6*math.Sin(a)})
	}
	for i := 0; i <= 24; i++ {
		a := 3.14159265 + (3*3.14159265/2-3.14159265)*float64(i)/float64(24)
		pts = append(pts, graphics.Point{X: 106 + 4*math.Cos(a), Y: 106 + 4*math.Sin(a)})
	}
	c3.FillPath(pts, graphics.Color{R: 88, G: 166, B: 255, A: 255}, false)
	fmt.Print("luneVerify y=100-105: ")
	for yy := 100; yy <= 105; yy++ {
		fmt.Printf("y=%d:[", yy)
		for xx := 98; xx <= 112; xx++ {
			p := c3.PixelAt(xx, yy)
			if p.A > 200 && p.B > 200 {
				fmt.Print("B")
			} else if p.A > 60 {
				fmt.Print("b")
			} else {
				fmt.Print(".")
			}
		}
		fmt.Print("]")
	}
	fmt.Println()
	// PaintBorder 后 y=100-105 行（x=98-112，与 luneVerify 对齐）
	fmt.Print("paintBorderRows y=100-105: ")
	for yy := 100; yy <= 105; yy++ {
		fmt.Printf("y=%d:[", yy)
		for xx := 98; xx <= 112; xx++ {
			p := canvas.PixelAt(xx, yy)
			if p.A > 200 && p.B > 200 {
				fmt.Print("B")
			} else if p.A > 60 {
				fmt.Print("b")
			} else if p.A > 0 {
				fmt.Print("x")
			} else {
				fmt.Print(".")
			}
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
