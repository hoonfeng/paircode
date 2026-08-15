package agent

import (
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

