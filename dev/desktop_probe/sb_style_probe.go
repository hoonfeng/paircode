// Command sb_style_probe loads the REAL companion frontend, switches to the
// Git panel, and dumps .git-sections' scrollbar palette (webkit-scrollbar
// properties resolved from GitPanel.vue styles) + scrollbar geometry.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	absDist, _ := filepath.Abs(distDir)
	htmlData, err := os.ReadFile(filepath.Join(absDist, "index.html"))
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style2, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}, text)
	}
	layout.FontMetricsFunc = func(family string, size float64, weight int, style2 string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}
	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	for i := 0; i < 8; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
	}
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 800); })`)

	// 切到 git 面板（Vue 应用内切换 activeActivity）
	_, _ = wv.JSInterpreter().RunJS(`
		try {
			var app = document.querySelector('#app').__vue_app__;
			if (app && app._instance && app._instance.proxy && app._instance.proxy.switchActivity) {
				app._instance.proxy.switchActivity('git');
			} else {
				document.querySelectorAll('.activity-btn')[1] && document.querySelectorAll('.activity-btn')[1].click();
			}
		} catch(e) { console.log('[sb] switch err ' + (e && e.message || e)); }
	`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 800); })`)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}

	// dump .git-sections 的 scrollbar 属性 + 几何
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			if strings.Contains(cn, "git-sections") || strings.Contains(cn, "history-list") ||
				strings.Contains(cn, "project-section") || strings.Contains(cn, "ws-section") {
				if st := o.Style(); st != nil {
					lb := o.LayoutBox()
					g := ""
					if lb != nil && rv.LayoutState() != nil {
						gg := rv.LayoutState().GeometryForBox(lb)
						g = fmt.Sprintf("x=%.0f y=%.0f w=%.0f h=%.0f", gg.Left(), gg.Top(), gg.BorderBoxWidth(), gg.BorderBoxHeight())
					}
					fmt.Printf("[sb] %s %s scrollbar-width=%q webkit-w=%q thumb-c=%q track-c=%q radius=%q ovfY=%d\n",
						cn, g,
						st.GetProperty("scrollbar-width"),
						st.GetProperty("-webkit-scrollbar-width"),
						st.GetProperty("-webkit-scrollbar-thumb-color"),
						st.GetProperty("-webkit-scrollbar-track-color"),
						st.GetProperty("-webkit-scrollbar-thumb-radius"),
						st.OverflowY)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	if out := wv.ConsoleOutput(); out != "" {
		fmt.Println("[CONSOLE]")
		fmt.Println(out)
	}
	_ = style.NewComputedStyle
}

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
				return string(data), nil
			}
		}
	}
}
