// Command ws_probe3 loads ide_ref_ws.html and dumps text segments via
// RenderObject tree (same traversal as term_probe), to verify white-space:pre.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	webDir := filepath.Join(wd, "cmd", "companion", "web-ui")
	htmlData, err := os.ReadFile(filepath.Join(webDir, "ide_ref_ws.html"))
	if err != nil {
		log.Fatalf("read: %v", err)
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
	wv.Resize(640, 200)
	_ = wv.JSInterpreter()
	wv.LoadHTML(string(htmlData))
	_, _ = wv.JSInterpreter().RunJS(`if (typeof window.onload === 'function') window.onload();`)
	interp := wv.JSInterpreter()
	if el := interp.GetEventLoop(); el != nil {
		for i := 0; i < 4; i++ {
			el.ProcessTasks(0)
			_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 50); })`)
			el.ProcessTasks(0)
		}
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}
	dumpTree(rendering.RenderObject(rv), 0, rv.LayoutState())
}

func dumpTree(ro rendering.RenderObject, depth int, state *layout.LayoutState) {
	if ro == nil {
		return
	}
	pad := strings.Repeat("  ", depth)
	name := ro.RenderName()
	lb := ro.LayoutBox()
	geom := ""
	if lb != nil && state != nil {
		g := state.GeometryForBox(lb)
		geom = fmt.Sprintf(" x=%.1f y=%.1f w=%.1f h=%.1f", g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight())
	}
	line := pad + name + geom
	if ro.IsText() {
		if rt, ok := ro.(*rendering.RenderText); ok {
			line += fmt.Sprintf(" text=%q", rt.Text())
			for _, s := range rt.Segments() {
				line += fmt.Sprintf("\n%s  SEG start=%d len=%d x=%.1f y=%.1f w=%.1f lineY=%.1f", pad, s.Start, s.Len, s.X, s.Y, s.Width, s.LineY)
			}
		}
	}
	fmt.Println(line)
	for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
		dumpTree(c, depth+1, state)
	}
}
