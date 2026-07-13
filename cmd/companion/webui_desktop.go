// 桌面版额外路由注册：chat/agent + market。
//
//go:build windows && !webonly

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
	"github.com/hoonfeng/paircode/cmd/companion/agenttools"
	"github.com/hoonfeng/paircode/cmd/companion/bridge"
	"github.com/hoonfeng/paircode/cmd/companion/core"
	"github.com/hoonfeng/paircode/cmd/companion/roleprompts"
	marketplacepanel "github.com/hoonfeng/paircode/cmd/companion/ui/marketplace"
	mcppanel "github.com/hoonfeng/paircode/cmd/companion/ui/mcp"
	"github.com/hoonfeng/paircode/cmd/companion/ui/skills"
	"github.com/hoonfeng/paircode/pkg/memory"
)

func registerExtraHandlers(mux *http.ServeMux, s *webServer) {
	mux.HandleFunc("/api/chat/send", s.handleChatSend)
	mux.HandleFunc("/ws", s.handleWebSocket) // WebSocket 端点（替代 SSE /api/chat/events）
	mux.HandleFunc("/api/chat/stop", s.handleChatStop)
	mux.HandleFunc("/api/chat/answer", s.handleChatAnswer)
	mux.HandleFunc("/api/chat/approve", s.handleChatApprove)
	mux.HandleFunc("/api/chat/feedback", s.handleChatFeedback)
	mux.HandleFunc("/api/marketplace/search", s.handleMarketplaceSearch)
	mux.HandleFunc("/api/marketplace/install", s.handleMarketplaceInstall)
	mux.HandleFunc("/api/marketplace/refresh", s.handleMarketplaceRefresh)

	// 长时记忆检索 API
	mux.HandleFunc("/api/memory/search", s.handleMemorySearch)
	mux.HandleFunc("/api/memory/list", s.handleMemoryList)
	mux.HandleFunc("/api/memory/rebuild", s.handleMemoryRebuild)
}

// emitPhase 发送阶段切换事件到事件通道。
func emitPhase(events chan agent.Event, phase string) {
	select {
	case events <- agent.Event{Type: agent.EventPhase, Content: phase}:
	default:
	}
}

