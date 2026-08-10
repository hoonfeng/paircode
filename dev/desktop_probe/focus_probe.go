// Command focus_probe: 在真实 dist（完整 Vue 应用 + xterm）中写入带空格
// 文本，对比 初始 / blur() / focus() 三个状态下每行 span 的宽度与行几何
// ——排查「终端聚焦/失焦空格间距不一致」。
package main

import (
	"fmt"
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
	for i := 0; i < 8; i++ {
		if el != nil {
			el.ProcessTasks(20)
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

const dumpSpansJS = `(function(){
  var term = window.__lastTerm;
  if (!term) return 'no-term';
  var rowsEl = document.querySelector('.xterm-rows');
  if (!rowsEl) return 'no-rows';
  var rows = rowsEl.children;
  var out = [];
  for (var i = 0; i < rows.length && i < 6; i++) {
    var r = rows[i].getBoundingClientRect();
    var spans = [];
    var kids = rows[i].children;
    for (var j = 0; j < (kids ? kids.length : 0) && j < 40; j++) {
      var k = kids[j];
      var kr = k.getBoundingClientRect();
      spans.push({t: k.textContent, w: Math.round(kr.width*1000)/1000, x: Math.round(kr.x*1000)/1000,
                  css: getComputedStyle(k).width, cls: k.className || ''});
    }
    out.push({i:i, y:Math.round(r.y*10)/10, h:Math.round(r.height*1000)/1000, text: rows[i].textContent, spans: spans});
  }
  var dims = term._core && term._core._renderService && term._core._renderService.dimensions;
  return JSON.stringify({cellW: dims ? dims.css.cell.width : null, cellH: dims ? dims.css.cell.height : null,
                         focus: term.element ? term.element.classList.contains('focus') : null, rows: out});
})()`

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
	for i := 0; i < 8; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
		time.Sleep(600 * time.Millisecond)
		runJobs(wv)
	}

	// ★ 初始加载的信息（banner/prompt）刚渲染完——记录 cellW 与行 span
	fmt.Println("═══════ ⓪ 初始加载的信息（刚渲染完） ═══════")
	fmt.Println(js(wv, dumpSpansJS))

	// 写带空格文本
	fmt.Println("[write] " + js(wv, `(function(){
		var term = window.__lastTerm;
		if (!term) return 'no-term';
		term.write('ABC DEF  GHI   JKL    MNO\r\n');
		term.write('A B C D E F G H I J\r\n');
		term.write('  leading spaces  \r\n');
		term.write('12345678901234567890\r\n');
		return 'written';
	})()`))
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)

	fmt.Println("═══════ ① 初始（未聚焦） ═══════")
	fmt.Println(js(wv, dumpSpansJS))

	// 聚焦
	fmt.Println("[do-focus] " + js(wv, `(function(){
		var term = window.__lastTerm;
		if (!term) return 'no-term';
		try { if (term.focus) term.focus(); } catch(e){ return 'focus-err ' + e; }
		return 'focused';
	})()`))
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)

	fmt.Println("═══════ ② 聚焦后 ═══════")
	fmt.Println(js(wv, dumpSpansJS))

	// 失焦
	fmt.Println("[do-blur] " + js(wv, `(function(){
		var term = window.__lastTerm;
		if (!term) return 'no-term';
		try { if (term.blur) term.blur(); } catch(e){ return 'blur-err ' + e; }
		return 'blurred';
	})()`))
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)

	fmt.Println("═══════ ③ 失焦后 ═══════")
	fmt.Println(js(wv, dumpSpansJS))

	// ── 多终端 cellW 一致性：点击新建标签创建第二个终端，对比 cellW ──
	fmt.Println("[multi] " + js(wv, `(function(){
		var term1 = window.__lastTerm;
		var dims1 = term1._core && term1._core._renderService && term1._core._renderService.dimensions;
		var w1 = dims1 ? dims1.css.cell.width : null;
		// 点击「新建终端」按钮
		var btns = document.querySelectorAll('.term-tab.new-tab');
		if (!btns.length) return 'no-newtab-btn';
		btns[0].click();
		return 'clicked w1=' + w1;
	})()`))
	time.Sleep(1500 * time.Millisecond)
	runJobs(wv)
	fmt.Println("[multi2] " + js(wv, `(function(){
		var term2 = window.__lastTerm;
		if (!term2) return 'no-term2';
		var dims2 = term2._core && term2._core._renderService && term2._core._renderService.dimensions;
		var w2 = dims2 ? dims2.css.cell.width : null;
		var h2 = dims2 ? dims2.css.cell.height : null;
		var tabs = document.querySelectorAll('.term-tab');
		var tabTexts = [];
		for (var i = 0; i < tabs.length; i++) tabTexts.push((tabs[i].textContent || '').trim().slice(0, 10));
		return JSON.stringify({term2_cellW: w2, term2_cellH: h2, tabCount: tabs.length, tabs: tabTexts});
	})()`))
	// 切回第一个终端，对比 cellW 是否变化
	fmt.Println("[multi3] " + js(wv, `(function(){
		var tabs = document.querySelectorAll('.term-tab');
		for (var i = 0; i < tabs.length; i++) {
			if ((tabs[i].textContent || '').indexOf('终端 1') >= 0) { tabs[i].click(); break; }
		}
		return 'switched-back';
	})()`))
	time.Sleep(800 * time.Millisecond)
	runJobs(wv)
	fmt.Println("[multi4] " + js(wv, `(function(){
		var term = window.__lastTerm;
		var dims = term._core && term._core._renderService && term._core._renderService.dimensions;
		return JSON.stringify({cellW: dims ? dims.css.cell.width : null, cellH: dims ? dims.css.cell.height : null,
		                       active: term.buffer.active === term.buffer.normal});
	})()`))

	log.Println("done")
}
