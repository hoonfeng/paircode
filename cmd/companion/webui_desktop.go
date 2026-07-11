// 桌面版额外路由注册：chat/agent + market。
//
//go:build windows

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
	"github.com/hoonfeng/paircode/cmd/companion/agenttools"
	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/roleprompts"
	marketplacepanel "github.com/hoonfeng/paircode/cmd/companion/ui/marketplace"
	mcppanel "github.com/hoonfeng/paircode/cmd/companion/ui/mcp"
	"github.com/hoonfeng/paircode/cmd/companion/ui/skills"
)

func registerExtraHandlers(mux *http.ServeMux, s *webServer) {
	mux.HandleFunc("/api/chat/send", s.handleChatSend)
	mux.HandleFunc("/api/chat/stop", s.handleChatStop)
	mux.HandleFunc("/api/chat/answer", s.handleChatAnswer)
	mux.HandleFunc("/api/chat/approve", s.handleChatApprove)
	mux.HandleFunc("/api/marketplace/search", s.handleMarketplaceSearch)
	mux.HandleFunc("/api/marketplace/install", s.handleMarketplaceInstall)

	// 长时记忆检索 API
	mux.HandleFunc("/api/memory/search", s.handleMemorySearch)
	mux.HandleFunc("/api/memory/list", s.handleMemoryList)
	mux.HandleFunc("/api/memory/rebuild", s.handleMemoryRebuild)
}

// emitPhase 发送阶段切换事件到 SSE 流。
func emitPhase(session *webAgentSession, phase string) {
	select {
	case session.events <- agent.Event{Type: agent.EventPhase, Content: phase}:
	default:
	}
}

