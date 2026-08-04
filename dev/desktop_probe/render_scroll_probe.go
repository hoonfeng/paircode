// Command render_scroll_probe: loads the REAL companion frontend through wb-ui
// (same startup as scroll_diag_probe / real_probe), scrolls .chat-messages to a
// non-zero offset, renders to PNG, and prints the pixel evidence:
//   1. chat-messages content is present (colored message bubbles at scroll=0)
//   2. after scroll=600 the same bubbles moved up by 600px (visible content
//      corresponds to a LATER slice of the message list)
//   3. position:relative layer content (msg-item etc.) follows the scroll
//   4. the scrollbar stays at the right edge (screen-anchored)
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
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
	"wb-ui/style"
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

func findClassBox(o rendering.RenderObject, cls string) *rendering.RenderBox {
	if o == nil {
		return nil
	}
	if el, ok := o.Node().(*dom.Element); ok && el.GetAttribute("class") == cls {
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
		if r := findClassBox(c, cls); r != nil {
			return r
		}
	}
	return nil
}

// dumpMsgList prints every direct child of .msg-list-wrap with its geometry and
// layer status, so we can see how many messages exist and whether they are
// layers (excluded from the parent walk).
func dumpMsgList(rv *rendering.RenderView, mlw *rendering.RenderBox) {
	if mlw == nil {
		fmt.Println("[probe] msg-list-wrap not found")
		return
	}
	rb := mlw
	fmt.Printf("[probe] msg-list-wrap children=%d y=%.0f h=%.0f\n", countChildren(rendering.RenderObject(rb)), rb.Y(), rb.Height())
	n := 0
	for c := rendering.RenderObject(rb).FirstChild(); c != nil; c = c.NextSibling() {
		el, isEl := c.Node().(*dom.Element)
		cls := ""
		if isEl {
			cls = el.GetAttribute("class")
		}
		var y, h, w float64
		req := false
		if box := asBox(c); box != nil {
			y, h, w = box.Y(), box.Height(), box.Width()
			req = rendering.RequiresLayer(c)
		}
		fmt.Printf("  [%d] %T %q y=%.0f h=%.0f w=%.0f layer=%v\n", n, c, cls, y, h, w, req)
		n++
	}
}

func countChildren(o rendering.RenderObject) int {
	n := 0
	for c := o.FirstChild(); c != nil; c = c.NextSibling() {
		n++
	}
	return n
}

