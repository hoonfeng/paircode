// PairCode IDE 桌面端入口。
// 直接使用 wb-ui SDK，无 HTTP/WS 通信。
// 构建条件：CGO_ENABLED=1（依赖 GLFW + Skia）
//
//go:build windows && cgo

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	log.Printf("[Desktop] PairCode IDE 桌面版 %s", version)

	core.Load()
	core.LoadLastProject()
	log.Printf("[Desktop] 工作区: %s", core.Root())

	// 创建 Handler 注册表（通过共享 handler 包注册所有路由）
	reg := bridge.NewRegistry()
	handler.RegisterAll(&handler.Router{Reg: reg})
	log.Printf("[Desktop] 已注册 %d 个 Handler", len(reg.AllRoutes()))

	// 创建 wb-ui WebView
	wv := webkit.NewWebView()
	width, height := 1280, 800
	wv.Resize(width, height)

	// 注册 desktopBridge 到 JS 解释器（必须在 LoadHTML 之前）
	registerDesktopBridge(wv, reg)

	// 加载 Vue 前端页面
	htmlContent, err := getVueAppHTML()
	if err != nil {
		log.Fatalf("[Desktop] 加载前端页面失败: %v", err)
	}
	if err := wv.LoadHTML(htmlContent); err != nil {
		log.Fatalf("[Desktop] LoadHTML 失败: %v", err)
	}
	log.Println("[Desktop] 前端页面已加载")

	// 创建 Host 窗口并启动事件循环
	host, err := app.NewHost(wv, width, height, "PairCode IDE")
	if err != nil {
		log.Fatalf("[Desktop] 创建窗口失败: %v", err)
	}
	log.Println("[Desktop] 窗口已启动，进入事件循环")
	host.Run()
	log.Println("[Desktop] 已退出。")
}

// registerDesktopBridge 在 JS 解释器上注册 desktopBridge 全局对象。
func registerDesktopBridge(wv *webkit.WebView, reg *bridge.Registry) {
	interp := wv.JSInterpreter()
	interp.SetupGlobal(&jsc.BufferLogger{})
	log.Println("[Desktop] 注册 desktopBridge...")

	bridgeObj := jsc.NewObject(interp.ObjectPrototype())

	bridgeCall := jsc.NewNativeFunction("bridgeCall", func(in *jsc.Interpreter, this jsc.JSValue, args []jsc.JSValue) jsc.JSValue {
		if len(args) < 2 {
			return bridgeErrorResult("至少需要 method 和 path 参数")
		}
		method := safeToString(args[0])
		path := safeToString(args[1])
		bodyJSON, paramsJSON := "", ""
		if len(args) > 2 {
			bodyJSON = safeToString(args[2])
		}
		if len(args) > 3 {
			paramsJSON = safeToString(args[3])
		}
		callReq := `{"method":"` + jsString(method) + `","path":"` + jsString(path) +
			`","body":` + maybeJSON(bodyJSON) + `,"params":` + maybeJSON(paramsJSON) + `}`
		result := reg.HandleBridgeCall(callReq)
		return jsc.StringValue(result)
	}, 4)
	bridgeObj.Set("call", jsc.FunctionValue(bridgeCall))
	bridgeObj.Set("onAgentEvent", jsc.FunctionValue(jsc.NewNativeFunction("onAgentEvent",
		func(in *jsc.Interpreter, this jsc.JSValue, args []jsc.JSValue) jsc.JSValue {
			return jsc.Undefined()
		}, 2)))
	bridgeObj.Set("onStatus", jsc.FunctionValue(jsc.NewNativeFunction("onStatus",
		func(in *jsc.Interpreter, this jsc.JSValue, args []jsc.JSValue) jsc.JSValue {
			return jsc.Undefined()
		}, 1)))

	interp.GlobalObject().Set("desktopBridge", jsc.ObjectValue(bridgeObj))
	interp.GlobalObject().Set("__DESKTOP_MODE__", jsc.BooleanValue(true))
	log.Println("[Desktop] desktopBridge 注册完成")
}

func getVueAppHTML() (string, error) {
	htmlPath := "cmd/desktop/web-ui/dist/index.html"
	if data, err := os.ReadFile(htmlPath); err == nil {
		return string(data), nil
	}
	return `<!DOCTYPE html><html><head><meta charset="utf-8"><title>PairCode IDE</title></head><body><div id="app"><h1>PairCode IDE 桌面版</h1><p>正在加载...</p></div><script>console.log("[Desktop] 桌面模式已启动, __DESKTOP_MODE__:",window.__DESKTOP_MODE__);</script></body></html>`, nil
}

func bridgeErrorResult(msg string) jsc.JSValue {
	resp, _ := json.Marshal(bridge.BridgeCallResponse{
		Status: 400,
		Body:   fmt.Sprintf(`{"error":"%s"}`, msg),
	})
	return jsc.StringValue(string(resp))
}

func safeToString(v jsc.JSValue) string {
	if v.IsString() {
		return v.AsString()
	}
	return fmt.Sprint(v)
}

func jsString(s string) string  { return strings.ReplaceAll(s, `"`, `\"`) }
func maybeJSON(s string) string { if s == "" { return `""` }; return s }
func jsonQuote(s string) string { b, _ := json.Marshal(s); return string(b) }

var _ = jsonQuote
