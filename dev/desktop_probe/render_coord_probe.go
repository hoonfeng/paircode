// Command render_coord_probe 打开设置面板后，遍历渲染树输出关键容器
// （settings-modal/modal-header/modal-body/settings-tabs/settings-body/
//  tab button/label/input）在渲染树 frame（相对坐标）与布局树
// BoxGeometry 中的位置，以及父链，用于定位「DOM rect 与像素渲染错位」。
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

func runJobs(wv *webkit.WebView) {
	for i := 0; i < 15; i++ {
		wv.JSInterpreter().RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func js(wv *webkit.WebView, code string) string {
	iv, err := wv.JSInterpreter().RunJS(code)
	if err != nil {
		return "[ERR] " + err.Error()
	}
	return iv.ToString()
}

func clsOf(el *dom.Element) string { return el.GetAttribute("class") }

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
	layout.FontMetricsFunc = func(family string, size float64, weight int, style string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	wv := webkit.NewWebView()
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
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
	wv.SetConsoleLogger(&jsc.BufferLogger{})
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	for i := 0; i < 4; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
		time.Sleep(600 * time.Millisecond)
		runJobs(wv)
	}
	// 打开设置面板
	fmt.Println(js(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		if (btn) { var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev); }
		return 'clicked: ' + !!btn;
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}
	state := rv.LayoutState()

	// 目标 class 过滤
	want := map[string]bool{
		"modal-overlay": true, "modal-content": true, "settings-modal": true,
		"modal-header": true, "modal-body": true, "settings-tabs": true,
		"settings-body": true, "setting-row": true, "setting-group": true,
		"group-title": true, "activity-bar": true,
	}

	desc := func(o rendering.RenderObject) string {
		if o == nil {
			return "<nil>"
		}
		nn := o.Node()
		if nn == nil {
			return "?"
		}
		if el, ok := nn.(*dom.Element); ok {
			c := el.GetAttribute("class")
			t := el.LocalName()
			txt := ""
			if t == "button" || t == "h2" || t == "label" {
				txt = "«" + strings.TrimSpace(el.TextContent())[:min(12, len(strings.TrimSpace(el.TextContent())))] + "»"
			}
			return fmt.Sprintf("%s.%s%s", t, c, txt)
		}
		return nn.NodeName()
	}

	var printTree func(o rendering.RenderObject, depth int)
	printTree = func(o rendering.RenderObject, depth int) {
		if o == nil || depth > 60 {
			return
		}
		if nn := o.Node(); nn != nil {
			if el, ok := nn.(*dom.Element); ok {
				c := el.GetAttribute("class")
				matched := false
				for w := range want {
					if strings.Contains(c, w) {
						matched = true
						break
					}
				}
				isTabBtn := strings.Contains(c, "settings-tabs") && el.LocalName() == "button"
				if matched || isTabBtn || el.LocalName() == "label" {
					var absX, absY float64 = -1, -1
					var fw, fh float64 = -1, -1
					var ox, oy float64 = -1, -1
					if rb, ok := o.(*rendering.RenderBox); ok {
						absX, absY = rb.AbsoluteX(), rb.AbsoluteY()
						fw, fh = rb.Width(), rb.Height()
						ox, oy = rb.X(), rb.Y()
					}
					gstr := ""
					if lb := o.LayoutBox(); lb != nil && state != nil {
						g := state.GeometryForBox(lb)
						gstr = fmt.Sprintf(" geom=(%.0f,%.0f %dx%d)", g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight())
					}
					txt := ""
					if el.LocalName() == "button" || el.LocalName() == "label" || el.LocalName() == "h2" || el.LocalName() == "div" {
						t := strings.TrimSpace(el.TextContent())
						if t != "" && len(t) < 20 {
							txt = " «" + t + "»"
						}
					}
					fmt.Printf("%*s[%s] frame=(%.0f,%.0f %dx%.0f) abs=(%.0f,%.0f)%s%s\n",
						depth*2, "", desc(o), ox, oy, fw, fh, absX, absY, gstr, txt)
				}
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			printTree(c, depth+1)
		}
	}
	printTree(rv, 0)

	// DOM getBoundingClientRect 对照
	fmt.Println("\n[DOM rect]")
	fmt.Println(js(wv, `(function(){
		var out = [];
		var sels = ['.modal-overlay','.modal-content.settings-modal','.modal-header','.modal-body','.settings-tabs','.settings-body','.settings-tabs button','.setting-row label','.setting-row input','.setting-row select'];
		for (var i=0;i<sels.length;i++) {
			var els = document.querySelectorAll(sels[i]);
			for (var j=0;j<els.length && j<3;j++) {
				var el = els[j];
				var r = el.getBoundingClientRect();
				var t = (el.textContent||'').trim().slice(0,10);
				out.push({sel:sels[i], idx:j, t:t, x:Math.round(r.left), y:Math.round(r.top), w:Math.round(r.width), h:Math.round(r.height)});
			}
		}
		return JSON.stringify(out);
	})()`))

	// ── 2. 打开工具配置弹窗，找 y∈[500,540] 的亮条元素 ──
	fmt.Println("\n[toolcfg] 打开弹窗:")
	fmt.Println(js(wv, `(function(){
		var st = window.__state;
		if (st && !st.rightPanelVisible) { st.rightPanelVisible = true; }
		return 'rightPanel: ' + (st ? !!st.rightPanelVisible : 'no-state');
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	fmt.Println(js(wv, `(function(){
		var btn = document.querySelector('.obtn-review-config');
		if (btn) { var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev); }
		return 'clicked: ' + !!btn;
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	// 找 y 在 495-545 的所有元素
	fmt.Println("[toolcfg] y∈[495,545] 元素:")
	fmt.Println(js(wv, `(function(){
		var out = [];
		var all = document.querySelectorAll('.tool-config-popover *');
		for (var i=0;i<all.length;i++) {
			var el = all[i];
			var r = el.getBoundingClientRect();
			if (r.top > 495 && r.bottom < 545 && r.width > 50) {
				var t = (el.textContent||'').trim().replace(/\\s+/g,' ').slice(0,24);
				out.push({tag: el.tagName.toLowerCase(), cls: (el.className||'').toString().slice(0,40), x:Math.round(r.left), y:Math.round(r.top), w:Math.round(r.width), h:Math.round(r.height), t:t});
			}
		}
		return JSON.stringify(out);
	})()`))
	// 弹窗整体 + 主要容器 rect
	fmt.Println("[toolcfg] 容器 rect:")
	fmt.Println(js(wv, `(function(){
		var out = {};
		var sels = ['.tool-config-popover','.tcp-panel','.rcp-header','.rcp-search','.tcp-switch-list','.tcp-footer','.chat-input'];
		for (var i=0;i<sels.length;i++) {
			var el = document.querySelector(sels[i]);
			if (el) { var r = el.getBoundingClientRect(); out[sels[i]] = {x:Math.round(r.left), y:Math.round(r.top), w:Math.round(r.width), h:Math.round(r.height)}; }
		}
		return JSON.stringify(out);
	})()`))

	// 渲染树中 y∈[498,545] 的所有对象（含 text）
	fmt.Println("\n[toolcfg] 渲染树 y∈[498,545]:")
	rv2 := wv.RenderView()
	state2 := rv2.LayoutState()
	var dumpRect func(o rendering.RenderObject, depth int)
	dumpRect = func(o rendering.RenderObject, depth int) {
		if o == nil || depth > 60 {
			return
		}
		var gx, gy, gw, gh float64 = -1, -1, -1, -1
		if lb := o.LayoutBox(); lb != nil && state2 != nil {
			g := state2.GeometryForBox(lb)
			gx, gy, gw, gh = g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight()
		}
		// RenderText 用 segments
		if rt, ok := o.(*rendering.RenderText); ok {
			segs := rt.Segments()
			for _, s := range segs {
				if s.Y >= 495 && s.Y <= 545 {
					fmt.Printf("%*s[TEXT] seg=(%.0f,%.0f %dx%.0f) «%s»\n", depth*2, "", s.X, s.Y, s.Width, s.Height, strings.TrimSpace(rt.Text())[:min(20, len(strings.TrimSpace(rt.Text())))])
				}
			}
		}
		if gy >= 495 && gy <= 545 {
			desc := "?"
			if nn := o.Node(); nn != nil {
				if el, ok := nn.(*dom.Element); ok {
					desc = el.LocalName() + "." + el.GetAttribute("class")
				} else {
					desc = nn.NodeName()
				}
			}
			fmt.Printf("%*s[%s] geom=(%.0f,%.0f %dx%.0f)\n", depth*2, "", desc, gx, gy, gw, gh)
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			dumpRect(c, depth+1)
		}
	}
	dumpRect(rv2, 0)

	// 渲染树：tcp-switch-list 子树（cat-header / switch-item / name / toggle / indicator）
	fmt.Println("\n[toolcfg] 渲染树 switch-list 子树:")
	rv3 := wv.RenderView()
	state3 := rv3.LayoutState()
	var dumpSub func(o rendering.RenderObject, depth int, match func(el *dom.Element) bool)
	dumpSub = func(o rendering.RenderObject, depth int, match func(el *dom.Element) bool) {
		if o == nil || depth > 80 {
			return
		}
		if nn := o.Node(); nn != nil {
			if el, ok := nn.(*dom.Element); ok && match(el) {
				var gx, gy, gw, gh float64 = -1, -1, -1, -1
				if lb := o.LayoutBox(); lb != nil && state3 != nil {
					g := state3.GeometryForBox(lb)
					gx, gy, gw, gh = g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight()
				}
				var fx, fy, fw, fh float64 = -1, -1, -1, -1
				if rb, ok := o.(*rendering.RenderBox); ok {
					fx, fy, fw, fh = rb.X(), rb.Y(), rb.Width(), rb.Height()
				}
				c := el.GetAttribute("class")
				txt := ""
				if el.LocalName() == "span" {
					txt = strings.TrimSpace(el.TextContent())
					if len(txt) > 14 {
						txt = txt[:14]
					}
				}
				fmt.Printf("%*s[%s.%s] geom=(%.0f,%.0f %dx%.0f) frame=(%.0f,%.0f %dx%.0f) «%s»\n",
					depth*2, "", el.LocalName(), c, gx, gy, gw, gh, fx, fy, fw, fh, txt)
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			dumpSub(c, depth+1, match)
		}
	}
	// 只输出 switch-list 内部：找到 tcp-switch-list 节点后从其子树开始
	var findAndDump func(o rendering.RenderObject)
	findAndDump = func(o rendering.RenderObject) {
		if o == nil {
			return
		}
		if nn := o.Node(); nn != nil {
			if el, ok := nn.(*dom.Element); ok && strings.Contains(el.GetAttribute("class"), "tcp-switch-list") {
				dumpSub(o, 0, func(el *dom.Element) bool {
					c := el.GetAttribute("class")
					return strings.Contains(c, "tcp-switch-category") || strings.Contains(c, "tcp-cat-header") ||
						strings.Contains(c, "tcp-switch-item") || strings.Contains(c, "tcp-switch-name") ||
						strings.Contains(c, "tcp-switch-info") || strings.Contains(c, "toggle-indicator") ||
						strings.Contains(c, "tool-switch-toggle") || strings.Contains(c, "tcp-cat-count")
				})
				return
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			findAndDump(c)
		}
	}
	findAndDump(rv2)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
