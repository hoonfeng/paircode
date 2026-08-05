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

// TestCompactStructure 压缩后：保系统前缀 + 移除中段；总数变少；摘要字符串含目标。
func TestCompactStructure(t *testing.T) {
	l := &Loop{} // 无 Compressor → 规则式摘要
	msgs := makeConvo(20)
	out, summary, dropped := l.compact(context.Background(), msgs)
	if dropped <= 0 {
		t.Fatal("应有中段被压缩")
	}
	if out[0].Role != RoleSystem {
		t.Error("系统前缀应保留在首位")
	}
	// 不再插入摘要消息——摘要由返回的 summary 携带，注入系统提示可变部分
	if !strings.Contains(summary, "## 目标") || !strings.Contains(summary, "重构") {
		t.Errorf("规则摘要应含目标，得 %q", summary)
	}
	// 摘要前缀带标记（与 summarize() 一致）
	if !strings.HasPrefix(summary, "[上下文已压缩") {
		t.Errorf("摘要应以标记开头，得 %q", summary)
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
	out, _, dropped := l.compact(context.Background(), msgs)
	if dropped <= 0 {
		t.Fatal("应有中段被压缩")
	}
	// out[0]=system, out[1]=最近段首条——绝不能是孤立 tool 结果。
	if out[1].Role == RoleTool {
		t.Errorf("最近段首条不应为孤立 tool 结果（破坏配对）：%+v", out[1])
	}
	// 全量校验：最近段里每条 tool 之前必有带 tool_calls 的 assistant。
	tail := out[1:]
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

// TestCompactLLMMode 有 Compressor → 用其摘要（mock 返回固定文本），摘要通过返回值携带。
func TestCompactLLMMode(t *testing.T) {
	mock := &MockProvider{Responses: []Message{{Role: RoleAssistant, Content: "已读取 20 个配置文件并完成重构计划。"}}}
	l := &Loop{Compressor: mock}
	_, summary, dropped := l.compact(context.Background(), makeConvo(20))
	if dropped <= 0 {
		t.Fatal("应有中段被压缩")
	}
	if !strings.Contains(summary, "LLM 摘要") || !strings.Contains(summary, "完成重构计划") {
		t.Errorf("应使用 LLM 摘要，得 %q", summary)
	}
	if mock.Calls() != 1 {
		t.Errorf("Compressor 应被调用 1 次，得 %d", mock.Calls())
	}
}

// TestCompactLLMFallback Compressor 返回过短（<10 字）→ 回退规则式摘要。
func TestCompactLLMFallback(t *testing.T) {
	mock := &MockProvider{Responses: []Message{{Role: RoleAssistant, Content: "短"}}}
	l := &Loop{Compressor: mock}
	_, summary, _ := l.compact(context.Background(), makeConvo(20))
	if !strings.Contains(summary, "规则摘要") {
		t.Errorf("过短摘要应回退规则式，得 %q", summary)
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

	// 超阈值：窗口很小 → 压缩 + 发事件 + 摘要存入 CompressedSummaries
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
	if len(l.CompressedSummaries) != 1 {
		t.Errorf("摘要应存入 CompressedSummaries，得 %d 条", len(l.CompressedSummaries))
	}
	if !strings.Contains(l.CompressedSummaries[0], "## 目标") {
		t.Errorf("规则摘要应含目标段，得 %q", l.CompressedSummaries[0])
	}
}

// TestMaybeCompactHardFloor 超大窗口配置（100 万 token）下相对阈值形同虚设：
// token 绝对量达到 compactHardFloor 时强制全量压缩。
func TestMaybeCompactHardFloor(t *testing.T) {
	// 构造约 13 万 token 的对话（超过 compactHardFloor=120000）
	msgs := []Message{
		{Role: RoleSystem, Content: "你是助手。"},
		{Role: RoleUser, Content: "任务"},
	}
	for i := 0; i < 60; i++ {
		id := "c" + strconv.Itoa(i)
		msgs = append(msgs,
			Message{Role: RoleAssistant, Content: strings.Repeat("分析", 100),
				ToolCalls: []ToolCall{{ID: id, Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}}}},
			Message{Role: RoleTool, ToolCallID: id, Name: "read_file", Content: strings.Repeat("文件内容", 500)},
		)
	}
	// 估算应超过硬地板
	if got := estimateTokens(msgs); got < compactHardFloor {
		t.Fatalf("测试数据应超硬地板：%d < %d", got, compactHardFloor)
	}

	// 超大窗口：相对阈值永不触发，但硬地板应强制压缩
	l := &Loop{MaxContextTokens: 1000000}
	out := l.maybeCompact(context.Background(), msgs)
	if len(out) >= len(msgs) {
		t.Error("超大窗口 + 超硬地板应压缩")
	}
	if len(l.CompressedSummaries) == 0 {
		t.Error("压缩后应生成摘要")
	}
	if l.compactCooldown <= 0 {
		t.Error("压缩后应进入冷却")
	}
}

// TestLoopRunNoAutoCompact 端到端：低窗口 + 多轮工具循环 → run 内不再自动压缩
// （2026-08-05：run 内压缩已关闭，早期工具输出是 LLM 后续引用的关键上下文，压缩会丢细节导致失忆）。
func TestLoopRunNoAutoCompact(t *testing.T) {
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
	// 第 19 轮自然终止 — loop 退出。
	responses = append(responses, Message{Role: RoleAssistant,
		Content: "压缩测试完成"})
	var compacted, done int
	l := &Loop{
		Provider: &MockProvider{Responses: responses}, Registry: reg,
		MaxContextTokens: 120, MaxIterations: 40,
		OnEvent: func(e Event) {
			switch e.Type {
			case EventCompacted:
				compacted++
			case EventDone:
				done++
			}
		},
	}
	if _, err := l.Run(context.Background(), "干活", nil); err != nil {
		t.Fatal(err)
	}
	if compacted != 0 {
		t.Error("run 内不应自动压缩（run 内压缩已关闭，防中段上下文被摘要丢弃）")
	}
	if done == 0 {
		t.Error("loop 应正常完成（EventDone）")
	}
}

// TestTrimToolResult 工具结果瘦身：超长 RoleTool 内容只保留首尾，原始 msgs 不动。
func TestTrimToolResult(t *testing.T) {
	l := &Loop{}

	// 1. 短内容不截断
	short := Message{Role: RoleTool, ToolCallID: "c1", Name: "read_file", Content: "短结果"}
	if got := l.trimToolResult(short); got.Content != "短结果" {
		t.Errorf("短内容不应截断：%q", got.Content)
	}

	// 2. 超长内容截断：保留开头与结尾，中间省略标记
	longContent := strings.Repeat("甲", 4000) + "中间关键内容" + strings.Repeat("乙", 4000) // 8004 rune > 9000? 不够
	longContent = strings.Repeat("甲", 6000) + "中间关键内容" + strings.Repeat("乙", 6000)   // 12005 rune
	long := Message{Role: RoleTool, ToolCallID: "c2", Name: "run_command", Content: longContent}
	got := l.trimToolResult(long)
	if len([]rune(got.Content)) >= len([]rune(longContent)) {
		t.Errorf("超长内容应被截断：%d → %d", len([]rune(longContent)), len([]rune(got.Content)))
	}
	if !strings.Contains(got.Content, "已截断") {
		t.Error("截断应含提示标记")
	}
	if !strings.Contains(got.Content, "甲") || !strings.Contains(got.Content, "乙") {
		t.Error("截断应保留开头与结尾关键内容")
	}
	// 3. 非 tool 消息（user/assistant/system）不处理
	for _, m := range []Message{
		{Role: RoleUser, Content: longContent},
		{Role: RoleAssistant, Content: longContent},
		{Role: RoleSystem, Content: longContent},
	} {
		if got := l.trimToolResult(m); got.Content != longContent {
			t.Error("非 tool 消息不应被截断")
		}
	}
}

// TestBuildCallContextTrimTool 验证 buildCallContext 生成瘦身副本且不改原始 msgs
// （持久化历史与 UI 展示无损）。
func TestBuildCallContextTrimTool(t *testing.T) {
	l := &Loop{}
	longContent := strings.Repeat("数据", 5000) // 15000 rune
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "任务"},
		{Role: RoleTool, ToolCallID: "c1", Name: "read_file", Content: longContent},
	}
	l.ephemeralMsgs = []Message{{Role: RoleUser, Content: "【背景】早期摘要"}}

	out := l.buildCallContext(msgs)
	if len(out) != len(msgs)+1 {
		t.Fatalf("输出应含 ephemeral：%d", len(out))
	}
	// 原始 msgs 不被修改（持久化/UI 无损）
	if msgs[2].Content != longContent {
		t.Error("原始 msgs 的工具结果不应被修改")
	}
	// LLM 视图为瘦身副本
	if out[2].Content == longContent || len([]rune(out[2].Content)) >= len([]rune(longContent)) {
		t.Error("LLM 视图应使用瘦身副本")
	}
	// ephemeral 消息（摘要）保留且不截断
	if out[3].Content != "【背景】早期摘要" {
		t.Error("ephemeral 摘要消息应原样保留")
	}
	// 调用后 ephemeralMsgs 清空
	if len(l.ephemeralMsgs) != 0 {
		t.Error("buildCallContext 应清空 ephemeralMsgs")
	}
}
