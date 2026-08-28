// 落盘事件标注一致性：验证「消息序列推导的 Turn/Step」与「agentloop 实际
// TurnNo/StepNo 及 [turn/N/start]/[step/N.M/start] 事件」完全一致。
// 这是渐进对齐方案的核心不变量（见 message_store.go 顶部注释）：
//
//	落盘 user=每轮一个 → turn 递增；assistant=每 step 一个 → step 递增；
//	tool/result 归并同 step；推导结果必须与 loop 运行时编号一致。
package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestLoopPersistTurnStepConsistency(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	// 3 次 LLM 调用：step1=tool call，step2=tool call，step3=纯文本收尾。
	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"nope.txt"}`}}}},
		{ToolCalls: []ToolCall{{ID: "c2", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"nope2.txt"}`}}}},
		{Content: "任务完成"},
	}}
	var events []Event
	loop := &Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5,
		OnEvent: func(e Event) { events = append(events, e) }}

	msgs, err := loop.Run(context.Background(), "一致性验证", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// ── 1) agentloop 实际编号 ──
	if loop.TurnNo != 1 {
		t.Errorf("TurnNo 应为 1，得 %d", loop.TurnNo)
	}
	if loop.StepNo != 3 {
		t.Errorf("StepNo 应为 3，得 %d", loop.StepNo)
	}

	// ── 2) 事件流中的 step/start 编号（每步应恰好一个，编号 1..3）──
	var stepStarts []int
	for _, e := range events {
		if e.Type == EventNotice {
			var turn, step int
			if n, _ := fmt.Sscanf(e.Content, "[step/%d.%d/start]", &turn, &step); n == 2 {
				// ★ Sscanf 顺序匹配：对 [step/1.1/end] 也会返回 n=2（前两个 %d 已成功）。
				// 必须校验完整前缀，避免 end 事件被误判为 start。
				prefix := fmt.Sprintf("[step/%d.%d/start]", turn, step)
				if !strings.HasPrefix(e.Content, prefix) {
					continue
				}
				if turn != 1 {
					t.Errorf("step/start 的 turn 应为 1，得 %d（内容 %q）", turn, e.Content)
				}
				stepStarts = append(stepStarts, step)
			}
		}
	}
	if len(stepStarts) != 3 {
		t.Fatalf("应发 3 个 step/start 事件，得 %d（%v）", len(stepStarts), stepStarts)
	}
	for i, s := range stepStarts {
		if s != i+1 {
			t.Errorf("step/start 编号应依次为 1,2,3，得 %v", stepStarts)
			break
		}
	}

	// ── 3) 模拟落盘标注：对 Run 返回的消息序列推导 Turn/Step（与
	//      PersistNewMessages 全量重写时的 annotateStoredEvents 同路径）──
	if len(msgs) == 0 {
		t.Fatal("Run 返回空消息序列")
	}
	stored := make([]StoredMessage, len(msgs))
	for i, m := range msgs {
		stored[i] = StoredMessage{Message: m}
	}
	annotateStoredEvents(stored)

	// 消息序列形态：system | user | assistant(tool_call) | tool | assistant(tool_call) | tool | assistant(content)
	// ★ 背景上下文快照（记忆存在时）为额外 user 消息（任务之后，带 backgroundCtxMarker），
	//   不计入「7 条主序列」；turnStepFor 已跳过快照（不递增 turn）。
	mainLen := 7
	if strings.HasPrefix(stored[len(msgs)-1].Message.Content, backgroundCtxMarker) {
		mainLen = len(stored) - 1
	}
	if len(stored) != 7 && len(stored) != mainLen && len(stored) != 8 {
		t.Errorf("消息序列应为 7 条（system+user+3assistant+2tool，快照可选 +1），得 %d", len(stored))
	}

	// ── 4) 落盘推导 Turn/Step vs agentloop 编号 ──
	// user → turn=1, step=0；assistant → step=1,2,3；tool → 归并前一个 assistant 的 step
	var userSeen bool
	var assistantSteps []int
	for i, sm := range stored {
		switch sm.Message.Role {
		case RoleUser:
			userSeen = true
			if sm.EventType != EventTypeUserMessage {
				t.Errorf("user 消息 EventType 应为 %q，得 %q", EventTypeUserMessage, sm.EventType)
			}
			if sm.Turn != 1 || sm.Step != 0 {
				t.Errorf("user 消息应 Turn=1 Step=0，得 Turn=%d Step=%d", sm.Turn, sm.Step)
			}
		case RoleAssistant:
			if !userSeen {
				t.Errorf("第 %d 条 assistant 前应有 user 消息", i)
			}
			if sm.EventType != EventTypeAssistantMessage {
				t.Errorf("assistant 消息 EventType 应为 %q，得 %q", EventTypeAssistantMessage, sm.EventType)
			}
			if sm.Turn != 1 {
				t.Errorf("assistant 消息 Turn 应为 1，得 %d", sm.Turn)
			}
			assistantSteps = append(assistantSteps, sm.Step)
		case RoleTool:
			if sm.EventType != EventTypeToolResult {
				t.Errorf("tool 消息 EventType 应为 %q，得 %q", EventTypeToolResult, sm.EventType)
			}
			if sm.Step != len(assistantSteps) {
				t.Errorf("第 %d 条 tool 消息应归并到前一个 assistant 的 step %d，得 Step=%d",
					i, len(assistantSteps), sm.Step)
			}
		}
	}
	// assistant step 递增：1,2,3（与 agentloop StepNo 一致）
	if len(assistantSteps) != 3 || assistantSteps[0] != 1 || assistantSteps[1] != 2 || assistantSteps[2] != 3 {
		t.Errorf("assistant 消息 Step 应依次为 1,2,3，得 %v", assistantSteps)
	}
}
