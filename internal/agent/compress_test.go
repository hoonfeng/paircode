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

// padConvo 生成足够长的对话（保前缀精简的 keep 已扩到 40 条，短对话不再触发精简）。
// 结构：system + 多轮 (user → assistant(tool_call) → tool)。
func padConvo(rounds int) []Message {
	msgs := []Message{{Role: RoleSystem, Content: "你是助手。"}}
	task := "请帮我重构这个项目的配置模块"
	for i := 0; i < rounds; i++ {
		if i == 0 {
			msgs = append(msgs, Message{Role: RoleUser, Content: task})
		} else {
			msgs = append(msgs, Message{Role: RoleUser, Content: "继续第" + strconv.Itoa(i) + "步"})
		}
		id := "c" + strconv.Itoa(i)
		msgs = append(msgs,
			Message{Role: RoleAssistant, Content: "我来读第" + strconv.Itoa(i) + "个文件",
				ToolCalls: []ToolCall{{ID: id, Type: "function",
					Function: FunctionCall{Name: "read_file", Arguments: `{"path":"f` + strconv.Itoa(i) + `.go"}`}}}},
			Message{Role: RoleTool, ToolCallID: id, Name: "read_file", Content: strings.Repeat("内容", 40)},
		)
	}
	return msgs
}

// TestCompactPrefixStable 段内「只追加」的核心断言（2026-09 决策）：
//   - 段内每步视图都是历史的**同一份追加**（MaybeCompact 不再改消息），
//     相邻两次提交逐字节共享前缀 → 缓存 100% 命中；
//   - 不产生任何 EventCompacted（段内不精简）；
//   - 体积控制交给「分段边界 + 跨段精简」，因此历史可以放心长大。
func TestCompactPrefixStable(t *testing.T) {
	l := &Loop{MaxContextTokens: 600, OnEvent: func(e Event) {
		if e.Type == EventCompacted {
			t.Errorf("段内不应触发精简事件（只追加）")
		}
	}}
	hist := padConvo(10)
	var prev []Message
	for step := 0; step < 120; step++ {
		id := "s" + strconv.Itoa(step)
		hist = append(hist,
			Message{Role: RoleAssistant, Content: strings.Repeat("分析", 40),
				ToolCalls: []ToolCall{{ID: id, Type: "function",
					Function: FunctionCall{Name: "read_file", Arguments: `{"path":"x.go"}`}}}},
			Message{Role: RoleTool, ToolCallID: id, Name: "read_file", Content: strings.Repeat("结果", 120)},
		)
		out := l.maybeCompact(context.Background(), hist)
		if len(out) != len(hist) {
			t.Fatalf("第 %d 步：段内不应改动消息数（%d → %d）", step, len(hist), len(out))
		}
		if prev != nil {
			// 公共前缀必须覆盖上一视图全部内容（只有新增消息在后面）
			common := 0
			for common < len(prev) && common < len(out) && prev[common].Content == out[common].Content {
				common++
			}
			if common != len(prev) {
				t.Fatalf("第 %d 步：公共前缀只命中 %d/%d → 段内前缀断裂", step, common, len(prev))
			}
		}
		prev = out
	}
	if l.compactSlots != 0 || len(l.compactArchive) != 0 {
		t.Errorf("段内不应产生视图块/归档：slots=%d archive=%d", l.compactSlots, len(l.compactArchive))
	}
	t.Logf("120 步只追加：末次视图 %d 条（历史 %d 条），公共前缀 100%% 命中", len(prev), len(hist))
}

// TestFullHistoryStripsCompactBlocks 还原完整时间线时必须剔除精简视图块
// （占位块/摘要块不是真实历史，落盘与前端展示不能出现）。
func TestFullHistoryStripsCompactBlocks(t *testing.T) {
	l := &Loop{}
	msgs := padConvo(30)
	out, _, dropped := l.compact(context.Background(), msgs)
	if dropped <= 0 {
		t.Fatal("应触发精简")
	}
	full := l.fullHistory(out)
	// 完整时间线 = 去块视图 + 归档；长度必须回到原始（+ 无新增时）
	if len(full) != len(msgs) {
		t.Errorf("还原后长度应等于原始：%d != %d", len(full), len(msgs))
	}
	for i, m := range full {
		if isCompactViewBlock(m) {
			t.Errorf("还原结果不应含视图块（位置 %d）：%q", i, m.Content)
		}
		if i < len(msgs) && m.Content != msgs[i].Content {
			t.Errorf("还原顺序错位 @%d：%q != %q", i, truncRunesAgent(m.Content, 40), truncRunesAgent(msgs[i].Content, 40))
		}
	}
}