// runOrchestrationLoop 外部编排循环。
// 每次 loop.Run 完成后，用编排 agent 分析已完成内容并决定下一个任务，
// 将规划记录到 .pair/tasks/ 目录，然后继续执行，直到编排 agent 判定全部完成。
// 支持执行状态持久化和断点续跑。
// restoredState 可选：断点续跑时传入中断的执行状态，避免从头开始。
func runOrchestrationLoop(ctx context.Context, prov agent.Provider, reg *agent.Registry, session *webAgentSession, task string, history []agent.Message, roots []string, root string, convID string, restoredState *agent.ExecutionState) []agent.Message {
	emitPhase(session, "自主执行中")

	allHistory := make([]agent.Message, len(history))
	copy(allHistory, history)

	webSysPrompt := buildWebSystemPrompt()
	planner := buildWebPlanner()
	missionTask := task
	var loopCount int
	if restoredState != nil {
		loopCount = restoredState.LoopCount
	}
	const maxLoops = 10 // 最大编排轮次，防无限循环

	// ── 初始化执行状态管理器 ──
	execMgr := agent.InitExecStateManager(root)
	var execState *agent.ExecutionState
	if restoredState != nil {
		execState = restoredState
		execState.Status = agent.ExecRunning
		missionTask = execState.MissionTask
		if missionTask == "" {
			missionTask = task
		}
		execMgr.Save(execState)
	} else {
		execState = execMgr.Create(task, maxLoops, convID)
		execState.MissionTask = missionTask
	}

	// ── Panic 静默恢复 ──
	ctxMap := map[string]string{"convId": convID, "task": task}
	defer execMgr.RecoverPanic(&execState, "runOrchestrationLoop", ctxMap)

	// ── 记录任务规划到 .pair/tasks/ ──
	saveTaskPlan := func(name, content string) {
		if root == "" {
			return
		}
		tasksDir := filepath.Join(root, ".pair", "tasks")
		os.MkdirAll(tasksDir, 0755)
		filePath := filepath.Join(tasksDir, name+".md")
		os.WriteFile(filePath, []byte(content), 0644)
	}

	// ── 文件变更追踪（通过 agent.FileChangeCallback 自动触发）──
	// 在 loop 开始前设置回调
	agent.FileChangeCallback = func(filePath string) {
		execMgr.RecordFileChange(execState, filePath)
	}

	// ── 初次规划阶段 ──
	if planner != nil {
		emitPhase(session, "规划阶段")
		execState.Phase = "规划阶段"
		execMgr.Save(execState)
		plan, perr := planner.Plan(ctx, missionTask, history)
		if perr == nil && len(plan.Steps) > 0 {
			evtArgs := planToUpdateArgs(plan)
			select {
			case session.events <- agent.Event{Type: agent.EventToolCall, Tool: "update_plan", Args: evtArgs}:
			default:
			}
			// 记录初始规划
			planContent := fmt.Sprintf("# 初始规划: %s\n\n## 推理\n%s\n\n## 步骤\n%s\n\n- 创建时间: %s\n- 状态: 进行中\n",
				missionTask, plan.Reasoning, planStepsText(plan), time.Now().Format("2006-01-02 15:04:05"))
			saveTaskPlan(fmt.Sprintf("plan_%s", time.Now().Format("20060102_150405")), planContent)
			missionTask = missionTask + "\n\n（规划 Agent 已制定以下计划，请据此连续执行、用 update_plan 更新各步状态）：\n" + planStepsText(plan)
			execState.MissionTask = missionTask
			execMgr.Save(execState)
		}
	}

	buildMainLoop := func() *agent.Loop {
		maxIter := core.Settings.MaxIterations * 2
		if maxIter <= 0 {
			maxIter = 60
		}
		return &agent.Loop{
			Provider:         prov,
			Registry:         reg,
			Autonomous:       true,
			System:           webSysPrompt,
			MaxIterations:    maxIter,
			MaxContextTokens: core.Settings.ContextMaxTokens,
			Compressor:       buildWebCompressor(),
			OnEvent: func(e agent.Event) {
				select {
				case session.events <- e:
				default:
				}
			},
			Approve: func(ctx context.Context, tc agent.ToolCall) (bool, string) {
				if core.Settings.RequireApproval {
					session.approvalCallID = tc.ID
					select {
					case session.events <- agent.Event{
						Type:   agent.EventApproval,
						Tool:   tc.Function.Name,
						Args:   tc.Function.Arguments,
						CallID: tc.ID,
					}:
					default:
					}
					select {
					case approved := <-session.approvalCh:
						return approved, ""
					case <-ctx.Done():
						session.approvalCallID = ""
						return false, "用户取消了操作"
					}
				}
				return true, ""
			},
		}
	}

	for loopCount < maxLoops {
		loopCount++
		if err := ctx.Err(); err != nil {
			execMgr.MarkFailed(execState, ctx.Err())
			return
		}

		emitPhase(session, fmt.Sprintf("执行 %d/%d", loopCount, maxLoops))
		execState.LoopCount = loopCount
		execState.Phase = fmt.Sprintf("执行 %d/%d", loopCount, maxLoops)
		execMgr.Save(execState)

		mainLoop := buildMainLoop()
		msgs, err := mainLoop.Run(ctx, missionTask, allHistory)

		// 如果不是首次循环，记录本轮完成摘要
		currentTask := missionTask
		if loopCount > 1 {
			currentTask = fmt.Sprintf("第 %d 轮任务", loopCount)
		}

		// ── 自动构建验证 ──
		if err == nil && root != "" {
			emitPhase(session, "验证中")
			execState.Phase = "验证中"
			execMgr.Save(execState)
			verifyResult := autoVerifyProject(root)
			if !verifyResult.success {
				// ── 记录构建错误日志 ──
				agent.WriteBuildError("autoVerifyProject", verifyResult.output,
					map[string]string{"convId": convID, "loopCount": fmt.Sprintf("%d", loopCount)})

				// 构建失败 → 使用 BugDetector 自动分析错误位置并生成详细的修复任务
				errMsg := agent.FormatBuildErrorForAgent(verifyResult.output, root)

				// 发通知到前端
				select {
				case session.events <- agent.Event{Type: agent.EventNotice, Content: errMsg}:
				default:
				}

				// 检查是否已设置 fixAttempts（首次修复时初始化为 0）
				// 从 execState.Errors 推断是否已有修复尝试
				fixAttempts := 0
				for _, e := range execState.Errors {
					if strings.Contains(e, "修复尝试") {
						fixAttempts++
					}
				}
				fixAttempts++

				if fixAttempts > 3 {
					// 超过 3 次修复尝试仍未通过，报告错误并退出
					finalErrMsg := fmt.Sprintf("❌ 连续 %d 次修复尝试后构建仍未通过:\n\n%s", fixAttempts, verifyResult.output)
					select {
					case session.events <- agent.Event{Type: agent.EventError, Content: finalErrMsg}:
					default:
					}
					execMgr.MarkFailed(execState, fmt.Errorf("连续 %d 次修复尝试失败", fixAttempts))
					return
				}

				// 将解析后的修复任务注入下一轮 loop
				missionTask = errMsg + "\n\n（请根据上述错误分析逐条修复，每次修复后运行 go_build 验证。全部修复完成后调用 finish_task。）"
				execState.MissionTask = missionTask
				execState.Errors = append(execState.Errors, fmt.Sprintf("构建失败（第 %d 次修复尝试）", fixAttempts))
				execMgr.Save(execState)
				allHistory = msgs
				continue
			}
			// ── 构建通过 → 检查 debug 日志中是否有需要分析的错误 ──
			emitPhase(session, "debug 分析")
			execState.Phase = "debug 分析"
			execMgr.Save(execState)
			errorSummary := ""
			if agent.GlobalDebugLogger != nil {
				errorSummary = agent.GlobalDebugLogger.GetErrorSummary(5)
			}
			if errorSummary != "" && errorSummary != "（无错误日志）" {
				// 有错误日志 → 注入到下一轮任务中
				debugTaskMsg := fmt.Sprintf("检测到以下错误日志，请分析并修复:\n\n%s\n\n（分析完成后，如果不需要修复或已修复，继续推进其他任务。）", errorSummary)
				select {
				case session.events <- agent.Event{Type: agent.EventNotice, Content: debugTaskMsg}:
				default:
				}
				// 注入 debug 分析任务（不中断主任务流，作为额外提示）
				msgs = append(msgs, agent.Message{Role: "user", Content: debugTaskMsg})
			}
		}

		if err == nil {
			// ── loop 正常完成 → 用编排 agent 分析下一步 ──
			emitPhase(session, "分析完成情况")
			execState.Phase = "分析完成情况"
			execMgr.Save(execState)
			if planner != nil {
				// 将本轮结果传给编排 agent 做分析
				analysisPrompt := fmt.Sprintf(`你执行了任务。请分析本轮结果并决定下一步。

总体任务: %s

如果全部完成，请回复：全部完成
如果还有下一步，请回复：下一步任务：<具体描述>`, task)

				analysis, aerr := planner.Plan(ctx, analysisPrompt, msgs)
				nextTask := ""
				if aerr == nil && len(analysis.Steps) > 0 {
					nextTask = analysis.Reasoning
					if idx := strings.Index(nextTask, "下一步任务："); idx >= 0 {
						nextTask = strings.TrimSpace(nextTask[idx+len("下一步任务："):])
					} else if idx := strings.Index(nextTask, "下一步："); idx >= 0 {
						nextTask = strings.TrimSpace(nextTask[idx+len("下一步："):])
					}
				}

				if nextTask == "" || strings.Contains(nextTask, "全部完成") || strings.Contains(strings.ToLower(nextTask), "all complete") {
					emitPhase(session, "全部完成")
					// 发射最终的 EventDone（含完成摘要），供前端展示完成报告
					select {
					case session.events <- agent.Event{Type: agent.EventDone, Content: fmt.Sprintf("全部任务已完成。\n\n完成轮次: %d\n总任务: %s\n", loopCount, task), DoneReason: "finish_task"}:
					default:
					}
					saveTaskPlan(fmt.Sprintf("plan_%s_complete", time.Now().Format("20060102_150405")),
						fmt.Sprintf("# 任务完成: %s\n\n## 完成时间\n%s\n\n## 完成摘要\n全部任务已完成。\n", task, time.Now().Format("2006-01-02 15:04:05")))
					execMgr.MarkCompleted(execState, "全部任务已完成")
					return
				}

				// 有下一步任务 → 记录规划并继续
				planContent := fmt.Sprintf("# 任务规划: %s\n\n## 当前轮次\n第 %d 轮\n\n## 已完成\n%s\n\n## 下一步\n%s\n\n- 时间: %s\n",
					task, loopCount, currentTask, nextTask, time.Now().Format("2006-01-02 15:04:05"))
				saveTaskPlan(fmt.Sprintf("plan_%s_r%02d", time.Now().Format("20060102_150405"), loopCount), planContent)

				// 记录完成的步骤
				execState.CompletedSteps = append(execState.CompletedSteps, agent.StepRecord{
					StepNum:     loopCount,
					Description: currentTask,
					Status:      "completed",
					CompletedAt: time.Now().Format("2006-01-02 15:04:05"),
					Summary:     fmt.Sprintf("完成 → 下一步: %s", nextTask),
				})

				emitPhase(session, "继续下一步")
				missionTask = nextTask + "\n\n（请基于已完成的上下文，继续执行上述下一步任务。完成后调用 finish_task。）"
				execState.MissionTask = missionTask
				execMgr.Save(execState)
				allHistory = msgs
				continue
			}
			// 没有编排 agent → 直接结束
			select {
			case session.events <- agent.Event{Type: agent.EventDone, Content: fmt.Sprintf("任务完成（自动模式）\n\n完成轮次: %d", loopCount), DoneReason: "task_complete"}:
			default:
			}
			execMgr.MarkCompleted(execState, "任务完成（无编排 agent）")
			return
		}

		if errors.Is(err, agent.ErrCirclingLoop) {
			saveTaskPlan(fmt.Sprintf("plan_%s_circling", time.Now().Format("20060102_150405")),
				fmt.Sprintf("# 任务绕圈: %s\n\n## 时间\n%s\n\n## 错误\n检测到重复绕圈，已停止。\n",
					task, time.Now().Format("2006-01-02 15:04:05")))
			execMgr.MarkFailed(execState, err)
			select {
			case session.events <- agent.Event{Type: agent.EventError, Content: fmt.Sprintf("自主模式异常终止: %v", err)}:
			default:
			}
			return
		}

		// 其他错误
		execMgr.MarkFailed(execState, err)
		select {
		case session.events <- agent.Event{Type: agent.EventError, Content: fmt.Sprintf("自主模式异常终止: %v", err)}:
		default:
		}
		return
	}

	// 达到最大轮次
	emitPhase(session, "达到最大执行轮次")
	execState.Status = agent.ExecFailed
	execState.Errors = append(execState.Errors, fmt.Sprintf("已达最大执行轮次 %d", maxLoops))
	execMgr.Save(execState)
	select {
	case session.events <- agent.Event{Type: agent.EventError, Content: fmt.Sprintf("自主模式已达最大执行轮次 %d，终止", maxLoops)}:
	default:
	}
	return allHistory
}

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
func buildWebPlanner() *agent.Planner {
	base := strings.TrimSpace(core.Settings.BaseURL)
	key := strings.TrimSpace(core.Settings.APIKey)
	if base == "" || key == "" {
		return nil
	}
	model := strings.TrimSpace(core.Settings.PlanModel)
	if model == "" {
		return nil
	}
	return &agent.Planner{
		Provider: &agent.OpenAIProvider{
			BaseURL: base, APIKey: key, Model: model,
			Temperature: 0.3, MaxTokens: 2048, ThinkingMode: "non-thinking",
		},
		SystemPrompt: roleprompts.LoadRolePrompt("planner", agent.DefaultPlannerPrompt()) + roleprompts.RolePhilosophy("planner"),
	}
}

