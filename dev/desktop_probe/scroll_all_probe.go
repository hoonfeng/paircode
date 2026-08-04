// Command scroll_all_probe loads the REAL companion frontend through wb-ui
// (same startup path as real_probe / scroll_diag_probe) and dumps the scroll
// metrics of EVERY overflow:auto/scroll container in the render tree:
// view (viewport len), total (content len), maxSy (scrollable range).
// Used to find "content clipped at bottom, cannot scroll to it" bugs that
// affect ALL scroll containers (chat, file tree, sidebar, etc).
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
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

func asRB(o rendering.RenderObject) *rendering.RenderBox {
	switch v := o.(type) {
	case *rendering.RenderBox:
		return v
	case *rendering.RenderBlock:
		return &v.RenderBox
	case *rendering.RenderBlockFlow:
		return &v.RenderBlock.RenderBox
	}
	return nil
}

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

	var walk func(o rendering.RenderObject, depth int)
	walk = func(o rendering.RenderObject, depth int) {
		if o == nil {
			return
		}
		if rb := asRB(o); rb != nil && rb.Style() != nil {
			st := rb.Style()
			isScroll := st.OverflowY == style.OverflowAuto || st.OverflowY == style.OverflowScroll ||
				st.OverflowX == style.OverflowAuto || st.OverflowX == style.OverflowScroll
			if isScroll {
				g := rv.LayoutState().GeometryForBox(rb.LayoutBox())
				m := rendering.VerticalScrollbarMetrics(rv, rb)
				cn, id := "", ""
				if el, ok := rb.Node().(*dom.Element); ok {
					cn = el.GetAttribute("class")
					id = el.GetAttribute("id")
				}
				state := "no-scroll"
				if m.OK {
					state = fmt.Sprintf("V: view=%.0f total=%.0f maxSy=%.0f thumb=%.0f", m.ViewLen, m.TotalLen, m.MaxScroll, m.ThumbLen)
				}
				cw, ch := rv.BoxContentSize(rb)
				fmt.Printf("[scroll] d=%d cls=%q id=%q pos=(%.0f,%.0f) size=%.0fx%.0f overflow=(x:%d y:%d) %s content=%.0fx%.0f\n",
					depth, cn, id, g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(),
					st.OverflowX, st.OverflowY, state, cw, ch)
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, depth+1)
		}
	}
	walk(rendering.RenderObject(rv), 0)
}
