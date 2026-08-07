package agent

import (
	"context"
	"strings"
	"testing"
)

// userMsg 构造一条用户消息。
func userMsg(content string) Message { return Message{Role: RoleUser, Content: content} }

// TestMarkHistoryUserMessages_单轮不标注 首次对话（无历史用户消息）不加前缀。
func TestMarkHistoryUserMessages_单轮不标注(t *testing.T) {
	msgs := []Message{userMsg("任务1")}
	MarkHistoryUserMessages(msgs, 0)
	if strings.Contains(msgs[0].Content, historyUserMarker) {
		t.Errorf("单条用户消息不应标注，得 %q", msgs[0].Content)
	}

	msgs = []Message{{Role: RoleSystem, Content: "sys"}, userMsg("任务1")}
	MarkHistoryUserMessages(msgs, 0)
	if strings.Contains(msgs[1].Content, historyUserMarker) {
		t.Errorf("system+单条用户消息不应标注，得 %q", msgs[1].Content)
	}
}

// TestMarkHistoryUserMessages_多轮标注 历史轮次用户消息加前缀、当前任务不加。
func TestMarkHistoryUserMessages_多轮标注(t *testing.T) {
	msgs := []Message{
		userMsg("任务1"),
		{Role: RoleAssistant, Content: "完成1"},
		userMsg("任务2"),
		{Role: RoleAssistant, Content: "完成2"},
		userMsg("任务3"), // 当前任务
	}
	MarkHistoryUserMessages(msgs, 0)

	if !strings.HasPrefix(msgs[0].Content, historyUserMarker) {
		t.Errorf("历史用户消息(任务1)应加前缀，得 %q", msgs[0].Content)
	}
	if !strings.HasPrefix(msgs[2].Content, historyUserMarker) {
		t.Errorf("历史用户消息(任务2)应加前缀，得 %q", msgs[2].Content)
	}
	if msgs[4].Content != "任务3" {
		t.Errorf("当前任务(最后一条用户消息)不应加前缀，得 %q", msgs[4].Content)
	}
	// 原内容保留在后缀
	if !strings.HasSuffix(msgs[0].Content, "任务1") {
		t.Errorf("标注后应保留原内容，得 %q", msgs[0].Content)
	}
}

// TestMarkHistoryUserMessages_幂等 已标注的消息再次调用不重复加前缀。
func TestMarkHistoryUserMessages_幂等(t *testing.T) {
	msgs := []Message{
		userMsg("任务1"),
		{Role: RoleAssistant, Content: "完成1"},
		userMsg("任务2"),
	}
	MarkHistoryUserMessages(msgs, 0)
	first := msgs[0].Content
	MarkHistoryUserMessages(msgs, 0) // 再次调用
	if msgs[0].Content != first {
		t.Errorf("幂等失败：二次标注内容变化 %q → %q", first, msgs[0].Content)
	}
	if strings.Count(msgs[0].Content, historyUserMarker) != 1 {
		t.Errorf("前缀应只出现一次，得 %q", msgs[0].Content)
	}
}

// TestMarkHistoryUserMessages_防重复末尾 user 末尾已是当前任务（防重复跳过追加）时，
// 该条不应被标注（它本身就是当前任务）。
func TestMarkHistoryUserMessages_防重复末尾(t *testing.T) {
	msgs := []Message{
		userMsg("任务1"),
		{Role: RoleAssistant, Content: "完成1"},
		userMsg("任务2"), // 与 task 相同，防重复场景下的当前任务
	}
	MarkHistoryUserMessages(msgs, 0)

	if !strings.HasPrefix(msgs[0].Content, historyUserMarker) {
		t.Errorf("历史用户消息应加前缀，得 %q", msgs[0].Content)
	}
	if msgs[2].Content != "任务2" {
		t.Errorf("末尾当前任务不应标注，得 %q", msgs[2].Content)
	}
}

// TestMarkHistoryUserMessages_摘要消息也标注 CondenseHistory 生成的回顾性
// user 摘要消息同样属于历史轮次，应被标注（与当前任务区分）。
func TestMarkHistoryUserMessages_摘要消息也标注(t *testing.T) {
	msgs := []Message{
		userMsg("【历史对话摘要】\n轮次1: 用户说...\n"),
		userMsg("任务2"), // 当前任务
	}
	MarkHistoryUserMessages(msgs, 0)
	if !strings.HasPrefix(msgs[0].Content, historyUserMarker) {
		t.Errorf("摘要消息应加历史前缀，得 %q", msgs[0].Content)
	}
	if msgs[1].Content != "任务2" {
		t.Errorf("当前任务不应标注，得 %q", msgs[1].Content)
	}
}

