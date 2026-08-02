// Command multi_probe verifies component polymorphism (:hover / :checked /
// :focus dynamic pseudo-classes) through the full pipeline:
//  1. state change (SetHovered / SetChecked) on the DOM element
//  2. MarkRenderTreeDirty → RebuildRenderTree (ClearCache + re-resolve)
//  3. computed style reflects the pseudo-class rule
package main

import (
	"fmt"
	"os"

	"wb-ui/dom"
	"wb-ui/html5"
	"wb-ui/page"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

const html = `<!DOCTYPE html><html><head><style>
:root { --bg3: #2f334d; --accent: #4ecca3; --fg2: #9aa5ce; }
.btn-primary { background: #2ac3de; color: #fff; padding: 6px 16px; }
.btn-primary:hover { background: #38b8d6; }
.btn-green { background: #3db98f; }
.btn-green:hover { background: #4ecca3; }
.switch .track { width: 34px; height: 18px; background: var(--bg3); border: 1px solid #555; position: relative; display: inline-block; }
.switch .track::after { content: ""; position: absolute; top: 2px; left: 2px; width: 12px; height: 12px; border-radius: 50%; background: var(--fg2); }
.switch input:checked + .track { background: var(--accent); }
.switch input:checked + .track::after { left: 18px; background: #fff; }
.input:focus { outline: 2px solid var(--accent); }
</style></head><body>
<button class="btn-primary" id="btn1">主按钮</button>
<button class="btn-green" id="btn2">绿色</button>
<label class="switch"><input type="checkbox" id="switch-1"><span class="track" id="track1"></span></label>
<input class="input" id="inp1">
</body></html>`

func main() {
	wv := webkit.NewWebView()
	if err := wv.LoadHTML(html); err != nil {
		fmt.Println("LoadHTML err:", err)
		os.Exit(1)
	}
	fr := wv.MainFrame().Frame()
	resolver := fr.Resolver()

	doc := wv.Document()
	getEl := func(id string) *dom.Element {
		for _, el := range doc.GetElementsByTagName("*") {
			if el.GetAttribute("id") == id {
				return el
			}
		}
		return nil
	}
	btn1 := getEl("btn1")
	sw := getEl("switch-1")
	inp := getEl("inp1")
	if btn1 == nil || sw == nil || inp == nil {
		fmt.Println("FAIL: elements not found")
		os.Exit(1)
	}

	// helper: force rebuild + layout then read computed style
	recalc := func() *rendering.RenderView {
		fr.MarkRenderTreeDirty()
		fr.RebuildRenderTree()
		wv.EnsureLayout()
		return wv.RenderView()
	}
	// read element's computed background via a fresh resolve
	bgOf := func(el *dom.Element) string {
		cs := resolver.ResolveElement(el)
		if cs == nil {
			return "?"
		}
		return fmt.Sprintf("#%02x%02x%02x", cs.BackgroundColor.R, cs.BackgroundColor.G, cs.BackgroundColor.B)
	}

	// ① :hover 多态
	recalc()
	b0 := bgOf(btn1)
	btn1.SetHovered(true)
	recalc()
	b1 := bgOf(btn1)
	fmt.Printf("btn1 :hover background: %s -> %s\n", b0, b1)
	if b1 == b0 {
		fmt.Println("  WARN: hover did not change btn1 background")
	} else {
		fmt.Println("  OK: hover polymorphic style applied")
	}
	btn1.SetHovered(false)

	// ② :checked 多态（开关 track 变色 + ::after 左移）
	recalc()
	if in, ok := html5.ToInputElement(sw); ok {
		in.SetChecked(true)
	}
	recalc()
	rv := wv.RenderView()
	state := rv.LayoutState()
	// 找 track 的 ::after 伪元素几何
	var afterLeft float64 = -1
	var walk func(ro rendering.RenderObject)
	walk = func(ro rendering.RenderObject) {
		if ro == nil {
			return
		}
		if n := ro.Node(); n != nil {
			if e, ok := n.(*dom.Element); ok && e.GetAttribute("id") == "track1" {
				for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
					if c.Node() == nil {
						afterLeft = state.GeometryForBox(c.LayoutBox()).Left()
					}
				}
			}
		}
		for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	trackCS := resolver.ResolveElement(getEl("track1"))
	fmt.Printf("switch checked: track bg=#%02x%02x%02x ::after.left=%.0f\n",
		trackCS.BackgroundColor.R, trackCS.BackgroundColor.G, trackCS.BackgroundColor.B, afterLeft)
	if trackCS.BackgroundColor.R == 0x4e && trackCS.BackgroundColor.G == 0xcc && trackCS.BackgroundColor.B == 0xa3 {
		fmt.Println("  OK: :checked track turned accent color")
	} else {
		fmt.Println("  WARN: :checked track color mismatch")
	}
	if afterLeft > 15 {
		fmt.Println("  OK: ::after thumb moved right (left=18px)")
	} else {
		fmt.Println("  WARN: ::after thumb did not move")
	}

	// ③ :focus 多态（outline 指示器）
	recalc()
	inp.SetFocused(true)
	recalc()
	fcs := resolver.ResolveElement(inp)
	fmt.Printf("input :focus outline width=%.0f style=%s color=#%02x%02x%02x set=%v\n",
		fcs.OutlineWidth.Value, fcs.OutlineStyle, fcs.OutlineColor.R, fcs.OutlineColor.G, fcs.OutlineColor.B, fcs.OutlineSet)
	if fcs.OutlineSet && fcs.OutlineWidth.Value > 0 {
		fmt.Println("  OK: :focus outline parsed")
	} else {
		fmt.Println("  WARN: :focus outline not parsed")
	}

	fmt.Println("=== multi_probe done ===")
}

// ensure page import used
var _ = page.NewSettings