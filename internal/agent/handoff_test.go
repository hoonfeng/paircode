package agent

// handoff_test.go — 会话交接（提交消息）单测：
//   - 触发阈值（token/条数）、刷新阈值（增量）、关闭开关；
//   - 视图组装（追加语义 / 跳过旧交接块 / 去 system / 越界保护）；
//   - LLM 整理（prompt 内容 / 围栏剥离）与规则式回退；
//   - BuildHandoffView 全流程（真实 MessageStore：生成 → 复用 → 刷新）。

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// handoffHist 构造测试历史：n 条 user/assistant 交替消息（CJK 文本）。
func handoffHist(n int, prefix string) []Message {
	out := make([]Message, 0, n)
	for i := 0; i < n; i++ {
		role := RoleUser
		if i%2 == 1 {
			role = RoleAssistant
		}
		out = append(out, Message{
			Role:    role,
			Content: fmt.Sprintf("%s 第%d条 内容内容内容内容内容内容内容内容内容内容内容内容内容内容内容内容内容内容内容内容", prefix, i),
		})
	}
	return out
}

// TestHandoff_ShouldHandoff 触发阈值：小对话不触发；token 达标 / 条数达标触发。
func TestHandoff_ShouldHandoff(t *testing.T) {
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_TRIGGER_TOKENS", "100")

	// 小对话（5 条 × ~90 tokens ≈ 470+，其中 100 阈值下……先验证"未达"场景：1 条）
	small := []Message{{Role: RoleUser, Content: "短"}}
	if ok, _ := ShouldHandoff(small, 0); ok {
		t.Fatal("1 条小消息不应触发交接")
	}

	// 20 条 × ~64 tokens ≈ 1280 > 100 → 触发
	big := handoffHist(20, "big")
	if ok, reason := ShouldHandoff(big, 0); !ok {
		t.Fatalf("token 达阈值应触发交接（%d 条）", len(big))
	} else if !strings.Contains(reason, "tokens") {
		t.Fatalf("原因应含 token 信息: %q", reason)
	}

	// 条数达标（token 阈值拉满 → 只能靠条数触发）
	t.Setenv("PAIR_HANDOFF_TRIGGER_TOKENS", "100000000")
	many := handoffHist(handoffTriggerMinMsgs+5, "many")
	if ok, reason := ShouldHandoff(many, 0); !ok {
		t.Fatalf("条数 ≥ %d 应触发交接", handoffTriggerMinMsgs)
	} else if !strings.Contains(reason, "条") {
		t.Fatalf("原因应含条数信息: %q", reason)
	}
}

// TestHandoff_ShouldHandoff_Disabled 关闭开关：PAIR_HANDOFF=0 → 永不触发。
func TestHandoff_ShouldHandoff_Disabled(t *testing.T) {
	t.Setenv("PAIR_HANDOFF", "0")
	t.Setenv("PAIR_HANDOFF_TRIGGER_TOKENS", "100")
	if ok, _ := ShouldHandoff(handoffHist(50, "x"), 0); ok {
		t.Fatal("PAIR_HANDOFF=0 时不应触发交接")
	}
}

// TestHandoff_ShouldRefresh 刷新判断：无记录必刷新；增量小复用；增量大刷新。
func TestHandoff_ShouldRefresh(t *testing.T) {
	t.Setenv("PAIR_HANDOFF_REFRESH_TOKENS", "300")

	hist := handoffHist(10, "h")

	// 无记录 → 必刷新
	if need, _ := ShouldRefreshHandoff(hist, nil); !need {
		t.Fatal("无交接记录应刷新")
	}
	// 空文本记录 → 必刷新
	if need, _ := ShouldRefreshHandoff(hist, &HandoffRecord{Text: "  "}); !need {
		t.Fatal("空文本记录应刷新")
	}
	// 记录锚 = 最后一条（增量=锚本身 1 条，极小）→ 不刷新
	rec := &HandoffRecord{Text: "【会话交接·提交消息】x", Anchor: handoffFingerprint(hist[len(hist)-1])}
	if need, _ := ShouldRefreshHandoff(hist, rec); need {
		t.Fatal("极小增量（仅锚条）不应刷新")
	}
	// 记录锚靠后、后续有小增量 → 不刷新
	rec2 := &HandoffRecord{Text: "text", Anchor: handoffFingerprint(hist[len(hist)-2])}
	if need, _ := ShouldRefreshHandoff(hist, rec2); need {
		t.Fatal("小增量不应刷新")
	}
	// 大增量 → 刷新（构造 30 条新消息）
	long := append(append([]Message{}, hist...), handoffHist(30, "new")...)
	if need, inc := ShouldRefreshHandoff(long, rec2); !need {
		t.Fatalf("大增量应刷新（inc=%d）", inc)
	}
	// 锚点丢失 → 保守兜底（最近 N 条）——此处兜底≈全量 10 条 ≈ 640 tokens > 300 阈值 → 刷新
	if need, inc := ShouldRefreshHandoff(hist, &HandoffRecord{Text: "t", Anchor: "no-such-anchor"}); !need || inc <= 0 {
		t.Fatalf("锚点丢失应保守刷新（need=%v inc=%d）", need, inc)
	}
}

