package main
import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"wb-ui/app"
	"wb-ui/webkit"
)
var prePolyfill = `if(!Object.getPrototypeOf)Object.getPrototypeOf=function(o){return o&&o.constructor?o.constructor.prototype:null}
if(!Object.setPrototypeOf)Object.setPrototypeOf=function(o,p){o.__proto__=p;return o}
if(typeof TextEncoder==='undefined'){TextEncoder=function(){this.encode=function(s){var arr=new Uint8Array(s.length);for(var i=0;i<s.length;i++)arr[i]=s.charCodeAt(i);return arr}}}
if(typeof TextDecoder==='undefined'){TextDecoder=function(){this.decode=function(a){return String.fromCharCode.apply(null,a)}}}
if(typeof structuredClone==='undefined')structuredClone=function(o){return JSON.parse(JSON.stringify(o))}
if(typeof crypto==='undefined')crypto={getRandomValues:function(arr){for(var i=0;i<arr.length;i++)arr[i]=Math.floor(Math.random()*256)}}
if(typeof fetch==='undefined')fetch=function(){return Promise.resolve({json:function(){return Promise.resolve({})},text:function(){return Promise.resolve('')}})}
if(typeof WebSocket==='undefined')WebSocket=function(){}
if(typeof CustomEvent==='undefined')CustomEvent=function(t,e){this.type=t||'';for(var k in(e||{}))this[k]=e[k]}
if(typeof ResizeObserver==='undefined')ResizeObserver=function(cb){this.observe=function(){};this.disconnect=function(){}}
if(typeof MutationObserver==='undefined')MutationObserver=function(cb){this.observe=function(){};this.disconnect=function(){};this.takeRecords=function(){return[]}}
if(typeof requestAnimationFrame==='undefined')requestAnimationFrame=function(fn){return setTimeout(fn,16)}
if(typeof cancelAnimationFrame==='undefined')cancelAnimationFrame=function(id){clearTimeout(id)}
if(typeof performance==='undefined')performance={now:function(){return Date.now()}}
if(typeof navigator==='undefined')navigator={userAgent:'PairCode Desktop'}
if(typeof queueMicrotask==='undefined')queueMicrotask=function(fn){Promise.resolve().then(fn)}
if(typeof Event==='undefined'){Event=function(t,e){this.type=t||'';for(var k in(e||{}))this[k]=e[k]};window.Event=Event}
if(typeof KeyboardEvent==='undefined'){KeyboardEvent=function(t,e){Event.call(this,t,e||{});this.key=(e||{}).key||'';this.ctrlKey=false};KeyboardEvent.prototype=Object.create(Event.prototype)}
if(typeof MouseEvent==='undefined'){MouseEvent=function(t,e){Event.call(this,t,e||{});this.clientX=0;this.clientY=0};MouseEvent.prototype=Object.create(Event.prototype)}
if(!Array.from)Array.from=function(a,fn,ctx){var r=[];for(var i=0;i<a.length;i++)r.push(fn?fn.call(ctx||null,a[i],i):a[i]);return r}
if(typeof window.getSelection==='undefined')window.getSelection=function(){return{anchorNode:null,anchorOffset:0,focusNode:null,focusOffset:0,rangeCount:0,getRangeAt:function(){return null},addRange:function(){},removeAllRanges:function(){}}}
`
func main() {
	log.SetFlags(log.Ltime)
	log.Println("[Desktop] PairCode IDE 桌面版 v1.0.6-desktop")

	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "web-ui", "dist")
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		distDir = filepath.Join(wd, "cmd", "desktop", "web-ui", "dist")
	}
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		distDir2 := filepath.Join(wd, "web-ui-minimal", "dist")
		if _, err2 := os.Stat(distDir2); err2 == nil {
			distDir = distDir2
		} else {
			distDir3 := filepath.Join(wd, "cmd", "desktop", "web-ui-minimal", "dist")
			if _, err3 := os.Stat(distDir3); err3 == nil {
				distDir = distDir3
			}
		}
	}
	log.Printf("[Desktop] distDir: %s", distDir)

	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()

	// Pre-polyfill: define DOM constructors and window methods BEFORE LoadHTML
	wv.EvalJS(`if(!Object.getPrototypeOf)Object.getPrototypeOf=function(o){return o&&o.constructor?o.constructor.prototype:null}`)
	wv.EvalJS(`if(!Object.setPrototypeOf)Object.setPrototypeOf=function(o,p){o.__proto__=p;return o}`)
	wv.EvalJS(prePolyfill)

	htmlData, err := os.ReadFile(distDir + "/index.html")
	s := string(htmlData)
	s = strings.Replace(s, `type="module"`, "", 1)
	s = strings.ReplaceAll(s, `crossorigin`, "")
	log.Printf("[LoadHTML] 开始加载, 大小=%d", len(s))
	err = wv.LoadHTML(s)
	if err != nil {
		log.Printf("[LoadHTML] 错误: %v", err)
	} else {
		log.Printf("[LoadHTML] 加载成功")
	}
	if out := wv.ConsoleOutput(); out != "" {
		log.Printf("[CONSOLE]\n%s", out)
	}
	// ── Init desktop bridge (Go handlers + JS bridge SDK) ──
	// Must be after LoadHTML (JS runtime ready) and before Vue mount.
	InitDesktopBridge(wv)

	// Diagnostic: check DOM after load
	wv.EvalJS(`(function(){
		var app = document.getElementById('app');
		console.log('[DIAG] getElementById("app")=' + (app ? app.id : 'null'));
		var body = document.body;
		console.log('[DIAG] body=' + (body ? body.tagName : 'null'));
		// Direct DOM check for body children
		try {
			var fc = body.firstChild;
			console.log('[DIAG] body.firstChild.type=' + typeof fc + ' val=' + fc);
			if(fc) console.log('[DIAG] body.firstChild.tagName=' + fc.tagName);
			else console.log('[DIAG] body.firstChild is null/undefined');
		} catch(e) { console.log('[DIAG] body.firstChild error: '+e); }
		// Check childNodes
		var cn = body.childNodes;
		console.log('[DIAG] body.childNodes.type=' + typeof cn + ' len=' + (cn ? cn.length : 'null'));
	})()`)
	// Vue-injected DOM and component styles.
	for i := 0; i < 8; i++ {
		wv.EnsureLayout()
		time.Sleep(100 * time.Millisecond)
	}
	wv.RebuildRenderTree()
	// Diagnostic: check if Vue rendered
	wv.EvalJS(`(function(){var a=document.getElementById('app');console.log('[DIAG2] app.childElementCount='+(a?a.childElementCount:'null')+', innerHTML.len='+(a?a.innerHTML.length:'null'))})()`)
	if out := wv.ConsoleOutput(); out != "" {
		log.Printf("[CONSOLE2]\n%s", out)
	}
	log.Println("[Desktop] window+render tree ready, creating host...")
	log.Println("[Desktop] window+render tree ready, creating host...")

	host, err := app.NewHost(wv, 1280, 800, "PairCode IDE")
	if err != nil {
		log.Printf("[Desktop] NewHost error: %v", err)
		return
	}
	log.Println("[Desktop] 窗口已启动，开始事件循环...")

	host.Run()
	log.Println("[Desktop] 已退出。")
}

func setupLoaders(wv *webkit.WebView, distDir string) {
	absDist, _ := filepath.Abs(distDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				data, err := os.ReadFile(filepath.Join(absDist, strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")))
				log.Printf("[SCRIPT] len=%d err=%v", len(data), err)
				return string(data), err
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
			data, _ := os.ReadFile(filepath.Join(absDist, strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")))
			// Remove Vue scoped [data-v-XXXXXXXX] selectors — wb-ui engine
			// does not inject data-v attributes onto DOM elements at runtime.
			re := regexp.MustCompile(`\[data-v-[a-f0-9]+\]`)
			cleaned := re.ReplaceAllString(string(data), "")
			return cleaned, nil
		}
		}
	}
}
