package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"wb-ui/app"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(log.Ltime)
	log.Println("[Desktop] PairCode IDE 桌面版 v1.0.6-desktop")

	wd, _ := os.Getwd()
	// Try web-ui/dist (full IDE) first
	distDir := filepath.Join(wd, "web-ui", "dist")
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		// Try relative to project root (running from cmd/desktop/)
		distDir = filepath.Join(wd, "cmd", "desktop", "web-ui", "dist")
	}
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		// Fallback: web-ui-minimal (demo)
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
	// Browser polyfills (before DOM bindings)
	wv.EvalJS(`if(!Object.getPrototypeOf)Object.getPrototypeOf=function(o){return o&&o.constructor?o.constructor.prototype:null}`)
	wv.EvalJS(`if(!Object.setPrototypeOf)Object.setPrototypeOf=function(o,p){o.__proto__=p;return o}`)
	// Define browser global constructors before LoadHTML so Vue doesn't throw ReferenceError.
	// These will be overwritten after LoadHTML by DOM bindings' proper versions.
	wv.EvalJS(`EventTarget=function EventTarget(){};
Node=function Node(){};Node.prototype=Object.create(EventTarget.prototype);Node.prototype.constructor=Node;
Element=function Element(){};Element.prototype=Object.create(Node.prototype);Element.prototype.constructor=Element;
SVGElement=function SVGElement(){};SVGElement.prototype=Object.create(Element.prototype);SVGElement.prototype.constructor=SVGElement;
HTMLElement=function HTMLElement(){};HTMLElement.prototype=Object.create(Element.prototype);HTMLElement.prototype.constructor=HTMLElement;
HTMLUnknownElement=function HTMLUnknownElement(){};HTMLUnknownElement.prototype=Object.create(HTMLElement.prototype);HTMLUnknownElement.prototype.constructor=HTMLUnknownElement;
Text=function Text(){};Text.prototype=Object.create(Node.prototype);Text.prototype.constructor=Text;
Comment=function Comment(){};Comment.prototype=Object.create(Node.prototype);Comment.prototype.constructor=Comment;
DocumentFragment=function DocumentFragment(){};DocumentFragment.prototype=Object.create(Node.prototype);DocumentFragment.prototype.constructor=DocumentFragment;
Attr=function Attr(){};Attr.prototype=Object.create(Node.prototype);Attr.prototype.constructor=Attr;`)

	htmlData, _ := os.ReadFile(distDir + "/index.html")
	s := string(htmlData)
	s = strings.Replace(s, `type="module"`, "", 1)
	s = strings.ReplaceAll(s, `crossorigin`, "")
	log.Printf("[LoadHTML] 开始加载, 大小=%d", len(s))
	err := wv.LoadHTML(s)
	if err != nil {
		log.Printf("[LoadHTML] 错误: %v", err)
	} else {
		log.Printf("[LoadHTML] 加载成功")
	}
	if out := wv.ConsoleOutput(); out != "" {
		log.Printf("[CONSOLE]\n%s", out)
	}

	// Post-load: full browser polyfills for IDE
	wv.EvalJS(`if(typeof MutationObserver==='undefined'){
  MutationObserver=function(){this.observe=function(){};this.disconnect=function(){}};
}`)
	wv.EvalJS(`if(typeof navigator==='undefined')navigator={onLine:true,userAgent:'Goja/1.0',language:'zh-CN',platform:'Win32'};
if(!navigator.onLine)navigator.onLine=true;
if(typeof localStorage==='undefined'){
  (function(){
    var store={};
    localStorage={
      getItem:function(k){return store.hasOwnProperty(k)?store[k]:null},
      setItem:function(k,v){store[k]=String(v)},
      removeItem:function(k){delete store[k]},
      clear:function(){store={}},
      key:function(i){var ks=Object.keys(store);return ks[i]||null},
      get length(){return Object.keys(store).length}
    };
  })();
}
if(typeof location==='undefined')location={href:'about:blank',origin:'file://',protocol:'file:',host:'',pathname:'/index.html',search:'',hash:''};
if(typeof history==='undefined')history={pushState:function(){},replaceState:function(){},back:function(){},forward:function(){},go:function(){},length:1,state:null};
if(typeof requestAnimationFrame==='undefined')requestAnimationFrame=function(cb){return setTimeout(cb,16)};
if(typeof cancelAnimationFrame==='undefined')cancelAnimationFrame=function(id){clearTimeout(id)};
if(typeof customElements==='undefined')customElements={define:function(){},get:function(){return null},whenDefined:function(){return Promise.resolve()}};
if(typeof matchMedia==='undefined')matchMedia=function(){return{matches:false,media:'',addListener:function(){},removeListener:function(){},addEventListener:function(){},removeEventListener:function(){}}};
`)

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
