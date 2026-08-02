// Command ide_tree_dump loads the REAL web IDE frontend (cmd/companion/web-ui/dist)
// through wb-ui and dumps the COMPLETE element tree — every render object with
// its DOM identity (tag/class/id), border-box geometry, and key computed styles —
// so we can compare element-by-element against the browser reference instead of
// guessing from pixels/OCR.
//
// Output: dev/desktop_probe/ide_tree_wb.json  (machine-readable)
//         stdout                            (human-readable tree)
//
// Run: go run ./dev/desktop_probe/ide_tree_dump.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
	"wb-ui/webkit"
)

// ElNode mirrors one element in the tree. Children keep the hierarchy so both
// wb-ui and Edge trees can be aligned by path (root→child index chain).
type ElNode struct {
	Tag     string    `json:"tag"`
	ID      string    `json:"id"`
	Class   string    `json:"class"`
	X       float64   `json:"x"`
	Y       float64   `json:"y"`
	W       float64   `json:"w"`
	H       float64   `json:"h"`
	Display string    `json:"display"`
	Color   string    `json:"color"`
	BG      string    `json:"bg"`
	FontSz  string    `json:"fontSz"`
	Text    string    `json:"text,omitempty"`
	Checked string    `json:"checked,omitempty"`
	Value   string    `json:"value,omitempty"`
	Depth   int       `json:"depth"`
	Idx     int       `json:"idx"` // index among siblings (tree path alignment)
	Children []ElNode `json:"children,omitempty"`
}

func treeSetupLoaders(wv *webkit.WebView, distDir string) {
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

func hexColor(c style.Color) string {
	if c.A == 0 {
		return ""
	}
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func displayName(d style.DisplayType) string {
	switch d {
	case style.DisplayInline:
		return "inline"
	case style.DisplayBlock:
		return "block"
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
	case style.DisplayTable:
		return "table"
	case style.DisplayListItem:
		return "list-item"
	case style.DisplayInlineTable:
		return "inline-table"
	}
	return fmt.Sprintf("disp%d", d)
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
	treeSetupLoaders(wv, distDir)
	_ = wv.JSInterpreter()
	// 占位 fetch：/api/* 返回空 JSON，组件渲染空态（与 Edge 无后端时一致）。
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		rt.RunJS(`(function(){
			if (!window.__origFetch) {
				window.__origFetch = window.fetch;
				window.fetch = function(url, opts) {
					var u = String(url);
					if (u.indexOf('/api/') === 0) {
						return Promise.resolve(new Response('{"ok":true,"data":[]}', {
							status: 200, headers: {'Content-Type':'application/json'}
						}));
					}
					return window.__origFetch.apply(window, arguments);
				};
			}
		})()`)
	}
	wv.LoadHTML(string(htmlData))
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("nil render view")
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
			n.Display = displayName(st.Display)
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
				if el.HasAttribute("checked") {
					n.Checked = el.GetAttribute("checked")
				}
				if el.LocalName() == "input" || el.LocalName() == "textarea" {
					if v := el.GetAttribute("value"); v != "" {
						n.Value = v
					}
				}
			} else {
				n.Tag = nn.NodeName()
				if rt, ok := o.(*rendering.RenderText); ok {
					n.Text = rt.Text()
					if len(n.Text) > 20 {
						n.Text = n.Text[:20] + "…"
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

	// JSON 输出
	jsonPath := filepath.Join(wd, "dev", "desktop_probe", "ide_tree_wb.json")
	raw, _ := json.MarshalIndent(root, "", " ")
	if err := os.WriteFile(jsonPath, raw, 0644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("=== wb-ui 完整元素树 dump → %s ===\n", jsonPath)

	// 人类可读树（缩进 + 关键信息）
	var dump func(n *ElNode)
	dump = func(n *ElNode) {
		ind := strings.Repeat("  ", n.Depth)
		label := n.Tag
		if n.ID != "" {
			label += "#" + n.ID
		}
		if n.Class != "" {
			label += "." + strings.ReplaceAll(n.Class, " ", ".")
		}
		info := fmt.Sprintf("%s%s x=%.1f y=%.1f w=%.1f h=%.1f [%s]", ind, label, n.X, n.Y, n.W, n.H, n.Display)
		if n.BG != "" {
			info += " bg=" + n.BG
		}
		if n.Color != "" {
			info += " col=" + n.Color
		}
		if n.FontSz != "" {
			info += " fs=" + n.FontSz
		}
		if n.Text != "" {
			info += " \"" + n.Text + "\""
		}
		if n.Checked != "" {
			info += " checked=" + n.Checked
		}
		fmt.Println(info)
		for i := range n.Children {
			dump(&n.Children[i])
		}
	}
	dump(root)
}
