package agent

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
)

// ─── 审核共享状态 e2e（错误计数移除后）──

// 注册一个需审批的测试工具（approve_me）。
func regApproveTool(r *Registry, name string) *atomic.Int32 {
	var calls atomic.Int32
	r.Register(&Tool{
		Name:             name,
		Description:      "需审批的测试工具",
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			calls.Add(1)
			return "OK:" + name, nil
		},
	})
	return &calls
}

// TestJSLoopApproveRejectState 审核驳回 → 共享状态记录 → 同一工具「同一参数」重试
// 免打扰自动驳回（参数指纹一致才短路；参数一变即重新送审，见下方 DifferentArgs 用例）。
func TestJSLoopApproveRejectState(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()
	loadRealAgentloop(t)

	reg := NewRegistry()
	toolCalls := regApproveTool(reg, "approve_me")

	// 用户审批回调：永远驳回（第 2 次申请应走免打扰自动驳回，不再触发此回调）
	var approveCalls atomic.Int32
	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{
			{ID: "c1", Type: "function", Function: FunctionCall{Name: "approve_me", Arguments: `{"v":1}`}},
		}},
		// 被驳回后 LLM 原样重试同一工具（参数完全相同 → 指纹一致 → 免打扰自动驳回）
		{ToolCalls: []ToolCall{
			{ID: "c2", Type: "function", Function: FunctionCall{Name: "approve_me", Arguments: `{"v":1}`}},
		}},
		{Content: "好的，不执行该写入"},
	}}
	var events []Event
	loop := &Loop{Provider: mock, Registry: reg, System: "approve-e2e",
		Approve: func(ctx context.Context, tc ToolCall) (bool, string) {
			approveCalls.Add(1)
			return false, "用户：不要执行写入"
		},
		OnEvent: func(e Event) { events = append(events, e) },
	}

	msgs, err := loop.Run(context.Background(), "调用 approve_me", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if mock.Calls() != 3 {
		t.Errorf("LLM 应调用 3 次，得 %d", mock.Calls())
	}
	// ① 用户审批回调只触发 1 次（第 2 次申请走免打扰自动驳回）
	if approveCalls.Load() != 1 {
		t.Errorf("审批回调应触发 1 次（免打扰自动驳回），得 %d", approveCalls.Load())
	}
	// ② 工具不应真正执行（两次都被驳回）
	if toolCalls.Load() != 0 {
		t.Errorf("approve_me 不应被执行，得 %d 次", toolCalls.Load())
	}
	// ③ 两条 tool 消息：第 1 条用户反馈，第 2 条免打扰自动驳回（含前次理由提示）
	var toolMsgs []Message
	for _, m := range msgs {
		if m.Role == RoleTool {
			toolMsgs = append(toolMsgs, m)
		}
	}
	if len(toolMsgs) != 2 {
		t.Fatalf("应有 2 条 tool 消息，得 %d", len(toolMsgs))
	}
	if !strings.Contains(toolMsgs[0].Content, "不要执行写入") {
		t.Errorf("第 1 条驳回反馈 = %q", toolMsgs[0].Content)
	}
	if !strings.Contains(toolMsgs[1].Content, "前一次驳回理由仍有效") {
		t.Errorf("第 2 条应为免打扰自动驳回（含前次理由提示），得 %q", toolMsgs[1].Content)
	}
	// ④ 共享审核状态：最近驳回 = approve_me，历史 1 条（自动驳回不重复记录）
	st := loop.getApproveState().Snapshot()
	if st["lastRejectedTool"] != "approve_me" {
		t.Errorf("lastRejectedTool = %v", st["lastRejectedTool"])
	}
	hist, _ := st["rejectedHistory"].([]map[string]any)
	if len(hist) != 1 {
		t.Errorf("历史应为 1 条（自动驳回不记录），得 %d", len(hist))
	}
	// ⑤ 无 blocked 停止（错误计数移除后不再自动停 turn）
	for _, e := range events {
		if e.Type == EventError && strings.Contains(e.Content, "自动停止") {
			t.Errorf("不应出现连续驳回自动停止事件：%q", e.Content)
		}
	}
}

