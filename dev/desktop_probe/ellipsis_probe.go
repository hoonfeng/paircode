// Command ellipsis_probe renders a flex row with an ellipsis item and checks
// the item's computed width vs its text width — does wb-ui compress a
// flex-auto item below its content (causing premature ellipsis)?
package main

import (
	"fmt"
	"log"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

const html = `<!DOCTYPE html>
<html><head><meta charset="utf-8">
<style>
.item-row { display:flex; align-items:center; gap:3px; padding:2px 4px; white-space:nowrap; width:270px; }
.chevron { width:12px; flex-shrink:0; }
.icon { width:14px; flex-shrink:0; }
.item-name { overflow:hidden; text-overflow:ellipsis; font-size:13px; }
.short-name { overflow:hidden; text-overflow:ellipsis; font-size:13px; }
</style>
</head><body>
<div class="item-row">
  <span class="chevron">&#9654;</span>
  <span class="icon">&#128196;</span>
  <span class="item-name">companion.exe long file name here</span>
</div>
<div class="item-row">
  <span class="chevron">&#9654;</span>
  <span class="icon">&#128196;</span>
  <span class="item-name">short.txt</span>
</div>
</body></html>`

func main() {
	log.SetFlags(0)
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style2, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}, text)
	}
	layout.FontMetricsFunc = func(family string, size float64, weight int, style2 string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}
	wv := webkit.NewWebView()
	_ = wv.JSInterpreter()
	wv.LoadHTML(html)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()

	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cls := el.GetAttribute("class")
			if cls == "item-name" || cls == "item-row" {
				x, y, w, h, ok2 := rendering.BoxGeometry(o)
				textW := 0.0
				if ok2 {
					textW = layout.MeasureTextFunc("", 13, 400, "", el.TextContent())
				}
				fmt.Printf("%s: x=%.0f y=%.0f w=%.1f h=%.1f textW=%.1f %s\n",
					cls, x, y, w, h, textW, map[bool]string{true: "OK", false: "TRUNCATED"}[w >= textW])
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
}
