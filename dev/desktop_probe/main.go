// Command desktop_probe loads the desktop IDE (web-ui/dist/index.html — a Vue
// SPA) through wb-ui's WebView exactly like cmd/desktop does, executes the
// scripts, and writes a render PNG for visual inspection.
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

func setupLoaders(wv *webkit.WebView, distDir string) {
	absDist, _ := filepath.Abs(distDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				return string(data), err
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				if err != nil {
					return "", err
				}
				re := regexp.MustCompile(`\[data-v-[a-f0-9]+\]`)
				return re.ReplaceAllString(string(data), ""), nil
			}
		}
	}
}

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, err := os.ReadFile(distDir + "/index.html")
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}
	log.Printf("HTML: %d bytes, distDir=%s", len(htmlData), distDir)

	// Mirror host.go: initialize the FontManager (Microsoft YaHei default)
	// so CJK measurement matches painting. Without it, skia falls back to a
	// bare Arial whose glyph coverage differs per character, producing
	// inconsistent widths (e.g. 会话 12px/char but 创建 5px/char).
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
	setupLoaders(wv, distDir)
	wv.Resize(1280, 800)
	if err := wv.LoadHTML(string(htmlData)); err != nil {
		log.Printf("LoadHTML err: %v", err)
	}
	if out := wv.ConsoleOutput(); out != "" {
		log.Printf("[CONSOLE] %s", out)
	}
	// Mirror host.go: after JS mounts Vue, rebuild so runtime styles (Vue
	// scoped CSS from the dist stylesheet) are collected and applied.
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	log.Printf("[PROBE] after LoadHTML: document=%v renderView=%v", wv.Document() != nil, wv.RenderView() != nil)

	// Count DOM elements to see whether Vue mounted.
	domCount := 0
	if doc := wv.Document(); doc != nil {
		var count func(n interface{ FirstChild() interface{} })
		_ = count
		_ = doc
	}
	_ = domCount

	pngBytes, err := wv.Render()
	if err != nil {
		log.Fatalf("Render: %v", err)
	}
	// Render() returns raw RGBA (w*h*4); wrap into a real PNG.
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
	out := filepath.Join(wd, "dev", "desktop_probe", "desktop.png")
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
