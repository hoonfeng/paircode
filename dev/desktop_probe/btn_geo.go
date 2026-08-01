// Command btn_geo dumps the chat-input button row geometry (ibb-btns,
// send-btn, input-bottom-bar, input-wrapper) from the companion frontend.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/dom"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

func setupLoaders(wv *webkit.WebView, distDir string) {
	absDist, _ := filepath.Abs(distDir)
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
				re := regexp.MustCompile(`\[data-v-[a-f0-9]+\]`)
				return re.ReplaceAllString(string(data), ""), nil
			}
		}
	}
}

func main() {
	log.SetFlags(0)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, _ := os.ReadFile(distDir + "/index.html")

	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.LoadHTML(string(htmlData))
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()

	fmt.Println("=== CHAT INPUT BUTTON ROW ===")
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cls := el.GetAttribute("class")
			if strings.Contains(cls, "chat-input-area") || strings.Contains(cls, "input-wrapper") ||
				strings.Contains(cls, "input-bottom-bar") || strings.Contains(cls, "ibb-btns") ||
				strings.Contains(cls, "send-btn") || strings.Contains(cls, "stop-btn") ||
				strings.Contains(cls, "obtn") || strings.Contains(cls, "chat-input") {
				x, y, w, h, ok2 := rendering.BoxGeometry(o)
				if ok2 {
					txt := ""
					if t := el.TextContent(); t != "" {
						txt = strings.TrimSpace(t)
						if len(txt) > 14 {
							txt = txt[:14]
						}
					}
					// Box model via layout state
					lb := o.LayoutBox()
					if lb != nil {
						if g := rv.LayoutState().GeometryForBox(lb); g != nil {
							fmt.Printf("  .%s xy=(%.0f,%.0f) wh=(%.0f,%.0f) m=(%.0f,%.0f,%.0f,%.0f) p=(%.0f,%.0f,%.0f,%.0f) b=(%.0f,%.0f,%.0f,%.0f) text=%q\n",
								cls, x, y, w, h,
								g.MarginBefore(), g.MarginAfter(), g.MarginStart(), g.MarginEnd(),
								g.PaddingTop(), g.PaddingRight(), g.PaddingBottom(), g.PaddingLeft(),
								g.BorderTop(), g.BorderRight(), g.BorderBottom(), g.BorderLeft(), txt)
						}
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
