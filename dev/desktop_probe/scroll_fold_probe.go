// Command scroll_fold_probe 验证「滚动加载历史消息后折叠状态保持收缩」：
//   1. panel-only 加载真实 dist + desktopbridge（真实 .pair/conversations 存储）
//   2. 点击多 run 会话（UI 触发 switchConv）→ 检查首次加载折叠状态（applyAutoCollapse）
//   3. 派发 scroll 事件触发 loadMoreMessages 加载更早消息
//   4. 复查折叠状态：older 消息的 thinking 应折叠（_collapsed=true）、tool_call 收缩
//      （_expanded=false）、assistant 折叠成完成摘要（_folded=true）——即 loadMoreMessages
//      prepend 后调用 applyAutoCollapse 的修复效果
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func setupLoadersF(wv *webkit.WebView, distDir string) {
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
}

func waitJSF(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func runJobsF(wv *webkit.WebView) {
	for i := 0; i < 12; i++ {
		wv.JSInterpreter().RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		time.Sleep(15 * time.Millisecond)
	}
}

func runJSF(wv *webkit.WebView, script string) string {
	v, err := wv.JSInterpreter().RunJS(script)
	if err != nil {
		return "[err] " + err.Error()
	}
	return v.ToString()
}

// 统计当前对话所有 assistant 消息的折叠状态
const foldStatsJS = `(function(){
	var st = window.__state;
	var msgs = st.messagesByConv[st.currentConvId] || [];
	var s = {assistant:0, folded:0, unfolded:0, thinkTotal:0, thinkExpanded:0, thinkCollapsed:0, thinkUndef:0, toolTotal:0, toolExpanded:0, toolCollapsed:0, toolUndef:0, firstIdx:-1, userRuns:0};
	for (var i = 0; i < msgs.length; i++) {
		var m = msgs[i];
		if (i === 0 && m._idx !== undefined) s.firstIdx = m._idx;
		if (m.role === 'user') s.userRuns++;
		if (m.role !== 'assistant') continue;
		s.assistant++;
		if (m._folded) s.folded++; else s.unfolded++;
		var segs = m.segments || [];
		for (var j = 0; j < segs.length; j++) {
			var g = segs[j];
			if (g.type === 'thinking') {
				s.thinkTotal++;
				if (g._collapsed === false) s.thinkExpanded++;
				else if (g._collapsed === true) s.thinkCollapsed++;
				else s.thinkUndef++;
			}
			if (g.type === 'tool_call') {
				s.toolTotal++;
				if (g._expanded === true) s.toolExpanded++;
				else if (g._expanded === false) s.toolCollapsed++;
				else s.toolUndef++;
			}
		}
	}
	return JSON.stringify(s);
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
	layout.FontMetricsFunc = func(family string, size float64, weight int, style2 string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	wv := webkit.NewWebView()
	setupLoadersF(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	origBefore := webkit.BeforePageScripts
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		if origBefore != nil {
			origBefore(rt)
		}
		rt.RunJS(`window.__DESKTOP_PANEL_MODE__ = true;`)
	}
	wv.LoadHTML(string(htmlData))
	waitJSF(wv, 500)
	waitJSF(wv, 500)
	runJobsF(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	runJSF(wv, `(function(){ var st = window.__state;
		if (!st) return 'no state';
		st.workspaceRoot = 'F:\\syproject\\gou-ide';
		st.workspaceName = 'gou-ide';
		st.workspaceFolders = ['F:\\syproject\\gou-ide'];
		return 'ok ws=' + st.workspaceRoot; })()`)
	waitJSF(wv, 300)
	runJobsF(wv)

	// ── 点击多 run 会话（>60 条）触发 switchConv ──
	sel := runJSF(wv, `(function(){
		var st = window.__state;
		var convs = st.conversations || [];
		for (var i = 0; i < convs.length; i++) {
			if (convs[i].msgCount > 60) return JSON.stringify({idx: i, id: convs[i].id, mc: convs[i].msgCount});
		}
		return JSON.stringify({idx: -1});
	})()`)
	fmt.Printf("[sfold] 选中候选: %s\n", sel)
	var idx int
	fmt.Sscanf(sel, `{"idx":%d`, &idx)
	if idx < 0 {
		fmt.Printf("[sfold] 未找到 >60 条会话，用第一个会话继续\n")
		idx = 0
	}
	fmt.Printf("[sfold] 点击会话 idx=%d\n", idx)
	runJSF(wv, fmt.Sprintf(`(function(){
		var items = document.querySelectorAll('.conv-item');
		if (!items[%d]) return 'no conv-item %d';
		items[%d].dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
		return 'clicked';
	})()`, idx, idx, idx))
	waitJSF(wv, 900)
	runJobsF(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsF(wv)

	before := runJSF(wv, foldStatsJS)
	fmt.Printf("[sfold] switchConv 后 BEFORE: %s\n", before)

	// ── 诊断1：初始加载空间——内容是否填满视口（区分 box 未找到 vs metrics 为 0） ──
	fmt.Printf("[sfold] 空间: %s\n", runJSF(wv, `(function(){
		var el = document.querySelector('.chat-messages');
		if (!el) return 'no el';
		return JSON.stringify({clientH: el.clientHeight, scrollH: el.scrollHeight, scrollTop: el.scrollTop, offsetH: el.offsetHeight, offsetW: el.offsetWidth, scrollW: el.scrollWidth, tag: el.tagName, cls: (el.className||'').slice(0,30)});
	})()`))

	// ── 诊断1c：JS 树 vs Go 树同一性判别（JS 侧打标记，Go 侧查） ──
	fmt.Printf("[sfold] 标记注入: %s\n", runJSF(wv, `(function(){
		var el = document.querySelector('.chat-messages');
		if (!el) return 'no el';
		el.setAttribute('data-probe-mark', 'yes');
		el.__probeProbe = 42;
		return 'marked tag=' + el.tagName + ' childCount=' + el.childElementCount;
	})()`))
	func() {
		doc := wv.Document()
		if doc == nil {
			fmt.Printf("[sfold] Go查标记: no doc\n")
			return
		}
		foundMark := false
		var walkDOM func(n dom.Node)
		walkDOM = func(n dom.Node) {
			if foundMark {
				return
			}
			if e, ok := n.(*dom.Element); ok {
				if e.GetAttribute("data-probe-mark") == "yes" {
					foundMark = true
					return
				}
			}
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				walkDOM(c)
			}
		}
		walkDOM(dom.Node(doc))
		fmt.Printf("[sfold] Go查标记: foundMark=%v\n", foundMark)
	}()

	// ── 诊断1b：Go 侧直接查 FindRenderBoxForNode(.chat-messages) + 滚动 metrics ──
	func() {
		rv := wv.RenderView()
		if rv == nil {
			fmt.Printf("[sfold] Go侧: no rv\n")
			return
		}
		doc := wv.Document()
		if doc == nil {
			fmt.Printf("[sfold] Go侧: no doc\n")
			return
		}
		var chatEl *dom.Element
		var walkDOM func(n dom.Node)
		walkDOM = func(n dom.Node) {
			if chatEl != nil {
				return
			}
			if e, ok := n.(*dom.Element); ok {
				if strings.Contains(e.GetAttribute("class"), "chat-messages") {
					chatEl = e
					return
				}
			}
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				walkDOM(c)
			}
		}
		walkDOM(dom.Node(doc))
		if chatEl == nil {
			fmt.Printf("[sfold] Go侧: .chat-messages DOM 未找到\n")
			return
		}
		box := rv.FindRenderBoxForNode(chatEl)
		if box == nil {
			fmt.Printf("[sfold] Go侧: FindRenderBoxForNode 返回 nil (class=%q)\n", chatEl.GetAttribute("class"))
			return
		}
		pb := box.PaddingBoxRect()
		tw, th := rv.BoxContentSize(box)
		vm := rendering.VerticalScrollbarMetrics(rv, box)
		fmt.Printf("[sfold] Go侧: box找到 pb=%.1fx%.1f content=%.1fx%.1f vscroll=%v\n", pb.Width, pb.Height, tw, th, vm.OK)
	}()

	// ── 诊断2b：真实 hit-test 路径——点击 folded-summary 中心，检查命中元素是否在折叠摘要内 ──
	func() {
		rv := wv.RenderView()
		if rv == nil {
			fmt.Printf("[sfold] hit-test: no rv\n")
			return
		}
		state := rv.LayoutState()
		var fsBox layout.Box
		var walkFindFS func(o rendering.RenderObject)
		walkFindFS = func(o rendering.RenderObject) {
			if o == nil || fsBox != nil {
				return
			}
			if el, ok := o.Node().(*dom.Element); ok {
				if strings.Contains(el.GetAttribute("class"), "folded-summary") {
					fsBox = o.LayoutBox()
					return
				}
			}
			for c := o.FirstChild(); c != nil; c = c.NextSibling() {
				walkFindFS(c)
			}
		}
		walkFindFS(rendering.RenderObject(rv))
		if fsBox == nil {
			fmt.Printf("[sfold] hit-test: no folded-summary box\n")
			return
		}
		g := state.GeometryForBox(fsBox)
		cx := g.Left() + g.BorderBoxWidth()/2
		cy := g.Top() + g.BorderBoxHeight()/2
		hit := rendering.HitTest(rv, cx, cy, "")
		if hit == nil {
			fmt.Printf("[sfold] hit-test: nil at (%.0f,%.0f)\n", cx, cy)
			return
		}
		cls := hit.GetAttribute("class")
		// 检查 hit 是否在 folded-summary 祖先链内
		inChain := false
		for n := dom.Node(hit); n != nil; n = n.ParentNode() {
			if e, ok := n.(*dom.Element); ok && strings.Contains(e.GetAttribute("class"), "folded-summary") {
				inChain = true
				break
			}
		}
		fmt.Printf("[sfold] hit-test: fs=(%.0f,%.0f) hitClass=%q inFSChain=%v tag=%s\n", cx, cy, cls, inChain, hit.LocalName())
		// 真实路径：对命中元素派发 click（冒泡），检查状态
		if inChain {
			hit.DispatchEvent(dom.NewMouseEvent(dom.EventClick, true, true, false))
			waitJSF(wv, 400)
			runJobsF(wv)
			wv.RebuildRenderTree()
			wv.EnsureLayout()
			runJobsF(wv)
			st := runJSF(wv, `(function(){
				var st = window.__state;
				var msgs = st.messagesByConv[st.currentConvId] || [];
				var idx = window.__diagIdx;
				var m = null;
				for (var i = 0; i < msgs.length; i++) { if (String(msgs[i]._idx) === String(idx)) { m = msgs[i]; break; } }
				return JSON.stringify({folded: m ? !!m._folded : null});
			})()`)
			fmt.Printf("[sfold] hit-test 派发后: %s\n", st)
		}
	}()

	// ── 诊断2：点击折叠摘要能否展开（用 data-idx 精确定位消息对象） ──
	fmt.Printf("[sfold] 点击前: %s\n", runJSF(wv, `(function(){
		var fs = document.querySelector('.folded-summary');
		if (!fs) return 'no fs';
		var item = fs.closest('.msg-item');
		var idx = item ? item.getAttribute('data-idx') : null;
		return JSON.stringify({fsExists: !!fs, idx: idx});
	})()`))
	runJSF(wv, `(function(){
		var fs = document.querySelector('.folded-summary');
		if (fs) {
			var item = fs.closest('.msg-item');
			window.__diagIdx = item ? item.getAttribute('data-idx') : null;
			fs.dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
			return 'clicked fs';
		}
		return 'no fs';
	})()`)
	waitJSF(wv, 500)
	runJobsF(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsF(wv)
	fmt.Printf("[sfold] 点击后: %s\n", runJSF(wv, `(function(){
		var st = window.__state;
		var msgs = st.messagesByConv[st.currentConvId] || [];
		var idx = window.__diagIdx;
		var m = null;
		for (var i = 0; i < msgs.length; i++) { if (String(msgs[i]._idx) === String(idx)) { m = msgs[i]; break; } }
		var fs = document.querySelector('.folded-summary');
		return JSON.stringify({idx: idx, found: !!m, folded: m ? !!m._folded : null, fsStillExists: !!fs, tlItems: document.querySelectorAll('.tl-item').length, thinkCollapsed: document.querySelectorAll('.tl-thinking-collapsed').length, tcHeader: document.querySelectorAll('.tl-tc-header').length});
	})()`))

	// ── 诊断2b：点击 thinking 折叠「思考…」→ 应展开（_collapsed false） ──
	runJSF(wv, `(function(){
		var tc = document.querySelector('.tl-thinking-collapsed');
		if (tc) { tc.dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true})); return 'clicked thinking'; }
		return 'no thinking';
	})()`)
	waitJSF(wv, 400)
	runJobsF(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsF(wv)
	fmt.Printf("[sfold] 点thinking后: %s\n", runJSF(wv, `(function(){
		var st = window.__state;
		var msgs = st.messagesByConv[st.currentConvId] || [];
		var idx = window.__diagIdx;
		var m = null;
		for (var i = 0; i < msgs.length; i++) { if (String(msgs[i]._idx) === String(idx)) { m = msgs[i]; break; } }
		var think = null;
		if (m) { for (var j = 0; j < m.segments.length; j++) { if (m.segments[j].type === 'thinking') { think = m.segments[j]; break; } } }
		return JSON.stringify({thinkCollapsed: think ? think._collapsed : null, thinkTextShown: !!document.querySelector('.tl-thinking-text'), thinkCollapsedLeft: document.querySelectorAll('.tl-thinking-collapsed').length});
	})()`))

	// ── 诊断2c：点击 tool_call header → 应展开详情（_expanded true） ──
	runJSF(wv, `(function(){
		var hd = document.querySelector('.tl-tc-header');
		if (hd) { hd.dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true})); return 'clicked tc'; }
		return 'no tc header';
	})()`)
	waitJSF(wv, 400)
	runJobsF(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsF(wv)
	fmt.Printf("[sfold] 点tool后: %s\n", runJSF(wv, `(function(){
		var st = window.__state;
		var msgs = st.messagesByConv[st.currentConvId] || [];
		var idx = window.__diagIdx;
		var m = null;
		for (var i = 0; i < msgs.length; i++) { if (String(msgs[i]._idx) === String(idx)) { m = msgs[i]; break; } }
		var tc = null;
		if (m) { for (var j = 0; j < m.segments.length; j++) { if (m.segments[j].type === 'tool_call') { tc = m.segments[j]; break; } } }
		return JSON.stringify({tcExpanded: tc ? !!tc._expanded : null, tcDetailShown: !!document.querySelector('.tl-tc-detail'), tcDetailCount: document.querySelectorAll('.tl-tc-detail').length});
	})()`))

	// ── 诊断3：引擎级几何——tl-tc-header / folded-summary / chat-messages 实际布局高度 ──
	fmt.Printf("[sfold] 引擎几何: %s\n", func() string {
		rv := wv.RenderView()
		if rv == nil {
			return "no render view"
		}
		state := rv.LayoutState()
		var sb strings.Builder
		count := 0
		var walk func(o rendering.RenderObject, depth int)
		walk = func(o rendering.RenderObject, depth int) {
			if o == nil || count >= 8 {
				return
			}
			if el, ok := o.Node().(*dom.Element); ok {
				cls := el.GetAttribute("class")
				if strings.Contains(cls, "tl-tc-header") || strings.Contains(cls, "tl-tc-param") || strings.Contains(cls, "tl-tc-name") || strings.Contains(cls, "folded-summary") {
					g := state.GeometryForBox(o.LayoutBox())
					ws := "?"
					if st := o.LayoutBox().Style(); st != nil {
						ws = fmt.Sprintf("%d", st.WhiteSpace)
					}
					fmt.Fprintf(&sb, "%s{h=%.1f,w=%.1f,ws=%s} ", cls, g.BorderBoxHeight(), g.BorderBoxWidth(), ws)
					count++
				}
			}
			for c := o.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c, depth+1)
			}
		}
		walk(rendering.RenderObject(rv), 0)
		return sb.String()
	}())

	// ── 诊断3b：扫描所有 tl-tc-param——找出折行实例并 dump 文本 ──
	fmt.Printf("[sfold] param扫描: %s\n", func() string {
		rv := wv.RenderView()
		if rv == nil {
			return "no rv"
		}
		state := rv.LayoutState()
		var sb strings.Builder
		var walk func(o rendering.RenderObject)
		walk = func(o rendering.RenderObject) {
			if o == nil {
				return
			}
			if el, ok := o.Node().(*dom.Element); ok {
				cls := el.GetAttribute("class")
				if strings.Contains(cls, "tl-tc-param") {
					g := state.GeometryForBox(o.LayoutBox())
					ws := "?"
					if st := o.LayoutBox().Style(); st != nil {
						ws = fmt.Sprintf("%d", st.WhiteSpace)
					}
					h := g.BorderBoxHeight()
					txt := el.TextContent()
					if len(txt) > 60 {
						txt = txt[:60]
					}
					txt = strings.ReplaceAll(txt, "\n", "\\n")
					fmt.Fprintf(&sb, "[h=%.1f ws=%s] %q | ", h, ws, txt)
				}
			}
			for c := o.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c)
			}
		}
		walk(rendering.RenderObject(rv))
		return sb.String()
	}())

	// DOM 层：展开的 thinking 文本 vs 折叠的"思考…"
	fmt.Printf("[sfold] DOM: %s\n", runJSF(wv, `(function(){
		var q = function(s){ return document.querySelectorAll(s).length; };
		return JSON.stringify({thinkText: q('.tl-thinking-text'), thinkCollapsed: q('.tl-thinking-collapsed'), foldedSummary: q('.folded-summary'), tcDetail: q('.tl-tc-detail')});
	})()`))

	// ── 派发 scroll 事件（等价用户上滚）→ loadMoreMessages 加载更早消息 ──
	for i := 0; i < 6; i++ {
		_ = runJSF(wv, `(function(){
			var el = document.querySelector('.chat-messages');
			if (!el) return 'no el';
			el.scrollTop = 0;
			el.dispatchEvent(new Event('scroll'));
			return 'scroll #' + i;
		})()`)
		waitJSF(wv, 700)
		runJobsF(wv)
		wv.RebuildRenderTree()
		wv.EnsureLayout()
		runJobsF(wv)
		cur := runJSF(wv, foldStatsJS)
		fmt.Printf("[sfold] 第 %d 次 scroll 后: %s\n", i+1, cur)
	}

	after := runJSF(wv, foldStatsJS)
	fmt.Printf("[sfold] scroll 派发后 AFTER: %s\n", after)
	fmt.Printf("[sfold] DOM(after): %s\n", runJSF(wv, `(function(){
		var q = function(s){ return document.querySelectorAll(s).length; };
		return JSON.stringify({thinkText: q('.tl-thinking-text'), thinkCollapsed: q('.tl-thinking-collapsed'), foldedSummary: q('.folded-summary'), tcDetail: q('.tl-tc-detail')});
	})()`))
	fmt.Printf("[sfold] ==== 期望：BEFORE/AFTER 均 thinkExpanded=0、toolExpanded=0、folded=assistant（全部收缩）====\n")
	fmt.Printf("[sfold] done\n")
}