// TestMarkHistoryUserMessages_继承前缀豁免 delegate 子 Loop 继承父 msgs 时，
// 前 skipPrefix 条保持原样（KV Cache 前缀一致），其余历史 user 照常标注。
func TestMarkHistoryUserMessages_继承前缀豁免(t *testing.T) {
	// 模拟子 Loop：继承父 msgs（含父当前任务"需要规划"，未标注）+ 委托消息 + childTask
	msgs := []Message{
		{Role: RoleSystem, Content: "父system"},
		userMsg("需要规划"), // 父的当前任务（未标注，父视角）
		userMsg("【任务委派 → planner】\n给计划"),
		userMsg("给计划"), // 子当前任务
	}
	// 子 Loop 的 skipPrefix = len(继承的 history) = 3（system+需要规划+委托消息）
	MarkHistoryUserMessages(msgs, 3)

	if msgs[1].Content != "需要规划" {
		t.Errorf("继承前缀内的父当前任务不应被标注，得 %q", msgs[1].Content)
	}
	if msgs[2].Content != "【任务委派 → planner】\n给计划" {
		t.Errorf("继承前缀内的委托消息不应被标注，得 %q", msgs[2].Content)
	}
	if msgs[3].Content != "给计划" {
		t.Errorf("子当前任务不应标注，得 %q", msgs[3].Content)
	}
}

// TestLoopRun_自闭环两轮标注 模拟真实用户场景：同一对话线程内连续发起两轮任务
// （第二轮 history 自动使用第一轮的 l.History）。验证第二轮 LLM 视角：
// 第一轮的用户消息被标注为「历史轮次」，第二轮任务（当前）不带标注。
func TestLoopRun_自闭环两轮标注(t *testing.T) {
	// 记录型 provider：捕获每次 LLM 调用收到的消息
	type recordProvider struct {
		responses []Message
		calls     [][]Message
	}
	rec := &recordProvider{
		responses: []Message{{Content: "第一轮回复"}, {Content: "第二轮回复"}},
	}
	recChat := func(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
		cp := make([]Message, len(messages))
		copy(cp, messages)
		rec.calls = append(rec.calls, cp)
		var msg Message
		if len(rec.responses) > 0 {
			msg = rec.responses[0]
			rec.responses = rec.responses[1:]
		} else {
			msg = Message{Content: "完成"}
		}
		msg.Role = RoleAssistant
		if onChunk != nil {
			onChunk(Chunk{Content: msg.Content, Done: true})
		}
		return msg, nil
	}
	prov := &funcProvider{chat: recChat}

	reg := NewRegistry()
	loop := &Loop{Provider: prov, Registry: reg, System: "test", MaxIterations: 3}
	if _, err := loop.Run(context.Background(), "第一轮任务", nil); err != nil {
		t.Fatalf("第一轮 Run: %v", err)
	}
	if _, err := loop.Run(context.Background(), "第二轮任务", nil); err != nil {
		t.Fatalf("第二轮 Run: %v", err)
	}
	if len(rec.calls) != 2 {
		t.Fatalf("应 2 次 LLM 调用，得 %d", len(rec.calls))
	}

	// 第二轮 LLM 视角：第一轮任务应带【历史轮次】标注
	var firstMarked, secondUnmarked bool
	for _, m := range rec.calls[1] {
		if m.Role != RoleUser {
			continue
		}
		if strings.Contains(m.Content, "第一轮任务") {
			firstMarked = strings.HasPrefix(m.Content, historyUserMarker)
		}
		if strings.Contains(m.Content, "第二轮任务") {
			secondUnmarked = !strings.HasPrefix(m.Content, historyUserMarker)
		}
	}
	if !firstMarked {
		t.Error("第二轮 LLM 视角：第一轮任务应被标注为【历史轮次消息】")
	}
	if !secondUnmarked {
		t.Error("第二轮 LLM 视角：第二轮任务（当前）不应被标注")
	}
}

// funcProvider 函数式 Provider 适配器（测试用）。
type funcProvider struct {
	chat func(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error)
}

func (p *funcProvider) Name() string { return "func" }
func (p *funcProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	return p.chat(ctx, messages, tools, onChunk)
}

// TestLoopRun_多轮历史标注 端到端：loop.Run 返回的 msgs 中历史用户消息带标注、
// 当前 task 不带，且不影响 Loop 正常运行。
func TestLoopRun_多轮历史标注(t *testing.T) {
	reg := NewRegistry()
	mock := &MockProvider{Responses: []Message{
		{Content: "收到"},
	}}
	hist := []Message{
		userMsg("第一轮任务"),
		{Role: RoleAssistant, Content: "第一轮完成"},
		userMsg("第二轮任务"),
		{Role: RoleAssistant, Content: "第二轮完成"},
	}
	loop := &Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 3}
	msgs, err := loop.Run(context.Background(), "第三轮任务", hist)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	histMarked := 0
	for _, m := range msgs {
		if m.Role != RoleUser {
			continue
		}
		if strings.HasPrefix(m.Content, historyUserMarker) {
			histMarked++
		}
	}
	if histMarked != 2 {
		t.Errorf("历史 2 条用户消息都应标注，得 %d", histMarked)
	}
	// 当前 task（第三轮任务）应为未标注的 user 消息（含时间戳注入）
	foundTask := false
	for _, m := range msgs {
		if m.Role == RoleUser && strings.Contains(m.Content, "第三轮任务") && !strings.HasPrefix(m.Content, historyUserMarker) {
			foundTask = true
		}
	}
	if !foundTask {
		t.Error("当前任务（第三轮任务）应作为未标注的 user 消息存在")
	}
}
