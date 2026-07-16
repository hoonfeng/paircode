// subagent_parallel_test.go 并行子 Agent 执行测试：验证多个子 Agent 并行执行时事件流正确性。
//
// 测试覆盖：
//   1. 两个子 Agent 并行执行，每个事件携带正确 AgentName
//   2. 并行子 Agent 的 EventDone/EventFinal 被过滤（不泄漏到父事件流）
//   3. 共享上下文在多个并行子 Agent 间正确传递
//   4. 冲突检测在并行场景下正常工作

package agent

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// TestParallelSubAgent_EventAgentName 并行子 Agent 事件标记测试：
// 两个子 Agent 并行执行，验证每个事件都携带对应 AgentName。
func TestParallelSubAgent_EventAgentName(t *testing.T) {
	var mu sync.Mutex
	var receivedEvents []Event

	// 直接测试 SubAgentSink：模拟两个子 Agent 的 OnEvent
	onEventA := SubAgentSink(func(e Event) {
		mu.Lock()
		receivedEvents = append(receivedEvents, e)
		mu.Unlock()
	}, "agentA")

	onEventB := SubAgentSink(func(e Event) {
		mu.Lock()
		receivedEvents = append(receivedEvents, e)
		mu.Unlock()
	}, "agentB")

	// 模拟两个子 Agent 并行发出事件
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		onEventA(Event{Type: EventThinking, Content: "A 在思考"})
		onEventA(Event{Type: EventContent, Content: "A 的内容"})
		onEventA(Event{Type: EventToolCall, Tool: "read_file", Args: `{"path":"a.go"}`})
		onEventA(Event{Type: EventToolResult, Tool: "read_file", Content: "A 的结果"})
		onEventA(Event{Type: EventFinal, Content: "A 完成"})   // 应被过滤
		onEventA(Event{Type: EventDone, Content: "A 完成"})     // 应被过滤
	}()

	go func() {
		defer wg.Done()
		onEventB(Event{Type: EventThinking, Content: "B 在思考"})
		onEventB(Event{Type: EventContent, Content: "B 的内容"})
		onEventB(Event{Type: EventToolCall, Tool: "read_file", Args: `{"path":"b.go"}`})
		onEventB(Event{Type: EventToolResult, Tool: "read_file", Content: "B 的结果"})
		onEventB(Event{Type: EventFinal, Content: "B 完成"})   // 应被过滤
		onEventB(Event{Type: EventDone, Content: "B 完成"})     // 应被过滤
	}()

	wg.Wait()

	// 验证：所有 AgentName 正确
	for _, e := range receivedEvents {
		if e.AgentName != "agentA" && e.AgentName != "agentB" {
			t.Errorf("事件应携带 AgentName（agentA 或 agentB），得 %q：%+v", e.AgentName, e)
		}
	}

	// 验证：没有 EventDone 泄漏
	for _, e := range receivedEvents {
		if e.Type == EventDone {
			t.Errorf("并行子 Agent 的 EventDone 不应泄漏：%+v", e)
		}
	}

	// 验证：没有 EventFinal 泄漏（从子 Agent）
	for _, e := range receivedEvents {
		if e.Type == EventFinal && e.AgentName != "" {
			t.Errorf("并行子 Agent 的 EventFinal 不应泄漏：%+v", e)
		}
	}

	// 验证：AgentA 和 AgentB 的事件都可见
	var agentAEvents, agentBEvents int
	for _, e := range receivedEvents {
		switch e.AgentName {
		case "agentA":
			agentAEvents++
		case "agentB":
			agentBEvents++
		}
	}
	if agentAEvents == 0 {
		t.Error("AgentA 的事件应可见")
	}
	if agentBEvents == 0 {
		t.Error("AgentB 的事件应可见")
	}
}

