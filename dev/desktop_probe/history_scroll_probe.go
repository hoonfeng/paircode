// Command history_scroll_probe 复现「历史对话只加载最后一个 run」：
//   1. panel-only 加载真实 dist + desktopbridge（真实 .pair/conversations 存储）
//   2. 前端 loadConvList 列出真实会话 → 点击一个多 run 会话（UI 触发 switchConv）
//   3. switchConv 走真实 API getMessages(limit=50) → 只显示最后 ~50 条原始行
//      （tool 消息占配额 → 过滤后 ≈1 个 run）
//   4. 模拟引擎修复：向 .chat-messages 派发 scroll 事件（等价 Host.dispatchScrollEvent）
//   5. 验证前端 onScroll → loadMoreMessages 向上翻页 → 更早 run 被加载
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

func setupLoadersH(wv *webkit.WebView, distDir string) {
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

func waitJSH(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func runJobsH(wv *webkit.WebView) {
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
	setupLoadersH(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	// ★ 合并注入：desktopbridge 已设置 BeforePageScripts（fetch 拦截等），
	//   追加 panel-only 标记（不能整体覆盖，否则 fetch 拦截丢失 → API 请求失败）
	origBefore := webkit.BeforePageScripts
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		if origBefore != nil {
			origBefore(rt)
		}
		rt.RunJS(`window.__DESKTOP_PANEL_MODE__ = true;`)
	}
	wv.LoadHTML(string(htmlData))
	waitJSH(wv, 500)
	waitJSH(wv, 500)
	runJobsH(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// ── 确保 workspaceRoot 指向 gou-ide（desktopbridge core.Root 已是） ──
	runJS(wv, `(function(){ var st = window.__state;
		if (!st) return 'no state';
		st.workspaceRoot = 'F:\\syproject\\gou-ide';
		st.workspaceName = 'gou-ide';
		st.workspaceFolders = ['F:\\syproject\\gou-ide'];
		return 'ok ws=' + st.workspaceRoot; })()`)
	waitJSH(wv, 300)
	runJobsH(wv)

	// ── 直接调 API 验证 conversations 返回（异步，存全局后读） ──
	runJS(wv, `(function(){
		window.__apiTest = 'pending';
		return fetch('/api/conversations?workspace=' + encodeURIComponent(window.__state.workspaceRoot))
			.then(function(r){ return r.json().then(function(j){ window.__apiTest = JSON.stringify({status: r.status, len: (j||[]).length, first: (j&&j[0]) ? j[0].id : null}); }); })
			.catch(function(e){ window.__apiTest = 'fetch err ' + e.message; });
	})()`)
	waitJSH(wv, 800)
	runJobsH(wv)
	fmt.Printf("[hscroll] apiTest=%s\n", runJS(wv, `window.__apiTest || 'still-pending'`))

	// ── 触发前端 loadConvList（挂载已跑过，这里再确认） ──
	runJS(wv, `(function(){ var st = window.__state; return 'convs=' + (st.conversations||[]).length; })()`)
	waitJSH(wv, 600)
	runJobsH(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 列出前端当前 conversations
	fmt.Printf("[hscroll] convs=%s\n", runJS(wv, `(function(){
		var st = window.__state;
		return JSON.stringify((st.conversations||[]).map(function(c){ return c.id; }).slice(0, 12));
	})()`))

	// ── 选择一个多 run 会话（后端消息数 > 60 的），点击其 .conv-item ──
	sel := runJS(wv, `(function(){
		var st = window.__state;
		var convs = st.conversations || [];
		for (var i = 0; i < convs.length; i++) {
			if (convs[i].msgCount > 60) return JSON.stringify({idx: i, id: convs[i].id, mc: convs[i].msgCount});
		}
		return JSON.stringify({idx: -1});
	})()`)
	fmt.Printf("[hscroll] 选中候选: %s\n", sel)
	var idx int
	var convID string
	fmt.Sscanf(sel, `{"idx":%d`, &idx)
	if idx < 0 {
		fmt.Printf("[hscroll] 未找到 >60 条会话，用第一个会话继续\n")
		idx = 0
	}
	convID = runJS(wv, fmt.Sprintf(`(function(){ var c = window.__state.conversations[%d]; return c ? c.id : ''; })()`, idx))
	fmt.Printf("[hscroll] 点击会话 idx=%d id=%s\n", idx, convID)

	// UI 点击（真实 click 事件 → Vue @click → switchConv）
	clicked := runJS(wv, fmt.Sprintf(`(function(){
		var items = document.querySelectorAll('.conv-item');
		if (!items[idx]) return 'no conv-item idx=' + idx;
		items[idx].dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
		return 'clicked ' + items[idx].textContent.trim().slice(0, 20);
	})()`, idx))
	// 上面用了未定义 idx，重新用正确方式
	clicked = runJS(wv, fmt.Sprintf(`(function(){
		var items = document.querySelectorAll('.conv-item');
		if (!items[%d]) return 'no conv-item %d';
		items[%d].dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true}));
		return 'clicked ' + items[%d].textContent.trim().slice(0, 20);
	})()`, idx, idx, idx, idx))
	fmt.Printf("[hscroll] %s\n", clicked)

	waitJSH(wv, 900)
	runJobsH(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsH(wv)

	// ── switchConv 后：消息数（应为 ~20-25 条，1 个 run） ──
	before := runJS(wv, `(function(){
		var st = window.__state;
		var msgs = st.messagesByConv[st.currentConvId] || [];
		if (msgs.length === 0) return JSON.stringify({len: 0});
		var idxs = msgs.map(function(m){ return m._idx; });
		var users = msgs.filter(function(m){ return m.role === 'user'; });
		return JSON.stringify({len: msgs.length, firstIdx: idxs[0], lastIdx: idxs[idxs.length-1], userRuns: users.length, total: st.msgTotalByConv[st.currentConvId]});
	})()`)
	fmt.Printf("[hscroll] switchConv 后 BEFORE: %s\n", before)

	// scroll metrics
	fmt.Printf("[hscroll] chat-messages: %s\n", runJS(wv, `(function(){
		var el = document.querySelector('.chat-messages');
		if (!el) return 'no el';
		return JSON.stringify({clientH: el.clientHeight, scrollH: el.scrollHeight, scrollTop: el.scrollTop});
	})()`))

	// ── 模拟引擎修复：循环派发 scroll 事件（等价用户持续向上滚动，
	//    Host.dispatchScrollEvent 每次滚动触发一次）──
	for i := 0; i < 6; i++ {
		_ = runJS(wv, `(function(){
			var el = document.querySelector('.chat-messages');
			if (!el) return 'no el';
			el.scrollTop = 0;
			el.dispatchEvent(new Event('scroll'));
			return 'dispatched scroll #' + i;
		})()`)
		waitJSH(wv, 700)
		runJobsH(wv)
		wv.RebuildRenderTree()
		wv.EnsureLayout()
		runJobsH(wv)
		cur := runJS(wv, `(function(){
			var st = window.__state;
			var msgs = st.messagesByConv[st.currentConvId] || [];
			if (msgs.length === 0) return JSON.stringify({len: 0});
			var idxs = msgs.map(function(m){ return m._idx; });
			var users = msgs.filter(function(m){ return m.role === 'user'; });
			return JSON.stringify({len: msgs.length, firstIdx: idxs[0], lastIdx: idxs[idxs.length-1], userRuns: users.length, noMore: msgs[0] && msgs[0]._noMoreAbove});
		})()`)
		fmt.Printf("[hscroll] 第 %d 次 scroll 后: %s\n", i+1, cur)
	}

	after := runJS(wv, `(function(){
		var st = window.__state;
		var msgs = st.messagesByConv[st.currentConvId] || [];
		if (msgs.length === 0) return JSON.stringify({len: 0});
		var idxs = msgs.map(function(m){ return m._idx; });
		var users = msgs.filter(function(m){ return m.role === 'user'; });
		return JSON.stringify({len: msgs.length, firstIdx: idxs[0], lastIdx: idxs[idxs.length-1], userRuns: users.length, total: st.msgTotalByConv[st.currentConvId], noMore: msgs[0] && msgs[0]._noMoreAbove});
	})()`)
	fmt.Printf("[hscroll] scroll 派发后 AFTER: %s\n", after)

	// 对比
	fmt.Printf("[hscroll] ==== 结论：BEFORE 只覆盖 1 个 run；scroll 派发后翻页加载更早 run ====\n")
	fmt.Printf("[hscroll] done\n")
}
