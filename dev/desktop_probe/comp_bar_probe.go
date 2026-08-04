// comp_bar_probe：加载真实前端，取 comp-bar 精确矩形并 dump 两端圆角区域像素
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
	os.Setenv("WB_PAINT_DEBUG", "1")
	os.Setenv("WB_CLIP_DEBUG", "1")
	wd, _ := os.Getwd()
	absDist, _ := filepath.Abs(filepath.Join(wd, "cmd", "companion", "web-ui", "dist"))
	htmlData, err := os.ReadFile(filepath.Join(absDist, "index.html"))
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
	setupLoaders(wv, absDist)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	for i := 0; i < 8; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
	}
	// 取 comp-bar 与 ctx-bar 的 getBoundingClientRect（通过 window.__probe 返回）
	_, _ = wv.JSInterpreter().RunJS(`
		var bar = document.querySelector('.comp-bar');
		if (bar) {
			// 模拟真实数据：注入 6 段 seg（对应 system/skills/mcp/tool/history/other）
			var colors = ['#58a6ff', '#3fb950', '#a371f7', '#d29922', '#f778ba', '#8b949e'];
			var widths = [18, 12, 8, 25, 22, 15];
			bar.innerHTML = '';
			for (var i = 0; i < 6; i++) {
				var seg = document.createElement('div');
				seg.className = 'comp-bar-seg';
				seg.style.width = widths[i] + '%';
				seg.style.height = '100%';
				seg.style.background = colors[i];
				seg.style.display = 'inline-block';
				bar.appendChild(seg);
			}
		}
		var r = {};
		['.comp-bar', '.ctx-bar'].forEach(function(sel, i){
			var el = document.querySelector(sel);
			if (el) {
				var b = el.getBoundingClientRect();
				r[sel] = {x: b.x, y: b.y, w: b.width, h: b.height, cls: el.className};
				var cs = getComputedStyle(el);
				r[sel].radius = cs.borderRadius;
				r[sel].bg = cs.backgroundColor;
				r[sel].ovf = cs.overflow;
				r[sel].children = el.children.length;
			} else { r[sel] = null; }
		});
		window.__probeResult = JSON.stringify(r);
	`)
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
	f, _ := os.Create(out)
	_ = png.Encode(f, img)
	f.Close()
	fmt.Println("saved", out)

	// 解析 probe 结果
	res, _ := wv.JSInterpreter().RunJS(`window.__probeResult`)
	fmt.Println("PROBE_RESULT:", res)
	// 直接重新查询（确保布局后）
	res2, _ := wv.JSInterpreter().RunJS(`JSON.stringify({c: (function(){var e=document.querySelector('.comp-bar');var b=e.getBoundingClientRect();return {x:b.x,y:b.y,w:b.width,h:b.height};})(), x:(function(){var e=document.querySelector('.ctx-bar');var b=e.getBoundingClientRect();return {x:b.x,y:b.y,w:b.width,h:b.height};})()})`)
	fmt.Println("RECTS:", res2)

	// dump comp-bar 区域
	_ = rendering.RenderBlockFlow{}
	_ = dom.Element{}
}

func setupLoaders(wv *webkit.WebView, absDist string) {
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