// TestParallelSubAgent_SharedContext 并行子 Agent 共享上下文测试：
// 两个子 Agent 通过共享上下文池交换信息，验证无冲突。
func TestParallelSubAgent_SharedContext(t *testing.T) {
	pool := NewSharedContext()

	// AgentA 写入知识
	pool.AddKnowledge("agentA", "arch", "MVC", "项目使用 MVC 架构")
	pool.StoreFile("agentA", "main.go", "package main")

	// AgentB 读取 AgentA 写入的知识
	entry, ok := pool.GetKnowledge("arch")
	if !ok {
		t.Fatal("AgentB 应能读到 AgentA 写入的知识")
	}
	if entry.Value != "MVC" {
		t.Errorf("知识值应为 MVC，得 %q", entry.Value)
	}

	// AgentB 读取 AgentA 缓存的文件
	content, ok := pool.LoadFile("main.go")
	if !ok || content != "package main" {
		t.Errorf("AgentB 应能读到 AgentA 缓存的文件，得 %q, %v", content, ok)
	}

	// AgentB 设置共享变量
	pool.SetVar("agentB", "port", "8080")

	// AgentA 读取共享变量
	val, ok := pool.GetVar("port")
	if !ok || val != "8080" {
		t.Errorf("AgentA 应能读到 AgentB 设置的变量，得 %q, %v", val, ok)
	}

	// 验证知识摘要
	summary := pool.KnowledgeSummary()
	if !strings.Contains(summary, "MVC") || !strings.Contains(summary, "main.go") {
		t.Errorf("知识摘要应含知识和文件，得 %q", summary)
	}

	// 验证无冲突
	conflicts := pool.DetectConflicts()
	if len(conflicts) > 0 {
		t.Errorf("不应有冲突，得 %+v", conflicts)
	}
}

// TestParallelSubAgent_ConflictDetection 并行子 Agent 冲突检测测试：
// 两个子 Agent 设置同一变量的不同值，检测到冲突。
func TestParallelSubAgent_ConflictDetection(t *testing.T) {
	pool := NewSharedContext()

	// AgentA 和 AgentB 设置同一变量的不同值
	pool.SetVar("agentA", "db_host", "localhost")
	pool.SetVar("agentB", "db_host", "192.168.1.1")

	conflicts := pool.DetectConflicts()
	if len(conflicts) != 1 {
		t.Fatalf("应检测到 1 个冲突，得 %d", len(conflicts))
	}
	if conflicts[0].Name != "db_host" {
		t.Errorf("冲突变量名应为 db_host，得 %q", conflicts[0].Name)
	}

	// 验证版本历史
	history := pool.GetVarHistory("db_host")
	if len(history) != 2 {
		t.Fatalf("应有 2 个版本，得 %d", len(history))
	}
	if history[0].AgentName != "agentA" || history[1].AgentName != "agentB" {
		t.Errorf("版本顺序应按写入时间，得 %+v", history)
	}
}

// TestParallelSubAgent_ContextInject 并行子 Agent 上下文注入测试：
// 验证 BuildContextInject 包含已有知识、文件缓存和工具提示。
func TestParallelSubAgent_ContextInject(t *testing.T) {
	pool := NewSharedContext()
	pool.AddKnowledge("planner", "arch", "MVC", "MVC架构")
	pool.StoreFile("coder", "main.go", "package main")
	pool.SetVar("analyzer", "port", "8080")

	inject := pool.BuildContextInject("executor")

	if !strings.Contains(inject, "MVC架构") {
		t.Error("注入应含知识摘要")
	}
	if !strings.Contains(inject, "main.go") {
		t.Error("注入应含文件列表")
	}
	if !strings.Contains(inject, "ctx_read_file") {
		t.Error("注入应含工具提示")
	}
	if !strings.Contains(inject, "port") {
		t.Error("注入应含共享变量")
	}

	// 空池返回工具提示（始终有工具说明）
	emptyPool := NewSharedContext()
	emptyInject := emptyPool.BuildContextInject("executor")
	if emptyInject == "" {
		t.Error("空池也应返回工具提示（始终有工具说明）")
	}
	if !strings.Contains(emptyInject, "ctx_read_file") {
		t.Errorf("空池注入应含工具提示，得 %q", emptyInject)
	}
}

