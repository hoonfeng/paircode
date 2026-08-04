// Command conv_scroll_probe verifies "relative-coordinate drift" of layer
// items inside a scrolled container. conv-list items are position:relative
// (each is a RenderLayer), so they exercise the child-layer paint path with
// the scroll translate. We render conv-list at scroll=0 and scroll=300 and
// dump each conv-title's text + absolute geometry; the Python side then does
// template matching to locate where each title actually landed in the
// scrolled frame and compares against the theoretical Y - 300.
package main

import (
	"bufio"
	"fmt"
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

	// Find conv-list scroll container.
	var list *rendering.RenderBox
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil || list != nil {
			return
		}
		if rb := asRB(o); rb != nil && rb.Style() != nil && clsOf(rb) == "conv-list" {
			list = rb
			return
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	if list == nil {
		log.Fatal("no conv-list")
	}
	log.Printf("conv-list frame=%.0f,%.0f %.0fx%.0f", list.X(), list.Y(), list.Width(), list.Height())

	// Collect conv-title text lines (absolute geometry from first segment).
	type title struct {
		text  string
		segX  float64
		segY  float64
		segW  float64
		cls   string
	}
	var titles []title
	var walkT func(o rendering.RenderObject)
	walkT = func(o rendering.RenderObject) {
		if o == nil {
			return
		}
		if rt, ok := o.(*rendering.RenderText); ok {
			txt := strings.TrimSpace(rt.Text())
			segs := rt.Segments()
			if len(segs) > 0 && txt != "" {
				l := title{text: txt, segX: segs[0].X, segY: segs[0].Y, segW: segs[0].Width}
				if rb := asRB(rt.Parent()); rb != nil {
					l.cls = clsOf(rb)
				}
				titles = append(titles, l)
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walkT(c)
		}
	}
	walkT(rendering.RenderObject(list))

	const scrollY = 300.0
	outDir := filepath.Join(wd, "dev", "desktop_probe")

	c1 := graphics.NewCanvas(1280, 800)
	rendering.Paint(rv, c1, rendering.Rect{X: 0, Y: 0, Width: 1280, Height: 800})
	saveCanvas(c1, filepath.Join(outDir, "conv_top.png"))

	rv.SetBoxScrollOffset(list, 0, scrollY)
	c2 := graphics.NewCanvas(1280, 800)
	rendering.Paint(rv, c2, rendering.Rect{X: 0, Y: 0, Width: 1280, Height: 800})
	saveCanvas(c2, filepath.Join(outDir, "conv_scroll300.png"))

	// Dump titles with geometry for the Python matcher.
	f, _ := os.Create(filepath.Join(outDir, "_conv_titles.txt"))
	w := bufio.NewWriter(f)
	fmt.Fprintf(w, "listY0=%.0f listY1=%.0f scroll=%.0f\n", list.Y(), list.Y()+list.Height(), scrollY)
	for _, t := range titles {
		fmt.Fprintf(w, "TITLE\t%s\t%.1f\t%.1f\t%.1f\t%s\n", t.text, t.segX, t.segY, t.segW, t.cls)
	}
	w.Flush()
	f.Close()
	log.Printf("dumped %d titles", len(titles))
}
