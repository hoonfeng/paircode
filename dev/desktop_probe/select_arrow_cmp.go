// Command select_arrow_cmp renders a native <select> (220x20) on white bg
// and dumps arrow pixels with Edge's exact black-core rule (RGB<80).
package main

import (
	"fmt"

	"wb-ui/dom"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
)

func main() {
	doc := dom.NewDocument()
	sel := doc.CreateElement("select")
	opt := doc.CreateElement("option")
	opt.SetAttribute("value", "gpt-4o")
	opt.SetTextContent("gpt-4o")
	sel.AppendChild(opt)
	doc.AppendChild(sel)

	st := style.NewComputedStyle()
	st.Color = style.Color{R: 0, G: 0, B: 0, A: 255}

	canvas := graphics.NewCanvas(220, 20)
	canvas.FillRect(0, 0, 220, 20, graphics.Color{R: 255, G: 255, B: 255, A: 255})

	info := rendering.NewPaintInfo(canvas, rendering.Rect{X: 0, Y: 0, Width: 220, Height: 20})

	box := rendering.NewRenderBox(sel, st)
	box.SetLocation(0, 0)
	box.SetSize(220, 20)
	rendering.PaintFormControl(box, info)

	fmt.Println("=== wb-ui 白底箭头（RGB<80=*, 80..200=o）cx=210.5 cy=10 ===")
	fmt.Println("pts=(207.5,9)→(210.5,12)→(213.5,9) stroke=2 round")
	for yy := 5; yy <= 15; yy++ {
		line := ""
		for xx := 202; xx <= 218; xx++ {
			p := canvas.PixelAt(xx, yy)
			switch {
			case p.R < 80 && p.G < 80 && p.B < 80:
				line += "*"
			case p.R < 220 && p.G < 220 && p.B < 220:
				line += "o"
			default:
				line += "."
			}
		}
		fmt.Printf("y=%2d: %s\n", yy, line)
	}
	fmt.Println("\n=== Edge s1 参照（黑核心 y84..88）===")
	fmt.Println("y=83: .......##....##....#.")
	fmt.Println("y=84: ......#**#..#**#...#.")
	fmt.Println("y=85: ......#***##***#...#.")
	fmt.Println("y=86: .......#******#....#.")
	fmt.Println("y=87: ........#****#.....#.")
	fmt.Println("y=88: .........#**#......#.")
	fmt.Println("y=89: ..........##.......#.")
}
