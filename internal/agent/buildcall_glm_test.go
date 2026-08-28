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

// TestBuildCallContextGLMUserFallback 无 user（system+assistant+tool）→ 插入占位 user 于 system 之后。
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
	if userIdx != 1 {
		t.Fatalf("占位 user 应插在 system 前缀之后（index=1），得 %d", userIdx)
	}
	// 后续 assistant/tool 顺序不变
	if out[2].Role != RoleAssistant || out[3].Role != RoleTool {
		t.Fatalf("占位后应为 assistant+tool，得 %s+%s", out[2].Role, out[3].Role)
	}
}

// TestBuildCallContextGLMNoSystemFallback 连 system 都没有（assistant+tool）→ 占位 user 插在最前。
func TestBuildCallContextGLMNoSystemFallback(t *testing.T) {
	l := &Loop{}
	msgs := []Message{
		{Role: RoleAssistant, Content: "回复"},
		{Role: RoleTool, ToolCallID: "c1", Content: "2"},
	}
	out := l.buildCallContext(msgs)
	if out[0].Role != RoleUser {
		t.Fatalf("无 system 时占位 user 应在最前，得 %s", out[0].Role)
	}
	if len(out) != 3 {
		t.Fatalf("应 3 条（插入 1 条），得 %d", len(out))
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

// TestCompactArchiveFullHistoryRestore 循环中途压缩后，落盘线必须能还原完整时间线：
// compact 归档被删中段 → fullHistory(prefix+归档+视图尾部) == 原始 msgs。
// 这是「前端刷新后历史只剩压缩摘要」问题的回归测试。
func TestCompactArchiveFullHistoryRestore(t *testing.T) {
	l := &Loop{}
	// 1 system + 15 组 user/assistant（31 条）→ keepFrom = 31-16 = 15 ≥ prefix=1，
	// dropped = msgs[1:15]（14 条，含 user）→ 压缩后视图无 user（GLM 1214 场景）
	orig := []Message{{Role: RoleSystem, Content: "sys"}}
	for i := 0; i < 15; i++ {
		orig = append(orig,
			Message{Role: RoleUser, Content: fmt.Sprintf("任务%d", i)},
			Message{Role: RoleAssistant, Content: fmt.Sprintf("回复%d", i)})
	}
	out, _, dropped := l.compact(context.Background(), orig)
	if dropped == 0 {
		t.Fatal("应触发压缩（dropped>0）")
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
	// 还原校验：fullHistory(压缩视图) == 原始完整时间线
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
