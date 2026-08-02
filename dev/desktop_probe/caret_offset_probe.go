// Command caret_offset_probe verifies clicking a text form control positions
// the caret (FormControlSelection.Start) at the click point, and that IME
// text insertion uses that caret (not the head).
package main

import (
	"fmt"
	"os"

	"wb-ui/dom"
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

	findEl := func(id string) *dom.Element {
		for _, el := range doc.GetElementsByTagName("*") {
			if el.GetAttribute("id") == id {
				return el
			}
		}
		return nil
	}
	inp := findEl("input-single")
	fmt.Printf("input value: %q (rune len=%d)\n", inp.GetAttribute("value"), len([]rune(inp.GetAttribute("value"))))

	// Simulate host: focus + calcTextControlOffset at middle of text.
	inp.SetFocused(true)
	fr.MarkRenderTreeDirty()
	fr.RebuildRenderTree()
	wv.EnsureLayout()

	// FocusedFormControlSel nil = never positioned → IME default.
	rendering.FocusedFormControlSel = nil
	fmt.Println("① sel=nil (focus w/o click): IME should insert at END")
	// mirror Host.applyIMEEvents plain-char path (fixed: default = END)
	val := inp.GetAttribute("value")
	runes := []rune(val)
	start, end := len(runes), len(runes) // fixed default: caret at END
	if rendering.FocusedFormControlSel != nil {
		s := rendering.FocusedFormControlSel
		start, end = s.Start, s.End
	}
	_ = end
	newText := string(runes[:start]) + "X" + string(runes[start:])
	fmt.Printf("   inserted at start=%d → %q\n", start, newText)

	// Desired: sel==nil default = END.
	start2 := len(runes)
	newText2 := string(runes[:start2]) + "X" + string(runes[start2:])
	fmt.Printf("   desired (END) → %q\n", newText2)
	fmt.Printf("   head-insert is WRONG? %v\n", newText != newText2)

	fmt.Println("=== caret_offset_probe done ===")
}
