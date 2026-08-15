// Command char_measure_probe loads the terminal-fit reference page through
// wb-ui and checks CHAR MEASURE STABILITY: xterm's CharSizeService.measure()
// result at different phases (initial open+fit → forced re-measure →
// resize same size → focus). User reports: initial state differs from
// focused state (space spacing inconsistent), and initial is wider than
// the browser.
//
// Key question: does wb-ui return a STABLE char width across re-measures?
// xterm re-measures on every resize (same-size resize included) and after
// focus-related refresh. If the engine's span-based canvas.measureText or
// offsetWidth returns different values depending on layout timing, the
// terminal width/space spacing will change between initial and focused.
//
// Output: [MEASURE] diagnostics lines
// Run: go run ./dev/desktop_probe/char_measure_probe.go
//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

func setupTermLoaders(wv *webkit.WebView, webDir string) {
	absDir, _ := filepath.Abs(webDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDir, clean))
				if err != nil {
					return "", err
				}
				code := string(data)
				if i := strings.Index(code, "//# sourceMappingURL="); i >= 0 {
					code = code[:i]
				}
				if strings.Contains(clean, "xterm") {
					log.Printf("[loader] %s read=%d bytes load=%v", clean, len(data), time.Since(time.Now()))
				}
				return code, nil
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDir, clean))
				return string(data), err
			}
		}
	}
}

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	webDir := filepath.Join(wd, "cmd", "companion", "web-ui")
	htmlData, err := os.ReadFile(filepath.Join(webDir, "ide_ref_termfit.html"))
	if err != nil {
		log.Fatalf("read ide_ref_termfit.html: %v", err)
	}
	log.Printf("[char_measure_probe] webDir=%s html=%d bytes", webDir, len(htmlData))

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
	setupTermLoaders(wv, webDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()

	wv.LoadHTML(string(htmlData))
	_, _ = wv.JSInterpreter().RunJS(`
		if (typeof window.onload === 'function') { window.onload(); }
	`)
	interp := wv.JSInterpreter()
	if el := interp.GetEventLoop(); el != nil {
		for i := 0; i < 12; i++ {
			el.ProcessTasks(0)
			_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 200); })`)
			el.ProcessTasks(0)
		}
	}
	if out := wv.ConsoleOutput(); out != "" {
		fmt.Println("[CONSOLE]")
		fmt.Println(out)
	}

	snapshot := func(label string) string {
		js, err := interp.RunJS(`(function(){
			var t = window.__term;
			if (!t) return JSON.stringify({err:'no __term'});
			var core = t._core || t;
			var cs = core._charSizeService;
			var d = (core._renderService && core._renderService.dimensions) || null;
			return JSON.stringify({
				cols: t.cols, rows: t.rows,
				charW: cs ? cs.width : null,
				charH: cs ? cs.height : null,
				hasValid: cs ? cs.hasValidSize : null,
				cssCellW: d && d.css ? d.css.cell.width : null,
				cssCellH: d && d.css ? d.css.cell.height : null
			});
		})()`)
		if err != nil {
			return fmt.Sprintf("err: %v", err)
		}
		return js.ToString()
	}

	// 阶段 1：初始（open + fit 后）
	fmt.Println("[MEASURE] initial: " + snapshot("initial"))

	// 阶段 2：强制重测（模拟 _afterResize 的 measure）
	if _, err := interp.RunJS(`window.__term._core._charSizeService.measure()`); err != nil {
		fmt.Println("[MEASURE] remeasure1 err:", err)
	}
	fmt.Println("[MEASURE] after remeasure1: " + snapshot("rem1"))

	// 阶段 3：再重测一次
	if _, err := interp.RunJS(`window.__term._core._charSizeService.measure()`); err != nil {
		fmt.Println("[MEASURE] remeasure2 err:", err)
	}
	fmt.Println("[MEASURE] after remeasure2: " + snapshot("rem2"))

	// 阶段 4：resize 相同尺寸（触发 _afterResize → measure）
	if _, err := interp.RunJS(`window.__term.resize(window.__term.cols, window.__term.rows)`); err != nil {
		fmt.Println("[MEASURE] resize err:", err)
	}
	fmt.Println("[MEASURE] after resize-same: " + snapshot("resize"))

	// 阶段 5：focus（可能触发 refresh/resize 链）
	if _, err := interp.RunJS(`window.__term.focus()`); err != nil {
		fmt.Println("[MEASURE] focus err:", err)
	}
	fmt.Println("[MEASURE] after focus: " + snapshot("focus"))

	// 阶段 6：直接测 canvas.measureText 的稳定性（xterm 优先路径）
	if _, err := interp.RunJS(`
		var m1 = window.__term._core._charSizeService._widthCache ? 
			window.__term._core._charSizeService._widthCache.get('W', false, false) : -1;
		var m2 = window.__term._core._charSizeService._widthCache ?
			window.__term._core._charSizeService._widthCache.get('W', false, false) : -1;
		window.__m1 = m1; window.__m2 = m2;
	`); err != nil {
		fmt.Println("[MEASURE] widthCache err:", err)
	}
	if v, err := interp.RunJS(`JSON.stringify({m1: window.__m1, m2: window.__m2})`); err == nil {
		fmt.Println("[MEASURE] widthCache W: " + v.ToString())
	}

	// 阶段 7：DOM offsetWidth 路径（xterm fallback：32 字符 span）
	if _, err := interp.RunJS(`
		var d = document.createElement('div');
		d.style.cssText = 'position:absolute;visibility:hidden;left:0;top:0;';
		var s = document.createElement('span');
		s.style.cssText = 'font-size:13px;font-family:Consolas,monospace;line-height:normal;white-space:pre;display:inline-block;';
		s.textContent = 'WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW';
		d.appendChild(s); document.body.appendChild(d);
		window.__ow1 = s.offsetWidth;
		s.textContent = 'WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW';
		window.__ow2 = s.offsetWidth;
		document.body.removeChild(d);
	`); err != nil {
		fmt.Println("[MEASURE] offsetWidth err:", err)
	}
	if v, err := interp.RunJS(`JSON.stringify({ow1: window.__ow1, ow2: window.__ow2})`); err == nil {
		fmt.Println("[MEASURE] offsetWidth 32W: " + v.ToString())
	}
}
