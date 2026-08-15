// Command font_metrics_probe 打印 wb-ui 引擎对 Consolas 13px 的
// ascent/descent/lineGap，对照浏览器（CSS hhea）的期望值。
//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"wb-ui/layout"
	"wb-ui/platform/graphics"

	"github.com/hoonfeng/goskia/skia"
)

func main() {
	wd, _ := os.Getwd()
	_ = wd
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style2, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}, text)
	}
	layout.FontMetricsFunc = func(family string, size float64, weight int, style string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}
	for _, fam := range []string{"Consolas", "Cascadia Code", "JetBrains Mono"} {
		for _, size := range []float64{13, 16} {
			f := graphics.Font{Family: fam, Size: size, Weight: 400, Style: "normal"}
			a, d, g := graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
			// 通过真实 typeface 取 CapHeight
			mgr := graphics.GetFontManager()
			tf := mgr.LookupTypeface(fam, 400, "normal")
			var capH float64
			if tf != nil {
				skF := skia.NewFont(tf, float32(size))
				m, _ := skF.Metrics()
				capH = float64(m.CapHeight)
			}
			fmt.Printf("%s %gpx: ascent=%.3f descent=%.3f lineGap=%.3f A+D=%.3f capHeight=%.3f\n",
				fam, size, a, d, g, a+d, capH)
		}
	}
	// 用 MeasureText 测 "LINE1" 的渲染高度（Skia 实际绘制）
	f := graphics.Font{Family: "Consolas", Size: 13, Weight: 400, Style: "normal"}
	w := graphics.MeasureText(f, "LINE1")
	fmt.Printf("MeasureText(Consolas 13, LINE1) = %.3f\n", w)
	// 检查 .pair 目录下是否有字体测量工具
	entries, _ := os.ReadDir(filepath.Join("dev", "desktop_probe"))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
	}
	_ = entries
	fmt.Println("done")
}
