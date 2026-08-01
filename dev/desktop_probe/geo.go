// Command desktop_geo dumps layout geometry of the desktop IDE (Vue app)
// key elements, mirroring cmd/desktop's load path.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/dom"
	"wb-ui/layout"
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
	wv.EnsureLayout()

	rv := wv.RenderView()
	state := rv.LayoutState()
	fmt.Println("=== RENDER TREE GEOMETRY (key classes) ===")
	var walk func(o rendering.RenderObject, depth int)
	walk = func(o rendering.RenderObject, depth int) {
		if depth > 30 {
			return
		}
		if o.Node() != nil {
			var cls string
			if el, ok := o.Node().(*dom.Element); ok {
				cls = el.GetAttribute("class")
			}
			if cls != "" && (strings.Contains(cls, "app-root") ||
				strings.Contains(cls, "activity-bar") ||
				strings.Contains(cls, "sidebar") ||
				strings.Contains(cls, "main-area") ||
				strings.Contains(cls, "right-panel") ||
				strings.Contains(cls, "status-bar") ||
				strings.Contains(cls, "titlebar") ||
				strings.Contains(cls, "file-explorer") ||
				strings.Contains(cls, "editor") ||
				strings.Contains(cls, "chat") ||
				strings.Contains(cls, "conversation") ||
				strings.Contains(cls, "rp-body")) {
				lb := o.LayoutBox()
				if lb != nil && state != nil {
					g := state.GeometryForBox(lb)
					var disp string
					var gtc, gtr, gta string
					if st := lb.Style(); st != nil {
						disp = st.Display.String()
						gtc = st.GridTemplateColumns
						gtr = st.GridTemplateRows
						gta = st.GridTemplateAreas
					}
					if cls == "app-root" || cls == "right-panel" || cls == "main-area" {
						fmt.Printf("  %-28s xy=(%.0f,%.0f) wh=(%.0f,%.0f) disp=%s cols=[%s] rows=[%s]\n",
							cls[:min(28, len(cls))], g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(), disp, gtc, gtr)
						if gta != "" {
							fmt.Printf("    areas=[%s]\n", gta)
						}
					}
					fmt.Printf("  %-28s xy=(%.0f,%.0f) wh=(%.0f,%.0f) disp=%s\n",
						cls[:min(28, len(cls))], g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(), disp)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, depth+1)
		}
	}
	walk(rv, 0)

	// Count render objects.
	n := 0
	var cnt func(o rendering.RenderObject)
	cnt = func(o rendering.RenderObject) {
		n++
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			cnt(c)
		}
	}
	cnt(rv)
	fmt.Printf("=== total render objects: %d ===\n", n)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var _ = layout.LayoutState{}