// TestHandoff_Increment 增量定位：锚点匹配 / 锚点丢失兜底 / 空锚返回全量。
func TestHandoff_Increment(t *testing.T) {
	hist := handoffHist(30, "inc") // 30 条（含 user/assistant 交替）

	// 空锚 → 全量
	if inc := handoffIncrement(hist, nil); len(inc) != len(hist) {
		t.Fatalf("空记录应返回全量历史，实际 %d", len(inc))
	}
	// 锚 = 倒数第 5 条 → 增量 = 最后 5 条（含锚本身）
	anchorMsg := stripSystemMsgs(hist)[len(hist)-5]
	inc := handoffIncrement(hist, &HandoffRecord{Anchor: handoffFingerprint(anchorMsg)})
	if len(inc) != 5 {
		t.Fatalf("锚起增量应为 5 条（含锚），实际 %d", len(inc))
	}
	// 锚点丢失 → 兜底最近 handoffKeepRecentMsgs 条
	inc2 := handoffIncrement(hist, &HandoffRecord{Anchor: "no-such-anchor"})
	if len(inc2) != handoffKeepRecentMsgs {
		t.Fatalf("锚丢失兜底应为 %d 条，实际 %d", handoffKeepRecentMsgs, len(inc2))
	}
	// handoffAnchor：序列锚——以倒数第 handoffKeepRecentMsgs 条指纹开头、共 handoffAnchorSeq 段
	hs := stripSystemMsgs(hist)
	base := len(hs) - handoffKeepRecentMsgs
	wantSeq := strings.Join([]string{
		handoffFingerprint(hs[base]),
		handoffFingerprint(hs[base+1]),
		handoffFingerprint(hs[base+2]),
	}, handoffAnchorSep)
	if got := handoffAnchor(hist); got != wantSeq {
		t.Fatalf("handoffAnchor 序列锚不符: %q", got)
	}
}

// TestHandoff_ComposeView 视图组装：追加语义 / 跳过旧交接块 / 去 system。
func TestHandoff_ComposeView(t *testing.T) {
	hist := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "u1"},
		{Role: RoleAssistant, Content: "a1"},
		{Role: RoleUser, Content: "u2"},
	}
	text := backgroundCtxMarker + handoffTitle + "\n交接正文"

	// rec 锚 = u1 → 增量 = 非 system 中 u1 起的 [u1, a1, u2]（含锚）
	rec := &HandoffRecord{Text: text, Anchor: handoffFingerprint(Message{Role: RoleUser, Content: "u1"})}
	view := ComposeHandoffView(hist, rec, text)
	if len(view) != 4 {
		t.Fatalf("视图应为 [交接 + u1 + a1 + u2]，实际 %d 条", len(view))
	}
	if view[0].Content != text {
		t.Fatalf("视图首条应为交接文本: %q", view[0].Content)
	}
	if view[1].Content != "u1" || view[2].Content != "a1" || view[3].Content != "u2" {
		t.Fatalf("视图应保留增量 [u1, a1, u2]: %q", []string{view[1].Content, view[2].Content, view[3].Content})
	}
	for _, m := range view {
		if m.Role == RoleSystem {
			t.Fatal("视图不应含 system（由 Run 统一构建）")
		}
	}

	// 跳过旧交接块：增量里混入旧交接消息 → 被过滤
	hist2 := append(append([]Message{}, hist...), Message{Role: RoleAssistant, Content: text}, Message{Role: RoleUser, Content: "u3"})
	view2 := ComposeHandoffView(hist2, rec, text)
	for _, m := range view2[1:] {
		if isHandoffText(m.Content) {
			t.Fatal("视图不应包含旧交接块")
		}
	}

	// 新生成场景（锚 = handoffAnchor，保留最近消息）：
	// 本历史仅 3 条非 system（< handoffKeepRecentMsgs）→ 锚 = 第 0 条 → 保留 3 条（含锚）
	recNew := &HandoffRecord{Text: text, Anchor: handoffAnchor(hist)}
	if view3 := ComposeHandoffView(hist, recNew, text); len(view3) != 4 { // 交接 + u1 + a1 + u2
		t.Fatalf("新生成视图应为 [交接 + u1 + a1 + u2]，实际 %d", len(view3))
	}
}

