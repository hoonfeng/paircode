// HTTP 服务器核心（无 GWui 依赖，webonly 和桌面模式共用）。
//
//go:build windows

package main

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/internal/agent"
	"github.com/hoonfeng/paircode/internal/agenttools"
	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/server/handler"
	marketplacepanel "github.com/hoonfeng/paircode/internal/ui/marketplace"
	mcppanel "github.com/hoonfeng/paircode/internal/ui/mcp"
	"github.com/hoonfeng/paircode/internal/ui/skills"
	"github.com/hoonfeng/paircode/internal/roleprompts"
	"github.com/hoonfeng/paircode/pkg/memory"
	"github.com/hoonfeng/paircode/pkg/summary"
)

// ─── 内嵌前端资源 ─────────────────────────────────────────────

//go:embed web-ui/dist
var webUIFiles embed.FS

//go:embed web-ui/ide_ref.html
var ideRefFile embed.FS

// webServer 是运行在 companion 内部的 HTTP 服务器。
type webServer struct {
	server    *http.Server
	port      int
	mu        sync.Mutex
	eventRing *eventRing // 全局事件环形缓冲（断连回放用）
}

// agentMgr 全局会话管理器：管理并行 agent 会话（Start/Stop/Subscribe 等）。
// web 层作为套壳，所有会话生命周期通过 agentMgr 管理。
var agentMgr = agent.NewSessionManager()

var ws *webServer

// ── 平台回调注册（由 webui_*.go 在启动时注册） ──
// 统一 API 层后，平台差异仅通过以下回调参数化：
//   buildProviderFn:   创建 LLM Provider（主模型）
//   buildSystemPromptFn:创建系统提示语
//   buildCompressorFn:  创建上下文压缩器（nil=规则式压缩）
//   buildPlanProviderFn:创建规划 Provider（自主模式，回退到 buildProviderFn）
type (
	buildProviderFn     func() agent.Provider
	buildSystemPromptFn func() string
	buildCompressorFn   func() agent.Compressor
	buildPlanProviderFn func() agent.Provider
)

var (
	webProvider         buildProviderFn
	webSystemPrompt     buildSystemPromptFn
	webCompressor       buildCompressorFn
	webPlanProvider     buildPlanProviderFn
)

// findMessageStoreRoot 在所有工作区文件夹中查找第一个有对话数据目录的路径。
// 解决 bug：当 WorkspaceFolders 排序变化后，core.Root() 指向了没有对话数据的文件夹，
// 导致 MessageStore 读不到已有的 index.json，对话列表为空。
func findMessageStoreRoot() string {
	primary := core.Root()
	if primary == "" {
		return ""
	}
	// 遍历 core.Folders，找第一个已有对话数据的目录
	for _, f := range core.Folders {
		idxPath := filepath.Join(f, ".pair", "conversations", "index.json")
		if info, err := os.Stat(idxPath); err == nil && !info.IsDir() {
			return f
		}
		// 也检查旧格式 conversations.json（尚未迁移的）
		legacyPath := filepath.Join(f, ".pair", "conversations.json")
		if info, err := os.Stat(legacyPath); err == nil && !info.IsDir() {
			return f
		}
	}
	return primary
}

// startWebUI 在后台启动 Web UI 服务器。
func startWebUI(port int) {
	if ws != nil {
		return
	}
	ws = &webServer{
		port:      port,
		eventRing: newEventRing(1000), // 缓存最近 1000 个全局事件用于断连回放
	}
	// 初始化 MessageStore（消息持久化的唯一权威），并迁移旧格式数据
	// ★ 先找已有对话数据的工作区目录（排序变化后 core.Root() 可能指向了没有对话数据的目录）
	root := findMessageStoreRoot()
	// 初始化市场系统（尝试从本地缓存加载，异步获取远程注册表）
	marketplacepanel.Init(root)
	if root != "" {
		agentMgr.SetWorkspaceRoot(root)
		agent.SetCodeGraphDB(agentMgr.RawDB())
		// 迁移旧 conversations.json + history_cache.json 到新格式
		convPath := filepath.Join(root, ".pair", "conversations.json")
		hcPath := filepath.Join(root, ".pair", "history_cache.json")
		if store := agentMgr.Store(); store != nil {
			if err := store.MigrateFromLegacy(convPath, hcPath); err != nil {
				fmt.Printf("[WebUI] 迁移旧对话数据失败（不阻塞启动）: %v\n", err)
			}
		}
		// ★ 启动时异步构建代码知识图谱（不阻塞 HTTP 服务器启动）
		go func() {
			if _, err := agent.EnsureCodeGraph(root); err != nil {
				log.Printf("[WebUI] 启动时构建代码图谱失败（不阻塞启动）: %v", err)
			} else {
				log.Printf("[WebUI] 代码知识图谱已就绪")
			}
		}()
	}
	if root := core.Root(); root != "" {
		memory.SetRoot(root)
		agent.InitTracker(root)
	}
	// 初始化 Skills 资源目录（供 LoadAllSkills 使用）
	if root := core.Root(); root != "" {
		agent.SkillProjectDir = filepath.Join(root, ".pair", "skills")
	}
	if sysDir := filepath.Join(core.ConfigDir(), "skills"); sysDir != "" {
		agent.SkillSystemDir = sysDir
	}
	// 初始化 MCP 配置路径（供 agent/mcp_config.go 读写 mcp.json）
	agent.MCPUserConfigPath = filepath.Join(core.ConfigDir(), "mcp.json")
	if root := core.Root(); root != "" {
		agent.MCPProjectConfigPath = filepath.Join(root, ".pair", "mcp.json")
	}

	// ★ 启动时初始化参考注册表，供 /api/tools 查询工具列表（无需先启动对话）
	if root := core.Root(); root != "" {
		initReg := agent.NewRegistry()
		agent.RegisterDefaultTools(initReg, root)
		agent.RegisterCommitMessageTool(initReg)
		agenttools.RegisterManagementTools(initReg, root)
		// 自动初始化 .pair/tools.json（不存在则创建）
		agent.LoadWorkspaceToolConfig(initReg, root)
		handler.SetToolsRegistry(initReg)
		log.Printf("[WebUI] 参考工具注册表已初始化（%d 个工具）", len(initReg.AllToolMeta()))
	}

	// 工作区文件夹变更时同步到 agent 路径解析
	core.OnSyncWorkspace = func(primaryChanged bool) {
		agent.WorkspaceRoots = core.Folders
		if primaryChanged {
			// 主工作区变更：同步数据库/技能/MCP/记忆路径到新工作区
			root := core.Root()
			if root != "" {
				agentMgr.SetWorkspaceRoot(root)
				agent.SetCodeGraphDB(agentMgr.RawDB())
				agent.SkillProjectDir = filepath.Join(root, ".pair", "skills")
				agent.MCPProjectConfigPath = filepath.Join(root, ".pair", "mcp.json")
				memory.SetRoot(root)
			}
		}
		
	}
	mux := http.NewServeMux()

	// ── 基础 API 路由（非 chat） ──
	mux.HandleFunc("/api/health", ws.handleHealth)
	mux.HandleFunc("/api/fs/list", ws.handleFSList)
	mux.HandleFunc("/api/fs/read", ws.handleFSRead)
	mux.HandleFunc("/api/fs/write", ws.handleFSWrite)
	mux.HandleFunc("/api/fs/rename", ws.handleFSRename)
	mux.HandleFunc("/api/fs/drives", ws.handleFSDrives)
	mux.HandleFunc("/api/workspace", ws.handleWorkspace)
	mux.HandleFunc("/api/settings", ws.handleSettings)
	mux.HandleFunc("/api/system/info", ws.handleSysInfo)
	mux.HandleFunc("/api/fs/search", ws.handleFSSearch)
	mux.HandleFunc("/api/fs/delete", ws.handleFSDelete)
	mux.HandleFunc("/api/fs/mkdir", ws.handleFSMkdir)
	mux.HandleFunc("/api/fs/image", ws.handleFSImage)
	mux.HandleFunc("/api/fs/file-info", ws.handleFSFileInfo)
	mux.HandleFunc("/api/fs/hex", ws.handleFSHex)
	mux.HandleFunc("/api/tasks", ws.handleTasks)
	mux.HandleFunc("/api/conversations", ws.handleConversations)
	mux.HandleFunc("/api/conversations/", ws.handleConversationByID)
	mux.HandleFunc("/api/taskplan", ws.handleTaskPlan)
	mux.HandleFunc("/api/system/exec", ws.handleExec)
	mux.HandleFunc("/api/models", ws.handleModels)
	mux.HandleFunc("/api/instructions", ws.handleInstructions)
	mux.HandleFunc("/api/philosophy", ws.handlePhilosophy)
	mux.HandleFunc("/api/mcp/list", ws.handleMCPList)
	mux.HandleFunc("/api/mcp/save", ws.handleMCPSave)
	mux.HandleFunc("/api/tools", handler.HandleTools)
	mux.HandleFunc("/api/tools/save", handler.HandleToolsSave)
	mux.HandleFunc("/api/tools/review", handler.HandleReviewConfig)
	mux.HandleFunc("/api/skills/list", ws.handleSkillsList)
	mux.HandleFunc("/api/skills/save", ws.handleSkillsSave)
	mux.HandleFunc("/api/skills/read", ws.handleSkillsRead)

	mux.HandleFunc("/api/skills/delete", ws.handleSkillsDelete)
	mux.HandleFunc("/api/tokens/stats", ws.handleTokensStats)
	mux.HandleFunc("/api/debug/logs", ws.handleDebugLogs)
	mux.HandleFunc("/api/debug/logs/", ws.handleDebugLogByID)

	// ── Git API 路由 ──
	mux.HandleFunc("/api/git/status", ws.handleGitStatus)
	mux.HandleFunc("/api/git/init", ws.handleGitInit)
	mux.HandleFunc("/api/git/diff", ws.handleGitDiff)
	mux.HandleFunc("/api/git/add", ws.handleGitAdd)
	mux.HandleFunc("/api/git/reset", ws.handleGitReset)
	mux.HandleFunc("/api/git/commit", ws.handleGitCommit)
	mux.HandleFunc("/api/git/log", ws.handleGitLog)
	mux.HandleFunc("/api/git-log", ws.handleGitLog) // 避广告拦截器
	mux.HandleFunc("/api/git/branch", ws.handleGitBranch)
	mux.HandleFunc("/api/git/checkout", ws.handleGitCheckout)
	mux.HandleFunc("/api/git/stash", ws.handleGitStash)
	mux.HandleFunc("/api/git/stash-list", ws.handleGitStashList)
	mux.HandleFunc("/api/git/ignore", ws.handleGitIgnore)
	mux.HandleFunc("/api/git/discard", ws.handleGitDiscard)
	mux.HandleFunc("/api/git/push", ws.handleGitPush)
	mux.HandleFunc("/api/git/pull", ws.handleGitPull)
	mux.HandleFunc("/api/git/remote", ws.handleGitRemote)

	// chat/agent 路由由 registerExtraHandlers 注册（桌面 vs webonly 不同实现）
	registerExtraHandlers(mux, ws)

	// 启动事件持久化后台 worker：订阅 SubscribeAll，处理 token/历史持久化。
	// 与传输层（WebSocket）解耦——即使无前端连接，agent 事件仍会被持久化。
	ws.startEventPersistWorker()

	// ★ 预热 system prompt 编译缓存（启动时在后台线程中预热，不阻塞主流程）
	go agent.PromptCacheWarmer.WarmUp(buildWebSystemPrompt)

	// ── 静态文件 ──
	subFS, err := fs.Sub(webUIFiles, "web-ui/dist")
	if err != nil {
		log.Printf("[WebUI] 内嵌资源加载失败: %v", err)
		return
	}
	fileServer := http.FileServer(http.FS(subFS))
	// 禁止浏览器缓存前端文件（避免 Edge 等浏览器加载旧版本 JS/CSS）
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		// ide_ref.html：调试参照收集器（Edge headless 访问 9090 加载真实前端）。
		if r.URL.Path == "/ide_ref.html" || r.URL.Path == "/ide_ref" {
			if data, err := ideRefFile.ReadFile("web-ui/ide_ref.html"); err == nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write(data)
				return
			}
		}
		fileServer.ServeHTTP(w, r)
	}))

	ws.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: corsMiddleware(mux),
	}

	go func() {
		log.Printf("[WebUI] PairCode Web IDE 启动于 http://localhost:%d", port)
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[WebUI] 服务器错误: %v", err)
		}
	}()
}

