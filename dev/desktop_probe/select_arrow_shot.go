// Command select_arrow_shot renders a native <select> via wb-ui's form-control
// painter into a canvas, then dumps the arrow-region pixels as ASCII so the
// chevron shape can be compared against the Edge reference (edge_select_shot.png).
//go:build ignore

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
	// Build a tiny document: <select><option>gpt-4o</option></select>
	doc := dom.NewDocument()
	sel := doc.CreateElement("select")
	opt := doc.CreateElement("option")
	opt.SetAttribute("value", "gpt-4o")
	opt.SetTextContent("gpt-4o")
	sel.AppendChild(opt)
	doc.AppendChild(sel)

	// Computed style with default text color (black).
	st := style.NewComputedStyle()
	st.Color = style.Color{R: 0, G: 0, B: 0, A: 255}

	canvas := graphics.NewCanvas(120, 20)
	info := rendering.NewPaintInfo(canvas, rendering.Rect{X: 0, Y: 0, Width: 120, Height: 20})

	box := rendering.NewRenderBox(sel, st)
	box.SetLocation(0, 0)
	box.SetSize(120, 20)
	if !rendering.PaintFormControl(box, info) {
		fmt.Println("PaintFormControl returned false")
		os.Exit(1)
	}

	// Dump arrow region ASCII (box 120x20 → chevron cx=110.5, cy=10).
	fmt.Println("=== wb-ui select 箭头区域 (x 102..118, y 3..17) ===")
	for yy := 3; yy <= 17; yy++ {
		line := make([]byte, 0, 17)
		for xx := 102; xx <= 118; xx++ {
			px := canvas.PixelAt(xx, yy)
			switch {
			case px.A == 0:
				line = append(line, '.')
			case px.R < 100 && px.G < 100 && px.B < 100:
				line = append(line, '*')
			default:
				line = append(line, 'o')
			}
		}
		fmt.Printf("%s y=%d\n", string(line), yy)
	}
}