// TestHandoff_Fallback 规则式回退：无 Provider 时输出含标题/前缀的兜底交接。
func TestHandoff_Fallback(t *testing.T) {
	hist := handoffHist(6, "f")
	text := BuildHandoffText(context.Background(), nil, nil, hist, "继续任务")
	if !strings.HasPrefix(text, backgroundCtxMarker) {
		t.Fatal("交接文本应以背景上下文标记开头（落盘锚点/轮次统计自动跳过）")
	}
	if !isHandoffText(text) {
		t.Fatal("交接文本应含标题标记")
	}
	if !strings.Contains(text, "规则式") {
		t.Fatalf("回退文本应标注规则式: %q", text[:80])
	}
}

// TestHandoff_LLMText LLM 整理：prompt 含任务与历史；输出围栏/标题复述被清洗。
func TestHandoff_LLMText(t *testing.T) {
	var gotPrompt string
	prov := &funcProvider{chat: func(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
		if len(messages) > 0 {
			gotPrompt = messages[0].Content
		}
		return Message{Role: RoleAssistant, Content: "```\n【会话交接·提交消息】\n## 与当前任务的相关性\n高：同一任务延续\n## 任务目标\n完成会话交接机制（handoff 标记）\n## 已完成\n实现 handoff.go 与单测\n```"}, nil
	}}
	hist := handoffHist(8, "llm")
	text := BuildHandoffText(context.Background(), prov, nil, hist, "实现会话交接功能")

	if !strings.Contains(gotPrompt, "即将继续的任务") || !strings.Contains(gotPrompt, "实现会话交接功能") {
		t.Fatalf("prompt 应包含当前任务: %q…", gotPrompt[:min(len(gotPrompt), 200)])
	}
	if !strings.Contains(gotPrompt, "对话历史节选") {
		t.Fatal("prompt 应包含历史节选标题")
	}
	if strings.Contains(text, "```") {
		t.Fatalf("输出不应残留代码围栏: %q", text)
	}
	if strings.Contains(text, "规则式") {
		t.Fatalf("LLM 输出有效时不应走规则式回退: %q", text)
	}
	// 标题只出现一次（模型的标题复述被剥掉；前缀中的标题为唯一一处）
	if strings.Count(text, handoffTitle) != 1 {
		t.Fatalf("标题应恰好出现一次: %q", text)
	}
	if !strings.Contains(text, "## 与当前任务的相关性") {
		t.Fatalf("正文应保留相关性小节: %q", text)
	}
}

// TestHandoff_LoopViewProvider 续跑接线（buildLoopHandoffView）：
// Provider 优先压缩模型；视图由 loop.History（含 system）组装且不含 system。
func TestHandoff_LoopViewProvider(t *testing.T) {
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_TRIGGER_TOKENS", "50")
	t.Setenv("PAIR_HANDOFF_REFRESH_TOKENS", "100000")

	root := t.TempDir()
	store := NewMessageStore(root)

	compressorCalls, providerCalls := 0, 0
	compressor := &funcProvider{chat: func(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
		compressorCalls++
		return Message{Role: RoleAssistant, Content: "## 与当前任务的相关性\n高：同一任务延续（压缩模型输出）\n## 已完成\n续跑接线验证与状态记录，详见会话存储"}, nil
	}}
	provider := &funcProvider{chat: func(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
		providerCalls++
		return Message{Role: RoleAssistant, Content: "provider 输出（不应被使用）"}, nil
	}}

	hist := append([]Message{{Role: RoleSystem, Content: "sys"}}, handoffHist(20, "loop")...)
	loop := &Loop{Provider: provider, Compressor: compressor, History: hist}
	view, ok := buildLoopHandoffView(context.Background(), loop, store, "conv_loop_handoff", "继续")
	if !ok {
		t.Fatal("续跑接线应启用交接视图")
	}
	if compressorCalls != 1 || providerCalls != 0 {
		t.Fatalf("应优先压缩模型（compressor=%d provider=%d）", compressorCalls, providerCalls)
	}
	if !isHandoffText(view[0].Content) || !strings.Contains(view[0].Content, "压缩模型输出") {
		t.Fatalf("视图首条应为压缩模型生成的交接: %q", view[0].Content[:80])
	}
	for _, m := range view {
		if m.Role == RoleSystem {
			t.Fatal("视图不应含 system（含 system 的历史应被剥离）")
		}
	}

	// 无 Compressor → 回退主 Provider
	loop2 := &Loop{Provider: provider, History: handoffHist(20, "loop2")}
	if _, ok2 := buildLoopHandoffView(context.Background(), loop2, store, "conv_loop_handoff2", "继续"); !ok2 {
		t.Fatal("回退主 Provider 也应启用交接")
	}
	if providerCalls != 1 {
		t.Fatalf("无 Compressor 时应回退主 Provider（provider=%d）", providerCalls)
	}
}

