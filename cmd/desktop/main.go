// PairCode IDE 桌面端入口。
// 直接使用 wb-ui SDK 提供桌面窗口和 Go↔JS 桥接。
//
// wb-ui 是完整的桌面 UI SDK，提供：
//   - app.Host：窗口管理 + 事件循环（GLFW + Skia GPU 渲染）
//   - webkit.WebView：HTML/CSS 渲染引擎 + JS 解释器
//   - jsc：纯 Go JavaScriptCore 引擎（支持 Proxy/Promise/ES Module）
//   - bindings：Go↔JS 桥接（注册 Go 函数为 JS 全局对象）
//
// 构建条件：CGO_ENABLED=1（依赖 GLFW + Skia）
//
//go:build windows && cgo

package main

import (
	"log"
	"os"
	"strings"
	"time"

	"wb-ui/app"
	"time"

	"wb-ui/app"
	"wb-ui/jsc"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/bridge"
	"github.com/hoonfeng/paircode/internal/core"
)

var version = "v1.0.3-desktop"

func main() {
	log.Printf("[Desktop] PairCode IDE 桌面版 %s", version)

	// ── 初始化核心配置 ──
	core.Load()
	core.LoadLastProject()

	if !core.Loaded {
		log.Println("[Desktop] 未发现已有配置，将使用默认设置。")
	}

	log.Printf("[Desktop] 工作区: %s", core.Root())

	// ── 创建 Handler 注册表 ──
	reg := bridge.NewRegistry()
	registerAllHandlers(reg)
	log.Printf("[Desktop] 已注册 %d 个 Handler", len(reg.AllRoutes()))

	// ── 创建 wb-ui WebView ──
	wv := webkit.NewWebView()
	width, height := 1280, 800
	wv.Resize(width, height)

	// ── 注册 desktopBridge 到 JS 解释器 ──
	// ★ 必须在 LoadHTML 之前注册，确保前端 JS 执行时 bridge 已就绪
	registerDesktopBridge(wv, reg)

	// ── 加载 Vue 前端页面 ──
	// 读取内嵌的 Vue 构建产物 HTML
	htmlContent, err := getVueAppHTML()
	if err != nil {
		log.Fatalf("[Desktop] 加载前端页面失败: %v", err)
	}
	if err := wv.LoadHTML(htmlContent); err != nil {
		log.Fatalf("[Desktop] LoadHTML 失败: %v", err)
	}
	log.Println("[Desktop] 前端页面已加载")

	// ── 创建 Host 窗口 ──
	host, err := app.NewHost(wv, width, height, "PairCode IDE")
	if err != nil {
		log.Fatalf("[Desktop] 创建窗口失败: %v", err)
	}

	// 信号处理（优雅退出）
	// ── 启动事件循环（阻塞直到窗口关闭） ──
	// 用户点击窗口关闭按钮后 Run() 返回，程序退出。
	log.Println("[Desktop] 窗口已启动，进入事件循环")
	host.Run()
	log.Println("[Desktop] 已退出。")
}
}

