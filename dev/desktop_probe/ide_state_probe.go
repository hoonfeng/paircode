// Command ide_state_probe loads the companion frontend through wb-ui and
// evaluates the actual JS state behind the plan-container v-if discrepancy.
package main

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(0)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, err := os.ReadFile(distDir + "/index.html")
	if err != nil {
		log.Fatalf("read index.html: %v", err)
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
				re := regexp.MustCompile(`\[data-v-[a-f0-9]+\]`)
				return re.ReplaceAllString(string(data), ""), nil
			}
		}
	}
	webkit.BeforePageScripts = func(rt *jsc.Interpreter) {
		rt.RunJS(`(function(){
			if (!window.__origFetch) {
				window.__origFetch = window.fetch;
				window.fetch = function(url, opts) {
					var u = String(url);
					if (u.indexOf('/api/') === 0) {
						return Promise.resolve(new Response('{"ok":true,"data":[]}', {
							status: 200, headers: {'Content-Type':'application/json'}
						}));
					}
					return window.__origFetch.apply(window, arguments);
				};
			}
			// 记录 Vue 是否调用 setAttribute('data-v-*')
			if (!window.__dvLog) {
				window.__dvLog = 0;
				var origSA = Element.prototype.setAttribute;
				Element.prototype.setAttribute = function(k, v) {
					if (String(k).indexOf('data-v') === 0) window.__dvLog++;
					return origSA.call(this, k, v);
				};
			}
		})()`)
	}
	_ = wv.JSInterpreter()
	if err := wv.LoadHTML(string(htmlData)); err != nil {
		log.Fatalf("LoadHTML: %v", err)
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()

	rt := wv.JSInterpreter()
	if rt == nil {
		log.Fatal("no JS interpreter")
	}

	// 检查 DOM 状态
	checks := []string{
		`!!document.querySelector('.plan-container')`,
		`(document.querySelector('.plan-container')||{}).className`,
		`(document.querySelector('.plan-container')||{}).getAttribute('class')`,
		`JSON.stringify([...document.querySelectorAll('.chat-area > *')].map(e=>e.className))`,
		`JSON.stringify([...document.querySelectorAll('.terminal-panel > *')].map(e=>e.className))`,
		`(document.querySelector('.term-empty')!==null)`,
		`(document.querySelector('.term-content')!==null)`,
		`(document.querySelector('.title-right')||{}).innerHTML`,
		`(document.querySelector('.title-center')||{}).textContent`,
		// chat-area 的 flex-direction 实际值
		`getComputedStyle ? (getComputedStyle(document.querySelector('.chat-area')||document.body)||{}).flexDirection : 'NO_GCS'`,
		`getComputedStyle ? (getComputedStyle(document.querySelector('.chat-area')||document.body)||{}).display : 'NO_GCS'`,
		`getComputedStyle ? (getComputedStyle(document.querySelector('.chat-area')||document.body)||{}).flexDirection : 'NO_GCS'`,
		`getComputedStyle ? JSON.stringify({dir:getComputedStyle(document.querySelector('.chat-area')||document.body).flexDirection, disp:getComputedStyle(document.querySelector('.chat-area')||document.body).display}) : 'NO_GCS'`,
		// DOM 是否有 Vue scoped data-v 属性
		`JSON.stringify(Object.keys(document.querySelector('.plan-container')||{}).filter(k=>k.startsWith('data-v')))`,
		`(document.querySelector('.plan-container')||{}).getAttribute('data-v-1aba76c4')`,
		`[...document.querySelector('.plan-container').attributes].map(a=>a.name).join(',')`,
		// setAttribute 基础能力
		`(function(){var d=document.createElement('div');d.setAttribute('data-v-x','');return d.getAttribute('data-v-x')===''?'SET_OK':'SET_FAIL';})()`,
		`(function(){var d=document.createElement('div');d.setAttribute('data-v-y','v');return d.outerHTML||d.getAttribute('data-v-y');})()`,
		// Vue app 挂载状态
		`JSON.stringify(Object.getOwnPropertyNames(document.querySelector('#app')||{}).filter(k=>k.startsWith('__')))`,
		`(document.querySelector('#app')||{}).__vue_app__ ? 'HAS_APP' : 'NO_APP'`,
		`(function(){var n=document.querySelector('#app')._vnode; return n ? JSON.stringify({scopeId:n.scopeId, type:(n.type&&n.type.__scopeId)||null}) : 'NO_VNODE';})()`,
		`'data-v setAttribute count: ' + (window.__dvLog||0)`,
	}
	for _, c := range checks {
		v, err := rt.RunJS(c)
		if err != nil {
			log.Printf("eval %q → ERR %v", c, err)
		} else {
			log.Printf("eval %q → %v", c, v)
		}
	}
}
