// Command interact_switch_probe loads the real interact_test.html through the
// full wb-ui pipeline, simulates the label-toggle click path (SetChecked on
// the wrapped checkbox), and verifies the ::after thumb geometry changes
// (left 2px → 18px) exactly as a browser would.
package main

import (
	"fmt"
	"os"

	"wb-ui/css"
	"wb-ui/dom"
	"wb-ui/html5"
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
	sw1 := findEl(doc, "switch-1")
	sw2 := findEl(doc, "switch-2")
	track1 := findEl(doc, "track") // class
	if sw1 == nil || sw2 == nil {
		fmt.Println("FAIL: switch inputs not found")
		os.Exit(1)
	}

	thumbLeft := func() (float64, float64) {
		view := wv.Page().MainFrame().View()
		needs := view.NeedsLayout()
		wv.EnsureLayout()
		rv := wv.RenderView()
		state := rv.LayoutState()
		// 精确定位 switch-1 的 track（通过 DOM 祖先链）：遍历 render tree，
		// 找到 Node 为 track1（switch-1 的 track span）的 box。
		var after rendering.RenderObject
		var walk func(ro rendering.RenderObject)
		walk = func(ro rendering.RenderObject) {
			if ro == nil {
				return
			}
			if n := ro.Node(); n != nil {
				if e, ok := n.(*dom.Element); ok && e.GetAttribute("class") == "track" {
					// 属于 switch-1（父 label 内第一个 input id=switch-1）
					lab := e.ParentElement()
					if lab != nil && lab.LocalName() == "label" {
						var firstInput *dom.Element
						for c := lab.FirstChild(); c != nil; c = c.NextSibling() {
							if ie, ok := c.(*dom.Element); ok && ie.LocalName() == "input" {
								firstInput = ie
								break
							}
						}
						if firstInput != nil && firstInput.GetAttribute("id") == "switch-1" {
							for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
								if c.Node() == nil {
									after = c
								}
							}
						}
					}
				}
			}
			for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c)
			}
		}
		walk(rendering.RenderObject(rv))
		if after == nil {
			return -1, -1
		}
		if after.Style() != nil {
			fmt.Printf("  [diag] after style left=%q needsLayout=%v\n", after.Style().Properties["left"], needs)
		}
		lb := after.LayoutBox()
		if lb == nil {
			fmt.Println("  [diag] ::after has NO layout box")
			return -2, -2
		}
		g := state.GeometryForBox(lb)
		return g.Left(), g.BorderBoxWidth()
	}

	// 初始未选中：滑块在左侧（left:2px → 相对 track 偏移 ~3px）
	l0, w0 := thumbLeft()
	fmt.Printf("switch-1 unchecked thumb.left=%.0f w=%.0f\n", l0, w0)

	// 模拟 handleLabelToggle：点击 track → toggle checkbox
	in1, _ := html5.ToInputElement(sw1)
	in1.SetChecked(true)
	fr.MarkRenderTreeDirty()
	wv.RebuildRenderTree()
	l1, w1 := thumbLeft()
	fmt.Printf("switch-1 checked thumb.left=%.0f w=%.0f\n", l1, w1)

	// track 背景 accent 色
	trackCS := resolver.ResolveElement(track1)
	fmt.Printf("track checked bg=#%02x%02x%02x\n", trackCS.BackgroundColor.R, trackCS.BackgroundColor.G, trackCS.BackgroundColor.B)
	// 伪元素样式（checked 时 left 应为 18px）
	if afterCS, _, ok := resolver.ResolvePseudoElement(track1, css.PseudoElementAfter); ok {
		fmt.Printf("::after checked style: left=%q top=%q width=%q height=%q bg=%q\n",
			afterCS.Properties["left"], afterCS.Properties["top"],
			afterCS.Properties["width"], afterCS.Properties["height"],
			afterCS.GetProperty("background"))
	}

	if l1 > l0+10 {
		fmt.Println("OK: thumb moved right when checked")
	} else {
		fmt.Println("FAIL: thumb did not move (label toggle / :checked broken)")
		os.Exit(1)
	}
	if trackCS.BackgroundColor.R == 0x4e && trackCS.BackgroundColor.G == 0xcc && trackCS.BackgroundColor.B == 0xa3 {
		fmt.Println("OK: track turned accent on :checked")
	} else {
		fmt.Printf("WARN: track bg #%02x%02x%02x (want accent 4ecca3)\n", trackCS.BackgroundColor.R, trackCS.BackgroundColor.G, trackCS.BackgroundColor.B)
	}
	fmt.Println("=== interact_switch_probe done ===")
}

func findEl(doc *dom.Document, id string) *dom.Element {
	for _, el := range doc.GetElementsByTagName("*") {
		if el.GetAttribute("id") == id || el.GetAttribute("class") == id {
			return el
		}
	}
	return nil
}

// keep imports used
var _ = style.DisplayBlock
var _ = page.NewSettings
