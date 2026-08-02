// Command ide_geo renders the REAL web IDE frontend (cmd/companion/web-ui/dist)
// and dumps the geometry of the key layout containers (activity-bar, sidebar,
// main area, right panel, status bar, editor…) so we can compare wb-ui layout
// against the browser (Edge) reference pixel-for-pixel.
//
// Run: go run ./dev/desktop_probe/ide_geo.go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
	"wb-ui/webkit"
)

func geoSetupLoaders(wv *webkit.WebView, distDir string) {
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
	htmlData, err := os.ReadFile(distDir + "/index.html")
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}
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
	geoSetupLoaders(wv, distDir)
	_ = wv.JSInterpreter()
	wv.LoadHTML(string(htmlData))
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("nil render view")
	}
	state := rv.LayoutState()

	type geo struct {
		sel  string
		x, y float64
		w, h float64
	}
	var geos []geo
	seen := map[string]bool{}

	var walk func(o rendering.RenderObject, depth int)
	walk = func(o rendering.RenderObject, depth int) {
		if o == nil || depth > 40 {
			return
		}
		if n := o.Node(); n != nil {
			if el, ok := n.(*dom.Element); ok {
				cls := el.GetAttribute("class")
				var key string
				switch {
				case strings.Contains(cls, "activity-bar"):
					key = "activity-bar"
				case strings.Contains(cls, "sidebar"):
					key = "sidebar"
				case strings.Contains(cls, "main-area"):
					key = "main-area"
				case strings.Contains(cls, "editor-area"):
					key = "editor-area"
				case strings.Contains(cls, "right-panel"):
					key = "right-panel"
				case strings.Contains(cls, "status-bar"), strings.Contains(cls, "statusbar"):
					key = "status-bar"
				case strings.Contains(cls, "tab-bar"):
					key = "tab-bar"
				case strings.Contains(cls, "titlebar"), strings.Contains(cls, "title-bar"):
					key = "titlebar"
				}
				if key != "" && !seen[key] {
					if lb := o.LayoutBox(); lb != nil && state != nil {
						g := state.GeometryForBox(lb)
						geos = append(geos, geo{sel: key, x: g.Left(), y: g.Top(), w: g.BorderBoxWidth(), h: g.BorderBoxHeight()})
						seen[key] = true
					}
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, depth+1)
		}
	}
	walk(rendering.RenderObject(rv), 0)

	sort.Slice(geos, func(i, j int) bool { return geos[i].y < geos[j].y })
	fmt.Println("=== 关键容器几何（wb-ui）===")
	for _, g := range geos {
		fmt.Printf("  %-12s x=%-6.1f y=%-6.1f w=%-6.1f h=%.1f\n", g.sel, g.x, g.y, g.w, g.h)
	}

	// 统计全树元素数 + display:flex 容器数（诊断布局是否错乱）
	elCount := 0
	flexCount := 0
	var walk3 func(o rendering.RenderObject)
	walk3 = func(o rendering.RenderObject) {
		if o == nil {
			return
		}
		if st := o.Style(); st != nil {
			if st.Display == style.DisplayFlex || st.Display == style.DisplayInlineFlex {
				flexCount++
			}
		}
		if n := o.Node(); n != nil {
			elCount++
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk3(c)
		}
	}
	walk3(rendering.RenderObject(rv))
	fmt.Printf("=== 渲染树统计: 元素=%d flex容器=%d ===\n", elCount, flexCount)
}
