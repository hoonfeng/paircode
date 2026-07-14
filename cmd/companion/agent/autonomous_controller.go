// 全自主控制器（AutonomousController）—— 外层自主编排器，持有自己的规划工具注册表。
// 职责：
//   - 用规划 LLM + update_plan 工具做高层规划（NEXT/DONE 模式）
//   - 每次迭代生成具体子任务 → 交给内层 Loop 执行
//   - 收集结果、验证、迭代
// 与内层 Loop 隔离：外层只有 plan 工具（update_plan），内层有 task 工具及全部执行工具。
//
// 复刻参考：F:\syproject\伴随式codeagent\src\agent\autonomous-loop.ts

package agent

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// ── 编辑距离计算（用于任务相似度检测，防重复） ──

// editDistance 计算两个字符串的编辑距离（Levenshtein Distance）。
func editDistance(a, b string) int {
	m, n := len(a), len(b)
	if m == 0 {
		return n
	}
	if n == 0 {
		return m
	}
	prev := make([]int, n+1)
	curr := make([]int, n+1)
	for j := 0; j <= n; j++ {
		prev[j] = j
	}
	for i := 1; i <= m; i++ {
		curr[0] = i
		for j := 1; j <= n; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = min3(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[n]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// taskSimilarity 计算两个任务字符串的归一化相似度（0~1），1 表示完全一致。
func taskSimilarity(a, b string) float64 {
	if a == "" && b == "" {
		return 1
	}
	if a == "" || b == "" {
		return 0
	}
	// 规范化：去除非核心字符，小写，截断到 100 字符
	norm := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r >= 0x4e00 {
				b.WriteRune(r)
			} else if r >= 'A' && r <= 'Z' {
				b.WriteRune(r - 'A' + 'a')
			}
		}
		ns := b.String()
		if len(ns) > 100 {
			ns = ns[:100]
		}
		return ns
	}
	sa, sb := norm(a), norm(b)
	if sa == "" && sb == "" {
		return 1
	}
	if sa == "" || sb == "" {
		return 0
	}
	dist := editDistance(sa, sb)
	maxLen := math.Max(float64(len(sa)), float64(len(sb)))
	if maxLen > 0 {
		return 1 - float64(dist)/maxLen
	}
	return 1
}

// ── 任务生成提示词 ──

const autoTaskGenPrompt = `你是任务拆分器。根据用户目标和已完成的工作，生成下一个具体、可执行的任务。

# 核心原则：以目标驱动，不做无关操作
首先分析用户目标是什么行为类型，只生成与之直接相关的任务：
- 测试 → 编写测试代码、安装依赖、运行测试、修复失败、验证通过
- 开发 → 编写功能代码、修改文件、实现特性
- 调试 → 定位问题、分析根因、修复 Bug
- 部署 → 配置环境、构建产物、启动服务
- 重构 → 提取函数、拆分模块、优化结构

# 分解原则
- 每个任务必须具体：包含动作 + 目标文件/模块 + 预期产出
- 一次只做一件事，任务粒度控制在单个 Agent 能在 3-5 步内完成
- 任务之间保持连贯性：前一步的产出是下一步的输入

# 禁止事项
- ❌ 不要生成"编写 README"、"创建文档"之类的文档任务——除非用户明确要求
- ❌ 不要生成与已完成任务内容重叠的任务

# 输出格式
如果是下一个任务：NEXT: 具体任务描述
如果所有目标已完成：DONE: 完成总结

# 终止条件
- 所有用户目标已达成 → DONE
- 连续 3 次失败或驳回 → DONE（说明受阻原因）
- 无法提出与已完成任务明显不同的新任务 → DONE

# 输出格式（严格）
NEXT: <下一个任务描述>
或
DONE: <完成原因>`

// ── 类型定义 ──

// TaskRunner 内层执行器的接口。AutonomousController 调用它以运行每个子任务。
// 返回最终输出文本和可能的错误。
type TaskRunner func(ctx context.Context, task string) (string, error)

// AutonomousConfig 自主控制器的配置。
type AutonomousConfig struct {
	MaxIterations int    // 最大迭代轮数
	SystemPrompt  string // 系统提示（含工具集描述等）
	PlanModel     string // 规划模型名（仅用于日志/调试）
}

// IterationRecord 单轮迭代的记录。
type IterationRecord struct {
	Index    int    // 迭代序号
	Task     string // 本轮任务
	Output   string // 执行输出摘要
	Duration time.Duration
}

// AutonomousController 外层自主编排器。
// 持有自己的规划 LLM 和计划工具（update_plan），通过 TaskRunner 调用内层 Loop 执行任务。
type AutonomousController struct {
	Provider Provider        // 规划用 LLM Provider
	Registry *Registry       // 外层工具注册表（仅含 update_plan 等 plan 工具）
	Config   AutonomousConfig

	aborted bool
	mu      sync.Mutex
	history []string // 已完成的迭代历史描述
}

// NewAutonomousController 创建 AutonomousController。
// planProvider 为规划用的 LLM Provider（建议 non-thinking 模型）。
// planReg 为外层工具注册表（应只含 update_plan 等 plan 工具）。
func NewAutonomousController(planProvider Provider, planReg *Registry, config AutonomousConfig) *AutonomousController {
	return &AutonomousController{
		Provider: planProvider,
		Registry: planReg,
		Config:   config,
	}
}

// Stop 停止自主模式运行（下一轮迭代检测到后退出）。
func (ac *AutonomousController) Stop() {
	ac.mu.Lock()
	ac.aborted = true
	ac.mu.Unlock()
}

// IsAborted 返回是否已停止。
func (ac *AutonomousController) IsAborted() bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	return ac.aborted
}

