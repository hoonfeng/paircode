// Command welcome_geo dumps geometry of the welcome-area elements in the wb-ui
// render (editor area with no file open) to find the text overlap.
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
	state := rv.LayoutState()

	fmt.Println("=== WELCOME AREA GEOMETRY (main-area subtree) ===")
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o.Node() != nil {
			el, ok := o.Node().(*dom.Element)
			cls := ""
			tag := "?"
			if ok {
				cls = el.GetAttribute("class")
				tag = el.LocalName()
			}
			if strings.Contains(cls, "welcome") || strings.Contains(cls, "main-area") ||
				strings.Contains(cls, "editor") || tag == "body" || tag == "html" {
				lb := o.LayoutBox()
				if lb != nil && state != nil {
					g := state.GeometryForBox(lb)
					var disp, flexDir string
					if st := lb.Style(); st != nil {
						disp = st.Display.String()
						flexDir = st.FlexDirection
					}
					fmt.Printf("  %s.%s xy=(%.0f,%.0f) wh=(%.0f,%.0f) disp=%s flexdir=%q\n",
						tag, cls[:min(28, len(cls))], g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(), disp, flexDir)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