// stopWebUI 停止 Web 服务器。
func stopWebUI() {
	if ws != nil && ws.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		ws.server.Shutdown(ctx)
	}
}

// corsMiddleware 添加 CORS 头。
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ─── API 处理函数 ────────────────────────────────────────────

func (s *webServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]any{"status": "ok", "workspace": core.Root(), "folders": core.Folders})
}

func (s *webServer) handleFSDrives(w http.ResponseWriter, r *http.Request) {
	drives := []string{}
	for _, d := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		p := string(d) + ":\\"
		if _, err := os.Stat(p); err == nil {
			drives = append(drives, p)
		}
	}
	jsonResp(w, drives)
}

func (s *webServer) handleFSList(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = core.Root()
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	type entry struct {
		Name    string `json:"name"`
		IsDir   bool   `json:"isDir"`
		Size    int64  `json:"size"`
		ModTime string `json:"modTime"`
	}
	result := make([]entry, 0, len(entries))
	for _, e := range entries {
		fi, err := e.Info()
		sz := int64(0)
		mt := ""
		if err == nil {
			sz = fi.Size()
			mt = fi.ModTime().Format("2006-01-02 15:04:05")
		}
		result = append(result, entry{
			Name: e.Name(), IsDir: e.IsDir(), Size: sz, ModTime: mt,
		})
	}
	jsonResp(w, result)
}


func (s *webServer) handleFSRead(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	data, err := os.ReadFile(path)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{
		"content": string(data),
		"size":    len(data),
		"path":    path,
	})
}

// imageMime 根据扩展名返回图片 MIME 类型。
func imageMime(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	}
	return ""
}

// handleFSImage 提供图片浏览（直接返回原始字节 + 正确 Content-Type）。
func (s *webServer) handleFSImage(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Query().Get("path"))
	mime := imageMime(path)
	if mime == "" {
		jsonErr(w, "不支持的文件类型")
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		jsonErr(w, "读取文件失败")
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Write(data)
}

// handleFSFileInfo 返回文件类型信息（用于前端判断展示方式）。
func (s *webServer) handleFSFileInfo(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Query().Get("path"))
	fi, err := os.Stat(path)
	if err != nil {
		jsonErr(w, "文件不存在或无法访问")
		return
	}
	if fi.IsDir() {
		jsonResp(w, map[string]any{"type": "directory", "size": fi.Size()})
		return
	}
	if fi.Size() > 100<<20 {
		jsonErr(w, "文件超过 100MB")
		return
	}

	header := make([]byte, 512)
	f, _ := os.Open(path)
	n, _ := f.Read(header)
	f.Close()

	mime := imageMime(path)
	isImage := mime != ""
	isBinary := n > 0 && bytes.IndexByte(header[:n], 0) >= 0

	fileType := "text"
	if isImage {
		fileType = "image"
	} else if isBinary {
		fileType = "binary"
	}

	jsonResp(w, map[string]any{
		"type":     fileType,
		"size":     fi.Size(),
		"isImage":  isImage,
		"isBinary": isBinary,
		"mimeType": mime,
	})
}

// handleFSHex 返回文件十六进制转储（支持 offset/length 分块，懒加载）。
func (s *webServer) handleFSHex(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Query().Get("path"))
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("length")

	offset := 0
	if offsetStr != "" {
		offset, _ = strconv.Atoi(offsetStr)
	}
	length := 256
	if limitStr != "" {
		length, _ = strconv.Atoi(limitStr)
		if length > 4096 {
			length = 4096
		}
	}

	fi, err := os.Stat(path)
	if err != nil {
		jsonErr(w, "文件不存在")
		return
	}
	fileSize := fi.Size()
	if fileSize > 100<<20 {
		jsonErr(w, "文件超过 100MB")
		return
	}

	f, err := os.Open(path)
	if err != nil {
		jsonErr(w, "无法打开文件")
		return
	}
	defer f.Close()

	if offset > int(fileSize) {
		offset = int(fileSize)
	}
	if offset+length > int(fileSize) {
		length = int(fileSize) - offset
	}
	if length <= 0 {
		jsonResp(w, map[string]any{
			"hex":      "",
			"offset":   offset,
			"fileSize": fileSize,
			"hasMore":  false,
		})
		return
	}

	buf := make([]byte, length)
	n, err := f.ReadAt(buf, int64(offset))
	if err != nil && n == 0 {
		jsonErr(w, "读取失败")
		return
	}
	buf = buf[:n]

	var lines []string
	bpl := 16
	for i := 0; i < n; i += bpl {
		end := i + bpl
		if end > n {
			end = n
		}
		chunk := buf[i:end]
		addr := fmt.Sprintf("%08X", offset+i)

		hexPart := ""
		for j, b := range chunk {
			if j > 0 && j%8 == 0 {
				hexPart += " "
			}
			hexPart += fmt.Sprintf("%02X ", b)
		}
		pad := bpl - len(chunk)
		for j := 0; j < pad; j++ {
			if j > 0 && (len(chunk)+j)%8 == 0 {
				hexPart += " "
			}
			hexPart += "   "
		}

		asciiPart := ""
		for _, b := range chunk {
			if b >= 32 && b <= 126 {
				asciiPart += string(b)
			} else {
				asciiPart += "."
			}
		}
		lines = append(lines, fmt.Sprintf("%s  %s %s", addr, hexPart, asciiPart))
	}

	jsonResp(w, map[string]any{
		"hex":      strings.Join(lines, "\n"),
		"fileSize": fileSize,
		"hasMore":  offset+n < int(fileSize),
	})
}

func (s *webServer) handleFSWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if err := os.MkdirAll(filepath.Dir(req.Path), 0755); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if err := os.WriteFile(req.Path, []byte(req.Content), 0644); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "path": req.Path})
}

func (s *webServer) handleFSRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if err := os.MkdirAll(filepath.Dir(req.To), 0755); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if err := os.Rename(req.From, req.To); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "from": req.From, "to": req.To})
}

func (s *webServer) handleFSDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Path == "" {
		jsonErr(w, "path 必填")
		return
	}
	fi, err := os.Stat(req.Path)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	if fi.IsDir() {
		if err := os.RemoveAll(req.Path); err != nil {
			jsonErr(w, err.Error())
			return
		}
	} else {
		if err := os.Remove(req.Path); err != nil {
			jsonErr(w, err.Error())
			return
		}
	}
	jsonResp(w, map[string]any{"ok": true, "path": req.Path})
}

func (s *webServer) handleFSMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Path == "" {
		jsonErr(w, "path 必填")
		return
	}
	if err := os.MkdirAll(req.Path, 0755); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "path": req.Path})
}

