// Command tl_tc_probe verifies the tool-call collapsed header (.tl-tc-header)
// overflow bug in the REAL companion frontend rendered by wb-ui:
//   1. loads the real dist + desktopbridge (same startup path as cmd/desktop)
//   2. fetches conversations, picks the first one whose messages contain a
//      tool_call segment
//   3. clicks that conv-item through the real host.handleClick flow
//   4. dumps .tl-tc-header layout boxes + child text geometry to detect
//      right-edge overflow beyond the blue rounded rect
//   5. renders PNG for pixel verification
package main

import (
	"encoding/json"
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
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func processTl(wv *webkit.WebView) {
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.MarkRenderTreeDirty()
		}
	}
}

func setupLoadersTl(wv *webkit.WebView, distDir string) {
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

func waitJS(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
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
	setupLoadersTl(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	waitJS(wv, 400)
	waitJS(wv, 500)

	// 加载对话列表，找第一条含 tool_call 的对话
	iv, _ := wv.JSInterpreter().RunJS(`(function(){
		var out = {errs: (window.__state) ? 0 : 'no state'};
		var p = fetch('/api/conversations?workspace=' + encodeURIComponent('F:\\syproject\\gou-ide')).then(function(r){ return r.json(); }).then(function(list){
			out.count = Array.isArray(list) ? list.length : -1;
			out.titles = Array.isArray(list) ? list.slice(0,6).map(function(c){ return c.id + '|' + (c.title||'').slice(0,20); }) : [];
			return list;
		}).catch(function(e){ out.err = String(e && e.message || e).slice(0,200); });
		window.__tlp = p;
		return JSON.stringify(out);
	})()`)
	fmt.Printf("[tl] conv list start: %s\n", iv.ToString())
	waitJS(wv, 600)

	// 探测前 5 个对话消息的 seg.type 分布，找含 tool_call 的
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var out = {scan: []};
		window.__tlp.then(function(list){
			if (!Array.isArray(list)) { out.err = 'no list'; return; }
			var chain = Promise.resolve();
			list.slice(0, 5).forEach(function(c){
				chain = chain.then(function(){
					return fetch('/api/conversations/' + encodeURIComponent(c.id) + '/messages').then(function(r){ return r.json(); }).then(function(d){
						var msgs = Array.isArray(d) ? d : (d && d.messages) ? d.messages : [];
						var types = {};
						var hasTool = false;
						msgs.forEach(function(m){
							var segs = m.segments || [];
							segs.forEach(function(s){ types[s.type] = (types[s.type]||0)+1; if (s.type === 'tool_call') hasTool = true; });
						});
						out.scan.push({id: c.id, msgs: msgs.length, types: types, hasTool: hasTool});
					});
				});
			});
			return chain.then(function(){
				var pick = out.scan.find(function(s){ return s.hasTool; });
				out.pick = pick ? pick.id : '';
				out.done = true;
			});
		});
		return JSON.stringify(out);
	})()`)
	fmt.Printf("[tl] scan: %s\n", iv.ToString())
	waitJS(wv, 1200)
	iv, _ = wv.JSInterpreter().RunJS(`(function(){ var o = {}; o.scan = window.__tlp ? 'pending' : 'none'; try { o.pick = window.__tlPick || ''; } catch(e){} return JSON.stringify(o); })()`)
	res := iv.ToString()
	pickID := ""
	// 从上一轮输出提取 pick id（scan 结果在上一轮已打印）
	_ = res
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var out = {};
		window.__tlp.then(function(list){
			if (!Array.isArray(list)) { out.err = 'no list'; return; }
			var chain = Promise.resolve();
			var found = '';
			list.slice(0, 5).forEach(function(c){
				chain = chain.then(function(){
					return fetch('/api/conversations/' + encodeURIComponent(c.id) + '/messages').then(function(r){ return r.json(); }).then(function(d){
						if (found) return;
						var msgs = Array.isArray(d) ? d : (d && d.messages) ? d.messages : [];
						for (var i=0;i<msgs.length;i++){
							var segs = msgs[i].segments || [];
							for (var j=0;j<segs.length;j++){
								if (segs[j].type === 'tool_call') { found = c.id; window.__tlPick = c.id; return; }
							}
						}
					});
				});
			});
			chain.then(function(){ out.found = found || ''; out.done = true; });
		});
		return JSON.stringify(out);
	})()`)
	fmt.Printf("[tl] pick run2: %s\n", iv.ToString())
	waitJS(wv, 1500)
	iv, _ = wv.JSInterpreter().RunJS(`(function(){ return JSON.stringify({pick: window.__tlPick || '', convCount: (window.__state && window.__state.conversations) ? window.__state.conversations.length : -1}); })()`)
	pickRes := iv.ToString()
	fmt.Printf("[tl] picked: %s\n", pickRes)
	pickID = ""
	if strings.Contains(pickRes, "\"pick\":\"") {
		parts := strings.Split(pickRes, "\"pick\":\"")
		if len(parts) > 1 {
			pickID = strings.Split(parts[1], "\"")[0]
		}
	}
	if pickID == "" {
		log.Fatal("no conversation with tool_call found")
	}
	fmt.Printf("[tl] using conv %s\n", pickID)

	// 注入 conversations 到 state（触发左侧列表渲染），然后点击目标 conv-item
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var out = {};
		var st = window.__state;
		if (!st) { out.fatal = 'no state'; return JSON.stringify(out); }
		st.workspaceRoot = 'F:\\syproject\\gou-ide';
		st.workspaceName = 'gou-ide';
		st.workspaceFolders = ['F:\\syproject\\gou-ide'];
		window.__tlp.then(function(list){
			if (Array.isArray(list)) { st.conversations = list; out.injected = list.length; }
		});
		return JSON.stringify(out);
	})()`)
	waitJS(wv, 600)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}

	// 找到目标 conv-item 的位置并点击
	var target *dom.Element
	var tx, ty float64
	var findConv func(o rendering.RenderObject)
	findConv = func(o rendering.RenderObject) {
		if target != nil {
			return
		}
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			if strings.Contains(cn, "conv-item") {
				lb := o.LayoutBox()
				if lb != nil && rv.LayoutState() != nil {
					g := rv.LayoutState().GeometryForBox(lb)
					tx = g.Left() + 40
					ty = g.Top() + 6
					target = el
					return
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			findConv(c)
		}
	}
	findConv(rendering.RenderObject(rv))
	if target == nil {
		log.Fatal("no conv-item rendered")
	}
	fmt.Printf("[tl] click conv at (%.0f, %.0f)\n", tx, ty)

	elHit := rendering.HitTest(rv, tx, ty, "onclick")
	if elHit != nil {
		elHit.DispatchEvent(dom.NewMouseEvent(dom.EventClick, true, true, false))
	}
	for i := 0; i < 12; i++ {
		wv.JSInterpreter().RunJobs()
		processTl(wv)
		time.Sleep(15 * time.Millisecond)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 等待消息异步加载 + 渲染
	waitJS(wv, 1200)
	for i := 0; i < 12; i++ {
		wv.JSInterpreter().RunJobs()
		processTl(wv)
		time.Sleep(20 * time.Millisecond)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 点击后状态诊断
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var st = window.__state;
		var cur = st ? st.currentConvId : '';
		var msgs = (st && st.messagesByConv && st.messagesByConv[cur]) || [];
		var segTypes = [];
		msgs.forEach(function(m){ (m.segments||[]).forEach(function(s){ segTypes.push(s.type); }); });
		var out = {
			currentConvId: cur,
			msgCount: msgs.length,
			segTypes: segTypes.slice(0, 20),
			folded: msgs.filter(function(m){ return m._folded; }).length,
			msgItems: document.querySelectorAll('.msg-item').length,
			tlTc: document.querySelectorAll('.tl-tc-header').length,
			foldedSummary: document.querySelectorAll('.folded-summary').length,
			tlThinking: document.querySelectorAll('.tl-think-body').length,
			convItems: document.querySelectorAll('.conv-item').length
		};
		return JSON.stringify(out);
	})()`)
	fmt.Printf("[tl] after-click state: %s\n", iv.ToString())

	// ★ 展开消息（_folded=false + tool_call 收缩态 _expanded=false），露出蓝色 tl-tc-header
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var st = window.__state;
		var cur = st.currentConvId;
		var msgs = st.messagesByConv[cur] || [];
		var unfolded = 0, tc = 0;
		msgs.forEach(function(m){
			if (m.role === 'assistant') { m._folded = false; unfolded++; }
			(m.segments||[]).forEach(function(s){
				if (s.type === 'tool_call') { s._expanded = false; tc++; }
			});
		});
		st.messages = msgs;
		return JSON.stringify({unfolded: unfolded, tc: tc});
	})()`)
	fmt.Printf("[tl] unfold: %s\n", iv.ToString())

	// ★ 注入长文本 tool_call（模拟 run_command 长命令 + 长输出首行），复现文本溢出
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var st = window.__state;
		var cur = st.currentConvId;
		var msgs = st.messagesByConv[cur] || [];
		var injected = 0;
		msgs.forEach(function(m){
			(m.segments||[]).forEach(function(s){
				if (s.type === 'tool_call' && injected === 0) {
					s.name = 'run_command';
					s.argsRaw = JSON.stringify({command: 'go run ./cmd/companion --with-a-very-long-flag-name ' + 'and-more-flags '.repeat(6)});
					s.result = 'output-line-1: ' + 'this-is-a-very-long-output-line-that-should-definitely-overflow-the-blue-rect '.repeat(2) + '|end';
					injected++;
				}
			});
		});
		st.messages = msgs;
		return JSON.stringify({injected: injected});
	})()`)
	fmt.Printf("[tl] inject long text: %s\n", iv.ToString())
	for i := 0; i < 10; i++ {
		wv.JSInterpreter().RunJobs()
		processTl(wv)
		time.Sleep(20 * time.Millisecond)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		return JSON.stringify({
			msgItems: document.querySelectorAll('.msg-item').length,
			tlTc: document.querySelectorAll('.tl-tc-header').length,
			foldedSummary: document.querySelectorAll('.folded-summary').length
		});
	})()`)
	fmt.Printf("[tl] after unfold DOM: %s\n", iv.ToString())

	// dump tl-tc-header 几何
	rv = wv.RenderView()
	if rv == nil {
		log.Fatal("no render view after click")
	}
	type boxInfo struct {
		Class  string  `json:"class"`
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		W      float64 `json:"w"`
		H      float64 `json:"h"`
		Right  float64 `json:"right"`
		MaxSub float64 `json:"maxSubRight"`
		Over   float64 `json:"overflow"`
		Texts  []string `json:"texts,omitempty"`
	}
	var headers []boxInfo
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			if strings.Contains(cn, "tl-tc-header") {
				lb := o.LayoutBox()
				if lb != nil && rv.LayoutState() != nil {
					g := rv.LayoutState().GeometryForBox(lb)
					bi := boxInfo{Class: cn, X: g.Left(), Y: g.Top(), W: g.BorderBoxWidth(), H: g.BorderBoxHeight(), Right: g.Left() + g.BorderBoxWidth()}
					// 收集 header 内所有文本右缘
					var maxSub float64
					var texts []string
					var walkSub func(s rendering.RenderObject)
					walkSub = func(s rendering.RenderObject) {
						if rt, ok := s.(*rendering.RenderText); ok {
							t := strings.TrimSpace(rt.Text())
							if t != "" {
								sb := rt.LayoutBox()
								if sb != nil && rv.LayoutState() != nil {
									sg := rv.LayoutState().GeometryForBox(sb)
									sgRight := sg.Left() + sg.BorderBoxWidth()
									if sgRight > maxSub {
										maxSub = sgRight
									}
									if len(texts) < 6 {
										if len(t) > 30 {
											t = t[:30] + "…"
										}
										texts = append(texts, fmt.Sprintf("%.0f:%s", sg.Left(), t))
									}
								}
							}
						}
						for c := s.FirstChild(); c != nil; c = c.NextSibling() {
							walkSub(c)
						}
					}
					walkSub(o)
					bi.MaxSub = maxSub
					bi.Over = maxSub - bi.Right
					bi.Texts = texts
					headers = append(headers, bi)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	fmt.Printf("[tl] tl-tc-header count=%d\n", len(headers))
	for _, h := range headers {
		status := "OK"
		if h.Over > 1 {
			status = "OVERFLOW"
		}
		fmt.Printf("[tl]   %s cls=%.30s rect=(%.0f,%.0f %.0fx%.0f) right=%.0f maxSub=%.0f over=%.1f texts=%v\n",
			status, h.Class, h.X, h.Y, h.W, h.H, h.Right, h.MaxSub, h.Over, h.Texts)
	}

	// dump .tl-tc-param/.tl-tc-summary rect（验证 flex 子项是否收缩）
	{
		var sub []string
		var walk2 func(s rendering.RenderObject)
		walk2 = func(s rendering.RenderObject) {
			if el, ok := s.Node().(*dom.Element); ok {
				if cn := el.GetAttribute("class"); strings.Contains(cn, "tl-tc-param") || strings.Contains(cn, "tl-tc-summary") || strings.Contains(cn, "tl-tc-name") {
					if lb := s.LayoutBox(); lb != nil && rv.LayoutState() != nil {
						g := rv.LayoutState().GeometryForBox(lb)
						sub = append(sub, fmt.Sprintf("%s rect=(%.0f,%.0f %.0fx%.0f) cw=%.1f padL=%.1f", cn, g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(), g.ContentWidth(), g.PaddingLeft()))
					}
				}
			}
			for c := s.FirstChild(); c != nil; c = c.NextSibling() {
				walk2(c)
			}
		}
		walk2(rendering.RenderObject(rv))
		for _, s := range sub {
			fmt.Printf("[tl]   SUB %s\n", s)
		}
	}

	// 消息区渲染截图（raw RGBA → PNG）
	if pngBytes, err := wv.Render(); err != nil {
		log.Printf("Render: %v", err)
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
		out := filepath.Join("dev", "desktop_probe", "tl_tc_render.png")
		f, err := os.Create(out)
		if err == nil {
			if err := png.Encode(f, img); err == nil {
				fmt.Printf("[tl] rendered %dx%d → %s\n", w, h, out)
			}
			f.Close()
		}
	}

	j, _ := json.MarshalIndent(headers, "", " ")
	os.WriteFile(filepath.Join("dev", "desktop_probe", "tl_tc_boxes.json"), j, 0o644)
	fmt.Printf("[tl] done\n")
}

// dumpSUB placeholder