// registerDesktopBridge 在 wb-ui 的 JS 解释器上注册 desktopBridge 全局对象。
// 前端 sdk.js 通过 window.desktopBridge.call(method, path, body, params)
// 直接调用 Go 端的 Handler，无需 HTTP/WebSocket。
func registerDesktopBridge(wv *webkit.WebView, reg *bridge.Registry) {
	interp := wv.JSInterpreter()
	interp.SetupGlobal(&jsc.BufferLogger{})

	log.Println("[Desktop] 注册 desktopBridge 到 JS 解释器...")

	// ── 创建 desktopBridge 对象 ──
	bridgeObj := jsc.NewObject(interp.ObjectPrototype())

	// bridgeCall — 前端 API 调用的统一入口
	// 签名：call(method, path, bodyJSON?, paramsJSON?) → BridgeCallResponse JSON
	bridgeCall := jsc.NewNativeFunction("bridgeCall", func(in *jsc.Interpreter, this jsc.JSValue, args []jsc.JSValue) jsc.JSValue {
		if len(args) < 2 {
			return bridgeErrorResult("至少需要 method 和 path 参数")
		}

		method := safeToString(args[0])
		path := safeToString(args[1])
		bodyJSON := ""
		paramsJSON := ""
		if len(args) > 2 {
			bodyJSON = safeToString(args[2])
		}
		if len(args) > 3 {
			paramsJSON = safeToString(args[3])
		}

		// 构造 BridgeCallRequest
		callReq := `{"method":"` + jsString(method) + `","path":"` + jsString(path) +
			`","body":` + maybeJSON(bodyJSON) + `,"params":` + maybeJSON(paramsJSON) + `}`

		result := reg.HandleBridgeCall(callReq)
		return jsc.StringValue(result)
	})
	bridgeObj.Set("call", jsc.FunctionValue(bridgeCall))

	// onAgentEvent — Go 端通过它推送 agent 事件（占位，由 Go 端 EvalJS 调用）
	bridgeObj.Set("onAgentEvent", jsc.FunctionValue(jsc.NewNativeFunction("onAgentEvent",
		func(in *jsc.Interpreter, this jsc.JSValue, args []jsc.JSValue) jsc.JSValue {
			return jsc.Undefined()
		}, 2)))

	// onStatus — Go 端通过它推送运行状态
	bridgeObj.Set("onStatus", jsc.FunctionValue(jsc.NewNativeFunction("onStatus",
		func(in *jsc.Interpreter, this jsc.JSValue, args []jsc.JSValue) jsc.JSValue {
			return jsc.Undefined()
		}, 1)))

	// 注册为全局对象
	interp.GlobalObject().Set("desktopBridge", jsc.ObjectValue(bridgeObj))
	interp.GlobalObject().Set("__DESKTOP_MODE__", jsc.BooleanValue(true))

	log.Println("[Desktop] desktopBridge 注册完成")
}

// ─── 事件推送（Go → JS） ──────────────────────────────────

// PushAgentEvent 向 JS 前端推送 agent 事件。
func PushAgentEvent(wv *webkit.WebView, convId string, data any) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("[Desktop] PushAgentEvent marshal 失败: %v", err)
		return
	}

	js := fmt.Sprintf(`if(window.desktopBridge&&window.desktopBridge.onAgentEvent){window.desktopBridge.onAgentEvent(%s,%s)}`,
		jsonQuote(convId), string(dataJSON))

	if _, err := wv.EvalJS(js); err != nil {
		log.Printf("[Desktop] PushAgentEvent EvalJS 失败: %v", err)
	}
}

// PushStatus 向 JS 前端推送运行状态。
func PushStatus(wv *webkit.WebView, payload any) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Desktop] PushStatus marshal 失败: %v", err)
		return
	}

	js := `if(window.desktopBridge&&window.desktopBridge.onStatus){window.desktopBridge.onStatus(` +
		string(payloadJSON) + `)}`

	if _, err := wv.EvalJS(js); err != nil {
		log.Printf("[Desktop] PushStatus EvalJS 失败: %v", err)
	}
}

// ─── 前端页面加载 ──────────────────────────────────────────

// getVueAppHTML 获取 Vue 前端页面 HTML 内容。
// TODO: 读取 web-ui/dist/index.html (Vite 构建产物)
// 当前返回简单测试页，待集成 Vue build 产物后替换。
func getVueAppHTML() (string, error) {
	// 尝试读取构建产物
	htmlPath := "cmd/desktop/web-ui/dist/index.html"
	if data, err := os.ReadFile(htmlPath); err == nil {
		return string(data), nil
	}

	// 退回到简单测试页
	return `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>PairCode IDE</title></head>
<body>
<div id="app">
	<h1>PairCode IDE 桌面版</h1>
	<p>正在加载...</p>
</div>
<script>
(function(){
	console.log("[Desktop] 桌面模式已启动");
	console.log("[Desktop] __DESKTOP_MODE__:", window.__DESKTOP_MODE__);
	console.log("[Desktop] desktopBridge:", typeof window.desktopBridge);

	// 测试 bridge 调用
	if(window.desktopBridge && window.desktopBridge.call) {
		var result = window.desktopBridge.call("GET", "/api/health", "", "");
		console.log("[Desktop] /api/health 响应:", result);
		var data = JSON.parse(result);
		document.getElementById("app").innerHTML = "<h1>PairCode IDE</h1><p>状态: " +
			(data.body ? JSON.parse(data.body).status : "unknown") + "</p>" +
			"<p>工作区: " + (data.body ? JSON.parse(data.body).workspace : "") + "</p>";
	}
})();
</script>
</body>
</html>`, nil
}