func (s *webServer) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		jsonResp(w, map[string]any{
			"root":    core.Root(),
			"folders": core.Folders,
			"loaded":  core.Loaded,
		})
	case "POST":
		var req struct {
			Action      string   `json:"action"`
			Root        string   `json:"root"`
			Folders     []string `json:"folders"`
			Name        string   `json:"name"`
			Path        string   `json:"path"`
			ParentDir   string   `json:"parentDir"`
			Lang        string   `json:"lang"`
			DeleteFiles bool     `json:"deleteFiles"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error())
			return
		}
		switch req.Action {
		case "create":
			if req.Name == "" && req.Root == "" {
				jsonErr(w, "需要 name 或 root 参数")
				return
			}
			root := req.Root
			if root == "" {
				home, _ := os.UserHomeDir()
				root = filepath.Join(home, "paircode-workspaces", req.Name)
			}
			if err := os.MkdirAll(root, 0755); err != nil {
				jsonErr(w, "创建工作区失败: "+err.Error())
				return
			}
			core.Folders = []string{root}
			core.Settings.LastProject = root
			core.Settings.WorkspaceFolders = core.Folders
			core.Loaded = true
			core.Save()
			if core.OnSyncWorkspace != nil {
				core.OnSyncWorkspace(true)
			}
			jsonResp(w, map[string]any{"ok": true, "root": root})

		case "add-folder":
			if req.Path == "" {
				jsonErr(w, "需要 path 参数")
				return
			}
			if _, err := os.Stat(req.Path); err != nil {
				jsonErr(w, "目录不存在: "+err.Error())
				return
			}
			for _, f := range core.Folders {
				if f == req.Path {
					jsonResp(w, map[string]any{"ok": true, "note": "already exists"})
					return
				}
			}
			core.Folders = append(core.Folders, req.Path)
			core.Settings.WorkspaceFolders = core.Folders
			core.Save()
			if core.OnSyncWorkspace != nil {
				core.OnSyncWorkspace(true)
			}
			jsonResp(w, map[string]any{"ok": true, "folders": core.Folders})

		case "remove-folder":
			if req.Path == "" {
				jsonErr(w, "需要 path 参数")
				return
			}
			newFolders := make([]string, 0, len(core.Folders))
			for _, f := range core.Folders {
				if f != req.Path {
					newFolders = append(newFolders, f)
				}
			}
			core.Folders = newFolders
			core.Settings.WorkspaceFolders = core.Folders
			core.Save()
			if core.OnSyncWorkspace != nil {
				core.OnSyncWorkspace(true)
			}
			jsonResp(w, map[string]any{"ok": true, "folders": core.Folders})

		case "new-project":
			parent := req.ParentDir
			if parent == "" {
				parent = core.Root()
				if parent == "" {
					jsonErr(w, "需要 parentDir 参数或先设置工作区")
					return
				}
			}
			if req.Name == "" {
				jsonErr(w, "需要 name 参数")
				return
			}
			projPath := filepath.Join(parent, req.Name)
			if err := os.MkdirAll(projPath, 0755); err != nil {
				jsonErr(w, "创建项目失败: "+err.Error())
				return
			}
			lang := req.Lang
			if lang == "" {
				lang = detectLang(req.Name)
			}
			genProjectTemplate(projPath, lang, req.Name)
			found := false
			for _, f := range core.Folders {
				if f == projPath {
					found = true
					break
				}
			}
			if !found {
				core.Folders = append(core.Folders, projPath)
				core.Settings.WorkspaceFolders = core.Folders
				core.Save()
			}
			jsonResp(w, map[string]any{"ok": true, "path": projPath, "lang": lang})

		case "delete":
			if req.Root == "" {
				jsonErr(w, "需要 root 参数（工作区路径）")
				return
			}
			root := req.Root
			// 1. 从 RecentProjects 中移除
			newProjects := make([]string, 0, len(core.Settings.RecentProjects))
			for _, p := range core.Settings.RecentProjects {
				if p != root {
					newProjects = append(newProjects, p)
				}
			}
			core.Settings.RecentProjects = newProjects

			// 2. 从 WorkspaceFolders 中移除
			newFolders := make([]string, 0, len(core.Settings.WorkspaceFolders))
			for _, f := range core.Settings.WorkspaceFolders {
				if f != root {
					newFolders = append(newFolders, f)
				}
			}
			core.Settings.WorkspaceFolders = newFolders

			// 3. 从 core.Folders 中移除
			core.Folders = newFolders

			// 4. 如果 LastProject 匹配，清空（避免 health 接口仍返回已删除的工作区）
			if core.Settings.LastProject == root {
				core.Settings.LastProject = ""
				if len(core.Folders) > 0 {
					core.Settings.LastProject = core.Folders[0]
				}
			}
			if core.Settings.LastProject == "" {
				core.Loaded = false
			}
			core.Save()

			// 5. 可选：删除工作区下的 .pair 目录（对话历史、快照等）
			if req.DeleteFiles {
				pairDir := filepath.Join(root, ".pair")
				if stat, err := os.Stat(pairDir); err == nil && stat.IsDir() {
					if err := os.RemoveAll(pairDir); err != nil {
						log.Printf("[Workspace] 删除 %s 失败: %v", pairDir, err)
					} else {
						log.Printf("[Workspace] 已删除 %s", pairDir)
					}
				}
			}

			// 同步工作区变更
			if core.OnSyncWorkspace != nil {
				core.OnSyncWorkspace(true)
			}
			jsonResp(w, map[string]any{"ok": true})

		default:
			if req.Root != "" {
				core.Folders = append([]string{req.Root}, req.Folders...)
				core.Settings.LastProject = req.Root
				core.Settings.WorkspaceFolders = core.Folders
				core.Loaded = true
				core.Save()
				if core.OnSyncWorkspace != nil {
					core.OnSyncWorkspace(true)
				}
			}
			jsonResp(w, map[string]any{"ok": true})
		}
	default:
		jsonErr(w, "不支持的方法")
	}
}

func (s *webServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		jsonResp(w, core.Settings)
	case "PUT":
		// convId 参数可选：来自工具栏切换时传当前对话 ID，仅实时更新该对话的 Loop
		convId := r.URL.Query().Get("convId")
		// 读取原始请求体
		body, err := io.ReadAll(r.Body)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		// 解析为 map 判断哪些字段被提交
		var rawMap map[string]json.RawMessage
		if err := json.Unmarshal(body, &rawMap); err != nil {
			jsonErr(w, err.Error())
			return
		}

		// 检查审核模式是否变更
		oldReviewMode := core.Settings.ReviewMode

		// 增量 merge：只更新 rawMap 中存在的字段
		sv := reflect.ValueOf(&core.Settings).Elem()
		st := sv.Type()
		for i := 0; i < sv.NumField(); i++ {
			field := st.Field(i)
			jsonKey := strings.Split(field.Tag.Get("json"), ",")[0]
			if jsonKey == "" || jsonKey == "-" {
				continue
			}
			rawVal, ok := rawMap[jsonKey]
			if !ok {
				continue
			}
			// 用 json.Unmarshal 解析到字段类型，保留类型精度
			newVal := reflect.New(field.Type).Interface()
			if err := json.Unmarshal(rawVal, newVal); err == nil {
				sv.Field(i).Set(reflect.ValueOf(newVal).Elem())
			}
		}

		core.Save()
		// 审核模式变更时，仅实时更新当前对话的 Loop（其他对话不受影响）
		if newReviewMode, ok := rawMap["reviewMode"]; ok {
			var newVal string
			if err := json.Unmarshal(newReviewMode, &newVal); err == nil && newVal != oldReviewMode && convId != "" {
				agentMgr.SetReviewMode(convId, newVal)
			}
		}
		// 同步工作区文件夹列表（确保 core.Folders 与 settings 一致）
		if _, ok := rawMap["workspaceFolders"]; ok && core.Settings.WorkspaceFolders != nil {
			core.Folders = core.Settings.WorkspaceFolders
			if core.OnSyncWorkspace != nil {
				core.OnSyncWorkspace(false)
			}
		}
		jsonResp(w, map[string]any{"ok": true})
	default:
		jsonErr(w, "不支持的方法")
	}
}

func (s *webServer) handleSysInfo(w http.ResponseWriter, r *http.Request) {
	host, _ := os.Hostname()
	cwd, _ := os.Getwd()
	jsonResp(w, map[string]any{
		"hostname":  host,
		"cwd":       cwd,
		"os":        "windows",
		"goos":      runtime.GOOS,
		"goarch":    runtime.GOARCH,
		"goversion": runtime.Version(),
		"workspace": core.Root(),
		"folders":   core.Folders,
		"version":   version,
	})
}
func (s *webServer) handleFSSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	searchPath := r.URL.Query().Get("path")
	if q == "" {
		jsonErr(w, "q 参数必填")
		return
	}
	if searchPath == "" {
		searchPath = core.Root()
	}
	type match struct {
		File string `json:"file"`
		Line int    `json:"line"`
		Text string `json:"text"`
	}
	results := []match{}
	seen := map[string]bool{}

	skipDirs := map[string]bool{
		".git": true, "node_modules": true, "vendor": true,
		".pair": true, ".trae": true, ".dbg": true, ".context": true,
		"__pycache__": true, ".venv": true, "venv": true,
		"bin": true, "obj": true, ".vs": true,
	}

	textExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".vue": true, ".html": true,
		".css": true, ".scss": true, ".json": true, ".md": true, ".yml": true, ".yaml": true,
		".xml": true, ".py": true, ".java": true, ".rs": true, ".c": true, ".h": true,
		".cpp": true, ".hpp": true, ".sh": true, ".bat": true, ".ps1": true, ".env": true,
		".gitignore": true, ".dockerfile": true, ".sql": true, ".rb": true, ".php": true,
		".swift": true, ".kt": true, ".toml": true, ".ini": true, ".cfg": true, ".conf": true,
		".txt": true, ".log": true, ".csv": true, ".tsv": true, ".svg": true,
		".svelte": true, ".astro": true, ".gradle": true, ".cmake": true,
		".lua": true, ".pl": true, ".pm": true, ".r": true, ".dart": true,
		".scala": true, ".zig": true, ".nim": true, ".hbs": true, ".ejs": true,
	}

	const maxUnknownSize = 512 * 1024
	const maxTextSize = 10 * 1024 * 1024
	const maxResults = 200

	filepath.Walk(searchPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			if skipDirs[strings.ToLower(fi.Name())] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if textExts[ext] {
			if fi.Size() > maxTextSize {
				return nil
			}
		} else {
			if fi.Size() > maxUnknownSize {
				return nil
			}
		}
		if len(results) >= maxResults {
			return filepath.SkipAll
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}

		header := make([]byte, 512)
		n, _ := f.Read(header)
		if n > 0 {
			for _, b := range header[:n] {
				if b == 0 {
					f.Close()
					return nil
				}
			}
		}
		f.Seek(0, 0)

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if strings.Contains(line, q) {
				if !seen[path] {
					seen[path] = true
					results = append(results, match{
						File: path,
						Line: lineNum,
						Text: strings.TrimSpace(line),
					})
					if len(results) >= maxResults {
						f.Close()
						return filepath.SkipAll
					}
				}
				break
			}
		}
		f.Close()
		return nil
	})
	jsonResp(w, results)
}

func (s *webServer) handleTasks(w http.ResponseWriter, r *http.Request) {
	root := core.Root()
	if root == "" {
		jsonResp(w, map[string]any{"tasks": []any{}})
		return
	}
	switch r.Method {
	case "GET":
		// 从 TaskManager 持久化目录 .pair/tasks/*.json 读取真实任务状态
		convID := r.URL.Query().Get("convId")
		type planStep struct {
			Step          string `json:"step"`
			Status        string `json:"status"`
			TaskID        string `json:"taskId"`
			Description   string `json:"description"`
			PlanStepIndex *int   `json:"planStepIndex,omitempty"`
			CreatedAt     string `json:"created_at"`
		}
		tasksDir := filepath.Join(root, ".pair", "tasks")
		entries, err := os.ReadDir(tasksDir)
		if err != nil {
			jsonResp(w, map[string]any{"tasks": []any{}})
			return
		}
		plan := make([]planStep, 0)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(tasksDir, e.Name()))
			if err != nil {
				continue
			}
			var t struct {
				ID            string `json:"id"`
				Subject       string `json:"subject"`
				Description   string `json:"description"`
				Status        string `json:"status"`
				ConvID        string `json:"convId"`
				PlanStepIndex *int   `json:"planStepIndex"`
				CreatedAt     string `json:"created_at"`
			}
			if err := json.Unmarshal(data, &t); err != nil {
				continue
			}
			// 按对话过滤：convID 非空时只返回该对话的任务
			if convID != "" && t.ConvID != convID {
				continue
			}
			plan = append(plan, planStep{
				Step:          t.Subject,
				Status:        t.Status,
				TaskID:        t.ID,
				Description:   t.Description,
				PlanStepIndex: t.PlanStepIndex,
				CreatedAt:     t.CreatedAt,
			})
		}
		// 按创建时间倒序排列（最新的在前）
		sort.Slice(plan, func(i, j int) bool { return plan[i].CreatedAt > plan[j].CreatedAt })
		jsonResp(w, map[string]any{"tasks": plan})
	default:
		jsonErr(w, "不支持的方法")
	}
}

// ─── 终端命令执行 API ──────────────────────────────────────────

func (s *webServer) handleExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Command string `json:"command"`
		Cwd     string `json:"cwd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Command == "" {
		jsonErr(w, "command 必填")
		return
	}
	currDir := req.Cwd
	if currDir == "" {
		currDir = core.Root()
		if currDir == "" {
			currDir, _ = os.Getwd()
		}
	}

	cmdName := "cmd.exe"
	args := []string{"/C", req.Command}

	cmd := exec.CommandContext(r.Context(), cmdName, args...)
	cmd.Dir = currDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := map[string]any{
		"stdout":   stdout.String(),
		"stderr":   stderr.String(),
		"exitCode": 0,
		"cwd":      currDir,
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exitCode"] = exitErr.ExitCode()
		} else {
			result["stderr"] = err.Error()
			result["exitCode"] = -1
		}
	}
	jsonResp(w, result)
}

// ─── 对话列表 API ────────────────────────────────────────────

func (s *webServer) handleConversations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		wsFilter := r.URL.Query().Get("workspace")
		store := agentMgr.Store()
		if store == nil {
			jsonResp(w, []agent.ConversationMeta{})
			return
		}
		metas, err := store.ListConversations(wsFilter)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		if metas == nil {
			metas = []agent.ConversationMeta{}
		}
		jsonResp(w, metas)

	case "POST":
		var req struct {
			ID            string `json:"id"`
			Title         string `json:"title"`
			WorkspaceRoot string `json:"workspaceRoot"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error())
			return
		}
		store := agentMgr.Store()
		if store == nil {
			jsonErr(w, "消息存储未初始化")
			return
		}
		id := req.ID
		if id == "" {
			id = fmt.Sprintf("conv_%d", time.Now().UnixNano())
		}
		wsRoot := req.WorkspaceRoot
		if wsRoot == "" {
			wsRoot = core.Root()
		}
		title := req.Title
		if title == "" {
			title = "新对话 " + time.Now().Format("15:04")
		}
		if err := store.CreateConversation(id, title, wsRoot); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "id": id, "title": title})

	default:
		jsonErr(w, "不支持的方法")
	}
}

func (s *webServer) handleConversationByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/conversations/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		jsonErr(w, "缺少对话 ID")
		return
	}
	id := parts[0]
	sub := ""
	if len(parts) >= 2 {
		sub = parts[1]
	}
	subSub := ""
	if len(parts) >= 3 {
		subSub = parts[2]
	}

	store := agentMgr.Store()
	if store == nil {
		jsonErr(w, "消息存储未初始化")
		return
	}

	switch r.Method {
	case "GET":
		switch {
		case sub == "messages" && subSub == "count":
			n, err := store.Count(id)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			jsonResp(w, map[string]any{"count": n})

		case sub == "messages":
			limit := 50
			if l := r.URL.Query().Get("limit"); l != "" {
				fmt.Sscanf(l, "%d", &limit)
			}
			beforeStr := r.URL.Query().Get("before")
			if beforeStr != "" {
				var before int
				fmt.Sscanf(beforeStr, "%d", &before)
				msgs, err := store.LoadBefore(id, before, limit)
				if err != nil {
					jsonErr(w, err.Error())
					return
				}
			if msgs == nil {
				msgs = []agent.StoredMessage{}
			}
			msgs = agent.MergeConsecutiveAssistants(msgs)
			total, _ := store.Count(id)
			jsonResp(w, map[string]any{"messages": msgs, "total": total})
		} else {
			msgs, total, err := store.LoadLatest(id, limit)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			if msgs == nil {
				msgs = []agent.StoredMessage{}
			}
			msgs = agent.MergeConsecutiveAssistants(msgs)
			jsonResp(w, map[string]any{"messages": msgs, "total": total})
			}

		case sub == "token-stats":
			meta, err := store.GetConversation(id)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			if meta == nil || meta.CtxStats == nil {
				jsonResp(w, map[string]any{
					"promptTokens": 0, "completionTokens": 0, "totalTokens": 0,
					"cacheHitTokens": 0, "cacheMissTokens": 0,
				})
				return
			}
			cs := meta.CtxStats
			m := map[string]any{
				"promptTokens":     cs.PromptTokens,
				"completionTokens": cs.CompletionTokens,
				"totalTokens":      cs.TotalTokens,
				"cacheHitTokens":   cs.PromptCacheHitTokens,
				"cacheMissTokens":  cs.PromptCacheMissTokens,
			}
			if cs.PromptBreakdown.SystemTokens > 0 || cs.PromptBreakdown.SkillsTokens > 0 ||
				cs.PromptBreakdown.MCPTokens > 0 || cs.PromptBreakdown.ToolTokens > 0 {
				m["systemTokens"] = cs.SystemTokens
				m["skillsTokens"] = cs.SkillsTokens
				m["mcpTokens"] = cs.MCPTokens
				m["toolTokens"] = cs.ToolTokens
				m["historyTokens"] = cs.HistoryTokens
				m["otherTokens"] = cs.OtherTokens
			}
			jsonResp(w, m)

		case sub == "":
			meta, err := store.GetConversation(id)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			if meta == nil {
				jsonErr(w, "对话不存在")
				return
			}
			jsonResp(w, meta)

		default:
			jsonErr(w, "未知的子路径: "+sub)
		}

	case "PUT":
		var req struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error())
			return
		}
		if req.Title != "" {
			if err := store.UpdateTitle(id, req.Title); err != nil {
				jsonErr(w, err.Error())
				return
			}
		}
		jsonResp(w, map[string]any{"ok": true})

	case "POST":
		if sub != "messages" {
			jsonErr(w, "请使用 /conversations/{id}/messages")
			return
		}
		var msg struct {
			Role     string          `json:"role"`
			Content  string          `json:"content"`
			Segments []agent.Segment `json:"segments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			jsonErr(w, err.Error())
			return
		}
		if msg.Role == "" {
			msg.Role = "user"
		}
		agentMsg := agent.Message{Role: agent.Role(msg.Role), Content: msg.Content}
		if err := store.AppendMessage(id, agentMsg, msg.Segments); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true})

	case "DELETE":
		if err := store.DeleteConversation(id); err != nil {
			jsonErr(w, err.Error())
			return
		}
		// 同步删除记忆索引（跨对话摘要索引）
		memory.Delete(id)
		jsonResp(w, map[string]any{"ok": true})

	default:
		jsonErr(w, "不支持的方法")
	}
}

