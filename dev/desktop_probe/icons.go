// Command desktop_icons checks whether SVG icons actually paint (nonzero
// pixels inside the activity bar icon area) in the desktop Vue app.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/rendering"
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
	log.SetFlags(0)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, _ := os.ReadFile(distDir + "/index.html")

	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.LoadHTML(string(htmlData))
	wv.EnsureLayout()
	rv := wv.RenderView()

	// Render to a raw pixel buffer via the render view.
	rendering.Paint(rv, nil, rendering.Rect{}) // no-op; use WebView.Render instead
	pngBytes, err := wv.Render()
	if err != nil {
		log.Fatal(err)
	}
	w, h := wv.Width(), wv.Height()
	_ = h
	px := func(x, y int) (uint8, uint8, uint8, uint8) {
		off := (y*w + x) * 4
		if off+3 < len(pngBytes) {
			return pngBytes[off], pngBytes[off+1], pngBytes[off+2], pngBytes[off+3]
		}
		return 0, 0, 0, 0
	}
	// Activity bar icons: x=0..48, first icon around y=38..66 (icon 18px).
	fmt.Println("=== ACTIVITY BAR ICON PIXELS (x=0..48, y=38..66) ===")
	ink := 0
	for y := 38; y <= 66; y++ {
		for x := 8; x <= 40; x++ {
			r, g, b, a := px(x, y)
			if a > 0 && !(r > 200 && g > 200 && b > 200) {
				ink++
			}
		}
	}
	fmt.Printf("icon ink pixels: %d (want > 20 for a visible 18x18 icon)\n", ink)
}
