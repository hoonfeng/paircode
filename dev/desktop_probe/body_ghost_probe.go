// Command body_ghost_probe 深挖 body 顶层 201 个幽灵节点的真实内容：
// 1. 每个文本/注释节点的 textContent 长度分布（是否真为空）
// 2. 非空节点的完整内容（判断是否 CSS/JS 片段泄漏）
// 3. #app 之前/之后的节点分界
// 4. head 里是否有残留
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

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	absDist, _ := filepath.Abs(distDir)
	htmlData, err := os.ReadFile(filepath.Join(absDist, "index.html"))
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
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	waitJS := func(ms int) {
		_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
	waitJS(400)

	js := `(function(){
		var out = {};
		var bks = document.body.childNodes;
		out.total = bks.length;
		// 长度分布
		var lens = {};
		var nonEmpty = [];
		for (var i=0; i<bks.length; i++) {
			var n = bks[i];
			var len;
			if (n.nodeType === 3) len = (n.textContent||'').length;
			else if (n.nodeType === 8) len = (n.textContent||'').length;
			else continue;
			var key = n.nodeType + ':' + len;
			lens[key] = (lens[key]||0) + 1;
			if (len > 0 && nonEmpty.length < 15) {
				nonEmpty.push('[' + i + ']' + (n.nodeType===8?'C:':'T:') + '"' + (n.textContent||'').slice(0,80) + '"');
			}
		}
		out.lens = lens;
		out.nonEmpty = nonEmpty;
		// 找到 #app 的索引
		for (var i=0; i<bks.length; i++) {
			if (bks[i].nodeType === 1) { out.appIdx = i; break; }
		}
		// head 子节点
		var hc = document.head.childNodes;
		out.headCount = hc.length;
		out.headNodes = [];
		for (var j=0; j<hc.length && j<15; j++) {
			var n = hc[j];
			var d;
			if (n.nodeType === 1) d = 'E:' + n.tagName;
			else if (n.nodeType === 3) d = 'T:"' + (n.textContent||'').slice(0,15) + '"';
			else if (n.nodeType === 8) d = 'C:' + (n.textContent||'').slice(0,15);
			else d = 'N' + n.nodeType;
			out.headNodes.push(d);
		}
		// #app 的兄弟节点（body 内 #app 前后的 text/comment 具体是什么）
		out.beforeApp = [];
		out.afterApp = [];
		for (var i=0; i<bks.length; i++) {
			var n = bks[i];
			var d;
			if (n.nodeType === 3) d = 'T[' + (n.textContent||'').length + ']';
			else if (n.nodeType === 8) d = 'C[' + (n.textContent||'').length + ']';
			else d = 'E:' + n.tagName;
			if (i < out.appIdx) out.beforeApp.push(d);
			else if (i > out.appIdx) out.afterApp.push(d);
		}
		// 非空节点里是否有 style/script 内容
		var suspicious = [];
		for (var i=0; i<bks.length; i++) {
			var n = bks[i];
			if (n.nodeType !== 3) continue;
			var t = (n.textContent||'');
			if (t.indexOf('{') >= 0 || t.indexOf('<') >= 0 || t.indexOf('function') >= 0) {
				suspicious.push('[' + i + '] len=' + t.length + ' head="' + t.slice(0,100) + '"');
				if (suspicious.length >= 8) break;
			}
		}
		out.suspicious = suspicious;
		return JSON.stringify(out);
	})()`
	v, err := wv.JSInterpreter().RunJS(js)
	if err != nil {
		log.Printf("dump err: %v", err)
	} else {
		fmt.Printf("[ghost] %s\n", v.ToString())
	}
}

func setupLoaders(wv *webkit.WebView, distDir string) {
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
