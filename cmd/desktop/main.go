package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/app"
	"wb-ui/dom"
	"wb-ui/webkit"
)

func main() {
	log.SetFlags(log.Ltime)
	log.Println("[Desktop] PairCode IDE 桌面版 v1.0.6-desktop")

	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "desktop", "web-ui-minimal", "dist")
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		distDir = filepath.Join(wd, "web-ui-minimal", "dist")
	}
	log.Printf("[Desktop] distDir: %s", distDir)

	wv := webkit.NewWebView()

	setupLoaders(wv, distDir)

	// 确保 JS 运行时在 LoadHTML 前初始化
	_ = wv.JSInterpreter()

	// 基础 polyfill
	wv.EvalJS(`if(!Object.getPrototypeOf)Object.getPrototypeOf=function(o){return o&&o.constructor?o.constructor.prototype:null}`)
	wv.EvalJS(`if(!Object.setPrototypeOf)Object.setPrototypeOf=function(o,p){o.__proto__=p;return o}`)

	// 加载 HTML
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

	// 输出控制台消息
	if out := wv.ConsoleOutput(); out != "" {
		log.Printf("[CONSOLE]\n%s", out)
	}

	// 检查 Go 层 DOM 树
	if doc := wv.Document(); doc != nil {
		log.Printf("[DOM] document=%v", doc)
		if app := doc.GetElementById("app"); app != nil {
			log.Printf("[DOM] #app found! children=%d", len(app.ChildNodes()))
			for i, c := range app.ChildNodes() {
				if el, ok := c.(*dom.Element); ok {
					log.Printf("[DOM] #app child[%d]: tag=%s class=%s", i, el.LocalName(), el.GetAttribute("class"))
				}
			}
			if app.HasAttribute("__vue_app__") {
				log.Printf("[DOM] #app __vue_app__=YES")
			}
		} else {
			log.Printf("[DOM] #app NOT FOUND in document")
			if body := doc.GetElementsByTagName("body"); len(body) > 0 {
				log.Printf("[DOM] body found, children=%d", len(body[0].ChildNodes()))
				for i, c := range body[0].ChildNodes() {
					if el, ok := c.(*dom.Element); ok {
						log.Printf("[DOM] body child[%d]: %s", i, el.LocalName())
					} else {
						log.Printf("[DOM] body child[%d]: %T", i, c)
					}
				}
			}
		}
	} else {
		log.Printf("[DOM] document is nil")
	}

	// JS 检查
	v, _ := wv.EvalJS(`document.getElementById('app') ? 'ok' : 'no-app'`)
	log.Printf("[Eval] getElementById('app')=%s", v.String())

	// 重建渲染树
	wv.RebuildRenderTree()

	// 启动窗口
	host, _ := app.NewHost(wv, 1280, 800, "PairCode IDE")
	log.Println("[Desktop] 窗口已启动, 等待3秒后截图...")
	time.Sleep(3 * time.Second)

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
