// Command desktop_overflow finds render objects whose bottom edge exceeds the
// viewport height (800) in the desktop Vue app.
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
	wv.EnsureLayout()

	rv := wv.RenderView()
	state := rv.LayoutState()
	fmt.Println("=== OBJECTS OVERFLOWING VIEWPORT (bottom > 800) ===")
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o.Node() != nil {
			lb := o.LayoutBox()
			if lb != nil && state != nil {
				g := state.GeometryForBox(lb)
				var ov string
				if st := lb.Style(); st != nil {
					ov = st.OverflowX.String() + "/" + st.OverflowY.String()
				}
				if strings.Contains(clsOf(o), "conv-sidebar") || strings.Contains(clsOf(o), "conv-stats") || strings.Contains(clsOf(o), "conv-list") {
					fmt.Printf("  [ov] %-24s xy=(%.0f,%.0f) wh=(%.0f,%.0f) overflow=%s\n",
						clsOf(o)[:min(30, len(clsOf(o)))], g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(), ov)
				}
				bottom := g.Top() + g.BorderBoxHeight()
				if bottom > 800 {
					fmt.Printf("  %s.%s xy=(%.0f,%.0f) wh=(%.0f,%.0f) bottom=%.0f ov=%s\n",
						tagOf(o), clsOf(o)[:min(30, len(clsOf(o)))], g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(), bottom, ov)
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

func clsOf(o rendering.RenderObject) string {
	if el, ok := o.Node().(*dom.Element); ok {
		return el.GetAttribute("class")
	}
	return ""
}

func tagOf(o rendering.RenderObject) string {
	if el, ok := o.Node().(*dom.Element); ok {
		return el.LocalName()
	}
	return "?"
}
