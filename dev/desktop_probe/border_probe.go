// 探针：用 wb-ui 渲染真实前端，检查对话列表 active conv-item 的
// 左边框（蓝色竖线）顶端是否圆角（圆角外像素 alpha 明显低于中段）。
// 用法：go run dev/desktop_probe/border_probe.go
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
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}
	state := rv.LayoutState()

	// 找 active conv-item 的 LayoutBox（通过布局树遍历）
	var activeBox layout.Box
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil || activeBox != nil {
			return
		}
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			if strings.Contains(cn, "conv-item") && strings.Contains(cn, "active") {
				activeBox = o.LayoutBox()
				return
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	if activeBox == nil {
		fmt.Println("[border-probe] no active conv-item found")
		return
	}

	g := state.GeometryForBox(activeBox)
	x, y := g.Left(), g.Top()
	w, h := g.BorderBoxWidth(), g.BorderBoxHeight()
	fmt.Printf("[border-probe] active conv-item x=%.1f y=%.1f w=%.1f h=%.1f\n", x, y, w, h)

	// Paint 到离屏 canvas（1280x800）
	canvas := graphics.NewCanvas(1280, 800)
	defer canvas.Release()
	rendering.Paint(rv, canvas, rendering.Rect{X: 0, Y: 0, Width: 1280, Height: 800})

	// 左边框区域：x=x..x+2（2px 边框），检查顶端圆角
	bx := int(x) + 1 // 边框中心像素
	topY := int(y)
	midY := int(y) + int(h)/2
	botY := int(y) + int(h) - 1

	tp := canvas.PixelAt(bx, topY)
	mp := canvas.PixelAt(bx, midY)
	bp := canvas.PixelAt(bx, botY)
	fmt.Printf("[border-probe] left border: top(%d,%d)=%+v mid(%d,%d)=%+v bot(%d,%d)=%+v\n",
		bx, topY, tp, bx, midY, mp, bx, botY, bp)

	// 判定：中段应为饱和蓝（#58a6ff 风格：B 高、R 低）；顶端圆角外不应是
	// 饱和蓝（应是背景色/淡蓝渐变）；底部圆角同理。
	isBlue := func(c graphics.Color) bool { return c.B >= 200 && c.R < 160 && c.A >= 200 }
	if !isBlue(mp) {
		fmt.Println("[border-probe] FAIL: mid border not saturated blue")
		os.Exit(1)
	}
	if isBlue(tp) {
		fmt.Println("[border-probe] FAIL: top corner NOT rounded (still saturated blue)")
		os.Exit(1)
	}
	if isBlue(bp) {
		fmt.Println("[border-probe] FAIL: bottom corner NOT rounded (still saturated blue)")
		os.Exit(1)
	}
	fmt.Println("[border-probe] PASS: left border rounded at both ends (top/bottom not saturated), mid blue")
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
