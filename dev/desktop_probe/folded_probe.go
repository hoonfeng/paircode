// Command folded_probe verifies the folded-summary row ("完成摘要" title +
// desc) CJK soft-wrap behavior in the REAL companion frontend rendered by
// wb-ui, at both wide (1280) and narrow (600) window widths.
//
// The browser wraps the CJK title "完成摘要" per character when the flex row
// runs out of space (each CJK ideograph is a soft-wrap opportunity). wb-ui
// must do the same instead of laying the whole run on one line and
// overlapping the following content.
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

func waitJS(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func runJobs(wv *webkit.WebView) {
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

type fsInfo struct {
	Class string  `json:"class"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	W     float64 `json:"w"`
	H     float64 `json:"h"`
	Right float64 `json:"right"`
	Lines int     `json:"lines"` // number of text lines in this subtree
	Texts []string
	Over  float64 `json:"overflow"`
}

func dumpFolded(wv *webkit.WebView, tag string) {
	rv := wv.RenderView()
	if rv == nil {
		log.Fatalf("no render view at %s", tag)
	}
	var out []fsInfo
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			if strings.Contains(cn, "folded-summary") || strings.Contains(cn, "folded-title") || strings.Contains(cn, "folded-desc") {
				lb := o.LayoutBox()
				if lb != nil && rv.LayoutState() != nil {
					g := rv.LayoutState().GeometryForBox(lb)
					fi := fsInfo{Class: cn, X: g.Left(), Y: g.Top(), W: g.BorderBoxWidth(), H: g.BorderBoxHeight(), Right: g.Left() + g.BorderBoxWidth()}
					var texts []string
					var lines int
					var maxRight float64
					var walkT func(s rendering.RenderObject)
					walkT = func(s rendering.RenderObject) {
						if rt, ok := s.(*rendering.RenderText); ok {
							t := strings.TrimSpace(rt.Text())
							if t != "" {
								sb := rt.LayoutBox()
								if sb != nil && rv.LayoutState() != nil {
									sg := rv.LayoutState().GeometryForBox(sb)
									if sg.Left()+sg.BorderBoxWidth() > maxRight {
										maxRight = sg.Left() + sg.BorderBoxWidth()
									}
									lines++
									if len(texts) < 8 {
										if len(t) > 24 {
											t = t[:24] + "…"
										}
										texts = append(texts, fmt.Sprintf("(%.0f,%.0f)%s", sg.Left(), sg.Top(), t))
									}
								}
							}
						}
						for c := s.FirstChild(); c != nil; c = c.NextSibling() {
							walkT(c)
						}
					}
					walkT(o)
					fi.Lines = lines
					fi.Texts = texts
					fi.Over = maxRight - fi.Right
					out = append(out, fi)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rendering.RenderObject(rv))
	fmt.Printf("[folded] == %s == count=%d\n", tag, len(out))
	for _, fi := range out {
		status := "OK"
		if fi.Over > 1 {
			status = "OVERFLOW"
		}
		fmt.Printf("[folded]   %s cls=%.20s rect=(%.0f,%.0f %.0fx%.0f) right=%.0f lines=%d over=%.1f texts=%v\n",
			status, fi.Class, fi.X, fi.Y, fi.W, fi.H, fi.Right, fi.Lines, fi.Over, fi.Texts)
	}
}

func renderPNG(wv *webkit.WebView, path string) {
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
		f, _ := os.Create(path)
		defer f.Close()
		_ = png.Encode(f, img)
		fmt.Printf("[folded] rendered %dx%d -> %s\n", w, h, path)
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
	setupLoadersF(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	// ★ 独立面板模式：desktopbridge 已恢复完整 IDE，本测试程序自行注入
	//   panel-only 标志（在页面脚本执行前），只加载右侧面板做折叠条验证。
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		rt.RunJS(`window.__DESKTOP_PANEL_MODE__ = true;`)
	}
	// 注入 data-v 属性探针：记录 setAttribute/removeAttribute 对 data-v-* 的调用。
	// 探针 patch Element.prototype.setAttribute（标准 DOM 设计，方法在 prototype 上），
	// 所有实例（HTML/SVG/Element）经原型链共享，hook 可捕获 Vue 的 data-v 属性写入。
	probeJS := `<script>
(function(){
  window.__attrLog = [];
  window.__probeLoaded = true;
  window.__probeErr = [];
  try {
    var orig = Element.prototype.setAttribute;
    window.__origSA = String(orig);
    Element.prototype.setAttribute = function(n, v){
      if (String(n).indexOf('data-v-') === 0) window.__attrLog.push('SET ' + this.tagName + '.' + String(this.className||'') + ' ' + n);
      return orig.apply(this, arguments);
    };
  } catch(e) { window.__probeErr.push('SA:' + String(e && e.message || e)); }
  try {
    var origR = Element.prototype.removeAttribute;
    Element.prototype.removeAttribute = function(n){
      if (String(n).indexOf('data-v-') === 0) window.__attrLog.push('DEL ' + this.tagName + '.' + String(this.className||'') + ' ' + n);
      return origR.apply(this, arguments);
    };
  } catch(e) { window.__probeErr.push('RA:' + String(e && e.message || e)); }
})();
</script>`
	htmlStr := strings.Replace(string(htmlData), "<script type=\"module\"", probeJS+"\n<script type=\"module\"", 1)
	wv.LoadHTML(htmlStr)
	waitJS(wv, 400)
	waitJS(wv, 600)

	// 找第一条含 tool_call 的对话
	iv, _ := wv.JSInterpreter().RunJS(`(function(){
		var out = {};
		var p = fetch('/api/conversations?workspace=' + encodeURIComponent('F:\\syproject\\gou-ide')).then(function(r){ return r.json(); }).then(function(list){
			window.__flp = list;
			out.count = Array.isArray(list) ? list.length : -1;
			var chain = Promise.resolve(); var found = '';
			(Array.isArray(list) ? list : []).slice(0, 6).forEach(function(c){
				chain = chain.then(function(){
					return fetch('/api/conversations/' + encodeURIComponent(c.id) + '/messages').then(function(r){ return r.json(); }).then(function(d){
						if (found) return;
						var msgs = Array.isArray(d) ? d : (d && d.messages) ? d.messages : [];
						for (var i=0;i<msgs.length;i++){
							var segs = msgs[i].segments || [];
							for (var j=0;j<segs.length;j++){
								if (segs[j].type === 'tool_call') { found = c.id; window.__fPick = c.id; return; }
							}
						}
					});
				});
			});
			return chain.then(function(){ out.found = found || ''; });
		}).catch(function(e){ out.err = String(e && e.message || e).slice(0,200); });
		return JSON.stringify(out);
	})()`)
	waitJS(wv, 2000)
	iv, _ = wv.JSInterpreter().RunJS(`(function(){ return JSON.stringify({pick: window.__fPick || '', list: Array.isArray(window.__flp) ? window.__flp.length : -1}); })()`)
	res := iv.ToString()
	fmt.Printf("[folded] pick: %s\n", res)
	pickID := ""
	if strings.Contains(res, "\"pick\":\"") {
		parts := strings.Split(res, "\"pick\":\"")
		if len(parts) > 1 {
			pickID = strings.Split(parts[1], "\"")[0]
		}
	}
	if pickID == "" {
		log.Fatal("no conversation with tool_call found")
	}

	// 注入 conversations + 点击 conv-item（保持折叠态！不 unfold）
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var st = window.__state;
		if (!st) { return JSON.stringify({fatal: 'no state'}); }
		st.workspaceRoot = 'F:\\syproject\\gou-ide';
		st.workspaceName = 'gou-ide';
		st.workspaceFolders = ['F:\\syproject\\gou-ide'];
		var list = window.__flp;
		if (Array.isArray(list)) { st.conversations = list; }
		return JSON.stringify({ok: 1, convs: Array.isArray(list) ? list.length : -1});
	})()`)
	waitJS(wv, 500)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 找 conv-item 并点击
	rv := wv.RenderView()
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
	fmt.Printf("[folded] click conv at (%.0f, %.0f)\n", tx, ty)
	elHit := rendering.HitTest(rv, tx, ty, "onclick")
	if elHit != nil {
		elHit.DispatchEvent(dom.NewMouseEvent(dom.EventClick, true, true, false))
	}
	runJobs(wv)
	waitJS(wv, 1200)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var st = window.__state;
		var cur = st ? st.currentConvId : '';
		var msgs = (st && st.messagesByConv && st.messagesByConv[cur]) || [];
		return JSON.stringify({
			cur: cur,
			foldedSummaries: document.querySelectorAll('.folded-summary').length,
			msgItems: document.querySelectorAll('.msg-item').length,
			totalMsgs: msgs.length
		});
	})()`)
	fmt.Printf("[folded] state: %s\n", iv.ToString())

	// ── 诊断：folded-desc 的真实 computed style + data-v 属性 ──
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var out = {n: 0};
		var all = document.querySelectorAll('.folded-desc');
		out.n = all.length;
		out.items = [];
		for (var i=0;i<all.length && i<3;i++){
			var d = all[i];
			var it = {cls: d.getAttribute('class')};
			it.dataV = d.getAttribute('data-v-cdb19c8e') === '';
			var cs = window.getComputedStyle ? getComputedStyle(d) : null;
			if (cs) {
				it.ws = cs.whiteSpace;
				it.ov = cs.overflow;
				it.ovX = cs.overflowX;
				it.to = cs.textOverflow;
				it.color = cs.color;
			}
			out.items.push(it);
		}
		return JSON.stringify(out);
	})()`)
	fmt.Printf("[folded] desc computed style: %s\n", iv.ToString())

	// ── 诊断 1.5：data-v 属性选择器匹配 + style 收集情况 ──
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var out = {};
		try { out.qs = document.querySelectorAll('.folded-desc[data-v-cdb19c8e]').length; } catch(e){ out.qsErr = String(e); }
		try {
			var ss = document.querySelectorAll('style');
			var arr = [];
			for (var i=0;i<ss.length;i++){ arr.push({len: (ss[i].textContent||'').length, head: (ss[i].textContent||'').slice(0,50)}); }
			out.styles = arr;
		} catch(e){ out.stylesErr = String(e); }
		try {
			var d = document.querySelector('.folded-desc');
			var a = [];
			if (d) {
				for (var j=0;j<d.attributes.length;j++){ a.push(String(d.attributes[j].name) + '=' + String(d.attributes[j].value)); }
			}
			out.attrs = a;
		} catch(e){ out.attrsErr = String(e); }
		try {
			var cs2 = window.getComputedStyle ? getComputedStyle(document.querySelector('.folded-desc')) : null;
			out.csType = typeof cs2;
			if (cs2) { out.csKeys = Object.keys(cs2).slice(0,10); out.csWs = cs2.whiteSpace; }
		} catch(e){ out.csErr = String(e); }
		try {
			var d2 = document.querySelector('.folded-desc');
			out.matchesClass = d2 ? d2.matches('.folded-desc') : 'no-el';
			out.matchesAttr = d2 ? d2.matches('[data-v-cdb19c8e]') : 'no-el';
			out.matchesBoth = d2 ? d2.matches('.folded-desc[data-v-cdb19c8e]') : 'no-el';
			out.hasAttr = d2 ? d2.hasAttribute('data-v-cdb19c8e') : 'no-el';
			out.getAttr = d2 ? d2.getAttribute('data-v-cdb19c8e') : 'no-el';
			out.attrNames = d2 ? (d2.getAttributeNames ? d2.getAttributeNames() : 'no-method') : 'no-el';
		} catch(e){ out.mErr = String(e); }
		try {
			var ids = ['data-v-cdb19c8e','data-v-245f3b5c','data-v-90f89230','data-v-b0465242'];
			var cnt = {};
			for (var k=0;k<ids.length;k++){ cnt[ids[k]] = document.querySelectorAll('['+ids[k]+']').length; }
			out.scopeCnt = cnt;
			out.scopeUnder = document.querySelectorAll('[data-v-undefined]').length;
		} catch(e){ out.scopeErr = String(e); }
		return JSON.stringify(out);
	})()`)
	fmt.Printf("[folded] match-diag: %s\n", iv.ToString())

	// ── 诊断 1.7：data-v 属性 set 日志（ChatView scopeId 是否被写入 DOM）──
	// hook 活性检测：探针 patch Element.prototype.setAttribute 后，手动调用
	// 应被捕获；同时检查 Vue 元素的方法来源（自有 vs 原型链）。
	iv, _ = wv.JSInterpreter().RunJS(`(function(){
		var log = window.__attrLog || [];
		var ids = ['data-v-cdb19c8e','data-v-245f3b5c','data-v-90f89230','data-v-b0465242'];
		var out = {probeLoaded: window.__probeLoaded === true, total: log.length, probeErr: window.__probeErr || []};
		out.hookAlive = /data-v-hooktest|attrLog/.test(String(Element.prototype.setAttribute));
		out.origSAkind = String(window.__origSA).slice(0, 60);
		out.hookSAkind = String(Element.prototype.setAttribute).slice(0, 60);
		// hook 活性：直接调一次，验证探针 hook 是否仍在 prototype 上生效
		var t = document.createElement('probe-check');
		t.setAttribute('data-v-hooktest', '');
		out.totalAfterManual = (window.__attrLog || []).length;
		out.manualTail = (window.__attrLog || []).slice(-2);
		// Vue 元素（带 data-v-90f89230）的方法来源
		try {
			var ve = document.querySelector('[data-v-90f89230]');
			if (ve) {
				out.vueEl = {
					tag: ve.tagName,
					ownSA: ve.hasOwnProperty('setAttribute'),
					saIsProto: ve.setAttribute === Element.prototype.setAttribute,
					protoType: typeof Element.prototype.setAttribute
				};
			} else {
				out.vueEl = null;
			}
		} catch(e) { out.vueElErr = String(e); }
		out.byId = {};
		ids.forEach(function(id){ out.byId[id] = log.filter(function(e){ return e.indexOf(id) >= 0; }).slice(0, 8); });
		out.sample = log.slice(0, 30);
		return JSON.stringify(out);
	})()`)
	fmt.Printf("[folded] attrLog: %s\n", iv.ToString())

	// ── 诊断 2：渲染对象 Style() 检查 folded-desc/folded-title ──
	{
		var walkEl func(o rendering.RenderObject)
		walkEl = func(o rendering.RenderObject) {
			if el, ok := o.Node().(*dom.Element); ok {
				cn := el.GetAttribute("class")
				if strings.Contains(cn, "folded-desc") || strings.Contains(cn, "folded-title") {
					cs := o.Style()
					to := "nil-style"
					if cs != nil {
						to = fmt.Sprintf("ws=%v ovX=%v ovY=%v textOv=%v disp=%v",
							cs.WhiteSpace, cs.OverflowX, cs.OverflowY, cs.TextOverflow, cs.Display)
					}
					fmt.Printf("[folded] ro-style %s (%T): %s\n", cn, o, to)
				}
			}
			for c := o.FirstChild(); c != nil; c = c.NextSibling() {
				walkEl(c)
			}
		}
		walkEl(rendering.RenderObject(rv))
	}

	// ── 诊断 3：folded-summary 子树渲染树 dump ──
	{
		var dumpSub func(o rendering.RenderObject, depth int)
		dumpSub = func(o rendering.RenderObject, depth int) {
			cn := ""
			if el, ok := o.Node().(*dom.Element); ok {
				cn = el.GetAttribute("class")
			} else if tn, ok := o.Node().(*dom.Text); ok {
				t := tn.Data()
				if len(t) > 20 {
					t = t[:20] + "…"
				}
				cn = "TEXT(" + t + ")"
			}
			lbInfo := ""
			if lb := o.LayoutBox(); lb != nil && rv.LayoutState() != nil {
				g := rv.LayoutState().GeometryForBox(lb)
				lbInfo = fmt.Sprintf(" rect=(%.0f,%.0f %.0fx%.0f)", g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight())
			}
			fmt.Printf("[folded] %*s%s %q%s\n", depth*2, "", fmt.Sprintf("%T", o), cn, lbInfo)
			for c := o.FirstChild(); c != nil; c = c.NextSibling() {
				dumpSub(c, depth+1)
			}
		}
		dumpSub(rendering.RenderObject(rv), 0)
	}

	// 1280 宽度 dump + 截图
	dumpFolded(wv, "w1280")
	renderPNG(wv, filepath.Join(wd, "dev", "desktop_probe", "folded_w1280.png"))

	// 缩窄到 600 模拟浏览器窄窗口
	wv.Resize(600, 800)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	dumpFolded(wv, "w600")
	renderPNG(wv, filepath.Join(wd, "dev", "desktop_probe", "folded_w600.png"))

	fmt.Printf("[folded] done\n")
}
