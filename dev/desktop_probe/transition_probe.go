// Command transition_probe verifies the four interaction fixes in one run:
//  1. tab :focus outline — UA 1px #4d90fe (matches browsers)
//  2. textarea :focus outline 2px painted correctly (pixel scan)
//  3. switch transition — thumb left animates over frames, not a jump
//  4. switch checked — track border-color == accent (solid, no gray line)
package main

import (
	"fmt"
	"os"

	"wb-ui/css"
	"wb-ui/dom"
	"wb-ui/html5"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

const htmlPath = "cmd/desktop/web-ui/interact_test.html"

func findID(doc *dom.Document, id string) *dom.Element {
	for _, el := range doc.GetElementsByTagName("*") {
		if el.GetAttribute("id") == id {
			return el
		}
	}
	return nil
}

func findClass(doc *dom.Document, cls string) *dom.Element {
	for _, el := range doc.GetElementsByTagName("*") {
		if el.GetAttribute("class") == cls {
			return el
		}
	}
	return nil
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

	// ① tab :focus outline (UA default 1px #4d90fe)
	tab := findClass(doc, "tab")
	tab.SetFocused(true)
	fr.MarkRenderTreeDirty()
	fr.RebuildRenderTree()
	tabCS := fr.Resolver().ResolveElement(tab)
	fmt.Printf("① tab :focus outline set=%v width=%.1f color=#%02x%02x%02x (UA default = browsers)\n",
		tabCS.OutlineSet, tabCS.OutlineWidth.Value, tabCS.OutlineColor.R, tabCS.OutlineColor.G, tabCS.OutlineColor.B)
	tab.SetFocused(false)

	// ② textarea :focus outline 2px painted (pixel scan)
	ta := findClass(doc, "textarea")
	ta.SetFocused(true)
	fr.MarkRenderTreeDirty()
	fr.RebuildRenderTree()
	wv.EnsureLayout()
	c := graphics.NewCanvas(1280, 940)
	rv := wv.RenderView()
	rv.MarkAllDirty()
	rendering.Paint(rv, c, rendering.Rect{0, 0, 1280, 940})
	var tx, ty float64
	var findRO func(ro rendering.RenderObject) bool
	findRO = func(ro rendering.RenderObject) bool {
		if ro == nil {
			return false
		}
		if ro.Node() == ta {
			if rb := ro.(*rendering.RenderBox); rb != nil {
				tx, ty = rb.X(), rb.Y()
				return true
			}
		}
		for cc := ro.FirstChild(); cc != nil; cc = cc.NextSibling() {
			if findRO(cc) {
				return true
			}
		}
		return false
	}
	findRO(rendering.RenderObject(rv))
	outPx := 0
	for y := int(ty) - 4; y < int(ty)+2; y++ {
		px := c.PixelAt(int(tx)+50, y)
		if px.R == 0x2a && px.G == 0xc3 && px.B == 0xde {
			outPx++
		}
	}
	fmt.Printf("② textarea :focus outline painted pixels=%d (want >=2, 2px accent2)\n", outPx)
	ta.SetFocused(false)

	// ③ switch checked + transition timing
	sw := findID(doc, "switch-1")
	track := findClass(doc, "track")
	in, _ := html5.ToInputElement(sw)

	// Transition timeline: host runs ApplyAnimations EVERY frame. Frame 1
	// records the old settled value (left=2px); the checked rebuild happens
	// between frames; the next ApplyAnimations detects 2→18 and animates.
	// NOTE: read the thumb position from the RENDER TREE pseudo-element box
	// (which ApplyAnimations interpolates), NOT ResolvePseudoElement (which
	// re-resolves an uninterpolated style each call).
	thumbLeft := func() (float64, bool) {
		var l float64
		var found bool
		var w func(o rendering.RenderObject)
		w = func(o rendering.RenderObject) {
			if o == nil || found {
				return
			}
			// Pseudo-element box: its DOM host is the nearest ancestor with a
			// Node() — the .track span. Match o whose style has a "left"
			// transition target and whose parent chain reaches track.
			if o.Style() != nil && o.Style().TransitionDuration > 0 && o.Node() == nil {
				for p := o.Parent(); p != nil; p = p.Parent() {
					if p.Node() == track {
						if l2, ok := parseLen(o.Style().GetProperty("left")); ok {
							l, found = l2, true
							return
						}
					}
				}
			}
			for c := o.FirstChild(); c != nil; c = c.NextSibling() {
				w(c)
			}
		}
		w(rendering.RenderObject(rv))
		return l, found
	}
	rendering.AnimationTime = 9.9 // frame N: old state (unchecked, left=2px)
	rendering.ApplyAnimations(wv.RenderView())
	oldLeft, _ := thumbLeft()
	in.SetChecked(true) // state change between frames
	fr.MarkRenderTreeDirty()
	fr.RebuildRenderTree()
	wv.EnsureLayout()
	rv = wv.RenderView() // ★ re-acquire after rebuild (page swapped renderView)
	// diag: what does the resolver produce for ::after right now?
	if pcs, _, ok := fr.Resolver().ResolvePseudoElement(track, css.PseudoElementAfter); ok {
		fmt.Printf("   [diag] ResolvePseudoElement::after left=%q\n", pcs.GetProperty("left"))
	}
	// diag: pseudo box presence + TransitionDuration in the render tree
	diagCount := 0
	var w2 func(o rendering.RenderObject)
	w2 = func(o rendering.RenderObject) {
		if o == nil {
			return
		}
		if o.Style() != nil && o.Style().TransitionDuration > 0 {
			host := "<nil>"
			if o.Node() != nil {
				host = o.Node().NodeName()
			}
			diagCount++
			fmt.Printf("   [diag] transition box node=%v host=%s dur=%.2f left=%q\n", o.Node(), host, o.Style().TransitionDuration, o.Style().GetProperty("left"))
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			w2(c)
		}
	}
	w2(rendering.RenderObject(rv))
	fmt.Printf("   [diag] transition-capable boxes=%d\n", diagCount)
	trCS := fr.Resolver().ResolveElement(track)
	fmt.Printf("③ checked track bg=#%02x%02x%02x border=#%02x%02x%02x solid=%v diagCustom=%d prop=%q\n",
		trCS.BackgroundColor.R, trCS.BackgroundColor.G, trCS.BackgroundColor.B,
		trCS.BorderColor("top").R, trCS.BorderColor("top").G, trCS.BorderColor("top").B,
		trCS.BorderColor("top") == trCS.BackgroundColor, len(trCS.CustomProperties), trCS.GetProperty("border-color"))
	rendering.AnimationTime = 10.0 // frame N+1: transition starts 2→18
	rendering.ApplyAnimations(wv.RenderView())
	startLeft, _ := thumbLeft()
	rendering.AnimationTime = 10.06 // +60ms of 200ms → ~30% → ~6.8px
	rendering.ApplyAnimations(wv.RenderView())
	midLeft, _ := thumbLeft()
	rendering.AnimationTime = 10.25 // +250ms > 200ms → done → 18px
	rendering.ApplyAnimations(wv.RenderView())
	endLeft, _ := thumbLeft()
	fmt.Printf("③ transition thumb: old=%v start=%v mid(t+60ms)=%v end(t+250ms)=%v\n",
		oldLeft, startLeft, midLeft, endLeft)
	if midLeft > oldLeft && midLeft < endLeft && endLeft >= 17.9 {
		fmt.Println("   ✓ transition animates smoothly 2px → 18px")
	} else {
		fmt.Println("   ✗ transition NOT animating (jump or stuck)")
	}
	fmt.Println("=== transition_probe done ===")
}

// parseLen parses a CSS length like "18px".
func parseLen(s string) (float64, bool) {
	if len(s) < 3 || s[len(s)-2:] != "px" {
		return 0, false
	}
	var v float64
	_, err := fmt.Sscanf(s[:len(s)-2], "%f", &v)
	return v, err == nil
}
