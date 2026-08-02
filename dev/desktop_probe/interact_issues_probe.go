// Command interact_issues_probe renders interact_test.html and verifies the
// six reported issues with concrete numbers:
//   1. switch track size (should be 34x18 border-box vs browser)
//   2. hover style change on buttons (SetHovered → background)
//   3. textarea :focus outline parsing
//   4. single-line input :focus outline parsing + radius
//   5. list-item.selected background (rgba semi-transparent)
//   6. transition property presence
package main

import (
	"fmt"
	"os"

	"wb-ui/css"
	"wb-ui/dom"
	"wb-ui/html5"
	"wb-ui/layout"
	"wb-ui/page"
	"wb-ui/rendering"
	"wb-ui/style"
	"wb-ui/webkit"
)

const htmlPath = "cmd/desktop/web-ui/interact_test.html"

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
	resolver := fr.Resolver()
	wv.EnsureLayout()

	doc := wv.Document()
	find := func(id string) *dom.Element {
		for _, el := range doc.GetElementsByTagName("*") {
			if el.GetAttribute("id") == id {
				return el
			}
		}
		return nil
	}
	findClass := func(cls string) *dom.Element {
		for _, el := range doc.GetElementsByTagName("*") {
			if el.GetAttribute("class") == cls {
				return el
			}
		}
		return nil
	}
	swTrack := findClass("track")
	btnPrimary := find("btn-primary")
	textarea := findClass("textarea")
	input := findClass("input")
	listItem := findClass("list-item")

	// helper: box geometry from render tree by DOM id/class
	geoOf := func(el *dom.Element) (x, y, w, h float64, ok bool) {
		state := wv.RenderView().LayoutState()
		var findRO func(ro rendering.RenderObject) bool
		findRO = func(ro rendering.RenderObject) bool {
			if ro == nil {
				return false
			}
			if ro.Node() == el {
				if lb := ro.LayoutBox(); lb != nil {
					geo := state.GeometryForBox(lb)
					x, y, w, h = geo.Left(), geo.Top(), geo.BorderBoxWidth(), geo.BorderBoxHeight()
					return true
				}
			}
			for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
				if findRO(c) {
					return true
				}
			}
			return false
		}
		findRO(rendering.RenderObject(wv.RenderView()))
		return x, y, w, h, x != 0 || y != 0
	}

	// ① switch track size
	if tx, ty, tw, th, tok := geoOf(swTrack); tok {
		fmt.Printf("① switch track: x=%.0f y=%.0f w=%.0f h=%.0f (CSS 34x18 border-box)\n", tx, ty, tw, th)
		tcs := resolver.ResolveElement(swTrack)
		fmt.Printf("   track style: height=%q width=%q boxSizing=%q borderT=%v\n",
			tcs.Height.String(), tcs.Width.String(), tcs.BoxSizing, tcs.BorderTopWidth.Value)
		if lab := swTrack.ParentElement(); lab != nil {
			lcs := resolver.ResolveElement(lab)
			fmt.Printf("   label style: display=%d align=%q\n", lcs.Display, lcs.AlignItems)
		}
	}

	// ② hover
	cs0 := resolver.ResolveElement(btnPrimary)
	fmt.Printf("② btn-primary bg before hover: #%02x%02x%02x\n", cs0.BackgroundColor.R, cs0.BackgroundColor.G, cs0.BackgroundColor.B)
	btnPrimary.SetHovered(true)
	fr.MarkRenderTreeDirty()
	fr.RebuildRenderTree()
	cs1 := resolver.ResolveElement(btnPrimary)
	fmt.Printf("② btn-primary bg after hover:  #%02x%02x%02x\n", cs1.BackgroundColor.R, cs1.BackgroundColor.G, cs1.BackgroundColor.B)
	if cs1.BackgroundColor != cs0.BackgroundColor {
		fmt.Println("   ✓ hover changes bg")
	} else {
		fmt.Println("   ✗ hover does NOT change bg")
	}
	btnPrimary.SetHovered(false)

	// ③ textarea focus outline
	textarea.SetFocused(true)
	fr.RebuildRenderTree()
	tcs := resolver.ResolveElement(textarea)
	fmt.Printf("③ textarea :focus outline set=%v width=%.0f style=%q color=#%02x%02x%02x\n",
		tcs.OutlineSet, tcs.OutlineWidth.Value, tcs.OutlineStyle, tcs.OutlineColor.R, tcs.OutlineColor.G, tcs.OutlineColor.B)
	textarea.SetFocused(false)

	// ④ input focus outline + radius
	input.SetFocused(true)
	fr.RebuildRenderTree()
	ics := resolver.ResolveElement(input)
	fmt.Printf("④ input :focus outline set=%v width=%.0f style=%q color=#%02x%02x%02x radius=%.0f\n",
		ics.OutlineSet, ics.OutlineWidth.Value, ics.OutlineStyle, ics.OutlineColor.R, ics.OutlineColor.G, ics.OutlineColor.B,
		ics.BorderRadius.Value)
	input.SetFocused(false)

	// ⑤ list-item.selected background
	li := findClass("list-item")
	if li != nil {
		li.SetAttribute("class", "list-item selected")
	}
	fr.RebuildRenderTree()
	lcs := resolver.ResolveElement(listItem)
	fmt.Printf("⑤ list-item.selected bg: #%02x%02x%02x a=%d\n",
		lcs.BackgroundColor.R, lcs.BackgroundColor.G, lcs.BackgroundColor.B, lcs.BackgroundColor.A)
	if lcs.BackgroundColor.A < 255 && lcs.BackgroundColor.A > 0 {
		fmt.Println("   ✓ semi-transparent rgba parsed")
	} else {
		fmt.Println("   ✗ rgba alpha not preserved")
	}

	// ⑥ transition
	trCS := resolver.ResolveElement(swTrack)
	fmt.Printf("⑥ track transition: prop=%q dur=%.2f\n", trCS.TransitionProperty, trCS.TransitionDuration)

	// ⑦ list selected background actual paint — check render tree style
	fmt.Println("=== interact_issues_probe done ===")
}

// keep imports used
var _ = layout.BuildLayoutTree
var _ = style.DisplayBlock
var _ = page.NewSettings
var _ = css.PseudoElementAfter
var _ = html5.InputText
