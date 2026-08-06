// Command scroll_fold_probe 验证「滚动加载历史消息后折叠状态保持收缩」：
//   1. panel-only 加载真实 dist + desktopbridge（真实 .pair/conversations 存储）
//   2. 点击多 run 会话（UI 触发 switchConv）→ 检查首次加载折叠状态（applyAutoCollapse）
//   3. 派发 scroll 事件触发 loadMoreMessages 加载更早消息
//   4. 复查折叠状态：older 消息的 thinking 应折叠（_collapsed=true）、tool_call 收缩
//      （_expanded=false）、assistant 折叠成完成摘要（_folded=true）——即 loadMoreMessages
//      prepend 后调用 applyAutoCollapse 的修复效果
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

func setupLoadersF(wv *webkit.WebView, distDir string) {
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
}

func waitJSF(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func runJobsF(wv *webkit.WebView) {
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

func runJSF(wv *webkit.WebView, script string) string {
	v, err := wv.JSInterpreter().RunJS(script)
	if err != nil {
		return "[err] " + err.Error()
	}
	return v.ToString()
}

// 统计当前对话所有 assistant 消息的折叠状态
const foldStatsJS = `(function(){
	var st = window.__state;
	var msgs = st.messagesByConv[st.currentConvId] || [];
	var s = {assistant:0, folded:0, unfolded:0, thinkTotal:0, thinkExpanded:0, thinkCollapsed:0, thinkUndef:0, toolTotal:0, toolExpanded:0, toolCollapsed:0, toolUndef:0, firstIdx:-1, userRuns:0};
	for (var i = 0; i < msgs.length; i++) {
		var m = msgs[i];
		if (i === 0 && m._idx !== undefined) s.firstIdx = m._idx;
		if (m.role === 'user') s.userRuns++;
		if (m.role !== 'assistant') continue;
		s.assistant++;
		if (m._folded) s.folded++; else s.unfolded++;
		var segs = m.segments || [];
		for (var j = 0; j < segs.length; j++) {
			var g = segs[j];
			if (g.type === 'thinking') {
				s.thinkTotal++;
				if (g._collapsed === false) s.thinkExpanded++;
				else if (g._collapsed === true) s.thinkCollapsed++;
				else s.thinkUndef++;
			}
			if (g.type === 'tool_call') {
				s.toolTotal++;
				if (g._expanded === true) s.toolExpanded++;
				else if (g._expanded === false) s.toolCollapsed++;
				else s.toolUndef++;
			}
		}
	}
	return JSON.stringify(s);
})()`

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
	setupLoadersF(wv, distDir)
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
	waitJSF(wv, 500)
	waitJSF(wv, 500)
	runJobsF(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	runJSF(wv, `(function(){ var st = window.__state;
		if (!st) return 'no state';
		st.workspaceRoot = 'F:\\syproject\\gou-ide';
		st.workspaceName = 'gou-ide';
		st.workspaceFolders = ['F:\\syproject\\gou-ide'];
		return 'ok ws=' + st.workspaceRoot; })()`)
	waitJSF(wv, 300)
	runJobsF(wv)

	// ── 点击多 run 会话（>60 条）触发 switchConv ──
	sel := runJSF(wv, `(function(){
		var st = window.__state;
		var convs = st.conversations || [];
		for (var i = 0; i < convs.length; i++) {
			if (convs[i].msgCount > 60) return JSON.stringify({idx: i, id: convs[i].id, mc: convs[i].msgCount});
		}
		return JSON.stringify({idx: -1});
	})()`)
	fmt.Printf("[sfold] 选中候选: %s\n", sel)
	var idx int
	fmt.Sscanf(sel, `{"idx":%d`, &idx)
	if idx < 0 {
		fmt.Printf("[sfold] 未找到 >60 条会话，用第一个会话继续\n")
		idx = 0
	}
	fmt.Printf("[sfold] 点击会话 idx=%d\n", idx)
	runJSF(wv, fmt.Sprintf(`(function(){
		var items = document.querySelectorAll('.conv-item');
		if (!items[%d]) return 'no conv-item %d';
		items[%d].dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
		return 'clicked';
	})()`, idx, idx, idx))
	waitJSF(wv, 900)
	runJobsF(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsF(wv)

	before := runJSF(wv, foldStatsJS)
	fmt.Printf("[sfold] switchConv 后 BEFORE: %s\n", before)

	// DOM 层：展开的 thinking 文本 vs 折叠的"思考…"
	fmt.Printf("[sfold] DOM: %s\n", runJSF(wv, `(function(){
		var q = function(s){ return document.querySelectorAll(s).length; };
		return JSON.stringify({thinkText: q('.tl-thinking-text'), thinkCollapsed: q('.tl-thinking-collapsed'), foldedSummary: q('.folded-summary'), tcDetail: q('.tl-tc-detail')});
	})()`))

	// ── 派发 scroll 事件（等价用户上滚）→ loadMoreMessages 加载更早消息 ──
	for i := 0; i < 6; i++ {
		_ = runJSF(wv, `(function(){
			var el = document.querySelector('.chat-messages');
			if (!el) return 'no el';
			el.scrollTop = 0;
			el.dispatchEvent(new Event('scroll'));
			return 'scroll #' + i;
		})()`)
		waitJSF(wv, 700)
		runJobsF(wv)
		wv.RebuildRenderTree()
		wv.EnsureLayout()
		runJobsF(wv)
		cur := runJSF(wv, foldStatsJS)
		fmt.Printf("[sfold] 第 %d 次 scroll 后: %s\n", i+1, cur)
	}

	after := runJSF(wv, foldStatsJS)
	fmt.Printf("[sfold] scroll 派发后 AFTER: %s\n", after)
	fmt.Printf("[sfold] DOM(after): %s\n", runJSF(wv, `(function(){
		var q = function(s){ return document.querySelectorAll(s).length; };
		return JSON.stringify({thinkText: q('.tl-thinking-text'), thinkCollapsed: q('.tl-thinking-collapsed'), foldedSummary: q('.folded-summary'), tcDetail: q('.tl-tc-detail')});
	})()`))
	fmt.Printf("[sfold] ==== 期望：BEFORE/AFTER 均 thinkExpanded=0、toolExpanded=0、folded=assistant（全部收缩）====\n")
	fmt.Printf("[sfold] done\n")
}
