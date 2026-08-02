// Command compound_probe tests whether a scoped compound selector
// (.foo[data-v-xxx]) applies overflow-y to an element carrying the attribute.
package main

import (
	"fmt"
	"log"
	"path/filepath"

	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
	"wb-ui/webkit"
)

const html = `<!DOCTYPE html>
<html><head><meta charset="utf-8">
<style>
.foo[data-v-abc123] { overflow-y: auto; flex: 1; padding: 2px 0; height: 100px; }
.bar { overflow-x: hidden; }
</style>
</head><body>
<div id="app">
  <div class="foo" data-v-abc123><div style="height:300px">tall content</div></div>
  <div class="bar">x</div>
</div>
</body></html>`

func main() {
	log.SetFlags(0)
	_ = filepath.Join
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style2, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}, text)
	}
	wv := webkit.NewWebView()
	_ = wv.JSInterpreter()
	wv.LoadHTML(html)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()

	var walk func(o rendering.RenderObject)
	count := 0
	walk = func(o rendering.RenderObject) {
		count++
		if nn := o.Node(); nn != nil {
			if el, ok2 := nn.(*dom.Element); ok2 {
				cls := el.GetAttribute("class")
				if cls == "foo" || cls == "bar" {
					st := o.Style()
					fmt.Printf("%s: ovfX=%d ovfY=%d padT=%+v flexGrow=%.1f\n",
						cls, st.OverflowX, st.OverflowY, st.PaddingTop, st.FlexGrow)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
	fmt.Printf("walked %d objects\n", count)
	_ = style.OverflowAuto
	_ = jsc.Undefined
}
