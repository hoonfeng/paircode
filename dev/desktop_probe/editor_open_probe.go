// Command editor_open_probe 复现「打开文件后聊天区内容被裁剪/消失」：
//   1. panel-only 加载真实 dist + desktopbridge（真实 .pair/conversations）
//   2. 点击多 run 会话 → 聊天区有消息
//   3. 渲染 PNG（打开文件前）
//   4. 文件树点击打开 go.mod（CodeMirror 6）
//   5. 渲染 PNG（打开文件后）
//   6. 对比聊天区 (429,67 601x711) 像素 ink：打开前后是否骤减
//      （用户反馈：打开文件后大量组件减去了编辑器占用的空间 → 绘制区域变小）
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

	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func setupLoadersE(wv *webkit.WebView, distDir string) {
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

func waitJSE(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func runJobsE(wv *webkit.WebView) {
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

func runJS(wv *webkit.WebView, script string) string {
	v, err := wv.JSInterpreter().RunJS(script)
	if err != nil {
		return "[err] " + err.Error()
	}
	return v.ToString()
}

func pngEncodeE(width, height int, rgba []byte) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	copy(img.Pix, rgba)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func renderPNG(wv *webkit.WebView, name string) {
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsE(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rgba, err := wv.Render()
	if err == nil && len(rgba) == 1280*800*4 {
		_ = os.WriteFile(name, pngEncodeE(1280, 800, rgba), 0o644)
		log.Printf("saved %s", name)
	} else {
		log.Printf("render %s failed: %v len=%d", name, err, len(rgba))
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
	layout.FontMetricsFunc = func(family string, size float64, weight int, style2 string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	wv := webkit.NewWebView()
	setupLoadersE(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	origBefore := webkit.BeforePageScripts
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		if origBefore != nil {
			origBefore(rt)
		}
		rt.RunJS(`window.__DESKTOP_PANEL_MODE__ = true;`)
	}
	wv.LoadHTML(string(htmlData))
	waitJSE(wv, 500)
	waitJSE(wv, 500)
	runJobsE(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	runJS(wv, `(function(){ var st = window.__state;
		if (!st) return 'no state';
		st.workspaceRoot = 'F:\\syproject\\gou-ide';
		st.workspaceName = 'gou-ide';
		st.workspaceFolders = ['F:\\syproject\\gou-ide'];
		return 'ok ws=' + st.workspaceRoot; })()`)
	waitJSE(wv, 300)
	runJobsE(wv)
	waitJSE(wv, 600)
	runJobsE(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 选一个多 run 会话点击
	sel := runJS(wv, `(function(){
		var st = window.__state;
		var convs = st.conversations || [];
		for (var i = 0; i < convs.length; i++) {
			if (convs[i].msgCount > 60) return JSON.stringify({idx: i, id: convs[i].id, mc: convs[i].msgCount});
		}
		return JSON.stringify({idx: -1});
	})()`)
	fmt.Printf("[eopen] 候选: %s\n", sel)
	var idx int
	fmt.Sscanf(sel, `{"idx":%d`, &idx)
	if idx < 0 {
		idx = 0
	}
	runJS(wv, fmt.Sprintf(`(function(){
		var items = document.querySelectorAll('.conv-item');
		if (!items[%d]) return 'no conv-item';
		items[%d].dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
		return 'clicked ' + items[%d].textContent.trim().slice(0, 20);
	})()`, idx, idx, idx))
	waitJSE(wv, 900)
	runJobsE(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsE(wv)

	// 会话消息数（打开文件前）
	fmt.Printf("[eopen] 会话: %s\n", runJS(wv, `(function(){
		var st = window.__state;
		var msgs = st.messagesByConv[st.currentConvId] || [];
		return JSON.stringify({len: msgs.length, convId: st.currentConvId});
	})()`))
	// 聊天区消息 DOM 数
	fmt.Printf("[eopen] 消息DOM: %s\n", runJS(wv, `(function(){
		var els = document.querySelectorAll('.chat-messages .msg, .chat-messages [class*=msg-item], .chat-messages [class*=message]');
		return 'n=' + els.length;
	})()`))

	// 打开文件前渲染
	renderPNG(wv, "_eopen_before.png")
	fmt.Printf("[eopen] BEFORE 渲染完成\n")

	// 打开 go.mod
	openIdx := runJS(wv, `(function(){
		var rows = document.querySelectorAll('.file-tree-item .item-row');
		for (var i = 0; i < rows.length; i++) {
			var nm = rows[i].querySelector('.item-name');
			if (nm && nm.textContent.trim() === 'go.mod') return String(i);
		}
		return '-1';
	})()`)
	fmt.Printf("[eopen] go.mod idx=%s\n", openIdx)
	if openIdx != "-1" {
		runJS(wv, fmt.Sprintf(`(function(){
			var rows = document.querySelectorAll('.file-tree-item .item-row');
			var el = rows[%s];
			if (!el) return 'no row';
			el.dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
			return 'clicked';
		})()`, openIdx))
	}
	waitJSE(wv, 1200)
	runJobsE(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsE(wv)

	// 打开文件后状态
	fmt.Printf("[eopen] 打开后: %s\n", runJS(wv, `(function(){
		var st = window.__state;
		var msgs = st.messagesByConv[st.currentConvId] || [];
		var ed = document.querySelector('.cm-editor');
		var edR = ed ? ed.getBoundingClientRect() : null;
		return JSON.stringify({msgs: msgs.length, cmEditor: edR ? Math.round(edR.left)+','+Math.round(edR.top)+' '+Math.round(edR.width)+'x'+Math.round(edR.height) : 'none'});
	})()`))
	// 聊天区消息 DOM 数（打开后）
	fmt.Printf("[eopen] 打开后消息DOM: %s\n", runJS(wv, `(function(){
		var els = document.querySelectorAll('.chat-messages .msg, .chat-messages [class*=msg-item], .chat-messages [class*=message]');
		return 'n=' + els.length;
	})()`))

	// 打开文件后渲染
	renderPNG(wv, "_eopen_after.png")
	fmt.Printf("[eopen] AFTER 渲染完成\n")
	fmt.Printf("[eopen] done\n")
}
