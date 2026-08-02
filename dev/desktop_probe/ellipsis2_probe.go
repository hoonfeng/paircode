// Command ellipsis2_probe tests the flex row with an SVG icon (like the
// companion SvgIcon) instead of a span — does the svg's flex size break the
// item-name width?
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
</style>
</head><body>
<div class="item-row">
  <span class="chevron">&#9654;</span>
  <svg class="icon" width="14" height="14"><path d="M0 0h14v14H0z"/></svg>
  <span class="item-name">companion.exe long file name here</span>
</div>
<div class="item-row">
  <span class="chevron">&#9654;</span>
  <svg class="icon" width="14" height="14"><path d="M0 0h14v14H0z"/></svg>
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
			if cls == "item-name" {
				x, y, w, h, ok2 := rendering.BoxGeometry(o)
				textW := 0.0
				if ok2 {
					textW = layout.MeasureTextFunc("", 13, 400, "", el.TextContent())
				}
				fmt.Printf("item-name: x=%.0f y=%.0f w=%.1f h=%.1f textW=%.1f %s\n",
					cls, x, y, w, h, textW, map[bool]string{true: "OK", false: "TRUNCATED"}[w >= textW])
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
}
