// Command matrix_probe 验证 DrawText 各种场景。
package main

import (
	"fmt"

	"wb-ui/platform/graphics"
)

func scanInk(c *graphics.Canvas, x0, x1, y0, y1 int, label string) {
	first := -1
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			col := c.PixelAt(x, y)
			if col.R < 200 || col.G < 200 || col.B < 200 {
				if first < 0 {
					first = y
				}
				fmt.Printf("%s ink at (%d,%d) #%02x%02x%02x\n", label, x, y, col.R, col.G, col.B)
				goto nextrow
			}
		}
	nextrow:
	}
	if first < 0 {
		fmt.Printf("%s: no ink in (%d..%d, %d..%d)\n", label, x0, x1, y0, y1)
	}
}

func main() {
	font := graphics.Font{Family: "Arial", Size: 14, Weight: 400, Style: "normal"}

	// A: 无 translate 无 clip
	c := graphics.NewCanvas(400, 300)
	c.Clear(graphics.Color{R: 255, G: 255, B: 255, A: 255})
	c.SetFillColor(graphics.Color{R: 0, G: 0, B: 0, A: 255})
	c.DrawText(10, 20, "HELLO", font, c.FillColor())
	scanInk(c, 0, 200, 0, 60, "A(plain)")

	// B: 无 clip 有 translate
	c2 := graphics.NewCanvas(400, 300)
	c2.Clear(graphics.Color{R: 255, G: 255, B: 255, A: 255})
	c2.Save()
	c2.Translate(-5, -5)
	c2.SetFillColor(graphics.Color{R: 0, G: 0, B: 0, A: 255})
	c2.DrawText(10, 20, "HELLO", font, c2.FillColor())
	c2.Restore()
	scanInk(c2, 0, 200, 0, 60, "B(translate)")

	// C: 有 clip 有 translate（模拟渲染管线）
	c3 := graphics.NewCanvas(400, 300)
	c3.Clear(graphics.Color{R: 255, G: 255, B: 255, A: 255})
	c3.Save()
	c3.Clip(graphics.Rect{X: 10, Y: 110, Width: 200, Height: 20})
	c3.Save()
	c3.Translate(-40, -40)
	c3.SetFillColor(graphics.Color{R: 0, G: 0, B: 0, A: 255})
	c3.DrawText(10, 166, "LINE2-1234567890abcdef", font, c3.FillColor())
	c3.Restore()
	c3.Restore()
	scanInk(c3, 0, 300, 0, 200, "C(clip+translate)")
}
