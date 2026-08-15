//go:build ignore

package main

// Command settings_textarea_probe 验证「指令」「思想」tab 的 textarea 尺寸：
//   textarea 固有高度 = rows × lineHeight + padding + border（浏览器标准）。
//   回归点：intrinsicContentHeight 之前只返回单行高度，rows=6 被压成一行 ≈30px。
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
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func taRunJobs(wv *webkit.WebView) {
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

func taJs(wv *webkit.WebView, code string) string {
	iv, err := wv.JSInterpreter().RunJS(code)
	if err != nil {
		return "[ERR] " + err.Error()
	}
	return iv.ToString()
}

func taLayout(wv *webkit.WebView) {
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	taRunJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
}

// taBoxByClass 遍历元素找 class 含关键词的 box 几何。
func taBoxByClass(wv *webkit.WebView, doc *dom.Document, classSub string) (float64, float64, bool) {
	rv := wv.RenderView()
	for _, el := range doc.GetElementsByTagName("div") {
		if strings.Contains(el.GetAttribute("class"), classSub) {
			if b := rv.FindRenderBoxForNode(el); b != nil {
				return b.Width(), b.Height(), true
			}
		}
	}
	return 0, 0, false
}

// taReport 输出所有 .inst-textarea 的几何（rows 属性 + box 高度/宽度）。
func taReport(wv *webkit.WebView, label string) {
	fmt.Printf("[%s] inst-textarea 几何:\n", label)
	doc := wv.Document()
	if doc == nil {
		fmt.Println("  document nil")
		return
	}
	els := doc.GetElementsByTagName("textarea")
	if len(els) == 0 {
		fmt.Println("  无 textarea")
		return
	}
	rv := wv.RenderView()
	for i, el := range els {
		rows := el.GetAttribute("rows")
		box := rv.FindRenderBoxForNode(el)
		if box == nil {
			fmt.Printf("  [%d] rows=%s → 无 box\n", i, rows)
			continue
		}
		h := box.Height()
		w := box.Width()
		fmt.Printf("  [%d] rows=%s box=%dx%d (期望 rows=6≈108, rows=3≈61, rows=2≈60, 修复前≈30)\n",
			i, rows, int(w), int(h))
	}
	// modal 与 settings-body 尺寸（窗口尺寸回归）
	if w, h, ok := taBoxByClass(wv, doc, "settings-modal"); ok {
		fmt.Printf("  .settings-modal box=%dx%d\n", int(w), int(h))
	}
	if w, h, ok := taBoxByClass(wv, doc, "modal-body"); ok {
		fmt.Printf("  .modal-body box=%dx%d\n", int(w), int(h))
	}
	if w, h, ok := taBoxByClass(wv, doc, "settings-body"); ok {
		fmt.Printf("  .settings-body box=%dx%d\n", int(w), int(h))
	}
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
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
		time.Sleep(600 * time.Millisecond)
		taRunJobs(wv)
	}
	taLayout(wv)

	// 打开设置面板
	_ = taJs(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		if (btn) { var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev); }
		return 'clicked';
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	taLayout(wv)

	// 切「指令」tab（index 5）
	_ = taJs(wv, `(function(){
		var btns = document.querySelectorAll('.settings-tabs button');
		if (btns.length > 5) { var ev = new Event('click', {bubbles:true}); btns[5].dispatchEvent(ev); }
		return 'tab5';
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	taLayout(wv)
	taReport(wv, "指令")

	// 切「思想」tab（index 6）
	_ = taJs(wv, `(function(){
		var btns = document.querySelectorAll('.settings-tabs button');
		if (btns.length > 6) { var ev = new Event('click', {bubbles:true}); btns[6].dispatchEvent(ev); }
		return 'tab6';
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	taLayout(wv)
	// 启用思想注入（v-if 控制 rows=3/rows=2 textarea 渲染）
	fmt.Println("[phil] 勾选「启用思想注入」:")
	_ = taJs(wv, `(function(){
		var cb = document.querySelector('.settings-modal .setting-group input[type="checkbox"]');
		if (cb) { cb.checked = true; var ev = new Event('change', {bubbles:true}); cb.dispatchEvent(ev); return 'checked'; }
		return 'no cb';
	})()`)
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
	time.Sleep(500 * time.Millisecond)
	taLayout(wv)
	taReport(wv, "思想")
	fmt.Println("DONE")
}
