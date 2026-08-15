// Command ime_editor_probe 验证 CM6 编辑器 IME 输入链路：
//   DOM Selection 定位 → applyIMEEvents 组合（compositionupdate）+ 提交（char）
//   → 检查文本插入位置（是否光标处/光标后）与 CM6 state 同步。
// 目标：复现用户报告「IME 文本插入到光标前（不是往后排）」。
//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/app"
	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/platform/ime"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func js(wv *webkit.WebView, code string) string {
	v, err := wv.JSInterpreter().RunJS(code)
	if err != nil {
		return "JSERR:" + err.Error()
	}
	return v.ToString()
}

func runJobs(wv *webkit.WebView) {
	for i := 0; i < 8; i++ {
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
		time.Sleep(15 * time.Millisecond)
	}
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
	for i := 0; i < 6; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
		time.Sleep(450 * time.Millisecond)
		runJobs(wv)
	}

	// 注入打开文件：内容含明确可定位的多行文本
	code := `package main

func main() {
	// TODO: implement
	fmt.Println("hello world")
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
	time.Sleep(200 * time.Millisecond)
	runJobs(wv)

	fmt.Println("[init] " + js(wv, `(function(){
		var co = document.querySelector('.cm-content');
		if (!co) return 'NO cm-content';
		var lines = co.querySelectorAll('.cm-line');
		var out = ['lines=' + lines.length];
		for (var i = 0; i < lines.length && i < 3; i++) {
			out.push('L' + i + '=[' + lines[i].textContent.slice(0, 30) + ']');
		}
		var v = window.__editorView;
		if (v) out.push('docLen=' + v.state.doc.length);
		return out.join(' | ');
	})()`))

	// 定位光标到第 3 行（"fmt.Println"）中间字符：用 collapse 到文本节点
	fmt.Println("[sel] " + js(wv, `(function(){
		var co = document.querySelector('.cm-content');
		var lines = co.querySelectorAll('.cm-line');
		if (lines.length < 3) return 'need>=3 lines, got ' + lines.length;
		var line = lines[2]; // "fmt.Println(\"hello world\")"
		var walker = document.createTreeWalker(line, NodeFilter.SHOW_TEXT);
		var target = null;
		while (walker.nextNode()) { target = walker.currentNode; if (target.nodeValue.length > 0) break; }
		if (!target) return 'no text node';
		var off = Math.min(4, target.nodeValue.length); // 第4个字符后
		var s = window.getSelection();
		s.collapse(target, off);
		return 'target=[' + target.nodeValue.slice(0, 20) + '] off=' + off + ' range0=' + s.rangeCount;
	})()`))
	runJobs(wv)

	host := app.NewHostForTest(wv, 1280, 800)
	cmEl, err := findContentEditable(wv)
	if err != nil {
		log.Fatalf("no cm-content: %v", err)
	}
	host.MockFocus(cmEl)
	fmt.Printf("[focus] contenteditable=%q\n", cmEl.GetAttribute("contenteditable"))

	// ★ 模拟 IME 组合：compositionupdate "拼" → 提交 "拼"
	host.ApplyIMEEventsForTest([]ime.Event{
		{Kind: ime.EventCompositionUpdate, Composition: "拼"},
		{Kind: ime.EventCharInput, Char: '拼'},
		{Kind: ime.EventCompositionEnd},
	})
	runJobs(wv)
	time.Sleep(150 * time.Millisecond)
	runJobs(wv)

	fmt.Println("[after] " + js(wv, `(function(){
		var out = [];
		var co = document.querySelector('.cm-content');
		var lines = co.querySelectorAll('.cm-line');
		if (lines.length < 3) return 'need>=3 lines, got ' + lines.length;
		var line = lines[2];
		out.push('L2=[' + line.textContent + ']');
		var all = co.textContent;
		out.push('hasPin=' + (all.indexOf('拼') >= 0) + ' pos=' + all.indexOf('拼'));
		var v = window.__editorView;
		if (v) {
			var ds = v.state.doc.toString();
			out.push('STATE-hasPin=' + (ds.indexOf('拼') >= 0) + ' pos=' + ds.indexOf('拼'));
			out.push('head=' + v.state.selection.main.head);
		}
		return out.join(' | ');
	})()`))

	// 对比：普通字符输入（MockKeyChar）位置
	// ★ 诊断：MockKeyChar 前 dump DOM selection（sstate.ranges 状态）——
	// CM6 readDOMChange 重建 DOM 后 selection 是否仍指向有效节点。
	fmt.Println("[sel-before-char] " + js(wv, `(function(){
		var s = window.getSelection();
		var out = [];
		out.push('rangeCount=' + s.rangeCount);
		if (s.rangeCount > 0) {
			try {
				var r = s.getRangeAt(0);
				var sc = r.startContainer;
				out.push('startC=' + (sc ? sc.nodeName + '.' + (sc.nodeType===3 ? 'text:' + sc.nodeValue.slice(0,10) : (sc.className||'')) : 'null') + ' off=' + r.startOffset);
				out.push('isConnected=' + (sc ? sc.isConnected : 'n/a'));
			} catch(e) { out.push('rangeErr=' + e.message); }
		}
		var cu = document.querySelector('.cm-cursor');
		out.push('cursorStyleLeft=' + (cu ? cu.style.left : 'null'));
		return out.join(' | ');
	})()`))
	host.MockKeyChar('X')
	runJobs(wv)
	time.Sleep(150 * time.Millisecond)
	runJobs(wv)
	fmt.Println("[char-after] " + js(wv, `(function(){
		var co = document.querySelector('.cm-content');
		var lines = co.querySelectorAll('.cm-line');
		var line = lines[2];
		var all = co.textContent;
		var v = window.__editorView;
		return 'L2=[' + line.textContent + '] X-pos=' + all.indexOf('X') + (v ? ' head=' + v.state.selection.main.head : '');
	})()`))

	fmt.Println("DONE")
}