// runOrchestrationLoop 外部编排循环。
// 每次 loop.Run 完成后，用编排 agent 分析已完成内容并决定下一个任务，
// 将规划记录到 .pair/tasks/ 目录，然后继续执行，直到编排 agent 判定全部完成。
// 支持执行状态持久化和断点续跑。
// restoredState 可选：断点续跑时传入中断的执行状态，避免从头开始。
//
//nolint:unused // 暂未调用，后续迭代启用自主编排模式
func runOrchestrationLoop(ctx context.Context, prov agent.Provider, reg *agent.Registry, events chan agent.Event, approvalCh chan bool, task string, history []agent.Message, roots []string, root string, convID string, restoredState *agent.ExecutionState) []agent.Message {
	emitPhase(events, "自主执行中")

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
		emitPhase(events, "规划阶段")
		execState.Phase = "规划阶段"
		execMgr.Save(execState)
		plan, perr := planner.Plan(ctx, missionTask, history)
		if perr == nil && len(plan.Steps) > 0 {
			evtArgs := planToUpdateArgs(plan)
			select {
			case events <- agent.Event{Type: agent.EventToolCall, Tool: "update_plan", Args: evtArgs}:
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
			Compressor:       bridge.BuildCompressor(),
			OnEvent: func(e agent.Event) {
				select {
				case events <- e:
				default:
				}
			},
			Approve: func(ctx context.Context, tc agent.ToolCall) (bool, string) {
				if core.Settings.RequireApproval {
					select {
					case events <- agent.Event{
						Type:   agent.EventApproval,
						Tool:   tc.Function.Name,
						Args:   tc.Function.Arguments,
						CallID: tc.ID,
					}:
					default:
					}
					select {
					case approved := <-approvalCh:
						return approved, ""
					case <-ctx.Done():
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
			return allHistory
		}

		emitPhase(events, fmt.Sprintf("执行 %d/%d", loopCount, maxLoops))
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
			emitPhase(events, "验证中")
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
				case events <- agent.Event{Type: agent.EventNotice, Content: errMsg}:
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
					case events <- agent.Event{Type: agent.EventError, Content: finalErrMsg}:
					default:
					}
					execMgr.MarkFailed(execState, fmt.Errorf("连续 %d 次修复尝试失败", fixAttempts))
					return allHistory
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
			emitPhase(events, "debug 分析")
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
				case events <- agent.Event{Type: agent.EventNotice, Content: debugTaskMsg}:
				default:
				}
				// 注入 debug 分析任务（不中断主任务流，作为额外提示）
				msgs = append(msgs, agent.Message{Role: "user", Content: debugTaskMsg})
			}
		}

		if err == nil {
			// ── loop 正常完成 → 用编排 agent 分析下一步 ──
			emitPhase(events, "分析完成情况")
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
					emitPhase(events, "全部完成")
					// 发射最终的 EventDone（含完成摘要），供前端展示完成报告
					select {
					case events <- agent.Event{Type: agent.EventDone, Content: fmt.Sprintf("全部任务已完成。\n\n完成轮次: %d\n总任务: %s\n", loopCount, task), DoneReason: "finish_task"}:
					default:
					}
					saveTaskPlan(fmt.Sprintf("plan_%s_complete", time.Now().Format("20060102_150405")),
						fmt.Sprintf("# 任务完成: %s\n\n## 完成时间\n%s\n\n## 完成摘要\n全部任务已完成。\n", task, time.Now().Format("2006-01-02 15:04:05")))
					execMgr.MarkCompleted(execState, "全部任务已完成")
					return msgs
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

				emitPhase(events, "继续下一步")
				missionTask = nextTask + "\n\n（请基于已完成的上下文，继续执行上述下一步任务。完成后调用 finish_task。）"
				execState.MissionTask = missionTask
				execMgr.Save(execState)
				allHistory = msgs
				continue
			}
			// 没有编排 agent → 直接结束
			select {
			case events <- agent.Event{Type: agent.EventDone, Content: fmt.Sprintf("任务完成（自动模式）\n\n完成轮次: %d", loopCount), DoneReason: "task_complete"}:
			default:
			}
			execMgr.MarkCompleted(execState, "任务完成（无编排 agent）")
			return msgs
		}

		if errors.Is(err, agent.ErrCirclingLoop) {
			saveTaskPlan(fmt.Sprintf("plan_%s_circling", time.Now().Format("20060102_150405")),
				fmt.Sprintf("# 任务绕圈: %s\n\n## 时间\n%s\n\n## 错误\n检测到重复绕圈，已停止。\n",
					task, time.Now().Format("2006-01-02 15:04:05")))
			execMgr.MarkFailed(execState, err)
			select {
			case events <- agent.Event{Type: agent.EventError, Content: fmt.Sprintf("自主模式异常终止: %v", err)}:
			default:
			}
			return allHistory
		}

		// 其他错误
		execMgr.MarkFailed(execState, err)
		return allHistory
	}

	// 达到最大轮次
	emitPhase(events, "达到最大执行轮次")
	execState.Status = agent.ExecFailed
	execState.Errors = append(execState.Errors, fmt.Sprintf("已达最大执行轮次 %d", maxLoops))
	execMgr.Save(execState)
	select {
	case events <- agent.Event{Type: agent.EventError, Content: fmt.Sprintf("自主模式已达最大执行轮次 %d，终止", maxLoops)}:
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

// ─── Chat / Agent SSE API ───────────────────────────────────

// buildWebLoopOpts 构建 agent.LoopOpts（收敛 provider/registry/system/history 构造逻辑）。
// 从原 handleChatSend 中提取，供非阻塞启动调用 agentMgr.Start 使用。
func (s *webServer) buildWebLoopOpts(convID, message string, autonomous bool) agent.LoopOpts {
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

		// 5. 注入最近对话的摘要（跨对话上下文感知）
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
	//
	// 历史中含上一轮的系统消息，和当前系统提示前缀相同——保留它以最大化 LLM 缓存命中
	// （连续请求共享相同 prompt 前缀时命中 KV-cache，大幅降低延迟与成本）。
	var history []agent.Message
	if convID != "" {
		if store := agentMgr.Store(); store != nil {
			raw, _ := store.LoadAll(convID)
			if raw != nil {
				history = make([]agent.Message, len(raw))
				for i := range raw {
					history[i] = raw[i]
					history[i].Reasoning = ""
				}
			}
		}
	}
	// 裁剪中断会话（用户停止）产生的不完整 assistant 消息，确保新消息有清晰分界
	history = agent.TrimInterruptedHistory(history)

	// ── 最大迭代数（自主模式翻倍） ──
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
		Compressor:       bridge.BuildCompressor(),
		History:          history,
		Autonomous:       autonomous,
	}
}

