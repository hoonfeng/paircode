// PairCode IDE 桌面端入口。
// 桌面模式在原生窗口中运行 Vue 前端，通过 desktopBridge 直调 Go 函数，
// 无 HTTP/WebSocket 通信负担。
//
//go:build windows

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/hoonfeng/paircode/internal/bridge"
	"github.com/hoonfeng/paircode/internal/core"
)

var version = "v1.0.3-desktop"

func main() {
	log.Printf("[Desktop] PairCode IDE 桌面版 %s", version)

	// ── 初始化核心配置（与 companion 共享） ──
	core.Load()
	core.LoadLastProject()

	if !core.Loaded {
		log.Println("[Desktop] 未发现已有配置，将使用默认设置。")
	}

	log.Printf("[Desktop] 工作区: %s", core.Root())
	log.Printf("[Desktop] 文件夹: %v", core.Folders)

	// ── 创建 Handler 注册表 ──
	reg := bridge.NewRegistry()

	// ── 注册所有 API Handler ──
	registerAllHandlers(reg)

	// ── 初始化桌面桥接 ──
	// 桌面桥接将 Registry 导出为 JS 全局函数 window.desktopBridge，
	// 使得前端 sdk.js 的 bridgeCall() 直接路由到 Go Handler。
	//
	// 实现方式：使用系统 WebView（WebView2 on Windows）加载前端页面，
	// 在页面加载前注入 desktopBridge 对象。
	bridgeInstance := bridge.NewDesktopBridge(reg)

	// ── 启动桌面窗口 ──
	// 创建原生窗口，加载 web-ui 页面，注入 desktopBridge。
	if err := bridgeInstance.StartWindow(1280, 800, "PairCode IDE"); err != nil {
		log.Fatalf("[Desktop] 窗口启动失败: %v", err)
	}
	defer bridgeInstance.Stop()

	log.Println("[Desktop] 桌面端已启动。")

	// 等待退出信号
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("[Desktop] 正在关闭...")
}

// registerAllHandlers 注册所有 API Handler 到注册表。
// 这些 Handler 与 cmd/companion/web_server.go 完全一致。
// TODO: 后续将 web_server.go 的 Handler 抽取到内部包共用。
func registerAllHandlers(reg *bridge.Registry) {
	// ── 系统 ──
	reg.Register("GET", "/api/health", handleHealth)

	// ── 文件系统 ──
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

	// ── 工作区 ──
	reg.Register("GET", "/api/workspace", handleWorkspace)
	reg.Register("GET", "/api/settings", handleSettings)
	reg.Register("PUT", "/api/settings", handleSettingsPut)
	reg.Register("POST", "/api/workspace", handleWorkspacePost)

	// ── 系统 ──
	reg.Register("GET", "/api/system/info", handleSysInfo)
	reg.Register("POST", "/api/system/exec", handleExec)

	// ── Git ──
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

	// ── 对话 ──
	reg.Register("POST", "/api/chat/send", handleChatSend)
	reg.Register("POST", "/api/chat/stop", handleChatStop)
	reg.Register("POST", "/api/chat/answer", handleChatAnswer)
	reg.Register("POST", "/api/chat/approve", handleChatApprove)
	reg.Register("POST", "/api/chat/feedback", handleChatFeedback)
	reg.Register("POST", "/api/chat/rollback", handleChatRollback)

	// ── 对话列表/消息 ──
	reg.Register("GET", "/api/conversations", handleConversations)
	reg.Register("POST", "/api/conversations", handleConversationCreate)
	reg.Register("GET", "/api/conversations/", handleConversationByID)

	// ── Tasks / Plan ──
	reg.Register("GET", "/api/tasks", handleTasks)
	reg.Register("GET", "/api/taskplan", handleTaskPlan)

	// ── 模型 ──
	reg.Register("GET", "/api/models", handleModels)

	// ── 指令 / 思想 ──
	reg.Register("GET", "/api/instructions", handleInstructions)
	reg.Register("PUT", "/api/instructions", handleInstructionsPut)
	reg.Register("GET", "/api/philosophy", handlePhilosophy)
	reg.Register("PUT", "/api/philosophy", handlePhilosophyPut)

	// ── MCP ──
	reg.Register("GET", "/api/mcp/list", handleMCPList)
	reg.Register("POST", "/api/mcp/save", handleMCPSave)

	// ── Skills ──
	reg.Register("GET", "/api/skills/list", handleSkillsList)
	reg.Register("GET", "/api/skills/read", handleSkillsRead)
	reg.Register("POST", "/api/skills/save", handleSkillsSave)
	reg.Register("POST", "/api/skills/delete", handleSkillsDelete)

	// ── Token / Debug ──
	reg.Register("GET", "/api/tokens/stats", handleTokensStats)
	reg.Register("GET", "/api/debug/logs", handleDebugLogs)
	reg.Register("GET", "/api/debug/logs/", handleDebugLogByID)

	// ── 市场 ──
	reg.Register("GET", "/api/marketplace/search", handleMarketplaceSearch)
	reg.Register("POST", "/api/marketplace/install", handleMarketplaceInstall)
	reg.Register("POST", "/api/marketplace/refresh", handleMarketplaceRefresh)

	// ── 记忆 ──
	reg.Register("GET", "/api/memory/search", handleMemorySearch)
	reg.Register("GET", "/api/memory/list", handleMemoryList)
	reg.Register("POST", "/api/memory/rebuild", handleMemoryRebuild)

	log.Printf("[Desktop] 已注册 %d 个 Handler", len(reg.AllRoutes()))
}