// ─── Agent 工作流规划文档 API ──────────────────────────────────

func (s *webServer) handleTaskPlan(w http.ResponseWriter, r *http.Request) {
	root := core.Root()
	if root == "" {
		jsonErr(w, "未设置工作区")
		return
	}
	pairDir := filepath.Join(root, ".pair")
	tasksDir := filepath.Join(pairDir, "tasks")
	os.MkdirAll(tasksDir, 0755)

	switch r.Method {
	case "GET":
		name := r.URL.Query().Get("name")
		if name != "" {
			filePath := filepath.Join(tasksDir, name+".md")
			data, err := os.ReadFile(filePath)
			if err != nil {
				jsonErr(w, "规划文档不存在: "+err.Error())
				return
			}
			jsonResp(w, map[string]any{"name": name, "content": string(data)})
			return
		}
		entries, err := os.ReadDir(tasksDir)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		plans := make([]map[string]any, 0)
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				plans = append(plans, map[string]any{
					"name": strings.TrimSuffix(e.Name(), ".md"),
					"file": filepath.Join(tasksDir, e.Name()),
				})
			}
		}
		jsonResp(w, plans)

	case "POST":
		var req struct {
			Name    string `json:"name"`
			Content string `json:"content"`
			Action  string `json:"action"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error())
			return
		}
		if req.Name == "" {
			req.Name = fmt.Sprintf("plan_%s", time.Now().Format("20060102_150405"))
		}
		filePath := filepath.Join(tasksDir, req.Name+".md")
		if req.Action == "append" || req.Action == "complete" {
			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			defer f.Close()
			if req.Content != "" {
				if _, err := f.WriteString("\n" + req.Content + "\n"); err != nil {
					jsonErr(w, err.Error())
					return
				}
			}
			if req.Action == "complete" {
				summary := fmt.Sprintf("\n## 完成时间\n%s\n\n---\n*任务规划已完成*\n", time.Now().Format("2006-01-02 15:04:05"))
				f.WriteString(summary)
			}
		} else {
			header := fmt.Sprintf("# 任务规划: %s\n\n- 创建时间: %s\n- 状态: 进行中\n\n", req.Name, time.Now().Format("2006-01-02 15:04:05"))
			content := header + req.Content
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				jsonErr(w, err.Error())
				return
			}
		}
		data, _ := os.ReadFile(filePath)
		jsonResp(w, map[string]any{"ok": true, "file": filePath, "content": string(data)})

	default:
		jsonErr(w, "不支持的方法")
	}
}

// ─── 模型列表 API ──────────────────────────────────────────

func (s *webServer) handleModels(w http.ResponseWriter, r *http.Request) {
	providers := core.GetProviders()
	modelMap := make(map[string][]string)
	for _, p := range providers {
		modelMap[p] = core.GetModels(p)
	}
	jsonResp(w, map[string]any{
		"providers":    providers,
		"models":       modelMap,
		"providerBaseURLs": core.GetProviderBaseURLs(),
	})
}

// ─── 指令 API ──────────────────────────────────────────────

func (s *webServer) handleInstructions(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "system"
	}

	switch scope {
	case "project":
		root := core.Root()
		if root == "" {
			jsonErr(w, "未设置工作区")
			return
		}
		pairDir := filepath.Join(root, ".pair")
		os.MkdirAll(pairDir, 0755)
		instPath := filepath.Join(pairDir, "instructions.md")

		switch r.Method {
		case "GET":
			data, err := os.ReadFile(instPath)
			if err != nil {
				jsonResp(w, map[string]any{"content": "", "path": instPath})
				return
			}
			jsonResp(w, map[string]any{"content": string(data), "path": instPath})

		case "PUT":
			var req struct {
				Content string `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				jsonErr(w, err.Error())
				return
			}
			if err := os.WriteFile(instPath, []byte(req.Content), 0644); err != nil {
				jsonErr(w, err.Error())
				return
			}
			jsonResp(w, map[string]any{"ok": true, "path": instPath})

		default:
			jsonErr(w, "不支持的方法")
		}

	default:
		switch r.Method {
		case "GET":
			jsonResp(w, map[string]any{"content": core.Settings.SystemInstructions})

		case "PUT":
			var req struct {
				Content string `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				jsonErr(w, err.Error())
				return
			}
			core.Settings.SystemInstructions = req.Content
			core.Save()
			jsonResp(w, map[string]any{"ok": true})

		default:
			jsonErr(w, "不支持的方法")
		}
	}
}

// ─── 思想配置 API ──────────────────────────────────────────

func (s *webServer) handlePhilosophy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		jsonResp(w, map[string]any{
			"enabled":  core.Settings.PhilosophyEnabled,
			"selected": core.Settings.PhilosophySelected,
			"roles":    core.Settings.PhilosophyRoles,
			"availableClassics": []map[string]string{
				{"id": "tao-te-ching", "name": "《道德经》"},
				{"id": "huangdi-yinfu-jing", "name": "《黄帝阴符经》"},
				{"id": "sunzi-bingfa", "name": "《孙子兵法》"},
				{"id": "lunyu", "name": "《论语》"},
				{"id": "yijing", "name": "《易经》"},
				{"id": "zhongyong", "name": "《中庸》"},
				{"id": "daxue", "name": "《大学》"},
			},
			"availableRoles": []map[string]string{
				{"id": "planner", "name": "规划 Agent"},
				{"id": "reviewer", "name": "审核 Agent"},
				{"id": "judge", "name": "评测 Agent"},
				{"id": "explorer", "name": "探索 Agent"},
				{"id": "verifier", "name": "验证 Agent"},
				{"id": "debugger", "name": "调试 Agent"},
				{"id": "executor", "name": "执行 Agent"},
			},
		})

	case "PUT":
		var req struct {
			Enabled  bool              `json:"enabled"`
			Selected []string          `json:"selected"`
			Roles    map[string]string `json:"roles"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error())
			return
		}
		core.Settings.PhilosophyEnabled = req.Enabled
		if req.Selected != nil {
			core.Settings.PhilosophySelected = req.Selected
		}
		if req.Roles != nil {
			if core.Settings.PhilosophyRoles == nil {
				core.Settings.PhilosophyRoles = make(map[string]string)
			}
			for k, v := range req.Roles {
				core.Settings.PhilosophyRoles[k] = v
			}
		}
		core.Save()
		jsonResp(w, map[string]any{"ok": true})

	default:
		jsonErr(w, "不支持的方法")
	}
}

// ─── MCP 列表 API ──────────────────────────────────────────

func (s *webServer) handleMCPList(w http.ResponseWriter, r *http.Request) {
	lv := r.URL.Query().Get("level")
	type mcpItem struct {
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Level   string   `json:"level"`
		Enabled bool     `json:"enabled"`
	}
	out := make([]mcpItem, 0)

	if lv == "" || lv == "all" || lv == "user" {
		for _, e := range mcppanel.ReadLevel(mcppanel.LevelUser) {
			out = append(out, mcpItem{
				Name: e.Name, Command: e.Command, Args: e.Args,
				Level: "user", Enabled: mcppanel.Enabled(mcppanel.LevelUser, e.Name),
			})
		}
	}
	if lv == "" || lv == "all" || lv == "project" {
		for _, e := range mcppanel.ReadLevel(mcppanel.LevelProject) {
			out = append(out, mcpItem{
				Name: e.Name, Command: e.Command, Args: e.Args,
				Level: "project", Enabled: mcppanel.Enabled(mcppanel.LevelProject, e.Name),
			})
		}
	}
	jsonResp(w, out)
}

// ─── MCP 保存/删除 API ────────────────────────────────────

func (s *webServer) handleMCPSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Action  string   `json:"action"`
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Level   string   `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	lv := mcppanel.LevelUser
	if req.Level == "project" {
		lv = mcppanel.LevelProject
	}

	switch req.Action {
	case "delete":
		if err := mcppanel.Delete(lv, req.Name); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "action": "deleted", "name": req.Name})
	case "toggle":
		old := mcppanel.Enabled(lv, req.Name)
		if err := mcppanel.SetEnabled(lv, req.Name, !old); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "action": "toggled", "name": req.Name, "enabled": !old})
	default:
		if req.Name == "" || req.Command == "" {
			jsonErr(w, "name 和 command 必填")
			return
		}
		if err := mcppanel.Upsert(lv, mcppanel.Entry{
			Name: req.Name, Command: req.Command, Args: req.Args,
		}); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "action": "saved", "name": req.Name})
	}
}

// ─── Token 统计 API（数据由 agent 自闭环持久化） ──────────────

func (s *webServer) handleTokensStats(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// 从 agent 自闭环存储读取（优先使用 query 参数中的 workspaceRoot，兜底用 core.Root()）
		wsRoot := r.URL.Query().Get("workspaceRoot")
		if wsRoot == "" {
			wsRoot = core.Root()
		}
		stats := agent.ReadTokenStatsForRoot(wsRoot)
		if stats == nil {
			jsonResp(w, map[string]any{
				"promptTokens": 0, "completionTokens": 0, "totalTokens": 0,
				"cacheHitTokens": 0, "cacheMissTokens": 0,
				"systemTokens": 0, "skillsTokens": 0, "mcpTokens": 0,
				"toolTokens": 0, "historyTokens": 0, "otherTokens": 0,
			})
			return
		}
		jsonResp(w, map[string]any{
			"promptTokens":     stats.PromptTokens,
			"completionTokens": stats.CompletionTokens,
			"totalTokens":      stats.TotalTokens,
			"cacheHitTokens":   stats.CacheHitTokens,
			"cacheMissTokens":  stats.CacheMissTokens,
			"systemTokens":     stats.SystemTokens,
			"skillsTokens":     stats.SkillsTokens,
			"mcpTokens":        stats.MCPTokens,
			"toolTokens":       stats.ToolTokens,
			"historyTokens":    stats.HistoryTokens,
			"otherTokens":      stats.OtherTokens,
		})
	case "POST":
		jsonResp(w, map[string]any{"ok": true, "message": "Token 统计由 agent 自闭环管理，无需手动重置"})
	default:
		jsonErr(w, "不支持的方法")
	}


}

// ─── Skills HTTP API ──────────────────────────────────────
func (s *webServer) handleSkillsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, "仅 GET")
		return
	}
	type skillItem struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Mode        string `json:"mode"`
		Level       string `json:"level"`
		Status      string `json:"status"`
	}
	skills := agent.LoadAllSkills()
	out := make([]skillItem, 0, len(skills))
	for _, sk := range skills {
		origMode := sk.Mode
		key := string(sk.Level) + "::" + sk.Name
		status := "on"
		if s, ok := agent.SkillStatusOverride[key]; ok {
			status = s
		} else {
			if v, ok := agent.SkillEnabled[key]; ok && !v {
				status = "off"
			} else if origMode == "always" {
				status = "max"
			}
		}
		out = append(out, skillItem{
			Name: sk.Name, Description: sk.Description,
			Mode: origMode, Level: string(sk.Level),
			Status: status,
		})
	}

	jsonResp(w, out)
}

func (s *webServer) handleSkillsDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Name == "" {
		jsonErr(w, "name 必填")
		return
	}
	if err := agent.DeleteSkill(agent.SkillProjectDir, req.Name); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "message": "已删除技能: " + req.Name})
}

func (s *webServer) handleSkillsSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Action string `json:"action"`
		Name   string `json:"name"`
		Level  string `json:"level"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Name == "" {
		jsonErr(w, "name 必填")
		return
	}
	lv := agent.LevelProject
	if req.Level == "system" {
		lv = agent.LevelSystem
	}
	switch req.Action {
	case "set-status":
		if req.Status != "off" && req.Status != "on" && req.Status != "max" {
			jsonErr(w, "status 必须为 off/on/max")
			return
		}
		agent.SkillSetStatus(req.Name, lv, req.Status)
		core.Settings.SkillStatusOverrides = agent.SkillStatusOverride
		core.Settings.SkillEnabledOverrides = agent.SkillEnabled
		core.Save()
		jsonResp(w, map[string]any{"ok": true, "action": "set-status", "name": req.Name, "status": req.Status})
	default:
		jsonErr(w, "未知 action: "+req.Action)
	}
}

