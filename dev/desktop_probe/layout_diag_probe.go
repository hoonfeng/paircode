// Command layout_diag_probe 完整布局诊断：复现「打开文件布局异常（内容绘制区域变小、
// 编辑区整体偏移）」。
// 设计（针对用户反馈"开启 debug 模式、收集完整日志"）：
//   阶段0：初始布局（未打开文件）— 收集完整几何/样式/滚动日志
//   阶段1：点击打开 go.mod（CM6 挂载）— 收集同一批日志 → 前后对比
//   阶段2：Resize 到 2560x1440（模拟最大化）— 再收集 → 验证"最大化后也出现"
//   每阶段输出：viewport / app-root / grid 各列 / editor 全链 / 滚动 / CM6 状态
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
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

const layoutHookJS = `
window.__errs = [];
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
};
`

// collectLayoutJS 收集完整布局日志（JSON）
const collectLayoutJS = `(function(){
  var o = {vw: {}, app: {}, cols: {}, editor: {}, cm: {}, scroll: {}, errs: window.__errs || []};
  o.vw.w = window.innerWidth; o.vw.h = window.innerHeight;
  var de = document.documentElement;
  o.vw.clientW = de.clientWidth; o.vw.clientH = de.clientHeight;
  o.vw.bodyScrollH = document.body ? document.body.scrollHeight : -1;
  o.vw.bodyScrollTop = document.body ? document.body.scrollTop : -1;
  o.vw.deScrollH = de.scrollHeight; o.vw.deScrollTop = de.scrollTop;
  function rect(sel, key) {
    var el = document.querySelector(sel);
    if (!el) { o[key] = sel + '=NULL'; return; }
    var r = el.getBoundingClientRect();
    var cs = getComputedStyle(el);
    o[key] = sel + '=(' + Math.round(r.left) + ',' + Math.round(r.top) + ' ' + Math.round(r.width) + 'x' + Math.round(r.height) + ') disp=' + cs.display + ' w=' + cs.width + ' minw=' + cs.minWidth + ' pos=' + cs.position + ' flex=' + cs.flex + ' ovf=' + cs.overflow + ' gridCol=' + (cs.gridColumn || '') + ' gridRow=' + (cs.gridRow || '');
  }
  rect('.app-root', 'app');
  rect('.activity-bar', 'cols.activity');
  rect('.sidebar', 'cols.sidebar');
  rect('.main-area', 'cols.main');
  rect('.right-container', 'cols.right');
  rect('.editor-area', 'editor.area');
  rect('.editor-body', 'editor.body');
  rect('.editor-wrapper', 'editor.wrapper');
  rect('.code-editor-wrapper', 'editor.codeWrapper');
  rect('.cm-editor', 'cm.editor');
  rect('.cm-scroller', 'cm.scroller');
  rect('.cm-content', 'cm.content');
  // app-root 的 grid-template 实际值
  var ar = document.querySelector('.app-root');
  if (ar) { var acs = getComputedStyle(ar); o.app.gridCols = acs.gridTemplateColumns; o.app.gridRows = acs.gridTemplateRows; o.app.gridGap = acs.gap; }
  // 滚动状态
  var ea = document.querySelector('.editor-area');
  if (ea) { o.editor.scrollTop = ea.scrollTop; o.editor.scrollH = ea.scrollHeight; o.editor.clientH = ea.clientHeight; }
  var sc = document.querySelector('.cm-scroller');
  if (sc) { o.cm.scScrollTop = sc.scrollTop; o.cm.scScrollH = sc.scrollHeight; o.cm.scClientH = sc.clientHeight; }
  var co = document.querySelector('.cm-content');
  if (co) { o.cm.contentChildren = co.children.length; o.cm.contentTextLen = (co.textContent || '').length; }
  o.cm.editorExists = !!document.querySelector('.cm-editor');
  o.cm.lineCount = document.querySelectorAll('.cm-line').length;
  // 文件树状态
  var ft = document.querySelector('.file-tree');
  o.ftExists = !!ft;
  o.ftItems = document.querySelectorAll('.file-tree-item .item-row').length;
  o.activeFile = (document.querySelector('.file-tree-item.active .item-name') || {}).textContent || '';
  return JSON.stringify(o);
})()`

