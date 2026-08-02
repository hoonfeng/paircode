// Command ide_show launches the REAL web IDE frontend (cmd/companion/web-ui/dist
// — the Vue 3 SPA served to browsers) through the wb-ui WebView in a real
// window, so every component (activity bar, sidebar file tree, tabs, buttons,
// inputs, switches, scrollbars, dialogs, status bar…) can be inspected and
// interacted with exactly as in the browser.
//
// Run: go run ./dev/desktop_probe/ide_show.go
package main

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/app"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

func ideSetupLoaders(wv *webkit.WebView, distDir string) {
	absDist, _ := filepath.Abs(distDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				if err != nil {
					log.Printf("[SCRIPT] src=%q err=%v", src, err)
				}
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
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, err := os.ReadFile(distDir + "/index.html")
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}
	log.Printf("[ide_show] distDir=%s html=%d bytes", distDir, len(htmlData))

	// Mirror host.go: initialize the FontManager so CJK measurement matches
	// painting (companion frontend is Chinese-heavy).
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style}, text)
	}
	layout.FontMetricsFunc = func(family string, size float64, weight int, style string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	wv := webkit.NewWebView()
	ideSetupLoaders(wv, distDir)
	_ = wv.JSInterpreter()

	// ★ 桌面环境标记：companion 前端默认走 /api/* fetch；无桥接时 fetch 会
	//   失败——注入占位 fetch 让 /api/* 返回空 JSON，组件仍能渲染初始状态。
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		rt.RunJS(`(function(){
			window.__DESKTOP_MODE__ = true;
			if (!window.__origFetch) {
				window.__origFetch = window.fetch;
				window.fetch = function(url, opts) {
					var u = String(url);
					if (u.indexOf('/api/') === 0) {
						return Promise.resolve(new Response('{"ok":true,"data":[]}', {
							status: 200, headers: {'Content-Type':'application/json'}
						}));
					}
					return window.__origFetch.apply(window, arguments);
				};
			}
		})()`)
	}

	wv.LoadHTML(string(htmlData))
	if out := wv.ConsoleOutput(); out != "" {
		log.Printf("[CONSOLE] %s", out)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	host, err := app.NewHost(wv, 1280, 800, "wb-ui 真实 IDE 验证")
	if err != nil {
		log.Fatalf("create window: %v", err)
	}
	log.Println("[ide_show] 窗口已启动（真实 companion 前端）；等待 Vue 挂载…")
	host.Run()
}
