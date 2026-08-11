package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"

	"wb-ui/app"
	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/rendering"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func main() {
	// ★ 全局异常捕获：traceback=all（panic 时输出所有 goroutine 堆栈），
	// 配合下方 defer recover 与 watchJSErrors，任何 Go/JS 异常都落到
	// _desktop_errors.log + 控制台，不再静默。
	debug.SetTraceback("all")
	log.SetFlags(log.Ltime)
	log.Println("[Desktop] PairCode IDE 桌面版 v1.0.6-desktop")

	// ★ WB_CPU_PROF=1：启动段（loadHTML + Vue mount + 终端初始化）Go CPU
	// profile，分析 goja VM 执行热点（优化终端启动慢）。3s 后自动停止。
	if os.Getenv("WB_CPU_PROF") != "" {
		f, err := os.Create("desktop_cpu.prof")
		if err == nil {
			_ = pprof.StartCPUProfile(f)
			go func() {
				time.Sleep(3 * time.Second)
				pprof.StopCPUProfile()
				f.Close()
				log.Println("[CPU-PROF] stopped → desktop_cpu.prof")
			}()
		}
	}

	// ★ 崩溃捕获：任何 panic 都写堆栈到 _desktop_panic.log（含所有 goroutine），
	//   方便实机复现"打开文件崩溃"时拿到完整现场。
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 1<<20)
			n := runtime.Stack(buf, true)
			msg := fmt.Sprintf("=== PANIC %v ===\n%s\n", r, buf[:n])
			_ = os.WriteFile("_desktop_panic.log", []byte(msg), 0644)
			log.Printf("[Desktop] PANIC: %v\n%s", r, buf[:n])
			appendErrLog("PANIC", msg)
			os.Exit(1)
		}
	}()

	wd, _ := os.Getwd()
	// ★ 与 web 端（9090）加载同一份前端：优先 cmd/companion/web-ui/dist，
	//   回退旧桌面构建 cmd/desktop/web-ui/dist / web-ui/dist。
	//   这样桌面端与浏览器（http://localhost:9090）渲染完全一致的前端。
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		distDir = filepath.Join(wd, "cmd", "desktop", "web-ui", "dist")
	}
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		distDir = filepath.Join(wd, "web-ui", "dist")
	}
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		log.Fatalf("[Desktop] cannot find web-ui/dist")
	}
	log.Printf("[Desktop] distDir: %s", distDir)

	htmlData, err := os.ReadFile(distDir + "/index.html")
	if err != nil {
		log.Fatalf("[Desktop] cannot read index.html: %v", err)
	}
	htmlStr := string(htmlData)

	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()

	// ★ 先初始化桥接（core 加载 + 真实 handler 注册 + fetch 拦截注入），
	//   再加载页面——页面 script 执行时 desktopBridge / go.bridge_call
	//   已就绪，/api/* 请求才能被拦截到本地 Go handler。
	desktopbridge.Init(wv)

	wv.LoadHTML(htmlStr)

	// ★ Go 主动调 JS（wb-ui CallFunction）：演示宿主直调页面全局函数。
	// __desktopNotify（desktopbridge 注入，BeforePageScripts 内定义）在
	// 页面脚本执行前已就绪——LoadHTML 返回后即可调用。业务侧可在任意
	// Go 事件点（agent done、进度推送、状态变化）以同一模式驱动前端。
	if v, err := wv.CallFunction("__desktopNotify", "PairCode IDE", "桌面端已就绪 — Go 主动调用 JS"); err != nil {
		log.Printf("[Desktop] CallFunction: %v", err)
	} else {
		log.Printf("[Desktop] CallFunction → %s", v.ToString())
	}

	// ★ 错误捕获 hook：把 JS 运行时错误记录到 window.__errs，
	//   WB_SNAP 布局快照的 errs 字段由此拿到（打开文件时若有 JS 异常，
	//   时间线上会清晰呈现）。
	if interp := wv.JSInterpreter(); interp != nil {
		_, _ = interp.RunJS(`window.__errs = [];
window.addEventListener('error', function(e){ window.__errs.push('error: ' + (e && (e.message || e.type))); }, true);
window.addEventListener('unhandledrejection', function(e){ window.__errs.push('rejection: ' + ((e && e.reason && e.reason.message) || String(e))); }, true);
var _ce = console.error;
console.error = function(){
  window.__errs.push('console.error: ' + Array.prototype.slice.call(arguments).map(function(a){
    var m = typeof a === 'string' ? a : ((a && a.message) || String(a));
    if (a && a.stack) m += ' | STACK: ' + String(a.stack).split('\n').slice(0, 6).join(' <- ');
    return m;
  }).join(' | ').slice(0, 500));
  return _ce.apply(console, arguments);
};`)
	}

	if out := wv.ConsoleOutput(); out != "" {
		log.Println("[CONSOLE]")
		log.Println(out)
	}

	log.Println("[LoadHTML] 加载成功")
	log.Println("[Desktop] window+render tree ready, creating host...")

	// 标准 Host：Skia 原生抗锯齿绘制，无需 DPR 超采样（SSAA 已移除）。
	host, err := app.NewHost(wv, 1280, 800, "PairCode IDE")
	if err != nil {
		log.Fatalf("[Desktop] 创建窗口失败: %v", err)
	}

	wv.EnsureLayout()
	writeRenderDiagnostic(wv)

	// ★ 注册 DumpRTCallback：Host.needsResizeDump 置位时（resize 后 / 终端
	// 自动化诊断 case）重新 dump 当前渲染树——启动时的 dump 早于 PTY 输出，
	// 看不到 xterm rows 动态内容。写带时间戳的文件（desktop_diag.log 可能
	// 被其他进程/残留实例占用导致 os.Create 失败）。
	app.DumpRTCallback = func(rv *rendering.RenderView) {
		if rv == nil {
			return
		}
		st := rv.LayoutState()
		if st == nil {
			return
		}
		f, err := os.Create(fmt.Sprintf("desktop_diag_%s.log", time.Now().Format("150405")))
		if err != nil {
			return
		}
		defer f.Close()
		fmt.Fprintln(f, "=== DESKTOP RENDER DIAGNOSTIC (re-dump) ===")
		fmt.Fprintln(f, "")
		fmt.Fprintln(f, "=== RENDER TREE ===")
		dumpRO(f, rv, 0, st)
		fmt.Fprintln(f, "")
		fmt.Fprintln(f, "=== ANOMALY ANALYSIS ===")
		reportAnomalies(f, rv, st)
		fmt.Fprintln(f, "")
		fmt.Fprintln(f, "=== DIAGNOSTIC COMPLETE ===")
	}

	// ★ 每帧回调：主循环消费外部队列的 JS 推送（终端 PTY 输出、agent
	// 事件）——goja 非线程安全，所有跨 goroutine 的 RunJS 必须在此
	// 主线程执行。
	// ★ WB_TERM_DIAG=1：每 90 帧查询终端 buffer 内容 + 行 span 渲染
	// 位置（诊断「初始文本不从顶部开始」——前 2 行为空是内容还是渲染）。
	termDiagFrame := 0
	errWatchFrame, lastErrsCount, lastConsoleLen = 0, 0, 0
	host.OnFrame = func() {
		desktopbridge.DrainMainQueue(wv)
		if os.Getenv("WB_TERM_DIAG") != "" && termDiagFrame%90 == 0 {
			diagTerm(wv, termDiagFrame/90)
		}
		// ★ 全局异常捕获并抛出：每 60 帧（≈1s）增量转储 JS 运行时错误
		// （window.__errs）与 console 输出到控制台 + _desktop_errors.log。
		// goja 非线程安全，RunJS 必须在此主线程执行，故不做独立 goroutine。
		errWatchFrame++
		if errWatchFrame%60 == 0 {
			watchJSErrors(wv)
		}
		termDiagFrame++
	}

	// ★ --probe-editor：自动化编辑器诊断模式——真实桌面全链路（桥接 + 前端 +
	// 渲染循环）下自动打开文件并聚焦编辑器，配合 WB_SNAP=1（组件绘制日志
	// _layout_snap.log）与 WB_DUMP_PNG=1/WB_DUMP_PNG_FRAME（真实窗口 PNG）
	// 反查「光标/背景是否绘制、绘制在哪」（组件绘制日志反向跟踪）。
	// 用法: set WB_SNAP=1 && set WB_DUMP_PNG=1 && set WB_DUMP_PNG_FRAME=420 && desktop.exe --probe-editor
	if hasArg("--probe-editor") {
		go probeEditor(wv)
	}

	log.Println("[Desktop] 窗口已启动，开始事件循环...")
	host.Run()
	log.Println("[Desktop] 已退出。")
}

