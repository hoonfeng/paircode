package agent

import (
	"encoding/json"
	"testing"
)

// ★ 2026-08-22 WS 断线补偿快照：buildSnapshotPayload 必须携带有序 events 序列，
// 前端按序重放重建 segments（修复「工具调用全聚在上、正文全聚在下」的时序丢失）。
func TestBuildSnapshotPayloadCarriesOrderedEvents(t *testing.T) {
	m := NewSessionManager()
	convID := "conv_snap_test"
	// 直接构造 running session（同包可访问私有字段）
	l := &Loop{}
	l.emit(Event{Type: EventThinking, Content: "分析"})
	l.emit(Event{Type: EventContent, Content: "先看代码"})
	l.emit(Event{Type: EventToolCall, Tool: "read_file", Args: `{"path":"main.go"}`, CallID: "c1"})
	l.emit(Event{Type: EventContent, Content: "继续"})
	m.mu.Lock()
	m.sessions[convID] = &Session{ConvID: convID, Running: true, Loop: l}
	m.mu.Unlock()

	payload := buildSnapshotPayload(m)
	if payload == nil {
		t.Fatal("快照 payload 为 nil")
	}
	var arr []map[string]any
	if err := json.Unmarshal(payload, &arr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(arr) != 1 {
		t.Fatalf("应 1 条快照，得 %d", len(arr))
	}
	snap := arr[0]
	if snap["type"] != "snapshot" || snap["convId"] != convID {
		t.Errorf("快照元数据: %+v", snap)
	}
	evs, ok := snap["events"].([]any)
	if !ok {
		t.Fatalf("快照缺 events 字段: %+v", snap)
	}
	want := []string{"thinking", "content", "tool_call", "content"}
	if len(evs) != len(want) {
		t.Fatalf("events 长度 %d != %d: %v", len(evs), len(want), evs)
	}
	for i, w := range want {
		ev := evs[i].(map[string]any)
		if ev["type"] != w {
			t.Errorf("event[%d].type=%v 期望 %s", i, ev["type"], w)
		}
	}
	// 顺序保真：content 在 tool_call 前后均有（不聚集成工具全上/正文全下）
	if evs[1].(map[string]any)["content"] != "先看代码" {
		t.Errorf("event[1] content=%v", evs[1])
	}
	if evs[2].(map[string]any)["callId"] != "c1" {
		t.Errorf("event[2] callId=%v", evs[2])
	}
}
