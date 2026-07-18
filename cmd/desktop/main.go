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
	// polyfill: safe style setProperty
	wv.EvalJS(`if(window.CSSStyleDeclaration&&CSSStyleDeclaration.prototype.setProperty){var _osp=CSSStyleDeclaration.prototype.setProperty;CSSStyleDeclaration.prototype.setProperty=function(p,v,i){try{return _osp.call(this,p,String(v),i)}catch(e){return}}}`)
	
	htmlData, _ := os.ReadFile(distDir + "/index.html")
	html := strings.Replace(string(htmlData), "<head>", `<head><script>window.__errors=[];window.onerror=function(m){window.__errors.push(''+m)};console.log('CONSOLE_OK')</script>`, 1)
	wv.LoadHTML(html)
	log.Println("[Desktop] 前端页面已加载")

	if out := wv.ConsoleOutput(); out != "" {
		log.Printf("[CONSOLE]\n%s", out)
	}

	// JS 诊断
	wv.EvalJS(`(function(){
		console.log('DIAG: app='+(document.getElementById('app')!==null)+' qs='+(document.querySelector('#app')!==null));
		if(window.__errors&&window.__errors.length) console.log('ERR:',window.__errors.join(';'));
		try {
			var app = document.getElementById('app');
			if(app) console.log('APP: children='+app.childNodes.length+' html='+(app.innerHTML||'').substring(0,100));
		} catch(e) { console.log('APP_ERR:'+e); }
	})()`)

	if out2 := wv.ConsoleOutput(); out2 != "" {
		log.Printf("[CONSOLE2]\n%s", out2)
	}

	// 从 Go DOM 检查
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
