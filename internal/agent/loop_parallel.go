package agent

import (
	"context"
	"strings"
	"sync"
)

// canParallelize 判断工具调用列表是否可以并行执行：
// - 2+ 个且全都是 ReadOnly → 直接并行（最快路径，无需预检）
// - 含任何非只读/需审批工具 → 保守退回串行（写操作涉及文件冲突与审批顺序，并行有风险）
func canParallelize(calls []ToolCall, reg *Registry) bool {
	if len(calls) < 2 {
		return false
	}
	for _, tc := range calls {
		t, ok := reg.Get(tc.Function.Name)
		if !ok {
			return false
		}
		if !t.ReadOnly || t.RequiresApproval {
			return false // 含写/需审批工具 → 串行（保守安全）
		}
	}
	return true
}

// tryParallelExecute 尝试并行执行工具调用。
// 纯只读工具 → 直接并行（无需预检）
// 混合工具 → 预检（串行审批）→ 并行执行（结果按原始顺序收集）
// 返回 (msgs, wasParallel) — wasParallel 为 false 表示条件不满足退回串行。
func (l *Loop) tryParallelExecute(ctx context.Context, calls []ToolCall, msgs []Message) ([]Message, bool) {
	if len(calls) < 2 {
		return msgs, false
	}

	// ── 情况 1：全部 ReadOnly → 直接并行（快路径）──
	allReadOnly := true
	for _, tc := range calls {
		t, ok := l.Registry.Get(tc.Function.Name)
		if !ok || !t.ReadOnly || t.RequiresApproval {
			allReadOnly = false
			break
		}
	}
	if allReadOnly {
		return l.executeReadOnlyParallel(ctx, calls, msgs), true
	}

	// ── 情况 2：混合工具 → 预检（串行审批）→ 并行执行 ──
	// 先做审批预检
	type approvedCall struct {
		tc       ToolCall
		approved bool
		feedback string
	}
	preflight := make([]approvedCall, len(calls))
	for i, tc := range calls {
		preflight[i].tc = tc
		preflight[i].approved = true
		l.emit(Event{Type: EventToolCall, Tool: tc.Function.Name, Args: tc.Function.Arguments, CallID: tc.ID})

		// 审批检查
		approved, feedback := l.resolveApproval(ctx, tc)
		if !approved {
			preflight[i].approved = false
			preflight[i].feedback = feedback
		}
	}

	// 收集需要并行执行的任务
	var toExec []int   // 需要执行的下标
	var wg sync.WaitGroup
	type execResult struct {
		idx    int
		output string
		err    error
	}
	results := make([]execResult, 0)

	for i, p := range preflight {
		if !p.approved {
			// 未通过审批的立即返回拒绝结果
			rej := strings.TrimSpace(p.feedback)
			if rej == "" {
				rej = "用户拒绝了此操作。请勿重试该操作；改用其他方式达成目标，或先向用户说明你为何需要它。"
			}
			l.emit(Event{Type: EventToolResult, Tool: p.tc.Function.Name, Content: rej, CallID: p.tc.ID})
			msgs = append(msgs, Message{Role: RoleTool, ToolCallID: p.tc.ID, Name: p.tc.Function.Name, Content: rej})
			l.trackCall(p.tc.Function.Name, p.tc.Function.Arguments, true)
			continue
		}
		// 审批通过 → 加入并行执行列表
		toExec = append(toExec, i)
		results = append(results, execResult{idx: i})
	}

	if len(toExec) == 0 {
		return msgs, true // 全部被拒，但也算并行处理过
	}

	// 解析参数（JSON）准备并行执行
	wg.Add(len(toExec))
	for k, idx := range toExec {
		go func(k, idx int) {
			defer wg.Done()
			out, err := l.Registry.Execute(ctx, preflight[idx].tc.Function.Name, preflight[idx].tc.Function.Arguments)
			results[k] = execResult{idx: idx, output: out, err: err}
		}(k, idx)
	}
	wg.Wait()

	// 按原始顺序收集结果
	for _, r := range results {
		output := r.output
		if r.err != nil {
			output = "Error: " + r.err.Error()
		}
		tc := preflight[r.idx].tc
		l.emit(Event{Type: EventToolResult, Tool: tc.Function.Name, Content: output, CallID: tc.ID})
		msgs = append(msgs, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: output})
		l.trackCall(tc.Function.Name, tc.Function.Arguments, r.err != nil || strings.HasPrefix(strings.TrimSpace(output), "Error:"))
		if tc.Function.Name == "generate_commit_message" {
			l.commitMessage = output
		}
	}

	return msgs, true
}

// executeReadOnlyParallel 并行执行只读工具调用，结果按原始顺序收集。
// 这是最快路径：所有工具已知 ReadOnly，无需审批预检。
func (l *Loop) executeReadOnlyParallel(ctx context.Context, calls []ToolCall, msgs []Message) []Message {
	type result struct {
		tc     ToolCall
		output string
		err    error
	}
	results := make([]result, len(calls))
	var wg sync.WaitGroup

	for i, tc := range calls {
		l.emit(Event{Type: EventToolCall, Tool: tc.Function.Name, Args: tc.Function.Arguments, CallID: tc.ID})
		wg.Add(1)
		go func(idx int, tc ToolCall) {
			defer wg.Done()
			out, err := l.Registry.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
			results[idx] = result{tc: tc, output: out, err: err}
		}(i, tc)
	}
	wg.Wait()

	// 按原始顺序收集结果
	for _, r := range results {
		output := r.output
		if r.err != nil {
			output = "Error: " + r.err.Error()
		}
		l.emit(Event{Type: EventToolResult, Tool: r.tc.Function.Name, Content: output, CallID: r.tc.ID})
		msgs = append(msgs, Message{Role: RoleTool, ToolCallID: r.tc.ID, Name: r.tc.Function.Name, Content: output})
		l.trackCall(r.tc.Function.Name, r.tc.Function.Arguments, r.err != nil || strings.HasPrefix(strings.TrimSpace(output), "Error:"))

		if r.tc.Function.Name == "generate_commit_message" {
			l.commitMessage = output
		}
	}
	return msgs
}

// resolveApproval 检查单个工具调用是否需要审批并执行审批流程。
// 返回 (approved, feedback)。
func (l *Loop) resolveApproval(ctx context.Context, tc ToolCall) (bool, string) {
	approveFn := l.Approve
	toolName := tc.Function.Name

	// 检查黑白名单
	inBlacklist := false
	for _, name := range l.ReviewBlacklist {
		if strings.Contains(toolName, name) {
			inBlacklist = true
			break
		}
	}
	inWhitelist := false
	if !inBlacklist {
		for _, name := range l.ReviewWhitelist {
			if strings.Contains(toolName, name) {
				inWhitelist = true
				break
			}
		}
	}
	if inBlacklist {
		switch l.getReviewMode() {
		case "auto":
			approveFn = l.aiReviewApprove
		default:
			approveFn = l.Approve
		}
	} else if inWhitelist {
		approveFn = nil
	} else {
		switch l.getReviewMode() {
		case "auto":
			approveFn = l.aiReviewApprove
		case "off":
			approveFn = nil
		}
	}

	if approveFn == nil {
		return true, ""
	}

	tool, ok := l.Registry.Get(toolName)
	if !ok || !tool.RequiresApproval {
		return true, ""
	}

	approved, feedback := approveFn(ctx, tc)
	return approved, feedback
}