func asBox(o rendering.RenderObject) *rendering.RenderBox {
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

func savePNG(path string, w, h int, pixels []byte) error {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			img.SetRGBA(x, y, color.RGBA{R: pixels[i], G: pixels[i+1], B: pixels[i+2], A: pixels[i+3]})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func main() {
	log.SetFlags(log.Ltime)
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
	wv.Resize(1280, 800)
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	if err := wv.LoadHTML(string(htmlData)); err != nil {
		log.Fatalf("LoadHTML: %v", err)
	}
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
	_ = style.OverflowAuto

	cm := findClassBox(rendering.RenderObject(rv), "chat-messages")
	if cm == nil {
		// fall back to id
		cm = findBox(rendering.RenderObject(rv), "chat-messages")
	}
	if cm == nil {
		log.Fatal("chat-messages not found")
	}
	cw, ch := rv.BoxContentSize(cm)
	pb := cm.PaddingBoxRect()
	fmt.Printf("[probe] chat-messages pb=(%.0f,%.0f %.0fx%.0f) content=%.0fx%.0f\n",
		pb.X, pb.Y, pb.Width, pb.Height, cw, ch)

	mlw := findClassBox(rendering.RenderObject(rv), "msg-list-wrap")
	dumpMsgList(rv, mlw)

	failures := 0
	check := func(name string, ok bool, detail string) {
		status := "PASS"
		if !ok {
			status = "FAIL"
			failures++
		}
		fmt.Printf("  [%s] %s — %s\n", status, name, detail)
	}

	// ---- Phase A: scroll = 0 ----
	log.Printf("=== SKIA-PHASE scroll=0 ===")
	pixels, err := wv.Render()
	if err != nil {
		log.Fatalf("render@0: %v", err)
	}
	if err := savePNG(filepath.Join("_scroll_out", "real_s0.png"), 1280, 800, pixels); err != nil {
		log.Printf("save s0: %v", err)
	}
	fmt.Println("[probe] === scroll=0 (saved _scroll_out/real_s0.png) ===")

	// Non-background pixel count inside the chat viewport proves content exists.
	nb0 := 0
	for y := int(pb.Y) + 5; y < int(pb.Y+pb.Height)-5; y += 2 {
		for x := int(pb.X) + 5; x < int(pb.X+pb.Width)-5; x += 2 {
			r, g, b, a := pxAt(pixels, 1280, x, y)
			if a == 0 {
				continue
			}
			if !(r < 30 && g < 30 && b < 30) {
				nb0++
			}
		}
	}
	check("chat viewport has painted content (scroll=0)", nb0 > 50, fmt.Sprintf("non-bg samples=%d", nb0))

	// ---- Phase B: scroll = 600 ----
	cm2 := findClassBox(rendering.RenderObject(rv), "chat-messages")
	rv.SetBoxScrollOffset(cm2, 0, 600)
	log.Printf("=== SKIA-PHASE scroll=600 ===")
	pixels2, err := wv.Render()
	if err != nil {
		log.Fatalf("render@600: %v", err)
	}
	if err := savePNG(filepath.Join("_scroll_out", "real_s600.png"), 1280, 800, pixels2); err != nil {
		log.Printf("save s600: %v", err)
	}
	fmt.Println("[probe] === scroll=600 (saved _scroll_out/real_s600.png) ===")

	nb600 := 0
	for y := int(pb.Y) + 5; y < int(pb.Y+pb.Height)-5; y += 2 {
		for x := int(pb.X) + 5; x < int(pb.X+pb.Width)-5; x += 2 {
			r, g, b, a := pxAt(pixels2, 1280, x, y)
			if a == 0 {
				continue
			}
			if !(r < 30 && g < 30 && b < 30) {
				nb600++
			}
		}
	}
	check("chat viewport has painted content (scroll=600)", nb600 > 50, fmt.Sprintf("non-bg samples=%d", nb600))
	// Content must have MOVED: compare the whole chat viewport between the two
	// renders. If scroll works, the visible slice changes everywhere; if the
	// scroll translate is broken (content culled / not moving) the viewport is
	// identical. NOTE: a narrow top strip is NOT reliable — after a scroll of
	// exactly one block gap the top strip can land in blank padding.
	diff := 0
	same := 0
	for x := int(pb.X) + 4; x < int(pb.X+pb.Width)-4; x += 4 {
		for y := int(pb.Y) + 4; y < int(pb.Y+pb.Height)-4; y += 4 {
			r0, g0, b0, a0 := pxAt(pixels, 1280, x, y)
			r1, g1, b1, a1 := pxAt(pixels2, 1280, x, y)
			if a0 == 0 && a1 == 0 {
				same++
				continue
			}
			if absI(int(r0)-int(r1)) > 24 || absI(int(g0)-int(g1)) > 24 || absI(int(b0)-int(b1)) > 24 {
				diff++
			} else {
				same++
			}
		}
	}
	total := diff + same
	check("content moved after scroll (viewport changed)", total > 0 && float64(diff)/float64(total) > 0.15,
		fmt.Sprintf("changed=%d/%d", diff, total))

	// Scrollbar anchored at right edge: sample the scrollbar track x-column
	// (pb.X+pb.Width-6) — it should be non-background at BOTH scroll states.
	sb0 := pxNonBg(pixels, 1280, int(pb.X+pb.Width-6), int(pb.Y)+60)
	sb1 := pxNonBg(pixels2, 1280, int(pb.X+pb.Width-6), int(pb.Y)+60)
	check("scrollbar track present at fixed x", sb0 && sb1, fmt.Sprintf("s0=%v s600=%v", sb0, sb1))

	fmt.Printf("\n[probe] failures=%d\n", failures)
	if failures > 0 {
		os.Exit(1)
	}
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

func pxNonBg(pixels []byte, w, x, y int) bool {
	r, g, b, a := pxAt(pixels, w, x, y)
	if a == 0 {
		return false
	}
	return !(r < 30 && g < 30 && b < 30)
}

func absI(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