const openGoModJS = `(function(){
  var rows = document.querySelectorAll('.file-tree-item .item-row');
  for (var i = 0; i < rows.length; i++) {
    var nm = rows[i].querySelector('.item-name');
    if (nm && nm.textContent.trim() === 'go.mod') return String(i);
  }
  return '-1';
})()`

const clickIdxJS = `(function(){
  var rows = document.querySelectorAll('.file-tree-item .item-row');
  var el = rows[IDX];
  if (!el) return;
  el.dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
})()`

func setupLoadersL(wv *webkit.WebView, distDir string) {
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

func waitJSL(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func runJobsL(wv *webkit.WebView) {
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

func runJSL(wv *webkit.WebView, script string) string {
	v, err := wv.JSInterpreter().RunJS(script)
	if err != nil {
		return "[err] " + err.Error()
	}
	return v.ToString()
}

func pngEncodeL(width, height int, rgba []byte) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	copy(img.Pix, rgba)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// collect 收集一阶段完整布局日志
func collect(wv *webkit.WebView, stage string, out *strings.Builder) {
	out.WriteString("\n===== STAGE: " + stage + " =====\n")
	geo := runJSL(wv, collectLayoutJS)
	out.WriteString("[GEOMETRY] " + geo + "\n")
	// 行号区诊断（cm-gutters）
	gut := runJSL(wv, `(function(){
	  var g = document.querySelector('.cm-gutters');
	  if (!g) return 'no-gutters';
	  var r = g.getBoundingClientRect();
	  return 'gutters=(' + Math.round(r.left) + ',' + Math.round(r.top) + ' ' + Math.round(r.width) + 'x' + Math.round(r.height) + ')';
	})()`)
	out.WriteString("[GUTTERS] " + gut + "\n")
	out.WriteString("[ERRS] " + runJSL(wv, `window.__errs.join(' ;; ')`) + "\n")
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

	out := &strings.Builder{}

	wv := webkit.NewWebView()
	wv.Resize(1280, 800)
	setupLoadersL(wv, distDir)
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	waitJSL(wv, 1500)
	runJobsL(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJSL(wv, layoutHookJS)

	// 阶段0：初始布局（未打开文件）
	collect(wv, "0-初始(未打开文件) 1280x800", out)

	// 阶段1：点击打开 go.mod
	idx := runJSL(wv, openGoModJS)
	out.WriteString("\n[OPEN] go.mod row idx=" + idx + "\n")
	runJSL(wv, strings.ReplaceAll(clickIdxJS, "IDX", idx))
	waitJSL(wv, 1500)
	runJobsL(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsL(wv)
	collect(wv, "1-打开go.mod后 1280x800", out)

	// 阶段2：最大化（2560x1440）
	wv.Resize(2560, 1440)
	waitJSL(wv, 800)
	runJobsL(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsL(wv)
	collect(wv, "2-最大化 2560x1440", out)

	// 阶段3：渲染 PNG（1280x800 打开文件后的实际画面）
	wv.Resize(1280, 800)
	waitJSL(wv, 600)
	runJobsL(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsL(wv)
	if rgba, err := wv.Render(); err == nil && len(rgba) == 1280*800*4 {
		os.WriteFile("_layout_diag_1280.png", pngEncodeL(1280, 800, rgba), 0o644)
		out.WriteString("\n[PNG] saved _layout_diag_1280.png\n")
	}
	// 阶段4：小窗口 1024x768（最小宽度下布局）
	wv.Resize(1024, 768)
	waitJSL(wv, 600)
	runJobsL(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsL(wv)
	collect(wv, "3-小窗口 1024x768", out)

	// 写完整日志文件
	logName := "_layout_diag_log.txt"
	os.WriteFile(logName, []byte(out.String()), 0o644)
	fmt.Print(out.String())
	fmt.Println("\n===== 完整日志已写入 " + logName + " =====")
}
