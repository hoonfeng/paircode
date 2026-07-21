package main
import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/app"
	"wb-ui/webkit"
)
var prePolyfill = `(function(){
// NOTE: Node/Element/HTMLElement/SVGElement/Comment/Text/DocumentFragment are
// provided by wb-ui's registerDOMBindings. Do NOT redefine them here.
// Only add polyfills for APIs wb-ui doesn't provide.
EventTarget=function EventTarget(){};
EventTarget.prototype.addEventListener=function(type,listener){if(!this._events)this._events={};if(!this._events[type])this._events[type]=[];this._events[type].push(listener)};
EventTarget.prototype.removeEventListener=function(type,listener){if(!this._events||!this._events[type])return;var idx=this._events[type].indexOf(listener);if(idx>=0)this._events[type].splice(idx,1)};
EventTarget.prototype.dispatchEvent=function(ev){if(!this._events||!this._events[ev.type])return true;var lst=this._events[ev.type].slice();for(var i=0;i<lst.length;i++){try{lst[i](ev)}catch(e){}}return !ev.defaultPrevented};
// Note: wb-ui handles core DOM classes. These stubs are only for compatibility:
if(typeof Node==='undefined'){Node=function Node(){};Node.prototype=Object.create(EventTarget.prototype)}
if(typeof Element==='undefined'){Element=function Element(){};Element.prototype=Object.create(Node.prototype)}
if(typeof SVGElement==='undefined'){SVGElement=function SVGElement(){};SVGElement.prototype=Object.create(Element.prototype)}
HTMLUnknownElement=function HTMLUnknownElement(){};HTMLUnknownElement.prototype=Object.create(HTMLElement.prototype);HTMLUnknownElement.prototype.constructor=HTMLUnknownElement;
Text=function Text(){};Text.prototype=Object.create(Node.prototype);Text.prototype.constructor=Text;
Comment=function Comment(){};Comment.prototype=Object.create(Node.prototype);Comment.prototype.constructor=Comment;
DocumentFragment=function DocumentFragment(){};DocumentFragment.prototype=Object.create(Node.prototype);DocumentFragment.prototype.constructor=DocumentFragment;
Attr=function Attr(){};Attr.prototype=Object.create(Node.prototype);Attr.prototype.constructor=Attr;
if(typeof window!=='undefined'){
window.addEventListener=function(type,listener){EventTarget.prototype.addEventListener.call(this,type,listener)};
window.removeEventListener=function(type,listener){EventTarget.prototype.removeEventListener.call(this,type,listener)};
window.dispatchEvent=function(ev){return EventTarget.prototype.dispatchEvent.call(this,ev)};
window.getComputedStyle=function(el){return el.computedStyle||{getPropertyValue:function(k){return''}}};
}
if(typeof document!=='undefined'){
document.addEventListener=function(type,listener){EventTarget.prototype.addEventListener.call(this,type,listener)};
document.removeEventListener=function(type,listener){EventTarget.prototype.removeEventListener.call(this,type,listener)};
document.dispatchEvent=function(ev){return EventTarget.prototype.dispatchEvent.call(this,ev)};
}
if(typeof console==='undefined')console={};
if(!console.error)console.error=function(){};
if(!console.warn)console.warn=function(){};
if(!console.info)console.info=function(){};
if(!console.debug)console.debug=function(){};
if(typeof performance==='undefined')performance={now:function(){return Date.now()},mark:function(){},measure:function(){},getEntries:function(){return[]},clearMarks:function(){}};
if(typeof CustomEvent==='undefined')CustomEvent=function(type,opts){var ev={type:type,bubbles:opts&&opts.bubbles||false,cancelable:opts&&opts.cancelable||false,detail:opts&&opts.detail||null,defaultPrevented:false,preventDefault:function(){this.defaultPrevented=true}};return ev};
if(typeof MouseEvent==='undefined')MouseEvent=function(type,opts){var ev={type:type,bubbles:opts&&opts.bubbles||false,cancelable:opts&&opts.cancelable||false,clientX:opts&&opts.clientX||0,clientY:opts&&opts.clientY||0,button:opts&&opts.button||0,preventDefault:function(){}};return ev};
if(typeof queueMicrotask==='undefined'){queueMicrotask=function(cb){Promise.resolve().then(cb)}}
if(typeof structuredClone==='undefined'){structuredClone=function(v){return JSON.parse(JSON.stringify(v))}}
if(typeof TextEncoder==='undefined'){
  TextEncoder=function(){this.encoding='utf-8'};
  TextEncoder.prototype.encode=function(s){
    var L=s.length,i=0,out=[];
    while(i<L){
      var c=s.charCodeAt(i++);
      if(c<0x80){out.push(c)}
      else if(c<0x800){out.push(192|c>>6,128|63&c)}
      else if(c<0xd800||c>=0xe000){out.push(224|c>>12,128|63&c>>6,128|63&c)}
      else{
        c=65536+(c-55296<<10)+(s.charCodeAt(i++)-56320);
        out.push(240|c>>18,128|63&c>>12,128|63&c>>6,128|63&c);
      }
    }
    return new Uint8Array(out);
  };
}
if(typeof TextDecoder==='undefined'){
  TextDecoder=function(){this.encoding='utf-8'};
  TextDecoder.prototype.decode=function(buf){
    var bytes=new Uint8Array(buf),i=0,L=bytes.length,out=[];
    while(i<L){
      var b=bytes[i++];
      if(b<128){out.push(String.fromCharCode(b))}
      else if(b<224){out.push(String.fromCharCode((31&b)<<6|63&bytes[i++]))}
      else if(b<240){out.push(String.fromCharCode((15&b)<<12|(63&bytes[i++])<<6|63&bytes[i++]))}
      else{
        var cp=(7&b)<<18|(63&bytes[i++])<<12|(63&bytes[i++])<<6|63&bytes[i++];
        out.push(String.fromCharCode(55296+(cp-65536>>10&1023),56320+(cp-65536&1023)));
      }
    }
    return out.join('');
  };
}
})()`

