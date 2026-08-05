// Command glyph_metrics 量化主字体(微软雅黑)与符号字体(Segoe UI Symbol)
// 的垂直 metrics 差异，用于修复折叠三角 ▸ 垂直不居中：
// PaintText 用主字体 ascent 计算基线，而 ▸ 实际用 Symbol 字体绘制，
// 两字体 glyph 视觉中心相对基线的位置不同 → 三角视觉偏移。
package main

import (
	"fmt"
	"os"

	"wb-ui/platform/graphics"

	"github.com/hoonfeng/goskia/skia"
)

func main() {
	_ = graphics.InitFontManager("")
	mgr := graphics.GetFontManager()
	if mgr != nil {
		mgr.LoadSystemFonts()
	}

	// 主字体：微软雅黑（primary，继承 font-family）
	primary := skia.NewTypeface("Microsoft YaHei", skia.FontStyle{Weight: 400, Width: 5, Slant: 0})
	// 符号字体：Segoe UI Symbol
	symbol := mgr.SymbolTypeface()

	fp := skia.NewFont(primary, 16)
	fs := skia.NewFont(symbol, 16)

	mp, _ := fp.Metrics()
	ms, _ := fs.Metrics()
	fmt.Printf("YaHei   : top=%+.2f ascent=%+.2f descent=%+.2f bottom=%+.2f lead=%+.2f\n",
		mp.Top, mp.Ascent, mp.Descent, mp.Bottom, mp.Leading)
	fmt.Printf("Symbol  : top=%+.2f ascent=%+.2f descent=%+.2f bottom=%+.2f lead=%+.2f\n",
		ms.Top, ms.Ascent, ms.Descent, ms.Bottom, ms.Leading)

	// ▸ 在 Symbol 字体中的 bounds（相对基线；top/left 负值=左/上）
	w, b := fs.MeasureText("▸", nil)
	fmt.Printf("Symbol ▸: width=%.2f left=%+.2f top=%+.2f right=%+.2f bottom=%+.2f\n",
		w, b.Left, b.Top, b.Right, b.Bottom)
	// ▸ 视觉中心相对基线
	cx := (b.Left + b.Right) / 2
	cy := (b.Top + b.Bottom) / 2
	fmt.Printf("  ▸ visual center: x=%+.2f y=%+.2f (y>0=基线下方)\n", cx, cy)

	// ▾ 在 Symbol 字体中的 bounds
	w4, b4 := fs.MeasureText("▾", nil)
	fmt.Printf("Symbol ▾: width=%.2f left=%+.2f top=%+.2f right=%+.2f bottom=%+.2f\n",
		w4, b4.Left, b4.Top, b4.Right, b4.Bottom)
	cx4 := (b4.Left + b4.Right) / 2
	cy4 := (b4.Top + b4.Bottom) / 2
	fmt.Printf("  ▾ visual center: x=%+.2f y=%+.2f (y>0=基线下方)\n", cx4, cy4)

	// YaHei 中 ▸ 的 bounds（若有 glyph）
	w2, b2 := fp.MeasureText("▸", nil)
	fmt.Printf("YaHei ▸ : width=%.2f left=%+.2f top=%+.2f right=%+.2f bottom=%+.2f (glyph=%d)\n",
		w2, b2.Left, b2.Top, b2.Right, b2.Bottom, fp.UnicharToGlyph('▸'))

	// 汉字在中文字体的 bounds（作为参照）
	_, b3 := fp.MeasureText("中", nil)
	fmt.Printf("YaHei 中: left=%+.2f top=%+.2f right=%+.2f bottom=%+.2f\n",
		b3.Left, b3.Top, b3.Right, b3.Bottom)

	// lineHeight = ascent-descent+leading（或 fontLineGap 的近似）
	fmt.Printf("\n[结论参考]\n")
	fmt.Printf("YaHei  lineHeight≈%.2f ascent=%.2f (baseline 在 line 内 %.0f%% 处)\n",
		mp.Ascent-mp.Descent+mp.Leading, mp.Ascent, 100*(-mp.Ascent)/(mp.Ascent-mp.Descent))
	fmt.Printf("Symbol lineHeight≈%.2f ascent=%.2f (baseline 在 line 内 %.0f%% 处)\n",
		ms.Ascent-ms.Descent+ms.Leading, ms.Ascent, 100*(-ms.Ascent)/(ms.Ascent-ms.Descent))
	os.Exit(0)
}
