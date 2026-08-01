// Command obtn_region diffs the ibb-btns area (input-bottom-bar buttons) to
// locate what's off: icons (SVG 12x12) or text.
package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	wd, _ := os.Getwd()
	edge := load(wd + "\\dev\\desktop_probe\\cmp_edge.png")
	wb := load(wd + "\\dev\\desktop_probe\\cmp_wbui.png")

	// obtn row: y=736..760, x=450..636. ASCII map.
	fmt.Println("=== ibb-btns (450,736)-(636,760) E=both 1=edge 2=wb ===")
	for y := 736; y < 760; y++ {
		row := ""
		for x := 450; x < 636; x++ {
			er, eg, eb := px(edge, x, y)
			wr, wg, wb2 := px(wb, x, y)
			eInk := colorDiff(er, eg, eb, 0x0d, 0x11, 0x17) > 24
			wInk := colorDiff(wr, wg, wb2, 0x0d, 0x11, 0x17) > 24
			switch {
			case eInk && wInk:
				row += "E"
			case eInk:
				row += "1"
			case wInk:
				row += "2"
			default:
				row += "."
			}
		}
		fmt.Println(row)
	}
	// Sample colors across the row
	fmt.Println("\n=== sample colors ===")
	for _, x := range []int{455, 470, 500, 525, 540, 565, 580, 610} {
		er, eg, eb := px(edge, x, 750)
		wr, wg, wb2 := px(wb, x, 750)
		fmt.Printf("x=%d edge=#%02x%02x%02x wb=#%02x%02x%02x\n", x, er, eg, eb, wr, wg, wb2)
	}
}

func colorDiff(r1, g1, b1, r2, g2, b2 int) int {
	return (abs(r1-r2) + abs(g1-g2) + abs(b1-b2)) / 3
}

func load(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	return img
}

func px(img image.Image, x, y int) (int, int, int) {
	r, g, b, _ := img.At(x, y).RGBA()
	return int(r >> 8), int(g >> 8), int(b >> 8)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
