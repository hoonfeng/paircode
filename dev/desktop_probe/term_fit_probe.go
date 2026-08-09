// Command term_fit_probe loads the terminal-fit reference page
// (cmd/companion/web-ui/ide_ref_termfit.html) through wb-ui and checks
// the FIT behavior: cols/rows computed by FitAddon.fit() right after
// open(), cell size, and rendered line widths — compared against Edge
// (edge_termfit_ref.go → termfit_tree_edge.json).
//
// Key question: does wb-ui compute the same cols as the browser for a
// fixed 800px container? If open() happens before font metrics are ready,
// the first measurement may return cell 8x16 instead of 7.15x15 → fit
// computes too few cols → terminal output width is wrong.
//
// Output: dev/desktop_probe/termfit_tree_wb.json + [FITDIAG] diagnostics
// Run: go run ./dev/desktop_probe/term_fit_probe.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
	"wb-ui/webkit"
)

type TNode struct {
	Tag      string  `json:"tag"`
	ID       string  `json:"id"`
	Class    string  `json:"class"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	W        float64 `json:"w"`
	H        float64 `json:"h"`
	Display  string  `json:"display"`
	Color    string  `json:"color"`
	BG       string  `json:"bg"`
	FontSz   string  `json:"fontSz"`
	Lh       string  `json:"lh,omitempty"`
	Text     string  `json:"text,omitempty"`
	Depth    int     `json:"depth"`
	Idx      int     `json:"idx"`
	Children []TNode `json:"children,omitempty"`
}

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	webDir := filepath.Join(wd, "cmd", "companion", "web-ui")
	htmlData, err := os.ReadFile(filepath.Join(webDir, "ide_ref_termfit.html"))
	if err != nil {
		log.Fatalf("read ide_ref_termfit.html: %v", err)
	}
	log.Printf("[term_fit_probe] webDir=%s html=%d bytes", webDir, len(htmlData))

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
	setupTermLoaders(wv, webDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()

	wv.LoadHTML(string(htmlData))
	_, _ = wv.JSInterpreter().RunJS(`
		if (typeof window.onload === 'function') { window.onload(); }
	`)
	interp := wv.JSInterpreter()
	if el := interp.GetEventLoop(); el != nil {
		for i := 0; i < 12; i++ {
			el.ProcessTasks(0)
			_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 200); })`)
			el.ProcessTasks(0)
		}
	}
	if out := wv.ConsoleOutput(); out != "" {
		fmt.Println("[CONSOLE]")
		fmt.Println(out)
	}
	diagJS, _ := interp.RunJS(`(function(){
		var t = window.__term;
		if (!t) return JSON.stringify({err:'no __term'});
		var core = t._core || t;
		var d = (core._renderService && core._renderService.dimensions) || null;
		var host = document.getElementById('term-host');
		var rows = host ? host.querySelector('.xterm-rows') : null;
		var hr = host ? host.getBoundingClientRect() : null;
		var sr = rows ? rows.getBoundingClientRect() : null;
		var widths = [];
		if (rows) {
			for (var i = 0; i < rows.children.length; i++) {
				var r = rows.children[i].getBoundingClientRect();
				widths.push(Math.round(r.width*100)/100);
			}
		}
		var cs = core._charSizeService;
		return JSON.stringify({
			cols: t.cols, rows: t.rows,
			cssCellW: d && d.css ? d.css.cell.width : null,
			cssCellH: d && d.css ? d.css.cell.height : null,
			charW: cs ? cs.width : null,
			charH: cs ? cs.height : null,
			hostW: hr ? Math.round(hr.width*10)/10 : null,
			hostH: hr ? Math.round(hr.height*10)/10 : null,
			screenW: sr ? Math.round(sr.width*10)/10 : null,
			lineWidths: widths
		});
	})()`)
	fmt.Println("[FITDIAG] " + diagJS.ToString())

	_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}
	state := rv.LayoutState()

	var build func(o rendering.RenderObject, depth, idx int) *TNode
	build = func(o rendering.RenderObject, depth, idx int) *TNode {
		if o == nil {
			return nil
		}
		n := &TNode{Depth: depth, Idx: idx}
		st := o.Style()
		if st != nil {
			d := displayName(st.Display)
			if lb := o.LayoutBox(); lb != nil && lb.Parent() != nil {
				if pcs := lb.Parent().Style(); pcs != nil {
					pd := pcs.Display
					if (pd == style.DisplayFlex || pd == style.DisplayInlineFlex) && !lb.IsAbsolutelyPositioned() {
						switch st.Display {
						case style.DisplayInline:
							d = "block"
						case style.DisplayInlineBlock:
							d = "block"
						case style.DisplayInlineFlex:
							d = "flex"
						}
					}
				}
			}
			n.Display = d
			n.Color = hexColor(st.Color)
			n.BG = hexColor(st.BackgroundColor)
			if st.FontSize.Value > 0 {
				n.FontSz = fmt.Sprintf("%.1f", st.FontSize.Value)
			}
			if st.LineHeight.Value > 0 {
				n.Lh = fmt.Sprintf("%.1f", st.LineHeight.Value)
			}
		}
		if nn := o.Node(); nn != nil {
			if el, ok := nn.(*dom.Element); ok {
				n.Tag = el.LocalName()
				n.ID = el.GetAttribute("id")
				n.Class = el.GetAttribute("class")
			} else {
				n.Tag = nn.NodeName()
				if rt, ok := o.(*rendering.RenderText); ok {
					t := strings.TrimSpace(rt.Text())
					if t != "" {
						if len(t) > 40 {
							t = t[:40] + "…"
						}
						n.Text = t
					}
				}
			}
		}
		if lb := o.LayoutBox(); lb != nil && state != nil {
			g := state.GeometryForBox(lb)
			n.X, n.Y = g.Left(), g.Top()
			n.W, n.H = g.BorderBoxWidth(), g.BorderBoxHeight()
		}
		ci := 0
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			if ch := build(c, depth+1, ci); ch != nil {
				n.Children = append(n.Children, *ch)
			}
			ci++
		}
		return n
	}

	root := build(rendering.RenderObject(rv), 0, 0)
	if root == nil {
		log.Fatal("empty tree")
	}
	jsonOut, _ := json.MarshalIndent(root, "", " ")
	_ = os.WriteFile(filepath.Join(wd, "dev", "desktop_probe", "termfit_tree_wb.json"), jsonOut, 0o644)
	fmt.Printf("[FIT] tree saved=termfit_tree_wb.json\n")
}

func setupTermLoaders(wv *webkit.WebView, webDir string) {
	absDir, _ := filepath.Abs(webDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				t0 := time.Now()
				data, err := os.ReadFile(filepath.Join(absDir, clean))
				if err != nil {
					return "", err
				}
				code := string(data)
				if i := strings.Index(code, "//# sourceMappingURL="); i >= 0 {
					code = code[:i]
				}
				loadDur := time.Since(t0)
				if strings.Contains(clean, "xterm") || strings.Contains(clean, "app.") || strings.Contains(clean, "chunk") {
					log.Printf("[loader] %s read=%d bytes load+strip=%v", clean, len(data), loadDur)
				}
				return code, nil
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDir, clean))
				return string(data), err
			}
		}
	}
}

func displayName(d style.DisplayType) string {
	switch d {
	case style.DisplayBlock:
		return "block"
	case style.DisplayInline:
		return "inline"
	case style.DisplayInlineBlock:
		return "inline-block"
	case style.DisplayFlex:
		return "flex"
	case style.DisplayInlineFlex:
		return "inline-flex"
	case style.DisplayGrid:
		return "grid"
	case style.DisplayNone:
		return "none"
	}
	return fmt.Sprintf("d%d", d)
}

func hexColor(c style.Color) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}
