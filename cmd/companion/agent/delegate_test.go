// delegate_test.go 测试多 agent 委托工具：缓存前缀命中 / 单轮委托 / 控制权转移 / 共享 State / 工具白名单。
//
// 关键断言（spec 4.6 缓存前缀稳定）：子 Loop 首次 LLM 调用的 messages 前缀
// 与父上一次调用逐字节一致 → prompt cache 命中。用 recordingProvider 记录每次 Chat
// 的 messages+tools，断言 recorded[1].messages[:2] == recorded[0].messages。

package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// recordingProvider 脚本化提供方 + 记录每次 Chat 的 messages/tools（断言缓存前缀用）。
// 与 MockProvider 区别：保留每次调用的完整 messages/tools 副本供测试事后检查。
type recordingProvider struct {
	responses []Message
	calls     int
	recorded  []recCall
}

type recCall struct {
	messages []Message
	tools    []ToolDefinition
}

func (r *recordingProvider) Name() string { return "recording" }
func (r *recordingProvider) Calls() int   { return r.calls }

func (r *recordingProvider) Chat(_ context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	// 记录副本（防 Run 后续修改底层数组影响断言）
	msgCopy := make([]Message, len(messages))
	copy(msgCopy, messages)
	toolsCopy := make([]ToolDefinition, len(tools))
	copy(toolsCopy, tools)
	r.recorded = append(r.recorded, recCall{msgCopy, toolsCopy})

	var msg Message
	if r.calls < len(r.responses) {
		msg = r.responses[r.calls]
	} else {
		msg = Message{Role: RoleAssistant, Content: "[FINAL]"} // 脚本耗尽兜底
	}
	r.calls++
	if msg.Role == "" {
		msg.Role = RoleAssistant
	}
	if onChunk != nil {
		onChunk(Chunk{Content: msg.Content, Reasoning: msg.Reasoning, ToolCalls: msg.ToolCalls, Done: true})
	}
	return msg, nil
}

// msgEqual 浅比较两条消息（Role+Content），用于断言缓存前缀逐条一致。
func msgEqual(a, b Message) bool {
	return a.Role == b.Role && a.Content == b.Content
}

// delegateTaskCall 构造一条 delegate_task 工具调用。
func delegateTaskCall(id, agent, task string) ToolCall {
	return ToolCall{ID: id, Type: "function", Function: FunctionCall{
		Name:      "delegate_task",
		Arguments: fmt.Sprintf(`{"agent_name":%q,"task":%q}`, agent, task),
	}}
}

func delegateSingleCall(id, agent, input string) ToolCall {
	return ToolCall{ID: id, Type: "function", Function: FunctionCall{
		Name:      "delegate_single_turn",
		Arguments: fmt.Sprintf(`{"agent_name":%q,"input":%q}`, agent, input),
	}}
}

// finishTaskMsg 构造一条 finish_task 工具调用的 assistant Message（直接作 responses 元素）。
func finishTaskMsg(id, result string) Message {
	return Message{ToolCalls: []ToolCall{{ID: id, Type: "function", Function: FunctionCall{
		Name:      "finish_task",
		Arguments: fmt.Sprintf(`{"result":%q}`, result),
	}}}}
}

func transferCall(id, agent string) ToolCall {
	return ToolCall{ID: id, Type: "function", Function: FunctionCall{
		Name:      "transfer_to_agent",
		Arguments: fmt.Sprintf(`{"agent_name":%q}`, agent),
	}}
}

