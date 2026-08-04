// Command pxtranslate verifies at the PIXEL level that canvas Translate
// actually moves subsequent draws — i.e. that the whole paint pipeline
// (Translate -> draw rect -> readback) produces different pixels after a
// translate. If this passes, "content not scrolling" must come from a
// higher layer (dirty rect / clip / draw order), not the canvas.
package main

import (
	"fmt"

	"wb-ui/platform/graphics"
)

func main() {
	// Raster surface so we can read pixels back directly.
	canvas := graphics.NewCanvas(400, 300)
	canvas.Clear(graphics.Color{R: 0, G: 0, B: 0, A: 255})

	// Draw a red 50x50 rect at (20,20).
	canvas.FillRect(20, 20, 50, 50, graphics.Color{R: 255, G: 0, B: 0, A: 255})

	px := canvas.PixelAt(30, 30)
	fmt.Printf("before-translate pixel(30,30)=%v (expect red 255,0,0)\n", px)

	// Translate +50 y then draw a green 50x50 rect at (20,20): it should
	// land at y=70..120 in device space.
	canvas.Save()
	canvas.Translate(0, 50)
	canvas.FillRect(20, 20, 50, 50, graphics.Color{R: 0, G: 255, B: 0, A: 255})
	canvas.Restore()

	px1 := canvas.PixelAt(30, 30) // should stay red (green is at y=70+)
	px2 := canvas.PixelAt(30, 80) // should be green
	px3 := canvas.PixelAt(30, 75) // should be green (top edge)
	fmt.Printf("after-translate pixel(30,30)=%v (expect red)\n", px1)
	fmt.Printf("after-translate pixel(30,80)=%v (expect green 0,255,0)\n", px2)
	fmt.Printf("after-translate pixel(30,75)=%v (expect green)\n", px3)
	fmt.Println("--- done ---")
}
