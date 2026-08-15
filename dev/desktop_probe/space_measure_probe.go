//go:build ignore

package main

import (
	"fmt"

	"wb-ui/platform/graphics"
)

func main() {
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	f := graphics.Font{Family: "Consolas", Size: 13, Weight: 400, Style: "normal"}
	for _, s := range []string{" ", "A", "W", "i", "中", "  ", "A A"} {
		fmt.Printf("measureText(%q) = %.4f\n", s, graphics.MeasureText(f, s))
	}
	// 字体 metrics
	ascent := graphics.GlobalFontAscent(f)
	descent := graphics.GlobalFontDescent(f)
	fmt.Printf("ascent=%.4f descent=%.4f\n", ascent, descent)
}
