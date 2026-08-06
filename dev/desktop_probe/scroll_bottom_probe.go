// Command scroll_bottom_probe 验证「聊天区新消息跳底」链路：
//   1. panel-only 加载真实 dist + desktopbridge，打开一个多消息会话
//   2. 检查 chat-messages 的 scrollTop/scrollHeight/clientHeight（几何桥）
//   3. 模拟前端 scrollToBottom（el.scrollTop = el.scrollHeight）
//   4. 检查 scrollTop 是否被钳制到 MaxScroll（totalH - viewH）→ 跳底是否生效
//   5. 引擎侧 BoxScrollOffset 对比（scrollTop setter → SetBoxScrollOffset）
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

func setupLoadersSB(wv *webkit.WebView, distDir string) {
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

func waitJSSB(wv *webkit.WebView, ms int) {
	_, _ = wv.JSInterpreter().RunJS(fmt.Sprintf(`new Promise(function(res){ setTimeout(res, %d); })`, ms))
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func runJobsSB(wv *webkit.WebView) {
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

func runJSSB(wv *webkit.WebView, script string) string {
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
	setupLoadersSB(wv, distDir)
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
	waitJSSB(wv, 500)
	waitJSSB(wv, 500)
	runJobsSB(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	runJSSB(wv, `(function(){ var st = window.__state;
		if (!st) return 'no state';
		st.workspaceRoot = 'F:\\syproject\\gou-ide';
		st.workspaceName = 'gou-ide';
		return 'ok'; })()`)
	waitJSSB(wv, 300)
	runJobsSB(wv)

	// 触发 loadConvList + 选择一个消息最多的会话
	runJSSB(wv, `(function(){ var st = window.__state;
		st.conversations = st.conversations || [];
		return 'convs=' + st.conversations.length + ' root=' + (st.workspaceRoot||''); })()`)
	waitJSSB(wv, 600)
	runJobsSB(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// 触发 /conversations API（loadConvList 异步拉取真实会话；不指定
	// workspace → 后端用默认 core.Root = wb-ui，真实 conversations 所在）
	runJSSB(wv, `(function(){
		window.__apiSB = 'pending';
		var ws = window.__state.workspaceRoot || '';
		var url = '/api/conversations' + (ws ? ('?workspace=' + encodeURIComponent(ws)) : '');
		return fetch(url)
			.then(function(r){ return r.json().then(function(j){ window.__apiSB = 'len=' + (j||[]).length + ' first=' + ((j&&j[0])?j[0].id:'-'); }); })
			.catch(function(e){ window.__apiSB = 'err ' + e.message; });
	})()`)
	waitJSSB(wv, 800)
	runJobsSB(wv)
	fmt.Printf("[sbot] api=%s\n", runJSSB(wv, `window.__apiSB`))
	waitJSSB(wv, 500)
	runJobsSB(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	sel := runJSSB(wv, `(function(){
		var st = window.__state;
		var convs = st.conversations || [];
		for (var i = 0; i < convs.length; i++) {
			if (convs[i].msgCount > 10) return JSON.stringify({idx: i, id: convs[i].id, mc: convs[i].msgCount});
		}
		return JSON.stringify({idx: -1});
	})()`)
	fmt.Printf("[sbot] 真实会话: %s\n", sel)

	// ── 注入大量假消息（验证跳底链路，不依赖真实会话）──
	runJSSB(wv, `(function(){
		var st = window.__state;
		if (!st.currentConvId) st.currentConvId = 'probe_conv';
		if (!st.messagesByConv) st.messagesByConv = {};
		var msgs = [];
		for (var i = 0; i < 40; i++) {
			msgs.push({id: 'm' + i, role: (i % 2 === 0 ? 'user' : 'assistant'), content: '这是第 ' + i + ' 条测试消息，内容较长用于撑高滚动容器。\n第二行内容。\n第三行内容。'});
		}
		st.messagesByConv[st.currentConvId] = msgs;
		st.messages = msgs;
		st.msgTotalByConv = st.msgTotalByConv || {};
		st.msgTotalByConv[st.currentConvId] = msgs.length;
		return 'injected ' + msgs.length + ' msgs';
	})()`)
	waitJSSB(wv, 600)
	runJobsSB(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	runJobsSB(wv)

	// 注入后：chat-messages 几何
	info := runJSSB(wv, `(function(){
		var st = window.__state;
		var msgs = st.messages || [];
		var el = document.querySelector('.chat-messages');
		if (!el) return JSON.stringify({msgLen: msgs.length, el: 'no-el'});
		return JSON.stringify({msgLen: msgs.length, clientH: el.clientHeight, scrollH: el.scrollHeight, scrollTop: el.scrollTop,
			needScroll: el.scrollHeight > el.clientHeight});
	})()`)
	fmt.Printf("[sbot] 注入消息后: %s\n", info)

	// 模拟 scrollToBottom：el.scrollTop = el.scrollHeight
	runJSSB(wv, `(function(){
		var el = document.querySelector('.chat-messages');
		if (!el) return 'no el';
		el.scrollTop = el.scrollHeight;
		return 'set scrollTop=' + el.scrollTop + ' scrollH=' + el.scrollHeight;
	})()`)
	waitJSSB(wv, 500)
	runJobsSB(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	// scrollToBottom 后：scrollTop 是否到 MaxScroll
	after := runJSSB(wv, `(function(){
		var el = document.querySelector('.chat-messages');
		if (!el) return 'no el';
		return JSON.stringify({scrollTop: el.scrollTop, scrollH: el.scrollHeight, clientH: el.clientHeight,
			maxScroll: el.scrollHeight - el.clientHeight, atBottom: el.scrollTop >= (el.scrollHeight - el.clientHeight - 2)});
	})()`)
	fmt.Printf("[sbot] scrollToBottom 后: %s\n", after)

	// 引擎侧 BoxScrollOffset（chat-messages 的滚动偏移）
	eng := runJSSB(wv, `(function(){
		var el = document.querySelector('.chat-messages');
		if (!el) return 'no el';
		return 'scrollTop-js=' + el.scrollTop;
	})()`)
	fmt.Printf("[sbot] JS scrollTop=%s\n", eng)
	fmt.Printf("[sbot] ==== 若 after.atBottom=true → 跳底链路正常；若 scrollH<=clientH 或 scrollTop 未变 → 跳底失败 ====\n")
	fmt.Printf("[sbot] done\n")
}
