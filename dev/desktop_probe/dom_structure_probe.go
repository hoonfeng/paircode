// Command dom_structure_probe 查看真实 dist 加载后的 DOM 结构：
// 1. body 顶层子节点构成（找泄漏的 text/comment 锚点来自哪个组件）
// 2. #app 内部第一层结构
// 3. conv-list 中 conv-item 的祖先链（判断挂载位置是否正确）
// 4. Vue vnode 树中 el 缺失节点的位置模式（badCount 的分布）
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

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	absDist, _ := filepath.Abs(distDir)
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
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	waitJS := func(ms int) {
		_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
	waitJS(500)
	waitJS(800)

	// ★ 注入数据（同 clean probe）
	_, _ = wv.JSInterpreter().RunJS(`(function(){
		var st = window.__state;
		if (!st) return 'no state';
		st.workspaceRoot = 'F:\\syproject\\gou-ide';
		return 'ok';
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`(function(){
		var p = fetch('/api/conversations?workspace=' + encodeURIComponent('F:\\syproject\\gou-ide')).then(function(r){ return r.json(); }).then(function(list){
			var st = window.__state;
			if (Array.isArray(list)) {
				st.conversations = list;
				if (list[0]) { st.currentConvId = list[0].id; st.messages = []; }
				return fetch('/api/conversations/' + encodeURIComponent(list[0].id) + '/messages').then(function(r){ return r.json(); }).then(function(msgs){
					if (Array.isArray(msgs)) { st.messages = msgs; st.messagesByConv[list[0].id] = msgs; }
					return 'done ' + msgs.length;
				});
			}
			return 'no list';
		});
		return 'started';
	})()`)
	waitJS(1000)
	// ★ body 顶层结构 + 指针一致性检查
	dumpJS := `(function(){
		var out = {};
		var bks = document.body.childNodes;
		out.bodyTotal = bks.length;
		// 指针一致性：childNodes 里的节点 parentNode 是否真的是 body
		out.ptrBroken = 0;
		out.elemInBody = 0;
		for (var i=0; i<bks.length; i++) {
			var n = bks[i];
			if (n.parentNode !== document.body) out.ptrBroken++;
			if (n.nodeType === 1) out.elemInBody++;
		}
		out.bodyChildren = [];
		for (var i=0; i<bks.length && i<90; i++) {
			var n = bks[i];
			var desc;
			if (n.nodeType === 1) {
				var cls = (n.className || '').toString();
				desc = 'E:' + n.tagName + (cls ? '.' + cls.slice(0, 40) : '') + ' id=' + (n.id||'');
			} else if (n.nodeType === 3) {
				desc = 'T:"' + (n.textContent||'').slice(0, 30) + '"';
			} else if (n.nodeType === 8) {
				desc = 'C:' + (n.textContent||'').slice(0, 30);
			} else {
				desc = 'N' + n.nodeType;
			}
			out.bodyChildren.push(desc);
		}
		// #app 内部
		var app = document.querySelector('#app');
		out.appChildren = [];
		if (app) {
			out.appParentIsBody = app.parentNode === document.body;
			var acs = app.childNodes;
			out.appTotal = acs.length;
			for (var j=0; j<acs.length && j<40; j++) {
				var n2 = acs[j];
				var d2;
				if (n2.nodeType === 1) {
					d2 = 'E:' + n2.tagName + '.' + (n2.className||'').toString().slice(0,50);
				} else if (n2.nodeType === 3) {
					d2 = 'T:"' + (n2.textContent||'').slice(0,20) + '"';
				} else if (n2.nodeType === 8) {
					d2 = 'C:' + (n2.textContent||'').slice(0,20);
				} else { d2 = 'N' + n2.nodeType; }
				out.appChildren.push(d2);
			}
		}
		// conv-item 祖先链
		var ci = document.querySelectorAll('.conv-item')[0];
		out.convAncestry = [];
		var cur = ci;
		var hops = 0;
		while (cur && cur !== document.body && cur !== document.documentElement && hops < 25) {
			var d3;
			if (cur.nodeType === 1) d3 = cur.tagName + '.' + (cur.className||'').toString().slice(0,50);
			else if (cur.nodeType === 8) d3 = 'C:' + (cur.textContent||'').slice(0,20);
			else d3 = 'N' + cur.nodeType + ':"' + (cur.textContent||'').slice(0,20) + '"';
			out.convAncestry.push(d3);
			cur = cur.parentNode;
			hops++;
		}
		out.convAncestry.push('-> ' + (cur ? cur.tagName : 'null'));
		out.convCount = document.querySelectorAll('.conv-item').length;
		return JSON.stringify(out);
	})()`
	v, err := wv.JSInterpreter().RunJS(dumpJS)
	if err != nil {
		log.Printf("dump err: %v", err)
	} else {
		fmt.Printf("[dom] %s\n", v.ToString())
	}
}

func setupLoaders(wv *webkit.WebView, distDir string) {
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
