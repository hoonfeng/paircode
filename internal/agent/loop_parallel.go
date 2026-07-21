package agent

import (
	"context"
	"strings"
	"sync"
)

// canParallelize 判断工具调用列表是否可以并行执行：2+ 个且全都是 ReadOnly。
func canParallelize(calls []ToolCall, reg *Registry) bool {
	if len(calls) < 2 {
		return false
	}
	for _, tc := range calls {
		t, ok := reg.Get(tc.Function.Name)
		if !ok || !t.ReadOnly || t.RequiresApproval {
			return false
		}
	}
	return true
}

// executeToolsParallel 并行执行只读工具调用，结果按原始顺序收集。
func (l *Loop) executeToolsParallel(ctx context.Context, calls []ToolCall, msgs []Message) []Message {
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

		if r.tc.Function.Name == "finish_task" {
			l.finishResult = &output
			l.emit(Event{Type: EventDone, Content: output, DoneReason: "finish_task"})
		}
		if r.tc.Function.Name == "generate_commit_message" {
			l.commitMessage = output
		}
	}
	return msgs
}
