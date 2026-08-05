// Command dom_integrity_probe 检查 #app 内部 DOM 树的指针一致性：
// 1. 每个元素的 childNodes[i].parentNode === 自身（父指针校验）
// 2. childNodes[i].nextSibling === childNodes[i+1]（兄弟链校验）
// 3. childNodes 与 firstChild/lastChild 一致性
// 4. 渲染树节点数 vs DOM 元素数
// 定位 Vue patch 崩溃（nextSibling of undefined）的 DOM 断裂点。
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
	"wb-ui/rendering"
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

	// 错误捕获
	errHook := `<script>
	(function(){
		window.__err = [];
		var oe = console.error.bind(console);
		console.error = function(){ try { window.__err.push(Array.prototype.slice.call(arguments).map(String).join(' ').slice(0,200)); } catch(e){} return oe.apply(console, arguments); };
		window.onerror = function(m){ try { window.__err.push('onerror: ' + String(m).slice(0,150)); } catch(e){} };
	})();
	</script>`
	htmlStr := string(htmlData)
	htmlStr = strings.Replace(htmlStr, "<script type=\"module\"", errHook+"<script type=\"module\"", 1)
	wv.LoadHTML(htmlStr)
	waitJS := func(ms int) {
		_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
	waitJS(400)
	waitJS(600)

	// 注入数据
	_, _ = wv.JSInterpreter().RunJS(`(function(){
		var st = window.__state;
		if (!st) return 'no state';
		st.workspaceRoot = 'F:\\syproject\\gou-ide';
		return 'ok';
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`(function(){
		var p = fetch('/api/conversations?workspace=' + encodeURIComponent('F:\\syproject\\gou-ide')).then(function(r){ return r.json(); }).then(function(list){
			var st = window.__state;
			if (Array.isArray(list) && list[0]) {
				st.conversations = list;
				st.currentConvId = list[0].id;
				st.messages = [];
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

	// ★ DOM 指针一致性检查（JS 侧遍历 #app 全部节点）
	js := `(function(){
		var out = {nodes: 0, elems: 0, parentBroken: [], sibBroken: [], firstChildBroken: [], lastChildBroken: [], depth: 0};
		var app = document.querySelector('#app');
		if (!app) return JSON.stringify({err: 'no app'});
		var rootEl = app.firstChild; // app-root
		function check(n, path, d) {
			if (!n) return;
			out.nodes++;
			if (n.nodeType === 1) out.elems++;
			if (d > out.depth) out.depth = d;
			// childNodes 内部一致性
			var kids = n.childNodes;
			var fc = n.firstChild;
			var lc = n.lastChild;
			if (kids.length > 0) {
				if (fc !== kids[0] && out.firstChildBroken.length < 10) out.firstChildBroken.push(path + ' firstChild!=childNodes[0]');
				if (lc !== kids[kids.length-1] && out.lastChildBroken.length < 10) out.lastChildBroken.push(path + ' lastChild!=childNodes[last]');
			} else {
				if (fc !== null && out.firstChildBroken.length < 10) out.firstChildBroken.push(path + ' has firstChild but 0 childNodes');
				if (lc !== null && out.lastChildBroken.length < 10) out.lastChildBroken.push(path + ' has lastChild but 0 childNodes');
			}
			for (var i=0; i<kids.length; i++) {
				var k = kids[i];
				if (k.parentNode !== n && out.parentBroken.length < 10) out.parentBroken.push(path + '[' + i + '] parent=' + (k.parentNode ? k.parentNode.tagName || k.parentNode.nodeName : 'null') + ' expect=' + (n.tagName||n.nodeName));
				if (i > 0 && k.previousSibling !== kids[i-1] && out.sibBroken.length < 10) out.sibBroken.push(path + '[' + i + '] prevSibling broken');
				if (i < kids.length-1 && k.nextSibling !== kids[i+1] && out.sibBroken.length < 10) out.sibBroken.push(path + '[' + i + '] nextSibling broken');
				check(k, path + '[' + i + ']' + (k.nodeType===1?':'+k.tagName:''), d+1);
			}
		}
		check(rootEl, 'app-root', 0);
		// 渲染树检查放 Go 侧，这里先统计 DOM
		return JSON.stringify(out);
	})()`
	v, err := wv.JSInterpreter().RunJS(js)
	if err != nil {
		log.Printf("integrity err: %v", err)
	} else {
		fmt.Printf("[integrity] %s\n", v.ToString())
	}

	// 错误日志
	iv, _ := wv.JSInterpreter().RunJS(`(function(){ var e=window.__err||[]; return JSON.stringify({count:e.length, list:e.slice(0,10)}); })()`)
	fmt.Printf("[errors] %s\n", iv.ToString())

	// ★ Go 侧渲染树 vs DOM 元素数
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}
	doc := wv.MainFrame().Document()
	domEls := doc.GetElementsByTagName("*")
	roCount := 0
	var walkRO func(o rendering.RenderObject)
	walkRO = func(o rendering.RenderObject) {
		roCount++
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walkRO(c)
		}
	}
	walkRO(rendering.RenderObject(rv))
	fmt.Printf("[ro] domElements=%d renderObjects=%d\n", len(domEls), roCount)

	// 渲染树里与 DOM 不一致的（有 RO 但 Node 不在 DOM）
	orphanRO := 0
	var walkRO2 func(o rendering.RenderObject)
	walkRO2 = func(o rendering.RenderObject) {
		n := o.Node()
		if n != nil {
			// 检查 node 是否在 document 树中（parentNode 链到 document）
			cur := n
			hops := 0
			for cur != nil && hops < 40 {
				p := cur.ParentNode()
				if p == nil {
					// 到顶
					break
				}
				cur = p
				hops++
			}
			if cur != nil && cur.NodeName() != "#document" {
				orphanRO++
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walkRO2(c)
		}
	}
	walkRO2(rendering.RenderObject(rv))
	fmt.Printf("[ro] renderObjects not in document: %d\n", orphanRO)
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