// planToUpdateArgs 把规划 Agent 的计划转成 update_plan 工具参数 JSON。
func planToUpdateArgs(plan agent.Plan) string {
	steps := make([]map[string]any, len(plan.Steps))
	for i, s := range plan.Steps {
		steps[i] = map[string]any{
			"id": s.ID, "description": s.Description,
			"status": "pending", "dependencies": s.Dependencies,
		}
	}
	b, _ := json.Marshal(map[string]any{"steps": steps})
	return string(b)
}

// planStepsText 把计划转成给执行 Agent 的纲领文本。
func planStepsText(plan agent.Plan) string {
	var sb strings.Builder
	sb.WriteString("## 规划思路\n" + plan.Reasoning + "\n\n## 执行步骤\n")
	for i, s := range plan.Steps {
		deps := ""
		if len(s.Dependencies) > 0 {
			deps = " [前置: " + strings.Join(s.Dependencies, ", ") + "]"
		}
		sb.WriteString(fmt.Sprintf("%d. **%s**%s\n", i+1, s.Description, deps))
	}
	return sb.String()
}

// buildWebCompressor 构建上下文压缩器。
func buildWebCompressor() agent.Provider {
	if !core.Settings.CompressEnabled {
		return nil
	}
	key := strings.TrimSpace(core.Settings.CompressAPIKey)
	if key == "" {
		key = strings.TrimSpace(core.Settings.APIKey)
	}
	base := strings.TrimSpace(core.Settings.CompressBaseURL)
	if base == "" {
		base = strings.TrimSpace(core.Settings.BaseURL)
	}
	model := strings.TrimSpace(core.Settings.CompressModel)
	if model == "" || key == "" || base == "" {
		return nil
	}
	return &agent.OpenAIProvider{
		BaseURL: base, APIKey: key, Model: model,
		Temperature:  0.3,
		MaxTokens:    1024,
		ThinkingMode: "non-thinking",
	}
}

