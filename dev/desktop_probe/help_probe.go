// Command help_probe 在 wb-ui（jsc）引擎中加载真实 dist，诊断「标题栏帮助点击无反应」：
//   1. 标题栏/帮助按钮是否渲染（rect）
//   2. 初始状态真实坐标 hit-test（帮助按钮区域命中什么）
//   3. 引擎 MouseEvent 派发（handleClick 等价路径）→ dropdown 是否出现
//   4. JS new MouseEvent 派发（对照）→ dropdown 是否出现
//   5. 点击菜单项「功能介绍」→ .help-modal 是否渲染（marked 解析是否报错）
//   6. 输出 console 错误
// 用法：go run ./dev/desktop_probe/help_probe.go
//go:build ignore

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

func wait(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(time.Duration(ms) * time.Millisecond)
	runJobs(wv)
}

func dropdownState(wv *webkit.WebView) string {
	return js(wv, `(function(){
		var dd = document.querySelector('.menu-dropdown');
		return JSON.stringify({dropdown: dd ? 'visible' : 'MISSING', items: document.querySelectorAll('.menu-dropdown .menu-item').length});
	})()`)
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
		wait(wv, 600)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// ── 1. 标题栏/帮助按钮是否渲染 ──
	fmt.Println("[1] 标题栏结构:")
	fmt.Println("  " + js(wv, `(function(){
		var out = {};
		var tb = document.querySelector('.titlebar');
		out.titlebar = tb ? rect(tb) : null;
		var mb = document.querySelector('.menubar');
		out.menubar = mb ? rect(mb) : null;
		var btns = document.querySelectorAll('.menu-btn');
		out.menuBtns = [];
		for (var i=0;i<btns.length;i++) {
			var b = btns[i];
			out.menuBtns.push({label: b.textContent.trim(), rect: rect(b), display: cs(b).display});
		}
		return JSON.stringify(out);
		function rect(el){ var r = el.getBoundingClientRect(); return {x:Math.round(r.left), y:Math.round(r.top), w:Math.round(r.width), h:Math.round(r.height)}; }
		function cs(el){ return getComputedStyle(el); }
	})()`))

	// ── 2. 初始状态真实坐标 hit-test ──
	fmt.Println("[2] 初始状态 hit-test（帮助按钮 rect 48,0,46,30 → 中心 71,15）:")
	rv := wv.RenderView()
	pts := [][2]float64{{71, 15}, {48, 5}, {94, 29}, {60, 15}, {130, 15}, {10, 15}}
	for _, pt := range pts {
		el := rendering.HitTest(rv, pt[0], pt[1], "")
		name := "<nil>"
		if el != nil {
			name = el.LocalName() + "." + el.GetAttribute("class")
		}
		fmt.Printf("  hit(%.0f,%.0f) → %s\n", pt[0], pt[1], name)
	}

	// ── 3. 引擎 MouseEvent 派发（handleClick 等价路径）──
	fmt.Println("[3] 引擎 MouseEvent 派发（handleClick 等价）:")
	el := rendering.HitTest(rv, 71, 15, "")
	if el != nil {
		el.DispatchEvent(dom.NewMouseEvent(dom.EventClick, true, true, false))
		wait(wv, 350)
		wv.RebuildRenderTree()
		wv.EnsureLayout()
		fmt.Println("  " + dropdownState(wv))
	} else {
		fmt.Println("  hit-test 未命中帮助按钮！")
	}

	// ── 4. JS new MouseEvent 派发（对照）──
	// 先关闭 dropdown（点 titlebar 触发 closeAllMenus）
	_ = js(wv, `(function(){ var tb = document.querySelector('.titlebar'); if (tb) tb.dispatchEvent(new MouseEvent('click', {bubbles:true})); })()`)
	wait(wv, 200)
	fmt.Println("[4] JS new MouseEvent 派发（对照）:")
	fmt.Println("  " + js(wv, `(function(){
		var btn = document.querySelector('.menu-btn');
		var out = {btnFound: !!btn};
		if (btn) btn.dispatchEvent(new MouseEvent('click', {bubbles:true, cancelable:true}));
		return JSON.stringify(out);
	})()`))
	wait(wv, 300)
	fmt.Println("  " + dropdownState(wv))

	// ── 4.5 dropdown 细节（divider/各项 rect，对比浏览器）──
	fmt.Println("[4.5] dropdown 细节:")
	fmt.Println("  " + js(wv, `(function(){
		var out = {};
		var dd = document.querySelector('.menu-dropdown');
		out.dropdownRect = dd ? rect(dd) : null;
		out.ddPadTop = dd ? cs(dd).paddingTop : null;
		out.ddPadBot = dd ? cs(dd).paddingBottom : null;
		var items = document.querySelectorAll('.menu-dropdown .menu-item');
		out.items = [];
		for (var i=0;i<items.length;i++) {
			var it = items[i];
			out.items.push({label: it.textContent.trim().slice(0,6), rect: rect(it), mt: cs(it).marginTop, mb: cs(it).marginBottom, lh: cs(it).lineHeight});
		}
		var dv = document.querySelectorAll('.menu-dropdown .menu-divider');
		out.dividers = [];
		for (var i=0;i<dv.length;i++) {
			out.dividers.push({rect: rect(dv[i]), mt: cs(dv[i]).marginTop, mb: cs(dv[i]).marginBottom});
		}
		return JSON.stringify(out);
		function rect(el){ var r = el.getBoundingClientRect(); return {x:Math.round(r.left), y:Math.round(r.top), w:Math.round(r.width), h:Math.round(r.height)}; }
		function cs(el){ return getComputedStyle(el); }
	})()`))

	// ── 5. 点击「功能介绍」菜单项 → HelpModal ──
	fmt.Println("[5] 点击功能介绍菜单项后 HelpModal:")
	fmt.Println("  " + js(wv, `(function(){
		var items = document.querySelectorAll('.menu-dropdown .menu-item');
		var out = {items: items.length};
		for (var i=0;i<items.length;i++) {
			var t = items[i].textContent.trim();
			if (t.indexOf('功能介绍') >= 0) {
				items[i].dispatchEvent(new MouseEvent('click', {bubbles:true, cancelable:true}));
				out.clicked = t;
			}
		}
		return JSON.stringify(out);
	})()`))
	wait(wv, 400)
	fmt.Println("  " + js(wv, `(function(){
		var out = {};
		var m = document.querySelector('.help-modal');
		out.helpModal = m ? {rect: rect(m), display: cs(m).display} : null;
		var ov = document.querySelector('.modal-overlay');
		out.overlay = ov ? {rect: rect(ov)} : null;
		var docs = document.querySelectorAll('.doc-nav-item');
		out.docNav = docs.length;
		var content = document.querySelector('.doc-content');
		out.contentHTML = content ? content.innerHTML.slice(0, 200) : null;
		out.helpModalCount = document.querySelectorAll('.help-modal').length;
		return JSON.stringify(out);
		function rect(el){ var r = el.getBoundingClientRect(); return {x:Math.round(r.left), y:Math.round(r.top), w:Math.round(r.width), h:Math.round(r.height)}; }
		function cs(el){ return getComputedStyle(el); }
	})()`))

	// ── 7. AboutModal 关闭按钮（点 HelpModal 头部「关于」）──
	fmt.Println("[7] AboutModal 关闭按钮:")
	fmt.Println("  " + js(wv, `(function(){
		var ab = document.querySelector('.btn-about');
		var out = {aboutBtn: !!ab};
		if (ab) ab.dispatchEvent(new MouseEvent('click', {bubbles:true, cancelable:true}));
		return JSON.stringify(out);
	})()`))
	wait(wv, 400)
	// ★ 完整渲染循环后再测（真实桌面端是 RebuildRenderTree + EnsureLayout 后绘制）
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
		fmt.Println("  " + js(wv, `(function(){
			var out = {};
			var m = document.querySelector('.about-modal');
			out.aboutModal = m ? rect(m) : null;
			var f = document.querySelector('.about-modal .modal-footer');
			out.footer = f ? rect(f) : null;
			var btns = document.querySelectorAll('.about-modal .modal-footer button');
			out.buttons = [];
			for (var i=0;i<btns.length;i++) {
				var b = btns[i];
				out.buttons.push({cls: b.className, text: b.textContent.trim(), rect: rect(b),
					ws: cs(b).whiteSpace, disp: cs(b).display, flexShrink: cs(b).flexShrink,
					minW: cs(b).minWidth, w: cs(b).width, lh: cs(b).lineHeight, fs: cs(b).fontSize,
					fam: cs(b).fontFamily, boxSizing: cs(b).boxSizing, clientW: b.clientWidth});
			}
			// 关闭按钮文字是否换行：取按钮文本节点数
			var closeBtn = document.querySelector('.about-modal .modal-footer .btn-secondary');
			out.closeTextNodes = closeBtn ? closeBtn.childNodes.length : null;
			return JSON.stringify(out);
			function rect(el){ var r = el.getBoundingClientRect(); return {x:Math.round(r.left), y:Math.round(r.top), w:Math.round(r.width), h:Math.round(r.height)}; }
			function cs(el){ return getComputedStyle(el); }
		})()`))

		// ── 7.5 AboutModal 关闭按钮布局几何 dump ──
		rv3 := wv.RenderView()
		st3 := rv3.LayoutState()
		var walkRO3 func(ro rendering.RenderObject)
		walkRO3 = func(ro rendering.RenderObject) {
			if ro == nil {
				return
			}
			if n := ro.Node(); n != nil {
				if el, ok := n.(*dom.Element); ok && strings.Contains(el.GetAttribute("class"), "btn-secondary") {
					lb := ro.LayoutBox()
					if lb != nil && st3 != nil {
						g := st3.GeometryForBox(lb)
						fmt.Printf("  about-btn-secondary: box=(%.1f,%.1f %.1fx%.1f) content=(%.1f,%.1f %.2fx%.2f) padL=%.1f padR=%.1f padT=%.1f padB=%.1f\n",
							g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(),
							g.ContentBoxLeft(), g.ContentBoxTop(), g.ContentWidth(), g.ContentHeight(),
							g.PaddingLeft(), g.PaddingRight(), g.PaddingTop(), g.PaddingBottom())
					}
					// dump 按钮第一个子节点（匿名 wrapper）几何
					if c := ro.FirstChild(); c != nil {
						if clb := c.LayoutBox(); clb != nil && st3 != nil {
							cg := st3.GeometryForBox(clb)
							fmt.Printf("  about-btn-wrapper: box=(%.1f,%.1f %.1fx%.1f) content=(%.1f,%.1f %.2fx%.2f) name=%s\n",
								cg.Left(), cg.Top(), cg.BorderBoxWidth(), cg.BorderBoxHeight(),
								cg.ContentBoxLeft(), cg.ContentBoxTop(), cg.ContentWidth(), cg.ContentHeight(),
								c.RenderName())
						}
					}
				}
			}
			for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
				walkRO3(c)
			}
		}
		walkRO3(rv3)

	// ── 8. 动态按钮实验：无 flex 压缩环境下的按钮文字换行 ──
	fmt.Println("[8] 动态按钮实验（普通流中 '关闭' 按钮）:")
	fmt.Println("  " + js(wv, `(function(){
		var host = document.createElement('div');
		host.style.cssText = 'position:fixed;left:10px;top:700px;z-index:99999;background:#333';
		host.innerHTML = '<button id="tbtn" style="padding:7px 16px;font-size:13px">关闭</button>';
		document.body.appendChild(host);
		var b = document.getElementById('tbtn');
		var r = b.getBoundingClientRect();
		var cs2 = getComputedStyle(b);
		return JSON.stringify({rect: {x:Math.round(r.left),y:Math.round(r.top),w:Math.round(r.width),h:Math.round(r.height)},
			whiteSpace: cs2.whiteSpace, disp: cs2.display, padL: cs2.paddingLeft, padR: cs2.paddingRight, fs: cs2.fontSize});
	})()`))
	wait(wv, 300)

	// ── 8.5 模拟 footer flex 容器实验 ──
	fmt.Println("[8.5] 模拟 footer flex 容器:")
	fmt.Println("  " + js(wv, `(function(){
		var f = document.createElement('div');
		f.id = 'testfooter';
		f.style.cssText = 'position:fixed;left:10px;top:620px;width:880px;display:flex;align-items:center;justify-content:flex-end;gap:8px;padding:12px 20px;background:#333;z-index:99999;box-sizing:border-box';
		f.innerHTML = '<button id="tb1" style="display:flex;align-items:center;gap:6px;padding:7px 16px;font-size:13px">查看帮助文档</button>' +
			'<button id="tb2" style="padding:7px 16px;font-size:13px">关闭</button>';
		document.body.appendChild(f);
		function R(id){ var el = document.getElementById(id); var r = el.getBoundingClientRect(); return {x:Math.round(r.left),y:Math.round(r.top),w:Math.round(r.width),h:Math.round(r.height)}; }
		var out = {footer: R('testfooter'), b1: R('tb1'), b2: R('tb2')};
		// b2 内文字是否折行：对比按钮高度
		out.b2TextLines = Math.round(R('tb2').h / 32);
		// b2 内容区宽度：clientWidth(内容+padding, 不含border) - padding
		var b2 = document.getElementById('tb2');
		var csB2 = getComputedStyle(b2);
		out.b2clientW = b2.clientWidth;
		out.b2offsetW = b2.offsetWidth;
		out.b2padL = csB2.paddingLeft;
		out.b2padR = csB2.paddingRight;
		out.b2contentW = b2.clientWidth - parseFloat(csB2.paddingLeft) - parseFloat(csB2.paddingRight);
		// 显式 white-space:nowrap 对比
		b2.style.whiteSpace = 'nowrap';
		// 强制重新布局后测量（nowrap 是否修复）
		var r2 = b2.getBoundingClientRect();
		out.b2nowrap = {w:Math.round(r2.width), h:Math.round(r2.height)};
		// span 实测文字宽度（13px 字体）
		var s = document.createElement('span');
		s.style.cssText = 'position:fixed;left:10px;top:570px;font-size:13px;background:#444;z-index:99999';
		s.textContent = '关闭';
		document.body.appendChild(s);
		var sr = s.getBoundingClientRect();
		out.closeTextW = Math.round(sr.width * 10) / 10;
		var s2 = document.createElement('span');
		s2.style.cssText = 'position:fixed;left:10px;top:550px;font-size:13px;z-index:99999';
		s2.textContent = '关';
		document.body.appendChild(s2);
		out.charW = Math.round(s2.getBoundingClientRect().width * 10) / 10;
		return JSON.stringify(out);
	})()`))
	wait(wv, 300)

	// ── 8.6 tb2 布局几何 dump ──
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
	fmt.Println("[8.6] tb2 布局几何:")
	rv2 := wv.RenderView()
	st2 := rv2.LayoutState()
	var walkRO func(ro rendering.RenderObject)
	walkRO = func(ro rendering.RenderObject) {
		if ro == nil {
			return
		}
		if n := ro.Node(); n != nil {
			if el, ok := n.(*dom.Element); ok && el.GetAttribute("id") == "tb2" {
				lb := ro.LayoutBox()
				if lb != nil && st2 != nil {
					g := st2.GeometryForBox(lb)
					fmt.Printf("  tb2: box=(%.1f,%.1f %.1fx%.1f) content=(%.1f,%.1f %.1fx%.1f) padL=%.1f padR=%.1f padT=%.1f padB=%.1f\n",
						g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(),
						g.ContentBoxLeft(), g.ContentBoxTop(), g.ContentWidth(), g.ContentHeight(),
						g.PaddingLeft(), g.PaddingRight(), g.PaddingTop(), g.PaddingBottom())
				} else {
					fmt.Printf("  tb2: no layout box\n")
				}
			}
		}
		for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
			walkRO(c)
		}
	}
	walkRO(rv2)
	fmt.Println("[6] console 错误:")
	cout := wv.ConsoleOutput()
	lines := strings.Split(cout, "\n")
	errLines := 0
	for _, ln := range lines {
		low := strings.ToLower(ln)
		if strings.Contains(low, "error") || strings.Contains(low, "cannot") || strings.Contains(low, "undefined") || strings.Contains(low, "typeerror") || strings.Contains(low, "is not a function") {
			fmt.Println("  [jserr]", ln)
			errLines++
			if errLines > 15 {
				break
			}
		}
	}
	fmt.Println("  [console] total lines:", len(lines))

	// ── 9. 文字宽度测量（13px，各字体）──
	fmt.Println("[9] 文字宽度测量（13px）:")
	for _, fam := range []string{"", "Segoe UI", "Microsoft YaHei", "SimSun", "sans-serif"} {
		w2 := layout.MeasureTextFunc(fam, 13, 400, "", "关闭")
		w1 := layout.MeasureTextFunc(fam, 13, 400, "", "关")
		fmt.Printf("  fam=%-18q 关闭=%.2f 关=%.2f\n", fam, w2, w1)
	}
}
