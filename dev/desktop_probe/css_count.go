// Command css_count counts the author style rules wb-ui actually parsed from
// the Vue app (dist CSS) vs the total in the file, revealing parser dropouts.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/css"
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

	// Total rules in the dist CSS file (count '{' at top level approximately).
	cssFiles, _ := filepath.Glob(filepath.Join(distDir, "assets", "*.css"))
	totalBrace := 0
	for _, cf := range cssFiles {
		data, _ := os.ReadFile(cf)
		totalBrace += strings.Count(string(data), "{")
	}
	fmt.Printf("dist CSS files: %d, approx total rules: %d\n", len(cssFiles), totalBrace)

	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.LoadHTML(string(htmlData))
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	rv := wv.RenderView()
	_ = rv

	// Parse the dist CSS directly with the css package to count rules.
	distCSS := filepath.Join(distDir, "assets", "style-HxyH992o.css")
	data, _ := os.ReadFile(distCSS)
	sheet := css.NewCSSStyleSheet()
	sheet.SetOrigin(css.OriginAuthor)
	p := css.NewParser(string(data))
	p.SetOrigin(css.OriginAuthor)
	for _, r := range p.ParseStyleSheet() {
		sheet.AppendRule(r)
	}
	rules := sheet.Rules()
	fmt.Printf("wb-ui css parser: %d rules from %d bytes\n", len(rules), len(data))
	selSet := map[string]bool{}
	for _, r := range rules {
		if sr, ok := r.(*css.StyleRule); ok {
			selSet[sr.Selectors.String()] = true
		}
	}
	for _, sel := range []string{".activity-bar", ".activity-bar button", ".sidebar", ".status-bar", ".right-panel", ".menu-btn"} {
		found := false
		for s := range selSet {
			if s == sel {
				found = true
			}
		}
		fmt.Printf("  selector %-24q parsed=%v\n", sel, found)
	}
}
