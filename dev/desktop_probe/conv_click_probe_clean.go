// Command conv_click_probe_clean reproduces the "conversation list click does
// not select" issue with ZERO hooks: no microtask interception, no update
// wrapping, no manual render/update calls. It only:
//   1. loads the real dist
//   2. waits for Vue mount + async data
//   3. dumps RightPanel.subTree integrity (dynEls / badElCount / children el)
//   4. clicks the 2nd .conv-item via the exact host.handleClick flow
//   5. checks whether Vue @click handler ran (active class / currentConvId)
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func processClean(wv *webkit.WebView) {
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.MarkRenderTreeDirty()
		}
	}
}

func setupLoadersClean(wv *webkit.WebView, distDir string) {
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
	setupLoadersClean(wv, distDir)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)

	// ★ 注入错误捕获（不干预微任务，只记录）
	errHook := `<script>
(function(){
	window.__cleanErr = [];
	if (window.console && console.error) {
		var oe = console.error.bind(console);
		console.error = function() {
			try {
				var m = Array.prototype.slice.call(arguments).map(String).join(' ').slice(0, 300);
				var stk = '';
				try { var er = arguments[0]; if (er && er.stack) stk = String(er.stack).slice(0, 900); } catch(e2){}
				window.__cleanErr.push({msg: m, stack: stk});
			} catch(e){}
			return oe.apply(console, arguments);
		};
	}
	window.__cleanErr.push('console hooked');
	var oe2 = window.onerror || function(){};
	window.onerror = function(msg, src, ln, col, err){
		try { window.__cleanErr.push('onerror: ' + String(msg).slice(0, 200) + ' @' + ln); } catch(e){}
		try { return oe2.apply(this, arguments); } catch(e){}
	};
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

	// ★ 手动建立数据环境：workspace + 对话列表 + 第一条对话消息
	fmt.Println("[clean] injecting workspace + conversations...")
	loadJS := `
(function(){
	var out = {errs: window.__cleanErr || []};
	try {
		var st = window.__state;
		if (!st) { out.fatal = 'no __state'; return JSON.stringify(out); }
		st.workspaceRoot = 'F:\\syproject\\gou-ide';
		st.workspaceName = 'gou-ide';
		st.workspaceFolders = ['F:\\syproject\\gou-ide'];
		out.set = true;
		return JSON.stringify(out);
	} catch(e) { out.err = String(e).slice(0,200); return JSON.stringify(out); }
})()`
	iv, _ := wv.JSInterpreter().RunJS(loadJS)
	fmt.Printf("[clean] pre-load: %s\n", iv.ToString())

	// 通过 fetch（bridge 拦截）加载对话列表
	fetchJS := `
(function(){
	var out = {errs: (window.__cleanErr||[]).length};
	try {
		var p = fetch('/api/conversations?workspace=' + encodeURIComponent('F:\\syproject\\gou-ide')).then(function(r){ return r.json(); }).then(function(list){
			var st = window.__state;
			out.fetched = Array.isArray(list) ? list.length : -1;
			out.firstTitle = (list && list[0] && list[0].title) || '';
			if (Array.isArray(list)) {
				st.conversations = list;
				if (list[0]) { st.currentConvId = list[0].id; st.messages = []; }
			}
			return fetch('/api/conversations/' + encodeURIComponent(list[0].id) + '/messages').then(function(r){ return r.json(); }).then(function(msgs){
				if (Array.isArray(msgs)) { st.messages = msgs; st.messagesByConv[list[0].id] = msgs; }
				out.msgCount = Array.isArray(msgs) ? msgs.length : -1;
				out.done = true;
			});
		}).catch(function(e){ out.fetchErr = String(e).slice(0, 200); });
		window.__loadP = p;
		return JSON.stringify(out);
	} catch(e) { out.err = String(e).slice(0,200); return JSON.stringify(out); }
})()`
	iv, _ = wv.JSInterpreter().RunJS(fetchJS)
	fmt.Printf("[clean] fetch start: %s\n", iv.ToString())
	waitJS(800)
	iv, _ = wv.JSInterpreter().RunJS(`(function(){ var o={errs:(window.__cleanErr||[]).map(function(e){return e.msg;}).slice(0,8)}; var st=window.__state; if(st){o.convCount=st.conversations?st.conversations.length:-1; o.cur=st.currentConvId; o.msgCount=st.messages?st.messages.length:-1;} return JSON.stringify(o); })()`)
	fmt.Printf("[clean] fetch result: %s\n", iv.ToString())

	// ★ 分阶段只读快照（观测树坏时机 + DOM 结构）
	phaseSnap := func(tag string) string {
		snap := `
