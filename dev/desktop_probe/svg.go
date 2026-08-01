// Command desktop_svg inspects how wb-ui sizes <svg> icons in the Vue app.
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
	wv.EnsureLayout()

	rv := wv.RenderView()
	state := rv.LayoutState()
	fmt.Println("=== SVG ELEMENTS (first 15) ===")
	n := 0
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if n >= 15 {
			return
		}
		if el, ok := o.Node().(*dom.Element); ok && el.LocalName() == "svg" {
			lb := o.LayoutBox()
			if lb != nil && state != nil {
				g := state.GeometryForBox(lb)
				st := lb.Style()
				fmt.Printf("  svg.%s xy=(%.0f,%.0f) wh=(%.0f,%.0f) attr[w=%s h=%s vb=%s] css[w=%v%s] disp=%v\n",
					clsShort(el.GetAttribute("class")),
					g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(),
					el.GetAttribute("width"), el.GetAttribute("height"), el.GetAttribute("viewBox"),
					st.Width.Value, st.Width.Unit, st.Display)
				n++
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
}

func clsShort(s string) string {
	if len(s) > 24 {
		return s[:24]
	}
	return s
}
