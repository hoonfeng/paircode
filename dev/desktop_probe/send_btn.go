// Command send_btn compares the send button (981,734 44x28) between Edge and
// wb-ui with an ASCII map, revealing what's wrong (icon missing? geometry?).
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

	// send-btn area
	fmt.Println("=== send-btn (981,734)-(1025,762) E=both 1=edge 2=wb ===")
	for y := 734; y < 762; y++ {
		row := ""
		for x := 981; x < 1025; x++ {
			er, eg, eb := px(edge, x, y)
			wr, wg, wb2 := px(wb, x, y)
			eInk := abs(er-0x0d)+abs(eg-0x11)+abs(eb-0x17) > 40 || abs(er-0x16)+abs(eg-0x1b)+abs(eb-0x22) > 40
			wInk := abs(wr-0x0d)+abs(wg-0x11)+abs(wb2-0x17) > 40 || abs(wr-0x16)+abs(wg-0x1b)+abs(wb2-0x22) > 40
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
	// Sample actual colors in the button
	fmt.Println("\n=== sample colors (edge / wb-ui) ===")
	for _, p := range [][2]int{{990, 740}, {1000, 748}, {1010, 750}, {995, 755}, {1005, 738}, {985, 745}} {
		x, y := p[0], p[1]
		er, eg, eb := px(edge, x, y)
		wr, wg, wb2 := px(wb, x, y)
		fmt.Printf("(%3d,%3d) edge=#%02x%02x%02x wb=#%02x%02x%02x\n", x, y, er, eg, eb, wr, wg, wb2)
	}
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
