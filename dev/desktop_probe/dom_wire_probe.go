// Command dom_wire_probe 用可靠手段回答两个问题：
// 1. DOM 树指针是否真的坏（Go 侧直接遍历 nodeBase，无 wrapper 误报）
// 2. body 顶层幽灵节点是谁插入的（BeforePageScripts hook appendChild/insertBefore，
//    在 Vue 执行前安装，记录所有 DOM 变更）
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/bindings"
	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

// goSideIntegrity 从 Go 侧遍历 DOM 树，检查 nodeBase 指针一致性（无 JS wrapper）。
// 返回 parent/sib/first/last 断裂计数 + 示例。
func goSideIntegrity(doc *dom.Document) string {
	type br struct{ path string }
	var parentBr, sibBr, fcBr, lcBr []br
	var count int
	var walk func(n dom.Node, path string, depth int)
	walk = func(n dom.Node, path string, depth int) {
		if n == nil || depth > 60 {
			return
		}
		count++
		kids := n.ChildNodes()
		fc := n.FirstChild()
		lc := n.LastChild()
		if len(kids) > 0 {
			if fc != kids[0] {
				if len(fcBr) < 8 {
					fcBr = append(fcBr, br{path + " firstChild!=childNodes[0]"})
				}
			}
			if lc != kids[len(kids)-1] {
				if len(lcBr) < 8 {
					lcBr = append(lcBr, br{path + " lastChild!=childNodes[last]"})
				}
			}
		} else if fc != nil || lc != nil {
			if len(fcBr) < 8 {
				fcBr = append(fcBr, br{path + " 0-child but first/last set"})
			}
		}
		for i, k := range kids {
			if k.ParentNode() != n {
				if len(parentBr) < 8 {
					parentBr = append(parentBr, br{fmt.Sprintf("%s[%d] parent=%v expect=%v", path, i, nodeLabel(k.ParentNode()), nodeLabel(n))})
				}
			}
			if i > 0 && k.PreviousSibling() != kids[i-1] {
				if len(sibBr) < 8 {
					sibBr = append(sibBr, br{fmt.Sprintf("%s[%d] prevSibling broken", path, i)})
				}
			}
			if i < len(kids)-1 && k.NextSibling() != kids[i+1] {
				if len(sibBr) < 8 {
					sibBr = append(sibBr, br{fmt.Sprintf("%s[%d] nextSibling broken", path, i)})
				}
			}
			walk(k, path+"["+fmt.Sprintf("%d", i)+"]"+nodeShort(k), depth+1)
		}
	}
	walk(dom.Node(doc), "doc", 0)
	// 统计 body 顶层
	bodyKids := doc.GetElementsByTagName("body")
	bodyChildCount := -1
	if len(bodyKids) > 0 {
		bodyChildCount = len(bodyKids[0].ChildNodes())
	}
	return fmt.Sprintf("{nodes:%d parentBr:%d sibBr:%d fcBr:%d lcBr:%d bodyChildren:%d parentBrSample:%v sibBrSample:%v}",
		count, len(parentBr), len(sibBr), len(fcBr), len(lcBr), bodyChildCount, parentBr, sibBr)
}

func nodeLabel(n dom.Node) string {
	if n == nil {
		return "nil"
	}
	return n.NodeName()
}