// TestCompactStructure 精简后：保系统前缀 + 占位块 + 摘要块 + 最近段；总数变少。
func TestCompactStructure(t *testing.T) {
	l := &Loop{} // 无 Compressor → 规则式摘要
	msgs := padConvo(30)
	out, summary, dropped := l.compact(context.Background(), msgs)
	if dropped <= 0 {
		t.Fatal("应有中段被精简")
	}
	if out[0].Role != RoleSystem {
		t.Error("系统前缀应保留在首位")
	}
	if l.compactSlots != 1 {
		t.Errorf("首次精简应追加 1 个占位块，得 %d", l.compactSlots)
	}
	// 摘要文本仍按原契约返回（并写入 CompressedSummaries / 摘要块）
	if !strings.Contains(summary, "## 目标") || !strings.Contains(summary, "重构") {
		t.Errorf("规则摘要应含目标，得 %q", summary)
	}
	if !strings.HasPrefix(summary, "[历史对话摘要") {
		t.Errorf("摘要应以中性标记开头，得 %q", summary)
	}
	if !strings.Contains(l.compactTrailing, "重构") {
		t.Errorf("摘要块应保留原始目标锚点，得 %q", truncRunesAgent(l.compactTrailing, 80))
	}
	if len(out) >= len(msgs) {
		t.Errorf("精简后应更短：%d → %d", len(msgs), len(out))
	}
	// 最近段保留在尾部（原最后一条仍是最后一条）
	if out[len(out)-1].Content != msgs[len(msgs)-1].Content {
		t.Error("最近段应原样保留在尾部")
	}
	// 最近段不得以孤立 tool 结果开头（配对保护）
	tailStart := 1 + l.compactSlots + 1
	if out[tailStart].Role == RoleTool {
		t.Errorf("最近段首条不应为孤立 tool 结果：%+v", out[tailStart])
	}
}

