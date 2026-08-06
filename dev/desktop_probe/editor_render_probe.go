// Command editor_render_probe 离屏渲染「编辑器(CodeMirror) + markdown 预览」并输出 PNG。
// 复用 editor_md_probe 的加载/点击流程，wv.Render() 输出 RGBA → 标准库编码 PNG。
// 输出：editor.png（go.mod 编辑器）、markdown.png（README.md 预览）
// 供实机截图对照浏览器渲染差异（用户反馈：编辑区/markdown 显示异常）。
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

	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func setupLoaders4(wv *webkit.WebView, distDir string) {
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

func waitJS4(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func runJobs4(wv *webkit.WebView) {
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

func runJS4(wv *webkit.WebView, script string) string {
	v, err := wv.JSInterpreter().RunJS(script)
	if err != nil {
		return "[err] " + err.Error()
	}
	return v.ToString()
}

// savePNG 把 RGBA 像素（4 字节/像素，行优先）保存为 PNG。
func savePNG(path string, width, height int, rgba []byte) error {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		row := rgba[y*width*4 : (y+1)*width*4]
		for x := 0; x < width; x++ {
			o := x * 4
			img.SetNRGBA(x, y, color.NRGBA{R: row[o], G: row[o+1], B: row[o+2], A: row[o+3]})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
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
	setupLoaders4(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	wv.LoadHTML(string(htmlData))
	waitJS4(wv, 600)
	waitJS4(wv, 600)
	runJobs4(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	clickFile := func(name string) string {
		return runJS4(wv, `(function(){
			var rows = document.querySelectorAll('.file-tree-item .item-row');
			for (var i = 0; i < rows.length; i++) {
				var nm = rows[i].querySelector('.item-name');
				if (nm && nm.textContent.trim() === '`+name+`') {
					rows[i].dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
					return 'clicked idx=' + i;
				}
			}
			return 'not found';
		})()`)
	}

	// ── 1. 打开 go.mod（编辑器） ──
	fmt.Printf("[rend] open go.mod: %s\n", clickFile("go.mod"))
	waitJS4(wv, 400)
	runJobs4(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	// CM6 行样式（编辑器字体链路）
	fmt.Printf("[rend] editorStyle=%s\n", runJS4(wv, `(function(){
		var ed = document.querySelector('.cm-editor');
		if (!ed) return 'no editor';
		var line = ed.querySelector('.cm-line');
		if (!line) return 'no line';
		var cs = getComputedStyle(line);
		var scs = getComputedStyle(ed.querySelector('.cm-scroller'));
		return JSON.stringify({fontFamily: cs.fontFamily, fontSize: cs.fontSize, fontWeight: cs.fontWeight,
			scrollerFF: scs.fontFamily, scrollerFS: scs.fontSize, lineCount: ed.querySelectorAll('.cm-line').length});
	})()`))
	// 渲染编辑器 → PNG
	rgba, _ := wv.Render()
	_ = savePNG(filepath.Join(wd, "_editor_render.png"), 1280, 800, rgba)
	fmt.Printf("[rend] saved _editor_render.png (%d bytes rgba)\n", len(rgba))

	// ── 2. 打开 README.md（markdown 预览） ──
	fmt.Printf("[rend] open README.md: %s\n", clickFile("README.md"))
	waitJS4(wv, 500)
	runJobs4(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Printf("[rend] mdActive=%s\n", runJS4(wv, `(function(){ var st = window.__state; return st ? st.activeFile : ''; })()`))
	fmt.Printf("[rend] mdSample=%s\n", runJS4(wv, `(function(){
		var wrap = document.querySelector('.md-preview-wrap');
		if (!wrap) return 'no md';
		var h1 = wrap.querySelector('h1');
		var code = wrap.querySelector('code');
		var p = wrap.querySelector('p');
		var cs = function(el){ if (!el) return null; var s = getComputedStyle(el); return {fs: s.fontSize, fw: s.fontWeight, ff: s.fontFamily, ta: s.textAlign}; };
		return JSON.stringify({h1: cs(h1), code: cs(code), p: cs(p), htmlLen: wrap.innerHTML.length});
	})()`))
	rgba2, _ := wv.Render()
	_ = savePNG(filepath.Join(wd, "_markdown_render.png"), 1280, 800, rgba2)
	fmt.Printf("[rend] saved _markdown_render.png\n")

	// ── 3. 打开历史对话（RightPanel markdown 消息） ──
	fmt.Printf("[rend] conv=%s\n", runJS4(wv, `(function(){
		var items = document.querySelectorAll('.conv-item');
		if (items.length === 0) return 'no conv-item';
		items[0].dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
		return 'clicked conv[0]';
	})()`))
	waitJS4(wv, 500)
	runJobs4(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rgba3, _ := wv.Render()
	_ = savePNG(filepath.Join(wd, "_conv_render.png"), 1280, 800, rgba3)
	fmt.Printf("[rend] saved _conv_render.png\n")

	fmt.Printf("[rend] 完成\n")
}
