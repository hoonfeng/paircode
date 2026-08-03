// Command ev_probe verifies click dispatch: registers a JS click listener on
// ws-item, simulates host.handleClick's flow (HitTest deepest → DispatchEvent
// with bubbles), and checks whether the JS callback fired.
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
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}

	// 1) 注册 JS 监听（ws-item 和 ws-name 各一个）
	_, _ = wv.JSInterpreter().RunJS(`
		window.__evtLog = [];
		(function(){
			var items = document.querySelectorAll('.ws-item');
			if (items.length === 0) { window.__evtLog.push('no-ws-item'); return; }
			items[0].addEventListener('click', function(){ window.__evtLog.push('WS-ITEM-CLICKED'); });
			var name = items[0].querySelector('.ws-name');
			if (name) name.addEventListener('click', function(){ window.__evtLog.push('WS-NAME-CLICKED'); });
		})();
	`)

	// 2) 找 ws-item / ws-name 的几何
	var wsItemX, wsItemY, wsItemW, wsItemH float64
	var wsNameX, wsNameY, wsNameW, wsNameH float64
	var findWs func(o rendering.RenderObject)
	findWs = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			lb := o.LayoutBox()
			if lb != nil && rv.LayoutState() != nil {
				g := rv.LayoutState().GeometryForBox(lb)
				x, y, w, h := g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight()
				switch {
				case strings.Contains(cn, "ws-item") && wsItemW == 0:
					wsItemX, wsItemY, wsItemW, wsItemH = x, y, w, h
				case strings.Contains(cn, "ws-name") && wsNameW == 0:
					wsNameX, wsNameY, wsNameW, wsNameH = x, y, w, h
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			findWs(c)
		}
	}
	findWs(rendering.RenderObject(rv))
	fmt.Printf("[ev] ws-item x=%.0f y=%.0f w=%.0f h=%.0f\n", wsItemX, wsItemY, wsItemW, wsItemH)
	fmt.Printf("[ev] ws-name x=%.0f y=%.0f w=%.0f h=%.0f\n", wsNameX, wsNameY, wsNameW, wsNameH)

	// 3) 模拟 host.handleClick：HitTest deepest → DispatchEvent(click, bubbles)
	clickPoints := map[string][2]float64{
		"ws-name(text)":    {wsNameX + wsNameW/2, wsNameY + wsNameH/2},
		"ws-item blank":    {wsItemX + wsItemW - 6, wsItemY + wsItemH/2},
		"ws-item center":   {wsItemX + wsItemW/2, wsItemY + wsItemH/2},
		"ws-left(icon区)":  {wsItemX + 12, wsItemY + wsItemH/2},
	}
	for name, pt := range clickPoints {
		deepest := rendering.HitTest(rv, pt[0], pt[1], "")
		if deepest != nil {
			deepest.DispatchEvent(dom.NewMouseEvent(dom.EventClick, true, true, false))
		}
		// 读标记
		v, _ := wv.JSInterpreter().RunJS(`JSON.stringify(window.__evtLog)`)
		d := "nil"
		if deepest != nil {
			d = deepest.LocalName() + "." + strings.Split(deepest.GetAttribute("class"), " ")[0]
		}
		fmt.Printf("[ev] click %-18s pt=(%.0f,%.0f) deepest=%s log=%s\n", name, pt[0], pt[1], d, v.ToString())
		// 清空日志
		_, _ = wv.JSInterpreter().RunJS(`window.__evtLog = []`)
	}
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
