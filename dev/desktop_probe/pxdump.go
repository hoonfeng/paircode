// Command pxdump prints pixel colors at key coordinates from both renders,
// revealing background-color mismatches (e.g. activity bar).
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

	pts := [][2]int{
		{10, 60}, {10, 200}, {10, 400}, // activity bar
		{60, 60}, {200, 60}, // titlebar / sidebar
		{330, 400}, {600, 400}, // main
		{1100, 200}, {1200, 100}, // conv-sidebar
		{700, 778}, {100, 790}, // status bar
		{500, 600}, // chat input
	}
	for _, p := range pts {
		x, y := p[0], p[1]
		er, eg, eb := px(edge, x, y)
		wr, wg, wb2 := px(wb, x, y)
		fmt.Printf("(%3d,%3d) edge=#%02x%02x%02x wb-ui=#%02x%02x%02x %s\n",
			x, y, er, eg, eb, wr, wg, wb2, mark(er, eg, eb, wr, wg, wb2))
	}
}

func mark(er, eg, eb, wr, wg, wb int) string {
	d := (abs(er-wr) + abs(eg-wg) + abs(eb-wb)) / 3
	if d > 32 {
		return "⟵ DIFF"
	}
	return "ok"
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
