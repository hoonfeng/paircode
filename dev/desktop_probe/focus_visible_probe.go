// Command focus_visible_probe verifies the :focus-visible semantics:
//  1. mouse focus (SetFocused only) → UA default outline NOT painted
//     (Chrome/Edge: clicking a button does not draw a focus ring)
//  2. keyboard focus (SetFocusByKeyboard(true)) → outline IS painted
//  3. author :focus styles (.textarea:focus outline:2px) still match on
//     mouse focus (user CSS targets :focus, unaffected)
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

func outlinePixels(c *graphics.Canvas, bx, by, bw, bh float64) int {
	n := 0
	// Scan 2px around the border-box top (outline 1px centered at Y-0.5).
	for py := int(by) - 2; py <= int(by); py++ {
		for px := int(bx) + 4; px < int(bx)+int(bw)-4; px += 2 {
			p := c.PixelAt(px, py)
			if p.R > 0x30 && p.B > 0x80 && p.G > 0x50 { // #4d90fe-ish blue
				n++
			}
		}
	}
	return n
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

	findClass := func(cls string) *dom.Element {
		for _, el := range doc.GetElementsByTagName("*") {
			if el.GetAttribute("class") == cls {
				return el
			}
		}
		return nil
	}
	var ta *dom.Element
	for _, el := range doc.GetElementsByTagName("textarea") {
		ta = el
		break
	}
	btn := findClass("btn-primary")

	paint := func() *graphics.Canvas {
		c := graphics.NewCanvas(1280, 940)
		wv.RenderView().MarkAllDirty()
		rendering.Paint(wv.RenderView(), c, rendering.Rect{0, 0, 1280, 940})
		return c
	}
	rebuild := func() {
		fr.MarkRenderTreeDirty()
		fr.RebuildRenderTree()
		wv.EnsureLayout()
	}

	// ① mouse focus button → no UA outline
	btn.SetFocused(true)
	rebuild()
	c1 := paint()
	bx, by, bw, bh, ok := boxOf(wv.RenderView(), btn)
	n1 := 0
	if ok {
		n1 = outlinePixels(c1, bx, by, bw, bh)
	}
	fmt.Printf("① button mouse-focus: outlinePixels=%d (want 0 → no ring on click)\n", n1)

	// ② keyboard focus button → UA outline painted
	btn.SetFocusByKeyboard(true)
	rebuild()
	cs2 := fr.Resolver().ResolveElement(btn)
	fmt.Printf("② button keyboard-focus: OutlineSet=%v width=%.1f color=#%02x%02x%02x\n",
		cs2.OutlineSet, cs2.OutlineWidth.Value, cs2.OutlineColor.R, cs2.OutlineColor.G, cs2.OutlineColor.B)
	c2 := paint()
	n2 := 0
	if ok {
		n2 = outlinePixels(c2, bx, by, bw, bh)
		fmt.Printf("   top-edge pixels y=%d..%d: ", int(by)-3, int(by))
		for py := int(by) - 3; py <= int(by); py++ {
			p := c2.PixelAt(int(bx)+int(bw)/2, py)
			fmt.Printf("(y=%d #%02x%02x%02x) ", py, p.R, p.G, p.B)
		}
		fmt.Println()
	}
	fmt.Printf("② button keyboard-focus: outlinePixels=%d (want >0 → ring on Tab)\n", n2)
	btn.SetFocused(false)
	btn.SetFocusByKeyboard(false)

	// ③ mouse focus textarea → author :focus outline 2px accent2 still painted
	ta.SetFocused(true)
	rebuild()
	c3 := paint()
	tx, ty, _, _, tok := boxOf(wv.RenderView(), ta)
	n3 := 0
	if tok {
		for py := int(ty) - 3; py < int(ty); py++ {
			p := c3.PixelAt(int(tx)+50, py)
			if p.R == 0x2a && p.G == 0xc3 && p.B == 0xde {
				n3++
			}
		}
	}
	fmt.Printf("③ textarea mouse-focus: author outline px=%d (want >0 → :focus still matches)\n", n3)

	if n1 == 0 && n2 > 0 && n3 > 0 {
		fmt.Println("=== ALL PASS ===")
	} else {
		fmt.Println("=== SOME FAILED ===")
		os.Exit(1)
	}
}
