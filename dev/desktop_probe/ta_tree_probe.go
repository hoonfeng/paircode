// Command ta_tree_probe dumps the render tree around the textarea to find
// why its walk enters with a pre-applied overflow clip (parent chain + order).
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
	var ta *dom.Element
	for _, el := range doc.GetElementsByTagName("textarea") {
		ta = el
		break
	}
	// Focus like the probe (rebuild after focus).
	ta.SetFocused(true)
	fr.MarkRenderTreeDirty()
	fr.RebuildRenderTree()
	wv.EnsureLayout()

	rv := wv.RenderView()
	var taRO rendering.RenderObject
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil {
			return
		}
		if o.Node() == ta {
			taRO = o
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))

	if taRO == nil {
		fmt.Println("textarea RO not found")
		os.Exit(1)
	}
	fmt.Println("=== textarea render parent chain ===")
	for p := taRO.Parent(); p != nil; p = p.Parent() {
		nm := "<anon>"
		if p.Node() != nil {
			nm = p.Node().NodeName()
		}
		fmt.Printf("  parent %s\n", nm)
	}
	fmt.Println("=== textarea siblings ===")
	if p := taRO.Parent(); p != nil {
		idx := 0
		for c := p.FirstChild(); c != nil; c = c.NextSibling() {
			nm := "<anon>"
			if c.Node() != nil {
				nm = c.Node().NodeName()
			}
			mark := ""
			if c == taRO {
				mark = "  ← TEXTAREA"
			}
			fmt.Printf("  [%d] %s%s\n", idx, nm, mark)
			idx++
		}
	}
	fmt.Println("=== done ===")
}
