// Command sbcheck2 probes conv-list / chat-messages scroll containers in detail:
// padding-box geometry, BoxContentSize, overflow style, and the resulting
// VerticalScrollbarMetrics gating decision.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(0)
	distDir := filepath.Join("cmd", "companion", "web-ui", "dist")
	absDist, _ := filepath.Abs(distDir)
	html, _ := os.ReadFile(filepath.Join(absDist, "index.html"))
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
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				return string(data), err
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				if err != nil {
					return "", err
				}
				return string(data), nil
			}
		}
	}
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
	wv.LoadHTML(string(html))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()

	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cls := el.GetAttribute("class")
			for _, want := range []string{"conv-list", "chat-messages", "file-explorer", "sidebar-content", "ws-section", "project-section", "msg-list", "rp-body"} {
				if strings.Contains(cls, want) {
					rb := renderBox(o)
					if rb == nil {
						fmt.Printf("%s: no box\n", want)
						continue
					}
					st := rb.Style()
					pb := rb.PaddingBoxRect()
					tw, th := rv.BoxContentSize(rb)
					ovY := "?"
					ovX := "?"
					if st != nil {
						ovY = overflowName(st.OverflowY)
						ovX = overflowName(st.OverflowX)
						fmt.Printf("%s: pb=(%.0f,%.0f %.0fx%.0f) content=(%.0f,%.0f) overflowX=%s overflowY=%s\n",
							want, pb.X, pb.Y, pb.Width, pb.Height, tw, th, ovX, ovY)
						fmt.Printf("   totalH=%.0f viewH=%.0f -> needV=%v (auto && totalH>viewH)\n",
							th, pb.Height, th > pb.Height)
						// show first few children geometry
						n := 0
						for c := o.FirstChild(); c != nil && n < 5; c = c.NextSibling() {
							if cbox := renderBox(c); cbox != nil {
								fr := cbox.FrameRect()
								fmt.Printf("   child: x=%.0f y=%.0f w=%.0f h=%.0f name=%s\n",
									fr.X, fr.Y, fr.Width, fr.Height, c.RenderName())
								n++
							}
						}
					} else {
						fmt.Printf("%s: no style\n", want)
					}
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
}

func overflowName(o style.OverflowType) string {
	switch o {
	case style.OverflowVisible:
		return "visible"
	case style.OverflowHidden:
		return "hidden"
	case style.OverflowAuto:
		return "auto"
	case style.OverflowScroll:
		return "scroll"
	}
	return fmt.Sprintf("?%d", int(o))
}

func renderBox(o rendering.RenderObject) *rendering.RenderBox {
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
