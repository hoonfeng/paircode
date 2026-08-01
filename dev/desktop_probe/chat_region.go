// Command chat_region diffs sub-regions of the chat-input area between Edge
// and wb-ui to locate component rendering anomalies.
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

	regions := []struct {
		name          string
		x0, y0, x1, y1 int
	}{
		{"chat-input-area bg", 429, 582, 1030, 778},
		{"input-wrapper", 437, 582, 1022, 771},
		{"textarea", 438, 583, 1021, 733},
		{"input-bottom-bar", 438, 733, 1021, 770},
		{"ibb-btns", 450, 736, 636, 760},
		{"send-btn", 981, 734, 1025, 762},
		{"conv-footer-btns", 1039, 710, 1270, 770},
	}
	for _, r := range regions {
		d, total := regionDiff(edge, wb, r.x0, r.y0, r.x1, r.y1)
		fmt.Printf("[%s] (%d,%d)-(%d,%d) diff=%d/%d (%.0f%%)\n",
			r.name, r.x0, r.y0, r.x1, r.y1, d, total, float64(d)/float64(total)*100)
	}
}

func regionDiff(a, b image.Image, x0, y0, x1, y1 int) (int, int) {
	diff, total := 0, 0
	for y := y0; y < y1 && y < a.Bounds().Dy(); y++ {
		for x := x0; x < x1 && x < a.Bounds().Dx(); x++ {
			total++
			ar, ag, ab := px(a, x, y)
			br, bg, bb := px(b, x, y)
			if (abs(ar-br)+abs(ag-bg)+abs(ab-bb))/3 > 32 {
				diff++
			}
		}
	}
	return diff, total
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
