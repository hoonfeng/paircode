package agent

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

// ★ GLM 兼容兜底测试（2026-08-27）：GLM（智谱）硬校验 messages 必须至少存在一条
//   user 消息（否则 1214）；compact 中途压缩可能把 user 丢进摘要且快照未落盘，
//   buildCallContext 最终兜底必须保证输出含 user。实测依据见 _temp/test_1214.py。

// TestBuildCallContextGLMUserFallback 无 user（system+assistant+tool）→ 追加占位 user 于末尾。
// ★ 2026-09-11 行为变更（缓存前缀断裂根因 A）：占位符从「system 之后（msg#1）」改为
//
//	「末尾追加」——插在 msg#1 会把整段历史挤后一位，「有无 user」翻转时前后请求前缀
//	断在 msg#1，provider 缓存整段 miss（实测同 turn 内 32→34→37 连续两次断裂）。
func TestBuildCallContextGLMUserFallback(t *testing.T) {
	l := &Loop{}
	msgs := []Message{
		{Role: RoleSystem, Content: "系统提示"},
		{Role: RoleAssistant, Content: "", ToolCalls: []ToolCall{{
			ID: "c1", Type: "function", Function: FunctionCall{Name: "calc", Arguments: "{}"},
		}}},
		{Role: RoleTool, ToolCallID: "c1", Content: "2"},
	}
	out := l.buildCallContext(msgs)
	nUser := 0
	userIdx := -1
	for i, m := range out {
		if m.Role == RoleUser {
			nUser++
			userIdx = i
		}
	}
	if nUser != 1 {
		t.Fatalf("应恰好插入 1 条 user，得 %d", nUser)
	}
	if userIdx != len(out)-1 {
		t.Fatalf("占位 user 应追加在末尾（index=%d），得 %d", len(out)-1, userIdx)
	}
	// 历史段位置逐字节不变（前缀稳定性的前提）
	for i := range msgs {
		if out[i].Role != msgs[i].Role || out[i].Content != msgs[i].Content {
			t.Fatalf("历史段位置被破坏：idx=%d got=%s want=%s", i, out[i].Role, msgs[i].Role)
		}
	}
}

// TestBuildCallContextGLMNoSystemFallback 连 system 都没有（assistant+tool）→ 占位 user 追加末尾。
func TestBuildCallContextGLMNoSystemFallback(t *testing.T) {
	l := &Loop{}
	msgs := []Message{
		{Role: RoleAssistant, Content: "回复"},
		{Role: RoleTool, ToolCallID: "c1", Content: "2"},
	}
	out := l.buildCallContext(msgs)
	if len(out) != 3 {
		t.Fatalf("应 3 条（追加 1 条），得 %d", len(out))
	}
	if out[2].Role != RoleUser {
		t.Fatalf("占位 user 应追加末尾，得 %s", out[2].Role)
	}
	if out[0].Role != RoleAssistant || out[1].Role != RoleTool {
		t.Fatalf("历史段位置被破坏：%s,%s", out[0].Role, out[1].Role)
	}
}

// TestBuildCallContextGLMUserPresentNoInsert 已有 user → 不插入（长度与内容不变）。
func TestBuildCallContextGLMUserPresentNoInsert(t *testing.T) {
	l := &Loop{}
	msgs := []Message{
		{Role: RoleSystem, Content: "系统提示"},
		{Role: RoleUser, Content: "当前任务"},
		{Role: RoleAssistant, Content: "ok"},
	}
	out := l.buildCallContext(msgs)
	if len(out) != 3 {
		t.Fatalf("已有 user 不应插入，长度应 3，得 %d", len(out))
	}
	if out[1].Role != RoleUser || out[1].Content != "当前任务" {
		t.Fatalf("原 user 应保留，得 %+v", out[1])
	}
}

// TestBuildCallContextGLMFallbackNotMutateInput 兜底不得修改原始 msgs（调用副本语义）。
func TestBuildCallContextGLMFallbackNotMutateInput(t *testing.T) {
	l := &Loop{}
	msgs := []Message{
		{Role: RoleSystem, Content: "系统提示"},
		{Role: RoleAssistant, Content: "回复"},
	}
	_ = l.buildCallContext(msgs)
	if len(msgs) != 2 {
		t.Fatalf("原始 msgs 不应被修改，长度应仍为 2，得 %d", len(msgs))
	}
}