// ─── 辅助 ────────────────────────────────────────────────────

func jsonResp(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}


// handleSkillsRead 读取技能正文内容。
// 支持 level 查询参数：system（全局 config/skills/）或 project（工作区 .pair/skills/，默认）。
func (s *webServer) handleSkillsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, "仅 GET")
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		jsonErr(w, "name 必填")
		return
	}
	// 白名单校验：仅允许字母、数字、下划线、中划线
	if !validSkillName(name) {
		jsonErr(w, "无效的键名")
		return
	}
	level := r.URL.Query().Get("level")
	var baseDir string
	switch level {
	case "system", "global":
		baseDir = agent.SkillSystemDir
	default:
		baseDir = agent.SkillProjectDir
	}
	skillPath := filepath.Join(baseDir, name, "SKILL.md")
	// 路径穿越防护：校验最终路径必须以 baseDir 开头
	absPath, err := filepath.Abs(skillPath)
	if err != nil || !strings.HasPrefix(absPath, filepath.Clean(baseDir)+string(filepath.Separator)) {
		jsonErr(w, "请求的资源不存在")
		return
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		// 默认(project)级别找不到时回退系统级
		if level == "" || level == "project" {
			if agent.SkillSystemDir != "" {
				fallback := filepath.Join(agent.SkillSystemDir, name, "SKILL.md")
				absFallback, fbErr := filepath.Abs(fallback)
				if fbErr == nil && strings.HasPrefix(absFallback, filepath.Clean(agent.SkillSystemDir)+string(filepath.Separator)) {
					data, fbErr = os.ReadFile(absFallback)
					if fbErr == nil {
						jsonResp(w, map[string]string{"name": name, "content": string(data), "level": "system"})
						return
					}
				}
			}
		}
		jsonErr(w, "请求的资源不存在")
		return
	}
	respLevel := level
	if respLevel == "" {
		respLevel = "project"
	}
	jsonResp(w, map[string]string{"name": name, "content": string(data), "level": respLevel})
}