// TestHandoff_SequenceAnchorDedup 序列锚：历史含重复消息（同角色同长度同前缀
// → 同指纹）时锚点定位不漂移——单指纹「从末尾向前找」会锚到后出现的重复项，
// 导致视图漏段/前缀漂移。
func TestHandoff_SequenceAnchorDedup(t *testing.T) {
	hist := handoffHist(40, "dup") // 40 条 → 基点 idx = 40-16 = 24
	dup := Message{Role: RoleAssistant, Content: "重复的工具输出：" + strings.Repeat("x", 40)}
	hist[8] = dup
	hist[24] = dup // 基点本身就是重复内容（第 8 条与其同指纹）

	rec := &HandoffRecord{Text: "【会话交接·提交消息】\n去重", Anchor: handoffAnchor(hist), KeptTokens: 1}

	// 追加与基点同内容的重复消息（尾部）+ 新任务
	h2 := append(append([]Message{}, hist...), dup, Message{Role: RoleUser, Content: "新任务"})

	idx := handoffAnchorIndex(h2, rec)
	if idx != 24 {
		t.Fatalf("序列锚应稳定定位到 24，实得 %d（漂移=视图漏段）", idx)
	}
	view := ComposeHandoffView(h2, rec, rec.Text)
	want := 1 + len(h2) - 24 // 交接 + h2[24:] 完整增量（含锚）
	if len(view) != want {
		t.Fatalf("视图应含完整增量（%d 条），实得 %d", want, len(view))
	}
	// 兼容路径（★ 2026-09-12 语义统一）：单指纹（存量旧记录）同样按
	// 「第一个匹配」稳定定位——原实现从末尾向前找，会锚到新增的重复项
	// （每次追加都漂移 → 保留段重排 → 跨轮前缀缓存全断）。
	legacy := &HandoffRecord{Text: "t", Anchor: handoffFingerprint(dup), KeptTokens: 1}
	if li := handoffAnchorIndex(h2, legacy); li != 8 {
		t.Fatalf("单指纹（兼容路径）应稳定定位到首个匹配 8，实得 %d", li)
	}
	h3 := append(append([]Message{}, h2...), dup)
	if li := handoffAnchorIndex(h3, legacy); li != 8 {
		t.Fatalf("单指纹定位应跨轮稳定（首个匹配 8），实得 %d", li)
	}
}

// TestHandoff_AnchorToolBoundary 边界对齐：视图首条不能是孤立 tool 结果——
// 其配对 assistant tool_call 若被折叠，sanitizeToolPairing 会丢弃该条（内容丢失）。
// 基点应向前移动，直到视图首条非 tool（宁可多保留，不制造孤儿）。
func TestHandoff_AnchorToolBoundary(t *testing.T) {
	hist := handoffHist(40, "tb") // 基点候选 idx=24
	// 令基点本身是 tool（其配对 assistant tool_call 在 23；25 为并行的第二条 tool）
	cid := "call_tb_1"
	hist[23] = Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: cid, Type: "function", Function: FunctionCall{Name: "read", Arguments: "{}"}}}}
	hist[24] = Message{Role: RoleTool, ToolCallID: cid, Name: "read", Content: "工具结果"}
	hist[25] = Message{Role: RoleTool, ToolCallID: "call_tb_2", Name: "read", Content: "工具结果2"}

	rec := &HandoffRecord{Text: "【会话交接·提交消息】\n边界", Anchor: handoffAnchor(hist), KeptTokens: 1}
	idx := handoffAnchorIndex(hist, rec)
	if idx != 23 {
		t.Fatalf("基点应前移到 23（hist[23]=assistant 非 tool），实得 %d", idx)
	}
	view := ComposeHandoffView(hist, rec, rec.Text)
	if len(view) < 3 {
		t.Fatalf("视图过短：%d", len(view))
	}
	if view[1].Role != RoleAssistant || len(view[1].ToolCalls) == 0 {
		t.Fatalf("视图首条应为带 tool_calls 的 assistant（配对完整），实得 role=%v", view[1].Role)
	}
	if view[2].Role != RoleTool {
		t.Fatalf("tool 结果应紧随配对调用之后，实得 role=%v", view[2].Role)
	}
}