// TestBuildCallContextPlaceholderFlipPrefixStable 前缀稳定性回归（缓存断裂根因 A）：
// 「msgs 有无 user」在相邻两次请求间翻转时（临时用户反馈注入 / 压缩吞掉 user / 新 Run），
// 历史段必须逐字节对齐——请求 A（无 user → 末尾占位）与请求 B（含反馈 → 无占位）的
// 公共前缀必须覆盖全部历史消息。修复前：A 的 msg#1 是占位符、B 的 msg#1 是历史消息，
// provider 缓存从 msg#1 起整段 miss（实测同 turn 内 32→34→37 连续两次断裂）。
func TestBuildCallContextPlaceholderFlipPrefixStable(t *testing.T) {
	base := []Message{
		{Role: RoleSystem, Content: "S"},
		{Role: RoleAssistant, Content: "a1"},
		{Role: RoleTool, ToolCallID: "t1", Content: "r1"},
		{Role: RoleAssistant, Content: "a2"},
		{Role: RoleTool, ToolCallID: "t2", Content: "r2"},
	}
	lA := &Loop{}
	outA := lA.buildCallContext(append([]Message{}, base...))
	lB := &Loop{}
	outB := lB.buildCallContext(append(append([]Message{}, base...), Message{Role: RoleUser, Content: "【用户反馈】x"}))
	if len(outA) != 6 {
		t.Fatalf("A 应 6 条（5 历史 + 占位），得 %d", len(outA))
	}
	if len(outB) != 6 {
		t.Fatalf("B 应 6 条（5 历史 + 反馈），得 %d", len(outB))
	}
	for i := 0; i < 5; i++ {
		if outA[i].Role != outB[i].Role || outA[i].Content != outB[i].Content || outA[i].ToolCallID != outB[i].ToolCallID {
			t.Fatalf("前缀在 msg#%d 断裂：A=%+v B=%+v", i, outA[i], outB[i])
		}
	}
	// 仅在末尾一条不同（占位符 vs 反馈）——正是可复用前缀应有的形态
	if outA[5].Role != RoleUser || outB[5].Role != RoleUser {
		t.Fatalf("末条均应为 user：A=%s B=%s", outA[5].Role, outB[5].Role)
	}
}

// TestCompactArchiveFullHistoryRestore 循环中途精简后，落盘线必须能还原完整时间线：
// compact 归档被删中段 → fullHistory(prefix+归档+视图尾部) == 原始 msgs。
// 这是「前端刷新后历史只剩精简摘要」问题的回归测试。
// ★ 2026-09 保前缀精简：视图里另有占位块/摘要块，还原时必须被剔除。
func TestCompactArchiveFullHistoryRestore(t *testing.T) {
	l := &Loop{}
	// 1 system + 45 组 user/assistant（91 条）→ keepFrom = 91-40 = 51 ≥ prefix=1，
	// dropped = msgs[1:51]（50 条，含 user）→ 精简后视图不再含这些真实消息。
	orig := []Message{{Role: RoleSystem, Content: "sys"}}
	for i := 0; i < 45; i++ {
		orig = append(orig,
			Message{Role: RoleUser, Content: fmt.Sprintf("任务%d", i)},
			Message{Role: RoleAssistant, Content: fmt.Sprintf("回复%d", i)})
	}
	out, _, dropped := l.compact(context.Background(), orig)
	if dropped == 0 {
		t.Fatal("应触发精简（dropped>0）")
	}
	// 归档校验：archive == 被删段且保序
	if len(l.compactArchive) != dropped {
		t.Fatalf("归档条数 %d 应等于 dropped %d", len(l.compactArchive), dropped)
	}
	for i := range l.compactArchive {
		if !reflect.DeepEqual(l.compactArchive[i], orig[1+i]) {
			t.Fatalf("归档[%d] 与原始被删段不一致", i)
		}
	}
	// 还原校验：fullHistory(精简视图) == 原始完整时间线（视图块被剔除）
	full := l.fullHistory(out)
	if len(full) != len(orig) {
		t.Fatalf("还原长度 %d 应等于原始 %d", len(full), len(orig))
	}
	for i := range full {
		if !reflect.DeepEqual(full[i], orig[i]) {
			t.Fatalf("还原[%d] 与原始不一致：%+v vs %+v", i, full[i], orig[i])
		}
	}
	// 兜底协同：还原版含真实 user（最初任务），lastUser 锚点可正常定位
	hasUser := false
	for _, m := range full {
		if m.Role == RoleUser {
			hasUser = true
			break
		}
	}
	if !hasUser {
		t.Fatal("还原版必须含 user 消息（锚点重组依赖）")
	}
}
