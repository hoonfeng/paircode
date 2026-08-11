// Command perf_editor 剖析 CM6 编辑器「键盘输入 → 渲染」全链耗时：
//   MockKeyChar（handleCharInput）→ JS 微任务（CM6 readDOMChange）
//   → RebuildRenderTree → EnsureLayout → Render(paint)
// 输出每阶段耗时分布，定位「事件响应极慢」的瓶颈。
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/app"
	"wb-ui/bindings"
	"wb-ui/dom"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

func runJobs(wv *webkit.WebView) {
	for i := 0; i < 6; i++ {
		interp := wv.JSInterpreter()
		interp.RunJobs()
		if el := interp.GetEventLoop(); el != nil {
			el.ProcessTasks(0)
		}
		interp.RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
	}
}

func js(wv *webkit.WebView, code string) string {
	iv, err := wv.JSInterpreter().RunJS(code)
	if err != nil {
		return "[ERR] " + err.Error()
	}
	return iv.ToString()
}

func ms(d time.Duration) float64 { return float64(d.Nanoseconds()) / 1e6 }

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
	wv.LoadHTML(string(htmlData))
	t0 := time.Now()
	for i := 0; i < 6; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
		time.Sleep(400 * time.Millisecond)
		runJobs(wv)
	}
	fmt.Printf("[boot] JS 加载+首轮渲染 = %.1fms\n", ms(time.Since(t0)))

	// 注入打开 Go 文件
	code := `package main

import "fmt"

// main is the entry point.
func main() {
	fmt.Println("hello")
}

# hash comment test
func add(a, b int) int {
	return a + b
}
`
	code = strings.ReplaceAll(code, "\\", "\\\\")
	code = strings.ReplaceAll(code, "\n", "\\n")
	code = strings.ReplaceAll(code, "'", "\\'")
	js(wv, `(function(){
		var p = '/workspace/main.go';
		var st = window.__state;
		st.openFiles = [p];
		st.activeFile = p;
		st.fileContents[p] = '`+code+`';
		return 'injected';
	})()`)
	runJobs(wv)
	time.Sleep(100 * time.Millisecond)
	runJobs(wv)

	// 首次完整渲染计时
	t0 = time.Now()
	wv.RebuildRenderTree()
	fmt.Printf("[render0] RebuildRenderTree = %.2fms\n", ms(time.Since(t0)))
	t0 = time.Now()
	wv.EnsureLayout()
	fmt.Printf("[render0] EnsureLayout     = %.2fms\n", ms(time.Since(t0)))
	t0 = time.Now()
	_, err = wv.Render()
	if err != nil {
		log.Fatalf("render: %v", err)
	}
	fmt.Printf("[render0] Render(paint)    = %.2fms\n", ms(time.Since(t0)))

	// 输入前确认 selection 有效（collapse 到行 1 文本 offset 2）
	js(wv, `(function(){
		var l = document.querySelectorAll('.cm-line')[0];
		var w = document.createTreeWalker(l, NodeFilter.SHOW_TEXT);
		var t = null;
		while (w.nextNode()) { t = w.currentNode; break; }
		if (!t) return 'noText';
		var s = window.getSelection();
		s.collapse(t, 2);
		return 'sel:' + s.rangeCount;
	})()`)

	// ★ 8 次键盘输入全链计时
	fmt.Println("== 键盘输入全链（8 次）==")
	bindings.ResetDOMStats()
	var sums = map[string]float64{}
	count := 0
	h := app.NewHostForTest(wv, 1280, 800)
	cmEl, err := findContentEditable(wv)
	if err != nil {
		log.Fatalf("no cm-content: %v", err)
	}
	h.MockFocus(cmEl)
	for i := 0; i < 8; i++ {
		tk := time.Now()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		// ① MockKeyChar（handleCharInput：InsertTextAtSelection + input 事件）
		tk = time.Now()
		h.MockKeyChar(rune('A' + i))
		dInput := ms(time.Since(tk))
		// ② JS 微任务（CM6 readDOMChange / observer）+ 宏任务（rAF/measure）
		tk = time.Now()
		interp := wv.JSInterpreter()
		interp.RunJobs()
		dMicro1 := ms(time.Since(tk))
		tk = time.Now()
		if el := interp.GetEventLoop(); el != nil {
			el.ProcessTasks(0)
		}
		dMacro := ms(time.Since(tk))
		tk = time.Now()
		interp.RunJobs()
		dMicro2 := ms(time.Since(tk))
		dJS := dMicro1 + dMacro + dMicro2
		// ③ 渲染树重建
		tk = time.Now()
		wv.RebuildRenderTree()
		dRebuild := ms(time.Since(tk))
		// ④ 布局
		tk = time.Now()
		wv.EnsureLayout()
		dLayout := ms(time.Since(tk))
		// ⑤ paint
		tk = time.Now()
		_, _ = wv.Render()
		dPaint := ms(time.Since(tk))
		total := dInput + dJS + dRebuild + dLayout + dPaint
		sums["input"] += dInput
		sums["js"] += dJS
		sums["micro1"] += dMicro1
		sums["macro"] += dMacro
		sums["micro2"] += dMicro2
		sums["rebuild"] += dRebuild
		sums["layout"] += dLayout
		sums["paint"] += dPaint
		sums["total"] += total
		count++
		fmt.Printf("  #%d total=%6.2fms (input=%4.2f micro1=%6.2f macro=%6.2f micro2=%5.2f rebuild=%5.2f layout=%5.2f paint=%5.2f)\n",
			i, total, dInput, dMicro1, dMacro, dMicro2, dRebuild, dLayout, dPaint)
	}
	if count > 0 {
		fmt.Printf("== 平均: input=%.2f micro1=%.2f macro=%.2f micro2=%.2f rebuild=%.2f layout=%.2f paint=%.2f total=%.2fms ==\n",
			sums["input"]/float64(count), sums["micro1"]/float64(count), sums["macro"]/float64(count), sums["micro2"]/float64(count),
			sums["rebuild"]/float64(count), sums["layout"]/float64(count), sums["paint"]/float64(count),
			sums["total"]/float64(count))
	}
	// 最终确认文本
	fmt.Println("[final] " + js(wv, `(function(){
		var co = document.querySelector('.cm-content');
		if (!co) return 'NO';
		var t = co.textContent;
		return 'len=' + t.length + ' head=[' + t.slice(0, 30) + ']';
	})()`))
	bindings.DumpDOMStats()
}

func findContentEditable(wv *webkit.WebView) (*dom.Element, error) {
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			if doc := fr.Document(); doc != nil {
				els := doc.GetElementsByClassName("cm-content")
				if len(els) > 0 {
					return els[0], nil
				}
				return nil, fmt.Errorf("no cm-content element")
			}
		}
	}
	return nil, fmt.Errorf("no document")
}