// Run 运行自主模式完整循环：生成任务 → 执行 → 收集结果 → 迭代。
// goal: 用户输入的原始目标。
// runner: 内层 Loop 的执行器（由桥接层提供，绑定到具体的内层 Loop）。
// onEvent: 事件回调（用于 UI 流式展示）。
// 返回最终报告文本。
func (ac *AutonomousController) Run(ctx context.Context, goal string, runner TaskRunner, onEvent func(Event)) (string, error) {
	ac.mu.Lock()
	ac.aborted = false
	ac.history = nil
	ac.mu.Unlock()

	maxIter := ac.Config.MaxIterations
	if maxIter <= 0 {
		maxIter = 15
	}

	var allRecords []IterationRecord

	for iter := 1; iter <= maxIter; iter++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if ac.IsAborted() {
			break
		}

		// ── Step 1: 生成下一个任务 ──
		decision, err := ac.generateNextTask(ctx, goal)
		if err != nil {
			onEvent(Event{Type: EventError, Content: fmt.Sprintf("任务生成失败: %v", err)})
			break
		}

		if decision.done != "" {
			onEvent(Event{Type: EventNotice, Content: fmt.Sprintf("自主完成：%s", decision.done)})
			break
		}

		task := decision.task
		onEvent(Event{Type: EventPhase, Content: fmt.Sprintf("执行阶段 %d/%d", iter, maxIter)})

		// 通知 UI：更新计划步骤（用 update_plan 事件通知前端）
		planJSON := fmt.Sprintf(`{"plan":[{"step":"%s","status":"in_progress"}]}`, escapeJSON(task))
		onEvent(Event{Type: EventToolCall, Tool: "update_plan", Args: planJSON})

		// ── Step 2: 构建任务上下文（含历史迭代信息，防重复执行） ──
		historyCtx := ac.buildHistoryContext()

		contextTask := fmt.Sprintf(
			"%s[当前任务: 迭代 %d]\n%s\n\n▸ 上图是前 %d 轮已完成的工作。不要重复执行其中任何一个。\n▸ 只专注于当前任务：%s\n[当前迭代任务描述]: %s",
			historyCtx, iter, task, len(ac.history), task, task,
		)

		// ── Step 3: 执行（交给内层 Loop） ──
		startTime := time.Now()
		finalOutput, execErr := runner(ctx, contextTask)
		duration := time.Since(startTime)

		// ── Step 4: 收口结果 ──
		resultSummary := ac.summarizeResult(task, finalOutput, execErr)

		record := IterationRecord{
			Index:    iter,
			Task:     task,
			Output:   finalOutput,
			Duration: duration,
		}
		allRecords = append(allRecords, record)

		// 存入历史（含关键摘要）
		ac.mu.Lock()
		ac.history = append(ac.history, resultSummary)
		ac.mu.Unlock()

		if execErr != nil {
			onEvent(Event{Type: EventError, Content: fmt.Sprintf("迭代 %d 执行错误: %v", iter, execErr)})
			if iter >= 3 && ac.detectStuck() {
				onEvent(Event{Type: EventNotice, Content: "连续多次失败，自主模式终止。"})
				break
			}
		} else {
			onEvent(Event{Type: EventNotice, Content: fmt.Sprintf("✅ 迭代 %d 完成 (%v)", iter, duration)})
		}

		// 检测重复任务
		if ac.detectCycling() {
			onEvent(Event{Type: EventNotice, Content: "检测到任务重复循环，自动终止。"})
			break
		}
	}

	// ── 生成最终总结报告 ──
	report := ac.buildFinalReport(goal, allRecords)
	return report, nil
}

