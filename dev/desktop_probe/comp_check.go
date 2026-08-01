// Command comp_check dumps geometry of sidebar/explorer/chat-empty internals.
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

	fmt.Println("=== SIDEBAR / EXPLORER / CHAT EMPTY INTERNALS ===")
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cls := el.GetAttribute("class")
			if strings.Contains(cls, "ws-") || strings.Contains(cls, "explorer") ||
				strings.Contains(cls, "file-") || strings.Contains(cls, "welcome") ||
				strings.Contains(cls, "chat-empty") || strings.Contains(cls, "tree") ||
				strings.Contains(cls, "folder") || strings.Contains(cls, "item") ||
				strings.Contains(cls, "explorer-") {
				x, y, w, h, ok2 := rendering.BoxGeometry(o)
				if ok2 && w > 0 && h > 0 {
					txt := ""
					if t := el.TextContent(); t != "" {
						txt = strings.TrimSpace(t)
						if len(txt) > 18 {
							txt = txt[:18]
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
