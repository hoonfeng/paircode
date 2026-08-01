// Command jsstyle tests whether runtime <style> injection works: create a
// style element via JS, append it, and check the DOM picks it up.
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

	// 1. Try appending a style element via JS.
	res, err := wv.EvalJS(`
		(function() {
			try {
				var s = document.createElement('style');
				s.id = 'jstest';
				s.textContent = '.activity-bar button { background: transparent !important; } button { background: transparent !important; }';
				document.head.appendChild(s);
				var count = document.getElementsByTagName('style').length;
				var has = !!document.getElementById('jstest');
				return 'count=' + count + ' has=' + has;
			} catch(e) { return 'ERR: ' + e.message; }
		})()
	`)
	fmt.Printf("appendChild result: %v err=%v\n", res, err)

	// 2. Rebuild + check the style count and whether a button resolves to red.
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	styles := wv.Document().GetElementsByTagName("style")
	fmt.Printf("after rebuild: style elements=%d\n", len(styles))

	// 3. Check a button's computed background after injection.
	btn, err := wv.EvalJS(`
		(function() {
			var b = document.querySelector('.activity-bar button');
			if (!b) return 'no button';
			return 'bg=' + getComputedStyle(b).backgroundColor;
		})()
	`)
	fmt.Printf("button computed bg: %v err=%v\n", btn, err)
}
