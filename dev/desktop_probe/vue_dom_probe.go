// Command vue_dom_probe 直接验证 wb-ui 渲染的 DOM 上 Vue 组件实例标记：
// 1. conv-item 元素上有哪些 __vue 属性（生产构建可能移除 __vueParentComponent）
// 2. 通过组件实例手动修改 state.currentConvId，验证响应式是否驱动 DOM 更新
// 3. 验证 Vue 的 effect 调度（nextTick/flushJobs）是否执行
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
	for i := 0; i < 10; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
		time.Sleep(300 * time.Millisecond)
	}
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 800); })`)
	time.Sleep(800 * time.Millisecond)
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}

	// 1) 收集 .conv-item 的 DOM 属性（找 Vue 组件标记）
	v, err := wv.JSInterpreter().RunJS(`(function(){
		var out = {};
		var items = document.querySelectorAll('.conv-item');
		out.count = items.length;
		// 元素上的所有自有属性名
		var it = items[1];
		var keys = [];
		for (var k in it) { keys.push(k); }
		out.elementKeys = keys.join(',');
		// 常见的 Vue 标记
		out.hasVueParent = ('__vueParentComponent' in it);
		out.hasVue = ('__vue__' in it);
		out.hasVnode = ('__vnode' in it);
		// 尝试读组件实例
		try {
			var inst = it.__vueParentComponent;
			if (inst) {
				out.instType = inst.type ? (inst.type.name || inst.type.__name || 'anon') : '?';
				out.hasSetupState = !!inst.setupState;
				if (inst.setupState && inst.setupState.state) {
					out.cur = inst.setupState.state.currentConvId;
					out.stateKeys = Object.keys(inst.setupState.state).slice(0,20).join(',');
					out.hasSwitchConv = typeof inst.setupState.switchConv;
				}
			}
		} catch(e) { out.instErr = String(e).slice(0,120); }
		return JSON.stringify(out);
	})()`)
	if err != nil {
		log.Printf("RunJS err: %v", err)
	} else {
		fmt.Printf("[vue] DOM 标记: %s\n", v.ToString())
	}

	// 2) 通过 #app.__vue_app__ 访问 app，检查 _instance 与组件树
	v2, err := wv.JSInterpreter().RunJS(`(function(){
		var out = {};
		var el = document.querySelector('#app');
		var app = el.__vue_app__;
		if (!app) return JSON.stringify({err:'no app'});
		out.app = true;
		out.instance = (app._instance ? 'yes type=' + (app._instance.type ? (app._instance.type.name||app._instance.type.__name||'anon') : '?') : 'null');
		// app._instance 为 null 时尝试找 container._vnode
		out.containerVnode = (el._vnode ? 'yes' : 'no');
		// Vue 全局钩子
		out.vueHook = (window.__VUE__ ? 'yes' : 'no');
		// 尝试从全局找组件
		return JSON.stringify(out);
	})()`)
	if err != nil {
		log.Printf("RunJS2 err: %v", err)
	} else {
		fmt.Printf("[vue] app 信息: %s\n", v2.ToString())
	}

	// 3) 决定性实验：如果拿得到组件实例，手动改 state.currentConvId
	v3, err := wv.JSInterpreter().RunJS(`(function(){
		var out = {};
		var items = document.querySelectorAll('.conv-item');
		if (!items[1]) return JSON.stringify({err:'no items'});
		var inst = items[1].__vueParentComponent;
		if (!inst || !inst.setupState || !inst.setupState.state) {
			// 尝试从 #app 的 vnode 树找 RightPanel
			return JSON.stringify({err:'no instance', hasVueParent: ('__vueParentComponent' in items[1])});
		}
		var state = inst.setupState.state;
		out.before = state.currentConvId;
		// 手动触发响应式更新
		state.currentConvId = 'conv_' + Date.now();
		out.after = state.currentConvId;
		// 检查 DOM 是否更新（active 切换）
		return JSON.stringify(out);
	})()`)
	if err != nil {
		log.Printf("RunJS3 err: %v", err)
	} else {
		fmt.Printf("[vue] 手动改 state: %s\n", v3.ToString())
	}

	// flush
	if wv.JSInterpreter() != nil {
		for i := 0; i < 10; i++ {
			wv.JSInterpreter().RunJobs()
			process(wv)
		}
		for i := 0; i < 4; i++ {
			time.Sleep(15 * time.Millisecond)
			process(wv)
			wv.JSInterpreter().RunJobs()
		}
	}
	wv.RebuildRenderTree()

	// 4) 读回 active 状态
	v4, err := wv.JSInterpreter().RunJS(`(function(){
		var items = document.querySelectorAll('.conv-item');
		var act = [];
		for (var i=0;i<items.length;i++) if (items[i].className.indexOf('active')>=0) act.push(i);
		var actTitle = '';
		for (var i=0;i<items.length;i++) if (items[i].className.indexOf('active')>=0) { actTitle = items[i].textContent.slice(0,20); break; }
		return JSON.stringify({active: act, activeTitle: actTitle});
	})()`)
	if err != nil {
		log.Printf("RunJS4 err: %v", err)
	} else {
		fmt.Printf("[vue] 手动修改后 DOM: %s\n", v4.ToString())
	}
}

func process(wv *webkit.WebView) {
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.MarkRenderTreeDirty()
		}
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
