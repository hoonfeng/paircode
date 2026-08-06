// Command think_scroll_probe 复现「思考文本内部滚动条」三异常：
//   1. 滚动条 thumb 默认在底部（初始 scrollTop 非 0？）
//   2. 向下滚动后文字向下移动且超出背景（--bg-tertiary）区域
//   3. 滚动条 thumb 下移到裁切区域被裁切
// 流程：加载真实 dist + desktopbridge → 切换含 thinking 的会话 → 展开 thinking
//   → 注入长文本触发内部滚动条 → dump .tl-thinking-text 滚动状态与 thumb 几何
//   → 设置不同 scrollTop 渲染 PNG 对比。
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
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
	"wb-ui/style"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func setupLoadersTS(wv *webkit.WebView, distDir string) {
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

func waitJSTS(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func runJobsTS(wv *webkit.WebView) {
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

func runJSTS(wv *webkit.WebView, script string) string {
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
	setupLoadersTS(wv, distDir)
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
	waitJSTS(wv, 500)
	waitJSTS(wv, 500)
	runJobsTS(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	runJSTS(wv, `(function(){ var st = window.__state;
		if (!st) return 'no state';
		st.workspaceRoot = 'F:\\syproject\\gou-ide';
		st.workspaceName = 'gou-ide';
		st.workspaceFolders = ['F:\\syproject\\gou-ide'];
		return 'ok ws=' + st.workspaceRoot; })()`)
	waitJSTS(wv, 300)
	runJobsTS(wv)

	// 找含 thinking 的会话并切换
	sel := runJSTS(wv, `(function(){
		var st = window.__state;
		var convs = st.conversations || [];
		for (var i = 0; i < convs.length; i++) {
			if (convs[i].msgCount > 10) return JSON.stringify({idx: i, id: convs[i].id, mc: convs[i].msgCount});
		}
		return JSON.stringify({idx: -1});
	})()`)
	fmt.Printf("[ts] 选中候选: %s\n", sel)
	var idx int
	fmt.Sscanf(sel, `{"idx":%d`, &idx)
	if idx < 0 {
		idx = 0
	}
	runJSTS(wv, fmt.Sprintf(`(function(){
		var items = document.querySelectorAll('.conv-item');
		if (!items[%d]) return 'no conv-item %d';
		items[%d].dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
		return 'clicked';
	})()`, idx, idx, idx))
	waitJSTS(wv, 900)
	runJobsTS(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsTS(wv)

	// 展开所有 assistant 消息 + thinking 段，并给第一个 thinking 注入超长文本（>300px 触发滚动条）
	fmt.Printf("[ts] 注入: %s\n", runJSTS(wv, `(function(){
		var st = window.__state;
		var msgs = st.messagesByConv[st.currentConvId] || [];
		var expanded = 0, injected = 0;
		var long = '';
		for (var i = 0; i < 40; i++) { long += '这是第 ' + i + ' 行思考内容，用于撑高思考区域触发内部滚动条，需要足够长才会超过三百像素的高度限制，所以多写一点文字确保真的会溢出。\n'; }
		for (var j = 0; j < msgs.length; j++) {
			var m = msgs[j];
			if (m.role !== 'assistant') continue;
			m._folded = false; expanded++;
			var segs = m.segments || [];
			if (segs.length === 0) {
				segs.push({type: 'thinking', content: long, _collapsed: false});
				injected++;
			}
			var hasThink = false;
			for (var k = 0; k < segs.length; k++) {
				var g = segs[k];
				if (g.type === 'thinking') {
					hasThink = true;
					g._collapsed = false;
					if (injected === 0) { g.content = long; injected++; }
				}
			}
			if (!hasThink && injected === 0) {
				segs.push({type: 'thinking', content: long, _collapsed: false});
				injected++;
			}
			m.segments = segs;
		}
		st.messages = msgs;
		return JSON.stringify({expanded: expanded, injected: injected});
	})()`))
	waitJSTS(wv, 400)
	runJobsTS(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsTS(wv)

	// 注入后 DOM 状态
	fmt.Printf("[ts] DOM: %s\n", runJSTS(wv, `(function(){
		return JSON.stringify({
			thinkText: document.querySelectorAll('.tl-thinking-text').length,
			chatH: (function(){ var e = document.querySelector('.chat-messages'); return e ? {ch: e.clientHeight, sh: e.scrollHeight} : null; })()
		});
	})()`))

	// dump .tl-thinking-text 引擎级几何 + 滚动状态
	fmt.Printf("[ts] 引擎: %s\n", func() string {
		rv := wv.RenderView()
		if rv == nil {
			return "no rv"
		}
		doc := wv.Document()
		var target *dom.Element
		var walk func(n dom.Node)
		walk = func(n dom.Node) {
			if target != nil {
				return
			}
			if e, ok := n.(*dom.Element); ok {
				if strings.Contains(e.GetAttribute("class"), "tl-thinking-text") {
					target = e
					return
				}
			}
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c)
			}
		}
		walk(dom.Node(doc))
		if target == nil {
			return "no .tl-thinking-text"
		}
		box := rv.FindRenderBoxForNode(target)
		if box == nil {
			return "box nil"
		}
		state := rv.LayoutState()
		_ = state
		pb := box.PaddingBoxRect()
		tw, th := rv.BoxContentSize(box)
		sx, sy := rv.BoxScrollOffset(box)
		vm := rendering.VerticalScrollbarMetrics(rv, box)
		ws := "?"
		if st := box.Style(); st != nil {
			ws = fmt.Sprintf("ovfY=%d", st.OverflowY)
		}
		return fmt.Sprintf("pb=(%.0f,%.0f %.0fx%.0f) content=%.0fx%.0f scroll=(%.0f,%.0f) %s vscroll={OK:%v trackLen=%.0f thumbLen=%.0f maxScroll=%.0f viewH=%.0f totalH=%.0f}",
			pb.X, pb.Y, pb.Width, pb.Height, tw, th, sx, sy, ws,
			vm.OK, vm.TrackLen, vm.ThumbLen, vm.MaxScroll, vm.ViewLen, vm.TotalLen)
	}())

	// JS 侧几何（几何桥）
	fmt.Printf("[ts] JS几何: %s\n", runJSTS(wv, `(function(){
		var e = document.querySelector('.tl-thinking-text');
		if (!e) return 'no el';
		return JSON.stringify({clientH: e.clientHeight, scrollH: e.scrollHeight, scrollTop: e.scrollTop, offsetH: e.offsetHeight});
	})()`))

	// 渲染当前（scrollTop=0）
	renderTS(wv, "think_scroll_s0.png")

	// 设 scrollTop=100，渲染 + dump
	runJSTS(wv, `(function(){ var e = document.querySelector('.tl-thinking-text'); if (e) { e.scrollTop = 100; } return 'set'; })()`)
	waitJSTS(wv, 200)
	runJobsTS(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsTS(wv)
	fmt.Printf("[ts] 滚动后 JS几何: %s\n", runJSTS(wv, `(function(){
		var e = document.querySelector('.tl-thinking-text');
		if (!e) return 'no el';
		return JSON.stringify({clientH: e.clientHeight, scrollH: e.scrollHeight, scrollTop: e.scrollTop});
	})()`))
	fmt.Printf("[ts] 滚动后引擎: %s\n", func() string {
		rv := wv.RenderView()
		if rv == nil {
			return "no rv"
		}
		doc := wv.Document()
		var target *dom.Element
		var walk func(n dom.Node)
		walk = func(n dom.Node) {
			if target != nil {
				return
			}
			if e, ok := n.(*dom.Element); ok {
				if strings.Contains(e.GetAttribute("class"), "tl-thinking-text") {
					target = e
					return
				}
			}
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c)
			}
		}
		walk(dom.Node(doc))
		if target == nil {
			return "no .tl-thinking-text"
		}
		box := rv.FindRenderBoxForNode(target)
		if box == nil {
			return "box nil"
		}
		sx, sy := rv.BoxScrollOffset(box)
		vm := rendering.VerticalScrollbarMetrics(rv, box)
		return fmt.Sprintf("scroll=(%.0f,%.0f) vscroll={OK:%v trackLen=%.0f thumbLen=%.0f maxScroll=%.0f}", sx, sy, vm.OK, vm.TrackLen, vm.ThumbLen, vm.MaxScroll)
	}())
	renderTS(wv, "think_scroll_s100.png")

	// ── 场景2：外层 chat-messages 也滚动（嵌套滚动），检查思考滚动条是否被裁切 ──
	runJSTS(wv, `(function(){
		var el = document.querySelector('.chat-messages');
		if (!el) return 'no chat';
		el.scrollTop = 300;
		return 'chat scrollTop=300';
	})()`)
	waitJSTS(wv, 200)
	runJobsTS(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsTS(wv)
	fmt.Printf("[ts] 外层滚动后: %s\n", runJSTS(wv, `(function(){
		var c = document.querySelector('.chat-messages');
		var t = document.querySelector('.tl-thinking-text');
		return JSON.stringify({chatTop: c ? c.scrollTop : null, thinkTop: t ? t.scrollTop : null, thinkBox: t ? (function(){ var r = t.getBoundingClientRect(); return {top: r.top, bottom: r.bottom, height: r.height}; })() : null});
	})()`))
	renderTS(wv, "think_scroll_chat300.png")

	// ── 场景3：思考段内部滚动到最大（thumb 到最底部），检查 thumb 是否被裁切 ──
	runJSTS(wv, `(function(){
		var t = document.querySelector('.tl-thinking-text');
		if (t) { t.scrollTop = 999999; }
		return 'think scrollTop=max';
	})()`)
	waitJSTS(wv, 200)
	runJobsTS(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsTS(wv)
	fmt.Printf("[ts] 思考滚到底: %s\n", runJSTS(wv, `(function(){
		var t = document.querySelector('.tl-thinking-text');
		return JSON.stringify({thinkTop: t ? t.scrollTop : null, chatTop: (function(){ var c = document.querySelector('.chat-messages'); return c ? c.scrollTop : null; })()});
	})()`))
	renderTS(wv, "think_scroll_thinkmax.png")

	// ── 场景4：引擎级 thumb 位置诊断（thumbY 公式 + 容器底边 vs thumb 底边）──
	fmt.Printf("[ts] thumb诊断: %s\n", func() string {
		rv := wv.RenderView()
		if rv == nil {
			return "no rv"
		}
		doc := wv.Document()
		var target *dom.Element
		var walk func(n dom.Node)
		walk = func(n dom.Node) {
			if target != nil {
				return
			}
			if e, ok := n.(*dom.Element); ok {
				if strings.Contains(e.GetAttribute("class"), "tl-thinking-text") {
					target = e
					return
				}
			}
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c)
			}
		}
		walk(dom.Node(doc))
		if target == nil {
			return "no .tl-thinking-text"
		}
		box := rv.FindRenderBoxForNode(target)
		if box == nil {
			return "box nil"
		}
		vm := rendering.VerticalScrollbarMetrics(rv, box)
		if !vm.OK {
			return "no vscroll"
		}
		pb := box.PaddingBoxRect()
		_, sy := rv.BoxScrollOffset(box)
		// 模拟绘制端 thumbY：vy + syRatio*thumbTrackSpace + arrowOffset(非webkit)
		webkit := false
		if st := box.Style(); st != nil {
			webkit = st.GetProperty("-webkit-scrollbar-width") != "" ||
				st.GetProperty("-webkit-scrollbar-height") != "" ||
				st.GetProperty("-webkit-scrollbar-thumb-color") != "" ||
				st.GetProperty("-webkit-scrollbar-track-color") != "" ||
				st.GetProperty("-webkit-scrollbar-thumb-radius") != ""
			if webkit {
				for _, k := range []string{"-webkit-scrollbar-width", "-webkit-scrollbar-height", "-webkit-scrollbar-thumb-color", "-webkit-scrollbar-track-color", "-webkit-scrollbar-thumb-radius", "scrollbar-width", "scrollbar-color"} {
					if v := st.GetProperty(k); v != "" {
						fmt.Printf("[ts] thumb诊断: %s=%q\n", k, v)
					}
				}
			}
		}
		// chat 外层滚动（思考容器在 chat 内）：设备位置 = pb - chatScroll
		var chatSy float64
		var chatBox *rendering.RenderBox
		for p := target.ParentNode(); p != nil; p = p.ParentNode() {
			if e, ok := p.(*dom.Element); ok {
				if strings.Contains(e.GetAttribute("class"), "chat-messages") {
					chatBox = rv.FindRenderBoxForNode(e)
					if chatBox != nil {
						_, chatSy = rv.BoxScrollOffset(chatBox)
					}
					break
				}
			}
		}
		syRatio := sy / vm.MaxScroll
		if syRatio < 0 {
			syRatio = 0
		}
		if syRatio > 1 {
			syRatio = 1
		}
		thumbTrackSpace := vm.TrackLen - vm.ThumbLen
		thumbY := pb.Y + syRatio*thumbTrackSpace
		if !webkit {
			thumbY += 12 + 5
		}
		thumbBottom := thumbY + vm.ThumbLen
		return fmt.Sprintf("pb=(%.0f,%.0f %.0fx%.0f) chatSy=%.0f deviceY=%.0f webkitSB=%v sy=%.0f maxScroll=%.0f trackLen=%.0f thumbLen=%.0f thumbY=%.0f thumbBottom=%.0f",
			pb.X, pb.Y, pb.Width, pb.Height, chatSy, pb.Y-chatSy, webkit, sy, vm.MaxScroll, vm.TrackLen, vm.ThumbLen, thumbY, thumbBottom)
	}())

	// ── 场景4：引擎级 thumb 位置诊断（thumbY 公式 + 容器底边 vs thumb 底边）──
	fmt.Printf("[ts] thumb诊断: %s\n", func() string {
		rv := wv.RenderView()
		if rv == nil {
			return "no rv"
		}
		doc := wv.Document()
		// 收集所有 tl-thinking-text 的 pb + scroll，找首个（注入长文本的 maxScroll=538）
		var sb strings.Builder
		var walk func(n dom.Node)
		walk = func(n dom.Node) {
			if e, ok := n.(*dom.Element); ok {
				if strings.Contains(e.GetAttribute("class"), "tl-thinking-text") {
					box := rv.FindRenderBoxForNode(e)
					if box != nil {
						pb := box.PaddingBoxRect()
						_, sy := rv.BoxScrollOffset(box)
						vm := rendering.VerticalScrollbarMetrics(rv, box)
						fmt.Fprintf(&sb, "[pb=(%.0f,%.0f) sy=%.0f max=%.0f] ", pb.X, pb.Y, sy, vm.MaxScroll)
					}
				}
			}
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				walk(c)
			}
		}
		walk(dom.Node(doc))
		return sb.String()
	}())

	// ── 场景5：sticky「▲ 收起」按钮 — 思考段展开后按钮应吸在聊天视口底部 ──
	fmt.Printf("[ts] sticky按钮: %s\n", func() string {
		rv := wv.RenderView()
		if rv == nil {
			return "no rv"
		}
		doc := wv.Document()
		var target *dom.Element
		var walk2 func(n dom.Node)
		walk2 = func(n dom.Node) {
			if target != nil {
				return
			}
			if e, ok := n.(*dom.Element); ok {
				if strings.Contains(e.GetAttribute("class"), "tl-think-fold") {
					target = e
					return
				}
			}
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				walk2(c)
			}
		}
		walk2(dom.Node(doc))
		if target == nil {
			return "no .tl-think-fold"
		}
		box := rv.FindRenderBoxForNode(target)
		if box == nil {
			return "box nil"
		}
		st := box.Style()
		pos := "?"
		if st != nil {
			pos = fmt.Sprintf("position=%d sticky=%v", st.Position, box.IsStickyPositioned())
		}
		// 计算 sticky offset
		dx, dy := rendering.ComputeStickyOffsetForTest(box, rv)
		pb := box.PaddingBoxRect()
		// 找滚动容器视口底
		vpBottom := 0.0
		scrollSy := 0.0
		for p := target.ParentNode(); p != nil; p = p.ParentNode() {
			if e, ok := p.(*dom.Element); ok {
				cb := rv.FindRenderBoxForNode(e)
				if cb != nil {
					if cs := cb.Style(); cs != nil && (cs.OverflowY == style.OverflowAuto || cs.OverflowY == style.OverflowScroll) {
						pbb := cb.PaddingBoxRect()
						vpBottom = pbb.Y + pbb.Height
						_, scrollSy = rv.BoxScrollOffset(cb)
						break
					}
				}
			}
		}
		return fmt.Sprintf("pb=(%.0f,%.0f %.0fx%.0f) %s stickyOffset=(%.0f,%.0f) scrollContainerBottom=%.0f scrollSy=%.0f",
			pb.X, pb.Y, pb.Width, pb.Height, pos, dx, dy, vpBottom, scrollSy)
	}())

	fmt.Printf("[ts] done\n")
}

// renderTS renders the webview to a PNG file.
func renderTS(wv *webkit.WebView, name string) {
	if pngBytes, err := wv.Render(); err != nil {
		log.Printf("Render: %v", err)
		return
	} else {
		w, h := wv.Width(), wv.Height()
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				off := (y*w + x) * 4
				if off+3 < len(pngBytes) {
					img.SetRGBA(x, y, color.RGBA{R: pngBytes[off], G: pngBytes[off+1], B: pngBytes[off+2], A: pngBytes[off+3]})
				}
			}
		}
		out := filepath.Join("dev", "desktop_probe", name)
		f, err := os.Create(out)
		if err == nil {
			if err := png.Encode(f, img); err == nil {
				fmt.Printf("[ts] rendered %dx%d → %s\n", w, h, out)
			}
			f.Close()
		}
	}
}

// 保证 import 使用
var _ = style.OverflowAuto