// TestHandoff_ViewPrefixStable 前缀稳定（KV 缓存命中的前提）：同一交接周期内
// 历史单调追加时，新视图必须是「旧视图 + 尾部新增」——前缀逐字节一致。
// 用含重复消息的历史验证序列锚下不自漂移（单指纹实现每轮会锚到不同位置 →
// 前缀断裂 + 漏段，本测试会失败）。
func TestHandoff_ViewPrefixStable(t *testing.T) {
	hist := handoffHist(40, "kv")
	dup := Message{Role: RoleAssistant, Content: "重复输出 " + strings.Repeat("y", 30)}
	hist[5] = dup
	hist[24] = dup // 基点处放重复内容（最险场景）

	rec := &HandoffRecord{Text: "【会话交接·提交消息】\n稳定", Anchor: handoffAnchor(hist), KeptTokens: 1}
	v1 := ComposeHandoffView(hist, rec, rec.Text)

	h2 := append(append([]Message{}, hist...), dup, Message{Role: RoleUser, Content: "继续"})
	v2 := ComposeHandoffView(h2, rec, rec.Text)

	h3 := append(append([]Message{}, h2...), Message{Role: RoleAssistant, Content: "好的"}, Message{Role: RoleUser, Content: "再来"})
	v3 := ComposeHandoffView(h3, rec, rec.Text)

	assertPrefix := func(name string, prev, cur []Message) {
		if len(cur) < len(prev) {
			t.Fatalf("%s: 视图长度回退（%d < %d）", name, len(cur), len(prev))
		}
		for i := range prev {
			if cur[i].Role != prev[i].Role || cur[i].Content != prev[i].Content || cur[i].ToolCallID != prev[i].ToolCallID {
				t.Fatalf("%s: 前缀第 %d 条被改写（KV 缓存断裂）\nprev=%v|%q\ncur =%v|%q",
					name, i, prev[i].Role, truncRunesAgent(prev[i].Content, 60), cur[i].Role, truncRunesAgent(cur[i].Content, 60))
			}
		}
	}
	assertPrefix("v2", v1, v2)
	assertPrefix("v3", v2, v3)
	// 增量只追加：v2 尾部 = [重复项, 继续]；v3 尾部 = [好的, 再来]
	if got := v2[len(v1):]; len(got) != 2 || got[0].Content != dup.Content || got[1].Content != "继续" {
		t.Fatalf("v2 尾部应为新增两条，实得 %d 条", len(got))
	}
	if got := v3[len(v2):]; len(got) != 2 || got[0].Content != "好的" || got[1].Content != "再来" {
		t.Fatalf("v3 尾部应为新增两条，实得 %d 条", len(got))
	}
}

// TestHandoff_LoopHistoryReuse 续跑链路复用：视图（[交接]+[锚起保留段]）经 Run 后
// 成为 loop.History 的一部分——段边界再次判断时锚点必须仍可匹配（复用成立，
// 不反复重建交接）。若视图不含锚（旧设计），History 里锚消息缺失 → 锚丢失 →
// 每段强制重建交接 → 每段一次前缀断裂（KV 缓存无法跨段命中）。
func TestHandoff_LoopHistoryReuse(t *testing.T) {
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_REFRESH_TOKENS", "100000")

	hist := handoffHist(40, "ru")
	rec := &HandoffRecord{Text: backgroundCtxMarker + handoffTitle + "\n复用", Anchor: handoffAnchor(hist), KeptTokens: 1}
	v1 := ComposeHandoffView(hist, rec, rec.Text)

	// 模拟 Run：History = [system] + 视图 + 段内新增（续跑消息、交互）
	inRun := append([]Message{{Role: RoleSystem, Content: "sys"}}, v1...)
	inRun = append(inRun,
		Message{Role: RoleUser, Content: "（工具预算续跑）继续"},
		Message{Role: RoleAssistant, Content: "继续处理"},
	)

	// 段边界复用判断：锚点应可匹配（否则强制刷新=每段重建）
	if need, inc := ShouldRefreshHandoff(inRun, rec); need {
		t.Fatalf("视图含锚后，续跑 History 应能匹配锚点复用（不应强制刷新，inc=%d）", inc)
	}
	v2 := ComposeHandoffView(inRun, rec, rec.Text)
	// v2 = [交接] + inRun 锚起增量 = v1 的全部 + 段内新增（前缀逐条一致）
	if len(v2) != len(v1)+2 {
		t.Fatalf("续跑复用视图应为 v1 + 段内新增 2 条（合计 %d），实得 %d", len(v1)+2, len(v2))
	}
	for i := range v1 {
		if v2[i].Role != v1[i].Role || v2[i].Content != v1[i].Content {
			t.Fatalf("续跑复用视图前缀应与首轮视图一致（第 %d 条被改写 → KV 断裂）", i)
		}
	}
	if v2[len(v2)-1].Content != "继续处理" {
		t.Fatalf("续跑复用视图尾部应为段内新增")
	}
}

