// Command input_border_probe renders interact_test.html, focuses the single-line
// input, and pixel-scans its border + focus outline:
//  1. border follows border-radius (rounded, not sharp corners)
//  2. focus outline (2px accent2) is rounded too
//  3. no double border (two nested strokes)
package main

import (
	"fmt"
	"os"

	"wb-ui/dom"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

const htmlPath = "cmd/desktop/web-ui/interact_test.html"

func boxOf(rv *rendering.RenderView, el *dom.Element) (x, y, w, h float64, ok bool) {
	var walk func(ro rendering.RenderObject) bool
	walk = func(ro rendering.RenderObject) bool {
		if ro == nil {
			return false
		}
		if ro.Node() == el {
			x, y, w, h, ok = rendering.BoxGeometry(ro)
			return ok
		}
		for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
			if walk(c) {
				return true
			}
		}
		return false
	}
	walk(rendering.RenderObject(rv))
	return
}

func px(c *graphics.Canvas, x, y int) (uint8, uint8, uint8) {
	p := c.PixelAt(x, y)
	return p.R, p.G, p.B
}

func main() {
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		fmt.Println("read:", err)
		os.Exit(1)
	}
	wv := webkit.NewWebView()
	if err := wv.LoadHTML(string(data)); err != nil {
		fmt.Println("LoadHTML:", err)
		os.Exit(1)
	}
	fr := wv.MainFrame().Frame()
	wv.EnsureLayout()
	doc := wv.Document()

	var inp *dom.Element
	for _, el := range doc.GetElementsByTagName("input") {
		if el.GetAttribute("id") == "input-single" {
			inp = el
			break
		}
	}
	inp.SetFocused(true) // mouse-style focus; author :focus outline must match
	fr.MarkRenderTreeDirty()
	fr.RebuildRenderTree()
	wv.EnsureLayout()

	c := graphics.NewCanvas(1280, 940)
	wv.RenderView().MarkAllDirty()
	rendering.Paint(wv.RenderView(), c, rendering.Rect{0, 0, 1280, 940})

	x, y, w, h, ok := boxOf(wv.RenderView(), inp)
	if !ok {
		fmt.Println("input box not found")
		os.Exit(1)
	}
	cs := fr.Resolver().ResolveElement(inp)
	fmt.Printf("style: BorderRadius=%.1f OutlineSet=%v OutlineW=%.1f border=%s/%s/%s/%s\n",
		cs.BorderRadius.Value, cs.OutlineSet, cs.OutlineWidth.Value,
		cs.BorderTopWidth.String(), cs.BorderRightWidth.String(), cs.BorderBottomWidth.String(), cs.BorderLeftWidth.String())
	ix, iy, iw, ih := int(x), int(y), int(w), int(h)
	fmt.Printf("input box: (%d,%d) %dx%d radius=4 expected\n", ix, iy, iw, ih)

	// Scan the top edge and corner region.
	fmt.Printf("corner pixels: ")
	for dy := 0; dy <= 4; dy++ {
		r, g, b := px(c, ix+dy, iy-2+dy)
		fmt.Printf("(%d,%d)=#%02x%02x%02x ", ix+dy, iy-2+dy, r, g, b)
	}
	fmt.Println()

	// Top edge midpoint: border pixel row. Count rows of border color above box.
	borderRows := 0
	for dy := 1; dy <= 6; dy++ {
		r, g, b := px(c, ix+iw/2, iy-dy)
		if int(r)+int(g)+int(b) < 700 { // not near-white page bg
			borderRows++
		}
	}
	fmt.Printf("top edge above box: %d non-bg rows (1=border only, 2+=double/outline)\n", borderRows)

	// Check the border corner (inside rounded corner should NOT have border color
	// at the extreme corner pixel).
	r0, g0, b0 := px(c, ix+1, iy+1)
	fmt.Printf("corner inside (ix+1,iy+1)=#%02x%02x%02x (should be bg or text, not border)\n", r0, g0, b0)

	fmt.Println("=== input_border_probe done ===")
}
