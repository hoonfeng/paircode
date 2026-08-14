package agent

import (
	"strings"
	"testing"
)

// ── 事件语义标注（消息落盘对齐 deepseek-harness）──────────────────

// TestAnnotateStoredEvents_TurnStep 验证 annotateStoredEvents 按消息序列
// 推导 EventType/Turn/Step（一次 Run = 一个 turn，assistant 消息递增 step，
// tool/result 归并同 step）。
func TestAnnotateStoredEvents_TurnStep(t *testing.T) {
	stored := []StoredMessage{
		{Message: Message{Role: RoleUser, Content: "任务1"}},
		{Message: Message{Role: RoleAssistant, Content: "第一步思考", ToolCalls: []ToolCall{{ID: "c1", Function: FunctionCall{Name: "read_file"}}}}},
		{Message: Message{Role: RoleTool, ToolCallID: "c1", Content: "file content"}},
		{Message: Message{Role: RoleAssistant, Content: "最终回复"}},
		{Message: Message{Role: RoleUser, Content: "任务2（新轮次）"}},
		{Message: Message{Role: RoleAssistant, Content: "回答2"}},
	}
	annotateStoredEvents(stored)

	want := []struct {
		event string
		turn  int
		step  int
	}{
		{EventTypeUserMessage, 1, 0},
		{EventTypeAssistantMessage, 1, 1},
		{EventTypeToolResult, 1, 1},
		{EventTypeAssistantMessage, 1, 2},
		{EventTypeUserMessage, 2, 0},
		{EventTypeAssistantMessage, 2, 1},
	}
	for i, w := range want {
		if stored[i].EventType != w.event {
			t.Errorf("[%d] EventType = %q, want %q", i, stored[i].EventType, w.event)
		}
		if stored[i].Turn != w.turn {
			t.Errorf("[%d] Turn = %d, want %d", i, stored[i].Turn, w.turn)
		}
		if stored[i].Step != w.step {
			t.Errorf("[%d] Step = %d, want %d", i, stored[i].Step, w.step)
		}
	}
}

// TestAnnotateStoredEvents_Empty 验证空列表安全。
func TestAnnotateStoredEvents_Empty(t *testing.T) {
	var stored []StoredMessage
	annotateStoredEvents(stored) // 不应 panic
}

// TestPersistNewMessages_EventAnnotation 端到端：PersistNewMessages 落盘后
// LoadLatest 读回的消息带 EventType/Turn/Step 标注（对齐事件流结构）。
func TestPersistNewMessages_EventAnnotation(t *testing.T) {
	store := NewMessageStore(t.TempDir())
	convID := "conv-evt"
	hist := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "任务"},
		{Role: RoleAssistant, Content: "思考", ToolCalls: []ToolCall{{ID: "c1", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}}}},
		{Role: RoleTool, ToolCallID: "c1", Content: "content"},
		{Role: RoleAssistant, Content: "完成"},
	}
	if err := store.PersistNewMessages(convID, hist); err != nil {
		t.Fatalf("PersistNewMessages: %v", err)
	}
	loaded, total, err := store.LoadLatest(convID, 10)
	if err != nil {
		t.Fatalf("LoadLatest: %v", err)
	}
	if total != 4 { // system 不落盘
		t.Fatalf("total = %d, want 4", total)
	}
	if loaded[0].EventType != EventTypeUserMessage || loaded[0].Turn != 1 || loaded[0].Step != 0 {
		t.Errorf("user 标注错误: %+v", loaded[0])
	}
	if loaded[1].EventType != EventTypeAssistantMessage || loaded[1].Turn != 1 || loaded[1].Step != 1 {
		t.Errorf("assistant(tool) 标注错误: %+v", loaded[1])
	}
	if loaded[2].EventType != EventTypeToolResult || loaded[2].Step != 1 {
		t.Errorf("tool/result 应归并 step 1: %+v", loaded[2])
	}
	if loaded[3].EventType != EventTypeAssistantMessage || loaded[3].Step != 2 {
		t.Errorf("assistant(final) 标注错误: %+v", loaded[3])
	}
}

// ── 崩溃恢复修复（对齐 TOOL_OUTCOME_UNKNOWN）─────────────────────

