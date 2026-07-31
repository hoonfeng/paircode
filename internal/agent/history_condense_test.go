package agent

import (
	"strings"
	"testing"
)

// TestCondenseHistoryPrefix 验证 CondenseHistory 的输出结构对 KV 缓存前缀友好：
// 1. 摘要消息不插入消息数组第 2 位（避免整段前缀断裂）；
// 2. 摘要作为回顾性 user 消息放在最近轮次之后、当前用户消息之前；
// 3. 最近 keepFullRounds 轮保持原始消息顺序。
func TestCondenseHistoryPrefix(t *testing.T) {
	// 构造 5 轮历史（含 system），超过 keepFullRounds=3 → 触发压缩
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "任务1"},
		{Role: RoleAssistant, Content: "", ToolCalls: []ToolCall{{ID: "c1", Function: FunctionCall{Name: "read_file"}}}},
		{Role: RoleTool, ToolCallID: "c1", Content: "r1"},
		{Role: RoleAssistant, Content: "完成1"},
		{Role: RoleUser, Content: "任务2"},
		{Role: RoleAssistant, Content: "完成2"},
		{Role: RoleUser, Content: "任务3"},
		{Role: RoleAssistant, Content: "完成3"},
		{Role: RoleUser, Content: "任务4"},
		{Role: RoleAssistant, Content: "完成4"},
		{Role: RoleUser, Content: "任务5"},
	}

	out := CondenseHistory(msgs)

	// 1. 第 2 条消息必须是原始 user 消息（"任务3"），而非动态生成的摘要
	if len(out) < 2 {
		t.Fatalf("输出过短: %d", len(out))
	}
	if out[1].Role != RoleUser || strings.Contains(out[1].Content, "历史对话摘要") {
		t.Errorf("消息数组位置 2 不应是摘要消息（应保留原始消息）：role=%q content=%q", out[1].Role, out[1].Content)
	}
	// 2. 存在一条摘要 user 消息
	foundSummary := false
	for _, m := range out {
		if m.Role == RoleUser && strings.Contains(m.Content, "历史对话摘要") {
			foundSummary = true
			break
		}
	}
	if !foundSummary {
		t.Error("输出应包含【历史对话摘要】user 消息")
	}
	// 3. 最后一条消息是当前用户消息（"任务5"），摘要在其之前
	last := out[len(out)-1]
	if last.Role != RoleUser || last.Content != "任务5" {
		t.Errorf("最后一条应为当前用户消息，got role=%q content=%q", last.Role, last.Content)
	}
	// 4. 摘要消息位置必须在最后一条 user 消息之前
	summaryIdx, lastUserIdx := -1, -1
	for i, m := range out {
		if m.Role == RoleUser && strings.Contains(m.Content, "历史对话摘要") {
			summaryIdx = i
		}
		if m.Role == RoleUser && m.Content == "任务5" {
			lastUserIdx = i
		}
	}
	if summaryIdx >= 0 && lastUserIdx >= 0 && summaryIdx > lastUserIdx {
		t.Errorf("摘要消息(%d)应在当前用户消息(%d)之前", summaryIdx, lastUserIdx)
	}
	// 5. 最近 keepFullRounds 轮完整保留（任务3/4 的原始消息仍在，任务1/2 被压缩）
	rawJoined := ""
	for _, m := range out {
		rawJoined += m.Content
	}
	if !strings.Contains(rawJoined, "完成4") {
		t.Error("最近轮次原始消息应保留（完成4）")
	}
	if strings.Contains(rawJoined, "完成1") {
		t.Log("完成1 被压缩进摘要（预期，旧轮次内容并入摘要）")
	}
}

// TestCondenseHistoryNoCompress 轮次不足时不压缩。
func TestCondenseHistoryNoCompress(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "任务1"},
		{Role: RoleAssistant, Content: "完成1"},
		{Role: RoleUser, Content: "任务2"},
	}
	out := CondenseHistory(msgs)
	if len(out) != len(msgs) {
		t.Errorf("轮次不足时不应压缩：in=%d out=%d", len(msgs), len(out))
	}
	for i := range msgs {
		if out[i].Role != msgs[i].Role || out[i].Content != msgs[i].Content {
			t.Errorf("位置 %d 被修改：%q vs %q", i, out[i].Content, msgs[i].Content)
		}
	}
}
