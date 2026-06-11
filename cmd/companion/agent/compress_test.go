package agent

import (
	"context"
	"strconv"
	"strings"
	"testing"
)

// TestEstimateTokens 启发式估算：CJK ×1.5、ASCII ×0.25、每条 +4、工具参数/ID 计入。
func TestEstimateTokens(t *testing.T) {
	// 2 CJK 字 ×1.5 = 3 + 每条 4 = 7
	if got := estimateTokens([]Message{{Role: RoleUser, Content: "你好"}}); got != 7 {
		t.Errorf("CJK: 得 %d，期望 7", got)
	}
	// 4 ASCII ×0.25 = 1 + 4 = 5
	if got := estimateTokens([]Message{{Role: RoleUser, Content: "abcd"}}); got != 5 {
		t.Errorf("ASCII: 得 %d，期望 5", got)
	}
	// 工具调用：name "echo"(4×0.25=1) + args(10×0.25+8=10.5) + 4 = 15.5→16
	m := Message{Role: RoleAssistant, ToolCalls: []ToolCall{{Function: FunctionCall{Name: "echo", Arguments: "{\"x\":\"ab\"}"}}}}
	if got := estimateTokens([]Message{m}); got != 16 {
		t.Errorf("工具调用: 得 %d，期望 16", got)
	}
}

// makeConvo 造一段对话：system + user 任务 + n 组 [assistant(tool_call) + tool 结果]。
func makeConvo(n int) []Message {
	msgs := []Message{
		{Role: RoleSystem, Content: "你是助手，遵守铁律。"},
		{Role: RoleUser, Content: "请帮我重构这个项目的配置模块"},
	}
	for i := 0; i < n; i++ {
		id := "c" + strconv.Itoa(i)
		msgs = append(msgs,
			Message{Role: RoleAssistant, Content: "我来读第 " + strconv.Itoa(i) + " 个文件",
				ToolCalls: []ToolCall{{ID: id, Type: "function", Function: FunctionCall{Name: "read_file", Arguments: "{\"path\":\"f" + strconv.Itoa(i) + ".go\"}"}}}},
			Message{Role: RoleTool, ToolCallID: id, Name: "read_file", Content: "文件内容若干行……"},
		)
	}
	return msgs
}

// TestCompactStructure 压缩后：保系统前缀 + 一条摘要 + 最近段；总数变少；规则式摘要含目标。
func TestCompactStructure(t *testing.T) {
	l := &Loop{} // 无 Compressor → 规则式摘要
	msgs := makeConvo(20)
	out, dropped := l.compact(context.Background(), msgs)
	if dropped <= 0 {
		t.Fatal("应有中段被压缩")
	}
	if out[0].Role != RoleSystem {
		t.Error("系统前缀应保留在首位")
	}
	if out[1].Role != RoleUser || !strings.Contains(out[1].Content, "[上下文已压缩") {
		t.Errorf("第二条应为摘要消息，得 %q", out[1].Content)
	}
	if !strings.Contains(out[1].Content, "## 目标") || !strings.Contains(out[1].Content, "重构") {
		t.Errorf("规则摘要应含目标，得 %q", out[1].Content)
	}
	if len(out) >= len(msgs) {
		t.Errorf("压缩后应更短：%d → %d", len(msgs), len(out))
	}
	// 最近段保留在尾部（原最后一条仍是最后一条）
	if out[len(out)-1].Content != msgs[len(msgs)-1].Content {
		t.Error("最近段应原样保留在尾部")
	}
}

// TestCompactToolPairing 压缩切点落在 tool 结果上时，最近段不能以孤立 tool 开头（否则 OpenAI 报错）。
func TestCompactToolPairing(t *testing.T) {
	l := &Loop{}
	// 造让 keepFrom(=len-16) 恰好落在 tool 上：system+user + 偶数对后再补一条 assistant 使 len 为奇。
	msgs := makeConvo(20)
	msgs = append(msgs, Message{Role: RoleAssistant, Content: "继续"})
	out, dropped := l.compact(context.Background(), msgs)
	if dropped <= 0 {
		t.Fatal("应有中段被压缩")
	}
	// out[0]=system, out[1]=摘要(user)，out[2]=最近段首条——绝不能是孤立 tool 结果。
	if out[2].Role == RoleTool {
		t.Errorf("最近段首条不应为孤立 tool 结果（破坏配对）：%+v", out[2])
	}
	// 全量校验：最近段里每条 tool 之前必有带 tool_calls 的 assistant。
	tail := out[2:]
	for i, m := range tail {
		if m.Role != RoleTool {
			continue
		}
		paired := false
		for j := i - 1; j >= 0; j-- {
			if len(tail[j].ToolCalls) > 0 {
				paired = true
				break
			}
			if tail[j].Role == RoleUser || tail[j].Role == RoleSystem {
				break
			}
		}
		if !paired {
			t.Errorf("tail[%d] 是孤立 tool 结果，无配对 assistant", i)
		}
	}
}