// hasArg 检查命令行参数。
func hasArg(name string) bool {
	for _, a := range os.Args {
		if a == name {
			return true
		}
	}
	return false
}

// probeEditor 自动化编辑器诊断：走 desktopbridge.PushMainJS 把 JS 投递到
// 主循环队列（goja 非线程安全，不能直接从 goroutine RunJS）。时序：
//   1) 注入 window.__state 打开测试文件（CodeEditor 挂载 + 行 1 内容）
//   2) 注入点击 .cm-content（mousedown detail:1 → CM6 聚焦 → 光标显示）
//   3) 等待渲染若干帧（WB_SNAP 每 600ms 记录 [paint] 组件绘制）
//   4) 收集 _layout_snap.log → _probe_editor_snap.log，退出
func probeEditor(wv *webkit.WebView) {
	log.Println("[Probe] --probe-editor: 等待页面加载...")
	time.Sleep(4 * time.Second)
	openJS := `(function(){
		var st = window.__state;
		if (!st) return 'no-state';
		var p = 'F:\\\\syproject\\wb-ui\\renderpipeline.go';
		if (!st.openFiles.includes(p)) st.openFiles.push(p);
		st.activeFile = p;
		var c = 'package rendering\n\n// ---- probe-editor 测试文件 ----\n\ntype T struct{}\n\nfunc main() {\n\tfmt.Println("hello")\n}\n';
		st.fileContents[p] = c;
		st.fileSavedContent[p] = c;
		st.fileDirty[p] = false;
		return 'opened:' + st.activeFile;
	})()`
	desktopbridge.PushMainJS(openJS)
	log.Println("[Probe] 已投递打开文件脚本")
	time.Sleep(3 * time.Second)
	focusJS := `(function(){
		// ★ 模拟真实鼠标命中：用户点击命中行内子元素（.cm-line 下的文字），
		// mousedown 需冒泡到 .cm-content 才会触发 CM6 的 focus——与 probe
		// 之前直接点 cm-content（目标即 handler 所在元素）不同。
		var line = document.querySelector('.cm-line') || document.querySelector('.cm-content');
		if (!line) return 'no-line';
		var r = line.getBoundingClientRect();
		var x = r.left + 5, y = r.top + 9;   // 行内（非行首，命中文字/span 区域）
		line.dispatchEvent(new MouseEvent('mousedown', {bubbles:true, cancelable:true, detail:1, clientX:x, clientY:y}));
		line.dispatchEvent(new MouseEvent('mouseup', {bubbles:true, cancelable:true, detail:1, clientX:x, clientY:y}));
		line.dispatchEvent(new MouseEvent('click', {bubbles:true, cancelable:true, detail:1, clientX:x, clientY:y}));
		var ed = document.querySelector('.cm-editor');
		window.__focusProbe = ed ? ed.classList.contains('cm-focused') : false;
		console.log('FOCUS-PROBE cm-focused=' + window.__focusProbe);
		// ★ dump 含 .cm-cursor 的完整 CSS 规则（从 <style> 元素文本找）
		try {
			var found = [];
			var sts = document.querySelectorAll('style');
			for (var si2 = 0; si2 < sts.length; si2++) {
				var txt = sts[si2].textContent || '';
				var re = /[^{}]*\.cm-cursor[^{}]*\{[^}]*\}/g;
				var m;
				while ((m = re.exec(txt)) !== null) {
					found.push(m[0].replace(/\s+/g, ' ').slice(-160));
				}
			}
			console.log('CURSOR-RULES ' + found.join(' || '));
		} catch (e) { console.log('CURSOR-RULES ERR ' + e); }
		// ★ 查询光标 DOM 几何/显示状态（判断是否在视口内、display 是否 block）
		var cu = document.querySelector('.cm-cursor');
		var cr = cu ? cu.getBoundingClientRect() : null;
		var ccs = cu ? getComputedStyle(cu) : null;
		console.log('CURSOR-PROBE rect=' + (cr ? '{' + cr.left + ',' + cr.top + ',' + cr.width + 'x' + cr.height + '}' : 'null')
			+ ' display=' + (ccs ? ccs.display : 'null') + ' cls=' + (cu ? cu.className : 'null')
			+ ' scrollerTop=' + (document.querySelector('.cm-scroller') ? document.querySelector('.cm-scroller').scrollTop : 'null'));
		// ★ 查询选区内容（蓝色块可能=选中文本）
		var sel = window.getSelection ? window.getSelection().toString() : '';
		console.log('SEL-PROBE len=' + sel.length + ' txt=' + sel.slice(0, 40).replace(/\n/g, '\\n'));
		// ★ 查询 CM6 state 光标位置（head）——点击后光标是否跳到文档末尾
		var v = window.__editorView;
		var head = v ? v.state.selection.main.head : -1;
		var docLen = v ? v.state.doc.length : -1;
		console.log('HEAD-PROBE head=' + head + ' docLen=' + docLen + ' doc=' + (v ? JSON.stringify(v.state.doc.toString().slice(0, 60)) : 'null'));
		return 'clicked line, cm-focused=' + window.__focusProbe;
	})()`
	desktopbridge.PushMainJS(focusJS)
	log.Println("[Probe] 已投递聚焦编辑器脚本")
	// ★ 聚焦后延迟再查一次最终状态（几何桥/布局收敛后光标是否仍 rect=0、
	// 滚动是否异常）——区分「点击瞬间未同步」与「最终布局错误」。
	time.Sleep(3 * time.Second)
	finalJS := `(function(){
		var cu = document.querySelector('.cm-cursor');
		var cr = cu ? cu.getBoundingClientRect() : null;
		var v = window.__editorView;
		var sc = document.querySelector('.cm-scroller');
		console.log('FINAL-PROBE cursor=' + (cr ? '{' + cr.left + ',' + cr.top + ',' + cr.width + 'x' + cr.height + '}' : 'null')
			+ ' scrollerTop=' + (sc ? sc.scrollTop : 'null')
			+ ' head=' + (v ? v.state.selection.main.head : -1)
			+ ' cm-focused=' + (document.querySelector('.cm-editor') ? document.querySelector('.cm-editor').classList.contains('cm-focused') : false));
		return 'final';
	})()`
	desktopbridge.PushMainJS(finalJS)
	time.Sleep(2 * time.Second)
	// 手动渲染多帧保存 PNG（覆盖光标闪烁相位——1200ms 周期 50% 隐藏，
	// 至少一张含光标）。wv.Render() 是纯 Go paint（不动 JS），与主循环
	// 并发读渲染树绘制，输出独立 PNG（几何与窗口一致）。
	for i := 0; i < 8; i++ {
		if pngBytes, err := wv.Render(); err == nil {
			_ = os.WriteFile(fmt.Sprintf("_probe_editor_%d.png", i), pngBytes, 0644)
			log.Printf("[Probe] 已保存 _probe_editor_%d.png (%d 字节)", i, len(pngBytes))
		} else {
			log.Printf("[Probe] Render %d err: %v", i, err)
		}
		time.Sleep(250 * time.Millisecond)
	}
	if data, err := os.ReadFile("_layout_snap.log"); err == nil {
		_ = os.WriteFile("_probe_editor_snap.log", data, 0644)
		log.Printf("[Probe] 已收集 _layout_snap.log → _probe_editor_snap.log (%d 字节)", len(data))
	} else {
		log.Printf("[Probe] 读取 _layout_snap.log 失败: %v（需 WB_SNAP=1）", err)
	}
	if _, err := os.Stat("_canvas_dump.png"); err == nil {
		log.Printf("[Probe] 窗口 PNG: _canvas_dump.png")
	}
	log.Println("[Probe] 完成，退出")
	os.Exit(0)
}

