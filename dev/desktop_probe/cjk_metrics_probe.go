// Command cjk_metrics_probe 测量中文字体（微软雅黑/宋体）的垂直度量，
// 确定 CJK 字形顶距 baseline 的距离——painter 用它定位 baseline 使
// CJK 字形顶贴行 box 顶（浏览器行为）。
//go:build ignore

package main

import (
	"fmt"

	"wb-ui/platform/graphics"

	"github.com/hoonfeng/goskia/skia"
)

func main() {
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	mgr := graphics.GetFontManager()
	if mgr == nil {
		fmt.Println("NO FONT MANAGER")
		return
	}
	// 字体名：直接查 typeface
	for _, fam := range []string{"Microsoft YaHei", "SimSun", "SimHei", "Consolas", "Segoe UI"} {
		tf := mgr.LookupTypeface(fam, 400, "normal")
		if tf == nil {
			fmt.Printf("%s: no typeface\n", fam)
			continue
		}
		p := skia.NewPaint()
		defer p.Release()
		for _, size := range []float64{13, 16} {
			skF := skia.NewFont(tf, float32(size))
			m, ls := skF.Metrics()
			fmt.Printf("%s %gpx: top=%.3f ascent=%.3f descent=%.3f bottom=%.3f leading=%.3f capH=%.3f xH=%.3f lineSpacing=%.3f\n",
				fam, size, m.Top, m.Ascent, m.Descent, m.Bottom, m.Leading, m.CapHeight, m.XHeight, ls)
			// CJK 字形 bounds：skia.Font.MeasureText 返回 rect（Left/Top 偏移）
			_, r := skF.MeasureText("中", p)
			fmt.Printf("  CJK'中' bounds: l=%.3f t=%.3f r=%.3f b=%.3f (top<0=字形顶在baseline上方)\n",
				r.Left, r.Top, r.Right, r.Bottom)
			_, r2 := skF.MeasureText("A", p)
			fmt.Printf("  'A' bounds: l=%.3f t=%.3f r=%.3f b=%.3f\n",
				r2.Left, r2.Top, r2.Right, r2.Bottom)
			skF.Release()
		}
	}
	// CJK typeface 同测
	if ct := mgr.CJKTypeface(); ct != nil {
		p := skia.NewPaint()
		defer p.Release()
		skF := skia.NewFont(ct, 13)
		m, _ := skF.Metrics()
		_, r := skF.MeasureText("中文", p)
		fmt.Printf("CJKTypeface 13px: top=%.3f ascent=%.3f capH=%.3f | '中文' bounds t=%.3f b=%.3f (top<0字形顶在baseline上方)\n",
			m.Top, m.Ascent, m.CapHeight, r.Top, r.Bottom)
		skF.Release()
	}
	fmt.Println("done")
}