// ─── Handler 注册 ──────────────────────────────────────────

func registerAllHandlers(reg *bridge.Registry) {
	// 这些 Handler 与 cmd/companion/web_server.go 完全一致。
	// TODO: 后续将 Handler 实现提取到共享包。
	reg.Register("GET", "/api/health", handleHealth)
	reg.Register("GET", "/api/fs/list", handleFSList)
	reg.Register("GET", "/api/fs/read", handleFSRead)
	reg.Register("POST", "/api/fs/write", handleFSWrite)
	reg.Register("POST", "/api/fs/rename", handleFSRename)
	reg.Register("DELETE", "/api/fs/delete", handleFSDelete)
	reg.Register("POST", "/api/fs/mkdir", handleFSMkdir)
	reg.Register("GET", "/api/fs/search", handleFSSearch)
	reg.Register("GET", "/api/fs/drives", handleFSDrives)
	reg.Register("GET", "/api/fs/image", handleFSImage)
	reg.Register("GET", "/api/fs/file-info", handleFSFileInfo)
	reg.Register("GET", "/api/fs/hex", handleFSHex)
	reg.Register("GET", "/api/workspace", handleWorkspace)
	reg.Register("GET", "/api/settings", handleSettings)
	reg.Register("PUT", "/api/settings", handleSettingsPut)
	reg.Register("POST", "/api/workspace", handleWorkspacePost)
	reg.Register("GET", "/api/system/info", handleSysInfo)
	reg.Register("POST", "/api/system/exec", handleExec)
	reg.Register("GET", "/api/git/status", handleGitStatus)
	reg.Register("GET", "/api/git/diff", handleGitDiff)
	reg.Register("POST", "/api/git/add", handleGitAdd)
	reg.Register("POST", "/api/git/commit", handleGitCommit)
	reg.Register("GET", "/api/git/log", handleGitLog)
	reg.Register("POST", "/api/git/branch", handleGitBranch)
	reg.Register("POST", "/api/git/checkout", handleGitCheckout)
	reg.Register("POST", "/api/git/stash", handleGitStash)
	reg.Register("GET", "/api/git/stash-list", handleGitStashList)
	reg.Register("GET", "/api/git/ignore", handleGitIgnore)
	reg.Register("POST", "/api/git/discard", handleGitDiscard)
	reg.Register("POST", "/api/git/push", handleGitPush)
	reg.Register("POST", "/api/git/pull", handleGitPull)
	reg.Register("GET", "/api/git/remote", handleGitRemote)
	reg.Register("GET", "/api/git/init", handleGitInit)
	reg.Register("POST", "/api/git/reset", handleGitReset)
	reg.Register("POST", "/api/chat/send", handleChatSend)
	reg.Register("POST", "/api/chat/stop", handleChatStop)
	reg.Register("POST", "/api/chat/answer", handleChatAnswer)
	reg.Register("POST", "/api/chat/approve", handleChatApprove)
	reg.Register("POST", "/api/chat/feedback", handleChatFeedback)
	reg.Register("POST", "/api/chat/rollback", handleChatRollback)
	reg.Register("GET", "/api/conversations", handleConversations)
	reg.Register("POST", "/api/conversations", handleConversationCreate)
	reg.Register("GET", "/api/conversations/", handleConversationByID)
	reg.Register("GET", "/api/tasks", handleTasks)
	reg.Register("GET", "/api/taskplan", handleTaskPlan)
	reg.Register("GET", "/api/models", handleModels)
	reg.Register("GET", "/api/instructions", handleInstructions)
	reg.Register("PUT", "/api/instructions", handleInstructionsPut)
	reg.Register("GET", "/api/philosophy", handlePhilosophy)
	reg.Register("PUT", "/api/philosophy", handlePhilosophyPut)
	reg.Register("GET", "/api/mcp/list", handleMCPList)
	reg.Register("POST", "/api/mcp/save", handleMCPSave)
	reg.Register("GET", "/api/skills/list", handleSkillsList)
	reg.Register("GET", "/api/skills/read", handleSkillsRead)
	reg.Register("POST", "/api/skills/save", handleSkillsSave)
	reg.Register("POST", "/api/skills/delete", handleSkillsDelete)
	reg.Register("GET", "/api/tokens/stats", handleTokensStats)
	reg.Register("GET", "/api/debug/logs", handleDebugLogs)
	reg.Register("GET", "/api/debug/logs/", handleDebugLogByID)
	reg.Register("GET", "/api/marketplace/search", handleMarketplaceSearch)
	reg.Register("POST", "/api/marketplace/install", handleMarketplaceInstall)
	reg.Register("POST", "/api/marketplace/refresh", handleMarketplaceRefresh)
	reg.Register("GET", "/api/memory/search", handleMemorySearch)
	reg.Register("GET", "/api/memory/list", handleMemoryList)
	reg.Register("POST", "/api/memory/rebuild", handleMemoryRebuild)
}

