// Command outline_px_probe renders the interact_test.html textarea with
// :focus and measures the painted outline width in pixels — the user reported
// the focus border thickness looked wrong.
package main

import (
	"fmt"
	"os"

	"wb-ui/dom"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

const htmlPath = "cmd/desktop/web-ui/interact_test.html"

func main() {
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		fmt.Println("read:", err)
		os.Exit(1)
	}
	wv := webkit.NewWebView()
	if err := wv.LoadHTML(string(data)); err != nil {
		fmt.Println("LoadHTML:", err)
		os.Exit(1)
	}
	fr := wv.MainFrame().Frame()
	wv.EnsureLayout()

	doc := wv.Document()
	var ta *dom.Element
	for _, el := range doc.GetElementsByTagName("textarea") {
		ta = el
		break
	}
	if ta == nil {
		fmt.Println("FAIL: no textarea")
		os.Exit(1)
	}

	ta.SetFocused(true)
	fr.MarkRenderTreeDirty()
	fr.RebuildRenderTree()
	wv.EnsureLayout()

	// 渲染到画布
	c := graphics.NewCanvas(1280, 940)
	rv := wv.RenderView()
	rv.MarkAllDirty() // 强制全量重绘，排除 dirtyRect 裁剪
	rendering.Paint(rv, c, rendering.Rect{0, 0, 1280, 940})
	fmt.Printf("post-Paint StateDepth=%d\n", c.StateDepth())
	// 直接测 outline 位置的像素（y=306..311, x=193）
	for y := 305; y <= 312; y++ {
		px := c.PixelAt(193, y)
		fmt.Printf("  post-Paint (193,%d)=#%02x%02x%02x\n", y, px.R, px.G, px.B)
	}
	// Paint 后直接重画 outline 位置，验证 canvas 状态
	c.StrokeRect(142, 308, 342, 98, 2, graphics.Color{R: 0x2a, G: 0xc3, B: 0xde, A: 0xFF})
	p3 := c.PixelAt(193, 307)
	p4 := c.PixelAt(193, 308)
	fmt.Printf("post-Paint StrokeRect outline pos: (193,307)=#%02x%02x%02x (193,308)=#%02x%02x%02x\n", p3.R, p3.G, p3.B, p4.R, p4.G, p4.B)
	// 测试 StrokeRect + PixelAt 一致性
	c.StrokeRect(100, 100, 50, 30, 2, graphics.Color{R: 0x2a, G: 0xc3, B: 0xde, A: 0xFF})
	p1 := c.PixelAt(105, 100)
	p2 := c.PixelAt(105, 130)
	fmt.Printf("test StrokeRect: (105,100)=#%02x%02x%02x (105,130)=#%02x%02x%02x\n", p1.R, p1.G, p1.B, p2.R, p2.G, p2.B)

	// 找 textarea box 位置
	var tx, ty, tw, th float64
	var findRO func(ro rendering.RenderObject) bool
	findRO = func(ro rendering.RenderObject) bool {
		if ro == nil {
			return false
		}
		if ro.Node() == ta {
			if rb := ro.(*rendering.RenderBox); rb != nil {
				tx, ty, tw, th = rb.X(), rb.Y(), rb.Width(), rb.Height()
				return true
			}
		}
		for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
			if findRO(c) {
				return true
			}
		}
		return false
	}
	findRO(rendering.RenderObject(rv))
	fmt.Printf("textarea box: x=%.0f y=%.0f w=%.0f h=%.0f\n", tx, ty, tw, th)
	// 渲染树 box 的 outline 状态
	var walk2 func(ro rendering.RenderObject) bool
	walk2 = func(ro rendering.RenderObject) bool {
		if ro == nil {
			return false
		}
		if ro.Node() == ta {
			if st := ro.Style(); st != nil {
				fmt.Printf("render tree textarea: OutlineSet=%v OutlineStyle=%q OutlineWidth=%.1f OutlineColor=#%02x%02x%02x borderColor=#%02x%02x%02x\n",
					st.OutlineSet, st.OutlineStyle, st.OutlineWidth.Value,
					st.OutlineColor.R, st.OutlineColor.G, st.OutlineColor.B,
					st.BorderColor("top").R, st.BorderColor("top").G, st.BorderColor("top").B)
			}
			return true
		}
		for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
			if walk2(c) {
				return true
			}
		}
		return false
	}
	walk2(rendering.RenderObject(rv))

	// 扫描上边缘的 accent 色像素宽度（outline 在 box 外上侧）
	// 扫描 x=tx+50 垂直线上 y 从 ty-6 到 ty 的像素
	outlinePixels := 0
	for y := int(ty) - 10; y < int(ty)+12; y++ {
		px := c.PixelAt(int(tx)+50, y)
		if px.R == 0x2a && px.G == 0xc3 && px.B == 0xde {
			fmt.Printf("  y=%d px=#%02x%02x%02x ← ACCENT\n", y, px.R, px.G, px.B)
			outlinePixels++
		}
	}
	fmt.Printf("outline pixels near textarea (accent #2ac3de): %d (want ~2)\n", outlinePixels)
	fmt.Println("=== outline_px_probe done ===")
}
