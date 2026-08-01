// Command send_svg dumps the send-btn's SVG icon markup (fill/stroke/path)
// from the wb-ui render.
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

	// Find the send-btn's svg (x≈995, y≈740).
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok && el.LocalName() == "svg" {
			x, y, _, _, ok2 := rendering.BoxGeometry(o)
			if ok2 && x > 990 && x < 1000 && y > 735 && y < 745 {
				fmt.Println("=== send-btn svg ===")
				fmt.Printf("class=%q viewBox=%q fill=%q\n", el.GetAttribute("class"), el.GetAttribute("viewBox"), el.GetAttribute("fill"))
				var dump func(n dom.Node, ind string)
				dump = func(n dom.Node, ind string) {
					if e, ok := n.(*dom.Element); ok {
						attrs := []string{"d", "fill", "stroke", "stroke-width", "x1", "y1", "x2", "y2", "points", "cx", "cy", "r", "color"}
						parts := []string{}
						for _, a := range attrs {
							if v := e.GetAttribute(a); v != "" {
								parts = append(parts, a+"="+v)
							}
						}
						fmt.Printf("  %s<%s> %s\n", ind, e.LocalName(), strings.Join(parts, " "))
					}
					for c := n.FirstChild(); c != nil; c = c.NextSibling() {
						dump(c, ind+"  ")
					}
				}
				dump(el, "")
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
}
