package main

// Command chat_tree_probe: dump the render subtree of the first msg-item in
// .chat-messages, with absolute geometry + layer status, to find why a long
// message loses its y≈367..575 slice when scrolled by 300.
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

func clsOf(rb *rendering.RenderBox) string {
	if el, ok := rb.Node().(*dom.Element); ok {
		return el.GetAttribute("class")
	}
	return ""
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, err := os.ReadFile(filepath.Join(distDir, "index.html"))
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
	for i := 0; i < 20; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
	}
	for i := 0; i < 8; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 1000); })`)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}

	// Find chat-messages and msg-list-wrap
	var list *rendering.RenderBox
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil || list != nil {
			return
		}
		if rb := asRB(o); rb != nil && rb.Style() != nil && clsOf(rb) == "msg-list-wrap" {
			list = rb
			return
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	if list == nil {
		log.Fatal("no msg-list-wrap")
	}
	log.Printf("msg-list-wrap frame=%.0f,%.0f %.0fx%.0f", list.X(), list.Y(), list.Width(), list.Height())

	// Dump first msg-item subtree
	n := 0
	var dump func(o rendering.RenderObject, depth int)
	dump = func(o rendering.RenderObject, depth int) {
		if o == nil || n > 400 {
			return
		}
		var rb *rendering.RenderBox
		var tag, cls string
		if el, ok := o.Node().(*dom.Element); ok {
			tag = el.TagName()
			cls = el.GetAttribute("class")
		}
		if b := asRB(o); b != nil {
			rb = b
		}
		reqL := false
		if rb != nil {
			reqL = rendering.RequiresLayer(o)
		}
		desc := ""
		if rt, ok := o.(*rendering.RenderText); ok {
			txt := strings.TrimSpace(rt.Text())
			if len(txt) > 24 {
				txt = txt[:24] + "…"
			}
			desc = fmt.Sprintf(" text=%q", txt)
		}
		if rb != nil {
			fmt.Printf("%s%s %q y=%.0f h=%.0f x=%.0f w=%.0f layer=%v%s\n",
				strings.Repeat("  ", depth), tag, cls, rb.Y(), rb.Height(), rb.X(), rb.Width(), reqL, desc)
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			dump(c, depth+1)
		}
	}
	for c := rendering.RenderObject(list).FirstChild(); c != nil; c = c.NextSibling() {
		n++
		dump(c, 0)
	}
	log.Printf("done")
}
