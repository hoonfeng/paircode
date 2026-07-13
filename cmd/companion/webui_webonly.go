// Web-only 版 handler 与事件持久化（无 bridge/GWui 依赖）。
// 与 webui_desktop.go 功能等价，但去除 bridge.BuildCompressor() 依赖
// （webonly 模式用 nil Compressor，规则式压缩同样工作）。
//
//go:build webonly

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
	"github.com/hoonfeng/paircode/cmd/companion/agenttools"
	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/roleprompts"
	mcppanel "github.com/hoonfeng/paircode/cmd/companion/ui/mcp"
	marketplacepanel "github.com/hoonfeng/paircode/cmd/companion/ui/marketplace"
	"github.com/hoonfeng/paircode/cmd/companion/ui/skills"
	"github.com/hoonfeng/paircode/pkg/memory"
)

// registerExtraHandlers 注册 webonly 模式需要的路由。
func registerExtraHandlers(mux *http.ServeMux, s *webServer) {
	mux.HandleFunc("/api/chat/send", s.handleChatSend)
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/api/chat/stop", s.handleChatStop)
	mux.HandleFunc("/api/chat/answer", s.handleChatAnswer)
	mux.HandleFunc("/api/chat/approve", s.handleChatApprove)
	mux.HandleFunc("/api/chat/feedback", s.handleChatFeedback)
	mux.HandleFunc("/api/marketplace/search", s.handleMarketplaceSearch)
	mux.HandleFunc("/api/marketplace/install", s.handleMarketplaceInstall)
	mux.HandleFunc("/api/memory/search", s.handleMemorySearch)
	mux.HandleFunc("/api/memory/list", s.handleMemoryList)
	mux.HandleFunc("/api/memory/rebuild", s.handleMemoryRebuild)
}