// ─── Handler 占位实现 ───────────────────────────────────────
// 这些是桩（stub），业务逻辑后续从 web_server.go 迁移到内部包共用。
// 先提供最小实现让桌面端可以编译运行。

func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]any{"status": "ok", "workspace": core.Root(), "folders": core.Folders})
}

func handleFSList(w http.ResponseWriter, r *http.Request)   { jsonResp(w, []string{}) }
func handleFSRead(w http.ResponseWriter, r *http.Request)   { jsonResp(w, "") }
func handleFSWrite(w http.ResponseWriter, r *http.Request)  { jsonResp(w, map[string]string{"status": "ok"}) }
func handleFSRename(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]string{"status": "ok"}) }
func handleFSDelete(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]string{"status": "ok"}) }
func handleFSMkdir(w http.ResponseWriter, r *http.Request)  { jsonResp(w, map[string]string{"status": "ok"}) }
func handleFSSearch(w http.ResponseWriter, r *http.Request) { jsonResp(w, []string{}) }
func handleFSDrives(w http.ResponseWriter, r *http.Request) { jsonResp(w, []string{}) }
func handleFSImage(w http.ResponseWriter, r *http.Request)  { jsonResp(w, "") }
func handleFSFileInfo(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]any{}) }
func handleFSHex(w http.ResponseWriter, r *http.Request)    { jsonResp(w, "") }

func handleWorkspace(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]any{}) }
func handleSettings(w http.ResponseWriter, r *http.Request)      { jsonResp(w, map[string]any{}) }
func handleSettingsPut(w http.ResponseWriter, r *http.Request)   { jsonResp(w, map[string]string{"status": "ok"}) }
func handleWorkspacePost(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]string{"status": "ok"}) }

func handleSysInfo(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]any{}) }
func handleExec(w http.ResponseWriter, r *http.Request)    { jsonResp(w, map[string]any{}) }

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

func handleTasks(w http.ResponseWriter, r *http.Request)   { jsonResp(w, []string{}) }
func handleTaskPlan(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]any{}) }

func handleModels(w http.ResponseWriter, r *http.Request) { jsonResp(w, []string{}) }

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

func handleTokensStats(w http.ResponseWriter, r *http.Request)    { jsonResp(w, map[string]any{}) }
func handleDebugLogs(w http.ResponseWriter, r *http.Request)      { jsonResp(w, []string{}) }
func handleDebugLogByID(w http.ResponseWriter, r *http.Request)   { jsonResp(w, "") }

func handleMarketplaceSearch(w http.ResponseWriter, r *http.Request)   { jsonResp(w, []string{}) }
func handleMarketplaceInstall(w http.ResponseWriter, r *http.Request)  { jsonResp(w, map[string]string{"status": "ok"}) }
func handleMarketplaceRefresh(w http.ResponseWriter, r *http.Request)  { jsonResp(w, map[string]string{"status": "ok"}) }

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

// 抑制未使用导入
var _ = strings.TrimSpace
