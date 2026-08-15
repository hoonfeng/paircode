// Command term_scroll_probe2 verifies xterm scrolling end-to-end in wb-ui:
// 1. open terminal, write many lines (scrollback > viewport)
// 2. dispatch a wheel event on the terminal DOM
// 3. check the scrollable-element's scrollTop / slider position moved
//go:build ignore

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
	// 写更多行（20 行 scrollback）
	_, _ = wv.JSInterpreter().RunJS(`
		var t = window.__term;
		if (t) { for (var i=0;i<25;i++) t.write('line ' + i + '\r\n'); }
	`)
	if el := interp.GetEventLoop(); el != nil {
		el.ProcessTasks(0)
		_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
		el.ProcessTasks(0)
	}
	// 先测 scrollLines API 能否滚动（排除事件链路，验证 Scrollable 本身）
	_, _ = wv.JSInterpreter().RunJS(`
		var t = window.__term;
		if (t) {
			var b = t.buffer;
			var ab = b ? (b.active || b._normal || b._store) : null;
			console.log('[API] before scrollLines buf rows=' + (ab ? ab.length : 'na'));
			t.scrollLines(3);
			console.log('[API] after scrollLines(3)');
		}
	`)
	if el := interp.GetEventLoop(); el != nil {
		for i := 0; i < 6; i++ {
			el.ProcessTasks(0)
			_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 100); })`)
			el.ProcessTasks(0)
		}
	}
	// 检查 xterm Viewport/ScrollableElement 状态
	_, _ = wv.JSInterpreter().RunJS(`
		var t = window.__term;
		var host = document.getElementById('term-host');
		var se = host.querySelector('.xterm-scrollable-element');
		if (t && t._core) {
			var vp = t._core._viewport;
			console.log('[DIAG] viewport=' + (vp ? 'yes' : 'no'));
			if (vp) {
				var se2 = vp._scrollableElement;
				console.log('[DIAG] scrollableElement=' + (se2 ? 'yes' : 'no'));
			if (se2) {
				console.log('[DIAG] se2 domNode class=' + (se2.getDomNode ? se2.getDomNode().className : 'na'));
				var dims = se2.getScrollDimensions ? se2.getScrollDimensions() : null;
				if (dims) console.log('[DIAG] dims height=' + dims.height + ' scrollHeight=' + dims.scrollHeight);
				var pos = se2.getScrollPosition ? se2.getScrollPosition() : null;
				if (pos) console.log('[DIAG] pos scrollTop=' + pos.scrollTop);
				console.log('[DIAG] mouseWheelToDispose len=' + (se2._mouseWheelToDispose ? se2._mouseWheelToDispose.length : 'na') +
					' handleMouseWheel=' + (se2._options ? se2._options.handleMouseWheel : 'na'));
			}
			}
			// domNode 与 querySelector 结果是否同一元素
			if (se2 && se2.getDomNode) {
				var dn = se2.getDomNode();
				console.log('[DIAG] same element=' + (dn === se) + ' seClass=' + (se ? se.className : 'na'));
			}
		}
	`)
	if el := interp.GetEventLoop(); el != nil {
		for i := 0; i < 4; i++ {
			el.ProcessTasks(0)
			_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 50); })`)
			el.ProcessTasks(0)
		}
	}
	// 派发 wheel 事件到 scrollable-element，检查 xterm 滚动是否响应
	_, _ = wv.JSInterpreter().RunJS(`
		var host = document.getElementById('term-host');
		var se = host.querySelector('.xterm-scrollable-element');
		if (se) {
			var t = window.__term;
			var vp = t && t._core ? t._core._viewport : null;
			// 先回到顶部
			if (vp) vp.scrollToLine(0);
			var posBefore = vp && vp._scrollableElement ? vp._scrollableElement.getScrollPosition().scrollTop : -1;
			// 手动捕获 xterm 的 wheel 处理结果
			window.__wheelErr = null;
			window.__origOnMouseWheel = vp && vp._scrollableElement ? vp._scrollableElement._onMouseWheel : null;
			if (vp && vp._scrollableElement && vp._scrollableElement._onMouseWheel) {
				var se0 = vp._scrollableElement;
				se0._onMouseWheel = function(e){
					try { window.__origOnMouseWheel.call(se0, e); window.__wheelErr = 'ok'; }
					catch(err) { window.__wheelErr = 'ERR:' + (err && err.stack || err); }
				};
			}
			// 派发真实 wheel（向下滚 = deltaY 正）
			var we = new WheelEvent('wheel', {deltaY: 100, deltaMode: 0, bubbles: true, cancelable: true});
			var ret = se.dispatchEvent(we);
			var posAfter = vp && vp._scrollableElement ? vp._scrollableElement.getScrollPosition().scrollTop : -1;
			console.log('[WHEEL] posBefore=' + posBefore + ' posAfter=' + posAfter + ' ret=' + ret + ' moved=' + (posBefore !== posAfter) + ' err=' + window.__wheelErr);
		}
	`)
	if el := interp.GetEventLoop(); el != nil {
		for i := 0; i < 4; i++ {
			el.ProcessTasks(0)
			_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 50); })`)
			el.ProcessTasks(0)
		}
	}
	// 检查滚动状态（xterm 6 API：term.buffer）
	_, _ = wv.JSInterpreter().RunJS(`
		var t3 = window.__term;
		if (t3) {
			var b = t3.buffer || t3.buffers;
			var keys = [];
			for (var k in b) keys.push(k);
			console.log('[WHEEL2] buffer keys=' + keys.join(','));
			var ab = b && b.active ? b.active : null;
			console.log('[WHEEL2] ydisp=' + (ab ? ab.ydisp : 'na') + ' ybase=' + (ab ? ab.ybase : 'na'));
		}
		var host2 = document.getElementById('term-host');
		var se2 = host2.querySelector('.xterm-scrollable-element');
		if (se2) {
			var r3 = se2.getBoundingClientRect();
			console.log('[WHEEL2] scrollableEl rect y=' + r3.top + ' h=' + r3.height);
		}
		var sl2 = host2.querySelector('.xterm-scrollable-element .slider');
		if (sl2) {
			var r2 = sl2.getBoundingClientRect();
			console.log('[WHEEL2] slider rect y=' + r2.top + ' h=' + r2.height);
		}
		var vp3 = host2.querySelector('.xterm-viewport');
		if (vp3) {
			console.log('[WHEEL2] vp scrollTop=' + vp3.scrollTop + ' scrollH=' + vp3.scrollHeight + ' clientH=' + vp3.clientHeight);
		}
	`)
	if el := interp.GetEventLoop(); el != nil {
		for i := 0; i < 6; i++ {
			el.ProcessTasks(0)
			_, _ = interp.RunJS(`new Promise(function(res){ setTimeout(res, 100); })`)
			el.ProcessTasks(0)
		}
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
