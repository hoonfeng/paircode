// Command conv_click_min 用「独立简单 Vue 项目」验证会话列表点击选中问题。
// 使用 node_modules 里真实 dist 同版本 Vue（vue.global.prod.js, 3.5.x），
// 模拟 ConvSidebar 的核心结构：v-for 会话列表 + :class="['conv-item', {active}]"
// + @click 切换 currentConvId。若此最小场景可复现 patch 崩溃/active 不切换，
// 则问题在引擎/Vue runtime 层；否则问题在应用层（组件复杂度触发）。
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()

	// 优先用真实 dist 同版本 Vue：cmd/companion/web-ui/node_modules/vue/dist/vue.global.prod.js
	vueCandidates := []string{
		filepath.Join(wd, "cmd", "companion", "web-ui", "node_modules", "vue", "dist", "vue.global.prod.js"),
		filepath.Join(wd, "..", "wb-ui", "dev", "vue.global.prod.js"),
	}
	var vueData []byte
	var vuePath string
	for _, p := range vueCandidates {
		d, err := os.ReadFile(p)
		if err == nil {
			vueData = d
			vuePath = p
			break
		}
	}
	if vueData == nil {
		log.Fatal("no vue.global.prod.js found")
	}
	fmt.Printf("[min] vue src: %s len=%d\n", vuePath, len(vueData))

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

	// ★ 与 ConvSidebar.vue 结构对齐的最小 app（内联模板模拟 SFC 编译结果）
	// 注意：ConvSidebar 改成【多根组件】（header + list 平级），模拟 GlobalDialogs.vue
	// 的多根结构——Vue 会为多根组件创建 Fragment（注释锚点）。验证 Fragment 锚点
	// 是否被 wb-ui 错误挂到 body。
	html := `<!DOCTYPE html><html><head><style>
		.conv-item { padding: 4px; }
		.conv-item.active { background: #3366ff; }
	</style></head><body>
	<div id="app"></div>
	<script type="module">
	const { createApp, reactive, ref } = Vue
	// 模拟 store：conversations + currentConvId
	const state = reactive({
		conversations: [
			{ id: 'c1', title: '会话一', msgCount: 3 },
			{ id: 'c2', title: '会话二', msgCount: 5 },
			{ id: 'c3', title: '会话三', msgCount: 7 },
		],
		currentConvId: 'c1'
	})
	// ★ 多根组件：两个 div 平级（无包裹）→ Vue 生成 Fragment
	const ConvSidebar = {
		props: { currentConvId: String, horizontal: Boolean },
		emits: ['switch-conversation', 'delete-conversation'],
		template: '<div class="conv-sidebar-header"><span>会话</span></div>'
			+ '<div class="conv-list">'
			+ '<div v-for="conv in localConvs" :key="conv.id"'
			+ '     :class="[\'conv-item\', { active: conv.id === currentConvId }]"'
			+ '     @click="$emit(\'switch-conversation\', conv.id)">'
			+ '  <div class="conv-title">{{ conv.title }}</div>'
			+ '  <div class="conv-meta">'
			+ '    <span class="conv-msg-count">{{ conv.msgCount || 0 }}</span>'
			+ '  </div>'
			+ '</div>'
			+ '<div v-if="localConvs.length === 0" class="conv-empty">暂无对话</div>'
			+ '</div>',
		computed: {
			localConvs() { return this.$root.$data.conversations; }
		}
	}
	const app = createApp({
		components: { ConvSidebar },
		setup() {
			return { state }
		},
		data() { return { conversations: state.conversations } },
		template: '<ConvSidebar :current-conv-id="state.currentConvId" @switch-conversation="sw" />',
		methods: {
			sw(id) { console.log('[min] switch to ' + id + ' (was ' + state.currentConvId + ')'); state.currentConvId = id; }
		}
	})
	app.mount('#app')
	window.__minState = state
	</script></body></html>`

	wv := webkit.NewWebView()
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		_, _ = rt.RunJS(string(vueData))
	}
	_ = wv.JSInterpreter()
	// 捕获 console.error / console.log（Vue warn 打到这里）
	_, _ = wv.JSInterpreter().RunJS(`
		window.__minErr = [];
		var _err = window.console.error.bind(window.console);
		window.console.error = function() {
			var m = Array.prototype.slice.call(arguments).map(String).join(' ').slice(0, 300);
			window.__minErr.push('[err] ' + m);
			return _err.apply(window.console, arguments);
		};
		var _log = window.console.log.bind(window.console);
		window.console.log = function() {
			var m = Array.prototype.slice.call(arguments).map(String).join(' ').slice(0, 200);
			window.__minErr.push('[log] ' + m);
			return _log.apply(window.console, arguments);
		};
	`)
	wv.LoadHTML(html)
	for i := 0; i < 8; i++ {
		wv.JSInterpreter().RunJobs()
		time.Sleep(120 * time.Millisecond)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()

	check := func(tag string) {
		v, err := wv.JSInterpreter().RunJS(`(function(){
			var out = [];
			var items = document.querySelectorAll('.conv-item');
			for (var i=0;i<items.length;i++) out.push((items[i].className.indexOf('active')>=0?'A':'-') + ':' + (items[i].textContent||'').slice(0,10));
			var st = window.__minState;
			return JSON.stringify({items: out, cur: st ? st.currentConvId : '?'});
		})()`)
		if err != nil {
			fmt.Printf("[min] %s err=%v\n", tag, err)
			return
		}
		fmt.Printf("[min] %s: %s\n", tag, v.ToString())
	}
	check("初始")

	// ★ 渲染树找 conv-item 几何（模拟 host.handleClick 真实路径）
	if rv != nil {
		var items []struct {
			el     *dom.Element
			x, y   float64
			active bool
		}
		var findConv func(o rendering.RenderObject)
		findConv = func(o rendering.RenderObject) {
			if el, ok := o.Node().(*dom.Element); ok {
				cn := el.GetAttribute("class")
				lb := o.LayoutBox()
				if strings.Contains(cn, "conv-item") && lb != nil && rv.LayoutState() != nil {
					g := rv.LayoutState().GeometryForBox(lb)
					items = append(items, struct {
						el     *dom.Element
						x, y   float64
						active bool
					}{el: el, x: g.Left(), y: g.Top(), active: strings.Contains(cn, "active")})
				}
			}
			for c := o.FirstChild(); c != nil; c = c.NextSibling() {
				findConv(c)
			}
		}
		findConv(rendering.RenderObject(rv))
		fmt.Printf("[min] found %d conv-items\n", len(items))
		for i, it := range items {
			active := ""
			if it.active {
				active = " ACTIVE"
			}
			fmt.Printf("[min]   #%d x=%.0f y=%.0f%s\n", i, it.x, it.y, active)
		}
		if len(items) >= 2 {
			target := items[1]
			cx := target.x + 40
			cy := target.y + 6
			fmt.Printf("[min] clicking #1 at (%.0f, %.0f)\n", cx, cy)

			el := rendering.HitTest(rv, cx, cy, "onclick")
			desc := "<nil>"
			if el != nil {
				desc = el.LocalName() + "." + el.GetAttribute("class") + " onclick=" + el.GetAttribute("onclick")
			}
			fmt.Printf("[min] HitTest(onclick) -> %s\n", desc)

			if el == nil {
				deepest := rendering.HitTest(rv, cx, cy, "")
				if deepest != nil {
					fmt.Printf("[min] HitTest(\"\") -> %s\n", deepest.LocalName())
					deepest.DispatchEvent(dom.NewMouseEvent(dom.EventClick, true, true, false))
				}
			} else {
				el.DispatchEvent(dom.NewMouseEvent(dom.EventClick, true, true, false))
			}

			for i := 0; i < 10; i++ {
				wv.JSInterpreter().RunJobs()
			}
			for i := 0; i < 6; i++ {
				time.Sleep(15 * time.Millisecond)
				wv.JSInterpreter().RunJobs()
			}
			wv.RebuildRenderTree()
			wv.EnsureLayout()
			check("after HitTest click #1")
		}
	}

	ve, _ := wv.JSInterpreter().RunJS(`JSON.stringify(window.__minErr || [])`)
	fmt.Printf("[min] console errors: %s\n", ve.ToString())
	check("最终")
}
