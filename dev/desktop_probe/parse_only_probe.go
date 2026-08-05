// Command parse_only_probe 二分验证幽灵节点的来源：
// A. 完整 dist index.html（含 Vue script）→ 计数 body 幽灵节点
// B. 去掉 module script 的 index.html（只解析 HTML+CSS）→ 计数
// C. 去掉 style 的 index.html → 计数
// 判断 198 个空节点来自「HTML 解析」还是「JS 执行/渲染管线」。
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
	htmlSrc := string(htmlData)

	// 变体 B：去掉 module script 行
	varB := htmlSrc
	lines := strings.Split(varB, "\n")
	var kept []string
	for _, ln := range lines {
		if strings.Contains(ln, "script type=\"module\"") {
			continue
		}
		kept = append(kept, ln)
	}
	varB = strings.Join(kept, "\n")

	// 变体 C：去掉 <style> ... </style> 块
	varC := htmlSrc
	if i := strings.Index(varC, "<style>"); i >= 0 {
		if j := strings.Index(varC[i:], "</style>"); j >= 0 {
			varC = varC[:i] + "<style></style>" + varC[i+j+len("</style>"):]
		}
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

	run := func(tag, html string) {
		wv := webkit.NewWebView()
		setupLoaders(wv, distDir)
		_ = wv.JSInterpreter()
		desktopbridge.Init(wv)
		wv.LoadHTML(html)
		waitJS := func(ms int) {
			_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
		waitJS(300)
		js := `(function(){
			var out = {total:0, txt0:0, cmt0:0, els:0};
			var bks = document.body.childNodes;
			out.total = bks.length;
			for (var i=0; i<bks.length; i++) {
				var n = bks[i];
				if (n.nodeType === 3) { if ((n.textContent||'').length === 0) out.txt0++; }
				else if (n.nodeType === 8) { if ((n.textContent||'').length === 0) out.cmt0++; }
				else if (n.nodeType === 1) out.els++;
			}
			return JSON.stringify(out);
		})()`
		v, err := wv.JSInterpreter().RunJS(js)
		if err != nil {
			log.Printf("[%s] err: %v", tag, err)
		} else {
			fmt.Printf("[%s] %s\n", tag, v.ToString())
		}
	}

	run("A-full", htmlSrc)
	run("B-no-script", varB)
	run("C-no-style", varC)
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