// handleChatSend 启动一次 agent 会话（非阻塞）。
// 构建 LoopOpts 后调用 agentMgr.Start 立即返回；前端通过全局 WebSocket /ws 接收事件流。
func (s *webServer) handleChatSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	var req struct {
		Message       string `json:"message"`
		SessionID     string `json:"sessionId"` // 保留兼容但不再使用
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
	// 兜底：若前端未传 workspaceRoot，用当前核心工作区
	if req.WorkspaceRoot == "" {
		req.WorkspaceRoot = core.Root()
	}

	if !core.Configured() {
		jsonErr(w, "未配置 API key。请在设置面板中配置 API Key 和模型。")
		return
	}

	// 持久化用户消息到 MessageStore（SessionManager.Start 内部会调 store.CreateConversation 若不存在）
	agentMgr.AppendPersistedUserMessage(req.ConvID, req.Message)

	// 构建 LoopOpts（provider/registry/system/history 全部收敛于此）
	opts := s.buildWebLoopOpts(req.ConvID, req.Message, req.Autonomous)
	opts.WorkspaceRoot = req.WorkspaceRoot

	// 审核开关：只需设 AutoReview 和 ReviewProvider，审核决策由 Loop 内部自决
	opts.AutoReview = core.Settings.AutoReview
	if core.Settings.AutoReview && core.Settings.ReviewModel != "" {
		opts.ReviewProvider = bridge.BuildReviewProvider()
	}
	// 自动 git 提交开关
	opts.AutoCommit = core.Settings.AutoCommit

	// 自主模式追加任务指引
	taskText := req.Message
	if req.Autonomous {
		taskText += "\n\n（自主模式：先用 update_plan 列出完整计划，然后连续完成所有步骤、全部完成后调用 finish_task 工具。）"
	}

	// 非阻塞启动：agentMgr.Start 内部 goroutine 跑 loop.Run，立即返回
	ctx := context.Background()
	if err := agentMgr.Start(ctx, req.ConvID, taskText, opts); err != nil {
		jsonErr(w, err.Error())
		return
	}

	// 立即返回（不等 agent 完成）
	jsonResp(w, map[string]any{"ok": true, "convId": req.ConvID})
}

// startEventPersistWorker 已简化：消息写盘由 Session goroutine 在 loop.Run 返回后直接完成。
// web 层只需设置 OnDone 回调——agent 在写盘后调用以生成对话摘要。
func (s *webServer) startEventPersistWorker() {
	agentMgr.OnDone = func(convID string) {
		go generateConversationSummary(convID, bridge.BuildCompressor())
	}
}

// handleChatStop 停止指定会话的 agent 运行。
func (s *webServer) handleChatStop(w http.ResponseWriter, r *http.Request) {
	convID := r.URL.Query().Get("convId")
	if convID == "" {
		// 兼容旧参数名
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

// handleMarketplaceRefresh 刷新远程市场缓存。
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
