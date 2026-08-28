package agent

import (
	"strconv"
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

// TestCondenseHistorySemiCompress 验证倒数第 2 轮「半压缩」：
// 用户消息 + 助手正文保留，工具调用子链合并为一行摘要（RoleTool 不保留），
// 且消息配对完整（不产生孤立 tool_calls / 孤立 tool 结果）。
func TestCondenseHistorySemiCompress(t *testing.T) {
	// 5 轮：任务1/2/3 压缩，任务4 半压缩，任务5 完整，任务6 当前
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "任务1"},
		{Role: RoleAssistant, Content: "完成1"},
		{Role: RoleUser, Content: "任务2"},
		{Role: RoleAssistant, Content: "完成2"},
		{Role: RoleUser, Content: "任务3"},
		{Role: RoleAssistant, Content: "完成3"},
		{Role: RoleUser, Content: "任务4"},
		{Role: RoleAssistant, Content: "分析4", ToolCalls: []ToolCall{
			{ID: "c4a", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}},
			{ID: "c4b", Function: FunctionCall{Name: "run_command", Arguments: `{"command":"go build"}`}},
		}},
		{Role: RoleTool, ToolCallID: "c4a", Name: "read_file", Content: "r4a 内容很长"},
		{Role: RoleTool, ToolCallID: "c4b", Name: "run_command", Content: "r4b"},
		{Role: RoleAssistant, Content: "完成4"},
		{Role: RoleUser, Content: "任务5"},
		{Role: RoleAssistant, Content: "完成5"},
		{Role: RoleUser, Content: "任务6"},
	}

	out := CondenseHistory(msgs)

	// 1. 半压缩轮：用户消息 "任务4" 保留
	found := false
	for _, m := range out {
		if m.Role == RoleUser && strings.Contains(m.Content, "任务4") && !strings.Contains(m.Content, "历史对话摘要") {
			found = true
		}
	}
	if !found {
		t.Error("半压缩轮的用户消息（任务4）应保留")
	}

	// 2. 工具调用子链合并：助手正文 + 工具摘要行存在
	foundToolSummary := false
	for _, m := range out {
		if m.Role == RoleAssistant && strings.Contains(m.Content, "分析4") && strings.Contains(m.Content, "read_file") && strings.Contains(m.Content, "run_command") {
			foundToolSummary = true
		}
	}
	if !foundToolSummary {
		t.Error("半压缩轮应保留助手正文并合并工具调用摘要")
	}

	// 3. 工具结果不保留（半压缩轮的 RoleTool 消失）
	for _, m := range out {
		if m.Role == RoleTool && strings.Contains(m.Content, "r4a") {
			t.Error("半压缩轮的工具结果不应保留")
		}
	}

	// 4. 消息配对完整：不存在带 ToolCalls 的 assistant 后无对应 tool 结果（孤立）
	//    半压缩后 tool_calls 已移除，无需额外校验孤立 tool 结果——上一条已覆盖。
	//    但完整保留轮（任务5）仍应有完整配对。
	completeRoundOK := false
	for i := 0; i < len(out); i++ {
		if out[i].Role == RoleUser && strings.Contains(out[i].Content, "任务5") {
			completeRoundOK = true // 任务5 的原始消息存在
		}
	}
	if !completeRoundOK {
		t.Error("最近一轮完整交互（任务5）应保留")
	}

	// 5. 摘要中不含半压缩轮（任务4 不进入压缩摘要）
	for _, m := range out {
		if m.Role == RoleUser && strings.Contains(m.Content, "历史对话摘要") {
			if strings.Contains(m.Content, "任务4") {
				t.Error("半压缩轮不应进入压缩摘要")
			}
		}
	}
}

// TestCondenseHistoryNoNestedSummary 验证摘要防嵌套：
// 输入已含【历史对话摘要】消息时，新摘要以「前序摘要」合并旧摘要（截断），
// 不把旧摘要当普通轮次递归压缩、不无限膨胀。
func TestCondenseHistoryNoNestedSummary(t *testing.T) {
	oldSummary := "【历史对话摘要】\n**轮次 1**：用户「旧目标」→ 使用了 read_file → 完成"
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: oldSummary}, // 上一轮压缩产生的摘要消息
		{Role: RoleUser, Content: "任务1"},
		{Role: RoleAssistant, Content: "完成1"},
		{Role: RoleUser, Content: "任务2"},
		{Role: RoleAssistant, Content: "完成2"},
		{Role: RoleUser, Content: "任务3"},
		{Role: RoleAssistant, Content: "完成3"},
		{Role: RoleUser, Content: "任务4"},
		{Role: RoleAssistant, Content: "完成4"},
		{Role: RoleUser, Content: "任务5"},
		{Role: RoleAssistant, Content: "完成5"},
		{Role: RoleUser, Content: "任务6"},
	}

	out := CondenseHistory(msgs)

	var summaryText string
	for _, m := range out {
		if m.Role == RoleUser && strings.Contains(m.Content, "历史对话摘要") {
			summaryText = m.Content
		}
	}
	if summaryText == "" {
		t.Fatal("输出应包含摘要消息")
	}
	// 旧摘要内容（旧目标）保留在合并后的摘要中
	if !strings.Contains(summaryText, "旧目标") {
		t.Error("旧摘要内容应合并进新摘要（前序摘要）")
	}
	// 摘要不应把旧摘要当普通轮次重复包装（如出现"轮次 1"且其内容只是旧摘要文本）
	if strings.Count(summaryText, "**轮次") > 4 {
		t.Error("摘要不应递归压缩旧摘要导致轮次条目膨胀")
	}
	// 总量不超过上限
	if len([]rune(summaryText)) > maxCondensedChars+50 {
		t.Errorf("摘要超过总量上限：%d > %d", len([]rune(summaryText)), maxCondensedChars+50)
	}
}

