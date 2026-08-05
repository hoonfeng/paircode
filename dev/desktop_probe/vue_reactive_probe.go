// Command vue_reactive_probe 在 wb-ui 引擎上直接加载 Vue 全局构建，
// 验证 reactive + effect 响应式链路：set 是否触发 effect、DOM 是否更新。
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	// wb-ui 自带 Vue 全局构建
	vuePath := filepath.Join(wd, "..", "wb-ui", "dev", "vue.global.prod.js")
	vueData, err := os.ReadFile(vuePath)
	if err != nil {
		log.Fatalf("read vue.global.prod.js: %v", err)
	}
	fmt.Printf("[vue] vue.global.prod.js len=%d\n", len(vueData))

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

	// 方式 A：直接 jsc interpreter 加载 Vue + 响应式测试（不渲染 DOM）
	in := jsc.NewInterpreter()
	in.SetupGlobal(nil)
	_, err = in.RunJS(string(vueData))
	if err != nil {
		log.Fatalf("RunJS(vue) error: %v", err)
	}
	fmt.Println("[vue] Vue 加载成功")

	// 响应式最小测试：reactive + effect + 深层 set
	jsTest := `(function(){
		var out = {};
		try {
			var Vue = window.Vue;
			out.hasVue = !!Vue;
			if (!Vue) return JSON.stringify(out);
			out.hasReactive = typeof Vue.reactive;
			out.hasEffect = typeof Vue.effect;
			var state = Vue.reactive({ currentConvId: 'conv_A', obj: { n: 1 } });
			out.isProxy = (state !== Object(state).__proto__); // 粗略
			var calls = [];
			Vue.effect(function(){ calls.push('render cur=' + state.currentConvId); });
			// 触发一次同步执行
			out.callsAfterSetup = calls.join('|');
			state.currentConvId = 'conv_B';
			// effect 是同步触发的吗？
			out.callsAfterSet = calls.join('|');
			// 深层对象
			state.obj.n = 99;
			out.objRead = state.obj.n;
			// computed
			try {
				var c = Vue.computed(function(){ return state.currentConvId + '!'; });
				out.computed = c.value;
				state.currentConvId = 'conv_C';
				out.computedAfter = c.value;
			} catch(e) { out.computedErr = String(e).slice(0,100); }
			// watch
			try {
				var watched = [];
				Vue.watch(function(){ return state.currentConvId; }, function(v,o){ watched.push(v); });
				state.currentConvId = 'conv_D';
				out.watchLen = watched.length;
				out.watchVal = watched.join('|');
			} catch(e) { out.watchErr = String(e).slice(0,100); }
		} catch(e) { out.err = String(e).slice(0,200); }
		return JSON.stringify(out);
	})()`
	v, err := in.RunJS(jsTest)
	if err != nil {
		log.Fatalf("RunJS(test) error: %v", err)
	}
	fmt.Printf("[vue] 响应式测试: %s\n", v.ToString())
	// flush promise 微任务（watch/effect scheduler 可能用微任务）
	for i := 0; i < 5; i++ {
		in.RunJobs()
	}
	v2, err := in.RunJS(`JSON.stringify(window.__last || null)`)
	if err != nil {
		fmt.Println("no window.__last")
	} else {
		fmt.Printf("[vue] after jobs: %s\n", v2.ToString())
	}

	// 方式 B：webview 完整环境渲染一个 Vue 最小 app，点击验证 DOM 更新
	fmt.Println("\n=== 方式 B：WebView 完整渲染 + 点击 ===")
	html := `<!DOCTYPE html><html><head><style>
		.item { padding: 4px; }
		.item.active { background: #3366ff; }
	</style></head><body>
	<div id="app"></div>
	<script type="module">
	const { createApp, reactive } = Vue
	const state = reactive({ current: 'a' })
	const app = createApp({
		setup() {
			return {
				items: [{id:'a',t:'A'},{id:'b',t:'B'},{id:'c',t:'C'}],
				get state() { return state },
				sw(id) { state.current = id; console.log('[test] sw to ' + id + ' state.current=' + state.current); },
			}
		},
		template: '<div id="list"><div v-for="it in items" :key="it.id" class="item" :class="{active: it.id === state.current}" @click="sw(it.id)">{{it.t}}</div></div>'
	})
	app.mount('#app')
	</script></body></html>`
	wv := webkit.NewWebView()
	// BeforePageScripts 钩子注入 Vue 全局（页面 script 执行前）
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		_, _ = rt.RunJS(string(vueData))
	}
	_ = wv.JSInterpreter()
	wv.LoadHTML(html)
	for i := 0; i < 6; i++ {
		wv.JSInterpreter().RunJobs()
		time.Sleep(100 * time.Millisecond)
	}
	// 检查 DOM
	check := func(tag string) {
		v, err := wv.JSInterpreter().RunJS(`(function(){
			var out = [];
			var items = document.querySelectorAll('.item');
			for (var i=0;i<items.length;i++) out.push((items[i].className.indexOf('active')>=0?'A':'-')+':'+items[i].textContent);
			return JSON.stringify(out);
		})()`)
		if err != nil {
			fmt.Printf("[test] %s err=%v\n", tag, err)
			return
		}
		fmt.Printf("[test] %s: %s\n", tag, v.ToString())
	}
	check("初始")

	// 点击第二个 item（模拟 host.handleClick：HitTest onclick → fallback deepest dispatch）
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}
	// 找 .item 的几何（通过渲染树 HitTest 需要 LayoutState，此处用 JS dispatch 验证响应式链路）
	_ = rv
	// 直接用 JS 模拟点击：调用 Vue 的 click handler 等价于 dispatch
	// 先试 DOM dispatchEvent
	_, _ = wv.JSInterpreter().RunJS(`document.querySelectorAll('.item')[1].dispatchEvent(new MouseEvent('click', {bubbles:true}))`)
	for i := 0; i < 10; i++ {
		wv.JSInterpreter().RunJobs()
	}
	time.Sleep(200 * time.Millisecond)
	for i := 0; i < 6; i++ {
		wv.JSInterpreter().RunJobs()
	}
	check("dispatch click #1")

	// 也试 host.handleClick 真实路径（通过渲染树 HitTest）
	fmt.Println("[test] ---- 渲染树 HitTest + DispatchEvent ----")
	if rv != nil {
		// 直接通过 DOM 树找第二个 item 元素并 dispatch
		_, _ = wv.JSInterpreter().RunJS(`(function(){
			var items = document.querySelectorAll('.item');
			var el = items[2];
			var ev = new MouseEvent('click', {bubbles:true, cancelable:true});
			el.dispatchEvent(ev);
		})()`)
		for i := 0; i < 10; i++ {
			wv.JSInterpreter().RunJobs()
		}
		time.Sleep(200 * time.Millisecond)
		for i := 0; i < 6; i++ {
			wv.JSInterpreter().RunJobs()
		}
	}
	check("dispatch click #2")
}
