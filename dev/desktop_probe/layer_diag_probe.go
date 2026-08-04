// Command layer_diag_probe: prints the render-layer tree around conv-list,
// each layer's owner class, CalculateRects() clip, and the box scroll offset —
// BEFORE and AFTER SetBoxScrollOffset(300). Used to diagnose why scrolled
// conv-item / conv-title child layers end up clipped at the wrong position.
package main

import (
	"fmt"
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

func clsOf(el *dom.Element) string {
	if el == nil {
		return ""
	}
	return el.GetAttribute("class")
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

	// Find the conv-list render box.
	var list *rendering.RenderBox
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil || list != nil {
			return
		}
		if rb := asRB(o); rb != nil && rb.Style() != nil {
			if n := rb.Node(); n != nil {
				if el, ok := n.(*dom.Element); ok && clsOf(el) == "conv-list" {
					list = rb
					return
				}
			}
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

	dump := func(tag string) {
		root := rv.RootLayer()
		fmt.Printf("\n===== %s =====\n", tag)
		// Find layer owned by conv-list box, then print its subtree.
		var printLayer func(l *rendering.RenderLayer, depth int)
		printLayer = func(l *rendering.RenderLayer, depth int) {
			name := "?"
			if l.Owner() != nil {
				if n := l.Owner().Node(); n != nil {
					if el, ok := n.(*dom.Element); ok && el != nil {
						name = el.LocalName() + "." + clsOf(el)
					} else {
						name = fmt.Sprintf("%T", l.Owner())
					}
				}
			}
			_, clip := l.CalculateRects()
			box := asRB(l.Owner())
			sx, sy := 0.0, 0.0
			if box != nil {
				sx, sy = rv.BoxScrollOffset(box)
			}
			fmt.Printf("%s%s layer owner=%s box=%.0f,%.0f %.0fx%.0f clip=%.0f,%.0f %.0fx%.0f scroll=(%.0f,%.0f)\n",
				strings.Repeat("  ", depth), "▸", name,
				box.X(), box.Y(), box.Width(), box.Height(),
				clip.X, clip.Y, clip.Width, clip.Height, sx, sy)
			for c := l.FirstChild(); c != nil; c = c.NextSibling() {
				printLayer(c, depth+1)
			}
		}
		// locate conv-list layer
		var findLayer func(l *rendering.RenderLayer) *rendering.RenderLayer
		findLayer = func(l *rendering.RenderLayer) *rendering.RenderLayer {
			if l == nil || l.Owner() == nil {
				return nil
			}
			if n := l.Owner().Node(); n != nil {
				if el, ok := n.(*dom.Element); ok && el != nil && clsOf(el) == "conv-list" {
					return l
				}
			}
			for c := l.FirstChild(); c != nil; c = c.NextSibling() {
				if r := findLayer(c); r != nil {
					return r
				}
			}
			return nil
		}
		cl := findLayer(root)
		if cl == nil {
			fmt.Println("conv-list layer NOT FOUND")
			return
		}
		printLayer(cl, 0)
	}

	dump("before scroll")
	rv.SetBoxScrollOffset(list, 0, 300)
	dump("after scroll 300")
}