// buildWebLoopOpts 构建 LoopOpts（无 bridge 依赖，Compressor 用 nil 规则式压缩）。
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
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)

	reg.Register(&agent.Tool{
		Name:        "finish_task",
		Description: "任务完成信号：全部任务完成时调用此工具结束本轮。result 为完成摘要。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"result": map[string]any{"type": "string", "description": "任务完成摘要"},
			},
			"required": []string{"result"},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			r, _ := args["result"].(string)
			return r, nil
		},
	})

	agenttools.RegisterManagementTools(reg, root)
	if cfgs := mcppanel.LoadConfigs(); len(cfgs) > 0 {
		agentCfgs := make([]agent.MCPServerConfig, len(cfgs))
		for i, c := range cfgs {
			agentCfgs[i] = agent.MCPServerConfig{Name: c.Name, Command: c.Command, Args: c.Args, Env: c.Env}
		}
		agent.RegisterMCPServers(reg, agentCfgs)
	}

	reloadWebLuaTools(reg, root)
	agent.InitDebugLogger(root, 50)

	sys := buildWebSystemPrompt()

	if root != "" {
		execMgr := agent.InitExecStateManager(root)
		var stateParts []string

		interrupted := execMgr.FindInterrupted()
		if interrupted != nil {
			stateSummary := interrupted.GetSummary()
			stateParts = append(stateParts,
				"## 项目未完成任务\n"+stateSummary+
					"\n注意：以上是项目中尚未完成的任务状态。请继续推进完成这些任务。"+
					"\n如果状态显示有中断的运行，请优先恢复并完成它。")
		}

		allStates := execMgr.ListAll()
		completedStates := make([]*agent.ExecutionState, 0)
		for _, st := range allStates {
			if st.Status == agent.ExecCompleted {
				completedStates = append(completedStates, st)
			}
		}
		if len(completedStates) > 0 {
			var completedSb strings.Builder
			completedSb.WriteString(fmt.Sprintf("## 已完成任务（最近 %d 条）\n\n", min(3, len(completedStates))))
			for i := 0; i < min(3, len(completedStates)); i++ {
				st := completedStates[i]
				completedSb.WriteString(fmt.Sprintf("- **%s** — %s (%d 轮, %d 文件变更)\n",
					st.Task, st.UpdatedAt, st.LoopCount, len(st.ModifiedFiles)))
			}
			stateParts = append(stateParts, completedSb.String())
		}

		if agent.GlobalDebugLogger != nil {
			errorSummary := agent.GlobalDebugLogger.GetErrorSummary(3)
			if errorSummary != "" && errorSummary != "（无错误日志）" {
				stateParts = append(stateParts,
					"## 项目中待处理的错误\n"+errorSummary+
						"\n注意：以上是检测到的错误。请分析并修复它们。")
			}
		}

		if len(allStates) > 0 {
			stats := fmt.Sprintf("## 项目执行统计\n- 总执行次数: %d\n- 运行中: %d\n- 已完成: %d\n- 失败: %d\n- 已取消: %d\n",
				len(allStates),
				countStates(allStates, agent.ExecRunning),
				countStates(allStates, agent.ExecCompleted),
				countStates(allStates, agent.ExecFailed),
				countStates(allStates, agent.ExecCancelled))
			stateParts = append(stateParts, stats)
		}

		if len(stateParts) > 0 {
			sys += "\n\n# 项目当前状态\n" + strings.Join(stateParts, "\n\n")
		}

		recentMemories := memory.List()
		if len(recentMemories) > 0 {
			var memSb strings.Builder
			limit := 5
			if len(recentMemories) < limit {
				limit = len(recentMemories)
			}
			memSb.WriteString(fmt.Sprintf("## 最近对话摘要（最近 %d 条）\n\n", limit))
			memSb.WriteString("> ⚠️ 以下摘要是**已完成的历史对话**，与当前对话无关。请勿重复执行已完成的任务。\n> 当前对话中用户的新消息在下方 `[User]` 消息中。\n\n")
			for i := 0; i < limit; i++ {
				m := recentMemories[i]
				title := m.Title
				if title == "" || title == "新对话" {
					title = "未命名对话"
				}
				memSb.WriteString(fmt.Sprintf("- **%s**", title))
				if m.Summary != "" {
					memSb.WriteString(": " + m.Summary)
				}
				memSb.WriteString("\n")
			}
			memSb.WriteString("\n（需要更详细的历史信息可用 memory_search / memory_read 检索具体对话。）")
			sys += "\n\n# 已完成对话历史\n" + memSb.String()
		}
	}

	// ── 加载对话历史（委托给 MessageStore） ──
	// 从 store 加载完整历史（含 ToolCalls/Reasoning），传给 LoopOpts.History。
	// SessionManager.Start 会设置到 loop.History。
	var history []agent.Message
	if convID != "" {
		if store := agentMgr.Store(); store != nil {
			history, _ = store.LoadAll(convID)
		}
	}

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
		Compressor:       nil, // webonly 模式使用规则式压缩，无需 bridge
		History:          history,
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
	// 持久化用户消息到 MessageStore（SessionManager.Start 内部会调 store.CreateConversation 若不存在）
	agentMgr.AppendPersistedUserMessage(req.ConvID, req.Message)
	opts := s.buildWebLoopOpts(req.ConvID, req.Message, req.Autonomous)
	opts.WorkspaceRoot = req.WorkspaceRoot
	// 审核开关：只需设 AutoReview 和 ReviewProvider，审核决策由 Loop 内部自决
	opts.AutoReview = core.Settings.AutoReview
	if core.Settings.AutoReview && core.Settings.ReviewModel != "" {
		rm := strings.TrimSpace(core.Settings.ReviewModel)
		base := strings.TrimSpace(core.Settings.BaseURL)
		key := strings.TrimSpace(core.Settings.APIKey)
		if rm != "" && base != "" && key != "" {
			opts.ReviewProvider = &agent.OpenAIProvider{BaseURL: base, APIKey: key, Model: rm, Temperature: -1, ThinkingMode: "non-thinking"}
		}
	}
	// 自动 git 提交开关
	opts.AutoCommit = core.Settings.AutoCommit

	taskText := req.Message
	if req.Autonomous {
		taskText += "\n\n（自主模式：先用 update_plan 列出完整计划，然后连续完成所有步骤、全部完成后调用 finish_task 工具。）"
	}

	ctx := context.Background()
	if err := agentMgr.Start(ctx, req.ConvID, taskText, opts); err != nil {
		jsonErr(w, err.Error())
		return
	}

	jsonResp(w, map[string]any{"ok": true, "convId": req.ConvID})
}

// startEventPersistWorker 启动后台 goroutine 订阅全局事件流（无 bridge 依赖）。
func (s *webServer) startEventPersistWorker() {
	ch := agentMgr.SubscribeAll()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.persistRunningHistories()
		}
	}()
	go func() {
		// 按需持久化：EventUsage（token 统计写入 store）、EventDone/EventError（diff-based 追加历史）
		// 不对 thinking/content/tool_call 等高频事件做磁盘 I/O，避免阻塞事件消费循环。
		for ge := range ch {
			convID := ge.ConvID
			if convID == "" {
				continue
			}
			switch ge.Event.Type {
			case agent.EventUsage:
				if ge.Event.Usage != nil {
					if store := agentMgr.Store(); store != nil {
						_ = store.SetCtxStats(convID, ge.Event.Usage)
					}
				}
			case agent.EventDone, agent.EventError:
				var hist []agent.Message
				if ge.Event.Type == agent.EventDone {
					hist = agentMgr.GetHistory(convID)
				} else {
					hist = agentMgr.GetCurrentHistory(convID)
				}
				if hist != nil {
				if store := agentMgr.Store(); store != nil {
					existing, _ := store.Count(convID)
					if len(hist) > existing {
						for i := existing; i < len(hist); i++ {
							_ = store.AppendMessage(convID, hist[i], agent.SegmentsFromMessage(hist[i], hist, i))
						}
					}
				}
			}
				if ge.Event.Type == agent.EventDone && convID != "" {
					go generateConversationSummary(convID, nil) // webonly 无 bridge compressor
				}
			}
			// 其他事件类型不触发磁盘写入——由上方 ticker goroutine 每 5 秒统一增量持久化运行中会话的历史。
		}
	}()
}

