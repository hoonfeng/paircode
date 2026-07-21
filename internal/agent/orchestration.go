// 编排引擎核心结构 —— 自主模式外层编排循环。
//
// 自主模式采用双层 loop 架构：
//   外层编排引擎（OrchestrationEngine）→ 阶段推进（规划→执行→验证→分析→修复→终态）
//   内层执行引擎（Loop）→ TAOR 循环（think→act→observe→repeat）
//
// OrchestrationEngine 负责：
//   - 多阶段编排（Planning→Executing→Verifying→Analyzing→Fixing→终态）
//   - 统一退出信号（OrchestrationResult）
//   - 阶段事件广播（供前端状态栏显示当前阶段）
//   - 子任务管理与错误恢复

package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ─── 阶段枚举 ──────────────────────────────────────────────

// OrchestrationPhase 编排阶段。
type OrchestrationPhase string

const (
	PhasePlanning  OrchestrationPhase = "planning"   // 规划阶段：生成步骤清单
	PhaseExecuting OrchestrationPhase = "executing"  // 执行阶段：loop.Run 内层 TAOR 循环
	PhaseVerifying OrchestrationPhase = "verifying"  // 验证阶段：构建/测试自动验证
	PhaseAnalyzing OrchestrationPhase = "analyzing"  // 分析阶段：编排 agent 判断下一步
	PhaseFixing    OrchestrationPhase = "fixing"     // 修复阶段：注入修复任务，继续执行
	PhaseDone      OrchestrationPhase = "done"       // 全部完成（终态）
	PhaseFailed    OrchestrationPhase = "failed"     // 失败终止（终态）
	PhaseCancelled OrchestrationPhase = "cancelled"  // 用户取消（终态）
)

// IsTerminal 返回 phase 是否为终态（done / failed / cancelled）。
func (p OrchestrationPhase) IsTerminal() bool {
	return p == PhaseDone || p == PhaseFailed || p == PhaseCancelled
}

// ─── 退出信号 ──────────────────────────────────────────────

// OrchestrationResult 编排引擎的退出信号——作为统一的终态出口，
// 外层（web server / bridge）据此判断结果类型并做相应处理。
type OrchestrationResult struct {
	Phase   OrchestrationPhase // 终态 phase（done / failed / cancelled）
	History []Message          // 完整对话历史（供后续断点续跑或分析）
	Summary string             // 完成摘要（用于最终通知及日志）
	Reason  string             // 完成/失败原因（done 时为成功总结，failed 时为错误描述）
	State   *ExecutionState    // 最终执行状态（含变更文件列表、步骤记录等）
}

// IsTerminal 返回结果为终态。
func (r *OrchestrationResult) IsTerminal() bool {
	return r != nil && r.Phase.IsTerminal()
}

// ─── 编排引擎 ──────────────────────────────────────────────

// OrchestrationEngine 自主模式编排引擎主体。
// 职责：编排多阶段生命周期（规划→执行→验证→分析→修复→终态），
// 管理 ExecutionState，驱动内层 Loop.Run，广播阶段事件。
type OrchestrationEngine struct {
	// ── 依赖注入（外部设置，不可变）──
	Provider     Provider   // LLM 提供方（执行 agent 用）
	Registry     *Registry  // 工具注册表（执行 agent 用）
	Planner      *Planner   // 规划 agent（可空；空=不规划，直接执行）
	Events       chan Event // 事件通道（广播阶段/事件到 UI；可空=静默）
	ApprovalCh   chan bool  // 审批通道（可空=自动放行）
	SystemPrompt string     // 系统提示词
	Root         string     // 工作区根目录
	Roots        []string   // 所有根目录列表
	ConvID       string     // 关联对话 ID

	// ── 运行参数（Run 前设置，单次运行有效）──
	Task           string          // 原始用户任务
	History        []Message       // 初始对话历史（可空）
	RestoredState  *ExecutionState // 断点续跑时传入的已保存状态（可空）
	MaxLoops       int             // 最大编排轮次（默认 10）
	MaxIterations  int             // 内层 loop 最大迭代数（默认 60）
	VerifyFunc     func(root string) (success bool, output string) // 构建验证回调（可空）
	MaxContextTokens int                                           // 上下文窗口大小（0=不压缩）
	Compressor     Provider                                        // 压缩模型（可空）
	ExecManager    *ExecStateManager                               // 执行状态管理器（自动初始化）

	// ── 运行时状态（内部管理）──
	Phase       OrchestrationPhase // 当前阶段
	State       *ExecutionState    // 当前执行状态（持久化到磁盘）
	allHistory  []Message          // 累积完整历史（伴随整个编排生命周期）
	missionTask string             // 当前执行任务文本（随阶段变化）
	loopCount   int                // 已执行编排轮数
	fixAttempts int                // 当前修复阶段内的尝试次数
	result      *OrchestrationResult // 最终结果（终态后设置）
	ctx         context.Context
}

