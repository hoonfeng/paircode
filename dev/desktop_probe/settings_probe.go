// Command settings_probe 在 wb-ui（goja）引擎中加载真实 dist：
//   1. 打开设置面板（点击 activity-bar 设置按钮）→ 输出 modal/tabs/rows 的 DOM 诊断
//   2. 打开工具配置弹窗（点击 RightPanel 工具配置按钮）→ 输出 popover 诊断
// 与浏览器（9090 web_debug element_query）对比定位「设置面板/工具配置展示异常」。
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

func js(wv *webkit.WebView, code string) string {
	iv, err := wv.JSInterpreter().RunJS(code)
	if err != nil {
		return "[ERR] " + err.Error()
	}
	return iv.ToString()
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

	// ── 1. 打开设置面板 ──
	fmt.Println("[settings] 打开设置面板:")
	fmt.Println("  " + js(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		var out = {btnFound: !!btn};
		if (btn) fireClick(btn);
		return JSON.stringify(out);
		function fireClick(el){ var ev = new Event('click', {bubbles:true}); el.dispatchEvent(ev); }
	})()`))
	// 等 Vue 渲染 SettingsModal
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(350 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	fmt.Println("  " + js(wv, `(function(){
		var out = {};
		var ov = document.querySelector('.settings-modal');
		out.modal = ov ? rect(ov) : null;
		// 细分：header/body/footer/settings-body/tabs 容器
		var h = document.querySelector('.settings-modal .modal-header');
		var b = document.querySelector('.settings-modal .modal-body');
		var f = document.querySelector('.settings-modal .modal-footer');
		var sb = document.querySelector('.settings-modal .settings-body');
		var tabsBox = document.querySelector('.settings-modal .settings-tabs');
		out.header = h ? rect(h) : null;
		out.body = b ? rect(b) : null;
		out.footer = f ? rect(f) : null;
		out.settingsBody = sb ? rect(sb) : null;
		out.tabsBox = tabsBox ? rect(tabsBox) : null;
		out.settingsBodyScrollH = sb ? sb.scrollHeight : null;
		out.settingsBodyClientH = sb ? sb.clientHeight : null;
		out.tabs = [];
		var btns = document.querySelectorAll('.settings-tabs button');
		for (var i=0;i<btns.length;i++) {
			var b2 = btns[i];
			out.tabs.push({label: b2.textContent.trim(), active: b2.className.indexOf('active')>=0, rect: rect(b2)});
		}
		out.rows = document.querySelectorAll('.setting-row').length;
		out.groups = document.querySelectorAll('.setting-group').length;
		out.allRows = [];
		var rows2 = document.querySelectorAll('.setting-row');
		for (var i=0;i<rows2.length;i++) {
			var r = rows2[i];
			var label = r.querySelector('label');
			var input = r.querySelector('input, select');
			out.allRows.push({
				i: i,
				label: label ? label.textContent.trim().slice(0,8) : null,
				labelY: label ? Math.round(label.getBoundingClientRect().top) : null,
				inputY: input ? Math.round(input.getBoundingClientRect().top) : null,
				inputH: input ? Math.round(input.getBoundingClientRect().height) : null,
				rowY: Math.round(r.getBoundingClientRect().top),
				rowH: Math.round(r.getBoundingClientRect().height)
			});
		}
		out.groupTitles = [];
		var gts = document.querySelectorAll('.group-title');
		for (var i=0;i<gts.length;i++) {
			out.groupTitles.push({text: gts[i].textContent.trim().slice(0,10),
				y: Math.round(gts[i].getBoundingClientRect().top),
				h: Math.round(gts[i].getBoundingClientRect().height)});
		}
		out.firstRows = [];
		var rows = document.querySelectorAll('.setting-row');
		for (var i=0;i<rows.length && i<4;i++) {
			var r = rows[i];
			var label = r.querySelector('label');
			var input = r.querySelector('input, select');
			out.firstRows.push({
				label: label ? (label.textContent.trim().slice(0,10)) : null,
				labelRect: label ? rect(label) : null,
				inputRect: input ? rect(input) : null,
				rowRect: rect(r)
			});
		}
		return JSON.stringify(out);
		function rect(el){
			var r = el.getBoundingClientRect();
			return {x:Math.round(r.left), y:Math.round(r.top), w:Math.round(r.width), h:Math.round(r.height)};
		}
	})()`))

	// ── 2. 打开工具配置弹窗（RightPanel 工具配置按钮）──
	fmt.Println("[toolcfg] 打开工具配置弹窗:")
	// 先确保右侧面板可见（chat activity）
	fmt.Println("  " + js(wv, `(function(){
		var st = window.__state;
		var out = {chatVisible: st ? !!st.rightPanelVisible : 'no-state'};
		if (st && !st.rightPanelVisible) { st.rightPanelVisible = true; }
		return JSON.stringify(out);
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(350 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	fmt.Println("  " + js(wv, `(function(){
		var btn = document.querySelector('.obtn-review-config');
		var out = {btnFound: !!btn};
		if (btn) fireClick(btn);
		return JSON.stringify(out);
		function fireClick(el){ var ev = new Event('click', {bubbles:true}); el.dispatchEvent(ev); }
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(350 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	fmt.Println("  " + js(wv, `(function(){
		var out = {};
		var pop = document.querySelector('.tool-config-popover');
		out.popover = pop ? rect(pop) : null;
		out.tabs = [];
		var tabs = document.querySelectorAll('.tcp-tab');
		for (var i=0;i<tabs.length;i++) { out.tabs.push({label:tabs[i].textContent.trim(), active:tabs[i].className.indexOf('active')>=0}); }
		out.switchItems = document.querySelectorAll('.tcp-switch-item').length;
		out.catHeaders = document.querySelectorAll('.tcp-cat-header').length;
		out.switchList = !!document.querySelector('.tcp-switch-list');
		out.reviewList = !!document.querySelector('.tcp-review-list');
		// 头部按钮（与浏览器对比按钮溢出）
		var hbtns = document.querySelectorAll('.rcp-header button, .rcp-btn-mini');
		out.headerBtns = [];
		for (var i=0;i<hbtns.length;i++) { out.headerBtns.push({text:hbtns[i].textContent.trim().slice(0,8), rect: rect(hbtns[i])}); }
		// popover 内部文本（区分加载中/加载失败/列表）
		var pop = document.querySelector('.tool-config-popover');
		out.popText = pop ? (pop.textContent||'').trim().slice(0, 80) : null;
		out.popChildren = pop ? pop.children.length : 0;
		var sp = document.querySelector('.tcp-panel');
		out.switchPanelText = sp ? (sp.textContent||'').trim().slice(0,80) : null;
		out.switchPanelRect = sp ? rect(sp) : null;
		return JSON.stringify(out);
		function rect(el){
			var r = el.getBoundingClientRect();
			return {x:Math.round(r.left), y:Math.round(r.top), w:Math.round(r.width), h:Math.round(r.height)};
		}
	})()`))

	// ── 3. 布局差异诊断（input flex / modal max-height / panel 高度） ──
	fmt.Println("[diag] 布局诊断:")
	// 重新打开设置面板（toolcfg 打开后 modal 可能已关闭）
	fmt.Println("  " + js(wv, `(function(){
		var btn = document.querySelector('.activity-bottom button');
		if (btn) { var ev = new Event('click', {bubbles:true}); btn.dispatchEvent(ev); }
		return 'clicked';
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(350 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("  " + js(wv, `(function(){
		var out = {};
		out.attrText = document.querySelectorAll('input[type="text"]').length;
		out.allInputs = document.querySelectorAll('input').length;
		var inp = document.querySelector('.setting-row input[type="text"]');
		if (inp) {
			var cs = getComputedStyle(inp);
			out.inputFlex = cs.flex;
			out.inputFlexGrow = cs.flexGrow;
			out.inputFlexBasis = cs.flexBasis;
			out.inputW = Math.round(inp.getBoundingClientRect().width);
			out.inputDisplay = cs.display;
			out.inputWidthCss = cs.width;
		}
		var sel = document.querySelector('.setting-row select');
		if (sel) {
			var cs2 = getComputedStyle(sel);
			out.selectFlex = cs2.flex;
			out.selectW = Math.round(sel.getBoundingClientRect().width);
		}
		var m = document.querySelector('.settings-modal');
		if (m) {
			var cm = getComputedStyle(m);
			out.modalH = Math.round(m.getBoundingClientRect().height);
			out.modalMaxH = cm.maxHeight;
			out.modalHeightCss = cm.height;
			out.modalFlexDir = cm.flexDirection;
			out.modalBodyH = (function(){ var b=document.querySelector('.modal-body'); return b?Math.round(b.getBoundingClientRect().height):null; })();
		}
		var sp = document.querySelector('.tcp-panel');
		if (sp) {
			var cs3 = getComputedStyle(sp);
			out.panelH = Math.round(sp.getBoundingClientRect().height);
			out.panelFlex = cs3.flex;
			out.panelOverflow = cs3.overflow;
			out.panelHeightCss = cs3.height;
			out.panelDisplay = cs3.display;
		}
		var pp = document.querySelector('.tool-config-popover');
		if (pp) {
			var cs4 = getComputedStyle(pp);
			out.popH = Math.round(pp.getBoundingClientRect().height);
			out.popMaxH = cs4.maxHeight;
			out.popOverflow = cs4.overflow;
			out.popPos = cs4.position;
			out.popTop = cs4.top;
			out.popBottom = cs4.bottom;
		}
		// input type 属性细节
		var fi = document.querySelector('.setting-row input');
		if (fi) {
			out.firstInputTypeAttr = fi.getAttribute('type');
			out.firstInputTypeProp = fi.type;
			out.firstInputHtml = fi.outerHTML.slice(0, 120);
		}
		out.attrTypeText = document.querySelectorAll('[type="text"]').length;
		out.attrTypeNoQ = document.querySelectorAll('input[type=text]').length;
		out.inputTagOnly = document.querySelectorAll('input').length;
		out.labelTagOnly = document.querySelectorAll('label').length;
		out.labelClass = document.querySelectorAll('.setting-row > label').length;
		return JSON.stringify(out);
	})()`))

	// ── 4. tab 切换验证（class active 变化 → attrVersion 样式刷新） ──
	fmt.Println("[tab] 切换 Agent tab 验证 active 类切换:")
	fmt.Println("  " + js(wv, `(function(){
		var btns = document.querySelectorAll('.settings-tabs button');
		var out = {found: btns.length};
		if (btns.length > 1) {
			var ev = new Event('click', {bubbles:true});
			btns[1].dispatchEvent(ev);
		}
		return JSON.stringify(out);
	})()`))
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
	time.Sleep(350 * time.Millisecond)
	runJobs(wv)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	fmt.Println("  " + js(wv, `(function(){
		var out = {};
		var btns = document.querySelectorAll('.settings-tabs button');
		out.tabActive = [];
		for (var i=0;i<btns.length;i++) {
			var cs = getComputedStyle(btns[i]);
			out.tabActive.push({
				label: btns[i].textContent.trim(),
				active: btns[i].className.indexOf('active') >= 0,
				bg: cs.backgroundColor,
				color: cs.color
			});
		}
		// Agent 面板内容是否出现
		out.agentGroups = document.querySelectorAll('.settings-body .setting-group').length;
		out.agentFirstRow = (function(){
			var r = document.querySelector('.settings-body .setting-row label');
			return r ? r.textContent.trim().slice(0,12) : null;
		})();
		return JSON.stringify(out);
	})()`))

	// console 错误
	cout := wv.ConsoleOutput()
	lines := strings.Split(cout, "\n")
	errLines := 0
	for _, ln := range lines {
		if strings.Contains(ln, "Error") || strings.Contains(ln, "Cannot") || strings.Contains(ln, "undefined") {
			fmt.Println("[jserr]", ln)
			errLines++
			if errLines > 10 {
				break
			}
		}
	}
	fmt.Println("[console] total lines:", len(lines))
}
