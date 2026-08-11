// Command edit_click_probe 验证真实桌面链路：点击编辑器 → CM6 同步
// DOM selection → 键盘输入成功。
//   MockFocus（模拟 FocusElement）→ JS mousedown（detail:1）
//   → runJobs（CM6 measure + updateSelection 同步 DOM selection）
//   → 检查 getSelection().getRangeAt(0)（sstate.ranges）
//   → MockKeyChar → 检查 DOM/state 是否出现字符
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
	for i := 0; i < 6; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 400); })`)
		time.Sleep(400 * time.Millisecond)
		runJobs(wv)
	}

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

	// ① MockFocus（模拟真实 desktop 点击聚焦）
	host := app.NewHostForTest(wv, 1280, 800)
	cmEl, err := findContentEditable(wv)
	if err != nil {
		log.Fatalf("no cm-content: %v", err)
	}
	host.MockFocus(cmEl)
	fmt.Println("[focus] contenteditable=" + cmEl.GetAttribute("contenteditable"))
	// hasFocus / activeElement（CM6 updateSelection 依赖）
	fmt.Println("[focus] " + js(wv, `(function(){
		var d = document;
		return 'hasFocus=' + d.hasFocus() + ' activeEl=' + (d.activeElement ? d.activeElement.className : 'null') +
			' isContent=' + (d.activeElement === document.querySelector('.cm-content'));
	})()`))

	// ② JS mousedown 点击行 1 中部（detail:1 = 单击）
	fmt.Println("[click] " + js(wv, `(function(){
		var co = document.querySelector('.cm-content');
		var l1 = co.querySelectorAll('.cm-line')[0];
		var r = l1.getBoundingClientRect();
		var cx = r.left + 40, cy = r.top + r.height/2;
		var ev = new MouseEvent('mousedown', {clientX: cx, clientY: cy, bubbles: true, cancelable: true, button: 0, detail: 1});
		var err = null;
		try { l1.dispatchEvent(ev); } catch(e) { err = e.message; }
		var v = window.__editorView;
		var h = v ? v.state.selection.main.head : -1;
		return 'click@' + cx.toFixed(1) + ',' + cy.toFixed(1) + ' head=' + h + ' err=' + err;
	})()`))
	runJobs(wv)
	time.Sleep(50 * time.Millisecond)
	runJobs(wv)

	// ③ 检查 CM6 同步后的 DOM selection（sstate.ranges）
	fmt.Println("[sel] " + js(wv, `(function(){
		var s = window.getSelection();
		if (!s) return 'no-sel';
		var out = ['rc=' + s.rangeCount];
		out.push('anchor=' + (s.anchorNode ? s.anchorNode.nodeName : String(s.anchorNode)) + '/' + s.anchorOffset);
		if (s.rangeCount == 0) return out.join(' | ');
		var r0 = s.getRangeAt(0);
		if (r0 === null || r0 === undefined) { out.push('r0=' + r0); }
		else {
			var n = r0.startContainer;
			var txt = n && n.nodeType == 3 ? '[' + n.nodeValue.slice(0, 12) + ']' : (n ? '<' + n.nodeName + '>' : String(n));
			out.push('start=#' + (n ? n.nodeName : '?') + '/' + r0.startOffset + ' ' + txt);
		}
		var v = window.__editorView;
		if (v && v.observer && v.observer.selectionRange) {
			var sr = v.observer.selectionRange;
			out.push('obsRange=' + (sr.anchorNode ? sr.anchorNode.nodeName : 'null') + '/' + sr.anchorOffset +
				' focus=' + (sr.focusNode ? sr.focusNode.nodeName : 'null') + '/' + sr.focusOffset);
		}
		// ★ CM6 updateSelection 的 rawSel = getSelection(view.root) 链路
		if (v) {
			var rt2 = v.root;
			out.push('root=' + (rt2 ? rt2.nodeName + '/' + rt2.nodeType : 'null') + ' str=' + String(rt2) + ' hasGetSel=' + (rt2 && typeof rt2.getSelection) +
				' isDoc=' + (rt2 === document) + ' sameFn=' + (rt2 && rt2.getSelection === window.getSelection));
			var selX = null;
			try {
				selX = rt2 && rt2.nodeType == 9 ? rt2.defaultView.getSelection() : rt2.getSelection();
			} catch(e) { out.push('selXErr=' + e.message); }
			out.push('selX=' + (selX ? typeof selX + '/' + (selX.collapse ? 'has-collapse' : 'NO-collapse') : 'null'));
			var ws2 = window.getSelection();
			var dgs = document.getSelection();
			out.push('winSel=' + (ws2 ? (ws2.collapse ? 'has-collapse' : 'NO-collapse') : 'null') +
				' same=' + (ws2 === selX) +
				' rc=' + ws2.rangeCount + ' collapseType=' + typeof ws2.collapse +
				' docSel=' + (dgs ? (dgs.collapse ? 'has-collapse' : 'NO-collapse') : 'null') +
				' docSameWin=' + (dgs === ws2) +
				' src=[' + String(document.getSelection).slice(0, 90) + ']');
		}
		if (v && v.docView) {
			try {
				var d = v.docView.inlineDOMNearPos(5, 1);
				out.push('inline=' + (d && d.node ? d.node.nodeName : 'null') + '/' + (d ? d.offset : '?'));
			} catch(e) { out.push('inlineErr=' + e.message); }
			try {
				v.docView.updateSelection(false, true);
				out.push('us=ok');
			} catch(e) { out.push('usErr=' + e.message); }
		}
		return out.join(' | ');
	})()`))

	// ④ 键盘输入
	host.MockKeyChar('Z')
	host.MockKeyChar('Z')
	runJobs(wv)
	time.Sleep(100 * time.Millisecond)
	runJobs(wv)

	// ⑤ 结果
	fmt.Println("[after] " + js(wv, `(function(){
		var out = [];
		var co = document.querySelector('.cm-content');
		var t = co.textContent;
		out.push('hasZZ=' + (t.indexOf('ZZ') >= 0));
		out.push('line1=[' + t.split('\n')[0] + ']');
		var v = window.__editorView;
		if (v) {
			var h = v.state.selection.main.head;
			var line = v.state.doc.lineAt(h);
			out.push('head=' + h + ' line=' + line.number);
			out.push('stateLen=' + v.state.doc.length);
			out.push('stateHasZZ=' + (v.state.doc.toString().indexOf('ZZ') >= 0));
		}
		return out.join(' | ');
	})()`))
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
