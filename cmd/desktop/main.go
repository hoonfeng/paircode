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
	// Try relative to executable (running from cmd/desktop/)
	distDir := filepath.Join(wd, "web-ui-minimal", "dist")
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		// Try relative to project root
		distDir = filepath.Join(wd, "cmd", "desktop", "web-ui-minimal", "dist")
	}
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		// Try web-ui
		distDir2 := filepath.Join(wd, "web-ui", "dist")
		if _, err2 := os.Stat(distDir2); err2 == nil {
			distDir = distDir2
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

	// Post-load polyfill: MutationObserver
	wv.EvalJS(`if(typeof MutationObserver==='undefined'){
  MutationObserver=function(){this.observe=function(){};this.disconnect=function(){}};
}`)

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
