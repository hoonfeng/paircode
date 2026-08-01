// Command caret_probe verifies multi-line textarea caret placement: clicking
// row 2/3 must produce an offset inside that row (not always row 1), honoring
// the CSS line-height (1.5) used by paintTextAreaText.
package main

import (
	"fmt"
	"strings"

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
			if text := s.TextContent(); strings.TrimSpace(text) != "" {
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
	// Find the textarea box.
	var bx, by, bw, bh float64
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
						bx, by, bw, bh = bb.AbsoluteX(), bb.AbsoluteY(), bb.Width(), bb.Height()
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
		fmt.Println("textarea box not found")
		return
	}
	fmt.Printf("textarea box=(%.0f,%.0f %.0fx%.0f) padding=%s/%s line-height=%v font-size=%v\n",
		bx, by, bw, bh, st.PaddingLeft.String(), st.PaddingTop.String(), st.LineHeight, st.FontSize)

	text := ta.TextContent()
	runes := []rune(text)
	lineStarts := []int{0}
	for i, r := range runes {
		if r == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}
	lineStarts = append(lineStarts, len(runes))
	for i := 0; i < len(lineStarts)-1; i++ {
		fmt.Printf("line %d: offsets [%d..%d) %q\n", i+1, lineStarts[i], lineStarts[i+1], text[lineStarts[i]:lineStarts[i+1]])
	}

	// Click each row's middle → expected offset inside that row.
	padX := 10.0
	padY := 8.0
	lineH := 13.0 * 1.5 // line-height:1.5
	font := graphics.Font{Family: "Consolas, monospace", Size: 13, Weight: 400}
	for i := 0; i < len(lineStarts)-1; i++ {
		lineText := runes[lineStarts[i]:lineStarts[i+1]]
		if len(lineText) == 0 {
			continue
		}
		// middle of the line
		midW := graphics.MeasureText(font, string(lineText)) / 2
		cx := bx + padX + midW
		cy := by + padY + float64(i)*lineH + lineH/2
		off := rendering.CalcFormControlCaretOffset(text, true, cx, cy, bx, by, font, padX, padY, lineH)
		inRow := off >= lineStarts[i] && off < lineStarts[i+1]
		status := "✅"
		if !inRow {
			status = "❌"
		}
		fmt.Printf("row %d click → offset %d (line %d [%d..%d)) %s\n",
			i+1, off, i+1, lineStarts[i], lineStarts[i+1], status)
	}
	fmt.Println("done")
}
