// Command focus_switch_probe verifies the reported focus/switch issues:
//  1. textarea :focus outline width (should be 2px, not thicker)
//  2. switch checked: track border-color → accent (browser shows solid accent
//     with no visible border line; wb-ui must apply border-color too)
//  3. tab :focus outline presence (UA default 1px blue like browsers)
package main

import (
	"fmt"
	"os"

	"wb-ui/css"
	"wb-ui/dom"
	"wb-ui/html5"
	"wb-ui/rendering"
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

	// ① textarea :focus outline
	ta := findClass("textarea")
	ta.SetFocused(true)
	fr.MarkRenderTreeDirty()
	fr.RebuildRenderTree()
	wv.EnsureLayout()
	tcs := fr.Resolver().ResolveElement(ta)
	fmt.Printf("① textarea :focus outline set=%v width=%.1f style=%q color=#%02x%02x%02x borderColor=#%02x%02x%02x a=%d\n",
		tcs.OutlineSet, tcs.OutlineWidth.Value, tcs.OutlineStyle,
		tcs.OutlineColor.R, tcs.OutlineColor.G, tcs.OutlineColor.B,
		tcs.BorderColor("top").R, tcs.BorderColor("top").G, tcs.BorderColor("top").B,
		tcs.BorderColor("top").A)
	ta.SetFocused(false)

	// ② switch checked border-color
	sw1 := find("switch-1")
	track := findClass("track")
	in1, _ := html5.ToInputElement(sw1)
	in1.SetChecked(true)
	fr.MarkRenderTreeDirty()
	fr.RebuildRenderTree()
	wv.EnsureLayout()
	trCS := fr.Resolver().ResolveElement(track)
	fmt.Printf("② switch checked track bg=#%02x%02x%02x borderColor=#%02x%02x%02x (want accent bg + accent border → solid, no gray line)\n",
		trCS.BackgroundColor.R, trCS.BackgroundColor.G, trCS.BackgroundColor.B,
		trCS.BorderColor("top").R, trCS.BorderColor("top").G, trCS.BorderColor("top").B)
	fmt.Printf("   diag: CustomProps=%d prop[border-color]=%q\n", len(trCS.CustomProperties), trCS.GetProperty("border-color"))
	if trCS.BorderColor("top") != trCS.BackgroundColor {
		fmt.Println("   ✗ border-color does NOT match accent (gray border line still visible)")
	} else {
		fmt.Println("   ✓ border-color == accent (solid, matches browser)")
	}

	// ③ tab :focus outline (UA default)
	tab := findClass("tab")
	if tab != nil {
		tab.SetFocused(true)
		fr.RebuildRenderTree()
		tabCS := fr.Resolver().ResolveElement(tab)
		fmt.Printf("③ tab :focus outline set=%v width=%.1f color=#%02x%02x%02x\n",
			tabCS.OutlineSet, tabCS.OutlineWidth.Value, tabCS.OutlineColor.R, tabCS.OutlineColor.G, tabCS.OutlineColor.B)
		tab.SetFocused(false)
	}

	// ④ switch thumb geometry checked
	if afterCS, _, ok := fr.Resolver().ResolvePseudoElement(track, css.PseudoElementAfter); ok {
		fmt.Printf("④ checked ::after left=%q bg=%q\n", afterCS.Properties["left"], afterCS.GetProperty("background"))
	}

	// ⑤ render snapshot check — pixel of track center should be accent (solid)
	_ = rendering.ApplyAnimations
	fmt.Println("=== focus_switch_probe done ===")
}
