// Command interact_show renders the standalone interaction test page
// (cmd/desktop/web-ui/interact_test.html) through the wb-ui desktop host
// in a real window, so the user can inspect buttons / inputs / textarea
// caret / switches / tabs / dialog rendering and interaction.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/app"
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
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "desktop", "web-ui")
	htmlPath := filepath.Join(distDir, "interact_test.html")
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot read %s: %v\n", htmlPath, err)
		os.Exit(1)
	}
	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()
	wv.LoadHTML(string(data))

	host, err := app.NewHost(wv, 1280, 860, "wb-ui 交互测试")
	if err != nil {
		log.Fatalf("create window: %v", err)
	}
	log.Println("[interact_show] 窗口已启动，拖动/点击测试…")
	host.Run()
}
