// Command conv_layout_probe loads the REAL companion frontend through wb-ui
// (same as real_probe), then dumps geometry + computed styles of the conv
// sidebar tree (conv-sidebar / conv-list / conv-item / conv-title / conv-meta)
// to diagnose:
//   1) text overflowing the collapsed (narrow) sidebar rectangle
//   2) active item blue highlight flicker while mouse moves
//
// It also simulates hover on items (SetHovered + relayout) and dumps the
// active item's background/border before/after each hover to detect
// flicker-prone state changes.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

type Geom struct {
	Left, Top, W, H float64
}

func geoOf(st *layout.LayoutState, o rendering.RenderObject) Geom {
	if o == nil || o.LayoutBox() == nil || st == nil {
		return Geom{}
	}
	g := st.GeometryForBox(o.LayoutBox())
	return Geom{g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight()}
}

func hexBG(c style.Color) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	absDist, _ := filepath.Abs(distDir)
	htmlData, err := os.ReadFile(filepath.Join(absDist, "index.html"))
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
	wv.Resize(1280, 800)
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
	st := rv.LayoutState()

	fmt.Println("======== conv-sidebar tree geometry ========")
	var walk func(o rendering.RenderObject, d int, parentW float64)
	walk = func(o rendering.RenderObject, d int, parentW float64) {
		if o == nil {
			return
		}
		if el, ok := o.Node().(*dom.Element); ok {
			cls := el.GetAttribute("class")
			if cls == "conv-sidebar" || strings.Contains(cls, "conv-list") ||
				strings.Contains(cls, "conv-item") || strings.Contains(cls, "conv-title") ||
				strings.Contains(cls, "conv-meta") || strings.Contains(cls, "conv-del") ||
				strings.Contains(cls, "conv-running") || strings.Contains(cls, "conv-msg-count") ||
				strings.Contains(cls, "conv-time") {
				g := geoOf(st, o)
				oStyle := o.Style()
				var text string
				if tn, ok := o.Node().FirstChild().(*dom.Text); ok {
					text = strings.TrimSpace(tn.Data())
					if len(text) > 40 {
						text = text[:40] + "..."
					}
				}
				ln := ""
				if el2, ok2 := o.Node().(*dom.Element); ok2 {
					ln = el2.LocalName()
				}
				fmt.Printf("%s<%s class=%q x=%.0f y=%.0f w=%.0f h=%.0f bg=%s color=%s overflow=%s ws=%s tw=%s>\n",
					strings.Repeat("  ", d), ln, cls,
					g.Left, g.Top, g.W, g.H,
					hexBG(oStyle.BackgroundColor), hexBG(oStyle.Color),
					oStyle.OverflowX, oStyle.WhiteSpace, text)
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, d+1, parentW)
		}
	}
	walk(rendering.RenderObject(rv), 0, 0)

	// ---- save baseline render PNG ----
	saveConvPNG(wv, filepath.Join("dev", "desktop_probe", "conv_base.png"))

	// ---- simulate hover on each conv-item, dump active item state ----
	fmt.Println("\n======== hover simulation (active item state) ========")
	var items []*dom.Element
	var collect func(o rendering.RenderObject)
	collect = func(o rendering.RenderObject) {
		if o == nil {
			return
		}
		if el, ok := o.Node().(*dom.Element); ok {
			if cn := el.GetAttribute("class"); strings.Contains(cn, "conv-item") {
				items = append(items, el)
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			collect(c)
		}
	}
	collect(rendering.RenderObject(rv))
	fmt.Printf("conv-items found: %d\n", len(items))
	for i, it := range items {
		cls := it.GetAttribute("class")
		_ = cls
		it.SetHovered(true)
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
				fr.SetNeedsLayout(true)
			}
		}
		wv.RebuildRenderTree()
		wv.EnsureLayout()
		rv = wv.RenderView()
		st = rv.LayoutState()
		// dump active item after hover
		for j, it2 := range items {
			cls2 := it2.GetAttribute("class")
			if !strings.Contains(cls2, "active") {
				continue
			}
			ro := findRO(rv, it2)
			if ro != nil {
				g := geoOf(st, ro)
				os := ro.Style()
				fmt.Printf("[hover %d on item%d(%s)] active-item%d x=%.0f y=%.0f w=%.0f h=%.0f bg=%s color=%s\n",
					i+1, i, cls, j, g.Left, g.Top, g.W, g.H, hexBG(os.BackgroundColor), hexBG(os.Color))
			}
		}
		it.SetHovered(false)
	}
	// save PNG with active hovered (item0 hovered)
	for _, it := range items {
		if strings.Contains(it.GetAttribute("class"), "active") {
			it.SetHovered(true)
			break
		}
	}
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.MarkRenderTreeDirty()
			fr.SetNeedsLayout(true)
		}
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	saveConvPNG(wv, filepath.Join("dev", "desktop_probe", "conv_hover_active.png"))
}

func saveConvPNG(wv *webkit.WebView, path string) {
	pngBytes, err := wv.Render()
	if err != nil {
		log.Printf("Render: %v", err)
		return
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
	f, err := os.Create(path)
	if err == nil {
		_ = png.Encode(f, img)
		f.Close()
		fmt.Printf("[conv] saved %s\n", path)
	}
}

func findRO(rv *rendering.RenderView, el *dom.Element) rendering.RenderObject {
	var found rendering.RenderObject
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil || found != nil {
			return
		}
		if o.Node() == dom.Node(el) {
			found = o
			return
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	return found
}

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
				return string(data), nil
			}
		}
	}
}
