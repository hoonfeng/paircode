package agent

// handoff_anchor_repeat_test.go — 会话交接锚点「重复轮次」回归（2026-09-12）。
//
// 实测根因（缓存命中率跨轮骤降）：
//
//	真实长会话里同一轮内会出现大量**模板化重复消息**（同一工具同参数反复调用、
//	模型每步都输出同一句套话、工具结果同名同量），于是「基点起连续 3 条指纹」
//	这一序列锚会在历史中**多处命中**（实测 4 处：107/229/351/473）。
//	原 handoffAnchorIndex 从末尾向前找第一个匹配 → 定位到最近一次出现处
//	（实测距末尾 15 条，真基点在第 107 条，漂移 366 条）→ 两个恶果：
//	  ① ComposeHandoffView 保留段退化为「最近十几条」，每次跨轮重排 →
//	     provider 前缀缓存从交接消息之后整段失效（实测命中率 98%→56%，
//	     每轮白付 ~24K tokens）；
//	  ② ShouldRefreshHandoff 的 increment ≈ KeptTokens → delta ≈ 0 →
//	     永不刷新（交接文本老化，新内容进不去）。
//
// 既有 TestHandoff_SequenceAnchorDedup / _ViewPrefixStable 只放了**单条**重复，
// 序列锚（3 条）足以区分，故未能捕获；本文件按「整轮同模式重复」构造。
//
// 修复后的定位语义：① 用记录内的 MsgCount 推回的**生成时基点**（精确、跨轮
// 恒定，穿透重复干扰、保住折叠瘦身）；② 未携带 MsgCount 的存量记录退化为
// 「从前往后第一个匹配」（历史前部不变 → 仍跨轮稳定）。

import (
	"strings"
	"testing"
)

// repeatRound 构造「整轮同模式重复」消息：assistant 套话 + 工具结果，逐条可互换。
func repeatRound(rounds, steps int) []Message {
	hist := []Message{{Role: RoleUser, Content: "第1轮任务"}}
	for r := 0; r < rounds; r++ {
		for s := 0; s < steps; s++ {
			hist = append(hist, Message{
				Role:    RoleAssistant,
				Content: "先查一下项目文件。",
				ToolCalls: []ToolCall{{
					ID: "call_" + strings.Repeat("a", 1), Type: "function",
					Function: FunctionCall{Name: "glob", Arguments: `{"pattern":"*.md"}`},
				}},
			})
			hist = append(hist, Message{Role: RoleTool, ToolCallID: "call_" + strings.Repeat("a", 1),
				Name: "glob", Content: "（找到 31 个文件）" + strings.Repeat("f", 30)})
		}
		hist = append(hist, Message{Role: RoleAssistant, Content: "本轮小结"})
		if r < rounds-1 {
			hist = append(hist, Message{Role: RoleUser, Content: "下一轮任务"})
		}
	}
	return hist
}

// expectedBaseIndex 生成时基点位置（handoffAnchorAt 的取值规则）。
func expectedBaseIndex(hist []Message, keep int) int {
	idx := len(hist) - keep
	if idx < 0 {
		idx = 0
	}
	for idx > 0 && hist[idx].Role == RoleTool {
		idx--
	}
	return idx
}

// countAnchorMatches 统计锚序列在历史中的命中处数（构造有效性自检）。
func countAnchorMatches(hist []Message, anchor string) int {
	parts := strings.Split(anchor, handoffAnchorSep)
	n := len(parts)
	cnt := 0
	for i := 0; i+n <= len(hist); i++ {
		ok := true
		for k := 0; k < n; k++ {
			if handoffFingerprint(hist[i+k]) != parts[k] {
				ok = false
				break
			}
		}
		if ok {
			cnt++
		}
	}
	return cnt
}

// assertViewPrefix 断言 prev 是 cur 的前缀（缓存可连续命中的充要条件）。
func assertViewPrefix(t *testing.T, name string, prev, cur []Message) {
	t.Helper()
	if len(cur) < len(prev) {
		t.Fatalf("%s: 视图长度回退（%d < %d）", name, len(cur), len(prev))
	}
	for i := range prev {
		if prev[i].Role != cur[i].Role || prev[i].Content != cur[i].Content || prev[i].ToolCallID != cur[i].ToolCallID {
			t.Fatalf("%s: 跨轮视图前缀第 %d 条被改写（provider 缓存从此点起全部 miss）\nprev=%v|%.60s\ncur =%v|%.60s",
				name, i, prev[i].Role, prev[i].Content, cur[i].Role, cur[i].Content)
		}
	}
}

