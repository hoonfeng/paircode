// Command range_ui_shot 渲染独立 range 页面（对照 ide_ref_range.html），
// 输出 PNG 供像素对比 wb-ui 滑块 vs Edge 标准。
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func runJobs(wv *webkit.WebView) {
	for i := 0; i < 15; i++ {
		wv.JSInterpreter().RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	// 独立 HTML：两行 range（对照 Edge 的 r1=0.3/2, r2=42）
	html := `<html><head><style>
		body { margin: 0; background: #0d1117; font-family: "Segoe UI", sans-serif; }
		.wrap { padding: 60px 80px; }
		.row { display: flex; align-items: center; gap: 8px; padding: 6px 0; }
		.row label { width: 120px; font-size: 13px; color: #e6edf3; }
		.row input[type="range"] { flex: 1; }
		.val { width: 30px; text-align: center; font-size: 12px; color: #8b949e; }
	</style></head><body>
	<div class="wrap">
	  <div class="row"><label>温度</label><input type="range" min="0" max="2" step="0.1" value="0.3" /><span class="val">0.3</span></div>
	  <div class="row"><label>音量</label><input type="range" min="0" max="100" step="1" value="42" /><span class="val">42</span></div>
	</div>
	</body></html>`

	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style2, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}, text)
	}

	wv := webkit.NewWebView()
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	wv.SetConsoleLogger(&jsc.BufferLogger{})
	desktopbridge.Init(wv)
	wv.LoadHTML(html)
	for i := 0; i < 2; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
		time.Sleep(350 * time.Millisecond)
		runJobs(wv)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)

	pngBytes, err := wv.Render()
	if err != nil {
		log.Fatalf("Render: %v", err)
	}
	W, H := wv.Width(), wv.Height()
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			off := (y*W + x) * 4
			if off+3 < len(pngBytes) {
				img.SetRGBA(x, y, color.RGBA{R: pngBytes[off], G: pngBytes[off+1], B: pngBytes[off+2], A: pngBytes[off+3]})
			}
		}
	}
	out := filepath.Join(wd, "dev", "desktop_probe", "wbui_range.png")
	f, _ := os.Create(out)
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatalf("encode: %v", err)
	}
	fmt.Printf("saved %dx%d → %s\n", W, H, out)
}
