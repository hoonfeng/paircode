package agent

import (
	"context"
	"strings"
	"testing"
)

// userMsg 构造一条用户消息。
func userMsg(content string) Message { return Message{Role: RoleUser, Content: content} }

// TestBuildHistory_排除末尾用户消息 末尾是用户消息（当前任务）时排除，
// 由 loop.Run 通过 task 参数重新添加。
func TestBuildHistory_排除末尾用户消息(t *testing.T) {
	msgs := []Message{
		userMsg("任务1"),
		{Role: RoleAssistant, Content: "完成1"},
		userMsg("任务2"), // 当前任务，应排除
	}
	hist := BuildHistory(msgs)
	if len(hist) != 2 {
		t.Fatalf("应排除末尾用户消息，得 %d 条", len(hist))
	}
	if hist[0].Content != "任务1" || hist[1].Role != RoleAssistant {
		t.Errorf("历史内容不符: %+v", hist)
	}
}

// TestBuildHistory_末尾非用户不排除 末尾不是用户消息（如中断恢复）时不排除，
// 防止丢失上轮回复。
func TestBuildHistory_末尾非用户不排除(t *testing.T) {
	msgs := []Message{
		userMsg("任务1"),
		{Role: RoleAssistant, Content: "完成1"},
	}
	hist := BuildHistory(msgs)
	if len(hist) != 2 {
		t.Fatalf("末尾非用户消息不应排除，得 %d 条", len(hist))
	}
}

// TestLoopRun_多轮历史原样注入 历史 user 消息进入 LLM 时内容原样
// （对齐 harness：不注入【历史轮次】前缀等任何内容污染），
// 当前 task 同样为原始用户输入（不含时间戳附加）。
func TestLoopRun_多轮历史原样注入(t *testing.T) {
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

	// 历史 user 消息必须内容原样（无前缀/无附加）
	for _, m := range msgs {
		if m.Role != RoleUser {
			continue
		}
		switch m.Content {
		case "第一轮任务", "第二轮任务":
			// 原样 ✓
		case "第三轮任务":
			// 当前任务原样 ✓
		default:
			t.Errorf("user 消息内容被污染: %q", m.Content)
		}
	}
}

// funcProvider 函数式 Provider 适配器（测试用）：记录每次 LLM 调用收到的消息。
type funcProvider struct {
	chat func(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error)
}

func (p *funcProvider) Name() string { return "func" }
func (p *funcProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	return p.chat(ctx, messages, tools, onChunk)
}

// TestLoopRun_自闭环两轮原样 模拟真实用户场景：同一对话线程内连续发起两轮任务。
// 验证第二轮 LLM 视角：第一轮任务与第二轮任务均为用户原始输入，
// 无任何注入前缀（对齐 harness 的消息结构区分方式）。
func TestLoopRun_自闭环两轮原样(t *testing.T) {
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

	// 第二轮 LLM 视角：两轮任务消息均为用户原始输入（无前缀、无附加内容）
	var firstRound, secondRound bool
	for _, m := range rec.calls[1] {
		if m.Role != RoleUser {
			continue
		}
		if m.Content == "第一轮任务" {
			firstRound = true
		}
		if m.Content == "第二轮任务" {
			secondRound = true
		}
	}
	if !firstRound {
		t.Errorf("第二轮 LLM 视角：第一轮任务应原样存在（标记=%v）", firstRound)
	}
	if !secondRound {
		t.Error("第二轮 LLM 视角：第二轮任务（当前）应原样存在")
	}
	// 全量检查：任何 user 消息都不应含历史标注前缀
	for _, m := range rec.calls[1] {
		if m.Role == RoleUser && strings.Contains(m.Content, "历史轮次消息") {
			t.Errorf("user 消息不应含历史轮次标注: %q", m.Content)
		}
	}
}

// TestLoopRun_当前任务无时间戳 当前任务消息 = 用户原始输入，
// 不附加「消息时间」等动态内容（对齐 harness：时间在事件元数据，不进 LLM 消息流）。
func TestLoopRun_当前任务无时间戳(t *testing.T) {
	reg := NewRegistry()
	mock := &MockProvider{Responses: []Message{{Content: "收到"}}}
	loop := &Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 3}
	msgs, err := loop.Run(context.Background(), "原始任务文本", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	foundTask := false
	for _, m := range msgs {
		if m.Role == RoleUser && m.Content == "原始任务文本" {
			foundTask = true
		}
		if m.Role == RoleUser && strings.Contains(m.Content, "消息时间") {
			t.Errorf("当前任务消息不应附加时间戳: %q", m.Content)
		}
	}
	if !foundTask {
		t.Error("当前任务应以原始文本存在")
	}
}
