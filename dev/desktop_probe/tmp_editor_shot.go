// Command tmp_editor_shot 复现桌面端编辑器（CM6）行号栏问题：
// 通过 window.__state 注入打开一个 40+ 行的 Go 文件，渲染后 dump
// .cm-editor/.cm-gutters/.cm-lineNumbers/.cm-gutterElement/.cm-activeLine*
// 的几何与 computed 背景，并保存 PNG 供像素对比。
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

	"wb-ui/app"
	"wb-ui/bindings"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func runJobs(wv *webkit.WebView) {
	for i := 0; i < 15; i++ {
		interp := wv.JSInterpreter()
		interp.RunJobs()
		// ★ 模拟 Host 主循环：ProcessTasks 执行 rAF 回调（CM6 measure）
		if el := interp.GetEventLoop(); el != nil {
			el.ProcessTasks(0)
		}
		interp.RunJobs()
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
	// ★ rAF hook：追踪 CM6 的 measure 是否注册/执行
	js(wv, `(function(){
		window.__rafHits = [];
		var orig = window.requestAnimationFrame;
		window.requestAnimationFrame = function(fn) {
			window.__rafHits.push((String(fn)||'').slice(0, 60));
			return orig(fn);
		};
		return 'hooked';
	})()`)
	fmt.Println("[step] before LoadHTML")
	wv.LoadHTML(string(htmlData))
	fmt.Println("[step] after LoadHTML")
	for i := 0; i < 6; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
		time.Sleep(600 * time.Millisecond)
		runJobs(wv)
	}

	// 注入打开文件：40 行 Go 代码（验证 2 位数行号 + 高亮）
	fmt.Println("[step] before inject")
	code := `package main

import "fmt"

// main is the entry point.
func main() {
	fmt.Println("hello")
	for i := 0; i < 10; i++ {
		fmt.Println(i)
	}
}

func add(a, b int) int {
	return a + b
}

func sub(a, b int) int {
	return a - b
}

func mul(a, b int) int {
	return a * b
}

func div(a, b int) int {
	return a / b
}

func square(x int) int {
	return x * x
}

func cube(x int) int {
	return x * x * x
}

func fib(n int) int {
	if n <= 1 {
		return n
	}
	return fib(n-1) + fib(n-2)
}
`
	code = strings.ReplaceAll(code, "\\", "\\\\")
	code = strings.ReplaceAll(code, "\n", "\\n")
	code = strings.ReplaceAll(code, "'", "\\'")
	// ★ 注入前 hook rAF（CM6 构造发生在注入的微任务里，hook 必须先于注入）
	js(wv, `(function(){
		window.__rafHits = [];
		window.__rafErr = '';
		window.__jsErrs = [];
		var origErr = console.error;
		console.error = function() {
			try { window.__jsErrs.push(Array.prototype.slice.call(arguments).map(function(a){ return (a && a.message) ? a.message : String(a); }).join(' ').slice(0, 200)); } catch(e) {}
			return origErr.apply(console, arguments);
		};
		var orig = window.requestAnimationFrame;
		window.__origStr = String(orig).slice(0, 150);
		window.requestAnimationFrame = function(fn) {
			window.__rafHits.push('REG:' + (String(fn)||'').slice(0, 50));
			return orig(function() {
				window.__rafHits.push('EXEC:' + (String(fn)||'').slice(0, 50));
				try { return fn(); }
				catch(e) { window.__rafErr = (window.__rafErr ? window.__rafErr + ' ;; ' : '') + e.message; throw e; }
			});
		};
		// ★ hook getBoundingClientRect 记录 CM6 测量 dummy（absolute .cm-line）的返回值
		window.__dummyRects = [];
		var origGBR = Element.prototype.getBoundingClientRect;
		Element.prototype.getBoundingClientRect = function() {
			var r = origGBR.apply(this, arguments);
			var isDummy = this.className === 'cm-line' && this.style && this.style.position === 'absolute';
			if (isDummy) {
				window.__dummyRects.push((r.height).toFixed(1) + 'x' + (r.width).toFixed(0));
			}
			return r;
		};
		return 'hooked-before-inject';
	})()`)
	js(wv, `(function(){
		var p = '/workspace/main.go';
		var st = window.__state;
		st.openFiles = [p];
		st.activeFile = p;
		st.fileContents[p] = '`+code+`';
		return 'injected openFiles=' + st.openFiles.length + ' active=' + st.activeFile;
	})()`)
	// 注入后：dump rAF 注册记录（看 CM6 measure 回调是否注册）+ defaultView 形态
	fmt.Println("[raf-hits] " + js(wv, `(function(){
		var h = window.__rafHits;
		var out = (h === undefined) ? 'hook LOST' : (h.length + ' calls');
		if (h) for (var i = 0; i < h.length; i++) out += ' | ' + i + ':' + h[i];
		out += ' | rafErr=[' + (window.__rafErr || '') + ']';
		out += ' | jsErrs=[' + ((window.__jsErrs || []).join(' ;; ').slice(0, 300)) + ']';
		out += ' | dummyRects=[' + ((window.__dummyRects || []).join(' ').slice(0, 200)) + ']';
		var dv = document.defaultView;
		out += ' | docDV:isWin=' + (dv === window) + ' hasRAF=' + !!(dv && dv.requestAnimationFrame);
		return out;
	})()`))
	if el := wv.JSInterpreter().GetEventLoop(); el != nil {
		fmt.Printf("[el] exists pending=%d\n", el.PendingTasks())
	} else {
		fmt.Println("[el] NIL")
	}
	// 注入后立即 ProcessTasks：看 measure 回调是否执行（EXEC）
	if el := wv.JSInterpreter().GetEventLoop(); el != nil {
		el.ProcessTasks(0)
		fmt.Printf("[el-after-pt] pending=%d\n", el.PendingTasks())
	}
	fmt.Println("[raf-after-pt] " + js(wv, `(function(){ return (window.__rafHits||[]).join(' | '); })()`))
	// ★ 诊断：注入后逐轮打印，定位 CM6 创建/measure 卡点
	if out := wv.ConsoleOutput(); out != "" {
		fmt.Println("[console-errs] " + out)
	}
	// 模拟宿主主循环：ResizeObserverCheck（CM6 靠 RO 初始回调触发 measure）
	bindings.ResizeObserverCheck(wv.JSInterpreter())
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("[after-ro-check] gutter=" + js(wv, `(function(){
		var ge = document.querySelectorAll('.cm-lineNumbers .cm-gutterElement')[1];
		var out = 'style=[' + (ge ? ge.getAttribute('style') : 'none') + ']';
		var sc = document.querySelector('.cm-scroller');
		var co = document.querySelector('.cm-content');
		out += ' | scCH=' + (sc ? sc.clientHeight : '?') + ' scCW=' + (sc ? sc.clientWidth : '?');
		out += ' | coCW=' + (co ? co.clientWidth : '?');
		var v = window.__editorView;
		if (v && v.viewState) {
			var o = v.viewState.heightOracle;
			out += ' | oracle.lh=' + (o ? o.lineHeight : '?');
			var vp = v.viewState.viewport;
			out += ' | viewport=' + (vp ? vp.from + '-' + vp.to : '?');
			var vl = v.viewState.viewportLines;
			out += ' | vpLines=' + (vl ? vl.length : '?') + (vl && vl[0] ? ' firstH=' + vl[0].height : '');
			// tile 结构：cm-content 子块数量 + 行数
			var kids = co ? co.children.length : -1;
			var lineCount = co ? co.querySelectorAll('.cm-line').length : -1;
			out += ' | coKids=' + kids + ' lines=' + lineCount;
			// ★ 实测 docView.tile 内部结构（堆栈级）
			try {
				var t = v.docView.tile;
				out += ' | tileK0=' + (t && t.constructor && t.constructor.name) + ' kids=' + (t ? t.children.length : '?');
				if (t && t.children[0]) {
					out += ' k0=' + t.children[0].constructor.name + '/' + (t.children[0].dom && t.children[0].dom.className);
					var c0 = t.children[0];
					if (c0.children) out += ' k0kids=' + c0.children.length + ' k0c0=' + (c0.children[0] && c0.children[0].constructor && c0.children[0].constructor.name) + '/' + (c0.children[0] && c0.children[0].dom && c0.children[0].dom.className);
				}
			} catch (te) { out += ' | tileErr=' + te.message; }
		} else { out += ' | no view'; }
		return out;
	})()`))
	// 立即复刻（注入后、runJobs 前——CM6 首个 rAF 测量的时序）
	fmt.Println("[pre-jobs-measure] " + js(wv, `(function(){
		var out = '';
		var d = document.createElement('div');
		d.className = 'cm-line';
		d.style.width = '99999px';
		d.style.position = 'absolute';
		d.textContent = 'abc def ghi jkl mno pqr stu';
		var tile = document.querySelector('.cm-content');
		if (!tile) return 'no tile';
		try {
			var mo = new MutationObserver(function(){});
			mo.ignore(function(){
				tile.appendChild(d);
				var dr = d.getBoundingClientRect();
				var rng = document.createRange();
				rng.setEnd(d.firstChild, d.firstChild.nodeValue.length);
				rng.setStart(d.firstChild, 0);
				var rects = rng.getClientRects();
				out = 'h=' + dr.height + ' rects=' + (rects ? rects.length : 'null') + (rects && rects.length ? ' w=' + rects[0].width.toFixed(1) : '');
				d.remove();
			});
		} catch(e) { out = 'ERR=' + e.message; }
		return out;
	})()`))
	for i := 0; i < 4; i++ {
		t0 := time.Now()
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
		time.Sleep(350 * time.Millisecond)
		runJobs(wv)
		if i == 0 {
			// 注入后第一轮：复刻 CM6 完整 measure（此时 CM6 刚创建）
			fmt.Println("[loop0-measure] " + js(wv, `(function(){
				var out = '';
				var d = document.createElement('div');
				d.className = 'cm-line';
				d.style.width = '99999px';
				d.style.position = 'absolute';
				d.textContent = 'abc def ghi jkl mno pqr stu';
				var tile = document.querySelector('.cm-content');
				if (!tile) return 'no tile';
				try {
					var mo = new MutationObserver(function(){});
					mo.ignore(function(){
						tile.appendChild(d);
						var dr = d.getBoundingClientRect();
						var rng = document.createRange();
						rng.setEnd(d.firstChild, d.firstChild.nodeValue.length);
						rng.setStart(d.firstChild, 0);
						var rects = rng.getClientRects();
						out = 'h=' + dr.height + ' rects=' + (rects ? rects.length : 'null') + (rects && rects.length ? ' w=' + rects[0].width.toFixed(1) : '');
						d.remove();
					});
				} catch(e) { out = 'ERR=' + e.message; }
				var ge = document.querySelectorAll('.cm-lineNumbers .cm-gutterElement')[1];
				if (ge) out += ' | gutter style=[' + (ge.getAttribute('style')||'') + ']';
				return out;
			})()`))
		}
		fmt.Printf("[loop] i=%d took=%v cmEditor=%s\n", i, time.Since(t0).Round(time.Millisecond), js(wv, `(function(){ return document.querySelector('.cm-editor') ? 'Y' : 'N'; })()`))
	}
	// 早期测量复刻（模拟 CM6 创建后首个 rAF 测量：Vue 刚 patch 完 DOM）
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 50); })`)
	time.Sleep(80 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("[early-measure] " + js(wv, `(function(){
		// 1) MutationObserver.ignore 支持？
		var obsErr = '';
		try {
			var mo = new MutationObserver(function(){});
			if (typeof mo.ignore === 'function') { mo.ignore(function(){}); }
			else { obsErr = 'no ignore method'; }
		} catch(e) { obsErr = e.message; }
		// 2) CM6 完整 measure 链（dummy + Range + observer.ignore）
		var d = document.createElement('div');
		d.className = 'cm-line';
		d.style.width = '99999px';
		d.style.position = 'absolute';
		d.textContent = 'abc def ghi jkl mno pqr stu';
		var tile = document.querySelector('.cm-content');
		if (!tile) return 'no tile | obsErr=' + obsErr;
		var out = 'obsErr=' + obsErr;
		try {
			var mo2 = new MutationObserver(function(){});
			mo2.ignore(function(){
				tile.appendChild(d);
				var dr = d.getBoundingClientRect();
				var rng = document.createRange();
				rng.setEnd(d.firstChild, d.firstChild.nodeValue.length);
				rng.setStart(d.firstChild, 0);
				var rects = rng.getClientRects();
				out += ' | h=' + dr.height + ' rects=' + (rects ? rects.length : 'null');
				if (rects && rects.length) out += ' r0=' + rects[0].width.toFixed(2) + 'x' + rects[0].height.toFixed(2);
				d.remove();
			});
		} catch(e) { out += ' | measureERR=' + e.message; }
		var ge = document.querySelectorAll('.cm-lineNumbers .cm-gutterElement')[1];
		if (ge) out += ' | gutterEl style=[' + (ge.getAttribute('style') || '') + ']';
		return out;
	})()`))
	for i := 0; i < 6; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
		time.Sleep(350 * time.Millisecond)
		runJobs(wv)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// dump CM6 DOM 几何 + 背景
	fmt.Println("[cm6] " + js(wv, `(function(){
		function info(sel){
			var el = document.querySelector(sel);
			if (!el) return sel + '=NULL';
			var r = el.getBoundingClientRect();
			var cs = getComputedStyle(el);
			var bg = cs.backgroundColor;
			var w = el.style ? el.style.width : '';
			var pos = cs.position;
			var disp = cs.display;
			var flexS = cs.flexShrink;
			var minW = cs.minWidth;
			var font = cs.fontSize;
			return sel + '=(' + Math.round(r.left) + ',' + Math.round(r.top) + ' ' + Math.round(r.width) + 'x' + Math.round(r.height) + ') bg=' + bg + ' pos=' + pos + ' disp=' + disp + ' flexS=' + flexS + ' minW=' + minW + ' styleW=[' + w + '] fs=' + font;
		}
		var out = [];
		out.push(info('.cm-editor'));
		out.push(info('.cm-scroller'));
		out.push(info('.cm-content'));
		out.push(info('.cm-gutters'));
		out.push(info('.cm-lineNumbers'));
		var first = document.querySelector('.cm-gutterElement');
		if (first) {
			var r = first.getBoundingClientRect();
			var cs = getComputedStyle(first);
			out.push('.cm-gutterElement[0]=(' + Math.round(r.left) + ',' + Math.round(r.top) + ' ' + Math.round(r.width) + 'x' + Math.round(r.height) + ') bg=' + cs.backgroundColor + ' pad=' + cs.paddingLeft + ',' + cs.paddingRight);
		}
		out.push('gutterEls=' + document.querySelectorAll('.cm-gutterElement').length);
		out.push('activeLine=' + (document.querySelector('.cm-activeLine') ? (function(){ var r = document.querySelector('.cm-activeLine').getBoundingClientRect(); var cs = getComputedStyle(document.querySelector('.cm-activeLine')); return '(' + Math.round(r.left) + ',' + Math.round(r.top) + ' ' + Math.round(r.width) + 'x' + Math.round(r.height) + ') bg=' + cs.backgroundColor; })() : 'NULL'));
		out.push('activeLineGutter=' + (document.querySelector('.cm-activeLineGutter') ? (function(){ var r = document.querySelector('.cm-activeLineGutter').getBoundingClientRect(); var cs = getComputedStyle(document.querySelector('.cm-activeLineGutter')); return '(' + Math.round(r.left) + ',' + Math.round(r.top) + ' ' + Math.round(r.width) + 'x' + Math.round(r.height) + ') bg=' + cs.backgroundColor; })() : 'NULL'));
		out.push('lines=' + document.querySelectorAll('.cm-line').length);
		out.push('cursor=' + (document.querySelector('.cm-cursor') ? 'Y' : 'N'));
		var g1 = document.querySelector('.cm-gutterElement');
		if (g1) out.push('g0txt=[' + (g1.textContent || '') + ']');
		// 详细：gutter 前 4 个元素 + activeLineGutter 文本
		var ln = document.querySelector('.cm-lineNumbers');
		if (ln) {
			var kids = [];
			for (var i = 0; i < ln.children.length && i < 5; i++) {
				var c = ln.children[i];
				var cr = c.getBoundingClientRect();
				kids.push(i + ':[txt=' + (c.textContent||'').trim() + ' cls=' + c.className + ' y=' + Math.round(cr.top) + ' h=' + Math.round(cr.height) + ']');
			}
			out.push('lineNumbersKids=' + kids.join(' '));
		}
		var alg = document.querySelector('.cm-activeLineGutter');
		if (alg) out.push('activeLineGutterTxt=[' + (alg.textContent||'').trim() + ']');
		// spacer 诊断：第一个 gutterElement 的内联样式与 computed 样式
		var sp = document.querySelector('.cm-lineNumbers .cm-gutterElement');
		if (sp) {
			var spcs = getComputedStyle(sp);
			out.push('spacer: cssText=[' + (sp.getAttribute('style') || '') + '] vis=' + spcs.visibility + ' h=' + spcs.height + ' minH=' + spcs.minHeight + ' boxSizing=' + spcs.boxSizing + ' lineH=' + spcs.lineHeight);
		}
		var ln2 = document.querySelector('.cm-lineNumbers');
		if (ln2) {
			var lncs = getComputedStyle(ln2);
			out.push('lineNumbers: cssText=[' + (ln2.getAttribute('style') || '') + '] vis=' + lncs.visibility + ' h=' + lncs.height + ' flexDir=' + lncs.flexDirection + ' overflow=' + lncs.overflow);
		}
		var g2 = document.querySelector('.cm-gutters');
		if (g2) {
			var gcs = getComputedStyle(g2);
			out.push('gutters: cssText=[' + (g2.getAttribute('style') || '') + '] vis=' + gcs.visibility + ' h=' + gcs.height + ' pos=' + gcs.position + ' bg=' + gcs.backgroundColor);
		}
		// 行号元素 style.height vs 内容行高
		var ge1 = document.querySelectorAll('.cm-lineNumbers .cm-gutterElement')[1];
		if (ge1) out.push('line1: style=[' + (ge1.getAttribute('style') || '') + ']');
		var ge2 = document.querySelectorAll('.cm-lineNumbers .cm-gutterElement')[2];
		if (ge2) out.push('line2: style=[' + (ge2.getAttribute('style') || '') + ']');
		var cl = document.querySelectorAll('.cm-line')[0];
		if (cl) {
			var clr = cl.getBoundingClientRect();
			var clcs = getComputedStyle(cl);
			out.push('.cm-line[0]=(' + Math.round(clr.top) + ' ' + Math.round(clr.height) + ') lh=' + clcs.lineHeight + ' fs=' + clcs.fontSize);
		}
		var co = document.querySelector('.cm-content');
		if (co) {
			var cors = getComputedStyle(co);
			out.push('content: padT=' + cors.paddingTop + ' padB=' + cors.paddingBottom + ' lh=' + cors.lineHeight);
		}
		// 复刻 CM6 TextWidth.measure 的 dummy 测量（absolute div.cm-line）
		var d = document.createElement('div');
		d.className = 'cm-line';
		d.style.width = '99999px';
		d.style.position = 'absolute';
		d.textContent = 'abc def ghi jkl mno pqr stu';
		var tile = document.querySelector('.cm-content');
		if (tile) {
			tile.appendChild(d);
			var dr = d.getBoundingClientRect();
			var dcs = getComputedStyle(d);
			out.push('dummy: h=' + dr.height + ' w=' + dr.width + ' fs=' + dcs.fontSize + ' lh=' + dcs.lineHeight + ' ff=' + dcs.fontFamily);
			d.remove();
		}
		// .cm-line computed 行高
		var cl0 = document.querySelector('.cm-line');
		if (cl0) {
			var cs0 = getComputedStyle(cl0);
			out.push('cm-line: fs=' + cs0.fontSize + ' lh=' + cs0.lineHeight + ' h=' + cs0.height + ' ff=' + cs0.fontFamily);
		}
		var sc2 = document.querySelector('.cm-scroller');
		if (sc2) out.push('scrollTop=' + sc2.scrollTop + ' scrollH=' + sc2.scrollHeight + ' clientH=' + sc2.clientHeight);
		var ed = document.querySelector('.cm-editor');
		if (ed) {
			out.push('editorCls=[' + ed.className + ']');
			var edcs = getComputedStyle(ed);
			out.push('editorBg(cs)=' + edcs.backgroundColor + ' font=' + edcs.fontFamily);
		}
		return out.join(' || ');
	})()`))

	// 渲染截图
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	pngBytes, err := wv.Render()
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	fmt.Printf("[render] bytes=%d\n", len(pngBytes))
	out := filepath.Join(wd, "dev", "desktop_probe", "editor_shot.png")
	f, err := os.Create(out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, wv.Width(), wv.Height()))
	for y := 0; y < wv.Height(); y++ {
		for x := 0; x < wv.Width(); x++ {
			off := (y*wv.Width() + x) * 4
			if off+3 < len(pngBytes) {
				img.SetRGBA(x, y, color.RGBA{R: pngBytes[off], G: pngBytes[off+1], B: pngBytes[off+2], A: pngBytes[off+3]})
			}
		}
	}
	_ = png.Encode(f, img)
	fmt.Println("[shot] → dev/desktop_probe/editor_shot.png")

	// 查找 .cm-gutters 渲染 box 的位置（确认 gutter 是否被绘制）
	_ = app.NewHostForTest(wv, 1280, 800)
	fmt.Println("DONE")
}