// TestCompactToolPairing 精简切点落在 tool 结果上时，最近段不能以孤立 tool 开头（否则 OpenAI 报错）。
func TestCompactToolPairing(t *testing.T) {
	l := &Loop{}
	// 造让 keepFrom(=len-keep) 恰好落在 tool 上的长度
	msgs := padConvo(30)
	out, _, dropped := l.compact(context.Background(), msgs)
	if dropped <= 0 {
		t.Fatal("应有中段被精简")
	}
	tailStart := 1 + l.compactSlots
	if l.compactTrailing != "" {
		tailStart++ // 摘要块
	}
	// 最近段首条绝不能是孤立 tool 结果
	if out[tailStart].Role == RoleTool {
		t.Errorf("最近段首条不应为孤立 tool 结果（破坏配对）：%+v", out[tailStart])
	}
	// 全量校验：最近段里每条 tool 之前必有带 tool_calls 的 assistant。
	tail := out[tailStart:]
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
	_, summary, dropped := l.compact(context.Background(), padConvo(30))
	if dropped <= 0 {
		t.Fatal("应有中段被精简")
	}
	if !strings.Contains(summary, "历史对话摘要") || !strings.Contains(summary, "完成重构计划") {
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
	_, summary, _ := l.compact(context.Background(), padConvo(30))
	if !strings.Contains(summary, "规则") {
		t.Errorf("过短摘要应回退规则式，得 %q", summary)
	}
}

// TestMaybeCompactTrigger 阈值控制：超窗口才精简；关闭(<=0)或未超阈值不动。
func TestMaybeCompactTrigger(t *testing.T) {
	msgs := padConvo(30) // 91 条

	// ★ 2026-09 决策：段内只追加——**任何窗口配置下**都不自动精简
	var events int
	newLoop := func(max int) *Loop {
		return &Loop{MaxContextTokens: max, OnEvent: func(e Event) {
			if e.Type == EventCompacted {
				events++
			}
		}}
	}
	for _, max := range []int{0, 100, 1000000} {
		l := newLoop(max)
		out := l.maybeCompact(context.Background(), msgs)
		if len(out) != len(msgs) {
			t.Errorf("窗口=%d：段内不应改动消息数（%d → %d）", max, len(msgs), len(out))
		}
		if l.compactSlots != 0 || len(l.compactArchive) != 0 || len(l.CompressedSummaries) != 0 {
			t.Errorf("窗口=%d：段内不应产生视图块/归档/摘要", max)
		}
	}
	if events != 0 {
		t.Errorf("段内不应发 EventCompacted，得 %d", events)
	}

	// 前端手动精简按钮：显式请求时仍生效（走 compact 保前缀视图）
	l := newLoop(100)
	l.CompactRequested = true
	out := l.maybeCompact(context.Background(), msgs)
	if len(out) >= len(msgs) {
		t.Errorf("手动精简应缩短上下文：%d → %d", len(msgs), len(out))
	}
	if events != 1 {
		t.Errorf("手动精简应发 1 次 EventCompacted，得 %d", events)
	}
	if len(l.CompressedSummaries) != 1 {
		t.Errorf("摘要应存入 CompressedSummaries，得 %d 条", len(l.CompressedSummaries))
	}
	if !strings.Contains(l.CompressedSummaries[0], "## 目标") {
		t.Errorf("规则摘要应含目标段，得 %q", l.CompressedSummaries[0])
	}
	if l.CompactRequested {
		t.Error("手动精简请求应被消费（清零）")
	}
}

// TestMaybeCompactHardFloor 硬地板为**可选保护**（★ 2026-09 默认关闭）：
//   - 默认（未设 PAIR_COMPACT_HARD_FLOOR）：硬地板恒为 0（关闭）
//   - 显式配置：HardFloorExceeded 在「窗口 > 地板 且 tokens ≥ 地板」时成立
//
// 说明：段内已不自动精简（只追加），硬地板只对**显式开启的旧阈值路径
// （maybeCompactLegacy）/ 未来需要时**有效；本用例直接验证判定函数与旧路径。
func TestMaybeCompactHardFloor(t *testing.T) {
	// 构造约 13 万 token 的对话
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
	tokens := estimateTokens(msgs)
	if tokens < 120000 {
		t.Fatalf("测试数据应超 12 万 token：%d", tokens)
	}

	// ① 默认：硬地板关闭 → 判定恒 false
	t.Setenv("PAIR_COMPACT_HARD_FLOOR", "")
	if CompactHardFloor() != 0 {
		t.Fatalf("默认硬地板应为 0（关闭），得 %d", CompactHardFloor())
	}
	if HardFloorExceeded(tokens, 1000000) {
		t.Error("硬地板关闭时 HardFloorExceeded 应为 false")
	}
	// 段内默认路径不动消息
	l := &Loop{MaxContextTokens: 1000000}
	if out := l.maybeCompact(context.Background(), msgs); len(out) != len(msgs) {
		t.Error("段内只追加：1M 窗口下 13 万 token 也不精简")
	}

	// ② 显式配置硬地板 120000 → 判定成立；旧阈值路径（maybeCompactLegacy）强制全量精简
	t.Setenv("PAIR_COMPACT_HARD_FLOOR", "120000")
	if CompactHardFloor() != 120000 {
		t.Fatalf("PAIR_COMPACT_HARD_FLOOR 应生效，得 %d", CompactHardFloor())
	}
	if !HardFloorExceeded(tokens, 1000000) {
		t.Error("窗口 > 地板且 tokens ≥ 地板时 HardFloorExceeded 应为 true")
	}
	l = &Loop{MaxContextTokens: 1000000}
	out := l.maybeCompactLegacy(context.Background(), msgs)
	if len(out) >= len(msgs) {
		t.Error("旧阈值路径下超硬地板应精简")
	}
	if len(l.CompressedSummaries) == 0 {
		t.Error("精简后应生成摘要数据")
	}
	if l.compactSlots != 1 {
		t.Errorf("应追加 1 个占位块，得 %d", l.compactSlots)
	}
	if len(l.compactArchive) == 0 {
		t.Error("被精简段应进入归档（落盘/展示线还原用）")
	}
}

// TestLoopRunNoAutoCompact 端到端：低窗口 + 多轮工具循环 → run 内不再自动压缩
// （2026-08-05：run 内压缩已关闭，早期工具输出是 LLM 后续引用的关键上下文，压缩会丢细节导致失忆）。
func TestLoopRunNoAutoCompact(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{
		Name: "echo", Description: "echo", ReadOnly: true,
		Parameters: objSchema(props{"x": strProp("x")}, "x"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return "echoed " + argStr(args, "x"), nil
		},
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
		MaxContextTokens: 120,
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
	longContent = strings.Repeat("甲", 6000) + "中间关键内容" + strings.Repeat("乙", 6000)  // 12005 rune
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

// TestBuildCallContextBackgroundBeforeTask 已被新语义取代：背景类 ephemeral 消息
// （历史摘要/执行日志/过期检查）不再插入「当前任务之前」的固定背景位——
// 背景快照已持久化到消息流（syncContextSnapshot，位置=当前任务之后，
// 对齐 dsh RuntimeContextProjection）。本测试删除，语义见
// TestBuildCallContextEphemeralMarkerAppended 与 TestSyncContextSnapshotIdempotent。

// TestBuildCallContextEphemeralMarkerAppended 验证：带背景标记的 ephemeral 消息
// 不再被收集进固定背景位（快照已持久化到消息流），全部按顺序追加末尾。
// 快照持久化后位置固定，ephemeral marker 消息仅兼容外部注入（追加末尾不影响前缀）。
func TestBuildCallContextEphemeralMarkerAppended(t *testing.T) {
	l := &Loop{}
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleAssistant, Content: "只有助手回复"},
	}
	l.ephemeralMsgs = []Message{
		{Role: RoleUser, Content: backgroundCtxMarker + "# 会话背景（外部注入测试）"},
	}
	out := l.buildCallContext(msgs)
	if len(out) != 3 {
		t.Fatalf("输出应含 3 条：%d", len(out))
	}
	if !strings.Contains(out[2].Content, "会话背景（外部注入测试）") {
		t.Errorf("marker ephemeral 应追加末尾（位置 2），实际 out[2]=%q", out[2].Content)
	}
}

// syncContextSnapshot 注入快照后 build 的前缀稳定验证见 TestBuildCallContextStableAcrossCalls
// （快照持久化语义：位置固定当前任务之后，零动态注入）。

// TestBuildSnapshotContent 验证 buildSnapshotContent：状态/记忆/知识库组装、
// 无内容时返回空（marker 与框架由 syncContextSnapshot 统一包裹）。
// ★ 2026-09：历史摘要段（原「# 上下文已压缩——历史摘要」）已删除——上下文
//   不再注入任何「压缩提示/摘要提示」文本，CompressedSummaries 只作运维数据保留。
func TestBuildSnapshotContent(t *testing.T) {
	l := &Loop{
		staleMsg:            "⚠️ 检测到 3 条可能过期的记忆/知识库条目",
		CompressedSummaries: []string{"[历史对话摘要] 轮次1 完成"},
		WorkspaceRoot:       "",
	}
	out := l.buildSnapshotContent()
	if !strings.Contains(out, "过期") {
		t.Error("应包含记忆/知识库过期检查")
	}
	// 摘要不再注入上下文（提示已删除）
	if strings.Contains(out, "历史摘要") || strings.Contains(out, "上下文已压缩") {
		t.Errorf("历史摘要提示段已删除，不应出现在快照正文中：%q", out)
	}
	if strings.Contains(out, "[历史对话摘要]") {
		t.Error("摘要正文不应注入快照（提示注入已删除）")
	}
	// 无摘要且非自主、无状态 → 不含摘要/自主/状态段（记忆/知识库为全局数据，
	// 存在于否不归 Loop 控制——只断言策略段不出现）
	empty := (&Loop{}).buildSnapshotContent()
	if strings.Contains(empty, "上下文已压缩") || strings.Contains(empty, "自主模式") || strings.Contains(empty, "过期") {
		t.Errorf("无摘要/非自主/无状态时不应含策略段，实际=%q", empty)
	}
}

// TestSyncContextSnapshotIdempotent 验证快照同步幂等语义（对齐 dsh
// RuntimeContextProjection）：内容相同 → 零注入；不同 → 追加新快照到
// 当前任务之后（随 tail 落盘），旧快照保留（append-only，位置固定）。
// ★ 2026-09：历史摘要不再进快照正文（提示注入已删除），故「内容变化」改用
// 状态提示（staleMsg）触发——摘要变化不再改变快照。
func TestSyncContextSnapshotIdempotent(t *testing.T) {
	l := &Loop{
		CompressedSummaries: []string{"[历史对话摘要] A"},
		staleMsg:            "⚠️ 检测到 1 条可能过期的记忆条目",
	}
	msgs := []Message{{Role: RoleUser, Content: "任务1"}}
	// 首次同步：注入快照（任务之后）
	msgs = l.syncContextSnapshot(msgs)
	if len(msgs) != 2 || msgs[1].Role != RoleUser || !strings.HasPrefix(msgs[1].Content, backgroundCtxMarker) {
		t.Fatalf("首次同步应注入快照到任务之后，实际 msgs=%+v", msgs)
	}
	// 内容相同 → 零注入
	again := l.syncContextSnapshot(append([]Message{}, msgs...))
	if len(again) != len(msgs) {
		t.Fatalf("内容相同应零注入，实际 len=%d -> %d", len(msgs), len(again))
	}
	// 摘要变化 → 不进快照正文 → 仍零注入（提示注入已删除）
	l.CompressedSummaries = append(l.CompressedSummaries, "[历史对话摘要] B")
	sameWithSummaries := l.syncContextSnapshot(append([]Message{}, msgs...))
	if len(sameWithSummaries) != len(msgs) {
		t.Fatalf("摘要不再进快照正文，摘要变化应零注入，实际 len=%d -> %d", len(msgs), len(sameWithSummaries))
	}
	// 内容变化（状态提示改变）→ 追加新快照（旧快照保留）
	l.staleMsg = "⚠️ 检测到 2 条可能过期的记忆条目"
	changed := l.syncContextSnapshot(append([]Message{}, msgs...))
	if len(changed) != 3 {
		t.Fatalf("内容变化应追加新快照（旧快照保留），实际 len=%d", len(changed))
	}
	if !strings.Contains(changed[2].Content, "2 条可能过期") {
		t.Error("新快照应含新的状态提示内容")
	}
}

// TestBuildCallContextStableAcrossCalls 验证：快照（过期检查/摘要）同步进消息流后，
// build 输出前缀稳定——快照位置固定（当前任务之后），跨迭代仅末尾单调增长，
// 这是 KV Cache 前缀命中的前提（快照不再每次迭代动态注入，位置不漂移）。
func TestBuildCallContextStableAcrossCalls(t *testing.T) {
	l := &Loop{
		staleMsg:            "⚠️ 检测到 3 条可能过期的记忆/知识库条目",
		CompressedSummaries: []string{"[历史对话摘要 — LLM] 轮次1 完成"},
	}
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "旧任务1"},
		{Role: RoleAssistant, Content: "旧回复"},
		{Role: RoleUser, Content: "当前任务"},
	}
	// ★ Run 开始：快照同步进消息流（任务之后）
	msgs = l.syncContextSnapshot(msgs)
	call1 := l.buildCallContext(msgs)
	// 模拟 iter1 后追加 assistant tool_call + tool 结果
	msgs = append(msgs,
		Message{Role: RoleAssistant, Content: "", ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}}}},
		Message{Role: RoleTool, ToolCallID: "c1", Content: "ok"},
	)
	call2 := l.buildCallContext(msgs)

	if len(call2) < len(call1) {
		t.Fatalf("iter2 消息数(%d)少于 iter1(%d)，前缀不完整", len(call2), len(call1))
	}
	for j := range call1 {
		if call2[j].Role != call1[j].Role || call2[j].Content != call1[j].Content {
			t.Errorf("iter1/iter2 前缀断裂！位置 %d:\n  iter1: role=%q content=%q\n  iter2: role=%q content=%q",
				j, call1[j].Role, truncStr(call1[j].Content, 40), call2[j].Role, truncStr(call2[j].Content, 40))
		}
	}
	// 快照（过期检查+摘要）应位于当前任务之后且在 iter1/iter2 相同位置；
	// 快照自带「非当前任务」声明（对齐 dsh runtime context snapshot：在任务
	// 之后，用声明让模型区分任务与背景，避免误把快照当最新输入）
	if !strings.Contains(call1[4].Content, "过期") {
		t.Errorf("快照应含过期检查且位于任务之后（位置 4），实际 call1[4]=%q", call1[4].Content)
	}
	if !strings.Contains(call2[4].Content, "过期") {
		t.Errorf("iter2 快照仍应含过期检查（位置固定），实际 call2[4]=%q", call2[4].Content)
	}
	// 快照必须是最后一条 user 且带非当前任务声明（快照在任务之后）
	last := call1[len(call1)-1]
	if last.Role != RoleUser || !strings.Contains(last.Content, "背景上下文") {
		t.Errorf("快照位于任务之后且带背景标记，实际 last=%+v", last)
	}
}