// ── 内部方法 ──

// generateNextTask 用规划 LLM 生成下一个任务（NEXT/DONE 模式）。
func (ac *AutonomousController) generateNextTask(ctx context.Context, goal string) (struct {
	task string
	done string
}, error) {
	ac.mu.Lock()
	history := make([]string, len(ac.history))
	copy(history, ac.history)
	ac.mu.Unlock()

	// 重复检测：最近 3 轮两两相似度都 > 0.7 → 高度重复，自动终止
	if len(history) >= 3 {
		recent := history[len(history)-3:]
		pairs := [][2]string{
			{recent[0], recent[1]},
			{recent[1], recent[2]},
			{recent[0], recent[2]},
		}
		allSimilar := true
		for _, p := range pairs {
			if taskSimilarity(p[0], p[1]) <= 0.7 {
				allSimilar = false
				break
			}
		}
		if allSimilar {
			return struct{ task string; done string }{
				done: "最近 3 轮任务高度重复（编辑距离检测），自动终止",
			}, nil
		}
	}

	historyText := "尚无已完成任务，请生成第一个具体任务。"
	if len(history) > 0 {
		historyText = fmt.Sprintf(
			"已完成的任务（共 %d 轮）：\n%s\n\n⚠️ 以上任务全部已经完成。请生成与它们明显不同的下一步任务。如果用户目标的核心已覆盖，回复 DONE。",
			len(history),
			strings.Join(history, "\n"),
		)
	}

	userContent := fmt.Sprintf(
		"[用户目标]\n%s\n\n%s\n\n请生成下一个具体任务或回复 DONE。",
		goal, historyText,
	)

	msgs := []Message{
		{Role: RoleSystem, Content: autoTaskGenPrompt},
		{Role: RoleUser, Content: userContent},
	}

	// 用规划 LLM 生成（注册表里只有 plan 工具可用）
	tools := ac.Registry.Definitions()
	var responseContent string

	resp, err := ac.Provider.Chat(ctx, msgs, tools, nil)
	if err != nil {
		return struct{ task string; done string }{}, fmt.Errorf("规划 LLM 调用失败: %w", err)
	}
	responseContent = resp.Content

	// 解析 NEXT/DONE
	text := strings.TrimSpace(responseContent)

	// 尝试匹配 NEXT: ...
	if idx := strings.Index(text, "NEXT:"); idx >= 0 {
		after := text[idx+5:]
		if colon := strings.Index(after, ":"); colon >= 0 {
			after = after[colon+1:]
		}
		taskStr := strings.TrimSpace(after)
		if taskStr != "" {
			// 重复检测：与历史任务比较
			for _, h := range history {
				// 从历史记录提取任务描述
				hTask := h
				if parts := strings.SplitN(h, "→", 2); len(parts) > 0 {
					hTask = strings.TrimSpace(parts[0])
				}
				if taskSimilarity(taskStr, hTask) > 0.5 {
					return struct{ task string; done string }{
						done: "检测到与历史任务重复，自动终止",
					}, nil
				}
			}
			return struct{ task string; done string }{task: taskStr}, nil
		}
	}

	// 尝试匹配 DONE: ...
	if idx := strings.Index(text, "DONE:"); idx >= 0 {
		after := strings.TrimSpace(text[idx+5:])
		if after == "" {
			after = "所有目标已完成"
		}
		return struct{ task string; done string }{done: after}, nil
	}

	// 包含 DONE 关键字
	if strings.Contains(strings.ToUpper(text), "DONE") {
		return struct{ task string; done string }{done: text}, nil
	}

	// 兜底：首行作为任务
	firstLine := text
	if lines := strings.SplitN(text, "\n", 2); len(lines) > 0 {
		firstLine = strings.TrimSpace(lines[0])
	}
	if len(firstLine) > 5 {
		return struct{ task string; done string }{task: firstLine}, nil
	}

	return struct{ task string; done string }{done: "无法生成更多任务"}, nil
}

