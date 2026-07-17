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

	distDir := "cmd/desktop/web-ui/dist"
	setupLoaders(wv, distDir)
	registerDesktopBridge(wv, reg)

	htmlPath := distDir + "/index.html"
	htmlData, err := os.ReadFile(htmlPath)
	if err != nil {
		log.Fatalf("[Desktop] 请先构建前端: cd cmd/desktop/web-ui && npm run build")
	}
	if err := wv.LoadHTML(string(htmlData)); err != nil {
		log.Fatalf("[Desktop] LoadHTML 失败: %v", err)
	}
	log.Println("[Desktop] 前端页面已加载")

	wv.RebuildRenderTree()

	host, err := app.NewHost(wv, 1280, 800, "PairCode IDE")
	if err != nil {
		log.Fatalf("[Desktop] 创建窗口失败: %v", err)
	}
	log.Println("[Desktop] 窗口已启动，进入事件循环")
	host.Run()
	log.Println("[Desktop] 已退出。")
}

func setupLoaders(wv *webkit.WebView, distDir string) {
	absDist, _ := filepath.Abs(distDir)
	log.Printf("[Desktop] 资源目录: %s", absDist)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				p := resolvePath(src, absDist)
				data, err := os.ReadFile(p)
				if err != nil {
					return "", fmt.Errorf("load script %q: %w", src, err)
				}
				return string(data), nil
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				p := resolvePath(href, absDist)
				data, err := os.ReadFile(p)
				if err != nil {
					return "", fmt.Errorf("load stylesheet %q: %w", href, err)
				}
				return string(data), nil
			}
		}
	}
}

func resolvePath(src, distDir string) string {
	p := strings.TrimPrefix(src, "file://")
	p = strings.TrimPrefix(p, "./")
	return filepath.Join(distDir, p)
}

func registerDesktopBridge(wv *webkit.WebView, reg *bridge.Registry) {
	interp := wv.JSInterpreter()
	interp.SetupGlobal(&jsc.BufferLogger{})
	log.Println("[Desktop] 注册 desktopBridge...")
	bridgeObj := jsc.NewObject(interp.ObjectPrototype())
	bridgeCall := jsc.NewNativeFunction("bridgeCall", func(in *jsc.Interpreter, this jsc.JSValue, args []jsc.JSValue) jsc.JSValue {
		if len(args) < 2 { return errResult("至少需要 method 和 path") }
		m, p := safeStr(args[0]), safeStr(args[1])
		b, q := "", ""
		if len(args) > 2 { b = safeStr(args[2]) }
		if len(args) > 3 { q = safeStr(args[3]) }
		req := fmt.Sprintf(`{"method":"%s","path":"%s","body":%s,"params":%s}`,
			escStr(m), escStr(p), maybeJSON(b), maybeJSON(q))
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
