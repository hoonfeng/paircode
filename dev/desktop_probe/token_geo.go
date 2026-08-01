// Command token_geo dumps geometry + text of the conv-stats (Token 统计)
// panel in the wb-ui desktop render.
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
	distDir := filepath.Join(wd, "cmd", "desktop", "web-ui", "dist")
	htmlData, _ := os.ReadFile(distDir + "/index.html")

	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.LoadHTML(string(htmlData))
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()

	fmt.Println("=== CONV-STATS (Token 统计) AREA ===")
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cls := el.GetAttribute("class")
			if strings.Contains(cls, "conv-stats") || strings.Contains(cls, "cs-") ||
				strings.Contains(cls, "cache-ring") || strings.Contains(cls, "conv-list") ||
				strings.Contains(cls, "conv-sidebar") || strings.Contains(cls, "conv-item") {
				x, y, w, h, ok2 := rendering.BoxGeometry(o)
				if ok2 {
					txt := ""
					if t := el.TextContent(); t != "" {
						txt = strings.TrimSpace(t)
						if len(txt) > 24 {
							txt = txt[:24]
						}
					}
					fmt.Printf("  .%s xy=(%.0f,%.0f) wh=(%.0f,%.0f) text=%q\n", cls, x, y, w, h, txt)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
}
