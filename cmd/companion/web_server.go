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
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
	"github.com/hoonfeng/paircode/cmd/companion/core"
	mcppanel "github.com/hoonfeng/paircode/cmd/companion/ui/mcp"
	"github.com/hoonfeng/paircode/pkg/memory"
	"github.com/hoonfeng/paircode/pkg/summary"
)

// ─── 内嵌前端资源 ─────────────────────────────────────────────

//go:embed web-ui/dist
var webUIFiles embed.FS

// webServer 是运行在 companion 内部的 HTTP 服务器。
type webServer struct {
	server *http.Server
	port   int
	mu     sync.Mutex
	// 当前 agent 运行状态（多会话支持）
	activeLoops map[string]*webAgentSession
	// historyCache 持久化对话历史缓存（convID → 完整 []Message）。
	// 替代 loadConversationHistory + BuildHistory 模式。
	// loop.Run 返回的完整 msgs 缓存于此，下次同 convID 请求直接复用（传 nil history，loop.Run 自动使用 l.History）。
	historyCache map[string][]agent.Message
}

// webAgentSession 一次 agent 运行会话。
type webAgentSession struct {
	cancel  context.CancelFunc
	events  chan agent.Event
	stopped bool

	// ask_user 问答通道：Handler 阻塞等用户回答
	askCh   chan string
	askDone bool // 标记回答已收到，防重复写入

	// 审批通道：Approve 阻塞等用户裁决
	approvalCh     chan bool
	approvalCallID string

	// loop 持久化的 Agent 引擎（跨 HTTP 请求复用，保持 l.History 自闭环）。
	// 同一 convID 的后续消息复用此 loop，前端只需发信号（task text），
	// loop 内部通过 l.History 自动管理完整对话历史。
	loop   *agent.Loop
	convID string // 此 loop 绑定的对话 ID
}

var ws *webServer

// startWebUI 在后台启动 Web UI 服务器。
// historyCachePath 返回 historyCache 持久化文件路径。
func historyCachePath() string {
	root := core.Root()
	if root == "" {
		return ""
	}
	return filepath.Join(root, ".pair", "history_cache.json")
}

// saveHistoryCache 将 historyCache 持久化到磁盘。
func (s *webServer) saveHistoryCache() {
	path := historyCachePath()
	if path == "" {
		return
	}
	s.mu.Lock()
	data, err := json.MarshalIndent(s.historyCache, "", "  ")
	s.mu.Unlock()
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	_ = os.WriteFile(path, data, 0644)
}

// loadHistoryCache 从磁盘加载 historyCache。
func (s *webServer) loadHistoryCache() {
	path := historyCachePath()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return // 文件不存在或读取失败 → 保持空 map
	}
	var cached map[string][]agent.Message
	if err := json.Unmarshal(data, &cached); err != nil {
		return
	}
	s.mu.Lock()
	for k, v := range cached {
		s.historyCache[k] = v
	}
	s.mu.Unlock()
}