// reloadWebLuaTools 加载工作区 .pair/tools/*.lua 自定义工具。
func reloadWebLuaTools(reg *agent.Registry, root string) {
	if !core.Settings.LuaTools {
		return
	}
	agent.LoadLuaTools(reg, filepath.Join(root, ".pair", "tools"))
}

func buildWebProvider() agent.Provider {
	s := core.Settings
	if s.APIKey == "" || s.BaseURL == "" {
		return nil
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

// ─── Chat / Agent SSE API ───────────────────────────────────

func (s *webServer) handleChatSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Message    string `json:"message"`
		SessionID  string `json:"sessionId"`
		Autonomous bool   `json:"autonomous"`
		ConvID     string `json:"convId"`
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
	if req.SessionID == "" {
		req.SessionID = fmt.Sprintf("sess_%d", time.Now().UnixNano())
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonErr(w, "不支持 SSE")
		return
	}

	if !core.Configured() {
		fmt.Fprintf(w, "data: %s\n\n", jsonStr(map[string]any{
			"type": "error", "content": "未配置 API key。请在设置面板中配置 API Key 和模型。",
		}))
		flusher.Flush()
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	session := &webAgentSession{
		cancel:     cancel,
		events:     make(chan agent.Event, 100),
		askCh:      make(chan string, 1),
		approvalCh: make(chan bool, 1),
	}

	s.mu.Lock()
	s.activeLoops[req.SessionID] = session
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.activeLoops, req.SessionID)
		s.mu.Unlock()
	}()

	prov := buildWebProvider()

	root := core.Root()
	agent.WorkspaceRoots = core.Folders
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)

	// finish_task：任务完成后 Agent 调用此工具结束本轮（loop 检测到后设置完成信号）。
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

	agenttools.RegisterManagementTools(reg)
	if cfgs := mcppanel.LoadConfigs(); len(cfgs) > 0 {
		agentCfgs := make([]agent.MCPServerConfig, len(cfgs))
		for i, c := range cfgs {
			agentCfgs[i] = agent.MCPServerConfig{Name: c.Name, Command: c.Command, Args: c.Args, Env: c.Env}
		}
		agent.RegisterMCPServers(reg, agentCfgs)
	}

	reloadWebLuaTools(reg, root)

	// ── 初始化调试日志系统 ──
	agent.InitDebugLogger(root, 50)

	sys := buildWebSystemPrompt()

	// ── 跨对话项目状态感知：注入完整的项目当前状态上下文 ──
	// 包括：未完成任务、已完成任务、错误日志、修改文件等
	if root != "" {
		execMgr := agent.InitExecStateManager(root)
		var stateParts []string

		// 1. 注入中断的任务状态
		interrupted := execMgr.FindInterrupted()
		if interrupted != nil {
			stateSummary := interrupted.GetSummary()
			stateParts = append(stateParts,
				"## 项目未完成任务\n"+stateSummary+
					"\n注意：以上是项目中尚未完成的任务状态。请继续推进完成这些任务。"+
					"\n如果状态显示有中断的运行，请优先恢复并完成它。")
		}

		// 2. 注入最近完成的任务摘要
		allStates := execMgr.ListAll()
		completedStates := make([]*agent.ExecutionState, 0)
		for _, s := range allStates {
			if s.Status == agent.ExecCompleted {
				completedStates = append(completedStates, s)
			}
		}
		if len(completedStates) > 0 {
			var completedSb strings.Builder
			completedSb.WriteString(fmt.Sprintf("## 已完成任务（最近 %d 条）\n\n", min(3, len(completedStates))))
			for i := 0; i < min(3, len(completedStates)); i++ {
				s := completedStates[i]
				completedSb.WriteString(fmt.Sprintf("- **%s** — %s (%d 轮, %d 文件变更)\n",
					s.Task, s.UpdatedAt, s.LoopCount, len(s.ModifiedFiles)))
			}
			stateParts = append(stateParts, completedSb.String())
		}

		// 3. 注入最近错误日志摘要
		if agent.GlobalDebugLogger != nil {
			errorSummary := agent.GlobalDebugLogger.GetErrorSummary(3)
			if errorSummary != "" && errorSummary != "（无错误日志）" {
				stateParts = append(stateParts,
					"## 项目中待处理的错误\n"+errorSummary+
						"\n注意：以上是检测到的错误。请分析并修复它们。")
			}
		}

		// 4. 注入所有执行状态统计
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
	}

	reg.Register(&agent.Tool{
		Name:        "ask_user",
		Description: "向用户提问，等待用户回答",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"question": map[string]any{"type": "string", "description": "问题内容"},
			},
			"required": []string{"question"},
		},
		RequiresApproval: false,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			question, _ := args["question"].(string)
			if question == "" {
				question = "（无问题内容）"
			}
			session.askDone = false
			select {
			case answer := <-session.askCh:
				return strings.TrimSpace(answer), nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		},
	})

	loop := &agent.Loop{
		Provider:         prov,
		Registry:         reg,
		Autonomous:       req.Autonomous,
		System:           sys,
		MaxIterations:    core.Settings.MaxIterations,
		MaxContextTokens: core.Settings.ContextMaxTokens,
		Compressor:       buildWebCompressor(),
		OnEvent: func(e agent.Event) {
			select {
			case session.events <- e:
			default:
			}
		},
		Approve: func(ctx context.Context, tc agent.ToolCall) (bool, string) {
			if core.Settings.RequireApproval {
				session.approvalCallID = tc.ID
				select {
				case session.events <- agent.Event{
					Type:   agent.EventApproval,
					Tool:   tc.Function.Name,
					Args:   tc.Function.Arguments,
					CallID: tc.ID,
				}:
				default:
				}
				select {
				case approved := <-session.approvalCh:
					return approved, ""
				case <-ctx.Done():
					session.approvalCallID = ""
					return false, "用户取消了操作"
				}
			}
			return true, ""
		},
	}

	// ── 构建对话上下文（Agent 自闭环管理历史） ──
	// 设计：loop.History 跨请求持久化对话历史，前端只发信号（task 文本）。
	// 有缓存历史时，直接设置 loop.History 并传 nil——loop.Run 内部使用 l.History
	// 并在末尾追加当前用户消息（task），再由 defer 自动更新 l.History。
	// 首次加载时从后端 conversations 存储重建历史。
	var history []agent.Message
	s.mu.Lock()
	cached, ok := s.historyCache[req.ConvID]
	s.mu.Unlock()
	if ok {
		// 自闭环模式：设置 loop.History 后传 nil，loop.Run 内部使用 l.History
		loop.History = cached
		history = nil
	} else {
		// 首次加载：从后端 conversations 存储重建（BuildHistory 已排除末条用户消息）
		history = s.loadConversationHistory(req.ConvID)
	}

	taskText := req.Message
	maxIter := loop.MaxIterations
	var restoredExecState *agent.ExecutionState
	if req.Autonomous {
		taskText += "\n\n（自主模式：先用 update_plan 列出完整计划，然后连续完成所有步骤、全部完成后调用 finish_task 工具。）"
		if maxIter <= 0 {
			maxIter = 60
		} else {
			maxIter *= 2
		}
		// ── 断点续跑检测：检查是否有中断的执行（不限定 ConvID，跨对话也可恢复） ──
		if root != "" {
			execMgr := agent.InitExecStateManager(root)
			interrupted := execMgr.FindInterrupted()
			if interrupted != nil {
				summary := fmt.Sprintf("发现未完成的任务（运行 ID: %s），将在本轮继续执行。\n当前进度: 第 %d/%d 轮\n阶段: %s\n修改的文件: %d 个\n错误: %d 个",
					interrupted.ID, interrupted.LoopCount, interrupted.MaxLoops, interrupted.Phase, len(interrupted.ModifiedFiles), len(interrupted.Errors))
				select {
				case session.events <- agent.Event{Type: agent.EventNotice, Content: summary}:
				default:
				}
				// 用保存的任务文本替换用户消息
				if interrupted.MissionTask != "" {
					taskText = interrupted.MissionTask
				}
				// 保存恢复的 state 引用，传递给编排循环
				restoredExecState = interrupted
			}
		}
	}
	loop.MaxIterations = maxIter

	done := make(chan struct{})
	go func() {
		defer close(done)
		if req.Autonomous {
			finalMsgs := runOrchestrationLoop(ctx, prov, reg, session, taskText, history, core.Folders, root, req.ConvID, restoredExecState)
			if req.ConvID != "" && len(finalMsgs) > 0 && !session.stopped {
				s.mu.Lock()
				s.historyCache[req.ConvID] = finalMsgs
				s.mu.Unlock()
				s.saveHistoryCache()
			}
			// 自主模式完成后也生成摘要
			if req.ConvID != "" {
				go generateConversationSummary(req.ConvID, buildWebCompressor())
			}
			return
		}

		msgs, err := loop.Run(ctx, taskText, history)
		// 持久化完整对话历史（供下轮复用，替代 loadConversationHistory）
		if req.ConvID != "" && !session.stopped {
			s.mu.Lock()
			s.historyCache[req.ConvID] = msgs
			s.mu.Unlock()
			s.saveHistoryCache()
		}
		if err != nil && !session.stopped {
			select {
			case session.events <- agent.Event{Type: agent.EventError, Content: err.Error()}:
			default:
			}
		}
	}()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	eventDone := false
	for !eventDone {
		select {
		case e, ok := <-session.events:
			if !ok {
				eventDone = true
				break
			}
			payload := map[string]any{
				"type":       string(e.Type),
				"content":    e.Content,
				"tool":       e.Tool,
				"args":       e.Args,
				"callId":     e.CallID,
				"doneReason": e.DoneReason,
			}
			if e.Usage != nil {
				payload["usage"] = e.Usage
			if e.Usage != nil {
				payload["usage"] = e.Usage
				// 持久化当前对话的上下文统计（含 breakdown 明细）
				if req.ConvID != "" {
						if conversations[i].ID == req.ConvID {
							conversations[i].TokenUsage = &ConversationTokenUsage{
								PromptTokens:     e.Usage.PromptTokens,
								CompletionTokens: e.Usage.CompletionTokens,
								TotalTokens:      e.Usage.PromptTokens + e.Usage.CompletionTokens,
								CacheHitTokens:   e.Usage.PromptCacheHitTokens,
								CacheMissTokens:  e.Usage.PromptCacheMissTokens,
								SystemTokens:     e.Usage.SystemTokens,
								SkillsTokens:     e.Usage.SkillsTokens,
								MCPTokens:        e.Usage.MCPTokens,
								ToolTokens:       e.Usage.ToolTokens,
								HistoryTokens:    e.Usage.HistoryTokens,
								OtherTokens:      e.Usage.OtherTokens,
							}
							break
						}
					}
					saveConversations()
				}
			}
		case <-heartbeat.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			eventDone = true
		case <-done:
			drainStart := time.Now()
		drainLoop:
			for time.Since(drainStart) < 2*time.Second {
				select {
				case e, ok := <-session.events:
					if !ok {
						eventDone = true
						break drainLoop
					}
					payload := map[string]any{
						"type":       string(e.Type),
						"content":    e.Content,
						"tool":       e.Tool,
						"args":       e.Args,
						"callId":     e.CallID,
						"doneReason": e.DoneReason,
					}
					fmt.Fprintf(w, "data: %s\n\n", jsonStr(payload))
					flusher.Flush()
				default:
					eventDone = true
					break drainLoop
				}
			}
			eventDone = true
		}
	}

	fmt.Fprintf(w, "data: %s\n\n", jsonStr(map[string]any{"type": "done"}))
	flusher.Flush()
}

