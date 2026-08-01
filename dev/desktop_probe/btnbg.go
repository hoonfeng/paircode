// Command btnbg dumps the computed background/color of the first activity-bar
// button to explain the #f0f0f0 (UA buttonface) vs Edge #0d1117 mismatch.
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
	// Mirror host.go: after JS mounts Vue, rebuild the render tree so
	// runtime-injected <style> elements (Vue scoped CSS) are collected.
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()

	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cls := el.GetAttribute("class")
			if strings.Contains(cls, "activity-bar") || (el.LocalName() == "button" && o.Parent() != nil) {
				if el.LocalName() == "button" {
					lb := o.LayoutBox()
					st := lb.Style()
					bg := st.BackgroundColor
					inline := el.GetAttribute("style")
					fmt.Printf("button cls=%q bg=#%02x%02x%02x image=[%s] disp=%v inline=[%.60s]\n",
						cls, bg.R, bg.G, bg.B, st.BackgroundImage, st.Display, inline)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)
}