// diagTerm 查询终端 buffer 内容与行 span 渲染位置（WB_TERM_DIAG 诊断）。
// ★ round 循环驱动 focus/blur：round%3==0 初始、==1 聚焦、==2 失焦，
// 对比三个状态下行 span 宽度是否一致（「聚焦/失焦空格间距不一致」排查）。
func diagTerm(wv *webkit.WebView, round int) {
	if interp := wv.JSInterpreter(); interp != nil {
		// ★ round 不能走 Sprintf 注入（js 里 round % 3 的 % 会冲突），
		// 先用 window.__diagRound 传值。
		_, _ = interp.RunJS(fmt.Sprintf("window.__diagRound = %d", round))
		js := `(function(){
			var round = window.__diagRound || 0;
			var t = window.__lastTerm;
			if (!t || !t.buffer || !t.buffer.active) return 'no-term';
			var focusState = t.element ? t.element.classList.contains('focus') : null;
			// round 驱动 focus/blur 切换（每 90 帧 ≈1.5s 一次）
			if (round % 3 === 1) { try { if (t.focus) t.focus(); } catch(e){} }
			if (round % 3 === 2) { try { if (t.blur) t.blur(); } catch(e){} }
			var lines = [];
			var n = Math.min(t.buffer.active.length, 12);
			for (var i = 0; i < n; i++) {
				var ln = t.buffer.active.getLine(i);
				lines.push((ln ? ln.translateToString(true) : '<null>'));
			}
			var rowsEl = t.element && t.element.querySelector('.xterm-rows');
			var rows = [];
			if (rowsEl) {
				var kids = rowsEl.children;
				var nn = Math.min(kids.length, 6);
				for (var j = 0; j < nn; j++) {
					var r = kids[j].getBoundingClientRect();
					var spans = [];
					var sk = kids[j].children;
					for (var s2 = 0; s2 < (sk ? sk.length : 0) && s2 < 40; s2++) {
						var kr = sk[s2].getBoundingClientRect();
						spans.push((sk[s2].textContent || '').slice(0,12) + ':' + Math.round(kr.width*1000)/1000 + '@' + Math.round(kr.left*1000)/1000);
					}
					rows.push({top: Math.round(r.top*10)/10, h: Math.round(r.height*1000)/1000,
					           text: (kids[j].textContent || '').slice(0, 40), spans: spans});
				}
			}
			var xe = t.element;
			var xr = xe ? xe.getBoundingClientRect() : null;
			var pad = xe ? getComputedStyle(xe).padding : '';
			var c = t._core ? (t._core._renderService && t._core._renderService.dimensions) : null;
			return JSON.stringify({
				round: round, focus: focusState,
				rows: t.rows, cols: t.cols,
				xterm: xr ? {top: Math.round(xr.top), h: Math.round(xr.height), w: Math.round(xr.width)} : null,
				pad: pad,
				cellH: c ? c.css.cell.height : null,
				cellW: c ? c.css.cell.width : null,
				viewportTop: t.buffer.active.viewportY,
				lines: lines,
				rowsInfo: rows
			});
})()`
		if v, err := interp.RunJS(js); err == nil {
			log.Printf("[TERM-DIAG %d] %s", round, v.ToString())
		} else {
			log.Printf("[TERM-DIAG %d] err: %v", round, err)
		}
	}
}