// NewOrchestrationEngine 创建编排引擎实例。
// provider、registry 为必填；planner 可空（不规划直接执行）；
// events、approvalCh 可空（静默/自动放行）。
// root 非空时自动初始化 ExecStateManager。
func NewOrchestrationEngine(
	provider Provider,
	registry *Registry,
	planner *Planner,
	events chan Event,
	approvalCh chan bool,
	sysPrompt string,
	root string,
	roots []string,
	convID string,
) *OrchestrationEngine {
	eng := &OrchestrationEngine{
		Provider:      provider,
		Registry:      registry,
		Planner:       planner,
		Events:        events,
		ApprovalCh:    approvalCh,
		SystemPrompt:  sysPrompt,
		Root:          root,
		Roots:         roots,
		ConvID:        convID,
		MaxLoops:      10,
		MaxIterations: 60,
	}
	if root != "" {
		eng.ExecManager = InitExecStateManager(root)
	}
	return eng
}

// ─── 公共方法 ──────────────────────────────────────────────

// Run 执行编排主入口。阻塞直到任一终态（done/failed/cancelled）后返回 OrchestrationResult。
// ctx 控制超时和取消（取消→PhaseCancelled）。
func (eng *OrchestrationEngine) Run(ctx context.Context) *OrchestrationResult {
	eng.ctx = ctx
	eng.Phase = PhasePlanning
	eng.allHistory = append([]Message(nil), eng.History...)

	// 初始化 ExecManager（若未在构造函数中初始化）
	if eng.ExecManager == nil && eng.Root != "" {
		eng.ExecManager = InitExecStateManager(eng.Root)
	}
	execMgr := eng.ExecManager

	// 创建或恢复执行状态
	if eng.RestoredState != nil {
		eng.State = eng.RestoredState
		eng.loopCount = eng.RestoredState.LoopCount
		eng.Phase = OrchestrationPhase(eng.RestoredState.Phase)
	} else {
		now := time.Now().Format("2006-01-02 15:04:05")
		id := fmt.Sprintf("exec_%s_%d", time.Now().Format("20060102_150405"), time.Now().UnixMilli()%1000)
		eng.State = &ExecutionState{
			ID:          id,
			Task:        eng.Task,
			MissionTask: eng.Task,
			LoopCount:   0,
			MaxLoops:    eng.MaxLoops,
			Phase:       string(eng.Phase),
			Status:      ExecRunning,
			CreatedAt:   now,
			UpdatedAt:   now,
			ConvID:      eng.ConvID,
		}
		if execMgr != nil {
			execMgr.Save(eng.State)
		}
	}

	// 编排循环：step 返回 true 表示终止
	for {
		select {
		case <-ctx.Done():
			return eng.terminate(PhaseCancelled, "用户取消", ctx.Err().Error())
		default:
		}

		if eng.step(ctx) {
			return eng.result
		}
	}
}

// step 单步推进编排状态机。
// 返回 true = 已终止（result 已设置），false = 继续循环。
func (eng *OrchestrationEngine) step(ctx context.Context) bool {
	switch eng.Phase {
	case PhasePlanning:
		return eng.stepPlanning(ctx)
	case PhaseExecuting:
		return eng.stepExecuting(ctx)
	case PhaseVerifying:
		return eng.stepVerifying(ctx)
	case PhaseAnalyzing:
		return eng.stepAnalyzing(ctx)
	case PhaseFixing:
		return eng.stepFixing(ctx)
	default:
		// 终态（done/failed/cancelled）——不应再 step
		return true
	}
}

