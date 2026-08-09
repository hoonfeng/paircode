package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"wb-ui/app"
	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/rendering"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func main() {
	log.SetFlags(log.Ltime)
	log.Println("[Desktop] PairCode IDE 桌面版 v1.0.6-desktop")

	// ★ 崩溃捕获：任何 panic 都写堆栈到 _desktop_panic.log（含所有 goroutine），
	//   方便实机复现"打开文件崩溃"时拿到完整现场。
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 1<<20)
			n := runtime.Stack(buf, true)
			msg := fmt.Sprintf("=== PANIC %v ===\n%s\n", r, buf[:n])
			_ = os.WriteFile("_desktop_panic.log", []byte(msg), 0644)
			log.Printf("[Desktop] PANIC: %v\n%s", r, buf[:n])
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

	// ★ 每帧回调：主循环消费外部队列的 JS 推送（终端 PTY 输出、agent
	// 事件）——goja 非线程安全，所有跨 goroutine 的 RunJS 必须在此
	// 主线程执行。
	host.OnFrame = func() {
		desktopbridge.DrainMainQueue(wv)
	}

	log.Println("[Desktop] 窗口已启动，开始事件循环...")
	host.Run()
	log.Println("[Desktop] 已退出。")
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
	if lb != nil && state != nil {
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
		if rt, ok := ro.(*rendering.RenderText); ok {
			segs := rt.Segments()
			if len(segs) > 0 {
				fmt.Fprintf(f, " segs=%d[W=%.0f H=%.0f]", len(segs), segs[0].Width, segs[0].Height)
			} else {
				fmt.Fprint(f, " segs=0")
			}
		}
		fmt.Fprintln(f)
	} else {
		fmt.Fprintf(f, "%s%s (%d ch) [no layout]\n", prefix, name, cnt)
	}

	for c := ro.FirstChild(); c != nil; c = c.NextSibling() {
		dumpRO(f, c, depth+1, state)
	}
}