func setupLoaders(wv *webkit.WebView, distDir string) {
	absDist, _ := filepath.Abs(distDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				log.Printf("[SCRIPT] src=%q len=%d err=%v", src, len(data), err)
				return string(data), err
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				if err != nil {
					return "", err
				}
				re := regexp.MustCompile(`\[data-v-[a-f0-9]+\]`)
				cleaned := re.ReplaceAllString(string(data), "")
				return cleaned, nil
			}
		}
	}
}

func writeRenderDiagnostic(wv *webkit.WebView) {
	rv := wv.RenderView()
	if rv == nil {
		log.Println("[DIAG] RenderView is nil")
		return
	}
	state := rv.LayoutState()

	// Debug: verify rp-body width right before the diagnostic dump.
	func() {
		var w2 func(ro rendering.RenderObject)
		w2 = func(ro rendering.RenderObject) {
			if ro == nil {
				return
			}
			if n := ro.Node(); n != nil {
				if el, ok := n.(*dom.Element); ok {
					if cls := el.GetAttribute("class"); cls == "rp-body" || cls == "right-panel" || cls == "file-explorer" {
						lb := ro.LayoutBox()
						fn := "nil"
						if lb != nil && state != nil {
							g := state.GeometryForBox(lb)
							fn = fmt.Sprintf("%.0fx%.0f", g.BorderBoxWidth(), g.BorderBoxHeight())
						}
						fmt.Fprintf(os.Stderr, "[PREDUMP] cls=%s ro=%p lb=%p lb.rect=%s\n",
							cls, ro, lb, fn)
					}
				}
			}
			if lb := ro.LayoutBox(); lb != nil && state != nil {
				g := state.GeometryForBox(lb)
				if g.BorderBoxWidth() == 0 && g.BorderBoxHeight() > 0 {
					name := "?"
					if n := ro.Node(); n != nil {
						if el, ok := n.(*dom.Element); ok {
							name = el.LocalName()
							if cls := el.GetAttribute("class"); cls != "" {
								name += "." + cls
							}
						} else {
							name = n.NodeName()
						}
					}
					fmt.Fprintf(os.Stderr, "[PREDUMP:w0] %s ro=%p lb=%p lb.rect=%.0fx%.0f (%.0f,%.0f)\n",
						name, ro, lb, g.BorderBoxWidth(), g.BorderBoxHeight(), g.Left(), g.Top())
				}
			}
			for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
				w2(c)
			}
		}
		w2(rv)
	}()

	f, err := os.Create("desktop_diag.log")
	if err != nil {
		log.Printf("[DIAG] Cannot create log: %v", err)
		return
	}
	defer f.Close()

	fmt.Fprintln(f, "=== DESKTOP RENDER DIAGNOSTIC ===")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "=== RENDER TREE ===")
	dumpRO(f, rv, 0, state)
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "=== ANOMALY ANALYSIS ===")
	reportAnomalies(f, rv, state)
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "=== DIAGNOSTIC COMPLETE ===")
	log.Println("[DIAG] Wrote desktop_diag.log")
}

