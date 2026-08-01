// Command probe_vue_style probes what style APIs Vue's runtime relies on.
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
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, _ := os.ReadFile(distDir + "/index.html")

	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.LoadHTML(string(htmlData))

	// Probe available style APIs in the wb-ui DOM/jsc environment.
	probes := []string{
		`"adoptedStyleSheets" in document`,
		`typeof document.adoptedStyleSheets`,
		`typeof document.createElement('style').sheet`,
		`typeof CSSStyleSheet`,
		`typeof document.styleSheets`,
		`typeof document.head.appendChild`,
		`typeof document.head.insertBefore`,
		`typeof document.head.prepend`,
		`typeof document.head.insertAdjacentHTML`,
		`typeof Element.prototype.after`,
		`typeof document.head.after`,
	}
	for _, p := range probes {
		res, err := wv.EvalJS(p)
		fmt.Printf("%-55s → %v err=%v\n", p, res, err)
	}
}
