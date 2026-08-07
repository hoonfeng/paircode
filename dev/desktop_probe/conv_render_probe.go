// Command conv_render_probe 验证桌面端（goja）完整加载后：
//   1. App.vue setup 顶层同步预取（go.bridge_call）是否填充 state.conversations
//   2. ConvSidebar 是否渲染出会话列表 DOM
// 定位「对话列表加载异常 现在空的」：数据链路正常（17 会话），问题可能在前端渲染。
package main

import (
	"fmt"
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
	for i := 0; i < 15; i++ {
		wv.JSInterpreter().RunJobs()
		if mf := wv.MainFrame(); mf != nil {
			if fr := mf.Frame(); fr != nil {
				fr.MarkRenderTreeDirty()
			}
		}
		time.Sleep(20 * time.Millisecond)
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

	wv := webkit.NewWebView()
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	// ScriptLoader/StyleSheetLoader：从 dist 加载脚本与样式（缺了则 Vue 不执行）
	absDist, _ := filepath.Abs(distDir)
	_ = absDist
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
	// 分阶段等待 Vue 模块初始化 + mount（dist 10MB，脚本执行慢）
	for i := 0; i < 4; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 500); })`)
		time.Sleep(600 * time.Millisecond)
		runJobs(wv)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 检查 state + ConvSidebar DOM
	iv, _ := wv.JSInterpreter().RunJS(`(function(){
		var out = {};
		var st = window.__state;
		out.hasState = !!st;
		if (st) {
			out.workspaceRoot = st.workspaceRoot;
			out.convs = Array.isArray(st.conversations) ? st.conversations.length : 'n/a';
			out.currentConvId = st.currentConvId;
			out.msgs = Array.isArray(st.messages) ? st.messages.length : 'n/a';
			if (Array.isArray(st.conversations) && st.conversations.length) {
				out.titles = st.conversations.slice(0,4).map(function(c){ return (c.title||'').slice(0,18); });
			}
		}
		// ConvSidebar DOM：找 conv-item / conv-list
		out.convItems = document.querySelectorAll('.conv-item').length;
		out.convList = !!document.querySelector('.conv-list, .conversation-list, .conv-sidebar');
		out.sidebarText = (function(){
			var els = document.querySelectorAll('.conv-item, .conv-title, .conv-name');
			var t = [];
			for (var i=0;i<els.length && i<6;i++) { t.push((els[i].textContent||'').trim().slice(0,20)); }
			return t;
		})();
		return JSON.stringify(out);
	})()`)
	fmt.Println("[convrender]", iv.ToString())
	// 打印 console 里的错误
	cout := wv.ConsoleOutput()
	lines := strings.Split(cout, "\n")
	errLines := 0
	for _, ln := range lines {
		if strings.Contains(ln, "Error") || strings.Contains(ln, "error") || strings.Contains(ln, "undefined") || strings.Contains(ln, "Cannot") {
			fmt.Println("[jserr]", ln)
			errLines++
			if errLines > 15 {
				break
			}
		}
	}
	fmt.Println("[console] total lines:", len(lines))
}
