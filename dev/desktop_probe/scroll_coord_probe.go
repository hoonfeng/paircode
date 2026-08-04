// Command scroll_coord_probe diagnoses "coordinate drift" after scrolling:
// it renders the file tree twice (scroll=0 and scroll=300), then for every
// text line reports whether ink appears at the theoretical position
// (absY - scroll) and whether residue remains at the original position.
// A correct scroll moves every line by exactly -scroll; any line that
// keeps ink at its original spot (or misses the theoretical spot) is
// drifting relative to its siblings.
package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func setupLoaders(wv *webkit.WebView, distDir string) {
	absDist, _ := filepath.Abs(distDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				return string(data), err
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				if err != nil {
					return "", err
				}
				re := regexp.MustCompile(`\[data-v-[a-f0-9]+\]`)
				return re.ReplaceAllString(string(data), ""), nil
			}
		}
	}
}

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

func clsOf(rb *rendering.RenderBox) string {
	if el, ok := rb.Node().(*dom.Element); ok {
		return el.GetAttribute("class")
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

// inkIn counts non-background pixels in a rect region of the canvas.
func inkIn(c *graphics.Canvas, x0, y0, x1, y1 int) int {
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	w := c.Width()
	h := c.Height()
	if x1 > w {
		x1 = w
	}
	if y1 > h {
		y1 = h
	}
	if x0 >= x1 || y0 >= y1 {
		return 0
	}
	px := c.Pixels()
	bg := []byte{22, 27, 34}
	cnt := 0
	for y := y0; y < y1 && y < c.Height(); y++ {
		for x := x0; x < x1 && x < w; x++ {
			i := (y*w + x) * 4
			d := int(px[i]) - int(bg[0])
			if d < 0 {
				d = -d
			}
			d2 := int(px[i+1]) - int(bg[1])
			if d2 < 0 {
				d2 = -d2
			}
			d3 := int(px[i+2]) - int(bg[2])
			if d3 < 0 {
				d3 = -d3
			}
			if d+d2+d3 > 90 {
				cnt++
			}
		}
	}
	return cnt
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, err := os.ReadFile(filepath.Join(distDir, "index.html"))
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}

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

	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	for i := 0; i < 8; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
	}
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 800); })`)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}

	// Collect layer owners so we can flag lines inside a child layer.
	layerOwners := map[rendering.RenderObject]bool{}
	var collectLayers func(l *rendering.RenderLayer)
	collectLayers = func(l *rendering.RenderLayer) {
		if l == nil {
			return
		}
		if o := l.Owner(); o != nil {
			layerOwners[o] = true
		}
		for c := l.FirstChild(); c != nil; c = c.NextSibling() {
			collectLayers(c)
		}
	}
	if rl := rv.RootLayer(); rl != nil {
		collectLayers(rl)
	}

	// Find the file-tree scroll container.
	var section *rendering.RenderBox
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil || section != nil {
			return
		}
		if rb := asRB(o); rb != nil && rb.Style() != nil && clsOf(rb) == "project-section" {
			section = rb
			return
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	if section == nil {
		log.Fatal("no project-section")
	}
	log.Printf("project-section frame=%.0f,%.0f %.0fx%.0f", section.X(), section.Y(), section.Width(), section.Height())

	// Dump the layer subtree rooted at the section's layer.
	var dumpLayers func(l *rendering.RenderLayer, depth int)
	dumpLayers = func(l *rendering.RenderLayer, depth int) {
		if l == nil {
			return
		}
		desc := ""
		if o := l.Owner(); o != nil {
			desc = clsOf(asRB(o))
			if rb := asRB(o); rb != nil && rb.Style() != nil {
				st := rb.Style()
				desc += fmt.Sprintf(" pos=%v ov=%v/%v op=%v", st.Position, st.OverflowX, st.OverflowY, st.Opacity)
			}
		}
		log.Printf("  %sLAYER[%d] %s", strings.Repeat("  ", depth), depth, desc)
		for c := l.FirstChild(); c != nil; c = c.NextSibling() {
			dumpLayers(c, depth+1)
		}
	}
	if rl := rv.RootLayer(); rl != nil {
		var findSectionLayer func(l *rendering.RenderLayer) *rendering.RenderLayer
		findSectionLayer = func(l *rendering.RenderLayer) *rendering.RenderLayer {
			if l == nil {
				return nil
			}
			if l.Owner() != nil {
				if rb := asRB(l.Owner()); rb != nil && rb == section {
					return l
				}
			}
			for c := l.FirstChild(); c != nil; c = c.NextSibling() {
				if found := findSectionLayer(c); found != nil {
					return found
				}
			}
			return nil
		}
		if sl := findSectionLayer(rl); sl != nil {
			log.Printf("section layer subtree:")
			dumpLayers(sl, 0)
		} else {
			log.Printf("WARNING: no layer owns project-section")
		}
	}


	// Collect text lines inside the section subtree.
	type line struct {
		text   string
		segX   float64
		segY   float64
		segW   float64
		cls    string
		inLayer bool
	}
	var lines []line
	var walkT func(o rendering.RenderObject)
	walkT = func(o rendering.RenderObject) {
		if o == nil {
			return
		}
		if rt, ok := o.(*rendering.RenderText); ok {
			txt := strings.TrimSpace(rt.Text())
			if txt == "" || strings.ContainsAny(txt, "\n") {
				// skip multiline blocks; only leaf labels
			}
			segs := rt.Segments()
			if len(segs) > 0 {
				l := line{text: txt, segX: segs[0].X, segY: segs[0].Y, segW: segs[0].Width}
				if rb := asRB(rt.Parent()); rb != nil {
					l.cls = clsOf(rb)
				}
				// is any ancestor a layer owner?
				for p := rt.Parent(); p != nil; p = p.Parent() {
					if layerOwners[p] {
						l.inLayer = true
						break
					}
				}
				if l.text != "" {
					lines = append(lines, l)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walkT(c)
		}
	}
	walkT(rendering.RenderObject(section))

	const scrollY = 600.0
	outDir := filepath.Join(wd, "dev", "desktop_probe")

	// Frame 1: no scroll.
	c1 := graphics.NewCanvas(1280, 800)
	rendering.Paint(rv, c1, rendering.Rect{X: 0, Y: 0, Width: 1280, Height: 800})
	saveCanvas(c1, filepath.Join(outDir, "sc_top.png"))

	// Frame 2: scrolled 300.
	rv.SetBoxScrollOffset(section, 0, scrollY)
	c2 := graphics.NewCanvas(1280, 800)
	rendering.Paint(rv, c2, rendering.Rect{X: 0, Y: 0, Width: 1280, Height: 800})
	saveCanvas(c2, filepath.Join(outDir, "sc_scroll600.png"))

	// Report per line: ink at original pos in frame1, at theoretical pos in
	// frame2, and residue at original pos in frame2.
	log.Printf("lines collected: %d", len(lines))
	shown := 0
	for _, l := range lines {
		if l.segY < section.Y() || l.segY > section.Y()+section.Height() {
			continue // only lines inside the initial viewport
		}
		shown++
		x0 := int(l.segX) - 2
		x1 := int(l.segX+l.segW) + 2
		y0 := int(l.segY)
		y1 := y0 + 16
		origInk := inkIn(c1, x0, y0, x1, y1)
		ty0 := int(l.segY - scrollY)
		theoInk := inkIn(c2, x0, ty0, x1, ty0+16)
		residInk := inkIn(c2, x0, y0, x1, y1)
		flag := ""
		if theoInk < 3 {
			flag = " !!NO-INK-AT-THEORY"
		}
		if residInk > 2 {
			flag += " !!RESIDUE"
		}
		layerMark := ""
		if l.inLayer {
			layerMark = " [LAYER]"
		}
		log.Printf("  y=%5.1f theo=%5.1f origInk=%3d theoInk=%3d resid=%3d %q cls=%s%s%s",
			l.segY, l.segY-scrollY, origInk, theoInk, residInk, l.text, l.cls, layerMark, flag)
	}
	log.Printf("checked %d lines", shown)
}
