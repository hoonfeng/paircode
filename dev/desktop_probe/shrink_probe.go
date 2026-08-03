// Command shrink_probe: minimal flex row with a long ellipsis item — verifies
// whether flex shrink applies (item-name should shrink below its text width).
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
* { box-sizing: border-box; }
.item-row { display:flex; align-items:center; gap:3px; padding:2px 4px; white-space:nowrap; width:278px; }
.chevron { width:12px; flex-shrink:0; }
.icon { width:14px; flex-shrink:0; }
.item-name { overflow:hidden; text-overflow:ellipsis; font-size:13px; }
</style>
</head><body>
<div class="item-row">
  <span class="chevron">&#9654;</span>
  <svg class="icon" width="14" height="14"><path d="M0 0h14v14H0z"/></svg>
  <span class="item-name">main_very_long_file_name_for_testing.txt</span>
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
				fmt.Printf("%s: x=%.0f y=%.0f w=%.1f h=%.1f\n", cls, x, y, w, h)
				_ = ok2
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
	// 读 layout 几何
	if ls := rv.LayoutState(); ls != nil {
		var lw2 func(o rendering.RenderObject)
		lw2 = func(o rendering.RenderObject) {
			if el, ok := o.Node().(*dom.Element); ok {
				cls := el.GetAttribute("class")
				if cls == "item-name" || cls == "item-row" {
					if lb := o.LayoutBox(); lb != nil {
						g := ls.GeometryForBox(lb)
						fmt.Printf("layout %s: contentW=%.1f borderBoxW=%.1f\n", cls, g.ContentWidth(), g.BorderBoxWidth())
					}
				}
			}
			for c := o.FirstChild(); c != nil; c = c.NextSibling() {
				lw2(c)
			}
		}
		lw2(rv)
	}
}

func asRB(o rendering.RenderObject) *rendering.RenderBox {
	switch v := o.(type) {
	case *rendering.RenderBox:
		return v
	case *rendering.RenderBlock:
		return &v.RenderBox
	case *rendering.RenderBlockFlow:
		return &v.RenderBlock.RenderBox
	case *rendering.RenderView:
		return &v.RenderBlockFlow.RenderBlock.RenderBox
	}
	return nil
}