func startWebUI(port int) {
	if ws != nil {
		return
	}
	ws = &webServer{
		port:         port,
		activeLoops:  make(map[string]*webAgentSession),
		historyCache: make(map[string][]agent.Message),
	}
	ws.loadHistoryCache() // 从磁盘恢复历史缓存
	memory.SetRoot(core.Root())
	// 工作区文件夹变更时同步到 agent 路径解析
	core.OnSyncWorkspace = func(primaryChanged bool) {
		agent.WorkspaceRoots = core.Folders
		log.Printf("[WebUI] 工作区文件夹已更新: %v", core.Folders)
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
	mux.HandleFunc("/api/tokens/stats", ws.handleTokensStats)
	mux.HandleFunc("/api/debug/logs", ws.handleDebugLogs)
	mux.HandleFunc("/api/debug/logs/", ws.handleDebugLogByID)

	// ── Git API 路由 ──
	mux.HandleFunc("/api/git/status", ws.handleGitStatus)
	mux.HandleFunc("/api/git/diff", ws.handleGitDiff)
	mux.HandleFunc("/api/git/add", ws.handleGitAdd)
	mux.HandleFunc("/api/git/reset", ws.handleGitReset)
	mux.HandleFunc("/api/git/commit", ws.handleGitCommit)
	mux.HandleFunc("/api/git/log", ws.handleGitLog)
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

	// ── 静态文件 ──
	subFS, err := fs.Sub(webUIFiles, "web-ui/dist")
	if err != nil {
		log.Printf("[WebUI] 内嵌资源加载失败: %v", err)
		return
	}
	fileServer := http.FileServer(http.FS(subFS))
	mux.Handle("/", fileServer)

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
			Action    string   `json:"action"`
			Root      string   `json:"root"`
			Folders   []string `json:"folders"`
			Name      string   `json:"name"`
			Path      string   `json:"path"`
			ParentDir string   `json:"parentDir"`
			Lang      string   `json:"lang"`
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
		var newSettings core.AppSettings
		if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
			jsonErr(w, err.Error())
			return
		}
		core.Settings = newSettings
		core.Save()
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
		"goos":      "windows",
		"workspace": core.Root(),
		"folders":   core.Folders,
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
	jsonResp(w, map[string]any{"tasks": []any{}, "message": "任务管理使用方式：运行 go run ./cmd/guitest/"})
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

// Conversation 对话。
// Conversation 一次对话会话。
// Summary 字段在对话结束后由 AI 生成，用于跨对话的上下文记忆注入。
type Conversation struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Messages  []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages,omitempty"`
	Summary   string `json:"summary,omitempty"`   // AI 生成的对话摘要（对话结束后由 LLM 总结）
	SummaryAt string `json:"summaryAt,omitempty"` // 摘要生成时间（空=未生成）
}

var (
	conversationsMu sync.Mutex
	conversations   []Conversation
)

func conversationsPath() string {
	root := core.Root()
	if root == "" {
		return ""
	}
	pairDir := filepath.Join(root, ".pair")
	os.MkdirAll(pairDir, 0755)
	return filepath.Join(pairDir, "conversations.json")
}

func loadConversations() {
	path := conversationsPath()
	if path == "" {
		conversations = []Conversation{}
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		conversations = []Conversation{}
		return
	}
	json.Unmarshal(data, &conversations)
	if conversations == nil {
		conversations = []Conversation{}
	}
}

func saveConversations() {
	path := conversationsPath()
	if path == "" {
		return
	}
	data, _ := json.MarshalIndent(conversations, "", "  ")
	os.WriteFile(path, data, 0644)
}