func reportAnomalies(f *os.File, ro rendering.RenderObject, state *layout.LayoutState) {
	type anomaly struct {
		desc string
		info string
	}
	var ans []anomaly
	var walk func(ro rendering.RenderObject, depth int)
	walk = func(ro rendering.RenderObject, depth int) {
		lb := ro.LayoutBox()
		if lb != nil && state != nil {
			g := state.GeometryForBox(lb)
			name := ro.RenderName()
			if g.Top() < -1 {
				ans = append(ans, anomaly{"负 Y 坐标", fmt.Sprintf("%s y=%.0f", name, g.Top())})
			}
			if g.Left() < -1 {
				ans = append(ans, anomaly{"负 X 坐标", fmt.Sprintf("%s x=%.0f", name, g.Left())})
			}
			if g.Left() > 1280 {
				ans = append(ans, anomaly{"越界 X > viewport", fmt.Sprintf("%s x=%.0f w=%.0f", name, g.Left(), g.BorderBoxWidth())})
			}
			if g.BorderBoxWidth() == 0 && ro.Style() != nil && ro.Style().Display != 0 {
				cs := ro.Style()
				if cs.BackgroundColor.A > 0 || cs.Width.Value > 0 {
					ans = append(ans, anomaly{"零宽但有背景/宽度", fmt.Sprintf("%s w=0 bg=#%02x%02x%02x", name, cs.BackgroundColor.R, cs.BackgroundColor.G, cs.BackgroundColor.B)})
					fmt.Fprintf(os.Stderr, "[W0] %s p=%p w=%.0f h=%.0f x=%.0f y=%.0f\n",
						name, lb, g.BorderBoxWidth(), g.BorderBoxHeight(), g.Left(), g.Top())
				}
			}
		}
		for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c, depth+1)
		}
	}
	walk(ro, 0)
	if len(ans) == 0 {
		fmt.Fprintln(f, "  无异常")
		return
	}
	for _, a := range ans {
		fmt.Fprintf(f, "  [%s] %s\n", a.desc, a.info)
	}
}

