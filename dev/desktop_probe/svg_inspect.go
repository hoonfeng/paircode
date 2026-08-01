// Command svg_inspect dumps the first activity-bar button's SVG markup and
// what wb-ui's SVG parser produced (shape count / kinds), to find why the
// icon body is missing.
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

	// Find the first svg-icon (activity bar).
	var svgEl *dom.Element
	var find func(n interface{})
	_ = find
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok && el.LocalName() == "svg" && svgEl == nil {
			svgEl = el
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
	if svgEl == nil {
		fmt.Println("no svg found")
		return
	}
	fmt.Println("=== first svg ===")
	fmt.Printf("viewBox=%q width=%q height=%q\n", svgEl.GetAttribute("viewBox"), svgEl.GetAttribute("width"), svgEl.GetAttribute("height"))
	fmt.Println("children:")
	var dump func(n dom.Node, indent string)
	dump = func(n dom.Node, indent string) {
		if el, ok := n.(*dom.Element); ok {
			d := el.GetAttribute("d")
			preview := d
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			fmt.Printf("%s<%s> fill=%q d=%q\n", indent, el.LocalName(), el.GetAttribute("fill"), preview)
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			dump(c, indent+"  ")
		}
	}
	dump(svgEl, "  ")
}