func (s *webServer) handleConversations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		loadConversations()
		type convBrief struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			CreatedAt string `json:"createdAt"`
			UpdatedAt string `json:"updatedAt"`
			MsgCount  int    `json:"msgCount"`
		}
		brief := make([]convBrief, 0, len(conversations))
		for _, c := range conversations {
			brief = append(brief, convBrief{
				ID: c.ID, Title: c.Title, CreatedAt: c.CreatedAt,
				UpdatedAt: c.UpdatedAt, MsgCount: len(c.Messages),
			})
		}
		jsonResp(w, brief)

	case "POST":
		var req struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error())
			return
		}
		loadConversations()
		conv := Conversation{
			ID:        fmt.Sprintf("conv_%d", time.Now().UnixNano()),
			Title:     req.Title,
			CreatedAt: time.Now().Format("2006-01-02T15:04:05"),
			UpdatedAt: time.Now().Format("2006-01-02T15:04:05"),
			Messages: []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{},
		}
		if conv.Title == "" {
			conv.Title = "新对话 " + time.Now().Format("15:04")
		}
		conversations = append(conversations, conv)
		saveConversations()
		jsonResp(w, conv)

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
	wantMessages := len(parts) >= 2 && parts[1] == "messages"
	wantTokenStats := len(parts) >= 2 && parts[1] == "token-stats"

	loadConversations()
	var conv *Conversation
	for i := range conversations {
		if conversations[i].ID == id {
			conv = &conversations[i]
			break
		}
	}
	if conv == nil {
		jsonErr(w, "对话不存在")
		return
	}

	switch r.Method {
	case "GET":
		if wantTokenStats {
			jsonResp(w, calcConvTokenUsage(conv))
			return
		}
		if wantMessages {
			jsonResp(w, conv.Messages)
		} else {
			jsonResp(w, conv)
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
			conv.Title = req.Title
		}
		conv.UpdatedAt = time.Now().Format("2006-01-02T15:04:05")
		saveConversations()
		jsonResp(w, conv)

	case "POST":
		if !wantMessages {
			jsonErr(w, "请使用 /conversations/{id}/messages")
			return
		}
		var msg struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			jsonErr(w, err.Error())
			return
		}
		if msg.Role == "" {
			msg.Role = "user"
		}
		conv.Messages = append(conv.Messages, msg)
		conv.UpdatedAt = time.Now().Format("2006-01-02T15:04:05")
		saveConversations()
		jsonResp(w, map[string]any{"ok": true, "messageIndex": len(conv.Messages) - 1})

	case "DELETE":
		newConvs := make([]Conversation, 0, len(conversations))
		for _, c := range conversations {
			if c.ID != id {
				newConvs = append(newConvs, c)
			}
		}
		conversations = newConvs
		saveConversations()
		// 同步删除记忆索引
		memory.Delete(id)
		jsonResp(w, map[string]any{"ok": true})

	default:
		jsonErr(w, "不支持的方法")
	}
}

