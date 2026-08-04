// Command minimal_scroll_probe isolates the "scrolled-in content not painted"
// bug with a minimal DOM: one overflow:auto container holding 10 rows of
// text. Paint frame 1 (top), scroll to max, paint frame 2, and diff the
// viewport region. If frame 2 shows the SAME rows as frame 1, the box scroll
// translate is not reaching the text painters.
package main

import (
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

func asRB(o rendering.RenderObject) *rendering.RenderBox {
	switch v := o.(type) {
	case *rendering.RenderBox:
		return v
	case *rendering.RenderBlock:
		return &v.RenderBox
	case *rendering.RenderBlockFlow:
		return &v.RenderBlock.RenderBox
	}
	return nil
}

func idOf(rb *rendering.RenderBox) string {
	if el, ok := rb.Node().(*dom.Element); ok {
		return el.GetAttribute("id")
	}
	return ""
}

func saveCanvas(c *graphics.Canvas, path string) {
	w, h := c.Width(), c.Height()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	px := c.Pixels()
	for i := 0; i < w*h; i++ {
		img.Pix[i*4+0] = px[i*4+0]
		img.Pix[i*4+1] = px[i*4+1]
		img.Pix[i*4+2] = px[i*4+2]
		img.Pix[i*4+3] = px[i*4+3]
	}
	f, err := os.Create(path)
	if err != nil {
		log.Printf("save %s: %v", path, err)
		return
	}
	defer f.Close()
	png.Encode(f, img)
	log.Printf("saved %s (%dx%d)", path, w, h)
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

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

	rows := ""
	for i := 0; i < 10; i++ {
		rows += `<div class="row" id="row-` + string(rune('0'+i)) + `">ROW-` + string(rune('0'+i)) + `</div>`
	}
	html := `<html><head><style>
body { margin: 0; font-size: 14px; }
#scroller { position: absolute; left: 0; top: 0; width: 200px; height: 150px; overflow-y: auto; background: #ffffff; }
.row { position: relative; height: 30px; line-height: 30px; color: #000000; }
</style></head><body><div id="scroller">` + rows + `</div></body></html>`

	wv := webkit.NewWebView()
	wv.LoadHTML(html)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}
	log.Printf("viewport %dx%d", rv.ViewWidth(), rv.ViewHeight())

	// Find the scroller box.
	var scroller *rendering.RenderBox
	nbox := 0
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil {
			return
		}
		if rb := asRB(o); rb != nil {
			nbox++
			if os.Getenv("WB_TRACE_BOX") != "" {
				nm := "?"
				if n := rb.Node(); n != nil {
					nm = n.NodeName()
				}
				log.Printf("box[%d] tag=%s id=%q x=%.0f y=%.0f w=%.0f h=%.0f",
					nbox, nm, idOf(rb), rb.X(), rb.Y(), rb.Width(), rb.Height())
			}
			if idOf(rb) == "scroller" {
				scroller = rb
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	log.Printf("walked %d boxes", nbox)
	if scroller == nil {
		log.Fatal("scroller not found")
	}
	m := rendering.VerticalScrollbarMetrics(rv, scroller)
	log.Printf("scroller view=%.0f total=%.0f maxScroll=%.0f", m.ViewLen, m.TotalLen, m.MaxScroll)
	if !m.OK {
		log.Fatal("no scrollbar metrics")
	}

	outDir := filepath.Join("dev", "desktop_probe")

	// Frame 1: no scroll.
	c1 := graphics.NewCanvas(200, 150)
	rendering.Paint(rv, c1, rendering.Rect{X: 0, Y: 0, Width: 200, Height: 150})
	saveCanvas(c1, filepath.Join(outDir, "ms_top.png"))

	// Frame 2: scrolled to max.
	rv.SetBoxScrollOffset(scroller, 0, m.MaxScroll)
	c2 := graphics.NewCanvas(200, 150)
	rendering.Paint(rv, c2, rendering.Rect{X: 0, Y: 0, Width: 200, Height: 150})
	saveCanvas(c2, filepath.Join(outDir, "ms_bottom.png"))

	// Diff the whole canvas (scrollbar strip excluded: x 0..195).
	diffs, total := 0, 0
	for y := 0; y < 150; y++ {
		for x := 0; x < 195; x++ {
			i := (y*200 + x) * 4
			d := 0
			for k := 0; k < 3; k++ {
				v := int(c1.Pixels()[i+k]) - int(c2.Pixels()[i+k])
				if v < 0 {
					v = -v
				}
				d += v
			}
			if d > 60 {
				diffs++
			}
			total++
		}
	}
	log.Printf("diff: %d/%d pixels changed (%.1f%%)", diffs, total, float64(diffs)*100/float64(total))
	log.Printf("scroll offsets=%d", rv.ScrollOffsetCount())
}
