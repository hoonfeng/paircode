// Command transition_probe verifies CSS transition interpolation: a button
// hover changes background-color from #2ac3de to #38b8d6 over 0.2s; the
// switch track ::after thumb left animates 2px → 18px. It drives the global
// animation clock forward and checks intermediate values.
package main

import (
	"fmt"
	"os"

	"wb-ui/css"
	"wb-ui/dom"
	"wb-ui/page"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

const html = `<!DOCTYPE html><html><head><style>
.btn-primary { background: #2ac3de; color: #fff; padding: 6px 16px; transition: background 0.2s; }
.btn-primary:hover { background: #38b8d6; }
.switch .track { width: 34px; height: 18px; background: #2f334d; border: 1px solid #555; position: relative; display: inline-block; transition: background 0.2s; }
.switch .track::after { content: ""; position: absolute; top: 2px; left: 2px; width: 12px; height: 12px; border-radius: 50%; background: #9aa5ce; transition: left 0.2s, background 0.2s; }
.switch input:checked + .track { background: #4ecca3; }
.switch input:checked + .track::after { left: 18px; background: #fff; }
</style></head><body>
<button class="btn-primary" id="btn1">按钮</button>
<label class="switch"><input type="checkbox" id="sw"><span class="track" id="track1"></span></label>
</body></html>`

func main() {
	wv := webkit.NewWebView()
	if err := wv.LoadHTML(html); err != nil {
		fmt.Println("LoadHTML:", err)
		os.Exit(1)
	}
	fr := wv.MainFrame().Frame()
	wv.EnsureLayout()
	pseudoAfter := css.PseudoElementAfter

	doc := wv.Document()
	btn1 := findEl(doc, "btn1")
	sw := findEl(doc, "sw")
	track1 := findEl(doc, "track1")
	if btn1 == nil || sw == nil {
		fmt.Println("FAIL: elements missing")
		os.Exit(1)
	}

	bgOf := func(el *dom.Element) string {
		cs := fr.Resolver().ResolveElement(el)
		return fmt.Sprintf("#%02x%02x%02x", cs.BackgroundColor.R, cs.BackgroundColor.G, cs.BackgroundColor.B)
	}

	// ① hover 过渡：先跑一帧建立 registry（模拟窗口每帧 ApplyAnimations），
	// 记录初始值 #2ac3de；hover → rebuild → 再推进时钟验证插值。
	rendering.AnimationTime = 0
	rendering.ApplyAnimations(wv.RenderView())
	fmt.Printf("hover: initial bg=%s\n", bgOf(btn1))
	btn1.SetHovered(true)
	fr.MarkRenderTreeDirty()
	fr.RebuildRenderTree()
	wv.EnsureLayout()

	rendering.AnimationTime = 0.05
	rendering.ApplyAnimations(wv.RenderView())
	fmt.Printf("hover: t=0.05s bg=%s (expect between #2ac3de and #38b8d6)\n", bgOf(btn1))

	rendering.AnimationTime = 0.5
	rendering.ApplyAnimations(wv.RenderView())
	fmt.Printf("hover: t=0.5s  bg=%s (expect #38b8d6)\n", bgOf(btn1))
	btn1.SetHovered(false)

	// ② 开关 checked 过渡：thumb left 2px → 18px
	sw.SetAttribute("checked", "checked")
	fr.MarkRenderTreeDirty()
	fr.RebuildRenderTree()
	wv.EnsureLayout()
	thumbLeft := func() float64 {
		rv := wv.RenderView()
		state := rv.LayoutState()
		var after rendering.RenderObject
		var walk func(ro rendering.RenderObject)
		walk = func(ro rendering.RenderObject) {
			if ro == nil {
				return
			}
			if n := ro.Node(); n != nil {
				if e, ok := n.(*dom.Element); ok && e.GetAttribute("id") == "track1" {
					for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
						if c.Node() == nil {
							after = c
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
			return -1
		}
		// 渲染树 box 的实际 style（transition 插值写入此）
		if st := after.Style(); st != nil {
			fmt.Printf("  [box] ::after left=%q\n", st.Properties["left"])
		}
		return state.GeometryForBox(after.LayoutBox()).Left()
	}
	// 过渡起点：用插值 left 写入后需重新布局——模拟 host 的 inFlight → SetNeedsLayout：
	step := func(t float64) float64 {
		rendering.AnimationTime = t
		rendering.ApplyAnimations(wv.RenderView())
		fr.SetNeedsLayout(true)
		wv.EnsureLayout()
		// 调试：伪元素 style 的 left
		if as, _, ok := fr.Resolver().ResolvePseudoElement(track1, pseudoAfter); ok {
			fmt.Printf("  [diag] t=%.1f ::after left=%q dur=%.1f\n", t, as.Properties["left"], as.TransitionDuration)
		}
		return thumbLeft()
	}
	lMid := step(0.1)
	lEnd := step(0.5)
	fmt.Printf("switch: t=0.1s thumb.left=%.1f (between 3 and 19)\n", lMid)
	fmt.Printf("switch: t=0.5s thumb.left=%.1f (expect ~19)\n", lEnd)

	fmt.Println("=== transition_probe done ===")
}

func findEl(doc *dom.Document, id string) *dom.Element {
	for _, el := range doc.GetElementsByTagName("*") {
		if el.GetAttribute("id") == id {
			return el
		}
	}
	return nil
}

var _ = page.NewSettings