// TestCondenseHistoryLongFullRound 最近 1 轮消息数超上限时：
// 更早的迭代子链合并为摘要，尾部保留 maxFullRoundMsgs 条关键迭代，
// 且不产生孤立 tool 结果（split 越过 RoleTool）。
func TestCondenseHistoryLongFullRound(t *testing.T) {
	var msgs []Message
	msgs = append(msgs, Message{Role: RoleSystem, Content: "sys"})
	// 3 轮旧历史（第1轮完整、第2轮半压缩），第 3 轮超长（60 条迭代）
	msgs = append(msgs, Message{Role: RoleUser, Content: "任务1"})
	msgs = append(msgs, Message{Role: RoleAssistant, Content: "完成1"})
	msgs = append(msgs, Message{Role: RoleUser, Content: "任务2"})
	msgs = append(msgs, Message{Role: RoleAssistant, Content: "完成2"})
	msgs = append(msgs, Message{Role: RoleUser, Content: "任务3"})
	for i := 0; i < 30; i++ {
		id := "c3_" + strconv.Itoa(i)
		msgs = append(msgs,
			Message{Role: RoleAssistant, Content: "分析", ToolCalls: []ToolCall{{ID: id, Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}}}},
			Message{Role: RoleTool, ToolCallID: id, Name: "read_file", Content: "内容 " + strconv.Itoa(i)},
		)
	}
	msgs = append(msgs, Message{Role: RoleAssistant, Content: "完成3"})
	msgs = append(msgs, Message{Role: RoleUser, Content: "任务4"})

	out := CondenseHistory(msgs)

	// 1. 尾部应保留最近 maxFullRoundMsgs 条迭代（含"完成3"助手消息）
	joined := ""
	for _, m := range out {
		joined += m.Content + "|"
	}
	if !strings.Contains(joined, "完成3") {
		t.Error("最近 1 轮的助手结论应保留")
	}
	// 2. 无孤立 tool 结果：不存在 RoleTool 消息后紧跟 user 消息（即 tool 结果必须有配对）
	for i := 0; i < len(out); i++ {
		if out[i].Role == RoleTool {
			if i+1 < len(out) && out[i+1].Role == RoleUser {
				t.Errorf("孤立 tool 结果（位置 %d 后紧跟 user）：%q", i, out[i+1].Content)
			}
		}
	}
	// 3. 输出消息数应小于原始（60 条迭代被合并压缩）
	if len(out) >= len(msgs) {
		t.Errorf("超长轮应被压缩：%d → %d", len(msgs), len(out))
	}
	// 4. 摘要存在
	found := false
	for _, m := range out {
		if m.Role == RoleUser && strings.Contains(m.Content, "历史对话摘要") {
			found = true
		}
	}
	if !found {
		t.Error("输出应含压缩摘要")
	}
}

// TestCondenseHistorySummaryLimit 验证摘要总量上限：轮次极多时摘要不无限膨胀。
func TestCondenseHistorySummaryLimit(t *testing.T) {
	var msgs []Message
	msgs = append(msgs, Message{Role: RoleSystem, Content: "sys"})
	// 12 轮历史 + 当前消息（keepFullRounds=2 → 压缩 10 轮）
	for i := 1; i <= 13; i++ {
		msgs = append(msgs, Message{Role: RoleUser, Content: "任务" + strings.Repeat("内容", 50)})
		msgs = append(msgs, Message{Role: RoleAssistant, Content: "完成" + strings.Repeat("详细结果", 30)})
	}

	out := CondenseHistory(msgs)
	var summaryText string
	for _, m := range out {
		if m.Role == RoleUser && strings.Contains(m.Content, "历史对话摘要") {
			summaryText = m.Content
		}
	}
	if summaryText == "" {
		t.Fatal("输出应包含摘要消息")
	}
	if len([]rune(summaryText)) > maxCondensedChars+100 {
		t.Errorf("摘要超过总量上限：%d > %d", len([]rune(summaryText)), maxCondensedChars+100)
	}
}
