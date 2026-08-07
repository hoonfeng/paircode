// Command conv_list_probe 验证桌面端 bridge /api/conversations 返回
//（对话列表"现在空的"问题：bridge 数据链路 vs 前端渲染链路）。
package main

import (
	"fmt"
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

	wv := webkit.NewWebView()
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		rt.RunJS(`window.__DESKTOP_PANEL_MODE__ = true;`)
	}
	wv.LoadHTML(string(htmlData))
	time.Sleep(800 * time.Millisecond)

	// 1) 直接调 go.bridge_call 拿会话列表（手动编码 %5C）
	iv, _ := wv.JSInterpreter().RunJS(`(function(){
		var out = {};
		try {
			var r = go.bridge_call('GET', '/api/conversations?workspace=F%3A%5Csyproject%5Cgou-ide', '', '');
			out.bridge = (typeof r === 'string') ? r : JSON.stringify(r);
		} catch(e) { out.bridgeErr = e.message || String(e); }
		try {
			var st = window.__state;
			out.hasState = !!st;
			if (st) {
				out.workspaceRoot = st.workspaceRoot;
				out.convs = Array.isArray(st.conversations) ? st.conversations.length : 'n/a';
			}
		} catch(e) { out.stateErr = e.message || String(e); }
		return JSON.stringify(out);
	})()`)
	s := iv.ToString()
	fmt.Println("[convlist] direct:", s[:min(len(s), 2500)])

	// 2) 模拟前端 api.js → fetch 拦截的完整路径：
	//    apiURL: new URL + searchParams.set → u.toString()（完整 URL）
	//    fetch 拦截: 剥离 origin → path + 原始 query 提取 → params 解码 → bridge_call
	iv2, _ := wv.JSInterpreter().RunJS(`(function(){
		var out = {};
		try {
			var u = new URL('/api/conversations', window.location.origin || 'http://localhost');
			u.searchParams.set('workspace', 'F:\\syproject\\gou-ide');
			var nu = u.toString();
			out.url = nu;
			// ── fetch 拦截逻辑（desktopbridge.injectJSBridge 原样）──
			if (nu.indexOf('://') >= 0) {
				var _u = new URL(nu);
				nu = _u.pathname + (nu.indexOf('?') >= 0 ? nu.substring(nu.indexOf('?')) : '');
			}
			out.normalized = nu;
			var qIdx = nu.indexOf('?');
			var path = qIdx >= 0 ? nu.substring(0, qIdx) : nu;
			var params = {};
			if (qIdx >= 0) {
				var qs = nu.substring(qIdx + 1);
				qs.split('&').forEach(function(pair) {
					var kv = pair.split('=');
					if (kv[0]) {
						var k = kv[0], v = kv.length > 1 ? kv[1] : '';
						try { k = decodeURIComponent(k); } catch(e) {}
						try { v = decodeURIComponent(v); } catch(e) {}
						params[k] = v;
					}
				});
			}
			out.path = path; out.params = params;
			var r = go.bridge_call('GET', path, '', JSON.stringify(params));
			out.resp = (typeof r === 'string') ? r : JSON.stringify(r);
		} catch(e) { out.err = e.message || String(e); }
		return JSON.stringify(out);
	})()`)
	s2 := iv2.ToString()
	fmt.Println("[convlist] fetch-pipeline:", s2[:min(len(s2), 2500)])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
