// Command scroll_diag_probe loads the REAL companion frontend through wb-ui
// (same startup path as real_probe) and dumps the scroll metrics of
// .chat-messages plus the geometry of its flex children. Used to verify that
// a long agent message's bubble height feeds the scroll container's content
// size (regression guard for "tail of long message unreachable by scroll").
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
	state := rv.LayoutState()

	targets := map[string]bool{
		"chat-messages": true, "msg-list-wrap": true, "msg-item": true,
		"msg-bubble": true, "msg-avatar": true, "msg-time": true,
	}

	var walk func(o rendering.RenderObject, depth int)
	walk = func(o rendering.RenderObject, depth int) {
		if o == nil {
			return
		}
		var rb *rendering.RenderBox
		switch v := o.(type) {
		case *rendering.RenderBox:
			rb = v
		case *rendering.RenderBlock:
			rb = &v.RenderBox
		case *rendering.RenderBlockFlow:
			rb = &v.RenderBlock.RenderBox
		}
		if rb != nil {
			var cn string
			if el, ok := rb.Node().(*dom.Element); ok {
				cn = el.GetAttribute("class")
			}
			if targets[cn] && state != nil {
				g := state.GeometryForBox(rb.LayoutBox())
				pb := rb.PaddingBoxRect()
				desc := ""
				if cn == "chat-messages" {
					if m := rendering.VerticalScrollbarMetrics(rv, rb); m.OK {
						desc = fmt.Sprintf(" SCROLL OK view=%.0f total=%.0f maxSy=%.0f thumb=%.0f", m.ViewLen, m.TotalLen, m.MaxScroll, m.ThumbLen)
					} else {
						cw, ch := rv.BoxContentSize(rb)
						desc = fmt.Sprintf(" NO-SCROLL (content=%.1fx%.1f)", cw, ch)
					}
				}
				fmt.Printf("[diag] %s x=%.0f y=%.0f w=%.0f h=%.0f pbTop=%.0f%s\n",
					cn, g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(), pb.Y, desc)
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, depth+1)
		}
	}
	walk(rendering.RenderObject(rv), 0)

	// Content-size check for chat-messages via BoxContentSize
	var find func(o rendering.RenderObject) *rendering.RenderBox
	find = func(o rendering.RenderObject) *rendering.RenderBox {
		if o == nil {
			return nil
		}
		if el, ok := o.Node().(*dom.Element); ok && el.GetAttribute("class") == "chat-messages" {
			if rb, ok := o.(*rendering.RenderBox); ok {
				return rb
			}
			if bl, ok := o.(*rendering.RenderBlock); ok {
				return &bl.RenderBox
			}
			if bf, ok := o.(*rendering.RenderBlockFlow); ok {
				return &bf.RenderBlock.RenderBox
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			if r := find(c); r != nil {
				return r
			}
		}
		return nil
	}
	if cm := find(rendering.RenderObject(rv)); cm != nil {
		cw, ch := rv.BoxContentSize(cm)
		oc := cm.Style().OverflowY
		fmt.Printf("[diag] chat-messages BoxContentSize=%.1fx%.1f overflowY=%d\n", cw, ch, oc)
	}

	// Markup sanity: print the CSS rule that sets .chat-messages overflow
	for _, sel := range []string{".chat-messages", ".msg-list-wrap", ".msg-item", ".msg-bubble"} {
		_ = sel
	}
	fmt.Println("[diag] style overflowY enum: Auto=", style.OverflowAuto, " Visible=", style.OverflowVisible)
}
