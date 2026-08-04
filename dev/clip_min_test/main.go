// Command clip_min_test verifies Canvas.ClipRoundRect actually rounds the
// fill: clip a 100x30 rect at r=6 then fill it solid — the 4 corners must
// stay background (clipped away by the arc).
package main

import (
	"fmt"

	"wb-ui/platform/graphics"
)

func main() {
	_ = graphics.InitFontManager("")
	w, h := 260, 80
	c := graphics.NewCanvas(w*2, h*2) // 与 Render() 相同的 SSAA 2x
	c.Scale(2, 2)
	// 背景
	c.FillRect(0, 0, float64(w), float64(h), graphics.Color{R: 22, G: 27, B: 34, A: 255})
	// 圆角 clip + 填充
	c.Save()
	c.ClipRoundRect(100, 20, 100, 30, 6)
	c.FillRect(100, 20, 100, 30, graphics.Color{R: 88, G: 166, B: 255, A: 255})
	c.Restore()

	p := c.Pixels()
	px := func(x, y int) string {
		// 逻辑坐标 (x,y) → 2x 缓冲 (2x, 2y)，缓冲宽 = w*2
		off := (y*2*(w*2) + x*2) * 4
		return fmt.Sprintf("(%d,%d,%d)", p[off], p[off+1], p[off+2])
	}
	// 左上角 (100,20) 应被圆角裁掉 → 背景色 (22,27,34)
	fmt.Println("TL (100,20)  :", px(100, 20), " expect bg (22,27,34)")
	// 圆角内侧 (104,20) 应部分蓝（弧线上）
	fmt.Println("TL in (104,20):", px(104, 20))
	// 主体 (130,30) 应为纯蓝
	fmt.Println("mid (130,30) :", px(130, 30), " expect (88,166,255)")
	// 顶边中段 (150,20) 应为纯蓝（在弧线下方）
	fmt.Println("top (150,20) :", px(150, 20), " expect (88,166,255)")
	// 右上角 (200,20) 应背景
	fmt.Println("TR (200,20)  :", px(200, 20), " expect bg")
	row := ""
	for x := 100; x <= 110; x++ {
		row += fmt.Sprintf("%s ", px(x, 20))
	}
	fmt.Println("TL row y=20 :", row)
}
