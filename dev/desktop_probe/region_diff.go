// Command region_diff crops the same region from the Edge reference and the
// wb-ui render and prints a compact pixel summary + ASCII map of differences,
// pinpointing where rendering diverges.
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
	edgePath := wd + "\\dev\\desktop_probe\\cmp_edge.png"
	wbPath := wd + "\\dev\\desktop_probe\\cmp_wbui.png"

	edge := load(edgePath)
	wb := load(wbPath)

	// Region under investigation: activity bar + sidebar top (x=0..140, y=30..140)
	regions := []struct {
		name          string
		x0, y0, x1, y1 int
	}{
		{"activity-bar+sidebar-top", 0, 30, 140, 140},
		{"sidebar-mid", 48, 140, 330, 400},
		{"status-bar", 0, 778, 1280, 800},
		{"conv-sidebar", 1024, 60, 1280, 400},
		{"chat-input", 429, 580, 1030, 780},
	}
	for _, r := range regions {
		diff, total := regionDiff(edge, wb, r.x0, r.y0, r.x1, r.y1)
		fmt.Printf("[%s] region=(%d,%d)-(%d,%d) diff=%d/%d (%.0f%%)\n",
			r.name, r.x0, r.y0, r.x1, r.y1, diff, total, float64(diff)/float64(total)*100)
	}
	// ASCII map for the activity bar region.
	printAsciiMap(edge, wb, 0, 30, 140, 140)
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

func regionDiff(a, b image.Image, x0, y0, x1, y1 int) (int, int) {
	diff, total := 0, 0
	for y := y0; y < y1 && y < a.Bounds().Dy(); y++ {
		for x := x0; x < x1 && x < a.Bounds().Dx(); x++ {
			total++
			ar, ag, ab := px(a, x, y)
			br, bg, bb := px(b, x, y)
			d := abs(ar-br) + abs(ag-bg) + abs(ab-bb)
			if d/3 > 32 {
				diff++
			}
		}
	}
	return diff, total
}

func printAsciiMap(a, b image.Image, x0, y0, x1, y1 int) {
	fmt.Printf("=== ASCII diff map (X=diff, .=same) %d x %d ===\n", x1-x0, y1-y0)
	for y := y0; y < y1; y += 2 {
		row := ""
		for x := x0; x < x1; x += 2 {
			ar, ag, ab := px(a, x, y)
			br, bg, bb := px(b, x, y)
			d := (abs(ar-br) + abs(ag-bg) + abs(ab-bb)) / 3
			if d > 32 {
				row += "X"
			} else {
				row += "."
			}
		}
		fmt.Println(row)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
