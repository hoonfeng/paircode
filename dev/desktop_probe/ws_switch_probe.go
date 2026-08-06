// Command ws_switch_probe 量化「工作区切换」在桌面版(wb-ui)的耗时构成：
//   1. panel-only=false 加载真实 dist 完整 IDE 布局 + desktopbridge（真实 handler）
//   2. 点击第二个 .ws-item（UI 真实 click → Vue @click → switchToWorkspace）
//   3. 分段计时：API 层（POST /workspace + state 更新）→ conversations 加载 → 渲染层
//   （RebuildRenderTree + EnsureLayout，等价 host 一次全量渲染）
//   对照：浏览器端同类 API 实测 ~1-17ms/次，若 probe 中 API 层也快而渲染层慢
//   → 桌面版慢在 wb-ui 引擎渲染，与后端无关。
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

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func setupLoaders2(wv *webkit.WebView, distDir string) {
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
				return string(data), nil
			}
		}
	}
}

func waitJS2(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func runJobs2(wv *webkit.WebView) {
	for i := 0; i < 8; i++ {
		wv.JSInterpreter().RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func runJS2(wv *webkit.WebView, script string) string {
	v, err := wv.JSInterpreter().RunJS(script)
	if err != nil {
		return "[err] " + err.Error()
	}
	return v.ToString()
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
	layout.FontMetricsFunc = func(family string, size float64, weight int, style2 string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	wv := webkit.NewWebView()
	setupLoaders2(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	// 完整 IDE 布局：不设 __DESKTOP_PANEL_MODE__
	wv.LoadHTML(string(htmlData))
	waitJS2(wv, 600)
	waitJS2(wv, 600)
	runJobs2(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// ── 当前工作区状态 ──
	fmt.Printf("[wsswitch] diag=%s\n", runJS2(wv, `(function(){
		return JSON.stringify({
			loc: location.href,
			origin: location.origin,
			fetchIsNative: (window.fetch && window.fetch.toString().indexOf('native code') >= 0),
			fetchStr: String(window.fetch).slice(0, 60),
			hasBridgeCall: typeof go !== 'undefined' && typeof go.bridge_call === 'function'
		});
	})()`))
	fmt.Printf("[wsswitch] root=%s\n", runJS2(wv, `(function(){ var st = window.__state; return st ? st.workspaceRoot : 'no-state'; })()`))
	// ── 直接测 go.bridge_call 与 fetch（单请求计时，确认拦截路径） ──
	fmt.Printf("[wsswitch] bridgeDirect=%s\n", runJS2(wv, `(function(){
		try {
			var t0 = Date.now();
			var r = go.bridge_call('GET', '/api/workspace', '', '{}');
			var dt = Date.now() - t0;
			return JSON.stringify({dt: dt, r: String(r).slice(0, 80)});
		} catch(e) { return 'err ' + e.message; }
	})()`))
	fmt.Printf("[wsswitch] fetchDirect=%s\n", runJS2(wv, `(function(){
		window.__fd = 'pending';
		return fetch('/api/workspace').then(function(res){
			var t0 = Date.now();
			return res.json().then(function(j){
				window.__fd = JSON.stringify({dt: Date.now()-t0, root: j.root});
				return window.__fd;
			});
		}).catch(function(e){ window.__fd = 'fetch err ' + e.message; return window.__fd; });
	})()`))
	waitJS2(wv, 300)
	fmt.Printf("[wsswitch] fetchDirectRes=%s\n", runJS2(wv, `window.__fd || 'pending'`))
	fmt.Printf("[wsswitch] wsItems=%s\n", runJS2(wv, `(function(){
		var items = document.querySelectorAll('.ws-item');
		var out = [];
		for (var i = 0; i < items.length; i++) out.push((i) + ':' + items[i].querySelector('.ws-name').textContent.trim());
		return JSON.stringify(out);
	})()`))
	fmt.Printf("[wsswitch] wsRects=%s\n", runJS2(wv, `(function(){
		var items = document.querySelectorAll('.ws-item');
		var out = [];
		for (var i = 0; i < items.length; i++) {
			var r = items[i].getBoundingClientRect();
			out.push(i + ':{x=' + Math.round(r.left) + ',y=' + Math.round(r.top) + ',w=' + Math.round(r.width) + ',h=' + Math.round(r.height) + '}');
		}
		return JSON.stringify(out);
	})()`))
	// conversations 数量（切换前）
	fmt.Printf("[wsswitch] convsBefore=%s\n", runJS2(wv, `(function(){ var st = window.__state; return st ? (st.conversations||[]).length : -1; })()`))

	// ── 目标：点击 wb-ui 那个 ws-item（真正不同的工作区；ws-list 有重复项，按名字找） ──
	target := runJS2(wv, `(function(){
		var items = document.querySelectorAll('.ws-item');
		for (var i = 0; i < items.length; i++) {
			var nm = items[i].querySelector('.ws-name').textContent.trim();
			if (nm === 'wb-ui') return JSON.stringify({idx: i, name: nm});
		}
		return 'none';
	})()`)
	fmt.Printf("[wsswitch] 目标 ws-item = %s\n", target)
	var tIdx int
	var tName string
	if strings.HasPrefix(target, "{") {
		fmt.Sscanf(target, `{"idx":%d`, &tIdx)
		fmt.Sscanf(target, `{"idx":%d,"name":"%s`, &tIdx, &tName)
		tName = strings.TrimSuffix(tName, `"`)
	} else {
		fmt.Printf("[wsswitch] 未找到 wb-ui ws-item，退出\n")
		return
	}

	// ── 点击前：hook fetch 记录请求耗时分布 ──
	runJS2(wv, `(function(){
		window.__reqLog = [];
		var _f = window.fetch;
		window.fetch = function(url, options) {
			var t0 = performance.now();
			return _f(url, options).then(function(r){
				var dt = Math.round(performance.now() - t0);
				window.__reqLog.push({u: String(url).slice(0, 40), dt: dt});
				return r;
			}, function(e){
				window.__reqLog.push({u: String(url).slice(0, 40), dt: -1, err: String(e).slice(0, 30)});
				throw e;
			});
		};
		return 'hooked';
	})()`)
	runJS2(wv, `window.__wsT0 = Date.now(); window.__wsPhase = 'clicked'; window.__wsApiMs = -1; window.__wsConvMs = -1;`)
	clickRet := runJS2(wv, fmt.Sprintf(`(function(){
		var items = document.querySelectorAll('.ws-item');
		items[%d].dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
		return 'clicked';
	})()`, tIdx))
	fmt.Printf("[wsswitch] %s\n", clickRet)
	// ── 点击 RunJS 内完成整链（RunString 隐式 flush microtask）──
	wv.JSInterpreter().RunJobs()
	waitJS2(wv, 50)
	fmt.Printf("[wsswitch] reqLog=%s\n", runJS2(wv, `JSON.stringify(window.__reqLog || [])`))

	// ── 分段计时：轮询 state.workspaceRoot 变化（API 层） ──
	rootChanged := false
	var apiMs int64 = -1
	for i := 0; i < 100 && !rootChanged; i++ {
		waitJS2(wv, 50)
		res := runJS2(wv, `(function(){
			if (window.__state.workspaceRoot.indexOf('wb-ui') >= 0) {
				window.__wsApiMs = Date.now() - window.__wsT0;
				return 'changed';
			}
			return 'waiting';
		})()`)
		if res == "changed" {
			rootChanged = true
		}
	}
	if rootChanged {
		v := runJS2(wv, `window.__wsApiMs`)
		fmt.Sscanf(v, "%d", &apiMs)
	}
	fmt.Printf("[wsswitch] API层(root切换): %d ms\n", apiMs)

	// ── conversations 加载完成 ──
	convLoaded := false
	var convMs int64 = -1
	for i := 0; i < 100 && !convLoaded; i++ {
		waitJS2(wv, 50)
		res := runJS2(wv, `(function(){
			var st = window.__state;
			var list = st.conversations || [];
			// conversations 加载完且属于 wb-ui（首个 id 存在即可）
			if (list.length > 0 && st.workspaceRoot.indexOf('wb-ui') >= 0) {
				window.__wsConvMs = Date.now() - window.__wsT0;
				return 'loaded ' + list.length;
			}
			return 'waiting ' + list.length;
		})()`)
		if strings.HasPrefix(res, "loaded") {
			convLoaded = true
			fmt.Printf("[wsswitch] conversations=%s\n", res)
		}
	}
	if convLoaded {
		v := runJS2(wv, `window.__wsConvMs`)
		fmt.Sscanf(v, "%d", &convMs)
	}
	fmt.Printf("[wsswitch] conversations加载: %d ms\n", convMs)

	// ── 渲染层：切换后全量渲染（等价 host 一次渲染循环） ──
	renderStart := time.Now()
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	renderMs := time.Since(renderStart).Milliseconds()
	fmt.Printf("[wsswitch] 渲染层(Rebuild+EnsureLayout): %d ms\n", renderMs)

	// ── 画一帧（paint） ──
	_ = wv.JSInterpreter()
	paintStart := time.Now()
	_, _ = wv.JSInterpreter().RunJS(`1`)
	mf := wv.MainFrame()
	if mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.MarkRenderTreeDirty()
		}
	}
	_, _ = wv.Render()
	paintMs := time.Since(paintStart).Milliseconds()
	fmt.Printf("[wsswitch] paint: %d ms\n", paintMs)

	// ── 切回 gou-ide ──
	runJS2(wv, `(function(){
		var items = document.querySelectorAll('.ws-item');
		for (var i = 0; i < items.length; i++) {
			var nm = items[i].querySelector('.ws-name').textContent.trim();
			if (nm === 'gou-ide') { items[i].dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true})); return 'clicked gou-ide'; }
		}
		return 'no gou-ide';
	})()`)
	waitJS2(wv, 500)
	runJobs2(wv)
	fmt.Printf("[wsswitch] 切回后 root=%s\n", runJS2(wv, `(function(){ var st = window.__state; return st ? st.workspaceRoot : 'no-state'; })()`))

	fmt.Printf("[wsswitch] 完成\n")
}
