// PairCode IDE 桌面端入口。
//go:build windows && cgo

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"wb-ui/app"
	"wb-ui/jsc"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/bridge"
	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/server/handler"
)

var version = "v1.0.6-desktop"

func main() {
	runtime.LockOSThread()
	log.Printf("[Desktop] PairCode IDE 桌面版 %s", version)
	core.Load()
	core.LoadLastProject()

	reg := bridge.NewRegistry()
	handler.RegisterAll(&handler.Router{Reg: reg})
	log.Printf("[Desktop] 已注册 %d 个 Handler", len(reg.AllRoutes()))

	wv := webkit.NewWebView()
	wv.Resize(1280, 800)

	distDir := "cmd/desktop/web-ui/dist"
	setupLoaders(wv, distDir)
	registerDesktopBridge(wv, reg)

	// Vue/Vite expected globals
	wv.EvalJS(`window.process={env:{NODE_ENV:"production","__NODE_ENV":"production"}}`)
	wv.EvalJS(`window.__VUE_OPTIONS_API__=true;window.__VUE_PROD_DEVTOOLS__=false`)
	wv.EvalJS(`window.__VUE_PROD_HYDRATION_MISMATCH_DETAILS__=false`)
	// Full polyfills based on WebKit reference — all missing Array/String/Number/ methods
	wv.EvalJS(`Object.getOwnPropertyNames=function(o){if(!o)return [];var k=[];for(var n in o)k.push(n);return k}`)
	wv.EvalJS(`Object.hasOwn=function(o,p){return Object.prototype.hasOwnProperty.call(o,p)}`)
	wv.EvalJS(`Object.fromEntries=function(e){var r={};for(var i=0;e&&i<e.length;i++)if(e[i])r[e[i][0]]=e[i][1];return r}`)
	wv.EvalJS(`if(!Array.prototype.flatMap)Array.prototype.flatMap=function(f){var r=[];for(var i=0;i<this.length;i++){var v=f(this[i],i,this);if(v&&v.length)for(var j=0;j<v.length;j++)r.push(v[j]);else r.push(v)}return r}`)
	wv.EvalJS(`if(!Array.prototype.join)Array.prototype.join=function(s){s=s!==undefined?s:',';var r='';for(var i=0;i<this.length;i++){if(i>0)r+=s;if(this[i]!==null&&this[i]!==undefined)r+=this[i]}return r}`)
	wv.EvalJS(`if(!Array.prototype.keys)Array.prototype.keys=function(){var i=0;return{next:function(){return i<this.length?{value:i++,done:false}:{done:true}}}}`)
	wv.EvalJS(`if(!Array.prototype.reduceRight)Array.prototype.reduceRight=function(f,i){var a=this;var s=i!==undefined?a.length-1:a.length-2;var v=i!==undefined?i:a[a.length-1];for(var j=s;j>=0;j--)v=f(v,a[j],j,a);return v}`)
	wv.EvalJS(`if(!Array.prototype.at)Array.prototype.at=function(i){var n=Number(i);if(isNaN(n))n=0;var l=this.length;n=n>=0?n:l+n;if(n<0||n>=l)return undefined;return this[n]}`)
	wv.EvalJS(`console.log('POLYFILLS_OK')`)
	// Set a global marker before loading Vue
	wv.EvalJS(`window.__VUE_STAGE__='before_loadhtml'`)
	htmlData, _ := os.ReadFile(distDir + "/index.html")
	html := strings.Replace(string(htmlData), "<head>", `<head><script>window.__errors=[];window.onerror=function(m){window.__errors.push(''+m);window.__VUE_ERR__=m};console.log('CONSOLE_OK')</script>`, 1)
	wv.LoadHTML(html)
	log.Println("[Desktop] 前端页面已加载")
	// Check stage after load
	wv.EvalJS(`console.log('STAGE_AFTER_LOAD: '+(window.__VUE_STAGE__||'NOT_SET'))`)
	wv.EvalJS(`if(window.__VUE_ERR__)console.log('VUE_ERR_CAUGHT:',window.__VUE_ERR__)`)

	// Test: try a minimal Vue-like script
	wv.EvalJS(`console.log('TEST: globalThis===window='+(globalThis===window))`)
	wv.EvalJS(`console.log('TEST: typeof createApp='+typeof window.createApp)`)
	// Check if the script actually ran: look for _Vue or other global from the iife
	wv.EvalJS(`console.log('TEST: _Vue='+typeof window._Vue+', $confirm='+typeof window.$confirm+', $toast='+typeof window.$toast)`)
	wv.EvalJS(`try{var t=document.getElementById('app');console.log('TEST: #app='+(t!==null)+' inner="'+t.innerHTML.substring(0,100)+'"')}catch(e){console.log('TEST_ERR:'+e)}`)

	if out := wv.ConsoleOutput(); out != "" {
		log.Printf("[CONSOLE]\n%s", out)
	}

	// JS 诊断
	wv.EvalJS(`(function(){
		console.log('DIAG: app='+(document.getElementById('app')!==null)+' qs='+(document.querySelector('#app')!==null));
		console.log('MOUNT_REACHED: '+(window.__MOUNT_REACHED__===true?'YES':'NO:'+typeof window.__MOUNT_REACHED__));
		console.log('STAGES: '+(window.__VUE_STAGE__||'NOT_SET'));
		console.log('ERR_CNT: '+(window.__errors?window.__errors.length:0));
		if(window.__errors&&window.__errors.length) console.log('ERR:',window.__errors.join('|'));
		// Check key Vue APIs
		var apis = ['createApp','reactive','ref','computed','defineComponent'];
		for(var i=0;i<apis.length;i++){
			try { console.log('API_'+apis[i]+': '+(typeof eval(apis[i]))); } catch(e){ console.log('API_'+apis[i]+': ERR:'+e); }
		}
		try {
			var app = document.getElementById('app');
			if(app) {
				console.log('APP: children='+app.childNodes.length+' html='+(app.innerHTML||'').substring(0,100));
				if(app.__vue_app__) console.log('VUE_APP: mounted');
				else console.log('VUE_APP: NOT found');
			}
			// Test DOM manipulation
			var d = document.createElement('div');
			d.id = 'test-div';
			d.textContent = 'hello world';
			document.body.appendChild(d);
			var testEl = document.getElementById('test-div');
			console.log('DOM_TEST: found='+(testEl!==null)+' text='+(testEl?testEl.textContent:'n/a'));
			// Test app.appendChild
			var child = document.createElement('span');
			child.textContent = 'test-child';
			app.appendChild(child);
			console.log('APPEND_TEST: app.children now='+app.childNodes.length);
		} catch(e) { console.log('APP_ERR:'+e); }
	})()`)

	if out2 := wv.ConsoleOutput(); out2 != "" {
		log.Printf("[CONSOLE2]\n%s", out2)
	}

	// DOM 检查
	if doc := wv.Document(); doc != nil {
		if app := doc.GetElementById("app"); app != nil {
			log.Printf("[DOM] #app children: %d", len(app.ChildNodes()))
		}
	}

	wv.RebuildRenderTree()

	host, _ := app.NewHost(wv, 1280, 800, "PairCode IDE")
	log.Println("[Desktop] 窗口已启动")
	host.Run()
	log.Println("[Desktop] 已退出。")
}