// TestJSLoopApproveThenPass 驳回后换策略直通（白名单）→ 工具执行 + 状态清除。
func TestJSLoopApproveThenPass(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()
	loadRealAgentloop(t)

	reg := NewRegistry()
	toolCalls := regApproveTool(reg, "approve_me")

	var approveCalls atomic.Int32
	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{
			{ID: "c1", Type: "function", Function: FunctionCall{Name: "approve_me", Arguments: `{"v":1}`}},
		}},
		{Content: "完成"},
	}}
	// 白名单直通：approve_me 免审
	loop := &Loop{Provider: mock, Registry: reg, System: "approve-pass-e2e",
		ReviewWhitelist: []string{"approve_me"},
		Approve: func(ctx context.Context, tc ToolCall) (bool, string) {
			approveCalls.Add(1)
			return false, "不应被调用"
		},
	}
	// 先手动预置最近驳回标记（模拟上次会话驳回），直通应清掉
	loop.getApproveState().recordReject("approve_me", "历史驳回")

	msgs, err := loop.Run(context.Background(), "调用 approve_me", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if approveCalls.Load() != 0 {
		t.Errorf("白名单直通不应触发审批回调，得 %d", approveCalls.Load())
	}
	if toolCalls.Load() != 1 {
		t.Errorf("approve_me 应执行 1 次，得 %d", toolCalls.Load())
	}
	if mock.Calls() != 2 {
		t.Errorf("LLM 应调用 2 次，得 %d", mock.Calls())
	}
	// 直通后最近驳回标记应被清空
	if loop.getApproveState().Snapshot()["lastRejectedTool"] != "" {
		t.Error("直通后该工具的最近驳回标记应清空")
	}
	_ = msgs
}

// TestJSLoopApproveRejectDifferentArgs 驳回指纹修复：同工具但「参数不同」的调用
// 必须重新送审，不得被前一工具的驳回记录连带免审驳回（否则 5 分钟内该工具的所有
// 合法调用都被误伤——原实现只比工具名，实测占全部驳回的 66%）。
func TestJSLoopApproveRejectDifferentArgs(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()
	loadRealAgentloop(t)

	reg := NewRegistry()
	toolCalls := regApproveTool(reg, "approve_me")

	var approveCalls atomic.Int32
	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{
			{ID: "c1", Type: "function", Function: FunctionCall{Name: "approve_me", Arguments: `{"v":1}`}},
		}},
		// 被驳回后换参数重试（不同文件/不同写入内容）→ 应按新请求重新送审
		{ToolCalls: []ToolCall{
			{ID: "c2", Type: "function", Function: FunctionCall{Name: "approve_me", Arguments: `{"v":2}`}},
		}},
		{Content: "好的，不执行该写入"},
	}}
	loop := &Loop{Provider: mock, Registry: reg, System: "approve-diffargs-e2e",
		Approve: func(ctx context.Context, tc ToolCall) (bool, string) {
			approveCalls.Add(1)
			return false, "用户：不要执行写入"
		},
	}

	msgs, err := loop.Run(context.Background(), "调用 approve_me", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if mock.Calls() != 3 {
		t.Errorf("LLM 应调用 3 次，得 %d", mock.Calls())
	}
	// ① 两次都走真实审批回调（参数不同 → 不得免审自动驳回）
	if approveCalls.Load() != 2 {
		t.Errorf("参数不同的重试应重新送审（审批回调 2 次），得 %d", approveCalls.Load())
	}
	// ② 工具仍不执行（回调一律驳回）
	if toolCalls.Load() != 0 {
		t.Errorf("approve_me 不应被执行，得 %d 次", toolCalls.Load())
	}
	// ③ 两条 tool 消息都是用户驳回反馈（第 2 条不得是免打扰自动驳回）
	var toolMsgs []Message
	for _, m := range msgs {
		if m.Role == RoleTool {
			toolMsgs = append(toolMsgs, m)
		}
	}
	if len(toolMsgs) != 2 {
		t.Fatalf("应有 2 条 tool 消息，得 %d", len(toolMsgs))
	}
	for i, m := range toolMsgs {
		if !strings.Contains(m.Content, "不要执行写入") {
			t.Errorf("第 %d 条应为用户驳回反馈，得 %q", i+1, m.Content)
		}
		if strings.Contains(m.Content, "前一次驳回理由仍有效") {
			t.Errorf("第 %d 条不应是免打扰自动驳回（参数已变），得 %q", i+1, m.Content)
		}
	}
	// ④ 共享审核状态：两次驳回都入历史（自动驳回不记录）
	st := loop.getApproveState().Snapshot()
	if st["lastRejectedTool"] != "approve_me" {
		t.Errorf("lastRejectedTool = %v", st["lastRejectedTool"])
	}
	hist, _ := st["rejectedHistory"].([]map[string]any)
	if len(hist) != 2 {
		t.Errorf("历史应为 2 条（两次真实送审均驳回），得 %d", len(hist))
	}
}