// validSkillName 校验技能名仅含字母、数字、下划线、中划线。
func validSkillName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func jsonErr(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func jsonStr(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// detectLang 根据项目名或目录内容推测语言。
func detectLang(name string) string {
	name = strings.ToLower(name)
	if strings.Contains(name, "go") || strings.HasSuffix(name, ".go") {
		return "go"
	}
	if strings.Contains(name, "rust") || strings.Contains(name, "rs") {
		return "rust"
	}
	if strings.Contains(name, "py") || strings.Contains(name, "python") {
		return "python"
	}
	if strings.Contains(name, "js") || strings.Contains(name, "node") || strings.Contains(name, "react") || strings.Contains(name, "vue") {
		return "javascript"
	}
	if strings.Contains(name, "ts") || strings.Contains(name, "typescript") {
		return "typescript"
	}
	if strings.Contains(name, "java") {
		return "java"
	}
	if strings.Contains(name, "html") || strings.Contains(name, "web") {
		return "html"
	}
	return "unknown"
}

// genProjectTemplate 在项目目录中生成初始模板文件。
func genProjectTemplate(projPath, lang, name string) {
	switch lang {
	case "go":
		modName := name
		if modName == "" || modName == "unknown" {
			modName = filepath.Base(projPath)
		}
		mainGo := fmt.Sprintf(`package main

import "fmt"

func main() {
	fmt.Println("Hello from %s!")
}
`, modName)
		os.WriteFile(filepath.Join(projPath, "main.go"), []byte(mainGo), 0644)
		os.WriteFile(filepath.Join(projPath, "go.mod"), []byte(fmt.Sprintf("module %s\n\ngo 1.24\n", modName)), 0644)

	case "python":
		os.WriteFile(filepath.Join(projPath, "main.py"), []byte(fmt.Sprintf(`# %s
def main():
    print("Hello from %s!")

if __name__ == "__main__":
    main()
`, name, name)), 0644)

	case "javascript", "typescript":
		os.WriteFile(filepath.Join(projPath, "index.js"), []byte(fmt.Sprintf(`// %s
console.log("Hello from %s!");
`, name, name)), 0644)

	case "html":
		os.WriteFile(filepath.Join(projPath, "index.html"), []byte(fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>%s</title>
</head>
<body>
  <h1>Hello from %s!</h1>
</body>
</html>`, name, name)), 0644)

	case "java":
		className := "App"
		if name != "" && name != "unknown" {
			parts := strings.Split(name, "-")
			for i, p := range parts {
				if len(p) > 0 {
					parts[i] = strings.ToUpper(p[:1]) + p[1:]
				}
			}
			className = strings.Join(parts, "")
		}
		os.WriteFile(filepath.Join(projPath, className+".java"), []byte(fmt.Sprintf(`public class %s {
    public static void main(String[] args) {
        System.out.println("Hello from %s!");
    }
}
`, className, name)), 0644)
	}
}

// loadConversationHistory 从 MessageStore 加载对话的全部过往消息作为 LLM 上下文。
// 委托给 agentMgr.Store().LoadAll(convID)；store 未注入时返回 nil。
func (s *webServer) loadConversationHistory(convID string) []agent.Message {
	if convID == "" {
		return nil
	}
	store := agentMgr.Store()
	if store == nil {
		return nil
	}
	msgs, err := store.LoadAll(convID)
	if err != nil {
		return nil
	}
	return msgs
}

// ── 统一 API 层（平台无关，差异通过回调参数化） ─────────────────

// systemStaticPrefixCache 缓存 system prompt 静态前缀（CACHE_BOUNDARY 之前部分）。
// 键由 workspace roots hash + SystemInstructions + Philosophy 构成；
// 只要这些不变，静态前缀就无需重新拼接。
var systemStaticPrefixCache struct {
	mu     sync.Mutex
	key    string
	prefix string
}

func systemStaticPrefixKey() string {
	si := core.Settings.SystemInstructions
	roots := strings.Join(core.Folders, "|")
	return fmt.Sprintf("%s|%s|%s", roots, si, roleprompts.PhilosophyPrompt())
}

// buildSystemStaticPrefix 构建 system prompt 静态前缀（CACHE_BOUNDARY 之前）。
// 结果会被缓存，避免每次请求重复拼接字符串。
func buildSystemStaticPrefix() string {
	k := systemStaticPrefixKey()
	systemStaticPrefixCache.mu.Lock()
	defer systemStaticPrefixCache.mu.Unlock()
	if systemStaticPrefixCache.key == k && systemStaticPrefixCache.prefix != "" {
		return systemStaticPrefixCache.prefix
	}
	var b strings.Builder
	b.WriteString(agent.DefaultSystemPrompt(core.Folders)) // 工作区路径基本固定，放 system prompt 静态前缀可最大化 KV Cache 命中
	if si := strings.TrimSpace(core.Settings.SystemInstructions); si != "" {
		b.WriteString("\n\n# 系统级指令（务必遵守）\n" + si)
	}
	b.WriteString(roleprompts.PhilosophyPrompt())
	b.WriteString(agent.SelfManagementPrompt())
	// 注意：不在此追加 CacheBoundary——唯一 boundary 由 ComposeSystemPrompt 统一添加，
	// 确保 SystemInstructions/Philosophy 等静态内容全部位于 boundary 之前（同一静态前缀内）。
	systemStaticPrefixCache.key = k
	systemStaticPrefixCache.prefix = b.String()
	return systemStaticPrefixCache.prefix
}

// buildWebSystemDynamicCache 缓存 buildWebSystemDynamic 的输出（30s TTL）。
// skills 列表/知识库/项目环境等低频变化，缓存避免每次 Loop 重建重复扫描文件系统，
// 从而让 system 动态后缀在同一配置下保持稳定（减少 KV 缓存前缀断裂点漂移）。
var buildWebSystemDynamicCache struct {
	mu  sync.Mutex
	ts  time.Time
	val string
}

// buildWebSystemDynamic 构建 system prompt 动态后缀（CACHE_BOUNDARY 之后）。
// 包括 skills、项目知识库、项目环境等会话特定内容。
func buildWebSystemDynamic() string {
	buildWebSystemDynamicCache.mu.Lock()
	defer buildWebSystemDynamicCache.mu.Unlock()
	if buildWebSystemDynamicCache.val != "" && time.Since(buildWebSystemDynamicCache.ts) < 30*time.Second {
		return buildWebSystemDynamicCache.val
	}

	var b strings.Builder
	root := core.Root()
	b.WriteString(skills.Prompt())
	if core.Settings.LuaTools {
		b.WriteString(agent.LuaToolsPrompt())
	}
	b.WriteString(agent.LongTermMemoryPrompt())
	b.WriteString(agent.ProjectRules(root))
	b.WriteString(agent.ProjectKnowledge(root, 2500))

	// ★ 项目环境：遍历所有工作区根目录，分别读取各自的 .pair/project.md
	if len(core.Folders) > 0 {
		b.WriteString("\n\n# 项目环境")
		for i, f := range core.Folders {
			projName := filepath.Base(f)
			if i == 0 {
				b.WriteString(fmt.Sprintf("\n\n### %s（主项目）\n", projName))
			} else {
				b.WriteString(fmt.Sprintf("\n\n### %s\n", projName))
			}
			b.WriteString(fmt.Sprintf("> 路径: %s\n", f))
			projEnv := agent.ReadProjectEnv(f)
			if projEnv != "" {
				// 去除 project.md 自带的 # 项目环境档案 顶栏标题，避免与 ### 嵌套层级混乱
				lines := strings.SplitN(projEnv, "\n", 2)
				if len(lines) > 1 && strings.HasPrefix(strings.TrimSpace(lines[0]), "# ") {
					b.WriteString(strings.TrimSpace(lines[1]) + "\n")
				} else {
					b.WriteString(projEnv + "\n")
				}
			} else {
				b.WriteString("（无环境配置）\n")
			}
		}
	}
	// 时间戳已移至用户消息内（Loop.Run 中注入），保持系统提示词缓存前缀稳定。
	val := b.String()
	buildWebSystemDynamicCache.ts = time.Now()
	buildWebSystemDynamicCache.val = val
	return val
}

// buildWebSystemPrompt 构建完整系统提示词（桌面和 web 端共享）。
// 使用唯一的 CACHE_BOUNDARY 分隔静态前缀与动态后缀，最大化 LLM KV Cache 命中率。
// ★ 通过 ComposeSystemPrompt 统一添加 boundary，避免双边界/漏边界。
func buildWebSystemPrompt() string {
	return agent.ComposeSystemPrompt(buildSystemStaticPrefix(), buildWebSystemDynamic())
}

// buildWebProvider 构建 LLM Provider（桌面和 web 端共享）。
func buildWebProvider() agent.Provider {
	s := core.Settings
	if s.APIKey == "" || s.BaseURL == "" {
		return nil
	}
	if s.MaxTokens > 0 && s.MaxTokens < 8192 {
		log.Printf("[WARN] maxTokens=%d 过小（<8192），可能导致思考/回复被截断。建议在设置中调大至 ≥8192", s.MaxTokens)
	}
	return &agent.OpenAIProvider{
		BaseURL:      s.BaseURL,
		APIKey:       s.APIKey,
		Model:        core.MainModel(),
		Temperature:  core.Temperature(),
		MaxTokens:    s.MaxTokens,
		ThinkingMode: s.ThinkingMode,
	}
}

// countStates 统计具备指定状态的执行状态数量。
func countStates(states []*agent.ExecutionState, status agent.ExecStatus) int {
	n := 0
	for _, s := range states {
		if s.Status == status {
			n++
		}
	}
	return n
}

// reloadWebLuaTools 加载工作区 .pair/tools/*.lua 自定义工具。
func reloadWebLuaTools(reg *agent.Registry, root string) {
	if !core.Settings.LuaTools {
		return
	}
	agent.LoadLuaTools(reg, filepath.Join(root, ".pair", "tools"))
}

// buildWebLoopOpts 构建 agent.LoopOpts（统一版本，平台差异通过 webCompressor 回调）。
func (s *webServer) buildWebLoopOpts(convID, message string, autonomous bool) agent.LoopOpts {
	prov := buildWebProvider()

	root := core.Root()
	agent.WorkspaceRoots = core.Folders
	if root != "" {
		agent.SkillProjectDir = filepath.Join(root, ".pair", "skills")
	}
	if sysDir := filepath.Join(core.ConfigDir(), "skills"); sysDir != "" {
		agent.SkillSystemDir = sysDir
	}
	agent.SkillEnabled = core.Settings.SkillEnabledOverrides
	agent.SkillStatusOverride = core.Settings.SkillStatusOverrides
	// 初始化 MCP 配置路径（供 buildWebLoopOpts 内 mcppanel.LoadConfigs 使用）
	agent.MCPUserConfigPath = filepath.Join(core.ConfigDir(), "mcp.json")
	if root != "" {
		agent.MCPProjectConfigPath = filepath.Join(root, ".pair", "mcp.json")
	}
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	agent.RegisterCommitMessageTool(reg)

	agenttools.RegisterManagementTools(reg, root)
	if cfgs := mcppanel.LoadConfigs(); len(cfgs) > 0 {
		agentCfgs := make([]agent.MCPServerConfig, len(cfgs))
		for i, c := range cfgs {
			agentCfgs[i] = agent.MCPServerConfig{Name: c.Name, Command: c.Command, Args: c.Args, Env: c.Env}
		}
		agent.RegisterMCPServers(reg, agentCfgs)
	}

	// ★ 应用工作区工具配置（.pair/tools.json）
	// 必须在 if 块外无条件执行：即使 MCP 配置为空，工具开关/审核配置也要生效。
	agent.LoadWorkspaceToolConfig(reg, root)

	reloadWebLuaTools(reg, root)
	agent.SetCodeGraphDB(agentMgr.RawDB())
	agent.InitDebugLogger(root, 50)

	sys := buildWebSystemPrompt()

	// ★ 注入工具使用指南（引导 LLM 优先使用专用工具而非 run_command）
	if guide := reg.UsageGuideText(); guide != "" {
		sys += "\n\n" + guide
	}

	// ★ 保存注册表引用，供 /api/tools 查询工具列表与状态
	handler.SetToolsRegistry(reg)


	

	var history []agent.Message
	var summaries []string

	var sumErr error
	if convID != "" {
		if store := agentMgr.Store(); store != nil {
			raw, loadErr := store.LoadAll(convID)
			if loadErr != nil {
				log.Printf("[chat] LoadAll 失败 conv=%s err=%v", convID, loadErr)
			}
			if raw != nil {
				history = make([]agent.Message, len(raw))
				for i := range raw {
					history[i] = raw[i]
					// ★ 保留 Reasoning（思考链）不丢弃！DeepSeek 文档明确要求：
					//   进行工具调用的轮次，reasoning_content 在后续所有请求中必须回传。
					//   无工具调用的轮次，API 也会自动忽略 reasoning_content，保留无害。
					//   之前清空会导致 Agent 失去历史思考连贯性（变笨），
					//   且有工具调用的轮次缺失 reasoning_content 可能引发 API 400 错误。
				}
			}
			// 加载已持久化的压缩摘要（页面刷新后恢复上下文连续性）
			summaries, sumErr = store.LoadCompressedSummaries(convID)
			if sumErr != nil {
				log.Printf("[chat] LoadCompressedSummaries 失败 conv=%s err=%v", convID, sumErr)
			}
		}
	}
	history = agent.TrimInterruptedHistory(history)

	// ★ 保存原始（未压缩）历史，供持久化使用——压缩版只给 LLM 上下文，不写回历史记录
	originalHistory := make([]agent.Message, len(history))
	copy(originalHistory, history)

	// ★★★ 会话连贯性上下文：主动注入任务进度、对话摘要、相关记忆、项目归属、工作区结构 ★★★
	// 核心思想：主动注入 > 被动工具。Agent 不应依赖自己"想起来"去调用 memory_* 工具，
	// 而应在每次对话恢复时由系统主动注入上下文，确保 Agent 知道"在干什么、干过什么"。
	if resumeCtx := agent.BuildResumeContext(convID, message, history, agentMgr.Store(), core.Folders); resumeCtx != "" {
		// 注入为系统提示的动态后缀（在 CACHE_BOUNDARY 之后，不影响 KV Cache 前缀）
		sys += "\n\n" + resumeCtx
	}

	// ★ 历史精简：跨轮次加载时只保留最近一轮完整交互细节，
	//   旧轮次压缩为 [用户消息, 助手最终报告]，丢弃中间 tool 输出。
	//   大幅减少上下文体积，同时保持语义连续性。
	history = agent.CondenseHistory(history)






	maxIter := core.Settings.MaxIterations


	if autonomous {
		if maxIter <= 0 {
			maxIter = 60
		} else {
			maxIter *= 2
		}
	}

	return agent.LoopOpts{
		Provider:         prov,
		Registry:         reg,
		System:           sys,
		MaxIterations:    maxIter,
		MaxContextTokens: core.Settings.ContextMaxTokens,
		Compressor:       webCompressor(),
		History:          history,          // 压缩版：供 LLM 上下文使用
		HistoryOriginal:  originalHistory,  // 原始版：供持久化使用，防止压缩版写回历史记录
		CompressedSummaries: summaries,
		Autonomous:       autonomous,
	}
}

// handleChatSend 启动一次 agent 会话（非阻塞）。
func (s *webServer) handleChatSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Message       string `json:"message"`
		SessionID     string `json:"sessionId"`
		Autonomous    bool   `json:"autonomous"`
		ConvID        string `json:"convId"`
		WorkspaceRoot string `json:"workspaceRoot"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Message == "" {
		jsonErr(w, "消息不能为空")
		return
	}
	const maxMsgLen = 50000
	if len(req.Message) > maxMsgLen {
		req.Message = req.Message[:maxMsgLen] + "\n\n…（消息过长，已截断至 " + fmt.Sprint(maxMsgLen) + " 字符）"
	}
	if req.ConvID == "" {
		req.ConvID = fmt.Sprintf("conv_%d", time.Now().UnixNano())
	}
	if req.WorkspaceRoot == "" {
		req.WorkspaceRoot = core.Root()
	}

	if !core.Configured() {
		jsonErr(w, "未配置 API key。请在设置面板中配置 API Key 和模型。")
		return
	}

	if err := agentMgr.AppendPersistedUserMessage(req.ConvID, req.Message); err != nil {
		log.Printf("[chat] AppendPersistedUserMessage 失败 conv=%s err=%v", req.ConvID, err)
		jsonErr(w, "写入用户消息失败: "+err.Error())
		return
	}
	// 记录当前消息索引，后续文件编辑快照关联到此消息
	if store := agentMgr.Store(); store != nil {
		if count, err := store.Count(req.ConvID); err == nil && count > 0 {
			if tr := agent.GetTracker(); tr != nil {
				tr.SetCurrentMsg(req.ConvID, count-1)
			}
		}
	}
	opts := s.buildWebLoopOpts(req.ConvID, req.Message, req.Autonomous)
	opts.WorkspaceRoot = req.WorkspaceRoot

	// 先从全局设置取审核配置（默认值）
	opts.ReviewMode = core.Settings.ReviewMode
	opts.ReviewBlacklist = core.Settings.ReviewBlacklist
	opts.ReviewWhitelist = core.Settings.ReviewWhitelist
	// ★ 如果请求中指定了工作区根路径，从工作区配置覆盖审核配置
	if req.WorkspaceRoot != "" {
		wrMode, wrBlack, wrWhite := agent.LoadWorkspaceReviewConfig(req.WorkspaceRoot)
		if wrMode != "" && wrMode != "auto" {
			opts.ReviewMode = wrMode
		}
		if wrBlack != nil {
			opts.ReviewBlacklist = wrBlack
		}
		if wrWhite != nil {
			opts.ReviewWhitelist = wrWhite
		}
	}
	if core.Settings.ReviewMode == "auto" && core.Settings.ReviewModel != "" {
		pm := strings.TrimSpace(core.Settings.PlanModel)
		base := strings.TrimSpace(core.Settings.BaseURL)
		key := strings.TrimSpace(core.Settings.APIKey)
		if pm != "" && base != "" && key != "" {
			opts.ReviewProvider = &agent.OpenAIProvider{BaseURL: base, APIKey: key, Model: pm, Temperature: -1, ThinkingMode: "non-thinking"}
		}
	}
	opts.AutoCommit = core.Settings.AutoCommit

	if req.Autonomous {
		pm := strings.TrimSpace(core.Settings.PlanModel)
		if pm != "" && core.Settings.BaseURL != "" && core.Settings.APIKey != "" {
			opts.PlanProvider = &agent.OpenAIProvider{
				BaseURL: core.Settings.BaseURL, APIKey: core.Settings.APIKey,
				Model: pm, Temperature: core.Temperature(), MaxTokens: core.Settings.MaxTokens,
				ThinkingMode: core.Settings.ThinkingMode,
			}
		} else if prov := buildWebProvider(); prov != nil {
			opts.PlanProvider = prov
		}
	}

	// 使用分离的 context：setupCtx 用于 Start 方法本身的超时（避免在获取锁或建表时永久阻塞），
	// Loop 的运行由内部独立的 context 管理（Stop 可取消），不受此超时影响。
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer setupCancel()
	if err := agentMgr.Start(setupCtx, req.ConvID, req.Message, opts); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "convId": req.ConvID})
}

// handleChatStop 停止指定会话。
func (s *webServer) handleChatStop(w http.ResponseWriter, r *http.Request) {
	convID := r.URL.Query().Get("convId")
	if convID == "" {
		convID = r.URL.Query().Get("sessionId")
	}
	if convID == "" {
		jsonErr(w, "缺少 convId 参数")
		return
	}
	agentMgr.Stop(convID)
	jsonResp(w, map[string]any{"ok": true})
}

func (s *webServer) handleChatCompact(w http.ResponseWriter, r *http.Request) {
	convID := r.URL.Query().Get("convId")
	if convID == "" {
		jsonErr(w, "缺少 convId 参数")
		return
	}
	agentMgr.Compact(convID)
	jsonResp(w, map[string]any{"ok": true})
}

// handleChatAnswer 发送 ask_user 的用户回答。
func (s *webServer) handleChatAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		ConvID string `json:"convId"`
		Answer string `json:"answer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.ConvID == "" {
		jsonErr(w, "convId 必填")
		return
	}
	if err := agentMgr.SendAnswer(req.ConvID, req.Answer); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

// handleChatApprove 发送审批结果。
func (s *webServer) handleChatApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		ConvID   string `json:"convId"`
		Approved bool   `json:"approved"`
		Reply    string `json:"reply"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.ConvID == "" {
		jsonErr(w, "convId 必填")
		return
	}
	if err := agentMgr.Approve(req.ConvID, req.Approved, req.Reply); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

// handleChatFeedback 发送运行时反馈。
func (s *webServer) handleChatFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		ConvID   string `json:"convId"`
		Feedback string `json:"feedback"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.ConvID == "" {
		jsonErr(w, "convId 必填")
		return
	}
	if err := agentMgr.SendFeedback(req.ConvID, req.Feedback); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

// handleChatRollback POST /api/chat/rollback 回滚到指定用户消息前的状态。
// 恢复该消息关联的所有文件快照，并删除该消息之后的对话历史。
// 请求体: { convId, msgIdx }
// convId 为对话 ID，msgIdx 为用户消息索引（0 基）。
func (s *webServer) handleChatRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		ConvID string `json:"convId"`
		MsgIdx int    `json:"msgIdx"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.ConvID == "" {
		jsonErr(w, "convId 必填")
		return
	}
	root := core.Root()
	if root == "" {
		jsonErr(w, "工作区未设置")
		return
	}
	var store agent.ConversationStore
	if agentMgr != nil {
		store = agentMgr.Store()
	}
	if err := agent.RollbackToMsg(root, req.ConvID, req.MsgIdx, store); err != nil {
		jsonErr(w, err.Error())
		return
	}
	// 停止正在运行的 agent 会话（如果有）
	agentMgr.Stop(req.ConvID)
	jsonResp(w, map[string]any{"ok": true, "msgIdx": req.MsgIdx})
}

// startEventPersistWorker 设置 OnDone 回调（compressor 由 webCompressor 回调提供）。
func (s *webServer) startEventPersistWorker() {
	agentMgr.OnDone = func(convID string) {
		go generateConversationSummary(convID, webCompressor())
	}
}

// ─── 市场搜索 API ──────────────────────────────────────────

func (s *webServer) handleMarketplaceSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "all"
	}
	results := marketplacepanel.Search(query, kind)
	type resultItem struct {
		ID          string   `json:"id"`
		Kind        string   `json:"kind"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		Command     string   `json:"command"`
		Args        []string `json:"args"`
		Installed   bool     `json:"installed"`
	}
	out := make([]resultItem, 0, len(results))
	for _, e := range results {
		out = append(out, resultItem{
			ID: e.ID, Kind: e.Kind, Name: e.Name,
			Description: e.Description, Tags: e.Tags,
			Command: e.Command, Args: e.Args,
			Installed: marketplacepanel.IsInstalled(e.ID),
		})
	}
	jsonResp(w, out)
}

func (s *webServer) handleMarketplaceInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		ID      string   `json:"id"`
		Kind    string   `json:"kind"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Scope   string   `json:"scope"` // "user" 或 "project"，默认 "user"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.ID == "" {
		jsonErr(w, "id 必填")
		return
	}
	if req.Scope == "" {
		req.Scope = "user"
	}
	var msg string
	var err error
	if req.Command != "" {
		entry := marketplacepanel.RegistryEntry{
			ID: req.ID, Kind: req.Kind,
			Command: req.Command, Args: req.Args,
		}
		msg, err = marketplacepanel.InstallEntry(entry, false, req.Scope)
	} else {
		msg, err = marketplacepanel.InstallScoped(req.ID, false, req.Scope)
	}
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "message": msg})
}