// ★ 全局异常捕获并抛出：增量转储 JS 运行时错误与 console 输出。
// window.__errs 由 LoadHTML 后注入的 hook（error / unhandledrejection /
// console.error）持续收集；console buffer 由 jsc.BufferLogger 持续累积
// （String() 返回全量、不清空），用长度差取增量——此前只在 LoadHTML 后
// 读一次 ConsoleOutput()，运行期的 JS 异常全被静默吞掉（「齿轮绘制异常 /
// 打不开文件」等无法定位）。
var (
	errWatchFrame  int
	lastErrsCount  int
	lastConsoleLen int
)

func watchJSErrors(wv *webkit.WebView) {
	interp := wv.JSInterpreter()
	if interp == nil {
		return
	}
	// 1) window.__errs 增量（JS error / unhandledrejection / console.error）
	if v, err := interp.RunJS(`(window.__errs ? window.__errs.length : 0)`); err == nil {
		n := int(v.ToNumber())
		if n > lastErrsCount {
			if ev, err := interp.RunJS(fmt.Sprintf("window.__errs.slice(%d).join('\\n')", lastErrsCount)); err == nil {
				text := ev.ToString()
				log.Printf("[JS-ERROR] %s", text)
				appendErrLog("JS-ERROR", text)
			}
			lastErrsCount = n
		}
	}
	// 2) console buffer 增量（console.log / warn / error 全部输出）
	out := wv.ConsoleOutput()
	if len(out) > lastConsoleLen {
		seg := out[lastConsoleLen:]
		log.Printf("[CONSOLE] %s", seg)
		appendErrLog("CONSOLE", seg)
		lastConsoleLen = len(out)
	}
}

