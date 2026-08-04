// Command chat_scroll_probe renders the chat area (scroll container
// .chat-messages) at scroll 0/300/600 and dumps the absolute geometry of
// text segments inside the first few msg-bubbles. The Python side verifies
// row-projection continuity: content must move by exactly the scroll delta
// per frame.
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

func setupLoaders2(wv *webkit.WebView, distDir string) {
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

func asRB2(o rendering.RenderObject) *rendering.RenderBox {
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

func clsOf2(rb *rendering.RenderBox) string {
	if el, ok := rb.Node().(*dom.Element); ok {
		return el.GetAttribute("class")
	}
	return ""
}

func saveCanvas2(c *graphics.Canvas, path string) {
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
	setupLoaders2(wv, distDir)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	// ★ 与 desktop 完全一致：desktop 用 NewHost(wv, 1280, 800)（LoadHTML 后 Resize）。
	//   不 Resize 的话视口是默认 800x600，flex 布局/消息宽度/滚动容器高度全不同，
	//   测出的布局不是 desktop 的真实布局。
	wv.Resize(1280, 800)
	log.Printf("viewport after Resize = %dx%d", wv.Width(), wv.Height())
	for i := 0; i < 20; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
	}
	for i := 0; i < 8; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 1000); })`)
	}
	// 等待 DOM 稳定：msg-item 数量连续 3 次采样相同才继续（聊天数据异步加载）
	stable := false
	last := -1
	same := 0
	for i := 0; i < 15 && !stable; i++ {
		v, _ := wv.EvalJS(`document.querySelectorAll('.msg-item').length`)
		msgCount := int(v.ToNumber())
		log.Printf("poll msg-item=%d (same=%d)", msgCount, same)
		if msgCount == last && msgCount > 0 {
			same++
			if same >= 3 {
				stable = true
			}
		} else {
			same = 0
		}
		last = msgCount
		if !stable {
			_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 1000); })`)
		}
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}

	// How many msg-bubbles exist in the DOM?
	if v, err := wv.EvalJS(`document.querySelectorAll('.msg-item').length`); err == nil {
		log.Printf("msg-item count in DOM = %v", v)
	} else {
		log.Printf("msg-item query failed: %v", err)
	}

	var chat *rendering.RenderBox
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if o == nil || chat != nil {
			return
		}
		if rb := asRB2(o); rb != nil && rb.Style() != nil && clsOf2(rb) == "chat-messages" {
			chat = rb
			return
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	if chat == nil {
		log.Fatal("no chat-messages")
	}
	log.Printf("chat-messages frame=%.0f,%.0f %.0fx%.0f", chat.X(), chat.Y(), chat.Width(), chat.Height())

	outDir := filepath.Join(wd, "dev", "desktop_probe")
	scrolls := []float64{0, 300, 600}
	for _, sy := range scrolls {
		if sy > 0 {
			rv.SetBoxScrollOffset(chat, 0, sy)
		}
		gotX, gotY := rv.BoxScrollOffset(chat)
		log.Printf("before paint: set sy=%.0f  read back=(%.0f, %.0f)", sy, gotX, gotY)
		log.Printf("=== SKIA-PHASE scroll=%.0f ===", sy)
		c := graphics.NewCanvas(1280, 800)
		rendering.Paint(rv, c, rendering.Rect{X: 0, Y: 0, Width: 1280, Height: 800})
		name := fmt.Sprintf("chat_s%03.0f.png", sy)
		saveCanvas2(c, filepath.Join(outDir, name))
	}
	log.Printf("chat box content: frame=%.0f,%.0f %.0fx%.0f", chat.X(), chat.Y(), chat.Width(), chat.Height())
	log.Printf("done")
}