var postPolyfill = `(function(){
if(typeof MutationObserver==='undefined'){
  MutationObserver=function(cb){this._cb=cb;this._observed=[]};
  MutationObserver.prototype.observe=function(target,opts){this._observed.push({target:target,opts:opts})};
  MutationObserver.prototype.disconnect=function(){this._observed=[]};
  MutationObserver.prototype.takeRecords=function(){return[]};
}
if(typeof WebSocket==='undefined'){
  WebSocket=function(url){this.url=url;this.readyState=3;this.CONNECTING=0;this.OPEN=1;this.CLOSING=2;this.CLOSED=3};
  WebSocket.prototype.send=function(){};
  WebSocket.prototype.close=function(){};
  WebSocket.CONNECTING=0;WebSocket.OPEN=1;WebSocket.CLOSING=2;WebSocket.CLOSED=3;
}
if(typeof navigator==='undefined')navigator={onLine:true,userAgent:'Goja/1.0',language:'zh-CN',platform:'Win32'};
if(!navigator.onLine)navigator.onLine=true;
if(typeof localStorage==='undefined'){
  var store={};
  localStorage={
    getItem:function(k){return store.hasOwnProperty(k)?store[k]:null},
    setItem:function(k,v){store[k]=String(v)},
    removeItem:function(k){delete store[k]},
    clear:function(){store={}},
    key:function(i){var ks=Object.keys(store);return ks[i]||null},
    get length(){return Object.keys(store).length}
  };
}
if(typeof location==='undefined')location={href:'about:blank',origin:'file://',protocol:'file:',host:'',pathname:'/index.html',search:'',hash:''};
if(typeof history==='undefined')history={pushState:function(){},replaceState:function(){},back:function(){},forward:function(){},go:function(){},length:1,state:null};
if(typeof requestAnimationFrame==='undefined')requestAnimationFrame=function(cb){return setTimeout(cb,16)};
if(typeof cancelAnimationFrame==='undefined')cancelAnimationFrame=function(id){clearTimeout(id)};
if(typeof customElements==='undefined')customElements={define:function(){},get:function(){return null},whenDefined:function(){return Promise.resolve()}};
if(typeof matchMedia==='undefined')matchMedia=function(){return{matches:false,media:'',addListener:function(){},removeListener:function(){},addEventListener:function(){},removeEventListener:function(){}}};
if(typeof SVGElement==='undefined'&&typeof Element!=='undefined'){
  SVGElement=function SVGElement(){};SVGElement.prototype=Object.create(Element.prototype);SVGElement.prototype.constructor=SVGElement;
}
if(typeof queueMicrotask==='undefined'){queueMicrotask=function(cb){Promise.resolve().then(cb)}}
if(typeof structuredClone==='undefined'){structuredClone=function(v){return JSON.parse(JSON.stringify(v))}}
})()`

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

	// Post-polyfill: more browser APIs
	wv.EvalJS(postPolyfill)
	// ── Init desktop bridge (Go handlers + JS bridge SDK) ──
	// Must be after LoadHTML (JS runtime ready) and before Vue mount.
	InitDesktopBridge(wv)

	// Diagnostic: check DOM after load
	wv.EvalJS(`(function(){
		var app = document.getElementById('app');
		console.log('[DIAG] getElementById("app")=' + (app ? app.id : 'null'));
		var qs = document.querySelector('#app');
		console.log('[DIAG] querySelector("#app")=' + (qs ? qs.id : 'null'));
		var body = document.body;
		console.log('[DIAG] body=' + (body ? body.tagName : 'null'));
		console.log('[DIAG] body.firstChild=' + (body ? (body.firstChild ? body.firstChild.tagName : 'no-firstChild') : 'no-body'));
	})()`)
	// Vue-injected DOM and component styles.
	for i := 0; i < 8; i++ {
		wv.EnsureLayout()
		time.Sleep(100 * time.Millisecond)
	}
	wv.RebuildRenderTree()

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
				return string(data), nil
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				data, _ := os.ReadFile(filepath.Join(absDist, strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")))
				return string(data), nil
			}
		}
	}
}