// appendErrLog 追加统一错误日志 _desktop_errors.log（带时间戳与分类），
// 与 _desktop_panic.log（panic 专用）互补——非 panic 的 JS/渲染错误
// 也全部落盘，实机复现后直接翻这一个文件。
func appendErrLog(kind, msg string) {
	f, err := os.OpenFile("_desktop_errors.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "=== [%s] %s ===\n%s\n", kind, time.Now().Format("2006-01-02 15:04:05"), msg)
}

func dumpRO(f *os.File, ro rendering.RenderObject, depth int, state *layout.LayoutState) {
	if ro == nil {
		return
	}
	prefix := strings.Repeat("  ", depth)
	name := ro.RenderName()

	cnt := 0
	for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
		cnt++
	}

	cs := ro.Style()
	lb := ro.LayoutBox()
	if rt, ok := ro.(*rendering.RenderText); ok {
		// RenderText 没有独立 layout box（WebKit 语义：几何在
		// InlineTextBox segments 里，由 inline formatting context 生成）。
		// 用 layoutBox 判断会误报 [no layout]——必须直接看 segs。
		segs := rt.Segments()
		txt := rt.OriginalText()
		if len(txt) > 12 {
			txt = txt[:12] + "..."
		}
		if len(segs) > 0 {
			fmt.Fprintf(f, "%sRenderText segs=%d first=(x=%.0f,y=%.0f w=%.0f h=%.0f) text=%q\n",
				prefix, len(segs), segs[0].X, segs[0].Y, segs[0].Width, segs[0].Height, txt)
		} else {
			fmt.Fprintf(f, "%sRenderText segs=0 text=%q [no inline layout]\n", prefix, txt)
		}
	} else if lb != nil && state != nil {
		g := state.GeometryForBox(lb)
		bgStr := ""
		dispStr := ""
		if cs != nil {
			if cs.BackgroundColor.A > 0 {
				bgStr = fmt.Sprintf(" bg=#%02x%02x%02x", cs.BackgroundColor.R, cs.BackgroundColor.G, cs.BackgroundColor.B)
			}
			dispStr = fmt.Sprintf(" disp=%d", cs.Display)
		}
		fmt.Fprintf(f, "%s%s (%d ch) x=%.0f y=%.0f w=%.0f h=%.0f p=%p%s%s",
			prefix, name, cnt, g.Left(), g.Top(), g.BorderBoxWidth(), g.BorderBoxHeight(), lb, dispStr, bgStr)
		fmt.Fprintln(f)
	} else {
		fmt.Fprintf(f, "%s%s (%d ch) [no layout]\n", prefix, name, cnt)
	}

	for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
		dumpRO(f, c, depth+1, state)
	}
}
