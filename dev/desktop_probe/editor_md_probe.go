// Command editor_md_probe 复现「编辑区打开文件显示异常」+「markdown 渲染异常」。
// 加载真实 dist + desktopbridge（真实 API）：
//   1. 文件树点击打开 go.mod（文本文件 → CodeEditor/CodeMirror 6）
//   2. 诊断 CodeMirror DOM：.cm-editor/.cm-scroller/.cm-content/.cm-line
//      存在性、行数、内容、className、全局错误捕获（CM6 初始化异常定位）
//   3. 打开 .md 文件（默认预览模式 → MarkdownRenderer marked v-html）
//   4. 诊断 markdown HTML：h/p/code/strong 的 computedStyle（font-size/weight/align）
//      + CSS 变量定义与 var() 引用解析
// 对照：浏览器端 web_debug 正常 → 差异在 wb-ui 引擎。
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func setupLoaders3(wv *webkit.WebView, distDir string) {
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
				if err != nil {
					return "", err
				}
				return string(data), nil
			}
		}
	}
}

func waitJS3(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func runJobs3(wv *webkit.WebView) {
	for i := 0; i < 8; i++ {
		wv.JSInterpreter().RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func runJS3(wv *webkit.WebView, script string) string {
	v, err := wv.JSInterpreter().RunJS(script)
	if err != nil {
		return "[err] " + err.Error()
	}
	return v.ToString()
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
	layout.FontMetricsFunc = func(family string, size float64, weight int, style2 string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	wv := webkit.NewWebView()
	setupLoaders3(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	waitJS3(wv, 600)
	waitJS3(wv, 600)
	runJobs3(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	fmt.Printf("[edmd] root=%s\n", runJS3(wv, `(function(){ var st = window.__state; return st ? st.workspaceRoot : 'no-state'; })()`))

	// ── 注入全局错误捕获（CM6 初始化异常定位） ──
	runJS3(wv, `(function(){
		window.__errs = [];
		window.addEventListener('error', function(e){ window.__errs.push('error: ' + (e && (e.message || e.type))); }, true);
		window.addEventListener('unhandledrejection', function(e){ window.__errs.push('rejection: ' + (e && e.reason ? (e.reason.message || e.reason) : e)); }, true);
		var _ce = console.error;
		console.error = function(){
			window.__errs.push('console.error: ' + Array.prototype.slice.call(arguments).map(function(a){
				var m = typeof a === 'string' ? a : ((a && a.message) || String(a));
				if (a && a.stack) m += ' | STACK: ' + String(a.stack).split('\n').slice(0, 8).join(' <- ');
				return m;
			}).join(' | ').slice(0, 700));
			return _ce.apply(console, arguments);
		};
		return 'hooked';
	})()`)

	// ── 文件树顶层节点 ──
	fmt.Printf("[edmd] treeTop=%s\n", runJS3(wv, `(function(){
		var rows = document.querySelectorAll('.file-tree-item .item-row');
		var out = [];
		for (var i = 0; i < rows.length && i < 15; i++) {
			var nm = rows[i].querySelector('.item-name');
			out.push(i + ':' + (nm ? nm.textContent.trim() : '?'));
		}
		return JSON.stringify(out);
	})()`))
	// ── 打开 go.mod ──
	fmt.Printf("[edmd] openGoMod=%s\n", runJS3(wv, `(function(){
		var rows = document.querySelectorAll('.file-tree-item .item-row');
		for (var i = 0; i < rows.length; i++) {
			var nm = rows[i].querySelector('.item-name');
			if (nm && nm.textContent.trim() === 'go.mod') {
				rows[i].dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
				return 'clicked idx=' + i;
			}
		}
		return 'go.mod not in top level';
	})()`))
	waitJS3(wv, 300)
	runJobs3(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	co := strings.TrimSpace(wv.ConsoleOutput())
	if len(co) > 300 {
		co = co[:300]
	}
	fmt.Printf("[edmd] consoleAfterOpen=%s\n", co)

	// ── CodeMirror DOM 结构诊断 ──
	fmt.Printf("[edmd] cmTree=%s\n", runJS3(wv, `(function(){
		var out = {errs: window.__errs || []};
		var ed = document.querySelector('.cm-editor');
		out.hasEditor = !!ed;
		var sc = document.querySelector('.cm-scroller');
		out.hasScroller = !!sc;
		if (sc) {
			var p = sc.parentNode;
			out.parentTag = p ? p.tagName : null;
			out.parentClass = p ? (p.getAttribute('class') || '(none)') : null;
		}
		var content = document.querySelector('.cm-content');
		out.hasContent = !!content;
		out.contentEditable = content ? (content.getAttribute('contenteditable') || '(none)') : null;
		out.contentLen = content ? content.textContent.length : -1;
		out.lineCount = content ? content.querySelectorAll('.cm-line').length : -1;
		var all = document.querySelectorAll('[class*="cm-"]');
		var cls = [];
		for (var i = 0; i < all.length && i < 12; i++) cls.push(all[i].tagName + '.' + all[i].getAttribute('class'));
		out.cmClasses = cls;
		var w = document.querySelector('.code-editor-wrapper');
		out.wrapperHTML0 = w ? w.innerHTML.slice(0, 260) : '';
		out.activeFile = (window.__state && window.__state.activeFile) || '';
		return JSON.stringify(out);
	})()`))

	// ── CSS 变量定义 ──
	fmt.Printf("[edmd] cmVars=%s\n", runJS3(wv, `(function(){
		var cs = getComputedStyle(document.documentElement);
		return JSON.stringify({
			fontCodeDef: cs.getPropertyValue('--font-code').trim(),
			textSecondaryDef: cs.getPropertyValue('--text-secondary').trim(),
			bgPrimaryDef: cs.getPropertyValue('--bg-primary').trim(),
		});
	})()`))

	// ── markdown 渲染诊断：打开一个 .md 文件（默认预览模式） ──
	fmt.Printf("[edmd] openMd=%s\n", runJS3(wv, `(function(){
		var rows = document.querySelectorAll('.file-tree-item .item-row');
		for (var i = 0; i < rows.length; i++) {
			var nm = rows[i].querySelector('.item-name');
			if (nm && nm.textContent.trim() === 'README.md') {
				rows[i].dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
				return 'clicked README.md idx=' + i;
			}
		}
		return 'README.md not found';
	})()`))
	waitJS3(wv, 400)
	runJobs3(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	fmt.Printf("[edmd] mdActive=%s\n", runJS3(wv, `(function(){
		var st = window.__state;
		return JSON.stringify({activeFile: st ? st.activeFile : '', openCount: st ? (st.openFiles || []).length : -1});
	})()`))
	fmt.Printf("[edmd] mdTree=%s\n", runJS3(wv, `(function(){
		var wrap = document.querySelector('.md-preview-wrap') || document.querySelector('.md-html');
		if (!wrap) return JSON.stringify({found: false, activeFile: (window.__state && window.__state.activeFile) || '', errs: window.__errs});
		var out = {found: true, htmlLen: wrap.innerHTML.length, htmlPreview: wrap.innerHTML.slice(0, 150)};
		var tags = ['h1', 'h2', 'h3', 'p', 'code', 'pre', 'strong', 'li', 'blockquote', 'table', 'a'];
		out.styles = {};
		for (var i = 0; i < tags.length; i++) {
			var el = wrap.querySelector(tags[i]);
			if (el) {
				var cs = getComputedStyle(el);
				out.styles[tags[i]] = {
					fs: cs.fontSize, fw: cs.fontWeight, ff: cs.fontFamily, lh: cs.lineHeight,
					ta: cs.textAlign, color: cs.color,
				};
			}
		}
		out.errs = window.__errs || [];
		return JSON.stringify(out);
	})()`))

	fmt.Printf("[edmd] 完成\n")
}
