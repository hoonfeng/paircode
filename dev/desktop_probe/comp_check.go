// Command comp_check dumps geometry of chat-area internals with box model.
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
	st := rv.LayoutState()

	fmt.Println("=== CHAT-AREA BOX MODEL ===")
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cls := el.GetAttribute("class")
			if strings.Contains(cls, "chat-area") || strings.Contains(cls, "chat-messages") ||
				strings.Contains(cls, "chat-input-area") || strings.Contains(cls, "chat-empty") {
				lb := o.LayoutBox()
				if lb == nil {
					return
				}
				g := st.GeometryForBox(lb)
				if g == nil {
					return
				}
				cs := o.Style()
				grow, shrink := -1.0, -1.0
				if cs != nil {
					grow, shrink = cs.FlexGrow, cs.FlexShrink
				}
				fmt.Printf("  .%s xy=(%.0f,%.0f) cw=%.0f ch=%.0f bbw=%.0f bbh=%.0f grow=%v shrink=%v\n",
					cls, g.Left(), g.Top(),
					g.ContentWidth(), g.ContentHeight(),
					g.BorderBoxWidth(), g.BorderBoxHeight(),
					grow, shrink)
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
}
