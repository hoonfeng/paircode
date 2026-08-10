// Command xtermcell_render_probe renders ide_ref_xtermcell.html through
// wb-ui's engine at 400x80 (same size as Edge's screenshot), so the text
// baseline inside xterm-style absolute spans can be pixel-compared.
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	htmlPath := filepath.Join(wd, "cmd", "companion", "web-ui", "ide_ref_xtermcell.html")
	htmlData, err := os.ReadFile(htmlPath)
	if err != nil {
		log.Fatalf("read html: %v", err)
	}
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style}, text)
	}
	layout.FontMetricsFunc = func(family string, size float64, weight int, style string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	wv := webkit.NewWebView()
	wv.Resize(400, 80)
	if err := wv.LoadHTML(string(htmlData)); err != nil {
		log.Printf("LoadHTML err: %v", err)
	}
	if out := wv.ConsoleOutput(); out != "" {
		log.Printf("[CONSOLE] %s", out)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	pngBytes, err := wv.Render()
	if err != nil {
		log.Fatalf("Render: %v", err)
	}
	w, h := wv.Width(), wv.Height()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := (y*w + x) * 4
			if off+3 < len(pngBytes) {
				img.SetRGBA(x, y, color.RGBA{R: pngBytes[off], G: pngBytes[off+1], B: pngBytes[off+2], A: pngBytes[off+3]})
			}
		}
	}
	out := filepath.Join(wd, "dev", "desktop_probe", "xtermcell_wbui_shot.png")
	f, err := os.Create(out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
	log.Printf("rendered %dx%d → %s", w, h, out)
}
