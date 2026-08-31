package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// ── agentloop（turn/step 双层循环）测试 ─────────

// TestLoopTurnStepFields 验证 turn/step 序号递增、结构化结束原因、事件携带 turn/step。
func TestLoopTurnStepFields(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read", Arguments: `{"path":"nope.txt"}`}}}},
		{Content: "任务完成"},
	}}
	var events []Event
	loop := &Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5,
		OnEvent: func(e Event) { events = append(events, e) }}

	msgs, err := loop.Run(context.Background(), "测试 turn/step", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	_ = msgs

	// turn 序号：一次 Run = 一个 turn
	if loop.TurnNo != 1 {
		t.Errorf("TurnNo 应为 1，得 %d", loop.TurnNo)
	}
	// step 序号：两次 LLM 调用 = 两个 step
	if loop.StepNo != 2 {
		t.Errorf("StepNo 应为 2，得 %d", loop.StepNo)
	}
	// 结构化结束原因：自然终止 → completed
	if loop.LastTurnReason != TurnCompleted {
		t.Errorf("LastTurnReason 应为 completed，得 %q", loop.LastTurnReason)
	}

	// 事件应携带 turn/step 序号，且 EventDone 带 TurnReason
	var sawDone bool
	var sawCallStep1, sawResultStep1 bool
	for _, e := range events {
		if e.Turn != 1 {
			t.Errorf("事件 %s 的 Turn 应为 1，得 %d", e.Type, e.Turn)
		}
		if e.Type == EventToolCall && e.Step != 1 {
			t.Errorf("tool_call 事件 Step 应为 1，得 %d", e.Step)
		}
		if e.Type == EventToolCall {
			sawCallStep1 = true
		}
		if e.Type == EventToolResult {
			sawResultStep1 = true
		}
		if e.Type == EventDone {
			sawDone = true
			if e.Step != 2 {
				t.Errorf("done 事件 Step 应为 2，得 %d", e.Step)
			}
			if e.TurnReason != string(TurnCompleted) {
				t.Errorf("done 事件 TurnReason 应为 completed，得 %q", e.TurnReason)
			}
			if e.DoneReason != "task_complete" {
				t.Errorf("done 事件 DoneReason 应保持 task_complete 兼容，得 %q", e.DoneReason)
			}
		}
	}
	if !sawCallStep1 || !sawResultStep1 || !sawDone {
		t.Errorf("缺事件：tool_call=%v tool_result=%v done=%v", sawCallStep1, sawResultStep1, sawDone)
	}
}

// TestLoopPreStepReject 验证 pre-step 钩子拒绝 → turn 以 blocked 结束，不调用 LLM。
func TestLoopPreStepReject(t *testing.T) {
	reg := NewRegistry()
	mock := &MockProvider{Responses: []Message{{Content: "不应被调用"}}}
	var events []Event
	loop := &Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5,
		OnEvent: func(e Event) { events = append(events, e) },
		// 拒绝所有 step → turn 以 blocked 结束
		PreStep: func(ctx context.Context, callMsgs []Message, turn, step int) ([]Message, bool, error) {
			if turn != 1 || step != 1 {
				t.Errorf("PreStep 应收到 turn=1 step=1，得 turn=%d step=%d", turn, step)
			}
			return nil, true, nil
		},
	}

	msgs, err := loop.Run(context.Background(), "测试拒绝", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if mock.Calls() != 0 {
		t.Errorf("PreStep 拒绝后不应调用 LLM，得 %d 次", mock.Calls())
	}
	if loop.LastTurnReason != TurnBlocked {
		t.Errorf("LastTurnReason 应为 blocked，得 %q", loop.LastTurnReason)
	}
	// blocked turn 不应产生 assistant/tool 消息（system+task 组装发生在 pre-step 之前，允许存在）
	for _, m := range msgs {
		if m.Role == RoleAssistant || m.Role == RoleTool {
			t.Errorf("blocked turn 不应有 assistant/tool 消息，得 %+v", m)
		}
	}
	// 末事件应为 done(blocked)
	last := events[len(events)-1]
	if last.Type != EventDone || last.DoneReason != "blocked" || last.TurnReason != string(TurnBlocked) {
		t.Errorf("末事件应为 EventDone(blocked)，得 %+v", last)
	}
}

// stopReasonProvider 可控制每轮 Chat 的 stop_reason（测试 max-tokens sticky 用）。
type stopReasonProvider struct {
	responses   []Message
	stopReasons []string
	calls       int
	capture     *[]Message // 非空时捕获每次进入模型的 messages
}

func (m *stopReasonProvider) Name() string { return "mock-stop" }
func (m *stopReasonProvider) Calls() int   { return m.calls }

func (m *stopReasonProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	var msg Message
	if m.calls < len(m.responses) {
		msg = m.responses[m.calls]
	} else {
		msg = Message{Role: RoleAssistant, Content: "完成"} // 脚本耗尽兜底
	}
	if m.capture != nil {
		*m.capture = messages
	}
	reason := ""
	if m.calls < len(m.stopReasons) {
		reason = m.stopReasons[m.calls]
	}
	m.calls++
	if msg.Role == "" {
		msg.Role = RoleAssistant
	}
	if onChunk != nil {
		onChunk(Chunk{Content: msg.Content, Reasoning: msg.Reasoning, ToolCalls: msg.ToolCalls, Done: true, StopReason: reason})
	}
	return msg, nil
}

