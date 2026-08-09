// Command term_scroll_probe loads the terminal reference page, writes extra
// scrollback lines, then checks viewport scrollTop/scrollHeight and performs
// a scroll to verify xterm scrolling works (browser parity).
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	webDir := filepath.Join(wd, "cmd", "companion", "web-ui")
	htmlData, err := os.ReadFile(filepath.Join(webDir, "ide_ref_term.html"))
	if err != nil {
		log.Fatalf("read: %v", err)
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
	setupTermLoaders(wv, webDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	wv.LoadHTML(string(htmlData))
	_, _ = wv.JSInterpreter().RunJS(`if (typeof window.onload === 'function') window.onload();`)
	interp := wv.JSInterpreter()
	if el := interp.GetEventLoop(); el != nil {
		for i := 0; i < 10; i++ {
			el.ProcessTasks(0)
			_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 200); })`)
			el.ProcessTasks(0)
		}
	}
	// 追加滚动内容 + 检查滚动能力
	js := `
	var host = document.getElementById('term-host');
	var vp = host.querySelector('.xterm-viewport');
	var out = [];
	out.push('viewport=' + (vp ? 'yes' : 'no'));
	if (vp) {
		out.push('vp scrollTop=' + vp.scrollTop + ' scrollH=' + vp.scrollHeight + ' clientH=' + vp.clientHeight);
		out.push('vp overflowY=' + getComputedStyle(vp).overflowY);
		out.push('vp overflow=' + getComputedStyle(vp).overflow);
	}
	var term = window.__term;
	out.push('term=' + (term ? 'yes' : 'no'));
	if (term) {
		out.push('rows=' + term.rows + ' cols=' + term.cols + ' bufferLen=' + term.buffer.length);
	}
	var sa = host.querySelector('.xterm-scroll-area');
	if (sa) {
		out.push('scrollArea style.h=' + sa.style.height + ' offsetH=' + sa.offsetHeight + ' offsetW=' + sa.offsetWidth);
	}
	var se = host.querySelector('.xterm-scrollable-element');
	if (se) {
		out.push('scrollableEl h=' + se.offsetHeight + ' style.h=' + se.style.height);
	}
	// 直接读 viewport 的布局 overflow
	var vp2 = host.querySelector('.xterm-viewport');
	if (vp2) {
		var cs2 = getComputedStyle(vp2);
		out.push('cs overflowY via getProp=' + cs2.getPropertyValue('overflow-y') + ' direct=' + cs2.overflowY + ' kebab=' + cs2['overflow-y']);
	}
	console.log('[SCROLL] ' + out.join(' | '));
	`
	_, _ = wv.JSInterpreter().RunJS(js)
	if el := interp.GetEventLoop(); el != nil {
		el.ProcessTasks(0)
	}
	if out := wv.ConsoleOutput(); out != "" {
		fmt.Println(out)
	}
	_ = strings.TrimSpace
}

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
