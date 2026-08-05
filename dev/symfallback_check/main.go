// Command symfallback_check 验证 wb-ui 在「未预加载字体文件」路径下
// （webkit.ensureFonts → InitFontManager("") 不调 LoadSystemFonts，m.fonts
// 为空）仍能通过 OS 名字查找解析出 symbol/emoji 字体——折叠三角 ▸ 因此
// 可渲染而非 .notdef 方块。
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
	if mgr == nil {
		fmt.Println("FAIL: mgr nil")
		os.Exit(1)
	}
	fmt.Printf("sans=%v symbol=%v emoji=%v\n",
		mgr.DefaultTypeface() != nil, mgr.SymbolTypeface() != nil, mgr.EmojiTypeface() != nil)
	if mgr.SymbolTypeface() == nil {
		fmt.Println("FAIL: symbolTF still nil (OS-name symbol fallback missing)")
		os.Exit(1)
	}
	f := skia.NewFont(mgr.SymbolTypeface(), 16)
	if f == nil {
		fmt.Println("FAIL: cannot make skia font")
		os.Exit(1)
	}
	g := f.UnicharToGlyph('▸')
	fmt.Printf("symbol glyph(▸)=%d\n", g)
	if g == 0 {
		fmt.Println("FAIL: ▸ has no glyph in symbol font")
		os.Exit(1)
	}
	fmt.Println("PASS")
}