// ─── 单步实现 ──────────────────────────────────────────

// stepPlanning 规划阶段：初始化执行状态，调用 Planner 生成步骤清单，
// 设置文件变更回调，然后进入执行阶段。
func (eng *OrchestrationEngine) stepPlanning(ctx context.Context) bool {
	eng.emitPhase("规划阶段")

	// 初始化执行状态管理器
	if eng.ExecManager == nil && eng.Root != "" {
		eng.ExecManager = InitExecStateManager(eng.Root)
	}

	// 更新状态
	if eng.State != nil {
		eng.State.Phase = "规划阶段"
		if eng.State.Status == "" {
			eng.State.Status = ExecRunning
		}
		if eng.State.MissionTask == "" {
			eng.State.MissionTask = eng.Task
		}
		if eng.ExecManager != nil {
			eng.ExecManager.Save(eng.State)
		}
	}

	eng.missionTask = eng.Task

	// 如果有规划 agent，执行规划
	if eng.Planner != nil {
		plan, perr := eng.Planner.Plan(ctx, eng.missionTask, eng.allHistory)
		if perr == nil && len(plan.Steps) > 0 {
			// 广播规划到前端
			evtArgs := planStepsText(plan)
			eng.emit(Event{Type: EventToolCall, Tool: "update_plan", Args: evtArgs})

			// 保存规划到 .pair/tasks/
			planContent := fmt.Sprintf("# 初始规划: %s\n\n## 推理\n%s\n\n## 步骤\n%s\n\n- 创建时间: %s\n- 状态: 进行中\n",
				eng.missionTask, plan.Reasoning, planStepsText(plan), time.Now().Format("2006-01-02 15:04:05"))
			eng.saveTaskPlan(fmt.Sprintf("plan_%s", time.Now().Format("20060102_150405")), planContent)

			// 注入规划到任务文本
			eng.missionTask = eng.missionTask + "\n\n（规划 Agent 已制定以下计划，请据此连续执行、用 update_tasks 追踪各步骤进度）：\n" + planStepsText(plan)
		}
	}

	// 设置文件变更回调
	if eng.Root != "" && eng.ExecManager != nil && eng.State != nil {
		FileChangeCallback = func(filePath string) {
			if eng.ExecManager != nil && eng.State != nil {
				eng.ExecManager.RecordFileChange(eng.State, filePath)
			}
		}
	}

	// 保存使命任务到状态
	if eng.State != nil {
		eng.State.MissionTask = eng.missionTask
		if eng.ExecManager != nil {
			eng.ExecManager.Save(eng.State)
		}
	}

	eng.Phase = PhaseExecuting
	return false
}

