// 验证 Courier New 通过 skia.NewTypeface 加载后是否渲染为斜体
package main

import (
	"fmt"
	"os"

	"github.com/hoonfeng/goskia/skia"
)

func render(tf *skia.Typeface, text string, size float32, path string) {
	w, h := 640, 80
	surf, err := skia.NewRasterSurfaceN32Premul(w, h)
	if err != nil {
		fmt.Println("surface err:", err)
		return
	}
	canvas := surf.Canvas()
	canvas.Clear(skia.RGB(255, 255, 255))
	paint := skia.NewPaint()
	paint.SetColor(skia.RGB(0, 0, 0))
	font := skia.NewFont(tf, size)
	canvas.DrawText(text, 20, 55, font, paint)
	data, err := surf.EncodePNG()
	if err != nil {
		fmt.Println("encode err:", err)
		return
	}
	os.WriteFile(path, data, 0o644)
	fmt.Println("wrote", path)
}

func main() {
	// 1. OS 名查找 Courier New（当前 fontmgr monoTF 路径）
	tf := skia.NewTypeface("Courier New", skia.FontStyle{Weight: 400, Width: 5, Slant: 0})
	if tf == nil {
		println("Courier New nil")
		os.Exit(1)
	}
	render(tf, "a b c d e f ( 3 x y z (3x) 测试中文", 13, "couri_os_13.png")

	// 2. data 加载 cour.ttf（常规体）
	if data, err := os.ReadFile(`C:\Windows\Fonts\cour.ttf`); err == nil {
		if tf2 := skia.NewTypefaceFromData(data, 0); tf2 != nil {
			render(tf2, "a b c d e f ( 3 x y z (3x) 测试中文", 13, "couri_data_13.png")
		}
	}
	// 3. data 加载 couri.ttf（斜体）作为斜体参照
	if data2, err2 := os.ReadFile(`C:\Windows\Fonts\couri.ttf`); err2 == nil {
		if tf3 := skia.NewTypefaceFromData(data2, 0); tf3 != nil {
			render(tf3, "a b c d e f ( 3 x y z (3x) 测试中文", 13, "couri_italic_13.png")
		}
	}
	// 4. OS 名查找 Consolas 对照
	if tf4 := skia.NewTypeface("Consolas", skia.FontStyle{Weight: 400, Width: 5, Slant: 0}); tf4 != nil {
		render(tf4, "a b c d e f ( 3 x y z (3x) 测试中文", 13, "consola_os_13.png")
	}
	println("done")
}
