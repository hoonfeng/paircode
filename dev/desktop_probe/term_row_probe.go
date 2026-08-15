// Command term_row_probe 加载真实 dist，等待终端创建（xterm DOM renderer），
// 查询首行文本位置/行高/行间距，输出 PNG 截图与 JSON 诊断，
// 用于对照浏览器（Edge 参照）的终端行渲染。
//go:build ignore

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func runJobs(wv *webkit.WebView) {
	el := wv.JSInterpreter().GetEventLoop()
	for i := 0; i < 15; i++ {
		if el != nil {
			el.ProcessTasks(20) // 驱动 setTimeout/RAF（模拟 desktop 主循环）
		}
		wv.JSInterpreter().RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func js(wv *webkit.WebView, code string) string {
	iv, err := wv.JSInterpreter().RunJS(code)
	if err != nil {
		return "[ERR] " + err.Error()
	}
	return iv.ToString()
}

func shot(wv *webkit.WebView, name string) {
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	pngBytes, err := wv.Render()
	if err != nil {
		log.Printf("Render %s: %v", name, err)
		return
	}
	w, h := wv.Width(), wv.Height()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := (y*w + x) * 4
			if off+3 < len(pngBytes) {
				img.SetRGBA(x, y, color.RGBA{R: pngBytes[off], G: pngBytes[off+1], B: pngBytes[off+2], A: pngBytes[off+3]})
			}
		}
	}
	wd, _ := os.Getwd()
	out := filepath.Join(wd, "dev", "desktop_probe", name)
	f, err := os.Create(out)
	if err != nil {
		log.Printf("create %s: %v", out, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Printf("encode %s: %v", out, err)
		return
	}
	log.Printf("shot %dx%d → %s", w, h, out)
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
	layout.FontMetricsFunc = func(family string, size float64, weight int, style string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	wv := webkit.NewWebView()
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
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
				return string(data), err
			}
		}
	}
	wv.SetConsoleLogger(&jsc.BufferLogger{})
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	for i := 0; i < 6; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
		time.Sleep(600 * time.Millisecond)
		runJobs(wv)
	}

	// 查询终端状态
	fmt.Println("[term] " + js(wv, `(function(){
		var wrap = document.querySelector('.term-xterm-wrap');
		if (!wrap) return JSON.stringify({wrap:false});
		var xterm = document.querySelector('.xterm');
		var rowsEl = document.querySelector('.xterm-rows');
		var term = window.__lastTerm;
		var out = {wrap:true, xterm:!!xterm, rowsEl:!!rowsEl, term:!!term};
		if (xterm) {
			var r = xterm.getBoundingClientRect();
			out.xtermRect = {x:r.x, y:r.y, w:r.width, h:r.height};
			var cs = window.getComputedStyle(xterm);
			out.xtermPad = {t:cs.paddingTop, b:cs.paddingBottom, l:cs.paddingLeft, r:cs.paddingRight};
			out.xtermLH = cs.lineHeight;
			out.xtermFS = cs.fontSize;
		}
		if (term) {
			out.cols = term.cols; out.rows = term.rows;
			var dims = term._core && term._core._renderService && term._core._renderService.dimensions;
			out.cell = dims && dims.css ? {w:dims.css.cell.width, h:dims.css.cell.height} : null;
		}
		if (rowsEl) {
			var rows = rowsEl.children;
			out.rowCount = rows.length;
			var arr = [];
			for (var i = 0; i < rows.length && i < 6; i++) {
				var rr = rows[i].getBoundingClientRect();
				arr.push({i:i, y:rr.y, h:rr.height, cls:rows[i].className, lh:window.getComputedStyle(rows[i]).lineHeight});
			}
			out.rows = arr;
			// 首行文本
			var firstText = '';
			if (rows.length > 0) {
				var t = rows[0].textContent || '';
				firstText = t.slice(0, 40);
			}
			out.firstText = firstText;
		}
		return JSON.stringify(out);
	})()`))

	// 写几行文本验证行距
	fmt.Println("[term] " + js(wv, `(function(){
		var term = window.__lastTerm;
		if (!term) return 'no-term';
		term.write('LINE1\r\nLINE2\r\nLINE3\r\nLINE4\r\n');
		return 'written';
	})()`))
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	fmt.Println("[term2] " + js(wv, `(function(){
		var rowsEl = document.querySelector('.xterm-rows');
		if (!rowsEl) return 'no-rows';
		var rows = rowsEl.children;
		var arr = [];
		for (var i = 0; i < rows.length && i < 6; i++) {
			var rr = rows[i].getBoundingClientRect();
			var cs = window.getComputedStyle(rows[i]);
			arr.push({i:i, y:Math.round(rr.y*100)/100, h:Math.round(rr.height*100)/100, lh:cs.lineHeight, fs:cs.fontSize});
		}
		return JSON.stringify(arr);
	})()`))

	shot(wv, "term_row_shot.png")
	log.Println("done")
}
