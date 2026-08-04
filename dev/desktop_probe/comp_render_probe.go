// Command comp_render_probe loads the REAL companion frontend through wb-ui
// (same path as cmd/desktop), renders a frame offscreen via WebView.Render()
// and writes the pixels to PNG. Used to verify comp-bar corner rounding
// without needing a GPU window capture.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/dom"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func main() {
	log.SetFlags(log.Ltime)
	os.Setenv("WB_PAINT_DEBUG", "1")  // 调试：打印 paint walk clip 日志
	os.Setenv("WB_CLIP_DEBUG", "1")   // 调试：打印 Canvas.ClipRoundRect 日志
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	absDist, _ := filepath.Abs(distDir)
	htmlData, err := os.ReadFile(filepath.Join(absDist, "index.html"))
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}
	log.Printf("[probe] dist=%s html=%d bytes", distDir, len(htmlData))

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
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	for i := 0; i < 8; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
	}
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
	_, _ = wv.JSInterpreter().RunJS(`
		var el = document.querySelector('.comp-bar');
		if (el) {
			var cs = getComputedStyle(el);
			console.log('[PROBE/comp] borderRadius=' + cs.borderRadius + ' overflow=' + cs.overflow + ' h=' + cs.height + ' bg=' + cs.background);
			console.log('[PROBE/comp] class=' + el.className + ' rect=' + JSON.stringify(el.getBoundingClientRect()));
			var seg = el.firstElementChild;
			if (seg) { var cs2 = getComputedStyle(seg); console.log('[PROBE/seg] class=' + seg.className + ' w=' + cs2.width + ' bg=' + cs2.background + ' h=' + cs2.height); }
		} else { console.log('[PROBE/comp] NOT FOUND'); }
	`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("=== COMP PROBE ===")
	// Go 侧遍历渲染树：直接读 comp-bar 的计算样式
	rv := wv.RenderView()
	if rv != nil {
		count := 0
		var walk func(o rendering.RenderObject, depth int)
		walk = func(o rendering.RenderObject, depth int) {
			if o == nil {
				return
			}
			count++
			if nn := o.Node(); nn != nil {
				if el, ok2 := nn.(*dom.Element); ok2 {
					cls := el.GetAttribute("class")
					if cls == "comp-bar" {
						st := o.Style()
						if st != nil {
							fmt.Printf("[GOPROBE] comp-bar BorderRadius=%s BorderW=%s/%s overflowY=%v\n",
								st.BorderRadius, st.BorderTopWidth, st.BorderLeftWidth, st.OverflowY)
						}
					}
				}
			}
			if depth < 3 {
				fmt.Printf("[GOPROBE] d%d %T\n", depth, o)
			}
			for c := o.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c, depth+1)
			}
		}
		walk(rv, 0)
		fmt.Printf("[GOPROBE] total=%d\n", count)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	pixels, err := wv.Render()
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	w, h := 1280, 800
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := (y*w + x) * 4
			img.Set(x, y, color.RGBA{R: pixels[off], G: pixels[off+1], B: pixels[off+2], A: pixels[off+3]})
		}
	}
	out := filepath.Join(wd, "dev", "desktop_probe", "_comp_render.png")
	f, err := os.Create(out)
	if err != nil {
		log.Fatalf("create: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatalf("encode: %v", err)
	}
	fmt.Println("saved", out)
}

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
				cleaned := re.ReplaceAllString(string(data), "")
				return cleaned, nil
			}
		}
	}
}
