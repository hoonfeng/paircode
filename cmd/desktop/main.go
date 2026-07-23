package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/app"
	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(log.Ltime)
	log.Println("[Desktop] PairCode IDE 桌面版 v1.0.6-desktop")

	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "desktop", "web-ui", "dist")
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

	wv.LoadHTML(htmlStr)

	InitDesktopBridge(wv)

	if out := wv.ConsoleOutput(); out != "" {
		log.Println("[CONSOLE]")
		log.Println(out)
	}

	log.Println("[LoadHTML] 加载成功")
	log.Println("[Desktop] window+render tree ready, creating host...")

	host, err := app.NewHost(wv, 1280, 800, "PairCode IDE")
	if err != nil {
		log.Fatalf("[Desktop] 创建窗口失败: %v", err)
	}

	wv.EnsureLayout()
	writeRenderDiagnostic(wv)

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
