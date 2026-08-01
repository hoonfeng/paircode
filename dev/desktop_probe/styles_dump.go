// Command styles_dump lists all <style> elements in the document after Vue
// mounts (runtime-injected scoped CSS) and whether any rule targets buttons.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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

	doc := wv.Document()
	styles := doc.GetElementsByTagName("style")
	fmt.Printf("=== <style> elements: %d ===\n", len(styles))
	btnRules := 0
	for i, s := range styles {
		txt := s.TextContent()
		preview := txt
		if len(preview) > 160 {
			preview = preview[:160]
		}
		fmt.Printf("[style %d] len=%d\n  %s\n", i, len(txt), strings.ReplaceAll(preview, "\n", " "))
		if strings.Contains(txt, "button") {
			btnRules++
		}
	}
	fmt.Printf("=== styles containing 'button': %d ===\n", btnRules)
}
