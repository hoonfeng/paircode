// Command scroll_paint_probe loads the REAL companion frontend through wb-ui,
// paints it WITHOUT scroll, then sets the box scroll offset of every scroll
// container to its max and paints again, saving both frames as PNGs. Used to
// verify at the pixel level that "content below the fold is reachable by
// scrolling" — if the bottom frame still shows the same top content (or the
// file tree tail never appears), the scroll offset is not applied at paint.
package main

import (
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
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

func asRB(o rendering.RenderObject) *rendering.RenderBox {
	switch v := o.(type) {
	case *rendering.RenderBox:
		return v
	case *rendering.RenderBlock:
		return &v.RenderBox
	case *rendering.RenderBlockFlow:
		return &v.RenderBlock.RenderBox
	}
	return nil
}

func clsOf(rb *rendering.RenderBox) string {
	if el, ok := rb.Node().(*dom.Element); ok {
		return el.GetAttribute("class")
	}
	return ""
}

func saveCanvas(c *graphics.Canvas, path string) {
	w, h := c.Width(), c.Height()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	px := c.Pixels()
	for i := 0; i < w*h; i++ {
		img.Pix[i*4+0] = px[i*4+0]
		img.Pix[i*4+1] = px[i*4+1]
		img.Pix[i*4+2] = px[i*4+2]
		img.Pix[i*4+3] = px[i*4+3]
	}
	f, err := os.Create(path)
	if err != nil {
		log.Printf("save %s: %v", path, err)
		return
	}
	defer f.Close()
	png.Encode(f, img)
	log.Printf("saved %s (%dx%d)", path, w, h)
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, err := os.ReadFile(filepath.Join(distDir, "index.html"))
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}

	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style2, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}, text)
	}
	layout.FontMetricsFunc = func(family string, size float64, weight int, style2 string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	for i := 0; i < 8; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
	}
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 800); })`)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}

	// Find scroll containers of interest.
	type target struct {
		name string
		rb   *rendering.RenderBox
		maxY float64
	}
	var targets []*target
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil {
			return
		}
		if rb := asRB(o); rb != nil && rb.Style() != nil {
			st := rb.Style()
			if st.OverflowY == style.OverflowAuto || st.OverflowY == style.OverflowScroll {
				cn := clsOf(rb)
				if cn == "project-section" || cn == "chat-messages" || cn == "conv-list" || cn == "ws-section" || cn == "sidebar-content" {
					m := rendering.VerticalScrollbarMetrics(rv, rb)
					maxY := float64(0)
					if m.OK {
						maxY = m.MaxScroll
					}
					targets = append(targets, &target{name: cn, rb: rb, maxY: maxY})
					log.Printf("target %s view=%.0f total=%.0f maxY=%.0f", cn, m.ViewLen, m.TotalLen, maxY)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))

	outDir := filepath.Join(wd, "dev", "desktop_probe")

	// Frame 1: no scroll.
	c1 := graphics.NewCanvas(1280, 800)
	rendering.Paint(rv, c1, rendering.Rect{X: 0, Y: 0, Width: 1280, Height: 800})
	saveCanvas(c1, filepath.Join(outDir, "sp_top.png"))

	// Frame 2: each scroll container scrolled to its max.
	for _, t := range targets {
		rv.SetBoxScrollOffset(t.rb, 0, t.maxY)
	}
	c2 := graphics.NewCanvas(1280, 800)
	rendering.Paint(rv, c2, rendering.Rect{X: 0, Y: 0, Width: 1280, Height: 800})
	saveCanvas(c2, filepath.Join(outDir, "sp_bottom.png"))

	// Ink-diff in the file-tree area (x=48..326, y=264..778) and chat area (x=429..1030, y=67..582).
	diffRegion := func(name string, x0, y0, x1, y1 int) {
		diffs := 0
		total := 0
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				i := (y*1280 + x) * 4
				d := 0
				for k := 0; k < 3; k++ {
					v := int(c1.Pixels()[i+k]) - int(c2.Pixels()[i+k])
					if v < 0 {
						v = -v
					}
					d += v
				}
				if d > 60 {
					diffs++
				}
				total++
			}
		}
		log.Printf("diff %s: %d/%d pixels changed (%.1f%%)", name, diffs, total, float64(diffs)*100/float64(total))
	}
	diffRegion("filetree", 48, 264, 326, 778)
	diffRegion("chat", 429, 67, 1030, 582)
	diffRegion("convlist", 1031, 98, 1280, 402)
}
