// Command space_probe 加载真实 dist，向终端写入带空格文本，检查
// xterm DOM span 结构（空格如何表示）与每个可见字符的像素位置，
// 用于定位「空格间距」问题（空格宽度 vs 字符宽度、连续空格折叠）。
package main

import (
	"fmt"
	"image"
	"image/color"
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

func runJobs(wv *webkit.WebView) {
	el := wv.JSInterpreter().GetEventLoop()
	for i := 0; i < 15; i++ {
		if el != nil {
			el.ProcessTasks(20)
		}
		wv.JSInterpreter().RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func js(wv *webkit.WebView, code string) string {
	iv, err := wv.JSInterpreter().RunJS(code)
	if err != nil {
		return "[ERR] " + err.Error()
	}
	return iv.ToString()
}

func shot(wv *webkit.WebView, name string) {
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	pngBytes, err := wv.Render()
	if err != nil {
		log.Printf("Render %s: %v", name, err)
		return
	}
	w, h := wv.Width(), wv.Height()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := (y*w + x) * 4
			if off+3 < len(pngBytes) {
				img.SetRGBA(x, y, color.RGBA{R: pngBytes[off], G: pngBytes[off+1], B: pngBytes[off+2], A: pngBytes[off+3]})
			}
		}
	}
	wd, _ := os.Getwd()
	out := filepath.Join(wd, "dev", "desktop_probe", name)
	f, err := os.Create(out)
	if err != nil {
		log.Printf("create %s: %v", out, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Printf("encode %s: %v", out, err)
		return
	}
	log.Printf("shot %dx%d → %s", w, h, out)
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
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
		time.Sleep(600 * time.Millisecond)
		runJobs(wv)
	}

	// 写带空格文本：等宽字符 + 1/2/3/4 空格
	fmt.Println("[space] " + js(wv, `(function(){
		var term = window.__lastTerm;
		if (!term) return 'no-term';
		term.write('ABC DEF  GHI   JKL    MNO\r\n');
		term.write('12345678901234567890\r\n');
		term.write('A B C D E F G H I J\r\n');
		term.write('中 文 空 格  测 试\r\n');
		return 'written';
	})()`))
	time.Sleep(400 * time.Millisecond)
	runJobs(wv)

	// 检查行 span 结构：每个 span 的 text + 几何
	fmt.Println("[spans] " + js(wv, `(function(){
		var rowsEl = document.querySelector('.xterm-rows');
		if (!rowsEl) return 'no-rows';
		var rows = rowsEl.children;
		var out = [];
		for (var i = 0; i < rows.length && i < 4; i++) {
			var r = rows[i].getBoundingClientRect();
			var spans = [];
			var kids = rows[i].children;
			for (var j = 0; j < (kids ? kids.length : 0) && j < 30; j++) {
				var k = kids[j];
				var kr = k.getBoundingClientRect();
				spans.push({t: k.textContent, x: Math.round(kr.x), w: Math.round(kr.width*100)/100, cw: Math.round(kr.width/7.146*100)/100});
			}
			out.push({row:i, y:Math.round(r.y), h:Math.round(r.height), text: rows[i].textContent, spans: spans});
		}
		// 只返回行 3（CJK）
		var r3 = out[3];
		// 补充每个 span 的 CSS 样式（position/left/width/transform）
		var kids3 = rows[3].children;
		var st = [];
		for (var j = 0; j < kids3.length; j++) {
			var cs = window.getComputedStyle(kids3[j]);
			st.push({pos: cs.position, left: cs.left, width: cs.width, disp: cs.display});
		}
		r3.styles = st;
		return JSON.stringify(r3);
	})()`))

	// 测量空格宽度（引擎层）：measureText(' ') vs measureText('A')
	fmt.Println("[measure] " + js(wv, `(function(){
		var canvas = document.createElement('canvas');
		var ctx = canvas.getContext('2d');
		ctx.font = '13px Consolas';
		var sp = ctx.measureText(' ').width;
		var a = ctx.measureText('A').width;
		var s2 = ctx.measureText('  ').width;
		var cn = ctx.measureText('中').width;
		var cnsp = ctx.measureText('中 ').width;
		var mW = ctx.measureText('W');
		var mA = ctx.measureText('A');
		// 直接创建 span 测试引擎布局宽度
		var doc = document;
		var s1 = doc.createElement('span');
		s1.style.cssText = 'position:absolute;visibility:hidden;white-space:pre;display:inline-block;';
		s1.style.fontSize = '13px'; s1.style.fontFamily = 'Consolas'; s1.style.lineHeight = 'normal';
		s1.textContent = ' ';
		var h1 = doc.createElement('div');
		h1.style.cssText = 'position:absolute;left:0;top:0;visibility:hidden;';
		h1.appendChild(s1);
		doc.body.appendChild(h1);
		var w1 = s1.getBoundingClientRect().width;
		var oh1 = s1.offsetWidth;
		doc.body.removeChild(h1);
		// static span 直接挂 body
		var s2a = doc.createElement('span');
		s2a.style.cssText = 'visibility:hidden;white-space:pre;display:inline-block;';
		s2a.style.fontSize = '13px'; s2a.style.fontFamily = 'Consolas'; s2a.style.lineHeight = 'normal';
		s2a.textContent = ' ';
		doc.body.appendChild(s2a);
		var w2 = s2a.getBoundingClientRect().width;
		var oh2 = s2a.offsetWidth;
		doc.body.removeChild(s2a);
		return JSON.stringify({space: sp, A: a, space2: s2, cjk: cn, cjk_space: cnsp,
			spanAbs: w1, spanAbsOffset: oh1, spanStatic: w2, spanStaticOffset: oh2,
			mW_h: mW.height, mW_w: mW.width, mA_h: mA.height,
			cellH: (function(){ var t = window.__lastTerm; var d = t && t._core && t._core._renderService && t._core._renderService.dimensions; return d ? d.css.cell.height : null; })()});
	})()`))

	// focus 前后 cell 宽度稳定性
	fmt.Println("[focus] " + js(wv, `(function(){
		var term = window.__lastTerm;
		var dims = term._core && term._core._renderService && term._core._renderService.dimensions;
		var before = dims ? dims.css.cell.width : null;
		// 模拟 focus
		if (term.textarea) {
			term.textarea.focus();
			term.focus();
		} else if (term.focus) {
			term.focus();
		}
		return 'before=' + before + ' focused';
	})()`))
	time.Sleep(300 * time.Millisecond)
	runJobs(wv)
	fmt.Println("[focus2] " + js(wv, `(function(){
		var term = window.__lastTerm;
		var dims = term._core && term._core._renderService && term._core._renderService.dimensions;
		var after = dims ? dims.css.cell.width : null;
		var rowsEl = document.querySelector('.xterm-rows');
		var r0 = rowsEl && rowsEl.children[0] ? rowsEl.children[0].getBoundingClientRect() : null;
		var sp = rowsEl && rowsEl.children[0] && rowsEl.children[0].children[1] ? rowsEl.children[0].children[1].getBoundingClientRect() : null;
		return JSON.stringify({cellW_after: after, row0_w: r0 ? Math.round(r0.width*100)/100 : null, spaceSpan_w: sp ? Math.round(sp.width*100)/100 : null});
	})()`))

	shot(wv, "term_space_shot.png")
	log.Println("done")
}