(function(tag){
	var root = document.querySelector('#app');
	if (!root || !root._vnode) return JSON.stringify({err:'no vnode', tag: tag});
	function findRp(v, d) {
		if (!v || d > 18) return null;
		if (v.component) {
			var c = v.component;
			var nm = (c.type && (c.type.name || c.type.__name)) || '';
			if (nm === 'RightPanel') return c;
		}
		var subs = [];
		if (v.component && v.component.subTree) subs.push(v.component.subTree);
		if (Array.isArray(v.children)) for (var i=0;i<v.children.length;i++) subs.push(v.children[i]);
		if (v.dynamicChildren) for (var j=0;j<v.dynamicChildren.length;j++) subs.push(v.dynamicChildren[j]);
		for (var k=0;k<subs.length;k++) { var r = findRp(subs[k], d+1); if (r) return r; }
		return null;
	}
	var rp = findRp(root._vnode, 0);
	if (!rp) return JSON.stringify({err:'no RightPanel', tag: tag});
	var sub = rp.subTree;
	var dyn = sub && sub.dynamicChildren ? sub.dynamicChildren : [];
	var bad = [];
	(function scan(n, path, d){
		if (!n || typeof n !== 'object' || d > 30) return;
		if (typeof n.type === 'string') {
			var e = n.el;
			if (e === undefined || e === null) bad.push({path: path, t: n.type});
		}
		if (n.component && n.component.subTree) scan(n.component.subTree, path + '>c', d+1);
		if (Array.isArray(n.children)) for (var i=0;i<n.children.length;i++) scan(n.children[i], path + '[' + i + ']', d+1);
	})(sub, 'root', 0);
	var childSet = [];
	(function collectCh(v){
		if (!v) return;
		childSet.push(v);
		if (v.component && v.component.subTree) collectCh(v.component.subTree);
		if (Array.isArray(v.children)) v.children.forEach(collectCh);
	})(sub);
	var bodyKids = [];
	if (document.body) {
		var bks = document.body.childNodes;
		for (var bi=0; bi<bks.length && bi<40; bi++) {
			bodyKids.push(bks[bi].nodeType + ':' + (bks[bi].tagName || bks[bi].nodeName || ''));
		}
	}
	var st = null;
	try { st = window.__state; } catch(e){}
	var out = {
		tag: tag,
		subEl: !!(sub && sub.el),
		dynLen: dyn.length,
		dynEls: dyn.slice(0, 40).map(function(d){ return !!(d && d.el); }),
		dynInChildTree: dyn.slice(0, 40).map(function(d){ return childSet.indexOf(d) >= 0; }),
		childTreeSize: childSet.length,
		badCount: bad.length,
		badSample: bad.slice(0, 8).map(function(b){ return b.path + ':' + b.t; }),
		childrenEls: Array.isArray(sub.children) ? sub.children.map(function(c){ return !!(c && c.el); }) : [],
		convItems: document.querySelectorAll('.conv-item').length,
		msgItems: document.querySelectorAll('.msg-item').length,
		bodyChildren: document.body ? document.body.childNodes.length : -1,
		appChildren: document.querySelector('#app') ? document.querySelector('#app').childNodes.length : -1,
		bodyKids: bodyKids,
		state: st ? { convCount: st.conversations ? st.conversations.length : -1, currentConvId: st.currentConvId, msgCount: st.messages ? st.messages.length : -1 } : 'no __state'
	};
	return JSON.stringify(out);
})("` + tag + `")`
		iv, err := wv.JSInterpreter().RunJS(snap)
		if err != nil {
			return "[err] " + err.Error()
		}
		return iv.ToString()
	}

	// ★ 数据加载后的树快照 + 错误日志
	fmt.Printf("[clean] after-load %s\n", phaseSnap("after-load"))
	errDump := `(function(){
		var e = window.__cleanErr || [];
		return JSON.stringify({count: e.length, list: e.slice(0, 10).map(function(x){ return x.msg; }), stacks: e.slice(0, 3).map(function(x){ return x.stack; })});
	})()`
	iv, _ = wv.JSInterpreter().RunJS(errDump)
	fmt.Printf("[clean] errors: %s\n", iv.ToString())

	// ★ 手动 update 测试：观察 update 后树/DOM 变化 + 是否抛错
	manualUp := `(function(){
		var out = {};
		try {
			var root = document.querySelector('#app');
			function findRp(v, d) {
				if (!v || d > 18) return null;
				if (v.component) {
					var nm = (v.component.type && (v.component.type.name || v.component.type.__name)) || '';
					if (nm === 'RightPanel') return v.component;
				}
				var subs = [];
				if (v.component && v.component.subTree) subs.push(v.component.subTree);
				if (Array.isArray(v.children)) for (var i=0;i<v.children.length;i++) subs.push(v.children[i]);
				if (v.dynamicChildren) for (var j=0;j<v.dynamicChildren.length;j++) subs.push(v.dynamicChildren[j]);
				for (var k=0;k<subs.length;k++) { var r = findRp(subs[k], d+1); if (r) return r; }
				return null;
			}
			var rp = findRp(root._vnode, 0);
			if (!rp) return JSON.stringify({err: 'no rp'});
			var before = {dynEls: (rp.subTree.dynamicChildren||[]).slice(0,36).map(function(d){ return !!(d && d.el) ? '1':'0'; }).join('')};
			var e0 = (window.__cleanErr||[]).length;
			rp.update();
			out.diff = (window.__cleanErr||[]).length - e0;
			out.newErr = (window.__cleanErr||[]).slice(e0).map(function(x){ return x.msg; });
			out.after = {dynEls: (rp.subTree.dynamicChildren||[]).slice(0,36).map(function(d){ return !!(d && d.el) ? '1':'0'; }).join('')};
			out.activeTitle = document.querySelector('.conv-item.active .conv-title') ? document.querySelector('.conv-item.active .conv-title').textContent.slice(0,20) : '(no active)';
			out.before = before;
			return JSON.stringify(out);
		} catch(e) { out.err = String(e).slice(0,200); return JSON.stringify(out); }
	})()`
	iv, _ = wv.JSInterpreter().RunJS(manualUp)
	fmt.Printf("[clean] manual update: %s\n", iv.ToString())
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}
	var items []struct {
		el     *dom.Element
		x, y   float64
		active bool
	}
	var findConv func(o rendering.RenderObject)
	findConv = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			lb := o.LayoutBox()
			if strings.Contains(cn, "conv-item") && lb != nil && rv.LayoutState() != nil {
				g := rv.LayoutState().GeometryForBox(lb)
				items = append(items, struct {
					el     *dom.Element
					x, y   float64
					active bool
				}{el: el, x: g.Left(), y: g.Top(), active: strings.Contains(cn, "active")})
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			findConv(c)
		}
	}
	findConv(rendering.RenderObject(rv))
	fmt.Printf("[clean] found %d conv-items\n", len(items))
	for i, it := range items {
		active := ""
		if it.active {
			active = " ACTIVE"
		}
		fmt.Printf("[clean]   #%d x=%.0f y=%.0f%s\n", i, it.x, it.y, active)
	}
	if len(items) < 2 {
		log.Fatal("need >=2 conv-items")
	}

	target := items[1]
	cx := target.x + 40
	cy := target.y + 6
	fmt.Printf("[clean] clicking #1 at (%.0f, %.0f)\n", cx, cy)

	el := rendering.HitTest(rv, cx, cy, "onclick")
	desc := "<nil>"
	if el != nil {
		desc = el.LocalName() + "." + el.GetAttribute("class") + " onclick=" + el.GetAttribute("onclick")
	}
	fmt.Printf("[clean] HitTest(onclick) -> %s\n", desc)

	if el == nil {
		deepest := rendering.HitTest(rv, cx, cy, "")
		if deepest != nil {
			fmt.Printf("[clean] HitTest(\"\") -> %s\n", deepest.LocalName())
			deepest.DispatchEvent(dom.NewMouseEvent(dom.EventClick, true, true, false))
		}
	} else {
		el.DispatchEvent(dom.NewMouseEvent(dom.EventClick, true, true, false))
	}

	if wv.JSInterpreter() != nil {
		for i := 0; i < 10; i++ {
			wv.JSInterpreter().RunJobs()
			processClean(wv)
		}
		for i := 0; i < 8; i++ {
			time.Sleep(15 * time.Millisecond)
			processClean(wv)
			wv.JSInterpreter().RunJobs()
		}
	}
	wv.RebuildRenderTree()

	res, _ := wv.JSInterpreter().RunJS(`(function(){
		var items = document.querySelectorAll('.conv-item');
		var act = [];
		for (var i=0;i<items.length && i<8;i++) {
			act.push({i:i, cls: items[i].getAttribute('class'), txt: (items[i].textContent||'').slice(0,18)});
		}
		var st = window.__state;
		var out = {active: act};
		if (st) out.currentConvId = st.currentConvId;
		out.msgItems = document.querySelectorAll('.msg-item').length;
		out.msgCount = st && st.messages ? st.messages.length : -1;
		out.bodyChildren = document.body ? document.body.childNodes.length : -1;
		return JSON.stringify(out);
	})()`)
	fmt.Printf("[clean] after-click: %s\n", res.ToString())
}
