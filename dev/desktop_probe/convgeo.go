// Command convgeo probes the real conv-list geometry in the desktop render
// tree: padding-box position/size + child count + content size, to see why
// the conversation list is not visibly rendering (or scrolling).
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
			for _, want := range []string{"conv-list", "conv-sidebar", "conv-item", "conv-title", "rp-body", "right-panel"} {
				if strings.Contains(cls, want) {
					rb := renderBox(o)
					if rb == nil {
						fmt.Printf("%s.%s: no box\n", el.LocalName(), cls)
						continue
					}
					pb := rb.PaddingBoxRect()
					fr := rb.FrameRect()
					tw, th := rv.BoxContentSize(rb)
					cs := rb.Style()
					ovY := "?"
					if cs != nil {
						ovY = fmt.Sprintf("%d", cs.OverflowY)
					}
					fmt.Printf("%s.%s: frame=(%.0f,%.0f %.0fx%.0f) padding=(%.0f,%.0f %.0fx%.0f) content=(%.0f,%.0f) ovY=%s children=",
						el.LocalName(), cls, fr.X, fr.Y, fr.Width, fr.Height, pb.X, pb.Y, pb.Width, pb.Height, tw, th, ovY)
					n := 0
					for c := o.FirstChild(); c != nil; c = c.NextSibling() {
						n++
						if n <= 3 {
							if cbox := renderBox(c); cbox != nil {
								cfr := cbox.FrameRect()
								fmt.Printf("[%s:%.0f,%.0f %.0fx%.0f] ", c.RenderName(), cfr.X, cfr.Y, cfr.Width, cfr.Height)
							} else {
								fmt.Printf("[%s:no-box] ", c.RenderName())
							}
						}
					}
					fmt.Printf("total=%d\n", n)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
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