func (s *webServer) handleMarketplaceRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	root := core.Root()
	if err := marketplacepanel.FetchRemoteRegistry(root, true); err != nil {
		jsonResp(w, map[string]any{"ok": false, "message": err.Error(), "status": marketplacepanel.FetchStatus()})
		return
	}
	jsonResp(w, map[string]any{"ok": true, "message": "远程市场已刷新", "status": marketplacepanel.FetchStatus()})
}

// ── 对话摘要生成 ──────────────────────────────────────────

// generateConversationSummary 用规则式生成对话摘要并保存到 MessageStore。
// 在对话完全结束后调用（goroutine 内异步）。prov 保留兼容但当前未使用（规则式摘要）。
func generateConversationSummary(convID string, prov agent.Provider) {
	if convID == "" {
		return
	}
	_ = prov

	store := agentMgr.Store()
	if store == nil {
		return
	}

	meta, err := store.GetConversation(convID)
	if err != nil || meta == nil {
		return
	}

	// 已有摘要 → 跳过（避免重复生成）
	if meta.Summary != "" {
		return
	}

	msgs, err := store.LoadAll(convID)
	if err != nil || len(msgs) == 0 {
		return
	}

	// 使用 pkg/summary 生成摘要
	summaryMsgs := make([]summary.Message, len(msgs))
	for i, m := range msgs {
		summaryMsgs[i] = summary.Message{Role: string(m.Role), Content: m.Content}
	}
	s := summary.Generate(summary.ConvInfo{
		ID: meta.ID, Title: meta.Title,
		CreatedAt: meta.CreatedAt, UpdatedAt: meta.UpdatedAt,
		Messages: summaryMsgs,
	})
	if s == "" {
		return
	}
	summaryAt := time.Now().Format("2006-01-02 15:04:05")
	store.SetSummary(convID, s, summaryAt)

	// 同步写入记忆索引
	assistantMsgs := make([]string, 0)
	allMsgs := make([]string, len(msgs))
	for i, m := range msgs {
		allMsgs[i] = m.Content
		if m.Role == agent.RoleAssistant {
			assistantMsgs = append(assistantMsgs, m.Content)
		}
	}
	memory.Upsert(memory.Entry{
		ID:           meta.ID,
		Title:        meta.Title,
		Summary:      s,
		CreatedAt:    meta.CreatedAt,
		UpdatedAt:    meta.UpdatedAt,
		MessageCount: len(msgs),
		Tags:         memory.ExtractTags(allMsgs),
		KeyPoints:    memory.ExtractKeyPoints(assistantMsgs),
		CompletedAt:  summaryAt,
	})
}

// ─── 记忆索引 API（委托 pkg/memory）─────────────────────────

func (s *webServer) handleMemorySearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results := memory.Search(q)
	jsonResp(w, map[string]any{"results": results})
}

func (s *webServer) handleMemoryList(w http.ResponseWriter, r *http.Request) {
	memories := memory.List()
	jsonResp(w, map[string]any{"memories": memories})
}

func (s *webServer) handleMemoryRebuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	store := agentMgr.Store()
	if store == nil {
		jsonErr(w, "消息存储未初始化")
		return
	}
	metas, err := store.ListConversations(core.Root())
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	count := 0
	for _, meta := range metas {
		msgs, err := store.LoadAll(meta.ID)
		if err != nil || len(msgs) < 2 {
			continue
		}
		summaryStr := meta.Summary
		if summaryStr == "" {
			summaryMsgs := make([]summary.Message, len(msgs))
			for j, m := range msgs {
				summaryMsgs[j] = summary.Message{Role: string(m.Role), Content: m.Content}
			}
			summaryStr = summary.Generate(summary.ConvInfo{ID: meta.ID, Title: meta.Title, Messages: summaryMsgs})
		}
		assistantMsgs := make([]string, 0)
		allMsgs := make([]string, len(msgs))
		for j, m := range msgs {
			allMsgs[j] = m.Content
			if m.Role == agent.RoleAssistant {
				assistantMsgs = append(assistantMsgs, m.Content)
			}
		}
		memory.Upsert(memory.Entry{
			ID: meta.ID, Title: meta.Title, Summary: summaryStr,
			CreatedAt: meta.CreatedAt, UpdatedAt: meta.UpdatedAt,
			MessageCount: len(msgs),
			Tags:         memory.ExtractTags(allMsgs),
			KeyPoints:    memory.ExtractKeyPoints(assistantMsgs),
			CompletedAt:  meta.SummaryAt,
		})
		count++
	}
	jsonResp(w, map[string]any{"status": "ok", "count": count})
}

// ─── Git API 处理函数 ─────────────────────────────────────────

// gitDirCtxKey 用于在 request context 中传递 git 目录覆盖（支持 ?path= 参数）。
type gitDirCtxKey struct{}

// withGitDir 从请求查询参数中读取 path，并注入到 request context。
// 若没有 ?path= 参数，返回原 request，不创建新 context。
func withGitDir(r *http.Request) *http.Request {
	if p := r.URL.Query().Get("path"); p != "" {
		ctx := context.WithValue(r.Context(), gitDirCtxKey{}, p)
		return r.WithContext(ctx)
	}
	return r
}

// gitRoot 返回 git 工作目录。若 context 中有 path 覆盖则优先使用。
func gitRoot(ctx context.Context) string {
	if ctx != nil {
		if dir, ok := ctx.Value(gitDirCtxKey{}).(string); ok && dir != "" {
			return dir
		}
	}
	if root := core.Root(); root != "" {
		return root
	}
	cwd, _ := os.Getwd()
	return cwd
}

// runGitInternal 执行 git 命令，返回标准输出。
func runGitInternal(ctx context.Context, args ...string) (string, error) {
	dir := gitRoot(ctx)
	if dir == "" {
		return "", fmt.Errorf("未设置工作区")
	}
	fullArgs := append([]string{"-C", dir, "-c", "core.quotepath=false"}, args...)
	cmd := exec.CommandContext(ctx, "git", fullArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		if out != "" {
			return out, nil
		}
		return "", fmt.Errorf("%s", errMsg)
	}
	return out, nil
}

// gitStatusResult Git 状态返回结构。
type gitStatusResult struct {
	Branch    string           `json:"branch"`
	Ahead     int              `json:"ahead"`
	Behind    int              `json:"behind"`
	IsRepo    bool             `json:"isRepo"`
	Staged    []gitChangeEntry `json:"staged"`
	Conflict  []gitChangeEntry `json:"conflict"`
	Modified  []gitChangeEntry `json:"modified"`
	Untracked []gitChangeEntry `json:"untracked"`
	Branches  []string         `json:"branches"`
	Error     string           `json:"error,omitempty"`
}

type gitChangeEntry struct {
	Path string `json:"path"`
	X    string `json:"x"`
	Y    string `json:"y"`
}

// handleGitStatus GET /api/git/status 获取完整的 Git 状态。
func (s *webServer) handleGitStatus(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	res := gitStatusResult{}

	out, err := runGitInternal(r.Context(), "rev-parse", "--is-inside-work-tree")
	if err != nil || out != "true" {
		res.Error = "非 Git 仓库或未设置工作区"
		jsonResp(w, res)
		return
	}
	res.IsRepo = true

	if b, err := runGitInternal(r.Context(), "branch", "--show-current"); err == nil {
		res.Branch = b
	}

	if ab, err := runGitInternal(r.Context(), "rev-list", "--left-right", "--count", "HEAD...@{upstream}"); err == nil {
		fmt.Sscanf(ab, "%d\t%d", &res.Ahead, &res.Behind)
	}

	statusOut, err := runGitInternal(r.Context(), "status", "--porcelain")
	if err == nil {
		for _, line := range strings.Split(statusOut, "\n") {
			if len(line) < 4 {
				continue
			}
			x := string(line[0])
			y := string(line[1])
			path := strings.TrimSpace(line[3:])
			if i := strings.Index(path, " -> "); i >= 0 {
				path = path[i+4:]
			}
			e := gitChangeEntry{Path: path, X: x, Y: y}
			switch {
			case x == "?" && y == "?":
				res.Untracked = append(res.Untracked, e)
			case x == "U" || y == "U" || (x == "D" && y == "D") || (x == "A" && y == "A"):
				res.Conflict = append(res.Conflict, e)
			default:
				if x != " " && x != "?" {
					res.Staged = append(res.Staged, e)
				}
				if y != " " && y != "?" {
					res.Modified = append(res.Modified, e)
				}
			}
		}
	}

	if branches, err := runGitInternal(r.Context(), "branch", "--format=%(refname:short)"); err == nil {
		for _, b := range strings.Split(branches, "\n") {
			if b = strings.TrimSpace(b); b != "" {
				res.Branches = append(res.Branches, b)
			}
		}
	}

	jsonResp(w, res)
}

// handleGitInit POST /api/git/init 初始化 Git 仓库（在指定目录下执行 git init）。
func (s *webServer) handleGitInit(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	out, err := runGitInternal(r.Context(), "init")
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"output": out})
}

