// Command explorer_probe: 加载 desktop IDE（web-ui/dist）并输出文件浏览器
// 区域内所有 RenderText 的 segments 几何与内容，用于诊断「那一栏文字显示异常」。
//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
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
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, err := os.ReadFile(distDir + "/index.html")
	if err != nil {
		fmt.Println("read index.html:", err)
		return
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
	setupLoaders(wv, distDir)
	wv.Resize(1280, 800)
	if err := wv.LoadHTML(string(htmlData)); err != nil {
		fmt.Println("LoadHTML err:", err)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	if out := wv.ConsoleOutput(); out != "" {
		fmt.Println("[CONSOLE]", out[:min(len(out), 2000)])
	}

	rv := wv.RenderView()
	if rv == nil {
		fmt.Println("no render view")
		return
	}
	state := rv.LayoutState()

	// 遍历渲染树，只关心文件浏览器侧边栏区域（x<340 或 class 含 explorer/file）
	var walk func(ro rendering.RenderObject, depth int)
	walk = func(ro rendering.RenderObject, depth int) {
		if ro == nil {
			return
		}
		name := ro.RenderName()
		// 输出关键容器元素（sidebar/file-explorer/header/toolbar/divider 等）的几何
		if n := ro.Node(); n != nil {
			if el, ok := n.(*dom.Element); ok {
				cls := el.GetAttribute("class")
				hit := false
				for _, kw := range []string{"sidebar", "explorer", "toolbar", "divider", "file-explorer", "project", "ws-"} {
					if strings.Contains(cls, kw) {
						hit = true
						break
					}
				}
				if hit && state != nil {
					lb := ro.LayoutBox()
					if lb != nil {
						g := state.GeometryForBox(lb)
						cs := ro.Style()
						ht := "?"
						bs := "?"
						if cs != nil {
							ht = fmt.Sprintf("h=%s", cs.Height)
							bs = fmt.Sprintf("boxSizing=%v", cs.BoxSizing)
						}
						fmt.Printf("  [BOX] class=%q w=%.0f h=%.0f x=%.0f y=%.0f %s %s\n", cls, g.BorderBoxWidth(), g.BorderBoxHeight(), g.Left(), g.Top(), ht, bs)
					}
				}
			}
		}
		if name == "RenderText" {
			if rt, ok := ro.(*rendering.RenderText); ok {
				txt := rt.Text()
				txt = strings.TrimSpace(txt)
				if txt != "" && len([]rune(txt)) <= 40 {
					segs := rt.Segments()
					pos := ""
					if len(segs) > 0 {
						s := segs[0]
						pos = fmt.Sprintf(" segs=%d first=(x=%.1f y=%.1f w=%.1f h=%.1f) start=%d len=%d",
							len(segs), s.X, s.Y, s.Width, s.Height, s.Start, s.Len)
						// 附加：全部 segment 的 x 序列
						var xs []string
						for _, seg := range segs {
							xs = append(xs, fmt.Sprintf("(%d:%d x%.1f w%.1f)", seg.Start, seg.Len, seg.X, seg.Width))
						}
						pos += " all=[" + strings.Join(xs, " ") + "]"
					} else {
						pos = " segs=0"
					}
					// 只输出文件浏览器区域内（x < 340）的短文本
					if len(segs) > 0 && segs[0].X < 340 {
						fmt.Printf("  [TEXT] %q%s\n", txt, pos)
					}
				}
			}
		}
		for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, depth+1)
		}
	}
	_ = state
	// 从 RenderView 的第一层开始遍历
	fmt.Println("=== 文件浏览器区域 (x<340) 的 RenderText segments ===")
	for c := rv.FirstChild(); c != nil; c = c.NextSibling() {
		walk(c, 0)
	}
	fmt.Println("=== done ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
