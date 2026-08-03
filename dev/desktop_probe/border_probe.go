// 探针：检查 active conv-item 的完整四边像素，确认只有左边框（蓝色竖线）。
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

	canvas := graphics.NewCanvas(1280, 800)
	defer canvas.Release()
	rendering.Paint(rv, canvas, rendering.Rect{X: 0, Y: 0, Width: 1280, Height: 800})

	// ★ 保存 conv-item 区域为原始 RGBA 字节（供 Python 对比 Edge）
	// 区域：x=x-2 .. x+12, y=y-4 .. y+h+4（含竖线与背景）
	sx0, sy0 := int(x)-2, int(y)-4
	sx1, sy1 := int(x)+14, int(y)+int(h)+6
	pix := canvas.Pixels()
	// Pixels() 返回 1280*800*4 的 RGBA
	out := make([]byte, 0, (sx1-sx0)*(sy1-sy0)*4)
	for yy := sy0; yy < sy1; yy++ {
		for xx := sx0; xx < sx1; xx++ {
			idx := (yy*1280 + xx) * 4
			if idx+3 < len(pix) {
				out = append(out, pix[idx], pix[idx+1], pix[idx+2], pix[idx+3])
			}
		}
	}
	os.WriteFile("tmp/conv_item_wb.rgba", out, 0o644)
	fmt.Printf("[border-probe] saved tmp/conv_item_wb.rgba (%dx%d)\n", sx1-sx0, sy1-sy0)

	isBlue := func(c graphics.Color) bool { return c.B >= 150 && c.B > c.R+40 && c.A >= 150 }
	isNonBg := func(c graphics.Color) bool {
		// 非 item 背景色（--bg-active #1f2b3d / 面板背景 #21262d）的像素
		return c.A > 100 && !(c.R > 25 && c.R < 45 && c.G > 35 && c.G < 55 && c.B > 55 && c.B < 80)
	}

	// 四边扫描：
	// 左边缘 x=x, 右边缘 x+w-1, 上边缘 y, 下边缘 y+h-1
	ix, iy := int(x), int(y)
	iw, ih := int(w), int(h)
	fmt.Printf("[border-probe] scanning edges: left x=%d, right x=%d, top y=%d, bottom y=%d\n", ix, ix+iw-1, iy, iy+ih-1)

	fmt.Println("[border-probe] LEFT edge (x=", ix, "..", ix+1, "):")
	for yy := iy; yy <= iy+ih-1; yy++ {
		row := fmt.Sprintf("  y=%3d:", yy)
		for xx := ix; xx <= ix+1; xx++ {
			px := canvas.PixelAt(xx, yy)
			if isBlue(px) {
				row += " B"
			} else if isNonBg(px) {
				row += " ?"
			} else {
				row += " ."
			}
		}
		fmt.Println(row)
	}

	fmt.Println("[border-probe] RIGHT edge (x=", ix+iw-2, "..", ix+iw-1, "):")
	for yy := iy; yy <= iy+ih-1; yy++ {
		px1 := canvas.PixelAt(ix+iw-2, yy)
		px2 := canvas.PixelAt(ix+iw-1, yy)
		mark := "."
		if isNonBg(px1) || isNonBg(px2) {
			mark = "?"
		}
		if isBlue(px1) || isBlue(px2) {
			mark = "B"
		}
		fmt.Printf("  y=%3d: %s\n", yy, mark)
	}

	fmt.Println("[border-probe] TOP edge (y=", iy, "..", iy+1, "):")
	for xx := ix; xx <= ix+iw-1; xx++ {
		px1 := canvas.PixelAt(xx, iy)
		px2 := canvas.PixelAt(xx, iy+1)
		mark := "."
		if isNonBg(px1) || isNonBg(px2) {
			mark = "?"
		}
		if isBlue(px1) || isBlue(px2) {
			mark = "B"
		}
		fmt.Printf("  x=%3d: %s\n", xx, mark)
	}

	fmt.Println("[border-probe] BOTTOM edge (y=", iy+ih-2, "..", iy+ih-1, "):")
	for xx := ix; xx <= ix+iw-1; xx++ {
		px1 := canvas.PixelAt(xx, iy+ih-2)
		px2 := canvas.PixelAt(xx, iy+ih-1)
		mark := "."
		if isNonBg(px1) || isNonBg(px2) {
			mark = "?"
		}
		if isBlue(px1) || isBlue(px2) {
			mark = "B"
		}
		fmt.Printf("  x=%3d: %s\n", xx, mark)
	}
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
