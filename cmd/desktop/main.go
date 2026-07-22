package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"wb-ui/app"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(log.Ltime)
	log.Println("[Desktop] PairCode IDE 桌面版 v1.0.6-desktop")

	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "web-ui", "dist")
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		distDir = filepath.Join(wd, "cmd", "desktop", "web-ui", "dist")
	}
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		distDir2 := filepath.Join(wd, "web-ui-minimal", "dist")
		if _, err2 := os.Stat(distDir2); err2 == nil {
			distDir = distDir2
		} else {
			distDir3 := filepath.Join(wd, "cmd", "desktop", "web-ui-minimal", "dist")
			if _, err3 := os.Stat(distDir3); err3 == nil {
				distDir = distDir3
			}
		}
	}
	log.Printf("[Desktop] distDir: %s", distDir)

	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()

	htmlData, err := os.ReadFile(distDir + "/index.html")
	s := string(htmlData)
	s = strings.Replace(s, `type="module"`, "", 1)
	s = strings.ReplaceAll(s, `crossorigin`, "")
	log.Printf("[LoadHTML] 开始加载, 大小=%d", len(s))
	err = wv.LoadHTML(s)
	if err != nil {
		log.Printf("[LoadHTML] 错误: %v", err)
	} else {
		log.Printf("[LoadHTML] 加载成功")
	}
	if out := wv.ConsoleOutput(); out != "" {
		log.Printf("[CONSOLE]\n%s", out)
	}
	InitDesktopBridge(wv)

	for i := 0; i < 8; i++ {
		wv.EnsureLayout()
		time.Sleep(100 * time.Millisecond)
	}
	wv.RebuildRenderTree()
	wv.EvalJS(`(function(){var a=document.getElementById('app');if(a&&a.childElementCount>0)console.log('[VUE] OK, innerHTML.len='+a.innerHTML.length);else console.log('[VUE] ERR, app is empty')})()`)
	if out := wv.ConsoleOutput(); out != "" {
		log.Printf("[CONSOLE2]\n%s", out)
	}

	log.Println("[Desktop] window+render tree ready, creating host...")
	wv.EnsureLayout()
	dumpRenderDiagnostic(wv)

	host, err := app.NewHost(wv, 1280, 800, "PairCode IDE")
	if err != nil {
		log.Printf("[Desktop] NewHost error: %v", err)
		return
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
				data, err := os.ReadFile(filepath.Join(absDist, strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")))
				log.Printf("[SCRIPT] len=%d err=%v", len(data), err)
				return string(data), err
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				data, _ := os.ReadFile(filepath.Join(absDist, strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")))
				re := regexp.MustCompile(`\[data-v-[a-f0-9]+\]`)
				cleaned := re.ReplaceAllString(string(data), "")
				return cleaned, nil
			}
		}
	}
}

func dumpRenderDiagnostic(wv *webkit.WebView) {
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
	fmt.Fprintln(f, "=== DIAGNOSTIC COMPLETE ===")
	log.Println("[DIAG] Wrote desktop_diag.log")
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