// buildHistoryContext 构建历史迭代上下文（注入到下一轮任务的提示中）。
func (ac *AutonomousController) buildHistoryContext() string {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	if len(ac.history) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[自主模式迭代历史 — 以下全部已完成，不要重复]\n")
	for i, h := range ac.history {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, h))
	}
	b.WriteString("\n")
	return b.String()
}

// summarizeResult 收口执行结果，生成一行摘要。
func (ac *AutonomousController) summarizeResult(task, output string, execErr error) string {
	if execErr != nil {
		return fmt.Sprintf("[第%d轮] %s → 失败: %s",
			len(ac.history)+1, truncStr(task, 80), truncStr(execErr.Error(), 100))
	}
	if len(output) > 10 {
		return fmt.Sprintf("[第%d轮] %s → 完成",
			len(ac.history)+1, truncStr(task, 80))
	}
	return fmt.Sprintf("[第%d轮] %s → 完成", len(ac.history)+1, truncStr(task, 80))
}

// detectStuck 检测是否卡住（连续多轮失败）。
func (ac *AutonomousController) detectStuck() bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	failCount := 0
	for i := len(ac.history) - 1; i >= 0 && i >= len(ac.history)-3; i-- {
		if strings.Contains(ac.history[i], "失败") {
			failCount++
		}
	}
	return failCount >= 3
}

// detectCycling 检测任务循环。
func (ac *AutonomousController) detectCycling() bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	if len(ac.history) < 4 {
		return false
	}
	recent := ac.history[len(ac.history)-4:]
	pairs := [][2]string{
		{recent[0], recent[2]},
		{recent[1], recent[3]},
	}
	for _, p := range pairs {
		if taskSimilarity(p[0], p[1]) > 0.65 {
			return true
		}
	}
	return false
}

// buildFinalReport 生成最终总结报告。
func (ac *AutonomousController) buildFinalReport(goal string, records []IterationRecord) string {
	var b strings.Builder
	b.WriteString("## 自主模式执行报告\n\n")
	b.WriteString(fmt.Sprintf("**目标**: %s\n\n", goal))
	b.WriteString(fmt.Sprintf("**总迭代数**: %d\n\n", len(records)))

	if len(records) > 0 {
		b.WriteString("### 执行记录\n\n")
		b.WriteString("| 轮次 | 任务 | 耗时 |\n")
		b.WriteString("|------|------|------|\n")
		for _, r := range records {
			b.WriteString(fmt.Sprintf("| %d | %s | %v |\n", r.Index, truncStr(r.Task, 60), r.Duration))
		}
		b.WriteString("\n")
	}

	// 提取涉及的文件
	fileSet := map[string]bool{}
	for _, r := range records {
		for _, word := range strings.Fields(r.Output) {
			if strings.Contains(word, ".") && !strings.Contains(word, " ") {
				clean := strings.Trim(word, "\"'`.,;:()[]{}")
				if len(clean) > 3 && !fileSet[clean] {
					fileSet[clean] = true
				}
			}
		}
	}
	if len(fileSet) > 0 {
		b.WriteString("### 涉及文件\n\n")
		for f := range fileSet {
			b.WriteString(fmt.Sprintf("- %s\n", f))
		}
		b.WriteString("\n")
	}

	b.WriteString("---\n")
	b.WriteString("*自主模式已完成全部迭代*")
	return b.String()
}

// ── 辅助 ──

func truncStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// RegisterPlanOnlyTools 注册只含 plan 工具的注册表（供外层 AutonomousController 使用）。
// 目前只有 update_plan（pan 工具）。
func RegisterPlanOnlyTools(r *Registry) {
	// 只注册 update_plan（规划工具），不含任何 task/执行工具
	registerPlanTool(r) // 从 plan.go 引入的 update_plan
}
