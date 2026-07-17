// PairCode IDE 桌面端入口。
//go:build windows && cgo

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"wb-ui/app"
	"wb-ui/jsc"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/bridge"
	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/server/handler"
)

var version = "v1.0.3-desktop"

func main() {
	runtime.LockOSThread()
	log.Printf("[Desktop] PairCode IDE 桌面版 %s", version)
	core.Load()
	core.LoadLastProject()
	log.Printf("[Desktop] 工作区: %s", core.Root())

	reg := bridge.NewRegistry()
	handler.RegisterAll(&handler.Router{Reg: reg})
	log.Printf("[Desktop] 已注册 %d 个 Handler", len(reg.AllRoutes()))

	wv := webkit.NewWebView()
	wv.Resize(1280, 800)
	registerDesktopBridge(wv, reg)

	html, err := buildInlinedHTML()
	if err != nil {
		log.Fatalf("[Desktop] 加载前端失败: %v", err)
	}
	if err := wv.LoadHTML(html); err != nil {
		log.Fatalf("[Desktop] LoadHTML 失败: %v", err)
	}
	log.Println("[Desktop] 前端页面已加载")

	host, err := app.NewHost(wv, 1280, 800, "PairCode IDE")
	if err != nil {
		log.Fatalf("[Desktop] 创建窗口失败: %v", err)
	}
	log.Println("[Desktop] 窗口已启动")
	host.Run()
	log.Println("[Desktop] 已退出。")
}

func registerDesktopBridge(wv *webkit.WebView, reg *bridge.Registry) {
	interp := wv.JSInterpreter()
	interp.SetupGlobal(&jsc.BufferLogger{})
	log.Println("[Desktop] 注册 desktopBridge...")

	bridgeObj := jsc.NewObject(interp.ObjectPrototype())
	bridgeCall := jsc.NewNativeFunction("bridgeCall", func(in *jsc.Interpreter, this jsc.JSValue, args []jsc.JSValue) jsc.JSValue {
		if len(args) < 2 {
			return errResult("至少需要 method 和 path")
		}
		method := safeStr(args[0])
		path := safeStr(args[1])
		b, p := "", ""
		if len(args) > 2 { b = safeStr(args[2]) }
		if len(args) > 3 { p = safeStr(args[3]) }
		req := fmt.Sprintf(`{"method":"%s","path":"%s","body":%s,"params":%s}`,
			escStr(method), escStr(path), maybeJSON(b), maybeJSON(p))
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

// buildInlinedHTML 读取 dist/index.html，将所有外部 JS/CSS 内联。
func buildInlinedHTML() (string, error) {
	dist := "cmd/desktop/web-ui/dist"
	idx := filepath.Join(dist, "index.html")
	b, err := os.ReadFile(idx)
	if err != nil {
		return fallbackHTML(), nil
	}
	html := string(b)

	// 内联 CSS
	reCSS := regexp.MustCompile(`<link[^>]+href="\./assets/([^"]+\.css)"[^>]*>`)
	html = reCSS.ReplaceAllStringFunc(html, func(m string) string {
		m2 := reCSS.FindStringSubmatch(m)
		d, e := os.ReadFile(filepath.Join(dist, "assets", m2[1]))
		if e != nil { return m }
		return "<style>\n" + string(d) + "\n</style>"
	})

	// 内联所有 JS 文件并移除模块语法
	reJS := regexp.MustCompile(`<script[^>]+src="\./assets/([^"]+\.js)"[^>]*></script>`)
	jsBundle := ""
	html = reJS.ReplaceAllStringFunc(html, func(m string) string {
		m2 := reJS.FindStringSubmatch(m)
		d, e := os.ReadFile(filepath.Join(dist, "assets", m2[1]))
		if e != nil { return m }
		s := string(d)
		// 移除 export 和顶层 import
		s = regexp.MustCompile(`^import\s+.*?;?\s*$`).ReplaceAllString(s, "")
		s = regexp.MustCompile(`\nexport\s+(default\s+)?`).ReplaceAllString(s, "\n")
		jsBundle += s + "\n"
		return ""
	})

	html = strings.ReplaceAll(html, ` type="module"`, "")
	html = strings.ReplaceAll(html, ` crossorigin`, "")

	if jsBundle != "" {
		html = strings.Replace(html, "</body>", "<script>\n"+jsBundle+"\n</script></body>", 1)
	}
	return html, nil
}

func fallbackHTML() string {
	return `<html><head><script>window.__DESKTOP_MODE__=true</script></head>
<body><h1>PairCode IDE 桌面版</h1><p>构建前端: cd cmd/desktop/web-ui && npm run build</p></body></html>`
}

// ─── 辅助 ──────────────────────────────────────────────

func errResult(msg string) jsc.JSValue {
	r, _ := json.Marshal(bridge.BridgeCallResponse{Status: 400, Body: fmt.Sprintf(`{"error":"%s"}`, msg)})
	return jsc.StringValue(string(r))
}
func safeStr(v jsc.JSValue) string { if v.IsString() { return v.AsString() }; return fmt.Sprint(v) }
func escStr(s string) string        { return strings.ReplaceAll(s, `"`, `\"`) }
func maybeJSON(s string) string     { if s == "" { return `""` }; return s }