// TestDelegateTask_MultiTurn 父调 delegate_task → 子调 finish_task(计划A) → 父 [FINAL]。
// 断言：父最终消息含工具结果"计划A" + 子首次调用前缀与父首次调用一致(缓存命中) + 子 system 作追加 instruction。
func TestDelegateTask_MultiTurn(t *testing.T) {
	root := &SubAgent{Name: "coordinator"}
	planner := &SubAgent{Name: "planner", System: "你是规划专家，输出简洁计划"}
	tree := NewAgentTree(root, planner)

	reg := NewRegistry()
	rec := &recordingProvider{responses: []Message{
		{ToolCalls: []ToolCall{delegateTaskCall("d1", "planner", "给计划")}},
		finishTaskMsg("f1", "计划A"),
		{Content: "完成 [FINAL]"},
	}}
	parent := &Loop{
		Provider:      rec,
		Registry:      reg,
		System:        "你是协调器",
		MaxIterations: 5,
		AgentTree:     tree,
		State:         map[string]any{},
	}
	RegisterDelegateTools(parent, tree)

	msgs, err := parent.Run(context.Background(), "需要规划", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rec.Calls() != 3 {
		t.Fatalf("应 3 次 LLM 调用(父→子→父)，得 %d", rec.Calls())
	}

	// 父最终消息应含 delegate_task 的工具结果"计划A"
	var gotResult bool
	for _, m := range msgs {
		if m.Role == RoleTool && strings.Contains(m.Content, "计划A") {
			gotResult = true
		}
	}
	if !gotResult {
		t.Errorf("父消息应含工具结果\"计划A\"，msgs=%+v", msgs)
	}

	// 缓存前缀：子首次调用(recorded[1])前 2 条应与父首次调用(recorded[0])一致
	if len(rec.recorded[1].messages) < 2 {
		t.Fatalf("子调用消息应至少 2 条，得 %d", len(rec.recorded[1].messages))
	}
	for i := 0; i < 2; i++ {
		if !msgEqual(rec.recorded[0].messages[i], rec.recorded[1].messages[i]) {
			t.Errorf("缓存前缀不一致：父[%d]=%+v 子[%d]=%+v", i, rec.recorded[0].messages[i], i, rec.recorded[1].messages[i])
		}
	}

	// 子 system 作追加 instruction：子调用第 3 条(user childTask)应含"你是规划专家"
	childTaskMsg := rec.recorded[1].messages[2]
	if !strings.Contains(childTaskMsg.Content, "你是规划专家") {
		t.Errorf("子 task 应含子 system 指令，得 %q", childTaskMsg.Content)
	}
	if !strings.Contains(childTaskMsg.Content, "给计划") {
		t.Errorf("子 task 应含原任务描述，得 %q", childTaskMsg.Content)
	}
}

// TestDelegateSingleTurn 单轮委托：子只做 1 次 LLM 调用，结果直接返回。
func TestDelegateSingleTurn(t *testing.T) {
	root := &SubAgent{Name: "coordinator"}
	calc := &SubAgent{Name: "calc"}
	tree := NewAgentTree(root, calc)

	reg := NewRegistry()
	rec := &recordingProvider{responses: []Message{
		{ToolCalls: []ToolCall{delegateSingleCall("d1", "calc", "1+1=?")}},
		{Content: "2"}, // 子单轮：无工具调用 → 视作完成
		{Content: "完成 [FINAL]"},
	}}
	parent := &Loop{
		Provider:      rec,
		Registry:      reg,
		System:        "你是协调器",
		MaxIterations: 5,
		AgentTree:     tree,
		State:         map[string]any{},
	}
	RegisterDelegateTools(parent, tree)

	msgs, err := parent.Run(context.Background(), "算一下", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// 单轮委托结果"2"应作工具结果回到父消息
	var gotTwo bool
	for _, m := range msgs {
		if m.Role == RoleTool && strings.Contains(m.Content, "2") {
			gotTwo = true
		}
	}
	if !gotTwo {
		t.Errorf("父消息应含单轮委托结果\"2\"，msgs=%+v", msgs)
	}
}

// TestTransferToAgent 控制权转移：父调 transfer_to_agent → 设 transferTarget 后 Loop 退出。
func TestTransferToAgent(t *testing.T) {
	root := &SubAgent{Name: "coordinator"}
	other := &SubAgent{Name: "other"}
	tree := NewAgentTree(root, other)

	reg := NewRegistry()
	rec := &recordingProvider{responses: []Message{
		{ToolCalls: []ToolCall{transferCall("t1", "other")}},
	}}
	parent := &Loop{
		Provider:      rec,
		Registry:      reg,
		System:        "你是协调器",
		MaxIterations: 5,
		AgentTree:     tree,
		State:         map[string]any{},
	}
	RegisterDelegateTools(parent, tree)

	if _, err := parent.Run(context.Background(), "转交", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if parent.transferTarget != "other" {
		t.Errorf("transferTarget 应为 \"other\"，得 %q", parent.transferTarget)
	}
}

// TestTransferToAgent_NotFound 目标 agent 不存在 → 工具报错，不转移。
func TestTransferToAgent_NotFound(t *testing.T) {
	root := &SubAgent{Name: "coordinator"}
	tree := NewAgentTree(root)
	reg := NewRegistry()
	rec := &recordingProvider{responses: []Message{
		{ToolCalls: []ToolCall{transferCall("t1", "ghost")}},
		{Content: "好的 [FINAL]"},
	}}
	parent := &Loop{Provider: rec, Registry: reg, MaxIterations: 5, AgentTree: tree, State: map[string]any{}}
	RegisterDelegateTools(parent, tree)

	if _, err := parent.Run(context.Background(), "转交", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if parent.transferTarget != "" {
		t.Errorf("不存在的 agent 不应转移，transferTarget=%q", parent.transferTarget)
	}
}

// TestSharedState 子 agent 通过共享 State 读到父写入的值。
// get_state 工具闭包捕获 state map（即 parent.State），子 Loop 继承同一引用。
func TestSharedState(t *testing.T) {
	state := map[string]any{"secret": "VAL"}
	root := &SubAgent{Name: "coordinator"}
	worker := &SubAgent{Name: "worker"}
	tree := NewAgentTree(root, worker)

	reg := NewRegistry()
	reg.Register(&Tool{
		Name:        "get_state",
		Description: "读取共享状态 secret",
		Parameters:  objSchema(props{}),
		ReadOnly:    true,
		Handler: func(_ context.Context, _ map[string]any) (string, error) {
			return fmt.Sprintf("secret=%v", state["secret"]), nil
		},
	})
	rec := &recordingProvider{responses: []Message{
		{ToolCalls: []ToolCall{delegateTaskCall("d1", "worker", "读 secret")}},
		{ToolCalls: []ToolCall{{ID: "g1", Type: "function", Function: FunctionCall{Name: "get_state", Arguments: `{}`}}}},
		finishTaskMsg("f1", "已读取"),
		{Content: "完成 [FINAL]"},
	}}
	var events []Event
	parent := &Loop{
		Provider:      rec,
		Registry:      reg,
		System:        "你是协调器",
		MaxIterations: 5,
		AgentTree:     tree,
		State:         state,
		OnEvent:       func(e Event) { events = append(events, e) },
	}
	RegisterDelegateTools(parent, tree)

	if _, err := parent.Run(context.Background(), "读状态", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// 子 agent 调 get_state 应返回 "secret=VAL"（证明共享 State 可读）
	var sawVal bool
	for _, e := range events {
		if e.Type == EventToolResult && e.Tool == "get_state" && strings.Contains(e.Content, "VAL") {
			sawVal = true
		}
	}
	if !sawVal {
		t.Errorf("子 agent 应通过共享 State 读到 VAL，events=%+v", events)
	}
}

// TestToolWhitelist 子 agent Tools 白名单裁剪：子 Registry 只含白名单工具 + finish_task。
func TestToolWhitelist(t *testing.T) {
	root := &SubAgent{Name: "coordinator"}
	// worker 只允许 read_file
	worker := &SubAgent{Name: "worker", Tools: []string{"read_file"}}
	tree := NewAgentTree(root, worker)

	reg := NewRegistry()
	noop := func(_ context.Context, _ map[string]any) (string, error) { return "", nil }
	reg.Register(&Tool{Name: "read_file", Handler: noop, ReadOnly: true})
	reg.Register(&Tool{Name: "write_file", Handler: noop, RequiresApproval: true})
	reg.Register(&Tool{Name: "edit_file", Handler: noop, RequiresApproval: true})

	rec := &recordingProvider{responses: []Message{
		{ToolCalls: []ToolCall{delegateTaskCall("d1", "worker", "干活")}},
		finishTaskMsg("f1", "done"),
		{Content: "完成 [FINAL]"},
	}}
	parent := &Loop{
		Provider:      rec,
		Registry:      reg,
		System:        "你是协调器",
		MaxIterations: 5,
		AgentTree:     tree,
		State:         map[string]any{},
	}
	RegisterDelegateTools(parent, tree)

	if _, err := parent.Run(context.Background(), "开始", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rec.Calls() < 2 {
		t.Fatalf("应至少 2 次 LLM 调用，得 %d", rec.Calls())
	}
	// recorded[1] = 子 agent 首次调用，其 tools 应只含 read_file + finish_task
	childTools := rec.recorded[1].tools
	has := func(name string) bool {
		for _, td := range childTools {
			if td.Function.Name == name {
				return true
			}
		}
		return false
	}
	if !has("read_file") {
		t.Errorf("子工具应含白名单 read_file，tools=%+v", childTools)
	}
	if !has("finish_task") {
		t.Errorf("子工具应含 finish_task，tools=%+v", childTools)
	}
	if has("write_file") {
		t.Errorf("子工具不应含白名单外的 write_file，tools=%+v", childTools)
	}
	if has("edit_file") {
		t.Errorf("子工具不应含白名单外的 edit_file，tools=%+v", childTools)
	}
	if has("delegate_task") {
		t.Errorf("子工具不应含 delegate_task(白名单裁剪应剔除)，tools=%+v", childTools)
	}
}

// TestDelegateTask_AgentNotFound 目标 agent 不存在 → 工具报错回灌，父继续。
func TestDelegateTask_AgentNotFound(t *testing.T) {
	root := &SubAgent{Name: "coordinator"}
	tree := NewAgentTree(root)
	reg := NewRegistry()
	rec := &recordingProvider{responses: []Message{
		{ToolCalls: []ToolCall{delegateTaskCall("d1", "ghost", "x")}},
		{Content: "好的 [FINAL]"},
	}}
	parent := &Loop{Provider: rec, Registry: reg, MaxIterations: 5, AgentTree: tree, State: map[string]any{}}
	RegisterDelegateTools(parent, tree)

	msgs, err := parent.Run(context.Background(), "委托", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// 工具结果应含"未找到 agent"
	var sawErr bool
	for _, m := range msgs {
		if m.Role == RoleTool && strings.Contains(m.Content, "未找到 agent") {
			sawErr = true
		}
	}
	if !sawErr {
		t.Errorf("不存在的 agent 应回灌错误，msgs=%+v", msgs)
	}
}