func (s *webServer) handleChatStop(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	s.mu.Lock()
	sess, ok := s.activeLoops[sessionID]
	if ok {
		sess.stopped = true
		sess.cancel()
		delete(s.activeLoops, sessionID)
	}
	s.mu.Unlock()
	jsonResp(w, map[string]any{"ok": ok})
}

func (s *webServer) handleChatAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		SessionID string `json:"sessionId"`
		Answer    string `json:"answer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.SessionID == "" {
		jsonErr(w, "sessionId 必填")
		return
	}
	s.mu.Lock()
	sess, ok := s.activeLoops[req.SessionID]
	s.mu.Unlock()
	if !ok || sess.askCh == nil {
		jsonErr(w, "会话不存在或未在等待回答")
		return
	}
	select {
	case sess.askCh <- req.Answer:
		jsonResp(w, map[string]any{"ok": true})
	default:
		jsonErr(w, "回答通道已满或已关闭")
	}
}

func (s *webServer) handleChatApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		SessionID string `json:"sessionId"`
		CallID    string `json:"callId"`
		Approved  bool   `json:"approved"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.SessionID == "" || req.CallID == "" {
		jsonErr(w, "sessionId 和 callId 必填")
		return
	}
	s.mu.Lock()
	sess, ok := s.activeLoops[req.SessionID]
	if !ok || sess.approvalCh == nil || sess.approvalCallID != req.CallID {
		s.mu.Unlock()
		jsonErr(w, "会话不存在或 callID 不匹配")
		return
	}
	sess.approvalCallID = ""
	s.mu.Unlock()
	select {
	case sess.approvalCh <- req.Approved:
		jsonResp(w, map[string]any{"ok": true})
	default:
		jsonErr(w, "审批通道已满或已关闭")
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

// ─── 自动构建验证 ──────────────────────────────────────────

type verifyResult struct {
	success bool
	output  string
}

// autoVerifyProject 自动检测项目类型并运行构建验证。
// 返回验证结果（成功/失败 + 输出）。
func autoVerifyProject(root string) verifyResult {
	if root == "" {
		return verifyResult{success: true, output: "（无工作区，跳过验证）"}
	}

	// 检测 Go 项目
	goModPath := filepath.Join(root, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		// 1. go vet
		ctx1, cancel1 := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel1()
		vet := exec.CommandContext(ctx1, "go", "vet", "-tags", "webonly", "./cmd/companion")
		vet.Dir = root
		vetOut, vetErr := vet.CombinedOutput()
		if vetErr != nil {
			return verifyResult{success: false,
				output: fmt.Sprintf("❌ go vet 失败:\n%s", string(vetOut))}
		}

		// 2. go build
		ctx2, cancel2 := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel2()
		build := exec.CommandContext(ctx2, "go", "build", "-tags", "webonly", "./cmd/companion")
		build.Dir = root
		buildOut, buildErr := build.CombinedOutput()
		if buildErr != nil {
			return verifyResult{success: false,
				output: fmt.Sprintf("❌ Go 构建失败:\n%s", string(buildOut))}
		}

		// 3. go test（仅运行时，不阻塞主流程）
		testResult := ""
		ctx3, cancel3 := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel3()
		test := exec.CommandContext(ctx3, "go", "test", "-count=1", "-timeout", "30s", "./cmd/companion/agent")
		test.Dir = root
		testOut, testErr := test.CombinedOutput()
		if testErr != nil {
			testResult = fmt.Sprintf("\n⚠️ 测试有失败:\n%s", string(testOut))
		} else {
			testResult = "\n✅ 测试通过"
		}

		return verifyResult{success: true,
			output: fmt.Sprintf("✅ Go 验证通过 (vet+build)%s", testResult)}
	}

	// 检测 Node.js 项目
	packagePath := filepath.Join(root, "package.json")
	if _, err := os.Stat(packagePath); err == nil {
		// 1. TypeScript 类型检查
		ctx1, cancel1 := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel1()
		tsc := exec.CommandContext(ctx1, "npx", "tsc", "--noEmit")
		tsc.Dir = root
		tscOut, tscErr := tsc.CombinedOutput()
		if tscErr == nil {
			// 2. 构建
			ctx2, cancel2 := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel2()
			vite := exec.CommandContext(ctx2, "npx", "vite", "build")
			vite.Dir = root
			viteOut, viteErr := vite.CombinedOutput()
			if viteErr != nil {
				return verifyResult{success: false,
					output: fmt.Sprintf("❌ Vite 构建失败:\n%s", string(viteOut))}
			}

			// 3. npm test（如果可用，快速运行）
			testResult := ""
			ctx3, cancel3 := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel3()
			npmTest := exec.CommandContext(ctx3, "npx", "--yes", "vitest", "run", "--reporter=verbose")
			npmTest.Dir = root
			testOut, testErr := npmTest.CombinedOutput()
			if testErr == nil {
				testResult = "\n✅ 前端测试通过"
			} else {
				// 不因为测试失败阻塞整个验证，仅记录
				testResult = fmt.Sprintf("\n⚠️ 前端测试结果:\n%s", string(testOut))
			}

			return verifyResult{success: true,
				output: fmt.Sprintf("✅ TypeScript + Vite 通过%s", testResult)}
		}
		// 仅构建（tsc 可能因为类型缺失而失败）
		ctx2, cancel2 := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel2()
		vite := exec.CommandContext(ctx2, "npx", "vite", "build")
		vite.Dir = root
		viteOut, viteErr := vite.CombinedOutput()
		if viteErr == nil {
			return verifyResult{success: true,
				output: "✅ Vite 构建通过（tsc 跳过）"}
		}
		return verifyResult{success: false,
			output: fmt.Sprintf("❌ 构建失败:\ntsc: %s\n\nvite: %s", string(tscOut), string(viteOut))}
	}

	// 未知项目类型，跳过自动验证
	return verifyResult{success: true, output: "（未识别的项目类型，跳过自动验证）"}
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
