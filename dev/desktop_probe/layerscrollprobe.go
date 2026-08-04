// Command layerscrollprobe verifies at the PIXEL level that a scroll offset
// on an overflow:auto container actually moves the pixels of a child that
// owns its own layer (position:relative). This is the exact desktop scenario
// (conv-list scrolls its conv-item children). If pixels do NOT move, the
// layer paint path is dropping the scroll translate.
package main

import (
	"fmt"
	"os"

	"wb-ui/dom"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/style"
)

func main() {
	os.Setenv("WB_LAYER_DEBUG", "1")
	os.Setenv("WB_SCROLL_DEBUG", "1")
	doc := dom.NewDocument()
	rv := rendering.NewRenderView(doc, style.NewComputedStyle())
	rv.SetViewportSize(400, 300)

	// Scroll container: overflow:auto, 200x200 at (0,0).
	contStyle := style.NewComputedStyle()
	contStyle.OverflowY = style.OverflowAuto
	contStyle.OverflowX = style.OverflowVisible
	cont := rendering.NewRenderBox(doc.CreateElement("div"), contStyle)
	cont.SetLocation(0, 0)
	cont.SetSize(200, 200)

	// Child A: position:relative (=> own layer), 100x50 at (10,10).
	childStyle := style.NewComputedStyle()
	childStyle.Position = style.PositionRelative
	childStyle.BackgroundColor = style.Color{R: 0xFF, G: 0, B: 0, A: 0xFF}
	child := rendering.NewRenderBox(doc.CreateElement("div"), childStyle)
	child.SetLocation(10, 10)
	child.SetSize(100, 50)
	cont.AddChild(child, nil)

	// Child B: position:static (no layer), 100x50 at (120,10).
	childB := rendering.NewRenderBox(doc.CreateElement("div"), style.NewComputedStyle())
	childB.SetLocation(120, 10)
	childB.SetSize(100, 50)
	childB.Style().BackgroundColor = style.Color{R: 0, G: 0xFF, B: 0, A: 0xFF}
	cont.AddChild(childB, nil)
	rv.AddChild(cont, nil)

	// Build layer tree (cont: overflow:auto -> layer; child: relative -> layer).
	comp := rendering.NewRenderLayerCompositor(rv)
	rootLayer := comp.BuildLayerTree(rendering.RenderObject(rv))
	rv.SetRootLayer(rootLayer)

	// Paint without scroll: red at (30,30).
	canvas1 := graphics.NewCanvas(400, 300)
	rendering.Paint(rv, canvas1, rendering.Rect{X: 0, Y: 0, Width: 400, Height: 300})
	p1 := canvas1.PixelAt(30, 30)
	fmt.Printf("no-scroll pixel(30,30)=%v (expect red)\n", p1)

	// Set scroll offset +20: child (y=10..60) appears at y=-10..40. So
	// (30,40) should still be red (child covers it), (30,50) should be bg.
	rv.SetBoxScrollOffset(cont, 0, 20)
	canvas2 := graphics.NewCanvas(400, 300)
	rendering.Paint(rv, canvas2, rendering.Rect{X: 0, Y: 0, Width: 400, Height: 300})
	p2 := canvas2.PixelAt(30, 50) // child moved up: y=50-20=30 -> still inside child? no: child y=10..60 minus 20 = -10..40, so y=50 is OUTSIDE -> bg
	fmt.Printf("scroll20 pixel(30,50)=%v (expect bg #00000000)\n", p2)

	// Compare: if (30,40) changed from red to something while (30,50) is bg,
	// content moved. Actually verify (30,40) is still red and (30,50) not.
	p3 := canvas2.PixelAt(30, 40) // child y=40-20=20 -> inside child -> red
	fmt.Printf("scroll20 pixel(30,40)=%v (expect red if moved)\n", p3)

	// Child B (static, no layer): at (120,10), scroll -20 -> y=-10..40.
	pB := canvas2.PixelAt(130, 40) // inside B if moved: y=40-20=20, x=130
	fmt.Printf("scroll20 pixel(130,40)=%v (childB static, expect green if moved)\n", pB)

	// 直接验证：child 未移动时应 (30,30) 红；移动后 (30,30) 应仍红（y=30 在 -10..40 内）
	// 若 translate 未生效，(30,30) 红而 (30,40) 也红（child 未动覆盖 30..50）
	p30 := canvas2.PixelAt(30, 30)
	fmt.Printf("scroll20 pixel(30,30)=%v (child y=10..60; moved -10..40 -> red; unmoved -> red)\n", p30)
	p31 := canvas2.PixelAt(30, 45)
	fmt.Printf("scroll20 pixel(30,45)=%v (moved: y=45-20=25 inside -> red; unmoved: y=45 inside 10..60 -> red)\n", p31)
	p32 := canvas2.PixelAt(30, 55)
	fmt.Printf("scroll20 pixel(30,55)=%v (moved: y=55-20=35 inside -> red; unmoved: y=55 inside -> red)\n", p32)

	red := graphics.Color{R: 0xFF, G: 0, B: 0, A: 0xFF}
	green := graphics.Color{R: 0, G: 0xFF, B: 0, A: 0xFF}
	if p3 == red {
		fmt.Println("RESULT-A: LAYER CHILD MOVED ✓ (pixel 30,40 red)")
	} else {
		fmt.Println("RESULT-A: LAYER CHILD DID NOT MOVE ✗")
	}
	if pB == green {
		fmt.Println("RESULT-B: STATIC CHILD MOVED ✓ (pixel 130,40 green)")
	} else {
		fmt.Println("RESULT-B: STATIC CHILD DID NOT MOVE ✗")
	}
	fmt.Println("--- done ---")
}
