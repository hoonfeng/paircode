// Command real_probe loads the REAL companion frontend through wb-ui with the
// SAME startup path as cmd/desktop (desktopbridge.Init → core.Load + real
// handlers + fetch interception to local Go), so wb-ui renders the real IDE
// with real workspace data. It then dumps the complete element tree (same
// format as ide_tree_dump) for comparison against Edge headless loading
// http://localhost:9090.
//
// Output: dev/desktop_probe/real_tree_wb.json + stdout tree
package main

import (
	"encoding/json"
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

type ElNode struct {
	Tag      string    `json:"tag"`
	ID       string    `json:"id"`
	Class    string    `json:"class"`
	X        float64   `json:"x"`
	Y        float64   `json:"y"`
	W        float64   `json:"w"`
	H        float64   `json:"h"`
	Display  string    `json:"display"`
	Color    string    `json:"color"`
	BG       string    `json:"bg"`
	FontSz   string    `json:"fontSz"`
	Text     string    `json:"text,omitempty"`
	Checked  string    `json:"checked,omitempty"`
	Value    string    `json:"value,omitempty"`
	Depth    int       `json:"depth"`
	Idx      int       `json:"idx"`
	Children []ElNode  `json:"children,omitempty"`
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
	log.Printf("[real_probe] distDir=%s html=%d bytes", distDir, len(htmlData))

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

	// ★ 与 cmd/desktop 完全相同的桥接初始化（真实 handler + fetch 拦截）。
	desktopbridge.Init(wv)

	wv.LoadHTML(string(htmlData))
	// 等待 Vue 挂载 + 真实数据异步加载（fetch → bridge → state → DOM）
	for i := 0; i < 8; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}
	state := rv.LayoutState()

	var build func(o rendering.RenderObject, depth, idx int) *ElNode
	build = func(o rendering.RenderObject, depth, idx int) *ElNode {
		if o == nil {
			return nil
		}
		n := &ElNode{Depth: depth, Idx: idx}
		st := o.Style()
		if st != nil {
			d := displayName(st.Display)
			// Flex items are blockified (CSS-DISPLAY-3 §2.7): Edge reports
			// block for span flex items whose computed display is inline.
			// Elements that are THEMSELVES flex/grid containers keep their
			// display (only inline* → block/flex).
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
			if st.FlexDirection != "" {
				n.Display += " " + st.FlexDirection
			}
			n.Color = hexColor(st.Color)
			n.BG = hexColor(st.BackgroundColor)
			if st.FontSize.Value > 0 {
				n.FontSz = fmt.Sprintf("%.1f", st.FontSize.Value)
			}
		}
		if nn := o.Node(); nn != nil {
			if el, ok := nn.(*dom.Element); ok {
				n.Tag = el.LocalName()
				n.ID = el.GetAttribute("id")
				n.Class = el.GetAttribute("class")
				if el.LocalName() == "input" {
					n.Value = el.GetAttribute("value")
				}
				if el.LocalName() == "textarea" {
					n.Value = el.TextContent()
				}
				n.Checked = el.GetAttribute("checked")
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
	os.WriteFile(filepath.Join("dev", "desktop_probe", "real_tree_wb.json"), jsonOut, 0o644)
	printTree(root, 0)
	fmt.Printf("\n[real_probe] tree nodes=%d saved=real_tree_wb.json\n", countNodes(root))
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
				// Keep Vue scoped [data-v-...] selectors: DOM carries data-v attrs.
				return string(data), nil
			}
		}
	}
}

func printTree(n *ElNode, depth int) {
	pad := strings.Repeat("  ", depth)
	cls := ""
	if n.Class != "" {
		cls = "." + strings.Fields(n.Class)[0]
	}
	id := ""
	if n.ID != "" {
		id = "#" + n.ID
	}
	text := ""
	if n.Text != "" {
		text = fmt.Sprintf(" %q", n.Text)
	}
	fmt.Printf("%s%s%s%s x=%.0f y=%.0f w=%.0f h=%.0f [%s]%s%s\n",
		pad, n.Tag, id, cls, n.X, n.Y, n.W, n.H, n.Display, text, colorInfo(n))
	for _, c := range n.Children {
		printTree(&c, depth+1)
	}
}

func colorInfo(n *ElNode) string {
	if n.BG != "" && n.BG != "#000000" {
		return " bg=" + n.BG
	}
	return ""
}

func countNodes(n *ElNode) int {
	c := 1
	for i := range n.Children {
		c += countNodes(&n.Children[i])
	}
	return c
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
