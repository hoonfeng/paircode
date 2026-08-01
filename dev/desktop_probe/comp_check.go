// Command comp_check renders the desktop app and reports ink coverage for
// specific components: welcome-logo (emoji), send-btn icon, obtn icons,
// select arrow — to spot components that fail to draw.
package main

import (
	"fmt"
	"image"
	"image/color"
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
	log.SetFlags(0)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, _ := os.ReadFile(distDir + "/index.html")

	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.LoadHTML(string(htmlData))
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	raw, err := wv.Render()
	if err != nil {
		log.Fatal(err)
	}
	w, h := wv.Width(), wv.Height()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := (y*w + x) * 4
			if off+3 < len(raw) {
				img.SetRGBA(x, y, colorRGBA(raw[off], raw[off+1], raw[off+2], raw[off+3]))
			}
		}
	}

	// Count ink (non-background) pixels in component boxes (from geo output).
	regions := []struct {
		name          string
		x0, y0, x1, y1 int
	}{
		{"welcome-logo (emoji)", 327, 261, 425, 325},
		{"send-btn icon", 995, 740, 1011, 756},
		{"obtn review-auto", 450, 736, 488, 760},
		{"conv-stats chevron", 1039, 392, 1049, 405},
		{"chat-empty-icon", 730, 153, 762, 185},
	}
	for _, r := range regions {
		ink := 0
		for y := r.y0; y < r.y1; y++ {
			for x := r.x0; x < r.x1; x++ {
				c := img.RGBAAt(x, y)
				if c.A > 0 && !(c.R > 230 && c.G > 230 && c.B > 230) {
					ink++
				}
			}
		}
		fmt.Printf("[%s] ink=%d (region %dx%d)\n", r.name, ink, r.x1-r.x0, r.y1-r.y0)
	}
}

func colorRGBA(r, g, b, a uint8) color.RGBA { return color.RGBA{R: r, G: g, B: b, A: a} }
