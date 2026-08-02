// Command ime_insert_probe verifies IME text insertion honors the caret:
// clicking mid-text then typing inserts at the caret (browser behavior),
// not appending to the end of the whole value.
package main

import (
	"fmt"
	"os"

	"wb-ui/dom"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

const htmlPath = "cmd/desktop/web-ui/interact_test.html"

// insertAtCaret mirrors Host.applyIMEEvents plain-char path.
func insertAtCaret(val string, sel *rendering.FormControlSelection, char string) (string, int) {
	runes := []rune(val)
	start, end := 0, len(runes)
	if sel != nil {
		start, end = sel.Start, sel.End
		if start > end {
			start, end = end, start
		}
		if start < 0 {
			start = 0
		}
		if end > len(runes) {
			end = len(runes)
		}
	}
	return string(runes[:start]) + char + string(runes[end:]), start + 1
}

// composeAtCaret mirrors EventCompositionUpdate: insert compose text at caret.
func composeAtCaret(base string, pos int, compose string) string {
	runes := []rune(base)
	if pos > len(runes) {
		pos = len(runes)
	}
	return string(runes[:pos]) + compose + string(runes[pos:])
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
	wv.EnsureLayout()
	doc := wv.Document()

	var ta *dom.Element
	for _, el := range doc.GetElementsByTagName("textarea") {
		ta = el
		break
	}
	val := ta.TextContent()
	runes := []rune(val)
	fmt.Printf("initial: %q (len=%d)\n", val, len(runes))

	// ① click at end of line 1 (offset 16) → type "X"
	line1 := "第一行文字 alpha beta"
	caret := len([]rune(line1))
	newVal, newCaret := insertAtCaret(val, &rendering.FormControlSelection{Start: caret, End: caret}, "X")
	fmt.Printf("① click @%d type X → %q caret=%d\n", caret, newVal, newCaret)
	ok1 := newVal == "第一行文字 alpha betaX\n第二行文字 gamma delta\n第三行文字 中文测试"
	fmt.Printf("   ✓ inserted at caret: %v\n", ok1)

	// ② click at start (offset 0) → type "Y"
	newVal2, _ := insertAtCaret(val, &rendering.FormControlSelection{Start: 0, End: 0}, "Y")
	fmt.Printf("② click @0 type Y → %q\n", newVal2)
	ok2 := newVal2 == "Y第一行文字 alpha beta\n第二行文字 gamma delta\n第三行文字 中文测试"
	fmt.Printf("   ✓ inserted at start: %v\n", ok2)

	// ③ selection replace: select "alpha" (line1 chars 6..11) → type "Z"
	selStart := len([]rune("第一行文字 "))
	selEnd := selStart + len("alpha")
	newVal3, newCaret3 := insertAtCaret(val, &rendering.FormControlSelection{Start: selStart, End: selEnd}, "Z")
	fmt.Printf("③ select alpha → type Z → %q caret=%d\n", newVal3, newCaret3)
	ok3 := newVal3 == "第一行文字 Z beta\n第二行文字 gamma delta\n第三行文字 中文测试"
	fmt.Printf("   ✓ replaced selection: %v\n", ok3)

	// ④ composition at caret: compose "中" at offset 7 (after 第一行文字)
	base := "第一行文字 beta\n第二行"
	composed := composeAtCaret(base, len([]rune("第一行文字 ")), "中")
	fmt.Printf("④ compose 中 at caret 7 → %q\n", composed)
	ok4 := composed == "第一行文字 中beta\n第二行"
	fmt.Printf("   ✓ composition at caret: %v\n", ok4)

	if ok1 && ok2 && ok3 && ok4 {
		fmt.Println("=== ALL PASS ===")
	} else {
		fmt.Println("=== SOME FAILED ===")
		os.Exit(1)
	}
}
