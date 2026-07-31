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
	distDir := filepath.Join(wd, "cmd", "desktop", "web-ui", "dist")
	htmlData, err := os.ReadFile(distDir + "/index.html")
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}
	log.Printf("HTML: %d bytes, distDir=%s", len(htmlData), distDir)

	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	wv.Resize(1280, 800)
	if err := wv.LoadHTML(string(htmlData)); err != nil {
		log.Printf("LoadHTML err: %v", err)
	}
	if out := wv.ConsoleOutput(); out != "" {
		log.Printf("[CONSOLE] %s", out)
	}
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
