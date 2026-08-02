// Command sb_probe loads the REAL companion frontend through wb-ui and dumps
// every scroll container (overflow:auto/scroll) with its content vs viewport
// sizes, so we can see exactly which boxes draw scrollbars (the "three
// scrollbars" report) and where text gets ellipsized.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(log.Ltime)
	distDir := filepath.Join("cmd", "companion", "web-ui", "dist")
	absDist, err := filepath.Abs(distDir)
	if err != nil {
		log.Fatalf("abs: %v", err)
	}
	html, err := os.ReadFile(filepath.Join(absDist, "index.html"))
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}
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
	treeSetupLoaders(wv, distDir)
	// 与 ide_tree_dump 相同的 fetch 拦截（空数据空态）
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		rt.RunJS(`(function(){
			if (!window.__origFetch) {
				window.__origFetch = window.fetch;
				window.fetch = function(url, opts) {
					var u = String(url);
					if (u.indexOf('/api/') === 0) {
						return Promise.resolve(new Response('{"ok":true,"data":[]}', {
							status: 200, headers: {'Content-Type':'application/json'}
						}));
					}
					return window.__origFetch.apply(window, arguments);
				};
			}
		})()`)
	}
	_ = wv.JSInterpreter()
	if err := wv.LoadHTML(string(html)); err != nil {
		log.Fatalf("LoadHTML: %v", err)
	}
	// 等待 Vue 挂载 + 布局
	rt := wv.JSInterpreter()
	_, _ = rt.RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)

	wv.RebuildRenderTree()
	wv.EnsureLayout()

	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}

	// 遍历渲染树，收集所有 overflow:auto/scroll 的 box
	var walk func(o rendering.RenderObject, depth int)
	walk = func(o rendering.RenderObject, depth int) {
		rb, ok := o.(*rendering.RenderBox)
		if ok {
			st := rb.Style()
			if st != nil {
				ox, oy := st.OverflowX, st.OverflowY
				if ox == style.OverflowAuto || ox == style.OverflowScroll ||
					oy == style.OverflowAuto || oy == style.OverflowScroll {
					cw, ch := rv.BoxContentSize(rb)
					pb := rb.PaddingBoxRect()
					padT := lengthPx(st.PaddingTop)
					padB := lengthPx(st.PaddingBottom)
					padL := lengthPx(st.PaddingLeft)
					padR := lengthPx(st.PaddingRight)
					viewW := pb.Width - padL - padR
					viewH := pb.Height - padT - padB
					name := o.RenderName()
					if el, ok2 := o.Node().(*dom.Element); ok2 {
						if cls := el.ClassName(); cls != "" {
							name += "." + cls
						}
					}
					fmt.Printf("[scroll] %s x=%.0f y=%.0f w=%.0f h=%.0f | content w=%.1f h=%.1f | view w=%.1f h=%.1f | ovf x=%d y=%d %s\n",
						name, pb.X, pb.Y, pb.Width, pb.Height,
						cw, ch, viewW, viewH, ox, oy,
						scrollTag(ox, oy, cw, ch, viewW, viewH))
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, depth+1)
		}
	}
	walk(rv, 0)
}

func lengthPx(l style.Length) float64 {
	if l.Unit == "px" {
		return l.Value
	}
	return 0
}

func scrollTag(ox, oy style.OverflowType, cw, ch, viewW, viewH float64) string {
	v := ""
	if (oy == style.OverflowScroll || (oy == style.OverflowAuto && ch > viewH)) && oy != style.OverflowHidden {
		v += " VSB"
	}
	if (ox == style.OverflowScroll || (ox == style.OverflowAuto && cw > viewW)) && ox != style.OverflowHidden {
		v += " HSB"
	}
	return v
}