// TestRepairInterruptedHistory_ToolUnknown 验证：中断的 assistant（有 tool_call
// 缺 result）被保留，缺结果的调用合成「结果未知」提示，消息序列完整。
func TestRepairInterruptedHistory_ToolUnknown(t *testing.T) {
	hist := []Message{
		{Role: RoleUser, Content: "任务"},
		{Role: RoleAssistant, Content: "已调用工具", ToolCalls: []ToolCall{
			{ID: "c1", Function: FunctionCall{Name: "write_file"}},
			{ID: "c2", Function: FunctionCall{Name: "read_file"}},
		}},
		{Role: RoleTool, ToolCallID: "c1", Content: "写入成功"},
		// c2 无 result → 中断
	}
	out := RepairInterruptedHistory(hist)
	if len(out) != 3 {
		t.Fatalf("len = %d, want 3（assistant 保留 + c2 合成 result）", len(out))
	}
	// assistant 保留（思考链不丢）
	if out[1].Role != RoleAssistant {
		t.Errorf("assistant 应保留: %+v", out[1])
	}
	// c2 合成 tool/result（TOOL_OUTCOME_UNKNOWN 语义）
	if out[2].Role != RoleTool || out[2].ToolCallID != "c2" {
		t.Fatalf("应合成 c2 的 tool/result: %+v", out[2])
	}
	if !strings.Contains(out[2].Content, "TOOL_OUTCOME_UNKNOWN") {
		t.Errorf("合成 result 应含结果未知提示，实际=%q", out[2].Content)
	}
	// 完整 result 的 c1 不重复合成
	for _, m := range out {
		if m.Role == RoleTool && m.ToolCallID == "c1" && m.Content != "写入成功" {
			t.Errorf("c1 的原始 result 不应被覆盖: %+v", m)
		}
	}
}

// TestRepairInterruptedHistory_KeepNextUser 验证：中断后预写入的新用户消息保留。
func TestRepairInterruptedHistory_KeepNextUser(t *testing.T) {
	hist := []Message{
		{Role: RoleUser, Content: "任务1"},
		{Role: RoleAssistant, Content: "", ToolCalls: []ToolCall{{ID: "c1", Function: FunctionCall{Name: "read_file"}}}},
		{Role: RoleTool, ToolCallID: "c1", Content: "中途中断，c1 无后续"},
		{Role: RoleUser, Content: "继续（新任务）"},
	}
	out := RepairInterruptedHistory(hist)
	// 最后一条 assistant（无 tool_call 的空回复）不在此场景——这里最后是 user
	// 场景：末尾 user → 前面 assistant 有 tool_call 无 result → 修复
	if len(out) < 3 {
		t.Fatalf("修复后应保留新任务，len=%d", len(out))
	}
	last := out[len(out)-1]
	if last.Role != RoleUser || last.Content != "继续（新任务）" {
		t.Errorf("新任务应保留在末尾: %+v", last)
	}
}

// TestRepairInterruptedHistory_NaturalComplete 验证：自然完成的对话原样保留。
func TestRepairInterruptedHistory_NaturalComplete(t *testing.T) {
	hist := []Message{
		{Role: RoleUser, Content: "任务"},
		{Role: RoleAssistant, Content: "直接回答，无工具"},
	}
	out := RepairInterruptedHistory(hist)
	if len(out) != 2 {
		t.Fatalf("自然完成不应改动: %+v", out)
	}
}

// TestRepairInterruptedHistory_EmptyAssistant 验证：空 assistant（无正文无 tool_call）
// 及其后的孤立消息被截断，但后续 user 保留。
func TestRepairInterruptedHistory_EmptyAssistant(t *testing.T) {
	hist := []Message{
		{Role: RoleUser, Content: "任务"},
		{Role: RoleAssistant, Content: "正常回复"},
		{Role: RoleAssistant, Content: ""}, // 空回合（中断残留）
		{Role: RoleUser, Content: "新任务"},
	}
	out := RepairInterruptedHistory(hist)
	if len(out) != 3 {
		t.Fatalf("应截断空 assistant 并保留新任务，len=%d", len(out))
	}
	if out[2].Content != "新任务" {
		t.Errorf("新任务应保留在末尾: %+v", out[2])
	}
}
