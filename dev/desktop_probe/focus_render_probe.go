// Command focus_render_probe: 验证引擎对「明确 style.width 的 inline-block
// span」（xterm DOM renderer 空格 span）的布局是否保持 JS 设定宽度，以及
// 聚焦/reflow 前后是否一致——排查「终端聚焦/失焦空格间距不一致」。
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

func runJS(wv *webkit.WebView, js string) string {
	if interp := wv.JSInterpreter(); interp != nil {
		if v, err := interp.RunJS(js); err == nil {
			return v.ToString()
		} else {
			return "ERR: " + err.Error()
		}
	}
	return "no-interp"
}

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	htmlPath := filepath.Join(wd, "cmd", "companion", "web-ui", "ide_ref_focus.html")
	htmlData, err := os.ReadFile(htmlPath)
	if err != nil {
		log.Fatalf("read html: %v", err)
	}
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style}, text)
	}
	layout.FontMetricsFunc = func(family string, size float64, weight int, style string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	wv := webkit.NewWebView()
	wv.Resize(400, 80)
	if err := wv.LoadHTML(string(htmlData)); err != nil {
		log.Printf("LoadHTML err: %v", err)
	}
	if out := wv.ConsoleOutput(); out != "" {
		log.Printf("[CONSOLE] %s", out)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	fmt.Println("── ① 首次布局（span style.width 明确） ──")
	fmt.Println(runJS(wv, "window.__dump()"))

	fmt.Println("── ② 模拟 blur：重建行 0 span（xterm refresh 重建） ──")
	fmt.Println(runJS(wv, "window.__rebuildRow(0)"))

	fmt.Println("── ③ 强制 RebuildRenderTree + EnsureLayout（聚焦类 reflow） ──")
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println(runJS(wv, "window.__dump()"))

	fmt.Println("── ④ 再触发一次 rebuild + 查 engine 渲染树 span 宽度 ──")
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println(runJS(wv, "window.__dump()"))
}
