// Command paren_probe directly renders Consolas 13px glyphs through goskia
// to quantify the ink height of '(' 'x' 'a' '{' 'n' ')' — verifying whether
// parens render full-height (12px) vs x-height (8px) like in browsers.
// Run: go run ./dev/desktop_probe/paren_probe.go
//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hoonfeng/goskia/skia"
)

func main() {
	wd, _ := os.Getwd()
	outDir := filepath.Join(wd, "dev", "desktop_probe")

	tf := skia.NewTypeface("Consolas", skia.FontStyle{Weight: 400, Width: 5, Slant: 0})
	if tf == nil {
		fmt.Println("CONSOLAS TF = NIL")
		os.Exit(1)
	}
	font := skia.NewFont(tf, 13)

	// Render each character on its own 24x22 tile, 7 chars across one row.
	const tileW, tileH = 24, 22
	chars := []string{"(", ")", "{", "}", "x", "a", "n", "f", "0", "I"}
	for i, ch := range chars {
		surf, err := skia.NewRasterSurfaceN32Premul(tileW, tileH)
		if err != nil {
			fmt.Printf("%s: surface err %v\n", ch, err)
			continue
		}
		canvas := surf.Canvas()
		paint := skia.NewPaint()
		paint.SetColor(skia.ColorWhite)
		canvas.DrawRect(skia.Rect{Left: 0, Top: 0, Right: tileW, Bottom: tileH}, paint) // white bg
		paint.SetColor(skia.ColorBlack)
		// baseline at y=18 (cap top ~y=7 for 13px Consolas)
		canvas.DrawText(ch, 4, 18, font, paint)
		png, err := surf.EncodePNG()
		if err != nil {
			fmt.Printf("%s: encode err %v\n", ch, err)
			continue
		}
		path := filepath.Join(outDir, fmt.Sprintf("paren_tile_%d.png", i))
		if err := os.WriteFile(path, png, 0o644); err != nil {
			fmt.Printf("%s: write err %v\n", ch, err)
			continue
		}
		fmt.Printf("TILE %s -> %s\n", ch, path)
	}
	fmt.Println("DONE")
}
