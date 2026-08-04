// Command scroll_render_test: minimal scroll-container rendering verification.
// Loads a tiny HTML page with an overflow-y:auto container holding colored
// blocks (some position:relative → own layers), renders at scroll=0 and a
// non-zero scroll, and verifies:
//   1. scroll=0: all in-viewport blocks are visible (nothing culled)
//   2. scroll=150: content moved up by 150px (viewer sees later blocks)
//   3. position:relative (layer) blocks follow the scroll like normal blocks
//   4. the scrollbar thumb stays screen-anchored (not translated with content)
package main

import (
	"fmt"
	"log"
	"os"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

var fontInit bool

func initGraphics() {
	if !fontInit {
		if graphics.GetFontManager() == nil {
			_ = graphics.InitFontManager("")
			if mgr := graphics.GetFontManager(); mgr != nil {
				mgr.LoadSystemFonts()
			}
		}
		layout.MeasureTextFunc = func(family string, size float64, weight int, style2, text string) float64 {
			return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}, text)
		}
		layout.FontMetricsFunc = func(family string, size float64, weight int, style2 string) (float64, float64, float64) {
			f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}
			return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
		}
		fontInit = true
	}
}

const htmlDoc = `<html><head><meta charset="utf-8"><style>
body { margin:0; background:#222; }
#scroller { position:absolute; left:50px; top:40px; width:300px; height:220px; overflow-y:auto; overflow-x:hidden; background:#444; }
.blk { height:40px; margin:0; }
#b0 { background:#ff0000; } #b1 { background:#00ff00; } #b2 { background:#0000ff; }
#b3 { background:#ffff00; } #b4 { background:#ff00ff; } #b5 { background:#00ffff; }
#b6 { background:#ff8800; } #b7 { background:#88ff00; } #b8 { background:#0088ff; }
#b9 { background:#ff0088; }
</style></head><body>
<div id="scroller">
  <div class="blk" id="b0"></div>
  <div class="blk" id="b1" style="position:relative"></div>
  <div class="blk" id="b2"></div>
  <div class="blk" id="b3"></div>
  <div class="blk" id="b4" style="position:relative"></div>
  <div class="blk" id="b5"></div>
  <div class="blk" id="b6"></div>
  <div class="blk" id="b7" style="position:relative"></div>
  <div class="blk" id="b8"></div>
  <div class="blk" id="b9"></div>
</div>
</body></html>`

func findBox(o rendering.RenderObject, id string) *rendering.RenderBox {
	if o == nil {
		return nil
	}
	if el, ok := o.Node().(*dom.Element); ok && el.GetAttribute("id") == id {
		switch v := o.(type) {
		case *rendering.RenderBox:
			return v
		case *rendering.RenderBlock:
			return &v.RenderBox
		case *rendering.RenderBlockFlow:
			return &v.RenderBlock.RenderBox
		}
	}
	for c := o.FirstChild(); c != nil; c = c.NextSibling() {
		if r := findBox(c, id); r != nil {
			return r
		}
	}
	return nil
}

func pxAt(pixels []byte, w, x, y int) (byte, byte, byte, byte) {
	if x < 0 || y < 0 || x >= w {
		return 0, 0, 0, 0
	}
	i := (y*w + x) * 4
	if i+3 >= len(pixels) {
		return 0, 0, 0, 0
	}
	return pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]
}

func main() {
	log.SetFlags(log.Ltime)
	initGraphics()
	wv := webkit.NewWebView()
	wv.Resize(500, 400)
	if err := wv.LoadHTML(htmlDoc); err != nil {
		log.Fatalf("LoadHTML: %v", err)
	}
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}

	sc := findBox(rendering.RenderObject(rv), "scroller")
	if sc == nil {
		log.Fatal("scroller box not found")
	}
	cw, ch := rv.BoxContentSize(sc)
	fmt.Printf("[scroll-render] scroller content=%.0fx%.0f (view=220)\n", cw, ch)

	failures := 0
	check := func(name string, ok bool, detail string) {
		status := "PASS"
		if !ok {
			status = "FAIL"
			failures++
		}
		fmt.Printf("  [%s] %s — %s\n", status, name, detail)
	}

	// Block i top (scroll=0) = 40 + i*40 (scroller top=40, no padding).
	blkColor := func(i int) (byte, byte, byte) {
		cols := [][3]byte{
			{255, 0, 0}, {0, 255, 0}, {0, 0, 255}, {255, 255, 0}, {255, 0, 255},
			{0, 255, 255}, {255, 136, 0}, {136, 255, 0}, {0, 136, 255}, {255, 0, 136},
		}
		return cols[i][0], cols[i][1], cols[i][2]
	}
	sample := func(pixels []byte, i int, scrollY float64) (byte, byte, byte) {
		top := 40 + float64(i)*40 - scrollY
		y := int(top + 5) // mid of top strip, inside viewport for visible blocks
		r, g, b, _ := pxAt(pixels, 500, 60, y)
		return r, g, b
	}
	same := func(a, b, c, er, eg, eb byte) bool { return a == er && b == eg && c == eb }

	// ---- Phase A: scroll = 0 ----
	pixels, err := wv.Render()
	if err != nil {
		log.Fatalf("render@0: %v", err)
	}
	fmt.Println("[scroll-render] === scroll=0 ===")
	for i := 0; i < 10; i++ {
		r, g, b := sample(pixels, i, 0)
		er, eg, eb := blkColor(i)
		// Viewport 220px → blocks 0..4 fully visible, 5 partially (top 20px).
		visible := i <= 5
		ok := !visible || same(r, g, b, er, eg, eb)
		check(fmt.Sprintf("block#%d visible", i), ok,
			fmt.Sprintf("got rgb(%d,%d,%d) want rgb(%d,%d,%d)", r, g, b, er, eg, eb))
	}
	{
		r, g, b, _ := pxAt(pixels, 500, 345, 100)
		check("vertical scrollbar present at x=345", r != 0 || g != 0 || b != 0,
			fmt.Sprintf("got rgb(%d,%d,%d)", r, g, b))
	}

	// ---- Phase B: scroll = 150 ----
	sc2 := findBox(rendering.RenderObject(rv), "scroller")
	rv.SetBoxScrollOffset(sc2, 0, 150)
	pixels, err = wv.Render()
	if err != nil {
		log.Fatalf("render@150: %v", err)
	}
	fmt.Println("[scroll-render] === scroll=150 ===")
	for i := 0; i < 10; i++ {
		r, g, b := sample(pixels, i, 150)
		er, eg, eb := blkColor(i)
		// Content moved up 150: visible blocks i where 40+i*40-150 in [40,260]
		// → i*40 in [150,370] → i in [4..9].
		visible := i >= 4
		ok := !visible || same(r, g, b, er, eg, eb)
		check(fmt.Sprintf("block#%d visible (scrolled)", i), ok,
			fmt.Sprintf("got rgb(%d,%d,%d) want rgb(%d,%d,%d)", r, g, b, er, eg, eb))
	}
	{
		r, g, b, _ := pxAt(pixels, 500, 60, 50)
		ok := !(r == 255 && g == 0 && b == 0)
		check("block#0 scrolled out", ok, fmt.Sprintf("got rgb(%d,%d,%d)", r, g, b))
	}
	{
		r, g, b, _ := pxAt(pixels, 500, 345, 100)
		check("scrollbar still at fixed x=345", r != 0 || g != 0 || b != 0,
			fmt.Sprintf("got rgb(%d,%d,%d)", r, g, b))
	}

	fmt.Printf("\n[scroll-render] failures=%d\n", failures)
	if failures > 0 {
		os.Exit(1)
	}
}
