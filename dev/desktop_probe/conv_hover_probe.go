// Command conv_hover_probe loads the REAL companion frontend through wb-ui,
// finds the ACTIVE conv-item, simulates :hover on it, and dumps:
//   - conv-item computed bg before/after (blue -> hover grey flash check)
//   - conv-title geometry before/after (flex shrink when .conv-del appears)
//   - conv-title RenderText segments (char count/width) before/after
//   - conv-del computed display before/after
//   - PNG screenshots before/after for pixel-level text overflow check
//
// Purpose: verify the reported desktop issue:
//   "默认收缩状态下文本向右超出收缩矩形 + 收缩矩形蓝色闪烁"
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

func convHoverMain() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	absDist, _ := filepath.Abs(distDir)
	htmlData, err := os.ReadFile(filepath.Join(absDist, "index.html"))
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}
	log.Printf("[convhover] distDir=%s html=%d bytes", distDir, len(htmlData))

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
	convHoverSetupLoaders(wv, distDir)
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

	// 1) before: 无 hover
	saveConvPNG(wv, filepath.Join("dev", "desktop_probe", "conv_hover_before.png"))
	fmt.Println("=== BEFORE (no hover) ===")
	dumpConvItems(rv, state)

	// 2) simulate hover on the ACTIVE conv-item
	target := findConvItem(rv, true)
	if target == nil {
		log.Fatal("active conv-item not found")
	}
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

	saveConvPNG(wv, filepath.Join("dev", "desktop_probe", "conv_hover_after.png"))
	fmt.Println("=== AFTER (hover active conv-item) ===")
	dumpConvItems(rv, state)

	// 3) also hover a NON-active conv-item for comparison
	target2 := findConvItem(rv, false)
	if target2 != nil {
		target2.SetHovered(true)
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
		saveConvPNG(wv, filepath.Join("dev", "desktop_probe", "conv_hover_nonactive.png"))
		fmt.Println("=== AFTER (hover non-active conv-item) ===")
		dumpConvItems(rv, state)
	}
}

func findConvItem(rv *rendering.RenderView, active bool) *dom.Element {
	var found *dom.Element
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil || found != nil {
			return
		}
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			if strings.Contains(cn, "conv-item") {
				isActive := strings.Contains(cn, "active")
				if isActive == active {
					found = el
					return
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	return found
}

func dumpConvItems(rv *rendering.RenderView, state *layout.LayoutState) {
	var walk func(o rendering.RenderObject, d int)
	walk = func(o rendering.RenderObject, d int) {
		if o == nil {
			return
		}
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			if strings.Contains(cn, "conv-item") || strings.Contains(cn, "conv-title") ||
				strings.Contains(cn, "conv-meta") || strings.Contains(cn, "conv-del") ||
				strings.Contains(cn, "conv-running") || strings.Contains(cn, "conv-msg-count") ||
				strings.Contains(cn, "conv-time") {
				st := o.Style()
				bg := "none"
				if st != nil {
					bg = hexColor2(st.BackgroundColor)
				}
				disp := "?"
				if st != nil {
					disp = fmt.Sprintf("%d", st.Display)
				}
				var lb string
				if o.LayoutBox() != nil && state != nil {
					g := state.GeometryForBox(o.LayoutBox())
					lb = fmt.Sprintf("x=%.0f y=%.0f w=%.0f h=%.0f", g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight())
				} else {
					lb = "[no layout]"
				}
				var cb string
				if rb := asRenderBox2(o); rb != nil {
					cr := rb.ContentBoxRect()
					cb = fmt.Sprintf(" frameCB(x=%.0f w=%.0f)", cr.X, cr.Width)
				}
				fmt.Printf("  <%s.%s> %s%s bg=%s disp=%s hovered=%v\n",
					el.LocalName(), cn, lb, cb, bg, disp, el.IsHovered())
				if rt, ok := o.(*rendering.RenderText); ok {
					for i, seg := range rt.Segments() {
						segText := ""
						if seg.Start+seg.Len <= len(rt.Text()) {
							segText = rt.Text()[seg.Start : seg.Start+seg.Len]
						}
						fmt.Printf("      seg[%d] text=%q X=%.1f W=%.1f H=%.1f\n", i, segText, seg.X, seg.Width, seg.Height)
					}
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, d+1)
		}
	}
	walk(rendering.RenderObject(rv), 0)
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
		fmt.Printf("[convhover] saved %s\n", path)
	}
}

func convHoverSetupLoaders(wv *webkit.WebView, distDir string) {
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

func hexColor2(c style.Color) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func asRenderBox2(o rendering.RenderObject) *rendering.RenderBox {
	switch v := o.(type) {
	case *rendering.RenderBox:
		return v
	case *rendering.RenderBlock:
		return &v.RenderBox
	case *rendering.RenderBlockFlow:
		return &v.RenderBlock.RenderBox
	case *rendering.RenderView:
		return &v.RenderBlockFlow.RenderBlock.RenderBox
	}
	return nil
}

func main() {
	convHoverMain()
}