// stepExecuting 执行阶段：构建内层 Loop 并调用 Run 执行当前使命任务。
// 处理错误（绕圈→PhaseFailed，普通错误→PhaseFailed），成功→PhaseVerifying。
func (eng *OrchestrationEngine) stepExecuting(ctx context.Context) bool {
	eng.loopCount++

	// 检测最大编排轮次
	if eng.loopCount > eng.MaxLoops {
		eng.emit(Event{Type: EventError, Content: fmt.Sprintf("已达最大编排轮次 %d，终止", eng.MaxLoops)})
		return eng.terminate(PhaseFailed, "超过最大轮次", fmt.Sprintf("已达最大编排轮次 %d", eng.MaxLoops)).IsTerminal()
	}

	eng.emitPhase(fmt.Sprintf("执行 %d/%d", eng.loopCount, eng.MaxLoops))

	// 更新状态
	if eng.State != nil {
		eng.State.LoopCount = eng.loopCount
		eng.State.Phase = fmt.Sprintf("执行 %d/%d", eng.loopCount, eng.MaxLoops)
		if eng.ExecManager != nil {
			eng.ExecManager.Save(eng.State)
		}
	}

	// 构建内层 Loop
	mainLoop := &Loop{
		Provider:         eng.Provider,
		Registry:         eng.Registry,
		Autonomous:       true,
		System:           eng.SystemPrompt,
		MaxIterations:    eng.MaxIterations,
		MaxContextTokens: eng.MaxContextTokens,
		Compressor:       eng.Compressor,
		OnEvent: func(e Event) {
			eng.emit(e)
		},
		Approve: eng.buildApproveFunc(),
	}

	// 执行内层 loop
	msgs, err := mainLoop.Run(ctx, eng.missionTask, eng.allHistory)

	// 更新累积历史
	eng.allHistory = msgs

	// 处理错误
	if err != nil {
		if errors.Is(err, ErrCirclingLoop) {
			eng.saveTaskPlan(fmt.Sprintf("plan_%s_circling", time.Now().Format("20060102_150405")),
				fmt.Sprintf("# 任务绕圈: %s\n\n## 时间\n%s\n\n## 错误\n检测到重复绕圈，已停止。\n",
					eng.Task, time.Now().Format("2006-01-02 15:04:05")))
			if eng.ExecManager != nil && eng.State != nil {
				eng.ExecManager.MarkFailed(eng.State, err)
			}
			return eng.terminate(PhaseFailed, "绕圈检测终止", err.Error()).IsTerminal()
		}
		// 其他错误
		if eng.ExecManager != nil && eng.State != nil {
			eng.ExecManager.MarkFailed(eng.State, err)
		}
		return eng.terminate(PhaseFailed, "执行错误", err.Error()).IsTerminal()
	}

	// 进入验证阶段
	eng.Phase = PhaseVerifying
	return false
}

// stepVerifying 验证阶段：执行构建/测试自动验证。
// 构建失败→最多 3 次修复尝试，超限→PhaseFailed。
// 构建通过→检查 debug 日志，然后进入分析阶段。
func (eng *OrchestrationEngine) stepVerifying(ctx context.Context) bool {
	// 如果没有验证函数或没有工作区，跳过验证
	if eng.VerifyFunc == nil || eng.Root == "" {
		eng.Phase = PhaseAnalyzing
		return false
	}

	eng.emitPhase("验证中")
	if eng.State != nil {
		eng.State.Phase = "验证中"
		if eng.ExecManager != nil {
			eng.ExecManager.Save(eng.State)
		}
	}

	// 执行构建验证
	success, output := eng.VerifyFunc(eng.Root)

	if !success {
		// ── 构建失败 ──
		WriteBuildError("autoVerifyProject", output,
			map[string]string{"convId": eng.ConvID, "loopCount": fmt.Sprintf("%d", eng.loopCount)})

		errMsg := FormatBuildErrorForAgent(output, eng.Root)

		// 发通知到前端
		eng.emit(Event{Type: EventNotice, Content: errMsg})

		eng.fixAttempts++

		if eng.fixAttempts > 3 {
			finalErrMsg := fmt.Sprintf("连续 %d 次修复尝试后构建仍未通过:\n\n%s", eng.fixAttempts, output)
			eng.emit(Event{Type: EventError, Content: finalErrMsg})
			if eng.ExecManager != nil && eng.State != nil {
				eng.ExecManager.MarkFailed(eng.State, fmt.Errorf("连续 %d 次修复尝试失败", eng.fixAttempts))
			}
			return eng.terminate(PhaseFailed, "修复失败", finalErrMsg).IsTerminal()
		}

		// 记录修复尝试到状态
		if eng.State != nil {
			eng.State.Errors = append(eng.State.Errors, fmt.Sprintf("构建失败（第 %d 次修复尝试）", eng.fixAttempts))
		}

		// 进入修复阶段
		eng.missionTask = errMsg + "\n\n（请根据上述错误分析逐条修复，每次修复后运行 go_build 验证。全部修复完成后输出最终报告。）"
		eng.Phase = PhaseFixing
		return false
	}

	// ── 构建通过 → 检查 debug 日志 ──
	eng.emitPhase("debug 分析")
	if eng.State != nil {
		eng.State.Phase = "debug 分析"
		if eng.ExecManager != nil {
			eng.ExecManager.Save(eng.State)
		}
	}

	if GlobalDebugLogger != nil {
		errorSummary := GlobalDebugLogger.GetErrorSummary(5)
		if errorSummary != "" && errorSummary != "（无错误日志）" {
			debugTaskMsg := fmt.Sprintf("检测到以下错误日志，请分析并修复:\n\n%s\n\n（分析完成后，如果不需要修复或已修复，继续推进其他任务。）", errorSummary)
			eng.emit(Event{Type: EventNotice, Content: debugTaskMsg})
			// debug 日志仅作通知，不注入对话历史（避免干扰 agent 判断）
			// 去除旧版「注入 debug 分析消息到历史」逻辑
		}
	}

	// 进入分析阶段
	eng.Phase = PhaseAnalyzing
	return false
}

