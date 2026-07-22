package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"wb-ui/app"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(log.Ltime)
	log.Println("[Desktop] PairCode IDE 桌面版 v1.0.6-desktop")

	// Determine the dist directory: look for ../web-ui/dist relative to the binary.
	binDir := filepath.Dir(os.Args[0])
	distDir := filepath.Join(binDir, "web-ui", "dist")
	altDist := filepath.Join(binDir, "..", "cmd", "desktop", "web-ui", "dist")
	altDist2 := filepath.Join(binDir, "..", "..", "cmd", "desktop", "web-ui", "dist")
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		if _, err2 := os.Stat(altDist); err2 == nil {
			distDir = altDist
		} else if _, err2 := os.Stat(altDist2); err2 == nil {
			distDir = altDist2
		} else {
			absBin, _ := filepath.Abs(binDir)
			log.Fatalf("[Desktop] cannot find web-ui/dist relative to binary at %s", absBin)
		}
	}
	log.Printf("[Desktop] distDir: %s", distDir)

	// Read index.html.
	htmlData, err := os.ReadFile(distDir + "/index.html")
	if err != nil {
		log.Fatalf("[Desktop] cannot read index.html: %v", err)
	}
	htmlStr := string(htmlData)

	// Create WebView.
	wv := webkit.NewWebView()

	// Register desktop bridge handlers.
	InitDesktopBridge(wv)

	// Load the page.
	wv.LoadHTML(htmlStr)

	log.Println("[LoadHTML] 加载成功")

	// Log any console output (look for Vue diagnostics from the app).
	if out := wv.ConsoleOutput(); out != "" {
		log.Println("[CONSOLE]")
		log.Println(out)
	}

	// Ensure layout + paint before creating the window.
	wv.EnsureLayout()
	writeRenderDiagnostic(wv)

	log.Println("[Desktop] window+render tree ready, creating host...")

	// Create the app host (window). The host calls wv.EvalJS for the
	// bridge-ready callback which may cause a second rebuild+layout.
	host, err := app.NewHost(wv, 1280, 800, "PairCode IDE")
	if err != nil {
		log.Fatalf("[Desktop] 创建窗口失败: %v", err)
	}

	// After the host is created, layout once more so the window canvas is clean.
	wv.EnsureLayout()
	writeRenderDiagnostic(wv)

	log.Println("[Desktop] 窗口已启动，开始事件循环...")

	// Start the host event loop (blocking).
	host.Run()

	log.Println("[Desktop] 已退出。")
}

func writeRenderDiagnostic(wv *webkit.WebView) {
	rv := wv.RenderView()
	if rv == nil {
		log.Println("[DIAG] RenderView is nil")
		return
	}

	f, err := os.Create("desktop_diag.log")
	if err != nil {
		log.Printf("[DIAG] Cannot create log: %v", err)
		return
	}
	defer f.Close()

	fmt.Fprintln(f, "=== DESKTOP RENDER DIAGNOSTIC ===")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "=== RENDER TREE ===")
	dumpRO(f, rv, 0)
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "=== ANOMALY ANALYSIS ===")
	reportAnomalies(f, rv)
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "=== ANOMALY ANALYSIS ===")
	reportAnomalies(f, rv)
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "=== DIAGNOSTIC COMPLETE ===")
	log.Println("[DIAG] Wrote desktop_diag.log")
}

func reportAnomalies(f *os.File, ro rendering.RenderObject) {
	type anomaly struct {
		desc string
		info string
	}
	var ans []anomaly
	var walk func(ro rendering.RenderObject, depth int)
	walk = func(ro rendering.RenderObject, depth int) {
		lb := ro.LayoutBox()
		if lb != nil {
			name := ro.RenderName()
			if lb.Rect.Y < -1 {
				ans = append(ans, anomaly{"负 Y 坐标", fmt.Sprintf("%s y=%.0f", name, lb.Rect.Y)})
			}
			if lb.Rect.X < -1 {
				ans = append(ans, anomaly{"负 X 坐标", fmt.Sprintf("%s x=%.0f", name, lb.Rect.X)})
			}
			if lb.Rect.X > 1280 {
				ans = append(ans, anomaly{"越界 X > viewport", fmt.Sprintf("%s x=%.0f w=%.0f", name, lb.Rect.X, lb.Rect.Width)})
			}
			if lb.Rect.Width == 0 && ro.Style() != nil && ro.Style().Display != 0 {
				cs := ro.Style()
				if cs.BackgroundColor.A > 0 || cs.Width.Value > 0 {
					ans = append(ans, anomaly{"零宽但有背景/宽度", fmt.Sprintf("%s w=0 bg=#%02x%02x%02x", name, cs.BackgroundColor.R, cs.BackgroundColor.G, cs.BackgroundColor.B)})
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

func dumpRO(f *os.File, ro rendering.RenderObject, depth int) {
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
	if lb != nil {
		bgStr := ""
		dispStr := ""
		if cs != nil {
			if cs.BackgroundColor.A > 0 {
				bgStr = fmt.Sprintf(" bg=#%02x%02x%02x", cs.BackgroundColor.R, cs.BackgroundColor.G, cs.BackgroundColor.B)
			}
			dispStr = fmt.Sprintf(" disp=%d", cs.Display)
		}
		fmt.Fprintf(f, "%s%s (%d ch) x=%.0f y=%.0f w=%.0f h=%.0f%s%s",
			prefix, name, cnt, lb.Rect.X, lb.Rect.Y, lb.Rect.Width, lb.Rect.Height, dispStr, bgStr)
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
		dumpRO(f, c, depth+1)
	}
}