// persistRunningHistories 增量追加所有运行中会话的新消息到 MessageStore。
func (s *webServer) persistRunningHistories() {
	for _, convID := range agentMgr.ListRunning() {
		hist := agentMgr.GetCurrentHistory(convID)
		if hist == nil {
			continue
		}
		if store := agentMgr.Store(); store != nil {
			existing, _ := store.Count(convID)
			if len(hist) > existing {
				for i := existing; i < len(hist); i++ {
					_ = store.AppendMessage(convID, hist[i], agent.SegmentsFromMessage(hist[i], hist, i))
				}
			}
		}
	}
}

// handleChatStop 停止指定会话的 agent 运行。
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

// handleChatAnswer 向指定会话发送 ask_user 的用户回答。
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

// handleChatApprove 向指定会话发送审批结果。
func (s *webServer) handleChatApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		ConvID   string `json:"convId"`
		Approved bool   `json:"approved"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.ConvID == "" {
		jsonErr(w, "convId 必填")
		return
	}
	if err := agentMgr.Approve(req.ConvID, req.Approved); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

// handleChatFeedback 向指定会话发送运行时反馈（补充/纠正）。
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
		Installed   bool     `json:"installed"`
	}
	out := make([]resultItem, 0, len(results))
	for _, e := range results {
		out = append(out, resultItem{
			ID: e.ID, Kind: e.Kind, Name: e.Name,
			Description: e.Description, Tags: e.Tags,
			Installed: marketplacepanel.IsInstalled(e.ID),
		})
	}
	jsonResp(w, out)
}

// ─── 市场安装 API ──────────────────────────────────────────

func (s *webServer) handleMarketplaceInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.ID == "" {
		jsonErr(w, "id 必填")
		return
	}
	msg, err := marketplacepanel.InstallScoped(req.ID, false)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "message": msg})
}

// ─── 辅助函数 ───────────────────────────────────────────────

// buildWebSystemPrompt 构建 web 模式的完整系统提示词。
func buildWebSystemPrompt() string {
	sys := agent.DefaultSystemPrompt(core.Folders)
	if si := strings.TrimSpace(core.Settings.SystemInstructions); si != "" {
		sys += "\n\n# 系统级指令（务必遵守）\n" + si
	}
	sys += roleprompts.PhilosophyPrompt()
	sys += skills.Prompt()
	sys += "\n\n# 自管理与扩展\n你可自我扩展：skill_list / load_skill / load_skill_resource / skill_write / skill_delete 管理技能；" +
		"mcp_list / mcp_add / mcp_remove 管理 MCP 服务器；marketplace_search / marketplace_install 从市场检索并安装 MCP 或技能。"
	if core.Settings.LuaTools {
		sys += "\n\n# 自定义工具（Lua）\n可在工作区 .pair/tools/ 下写 .lua 脚本自定义工具。"
	}
	sys += "\n\n# 长时记忆检索\n你可以使用以下内部工具检索历史已完成对话的记忆（用于了解之前的工作成果）：\n" +
		"- `memory_search` 搜索历史记忆（标题/摘要/标签/关键点），按关键词筛选\n" +
		"- `memory_list` 列出所有历史记忆（按完成时间倒序）\n" +
		"- `memory_count` 查询记忆总数\n" +
		"注意：新对话开始时系统已自动注入最近的对话摘要到本提示中；如需更详细的历史记录可使用上述工具检索。"
	root := core.Root()
	sys += agent.ProjectRules(root)
	sys += agent.ProjectKnowledge(root, 2500)
	return sys
}

// buildWebProvider 构建 web 模式的 LLM Provider。
func buildWebProvider() agent.Provider {
	s := core.Settings
	if s.APIKey == "" || s.BaseURL == "" {
		return nil
	}
	// 配置健康检查：maxTokens 过小会导致思考/回复被截断
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

// reloadWebLuaTools 加载工作区 .pair/tools/*.lua 自定义工具。
func reloadWebLuaTools(reg *agent.Registry, root string) {
	if !core.Settings.LuaTools {
		log.Printf("[LuaTools] Lua 工具已全局禁用")
		return
	}
	dir := filepath.Join(root, ".pair", "tools")
	loaded := agent.LoadLuaTools(reg, dir)
	log.Printf("[LuaTools] 工作区 %s 的 .pair/tools/ 加载了 %d 个 Lua 工具: %v", root, len(loaded), loaded)
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
