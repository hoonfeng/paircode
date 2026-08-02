// Command vue_scope_probe loads the Vue global build through wb-ui's JS engine
// (goja) and renders a minimal __scopeId component, then checks whether the
// data-v attribute lands on the DOM. This isolates whether Vue 3.5 scopeId
// propagation is broken at the ENGINE level or only for companion's bundle.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

const testHTML = `<!DOCTYPE html>
<html><head>
<meta charset="utf-8">
</head><body>
<div id="app"></div>
<script>
window.__scopeLog = [];
(function(){
	var origSA = Element.prototype.setAttribute;
	Element.prototype.setAttribute = function(k, v) {
		if (String(k).indexOf('data-v') === 0) window.__scopeLog.push(k);
		return origSA.call(this, k, v);
	};
})();
</script>
</body></html>`

func main() {
	log.SetFlags(0)
	vuePath := filepath.Join("..", "wb-ui", "dev", "vue.global.prod.js")
	vueJS, err := os.ReadFile(vuePath)
	if err != nil {
		log.Fatalf("read vue.global.prod.js: %v", err)
	}
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style}, text)
	}

	wv := webkit.NewWebView()
	_ = wv.JSInterpreter()
	if err := wv.LoadHTML(testHTML); err != nil {
		log.Fatalf("LoadHTML: %v", err)
	}

	rt := wv.JSInterpreter()
	if rt == nil {
		log.Fatal("no JS interpreter")
	}
	// 加载 Vue 全局版
	if _, err := rt.RunJS(string(vueJS)); err != nil {
		log.Fatalf("load vue: %v", err)
	}

	// 最小 scoped 组件
	scoped := `
(function(){
	var Comp = {
		template: '<div class="inner"><span class="leaf">hello</span></div>',
		__scopeId: 'data-v-test123'
	};
	Vue.createApp(Comp).mount('#app');
})();
`
	if _, err := rt.RunJS(scoped); err != nil {
		log.Fatalf("mount scoped comp: %v", err)
	}
	// 让事件循环跑完（Vue mount 用 promise/microtask）
	_, _ = rt.RunJS(`new Promise(function(res){ setTimeout(res, 50); })`)

	checks := []string{
		`(document.querySelector('.inner')||{}).getAttribute('data-v-test123')`,
		`(document.querySelector('.leaf')||{}).getAttribute('data-v-test123')`,
		`JSON.stringify(window.__scopeLog)`,
		`document.querySelector('.inner') ? document.querySelector('.inner').outerHTML : 'NO_INNER'`,
		`typeof Vue`,
		`(document.querySelector('#app').__vue_app__ ? 'HAS_APP' : 'NO_APP')`,
		// 属性存储位置
		`JSON.stringify(Object.keys(document.querySelector('.inner')||{}).filter(k=>k.startsWith('data-v')))`,
		`(document.querySelector('.inner').getAttribute('data-v-test123') === '' ? 'EMPTY_STR' : JSON.stringify(document.querySelector('.inner').getAttribute('data-v-test123')))`,
		`JSON.stringify((document.querySelector('.inner').attributes||[]).length)`,
		`(typeof document.querySelector('.inner').setAttribute === 'function' ? 'HAS_SA' : 'NO_SA')`,
		`(document.querySelector('.inner').setAttribute === Element.prototype.setAttribute ? 'SAME_REF' : 'DIFF_REF')`,
	}
	for _, c := range checks {
		v, err := rt.RunJS(c)
		if err != nil {
			fmt.Printf("eval %q → ERR %v\n", c, err)
		} else {
			fmt.Printf("eval %q → %v\n", c, v)
		}
	}
}