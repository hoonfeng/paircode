// Command minflex_probe: 最小化复现 sidebar-header height:32px 在 flex column 中失效的问题
package main

import (
	"fmt"
	"os"
	"strings"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

func main() {
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style}, text)
	}
	layout.FontMetricsFunc = func(family string, size float64, weight int, style string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	html := `<!DOCTYPE html><html><head><style>
* { box-sizing: border-box; }
body { margin:0; }
.sidebar { display:flex; flex-direction:column; width:279px; height:748px; background:#161b22; }
.sidebar-header { height:32px; display:flex; align-items:center; padding:0 12px; font-size:11px; color:#8b949e; border-bottom:1px solid #30363d; flex-shrink:0; }
.sidebar-content { flex:1; overflow:auto; }
.tb { display:flex; align-items:center; padding:4px 8px; border-bottom:1px solid #30363d; flex-shrink:0; }
</style></head><body>
<div class="sidebar">
  <div class="sidebar-header"><span>文件浏览器</span></div>
  <div class="tb"><span>工作区</span></div>
  <div class="sidebar-content"><span>内容区域</span></div>
</div>
</body></html>`

	wv := webkit.NewWebView()
	wv.Resize(1280, 800)
	if err := wv.LoadHTML(html); err != nil {
		fmt.Println("LoadHTML err:", err)
	}
	wv.EnsureLayout()

	rv := wv.RenderView()
	if rv == nil {
		fmt.Println("no render view")
		return
	}
	state := rv.LayoutState()
	var walk func(ro rendering.RenderObject, depth int)
	walk = func(ro rendering.RenderObject, depth int) {
		if ro == nil {
			return
		}
		if n := ro.Node(); n != nil {
			if el, ok := n.(*dom.Element); ok {
				cls := el.GetAttribute("class")
				lb := ro.LayoutBox()
				if lb != nil && state != nil {
					g := state.GeometryForBox(lb)
					cs := ro.Style()
					ht := "?"
					if cs != nil {
						ht = cs.Height.String()
					}
					fmt.Printf("[BOX] class=%-16s h=%s renderH=%.0f x=%.0f y=%.0f w=%.0f\n", cls, ht, g.BorderBoxHeight(), g.Left(), g.Top(), g.BorderBoxWidth())
				}
			}
		}
		for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, depth+1)
		}
	}
	for c := rv.FirstChild(); c != nil; c = c.NextSibling() {
		walk(c, 0)
	}
	_ = os.Stderr
	_ = strings.TrimSpace
}