// TestHandoff_AnchorStableUnderRepeatedRounds 重复轮次下锚点必须精确且跨轮恒定，
// 视图前缀必须单调延展（缓存命中的前提）。
func TestHandoff_AnchorStableUnderRepeatedRounds(t *testing.T) {
	hist := repeatRound(3, 20) // 3 轮 × 20 步同模式 → 锚序列必然多处命中
	anchor := handoffAnchor(hist)

	if got := countAnchorMatches(hist, anchor); got < 2 {
		t.Fatalf("构造无效：锚序列仅 %d 处命中（需 ≥2 处才能暴露漂移）", got)
	}

	// ① 新记录（带 MsgCount）→ 精确定位到生成时基点（穿透重复干扰）
	want := expectedBaseIndex(hist, handoffKeepRecentMsgs)
	recNew := &HandoffRecord{Text: handoffTitle + "\n重复轮次", Anchor: anchor, MsgCount: len(hist)}
	if idx := handoffAnchorIndex(hist, recNew); idx != want {
		t.Fatalf("带 MsgCount 的记录应精确定位到生成时基点 %d，实得 %d", want, idx)
	}
	kept := len(hist) - want
	if kept > handoffKeepRecentMsgs+2 {
		t.Fatalf("保留段 %d 条，超出保留深度 %d（折叠瘦身失效）", kept, handoffKeepRecentMsgs)
	}

	// ② 存量记录（无 MsgCount，如旧 .handoff.json）→ 首个匹配，仍跨轮稳定
	recOld := &HandoffRecord{Text: handoffTitle + "\n旧记录", Anchor: anchor}
	idxOld := handoffAnchorIndex(hist, recOld)
	if idxOld < 0 {
		t.Fatalf("旧记录锚点未定位")
	}
	if idxOld > want {
		t.Fatalf("旧记录定位 %d 晚于生成时基点 %d（会漂移到重复项 → 保留段重排）", idxOld, want)
	}

	// ③ 跨轮稳定性：追加消息后两者定位都不变
	hist2 := append(append([]Message{}, hist...), Message{Role: RoleAssistant, Content: "好的"},
		Message{Role: RoleUser, Content: "继续"})
	if idx := handoffAnchorIndex(hist2, recNew); idx != want {
		t.Fatalf("追加后（带 MsgCount）锚点漂移：%d → %d（前缀缓存必断）", want, idx)
	}
	if idx := handoffAnchorIndex(hist2, recOld); idx != idxOld {
		t.Fatalf("追加后（旧记录）锚点漂移：%d → %d（前缀缓存必断）", idxOld, idx)
	}

	// ④ 视图前缀单调延展
	v1 := ComposeHandoffView(hist, recNew, recNew.Text)
	v1old := ComposeHandoffView(hist, recOld, recOld.Text)
	v2 := ComposeHandoffView(hist2, recNew, recNew.Text)
	v2old := ComposeHandoffView(hist2, recOld, recOld.Text)
	assertViewPrefix(t, "v2", v1, v2)
	assertViewPrefix(t, "v2old", v1old, v2old)
	t.Logf("重复轮次：命中 %d 处；精确基点 %d（保留 %d 条）；旧记录定位 %d；视图前缀稳定 %d→%d 条",
		countAnchorMatches(hist, anchor), want, kept, idxOld, len(v1), len(v2))
}

// TestHandoff_RefreshIncrementAfterAnchorFix 锚点修正后 increment 必须反映
// 「基点之后的新增内容」——旧实现漂移到末尾时 increment≈0（实测日志「增量 ~0 tokens」），
// 导致交接永不刷新（新内容进不去交接要点）。
func TestHandoff_RefreshIncrementAfterAnchorFix(t *testing.T) {
	hist := repeatRound(3, 20)
	base := expectedBaseIndex(hist, handoffKeepRecentMsgs)
	rec := &HandoffRecord{Text: handoffTitle + "\n重复轮次", Anchor: handoffAnchor(hist),
		MsgCount: len(hist), KeptTokens: estimateTokens(handoffIncrement(hist,
			&HandoffRecord{Text: "x", Anchor: handoffAnchor(hist), MsgCount: len(hist)}))}

	// 追加一整轮同模式消息（模拟长任务持续推进）
	grown := append(append([]Message{}, hist...), repeatRound(1, 20)...)
	need, inc := ShouldRefreshHandoff(grown, rec)
	if inc <= 0 {
		t.Fatalf("增量估算 %d ≤ 0（锚点漂移导致永不刷新）", inc)
	}
	newTokens := estimateTokens(grown[base:])
	if inc < newTokens/3 {
		t.Fatalf("增量 %d 明显小于基点后内容（%d）——锚点未定位到真实基点", inc, newTokens)
	}
	t.Logf("增量估算：基点后内容 %d tokens，追加一轮后增量 %d tokens（刷新=%v）", newTokens, inc, need)
}
