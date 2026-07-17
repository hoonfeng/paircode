// PairCode IDE 桌面端入口。
//go:build windows && cgo

package main

import (
	"encoding/json"
	"fmt"
	"log"
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

	log.Printf("[Desktop] 正在加载前端页面...")
	if err := wv.LoadHTML(testPage()); err != nil {
		log.Fatalf("[Desktop] LoadHTML 失败: %v", err)
	}
	log.Println("[Desktop] 前端页面已加载")

	host, err := app.NewHost(wv, 1280, 800, "PairCode IDE")
	if err != nil {
		log.Fatalf("[Desktop] 创建窗口失败: %v", err)
	}
	log.Println("[Desktop] 窗口已启动，进入事件循环")
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
		if len(args) > 2 {
			b = safeStr(args[2])
		}
		if len(args) > 3 {
			p = safeStr(args[3])
		}
		req := fmt.Sprintf(`{"method":"%s","path":"%s","body":%s,"params":%s}`,
			escStr(method), escStr(path), maybeJSON(b), maybeJSON(p))
		return jsc.StringValue(reg.HandleBridgeCall(req))
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

func testPage() string {
	return `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>PairCode IDE 桌面端</title></head>
<body style="background:#0d1117;color:#e6edf3;font-family:sans-serif;padding:40px">
<h1>PairCode IDE</h1>
<p id="status">桥接测试中...</p>
<script>
try {
	var result = window.desktopBridge.call("GET", "/api/health", "", "");
	var parsed = JSON.parse(result);
	var body = JSON.parse(parsed.body);
	document.getElementById("status").textContent =
		"桥接正常 ✓  工作区: " + body.workspace + "  状态: " + body.status;
} catch(e) {
	document.getElementById("status").textContent = "错误: " + e.message;
}
</script>
</body>
</html>`
}

func errResult(msg string) jsc.JSValue {
	r, _ := json.Marshal(bridge.BridgeCallResponse{Status: 400, Body: fmt.Sprintf(`{"error":"%s"}`, msg)})
	return jsc.StringValue(string(r))
}

func safeStr(v jsc.JSValue) string {
	if v.IsString() {
		return v.AsString()
	}
	return fmt.Sprint(v)
}

func escStr(s string) string    { return strings.ReplaceAll(s, `"`, `\"`) }
func maybeJSON(s string) string { if s == "" { return `""` }; return s }
