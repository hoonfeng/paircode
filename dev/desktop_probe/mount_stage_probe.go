// Command mount_stage_probe 分阶段检查 Vue 挂载过程中的 DOM 变化：
// 阶段0: LoadHTML 后（script 未执行）→ body 子节点
// 阶段1: Vue 挂载后（waitJS 400ms）→ body 子节点 + #app 内部
// 阶段2: 注入对话数据后 → body 子节点（是否又泄漏）
// 对比各阶段 body childNodes 数量与构成，定位 198 个空节点何时、如何插入。
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

	dumpBody := func(tag string) {
		js := `(function(){
			var out = {total:0, txt:0, cmt:0, els:[], first:[]};
			var bks = document.body.childNodes;
			out.total = bks.length;
			for (var i=0; i<bks.length; i++) {
				var n = bks[i];
				if (n.nodeType === 3) out.txt++;
				else if (n.nodeType === 8) out.cmt++;
				else if (n.nodeType === 1) out.els.push(n.tagName + '#' + (n.id||'') + '.' + (n.className||'').toString().slice(0,20));
				if (i < 12) {
					var d;
					if (n.nodeType === 1) d = 'E:' + n.tagName;
					else if (n.nodeType === 3) d = 'T:"' + (n.textContent||'').slice(0,12) + '"';
					else if (n.nodeType === 8) d = 'C:' + (n.textContent||'').slice(0,12);
					else d = 'N' + n.nodeType;
					out.first.push(d);
				}
			}
			return JSON.stringify(out);
		})()`
		v, err := wv.JSInterpreter().RunJS(js)
		if err != nil {
			log.Printf("[%s] dump err: %v", tag, err)
			return
		}
		fmt.Printf("[%s] %s\n", tag, v.ToString())
	}

	// 阶段0: 加载前先注入一个 hook，统计 appendChild/insertBefore 的调用方
	_, _ = wv.JSInterpreter().RunJS(`(function(){
		window.__mut = {appends: 0, inserts: 0, removes: 0, commentToBody: 0, textToBody: 0};
		var origAppend = Element.prototype.appendChild;
		var origInsert = Element.prototype.insertBefore;
		Element.prototype.appendChild = function(n) {
			window.__mut.appends++;
			if (this === document.body && (n.nodeType === 8 || n.nodeType === 3)) {
				window.__mut.commentToBody++;
				if (window.__mut.bodyLogs === undefined) window.__mut.bodyLogs = [];
				if (window.__mut.bodyLogs.length < 20) {
					window.__mut.bodyLogs.push('append body ' + (n.nodeType===8?'C:':'T:"') + (n.textContent||'').slice(0,20) + '"');
				}
			}
			return origAppend.call(this, n);
		};
		Element.prototype.insertBefore = function(n, ref) {
			window.__mut.inserts++;
			if (this === document.body && (n.nodeType === 8 || n.nodeType === 3)) {
				window.__mut.commentToBody++;
				if (window.__mut.bodyLogs === undefined) window.__mut.bodyLogs = [];
				if (window.__mut.bodyLogs.length < 20) {
					window.__mut.bodyLogs.push('insert body ' + (n.nodeType===8?'C:':'T:"') + (n.textContent||'').slice(0,20) + '" ref=' + (ref ? (ref.nodeType===8?'C':'E:'+ref.tagName) : 'null'));
				}
			}
			return origInsert.call(this, n, ref);
		};
		return 'hooked';
	})()`)

	wv.LoadHTML(string(htmlData))
	waitJS := func(ms int) {
		_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
	waitJS(300)
	dumpBody("stage0-after-load")
	waitJS(600)
	dumpBody("stage1-vue-mounted")

	// 注入数据
	_, _ = wv.JSInterpreter().RunJS(`(function(){
		var st = window.__state;
		if (st) st.workspaceRoot = 'F:\\syproject\\gou-ide';
		return st ? 'ok' : 'no state';
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`(function(){
		var p = fetch('/api/conversations?workspace=' + encodeURIComponent('F:\\syproject\\gou-ide')).then(function(r){ return r.json(); }).then(function(list){
			var st = window.__state;
			if (Array.isArray(list) && list[0]) {
				st.conversations = list;
				st.currentConvId = list[0].id;
				return fetch('/api/conversations/' + encodeURIComponent(list[0].id) + '/messages').then(function(r){ return r.json(); }).then(function(msgs){
					if (Array.isArray(msgs)) { st.messages = msgs; st.messagesByConv[list[0].id] = msgs; }
					return 'done';
				});
			}
			return 'no list';
		});
		return 'started';
	})()`)
	waitJS(800)
	dumpBody("stage2-after-data")

	// 输出 mutation 统计
	js := `(function(){
		var m = window.__mut;
		return JSON.stringify(m);
	})()`
	v, err := wv.JSInterpreter().RunJS(js)
	if err != nil {
		log.Printf("mut err: %v", err)
	} else {
		fmt.Printf("[mut] %s\n", v.ToString())
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
