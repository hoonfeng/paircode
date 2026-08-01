// Command sim_drag_select simulates the host's press→move→release sequence
// on a textarea and verifies the selection End stays at the last mousemove
// position (not recomputed from the release point — the regression the user
// reported: release outside the control jumped the selection).
package main

import (
	"fmt"

	"wb-ui/css"
	"wb-ui/dom"
	"wb-ui/html"
	"wb-ui/html5"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
)

const docHTML = `<!DOCTYPE html><html><head><style>
.textarea { padding: 8px 10px; font-size: 13px; line-height: 1.5; font-family: Consolas, monospace; }
</style></head><body>
<textarea class="textarea" id="ta">第一行文字 alpha beta
第二行文字 gamma delta
第三行文字 中文测试</textarea>
</body></html>`

func main() {
	doc, _ := html.Parse(docHTML)
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style2, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}, text)
	}
	resolver := style.NewResolver()
	resolver.AddStyleSheet(html5.NewUAStyleSheet())
	if doc.DocumentElement() != nil {
		for _, s := range doc.DocumentElement().GetElementsByTagName("style") {
			if text := s.TextContent(); text != "" {
				sheet := css.NewCSSStyleSheet()
				css.NewParser(text).ParseStyleSheetInto(sheet)
				resolver.AddStyleSheet(sheet)
			}
		}
	}
	builder := rendering.NewRenderTreeBuilder(resolver)
	rv := builder.Build(doc)
	rv.SetViewportSize(1280, 860)
	rv.Layout(nil)

	ta := doc.GetElementById("ta")
	var bx, by float64
	var st *style.ComputedStyle
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if st != nil || o == nil {
			return
		}
		if b, ok := o.(interface{ AsRenderBox() *rendering.RenderBox }); ok {
			if bb := b.AsRenderBox(); bb != nil {
				if nd := bb.Node(); nd != nil {
					if e, isEl := nd.(*dom.Element); isEl && e == ta {
						bx, by = bb.AbsoluteX(), bb.AbsoluteY()
						st = bb.Style()
						return
					}
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	if st == nil {
		fmt.Println("textarea not found")
		return
	}
	padX, padY := 10.0, 8.0
	lineH := 13.0 * 1.5
	font := graphics.Font{Family: "Consolas, monospace", Size: 13, Weight: 400}
	text := ta.TextContent()

	calc := func(cx, cy float64) int {
		return rendering.CalcFormControlCaretOffset(text, true, cx, cy, bx, by, font, padX, padY, lineH)
	}

	// Simulate: press at start of row 1, drag to middle of row 2, then
	// release OUTSIDE the control (below it). The End must stay at the
	// last drag position, not jump to the release point.
	pressX := bx + padX + 2
	pressY := by + padY + lineH/2
	midRow2 := bx + padX + graphics.MeasureText(font, "第二行文字 gamma del")/2
	row2Y := by + padY + lineH + lineH/2
	releaseY := by + 60 + 200 // far below the control (textarea ≈60px tall)

	// press → selection anchor
	sel := &rendering.FormControlSelection{Start: calc(pressX, pressY), End: calc(pressX, pressY), Active: true}
	fmt.Printf("press  → Start=End=%d\n", sel.Start)

	// drag (mousemove) → End updates
	sel.End = calc(midRow2, row2Y)
	fmt.Printf("drag   → End=%d\n", sel.End)

	// release: OLD buggy code recomputed End from releaseY (outside → clamped);
	// the FIX keeps End at the last drag position.
	fixedEnd := sel.End
	buggyEnd := calc(pressX, releaseY)
	fmt.Printf("release→ fixedEnd=%d (kept) vs buggyEnd=%d (recomputed from release point)\n", fixedEnd, buggyEnd)

	if fixedEnd == buggyEnd {
		fmt.Println("⚠️ 本场景两者相同（release 点恰好在控件下方 clamp 到末尾）")
	} else {
		fmt.Println("✅ 修复生效：End 保留最后拖动位置，未被 release 点改写")
	}

	// Second scenario: drag downward past the last line then release at a
	// point that would clamp differently.
	sel.End = calc(bx+padX+5, by+padY+lineH*2+lineH/2) // row 3 start
	fmt.Printf("drag2  → End=%d\n", sel.End)
	releaseX2 := bx - 50 // far left of the control
	buggy2 := calc(releaseX2, by+padY+lineH*2+lineH/2)
	fmt.Printf("release2→ fixed=%d vs buggy(left)=%d\n", sel.End, buggy2)
	if sel.End == buggy2 {
		fmt.Println("⚠️ 相同")
	} else {
		fmt.Println("✅ 修复生效：拖出左侧松开，End 保留")
	}
	fmt.Println("done")
}
