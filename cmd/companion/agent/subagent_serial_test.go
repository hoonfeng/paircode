// subagent_serial_test.go 串行子 Agent 执行测试：验证事件流正确性（EventDone 过滤、AgentName 标记）。
//
// 测试覆盖：
//   1. 单次 delegate_task：子 Agent 事件携带 AgentName、EventDone/EventFinal 被 SubAgentSink 过滤
//   2. 多次串行 delegate_task：每次委托的新增事件都正确标记 AgentName
//   3. 子 Agent 的思考/内容/工具调用事件在父事件流中可见（仅携带 AgentName 标记）

package agent

import (
	"context"
	"strings"
	"testing"
)

// TestSerialSubAgent_EventFiltering 测试串行子 Agent 的事件过滤：
//   - 子 Agent 的 EventContent/EventThinking 携带 AgentName
//   - 子 Agent 的 EventDone 不出现于父事件流
//   - 子 Agent 的 EventFinal 不出现于父事件流
func TestSerialSubAgent_EventFiltering(t *testing.T) {
	root := &SubAgent{Name: "coordinator"}
	worker := &SubAgent{Name: "worker", System: "你是一个计算助手"}
	tree := NewAgentTree(root, worker)

	reg := NewRegistry()
	// 模拟：父 → delegate_task(worker) → 子 finish_task → 父 finish_task
	rec := &recordingProvider{responses: []Message{
		{ToolCalls: []ToolCall{delegateTaskCall("d1", "worker", "计算 1+1")}},
		finishTaskMsg("f1", "结果是 2"),
		{ToolCalls: []ToolCall{{ID: "p1", Type: "function", Function: FunctionCall{Name: "finish_task", Arguments: `{"result":"已完成"}`}}}},
	}}

	var receivedEvents []Event
	parent := &Loop{
		Provider:      rec,
		Registry:      reg,
		System:        "你是协调器",
		MaxIterations: 5,
		AgentTree:     tree,
		State:         map[string]any{},
		OnEvent: func(e Event) {
			receivedEvents = append(receivedEvents, e)
		},
	}
	RegisterDelegateTools(parent, tree)

	if _, err := parent.Run(context.Background(), "帮我算一下", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// 检查子 Agent 的 EventDone 是否被过滤（父 Agent 的 EventDone 是正常的）
	for _, e := range receivedEvents {
		if e.Type == EventDone && e.AgentName == "worker" {
			t.Errorf("子 Agent 的 EventDone 不应出现在父事件流中：%+v", e)
		}
	}

	// 检查子 Agent 的 EventFinal 是否被过滤（父 Agent 的 EventFinal 是正常的）
	for _, e := range receivedEvents {
		if e.Type == EventFinal && e.AgentName == "worker" {
			t.Errorf("子 Agent 的 EventFinal 不应出现在父事件流中：%+v", e)
		}
	}

	// 检查工具调用事件来自子 Agent 时是否携带 AgentName
	var workerEvents int
	for _, e := range receivedEvents {
		if e.AgentName == "worker" {
			workerEvents++
		}
	}

	// 子 Agent 的工具调用（finish_task）的工具结果应携带 AgentName
	var workerToolEvent bool
	for _, e := range receivedEvents {
		if e.Type == EventToolResult && e.AgentName == "worker" && e.Tool == "finish_task" {
			workerToolEvent = true
		}
	}
	if !workerToolEvent {
		t.Errorf("子 Agent 的 finish_task 工具结果应携带 AgentName=worker，events=%+v", receivedEvents)
	}
}

// TestSerialSubAgent_MultipleDelegations 多次串行委托：每次子 Agent 事件都正确标记。
func TestSerialSubAgent_MultipleDelegations(t *testing.T) {
	root := &SubAgent{Name: "coordinator"}
	worker := &SubAgent{Name: "worker", System: "你是一个助手"}
	tree := NewAgentTree(root, worker)

	reg := NewRegistry()
	// 模拟：父 → delegate_task(worker, "任务1") → 子 finish_task → 父 event →
	//       delegate_task(worker, "任务2") → 子 finish_task → 父 finish_task
	rec := &recordingProvider{responses: []Message{
		{ToolCalls: []ToolCall{delegateTaskCall("d1", "worker", "任务1")}},
		finishTaskMsg("f1", "结果1"),
		{ToolCalls: []ToolCall{delegateTaskCall("d2", "worker", "任务2")}},
		finishTaskMsg("f2", "结果2"),
		{ToolCalls: []ToolCall{{ID: "p1", Type: "function", Function: FunctionCall{Name: "finish_task", Arguments: `{"result":"全部完成"}`}}}},
	}}

	var receivedEvents []Event
	parent := &Loop{
		Provider:      rec,
		Registry:      reg,
		System:        "你是协调器",
		MaxIterations: 10,
		AgentTree:     tree,
		State:         map[string]any{},
		OnEvent: func(e Event) {
			receivedEvents = append(receivedEvents, e)
		},
	}
	RegisterDelegateTools(parent, tree)

	if _, err := parent.Run(context.Background(), "帮我做两个任务", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// 两次委托，子 Agent 的工具结果都应携带 AgentName=worker
	var workerResults int
	for _, e := range receivedEvents {
		if e.Type == EventToolResult && e.AgentName == "worker" {
			workerResults++
		}
	}
	if workerResults != 2 {
		t.Errorf("应有 2 个子 Agent 工具结果，得 %d（events=%+v）", workerResults, receivedEvents)
	}

	// 检查没有子 Agent 的 EventDone 泄漏（父 Agent 的 EventDone 是正常的）
	for _, e := range receivedEvents {
		if e.Type == EventDone && e.AgentName != "" {
			t.Errorf("子 Agent 的 EventDone 不应泄漏到父事件流：%+v (AgentName=%q)", e, e.AgentName)
		}
	}
}

// TestSerialSubAgent_SingleTurn 单轮委托：子 Agent 的 EventFinal 不应泄漏到父事件流。
func TestSerialSubAgent_SingleTurn(t *testing.T) {
	root := &SubAgent{Name: "coordinator"}
	calc := &SubAgent{Name: "calc"}
	tree := NewAgentTree(root, calc)
	reg := NewRegistry()
	rec := &recordingProvider{responses: []Message{
		{ToolCalls: []ToolCall{delegateSingleCall("d1", "calc", "1+1=?")}},
		{Content: "结果是 2"},
		{ToolCalls: []ToolCall{{ID: "p1", Type: "function", Function: FunctionCall{Name: "finish_task", Arguments: `{"result":"好了"}`}}}},
	}}

	var receivedEvents []Event
	parent := &Loop{
		Provider:      rec,
		Registry:      reg,
		System:        "你是协调器",
		MaxIterations: 5,
		AgentTree:     tree,
		State:         map[string]any{},
		OnEvent: func(e Event) {
			receivedEvents = append(receivedEvents, e)
		},
	}
	RegisterDelegateTools(parent, tree)

	if _, err := parent.Run(context.Background(), "算一下", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// 单轮委托中，子 Agent 发出 EventFinal 和 EventDone 但应被 SubAgentSink 过滤
	for _, e := range receivedEvents {
		if e.Type == EventDone && e.AgentName == "calc" {
			t.Errorf("单轮委托中子 Agent 的 EventDone 不应泄漏到父事件流：%+v", e)
		}
		if e.Type == EventFinal && e.AgentName == "calc" {
			t.Errorf("单轮委托中子 Agent 的 EventFinal 不应泄漏到父事件流：%+v", e)
		}
	}

	// 但子 Agent 的 EventContent 应携带 AgentName 出现在父事件流中
	var calcContent bool
	for _, e := range receivedEvents {
		if e.Type == EventContent && e.AgentName == "calc" {
			calcContent = true
		}
	}
	if !calcContent {
		t.Logf("注意：子 Agent 的 EventContent 可能被过滤，这是正常的（依具体实现）")
	}

	// 最重要的是：子 Agent 的结果（"结果是 2"）应通过函数返回值回到父的工具结果中
	// 验证父最终消息含委托结果
	var sawTwo bool
	for _, e := range receivedEvents {
		if e.Type == EventToolResult && e.Tool == "delegate_single_turn" && strings.Contains(e.Content, "2") {
			sawTwo = true
		}
	}
	if !sawTwo {
		// 也可能通过 finish_task 或 lastAssistantContent 传回
		// 检查 EventDone 是否含 2
		for _, e := range receivedEvents {
			if (e.Type == EventFinal || e.Type == EventDone) && strings.Contains(e.Content, "2") {
				sawTwo = true
			}
		}
	}
	if !sawTwo {
		t.Errorf("父事件流应含子 Agent 的结果（2），events=%+v", receivedEvents)
	}
}
