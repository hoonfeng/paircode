// Command ime_insert_probe simulates the full click→caret→IME-input chain on
// the single-line input of interact_test.html and prints the resulting value,
// proving where a typed character lands.
//
// It mirrors Host.calcTextControlOffset + Host.applyIMEEvents(EventCharInput):
//  1. click the input at its text-middle → caret offset
//  2. FocusedFormControlSel.Start = offset (host behavior)
//  3. plain char input at sel.Start
// and also the no-click case (sel == nil → caret defaults to END).
package main

import (
	"fmt"
	"os"

	"wb-ui/dom"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
	"wb-ui/webkit"
)

const htmlPath = "cmd/desktop/web-ui/interact_test.html"

func findEl(doc *dom.Document, id string) *dom.Element {
	for _, el := range doc.GetElementsByTagName("*") {
		if el.GetAttribute("id") == id {
			return el
		}
	}
	return nil
}
func boxOf(rv *rendering.RenderView, el *dom.Element) (x, y, w, h float64, st *style.ComputedStyle, ok bool) {
	var walk func(ro rendering.RenderObject) bool
	walk = func(ro rendering.RenderObject) bool {
		if ro == nil {
			return false
		}
		if b2, isBox := ro.(*rendering.RenderBox); isBox {
			nn := "?"
			idAttr := "?"
			if n := ro.Node(); n != nil {
				nn = n.NodeName()
				if e2, ok4 := n.(*dom.Element); ok4 {
					idAttr = e2.GetAttribute("id")
				}
			}
			if nn == "INPUT" || nn == "TEXTAREA" {
				fmt.Printf("  [diag] box=%T node=%T nodeName=%q id=%q abs=(%.1f,%.1f) size=(%.1f,%.1f)\n",
					b2, ro.Node(), nn, idAttr, b2.AbsoluteX(), b2.AbsoluteY(), b2.Width(), b2.Height())
			}
		}
		if n := ro.Node(); n != nil {
			if e, ok2 := n.(*dom.Element); ok2 && e.GetAttribute("id") == el.GetAttribute("id") && e.GetAttribute("id") != "" {
				if box, ok3 := ro.(*rendering.RenderBox); ok3 {
					x = box.AbsoluteX()
					y = box.AbsoluteY()
					w = box.Width()
					h = box.Height()
					st = box.Style()
					ok = true
					return true
				}
			}
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
func caretAt(st *style.ComputedStyle, text string, bx, by, bw, bh, cssX, cssY float64) int {
	fontSize, family := 14.0, "Consolas"
	weight := 400
	if st != nil {
		if st.FontSize.Value > 0 && !st.FontSize.IsAuto() {
			fontSize = st.FontSize.Value
		}
		if st.FontFamily != "" {
			family = st.FontFamily
		}
	}
	font := graphics.Font{Family: family, Size: fontSize, Weight: weight}
	ascent := graphics.GlobalFontAscent(font)
	if ascent <= 0 {
		ascent = fontSize * 0.8
	}
	descent := graphics.GlobalFontDescent(font)
	if descent < 0 {
		descent = 0
	}
	lineH := ascent + descent
	if st != nil {
		switch st.LineHeight.Unit {
		case "px":
			if st.LineHeight.Value > 0 {
				lineH = st.LineHeight.Value
			}
		case "%":
			if st.LineHeight.Value > 0 {
				lineH = st.LineHeight.Value / 100 * fontSize
			}
		case "":
			if st.LineHeight.Value > 0 {
				lineH = st.LineHeight.Value * fontSize
			}
		}
	}
	if lineH <= 0 {
		lineH = fontSize * 1.2
	}
	padX := 4.0
	padY := 4.0
	if st != nil {
		if v := st.PaddingLeft.Value; v > 0 && !st.PaddingLeft.IsAuto() {
			padX = v
		}
		if v := st.PaddingTop.Value; v > 0 && !st.PaddingTop.IsAuto() {
			padY = v
		}
	}
	return rendering.CalcFormControlCaretOffset(text, true, cssX, cssY, bx, by, font, padX, padY, lineH)
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
	_ = fr
	fr.MarkRenderTreeDirty()
	fr.RebuildRenderTree()
	wv.EnsureLayout()
	doc := wv.Document()
	inp := findEl(doc, "input-single")
	val := inp.GetAttribute("value")
	fmt.Printf("input value=%q runeLen=%d id=%q\n", val, len([]rune(val)), inp.GetAttribute("id"))

	bx, by, bw, bh, st, ok := boxOf(wv.RenderView(), inp)
	if !ok {
		// Diagnose: walk the render tree and print node names.
		var dump func(ro rendering.RenderObject, ind string)
		dump = func(ro rendering.RenderObject, ind string) {
			if ro == nil {
				return
			}
			name := "?"
			if n := ro.Node(); n != nil {
				name = n.NodeName()
			}
			fmt.Printf("%s%s %T\n", ind, name, ro)
			for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
				dump(c, ind+"  ")
			}
		}
		dump(rendering.RenderObject(wv.RenderView()), "")
		fmt.Println("input box not found")
		os.Exit(1)
	}
	fmt.Printf("input box: bx=%.1f by=%.1f bw=%.1f bh=%.1f padLeft=%.1f\n", bx, by, bw, bh, st.PaddingLeft.Value)

	// ① click at the text middle (halfway through "hello wb-ui 你好")
	midX := bx + st.PaddingLeft.Value + graphics.MeasureText(graphics.Font{Family: "Consolas", Size: 13, Weight: 400}, "hello wb")*0.5
	midY := by + bh/2
	off := caretAt(st, val, bx, by, bw, bh, midX, midY)
	fmt.Printf("① click at cssX=%.1f (text-middle) → caret offset=%d\n", midX, off)

	// host sets sel from the click offset
	rendering.FocusedFormControlSel = &rendering.FormControlSelection{Start: off, End: off, Active: true}
	runes := []rune(val)
	start, end := len(runes), len(runes)
	if s := rendering.FocusedFormControlSel; s != nil {
		start, end = s.Start, s.End
	}
	newVal := string(runes[:start]) + "X" + string(runes[end:])
	fmt.Printf("① typed 'X' with sel.Start=%d → %q  (click-middle should be HERE)\n", start, newVal)

	// ② no click (sel == nil) → default must be END
	rendering.FocusedFormControlSel = nil
	runes = []rune(val)
	start, end = len(runes), len(runes)
	newVal2 := string(runes[:start]) + "X" + string(runes[end:])
	fmt.Printf("② sel==nil (focus w/o click) → %q  (should be END)\n", newVal2)

	// ③ click at the very START of the text (like clicking padding area)
	off0 := caretAt(st, val, bx, by, bw, bh, bx+1, midY)
	fmt.Printf("③ click at bx+1 → offset=%d (clicking the left edge legitimately puts caret at 0)\n", off0)

	// ④ multi-line textarea click positioning on each row
	ta := findEl(doc, "input-multi")
	tval := ta.GetAttribute("value")
	tbx, tby, tbw, tbh, tst, tok := boxOf(wv.RenderView(), ta)
	if !tok {
		fmt.Println("textarea box not found")
	} else {
		fmt.Printf("④ textarea box: (%.0f,%.0f) %.0fx%.0f padL=%.1f padT=%.1f lineH=%.1f\n",
			tbx, tby, tbw, tbh, tst.PaddingLeft.Value, tst.PaddingTop.Value, tst.LineHeight.Value*13)
		tlineH := tst.LineHeight.Value * 13
		if tlineH <= 0 {
			tlineH = 19.5
		}
		for row := 0; row < 3; row++ {
			cy := tby + tst.PaddingTop.Value + tlineH*float64(row) + tlineH/2
			cx := tbx + tst.PaddingLeft.Value + 60
		o := caretAt(tst, tval, tbx, tby, tbw, tbh, cx, cy)
			fmt.Printf("④ click row %d at (%.0f,%.0f) → offset=%d (row content len≈%d)\n", row, cx, cy, o, rowLens[row])
		}
	}

	fmt.Println("=== ime_insert_probe done ===")
}

var rowLens = []int{16, 17, 10}
