// Command tmp_char_w_probe: 测量 wb-ui 引擎对 fold marker "›"、数字、空格的
// 渲染宽度（JetBrains Mono 13px），对比浏览器（Edge 参照）验证行号栏/
// foldGutter 宽度差异来源。
// 运行: set CGO_ENABLED=1 && go run ./dev/desktop_probe/tmp_char_w_probe.go
package main

import (
	"fmt"
	"os"

	"wb-ui/layout"
	"wb-ui/platform/graphics"
)

func main() {
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style2, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}, text)
	}
	// 与 index.html --font-editor 相同的字体栈
	families := []string{"JetBrains Mono", "Consolas", "monospace", "Courier New"}
	tests := []string{"›", "1234567890", " ", "1", "package main", "▶"}
	for _, fam := range families {
		for _, t := range tests {
			w := graphics.MeasureText(graphics.Font{Family: fam, Size: 13, Weight: 400}, t)
			fmt.Printf("[w] fam=%-16s text=[%s] w=%.2f\n", fam, t, w)
		}
	}
	fmt.Println("DONE")
	_ = os.Args
}
