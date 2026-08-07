// Command modal_scroll_probe 打开设置面板，查询 settings-body 的滚动几何
// 与 overflow 状态，定位「内容显示不全+不能滚动」。
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

func js(wv *webkit.WebView, code string) string {
	iv, err := wv.JSInterpreter().RunJS(code)
	if err != nil {
		return "[ERR] " + err.Error()
	}
	return iv.ToString()
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
	for i := 0; i < 4; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
		time.Sleep(600 * time.Millisecond)
		runJobs(wv)
	}

	// 打开设置面板
	fmt.Println("[open settings] " + js(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		if (!btn) return 'no btn';
		var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev);
		return 'clicked';
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	runJobs(wv)

	// 查询滚动几何
	fmt.Println("[geom] " + js(wv, `(function(){
		var sel = document.querySelector('.settings-body');
		var modal = document.querySelector('.modal-content');
		var body = document.querySelector('.modal-body');
		if (!sel) return 'no .settings-body';
		function g(el) {
			if (!el) return null;
			var r = el.getBoundingClientRect();
			var cs = getComputedStyle(el);
			return {cls: el.className, x: r.x, y: r.y, w: r.width, h: r.height,
				scrollH: el.scrollHeight, clientH: el.clientHeight,
				overflow: cs.overflow, overflowY: cs.overflowY, display: cs.display,
				flex: cs.flex, pos: cs.position, maxH: cs.maxHeight};
		}
		return JSON.stringify({
			modal: g(modal), body: g(body), settingsBody: g(sel),
			children: Array.from(sel.children).slice(0, 3).map(function(c){ return {cls: c.className, tag: c.tagName}; })
		});
	})()`))

	// 验证滚动：设置 scrollTop 后检查是否真正滚动
	fmt.Println("[scroll0] " + js(wv, `(function(){
		var sel = document.querySelector('.settings-body');
		sel.scrollTop = 100;
		return 'scrollTop set: ' + sel.scrollTop;
	})()`))
	runJobs(wv)
	fmt.Println("[scroll1] " + js(wv, `(function(){
		var sel = document.querySelector('.settings-body');
		return JSON.stringify({scrollTop: sel.scrollTop, lastLabelY: (function(){
			var labs = document.querySelectorAll('.settings-body label');
			var out = [];
			for (var i = 0; i < labs.length && i < 12; i++) {
				var r = labs[i].getBoundingClientRect();
				out.push(Math.round(r.y));
			}
			return out;
		})()});
	})()`))
	// 滚回顶部
	fmt.Println("[scroll2] " + js(wv, `(function(){
		var sel = document.querySelector('.settings-body');
		sel.scrollTop = 0;
		return 'reset: ' + sel.scrollTop;
	})()`))
	runJobs(wv)

	// 打开工具配置弹窗（先关闭设置面板）
	fmt.Println("[toolcfg] " + js(wv, `(function(){
		var st = window.__state;
		if (st && st.settingsOpen !== undefined) { st.settingsOpen = false; }
		var close = document.querySelector('.modal-close');
		if (close) { var ev = new Event('click', {bubbles:true}); close.dispatchEvent(ev); }
		if (st && !st.rightPanelVisible) { st.rightPanelVisible = true; }
		return 'closed';
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	fmt.Println("[toolcfg] " + js(wv, `(function(){
		var btn = document.querySelector('.obtn-review-config');
		if (!btn) return 'no btn';
		var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev);
		return 'clicked';
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	fmt.Println("[toolcfg-geom] " + js(wv, `(function(){
		var modal = document.querySelector('.modal-content');
		var body = document.querySelector('.modal-body');
		var list = document.querySelector('.tool-list, .toolcfg-list, [class*=tool-list]');
		var sb = document.querySelector('.modal-body > div:nth-child(2)');
		function g(el) {
			if (!el) return null;
			var r = el.getBoundingClientRect();
			var cs = getComputedStyle(el);
			return {cls: el.className, x: r.x, y: r.y, w: r.width, h: r.height,
				scrollH: el.scrollHeight, clientH: el.clientHeight, overflowY: cs.overflowY,
				flex: cs.flex, pos: cs.position, maxH: cs.maxHeight};
		}
		return JSON.stringify({modal: g(modal), body: g(body), scrollable: g(sb), list: g(list)});
	})()`))
}
