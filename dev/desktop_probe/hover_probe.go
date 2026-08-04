// Command hover_probe loads the REAL companion frontend through wb-ui (same
// path as real_probe), renders it, then simulates :hover on the first
// comp-bar-seg element and re-renders, dumping:
//   - seg computed background-color before/after hover
//   - element geometry before/after (position drift check)
//   - PNG screenshots before/after for pixel diff
//
// Purpose: verify why hover background does not show on the desktop renderer.
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

func hoverProbeMain() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	absDist, _ := filepath.Abs(distDir)
	htmlData, err := os.ReadFile(filepath.Join(absDist, "index.html"))
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}
	log.Printf("[hover] distDir=%s html=%d bytes", distDir, len(htmlData))

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
	hoverSetupLoaders(wv, distDir)
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
	if rv == nil {
		log.Fatal("no render view")
	}
	state := rv.LayoutState()

	// 1) 无 hover 渲染
	savePNG(wv, filepath.Join("dev", "desktop_probe", "hover_before.png"))
	segsBefore := dumpSegs(rv, state)
	fmt.Printf("[hover] BEFORE:\n%s", segsBefore)

	// 2) 模拟 hover：找第一个【非 active】conv-item（.conv-item:hover 规则不依赖 .active）
	target := findNonActive(rv, "conv-item")
	hoverCls := "conv-item"
	if target == nil {
		target = findClass(rv, "comp-bar-seg")
		hoverCls = "comp-bar-seg"
	}
	if target == nil {
		log.Fatal("hover target not found")
	}
	fmt.Printf("[hover] hover target: <%s class=%q>\n", target.LocalName(), target.GetAttribute("class"))
	dumpOne(rv, state, hoverCls, "TARGET-BEFORE")
	target.SetHovered(true)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.MarkRenderTreeDirty()
			fr.SetNeedsLayout(true)
		}
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv = wv.RenderView()
	state = rv.LayoutState()

	savePNG(wv, filepath.Join("dev", "desktop_probe", "hover_after.png"))
	rv = wv.RenderView()
	state = rv.LayoutState()
	dumpOne(rv, state, hoverCls, "TARGET-AFTER")
	dumpSegs(rv, state)
}

// dumpOne prints computed bg of elements matching cls.
func dumpOne(rv *rendering.RenderView, state *layout.LayoutState, cls, label string) {
	var walk func(o rendering.RenderObject, d int)
	walk = func(o rendering.RenderObject, d int) {
		if o == nil {
			return
		}
		if el, ok := o.Node().(*dom.Element); ok {
			if cn := el.GetAttribute("class"); strings.Contains(cn, cls) {
				if o.LayoutBox() != nil && state != nil {
					g := state.GeometryForBox(o.LayoutBox())
					st := o.Style()
					fmt.Printf("  [%s] class=%q x=%.0f y=%.0f w=%.0f h=%.0f bg=%s\n",
						label, cn, g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(),
						hexColor(st.BackgroundColor))
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, d+1)
		}
	}
	walk(rendering.RenderObject(rv), 0)
}

// dumpSegs prints geometry + computed bg for every comp-bar-seg.
func dumpSegs(rv *rendering.RenderView, state *layout.LayoutState) string {
	var sb strings.Builder
	var walk func(o rendering.RenderObject, d int)
	walk = func(o rendering.RenderObject, d int) {
		if o == nil {
			return
		}
		if el, ok := o.Node().(*dom.Element); ok {
			if cn := el.GetAttribute("class"); strings.Contains(cn, "comp-bar-seg") {
				g := state.GeometryForBox(o.LayoutBox())
				st := o.Style()
				fmt.Fprintf(&sb, "  seg class=%q x=%.0f y=%.0f w=%.0f h=%.0f bg=%s\n",
					cn, g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(),
					hexColor(st.BackgroundColor))
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, d+1)
		}
	}
	walk(rendering.RenderObject(rv), 0)
	return sb.String()
}

func findClass(rv *rendering.RenderView, cls string) *dom.Element {
	var found *dom.Element
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil || found != nil {
			return
		}
		if el, ok := o.Node().(*dom.Element); ok {
			if cn := el.GetAttribute("class"); strings.Contains(cn, cls) {
				found = el
				return
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	return found
}

// findNonActive returns the first element whose class contains cls but NOT "active".
func findNonActive(rv *rendering.RenderView, cls string) *dom.Element {
	var found *dom.Element
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil || found != nil {
			return
		}
		if el, ok := o.Node().(*dom.Element); ok {
			if cn := el.GetAttribute("class"); strings.Contains(cn, cls) && !strings.Contains(cn, "active") {
				found = el
				return
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	return found
}

func savePNG(wv *webkit.WebView, path string) {
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
		fmt.Printf("[hover] saved %s\n", path)
	}
}

func hoverSetupLoaders(wv *webkit.WebView, distDir string) {
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

func hexColor(c style.Color) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func main() {
	hoverProbeMain()
}
