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
var prePolyfill = `if(typeof TextEncoder==='undefined'){TextEncoder=function(){this.encode=function(s){var arr=new Uint8Array(s.length);for(var i=0;i<s.length;i++)arr[i]=s.charCodeAt(i);return arr;}}}
if(typeof TextDecoder==='undefined'){TextDecoder=function(){this.decode=function(arr){var s='';for(var i=0;i<arr.length;i++)s+=String.fromCharCode(arr[i]);return s;}}}
if(typeof WebSocket==='undefined'){WebSocket=function(){console.warn('WebSocket not available in desktop mode')}}
if(typeof crypto==='undefined'){crypto={getRandomValues:function(arr){for(var i=0;i<arr.length;i++)arr[i]=Math.floor(Math.random()*256)}}}
if(typeof structuredClone==='undefined'){structuredClone=function(obj){return JSON.parse(JSON.stringify(obj))}}
if(typeof fetch==='undefined'){fetch=function(){return Promise.reject(new Error('fetch not available in desktop mode'))}}
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
				return string(data), nil
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