// TestLoopPreStepRewrite 验证 pre-step 钩子改写输入 → LLM 收到改写后的消息。
func TestLoopPreStepRewrite(t *testing.T) {
	var got []Message
	prov := &stopReasonProvider{responses: []Message{{Content: "完成"}}, capture: &got}
	loop := &Loop{Provider: prov, Registry: NewRegistry(), System: "test", MaxIterations: 5,
		PreStep: func(ctx context.Context, callMsgs []Message, turn, step int) ([]Message, bool, error) {
			// 追加一条引导消息，验证 LLM 收到改写后的上下文
			rewritten := append([]Message(nil), callMsgs...)
			rewritten = append(rewritten, Message{Role: RoleUser, Content: "【pre-step 注入】请先总结"})
			return rewritten, false, nil
		},
	}

	if _, err := loop.Run(context.Background(), "测试改写", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if loop.LastTurnReason != TurnCompleted {
		t.Errorf("LastTurnReason 应为 completed，得 %q", loop.LastTurnReason)
	}
	found := false
	for _, m := range got {
		if m.Role == RoleUser && strings.Contains(m.Content, "pre-step 注入") {
			found = true
		}
	}
	if !found {
		t.Errorf("LLM 未收到 pre-step 改写后的消息，得 %d 条", len(got))
	}
}

// TestLoopMaxTokensSticky 验证 max-tokens sticky：任一 step 触顶后，turn 结束原因不得降级。
func TestLoopMaxTokensSticky(t *testing.T) {
	reg := NewRegistry()
	// 第 1 轮：tool call + stop_reason=length（截断）→ hadMaxTokens=true
	// 第 2 轮：正常完成（无 tool call）→ 结果不得降级为 completed
	prov := &stopReasonProvider{
		responses: []Message{
			{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read", Arguments: `{"path":"x.txt"}`}}}},
			{Content: "总结完成"},
		},
		stopReasons: []string{"length", "stop"},
	}
	loop := &Loop{Provider: prov, Registry: reg, System: "test", MaxIterations: 5}

	if _, err := loop.Run(context.Background(), "测试 max-tokens", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !loop.hadMaxTokens {
		t.Error("hadMaxTokens 应为 true（曾触发截断）")
	}
	if loop.LastTurnReason != TurnMaxTokens {
		t.Errorf("LastTurnReason 应为 max-tokens（sticky），得 %q", loop.LastTurnReason)
	}
}

// TestLoopCancelAborted 验证 ctx 取消 → turn 以 aborted 结束并记录取消原因。
func TestLoopCancelAborted(t *testing.T) {
	reg := NewRegistry()
	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read", Arguments: `{"path":"x.txt"}`}}}},
		{Content: "继续"},
	}}
	loop := &Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消
	_, err := loop.Run(ctx, "测试取消", nil)
	if err == nil {
		t.Fatal("ctx 取消后 Run 应返回错误")
	}
	if loop.LastTurnReason != TurnAborted {
		t.Errorf("LastTurnReason 应为 aborted，得 %q", loop.LastTurnReason)
	}
	if loop.CancelCause.Kind != CancelByContext {
		t.Errorf("CancelCause.Kind 应为 context，得 %q", loop.CancelCause.Kind)
	}
	if loop.TurnNo != 1 {
		t.Errorf("TurnNo 应为 1（turn 已打开），得 %d", loop.TurnNo)
	}
}

// TestLoopFollowUpOpensNextTurn 验证 follow-up 消息驱动继续（类似 next-turn 输入）。
// 注意：连续两轮纯文字会被 content-only 防护捕获（现有行为），故 follow-up 后应调工具。
func TestLoopFollowUpOpensNextTurn(t *testing.T) {
	reg := NewRegistry()
	// 第 1 轮自然终止（有正文无工具）→ followUpQueue 非空 → 注入继续
	// 第 2 轮调用工具 → 第 3 轮自然终止 → 完成
	mock := &MockProvider{Responses: []Message{
		{Content: "第一段完成"},
		{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read", Arguments: `{"path":"x.txt"}`}}}},
		{Content: "第二段完成"},
	}}
	loop := &Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5}
	loop.followUpQueue = []Message{{Role: RoleUser, Content: "继续做第二件事"}}

	msgs, err := loop.Run(context.Background(), "测试 follow-up", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if mock.Calls() != 3 {
		t.Errorf("应调用 LLM 3 次（follow-up 驱动第 2/3 次），得 %d", mock.Calls())
	}
	if loop.LastTurnReason != TurnCompleted {
		t.Errorf("LastTurnReason 应为 completed，得 %q", loop.LastTurnReason)
	}
	if len(msgs) < 6 {
		t.Errorf("应包含 user+assistant+user+assistant+tool+assistant 至少 6 条消息，得 %d", len(msgs))
	}
}

// TestEndTurnInference 验证 endTurn 兜底推断（未显式设置时按 err/ctx）。
func TestEndTurnInference(t *testing.T) {
	loop := &Loop{}
	loop.endTurn(nil, false)
	if loop.LastTurnReason != TurnCompleted {
		t.Errorf("无错误无取消 → completed，得 %q", loop.LastTurnReason)
	}

	loop2 := &Loop{}
	loop2.endTurn(errors.New("boom"), false)
	if loop2.LastTurnReason != TurnError {
		t.Errorf("有错误 → error，得 %q", loop2.LastTurnReason)
	}

	loop3 := &Loop{}
	loop3.endTurn(nil, true)
	if loop3.LastTurnReason != TurnAborted {
		t.Errorf("ctx 已取消 → aborted，得 %q", loop3.LastTurnReason)
	}

	// 已显式设置 → 不覆盖
	loop4 := &Loop{LastTurnReason: TurnBlocked}
	loop4.endTurn(nil, false)
	if loop4.LastTurnReason != TurnBlocked {
		t.Errorf("显式设置不应被覆盖，得 %q", loop4.LastTurnReason)
	}
}