// ─── Handler 桩实现（占位，后续从 web_server.go 提取共享实现） ──

func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]any{"status": "ok", "workspace": core.Root(), "folders": core.Folders})
}
func handleFSList(w http.ResponseWriter, r *http.Request)     { jsonResp(w, []string{}) }
func handleFSRead(w http.ResponseWriter, r *http.Request)     { jsonResp(w, "") }
func handleFSWrite(w http.ResponseWriter, r *http.Request)    { jsonResp(w, map[string]string{"status": "ok"}) }
func handleFSRename(w http.ResponseWriter, r *http.Request)   { jsonResp(w, map[string]string{"status": "ok"}) }
func handleFSDelete(w http.ResponseWriter, r *http.Request)   { jsonResp(w, map[string]string{"status": "ok"}) }
func handleFSMkdir(w http.ResponseWriter, r *http.Request)    { jsonResp(w, map[string]string{"status": "ok"}) }
func handleFSSearch(w http.ResponseWriter, r *http.Request)   { jsonResp(w, []string{}) }
func handleFSDrives(w http.ResponseWriter, r *http.Request)   { jsonResp(w, []string{}) }
func handleFSImage(w http.ResponseWriter, r *http.Request)    { jsonResp(w, "") }
func handleFSFileInfo(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]any{}) }
func handleFSHex(w http.ResponseWriter, r *http.Request)      { jsonResp(w, "") }
func handleWorkspace(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]any{}) }
func handleSettings(w http.ResponseWriter, r *http.Request)      { jsonResp(w, map[string]any{}) }
func handleSettingsPut(w http.ResponseWriter, r *http.Request)   { jsonResp(w, map[string]string{"status": "ok"}) }
func handleWorkspacePost(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]string{"status": "ok"}) }
func handleSysInfo(w http.ResponseWriter, r *http.Request)  { jsonResp(w, map[string]any{}) }
func handleExec(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]any{}) }
func handleGitStatus(w http.ResponseWriter, r *http.Request)     { jsonResp(w, "") }
func handleGitDiff(w http.ResponseWriter, r *http.Request)       { jsonResp(w, "") }
func handleGitAdd(w http.ResponseWriter, r *http.Request)        { jsonResp(w, map[string]string{"status": "ok"}) }
func handleGitCommit(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]string{"status": "ok"}) }
func handleGitLog(w http.ResponseWriter, r *http.Request)        { jsonResp(w, []string{}) }
func handleGitBranch(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]any{}) }
func handleGitCheckout(w http.ResponseWriter, r *http.Request)   { jsonResp(w, map[string]string{"status": "ok"}) }
func handleGitStash(w http.ResponseWriter, r *http.Request)      { jsonResp(w, map[string]string{"status": "ok"}) }
func handleGitStashList(w http.ResponseWriter, r *http.Request)  { jsonResp(w, []string{}) }
func handleGitIgnore(w http.ResponseWriter, r *http.Request)     { jsonResp(w, "") }
func handleGitDiscard(w http.ResponseWriter, r *http.Request)    { jsonResp(w, map[string]string{"status": "ok"}) }
func handleGitPush(w http.ResponseWriter, r *http.Request)       { jsonResp(w, map[string]string{"status": "ok"}) }
func handleGitPull(w http.ResponseWriter, r *http.Request)       { jsonResp(w, map[string]string{"status": "ok"}) }
func handleGitRemote(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]any{}) }
func handleGitInit(w http.ResponseWriter, r *http.Request)       { jsonResp(w, map[string]string{"status": "ok"}) }
func handleGitReset(w http.ResponseWriter, r *http.Request)      { jsonResp(w, map[string]string{"status": "ok"}) }
func handleChatSend(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]string{"status": "ok"}) }
func handleChatStop(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]string{"status": "ok"}) }
func handleChatAnswer(w http.ResponseWriter, r *http.Request)   { jsonResp(w, map[string]string{"status": "ok"}) }
func handleChatApprove(w http.ResponseWriter, r *http.Request)  { jsonResp(w, map[string]string{"status": "ok"}) }
func handleChatFeedback(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]string{"status": "ok"}) }
func handleChatRollback(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]string{"status": "ok"}) }
func handleConversations(w http.ResponseWriter, r *http.Request)      { jsonResp(w, []string{}) }
func handleConversationCreate(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]any{}) }
func handleConversationByID(w http.ResponseWriter, r *http.Request)   { jsonResp(w, map[string]any{}) }
func handleTasks(w http.ResponseWriter, r *http.Request)    { jsonResp(w, []string{}) }
func handleTaskPlan(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]any{}) }
func handleModels(w http.ResponseWriter, r *http.Request)   { jsonResp(w, []string{}) }
func handleInstructions(w http.ResponseWriter, r *http.Request)     { jsonResp(w, "") }
func handleInstructionsPut(w http.ResponseWriter, r *http.Request)  { jsonResp(w, map[string]string{"status": "ok"}) }
func handlePhilosophy(w http.ResponseWriter, r *http.Request)       { jsonResp(w, "") }
func handlePhilosophyPut(w http.ResponseWriter, r *http.Request)    { jsonResp(w, map[string]string{"status": "ok"}) }
func handleMCPList(w http.ResponseWriter, r *http.Request)  { jsonResp(w, []string{}) }
func handleMCPSave(w http.ResponseWriter, r *http.Request)  { jsonResp(w, map[string]string{"status": "ok"}) }
func handleSkillsList(w http.ResponseWriter, r *http.Request)   { jsonResp(w, []string{}) }
func handleSkillsRead(w http.ResponseWriter, r *http.Request)   { jsonResp(w, "") }
func handleSkillsSave(w http.ResponseWriter, r *http.Request)   { jsonResp(w, map[string]string{"status": "ok"}) }
func handleSkillsDelete(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]string{"status": "ok"}) }
func handleTokensStats(w http.ResponseWriter, r *http.Request)  { jsonResp(w, map[string]any{}) }
func handleDebugLogs(w http.ResponseWriter, r *http.Request)    { jsonResp(w, []string{}) }
func handleDebugLogByID(w http.ResponseWriter, r *http.Request) { jsonResp(w, "") }
func handleMarketplaceSearch(w http.ResponseWriter, r *http.Request)  { jsonResp(w, []string{}) }
func handleMarketplaceInstall(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]string{"status": "ok"}) }
func handleMarketplaceRefresh(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]string{"status": "ok"}) }
func handleMemorySearch(w http.ResponseWriter, r *http.Request)  { jsonResp(w, []string{}) }
func handleMemoryList(w http.ResponseWriter, r *http.Request)    { jsonResp(w, []string{}) }
func handleMemoryRebuild(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]string{"status": "ok"}) }

// ─── 辅助函数 ──────────────────────────────────────────────

func jsonResp(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
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

func jsString(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

func maybeJSON(s string) string {
	if s == "" {
		return `""`
	}
	return s
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// 确保时间包被使用
var _ = time.Now