// calcConvTokenUsage 从对话消息估算 token 用量。
func calcConvTokenUsage(conv *Conversation) map[string]int {
	var prompt, completion float64
	for _, m := range conv.Messages {
		tokens := float64(0)
		cjk := 0
		other := 0
		for _, r := range m.Content {
			if (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF) || (r >= 0xF900 && r <= 0xFAFF) {
				cjk++
			} else {
				other++
			}
		}
		tokens = float64(cjk)*1.5 + float64(other)*0.25 + 4

		if m.Role == "user" || m.Role == "system" {
			prompt += tokens
		} else {
			completion += tokens
		}
	}
	promptInt := int(prompt + 0.5)
	completionInt := int(completion + 0.5)
	return map[string]int{
		"promptTokens":     promptInt,
		"completionTokens": completionInt,
		"totalTokens":      promptInt + completionInt,
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
		"providers": providers,
		"models":    modelMap,
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
				Level: "user", Enabled: true,
			})
		}
	}
	if lv == "" || lv == "all" || lv == "project" {
		for _, e := range mcppanel.ReadLevel(mcppanel.LevelProject) {
			out = append(out, mcpItem{
				Name: e.Name, Command: e.Command, Args: e.Args,
				Level: "project", Enabled: true,
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

// ─── 工作区级 Token 统计 ────────────────────────────────────

// WorkspaceTokenStats 工作区累计 token 用量。
type WorkspaceTokenStats struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
	CacheHitTokens   int `json:"cacheHitTokens"`
	CacheMissTokens  int `json:"cacheMissTokens"`
	SystemTokens     int `json:"systemTokens"`
	SkillsTokens     int `json:"skillsTokens"`
	MCPTokens        int `json:"mcpTokens"`
	ToolTokens       int `json:"toolTokens"`
	HistoryTokens    int `json:"historyTokens"`
	OtherTokens      int `json:"otherTokens"`
}

var (
	wsTokenStats       WorkspaceTokenStats
	wsTokenStatsMu     sync.Mutex
	wsTokenStatsLoaded bool
)

func tokenStatsPath() string {
	root := core.Root()
	if root == "" {
		return ""
	}
	return filepath.Join(root, ".pair", "token-stats.json")
}

func loadWorkspaceTokenStats() {
	wsTokenStatsMu.Lock()
	defer wsTokenStatsMu.Unlock()
	if wsTokenStatsLoaded {
		return
	}
	wsTokenStatsLoaded = true
	path := tokenStatsPath()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &wsTokenStats)
}

func saveWorkspaceTokenStats() {
	path := tokenStatsPath()
	if path == "" {
		return
	}
	data, _ := json.MarshalIndent(wsTokenStats, "", "  ")
	os.WriteFile(path, data, 0644)
}

func accumulateWorkspaceTokens(promptTokens, completionTokens, cacheHit, cacheMiss int, pb *agent.PromptBreakdown) {
	wsTokenStatsMu.Lock()
	defer wsTokenStatsMu.Unlock()
	wsTokenStats.PromptTokens += promptTokens
	wsTokenStats.CompletionTokens += completionTokens
	wsTokenStats.TotalTokens += promptTokens + completionTokens
	wsTokenStats.CacheHitTokens += cacheHit
	wsTokenStats.CacheMissTokens += cacheMiss
	if pb != nil {
		wsTokenStats.SystemTokens = pb.SystemTokens
		wsTokenStats.SkillsTokens = pb.SkillsTokens
		wsTokenStats.MCPTokens = pb.MCPTokens
		wsTokenStats.ToolTokens = pb.ToolTokens
		wsTokenStats.HistoryTokens = pb.HistoryTokens
		wsTokenStats.OtherTokens = pb.OtherTokens
	}
	saveWorkspaceTokenStats()
}

func resetWorkspaceTokenStats() {
	wsTokenStatsMu.Lock()
	defer wsTokenStatsMu.Unlock()
	wsTokenStats = WorkspaceTokenStats{}
	saveWorkspaceTokenStats()
}

func (s *webServer) handleTokensStats(w http.ResponseWriter, r *http.Request) {
	loadWorkspaceTokenStats()
	switch r.Method {
	case "GET":
		wsTokenStatsMu.Lock()
		stats := wsTokenStats
		wsTokenStatsMu.Unlock()
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
		resetWorkspaceTokenStats()
		jsonResp(w, map[string]any{"ok": true, "message": "Token 统计已重置"})
	default:
		jsonErr(w, "不支持的方法")
	}
}

// ─── 辅助 ────────────────────────────────────────────────────

func jsonResp(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
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

// loadConversationHistory 从对话文件加载过往消息作为 LLM 上下文。
// 同时注入其他对话的历史摘要，让 Agent 了解之前对话的背景。
// 用 agent.BuildHistory 排除最后一条消息避免 loop.Run 重复添加。
func (s *webServer) loadConversationHistory(convID string) []agent.Message {
	// 直接使用内存中的 conversations，不从磁盘重读（避免并发 save 的竞态）
	hist := make([]agent.Message, 0, 32)

	// ── 1. 加载当前对话的过往消息 ──
	if convID == "" {
		return hist
	}
	var conv *Conversation
	for i := range conversations {
		if conversations[i].ID == convID {
			conv = &conversations[i]
			break
		}
	}
	if conv == nil || len(conv.Messages) == 0 {
		return hist
	}
	// 统一用 agent.BuildHistory 排除最后一条用户消息，
	// 避免 loop.Run 内部再次添加时造成重复。
	all := make([]agent.Message, 0, len(conv.Messages))
	for _, msg := range conv.Messages {
		role := agent.RoleAssistant
		switch msg.Role {
		case "user":
			role = agent.RoleUser
		case "system":
			role = agent.RoleSystem
		}
		all = append(all, agent.Message{Role: role, Content: msg.Content})
	}
	return agent.BuildHistory(all)
}

// ── 对话摘要生成 ──────────────────────────────────────────

// generateConversationSummary 用 LLM 生成对话摘要并保存到 conversations.json。
// 在对话完全结束后调用（goroutine 内异步）。prov 为用于生成摘要的 provider。
// 如 prov 为 nil 或生成失败，则用规则式摘要作为兜底，不影响主流程。
func generateConversationSummary(convID string, prov agent.Provider) {
	if convID == "" {
		return
	}
	_ = prov

	loadConversations()
	var conv *Conversation
	for i := range conversations {
		if conversations[i].ID == convID {
			conv = &conversations[i]
			break
		}
	}
	if conv == nil || len(conv.Messages) == 0 {
		return
	}

	// 已有摘要且消息数没变 → 跳过（避免重复生成）
	if conv.Summary != "" {
		return
	}

	// 使用 pkg/summary 生成摘要
	msgs := make([]summary.Message, len(conv.Messages))
	for i, m := range conv.Messages {
		msgs[i] = summary.Message{Role: m.Role, Content: m.Content}
	}
	s := summary.Generate(summary.ConvInfo{
		ID: conv.ID, Title: conv.Title,
		CreatedAt: conv.CreatedAt, UpdatedAt: conv.UpdatedAt,
		Messages: msgs,
	})
	if s == "" {
		return
	}
	conv.Summary = s
	conv.SummaryAt = time.Now().Format("2006-01-02 15:04:05")
	saveConversations()

	// 同步写入记忆索引
	assistantMsgs := make([]string, 0)
	allMsgs := make([]string, len(conv.Messages))
	for i, m := range conv.Messages {
		allMsgs[i] = m.Content
		if m.Role == "assistant" {
			assistantMsgs = append(assistantMsgs, m.Content)
		}
	}
	memory.Upsert(memory.Entry{
		ID:           conv.ID,
		Title:        conv.Title,
		Summary:      s,
		CreatedAt:    conv.CreatedAt,
		UpdatedAt:    conv.UpdatedAt,
		MessageCount: len(conv.Messages),
		Tags:         memory.ExtractTags(allMsgs),
		KeyPoints:    memory.ExtractKeyPoints(assistantMsgs),
		CompletedAt:  conv.SummaryAt,
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
	loadConversations()
	count := 0
	for _, c := range conversations {
		if len(c.Messages) < 2 {
			continue
		}
		s := c.Summary
		if s == "" {
			msgs := make([]summary.Message, len(c.Messages))
			for j, m := range c.Messages {
				msgs[j] = summary.Message{Role: m.Role, Content: m.Content}
			}
			s = summary.Generate(summary.ConvInfo{ID: c.ID, Title: c.Title, Messages: msgs})
		}
		assistantMsgs := make([]string, 0)
		allMsgs := make([]string, len(c.Messages))
		for j, m := range c.Messages {
			allMsgs[j] = m.Content
			if m.Role == "assistant" {
				assistantMsgs = append(assistantMsgs, m.Content)
			}
		}
		memory.Upsert(memory.Entry{
			ID: c.ID, Title: c.Title, Summary: s,
			CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
			MessageCount: len(c.Messages),
			Tags:         memory.ExtractTags(allMsgs),
			KeyPoints:    memory.ExtractKeyPoints(assistantMsgs),
			CompletedAt:  c.SummaryAt,
		})
		count++
	}
	jsonResp(w, map[string]any{"status": "ok", "count": count})
}

// ─── Git API 处理函数 ─────────────────────────────────────────

// gitRoot 返回 git 工作目录（工作区根），无工作区时返回空。
func gitRoot() string {
	if root := core.Root(); root != "" {
		return root
	}
	cwd, _ := os.Getwd()
	return cwd
}

// runGitInternal 执行 git 命令，返回标准输出。
func runGitInternal(ctx context.Context, args ...string) (string, error) {
	dir := gitRoot()
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
		return "", fmt.Errorf(errMsg)
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
			line = strings.TrimSpace(line)
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

// handleGitDiff GET /api/git/diff?file=xxx&staged=true 查看文件差异。
func (s *webServer) handleGitDiff(w http.ResponseWriter, r *http.Request) {
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

	args := []string{"log", "--max-count=" + count, "--pretty=format:%H|%h|%an|%ai|%s"}
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
	root := gitRoot()
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