func nodeShort(n dom.Node) string {
	switch n.NodeType() {
	case dom.NodeElement:
		if e, ok := n.(*dom.Element); ok {
			return ":" + e.TagName()
		}
	case dom.NodeText:
		return "#text"
	case dom.NodeComment:
		return "#comment"
	}
	return ":" + n.NodeName()
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

	// ★ BeforePageScripts：bindings 已注册、Vue 未执行。安装 DOM 变更日志。
	// 注意：必须在 desktopbridge.Init 之后设置（Init 会覆盖包变量），且链式包装原 hook。
	installDomHook := func(in *jsc.Interpreter) {
		hookJS := `
(function(){
	var log = [];
	window.__domLog = log;
	function tag(n){
		if (!n) return 'null';
		var t = n.nodeType + ':' + (n.tagName || n.nodeName || '');
		var tc = '';
		try { tc = String(n.textContent || '').slice(0, 14); } catch(e){}
		return t + (tc ? '(' + tc + ')' : '');
	}
	function rec(op, p, c, r){
		try {
			if (log.length < 6000) log.push(op + ' ' + tag(p) + ' <- ' + tag(c) + (r ? ' before ' + tag(r) : ''));
			// ★ 记录 body 顶层插入/移除时的 JS 调用栈（定位 Vue 哪个函数）
			if (p && p.nodeName === 'BODY' && log.length < 6000) {
				try { log.push('STACK: ' + new Error().stack.split('\n').slice(1, 7).join(' | ')); } catch(e2){}
			}
		} catch(e){}
	}
	// bindings 的 appendChild/insertBefore 是 per-instance 方法（wrapElement 里 obj.Set），
	// 修改 Element.prototype 无效。document.body 由 elementWrapperCache 缓存，Vue 每次
	// 访问 document.body 拿到同一个 wrapper → 实例 hook 有效。
	var body = document.body;
	if (body) {
		var bA = body.appendChild;
		body.appendChild = function(c){ rec('A', this, c, null); return bA.apply(this, arguments); };
		var bI = body.insertBefore;
		body.insertBefore = function(c, r){ rec('I', this, c, r); return bI.apply(this, arguments); };
		var bR = body.removeChild;
		body.removeChild = function(c){ rec('R', this, c, null); return bR.apply(this, arguments); };
	}
	var dc = document;
	var origCE = dc.createElement;
	dc.createElement = function(t){ try { if (log.length < 6000) log.push('CE ' + t); } catch(e){} return origCE.apply(this, arguments); };
	var origCT = dc.createTextNode;
	dc.createTextNode = function(t){ try { if (log.length < 6000) log.push('CT ' + String(t).slice(0,8)); } catch(e){} return origCT.apply(this, arguments); };
	window.__hookReady = true;
})();`
		if _, err := in.RunJS(hookJS); err != nil {
			log.Printf("[hook] install failed: %v", err)
		} else {
			log.Printf("[hook] installed (BeforePageScripts)")
		}
	}

	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)

	// ★ 包装 OnNodeInserted/OnNodeRemoved（bindings 每次 JS 插入/移除都会调用），
	//   在 Go 侧记录 (父节点 → 子节点) 映射，精确追踪幽灵节点的插入来源。
	type insRec struct{ parent, child string }
	var insertLog []insRec
	prevIns := bindings.OnNodeInserted
	prevRem := bindings.OnNodeRemoved
	bindings.OnNodeInserted = func(n dom.Node) {
		if prevIns != nil {
			prevIns(n)
		}
		if n != nil {
			p := n.ParentNode()
			pc, cc := "", ""
			if p != nil {
				pc = p.NodeName()
				if e, ok := p.(*dom.Element); ok {
					pc = e.TagName() + "#" + e.GetAttribute("id")
				}
			}
			cc = n.NodeName()
			if e, ok := n.(*dom.Element); ok {
				cc = e.TagName() + "." + e.GetAttribute("class")
			}
			insertLog = append(insertLog, insRec{pc, cc})
		}
	}
	bindings.OnNodeRemoved = func(n dom.Node) {
		if prevRem != nil {
			prevRem(n)
		}
	}
	_ = insertLog

	// ★ desktopbridge.Init 已覆盖 BeforePageScripts，这里链式包装：
	//   先跑原 hook（fetch 拦截等），再安装 DOM 变更日志（document.body 实例 hook + create* 计数）。
	prevHook := webkit.BeforePageScripts
	webkit.BeforePageScripts = func(in *jsc.Interpreter) {
		if prevHook != nil {
			prevHook(in)
		}
		installDomHook(in)
	}

	wv.LoadHTML(string(htmlData))

	waitJS := func(ms int) {
		_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
	waitJS(400)
	waitJS(600)

	// ★ 1. Go 侧完整性（LoadHTML 后立即）
	doc := wv.MainFrame().Document()
	if doc != nil {
		fmt.Printf("[go-integrity after-load] %s\n", goSideIntegrity(doc))
	}

	// ★ 2. DOM 变更日志分析（body 顶层插入记录）
	iv, _ := wv.JSInterpreter().RunJS(`(function(){
		var log = window.__domLog || [];
		var bodyOps = [];
		for (var i=0;i<log.length;i++) {
			var m = log[i];
			if (m.indexOf('A BODY') === 0 || m.indexOf('I BODY') === 0) bodyOps.push(m);
		}
		var byOp = {};
		for (var i=0;i<log.length;i++) {
			var op = log[i].slice(0,1);
			byOp[op] = (byOp[op]||0) + 1;
		}
		return JSON.stringify({total: log.length, byOp: byOp, bodyOps: bodyOps.slice(0, 30), bodyOpCount: bodyOps.length, first: log.slice(0, 15), last: log.slice(-15)});
	})()`)
		fmt.Printf("[domlog] %s\n", iv.ToString())
		// ★ 提取带 STACK 的完整调用栈
		iv, _ = wv.JSInterpreter().RunJS(`(function(){
			var log = window.__domLog || [];
			var stacks = [];
			var ops = [];
			for (var i=0;i<log.length;i++) {
				var m = log[i];
				if (m.indexOf('STACK:') === 0) stacks.push(m.slice(6));
				else if (m.indexOf('I 1:BODY') === 0 || m.indexOf('R 1:BODY') === 0 || m.indexOf('A 1:BODY') === 0) ops.push(m);
			}
			return JSON.stringify({stackCount: stacks.length, uniqueStacks: stacks.filter(function(s,i){return stacks.indexOf(s)===i;}).slice(0,6), opsHead: ops.slice(0,10), opsTail: ops.slice(-6)});
		})()`)
		fmt.Printf("[bodyops] %s\n", iv.ToString())

	// ★ 3. 幽灵节点身份（从 body 顶层读 nodeType+nodeName+parent，避免 wrapper 误报）
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var b = document.body;
		var out = {count: b.childNodes.length};
		var arr = [];
		for (var i=0;i<b.childNodes.length && i<12;i++) {
			var n = b.childNodes[i];
			arr.push({i:i, t:n.nodeType, name:n.nodeName||'', tc:String(n.textContent||'').slice(0,8), parent: n.parentNode===b ? 'body' : 'OTHER'});
		}
		out.head = arr;
		return JSON.stringify(out);
	})()`)
	fmt.Printf("[ghost] %s\n", iv.ToString())

	// ★ 3.5 Go 侧 OnNodeInserted 日志：谁往 body 顶层插了节点？
	bodyIns := 0
	appIns := 0
	otherIns := 0
	for _, r := range insertLog {
		switch {
		case r.parent == "body" || strings.HasPrefix(r.parent, "BODY"):
			bodyIns++
		case strings.Contains(r.parent, "#app"):
			appIns++
		default:
			otherIns++
		}
	}
	fmt.Printf("[insertlog] total=%d body=%d app=%d other=%d\n", len(insertLog), bodyIns, appIns, otherIns)
	shown := 0
	for _, r := range insertLog {
		if strings.HasPrefix(r.parent, "body") || r.parent == "BODY" {
			fmt.Printf("[insertlog] BODY <- %s\n", r.child)
			shown++
			if shown >= 25 {
				break
			}
		}
	}
	if shown == 0 && bodyIns == 0 {
		fmt.Printf("[insertlog] (no body inserts recorded — check hook)\n")
	}

	// ★ 3.6 Vue 挂载状态 + domlog 自测
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var app = document.querySelector('#app');
		var out = {
			vnode: !!(app && app._vnode),
			hasState: !!window.__state,
			hookReady: !!window.__hookReady,
			domLogLen: (window.__domLog||[]).length
		};
		// hook 自测：手动往 body 追加一个文本节点，验证实例 hook 是否工作
		try {
			var t = document.createTextNode('selftest');
			document.body.appendChild(t);
			out.selfTest = true;
		} catch(e) { out.selfTest = 'err: ' + String(e).slice(0,80); }
		return JSON.stringify(out);
	})()`)
	fmt.Printf("[vuestate] %s\n", iv.ToString())
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var log = window.__domLog || [];
		var tail = log.slice(-8);
		return JSON.stringify({len: log.length, tail: tail});
	})()`)
	fmt.Printf("[domlog-after-selftest] %s\n", iv.ToString())

	// ★ 4. 数据注入 + 渲染，再看完整性
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

	// 触发一次渲染流程
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	roCount := 0
	var walkRO func(o rendering.RenderObject)
	walkRO = func(o rendering.RenderObject) {
		roCount++
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walkRO(c)
		}
	}
	if rv != nil {
		walkRO(rendering.RenderObject(rv))
	}
	doc = wv.MainFrame().Document()
	fmt.Printf("[go-integrity after-render] %s\n", goSideIntegrity(doc))
	fmt.Printf("[ro] renderObjects=%d\n", roCount)

	// 错误日志
	iv, _ = wv.JSInterpreter().RunJS(`(function(){ var e=window.__err||[]; return JSON.stringify({count:e.length, list:e.slice(0,8)}); })()`)
	fmt.Printf("[errors] %s\n", iv.ToString())
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