// stepAnalyzing 分析阶段：编排 agent 判断下一步。
// Planner 为空→直接 PhaseDone。
// 分析得出「全部完成」→ PhaseDone，否则注入下一步任务→ PhaseExecuting。
func (eng *OrchestrationEngine) stepAnalyzing(ctx context.Context) bool {
	// 如果没有编排 agent，视为完成
	if eng.Planner == nil {
		eng.emit(Event{Type: EventDone, Content: fmt.Sprintf("任务完成（自动模式）\n\n完成轮次: %d", eng.loopCount), DoneReason: "task_complete"})
		if eng.ExecManager != nil && eng.State != nil {
			eng.ExecManager.MarkCompleted(eng.State, "任务完成（无编排 agent）")
		}
		return eng.terminate(PhaseDone, "任务完成", "无编排 agent，自动结束").IsTerminal()
	}

	eng.emitPhase("分析完成情况")
	if eng.State != nil {
		eng.State.Phase = "分析完成情况"
		if eng.ExecManager != nil {
			eng.ExecManager.Save(eng.State)
		}
	}

	// 用编排 agent 分析已完成内容并决定下一步
	analysisPrompt := fmt.Sprintf(`你执行了任务。请分析本轮结果并决定下一步。

总体任务: %s

如果全部完成，请回复：全部完成
如果还有下一步，请回复：下一步任务：<具体描述>`, eng.Task)

	analysis, aerr := eng.Planner.Plan(ctx, analysisPrompt, eng.allHistory)
	nextTask := ""
	if aerr == nil && len(analysis.Steps) > 0 {
		nextTask = analysis.Reasoning
		// 尝试解析"下一步任务:"前缀
		prefixes := []string{"下一步任务：", "下一步：", "Next task: ", "Next: "}
		for _, p := range prefixes {
			if idx := strings.Index(nextTask, p); idx >= 0 {
				nextTask = strings.TrimSpace(nextTask[idx+len(p):])
				break
			}
		}
	}

	// 判断是否全部完成
	if nextTask == "" || strings.Contains(nextTask, "全部完成") || strings.Contains(strings.ToLower(nextTask), "all complete") {
		eng.emitPhase("全部完成")
		eng.emit(Event{Type: EventDone, Content: fmt.Sprintf("全部任务已完成。\n\n完成轮次: %d\n总任务: %s\n", eng.loopCount, eng.Task), DoneReason: "task_complete"})
		eng.saveTaskPlan(fmt.Sprintf("plan_%s_complete", time.Now().Format("20060102_150405")),
			fmt.Sprintf("# 任务完成: %s\n\n## 完成时间\n%s\n\n## 完成摘要\n全部任务已完成。\n", eng.Task, time.Now().Format("2006-01-02 15:04:05")))
		if eng.ExecManager != nil && eng.State != nil {
			eng.ExecManager.MarkCompleted(eng.State, "全部任务已完成")
		}
		return eng.terminate(PhaseDone, "全部任务已完成", "").IsTerminal()
	}

	// 有下一步任务
	currentTask := eng.missionTask
	if eng.loopCount > 1 {
		currentTask = fmt.Sprintf("第 %d 轮任务", eng.loopCount)
	}

	planContent := fmt.Sprintf("# 任务规划: %s\n\n## 当前轮次\n第 %d 轮\n\n## 已完成\n%s\n\n## 下一步\n%s\n\n- 时间: %s\n",
		eng.Task, eng.loopCount, currentTask, nextTask, time.Now().Format("2006-01-02 15:04:05"))
	eng.saveTaskPlan(fmt.Sprintf("plan_%s_r%02d", time.Now().Format("20060102_150405"), eng.loopCount), planContent)

	// 记录完成的步骤
	if eng.State != nil {
		eng.State.CompletedSteps = append(eng.State.CompletedSteps, StepRecord{
			StepNum:     eng.loopCount,
			Description: currentTask,
			Status:      "completed",
			CompletedAt: time.Now().Format("2006-01-02 15:04:05"),
			Summary:     fmt.Sprintf("完成 → 下一步: %s", nextTask),
		})
	}

	eng.emitPhase("继续下一步")

	// 注入下一步任务
	eng.missionTask = nextTask + "\n\n（请基于已完成的上下文，继续执行上述下一步任务。完成后输出最终报告。）"
	if eng.State != nil {
		eng.State.MissionTask = eng.missionTask
		if eng.ExecManager != nil {
			eng.ExecManager.Save(eng.State)
		}
	}

	eng.Phase = PhaseExecuting
	return false
}