// TestParallelOrchestrator_ExecuteBatch 测试 executeBatch 的并发执行：
// 验证多个子 Agent 同时运行时事件流正确。
func TestParallelOrchestrator_ExecuteBatch(t *testing.T) {
	root := &SubAgent{Name: "coordinator"}
	agentA := &SubAgent{Name: "agentA", System: "你是 Agent A"}
	agentB := &SubAgent{Name: "agentB", System: "你是 Agent B"}
	tree := NewAgentTree(root, agentA, agentB)

	reg := NewRegistry()
	noop := func(_ context.Context, _ map[string]any) (string, error) { return "done", nil }
	reg.Register(&Tool{Name: "read_file", Handler: noop, ReadOnly: true})

	// 为两个子 Agent 准备足够的响应
	// agentA: 读文件 → finish_task
	// agentB: 读文件 → finish_task
	parent := &Loop{
		Provider: &MockProvider{Responses: []Message{
			{ToolCalls: []ToolCall{{ID: "a1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}}}},
			finishTaskMsg("f1", "A 完成"),
			{ToolCalls: []ToolCall{{ID: "b1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"b.go"}`}}}},
			finishTaskMsg("f2", "B 完成"),
		}},
		Registry:      reg,
		System:        "你是协调器",
		MaxIterations: 10,
		currentMsgs:   nil,
		AgentTree:     tree,
		State:         map[string]any{},
	}
	_ = tree // tree is used implicitly via parent.AgentTree

	pool := NewSharedContext()
	orch := NewParallelOrchestrator(parent, tree, pool)
	_ = orch

	// 构造两个并行子任务
	subTasks := []SubTask{
		{ID: "task-A", AgentName: "agentA", Input: "分析文件 A", Dependencies: []string{}},
		{ID: "task-B", AgentName: "agentB", Input: "分析文件 B", Dependencies: []string{}},
	}

	// 构建并行组（都在组0）
	groups := orch.buildParallelGroups(subTasks)
	if len(groups) != 1 || len(groups[0]) != 2 {
		t.Fatalf("两个无依赖任务应在同一组，得 %+v", groups)
	}

	// 执行第一批
	batch := taskGroup{indices: []int{0, 1}}
	results := orch.executeBatch(context.Background(), subTasks, batch)

	// 验证结果
	if len(results) != 2 {
		t.Fatalf("应有 2 个结果，得 %d", len(results))
	}

	// 验证每个结果都有正确的 AgentName
	resultsByName := map[string]SubTaskResult{}
	for _, r := range results {
		resultsByName[r.AgentName] = r
	}

	for _, name := range []string{"agentA", "agentB"} {
		r, ok := resultsByName[name]
		if !ok {
			t.Errorf("缺少 agent %q 的结果", name)
			continue
		}
		if r.Error != "" {
			t.Errorf("agent %q 执行出错：%s", name, r.Error)
		}
		if !strings.Contains(r.Output, "完成") {
			t.Errorf("agent %q 结果应含完成字样，得 %q", name, r.Output)
		}
	}

	// 验证共享上下文中有两个知识条目
	allKnowledge := pool.GetAllKnowledge()
	if len(allKnowledge) != 2 {
		t.Errorf("共享上下文应有 2 条知识，得 %d", len(allKnowledge))
	}
}

// TestParallelSubAgent_EventDoneFiltered 验证并行子 Agent 的 EventDone 不泄漏。
func TestParallelSubAgent_EventDoneFiltered(t *testing.T) {
	var receivedEvents []Event
	parentOnEvent := func(e Event) {
		receivedEvents = append(receivedEvents, e)
	}

	sink := SubAgentSink(parentOnEvent, "agentA")

	// 发送 EventDone——应被过滤
	sink(Event{Type: EventDone, Content: "完成", DoneReason: "task_complete"})

	if len(receivedEvents) != 0 {
		t.Errorf("EventDone 应被 SubAgentSink 过滤，但有 %d 个事件泄漏", len(receivedEvents))
	}

	// 发送 EventFinal——应被过滤
	sink(Event{Type: EventFinal, Content: "最终回复"})

	if len(receivedEvents) != 0 {
		t.Errorf("EventFinal 应被 SubAgentSink 过滤，但有 %d 个事件泄漏", len(receivedEvents))
	}

	// 发送 EventError——应被过滤
	sink(Event{Type: EventError, Content: "错误"})

	if len(receivedEvents) != 0 {
		t.Errorf("EventError 应被 SubAgentSink 过滤，但有 %d 个事件泄漏", len(receivedEvents))
	}

	// 发送 EventContent——应通过（带 AgentName）
	sink(Event{Type: EventContent, Content: "正常内容"})

	if len(receivedEvents) != 1 {
		t.Fatalf("EventContent 应通过 SubAgentSink，但有 %d 个事件", len(receivedEvents))
	}
	if receivedEvents[0].AgentName != "agentA" {
		t.Errorf("通过的事件应标记 AgentName=agentA，得 %q", receivedEvents[0].AgentName)
	}
	if receivedEvents[0].Content != "正常内容" {
		t.Errorf("事件内容应保持不变，得 %q", receivedEvents[0].Content)
	}
}