// TestCompactLLMMode 有 Compressor → 用其摘要（mock 返回固定文本），摘要进压缩消息。
func TestCompactLLMMode(t *testing.T) {
	mock := &MockProvider{Responses: []Message{{Role: RoleAssistant, Content: "已读取 20 个配置文件并完成重构计划。"}}}
	l := &Loop{Compressor: mock}
	out, dropped := l.compact(context.Background(), makeConvo(20))
	if dropped <= 0 {
		t.Fatal("应有中段被压缩")
	}
	if !strings.Contains(out[1].Content, "LLM 摘要") || !strings.Contains(out[1].Content, "完成重构计划") {
		t.Errorf("应使用 LLM 摘要，得 %q", out[1].Content)
	}
	if mock.Calls() != 1 {
		t.Errorf("Compressor 应被调用 1 次，得 %d", mock.Calls())
	}
}

// TestCompactLLMFallback Compressor 返回过短（<10 字）→ 回退规则式摘要。
func TestCompactLLMFallback(t *testing.T) {
	mock := &MockProvider{Responses: []Message{{Role: RoleAssistant, Content: "短"}}}
	l := &Loop{Compressor: mock}
	out, _ := l.compact(context.Background(), makeConvo(20))
	if !strings.Contains(out[1].Content, "规则摘要") {
		t.Errorf("过短摘要应回退规则式，得 %q", out[1].Content)
	}
}

// TestMaybeCompactTrigger 阈值控制：超窗口才压缩；关闭(<=0)或未超阈值不动。
func TestMaybeCompactTrigger(t *testing.T) {
	msgs := makeConvo(20) // ~42 条

	// 关闭：MaxContextTokens<=0 → 原样
	l := &Loop{}
	if out := l.maybeCompact(context.Background(), msgs); len(out) != len(msgs) {
		t.Error("MaxContextTokens<=0 应不压缩")
	}

	// 未超阈值：窗口很大 → 原样，无事件
	var events int
	l = &Loop{MaxContextTokens: 10_000_000, OnEvent: func(e Event) {
		if e.Type == EventCompacted {
			events++
		}
	}}
	if out := l.maybeCompact(context.Background(), msgs); len(out) != len(msgs) {
		t.Error("远未超阈值应不压缩")
	}
	if events != 0 {
		t.Error("未压缩不应发 EventCompacted")
	}

	// 超阈值：窗口很小 → 压缩 + 发事件
	l = &Loop{MaxContextTokens: 100, OnEvent: func(e Event) {
		if e.Type == EventCompacted {
			events++
		}
	}}
	out := l.maybeCompact(context.Background(), msgs)
	if len(out) >= len(msgs) {
		t.Errorf("超阈值应压缩：%d → %d", len(msgs), len(out))
	}
	if events != 1 {
		t.Errorf("压缩应发 1 次 EventCompacted，得 %d", events)
	}
}

// TestLoopRunCompacts 端到端：低窗口 + 多轮工具循环 → loop 跑通且至少压缩一次。
func TestLoopRunCompacts(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{
		Name: "echo", Description: "echo", ReadOnly: true,
		Parameters: objSchema(props{"x": strProp("x")}, "x"),
		Handler:    func(_ context.Context, args map[string]any) (string, error) { return "echoed " + argStr(args, "x"), nil },
	})
	var responses []Message
	for i := 0; i < 18; i++ {
		responses = append(responses, Message{Role: RoleAssistant,
			ToolCalls: []ToolCall{{ID: "c" + strconv.Itoa(i), Type: "function",
				Function: FunctionCall{Name: "echo", Arguments: "{\"x\":\"padding content to grow the running context window steadily\"}"}}}})
	}
	// 第 19 轮脚本耗尽 → MockProvider 兜底返回 [FINAL]，loop 结束。
	var compacted, final int
	l := &Loop{
		Provider: &MockProvider{Responses: responses}, Registry: reg,
		MaxContextTokens: 120, MaxIterations: 40,
		OnEvent: func(e Event) {
			switch e.Type {
			case EventCompacted:
				compacted++
			case EventFinal:
				final++
			}
		},
	}
	if _, err := l.Run(context.Background(), "干活", nil); err != nil {
		t.Fatal(err)
	}
	if compacted == 0 {
		t.Error("多轮长上下文应至少压缩一次")
	}
	if final == 0 {
		t.Error("loop 应正常完成（EventFinal）")
	}
}
