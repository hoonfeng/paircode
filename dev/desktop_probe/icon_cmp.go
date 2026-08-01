// Command icon_cmp compares the activity-bar icon strip (x=0..48, y=30..260)
// between Edge and wb-ui, printing an ASCII map of where icons differ and
// sampling icon ink pixels (nonzero, non-background) per button.
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

	// Activity bar background #0d1117; icon ink = pixels significantly different.
	fmt.Println("=== ACTIVITY BAR ICON STRIP: Edge vs wb-ui ===")
	for btn := 0; btn < 6; btn++ {
		y0 := 34 + btn*42
		y1 := y0 + 40
		eInk, wInk := 0, 0
		// Count icon ink pixels (differs from bg #0d1117 by >40).
		for y := y0; y < y1; y++ {
			for x := 0; x < 48; x++ {
				er, eg, eb := px(edge, x, y)
				wr, wg, wb2 := px(wb, x, y)
				if bgDiff(er, eg, eb) > 40 {
					eInk++
				}
				if bgDiff(wr, wg, wb2) > 40 {
					wInk++
				}
			}
		}
		fmt.Printf("button[%d] y=%d..%d: edge_ink=%d wb_ink=%d\n", btn, y0, y1, eInk, wInk)
	}
	// ASCII map of first button icon (x=0..48, y=34..74): E=both, 1=edge only, 2=wb only, .=bg
	fmt.Println("\n=== first icon map (E=both 1=edge 2=wb . =bg) ===")
	for y := 34; y < 74; y += 2 {
		row := ""
		for x := 0; x < 48; x++ {
			er, eg, eb := px(edge, x, y)
			wr, wg, wb2 := px(wb, x, y)
			e := bgDiff(er, eg, eb) > 40
			w := bgDiff(wr, wg, wb2) > 40
			switch {
			case e && w:
				row += "E"
			case e:
				row += "1"
			case w:
				row += "2"
			default:
				row += "."
			}
		}
		fmt.Println(row)
	}
}

func bgDiff(r, g, b int) int {
	return abs(r-0x0d) + abs(g-0x11) + abs(b-0x17)
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