// TestHandoff_ParseRelevance 相关性解析：取最早出现的 高/部分/无关（中文优先，英文兜底）。
func TestHandoff_ParseRelevance(t *testing.T) {
	cases := []struct{ in, want string }{
		{"高", handoffRelHigh},
		{"部分", handoffRelPartial},
		{"无关", handoffRelNone},
		{"## 与当前任务的相关性\n高：同一任务延续", handoffRelHigh},
		{"## 与当前任务的相关性\n无关（全新任务）", handoffRelNone},
		{"高：同一任务，部分功能复用", handoffRelHigh}, // 最早出现者优先
		{"partial", handoffRelPartial},    // 英文兜底
		{"None.", handoffRelNone},
		{"随便写点什么没有关键词", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := parseRelevance(tc.in); got != tc.want {
			t.Errorf("parseRelevance(%q) = %q，期望 %q", tc.in, got, tc.want)
		}
	}
}

// TestHandoff_KeepDepthByRel 相关性驱动保留段深度：高=16 / 部分=8 / 无关=4。
func TestHandoff_KeepDepthByRel(t *testing.T) {
	hist := handoffHist(30, "kd") // 全非 tool → 边界对齐不动
	for _, tc := range []struct {
		rel  string
		keep int
	}{
		{handoffRelHigh, handoffKeepRelHigh},
		{handoffRelPartial, handoffKeepRelPartial},
		{handoffRelNone, handoffKeepRelNone},
		{"", handoffKeepRelHigh}, // 空=未知 → 保守按高
	} {
		rec := &HandoffRecord{
			Text:      backgroundCtxMarker + handoffTitle + "\n" + tc.rel,
			Anchor:    handoffAnchorAt(hist, tc.keep),
			Relevance: tc.rel,
		}
		if idx := handoffAnchorIndex(stripSystemMsgs(hist), rec); idx != len(hist)-tc.keep {
			t.Fatalf("rel=%q 锚点应为倒数第 %d 条（idx=%d），实得 %d", tc.rel, tc.keep, len(hist)-tc.keep, idx)
		}
		view := ComposeHandoffView(hist, rec, rec.Text)
		if len(view) != 1+tc.keep {
			t.Fatalf("rel=%q 视图应为 交接 + %d 条，实得 %d", tc.rel, tc.keep, len(view))
		}
	}
}

// TestHandoff_JudgeFlow B/C 全流程（真实 MessageStore）：
//
//	① 生成（判官不参与）→ rel 从文本解析；② 增量 < 复检周期 → 判官不调用；
//	③ 增量 ≥ 周期且判官判定不变 → 仅推进复检基线（不重生成）；
//	④ 判官判定变化 → 强制重生成 + 保留段深度随新相关性（无关=4 条）；
//	⑤ 判官失败 → 零副作用（不重生成）+ 复检基线仍推进（防每轮重试风暴）。
func TestHandoff_JudgeFlow(t *testing.T) {
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_TRIGGER_TOKENS", "100")
	t.Setenv("PAIR_HANDOFF_REFRESH_TOKENS", "100000000") // 刷新阈值拉满 → 只有判官能触发重生成
	t.Setenv("PAIR_HANDOFF_RECHECK_TOKENS", "200")

	root := t.TempDir()
	store := NewMessageStore(root)
	conv := "conv_judge"

	llmCalls := 0
	prov := &funcProvider{chat: func(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
		llmCalls++
		return Message{Role: RoleAssistant, Content: "## 与当前任务的相关性\n高：同一任务延续，继续验证判官流程与保留段深度"}, nil
	}}
	judgeCalls := 0
	judgeRel := handoffRelHigh
	judge := &funcProvider{chat: func(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
		judgeCalls++
		return Message{Role: RoleAssistant, Content: judgeRel}, nil
	}}

	hist := handoffHist(20, "judge") // ≈1280 tokens > 100 → 触发

	// ① 首次生成：判官不参与（生成路径不做复检）
	view, ok := BuildHandoffView(context.Background(), prov, judge, store, conv, hist, "继续", 0)
	if !ok || len(view) == 0 {
		t.Fatal("首次应启用交接")
	}
	if judgeCalls != 0 || llmCalls != 1 {
		t.Fatalf("生成路径不应调用判官（judge=%d llm=%d）", judgeCalls, llmCalls)
	}
	rec, _ := store.LoadHandoff(conv)
	if rec == nil || rec.Relevance != handoffRelHigh {
		t.Fatalf("相关性应从文本解析为「高」: %+v", rec)
	}

	// ② 增量（1 条 ≈64 tokens）< 复检周期 200 → 判官不调用
	hist2 := append(append([]Message{}, hist...), handoffHist(1, "small")...)
	if _, ok := BuildHandoffView(context.Background(), prov, judge, store, conv, hist2, "继续", 0); !ok {
		t.Fatal("② 应启用交接")
	}
	if judgeCalls != 0 {
		t.Fatalf("增量未达周期不应调用判官（judge=%d）", judgeCalls)
	}

	// ③ 增量 ≥ 周期 + 判官判定不变 → 仅推进复检基线
	hist3 := append(append([]Message{}, hist...), handoffHist(8, "mid")...)
	llmBefore := llmCalls
	if _, ok := BuildHandoffView(context.Background(), prov, judge, store, conv, hist3, "继续", 0); !ok {
		t.Fatal("③ 应启用交接")
	}
	if judgeCalls != 1 {
		t.Fatalf("增量达周期应复检一次（judge=%d）", judgeCalls)
	}
	if llmCalls != llmBefore {
		t.Fatalf("判定不变不应重生成（llm=%d→%d）", llmBefore, llmCalls)
	}
	if rec3, _ := store.LoadHandoff(conv); rec3 == nil || rec3.RelCheckedInc <= 0 {
		t.Fatalf("复检基线应推进: %+v", rec3)
	}

	// ④ 判官判定变化（高→无关）→ 强制重生成 + 保留段深度=4
	judgeRel = handoffRelNone
	hist4 := append(append([]Message{}, hist3...), handoffHist(8, "more")...)
	llmBefore = llmCalls
	view4, ok := BuildHandoffView(context.Background(), prov, judge, store, conv, hist4, "全新任务", 0)
	if !ok {
		t.Fatal("④ 应启用交接")
	}
	if judgeCalls != 2 {
		t.Fatalf("④ 应第二次复检（judge=%d）", judgeCalls)
	}
	if llmCalls != llmBefore+1 {
		t.Fatalf("判定变化应强制重生成（llm=%d→%d）", llmBefore, llmCalls)
	}
	rec4, _ := store.LoadHandoff(conv)
	if rec4 == nil || rec4.Relevance != handoffRelNone {
		t.Fatalf("相关性应更新为「无关」: %+v", rec4)
	}
	hs4 := stripSystemMsgs(hist4)
	wantAnchor := strings.Join([]string{
		handoffFingerprint(hs4[len(hs4)-4]),
		handoffFingerprint(hs4[len(hs4)-3]),
		handoffFingerprint(hs4[len(hs4)-2]),
	}, handoffAnchorSep)
	if rec4.Anchor != wantAnchor {
		t.Fatalf("无关场景锚点应为最近 %d 条（含边界对齐）", handoffKeepRelNone)
	}
	if len(view4) != 1+handoffKeepRelNone {
		t.Fatalf("无关场景视图应为 交接 + %d 条，实得 %d", handoffKeepRelNone, len(view4))
	}
	if rec4.RelCheckedInc != 0 {
		t.Fatalf("重生成后复检基线应归零: %d", rec4.RelCheckedInc)
	}

	// ⑤ 判官失败 → 零副作用 + 复检基线推进
	failingJudge := &funcProvider{chat: func(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
		return Message{}, fmt.Errorf("judge boom")
	}}
	hist5 := append(append([]Message{}, hist4...), handoffHist(8, "more2")...)
	llmBefore = llmCalls
	if _, ok := BuildHandoffView(context.Background(), prov, failingJudge, store, conv, hist5, "继续", 0); !ok {
		t.Fatal("⑤ 应启用交接")
	}
	if llmCalls != llmBefore {
		t.Fatalf("判官失败不应重生成（llm=%d→%d）", llmBefore, llmCalls)
	}
	if rec5, _ := store.LoadHandoff(conv); rec5 == nil || rec5.Relevance != handoffRelNone {
		t.Fatalf("判官失败应保持原相关性: %+v", rec5)
	} else if rec5.RelCheckedInc <= 0 {
		t.Fatalf("判官失败也应推进复检基线（防每轮重试风暴）: %+v", rec5)
	}
}

// TestHandoff_ViewFlow 全流程（真实 MessageStore）：生成 → 复用 → 刷新。
func TestHandoff_ViewFlow(t *testing.T) {
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_TRIGGER_TOKENS", "100")
	t.Setenv("PAIR_HANDOFF_REFRESH_TOKENS", "300")

	root := t.TempDir()
	store := NewMessageStore(root)
	conv := "conv_handoff_test"

	calls := 0
	prov := &funcProvider{chat: func(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
		calls++
		return Message{Role: RoleAssistant, Content: "## 与当前任务的相关性\n高：同一任务延续（LLM 生成）\n## 已完成\n第一批工作与状态记录详见会话存储"}, nil
	}}

	hist := handoffHist(20, "flow") // ≈1280 tokens > 100 → 触发

	// ① 首次：生成交接
	view, ok := BuildHandoffView(context.Background(), prov, nil, store, conv, hist, "继续", 0)
	if !ok {
		t.Fatal("应启用交接视图")
	}
	if calls != 1 {
		t.Fatalf("首次应调用 LLM 一次，实际 %d", calls)
	}
	// 视图 = 交接 + 最近 handoffKeepRecentMsgs 条原文（20 条历史：锚=第 4 条 → 保留 16 条，含锚）
	wantKeep := handoffKeepRecentMsgs
	if len(view) != 1+wantKeep || !isHandoffText(view[0].Content) {
		t.Fatalf("首次视图应为「交接 + 最近 %d 条」，实际 %d 条", wantKeep, len(view))
	}
	if !strings.Contains(view[0].Content, "LLM 生成") {
		t.Fatalf("首次交接应来自 LLM 输出: %q", view[0].Content)
	}
	// 最近消息原文保留（不丢最近状态）
	if view[len(view)-1].Content != hist[len(hist)-1].Content {
		t.Fatalf("视图末条应为历史最近消息原文")
	}
	rec, lerr := store.LoadHandoff(conv)
	if lerr != nil || rec == nil {
		t.Fatalf("交接记录应已持久化: %v", lerr)
	}
	if rec.Anchor == "" || rec.MsgCount != len(hist) {
		t.Fatalf("记录应有锚点与条数（anchor=%q count=%d）", rec.Anchor, rec.MsgCount)
	}
	// 锚点应为「倒数第 handoffKeepRecentMsgs 条起连续 handoffAnchorSeq 条」的序列锚
	hs := stripSystemMsgs(hist)
	base := len(hs) - handoffKeepRecentMsgs
	wantAnchor := strings.Join([]string{
		handoffFingerprint(hs[base]),
		handoffFingerprint(hs[base+1]),
		handoffFingerprint(hs[base+2]),
	}, handoffAnchorSep)
	if rec.Anchor != wantAnchor {
		t.Fatalf("锚点应为倒数第 %d 条起的 %d 连序列锚，实际 %q", handoffKeepRecentMsgs, handoffAnchorSeq, rec.Anchor)
	}

	// ② 复用：同历史再调 → 不再调 LLM
	view2, ok2 := BuildHandoffView(context.Background(), prov, nil, store, conv, hist, "继续", 0)
	if !ok2 {
		t.Fatal("第二次也应返回交接视图")
	}
	if calls != 1 {
		t.Fatalf("零增量应复用，LLM 调用不应增加（实际 %d）", calls)
	}
	if len(view2) != len(view) {
		t.Fatalf("复用视图应与首次一致（%d），实际 %d", len(view), len(view2))
	}

	// ③ 刷新：追加大量新消息 → 重新生成
	long := append(append([]Message{}, hist...), handoffHist(40, "more")...)
	view3, ok3 := BuildHandoffView(context.Background(), prov, nil, store, conv, long, "继续", 0)
	if !ok3 {
		t.Fatal("第三次应返回交接视图")
	}
	if calls != 2 {
		t.Fatalf("大增量应刷新（calls=2），实际 %d", calls)
	}
	rec3, _ := store.LoadHandoff(conv)
	if rec3 == nil || rec3.MsgCount != len(long) || rec3.Anchor == rec.Anchor {
		t.Fatalf("刷新后记录应更新（count=%d anchor 应变化）: %+v", len(long), rec3)
	}
	if len(view3) != 1+wantKeep {
		t.Fatalf("刷新视图应保留最近 %d 条，实际 %d", wantKeep, len(view3))
	}

	// ④ 关闭开关 → 不启用
	t.Setenv("PAIR_HANDOFF", "0")
	if _, ok4 := BuildHandoffView(context.Background(), prov, nil, store, conv, long, "继续", 0); ok4 {
		t.Fatal("PAIR_HANDOFF=0 时不应启用交接")
	}
}