// stepFixing 修复阶段：注入修复任务，让内层 loop 继续执行。
// missionTask 已在 stepVerifying 中设置好，直接转 PhaseExecuting。
func (eng *OrchestrationEngine) stepFixing(ctx context.Context) bool {
	eng.emitPhase("修复中")
	if eng.State != nil {
		eng.State.Phase = "修复中"
		eng.State.MissionTask = eng.missionTask
		if eng.ExecManager != nil {
			eng.ExecManager.Save(eng.State)
		}
	}
	eng.Phase = PhaseExecuting
	return false
}

// ─── 辅助方法 ──────────────────────────────────────────────

// buildApproveFunc 构建内层 Loop 的审批回调。
// ApprovalCh==nil 时返回 nil（自动放行），否则返回阻塞等待审批通道的函数。
func (eng *OrchestrationEngine) buildApproveFunc() func(ctx context.Context, tc ToolCall) (bool, string) {
	if eng.ApprovalCh == nil {
		return nil // 自动放行
	}
	return func(ctx context.Context, tc ToolCall) (bool, string) {
		// 发送审批事件到前端
		eng.emit(Event{
			Type:   EventApproval,
			Tool:   tc.Function.Name,
			Args:   tc.Function.Arguments,
			CallID: tc.ID,
		})
		select {
		case approved := <-eng.ApprovalCh:
			return approved, ""
		case <-ctx.Done():
			return false, "用户取消了操作"
		}
	}
}

// terminate 设置终态并返回 OrchestrationResult。
func (eng *OrchestrationEngine) terminate(phase OrchestrationPhase, summary, reason string) *OrchestrationResult {
	eng.Phase = phase
	eng.result = &OrchestrationResult{
		Phase:   phase,
		History: eng.allHistory,
		Summary: summary,
		Reason:  reason,
		State:   eng.State,
	}
	if eng.State != nil {
		eng.State.Phase = string(phase)
	}
	eng.emitPhase(string(phase))
	return eng.result
}

// emitPhase 非阻塞广播阶段切换事件。
func (eng *OrchestrationEngine) emitPhase(phase string) {
	if eng.Events != nil {
		select {
		case eng.Events <- Event{Type: EventPhase, Content: phase}:
		default:
		}
	}
}

// emit 非阻塞发送通用事件。
func (eng *OrchestrationEngine) emit(e Event) {
	if eng.Events != nil {
		select {
		case eng.Events <- e:
		default:
		}
	}
}

// saveTaskPlan 保存规划到 .pair/tasks/ 目录下。
// name 为文件名（不含扩展名），content 为规划内容文本。
func (eng *OrchestrationEngine) saveTaskPlan(name, content string) {
	if eng.Root == "" {
		return
	}
	dir := filepath.Join(eng.Root, ".pair", "tasks")
	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, name+".md")
	os.WriteFile(path, []byte(content), 0644)
}

// ─── 工具函数 ──────────────────────────────────────────────

// planStepsText 将 Plan 的 steps 列表转为可读的多行文本。
func planStepsText(plan Plan) string {
	var b strings.Builder
	for i, step := range plan.Steps {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("%d. %s", i+1, step.Description))
	}
	return b.String()
}