// handleGitDiff GET /api/git/diff?file=xxx&staged=true 查看文件差异。
func (s *webServer) handleGitDiff(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	file := r.URL.Query().Get("file")
	staged := r.URL.Query().Get("staged") == "true"

	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	if file != "" {
		args = append(args, "--", file)
	}
	out, err := runGitInternal(r.Context(), args...)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	if out == "" {
		out = "（无改动）"
	}
	jsonResp(w, map[string]any{"diff": out})
}

// handleGitAdd POST /api/git/add 暂存文件。
func (s *webServer) handleGitAdd(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Files []string `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	args := []string{"add"}
	if len(req.Files) > 0 {
		args = append(append(args, "--"), req.Files...)
	} else {
		args = append(args, "-A")
	}
	_, err := runGitInternal(r.Context(), args...)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

// handleGitReset POST /api/git/reset 取消暂存文件。
func (s *webServer) handleGitReset(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Files []string `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	args := []string{"reset", "-q", "HEAD", "--"}
	if len(req.Files) > 0 {
		args = append(args, req.Files...)
	} else {
		args = append(args, ".")
	}
	_, err := runGitInternal(r.Context(), args...)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

// handleGitCommit POST /api/git/commit 提交暂存区。
func (s *webServer) handleGitCommit(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Message string `json:"message"`
		All     bool   `json:"all"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		jsonErr(w, "提交信息不能为空")
		return
	}
	args := []string{"commit", "-m", req.Message}
	if req.All {
		args = append(args, "-a")
	}
	out, err := runGitInternal(r.Context(), args...)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "output": out})
}

// handleGitLog GET /api/git/log?count=15&file=xxx 查看提交历史。
func (s *webServer) handleGitLog(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	count := r.URL.Query().Get("count")
	file := r.URL.Query().Get("file")
	if count == "" {
		count = "50"
	}

	type commitEntry struct {
		Hash   string `json:"hash"`
		Short  string `json:"short"`
		Author string `json:"author"`
		Date   string `json:"date"`
		Msg    string `json:"msg"`
	}

	args := []string{"log", "--max-count=" + count, "--pretty=format:%H|%h|%an|%ai|%B"}
	if file != "" {
		args = append(args, "--", file)
	}
	out, err := runGitInternal(r.Context(), args...)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	var commits []commitEntry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) >= 5 {
			commits = append(commits, commitEntry{
				Hash: parts[0], Short: parts[1],
				Author: parts[2], Date: parts[3], Msg: parts[4],
			})
		}
	}
	jsonResp(w, commits)
}

// handleGitBranch POST /api/git/branch 分支操作。
func (s *webServer) handleGitBranch(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Action  string `json:"action"`
		Name    string `json:"name"`
		NewName string `json:"newName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}

	switch req.Action {
	case "list":
		out, err := runGitInternal(r.Context(), "branch", "--all", "--format=%(refname:short)")
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		branches := strings.Split(out, "\n")
		jsonResp(w, branches)
	case "create":
		if req.Name == "" {
			jsonErr(w, "分支名不能为空")
			return
		}
		_, err := runGitInternal(r.Context(), "branch", req.Name)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true})
	case "delete":
		if req.Name == "" {
			jsonErr(w, "分支名不能为空")
			return
		}
		_, err := runGitInternal(r.Context(), "branch", "-D", req.Name)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true})
	case "switch":
		if req.Name == "" {
			jsonErr(w, "分支名不能为空")
			return
		}
		_, err := runGitInternal(r.Context(), "checkout", req.Name)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true})
	case "create-switch":
		if req.Name == "" {
			jsonErr(w, "分支名不能为空")
			return
		}
		_, err := runGitInternal(r.Context(), "checkout", "-b", req.Name)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true})
	case "rename":
		if req.Name == "" || req.NewName == "" {
			jsonErr(w, "name 和 newName 不能为空")
			return
		}
		_, err := runGitInternal(r.Context(), "branch", "-m", req.Name, req.NewName)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true})
	default:
		jsonErr(w, "未知操作: "+req.Action)
	}
}

// handleGitCheckout POST /api/git/checkout 切换分支或恢复文件。
func (s *webServer) handleGitCheckout(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Target string `json:"target"`
		IsFile bool   `json:"isFile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Target == "" {
		jsonErr(w, "target 不能为空")
		return
	}
	var out string
	var err error
	if req.IsFile {
		out, err = runGitInternal(r.Context(), "checkout", "--", req.Target)
	} else {
		out, err = runGitInternal(r.Context(), "checkout", req.Target)
	}
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "output": out})
}

// handleGitStash POST /api/git/stash 暂存操作（push/pop/drop/apply）。
func (s *webServer) handleGitStash(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Action  string `json:"action"`
		Message string `json:"message"`
		Index   string `json:"index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Action == "" {
		req.Action = "push"
	}

	var out string
	var err error
	switch req.Action {
	case "push":
		args := []string{"stash", "push"}
		if strings.TrimSpace(req.Message) != "" {
			args = append(args, "-m", req.Message)
		}
		out, err = runGitInternal(r.Context(), args...)
	case "pop":
		args := []string{"stash", "pop"}
		if req.Index != "" {
			args = append(args, req.Index)
		}
		out, err = runGitInternal(r.Context(), args...)
	case "apply":
		args := []string{"stash", "apply"}
		if req.Index != "" {
			args = append(args, req.Index)
		}
		out, err = runGitInternal(r.Context(), args...)
	case "drop":
		args := []string{"stash", "drop"}
		if req.Index != "" {
			args = append(args, req.Index)
		}
		out, err = runGitInternal(r.Context(), args...)
	default:
		jsonErr(w, "未知 stash 操作: "+req.Action)
		return
	}
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "output": out})
}

// handleGitStashList GET /api/git/stash-list 列出 stash 列表。
func (s *webServer) handleGitStashList(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	out, err := runGitInternal(r.Context(), "stash", "list", "--format=%(stash): %(stashsubject) | %(stashdate:short)")
	if err != nil {
		jsonResp(w, []any{})
		return
	}
	type stashEntry struct {
		Index string `json:"index"`
		Ref   string `json:"ref"`
		Msg   string `json:"msg"`
		Date  string `json:"date"`
	}
	var stashes []stashEntry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) < 2 {
			continue
		}
		ref := parts[0]
		rest := parts[1]
		detail := strings.SplitN(rest, " | ", 2)
		msg := rest
		date := ""
		if len(detail) == 2 {
			msg = detail[0]
			date = detail[1]
		}
		stashes = append(stashes, stashEntry{
			Index: ref, Ref: ref, Msg: msg, Date: date,
		})
	}
	jsonResp(w, stashes)
}

// handleGitIgnore GET/POST /api/git/ignore 管理 .gitignore。
func (s *webServer) handleGitIgnore(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	root := gitRoot(r.Context())
	ignorePath := filepath.Join(root, ".gitignore")

	switch r.Method {
	case "GET":
		data, err := os.ReadFile(ignorePath)
		if err != nil {
			jsonResp(w, map[string]any{"content": "", "rules": []string{}})
			return
		}
		content := string(data)
		rules := strings.Split(content, "\n")
		jsonResp(w, map[string]any{"content": content, "rules": rules})

	case "POST":
		var req struct {
			Content string `json:"content"`
			Append  string `json:"append"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error())
			return
		}
		if req.Append != "" {
			f, err := os.OpenFile(ignorePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			defer f.Close()
			if _, err := f.WriteString("\n" + req.Append + "\n"); err != nil {
				jsonErr(w, err.Error())
				return
			}
			jsonResp(w, map[string]any{"ok": true, "appended": req.Append})
		} else {
			if err := os.WriteFile(ignorePath, []byte(req.Content), 0644); err != nil {
				jsonErr(w, err.Error())
				return
			}
			jsonResp(w, map[string]any{"ok": true})
		}

	default:
		jsonErr(w, "不支持的方法")
	}
}

// handleGitDiscard POST /api/git/discard 丢弃工作区更改。
func (s *webServer) handleGitDiscard(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Files []string `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	args := []string{"checkout", "--"}
	if len(req.Files) > 0 {
		args = append(args, req.Files...)
	} else {
		jsonErr(w, "请指定要丢弃的文件")
		return
	}
	_, err := runGitInternal(r.Context(), args...)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

// handleGitPush POST /api/git/push 推送。
func (s *webServer) handleGitPush(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Remote string `json:"remote"`
		Branch string `json:"branch"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	args := []string{"push"}
	if req.Remote != "" {
		args = append(args, req.Remote)
	}
	if req.Branch != "" {
		args = append(args, req.Branch)
	}
	out, err := runGitInternal(r.Context(), args...)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "output": out})
}

// handleGitPull POST /api/git/pull 拉取。
func (s *webServer) handleGitPull(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Remote string `json:"remote"`
		Branch string `json:"branch"`
		Rebase bool   `json:"rebase"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	args := []string{"pull"}
	if req.Rebase {
		args = append(args, "--rebase")
	} else {
		args = append(args, "--ff-only")
	}
	if req.Remote != "" {
		args = append(args, req.Remote)
	}
	if req.Branch != "" {
		args = append(args, req.Branch)
	}
	out, err := runGitInternal(r.Context(), args...)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "output": out})
}

// handleGitRemote GET/POST /api/git/remote 远程仓库管理。
func (s *webServer) handleGitRemote(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	switch r.Method {
	case "GET":
		type remoteInfo struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}
		out, err := runGitInternal(r.Context(), "remote", "-v")
		if err != nil {
			jsonResp(w, []remoteInfo{})
			return
		}
		var remotes []remoteInfo
		seen := map[string]bool{}
		for _, line := range strings.Split(out, "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[0]
				url := parts[1]
				if !seen[name] {
					remotes = append(remotes, remoteInfo{Name: name, URL: url})
					seen[name] = true
				}
			}
		}
		jsonResp(w, remotes)

	case "POST":
		var req struct {
			Action string `json:"action"`
			Name   string `json:"name"`
			URL    string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error())
			return
		}
		switch req.Action {
		case "add":
			if req.Name == "" || req.URL == "" {
				jsonErr(w, "name 和 url 不能为空")
				return
			}
			_, err := runGitInternal(r.Context(), "remote", "add", req.Name, req.URL)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			jsonResp(w, map[string]any{"ok": true})
		case "remove":
			if req.Name == "" {
				jsonErr(w, "name 不能为空")
				return
			}
			_, err := runGitInternal(r.Context(), "remote", "remove", req.Name)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			jsonResp(w, map[string]any{"ok": true})
		case "set-url":
			if req.Name == "" || req.URL == "" {
				jsonErr(w, "name 和 url 不能为空")
				return
			}
			_, err := runGitInternal(r.Context(), "remote", "set-url", req.Name, req.URL)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			jsonResp(w, map[string]any{"ok": true})
		default:
			jsonErr(w, "未知 remote 操作: "+req.Action)
		}

	default:
		jsonErr(w, "不支持的方法")
	}
}

// ─── Debug 日志 API ──────────────────────────────────────────

func (s *webServer) handleDebugLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, "仅 GET")
		return
	}
	if agent.GlobalDebugLogger == nil {
		jsonResp(w, map[string]any{"logs": []any{}, "counts": map[string]int{}})
		return
	}

	level := r.URL.Query().Get("level")
	source := r.URL.Query().Get("source")
	limitStr := r.URL.Query().Get("limit")
	limit := 0
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	var logs []*agent.LogEntry
	if level != "" || source != "" {
		logs = agent.GlobalDebugLogger.FilterLogs(agent.LogLevel(level), source, limit)
	} else {
		logs = agent.GlobalDebugLogger.ListLogs(limit)
	}

	counts := agent.GlobalDebugLogger.CountLogs()

	jsonResp(w, map[string]any{
		"logs":   logs,
		"counts": counts,
	})
}

func (s *webServer) handleDebugLogByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, "仅 GET")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/debug/logs/")
	if id == "" {
		jsonErr(w, "id 必填")
		return
	}
	if agent.GlobalDebugLogger == nil {
		jsonErr(w, "调试日志未初始化")
		return
	}
	entry := agent.GlobalDebugLogger.GetLog(id)
	if entry == nil {
		jsonErr(w, "日志不存在: "+id)
		return
	}
	jsonResp(w, entry)
}