func setupLoaders(wv *webkit.WebView, distDir string) {
	absDist, _ := filepath.Abs(distDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				data, _ := os.ReadFile(filepath.Join(absDist, strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")))
				return string(data), nil
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				data, _ := os.ReadFile(filepath.Join(absDist, strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")))
				return string(data), nil
			}
		}
	}
}

func registerDesktopBridge(wv *webkit.WebView, reg *bridge.Registry) {
	interp := wv.JSInterpreter()
	logger := &jsc.BufferLogger{}
	interp.SetupGlobal(logger)
	wv.SetConsoleLogger(logger)
	log.Println("[Desktop] 注册 desktopBridge...")
	bridgeObj := jsc.NewObject(interp.ObjectPrototype())
	bridgeCall := jsc.NewNativeFunction("bridgeCall", func(in *jsc.Interpreter, this jsc.JSValue, args []jsc.JSValue) jsc.JSValue {
		if len(args) < 2 { return errResult("need method and path") }
		m, p := safeStr(args[0]), safeStr(args[1])
		b, q := "", ""
		if len(args) > 2 { b = safeStr(args[2]) }
		if len(args) > 3 { q = safeStr(args[3]) }
		req := fmt.Sprintf(`{"method":"%s","path":"%s","body":%s,"params":%s}`, escStr(m), escStr(p), maybeJSON(b), maybeJSON(q))
		return jsc.StringValue(reg.HandleBridgeCall(req))
	}, 4)
	bridgeObj.Set("call", jsc.FunctionValue(bridgeCall))
	bridgeObj.Set("onAgentEvent", jsc.FunctionValue(jsc.NewNativeFunction("onAgentEvent",
		func(in *jsc.Interpreter, this jsc.JSValue, args []jsc.JSValue) jsc.JSValue { return jsc.Undefined() }, 2)))
	bridgeObj.Set("onStatus", jsc.FunctionValue(jsc.NewNativeFunction("onStatus",
		func(in *jsc.Interpreter, this jsc.JSValue, args []jsc.JSValue) jsc.JSValue { return jsc.Undefined() }, 1)))
	interp.GlobalObject().Set("desktopBridge", jsc.ObjectValue(bridgeObj))
	interp.GlobalObject().Set("__DESKTOP_MODE__", jsc.BooleanValue(true))
	log.Println("[Desktop] desktopBridge 注册完成")
}

func errResult(msg string) jsc.JSValue {
	r, _ := json.Marshal(bridge.BridgeCallResponse{Status: 400, Body: fmt.Sprintf(`{"error":"%s"}`, msg)})
	return jsc.StringValue(string(r))
}
func safeStr(v jsc.JSValue) string { if v.IsString() { return v.AsString() }; return fmt.Sprint(v) }
func escStr(s string) string        { return strings.ReplaceAll(s, `"`, `\"`) }
func maybeJSON(s string) string     { if s == "" { return `""` }; return s }
