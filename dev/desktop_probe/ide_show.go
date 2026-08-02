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
				// Keep Vue scoped [data-v-...] selectors: DOM carries data-v attrs.
				return string(data), nil
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

	// ★ 桌面环境标记：模拟 /api/* 返回工作区 + 文件数据，让文件树渲染
	//   （真实用户场景：三个滚动条、item 点击、文本省略）。
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		rt.RunJS(`(function(){
			window.__DESKTOP_MODE__ = true;
			window.__origFetch = window.fetch;
			function makeResp(obj) {
				return Promise.resolve({
					ok: true, status: 200, statusText: 'OK',
					json: function() { return Promise.resolve(obj); },
					text: function() { return Promise.resolve(JSON.stringify(obj)); }
				});
			}
			window.fetch = function(url, opts) {
				var u = String(url);
				if (u.indexOf('/api/health') === 0) {
					return makeResp({status: 'ok', workspace: 'F:\\syproject\\gou-ide', folders: ['F:\\syproject\\gou-ide']});
				}
				if (u.indexOf('/api/fs/list') === 0) {
					var entries = [
						{name: 'cmd', isDir: true, size: 0},
						{name: 'go.mod', isDir: false, size: 100},
						{name: 'internal', isDir: true, size: 0},
						{name: 'pkg', isDir: true, size: 0},
						{name: 'companion.exe', isDir: false, size: 300},
						{name: 'config', isDir: true, size: 0},
						{name: 'main_very_long_file_name_for_testing.txt', isDir: false, size: 10}
					];
					return makeResp(entries);
				}
				if (u.indexOf('/api/settings') === 0) {
					return makeResp({ok: true, recentProjects: ['F:\\syproject\\gou-ide'], workspaceFolderLists: {}});
				}
				if (u.indexOf('/api/') === 0) {
					return makeResp({ok: true, data: []});
				}
				return window.__origFetch.apply(window, arguments);
			};
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
